package graphics

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 66 — VertexBuffer's six generic transfer members.
// ---------------------------------------------------------------------------
//
// All six are CLR generic instance methods, so each projects as a package-level
// generic function taking the receiver first. Five funnel into the sixth, which
// is what the IL does:
//
//	SetData<T>(T[])                        -> SetData(0, data, 0, len, 0)
//	SetData<T>(T[], int, int)              -> SetData(0, data, start, count, 0)
//	SetData<T>(int, T[], int, int, int)    -> CopyData(..., options 0, true)
//	GetData<T>(...)                           the same three
//
// # Why the RAW CNA route and not the typed one
//
// CNA publishes two upload families. The TYPED one packs a value of one of its
// built-in `CNA_VertexType` layouts; the RAW one takes bytes, a vertex count
// and an explicit byte stride. XNA's `SetData<T>` is the second shape exactly:
// its T is any struct and its size is `sizeof(T)`, which is the stride. Using
// the typed route would mean closing T to a set CNA named, which the reference
// does not do -- so the raw route is the faithful one, not the fallback.
//
// `cna_vertex_buffer_set_data_raw_at`'s offset indexes THE BUFFER, which is the
// one offset in CNA's transfer family that does, and is what XNA's
// `offsetInBytes` means.
//
// # CopyData's guards, in the reference's order
//
//	Helpers.CheckDisposed                     ObjectDisposedException
//	data == null || data.Length == 0          ArgumentNullException("data")
//	the buffer is bound to the device         NOT REPRODUCED -- see below
//	getting from a WriteOnly buffer           NotSupportedException
//	Helpers.ValidateCopyParameters            ArgumentOutOfRangeException x3
//	vertexStride != 0 && vertexStride < sizeof(T)
//	                                          ArgumentOutOfRangeException(VertexStrideTooSmall)
//	bytes + offsetInBytes <= _size            InvalidOperationException
//
// The bound-buffer check is absent for the same measured reason IndexBuffer's
// is: `GraphicsDevice::GetVertexBuffers` is not projected yet, so there is no
// way to ask which buffers are bound. It is recorded here rather than skipped.

// errVertexTransferNotSupported projects System.NotSupportedException.
var errVertexTransferNotSupported = errors.New("operation is not supported")

// errVertexTransferInvalidOperation projects System.InvalidOperationException.
var errVertexTransferInvalidOperation = errors.New("operation is not valid")

// errVertexStrideUnsupported is the ONE place this projection is narrower than
// the reference, and it is a measured upstream limit rather than a choice.
//
// XNA's strided transfer writes `sizeof(T)` bytes at `offsetInBytes + i*stride`
// and LEAVES THE GAPS ALONE -- which is the whole point of an interleaved
// buffer, where the gaps hold other attributes. CNA publishes no route with a
// separate source and destination stride: `vertex_stride` there describes the
// SOURCE vertex, so composing the padded image on the Go side would write zeros
// into bytes XNA preserves.
//
// Refusing is the honest answer. Writing the gaps would corrupt data a consumer
// put there through another call, and would do it silently.
var errVertexStrideUnsupported = errors.New(
	"a vertex stride larger than the element size needs a CNA route that writes a strided window without touching the gaps, and 0.21.0 publishes none")

// VertexBufferSetDataBySliceOfT is VertexBuffer::SetData<T>(T[]).
func VertexBufferSetDataBySliceOfT[T any](buffer *VertexBuffer, data []T) error {
	return VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(buffer, 0, data, 0, int32(len(data)), 0)
}

// VertexBufferSetDataBySliceOfTAndInt32AndInt32 is
// VertexBuffer::SetData<T>(T[], int32, int32).
func VertexBufferSetDataBySliceOfTAndInt32AndInt32[T any](buffer *VertexBuffer, data []T, startIndex, elementCount int32) error {
	return VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(buffer, 0, data, startIndex, elementCount, 0)
}

// VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32 is
// VertexBuffer::SetData<T>(int32 offsetInBytes, T[], int32, int32, int32),
// which the other two funnel into with a ZERO stride -- and zero means
// "tightly packed", not "no stride".
func VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32[T any](
	buffer *VertexBuffer, offsetInBytes int32, data []T, startIndex, elementCount, vertexStride int32,
) error {
	byteCount, nativeVertices, err := prepareVertexTransfer[T](buffer, data, startIndex, elementCount, offsetInBytes, vertexStride, true)
	if err != nil {
		return err
	}
	resource := buffer.nativeResource()
	if resource == nil {
		return errVertexBufferNil
	}
	return resource.SetVertexDataRaw(uint64(offsetInBytes),
		unsafe.Pointer(&data[startIndex]), byteCount, nativeVertices, uint32(buffer.vertexStride))
}

// VertexBufferGetDataBySliceOfT is VertexBuffer::GetData<T>(T[]).
func VertexBufferGetDataBySliceOfT[T any](buffer *VertexBuffer, data []T) error {
	return VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(buffer, 0, data, 0, int32(len(data)), 0)
}

// VertexBufferGetDataBySliceOfTAndInt32AndInt32 is
// VertexBuffer::GetData<T>(T[], int32, int32).
func VertexBufferGetDataBySliceOfTAndInt32AndInt32[T any](buffer *VertexBuffer, data []T, startIndex, elementCount int32) error {
	return VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(buffer, 0, data, startIndex, elementCount, 0)
}

// VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32 is
// VertexBuffer::GetData<T>(int32 offsetInBytes, T[], int32, int32, int32).
//
// CNA's `cna_vertex_buffer_get_data_raw` takes a BUFFER offset too, so the read
// path needs no special case -- unlike IndexBuffer's, whose CNA read always
// begins at index zero.
func VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32[T any](
	buffer *VertexBuffer, offsetInBytes int32, data []T, startIndex, elementCount, vertexStride int32,
) error {
	byteCount, nativeVertices, err := prepareVertexTransfer[T](buffer, data, startIndex, elementCount, offsetInBytes, vertexStride, false)
	if err != nil {
		return err
	}
	if buffer.BufferUsage() == BufferUsageWriteOnly {
		return fmt.Errorf("%w: %s", errVertexTransferNotSupported, writeOnlyGetNotSupported)
	}
	resource := buffer.nativeResource()
	if resource == nil {
		return errVertexBufferNil
	}
	return resource.GetVertexDataRaw(uint64(offsetInBytes),
		unsafe.Pointer(&data[startIndex]), byteCount, nativeVertices, uint32(buffer.vertexStride))
}

// prepareVertexTransfer is CopyData's guard prefix, shared by both directions.
//
// It reports the BYTE COUNT the transfer moves and the vertex count CNA must be
// told, which are not the same number as the reference's `elementCount`.
//
// # CNA counts in the BUFFER's vertices, not in T
//
// A measured constraint, not a guess: `cna_vertex_buffer_set_data_raw_at`
// refuses a stride that is not the buffer's own --
//
//	"The vertex stride does not match this VertexBuffer's VertexDeclaration."
//
// -- so the stride handed to CNA is always the stride CNA reported for the
// buffer, and the vertex count is the byte count divided by it. For the
// ordinary case where sizeof(T) IS the declaration's stride these are the same
// numbers the reference uses. For a buffer whose stride is wider, four 16-byte
// values fill two 32-byte vertices, which is exactly what the reference's
// single memcpy does with the same bytes.
//
// A byte count that is not a whole number of the buffer's vertices cannot be
// expressed to CNA at all, and is refused rather than rounded.
func prepareVertexTransfer[T any](
	buffer *VertexBuffer, data []T, startIndex, elementCount, offsetInBytes, vertexStride int32, setting bool,
) (uint64, uint64, error) {
	if buffer == nil || buffer.resource == nil {
		return 0, 0, errVertexBufferNil
	}
	if buffer.IsDisposed() {
		return 0, 0, fmt.Errorf("%w: VertexBuffer", interop.ErrDisposed)
	}
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("%w: data: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if err := validateCopyParameters(int32(len(data)), startIndex, elementCount); err != nil {
		return 0, 0, err
	}
	elementSize := vertexElementSize[T]()
	// The reference's arithmetic, exactly:
	//
	//	bytes = sizeof(T) * elementCount;
	//	if (vertexStride != 0) {
	//	    padding = vertexStride - sizeof(T);
	//	    if (padding < 0) throw ("vertexStride", VertexStrideTooSmall);
	//	    if (elementCount > 1) bytes += (elementCount - 1) * padding;
	//	}
	bytes := int64(elementSize) * int64(elementCount)
	if vertexStride != 0 {
		padding := vertexStride - elementSize
		if padding < 0 {
			return 0, 0, fmt.Errorf("%w: vertexStride: %s", errArgumentOutOfRange, vertexStrideTooSmall)
		}
		if padding > 0 {
			// The measured upstream limit. See errVertexStrideUnsupported.
			return 0, 0, fmt.Errorf("%w: stride %d for a %d-byte element", errVertexStrideUnsupported, vertexStride, elementSize)
		}
		if elementCount > 1 {
			bytes += int64(elementCount-1) * int64(padding)
		}
	}
	if offsetInBytes < 0 {
		return 0, 0, fmt.Errorf("%w: offsetInBytes: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if bytes+int64(offsetInBytes) > buffer.bufferByteSize() {
		return 0, 0, fmt.Errorf("%w: %s", errVertexTransferInvalidOperation, resourceDataMustBeCorrectSize)
	}
	bufferStride := int64(buffer.vertexStride)
	if bufferStride <= 0 {
		return 0, 0, errVertexBufferNil
	}
	if bytes%bufferStride != 0 || int64(offsetInBytes)%bufferStride != 0 {
		return 0, 0, fmt.Errorf("%w: %d bytes at offset %d is not a whole number of this buffer's %d-byte vertices",
			errVertexPartialVertexUnsupported, bytes, offsetInBytes, bufferStride)
	}
	_ = setting
	return uint64(bytes), uint64(bytes / bufferStride), nil
}

// errVertexPartialVertexUnsupported is the second measured narrowing, and it
// comes from the same place the first does. CNA's raw transfer is described in
// WHOLE VERTICES of the buffer's own stride, and refuses any other stride by
// name, so a transfer covering part of a vertex -- or starting part-way into
// one -- has no expression in this ABI. XNA's is a byte memcpy and has no such
// rule.
var errVertexPartialVertexUnsupported = errors.New(
	"CNA describes a raw vertex transfer in whole vertices of the buffer's own stride, and 0.21.0 publishes no byte-granular route")
