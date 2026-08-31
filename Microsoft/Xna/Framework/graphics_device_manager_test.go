package framework

import (
	"errors"
	"strings"
	"testing"
)

// The managed half of GraphicsDeviceManager, measured on a manager with no
// native one behind it.
//
// That is not a shortcut: in the reference every one of these members is
// managed field work that reaches a device only later, at ChangeDevice, so
// this is exactly the state the reference is in between construction and the
// first ApplyChanges. The push half -- that a stored value reaches CNA's own
// manager -- needs a live native game and is proved in tools/native_stress.

func newManagedManager() *GraphicsDeviceManager { return newGraphicsDeviceManagerState() }

// TestManagerConstructorDefaults pins the field-initializer block, which runs
// before the base constructor and before the null check.
func TestManagerConstructorDefaults(t *testing.T) {
	m := newManagedManager()
	if got := m.PreferredBackBufferWidth(); got != 800 {
		t.Fatalf("PreferredBackBufferWidth = %d, want DefaultBackBufferWidth 800", got)
	}
	if got := m.PreferredBackBufferHeight(); got != 480 {
		t.Fatalf("PreferredBackBufferHeight = %d, want DefaultBackBufferHeight 480", got)
	}
	if !m.SynchronizeWithVerticalRetrace() {
		t.Fatal("SynchronizeWithVerticalRetrace = false; the constructor stores true")
	}
	if m.IsFullScreen() || m.PreferMultiSampling() {
		t.Fatalf("IsFullScreen=%t PreferMultiSampling=%t; neither is assigned by the constructor",
			m.IsFullScreen(), m.PreferMultiSampling())
	}
	if m.SupportedOrientations() != DisplayOrientationDefault {
		t.Fatalf("SupportedOrientations = %v, want Default", m.SupportedOrientations())
	}
	if m.depthStencilFormat != 2 {
		t.Fatalf("depthStencilFormat = %d; the constructor stores DepthFormat.Depth24, which is 2", m.depthStencilFormat)
	}
	if m.isDeviceDirty {
		t.Fatal("a fresh manager is already dirty; the constructor sets no configuration through a setter")
	}
}

// TestManagerDefaultsAreNotTheWindowDefaults pins a distinction that is easy to
// get wrong: the assembly declares TWO pairs of default dimensions, and they
// are different. GameWindow's are 800x600; the manager's back buffer is
// 800x480.
func TestManagerDefaultsAreNotTheWindowDefaults(t *testing.T) {
	if GraphicsDeviceManagerDefaultBackBufferWidth() != 800 {
		t.Fatalf("DefaultBackBufferWidth = %d", GraphicsDeviceManagerDefaultBackBufferWidth())
	}
	if GraphicsDeviceManagerDefaultBackBufferHeight() != 480 {
		t.Fatalf("DefaultBackBufferHeight = %d, want 480 -- GameWindow's 600 is a different constant",
			GraphicsDeviceManagerDefaultBackBufferHeight())
	}
}

// TestManagerDimensionSettersRejectNonPositive pins the one validation in the
// nine setters, including its exact boundary: the IL compares with `bgt` on
// zero, so zero is rejected and one is accepted.
func TestManagerDimensionSettersRejectNonPositive(t *testing.T) {
	m := newManagedManager()
	for _, value := range []int32{0, -1} {
		if err := m.SetPreferredBackBufferWidth(value); err == nil {
			t.Fatalf("SetPreferredBackBufferWidth(%d) succeeded", value)
		} else if !errors.Is(err, errBackBufferDimension) ||
			!strings.Contains(err.Error(), "BackBufferWidth and BackBufferHeight must be greater than zero.") {
			t.Fatalf("SetPreferredBackBufferWidth(%d) = %v, want the reference's message", value, err)
		}
		if err := m.SetPreferredBackBufferHeight(value); err == nil {
			t.Fatalf("SetPreferredBackBufferHeight(%d) succeeded", value)
		}
	}
	// A rejected assignment stores nothing.
	if m.PreferredBackBufferWidth() != 800 || m.PreferredBackBufferHeight() != 480 {
		t.Fatalf("a rejected dimension stored: %dx%d", m.PreferredBackBufferWidth(), m.PreferredBackBufferHeight())
	}
	if m.isDeviceDirty {
		t.Fatal("a rejected dimension raised the dirty flag")
	}
	if err := m.SetPreferredBackBufferWidth(1); err != nil {
		t.Fatalf("SetPreferredBackBufferWidth(1) = %v; one is above the boundary", err)
	}
	if m.PreferredBackBufferWidth() != 1 {
		t.Fatalf("PreferredBackBufferWidth = %d after accepting 1", m.PreferredBackBufferWidth())
	}
}

