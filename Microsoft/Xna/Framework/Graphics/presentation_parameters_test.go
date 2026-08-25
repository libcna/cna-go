package graphics

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestPresentationParametersConstructorDefaults pins the exact constructed
// state. The reference constructor's whole body after the base call is
// `this.IsFullScreen = true`, so that is the one non-default member and every
// other one takes its CLR zero.
func TestPresentationParametersConstructorDefaults(t *testing.T) {
	p := NewPresentationParameters()

	if p.BackBufferWidth() != 0 || p.BackBufferHeight() != 0 {
		t.Fatalf("back buffer extents = %d x %d", p.BackBufferWidth(), p.BackBufferHeight())
	}
	if p.BackBufferFormat() != SurfaceFormatColor {
		t.Fatalf("BackBufferFormat = %d, want SurfaceFormat.Color", p.BackBufferFormat())
	}
	if p.DepthStencilFormat() != DepthFormatNone {
		t.Fatalf("DepthStencilFormat = %d, want DepthFormat.None", p.DepthStencilFormat())
	}
	if p.MultiSampleCount() != 0 {
		t.Fatalf("MultiSampleCount = %d", p.MultiSampleCount())
	}
	if p.DisplayOrientation() != framework.DisplayOrientationDefault {
		t.Fatalf("DisplayOrientation = %d", p.DisplayOrientation())
	}
	if p.PresentationInterval() != PresentIntervalDefault {
		t.Fatalf("PresentationInterval = %d", p.PresentationInterval())
	}
	if p.RenderTargetUsage() != RenderTargetUsageDiscardContents {
		t.Fatalf("RenderTargetUsage = %d", p.RenderTargetUsage())
	}
	if p.DeviceWindowHandle() != 0 {
		t.Fatalf("DeviceWindowHandle = %d, want IntPtr.Zero", p.DeviceWindowHandle())
	}
	// The one reference quirk: a fresh descriptor is full-screen.
	if !p.IsFullScreen() {
		t.Fatal("IsFullScreen default = false, want true")
	}
	if got := p.Bounds(); got != (framework.Rectangle{}) {
		t.Fatalf("Bounds default = %+v", got)
	}
}

// TestPresentationParametersStoresWithoutValidating records that no accessor
// validates anything: the reference is a descriptor, so degenerate extents and
// undefined enum bits round-trip exactly.
func TestPresentationParametersStoresWithoutValidating(t *testing.T) {
	p := NewPresentationParameters()

	p.SetBackBufferWidth(-1)
	p.SetBackBufferHeight(math.MinInt32)
	p.SetMultiSampleCount(-7)
	if p.BackBufferWidth() != -1 || p.BackBufferHeight() != math.MinInt32 || p.MultiSampleCount() != -7 {
		t.Fatalf("negative extents = %d %d %d", p.BackBufferWidth(), p.BackBufferHeight(), p.MultiSampleCount())
	}

	p.SetBackBufferFormat(SurfaceFormat(12345))
	p.SetDepthStencilFormat(DepthFormat(-9))
	p.SetDisplayOrientation(framework.DisplayOrientation(1 << 20))
	p.SetPresentationInterval(PresentInterval(999))
	p.SetRenderTargetUsage(RenderTargetUsage(-2))
	if p.BackBufferFormat() != SurfaceFormat(12345) || p.DepthStencilFormat() != DepthFormat(-9) ||
		p.DisplayOrientation() != framework.DisplayOrientation(1<<20) ||
		p.PresentationInterval() != PresentInterval(999) || p.RenderTargetUsage() != RenderTargetUsage(-2) {
		t.Fatal("an undefined enum value did not round-trip unchanged")
	}
}

// TestPresentationParametersBoundsTracksExtents pins that Bounds is computed
// from the two live extents rather than stored, and always originates at zero.
func TestPresentationParametersBoundsTracksExtents(t *testing.T) {
	p := NewPresentationParameters()
	p.SetBackBufferWidth(1280)
	p.SetBackBufferHeight(720)
	if got := p.Bounds(); got != (framework.Rectangle{X: 0, Y: 0, Width: 1280, Height: 720}) {
		t.Fatalf("Bounds = %+v", got)
	}
	p.SetBackBufferWidth(640)
	if got := p.Bounds(); got.Width != 640 || got.Height != 720 {
		t.Fatalf("Bounds did not track the extent change: %+v", got)
	}
	p.SetBackBufferWidth(-5)
	if got := p.Bounds(); got.Width != -5 {
		t.Fatalf("Bounds rejected a degenerate extent: %+v", got)
	}
}

