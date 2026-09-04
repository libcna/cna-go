package graphics

import (
	"fmt"
	"io"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// RenderTarget2D is Microsoft.Xna.Framework.Graphics.RenderTarget2D: a texture
// the device can draw INTO, and the profile's first type whose existence makes
// CLR base substitutability a live requirement rather than a latent one.
//
//	.class public RenderTarget2D extends Texture2D
//	       implements IDynamicGraphicsResource, IGraphicsResource, IDisposable
//	  .field assembly RenderTargetHelper helper
//	  .field assembly bool _contentLost
//	  .field private EventHandler`1<EventArgs> <backing_store>ContentLost
//
// # It is a Texture2D, and CNA agrees
//
// The composition is the settled one: a private named `*Texture2D`, no
// embedding, no accessor. What is new is that a RenderTarget2D must be usable
// where the profile names Texture2D in a public signature -- seven positions,
// every one of them SpriteBatch.Draw's `texture` -- and Texture2DReference is
// how a Go parameter accepts it.
//
// The native half needs no adapter at all: CNA's texture routes take a
// "Texture2D or matching render-target handle", so the same handle answers both
// surfaces. The Go interface exists because Go has no subtyping, not because
// the runtime has two objects.
//
// # Ownership
//
//	OWNED, generation-checked, released through cna_render_target_destroy.
//
// It is a DISTINCT CNA kind. `cna_texture2d_destroy` is documented as
// destroying a Texture2D but not a render target, so the composed
// GraphicsResource carries the render-target kind and destroys it through the
// route that matches -- one native owner, the right destroy.
type RenderTarget2D struct {
	// texture is the composed Texture2D, which carries the composed Texture,
	// which carries the composed GraphicsResource and the one native handle.
	texture *Texture2D
	// The three values RenderTargetHelper stores. Each is what CNA APPLIED,
	// which is the same split the reference has: CreateRenderTarget hands its
	// preferences to GraphicsAdapter::QueryFormat, and the helper stores what
	// the query SELECTED.
	depthStencilFormat DepthFormat
	multiSampleCount   int32
	usage              RenderTargetUsage
	// contentLost is `_contentLost`, the reference's own latching field. See
	// IsContentLost.
	contentLost bool
	// rendererAvailable is CNA's, and has no XNA counterpart. CNA permits
	// creation on a backend with no real off-screen storage: creation succeeds,
	// this is false, and binding reports NOT_SUPPORTED. It is kept so the
	// projection can say WHICH refusal a failed bind is.
	rendererAvailable bool
	// contentLostEvent is the `<backing_store>ContentLost` delegate field.
	contentLostEvent framework.EventSource[*framework.EventArgs]
}

// NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32 is
// RenderTarget2D::.ctor(GraphicsDevice, Int32, Int32).
//
// Its IL passes five `ldc.i4.0` to CreateRenderTarget, so every default is read
// off the constructor rather than chosen:
//
//	mipMap                    false
//	preferredFormat           SurfaceFormat.Color
//	preferredDepthFormat      DepthFormat.None
//	preferredMultiSampleCount 0
//	usage                     RenderTargetUsage.DiscardContents
func NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32(
	graphicsDevice *GraphicsDevice, width, height int32,
) (*RenderTarget2D, error) {
	return NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage(
		graphicsDevice, width, height, false, SurfaceFormatColor, DepthFormatNone, 0, RenderTargetUsageDiscardContents)
}

// NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormat
// is RenderTarget2D::.ctor(GraphicsDevice, Int32, Int32, Boolean, SurfaceFormat,
// DepthFormat), which is the eight-argument one with two `ldc.i4.0`:
// no multisampling and DiscardContents.
func NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormat(
	graphicsDevice *GraphicsDevice, width, height int32, mipMap bool,
	preferredFormat SurfaceFormat, preferredDepthFormat DepthFormat,
) (*RenderTarget2D, error) {
	return NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage(
		graphicsDevice, width, height, mipMap, preferredFormat, preferredDepthFormat, 0, RenderTargetUsageDiscardContents)
}

// NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage
// is the full RenderTarget2D::.ctor, which is the one the other two forward to.
//
// # The arguments are PREFERENCES
//
//	CreateRenderTarget(...)
//	  if (graphicsDevice == null)
//	      throw new ArgumentNullException("graphicsDevice",
//	                                      FrameworkResources.DeviceCannotBeNullOnResourceCreate);
//	  graphicsDevice.Adapter.QueryFormat(..., preferredFormat, preferredDepthFormat,
//	                                     preferredMultiSampleCount,
//	                                     out selectedFormat, out selectedDepth, out selectedSamples);
//	  Texture2D.ValidateCreationParameters(..., selectedFormat, mipMap);
//	  this.helper = new RenderTargetHelper(this, width, height, selectedFormat,
//	                                       selectedDepth, selectedSamples, usage, ...);
//
// `preferredFormat` and `preferredDepthFormat` are Microsoft's own parameter
// names, and the properties report the SELECTED values. CNA makes the same
// distinction from the other side: CNA_RenderTargetInfo carries the APPLIED
// format, depth format, sample count and usage, and this projection reads them
// from there rather than echoing the arguments back.
//
// The one guard reproduced is the reference's first. Everything after it is a
// capability decision CNA makes, and CNA reports its own refusal rather than
// D3D9's message, for the reason Texture2D's constructors already record.
func NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage(
	graphicsDevice *GraphicsDevice, width, height int32, mipMap bool,
	preferredFormat SurfaceFormat, preferredDepthFormat DepthFormat,
	preferredMultiSampleCount int32, usage RenderTargetUsage,
) (*RenderTarget2D, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if width < 0 || height < 0 {
		return nil, fmt.Errorf("%w: a render-target dimension is negative: %dx%d",
			errGraphicsResourceArgument, width, height)
	}
	if preferredMultiSampleCount < 0 {
		return nil, fmt.Errorf("%w: a multisample count is negative: %d",
			errGraphicsResourceArgument, preferredMultiSampleCount)
	}
	resource, info, err := graphicsDevice.device.CreateRenderTarget2D(
		uint32(width), uint32(height), mipMap,
		uint32(preferredFormat), uint32(preferredDepthFormat),
		preferredMultiSampleCount, uint32(usage))
	if err != nil {
		return nil, err
	}
	target := &RenderTarget2D{
		texture: newTexture2D(graphicsDevice, resource, interop.TextureInfo{
			Width: info.Width, Height: info.Height, Levels: info.LevelCount, Format: info.Format,
		}, nil),
		depthStencilFormat: DepthFormat(info.DepthFormat),
		multiSampleCount:   info.MultiSampleCount,
		usage:              RenderTargetUsage(info.Usage),
		contentLost:        info.IsContentLost,
		rendererAvailable:  info.RendererAvailable,
	}
	// The CLR `this`, rebound through the whole chain: ToString must answer
	// "Microsoft.Xna.Framework.Graphics.RenderTarget2D", and Disposing must
	// announce this object rather than any of its three composed halves.
	target.texture.bindDerived(target)
	return target, nil
}

// clrTypeName is System.Object::ToString's answer for a RenderTarget2D.
func (t *RenderTarget2D) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.RenderTarget2D"
}

