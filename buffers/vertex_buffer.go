package buffers

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type VertexBuffer struct {
	Id     uint32
	Stride int32
	layout []Element
}

func (vb *VertexBuffer) Bind() {
	gl.BindBuffer(gl.ARRAY_BUFFER, vb.Id)
}

func (vb *VertexBuffer) UnBind() {
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func (vb *VertexBuffer) SetData(values []float32, usage BufUsage) {

	vb.Bind()

	sizeInBytes := len(values) * 4
	if sizeInBytes == 0 {
		gl.BufferData(gl.ARRAY_BUFFER, 0, gl.Ptr(nil), usage.ToGL())
	} else {
		gl.BufferData(gl.ARRAY_BUFFER, sizeInBytes, gl.Ptr(&values[0]), usage.ToGL())
	}
}

func (vb *VertexBuffer) GetLayout() []Element {
	e := make([]Element, len(vb.layout))
	copy(e, vb.layout)
	return e
}

func (vb *VertexBuffer) SetLayout(layout ...Element) {

	vb.Stride = 0
	vb.layout = layout

	for i := 0; i < len(vb.layout); i++ {

		vb.layout[i].Offset = int(vb.Stride)
		vb.Stride += vb.layout[i].Size()
	}
}

func NewVertexBuffer(layout ...Element) VertexBuffer {

	vb := VertexBuffer{}

	gl.GenBuffers(1, &vb.Id)
	if vb.Id == 0 {
		logging.ErrLog.Panicln("Failed to create OpenGL buffer")
	}

	vb.SetLayout(layout...)
	return vb
}
