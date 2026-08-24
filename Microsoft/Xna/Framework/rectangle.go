package framework

import "fmt"

// Rectangle is an XNA integer rectangle with value-copy semantics.
type Rectangle struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
}

func NewRectangle(x, y, width, height int32) Rectangle {
	return Rectangle{X: x, Y: y, Width: width, Height: height}
}

func (r *Rectangle) OffsetByPoint(amount Point) {
	r.X += amount.X
	r.Y += amount.Y
}

func (r *Rectangle) OffsetByInt32AndInt32(offsetX, offsetY int32) {
	r.X += offsetX
	r.Y += offsetY
}

func (r *Rectangle) Inflate(horizontalAmount, verticalAmount int32) {
	r.X -= horizontalAmount
	r.Y -= verticalAmount
	r.Width += horizontalAmount * 2
	r.Height += verticalAmount * 2
}

func (r Rectangle) ContainsByInt32AndInt32(x, y int32) bool {
	return r.X <= x && x < r.Right() && r.Y <= y && y < r.Bottom()
}

func (r Rectangle) ContainsByPoint(value Point) bool {
	return r.ContainsByInt32AndInt32(value.X, value.Y)
}

func (r Rectangle) ContainsByRefPointAndOutBoolean(value *Point) bool {
	return r.ContainsByPoint(*value)
}

func (r Rectangle) ContainsByRectangle(value Rectangle) bool {
	return r.X <= value.X && value.Right() <= r.Right() &&
		r.Y <= value.Y && value.Bottom() <= r.Bottom()
}

func (r Rectangle) ContainsByRefRectangleAndOutBoolean(value *Rectangle) bool {
	return r.ContainsByRectangle(*value)
}

func (r Rectangle) IntersectsByRectangle(value Rectangle) bool {
	return value.X < r.Right() && r.X < value.Right() &&
		value.Y < r.Bottom() && r.Y < value.Bottom()
}

func (r Rectangle) IntersectsByRefRectangleAndOutBoolean(value *Rectangle) bool {
	return r.IntersectsByRectangle(*value)
}

func RectangleIntersectByRectangleAndRectangle(value1, value2 Rectangle) Rectangle {
	right := minInt32(value1.Right(), value2.Right())
	bottom := minInt32(value1.Bottom(), value2.Bottom())
	left := maxInt32(value1.X, value2.X)
	top := maxInt32(value1.Y, value2.Y)
	if right > left && bottom > top {
		return Rectangle{X: left, Y: top, Width: right - left, Height: bottom - top}
	}
	return RectangleEmpty()
}

func RectangleIntersectByRefRectangleAndRefRectangleAndOutRectangle(value1, value2 *Rectangle) Rectangle {
	return RectangleIntersectByRectangleAndRectangle(*value1, *value2)
}

func RectangleUnionByRectangleAndRectangle(value1, value2 Rectangle) Rectangle {
	right := maxInt32(value1.Right(), value2.Right())
	bottom := maxInt32(value1.Bottom(), value2.Bottom())
	left := minInt32(value1.X, value2.X)
	top := minInt32(value1.Y, value2.Y)
	return Rectangle{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func RectangleUnionByRefRectangleAndRefRectangleAndOutRectangle(value1, value2 *Rectangle) Rectangle {
	return RectangleUnionByRectangleAndRectangle(*value1, *value2)
}

func (r Rectangle) EqualsByRectangle(other Rectangle) bool {
	return r == other
}

func (r Rectangle) EqualsByObject(value any) bool {
	other, ok := value.(Rectangle)
	return ok && r.EqualsByRectangle(other)
}

func (r Rectangle) ToString() string {
	return fmt.Sprintf("{X:%d Y:%d Width:%d Height:%d}", r.X, r.Y, r.Width, r.Height)
}

func (r Rectangle) GetHashCode() int32 {
	return r.X + r.Y + r.Width + r.Height
}

func RectangleOperatorEqualityByRectangleAndRectangle(left, right Rectangle) bool {
	return left == right
}

func RectangleOperatorInequalityByRectangleAndRectangle(left, right Rectangle) bool {
	return left != right
}

func (r Rectangle) Left() int32 {
	return r.X
}

func (r Rectangle) Right() int32 {
	return r.X + r.Width
}

func (r Rectangle) Top() int32 {
	return r.Y
}

func (r Rectangle) Bottom() int32 {
	return r.Y + r.Height
}

func (r Rectangle) Location() Point {
	return Point{X: r.X, Y: r.Y}
}

func (r *Rectangle) SetLocation(value Point) {
	r.X = value.X
	r.Y = value.Y
}

func (r Rectangle) Center() Point {
	return Point{X: r.X + r.Width/2, Y: r.Y + r.Height/2}
}

func RectangleEmpty() Rectangle {
	return Rectangle{}
}

func (r Rectangle) IsEmpty() bool {
	return r.X == 0 && r.Y == 0 && r.Width == 0 && r.Height == 0
}

func minInt32(left, right int32) int32 {
	if left < right {
		return left
	}
	return right
}

func maxInt32(left, right int32) int32 {
	if left > right {
		return left
	}
	return right
}
