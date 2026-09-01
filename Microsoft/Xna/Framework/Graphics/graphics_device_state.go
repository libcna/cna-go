package graphics

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// GraphicsDevice's render-state surface, and the two Clear overloads that carry
// a buffer mask.
//
// # Why every accessor here asks CNA
//
// Five of the reference's getters are single field reads over a managed cache:
//
//	get_BlendFactor        ldfld cachedBlendFactor       (12 bytes: ldflda + ldobj)
//	get_MultiSampleMask    ldfld cachedMultiSampleMask   (7 bytes)
//	get_ReferenceStencil   ldfld cachedReferenceStencil  (7 bytes)
//	get_GraphicsProfile    ldfld _graphicsProfile        (7 bytes)
//	get_IsDisposed         ldfld isDisposed              (7 bytes)
//
// and two reach D3D after a disposal check:
//
//	get_ScissorRectangle   Helpers::CheckDisposed, then IDirect3DDevice9::GetScissorRect
//	get_GraphicsDeviceStatus  CheckDisposed, then TestCooperativeLevel
//
// Every setter begins with `Helpers::CheckDisposed(this, pComPtr)` and then
// reaches the device.
//
// CNA-Go projects ALL of them as native reads, and the reason is that the
// managed cache is not reproducible here. The reference's constructor fills
// those fields when it creates the D3D device; CNA-Go does not create the
// device -- CNA does, and CNA-Go's GraphicsDevice is a borrowed, generation-
// checked facade over it. A managed cache would therefore start at Go's zero
// values and disagree with the live device until something wrote to it, which
// is precisely the second-source-of-truth failure the settled rule forbids.
//
// The consequence is stated rather than hidden: five members that carry no
// error in the reference carry one here.

// errGraphicsDeviceNil is the Go-only guard for a facade with no device behind
// it. The CLR has no null `this`, so there is no reference counterpart.
var errGraphicsDeviceNil = errors.New("GraphicsDevice is nil")

func (d *GraphicsDevice) live() (*interop.Device, error) {
	if d == nil || d.device == nil {
		return nil, errGraphicsDeviceNil
	}
	return d.device, nil
}

// BlendFactor is GraphicsDevice::get_BlendFactor.
func (d *GraphicsDevice) BlendFactor() (framework.Color, error) {
	device, err := d.live()
	if err != nil {
		return framework.Color{}, err
	}
	r, g, b, a, err := device.BlendFactor()
	if err != nil {
		return framework.Color{}, err
	}
	return framework.NewColorByInt32AndInt32AndInt32AndInt32(int32(r), int32(g), int32(b), int32(a)), nil
}

// SetBlendFactor is GraphicsDevice::set_BlendFactor.
func (d *GraphicsDevice) SetBlendFactor(value framework.Color) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.SetBlendFactor(value.R(), value.G(), value.B(), value.A())
}

// MultiSampleMask is GraphicsDevice::get_MultiSampleMask.
func (d *GraphicsDevice) MultiSampleMask() (int32, error) {
	device, err := d.live()
	if err != nil {
		return 0, err
	}
	return device.MultiSampleMask()
}

// SetMultiSampleMask is GraphicsDevice::set_MultiSampleMask.
//
// The reference validates nothing: every one of the 32 bits is a sample, so
// there is no out-of-range value to refuse and no throw site in the IL.
func (d *GraphicsDevice) SetMultiSampleMask(value int32) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.SetMultiSampleMask(value)
}

// ReferenceStencil is GraphicsDevice::get_ReferenceStencil.
func (d *GraphicsDevice) ReferenceStencil() (int32, error) {
	device, err := d.live()
	if err != nil {
		return 0, err
	}
	return device.ReferenceStencil()
}

// SetReferenceStencil is GraphicsDevice::set_ReferenceStencil.
func (d *GraphicsDevice) SetReferenceStencil(value int32) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.SetReferenceStencil(value)
}

// ScissorRectangle is GraphicsDevice::get_ScissorRectangle, which reaches the
// device in the reference too.
func (d *GraphicsDevice) ScissorRectangle() (framework.Rectangle, error) {
	device, err := d.live()
	if err != nil {
		return framework.Rectangle{}, err
	}
	value, err := device.ScissorRectangle()
	if err != nil {
		return framework.Rectangle{}, err
	}
	return framework.NewRectangle(value.X, value.Y, value.Width, value.Height), nil
}

