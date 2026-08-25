package graphics

import (
	"reflect"
	"testing"
)

func TestGraphicsDeviceStatusCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value GraphicsDeviceStatus
		want  int32
	}{
		{"Normal", GraphicsDeviceStatusNormal, 0},
		{"Lost", GraphicsDeviceStatusLost, 1},
		{"NotReset", GraphicsDeviceStatusNotReset, 2},
	}
	if len(values) != 3 {
		t.Fatalf("GraphicsDeviceStatus literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("GraphicsDeviceStatus%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("GraphicsDeviceStatus%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(GraphicsDeviceStatusNormal).Kind(); got != reflect.Int32 {
		t.Fatalf("GraphicsDeviceStatus underlying kind = %s, want int32", got)
	}
}

func TestGraphicsDeviceStatusZeroAndArbitraryRawValues(t *testing.T) {
	var zero GraphicsDeviceStatus
	if zero != GraphicsDeviceStatusNormal {
		t.Fatalf("zero GraphicsDeviceStatus = %d, want Normal (%d)", zero, GraphicsDeviceStatusNormal)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(GraphicsDeviceStatus(raw)); got != raw {
			t.Fatalf("GraphicsDeviceStatus(%d) = %d", raw, got)
		}
	}
}

func TestGraphicsDeviceStatusSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "graphics_device_status.go", "GraphicsDeviceStatus"); got != false {
		t.Fatalf("GraphicsDeviceStatus xna:flags directive = %t, want false", got)
	}
}
