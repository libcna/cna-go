package graphics

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 66 — VertexBuffer.
// ---------------------------------------------------------------------------

// VertexBuffer is Microsoft.Xna.Framework.Graphics.VertexBuffer:
//
//	.class public auto ansi beforefieldinit VertexBuffer
//	       extends Microsoft.Xna.Framework.Graphics.GraphicsResource
//	       implements Microsoft.Xna.Framework.Graphics.IGraphicsResource
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_vertex_buffer_destroy.
//
// The DECLARATION it was created from is not owned by the buffer: the reference
// stores the same object the caller passed and `get_VertexDeclaration` hands it
// back, so disposing a buffer does not dispose its declaration. CNA agrees --
// its create-info copies what it needs and the declaration handle stays the
// caller's.
//
// # Its three properties are field reads
//
//	get_VertexCount         ldarg.0; ldfld _vertexCount; ret
//	get_VertexDeclaration   ldarg.0; ldfld _vertexDeclaration; ret
//	get_BufferUsage         ConvertDxBufferUsageToXna(_usage)
//
// None reaches D3D and none checks disposal, so all three answer after Dispose.
type VertexBuffer struct {
	// resource is the composed GraphicsResource, which owns the CNA handle.
	resource *GraphicsResource
	// declaration is `_vertexDeclaration`, the caller's object handed back
	// unchanged. The buffer does not own it.
	declaration *VertexDeclaration
	vertexCount int32
	bufferUsage BufferUsage
	// vertexStride is the reference's `_size / _vertexCount`: the declaration's
	// stride, which is what the buffer's byte size is measured in. It is CNA's
	// reported stride rather than the declaration's own, for the reason the
	// index buffer records CNA's width: a renderer that padded a vertex would
	// otherwise be invisible.
	vertexStride int32
}

// errVertexBufferNil is the Go-only guard for a zero VertexBuffer.
var errVertexBufferNil = errors.New("VertexBuffer is nil or uninitialized")

// errVertexBufferDeviceRequired is the same measured Go-only guard IndexBuffer
// carries: VertexBuffer's constructors do not null-check their device either.
var errVertexBufferDeviceRequired = errors.New("the graphics device must not be nil")

// The three FrameworkResources strings VertexBuffer and FromType add.
const (
	vertexStrideTooSmall      = "The vertex stride is too small for the type of data requested. This is not allowed."
	vertexTypeNotValueType    = "Invalid vertex type. {0} is not a value type."
	vertexTypeNotIVertexType  = "Invalid vertex type. {0} does not implement the IVertexType interface."
	vertexTypeNullDeclaration = "Invalid vertex type. {0} returned a null VertexDeclaration."
	vertexTypeWrongSize       = "Invalid vertex type. The size of {0} does not match the stride of its vertex declaration."
)

// NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage is
// VertexBuffer::.ctor(GraphicsDevice, VertexDeclaration, int32, BufferUsage):
//
//	if (vertexDeclaration == null)
//	    throw new ArgumentNullException("vertexDeclaration", FrameworkResources.NullNotAllowed);
//	if (vertexCount <= 0)
//	    throw new ArgumentOutOfRangeException("vertexCount",
//	        FrameworkResources.ResourcesMustBeGreaterThanZeroSize);
//	_parent = graphicsDevice;
//	CreateBuffer(vertexDeclaration, vertexCount, ConvertXnaBufferUsageToDx(usage), MANAGED);
//
// The declaration check runs FIRST, before the count and before the device is
// stored -- so a null declaration is refused whatever the other two are. And,
// as with IndexBuffer, there is NO device null check: the reference
// dereferences it later, so a nil device is refused here in Go's own words.
func NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, vertexDeclaration *VertexDeclaration, vertexCount int32, usage BufferUsage,
) (*VertexBuffer, error) {
	if vertexDeclaration == nil {
		return nil, fmt.Errorf("%w: vertexDeclaration: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if vertexCount <= 0 {
		return nil, fmt.Errorf("%w: vertexCount: %s", errArgumentOutOfRange, resourcesMustBeGreaterThanZeroSize)
	}
	if graphicsDevice == nil {
		return nil, fmt.Errorf("%w: graphicsDevice", errVertexBufferDeviceRequired)
	}
	device, err := graphicsDevice.live()
	if err != nil {
		return nil, err
	}
	// The declaration's CNA handle is created HERE, on first use. See
	// VertexDeclaration.nativeDeclaration for why it is deferred rather than
	// built in that type's constructor.
	native, err := vertexDeclaration.nativeDeclaration(device.Runtime())
	if err != nil {
		return nil, err
	}
	resource, err := device.CreateVertexBuffer(native, vertexCount, nativeBufferUsage(usage), false)
	if err != nil {
		return nil, err
	}
	info, err := resource.VertexBufferInfo()
	if err != nil {
		_ = resource.Dispose()
		return nil, err
	}
	return newVertexBuffer(graphicsDevice, resource, vertexDeclaration, info), nil
}

// NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage is
// VertexBuffer::.ctor(GraphicsDevice, Type, int32, BufferUsage), whose first
// statement is `VertexDeclaration.FromType(vertexType)` -- and whose vertexCount
// check therefore runs SECOND, after the type is resolved. A caller passing
// both a bad type and a bad count is told about the type.
func NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
	graphicsDevice *GraphicsDevice, vertexType reflect.Type, vertexCount int32, usage BufferUsage,
) (*VertexBuffer, error) {
	declaration, err := vertexDeclarationFromType(vertexType)
	if err != nil {
		return nil, err
	}
	return NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		graphicsDevice, declaration, vertexCount, usage)
}

