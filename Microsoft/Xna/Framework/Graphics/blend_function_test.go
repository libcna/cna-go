package graphics

import (
	"reflect"
	"testing"
)

func TestBlendFunctionCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value BlendFunction
		want  int32
	}{
		{"Add", BlendFunctionAdd, 0},
		{"Subtract", BlendFunctionSubtract, 1},
		{"ReverseSubtract", BlendFunctionReverseSubtract, 2},
		{"Min", BlendFunctionMin, 3},
		{"Max", BlendFunctionMax, 4},
	}
	if len(values) != 5 {
		t.Fatalf("BlendFunction literal count = %d, want 5", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("BlendFunction%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("BlendFunction%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(BlendFunctionAdd).Kind(); got != reflect.Int32 {
		t.Fatalf("BlendFunction underlying kind = %s, want int32", got)
	}
}

func TestBlendFunctionZeroAndArbitraryRawValues(t *testing.T) {
	var zero BlendFunction
	if zero != BlendFunctionAdd {
		t.Fatalf("zero BlendFunction = %d, want Add (%d)", zero, BlendFunctionAdd)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(BlendFunction(raw)); got != raw {
			t.Fatalf("BlendFunction(%d) = %d", raw, got)
		}
	}
}

func TestBlendFunctionSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "blend_function.go", "BlendFunction"); got != false {
		t.Fatalf("BlendFunction xna:flags directive = %t, want false", got)
	}
}
