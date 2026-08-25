package audio

import (
	"reflect"
	"testing"
)

func TestAudioChannelsCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value AudioChannels
		want  int32
	}{
		{"Mono", AudioChannelsMono, 1},
		{"Stereo", AudioChannelsStereo, 2},
	}
	if len(values) != 2 {
		t.Fatalf("AudioChannels literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("AudioChannels%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("AudioChannels%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(AudioChannelsMono).Kind(); got != reflect.Int32 {
		t.Fatalf("AudioChannels underlying kind = %s, want int32", got)
	}
}

func TestAudioChannelsZeroAndArbitraryRawValues(t *testing.T) {
	var zero AudioChannels
	// The pinned contract declares no zero literal for this enum, so the Go
	// zero value is an ordinary undefined raw value, not a named constant.
	if int32(zero) != 0 {
		t.Fatalf("zero AudioChannels = %d, want 0", int32(zero))
	}
	if zero == AudioChannelsMono {
		t.Fatal("zero AudioChannels unexpectedly equals Mono")
	}
	if zero == AudioChannelsStereo {
		t.Fatal("zero AudioChannels unexpectedly equals Stereo")
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(AudioChannels(raw)); got != raw {
			t.Fatalf("AudioChannels(%d) = %d", raw, got)
		}
	}
}

func TestAudioChannelsSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "audio_channels.go", "AudioChannels"); got != false {
		t.Fatalf("AudioChannels xna:flags directive = %t, want false", got)
	}
}
