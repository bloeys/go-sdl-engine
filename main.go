package main

import (
	"fmt"
	"runtime"
	"strconv"

	imgui "github.com/AllenDang/cimgui-go"
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/assets"
	"github.com/bloeys/nmage/buffers"
	"github.com/bloeys/nmage/camera"
	"github.com/bloeys/nmage/engine"
	"github.com/bloeys/nmage/input"
	"github.com/bloeys/nmage/logging"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/meshes"
	"github.com/bloeys/nmage/renderer/rend3dgl"
	"github.com/bloeys/nmage/timing"
	nmageimgui "github.com/bloeys/nmage/ui/imgui"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

/*
@TODO:
	- Rendering:
		- Blinn-Phong lighting model ✅
		- Directional lights ✅
		- Point lights ✅
		- Spotlights ✅
		- Directional light shadows ✅
		- Point light shadows ✅
		- Spotlight shadows ✅
		- UBO support
		- HDR
		- Cascaded shadow mapping
		- Skeletal animations
	- Proper model loading (i.e. load model by reading all its meshes, textures, and so on together)
	- Create VAO struct independent from VBO to support multi-VBO use cases (e.g. instancing) ✅
	- Renderer batching
	- Scene graph
	- Separate engine loop from rendering loop? or leave it to the user?
	- Abstract keys enum away from sdl
	- Proper Asset loading
	- Frustum culling
	- Material system editor with fields automatically extracted from the shader
*/

type DirLight struct {
	Dir           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3
}

var (
	dirLightSize float32 = 30
	dirLightNear float32 = 0.1
	dirLightFar  float32 = 30
	dirLightPos          = gglm.NewVec3(0, 10, 0)
)

func (d *DirLight) GetProjViewMat() gglm.Mat4 {

	// Some arbitrary position for the directional light
	pos := dirLightPos

	size := dirLightSize
	nearClip := dirLightNear
	farClip := dirLightFar

	projMat := gglm.Ortho(-size, size, -size, size, nearClip, farClip).Mat4
	viewMat := gglm.LookAtRH(pos, pos.Clone().Add(&d.Dir), gglm.NewVec3(0, 1, 0)).Mat4

	return *projMat.Mul(&viewMat)
}

// Check https://wiki.ogre3d.org/tiki-index.php?page=-Point+Light+Attenuation for values
type PointLight struct {
	Pos           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3

	// @TODO
	Radius float32

	Constant  float32
	Linear    float32
	Quadratic float32

	FarPlane float32
}

const (
	MaxPointLights = 8

	// If this changes update the array depth map shader
	MaxSpotLights = 4
)

var (
	pointLightNear float32 = 1
)

func (p *PointLight) GetProjViewMats(shadowMapWidth, shadowMapHeight float32) [6]gglm.Mat4 {

	aspect := float32(shadowMapWidth) / float32(shadowMapHeight)
	projMat := gglm.Perspective(90*gglm.Deg2Rad, aspect, pointLightNear, p.FarPlane)

	projViewMats := [6]gglm.Mat4{
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(1, 0, 0).Add(&p.Pos), gglm.NewVec3(0, -1, 0)).Mat4),
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(-1, 0, 0).Add(&p.Pos), gglm.NewVec3(0, -1, 0)).Mat4),
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(0, 1, 0).Add(&p.Pos), gglm.NewVec3(0, 0, 1)).Mat4),
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(0, -1, 0).Add(&p.Pos), gglm.NewVec3(0, 0, -1)).Mat4),
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(0, 0, 1).Add(&p.Pos), gglm.NewVec3(0, -1, 0)).Mat4),
		*projMat.Clone().Mul(&gglm.LookAtRH(&p.Pos, gglm.NewVec3(0, 0, -1).Add(&p.Pos), gglm.NewVec3(0, -1, 0)).Mat4),
	}

	return projViewMats
}

type SpotLight struct {
	Pos            gglm.Vec3
	Dir            gglm.Vec3
	DiffuseColor   gglm.Vec3
	SpecularColor  gglm.Vec3
	InnerCutoffRad float32
	OuterCutoffRad float32

	// Near plane like 0.x (or anything too small) causes shadows to not work properly.
	// Needs adjusting as the distance of light to object increases
	NearPlane float32

	FarPlane float32
}

func (s *SpotLight) GetProjViewMat() gglm.Mat4 {

	projMat := gglm.Perspective(s.OuterCutoffRad*2, 1, s.NearPlane, s.FarPlane)

	// Adjust up vector if lightDir is parallel or nearly parallel to upVector
	// as lookat view matrix breaks if up and look at are parallel
	up := gglm.NewVec3(0, 1, 0)
	if gglm.Abs32(gglm.DotVec3(&s.Dir, up)) > 0.99 {
		up.SetXY(1, 0)
	}

	viewMat := gglm.LookAtRH(&s.Pos, s.Pos.Clone().Add(&s.Dir), up).Mat4

	return *projMat.Mul(&viewMat)
}

func (s *SpotLight) InnerCutoffCos() float32 {
	return gglm.Cos32(s.InnerCutoffRad)
}

func (s *SpotLight) OuterCutoffCos() float32 {
	return gglm.Cos32(s.OuterCutoffRad)
}

const (
	camSpeed         = 15
	mouseSensitivity = 0.5

	unscaledWindowWidth  = 1280
	unscaledWindowHeight = 720
)

