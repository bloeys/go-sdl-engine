package nmageimgui

import (
	imgui "github.com/AllenDang/cimgui-go"
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/materials"
	"github.com/bloeys/nmage/timing"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

type ImguiInfo struct {
	ImCtx imgui.Context

	Mat        materials.Material
	VaoID      uint32
	VboID      uint32
	IndexBufID uint32
	// This is a pointer so we can send a stable pointer to C code
	TexID *uint32
}

func (i *ImguiInfo) FrameStart(winWidth, winHeight float32) {

	// if err := i.ImCtx.SetCurrent(); err != nil {
	// 	assert.T(false, "Setting imgui ctx as current failed. Err: "+err.Error())
	// }

	imIO := imgui.CurrentIO()
	imIO.SetDisplaySize(imgui.Vec2{X: float32(winWidth), Y: float32(winHeight)})
	imIO.SetDeltaTime(timing.DT())

	imgui.NewFrame()
}

func (i *ImguiInfo) Render(winWidth, winHeight float32, fbWidth, fbHeight int32) {

	// if err := i.ImCtx.SetCurrent(); err != nil {
	// 	assert.T(false, "Setting imgui ctx as current failed. Err: "+err.Error())
	// }

	imgui.Render()

	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	if fbWidth <= 0 || fbHeight <= 0 {
		return
	}

	drawData := imgui.CurrentDrawData()
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

	i.Mat.Bind()
	i.Mat.SetUnifInt32("Texture", 0)

	// @PERF: only update the ortho matrix on window resize
	orthoMat := gglm.Ortho(0, float32(winWidth), 0, float32(winHeight), 0, 20)
	i.Mat.SetUnifMat4("ProjMtx", &orthoMat.Mat4)
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	gl.BindVertexArray(i.VaoID)
	gl.BindBuffer(gl.ARRAY_BUFFER, i.VboID)

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	i.Mat.EnableAttribute("Position")
	i.Mat.EnableAttribute("UV")
	i.Mat.EnableAttribute("Color")
	gl.VertexAttribPointerWithOffset(uint32(i.Mat.GetAttribLoc("Position")), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetPos))
	gl.VertexAttribPointerWithOffset(uint32(i.Mat.GetAttribLoc("UV")), 2, gl.FLOAT, false, int32(vertexSize), uintptr(vertexOffsetUv))
	gl.VertexAttribPointerWithOffset(uint32(i.Mat.GetAttribLoc("Color")), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), uintptr(vertexOffsetCol))

	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	if indexSize == 4 {
		drawType = gl.UNSIGNED_INT
	}

	// Draw
	for _, list := range drawData.CommandLists() {

		vertexBuffer, vertexBufferSize := list.GetVertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, i.VboID)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.GetIndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, i.IndexBufID)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {

				gl.ActiveTexture(gl.TEXTURE0)
				gl.BindTexture(gl.TEXTURE_2D, *i.TexID)
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))

				gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, int32(cmd.ElemCount()), uint32(drawType), uintptr(int(cmd.IdxOffset())*indexSize), int32(cmd.VtxOffset()))
			}
		}
	}

	//Reset gl state
	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (i *ImguiInfo) AddFontTTF(fontPath string, fontSize float32, fontConfig *imgui.FontConfig, glyphRanges *imgui.GlyphRange) imgui.Font {

	fontConfigToUse := imgui.NewFontConfig()
	if fontConfig != nil {
		fontConfigToUse = *fontConfig
	}

	glyphRangesToUse := imgui.NewGlyphRange()
	if glyphRanges != nil {
		glyphRangesToUse = *glyphRanges
	}

	imIO := imgui.CurrentIO()

	a := imIO.Fonts()
	f := a.AddFontFromFileTTFV(fontPath, fontSize, fontConfigToUse, glyphRangesToUse.Data())
	pixels, width, height, _ := a.GetTextureDataAsAlpha8()

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, *i.TexID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(width), int32(height), 0, gl.RED, gl.UNSIGNED_BYTE, pixels)

	return f
}

