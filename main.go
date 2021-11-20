package main

import (
	"runtime"

	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/go-sdl-engine/buffers"
	"github.com/bloeys/go-sdl-engine/input"
	"github.com/bloeys/go-sdl-engine/logging"
	"github.com/bloeys/go-sdl-engine/res/models"
	"github.com/bloeys/go-sdl-engine/shaders"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

//TODO:
//Abstract UI
//Asset loading
//Rework buffers package
//Interleaved or packed buffers (xyzxyzxyz OR xxxyyyzzz)
//Timing and deltatime

type ImguiInfo struct {
	imCtx *imgui.Context

	vaoID      uint32
	vboID      uint32
	indexBufID uint32
	texID      uint32
}

var (
	winWidth  int32 = 1280
	winHeight int32 = 720

	isRunning bool = true
	window    *sdl.Window

	simpleShader shaders.ShaderProgram
	imShader     shaders.ShaderProgram
	bo           *buffers.BufferObject

	modelMat = gglm.NewTrMatId()
	projMat  = &gglm.Mat4{}

	imguiInfo *ImguiInfo
)

func main() {

	runtime.LockOSThread()

	err := initSDL()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to init SDL. Err:", err)
	}

	window, err = sdl.CreateWindow("Go SDL Engine", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, winWidth, winHeight, sdl.WINDOW_OPENGL|sdl.WINDOW_RESIZABLE)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create window. Err: ", err)
	}
	defer window.Destroy()

	glCtx, err := window.GLCreateContext()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create OpenGL context. Err: ", err)
	}
	defer sdl.GLDeleteContext(glCtx)

	err = initOpenGL()
	if err != nil {
		logging.ErrLog.Fatalln(err)
	}

	loadShaders()
	loadBuffers()
	initImGUI()

	simpleShader.SetAttribute("vertPos", bo, bo.VertPosBuf)
	simpleShader.EnableAttribute("vertPos")

	// simpleShader.SetAttribute("vertColor", bo, bo.ColorBuf)
	// simpleShader.EnableAttribute("vertColor")

	//Movement, scale and rotation
	translationMat := gglm.NewTranslationMat(gglm.NewVec3(0, 0, 0))
	scaleMat := gglm.NewScaleMat(gglm.NewVec3(0.25, 0.25, 0.25))
	rotMat := gglm.NewRotMat(gglm.NewQuatEuler(gglm.NewVec3(0, 0, 0).AsRad()))

	modelMat.Mul(translationMat.Mul(rotMat.Mul(scaleMat)))
	simpleShader.SetUnifMat4("modelMat", &modelMat.Mat4)

	//Moves objects into the cameras view
	camPos := gglm.NewVec3(0, 0, 10)
	targetPos := gglm.NewVec3(0, 0, 0)
	viewMat := gglm.LookAt(camPos, targetPos, gglm.NewVec3(0, 1, 0))
	simpleShader.SetUnifMat4("viewMat", &viewMat.Mat4)

	//Perspective/Depth
	projMat := gglm.Perspective(45*gglm.Deg2Rad, float32(winWidth)/float32(winHeight), 0.1, 20)
	simpleShader.SetUnifMat4("projMat", projMat)

	//Game loop
	for isRunning {

		handleInputs()
		runGameLogic()
		draw()

		sdl.Delay(17)
	}
}

func initSDL() error {

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return err
	}

	sdl.GLSetAttribute(sdl.MAJOR_VERSION, 4)
	sdl.GLSetAttribute(sdl.MINOR_VERSION, 1)

	// R(0-255) G(0-255) B(0-255)
	sdl.GLSetAttribute(sdl.GL_RED_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_GREEN_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_BLUE_SIZE, 8)

	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)
	sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, 8)

	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)

	return nil
}

func initOpenGL() error {

	if err := gl.Init(); err != nil {
		return err
	}

	gl.ClearColor(0, 0, 0, 1)
	return nil
}

