package graphics

import (
	"reflect"
	"testing"
)

func TestPrimitiveTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value PrimitiveType
		want  int32
	}{
		{"TriangleList", PrimitiveTypeTriangleList, 0},
		{"TriangleStrip", PrimitiveTypeTriangleStrip, 1},
		{"LineList", PrimitiveTypeLineList, 2},
		{"LineStrip", PrimitiveTypeLineStrip, 3},
	}
	if len(values) != 4 {
		t.Fatalf("PrimitiveType literal count = %d, want 4", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("PrimitiveType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("PrimitiveType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(PrimitiveTypeTriangleList).Kind(); got != reflect.Int32 {
		t.Fatalf("PrimitiveType underlying kind = %s, want int32", got)
	}
}

func TestPrimitiveTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero PrimitiveType
	if zero != PrimitiveTypeTriangleList {
		t.Fatalf("zero PrimitiveType = %d, want TriangleList (%d)", zero, PrimitiveTypeTriangleList)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(PrimitiveType(raw)); got != raw {
			t.Fatalf("PrimitiveType(%d) = %d", raw, got)
		}
	}
}

func TestPrimitiveTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "primitive_type.go", "PrimitiveType"); got != false {
		t.Fatalf("PrimitiveType xna:flags directive = %t, want false", got)
	}
}
