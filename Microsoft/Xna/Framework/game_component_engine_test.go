package framework

import (
	"errors"
	"reflect"
	"testing"
)

// engineComponent is an in-package test conformer of the three component
// contracts. It is deliberately NOT GameComponent: this file qualifies the
// engine that Game keeps around Components, and the engine must work for any
// IGameComponent, not only for the one class the reference ships.
type engineComponent struct {
	name        string
	enabled     bool
	visible     bool
	updateOrder int32
	drawOrder   int32

	initializeCount int
	initializeError error

	log *[]string

	enabledChanged     EventSource[*EventArgs]
	updateOrderChanged EventSource[*EventArgs]
	visibleChanged     EventSource[*EventArgs]
	drawOrderChanged   EventSource[*EventArgs]
}

func newEngineComponent(name string, log *[]string) *engineComponent {
	return &engineComponent{name: name, enabled: true, visible: true, log: log}
}

func (c *engineComponent) Initialize() error {
	c.initializeCount++
	c.record("init:" + c.name)
	return c.initializeError
}

func (c *engineComponent) record(entry string) {
	if c.log != nil {
		*c.log = append(*c.log, entry)
	}
}

func (c *engineComponent) Enabled() bool      { return c.enabled }
func (c *engineComponent) UpdateOrder() int32 { return c.updateOrder }
func (c *engineComponent) Visible() bool      { return c.visible }
func (c *engineComponent) DrawOrder() int32   { return c.drawOrder }

func (c *engineComponent) Update(GameTime) { c.record("update:" + c.name) }
func (c *engineComponent) Draw(GameTime)   { c.record("draw:" + c.name) }

func (c *engineComponent) AddEnabledChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.enabledChanged.Add(h)
}
func (c *engineComponent) RemoveEnabledChangedHandler(s EventSubscription) error {
	return c.enabledChanged.Remove(s)
}
func (c *engineComponent) AddUpdateOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.updateOrderChanged.Add(h)
}
func (c *engineComponent) RemoveUpdateOrderChangedHandler(s EventSubscription) error {
	return c.updateOrderChanged.Remove(s)
}
func (c *engineComponent) AddVisibleChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.visibleChanged.Add(h)
}
func (c *engineComponent) RemoveVisibleChangedHandler(s EventSubscription) error {
	return c.visibleChanged.Remove(s)
}
func (c *engineComponent) AddDrawOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.drawOrderChanged.Add(h)
}
func (c *engineComponent) RemoveDrawOrderChangedHandler(s EventSubscription) error {
	return c.drawOrderChanged.Remove(s)
}

// SetUpdateOrder reproduces GameComponent::set_UpdateOrder: suppress when
// unchanged, store, then raise with the component itself as sender.
func (c *engineComponent) SetUpdateOrder(value int32) error {
	if c.updateOrder == value {
		return nil
	}
	c.updateOrder = value
	return c.updateOrderChanged.Raise(c, EventArgsEmpty())
}

func (c *engineComponent) SetDrawOrder(value int32) error {
	if c.drawOrder == value {
		return nil
	}
	c.drawOrder = value
	return c.drawOrderChanged.Raise(c, EventArgsEmpty())
}

var (
	_ IGameComponent = (*engineComponent)(nil)
	_ IUpdateable    = (*engineComponent)(nil)
	_ IDrawable      = (*engineComponent)(nil)
	_ IUpdateable    = (*updateOnlyComponent)(nil)
)

// plainComponent is neither IUpdateable nor IDrawable, which the reference's
// two independent `isinst` tests both allow.
type plainComponent struct{ initialized int }

func (c *plainComponent) Initialize() error { c.initialized++; return nil }

// updateOnlyComponent satisfies IUpdateable but not IDrawable, so exactly one
// of the reference's two independent `isinst` tests admits it.
type updateOnlyComponent struct {
	updateOrder        int32
	updateOrderChanged EventSource[*EventArgs]
	enabledChanged     EventSource[*EventArgs]
}

