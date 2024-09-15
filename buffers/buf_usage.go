package buffers

import (
	"fmt"

	"github.com/bloeys/nmage/assert"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type BufUsage int

// Full docs for buffer usage can be found here: https://registry.khronos.org/OpenGL-Refpages/gl4/html/glBufferData.xhtml
const (
	BufUsage_Unknown BufUsage = iota

	//Buffer is set only once and used many times
	BufUsage_Static_Draw
	//Buffer is changed a lot and used many times
	BufUsage_Dynamic_Draw
	//Buffer is set only once and used by the GPU at most a few times
	BufUsage_Stream_Draw

	BufUsage_Static_Read
	BufUsage_Dynamic_Read
	BufUsage_Stream_Read

	BufUsage_Static_Copy
	BufUsage_Dynamic_Copy
	BufUsage_Stream_Copy
)

func (b BufUsage) ToGL() uint32 {
	switch b {
	case BufUsage_Static_Draw:
		return gl.STATIC_DRAW
	case BufUsage_Dynamic_Draw:
		return gl.DYNAMIC_DRAW
	case BufUsage_Stream_Draw:
		return gl.STREAM_DRAW

	case BufUsage_Static_Read:
		return gl.STATIC_READ
	case BufUsage_Dynamic_Read:
		return gl.DYNAMIC_READ
	case BufUsage_Stream_Read:
		return gl.STREAM_READ

	case BufUsage_Static_Copy:
		return gl.STATIC_COPY
	case BufUsage_Dynamic_Copy:
		return gl.DYNAMIC_COPY
	case BufUsage_Stream_Copy:
		return gl.STREAM_COPY
	}

	assert.T(false, fmt.Sprintf("Unexpected BufUsage value '%v'", b))
	return 0
}