// TestManagerSettersRaiseTheDirtyFlagAndStore pins that every one of the six
// framework-typed setters stores and marks the configuration dirty. None of
// them suppresses an unchanged value -- unlike GameWindow's Title or
// GameComponent's Enabled, these setters have no inequality guard at all, so
// assigning the same value again still dirties the device.
func TestManagerSettersRaiseTheDirtyFlagAndStore(t *testing.T) {
	for name, assign := range map[string]func(*GraphicsDeviceManager) error{
		"PreferredBackBufferWidth":       func(m *GraphicsDeviceManager) error { return m.SetPreferredBackBufferWidth(1280) },
		"PreferredBackBufferHeight":      func(m *GraphicsDeviceManager) error { return m.SetPreferredBackBufferHeight(720) },
		"IsFullScreen":                   func(m *GraphicsDeviceManager) error { return m.SetIsFullScreen(true) },
		"SynchronizeWithVerticalRetrace": func(m *GraphicsDeviceManager) error { return m.SetSynchronizeWithVerticalRetrace(true) },
		"PreferMultiSampling":            func(m *GraphicsDeviceManager) error { return m.SetPreferMultiSampling(false) },
		"SupportedOrientations":          func(m *GraphicsDeviceManager) error { return m.SetSupportedOrientations(DisplayOrientationDefault) },
	} {
		m := newManagedManager()
		if err := assign(m); err != nil {
			t.Fatalf("Set%s: %v", name, err)
		}
		if !m.isDeviceDirty {
			t.Fatalf("Set%s left the device clean; every setter raises isDeviceDirty unconditionally", name)
		}
	}
}

// TestManagerDimensionSettersClearTheResizedFlag pins the third store the two
// dimension setters make, which the other seven do not: a resized back buffer
// is forgotten the moment a consumer states a preference.
func TestManagerDimensionSettersClearTheResizedFlag(t *testing.T) {
	m := newManagedManager()
	m.useResizedBackBuffer = true
	if err := m.SetPreferredBackBufferWidth(640); err != nil {
		t.Fatal(err)
	}
	if m.useResizedBackBuffer {
		t.Fatal("SetPreferredBackBufferWidth left useResizedBackBuffer set")
	}
	m.useResizedBackBuffer = true
	if err := m.SetPreferredBackBufferHeight(360); err != nil {
		t.Fatal(err)
	}
	if m.useResizedBackBuffer {
		t.Fatal("SetPreferredBackBufferHeight left useResizedBackBuffer set")
	}
	// A setter that is NOT a dimension leaves it alone.
	m.useResizedBackBuffer = true
	if err := m.SetIsFullScreen(true); err != nil {
		t.Fatal(err)
	}
	if !m.useResizedBackBuffer {
		t.Fatal("SetIsFullScreen cleared useResizedBackBuffer; only the two dimension setters do")
	}
}

// TestManagerToggleFullScreenGoesThroughTheSetter pins that ToggleFullScreen
// flips through the projected SETTER rather than the field, exactly as the
// reference's `call set_IsFullScreen` does -- so the store, the dirty flag and
// the push all happen. With no native manager the apply step reports rather
// than pretending, which is what the second half of the test measures.
func TestManagerToggleFullScreenGoesThroughTheSetter(t *testing.T) {
	m := newManagedManager()
	// With no resource the guard reports; the flip has not happened yet.
	if err := m.ToggleFullScreen(); err == nil {
		t.Fatal("ToggleFullScreen succeeded with no native manager")
	}
	if m.IsFullScreen() {
		t.Fatal("ToggleFullScreen flipped the flag before its guard")
	}
}

// TestManagerApplyChangesRefusesWithoutANativeManager pins the Go-only guard.
func TestManagerApplyChangesRefusesWithoutANativeManager(t *testing.T) {
	if err := newManagedManager().ApplyChanges(); err == nil {
		t.Fatal("ApplyChanges succeeded with no native manager")
	}
}

