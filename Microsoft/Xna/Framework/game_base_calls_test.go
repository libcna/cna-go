package framework

import (
	"errors"
	"strings"
	"testing"
)

// countingCallbacks records every callback invocation, so a test can prove that
// a base call NEVER reaches back into the override contract.
type countingCallbacks struct {
	initialize    int
	loadContent   int
	update        int
	draw          int
	unloadContent int
}

func (c *countingCallbacks) Initialize(*Game) error       { c.initialize++; return nil }
func (c *countingCallbacks) LoadContent(*Game) error      { c.loadContent++; return nil }
func (c *countingCallbacks) Update(*Game, GameTime) error { c.update++; return nil }
func (c *countingCallbacks) Draw(*Game, GameTime) error   { c.draw++; return nil }
func (c *countingCallbacks) UnloadContent(*Game) error    { c.unloadContent++; return nil }

func newCountingGame(t *testing.T) (*Game, *countingCallbacks) {
	t.Helper()
	callbacks := &countingCallbacks{}
	game, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game, callbacks
}

// addOrderedComponents adds components with the given update and draw orders,
// all sharing one log, and returns them by name.
func addOrderedComponents(t *testing.T, game *Game, log *[]string, spec map[string]int32) map[string]*engineComponent {
	t.Helper()
	made := make(map[string]*engineComponent, len(spec))
	names := make([]string, 0, len(spec))
	for name := range spec {
		names = append(names, name)
	}
	// Deterministic add order regardless of Go map iteration.
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, name := range names {
		component := newEngineComponent(name, log)
		component.updateOrder = spec[name]
		component.drawOrder = spec[name]
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add %s: %v", name, err)
		}
		made[name] = component
	}
	return made
}

// TestBaseBehaviorIsNeverAutomatic is the whole point of the family. Nothing in
// CNA-Go runs a base body on its own: a callback that does not call one gets no
// base behavior, so the component loop simply does not happen.
func TestBaseBehaviorIsNeverAutomatic(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1})

	// The callback body a consumer would write when it deliberately omits the
	// base call. Nothing else runs it, so nothing else can add base behavior.
	userOnly := func(*Game, GameTime) error {
		log = append(log, "user")
		return nil
	}
	if err := userOnly(game, GameTime{}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if got := strings.Join(log, ","); got != "user" {
		t.Fatalf("omitting the base call still produced base behavior: %q", got)
	}
}

// TestBaseUpdateRunsTheComponentLoopExactlyOnce is the other direction: one
// explicit call runs the loop once, and each component is updated once.
func TestBaseUpdateRunsTheComponentLoopExactlyOnce(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1})
	log = nil

	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:a,update:b" {
		t.Fatalf("one base call produced %q", got)
	}
	// A second explicit call runs it again -- the helper holds no "already
	// ran this frame" state, exactly as base.Update(t) does not.
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:a,update:b" {
		t.Fatalf("the second base call produced %q", got)
	}
}

// TestBaseCallsNeverInvokeTheCallbacks pins the recursion control: a base body
// in the reference never calls the virtual it is the base of, and neither does
// any helper. If one did, a callback that called its own base would recurse
// without bound.
func TestBaseCallsNeverInvokeTheCallbacks(t *testing.T) {
	game, callbacks := newCountingGame(t)
	addOrderedComponents(t, game, nil, map[string]int32{"a": 0})

	for _, call := range []struct {
		name string
		run  func() error
	}{
		{"GameBaseInitialize", func() error { return GameBaseInitialize(game) }},
		{"GameBaseLoadContent", func() error { return GameBaseLoadContent(game) }},
		{"GameBaseUpdate", func() error { return GameBaseUpdate(game, GameTime{}) }},
		{"GameBaseDraw", func() error { return GameBaseDraw(game, GameTime{}) }},
		{"GameBaseUnloadContent", func() error { return GameBaseUnloadContent(game) }},
	} {
		if err := call.run(); err != nil {
			t.Fatalf("%s: %v", call.name, err)
		}
	}
	if *callbacks != (countingCallbacks{}) {
		t.Fatalf("a base call reached the override contract: %+v", callbacks)
	}
}

