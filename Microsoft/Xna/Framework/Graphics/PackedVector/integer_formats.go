package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

type Byte4 struct{ packedValue uint32 }

func NewByte4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Byte4 {
	return Byte4{packedValue: packByte4(x, y, z, w)}
}
func NewByte4ByVector4(vector framework.Vector4) Byte4 {
	return Byte4{packedValue: packByte4(vector.X, vector.Y, vector.Z, vector.W)}
}
func packByte4(x, y, z, w float32) uint32 {
	return packUnsigned(255, x) |
		packUnsigned(255, y)<<8 |
		packUnsigned(255, z)<<16 |
		packUnsigned(255, w)<<24
}
func (v Byte4) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: float32(v.packedValue & 0xff),
		Y: float32((v.packedValue >> 8) & 0xff),
		Z: float32((v.packedValue >> 16) & 0xff),
		W: float32((v.packedValue >> 24) & 0xff),
	}
}
func (v *Byte4) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packByte4(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Byte4) PackedValue() uint32          { return v.packedValue }
func (v *Byte4) SetPackedValue(value uint32) { v.packedValue = value }
func (v Byte4) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v Byte4) GetHashCode() int32           { return int32(v.packedValue) }
func (v Byte4) EqualsByObject(value any) bool {
	other, ok := value.(Byte4)
	return ok && v.EqualsByByte4(other)
}
func (v Byte4) EqualsByByte4(other Byte4) bool { return v.packedValue == other.packedValue }
func Byte4OperatorEqualityByByte4AndByte4(left, right Byte4) bool {
	return left.EqualsByByte4(right)
}
func Byte4OperatorInequalityByByte4AndByte4(left, right Byte4) bool {
	return !left.EqualsByByte4(right)
}

type Short2 struct{ packedValue uint32 }

func NewShort2BySingleAndSingle(x, y float32) Short2 {
	return Short2{packedValue: packShort2(x, y)}
}
func NewShort2ByVector2(vector framework.Vector2) Short2 {
	return Short2{packedValue: packShort2(vector.X, vector.Y)}
}
func packShort2(x, y float32) uint32 {
	return packSigned(0xffff, x) | packSigned(0xffff, y)<<16
}
func (v Short2) ToVector2() framework.Vector2 {
	return framework.Vector2{
		X: float32(int16(uint16(v.packedValue))),
		Y: float32(int16(uint16(v.packedValue >> 16))),
	}
}
func (v Short2) ToVector4() framework.Vector4 {
	vector := v.ToVector2()
	return framework.Vector4{X: vector.X, Y: vector.Y, W: 1}
}
func (v *Short2) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packShort2(vector.X, vector.Y)
}
func (v Short2) PackedValue() uint32          { return v.packedValue }
func (v *Short2) SetPackedValue(value uint32) { v.packedValue = value }
func (v Short2) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v Short2) GetHashCode() int32           { return int32(v.packedValue) }
func (v Short2) EqualsByObject(value any) bool {
	other, ok := value.(Short2)
	return ok && v.EqualsByShort2(other)
}
func (v Short2) EqualsByShort2(other Short2) bool { return v.packedValue == other.packedValue }
func Short2OperatorEqualityByShort2AndShort2(left, right Short2) bool {
	return left.EqualsByShort2(right)
}
func Short2OperatorInequalityByShort2AndShort2(left, right Short2) bool {
	return !left.EqualsByShort2(right)
}

type Short4 struct{ packedValue uint64 }

func NewShort4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Short4 {
	return Short4{packedValue: packShort4(x, y, z, w)}
}
func NewShort4ByVector4(vector framework.Vector4) Short4 {
	return Short4{packedValue: packShort4(vector.X, vector.Y, vector.Z, vector.W)}
}
func packShort4(x, y, z, w float32) uint64 {
	return uint64(packSigned(0xffff, x)) |
		uint64(packSigned(0xffff, y))<<16 |
		uint64(packSigned(0xffff, z))<<32 |
		uint64(packSigned(0xffff, w))<<48
}
func (v Short4) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: float32(int16(uint16(v.packedValue))),
		Y: float32(int16(uint16(v.packedValue >> 16))),
		Z: float32(int16(uint16(v.packedValue >> 32))),
		W: float32(int16(uint16(v.packedValue >> 48))),
	}
}
func (v *Short4) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packShort4(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Short4) PackedValue() uint64          { return v.packedValue }
func (v *Short4) SetPackedValue(value uint64) { v.packedValue = value }
func (v Short4) ToString() string             { return formatPacked(v.packedValue, 16) }
func (v Short4) GetHashCode() int32           { return hashUint64(v.packedValue) }
func (v Short4) EqualsByObject(value any) bool {
	other, ok := value.(Short4)
	return ok && v.EqualsByShort4(other)
}
func (v Short4) EqualsByShort4(other Short4) bool { return v.packedValue == other.packedValue }
func Short4OperatorEqualityByShort4AndShort4(left, right Short4) bool {
	return left.EqualsByShort4(right)
}
func Short4OperatorInequalityByShort4AndShort4(left, right Short4) bool {
	return !left.EqualsByShort4(right)
}
