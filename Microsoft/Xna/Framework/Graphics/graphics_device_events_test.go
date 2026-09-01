package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// TestTheDeviceFacadeSignalReleaserIsInstalled pins the seam the registrations
// are released across.
//
// CNA's contract is explicit -- "A registration is a C-owned resource of the
// active game. It must be released with cna_graphics_device_unsubscribe before
// cna_game_destroy succeeds" -- and the object whose disposal ends a facade's
// life, the GraphicsDeviceManager, lives in a package that cannot name a
// GraphicsDevice. The release therefore crosses internal/servicebridge, and a
// releaser that was never installed would leave every registration behind with
// nothing reporting it.
func TestTheDeviceFacadeSignalReleaserIsInstalled(t *testing.T) {
	// A facade that installed nothing releases nothing and reports nothing.
	if err := servicebridge.ReleaseDeviceFacadeSignals(&GraphicsDevice{}); err != nil {
		t.Fatalf("releasing a facade with no subscription = %v, want nil", err)
	}
	// A nil facade, and a value that is not one, are both successful no-ops:
	// the bridge is untyped and the releaser is what decides.
	if err := servicebridge.ReleaseDeviceFacadeSignals(nil); err != nil {
		t.Fatalf("releasing a nil facade = %v", err)
	}
	if err := servicebridge.ReleaseDeviceFacadeSignals("not a facade"); err != nil {
		t.Fatalf("releasing a non-facade = %v", err)
	}
	// The distinguishing case: a facade that HAS a signals value. A bridge with
	// no releaser installed answers nil for it too, and leaves the value in
	// place; the installed one clears it. That is the observable a managed test
	// can reach without a live device.
	device := &GraphicsDevice{}
	device.events.signals = &interop.DeviceSignals{}
	if err := servicebridge.ReleaseDeviceFacadeSignals(device); err != nil {
		t.Fatalf("releasing through the bridge = %v", err)
	}
	if device.events.signals != nil {
		t.Fatal("the bridge did not reach the Graphics package's releaser; the registrations would outlive the game")
	}
	// And releasing again is a no-op, because a second release would hand CNA a
	// stale registration and CNA answers that with CNA_RESULT_INVALID_HANDLE.
	if err := servicebridge.ReleaseDeviceFacadeSignals(device); err != nil {
		t.Fatalf("second release = %v", err)
	}
}

// TestDeviceEventAccessorsRefuseANilFacade pins the Go-only guard every member
// of a facade carries.
func TestDeviceEventAccessorsRefuseANilFacade(t *testing.T) {
	var device *GraphicsDevice
	handler := func(sender any, args *framework.EventArgs) error { return nil }
	for name, add := range map[string]func() error{
		"Disposing":       func() error { _, err := device.AddDisposingHandler(handler); return err },
		"DeviceLost":      func() error { _, err := device.AddDeviceLostHandler(handler); return err },
		"DeviceReset":     func() error { _, err := device.AddDeviceResetHandler(handler); return err },
		"DeviceResetting": func() error { _, err := device.AddDeviceResettingHandler(handler); return err },
		"ResourceCreated": func() error {
			_, err := device.AddResourceCreatedHandler(func(any, *ResourceCreatedEventArgs) error { return nil })
			return err
		},
		"ResourceDestroyed": func() error {
			_, err := device.AddResourceDestroyedHandler(func(any, *ResourceDestroyedEventArgs) error { return nil })
			return err
		},
	} {
		if err := add(); err == nil {
			t.Errorf("Add%sHandler on a nil facade was accepted", name)
		}
	}
	if err := device.DisposeByNone(); err == nil {
		t.Error("Dispose on a nil facade was accepted")
	}
	// Finalize is Dispose(false), whose branch returns before it reaches the
	// device -- so it answers the nil guard, not the disposal.
	if err := device.Finalize(); err == nil {
		t.Error("Finalize on a nil facade was accepted")
	}
}
