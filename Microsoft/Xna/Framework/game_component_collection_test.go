package framework

import (
	"errors"
	"fmt"
	"testing"
)

// component is a minimal IGameComponent whose only purpose is to be a distinct
// reference identity. CNA-Go ships no GameComponent, so the tests supply their
// own conformer exactly as an external consumer would.
type component struct{ name string }

func (c *component) Initialize() error { return nil }

// collectionRecorder captures what each raise handed a handler, so ordering,
// sender identity and args identity are asserted exactly rather than by count.
type collectionRecorder struct {
	events     []string
	senders    []any
	args       []*GameComponentCollectionEventArgs
	components []IGameComponent
}

func (r *collectionRecorder) handler(name string) EventHandler[*GameComponentCollectionEventArgs] {
	return func(sender any, args *GameComponentCollectionEventArgs) error {
		r.events = append(r.events, name)
		r.senders = append(r.senders, sender)
		r.args = append(r.args, args)
		r.components = append(r.components, args.GameComponent())
		return nil
	}
}

func (r *collectionRecorder) names() []string { return r.events }

func componentNames(items []IGameComponent) []string {
	names := make([]string, len(items))
	for i, item := range items {
		if item == nil {
			names[i] = "<nil>"
			continue
		}
		names[i] = item.(*component).name
	}
	return names
}

func drain(t *testing.T, iterator Iterator[IGameComponent]) []IGameComponent {
	t.Helper()
	var seen []IGameComponent
	for {
		value, ok, err := iterator.Next()
		if err != nil {
			t.Fatalf("unexpected enumeration error: %v", err)
		}
		if !ok {
			return seen
		}
		seen = append(seen, value)
	}
}

func mustAdd(t *testing.T, c *GameComponentCollection, items ...IGameComponent) {
	t.Helper()
	for _, item := range items {
		if err := c.Add(item); err != nil {
			t.Fatalf("Add(%v): %v", item, err)
		}
	}
}

func equalNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestNewGameComponentCollectionIsEmpty(t *testing.T) {
	c := NewGameComponentCollection()
	if c.Count() != 0 {
		t.Fatalf("Count = %d, want 0", c.Count())
	}
	// The base constructor assigns a fresh List<IGameComponent>, so the store
	// is mutable from the start and Add cannot report an unsupported store.
	if err := c.Add(&component{name: "a"}); err != nil {
		t.Fatalf("Add on a fresh collection: %v", err)
	}
}

func TestAddAppendsInOrderAndRaisesComponentAdded(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentAddedHandler(recorder.handler("added")); err != nil {
		t.Fatal(err)
	}
	first, second := &component{name: "first"}, &component{name: "second"}
	mustAdd(t, c, first, second)

	if c.Count() != 2 {
		t.Fatalf("Count = %d, want 2", c.Count())
	}
	if got := componentNames(drain(t, c.GetEnumerator())); !equalNames(got, []string{"first", "second"}) {
		t.Fatalf("enumeration = %v, want [first second]", got)
	}
	if !equalNames(recorder.names(), []string{"added", "added"}) {
		t.Fatalf("events = %v", recorder.names())
	}
	// The sender is the collection itself: OnComponentAdded pushes ldarg.0.
	for i, sender := range recorder.senders {
		if sender != any(c) {
			t.Fatalf("sender[%d] = %v, want the collection", i, sender)
		}
	}
	if recorder.components[0] != IGameComponent(first) || recorder.components[1] != IGameComponent(second) {
		t.Fatalf("components = %v", componentNames(recorder.components))
	}
	// Each raise allocates a fresh GameComponentCollectionEventArgs, unlike
	// the argument-free XNA events that share EventArgs.Empty.
	if recorder.args[0] == recorder.args[1] {
		t.Fatal("two raises shared one event args instance")
	}
}

func TestAddRejectsDuplicateWithoutMutatingOrRaising(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentAddedHandler(recorder.handler("added")); err != nil {
		t.Fatal(err)
	}
	only := &component{name: "only"}
	mustAdd(t, c, only)

	err := c.Add(only)
	if err == nil {
		t.Fatal("adding the same component twice must fail")
	}
	if !errors.Is(err, errCollectionArgument) {
		t.Fatalf("duplicate must project ArgumentException, got %v", err)
	}
	if c.Count() != 1 {
		t.Fatalf("a rejected duplicate must not mutate, Count = %d", c.Count())
	}
	if !equalNames(recorder.names(), []string{"added"}) {
		t.Fatalf("a rejected duplicate must not raise, events = %v", recorder.names())
	}
}