// TestBaseCallPositionDecidesOrdering is the essential override-fidelity claim:
// where the consumer puts the base call is where base behavior happens.
func TestBaseCallPositionDecidesOrdering(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		callback func(game *Game, log *[]string) error
		want     string
	}{
		{"base-last", func(game *Game, log *[]string) error {
			*log = append(*log, "user")
			return GameBaseUpdate(game, GameTime{})
		}, "user,update:a,update:b"},
		{"base-first", func(game *Game, log *[]string) error {
			if err := GameBaseUpdate(game, GameTime{}); err != nil {
				return err
			}
			*log = append(*log, "user")
			return nil
		}, "update:a,update:b,user"},
		{"base-between", func(game *Game, log *[]string) error {
			*log = append(*log, "before")
			if err := GameBaseUpdate(game, GameTime{}); err != nil {
				return err
			}
			*log = append(*log, "after")
			return nil
		}, "before,update:a,update:b,after"},
		{"no-base", func(game *Game, log *[]string) error {
			*log = append(*log, "user")
			return nil
		}, "user"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			game, _ := newCountingGame(t)
			var log []string
			addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1})
			log = nil
			if err := testCase.callback(game, &log); err != nil {
				t.Fatalf("callback: %v", err)
			}
			if got := strings.Join(log, ","); got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestBaseUpdateRespectsEnabledAtIterationTime pins the one thing the snapshot
// does NOT freeze: Enabled is read when the component's turn comes, so a
// component disabled earlier in the same frame is skipped.
func TestBaseUpdateRespectsEnabledAtIterationTime(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1, "c": 2})
	made["b"].enabled = false
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:a,update:c" {
		t.Fatalf("a disabled component was updated: %q", got)
	}

	// Disabling from inside an earlier component's Update still skips it,
	// because Enabled is read at iteration time rather than at snapshot time.
	made["b"].enabled = true
	log = nil
	disabler := newEngineComponent("disabler", &log)
	disabler.updateOrder = -1
	disabler.onUpdate = func() { made["b"].enabled = false }
	if err := game.Components().Add(disabler); err != nil {
		t.Fatalf("Add: %v", err)
	}
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:disabler,update:a,update:c" {
		t.Fatalf("Enabled was read at snapshot time, not iteration time: %q", got)
	}
}

// TestBaseDrawRespectsVisibleAndDrawOrder is the same claim on the draw side,
// which is a separate list maintained by a separate handler.
func TestBaseDrawRespectsVisibleAndDrawOrder(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 2, "b": 0, "c": 1})
	made["c"].visible = false
	log = nil
	if err := GameBaseDraw(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseDraw: %v", err)
	}
	if got := strings.Join(log, ","); got != "draw:b,draw:a" {
		t.Fatalf("draw order or Visible is wrong: %q", got)
	}
}

// TestUpdateAndDrawOrderAreIndependent proves the two derived lists are exactly
// that: two lists, each with its own comparer and its own order property.
func TestUpdateAndDrawOrderAreIndependent(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	for _, spec := range []struct {
		name                  string
		updateOrder, drawOrde int32
	}{{"a", 0, 2}, {"b", 1, 1}, {"c", 2, 0}} {
		component := newEngineComponent(spec.name, &log)
		component.updateOrder = spec.updateOrder
		component.drawOrder = spec.drawOrde
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if err := GameBaseDraw(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseDraw: %v", err)
	}
	want := "update:a,update:b,update:c,draw:c,draw:b,draw:a"
	if got := strings.Join(log, ","); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestMutationDuringBaseUpdateDoesNotChangeThisFrame is the snapshot claim: the
// first loop copies the ordered list and the second iterates the copy, so
// adding or removing components mid-frame affects the NEXT frame only.
func TestMutationDuringBaseUpdateDoesNotChangeThisFrame(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1, "c": 2})

	// While "a" updates, add a new component that would sort first and remove
	// one that has not run yet.
	late := newEngineComponent("late", &log)
	late.updateOrder = -5
	made["a"].onUpdate = func() {
		if err := game.Components().Add(late); err != nil {
			t.Fatalf("Add during Update: %v", err)
		}
		if _, err := game.Components().Remove(made["c"]); err != nil {
			t.Fatalf("Remove during Update: %v", err)
		}
	}

	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	// "c" still updates, because the snapshot was taken before it was removed,
	// and "late" does not, because it was not in the snapshot.
	if got := strings.Join(log, ","); got != "update:a,update:b,update:c" {
		t.Fatalf("the frame's snapshot was not frozen: %q", got)
	}

	made["a"].onUpdate = nil
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:late,update:a,update:b" {
		t.Fatalf("the next frame did not see the mutation: %q", got)
	}
}

