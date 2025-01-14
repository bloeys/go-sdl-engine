package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
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
		- Create VAO struct independent from VBO to support multi-VBO use cases (e.g. instancing) ✅
		- Normals maps ✅
		- HDR ✅
		- Fix bad point light acne ✅
		- UBO support ✅
		- Skeletal animations
		- (?) Cascaded shadow mapping
	- In some cases we DO want input even when captured by UI. We need two systems within input package, one filtered and one not✅
	- (?) Support OpenGL 4.1 and 4.6, and default to 4.6
	- Proper model loading (i.e. load model by reading all its meshes, textures, and so on together)
	- Renderer batching
	- Scene graph
	- (?) Separate engine loop from rendering loop
	- Frustum culling
	- Proper Asset loading system
	- Material system editor with fields automatically extracted from the shader
*/

type DirLight struct {
	Dir           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3
}

var (
	renderDirLightShadows   = true
	renderPointLightShadows = true
	renderSpotLightShadows  = true

	dirLightSize float32 = 30
	dirLightNear float32 = 0.1
	dirLightFar  float32 = 30
	dirLightPos          = gglm.NewVec3(0, 10, 0)

	pointLightRadiusToFarPlaneRatio float32 = 1.25
)

func (d *DirLight) GetProjViewMat() gglm.Mat4 {

	// Some arbitrary position for the directional light
	pos := dirLightPos

	size := dirLightSize
	nearClip := dirLightNear
	farClip := dirLightFar

	up := gglm.NewVec3(0, 1, 0)
	projMat := gglm.Ortho(-size, size, -size, size, nearClip, farClip).Mat4
	viewMat := gglm.LookAtRH(&pos, pos.Clone().Add(&d.Dir), &up).Mat4

	return *projMat.Mul(&viewMat)
}

