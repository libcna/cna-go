package framework

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

// timeoutAfterOneSecond bounds the reentrancy test, so a deadlock fails the
// test instead of hanging the suite.
func timeoutAfterOneSecond() <-chan time.Time { return time.After(time.Second) }

// TestGameComponentConstructorDefaults covers the 21-byte constructor: Enabled
// starts true, UpdateOrder starts zero, the Game is stored as handed over, and
// nothing is validated.
func TestGameComponentConstructorDefaults(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if !component.Enabled() {
		t.Fatal("Enabled must default to true; the constructor's field initializer stores 1")
	}
	if component.UpdateOrder() != 0 {
		t.Fatalf("UpdateOrder must default to 0, got %d", component.UpdateOrder())
	}
	if component.Game() != game {
		t.Fatal("the Game reference was not stored as handed over")
	}
	// The reference does not null-check its Game argument.
	orphan := NewGameComponent(nil)
	if orphan == nil {
		t.Fatal("a nil Game must be accepted; the constructor validates nothing")
	}
	if orphan.Game() != nil {
		t.Fatal("a nil Game must be stored as nil")
	}
	if !orphan.Enabled() {
		t.Fatal("a component with no Game still starts Enabled")
	}
}

// TestGameComponentBaseMethodsAreNoOps: Initialize and Update are each a bare
// `ret` of code size 1 in the reference.
func TestGameComponentBaseMethodsAreNoOps(t *testing.T) {
	component := NewGameComponent(nil)
	if err := component.Initialize(); err != nil {
		t.Fatalf("base Initialize must not fail: %v", err)
	}
	component.Update(GameTime{})
	component.Finalize()
	if !component.Enabled() || component.UpdateOrder() != 0 {
		t.Fatal("a no-op body changed the component's state")
	}
}