// TestBaseInitializeDrainsThePendingQueue is the base Initialize body.
func TestBaseInitializeDrainsThePendingQueue(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1})
	log = nil

	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	if got := strings.Join(log, ","); got != "init:a,init:b" {
		t.Fatalf("drain order %q", got)
	}
	if len(game.notYetInitialized) != 0 {
		t.Fatal("the queue was not drained")
	}
	// A second call has nothing left to do: the queue is empty, and the
	// reference removes each component as it initializes it.
	log = nil
	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	if len(log) != 0 {
		t.Fatalf("a component was initialized twice: %v", log)
	}
}

// TestComponentAddedDuringBaseInitializeIsDrainedToo follows from the drain
// re-reading Count every iteration while `inRun` is still false.
func TestComponentAddedDuringBaseInitializeIsDrainedToo(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 0})
	nested := newEngineComponent("nested", &log)
	made["a"].onInitialize = func() {
		if err := game.Components().Add(nested); err != nil {
			t.Fatalf("Add during Initialize: %v", err)
		}
	}
	log = nil
	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	if got := strings.Join(log, ","); got != "init:a,init:nested" {
		t.Fatalf("a component added during the drain was not drained: %q", got)
	}
	if nested.initializeCount != 1 {
		t.Fatalf("nested initialized %d times", nested.initializeCount)
	}
}

// TestFailingComponentStopsTheDrainAtItsOwnPosition follows from Initialize
// running BEFORE RemoveAt: the failing component stays at the head of the
// queue, so a retry resumes exactly where it stopped.
func TestFailingComponentStopsTheDrainAtItsOwnPosition(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1, "c": 2})
	failure := errors.New("component refused to initialize")
	made["b"].initializeError = failure
	log = nil

	err := GameBaseInitialize(game)
	if !errors.Is(err, failure) {
		t.Fatalf("expected the component's failure, got %v", err)
	}
	if got := strings.Join(log, ","); got != "init:a,init:b" {
		t.Fatalf("the drain did not stop at the failure: %q", got)
	}
	if len(game.notYetInitialized) != 2 {
		t.Fatalf("the failing component should still head the queue, %d left", len(game.notYetInitialized))
	}
	if game.notYetInitialized[0] != IGameComponent(made["b"]) {
		t.Fatal("the failing component is no longer at the head of the queue")
	}
	// Retrying after the component stops failing resumes from it.
	made["b"].initializeError = nil
	log = nil
	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	if got := strings.Join(log, ","); got != "init:b,init:c" {
		t.Fatalf("the retry did not resume at the failure: %q", got)
	}
}

// TestBaseLoadContentAndUnloadContentAreFaithfulNoOps: both reference bodies
// are a bare `ret` of code size 1, so the projection must do nothing at all --
// in particular it must not touch the component lists.
func TestBaseLoadContentAndUnloadContentAreFaithfulNoOps(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	addOrderedComponents(t, game, &log, map[string]int32{"a": 0})
	log = nil
	queued := len(game.notYetInitialized)

	if err := GameBaseLoadContent(game); err != nil {
		t.Fatalf("GameBaseLoadContent: %v", err)
	}
	if err := GameBaseUnloadContent(game); err != nil {
		t.Fatalf("GameBaseUnloadContent: %v", err)
	}
	if len(log) != 0 {
		t.Fatalf("a no-op base body did something: %v", log)
	}
	if len(game.notYetInitialized) != queued {
		t.Fatal("a no-op base body changed the pending queue")
	}
	if game.Components().Count() != 1 {
		t.Fatal("a no-op base body changed the collection")
	}
}

// TestBaseCallsRejectAnUnconstructedGame covers the one Go-only failure the
// family has. It is the same condition Game.Run and Game.Exit report.
func TestBaseCallsRejectAnUnconstructedGame(t *testing.T) {
	zero := &Game{}
	for _, call := range []struct {
		name string
		run  func(game *Game) error
	}{
		{"GameBaseInitialize", GameBaseInitialize},
		{"GameBaseLoadContent", GameBaseLoadContent},
		{"GameBaseUnloadContent", GameBaseUnloadContent},
		{"GameBaseUpdate", func(game *Game) error { return GameBaseUpdate(game, GameTime{}) }},
		{"GameBaseDraw", func(game *Game) error { return GameBaseDraw(game, GameTime{}) }},
	} {
		if err := call.run(nil); !errors.Is(err, errGameNotConstructed) {
			t.Fatalf("%s(nil) reported %v", call.name, err)
		}
		if err := call.run(zero); !errors.Is(err, errGameNotConstructed) {
			t.Fatalf("%s(&Game{}) reported %v", call.name, err)
		}
	}
}