// Based on: https://lisyarus.github.io/blog/posts/point-light-attenuation.html
type PointLight struct {
	Pos           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3

	Radius  float32
	Falloff float32

	// MaxBias is the max shadow bias applied for this light.
	// A usual value is 0.05
	MaxBias float32

	// NearPlane is the distance where if the pixel
	// is closer to the light than this distance, no shadow will be casted.
	//
	// This helps not produce shadows from within objects.
	// Same idea a camera near plane.
	NearPlane float32

	// Far plane is the max distance at which shadows from this
	// light will show.
	//
	// This should be a bit bigger than the radius, as an object
	// at the edge of the radius should still cast a shadow, and
	// so this shadow will be further than the radius.
	//
	// Something like 'FarPlane=Radius*1.25' might work.
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

	targetPos0 := gglm.NewVec3(1+p.Pos.X(), p.Pos.Y(), p.Pos.Z())
	targetPos1 := gglm.NewVec3(-1+p.Pos.X(), p.Pos.Y(), p.Pos.Z())
	targetPos2 := gglm.NewVec3(p.Pos.X(), 1+p.Pos.Y(), p.Pos.Z())
	targetPos3 := gglm.NewVec3(p.Pos.X(), -1+p.Pos.Y(), p.Pos.Z())
	targetPos4 := gglm.NewVec3(p.Pos.X(), p.Pos.Y(), 1+p.Pos.Z())
	targetPos5 := gglm.NewVec3(p.Pos.X(), p.Pos.Y(), -1+p.Pos.Z())

	worldUp0 := gglm.NewVec3(0, -1, 0)
	worldUp1 := gglm.NewVec3(0, -1, 0)
	worldUp2 := gglm.NewVec3(0, 0, 1)
	worldUp3 := gglm.NewVec3(0, 0, -1)
	worldUp4 := gglm.NewVec3(0, -1, 0)
	worldUp5 := gglm.NewVec3(0, -1, 0)

	lookAt0 := gglm.LookAtRH(&p.Pos, &targetPos0, &worldUp0)
	lookAt1 := gglm.LookAtRH(&p.Pos, &targetPos1, &worldUp1)
	lookAt2 := gglm.LookAtRH(&p.Pos, &targetPos2, &worldUp2)
	lookAt3 := gglm.LookAtRH(&p.Pos, &targetPos3, &worldUp3)
	lookAt4 := gglm.LookAtRH(&p.Pos, &targetPos4, &worldUp4)
	lookAt5 := gglm.LookAtRH(&p.Pos, &targetPos5, &worldUp5)

	projViewMats := [6]gglm.Mat4{
		*projMat.Clone().Mul(&lookAt0.Mat4),
		*projMat.Clone().Mul(&lookAt1.Mat4),
		*projMat.Clone().Mul(&lookAt2.Mat4),
		*projMat.Clone().Mul(&lookAt3.Mat4),
		*projMat.Clone().Mul(&lookAt4.Mat4),
		*projMat.Clone().Mul(&lookAt5.Mat4),
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
	if gglm.Abs32(gglm.DotVec3(&s.Dir, &up)) > 0.99 {
		up.SetXY(1, 0)
	}

	viewMat := gglm.LookAtRH(&s.Pos, s.Pos.Clone().Add(&s.Dir), &up).Mat4

	return *projMat.Mul(&viewMat)
}

func (s *SpotLight) InnerCutoffCos() float32 {
	return gglm.Cos32(s.InnerCutoffRad)
}

func (s *SpotLight) OuterCutoffCos() float32 {
	return gglm.Cos32(s.OuterCutoffRad)
}

type GlobalMatricesUboData struct {
	CamPos      gglm.Vec3
	ProjViewMat gglm.Mat4
}

type DirLightUboData struct {
	Dir           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3
}

type PointLightUboData struct {
	Pos           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3
	Radius        float32
	Falloff       float32
	MaxBias       float32
	NearPlane     float32
	FarPlane      float32
}

type SpotLightUboData struct {
	Pos           gglm.Vec3
	Dir           gglm.Vec3
	DiffuseColor  gglm.Vec3
	SpecularColor gglm.Vec3
	InnerCutoff   float32
	OuterCutoff   float32
}

type LightsUboData struct {
	DirLight     DirLightUboData
	PointLights  [POINT_LIGHT_COUNT]PointLightUboData
	SpotLights   [SPOT_LIGHT_COUNT]SpotLightUboData
	AmbientColor gglm.Vec3
}

const (

	// These must match the shader values
	POINT_LIGHT_COUNT = 8
	SPOT_LIGHT_COUNT  = 4

	UNSCALED_WINDOW_WIDTH  = 1280
	UNSCALED_WINDOW_HEIGHT = 720

	PROFILE_CPU = false
	PROFILE_MEM = false

	FRAME_TIME_MS_SAMPLES = 10000
)

var (
	globalMatricesUboData GlobalMatricesUboData
	globalMatricesUbo     buffers.UniformBuffer

	lightsUboData LightsUboData
	lightsUbo     buffers.UniformBuffer

	frameTimesMsIndex int       = 0
	frameTimesMs      []float32 = make([]float32, 0, FRAME_TIME_MS_SAMPLES)

	camMoveSpeed float32 = 15
	camRotSpeed  float32 = 0.5

	window engine.Window

	pitch float32 = 0
	yaw   float32 = -1.5
	cam   camera.Camera

	renderToBackBuffer = true

	// Demo fbo
	renderToDemoFbo = false
	demoFboScale    = gglm.NewVec2(0.25, 0.25)
	demoFboOffset   = gglm.NewVec2(0.75, -0.75)
	demoFbo         buffers.Framebuffer

	// Dir light fbo
	showDirLightDepthMapFbo   = false
	dirLightDepthMapFboScale  = gglm.NewVec2(0.25, 0.25)
	dirLightDepthMapFboOffset = gglm.NewVec2(0.75, -0.2)
	dirLightDepthMapFbo       buffers.Framebuffer

	// Point light fbo
	pointLightDepthMapFbo buffers.Framebuffer

	// Spot light fbo
	spotLightDepthMapFbo buffers.Framebuffer

	// Hdr Fbo
	hdrRendering                    = true
	hdrExposure             float32 = 1
	tonemappedScreenQuadMat materials.Material
	hdrFbo                  buffers.Framebuffer

	screenQuadVao buffers.VertexArray
	screenQuadMat materials.Material

	unlitMat           materials.Material
	whiteMat           materials.Material
	containerMat       materials.Material
	groundMat          materials.Material
	palleteMat         materials.Material
	skyboxMat          materials.Material
	depthMapMat        materials.Material
	arrayDepthMapMat   materials.Material
	omnidirDepthMapMat materials.Material
	debugDepthMat      materials.Material

	cubeMesh   meshes.Mesh
	sphereMesh meshes.Mesh
	chairMesh  meshes.Mesh
	skyboxMesh meshes.Mesh

	cubeModelMat = gglm.NewTrMatId()

	renderSkybox      = true
	renderDepthBuffer = false

	skyboxCmap assets.Cubemap

	dpiScaling float32

	// Light settings
	dirLightDir = gglm.NewVec3(0, -0.5, -0.8)
	// Lights
	dirLight = DirLight{
		Dir:           *dirLightDir.Normalize(),
		DiffuseColor:  gglm.NewVec3(63.0/255, 63.0/255, 63.0/255),
		SpecularColor: gglm.NewVec3(1, 1, 1),
	}
	pointLights = [POINT_LIGHT_COUNT]PointLight{
		{
			Pos:           gglm.NewVec3(0, 4, -3),
			DiffuseColor:  gglm.NewVec3(1, 0, 0),
			SpecularColor: gglm.NewVec3(1, 1, 1),
			Radius:        10,
			Falloff:       1.0,
			MaxBias:       0.05,
			NearPlane:     0.2,
			FarPlane:      20 * pointLightRadiusToFarPlaneRatio,
		},
		{
			Pos:           gglm.NewVec3(5, 0, 0),
			DiffuseColor:  gglm.NewVec3(1, 1, 1),
			SpecularColor: gglm.NewVec3(1, 1, 1),
			Radius:        10,
			Falloff:       1.0,
			MaxBias:       0.05,
			NearPlane:     0.2,
			FarPlane:      20 * pointLightRadiusToFarPlaneRatio,
		},
		{
			Pos:           gglm.NewVec3(-3, 4, 3),
			DiffuseColor:  gglm.NewVec3(1, 1, 1),
			SpecularColor: gglm.NewVec3(1, 1, 1),
			Radius:        10,
			Falloff:       1.0,
			MaxBias:       0.05,
			NearPlane:     0.2,
			FarPlane:      20 * pointLightRadiusToFarPlaneRatio,
		},
	}

	spotLightDir0 = gglm.NewVec3(1.5, -0.9, 0)
	spotLights    = [SPOT_LIGHT_COUNT]SpotLight{
		{
			Pos:           gglm.NewVec3(-4, 7, 5),
			Dir:           *spotLightDir0.Normalize(),
			DiffuseColor:  gglm.NewVec3(1, 0, 1),
			SpecularColor: gglm.NewVec3(1, 1, 1),
			// These must be cosine values
			InnerCutoffRad: 15 * gglm.Deg2Rad,
			OuterCutoffRad: 20 * gglm.Deg2Rad,

			NearPlane: 2,
			FarPlane:  50,
		},
	}
)

type Game struct {
	WinWidth  int32
	WinHeight int32
	Win       *engine.Window
	Rend      *rend3dgl.Rend3DGL
	ImGUIInfo nmageimgui.ImguiInfo
}

func main() {

	//Init engine
	err := engine.Init()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to init nMage. Err:", err)
	}

	//Create window
	dpiScaling = getDpiScaling(UNSCALED_WINDOW_WIDTH, UNSCALED_WINDOW_HEIGHT)
	window, err = engine.CreateOpenGLWindowCentered("nMage", int32(UNSCALED_WINDOW_WIDTH*dpiScaling), int32(UNSCALED_WINDOW_HEIGHT*dpiScaling), engine.WindowFlags_RESIZABLE)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create window. Err: ", err)
	}
	defer window.Destroy()

	engine.SetMSAA(true)
	engine.SetVSync(false)
	engine.SetSrgbFramebuffer(true)

	game := &Game{
		Win:       &window,
		WinWidth:  int32(UNSCALED_WINDOW_WIDTH * dpiScaling),
		WinHeight: int32(UNSCALED_WINDOW_HEIGHT * dpiScaling),
		Rend:      rend3dgl.NewRend3DGL(),
		ImGUIInfo: nmageimgui.NewImGui("./res/shaders/imgui.glsl"),
	}
	window.EventCallbacks = append(window.EventCallbacks, game.handleWindowEvents)

	if PROFILE_CPU {

		pf, err := os.Create("cpu.pprof")
		if err == nil {
			defer pf.Close()
			pprof.StartCPUProfile(pf)
		} else {
			logging.ErrLog.Printf("Creating cpu.pprof file failed. CPU profiling will not run. Err=%v\n", err)
		}
	}

	window.SDLWin.SetTitle("nMage")
	engine.Run(game, &window, game.Rend, game.ImGUIInfo)

	if PROFILE_CPU {
		pprof.StopCPUProfile()
	}

	if PROFILE_MEM {

		heapProfile, err := os.Create("heap.pprof")
		if err == nil {

			err = pprof.WriteHeapProfile(heapProfile)
			if err != nil {
				logging.ErrLog.Printf("Writing heap profile to heap.pprof failed. Err=%v\n", err)
			}

			heapProfile.Close()

		} else {
			logging.ErrLog.Printf("Creating heap.pprof file failed. Err=%v\n", err)
		}
	}
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
		logging.ErrLog.Printf("Failed to get DPI with error '%s'. Using default DPI of '%f'\n", err.Error(), defaultDpi)
	}

	// Scaling factor (e.g. will be 1.25 for 125% scaling on windows)
	dpiScaling := dpiHorizontal / defaultDpi

	logging.InfoLog.Printf(
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

	camPos := gglm.NewVec3(0, 10, 20)
	camForward := gglm.NewVec3(0, 0, -1)
	camWorldUp := gglm.NewVec3(0, 1, 0)
	cam = camera.NewPerspective(
		&camPos,
		&camForward,
		&camWorldUp,
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

	brickwallDiffuseTex, err := assets.LoadTexturePNG("./res/textures/brickwall.png", &assets.TextureLoadOptions{})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	brickwallNormalTex, err := assets.LoadTexturePNG("./res/textures/brickwall-normal.png", &assets.TextureLoadOptions{NoSrgba: true})
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
	screenQuadMat.SetUnifVec2("scale", &demoFboScale)
	screenQuadMat.SetUnifVec2("offset", &demoFboOffset)
	screenQuadMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))

	tonemappedScreenQuadMat = materials.NewMaterial("Tonemapped Screen Quad Mat", "./res/shaders/tonemapped-screen-quad.glsl")
	tonemappedScreenQuadMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))

	unlitMat = materials.NewMaterial("Unlit mat", "./res/shaders/simple-unlit.glsl")
	unlitMat.Settings.Set(materials.MaterialSettings_HasModelMtx)
	unlitMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))

	whiteMat = materials.NewMaterial("White mat", "./res/shaders/simple.glsl")
	whiteMat.Settings.Set(materials.MaterialSettings_HasModelMtx)
	whiteMat.Shininess = 64
	whiteMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	whiteMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	whiteMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	whiteMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
	whiteMat.SetUnifInt32("dirLightShadowMap", int32(materials.TextureSlot_ShadowMap1))
	whiteMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	whiteMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	containerMat = materials.NewMaterial("Container mat", "./res/shaders/simple.glsl")
	containerMat.Settings.Set(materials.MaterialSettings_HasModelMtx)
	containerMat.Shininess = 64
	containerMat.DiffuseTex = containerDiffuseTex.TexID
	containerMat.SpecularTex = containerSpecularTex.TexID
	containerMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	containerMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	containerMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	containerMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	containerMat.SetUnifFloat32("material.shininess", containerMat.Shininess)
	containerMat.SetUnifInt32("dirLightShadowMap", int32(materials.TextureSlot_ShadowMap1))
	containerMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	containerMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	groundMat = materials.NewMaterial("Ground mat", "./res/shaders/simple.glsl")
	groundMat.Settings.Set(materials.MaterialSettings_HasModelMtx)
	groundMat.Shininess = 64
	groundMat.DiffuseTex = brickwallDiffuseTex.TexID
	groundMat.NormalTex = brickwallNormalTex.TexID
	groundMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	groundMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	groundMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	groundMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	groundMat.SetUnifFloat32("material.shininess", groundMat.Shininess)
	groundMat.SetUnifInt32("dirLightShadowMap", int32(materials.TextureSlot_ShadowMap1))
	groundMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	groundMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	palleteMat = materials.NewMaterial("Pallete mat", "./res/shaders/simple.glsl")
	palleteMat.Settings.Set(materials.MaterialSettings_HasModelMtx)
	palleteMat.Shininess = 64
	palleteMat.DiffuseTex = palleteTex.TexID
	palleteMat.SetUnifInt32("material.diffuse", int32(materials.TextureSlot_Diffuse))
	palleteMat.SetUnifInt32("material.specular", int32(materials.TextureSlot_Specular))
	palleteMat.SetUnifInt32("material.normal", int32(materials.TextureSlot_Normal))
	palleteMat.SetUnifInt32("material.emission", int32(materials.TextureSlot_Emission))
	palleteMat.SetUnifFloat32("material.shininess", palleteMat.Shininess)
	palleteMat.SetUnifInt32("dirLightShadowMap", int32(materials.TextureSlot_ShadowMap1))
	palleteMat.SetUnifInt32("pointLightCubeShadowMaps", int32(materials.TextureSlot_Cubemap_Array))
	palleteMat.SetUnifInt32("spotLightShadowMaps", int32(materials.TextureSlot_ShadowMap_Array1))

	debugDepthMat = materials.NewMaterial("Debug depth mat", "./res/shaders/debug-depth.glsl")
	debugDepthMat.Settings.Set(materials.MaterialSettings_HasModelMtx)

	depthMapMat = materials.NewMaterial("Depth Map mat", "./res/shaders/depth-map.glsl")
	depthMapMat.Settings.Set(materials.MaterialSettings_HasModelMtx)

	arrayDepthMapMat = materials.NewMaterial("Array Depth Map mat", "./res/shaders/array-depth-map.glsl")
	arrayDepthMapMat.Settings.Set(materials.MaterialSettings_HasModelMtx)

	omnidirDepthMapMat = materials.NewMaterial("Omnidirectional Depth Map mat", "./res/shaders/omnidirectional-depth-map.glsl")
	omnidirDepthMapMat.Settings.Set(materials.MaterialSettings_HasModelMtx)

	skyboxMat = materials.NewMaterial("Skybox mat", "./res/shaders/skybox.glsl")
	skyboxMat.CubemapTex = skyboxCmap.TexID
	skyboxMat.SetUnifInt32("skybox", int32(materials.TextureSlot_Cubemap))

	// Cube model mat
	translationMat := gglm.NewTranslationMat(0, 0, 0)

	scaleMat := gglm.NewScaleMat(1, 1, 1)

	rotMatRot := gglm.NewQuatEuler(-90*gglm.Deg2Rad, -90*gglm.Deg2Rad, 0)
	rotMat := gglm.NewRotMatQuat(&rotMatRot)
	cubeModelMat.Mul(translationMat.Mul(rotMat.Mul(&scaleMat)))

	// Screen quad vao setup.
	// We don't actually care about the values here because the quad is hardcoded in the shader,
	// but we just want to have a vao with 6 vertices and uv0 so opengl can be called properly
	screenQuadVbo := buffers.NewVertexBuffer(buffers.Element{ElementType: buffers.DataTypeVec3}, buffers.Element{ElementType: buffers.DataTypeVec2})
	screenQuadVbo.SetData(make([]float32, 6), buffers.BufUsage_Static_Draw)
	screenQuadVao = buffers.NewVertexArray()
	screenQuadVao.AddVertexBuffer(screenQuadVbo)

	// Fbos and lights
	g.initFbos()
	// Ubos
	g.initUbos()

	// Initial camera update
	cam.Update()
	updateAllProjViewMats(cam.ProjMat, cam.ViewMat)

	lightsUboData.AmbientColor = gglm.NewVec3(20.0/255, 20.0/255, 20.0/255)
	g.applyLightUpdates()
}

