package buffers

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type VertexArray struct {
	Id          uint32
	Vbos        []VertexBuffer
	IndexBuffer IndexBuffer
}

func (va *VertexArray) Bind() {
	gl.BindVertexArray(va.Id)
}

func (va *VertexArray) UnBind() {
	gl.BindVertexArray(0)
}

func (va *VertexArray) AddVertexBuffer(vbo VertexBuffer) {

	// NOTE: VBOs are only bound at 'VertexAttribPointer' (and related) calls

	va.Bind()
	vbo.Bind()

	for i := 0; i < len(vbo.layout); i++ {

		l := &vbo.layout[i]

		gl.EnableVertexAttribArray(uint32(i))
		gl.VertexAttribPointerWithOffset(uint32(i), l.ElementType.CompCount(), l.ElementType.GLType(), false, vbo.Stride, uintptr(l.Offset))
	}
}

func (va *VertexArray) SetIndexBuffer(ib IndexBuffer) {
	va.Bind()
	ib.Bind()
	va.IndexBuffer = ib
}

func NewVertexArray() VertexArray {

	vao := VertexArray{}

	gl.GenVertexArrays(1, &vao.Id)
	if vao.Id == 0 {
		logging.ErrLog.Println("Failed to create OpenGL vertex array object")
	}

	return vao
}
