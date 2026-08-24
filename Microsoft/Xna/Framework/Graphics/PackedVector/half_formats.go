package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

type HalfSingle struct{ packedValue uint16 }

func NewHalfSingle(value float32) HalfSingle { return HalfSingle{packedValue: packHalf(value)} }

func (v HalfSingle) ToSingle() float32 { return unpackHalf(v.packedValue) }
func (v HalfSingle) ToVector4() framework.Vector4 {
	return framework.Vector4{X: v.ToSingle(), W: 1}
}
func (v *HalfSingle) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packHalf(vector.X)
}
func (v HalfSingle) PackedValue() uint16          { return v.packedValue }
func (v *HalfSingle) SetPackedValue(value uint16) { v.packedValue = value }
func (v HalfSingle) ToString() string             { return formatSingle(v.ToSingle()) }
func (v HalfSingle) GetHashCode() int32           { return int32(v.packedValue) }
func (v HalfSingle) EqualsByObject(value any) bool {
	other, ok := value.(HalfSingle)
	return ok && v.EqualsByHalfSingle(other)
}
func (v HalfSingle) EqualsByHalfSingle(other HalfSingle) bool {
	return v.packedValue == other.packedValue
}
func HalfSingleOperatorEqualityByHalfSingleAndHalfSingle(left, right HalfSingle) bool {
	return left.EqualsByHalfSingle(right)
}
func HalfSingleOperatorInequalityByHalfSingleAndHalfSingle(left, right HalfSingle) bool {
	return !left.EqualsByHalfSingle(right)
}

type HalfVector2 struct{ packedValue uint32 }

func NewHalfVector2BySingleAndSingle(x, y float32) HalfVector2 {
	return HalfVector2{packedValue: packHalfVector2(x, y)}
}
func NewHalfVector2ByVector2(vector framework.Vector2) HalfVector2 {
	return HalfVector2{packedValue: packHalfVector2(vector.X, vector.Y)}
}
func packHalfVector2(x, y float32) uint32 {
	return uint32(packHalf(x)) | uint32(packHalf(y))<<16
}
func (v HalfVector2) ToVector2() framework.Vector2 {
	return framework.Vector2{X: unpackHalf(uint16(v.packedValue)), Y: unpackHalf(uint16(v.packedValue >> 16))}
}
func (v HalfVector2) ToVector4() framework.Vector4 {
	vector := v.ToVector2()
	return framework.Vector4{X: vector.X, Y: vector.Y, W: 1}
}
func (v *HalfVector2) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packHalfVector2(vector.X, vector.Y)
}
func (v HalfVector2) PackedValue() uint32          { return v.packedValue }
func (v *HalfVector2) SetPackedValue(value uint32) { v.packedValue = value }
func (v HalfVector2) ToString() string             { return v.ToVector2().ToString() }
func (v HalfVector2) GetHashCode() int32           { return int32(v.packedValue) }
func (v HalfVector2) EqualsByObject(value any) bool {
	other, ok := value.(HalfVector2)
	return ok && v.EqualsByHalfVector2(other)
}
func (v HalfVector2) EqualsByHalfVector2(other HalfVector2) bool {
	return v.packedValue == other.packedValue
}
func HalfVector2OperatorEqualityByHalfVector2AndHalfVector2(left, right HalfVector2) bool {
	return left.EqualsByHalfVector2(right)
}
func HalfVector2OperatorInequalityByHalfVector2AndHalfVector2(left, right HalfVector2) bool {
	return !left.EqualsByHalfVector2(right)
}

type HalfVector4 struct{ packedValue uint64 }

func NewHalfVector4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) HalfVector4 {
	return HalfVector4{packedValue: packHalfVector4(x, y, z, w)}
}
func NewHalfVector4ByVector4(vector framework.Vector4) HalfVector4 {
	return HalfVector4{packedValue: packHalfVector4(vector.X, vector.Y, vector.Z, vector.W)}
}
func packHalfVector4(x, y, z, w float32) uint64 {
	return uint64(packHalf(x)) |
		uint64(packHalf(y))<<16 |
		uint64(packHalf(z))<<32 |
		uint64(packHalf(w))<<48
}
func (v HalfVector4) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackHalf(uint16(v.packedValue)),
		Y: unpackHalf(uint16(v.packedValue >> 16)),
		Z: unpackHalf(uint16(v.packedValue >> 32)),
		W: unpackHalf(uint16(v.packedValue >> 48)),
	}
}
func (v *HalfVector4) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packHalfVector4(vector.X, vector.Y, vector.Z, vector.W)
}
func (v HalfVector4) PackedValue() uint64          { return v.packedValue }
func (v *HalfVector4) SetPackedValue(value uint64) { v.packedValue = value }
func (v HalfVector4) ToString() string             { return v.ToVector4().ToString() }
func (v HalfVector4) GetHashCode() int32           { return hashUint64(v.packedValue) }
func (v HalfVector4) EqualsByObject(value any) bool {
	other, ok := value.(HalfVector4)
	return ok && v.EqualsByHalfVector4(other)
}
func (v HalfVector4) EqualsByHalfVector4(other HalfVector4) bool {
	return v.packedValue == other.packedValue
}
func HalfVector4OperatorEqualityByHalfVector4AndHalfVector4(left, right HalfVector4) bool {
	return left.EqualsByHalfVector4(right)
}
func HalfVector4OperatorInequalityByHalfVector4AndHalfVector4(left, right HalfVector4) bool {
	return !left.EqualsByHalfVector4(right)
}
