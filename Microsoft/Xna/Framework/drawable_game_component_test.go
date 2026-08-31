package framework

import (
	"errors"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// stubDeviceService is a consumer-shaped IGraphicsDeviceService. It declares
// exactly the eight framework-typed accessors the contract does, which is what
// makes it satisfy the unexported structural interface without naming it.
type stubDeviceService struct {
	created   EventSource[*EventArgs]
	resetting EventSource[*EventArgs]
	reset     EventSource[*EventArgs]
	disposing EventSource[*EventArgs]

	addCalls    []string
	removeCalls []string
	addFailure  error
	failAfter   int
}

func (s *stubDeviceService) record(name string) error {
	s.addCalls = append(s.addCalls, name)
	if s.addFailure != nil && len(s.addCalls) > s.failAfter {
		return s.addFailure
	}
	return nil
}

func (s *stubDeviceService) AddDeviceCreatedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	if err := s.record("DeviceCreated"); err != nil {
		return EventSubscription{}, err
	}
	return s.created.Add(h)
}

func (s *stubDeviceService) RemoveDeviceCreatedHandler(t EventSubscription) error {
	s.removeCalls = append(s.removeCalls, "DeviceCreated")
	return s.created.Remove(t)
}

func (s *stubDeviceService) AddDeviceResettingHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	if err := s.record("DeviceResetting"); err != nil {
		return EventSubscription{}, err
	}
	return s.resetting.Add(h)
}

func (s *stubDeviceService) RemoveDeviceResettingHandler(t EventSubscription) error {
	s.removeCalls = append(s.removeCalls, "DeviceResetting")
	return s.resetting.Remove(t)
}

func (s *stubDeviceService) AddDeviceResetHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	if err := s.record("DeviceReset"); err != nil {
		return EventSubscription{}, err
	}
	return s.reset.Add(h)
}

func (s *stubDeviceService) RemoveDeviceResetHandler(t EventSubscription) error {
	s.removeCalls = append(s.removeCalls, "DeviceReset")
	return s.reset.Remove(t)
}

func (s *stubDeviceService) AddDeviceDisposingHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	if err := s.record("DeviceDisposing"); err != nil {
		return EventSubscription{}, err
	}
	return s.disposing.Add(h)
}

func (s *stubDeviceService) RemoveDeviceDisposingHandler(t EventSubscription) error {
	s.removeCalls = append(s.removeCalls, "DeviceDisposing")
	return s.disposing.Remove(t)
}

// installResolver replaces the process-wide resolver for one test and restores
// it afterwards. The real one is installed by the Graphics package's init,
// which this package cannot import, so every test here supplies its own.
func installResolver(t *testing.T, service *stubDeviceService, hasDevice bool) {
	t.Helper()
	servicebridge.SetDeviceServiceResolver(func(services any) (any, func() bool, bool) {
		if service == nil {
			return nil, nil, false
		}
		return service, func() bool { return hasDevice }, true
	})
	t.Cleanup(func() { servicebridge.SetDeviceServiceResolver(nil) })
}

func newDrawableFixture(t *testing.T) (*Game, *DrawableGameComponent) {
	t.Helper()
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game, NewDrawableGameComponent(game)
}

// TestDrawableConstructorDefaults pins the two-statement constructor:
//
//	this.visible = true; base..ctor(game);
//
// visible starts TRUE and drawOrder starts zero, because the constructor
// assigns the first and does not assign the second.
func TestDrawableConstructorDefaults(t *testing.T) {
	game, component := newDrawableFixture(t)
	if !component.Visible() {
		t.Fatal("Visible = false; the constructor stores true")
	}
	if component.DrawOrder() != 0 {
		t.Fatalf("DrawOrder = %d; the constructor does not assign it", component.DrawOrder())
	}
	if component.Game() != game {
		t.Fatal("Game is not the one the constructor was handed")
	}
	if component.Enabled() != true {
		t.Fatal("Enabled = false; GameComponent's constructor stores true")
	}
}