func (c *updateOnlyComponent) Initialize() error  { return nil }
func (c *updateOnlyComponent) Enabled() bool      { return true }
func (c *updateOnlyComponent) UpdateOrder() int32 { return c.updateOrder }
func (c *updateOnlyComponent) Update(GameTime)    {}
func (c *updateOnlyComponent) AddEnabledChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.enabledChanged.Add(h)
}
func (c *updateOnlyComponent) RemoveEnabledChangedHandler(s EventSubscription) error {
	return c.enabledChanged.Remove(s)
}
func (c *updateOnlyComponent) AddUpdateOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.updateOrderChanged.Add(h)
}
func (c *updateOnlyComponent) RemoveUpdateOrderChangedHandler(s EventSubscription) error {
	return c.updateOrderChanged.Remove(s)
}

type callbackRecorder struct{}

func (callbackRecorder) Initialize(*Game) error       { return nil }
func (callbackRecorder) LoadContent(*Game) error      { return nil }
func (callbackRecorder) Update(*Game, GameTime) error { return nil }
func (callbackRecorder) Draw(*Game, GameTime) error   { return nil }
func (callbackRecorder) UnloadContent(*Game) error    { return nil }

func newTestGame(t *testing.T) *Game {
	t.Helper()
	game, err := NewGame(callbackRecorder{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

func updateableNames(game *Game) []string {
	names := make([]string, 0, len(game.updateableComponents))
	for _, entry := range game.updateableComponents {
		names = append(names, entry.component.(*engineComponent).name)
	}
	return names
}

func drawableNames(game *Game) []string {
	names := make([]string, 0, len(game.drawableComponents))
	for _, entry := range game.drawableComponents {
		names = append(names, entry.component.(*engineComponent).name)
	}
	return names
}

// TestGameComponentsAndServicesAreStableManagedIdentities is the whole point of
// Foundation 30: both getters are one `ldfld` over a field the constructor
// assigned once, so they are infallible, allocate nothing, and hand back the
// same object every time.
func TestGameComponentsAndServicesAreStableManagedIdentities(t *testing.T) {
	game := newTestGame(t)

	components, again := game.Components(), game.Components()
	if components == nil {
		t.Fatal("Components() returned nil; the constructor allocates the collection")
	}
	if components != again {
		t.Fatal("Components() allocated a second collection; the reference getter is one field read")
	}
	services, servicesAgain := game.Services(), game.Services()
	if services == nil {
		t.Fatal("Services() returned nil; the constructor allocates the container")
	}
	if services != servicesAgain {
		t.Fatal("Services() allocated a second container; the reference getter is one field read")
	}
	if components.Count() != 0 {
		t.Fatalf("a fresh collection has Count 0, got %d", components.Count())
	}
}

// TestGameComponentsKeepsReferenceSemantics proves the collection is shared
// state rather than a copy: a mutation through one alias is visible through
// every other, exactly as a CLR reference type behaves.
func TestGameComponentsKeepsReferenceSemantics(t *testing.T) {
	game := newTestGame(t)
	alias := game.Components()
	component := newEngineComponent("a", nil)
	if err := alias.Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if game.Components().Count() != 1 {
		t.Fatal("a mutation through one alias was not visible through the getter")
	}
	if !game.Components().Contains(component) {
		t.Fatal("the added component is not in the collection the getter returns")
	}
}

// TestGameServicesKeepsReferenceSemantics is the same claim for the container.
func TestGameServicesKeepsReferenceSemantics(t *testing.T) {
	game := newTestGame(t)
	key := reflect.TypeOf((*IUpdateable)(nil)).Elem()
	component := newEngineComponent("a", nil)
	if err := game.Services().AddService(key, component); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	found, err := game.Services().GetService(key)
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if found != any(component) {
		t.Fatal("a registration through the getter was not visible through the getter")
	}
}

// TestComponentAddedBeforeRunIsQueuedNotInitialized covers the `inRun` guard:
// before the run sequence raises the flag, an added component is queued rather
// than initialized.
func TestComponentAddedBeforeRunIsQueuedNotInitialized(t *testing.T) {
	game := newTestGame(t)
	component := newEngineComponent("a", nil)
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if component.initializeCount != 0 {
		t.Fatalf("component initialized %d times before the game ran", component.initializeCount)
	}
	if len(game.notYetInitialized) != 1 || game.notYetInitialized[0] != IGameComponent(component) {
		t.Fatalf("component was not queued: %v", game.notYetInitialized)
	}
}

// TestComponentAddedWhileRunningInitializesImmediately is the other half of the
// same guard.
func TestComponentAddedWhileRunningInitializesImmediately(t *testing.T) {
	game := newTestGame(t)
	game.inRun = true
	component := newEngineComponent("a", nil)
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if component.initializeCount != 1 {
		t.Fatalf("expected exactly one Initialize while running, got %d", component.initializeCount)
	}
	if len(game.notYetInitialized) != 0 {
		t.Fatal("a component added while running must not be queued")
	}
}

// TestFailingComponentInitializeStopsTheHandlerChain proves the ordering inside
// GameComponentAdded: initialization runs FIRST, so a failure leaves the
// component out of both derived lists, and because Raise stops at the first
// failing handler a consumer's own handler never runs either.
func TestFailingComponentInitializeStopsTheHandlerChain(t *testing.T) {
	game := newTestGame(t)
	game.inRun = true
	failure := errors.New("component refused to initialize")
	component := newEngineComponent("a", nil)
	component.initializeError = failure

	consumerRan := false
	if _, err := game.Components().AddComponentAddedHandler(func(any, *GameComponentCollectionEventArgs) error {
		consumerRan = true
		return nil
	}); err != nil {
		t.Fatalf("AddComponentAddedHandler: %v", err)
	}

	err := game.Components().Add(component)
	if !errors.Is(err, failure) {
		t.Fatalf("expected the component's own failure, got %v", err)
	}
	if consumerRan {
		t.Fatal("a consumer handler ran after the engine handler failed")
	}
	if len(game.updateableComponents) != 0 || len(game.drawableComponents) != 0 {
		t.Fatal("a component whose Initialize failed was still tracked")
	}
	// The collection itself has already mutated: InsertItem mutates before it
	// announces, which Foundation 26 qualified and this must not change.
	if game.Components().Count() != 1 {
		t.Fatalf("expected the failed announcement to leave the insertion applied, Count=%d", game.Components().Count())
	}
}

// TestUpdateOrderPlacementIsSortedWithStableTies is the ordering derivation:
// BinarySearch converges left of a run of equal orders and the explicit forward
// walk steps past it, so ties keep insertion order.
func TestUpdateOrderPlacementIsSortedWithStableTies(t *testing.T) {
	game := newTestGame(t)
	names := []string{"a", "b", "c", "d", "e"}
	orders := []int32{10, 5, 10, 1, 5}
	for i, name := range names {
		component := newEngineComponent(name, nil)
		component.updateOrder = orders[i]
		component.drawOrder = orders[i]
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add %s: %v", name, err)
		}
	}
	want := []string{"d", "b", "e", "a", "c"}
	if got := updateableNames(game); !equalNames(got, want) {
		t.Fatalf("update order %v, want %v", got, want)
	}
	if got := drawableNames(game); !equalNames(got, want) {
		t.Fatalf("draw order %v, want %v", got, want)
	}
}

// TestOrderChangeRePlacesAtTheEndOfItsTieRun follows from the remove-then-
// reinsert shape: the component is gone from the list when the search runs, so
// it is placed by the same forward walk as a fresh add.
func TestOrderChangeRePlacesAtTheEndOfItsTieRun(t *testing.T) {
	game := newTestGame(t)
	made := map[string]*engineComponent{}
	for i, name := range []string{"a", "b", "c"} {
		component := newEngineComponent(name, nil)
		component.updateOrder = int32(i)
		made[name] = component
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add %s: %v", name, err)
		}
	}
	if got := updateableNames(game); !equalNames(got, []string{"a", "b", "c"}) {
		t.Fatalf("setup order %v", got)
	}
	// a moves from 0 to 2, which is c's order, so it lands after c.
	if err := made["a"].SetUpdateOrder(2); err != nil {
		t.Fatalf("SetUpdateOrder: %v", err)
	}
	if got := updateableNames(game); !equalNames(got, []string{"b", "c", "a"}) {
		t.Fatalf("after reorder %v, want [b c a]", got)
	}
	// Setting the same value again suppresses the event, so nothing moves.
	if err := made["a"].SetUpdateOrder(2); err != nil {
		t.Fatalf("SetUpdateOrder: %v", err)
	}
	if got := updateableNames(game); !equalNames(got, []string{"b", "c", "a"}) {
		t.Fatalf("a suppressed event moved the component: %v", got)
	}
}

// TestDrawOrderChangeRePlaces is the same claim on the drawable list, which is
// maintained by a separate handler over a separate comparer.
func TestDrawOrderChangeRePlaces(t *testing.T) {
	game := newTestGame(t)
	made := map[string]*engineComponent{}
	for i, name := range []string{"a", "b", "c"} {
		component := newEngineComponent(name, nil)
		component.drawOrder = int32(i)
		made[name] = component
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add %s: %v", name, err)
		}
	}
	if err := made["c"].SetDrawOrder(-1); err != nil {
		t.Fatalf("SetDrawOrder: %v", err)
	}
	if got := drawableNames(game); !equalNames(got, []string{"c", "a", "b"}) {
		t.Fatalf("after reorder %v, want [c a b]", got)
	}
}