func loadShaders() {

	var err error
	simpleShader, err = shaders.NewShaderProgram()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader program. Err: ", err)
	}

	vertShader, err := shaders.LoadAndCompilerShader("./res/shaders/simple.vert.glsl", shaders.VertexShaderType)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader. Err: ", err)
	}

	fragShader, err := shaders.LoadAndCompilerShader("./res/shaders/simple.frag.glsl", shaders.FragmentShaderType)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader. Err: ", err)
	}

	simpleShader.AttachShader(vertShader)
	simpleShader.AttachShader(fragShader)
	simpleShader.Link()

	//ImGUI shader
	imShader, err = shaders.NewShaderProgram()
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader program. Err: ", err)
	}

	imguiVertShader, err := shaders.LoadAndCompilerShader("./res/shaders/imgui.vert.glsl", shaders.VertexShaderType)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader. Err: ", err)
	}

	imguiFragShader, err := shaders.LoadAndCompilerShader("./res/shaders/imgui.frag.glsl", shaders.FragmentShaderType)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new shader. Err: ", err)
	}

	imShader.AttachShader(imguiVertShader)
	imShader.AttachShader(imguiFragShader)
	imShader.Link()
}

func loadBuffers() {

	vertices := []float32{
		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
		-0.5, -0.5, 0,
	}
	// colors := []float32{
	// 	1, 0, 0,
	// 	0, 0, 1,
	// 	0, 0, 1,
	// 	0, 0, 1,
	// }
	indices := []uint32{0, 1, 3, 1, 2, 3}

	//Load obj
	objInfo, err := models.LoadObj("./res/models/obj.obj")
	if err != nil {
		panic(err)
	}
	// logging.InfoLog.Printf("%v", objInfo.TriIndices)

	vertices = objInfo.VertPos
	indices = objInfo.TriIndices

	bo = buffers.NewBufferObject()
	bo.GenBuffer(vertices, buffers.BufUsageStatic, buffers.BufTypeVertPos, buffers.DataTypeVec3)
	// bo.GenBuffer(colors, buffers.BufUsageStatic, buffers.BufTypeColor, buffers.DataTypeVec3)
	bo.GenBufferUint32(indices, buffers.BufUsageStatic, buffers.BufTypeIndex, buffers.DataTypeUint32)
}

func initImGUI() {

	imguiInfo = &ImguiInfo{
		imCtx: imgui.CreateContext(nil),
	}

	imIO := imgui.CurrentIO()
	imIO.SetBackendFlags(imIO.GetBackendFlags() | imgui.BackendFlagsRendererHasVtxOffset)

	gl.GenVertexArrays(1, &imguiInfo.vaoID)
	gl.GenBuffers(1, &imguiInfo.vboID)
	gl.GenBuffers(1, &imguiInfo.indexBufID)
	gl.GenTextures(1, &imguiInfo.texID)

	// Upload font to gpu
	gl.BindTexture(gl.TEXTURE_2D, imguiInfo.texID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	image := imIO.Fonts().TextureDataAlpha8()
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height), 0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	// Store our identifier
	imIO.Fonts().SetTextureID(imgui.TextureID(imguiInfo.texID))

	//Shader attributes
	imShader.Activate()
	imShader.EnableAttribute("Position")
	imShader.EnableAttribute("UV")
	imShader.EnableAttribute("Color")
	imShader.Deactivate()

	//Init imgui input mapping
	keys := map[int]int{
		imgui.KeyTab:        sdl.SCANCODE_TAB,
		imgui.KeyLeftArrow:  sdl.SCANCODE_LEFT,
		imgui.KeyRightArrow: sdl.SCANCODE_RIGHT,
		imgui.KeyUpArrow:    sdl.SCANCODE_UP,
		imgui.KeyDownArrow:  sdl.SCANCODE_DOWN,
		imgui.KeyPageUp:     sdl.SCANCODE_PAGEUP,
		imgui.KeyPageDown:   sdl.SCANCODE_PAGEDOWN,
		imgui.KeyHome:       sdl.SCANCODE_HOME,
		imgui.KeyEnd:        sdl.SCANCODE_END,
		imgui.KeyInsert:     sdl.SCANCODE_INSERT,
		imgui.KeyDelete:     sdl.SCANCODE_DELETE,
		imgui.KeyBackspace:  sdl.SCANCODE_BACKSPACE,
		imgui.KeySpace:      sdl.SCANCODE_BACKSPACE,
		imgui.KeyEnter:      sdl.SCANCODE_RETURN,
		imgui.KeyEscape:     sdl.SCANCODE_ESCAPE,
		imgui.KeyA:          sdl.SCANCODE_A,
		imgui.KeyC:          sdl.SCANCODE_C,
		imgui.KeyV:          sdl.SCANCODE_V,
		imgui.KeyX:          sdl.SCANCODE_X,
		imgui.KeyY:          sdl.SCANCODE_Y,
		imgui.KeyZ:          sdl.SCANCODE_Z,
	}

	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	for imguiKey, nativeKey := range keys {
		imIO.KeyMap(imguiKey, nativeKey)
	}
}

