package graphics

import (
	"reflect"
	"testing"
)

func TestBlendCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value Blend
		want  int32
	}{
		{"One", BlendOne, 0},
		{"Zero", BlendZero, 1},
		{"SourceColor", BlendSourceColor, 2},
		{"InverseSourceColor", BlendInverseSourceColor, 3},
		{"SourceAlpha", BlendSourceAlpha, 4},
		{"InverseSourceAlpha", BlendInverseSourceAlpha, 5},
		{"DestinationColor", BlendDestinationColor, 6},
		{"InverseDestinationColor", BlendInverseDestinationColor, 7},
		{"DestinationAlpha", BlendDestinationAlpha, 8},
		{"InverseDestinationAlpha", BlendInverseDestinationAlpha, 9},
		{"BlendFactor", BlendBlendFactor, 10},
		{"InverseBlendFactor", BlendInverseBlendFactor, 11},
		{"SourceAlphaSaturation", BlendSourceAlphaSaturation, 12},
	}
	if len(values) != 13 {
		t.Fatalf("Blend literal count = %d, want 13", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("Blend%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("Blend%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(BlendOne).Kind(); got != reflect.Int32 {
		t.Fatalf("Blend underlying kind = %s, want int32", got)
	}
}

func TestBlendZeroAndArbitraryRawValues(t *testing.T) {
	var zero Blend
	if zero != BlendOne {
		t.Fatalf("zero Blend = %d, want One (%d)", zero, BlendOne)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(Blend(raw)); got != raw {
			t.Fatalf("Blend(%d) = %d", raw, got)
		}
	}
}

func TestBlendSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "blend.go", "Blend"); got != false {
		t.Fatalf("Blend xna:flags directive = %t, want false", got)
	}
}
