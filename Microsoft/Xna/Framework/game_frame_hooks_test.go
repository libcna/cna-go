package framework

import (
	"reflect"
	"testing"
)

// TestGameRunHooksAreAuthoritativeNoOps holds the two one-byte bodies. Each is
// `IL_0000: ret`, so each must do nothing at all -- not to the component
// engine, not to the pending queue, and not to the event lists.
func TestGameRunHooksAreAuthoritativeNoOps(t *testing.T) {
	game := newEventGame(t)
	component := &frameHookComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Components().Add: %v", err)
	}
	raises := 0
	if _, err := game.AddExitingHandler(func(any, *EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatalf("AddExitingHandler: %v", err)
	}
	if err := game.BeginRun(); err != nil {
		t.Fatalf("BeginRun: %v", err)
	}
	if err := game.EndRun(); err != nil {
		t.Fatalf("EndRun: %v", err)
	}
	if component.initialized != 0 || component.updates != 0 {
		t.Fatalf("a run hook touched the component engine: initialized=%d updates=%d",
			component.initialized, component.updates)
	}
	if raises != 0 {
		t.Fatalf("a run hook raised %d events, want 0", raises)
	}
	if game.Components().Count() != 1 {
		t.Fatalf("a run hook changed Components: count=%d", game.Components().Count())
	}
	if game.inRun {
		t.Fatal("BeginRun raised inRun; the reference raises it in RunGame, not in the virtual")
	}
}

// TestGameBeginDrawAnswersTrueWithNoManager pins the branch the reference takes
// when graphicsDeviceManager is null, which is the state CNA-Go always has.
// The Boolean is the frame's drawing decision, not a success flag, and it is a
// channel separate from the error.
func TestGameBeginDrawAnswersTrueWithNoManager(t *testing.T) {
	game := newEventGame(t)
	shouldDraw, err := game.BeginDraw()
	if err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if !shouldDraw {
		t.Fatal("BeginDraw answered false with no registered IGraphicsDeviceManager")
	}
	if err := game.EndDraw(); err != nil {
		t.Fatalf("EndDraw: %v", err)
	}
	// Repeating either one is not a state machine: neither body has state.
	for i := 0; i < 3; i++ {
		if again, againErr := game.BeginDraw(); !again || againErr != nil {
			t.Fatalf("BeginDraw repeat %d answered (%t, %v)", i, again, againErr)
		}
		if err := game.EndDraw(); err != nil {
			t.Fatalf("EndDraw repeat %d: %v", i, err)
		}
	}
}

// TestGameDrawHooksTouchNothing keeps the two draw hooks from acquiring managed
// side effects. Their whole bodies are the manager call and the logging event;
// with no manager they are observably empty.
func TestGameDrawHooksTouchNothing(t *testing.T) {
	game := newEventGame(t)
	component := &frameHookComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Components().Add: %v", err)
	}
	if err := GameBaseInitialize(game); err != nil {
		t.Fatalf("GameBaseInitialize: %v", err)
	}
	before := component.updates
	if _, err := game.BeginDraw(); err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if err := game.EndDraw(); err != nil {
		t.Fatalf("EndDraw: %v", err)
	}
	if component.updates != before {
		t.Fatalf("a draw hook ran the component loop: %d -> %d", before, component.updates)
	}
	if component.draws != 0 {
		t.Fatalf("a draw hook drew a component %d times; base Draw is a different member", component.draws)
	}
}

// TestGameFrameHooksRefuseAnUnconstructedGame covers the family's one Go-only
// failure. CLR always ran a constructor; Go lets a consumer produce a Game that
// did not, and every public entry point that takes one refuses it.
func TestGameFrameHooksRefuseAnUnconstructedGame(t *testing.T) {
	zero := &Game{}
	for name, call := range map[string]func(*Game) error{
		"BeginRun": (*Game).BeginRun,
		"EndRun":   (*Game).EndRun,
		"EndDraw":  (*Game).EndDraw,
	} {
		if err := call(zero); err == nil {
			t.Fatalf("%s(&Game{}) reported no error", name)
		}
	}
	shouldDraw, err := zero.BeginDraw()
	if err == nil {
		t.Fatal("BeginDraw(&Game{}) reported no error")
	}
	if shouldDraw {
		t.Fatal("a refused BeginDraw must not also admit the frame")
	}
}

