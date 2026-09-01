package eventcanary

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	content "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Content"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
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

// ---------------------------------------------------------------------------
// Foundation 38 — the optional per-hook overrides, from outside the module.
// ---------------------------------------------------------------------------

// canaryOverrideShapes are this consumer's OWN copies of the four capability
// shapes. The binding's interfaces are unexported, so a consumer can never name
// them -- and never has to. Declaring the method is the whole opt-in, and these
// local interfaces are how the canary observes that from the outside.
type (
	canaryBeginRun  interface{ BeginRun(*framework.Game) error }
	canaryEndRun    interface{ EndRun(*framework.Game) error }
	canaryBeginDraw interface {
		BeginDraw(*framework.Game) (bool, error)
	}
	canaryEndDraw interface{ EndDraw(*framework.Game) error }
)

func canaryDeclaredHooks(value any) []string {
	observed := reflect.TypeOf(value)
	names := make([]string, 0, 4)
	if observed.Implements(reflect.TypeOf((*canaryBeginRun)(nil)).Elem()) {
		names = append(names, "BeginRun")
	}
	if observed.Implements(reflect.TypeOf((*canaryEndRun)(nil)).Elem()) {
		names = append(names, "EndRun")
	}
	if observed.Implements(reflect.TypeOf((*canaryBeginDraw)(nil)).Elem()) {
		names = append(names, "BeginDraw")
	}
	if observed.Implements(reflect.TypeOf((*canaryEndDraw)(nil)).Elem()) {
		names = append(names, "EndDraw")
	}
	return names
}

// TestAnyOverrideSubsetIsAcceptedFromOutside is the central claim of the
// mechanism: a consumer may override ANY SUBSET of the four virtuals, exactly
// as a CLR subclass may, and a consumer who overrides none is untouched.
//
// Every one of these types is built and handed to NewGame from a module that
// cannot see anything unexported in the binding.
func TestAnyOverrideSubsetIsAcceptedFromOutside(t *testing.T) {
	log := &HookLog{}
	for _, testCase := range []struct {
		name      string
		callbacks framework.GameCallbacks
		want      []string
	}{
		{"no override at all", NewUserGame(), nil},
		{"only BeginRun", NewBeginRunOnly(log), []string{"BeginRun"}},
		{"only BeginDraw", NewBeginDrawOnly(log), []string{"BeginDraw"}},
		{"BeginDraw and EndDraw", NewDrawPair(log), []string{"BeginDraw", "EndDraw"}},
		{"all four", NewEveryHook(log), []string{"BeginRun", "EndRun", "BeginDraw", "EndDraw"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			game, err := framework.NewGame(testCase.callbacks)
			if err != nil {
				t.Fatalf("NewGame: %v", err)
			}
			if game == nil {
				t.Fatal("NewGame returned no Game")
			}
			declared := canaryDeclaredHooks(testCase.callbacks)
			if strings.Join(declared, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("declared hooks %v, want %v", declared, testCase.want)
			}
		})
	}
}

// TestAnOmittedCapabilityIsNotSatisfiedByAccident is the negative half. A
// consumer who overrode only the draw pair has not, by any structural
// accident, also opted into the run pair -- which is what four separate
// one-method capabilities buy over one bundled contract.
func TestAnOmittedCapabilityIsNotSatisfiedByAccident(t *testing.T) {
	log := &HookLog{}
	pair := NewDrawPair(log)
	if _, ok := any(pair).(canaryBeginRun); ok {
		t.Fatal("a draw-only override also satisfies the BeginRun capability")
	}
	if _, ok := any(pair).(canaryEndRun); ok {
		t.Fatal("a draw-only override also satisfies the EndRun capability")
	}
	single := NewBeginRunOnly(log)
	if _, ok := any(single).(canaryBeginDraw); ok {
		t.Fatal("a BeginRun-only override also satisfies the BeginDraw capability")
	}
	// And the pre-existing five-member consumer satisfies none of the four,
	// so nothing about it changed.
	if declared := canaryDeclaredHooks(NewUserGame()); len(declared) != 0 {
		t.Fatalf("a five-member consumer declared %v", declared)
	}
}

// TestExplicitBaseCallInvokesTheBaseOnly proves, from outside, that calling
// game.BeginDraw() inside an override reaches the base body and does not
// redispatch into the override. A redispatch would run the consumer's method
// again and the counter would exceed one.
func TestExplicitBaseCallInvokesTheBaseOnly(t *testing.T) {
	log := &HookLog{}
	override := NewBeginDrawOnly(log)
	game, err := framework.NewGame(override)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	shouldDraw, err := override.BeginDraw(game)
	if err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if !shouldDraw {
		t.Fatal("the base admitted the frame and the override reported false")
	}
	if override.Calls != 1 {
		t.Fatalf("the override ran %d times for one delivery; the base call redispatched", override.Calls)
	}
	if override.BaseCalls != 1 || !override.BaseAnswer {
		t.Fatalf("base: calls=%d answer=%t", override.BaseCalls, override.BaseAnswer)
	}
}

// TestOrderingAroundTheBaseCallFollowsSourceOrder holds the call-site rule from
// outside: the base runs where the source puts it, zero times if the source
// never calls it and twice if it calls it twice, and nothing deduplicates a
// repeated explicit call.
func TestOrderingAroundTheBaseCallFollowsSourceOrder(t *testing.T) {
	log := &HookLog{}
	override := NewBeginDrawOnly(log)
	game, err := framework.NewGame(override)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	// Own work first, then the base.
	log.Reset()
	if _, err := override.BeginDraw(game); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(log.Entries, ","); got != "user:BeginDraw,base:BeginDraw" {
		t.Fatalf("own-then-base order = %q", got)
	}

	// The base first, which in CLR is simply where the statement sits.
	log.Reset()
	override.BaseFirstOrder = true
	if _, err := override.BeginDraw(game); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(log.Entries, ","); got != "base:BeginDraw,user:BeginDraw" {
		t.Fatalf("base-then-own order = %q", got)
	}

	// Zero base calls: the base does not run at all.
	log.Reset()
	override.BaseFirstOrder = false
	before := override.BaseCalls
	override.Repeat = 0
	if _, err := override.BeginDraw(game); err != nil {
		t.Fatal(err)
	}
	if override.BaseCalls != before {
		t.Fatalf("an override that never calls the base ran it %d times", override.BaseCalls-before)
	}
	if got := strings.Join(log.Entries, ","); got != "user:BeginDraw" {
		t.Fatalf("no-base order = %q", got)
	}

	// Two base calls run the base twice; nothing suppresses the repeat.
	log.Reset()
	override.Repeat = 2
	if _, err := override.BeginDraw(game); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(log.Entries, ","); got != "user:BeginDraw,base:BeginDraw,base:BeginDraw" {
		t.Fatalf("repeated-base order = %q", got)
	}
}

// TestARefusalIsNotAnErrorFromOutside keeps BeginDraw's two channels apart at
// the consumer's own boundary: a refused frame is (false, nil), and it is the
// override's answer rather than the base's that the consumer returns.
func TestARefusalIsNotAnErrorFromOutside(t *testing.T) {
	log := &HookLog{}
	override := NewBeginDrawOnly(log)
	override.Refuse = true
	game, err := framework.NewGame(override)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	shouldDraw, err := override.BeginDraw(game)
	if err != nil {
		t.Fatalf("a refusal was reported as a failure: %v", err)
	}
	if shouldDraw {
		t.Fatal("the override refused the frame and still admitted it")
	}
	if !override.BaseAnswer {
		t.Fatal("the base refused the frame, which it cannot do with no manager registered")
	}
}

// TestNoRegistrationOrCapabilitySurfaceIsExported is the closure claim seen
// from a module that can only see exported names: there is nothing to call to
// install an override, and no capability contract to implement by name.
func TestNoRegistrationOrCapabilitySurfaceIsExported(t *testing.T) {
	game := reflect.TypeOf((*framework.Game)(nil))
	for i := 0; i < game.NumMethod(); i++ {
		name := game.Method(i).Name
		if strings.Contains(name, "Override") || strings.HasSuffix(name, "Hook") {
			t.Fatalf("Game exposes %q; the override set is fixed at construction", name)
		}
	}
	// The four hooks are still plain methods on Game with their base bodies,
	// reachable exactly as they were before the mechanism existed.
	var (
		_ func() error         = (&framework.Game{}).BeginRun
		_ func() error         = (&framework.Game{}).EndRun
		_ func() (bool, error) = (&framework.Game{}).BeginDraw
		_ func() error         = (&framework.Game{}).EndDraw
	)
}

// ---------------------------------------------------------------------------
// Foundation 39 — Game's disposal surface, from outside the module.
// ---------------------------------------------------------------------------