func (g *Game) initUbos() {

	globalMatricesUbo = buffers.NewUniformBuffer(
		[]buffers.UniformBufferFieldInput{
			{Id: 0, Type: buffers.DataTypeVec3},
			{Id: 1, Type: buffers.DataTypeMat4},
		},
		buffers.BufUsage_Dynamic_Draw,
	)

	globalMatricesUbo.SetBindPoint(0)
	groundMat.SetUniformBlockBindingPoint("GlobalMatrices", 0)
	whiteMat.SetUniformBlockBindingPoint("GlobalMatrices", 0)
	containerMat.SetUniformBlockBindingPoint("GlobalMatrices", 0)
	palleteMat.SetUniformBlockBindingPoint("GlobalMatrices", 0)

	lightsUbo = buffers.NewUniformBuffer(
		[]buffers.UniformBufferFieldInput{
			// Dir light
			{Id: 0, Type: buffers.DataTypeStruct,
				Subfields: []buffers.UniformBufferFieldInput{
					{Id: 1, Type: buffers.DataTypeVec3}, // 12 00
					{Id: 2, Type: buffers.DataTypeVec3}, // 12 16
					{Id: 3, Type: buffers.DataTypeVec3}, // 12 32
				},
			},
			// Point lights
			{Id: 5, Type: buffers.DataTypeStruct,
				Count: POINT_LIGHT_COUNT,
				Subfields: []buffers.UniformBufferFieldInput{
					{Id: 6, Type: buffers.DataTypeVec3},     // 12 48
					{Id: 7, Type: buffers.DataTypeVec3},     // 12 64
					{Id: 8, Type: buffers.DataTypeVec3},     // 12 80
					{Id: 9, Type: buffers.DataTypeFloat32},  // 04 92
					{Id: 10, Type: buffers.DataTypeFloat32}, // 04 96
					{Id: 11, Type: buffers.DataTypeFloat32}, // 04 100
					{Id: 12, Type: buffers.DataTypeFloat32}, // 04 104
					{Id: 13, Type: buffers.DataTypeFloat32}, // 04 108
				},
			},
			// Spot lights
			{Id: 14, Type: buffers.DataTypeStruct,
				Count: SPOT_LIGHT_COUNT,
				Subfields: []buffers.UniformBufferFieldInput{
					{Id: 15, Type: buffers.DataTypeVec3},    // 12 112
					{Id: 16, Type: buffers.DataTypeVec3},    // 12 128
					{Id: 17, Type: buffers.DataTypeVec3},    // 12 144
					{Id: 18, Type: buffers.DataTypeVec3},    // 12 160
					{Id: 19, Type: buffers.DataTypeFloat32}, // 04 172
					{Id: 20, Type: buffers.DataTypeFloat32}, // 04 176
				},
			},

			// Ambient
			{Id: 21, Type: buffers.DataTypeVec3}, // 12 192
		},
		buffers.BufUsage_Dynamic_Draw,
	)

	// fmt.Printf("\n==Lights UBO (id=%d)==\nSize=%d\nFields: %+v\n\n", lightsUbo.Id, lightsUbo.Size, lightsUbo.Fields)

	lightsUbo.SetBindPoint(1)
	groundMat.SetUniformBlockBindingPoint("Lights", 1)
	whiteMat.SetUniformBlockBindingPoint("Lights", 1)
	containerMat.SetUniformBlockBindingPoint("Lights", 1)
	palleteMat.SetUniformBlockBindingPoint("Lights", 1)
}

