package framework

import "testing"

func newGoldenFrustum() *BoundingFrustum {
	projection := MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(MathHelperPiOver4, 4.0/3.0, 1, 10)
	view := MatrixCreateLookAtByVector3AndVector3AndVector3(Vector3{Z: 5}, Vector3Zero(), Vector3Up())
	return NewBoundingFrustum(MatrixMultiplyByMatrixAndMatrix(view, projection))
}

func TestBoundingFrustumPlanesCornersAndClassIdentity(t *testing.T) {
	f := newGoldenFrustum()
	near := f.Near()
	for i, value := range []float32{near.Normal.X, near.Normal.Y, near.Normal.Z, near.D} {
		requireFloatBits(t, value, [4]uint32{0x80000000, 0x80000000, 0x3F800000, 0xC0800000}[i])
	}
	top := f.Top()
	for i, value := range []float32{top.Normal.X, top.Normal.Y, top.Normal.Z, top.D} {
		requireFloatBits(t, value, [4]uint32{0, 0x3F6C835F, 0x3EC3EF16, 0xBFF4EADB}[i])
	}
	corners := f.GetCornersByNone()
	requireVector3Bits(t, corners[0], [3]uint32{0xBF0D6289, 0x3ED413CB, 0x40800000})
	requireVector3Bits(t, corners[6], [3]uint32{0x40B0BB28, 0xC0848C5D, 0xC09FFFF8})
	if BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(f, NewBoundingFrustum(f.Matrix())) != true {
		t.Fatal("equal frustum matrices did not compare equal")
	}
	if BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(f, nil) {
		t.Fatal("non-nil frustum compared equal to nil")
	}
}

func TestBoundingFrustumContainmentGJKAndRayGoldens(t *testing.T) {
	f := newGoldenFrustum()
	insideBox := NewBoundingBox(NewVector3BySingle(-0.5), NewVector3BySingle(0.5))
	outsideBox := NewBoundingBox(NewVector3BySingle(100), NewVector3BySingle(101))
	insideSphere := NewBoundingSphere(Vector3Zero(), 0.5)
	outsideSphere := NewBoundingSphere(NewVector3BySingle(100), 0.5)
	projection := MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(MathHelperPiOver4, 4.0/3.0, 1, 10)
	distant := NewBoundingFrustum(MatrixMultiplyByMatrixAndMatrix(
		MatrixCreateLookAtByVector3AndVector3AndVector3(Vector3{100, 0, 5}, Vector3{100, 0, 0}, Vector3Up()), projection))
	got := []bool{f.IntersectsByBoundingBox(insideBox), f.IntersectsByBoundingBox(outsideBox), f.IntersectsByBoundingSphere(insideSphere), f.IntersectsByBoundingSphere(outsideSphere), f.IntersectsByBoundingFrustum(distant)}
	want := []bool{true, false, true, false, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("GJK result %d = %t, want %t", i, got[i], want[i])
		}
	}
	distance, ok := f.IntersectsByRay(NewRay(Vector3{Z: 20}, Vector3Forward()))
	if !ok {
		t.Fatal("frustum ray reported null")
	}
	requireFloatBits(t, distance, 0x41800000)
}