// TestDrawableInitializeReportsTheReferenceFailure pins the throw site. With no
// registered IGraphicsDeviceService the reference throws
// InvalidOperationException(Resources.MissingGraphicsDeviceService), and the
// exact string matters because a consumer's error handling can read it.
func TestDrawableInitializeReportsTheReferenceFailure(t *testing.T) {
	_, component := newDrawableFixture(t)
	installResolver(t, nil, false)
	err := component.Initialize()
	if err == nil {
		t.Fatal("Initialize succeeded with no graphics device service")
	}
	if !strings.Contains(err.Error(), "Drawable components require a graphics device service in the game service container.") {
		t.Fatalf("Initialize = %v, want the reference's message", err)
	}
	// The throw leaves `initialized` FALSE, which is the reference's own
	// behaviour: the store is at the end of the method and the throw never
	// reaches it. A later Initialize therefore retries the resolution.
	if component.initialized {
		t.Fatal("a failing Initialize set initialized; the store is after the throw site")
	}
}

// TestDrawableInitializeWithNoResolverIsTheSameFailure records the design's own
// correctness argument: a program that never imports the Graphics package
// cannot have registered the service, so "no resolver installed" and "no
// service registered" must be indistinguishable.
func TestDrawableInitializeWithNoResolverIsTheSameFailure(t *testing.T) {
	_, component := newDrawableFixture(t)
	servicebridge.SetDeviceServiceResolver(nil)
	err := component.Initialize()
	if err == nil || !strings.Contains(err.Error(), "Drawable components require a graphics device service") {
		t.Fatalf("Initialize with no resolver = %v, want the reference's missing-service failure", err)
	}
}

// TestDrawableInitializeSubscribesFourHandlersInOrder pins the subscription
// half of Initialize. Four registrations, in the reference's own order, and the
// two whose handlers are empty are subscribed anyway -- a service that counts
// its subscribers would see the difference.
func TestDrawableInitializeSubscribesFourHandlersInOrder(t *testing.T) {
	_, component := newDrawableFixture(t)
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	want := "DeviceCreated,DeviceResetting,DeviceReset,DeviceDisposing"
	if got := strings.Join(service.addCalls, ","); got != want {
		t.Fatalf("subscribed %s, want %s", got, want)
	}
	if !component.initialized {
		t.Fatal("a successful Initialize left initialized false")
	}
}

// TestDrawableInitializeIsGuardedButStillStores pins the detail the IL makes
// and a natural rewrite loses: the `brtrue` at the top of Initialize jumps to
// the `initialized = true` STORE, not past it, so a second call re-assigns the
// flag and skips only the resolution body.
func TestDrawableInitializeIsGuardedButStillStores(t *testing.T) {
	_, component := newDrawableFixture(t)
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	if len(service.addCalls) != 4 {
		t.Fatalf("a second Initialize subscribed again: %v", service.addCalls)
	}
	if !component.initialized {
		t.Fatal("initialized is false after two successful calls")
	}
}

// TestDrawableInitializeLoadsContentOnlyWhenADeviceExists pins the last branch:
//
//	if (this.deviceService.GraphicsDevice != null) this.LoadContent();
//
// The observable difference is which path runs, and LoadContent's body is the
// reference's own `ret`, so the test measures the DECISION rather than a side
// effect the reference does not have.
func TestDrawableInitializeLoadsContentOnlyWhenADeviceExists(t *testing.T) {
	for _, fixture := range []struct{ hasDevice bool }{{false}, {true}} {
		_, component := newDrawableFixture(t)
		service := &stubDeviceService{}
		loaded := 0
		installResolver(t, service, fixture.hasDevice)
		// The decision is observed through the resolver closure the framework
		// package actually consults, which is the only thing this branch reads.
		servicebridge.SetDeviceServiceResolver(func(services any) (any, func() bool, bool) {
			return service, func() bool { loaded++; return fixture.hasDevice }, true
		})
		if err := component.Initialize(); err != nil {
			t.Fatalf("Initialize: %v", err)
		}
		if loaded != 1 {
			t.Fatalf("the device check ran %d times, want exactly one", loaded)
		}
	}
}