// TestGameFrameHooksAreNotGameCallbacksMembers is the structural half of the
// promise this milestone makes to every existing consumer: the mandatory
// override contract still has exactly its five members, and none of the four
// frame hooks joined it.
func TestGameFrameHooksAreNotGameCallbacksMembers(t *testing.T) {
	contract := reflect.TypeOf((*GameCallbacks)(nil)).Elem()
	if contract.NumMethod() != 5 {
		t.Fatalf("GameCallbacks has %d methods, want exactly 5", contract.NumMethod())
	}
	want := map[string]bool{
		"Initialize": true, "LoadContent": true, "Update": true,
		"Draw": true, "UnloadContent": true,
	}
	for i := 0; i < contract.NumMethod(); i++ {
		name := contract.Method(i).Name
		if !want[name] {
			t.Fatalf("GameCallbacks gained a member %q", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("GameCallbacks lost members %v", want)
	}
	// And the four hooks are methods on Game, where Microsoft declared them.
	game := reflect.TypeOf((*Game)(nil))
	for name, signature := range map[string]string{
		"BeginRun":  "func(*framework.Game) error",
		"EndRun":    "func(*framework.Game) error",
		"BeginDraw": "func(*framework.Game) (bool, error)",
		"EndDraw":   "func(*framework.Game) error",
	} {
		method, ok := game.MethodByName(name)
		if !ok {
			t.Fatalf("Game has no method %q", name)
		}
		if got := method.Type.String(); got != signature {
			t.Fatalf("Game.%s has signature %s, want %s", name, got, signature)
		}
	}
}

// TestNoGameBaseAdapterWasAddedForAFrameHook holds the Foundation-31 closure
// rule from the other side. The GameBase... family is keyed by the GameCallbacks
// members, and none of these four is one, so inventing GameBaseBeginRun and
// friends because the names look symmetrical is exactly the mistake the
// registry exists to prevent.
func TestNoGameBaseAdapterWasAddedForAFrameHook(t *testing.T) {
	// The five that exist, asserted at compile time.
	var (
		_ func(*Game) error           = GameBaseInitialize
		_ func(*Game) error           = GameBaseLoadContent
		_ func(*Game) error           = GameBaseUnloadContent
		_ func(*Game, GameTime) error = GameBaseUpdate
		_ func(*Game, GameTime) error = GameBaseDraw
	)
	// A sixth would have to be declared to be referenced, and nothing here
	// references one. The registry-driven verifier rule is what enforces this
	// across the package; this test records the intent next to the members.
}

// frameHookComponent is a minimal IUpdateable/IDrawable conformer used to prove
// the four hooks touch no managed state.
type frameHookComponent struct {
	initialized int
	updates     int
	draws       int

	enabledChanged     EventSource[*EventArgs]
	updateOrderChanged EventSource[*EventArgs]
	visibleChanged     EventSource[*EventArgs]
	drawOrderChanged   EventSource[*EventArgs]
}

func (c *frameHookComponent) Initialize() error  { c.initialized++; return nil }
func (c *frameHookComponent) Update(GameTime)    { c.updates++ }
func (c *frameHookComponent) Draw(GameTime)      { c.draws++ }
func (c *frameHookComponent) Enabled() bool      { return true }
func (c *frameHookComponent) UpdateOrder() int32 { return 0 }
func (c *frameHookComponent) Visible() bool      { return true }
func (c *frameHookComponent) DrawOrder() int32   { return 0 }

func (c *frameHookComponent) AddEnabledChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.enabledChanged.Add(h)
}
func (c *frameHookComponent) RemoveEnabledChangedHandler(s EventSubscription) error {
	return c.enabledChanged.Remove(s)
}
func (c *frameHookComponent) AddUpdateOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.updateOrderChanged.Add(h)
}
func (c *frameHookComponent) RemoveUpdateOrderChangedHandler(s EventSubscription) error {
	return c.updateOrderChanged.Remove(s)
}
func (c *frameHookComponent) AddVisibleChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.visibleChanged.Add(h)
}
func (c *frameHookComponent) RemoveVisibleChangedHandler(s EventSubscription) error {
	return c.visibleChanged.Remove(s)
}
func (c *frameHookComponent) AddDrawOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.drawOrderChanged.Add(h)
}
func (c *frameHookComponent) RemoveDrawOrderChangedHandler(s EventSubscription) error {
	return c.drawOrderChanged.Remove(s)
}

var (
	_ IGameComponent = (*frameHookComponent)(nil)
	_ IUpdateable    = (*frameHookComponent)(nil)
	_ IDrawable      = (*frameHookComponent)(nil)
)
