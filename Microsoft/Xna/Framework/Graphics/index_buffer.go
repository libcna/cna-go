package graphics

import (
	"errors"
	"fmt"
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 65 — IndexBuffer.
// ---------------------------------------------------------------------------

// IndexBuffer is Microsoft.Xna.Framework.Graphics.IndexBuffer:
//
//	.class public auto ansi beforefieldinit IndexBuffer
//	       extends Microsoft.Xna.Framework.Graphics.GraphicsResource
//	       implements Microsoft.Xna.Framework.Graphics.IGraphicsResource
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_index_buffer_destroy.
//
// The composed GraphicsResource holds the one CNA handle, exactly as it does
// for a texture, and the buffer is created on the device's owner OS thread from
// inside a lifecycle callback -- which CNA requires, because
// cna_index_buffer_create takes a callback-scoped device handle.
//
// # Its three properties are field reads
//
//	get_IndexCount        ldarg.0; ldfld _indexCount; ret
//	get_IndexElementSize  _indexSize == 2 ? SixteenBits : ThirtyTwoBits
//	get_BufferUsage       ConvertDxBufferUsageToXna(_usage)
//
// None reaches D3D and none checks disposal, so all three answer after Dispose.
// CNA-Go records what the constructor asked for and what CNA reported, and
// answers from those fields for the same reason: asking CNA per call would make
// three infallible members fallible.
type IndexBuffer struct {
	// resource is the composed GraphicsResource, which owns the CNA handle.
	resource *GraphicsResource
	// The three values the reference keeps as private fields.
	indexCount       int32
	indexElementSize IndexElementSize
	bufferUsage      BufferUsage
}

// errIndexBufferNil is the Go-only guard for a zero IndexBuffer.
var errIndexBufferNil = errors.New("IndexBuffer is nil or uninitialized")

// errIndexBufferDeviceRequired is the OTHER Go-only guard, and the reason it
// exists is measured rather than assumed: IndexBuffer's constructors do not
// null-check their device, so the reference throws NullReferenceException from
// a later dereference. Go cannot project that, and this says so in its own
// words instead of borrowing another member's message.
var errIndexBufferDeviceRequired = errors.New("the graphics device must not be nil")

// The three FrameworkResources strings IndexBuffer's own members throw.
const (
	resourcesMustBeGreaterThanZeroSize = "Resource size must be greater than zero."
	writeOnlyGetNotSupported           = "Calling GetData on a resource that was created with BufferUsage.WriteOnly is not supported."
	resourceDataMustBeCorrectSize      = "The array is not the correct size for the amount of data requested."
	mustBeValidIndex                   = "This parameter must be a valid index within the array."
)

// NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage is
// IndexBuffer::.ctor(GraphicsDevice, IndexElementSize, int32, BufferUsage):
//
//	if (indexCount <= 0)
//	    throw new ArgumentOutOfRangeException("indexCount",
//	        FrameworkResources.ResourcesMustBeGreaterThanZeroSize);
//	_parent = graphicsDevice;
//	CreateBuffer(indexCount, indexElementSize == SixteenBits ? 2 : 4,
//	             ConvertXnaBufferUsageToDx(usage), D3DPOOL_MANAGED);
//
// Two things the IL says that a reader would otherwise guess wrong.
//
// The count check runs BEFORE the device is stored, so a bad count is refused
// whatever the device is. And there is NO null check on graphicsDevice: the
// reference dereferences it later and throws NullReferenceException. Go cannot
// project an NRE, so a nil device is refused here by a Go-only guard, which is
// named as such rather than dressed up as the reference's behaviour.
//
// The width mapping is the reference's own `beq` on SixteenBits: that literal
// alone means two bytes and EVERY other value -- including one outside the
// enum -- means four.
func NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, indexElementSize IndexElementSize, indexCount int32, usage BufferUsage,
) (*IndexBuffer, error) {
	return newIndexBufferWithDynamic(graphicsDevice, indexElementSize, indexCount, usage, false)
}

// newIndexBufferWithDynamic is the body this type's constructor and
// DynamicIndexBuffer's share, for the reason the vertex side has one: the
// reference's two constructors differ by a single D3DUSAGE_DYNAMIC bit into one
// private CreateBuffer, and CNA's create-info differs by a single `dynamic`
// field. One body with one flag cannot drift; two bodies could.
func newIndexBufferWithDynamic(
	graphicsDevice *GraphicsDevice, indexElementSize IndexElementSize, indexCount int32, usage BufferUsage,
	dynamic bool,
) (*IndexBuffer, error) {
	if indexCount <= 0 {
		return nil, fmt.Errorf("%w: indexCount: %s", errArgumentOutOfRange, resourcesMustBeGreaterThanZeroSize)
	}
	if graphicsDevice == nil {
		// A GO-ONLY refusal, with its own words. The reference has NO device
		// check here: it stores the argument and dereferences it two
		// statements later, so C# gets a NullReferenceException. Borrowing
		// DeviceCannotBeNullOnResourceCreate -- which Texture2D's constructor
		// really does throw -- would attribute a message to a throw site that
		// does not have one.
		return nil, fmt.Errorf("%w: graphicsDevice", errIndexBufferDeviceRequired)
	}
	device, err := graphicsDevice.live()
	if err != nil {
		return nil, err
	}
	resource, err := device.CreateIndexBuffer(indexCount, nativeIndexElementSize(indexElementSize), nativeBufferUsage(usage), dynamic)
	if err != nil {
		return nil, err
	}
	// CNA reports what it applied. The reference reads its own fields back the
	// same way -- `_indexSize` is what CreateBuffer stored -- so the projection
	// records CNA's answer rather than the request, and a renderer that
	// widened an index would be visible instead of hidden.
	info, err := resource.IndexBufferInfo()
	if err != nil {
		_ = resource.Dispose()
		return nil, err
	}
	return newIndexBuffer(graphicsDevice, resource, info), nil
}

// NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage is
// IndexBuffer::.ctor(GraphicsDevice, Type, int32, BufferUsage), whose one
// difference is where the element width comes from:
//
//	CreateBuffer(indexCount, Marshal.SizeOf(indexType), ...)
//
// `Marshal.SizeOf` is the UNMANAGED size of the type, so `typeof(short)` is two
// and `typeof(int)` is four -- and any other type is whatever its marshalled
// layout happens to be, which for a 3-byte struct would produce a buffer XNA's
// own IndexElementSize cannot name.
//
// CNA's index buffer stores 16-bit or 32-bit elements and nothing else, so the
// accepted set is CLOSED at the four Go types whose unmanaged size is 2 or 4
// and which an index actually is. A type outside it is refused BY NAME, before
// the device is reached -- which is more useful than letting CNA refuse a width
// it was never told about.
func NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, indexType reflect.Type, indexCount int32, usage BufferUsage,
) (*IndexBuffer, error) {
	if indexCount <= 0 {
		return nil, fmt.Errorf("%w: indexCount: %s", errArgumentOutOfRange, resourcesMustBeGreaterThanZeroSize)
	}
	size, err := indexElementSizeForType(indexType)
	if err != nil {
		return nil, err
	}
	return NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		graphicsDevice, size, indexCount, usage)
}

// errUnsupportedIndexType is the refusal for a Type outside the closed set. The
// reference has no counterpart: `Marshal.SizeOf` accepts any blittable type and
// produces a buffer whose element width XNA then cannot name, so the difference
// is recorded rather than hidden.
var errUnsupportedIndexType = errors.New("an index buffer element must be a 16-bit or 32-bit integer")

// indexElementSizeForType is the projection of `Marshal.SizeOf(indexType)`,
// narrowed to the widths CNA stores. The four Go types are the ones a C#
// caller's `typeof(short)`, `typeof(ushort)`, `typeof(int)` and `typeof(uint)`
// correspond to.
func indexElementSizeForType(indexType reflect.Type) (IndexElementSize, error) {
	if indexType == nil {
		return 0, fmt.Errorf("%w: indexType: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	switch indexType.Kind() {
	case reflect.Int16, reflect.Uint16:
		return IndexElementSizeSixteenBits, nil
	case reflect.Int32, reflect.Uint32:
		return IndexElementSizeThirtyTwoBits, nil
	default:
		return 0, fmt.Errorf("%w: %s", errUnsupportedIndexType, indexType.String())
	}
}

// nativeIndexElementSize maps XNA's literal onto CNA's identity EXPLICITLY.
// The two numberings agree today, and a cast would make that agreement
// invisible: this is the one place the correspondence is stated, so a change on
// either side is a compile error rather than a silently reinterpreted buffer.
func nativeIndexElementSize(value IndexElementSize) uint32 {
	if value == IndexElementSizeThirtyTwoBits {
		return interop.IndexElementSizeThirtyTwoBits
	}
	// Every other value takes the sixteen-bit branch, which is the reference's
	// own `beq SixteenBits` inverted -- except that the REFERENCE widens the
	// unknown value and CNA-Go narrows it. See indexElementWidth for why the
	// two disagree and which one this follows.
	return interop.IndexElementSizeSixteenBits
}

// nativeBufferUsage maps XNA's BufferUsage onto CNA's identity, explicitly and
// for the same reason.
func nativeBufferUsage(value BufferUsage) uint32 {
	if value == BufferUsageWriteOnly {
		return 1
	}
	return 0
}

// newIndexBuffer composes the base, installs the CLR `this` and records what
// CNA applied.
func newIndexBuffer(device *GraphicsDevice, resource *interop.Resource, info interop.IndexBufferInfo) *IndexBuffer {
	size := IndexElementSizeSixteenBits
	if info.IndexElementSize == interop.IndexElementSizeThirtyTwoBits {
		size = IndexElementSizeThirtyTwoBits
	}
	usage := BufferUsageNone
	if info.BufferUsage == 1 {
		usage = BufferUsageWriteOnly
	}
	buffer := &IndexBuffer{
		resource:         newGraphicsResource(device, resource),
		indexCount:       info.IndexCount,
		indexElementSize: size,
		bufferUsage:      usage,
	}
	buffer.resource.bindDerived(buffer)
	return buffer
}

// IndexCount is IndexBuffer::get_IndexCount, one field read.
func (b *IndexBuffer) IndexCount() int32 {
	if b == nil {
		return 0
	}
	return b.indexCount
}

// IndexElementSize is IndexBuffer::get_IndexElementSize:
//
//	_indexSize == 2 ? SixteenBits : ThirtyTwoBits
//
// a comparison over the stored width, not a stored enum.
func (b *IndexBuffer) IndexElementSize() IndexElementSize {
	if b == nil {
		return IndexElementSizeSixteenBits
	}
	return b.indexElementSize
}

// BufferUsage is IndexBuffer::get_BufferUsage, which converts the stored D3D
// usage bits back to XNA's enum. CNA reports the usage it applied and the
// projection records that.
func (b *IndexBuffer) BufferUsage() BufferUsage {
	if b == nil {
		return BufferUsageNone
	}
	return b.bufferUsage
}

// clrTypeName is System.Object::ToString's answer for an IndexBuffer.
func (b *IndexBuffer) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.IndexBuffer"
}

