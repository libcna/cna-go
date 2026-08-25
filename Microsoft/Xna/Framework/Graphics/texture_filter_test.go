package graphics

import (
	"reflect"
	"testing"
)

func TestTextureFilterCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value TextureFilter
		want  int32
	}{
		{"Linear", TextureFilterLinear, 0},
		{"Point", TextureFilterPoint, 1},
		{"Anisotropic", TextureFilterAnisotropic, 2},
		{"LinearMipPoint", TextureFilterLinearMipPoint, 3},
		{"PointMipLinear", TextureFilterPointMipLinear, 4},
		{"MinLinearMagPointMipLinear", TextureFilterMinLinearMagPointMipLinear, 5},
		{"MinLinearMagPointMipPoint", TextureFilterMinLinearMagPointMipPoint, 6},
		{"MinPointMagLinearMipLinear", TextureFilterMinPointMagLinearMipLinear, 7},
		{"MinPointMagLinearMipPoint", TextureFilterMinPointMagLinearMipPoint, 8},
	}
	if len(values) != 9 {
		t.Fatalf("TextureFilter literal count = %d, want 9", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("TextureFilter%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("TextureFilter%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(TextureFilterLinear).Kind(); got != reflect.Int32 {
		t.Fatalf("TextureFilter underlying kind = %s, want int32", got)
	}
}

func TestTextureFilterZeroAndArbitraryRawValues(t *testing.T) {
	var zero TextureFilter
	if zero != TextureFilterLinear {
		t.Fatalf("zero TextureFilter = %d, want Linear (%d)", zero, TextureFilterLinear)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(TextureFilter(raw)); got != raw {
			t.Fatalf("TextureFilter(%d) = %d", raw, got)
		}
	}
}

func TestTextureFilterSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "texture_filter.go", "TextureFilter"); got != false {
		t.Fatalf("TextureFilter xna:flags directive = %t, want false", got)
	}
}
