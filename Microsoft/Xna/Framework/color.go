package framework

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

func clampByte(value int32) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}