var (
	window *engine.Window

	pitch float32 = 0
	yaw   float32 = -1.5
	cam   *camera.Camera

	// Demo fbo
	renderToDemoFbo    = false
	renderToBackBuffer = true
	demoFboScale       = gglm.NewVec2(0.25, 0.25)
	demoFboOffset      = gglm.NewVec2(0.75, -0.75)
	demoFbo            buffers.Framebuffer

	// Dir light fbo
	showDirLightDepthMapFbo   = false
	dirLightDepthMapFboScale  = gglm.NewVec2(0.25, 0.25)
	dirLightDepthMapFboOffset = gglm.NewVec2(0.75, -0.2)
	dirLightDepthMapFbo       buffers.Framebuffer

	// Point light fbo
	pointLightDepthMapFbo buffers.Framebuffer

	// Spot light fbo
	spotLightDepthMapFbo buffers.Framebuffer

	screenQuadVao buffers.VertexArray
	screenQuadMat *materials.Material

	unlitMat           *materials.Material
	whiteMat           *materials.Material
	containerMat       *materials.Material
	palleteMat         *materials.Material
	skyboxMat          *materials.Material
	depthMapMat        *materials.Material
	arrayDepthMapMat   *materials.Material
	omnidirDepthMapMat *materials.Material
	debugDepthMat      *materials.Material

	cubeMesh   *meshes.Mesh
	sphereMesh *meshes.Mesh
	chairMesh  *meshes.Mesh
	skyboxMesh *meshes.Mesh

	cubeModelMat = gglm.NewTrMatId()

	renderSkybox      = true
	renderDepthBuffer bool

	skyboxCmap assets.Cubemap

	dpiScaling float32

	// Light settings
	ambientColor = gglm.NewVec3(0, 0, 0)

	// Lights
	dirLight = DirLight{
		Dir:           *gglm.NewVec3(0, -0.5, -0.8).Normalize(),
		DiffuseColor:  *gglm.NewVec3(1, 1, 1),
		SpecularColor: *gglm.NewVec3(1, 1, 1),
	}
	pointLights = [...]PointLight{
		{
			Pos:           *gglm.NewVec3(0, 2, -2),
			DiffuseColor:  *gglm.NewVec3(1, 0, 0),
			SpecularColor: *gglm.NewVec3(1, 1, 1),
			// These values are for 50m range
			Constant:  1.0,
			Linear:    0.09,
			Quadratic: 0.032,

			FarPlane: 25,
		},
		{
			Pos:           *gglm.NewVec3(0, -5, 0),
			DiffuseColor:  *gglm.NewVec3(0, 1, 0),
			SpecularColor: *gglm.NewVec3(1, 1, 1),
			Constant:      1.0,
			Linear:        0.09,
			Quadratic:     0.032,
			FarPlane:      25,
		},
		{
			Pos:           *gglm.NewVec3(5, 0, 0),
			DiffuseColor:  *gglm.NewVec3(1, 1, 1),
			SpecularColor: *gglm.NewVec3(1, 1, 1),
			Constant:      1.0,
			Linear:        0.09,
			Quadratic:     0.032,
			FarPlane:      25,
		},
		{
			Pos:           *gglm.NewVec3(-3, 4, 3),
			DiffuseColor:  *gglm.NewVec3(1, 1, 1),
			SpecularColor: *gglm.NewVec3(1, 1, 1),
			Constant:      1.0,
			Linear:        0.09,
			Quadratic:     0.032,
			FarPlane:      25,
		},
	}
	spotLights = [...]SpotLight{
		{
			Pos:           *gglm.NewVec3(-4, 7, 5),
			Dir:           *gglm.NewVec3(1.5, -0.9, 0).Normalize(),
			DiffuseColor:  *gglm.NewVec3(1, 0, 1),
			SpecularColor: *gglm.NewVec3(1, 1, 1),
			// These must be cosine values
			InnerCutoffRad: 15 * gglm.Deg2Rad,
			OuterCutoffRad: 20 * gglm.Deg2Rad,

			NearPlane: 1,
			FarPlane:  30,
		},
	}
)

type Game struct {
	WinWidth  int32
	WinHeight int32
	Win       *engine.Window
	ImGUIInfo nmageimgui.ImguiInfo
}

func main() {

	//Init engine
	err := engine.Init()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to init nMage. Err:", err)
	}

	//Create window
	dpiScaling = getDpiScaling(unscaledWindowWidth, unscaledWindowHeight)
	window, err = engine.CreateOpenGLWindowCentered("nMage", int32(unscaledWindowWidth*dpiScaling), int32(unscaledWindowHeight*dpiScaling), engine.WindowFlags_RESIZABLE, rend3dgl.NewRend3DGL())
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create window. Err: ", err)
	}
	defer window.Destroy()

	engine.SetMSAA(true)
	engine.SetVSync(false)
	engine.SetSrgbFramebuffer(true)

	game := &Game{
		Win:       window,
		WinWidth:  int32(unscaledWindowWidth * dpiScaling),
		WinHeight: int32(unscaledWindowHeight * dpiScaling),
		ImGUIInfo: nmageimgui.NewImGui("./res/shaders/imgui.glsl"),
	}
	window.EventCallbacks = append(window.EventCallbacks, game.handleWindowEvents)

	engine.Run(game, window, game.ImGUIInfo)
}

func (g *Game) handleWindowEvents(e sdl.Event) {

	switch e := e.(type) {
	case *sdl.WindowEvent:
		if e.Event == sdl.WINDOWEVENT_SIZE_CHANGED {

			g.WinWidth = e.Data1
			g.WinHeight = e.Data2
			cam.AspectRatio = float32(g.WinWidth) / float32(g.WinHeight)

			cam.Update()
			updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
		}
	}
}

