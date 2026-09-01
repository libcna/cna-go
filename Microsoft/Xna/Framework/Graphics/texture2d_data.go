package graphics

import (
	"fmt"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
	"github.com/openeggbert/cna-go/internal/interop"
)

// Texture2D's typed transfers, and the generic-method projection rule they are
// the profile's first users of.
//
// # Why these are functions and not methods
//
// Go METHODS CANNOT DECLARE TYPE PARAMETERS. `func (t *Texture2D) SetData[T any]`
// does not compile, and no arrangement of receivers makes it: the restriction is
// in the language, not in this binding. A CLR generic instance method therefore
// has no method-shaped projection at all.
//
// The settled response to "this member cannot be a method here" already exists
// in the profile. The cross-package cycle rule turns such a member into a
// package-level function whose FIRST parameter is the receiver, and that is what
// a generic method becomes:
//
//	Texture2D::SetData<T>(T[])  ->  Texture2DSetDataBySliceOfT[T any](*Texture2D, []T) error
//
// The suffix names the type parameter the member DECLARES. `!!0` in the contract
// is an IL token meaning "this method's first type parameter"; before
// Foundation 54 nothing resolved it and the verifier expected `SliceOf0`, a name
// for a position rather than a type.
//
// # The constraint, and what Go cannot say
//
// The CLR constrains T as `valuetype .ctor T` -- a struct with a default
// constructor. Go has no such constraint, so the projection is `[T any]` and the
// element type is checked at RUNTIME, against the closed set CNA declares. That
// is not a loosening of the contract: the CLR also fails at runtime for a struct
// whose layout does not match the surface format, and CNA validates
// type/format compatibility itself. What Go loses is the compile-time refusal of
// a reference type, and that is recorded here rather than papered over.

// textureElementType resolves a Go element type to the CNA_TEXTURE_DATA_*
// identity that describes it, and to the byte width that identity implies.
//
// The mapping is closed and total over CNA's eighteen identities: every one of
// them has a Go type, and every Go type here is one XNA's SurfaceFormat family
// declares. A type that is not in it is refused by name rather than transferred
// as raw bytes of the wrong shape.
//
// The WIDTH is returned so the caller can check it against the Go type's actual
// size. CNA identifies an element by what it means, not by how large it is, so
// nothing on the native side would notice a Go struct that had grown a field.
func textureElementType[T any]() (uint32, uintptr, string, bool) {
	var zero T
	switch any(zero).(type) {
	case framework.Color:
		return interop.TextureDataColor, 4, "Color", true
	case packedvector.Bgr565:
		return interop.TextureDataBgr565, 2, "Bgr565", true
	case packedvector.Bgra5551:
		return interop.TextureDataBgra5551, 2, "Bgra5551", true
	case packedvector.Bgra4444:
		return interop.TextureDataBgra4444, 2, "Bgra4444", true
	case byte:
		return interop.TextureDataByte, 1, "Byte", true
	case packedvector.NormalizedByte2:
		return interop.TextureDataNormalizedByte2, 2, "NormalizedByte2", true
	case packedvector.NormalizedByte4:
		return interop.TextureDataNormalizedByte4, 4, "NormalizedByte4", true
	case packedvector.Rgba1010102:
		return interop.TextureDataRgba1010102, 4, "Rgba1010102", true
	case packedvector.Rg32:
		return interop.TextureDataRg32, 4, "Rg32", true
	case packedvector.Rgba64:
		return interop.TextureDataRgba64, 8, "Rgba64", true
	case packedvector.Alpha8:
		return interop.TextureDataAlpha8, 1, "Alpha8", true
	case float32:
		return interop.TextureDataSingle, 4, "Single", true
	case framework.Vector2:
		return interop.TextureDataVector2, 8, "Vector2", true
	case framework.Vector4:
		return interop.TextureDataVector4, 16, "Vector4", true
	case packedvector.HalfSingle:
		return interop.TextureDataHalfSingle, 2, "HalfSingle", true
	case packedvector.HalfVector2:
		return interop.TextureDataHalfVector2, 4, "HalfVector2", true
	case packedvector.HalfVector4:
		return interop.TextureDataHalfVector4, 8, "HalfVector4", true
	case uint16:
		return interop.TextureDataUShort, 2, "UShort", true
	default:
		return 0, 0, "", false
	}
}

// resolveTextureElement is the shared prologue of all six members: it resolves
// the element identity and REQUIRES the Go type's own size to be the width that
// identity means.
//
// The size check is the load-bearing half. CNA identifies an element by what it
// represents, so a Go type whose layout drifted -- a packed vector that gained a
// field, a Color that stopped being four bytes -- would be copied wholesale into
// a buffer CNA reads with a different stride, and nothing on either side would
// report it. This is the same class of defect the ABI layout probe exists for,
// on the Go side of the boundary where that probe cannot see.
func resolveTextureElement[T any]() (uint32, uintptr, error) {
	identity, width, name, known := textureElementType[T]()
	if !known {
		var zero T
		return 0, 0, fmt.Errorf("%w: %T is not one of the eighteen element types a texture transfer accepts",
			errGraphicsResourceArgument, zero)
	}
	if size := unsafe.Sizeof(*new(T)); size != width {
		return 0, 0, fmt.Errorf("%w: the Go type for %s is %d bytes and CNA reads %d",
			errGraphicsResourceArgument, name, size, width)
	}
	return identity, width, nil
}