// texture2D satisfies Texture2DReference with the composed base, which is the
// SAME logical native object: CNA's texture routes accept this handle.
func (t *RenderTarget2D) texture2D() *Texture2D {
	if t == nil {
		return nil
	}
	return t.texture
}

// DepthStencilFormat is RenderTarget2D::get_DepthStencilFormat:
//
//	ldfld helper; ldfld RenderTargetHelper::depthFormat
//
// Two field reads over the format QueryFormat selected, so it is infallible and
// answers after disposal.
func (t *RenderTarget2D) DepthStencilFormat() DepthFormat {
	if t == nil {
		return DepthFormatNone
	}
	return t.depthStencilFormat
}

// MultiSampleCount is RenderTarget2D::get_MultiSampleCount, the same shape over
// the sample count QueryFormat selected.
func (t *RenderTarget2D) MultiSampleCount() int32 {
	if t == nil {
		return 0
	}
	return t.multiSampleCount
}

// RenderTargetUsage is RenderTarget2D::get_RenderTargetUsage, the same shape
// over the usage the constructor was given -- this one is stored, not selected.
func (t *RenderTarget2D) RenderTargetUsage() RenderTargetUsage {
	if t == nil {
		return RenderTargetUsageDiscardContents
	}
	return t.usage
}

// IsContentLost is RenderTarget2D::get_IsContentLost, and it LATCHES:
//
//	if (!_contentLost) _contentLost = _parent.IsDeviceLost;
//	return _contentLost;
//
// Once true it stays true until something clears the field, which in the
// reference is the render-target helper on recreation. The projection asks CNA
// the same question -- CNA_RenderTargetInfo::is_content_lost, which is true from
// the moment a renderer reports a real device reset until the target is next
// bound -- and latches it the same way, so the Go field is the reference's
// field and CNA's report plays the device's part.
//
// It is fallible because the read reaches CNA, which the reference's read of
// GraphicsDevice::IsDeviceLost reaches D3D for. A disposed target reports rather
// than answering false, because the question is about a resource that is gone.
func (t *RenderTarget2D) IsContentLost() (bool, error) {
	if t == nil || t.texture == nil {
		return false, errGraphicsResourceNil
	}
	if t.contentLost {
		return true, nil
	}
	resource := t.texture.nativeResource()
	if resource == nil {
		return false, interop.ErrDisposed
	}
	info, err := resource.RenderTargetInfo()
	if err != nil {
		return false, err
	}
	if info.IsContentLost {
		t.contentLost = true
	}
	return t.contentLost, nil
}

