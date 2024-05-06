package materials

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/logging"
	"github.com/bloeys/nmage/shaders"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type TextureSlot uint32

const (
	TextureSlot_Diffuse          TextureSlot = 0
	TextureSlot_Specular         TextureSlot = 1
	TextureSlot_Normal           TextureSlot = 2
	TextureSlot_Emission         TextureSlot = 3
	TextureSlot_Cubemap          TextureSlot = 10
	TextureSlot_Cubemap_Array    TextureSlot = 11
	TextureSlot_ShadowMap1       TextureSlot = 12
	TextureSlot_ShadowMap_Array1 TextureSlot = 13
)

type MaterialSettings uint64

const (
	MaterialSettings_None        MaterialSettings = iota
	MaterialSettings_HasModelMat MaterialSettings = 1 << (iota - 1)
	MaterialSettings_HasNormalMat
)

func (ms *MaterialSettings) Set(flags MaterialSettings) {
	*ms |= flags
}

func (ms *MaterialSettings) Remove(flags MaterialSettings) {
	*ms &= ^flags
}

func (ms *MaterialSettings) Has(flags MaterialSettings) bool {
	return *ms&flags == flags
}

type Material struct {
	Name       string
	ShaderProg shaders.ShaderProgram
	Settings   MaterialSettings

	UnifLocs   map[string]int32
	AttribLocs map[string]int32

	// @TODO do this in a better way. Perhaps something like how we do fbo attachments
	// Phong shading
	DiffuseTex  uint32
	SpecularTex uint32
	NormalTex   uint32
	EmissionTex uint32

	// Shininess of specular highlights
	Shininess float32

	// Cubemaps
	CubemapTex      uint32
	CubemapArrayTex uint32

	// Shadowmaps
	ShadowMapTex1      uint32
	ShadowMapTexArray1 uint32
}

func (m *Material) Bind() {

	m.ShaderProg.Bind()

	if m.DiffuseTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Diffuse))
		gl.BindTexture(gl.TEXTURE_2D, m.DiffuseTex)
	}

	if m.SpecularTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Specular))
		gl.BindTexture(gl.TEXTURE_2D, m.SpecularTex)
	}

	if m.NormalTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Normal))
		gl.BindTexture(gl.TEXTURE_2D, m.NormalTex)
	}

	if m.EmissionTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Emission))
		gl.BindTexture(gl.TEXTURE_2D, m.EmissionTex)
	}

	if m.CubemapTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Cubemap))
		gl.BindTexture(gl.TEXTURE_CUBE_MAP, m.CubemapTex)
	}

	if m.CubemapArrayTex != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Cubemap_Array))
		gl.BindTexture(gl.TEXTURE_CUBE_MAP_ARRAY, m.CubemapArrayTex)
	}

	if m.ShadowMapTex1 != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_ShadowMap1))
		gl.BindTexture(gl.TEXTURE_2D, m.ShadowMapTex1)
	}

	if m.ShadowMapTexArray1 != 0 {
		gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_ShadowMap_Array1))
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, m.ShadowMapTexArray1)
	}
}

func (m *Material) UnBind() {
	gl.UseProgram(0)
}

func (m *Material) GetAttribLoc(attribName string) int32 {

	loc, ok := m.AttribLocs[attribName]
	if ok {
		return loc
	}

	loc = gl.GetAttribLocation(m.ShaderProg.Id, gl.Str(attribName+"\x00"))
	assert.T(loc != -1, "Attribute '"+attribName+"' doesn't exist on material "+m.Name)
	m.AttribLocs[attribName] = loc
	return loc
}

func (m *Material) GetUnifLoc(uniformName string) int32 {

	loc, ok := m.UnifLocs[uniformName]
	if ok {
		return loc
	}

	loc = gl.GetUniformLocation(m.ShaderProg.Id, gl.Str(uniformName+"\x00"))
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
	gl.ProgramUniform1i(m.ShaderProg.Id, m.GetUnifLoc(uniformName), val)
}

func (m *Material) SetUnifFloat32(uniformName string, val float32) {
	gl.ProgramUniform1f(m.ShaderProg.Id, m.GetUnifLoc(uniformName), val)
}

func (m *Material) SetUnifVec2(uniformName string, vec2 *gglm.Vec2) {
	gl.ProgramUniform2fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, &vec2.Data[0])
}

func (m *Material) SetUnifVec3(uniformName string, vec3 *gglm.Vec3) {
	gl.ProgramUniform3fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, &vec3.Data[0])
}

func (m *Material) SetUnifVec4(uniformName string, vec4 *gglm.Vec4) {
	gl.ProgramUniform4fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, &vec4.Data[0])
}

func (m *Material) SetUnifMat2(uniformName string, mat2 *gglm.Mat2) {
	gl.ProgramUniformMatrix2fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, false, &mat2.Data[0][0])
}

func (m *Material) SetUnifMat3(uniformName string, mat3 *gglm.Mat3) {
	gl.ProgramUniformMatrix3fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, false, &mat3.Data[0][0])
}

func (m *Material) SetUnifMat4(uniformName string, mat4 *gglm.Mat4) {
	gl.ProgramUniformMatrix4fv(m.ShaderProg.Id, m.GetUnifLoc(uniformName), 1, false, &mat4.Data[0][0])
}

func (m *Material) Delete() {
	gl.DeleteProgram(m.ShaderProg.Id)
}

func NewMaterial(matName, shaderPath string) Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShader(shaderPath)
	if err != nil {
		logging.ErrLog.Fatalf("Failed to create new material '%s'. Err: %s\n", matName, err.Error())
	}

	return Material{Name: matName, ShaderProg: shdrProg, UnifLocs: make(map[string]int32), AttribLocs: make(map[string]int32)}
}

func NewMaterialSrc(matName string, shaderSrc []byte) Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShaderSrc(shaderSrc)
	if err != nil {
		logging.ErrLog.Fatalf("Failed to create new material '%s'. Err: %s\n", matName, err.Error())
	}

	return Material{Name: matName, ShaderProg: shdrProg, UnifLocs: make(map[string]int32), AttribLocs: make(map[string]int32)}
}