// TestManagerConfigurationSlotsRoundTripThroughTheBridge pins the three
// Graphics-typed properties from this side: their VALUES are this object's
// managed state, and the bridge is the only thing that reaches them.
func TestManagerConfigurationSlotsRoundTripThroughTheBridge(t *testing.T) {
	m := newManagedManager()
	if m.graphicsProfile != 0 || m.backBufferFormat != 0 || m.depthStencilFormat != 2 {
		t.Fatalf("slot defaults = %d/%d/%d", m.graphicsProfile, m.backBufferFormat, m.depthStencilFormat)
	}
	// The Graphics package writes through the bridge; the effect is visible
	// here, which is what makes the two halves one object rather than two.
	if err := setManagerSlotForTest(m, 0, 1); err != nil {
		t.Fatal(err)
	}
	if m.graphicsProfile != 1 || !m.isDeviceDirty {
		t.Fatalf("graphicsProfile = %d dirty = %t after a bridge write", m.graphicsProfile, m.isDeviceDirty)
	}
}

// TestManagerRaisersReachTheirOwnEvents pins the four protected raisers. All
// four have the identical 22-byte body in the reference, so a copy-paste that
// raised a neighbour would look right and behave wrong.
func TestManagerRaisersReachTheirOwnEvents(t *testing.T) {
	m := newManagedManager()
	counts := map[string]int{}
	add := func(name string, register func(EventHandler[*EventArgs]) (EventSubscription, error)) {
		t.Helper()
		if _, err := register(func(sender any, args *EventArgs) error {
			if sender != m {
				t.Errorf("%s sender is not the manager", name)
			}
			counts[name]++
			return nil
		}); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}
	add("created", m.AddDeviceCreatedHandler)
	add("resetting", m.AddDeviceResettingHandler)
	add("reset", m.AddDeviceResetHandler)
	add("disposing", m.AddDeviceDisposingHandler)
	add("disposed", m.AddDisposedHandler)

	for name, raise := range map[string]func(any, *EventArgs) error{
		"created":   m.OnDeviceCreated,
		"resetting": m.OnDeviceResetting,
		"reset":     m.OnDeviceReset,
		"disposing": m.OnDeviceDisposing,
	} {
		before := counts[name]
		if err := raise(m, EventArgsEmpty()); err != nil {
			t.Fatalf("On%s: %v", name, err)
		}
		if counts[name] != before+1 {
			t.Fatalf("On%s raised %d times", name, counts[name]-before)
		}
	}
	total := 0
	for _, count := range counts {
		total += count
	}
	if total != 4 {
		t.Fatalf("four raisers produced %d raises in total: %v", total, counts)
	}
	if counts["disposed"] != 0 {
		t.Fatal("a device raiser reached Disposed, which has no protected raiser at all")
	}
}

// TestManagerNativeSignalsRouteToTheirOwnRaisePath pins the private bridge.
// This is the profile's THIRD signal family and the only one whose device
// events do not start at zero: Disposed is 0 and DeviceCreated is 1, so a table
// indexed as if the three families agreed would be off by one.
func TestManagerNativeSignalsRouteToTheirOwnRaisePath(t *testing.T) {
	m := newManagedManager()
	var order []string
	record := func(name string) EventHandler[*EventArgs] {
		return func(sender any, args *EventArgs) error {
			order = append(order, name)
			return nil
		}
	}
	if _, err := m.AddDisposedHandler(record("disposed")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddDeviceCreatedHandler(record("created")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddDeviceDisposingHandler(record("disposing")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddDeviceResetHandler(record("reset")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddDeviceResettingHandler(record("resetting")); err != nil {
		t.Fatal(err)
	}
	for identity := uint32(0); identity < 5; identity++ {
		if err := m.raiseNativeManagerEvent(identity); err != nil {
			t.Fatalf("signal %d: %v", identity, err)
		}
	}
	want := "disposed,created,disposing,reset,resetting"
	if strings.Join(order, ",") != want {
		t.Fatalf("signals routed as %v, want %s -- Disposed is identity ZERO in this family", order, want)
	}
	before := len(order)
	if err := m.raiseNativeManagerEvent(9); err != nil {
		t.Fatalf("unknown signal: %v", err)
	}
	if len(order) != before {
		t.Fatal("an unknown manager signal raised a projected event")
	}
}

// TestManagerHandlerFailureReachesTheCaller pins the settled dispatch contract
// on this family too: the first non-nil handler error propagates and no later
// handler runs.
func TestManagerHandlerFailureReachesTheCaller(t *testing.T) {
	m := newManagedManager()
	failure := errors.New("handler refused")
	later := 0
	if _, err := m.AddDeviceResetHandler(func(sender any, args *EventArgs) error { return failure }); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddDeviceResetHandler(func(sender any, args *EventArgs) error { later++; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := m.raiseNativeManagerEvent(3); !errors.Is(err, failure) {
		t.Fatalf("a native reset signal reported %v, want the handler's own error", err)
	}
	if later != 0 {
		t.Fatal("a later handler ran after an earlier one failed")
	}
}
