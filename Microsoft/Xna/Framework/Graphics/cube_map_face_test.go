package graphics

import (
	"reflect"
	"testing"
)

func TestCubeMapFaceCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value CubeMapFace
		want  int32
	}{
		{"PositiveX", CubeMapFacePositiveX, 0},
		{"NegativeX", CubeMapFaceNegativeX, 1},
		{"PositiveY", CubeMapFacePositiveY, 2},
		{"NegativeY", CubeMapFaceNegativeY, 3},
		{"PositiveZ", CubeMapFacePositiveZ, 4},
		{"NegativeZ", CubeMapFaceNegativeZ, 5},
	}
	if len(values) != 6 {
		t.Fatalf("CubeMapFace literal count = %d, want 6", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("CubeMapFace%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("CubeMapFace%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(CubeMapFacePositiveX).Kind(); got != reflect.Int32 {
		t.Fatalf("CubeMapFace underlying kind = %s, want int32", got)
	}
}

func TestCubeMapFaceZeroAndArbitraryRawValues(t *testing.T) {
	var zero CubeMapFace
	if zero != CubeMapFacePositiveX {
		t.Fatalf("zero CubeMapFace = %d, want PositiveX (%d)", zero, CubeMapFacePositiveX)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(CubeMapFace(raw)); got != raw {
			t.Fatalf("CubeMapFace(%d) = %d", raw, got)
		}
	}
}

func TestCubeMapFaceSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "cube_map_face.go", "CubeMapFace"); got != false {
		t.Fatalf("CubeMapFace xna:flags directive = %t, want false", got)
	}
}