func getDpiScaling(unscaledWindowWidth, unscaledWindowHeight int32) float32 {

	// Great read on DPI here: https://nlguillemot.wordpress.com/2016/12/11/high-dpi-rendering/

	// The no-scaling DPI on different platforms (e.g. when scale=100% on windows)
	var defaultDpi float32 = 96
	if runtime.GOOS == "windows" {
		defaultDpi = 96
	} else if runtime.GOOS == "darwin" {
		defaultDpi = 72
	}

	// Current DPI of the monitor
	_, dpiHorizontal, _, err := sdl.GetDisplayDPI(0)
	if err != nil {
		dpiHorizontal = defaultDpi
		fmt.Printf("Failed to get DPI with error '%s'. Using default DPI of '%f'\n", err.Error(), defaultDpi)
	}

	// Scaling factor (e.g. will be 1.25 for 125% scaling on windows)
	dpiScaling := dpiHorizontal / defaultDpi

	fmt.Printf(
		"Default DPI=%f\nHorizontal DPI=%f\nDPI scaling=%f\nUnscaled window size (width, height)=(%d, %d)\nScaled window size (width, height)=(%d, %d)\n\n",
		defaultDpi,
		dpiHorizontal,
		dpiScaling,
		unscaledWindowWidth, unscaledWindowHeight,
		int32(float32(unscaledWindowWidth)*dpiScaling), int32(float32(unscaledWindowHeight)*dpiScaling),
	)

	return dpiScaling
}

func (g *Game) Init() {

	var err error

	// Camera
	winWidth, winHeight := g.Win.SDLWin.GetSize()
	cam = camera.NewPerspective(
		gglm.NewVec3(0, 0, 10),
		gglm.NewVec3(0, 0, -1),
		gglm.NewVec3(0, 1, 0),
		0.1, 200,
		45*gglm.Deg2Rad,
		float32(winWidth)/float32(winHeight),
	)

	//Load meshes
	cubeMesh, err = meshes.NewMesh("Cube", "./res/models/cube.fbx", 0)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load mesh. Err: ", err)
	}

	sphereMesh, err = meshes.NewMesh("Sphere", "./res/models/sphere.fbx", 0)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load mesh. Err: ", err)
	}

	chairMesh, err = meshes.NewMesh("Chair", "./res/models/chair.fbx", 0)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load mesh. Err: ", err)
	}

	skyboxMesh, err = meshes.NewMesh("Skybox", "./res/models/skybox-cube.obj", 0)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load mesh. Err: ", err)
	}

	//Load textures
	whiteTex, err := assets.LoadTexturePNG("./res/textures/white.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	blackTex, err := assets.LoadTexturePNG("./res/textures/black.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	containerDiffuseTex, err := assets.LoadTexturePNG("./res/textures/container-diffuse.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	containerSpecularTex, err := assets.LoadTexturePNG("./res/textures/container-specular.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	palleteTex, err := assets.LoadTexturePNG("./res/textures/pallete-endesga-64-1x.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	skyboxCmap, err = assets.LoadCubemapTextures(
		"./res/textures/sb-right.jpg", "./res/textures/sb-left.jpg",
		"./res/textures/sb-top.jpg", "./res/textures/sb-bottom.jpg",
		"./res/textures/sb-front.jpg", "./res/textures/sb-back.jpg",
		&assets.TextureLoadOptions{},
	)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load cubemap. Err: ", err)
	}

	//
	// Create materials and assign any unused texture slots to black
	//
	screenQuadMat = materials.NewMaterial("Screen Quad Mat", "./res/shaders/screen-quad.glsl")
	screenQuadMat.SetUnifVec2("scale", demoFboScale)
	screenQuadMat.SetUnifVec2("offset", demoFboOffset)
	screenQuadMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))

	unlitMat = materials.NewMaterial("Unlit mat", "./res/shaders/simple-unlit.glsl")
	unlitMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))

	whiteMat = materials.NewMaterial("White mat", "./res/shaders/simple.glsl")
	whiteMat.Shininess = 64
	whiteMat.DiffuseTex = whiteTex.TexID
	whiteMat.SpecularTex = blackTex.TexID
	whiteMat.NormalTex = blackTex.TexID
	whiteMat.EmissionTex = blackTex.TexID
	whiteMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	whiteMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	// whiteMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	whiteMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	whiteMat.SetUnifVec3("ambientColor", ambientColor)
	whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
	whiteMat.SetUnifVec3("dirLight.dir", &dirLight.Dir)
	whiteMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
	whiteMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
	whiteMat.SetUnifInt32("dirLight.shadowMap", int32(materials.TextureSlot_ShadowMap1))
	whiteMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	whiteMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	containerMat = materials.NewMaterial("Container mat", "./res/shaders/simple.glsl")
	containerMat.Shininess = 64
	containerMat.DiffuseTex = containerDiffuseTex.TexID
	containerMat.SpecularTex = containerSpecularTex.TexID
	containerMat.NormalTex = blackTex.TexID
	containerMat.EmissionTex = blackTex.TexID
	containerMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	containerMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	// containerMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	containerMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	containerMat.SetUnifVec3("ambientColor", ambientColor)
	containerMat.SetUnifFloat32("material.shininess", containerMat.Shininess)
	containerMat.SetUnifVec3("dirLight.dir", &dirLight.Dir)
	containerMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
	containerMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
	containerMat.SetUnifInt32("dirLight.shadowMap", int32(materials.TextureSlot_ShadowMap1))
	containerMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	containerMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	palleteMat = materials.NewMaterial("Pallete mat", "./res/shaders/simple.glsl")
	palleteMat.Shininess = 64
	palleteMat.DiffuseTex = palleteTex.TexID
	palleteMat.SpecularTex = blackTex.TexID
	palleteMat.NormalTex = blackTex.TexID
	palleteMat.EmissionTex = blackTex.TexID
	palleteMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	palleteMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	// palleteMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	palleteMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	palleteMat.SetUnifVec3("ambientColor", ambientColor)
	palleteMat.SetUnifFloat32("material.shininess", palleteMat.Shininess)
	palleteMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
	palleteMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
	palleteMat.SetUnifInt32("dirLight.shadowMap", int32(materials.TextureSlot_ShadowMap1))
	palleteMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	palleteMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	debugDepthMat = materials.NewMaterial("Debug depth mat", "./res/shaders/debug-depth.glsl")

	depthMapMat = materials.NewMaterial("Depth Map mat", "./res/shaders/depth-map.glsl")

	arrayDepthMapMat = materials.NewMaterial("Array Depth Map mat", "./res/shaders/array-depth-map.glsl")

	omnidirDepthMapMat = materials.NewMaterial("Omnidirectional Depth Map mat", "./res/shaders/omnidirectional-depth-map.glsl")

	skyboxMat = materials.NewMaterial("Skybox mat", "./res/shaders/skybox.glsl")
	skyboxMat.CubemapTex = skyboxCmap.TexID
	skyboxMat.SetUnifInt32("skybox", int32(materials.TextureSlot_Cubemap))

	// Cube model mat
	translationMat := gglm.NewTranslationMat(gglm.NewVec3(0, 0, 0))
	scaleMat := gglm.NewScaleMat(gglm.NewVec3(1, 1, 1))
	rotMat := gglm.NewRotMat(gglm.NewQuatEuler(gglm.NewVec3(-90, -90, 0).AsRad()))
	cubeModelMat.Mul(translationMat.Mul(rotMat.Mul(scaleMat)))

	// Screen quad vao setup.
	// We don't actually care about the values here because the quad is hardcoded in the shader,
	// but we just want to have a vao with 6 vertices and uv0 so opengl can be called properly
	screenQuadVbo := buffers.NewVertexBuffer(buffers.Element{ElementType: buffers.DataTypeVec3}, buffers.Element{ElementType: buffers.DataTypeVec2})
	screenQuadVbo.SetData(make([]float32, 6), buffers.BufUsage_Static)
	screenQuadVao = buffers.NewVertexArray()
	screenQuadVao.AddVertexBuffer(screenQuadVbo)

	// Fbos and lights
	g.initFbos()
	g.updateLights()

	// Initial camera update
	cam.Update()
	updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
}

