package buffers

import (
	"math"
	"reflect"

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

func (ub *UniformBuffer) SetStruct(inputStruct any) {

	if inputStruct == nil {
		logging.ErrLog.Panicf("UniformBuffer.SetStruct called with a value that is nil")
	}

	structVal := reflect.ValueOf(inputStruct)
	if structVal.Kind() != reflect.Struct {
		logging.ErrLog.Panicf("UniformBuffer.SetStruct called with a value that is not a struct. Val=%v\n", inputStruct)
	}

	if structVal.NumField() != len(ub.Fields) {
		logging.ErrLog.Panicf("struct fields must match uniform buffer fields, but uniform buffer contains %d fields, while the passed struct has %d fields\n", len(ub.Fields), structVal.NumField())
	}

	writeIndex := 0
	buf := make([]byte, ub.Size)
	for i := 0; i < len(ub.Fields); i++ {

		ubField := &ub.Fields[i]
		valField := structVal.Field(i)

		if valField.Kind() == reflect.Pointer {
			valField = valField.Elem()
		}

		typeMatches := false
		writeIndex = int(ubField.AlignedOffset)

		switch ubField.Type {

		case DataTypeUint32:
			t := valField.Type()
			typeMatches = t.Name() == "uint32"

			if typeMatches {
				Write32BitIntegerToByteBuf(buf, &writeIndex, uint32(valField.Uint()))
			}

		case DataTypeFloat32:
			t := valField.Type()
			typeMatches = t.Name() == "float32"

			if typeMatches {
				WriteF32ToByteBuf(buf, &writeIndex, float32(valField.Float()))
			}

		case DataTypeInt32:
			t := valField.Type()
			typeMatches = t.Name() == "int32"

			if typeMatches {
				Write32BitIntegerToByteBuf(buf, &writeIndex, uint32(valField.Int()))
			}

		case DataTypeVec2:

			v2, ok := valField.Interface().(gglm.Vec2)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, v2.Data[:])
			}

		case DataTypeVec3:
			v3, ok := valField.Interface().(gglm.Vec3)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, v3.Data[:])
			}

		case DataTypeVec4:
			v4, ok := valField.Interface().(gglm.Vec4)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, v4.Data[:])
			}

		case DataTypeMat2:
			m2, ok := valField.Interface().(gglm.Mat2)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, m2.Data[0][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m2.Data[1][:])
			}

		case DataTypeMat3:
			m3, ok := valField.Interface().(gglm.Mat3)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, m3.Data[0][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m3.Data[1][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m3.Data[2][:])
			}

		case DataTypeMat4:
			m4, ok := valField.Interface().(gglm.Mat4)
			typeMatches = ok

			if typeMatches {
				WriteF32SliceToByteBuf(buf, &writeIndex, m4.Data[0][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m4.Data[1][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m4.Data[2][:])
				WriteF32SliceToByteBuf(buf, &writeIndex, m4.Data[3][:])
			}

		default:
			assert.T(false, "Unknown uniform buffer data type passed. DataType '%d'", ubField.Type)
		}

		if !typeMatches {
			logging.ErrLog.Panicf("Struct field ordering and types must match uniform buffer fields, but at field index %d got UniformBufferField=%v but a struct field of %s\n", i, ubField, valField.String())
		}
	}

	if writeIndex == 0 {
		return
	}

	gl.BufferSubData(gl.UNIFORM_BUFFER, 0, writeIndex, gl.Ptr(&buf[0]))
}

func Write32BitIntegerToByteBuf[T uint32 | int32](buf []byte, startIndex *int, val T) {

	assert.T(*startIndex+4 <= len(buf), "failed to write uint32/int32 to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d", *startIndex, len(buf))

	buf[*startIndex] = byte(val)
	buf[*startIndex+1] = byte(val >> 8)
	buf[*startIndex+2] = byte(val >> 16)
	buf[*startIndex+3] = byte(val >> 24)

	*startIndex += 4
}

func WriteF32ToByteBuf(buf []byte, startIndex *int, val float32) {

	assert.T(*startIndex+4 <= len(buf), "failed to write float32 to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d", *startIndex, len(buf))

	bits := math.Float32bits(val)

	buf[*startIndex] = byte(bits)
	buf[*startIndex+1] = byte(bits >> 8)
	buf[*startIndex+2] = byte(bits >> 16)
	buf[*startIndex+3] = byte(bits >> 24)

	*startIndex += 4
}

func WriteF32SliceToByteBuf(buf []byte, startIndex *int, vals []float32) {

	assert.T(*startIndex+4 <= len(buf), "failed to write slice of float32 to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", *startIndex, len(buf), len(vals)*4)

	for i := 0; i < len(vals); i++ {

		bits := math.Float32bits(vals[i])

		buf[*startIndex] = byte(bits)
		buf[*startIndex+1] = byte(bits >> 8)
		buf[*startIndex+2] = byte(bits >> 16)
		buf[*startIndex+3] = byte(bits >> 24)

		*startIndex += 4
	}
}

func ReflectValueMatchesUniformBufferField(v reflect.Value, ubField *UniformBufferField) bool {

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	switch ubField.Type {

	case DataTypeUint32:
		t := v.Type()
		return t.Name() == "uint32"
	case DataTypeFloat32:
		t := v.Type()
		return t.Name() == "float32"
	case DataTypeInt32:
		t := v.Type()
		return t.Name() == "int32"
	case DataTypeVec2:
		_, ok := v.Interface().(gglm.Vec2)
		return ok
	case DataTypeVec3:
		_, ok := v.Interface().(gglm.Vec3)
		return ok
	case DataTypeVec4:
		_, ok := v.Interface().(gglm.Vec4)
		return ok
	case DataTypeMat2:
		_, ok := v.Interface().(gglm.Mat2)
		return ok
	case DataTypeMat3:
		_, ok := v.Interface().(gglm.Mat3)
		return ok
	case DataTypeMat4:
		_, ok := v.Interface().(gglm.Mat4)
		return ok

	default:
		assert.T(false, "Unknown uniform buffer data type passed. DataType '%d'", ubField.Type)
		return false
	}
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
