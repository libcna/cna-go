package graphics

import (
	"reflect"
	"testing"
)

func TestColorWriteChannelsCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value ColorWriteChannels
		want  int32
	}{
		{"None", ColorWriteChannelsNone, 0},
		{"Red", ColorWriteChannelsRed, 1},
		{"Green", ColorWriteChannelsGreen, 2},
		{"Blue", ColorWriteChannelsBlue, 4},
		{"Alpha", ColorWriteChannelsAlpha, 8},
		{"All", ColorWriteChannelsAll, 15},
	}
	if len(values) != 6 {
		t.Fatalf("ColorWriteChannels literal count = %d, want 6", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("ColorWriteChannels%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("ColorWriteChannels%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(ColorWriteChannelsNone).Kind(); got != reflect.Int32 {
		t.Fatalf("ColorWriteChannels underlying kind = %s, want int32", got)
	}
}

func TestColorWriteChannelsZeroAndArbitraryRawValues(t *testing.T) {
	var zero ColorWriteChannels
	if zero != ColorWriteChannelsNone {
		t.Fatalf("zero ColorWriteChannels = %d, want None (%d)", zero, ColorWriteChannelsNone)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(ColorWriteChannels(raw)); got != raw {
			t.Fatalf("ColorWriteChannels(%d) = %d", raw, got)
		}
	}
}

func TestColorWriteChannelsSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "color_write_channels.go", "ColorWriteChannels"); got != true {
		t.Fatalf("ColorWriteChannels xna:flags directive = %t, want true", got)
	}
}
