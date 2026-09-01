package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 64 — VertexDeclaration, the first vertex type.
// ---------------------------------------------------------------------------

// VertexDeclaration is Microsoft.Xna.Framework.Graphics.VertexDeclaration:
//
//	.class public auto ansi beforefieldinit VertexDeclaration
//	       extends Microsoft.Xna.Framework.Graphics.GraphicsResource
//
// # Its whole public surface is two members and the base's nine
//
//	.ctor(int32 vertexStride, VertexElement[] elements)
//	.ctor(VertexElement[] elements)
//	get_VertexStride     ldarg.0; ldfld _vertexStride; ret
//	GetVertexElements    ldarg.0; ldfld _elements; Array.Clone(); castclass; ret
//	Dispose(bool)        the override below
//
// `Bind`, `Unbind`, `FromType` and `_binding` are `assembly`, so none is public
// surface here or there.
//
// # Its PUBLIC surface reaches nothing native, and that is the reference's shape
//
// The constructor clones the element array, stores a stride and calls
// VertexElementValidator. It never touches a device: `GraphicsResource::_parent`
// is assigned by `Bind`, which is internal and which only the draw path calls.
// So a constructed declaration has a NULL GraphicsDevice, and answers so.
//
// # Ownership
//
//	OWNED, created LAZILY, destroyed with cna_vertex_declaration_destroy.
//
// Foundation 64 projected this type with no native half at all, because nothing
// consumed one: every public member answers from the fields the reference
// itself reads, so binding CNA's declaration routes would have added routes
// with no real consumer. Foundation 66 is the milestone that gives them one --
// `cna_vertex_buffer_create` takes a `CNA_VertexDeclarationHandle` -- so the
// handle is created at the FIRST operation that needs it, which today is
// exactly that call.
//
// Deferring it rather than creating it in the constructor is not a convenience:
// `cna_vertex_declaration_create` needs no device and would work eagerly, but a
// declaration a consumer builds and never binds would then own a CNA object for
// nothing, and its Dispose would have to destroy one the reference does not
// have. The lazy handle keeps the managed lifetime the reference has and adds a
// native one only where a native consumer exists.
type VertexDeclaration struct {
	// resource is the composed GraphicsResource. Private named composition, no
	// embedding, no accessor -- the settled rule.
	resource *GraphicsResource
	// elements is `_elements`, the CLONE the constructor made. The reference
	// clones on the way in and again on the way out, so a caller can mutate
	// neither the array it passed nor the array it received.
	elements []VertexElement
	// vertexStride is `_vertexStride`, either the caller's or the one
	// VertexElementValidator::GetVertexStride computed.
	vertexStride int32
	// native is the CNA declaration handle, created lazily. It is nil until
	// something needs one, which today is only vertex-buffer creation.
	native *interop.Resource
}

// nativeDeclaration creates the CNA declaration on first use and answers the
// same handle afterwards.
//
// It passes the stride EXPLICITLY, always. CNA's stride-less route recomputes
// one from the elements, and that would silently replace an explicit stride the
// caller gave the two-argument constructor -- a declaration built as
// `new VertexDeclaration(32, elements)` over a 16-byte layout would become a
// 16-byte one. The reference stores what it was given; so does this.
func (d *VertexDeclaration) nativeDeclaration(runtime *interop.Runtime) (*interop.Resource, error) {
	if d == nil {
		return nil, errVertexDeclarationNil
	}
	if d.native != nil {
		return d.native, nil
	}
	if runtime == nil {
		return nil, errVertexDeclarationNil
	}
	flattened := make([]int32, 0, len(d.elements)*4)
	for index := range d.elements {
		flattened = append(flattened,
			d.elements[index].Offset(),
			int32(d.elements[index].VertexElementFormat()),
			int32(d.elements[index].VertexElementUsage()),
			d.elements[index].UsageIndex())
	}
	created, err := runtime.CreateVertexDeclaration(d.vertexStride, true, flattened)
	if err != nil {
		return nil, err
	}
	d.native = created
	return created, nil
}

// errVertexDeclarationNil is the Go-only guard. Go can produce a zero
// VertexDeclaration whose constructor never ran, and such an object has no
// composed base to forward to.
var errVertexDeclarationNil = fmt.Errorf("VertexDeclaration is nil or uninitialized")

