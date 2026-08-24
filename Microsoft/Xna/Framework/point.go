package framework

import "fmt"

// Point is a two-dimensional integer coordinate with XNA value semantics.
type Point struct {
	X int32
	Y int32
}

func NewPoint(x, y int32) Point {
	return Point{X: x, Y: y}
}

func (p Point) EqualsByPoint(other Point) bool {
	return p.X == other.X && p.Y == other.Y
}

func (p Point) EqualsByObject(value any) bool {
	other, ok := value.(Point)
	return ok && p.EqualsByPoint(other)
}

func (p Point) GetHashCode() int32 {
	return p.X + p.Y
}

func (p Point) ToString() string {
	return fmt.Sprintf("{X:%d Y:%d}", p.X, p.Y)
}

func PointOperatorEqualityByPointAndPoint(left, right Point) bool {
	return left.EqualsByPoint(right)
}

func PointOperatorInequalityByPointAndPoint(left, right Point) bool {
	return !left.EqualsByPoint(right)
}

func PointZero() Point {
	return Point{}
}