func (g *Game) initFbos() {

	// Demo fbo
	demoFbo = buffers.NewFramebuffer(uint32(g.WinWidth), uint32(g.WinHeight))

	demoFbo.NewColorAttachment(
		buffers.FramebufferAttachmentType_Texture,
		buffers.FramebufferAttachmentDataFormat_SRGBA,
	)

	demoFbo.NewDepthStencilAttachment(
		buffers.FramebufferAttachmentType_Renderbuffer,
		buffers.FramebufferAttachmentDataFormat_Depth24Stencil8,
	)

	assert.T(demoFbo.IsComplete(), "Demo fbo is not complete after init")

	// Depth map fbo
	dirLightDepthMapFbo = buffers.NewFramebuffer(1024, 1024)
	dirLightDepthMapFbo.SetNoColorBuffer()
	dirLightDepthMapFbo.NewDepthAttachment(
		buffers.FramebufferAttachmentType_Texture,
		buffers.FramebufferAttachmentDataFormat_DepthF32,
	)

	assert.T(dirLightDepthMapFbo.IsComplete(), "Depth map fbo is not complete after init")

	// Point light depth map fbo
	pointLightDepthMapFbo = buffers.NewFramebuffer(1024, 1024)
	pointLightDepthMapFbo.SetNoColorBuffer()
	pointLightDepthMapFbo.NewDepthCubemapArrayAttachment(
		buffers.FramebufferAttachmentDataFormat_DepthF32,
		MaxPointLights,
	)

	assert.T(pointLightDepthMapFbo.IsComplete(), "Point light depth map fbo is not complete after init")

	// Spot light depth map fbo
	spotLightDepthMapFbo = buffers.NewFramebuffer(1024, 1024)
	spotLightDepthMapFbo.SetNoColorBuffer()
	spotLightDepthMapFbo.NewDepthTextureArrayAttachment(
		buffers.FramebufferAttachmentDataFormat_DepthF32,
		MaxSpotLights,
	)

	assert.T(spotLightDepthMapFbo.IsComplete(), "Spot light depth map fbo is not complete after init")
}

