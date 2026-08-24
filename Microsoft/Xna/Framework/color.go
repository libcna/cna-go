package framework

import (
	"fmt"
	"math"
)

// Color stores an XNA packed RGBA value. XNA exposes channels as properties,
// so CNA-Go uses accessors rather than public fields.
type Color struct {
	r uint8
	g uint8
	b uint8
	a uint8
}

func NewColorByInt32AndInt32AndInt32(red, green, blue int32) Color {
	return NewColorByInt32AndInt32AndInt32AndInt32(red, green, blue, 255)
}

func NewColorByInt32AndInt32AndInt32AndInt32(red, green, blue, alpha int32) Color {
	return Color{r: clampByte(red), g: clampByte(green), b: clampByte(blue), a: clampByte(alpha)}
}

func NewColorByVector3(value Vector3) Color {
	return NewColorBySingleAndSingleAndSingleAndSingle(value.X, value.Y, value.Z, 1)
}

func NewColorByVector4(value Vector4) Color {
	return NewColorBySingleAndSingleAndSingleAndSingle(value.X, value.Y, value.Z, value.W)
}

func NewColorBySingleAndSingleAndSingle(red, green, blue float32) Color {
	return NewColorBySingleAndSingleAndSingleAndSingle(red, green, blue, 1)
}

func NewColorBySingleAndSingleAndSingleAndSingle(red, green, blue, alpha float32) Color {
	return Color{r: packColorUNorm(red), g: packColorUNorm(green), b: packColorUNorm(blue), a: packColorUNorm(alpha)}
}

func (c Color) R() uint8          { return c.r }
func (c *Color) SetR(value uint8) { c.r = value }
func (c Color) G() uint8          { return c.g }
func (c *Color) SetG(value uint8) { c.g = value }
func (c Color) B() uint8          { return c.b }
func (c *Color) SetB(value uint8) { c.b = value }
func (c Color) A() uint8          { return c.a }
func (c *Color) SetA(value uint8) { c.a = value }

func (c Color) PackedValue() uint32 {
	return uint32(c.r) | uint32(c.g)<<8 | uint32(c.b)<<16 | uint32(c.a)<<24
}

func (c *Color) SetPackedValue(value uint32) {
	c.r = uint8(value)
	c.g = uint8(value >> 8)
	c.b = uint8(value >> 16)
	c.a = uint8(value >> 24)
}

func ColorCornflowerBlue() Color {
	return Color{r: 100, g: 149, b: 237, a: 255}
}

func ColorWhite() Color {
	return Color{r: 255, g: 255, b: 255, a: 255}
}

func (c Color) ToVector3() Vector3 {
	return Vector3{X: float32(c.r) / 255, Y: float32(c.g) / 255, Z: float32(c.b) / 255}
}

func (c Color) ToVector4() Vector4 {
	return Vector4{X: float32(c.r) / 255, Y: float32(c.g) / 255, Z: float32(c.b) / 255, W: float32(c.a) / 255}
}

func ColorFromNonPremultipliedByVector4(value Vector4) Color {
	return NewColorBySingleAndSingleAndSingleAndSingle(value.X*value.W, value.Y*value.W, value.Z*value.W, value.W)
}

func ColorFromNonPremultipliedByInt32AndInt32AndInt32AndInt32(red, green, blue, alpha int32) Color {
	return Color{
		r: clampByte64(int64(red) * int64(alpha) / 255),
		g: clampByte64(int64(green) * int64(alpha) / 255),
		b: clampByte64(int64(blue) * int64(alpha) / 255),
		a: clampByte(alpha),
	}
}

func ColorLerp(value1, value2 Color, amount float32) Color {
	fraction := int32(packUNorm(65536, amount))
	return Color{
		r: uint8(int32(value1.r) + ((int32(value2.r)-int32(value1.r))*fraction)>>16),
		g: uint8(int32(value1.g) + ((int32(value2.g)-int32(value1.g))*fraction)>>16),
		b: uint8(int32(value1.b) + ((int32(value2.b)-int32(value1.b))*fraction)>>16),
		a: uint8(int32(value1.a) + ((int32(value2.a)-int32(value1.a))*fraction)>>16),
	}
}

func ColorMultiply(value Color, scale float32) Color {
	scaled := scale * 65536
	var fixed uint32
	if math.IsNaN(float64(scaled)) || scaled <= 0 {
		fixed = 0
	} else if scaled >= 16777215 {
		fixed = 16777215
	} else {
		fixed = uint32(scaled)
	}
	return Color{
		r: uint8(minUint32(255, uint32(value.r)*fixed>>16)),
		g: uint8(minUint32(255, uint32(value.g)*fixed>>16)),
		b: uint8(minUint32(255, uint32(value.b)*fixed>>16)),
		a: uint8(minUint32(255, uint32(value.a)*fixed>>16)),
	}
}

func (c Color) EqualsByColor(other Color) bool { return c.PackedValue() == other.PackedValue() }
func (c Color) EqualsByObject(value any) bool {
	other, ok := value.(Color)
	return ok && c.EqualsByColor(other)
}
func (c Color) GetHashCode() int32 { return int32(c.PackedValue()) }
func (c Color) ToString() string {
	return fmt.Sprintf("{R:%d G:%d B:%d A:%d}", c.r, c.g, c.b, c.a)
}

func ColorOperatorEqualityByColorAndColor(left, right Color) bool { return left.EqualsByColor(right) }
func ColorOperatorInequalityByColorAndColor(left, right Color) bool {
	return !left.EqualsByColor(right)
}
func ColorOperatorMultiplyByColorAndSingle(value Color, scale float32) Color {
	return ColorMultiply(value, scale)
}

func packColorUNorm(value float32) uint8 { return uint8(packUNorm(255, value)) }

func packUNorm(bitmask float32, value float32) uint32 {
	scaled := value * bitmask
	if math.IsNaN(float64(scaled)) || scaled <= 0 {
		return 0
	}
	if scaled >= bitmask {
		return uint32(bitmask)
	}
	return uint32(math.RoundToEven(float64(scaled)))
}

func clampByte(value int32) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

func clampByte64(value int64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

func minUint32(left, right uint32) uint32 {
	if left < right {
		return left
	}
	return right
}