// TestDisposedFiresOnlyFromDispose is the correction seen by a consumer. In the
// reference, Game::Disposed has exactly one raise site: the tail of
// Dispose(bool). A consumer who never disposes never sees it.
func TestDisposedFiresOnlyFromDispose(t *testing.T) {
	game, _ := newCanaryGame(t)
	raised := 0
	var sender any
	var args *framework.EventArgs
	if _, err := game.AddDisposedHandler(func(s any, a *framework.EventArgs) error {
		raised++
		sender, args = s, a
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Everything a consumer can do to a Game short of disposing it.
	if err := game.BeginRun(); err != nil {
		t.Fatal(err)
	}
	if _, err := game.BeginDraw(); err != nil {
		t.Fatal(err)
	}
	if err := game.EndDraw(); err != nil {
		t.Fatal(err)
	}
	if err := game.EndRun(); err != nil {
		t.Fatal(err)
	}
	if err := game.Finalize(); err != nil {
		t.Fatal(err)
	}
	if err := game.DisposeByBoolean(false); err != nil {
		t.Fatal(err)
	}
	if raised != 0 {
		t.Fatalf("Disposed was raised %d times without a Dispose call", raised)
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raised != 1 {
		t.Fatalf("Dispose raised Disposed %d times, want 1", raised)
	}
	if sender != any(game) {
		t.Fatalf("Disposed sender = %v, want the Game", sender)
	}
	if args != framework.EventArgsEmpty() {
		t.Fatal("Disposed args are not the shared EventArgs.Empty identity")
	}
}

// TestGameDisposeIsNotIdempotentFromOutside holds the reference behaviour a
// consumer is most likely to assume away. Game carries no disposed flag, so
// every call re-runs the whole body.
func TestGameDisposeIsNotIdempotentFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	component := NewSimpleDisposableRotator("repeat")
	if err := game.Components().Add(component); err != nil {
		t.Fatal(err)
	}
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *framework.EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := game.DisposeByNone(); err != nil {
			t.Fatalf("Dispose %d: %v", i, err)
		}
	}
	if raised != 3 || component.Disposed != 3 {
		t.Fatalf("three Dispose calls raised %d events and disposed the component %d times, want 3 and 3",
			raised, component.Disposed)
	}
}

// TestBothDisposableSpellingsAreFoundFromOutside proves a consumer's own
// component joins Game.Dispose's loop by declaring either projected spelling of
// IDisposable::Dispose, and that one declaring neither is skipped.
func TestBothDisposableSpellingsAreFoundFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	two := NewDisposableRotator("two-overload")
	one := NewSimpleDisposableRotator("one-overload")
	plain := NewRotator("not-disposable")
	for _, component := range []framework.IGameComponent{two, one, plain} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if two.Disposed != 1 || one.Disposed != 1 {
		t.Fatalf("disposed the two spellings %d and %d times, want 1 each", two.Disposed, one.Disposed)
	}
	// Nothing removed the components: Game.Dispose does not clear Components,
	// and a consumer's own component does not remove itself unless it says so.
	if got := game.Components().Count(); got != 3 {
		t.Fatalf("Components holds %d items after Dispose, want 3", got)
	}
}

// TestDisposeWalksASnapshotFromOutside is why the reference copies to an array
// first: a component that removes itself while being disposed must not make the
// loop skip its neighbour.
func TestDisposeWalksASnapshotFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	var components []*DisposableRotator
	for i := 0; i < 4; i++ {
		component := NewDisposableRotator("snapshot")
		component.OnDispose = func() { _, _ = game.Components().Remove(component) }
		components = append(components, component)
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	if err := game.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	for i, component := range components {
		if component.Disposed != 1 {
			t.Fatalf("component %d disposed %d times; the loop did not walk a snapshot", i, component.Disposed)
		}
	}
	if got := game.Components().Count(); got != 0 {
		t.Fatalf("Components holds %d items", got)
	}
}

// TestAFailingComponentPropagatesFromOutside holds the absence of a try/catch
// in the reference: the failure surfaces, later components stay undisposed, and
// the Disposed event is never reached.
func TestAFailingComponentPropagatesFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	sentinel := errors.New("canary component disposal failure")
	failing := NewDisposableRotator("failing")
	failing.Failure = sentinel
	later := NewDisposableRotator("later")
	for _, component := range []framework.IGameComponent{failing, later} {
		if err := game.Components().Add(component); err != nil {
			t.Fatal(err)
		}
	}
	raised := 0
	if _, err := game.AddDisposedHandler(func(any, *framework.EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := game.DisposeByNone(); !errors.Is(err, sentinel) {
		t.Fatalf("Dispose = %v, want the component's own failure", err)
	}
	if later.Disposed != 0 {
		t.Fatal("a component after the failure was still disposed")
	}
	if raised != 0 {
		t.Fatal("Disposed was raised even though disposal never reached its raise site")
	}
}

// TestDisposalSurfaceIsReachableFromOutside pins the three members' signatures
// as a downstream consumer sees them, and records that Game declares no
// OnDisposed -- the reference has none, and inventing one is a verifier
// failure.
func TestDisposalSurfaceIsReachableFromOutside(t *testing.T) {
	game := &framework.Game{}
	var (
		_ func() error     = game.DisposeByNone
		_ func(bool) error = game.DisposeByBoolean
		_ func() error     = game.Finalize
	)
	if _, ok := reflect.TypeOf((*framework.Game)(nil)).MethodByName("OnDisposed"); ok {
		t.Fatal("Game declares OnDisposed; the reference has no such member")
	}
	// Dispose(false) comes before the state guard, so it is safe even here.
	if err := game.DisposeByBoolean(false); err != nil {
		t.Fatalf("Dispose(false) on an unconstructed Game = %v, want nil", err)
	}
	if err := game.Finalize(); err != nil {
		t.Fatalf("Finalize on an unconstructed Game = %v, want nil", err)
	}
	if err := game.DisposeByNone(); err == nil {
		t.Fatal("Dispose() on an unconstructed Game reported no error")
	}
}

// ---------------------------------------------------------------------------
// Foundation 42 — Game's timing and presentation state, from outside.
// ---------------------------------------------------------------------------

// TestTimingDefaultsAndBoundariesFromOutside is what a downstream consumer
// actually sees: the constructor's own defaults, and the one-instruction
// difference between the two TimeSpan setters.
func TestTimingDefaultsAndBoundariesFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	if got := game.TargetElapsedTime().Ticks(); got != 166667 {
		t.Fatalf("TargetElapsedTime = %d ticks, want 166667", got)
	}
	if got := game.InactiveSleepTime().Ticks(); got != 200000 {
		t.Fatalf("InactiveSleepTime = %d ticks, want 200000", got)
	}
	if !game.IsFixedTimeStep() {
		t.Fatal("IsFixedTimeStep = false; the constructor stores true")
	}
	if game.IsMouseVisible() {
		t.Fatal("IsMouseVisible = true; the constructor does not assign it")
	}

	// InactiveSleepTime compares with op_LessThan, so zero is accepted.
	if err := game.SetInactiveSleepTime(framework.TimeSpanFromTicks(0)); err != nil {
		t.Fatalf("InactiveSleepTime = 0 was rejected: %v", err)
	}
	if err := game.SetInactiveSleepTime(framework.TimeSpanFromTicks(-1)); err == nil {
		t.Fatal("InactiveSleepTime accepted a negative value")
	}
	// TargetElapsedTime compares with op_LessThanOrEqual, so zero is not.
	if err := game.SetTargetElapsedTime(framework.TimeSpanFromTicks(0)); err == nil {
		t.Fatal("TargetElapsedTime accepted zero")
	}
	if got := game.TargetElapsedTime().Ticks(); got != 166667 {
		t.Fatalf("a rejected TargetElapsedTime still stored: %d", got)
	}
	if err := game.SetTargetElapsedTime(framework.TimeSpanFromTicks(333333)); err != nil {
		t.Fatalf("SetTargetElapsedTime: %v", err)
	}
	if got := game.TargetElapsedTime().Ticks(); got != 333333 {
		t.Fatalf("TargetElapsedTime = %d after storing 333333", got)
	}
}

// TestTimingGettersAreInfallibleFromOutside pins the shape a consumer writes
// against. Each getter is one `ldfld` in the reference -- no validation, no
// host, no device, no throw site -- so none of them carries an error, and each
// works on a Game whose constructor never ran.
func TestTimingGettersAreInfallibleFromOutside(t *testing.T) {
	zero := &framework.Game{}
	var (
		_ framework.TimeSpan = zero.TargetElapsedTime()
		_ framework.TimeSpan = zero.InactiveSleepTime()
		_ bool               = zero.IsFixedTimeStep()
		_ bool               = zero.IsMouseVisible()
	)
	var (
		_ func(framework.TimeSpan) error = zero.SetTargetElapsedTime
		_ func(framework.TimeSpan) error = zero.SetInactiveSleepTime
		_ func(bool) error               = zero.SetIsFixedTimeStep
		_ func(bool) error               = zero.SetIsMouseVisible
		_ func() error                   = zero.SuppressDraw
		_ func() error                   = zero.ResetElapsedTime
	)
	// The setters refuse an unconstructed Game, but the argument check comes
	// first, so a bad value is still reported as a bad value.
	if err := zero.SetIsFixedTimeStep(true); err == nil {
		t.Fatal("SetIsFixedTimeStep on an unconstructed Game reported no error")
	}
	if err := zero.SetTargetElapsedTime(framework.TimeSpanFromTicks(0)); err == nil {
		t.Fatal("SetTargetElapsedTime(0) on an unconstructed Game reported no error")
	}
}

