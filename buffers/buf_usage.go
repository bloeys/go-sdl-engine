package buffers

import (
	"fmt"

	"github.com/bloeys/nmage/assert"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type BufUsage int

const (
	BufUsage_Unknown BufUsage = iota

	//Buffer is set only once and used many times
	BufUsage_Static
	//Buffer is changed a lot and used many times
	BufUsage_Dynamic
	//Buffer is set only once and used by the GPU at most a few times
	BufUsage_Stream
)

func (b BufUsage) ToGL() uint32 {
	switch b {
	case BufUsage_Static:
		return gl.STATIC_DRAW
	case BufUsage_Dynamic:
		return gl.DYNAMIC_DRAW
	case BufUsage_Stream:
		return gl.STREAM_DRAW
	}

	assert.T(false, fmt.Sprintf("Unexpected BufUsage value '%v'", b))
	return 0
}
