package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

type NormalizedByte2 struct{ packedValue uint16 }

func NewNormalizedByte2BySingleAndSingle(x, y float32) NormalizedByte2 {
	return NormalizedByte2{packedValue: packNormalizedByte2(x, y)}
}
func NewNormalizedByte2ByVector2(vector framework.Vector2) NormalizedByte2 {
	return NormalizedByte2{packedValue: packNormalizedByte2(vector.X, vector.Y)}
}
func packNormalizedByte2(x, y float32) uint16 {
	return uint16(packSNorm(0xff, x) | packSNorm(0xff, y)<<8)
}
func (v NormalizedByte2) ToVector2() framework.Vector2 {
	return framework.Vector2{
		X: unpackSNorm(0xff, uint32(v.packedValue)),
		Y: unpackSNorm(0xff, uint32(v.packedValue)>>8),
	}
}
func (v NormalizedByte2) ToVector4() framework.Vector4 {
	vector := v.ToVector2()
	return framework.Vector4{X: vector.X, Y: vector.Y, W: 1}
}
func (v *NormalizedByte2) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packNormalizedByte2(vector.X, vector.Y)
}
func (v NormalizedByte2) PackedValue() uint16          { return v.packedValue }
func (v *NormalizedByte2) SetPackedValue(value uint16) { v.packedValue = value }
func (v NormalizedByte2) ToString() string             { return formatPacked(uint64(v.packedValue), 4) }
func (v NormalizedByte2) GetHashCode() int32           { return int32(v.packedValue) }
func (v NormalizedByte2) EqualsByObject(value any) bool {
	other, ok := value.(NormalizedByte2)
	return ok && v.EqualsByNormalizedByte2(other)
}
func (v NormalizedByte2) EqualsByNormalizedByte2(other NormalizedByte2) bool {
	return v.packedValue == other.packedValue
}
func NormalizedByte2OperatorEqualityByNormalizedByte2AndNormalizedByte2(left, right NormalizedByte2) bool {
	return left.EqualsByNormalizedByte2(right)
}
func NormalizedByte2OperatorInequalityByNormalizedByte2AndNormalizedByte2(left, right NormalizedByte2) bool {
	return !left.EqualsByNormalizedByte2(right)
}

type NormalizedByte4 struct{ packedValue uint32 }

func NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) NormalizedByte4 {
	return NormalizedByte4{packedValue: packNormalizedByte4(x, y, z, w)}
}
func NewNormalizedByte4ByVector4(vector framework.Vector4) NormalizedByte4 {
	return NormalizedByte4{packedValue: packNormalizedByte4(vector.X, vector.Y, vector.Z, vector.W)}
}
func packNormalizedByte4(x, y, z, w float32) uint32 {
	return packSNorm(0xff, x) |
		packSNorm(0xff, y)<<8 |
		packSNorm(0xff, z)<<16 |
		packSNorm(0xff, w)<<24
}
func (v NormalizedByte4) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackSNorm(0xff, v.packedValue),
		Y: unpackSNorm(0xff, v.packedValue>>8),
		Z: unpackSNorm(0xff, v.packedValue>>16),
		W: unpackSNorm(0xff, v.packedValue>>24),
	}
}
func (v *NormalizedByte4) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packNormalizedByte4(vector.X, vector.Y, vector.Z, vector.W)
}
func (v NormalizedByte4) PackedValue() uint32          { return v.packedValue }
func (v *NormalizedByte4) SetPackedValue(value uint32) { v.packedValue = value }
func (v NormalizedByte4) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v NormalizedByte4) GetHashCode() int32           { return int32(v.packedValue) }
func (v NormalizedByte4) EqualsByObject(value any) bool {
	other, ok := value.(NormalizedByte4)
	return ok && v.EqualsByNormalizedByte4(other)
}
func (v NormalizedByte4) EqualsByNormalizedByte4(other NormalizedByte4) bool {
	return v.packedValue == other.packedValue
}
func NormalizedByte4OperatorEqualityByNormalizedByte4AndNormalizedByte4(left, right NormalizedByte4) bool {
	return left.EqualsByNormalizedByte4(right)
}
func NormalizedByte4OperatorInequalityByNormalizedByte4AndNormalizedByte4(left, right NormalizedByte4) bool {
	return !left.EqualsByNormalizedByte4(right)
}

type NormalizedShort2 struct{ packedValue uint32 }