func (g *Game) initFbos() {

	// @TODO: Resize window sized fbos on window resize

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
	dirLightDepthMapFbo = buffers.NewFramebuffer(4096, 4096)
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

	// Hdr fbo
	hdrFbo = buffers.NewFramebuffer(uint32(g.WinWidth), uint32(g.WinHeight))
	hdrFbo.NewColorAttachment(
		buffers.FramebufferAttachmentType_Texture,
		buffers.FramebufferAttachmentDataFormat_RGBAF16,
	)

	hdrFbo.NewDepthStencilAttachment(
		buffers.FramebufferAttachmentType_Renderbuffer,
		buffers.FramebufferAttachmentDataFormat_Depth24Stencil8,
	)

	assert.T(hdrFbo.IsComplete(), "Hdr fbo is not complete after init")
}

// applyLightUpdates updates materials and light ubo using
// data from the game's light structs
func (g *Game) applyLightUpdates() {

	// Directional light
	lightsUboData.DirLight = DirLightUboData(dirLight)
	whiteMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id
	containerMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id
	groundMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id
	palleteMat.ShadowMapTex1 = dirLightDepthMapFbo.Attachments[0].Id

	// Point lights
	for i := 0; i < len(pointLights); i++ {

		p := &pointLights[i]
		lightsUboData.PointLights[i] = PointLightUboData(*p)
	}

	whiteMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id
	containerMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id
	groundMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id
	palleteMat.CubemapArrayTex = pointLightDepthMapFbo.Attachments[0].Id

	// Spotlights
	for i := 0; i < len(spotLights); i++ {

		l := &spotLights[i]
		innerCutoffCos := l.InnerCutoffCos()
		outerCutoffCos := l.OuterCutoffCos()

		lightsUboData.SpotLights[i] = SpotLightUboData{
			Pos:           l.Pos,
			Dir:           l.Dir,
			DiffuseColor:  l.DiffuseColor,
			SpecularColor: l.SpecularColor,
			InnerCutoff:   innerCutoffCos,
			OuterCutoff:   outerCutoffCos,
		}
	}

	whiteMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
	containerMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
	groundMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id
	palleteMat.ShadowMapTexArray1 = spotLightDepthMapFbo.Attachments[0].Id

	// Apply changes
	lightsUbo.Bind()
	lightsUbo.SetStruct(lightsUboData)
}

