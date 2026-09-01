package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 73 — RenderTargetCube and RenderTargetBinding.
// ---------------------------------------------------------------------------

// RenderTargetCube is Microsoft.Xna.Framework.Graphics.RenderTargetCube:
//
//	.class public RenderTargetCube extends TextureCube
//	       implements IDynamicGraphicsResource, IGraphicsResource, IDisposable
//
// It is RenderTarget2D's cube twin, one link further down the chain that
// Foundation 71 completed: GraphicsResource -> Texture -> TextureCube ->
// RenderTargetCube.
//
// # Ownership
//
//	OWNED, generation-checked, released through cna_render_target_destroy.
//
// A DISTINCT CNA kind, exactly as RenderTarget2D is:
// `cna_texturecube_destroy` is documented as destroying a TextureCube but NOT a
// RenderTargetCube, so the composed GraphicsResource carries the render-target
// kind and destroys it through the route that matches.
type RenderTargetCube struct {
	// cube is the composed TextureCube, which carries the composed Texture,
	// which carries the composed GraphicsResource and the one native handle.
	cube *TextureCube
	// The three values RenderTargetHelper stores: what CNA APPLIED, which is
	// the same split RenderTarget2D records.
	depthStencilFormat DepthFormat
	multiSampleCount   int32
	usage              RenderTargetUsage
	contentLost        bool
	rendererAvailable  bool
	contentLostEvent   framework.EventSource[*framework.EventArgs]
}

// NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormat
// is RenderTargetCube::.ctor(GraphicsDevice, Int32, Boolean, SurfaceFormat,
// DepthFormat), whose IL passes two `ldc.i4.0` to CreateRenderTarget:
//
//	preferredMultiSampleCount 0
//	usage                     RenderTargetUsage.DiscardContents
func NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormat(
	graphicsDevice *GraphicsDevice, size int32, mipMap bool,
	preferredFormat SurfaceFormat, preferredDepthFormat DepthFormat,
) (*RenderTargetCube, error) {
	return NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage(
		graphicsDevice, size, mipMap, preferredFormat, preferredDepthFormat, 0, RenderTargetUsageDiscardContents)
}

// NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage
// is the seven-argument constructor the other funnels into.
//
// The one guard reproduced is the reference's first; everything after it is a
// capability decision CNA makes, and CNA reports its own refusal rather than
// D3D9's message -- the decision RenderTarget2D's constructors already record.
func NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage(
	graphicsDevice *GraphicsDevice, size int32, mipMap bool,
	preferredFormat SurfaceFormat, preferredDepthFormat DepthFormat,
	preferredMultiSampleCount int32, usage RenderTargetUsage,
) (*RenderTargetCube, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if size < 0 {
		return nil, fmt.Errorf("%w: a cube face size is negative: %d", errGraphicsResourceArgument, size)
	}
	if preferredMultiSampleCount < 0 {
		return nil, fmt.Errorf("%w: a multisample count is negative: %d",
			errGraphicsResourceArgument, preferredMultiSampleCount)
	}
	resource, info, err := graphicsDevice.device.CreateRenderTargetCube(
		uint32(size), mipMap, uint32(preferredFormat), uint32(preferredDepthFormat),
		preferredMultiSampleCount, uint32(usage))
	if err != nil {
		return nil, err
	}
	// CNA's render-target info reports width and height; a cube's faces are
	// square, so the SIZE is the width CNA applied.
	target := &RenderTargetCube{
		cube: newTextureCube(graphicsDevice, resource, interop.TextureCubeInfo{
			Size: info.Width, Levels: info.LevelCount, Format: info.Format,
		}, nil),
		depthStencilFormat: DepthFormat(info.DepthFormat),
		multiSampleCount:   info.MultiSampleCount,
		usage:              RenderTargetUsage(info.Usage),
		contentLost:        info.IsContentLost,
		rendererAvailable:  info.RendererAvailable,
	}
	target.cube.bindDerived(target)
	return target, nil
}

// clrTypeName is System.Object::ToString's answer for a RenderTargetCube.
func (t *RenderTargetCube) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.RenderTargetCube"
}

// textureBase satisfies TextureReference across two composition links.
func (t *RenderTargetCube) textureBase() *Texture {
	if t == nil || t.cube == nil {
		return nil
	}
	return t.cube.textureBase()
}