func handleInputs() {

	imIO := imgui.CurrentIO()

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

		switch e := event.(type) {

		case *sdl.MouseWheelEvent:

			var deltaX, deltaY float32
			if e.X > 0 {
				deltaX++
			} else if e.X < 0 {
				deltaX--
			}

			if e.Y > 0 {
				deltaY++
			} else if e.Y < 0 {
				deltaY--
			}

			imIO.AddMouseWheelDelta(deltaX, deltaY)

		case *sdl.KeyboardEvent:
			input.HandleKeyboardEvent(e)

			if e.Type == sdl.KEYDOWN {
				imIO.KeyPress(int(e.Keysym.Scancode))
			} else if e.Type == sdl.KEYUP {
				imIO.KeyRelease(int(e.Keysym.Scancode))
			}

		case *sdl.TextInputEvent:
			imIO.AddInputCharacters(string(e.Text[:]))

		case *sdl.MouseButtonEvent:
			input.HandleMouseEvent(e)

		case *sdl.WindowEvent:

			//NOTE: SDL is not firing window resize, but is resizing the window by itself
			// if e.Type != sdl.WINDOWEVENT_SIZE_CHANGED {
			// 	continue
			// }

			// winWidth = e.Data1
			// winHeight = e.Data2
			// window.SetSize(int32(winWidth), int32(winHeight))

			// projMat = gglm.Perspective(45*gglm.Deg2Rad, float32(winWidth)/float32(winHeight), 0.1, 20)
			// simpleShader.SetUnifMat4("projMat", projMat)

		case *sdl.QuitEvent:
			isRunning = false
		}
	}

	currWinWidth, currWinHeight := window.GetSize()
	if winWidth != currWinWidth || winHeight != currWinHeight {
		handleWindowResize(currWinWidth, currWinHeight)
	}

	// If a mouse press event came, always pass it as "mouse held this frame", so we don't miss click-release events that are shorter than 1 frame.
	x, y, _ := sdl.GetMouseState()
	imIO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})

	imIO.SetMouseButtonDown(0, input.MouseDown(sdl.BUTTON_LEFT))
	imIO.SetMouseButtonDown(1, input.MouseDown(sdl.BUTTON_RIGHT))
	imIO.SetMouseButtonDown(2, input.MouseDown(sdl.BUTTON_MIDDLE))

	imIO.KeyShift(sdl.SCANCODE_LSHIFT, sdl.SCANCODE_RSHIFT)
	imIO.KeyCtrl(sdl.SCANCODE_LCTRL, sdl.SCANCODE_RCTRL)
	imIO.KeyAlt(sdl.SCANCODE_LALT, sdl.SCANCODE_RALT)
}

func handleWindowResize(newWinWidth, newWinHeight int32) {

	winWidth = newWinWidth
	winHeight = newWinHeight

	fbWidth, fbHeight := window.GLGetDrawableSize()
	if fbWidth <= 0 || fbHeight <= 0 {
		return
	}
	gl.Viewport(0, 0, fbWidth, fbHeight)

	projMat = gglm.Perspective(45*gglm.Deg2Rad, float32(winWidth)/float32(winHeight), 0.1, 20)
	simpleShader.SetUnifMat4("projMat", projMat)
}

var time uint64 = 0
var name string = ""

var ambientColor gglm.Vec3
var ambientColorStrength float32 = 1

