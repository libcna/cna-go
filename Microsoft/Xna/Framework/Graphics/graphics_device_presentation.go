package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 73 — Reset, PresentationParameters and GetBackBufferData.
// ---------------------------------------------------------------------------

// cannotGetBackBufferActiveRenderTargets is the one FrameworkResources string
// GetBackBufferData throws that no other member claims.
const cannotGetBackBufferActiveRenderTargets = "Cannot use GetBackBufferData when a render target is active."

// PresentationParameters is GraphicsDevice::get_PresentationParameters:
//
//	ldarg.0; ldfld pPublicCachedParams; ret
//
// one field read over a PUBLIC cached copy the reference's constructor and its
// Reset fill -- a DIFFERENT object from the `pInternalCachedParams` Reset()
// re-applies.
//
// # Why it is fallible here and infallible there
//
// The reference caches both copies when ITS constructor creates the D3D device.
// CNA-Go does not create the device: the Game's manager does, natively, and
// this facade borrows it. A managed cache filled at facade construction would
// start at Go zero values and disagree with the live device until something
// wrote it, so this asks CNA -- the same decision, on the same evidence, that
// Foundation 51 made for the fifteen render-state members.
//
// The value is a FRESH PresentationParameters every call, because CNA reports a
// value rather than an object. The reference returns the same object every
// time, so a caller who kept and mutated it would see their change on the next
// read and here would not. That is recorded rather than papered over -- and
// mutating that object does not reach the device in the reference either, so
// the observable difference is object identity and nothing else.
func (d *GraphicsDevice) PresentationParameters() (*PresentationParameters, error) {
	device, err := d.live()
	if err != nil {
		return nil, err
	}
	value, err := device.PresentationParameters()
	if err != nil {
		return nil, err
	}
	return presentationParametersFromInterop(value), nil
}

// presentationParametersFromInterop is the one place CNA's flattened value
// becomes the projected object.
//
// Two fields have no counterpart on the other side. CNA carries a
// `headless_ext` flag XNA has nothing for, and XNA carries a DeviceWindowHandle
// CNA's presentation value does not -- CNA owns the window. So the projected
// object's DeviceWindowHandle stays IntPtr.Zero, which is what a Game-created
// device reports in the reference before a host assigns one.
func presentationParametersFromInterop(value interop.PresentationValue) *PresentationParameters {
	return &PresentationParameters{
		backBufferWidth:      value.BackBufferWidth,
		backBufferHeight:     value.BackBufferHeight,
		backBufferFormat:     SurfaceFormat(value.BackBufferFormat),
		depthStencilFormat:   DepthFormat(value.DepthStencilFormat),
		multiSampleCount:     value.MultiSampleCount,
		displayOrientation:   framework.DisplayOrientation(value.DisplayOrientation),
		presentationInterval: PresentInterval(value.PresentationInterval),
		renderTargetUsage:    RenderTargetUsage(value.RenderTargetUsage),
		isFullScreen:         value.IsFullScreen,
	}
}

// presentationParametersToInterop is its inverse. `headless_ext` is left false:
// it is CNA's own extension and XNA has nothing that could set it, so a Reset
// through this projection never asks for a headless device.
func presentationParametersToInterop(value *PresentationParameters) interop.PresentationValue {
	return interop.PresentationValue{
		BackBufferFormat:     int32(value.BackBufferFormat()),
		BackBufferWidth:      value.BackBufferWidth(),
		BackBufferHeight:     value.BackBufferHeight(),
		DepthStencilFormat:   int32(value.DepthStencilFormat()),
		MultiSampleCount:     value.MultiSampleCount(),
		PresentationInterval: int32(value.PresentationInterval()),
		DisplayOrientation:   int32(value.DisplayOrientation()),
		RenderTargetUsage:    int32(value.RenderTargetUsage()),
		IsFullScreen:         value.IsFullScreen(),
	}
}

// ResetByNone is GraphicsDevice::Reset():
//
//	Reset(pInternalCachedParams, pCurrentAdapter);
//
// nineteen bytes that forward the device's OWN cached parameters and its
// current adapter, so it re-applies whatever is already in effect.
//
// CNA has a route for exactly that shape -- cna_graphics_device_reset takes no
// parameters and re-applies the device's own -- so this reaches it rather than
// reading the parameters back and handing them to the wider route. The
// difference is real: reading them back first would apply what CNA REPORTS, and
// the reference applies what it CACHED.
func (d *GraphicsDevice) ResetByNone() error {
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.ResetDevice()
}

// ResetByPresentationParameters is
// GraphicsDevice::Reset(PresentationParameters):
//
//	Reset(presentationParameters, pCurrentAdapter);
//
// fourteen bytes forwarding the caller's parameters and the CURRENT adapter --
// which CNA expresses as a null adapter index, documented as "keep the current
// adapter".
func (d *GraphicsDevice) ResetByPresentationParameters(presentationParameters *PresentationParameters) error {
	return d.reset(presentationParameters, nil)
}

