package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 71 — TextureCube.
// ---------------------------------------------------------------------------

// TextureCube is Microsoft.Xna.Framework.Graphics.TextureCube:
//
//	.class public auto ansi beforefieldinit TextureCube
//	       extends Microsoft.Xna.Framework.Graphics.Texture
//
// The third link of the graphics base chain, beside Texture2D and Texture3D,
// and the second one CNA-Go projects.
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_texturecube_destroy.
//
// CNA documents that route as destroying a TextureCube but NOT a
// RenderTargetCube, which is the same per-kind split cna_texture2d_destroy has
// against a render target — so the resource kind is its own and the destroy
// dispatch is per kind, not per family.
//
// # Its one declared property is a field read
//
//	get_Size   ldarg.0; ldfld _size; ret
//
// `_size` is stored once by the constructor from the CREATED cube's
// D3DSURFACE_DESC, exactly as Texture2D's `_width` is, so it answers after
// disposal and carries no error.
type TextureCube struct {
	// texture is the composed Texture. Private named composition, never
	// embedding, and no accessor for it.
	texture *Texture
	// size is `_size`, the width and height of every square face.
	size int32
}

// newTextureCube is TextureCube::InitializeDescription, the whole of what the
// constructor does after the native object exists:
//
//	GetLevelDesc(D3DCUBEMAP_FACE_POSITIVE_X, 0, &desc);
//	_size = desc.Width;
//	Texture::InitializeDescription(format);
//
// The size comes from the CREATED cube's first face rather than from the
// argument, which is why it is read out of CNA's reported info: a renderer that
// rounded a size up would be visible instead of hidden.
func newTextureCube(device *GraphicsDevice, resource *interop.Resource, info interop.TextureCubeInfo, requested *SurfaceFormat) *TextureCube {
	format := SurfaceFormat(info.Format)
	if requested != nil {
		format = *requested
	}
	cube := &TextureCube{
		texture: newTexture(device, resource, format, int32(info.Levels)),
		size:    int32(info.Size),
	}
	// The CLR `this`, rebinding the whole chain: GraphicsResource::ToString
	// answers with the RUNTIME type's full name.
	cube.texture.bindDerived(cube)
	return cube
}

// NewTextureCube is TextureCube::.ctor(GraphicsDevice, Int32, Boolean, SurfaceFormat), the type's
// one public constructor.
//
// Its guard is the reference's own first check, and it is the only one
// reproduced here:
//
//	if (graphicsDevice == null)
//	    throw new ArgumentNullException("graphicsDevice",
//	                                    FrameworkResources.DeviceCannotBeNullOnResourceCreate);
//
// Everything after it -- a size of zero, a format the renderer has no cube
// storage for -- is validated by CNA, which refuses with its own result. That
// is the decision Texture2D's constructors already took, for the same reason:
// reproducing D3D9's cube-capability tables would mean asserting a support
// decision CNA-Go did not make.
func NewTextureCube(
	graphicsDevice *GraphicsDevice, size int32, mipMap bool, format SurfaceFormat,
) (*TextureCube, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if size < 0 {
		// CNA takes the size as uint32, so a negative would arrive as an
		// enormous positive. This refusal exists so the conversion cannot
		// silently invent a dimension.
		return nil, fmt.Errorf("%w: a cube face size is negative: %d",
			errGraphicsResourceArgument, size)
	}
	resource, info, err := graphicsDevice.device.CreateTextureCube(uint32(size), mipMap, uint32(format))
	if err != nil {
		return nil, err
	}
	return newTextureCube(graphicsDevice, resource, info, &format), nil
}

// Size is TextureCube::get_Size, one `ldfld` with no disposal check.
func (t *TextureCube) Size() int32 {
	if t == nil {
		return 0
	}
	return t.size
}

// clrTypeName is System.Object::ToString's answer for a TextureCube.
func (t *TextureCube) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.TextureCube" }

// bindDerived forwards the CLR `this` to the root of the chain.
func (t *TextureCube) bindDerived(derived graphicsResourceObject) {
	if t == nil || t.texture == nil {
		return
	}
	t.texture.bindDerived(derived)
}

// textureBase answers with the composed Texture, which is what makes a
// TextureCube usable at a Texture-typed parameter position.
func (t *TextureCube) textureBase() *Texture {
	if t == nil {
		return nil
	}
	return t.texture
}

// nativeResource is the one owned CNA handle, for this package's own
// operations. Unexported; it never escapes.
func (t *TextureCube) nativeResource() *interop.Resource {
	if t == nil || t.texture == nil {
		return nil
	}
	return t.texture.nativeResource()
}

// The eleven inherited members: two from Texture, nine from GraphicsResource.

// Format is Texture::get_Format.
func (t *TextureCube) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.texture.Format()
}

// LevelCount is Texture::get_LevelCount.
func (t *TextureCube) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.texture.LevelCount()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (t *TextureCube) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.texture.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *TextureCube) Name() string {
	if t == nil {
		return ""
	}
	return t.texture.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *TextureCube) SetName(value string) {
	if t == nil {
		return
	}
	t.texture.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *TextureCube) Tag() any {
	if t == nil {
		return nil
	}
	return t.texture.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *TextureCube) SetTag(value any) {
	if t == nil {
		return
	}
	t.texture.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (t *TextureCube) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.texture.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (t *TextureCube) ToString() string {
	if t == nil {
		return ""
	}
	return t.texture.ToString()
}

// AddDisposingHandler is add_Disposing.
func (t *TextureCube) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.texture.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *TextureCube) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited. `callvirt` reaches
// THIS type's Dispose(bool).
func (t *TextureCube) DisposeByNone() error {
	return t.DisposeByBoolean(true)
}

// DisposeByBoolean is TextureCube::Dispose(bool), the same shape Texture2D's
// has: the finalizer path STILL releases the native cube, because
// `!TextureCube()` passes a hardcoded true to ReleaseNativeObject regardless of
// which branch reached it, and the base call is in a `finally`.
func (t *TextureCube) DisposeByBoolean(disposing bool) error {
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