// SetScissorRectangle is GraphicsDevice::set_ScissorRectangle.
func (d *GraphicsDevice) SetScissorRectangle(value framework.Rectangle) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.SetScissorRectangle(interop.ScissorRectangle{
		X: value.X, Y: value.Y, Width: value.Width, Height: value.Height,
	})
}

// SetViewport is GraphicsDevice::set_Viewport, the setter whose getter has been
// projected since Foundation 1.
func (d *GraphicsDevice) SetViewport(value Viewport) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.SetViewport(interop.Viewport{
		X: value.X(), Y: value.Y(), Width: value.Width(), Height: value.Height(),
		MinDepth: value.MinDepth(), MaxDepth: value.MaxDepth(),
	})
}

// GraphicsProfile is GraphicsDevice::get_GraphicsProfile.
//
// CNA's CNA_GRAPHICS_PROFILE_REACH is 0 and HI_DEF is 1, which are XNA's own
// two values, so the identity crosses unchanged.
func (d *GraphicsDevice) GraphicsProfile() (GraphicsProfile, error) {
	device, err := d.live()
	if err != nil {
		return 0, err
	}
	value, err := device.GraphicsProfile()
	if err != nil {
		return 0, err
	}
	return GraphicsProfile(value), nil
}

// GraphicsDeviceStatus is GraphicsDevice::get_GraphicsDeviceStatus, which in
// the reference calls TestCooperativeLevel and maps three HRESULTs.
//
// CNA's three identities are XNA's three values -- NORMAL 0, LOST 1,
// NOT_RESET 2 -- so the identity crosses unchanged here too.
func (d *GraphicsDevice) GraphicsDeviceStatus() (GraphicsDeviceStatus, error) {
	device, err := d.live()
	if err != nil {
		return 0, err
	}
	value, err := device.Status()
	if err != nil {
		return 0, err
	}
	return GraphicsDeviceStatus(value), nil
}

// IsDisposed is GraphicsDevice::get_IsDisposed.
//
// It reports what CNA'S device says about itself, which is NOT the same
// question as whether this Go facade is still usable: a facade from a previous
// native generation is rejected by the generation check before the call is
// made, and reports that instead.
func (d *GraphicsDevice) IsDisposed() (bool, error) {
	device, err := d.live()
	if err != nil {
		return false, err
	}
	return device.IsDisposed()
}

// ClearByClearOptionsAndColorAndSingleAndInt32 is
// GraphicsDevice::Clear(ClearOptions, Color, Single, Int32) -- the overload the
// other three funnel into.
//
// XNA's three ClearOptions bits are Target 1, DepthBuffer 2 and Stencil 4, and
// CNA's CNA_CLEAR_OPTION_* are the same three values, so the mask crosses
// unchanged.
func (d *GraphicsDevice) ClearByClearOptionsAndColorAndSingleAndInt32(
	options ClearOptions, color framework.Color, depth float32, stencil int32,
) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.ClearWithOptions(uint32(options), color.R(), color.G(), color.B(), color.A(), depth, stencil)
}

// ClearByClearOptionsAndVector4AndSingleAndInt32 is
// GraphicsDevice::Clear(ClearOptions, Vector4, Single, Int32), which is twenty
// bytes of IL:
//
//	Color local = new Color(color);            // the Vector4 constructor
//	this.Clear(options, local, depth, stencil);
//
// so it converts and forwards, and the conversion is the Color constructor's
// own -- not a second rounding rule invented here.
func (d *GraphicsDevice) ClearByClearOptionsAndVector4AndSingleAndInt32(
	options ClearOptions, color framework.Vector4, depth float32, stencil int32,
) error {
	return d.ClearByClearOptionsAndColorAndSingleAndInt32(
		options, framework.NewColorByVector4(color), depth, stencil)
}