// vertexDeclarationFromType is VertexDeclaration::FromType(Type), which is
// `assembly` -- not public surface here or there -- and is reproduced check for
// check because it is the whole failure surface of the Type-keyed constructor:
//
//	if (!vertexType.IsValueType)
//	    throw new ArgumentException(Format(VertexTypeNotValueType, name));
//	IVertexType instance = Activator.CreateInstance(vertexType) as IVertexType;
//	if (instance == null)
//	    throw new ArgumentException(Format(VertexTypeNotIVertexType, name));
//	VertexDeclaration declaration = instance.VertexDeclaration;
//	if (declaration == null)
//	    throw new InvalidOperationException(Format(VertexTypeNullDeclaration, name));
//	if (Marshal.SizeOf(vertexType) != declaration.VertexStride)
//	    throw new InvalidOperationException(Format(VertexTypeWrongSize, name));
//	return declaration;
//
// Every step is expressible in Go. `IsValueType` is a struct kind,
// `Activator.CreateInstance` is `reflect.New(t).Elem()`, and the interface test
// is a Go type assertion -- which is why Foundation 66 projects IVertexType at
// all: without it this member would be one nothing could satisfy.
//
// `Marshal.SizeOf` is the UNMANAGED size and `unsafe.Sizeof` is Go's own
// layout. They agree for the vertex structs a consumer writes here -- packed
// float and byte fields -- and where they could disagree, Go's is the one that
// decides how many bytes actually move, so it is the one checked.
func vertexDeclarationFromType(vertexType reflect.Type) (*VertexDeclaration, error) {
	if vertexType == nil {
		return nil, fmt.Errorf("%w: vertexType: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if vertexType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: %s", errArgument,
			formatVertexElementMessage(vertexTypeNotValueType, vertexType.String()))
	}
	instance, ok := reflect.New(vertexType).Elem().Interface().(IVertexType)
	if !ok {
		// The reference constructs the value and casts it. A Go value type
		// whose method has a POINTER receiver is not in the value's method set,
		// so it fails here exactly as a C# struct that does not implement the
		// interface does -- and the message names the type either way.
		return nil, fmt.Errorf("%w: %s", errArgument,
			formatVertexElementMessage(vertexTypeNotIVertexType, vertexType.String()))
	}
	declaration := instance.VertexDeclaration()
	if declaration == nil {
		return nil, fmt.Errorf("%w: %s", errVertexTypeInvalidOperation,
			formatVertexElementMessage(vertexTypeNullDeclaration, vertexType.String()))
	}
	if int32(vertexType.Size()) != declaration.VertexStride() {
		return nil, fmt.Errorf("%w: %s", errVertexTypeInvalidOperation,
			formatVertexElementMessage(vertexTypeWrongSize, vertexType.String()))
	}
	return declaration, nil
}

// errVertexTypeInvalidOperation projects the System.InvalidOperationException
// FromType's last two checks throw. The first two throw ArgumentException, and
// the split is the reference's own.
var errVertexTypeInvalidOperation = errors.New("operation is not valid")