// Texture2DSetDataBySliceOfT is Texture2D::SetData<T>(T[]).
//
//	SetData(0, null, data, 0, data == null ? 0 : data.Length)
//
// The null branch is the reference's own and is preserved: a null array is
// forwarded with an element count of zero rather than refused here, because the
// refusal the reference makes happens further in, inside CopyData.
func Texture2DSetDataBySliceOfT[T any](texture Texture2DReference, data []T) error {
	return Texture2DSetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, 0, nil, data, 0, int32(len(data)))
}

// Texture2DSetDataBySliceOfTAndInt32AndInt32 is
// Texture2D::SetData<T>(T[], Int32, Int32).
//
//	SetData(0, null, data, startIndex, elementCount)
func Texture2DSetDataBySliceOfTAndInt32AndInt32[T any](
	texture Texture2DReference, data []T, startIndex, elementCount int32,
) error {
	return Texture2DSetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, 0, nil, data, startIndex, elementCount)
}

// Texture2DSetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32 is
// Texture2D::SetData<T>(Int32, Nullable<Rectangle>, T[], Int32, Int32) -- the
// overload the other two funnel into, and the one that reaches CNA.
func Texture2DSetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[T any](
	texture Texture2DReference, level int32, rect *framework.Rectangle, data []T, startIndex, elementCount int32,
) error {
	resource, transfer, identity, err := prepareTransfer[T](texture, level, rect, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	var pointer unsafe.Pointer
	if len(data) > 0 {
		pointer = unsafe.Pointer(&data[0])
	}
	return resource.SetTextureData(identity, transfer, pointer, uint64(len(data)))
}

// Texture2DGetDataBySliceOfT is Texture2D::GetData<T>(T[]), the mirror of the
// first SetData overload with the same null-array shape.
func Texture2DGetDataBySliceOfT[T any](texture Texture2DReference, data []T) error {
	return Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, 0, nil, data, 0, int32(len(data)))
}

// Texture2DGetDataBySliceOfTAndInt32AndInt32 is
// Texture2D::GetData<T>(T[], Int32, Int32).
func Texture2DGetDataBySliceOfTAndInt32AndInt32[T any](
	texture Texture2DReference, data []T, startIndex, elementCount int32,
) error {
	return Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, 0, nil, data, startIndex, elementCount)
}

// Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32 is
// Texture2D::GetData<T>(Int32, Nullable<Rectangle>, T[], Int32, Int32).
//
// CNA reports how many elements the transfer REQUIRES alongside the copy, and a
// destination too small is a refused call rather than a partial fill -- CNA's
// own documented behaviour, and the reason this member reports rather than
// silently truncating.
func Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[T any](
	texture Texture2DReference, level int32, rect *framework.Rectangle, data []T, startIndex, elementCount int32,
) error {
	resource, transfer, identity, err := prepareTransfer[T](texture, level, rect, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	var pointer unsafe.Pointer
	if len(data) > 0 {
		pointer = unsafe.Pointer(&data[0])
	}
	required, err := resource.GetTextureData(identity, transfer, pointer, uint64(len(data)))
	if err != nil {
		return err
	}
	if required > uint64(len(data)) {
		return fmt.Errorf("%w: the transfer needs %d elements and the destination holds %d",
			errGraphicsResourceArgument, required, len(data))
	}
	return nil
}

// prepareTransfer resolves the element identity and builds the transfer both
// directions share.
func prepareTransfer[T any](
	reference Texture2DReference, level int32, rect *framework.Rectangle, length int, startIndex, elementCount int32,
) (*interop.Resource, interop.TextureTransfer, uint32, error) {
	texture := resolveTexture2D(reference)
	if texture == nil || texture.nativeResource() == nil {
		return nil, interop.TextureTransfer{}, 0, interop.ErrDisposed
	}
	identity, _, err := resolveTextureElement[T]()
	if err != nil {
		return nil, interop.TextureTransfer{}, 0, err
	}
	if startIndex < 0 || elementCount < 0 {
		return nil, interop.TextureTransfer{}, 0, fmt.Errorf(
			"%w: a transfer window is negative: start %d count %d",
			errGraphicsResourceArgument, startIndex, elementCount)
	}
	if int(startIndex)+int(elementCount) > length {
		return nil, interop.TextureTransfer{}, 0, fmt.Errorf(
			"%w: the transfer window [%d,%d) leaves an array of %d",
			errGraphicsResourceArgument, startIndex, int(startIndex)+int(elementCount), length)
	}
	transfer := interop.TextureTransfer{
		Level:        level,
		StartIndex:   uint64(startIndex),
		ElementCount: uint64(elementCount),
	}
	if rect != nil {
		transfer.HasRectangle = true
		transfer.X, transfer.Y = rect.X, rect.Y
		transfer.Width, transfer.Height = rect.Width, rect.Height
	}
	return texture.nativeResource(), transfer, identity, nil
}
