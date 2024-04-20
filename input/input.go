// The input package provides an interface to mouse and keyboard inputs
// like key clicks and releases, along with some higher level constructs like
// pressed/released this frames, double clicks, and normalized inputs.
//
// The input package has two sets of functions for most cases, where one
// is in the form 'xy' and the other 'xyCaptured'. The captured form
// always returns normal events even if the mouse or keyboard are captured
// by the UI system. The 'xy' form however will return zero/false if the
// respective input device is currently captured (with the exception of mouse position, that is always correctly returned).
//
// For most cases, you want to use the 'xy' form. For example, you only want to receive
// key down events for game character movement when the UI isn't capturing the keyboard,
// because otherwise the character will move while typing in a UI textbox.
//
// The functions IsMouseCaptured and IsKeyboardCaptured are also available.
package input

import (
	"github.com/veandco/go-sdl2/sdl"
)

type keyState struct {
	Key                 sdl.Keycode
	State               int
	IsPressedThisFrame  bool
	IsReleasedThisFrame bool
}

type mouseBtnState struct {
	Btn   int
	State int

	IsPressedThisFrame  bool
	IsReleasedThisFrame bool
	IsDoubleClicked     bool
}

type mouseMotionState struct {
	XDelta int32
	YDelta int32
	XPos   int32
	YPos   int32
}

type mouseWheelState struct {
	XDelta int32
	YDelta int32
}

var (
	mouseWheel  = mouseWheelState{}
	mouseMotion = mouseMotionState{}
	mouseBtnMap = make(map[int]mouseBtnState)
	keyMap      = make(map[sdl.Keycode]keyState)

	isQuitRequested    bool
	isMouseCaptured    bool
	isKeyboardCaptured bool
)

func EventLoopStart(mouseGotCaptured, keyboardGotCaptured bool) {

	isMouseCaptured = mouseGotCaptured
	isKeyboardCaptured = keyboardGotCaptured

	// Update per-frame state
	for k, v := range keyMap {
		v.IsPressedThisFrame = false
		v.IsReleasedThisFrame = false
		keyMap[k] = v
	}

	for k, v := range mouseBtnMap {
		v.IsPressedThisFrame = false
		v.IsReleasedThisFrame = false
		v.IsDoubleClicked = false
		mouseBtnMap[k] = v
	}

	mouseMotion.XDelta = 0
	mouseMotion.YDelta = 0

	mouseWheel.XDelta = 0
	mouseWheel.YDelta = 0

	isQuitRequested = false
}

func ClearKeyboardState() {
	clear(keyMap)
}

func ClearMouseState() {
	clear(mouseBtnMap)
	mouseMotion = mouseMotionState{}
	mouseWheel = mouseWheelState{}
}

func HandleQuitEvent(e *sdl.QuitEvent) {
	isQuitRequested = true
}

func IsMouseCaptured() bool {
	return isMouseCaptured
}

func IsKeyboardCaptured() bool {
	return isKeyboardCaptured
}

func IsQuitClicked() bool {
	return isQuitRequested
}

func HandleKeyboardEvent(e *sdl.KeyboardEvent) {

	ks, ok := keyMap[e.Keysym.Sym]
	if !ok {
		ks = keyState{Key: e.Keysym.Sym}
	}

	ks.State = int(e.State)
	ks.IsPressedThisFrame = e.State == sdl.PRESSED && e.Repeat == 0
	ks.IsReleasedThisFrame = e.State == sdl.RELEASED && e.Repeat == 0

	keyMap[ks.Key] = ks
}

func HandleMouseBtnEvent(e *sdl.MouseButtonEvent) {

	mb, ok := mouseBtnMap[int(e.Button)]
	if !ok {
		mb = mouseBtnState{Btn: int(e.Button)}
	}

	mb.State = int(e.State)
	mb.IsDoubleClicked = e.Clicks == 2 && e.State == sdl.PRESSED
	mb.IsPressedThisFrame = e.State == sdl.PRESSED
	mb.IsReleasedThisFrame = e.State == sdl.RELEASED

	mouseBtnMap[int(e.Button)] = mb
}

func HandleMouseMotionEvent(e *sdl.MouseMotionEvent) {

	mouseMotion.XPos = e.X
	mouseMotion.YPos = e.Y

	mouseMotion.XDelta = e.XRel
	mouseMotion.YDelta = e.YRel
}

func HandleMouseWheelEvent(e *sdl.MouseWheelEvent) {
	mouseWheel.XDelta = e.X
	mouseWheel.YDelta = e.Y
}

