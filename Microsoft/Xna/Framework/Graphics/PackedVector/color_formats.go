package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

type Alpha8 struct{ packedValue uint8 }

func NewAlpha8(alpha float32) Alpha8 { return Alpha8{packedValue: uint8(packUNorm(255, alpha))} }

func (v Alpha8) ToAlpha() float32 { return unpackUNorm(255, uint32(v.packedValue)) }
func (v Alpha8) ToVector4() framework.Vector4 {
	return framework.Vector4{W: v.ToAlpha()}
}
func (v *Alpha8) PackFromVector4(vector framework.Vector4) {
	v.packedValue = uint8(packUNorm(255, vector.W))
}
func (v Alpha8) PackedValue() uint8          { return v.packedValue }
func (v *Alpha8) SetPackedValue(value uint8) { v.packedValue = value }
func (v Alpha8) ToString() string            { return formatPacked(uint64(v.packedValue), 2) }
func (v Alpha8) GetHashCode() int32          { return int32(v.packedValue) }
func (v Alpha8) EqualsByObject(value any) bool {
	other, ok := value.(Alpha8)
	return ok && v.EqualsByAlpha8(other)
}
func (v Alpha8) EqualsByAlpha8(other Alpha8) bool { return v.packedValue == other.packedValue }
func Alpha8OperatorEqualityByAlpha8AndAlpha8(left, right Alpha8) bool {
	return left.EqualsByAlpha8(right)
}
func Alpha8OperatorInequalityByAlpha8AndAlpha8(left, right Alpha8) bool {
	return !left.EqualsByAlpha8(right)
}

type Bgr565 struct{ packedValue uint16 }

func NewBgr565BySingleAndSingleAndSingle(x, y, z float32) Bgr565 {
	return Bgr565{packedValue: packBgr565(x, y, z)}
}
func NewBgr565ByVector3(vector framework.Vector3) Bgr565 {
	return Bgr565{packedValue: packBgr565(vector.X, vector.Y, vector.Z)}
}
func packBgr565(x, y, z float32) uint16 {
	return uint16(packUNorm(31, x)<<11 | packUNorm(63, y)<<5 | packUNorm(31, z))
}
func (v Bgr565) ToVector3() framework.Vector3 {
	return framework.Vector3{
		X: unpackUNorm(31, uint32(v.packedValue)>>11),
		Y: unpackUNorm(63, uint32(v.packedValue)>>5),
		Z: unpackUNorm(31, uint32(v.packedValue)),
	}
}
func (v Bgr565) ToVector4() framework.Vector4 {
	vector := v.ToVector3()
	return framework.Vector4{X: vector.X, Y: vector.Y, Z: vector.Z, W: 1}
}
func (v *Bgr565) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packBgr565(vector.X, vector.Y, vector.Z)
}
func (v Bgr565) PackedValue() uint16          { return v.packedValue }
func (v *Bgr565) SetPackedValue(value uint16) { v.packedValue = value }
func (v Bgr565) ToString() string             { return formatPacked(uint64(v.packedValue), 4) }
func (v Bgr565) GetHashCode() int32           { return int32(v.packedValue) }
func (v Bgr565) EqualsByObject(value any) bool {
	other, ok := value.(Bgr565)
	return ok && v.EqualsByBgr565(other)
}
func (v Bgr565) EqualsByBgr565(other Bgr565) bool { return v.packedValue == other.packedValue }
func Bgr565OperatorEqualityByBgr565AndBgr565(left, right Bgr565) bool {
	return left.EqualsByBgr565(right)
}
func Bgr565OperatorInequalityByBgr565AndBgr565(left, right Bgr565) bool {
	return !left.EqualsByBgr565(right)
}

type Bgra4444 struct{ packedValue uint16 }

