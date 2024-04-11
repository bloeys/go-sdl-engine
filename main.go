package main

import (
	"fmt"
	"runtime"

	imgui "github.com/AllenDang/cimgui-go"
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assets"
	"github.com/bloeys/nmage/camera"
	"github.com/bloeys/nmage/engine"
	"github.com/bloeys/nmage/entity"
	"github.com/bloeys/nmage/input"
	"github.com/bloeys/nmage/logging"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/meshes"
	"github.com/bloeys/nmage/registry"
	"github.com/bloeys/nmage/renderer/rend3dgl"
	"github.com/bloeys/nmage/timing"
	nmageimgui "github.com/bloeys/nmage/ui/imgui"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

/*
@TODO:
	- Rendering:
		- Phong lighting model
		- Point lights
		- Spotlights
		- HDR
		- Cascaded shadow mapping
		- Skeletal animations
	- Create VAO struct independent from VBO to support multi-VBO use cases (e.g. instancing)
	- Renderer batching
	- Scene graph
	- Separate engine loop from rendering loop? or leave it to the user?
	- Abstract keys enum away from sdl
	- Proper Asset loading
	- Frustum culling
	- Material system editor with fields automatically extracted from the shader
*/

const (
	camSpeed         = 15
	mouseSensitivity = 0.5

	unscaledWindowWidth  = 1280
	unscaledWindowHeight = 720
)

var (
	window *engine.Window

	pitch float32 = 0
	yaw   float32 = 0
	cam   *camera.Camera

	whiteMat      *materials.Material
	containerMat  *materials.Material
	palleteMat    *materials.Material
	skyboxMat     *materials.Material
	debugDepthMat *materials.Material

	cubeMesh   *meshes.Mesh
	sphereMesh *meshes.Mesh
	chairMesh  *meshes.Mesh
	skyboxMesh *meshes.Mesh

	cubeModelMat = gglm.NewTrMatId()

	debugDrawDepthBuffer bool

	skyboxCmap assets.Cubemap

	dpiScaling float32

	// Light settings
	ambientColor = gglm.NewVec3(0.2, 0.2, 0.2)

	// Lights
	lightPos1   = gglm.NewVec3(-2, 10, 0)
	lightColor1 = gglm.NewVec3(1, 1, 1)
)

type OurGame struct {
	Win       *engine.Window
	ImGUIInfo nmageimgui.ImguiInfo
}

type TransformComp struct {
	entity.BaseComp

	Pos   *gglm.Vec3
	Rot   *gglm.Quat
	Scale *gglm.Vec3
}

func (t *TransformComp) Name() string {
	return "Transform Component"
}

func Test() {

	// lvl := level.NewLevel("test level")
	testRegistry := registry.NewRegistry[int](100)

	e1, e1Handle := testRegistry.New()
	e1CompContainer := entity.NewCompContainer()
	fmt.Printf("Entity 1: %+v; Handle: %+v; Index: %+v; Gen: %+v; Flags: %+v\n", e1, e1Handle, e1Handle.Index(), e1Handle.Generation(), e1Handle.Flags())

	trComp := entity.GetComp[*TransformComp](&e1CompContainer)
	fmt.Println("Get comp before adding any:", trComp)

	entity.AddComp(e1Handle, &e1CompContainer, &TransformComp{
		Pos:   gglm.NewVec3(0, 0, 0),
		Rot:   gglm.NewQuatEulerXYZ(0, 0, 0),
		Scale: gglm.NewVec3(0, 0, 0),
	})
	trComp = entity.GetComp[*TransformComp](&e1CompContainer)
	fmt.Println("Get transform comp:", trComp)

	e2, e2Handle := testRegistry.New()
	e3, e3Handle := testRegistry.New()
	e4, e4Handle := testRegistry.New()
	fmt.Printf("Entity 2: %+v; Handle: %+v; Index: %+v; Gen: %+v; Flags: %+v\n", e2, e2Handle, e2Handle.Index(), e2Handle.Generation(), e2Handle.Flags())
	fmt.Printf("Entity 3: %+v; Handle: %+v; Index: %+v; Gen: %+v; Flags: %+v\n", e3, e3Handle, e3Handle.Index(), e3Handle.Generation(), e3Handle.Flags())
	fmt.Printf("Entity 4: %+v; Handle: %+v; Index: %+v; Gen: %+v; Flags: %+v\n", e4, e4Handle, e4Handle.Index(), e4Handle.Generation(), e4Handle.Flags())

	*e2 = 1000
	fmt.Printf("Entity 2 value after registry get: %+v\n", *testRegistry.Get(e2Handle))

	testRegistry.Free(e2Handle)
	fmt.Printf("Entity 2 value after free: %+v\n", testRegistry.Get(e2Handle))

	e5, e5Handle := testRegistry.New()
	fmt.Printf("Entity 5: %+v; Handle: %+v; Index: %+v; Gen: %+v; Flags: %+v\n", e5, e5Handle, e5Handle.Index(), e5Handle.Generation(), e5Handle.Flags())
}

func main() {

	// Test()
	// return

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

	game := &OurGame{
		Win:       window,
		ImGUIInfo: nmageimgui.NewImGui("./res/shaders/imgui.glsl"),
	}
	window.EventCallbacks = append(window.EventCallbacks, game.handleWindowEvents)

	engine.Run(game, window, game.ImGUIInfo)
}

func (g *OurGame) handleWindowEvents(e sdl.Event) {

	switch e := e.(type) {
	case *sdl.WindowEvent:
		if e.Event == sdl.WINDOWEVENT_SIZE_CHANGED {

			width := e.Data1
			height := e.Data2
			cam.AspectRatio = float32(width) / float32(height)
			cam.Update()

			palleteMat.SetUnifMat4("projMat", &cam.ProjMat)
			debugDepthMat.SetUnifMat4("projMat", &cam.ProjMat)
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

func (g *OurGame) Init() {

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
	whiteTex, err := assets.LoadTexturePNG("./res/textures/white.png", &assets.TextureLoadOptions{TextureIsSrgba: true})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	containerDiffuseTex, err := assets.LoadTexturePNG("./res/textures/container-diffuse.png", &assets.TextureLoadOptions{TextureIsSrgba: true})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	containerSpecularTex, err := assets.LoadTexturePNG("./res/textures/container-specular.png", &assets.TextureLoadOptions{TextureIsSrgba: true})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	palleteTex, err := assets.LoadTexturePNG("./res/textures/pallete-endesga-64-1x.png", &assets.TextureLoadOptions{TextureIsSrgba: true})
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load texture. Err: ", err)
	}

	skyboxCmap, err = assets.LoadCubemapTextures(
		"./res/textures/sb-right.jpg", "./res/textures/sb-left.jpg",
		"./res/textures/sb-top.jpg", "./res/textures/sb-bottom.jpg",
		"./res/textures/sb-front.jpg", "./res/textures/sb-back.jpg",
		&assets.TextureLoadOptions{TextureIsSrgba: true},
	)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to load cubemap. Err: ", err)
	}

	// Create materials
	whiteMat = materials.NewMaterial("White mat", "./res/shaders/simple.glsl")
	whiteMat.Shininess = 128
	whiteMat.DiffuseTex = whiteTex.TexID
	whiteMat.SetUnifMat4("projMat", &cam.ProjMat)
	whiteMat.SetUnifVec3("ambientColor", ambientColor)
	whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
	whiteMat.SetUnifVec3("lightPos1", lightPos1)
	whiteMat.SetUnifVec3("lightColor1", lightColor1)

	containerMat = materials.NewMaterial("Container mat", "./res/shaders/simple.glsl")
	containerMat.Shininess = 128
	containerMat.DiffuseTex = containerDiffuseTex.TexID
	containerMat.SpecularTex = containerSpecularTex.TexID
	containerMat.SetUnifMat4("projMat", &cam.ProjMat)
	containerMat.SetUnifVec3("ambientColor", ambientColor)
	containerMat.SetUnifFloat32("material.shininess", containerMat.Shininess)
	containerMat.SetUnifVec3("lightPos1", lightPos1)
	containerMat.SetUnifVec3("lightColor1", lightColor1)

	palleteMat = materials.NewMaterial("Pallete mat", "./res/shaders/simple.glsl")
	palleteMat.Shininess = 128
	palleteMat.DiffuseTex = palleteTex.TexID
	palleteMat.SetUnifMat4("projMat", &cam.ProjMat)
	palleteMat.SetUnifVec3("ambientColor", ambientColor)
	palleteMat.SetUnifFloat32("material.shininess", palleteMat.Shininess)
	palleteMat.SetUnifVec3("lightPos1", lightPos1)
	palleteMat.SetUnifVec3("lightColor1", lightColor1)

	debugDepthMat = materials.NewMaterial("Debug depth mat", "./res/shaders/debug-depth.glsl")
	debugDepthMat.SetUnifMat4("projMat", &cam.ProjMat)

	skyboxMat = materials.NewMaterial("Skybox mat", "./res/shaders/skybox.glsl")

	// Movement, scale and rotation
	translationMat := gglm.NewTranslationMat(gglm.NewVec3(0, 0, 0))
	scaleMat := gglm.NewScaleMat(gglm.NewVec3(1, 1, 1))
	rotMat := gglm.NewRotMat(gglm.NewQuatEuler(gglm.NewVec3(-90, -90, 0).AsRad()))
	cubeModelMat.Mul(translationMat.Mul(rotMat.Mul(scaleMat)))

	updateViewMat()
}

func (g *OurGame) Update() {

	if input.IsQuitClicked() || input.KeyClicked(sdl.K_ESCAPE) {
		engine.Quit()
	}

	g.updateCameraLookAround()
	g.updateCameraPos()

	//Rotating cubes
	if input.KeyDown(sdl.K_SPACE) {
		cubeModelMat.Rotate(10*timing.DT()*gglm.Deg2Rad, gglm.NewVec3(1, 1, 1).Normalize())
	}

	g.showDebugWindow()

	if input.KeyClicked(sdl.K_F4) {
		fmt.Printf("Pos: %s; Forward: %s; |Forward|: %f\n", cam.Pos.String(), cam.Forward.String(), cam.Forward.Mag())
	}

	g.Win.SDLWin.SetTitle(fmt.Sprint("nMage (", timing.GetAvgFPS(), " fps)"))
}

func (g *OurGame) showDebugWindow() {

	imgui.ShowDemoWindow()

	imgui.Begin("Debug controls")

	// Camera
	if imgui.DragFloat3("Cam Pos", &cam.Pos.Data) {
		updateViewMat()
	}
	if imgui.DragFloat3("Cam Forward", &cam.Forward.Data) {
		updateViewMat()
	}

	imgui.Spacing()

	// Light settings
	if imgui.DragFloat3("Ambient Color", &ambientColor.Data) {
		whiteMat.SetUnifVec3("ambientColor", ambientColor)
		containerMat.SetUnifVec3("ambientColor", ambientColor)
		palleteMat.SetUnifVec3("ambientColor", ambientColor)
	}

	imgui.Spacing()

	if imgui.DragFloat("Specular Shininess", &whiteMat.Shininess) {
		whiteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		containerMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
		palleteMat.SetUnifFloat32("material.shininess", whiteMat.Shininess)
	}

	imgui.Spacing()

	// Lights
	if imgui.DragFloat3("Light Pos 1", &lightPos1.Data) {
		whiteMat.SetUnifVec3("lightPos1", lightPos1)
		containerMat.SetUnifVec3("lightPos1", lightPos1)
		palleteMat.SetUnifVec3("lightPos1", lightPos1)
	}

	if imgui.DragFloat3("Light Color 1", &lightColor1.Data) {
		whiteMat.SetUnifVec3("lightColor1", lightColor1)
		containerMat.SetUnifVec3("lightColor1", lightColor1)
		palleteMat.SetUnifVec3("lightColor1", lightColor1)
	}

	imgui.Spacing()

	// Other
	imgui.Checkbox("Debug depth buffer", &debugDrawDepthBuffer)

	imgui.End()
}

func (g *OurGame) updateCameraLookAround() {

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

	updateViewMat()
}

func (g *OurGame) updateCameraPos() {

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
		updateViewMat()
	}
}

func (g *OurGame) Render() {

	tempModelMatrix := cubeModelMat.Clone()

	whiteMat.SetUnifVec3("camPos", &cam.Pos)
	containerMat.SetUnifVec3("camPos", &cam.Pos)
	palleteMat.SetUnifVec3("camPos", &cam.Pos)

	sunMat := palleteMat
	chairMat := palleteMat
	cubeMat := containerMat
	if debugDrawDepthBuffer {
		sunMat = debugDepthMat
		chairMat = debugDepthMat
		cubeMat = debugDepthMat
	}

	// Draw sun
	window.Rend.Draw(sphereMesh, gglm.NewTrMatId().Translate(lightPos1).Scale(gglm.NewVec3(0.1, 0.1, 0.1)), sunMat)

	// Chair
	window.Rend.Draw(chairMesh, tempModelMatrix, chairMat)

	// Cubes
	rowSize := 1
	for y := 0; y < rowSize; y++ {
		for x := 0; x < rowSize; x++ {
			tempModelMatrix.Translate(gglm.NewVec3(-6, 0, 0))
			window.Rend.Draw(cubeMesh, tempModelMatrix, cubeMat)
		}
		tempModelMatrix.Translate(gglm.NewVec3(float32(rowSize), -1, 0))
	}

	g.DrawSkybox()
}

func (g *OurGame) DrawSkybox() {

	gl.Disable(gl.CULL_FACE)
	gl.DepthFunc(gl.LEQUAL)
	skyboxMesh.Buf.Bind()
	skyboxMat.Bind()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, skyboxCmap.TexID)

	viewMat := cam.ViewMat.Clone()
	viewMat.Set(0, 3, 0)
	viewMat.Set(1, 3, 0)
	viewMat.Set(2, 3, 0)
	viewMat.Set(3, 0, 0)
	viewMat.Set(3, 1, 0)
	viewMat.Set(3, 2, 0)
	viewMat.Set(3, 3, 0)

	skyboxMat.SetUnifMat4("viewMat", viewMat)
	skyboxMat.SetUnifMat4("projMat", &cam.ProjMat)
	for i := 0; i < len(skyboxMesh.SubMeshes); i++ {
		gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, skyboxMesh.SubMeshes[i].IndexCount, gl.UNSIGNED_INT, uintptr(skyboxMesh.SubMeshes[i].BaseIndex), skyboxMesh.SubMeshes[i].BaseVertex)
	}

	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.CULL_FACE)
}

func (g *OurGame) FrameEnd() {
}

func (g *OurGame) DeInit() {
	g.Win.Destroy()
}

func updateViewMat() {
	cam.Update()
	whiteMat.SetUnifMat4("viewMat", &cam.ViewMat)
	containerMat.SetUnifMat4("viewMat", &cam.ViewMat)
	palleteMat.SetUnifMat4("viewMat", &cam.ViewMat)
	debugDepthMat.SetUnifMat4("viewMat", &cam.ViewMat)
}
