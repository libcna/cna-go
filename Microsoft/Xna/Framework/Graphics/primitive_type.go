package graphics

// PrimitiveType identifies how XNA assembles vertices into primitives.
type PrimitiveType int32

const (
	PrimitiveTypeTriangleList  PrimitiveType = 0
	PrimitiveTypeTriangleStrip PrimitiveType = 1
	PrimitiveTypeLineList      PrimitiveType = 2
	PrimitiveTypeLineStrip     PrimitiveType = 3
)