const DefaultImguiShader = `
//shader:vertex
#version 410

uniform mat4 ProjMtx;

in vec2 Position;
in vec2 UV;
in vec4 Color;

out vec2 Frag_UV;
out vec4 Frag_Color;

// Imgui doesn't handle srgb correctly, and looks too bright and wrong in srgb buffers (see: https://github.com/ocornut/imgui/issues/578).
// While not a complete fix (that would require changes in imgui itself), moving incoming srgba colors to linear in the vertex shader helps make things look better.
vec4 srgba_to_linear(vec4 srgbaColor){

    #define gamma_correction 2.2

    return vec4(
        pow(srgbaColor.r, gamma_correction),
        pow(srgbaColor.g, gamma_correction),
        pow(srgbaColor.b, gamma_correction),
        srgbaColor.a
    );
}

void main()
{
    Frag_UV = UV;
    Frag_Color = srgba_to_linear(Color);
    gl_Position = ProjMtx * vec4(Position.xy, 0, 1);
}

//shader:fragment
#version 410

uniform sampler2D Texture;

in vec2 Frag_UV;
in vec4 Frag_Color;

out vec4 Out_Color;

void main()
{
    Out_Color = vec4(Frag_Color.rgb, Frag_Color.a * texture(Texture, Frag_UV.st).r);
}
`

// NewImGui setups imgui using the passed shader.
// If the path is empty a default nMage shader is used
func NewImGui(shaderPath string) ImguiInfo {

	var imguiMat materials.Material
	if shaderPath == "" {
		imguiMat = materials.NewMaterialSrc("ImGUI Mat", []byte(DefaultImguiShader))
	} else {
		imguiMat = materials.NewMaterial("ImGUI Mat", shaderPath)
	}

	imguiInfo := ImguiInfo{
		ImCtx: imgui.CreateContext(),
		Mat:   imguiMat,
		TexID: new(uint32),
	}

	io := imgui.CurrentIO()
	io.SetConfigFlags(io.ConfigFlags() | imgui.ConfigFlagsDockingEnable)
	io.SetBackendFlags(io.BackendFlags() | imgui.BackendFlagsRendererHasVtxOffset)

	gl.GenVertexArrays(1, &imguiInfo.VaoID)
	gl.GenBuffers(1, &imguiInfo.VboID)
	gl.GenBuffers(1, &imguiInfo.IndexBufID)
	gl.GenTextures(1, imguiInfo.TexID)

	// Upload font to gpu
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, *imguiInfo.TexID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	pixels, width, height, _ := io.Fonts().GetTextureDataAsAlpha8()
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(width), int32(height), 0, gl.RED, gl.UNSIGNED_BYTE, pixels)

	// Store our identifier
	io.Fonts().SetTexID(imgui.TextureID(imguiInfo.TexID))

	//Shader attributes
	imguiInfo.Mat.Bind()
	imguiInfo.Mat.EnableAttribute("Position")
	imguiInfo.Mat.EnableAttribute("UV")
	imguiInfo.Mat.EnableAttribute("Color")
	imguiInfo.Mat.UnBind()

	return imguiInfo
}

