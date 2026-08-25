package graphics

import (
	"reflect"
	"testing"
)

func TestCompareFunctionCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value CompareFunction
		want  int32
	}{
		{"Always", CompareFunctionAlways, 0},
		{"Never", CompareFunctionNever, 1},
		{"Less", CompareFunctionLess, 2},
		{"LessEqual", CompareFunctionLessEqual, 3},
		{"Equal", CompareFunctionEqual, 4},
		{"GreaterEqual", CompareFunctionGreaterEqual, 5},
		{"Greater", CompareFunctionGreater, 6},
		{"NotEqual", CompareFunctionNotEqual, 7},
	}
	if len(values) != 8 {
		t.Fatalf("CompareFunction literal count = %d, want 8", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("CompareFunction%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("CompareFunction%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(CompareFunctionAlways).Kind(); got != reflect.Int32 {
		t.Fatalf("CompareFunction underlying kind = %s, want int32", got)
	}
}

func TestCompareFunctionZeroAndArbitraryRawValues(t *testing.T) {
	var zero CompareFunction
	if zero != CompareFunctionAlways {
		t.Fatalf("zero CompareFunction = %d, want Always (%d)", zero, CompareFunctionAlways)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(CompareFunction(raw)); got != raw {
			t.Fatalf("CompareFunction(%d) = %d", raw, got)
		}
	}
}

func TestCompareFunctionSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "compare_function.go", "CompareFunction"); got != false {
		t.Fatalf("CompareFunction xna:flags directive = %t, want false", got)
	}
}
