package framework

// Vector2 is a two-dimensional XNA vector implemented entirely in Go.
type Vector2 struct {
	X float32
	Y float32
}

func NewVector2BySingle(value float32) Vector2 {
	return Vector2{X: value, Y: value}
}

func NewVector2BySingleAndSingle(x, y float32) Vector2 {
	return Vector2{X: x, Y: y}
}

func Vector2AddByVector2AndVector2(value1, value2 Vector2) Vector2 {
	return Vector2{X: value1.X + value2.X, Y: value1.Y + value2.Y}
}

// LengthSquared returns the squared Euclidean length of v.
func (v Vector2) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y
}
