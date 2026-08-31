package framework

import (
	"errors"
	"reflect"
	"testing"
)

// disposalComponent is a consumer's own component that declares the
// two-overload disposal spelling, which is the one Microsoft's own components
// take: a type declaring Dispose() and Dispose(bool) projects them as
// DisposeByNone and DisposeByBoolean.
type disposalComponent struct {
	name     string
	disposed int
	failure  error
	onceMore func()
}

func (c *disposalComponent) Initialize() error { return nil }

func (c *disposalComponent) DisposeByNone() error { return c.DisposeByBoolean(true) }

func (c *disposalComponent) DisposeByBoolean(disposing bool) error {
	if !disposing {
		return nil
	}
	c.disposed++
	if c.onceMore != nil {
		c.onceMore()
	}
	return c.failure
}

// simpleDisposalComponent declares only Dispose(), which is the projection a
// type that does NOT also declare Dispose(bool) receives. Both spellings are
// legitimate projections of IDisposable::Dispose and both must be found.
type simpleDisposalComponent struct {
	disposed int
}

func (c *simpleDisposalComponent) Initialize() error { return nil }
func (c *simpleDisposalComponent) Dispose() error    { c.disposed++; return nil }

// undisposableComponent declares no disposal member at all, which is the
// `isinst` failing: the reference skips it with no error, and so does this.
type undisposableComponent struct{ initialized int }

func (c *undisposableComponent) Initialize() error { c.initialized++; return nil }