// GetMousePos returns the window coordinates of the mouse regardless of whether the mouse is captured or not
func GetMousePos() (x, y int32) {
	return mouseMotion.XPos, mouseMotion.YPos
}

// GetMouseMotion returns how many pixels were moved last frame
func GetMouseMotion() (xDelta, yDelta int32) {

	if isMouseCaptured {
		return 0, 0
	}

	return GetMouseMotionCaptured()
}

func GetMouseMotionCaptured() (xDelta, yDelta int32) {
	return mouseMotion.XDelta, mouseMotion.YDelta
}

func GetMouseMotionNorm() (xDelta, yDelta int32) {

	if isMouseCaptured {
		return 0, 0
	}

	return GetMouseMotionNormCaptured()
}

func GetMouseMotionNormCaptured() (xDelta, yDelta int32) {

	x, y := mouseMotion.XDelta, mouseMotion.YDelta
	if x > 0 {
		x = 1
	} else if x < 0 {
		x = -1
	}

	if y > 0 {
		y = -1
	} else if y < 0 {
		y = 1
	}

	return x, y
}

func GetMouseWheelMotion() (xDelta, yDelta int32) {

	if isMouseCaptured {
		return 0, 0
	}

	return GetMouseWheelMotionCaptured()
}

func GetMouseWheelMotionCaptured() (xDelta, yDelta int32) {
	return mouseWheel.XDelta, mouseWheel.YDelta
}

// GetMouseWheelXNorm returns 1 if mouse wheel xDelta > 0, -1 if xDelta < 0, and 0 otherwise
func GetMouseWheelXNorm() int32 {

	if isMouseCaptured {
		return 0
	}

	return GetMouseWheelXNormCaptured()
}

// GetMouseWheelXNormCaptured returns 1 if mouse wheel xDelta > 0, -1 if xDelta < 0, and 0 otherwise
func GetMouseWheelXNormCaptured() int32 {

	if mouseWheel.XDelta > 0 {
		return 1
	} else if mouseWheel.XDelta < 0 {
		return -1
	}

	return 0
}

// GetMouseWheelYNorm returns 1 if mouse wheel yDelta > 0, -1 if yDelta < 0, and 0 otherwise
func GetMouseWheelYNorm() int32 {

	if isMouseCaptured {
		return 0
	}

	return GetMouseWheelYNormCaptured()
}

// GetMouseWheelYNormCaptured returns 1 if mouse wheel yDelta > 0, -1 if yDelta < 0, and 0 otherwise
func GetMouseWheelYNormCaptured() int32 {

	if mouseWheel.YDelta > 0 {
		return 1
	} else if mouseWheel.YDelta < 0 {
		return -1
	}

	return 0
}

func KeyClicked(kc sdl.Keycode) bool {

	if isKeyboardCaptured {
		return false
	}

	return KeyClickedCaptured(kc)
}

func KeyClickedCaptured(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.IsPressedThisFrame
}

func KeyReleased(kc sdl.Keycode) bool {

	if isKeyboardCaptured {
		return false
	}

	return KeyReleasedCaptured(kc)
}

func KeyReleasedCaptured(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.IsReleasedThisFrame
}

func KeyDown(kc sdl.Keycode) bool {

	if isKeyboardCaptured {
		return false
	}

	return KeyDownCaptured(kc)
}

func KeyDownCaptured(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.State == sdl.PRESSED
}

func KeyUp(kc sdl.Keycode) bool {

	if isKeyboardCaptured {
		return false
	}

	return KeyUpCaptured(kc)
}

func KeyUpCaptured(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return true
	}

	return ks.State == sdl.RELEASED
}

func MouseClicked(mb int) bool {

	if isMouseCaptured {
		return false
	}

	return MouseClickedCaptued(mb)
}

func MouseClickedCaptued(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsPressedThisFrame
}

func MouseDoubleClicked(mb int) bool {

	if isMouseCaptured {
		return false
	}

	return MouseDoubleClickedCaptured(mb)
}

func MouseDoubleClickedCaptured(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsDoubleClicked
}

func MouseReleased(mb int) bool {

	if isMouseCaptured {
		return false
	}

	return MouseReleasedCaptured(mb)
}

func MouseReleasedCaptured(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsReleasedThisFrame
}

func MouseDown(mb int) bool {

	if isMouseCaptured {
		return false
	}

	return MouseDownCaptued(mb)
}

func MouseDownCaptued(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.State == sdl.PRESSED
}

func MouseUp(mb int) bool {

	if isMouseCaptured {
		return false
	}

	return MouseUpCaptured(mb)
}

func MouseUpCaptured(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return true
	}

	return btn.State == sdl.RELEASED
}