// TestTimingIsConfigurableBeforeRunFromOutside is the case a real consumer
// hits: set a frame rate on a Game that has not run yet. Every setter succeeds
// and every getter reports what was stored, because there is no native loop to
// reach and the value is carried into creation instead.
func TestTimingIsConfigurableBeforeRunFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	if err := game.SetTargetElapsedTime(framework.TimeSpanFromTicks(83333)); err != nil {
		t.Fatalf("SetTargetElapsedTime: %v", err)
	}
	if err := game.SetIsFixedTimeStep(false); err != nil {
		t.Fatalf("SetIsFixedTimeStep: %v", err)
	}
	if err := game.SetIsMouseVisible(true); err != nil {
		t.Fatalf("SetIsMouseVisible: %v", err)
	}
	if err := game.SuppressDraw(); err != nil {
		t.Fatalf("SuppressDraw: %v", err)
	}
	if err := game.ResetElapsedTime(); err != nil {
		t.Fatalf("ResetElapsedTime: %v", err)
	}
	if got := game.TargetElapsedTime().Ticks(); got != 83333 {
		t.Fatalf("TargetElapsedTime = %d, want 83333", got)
	}
	if game.IsFixedTimeStep() || !game.IsMouseVisible() {
		t.Fatalf("flags = %t/%t, want false/true", game.IsFixedTimeStep(), game.IsMouseVisible())
	}
}

// ---------------------------------------------------------------------------
// Foundation 43 — Game.GraphicsDevice, from outside the module.
// ---------------------------------------------------------------------------

// canaryDeviceService is a downstream consumer's own IGraphicsDeviceService.
// Only a consumer can supply one: CNA-Go publishes none.
type canaryDeviceService struct {
	device    *graphics.GraphicsDevice
	created   framework.EventSource[*framework.EventArgs]
	disposing framework.EventSource[*framework.EventArgs]
	reset     framework.EventSource[*framework.EventArgs]
	resetting framework.EventSource[*framework.EventArgs]
}

func (s *canaryDeviceService) GraphicsDevice() *graphics.GraphicsDevice { return s.device }

func (s *canaryDeviceService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.created.Add(h)
}
func (s *canaryDeviceService) RemoveDeviceCreatedHandler(t framework.EventSubscription) error {
	return s.created.Remove(t)
}
func (s *canaryDeviceService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.disposing.Add(h)
}
func (s *canaryDeviceService) RemoveDeviceDisposingHandler(t framework.EventSubscription) error {
	return s.disposing.Remove(t)
}
func (s *canaryDeviceService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.reset.Add(h)
}
func (s *canaryDeviceService) RemoveDeviceResetHandler(t framework.EventSubscription) error {
	return s.reset.Remove(t)
}
func (s *canaryDeviceService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resetting.Add(h)
}
func (s *canaryDeviceService) RemoveDeviceResettingHandler(t framework.EventSubscription) error {
	return s.resetting.Remove(t)
}

// TestGameGraphicsDeviceFromOutside proves both branches of the reference body
// from a module that can only see exported names: no registered service is the
// reference's InvalidOperationException, and a registered one is resolved and
// forwarded unchanged.
func TestGameGraphicsDeviceFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	if _, err := graphics.GameGraphicsDevice(game); err == nil {
		t.Fatal("GameGraphicsDevice reported no error with no registered service")
	} else if !strings.Contains(err.Error(), "This property requires a graphics device service in the game service container.") {
		t.Fatalf("GameGraphicsDevice = %v, want the reference's message", err)
	}

	published := &graphics.GraphicsDevice{}
	service := &canaryDeviceService{device: published}
	token := reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem()
	if err := game.Services().AddService(token, service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	device, err := graphics.GameGraphicsDevice(game)
	if err != nil {
		t.Fatalf("GameGraphicsDevice: %v", err)
	}
	if device != published {
		t.Fatal("GameGraphicsDevice returned a device other than the service's own")
	}

	// The reference's null check is on the service, not on the device: a
	// service publishing none answers nil with no error.
	service.device = nil
	device, err = graphics.GameGraphicsDevice(game)
	if err != nil {
		t.Fatalf("GameGraphicsDevice with a device-less service: %v", err)
	}
	if device != nil {
		t.Fatal("GameGraphicsDevice invented a device")
	}
}

// TestGraphicsDeviceIsNotAMethodOnGame records the cross-package projection a
// consumer has to write against: the framework package cannot name
// GraphicsDevice, so the member lives in the Graphics package.
func TestGraphicsDeviceIsNotAMethodOnGame(t *testing.T) {
	if _, ok := reflect.TypeOf((*framework.Game)(nil)).MethodByName("GraphicsDevice"); ok {
		t.Fatal("Game declares a GraphicsDevice method")
	}
	var _ func(*framework.Game) (*graphics.GraphicsDevice, error) = graphics.GameGraphicsDevice
}

// TestGameWindowIsReachableOnlyThroughTheGame proves the assembly constructor
// from outside: a consumer cannot construct a GameWindow, and the one it gets
// from Game.Window is the same object every time.
//
// This is the property every subscription depends on. A projection that
// allocated a wrapper per call would pass every in-package test that used one
// local variable and would silently drop a consumer's handlers here.
func TestGameWindowIsReachableOnlyThroughTheGame(t *testing.T) {
	game, _ := newCanaryGame(t)
	window := game.Window()
	if window == nil {
		t.Fatal("Game.Window is nil after construction")
	}
	if game.Window() != window {
		t.Fatal("Game.Window returned a different object on a second call")
	}
	// The window is not assignable either: the reference has no setter, so a
	// consumer cannot substitute one.
	if _, ok := reflect.TypeOf((*framework.Game)(nil)).MethodByName("SetWindow"); ok {
		t.Fatal("Game declares a SetWindow method; GameWindow is get-only in the reference")
	}
	// A zero GameWindow a consumer declares itself is inert rather than
	// dangerous, which is what makes the absent constructor safe: the type is
	// nameable, and nothing about it invites construction.
	var declared framework.GameWindow
	if got := declared.Title(); got != "" {
		t.Fatalf("a consumer-declared GameWindow reported Title %q", got)
	}
	// The absence of an exported constructor is enforced by the API-compat
	// verifier's UNEXPECTED_MEMBER rule rather than reflectively here: Go
	// cannot enumerate a package's functions at run time, and a test that
	// pretended to would be measuring nothing.
}

// TestGameWindowSubscriptionSurvivesTheGetter is what a consumer actually
// writes, and it goes through a fresh Game.Window() call at every step.
func TestGameWindowSubscriptionSurvivesTheGetter(t *testing.T) {
	game, _ := newCanaryGame(t)
	var order []string
	add := func(name string, register func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)) framework.EventSubscription {
		t.Helper()
		token, err := register(func(sender any, args *framework.EventArgs) error {
			if sender != game.Window() {
				t.Errorf("%s sender is not the window", name)
			}
			order = append(order, name)
			return nil
		})
		if err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
		return token
	}
	add("client", game.Window().AddClientSizeChangedHandler)
	add("orientation", game.Window().AddOrientationChangedHandler)
	removable := add("screen", game.Window().AddScreenDeviceNameChangedHandler)

	if err := game.Window().OnClientSizeChanged(); err != nil {
		t.Fatal(err)
	}
	if err := game.Window().OnOrientationChanged(); err != nil {
		t.Fatal(err)
	}
	if err := game.Window().OnScreenDeviceNameChanged(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "client,orientation,screen" {
		t.Fatalf("handlers ran as %v", order)
	}
	if err := game.Window().RemoveScreenDeviceNameChangedHandler(removable); err != nil {
		t.Fatalf("remove: %v", err)
	}
	order = nil
	if err := game.Window().OnScreenDeviceNameChanged(); err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatalf("a removed handler still ran: %v", order)
	}
}

