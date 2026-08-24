package framework

import "testing"

func TestVector3DirectionsAndCrossMatchXNAHandedness(t *testing.T) {
	if got := Vector3Forward(); got != (Vector3{Z: -1}) {
		t.Fatalf("Forward = %v", got)
	}
	if got := Vector3CrossByVector3AndVector3(Vector3Right(), Vector3Up()); got != Vector3Backward() {
		t.Fatalf("Right x Up = %v, want Backward", got)
	}
}

func TestVector3TransformNegativeLengthAndIndexSemantics(t *testing.T) {
	matrix := MatrixIdentity()
	Vector3TransformBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(
		[]Vector3{{}}, 0, &matrix, []Vector3{{}}, 0, -1)
	deferred := func() { _ = recover() }
	defer deferred()
	Vector3TransformBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(
		[]Vector3{{}}, -1, &matrix, []Vector3{{}}, 0, 1)
	t.Fatal("negative source index did not panic")
}

func TestVector3HashMatchesXNA(t *testing.T) {
	if got := (Vector3{1, 2, 3}).GetHashCode(); got != -1077936128 {
		t.Fatalf("hash = %d", got)
	}
}
