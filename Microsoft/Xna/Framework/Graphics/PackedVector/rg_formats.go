package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

type Rg32 struct{ packedValue uint32 }

func NewRg32BySingleAndSingle(x, y float32) Rg32 {
	return Rg32{packedValue: packRg32(x, y)}
}
func NewRg32ByVector2(vector framework.Vector2) Rg32 {
	return Rg32{packedValue: packRg32(vector.X, vector.Y)}
}
func packRg32(x, y float32) uint32 {
	return packUNorm(65535, x) | packUNorm(65535, y)<<16
}
func (v Rg32) ToVector2() framework.Vector2 {
	return framework.Vector2{
		X: unpackUNorm(65535, v.packedValue),
		Y: unpackUNorm(65535, v.packedValue>>16),
	}
}
func (v Rg32) ToVector4() framework.Vector4 {
	vector := v.ToVector2()
	return framework.Vector4{X: vector.X, Y: vector.Y, W: 1}
}
func (v *Rg32) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packRg32(vector.X, vector.Y)
}
func (v Rg32) PackedValue() uint32          { return v.packedValue }
func (v *Rg32) SetPackedValue(value uint32) { v.packedValue = value }
func (v Rg32) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v Rg32) GetHashCode() int32           { return int32(v.packedValue) }
func (v Rg32) EqualsByObject(value any) bool {
	other, ok := value.(Rg32)
	return ok && v.EqualsByRg32(other)
}
func (v Rg32) EqualsByRg32(other Rg32) bool { return v.packedValue == other.packedValue }
func Rg32OperatorEqualityByRg32AndRg32(left, right Rg32) bool {
	return left.EqualsByRg32(right)
}
func Rg32OperatorInequalityByRg32AndRg32(left, right Rg32) bool {
	return !left.EqualsByRg32(right)
}

type Rgba1010102 struct{ packedValue uint32 }

func NewRgba1010102BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Rgba1010102 {
	return Rgba1010102{packedValue: packRgba1010102(x, y, z, w)}
}
func NewRgba1010102ByVector4(vector framework.Vector4) Rgba1010102 {
	return Rgba1010102{packedValue: packRgba1010102(vector.X, vector.Y, vector.Z, vector.W)}
}
func packRgba1010102(x, y, z, w float32) uint32 {
	return packUNorm(1023, x) |
		packUNorm(1023, y)<<10 |
		packUNorm(1023, z)<<20 |
		packUNorm(3, w)<<30
}
func (v Rgba1010102) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackUNorm(1023, v.packedValue),
		Y: unpackUNorm(1023, v.packedValue>>10),
		Z: unpackUNorm(1023, v.packedValue>>20),
		W: unpackUNorm(3, v.packedValue>>30),
	}
}
func (v *Rgba1010102) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packRgba1010102(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Rgba1010102) PackedValue() uint32          { return v.packedValue }
func (v *Rgba1010102) SetPackedValue(value uint32) { v.packedValue = value }
func (v Rgba1010102) ToString() string             { return formatPacked(uint64(v.packedValue), 8) }
func (v Rgba1010102) GetHashCode() int32           { return int32(v.packedValue) }
func (v Rgba1010102) EqualsByObject(value any) bool {
	other, ok := value.(Rgba1010102)
	return ok && v.EqualsByRgba1010102(other)
}
func (v Rgba1010102) EqualsByRgba1010102(other Rgba1010102) bool {
	return v.packedValue == other.packedValue
}
func Rgba1010102OperatorEqualityByRgba1010102AndRgba1010102(left, right Rgba1010102) bool {
	return left.EqualsByRgba1010102(right)
}
func Rgba1010102OperatorInequalityByRgba1010102AndRgba1010102(left, right Rgba1010102) bool {
	return !left.EqualsByRgba1010102(right)
}

type Rgba64 struct{ packedValue uint64 }

func NewRgba64BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Rgba64 {
	return Rgba64{packedValue: packRgba64(x, y, z, w)}
}
func NewRgba64ByVector4(vector framework.Vector4) Rgba64 {
	return Rgba64{packedValue: packRgba64(vector.X, vector.Y, vector.Z, vector.W)}
}
func packRgba64(x, y, z, w float32) uint64 {
	return uint64(packUNorm(65535, x)) |
		uint64(packUNorm(65535, y))<<16 |
		uint64(packUNorm(65535, z))<<32 |
		uint64(packUNorm(65535, w))<<48
}
func (v Rgba64) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackUNorm(65535, uint32(v.packedValue)),
		Y: unpackUNorm(65535, uint32(v.packedValue>>16)),
		Z: unpackUNorm(65535, uint32(v.packedValue>>32)),
		W: unpackUNorm(65535, uint32(v.packedValue>>48)),
	}
}
func (v *Rgba64) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packRgba64(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Rgba64) PackedValue() uint64          { return v.packedValue }
func (v *Rgba64) SetPackedValue(value uint64) { v.packedValue = value }
func (v Rgba64) ToString() string             { return formatPacked(v.packedValue, 16) }
func (v Rgba64) GetHashCode() int32           { return hashUint64(v.packedValue) }
func (v Rgba64) EqualsByObject(value any) bool {
	other, ok := value.(Rgba64)
	return ok && v.EqualsByRgba64(other)
}
func (v Rgba64) EqualsByRgba64(other Rgba64) bool { return v.packedValue == other.packedValue }
func Rgba64OperatorEqualityByRgba64AndRgba64(left, right Rgba64) bool {
	return left.EqualsByRgba64(right)
}
func Rgba64OperatorInequalityByRgba64AndRgba64(left, right Rgba64) bool {
	return !left.EqualsByRgba64(right)
}
