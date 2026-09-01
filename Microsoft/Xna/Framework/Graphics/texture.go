package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// Texture is Microsoft.Xna.Framework.Graphics.Texture, the middle link of the
// graphics base chain:
//
//	GraphicsResource -> Texture -> Texture2D
//	                            -> Texture3D
//	                            -> TextureCube
//
// # Its whole public surface is two read-only properties
//
//	.method public hidebysig specialname instance SurfaceFormat get_Format()
//	  ldarg.0; ldfld _format; ret
//
//	.method public hidebysig specialname instance int32 get_LevelCount()
//	  ldarg.0; ldfld _levelCount; ret
//
// Nothing else on it is public: `InitializeDescription`, `CreateStateWrapper`,
// `GetComPtr` and `MustClamp` are `famandassem` or `assembly`, and the seven
// static helpers are `assembly`. It declares no constructor a consumer can
// reach, no Dispose override, and no ToString -- which is why a Texture2D's
// ToString is GraphicsResource's.
//
// Both getters are plain field reads with NO disposal check, so both are
// infallible and both answer after disposal. That is the reference's behaviour,
// not a relaxation: `_format` and `_levelCount` are managed fields that
// `InitializeDescription` fills once at construction.
//
// # What fills them
//
//	InitializeDescription(SurfaceFormat format)
//	  _format = format;
//	  _levelCount = GetComPtr()->GetLevelCount();     // IDirect3DBaseTexture9
//
// The format is the one the constructor asked for and the level count comes
// from the created texture, which is exactly the split CNA reports: CNA-Go
// stores the requested format and reads the level count out of
// `interop.TextureInfo`.
type Texture struct {
	// resource is the composed GraphicsResource. Private named composition, no
	// embedding, no accessor -- the settled rule.
	resource *GraphicsResource
	// format is `_format`, stored by InitializeDescription.
	format SurfaceFormat
	// levelCount is `_levelCount`, read from the created native texture.
	levelCount int32
}

// newTexture composes a GraphicsResource and records the description
// InitializeDescription records. It is unexported because Texture's constructor
// is `assembly` in the reference: a consumer cannot make one there either.
func newTexture(device *GraphicsDevice, resource *interop.Resource, format SurfaceFormat, levelCount int32) *Texture {
	texture := &Texture{
		resource:   newGraphicsResource(device, resource),
		format:     format,
		levelCount: levelCount,
	}
	texture.resource.bindDerived(texture)
	return texture
}

// clrTypeName answers for a bare Texture. A Texture2D rebinds the chain to
// itself, so this is reached only by a Texture that nothing composes -- which
// the reference's `abstract` makes unconstructable, and CNA-Go's unexported
// constructor makes unreachable from outside the package.
func (t *Texture) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.Texture" }

// bindDerived forwards the CLR `this` to the root of the chain. Texture holds
// no copy of its own: there is one outermost object, and one place that answers
// with it.
func (t *Texture) bindDerived(derived graphicsResourceObject) {
	if t == nil || t.resource == nil {
		return
	}
	t.resource.bindDerived(derived)
}

// Format is Texture::get_Format, one `ldfld`, no disposal check.
func (t *Texture) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.format
}

// LevelCount is Texture::get_LevelCount, one `ldfld`, no disposal check.
func (t *Texture) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.levelCount
}

// ---------------------------------------------------------------------------
// The seven members Texture inherits from GraphicsResource, forwarded.
//
// Every one is written out rather than promoted. Texture overrides none of
// them, so each forwards to the composed base unchanged -- but writing them out
// is what makes the inherited set a fact the verifier checks rather than a
// property of Go's promotion rules.
// ---------------------------------------------------------------------------

// Every forwarder below tolerates a nil receiver and a nil composed base, for
// the reason the whole package does: a Go zero value is a state the CLR has no
// counterpart for -- there is no "half-constructed" object there -- and a panic
// crossing into consumer code is reserved for violated internal invariants. A
// zero value answers with the zero value the corresponding field would hold.

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (t *Texture) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *Texture) Name() string {
	if t == nil {
		return ""
	}
	return t.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *Texture) SetName(value string) {
	if t == nil {
		return
	}
	t.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *Texture) Tag() any {
	if t == nil {
		return nil
	}
	return t.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *Texture) SetTag(value any) {
	if t == nil {
		return
	}
	t.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (t *Texture) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString, which resolves the runtime type
// through the CLR `this` -- so a Texture2D composing this answers with its own
// name, not with Texture's.
func (t *Texture) ToString() string {
	if t == nil {
		return ""
	}
	return t.resource.ToString()
}

// AddDisposingHandler is add_Disposing, on the base's own registration list:
// the event is declared by GraphicsResource.
func (t *Texture) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *Texture) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.resource.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), and it is spelled `Dispose` rather
// than `DisposeByNone` because on THIS type it is not one of two.
//
// The settled overload rule reads the derived type's EFFECTIVE method set.
// GraphicsResource declares two Dispose members, but only `Dispose()` is
// public; `Dispose(bool)` is `family`, so it is not inherited public surface.
// Texture declares no Dispose of its own -- the reference has no override,
// because Texture holds no native object beyond the handle GraphicsResource
// already carries -- so its Dispose group has exactly one member and takes the
// unsuffixed name. Texture2D's group has two, and takes both suffixes. The same
// CLR member is therefore spelled differently on the two types, which is what
// the rule says to do and is worth stating because it looks like an
// inconsistency until the group sizes are counted.
//
// The reference's `callvirt Dispose(bool)` reaches the derived override. A bare
// Texture has none, so this is GraphicsResource's own body: the flag and the
// event, and no native release. Nothing constructs a bare Texture -- the
// reference's class is `abstract` and CNA-Go's constructor is unexported.
func (t *Texture) Dispose() error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.resource.DisposeByNone()
}

// The three unexported reaches a derived texture needs into the chain. None is
// public surface: the raw-handle rule forbids the last one being exported, and
// the contract declares no accessor for the base object.

// releaseNativeObject destroys the one owned CNA resource.
func (t *Texture) releaseNativeObject() error {
	if t == nil {
		return nil
	}
	return t.resource.releaseNativeObject()
}

// disposeGraphicsResource is `base.Dispose(disposing)` for a derived override.
func (t *Texture) disposeGraphicsResource(disposing bool) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.resource.DisposeByBoolean(disposing)
}

// nativeResource is the owned CNA handle, for a derived type's own operations.
func (t *Texture) nativeResource() *interop.Resource {
	if t == nil {
		return nil
	}
	return t.resource.nativeResource()
}
