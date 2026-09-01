package graphics

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 65 — IndexBuffer's six generic transfer members.
// ---------------------------------------------------------------------------
//
// All six are CLR generic INSTANCE methods, so the settled generic-method rule
// projects each as a package-level generic function taking the receiver first.
// Go methods cannot declare type parameters; no arrangement of receivers makes
// them.
//
// # The overload family, and what each one funnels into
//
//	SetData<T>(T[])                          -> SetData(0, data, 0, data.Length)
//	SetData<T>(T[], int, int)                -> SetData(0, data, start, count)
//	SetData<T>(int, T[], int, int)           -> CopyData(offset, ..., None, true)
//	GetData<T>(T[])                          -> GetData(0, data, 0, data.Length)
//	GetData<T>(T[], int, int)                -> GetData(0, data, start, count)
//	GetData<T>(int, T[], int, int)           -> CopyData(offset, ..., 16, false)
//
// The two short overloads are literally `call SetData(0, ...)` in the IL, so
// the projection funnels the same way: one body carries the semantics and five
// members reach it. A `null` array becomes length ZERO in the two shortest
// overloads before CopyData sees it, which then refuses it -- so the refusal is
// the same either way.
//
// # What CopyData checks, in its order
//
//	Helpers.CheckDisposed(this, pComPtr)     ObjectDisposedException
//	data == null || data.Length == 0         ArgumentNullException("data", NullNotAllowed)
//	the buffer is the device's current
//	  Indices, and options is not
//	  Discard|NoOverwrite                    the D3D "still bound" failure
//	getting from a WriteOnly buffer          NotSupportedException(WriteOnlyGetNotSupported)
//	Helpers.ValidateCopyParameters           ArgumentOutOfRangeException(MustBeValidIndex)
//	sizeof(T)*elementCount + offsetInBytes
//	  <= _bufferSize                         InvalidOperationException(ResourceDataMustBeCorrectSize)
//
// Every one of those is reproduced except the third, and the third is recorded
// rather than skipped: CNA-Go does not project GraphicsDevice::Indices yet, so
// there is no way to ask whether this buffer is the bound one. When that member
// arrives the check has a place to live; until then a transfer to a bound
// buffer reaches CNA, which answers for itself.

// errIndexTransferNotSupported projects System.NotSupportedException.
var errIndexTransferNotSupported = errors.New("operation is not supported")

// errIndexTransferInvalidOperation projects System.InvalidOperationException.
var errIndexTransferInvalidOperation = errors.New("operation is not valid")

// errUnsupportedIndexElement is the refusal for a T outside the closed set.
//
// The reference's constraint is `valuetype .ctor T`: any struct at all, whose
// unmanaged size decides how many bytes move. CNA's index transfer names an
// element WIDTH instead, so the projection's set is closed at the four Go types
// an index is, and a T outside it is refused by name rather than transferred at
// a width CNA was never told about.
var errUnsupportedIndexElement = errors.New("an index transfer element must be a 16-bit or 32-bit integer")

// indexElementWidth reports the byte width one element of T occupies, and is
// the Go side of the same check the texture transfer makes:
//
//	CNA identifies an element by its WIDTH, never by the Go type's size.
//
// So a T whose layout is not the width its identity means would be copied into
// a buffer CNA reads with a different stride, and nothing on either side would
// report it. The check is `unsafe.Sizeof` against the width, by name.
func indexElementWidth[T any]() (uint32, IndexElementSize, error) {
	var zero T
	switch any(zero).(type) {
	case int16, uint16:
		if unsafe.Sizeof(zero) != 2 {
			return 0, 0, fmt.Errorf("%w: %T is %d bytes, not 2", errUnsupportedIndexElement, zero, unsafe.Sizeof(zero))
		}
		return 2, IndexElementSizeSixteenBits, nil
	case int32, uint32:
		if unsafe.Sizeof(zero) != 4 {
			return 0, 0, fmt.Errorf("%w: %T is %d bytes, not 4", errUnsupportedIndexElement, zero, unsafe.Sizeof(zero))
		}
		return 4, IndexElementSizeThirtyTwoBits, nil
	default:
		return 0, 0, fmt.Errorf("%w: %T", errUnsupportedIndexElement, zero)
	}
}