func (g *Game) updateLights() {

	// Directional light
	whiteMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id
	containerMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id
	palleteMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id

	// Point lights
	for i := 0; i < len(pointLights); i++ {

		p := &pointLights[i]
		indexString := "pointLights[" + strconv.Itoa(i) + "]"

		whiteMat.SetUnifVec3(indexString+".pos", &p.Pos)
		containerMat.SetUnifVec3(indexString+".pos", &p.Pos)
		palleteMat.SetUnifVec3(indexString+".pos", &p.Pos)

		whiteMat.SetUnifVec3(indexString+".diffuseColor", &p.DiffuseColor)
		containerMat.SetUnifVec3(indexString+".diffuseColor", &p.DiffuseColor)
		palleteMat.SetUnifVec3(indexString+".diffuseColor", &p.DiffuseColor)

		whiteMat.SetUnifVec3(indexString+".specularColor", &p.SpecularColor)
		containerMat.SetUnifVec3(indexString+".specularColor", &p.SpecularColor)
		palleteMat.SetUnifVec3(indexString+".specularColor", &p.SpecularColor)

		whiteMat.SetUnifFloat32(indexString+".constant", p.Constant)
		containerMat.SetUnifFloat32(indexString+".constant", p.Constant)
		palleteMat.SetUnifFloat32(indexString+".constant", p.Constant)

		whiteMat.SetUnifFloat32(indexString+".linear", p.Linear)
		containerMat.SetUnifFloat32(indexString+".linear", p.Linear)
		palleteMat.SetUnifFloat32(indexString+".linear", p.Linear)

		whiteMat.SetUnifFloat32(indexString+".quadratic", p.Quadratic)
		containerMat.SetUnifFloat32(indexString+".quadratic", p.Quadratic)
		palleteMat.SetUnifFloat32(indexString+".quadratic", p.Quadratic)

		whiteMat.SetUnifFloat32(indexString+".farPlane", p.FarPlane)
		containerMat.SetUnifFloat32(indexString+".farPlane", p.FarPlane)
		palleteMat.SetUnifFloat32(indexString+".farPlane", p.FarPlane)
	}

	whiteMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id
	containerMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id
	palleteMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id

	// Spotlights
	for i := 0; i < len(spotLights); i++ {

		l := &spotLights[i]
		innerCutoffCos := l.InnerCutoffCos()
		outerCutoffCos := l.OuterCutoffCos()

		indexString := "spotLights[" + strconv.Itoa(i) + "]"

		whiteMat.SetUnifVec3(indexString+".pos", &l.Pos)
		containerMat.SetUnifVec3(indexString+".pos", &l.Pos)
		palleteMat.SetUnifVec3(indexString+".pos", &l.Pos)

		whiteMat.SetUnifVec3(indexString+".dir", &l.Dir)
		containerMat.SetUnifVec3(indexString+".dir", &l.Dir)
		palleteMat.SetUnifVec3(indexString+".dir", &l.Dir)

		whiteMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)
		containerMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)
		palleteMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)

		whiteMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)
		containerMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)
		palleteMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)

		whiteMat.SetUnifFloat32(indexString+".innerCutoff", innerCutoffCos)
		containerMat.SetUnifFloat32(indexString+".innerCutoff", innerCutoffCos)
		palleteMat.SetUnifFloat32(indexString+".innerCutoff", innerCutoffCos)

		whiteMat.SetUnifFloat32(indexString+".outerCutoff", outerCutoffCos)
		containerMat.SetUnifFloat32(indexString+".outerCutoff", outerCutoffCos)
		palleteMat.SetUnifFloat32(indexString+".outerCutoff", outerCutoffCos)
	}

	whiteMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
	containerMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
	palleteMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
}

func (g *Game) Update() {

	if input.IsQuitClicked() || input.KeyClicked(sdl.K_ESCAPE) {
		engine.Quit()
	}

	g.updateCameraLookAround()
	g.updateCameraPos()

	g.showDebugWindow()

	if input.KeyClicked(sdl.K_F4) {
		fmt.Printf("Pos: %s; Forward: %s; |Forward|: %f\n", cam.Pos.String(), cam.Forward.String(), cam.Forward.Mag())
	}

	g.Win.SDLWin.SetTitle(fmt.Sprint("nMage (", timing.GetAvgFPS(), " fps)"))
}