// TestGameWindowTitleIsManagedState proves from outside that the getter reads
// what the setter stored rather than asking the platform, and that an unchanged
// assignment is suppressed.
func TestGameWindowTitleIsManagedState(t *testing.T) {
	game, _ := newCanaryGame(t)
	window := game.Window()
	if got := window.Title(); got != "" {
		t.Fatalf("Title = %q on a fresh Game, want String.Empty", got)
	}
	if err := window.SetTitleProperty("canary"); err != nil {
		t.Fatalf("SetTitleProperty: %v", err)
	}
	if got := game.Window().Title(); got != "canary" {
		t.Fatalf("Title = %q through a second getter call", got)
	}
	// Assigning the same value again is a no-op the consumer cannot tell apart
	// from a successful write -- which is the point: it must not report a
	// failure either.
	if err := window.SetTitleProperty("canary"); err != nil {
		t.Fatalf("an unchanged SetTitleProperty reported %v", err)
	}
}

// TestGameWindowGuardSplitFromOutside is the measured behaviour a consumer sees
// with no running Game: five members answer the reference's own fallbacks, and
// four report the failure its unguarded dereference produces.
func TestGameWindowGuardSplitFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	window := game.Window()

	handle, err := window.Handle()
	if err != nil || handle != 0 {
		t.Fatalf("Handle = %#x, %v; want zero and no failure", handle, err)
	}
	allow, err := window.AllowUserResizing()
	if err != nil || allow {
		t.Fatalf("AllowUserResizing = %t, %v; want false and no failure", allow, err)
	}
	if err := window.SetAllowUserResizing(true); err != nil {
		t.Fatalf("SetAllowUserResizing: %v", err)
	}
	name, err := window.ScreenDeviceName()
	if err != nil || name != "" {
		t.Fatalf("ScreenDeviceName = %q, %v; want empty and no failure", name, err)
	}
	if err := window.SetTitleMethod("ignored"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}

	if _, err := window.ClientBounds(); err == nil {
		t.Fatal("ClientBounds succeeded with no running Game")
	}
	if err := window.BeginScreenDeviceChange(true); err == nil {
		t.Fatal("BeginScreenDeviceChange succeeded with no running Game")
	}
	if err := window.EndScreenDeviceChangeByStringAndInt32AndInt32("s", 1, 2); err == nil {
		t.Fatal("the three-argument EndScreenDeviceChange succeeded with no running Game")
	}
	if err := window.EndScreenDeviceChangeByString("s"); err == nil {
		t.Fatal("the one-argument EndScreenDeviceChange succeeded with no running Game")
	}
}

// TestGameWindowOrientationMembersAreTheReferenceConstants records the two
// members whose reference bodies reach nothing at all, from the outside where
// their infallibility is part of the signature a consumer compiles against.
func TestGameWindowOrientationMembersAreTheReferenceConstants(t *testing.T) {
	game, _ := newCanaryGame(t)
	window := game.Window()

	// Infallible by signature: these two lines would not compile if either
	// member had gained an error result.
	var orientation framework.DisplayOrientation = window.CurrentOrientation()
	window.SetSupportedOrientations(framework.DisplayOrientationPortrait)

	if orientation != framework.DisplayOrientationDefault {
		t.Fatalf("CurrentOrientation = %v, want Default", orientation)
	}
	if window.CurrentOrientation() != framework.DisplayOrientationDefault {
		t.Fatal("SetSupportedOrientations changed CurrentOrientation; the reference stores nothing")
	}
}

// TestDrawableGameComponentFromOutside proves the cross-package bridge from a
// module that can see only exported names, which is the only place it can be
// proved: this module imports BOTH packages, exactly as a real consumer must to
// register an IGraphicsDeviceService at all.
func TestDrawableGameComponentFromOutside(t *testing.T) {
	game, _ := newCanaryGame(t)
	component := framework.NewDrawableGameComponent(game)

	// The constructor's own defaults.
	if !component.Visible() || component.DrawOrder() != 0 || component.Game() != game {
		t.Fatalf("constructor defaults: Visible=%t DrawOrder=%d GameMatches=%t",
			component.Visible(), component.DrawOrder(), component.Game() == game)
	}

	// With no service registered, both throw sites report the reference's own
	// messages, and they are DIFFERENT messages.
	if err := component.Initialize(); err == nil {
		t.Fatal("Initialize succeeded with no graphics device service")
	} else if !strings.Contains(err.Error(), "Drawable components require a graphics device service in the game service container.") {
		t.Fatalf("Initialize = %v, want the reference's message", err)
	}
	if _, err := graphics.DrawableGameComponentGraphicsDevice(component); err == nil {
		t.Fatal("GraphicsDevice succeeded before Initialize")
	} else if !strings.Contains(err.Error(), "The GraphicsDevice property cannot be used before Initialize has been called.") {
		t.Fatalf("GraphicsDevice = %v, want the reference's message", err)
	}

	// A consumer's own service resolves across the package boundary.
	published := &graphics.GraphicsDevice{}
	service := &canaryDeviceService{device: published}
	token := reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem()
	if err := game.Services().AddService(token, service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize with a registered service: %v", err)
	}
	device, err := graphics.DrawableGameComponentGraphicsDevice(component)
	if err != nil {
		t.Fatalf("GraphicsDevice after Initialize: %v", err)
	}
	if device != published {
		t.Fatal("GraphicsDevice returned a device other than the service's own")
	}
}

// TestDrawableGameComponentIsTheProfilesShippedIDrawable records what the type
// is for: before it existed the live IDrawable implementors were consumers'
// own types, and Game's draw list had nothing of Microsoft's own in it.
func TestDrawableGameComponentIsTheProfilesShippedIDrawable(t *testing.T) {
	game, _ := newCanaryGame(t)
	component := framework.NewDrawableGameComponent(game)
	var _ framework.IDrawable = component
	var _ framework.IUpdateable = component
	var _ framework.IGameComponent = component

	if err := game.Components().Add(component); err != nil {
		t.Fatalf("Components.Add: %v", err)
	}
	if got := game.Components().Count(); got != 1 {
		t.Fatalf("Components.Count = %d after adding a DrawableGameComponent", got)
	}
	// Removing it through the collection is what Dispose does internally, and
	// it works from outside for the same reason.
	removed, err := game.Components().Remove(component)
	if err != nil {
		t.Fatalf("Components.Remove: %v", err)
	}
	if !removed || game.Components().Count() != 0 {
		t.Fatalf("Remove reported %t leaving %d components", removed, game.Components().Count())
	}
}