func TestNilComponentIsStoredButAnnouncesNothingAndStillDuplicates(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentAddedHandler(recorder.handler("added")); err != nil {
		t.Fatal(err)
	}
	// InsertItem inserts unconditionally and only then tests `item != null`,
	// so a nil component occupies a slot while raising nothing.
	if err := c.Add(nil); err != nil {
		t.Fatalf("Add(nil): %v", err)
	}
	if c.Count() != 1 {
		t.Fatalf("Count = %d, want 1", c.Count())
	}
	if len(recorder.names()) != 0 {
		t.Fatalf("a nil component must raise nothing, events = %v", recorder.names())
	}
	// IndexOf(nil) now finds the stored nil, so the duplicate test rejects a
	// second one.
	if err := c.Add(nil); !errors.Is(err, errCollectionArgument) {
		t.Fatalf("a second nil must be rejected as a duplicate, got %v", err)
	}
}

func TestRemoveMutatesBeforeItAnnounces(t *testing.T) {
	c := NewGameComponentCollection()
	first, second := &component{name: "first"}, &component{name: "second"}
	mustAdd(t, c, first, second)

	var observed int32 = -1
	if _, err := c.AddComponentRemovedHandler(func(sender any, args *GameComponentCollectionEventArgs) error {
		observed = sender.(*GameComponentCollection).Count()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	removed, err := c.Remove(first)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("Remove must report true for a present component")
	}
	// RemoveItem calls base.RemoveItem BEFORE OnComponentRemoved, so a
	// handler sees the collection already shortened.
	if observed != 1 {
		t.Fatalf("handler observed Count = %d, want 1", observed)
	}
	if got := componentNames(drain(t, c.GetEnumerator())); !equalNames(got, []string{"second"}) {
		t.Fatalf("remaining = %v, want [second]", got)
	}
}

func TestRemoveOfAbsentComponentChangesNothing(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentRemovedHandler(recorder.handler("removed")); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, c, &component{name: "present"})

	removed, err := c.Remove(&component{name: "absent"})
	if err != nil {
		t.Fatalf("removing an absent component must not fail: %v", err)
	}
	if removed {
		t.Fatal("Remove must report false for an absent component")
	}
	if c.Count() != 1 || len(recorder.names()) != 0 {
		t.Fatalf("Count = %d, events = %v", c.Count(), recorder.names())
	}
}

func TestClearAnnouncesEveryElementBeforeMutating(t *testing.T) {
	c := NewGameComponentCollection()
	first, second, third := &component{name: "first"}, &component{name: "second"}, &component{name: "third"}
	mustAdd(t, c, first, second, third)

	var counts []int32
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentRemovedHandler(func(sender any, args *GameComponentCollectionEventArgs) error {
		counts = append(counts, sender.(*GameComponentCollection).Count())
		return recorder.handler("removed")(sender, args)
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	// ClearItems raises for every element, index 0 upward, and only then
	// calls base.ClearItems, so every handler saw the full collection.
	if !equalNames(componentNames(recorder.components), []string{"first", "second", "third"}) {
		t.Fatalf("announced %v, want [first second third]", componentNames(recorder.components))
	}
	for i, count := range counts {
		if count != 3 {
			t.Fatalf("handler %d observed Count = %d, want 3: Clear announces before it mutates", i, count)
		}
	}
	if c.Count() != 0 {
		t.Fatalf("Count after Clear = %d, want 0", c.Count())
	}
}

func TestClearAnnouncesNilElementsUnlikeRemove(t *testing.T) {
	c := NewGameComponentCollection()
	if err := c.Add(nil); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, c, &component{name: "real"})

	recorder := &collectionRecorder{}
	if _, err := c.AddComponentRemovedHandler(recorder.handler("removed")); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	// ClearItems has NO null check, unlike RemoveItem: it raises for the nil
	// element too, with an event args whose GameComponent is nil.
	if !equalNames(componentNames(recorder.components), []string{"<nil>", "real"}) {
		t.Fatalf("announced %v, want [<nil> real]", componentNames(recorder.components))
	}
	if recorder.args[0].GameComponent() != nil {
		t.Fatal("the nil element must be announced with a nil GameComponent")
	}
}

func TestRemoveDoesNotAnnounceANilElement(t *testing.T) {
	c := NewGameComponentCollection()
	if err := c.Add(nil); err != nil {
		t.Fatal(err)
	}
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentRemovedHandler(recorder.handler("removed")); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveAt(0); err != nil {
		t.Fatal(err)
	}
	if len(recorder.names()) != 0 {
		t.Fatalf("RemoveItem must not announce a nil element, events = %v", recorder.names())
	}
	if c.Count() != 0 {
		t.Fatalf("Count = %d, want 0", c.Count())
	}
}

func TestClearOnEmptyCollectionStillInvalidatesEnumerators(t *testing.T) {
	c := NewGameComponentCollection()
	iterator := c.GetEnumerator()
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	// List<T>.Clear increments _version unconditionally: its IL skips only
	// the Array.Clear and the `_size = 0` store when the list is empty.
	if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("Clear of an empty collection must still invalidate, got %v", err)
	}
}