// validateCopyParameters is Microsoft.Xna.Framework.Helpers::ValidateCopyParameters,
// 67 bytes shared by every buffer transfer in the reference:
//
//	if (dataIndex < 0 || dataIndex > dataLength)      throw ("dataIndex", MustBeValidIndex)
//	if (elementCount + dataIndex > dataLength)        throw ("elementCount", MustBeValidIndex)
//	if (elementCount <= 0)                            throw ("elementCount", MustBeValidIndex)
//
// The order matters and the PARAMETER NAMES differ between the three, which is
// what a caller reads first. Note the middle one: `dataIndex == dataLength` is
// accepted by the FIRST check and rejected by the second whenever the count is
// positive -- and the count is required positive by the third, so an empty
// window at the end of the array is refused, not accepted.
func validateCopyParameters(dataLength, dataIndex, elementCount int32) error {
	if dataIndex < 0 || dataIndex > dataLength {
		return fmt.Errorf("%w: dataIndex: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if elementCount+dataIndex > dataLength {
		return fmt.Errorf("%w: elementCount: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if elementCount <= 0 {
		return fmt.Errorf("%w: elementCount: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	return nil
}

// IndexBufferSetDataBySliceOfT is IndexBuffer::SetData<T>(T[]):
//
//	SetData(0, data, 0, data == null ? 0 : data.Length);
func IndexBufferSetDataBySliceOfT[T any](buffer *IndexBuffer, data []T) error {
	return IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32(buffer, 0, data, 0, int32(len(data)))
}

// IndexBufferSetDataBySliceOfTAndInt32AndInt32 is
// IndexBuffer::SetData<T>(T[], int32, int32):
//
//	SetData(0, data, startIndex, elementCount);
func IndexBufferSetDataBySliceOfTAndInt32AndInt32[T any](buffer *IndexBuffer, data []T, startIndex, elementCount int32) error {
	return IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32(buffer, 0, data, startIndex, elementCount)
}

// IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32 is
// IndexBuffer::SetData<T>(int32 offsetInBytes, T[], int32, int32), the member
// the other two funnel into:
//
//	CopyData(offsetInBytes, data, startIndex, elementCount, 0, true);
//
// `0` is SetDataOptions.None, which the reference hardcodes: the streaming
// options belong to DynamicIndexBuffer's own SetData overloads, which are a
// different type CNA-Go does not project.
func IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32[T any](buffer *IndexBuffer, offsetInBytes int32, data []T, startIndex, elementCount int32) error {
	width, size, err := prepareIndexTransfer[T](buffer, data, startIndex, elementCount, offsetInBytes, true)
	if err != nil {
		return err
	}
	resource := buffer.nativeResource()
	if resource == nil {
		return errIndexBufferNil
	}
	// The window is the CALLER'S: CNA reads `element_count` elements starting
	// at `start_index` in the array it is handed, exactly as the reference
	// pins `data[startIndex]` and copies from there.
	if offsetInBytes == 0 {
		return resource.SetIndexData(nativeIndexElementSize(size), 0,
			uint64(startIndex), uint64(elementCount), unsafe.Pointer(&data[0]), uint64(len(data)))
	}
	_ = width
	return resource.SetIndexDataAt(uint64(offsetInBytes), nativeIndexElementSize(size), 0,
		uint64(startIndex), uint64(elementCount), unsafe.Pointer(&data[0]), uint64(len(data)))
}

// IndexBufferGetDataBySliceOfT is IndexBuffer::GetData<T>(T[]).
func IndexBufferGetDataBySliceOfT[T any](buffer *IndexBuffer, data []T) error {
	return IndexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32(buffer, 0, data, 0, int32(len(data)))
}

// IndexBufferGetDataBySliceOfTAndInt32AndInt32 is
// IndexBuffer::GetData<T>(T[], int32, int32).
func IndexBufferGetDataBySliceOfTAndInt32AndInt32[T any](buffer *IndexBuffer, data []T, startIndex, elementCount int32) error {
	return IndexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32(buffer, 0, data, startIndex, elementCount)
}

// IndexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32 is
// IndexBuffer::GetData<T>(int32 offsetInBytes, T[], int32, int32):
//
//	CopyData(offsetInBytes, data, startIndex, elementCount, 16, false);
//
// `16` is D3DLOCK_READONLY, an internal lock flag rather than a
// SetDataOptions value, which is why the read path passes a constant the
// public enum has no name for. CNA takes options None on its read route and
// rejects anything else, so nothing crosses for it.
//
// CNA's readback begins at buffer index ZERO and writes into the caller's array
// at `start_index`, "matching the documented XNA overload contract" -- so
// `offsetInBytes` has no counterpart on the read route and a non-zero one is
// refused rather than silently ignored.
func IndexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32[T any](buffer *IndexBuffer, offsetInBytes int32, data []T, startIndex, elementCount int32) error {
	_, size, err := prepareIndexTransfer[T](buffer, data, startIndex, elementCount, offsetInBytes, false)
	if err != nil {
		return err
	}
	if buffer.BufferUsage() == BufferUsageWriteOnly {
		return fmt.Errorf("%w: %s", errIndexTransferNotSupported, writeOnlyGetNotSupported)
	}
	if offsetInBytes != 0 {
		return fmt.Errorf("%w: offsetInBytes: CNA reads back from index zero", errArgumentOutOfRange)
	}
	resource := buffer.nativeResource()
	if resource == nil {
		return errIndexBufferNil
	}
	_, err = resource.GetIndexData(nativeIndexElementSize(size),
		uint64(startIndex), uint64(elementCount), unsafe.Pointer(&data[0]), uint64(len(data)))
	return err
}

// prepareIndexTransfer is CopyData's guard prefix, shared by both directions
// because the reference shares it too.
func prepareIndexTransfer[T any](buffer *IndexBuffer, data []T, startIndex, elementCount, offsetInBytes int32, setting bool) (uint32, IndexElementSize, error) {
	if buffer == nil || buffer.resource == nil {
		return 0, 0, errIndexBufferNil
	}
	if buffer.IsDisposed() {
		// Helpers.CheckDisposed(this, pComPtr), which throws
		// ObjectDisposedException naming the type.
		return 0, 0, fmt.Errorf("%w: IndexBuffer", interop.ErrDisposed)
	}
	// `data == null || data.Length == 0` is ONE branch in the IL, and both
	// arrive at ArgumentNullException("data"). Go has no null slice distinct
	// from an empty one and does not need one.
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("%w: data: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	width, size, err := indexElementWidth[T]()
	if err != nil {
		return 0, 0, err
	}
	if err := validateCopyParameters(int32(len(data)), startIndex, elementCount); err != nil {
		return 0, 0, err
	}
	// sizeof(T) * elementCount + offsetInBytes must fit the buffer, which the
	// reference measures against `_bufferSize` -- the width it CREATED the
	// buffer with, not the width of T. So a 32-bit transfer into a 16-bit
	// buffer is refused here by size, before CNA is asked.
	if offsetInBytes < 0 {
		return 0, 0, fmt.Errorf("%w: offsetInBytes: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	bufferSize := int64(buffer.IndexCount()) * int64(indexElementWidthFor(buffer.IndexElementSize()))
	if int64(width)*int64(elementCount)+int64(offsetInBytes) > bufferSize {
		return 0, 0, fmt.Errorf("%w: %s", errIndexTransferInvalidOperation, resourceDataMustBeCorrectSize)
	}
	_ = setting
	return width, size, nil
}

// indexElementWidthFor is the reference's `_indexSize`: the width the buffer
// was created with, which is what its size is measured in.
func indexElementWidthFor(size IndexElementSize) uint32 {
	if size == IndexElementSizeThirtyTwoBits {
		return 4
	}
	return 2
}
