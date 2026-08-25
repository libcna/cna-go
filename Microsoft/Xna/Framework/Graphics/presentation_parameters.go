package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// PresentationParameters is XNA's pure managed swap-chain descriptor. It is a
// CLR class and keeps CLR reference semantics, and it owns no native object:
// Microsoft.Xna.Framework.Graphics.dll declares exactly one assembly-visible
// field, a nested `Settings` value struct of ten public fields, and every
// public accessor is one ldflda into it plus one ldfld or stfld.
//
// A descriptor is not a device. Nothing here creates, resets, or presents a
// GraphicsDevice; enumerates a display; looks up a window; or reaches SDL or
// CNA. Storing a platform window handle in DeviceWindowHandle does not mean
// the binding knows how to use it, and no member validates that it could.
//
// The stored fields below mirror the reference `Settings` struct one for one.
// The reference stores IsFullScreen as an int32 whose setter normalizes any
// true value to 1 and whose getter reports `field != 0`; a Go bool is exactly
// that two-valued normalization, so no int32 mirror is needed to reproduce it.
type PresentationParameters struct {
	backBufferWidth      int32
	backBufferHeight     int32
	backBufferFormat     SurfaceFormat
	depthStencilFormat   DepthFormat
	multiSampleCount     int32
	displayOrientation   framework.DisplayOrientation
	presentationInterval PresentInterval
	renderTargetUsage    RenderTargetUsage
	deviceWindowHandle   uintptr
	isFullScreen         bool
}

// NewPresentationParameters reproduces PresentationParameters::.ctor, whose
// entire body after the base call is
//
//	IL_0006: ldarg.0
//	IL_0007: ldc.i4.1
//	IL_0008: call PresentationParameters::set_IsFullScreen(bool)
//
// So **IsFullScreen defaults to true** and every other member takes its CLR
// default: both back-buffer extents and MultiSampleCount are 0,
// BackBufferFormat is SurfaceFormat.Color, DepthStencilFormat is
// DepthFormat.None, DisplayOrientation is DisplayOrientation.Default,
// PresentationInterval is PresentInterval.Default, RenderTargetUsage is
// RenderTargetUsage.DiscardContents, and DeviceWindowHandle is IntPtr.Zero.
//
// A full-screen default on a zero-sized back buffer reads oddly, and in
// practice GraphicsDeviceManager overwrites it before any device is created.
// It is nonetheless what the reference constructor does, and it is preserved.
func NewPresentationParameters() *PresentationParameters {
	return &PresentationParameters{isFullScreen: true}
}

// Clone reproduces PresentationParameters::Clone: it constructs a fresh
// instance and then overwrites its whole `settings` value struct with the
// source's. Because Settings is a value type, that single stfld copies every
// member, so the result is fully independent -- including IsFullScreen, whose
// constructor-assigned true is overwritten by the source's value.
//
// The clone is a new reference. Mutating either instance afterwards does not
// affect the other.
func (p *PresentationParameters) Clone() *PresentationParameters {
	clone := NewPresentationParameters()
	*clone = *p
	return clone
}

// BackBufferWidth is one stored descriptor field. The reference validates
// nothing: a zero or negative extent is stored and read back unchanged.
func (p *PresentationParameters) BackBufferWidth() int32 { return p.backBufferWidth }

// SetBackBufferWidth stores the exact value with no validation.
func (p *PresentationParameters) SetBackBufferWidth(value int32) { p.backBufferWidth = value }

// BackBufferHeight is one stored descriptor field.
func (p *PresentationParameters) BackBufferHeight() int32 { return p.backBufferHeight }

// SetBackBufferHeight stores the exact value with no validation.
func (p *PresentationParameters) SetBackBufferHeight(value int32) { p.backBufferHeight = value }

// BackBufferFormat is one stored descriptor field. The reference does not
// check the value against any device capability, so an undefined raw
// SurfaceFormat round-trips unchanged.
func (p *PresentationParameters) BackBufferFormat() SurfaceFormat { return p.backBufferFormat }

// SetBackBufferFormat stores the exact bits with no validation.
func (p *PresentationParameters) SetBackBufferFormat(value SurfaceFormat) {
	p.backBufferFormat = value
}

// DepthStencilFormat is one stored descriptor field.
func (p *PresentationParameters) DepthStencilFormat() DepthFormat { return p.depthStencilFormat }

// SetDepthStencilFormat stores the exact bits with no validation.
func (p *PresentationParameters) SetDepthStencilFormat(value DepthFormat) {
	p.depthStencilFormat = value
}

// MultiSampleCount is one stored descriptor field.
func (p *PresentationParameters) MultiSampleCount() int32 { return p.multiSampleCount }

// SetMultiSampleCount stores the exact value with no validation.
func (p *PresentationParameters) SetMultiSampleCount(value int32) { p.multiSampleCount = value }

// DisplayOrientation is one stored descriptor field.
func (p *PresentationParameters) DisplayOrientation() framework.DisplayOrientation {
	return p.displayOrientation
}

// SetDisplayOrientation stores the exact bits with no validation, so an
// unknown orientation flag combination round-trips unchanged.
func (p *PresentationParameters) SetDisplayOrientation(value framework.DisplayOrientation) {
	p.displayOrientation = value
}

// PresentationInterval is one stored descriptor field.
func (p *PresentationParameters) PresentationInterval() PresentInterval {
	return p.presentationInterval
}

// SetPresentationInterval stores the exact bits with no validation.
func (p *PresentationParameters) SetPresentationInterval(value PresentInterval) {
	p.presentationInterval = value
}

// RenderTargetUsage is one stored descriptor field.
func (p *PresentationParameters) RenderTargetUsage() RenderTargetUsage {
	return p.renderTargetUsage
}

// SetRenderTargetUsage stores the exact bits with no validation.
func (p *PresentationParameters) SetRenderTargetUsage(value RenderTargetUsage) {
	p.renderTargetUsage = value
}

// DeviceWindowHandle is the descriptor's platform window handle, declared
// System.IntPtr by XNA and projected as the pointer-sized bit value it carries.
//
// The value is opaque. It must not be dereferenced, it is not a CNA or SDL
// handle, and a non-zero value here is no evidence that a window exists.
// IntPtr.Zero is uintptr(0). The reference accessors are one ldfld and one
// stfld with no validation whatsoever, so any bit pattern round-trips exactly.
func (p *PresentationParameters) DeviceWindowHandle() uintptr { return p.deviceWindowHandle }

// SetDeviceWindowHandle stores the exact bit pattern with no validation. It
// looks up nothing, opens nothing, and asserts nothing about the value.
func (p *PresentationParameters) SetDeviceWindowHandle(value uintptr) {
	p.deviceWindowHandle = value
}

// IsFullScreen is one stored descriptor field, and the one the constructor
// initializes to true.
func (p *PresentationParameters) IsFullScreen() bool { return p.isFullScreen }

// SetIsFullScreen stores the exact value. It changes no display mode and
// resizes no window; it records an intent that a future device creation would
// read.
func (p *PresentationParameters) SetIsFullScreen(value bool) { p.isFullScreen = value }

// Bounds is a computed read-only projection, not a stored field. The reference
// getter is exactly `new Rectangle(0, 0, BackBufferWidth, BackBufferHeight)`,
// so it tracks the two extents live and reports a degenerate rectangle for a
// freshly constructed descriptor.
func (p *PresentationParameters) Bounds() framework.Rectangle {
	return framework.NewRectangle(0, 0, p.backBufferWidth, p.backBufferHeight)
}
