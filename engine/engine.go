package engine

import (
	"image"
	"image/color"
	"runtime"

	imgui "github.com/AllenDang/cimgui-go"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/assets"
	"github.com/bloeys/nmage/input"
	"github.com/bloeys/nmage/timing"
	nmageimgui "github.com/bloeys/nmage/ui/imgui"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	isInited = false

	isSdlButtonLeftDown   = false
	isSdlButtonMiddleDown = false
	isSdlButtonRightDown  = false

	ImguiRelativeMouseModePosX float32
	ImguiRelativeMouseModePosY float32
)

type Window struct {
	SDLWin         *sdl.Window
	GlCtx          sdl.GLContext
	EventCallbacks []func(sdl.Event)
}

func (w *Window) handleInputs() {

	imIo := imgui.CurrentIO()

	imguiCaptureMouse := imIo.WantCaptureMouse()
	imguiCaptureKeyboard := imIo.WantCaptureKeyboard()

	input.EventLoopStart(imguiCaptureMouse, imguiCaptureKeyboard)

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

		//Fire callbacks
		for i := 0; i < len(w.EventCallbacks); i++ {
			w.EventCallbacks[i](event)
		}

		//Internal processing
		switch e := event.(type) {

		case *sdl.MouseWheelEvent:

			input.HandleMouseWheelEvent(e)
			imIo.AddMouseWheelDelta(float32(e.X), float32(e.Y))

		case *sdl.KeyboardEvent:

			input.HandleKeyboardEvent(e)

			// Send modifier key updates to imgui (based on the imgui SDL backend)
			imIo.AddKeyEvent(imgui.ModCtrl, e.Keysym.Mod&sdl.KMOD_CTRL != 0)
			imIo.AddKeyEvent(imgui.ModShift, e.Keysym.Mod&sdl.KMOD_SHIFT != 0)
			imIo.AddKeyEvent(imgui.ModAlt, e.Keysym.Mod&sdl.KMOD_ALT != 0)
			imIo.AddKeyEvent(imgui.ModSuper, e.Keysym.Mod&sdl.KMOD_GUI != 0)

			imIo.AddKeyEvent(nmageimgui.SdlScancodeToImGuiKey(e.Keysym.Scancode), e.Type == sdl.KEYDOWN)

		case *sdl.TextInputEvent:
			imIo.AddInputCharactersUTF8(e.GetText())

		case *sdl.MouseButtonEvent:

			input.HandleMouseBtnEvent(e)
			isPressed := e.State == sdl.PRESSED

			if e.Button == sdl.BUTTON_LEFT {
				isSdlButtonLeftDown = isPressed
			} else if e.Button == sdl.BUTTON_MIDDLE {
				isSdlButtonMiddleDown = isPressed
			} else if e.Button == sdl.BUTTON_RIGHT {
				isSdlButtonRightDown = isPressed
			}

		case *sdl.MouseMotionEvent:

			input.HandleMouseMotionEvent(e)

		case *sdl.WindowEvent:

			if e.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
				w.handleWindowResize()
			}

		case *sdl.QuitEvent:
			input.HandleQuitEvent(e)
		}
	}

	if sdl.GetRelativeMouseMode() {
		imIo.SetMousePos(imgui.Vec2{X: ImguiRelativeMouseModePosX, Y: ImguiRelativeMouseModePosY})
	} else {
		x, y, _ := sdl.GetMouseState()
		imIo.SetMousePos(imgui.Vec2{X: float32(x), Y: float32(y)})
	}

	// If a mouse press event came, always pass it as "mouse held this frame", so we don't miss click-release events that are shorter than 1 frame.
	imIo.SetMouseButtonDown(int(imgui.MouseButtonLeft), isSdlButtonLeftDown)
	imIo.SetMouseButtonDown(int(imgui.MouseButtonRight), isSdlButtonRightDown)
	imIo.SetMouseButtonDown(int(imgui.MouseButtonMiddle), isSdlButtonMiddleDown)
}

func (w *Window) handleWindowResize() {

	fbWidth, fbHeight := w.SDLWin.GLGetDrawableSize()
	if fbWidth <= 0 || fbHeight <= 0 {
		return
	}
	gl.Viewport(0, 0, fbWidth, fbHeight)
}

func (w *Window) Destroy() error {
	return w.SDLWin.Destroy()
}

func Init() error {

	isInited = true

	runtime.LockOSThread()
	timing.Init()
	err := initSDL()

	return err
}

func initSDL() error {

	err := sdl.Init(sdl.INIT_TIMER | sdl.INIT_VIDEO)
	if err != nil {
		return err
	}

	sdl.ShowCursor(1)

	sdl.GLSetAttribute(sdl.MAJOR_VERSION, 4)
	sdl.GLSetAttribute(sdl.MINOR_VERSION, 1)

	sdl.GLSetAttribute(sdl.GL_RED_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_GREEN_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_BLUE_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_ALPHA_SIZE, 8)

	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)
	sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, 8)

	sdl.GLSetAttribute(sdl.GL_FRAMEBUFFER_SRGB_CAPABLE, 1)

	// Allows us to do MSAA
	sdl.GLSetAttribute(sdl.GL_MULTISAMPLEBUFFERS, 1)
	sdl.GLSetAttribute(sdl.GL_MULTISAMPLESAMPLES, 4)

	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)

	return nil
}