// TestPresentationParametersDeviceWindowHandleIsAnOpaqueBitValue pins the
// System.IntPtr projection: the value is stored and returned verbatim, with no
// validation and no interpretation.
func TestPresentationParametersDeviceWindowHandleIsAnOpaqueBitValue(t *testing.T) {
	p := NewPresentationParameters()
	for _, handle := range []uintptr{
		0,
		1,
		0x7fffffff,
		^uintptr(0),
		^uintptr(0) >> 1,
		0xdeadbeef,
	} {
		p.SetDeviceWindowHandle(handle)
		if got := p.DeviceWindowHandle(); got != handle {
			t.Fatalf("DeviceWindowHandle round-trip = %#x, want %#x", got, handle)
		}
	}
	// IntPtr.Zero is uintptr(0), and assigning it does not "close" anything.
	p.SetDeviceWindowHandle(0)
	if p.DeviceWindowHandle() != 0 {
		t.Fatal("IntPtr.Zero did not round-trip")
	}
}

// TestPresentationParametersCloneIsIndependent pins Clone's reference
// semantics: it copies the whole settings value struct into a fresh instance,
// so the result shares nothing with its source, and the source's IsFullScreen
// overwrites the clone's constructor-assigned true.
func TestPresentationParametersCloneIsIndependent(t *testing.T) {
	source := NewPresentationParameters()
	source.SetBackBufferWidth(800)
	source.SetBackBufferHeight(600)
	source.SetBackBufferFormat(SurfaceFormatBgra5551)
	source.SetDepthStencilFormat(DepthFormatDepth24Stencil8)
	source.SetMultiSampleCount(4)
	source.SetDisplayOrientation(framework.DisplayOrientationPortrait)
	source.SetPresentationInterval(PresentIntervalTwo)
	source.SetRenderTargetUsage(RenderTargetUsagePreserveContents)
	source.SetDeviceWindowHandle(0x1234)
	source.SetIsFullScreen(false)

	clone := source.Clone()
	if clone == source {
		t.Fatal("Clone returned the same reference")
	}
	if *clone != *source {
		t.Fatalf("Clone did not copy every member:\n got %+v\nwant %+v", *clone, *source)
	}
	// The source's false overwrote the clone constructor's true.
	if clone.IsFullScreen() {
		t.Fatal("Clone kept its own constructor default for IsFullScreen")
	}

	clone.SetBackBufferWidth(1)
	clone.SetDeviceWindowHandle(0)
	if source.BackBufferWidth() != 800 || source.DeviceWindowHandle() != 0x1234 {
		t.Fatal("mutating the clone changed the source")
	}
	source.SetMultiSampleCount(8)
	if clone.MultiSampleCount() != 4 {
		t.Fatal("mutating the source changed the clone")
	}
}

// TestPresentationParametersKeepsReferenceSemantics pins that the descriptor
// itself is a CLR reference type: two variables naming one instance observe
// the same mutations, which is what distinguishes it from Clone.
func TestPresentationParametersKeepsReferenceSemantics(t *testing.T) {
	p := NewPresentationParameters()
	alias := p
	alias.SetBackBufferWidth(1920)
	alias.SetIsFullScreen(false)
	if p.BackBufferWidth() != 1920 || p.IsFullScreen() {
		t.Fatalf("aliased mutation was not observed: %d %t", p.BackBufferWidth(), p.IsFullScreen())
	}
	if p != alias {
		t.Fatal("two references to one instance are not identical")
	}
}

// TestPresentationParametersIsFullScreenNormalizes records that the reference
// stores an int32 whose setter normalizes and whose getter reports `!= 0`, a
// round trip a Go bool reproduces exactly.
func TestPresentationParametersIsFullScreenNormalizes(t *testing.T) {
	p := NewPresentationParameters()
	p.SetIsFullScreen(false)
	if p.IsFullScreen() {
		t.Fatal("false did not round-trip")
	}
	p.SetIsFullScreen(true)
	if !p.IsFullScreen() {
		t.Fatal("true did not round-trip")
	}
}