func TestSetItemAlwaysFails(t *testing.T) {
	c := NewGameComponentCollection()
	only := &component{name: "only"}
	mustAdd(t, c, only)

	// In range: SetItem throws NotSupportedException having read neither of
	// its arguments.
	err := c.SetItemProperty(0, &component{name: "replacement"})
	if !errors.Is(err, errCollectionNotSupported) {
		t.Fatalf("in-range assignment must project NotSupportedException, got %v", err)
	}
	// Out of range: set_Item validates the index FIRST, so the range failure
	// is reported instead of the unsupported one.
	err = c.SetItemProperty(5, &component{name: "replacement"})
	if !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("out-of-range assignment must report the range failure first, got %v", err)
	}
	if errors.Is(err, errCollectionNotSupported) {
		t.Fatal("an out-of-range assignment must not reach SetItem at all")
	}
	// The declared protected override reads neither argument and has no
	// success path at all.
	if err := c.SetItemMethod(-99, nil); !errors.Is(err, errCollectionNotSupported) {
		t.Fatalf("SetItemMethod must always fail, got %v", err)
	}
	// Nothing was mutated by any of it.
	got, err := c.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if got != IGameComponent(only) {
		t.Fatal("a failed assignment must not mutate")
	}
}

func TestIndexBoundsMatchTheReferenceGuards(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"}, &component{name: "b"})

	for _, index := range []int32{-1, 2, 3} {
		if _, err := c.Item(index); !errors.Is(err, errCollectionArgumentOutOfRange) {
			t.Fatalf("Item(%d) must fail, got %v", index, err)
		}
		if err := c.RemoveAt(index); !errors.Is(err, errCollectionArgumentOutOfRange) {
			t.Fatalf("RemoveAt(%d) must fail, got %v", index, err)
		}
	}
	// Insert's guard is `index > Count`, which admits Count itself, unlike
	// the indexer setter's and RemoveAt's `index >= Count`.
	if err := c.Insert(2, &component{name: "appended"}); err != nil {
		t.Fatalf("Insert at Count must be legal: %v", err)
	}
	if err := c.Insert(4, &component{name: "past"}); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("Insert past Count must fail, got %v", err)
	}
	if err := c.Insert(-1, &component{name: "negative"}); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("Insert at a negative index must fail, got %v", err)
	}
}

func TestInsertValidatesRangeBeforeTheDuplicateTest(t *testing.T) {
	c := NewGameComponentCollection()
	only := &component{name: "only"}
	mustAdd(t, c, only)

	// Collection<T>.Insert validates the index and only then reaches
	// InsertItem, so an out-of-range insert of a duplicate reports the range
	// failure, not the duplicate one.
	err := c.Insert(9, only)
	if !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("range failure must win, got %v", err)
	}
	if errors.Is(err, errCollectionArgument) {
		t.Fatal("the duplicate test must not run for an out-of-range index")
	}
}