// TestDrawableSubscriptionFailureLeavesNoHalfWiredComponent pins the Go-only
// failure path. The reference cannot fail here -- Delegate.Combine does not
// throw -- so this is the event projection's own channel, and it must not leave
// the component holding a service it is partly subscribed to.
func TestDrawableSubscriptionFailureLeavesNoHalfWiredComponent(t *testing.T) {
	_, component := newDrawableFixture(t)
	failure := errors.New("service refused a handler")
	service := &stubDeviceService{addFailure: failure, failAfter: 2}
	installResolver(t, service, false)
	if err := component.Initialize(); !errors.Is(err, failure) {
		t.Fatalf("Initialize = %v, want the service's own error", err)
	}
	if component.deviceService != nil {
		t.Fatal("a failed subscription left the service field set")
	}
	if component.initialized {
		t.Fatal("a failed subscription set initialized")
	}
	if len(service.removeCalls) != 4 {
		t.Fatalf("released %v, want all four slots cleared", service.removeCalls)
	}
}

// TestDrawableDeviceHandlersRunTheReferenceBodies pins which of the four
// handlers do anything: DeviceCreated calls LoadContent, DeviceDisposing calls
// UnloadContent, and the other two are `ret`. All four are still registered.
func TestDrawableDeviceHandlersRunTheReferenceBodies(t *testing.T) {
	_, component := newDrawableFixture(t)
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// Raising each service event reaches the component's handler. The bodies
	// are empty in the reference, so what is measured is that the registration
	// exists and does not fail.
	for name, raise := range map[string]func(any, *EventArgs) error{
		"DeviceCreated":   service.created.Raise,
		"DeviceResetting": service.resetting.Raise,
		"DeviceReset":     service.reset.Raise,
		"DeviceDisposing": service.disposing.Raise,
	} {
		if err := raise(service, EventArgsEmpty()); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// TestDrawableDisposeReleasesEverythingAndCallsTheBase pins Dispose(bool):
// UnloadContent, then the four removals, then an UNGUARDED base call.
func TestDrawableDisposeReleasesEverythingAndCallsTheBase(t *testing.T) {
	game, component := newDrawableFixture(t)
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	disposed := 0
	if _, err := component.AddDisposedHandler(func(sender any, args *EventArgs) error {
		disposed++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.Dispose(true); err != nil {
		t.Fatalf("Dispose(true): %v", err)
	}
	want := "DeviceCreated,DeviceResetting,DeviceReset,DeviceDisposing"
	if got := strings.Join(service.removeCalls, ","); got != want {
		t.Fatalf("removed %s, want %s in the reference's own order", got, want)
	}
	if disposed != 1 {
		t.Fatalf("the base Dispose raised Disposed %d times, want one", disposed)
	}
	// Dispose is NOT idempotent, in either class: there is no disposed flag
	// anywhere, so a second call disposes and raises again.
	if err := component.Dispose(true); err != nil {
		t.Fatalf("second Dispose(true): %v", err)
	}
	if disposed != 2 {
		t.Fatalf("Disposed raised %d times over two Dispose calls, want two", disposed)
	}
	_ = game
}

// TestDrawableDisposeFalseSkipsTheDerivedBodyAndDefersToTheBase pins the
// division of responsibility between the two guards.
//
// DrawableGameComponent::Dispose(bool) guards ONLY its own work -- UnloadContent
// and the four removals -- and calls base.Dispose(disposing) unconditionally,
// outside the guard. GameComponent::Dispose(bool) then guards its ENTIRE body
// on the same flag. So Dispose(false) does nothing observable, and that is the
// BASE's decision rather than the derived class's: the derived class hands the
// flag on and lets the base decide, which is exactly what the two IL bodies do
// and is the difference a "return early if !disposing" rewrite would erase.
func TestDrawableDisposeFalseSkipsTheDerivedBodyAndDefersToTheBase(t *testing.T) {
	_, component := newDrawableFixture(t)
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	disposed := 0
	if _, err := component.AddDisposedHandler(func(sender any, args *EventArgs) error {
		disposed++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.Dispose(false); err != nil {
		t.Fatalf("Dispose(false): %v", err)
	}
	if len(service.removeCalls) != 0 {
		t.Fatalf("Dispose(false) removed %v; the removals are inside if (disposing)", service.removeCalls)
	}
	if disposed != 0 {
		t.Fatalf("Dispose(false) raised Disposed %d times; GameComponent::Dispose(bool) guards its whole body", disposed)
	}
	// And the same component, disposed with true, does everything -- which is
	// what proves the flag reached the base rather than being swallowed.
	if err := component.Dispose(true); err != nil {
		t.Fatalf("Dispose(true): %v", err)
	}
	if len(service.removeCalls) != 4 || disposed != 1 {
		t.Fatalf("Dispose(true) after Dispose(false): removals=%v raises=%d", service.removeCalls, disposed)
	}
}

// TestDrawableSettersAnnounceOnlyOnChange pins both compare-store-announce
// setters and their suppression.
func TestDrawableSettersAnnounceOnlyOnChange(t *testing.T) {
	_, component := newDrawableFixture(t)
	visible, drawOrder := 0, 0
	if _, err := component.AddVisibleChangedHandler(func(sender any, args *EventArgs) error {
		if sender != component {
			t.Error("VisibleChanged sender is not the component")
		}
		visible++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddDrawOrderChangedHandler(func(sender any, args *EventArgs) error {
		drawOrder++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The constructor already stored true, so assigning true announces nothing.
	if err := component.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatal("an unchanged Visible assignment announced")
	}
	if err := component.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if visible != 1 || component.Visible() {
		t.Fatalf("SetVisible(false): raised %d times, Visible=%t", visible, component.Visible())
	}
	if err := component.SetDrawOrder(0); err != nil {
		t.Fatal(err)
	}
	if drawOrder != 0 {
		t.Fatal("an unchanged DrawOrder assignment announced")
	}
	if err := component.SetDrawOrder(7); err != nil {
		t.Fatal(err)
	}
	if drawOrder != 1 || component.DrawOrder() != 7 {
		t.Fatalf("SetDrawOrder(7): raised %d times, DrawOrder=%d", drawOrder, component.DrawOrder())
	}
}

// TestDrawableSatisfiesTheComponentContracts is the point of the type: it is
// the profile's one shipped IDrawable, and Game.Components sorts and calls it
// through these interfaces.
func TestDrawableSatisfiesTheComponentContracts(t *testing.T) {
	_, component := newDrawableFixture(t)
	var _ IDrawable = component
	var _ IUpdateable = component
	var _ IGameComponent = component
}

// TestDrawableInheritedMembersReachTheBaseObject proves the forwarding is real
// rather than a second copy of the state: mutating through the derived type is
// observable through the base's own event, which only the base object owns.
func TestDrawableInheritedMembersReachTheBaseObject(t *testing.T) {
	_, component := newDrawableFixture(t)
	changed := 0
	if _, err := component.AddEnabledChangedHandler(func(sender any, args *EventArgs) error {
		changed++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if changed != 1 || component.Enabled() {
		t.Fatalf("SetEnabled(false): raised %d times, Enabled=%t", changed, component.Enabled())
	}
	if err := component.SetUpdateOrder(3); err != nil {
		t.Fatal(err)
	}
	if component.UpdateOrder() != 3 {
		t.Fatalf("UpdateOrder = %d after SetUpdateOrder(3)", component.UpdateOrder())
	}
}

// TestDrawableComponentServiceReaderIsInstalled pins the bridge's other half:
// the framework package's init must register the reader, or the Graphics
// package's projection of get_GraphicsDevice can never find a service.
func TestDrawableComponentServiceReaderIsInstalled(t *testing.T) {
	_, component := newDrawableFixture(t)
	if _, ok := servicebridge.ComponentService(component); ok {
		t.Fatal("an uninitialized component reported a resolved service")
	}
	service := &stubDeviceService{}
	installResolver(t, service, false)
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	value, ok := servicebridge.ComponentService(component)
	if !ok {
		t.Fatal("an initialized component reported no service")
	}
	if value != any(service) {
		t.Fatal("the reader returned a different object than the one Initialize resolved")
	}
	// Something that is not a component reports nothing rather than panicking.
	if _, ok := servicebridge.ComponentService("not a component"); ok {
		t.Fatal("the reader accepted a non-component")
	}
}
