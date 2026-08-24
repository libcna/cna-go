package framework

import "testing"

func TestBoundingSphereXnaContainmentAndTangentSemantics(t *testing.T) {
	sphere := NewBoundingSphere(Vector3Zero(), 1)
	if got := sphere.ContainsByVector3(Vector3UnitX()); got != ContainmentTypeDisjoint {
		t.Fatalf("edge containment = %d", got)
	}
	if sphere.IntersectsByBoundingSphere(NewBoundingSphere(Vector3{X: 2}, 1)) {
		t.Fatal("externally tangent spheres should not intersect in XNA")
	}
}

func TestBoundingSphereCreateFromPointsXnaGolden(t *testing.T) {
	got := BoundingSphereCreateFromPoints([]Vector3{{-4, 1, 0}, {6, -2, 3}, {0, 8, -5}, {2, 0, 9}})
	values := []float32{got.Center.X, got.Center.Y, got.Center.Z, got.Radius}
	want := []uint32{0x3F800000, 0x40800000, 0x40000000, 0x4101FC10}
	for i := range values {
		requireFloatBits(t, values[i], want[i])
	}
}

func TestBoundingSphereRejectsNegativeRadius(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("negative radius did not panic")
		}
	}()
	_ = NewBoundingSphere(Vector3Zero(), -1)
}

func TestBoundingSphereDegenerateAndTransformScaleSemantics(t *testing.T) {
	point := Vector3{X: 2, Y: -3, Z: 4}
	degenerate := BoundingSphereCreateFromPoints([]Vector3{point, point})
	if degenerate.Center != point || degenerate.Radius != 0 || degenerate.ContainsByVector3(point) != ContainmentTypeDisjoint {
		t.Fatalf("degenerate sphere = %v, containment=%v", degenerate, degenerate.ContainsByVector3(point))
	}
	matrix := MatrixMultiplyByMatrixAndMatrix(
		MatrixCreateScaleBySingleAndSingleAndSingle(-2, 3, 4),
		MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	transformed := NewBoundingSphere(Vector3{X: 1, Y: 2, Z: 3}, 2).TransformByMatrix(matrix)
	if transformed.Center != (Vector3{X: 3, Y: 12, Z: 19}) || transformed.Radius != 8 {
		t.Fatalf("nonuniform reflected transform = %v", transformed)
	}
}
