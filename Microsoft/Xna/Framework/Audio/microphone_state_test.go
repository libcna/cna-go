package audio

import (
	"reflect"
	"testing"
)

func TestMicrophoneStateCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value MicrophoneState
		want  int32
	}{
		{"Started", MicrophoneStateStarted, 0},
		{"Stopped", MicrophoneStateStopped, 1},
	}
	if len(values) != 2 {
		t.Fatalf("MicrophoneState literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("MicrophoneState%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("MicrophoneState%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(MicrophoneStateStarted).Kind(); got != reflect.Int32 {
		t.Fatalf("MicrophoneState underlying kind = %s, want int32", got)
	}
}

func TestMicrophoneStateZeroAndArbitraryRawValues(t *testing.T) {
	var zero MicrophoneState
	if zero != MicrophoneStateStarted {
		t.Fatalf("zero MicrophoneState = %d, want Started (%d)", zero, MicrophoneStateStarted)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(MicrophoneState(raw)); got != raw {
			t.Fatalf("MicrophoneState(%d) = %d", raw, got)
		}
	}
}

func TestMicrophoneStateSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "microphone_state.go", "MicrophoneState"); got != false {
		t.Fatalf("MicrophoneState xna:flags directive = %t, want false", got)
	}
}
