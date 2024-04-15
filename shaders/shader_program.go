package shaders

import (
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type ShaderProgram struct {
	Id           uint32
	VertShaderId uint32
	FragShaderId uint32
	GeomShaderId uint32
}

func (sp *ShaderProgram) AttachShader(shader Shader) {

	gl.AttachShader(sp.Id, shader.Id)
	switch shader.Type {
	case ShaderType_Vertex:
		sp.VertShaderId = shader.Id
	case ShaderType_Fragment:
		sp.FragShaderId = shader.Id
	case ShaderType_Geometry:
		sp.GeomShaderId = shader.Id
	default:
		logging.ErrLog.Fatalf("Unknown shader type '%d' for shader id '%d'\n", shader.Type, shader.Id)
	}
}

func (sp *ShaderProgram) Link() {

	gl.LinkProgram(sp.Id)

	if sp.VertShaderId != 0 {
		gl.DeleteShader(sp.VertShaderId)
	}

	if sp.FragShaderId != 0 {
		gl.DeleteShader(sp.FragShaderId)
	}

	if sp.GeomShaderId != 0 {
		gl.DeleteShader(sp.GeomShaderId)
	}
}

func (s *ShaderProgram) Bind() {
	gl.UseProgram(s.Id)
}

func (s *ShaderProgram) UnBind() {
	gl.UseProgram(0)
}
