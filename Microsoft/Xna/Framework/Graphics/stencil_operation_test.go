package graphics

import (
	"reflect"
	"testing"
)

func TestStencilOperationCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value StencilOperation
		want  int32
	}{
		{"Keep", StencilOperationKeep, 0},
		{"Zero", StencilOperationZero, 1},
		{"Replace", StencilOperationReplace, 2},
		{"Increment", StencilOperationIncrement, 3},
		{"Decrement", StencilOperationDecrement, 4},
		{"IncrementSaturation", StencilOperationIncrementSaturation, 5},
		{"DecrementSaturation", StencilOperationDecrementSaturation, 6},
		{"Invert", StencilOperationInvert, 7},
	}
	if len(values) != 8 {
		t.Fatalf("StencilOperation literal count = %d, want 8", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("StencilOperation%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("StencilOperation%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(StencilOperationKeep).Kind(); got != reflect.Int32 {
		t.Fatalf("StencilOperation underlying kind = %s, want int32", got)
	}
}

func TestStencilOperationZeroAndArbitraryRawValues(t *testing.T) {
	var zero StencilOperation
	if zero != StencilOperationKeep {
		t.Fatalf("zero StencilOperation = %d, want Keep (%d)", zero, StencilOperationKeep)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(StencilOperation(raw)); got != raw {
			t.Fatalf("StencilOperation(%d) = %d", raw, got)
		}
	}
}

func TestStencilOperationSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "stencil_operation.go", "StencilOperation"); got != false {
		t.Fatalf("StencilOperation xna:flags directive = %t, want false", got)
	}
}
