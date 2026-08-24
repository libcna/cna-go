package graphics

import (
	"fmt"
	"strconv"
)

// VertexElementFormat defines the storage format of one vertex element.
type VertexElementFormat int32

const (
	VertexElementFormatSingle           VertexElementFormat = 0
	VertexElementFormatVector2          VertexElementFormat = 1
	VertexElementFormatVector3          VertexElementFormat = 2
	VertexElementFormatVector4          VertexElementFormat = 3
	VertexElementFormatColor            VertexElementFormat = 4
	VertexElementFormatByte4            VertexElementFormat = 5
	VertexElementFormatShort2           VertexElementFormat = 6
	VertexElementFormatShort4           VertexElementFormat = 7
	VertexElementFormatNormalizedShort2 VertexElementFormat = 8
	VertexElementFormatNormalizedShort4 VertexElementFormat = 9
	VertexElementFormatHalfVector2      VertexElementFormat = 10
	VertexElementFormatHalfVector4      VertexElementFormat = 11
)

// VertexElementUsage describes how one vertex element is consumed.
type VertexElementUsage int32

const (
	VertexElementUsagePosition          VertexElementUsage = 0
	VertexElementUsageColor             VertexElementUsage = 1
	VertexElementUsageTextureCoordinate VertexElementUsage = 2
	VertexElementUsageNormal            VertexElementUsage = 3
	VertexElementUsageBinormal          VertexElementUsage = 4
	VertexElementUsageTangent           VertexElementUsage = 5
	VertexElementUsageBlendIndices      VertexElementUsage = 6
	VertexElementUsageBlendWeight       VertexElementUsage = 7
	VertexElementUsageDepth             VertexElementUsage = 8
	VertexElementUsageFog               VertexElementUsage = 9
	VertexElementUsagePointSize         VertexElementUsage = 10
	VertexElementUsageSample            VertexElementUsage = 11
	VertexElementUsageTessellateFactor  VertexElementUsage = 12
)

// VertexElement is a managed value descriptor for one item in a vertex layout.
// Its Go zero value corresponds to Single/Position at offset and usage index zero;
// it does not represent an additional XNA parameterless constructor.
type VertexElement struct {
	offset     int32
	format     VertexElementFormat
	usage      VertexElementUsage
	usageIndex int32
}

func NewVertexElement(offset int32, elementFormat VertexElementFormat, elementUsage VertexElementUsage, usageIndex int32) VertexElement {
	return VertexElement{offset: offset, format: elementFormat, usage: elementUsage, usageIndex: usageIndex}
}

func (v VertexElement) Offset() int32                            { return v.offset }
func (v *VertexElement) SetOffset(value int32)                   { v.offset = value }
func (v VertexElement) VertexElementFormat() VertexElementFormat { return v.format }
func (v *VertexElement) SetVertexElementFormat(value VertexElementFormat) {
	v.format = value
}
func (v VertexElement) VertexElementUsage() VertexElementUsage { return v.usage }
func (v *VertexElement) SetVertexElementUsage(value VertexElementUsage) {
	v.usage = value
}
func (v VertexElement) UsageIndex() int32          { return v.usageIndex }
func (v *VertexElement) SetUsageIndex(value int32) { v.usageIndex = value }

func (v VertexElement) Equals(value any) bool {
	other, ok := value.(VertexElement)
	return ok && VertexElementOperatorEqualityByVertexElementAndVertexElement(v, other)
}

func (v VertexElement) GetHashCode() int32 {
	hash := v.offset ^ int32(v.format) ^ int32(v.usage) ^ v.usageIndex
	if hash == 0 {
		return 2147483647
	}
	return hash
}

func (v VertexElement) ToString() string {
	return fmt.Sprintf(
		"{Offset:%d Format:%s Usage:%s UsageIndex:%d}",
		v.offset,
		vertexElementFormatString(v.format),
		vertexElementUsageString(v.usage),
		v.usageIndex,
	)
}

func VertexElementOperatorEqualityByVertexElementAndVertexElement(left, right VertexElement) bool {
	return left.offset == right.offset &&
		left.usageIndex == right.usageIndex &&
		left.usage == right.usage &&
		left.format == right.format
}

func VertexElementOperatorInequalityByVertexElementAndVertexElement(left, right VertexElement) bool {
	return !VertexElementOperatorEqualityByVertexElementAndVertexElement(left, right)
}

func vertexElementFormatString(value VertexElementFormat) string {
	switch value {
	case VertexElementFormatSingle:
		return "Single"
	case VertexElementFormatVector2:
		return "Vector2"
	case VertexElementFormatVector3:
		return "Vector3"
	case VertexElementFormatVector4:
		return "Vector4"
	case VertexElementFormatColor:
		return "Color"
	case VertexElementFormatByte4:
		return "Byte4"
	case VertexElementFormatShort2:
		return "Short2"
	case VertexElementFormatShort4:
		return "Short4"
	case VertexElementFormatNormalizedShort2:
		return "NormalizedShort2"
	case VertexElementFormatNormalizedShort4:
		return "NormalizedShort4"
	case VertexElementFormatHalfVector2:
		return "HalfVector2"
	case VertexElementFormatHalfVector4:
		return "HalfVector4"
	default:
		return strconv.FormatInt(int64(value), 10)
	}
}

func vertexElementUsageString(value VertexElementUsage) string {
	switch value {
	case VertexElementUsagePosition:
		return "Position"
	case VertexElementUsageColor:
		return "Color"
	case VertexElementUsageTextureCoordinate:
		return "TextureCoordinate"
	case VertexElementUsageNormal:
		return "Normal"
	case VertexElementUsageBinormal:
		return "Binormal"
	case VertexElementUsageTangent:
		return "Tangent"
	case VertexElementUsageBlendIndices:
		return "BlendIndices"
	case VertexElementUsageBlendWeight:
		return "BlendWeight"
	case VertexElementUsageDepth:
		return "Depth"
	case VertexElementUsageFog:
		return "Fog"
	case VertexElementUsagePointSize:
		return "PointSize"
	case VertexElementUsageSample:
		return "Sample"
	case VertexElementUsageTessellateFactor:
		return "TessellateFactor"
	default:
		return strconv.FormatInt(int64(value), 10)
	}
}
