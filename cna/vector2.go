package cna

// Vector2 is a two-dimensional vector implemented entirely in Go.
type Vector2 struct {
	X float32
	Y float32
}

// NewVector2 constructs a vector from its components.
func NewVector2(x, y float32) Vector2 {
	return Vector2{X: x, Y: y}
}

// Add returns the component-wise sum of v and other.
func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Scale returns v multiplied by scale.
func (v Vector2) Scale(scale float32) Vector2 {
	return Vector2{X: v.X * scale, Y: v.Y * scale}
}

// LengthSquared returns the squared Euclidean length of v.
func (v Vector2) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y
}
