package graphics

import (
	"reflect"
	"testing"
)

func TestSurfaceFormatCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value SurfaceFormat
		want  int32
	}{
		{"Color", SurfaceFormatColor, 0},
		{"Bgr565", SurfaceFormatBgr565, 1},
		{"Bgra5551", SurfaceFormatBgra5551, 2},
		{"Bgra4444", SurfaceFormatBgra4444, 3},
		{"Dxt1", SurfaceFormatDxt1, 4},
		{"Dxt3", SurfaceFormatDxt3, 5},
		{"Dxt5", SurfaceFormatDxt5, 6},
		{"NormalizedByte2", SurfaceFormatNormalizedByte2, 7},
		{"NormalizedByte4", SurfaceFormatNormalizedByte4, 8},
		{"Rgba1010102", SurfaceFormatRgba1010102, 9},
		{"Rg32", SurfaceFormatRg32, 10},
		{"Rgba64", SurfaceFormatRgba64, 11},
		{"Alpha8", SurfaceFormatAlpha8, 12},
		{"Single", SurfaceFormatSingle, 13},
		{"Vector2", SurfaceFormatVector2, 14},
		{"Vector4", SurfaceFormatVector4, 15},
		{"HalfSingle", SurfaceFormatHalfSingle, 16},
		{"HalfVector2", SurfaceFormatHalfVector2, 17},
		{"HalfVector4", SurfaceFormatHalfVector4, 18},
		{"HdrBlendable", SurfaceFormatHdrBlendable, 19},
	}
	for _, item := range values {
		if got := int32(item.value); got != item.want {
			t.Errorf("SurfaceFormat%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(SurfaceFormatColor).Kind(); got != reflect.Int32 {
		t.Fatalf("SurfaceFormat underlying kind = %s, want int32", got)
	}
}

func TestSurfaceFormatZeroAndArbitraryRawValues(t *testing.T) {
	var zero SurfaceFormat
	if zero != SurfaceFormatColor {
		t.Fatalf("zero SurfaceFormat = %d, want Color (%d)", zero, SurfaceFormatColor)
	}
	for _, raw := range []int32{20, 12345, -1} {
		if got := int32(SurfaceFormat(raw)); got != raw {
			t.Fatalf("SurfaceFormat(%d) = %d", raw, got)
		}
	}
}
