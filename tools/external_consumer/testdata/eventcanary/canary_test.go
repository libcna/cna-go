package eventcanary

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
)

func TestExternalTypeSatisfiesBothComponentContracts(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator
	var drawable framework.IDrawable = rotator

	updateable.Update(framework.NewGameTimeByNone())
	drawable.Draw(framework.NewGameTimeByNone())
	if rotator.Updates != 1 || rotator.Draws != 1 {
		t.Fatalf("Updates=%d Draws=%d", rotator.Updates, rotator.Draws)
	}
	if updateable.Enabled() || drawable.Visible() ||
		updateable.UpdateOrder() != 0 || drawable.DrawOrder() != 0 {
		t.Fatal("a zero-valued external conformer reported non-zero contract state")
	}
}

func TestExternalTypeRaisesItsOwnEvents(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator

	var order []string
	var senders []any
	var args []*framework.EventArgs
	record := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, a *framework.EventArgs) error {
			order = append(order, name)
			senders = append(senders, sender)
			args = append(args, a)
			return nil
		}
	}

	first, err := updateable.AddEnabledChangedHandler(record("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := updateable.AddEnabledChangedHandler(record("second"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two external registrations produced one token")
	}

	if err := rotator.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("handler order = %v", order)
	}
	for i := range senders {
		if senders[i] != any(rotator) {
			t.Fatalf("sender[%d] is not the declaring external instance", i)
		}
		if args[i] != framework.EventArgsEmpty() {
			t.Fatalf("args[%d] is not the shared EventArgs.Empty identity", i)
		}
	}

	// Setting the same value again raises nothing.
	order = nil
	if err := rotator.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatalf("an unchanged value raised %v", order)
	}
}

// TestExternalDuplicateRegistrationsAreRemovedSeparately is the token-identity
// projection seen from outside the binding.
func TestExternalDuplicateRegistrationsAreRemovedSeparately(t *testing.T) {
	rotator := NewRotator("external")
	runs := 0
	same := func(sender any, a *framework.EventArgs) error {
		runs++
		return nil
	}

	first, err := rotator.AddVisibleChangedHandler(same)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rotator.AddVisibleChangedHandler(same)
	if err != nil {
		t.Fatal(err)
	}

	if err := rotator.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if runs != 2 {
		t.Fatalf("one handler added twice ran %d times, want 2", runs)
	}

	runs = 0
	if err := rotator.RemoveVisibleChangedHandler(first); err != nil {
		t.Fatal(err)
	}
	if err := rotator.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Fatalf("after removing one duplicate the handler ran %d times, want 1", runs)
	}

	runs = 0
	if err := rotator.RemoveVisibleChangedHandler(second); err != nil {
		t.Fatal(err)
	}
	if err := rotator.RemoveVisibleChangedHandler(second); err != nil {
		t.Fatalf("removing an already-removed token = %v", err)
	}
	if err := rotator.RemoveVisibleChangedHandler(framework.EventSubscription{}); err != nil {
		t.Fatalf("removing the zero token = %v", err)
	}
	if err := rotator.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if runs != 0 {
		t.Fatalf("after removing both registrations the handler ran %d times", runs)
	}
}

// TestExternalTokensDoNotCrossInstances proves a token from one external
// instance cannot disturb another.
func TestExternalTokensDoNotCrossInstances(t *testing.T) {
	left, right := NewRotator("left"), NewRotator("right")
	leftRuns, rightRuns := 0, 0

	leftToken, err := left.AddDrawOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		leftRuns++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := right.AddDrawOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		rightRuns++
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Handing the left instance's token to the right instance must disturb
	// neither event.
	if err := right.RemoveDrawOrderChangedHandler(leftToken); err != nil {
		t.Fatalf("cross-instance removal = %v", err)
	}
	if err := left.SetDrawOrder(3); err != nil {
		t.Fatal(err)
	}
	if err := right.SetDrawOrder(4); err != nil {
		t.Fatal(err)
	}
	if leftRuns != 1 || rightRuns != 1 {
		t.Fatalf("leftRuns=%d rightRuns=%d, want both events intact", leftRuns, rightRuns)
	}
}

// TestExternalHandlerFailurePropagates proves the external raiser sees a
// handler's failure and that no later handler ran.
func TestExternalHandlerFailurePropagates(t *testing.T) {
	rotator := NewRotator("external")
	failure := errors.New("external observer refused")
	later := false

	if _, err := rotator.AddUpdateOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		return failure
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := rotator.AddUpdateOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		later = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := rotator.SetUpdateOrder(7); err != nil {
		if !errors.Is(err, failure) {
			t.Fatalf("SetUpdateOrder = %v, want the handler failure", err)
		}
	} else {
		t.Fatal("SetUpdateOrder swallowed the handler failure")
	}
	if later {
		t.Fatal("a later handler ran after an earlier one failed")
	}
	if rotator.UpdateOrder() != 7 {
		t.Fatal("the failed raise rolled back the state change")
	}
}

