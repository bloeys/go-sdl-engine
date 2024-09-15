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
	// Count should be set in case this field is an array of type `[Count]Type`.
	// Count=0 is valid and is equivalent to Count=1, which means the type is NOT an array, but a single field.
	Count uint16

	// Subfields is used when type is a struct, in which case it holds the fields of the struct.
	// Ids do not have to be unique across structs.
	Subfields []UniformBufferFieldInput
}

type UniformBufferField struct {
	Id            uint16
	AlignedOffset uint16
	// Count should be set in case this field is an array of type `[Count]Type`.
	// Count=0 is valid and is equivalent to Count=1, which means the type is NOT an array, but a single field.
	Count uint16
	Type  ElementType

	// Subfields is used when type is a struct, in which case it holds the fields of the struct.
	// Ids do not have to be unique across structs.
	Subfields []UniformBufferField
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

func (ub *UniformBuffer) SetBindPoint(bindPointIndex uint32) {
	gl.BindBufferBase(gl.UNIFORM_BUFFER, bindPointIndex, ub.Id)
}

func addUniformBufferFieldsToArray(startAlignedOffset uint16, arrayToAddTo *[]UniformBufferField, fieldsToAdd []UniformBufferFieldInput) (totalSize uint32) {

	if len(fieldsToAdd) == 0 {
		return 0
	}

	// This function is recursive so only size the array once
	if cap(*arrayToAddTo) == 0 {
		*arrayToAddTo = make([]UniformBufferField, 0, len(fieldsToAdd))
	}

	var alignedOffset uint16 = 0
	fieldIdToTypeMap := make(map[uint16]ElementType, len(fieldsToAdd))

	for i := 0; i < len(fieldsToAdd); i++ {

		f := fieldsToAdd[i]
		if f.Count == 0 {
			f.Count = 1
		}

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
		//
		// Note that arrays of scalars/vectors are always aligned to 16 bytes, like a vec4
		var alignmentBoundary uint16 = 16
		if f.Count == 1 {
			alignmentBoundary = f.Type.GlStd140AlignmentBoundary()
		}

		alignmentError := alignedOffset % alignmentBoundary
		if alignmentError != 0 {
			alignedOffset += alignmentBoundary - alignmentError
		}

		newField := UniformBufferField{Id: f.Id, Type: f.Type, AlignedOffset: startAlignedOffset + alignedOffset, Count: f.Count}
		*arrayToAddTo = append(*arrayToAddTo, newField)

		// Prepare aligned offset for the next field.
		//
		// Matrices are treated as an array of column vectors, where each column is a vec4,
		// that's why we have a multiplier depending on how many columns we have when calculating
		// the offset
		multiplier := uint16(1)
		if f.Type == DataTypeMat2 {
			multiplier = 2
		} else if f.Type == DataTypeMat3 {
			multiplier = 3
		} else if f.Type == DataTypeMat4 {
			multiplier = 4
		}

		if f.Type == DataTypeStruct {

			subfieldsAlignedOffset := uint16(addUniformBufferFieldsToArray(startAlignedOffset+alignedOffset, arrayToAddTo, f.Subfields))

			// Pad structs to 16 byte boundary
			padTo16Boundary(&subfieldsAlignedOffset)
			alignedOffset += subfieldsAlignedOffset * f.Count

		} else {
			alignedOffset = newField.AlignedOffset + alignmentBoundary*f.Count*multiplier - startAlignedOffset
		}
	}

	return uint32(alignedOffset)
}

func padTo16Boundary[T uint16 | int | int32](val *T) {
	alignmentError := *val % 16
	if alignmentError != 0 {
		*val += 16 - alignmentError
	}
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
	setStruct(ub.Fields, make([]byte, ub.Size), inputStruct, 1000_000, false, 0)
}

func setStruct(fields []UniformBufferField, buf []byte, inputStruct any, maxFieldsToConsume int, onlyBufWrite bool, writeOffset int) (bytesWritten, fieldsConsumed int) {

	if len(fields) == 0 {
		return
	}

	if inputStruct == nil {
		logging.ErrLog.Panicf("UniformBuffer.SetStruct called with a value that is nil")
	}

	structVal := reflect.ValueOf(inputStruct)
	if structVal.Kind() != reflect.Struct {
		logging.ErrLog.Panicf("UniformBuffer.SetStruct called with a value that is not a struct. Val=%v\n", inputStruct)
	}

	structFieldIndex := 0
	// structFieldCount := structVal.NumField()
	for fieldIndex := 0; fieldIndex < len(fields) && fieldIndex < maxFieldsToConsume; fieldIndex++ {

		ubField := &fields[fieldIndex]
		valField := structVal.Field(structFieldIndex)

		fieldsConsumed++
		structFieldIndex++

		kind := valField.Kind()
		if kind == reflect.Pointer {
			valField = valField.Elem()
		}

		var elementType reflect.Type
		isArray := kind == reflect.Slice || kind == reflect.Array
		if isArray {
			elementType = valField.Type().Elem()
			kind = elementType.Kind()
		} else {
			elementType = valField.Type()
		}

		if isArray {
			assert.T(valField.Len() == int(ubField.Count), "ubo field of id=%d is an array/slice field of length=%d but got input of length=%d\n", ubField.Id, ubField.Count, valField.Len())
		}

		typeMatches := false
		bytesWritten = int(ubField.AlignedOffset) + writeOffset

		switch ubField.Type {

		case DataTypeUint32:

			typeMatches = elementType.Name() == "uint32"
			if typeMatches {

				if isArray {
					Write32BitIntegerSliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]uint32))
				} else {
					Write32BitIntegerToByteBuf(buf, &bytesWritten, uint32(valField.Uint()))
				}
			}

		case DataTypeFloat32:

			typeMatches = elementType.Name() == "float32"
			if typeMatches {

				if isArray {
					WriteF32SliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]float32))
				} else {
					WriteF32ToByteBuf(buf, &bytesWritten, float32(valField.Float()))
				}
			}

		case DataTypeInt32:

			typeMatches = elementType.Name() == "int32"
			if typeMatches {

				if isArray {
					Write32BitIntegerSliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]int32))
				} else {
					Write32BitIntegerToByteBuf(buf, &bytesWritten, uint32(valField.Int()))
				}
			}

		case DataTypeVec2:

			typeMatches = elementType.Name() == "Vec2"

			if typeMatches {

				if isArray {
					WriteVec2SliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]gglm.Vec2))
				} else {
					v2 := valField.Interface().(gglm.Vec2)
					WriteF32SliceToByteBuf(buf, &bytesWritten, v2.Data[:])
				}
			}

		case DataTypeVec3:

			typeMatches = elementType.Name() == "Vec3"

			if typeMatches {

				if isArray {
					WriteVec3SliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]gglm.Vec3))
				} else {
					v3 := valField.Interface().(gglm.Vec3)
					WriteF32SliceToByteBuf(buf, &bytesWritten, v3.Data[:])
				}
			}

		case DataTypeVec4:

			typeMatches = elementType.Name() == "Vec4"

			if typeMatches {

				if isArray {
					WriteVec4SliceToByteBufWithAlignment(buf, &bytesWritten, 16, valField.Slice(0, valField.Len()).Interface().([]gglm.Vec4))
				} else {
					v3 := valField.Interface().(gglm.Vec4)
					WriteF32SliceToByteBuf(buf, &bytesWritten, v3.Data[:])
				}
			}

		case DataTypeMat2:

			typeMatches = elementType.Name() == "Mat2"

			if typeMatches {

				if isArray {
					m2Arr := valField.Interface().([]gglm.Mat2)
					WriteMat2SliceToByteBufWithAlignment(buf, &bytesWritten, 16*2, m2Arr)
				} else {
					m := valField.Interface().(gglm.Mat2)
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[0][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[1][:])
				}
			}

		case DataTypeMat3:

			typeMatches = elementType.Name() == "Mat3"

			if typeMatches {

				if isArray {
					m3Arr := valField.Interface().([]gglm.Mat3)
					WriteMat3SliceToByteBufWithAlignment(buf, &bytesWritten, 16*3, m3Arr)
				} else {
					m := valField.Interface().(gglm.Mat3)
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[0][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[1][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[2][:])
				}
			}

		case DataTypeMat4:

			typeMatches = elementType.Name() == "Mat4"

			if typeMatches {

				if isArray {
					m4Arr := valField.Interface().([]gglm.Mat4)
					WriteMat4SliceToByteBufWithAlignment(buf, &bytesWritten, 16*4, m4Arr)
				} else {
					m := valField.Interface().(gglm.Mat4)
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[0][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[1][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[2][:])
					WriteF32SliceToByteBuf(buf, &bytesWritten, m.Data[3][:])
				}
			}

		case DataTypeStruct:

			typeMatches = kind == reflect.Struct

			if typeMatches {

				if isArray {

					offset := 0
					arrSize := valField.Len()
					fieldsToUse := fields[fieldIndex+1:]
					for i := 0; i < arrSize; i++ {

						setStructBytesWritten, setStructFieldsConsumed := setStruct(fieldsToUse, buf, valField.Index(i).Interface(), elementType.NumField(), true, offset*i)

						if offset == 0 {
							offset = setStructBytesWritten
							padTo16Boundary(&offset)

							bytesWritten += offset * arrSize

							// Tracking consumed fields is needed because if we have a struct inside another struct
							// elementType.NumField() will only give us the fields consumed by the first struct,
							// but we need to count all fields of all nested structs inside this one
							fieldIndex += setStructFieldsConsumed
							fieldsConsumed += setStructFieldsConsumed
						}
					}

				} else {

					setStructBytesWritten, setStructFieldsConsumed := setStruct(fields[fieldIndex+1:], buf, valField.Interface(), valField.NumField(), true, writeOffset)

					bytesWritten += setStructBytesWritten
					fieldIndex += setStructFieldsConsumed
					fieldsConsumed += setStructFieldsConsumed
				}
			}

		default:
			assert.T(false, "Unknown uniform buffer data type passed. DataType '%d'", ubField.Type)
		}

		if !typeMatches {
			logging.ErrLog.Panicf("Struct field ordering and types must match uniform buffer fields, but at field index %d got UniformBufferField=%v but a struct field of type %s\n", fieldIndex, ubField, valField.String())
		}
	}

	if bytesWritten == 0 {
		return 0, fieldsConsumed
	}

	if !onlyBufWrite {
		gl.BufferSubData(gl.UNIFORM_BUFFER, 0, bytesWritten, gl.Ptr(&buf[0]))
	}

	return bytesWritten - int(fields[0].AlignedOffset) - writeOffset, fieldsConsumed
}