func newDisposalGame(t *testing.T) *Game {
	t.Helper()
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

// TestDisposeDisposesEveryDisposableComponentInOrder pins the loop: a snapshot
// of Components, walked forward, disposing every component that declares the
// IDisposable member and skipping every one that does not.
func TestDisposeDisposesEveryDisposableComponentInOrder(t *testing.T) {
	game := newDisposalGame(t)
	first := &disposalComponent{name: "first"}
	simple := &simpleDisposalComponent{}
	plain := &undisposableComponent{}
	second := &disposalComponent{name: "second"}
	for _, component := range []IGameComponent{first, simple, plain, second} {
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if first.disposed != 1 || second.disposed != 1 {
		t.Fatalf("two-overload components disposed %d and %d times, want 1 each", first.disposed, second.disposed)
	}
	if simple.disposed != 1 {
		t.Fatalf("the single-overload component disposed %d times, want 1", simple.disposed)
	}
	if plain.initialized != 0 {
		t.Fatal("a component with no disposal member was touched")
	}
}

// TestDisposeWalksASnapshotSoASelfRemovingComponentIsStillReached is why the
// reference copies to an array first. GameComponent::Dispose(bool) removes
// itself from Game.Components, so disposing one component MUTATES the
// collection the loop would otherwise be walking.
func TestDisposeWalksASnapshotSoASelfRemovingComponentIsStillReached(t *testing.T) {
	game := newDisposalGame(t)
	var components []*disposalComponent
	for i := 0; i < 4; i++ {
		component := &disposalComponent{}
		component.onceMore = func() {
			// Remove itself, exactly as GameComponent::Dispose(bool) does.
			_, _ = game.Components().Remove(component)
		}
		components = append(components, component)
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	for i, component := range components {
		if component.disposed != 1 {
			t.Fatalf("component %d disposed %d times; the loop did not walk a snapshot", i, component.disposed)
		}
	}
	if got := game.Components().Count(); got != 0 {
		t.Fatalf("Components still holds %d items", got)
	}
}

// TestAFailingComponentStopsDisposalWhereItFailed holds the absence of a
// try/catch. A component that throws propagates straight out of Game.Dispose,
// leaving every later component undisposed and the Disposed event unraised.
func TestAFailingComponentStopsDisposalWhereItFailed(t *testing.T) {
	game := newDisposalGame(t)
	sentinel := errors.New("component disposal failure")
	first := &disposalComponent{}
	failing := &disposalComponent{failure: sentinel}
	later := &disposalComponent{}
	for _, component := range []IGameComponent{first, failing, later} {
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByNone(); !errors.Is(err, sentinel) {
		t.Fatalf("Dispose = %v, want the component's own failure", err)
	}
	if first.disposed != 1 {
		t.Fatalf("the component before the failure disposed %d times", first.disposed)
	}
	if later.disposed != 0 {
		t.Fatal("a component after the failure was still disposed; the reference has no exception handler")
	}
	if raised != 0 {
		t.Fatal("Disposed was raised even though disposal did not reach its raise site")
	}
}

// TestDisposeIsNotIdempotent is the reference behaviour a reader is most likely
// to assume away. Game has no disposed flag anywhere: every call re-runs the
// whole body and raises Disposed AGAIN.
func TestGameDisposeIsNotIdempotent(t *testing.T) {
	game := newDisposalGame(t)
	component := &simpleDisposalComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := game.DisposeByNone(); err != nil {
			t.Fatalf("Dispose %d: %v", i, err)
		}
	}
	if raised != 3 {
		t.Fatalf("three Dispose calls raised Disposed %d times, want 3", raised)
	}
	// The component is still in Components -- nothing here removes it -- so it
	// is disposed once per call too.
	if component.disposed != 3 {
		t.Fatalf("the component disposed %d times across three Dispose calls, want 3", component.disposed)
	}
}

// TestDisposeFalseReturnsImmediately pins IL_0000. Dispose(false) is the
// finalizer path and does nothing at all: no component is touched, no event is
// raised, and it is safe even on a Game whose constructor never ran.
func TestDisposeFalseReturnsImmediately(t *testing.T) {
	game := newDisposalGame(t)
	component := &simpleDisposalComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByBoolean(false); err != nil {
		t.Fatalf("Dispose(false) = %v, want nil", err)
	}
	if component.disposed != 0 || raised != 0 {
		t.Fatalf("Dispose(false) disposed %d components and raised %d events", component.disposed, raised)
	}
	// The guard is reached only after the disposing check, so an unconstructed
	// Game accepts Dispose(false) and refuses Dispose(true).
	zero := &Game{}
	if err := zero.DisposeByBoolean(false); err != nil {
		t.Fatalf("Dispose(false) on an unconstructed Game = %v, want nil", err)
	}
	if err := zero.DisposeByBoolean(true); !errors.Is(err, errGameNotConstructed) {
		t.Fatalf("Dispose(true) on an unconstructed Game = %v", err)
	}
	if err := zero.DisposeByNone(); !errors.Is(err, errGameNotConstructed) {
		t.Fatalf("Dispose() on an unconstructed Game = %v", err)
	}
	// Finalize runs Dispose(false), so it is a no-op on anything -- including
	// a Game whose constructor never ran, because the disposing check comes
	// before the state guard.
	if err := zero.Finalize(); err != nil {
		t.Fatalf("Finalize on an unconstructed Game = %v, want nil", err)
	}
	if err := game.Finalize(); err != nil {
		t.Fatalf("Finalize = %v, want nil", err)
	}
	if component.disposed != 0 || raised != 0 {
		t.Fatal("Finalize did something; its Dispose(false) returns at the first instruction")
	}
}

// TestDisposeReentrancyFromAHandlerDoesNotDeadlock covers the one place the
// CLR monitor's reentrancy is reachable: a Disposed handler that disposes
// again. CLR recurses; the TryLock projection proceeds without re-acquiring,
// which is the same observable behaviour on the single-threaded path.
func TestDisposeReentrancyFromAHandlerDoesNotDeadlock(t *testing.T) {
	game := newDisposalGame(t)
	depth, raised := 0, 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		raised++
		if depth < 2 {
			depth++
			return game.DisposeByNone()
		}
		return nil
	}); err != nil {
		t.Fatalf("AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raised != 3 {
		t.Fatalf("reentrant disposal raised Disposed %d times, want 3", raised)
	}
}

// TestDisposalSurfaceIsTheProjectedShape holds the three members' signatures
// against the pinned contract's declarations.
func TestDisposalSurfaceIsTheProjectedShape(t *testing.T) {
	game := reflect.TypeOf((*Game)(nil))
	for name, signature := range map[string]string{
		"DisposeByNone":    "func(*framework.Game) error",
		"DisposeByBoolean": "func(*framework.Game, bool) error",
		"Finalize":         "func(*framework.Game) error",
	} {
		method, ok := game.MethodByName(name)
		if !ok {
			t.Fatalf("Game has no method %q", name)
		}
		if got := method.Type.String(); got != signature {
			t.Fatalf("Game.%s has signature %s, want %s", name, got, signature)
		}
	}
	// There is no OnDisposed, and inventing one is a verifier failure: the
	// reference invokes the delegate field directly from Dispose(bool).
	if _, ok := game.MethodByName("OnDisposed"); ok {
		t.Fatal("Game declares OnDisposed; the reference has no such member")
	}
}

// TestTheTwoDisposableSpellingsAreBothFoundAndNothingElseIs pins the Go
// projection of `component as IDisposable`.
func TestTheTwoDisposableSpellingsAreBothFoundAndNothingElseIs(t *testing.T) {
	if err := disposeGameComponent(&undisposableComponent{}); err != nil {
		t.Fatalf("a component with no disposal member reported %v, want nil", err)
	}
	two := &disposalComponent{}
	if err := disposeGameComponent(two); err != nil {
		t.Fatalf("two-overload spelling: %v", err)
	}
	if two.disposed != 1 {
		t.Fatalf("two-overload spelling disposed %d times", two.disposed)
	}
	one := &simpleDisposalComponent{}
	if err := disposeGameComponent(one); err != nil {
		t.Fatalf("single-overload spelling: %v", err)
	}
	if one.disposed != 1 {
		t.Fatalf("single-overload spelling disposed %d times", one.disposed)
	}
	// A nil component is skipped, exactly as `isinst` on null yields null.
	if err := disposeGameComponent(nil); err != nil {
		t.Fatalf("a nil component reported %v, want nil", err)
	}
}

// TestAGameComponentIsDisposedThroughItsOwnProjection is the end-to-end shape:
// Microsoft's own component type, in Components, disposed by Game.Dispose,
// removing itself and raising its own Disposed event on the way out.
func TestAGameComponentIsDisposedThroughItsOwnProjection(t *testing.T) {
	game := newDisposalGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	componentDisposed := 0
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		componentDisposed++
		return nil
	}); err != nil {
		t.Fatalf("component AddDisposedHandler: %v", err)
	}
	gameDisposed := 0
	if _, err := game.AddDisposedHandler(func(any, *EventArgs) error {
		gameDisposed++
		// The component has already left Components by the time the Game's own
		// event is raised: components are disposed before the raise site.
		if got := game.Components().Count(); got != 0 {
			t.Errorf("Game.Disposed observed %d components still present", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("game AddDisposedHandler: %v", err)
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if componentDisposed != 1 || gameDisposed != 1 {
		t.Fatalf("component raised %d times and Game %d times, want 1 each", componentDisposed, gameDisposed)
	}
}