func NewNormalizedShort2BySingleAndSingle(x, y float32) NormalizedShort2 {
	return NormalizedShort2{packedValue: packNormalizedShort2(x, y)}
}
func NewNormalizedShort2ByVector2(vector framework.Vector2) NormalizedShort2 {
	return NormalizedShort2{packedValue: packNormalizedShort2(vector.X, vector.Y)}
}
func packNormalizedShort2(x, y float32) uint32 {
	return packSNorm(0xffff, x) | packSNorm(0xffff, y)<<16
}
func (v NormalizedShort2) ToVector2() framework.Vector2 {
	return framework.Vector2{
		X: unpackSNorm(0xffff, v.packedValue),
		Y: unpackSNorm(0xffff, v.packedValue>>16),
	}
}
func (v NormalizedShort2) ToVector4() framework.Vector4 {
	vector := v.ToVector2()
	return framework.Vector4{X: vector.X, Y: vector.Y, W: 1}
}
func (v *NormalizedShort2) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packNormalizedShort2(vector.X, vector.Y)
}
func (v NormalizedShort2) PackedValue() uint32          { return v.packedValue }
func (v *NormalizedShort2) SetPackedValue(value uint32) { v.packedValue = value }
func (v NormalizedShort2) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v NormalizedShort2) GetHashCode() int32           { return int32(v.packedValue) }
func (v NormalizedShort2) EqualsByObject(value any) bool {
	other, ok := value.(NormalizedShort2)
	return ok && v.EqualsByNormalizedShort2(other)
}
func (v NormalizedShort2) EqualsByNormalizedShort2(other NormalizedShort2) bool {
	return v.packedValue == other.packedValue
}
func NormalizedShort2OperatorEqualityByNormalizedShort2AndNormalizedShort2(left, right NormalizedShort2) bool {
	return left.EqualsByNormalizedShort2(right)
}
func NormalizedShort2OperatorInequalityByNormalizedShort2AndNormalizedShort2(left, right NormalizedShort2) bool {
	return !left.EqualsByNormalizedShort2(right)
}

type NormalizedShort4 struct{ packedValue uint64 }

func NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) NormalizedShort4 {
	return NormalizedShort4{packedValue: packNormalizedShort4(x, y, z, w)}
}
func NewNormalizedShort4ByVector4(vector framework.Vector4) NormalizedShort4 {
	return NormalizedShort4{packedValue: packNormalizedShort4(vector.X, vector.Y, vector.Z, vector.W)}
}
func packNormalizedShort4(x, y, z, w float32) uint64 {
	return uint64(packSNorm(0xffff, x)) |
		uint64(packSNorm(0xffff, y))<<16 |
		uint64(packSNorm(0xffff, z))<<32 |
		uint64(packSNorm(0xffff, w))<<48
}
func (v NormalizedShort4) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackSNorm(0xffff, uint32(v.packedValue)),
		Y: unpackSNorm(0xffff, uint32(v.packedValue>>16)),
		Z: unpackSNorm(0xffff, uint32(v.packedValue>>32)),
		W: unpackSNorm(0xffff, uint32(v.packedValue>>48)),
	}
}
func (v *NormalizedShort4) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packNormalizedShort4(vector.X, vector.Y, vector.Z, vector.W)
}
func (v NormalizedShort4) PackedValue() uint64          { return v.packedValue }
func (v *NormalizedShort4) SetPackedValue(value uint64) { v.packedValue = value }
func (v NormalizedShort4) ToString() string             { return formatPacked(v.packedValue, 16) }
func (v NormalizedShort4) GetHashCode() int32           { return hashUint64(v.packedValue) }
func (v NormalizedShort4) EqualsByObject(value any) bool {
	other, ok := value.(NormalizedShort4)
	return ok && v.EqualsByNormalizedShort4(other)
}
func (v NormalizedShort4) EqualsByNormalizedShort4(other NormalizedShort4) bool {
	return v.packedValue == other.packedValue
}
func NormalizedShort4OperatorEqualityByNormalizedShort4AndNormalizedShort4(left, right NormalizedShort4) bool {
	return left.EqualsByNormalizedShort4(right)
}
func NormalizedShort4OperatorInequalityByNormalizedShort4AndNormalizedShort4(left, right NormalizedShort4) bool {
	return !left.EqualsByNormalizedShort4(right)
}
