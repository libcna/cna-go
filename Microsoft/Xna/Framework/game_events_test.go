package framework

import (
	"errors"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// eventTestCallbacks is the minimum GameCallbacks a constructed Game needs. The
// five-member contract is unchanged by this milestone, and this type existing
// with exactly five methods is part of the proof.
type eventTestCallbacks struct{}

func (eventTestCallbacks) Initialize(*Game) error       { return nil }
func (eventTestCallbacks) LoadContent(*Game) error      { return nil }
func (eventTestCallbacks) Update(*Game, GameTime) error { return nil }
func (eventTestCallbacks) Draw(*Game, GameTime) error   { return nil }
func (eventTestCallbacks) UnloadContent(*Game) error    { return nil }

func newEventGame(t *testing.T) *Game {
	t.Helper()
	game, err := NewGame(eventTestCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

type recordedRaise struct {
	sender any
	args   *EventArgs
}

// TestGameEventAccessorsAreIndependent proves the four events are four separate
// registration lists: a handler added to one never runs for another, and a
// token from one never removes from another.
func TestGameEventAccessorsAreIndependent(t *testing.T) {
	game := newEventGame(t)
	var seen []string
	add := func(name string, adder func(EventHandler[*EventArgs]) (EventSubscription, error)) EventSubscription {
		token, err := adder(func(any, *EventArgs) error {
			seen = append(seen, name)
			return nil
		})
		if err != nil {
			t.Fatalf("Add%sHandler: %v", name, err)
		}
		return token
	}
	activatedToken := add("Activated", game.AddActivatedHandler)
	add("Deactivated", game.AddDeactivatedHandler)
	add("Exiting", game.AddExitingHandler)
	add("Disposed", game.AddDisposedHandler)

	// A token owned by Activated must leave every other list untouched.
	for _, remove := range []func(EventSubscription) error{
		game.RemoveDeactivatedHandler,
		game.RemoveExitingHandler,
		game.RemoveDisposedHandler,
	} {
		if err := remove(activatedToken); err != nil {
			t.Fatalf("cross-event Remove reported %v, want nil", err)
		}
	}

	for _, event := range []uint32{
		interop.GameEventActivated,
		interop.GameEventDeactivated,
		interop.GameEventExiting,
	} {
		if err := game.raiseNativeGameEvent(event); err != nil {
			t.Fatalf("raiseNativeGameEvent(%d): %v", event, err)
		}
	}
	// Disposed is the one event the host does not raise. Its reference raise
	// site is the tail of Dispose(bool), so the managed member drives it here
	// and the native disposal signal drives nothing.
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	want := []string{"Activated", "Deactivated", "Exiting", "Disposed"}
	if len(seen) != len(want) {
		t.Fatalf("delivered %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("delivered %v, want %v", seen, want)
		}
	}
}

// TestGameOnActivatedIgnoresSenderAndRaisesWithGame pins the exact IL: the
// sender parameter is accepted and dropped, and `this` is pushed instead.
func TestGameOnActivatedIgnoresSenderAndRaisesWithGame(t *testing.T) {
	game := newEventGame(t)
	other := newEventGame(t)
	var got recordedRaise
	if _, err := game.AddActivatedHandler(func(sender any, args *EventArgs) error {
		got = recordedRaise{sender: sender, args: args}
		return nil
	}); err != nil {
		t.Fatalf("AddActivatedHandler: %v", err)
	}
	custom := &EventArgs{}
	if err := game.OnActivated(other, custom); err != nil {
		t.Fatalf("OnActivated: %v", err)
	}
	if got.sender != any(game) {
		t.Fatalf("sender = %v, want the raising Game", got.sender)
	}
	if got.args != custom {
		t.Fatalf("args were not forwarded unchanged")
	}
}

// TestGameOnDeactivatedIgnoresSenderAndRaisesWithGame is the same shape over
// Deactivated, whose IL is byte-for-byte identical apart from the field.
func TestGameOnDeactivatedIgnoresSenderAndRaisesWithGame(t *testing.T) {
	game := newEventGame(t)
	other := newEventGame(t)
	var got recordedRaise
	if _, err := game.AddDeactivatedHandler(func(sender any, args *EventArgs) error {
		got = recordedRaise{sender: sender, args: args}
		return nil
	}); err != nil {
		t.Fatalf("AddDeactivatedHandler: %v", err)
	}
	custom := &EventArgs{}
	if err := game.OnDeactivated(other, custom); err != nil {
		t.Fatalf("OnDeactivated: %v", err)
	}
	if got.sender != any(game) {
		t.Fatalf("sender = %v, want the raising Game", got.sender)
	}
	if got.args != custom {
		t.Fatalf("args were not forwarded unchanged")
	}
}

// TestGameOnExitingRaisesWithNilSender is the single-instruction difference
// that separates OnExiting from its two siblings: `ldnull` where they push
// `ldarg.0`. Nothing in the reference ever gives this event a sender.
func TestGameOnExitingRaisesWithNilSender(t *testing.T) {
	game := newEventGame(t)
	sawHandler := false
	var got recordedRaise
	if _, err := game.AddExitingHandler(func(sender any, args *EventArgs) error {
		sawHandler = true
		got = recordedRaise{sender: sender, args: args}
		return nil
	}); err != nil {
		t.Fatalf("AddExitingHandler: %v", err)
	}
	custom := &EventArgs{}
	if err := game.OnExiting(game, custom); err != nil {
		t.Fatalf("OnExiting: %v", err)
	}
	if !sawHandler {
		t.Fatal("Exiting handler did not run")
	}
	if got.sender != nil {
		t.Fatalf("sender = %v, want nil", got.sender)
	}
	if got.args != custom {
		t.Fatalf("args were not forwarded unchanged")
	}
}

// TestGameNativeActivationIsEdgeTriggered reproduces HostActivated's and
// HostDeactivated's guard: each raises only on a real transition, and each
// records the state BEFORE it announces.
func TestGameNativeActivationIsEdgeTriggered(t *testing.T) {
	game := newEventGame(t)
	activations, deactivations := 0, 0
	activeDuringRaise := []bool{}
	if _, err := game.AddActivatedHandler(func(any, *EventArgs) error {
		activations++
		activeDuringRaise = append(activeDuringRaise, game.isActive)
		return nil
	}); err != nil {
		t.Fatalf("AddActivatedHandler: %v", err)
	}
	if _, err := game.AddDeactivatedHandler(func(any, *EventArgs) error {
		deactivations++
		activeDuringRaise = append(activeDuringRaise, game.isActive)
		return nil
	}); err != nil {
		t.Fatalf("AddDeactivatedHandler: %v", err)
	}

	// A deactivation before any activation is suppressed: isActive is false.
	if err := game.raiseNativeGameEvent(interop.GameEventDeactivated); err != nil {
		t.Fatalf("raise: %v", err)
	}
	if deactivations != 0 {
		t.Fatalf("Deactivated raised %d times before any activation, want 0", deactivations)
	}

	// Two consecutive activation signals are one CLR transition.
	for i := 0; i < 2; i++ {
		if err := game.raiseNativeGameEvent(interop.GameEventActivated); err != nil {
			t.Fatalf("raise: %v", err)
		}
	}
	if activations != 1 {
		t.Fatalf("Activated raised %d times for two signals, want 1", activations)
	}

	for i := 0; i < 2; i++ {
		if err := game.raiseNativeGameEvent(interop.GameEventDeactivated); err != nil {
			t.Fatalf("raise: %v", err)
		}
	}
	if deactivations != 1 {
		t.Fatalf("Deactivated raised %d times for two signals, want 1", deactivations)
	}

	// A second activation after a deactivation is a new transition.
	if err := game.raiseNativeGameEvent(interop.GameEventActivated); err != nil {
		t.Fatalf("raise: %v", err)
	}
	if activations != 2 {
		t.Fatalf("Activated raised %d times, want 2", activations)
	}

	want := []bool{true, false, true}
	if len(activeDuringRaise) != len(want) {
		t.Fatalf("observed %v, want %v", activeDuringRaise, want)
	}
	for i := range want {
		if activeDuringRaise[i] != want[i] {
			t.Fatalf("observed %v, want %v: the flag must be set before the announcement", activeDuringRaise, want)
		}
	}
}

// TestGameNativeExitingIsNotEdgeTriggered separates Exiting from the two
// activation events: HostExiting has no guard at all, so every signal raises.
func TestGameNativeExitingIsNotEdgeTriggered(t *testing.T) {
	game := newEventGame(t)
	raises := 0
	if _, err := game.AddExitingHandler(func(any, *EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatalf("AddExitingHandler: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := game.raiseNativeGameEvent(interop.GameEventExiting); err != nil {
			t.Fatalf("raise: %v", err)
		}
	}
	if raises != 3 {
		t.Fatalf("Exiting raised %d times for three signals, want 3", raises)
	}
}

// TestGameNativeEventArgsAreTheSharedEmpty proves every native-sourced raise
// pushes the one System.EventArgs.Empty identity, exactly as the reference's
// `ldsfld EventArgs::Empty` does at each of the four sites.
//
// Each event is driven through its OWN raise path: the three the host raises
// through the native signal, and Disposed through managed Dispose, which is
// the only place the reference raises it.
func TestGameNativeEventArgsAreTheSharedEmpty(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		raise func(*Game) error
		add   func(*Game) func(EventHandler[*EventArgs]) (EventSubscription, error)
	}{
		{"Activated", func(g *Game) error { return g.raiseNativeGameEvent(interop.GameEventActivated) },
			func(g *Game) func(EventHandler[*EventArgs]) (EventSubscription, error) { return g.AddActivatedHandler }},
		{"Deactivated", func(g *Game) error {
			g.isActive = true
			return g.raiseNativeGameEvent(interop.GameEventDeactivated)
		}, func(g *Game) func(EventHandler[*EventArgs]) (EventSubscription, error) {
			return g.AddDeactivatedHandler
		}},
		{"Exiting", func(g *Game) error { return g.raiseNativeGameEvent(interop.GameEventExiting) },
			func(g *Game) func(EventHandler[*EventArgs]) (EventSubscription, error) { return g.AddExitingHandler }},
		{"Disposed", func(g *Game) error { return g.DisposeByNone() },
			func(g *Game) func(EventHandler[*EventArgs]) (EventSubscription, error) { return g.AddDisposedHandler }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			game := newEventGame(t)
			var got *EventArgs
			if _, err := testCase.add(game)(func(_ any, args *EventArgs) error {
				got = args
				return nil
			}); err != nil {
				t.Fatalf("Add: %v", err)
			}
			if err := testCase.raise(game); err != nil {
				t.Fatalf("raise: %v", err)
			}
			if got != EventArgsEmpty() {
				t.Fatalf("args are not the shared EventArgs.Empty identity")
			}
		})
	}
}

// TestGameDisposedRaisesWithTheGame pins Dispose(bool)'s tail, which pushes
// `ldarg.0` as the sender and invokes the delegate field directly -- there is
// no OnDisposed on Game to route through.
//
// The raise is driven through managed Dispose because that is the reference's
// ONLY raise site for this event; see TestTheNativeDisposalSignalRaisesNothing
// for the other half of the same claim.
func TestGameDisposedRaisesWithTheGame(t *testing.T) {
	game := newEventGame(t)
	var got recordedRaise
	if _, err := game.AddDisposedHandler(func(sender any, args *EventArgs) error {
		got = recordedRaise{sender: sender, args: args}
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if got.sender != any(game) {
		t.Fatalf("sender = %v, want the Game", got.sender)
	}
	if got.args != EventArgsEmpty() {
		t.Fatalf("args = %v, want the shared EventArgs.Empty", got.args)
	}
}

// TestTheNativeDisposalSignalRaisesNothing is the correction Foundation 39
// makes, stated as a test rather than as prose.
//
// CNA delivers CNA_GAME_EVENT_DISPOSED from inside cna_game_destroy. Game's own
// Disposed event is raised from Dispose(bool) and from nowhere else, so the
// native signal must not drive it: doing so would raise the event at a moment
// the reference has no raise at, and would raise it for a consumer who never
// disposed anything.
func TestTheNativeDisposalSignalRaisesNothing(t *testing.T) {
	game := newEventGame(t)
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := game.raiseNativeGameEvent(interop.GameEventDisposed); err != nil {
			t.Fatalf("native disposal signal reported %v", err)
		}
	}
	if raised != 0 {
		t.Fatalf("the native disposal signal raised Game.Disposed %d times", raised)
	}
	// And the managed path still works, on the same Game, afterwards.
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raised != 1 {
		t.Fatalf("managed Dispose raised Game.Disposed %d times, want 1", raised)
	}
}

// TestGameEventDispatchIsRegistrationOrder is the reason the bridge installs
// exactly one native subscription per event. CNA invokes multiple native
// registrations on one event in REVERSE order, measured against the pinned
// binary; routing every Go handler through one EventSource makes the order the
// registration order a CLR multicast invocation list uses.
func TestGameEventDispatchIsRegistrationOrder(t *testing.T) {
	game := newEventGame(t)
	var order []int
	for i := 0; i < 4; i++ {
		index := i
		if _, err := game.AddExitingHandler(func(any, *EventArgs) error {
			order = append(order, index)
			return nil
		}); err != nil {
			t.Fatalf("AddExitingHandler: %v", err)
		}
	}
	if err := game.raiseNativeGameEvent(interop.GameEventExiting); err != nil {
		t.Fatalf("raise: %v", err)
	}
	for i, value := range order {
		if value != i {
			t.Fatalf("dispatch order %v is not registration order", order)
		}
	}
	if len(order) != 4 {
		t.Fatalf("dispatched %d handlers, want 4", len(order))
	}
}

// TestGameEventDuplicateRegistrationsAreIndependent holds the settled event
// projection: the token names the registration, not the handler value.
func TestGameEventDuplicateRegistrationsAreIndependent(t *testing.T) {
	game := newEventGame(t)
	calls := 0
	handler := func(any, *EventArgs) error {
		calls++
		return nil
	}
	first, err := game.AddDisposedHandler(handler)
	if err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if _, err := game.AddDisposedHandler(handler); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if calls != 2 {
		t.Fatalf("one handler added twice ran %d times, want 2", calls)
	}
	if err := game.RemoveDisposedHandler(first); err != nil {
		t.Fatalf("RemoveDisposedHandler: %v", err)
	}
	calls = 0
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if calls != 1 {
		t.Fatalf("after removing one of two registrations the handler ran %d times, want 1", calls)
	}
	// A stale token, the zero token and a repeat removal are all harmless.
	for _, token := range []EventSubscription{first, {}, first} {
		if err := game.RemoveDisposedHandler(token); err != nil {
			t.Fatalf("harmless Remove reported %v, want nil", err)
		}
	}
	calls = 0
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if calls != 1 {
		t.Fatalf("harmless removals changed the list: handler ran %d times, want 1", calls)
	}
}

// TestGameEventNilHandlerRegistersNothing mirrors Delegate.Combine(existing,
// null) returning the existing list unchanged.
func TestGameEventNilHandlerRegistersNothing(t *testing.T) {
	game := newEventGame(t)
	token, err := game.AddActivatedHandler(nil)
	if err != nil {
		t.Fatalf("AddActivatedHandler(nil): %v", err)
	}
	if token != (EventSubscription{}) {
		t.Fatal("a nil handler must return the zero token")
	}
	if err := game.raiseNativeGameEvent(interop.GameEventActivated); err != nil {
		t.Fatalf("raise: %v", err)
	}
}

// TestGameEventHandlerFailureStopsDispatch holds the settled contract: the
// first non-nil handler error propagates, later handlers do not run, and the
// registration list survives.
func TestGameEventHandlerFailureStopsDispatch(t *testing.T) {
	game := newEventGame(t)
	sentinel := errors.New("game event handler sentinel")
	ran := 0
	if _, err := game.AddExitingHandler(func(any, *EventArgs) error {
		ran++
		return sentinel
	}); err != nil {
		t.Fatalf("AddExitingHandler: %v", err)
	}
	later := 0
	if _, err := game.AddExitingHandler(func(any, *EventArgs) error {
		later++
		return nil
	}); err != nil {
		t.Fatalf("AddExitingHandler: %v", err)
	}
	if err := game.raiseNativeGameEvent(interop.GameEventExiting); !errors.Is(err, sentinel) {
		t.Fatalf("raise reported %v, want the handler sentinel", err)
	}
	if later != 0 {
		t.Fatalf("a handler after the failing one ran %d times, want 0", later)
	}
	if err := game.raiseNativeGameEvent(interop.GameEventExiting); !errors.Is(err, sentinel) {
		t.Fatalf("second raise reported %v, want the same sentinel: a failed dispatch must not drop registrations", err)
	}
	if ran != 2 {
		t.Fatalf("failing handler ran %d times over two raises, want 2", ran)
	}
}

// TestGameEventUnknownIdentityIsInert proves an identity outside
// CNA_GAME_EVENT_MAXIMUM raises nothing and reports nothing. CNA rejects such
// an identity at subscription time with CNA_RESULT_INVALID_ARGUMENT, so this
// can only be reached by a future ABI, and it must not be routed to an
// arbitrary event.
func TestGameEventUnknownIdentityIsInert(t *testing.T) {
	game := newEventGame(t)
	for _, add := range []func(EventHandler[*EventArgs]) (EventSubscription, error){
		game.AddActivatedHandler,
		game.AddDeactivatedHandler,
		game.AddExitingHandler,
		game.AddDisposedHandler,
	} {
		if _, err := add(func(any, *EventArgs) error {
			t.Fatal("an unknown native identity raised a projected event")
			return nil
		}); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	game.isActive = true
	for _, event := range []uint32{4, 5, 99, ^uint32(0)} {
		if err := game.raiseNativeGameEvent(event); err != nil {
			t.Fatalf("raiseNativeGameEvent(%d) reported %v, want nil", event, err)
		}
	}
}

// TestGameEventBridgeToleratesNilGame keeps the private bridge total. Nothing
// in the binding can deliver a signal to a nil Game today, and if the ownership
// tables ever changed, dropping the signal is the correct answer rather than a
// panic crossing a C frame.
func TestGameEventBridgeToleratesNilGame(t *testing.T) {
	var game *Game
	if err := game.raiseNativeGameEvent(interop.GameEventExiting); err != nil {
		t.Fatalf("nil Game bridge reported %v, want nil", err)
	}
}

// TestGameEventSubscriptionCarriesNoNativeIdentity holds the leak rule at the
// type level: the token a Game event hands out is the same opaque
// EventSubscription every other projected XNA event uses, with no exported
// field and nothing derived from a CNA registration handle.
func TestGameEventSubscriptionCarriesNoNativeIdentity(t *testing.T) {
	game := newEventGame(t)
	token, err := game.AddDisposedHandler(func(any, *EventArgs) error { return nil })
	if err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	var zero EventSubscription
	if token == zero {
		t.Fatal("a real registration must not produce the zero token")
	}
	// The registration is a Go pointer into this package's own table. It is
	// never a CNA_GameEventRegistrationHandle, whose values the pinned runtime
	// hands out as small non-zero integers.
	if token.registration == nil || token.registration.owner != any(&game.disposed) {
		t.Fatal("the token does not name this event's own registration list")
	}
}

// TestGameCallbacksContractIsStillFiveMembers is the compile-time half of the
// promise that this milestone broke nothing: eventTestCallbacks declares
// exactly the five original methods and nothing else, and it still satisfies
// GameCallbacks.
func TestGameCallbacksContractIsStillFiveMembers(t *testing.T) {
	var _ GameCallbacks = eventTestCallbacks{}
	if _, err := NewGame(eventTestCallbacks{}); err != nil {
		t.Fatalf("a five-member GameCallbacks implementation was rejected: %v", err)
	}
}
