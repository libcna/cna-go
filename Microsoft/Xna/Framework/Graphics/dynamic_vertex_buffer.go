package graphics

import (
	"errors"
	"reflect"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// DynamicVertexBuffer is a vertex buffer a game rewrites every frame.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
//	.class public auto ansi beforefieldinit DynamicVertexBuffer
//	       extends VertexBuffer
//	  .field assembly bool _contentLost
//	  .field private EventHandler`1<EventArgs> <backing_store>ContentLost
//
// # What it adds to its base, which is less than it looks
//
// The type declares two constructors, two SetData overloads, one property and
// one event, and every one of them is thin:
//
//	.ctor        the base's creation with D3DUSAGE_DYNAMIC (0x200) set
//	SetData      the base's CopyData with a SetDataOptions the base cannot take
//	IsContentLost a LATCHING read
//	ContentLost   a delegate field raised from the device's reset path
//
// The base's own thirteen members are inherited unchanged, and the composed
// VertexBuffer carries all of them.
//
// # The dynamic flag is one field in CNA's create info
//
// The Foundation 83 probe measured this against the canonical headers rather
// than guessing it: `CNA_VertexBufferCreateInfo` carries a `dynamic` field
// documented as "True to construct DynamicVertexBuffer; false to construct
// VertexBuffer", and Foundation 66 already plumbed it through. So this type
// creates its buffer with the SAME route as its base and one flag flipped,
// which is what the reference does with D3DUSAGE_DYNAMIC.
type DynamicVertexBuffer struct {
	// buffer is the composed VertexBuffer, which carries the composed
	// GraphicsResource and the one native handle.
	buffer *VertexBuffer

	// contentLost is `_contentLost`, the reference's own latching field.
	contentLost bool
	// contentLostEvent is the `<backing_store>ContentLost` delegate field.
	contentLostEvent framework.EventSource[*framework.EventArgs]
}

// errDynamicVertexBufferNil is the Go-only guard for a zero value.
var errDynamicVertexBufferNil = errors.New("DynamicVertexBuffer is nil or uninitialized")

// NewDynamicVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage
// is DynamicVertexBuffer::.ctor(GraphicsDevice, VertexDeclaration, Int32,
// BufferUsage), 116 bytes:
//
//	if (vertexDeclaration == null)
//	    throw new ArgumentNullException("vertexDeclaration", NullNotAllowed);
//	if (vertexCount <= 0)
//	    throw new ArgumentOutOfRangeException("vertexCount", ResourcesMustBeGreaterThanZeroSize);
//	... CreateBuffer(..., D3DUSAGE_DYNAMIC, ...) ...
//
// Same two guards as the base's, in the same order, and the only difference at
// the creation is the usage flag.
func NewDynamicVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, vertexDeclaration *VertexDeclaration, vertexCount int32, usage BufferUsage,
) (*DynamicVertexBuffer, error) {
	buffer, err := newVertexBufferWithDynamic(graphicsDevice, vertexDeclaration, vertexCount, usage, true)
	if err != nil {
		return nil, err
	}
	dynamic := &DynamicVertexBuffer{buffer: buffer}
	buffer.bindDerived(dynamic)
	return dynamic, nil
}

// NewDynamicVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage is
// DynamicVertexBuffer::.ctor(GraphicsDevice, Type, Int32, BufferUsage), 104
// bytes, and its guard ORDER is the base's rather than its sibling's: the
// declaration is resolved from the type FIRST, so a caller passing both a bad
// type and a bad count is told about the type.
func NewDynamicVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, vertexType reflect.Type, vertexCount int32, usage BufferUsage,
) (*DynamicVertexBuffer, error) {
	declaration, err := vertexDeclarationFromType(vertexType)
	if err != nil {
		return nil, err
	}
	return NewDynamicVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		graphicsDevice, declaration, vertexCount, usage)
}

// clrTypeName is System.Object::ToString's answer for a DynamicVertexBuffer.
func (b *DynamicVertexBuffer) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer"
}

// vertexBufferBase is the composed base, for the package functions that take a
// buffer and for the reference-interface machinery.
func (b *DynamicVertexBuffer) vertexBufferBase() *VertexBuffer {
	if b == nil {
		return nil
	}
	return b.buffer
}