func runGameLogic() {

	if input.KeyDown(sdl.K_w) {
		modelMat.Translate(gglm.NewVec3(0, 0, -0.1))
	}
	if input.KeyDown(sdl.K_s) {
		modelMat.Translate(gglm.NewVec3(0, 0, 0.1))
	}
	if input.KeyDown(sdl.K_d) {
		modelMat.Translate(gglm.NewVec3(0.1, 0, 0))
	}
	if input.KeyDown(sdl.K_a) {
		modelMat.Translate(gglm.NewVec3(-0.1, 0, 00))
	}

	simpleShader.SetUnifMat4("modelMat", &modelMat.Mat4)

	//ImGUI
	imIO := imgui.CurrentIO()
	imIO.SetDisplaySize(imgui.Vec2{X: float32(winWidth), Y: float32(winHeight)})

	// Setup time step (we don't use SDL_GetTicks() because it is using millisecond resolution)
	frequency := sdl.GetPerformanceFrequency()
	currentTime := sdl.GetPerformanceCounter()
	if time > 0 {
		imIO.SetDeltaTime(float32(currentTime-time) / float32(frequency))
	} else {
		imIO.SetDeltaTime(1.0 / 60.0)
	}
	time = currentTime

	imgui.NewFrame()
	if imgui.Button("Click Me!") {
		logging.InfoLog.Println("Clicked!")
	}
	imgui.InputText("Name", &name)

	if imgui.SliderFloat3("Ambient Color", &ambientColor.Data, 0, 1) {
		simpleShader.SetUnifVec3("ambientLightColor", &ambientColor)
	}

	if imgui.SliderFloat("Ambient Color Strength", &ambientColorStrength, 0, 1) {
		simpleShader.SetUnifFloat32("ambientStrength", ambientColorStrength)
	}

	imgui.Render()
}

func draw() {

	gl.Disable(gl.SCISSOR_TEST)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	simpleShader.Activate()

	//DRAW
	bo.Activate()
	gl.DrawElements(gl.TRIANGLES, 36, gl.UNSIGNED_INT, gl.PtrOffset(0))
	bo.Deactivate()

	drawUI()

	window.GLSwap()
}

func drawUI() {

	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	fbWidth, fbHeight := window.GLGetDrawableSize()
	if fbWidth <= 0 || fbHeight <= 0 {
		return
	}

	drawData := imgui.RenderedDrawData()
	drawData.ScaleClipRects(imgui.Vec2{
		X: float32(fbWidth) / float32(winWidth),
		Y: float32(fbHeight) / float32(winHeight),
	})

	// Setup render state: alpha-blending enabled, no face culling, no depth testing, scissor enabled, polygon fill
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Setup viewport, orthographic projection matrix
	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.

	imShader.Activate()

	gl.Uniform1i(gl.GetUniformLocation(imShader.ID, gl.Str("Texture\x00")), 0)

	//PERF: only update the ortho matrix on window resize
	orthoMat := gglm.Ortho(0, float32(winWidth), 0, float32(winHeight), 0, 20)
	imShader.SetUnifMat4("ProjMtx", &orthoMat.Mat4)
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	gl.BindVertexArray(imguiInfo.vaoID)
	gl.BindBuffer(gl.ARRAY_BUFFER, imguiInfo.vboID)

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	imShader.EnableAttribute("Position")
	imShader.EnableAttribute("UV")
	imShader.EnableAttribute("Color")
	gl.VertexAttribPointerWithOffset(uint32(imShader.GetAttribLoc("Position")), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetPos))
	gl.VertexAttribPointerWithOffset(uint32(imShader.GetAttribLoc("UV")), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetUv))
	gl.VertexAttribPointerWithOffset(uint32(imShader.GetAttribLoc("Color")), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), uintptr(vertexOffsetCol))

	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	if indexSize == 4 {
		drawType = gl.UNSIGNED_INT
	}

	// Draw
	for _, list := range drawData.CommandLists() {

		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, imguiInfo.vboID)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.IndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, imguiInfo.indexBufID)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {

				gl.BindTexture(gl.TEXTURE_2D, imguiInfo.texID)
				// gl.BindTexture(gl.TEXTURE_2D, uint32(cmd.TextureID()))
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				gl.DrawElementsBaseVertex(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), gl.PtrOffset(cmd.IndexOffset()*indexSize), int32(cmd.VertexOffset()))
			}
		}
	}

	gl.BindVertexArray(imguiInfo.vaoID)
	gl.BindBuffer(gl.ARRAY_BUFFER, imguiInfo.vboID)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, imguiInfo.indexBufID)
}
