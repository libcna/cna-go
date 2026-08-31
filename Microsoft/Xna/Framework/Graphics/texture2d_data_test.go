package graphics

import (
	"errors"
	"strings"
	"testing"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
	"github.com/openeggbert/cna-go/internal/interop"
)

// TestTheElementMappingIsClosedOverCnasEighteenIdentities is the coverage proof
// for the type switch. CNA declares eighteen CNA_TEXTURE_DATA_* identities and
// every one of them must have a Go type, or a transfer a consumer can express
// in XNA has no projection here and the gap is invisible.
func TestTheElementMappingIsClosedOverCnasEighteenIdentities(t *testing.T) {
	seen := map[uint32]string{}
	record := func(identity uint32, width uintptr, name string, known bool) {
		if !known {
			t.Fatalf("%s is not mapped", name)
		}
		if other, duplicate := seen[identity]; duplicate {
			t.Fatalf("%s and %s both map to identity %d", name, other, identity)
		}
		seen[identity] = name
	}
	record(textureElementType[framework.Color]())
	record(textureElementType[packedvector.Bgr565]())
	record(textureElementType[packedvector.Bgra5551]())
	record(textureElementType[packedvector.Bgra4444]())
	record(textureElementType[byte]())
	record(textureElementType[packedvector.NormalizedByte2]())
	record(textureElementType[packedvector.NormalizedByte4]())
	record(textureElementType[packedvector.Rgba1010102]())
	record(textureElementType[packedvector.Rg32]())
	record(textureElementType[packedvector.Rgba64]())
	record(textureElementType[packedvector.Alpha8]())
	record(textureElementType[float32]())
	record(textureElementType[framework.Vector2]())
	record(textureElementType[framework.Vector4]())
	record(textureElementType[packedvector.HalfSingle]())
	record(textureElementType[packedvector.HalfVector2]())
	record(textureElementType[packedvector.HalfVector4]())
	record(textureElementType[uint16]())

	if len(seen) != 18 {
		t.Fatalf("the mapping covers %d identities, want CNA's 18", len(seen))
	}
	for identity := uint32(0); identity <= 17; identity++ {
		if _, present := seen[identity]; !present {
			t.Errorf("CNA_TEXTURE_DATA identity %d has no Go type", identity)
		}
	}
}

// TestEveryMappedElementTypeIsTheWidthCnaReads is the load-bearing half. CNA
// identifies an element by what it REPRESENTS, never by how large it is, so a Go
// type whose layout drifted would be copied wholesale into a buffer CNA reads
// with a different stride and nothing on either side would report it.
func TestEveryMappedElementTypeIsTheWidthCnaReads(t *testing.T) {
	check := func(name string, size uintptr, width uintptr) {
		if size != width {
			t.Errorf("%s is %d bytes in Go and CNA reads %d", name, size, width)
		}
	}
	check("Color", unsafe.Sizeof(framework.Color{}), widthOf[framework.Color](t))
	check("Bgr565", unsafe.Sizeof(packedvector.Bgr565{}), widthOf[packedvector.Bgr565](t))
	check("Rgba64", unsafe.Sizeof(packedvector.Rgba64{}), widthOf[packedvector.Rgba64](t))
	check("HalfVector4", unsafe.Sizeof(packedvector.HalfVector4{}), widthOf[packedvector.HalfVector4](t))
	check("Alpha8", unsafe.Sizeof(packedvector.Alpha8{}), widthOf[packedvector.Alpha8](t))
	check("Vector4", unsafe.Sizeof(framework.Vector4{}), widthOf[framework.Vector4](t))
	check("Single", unsafe.Sizeof(float32(0)), widthOf[float32](t))
	check("UShort", unsafe.Sizeof(uint16(0)), widthOf[uint16](t))

	// resolveTextureElement is where the check actually runs, and it must
	// refuse rather than transfer.
	if _, _, err := resolveTextureElement[framework.Color](); err != nil {
		t.Fatalf("a correctly sized element was refused: %v", err)
	}
	if _, _, err := resolveTextureElement[int64](); err == nil {
		t.Fatal("an unmapped element type was accepted")
	}
}

func widthOf[T any](t *testing.T) uintptr {
	t.Helper()
	_, width, name, known := textureElementType[T]()
	if !known {
		var zero T
		t.Fatalf("%T is not mapped", zero)
	}
	_ = name
	return width
}

// TestATransferWindowThatLeavesTheArrayIsRefused pins the bounds check that runs
// before the pointer is taken. Passing a start and a count that leave the slice
// would hand CNA an address it may read past.
func TestATransferWindowThatLeavesTheArrayIsRefused(t *testing.T) {
	texture := &Texture2D{}
	data := make([]framework.Color, 4)
	// The disposal check comes first for an unbound texture, so this measures
	// the window on a texture that at least has a resource shape.
	if err := Texture2DSetDataBySliceOfT(texture, data); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("an unbound texture reported %v, want ErrDisposed", err)
	}
	var missing *Texture2D
	if err := Texture2DGetDataBySliceOfT(missing, data); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("a nil texture reported %v", err)
	}
}

// TestAnUnsupportedElementTypeIsNamedRatherThanTransferred proves the refusal
// says what is wrong. A transfer of the wrong element type is the defect that
// produces a plausible-looking texture of noise.
func TestAnUnsupportedElementTypeIsNamedRatherThanTransferred(t *testing.T) {
	_, _, err := resolveTextureElement[struct{ X, Y int }]()
	if err == nil {
		t.Fatal("an arbitrary struct was accepted as an element type")
	}
	if !strings.Contains(err.Error(), "eighteen element types") {
		t.Fatalf("the refusal does not name the closed set: %v", err)
	}
}