// TestRemovingAComponentUntracksAndUnsubscribes proves the removal handler both
// drops the component from the derived lists and removes Game's order-changed
// registration, so a later order change moves nothing.
func TestRemovingAComponentUntracksAndUnsubscribes(t *testing.T) {
	game := newTestGame(t)
	kept := newEngineComponent("kept", nil)
	kept.updateOrder = 5
	removed := newEngineComponent("removed", nil)
	removed.updateOrder = 1
	for _, component := range []*engineComponent{kept, removed} {
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if got := updateableNames(game); !equalNames(got, []string{"removed", "kept"}) {
		t.Fatalf("setup %v", got)
	}
	ok, err := game.Components().Remove(removed)
	if err != nil || !ok {
		t.Fatalf("Remove: %v %v", ok, err)
	}
	if got := updateableNames(game); !equalNames(got, []string{"kept"}) {
		t.Fatalf("after remove %v", got)
	}
	if len(game.notYetInitialized) != 1 {
		t.Fatalf("the removed component should also leave the pending queue, got %d entries", len(game.notYetInitialized))
	}
	// Game's registration is gone, so raising the event must not re-insert.
	if err := removed.SetUpdateOrder(99); err != nil {
		t.Fatalf("SetUpdateOrder: %v", err)
	}
	if got := updateableNames(game); !equalNames(got, []string{"kept"}) {
		t.Fatalf("an unsubscribed component was re-tracked: %v", got)
	}
}

// TestClearAnnouncesEveryComponentAndUntracksAll exercises the interaction
// Foundation 26 pinned: ClearItems announces the whole collection before it
// mutates, and each announcement reaches the engine's removal handler.
func TestClearAnnouncesEveryComponentAndUntracksAll(t *testing.T) {
	game := newTestGame(t)
	for _, name := range []string{"a", "b", "c"} {
		if err := game.Components().Add(newEngineComponent(name, nil)); err != nil {
			t.Fatalf("Add %s: %v", name, err)
		}
	}
	if err := game.Components().Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(game.updateableComponents) != 0 || len(game.drawableComponents) != 0 {
		t.Fatal("Clear left components tracked")
	}
	if len(game.notYetInitialized) != 0 {
		t.Fatal("Clear left components queued for initialization")
	}
	if game.Components().Count() != 0 {
		t.Fatal("Clear did not empty the collection")
	}
}

// TestNilComponentReachesTheEngineOnlyThroughClear pins the asymmetry that
// Foundation 26 measured: RemoveItem null-checks before announcing and
// ClearItems does not, so a nil element is announced exactly once, by Clear,
// and the engine must survive it.
func TestNilComponentReachesTheEngineOnlyThroughClear(t *testing.T) {
	game := newTestGame(t)
	if err := game.Components().Add(nil); err != nil {
		t.Fatalf("Add(nil): %v", err)
	}
	if len(game.notYetInitialized) != 0 {
		t.Fatal("InsertItem announces nothing for a nil component, so nothing may be queued")
	}
	if err := game.Components().Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if game.Components().Count() != 0 {
		t.Fatal("Clear did not empty the collection")
	}
}

// TestComponentThatIsNeitherUpdateableNorDrawableIsStillQueued follows from the
// two independent `isinst` tests: neither placement is a precondition of the
// initialization step.
func TestComponentThatIsNeitherUpdateableNorDrawableIsStillQueued(t *testing.T) {
	game := newTestGame(t)
	component := &plainComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(game.notYetInitialized) != 1 {
		t.Fatal("a plain IGameComponent must still be queued for initialization")
	}
	if len(game.updateableComponents) != 0 || len(game.drawableComponents) != 0 {
		t.Fatal("a plain IGameComponent must not be tracked in either derived list")
	}
}

// TestUpdateableOnlyComponentIsTrackedInOneListOnly proves the two placements
// are independent: the reference tests IUpdateable and IDrawable separately, so
// a component that is only one of them lands in only one derived list.
func TestUpdateableOnlyComponentIsTrackedInOneListOnly(t *testing.T) {
	game := newTestGame(t)
	component := &updateOnlyComponent{}
	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(game.updateableComponents) != 1 {
		t.Fatalf("expected one tracked updateable, got %d", len(game.updateableComponents))
	}
	if len(game.drawableComponents) != 0 {
		t.Fatalf("a non-IDrawable component was tracked as drawable")
	}
	if _, err := game.Components().Remove(component); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(game.updateableComponents) != 0 {
		t.Fatal("removal left the component tracked")
	}
}

// TestComparerReturnsOneForEqualOrders is the single fact the whole ordering
// derivation rests on, asserted directly against the two projected comparers.
func TestComparerReturnsOneForEqualOrders(t *testing.T) {
	left, right := newEngineComponent("l", nil), newEngineComponent("r", nil)
	left.updateOrder, right.updateOrder = 7, 7
	left.drawOrder, right.drawOrder = 7, 7
	if got := compareUpdateOrder(left, right); got != 1 {
		t.Fatalf("equal update orders must compare 1, got %d", got)
	}
	if got := compareDrawOrder(left, right); got != 1 {
		t.Fatalf("equal draw orders must compare 1, got %d", got)
	}
	if got := compareUpdateOrder(left, left); got != 0 {
		t.Fatalf("reference identity must compare 0, got %d", got)
	}
	if got := compareUpdateOrder(nil, nil); got != 0 {
		t.Fatalf("two nulls must compare 0, got %d", got)
	}
	if got := compareUpdateOrder(nil, right); got != 1 {
		t.Fatalf("a null x must compare 1, got %d", got)
	}
	if got := compareUpdateOrder(left, nil); got != -1 {
		t.Fatalf("a null y must compare -1, got %d", got)
	}
}

// TestBinarySearchMatchesTheReferenceMissEncoding checks the one arithmetic
// detail the placement depends on: a miss returns the bitwise complement of the
// insertion point.
func TestBinarySearchMatchesTheReferenceMissEncoding(t *testing.T) {
	values := []int32{1, 3, 5, 7}
	compare := func(a, b int32) int32 {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}
	for _, item := range []struct {
		value int32
		want  int32
	}{{1, 0}, {3, 1}, {5, 2}, {7, 3}, {0, ^int32(0)}, {2, ^int32(1)}, {4, ^int32(2)}, {8, ^int32(4)}} {
		if got := binarySearchEntries(values, item.value, compare); got != item.want {
			t.Fatalf("binarySearch(%d) = %d, want %d", item.value, got, item.want)
		}
	}
}

// TestEngineHandlersRunBeforeConsumerHandlers pins the subscription order the
// constructor establishes, which is what lets a consumer observe a consistent
// Game from inside their own ComponentAdded handler.
func TestEngineHandlersRunBeforeConsumerHandlers(t *testing.T) {
	game := newTestGame(t)
	trackedWhenConsumerRan := -1
	if _, err := game.Components().AddComponentAddedHandler(func(any, *GameComponentCollectionEventArgs) error {
		trackedWhenConsumerRan = len(game.updateableComponents)
		return nil
	}); err != nil {
		t.Fatalf("AddComponentAddedHandler: %v", err)
	}
	if err := game.Components().Add(newEngineComponent("a", nil)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if trackedWhenConsumerRan != 1 {
		t.Fatalf("the consumer handler saw %d tracked components; the engine handler must have run first", trackedWhenConsumerRan)
	}
}
