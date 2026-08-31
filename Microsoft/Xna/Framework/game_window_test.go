package framework

import (
	"errors"
	"testing"
)

func newWindowGame(t *testing.T) *Game {
	t.Helper()
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

// TestWindowIdentityIsStable is the property the whole projection hangs on.
//
// Game::get_Window resolves host.Window, WindowsGameHost::get_Window is one
// field read, and EnsureHost() runs inside Game::.ctor -- so in the reference
// every call returns the SAME GameWindow object for the Game's whole life. A
// consumer who subscribes to game.Window.ClientSizeChanged and later reads
// game.Window.Title is talking to one object, and a projection that handed out
// a fresh wrapper per call would silently lose every subscription.
func TestWindowIdentityIsStable(t *testing.T) {
	game := newWindowGame(t)
	first := game.Window()
	if first == nil {
		t.Fatal("Game.Window returned nil; the constructor allocates the window")
	}
	if second := game.Window(); second != first {
		t.Fatal("Game.Window returned a different object on the second call")
	}
	other := newWindowGame(t)
	if other.Window() == first {
		t.Fatal("two Games share one GameWindow; each host owns its own")
	}
}

// TestWindowSubscriptionSurvivesARepeatedGetter is the consequence that makes
// the identity worth pinning: it is what a consumer actually does.
func TestWindowSubscriptionSurvivesARepeatedGetter(t *testing.T) {
	game := newWindowGame(t)
	raised := 0
	if _, err := game.Window().AddClientSizeChangedHandler(func(sender any, args *EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddClientSizeChangedHandler: %v", err)
	}
	// The raise goes through a SECOND call to the getter, which is exactly
	// where a per-call wrapper would drop the registration.
	if err := game.Window().OnClientSizeChanged(); err != nil {
		t.Fatalf("OnClientSizeChanged: %v", err)
	}
	if raised != 1 {
		t.Fatalf("handler ran %d times, want 1", raised)
	}
}

// TestWindowTitleStartsEmpty pins the assembly constructor's one assignment:
//
//	.ctor()  base..ctor(); this.title = String.Empty
func TestWindowTitleStartsEmpty(t *testing.T) {
	if got := newWindowGame(t).Window().Title(); got != "" {
		t.Fatalf("Title = %q, want the empty string the constructor stores", got)
	}
}

// TestWindowTitleRoundTripsThroughTheManagedField pins that the getter reads
// the field the setter wrote, and not a native query. With no live native game
// there is nothing to query, so a projection that asked CNA would answer the
// empty string here forever.
func TestWindowTitleRoundTripsThroughTheManagedField(t *testing.T) {
	window := newWindowGame(t).Window()
	if err := window.SetTitleProperty("Blupi"); err != nil {
		t.Fatalf("SetTitleProperty: %v", err)
	}
	if got := window.Title(); got != "Blupi" {
		t.Fatalf("Title = %q, want %q", got, "Blupi")
	}
	if err := window.SetTitleProperty(""); err != nil {
		t.Fatalf("SetTitleProperty(empty): %v", err)
	}
	if got := window.Title(); got != "" {
		t.Fatalf("Title = %q after assigning the empty string, want it back", got)
	}
}

// TestWindowTitleSuppressesAnUnchangedValue pins set_Title's inequality guard:
//
//	if (this.title != value) { this.title = value; this.SetTitle(this.title); }
//
// The observable consequence of the guard is that an unchanged assignment
// performs no platform call at all. Here it is proved from the managed side --
// the store is skipped -- and native_stress proves the other half from a live
// run, where an unchanged assignment from a non-owner goroutine succeeds
// because it never reaches CNA while a changed one is refused.
func TestWindowTitleSuppressesAnUnchangedValue(t *testing.T) {
	window := newWindowGame(t).Window()
	if err := window.SetTitleProperty("same"); err != nil {
		t.Fatalf("SetTitleProperty: %v", err)
	}
	window.title = "sentinel written behind the setter"
	if err := window.SetTitleProperty("sentinel written behind the setter"); err != nil {
		t.Fatalf("SetTitleProperty(unchanged): %v", err)
	}
	if got := window.Title(); got != "sentinel written behind the setter" {
		t.Fatalf("Title = %q; an unchanged assignment must not rewrite the field", got)
	}
}

// TestCurrentOrientationIsTheReferenceConstant pins the most surprising member
// of the type. WindowsGameWindow::get_CurrentOrientation is the whole of
//
//	ldc.i4.0
//	ret
//
// so in the selected Windows runtime profile the answer is
// DisplayOrientation.Default on every machine, in every window state, with no
// platform query anywhere. CNA-Go answers the same constant and never asks CNA,
// which is why this member is infallible.
func TestCurrentOrientationIsTheReferenceConstant(t *testing.T) {
	if got := newWindowGame(t).Window().CurrentOrientation(); got != DisplayOrientationDefault {
		t.Fatalf("CurrentOrientation = %v, want DisplayOrientationDefault", got)
	}
}

// TestSetSupportedOrientationsDoesNothing pins the other constant-shaped
// member. WindowsGameWindow::SetSupportedOrientations is one `ret`.
//
// The test is deliberately about ABSENCE: no state changes, nothing is stored,
// and nothing else on the window moves. A future implementation that "helpfully"
// forwarded to GraphicsDeviceManager's native orientation route would fail here
// only if it stored something -- so the assertion is on every observable the
// window has, not on a flag.
func TestSetSupportedOrientationsDoesNothing(t *testing.T) {
	window := newWindowGame(t).Window()
	if err := window.SetTitleProperty("before"); err != nil {
		t.Fatalf("SetTitleProperty: %v", err)
	}
	window.SetSupportedOrientations(DisplayOrientationLandscapeLeft | DisplayOrientationPortrait)
	if got := window.Title(); got != "before" {
		t.Fatalf("Title = %q after SetSupportedOrientations", got)
	}
	if got := window.CurrentOrientation(); got != DisplayOrientationDefault {
		t.Fatalf("CurrentOrientation = %v after SetSupportedOrientations; the member stores nothing", got)
	}
	if window.inDeviceTransition {
		t.Fatal("SetSupportedOrientations moved the device-transition flag")
	}
}

// TestGuardedMembersAnswerTheReferenceFallback pins the first half of the
// measured guard split. With no window WindowsGameWindow answers IntPtr.Zero,
// false and String.Empty, and does nothing on assignment -- it does not throw.
func TestGuardedMembersAnswerTheReferenceFallback(t *testing.T) {
	window := newWindowGame(t).Window()
	handle, err := window.Handle()
	if err != nil {
		t.Fatalf("Handle reported a failure with no window: %v", err)
	}
	if handle != 0 {
		t.Fatalf("Handle = %#x with no window, want IntPtr.Zero", handle)
	}
	allow, err := window.AllowUserResizing()
	if err != nil {
		t.Fatalf("AllowUserResizing reported a failure with no window: %v", err)
	}
	if allow {
		t.Fatal("AllowUserResizing = true with no window, want false")
	}
	if err := window.SetAllowUserResizing(true); err != nil {
		t.Fatalf("SetAllowUserResizing reported a failure with no window: %v", err)
	}
	name, err := window.ScreenDeviceName()
	if err != nil {
		t.Fatalf("ScreenDeviceName reported a failure with no window: %v", err)
	}
	if name != "" {
		t.Fatalf("ScreenDeviceName = %q with no window, want String.Empty", name)
	}
	if err := window.SetTitleMethod("ignored"); err != nil {
		t.Fatalf("SetTitle reported a failure with no window: %v", err)
	}
}

// TestUnguardedMembersReportTheReferenceFailure pins the other half. These
// three dereference mainForm with NO null check, so the reference throws
// NullReferenceException; inventing a zero Rectangle here would make a failure
// indistinguishable from a real measurement of a zero-sized window.
func TestUnguardedMembersReportTheReferenceFailure(t *testing.T) {
	window := newWindowGame(t).Window()
	if _, err := window.ClientBounds(); err == nil {
		t.Fatal("ClientBounds succeeded with no window; the reference dereferences null")
	}
	if err := window.BeginScreenDeviceChange(true); err == nil {
		t.Fatal("BeginScreenDeviceChange succeeded with no window")
	}
	if err := window.EndScreenDeviceChangeByStringAndInt32AndInt32("screen", 640, 480); err == nil {
		t.Fatal("EndScreenDeviceChange succeeded with no window")
	}
	if err := window.EndScreenDeviceChangeByString("screen"); err == nil {
		t.Fatal("the one-argument overload succeeded with no window; it reads ClientBounds first")
	}
}

// TestBeginScreenDeviceChangeLeavesTheFlagClearOnFailure pins the instruction
// ORDER of the two transition members, which differ from each other:
//
//	Begin  mainForm.BeginScreenDeviceChange(x); inDeviceTransition = true;
//	End    try { ... } finally { inDeviceTransition = false; }
//
// Begin sets the flag AFTER the call, so a failing call leaves it clear. End
// clears it in a finally, so a failing call still clears it. Swapping either
// order would leave a Game permanently stuck in or out of a transition.
func TestBeginScreenDeviceChangeLeavesTheFlagClearOnFailure(t *testing.T) {
	window := newWindowGame(t).Window()
	if err := window.BeginScreenDeviceChange(false); err == nil {
		t.Fatal("BeginScreenDeviceChange succeeded with no window")
	}
	if window.inDeviceTransition {
		t.Fatal("a failing BeginScreenDeviceChange set the transition flag; the store follows the call")
	}
	window.inDeviceTransition = true
	if err := window.EndScreenDeviceChangeByStringAndInt32AndInt32("screen", 1, 1); err == nil {
		t.Fatal("EndScreenDeviceChange succeeded with no window")
	}
	if window.inDeviceTransition {
		t.Fatal("a failing EndScreenDeviceChange left the transition flag set; the store is in a finally")
	}
}

// TestTheThreePublicEventsRaiseSeparately pins that each protected raiser
// reaches its own registration list. All six have the identical 26-byte body,
// so a copy-paste that raised the wrong field would look right and behave
// wrong.
func TestTheThreePublicEventsRaiseSeparately(t *testing.T) {
	window := newWindowGame(t).Window()
	var clientSize, orientation, screenName int
	mustAdd := func(add func(EventHandler[*EventArgs]) (EventSubscription, error), counter *int) {
		t.Helper()
		if _, err := add(func(sender any, args *EventArgs) error {
			if sender != window {
				t.Errorf("handler sender = %v, want the window itself", sender)
			}
			if args != EventArgsEmpty() {
				t.Error("handler args are not EventArgs.Empty")
			}
			*counter++
			return nil
		}); err != nil {
			t.Fatalf("add handler: %v", err)
		}
	}
	mustAdd(window.AddClientSizeChangedHandler, &clientSize)
	mustAdd(window.AddOrientationChangedHandler, &orientation)
	mustAdd(window.AddScreenDeviceNameChangedHandler, &screenName)

	if err := window.OnClientSizeChanged(); err != nil {
		t.Fatal(err)
	}
	if clientSize != 1 || orientation != 0 || screenName != 0 {
		t.Fatalf("after OnClientSizeChanged: %d/%d/%d, want 1/0/0", clientSize, orientation, screenName)
	}
	if err := window.OnOrientationChanged(); err != nil {
		t.Fatal(err)
	}
	if clientSize != 1 || orientation != 1 || screenName != 0 {
		t.Fatalf("after OnOrientationChanged: %d/%d/%d, want 1/1/0", clientSize, orientation, screenName)
	}
	if err := window.OnScreenDeviceNameChanged(); err != nil {
		t.Fatal(err)
	}
	if clientSize != 1 || orientation != 1 || screenName != 1 {
		t.Fatalf("after OnScreenDeviceNameChanged: %d/%d/%d, want 1/1/1", clientSize, orientation, screenName)
	}
}

// TestTheAssemblyEventsHaveNoPublicAccessorButStillRaise pins the CLR
// `assembly` projection. Activated, Deactivated and Paint are assembly-visible
// in the reference -- only Game subscribes to them -- so they map to unexported
// package-scope registration lists with no Add/Remove pair. Their protected
// On... raisers ARE public contract members, and with nothing subscribed they
// raise nothing and report no failure, exactly as the reference's null check
// does.
func TestTheAssemblyEventsHaveNoPublicAccessorButStillRaise(t *testing.T) {
	window := newWindowGame(t).Window()
	for name, raise := range map[string]func() error{
		"OnActivated":   window.OnActivated,
		"OnDeactivated": window.OnDeactivated,
		"OnPaint":       window.OnPaint,
	} {
		if err := raise(); err != nil {
			t.Fatalf("%s with no subscriber: %v", name, err)
		}
	}
	// The lists are reachable from inside the package, which is precisely what
	// `assembly` visibility means, and they deliver when they are used.
	seen := 0
	if _, err := window.paint.Add(func(sender any, args *EventArgs) error {
		seen++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := window.OnPaint(); err != nil {
		t.Fatal(err)
	}
	if seen != 1 {
		t.Fatalf("the assembly-visible Paint list ran %d handlers, want 1", seen)
	}
}

// TestNativeWindowSignalsReachTheirOwnRaisePath pins the private bridge. The
// window family numbers from zero exactly as the game family does, so a signal
// routed by the wrong table would carry a perfectly valid-looking identity.
func TestNativeWindowSignalsReachTheirOwnRaisePath(t *testing.T) {
	window := newWindowGame(t).Window()
	var got []string
	record := func(name string) EventHandler[*EventArgs] {
		return func(sender any, args *EventArgs) error {
			got = append(got, name)
			return nil
		}
	}
	if _, err := window.AddClientSizeChangedHandler(record("client")); err != nil {
		t.Fatal(err)
	}
	if _, err := window.AddOrientationChangedHandler(record("orientation")); err != nil {
		t.Fatal(err)
	}
	if _, err := window.AddScreenDeviceNameChangedHandler(record("screen")); err != nil {
		t.Fatal(err)
	}
	for _, event := range []uint32{0, 1, 2} {
		if err := window.raiseNativeWindowEvent(event); err != nil {
			t.Fatalf("signal %d: %v", event, err)
		}
	}
	if len(got) != 3 || got[0] != "client" || got[1] != "orientation" || got[2] != "screen" {
		t.Fatalf("signals routed to %v, want [client orientation screen]", got)
	}
	// An identity outside the canonical range raises nothing rather than
	// picking a neighbour.
	before := len(got)
	if err := window.raiseNativeWindowEvent(9); err != nil {
		t.Fatalf("unknown signal: %v", err)
	}
	if len(got) != before {
		t.Fatal("an unknown window signal raised a projected event")
	}
}

// TestAWindowHandlerFailureReachesTheCaller pins that the settled event
// dispatch contract applies here too: the first non-nil handler error
// propagates and no later handler runs.
func TestAWindowHandlerFailureReachesTheCaller(t *testing.T) {
	window := newWindowGame(t).Window()
	failure := errors.New("handler refused")
	later := 0
	if _, err := window.AddClientSizeChangedHandler(func(sender any, args *EventArgs) error {
		return failure
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := window.AddClientSizeChangedHandler(func(sender any, args *EventArgs) error {
		later++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := window.OnClientSizeChanged(); !errors.Is(err, failure) {
		t.Fatalf("OnClientSizeChanged = %v, want the handler's own error", err)
	}
	if later != 0 {
		t.Fatal("a later handler ran after an earlier one failed")
	}
}