// nativeResource is the one owned CNA handle. Unexported.
func (t *RenderTargetCube) nativeResource() *interop.Resource {
	if t == nil || t.cube == nil {
		return nil
	}
	return t.cube.nativeResource()
}

// DepthStencilFormat is RenderTargetCube::get_DepthStencilFormat, two field
// reads over the format QueryFormat selected.
func (t *RenderTargetCube) DepthStencilFormat() DepthFormat {
	if t == nil {
		return DepthFormatNone
	}
	return t.depthStencilFormat
}

// MultiSampleCount is RenderTargetCube::get_MultiSampleCount.
func (t *RenderTargetCube) MultiSampleCount() int32 {
	if t == nil {
		return 0
	}
	return t.multiSampleCount
}

// RenderTargetUsage is RenderTargetCube::get_RenderTargetUsage.
func (t *RenderTargetCube) RenderTargetUsage() RenderTargetUsage {
	if t == nil {
		return RenderTargetUsageDiscardContents
	}
	return t.usage
}

// IsContentLost is RenderTargetCube::get_IsContentLost, the same latching read
// RenderTarget2D's is.
func (t *RenderTargetCube) IsContentLost() (bool, error) {
	if t == nil || t.cube == nil {
		return false, errGraphicsResourceNil
	}
	if t.contentLost {
		return true, nil
	}
	resource := t.cube.nativeResource()
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

// AddContentLostHandler is add_ContentLost. The event cannot fire in either
// qualified environment, for the reason RenderTarget2D's cannot: CNA raises it
// only on the DIRECTX9, DIRECT2D and SKIA renderer families.
func (t *RenderTargetCube) AddContentLostHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.contentLostEvent.Add(handler)
}