func (g *Game) showDebugWindow() {

	imgui.ShowDemoWindow()

	imgui.Begin("Debug controls")

	// Camera
	imgui.Text("Camera")
	if imgui.DragFloat3("Cam Pos", &cam.Pos.Data) {
		cam.Update()
		updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
	}
	if imgui.DragFloat3("Cam Forward", &cam.Forward.Data) {
		cam.Update()
		updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
	}

	imgui.Spacing()

	// Ambient light
	imgui.Text("Ambient Light")

	if imgui.DragFloat3("Ambient Color", &ambientColor.Data) {
		whiteMat.SetUnifVec3("ambientColor", ambientColor)
		containerMat.SetUnifVec3("ambientColor", ambientColor)
		palleteMat.SetUnifVec3("ambientColor", ambientColor)
	}

	imgui.Spacing()

	// Directional light
	imgui.Text("Directional Light")

	imgui.Checkbox("Render Directional Light Shadows", &renderDirLightShadows)

	if imgui.DragFloat3("Direction", &dirLight.Dir.Data) {
		whiteMat.SetUnifVec3("dirLight.dir", &dirLight.Dir)
		containerMat.SetUnifVec3("dirLight.dir", &dirLight.Dir)
		palleteMat.SetUnifVec3("dirLight.dir", &dirLight.Dir)
	}

	if imgui.DragFloat3("Diffuse Color", &dirLight.DiffuseColor.Data) {
		whiteMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
		containerMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
		palleteMat.SetUnifVec3("dirLight.diffuseColor", &dirLight.DiffuseColor)
	}

	if imgui.DragFloat3("Specular Color", &dirLight.SpecularColor.Data) {
		whiteMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
		containerMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
		palleteMat.SetUnifVec3("dirLight.specularColor", &dirLight.SpecularColor)
	}

	imgui.DragFloat3("dPos", &dirLightPos.Data)
	imgui.DragFloat("dSize", &dirLightSize)
	imgui.DragFloat("dNear", &dirLightNear)
	imgui.DragFloat("dFar", &dirLightFar)

	imgui.Spacing()

	// Specular
	imgui.Text("Specular Settings")

	if imgui.DragFloat("Specular Shininess", &whiteMat.Shininess) {
		whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		containerMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		palleteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
	}

	imgui.Spacing()

	// Point lights
	imgui.Checkbox("Render Point Light Shadows", &renderPointLightShadows)
	if imgui.BeginListBoxV("Point Lights", imgui.Vec2{Y: 200}) {

		for i := 0; i < len(pointLights); i++ {

			pl := &pointLights[i]
			indexNumString := strconv.Itoa(i)

			if !imgui.TreeNodeExStrV("Point Light "+indexNumString, imgui.TreeNodeFlagsSpanAvailWidth) {
				continue
			}

			indexString := "pointLights[" + indexNumString + "]"

			if imgui.DragFloat3("Pos", &pl.Pos.Data) {
				whiteMat.SetUnifVec3(indexString+".pos", &pl.Pos)
				containerMat.SetUnifVec3(indexString+".pos", &pl.Pos)
				palleteMat.SetUnifVec3(indexString+".pos", &pl.Pos)
			}

			if imgui.DragFloat3("Diffuse Color", &pl.DiffuseColor.Data) {
				whiteMat.SetUnifVec3(indexString+".diffuseColor", &pl.DiffuseColor)
				containerMat.SetUnifVec3(indexString+".diffuseColor", &pl.DiffuseColor)
				palleteMat.SetUnifVec3(indexString+".diffuseColor", &pl.DiffuseColor)
			}

			if imgui.DragFloat3("Specular Color", &pl.SpecularColor.Data) {
				whiteMat.SetUnifVec3(indexString+".specularColor", &pl.SpecularColor)
				containerMat.SetUnifVec3(indexString+".specularColor", &pl.SpecularColor)
				palleteMat.SetUnifVec3(indexString+".specularColor", &pl.SpecularColor)
			}

			imgui.TreePop()
		}

		imgui.EndListBox()
	}

	// Spot lights
	imgui.Checkbox("Render Spot Light Shadows", &renderSpotLightShadows)

	if imgui.BeginListBoxV("Spot Lights", imgui.Vec2{Y: 200}) {

		for i := 0; i < len(spotLights); i++ {

			l := &spotLights[i]
			indexNumString := strconv.Itoa(i)

			if !imgui.TreeNodeExStrV("Spot Light "+indexNumString, imgui.TreeNodeFlagsSpanAvailWidth) {
				continue
			}

			indexString := "spotLights[" + indexNumString + "]"

			if imgui.DragFloat3("Pos", &l.Pos.Data) {
				whiteMat.SetUnifVec3(indexString+".pos", &l.Pos)
				containerMat.SetUnifVec3(indexString+".pos", &l.Pos)
				palleteMat.SetUnifVec3(indexString+".pos", &l.Pos)
			}

			if imgui.DragFloat3("Dir", &l.Dir.Data) {
				whiteMat.SetUnifVec3(indexString+".dir", &l.Dir)
				containerMat.SetUnifVec3(indexString+".dir", &l.Dir)
				palleteMat.SetUnifVec3(indexString+".dir", &l.Dir)
			}

			if imgui.DragFloat3("Diffuse Color", &l.DiffuseColor.Data) {
				whiteMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)
				containerMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)
				palleteMat.SetUnifVec3(indexString+".diffuseColor", &l.DiffuseColor)
			}

			if imgui.DragFloat3("Specular Color", &l.SpecularColor.Data) {
				whiteMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)
				containerMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)
				palleteMat.SetUnifVec3(indexString+".specularColor", &l.SpecularColor)
			}

			if imgui.DragFloat("Inner Cutoff Radians", &l.InnerCutoffRad) {

				cos := l.InnerCutoffCos()

				whiteMat.SetUnifFloat32(indexString+".innerCutoff", cos)
				containerMat.SetUnifFloat32(indexString+".innerCutoff", cos)
				palleteMat.SetUnifFloat32(indexString+".innerCutoff", cos)
			}

			if imgui.DragFloat("Outer Cutoff Radians", &l.OuterCutoffRad) {

				cos := l.OuterCutoffCos()

				whiteMat.SetUnifFloat32(indexString+".outerCutoff", cos)
				containerMat.SetUnifFloat32(indexString+".outerCutoff", cos)
				palleteMat.SetUnifFloat32(indexString+".outerCutoff", cos)
			}

			imgui.DragFloat("Spot Near Plane", &l.NearPlane)
			imgui.DragFloat("Spot Far Plane", &l.FarPlane)

			imgui.TreePop()
		}

		imgui.EndListBox()
	}

	// Demo fbo
	imgui.Text("Demo Framebuffer")
	imgui.Checkbox("Show FBO##0", &renderToDemoFbo)
	imgui.DragFloat2("Scale##0", &demoFboScale.Data)
	imgui.DragFloat2("Offset##0", &demoFboOffset.Data)

	// Depth map fbo
	imgui.Text("Directional Light Depth Map Framebuffer")
	imgui.Checkbox("Show FBO##1", &showDirLightDepthMapFbo)
	imgui.DragFloat2("Scale##1", &dirLightDepthMapFboScale.Data)
	imgui.DragFloat2("Offset##1", &dirLightDepthMapFboOffset.Data)

	// Other
	imgui.Text("Other Settings")

	imgui.Checkbox("Render skybox", &renderSkybox)
	imgui.Checkbox("Render to back buffer", &renderToBackBuffer)
	imgui.Checkbox("Render depth buffer", &renderDepthBuffer)

	imgui.End()
}

func (g *Game) updateCameraLookAround() {

	mouseX, mouseY := input.GetMouseMotion()
	if (mouseX == 0 && mouseY == 0) || !input.MouseDown(sdl.BUTTON_RIGHT) {
		return
	}

	// Yaw
	yaw += float32(mouseX) * mouseSensitivity * timing.DT()

	// Pitch
	pitch += float32(-mouseY) * mouseSensitivity * timing.DT()
	if pitch > 1.5 {
		pitch = 1.5
	}

	if pitch < -1.5 {
		pitch = -1.5
	}

	// Update cam forward
	cam.UpdateRotation(pitch, yaw)

	updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
}

