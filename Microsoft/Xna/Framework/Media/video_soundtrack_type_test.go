package media

import (
	"reflect"
	"testing"
)

func TestVideoSoundtrackTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value VideoSoundtrackType
		want  int32
	}{
		{"Music", VideoSoundtrackTypeMusic, 0},
		{"Dialog", VideoSoundtrackTypeDialog, 1},
		{"MusicAndDialog", VideoSoundtrackTypeMusicAndDialog, 2},
	}
	if len(values) != 3 {
		t.Fatalf("VideoSoundtrackType literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("VideoSoundtrackType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("VideoSoundtrackType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(VideoSoundtrackTypeMusic).Kind(); got != reflect.Int32 {
		t.Fatalf("VideoSoundtrackType underlying kind = %s, want int32", got)
	}
}

func TestVideoSoundtrackTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero VideoSoundtrackType
	if zero != VideoSoundtrackTypeMusic {
		t.Fatalf("zero VideoSoundtrackType = %d, want Music (%d)", zero, VideoSoundtrackTypeMusic)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(VideoSoundtrackType(raw)); got != raw {
			t.Fatalf("VideoSoundtrackType(%d) = %d", raw, got)
		}
	}
}

func TestVideoSoundtrackTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "video_soundtrack_type.go", "VideoSoundtrackType"); got != false {
		t.Fatalf("VideoSoundtrackType xna:flags directive = %t, want false", got)
	}
}
