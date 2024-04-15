package shaders

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type ShaderType int32

func (s ShaderType) ToGl() uint32 {

	switch s {
	case ShaderType_Vertex:
		return gl.VERTEX_SHADER
	case ShaderType_Fragment:
		return gl.FRAGMENT_SHADER
	case ShaderType_Geometry:
		return gl.GEOMETRY_SHADER

	default:
		logging.ErrLog.Fatalf("Unknown shader type '%d'\n", s)
		return 0
	}
}

const (
	ShaderType_Unknown ShaderType = iota
	ShaderType_Vertex
	ShaderType_Fragment
	ShaderType_Geometry
)