// RemoveContentLostHandler is remove_ContentLost.
func (t *RenderTargetCube) RemoveContentLostHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.contentLostEvent.Remove(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited, whose `callvirt`
// reaches this type's override.
func (t *RenderTargetCube) DisposeByNone() error {
	return t.DisposeByBoolean(true)
}

// DisposeByBoolean is RenderTargetCube::Dispose(bool), whose whole override is
// the base call -- its `~RenderTargetCube()` is a single `ret`, exactly as
// RenderTarget2D's is.
func (t *RenderTargetCube) DisposeByBoolean(disposing bool) error {
	if t == nil || t.cube == nil {
		return errGraphicsResourceNil
	}
	return t.cube.DisposeByBoolean(disposing)
}

// The twelve members RenderTargetCube inherits: one from TextureCube, two from
// Texture and nine from GraphicsResource.

// Size is TextureCube::get_Size.
func (t *RenderTargetCube) Size() int32 {
	if t == nil {
		return 0
	}
	return t.cube.Size()
}

// Format is Texture::get_Format.
func (t *RenderTargetCube) Format() SurfaceFormat {
	if t == nil {
		return 0
	}
	return t.cube.Format()
}

// LevelCount is Texture::get_LevelCount.
func (t *RenderTargetCube) LevelCount() int32 {
	if t == nil {
		return 0
	}
	return t.cube.LevelCount()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (t *RenderTargetCube) GraphicsDevice() *GraphicsDevice {
	if t == nil {
		return nil
	}
	return t.cube.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (t *RenderTargetCube) Name() string {
	if t == nil {
		return ""
	}
	return t.cube.Name()
}

// SetName is GraphicsResource::set_Name.
func (t *RenderTargetCube) SetName(value string) {
	if t == nil {
		return
	}
	t.cube.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (t *RenderTargetCube) Tag() any {
	if t == nil {
		return nil
	}
	return t.cube.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (t *RenderTargetCube) SetTag(value any) {
	if t == nil {
		return
	}
	t.cube.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (t *RenderTargetCube) IsDisposed() bool {
	if t == nil {
		return true
	}
	return t.cube.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (t *RenderTargetCube) ToString() string {
	if t == nil {
		return ""
	}
	return t.cube.ToString()
}

// AddDisposingHandler is add_Disposing.
func (t *RenderTargetCube) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if t == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return t.cube.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (t *RenderTargetCube) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if t == nil {
		return errGraphicsResourceNil
	}
	return t.cube.RemoveDisposingHandler(subscription)
}

// RenderTargetBinding is Microsoft.Xna.Framework.Graphics.RenderTargetBinding:
//
//	.class public sequential sealed RenderTargetBinding extends System.ValueType
//	  .field private Texture _renderTarget
//	  .field private CubeMapFace _cubeMapFace
//
// A VALUE type with two fields and no mutators, so the Go projection is a
// struct value and its two getters are field reads.
//
// # RenderTarget is typed `Texture`, not RenderTarget2D
//
// which is what lets ONE binding carry either a 2D target or a cube face, and
// is why the getter answers the base type. A consumer who needs the concrete
// one knows which constructor they used.
type RenderTargetBinding struct {
	renderTarget *Texture
	cubeMapFace  CubeMapFace
	// native is the handle CNA binds. It is kept beside the base texture
	// because the base cannot answer which CNA kind it is, and the render-target
	// routes take a render-target handle.
	native *interop.Resource
}

// NewRenderTargetBindingByRenderTarget2D is
// RenderTargetBinding::.ctor(RenderTarget2D):
//
//	if (renderTarget == null)
//	    throw new ArgumentNullException("renderTarget", FrameworkResources.NullNotAllowed);
//	_renderTarget = renderTarget;
//	_cubeMapFace = CubeMapFace.PositiveX;
//
// The face is PositiveX rather than a "none": the CLR enum has no such member,
// and CNA requires the field to be zero for a 2D target -- which PositiveX is.
func NewRenderTargetBindingByRenderTarget2D(renderTarget *RenderTarget2D) (RenderTargetBinding, error) {
	if renderTarget == nil {
		return RenderTargetBinding{}, fmt.Errorf("%w: renderTarget: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	return RenderTargetBinding{
		renderTarget: renderTarget.texture.textureBase(),
		cubeMapFace:  CubeMapFacePositiveX,
		native:       renderTarget.texture.nativeResource(),
	}, nil
}

// NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace is
// RenderTargetBinding::.ctor(RenderTargetCube, CubeMapFace), whose second guard
// is a range check on the face:
//
//	if (cubeMapFace < PositiveX || cubeMapFace > NegativeZ)
//	    throw new ArgumentOutOfRangeException("cubeMapFace",
//	        string.Format(CurrentCulture, FrameworkResources.InvalidEnumValue, cubeMapFace));
func NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(
	renderTarget *RenderTargetCube, cubeMapFace CubeMapFace,
) (RenderTargetBinding, error) {
	if renderTarget == nil {
		return RenderTargetBinding{}, fmt.Errorf("%w: renderTarget: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if cubeMapFace < CubeMapFacePositiveX || cubeMapFace > CubeMapFaceNegativeZ {
		return RenderTargetBinding{}, fmt.Errorf("%w: cubeMapFace: %d", errArgumentOutOfRange, int32(cubeMapFace))
	}
	return RenderTargetBinding{
		renderTarget: renderTarget.textureBase(),
		cubeMapFace:  cubeMapFace,
		native:       renderTarget.nativeResource(),
	}, nil
}

// RenderTargetBindingOperatorImplicitByRenderTarget2D is
// RenderTargetBinding::op_Implicit(RenderTarget2D), the conversion a C# caller
// gets for free when they pass a RenderTarget2D where a binding is wanted.
//
// It is projected as a named function because Go has no implicit conversion,
// and its body is `newobj .ctor(RenderTarget2D); ret` -- so it carries exactly
// that constructor's null refusal and nothing else.
func RenderTargetBindingOperatorImplicitByRenderTarget2D(renderTarget *RenderTarget2D) (RenderTargetBinding, error) {
	return NewRenderTargetBindingByRenderTarget2D(renderTarget)
}

// RenderTarget is RenderTargetBinding::get_RenderTarget, one `ldfld` of the
// BASE-typed field.
func (b RenderTargetBinding) RenderTarget() *Texture {
	return b.renderTarget
}

// CubeMapFace is RenderTargetBinding::get_CubeMapFace, one `ldfld`.
func (b RenderTargetBinding) CubeMapFace() CubeMapFace {
	return b.cubeMapFace
}
