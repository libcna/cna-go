package input

import (
	"reflect"
	"testing"
)

func TestGamePadTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value GamePadType
		want  int32
	}{
		{"Unknown", GamePadTypeUnknown, 0},
		{"GamePad", GamePadTypeGamePad, 1},
		{"Wheel", GamePadTypeWheel, 2},
		{"ArcadeStick", GamePadTypeArcadeStick, 3},
		{"FlightStick", GamePadTypeFlightStick, 4},
		{"DancePad", GamePadTypeDancePad, 5},
		{"Guitar", GamePadTypeGuitar, 6},
		{"AlternateGuitar", GamePadTypeAlternateGuitar, 7},
		{"DrumKit", GamePadTypeDrumKit, 8},
		{"BigButtonPad", GamePadTypeBigButtonPad, 768},
	}
	if len(values) != 10 {
		t.Fatalf("GamePadType literal count = %d, want 10", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("GamePadType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("GamePadType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(GamePadTypeUnknown).Kind(); got != reflect.Int32 {
		t.Fatalf("GamePadType underlying kind = %s, want int32", got)
	}
}

func TestGamePadTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero GamePadType
	if zero != GamePadTypeUnknown {
		t.Fatalf("zero GamePadType = %d, want Unknown (%d)", zero, GamePadTypeUnknown)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(GamePadType(raw)); got != raw {
			t.Fatalf("GamePadType(%d) = %d", raw, got)
		}
	}
}

func TestGamePadTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "game_pad_type.go", "GamePadType"); got != false {
		t.Fatalf("GamePadType xna:flags directive = %t, want false", got)
	}
}