// TestExternalConsumerCannotRaiseThroughTheContract is the capability boundary:
// a consumer holding only the contract can add and remove handlers, and there
// is no projected way to raise. The EventSource fields are unexported, so the
// only raiser is the declaring type itself.
func TestExternalConsumerCannotRaiseThroughTheContract(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator

	if _, isRaiser := any(updateable).(interface {
		Raise(any, *framework.EventArgs) error
	}); isRaiser {
		t.Fatal("the contract exposes a raise operation to plain consumers")
	}
	if _, isSource := any(updateable).(interface {
		Add(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	}); isSource {
		t.Fatal("the contract exposes the EventSource surface to plain consumers")
	}
}

// TestExternalUseOfTheEventArgsCarriers proves the three System.EventArgs
// carriers are usable from outside, including the two whose construction is
// deliberately not public.
func TestExternalUseOfTheEventArgsCarriers(t *testing.T) {
	// GameComponentCollectionEventArgs declares a public constructor.
	carrier := framework.NewGameComponentCollectionEventArgs(nil)
	if carrier == nil || carrier.GameComponent() != nil {
		t.Fatalf("carrier = %v", carrier)
	}
	// A carrier can be raised through an EventSource of its own type.
	var source framework.EventSource[*framework.GameComponentCollectionEventArgs]
	seen := 0
	if _, err := source.Add(func(sender any, args *framework.GameComponentCollectionEventArgs) error {
		if args != carrier {
			t.Fatal("the raised carrier is not the exact instance")
		}
		seen++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := source.Raise(nil, carrier); err != nil {
		t.Fatal(err)
	}
	if seen != 1 {
		t.Fatalf("handler ran %d times", seen)
	}
}

// The collection conformance suite. Everything below holds only the published
// contract: no CNA-Go internal, no unexported adapter, no backing store.

func drainCollection(t *testing.T, iterator framework.Iterator[framework.IGameComponent]) []framework.IGameComponent {
	t.Helper()
	var seen []framework.IGameComponent
	for {
		value, ok, err := iterator.Next()
		if err != nil {
			t.Fatalf("unexpected enumeration failure: %v", err)
		}
		if !ok {
			return seen
		}
		seen = append(seen, value)
	}
}

func TestExternalCallerCanUseTheInheritedCollectionSurface(t *testing.T) {
	// Every one of these is inherited from Collection<IGameComponent>. Not one
	// is declared by GameComponentCollection in the XNA metadata, so if the
	// composition projection had not been taken, this whole test would be
	// unwritable and the collection would be one nothing can be added to.
	collection := framework.NewGameComponentCollection()
	if collection.Count() != 0 {
		t.Fatalf("Count = %d, want 0", collection.Count())
	}

	first, second := NewRotator("first"), NewRotator("second")
	if err := collection.Add(first); err != nil {
		t.Fatal(err)
	}
	if err := collection.Add(second); err != nil {
		t.Fatal(err)
	}
	if collection.Count() != 2 {
		t.Fatalf("Count = %d, want 2", collection.Count())
	}
	indexed, err := collection.Item(1)
	if err != nil {
		t.Fatal(err)
	}
	if indexed != framework.IGameComponent(second) {
		t.Fatal("Item(1) is not the second component")
	}
	if collection.IndexOf(first) != 0 || !collection.Contains(second) {
		t.Fatal("IndexOf/Contains disagree with insertion order")
	}
	if got := drainCollection(t, collection.GetEnumerator()); len(got) != 2 ||
		got[0] != framework.IGameComponent(first) || got[1] != framework.IGameComponent(second) {
		t.Fatalf("enumeration = %v", got)
	}
	third := NewRotator("third")
	if err := collection.Insert(1, third); err != nil {
		t.Fatal(err)
	}
	middle, err := collection.Item(1)
	if err != nil {
		t.Fatal(err)
	}
	if middle != framework.IGameComponent(third) {
		t.Fatal("Insert did not place at the requested index")
	}
	destination := make([]framework.IGameComponent, 3)
	if err := collection.CopyTo(destination, 0); err != nil {
		t.Fatal(err)
	}
	if destination[0] != framework.IGameComponent(first) {
		t.Fatal("CopyTo did not copy in order")
	}
	removed, err := collection.Remove(third)
	if err != nil || !removed {
		t.Fatalf("Remove = %v, %v", removed, err)
	}
	if err := collection.RemoveAt(0); err != nil {
		t.Fatal(err)
	}
	if err := collection.Clear(); err != nil {
		t.Fatal(err)
	}
	if collection.Count() != 0 {
		t.Fatalf("Count = %d after Clear, want 0", collection.Count())
	}
}

func TestExternalCallerObservesTheExactMutationAndAnnouncementOrder(t *testing.T) {
	collection := framework.NewGameComponentCollection()
	probe := &CollectionProbe{}
	addedToken, err := collection.AddComponentAddedHandler(probe.Handler("added"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collection.AddComponentRemovedHandler(probe.Handler("removed")); err != nil {
		t.Fatal(err)
	}

	first, second := NewRotator("first"), NewRotator("second")
	if err := collection.Add(first); err != nil {
		t.Fatal(err)
	}
	if err := collection.Add(second); err != nil {
		t.Fatal(err)
	}
	// Add mutates and THEN announces, so each handler saw the new count.
	if len(probe.Counts) != 2 || probe.Counts[0] != 1 || probe.Counts[1] != 2 {
		t.Fatalf("counts observed during Add = %v, want [1 2]", probe.Counts)
	}
	// The sender is the collection, and each raise carries a fresh args.
	for i, sender := range probe.Senders {
		if sender != any(collection) {
			t.Fatalf("sender[%d] is not the collection", i)
		}
	}
	if probe.Args[0] == probe.Args[1] {
		t.Fatal("two raises shared one event args instance")
	}
	if probe.Components[0] != framework.IGameComponent(first) ||
		probe.Components[1] != framework.IGameComponent(second) {
		t.Fatal("the announced components are not the added ones")
	}

	// Remove also mutates first.
	probe.Reset()
	if _, err := collection.Remove(first); err != nil {
		t.Fatal(err)
	}
	if len(probe.Counts) != 1 || probe.Counts[0] != 1 {
		t.Fatalf("count observed during Remove = %v, want [1]", probe.Counts)
	}

	// Clear announces the WHOLE collection before it mutates, which is the
	// opposite order and is visible from outside.
	if err := collection.Add(NewRotator("third")); err != nil {
		t.Fatal(err)
	}
	probe.Reset()
	if err := collection.Clear(); err != nil {
		t.Fatal(err)
	}
	if len(probe.Counts) != 2 || probe.Counts[0] != 2 || probe.Counts[1] != 2 {
		t.Fatalf("counts observed during Clear = %v, want [2 2]", probe.Counts)
	}
	if collection.Count() != 0 {
		t.Fatalf("Count = %d after Clear", collection.Count())
	}

	// Removing a registration stops only that one.
	if err := collection.RemoveComponentAddedHandler(addedToken); err != nil {
		t.Fatal(err)
	}
	probe.Reset()
	if err := collection.Add(NewRotator("fourth")); err != nil {
		t.Fatal(err)
	}
	if len(probe.Events) != 0 {
		t.Fatalf("a removed handler still ran: %v", probe.Events)
	}
}

func TestExternalCallerObservesTheDuplicateAndNilCases(t *testing.T) {
	collection := framework.NewGameComponentCollection()
	probe := &CollectionProbe{}
	if _, err := collection.AddComponentAddedHandler(probe.Handler("added")); err != nil {
		t.Fatal(err)
	}
	if _, err := collection.AddComponentRemovedHandler(probe.Handler("removed")); err != nil {
		t.Fatal(err)
	}

	only := NewRotator("only")
	if err := collection.Add(only); err != nil {
		t.Fatal(err)
	}
	probe.Reset()
	if err := collection.Add(only); err == nil {
		t.Fatal("adding the same component twice must fail")
	}
	if collection.Count() != 1 || len(probe.Events) != 0 {
		t.Fatalf("a rejected duplicate changed something: Count=%d events=%v", collection.Count(), probe.Events)
	}

	// A nil component is stored and announces nothing on insert, but Clear
	// announces it, because ClearItems has no null check.
	nilBearing := framework.NewGameComponentCollection()
	nilProbe := &CollectionProbe{}
	if _, err := nilBearing.AddComponentAddedHandler(nilProbe.Handler("added")); err != nil {
		t.Fatal(err)
	}
	if _, err := nilBearing.AddComponentRemovedHandler(nilProbe.Handler("removed")); err != nil {
		t.Fatal(err)
	}
	if err := nilBearing.Add(nil); err != nil {
		t.Fatal(err)
	}
	if nilBearing.Count() != 1 || len(nilProbe.Events) != 0 {
		t.Fatalf("a nil component announced something: Count=%d events=%v", nilBearing.Count(), nilProbe.Events)
	}
	if err := nilBearing.Add(nil); err == nil {
		t.Fatal("a second nil must be rejected as a duplicate")
	}
	nilProbe.Reset()
	if err := nilBearing.Clear(); err != nil {
		t.Fatal(err)
	}
	if len(nilProbe.Events) != 1 || nilProbe.Components[0] != nil {
		t.Fatalf("Clear must announce the nil element: events=%v components=%v", nilProbe.Events, nilProbe.Components)
	}
}

func TestExternalCallerObservesTheIndexerAndItsGuards(t *testing.T) {
	collection := framework.NewGameComponentCollection()
	if err := collection.Add(NewRotator("a")); err != nil {
		t.Fatal(err)
	}
	// Assignment through the inherited indexer setter never succeeds.
	if err := collection.SetItemProperty(0, NewRotator("b")); err == nil {
		t.Fatal("indexed assignment must fail")
	}
	// Out of range reports the range failure instead, because set_Item
	// validates before it reaches SetItem.
	if err := collection.SetItemProperty(7, NewRotator("b")); err == nil {
		t.Fatal("out-of-range assignment must fail")
	}
	// The declared protected override is projected too and always fails.
	if err := collection.SetItemMethod(0, NewRotator("b")); err == nil {
		t.Fatal("SetItemMethod must fail")
	}
	survivor, err := collection.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if survivor.(*Rotator).Name() != "a" {
		t.Fatal("a failed assignment mutated the collection")
	}
	// Insert admits Count itself; RemoveAt and the setter do not.
	if err := collection.Insert(1, NewRotator("appended")); err != nil {
		t.Fatalf("Insert at Count must be legal: %v", err)
	}
	if err := collection.RemoveAt(2); err == nil {
		t.Fatal("RemoveAt at Count must fail")
	}
	if _, err := collection.Item(-1); err == nil {
		t.Fatal("a negative index must fail")
	}
}

func TestExternalCallerObservesEnumerationInvalidation(t *testing.T) {
	collection := framework.NewGameComponentCollection()
	if err := collection.Add(NewRotator("a")); err != nil {
		t.Fatal(err)
	}
	iterator := collection.GetEnumerator()
	if _, ok, err := iterator.Next(); !ok || err != nil {
		t.Fatalf("first step = ok %v err %v", ok, err)
	}
	if err := collection.Add(NewRotator("b")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := iterator.Next(); err == nil {
		t.Fatal("a mutation must invalidate a live enumerator")
	}
	// Clearing an already empty collection still invalidates, because
	// List<T>.Clear increments its version unconditionally.
	empty := framework.NewGameComponentCollection()
	emptyIterator := empty.GetEnumerator()
	if err := empty.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := emptyIterator.Next(); err == nil {
		t.Fatal("Clear of an empty collection must still invalidate")
	}
}

func TestExternalCallerObservesWhatAFailedAnnouncementLeavesBehind(t *testing.T) {
	boom := errors.New("external handler failure")

	// Add mutates before it announces, so the component stays.
	added := framework.NewGameComponentCollection()
	addProbe := &CollectionProbe{}
	if _, err := added.AddComponentAddedHandler(addProbe.Failing("added", boom)); err != nil {
		t.Fatal(err)
	}
	if err := added.Add(NewRotator("a")); !errors.Is(err, boom) {
		t.Fatalf("the handler failure must reach the caller, got %v", err)
	}
	if added.Count() != 1 {
		t.Fatalf("Count = %d, want 1: Add mutates before it announces", added.Count())
	}

	// Clear announces before it mutates, so nothing is removed.
	cleared := framework.NewGameComponentCollection()
	if err := cleared.Add(NewRotator("a")); err != nil {
		t.Fatal(err)
	}
	if err := cleared.Add(NewRotator("b")); err != nil {
		t.Fatal(err)
	}
	clearProbe := &CollectionProbe{}
	if _, err := cleared.AddComponentRemovedHandler(clearProbe.Failing("removed", boom)); err != nil {
		t.Fatal(err)
	}
	if err := cleared.Clear(); !errors.Is(err, boom) {
		t.Fatalf("the handler failure must reach the caller, got %v", err)
	}
	if cleared.Count() != 2 {
		t.Fatalf("Count = %d, want 2: Clear announces before it mutates", cleared.Count())
	}
	if len(clearProbe.Events) != 1 {
		t.Fatalf("dispatch must stop at the first failure, events = %v", clearProbe.Events)
	}
}

func TestExternalCallerCannotReachTheCollectionImplementation(t *testing.T) {
	collection := framework.NewGameComponentCollection()
	if err := collection.Add(NewRotator("a")); err != nil {
		t.Fatal(err)
	}
	// CopyTo is the only way out, and it hands over a copy.
	destination := make([]framework.IGameComponent, 1)
	if err := collection.CopyTo(destination, 0); err != nil {
		t.Fatal(err)
	}
	destination[0] = NewRotator("hijacked")
	survivor, err := collection.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if survivor.(*Rotator).Name() != "a" {
		t.Fatal("CopyTo aliased the backing store")
	}
	// The collection is a CLR class, so it keeps reference semantics.
	alias := collection
	if err := alias.Add(NewRotator("b")); err != nil {
		t.Fatal(err)
	}
	if collection.Count() != 2 {
		t.Fatal("two variables naming one collection must observe one state")
	}
}

func TestExternalTypeSatisfiesTheDevicePublicationContract(t *testing.T) {
	service := &DeviceService{}
	var contract graphics.IGraphicsDeviceService = service

	// The accessor is infallible and reports nil before a device exists,
	// exactly as the reference implementor's one-ldfld getter returns null.
	if contract.GraphicsDevice() != nil {
		t.Fatal("a service with no device must report nil")
	}

	var seen []string
	token, err := contract.AddDeviceCreatedHandler(func(sender any, args *framework.EventArgs) error {
		if sender != any(service) || args != framework.EventArgsEmpty() {
			t.Fatal("sender or args identity was lost")
		}
		seen = append(seen, "created")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RaiseDeviceCreated(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "created" {
		t.Fatalf("seen = %v", seen)
	}
	if err := contract.RemoveDeviceCreatedHandler(token); err != nil {
		t.Fatal(err)
	}
	seen = nil
	if err := service.RaiseDeviceCreated(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 0 {
		t.Fatalf("a removed handler still ran: %v", seen)
	}
}

// silentService declares the nine contract members and NOTHING else: no raise
// helper, no exported event field, no way for anyone to make it fire.
type silentService struct {
	created   framework.EventSource[*framework.EventArgs]
	disposing framework.EventSource[*framework.EventArgs]
	reset     framework.EventSource[*framework.EventArgs]
	resetting framework.EventSource[*framework.EventArgs]
}

func (s *silentService) GraphicsDevice() *graphics.GraphicsDevice { return nil }
func (s *silentService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.created.Add(h)
}
func (s *silentService) RemoveDeviceCreatedHandler(sub framework.EventSubscription) error {
	return s.created.Remove(sub)
}
func (s *silentService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.disposing.Add(h)
}
func (s *silentService) RemoveDeviceDisposingHandler(sub framework.EventSubscription) error {
	return s.disposing.Remove(sub)
}
func (s *silentService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.reset.Add(h)
}
func (s *silentService) RemoveDeviceResetHandler(sub framework.EventSubscription) error {
	return s.reset.Remove(sub)
}
func (s *silentService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resetting.Add(h)
}
func (s *silentService) RemoveDeviceResettingHandler(sub framework.EventSubscription) error {
	return s.resetting.Remove(sub)
}

func TestTheDeviceContractIsExactlyTheNineAccessors(t *testing.T) {
	// A conformer that declares the nine members and nothing else satisfies
	// the contract, which is the claim: the contract requires exactly the one
	// accessor and the four two-accessor event pairs, and requires no way to
	// raise. Adding a tenth requirement would break this line at compile time.
	var contract graphics.IGraphicsDeviceService = &silentService{}

	// A consumer holding the contract can subscribe and unsubscribe, and can
	// do nothing else. There is no raise operation on the contract at all.
	seen := 0
	token, err := contract.AddDeviceResetHandler(func(any, *framework.EventArgs) error {
		seen++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != 0 {
		t.Fatal("subscribing must not fire anything")
	}
	if err := contract.RemoveDeviceResetHandler(token); err != nil {
		t.Fatal(err)
	}
	if contract.GraphicsDevice() != nil {
		t.Fatal("a service publishing no device must report nil")
	}
}

// ---------------------------------------------------------------------------
// Foundation 31 — the component loop and the explicit base call, proved from
// outside the binding.
//
// Everything below runs in a module whose only dependency is an extracted
// CNA-Go source artifact, with GOWORK=off and no sibling checkout. Nothing here
// can see an unexported field, a private list, or an implementation detail: the
// claims are made with the published surface alone.
// ---------------------------------------------------------------------------

func joinLog(entries []string) string { return strings.Join(entries, ",") }

// newCanaryGame builds a Game from a consumer's own GameCallbacks and returns
// both, which is the only construction path a downstream user has.
func newCanaryGame(t *testing.T) (*framework.Game, *UserGame) {
	t.Helper()
	user := NewUserGame()
	game, err := framework.NewGame(user)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if game == nil {
		t.Fatal("NewGame returned no Game")
	}
	return game, user
}

// TestConsumerConstructsAGameAndReachesItsManagedState is canary claims 1 and 2:
// a downstream user constructs a Game and obtains stable Components and
// Services identities.
func TestConsumerConstructsAGameAndReachesItsManagedState(t *testing.T) {
	game, _ := newCanaryGame(t)

	components := game.Components()
	services := game.Services()
	if components == nil || services == nil {
		t.Fatal("a constructed Game must expose both managed objects")
	}
	if components != game.Components() || services != game.Services() {
		t.Fatal("the getters allocated a second object; the reference getter is one field read")
	}
	if components.Count() != 0 {
		t.Fatalf("a fresh collection has Count 0, got %d", components.Count())
	}
	// Neither getter is fallible: they take no error result at all, which this
	// line asserts at compile time by using them in single-value position.
	var _ *framework.GameComponentCollection = game.Components()
	var _ *framework.GameServiceContainer = game.Services()
}

// TestConsumerAddsComponentsAndSubscribesToTheCollection is canary claims 3
// and 4.
func TestConsumerAddsComponentsAndSubscribesToTheCollection(t *testing.T) {
	game, _ := newCanaryGame(t)
	probe := &CollectionProbe{}
	if _, err := game.Components().AddComponentAddedHandler(probe.Handler("added")); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Components().AddComponentRemovedHandler(probe.Handler("removed")); err != nil {
		t.Fatal(err)
	}

	var log []string
	first := NewUserComponent("first", &log, 0, 0)
	second := NewUserComponent("second", &log, 0, 0)
	for _, component := range []*UserComponent{first, second} {
		if err := game.Components().Add(component); err != nil {
			t.Fatalf("Add %s: %v", component.Name, err)
		}
	}
	if game.Components().Count() != 2 {
		t.Fatalf("Count is %d", game.Components().Count())
	}
	if joinLog(probe.Events) != "added,added" {
		t.Fatalf("collection events %v", probe.Events)
	}
	// The consumer's handler ran AFTER the engine had already tracked the
	// component, which the counts show: Add mutates before it announces.
	if probe.Counts[0] != 1 || probe.Counts[1] != 2 {
		t.Fatalf("handler-observed counts %v", probe.Counts)
	}
	if probe.Components[1] != framework.IGameComponent(second) {
		t.Fatal("the event args did not carry the added component")
	}

	ok, err := game.Components().Remove(first)
	if err != nil || !ok {
		t.Fatalf("Remove: %v %v", ok, err)
	}
	if joinLog(probe.Events) != "added,added,removed" {
		t.Fatalf("collection events %v", probe.Events)
	}
}

// TestConsumerCallbackCallsTheBaseAndSeesTheComponentLoop is canary claims 5,
// 6, 7 and 8: the consumer's own overrides call the base, and the resulting
// component ordering is observable.
func TestConsumerCallbackCallsTheBaseAndSeesTheComponentLoop(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	// Deliberately added in an order that is neither the update order nor the
	// draw order, so what comes out is the engine's doing rather than the
	// consumer's add order.
	for _, spec := range []struct {
		name                   string
		updateOrder, drawOrder int32
	}{{"mid", 5, 5}, {"last", 9, 1}, {"first", 1, 9}} {
		if err := game.Components().Add(NewUserComponent(spec.name, &log, spec.updateOrder, spec.drawOrder)); err != nil {
			t.Fatalf("Add %s: %v", spec.name, err)
		}
	}

	// Claim 5: the callback calls GameBaseInitialize, and that is what
	// initializes the queued components.
	if len(log) != 0 {
		t.Fatalf("adding a component before the game ran must not initialize it: %v", log)
	}
	if err := user.Initialize(game); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// Initialization order is ADD order, not UpdateOrder. The pending queue is
	// a plain list the collection handler appends to and the drain consumes
	// from index 0, and it is a different list from the two ordered ones: only
	// Update and Draw are ordered.
	if joinLog(log) != "init:mid,init:last,init:first" {
		t.Fatalf("initialization order %q, want add order", joinLog(log))
	}
	if joinLog(user.Log) != "user:Initialize" {
		t.Fatalf("the consumer's own Initialize work did not run: %v", user.Log)
	}

	// Claims 6 and 8: the callback calls GameBaseUpdate and the update order
	// is observable and is UpdateOrder, not add order.
	log, user.Log = nil, nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if joinLog(log) != "update:first,update:mid,update:last" {
		t.Fatalf("update order %q", joinLog(log))
	}

	// Claim 7: the callback calls GameBaseDraw, whose order is DrawOrder --
	// the reverse here, which proves the two lists are independent.
	log = nil
	if err := user.Draw(game, framework.GameTime{}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if joinLog(log) != "draw:last,draw:mid,draw:first" {
		t.Fatalf("draw order %q", joinLog(log))
	}
}

// TestOmittingTheBaseCallPreventsBaseComponentIteration is canary claim 9, and
// it is the claim that matters most: base behavior is NOT automatic.
func TestOmittingTheBaseCallPreventsBaseComponentIteration(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	if err := game.Components().Add(NewUserComponent("only", &log, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}

	user.CallBase["Update"] = false
	user.CallBase["Draw"] = false
	log, user.Log = nil, nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if err := user.Draw(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if len(log) != 0 {
		t.Fatalf("a callback that omitted its base call still iterated components: %v", log)
	}
	if joinLog(user.Log) != "user:Update,user:Draw" {
		t.Fatalf("the consumer's own work did not run: %v", user.Log)
	}

	// Turning the base call back on restores it, with no state carried over.
	user.CallBase["Update"] = true
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:only" {
		t.Fatalf("restoring the base call produced %q", joinLog(log))
	}
}

// TestTheBaseCallPositionChangesOrderingRelativeToUserCode is canary claim 10.
func TestTheBaseCallPositionChangesOrderingRelativeToUserCode(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		baseFirst bool
		want      string
	}{
		{"user-then-base", false, "user:Update,update:one,update:two"},
		{"base-then-user", true, "update:one,update:two,user:Update"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			game, user := newCanaryGame(t)
			var interleaved []string
			for i, name := range []string{"one", "two"} {
				if err := game.Components().Add(NewUserComponent(name, &interleaved, int32(i), int32(i))); err != nil {
					t.Fatal(err)
				}
			}
			if err := user.Initialize(game); err != nil {
				t.Fatal(err)
			}

			// The consumer's own record goes into the SAME log the components
			// write to, so the order is one sequence rather than two.
			interleaved = nil
			user.Log = nil
			user.BaseFirst["Update"] = testCase.baseFirst
			// The consumer's own entry and the components' entries land in two
			// logs, so the single sequence is rebuilt from the choice under
			// test: with the base first, base entries precede the consumer's.
			if err := user.Update(game, framework.GameTime{}); err != nil {
				t.Fatal(err)
			}
			var got string
			if testCase.baseFirst {
				got = joinLog(append(append([]string{}, interleaved...), user.Log...))
			} else {
				got = joinLog(append(append([]string{}, user.Log...), interleaved...))
			}
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestBaseCallsNeverReEnterTheConsumerCallbacks is the recursion control from
// outside: a base call must not invoke the override contract, or a consumer
// whose override calls its base would recurse without bound.
//
// The proof is that this test terminates: every override below calls its base,
// so a base that called back would not return.
func TestBaseCallsNeverReEnterTheConsumerCallbacks(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	if err := game.Components().Add(NewUserComponent("only", &log, 0, 0)); err != nil {
		t.Fatal(err)
	}
	for _, call := range []func() error{
		func() error { return user.Initialize(game) },
		func() error { return user.LoadContent(game) },
		func() error { return user.Update(game, framework.GameTime{}) },
		func() error { return user.Draw(game, framework.GameTime{}) },
		func() error { return user.UnloadContent(game) },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	// Each override recorded its own work exactly once, so no base call
	// re-entered a callback.
	want := "user:Initialize,user:LoadContent,user:Update,user:Draw,user:UnloadContent"
	if joinLog(user.Log) != want {
		t.Fatalf("callback invocations %q, want %q", joinLog(user.Log), want)
	}
	// And the component ran its lifecycle exactly once each.
	if joinLog(log) != "init:only,update:only,draw:only" {
		t.Fatalf("component lifecycle %q", joinLog(log))
	}
}

// TestOneBaseCallIteratesEachComponentExactlyOnce is the duplicated-invocation
// control: nothing in CNA-Go silently calls base Update or base Draw a second
// time.
func TestOneBaseCallIteratesEachComponentExactlyOnce(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	if err := game.Components().Add(NewUserComponent("only", &log, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 4; frame++ {
		log = nil
		if err := user.Update(game, framework.GameTime{}); err != nil {
			t.Fatal(err)
		}
		if err := user.Draw(game, framework.GameTime{}); err != nil {
			t.Fatal(err)
		}
		if joinLog(log) != "update:only,draw:only" {
			t.Fatalf("frame %d produced %q", frame, joinLog(log))
		}
	}
}

// TestConsumerObservesEnabledVisibleAndOrderChanges proves the engine reacts to
// the component's own events, from outside: nothing here reaches Game to tell
// it that an order changed.
func TestConsumerObservesEnabledVisibleAndOrderChanges(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	components := map[string]*UserComponent{}
	for i, name := range []string{"a", "b", "c"} {
		component := NewUserComponent(name, &log, int32(i), int32(i))
		components[name] = component
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}

	// Disabling one skips it; the others keep their order.
	if err := components["b"].SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:a,update:c" {
		t.Fatalf("Enabled was ignored: %q", joinLog(log))
	}

	// Changing an update order re-places the component, which only the engine
	// can do: the consumer raised an event and nothing else.
	if err := components["b"].SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := components["a"].SetUpdateOrder(99); err != nil {
		t.Fatal(err)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:b,update:c,update:a" {
		t.Fatalf("UpdateOrderChanged did not re-place the component: %q", joinLog(log))
	}

	// Visible and DrawOrder are the separate draw-side pair.
	if err := components["c"].SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if err := components["a"].SetDrawOrder(-1); err != nil {
		t.Fatal(err)
	}
	log = nil
	if err := user.Draw(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "draw:a,draw:b" {
		t.Fatalf("draw side ignored Visible or DrawOrder: %q", joinLog(log))
	}
}

// TestRemovingAComponentStopsItsBaseIteration is the untracking claim from
// outside.
func TestRemovingAComponentStopsItsBaseIteration(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	kept := NewUserComponent("kept", &log, 0, 0)
	dropped := NewUserComponent("dropped", &log, 1, 1)
	for _, component := range []*UserComponent{kept, dropped} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Components().Remove(dropped); err != nil {
		t.Fatal(err)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:kept" {
		t.Fatalf("a removed component still ran: %q", joinLog(log))
	}
	// Its order-changed registration is gone too, so raising the event moves
	// nothing back into the loop.
	if err := dropped.SetUpdateOrder(-100); err != nil {
		t.Fatal(err)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:kept" {
		t.Fatalf("an unsubscribed component was re-tracked: %q", joinLog(log))
	}
}

// TestBaseCallsExposeNoNativeHandle is canary claim 12. Every base-call helper
// takes a *Game and returns only an error: nothing in the family carries a
// uintptr, an unsafe.Pointer, or any other native handle, and this file
// compiles against the whole published surface without importing one.
func TestBaseCallsExposeNoNativeHandle(t *testing.T) {
	game, _ := newCanaryGame(t)
	// The signatures, asserted at compile time.
	var (
		_ func(*framework.Game) error                     = framework.GameBaseInitialize
		_ func(*framework.Game) error                     = framework.GameBaseLoadContent
		_ func(*framework.Game) error                     = framework.GameBaseUnloadContent
		_ func(*framework.Game, framework.GameTime) error = framework.GameBaseUpdate
		_ func(*framework.Game, framework.GameTime) error = framework.GameBaseDraw
	)
	if err := framework.GameBaseUpdate(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
}

// TestAnUnconstructedGameIsRejectedByEveryBaseCall covers the family's one
// Go-only failure from outside: a zero Game is reachable in Go and is refused.
func TestAnUnconstructedGameIsRejectedByEveryBaseCall(t *testing.T) {
	zero := &framework.Game{}
	for name, call := range map[string]func(*framework.Game) error{
		"GameBaseInitialize":    framework.GameBaseInitialize,
		"GameBaseLoadContent":   framework.GameBaseLoadContent,
		"GameBaseUnloadContent": framework.GameBaseUnloadContent,
		"GameBaseUpdate":        func(g *framework.Game) error { return framework.GameBaseUpdate(g, framework.GameTime{}) },
		"GameBaseDraw":          func(g *framework.Game) error { return framework.GameBaseDraw(g, framework.GameTime{}) },
	} {
		if err := call(nil); err == nil {
			t.Fatalf("%s(nil) reported no error", name)
		}
		if err := call(zero); err == nil {
			t.Fatalf("%s(&Game{}) reported no error", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Foundation 32 — GameComponent, from outside the binding.
// ---------------------------------------------------------------------------

// TestConsumerUsesTheShippedGameComponent proves a downstream user can use
// XNA's own component class rather than writing a conformer, and that it drives
// the engine with no adapter in between.
func TestConsumerUsesTheShippedGameComponent(t *testing.T) {
	game, user := newCanaryGame(t)
	first := framework.NewGameComponent(game)
	second := framework.NewGameComponent(game)

	// Constructor defaults, from outside.
	if !first.Enabled() || first.UpdateOrder() != 0 || first.Game() != game {
		t.Fatal("constructor defaults are wrong")
	}
	// The constructor validates nothing, so a component with no Game is legal.
	if orphan := framework.NewGameComponent(nil); orphan.Game() != nil || !orphan.Enabled() {
		t.Fatal("a nil Game must be accepted and stored")
	}

	if err := second.SetUpdateOrder(-1); err != nil {
		t.Fatal(err)
	}
	for _, component := range []*framework.GameComponent{first, second} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	// A GameComponent is IUpdateable and IGameComponent but NOT IDrawable, so
	// base Draw iterates nothing.
	if _, drawable := any(first).(framework.IDrawable); drawable {
		t.Fatal("GameComponent must not satisfy IDrawable")
	}
	if err := user.Draw(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
}

// TestDisposingAComponentRemovesItFromGameComponents is canary claim 11 in its
// strongest form, which needed GameComponent to exist.
func TestDisposingAComponentRemovesItFromGameComponents(t *testing.T) {
	game, _ := newCanaryGame(t)
	component := framework.NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	if game.Components().Count() != 1 {
		t.Fatal("setup failed")
	}

	// The Disposed handler observes a Game the component has already left,
	// because Dispose removes before it announces.
	observed := int32(-1)
	raises := 0
	if _, err := component.AddDisposedHandler(func(sender any, args *framework.EventArgs) error {
		raises++
		observed = game.Components().Count()
		if sender != any(component) {
			t.Error("Disposed must be raised with the component as sender")
		}
		if args != framework.EventArgsEmpty() {
			t.Error("Disposed must carry the shared EventArgs.Empty identity")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := component.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if game.Components().Count() != 0 {
		t.Fatal("Dispose did not remove the component from Game.Components")
	}
	if observed != 0 {
		t.Fatalf("the Disposed handler observed Count=%d; removal precedes the announcement", observed)
	}
	if raises != 1 {
		t.Fatalf("Disposed raised %d times", raises)
	}

	// It is not idempotent, exactly as the reference is not.
	if err := component.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if raises != 2 {
		t.Fatalf("a second Dispose raised %d times in total", raises)
	}
}

// TestDisposedComponentStopsBeingUpdated joins the two halves: disposal removes
// the component from Components, which untracks it, which stops base Update
// from reaching it.
func TestDisposedComponentStopsBeingUpdated(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	// A consumer's own component, so the update is observable, plus a shipped
	// GameComponent whose Dispose does the removing.
	watcher := NewUserComponent("watcher", &log, 0, 0)
	if err := game.Components().Add(watcher); err != nil {
		t.Fatal(err)
	}
	component := framework.NewGameComponent(game)
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}

	removed := 0
	if _, err := game.Components().AddComponentRemovedHandler(func(any, *framework.GameComponentCollectionEventArgs) error {
		removed++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("Dispose announced %d removals", removed)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:watcher" {
		t.Fatalf("after disposal the loop ran %q", joinLog(log))
	}
	if game.Components().Count() != 1 {
		t.Fatalf("Count is %d", game.Components().Count())
	}
}

// TestGameComponentOrderChangeRePlacesItInTheLoop proves the shipped class's
// own setter drives the engine: nothing here tells Game the order changed.
func TestGameComponentOrderChangeRePlacesItInTheLoop(t *testing.T) {
	game, user := newCanaryGame(t)
	var log []string
	// Two consumer components bracket a shipped one, so its movement between
	// them is visible in one sequence.
	low := NewUserComponent("low", &log, 0, 0)
	high := NewUserComponent("high", &log, 100, 100)
	moving := framework.NewGameComponent(game)
	if err := moving.SetUpdateOrder(50); err != nil {
		t.Fatal(err)
	}
	movingRan := 0
	if _, err := moving.AddUpdateOrderChangedHandler(func(sender any, args *framework.EventArgs) error {
		movingRan++
		if sender != any(moving) {
			t.Error("UpdateOrderChanged must be raised with the component as sender")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, component := range []framework.IGameComponent{low, moving, high} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}

	// An unchanged assignment announces nothing at all.
	if err := moving.SetUpdateOrder(50); err != nil {
		t.Fatal(err)
	}
	if movingRan != 0 {
		t.Fatal("an unchanged assignment announced")
	}

	// Moving it past `high` re-places it, and the consumer's handler runs
	// after the engine's, because Game subscribed first.
	if err := moving.SetUpdateOrder(200); err != nil {
		t.Fatal(err)
	}
	if movingRan != 1 {
		t.Fatalf("the change announced %d times", movingRan)
	}
	log = nil
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if joinLog(log) != "update:low,update:high" {
		t.Fatalf("the ordered loop is %q; the shipped component moved to the end", joinLog(log))
	}

	// Disabling it is the other gate, and it does not move anything.
	if err := moving.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if moving.Enabled() {
		t.Fatal("SetEnabled did not store")
	}
}

// TestGameComponentSatisfiesItsDeclaredContractsFromOutside is the compiler-
// level witness, made from a module that is not part of the binding: if
// GameComponent stopped satisfying either contract, this file would not build.
func TestGameComponentSatisfiesItsDeclaredContractsFromOutside(t *testing.T) {
	var (
		_ framework.IGameComponent = (*framework.GameComponent)(nil)
		_ framework.IUpdateable    = (*framework.GameComponent)(nil)
	)
	// And it can be stored where either contract is required.
	game, _ := newCanaryGame(t)
	var asComponent framework.IGameComponent = framework.NewGameComponent(game)
	if err := game.Components().Add(asComponent); err != nil {
		t.Fatal(err)
	}
	var asUpdateable framework.IUpdateable = asComponent.(*framework.GameComponent)
	if !asUpdateable.Enabled() || asUpdateable.UpdateOrder() != 0 {
		t.Fatal("the contract view disagrees with the class")
	}
	if err := asComponent.Initialize(); err != nil {
		t.Fatalf("the contract's Initialize channel reported %v", err)
	}
}

// ---------------------------------------------------------------------------
// Foundation 34 — Game's four events, from outside the binding.
// ---------------------------------------------------------------------------

// TestConsumerSubscribesToAllFourGameEvents is the structural half of the
// native bridge qualification: a downstream user reaches every one of the eight
// projected accessors, gets a token from each Add, and hands each token back.
//
// It deliberately does not run the Game. Delivery of a real native signal is
// proved separately by the native stress harness, which runs the pinned
// runtime; what this proves is that the public surface is complete, usable and
// free of any native identity from a module that cannot see the binding's
// internals at all.
func TestConsumerSubscribesToAllFourGameEvents(t *testing.T) {
	game, _ := newCanaryGame(t)
	// The eight accessor signatures, asserted at compile time. Every one takes
	// or returns only EventHandler[*EventArgs] and EventSubscription.
	var (
		_ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = game.AddActivatedHandler
		_ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = game.AddDeactivatedHandler
		_ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = game.AddExitingHandler
		_ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = game.AddDisposedHandler
		_ func(framework.EventSubscription) error                                                 = game.RemoveActivatedHandler
		_ func(framework.EventSubscription) error                                                 = game.RemoveDeactivatedHandler
		_ func(framework.EventSubscription) error                                                 = game.RemoveExitingHandler
		_ func(framework.EventSubscription) error                                                 = game.RemoveDisposedHandler
	)
	// And the three protected raise sites, which are ordinary methods here.
	var (
		_ func(any, *framework.EventArgs) error = game.OnActivated
		_ func(any, *framework.EventArgs) error = game.OnDeactivated
		_ func(any, *framework.EventArgs) error = game.OnExiting
	)

	var seen []string
	adders := []struct {
		name   string
		add    func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
		remove func(framework.EventSubscription) error
	}{
		{"Activated", game.AddActivatedHandler, game.RemoveActivatedHandler},
		{"Deactivated", game.AddDeactivatedHandler, game.RemoveDeactivatedHandler},
		{"Exiting", game.AddExitingHandler, game.RemoveExitingHandler},
		{"Disposed", game.AddDisposedHandler, game.RemoveDisposedHandler},
	}
	tokens := make([]framework.EventSubscription, 0, len(adders))
	for _, entry := range adders {
		name := entry.name
		token, err := entry.add(func(any, *framework.EventArgs) error {
			seen = append(seen, name)
			return nil
		})
		if err != nil {
			t.Fatalf("Add%sHandler: %v", name, err)
		}
		if token == (framework.EventSubscription{}) {
			t.Fatalf("Add%sHandler returned the zero token for a real handler", name)
		}
		tokens = append(tokens, token)
	}

	if err := game.OnActivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("OnActivated: %v", err)
	}
	if err := game.OnDeactivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("OnDeactivated: %v", err)
	}
	if err := game.OnExiting(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("OnExiting: %v", err)
	}
	if strings.Join(seen, ",") != "Activated,Deactivated,Exiting" {
		t.Fatalf("observed %v, want the three raise sites in order", seen)
	}

	for i, entry := range adders {
		if err := entry.remove(tokens[i]); err != nil {
			t.Fatalf("Remove%sHandler: %v", entry.name, err)
		}
	}
	seen = nil
	if err := game.OnActivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("OnActivated after removal: %v", err)
	}
	if err := game.OnExiting(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("OnExiting after removal: %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("removed handlers still ran: %v", seen)
	}
}

// TestConsumerObservesTheExitingSenderQuirk proves the one-instruction
// difference from outside: OnExiting pushes `ldnull` where its two siblings
// push `this`.
func TestConsumerObservesTheExitingSenderQuirk(t *testing.T) {
	game, _ := newCanaryGame(t)
	var activatedSender, exitingSender any
	sawExiting := false
	if _, err := game.AddActivatedHandler(func(sender any, _ *framework.EventArgs) error {
		activatedSender = sender
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := game.AddExitingHandler(func(sender any, _ *framework.EventArgs) error {
		exitingSender = sender
		sawExiting = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// A sender argument is supplied to both, and both ignore it.
	decoy, _ := newCanaryGame(t)
	if err := game.OnActivated(decoy, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if err := game.OnExiting(decoy, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if activatedSender != any(game) {
		t.Fatalf("Activated sender = %v, want the Game", activatedSender)
	}
	if !sawExiting {
		t.Fatal("Exiting handler did not run")
	}
	if exitingSender != nil {
		t.Fatalf("Exiting sender = %v, want nil", exitingSender)
	}
}

// TestConsumerGameEventDuplicatesAndStaleRemovals holds the settled token rule
// on Game: two Adds of one handler are two registrations, each removed
// separately, and every kind of absent token is inert.
func TestConsumerGameEventDuplicatesAndStaleRemovals(t *testing.T) {
	game, _ := newCanaryGame(t)
	other, _ := newCanaryGame(t)
	calls := 0
	handler := func(any, *framework.EventArgs) error {
		calls++
		return nil
	}
	first, err := game.AddActivatedHandler(handler)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := game.AddActivatedHandler(handler); err != nil {
		t.Fatal(err)
	}
	if err := game.OnActivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("one handler added twice ran %d times, want 2", calls)
	}
	if err := game.RemoveActivatedHandler(first); err != nil {
		t.Fatal(err)
	}
	foreign, err := other.AddActivatedHandler(handler)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []framework.EventSubscription{first, {}, foreign} {
		if err := game.RemoveActivatedHandler(token); err != nil {
			t.Fatalf("an absent token reported %v, want nil", err)
		}
	}
	calls = 0
	if err := game.OnActivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("after one removal and three inert removals the handler ran %d times, want 1", calls)
	}
	// The other Game's registration is untouched by any of it.
	calls = 0
	if err := other.OnActivated(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("the other Game's registration ran %d times, want 1", calls)
	}
}

// TestGameEventsExposeNoNativeRegistrationHandle is the leak claim, made from a
// module that has no access to internal/interop at all. Every exported name in
// the family is spelled in terms of two adapter types and nothing else; there
// is no uintptr, no unsafe.Pointer and no CNA registration handle anywhere in
// the surface, and the assertions above are the compiler's proof.
func TestGameEventsExposeNoNativeRegistrationHandle(t *testing.T) {
	game, _ := newCanaryGame(t)
	token, err := game.AddDisposedHandler(func(any, *framework.EventArgs) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	// EventSubscription is a struct with no exported field, so a consumer can
	// hold it, compare it and hand it back, and can do nothing else with it.
	if got := reflect.TypeOf(token); got.Kind() != reflect.Struct {
		t.Fatalf("EventSubscription kind = %v, want a struct", got.Kind())
	}
	for i := 0; i < reflect.TypeOf(token).NumField(); i++ {
		field := reflect.TypeOf(token).Field(i)
		if field.IsExported() {
			t.Fatalf("EventSubscription exports field %q", field.Name)
		}
	}
	// And the Game's own event methods name no native type.
	gameType := reflect.TypeOf(game)
	for i := 0; i < gameType.NumMethod(); i++ {
		method := gameType.Method(i)
		if !strings.Contains(method.Name, "Handler") && !strings.HasPrefix(method.Name, "On") {
			continue
		}
		signature := method.Type.String()
		for _, banned := range []string{"uintptr", "unsafe.Pointer", "cgo.Handle", "interop."} {
			if strings.Contains(signature, banned) {
				t.Fatalf("%s exposes %q in %s", method.Name, banned, signature)
			}
		}
	}
}

// TestGameStaysUsableAfterEveryHandlerIsRemoved proves the native subscription
// is per Game rather than per handler: removing every Go handler leaves the
// Game fully usable, and a later Add is delivered to normally.
func TestGameStaysUsableAfterEveryHandlerIsRemoved(t *testing.T) {
	game, user := newCanaryGame(t)
	token, err := game.AddExitingHandler(func(any, *framework.EventArgs) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := game.RemoveExitingHandler(token); err != nil {
		t.Fatal(err)
	}
	// The component engine is untouched by any of it.
	component := NewRotator("after-removal")
	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	if err := framework.GameBaseInitialize(game); err != nil {
		t.Fatal(err)
	}
	if err := framework.GameBaseUpdate(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if component.Updates != 1 {
		t.Fatalf("component updated %d times after event removal, want 1", component.Updates)
	}
	// And a fresh subscription still works.
	late := 0
	if _, err := game.AddExitingHandler(func(any, *framework.EventArgs) error {
		late++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := game.OnExiting(nil, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if late != 1 {
		t.Fatalf("a handler added after every removal ran %d times, want 1", late)
	}
	_ = user
}

// ---------------------------------------------------------------------------
// Foundation 35 — Game's four frame-boundary virtuals, from outside.
// ---------------------------------------------------------------------------

// TestConsumerReachesTheFourFrameHooks proves the four members exist on Game
// with the exact signatures Microsoft's declarations map to, that BeginDraw's
// Boolean is a separate channel from its error, and that none of them touches
// the component engine.
func TestConsumerReachesTheFourFrameHooks(t *testing.T) {
	game, user := newCanaryGame(t)
	// The signatures, asserted at compile time.
	var (
		_ func() error         = game.BeginRun
		_ func() error         = game.EndRun
		_ func() (bool, error) = game.BeginDraw
		_ func() error         = game.EndDraw
	)

	component := NewRotator("frame-hooks")
	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := component.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	if err := user.Initialize(game); err != nil {
		t.Fatal(err)
	}

	if err := game.BeginRun(); err != nil {
		t.Fatalf("BeginRun: %v", err)
	}
	shouldDraw, err := game.BeginDraw()
	if err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if !shouldDraw {
		t.Fatal("BeginDraw refused the frame with no IGraphicsDeviceManager registered")
	}
	if err := game.EndDraw(); err != nil {
		t.Fatalf("EndDraw: %v", err)
	}
	if err := game.EndRun(); err != nil {
		t.Fatalf("EndRun: %v", err)
	}
	if component.Updates != 0 || component.Draws != 0 {
		t.Fatalf("a frame hook ran the component loop: updates=%d draws=%d",
			component.Updates, component.Draws)
	}
	// The component loop is still a different member, and still works.
	if err := user.Update(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if err := user.Draw(game, framework.GameTime{}); err != nil {
		t.Fatal(err)
	}
	if component.Updates != 1 || component.Draws != 1 {
		t.Fatalf("after the base calls: updates=%d draws=%d, want 1 and 1",
			component.Updates, component.Draws)
	}
}

// TestFrameHooksRefuseAnUnconstructedGameFromOutside covers the family's one
// Go-only failure, and proves a refused BeginDraw does not also admit the frame.
func TestFrameHooksRefuseAnUnconstructedGameFromOutside(t *testing.T) {
	zero := &framework.Game{}
	for name, call := range map[string]func() error{
		"BeginRun": zero.BeginRun,
		"EndRun":   zero.EndRun,
		"EndDraw":  zero.EndDraw,
	} {
		if err := call(); err == nil {
			t.Fatalf("%s on an unconstructed Game reported no error", name)
		}
	}
	shouldDraw, err := zero.BeginDraw()
	if err == nil {
		t.Fatal("BeginDraw on an unconstructed Game reported no error")
	}
	if shouldDraw {
		t.Fatal("a refused BeginDraw also admitted the frame")
	}
}

// TestTheOverrideContractStillHasExactlyFiveMembers is the compatibility claim
// this milestone makes to every consumer who already wrote a GameCallbacks
// implementation: nothing was added to the interface they satisfy, so their
// code still compiles and still runs.
func TestTheOverrideContractStillHasExactlyFiveMembers(t *testing.T) {
	contract := reflect.TypeOf((*framework.GameCallbacks)(nil)).Elem()
	if contract.NumMethod() != 5 {
		names := make([]string, 0, contract.NumMethod())
		for i := 0; i < contract.NumMethod(); i++ {
			names = append(names, contract.Method(i).Name)
		}
		t.Fatalf("GameCallbacks has %d members (%s), want exactly 5", contract.NumMethod(), strings.Join(names, ","))
	}
	// And the canary's own conformer, written before any of this existed,
	// still satisfies it.
	var _ framework.GameCallbacks = (*UserGame)(nil)
}
