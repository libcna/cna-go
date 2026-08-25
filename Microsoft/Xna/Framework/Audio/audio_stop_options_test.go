package audio

import (
	"reflect"
	"testing"
)

func TestAudioStopOptionsCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value AudioStopOptions
		want  int32
	}{
		{"AsAuthored", AudioStopOptionsAsAuthored, 0},
		{"Immediate", AudioStopOptionsImmediate, 1},
	}
	if len(values) != 2 {
		t.Fatalf("AudioStopOptions literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("AudioStopOptions%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("AudioStopOptions%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(AudioStopOptionsAsAuthored).Kind(); got != reflect.Int32 {
		t.Fatalf("AudioStopOptions underlying kind = %s, want int32", got)
	}
}

func TestAudioStopOptionsZeroAndArbitraryRawValues(t *testing.T) {
	var zero AudioStopOptions
	if zero != AudioStopOptionsAsAuthored {
		t.Fatalf("zero AudioStopOptions = %d, want AsAuthored (%d)", zero, AudioStopOptionsAsAuthored)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(AudioStopOptions(raw)); got != raw {
			t.Fatalf("AudioStopOptions(%d) = %d", raw, got)
		}
	}
}

func TestAudioStopOptionsSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "audio_stop_options.go", "AudioStopOptions"); got != false {
		t.Fatalf("AudioStopOptions xna:flags directive = %t, want false", got)
	}
}