// NewVertexDeclarationByInt32AndSliceOfVertexElement is
// VertexDeclaration::.ctor(int32, VertexElement[]):
//
//	if (elements == null || elements.Length == 0)
//	    throw new ArgumentNullException("elements", FrameworkResources.NullNotAllowed);
//	_elements = (VertexElement[])elements.Clone();
//	_vertexStride = vertexStride;
//	VertexElementValidator.Validate(vertexStride, _elements);
//
// The empty-array case is the one worth naming: an EMPTY array takes the same
// branch as a null one, so `new VertexDeclaration(16)` throws
// ArgumentNullException rather than producing a declaration with no elements.
// Go has no null slice distinct from an empty one, and it does not need one:
// both are refused, which is what the reference does with both.
//
// The `.param [2] ParamArrayAttribute` makes the elements a C# `params` array,
// and the projection is a SLICE rather than a Go variadic. `params` is a
// call-site convenience over a parameter whose CLR type is an array; the
// settled array rule projects that to a slice, and a variadic would spell the
// same member differently from every other array position in the profile.
func NewVertexDeclarationByInt32AndSliceOfVertexElement(vertexStride int32, elements []VertexElement) (*VertexDeclaration, error) {
	if len(elements) == 0 {
		return nil, fmt.Errorf("%w: elements: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned := append([]VertexElement(nil), elements...)
	if err := validateVertexElements(vertexStride, cloned); err != nil {
		return nil, err
	}
	return newVertexDeclaration(cloned, vertexStride), nil
}

// NewVertexDeclarationBySliceOfVertexElement is
// VertexDeclaration::.ctor(VertexElement[]), which differs in one statement:
//
//	int stride = VertexElementValidator.GetVertexStride(_elements);
//	_vertexStride = stride;
//	VertexElementValidator.Validate(stride, _elements);
//
// The computed stride is the largest END offset among the elements, so a
// declaration whose elements leave a gap strides over it. It is then validated
// like any other, which is why this constructor can still fail: a computed
// stride that is not a multiple of four is rejected by the same check the
// explicit one is.
func NewVertexDeclarationBySliceOfVertexElement(elements []VertexElement) (*VertexDeclaration, error) {
	if len(elements) == 0 {
		return nil, fmt.Errorf("%w: elements: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned := append([]VertexElement(nil), elements...)
	stride := vertexStrideForElements(cloned)
	if err := validateVertexElements(stride, cloned); err != nil {
		return nil, err
	}
	return newVertexDeclaration(cloned, stride), nil
}

// newVertexDeclaration composes the base and installs the CLR `this`. The
// device and the native resource are both nil, which is the state the
// reference's constructor leaves: it calls no GraphicsResource constructor that
// would assign `_parent`, and assigns no `_internalHandle`.
func newVertexDeclaration(elements []VertexElement, vertexStride int32) *VertexDeclaration {
	declaration := &VertexDeclaration{
		resource:     newGraphicsResource(nil, nil),
		elements:     elements,
		vertexStride: vertexStride,
	}
	declaration.resource.bindDerived(declaration)
	return declaration
}

// VertexStride is VertexDeclaration::get_VertexStride, one `ldfld`. It answers
// after disposal, because the reference's getter has no disposal check.
func (d *VertexDeclaration) VertexStride() int32 {
	if d == nil {
		return 0
	}
	return d.vertexStride
}

// GetVertexElements is VertexDeclaration::GetVertexElements:
//
//	ldarg.0; ldfld _elements; callvirt Array.Clone(); castclass; ret
//
// The CLONE is the member. A projection returning the stored slice would let a
// caller rewrite a validated declaration in place -- and the reference clones
// precisely so they cannot, which is why this is not a field read like
// VertexStride is.
func (d *VertexDeclaration) GetVertexElements() []VertexElement {
	if d == nil {
		return nil
	}
	return append([]VertexElement(nil), d.elements...)
}

// clrTypeName is System.Object::ToString's answer for a VertexDeclaration,
// which is what GraphicsResource::ToString falls back to for an unnamed one.
func (d *VertexDeclaration) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.VertexDeclaration"
}

// bindDerived forwards the CLR `this` to the composed base. Nothing derives
// from VertexDeclaration in the pinned contract, so this exists for the same
// reason Texture's does: the chain has one shape.
func (d *VertexDeclaration) bindDerived(derived graphicsResourceObject) {
	if d == nil || d.resource == nil {
		return
	}
	d.resource.bindDerived(derived)
}

// nativeResource is nil for a declaration: it owns no CNA handle. It exists so
// the type satisfies the same unexported shape the rest of the family has.
func (d *VertexDeclaration) nativeResource() *interop.Resource {
	if d == nil || d.resource == nil {
		return nil
	}
	return d.resource.nativeResource()
}

// The nine inherited GraphicsResource members, written out rather than
// promoted. See texture2d_resource.go for why composition forwards explicitly.

// GraphicsDevice is GraphicsResource::get_GraphicsDevice. It is NIL for a
// declaration a consumer constructed: `_parent` is assigned by the internal
// Bind, which no public member reaches. That is the reference's answer too, and
// it is reported rather than filled in with a device this type never saw.
func (d *VertexDeclaration) GraphicsDevice() *GraphicsDevice {
	if d == nil {
		return nil
	}
	return d.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (d *VertexDeclaration) Name() string {
	if d == nil {
		return ""
	}
	return d.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (d *VertexDeclaration) SetName(value string) {
	if d == nil {
		return
	}
	d.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (d *VertexDeclaration) Tag() any {
	if d == nil {
		return nil
	}
	return d.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (d *VertexDeclaration) SetTag(value any) {
	if d == nil {
		return
	}
	d.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (d *VertexDeclaration) IsDisposed() bool {
	if d == nil {
		return true
	}
	return d.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString: the Name when it has one, and
// otherwise this object's runtime CLR type name.
func (d *VertexDeclaration) ToString() string {
	if d == nil {
		return ""
	}
	return d.resource.ToString()
}

// AddDisposingHandler is add_Disposing, on the base's registration list.
func (d *VertexDeclaration) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if d == nil {
		return framework.EventSubscription{}, errVertexDeclarationNil
	}
	return d.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (d *VertexDeclaration) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errVertexDeclarationNil
	}
	return d.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited. `callvirt` reaches
// THIS type's Dispose(bool).
func (d *VertexDeclaration) DisposeByNone() error {
	return d.DisposeByBoolean(true)
}

// DisposeByBoolean is VertexDeclaration::Dispose(bool):
//
//	if (disposing) { try { ~VertexDeclaration(); } finally { base.Dispose(true);  } }
//	else           { try { !VertexDeclaration(); } finally { base.Dispose(false); } }
//
//	!VertexDeclaration()  { Unbind(); }
//	~VertexDeclaration()  { Unbind(); }
//
// Both finalizer bodies are the SAME single call, which is unusual in this
// family -- Texture2D's two branches differ only in what they hand the base,
// and here even that is the whole difference.
//
// `Unbind` releases the DeclarationBinding that `Bind` created:
//
//	if (_binding != null) { _parent.vertexDeclarationManager.ReleaseBinding(_binding);
//	                        _binding = null; }
//
// `Bind` is `assembly` and no public member reaches it, so `_binding` is
// permanently null for every declaration a consumer can construct and Unbind is
// a no-op with its branch not taken. That is not a gap being skipped: it is the
// state the reference is in for the same object, and it is why this override
// adds no failure of its own.
func (d *VertexDeclaration) DisposeByBoolean(disposing bool) error {
	if d == nil {
		return errVertexDeclarationNil
	}
	// Unbind(), whose one branch is unreachable from public surface.
	//
	// The CNA handle is released here, on both branches, for the reason every
	// other graphics resource releases on both: `!VertexDeclaration()` is the
	// same body as `~VertexDeclaration()` in the reference, and a finalizer
	// path that kept a native object alive would leak it. A declaration that
	// never reached a vertex buffer has no handle and releases nothing.
	var released error
	if d.native != nil && !d.resource.IsDisposed() {
		native := d.native
		d.native = nil
		released = native.Dispose()
	}
	baseErr := d.resource.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