func NewBgra4444BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Bgra4444 {
	return Bgra4444{packedValue: packBgra4444(x, y, z, w)}
}
func NewBgra4444ByVector4(vector framework.Vector4) Bgra4444 {
	return Bgra4444{packedValue: packBgra4444(vector.X, vector.Y, vector.Z, vector.W)}
}
func packBgra4444(x, y, z, w float32) uint16 {
	return uint16(packUNorm(15, x)<<8 | packUNorm(15, y)<<4 | packUNorm(15, z) | packUNorm(15, w)<<12)
}
func (v Bgra4444) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackUNorm(15, uint32(v.packedValue)>>8),
		Y: unpackUNorm(15, uint32(v.packedValue)>>4),
		Z: unpackUNorm(15, uint32(v.packedValue)),
		W: unpackUNorm(15, uint32(v.packedValue)>>12),
	}
}
func (v *Bgra4444) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packBgra4444(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Bgra4444) PackedValue() uint16          { return v.packedValue }
func (v *Bgra4444) SetPackedValue(value uint16) { v.packedValue = value }
func (v Bgra4444) ToString() string             { return formatPacked(uint64(v.packedValue), 4) }
func (v Bgra4444) GetHashCode() int32           { return int32(v.packedValue) }
func (v Bgra4444) EqualsByObject(value any) bool {
	other, ok := value.(Bgra4444)
	return ok && v.EqualsByBgra4444(other)
}
func (v Bgra4444) EqualsByBgra4444(other Bgra4444) bool {
	return v.packedValue == other.packedValue
}
func Bgra4444OperatorEqualityByBgra4444AndBgra4444(left, right Bgra4444) bool {
	return left.EqualsByBgra4444(right)
}
func Bgra4444OperatorInequalityByBgra4444AndBgra4444(left, right Bgra4444) bool {
	return !left.EqualsByBgra4444(right)
}

type Bgra5551 struct{ packedValue uint16 }

func NewBgra5551BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Bgra5551 {
	return Bgra5551{packedValue: packBgra5551(x, y, z, w)}
}
func NewBgra5551ByVector4(vector framework.Vector4) Bgra5551 {
	return Bgra5551{packedValue: packBgra5551(vector.X, vector.Y, vector.Z, vector.W)}
}
func packBgra5551(x, y, z, w float32) uint16 {
	return uint16(packUNorm(31, x)<<10 | packUNorm(31, y)<<5 | packUNorm(31, z) | packUNorm(1, w)<<15)
}
func (v Bgra5551) ToVector4() framework.Vector4 {
	return framework.Vector4{
		X: unpackUNorm(31, uint32(v.packedValue)>>10),
		Y: unpackUNorm(31, uint32(v.packedValue)>>5),
		Z: unpackUNorm(31, uint32(v.packedValue)),
		W: unpackUNorm(1, uint32(v.packedValue)>>15),
	}
}
func (v *Bgra5551) PackFromVector4(vector framework.Vector4) {
	v.packedValue = packBgra5551(vector.X, vector.Y, vector.Z, vector.W)
}
func (v Bgra5551) PackedValue() uint16          { return v.packedValue }
func (v *Bgra5551) SetPackedValue(value uint16) { v.packedValue = value }
func (v Bgra5551) ToString() string             { return formatPacked(uint64(v.packedValue), 4) }
func (v Bgra5551) GetHashCode() int32           { return int32(v.packedValue) }
func (v Bgra5551) EqualsByObject(value any) bool {
	other, ok := value.(Bgra5551)
	return ok && v.EqualsByBgra5551(other)
}
func (v Bgra5551) EqualsByBgra5551(other Bgra5551) bool {
	return v.packedValue == other.packedValue
}
func Bgra5551OperatorEqualityByBgra5551AndBgra5551(left, right Bgra5551) bool {
	return left.EqualsByBgra5551(right)
}
func Bgra5551OperatorInequalityByBgra5551AndBgra5551(left, right Bgra5551) bool {
	return !left.EqualsByBgra5551(right)
}
