package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 56 — GraphicsResource, the root of the graphics base chain.
// ---------------------------------------------------------------------------

// GraphicsResource is Microsoft.Xna.Framework.Graphics.GraphicsResource: the
// abstract root eleven graphics types derive from, and the type whose deferral
// blocked all eleven.
//
// # It has no public constructor, and that is read from the IL
//
//	.class public abstract auto ansi beforefieldinit GraphicsResource
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//	  .method assembly hidebysig specialname rtspecialname instance void .ctor()
//
// `assembly` is internal. A consumer cannot construct one in C# either, so
// CNA-Go declares no `NewGraphicsResource` and the pinned contract declares no
// constructor member for it. The type exists so its surface can be composed
// onto the types that DO have constructors.
//
// # Its state, field for field
//
//	.field private string  _localName
//	.field private object  _localTag
//	.field class GraphicsDevice _parent
//	.field assembly uint64 _internalHandle
//	.field assembly bool   isDisposed
//	.field private EventHandler`1<EventArgs> <backing_store>Disposing
//
// `_internalHandle` is the native object. In CNA-Go that is one
// `*interop.Resource`, and it lives HERE rather than on the derived type, for
// the reason the reference puts the handle here too: there is exactly one
// native owner per logical object, and a derived wrapper that held its own
// would be a second one. The derived types keep only what the reference keeps
// on them -- Texture's format and level count, Texture2D's width and height.
//
// # Ownership
//
//	OWNED, generation-checked, released exactly once.
//
// `interop.Resource` carries the kind tag, so the type-specific release the
// reference's `ReleaseNativeObject` override performs is already inside
// `Resource.Dispose`. That is why CNA-Go needs no per-derived-type release
// hook: the one CNA resource knows how to destroy itself.
//
// # Name and Tag are managed here, and CNA's own routes are deliberately unused
//
// The reference's accessors are conditional:
//
//	get_Name: if (_internalHandle != 0)
//	              return _parent.Resources.GetCachedName(_internalHandle);
//	          return _localName;
//
// `DeviceResourceManager::GetCachedName` is 79 bytes of
// `Dictionary<ulong, ResourceData>` under a `Monitor`, answering `String.Empty`
// for an absent key. It is managed storage reached indirectly, with no D3D call
// and no throw site, which is why both accessors are infallible.
//
// CNA DOES have a counterpart -- `cna_graphics_resource_set_name`,
// `copy_name`, `get_tag`, `set_tag`, `get_is_disposed` and six more -- and
// Foundation 56 said it did not. Foundation 57 measured them against the pinned
// artifact and left them unbound for reasons that are now recorded per route in
// tools/native_abi's `deliberatelyUnboundRoutes`. Two decide this member:
//
//	the routes REFUSE a SpriteBatch handle (CNA result 2), and SpriteBatch is a
//	GraphicsResource in the pinned contract, so binding them would make one XNA
//	member answer for textures and fail for sprite batches; and
//
//	cna_graphics_resource_set_name validates UTF-8 and refused an embedded NUL,
//	while set_Name validates nothing at all -- so binding it would refuse names
//	the reference accepts and make an infallible member fallible.
//
// Both are therefore MANAGED STORED members and neither is fallible, which is
// the settled per-operation fallibility rule applied to the reference's body.
type GraphicsResource struct {
	// resource is the one native owner. It is nil for a resource whose
	// construction is still in progress and for the abstract base itself.
	resource *interop.Resource
	// device is `_parent`, the borrowed device facade the resource was created
	// on. get_GraphicsDevice is one `ldfld` with no disposal check, so this is
	// returned as stored, including after disposal.
	device *GraphicsDevice
	// name and tag stand in for the reference's DeviceResourceManager cache.
	name string
	tag  any
	// isDisposed is the reference's own `assembly bool isDisposed`, which is
	// what makes GraphicsResource disposal idempotent -- unlike GameComponent's,
	// which has no such flag and re-runs.
	isDisposed bool
	// disposing is the `<backing_store>Disposing` delegate field.
	disposing framework.EventSource[*framework.EventArgs]
	// derived is the CLR `this`: the outermost object this resource is the base
	// of. See GameComponent.derived for why a composed base needs one; here it
	// carries TWO identity sites, the Disposing sender and ToString's subject.
	derived graphicsResourceObject
}

