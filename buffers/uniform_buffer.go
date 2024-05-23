package buffers

import (
	"github.com/bloeys/gglm/gglm"
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type UniformBufferFieldInput struct {
	Id   uint16
	Type ElementType
}

type UniformBufferField struct {
	Id            uint16
	AlignedOffset uint16
	Type          ElementType
}

type UniformBuffer struct {
	Id uint32
	// Size is the allocated memory in bytes on the GPU for this uniform buffer
	Size   uint32
	Fields []UniformBufferField
}

func (ub *UniformBuffer) Bind() {
	gl.BindBuffer(gl.UNIFORM_BUFFER, ub.Id)
}

func (ub *UniformBuffer) UnBind() {
	gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
}

func (ub *UniformBuffer) addFields(fields []UniformBufferFieldInput) (totalSize uint32) {

	if len(fields) == 0 {
		return 0
	}

	var alignedOffset uint16 = 0
	ub.Fields = make([]UniformBufferField, 0, len(fields))
	fieldIdToTypeMap := make(map[uint16]ElementType, len(fields))

	for i := 0; i < len(fields); i++ {

		f := fields[i]

		existingFieldType, ok := fieldIdToTypeMap[f.Id]
		assert.T(!ok, "Uniform buffer field id is reused within the same uniform buffer. FieldId=%d was first used on a field with type=%s and then used on a different field with type=%s\n", f.Id, existingFieldType.String(), f.Type.String())

		// To understand this take an example. Say we have a total offset of 100 and we are adding a vec4.
		// Vec4s must be aligned to a 16 byte boundary but 100 is not (100 % 16 != 0).
		//
		// To fix this, we take the alignment error which is alignErr=100 % 16=4, but this is error to the nearest
		// boundary, which is below the offset.
		//
		// To get the nearest boundary larger than the offset we can:
		// offset + (boundary - alignErr) == 100 + (16 - 4) == 112; 112 % 16 == 0, meaning its a boundary
		alignmentBoundary := f.Type.GlStd140AlignmentBoundary()
		alignmentError := alignedOffset % alignmentBoundary
		if alignmentError != 0 {
			alignedOffset += alignmentBoundary - alignmentError
		}

		newField := UniformBufferField{Id: f.Id, Type: f.Type, AlignedOffset: alignedOffset}
		ub.Fields = append(ub.Fields, newField)

		// Prepare aligned offset for the next field
		alignedOffset = newField.AlignedOffset + uint16(f.Type.GlStd140BaseAlignment())
	}

	return uint32(alignedOffset)
}

func (ub *UniformBuffer) getField(fieldId uint16, fieldType ElementType) UniformBufferField {

	for i := 0; i < len(ub.Fields); i++ {

		f := ub.Fields[i]

		if f.Id != fieldId {
			continue
		}

		assert.T(f.Type == fieldType, "Uniform buffer field id is reused within the same uniform buffer. FieldId=%d was first used on a field with type=%v, but is now being used on a field with type=%v\n", fieldId, f.Type.String(), fieldType.String())

		return f
	}

	logging.ErrLog.Panicf("couldn't find uniform buffer field of id=%d and type=%s\n", fieldId, fieldType.String())
	return UniformBufferField{}
}

func (ub *UniformBuffer) SetInt32(fieldId uint16, val int32) {

	f := ub.getField(fieldId, DataTypeInt32)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4, gl.Ptr(&val))
}

func (ub *UniformBuffer) SetUint32(fieldId uint16, val uint32) {

	f := ub.getField(fieldId, DataTypeUint32)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4, gl.Ptr(&val))
}

func (ub *UniformBuffer) SetFloat32(fieldId uint16, val float32) {

	f := ub.getField(fieldId, DataTypeFloat32)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4, gl.Ptr(&val))
}

func (ub *UniformBuffer) SetVec2(fieldId uint16, val *gglm.Vec2) {
	f := ub.getField(fieldId, DataTypeVec2)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*2, gl.Ptr(&val.Data[0]))
}

func (ub *UniformBuffer) SetVec3(fieldId uint16, val *gglm.Vec3) {
	f := ub.getField(fieldId, DataTypeVec3)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*3, gl.Ptr(&val.Data[0]))
}

func (ub *UniformBuffer) SetVec4(fieldId uint16, val *gglm.Vec4) {
	f := ub.getField(fieldId, DataTypeVec4)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*4, gl.Ptr(&val.Data[0]))
}

func (ub *UniformBuffer) SetMat2(fieldId uint16, val *gglm.Mat2) {
	f := ub.getField(fieldId, DataTypeMat2)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*4, gl.Ptr(&val.Data[0][0]))
}

func (ub *UniformBuffer) SetMat3(fieldId uint16, val *gglm.Mat3) {
	f := ub.getField(fieldId, DataTypeMat3)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*9, gl.Ptr(&val.Data[0][0]))
}

func (ub *UniformBuffer) SetMat4(fieldId uint16, val *gglm.Mat4) {
	f := ub.getField(fieldId, DataTypeMat4)
	gl.BufferSubData(gl.UNIFORM_BUFFER, int(f.AlignedOffset), 4*16, gl.Ptr(&val.Data[0][0]))
}

func NewUniformBuffer(fields []UniformBufferFieldInput) UniformBuffer {

	ub := UniformBuffer{}

	ub.Size = ub.addFields(fields)

	gl.GenBuffers(1, &ub.Id)
	if ub.Id == 0 {
		logging.ErrLog.Panicln("Failed to create OpenGL buffer for a uniform buffer")
	}

	ub.Bind()
	gl.BufferData(gl.UNIFORM_BUFFER, int(ub.Size), gl.Ptr(nil), gl.STATIC_DRAW)

	return ub
}
