package touch

import (
	"reflect"
	"testing"
)

func TestGestureTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value GestureType
		want  int32
	}{
		{"None", GestureTypeNone, 0},
		{"Tap", GestureTypeTap, 1},
		{"DoubleTap", GestureTypeDoubleTap, 2},
		{"Hold", GestureTypeHold, 4},
		{"HorizontalDrag", GestureTypeHorizontalDrag, 8},
		{"VerticalDrag", GestureTypeVerticalDrag, 16},
		{"FreeDrag", GestureTypeFreeDrag, 32},
		{"Pinch", GestureTypePinch, 64},
		{"Flick", GestureTypeFlick, 128},
		{"DragComplete", GestureTypeDragComplete, 256},
		{"PinchComplete", GestureTypePinchComplete, 512},
	}
	if len(values) != 11 {
		t.Fatalf("GestureType literal count = %d, want 11", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("GestureType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("GestureType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(GestureTypeNone).Kind(); got != reflect.Int32 {
		t.Fatalf("GestureType underlying kind = %s, want int32", got)
	}
}

func TestGestureTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero GestureType
	if zero != GestureTypeNone {
		t.Fatalf("zero GestureType = %d, want None (%d)", zero, GestureTypeNone)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(GestureType(raw)); got != raw {
			t.Fatalf("GestureType(%d) = %d", raw, got)
		}
	}
}

func TestGestureTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "gesture_type.go", "GestureType"); got != true {
		t.Fatalf("GestureType xna:flags directive = %t, want true", got)
	}
}
