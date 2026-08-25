package graphics

import (
	"reflect"
	"testing"
)

func TestTextureAddressModeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value TextureAddressMode
		want  int32
	}{
		{"Wrap", TextureAddressModeWrap, 0},
		{"Clamp", TextureAddressModeClamp, 1},
		{"Mirror", TextureAddressModeMirror, 2},
	}
	if len(values) != 3 {
		t.Fatalf("TextureAddressMode literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("TextureAddressMode%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("TextureAddressMode%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(TextureAddressModeWrap).Kind(); got != reflect.Int32 {
		t.Fatalf("TextureAddressMode underlying kind = %s, want int32", got)
	}
}

func TestTextureAddressModeZeroAndArbitraryRawValues(t *testing.T) {
	var zero TextureAddressMode
	if zero != TextureAddressModeWrap {
		t.Fatalf("zero TextureAddressMode = %d, want Wrap (%d)", zero, TextureAddressModeWrap)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(TextureAddressMode(raw)); got != raw {
			t.Fatalf("TextureAddressMode(%d) = %d", raw, got)
		}
	}
}

func TestTextureAddressModeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "texture_address_mode.go", "TextureAddressMode"); got != false {
		t.Fatalf("TextureAddressMode xna:flags directive = %t, want false", got)
	}
}
