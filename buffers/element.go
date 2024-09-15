package buffers

import (
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/logging"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Element represents an element that makes up a buffer (e.g. Vec3 at an offset of 12 bytes)
type Element struct {
	Offset int
	ElementType
}

// ElementType is the type of an element thats makes up a buffer (e.g. Vec3)
type ElementType uint8

const (
	DataTypeUnknown ElementType = iota

	DataTypeUint32
	DataTypeInt32
	DataTypeFloat32

	DataTypeVec2
	DataTypeVec3
	DataTypeVec4

	DataTypeMat2
	DataTypeMat3
	DataTypeMat4

	DataTypeStruct
)

func (dt ElementType) GLType() uint32 {

	switch dt {

	case DataTypeUint32:
		return gl.UNSIGNED_INT
	case DataTypeInt32:
		return gl.INT
	case DataTypeFloat32:
		fallthrough

	case DataTypeVec2:
		fallthrough
	case DataTypeVec3:
		fallthrough
	case DataTypeVec4:
		fallthrough
	case DataTypeMat2:
		fallthrough
	case DataTypeMat3:
		fallthrough
	case DataTypeMat4:
		return gl.FLOAT

	case DataTypeStruct:
		logging.ErrLog.Fatalf("ElementType.GLType of DataTypeStruct is not supported")
		return 0

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

// CompSize returns the size in bytes for one component of the type (e.g. for Vec2 its 4).
// Bools return 1, although in layout=std140 its 4
func (dt ElementType) CompSize() int32 {

	switch dt {

	case DataTypeUint32:
		fallthrough
	case DataTypeFloat32:
		fallthrough
	case DataTypeInt32:
		fallthrough
	case DataTypeVec2:
		fallthrough
	case DataTypeVec3:
		fallthrough
	case DataTypeVec4:
		fallthrough
	case DataTypeMat2:
		fallthrough
	case DataTypeMat3:
		fallthrough
	case DataTypeMat4:
		return 4

	case DataTypeStruct:
		logging.ErrLog.Fatalf("ElementType.CompSize of DataTypeStruct is not supported")
		return 0

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

// CompCount returns the number of components in the element (e.g. for Vec2 its 2)
func (dt ElementType) CompCount() int32 {

	switch dt {
	case DataTypeUint32:
		fallthrough
	case DataTypeFloat32:
		fallthrough
	case DataTypeInt32:
		return 1

	case DataTypeVec2:
		return 2
	case DataTypeVec3:
		return 3
	case DataTypeVec4:
		return 4

	case DataTypeMat2:
		return 2 * 2
	case DataTypeMat3:
		return 3 * 3
	case DataTypeMat4:
		return 4 * 4

	case DataTypeStruct:
		logging.ErrLog.Fatalf("ElementType.CompCount of DataTypeStruct is not supported")
		return 0

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

// Size returns the total size in bytes (e.g. for vec3 its 3*4=12 bytes)
func (dt ElementType) Size() int32 {

	switch dt {

	case DataTypeUint32:
		fallthrough
	case DataTypeFloat32:
		fallthrough
	case DataTypeInt32:
		return 4

	case DataTypeVec2:
		return 2 * 4
	case DataTypeVec3:
		return 3 * 4
	case DataTypeVec4:
		return 4 * 4

	case DataTypeMat2:
		return 2 * 2 * 4
	case DataTypeMat3:
		return 3 * 3 * 4
	case DataTypeMat4:
		return 4 * 4 * 4

	case DataTypeStruct:
		logging.ErrLog.Fatalf("ElementType.Size of DataTypeStruct is not supported")
		return 0

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

func (dt ElementType) GlStd140SizeBytes() uint8 {

	switch dt {

	case DataTypeUint32:
		fallthrough
	case DataTypeFloat32:
		fallthrough
	case DataTypeInt32:
		return 4

	case DataTypeVec2:
		return 4 * 2

	case DataTypeVec3:
		return 4 * 3

	case DataTypeVec4:
		return 4 * 4

		// Matrices follow: (vec4Alignment) * numColumns
	case DataTypeMat2:
		return 2 * 2 * 4
	case DataTypeMat3:
		return 3 * 3 * 4
	case DataTypeMat4:
		return 4 * 4 * 4

	case DataTypeStruct:
		logging.ErrLog.Fatalf("ElementType.GlStd140SizeBytes of DataTypeStruct is not supported")
		return 0

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

func (dt ElementType) GlStd140AlignmentBoundary() uint16 {

	switch dt {

	case DataTypeUint32:
		fallthrough
	case DataTypeFloat32:
		fallthrough
	case DataTypeInt32:
		return 4

	case DataTypeVec2:
		return 8

	case DataTypeVec3:
		fallthrough
	case DataTypeVec4:
		fallthrough
	case DataTypeMat2:
		fallthrough
	case DataTypeMat3:
		fallthrough
	case DataTypeMat4:
		fallthrough
	case DataTypeStruct:
		return 16

	default:
		assert.T(false, "Unknown data type passed. DataType '%d'", dt)
		return 0
	}
}

func (dt ElementType) String() string {

	switch dt {

	case DataTypeUint32:
		return "uint32"
	case DataTypeFloat32:
		return "float32"
	case DataTypeInt32:
		return "int32"

	case DataTypeVec2:
		return "Vec2"
	case DataTypeVec3:
		return "Vec3"
	case DataTypeVec4:
		return "Vec4"

	case DataTypeMat2:
		return "Mat2"
	case DataTypeMat3:
		return "Mat3"
	case DataTypeMat4:
		return "Mat4"

	case DataTypeStruct:
		return "Struct"

	default:
		return "Unknown"
	}
}
