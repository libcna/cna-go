package graphics

import (
	"reflect"
	"testing"
)

func TestFillModeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value FillMode
		want  int32
	}{
		{"Solid", FillModeSolid, 0},
		{"WireFrame", FillModeWireFrame, 1},
	}
	if len(values) != 2 {
		t.Fatalf("FillMode literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("FillMode%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("FillMode%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(FillModeSolid).Kind(); got != reflect.Int32 {
		t.Fatalf("FillMode underlying kind = %s, want int32", got)
	}
}

func TestFillModeZeroAndArbitraryRawValues(t *testing.T) {
	var zero FillMode
	if zero != FillModeSolid {
		t.Fatalf("zero FillMode = %d, want Solid (%d)", zero, FillModeSolid)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(FillMode(raw)); got != raw {
			t.Fatalf("FillMode(%d) = %d", raw, got)
		}
	}
}

func TestFillModeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "fill_mode.go", "FillMode"); got != false {
		t.Fatalf("FillMode xna:flags directive = %t, want false", got)
	}
}