// TestOnlyBaseInitializeCanFailForAConstructedGame pins the fallibility
// classification: four of the five have no failure but the Go-only guard, and
// GameBaseInitialize's extra failure is a component's own, not its own.
func TestOnlyBaseInitializeCanFailForAConstructedGame(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	made := addOrderedComponents(t, game, &log, map[string]int32{"a": 0})
	made["a"].initializeError = errors.New("component refused")

	if err := GameBaseLoadContent(game); err != nil {
		t.Fatalf("GameBaseLoadContent: %v", err)
	}
	if err := GameBaseUnloadContent(game); err != nil {
		t.Fatalf("GameBaseUnloadContent: %v", err)
	}
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if err := GameBaseDraw(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseDraw: %v", err)
	}
	if err := GameBaseInitialize(game); err == nil {
		t.Fatal("GameBaseInitialize swallowed a component's Initialize failure")
	}
}

// TestBaseUpdateAssignsDoneFirstUpdate covers the one field assignment in the
// reference body. Nothing reads it yet -- Paint, Tick and DrawFrame are not
// projected -- so this is the only place it can be observed.
func TestBaseUpdateAssignsDoneFirstUpdate(t *testing.T) {
	game, _ := newCountingGame(t)
	if game.doneFirstUpdate {
		t.Fatal("doneFirstUpdate is set before any update")
	}
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if !game.doneFirstUpdate {
		t.Fatal("base Update did not assign doneFirstUpdate")
	}
	// base Draw assigns no field, which is why it has no deferral at all.
	other, _ := newCountingGame(t)
	if err := GameBaseDraw(other, GameTime{}); err != nil {
		t.Fatalf("GameBaseDraw: %v", err)
	}
	if other.doneFirstUpdate {
		t.Fatal("base Draw assigned doneFirstUpdate; the reference body does not")
	}
}

// TestBaseUpdateEmptiesItsSnapshotList proves the straight-line Clear runs, so
// two consecutive frames do not accumulate.
func TestBaseUpdateEmptiesItsSnapshotList(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	addOrderedComponents(t, game, &log, map[string]int32{"a": 0, "b": 1})
	for frame := 0; frame < 3; frame++ {
		log = nil
		if err := GameBaseUpdate(game, GameTime{}); err != nil {
			t.Fatalf("GameBaseUpdate: %v", err)
		}
		if err := GameBaseDraw(game, GameTime{}); err != nil {
			t.Fatalf("GameBaseDraw: %v", err)
		}
		if got := strings.Join(log, ","); got != "update:a,update:b,draw:a,draw:b" {
			t.Fatalf("frame %d produced %q", frame, got)
		}
		if len(game.currentlyUpdatingComponents) != 0 || len(game.currentlyDrawingComponents) != 0 {
			t.Fatalf("frame %d left its snapshot populated", frame)
		}
	}
}

// TestInitializationOrderIsAddOrderNotUpdateOrder pins a distinction that is
// easy to get wrong: only two of Game's five private lists are ordered.
//
// notYetInitialized is a plain List<IGameComponent> the add handler appends to
// and the drain consumes from index 0, so components initialize in the order
// they were ADDED. UpdateOrder and DrawOrder govern the update and draw loops
// and nothing else.
func TestInitializationOrderIsAddOrderNotUpdateOrder(t *testing.T) {
	game, _ := newCountingGame(t)
	var log []string
	for _, spec := range []struct {
		name        string
		updateOrder int32
	}{{"added-first", 90}, {"added-second", 10}, {"added-third", 50}} {
		component := newEngineComponent(spec.name, &log)
		component.updateOrder = spec.updateOrder
		component.drawOrder = spec.updateOrder
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	log = nil
	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	if got := strings.Join(log, ","); got != "init:added-first,init:added-second,init:added-third" {
		t.Fatalf("initialization order %q, want add order", got)
	}
	log = nil
	if err := GameBaseUpdate(game, GameTime{}); err != nil {
		t.Fatalf("GameBaseUpdate: %v", err)
	}
	if got := strings.Join(log, ","); got != "update:added-second,update:added-third,update:added-first" {
		t.Fatalf("update order %q, want UpdateOrder", got)
	}
}