// setContentLost is RenderTarget2D::SetContentLost(bool), the
// IDynamicGraphicsResource member the reference dispatches on. It stores the
// flag and raises ContentLost only when the flag is true, exactly as the
// dynamic buffers' does -- one body, four implementing types.
//
// Foundation 84 projected it. Until then the type carried the latch and the
// event with nothing that could clear or raise them, because the interface the
// reference dispatches through was not projected at all.
func (t *RenderTarget2D) setContentLost(isContentLost bool) {
	t.contentLost = isContentLost
	if isContentLost {
		_ = t.contentLostEvent.Raise(t, framework.EventArgsEmpty())
	}
}

// AddContentLostHandler is add_ContentLost, on the settled two-accessor event
// projection.
//
// # The event has no raise site CNA-Go can reach
//
// The reference raises ContentLost from RenderTargetHelper's device-reset path,
// which runs when D3D9 reports a lost device. CNA has a counterpart --
// cna_render_target_subscribe_content_lost -- and its documentation is explicit
// that ONLY the DIRECTX9, DIRECT2D and SKIA renderer families can report a lost
// device; every other family never raises it, and a subscription there is valid
// and simply silent. The qualified artifacts are HEADLESS and SOFTWARE, so this
// event cannot fire in the qualified environment at all.
//
// The accessors are projected because the contract declares them, and the
// registration list is real. What is recorded rather than claimed is that no
// qualified environment can deliver one.
func (t *RenderTarget2D) AddContentLostHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.contentLostEvent.Add(handler)
}