// DisplayMode is GraphicsDevice::get_DisplayMode.
//
// The reference reads its adapter's current mode; CNA-Go asks CNA for the
// device's, through cna_graphics_device_get_display_mode, which is the same
// question asked of the object that owns the answer.
//
// The DisplayMode it returns is built from CNA's width, height and format and
// NOT from CNA's aspect ratio: the reference computes that from the two
// dimensions with a defined answer for a zero width, and a second computed
// value could disagree with it. See display_mode.go.
func (d *GraphicsDevice) DisplayMode() (*DisplayMode, error) {
	device, err := d.live()
	if err != nil {
		return nil, err
	}
	mode, err := device.DisplayMode()
	if err != nil {
		return nil, err
	}
	return &DisplayMode{
		width:  mode.Width,
		height: mode.Height,
		format: SurfaceFormat(mode.Format),
	}, nil
}

// PresentByNone is GraphicsDevice::Present(), which is ten bytes of IL forwarding to
// the three-pointer overload with three nulls:
//
//	this.Present((tagRECT*)null, (tagRECT*)null, (HWND__*)null);
//
// The pointer overload is a separate projected member and is still missing: it
// takes two Nullable<Rectangle> and an IntPtr window handle, and CNA exposes no
// route that presents a sub-rectangle to another window.
func (d *GraphicsDevice) PresentByNone() error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.Present()
}

// The two exceptions Texture2D's constructors throw before they reach CNA. They
// live here rather than in foundation.go because they are error SHAPES, and this
// file already holds the package's other one.
var (
	// errGraphicsResourceArgumentNull projects System.ArgumentNullException.
	errGraphicsResourceArgumentNull = errors.New("graphics resource argument is nil")
	// errGraphicsResourceArgument projects System.ArgumentException.
	errGraphicsResourceArgument = errors.New("graphics resource argument is invalid")
	// errGraphicsResourceNil is the Go-only refusal a nil receiver needs. The
	// reference has no counterpart: a CLR instance member cannot be reached
	// through a null reference at all, and the CLR raises NullReferenceException
	// at the call site rather than inside the member.
	errGraphicsResourceNil = errors.New("GraphicsResource is nil")
)

// The exact FrameworkResources string the reference's resource constructors
// load, read by tools/resource_strings out of the retained
// Microsoft.Xna.Framework.dll under the key the IL names.
const deviceCannotBeNullOnResourceCreate = "The GraphicsDevice must not be null when creating new resources."

// SetRenderTargetByRenderTarget2D is
// GraphicsDevice::SetRenderTarget(RenderTarget2D):
//
//	if (renderTarget != null) {
//	    RenderTargetBinding binding = new RenderTargetBinding(renderTarget);
//	    SetRenderTargets(&binding, 1);
//	} else {
//	    SetRenderTargets(null, 0);
//	}
//
// A null target is not a refusal: it is how the back buffer is restored, and
// the reference expresses that as a zero-length binding array. CNA says the
// same thing with `CNA_INVALID_HANDLE`, and interop passes a nil resource
// through as exactly that.
//
// # What a failure here means
//
// CNA permits a render target to be CREATED on a backend with no real
// off-screen storage: creation succeeds, CNA_RenderTargetInfo::renderer_available
// is false, and binding reports CNA_RESULT_NOT_SUPPORTED. That is a renderer
// capability rather than a caller mistake, and it is reported as CNA states it
// rather than translated into one of D3D9's messages.
func (d *GraphicsDevice) SetRenderTargetByRenderTarget2D(renderTarget *RenderTarget2D) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if renderTarget == nil {
		if err := d.device.SetRenderTarget2D(nil); err != nil {
			return err
		}
		d.renderTargets = nil
		return nil
	}
	resource := renderTarget.texture.nativeResource()
	if resource == nil {
		return interop.ErrDisposed
	}
	// The binding the reference builds. Foundation 73 keeps it, because
	// GetRenderTargets must answer the SAME objects that were bound and
	// GetBackBufferData must be able to see that one is active.
	binding, err := NewRenderTargetBindingByRenderTarget2D(renderTarget)
	if err != nil {
		return err
	}
	if err := d.device.SetRenderTarget2D(resource); err != nil {
		return err
	}
	d.renderTargets = []RenderTargetBinding{binding}
	return nil
}