func CreateOpenGLWindow(title string, x, y, width, height int32, flags WindowFlags) (Window, error) {
	return createWindow(title, x, y, width, height, WindowFlags_OPENGL|flags)
}

func CreateOpenGLWindowCentered(title string, width, height int32, flags WindowFlags) (Window, error) {
	return createWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, width, height, WindowFlags_OPENGL|flags)
}

func createWindow(title string, x, y, width, height int32, flags WindowFlags) (Window, error) {

	assert.T(isInited, "engine.Init() was not called!")

	win := Window{
		SDLWin:         nil,
		EventCallbacks: make([]func(sdl.Event), 0),
	}

	var err error

	win.SDLWin, err = sdl.CreateWindow(title, x, y, width, height, uint32(flags))
	if err != nil {
		return win, err
	}

	win.GlCtx, err = win.SDLWin.GLCreateContext()
	if err != nil {
		return win, err
	}

	err = initOpenGL()
	if err != nil {
		return win, err
	}

	setupDefaultTextures()

	// Get rid of the blinding white startup screen (unfortunately there is still one frame of white)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
	win.SDLWin.GLSwap()

	return win, err
}

func initOpenGL() error {

	if err := gl.Init(); err != nil {
		return err
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.STENCIL_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)

	gl.Enable(gl.BLEND)
	gl.Enable(gl.MULTISAMPLE)
	gl.Enable(gl.FRAMEBUFFER_SRGB)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.ClearColor(0, 0, 0, 1)

	return nil
}

func setupDefaultTextures() error {

	// 1x1 black texture
	defaultBlackImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	defaultBlackImg.Set(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 1})
	defaultBlackImgTex, err := assets.LoadTextureInMemPngImg(defaultBlackImg, &assets.TextureLoadOptions{NoSrgba: true})
	if err != nil {
		return err
	}
	assets.DefaultBlackTexId = defaultBlackImgTex

	// 1x1 white texture
	defaultWhiteImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	defaultWhiteImg.Set(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 1})
	defaultWhiteImgTex, err := assets.LoadTextureInMemPngImg(defaultWhiteImg, &assets.TextureLoadOptions{NoSrgba: true})
	if err != nil {
		return err
	}
	assets.DefaultWhiteTexId = defaultWhiteImgTex

	// Default diffuse
	assets.DefaultDiffuseTexId = defaultWhiteImgTex

	// Default specular
	assets.DefaultSpecularTexId = defaultBlackImgTex

	// Default Normal map which is created to be RGB(0.5,0.5,1), which when multiplied by TBN matrix gives the vertex normal.
	// 128 is better than 127 for normal maps. See 'Flat Color' section here: http://wiki.polycount.com/wiki/Normal_map
	// Basically, 127 can create seams while 128 looks correct
	defaultNormalMapImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	defaultNormalMapImg.Set(0, 0, color.NRGBA{R: 128, G: 128, B: 255, A: 1})
	defaultNormalMapTex, err := assets.LoadTextureInMemPngImg(defaultNormalMapImg, &assets.TextureLoadOptions{NoSrgba: true})
	if err != nil {
		return err
	}

	assets.DefaultNormalTexId = defaultNormalMapTex

	// Default emission
	assets.DefaultEmissionTexId = defaultBlackImgTex

	assert.T(assets.DefaultBlackTexId.TexID != 0, "The default black texture handle is zero. Either texture wasn't created or handle wasn't updated")
	assert.T(assets.DefaultWhiteTexId.TexID != 0, "The default white texture handle is zero. Either texture wasn't created or handle wasn't updated")
	assert.T(assets.DefaultDiffuseTexId.TexID != 0, "The default diffuse texture handle is zero. Either texture wasn't created or handle wasn't updated")
	assert.T(assets.DefaultSpecularTexId.TexID != 0, "The default specular texture handle is zero. Either texture wasn't created or handle wasn't updated")
	assert.T(assets.DefaultNormalTexId.TexID != 0, "The default normal texture handle is zero. Either texture wasn't created or handle wasn't updated")
	assert.T(assets.DefaultEmissionTexId.TexID != 0, "The default emission texture handle is zero. Either texture wasn't created or handle wasn't updated")

	return nil
}

func SetSrgbFramebuffer(isEnabled bool) {

	if isEnabled {
		gl.Enable(gl.FRAMEBUFFER_SRGB)
	} else {
		gl.Disable(gl.FRAMEBUFFER_SRGB)
	}
}

func SetVSync(enabled bool) {

	if enabled {
		sdl.GLSetSwapInterval(1)
	} else {
		sdl.GLSetSwapInterval(0)
	}
}

func SetMSAA(isEnabled bool) {

	if isEnabled {
		gl.Enable(gl.MULTISAMPLE)
	} else {
		gl.Disable(gl.MULTISAMPLE)
	}
}
