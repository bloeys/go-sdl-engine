package engine

import (
	"github.com/bloeys/nmage/renderer"
	"github.com/bloeys/nmage/timing"
	nmageimgui "github.com/bloeys/nmage/ui/imgui"
	"github.com/go-gl/gl/v4.1-core/gl"
)

var (
	isRunning = false
)

type Game interface {
	Init()

	Update()
	Render()
	FrameEnd()

	DeInit()
}

func Run(g Game, w *Window, rend renderer.Render, ui nmageimgui.ImguiInfo) {

	isRunning = true

	// Run init with an active Imgui frame to allow init full imgui access
	timing.FrameStarted()
	w.handleInputs()

	width, height := w.SDLWin.GetSize()
	ui.FrameStart(float32(width), float32(height))

	g.Init()

	fbWidth, fbHeight := w.SDLWin.GLGetDrawableSize()
	ui.Render(float32(width), float32(height), fbWidth, fbHeight)

	timing.FrameEnded()

	for isRunning {

		//PERF: Cache these
		width, height = w.SDLWin.GetSize()
		fbWidth, fbHeight = w.SDLWin.GLGetDrawableSize()

		timing.FrameStarted()
		w.handleInputs()
		ui.FrameStart(float32(width), float32(height))

		g.Update()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		g.Render()
		ui.Render(float32(width), float32(height), fbWidth, fbHeight)
		w.SDLWin.GLSwap()

		g.FrameEnd()
		rend.FrameEnd()
		timing.FrameEnded()
	}

	g.DeInit()
}

func Quit() {
	isRunning = false
}