func SdlScancodeToImGuiKey(scancode sdl.Scancode) imgui.Key {

	switch scancode {
	case sdl.SCANCODE_TAB:
		return imgui.KeyTab
	case sdl.SCANCODE_LEFT:
		return imgui.KeyLeftArrow
	case sdl.SCANCODE_RIGHT:
		return imgui.KeyRightArrow
	case sdl.SCANCODE_UP:
		return imgui.KeyUpArrow
	case sdl.SCANCODE_DOWN:
		return imgui.KeyDownArrow
	case sdl.SCANCODE_PAGEUP:
		return imgui.KeyPageUp
	case sdl.SCANCODE_PAGEDOWN:
		return imgui.KeyPageDown
	case sdl.SCANCODE_HOME:
		return imgui.KeyHome
	case sdl.SCANCODE_END:
		return imgui.KeyEnd
	case sdl.SCANCODE_INSERT:
		return imgui.KeyInsert
	case sdl.SCANCODE_DELETE:
		return imgui.KeyDelete
	case sdl.SCANCODE_BACKSPACE:
		return imgui.KeyBackspace
	case sdl.SCANCODE_SPACE:
		return imgui.KeySpace
	case sdl.SCANCODE_RETURN:
		return imgui.KeyEnter
	case sdl.SCANCODE_ESCAPE:
		return imgui.KeyEscape
	case sdl.SCANCODE_APOSTROPHE:
		return imgui.KeyApostrophe
	case sdl.SCANCODE_COMMA:
		return imgui.KeyComma
	case sdl.SCANCODE_MINUS:
		return imgui.KeyMinus
	case sdl.SCANCODE_PERIOD:
		return imgui.KeyPeriod
	case sdl.SCANCODE_SLASH:
		return imgui.KeySlash
	case sdl.SCANCODE_SEMICOLON:
		return imgui.KeySemicolon
	case sdl.SCANCODE_EQUALS:
		return imgui.KeyEqual
	case sdl.SCANCODE_LEFTBRACKET:
		return imgui.KeyLeftBracket
	case sdl.SCANCODE_BACKSLASH:
		return imgui.KeyBackslash
	case sdl.SCANCODE_RIGHTBRACKET:
		return imgui.KeyRightBracket
	case sdl.SCANCODE_GRAVE:
		return imgui.KeyGraveAccent
	case sdl.SCANCODE_CAPSLOCK:
		return imgui.KeyCapsLock
	case sdl.SCANCODE_SCROLLLOCK:
		return imgui.KeyScrollLock
	case sdl.SCANCODE_NUMLOCKCLEAR:
		return imgui.KeyNumLock
	case sdl.SCANCODE_PRINTSCREEN:
		return imgui.KeyPrintScreen
	case sdl.SCANCODE_PAUSE:
		return imgui.KeyPause
	case sdl.SCANCODE_KP_0:
		return imgui.KeyKeypad0
	case sdl.SCANCODE_KP_1:
		return imgui.KeyKeypad1
	case sdl.SCANCODE_KP_2:
		return imgui.KeyKeypad2
	case sdl.SCANCODE_KP_3:
		return imgui.KeyKeypad3
	case sdl.SCANCODE_KP_4:
		return imgui.KeyKeypad4
	case sdl.SCANCODE_KP_5:
		return imgui.KeyKeypad5
	case sdl.SCANCODE_KP_6:
		return imgui.KeyKeypad6
	case sdl.SCANCODE_KP_7:
		return imgui.KeyKeypad7
	case sdl.SCANCODE_KP_8:
		return imgui.KeyKeypad8
	case sdl.SCANCODE_KP_9:
		return imgui.KeyKeypad9
	case sdl.SCANCODE_KP_PERIOD:
		return imgui.KeyKeypadDecimal
	case sdl.SCANCODE_KP_DIVIDE:
		return imgui.KeyKeypadDivide
	case sdl.SCANCODE_KP_MULTIPLY:
		return imgui.KeyKeypadMultiply
	case sdl.SCANCODE_KP_MINUS:
		return imgui.KeyKeypadSubtract
	case sdl.SCANCODE_KP_PLUS:
		return imgui.KeyKeypadAdd
	case sdl.SCANCODE_KP_ENTER:
		return imgui.KeyKeypadEnter
	case sdl.SCANCODE_KP_EQUALS:
		return imgui.KeyKeypadEqual
	case sdl.SCANCODE_LSHIFT:
		return imgui.KeyLeftShift
	case sdl.SCANCODE_LCTRL:
		return imgui.KeyLeftCtrl
	case sdl.SCANCODE_LALT:
		return imgui.KeyLeftAlt
	case sdl.SCANCODE_LGUI:
		return imgui.KeyLeftSuper
	case sdl.SCANCODE_RSHIFT:
		return imgui.KeyRightShift
	case sdl.SCANCODE_RCTRL:
		return imgui.KeyRightCtrl
	case sdl.SCANCODE_RALT:
		return imgui.KeyRightAlt
	case sdl.SCANCODE_RGUI:
		return imgui.KeyRightSuper
	case sdl.SCANCODE_MENU:
		return imgui.KeyMenu
	case sdl.SCANCODE_0:
		return imgui.Key0
	case sdl.SCANCODE_1:
		return imgui.Key1
	case sdl.SCANCODE_2:
		return imgui.Key2
	case sdl.SCANCODE_3:
		return imgui.Key3
	case sdl.SCANCODE_4:
		return imgui.Key4
	case sdl.SCANCODE_5:
		return imgui.Key5
	case sdl.SCANCODE_6:
		return imgui.Key6
	case sdl.SCANCODE_7:
		return imgui.Key7
	case sdl.SCANCODE_8:
		return imgui.Key8
	case sdl.SCANCODE_9:
		return imgui.Key9
	case sdl.SCANCODE_A:
		return imgui.KeyA
	case sdl.SCANCODE_B:
		return imgui.KeyB
	case sdl.SCANCODE_C:
		return imgui.KeyC
	case sdl.SCANCODE_D:
		return imgui.KeyD
	case sdl.SCANCODE_E:
		return imgui.KeyE
	case sdl.SCANCODE_F:
		return imgui.KeyF
	case sdl.SCANCODE_G:
		return imgui.KeyG
	case sdl.SCANCODE_H:
		return imgui.KeyH
	case sdl.SCANCODE_I:
		return imgui.KeyI
	case sdl.SCANCODE_J:
		return imgui.KeyJ
	case sdl.SCANCODE_K:
		return imgui.KeyK
	case sdl.SCANCODE_L:
		return imgui.KeyL
	case sdl.SCANCODE_M:
		return imgui.KeyM
	case sdl.SCANCODE_N:
		return imgui.KeyN
	case sdl.SCANCODE_O:
		return imgui.KeyO
	case sdl.SCANCODE_P:
		return imgui.KeyP
	case sdl.SCANCODE_Q:
		return imgui.KeyQ
	case sdl.SCANCODE_R:
		return imgui.KeyR
	case sdl.SCANCODE_S:
		return imgui.KeyS
	case sdl.SCANCODE_T:
		return imgui.KeyT
	case sdl.SCANCODE_U:
		return imgui.KeyU
	case sdl.SCANCODE_V:
		return imgui.KeyV
	case sdl.SCANCODE_W:
		return imgui.KeyW
	case sdl.SCANCODE_X:
		return imgui.KeyX
	case sdl.SCANCODE_Y:
		return imgui.KeyY
	case sdl.SCANCODE_Z:
		return imgui.KeyZ
	case sdl.SCANCODE_F1:
		return imgui.KeyF1
	case sdl.SCANCODE_F2:
		return imgui.KeyF2
	case sdl.SCANCODE_F3:
		return imgui.KeyF3
	case sdl.SCANCODE_F4:
		return imgui.KeyF4
	case sdl.SCANCODE_F5:
		return imgui.KeyF5
	case sdl.SCANCODE_F6:
		return imgui.KeyF6
	case sdl.SCANCODE_F7:
		return imgui.KeyF7
	case sdl.SCANCODE_F8:
		return imgui.KeyF8
	case sdl.SCANCODE_F9:
		return imgui.KeyF9
	case sdl.SCANCODE_F10:
		return imgui.KeyF10
	case sdl.SCANCODE_F11:
		return imgui.KeyF11
	case sdl.SCANCODE_F12:
		return imgui.KeyF12
	default:
		return imgui.KeyNone
	}
}
