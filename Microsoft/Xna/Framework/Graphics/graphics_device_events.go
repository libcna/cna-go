package graphics

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// ---------------------------------------------------------------------------
// Foundation 62 — GraphicsDevice's six events and its disposal.
// ---------------------------------------------------------------------------

// # The raise path is NATIVE, and that is the settled rule rather than a choice
//
// All six events have a `raise_*` protected virtual in the reference and every
// raise site is inside the device's own runtime code: the lost/reset detection,
// the graphics-resource base constructor, and Dispose. XNA's device plays a part
// CNA's device plays here, so the canonical CNA signal IS the reference's raise
// path -- the same shape Game's three host-driven events already have, and the
// opposite of Game::Disposed, whose raise site is managed and whose native
// signal is bound LIFECYCLE_ONLY.
//
//	Disposing         CNA_GRAPHICS_DEVICE_EVENT_DISPOSING
//	DeviceLost        CNA_GRAPHICS_DEVICE_EVENT_DEVICE_LOST
//	DeviceReset       CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESET
//	DeviceResetting   CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING
//	ResourceCreated   cna_graphics_device_subscribe_resource_created
//	ResourceDestroyed cna_graphics_device_subscribe_resource_destroyed
//
// # Subscription is lazy, and released with the facade's generation
//
// A device that nobody listens to installs nothing. The first Add* call
// subscribes all six at once -- CNA's registrations are per event and CNA-Go
// takes them together so a partial failure cannot leave some installed -- and
// the registrations are released when the facade's generation ends.
//
// # What the two payload events can and cannot report
//
// ResourceCreatedEventArgs::Resource is `System.Object`, and CNA reports only
// that a resource was supplied:
//
//	The canonical event is raised from the graphics-resource base constructor,
//	so the reported object is still under construction: its concrete type does
//	not exist yet and no member of it can be queried.
//
// so the projected args carry nil. ResourceDestroyedEventArgs::Name is real --
// CNA hands over callback-scoped UTF-8 which the trampoline copies before it
// expires -- and its Tag is caller-owned native state CNA reports as presence
// only, so that carries nil too. Both are divergences from the reference and
// both are recorded rather than filled with an invention.

func init() {
	// The Graphics package's half of the facade-signal seam. The framework
	// package's GraphicsDeviceManager.Dispose calls it through the bridge, so
	// the registrations CNA requires released before cna_game_destroy really
	// are -- with no public API on either side.
	servicebridge.SetDeviceFacadeSignalReleaser(func(facade any) error {
		device, typed := facade.(*GraphicsDevice)
		if !typed || device == nil {
			return nil
		}
		return device.releaseDeviceSignals()
	})
}

// deviceEvents is the six registration lists and the native subscription behind
// them, held on the facade because the facade is what a consumer registers on.
type deviceEvents struct {
	disposing         framework.EventSource[*framework.EventArgs]
	deviceLost        framework.EventSource[*framework.EventArgs]
	deviceReset       framework.EventSource[*framework.EventArgs]
	deviceResetting   framework.EventSource[*framework.EventArgs]
	resourceCreated   framework.EventSource[*ResourceCreatedEventArgs]
	resourceDestroyed framework.EventSource[*ResourceDestroyedEventArgs]
	signals           *interop.DeviceSignals
}

// ensureDeviceSignals installs the native subscriptions on first use.
func (d *GraphicsDevice) ensureDeviceSignals() error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if d.events.signals != nil {
		return nil
	}
	signals, err := interop.SubscribeDeviceEvents(d.device, d.deliverDeviceSignal)
	if err != nil {
		return err
	}
	d.events.signals = signals
	return nil
}

// deliverDeviceSignal is the one sink every canonical device signal reaches. It
// runs on the owner thread inside a native callback, so it raises and returns:
// a handler failure is the runtime's to record, and a panic is contained by the
// trampoline above it.
func (d *GraphicsDevice) deliverDeviceSignal(event uint32, payload interop.DeviceSignalPayload) error {
	switch event {
	case interop.DeviceEventDisposing:
		return d.events.disposing.Raise(d, framework.EventArgsEmpty())
	case interop.DeviceEventDeviceLost:
		return d.events.deviceLost.Raise(d, framework.EventArgsEmpty())
	case interop.DeviceEventDeviceReset:
		return d.events.deviceReset.Raise(d, framework.EventArgsEmpty())
	case interop.DeviceEventDeviceResetting:
		return d.events.deviceResetting.Raise(d, framework.EventArgsEmpty())
	case interop.DeviceEventResourceCreated:
		// nil, and the reason is CNA's own: the object does not exist yet.
		return d.events.resourceCreated.Raise(d, newResourceCreatedEventArgs(nil))
	case interop.DeviceEventResourceDestroyed:
		// The name survives; the tag is caller-owned native state CNA reports
		// as presence only.
		return d.events.resourceDestroyed.Raise(d, newResourceDestroyedEventArgs(payload.Name, nil))
	}
	return nil
}

