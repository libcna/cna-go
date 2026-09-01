package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 71 — the managed half of TextureCube and Texture3D.
// ---------------------------------------------------------------------------
//
// The native half -- a real cube and a real volume created, written and read
// back -- is what the `texture-volume` stress scenario does. Everything here
// reaches no runtime.
//
// `liveTexture` is why the transfer tests can run at all: prepareTransfer asks
// for the native resource FIRST, which is the order Texture2D's already takes,
// so a texture with no resource at all reports disposal before any argument is
// looked at. A zero interop.Resource is non-nil with a zero handle, so it gets
// past that check and fails at the native call -- which is exactly the state a
// disposed texture is in, and the state these tests need.

// liveTexture builds a composed Texture over a resource that exists and holds
// no handle.
func liveTexture() *Texture {
	return newTexture(nil, &interop.Resource{}, SurfaceFormatColor, 1)
}

// TestBothConstructorsRefuseANilDeviceWithMicrosoftsSentence pins the one guard
// each constructor reproduces. Every check after it is CNA's, on the decision
// Texture2D's constructors already took.
func TestBothConstructorsRefuseANilDeviceWithMicrosoftsSentence(t *testing.T) {
	if _, err := NewTextureCube(nil, 4, false, SurfaceFormatColor); err == nil {
		t.Fatal("a nil device built a cube")
	} else if !errors.Is(err, errGraphicsResourceArgumentNull) ||
		!strings.Contains(err.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatalf("%v, want ArgumentNullException(graphicsDevice, DeviceCannotBeNullOnResourceCreate)", err)
	}
	if _, err := NewTexture3D(nil, 4, 4, 4, false, SurfaceFormatColor); err == nil {
		t.Fatal("a nil device built a volume")
	} else if !errors.Is(err, errGraphicsResourceArgumentNull) ||
		!strings.Contains(err.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatalf("%v, want ArgumentNullException(graphicsDevice, DeviceCannotBeNullOnResourceCreate)", err)
	}
}

// TestANegativeDimensionIsRefusedBeforeTheUnsignedConversion pins the Go-only
// guard and says why it exists: CNA takes every dimension as uint32, so a
// negative would arrive as an enormous positive and CNA would try to allocate
// it. This refusal is not the reference's -- the reference's own check is
// further in -- and it exists so the conversion cannot invent a dimension.
func TestANegativeDimensionIsRefusedBeforeTheUnsignedConversion(t *testing.T) {
	device := &GraphicsDevice{}
	if _, err := NewTextureCube(device, -1, false, SurfaceFormatColor); err == nil {
		t.Fatal("a negative cube size was accepted")
	}
	for _, dimensions := range [][3]int32{{-1, 4, 4}, {4, -1, 4}, {4, 4, -1}} {
		if _, err := NewTexture3D(device, dimensions[0], dimensions[1], dimensions[2], false, SurfaceFormatColor); err == nil {
			t.Fatalf("a negative volume dimension %v was accepted", dimensions)
		}
	}
}

// TestTheFourGeometryReadersAnswerWithoutAnyNativeHalf pins that all four are
// managed field reads: they answer on an object with no CNA handle at all, and
// they answer on a nil receiver, because the reference's are `ldfld` with no
// disposal check.
func TestTheFourGeometryReadersAnswerWithoutAnyNativeHalf(t *testing.T) {
	cube := &TextureCube{size: 64}
	if cube.Size() != 64 {
		t.Fatalf("Size() = %d", cube.Size())
	}
	volume := &Texture3D{width: 8, height: 16, depth: 32}
	if volume.Width() != 8 || volume.Height() != 16 || volume.Depth() != 32 {
		t.Fatalf("(%d, %d, %d)", volume.Width(), volume.Height(), volume.Depth())
	}
	var nilCube *TextureCube
	var nilVolume *Texture3D
	if nilCube.Size() != 0 || nilVolume.Width() != 0 || nilVolume.Height() != 0 || nilVolume.Depth() != 0 {
		t.Fatal("a nil receiver answered with a value")
	}
}

