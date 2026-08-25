package graphics

import (
	"reflect"
	"testing"
)

func TestIndexElementSizeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value IndexElementSize
		want  int32
	}{
		{"SixteenBits", IndexElementSizeSixteenBits, 0},
		{"ThirtyTwoBits", IndexElementSizeThirtyTwoBits, 1},
	}
	if len(values) != 2 {
		t.Fatalf("IndexElementSize literal count = %d, want 2", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("IndexElementSize%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("IndexElementSize%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(IndexElementSizeSixteenBits).Kind(); got != reflect.Int32 {
		t.Fatalf("IndexElementSize underlying kind = %s, want int32", got)
	}
}

func TestIndexElementSizeZeroAndArbitraryRawValues(t *testing.T) {
	var zero IndexElementSize
	if zero != IndexElementSizeSixteenBits {
		t.Fatalf("zero IndexElementSize = %d, want SixteenBits (%d)", zero, IndexElementSizeSixteenBits)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(IndexElementSize(raw)); got != raw {
			t.Fatalf("IndexElementSize(%d) = %d", raw, got)
		}
	}
}

func TestIndexElementSizeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "index_element_size.go", "IndexElementSize"); got != false {
		t.Fatalf("IndexElementSize xna:flags directive = %t, want false", got)
	}
}
