package touch

import (
	"fmt"
	"math"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TouchLocation is a managed value describing one XNA touch point and,
// optionally, its previous sample. Constructing it claims no touch capability:
// CNA-Go exposes no TouchPanel and reads no device.
type TouchLocation struct {
	id        int32
	state     TouchLocationState
	x         float32
	y         float32
	prevState TouchLocationState
	prevX     float32
	prevY     float32
}

// NewTouchLocationByInt32AndTouchLocationStateAndVector2 stores a touch point
// with no previous sample. The reference constructor zeroes the previous
// fields and leaves the previous state Invalid.
func NewTouchLocationByInt32AndTouchLocationStateAndVector2(id int32, state TouchLocationState, position framework.Vector2) TouchLocation {
	return TouchLocation{id: id, state: state, x: position.X, y: position.Y}
}

// NewTouchLocationByInt32AndTouchLocationStateAndVector2AndTouchLocationStateAndVector2
// stores a touch point together with its previous sample.
func NewTouchLocationByInt32AndTouchLocationStateAndVector2AndTouchLocationStateAndVector2(id int32, state TouchLocationState, position framework.Vector2, previousState TouchLocationState, previousPosition framework.Vector2) TouchLocation {
	return TouchLocation{
		id: id, state: state, x: position.X, y: position.Y,
		prevState: previousState, prevX: previousPosition.X, prevY: previousPosition.Y,
	}
}

func (l TouchLocation) Id() int32                 { return l.id }
func (l TouchLocation) State() TouchLocationState { return l.state }
func (l TouchLocation) Position() framework.Vector2 {
	return framework.Vector2{X: l.x, Y: l.y}
}

// TryGetPreviousLocation reproduces the reference out-parameter method. When
// the previous state is the zero literal Invalid it reports false and yields a
// location whose Id is -1 and whose other fields are zero; otherwise it yields
// the previous sample promoted into a location that itself has no previous
// sample.
func (l TouchLocation) TryGetPreviousLocation() (bool, TouchLocation) {
	if l.prevState == TouchLocationStateInvalid {
		return false, TouchLocation{id: -1}
	}
	return true, TouchLocation{id: l.id, state: l.prevState, x: l.prevX, y: l.prevY}
}

// EqualsByTouchLocation reproduces the reference IEquatable implementation,
// which deliberately compares only the identifier and the two positions. It
// ignores both TouchLocationState fields, so it can report true where the
// equality operator reports false.
func (l TouchLocation) EqualsByTouchLocation(other TouchLocation) bool {
	return l.id == other.id && l.x == other.x && l.y == other.y &&
		l.prevX == other.prevX && l.prevY == other.prevY
}

func (l TouchLocation) EqualsByObject(obj any) bool {
	other, ok := obj.(TouchLocation)
	return ok && l.EqualsByTouchLocation(other)
}

// GetHashCode reproduces the reference sum of Int32.GetHashCode and two
// Single.GetHashCode values, including the CLR canonicalization of both signed
// zeros and the wrapping Int32 addition.
func (l TouchLocation) GetHashCode() int32 {
	return l.id + singleHashCode(l.x) + singleHashCode(l.y)
}

func (l TouchLocation) ToString() string {
	return fmt.Sprintf("{Position:%s}", l.Position().ToString())
}

// TouchLocationOperatorEqualityByTouchLocationAndTouchLocation reproduces the
// reference operator, which compares all seven fields including both
// TouchLocationState values. It is deliberately stricter than
// EqualsByTouchLocation.
func TouchLocationOperatorEqualityByTouchLocationAndTouchLocation(value1, value2 TouchLocation) bool {
	return value1.id == value2.id && value1.state == value2.state &&
		value1.x == value2.x && value1.y == value2.y &&
		value1.prevState == value2.prevState &&
		value1.prevX == value2.prevX && value1.prevY == value2.prevY
}

func TouchLocationOperatorInequalityByTouchLocationAndTouchLocation(value1, value2 TouchLocation) bool {
	return !TouchLocationOperatorEqualityByTouchLocationAndTouchLocation(value1, value2)
}

// singleHashCode reproduces System.Single.GetHashCode: both signed zeros hash
// to zero and every other value hashes to its raw bit pattern.
func singleHashCode(value float32) int32 {
	if value == 0 {
		return 0
	}
	return int32(math.Float32bits(value))
}