func (g *Game) updateCameraPos() {

	update := false

	var camSpeedScale float32 = 1.0
	if input.KeyDown(sdl.K_LSHIFT) {
		camSpeedScale = 2
	}

	// Forward and backward
	if input.KeyDown(sdl.K_w) {
		cam.Pos.Add(cam.Forward.Clone().Scale(camSpeed * camSpeedScale * timing.DT()))
		update = true
	} else if input.KeyDown(sdl.K_s) {
		cam.Pos.Add(cam.Forward.Clone().Scale(-camSpeed * camSpeedScale * timing.DT()))
		update = true
	}

	// Left and right
	if input.KeyDown(sdl.K_d) {
		cam.Pos.Add(gglm.Cross(&cam.Forward, &cam.WorldUp).Normalize().Scale(camSpeed * camSpeedScale * timing.DT()))
		update = true
	} else if input.KeyDown(sdl.K_a) {
		cam.Pos.Add(gglm.Cross(&cam.Forward, &cam.WorldUp).Normalize().Scale(-camSpeed * camSpeedScale * timing.DT()))
		update = true
	}

	if update {
		cam.Update()
		updateAllProjViewMats(cam.ProjMat, cam.ViewMat)
	}
}

var (
	renderDirLightShadows   = true
	renderPointLightShadows = true
	renderSpotLightShadows  = true

	rotatingCubeSpeedDeg1 float32 = 45
	rotatingCubeSpeedDeg2 float32 = 120
	rotatingCubeSpeedDeg3 float32 = 120
	rotatingCubeTrMat1            = *gglm.NewTrMatId().Translate(gglm.NewVec3(-4, -1, 4))
	rotatingCubeTrMat2            = *gglm.NewTrMatId().Translate(gglm.NewVec3(-1, 0.5, 4))
	rotatingCubeTrMat3            = *gglm.NewTrMatId().Translate(gglm.NewVec3(5, 0.5, 4))
)

func (g *Game) Render() {

	whiteMat.SetUnifVec3("camPos", &cam.Pos)
	containerMat.SetUnifVec3("camPos", &cam.Pos)
	palleteMat.SetUnifVec3("camPos", &cam.Pos)

	rotatingCubeTrMat1.Rotate(rotatingCubeSpeedDeg1*gglm.Deg2Rad*timing.DT(), gglm.NewVec3(0, 1, 0))
	rotatingCubeTrMat2.Rotate(rotatingCubeSpeedDeg2*gglm.Deg2Rad*timing.DT(), gglm.NewVec3(1, 1, 0))
	rotatingCubeTrMat3.Rotate(rotatingCubeSpeedDeg3*gglm.Deg2Rad*timing.DT(), gglm.NewVec3(1, 1, 1))

	if renderDirLightShadows {
		g.renderDirectionalLightShadowmap()
	}

	if renderSpotLightShadows {
		g.renderSpotLightShadowmaps()
	}

	if renderPointLightShadows {
		g.renderPointLightShadowmaps()
	}

	if renderToBackBuffer {

		if renderDepthBuffer {
			g.RenderScene(debugDepthMat)
		} else {
			g.RenderScene(nil)
		}
	}

	if renderSkybox {
		g.DrawSkybox()
	}

	if renderToDemoFbo {
		g.renderDemoFob()
	}
}

func (g *Game) renderDirectionalLightShadowmap() {

	// Set some uniforms
	dirLightProjViewMat := dirLight.GetProjViewMat()

	whiteMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)
	containerMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)
	palleteMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)

	depthMapMat.SetUnifMat4("projViewMat", &dirLightProjViewMat)

	// Start rendering
	dirLightDepthMapFbo.BindWithViewport()
	dirLightDepthMapFbo.Clear()

	// Culling front faces helps 'peter panning' when
	// drawing shadow maps, but works only for solids with a back face (i.e. quads won't cast shadows).
	// Check more here: https://learnopengl.com/Advanced-Lighting/Shadows/Shadow-Mapping
	//
	// Some note that this is too troublesome and fails in many cases. Might be better to remove.
	gl.CullFace(gl.FRONT)
	g.RenderScene(depthMapMat)
	gl.CullFace(gl.BACK)

	dirLightDepthMapFbo.UnBindWithViewport(uint32(g.WinWidth), uint32(g.WinHeight))

	if showDirLightDepthMapFbo {
		screenQuadMat.DiffuseTex = dirLightDepthMapFbo.Attachments[0].Id
		screenQuadMat.SetUnifVec2("offset", dirLightDepthMapFboOffset)
		screenQuadMat.SetUnifVec2("scale", dirLightDepthMapFboScale)
		screenQuadMat.Bind()
		window.Rend.DrawVertexArray(screenQuadMat, &screenQuadVao, 0, 6)
	}
}

func (g *Game) renderSpotLightShadowmaps() {

	for i := 0; i < len(spotLights); i++ {

		l := &spotLights[i]
		indexStr := strconv.Itoa(i)
		projViewMatIndexStr := "spotLightProjViewMats[" + indexStr + "]"

		// Set render uniforms
		projViewMat := l.GetProjViewMat()

		whiteMat.SetUnifMat4(projViewMatIndexStr, &projViewMat)
		containerMat.SetUnifMat4(projViewMatIndexStr, &projViewMat)
		palleteMat.SetUnifMat4(projViewMatIndexStr, &projViewMat)

		// Set depth uniforms
		arrayDepthMapMat.SetUnifMat4("projViewMats["+indexStr+"]", &projViewMat)
	}

	// Render
	spotLightDepthMapFbo.BindWithViewport()
	spotLightDepthMapFbo.Clear()

	// Front culling created issues
	// gl.CullFace(gl.FRONT)
	g.RenderScene(arrayDepthMapMat)
	// gl.CullFace(gl.BACK)

	spotLightDepthMapFbo.UnBindWithViewport(uint32(g.WinWidth), uint32(g.WinHeight))
}

