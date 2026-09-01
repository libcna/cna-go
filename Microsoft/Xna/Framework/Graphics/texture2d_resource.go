package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// The twelve members Texture2D inherits from its composed base chain, plus the
// disposal override it declares itself.
//
// Every inherited member is written out rather than promoted, which is the
// whole point of composition over embedding: a promoted member would keep the
// base's body wherever the derived one was not redeclared with exactly the
// right shape, and Texture2D overrides Dispose(bool).
//
// Nine come from GraphicsResource and two from Texture. The reference declares
// none of them on Texture2D, so none is invented surface: they are the public
// members a consumer holding a Texture2D really has in C#.

// clrTypeName is System.Object::ToString's answer for a Texture2D, which is the
// CLR `this` GraphicsResource::ToString falls back to. It is unexported because
// the reference has no such member: it is the Go spelling of a runtime type
// identity the CLR reads off the object header.
func (t *Texture2D) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.Texture2D" }

// bindDerived forwards the CLR `this` up the chain. Texture2D holds no copy of
// its own: there is one outermost object, and one place that answers with it.
func (t *Texture2D) bindDerived(derived graphicsResourceObject) {
	if t == nil || t.texture == nil {
		return
	}
	t.texture.bindDerived(derived)
}

// nativeResource is the one owned CNA handle, reached through the chain. It is
// unexported and never escapes the package.
func (t *Texture2D) nativeResource() *interop.Resource {
	if t == nil || t.texture == nil {
		return nil
	}
	return t.texture.nativeResource()
}

// Format is Texture::get_Format. The stored format is the one the constructor
// asked for, or -- on the FromStream path, where the reference passes no value
// -- the one the created surface actually has.
func (t *Texture2D) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.texture.Format()
}

// LevelCount is Texture::get_LevelCount, the mip level count of the created
// texture. It is CNA's reported level count, which is the same thing the
// reference reads out of IDirect3DBaseTexture9::GetLevelCount.
func (t *Texture2D) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.texture.LevelCount()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice, the device the
// texture was created on. One field read, answered after disposal too.
func (t *Texture2D) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.texture.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *Texture2D) Name() string {
	if t == nil {
		return ""
	}
	return t.texture.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *Texture2D) SetName(value string) {
	if t == nil {
		return
	}
	t.texture.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *Texture2D) Tag() any {
	if t == nil {
		return nil
	}
	return t.texture.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *Texture2D) SetTag(value any) {
	if t == nil {
		return
	}
	t.texture.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed, the flag that makes this
// family's disposal idempotent.
func (t *Texture2D) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.texture.IsDisposed()
}

// ToString is GraphicsResource::ToString: the resource's Name when it has one,
// and otherwise the runtime type's full CLR name -- which for this object is
// "Microsoft.Xna.Framework.Graphics.Texture2D", resolved through the CLR `this`
// the constructor installed.
func (t *Texture2D) ToString() string {
	if t == nil {
		return ""
	}
	return t.texture.ToString()
}

// AddDisposingHandler is add_Disposing, on the base's own registration list:
// the event is declared by GraphicsResource and raised from its Dispose(bool).
func (t *Texture2D) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.texture.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *Texture2D) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited:
//
//	Dispose(true);
//	GC.SuppressFinalize(this);
//
// `callvirt`, so it dispatches to THIS type's Dispose(bool) and releases the
// native texture. Forwarding to the composed base's DisposeByNone instead would
// reproduce the base's slot, set the flag, raise Disposing and leak the CNA
// texture -- which is the exact failure Go's lack of virtual dispatch invites.
func (t *Texture2D) DisposeByNone() error {
	return t.DisposeByBoolean(true)
}

// DisposeByBoolean is Texture2D::Dispose(bool), reproduced from its IL:
//
//	if (disposing) { try { ~Texture2D(); } finally { base.Dispose(true); } }
//	else           { try { !Texture2D(); } finally { base.Dispose(false); } }
//
//	!Texture2D()  { if (!isDisposed) { ReleaseNativeObject(true);
//	                                   CleanupSavedData(); } }
//	~Texture2D()  { !Texture2D(); }
//
// Three facts the shape encodes, and one asymmetry worth naming.
//
// The two branches differ ONLY in the flag handed to the base: `!Texture2D()`
// is the same body on both, and it passes a hardcoded `ldc.i4.1` to
// ReleaseNativeObject regardless. So `Dispose(false)` -- the finalizer path --
// still destroys the native texture. SpriteBatch's override guards its release
// on `disposing` and does not; the two ILs disagree and both are reproduced as
// written rather than smoothed to match.
//
// The release runs BEFORE the base sets isDisposed and raises Disposing, so a
// Disposing handler observes IsDisposed == true and a texture whose native half
// is already gone.
//
// The base call is in a `finally`, so it runs even when the release fails. That
// is reproduced: the base's flag and event are not skipped by a native failure,
// and the release's error is the one returned.
func (t *Texture2D) DisposeByBoolean(disposing bool) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	var released error
	if !t.texture.IsDisposed() {
		released = t.texture.releaseNativeObject()
	}
	baseErr := t.texture.disposeGraphicsResource(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
