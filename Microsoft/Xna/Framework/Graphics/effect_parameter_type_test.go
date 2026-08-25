package graphics

import (
	"reflect"
	"testing"
)

func TestEffectParameterTypeCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value EffectParameterType
		want  int32
	}{
		{"Void", EffectParameterTypeVoid, 0},
		{"Bool", EffectParameterTypeBool, 1},
		{"Int32", EffectParameterTypeInt32, 2},
		{"Single", EffectParameterTypeSingle, 3},
		{"String", EffectParameterTypeString, 4},
		{"Texture", EffectParameterTypeTexture, 5},
		{"Texture1D", EffectParameterTypeTexture1D, 6},
		{"Texture2D", EffectParameterTypeTexture2D, 7},
		{"Texture3D", EffectParameterTypeTexture3D, 8},
		{"TextureCube", EffectParameterTypeTextureCube, 9},
	}
	if len(values) != 10 {
		t.Fatalf("EffectParameterType literal count = %d, want 10", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("EffectParameterType%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("EffectParameterType%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(EffectParameterTypeVoid).Kind(); got != reflect.Int32 {
		t.Fatalf("EffectParameterType underlying kind = %s, want int32", got)
	}
}

func TestEffectParameterTypeZeroAndArbitraryRawValues(t *testing.T) {
	var zero EffectParameterType
	if zero != EffectParameterTypeVoid {
		t.Fatalf("zero EffectParameterType = %d, want Void (%d)", zero, EffectParameterTypeVoid)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(EffectParameterType(raw)); got != raw {
			t.Fatalf("EffectParameterType(%d) = %d", raw, got)
		}
	}
}

func TestEffectParameterTypeSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "effect_parameter_type.go", "EffectParameterType"); got != false {
		t.Fatalf("EffectParameterType xna:flags directive = %t, want false", got)
	}
}
