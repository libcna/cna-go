package graphics

import (
	"reflect"
	"testing"
)

func TestEffectParameterClassCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value EffectParameterClass
		want  int32
	}{
		{"Scalar", EffectParameterClassScalar, 0},
		{"Vector", EffectParameterClassVector, 1},
		{"Matrix", EffectParameterClassMatrix, 2},
		{"Object", EffectParameterClassObject, 3},
		{"Struct", EffectParameterClassStruct, 4},
	}
	if len(values) != 5 {
		t.Fatalf("EffectParameterClass literal count = %d, want 5", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("EffectParameterClass%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("EffectParameterClass%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(EffectParameterClassScalar).Kind(); got != reflect.Int32 {
		t.Fatalf("EffectParameterClass underlying kind = %s, want int32", got)
	}
}

func TestEffectParameterClassZeroAndArbitraryRawValues(t *testing.T) {
	var zero EffectParameterClass
	if zero != EffectParameterClassScalar {
		t.Fatalf("zero EffectParameterClass = %d, want Scalar (%d)", zero, EffectParameterClassScalar)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(EffectParameterClass(raw)); got != raw {
			t.Fatalf("EffectParameterClass(%d) = %d", raw, got)
		}
	}
}

func TestEffectParameterClassSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "effect_parameter_class.go", "EffectParameterClass"); got != false {
		t.Fatalf("EffectParameterClass xna:flags directive = %t, want false", got)
	}
}
