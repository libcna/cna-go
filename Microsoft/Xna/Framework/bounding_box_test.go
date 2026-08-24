package framework

import (
	"math"
	"testing"
)

func TestBoundingBoxContainmentAndNaNMatchXNA(t *testing.T) {
	box := NewBoundingBox(NewVector3BySingle(-1), NewVector3BySingle(1))
	if got := box.ContainsByVector3(Vector3{X: 1}); got != ContainmentTypeContains {
		t.Fatalf("edge containment = %d", got)
	}
	nan := float32(math.NaN())
	if got := box.ContainsByVector3(Vector3{X: nan}); got != ContainmentTypeDisjoint {
		t.Fatalf("NaN point containment = %d", got)
	}
	nanBox := NewBoundingBox(Vector3{nan, -1, -1}, Vector3{nan, 1, 1})
	if !box.IntersectsByBoundingBox(nanBox) {
		t.Fatal("XNA NaN box intersection should be true")
	}
}

func TestBoundingBoxCornerOrder(t *testing.T) {
	box := NewBoundingBox(Vector3{1, 2, 3}, Vector3{4, 5, 6})
	want := []Vector3{{1, 5, 6}, {4, 5, 6}, {4, 2, 6}, {1, 2, 6}, {1, 5, 3}, {4, 5, 3}, {4, 2, 3}, {1, 2, 3}}
	got := box.GetCornersByNone()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("corner %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoundingBoxCreationAndMerge(t *testing.T) {
	fromPoints := BoundingBoxCreateFromPoints([]Vector3{{X: 2, Y: -1, Z: 5}, {X: -3, Y: 4, Z: 1}})
	if fromPoints.Min != (Vector3{X: -3, Y: -1, Z: 1}) || fromPoints.Max != (Vector3{X: 2, Y: 4, Z: 5}) {
		t.Fatalf("from points = %v", fromPoints)
	}
	fromSphere := BoundingBoxCreateFromSphereByBoundingSphere(NewBoundingSphere(Vector3{X: 2}, 3))
	if fromSphere.Min != (Vector3{X: -1, Y: -3, Z: -3}) || fromSphere.Max != (Vector3{X: 5, Y: 3, Z: 3}) {
		t.Fatalf("from sphere = %v", fromSphere)
	}
	merged := BoundingBoxCreateMergedByBoundingBoxAndBoundingBox(fromPoints, fromSphere)
	if merged.Min != (Vector3{X: -3, Y: -3, Z: -3}) || merged.Max != (Vector3{X: 5, Y: 4, Z: 5}) {
		t.Fatalf("merged = %v", merged)
	}
}
