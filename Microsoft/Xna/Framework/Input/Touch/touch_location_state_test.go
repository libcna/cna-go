package touch

import (
	"reflect"
	"testing"
)

func TestTouchLocationStateCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value TouchLocationState
		want  int32
	}{
		{"Invalid", TouchLocationStateInvalid, 0},
		{"Released", TouchLocationStateReleased, 1},
		{"Pressed", TouchLocationStatePressed, 2},
		{"Moved", TouchLocationStateMoved, 3},
	}
	if len(values) != 4 {
		t.Fatalf("TouchLocationState literal count = %d, want 4", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("TouchLocationState%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("TouchLocationState%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(TouchLocationStateInvalid).Kind(); got != reflect.Int32 {
		t.Fatalf("TouchLocationState underlying kind = %s, want int32", got)
	}
}

func TestTouchLocationStateZeroAndArbitraryRawValues(t *testing.T) {
	var zero TouchLocationState
	if zero != TouchLocationStateInvalid {
		t.Fatalf("zero TouchLocationState = %d, want Invalid (%d)", zero, TouchLocationStateInvalid)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(TouchLocationState(raw)); got != raw {
			t.Fatalf("TouchLocationState(%d) = %d", raw, got)
		}
	}
}

func TestTouchLocationStateSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "touch_location_state.go", "TouchLocationState"); got != false {
		t.Fatalf("TouchLocationState xna:flags directive = %t, want false", got)
	}
}