func TestInsertPlacesAtTheRequestedIndex(t *testing.T) {
	c := NewGameComponentCollection()
	first, third := &component{name: "first"}, &component{name: "third"}
	mustAdd(t, c, first, third)
	if err := c.Insert(1, &component{name: "second"}); err != nil {
		t.Fatal(err)
	}
	if got := componentNames(drain(t, c.GetEnumerator())); !equalNames(got, []string{"first", "second", "third"}) {
		t.Fatalf("order = %v", got)
	}
}

func TestIndexOfAndContainsUseReferenceIdentity(t *testing.T) {
	c := NewGameComponentCollection()
	first, second := &component{name: "same"}, &component{name: "same"}
	mustAdd(t, c, first)

	if c.IndexOf(first) != 0 {
		t.Fatalf("IndexOf(first) = %d, want 0", c.IndexOf(first))
	}
	// Two distinct instances that merely look alike are different CLR
	// references, and EqualityComparer<IGameComponent>.Default reduces to
	// Object.Equals, which no XNA component overrides.
	if c.IndexOf(second) != -1 {
		t.Fatalf("IndexOf(second) = %d, want -1", c.IndexOf(second))
	}
	if !c.Contains(first) || c.Contains(second) {
		t.Fatal("Contains must follow the same identity rule as IndexOf")
	}
	if c.IndexOf(nil) != -1 {
		t.Fatal("IndexOf(nil) on a collection with no nil element must be -1")
	}
}

func TestCopyToReportsArrayCopyFailuresInOrder(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"}, &component{name: "b"})

	if err := c.CopyTo(nil, 0); !errors.Is(err, errCollectionArgumentNull) {
		t.Fatalf("nil destination must project ArgumentNullException, got %v", err)
	}
	destination := make([]IGameComponent, 3)
	if err := c.CopyTo(destination, -1); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("negative index must project ArgumentOutOfRangeException, got %v", err)
	}
	if err := c.CopyTo(destination, 2); !errors.Is(err, errCollectionArgument) {
		t.Fatalf("a destination too small must project ArgumentException, got %v", err)
	}
	if err := c.CopyTo(destination, 1); err != nil {
		t.Fatal(err)
	}
	if got := componentNames(destination); !equalNames(got, []string{"<nil>", "a", "b"}) {
		t.Fatalf("copied %v", got)
	}
}

func TestEnumerationFailsFastAfterEveryMutation(t *testing.T) {
	mutations := map[string]func(*GameComponentCollection) error{
		"Add":      func(c *GameComponentCollection) error { return c.Add(&component{name: "new"}) },
		"Insert":   func(c *GameComponentCollection) error { return c.Insert(0, &component{name: "new"}) },
		"RemoveAt": func(c *GameComponentCollection) error { return c.RemoveAt(0) },
		"Remove": func(c *GameComponentCollection) error {
			item, err := c.Item(0)
			if err != nil {
				return err
			}
			_, err = c.Remove(item)
			return err
		},
		"Clear": func(c *GameComponentCollection) error { return c.Clear() },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			c := NewGameComponentCollection()
			mustAdd(t, c, &component{name: "a"}, &component{name: "b"})
			iterator := c.GetEnumerator()
			if _, _, err := iterator.Next(); err != nil {
				t.Fatal(err)
			}
			if err := mutate(c); err != nil {
				t.Fatal(err)
			}
			if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
				t.Fatalf("%s must invalidate a live enumerator, got %v", name, err)
			}
		})
	}
}

func TestFailedAssignmentDoesNotInvalidateEnumerators(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"})
	iterator := c.GetEnumerator()
	if err := c.SetItemProperty(0, &component{name: "b"}); err == nil {
		t.Fatal("assignment must fail")
	}
	// SetItem throws before any store, so the version never moves.
	if got := componentNames(drain(t, iterator)); !equalNames(got, []string{"a"}) {
		t.Fatalf("enumeration = %v, want [a]", got)
	}
}