// ResetByPresentationParametersAndGraphicsAdapter is
// GraphicsDevice::Reset(PresentationParameters, GraphicsAdapter), the overload
// the other two funnel into.
//
// Its first two statements are the guards, in this order:
//
//	if (presentationParameters == null)
//	    throw new ArgumentNullException("presentationParameters", NullNotAllowed);
//	if (graphicsAdapter == null)
//	    throw new ArgumentNullException("graphicsAdapter", NullNotAllowed);
//
// Everything after them -- the DeviceResetting and DeviceReset events, the
// resource lifecycle, the D3D reset itself -- belongs to the device's own
// runtime, and CNA's reset performs it. CNA-Go does NOT raise the two events
// itself: the settled event rule requires a projected XNA event to have its
// AUTHORITATIVE raise path, and the reference's is inside the device runtime
// CNA now is. Both canonical CNA signals are already bound and routed by
// Foundation 62; whether a reset raises them is CNA's to answer, and the
// qualified artifacts were measured never raising them.
func (d *GraphicsDevice) ResetByPresentationParametersAndGraphicsAdapter(
	presentationParameters *PresentationParameters, graphicsAdapter *GraphicsAdapter,
) error {
	if presentationParameters == nil {
		return fmt.Errorf("%w: presentationParameters: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if graphicsAdapter == nil {
		return fmt.Errorf("%w: graphicsAdapter: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	index := uint32(graphicsAdapter.index)
	return d.reset(presentationParameters, &index)
}

func (d *GraphicsDevice) reset(presentationParameters *PresentationParameters, adapterIndex *uint32) error {
	if presentationParameters == nil {
		return fmt.Errorf("%w: presentationParameters: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	device, err := d.live()
	if err != nil {
		return err
	}
	return device.ResetDeviceWithParameters(presentationParametersToInterop(presentationParameters), adapterIndex)
}

// PresentByNullableOfRectangleAndNullableOfRectangleAndIntPtr is
// GraphicsDevice::Present(Nullable<Rectangle>, Nullable<Rectangle>, IntPtr) --
// the overload that presents a SUB-RECTANGLE of the back buffer into a
// sub-rectangle of a window that is not the device's own.
//
// # It is BLOCKED_UPSTREAM, and it says so
//
// CNA's only present route is `cna_graphics_device_present(handle)`. It takes
// no rectangles and no window handle, and there is no second one: the whole C
// ABI has exactly this route. Calling it for this overload would present the
// WHOLE back buffer into the DEVICE's own window and silently discard all three
// arguments, which is a different operation wearing this one's name.
//
// So the member exists, is reachable, and refuses with a message naming the
// route and what it lacks. `Present()` itself is projected and works; it is
// `PresentByNone` and has been since Foundation 51.
func (d *GraphicsDevice) PresentByNullableOfRectangleAndNullableOfRectangleAndIntPtr(
	sourceRectangle *framework.Rectangle, destinationRectangle *framework.Rectangle,
	overrideWindowHandle uintptr,
) error {
	if _, err := d.live(); err != nil {
		return err
	}
	return errPresentRectangles
}

// errPresentRectangles records the one member here CNA cannot express, in its
// own words rather than as a borrowed XNA message.
var errPresentRectangles = errors.New(
	"CNA's C ABI has one present route, cna_graphics_device_present, which takes no source rectangle, no destination rectangle and no window handle: presenting the whole back buffer into the device's own window would be a different operation under this overload's name")

// GraphicsDeviceGetBackBufferDataBySliceOfT is
// GraphicsDevice::GetBackBufferData<T>(T[]):
//
//	GetBackBufferData(null, data, 0, data == null ? 0 : data.Length)
//
// a package function on the settled generic-method rule.
func GraphicsDeviceGetBackBufferDataBySliceOfT[T any](device *GraphicsDevice, data []T) error {
	return GraphicsDeviceGetBackBufferDataByNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		device, nil, data, 0, int32(len(data)))
}

// GraphicsDeviceGetBackBufferDataBySliceOfTAndInt32AndInt32 is
// GraphicsDevice::GetBackBufferData<T>(T[], Int32, Int32).
func GraphicsDeviceGetBackBufferDataBySliceOfTAndInt32AndInt32[T any](
	device *GraphicsDevice, data []T, startIndex, elementCount int32,
) error {
	return GraphicsDeviceGetBackBufferDataByNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		device, nil, data, startIndex, elementCount)
}

// GraphicsDeviceGetBackBufferDataByNullableOfRectangleAndSliceOfTAndInt32AndInt32
// is GraphicsDevice::GetBackBufferData<T>(Nullable<Rectangle>, T[], Int32,
// Int32) -- the overload the other two funnel into.
//
// # Two guards are the reference's own and one is not reproduced
//
//	if (!_profileCapabilities.GetBackBufferData)
//	    ThrowNotSupportedException(ProfileFeatureNotSupported, "GetBackBufferData");
//	if (a render target is active)
//	    throw new InvalidOperationException(CannotGetBackBufferActiveRenderTargets);
//
// The second is reproduced, because the projection holds the render target it
// bound and can see one. The first is not, for the reason every other
// ProfileCapabilities check here is not: it is not public XNA surface and
// CNA-Go models no part of it.
//
// # The element set is ONE type wide, and it is CNA's limit
//
// `cna_graphics_device_get_backbuffer_data_window` takes `CNA_Color*` and has
// no data-type parameter, exactly as the cube and volume transfers do. The
// reference accepts any `valuetype .ctor T` whose size divides the back-buffer
// format's. So a T outside framework.Color is refused BY NAME with a message
// that says whose limit it is -- the same narrowing Foundation 71 recorded, and
// it must not be reworded into an XNA restriction.
func GraphicsDeviceGetBackBufferDataByNullableOfRectangleAndSliceOfTAndInt32AndInt32[T any](
	device *GraphicsDevice, rect *framework.Rectangle, data []T, startIndex, elementCount int32,
) error {
	native, err := device.live()
	if err != nil {
		return err
	}
	if err := checkNoActiveRenderTarget(device.renderTargets); err != nil {
		return err
	}
	if err := resolveVolumeElement[T]("GraphicsDevice.GetBackBufferData"); err != nil {
		return err
	}
	if err := checkTransferWindow(len(data), startIndex, elementCount); err != nil {
		return err
	}
	var x, y, width, height int32
	if rect != nil {
		x, y, width, height = rect.X, rect.Y, rect.Width, rect.Height
	}
	return native.BackBufferData(rect != nil, x, y, width, height,
		uint64(startIndex), uint64(elementCount), sliceStart(data), uint64(len(data)))
}

// checkNoActiveRenderTarget is GetBackBufferData's second guard, alone.
//
// It is a free function for the reason Foundation 72's verifyUserPrimitives is:
// the guard sits BEHIND Helpers.CheckDisposed, so a test that only has a device
// with no native half can never reach it through the exported member. Extracting
// it lets the refusal's message and its exception kind be pinned without a
// renderer, and leaves the exported member's ORDER pinned by the coverage
// control that calls it.
func checkNoActiveRenderTarget(bindings []RenderTargetBinding) error {
	if len(bindings) != 0 {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, cannotGetBackBufferActiveRenderTargets)
	}
	return nil
}

// NewGraphicsDevice is
// GraphicsDevice::.ctor(GraphicsAdapter, GraphicsProfile, PresentationParameters),
// the type's ONE public constructor and the profile's only way to hold a
// graphics device a consumer owns.
//
// Its guards, in the reference's order:
//
//	if (adapter == null)
//	    throw new ArgumentNullException("adapter", FrameworkResources.NullNotAllowed);
//	if (presentationParameters == null)
//	    throw new ArgumentNullException("presentationParameters", NullNotAllowed);
//	if (!adapter.IsProfileSupported(graphicsProfile))
//	    ThrowNotSupportedException(ProfileNotSupported, ...);   // NOT reproduced
//	CreateDevice(adapter, graphicsProfile, presentationParameters);
//
// # Ownership, and the one place this type has two kinds
//
//	OWNED, released with cna_graphics_device_destroy.
//
// Every GraphicsDevice a consumer could reach before this was BORROWED: the
// Game's manager creates one natively and publishes a facade over it. This
// constructor creates one CNA owns nothing of, and CNA tells the two apart
// itself -- `cna_graphics_device_destroy` accepts only a caller-created handle
// and refuses a Game's, which the Foundation 73 probe confirmed in both
// directions.
//
// The profile check is not reproduced, for the reason every other
// ProfileCapabilities check here is not: that table is not public XNA surface.
// `GraphicsAdapter.IsProfileSupported` IS projected, so a consumer can perform
// the check the reference performs; CNA refuses a profile its backend cannot
// provide, in its own words.
func NewGraphicsDevice(
	adapter *GraphicsAdapter, graphicsProfile GraphicsProfile,
	presentationParameters *PresentationParameters,
) (*GraphicsDevice, error) {
	if adapter == nil {
		return nil, fmt.Errorf("%w: adapter: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if presentationParameters == nil {
		return nil, fmt.Errorf("%w: presentationParameters: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	runtime, live := interop.CurrentRuntime()
	if !live {
		// A CNA-Go architecture fact rather than a CNA one, named as such.
		// cna_graphics_device_create is a process-level call -- the Foundation
		// 73 probe made one with no Game at all -- but CNA-Go opens the native
		// library for the duration of a Game's SESSION and closes it when the
		// session ends. A device created outside one would hold a handle into a
		// library that is about to be unloaded.
		return nil, errDeviceNeedsSession
	}
	device, err := runtime.CreateOwnedDevice(
		uint32(adapter.index), uint32(graphicsProfile),
		presentationParametersToInterop(presentationParameters))
	if err != nil {
		return nil, err
	}
	return &GraphicsDevice{device: device}, nil
}

// errDeviceNeedsSession records the one narrowing this constructor has, in
// CNA-Go's own words. It is the same shape GraphicsAdapter's statics carry, and
// for a related reason: the adapter a caller must pass here is itself only
// obtainable from inside a lifecycle callback.
var errDeviceNeedsSession = errors.New(
	"a caller-created GraphicsDevice needs a live native session: CNA-Go opens the CNA library for the duration of a Game's session and closes it at the end, so a device created outside one would hold a handle into an unloaded library")