// TestGameComponentSettersSuppressAnUnchangedValue is the compare-store-announce
// shape, and the suppression is what keeps Game's engine from re-placing a
// component whose order did not actually change.
func TestGameComponentSettersSuppressAnUnchangedValue(t *testing.T) {
	component := NewGameComponent(nil)
	var raised []string
	if _, err := component.AddEnabledChangedHandler(func(any, *EventArgs) error {
		raised = append(raised, "enabled")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddUpdateOrderChangedHandler(func(any, *EventArgs) error {
		raised = append(raised, "order")
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Assigning the value it already has announces nothing.
	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := component.SetUpdateOrder(0); err != nil {
		t.Fatal(err)
	}
	if len(raised) != 0 {
		t.Fatalf("a suppressed assignment announced: %v", raised)
	}

	if err := component.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := component.SetUpdateOrder(7); err != nil {
		t.Fatal(err)
	}
	if strings.Join(raised, ",") != "enabled,order" {
		t.Fatalf("changed assignments announced %v", raised)
	}
	if component.Enabled() || component.UpdateOrder() != 7 {
		t.Fatal("the new values were not stored")
	}
	// And back again, which is a change too.
	raised = nil
	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(raised, ",") != "enabled" {
		t.Fatalf("changing back announced %v", raised)
	}
}

// TestGameComponentSetterStoresBeforeItAnnounces: the handler observes the NEW
// value, because the store precedes the virtual call.
func TestGameComponentSetterStoresBeforeItAnnounces(t *testing.T) {
	component := NewGameComponent(nil)
	var observed []string
	if _, err := component.AddUpdateOrderChangedHandler(func(sender any, args *EventArgs) error {
		observed = append(observed, strconv.Itoa(int(sender.(*GameComponent).UpdateOrder())))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.SetUpdateOrder(42); err != nil {
		t.Fatal(err)
	}
	if strings.Join(observed, ",") != "42" {
		t.Fatalf("the handler observed %v; the store must precede the announcement", observed)
	}
}

// TestGameComponentOnMethodsIgnoreTheirSenderArgument is the quirk Game's
// engine depends on: OnEnabledChanged and OnUpdateOrderChanged accept a sender,
// ignore it, and raise with `this`.
func TestGameComponentOnMethodsIgnoreTheirSenderArgument(t *testing.T) {
	component := NewGameComponent(nil)
	decoy := NewGameComponent(nil)
	var senders []any
	record := func(sender any, args *EventArgs) error {
		senders = append(senders, sender)
		return nil
	}
	if _, err := component.AddEnabledChangedHandler(record); err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddUpdateOrderChangedHandler(record); err != nil {
		t.Fatal(err)
	}

	// Call the raise sites directly with a deliberately wrong sender.
	if err := component.OnEnabledChanged(decoy, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if err := component.OnUpdateOrderChanged(decoy, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if len(senders) != 2 {
		t.Fatalf("expected two raises, got %d", len(senders))
	}
	for i, sender := range senders {
		if sender != any(component) {
			t.Fatalf("raise %d used the sender argument instead of the component itself", i)
		}
	}
}

// TestGameComponentEventArgsAreTheSharedEmptyIdentity: every raise site loads
// System.EventArgs::Empty, so the args a handler receives are that identity.
func TestGameComponentEventArgsAreTheSharedEmptyIdentity(t *testing.T) {
	component := NewGameComponent(nil)
	var got []*EventArgs
	record := func(sender any, args *EventArgs) error {
		got = append(got, args)
		return nil
	}
	if _, err := component.AddEnabledChangedHandler(record); err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddDisposedHandler(record); err != nil {
		t.Fatal(err)
	}
	if err := component.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two raises, got %d", len(got))
	}
	for i, args := range got {
		if args != EventArgsEmpty() {
			t.Fatalf("raise %d did not carry the shared EventArgs.Empty identity", i)
		}
	}
}

// TestDisposingAComponentRemovesItFromGameComponents is the member that blocked
// this type for five milestones: Dispose(bool) runs
// Game.Components.Remove(this).
func TestDisposingAComponentRemovesItFromGameComponents(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if game.Components().Count() != 1 {
		t.Fatal("setup failed")
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if game.Components().Count() != 0 {
		t.Fatal("Dispose did not remove the component from Game.Components")
	}
	// And Game's engine untracked it, because Remove reaches RemoveItem, which
	// announces, which Game's own handler consumes.
	if len(game.updateableComponents) != 0 {
		t.Fatal("Dispose left the component tracked in the update list")
	}
	if len(game.notYetInitialized) != 0 {
		t.Fatal("Dispose left the component queued for initialization")
	}
}

// TestDisposeRemovesBeforeItAnnounces pins the order inside the critical
// section: a Disposed handler sees a Game that no longer contains the
// component.
func TestDisposeRemovesBeforeItAnnounces(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	observedCount := int32(-1)
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		observedCount = game.Components().Count()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if observedCount != 0 {
		t.Fatalf("the Disposed handler observed Count=%d; removal must precede the announcement", observedCount)
	}
}

// TestDisposeIsNotIdempotent: the reference discards Remove's boolean and
// raises Disposed unconditionally, so a second Dispose raises again.
func TestDisposeIsNotIdempotent(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	raises := 0
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := component.DisposeByNone(); err != nil {
			t.Fatalf("Dispose %d: %v", i, err)
		}
	}
	if raises != 3 {
		t.Fatalf("Disposed was raised %d times; the reference raises on every Dispose", raises)
	}
	if game.Components().Count() != 0 {
		t.Fatal("repeated Dispose changed the collection after the first")
	}
}

// TestDisposeWithNoGameSkipsTheRemovalButStillAnnounces: the reference's
// `if (Game != null)` guards only the removal.
func TestDisposeWithNoGameSkipsTheRemovalButStillAnnounces(t *testing.T) {
	component := NewGameComponent(nil)
	raises := 0
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raises != 1 {
		t.Fatalf("Disposed raised %d times", raises)
	}
}

// TestDisposeByBooleanFalseDoesNothing: the whole body is behind
// `if (!disposing) return`, which is also why Finalize is observably a no-op.
func TestDisposeByBooleanFalseDoesNothing(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	raises := 0
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByBoolean(false); err != nil {
		t.Fatalf("Dispose(false): %v", err)
	}
	component.Finalize()
	if raises != 0 {
		t.Fatal("Dispose(false) announced")
	}
	if game.Components().Count() != 1 {
		t.Fatal("Dispose(false) changed the collection")
	}
}

// TestDisposeIsReentrantRatherThanDeadlocking is the concurrency projection.
// CLR's Monitor recurses on this path; a plain Go Lock would deadlock, so the
// projection uses TryLock and reentry proceeds.
func TestDisposeIsReentrantRatherThanDeadlocking(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	depth, raises := 0, 0
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error {
		raises++
		if depth == 0 {
			depth++
			// Reentry from inside the critical section. CLR's Monitor is
			// reentrant per thread and recurses here; this must not deadlock.
			return component.DisposeByNone()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- component.DisposeByNone() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("reentrant Dispose: %v", err)
		}
	case <-timeoutAfterOneSecond():
		t.Fatal("reentrant Dispose deadlocked")
	}
	if raises != 2 {
		t.Fatalf("Disposed raised %d times; reentry raises again exactly as the reference recursion does", raises)
	}
}

// TestGameComponentEventsAreIndependent: three CLR events, three registration
// lists, and removing one registration leaves the others alone.
func TestGameComponentEventsAreIndependent(t *testing.T) {
	component := NewGameComponent(nil)
	var seen []string
	handler := func(label string) EventHandler[*EventArgs] {
		return func(any, *EventArgs) error {
			seen = append(seen, label)
			return nil
		}
	}
	enabledToken, err := component.AddEnabledChangedHandler(handler("enabled"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddUpdateOrderChangedHandler(handler("order")); err != nil {
		t.Fatal(err)
	}
	if _, err := component.AddDisposedHandler(handler("disposed")); err != nil {
		t.Fatal(err)
	}

	if err := component.RemoveEnabledChangedHandler(enabledToken); err != nil {
		t.Fatal(err)
	}
	if err := component.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err := component.SetUpdateOrder(1); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(seen, ",") != "order,disposed" {
		t.Fatalf("events are not independent: %v", seen)
	}
}

// TestGameComponentDrivesTheEngineEndToEnd is the milestone's point: the class
// and the engine were built separately from the same IL and must now work
// together with no adapter between them.
func TestGameComponentDrivesTheEngineEndToEnd(t *testing.T) {
	game, _ := newCountingGame(t)
	first := NewGameComponent(game)
	second := NewGameComponent(game)
	if err := first.SetUpdateOrder(10); err != nil {
		t.Fatal(err)
	}
	for _, component := range []*GameComponent{first, second} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	// second has order 0 and sorts first.
	if len(game.updateableComponents) != 2 {
		t.Fatalf("engine tracked %d components", len(game.updateableComponents))
	}
	if game.updateableComponents[0].component != IUpdateable(second) {
		t.Fatal("the ordered list is not in UpdateOrder")
	}
	// Changing an order re-places it, through the class's own event.
	if err := second.SetUpdateOrder(99); err != nil {
		t.Fatal(err)
	}
	if game.updateableComponents[0].component != IUpdateable(first) {
		t.Fatal("the order change did not re-place the component")
	}
	// base Initialize drains both, base Update runs both, and neither fails.
	if err := GameBaseInitialize(game); err != nil {
		t.Fatal(err)
	}
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatal(err)
	}
	// A GameComponent is not IDrawable, so nothing is tracked for drawing.
	if len(game.drawableComponents) != 0 {
		t.Fatal("GameComponent must not be tracked as drawable; it does not implement IDrawable")
	}
	// Disposing removes it and untracks it.
	if err := first.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if game.Components().Count() != 1 || len(game.updateableComponents) != 1 {
		t.Fatal("Dispose did not remove and untrack the component")
	}
}

// TestGameComponentIsNotDrawable pins the type test the engine performs: the
// reference class implements IGameComponent and IUpdateable and NOT IDrawable,
// which is DrawableGameComponent's job.
func TestGameComponentIsNotDrawable(t *testing.T) {
	var component any = NewGameComponent(nil)
	if _, isDrawable := component.(IDrawable); isDrawable {
		t.Fatal("GameComponent must not satisfy IDrawable")
	}
	if _, isComponent := component.(IGameComponent); !isComponent {
		t.Fatal("GameComponent must satisfy IGameComponent")
	}
	if _, isUpdateable := component.(IUpdateable); !isUpdateable {
		t.Fatal("GameComponent must satisfy IUpdateable")
	}
}

// TestFailingDisposedHandlerReportsItsFailure: the raise's error is not
// discarded, unlike Remove's boolean, which the reference pops.
func TestFailingDisposedHandlerReportsItsFailure(t *testing.T) {
	game, _ := newCountingGame(t)
	component := NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("handler refused")
	if _, err := component.AddDisposedHandler(func(any, *EventArgs) error { return failure }); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); !errors.Is(err, failure) {
		t.Fatalf("expected the handler's failure, got %v", err)
	}
	// The removal had already happened, because it precedes the announcement.
	if game.Components().Count() != 0 {
		t.Fatal("the failed announcement rolled back the removal")
	}
}
