package input

import "github.com/veandco/go-sdl2/sdl"

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
	keyMap        = make(map[sdl.Keycode]keyState)
	mouseBtnMap   = make(map[int]mouseBtnState)
	mouseMotion   = mouseMotionState{}
	mouseWheel    = mouseWheelState{}
	quitRequested bool
)

func EventLoopStart() {

	for _, v := range keyMap {
		v.IsPressedThisFrame = false
		v.IsReleasedThisFrame = false
	}

	for _, v := range mouseBtnMap {
		v.IsPressedThisFrame = false
		v.IsReleasedThisFrame = false
		v.IsDoubleClicked = false
	}

	mouseMotion.XDelta = 0
	mouseMotion.YDelta = 0

	mouseWheel.XDelta = 0
	mouseWheel.YDelta = 0

	quitRequested = false
}

func HandleQuitEvent(e *sdl.QuitEvent) {
	quitRequested = true
}

func IsQuitClicked() bool {
	return quitRequested
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

// GetMousePos returns the window coordinates of the mouse
func GetMousePos() (x, y int32) {
	return mouseMotion.XPos, mouseMotion.YPos
}

// GetMouseMotion returns how many pixels were moved last frame
func GetMouseMotion() (xDelta, yDelta int32) {
	return mouseMotion.XDelta, mouseMotion.YDelta
}

func GetMouseMotionNorm() (xDelta, yDelta int32) {

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
	return mouseWheel.XDelta, mouseWheel.YDelta
}

// GetMouseWheelXNorm returns 1 if mouse wheel xDelta > 0, -1 if xDelta < 0, and 0 otherwise
func GetMouseWheelXNorm() int32 {

	if mouseWheel.XDelta > 0 {
		return 1
	} else if mouseWheel.XDelta < 0 {
		return -1
	}

	return 0
}

// returns 1 if mouse wheel yDelta > 0, -1 if yDelta < 0, and 0 otherwise
func GetMouseWheelYNorm() int32 {

	if mouseWheel.YDelta > 0 {
		return 1
	} else if mouseWheel.YDelta < 0 {
		return -1
	}

	return 0
}

func KeyClicked(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.IsPressedThisFrame
}

func KeyReleased(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.IsReleasedThisFrame
}

func KeyDown(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return false
	}

	return ks.State == sdl.PRESSED
}

func KeyUp(kc sdl.Keycode) bool {

	ks, ok := keyMap[kc]
	if !ok {
		return true
	}

	return ks.State == sdl.RELEASED
}

func MouseClicked(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsPressedThisFrame
}

func MouseDoubleClicked(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsDoubleClicked
}

func MouseReleased(mb int) bool {
	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.IsReleasedThisFrame
}

func MouseDown(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return false
	}

	return btn.State == sdl.PRESSED
}

func MouseUp(mb int) bool {

	btn, ok := mouseBtnMap[mb]
	if !ok {
		return true
	}

	return btn.State == sdl.RELEASED
}
