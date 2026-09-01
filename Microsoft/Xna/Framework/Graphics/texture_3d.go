package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 71 — Texture3D.
// ---------------------------------------------------------------------------

// Texture3D is Microsoft.Xna.Framework.Graphics.Texture3D:
//
//	.class public auto ansi beforefieldinit Texture3D
//	       extends Microsoft.Xna.Framework.Graphics.Texture
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_texture3d_destroy.
//
// # Its three declared properties are field reads
//
//	get_Width   ldarg.0; ldfld _width;  ret
//	get_Height  ldarg.0; ldfld _height; ret
//	get_Depth   ldarg.0; ldfld _depth;  ret
//
// All three are stored once by the constructor from the CREATED volume's
// D3DVOLUME_DESC, so all three answer after disposal and none carries an error
// -- the same shape Texture2D's Width and Height have, and for the same
// measured reason.
type Texture3D struct {
	// texture is the composed Texture. Private named composition, never
	// embedding, and no accessor for it.
	texture *Texture
	width   int32
	height  int32
	depth   int32
}

// newTexture3D is Texture3D::InitializeDescription:
//
//	GetLevelDesc(0, &desc);
//	_width = desc.Width; _height = desc.Height; _depth = desc.Depth;
//	Texture::InitializeDescription(format);
//
// The three dimensions come from the CREATED volume rather than from the
// arguments, which is why they are read out of CNA's reported info.
func newTexture3D(device *GraphicsDevice, resource *interop.Resource, info interop.Texture3DInfo, requested *SurfaceFormat) *Texture3D {
	format := SurfaceFormat(info.Format)
	if requested != nil {
		format = *requested
	}
	texture := &Texture3D{
		texture: newTexture(device, resource, format, int32(info.Levels)),
		width:   int32(info.Width),
		height:  int32(info.Height),
		depth:   int32(info.Depth),
	}
	texture.texture.bindDerived(texture)
	return texture
}

// NewTexture3D is Texture3D::.ctor(GraphicsDevice, Int32, Int32, Int32, Boolean,
// SurfaceFormat), the type's one public constructor.
//
// Its one reproduced guard is the reference's own first check; everything after
// it is CNA's, on the decision Texture2D's constructors already took.
//
// CNA is explicit that a volume texture is a RENDERER capability rather than a
// universal one: cna_texture3d_create documents "unsupported renderers return
// NOT_SUPPORTED". A refusal from here is therefore CNA reporting the renderer,
// not the binding failing, and it is reported as CNA's own result.
func NewTexture3D(
	graphicsDevice *GraphicsDevice, width, height, depth int32, mipMap bool, format SurfaceFormat,
) (*Texture3D, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if width < 0 || height < 0 || depth < 0 {
		return nil, fmt.Errorf("%w: a volume dimension is negative: %dx%dx%d",
			errGraphicsResourceArgument, width, height, depth)
	}
	resource, info, err := graphicsDevice.device.CreateTexture3D(
		uint32(width), uint32(height), uint32(depth), mipMap, uint32(format))
	if err != nil {
		return nil, err
	}
	return newTexture3D(graphicsDevice, resource, info, &format), nil
}

// Width is Texture3D::get_Width, one `ldfld` with no disposal check.
func (t *Texture3D) Width() int32 {
	if t == nil {
		return 0
	}
	return t.width
}

// Height is Texture3D::get_Height.
func (t *Texture3D) Height() int32 {
	if t == nil {
		return 0
	}
	return t.height
}

// Depth is Texture3D::get_Depth.
func (t *Texture3D) Depth() int32 {
	if t == nil {
		return 0
	}
	return t.depth
}

// clrTypeName is System.Object::ToString's answer for a Texture3D.
func (t *Texture3D) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.Texture3D" }

// bindDerived forwards the CLR `this` to the root of the chain.
func (t *Texture3D) bindDerived(derived graphicsResourceObject) {
	if t == nil || t.texture == nil {
		return
	}
	t.texture.bindDerived(derived)
}

// textureBase answers with the composed Texture.
func (t *Texture3D) textureBase() *Texture {
	if t == nil {
		return nil
	}
	return t.texture
}

// nativeResource is the one owned CNA handle. Unexported; it never escapes.
func (t *Texture3D) nativeResource() *interop.Resource {
	if t == nil || t.texture == nil {
		return nil
	}
	return t.texture.nativeResource()
}

// The eleven inherited members: two from Texture, nine from GraphicsResource.

// Format is Texture::get_Format.
func (t *Texture3D) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.texture.Format()
}

// LevelCount is Texture::get_LevelCount.
func (t *Texture3D) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.texture.LevelCount()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (t *Texture3D) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.texture.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *Texture3D) Name() string {
	if t == nil {
		return ""
	}
	return t.texture.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *Texture3D) SetName(value string) {
	if t == nil {
		return
	}
	t.texture.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *Texture3D) Tag() any {
	if t == nil {
		return nil
	}
	return t.texture.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *Texture3D) SetTag(value any) {
	if t == nil {
		return
	}
	t.texture.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (t *Texture3D) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.texture.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (t *Texture3D) ToString() string {
	if t == nil {
		return ""
	}
	return t.texture.ToString()
}

// AddDisposingHandler is add_Disposing.
func (t *Texture3D) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.texture.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *Texture3D) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (t *Texture3D) DisposeByNone() error {
	return t.DisposeByBoolean(true)
}

// DisposeByBoolean is Texture3D::Dispose(bool), the shape every derived
// graphics resource here has.
func (t *Texture3D) DisposeByBoolean(disposing bool) error {
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