// releaseDeviceSignals releases every native registration. It is called when a
// facade's generation ends and is idempotent.
func (d *GraphicsDevice) releaseDeviceSignals() error {
	if d == nil || d.events.signals == nil {
		return nil
	}
	signals := d.events.signals
	d.events.signals = nil
	return signals.Release()
}

// The six events, on the settled two-accessor projection. Each Add installs the
// native subscriptions if they are not installed yet, which is why an Add is
// fallible beyond the registration list itself.

func (d *GraphicsDevice) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.disposing.Add(handler)
}

func (d *GraphicsDevice) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.disposing.Remove(subscription)
}

func (d *GraphicsDevice) AddDeviceLostHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.deviceLost.Add(handler)
}

func (d *GraphicsDevice) RemoveDeviceLostHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.deviceLost.Remove(subscription)
}

func (d *GraphicsDevice) AddDeviceResetHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.deviceReset.Add(handler)
}

func (d *GraphicsDevice) RemoveDeviceResetHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.deviceReset.Remove(subscription)
}

func (d *GraphicsDevice) AddDeviceResettingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.deviceResetting.Add(handler)
}

func (d *GraphicsDevice) RemoveDeviceResettingHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.deviceResetting.Remove(subscription)
}

func (d *GraphicsDevice) AddResourceCreatedHandler(handler framework.EventHandler[*ResourceCreatedEventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.resourceCreated.Add(handler)
}

func (d *GraphicsDevice) RemoveResourceCreatedHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.resourceCreated.Remove(subscription)
}

func (d *GraphicsDevice) AddResourceDestroyedHandler(handler framework.EventHandler[*ResourceDestroyedEventArgs]) (framework.EventSubscription, error) {
	if err := d.ensureDeviceSignals(); err != nil {
		return framework.EventSubscription{}, err
	}
	return d.events.resourceDestroyed.Add(handler)
}

func (d *GraphicsDevice) RemoveResourceDestroyedHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.events.resourceDestroyed.Remove(subscription)
}

// DisposeByNone is GraphicsDevice::Dispose(), the sealed IDisposable member:
//
//	Dispose(true);
//	GC.SuppressFinalize(this);
//
// # It really disposes the device the Game owns
//
// That is the reference's behaviour and it is reproduced. The facade's ownership
// is still BORROWED -- CNA-Go never disposes a device on its own, and the one
// thing the borrowed-device rule forbids is retaining or releasing the
// callback-scoped handle without being asked. A consumer who calls this has
// asked, exactly as a consumer of the reference has.
func (d *GraphicsDevice) DisposeByNone() error {
	return d.DisposeByBoolean(true)
}

// DisposeByBoolean is GraphicsDevice::Dispose(bool). CNA's own
// cna_graphics_device_dispose is the counterpart of the reference's disposing
// branch: it disposes the device and raises the canonical Disposing signal,
// which reaches the projected event through the same subscription every other
// device signal uses.
//
// The false branch is the finalizer's, which Go never takes on its own, and the
// reference's own `Dispose(false)` releases the unmanaged half without raising.
// CNA offers one disposal route and no unmanaged-only variant, so the false
// branch is a no-op here and says so rather than calling the same route twice.
func (d *GraphicsDevice) DisposeByBoolean(disposing bool) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if !disposing {
		return nil
	}
	if err := d.device.DisposeDevice(); err != nil {
		return err
	}
	return d.releaseDeviceSignals()
}

// Finalize is GraphicsDevice::Finalize, `Dispose(false)`. Nothing calls it on
// its own: Go has no CLR finalization and CNA-Go registers no runtime finalizer.
func (d *GraphicsDevice) Finalize() error {
	return d.DisposeByBoolean(false)
}
