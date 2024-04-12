package buffers

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type IndexBuffer struct {
	Id uint32
	// IndexBufCount is the number of elements in the index buffer. Updated in IndexBuffer.SetData
	IndexBufCount int32
}

func (ib *IndexBuffer) Bind() {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ib.Id)
}

func (ib *IndexBuffer) UnBind() {
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
}

func (ib *IndexBuffer) SetData(values []uint32) {

	ib.Bind()

	sizeInBytes := len(values) * 4
	ib.IndexBufCount = int32(len(values))

	if sizeInBytes == 0 {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 0, gl.Ptr(nil), BufUsage_Static.ToGL())
	} else {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, sizeInBytes, gl.Ptr(&values[0]), BufUsage_Static.ToGL())
	}
}

func NewIndexBuffer() IndexBuffer {

	ib := IndexBuffer{}

	gl.GenBuffers(1, &ib.Id)
	if ib.Id == 0 {
		logging.ErrLog.Println("Failed to create OpenGL buffer")
	}

	return ib
}