// graphicsResourceObject is the CLR `this` a composed GraphicsResource needs.
//
// Two of the base's members use `ldarg.0` as an OBJECT. One needs the object
// itself, as the Disposing sender; the other needs its TYPE, because
// GraphicsResource::ToString falls back to `System.Object::ToString()`, which
// answers with the runtime type's full CLR name. A Texture2D must therefore
// answer "Microsoft.Xna.Framework.Graphics.Texture2D" and not the base's name.
type graphicsResourceObject interface {
	// clrTypeName is System.Object::ToString's answer for this object.
	clrTypeName() string
}

// clrTypeName makes GraphicsResource its own `this` when nothing composes it.
func (r *GraphicsResource) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.GraphicsResource"
}

// bindDerived installs the CLR `this`. Every constructor of a type that
// composes a GraphicsResource calls it, and nothing else does.
func (r *GraphicsResource) bindDerived(derived graphicsResourceObject) { r.derived = derived }

// self is `ldarg.0` as an OBJECT: the outermost object of the composition
// chain, which is the resource itself when nothing composes it.
func (r *GraphicsResource) self() graphicsResourceObject {
	if r.derived != nil {
		return r.derived
	}
	return r
}

// newGraphicsResource is the reference's `assembly` constructor plus the two
// values the reference's derived constructors store immediately after it: the
// parent device and the native handle.
func newGraphicsResource(device *GraphicsDevice, resource *interop.Resource) *GraphicsResource {
	return &GraphicsResource{device: device, resource: resource}
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice:
//
//	ldarg.0; ldfld _parent; ret
//
// One field read, no disposal check, no native call. It returns whatever the
// constructor stored, including after the resource is disposed, because that is
// what the reference returns.
func (r *GraphicsResource) GraphicsDevice() *GraphicsDevice {
	if r == nil {
		return nil
	}
	return r.device
}

// Name is GraphicsResource::get_Name, over managed storage. See the type
// comment for why the reference's DeviceResourceManager cache is not reachable.
func (r *GraphicsResource) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}

// SetName is GraphicsResource::set_Name. It validates nothing: the reference
// stores whatever it is handed, including null, which projects to "".
func (r *GraphicsResource) SetName(value string) {
	if r == nil {
		return
	}
	r.name = value
}

// Tag is GraphicsResource::get_Tag, `System.Object` and therefore `any`.
func (r *GraphicsResource) Tag() any {
	if r == nil {
		return nil
	}
	return r.tag
}

// SetTag is GraphicsResource::set_Tag, which also validates nothing.
func (r *GraphicsResource) SetTag(value any) {
	if r == nil {
		return
	}
	r.tag = value
}

// IsDisposed is GraphicsResource::get_IsDisposed, one `ldfld` over the flag
// that makes this family's disposal idempotent.
func (r *GraphicsResource) IsDisposed() bool {
	if r == nil {
		return true
	}
	return r.isDisposed
}

// ToString is GraphicsResource::ToString, which is `public hidebysig virtual`
// and the only ToString in the chain -- neither Texture nor Texture2D declares
// one, so every resource in the family answers with this body:
//
//	string local = _internalHandle != 0
//	    ? _parent.Resources.GetCachedName(_internalHandle)
//	    : _localName;
//	if (!String.IsNullOrEmpty(local)) return local;
//	return base.ToString();
//
// `base.ToString()` is `System.Object::ToString`, which answers with the
// RUNTIME type's full name. That is the second identity site: a named Texture2D
// answers with its name, and an unnamed one must answer
// "Microsoft.Xna.Framework.Graphics.Texture2D", not the base's name.
//
// It is infallible for the reason every ToString in the profile is: the
// fallibility rule excludes it by name, and this body reaches nothing native.
func (r *GraphicsResource) ToString() string {
	if r == nil {
		return ""
	}
	if r.name != "" {
		return r.name
	}
	return r.self().clrTypeName()
}

