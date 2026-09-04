package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// VertexBufferReference and IndexBufferReference are the Go projections of a
// position whose CLR type is VertexBuffer or IndexBuffer.
//
// # Why they appeared at Foundation 84 and not earlier
//
// Both bases had positions from the day they were projected --
// GraphicsDevice::SetVertexBuffer's `vertexBuffer`, VertexBufferBinding's
// `vertexBuffer`, GraphicsDevice::Indices' setter -- and their substitutability
// requirement was measured LATENT the whole time, because a position only
// matters when something can be handed to it that is not the base itself. The
// dynamic buffers are the first and only projected types that derive from
// these two, so the requirement went LIVE the way Texture2D's did when
// RenderTarget2D arrived and TextureCube's did when EnvironmentMapEffect gave
// it a position.
//
// A game that streams geometry writes exactly this:
//
//	GraphicsDevice.SetVertexBuffer(dynamicVertexBuffer);
//	GraphicsDevice.Indices = dynamicIndexBuffer;
//
// so a `*VertexBuffer` parameter would be a position the very type the dynamic
// buffers exist for cannot reach.
//
// # No return widens
//
// GraphicsDevice::get_Indices hands back an IndexBuffer and
// GetVertexBuffers hands back VertexBufferBindings holding one, and the
// reference's own consumer reads buffer members off them. Neither is a member
// whose derived identity is the point the way Effect::Clone's is, so the
// settled default holds: parameters widen, returns stay concrete. What is lost
// is the C# downcast on a getter, which is recorded here rather than papered
// over -- a consumer who put a DynamicVertexBuffer in gets the composed
// VertexBuffer back out and cannot call SetData with SetDataOptions on it. They
// hold the dynamic value they created, which is the object the reference's own
// consumer keeps too.
type VertexBufferReference interface {
	// The VertexBuffer half of whatever the value is. Unexported, so the
	// interface is unsatisfiable outside this module, and suffixed `Base`
	// because both implementing types already hold a field named `buffer`.
	vertexBufferBase() *VertexBuffer

	// VertexBuffer's own declared surface.
	VertexDeclaration() *VertexDeclaration
	VertexCount() int32
	BufferUsage() BufferUsage

	// GraphicsResource's, inherited by VertexBuffer and re-exposed by
	// DynamicVertexBuffer.
	GraphicsDevice() *GraphicsDevice
	Name() string
	SetName(value string)
	Tag() any
	SetTag(value any)
	IsDisposed() bool
	ToString() string
	AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDisposingHandler(subscription framework.EventSubscription) error

	// Disposal is absent for the reason it is absent from EffectReference, and
	// the split is the same one: VertexBuffer DECLARES the protected
	// `Dispose(Boolean)` override, so it projects DisposeByNone AND
	// DisposeByBoolean; DynamicVertexBuffer declares no Dispose at all, so its
	// inherited PUBLIC surface carries one no-argument Dispose. An interface
	// member would have to require a member the pinned contract does not give
	// one of the two types.
}

// IndexBufferReference is VertexBufferReference's counterpart, on the same
// measurement: GraphicsDevice::set_Indices is its live position and
// DynamicIndexBuffer is its one derived type.
type IndexBufferReference interface {
	indexBufferBase() *IndexBuffer

	// IndexBuffer's own declared surface.
	IndexCount() int32
	IndexElementSize() IndexElementSize
	BufferUsage() BufferUsage

	// GraphicsResource's.
	GraphicsDevice() *GraphicsDevice
	Name() string
	SetName(value string)
	Tag() any
	SetTag(value any)
	IsDisposed() bool
	ToString() string
	AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDisposingHandler(subscription framework.EventSubscription) error
}

// vertexBufferBase makes a VertexBuffer its own reference.
func (b *VertexBuffer) vertexBufferBase() *VertexBuffer { return b }

// indexBufferBase makes an IndexBuffer its own reference.
func (b *IndexBuffer) indexBufferBase() *IndexBuffer { return b }

// resolveVertexBuffer is the `ldarg` a CLR call site does for free. It answers
// nil for a nil interface AND for an interface holding a typed nil, because the
// reference sees one null either way.
func resolveVertexBuffer(reference VertexBufferReference) *VertexBuffer {
	if reference == nil {
		return nil
	}
	return reference.vertexBufferBase()
}

// resolveIndexBuffer is its counterpart.
func resolveIndexBuffer(reference IndexBufferReference) *IndexBuffer {
	if reference == nil {
		return nil
	}
	return reference.indexBufferBase()
}

// The compiler is the proof that both halves of each family satisfy their
// interface. A dynamic buffer that forgot one forwarding member fails here.
var (
	_ VertexBufferReference = (*VertexBuffer)(nil)
	_ VertexBufferReference = (*DynamicVertexBuffer)(nil)
	_ IndexBufferReference  = (*IndexBuffer)(nil)
	_ IndexBufferReference  = (*DynamicIndexBuffer)(nil)
)
