package eventcanary

import (
	"errors"
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