func (g *Game) renderPointLightShadowmaps() {

	pointLightDepthMapFbo.BindWithViewport()
	pointLightDepthMapFbo.Clear()

	for i := 0; i < len(pointLights); i++ {

		p := &pointLights[i]

		// Generic uniforms
		omnidirDepthMapMat.SetUnifVec3("lightPos", &p.Pos)
		omnidirDepthMapMat.SetUnifInt32("cubemapIndex", int32(i))
		omnidirDepthMapMat.SetUnifFloat32("farPlane", p.FarPlane)

		// Set projView matrices
		projViewMats := p.GetProjViewMats(float32(pointLightDepthMapFbo.Width), float32(pointLightDepthMapFbo.Height))
		for j := 0; j < len(projViewMats); j++ {
			omnidirDepthMapMat.SetUnifMat4("cubemapProjViewMats["+strconv.Itoa(j)+"]", &projViewMats[j])
		}

		g.RenderScene(omnidirDepthMapMat)
	}

	pointLightDepthMapFbo.UnBindWithViewport(uint32(g.WinWidth), uint32(g.WinHeight))
}

func (g *Game) renderDemoFob() {

	demoFbo.Bind()
	demoFbo.Clear()

	if renderDepthBuffer {
		g.RenderScene(debugDepthMat)
	} else {
		g.RenderScene(nil)
	}

	if renderSkybox {
		g.DrawSkybox()
	}

	demoFbo.UnBind()

	screenQuadMat.DiffuseTex = demoFbo.Attachments[0].Id
	screenQuadMat.SetUnifVec2("offset", demoFboOffset)
	screenQuadMat.SetUnifVec2("scale", demoFboScale)

	window.Rend.DrawVertexArray(screenQuadMat, &screenQuadVao, 0, 6)
}

func (g *Game) RenderScene(overrideMat *materials.Material) {

	tempModelMatrix := cubeModelMat.Clone()

	// See if we need overrides
	sunMat := palleteMat
	chairMat := palleteMat
	cubeMat := containerMat

	if overrideMat != nil {
		sunMat = overrideMat
		chairMat = overrideMat
		cubeMat = overrideMat
	}

	// Draw dir light
	window.Rend.DrawMesh(sphereMesh, gglm.NewTrMatId().Translate(gglm.NewVec3(0, 10, 0)).Scale(gglm.NewVec3(0.1, 0.1, 0.1)), sunMat)

	// Draw point lights
	for i := 0; i < len(pointLights); i++ {

		pl := &pointLights[i]
		window.Rend.DrawMesh(cubeMesh, gglm.NewTrMatId().Translate(&pl.Pos).Scale(gglm.NewVec3(0.1, 0.1, 0.1)), sunMat)
	}

	// Chair
	window.Rend.DrawMesh(chairMesh, tempModelMatrix, chairMat)

	// Ground
	window.Rend.DrawMesh(cubeMesh, gglm.NewTrMatId().Translate(gglm.NewVec3(0, -3, 0)).Scale(gglm.NewVec3(20, 1, 20)), cubeMat)

	// Cubes
	tempModelMatrix.Translate(gglm.NewVec3(-6, 0, 0))
	window.Rend.DrawMesh(cubeMesh, tempModelMatrix, cubeMat)

	tempModelMatrix.Translate(gglm.NewVec3(0, -1, -4))
	window.Rend.DrawMesh(cubeMesh, tempModelMatrix, cubeMat)

	// Rotating cubes
	window.Rend.DrawMesh(cubeMesh, &rotatingCubeTrMat1, cubeMat)
	window.Rend.DrawMesh(cubeMesh, &rotatingCubeTrMat2, cubeMat)
	window.Rend.DrawMesh(cubeMesh, &rotatingCubeTrMat3, cubeMat)

	// Cubes generator
	// rowSize := 1
	// for y := 0; y < rowSize; y++ {
	// 	for x := 0; x < rowSize; x++ {
	// 		tempModelMatrix.Translate(gglm.NewVec3(-6, 0, 0))
	// 		window.Rend.DrawMesh(cubeMesh, tempModelMatrix, cubeMat)
	// 	}
	// 	tempModelMatrix.Translate(gglm.NewVec3(float32(rowSize), -1, 0))
	// }
}

func (g *Game) DrawSkybox() {

	gl.Disable(gl.CULL_FACE)
	gl.DepthFunc(gl.LEQUAL)

	window.Rend.DrawCubemap(skyboxMesh, skyboxMat)

	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.CULL_FACE)
}

func (g *Game) FrameEnd() {
}

func (g *Game) DeInit() {
	g.Win.Destroy()
}

func updateAllProjViewMats(projMat, viewMat gglm.Mat4) {

	projViewMat := projMat.Clone().Mul(&viewMat)

	unlitMat.SetUnifMat4("projViewMat", projViewMat)
	whiteMat.SetUnifMat4("projViewMat", projViewMat)
	containerMat.SetUnifMat4("projViewMat", projViewMat)
	palleteMat.SetUnifMat4("projViewMat", projViewMat)
	debugDepthMat.SetUnifMat4("projViewMat", projViewMat)

	// Update skybox projViewMat
	skyboxViewMat := viewMat.Clone()
	skyboxViewMat.Set(0, 3, 0)
	skyboxViewMat.Set(1, 3, 0)
	skyboxViewMat.Set(2, 3, 0)
	skyboxViewMat.Set(3, 0, 0)
	skyboxViewMat.Set(3, 1, 0)
	skyboxViewMat.Set(3, 2, 0)
	skyboxViewMat.Set(3, 3, 0)
	skyboxMat.SetUnifMat4("projViewMat", projMat.Clone().Mul(skyboxViewMat))
}
