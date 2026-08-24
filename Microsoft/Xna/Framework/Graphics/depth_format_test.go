package graphics

import (
	"reflect"
	"testing"
)

func TestDepthFormatCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value DepthFormat
		want  int32
	}{
		{"None", DepthFormatNone, 0},
		{"Depth16", DepthFormatDepth16, 1},
		{"Depth24", DepthFormatDepth24, 2},
		{"Depth24Stencil8", DepthFormatDepth24Stencil8, 3},
	}
	for _, item := range values {
		if got := int32(item.value); got != item.want {
			t.Errorf("DepthFormat%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(DepthFormatNone).Kind(); got != reflect.Int32 {
		t.Fatalf("DepthFormat underlying kind = %s, want int32", got)
	}
}

func TestDepthFormatZeroAndArbitraryRawValues(t *testing.T) {
	var zero DepthFormat
	if zero != DepthFormatNone {
		t.Fatalf("zero DepthFormat = %d, want None (%d)", zero, DepthFormatNone)
	}
	for _, raw := range []int32{4, 12345, -1} {
		if got := int32(DepthFormat(raw)); got != raw {
			t.Fatalf("DepthFormat(%d) = %d", raw, got)
		}
	}
}
