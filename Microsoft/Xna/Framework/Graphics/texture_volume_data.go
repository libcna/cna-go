package graphics

import (
	"fmt"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 71 — the cube and volume typed transfers.
// ---------------------------------------------------------------------------
//
// Twelve generic members -- six on each type -- on the settled generic-method
// rule: a CLR generic instance method has no method-shaped projection in Go, so
// it becomes a package-level function whose first parameter is the receiver.
//
// # The element set is ONE type wide, and that is CNA's limit rather than XNA's
//
// `cna_texture2d_set_data` takes a `CNA_TextureDataType` identity and eighteen
// element representations behind it. `cna_texturecube_set_data` and
// `cna_texture3d_set_data` take `const CNA_Color*` and nothing else: their
// prototypes have no data-type parameter at all, so there is no way to tell CNA
// that an array holds Bgr565 or HalfVector4.
//
// The reference accepts any `valuetype .ctor T` whose size divides the surface
// format's, exactly as it does for Texture2D. So a T outside the accepted set
// is refused BY NAME here, with a reason that says whose limit it is: this is a
// CNA ABI EXPRESSIVENESS limitation, not an XNA restriction, and it must not be
// reworded into one. It is the same class of narrowing Foundation 66 recorded
// for a vertex buffer's partial-vertex transfers.

// errVolumeElementUnsupported is the refusal a T outside the one-element set
// gets. The reference has no counterpart.
func volumeElementRefusal[T any](member string) error {
	var zero T
	return fmt.Errorf(
		"%w: %s accepts only framework.Color elements, and got %T: CNA's cube and volume transfer routes take CNA_Color and carry no data-type identity, unlike cna_texture2d_set_data",
		errGraphicsResourceArgument, member, zero)
}

// resolveVolumeElement is the shared prologue. It requires T to be
// framework.Color AND requires the Go type's own size to be the four bytes
// CNA_Color is, for the reason resolveTextureElement does: a Color that stopped
// being four bytes would be copied wholesale into a buffer CNA reads with a
// different stride, and nothing on either side would report it.
func resolveVolumeElement[T any](member string) error {
	var zero T
	if _, isColor := any(zero).(framework.Color); !isColor {
		return volumeElementRefusal[T](member)
	}
	if size := unsafe.Sizeof(*new(T)); size != 4 {
		return fmt.Errorf("%w: the Go type for Color is %d bytes and CNA reads 4",
			errGraphicsResourceArgument, size)
	}
	return nil
}

// checkTransferWindow is the array-window check both families share with the
// 2D one. It is the same shape rather than a shared function because each
// member reports its own parameter names, which the settled rule requires.
func checkTransferWindow(length int, startIndex, elementCount int32) error {
	if startIndex < 0 || elementCount < 0 {
		return fmt.Errorf("%w: a transfer window is negative: start %d count %d",
			errGraphicsResourceArgument, startIndex, elementCount)
	}
	if int(startIndex)+int(elementCount) > length {
		return fmt.Errorf("%w: the transfer window [%d,%d) leaves an array of %d",
			errGraphicsResourceArgument, startIndex, int(startIndex)+int(elementCount), length)
	}
	return nil
}

func sliceStart[T any](data []T) unsafe.Pointer {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Pointer(&data[0])
}

// ---------------------------------------------------------------------------
// TextureCube
// ---------------------------------------------------------------------------

// TextureCubeSetDataByCubeMapFaceAndSliceOfT is
// TextureCube::SetData<T>(CubeMapFace, T[]):
//
//	SetData(cubeMapFace, 0, null, data, 0, data == null ? 0 : data.Length)
//
// The null branch is the reference's own -- a null array is forwarded with a
// count of zero rather than refused here, because the refusal happens further
// in, inside CopyData.
func TextureCubeSetDataByCubeMapFaceAndSliceOfT[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, data []T,
) error {
	return TextureCubeSetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, cubeMapFace, 0, nil, data, 0, int32(len(data)))
}

// TextureCubeSetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32 is
// TextureCube::SetData<T>(CubeMapFace, T[], Int32, Int32).
func TextureCubeSetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, data []T, startIndex, elementCount int32,
) error {
	return TextureCubeSetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, cubeMapFace, 0, nil, data, startIndex, elementCount)
}

// TextureCubeSetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32
// is TextureCube::SetData<T>(CubeMapFace, Int32, Nullable<Rectangle>, T[],
// Int32, Int32) -- the overload the other two funnel into, and the one that
// reaches CNA.
func TextureCubeSetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, level int32, rect *framework.Rectangle,
	data []T, startIndex, elementCount int32,
) error {
	resource, transfer, err := prepareCubeTransfer[T](resolveTextureCube(texture), "TextureCube.SetData", cubeMapFace, level, rect, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	if err := resource.SetTextureCubeData(transfer, sliceStart(data), uint64(len(data))); err != nil {
		return err
	}
	// CopyData's tail on the SETTING path, after the result check: a
	// RenderTargetCube's content-lost latch clears on a successful upload.
	resolveTextureCube(texture).noteContentRestored()
	return nil
}

// TextureCubeGetDataByCubeMapFaceAndSliceOfT is
// TextureCube::GetData<T>(CubeMapFace, T[]).
func TextureCubeGetDataByCubeMapFaceAndSliceOfT[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, data []T,
) error {
	return TextureCubeGetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, cubeMapFace, 0, nil, data, 0, int32(len(data)))
}

// TextureCubeGetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32 is
// TextureCube::GetData<T>(CubeMapFace, T[], Int32, Int32).
func TextureCubeGetDataByCubeMapFaceAndSliceOfTAndInt32AndInt32[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, data []T, startIndex, elementCount int32,
) error {
	return TextureCubeGetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		texture, cubeMapFace, 0, nil, data, startIndex, elementCount)
}

// TextureCubeGetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32
// is TextureCube::GetData<T>(CubeMapFace, Int32, Nullable<Rectangle>, T[],
// Int32, Int32).
//
// CNA reports how many elements the region REQUIRES alongside the copy, and a
// destination too small is a refused call rather than a partial fill, which is
// what the 2D readback already does.
func TextureCubeGetDataByCubeMapFaceAndInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[T any](
	texture TextureCubeReference, cubeMapFace CubeMapFace, level int32, rect *framework.Rectangle,
	data []T, startIndex, elementCount int32,
) error {
	resource, transfer, err := prepareCubeTransfer[T](resolveTextureCube(texture), "TextureCube.GetData", cubeMapFace, level, rect, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	required, err := resource.GetTextureCubeData(transfer, sliceStart(data), uint64(len(data)))
	if err != nil {
		return err
	}
	if required > uint64(len(data)) {
		return fmt.Errorf("%w: the transfer needs %d elements and the destination holds %d",
			errGraphicsResourceArgument, required, len(data))
	}
	return nil
}

func prepareCubeTransfer[T any](
	texture *TextureCube, member string, cubeMapFace CubeMapFace, level int32,
	rect *framework.Rectangle, length int, startIndex, elementCount int32,
) (*interop.Resource, interop.TextureCubeTransfer, error) {
	if texture == nil || texture.nativeResource() == nil {
		return nil, interop.TextureCubeTransfer{}, interop.ErrDisposed
	}
	if err := resolveVolumeElement[T](member); err != nil {
		return nil, interop.TextureCubeTransfer{}, err
	}
	if err := checkTransferWindow(length, startIndex, elementCount); err != nil {
		return nil, interop.TextureCubeTransfer{}, err
	}
	transfer := interop.TextureCubeTransfer{
		Face:         uint32(cubeMapFace),
		Level:        level,
		StartIndex:   uint64(startIndex),
		ElementCount: uint64(elementCount),
	}
	if rect != nil {
		transfer.HasRectangle = true
		transfer.X, transfer.Y = rect.X, rect.Y
		transfer.Width, transfer.Height = rect.Width, rect.Height
	}
	return texture.nativeResource(), transfer, nil
}

// ---------------------------------------------------------------------------
// Texture3D
// ---------------------------------------------------------------------------

// Texture3DSetDataBySliceOfT is Texture3D::SetData<T>(T[]):
//
//	SetData(0, 0, 0, _width, _height, 0, _depth, data, 0,
//	        data == null ? 0 : data.Length)
//
// The whole volume, read off the IL: the box is (left 0, top 0, right Width,
// bottom Height, front 0, back Depth) and the argument order in the reference's
// call is level, left, top, right, bottom, FRONT, BACK -- with `_height` before
// the front literal, which is the one place a reader would get the order wrong.
func Texture3DSetDataBySliceOfT[T any](texture *Texture3D, data []T) error {
	return Texture3DSetDataBySliceOfTAndInt32AndInt32(texture, data, 0, int32(len(data)))
}

// Texture3DSetDataBySliceOfTAndInt32AndInt32 is
// Texture3D::SetData<T>(T[], Int32, Int32), which passes the same whole-volume
// box with the caller's array window.
func Texture3DSetDataBySliceOfTAndInt32AndInt32[T any](
	texture *Texture3D, data []T, startIndex, elementCount int32,
) error {
	return Texture3DSetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32(
		texture, 0, 0, 0, texture.Width(), texture.Height(), 0, texture.Depth(),
		data, startIndex, elementCount)
}

// Texture3DSetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32
// is Texture3D::SetData<T>(Int32, Int32, Int32, Int32, Int32, Int32, Int32,
// T[], Int32, Int32) -- (level, left, top, right, bottom, front, back).
func Texture3DSetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32[T any](
	texture *Texture3D, level, left, top, right, bottom, front, back int32,
	data []T, startIndex, elementCount int32,
) error {
	resource, transfer, err := prepareVolumeTransfer[T](texture, "Texture3D.SetData",
		level, left, top, right, bottom, front, back, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	return resource.SetTexture3DData(transfer, sliceStart(data), uint64(len(data)))
}

// Texture3DGetDataBySliceOfT is Texture3D::GetData<T>(T[]).
func Texture3DGetDataBySliceOfT[T any](texture *Texture3D, data []T) error {
	return Texture3DGetDataBySliceOfTAndInt32AndInt32(texture, data, 0, int32(len(data)))
}

// Texture3DGetDataBySliceOfTAndInt32AndInt32 is
// Texture3D::GetData<T>(T[], Int32, Int32).
func Texture3DGetDataBySliceOfTAndInt32AndInt32[T any](
	texture *Texture3D, data []T, startIndex, elementCount int32,
) error {
	return Texture3DGetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32(
		texture, 0, 0, 0, texture.Width(), texture.Height(), 0, texture.Depth(),
		data, startIndex, elementCount)
}

// Texture3DGetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32
// is Texture3D::GetData<T>(Int32 ×7, T[], Int32, Int32).
func Texture3DGetDataByInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndInt32AndSliceOfTAndInt32AndInt32[T any](
	texture *Texture3D, level, left, top, right, bottom, front, back int32,
	data []T, startIndex, elementCount int32,
) error {
	resource, transfer, err := prepareVolumeTransfer[T](texture, "Texture3D.GetData",
		level, left, top, right, bottom, front, back, len(data), startIndex, elementCount)
	if err != nil {
		return err
	}
	required, err := resource.GetTexture3DData(transfer, sliceStart(data), uint64(len(data)))
	if err != nil {
		return err
	}
	if required > uint64(len(data)) {
		return fmt.Errorf("%w: the transfer needs %d elements and the destination holds %d",
			errGraphicsResourceArgument, required, len(data))
	}
	return nil
}

func prepareVolumeTransfer[T any](
	texture *Texture3D, member string, level, left, top, right, bottom, front, back int32,
	length int, startIndex, elementCount int32,
) (*interop.Resource, interop.Texture3DTransfer, error) {
	if texture == nil || texture.nativeResource() == nil {
		return nil, interop.Texture3DTransfer{}, interop.ErrDisposed
	}
	if err := resolveVolumeElement[T](member); err != nil {
		return nil, interop.Texture3DTransfer{}, err
	}
	if err := checkTransferWindow(length, startIndex, elementCount); err != nil {
		return nil, interop.Texture3DTransfer{}, err
	}
	return texture.nativeResource(), interop.Texture3DTransfer{
		Level: level, Left: left, Top: top, Right: right, Bottom: bottom, Front: front, Back: back,
		StartIndex: uint64(startIndex), ElementCount: uint64(elementCount),
	}, nil
}