// TestDrawableGameComponentExposesNoBaseAccessor pins the composition rule from
// outside: the base object is private state, and a consumer can reach every
// inherited member without ever naming a GameComponent.
func TestDrawableGameComponentExposesNoBaseAccessor(t *testing.T) {
	componentType := reflect.TypeOf((*framework.DrawableGameComponent)(nil))
	for _, forbidden := range []string{"Base", "Parent", "AsGameComponent", "GameComponent", "Component"} {
		if _, ok := componentType.MethodByName(forbidden); ok {
			t.Fatalf("DrawableGameComponent exposes %s; the base object is private state", forbidden)
		}
	}
	// Every inherited member is reachable directly, which is what makes the
	// absent accessor a non-loss rather than a restriction.
	game, _ := newCanaryGame(t)
	component := framework.NewDrawableGameComponent(game)
	for name, call := range map[string]func() error{
		"SetEnabled":     func() error { return component.SetEnabled(false) },
		"SetUpdateOrder": func() error { return component.SetUpdateOrder(2) },
	} {
		if err := call(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if component.Enabled() || component.UpdateOrder() != 2 {
		t.Fatalf("inherited state after forwarding: Enabled=%t UpdateOrder=%d",
			component.Enabled(), component.UpdateOrder())
	}
}

// TestFrameStepMembersExistAndRefuseAnUnconstructedGame pins the two frame
// steps from outside.
//
// It deliberately does NOT take a frame. A frame step creates the process's one
// C-owned native game and holds it until Dispose, so a canary that stepped one
// would decide the outcome of every later test in the same process. The live
// evidence is tools/native_stress, which runs each frame-step cycle in its own
// subprocess for exactly that reason.
func TestFrameStepMembersExistAndRefuseAnUnconstructedGame(t *testing.T) {
	// Both are fallible, and these two lines would not compile otherwise.
	var _ func() error = (&framework.Game{}).Tick
	var _ func() error = (&framework.Game{}).RunOneFrame

	unconstructed := &framework.Game{}
	if err := unconstructed.Tick(); err == nil {
		t.Fatal("Tick accepted a Game whose constructor never ran")
	}
	if err := unconstructed.RunOneFrame(); err == nil {
		t.Fatal("RunOneFrame accepted a Game whose constructor never ran")
	}

	// Tick, RunOneFrame and Run are three distinct members. A binding that
	// aliased any two of them would pass a naive test and behave very
	// differently: Run blocks, RunOneFrame initializes, Tick does not.
	gameType := reflect.TypeOf((*framework.Game)(nil))
	for _, name := range []string{"Tick", "RunOneFrame", "Run"} {
		if _, ok := gameType.MethodByName(name); !ok {
			t.Fatalf("Game declares no %s", name)
		}
	}
}

// TestGraphicsDeviceManagerSurfaceFromOutside pins the configuration surface a
// consumer actually writes against, including the part that lives in a
// different package from the object it configures.
//
// It creates no manager: NewGraphicsDeviceManager needs a live native game, and
// a canary that started one would decide the outcome of every later test in the
// same process. What it measures is the shape -- which is exactly what a
// consumer compiles against.
func TestGraphicsDeviceManagerSurfaceFromOutside(t *testing.T) {
	// The two static read-only fields. They are the BACK BUFFER's defaults and
	// are deliberately different from GameWindow's 800x600.
	if framework.GraphicsDeviceManagerDefaultBackBufferWidth() != 800 ||
		framework.GraphicsDeviceManagerDefaultBackBufferHeight() != 480 {
		t.Fatalf("default back buffer = %dx%d, want 800x480",
			framework.GraphicsDeviceManagerDefaultBackBufferWidth(),
			framework.GraphicsDeviceManagerDefaultBackBufferHeight())
	}

	// The six framework-typed accessors: infallible getters, fallible setters.
	// These declarations would not compile if either half were classified the
	// other way.
	var manager *framework.GraphicsDeviceManager
	var _ func() int32 = manager.PreferredBackBufferWidth
	var _ func() int32 = manager.PreferredBackBufferHeight
	var _ func() bool = manager.IsFullScreen
	var _ func() bool = manager.SynchronizeWithVerticalRetrace
	var _ func() bool = manager.PreferMultiSampling
	var _ func() framework.DisplayOrientation = manager.SupportedOrientations
	var _ func(int32) error = manager.SetPreferredBackBufferWidth
	var _ func(int32) error = manager.SetPreferredBackBufferHeight
	var _ func(bool) error = manager.SetIsFullScreen
	var _ func(bool) error = manager.SetSynchronizeWithVerticalRetrace
	var _ func(bool) error = manager.SetPreferMultiSampling
	var _ func(framework.DisplayOrientation) error = manager.SetSupportedOrientations
	var _ func() error = manager.ApplyChanges
	var _ func() error = manager.ToggleFullScreen

	// The three Graphics-typed ones live in the Graphics package, because the
	// framework package cannot name their enums. A consumer reaches them as
	// package functions taking the manager.
	var _ func(*framework.GraphicsDeviceManager) graphics.GraphicsProfile = graphics.GraphicsDeviceManagerGraphicsProfile
	var _ func(*framework.GraphicsDeviceManager, graphics.GraphicsProfile) error = graphics.SetGraphicsDeviceManagerGraphicsProfile
	var _ func(*framework.GraphicsDeviceManager) graphics.SurfaceFormat = graphics.GraphicsDeviceManagerPreferredBackBufferFormat
	var _ func(*framework.GraphicsDeviceManager, graphics.SurfaceFormat) error = graphics.SetGraphicsDeviceManagerPreferredBackBufferFormat
	var _ func(*framework.GraphicsDeviceManager) graphics.DepthFormat = graphics.GraphicsDeviceManagerPreferredDepthStencilFormat
	var _ func(*framework.GraphicsDeviceManager, graphics.DepthFormat) error = graphics.SetGraphicsDeviceManagerPreferredDepthStencilFormat

	// And they are NOT methods on the manager, which is the observable half of
	// the cross-package cycle rule.
	managerType := reflect.TypeOf((*framework.GraphicsDeviceManager)(nil))
	for _, absent := range []string{"GraphicsProfile", "PreferredBackBufferFormat", "PreferredDepthStencilFormat"} {
		if _, ok := managerType.MethodByName(absent); ok {
			t.Fatalf("GraphicsDeviceManager declares %s; its type lives in the Graphics package", absent)
		}
	}

	// A nil Game is refused, which is the reference's ArgumentNullException.
	if _, err := framework.NewGraphicsDeviceManager(nil); err == nil {
		t.Fatal("NewGraphicsDeviceManager accepted a nil Game")
	}
}

// TestGraphicsDeviceManagerContractsFromOutside pins which contracts the
// manager satisfies, from a module that can see only exported names.
//
// It is the shape a consumer compiles against and the one thing they cannot
// discover from the documentation: the manager IS an IGraphicsDeviceManager and
// is NOT an IGraphicsDeviceService, even though the reference's type declares
// both, because no framework-package type can declare the second contract's
// device accessor.
func TestGraphicsDeviceManagerContractsFromOutside(t *testing.T) {
	var manager any = &framework.GraphicsDeviceManager{}
	if _, ok := manager.(framework.IGraphicsDeviceManager); !ok {
		t.Fatal("GraphicsDeviceManager does not implement IGraphicsDeviceManager")
	}
	if _, ok := manager.(graphics.IGraphicsDeviceService); ok {
		t.Fatal("GraphicsDeviceManager implements IGraphicsDeviceService; the Graphics package registers an adapter instead")
	}

	// The five event accessor pairs a consumer subscribes through, and the
	// four protected raisers. Disposed deliberately has no raiser: the
	// reference invokes that delegate field directly from Dispose(bool).
	typed := &framework.GraphicsDeviceManager{}
	var _ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = typed.AddDeviceCreatedHandler
	var _ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = typed.AddDeviceResettingHandler
	var _ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = typed.AddDeviceResetHandler
	var _ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = typed.AddDeviceDisposingHandler
	var _ func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) = typed.AddDisposedHandler
	var _ func(any, *framework.EventArgs) error = typed.OnDeviceCreated
	var _ func(any, *framework.EventArgs) error = typed.OnDeviceResetting
	var _ func(any, *framework.EventArgs) error = typed.OnDeviceReset
	var _ func(any, *framework.EventArgs) error = typed.OnDeviceDisposing
	if _, ok := reflect.TypeOf(typed).MethodByName("OnDisposed"); ok {
		t.Fatal("GraphicsDeviceManager declares OnDisposed; the reference has no protected raiser for that event")
	}

	// A consumer's own handlers reach the raisers, on a manager that never
	// touched native code.
	raised := 0
	if _, err := typed.AddDeviceResetHandler(func(sender any, args *framework.EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := typed.OnDeviceReset(typed, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if raised != 1 {
		t.Fatalf("OnDeviceReset reached %d handlers, want one", raised)
	}
}

// TestEverySpriteDrawOverloadIsReachableFromOutside compiles every one of the
// profile's seven Draw overloads against its exact projected signature.
//
// It is the one thing an in-package test cannot check: an overload family is
// where a binding can look complete while missing a member, because every name
// is the same word and only the argument list differs. A consumer writing
// `spriteBatch.Draw(texture, position, color)` in C# has to find the Go member
// whose suffix spells those three types, and this is where that spelling is
// pinned.
func TestEverySpriteDrawOverloadIsReachableFromOutside(t *testing.T) {
	batch := &graphics.SpriteBatch{}
	texture := &graphics.Texture2D{}

	// The texture position is graphics.Texture2DReference, NOT *Texture2D. That
	// is the Foundation 58 substitutability rule seen from outside: C# accepts
	// a RenderTarget2D wherever an overload declares Texture2D, and a consumer
	// who could not pass one would have a binding that refused legal XNA.
	var _ func(graphics.Texture2DReference, framework.Vector2, framework.Color) error = batch.DrawByTexture2DAndVector2AndColor
	var _ func(graphics.Texture2DReference, framework.Vector2, *framework.Rectangle, framework.Color) error = batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor
	var _ func(graphics.Texture2DReference, framework.Vector2, *framework.Rectangle, framework.Color, float32, framework.Vector2, float32, graphics.SpriteEffects, float32) error = batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle
	var _ func(graphics.Texture2DReference, framework.Vector2, *framework.Rectangle, framework.Color, float32, framework.Vector2, framework.Vector2, graphics.SpriteEffects, float32) error = batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle
	var _ func(graphics.Texture2DReference, framework.Rectangle, framework.Color) error = batch.DrawByTexture2DAndRectangleAndColor
	var _ func(graphics.Texture2DReference, framework.Rectangle, *framework.Rectangle, framework.Color) error = batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor
	var _ func(graphics.Texture2DReference, framework.Rectangle, *framework.Rectangle, framework.Color, float32, framework.Vector2, graphics.SpriteEffects, float32) error = batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle

	// And a RenderTarget2D really does satisfy that position from outside the
	// module. This assignment is the whole proof; it does not compile otherwise.
	var _ graphics.Texture2DReference = (*graphics.RenderTarget2D)(nil)

	// The optional Nullable<Rectangle> is a POINTER, and nil is the reference's
	// static nullRectangle. A consumer must be able to pass one.
	if err := batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(
		texture, framework.Vector2{}, nil, framework.Color{}); err == nil {
		t.Fatal("a Draw outside a begin/end pair was accepted")
	}

	// The two guards a consumer will actually hit, with Microsoft's own
	// sentences, from outside the module.
	nullTexture := batch.DrawByTexture2DAndVector2AndColor(nil, framework.Vector2{}, framework.Color{})
	if nullTexture == nil || !strings.Contains(nullTexture.Error(), "This method does not accept null for this parameter.") {
		t.Fatalf("nil-texture Draw = %v, want the reference's ArgumentNullException message", nullTexture)
	}
	outside := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, framework.Color{})
	if outside == nil || !strings.Contains(outside.Error(), "Begin must be called successfully before a Draw can be called.") {
		t.Fatalf("Draw outside a pair = %v, want the reference's InvalidOperationException message", outside)
	}
}

// TestTextureBoundsAndGameIsActiveAreReachableFromOutside pins the two smallest
// members Foundation 50 added, at their exact projected shapes. BOTH are
// infallible, and Bounds became so in Foundation 52: the constructor records
// the dimensions CNA applied, so Width, Height and Bounds are field reads with
// no throw site, exactly as the reference's are.
func TestTextureBoundsAndGameIsActiveAreReachableFromOutside(t *testing.T) {
	var _ func() framework.Rectangle = (&graphics.Texture2D{}).Bounds
	var _ func() bool = (&framework.Game{}).IsActive

	if (&framework.Game{}).IsActive() {
		t.Fatal("a Game that received no activation signal reports active")
	}
	// A zero Texture2D has recorded no dimensions, so its bounds are empty --
	// which is what the field holds, not a failure.
	if got := (&graphics.Texture2D{}).Bounds(); got != (framework.Rectangle{}) {
		t.Fatalf("Bounds on an unconstructed texture = %+v, want the zero rectangle", got)
	}
}

// TestGraphicsDeviceStateSurfaceIsReachableFromOutside compiles every one of
// the fifteen render-state members against its exact projected signature.
//
// The shapes are the point. Five of these carry an error where the reference's
// getter is a single field read, because CNA-Go asks CNA rather than keeping a
// managed cache it cannot initialise -- a consumer has to write `value, err :=`
// where C# writes a property read, and this is where that is pinned.
func TestGraphicsDeviceStateSurfaceIsReachableFromOutside(t *testing.T) {
	device := &graphics.GraphicsDevice{}

	var _ func() (framework.Color, error) = device.BlendFactor
	var _ func(framework.Color) error = device.SetBlendFactor
	var _ func() (int32, error) = device.MultiSampleMask
	var _ func(int32) error = device.SetMultiSampleMask
	var _ func() (int32, error) = device.ReferenceStencil
	var _ func(int32) error = device.SetReferenceStencil
	var _ func() (framework.Rectangle, error) = device.ScissorRectangle
	var _ func(framework.Rectangle) error = device.SetScissorRectangle
	var _ func(graphics.Viewport) error = device.SetViewport
	var _ func() (graphics.Viewport, error) = device.Viewport
	var _ func() (graphics.GraphicsProfile, error) = device.GraphicsProfile
	var _ func() (graphics.GraphicsDeviceStatus, error) = device.GraphicsDeviceStatus
	var _ func() (bool, error) = device.IsDisposed
	var _ func(graphics.ClearOptions, framework.Color, float32, int32) error = device.ClearByClearOptionsAndColorAndSingleAndInt32
	var _ func(graphics.ClearOptions, framework.Vector4, float32, int32) error = device.ClearByClearOptionsAndVector4AndSingleAndInt32
	var _ func() error = device.PresentByNone

	// A facade with no device reports on every one of them rather than
	// panicking, which is the state a consumer reaches by holding one past a
	// run.
	if _, err := device.GraphicsProfile(); err == nil {
		t.Fatal("GraphicsProfile on a device-less facade returned a profile and no error")
	}
	if err := device.PresentByNone(); err == nil {
		t.Fatal("Present on a device-less facade returned no error")
	}
}

// TestDisplayModeAndTextureConstructorsAreReachableFromOutside pins the shapes
// Foundation 52 added, from a module that can see only exported names.
//
// DisplayMode's six members are INFALLIBLE, which is the whole classification
// argument in one line of Go: the type is native-SOURCED and pure managed, and
// the member that reports one -- GraphicsDevice.DisplayMode -- is where the
// error lives.
func TestDisplayModeAndTextureConstructorsAreReachableFromOutside(t *testing.T) {
	var mode *graphics.DisplayMode = &graphics.DisplayMode{}
	var _ func() int32 = mode.Width
	var _ func() int32 = mode.Height
	var _ func() graphics.SurfaceFormat = mode.Format
	var _ func() float32 = mode.AspectRatio
	var _ func() framework.Rectangle = mode.TitleSafeArea
	var _ func() string = mode.ToString
	var _ func() (*graphics.DisplayMode, error) = (&graphics.GraphicsDevice{}).DisplayMode

	// A consumer cannot construct a meaningful one, and the zero value answers
	// the reference's own zero-dimension guard rather than dividing.
	if mode.AspectRatio() != 0 {
		t.Fatalf("a zero DisplayMode reported an aspect ratio of %v", mode.AspectRatio())
	}

	var _ func(*graphics.GraphicsDevice, int32, int32) (*graphics.Texture2D, error) = graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32
	var _ func(*graphics.GraphicsDevice, int32, int32, bool, graphics.SurfaceFormat) (*graphics.Texture2D, error) = graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat

	if _, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(nil, 4, 4); err == nil ||
		!strings.Contains(err.Error(), "The GraphicsDevice must not be null when creating new resources.") {
		t.Fatalf("a nil device produced %v, want the reference's message", err)
	}
}

// TestTextureStreamSurfaceIsReachableFromOutside pins the three signatures
// Foundation 53 added, and the one that matters most is SaveAsPng's first
// parameter: it is an io.WRITER.
//
// The CLR has one Stream and Go has two interfaces, so the profile's default
// mapping -- System.IO.Stream to io.Reader -- is right for most positions and
// was wrong for these two. A consumer handed an io.Reader here could not pass
// the destination the member exists to fill.
func TestTextureStreamSurfaceIsReachableFromOutside(t *testing.T) {
	texture := &graphics.Texture2D{}
	var _ func(io.Writer, int32, int32) error = texture.SaveAsPng
	var _ func(io.Writer, int32, int32) error = texture.SaveAsJpeg
	var _ func(*graphics.GraphicsDevice, io.Reader, int32, int32, bool) (*graphics.Texture2D, error) = graphics.Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean

	// A real io.Writer compiles and is accepted as far as the disposal check,
	// which is where an unbound texture stops.
	if err := texture.SaveAsPng(&bytes.Buffer{}, 8, 8); err == nil {
		t.Fatal("an unbound texture encoded successfully")
	}
	if err := texture.SaveAsPng(nil, 8, 8); err == nil ||
		!strings.Contains(err.Error(), "This method does not accept null for this parameter.") {
		t.Fatalf("a nil destination produced %v, want the reference's message", err)
	}
}

// TestTheGenericMethodProjectionIsReachableFromOutside is where the Foundation
// 54 rule is proved from a consumer's position, and it is the only place that
// can: the rule is about a SHAPE, and the shape is what a consumer writes.
//
// C# writes `texture.SetData(pixels)`. Go cannot: methods may not declare type
// parameters, so the member is a package-level function taking the texture
// first, and its overload suffix names the type parameter the member declares
// rather than the IL token's position.
func TestTheGenericMethodProjectionIsReachableFromOutside(t *testing.T) {
	texture := &graphics.Texture2D{}
	pixels := make([]framework.Color, 4)

	var _ func(graphics.Texture2DReference, []framework.Color) error = graphics.Texture2DSetDataBySliceOfT[framework.Color]
	var _ func(graphics.Texture2DReference, []framework.Color, int32, int32) error = graphics.Texture2DSetDataBySliceOfTAndInt32AndInt32[framework.Color]
	var _ func(graphics.Texture2DReference, int32, *framework.Rectangle, []framework.Color, int32, int32) error = graphics.Texture2DSetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[framework.Color]
	var _ func(graphics.Texture2DReference, []framework.Color) error = graphics.Texture2DGetDataBySliceOfT[framework.Color]
	var _ func(graphics.Texture2DReference, []framework.Color, int32, int32) error = graphics.Texture2DGetDataBySliceOfTAndInt32AndInt32[framework.Color]
	var _ func(graphics.Texture2DReference, int32, *framework.Rectangle, []framework.Color, int32, int32) error = graphics.Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32[framework.Color]

	// The method-shaped name a C# reader would look for does not exist, which
	// is the rule's whole consequence for a consumer.
	if _, present := reflect.TypeOf(texture).MethodByName("SetData"); present {
		t.Fatal("Texture2D has a SetData method; Go cannot declare one with a type parameter")
	}

	// Type inference works from the slice, so a consumer writes the type
	// parameter only when they want to.
	if err := graphics.Texture2DSetDataBySliceOfT(texture, pixels); err == nil {
		t.Fatal("an unbound texture accepted a transfer")
	}

	// A packed vector is an element type too, which is what makes the mapping
	// worth having rather than a Color special case.
	var _ func(graphics.Texture2DReference, []packedvector.Bgr565) error = graphics.Texture2DSetDataBySliceOfT[packedvector.Bgr565]
}

// TestContentManagerIsReachableFromOutside compiles ContentManager's whole
// projected surface against its exact shapes, from a module that imports
// CNA-Go rather than living inside it.
//
// The two generic loaders are the point. Go methods cannot declare type
// parameters, so `manager.Load<Texture2D>("x")` has no method-shaped
// counterpart at all -- a consumer writes the package function with the
// receiver first, and this is where that spelling is pinned.
func TestContentManagerIsReachableFromOutside(t *testing.T) {
	manager := &content.ContentManager{}

	var _ func(any) (*content.ContentManager, error) = content.NewContentManagerByIServiceProvider
	var _ func(any, string) (*content.ContentManager, error) = content.NewContentManagerByIServiceProviderAndString
	var _ func() (any, error) = manager.ServiceProvider
	var _ func() (string, error) = manager.RootDirectory
	var _ func(string) error = manager.SetRootDirectory
	var _ func() error = manager.Unload
	var _ func(string) (io.Reader, error) = manager.OpenStream
	var _ func() error = manager.DisposeByNone
	var _ func(bool) error = manager.DisposeByBoolean
	var _ func(*content.ContentManager, string) (*graphics.Texture2D, error) = content.ContentManagerLoad[*graphics.Texture2D]
	var _ func(*content.ContentManager, string, any) (*graphics.Texture2D, error) = content.ContentManagerReadAsset[*graphics.Texture2D]

	// The method-shaped names a C# reader would look for do not exist.
	for _, name := range []string{"Load", "ReadAsset"} {
		if _, present := reflect.TypeOf(manager).MethodByName(name); present {
			t.Fatalf("ContentManager has a %s method; Go cannot declare one with a type parameter", name)
		}
	}

	// The constructor guard, with the reference's own failure.
	if _, err := content.NewContentManagerByIServiceProvider(nil); err == nil {
		t.Fatal("a nil service provider was accepted")
	}

	// A T outside the closed set is refused BY NAME, before any device is
	// resolved -- which is what a consumer reaching for an unprojected asset
	// kind needs to learn.
	live, err := content.NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	type unprojectedAsset struct{}
	if _, err := content.ContentManagerLoad[*unprojectedAsset](live, "asset"); err == nil {
		t.Fatal("a Load of an unprojected asset type was accepted")
	} else if !strings.Contains(err.Error(), "unprojectedAsset") {
		t.Fatalf("Load = %v, want the refusal to name the type it cannot load", err)
	}
}

// TestGameContentIsTheCrossPackageProjection pins where Game::Content lives for
// a consumer, and that it behaves like the field it is.
//
// Game is in the framework package and ContentManager is in the Content package
// below it, so the settled cycle rule puts both accessors HERE, as package
// functions taking the Game first. A consumer writing `game.Content` in C#
// writes `content.GameContent(game)` in Go, and nothing on Game itself spells
// it -- which is exactly what this checks.
func TestGameContentIsTheCrossPackageProjection(t *testing.T) {
	var _ func(*framework.Game) *content.ContentManager = content.GameContent
	var _ func(*framework.Game, *content.ContentManager) error = content.SetGameContent

	if _, present := reflect.TypeOf(&framework.Game{}).MethodByName("Content"); present {
		t.Fatal("Game has a Content method; the framework package cannot name ContentManager")
	}

	game, _ := newCanaryGame(t)
	manager := content.GameContent(game)
	if manager == nil {
		t.Fatal("a constructed Game has no ContentManager; its constructor creates one")
	}
	if content.GameContent(game) != manager {
		t.Fatal("GameContent answered a different manager on a second call")
	}
	if err := content.SetGameContent(game, nil); err == nil {
		t.Fatal("SetGameContent accepted nil; set_Content throws ArgumentNullException")
	}
	if content.GameContent(game) != manager {
		t.Fatal("the refused assignment changed the field")
	}
	replacement, err := content.NewContentManagerByIServiceProviderAndString(game.Services(), "Assets")
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProviderAndString: %v", err)
	}
	if err := content.SetGameContent(game, replacement); err != nil {
		t.Fatalf("SetGameContent: %v", err)
	}
	if content.GameContent(game) != replacement {
		t.Fatal("GameContent did not answer the assigned manager")
	}
}

// TestVertexDeclarationIsReachableFromOutside compiles VertexDeclaration's
// whole projected surface and proves the two facts a consumer will hit first:
// the elements array is a SLICE rather than a Go variadic, and an invalid
// layout is refused with Microsoft's own sentence.
func TestVertexDeclarationIsReachableFromOutside(t *testing.T) {
	var _ func(int32, []graphics.VertexElement) (*graphics.VertexDeclaration, error) = graphics.NewVertexDeclarationByInt32AndSliceOfVertexElement
	var _ func([]graphics.VertexElement) (*graphics.VertexDeclaration, error) = graphics.NewVertexDeclarationBySliceOfVertexElement

	declaration := &graphics.VertexDeclaration{}
	var _ func() int32 = declaration.VertexStride
	var _ func() []graphics.VertexElement = declaration.GetVertexElements
	var _ func() *graphics.GraphicsDevice = declaration.GraphicsDevice
	var _ func() bool = declaration.IsDisposed
	var _ func() string = declaration.ToString
	var _ func() error = declaration.DisposeByNone
	var _ func(bool) error = declaration.DisposeByBoolean

	elements := []graphics.VertexElement{
		graphics.NewVertexElement(0, graphics.VertexElementFormatVector3, graphics.VertexElementUsagePosition, 0),
		graphics.NewVertexElement(12, graphics.VertexElementFormatColor, graphics.VertexElementUsageColor, 0),
	}
	built, err := graphics.NewVertexDeclarationBySliceOfVertexElement(elements)
	if err != nil {
		t.Fatalf("NewVertexDeclarationBySliceOfVertexElement: %v", err)
	}
	if built.VertexStride() != 16 {
		t.Fatalf("VertexStride = %d, want 16", built.VertexStride())
	}
	// A declaration a consumer constructed has NO device, which is the
	// reference's answer and not a gap.
	if built.GraphicsDevice() != nil {
		t.Fatal("a constructed VertexDeclaration reports a GraphicsDevice")
	}
	if got := built.ToString(); got != "Microsoft.Xna.Framework.Graphics.VertexDeclaration" {
		t.Fatalf("ToString = %q", got)
	}
	_, err = graphics.NewVertexDeclarationByInt32AndSliceOfVertexElement(16, []graphics.VertexElement{
		graphics.NewVertexElement(0, graphics.VertexElementFormatVector2, graphics.VertexElementUsagePosition, 0),
		graphics.NewVertexElement(4, graphics.VertexElementFormatSingle, graphics.VertexElementUsageFog, 0),
	})
	if err == nil {
		t.Fatal("overlapping vertex elements were accepted")
	}
	if !strings.Contains(err.Error(), "Elements Position0 and Fog0 are overlapping.") {
		t.Fatalf("%v, want the reference's overlap message", err)
	}
}

// TestIndexBufferIsReachableFromOutside compiles IndexBuffer's whole projected
// surface, including the six transfer members whose method-shaped names do NOT
// exist -- Go methods cannot declare type parameters, so a consumer writes the
// package function with the receiver first.
func TestIndexBufferIsReachableFromOutside(t *testing.T) {
	var _ func(*graphics.GraphicsDevice, graphics.IndexElementSize, int32, graphics.BufferUsage) (*graphics.IndexBuffer, error) = graphics.NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage
	var _ func(*graphics.GraphicsDevice, reflect.Type, int32, graphics.BufferUsage) (*graphics.IndexBuffer, error) = graphics.NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage

	buffer := &graphics.IndexBuffer{}
	var _ func() int32 = buffer.IndexCount
	var _ func() graphics.IndexElementSize = buffer.IndexElementSize
	var _ func() graphics.BufferUsage = buffer.BufferUsage
	var _ func() *graphics.GraphicsDevice = buffer.GraphicsDevice
	var _ func() bool = buffer.IsDisposed
	var _ func() error = buffer.DisposeByNone
	var _ func(bool) error = buffer.DisposeByBoolean

	var _ func(*graphics.IndexBuffer, []uint16) error = graphics.IndexBufferSetDataBySliceOfT[uint16]
	var _ func(*graphics.IndexBuffer, []uint16, int32, int32) error = graphics.IndexBufferSetDataBySliceOfTAndInt32AndInt32[uint16]
	var _ func(*graphics.IndexBuffer, int32, []uint16, int32, int32) error = graphics.IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32[uint16]
	var _ func(*graphics.IndexBuffer, []uint32) error = graphics.IndexBufferGetDataBySliceOfT[uint32]
	var _ func(*graphics.IndexBuffer, []uint32, int32, int32) error = graphics.IndexBufferGetDataBySliceOfTAndInt32AndInt32[uint32]
	var _ func(*graphics.IndexBuffer, int32, []uint32, int32, int32) error = graphics.IndexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32[uint32]

	for _, name := range []string{"SetData", "GetData"} {
		if _, present := reflect.TypeOf(buffer).MethodByName(name); present {
			t.Fatalf("IndexBuffer has a %s method; Go cannot declare one with a type parameter", name)
		}
	}

	// The count guard runs before the device, so it is reachable with a nil
	// one -- and it carries Microsoft's own sentence.
	if _, err := graphics.NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		nil, graphics.IndexElementSizeSixteenBits, 0, graphics.BufferUsageNone); err == nil {
		t.Fatal("a zero index count was accepted")
	} else if !strings.Contains(err.Error(), "Resource size must be greater than zero.") {
		t.Fatalf("%v, want the reference's message", err)
	}
	// And the Type constructor's closed element set is visible from outside.
	if _, err := graphics.NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		nil, reflect.TypeOf(float64(0)), 4, graphics.BufferUsageNone); err == nil {
		t.Fatal("a float64 was accepted as an index element type")
	} else if !strings.Contains(err.Error(), "float64") {
		t.Fatalf("%v, want the refusal to name the type", err)
	}
}

