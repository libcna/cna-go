package graphics

import (
	"reflect"
	"testing"
)

func TestSetDataOptionsCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value SetDataOptions
		want  int32
	}{
		{"None", SetDataOptionsNone, 0},
		{"Discard", SetDataOptionsDiscard, 1},
		{"NoOverwrite", SetDataOptionsNoOverwrite, 2},
	}
	if len(values) != 3 {
		t.Fatalf("SetDataOptions literal count = %d, want 3", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("SetDataOptions%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("SetDataOptions%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(SetDataOptionsNone).Kind(); got != reflect.Int32 {
		t.Fatalf("SetDataOptions underlying kind = %s, want int32", got)
	}
}

func TestSetDataOptionsZeroAndArbitraryRawValues(t *testing.T) {
	var zero SetDataOptions
	if zero != SetDataOptionsNone {
		t.Fatalf("zero SetDataOptions = %d, want None (%d)", zero, SetDataOptionsNone)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(SetDataOptions(raw)); got != raw {
			t.Fatalf("SetDataOptions(%d) = %d", raw, got)
		}
	}
}

func TestSetDataOptionsSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "set_data_options.go", "SetDataOptions"); got != true {
		t.Fatalf("SetDataOptions xna:flags directive = %t, want true", got)
	}
}
