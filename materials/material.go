package materials

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/logging"
	"github.com/bloeys/nmage/shaders"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type Material struct {
	Name       string
	ShaderProg shaders.ShaderProgram

	UnifLocs   map[string]int32
	AttribLocs map[string]int32

	// Phong shading
	DiffuseTex  uint32
	SpecularTex uint32
	NormalTex   uint32
	EmissionTex uint32

	Shininess float32
}

func (m *Material) Bind() {

	gl.UseProgram(m.ShaderProg.ID)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, m.DiffuseTex)

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, m.SpecularTex)

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, m.NormalTex)

	gl.ActiveTexture(gl.TEXTURE3)
	gl.BindTexture(gl.TEXTURE_2D, m.EmissionTex)
}

func (m *Material) UnBind() {
	gl.UseProgram(0)

	//TODO: Should we unbind textures here? Are these two lines needed?
	// gl.ActiveTexture(gl.TEXTURE0)
	// gl.BindTexture(gl.TEXTURE_2D, 0)

	// gl.ActiveTexture(gl.TEXTURE1)
	// gl.BindTexture(gl.TEXTURE_2D, 0)

	// gl.ActiveTexture(gl.TEXTURE2)
	// gl.BindTexture(gl.TEXTURE_2D, 0)

	// gl.ActiveTexture(gl.TEXTURE3)
	// gl.BindTexture(gl.TEXTURE_2D, 0)
}

func (m *Material) GetAttribLoc(attribName string) int32 {

	loc, ok := m.AttribLocs[attribName]
	if ok {
		return loc
	}

	loc = gl.GetAttribLocation(m.ShaderProg.ID, gl.Str(attribName+"\x00"))
	assert.T(loc != -1, "Attribute '"+attribName+"' doesn't exist on material "+m.Name)
	m.AttribLocs[attribName] = loc
	return loc
}

func (m *Material) GetUnifLoc(uniformName string) int32 {

	loc, ok := m.UnifLocs[uniformName]
	if ok {
		return loc
	}

	loc = gl.GetUniformLocation(m.ShaderProg.ID, gl.Str(uniformName+"\x00"))
	assert.T(loc != -1, "Uniform '"+uniformName+"' doesn't exist on material "+m.Name)
	m.UnifLocs[uniformName] = loc
	return loc
}

func (m *Material) EnableAttribute(attribName string) {
	gl.EnableVertexAttribArray(uint32(m.GetAttribLoc(attribName)))
}

func (m *Material) DisableAttribute(attribName string) {
	gl.DisableVertexAttribArray(uint32(m.GetAttribLoc(attribName)))
}

func (m *Material) SetUnifInt32(uniformName string, val int32) {
	gl.ProgramUniform1i(m.ShaderProg.ID, m.GetUnifLoc(uniformName), val)
}

func (m *Material) SetUnifFloat32(uniformName string, val float32) {
	gl.ProgramUniform1f(m.ShaderProg.ID, m.GetUnifLoc(uniformName), val)
}

func (m *Material) SetUnifVec2(uniformName string, vec2 *gglm.Vec2) {
	gl.ProgramUniform2fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, &vec2.Data[0])
}

func (m *Material) SetUnifVec3(uniformName string, vec3 *gglm.Vec3) {
	gl.ProgramUniform3fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, &vec3.Data[0])
}

func (m *Material) SetUnifVec4(uniformName string, vec4 *gglm.Vec4) {
	gl.ProgramUniform4fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, &vec4.Data[0])
}

func (m *Material) SetUnifMat2(uniformName string, mat2 *gglm.Mat2) {
	gl.ProgramUniformMatrix2fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, false, &mat2.Data[0][0])
}

func (m *Material) SetUnifMat3(uniformName string, mat3 *gglm.Mat3) {
	gl.ProgramUniformMatrix3fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, false, &mat3.Data[0][0])
}

func (m *Material) SetUnifMat4(uniformName string, mat4 *gglm.Mat4) {
	gl.ProgramUniformMatrix4fv(m.ShaderProg.ID, m.GetUnifLoc(uniformName), 1, false, &mat4.Data[0][0])
}

func (m *Material) Delete() {
	gl.DeleteProgram(m.ShaderProg.ID)
}

func NewMaterial(matName, shaderPath string) *Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShader(shaderPath)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new material. Err: ", err)
	}

	return &Material{Name: matName, ShaderProg: shdrProg, UnifLocs: make(map[string]int32), AttribLocs: make(map[string]int32)}
}

func NewMaterialSrc(matName string, shaderSrc []byte) *Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShaderSrc(shaderSrc)
	if err != nil {
		logging.ErrLog.Fatalln("Failed to create new material. Err: ", err)
	}

	return &Material{Name: matName, ShaderProg: shdrProg, UnifLocs: make(map[string]int32), AttribLocs: make(map[string]int32)}
}