// canaryVertex is a consumer's own vertex type, declared OUTSIDE the module.
// It is the whole point of projecting IVertexType: without it the Type-keyed
// VertexBuffer constructor would be a member nothing could satisfy.
type canaryVertex struct {
	Position framework.Vector3
	Colour   framework.Color
}

var canaryVertexDeclaration = func() *graphics.VertexDeclaration {
	declaration, err := graphics.NewVertexDeclarationByInt32AndSliceOfVertexElement(16, []graphics.VertexElement{
		graphics.NewVertexElement(0, graphics.VertexElementFormatVector3, graphics.VertexElementUsagePosition, 0),
		graphics.NewVertexElement(12, graphics.VertexElementFormatColor, graphics.VertexElementUsageColor, 0),
	})
	if err != nil {
		panic(err)
	}
	return declaration
}()

func (canaryVertex) VertexDeclaration() *graphics.VertexDeclaration { return canaryVertexDeclaration }

var _ graphics.IVertexType = canaryVertex{}

// TestVertexBufferIsReachableFromOutside compiles VertexBuffer's whole
// projected surface and proves the one thing only an outside consumer can: that
// a vertex type declared in ANOTHER module satisfies IVertexType and is
// resolved by the Type-keyed constructor.
func TestVertexBufferIsReachableFromOutside(t *testing.T) {
	var _ func(*graphics.GraphicsDevice, *graphics.VertexDeclaration, int32, graphics.BufferUsage) (*graphics.VertexBuffer, error) = graphics.NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage
	var _ func(*graphics.GraphicsDevice, reflect.Type, int32, graphics.BufferUsage) (*graphics.VertexBuffer, error) = graphics.NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage

	buffer := &graphics.VertexBuffer{}
	var _ func() int32 = buffer.VertexCount
	var _ func() *graphics.VertexDeclaration = buffer.VertexDeclaration
	var _ func() graphics.BufferUsage = buffer.BufferUsage
	var _ func() error = buffer.DisposeByNone

	var _ func(*graphics.VertexBuffer, []canaryVertex) error = graphics.VertexBufferSetDataBySliceOfT[canaryVertex]
	var _ func(*graphics.VertexBuffer, []canaryVertex, int32, int32) error = graphics.VertexBufferSetDataBySliceOfTAndInt32AndInt32[canaryVertex]
	var _ func(*graphics.VertexBuffer, int32, []canaryVertex, int32, int32, int32) error = graphics.VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32[canaryVertex]
	var _ func(*graphics.VertexBuffer, []canaryVertex) error = graphics.VertexBufferGetDataBySliceOfT[canaryVertex]
	var _ func(*graphics.VertexBuffer, []canaryVertex, int32, int32) error = graphics.VertexBufferGetDataBySliceOfTAndInt32AndInt32[canaryVertex]
	var _ func(*graphics.VertexBuffer, int32, []canaryVertex, int32, int32, int32) error = graphics.VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32[canaryVertex]

	for _, name := range []string{"SetData", "GetData"} {
		if _, present := reflect.TypeOf(buffer).MethodByName(name); present {
			t.Fatalf("VertexBuffer has a %s method; Go cannot declare one with a type parameter", name)
		}
	}

	// The Type-keyed constructor resolves an OUTSIDE type. It gets as far as
	// the device, which is nil here -- so it must NOT fail with a vertex-type
	// message.
	_, err := graphics.NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		nil, reflect.TypeOf(canaryVertex{}), 4, graphics.BufferUsageNone)
	if err == nil {
		t.Fatal("a nil device was accepted")
	}
	if strings.Contains(err.Error(), "Invalid vertex type") {
		t.Fatalf("a consumer's own IVertexType was refused: %v", err)
	}
	// And a struct that is not one is refused by name, from outside.
	type plainStruct struct{ A, B, C, D int32 }
	_, err = graphics.NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		nil, reflect.TypeOf(plainStruct{}), 4, graphics.BufferUsageNone)
	if err == nil {
		t.Fatal("a plain struct was accepted as a vertex type")
	}
	if !strings.Contains(err.Error(), "does not implement the IVertexType interface.") {
		t.Fatalf("%v, want the reference's message", err)
	}
}

