package media

import (
	"reflect"
	"testing"
)

func TestMediaStateCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value MediaState
		want  int32
	}{
		{"Stopped", MediaStateStopped, 0},
		{"Playing", MediaStatePlaying, 1},
		{"Paused", MediaStatePaused, 2},
	}
	if len(values) != 3 {
		t.Fatalf("MediaState literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("MediaState%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("MediaState%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(MediaStateStopped).Kind(); got != reflect.Int32 {
		t.Fatalf("MediaState underlying kind = %s, want int32", got)
	}
}

func TestMediaStateZeroAndArbitraryRawValues(t *testing.T) {
	var zero MediaState
	if zero != MediaStateStopped {
		t.Fatalf("zero MediaState = %d, want Stopped (%d)", zero, MediaStateStopped)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(MediaState(raw)); got != raw {
			t.Fatalf("MediaState(%d) = %d", raw, got)
		}
	}
}

func TestMediaStateSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "media_state.go", "MediaState"); got != false {
		t.Fatalf("MediaState xna:flags directive = %t, want false", got)
	}
}
