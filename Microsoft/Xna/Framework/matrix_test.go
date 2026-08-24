package framework

import (
	"math"
	"testing"
)

func TestMatrixConventionsAndXnaGoldens(t *testing.T) {
	matrix := MatrixMultiplyByMatrixAndMatrix(MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4), MatrixCreateRotationYBySingle(0.25))
	matrix = MatrixMultiplyByMatrixAndMatrix(matrix, MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	if matrix.Translation() != (Vector3{5, 6, 7}) {
		t.Fatalf("translation = %v", matrix.Translation())
	}
	product := MatrixMultiplyByMatrixAndMatrix(matrix, MatrixInvertByMatrix(matrix))
	want := [16]uint32{0x3F800000, 0, 0xB2000000, 0, 0, 0x3F800000, 0, 0, 0x33000000, 0, 0x3F800000, 0, 0x34000000, 0, 0, 0x3F800000}
	values := [16]float32{product.M11, product.M12, product.M13, product.M14, product.M21, product.M22, product.M23, product.M24, product.M31, product.M32, product.M33, product.M34, product.M41, product.M42, product.M43, product.M44}
	for i := range values {
		requireFloatBits(t, values[i], want[i])
	}
	if MatrixIdentity().GetHashCode() != -33554432 {
		t.Fatalf("identity hash = %d", MatrixIdentity().GetHashCode())
	}
}

func TestMatrixSingularInvertAndInfinityMatchXNA(t *testing.T) {
	singular := MatrixInvertByMatrix(Matrix{})
	for _, value := range []float32{singular.M11, singular.M22, singular.M33, singular.M44} {
		if !math.IsNaN(float64(value)) {
			t.Fatalf("singular inverse component = %v", value)
		}
	}
	perspective := MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(4, 3, 0.1, float32(math.Inf(1)))
	requireFloatBits(t, perspective.M33, 0xFFC00000)
	requireFloatBits(t, perspective.M43, 0xFFC00000)
	rotation := MatrixCreateRotationYBySingle(123456.789)
	requireFloatBits(t, rotation.M11, 0x3D53E807)
	requireFloatBits(t, rotation.M31, 0xBF7FA83D)
}

func TestMatrixDecomposeMirroredXnaGolden(t *testing.T) {
	m := MatrixMultiplyByMatrixAndMatrix(MatrixCreateScaleBySingleAndSingleAndSingle(-2, 3, 4), MatrixCreateRotationYBySingle(0.25))
	m = MatrixMultiplyByMatrixAndMatrix(m, MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	ok, scale, rotation, translation := m.Decompose()
	if !ok {
		t.Fatal("mirrored matrix did not decompose")
	}
	want := [10]uint32{0x40000000, 0x40400000, 0xC0800000, 0, 0x3F7E00AA, 0, 0xBDFF5579, 0x40A00000, 0x40C00000, 0x40E00000}
	values := [10]float32{scale.X, scale.Y, scale.Z, rotation.X, rotation.Y, rotation.Z, rotation.W, translation.X, translation.Y, translation.Z}
	for i := range values {
		requireFloatBits(t, values[i], want[i])
	}
}

func TestMatrixToStringMatchesXNA(t *testing.T) {
	want := "{ {M11:1 M12:0 M13:0 M14:0} {M21:0 M22:1 M23:0 M24:0} {M31:0 M32:0 M33:1 M34:0} {M41:0 M42:0 M43:0 M44:1} }"
	if got := MatrixIdentity().ToString(); got != want {
		t.Fatalf("ToString = %q", got)
	}
}

func TestMatrixInvertAndDecomposeFamilies(t *testing.T) {
	matrices := []Matrix{
		MatrixIdentity(),
		MatrixCreateTranslationBySingleAndSingleAndSingle(3, -4, 5),
		MatrixCreateScaleBySingle(2),
		MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4),
		MatrixCreateRotationXBySingle(0.37),
		MatrixMultiplyByMatrixAndMatrix(
			MatrixMultiplyByMatrixAndMatrix(MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4), MatrixCreateRotationZBySingle(-0.61)),
			MatrixCreateTranslationBySingleAndSingleAndSingle(7, 8, -9)),
		MatrixCreateScaleBySingleAndSingleAndSingle(1e-20, 2, 3),
	}
	for i, matrix := range matrices {
		product := MatrixMultiplyByMatrixAndMatrix(matrix, MatrixInvertByMatrix(matrix))
		values := []float32{product.M11 - 1, product.M12, product.M13, product.M14, product.M21, product.M22 - 1, product.M23, product.M24, product.M31, product.M32, product.M33 - 1, product.M34, product.M41, product.M42, product.M43, product.M44 - 1}
		for _, value := range values {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || math.Abs(float64(value)) > 2e-5 {
				t.Fatalf("inverse family %d residual = %v; product=%v", i, value, product)
			}
		}
	}

	translationOnly := MatrixCreateTranslationBySingleAndSingleAndSingle(3, 4, 5)
	ok, scale, rotation, translation := translationOnly.Decompose()
	if !ok || scale != Vector3One() || rotation != QuaternionIdentity() || translation != (Vector3{3, 4, 5}) {
		t.Fatalf("translation decompose = %t %v %v %v", ok, scale, rotation, translation)
	}
	nonUniform := MatrixMultiplyByMatrixAndMatrix(MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4), MatrixCreateRotationYBySingle(0.3))
	ok, scale, _, translation = nonUniform.Decompose()
	if !ok || scale != (Vector3{2, 3, 4}) || translation != Vector3Zero() {
		t.Fatalf("nonuniform decompose = %t %v %v", ok, scale, translation)
	}
	shear := MatrixIdentity()
	shear.M12 = 1
	ok, _, rotation, _ = shear.Decompose()
	if ok || rotation != QuaternionIdentity() {
		t.Fatalf("shear decompose = %t %v", ok, rotation)
	}
}

func TestMatrixInvalidProjectionAndNonFiniteInvert(t *testing.T) {
	expectPanic := func(name string, operation func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s did not panic", name)
			}
		}()
		operation()
	}
	expectPanic("zero field of view", func() {
		MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(0, 1, 0.1, 100)
	})
	expectPanic("inverted planes", func() {
		MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(4, 3, 10, 1)
	})

	nanMatrix := MatrixIdentity()
	nanMatrix.M11 = float32(math.NaN())
	if got := MatrixInvertByMatrix(nanMatrix); !math.IsNaN(float64(got.M11)) {
		t.Fatalf("NaN inverse M11 = %v", got.M11)
	}
	infiniteMatrix := MatrixIdentity()
	infiniteMatrix.M11 = float32(math.Inf(1))
	got := MatrixInvertByMatrix(infiniteMatrix)
	if got.M11 != 0 || !math.IsNaN(float64(got.M22)) {
		t.Fatalf("infinite inverse = %v", got)
	}
}