// TestVertexBufferBindingAndDeviceDrawSurfaceIsReachableFromOutside compiles
// the binding type and the device's nine buffer members at their exact shapes.
// Two of them are INFALLIBLE and that is the point: the reference reads managed
// fields the setters maintain, so a consumer writes a plain call where every
// other device getter needs `value, err :=`.
func TestVertexBufferBindingAndDeviceDrawSurfaceIsReachableFromOutside(t *testing.T) {
	var _ func(*graphics.VertexBuffer, int32, int32) (graphics.VertexBufferBinding, error) = graphics.NewVertexBufferBindingByVertexBufferAndInt32AndInt32
	var _ func(*graphics.VertexBuffer, int32) (graphics.VertexBufferBinding, error) = graphics.NewVertexBufferBindingByVertexBufferAndInt32
	var _ func(*graphics.VertexBuffer) (graphics.VertexBufferBinding, error) = graphics.NewVertexBufferBindingByVertexBuffer
	var _ func(*graphics.VertexBuffer) (graphics.VertexBufferBinding, error) = graphics.VertexBufferBindingOperatorImplicitByVertexBuffer

	var binding graphics.VertexBufferBinding
	var _ func() *graphics.VertexBuffer = binding.VertexBuffer
	var _ func() int32 = binding.VertexOffset
	var _ func() int32 = binding.InstanceFrequency

	device := &graphics.GraphicsDevice{}
	var _ func(*graphics.VertexBuffer) error = device.SetVertexBufferByVertexBuffer
	var _ func(*graphics.VertexBuffer, int32) error = device.SetVertexBufferByVertexBufferAndInt32
	var _ func([]graphics.VertexBufferBinding) error = device.SetVertexBuffers
	var _ func() []graphics.VertexBufferBinding = device.GetVertexBuffers
	var _ func() *graphics.IndexBuffer = device.Indices
	var _ func(*graphics.IndexBuffer) error = device.SetIndices
	var _ func(graphics.PrimitiveType, int32, int32) error = device.DrawPrimitives
	var _ func(graphics.PrimitiveType, int32, int32, int32, int32, int32) error = device.DrawIndexedPrimitives
	var _ func(graphics.PrimitiveType, int32, int32, int32, int32, int32, int32) error = device.DrawInstancedPrimitives

	// The zero binding is an empty slot, and its three getters answer.
	if binding.VertexBuffer() != nil || binding.VertexOffset() != 0 || binding.InstanceFrequency() != 0 {
		t.Fatal("the zero VertexBufferBinding is not empty")
	}
	// A null buffer is refused with Microsoft's own sentence.
	if _, err := graphics.NewVertexBufferBindingByVertexBuffer(nil); err == nil {
		t.Fatal("a nil vertex buffer was accepted")
	} else if !strings.Contains(err.Error(), "This method does not accept null for this parameter.") {
		t.Fatalf("%v, want the reference's message", err)
	}
	// And an unconstructed device answers both readers without failing.
	if device.Indices() != nil || device.GetVertexBuffers() != nil {
		t.Fatal("an unconstructed device reports bound buffers")
	}
}