// AddDisposingHandler is add_Disposing, `Delegate.Combine` under a
// `synchronized` method, on the settled two-accessor event projection.
func (r *GraphicsResource) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if r == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return r.disposing.Add(handler)
}

// RemoveDisposingHandler is remove_Disposing, `Delegate.Remove`.
func (r *GraphicsResource) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if r == nil {
		return errGraphicsResourceNil
	}
	return r.disposing.Remove(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), the sealed IDisposable member:
//
//	.method public hidebysig virtual FINAL instance void Dispose()
//	  Dispose(true);
//	  GC.SuppressFinalize(this);
//
// `callvirt Dispose(bool)` dispatches to the DERIVED override, so a Texture2D
// disposed through this member releases its native texture. Every derived type
// in the chain therefore declares its own DisposeByNone forwarding to its OWN
// DisposeByBoolean; forwarding to the composed base's would reproduce the
// base's slot and leak the native object.
//
// GC.SuppressFinalize has nothing to suppress: CNA-Go registers no Go finalizer
// on a graphics resource, so the call is a no-op rather than an unreproduced
// step. Deterministic disposal is the whole contract here.
func (r *GraphicsResource) DisposeByNone() error {
	return r.DisposeByBoolean(true)
}

// DisposeByBoolean is GraphicsResource::Dispose(bool), the protected virtual
// that is the base half of every graphics resource's disposal:
//
//	if (disposing) { ~GraphicsResource(); }
//	else { try { !GraphicsResource(); } finally { Object.Finalize(); } }
//
//	!GraphicsResource()  { isDisposed = true; }
//	~GraphicsResource()  { if (!isDisposed) { !GraphicsResource();
//	                                          Disposing(this, EventArgs.Empty); } }
//
// Three facts a reader should not have to reconstruct:
//
// The flag is set BEFORE the event is raised, so a Disposing handler observes
// IsDisposed == true -- and, because the derived override releases the native
// object before calling this, a handler also observes a resource whose native
// half is already gone.
//
// It is IDEMPOTENT. GameComponent's Dispose has no such flag and re-runs;
// this one has `isDisposed` and raises Disposing exactly once, ever.
//
// `Dispose(false)` sets the flag and raises NOTHING. That is the finalizer
// path, which Go never takes on its own, and it is projected because the
// contract declares the member.
func (r *GraphicsResource) DisposeByBoolean(disposing bool) error {
	if r == nil {
		return errGraphicsResourceNil
	}
	if !disposing {
		r.isDisposed = true
		return nil
	}
	if r.isDisposed {
		return nil
	}
	r.isDisposed = true
	return r.disposing.Raise(r.self(), framework.EventArgsEmpty())
}

// Finalize is GraphicsResource::Finalize, the protected finalizer:
//
//	Dispose(false);
//
// `callvirt`, so on a Texture2D it reaches the derived override, whose false
// branch still releases the native object. Nothing calls this on its own: Go
// has no CLR finalization and CNA-Go registers no runtime finalizer. It is
// projected because the pinned contract declares it.
func (r *GraphicsResource) Finalize() error {
	return r.DisposeByBoolean(false)
}

// releaseNativeObject is the composed base's half of the reference's
// `IGraphicsResource::ReleaseNativeObject` override chain.
//
// The reference's overrides are D3D9-specific -- Texture2D's releases its
// state-tracker wrapper, then its COM pointer, then the device's references to
// the handle. CNA's resource handles carry their own kind, so
// `interop.Resource.Dispose` performs the type-specific destruction and one
// unexported member covers the whole family. It is unexported because
// `IGraphicsResource` is `.class interface private` -- internal -- and is not
// public surface in the reference either.
func (r *GraphicsResource) releaseNativeObject() error {
	if r == nil || r.resource == nil {
		return nil
	}
	return r.resource.Dispose()
}

// nativeResource is the one owned CNA handle, for the derived types' own
// operations. It is unexported and never escapes the package: the raw-handle
// rule forbids a public accessor and the verifier enforces it.
func (r *GraphicsResource) nativeResource() *interop.Resource {
	if r == nil {
		return nil
	}
	return r.resource
}
