package framework

import "testing"

func TestPlaneXnaGoldens(t *testing.T) {
	degenerate := NewPlaneByVector3AndVector3AndVector3(Vector3Zero(), Vector3Zero(), Vector3Zero())
	for i, value := range []float32{degenerate.Normal.X, degenerate.Normal.Y, degenerate.Normal.Z, degenerate.D} {
		requireFloatBits(t, value, [4]uint32{0xFFC00000, 0xFFC00000, 0xFFC00000, 0x7FC00000}[i])
	}
	nearUnit := PlaneNormalizeByPlane(NewPlaneByVector3AndSingle(Vector3{0.6, 0.79999995, 0}, 2))
	for i, value := range []float32{nearUnit.Normal.X, nearUnit.Normal.Y, nearUnit.Normal.Z, nearUnit.D} {
		requireFloatBits(t, value, [4]uint32{0x3F19999A, 0x3F4CCCCC, 0, 0x40000000}[i])
	}
	box := NewBoundingBox(NewVector3BySingle(-1), NewVector3BySingle(1))
	if got := (Plane{}).IntersectsByBoundingBox(box); got != PlaneIntersectionTypeIntersecting {
		t.Fatalf("coplanar box classification = %d", got)
	}
}

func TestMatrixReflectionRefNormalizesInput(t *testing.T) {
	plane := NewPlaneByVector3AndSingle(Vector3{X: 2}, 4)
	matrix := MatrixCreateReflectionByRefPlaneAndOutMatrix(&plane)
	if plane.Normal.X != 1 || plane.D != 2 || matrix.M11 != -1 || matrix.M41 != -4 {
		t.Fatalf("plane=%v matrix=%v", plane, matrix)
	}
}

func TestPlaneMatrixAndQuaternionTransforms(t *testing.T) {
	plane := NewPlaneByVector3AndSingle(Vector3UnitX(), -2)
	translated := PlaneTransformByPlaneAndMatrix(plane, MatrixCreateTranslationBySingleAndSingleAndSingle(3, 0, 0))
	if translated != (Plane{Normal: Vector3UnitX(), D: -5}) {
		t.Fatalf("translated plane = %v", translated)
	}
	rotation := QuaternionCreateFromAxisAngleByVector3AndSingle(Vector3Up(), MathHelperPiOver2)
	rotated := PlaneTransformByPlaneAndQuaternion(NewPlaneByVector3AndSingle(Vector3Forward(), 4), rotation)
	reference := Vector3TransformByVector3AndQuaternion(Vector3Forward(), rotation)
	if rotated.Normal != reference || rotated.D != 4 {
		t.Fatalf("rotated plane = %v, normal want %v", rotated, reference)
	}
}