func (g *Game) Update() {

	if input.IsQuitClicked() || input.KeyClicked(sdl.K_ESCAPE) {
		engine.Quit()
	}

	g.updateCameraLookAround()
	g.updateCameraPos()

	globalMatricesUboData.CamPos = cam.Pos
	updateAllProjViewMats(cam.ProjMat, cam.ViewMat)

	g.showDebugWindow()
}

func (g *Game) showDebugWindow() {

	imgui.ShowDemoWindow()

	imgui.Begin("Debug controls")

	imgui.PushStyleColorVec4(imgui.ColText, imgui.NewColor(1, 1, 0, 1).FieldValue)
	imgui.LabelText("FPS", fmt.Sprint(timing.GetAvgFPS()))
	imgui.PopStyleColor()

	if len(frameTimesMs) < FRAME_TIME_MS_SAMPLES {
		frameTimesMs = append(frameTimesMs, timing.DT()*1000)
	} else {
		frameTimesMs[frameTimesMsIndex] = timing.DT() * 1000

		frameTimesMsIndex++
		if frameTimesMsIndex >= len(frameTimesMs) {
			frameTimesMsIndex = 0
		}
	}

	imgui.PlotLinesFloatPtrV("Frame Times", frameTimesMs, int32(len(frameTimesMs)), 0, "", 0, 16, imgui.Vec2{Y: 50}, 4)

	imgui.Spacing()

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

	imgui.Text("HDR")
	imgui.Checkbox("Enable HDR", &hdrRendering)
	if imgui.DragFloatV("Exposure", &hdrExposure, 0.1, -10, 100, "%.3f", imgui.SliderFlagsNone) {
		tonemappedScreenQuadMat.SetUnifFloat32("exposure", hdrExposure)
	}

	imgui.Spacing()

	//
	// Lights
	//
	updateLights := false

	// Ambient light
	imgui.Text("Ambient Light")

	if imgui.ColorEdit3("Ambient Color", &lightsUboData.AmbientColor.Data) {
		updateLights = true
	}

	imgui.Spacing()

	// Directional light
	imgui.Text("Directional Light")

	imgui.Checkbox("Render Directional Light Shadows", &renderDirLightShadows)

	if imgui.DragFloat3("Direction", &dirLight.Dir.Data) {
		updateLights = true
	}

	if imgui.ColorEdit3("Diffuse Color", &dirLight.DiffuseColor.Data) {
		updateLights = true
	}

	if imgui.ColorEdit3("Specular Color", &dirLight.SpecularColor.Data) {
		updateLights = true
	}

	if imgui.DragFloat3("dPos", &dirLightPos.Data) {
		updateLights = true
	}
	if imgui.DragFloat("dSize", &dirLightSize) {
		updateLights = true
	}
	if imgui.DragFloat("dNear", &dirLightNear) {
		updateLights = true
	}
	if imgui.DragFloat("dFar", &dirLightFar) {
		updateLights = true
	}

	imgui.Spacing()

	// Specular
	imgui.Text("Specular Settings")

	if imgui.DragFloat("Specular Shininess", &whiteMat.Shininess) {
		whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		containerMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		groundMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
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

			if imgui.DragFloat3("Pos", &pl.Pos.Data) {
				updateLights = true
			}

			if imgui.ColorEdit3("Diffuse Color", &pl.DiffuseColor.Data) {
				updateLights = true
			}

			if imgui.ColorEdit3("Specular Color", &pl.SpecularColor.Data) {
				updateLights = true
			}

			if imgui.DragFloatV("Falloff", &pl.Falloff, 0.1, 0, 100, "%.3f", imgui.SliderFlagsNone) {
				updateLights = true
			}

			if imgui.DragFloatV("Radius", &pl.Radius, 0.2, 0, 500, "%.3f", imgui.SliderFlagsNone) {
				updateLights = true
				pl.FarPlane = pl.Radius * pointLightRadiusToFarPlaneRatio
			}

			if imgui.DragFloatV("Max Bias", &pl.MaxBias, 0.01, 0, 10, "%.3f", imgui.SliderFlagsNone) {
				updateLights = true
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

			if imgui.DragFloat3("Pos", &l.Pos.Data) {
				updateLights = true
			}

			if imgui.DragFloat3("Dir", &l.Dir.Data) {
				updateLights = true
			}

			if imgui.ColorEdit3("Diffuse Color", &l.DiffuseColor.Data) {
				updateLights = true
			}

			if imgui.ColorEdit3("Specular Color", &l.SpecularColor.Data) {
				updateLights = true
			}

			if imgui.DragFloatRange2V(
				"Cutoff Radians",
				&l.InnerCutoffRad,
				&l.OuterCutoffRad,
				0.1,
				0,
				0,
				"%.3f",
				"%.3f",
				imgui.SliderFlagsNone,
			) {
				updateLights = true
			}

			imgui.DragFloat("Spot Near Plane", &l.NearPlane)
			imgui.DragFloat("Spot Far Plane", &l.FarPlane)

			imgui.TreePop()
		}

		imgui.EndListBox()
	}

	if updateLights {
		g.applyLightUpdates()
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

	const MAX_MOUSE_MOVE = 300
	mouseX = gglm.Clamp(mouseX, -MAX_MOUSE_MOVE, MAX_MOUSE_MOVE)
	mouseY = gglm.Clamp(mouseY, -MAX_MOUSE_MOVE, MAX_MOUSE_MOVE)

	// Yaw
	yaw += float32(mouseX) * camRotSpeed * timing.DT()

	// Pitch
	pitch += float32(-mouseY) * camRotSpeed * timing.DT()
	if pitch > 1.5 {
		pitch = 1.5
	}

	if pitch < -1.5 {
		pitch = -1.5
	}

	cam.UpdateRotation(pitch, yaw)
}

func (g *Game) updateCameraPos() {

	update := false

	var camSpeedScale float32 = 1.0
	if input.KeyDown(sdl.K_LSHIFT) {
		camSpeedScale = 2
	}

	// Forward and backward
	if input.KeyDown(sdl.K_w) {
		cam.Pos.Add(cam.Forward.Clone().Scale(camMoveSpeed * camSpeedScale * timing.DT()))
		update = true
	} else if input.KeyDown(sdl.K_s) {
		cam.Pos.Add(cam.Forward.Clone().Scale(-camMoveSpeed * camSpeedScale * timing.DT()))
		update = true
	}

	// Left and right
	if input.KeyDown(sdl.K_d) {
		cross := gglm.Cross(&cam.Forward, &cam.WorldUp)
		cam.Pos.Add(cross.Normalize().Scale(camMoveSpeed * camSpeedScale * timing.DT()))
		update = true
	} else if input.KeyDown(sdl.K_a) {
		cross := gglm.Cross(&cam.Forward, &cam.WorldUp)
		cam.Pos.Add(cross.Normalize().Scale(-camMoveSpeed * camSpeedScale * timing.DT()))
		update = true
	}

	if update {
		cam.Update()
	}
}

var (
	rotatingCubeSpeedDeg1 float32 = 45
	rotatingCubeSpeedDeg2 float32 = 120
	rotatingCubeSpeedDeg3 float32 = 120
	rotatingCubeTrMat1            = gglm.NewTrMatWithPos(-4, -1, 4)
	rotatingCubeTrMat2            = gglm.NewTrMatWithPos(-1, 0.5, 4)
	rotatingCubeTrMat3            = gglm.NewTrMatWithPos(5, 0.5, 4)
)

func (g *Game) Render() {

	globalMatricesUbo.Bind()
	globalMatricesUbo.SetStruct(globalMatricesUboData)

	rotatingCubeTrMat1.Rotate(rotatingCubeSpeedDeg1*gglm.Deg2Rad*timing.DT(), 0, 1, 0)
	rotatingCubeTrMat2.Rotate(rotatingCubeSpeedDeg2*gglm.Deg2Rad*timing.DT(), 1, 1, 0)
	rotatingCubeTrMat3.Rotate(rotatingCubeSpeedDeg3*gglm.Deg2Rad*timing.DT(), 1, 1, 1)

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
			g.RenderScene(&debugDepthMat)
		} else if hdrRendering {
			g.renderHdrFbo()
		} else {

			g.RenderScene(nil)
			if renderSkybox {
				g.DrawSkybox()
			}
		}
	}

	if renderToDemoFbo {
		g.renderDemoFbo()
	}
}

