package graphics

import (
	"reflect"
	"testing"
)

func TestRenderTargetUsageCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value RenderTargetUsage
		want  int32
	}{
		{"DiscardContents", RenderTargetUsageDiscardContents, 0},
		{"PreserveContents", RenderTargetUsagePreserveContents, 1},
		{"PlatformContents", RenderTargetUsagePlatformContents, 2},
	}
	if len(values) != 3 {
		t.Fatalf("RenderTargetUsage literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("RenderTargetUsage%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("RenderTargetUsage%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(RenderTargetUsageDiscardContents).Kind(); got != reflect.Int32 {
		t.Fatalf("RenderTargetUsage underlying kind = %s, want int32", got)
	}
}

func TestRenderTargetUsageZeroAndArbitraryRawValues(t *testing.T) {
	var zero RenderTargetUsage
	if zero != RenderTargetUsageDiscardContents {
		t.Fatalf("zero RenderTargetUsage = %d, want DiscardContents (%d)", zero, RenderTargetUsageDiscardContents)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(RenderTargetUsage(raw)); got != raw {
			t.Fatalf("RenderTargetUsage(%d) = %d", raw, got)
		}
	}
}

func TestRenderTargetUsageSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "render_target_usage.go", "RenderTargetUsage"); got != false {
		t.Fatalf("RenderTargetUsage xna:flags directive = %t, want false", got)
	}
}
