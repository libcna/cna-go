package graphics

// dynamicGraphicsResource is the Go projection of
// Microsoft.Xna.Framework.Graphics.IDynamicGraphicsResource, which is
// `assembly` and therefore NOT in the pinned public contract:
//
//	.class interface abstract auto ansi IDynamicGraphicsResource
//	  .method public hidebysig newslot abstract virtual
//	          instance bool get_IsContentLost()
//	  .method public hidebysig newslot abstract virtual
//	          instance void SetContentLost(bool)
//
// It is projected unexported for exactly that reason: the interface is real
// machinery the reference DISPATCHES ON, so leaving it out would lose a
// behaviour, and exporting it would add a type the contract does not declare.
//
// Only SetContentLost is here. IsContentLost is a PUBLIC member of each
// implementing type and is projected there; nothing internal dispatches on it.
type dynamicGraphicsResource interface {
	// setContentLost is SetContentLost(bool): it stores the flag and, when the
	// flag is TRUE, raises ContentLost with EventArgs.Empty.
	//
	//	ldarg.0; ldarg.1; stfld _contentLost
	//	ldarg.1; brfalse ret
	//	ldarg.0; dup; ldsfld EventArgs::Empty; callvirt raise_ContentLost
	//
	// Infallible: the reference's body is a store and a delegate invocation,
	// and neither reaches a runtime.
	setContentLost(isContentLost bool)
}

// A successful upload CLEARS the content-lost latch, which is the measured fact
// this interface exists for.
//
// Every CopyData in the family ends the SETTING path with the same four
// instructions -- VertexBuffer's, IndexBuffer's, Texture2D's and TextureCube's,
// measured one by one:
//
//	IL_02a4:  ldarg.s   isSetting
//	IL_02a6:  brfalse.s IL_02ce
//	IL_02a8:  ldarg.0
//	IL_02a9:  isinst    IDynamicGraphicsResource
//	          ...
//	IL_02b7:  callvirt  IDynamicGraphicsResource::SetContentLost(bool)   // false
//
// So writing new data to a dynamic resource declares its content no longer
// lost, and a consumer that reacts to IsContentLost by re-uploading sees the
// flag go false because of the upload rather than because the device recovered.
// The GETTING path does not, and neither does a FAILED set: the clear is after
// the result check, so a refused upload leaves the latch exactly as it was.
//
// The `isinst` is on `ldarg.0`, the OBJECT, so it is an identity site: a
// composed base must resolve the whole object before asking whether it is
// dynamic, or the answer is always no.

// The compiler is the proof that every type the reference declares
// IDynamicGraphicsResource on carries the member. RenderTarget2D and
// RenderTargetCube are the texture half of the family and are listed with the
// buffers because the interface is one interface.
var (
	_ dynamicGraphicsResource = (*DynamicVertexBuffer)(nil)
	_ dynamicGraphicsResource = (*DynamicIndexBuffer)(nil)
	_ dynamicGraphicsResource = (*RenderTarget2D)(nil)
	_ dynamicGraphicsResource = (*RenderTargetCube)(nil)
)