func (g *Game) renderDirectionalLightShadowmap() {

	// Set some uniforms
	dirLightProjViewMat := dirLight.GetProjViewMat()

	whiteMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)
	containerMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)
	groundMat.SetUnifMat4("dirLightProjViewMat", &dirLightProjViewMat)
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
	g.RenderScene(&depthMapMat)
	gl.CullFace(gl.BACK)

	dirLightDepthMapFbo.UnBindWithViewport(uint32(g.WinWidth), uint32(g.WinHeight))

	if showDirLightDepthMapFbo {
		screenQuadMat.DiffuseTex = dirLightDepthMapFbo.Attachments[0].Id
		screenQuadMat.SetUnifVec2("offset", &dirLightDepthMapFboOffset)
		screenQuadMat.SetUnifVec2("scale", &dirLightDepthMapFboScale)
		screenQuadMat.Bind()
		g.Rend.DrawVertexArray(&screenQuadMat, &screenQuadVao, 0, 6)
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
		groundMat.SetUnifMat4(projViewMatIndexStr, &projViewMat)
		palleteMat.SetUnifMat4(projViewMatIndexStr, &projViewMat)

		// Set depth uniforms
		arrayDepthMapMat.SetUnifMat4("projViewMats["+indexStr+"]", &projViewMat)
	}

	// Render
	spotLightDepthMapFbo.BindWithViewport()
	spotLightDepthMapFbo.Clear()

	// Front culling created issues
	// gl.CullFace(gl.FRONT)
	g.RenderScene(&arrayDepthMapMat)
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

		g.RenderScene(&omnidirDepthMapMat)
	}

	pointLightDepthMapFbo.UnBindWithViewport(uint32(g.WinWidth), uint32(g.WinHeight))
}

