package media

import (
	"reflect"
	"testing"
)

func TestMediaSourceTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value MediaSourceType
		want  int32
	}{
		{"LocalDevice", MediaSourceTypeLocalDevice, 0},
		{"WindowsMediaConnect", MediaSourceTypeWindowsMediaConnect, 4},
	}
	if len(values) != 2 {
		t.Fatalf("MediaSourceType literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("MediaSourceType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("MediaSourceType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(MediaSourceTypeLocalDevice).Kind(); got != reflect.Int32 {
		t.Fatalf("MediaSourceType underlying kind = %s, want int32", got)
	}
}

func TestMediaSourceTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero MediaSourceType
	if zero != MediaSourceTypeLocalDevice {
		t.Fatalf("zero MediaSourceType = %d, want LocalDevice (%d)", zero, MediaSourceTypeLocalDevice)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(MediaSourceType(raw)); got != raw {
			t.Fatalf("MediaSourceType(%d) = %d", raw, got)
		}
	}
}

func TestMediaSourceTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "media_source_type.go", "MediaSourceType"); got != false {
		t.Fatalf("MediaSourceType xna:flags directive = %t, want false", got)
	}
}
