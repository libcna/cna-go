package audio

import (
	"reflect"
	"testing"
)

func TestSoundStateCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value SoundState
		want  int32
	}{
		{"Playing", SoundStatePlaying, 0},
		{"Paused", SoundStatePaused, 1},
		{"Stopped", SoundStateStopped, 2},
	}
	if len(values) != 3 {
		t.Fatalf("SoundState literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("SoundState%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("SoundState%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(SoundStatePlaying).Kind(); got != reflect.Int32 {
		t.Fatalf("SoundState underlying kind = %s, want int32", got)
	}
}

func TestSoundStateZeroAndArbitraryRawValues(t *testing.T) {
	var zero SoundState
	if zero != SoundStatePlaying {
		t.Fatalf("zero SoundState = %d, want Playing (%d)", zero, SoundStatePlaying)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(SoundState(raw)); got != raw {
			t.Fatalf("SoundState(%d) = %d", raw, got)
		}
	}
}

func TestSoundStateSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "sound_state.go", "SoundState"); got != false {
		t.Fatalf("SoundState xna:flags directive = %t, want false", got)
	}
}
