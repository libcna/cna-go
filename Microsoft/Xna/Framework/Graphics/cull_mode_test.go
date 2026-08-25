package graphics

import (
	"reflect"
	"testing"
)

func TestCullModeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value CullMode
		want  int32
	}{
		{"None", CullModeNone, 0},
		{"CullClockwiseFace", CullModeCullClockwiseFace, 1},
		{"CullCounterClockwiseFace", CullModeCullCounterClockwiseFace, 2},
	}
	if len(values) != 3 {
		t.Fatalf("CullMode literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("CullMode%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("CullMode%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(CullModeNone).Kind(); got != reflect.Int32 {
		t.Fatalf("CullMode underlying kind = %s, want int32", got)
	}
}

func TestCullModeZeroAndArbitraryRawValues(t *testing.T) {
	var zero CullMode
	if zero != CullModeNone {
		t.Fatalf("zero CullMode = %d, want None (%d)", zero, CullModeNone)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(CullMode(raw)); got != raw {
			t.Fatalf("CullMode(%d) = %d", raw, got)
		}
	}
}

func TestCullModeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "cull_mode.go", "CullMode"); got != false {
		t.Fatalf("CullMode xna:flags directive = %t, want false", got)
	}
}