// TestTheElementSetIsOneTypeWideAndSaysWhoseLimitItIs pins the narrowing AND
// its attribution. `cna_texture2d_set_data` takes a CNA_TextureDataType and
// eighteen representations; the cube and volume routes take `const CNA_Color*`
// and carry no data-type parameter at all. The reference accepts any
// `valuetype .ctor T`, so this is a CNA ABI expressiveness limit and the
// message must say so rather than reading like an XNA rule.
func TestTheElementSetIsOneTypeWideAndSaysWhoseLimitItIs(t *testing.T) {
	cube := &TextureCube{texture: liveTexture(), size: 4}
	volume := &Texture3D{texture: liveTexture(), width: 4, height: 4, depth: 4}
	// Every refusal must name CNA, not XNA.
	for name, call := range map[string]func() error{
		"cube": func() error {
			return TextureCubeSetDataByCubeMapFaceAndSliceOfT(cube, CubeMapFacePositiveX, []packedvector.Bgr565{{}})
		},
		"volume": func() error { return Texture3DSetDataBySliceOfT(volume, []packedvector.Bgr565{{}}) },
	} {
		err := call()
		if err == nil {
			t.Fatalf("%s: a Bgr565 element was accepted", name)
		}
		if !strings.Contains(err.Error(), "framework.Color") {
			t.Fatalf("%s: %v does not name the accepted element type", name, err)
		}
		if !strings.Contains(err.Error(), "CNA") {
			t.Fatalf("%s: %v does not attribute the limit to CNA", name, err)
		}
		if strings.Contains(err.Error(), "XNA") {
			t.Fatalf("%s: %v attributes a CNA limit to XNA", name, err)
		}
	}
}

// TestAColorTransferPassesTheElementCheckAndReachesTheResource is the other
// half: with the accepted element type, both families get past the element gate
// and report the DISPOSED sentinel, which is only reachable once the resource
// is asked for.
func TestAColorTransferPassesTheElementCheckAndReachesTheResource(t *testing.T) {
	cube := &TextureCube{texture: liveTexture(), size: 4}
	volume := &Texture3D{texture: liveTexture(), width: 4, height: 4, depth: 4}
	pixels := make([]framework.Color, 16)
	for name, call := range map[string]func() error{
		"cube set":   func() error { return TextureCubeSetDataByCubeMapFaceAndSliceOfT(cube, CubeMapFacePositiveX, pixels) },
		"cube get":   func() error { return TextureCubeGetDataByCubeMapFaceAndSliceOfT(cube, CubeMapFacePositiveX, pixels) },
		"volume set": func() error { return Texture3DSetDataBySliceOfT(volume, pixels) },
		"volume get": func() error { return Texture3DGetDataBySliceOfT(volume, pixels) },
	} {
		if err := call(); !errors.Is(err, interop.ErrDisposed) {
			t.Fatalf("%s: error = %v, want the element check passed and the resource reached", name, err)
		}
	}
}

// TestTheTransferWindowIsCheckedAgainstTheCallersArray pins the window guard on
// every one of the twelve members, which is the coverage control: a member that
// forwarded without checking would pass every test above.
func TestTheTransferWindowIsCheckedAgainstTheCallersArray(t *testing.T) {
	cube := &TextureCube{texture: liveTexture(), size: 4}
	volume := &Texture3D{texture: liveTexture(), width: 4, height: 4, depth: 4}
	pixels := make([]framework.Color, 4)
	for name, call := range map[string]func() error{
		"cube set window": func() error {
			return TextureCubeSetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32(cube, CubeMapFacePositiveX, pixels, 2, 4)
		},
		"cube get window": func() error {
			return TextureCubeGetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32(cube, CubeMapFacePositiveX, pixels, 2, 4)
		},
		"cube set rect": func() error {
			return TextureCubeSetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
				cube, CubeMapFacePositiveX, 0, nil, pixels, -1, 1)
		},
		"cube get rect": func() error {
			return TextureCubeGetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
				cube, CubeMapFacePositiveX, 0, nil, pixels, 0, -1)
		},
		"volume set window": func() error {
			return Texture3DSetDataBySliceOfTAndInt32AndInt32(volume, pixels, 2, 4)
		},
		"volume get window": func() error {
			return Texture3DGetDataBySliceOfTAndInt32AndInt32(volume, pixels, 2, 4)
		},
		"volume set box": func() error {
			return Texture3DSetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32(
				volume, 0, 0, 0, 4, 4, 0, 4, pixels, -1, 1)
		},
		"volume get box": func() error {
			return Texture3DGetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32(
				volume, 0, 0, 0, 4, 4, 0, 4, pixels, 0, -1)
		},
	} {
		err := call()
		if err == nil || errors.Is(err, interop.ErrDisposed) {
			t.Fatalf("%s: error = %v, want the window refusal", name, err)
		}
	}
}