func TestEnumerationDetectsMutationEvenAfterItIsExhausted(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"})
	iterator := c.GetEnumerator()
	if got := componentNames(drain(t, iterator)); !equalNames(got, []string{"a"}) {
		t.Fatalf("enumeration = %v", got)
	}
	// A finished enumerator keeps returning false while nothing changes,
	// because MoveNext re-enters MoveNextRare and re-checks the version.
	if _, ok, err := iterator.Next(); ok || err != nil {
		t.Fatalf("a finished enumerator must stay finished, got ok=%v err=%v", ok, err)
	}
	if err := c.Add(&component{name: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("the version check runs before the bounds test, got %v", err)
	}
}

func TestRepeatedEnumerationIsIndependent(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"}, &component{name: "b"})
	first, second := c.GetEnumerator(), c.GetEnumerator()
	if got := componentNames(drain(t, first)); !equalNames(got, []string{"a", "b"}) {
		t.Fatalf("first = %v", got)
	}
	if got := componentNames(drain(t, second)); !equalNames(got, []string{"a", "b"}) {
		t.Fatalf("second = %v", got)
	}
}

func TestHandlerFailureLeavesAnAppliedMutationApplied(t *testing.T) {
	c := NewGameComponentCollection()
	boom := errors.New("handler failed")
	if _, err := c.AddComponentAddedHandler(func(any, *GameComponentCollectionEventArgs) error { return boom }); err != nil {
		t.Fatal(err)
	}
	err := c.Add(&component{name: "a"})
	if !errors.Is(err, boom) {
		t.Fatalf("the handler failure must propagate, got %v", err)
	}
	// InsertItem mutated before it announced, so the component is present
	// even though the announcement failed.
	if c.Count() != 1 {
		t.Fatalf("Count = %d, want 1: Insert mutates before it announces", c.Count())
	}
}

func TestHandlerFailureLeavesClearUnapplied(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"}, &component{name: "b"})
	boom := errors.New("handler failed")
	if _, err := c.AddComponentRemovedHandler(func(any, *GameComponentCollectionEventArgs) error { return boom }); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); !errors.Is(err, boom) {
		t.Fatalf("the handler failure must propagate, got %v", err)
	}
	// ClearItems announces the whole collection first, so a failure during
	// the announcement leaves it entirely intact.
	if c.Count() != 2 {
		t.Fatalf("Count = %d, want 2: Clear announces before it mutates", c.Count())
	}
}