// IsContentLost is DynamicVertexBuffer::get_IsContentLost, 32 bytes, and it
// LATCHES:
//
//	if (!_contentLost) _contentLost = GraphicsDevice.IsDeviceLost;
//	return _contentLost;
//
// Once true it stays true, and while false it re-reads the device every call.
// The projection asks CNA the same question -- `CNA_VertexBufferInfo
// .is_content_lost` -- and latches it the same way, so the Go field is the
// reference's field and CNA's report plays the device's part. It is the same
// shape RenderTarget2D's already has.
//
// # It can never answer true on a qualified artifact
//
// CNA documents the field as "Dynamic content-loss state; currently always
// false". So this member reports false on HEADLESS and SOFTWARE whatever
// happens, and the latch is exercised only by the field. That is a measured
// limitation of the qualified environment, not of the projection: the reference
// answers true only after a real D3D9 device loss, which neither artifact can
// produce either.
//
// It is FALLIBLE because the read reaches CNA, which the reference's read of
// GraphicsDevice::IsDeviceLost reaches D3D for.
func (b *DynamicVertexBuffer) IsContentLost() (bool, error) {
	if b == nil || b.buffer == nil {
		return false, errDynamicVertexBufferNil
	}
	if b.contentLost {
		return true, nil
	}
	resource := b.buffer.nativeResource()
	if resource == nil {
		return false, interop.ErrDisposed
	}
	info, err := resource.VertexBufferInfo()
	if err != nil {
		return false, err
	}
	if info.IsContentLost {
		b.contentLost = true
	}
	return b.contentLost, nil
}

// setContentLost is DynamicVertexBuffer::SetContentLost(bool), the
// IDynamicGraphicsResource member, 23 bytes:
//
//	ldarg.0; ldarg.1; stfld _contentLost
//	ldarg.1; brfalse ret
//	ldarg.0; dup; ldsfld EventArgs::Empty; callvirt raise_ContentLost
//
// FALSE stores and returns; TRUE stores and raises. Both device reset and a
// successful upload reach it, with true and false respectively, so this one
// member is both the only raise site the event has and the only thing that ever
// clears the latch.
//
// The raise's error is dropped because the reference's `callvirt` has no result
// -- a handler that throws propagates out of SetContentLost, and the member the
// projection is reached from, SetData, already carries an error for the upload.
// A handler's failure is not the upload's.
func (b *DynamicVertexBuffer) setContentLost(isContentLost bool) {
	b.contentLost = isContentLost
	if isContentLost {
		_ = b.contentLostEvent.Raise(b, framework.EventArgsEmpty())
	}
}

// AddContentLostHandler is add_ContentLost, on the settled two-accessor event
// projection.
//
// # The event has no raise site a qualified artifact can reach
//
// The reference raises it from `SetContentLost(bool)`, an assembly-internal
// member the device calls on a reset -- it stores the flag and, when true,
// invokes the delegate with EventArgs.Empty. CNA has a counterpart,
// `cna_vertex_buffer_subscribe_content_lost`, and its documentation is explicit
// that only the DirectX9, Direct2D and Skia renderer families can lose a device;
// families that cannot "never raise it". The qualified artifacts are HEADLESS
// and SOFTWARE, so the event cannot fire in the qualified environment at all.
//
// The accessors are projected because the contract declares them and the
// registration list is real, and the route stays unbound for the reason
// RenderTarget2D's does. What is recorded rather than claimed is that no
// qualified environment can deliver one.
func (b *DynamicVertexBuffer) AddContentLostHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errDynamicVertexBufferNil
	}
	return b.contentLostEvent.Add(handler)
}

// RemoveContentLostHandler is remove_ContentLost.
func (b *DynamicVertexBuffer) RemoveContentLostHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errDynamicVertexBufferNil
	}
	return b.contentLostEvent.Remove(subscription)
}

// ---------------------------------------------------------------------------
// The two SetData overloads, which are the whole reason the type exists.
// ---------------------------------------------------------------------------

// DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions is
// DynamicVertexBuffer::SetData<T>(T[], Int32, Int32, SetDataOptions), 14 bytes
// -- it forwards to the offset overload with a zero offset and a zero stride,
// exactly as the base's short overloads forward to its long one.
func DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions[T any](
	buffer *DynamicVertexBuffer, data []T, startIndex, elementCount int32, options SetDataOptions,
) error {
	return DynamicVertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32AndSetDataOptions(
		buffer, 0, data, startIndex, elementCount, 0, options)
}

// DynamicVertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32AndSetDataOptions
// is DynamicVertexBuffer::SetData<T>(Int32, T[], Int32, Int32, Int32,
// SetDataOptions), 22 bytes: it converts the options and calls the base's
// CopyData with the dynamic flag set.
//
// The guard prefix is the BASE's -- the same prepareVertexTransfer every static
// overload runs -- because the reference reaches the same CopyData. What
// differs is one argument, and it is the one CNA's plain upload route cannot
// carry.
func DynamicVertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32AndSetDataOptions[T any](
	buffer *DynamicVertexBuffer, offsetInBytes int32, data []T, startIndex, elementCount, vertexStride int32,
	options SetDataOptions,
) error {
	base := buffer.vertexBufferBase()
	if base == nil {
		return errDynamicVertexBufferNil
	}
	byteCount, nativeVertices, err := prepareVertexTransfer[T](base, data, startIndex, elementCount, offsetInBytes, vertexStride, true)
	if err != nil {
		return err
	}
	resource := base.nativeResource()
	if resource == nil {
		return errVertexBufferNil
	}
	if err := resource.SetVertexDataRawWithOptions(uint64(offsetInBytes),
		unsafe.Pointer(&data[startIndex]), byteCount, nativeVertices,
		uint32(base.vertexStride), nativeSetDataOptions(options)); err != nil {
		return err
	}
	// The base's CopyData tail, which this overload reaches through the same
	// body: a successful upload declares the content no longer lost.
	base.noteContentRestored()
	return nil
}

// ---------------------------------------------------------------------------
// The inherited public surface of VertexBuffer, forwarded.
//
// The three generic SetData/GetData overloads are ABSENT on purpose: the
// settled generic-method rule projects them as package functions naming the
// DECLARING type, and `VertexBufferSetDataBySliceOfT(base, ...)` already
// accepts this type through the composed base. There is no second Go identity
// to project for them.
// ---------------------------------------------------------------------------

// VertexDeclaration is VertexBuffer::get_VertexDeclaration.
func (b *DynamicVertexBuffer) VertexDeclaration() *VertexDeclaration {
	if b == nil {
		return nil
	}
	return b.buffer.VertexDeclaration()
}

// VertexCount is VertexBuffer::get_VertexCount.
func (b *DynamicVertexBuffer) VertexCount() int32 {
	if b == nil {
		return 0
	}
	return b.buffer.VertexCount()
}

// BufferUsage is VertexBuffer::get_BufferUsage.
func (b *DynamicVertexBuffer) BufferUsage() BufferUsage {
	if b == nil {
		return BufferUsageNone
	}
	return b.buffer.BufferUsage()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (b *DynamicVertexBuffer) GraphicsDevice() *GraphicsDevice {
	if b == nil {
		return nil
	}
	return b.buffer.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (b *DynamicVertexBuffer) Name() string {
	if b == nil {
		return ""
	}
	return b.buffer.Name()
}

// SetName is GraphicsResource::set_Name.
func (b *DynamicVertexBuffer) SetName(value string) {
	if b == nil {
		return
	}
	b.buffer.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (b *DynamicVertexBuffer) Tag() any {
	if b == nil {
		return nil
	}
	return b.buffer.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (b *DynamicVertexBuffer) SetTag(value any) {
	if b == nil {
		return
	}
	b.buffer.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (b *DynamicVertexBuffer) IsDisposed() bool {
	if b == nil {
		return true
	}
	return b.buffer.IsDisposed()
}

// ToString is GraphicsResource::ToString, which answers this type's name.
func (b *DynamicVertexBuffer) ToString() string {
	if b == nil {
		return ""
	}
	return b.buffer.ToString()
}

// AddDisposingHandler is add_Disposing.
func (b *DynamicVertexBuffer) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errDynamicVertexBufferNil
	}
	return b.buffer.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (b *DynamicVertexBuffer) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errDynamicVertexBufferNil
	}
	return b.buffer.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), and this type's ONLY dispose member:
// it declares no Dispose of its own, so its inherited PUBLIC surface carries
// one that takes no argument -- the same split the stock effects have against
// Effect, which DOES declare the protected overload.
func (b *DynamicVertexBuffer) Dispose() error {
	if b == nil {
		return errDynamicVertexBufferNil
	}
	return b.buffer.DisposeByNone()
}
