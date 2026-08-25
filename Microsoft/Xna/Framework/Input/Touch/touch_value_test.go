package touch

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func TestGestureSampleStoresEveryComponentUnchanged(t *testing.T) {
	timestamp := framework.TimeSpanFromTicks(1234567)
	sample := NewGestureSample(
		GestureTypeFreeDrag, timestamp,
		framework.Vector2{X: 1, Y: 2}, framework.Vector2{X: 3, Y: 4},
		framework.Vector2{X: 5, Y: 6}, framework.Vector2{X: 7, Y: 8},
	)
	if sample.GestureType() != GestureTypeFreeDrag {
		t.Fatalf("GestureType = %d", sample.GestureType())
	}
	if sample.Timestamp() != timestamp {
		t.Fatalf("Timestamp = %v", sample.Timestamp())
	}
	for name, got := range map[string]framework.Vector2{
		"Position": sample.Position(), "Position2": sample.Position2(),
		"Delta": sample.Delta(), "Delta2": sample.Delta2(),
	} {
		if got == (framework.Vector2{}) {
			t.Fatalf("%s was not stored", name)
		}
	}
	if sample.Position() != (framework.Vector2{X: 1, Y: 2}) || sample.Position2() != (framework.Vector2{X: 3, Y: 4}) ||
		sample.Delta() != (framework.Vector2{X: 5, Y: 6}) || sample.Delta2() != (framework.Vector2{X: 7, Y: 8}) {
		t.Fatal("gesture components were stored in the wrong order")
	}
	// An arbitrary raw GestureType is not validated.
	arbitrary := NewGestureSample(GestureType(1<<20), timestamp,
		framework.Vector2{}, framework.Vector2{}, framework.Vector2{}, framework.Vector2{})
	if int32(arbitrary.GestureType()) != 1<<20 {
		t.Fatalf("arbitrary gesture type = %d", arbitrary.GestureType())
	}
}

func TestTouchLocationSingleSampleConstructor(t *testing.T) {
	location := NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		7, TouchLocationStateMoved, framework.Vector2{X: 1.5, Y: -2.5})
	if location.Id() != 7 || location.State() != TouchLocationStateMoved {
		t.Fatalf("id/state = %d/%d", location.Id(), location.State())
	}
	if location.Position() != (framework.Vector2{X: 1.5, Y: -2.5}) {
		t.Fatalf("Position = %v", location.Position())
	}
	// With no previous sample the reference reports false and yields a
	// location whose Id is -1 and whose other fields are zero.
	ok, previous := location.TryGetPreviousLocation()
	if ok {
		t.Fatal("TryGetPreviousLocation reported a previous sample that was never supplied")
	}
	if previous.Id() != -1 || previous.State() != TouchLocationStateInvalid ||
		previous.Position() != (framework.Vector2{}) {
		t.Fatalf("empty previous location = %+v", previous)
	}
}

func TestTouchLocationPreviousSamplePromotion(t *testing.T) {
	location := NewTouchLocationByInt32AndTouchLocationStateAndVector2AndTouchLocationStateAndVector2(
		9, TouchLocationStateMoved, framework.Vector2{X: 10, Y: 20},
		TouchLocationStatePressed, framework.Vector2{X: 30, Y: 40})
	ok, previous := location.TryGetPreviousLocation()
	if !ok {
		t.Fatal("TryGetPreviousLocation did not report the supplied previous sample")
	}
	if previous.Id() != 9 || previous.State() != TouchLocationStatePressed ||
		previous.Position() != (framework.Vector2{X: 30, Y: 40}) {
		t.Fatalf("promoted previous location = %+v", previous)
	}
	// The promoted location itself carries no previous sample.
	if nested, _ := previous.TryGetPreviousLocation(); nested {
		t.Fatal("the promoted previous location reported a previous sample of its own")
	}
}

// TestTouchLocationEqualityAndOperatorDisagree records a genuine XNA
// asymmetry: the equality operator compares all seven fields while
// Equals(TouchLocation) deliberately ignores both TouchLocationState fields.
func TestTouchLocationEqualityAndOperatorDisagree(t *testing.T) {
	position := framework.Vector2{X: 1, Y: 2}
	pressed := NewTouchLocationByInt32AndTouchLocationStateAndVector2(3, TouchLocationStatePressed, position)
	moved := NewTouchLocationByInt32AndTouchLocationStateAndVector2(3, TouchLocationStateMoved, position)

	if !pressed.EqualsByTouchLocation(moved) {
		t.Fatal("Equals must ignore the state field")
	}
	if !pressed.EqualsByObject(moved) {
		t.Fatal("Equals(object) must delegate to the typed comparison")
	}
	if TouchLocationOperatorEqualityByTouchLocationAndTouchLocation(pressed, moved) {
		t.Fatal("the equality operator must observe the state field")
	}
	if !TouchLocationOperatorInequalityByTouchLocationAndTouchLocation(pressed, moved) {
		t.Fatal("the inequality operator must be the exact negation")
	}
	if pressed.EqualsByObject("not a touch location") || pressed.EqualsByObject(nil) {
		t.Fatal("Equals(object) accepted a foreign type")
	}
	different := NewTouchLocationByInt32AndTouchLocationStateAndVector2(4, TouchLocationStatePressed, position)
	if pressed.EqualsByTouchLocation(different) {
		t.Fatal("different identifiers compared equal")
	}
}

func TestTouchLocationHashAndString(t *testing.T) {
	location := NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		5, TouchLocationStatePressed, framework.Vector2{X: 1, Y: 2})
	// 5 + bits(1) + bits(2) with wrapping Int32 addition.
	want := int32(5) + int32(math.Float32bits(1)) + int32(math.Float32bits(2))
	if got := location.GetHashCode(); got != want {
		t.Fatalf("GetHashCode = %d, want %d", got, want)
	}
	// Both signed zeros hash to zero, so the position contributes nothing.
	zeroed := NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		11, TouchLocationStatePressed,
		framework.Vector2{X: 0, Y: float32(math.Copysign(0, -1))})
	if got := zeroed.GetHashCode(); got != 11 {
		t.Fatalf("signed-zero GetHashCode = %d, want 11", got)
	}
	if got := location.ToString(); got != "{Position:{X:1 Y:2}}" {
		t.Fatalf("ToString = %q", got)
	}
	// The state is deliberately absent from the reference format.
	var zero TouchLocation
	if got := zero.ToString(); got != "{Position:{X:0 Y:0}}" {
		t.Fatalf("zero ToString = %q", got)
	}
}
