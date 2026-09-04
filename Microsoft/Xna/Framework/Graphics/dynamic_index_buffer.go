package graphics

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// DynamicIndexBuffer is an index buffer a game rewrites every frame.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
//	.class public auto ansi beforefieldinit DynamicIndexBuffer
//	       extends IndexBuffer
//	  .field assembly bool _contentLost
//	  .field private EventHandler`1<EventArgs> <backing_store>ContentLost
//
// It is DynamicVertexBuffer's exact counterpart -- same two added members, same
// latching property, same assembly-internal raise site -- and differs from it
// in one measured place: the CONSTRUCTOR GUARD ORDER. See the constructors.
type DynamicIndexBuffer struct {
	// buffer is the composed IndexBuffer, which carries the composed
	// GraphicsResource and the one native handle.
	buffer *IndexBuffer

	// contentLost is `_contentLost`.
	contentLost bool
	// contentLostEvent is the `<backing_store>ContentLost` delegate field.
	contentLostEvent framework.EventSource[*framework.EventArgs]
}

// errDynamicIndexBufferNil is the Go-only guard for a zero value.
var errDynamicIndexBufferNil = errors.New("DynamicIndexBuffer is nil or uninitialized")

// NewDynamicIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage
// is DynamicIndexBuffer::.ctor(GraphicsDevice, IndexElementSize, Int32,
// BufferUsage), 89 bytes, and its FIRST statement is the count check:
//
//	if (indexCount <= 0)
//	    throw new ArgumentOutOfRangeException("indexCount", ResourcesMustBeGreaterThanZeroSize);
//
// which is its base's order too. This is where the two dynamic buffers diverge:
// DynamicVertexBuffer's declaration-keyed constructor checks its DECLARATION
// first, because a null one would crash the creation; an IndexElementSize is a
// value type and cannot be null, so there is nothing to check ahead of the
// count.
func NewDynamicIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, indexElementSize IndexElementSize, indexCount int32, usage BufferUsage,
) (*DynamicIndexBuffer, error) {
	buffer, err := newIndexBufferWithDynamic(graphicsDevice, indexElementSize, indexCount, usage, true)
	if err != nil {
		return nil, err
	}
	dynamic := &DynamicIndexBuffer{buffer: buffer}
	buffer.bindDerived(dynamic)
	return dynamic, nil
}

// NewDynamicIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage is
// DynamicIndexBuffer::.ctor(GraphicsDevice, Type, Int32, BufferUsage), 96
// bytes, and it too checks the COUNT first, before resolving the type:
//
//	if (indexCount <= 0) throw ...;
//	base(graphicsDevice, indexType, indexCount, usage)
//
// The opposite of the vertex side's Type-keyed constructor, which resolves the
// type first. A caller passing both a bad type and a bad count is told about
// the COUNT here and about the TYPE there -- a difference no reader would
// predict, and the reason both are written out rather than shared.
func NewDynamicIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, indexType reflect.Type, indexCount int32, usage BufferUsage,
) (*DynamicIndexBuffer, error) {
	if indexCount <= 0 {
		return nil, fmt.Errorf("%w: indexCount: %s", errArgumentOutOfRange, resourcesMustBeGreaterThanZeroSize)
	}
	size, err := indexElementSizeForType(indexType)
	if err != nil {
		return nil, err
	}
	return NewDynamicIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		graphicsDevice, size, indexCount, usage)
}

// clrTypeName is System.Object::ToString's answer for a DynamicIndexBuffer.
func (b *DynamicIndexBuffer) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer"
}

// indexBufferBase is the composed base.
func (b *DynamicIndexBuffer) indexBufferBase() *IndexBuffer {
	if b == nil {
		return nil
	}
	return b.buffer
}

// IsContentLost is DynamicIndexBuffer::get_IsContentLost, 32 bytes, the same
// latch DynamicVertexBuffer's is:
//
//	if (!_contentLost) _contentLost = GraphicsDevice.IsDeviceLost;
//	return _contentLost;
//
// CNA's `CNA_IndexBufferInfo.is_content_lost` plays the device's part and is
// documented as always false, so on a qualified artifact this answers false and
// the latch is exercised only through the field. See DynamicVertexBuffer
// .IsContentLost for why that is a property of the environment rather than of
// the projection.
func (b *DynamicIndexBuffer) IsContentLost() (bool, error) {
	if b == nil || b.buffer == nil {
		return false, errDynamicIndexBufferNil
	}
	if b.contentLost {
		return true, nil
	}
	resource := b.buffer.nativeResource()
	if resource == nil {
		return false, interop.ErrDisposed
	}
	info, err := resource.IndexBufferInfo()
	if err != nil {
		return false, err
	}
	if info.IsContentLost {
		b.contentLost = true
	}
	return b.contentLost, nil
}

// setContentLost is DynamicIndexBuffer::SetContentLost(bool), byte for byte
// DynamicVertexBuffer's: store, and raise only when the flag is true.
func (b *DynamicIndexBuffer) setContentLost(isContentLost bool) {
	b.contentLost = isContentLost
	if isContentLost {
		_ = b.contentLostEvent.Raise(b, framework.EventArgsEmpty())
	}
}

