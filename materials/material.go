package materials

import (
	_ "unsafe"

	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/assets"
	"github.com/bloeys/nmage/logging"
	"github.com/bloeys/nmage/shaders"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// @TODO: This noescape magic is to avoid heap allocations done when
// passing vectors or matrices into cgo via set uniform calls.
//
// But I would rather this kind of stuff is done on the gl wrapper level.
// Should we wrap the OpenGL APIs we use ourself?

var (
	lastMatId uint32
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
	MaterialSettings_HasModelMtx MaterialSettings = 1 << (iota - 1)
	MaterialSettings_HasNormalMtx
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
	Id         uint32
	Name       string
	ShaderProg shaders.ShaderProgram
	Settings   MaterialSettings

	UnifLocs   map[string]int32
	AttribLocs map[string]int32

	// @TODO: Do this in a better way?. Perhaps something like how we do fbo attachments? Or keep it?
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

	gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Diffuse))
	gl.BindTexture(gl.TEXTURE_2D, m.DiffuseTex)

	gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Specular))
	gl.BindTexture(gl.TEXTURE_2D, m.SpecularTex)

	gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Normal))
	gl.BindTexture(gl.TEXTURE_2D, m.NormalTex)

	gl.ActiveTexture(uint32(gl.TEXTURE0 + TextureSlot_Emission))
	gl.BindTexture(gl.TEXTURE_2D, m.EmissionTex)

	// @TODO: Have defaults for these
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

func (m *Material) SetUniformBlockBindingPoint(uniformBlockName string, bindPointIndex uint32) {

	nullStr := gl.Str(uniformBlockName + "\x00")
	index := gl.GetUniformBlockIndex(m.ShaderProg.Id, nullStr)
	assert.T(
		index != gl.INVALID_INDEX,
		"SetUniformBlockBindingPoint for material=%s (matId=%d; shaderId=%d) failed because the uniform block=%s wasn't found",
		m.Name,
		m.Id,
		m.ShaderProg.Id,
		uniformBlockName,
	)
	gl.UniformBlockBinding(m.ShaderProg.Id, index, bindPointIndex)
}

func (m *Material) GetAttribLoc(attribName string) int32 {

	loc, ok := m.AttribLocs[attribName]
	if ok {
		return loc
	}

	name := gl.Str(attribName + "\x00")
	loc = gl.GetAttribLocation(m.ShaderProg.Id, name)
	assert.T(loc != -1, "Attribute '"+attribName+"' doesn't exist on material "+m.Name)
	m.AttribLocs[attribName] = loc
	return loc
}

func (m *Material) GetUnifLoc(uniformName string) int32 {

	loc, ok := m.UnifLocs[uniformName]
	if ok {
		return loc
	}

	name := gl.Str(uniformName + "\x00")
	loc = gl.GetUniformLocation(m.ShaderProg.Id, name)
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
	internalSetUnifVec2(m.ShaderProg.Id, m.GetUnifLoc(uniformName), vec2)
}

//go:noescape
//go:linkname internalSetUnifVec2 github.com/bloeys/nmage/materials.SetUnifVec2
func internalSetUnifVec2(shaderProgId uint32, unifLoc int32, vec2 *gglm.Vec2)

func SetUnifVec2(shaderProgId uint32, unifLoc int32, vec2 *gglm.Vec2) {
	gl.ProgramUniform2fv(shaderProgId, unifLoc, 1, &vec2.Data[0])
}

func (m *Material) SetUnifVec3(uniformName string, vec3 *gglm.Vec3) {
	internalSetUnifVec3(m.ShaderProg.Id, m.GetUnifLoc(uniformName), vec3)
}

//go:noescape
//go:linkname internalSetUnifVec3 github.com/bloeys/nmage/materials.SetUnifVec3
func internalSetUnifVec3(shaderProgId uint32, unifLoc int32, vec3 *gglm.Vec3)

func SetUnifVec3(shaderProgId uint32, unifLoc int32, vec3 *gglm.Vec3) {
	gl.ProgramUniform3fv(shaderProgId, unifLoc, 1, &vec3.Data[0])
}

func (m *Material) SetUnifVec4(uniformName string, vec4 *gglm.Vec4) {
	internalSetUnifVec4(m.ShaderProg.Id, m.GetUnifLoc(uniformName), vec4)
}