func Write32BitIntegerToByteBuf[T uint32 | int32](buf []byte, startIndex *int, val T) {

	assert.T(*startIndex+4 <= len(buf), "failed to write uint32/int32 to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d", *startIndex, len(buf))

	buf[*startIndex] = byte(val)
	buf[*startIndex+1] = byte(val >> 8)
	buf[*startIndex+2] = byte(val >> 16)
	buf[*startIndex+3] = byte(val >> 24)

	*startIndex += 4
}

func Write32BitIntegerSliceToByteBufWithAlignment[T uint32 | int32](buf []byte, startIndex *int, alignmentPerField int, vals []T) {

	assert.T(*startIndex+len(vals)*alignmentPerField <= len(buf), "failed to write uint32/int32 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerField, *startIndex, len(buf), len(vals)*alignmentPerField)

	for i := 0; i < len(vals); i++ {

		val := vals[i]

		buf[*startIndex] = byte(val)
		buf[*startIndex+1] = byte(val >> 8)
		buf[*startIndex+2] = byte(val >> 16)
		buf[*startIndex+3] = byte(val >> 24)

		*startIndex += alignmentPerField
	}
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

	assert.T(*startIndex+len(vals)*4 <= len(buf), "failed to write slice of float32 to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", *startIndex, len(buf), len(vals)*4)

	for i := 0; i < len(vals); i++ {

		bits := math.Float32bits(vals[i])

		buf[*startIndex] = byte(bits)
		buf[*startIndex+1] = byte(bits >> 8)
		buf[*startIndex+2] = byte(bits >> 16)
		buf[*startIndex+3] = byte(bits >> 24)

		*startIndex += 4
	}
}

func WriteF32SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerField int, vals []float32) {

	assert.T(*startIndex+len(vals)*alignmentPerField <= len(buf), "failed to write slice of float32 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerField, *startIndex, len(buf), len(vals)*alignmentPerField)

	for i := 0; i < len(vals); i++ {

		bits := math.Float32bits(vals[i])

		buf[*startIndex] = byte(bits)
		buf[*startIndex+1] = byte(bits >> 8)
		buf[*startIndex+2] = byte(bits >> 16)
		buf[*startIndex+3] = byte(bits >> 24)

		*startIndex += alignmentPerField
	}
}

func WriteVec2SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerVector int, vals []gglm.Vec2) {

	assert.T(*startIndex+len(vals)*alignmentPerVector <= len(buf), "failed to write slice of gglm.Vec2 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerVector, *startIndex, len(buf), len(vals)*alignmentPerVector)

	for i := 0; i < len(vals); i++ {

		bitsX := math.Float32bits(vals[i].X())
		bitsY := math.Float32bits(vals[i].Y())

		buf[*startIndex] = byte(bitsX)
		buf[*startIndex+1] = byte(bitsX >> 8)
		buf[*startIndex+2] = byte(bitsX >> 16)
		buf[*startIndex+3] = byte(bitsX >> 24)

		buf[*startIndex+4] = byte(bitsY)
		buf[*startIndex+5] = byte(bitsY >> 8)
		buf[*startIndex+6] = byte(bitsY >> 16)
		buf[*startIndex+7] = byte(bitsY >> 24)

		*startIndex += alignmentPerVector
	}
}

func WriteVec3SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerVector int, vals []gglm.Vec3) {

	assert.T(*startIndex+len(vals)*alignmentPerVector <= len(buf), "failed to write slice of gglm.Vec3 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerVector, *startIndex, len(buf), len(vals)*alignmentPerVector)

	for i := 0; i < len(vals); i++ {

		bitsX := math.Float32bits(vals[i].X())
		bitsY := math.Float32bits(vals[i].Y())
		bitsZ := math.Float32bits(vals[i].Z())

		buf[*startIndex] = byte(bitsX)
		buf[*startIndex+1] = byte(bitsX >> 8)
		buf[*startIndex+2] = byte(bitsX >> 16)
		buf[*startIndex+3] = byte(bitsX >> 24)

		buf[*startIndex+4] = byte(bitsY)
		buf[*startIndex+5] = byte(bitsY >> 8)
		buf[*startIndex+6] = byte(bitsY >> 16)
		buf[*startIndex+7] = byte(bitsY >> 24)

		buf[*startIndex+8] = byte(bitsZ)
		buf[*startIndex+9] = byte(bitsZ >> 8)
		buf[*startIndex+10] = byte(bitsZ >> 16)
		buf[*startIndex+11] = byte(bitsZ >> 24)

		*startIndex += alignmentPerVector
	}
}

func WriteVec4SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerVector int, vals []gglm.Vec4) {

	assert.T(*startIndex+len(vals)*alignmentPerVector <= len(buf), "failed to write slice of gglm.Vec4 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerVector, *startIndex, len(buf), len(vals)*alignmentPerVector)

	for i := 0; i < len(vals); i++ {

		bitsX := math.Float32bits(vals[i].X())
		bitsY := math.Float32bits(vals[i].Y())
		bitsZ := math.Float32bits(vals[i].Z())
		bitsW := math.Float32bits(vals[i].W())

		buf[*startIndex] = byte(bitsX)
		buf[*startIndex+1] = byte(bitsX >> 8)
		buf[*startIndex+2] = byte(bitsX >> 16)
		buf[*startIndex+3] = byte(bitsX >> 24)

		buf[*startIndex+4] = byte(bitsY)
		buf[*startIndex+5] = byte(bitsY >> 8)
		buf[*startIndex+6] = byte(bitsY >> 16)
		buf[*startIndex+7] = byte(bitsY >> 24)

		buf[*startIndex+8] = byte(bitsZ)
		buf[*startIndex+9] = byte(bitsZ >> 8)
		buf[*startIndex+10] = byte(bitsZ >> 16)
		buf[*startIndex+11] = byte(bitsZ >> 24)

		buf[*startIndex+12] = byte(bitsW)
		buf[*startIndex+13] = byte(bitsW >> 8)
		buf[*startIndex+14] = byte(bitsW >> 16)
		buf[*startIndex+15] = byte(bitsW >> 24)

		*startIndex += alignmentPerVector
	}
}

func WriteMat2SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerMatrix int, vals []gglm.Mat2) {

	assert.T(*startIndex+len(vals)*alignmentPerMatrix <= len(buf), "failed to write slice of gglm.Mat2 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerMatrix, *startIndex, len(buf), len(vals)*alignmentPerMatrix)

	for i := 0; i < len(vals); i++ {

		m := &vals[i]

		WriteVec2SliceToByteBufWithAlignment(
			buf,
			startIndex,
			16,
			[]gglm.Vec2{
				{Data: m.Data[0]},
				{Data: m.Data[1]},
			},
		)
	}
}

func WriteMat3SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerMatrix int, vals []gglm.Mat3) {

	assert.T(*startIndex+len(vals)*alignmentPerMatrix <= len(buf), "failed to write slice of gglm.Mat3 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerMatrix, *startIndex, len(buf), len(vals)*alignmentPerMatrix)

	for i := 0; i < len(vals); i++ {

		m := &vals[i]

		WriteVec3SliceToByteBufWithAlignment(
			buf,
			startIndex,
			16,
			[]gglm.Vec3{
				{Data: m.Data[0]},
				{Data: m.Data[1]},
				{Data: m.Data[2]},
			},
		)
	}
}

func WriteMat4SliceToByteBufWithAlignment(buf []byte, startIndex *int, alignmentPerMatrix int, vals []gglm.Mat4) {

	assert.T(*startIndex+len(vals)*alignmentPerMatrix <= len(buf), "failed to write slice of gglm.Mat2 with custom alignment=%d to buffer because the buffer doesn't have enough space. Start index=%d, Buffer length=%d, but needs %d bytes free", alignmentPerMatrix, *startIndex, len(buf), len(vals)*alignmentPerMatrix)

	for i := 0; i < len(vals); i++ {

		m := &vals[i]

		WriteVec4SliceToByteBufWithAlignment(
			buf,
			startIndex,
			16,
			[]gglm.Vec4{
				{Data: m.Data[0]},
				{Data: m.Data[1]},
				{Data: m.Data[2]},
				{Data: m.Data[3]},
			},
		)
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

	ub.Size = addUniformBufferFieldsToArray(0, &ub.Fields, fields)

	gl.GenBuffers(1, &ub.Id)
	if ub.Id == 0 {
		logging.ErrLog.Panicln("Failed to create OpenGL buffer for a uniform buffer")
	}

	ub.Bind()
	gl.BufferData(gl.UNIFORM_BUFFER, int(ub.Size), gl.Ptr(nil), gl.STATIC_DRAW)
	ub.UnBind()

	return ub
}