// newVertexBuffer composes the base, installs the CLR `this` and records what
// CNA applied.
func newVertexBuffer(device *GraphicsDevice, resource *interop.Resource, declaration *VertexDeclaration, info interop.VertexBufferInfo) *VertexBuffer {
	usage := BufferUsageNone
	if info.BufferUsage == 1 {
		usage = BufferUsageWriteOnly
	}
	buffer := &VertexBuffer{
		resource:     newGraphicsResource(device, resource),
		declaration:  declaration,
		vertexCount:  info.VertexCount,
		bufferUsage:  usage,
		vertexStride: info.VertexStride,
	}
	buffer.resource.bindDerived(buffer)
	return buffer
}

// VertexCount is VertexBuffer::get_VertexCount, one field read.
func (b *VertexBuffer) VertexCount() int32 {
	if b == nil {
		return 0
	}
	return b.vertexCount
}

// VertexDeclaration is VertexBuffer::get_VertexDeclaration, one field read that
// hands back THE CALLER'S OBJECT -- the same pointer they passed, not a copy.
func (b *VertexBuffer) VertexDeclaration() *VertexDeclaration {
	if b == nil {
		return nil
	}
	return b.declaration
}

// BufferUsage is VertexBuffer::get_BufferUsage.
func (b *VertexBuffer) BufferUsage() BufferUsage {
	if b == nil {
		return BufferUsageNone
	}
	return b.bufferUsage
}

// clrTypeName is System.Object::ToString's answer for a VertexBuffer.
func (b *VertexBuffer) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.VertexBuffer"
}

// bindDerived forwards the CLR `this` to the composed base.
func (b *VertexBuffer) bindDerived(derived graphicsResourceObject) {
	if b == nil || b.resource == nil {
		return
	}
	b.resource.bindDerived(derived)
}

// nativeResource is the one owned CNA handle. Unexported; it never escapes.
func (b *VertexBuffer) nativeResource() *interop.Resource {
	if b == nil || b.resource == nil {
		return nil
	}
	return b.resource.nativeResource()
}

// bufferByteSize is the reference's `_size`: the declaration's stride times the
// vertex count, which is what every transfer's fit check is measured against.
func (b *VertexBuffer) bufferByteSize() int64 {
	if b == nil {
		return 0
	}
	return int64(b.vertexStride) * int64(b.vertexCount)
}

// The nine inherited GraphicsResource members.

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (b *VertexBuffer) GraphicsDevice() *GraphicsDevice {
	if b == nil {
		return nil
	}
	return b.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (b *VertexBuffer) Name() string {
	if b == nil {
		return ""
	}
	return b.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (b *VertexBuffer) SetName(value string) {
	if b == nil {
		return
	}
	b.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (b *VertexBuffer) Tag() any {
	if b == nil {
		return nil
	}
	return b.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (b *VertexBuffer) SetTag(value any) {
	if b == nil {
		return
	}
	b.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (b *VertexBuffer) IsDisposed() bool {
	if b == nil {
		return true
	}
	return b.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (b *VertexBuffer) ToString() string {
	if b == nil {
		return ""
	}
	return b.resource.ToString()
}

// AddDisposingHandler is add_Disposing.
func (b *VertexBuffer) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errVertexBufferNil
	}
	return b.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (b *VertexBuffer) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errVertexBufferNil
	}
	return b.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (b *VertexBuffer) DisposeByNone() error {
	return b.DisposeByBoolean(true)
}

// DisposeByBoolean is VertexBuffer::Dispose(bool), the same shape Texture2D's
// and IndexBuffer's have. It does NOT dispose the declaration: the reference
// stores the caller's object and disposes nothing it did not create.
func (b *VertexBuffer) DisposeByBoolean(disposing bool) error {
	if b == nil {
		return errVertexBufferNil
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

// vertexElementSize is `sizeof(T)` -- the size CopyData measures every transfer
// in. Go's own layout is what decides how many bytes actually move, so it is
// what is used, and the one place a CLR `Marshal.SizeOf` could disagree is
// recorded in vertexDeclarationFromType.
func vertexElementSize[T any]() int32 {
	var zero T
	return int32(unsafe.Sizeof(zero))
}
