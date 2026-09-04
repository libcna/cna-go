package framework

import (
	"errors"
	"testing"
)

// The live-list source is the ReadOnlyCollection<T> backing that projects a
// `List<T>` rather than a `T[]`, and it needs its own tests here for a
// structural reason: its one XNA consumer, ModelEffectCollection, HIDES the
// inherited GetEnumerator with its own, so the base's enumerator is unreachable
// through that family and the base's bounds checks are only reached through
// members the consumer forwards. Testing it from the adapter's own package is
// what covers the rest.

type liveListOwner struct{ items []*int }

func (o *liveListOwner) list() []*int { return o.items }

func newLiveFixture(count int) (*liveListOwner, *ReadOnlyCollection[*int]) {
	owner := &liveListOwner{}
	for i := 0; i < count; i++ {
		value := i
		owner.items = append(owner.items, &value)
	}
	return owner, NewReadOnlyCollectionOverLiveReferences(owner.list)
}

// TestLiveListViewSeesLaterAdditions is the whole reason this source exists:
// the CLR view stores the List REFERENCE, so it sees what the owner does next.
func TestLiveListViewSeesLaterAdditions(t *testing.T) {
	owner, view := newLiveFixture(2)
	if got := view.Count(); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	extra := 99
	owner.items = append(owner.items, &extra)
	if got := view.Count(); got != 3 {
		t.Fatalf("Count = %d after the owner appended, want 3: the view is not live", got)
	}
	item, err := view.Item(2)
	if err != nil || item != &extra {
		t.Fatalf("Item(2) = %v, %v", item, err)
	}
	// And a removal is seen too.
	owner.items = owner.items[:1]
	if got := view.Count(); got != 1 {
		t.Fatalf("Count = %d after the owner truncated, want 1", got)
	}
}

// TestLiveListItemChecksItsBounds pins the indexer's range failure, which is
// the underlying list's rather than the view's.
func TestLiveListItemChecksItsBounds(t *testing.T) {
	_, view := newLiveFixture(2)
	if _, err := view.Item(0); err != nil {
		t.Fatalf("Item(0): %v", err)
	}
	if _, err := view.Item(2); err == nil {
		t.Fatal("Item accepted an index at the end")
	}
	if _, err := view.Item(-1); err == nil {
		t.Fatal("Item accepted a negative index")
	}
	_, empty := newLiveFixture(0)
	if _, err := empty.Item(0); err == nil {
		t.Fatal("Item on an empty view accepted index 0")
	}
}

// TestLiveListCopyToCarriesItsThreeFailures pins CopyTo's guards in order.
func TestLiveListCopyToCarriesItsThreeFailures(t *testing.T) {
	_, view := newLiveFixture(3)
	destination := make([]*int, 5)
	if err := view.CopyTo(destination, 1); err != nil {
		t.Fatalf("CopyTo: %v", err)
	}
	if destination[0] != nil || destination[1] == nil || destination[3] == nil {
		t.Fatal("CopyTo wrote the wrong slots")
	}
	if err := view.CopyTo(nil, 0); err == nil {
		t.Fatal("CopyTo accepted a nil destination")
	}
	if err := view.CopyTo(destination, -1); err == nil {
		t.Fatal("CopyTo accepted a negative index")
	}
	if err := view.CopyTo(make([]*int, 2), 0); err == nil {
		t.Fatal("CopyTo accepted a destination too small")
	}
	// A destination that is big enough but not from that OFFSET is refused too.
	if err := view.CopyTo(make([]*int, 3), 1); err == nil {
		t.Fatal("CopyTo accepted an offset that runs off the end")
	}
}

// TestLiveListEnumeratorIsVersionChecked pins the difference from the array
// source: List<T>.Enumerator fails fast once the list has changed.
func TestLiveListEnumeratorIsVersionChecked(t *testing.T) {
	owner, view := newLiveFixture(3)
	iterator := view.GetEnumerator()
	if _, ok, err := iterator.Next(); err != nil || !ok {
		t.Fatalf("first step: %v, %v", ok, err)
	}
	extra := 42
	owner.items = append(owner.items, &extra)
	if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("Next after a mutation = %v, want the enumeration-failed refusal", err)
	}

	// An undisturbed walk visits every element and then stops cleanly.
	fresh := view.GetEnumerator()
	seen := 0
	for {
		_, ok, err := fresh.Next()
		if err != nil {
			t.Fatalf("undisturbed enumeration: %v", err)
		}
		if !ok {
			break
		}
		seen++
	}
	if seen != 4 {
		t.Fatalf("enumerated %d elements, want 4", seen)
	}
}

// TestLiveListIndexOfIsIdentity pins the comparer: the reference element type
// overrides no Equals, so EqualityComparer<T>.Default is reference identity.
func TestLiveListIndexOfIsIdentity(t *testing.T) {
	owner, view := newLiveFixture(2)
	if got := view.IndexOf(owner.items[1]); got != 1 {
		t.Fatalf("IndexOf = %d, want 1", got)
	}
	// A DIFFERENT pointer holding the same value is not the same element.
	same := *owner.items[1]
	if got := view.IndexOf(&same); got != -1 {
		t.Fatalf("IndexOf of an equal-but-distinct pointer = %d, want -1", got)
	}
	if !view.Contains(owner.items[0]) {
		t.Fatal("Contains missed an element the list holds")
	}
	if view.Contains(&same) {
		t.Fatal("Contains matched by value; the comparer is identity")
	}
}