// RemoveContentLostHandler is remove_ContentLost.
func (t *RenderTarget2D) RemoveContentLostHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.contentLostEvent.Remove(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited, whose `callvirt`
// reaches this type's override.
func (t *RenderTarget2D) DisposeByNone() error {
	return t.DisposeByBoolean(true)
}

// DisposeByBoolean is RenderTarget2D::Dispose(bool), whose IL is
//
//	if (disposing) { try { } finally { base.Dispose(true); } }
//	else                                base.Dispose(false);
//
// The try block is EMPTY -- `~RenderTarget2D()` is a single `ret` -- so the
// whole override is the base call, and both branches make it. RenderTarget2D
// adds nothing to its base's disposal, which is worth stating because the shape
// looks like it does: the `try/finally` exists so a `~RenderTarget2D` that did
// something would not be able to skip the base.
func (t *RenderTarget2D) DisposeByBoolean(disposing bool) error {
	if t == nil || t.texture == nil {
		return errGraphicsResourceNil
	}
	return t.texture.DisposeByBoolean(disposing)
}

// ---------------------------------------------------------------------------
// The sixteen members RenderTarget2D inherits from its composed base chain.
//
// Nine come from GraphicsResource, two from Texture and five from Texture2D.
// Every one is written out rather than promoted, and each forwards to the
// nearest composed link rather than reaching past it: the chain is four deep,
// and a forwarder that skipped a link would skip whatever that link overrides.
//
// The inherited members whose Go projection is a PACKAGE FUNCTION are absent by
// rule, not by omission. `Texture2D::FromStream` is static and its Go spelling
// already names its declaring type; the six SetData/GetData overloads are
// generic instance methods the generic-method rule turned into functions whose
// first parameter is the receiver, and that parameter is a Texture2DReference,
// so `Texture2DSetDataBySliceOfT(renderTarget, pixels)` already works.
// ---------------------------------------------------------------------------

// Width is Texture2D::get_Width, the width CNA applied to the created target.
func (t *RenderTarget2D) Width() int32 {
	if t == nil {
		return 0
	}
	return t.texture.Width()
}

// Height is Texture2D::get_Height.
func (t *RenderTarget2D) Height() int32 {
	if t == nil {
		return 0
	}
	return t.texture.Height()
}

// Bounds is Texture2D::get_Bounds.
func (t *RenderTarget2D) Bounds() framework.Rectangle {
	if t == nil {
		return framework.Rectangle{}
	}
	return t.texture.Bounds()
}

// SaveAsPng is Texture2D::SaveAsPng(Stream, Int32, Int32). It reads the render
// target through CNA's texture routes, which accept a render-target handle --
// which is how a render target's contents are inspected at all.
func (t *RenderTarget2D) SaveAsPng(stream io.Writer, width, height int32) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.SaveAsPng(stream, width, height)
}

// SaveAsJpeg is Texture2D::SaveAsJpeg(Stream, Int32, Int32).
func (t *RenderTarget2D) SaveAsJpeg(stream io.Writer, width, height int32) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.SaveAsJpeg(stream, width, height)
}

// Format is Texture::get_Format, the color format CNA applied.
func (t *RenderTarget2D) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.texture.Format()
}

// LevelCount is Texture::get_LevelCount.
func (t *RenderTarget2D) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.texture.LevelCount()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (t *RenderTarget2D) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.texture.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *RenderTarget2D) Name() string {
	if t == nil {
		return ""
	}
	return t.texture.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *RenderTarget2D) SetName(value string) {
	if t == nil {
		return
	}
	t.texture.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *RenderTarget2D) Tag() any {
	if t == nil {
		return nil
	}
	return t.texture.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *RenderTarget2D) SetTag(value any) {
	if t == nil {
		return
	}
	t.texture.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (t *RenderTarget2D) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.texture.IsDisposed()
}

// ToString is GraphicsResource::ToString, whose fallback is the RUNTIME type --
// which for this object is RenderTarget2D, resolved through the CLR `this` the
// constructor installed and forwarded across three composition links.
func (t *RenderTarget2D) ToString() string {
	if t == nil {
		return ""
	}
	return t.texture.ToString()
}

// AddDisposingHandler is add_Disposing, on GraphicsResource's registration list.
func (t *RenderTarget2D) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.texture.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *RenderTarget2D) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.texture.RemoveDisposingHandler(subscription)
}