//go:noescape
//go:linkname internalSetUnifVec4 github.com/bloeys/nmage/materials.SetUnifVec4
func internalSetUnifVec4(shaderProgId uint32, unifLoc int32, vec4 *gglm.Vec4)

func SetUnifVec4(shaderProgId uint32, unifLoc int32, vec4 *gglm.Vec4) {
	gl.ProgramUniform4fv(shaderProgId, unifLoc, 1, &vec4.Data[0])
}

func (m *Material) SetUnifMat2(uniformName string, mat2 *gglm.Mat2) {
	internalSetUnifMat2(m.ShaderProg.Id, m.GetUnifLoc(uniformName), mat2)
}

//go:noescape
//go:linkname internalSetUnifMat2 github.com/bloeys/nmage/materials.SetUnifMat2
func internalSetUnifMat2(shaderProgId uint32, unifLoc int32, mat2 *gglm.Mat2)

func SetUnifMat2(shaderProgId uint32, unifLoc int32, mat2 *gglm.Mat2) {
	gl.ProgramUniformMatrix2fv(shaderProgId, unifLoc, 1, false, &mat2.Data[0][0])
}

func (m *Material) SetUnifMat3(uniformName string, mat3 *gglm.Mat3) {
	internalSetUnifMat3(m.ShaderProg.Id, m.GetUnifLoc(uniformName), mat3)
}

//go:noescape
//go:linkname internalSetUnifMat3 github.com/bloeys/nmage/materials.SetUnifMat3
func internalSetUnifMat3(shaderProgId uint32, unifLoc int32, mat3 *gglm.Mat3)

func SetUnifMat3(shaderProgId uint32, unifLoc int32, mat3 *gglm.Mat3) {
	gl.ProgramUniformMatrix3fv(shaderProgId, unifLoc, 1, false, &mat3.Data[0][0])
}

func (m *Material) SetUnifMat4(uniformName string, mat4 *gglm.Mat4) {
	internalSetUnifMat4(m.ShaderProg.Id, m.GetUnifLoc(uniformName), mat4)
}

//go:noescape
//go:linkname internalSetUnifMat4 github.com/bloeys/nmage/materials.SetUnifMat4
func internalSetUnifMat4(shaderProgId uint32, unifLoc int32, mat4 *gglm.Mat4)

func SetUnifMat4(shaderProgId uint32, unifLoc int32, mat4 *gglm.Mat4) {
	gl.ProgramUniformMatrix4fv(shaderProgId, unifLoc, 1, false, &mat4.Data[0][0])
}

func (m *Material) Delete() {
	gl.DeleteProgram(m.ShaderProg.Id)
}

func getNewMatId() uint32 {
	lastMatId++
	return lastMatId
}

func NewMaterial(matName, shaderPath string) Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShader(shaderPath)
	if err != nil {
		logging.ErrLog.Fatalf("Failed to create new material '%s'. Err: %s\n", matName, err.Error())
	}

	return Material{
		Id:         getNewMatId(),
		Name:       matName,
		ShaderProg: shdrProg,
		UnifLocs:   make(map[string]int32),
		AttribLocs: make(map[string]int32),

		DiffuseTex:  assets.DefaultDiffuseTexId.TexID,
		SpecularTex: assets.DefaultSpecularTexId.TexID,
		NormalTex:   assets.DefaultNormalTexId.TexID,
		EmissionTex: assets.DefaultEmissionTexId.TexID,
	}
}

func NewMaterialSrc(matName string, shaderSrc []byte) Material {

	shdrProg, err := shaders.LoadAndCompileCombinedShaderSrc(shaderSrc)
	if err != nil {
		logging.ErrLog.Fatalf("Failed to create new material '%s'. Err: %s\n", matName, err.Error())
	}

	return Material{
		Id:         getNewMatId(),
		Name:       matName,
		ShaderProg: shdrProg,
		UnifLocs:   make(map[string]int32),
		AttribLocs: make(map[string]int32),

		DiffuseTex:  assets.DefaultDiffuseTexId.TexID,
		SpecularTex: assets.DefaultSpecularTexId.TexID,
		NormalTex:   assets.DefaultNormalTexId.TexID,
		EmissionTex: assets.DefaultEmissionTexId.TexID,
	}
}