// AddContentLostHandler is add_ContentLost. The raise site is the same
// assembly-internal SetContentLost the vertex side has, and CNA's
// `cna_index_buffer_subscribe_content_lost` documents the same renderer
// restriction: only DirectX9, Direct2D and Skia can lose a device. Neither
// qualified artifact can, so the event cannot fire in this environment.
func (b *DynamicIndexBuffer) AddContentLostHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errDynamicIndexBufferNil
	}
	return b.contentLostEvent.Add(handler)
}

// RemoveContentLostHandler is remove_ContentLost.
func (b *DynamicIndexBuffer) RemoveContentLostHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errDynamicIndexBufferNil
	}
	return b.contentLostEvent.Remove(subscription)
}

// DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions is
// DynamicIndexBuffer::SetData<T>(T[], Int32, Int32, SetDataOptions), which
// forwards with a zero buffer offset.
func DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions[T any](
	buffer *DynamicIndexBuffer, data []T, startIndex, elementCount int32, options SetDataOptions,
) error {
	return DynamicIndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndSetDataOptions(
		buffer, 0, data, startIndex, elementCount, options)
}

// DynamicIndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndSetDataOptions
// is DynamicIndexBuffer::SetData<T>(Int32, T[], Int32, Int32, SetDataOptions),
// which converts the options and calls the base's CopyData.
//
// # The index side needed no new route
//
// CNA's `cna_index_buffer_set_data` and `..._set_data_at` already carry an
// `options` argument, and the static IndexBuffer overloads pass a hardcoded
// zero because the reference hardcodes SetDataOptions.None there. So this
// member is the SAME two routes with the caller's options instead of a
// constant. The vertex side is where a route was missing, and that asymmetry is
// CNA's, not a projection choice.
func DynamicIndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndSetDataOptions[T any](
	buffer *DynamicIndexBuffer, offsetInBytes int32, data []T, startIndex, elementCount int32, options SetDataOptions,
) error {
	base := buffer.indexBufferBase()
	if base == nil {
		return errDynamicIndexBufferNil
	}
	width, size, err := prepareIndexTransfer[T](base, data, startIndex, elementCount, offsetInBytes, true)
	if err != nil {
		return err
	}
	resource := base.nativeResource()
	if resource == nil {
		return errIndexBufferNil
	}
	_ = width
	var uploadErr error
	if offsetInBytes == 0 {
		uploadErr = resource.SetIndexData(nativeIndexElementSize(size), nativeSetDataOptions(options),
			uint64(startIndex), uint64(elementCount), unsafe.Pointer(&data[0]), uint64(len(data)))
	} else {
		uploadErr = resource.SetIndexDataAt(uint64(offsetInBytes), nativeIndexElementSize(size), nativeSetDataOptions(options),
			uint64(startIndex), uint64(elementCount), unsafe.Pointer(&data[0]), uint64(len(data)))
	}
	if uploadErr != nil {
		return uploadErr
	}
	base.noteContentRestored()
	return nil
}

// ---------------------------------------------------------------------------
// The inherited public surface of IndexBuffer, forwarded. The generic
// SetData/GetData overloads are absent for the settled reason: they project as
// package functions naming their DECLARING type, and those already accept this
// buffer through the composed base.
// ---------------------------------------------------------------------------

// IndexCount is IndexBuffer::get_IndexCount.
func (b *DynamicIndexBuffer) IndexCount() int32 {
	if b == nil {
		return 0
	}
	return b.buffer.IndexCount()
}

// IndexElementSize is IndexBuffer::get_IndexElementSize.
func (b *DynamicIndexBuffer) IndexElementSize() IndexElementSize {
	if b == nil {
		return IndexElementSizeSixteenBits
	}
	return b.buffer.IndexElementSize()
}

// BufferUsage is IndexBuffer::get_BufferUsage.
func (b *DynamicIndexBuffer) BufferUsage() BufferUsage {
	if b == nil {
		return BufferUsageNone
	}
	return b.buffer.BufferUsage()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (b *DynamicIndexBuffer) GraphicsDevice() *GraphicsDevice {
	if b == nil {
		return nil
	}
	return b.buffer.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (b *DynamicIndexBuffer) Name() string {
	if b == nil {
		return ""
	}
	return b.buffer.Name()
}

// SetName is GraphicsResource::set_Name.
func (b *DynamicIndexBuffer) SetName(value string) {
	if b == nil {
		return
	}
	b.buffer.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (b *DynamicIndexBuffer) Tag() any {
	if b == nil {
		return nil
	}
	return b.buffer.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (b *DynamicIndexBuffer) SetTag(value any) {
	if b == nil {
		return
	}
	b.buffer.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (b *DynamicIndexBuffer) IsDisposed() bool {
	if b == nil {
		return true
	}
	return b.buffer.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (b *DynamicIndexBuffer) ToString() string {
	if b == nil {
		return ""
	}
	return b.buffer.ToString()
}

// AddDisposingHandler is add_Disposing.
func (b *DynamicIndexBuffer) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errDynamicIndexBufferNil
	}
	return b.buffer.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (b *DynamicIndexBuffer) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errDynamicIndexBufferNil
	}
	return b.buffer.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), this type's only dispose member: it
// declares no Dispose of its own, so its inherited PUBLIC surface carries the
// no-argument one and not the protected overload.
func (b *DynamicIndexBuffer) Dispose() error {
	if b == nil {
		return errDynamicIndexBufferNil
	}
	return b.buffer.DisposeByNone()
}
