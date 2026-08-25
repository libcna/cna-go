package graphics

import (
	"reflect"
	"testing"
)

func TestPresentIntervalCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value PresentInterval
		want  int32
	}{
		{"Default", PresentIntervalDefault, 0},
		{"One", PresentIntervalOne, 1},
		{"Two", PresentIntervalTwo, 2},
		{"Immediate", PresentIntervalImmediate, 3},
	}
	if len(values) != 4 {
		t.Fatalf("PresentInterval literal count = %d, want 4", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("PresentInterval%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("PresentInterval%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(PresentIntervalDefault).Kind(); got != reflect.Int32 {
		t.Fatalf("PresentInterval underlying kind = %s, want int32", got)
	}
}

func TestPresentIntervalZeroAndArbitraryRawValues(t *testing.T) {
	var zero PresentInterval
	if zero != PresentIntervalDefault {
		t.Fatalf("zero PresentInterval = %d, want Default (%d)", zero, PresentIntervalDefault)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(PresentInterval(raw)); got != raw {
			t.Fatalf("PresentInterval(%d) = %d", raw, got)
		}
	}
}

func TestPresentIntervalSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "present_interval.go", "PresentInterval"); got != false {
		t.Fatalf("PresentInterval xna:flags directive = %t, want false", got)
	}
}