func (g *Game) renderDemoFbo() {

	demoFbo.Bind()
	demoFbo.Clear()

	if renderDepthBuffer {
		g.RenderScene(&debugDepthMat)
	} else {
		g.RenderScene(nil)
	}

	if renderSkybox {
		g.DrawSkybox()
	}

	demoFbo.UnBind()

	screenQuadMat.DiffuseTex = demoFbo.Attachments[0].Id
	screenQuadMat.SetUnifVec2("offset", &demoFboOffset)
	screenQuadMat.SetUnifVec2("scale", &demoFboScale)

	g.Rend.DrawVertexArray(&screenQuadMat, &screenQuadVao, 0, 6)
}

func (g *Game) renderHdrFbo() {

	hdrFbo.Bind()
	hdrFbo.Clear()

	g.RenderScene(nil)

	if renderSkybox {
		g.DrawSkybox()
	}

	hdrFbo.UnBind()

	tonemappedScreenQuadMat.DiffuseTex = hdrFbo.Attachments[0].Id
	g.Rend.DrawVertexArray(&tonemappedScreenQuadMat, &screenQuadVao, 0, 6)
}

func (g *Game) RenderScene(overrideMat *materials.Material) {

	tempModelMatrix := *cubeModelMat.Clone()

	// See if we need overrides
	sunMat := palleteMat
	chairMat := palleteMat
	cubeMat := containerMat
	groundMat := groundMat

	if overrideMat != nil {
		sunMat = *overrideMat
		chairMat = *overrideMat
		cubeMat = *overrideMat
		groundMat = *overrideMat
	}

	// Draw dir light
	dirLightTrMat := gglm.NewTrMatId()
	g.Rend.DrawMesh(&sphereMesh, dirLightTrMat.Translate(0, 10, 0).Scale(0.1, 0.1, 0.1), &sunMat)

	// Draw point lights
	for i := 0; i < len(pointLights); i++ {

		pl := &pointLights[i]
		plTrMat := gglm.NewTrMatId()
		g.Rend.DrawMesh(&cubeMesh, plTrMat.TranslateVec(&pl.Pos).Scale(0.1, 0.1, 0.1), &sunMat)
	}

	// Chair
	g.Rend.DrawMesh(&chairMesh, &tempModelMatrix, &chairMat)

	// Ground
	groundTrMat := gglm.NewTrMatId()
	g.Rend.DrawMesh(&cubeMesh, groundTrMat.Translate(0, -3, 0).Scale(20, 1, 20), &groundMat)

	// Cubes
	tempModelMatrix.Translate(-6, 0, 0)
	g.Rend.DrawMesh(&cubeMesh, &tempModelMatrix, &cubeMat)

	tempModelMatrix.Translate(0, -1, -4)
	g.Rend.DrawMesh(&cubeMesh, &tempModelMatrix, &cubeMat)

	// Rotating cubes
	g.Rend.DrawMesh(&cubeMesh, &rotatingCubeTrMat1, &cubeMat)
	g.Rend.DrawMesh(&cubeMesh, &rotatingCubeTrMat2, &cubeMat)
	g.Rend.DrawMesh(&cubeMesh, &rotatingCubeTrMat3, &cubeMat)

	// Cubes generator
	// rowSize := 1
	// for y := 0; y < rowSize; y++ {
	// 	for x := 0; x < rowSize; x++ {
	// 		tempModelMatrix.Translate(gglm.NewVec3(-6, 0, 0))
	// 		g.Rend.DrawMesh(cubeMesh, tempModelMatrix, cubeMat)
	// 	}
	// 	tempModelMatrix.Translate(gglm.NewVec3(float32(rowSize), -1, 0))
	// }
}

func (g *Game) DrawSkybox() {

	gl.Disable(gl.CULL_FACE)
	gl.DepthFunc(gl.LEQUAL)

	g.Rend.DrawCubemap(&skyboxMesh, &skyboxMat)

	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.CULL_FACE)
}

func (g *Game) FrameEnd() {
}

func (g *Game) DeInit() {
	g.Win.Destroy()
}

func updateAllProjViewMats(projMat, viewMat gglm.Mat4) {

	projViewMat := *projMat.Clone().Mul(&viewMat)
	globalMatricesUboData.ProjViewMat = projViewMat

	unlitMat.SetUnifMat4("projViewMat", &projViewMat)
	debugDepthMat.SetUnifMat4("projViewMat", &projViewMat)

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
