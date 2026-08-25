package input

import (
	"reflect"
	"testing"
)

func TestGamePadDeadZoneCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value GamePadDeadZone
		want  int32
	}{
		{"None", GamePadDeadZoneNone, 0},
		{"IndependentAxes", GamePadDeadZoneIndependentAxes, 1},
		{"Circular", GamePadDeadZoneCircular, 2},
	}
	if len(values) != 3 {
		t.Fatalf("GamePadDeadZone literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("GamePadDeadZone%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("GamePadDeadZone%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(GamePadDeadZoneNone).Kind(); got != reflect.Int32 {
		t.Fatalf("GamePadDeadZone underlying kind = %s, want int32", got)
	}
}

func TestGamePadDeadZoneZeroAndArbitraryRawValues(t *testing.T) {
	var zero GamePadDeadZone
	if zero != GamePadDeadZoneNone {
		t.Fatalf("zero GamePadDeadZone = %d, want None (%d)", zero, GamePadDeadZoneNone)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(GamePadDeadZone(raw)); got != raw {
			t.Fatalf("GamePadDeadZone(%d) = %d", raw, got)
		}
	}
}

func TestGamePadDeadZoneSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "game_pad_dead_zone.go", "GamePadDeadZone"); got != false {
		t.Fatalf("GamePadDeadZone xna:flags directive = %t, want false", got)
	}
}