// checkDisposed is Helpers::CheckDisposed(object, native int), the same
// identity site VertexBuffer.checkDisposed is and for the same reason: the
// `ldarg.0` the reference pushes as an OBJECT decides the exception's type
// name, so a disposed DynamicIndexBuffer names itself.
func (b *IndexBuffer) checkDisposed() error {
	if !b.IsDisposed() {
		return nil
	}
	return fmt.Errorf("%w: %s", interop.ErrDisposed, b.resource.self().clrTypeName())
}

// noteContentRestored is CopyData's tail on the SETTING path, the same
// identity site VertexBuffer.noteContentRestored is.
func (b *IndexBuffer) noteContentRestored() {
	if dynamic, isDynamic := b.resource.self().(dynamicGraphicsResource); isDynamic {
		dynamic.setContentLost(false)
	}
}

// bindDerived forwards the CLR `this` to the composed base.
func (b *IndexBuffer) bindDerived(derived graphicsResourceObject) {
	if b == nil || b.resource == nil {
		return
	}
	b.resource.bindDerived(derived)
}

// nativeResource is the one owned CNA handle, for this package's own
// operations. Unexported; it never escapes.
func (b *IndexBuffer) nativeResource() *interop.Resource {
	if b == nil || b.resource == nil {
		return nil
	}
	return b.resource.nativeResource()
}

// The nine inherited GraphicsResource members.

// GraphicsDevice is GraphicsResource::get_GraphicsDevice, the device the buffer
// was created on.
func (b *IndexBuffer) GraphicsDevice() *GraphicsDevice {
	if b == nil {
		return nil
	}
	return b.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (b *IndexBuffer) Name() string {
	if b == nil {
		return ""
	}
	return b.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (b *IndexBuffer) SetName(value string) {
	if b == nil {
		return
	}
	b.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (b *IndexBuffer) Tag() any {
	if b == nil {
		return nil
	}
	return b.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (b *IndexBuffer) SetTag(value any) {
	if b == nil {
		return
	}
	b.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (b *IndexBuffer) IsDisposed() bool {
	if b == nil {
		return true
	}
	return b.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (b *IndexBuffer) ToString() string {
	if b == nil {
		return ""
	}
	return b.resource.ToString()
}

// AddDisposingHandler is add_Disposing.
func (b *IndexBuffer) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errIndexBufferNil
	}
	return b.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (b *IndexBuffer) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errIndexBufferNil
	}
	return b.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited. `callvirt` reaches
// THIS type's Dispose(bool) and releases the CNA buffer.
func (b *IndexBuffer) DisposeByNone() error {
	return b.DisposeByBoolean(true)
}

// DisposeByBoolean is IndexBuffer::Dispose(bool):
//
//	if (disposing) { try { ~IndexBuffer(); } finally { base.Dispose(true);  } }
//	else           { try { !IndexBuffer(); } finally { base.Dispose(false); } }
//
//	!IndexBuffer()  { if (!isDisposed) ReleaseNativeObject(true); }
//	~IndexBuffer()  { !IndexBuffer(); }
//
// The same shape Texture2D's has, including the asymmetry worth naming: the
// finalizer path STILL releases the native buffer, because `!IndexBuffer()`
// passes a hardcoded true to ReleaseNativeObject regardless of which branch
// reached it. The base call is in a `finally`, so the flag and the Disposing
// event are not skipped by a native failure, and the release's error is the one
// returned.
func (b *IndexBuffer) DisposeByBoolean(disposing bool) error {
	if b == nil {
		return errIndexBufferNil
	}
	var released error
	if !b.resource.IsDisposed() {
		released = b.resource.releaseNativeObject()
	}
	baseErr := b.resource.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
