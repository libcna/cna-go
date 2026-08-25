package graphics

import (
	"reflect"
	"testing"
)

func TestGraphicsProfileCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value GraphicsProfile
		want  int32
	}{
		{"Reach", GraphicsProfileReach, 0},
		{"HiDef", GraphicsProfileHiDef, 1},
	}
	for _, item := range values {
		if got := int32(item.value); got != item.want {
			t.Errorf("GraphicsProfile%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(GraphicsProfileReach).Kind(); got != reflect.Int32 {
		t.Fatalf("GraphicsProfile underlying kind = %s, want int32", got)
	}
}

func TestGraphicsProfileZeroAndArbitraryRawValues(t *testing.T) {
	var zero GraphicsProfile
	if zero != GraphicsProfileReach {
		t.Fatalf("zero GraphicsProfile = %d, want Reach (%d)", zero, GraphicsProfileReach)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(GraphicsProfile(raw)); got != raw {
			t.Fatalf("GraphicsProfile(%d) = %d", raw, got)
		}
	}
}