// TestTheWholeVolumeBoxComesFromTheTexturesOwnDimensions pins the defaults the
// two short Texture3D overloads pass, which the IL states as
//
//	SetData(0, 0, 0, _width, _height, 0, _depth, data, startIndex, elementCount)
//
// with `_height` sitting BEFORE the front literal -- the one place the argument
// order is easy to get wrong. A projection that swapped front and bottom would
// still compile and would transfer a different box.
func TestTheWholeVolumeBoxComesFromTheTexturesOwnDimensions(t *testing.T) {
	volume := &Texture3D{texture: liveTexture(), width: 8, height: 16, depth: 32}
	// The short overload and the explicit one with the reference's own box must
	// be the same call, and both reach the resource.
	short := Texture3DSetDataBySliceOfT(volume, make([]framework.Color, 1))
	explicit := Texture3DSetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32(
		volume, 0, 0, 0, 8, 16, 0, 32, make([]framework.Color, 1), 0, 1)
	if !errors.Is(short, interop.ErrDisposed) || !errors.Is(explicit, interop.ErrDisposed) {
		t.Fatalf("short = %v, explicit = %v; both must reach the resource", short, explicit)
	}
	transfer := func(t *Texture3D) interop.Texture3DTransfer {
		_, value, err := prepareVolumeTransfer[framework.Color](
			&Texture3D{texture: t.texture, width: t.width, height: t.height, depth: t.depth},
			"probe", 0, 0, 0, t.Width(), t.Height(), 0, t.Depth(), 1, 0, 1)
		if err != nil {
			return interop.Texture3DTransfer{}
		}
		return value
	}
	got := transfer(volume)
	want := interop.Texture3DTransfer{
		Right: 8, Bottom: 16, Back: 32, ElementCount: 1,
	}
	if got != want {
		t.Fatalf("the whole-volume box is %+v, want %+v", got, want)
	}
}

// TestTheCubeFaceIsPartOfTheTransferRatherThanTheHandle pins the shape CNA's
// ABI takes: a cube is ONE handle and every transfer names a face, which is why
// all six cube members carry a CubeMapFace and the type itself does not.
func TestTheCubeFaceIsPartOfTheTransferRatherThanTheHandle(t *testing.T) {
	cube := &TextureCube{texture: liveTexture(), size: 4}
	for _, face := range []CubeMapFace{
		CubeMapFacePositiveX, CubeMapFaceNegativeX, CubeMapFacePositiveY,
		CubeMapFaceNegativeY, CubeMapFacePositiveZ, CubeMapFaceNegativeZ,
	} {
		_, transfer, err := prepareCubeTransfer[framework.Color](cube, "probe", face, 0, nil, 1, 0, 1)
		if err != nil {
			t.Fatalf("face %v: %v", face, err)
		}
		if transfer.Face != uint32(face) {
			t.Fatalf("face %v crossed as %d", face, transfer.Face)
		}
		if transfer.HasRectangle {
			t.Fatalf("face %v: a nil rectangle became a rectangle", face)
		}
	}
	// And a rectangle really reaches the transfer.
	rect := framework.NewRectangle(1, 2, 3, 4)
	_, transfer, err := prepareCubeTransfer[framework.Color](cube, "probe", CubeMapFacePositiveX, 2, &rect, 1, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !transfer.HasRectangle || transfer.X != 1 || transfer.Y != 2 || transfer.Width != 3 || transfer.Height != 4 || transfer.Level != 2 {
		t.Fatalf("the rectangle and level did not reach the transfer: %+v", transfer)
	}
}

// TestBothTypesAreUsableAtATextureParameterPosition pins that the
// substitutable-base rule reaches them: a Texture-typed position accepts a
// TextureCube and a Texture3D, which in the CLR is the IS-A the base chain
// gives for free.
func TestBothTypesAreUsableAtATextureParameterPosition(t *testing.T) {
	base := liveTexture()
	cube := &TextureCube{texture: base}
	volume := &Texture3D{texture: base}
	var _ TextureReference = cube
	var _ TextureReference = volume
	if resolveTexture(cube) != base || resolveTexture(volume) != base {
		t.Fatal("the composed Texture did not answer at a Texture position")
	}
	var nilCube *TextureCube
	if resolveTexture(nilCube) != nil {
		t.Fatal("a typed nil answered with a Texture")
	}
}
