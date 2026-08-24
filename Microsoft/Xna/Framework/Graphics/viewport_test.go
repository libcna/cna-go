package graphics

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func requireViewportBits(t *testing.T, got float32, want uint32) {
	t.Helper()
	if bits := math.Float32bits(got); bits != want {
		t.Fatalf("bits(%v) = %08X, want %08X", got, bits, want)
	}
}

func TestViewportProjectUnprojectXnaGoldens(t *testing.T) {
	v := NewViewportByInt32AndInt32AndInt32AndInt32(11, 13, 640, 360)
	v.SetMinDepth(0.2)
	v.SetMaxDepth(0.9)
	world := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(1.5, 0.75, 2), framework.MatrixCreateRotationYBySingle(0.31))
	world = framework.MatrixMultiplyByMatrixAndMatrix(world, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(2, -1, 0.5))
	view := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3{X: 4, Y: 3, Z: 8}, framework.Vector3Zero(), framework.Vector3Up())
	projection := framework.MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(0.9, 16.0/9.0, 0.1, 100)
	projected := v.Project(framework.Vector3{X: 0.25, Y: -0.5, Z: 1.25}, projection, view, world)
	for i, value := range []float32{projected.X, projected.Y, projected.Z} {
		requireViewportBits(t, value, [3]uint32{0x43D42808, 0x43AC9F3C, 0x3F63AFF4}[i])
	}
	unprojected := v.Unproject(projected, projection, view, world)
	for i, value := range []float32{unprojected.X, unprojected.Y, unprojected.Z} {
		requireViewportBits(t, value, [3]uint32{0x3E7FFE10, 0xBEFFF906, 0x3FA00111}[i])
	}
}

func TestViewportUnprojectSingularReturnsNaNs(t *testing.T) {
	v := NewViewportByInt32AndInt32AndInt32AndInt32(11, 13, 640, 360)
	v.SetMinDepth(0.2)
	v.SetMaxDepth(0.9)
	got := v.Unproject(framework.Vector3{X: 100, Y: 50, Z: 0.5}, framework.MatrixIdentity(), framework.MatrixIdentity(), framework.Matrix{})
	for _, value := range []float32{got.X, got.Y, got.Z} {
		requireViewportBits(t, value, 0xFFC00000)
	}
}

func TestViewportProjectDepthAndOffsetBoundaries(t *testing.T) {
	v := NewViewportByInt32AndInt32AndInt32AndInt32(11, 13, 640, 360)
	v.SetMinDepth(0.2)
	v.SetMaxDepth(0.9)
	identity := framework.MatrixIdentity()
	minimum := v.Project(framework.Vector3{X: -1, Y: 1, Z: 0}, identity, identity, identity)
	if minimum.X != 11 || minimum.Y != 13 || minimum.Z != v.MinDepth() {
		t.Fatalf("minimum boundary = %v", minimum)
	}
	maximum := v.Project(framework.Vector3{X: 1, Y: -1, Z: 1}, identity, identity, identity)
	if maximum.X != 651 || maximum.Y != 373 || maximum.Z != v.MaxDepth() {
		t.Fatalf("maximum boundary = %v", maximum)
	}
	if got := v.Unproject(minimum, identity, identity, identity); got != (framework.Vector3{X: -1, Y: 1, Z: 0}) {
		t.Fatalf("minimum unproject = %v", got)
	}
}