func TestClearRereadsCountSoAHandlerCanExtendTheLoop(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"})
	recorder := &collectionRecorder{}
	added := false
	if _, err := c.AddComponentRemovedHandler(func(sender any, args *GameComponentCollectionEventArgs) error {
		if !added {
			added = true
			if err := sender.(*GameComponentCollection).Add(&component{name: "late"}); err != nil {
				return err
			}
		}
		return recorder.handler("removed")(sender, args)
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	// ClearItems re-reads base.Count on every iteration, so the component the
	// first handler added is announced too.
	if !equalNames(componentNames(recorder.components), []string{"a", "late"}) {
		t.Fatalf("announced %v, want [a late]", componentNames(recorder.components))
	}
	if c.Count() != 0 {
		t.Fatalf("Count = %d, want 0", c.Count())
	}
}

func TestEventAccessorsFollowTheSettledProjection(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	// A nil handler registers nothing and returns the zero token.
	token, err := c.AddComponentAddedHandler(nil)
	if err != nil {
		t.Fatal(err)
	}
	if (token != EventSubscription{}) {
		t.Fatal("a nil handler must return the zero token")
	}
	// Removing a zero, foreign, or already-removed token is harmless.
	if err := c.RemoveComponentAddedHandler(token); err != nil {
		t.Fatal(err)
	}
	other, err := NewGameComponentCollection().AddComponentAddedHandler(recorder.handler("foreign"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveComponentAddedHandler(other); err != nil {
		t.Fatal(err)
	}

	first, err := c.AddComponentAddedHandler(recorder.handler("one"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddComponentAddedHandler(recorder.handler("two")); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, c, &component{name: "a"})
	if !equalNames(recorder.names(), []string{"one", "two"}) {
		t.Fatalf("dispatch order = %v, want [one two]", recorder.names())
	}
	if err := c.RemoveComponentAddedHandler(first); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, c, &component{name: "b"})
	if !equalNames(recorder.names(), []string{"one", "two", "two"}) {
		t.Fatalf("after removal = %v", recorder.names())
	}
}

func TestComponentRemovedIsSeparateFromComponentAdded(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentAddedHandler(recorder.handler("added")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddComponentRemovedHandler(recorder.handler("removed")); err != nil {
		t.Fatal(err)
	}
	item := &component{name: "a"}
	mustAdd(t, c, item)
	if _, err := c.Remove(item); err != nil {
		t.Fatal(err)
	}
	if !equalNames(recorder.names(), []string{"added", "removed"}) {
		t.Fatalf("events = %v, want [added removed]", recorder.names())
	}
}

func TestDeclaredOverridesAreTheSameCodeTheCollectionDispatchesTo(t *testing.T) {
	c := NewGameComponentCollection()
	recorder := &collectionRecorder{}
	if _, err := c.AddComponentAddedHandler(recorder.handler("added")); err != nil {
		t.Fatal(err)
	}
	// Calling the projected protected override directly behaves exactly as
	// reaching it through Add, including the duplicate test and the raise.
	item := &component{name: "direct"}
	if err := c.InsertItem(0, item); err != nil {
		t.Fatal(err)
	}
	if c.Count() != 1 || !equalNames(recorder.names(), []string{"added"}) {
		t.Fatalf("Count = %d, events = %v", c.Count(), recorder.names())
	}
	if err := c.InsertItem(0, item); !errors.Is(err, errCollectionArgument) {
		t.Fatalf("the duplicate test must run here too, got %v", err)
	}
	if err := c.RemoveItem(0); err != nil {
		t.Fatal(err)
	}
	if c.Count() != 0 {
		t.Fatalf("Count = %d, want 0", c.Count())
	}
	if err := c.ClearItems(); err != nil {
		t.Fatal(err)
	}
}

func TestBackingStoreIsNotReachable(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"})
	// CopyTo hands out a copy: mutating it must not reach the collection.
	destination := make([]IGameComponent, 1)
	if err := c.CopyTo(destination, 0); err != nil {
		t.Fatal(err)
	}
	destination[0] = &component{name: "hijacked"}
	got, err := c.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if got.(*component).name != "a" {
		t.Fatal("CopyTo must not alias the backing store")
	}
}

func TestCollectionKeepsReferenceSemantics(t *testing.T) {
	first := NewGameComponentCollection()
	second := first
	mustAdd(t, second, &component{name: "a"})
	if first.Count() != 1 {
		t.Fatal("two variables naming one collection must observe one state")
	}
}

func TestRemovedElementIsNotRetained(t *testing.T) {
	c := NewGameComponentCollection()
	mustAdd(t, c, &component{name: "a"}, &component{name: "b"})
	if err := c.RemoveAt(0); err != nil {
		t.Fatal(err)
	}
	// List<T>.RemoveAt clears the vacated slot. Growing the collection again
	// must not resurrect the removed element.
	mustAdd(t, c, &component{name: "c"})
	if got := componentNames(drain(t, c.GetEnumerator())); !equalNames(got, []string{"b", "c"}) {
		t.Fatalf("order = %v, want [b c]", got)
	}
}

func TestReferenceIdentityEqualNeverPanics(t *testing.T) {
	// A Go interface may be satisfied by a value type, and == panics when two
	// such values share a dynamic type that is not comparable. No CLR
	// implementor can be in that state, and reporting "not the same element"
	// is better than a panic from inside a collection operation.
	left, right := uncomparableComponent{tags: []string{"a"}}, uncomparableComponent{tags: []string{"a"}}
	if referenceIdentityEqual[IGameComponent](left, right) {
		t.Fatal("two non-comparable value implementors are never one CLR reference")
	}
	if !referenceIdentityEqual[IGameComponent](nil, nil) {
		t.Fatal("two nil components are two null references")
	}
	if referenceIdentityEqual[IGameComponent](nil, &component{name: "a"}) {
		t.Fatal("nil is not any component")
	}
	c := NewGameComponentCollection()
	if err := c.Add(left); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(c.IndexOf(right)); got != "-1" {
		t.Fatalf("IndexOf = %s, want -1", got)
	}
}

type uncomparableComponent struct{ tags []string }

func (uncomparableComponent) Initialize() error { return nil }
