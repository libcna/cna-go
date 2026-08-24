package framework

import (
	"math"
	"testing"
)

func requireFloatBits(t *testing.T, got float32, want uint32) {
	t.Helper()
	if bits := math.Float32bits(got); bits != want {
		t.Fatalf("bits(%v) = %08X, want %08X", got, bits, want)
	}
}

func requireVector3Bits(t *testing.T, got Vector3, want [3]uint32) {
	t.Helper()
	requireFloatBits(t, got.X, want[0])
	requireFloatBits(t, got.Y, want[1])
	requireFloatBits(t, got.Z, want[2])
}

func TestVector2NormalizeZeroMatchesXNA(t *testing.T) {
	got := Vector2NormalizeByVector2(Vector2Zero())
	requireFloatBits(t, got.X, 0xFFC00000)
	requireFloatBits(t, got.Y, 0xFFC00000)
}

func TestVector2TransformOverlapUsesForwardMutationOrder(t *testing.T) {
	values := []Vector2{{X: 1}, {X: 2}, {X: 3}}
	matrix := MatrixCreateTranslationBySingleAndSingleAndSingle(10, 0, 0)
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(values, 0, &matrix, values, 1, 2)
	if values[1].X != 11 || values[2].X != 21 {
		t.Fatalf("forward overlap = %v", values)
	}
}

func TestVectorArrayTransformRangeAndStorageSemantics(t *testing.T) {
	matrix := MatrixCreateTranslationBySingleAndSingleAndSingle(10, 20, 0)
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(
		[]Vector2{}, 0, &matrix, []Vector2{}, 0, 0)

	one := []Vector2{{X: 1, Y: 2}}
	destination := make([]Vector2, 1)
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(
		one, 0, &matrix, destination, 0, 1)
	if destination[0] != (Vector2{X: 11, Y: 22}) {
		t.Fatalf("one-element transform = %v", destination)
	}

	same := []Vector2{{X: 1}, {X: 2}}
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(
		same, 0, &matrix, same, 0, 2)
	if same[0].X != 11 || same[1].X != 12 {
		t.Fatalf("same-slice transform = %v", same)
	}

	backward := []Vector2{{X: 1}, {X: 2}, {X: 3}}
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(
		backward, 1, &matrix, backward, 0, 2)
	if backward[0].X != 12 || backward[1].X != 13 || backward[2].X != 3 {
		t.Fatalf("backward overlap = %v", backward)
	}

	expectPanic := func(name string, operation func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s did not panic", name)
			}
		}()
		operation()
	}
	expectPanic("nil source", func() {
		Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(nil, 0, &matrix, []Vector2{}, 0, 0)
	})
	expectPanic("source too short", func() {
		Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(one, 0, &matrix, destination, 0, 2)
	})
	expectPanic("destination too short", func() {
		Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(one, 0, &matrix, []Vector2{}, 0, 1)
	})
	expectPanic("negative destination index", func() {
		Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(one, 0, &matrix, destination, -1, 1)
	})
}

func TestVector2EqualityRejectsNaN(t *testing.T) {
	v := Vector2{X: float32(math.NaN())}
	if v.EqualsByVector2(v) || Vector2OperatorEqualityByVector2AndVector2(v, v) {
		t.Fatal("NaN vector compared equal")
	}
}
