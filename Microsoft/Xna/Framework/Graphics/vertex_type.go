package graphics

// ---------------------------------------------------------------------------
// Foundation 66 — IVertexType.
// ---------------------------------------------------------------------------

// IVertexType is Microsoft.Xna.Framework.Graphics.IVertexType:
//
//	.class interface public abstract auto ansi IVertexType
//	  .method public hidebysig newslot specialname abstract virtual
//	          instance VertexDeclaration get_VertexDeclaration()
//
// One read-only property, and it is the whole interface. A consumer's vertex
// struct implements it so that `new VertexBuffer(device, typeof(MyVertex), ...)`
// can find the layout, which is exactly what it is for in C#.
//
// # It is what makes the Type-keyed constructor reachable
//
// VertexDeclaration::FromType is `assembly` -- not public surface here or there
// -- and its second check is `Activator.CreateInstance(type) as IVertexType`.
// Without a projected IVertexType, every Go type would fail that check and the
// Type-keyed VertexBuffer constructor would be a member nothing could satisfy.
// With it, a consumer's own struct satisfies it exactly as a C# one does, and
// the reflection FromType needs -- construct a zero value, ask it for its
// declaration -- is expressible in Go.
type IVertexType interface {
	// VertexDeclaration is get_VertexDeclaration.
	VertexDeclaration() *VertexDeclaration
}
