package touch

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func sampleLocation(id int32, x, y float32) TouchLocation {
	return NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		id, TouchLocationStateMoved, framework.Vector2{X: x, Y: y})
}

func sampleLocationWithPrevious(id int32, x, y, previousX, previousY float32) TouchLocation {
	return NewTouchLocationByInt32AndTouchLocationStateAndVector2AndTouchLocationStateAndVector2(
		id, TouchLocationStateMoved, framework.Vector2{X: x, Y: y},
		TouchLocationStatePressed, framework.Vector2{X: previousX, Y: previousY})
}

func mustCollection(t *testing.T, touches []TouchLocation) TouchCollection {
	t.Helper()
	collection, err := NewTouchCollection(touches)
	if err != nil {
		t.Fatalf("NewTouchCollection = %v", err)
	}
	return collection
}

// TestTouchCollectionConstructorValidatesInReferenceOrder pins both guards and
// the order the reference checks them in: nil first, then the eight-slot
// capacity.
func TestTouchCollectionConstructorValidatesInReferenceOrder(t *testing.T) {
	if _, err := NewTouchCollection(nil); !errors.Is(err, errTouchArgumentNull) {
		t.Fatalf("nil slice = %v, want an argument-null projection", err)
	}

	nine := make([]TouchLocation, 9)
	if _, err := NewTouchCollection(nine); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("nine touches = %v, want an out-of-range projection", err)
	}

	// Exactly eight is accepted: the check is `> 8`, not `>= 8`.
	eight := make([]TouchLocation, 8)
	for i := range eight {
		eight[i] = sampleLocation(int32(i), float32(i), float32(i))
	}
	full := mustCollection(t, eight)
	if full.Count() != 8 {
		t.Fatalf("eight touches produced Count = %d", full.Count())
	}
	// An empty but non-nil slice is accepted and yields a connected, empty
	// collection.
	empty := mustCollection(t, []TouchLocation{})
	if empty.Count() != 0 || !empty.IsConnected() {
		t.Fatalf("empty slice = Count %d, IsConnected %t", empty.Count(), empty.IsConnected())
	}
}

// TestTouchCollectionConstructorCarriesPreviousSamples pins that the
// constructor asks each location for its previous sample and stores a zeroed
// previous state when there is none.
func TestTouchCollectionConstructorCarriesPreviousSamples(t *testing.T) {
	withPrevious := sampleLocationWithPrevious(7, 10, 20, 1, 2)
	withoutPrevious := sampleLocation(8, 30, 40)
	collection := mustCollection(t, []TouchLocation{withPrevious, withoutPrevious})

	stored, err := collection.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	hasPrevious, previous := stored.TryGetPreviousLocation()
	if !hasPrevious {
		t.Fatal("the stored location lost its previous sample")
	}
	if got := previous.Position(); got != (framework.Vector2{X: 1, Y: 2}) {
		t.Fatalf("previous position = %+v", got)
	}
	if previous.State() != TouchLocationStatePressed {
		t.Fatalf("previous state = %d", previous.State())
	}

	stored, err = collection.Item(1)
	if err != nil {
		t.Fatal(err)
	}
	if hasPrevious, _ := stored.TryGetPreviousLocation(); hasPrevious {
		t.Fatal("a location with no previous sample gained one")
	}
	if stored.Position() != (framework.Vector2{X: 30, Y: 40}) || stored.Id() != 8 {
		t.Fatalf("second location = %+v", stored)
	}
}

// TestTouchCollectionIndexerValidates pins the one reference check on the
// getter and the unconditional throw on the setter.
func TestTouchCollectionIndexerValidates(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(1, 1, 1), sampleLocation(2, 2, 2)})

	for _, index := range []int32{-1, 2, 8, 1 << 30} {
		if _, err := collection.Item(index); !errors.Is(err, errTouchArgumentOutOfRange) {
			t.Fatalf("Item(%d) = %v, want an out-of-range projection", index, err)
		}
	}
	got, err := collection.Item(1)
	if err != nil || got.Id() != 2 {
		t.Fatalf("Item(1) = %+v, %v", got, err)
	}

	// The setter throws before validating anything, so even a valid index and
	// value are rejected.
	if err := collection.SetItem(0, sampleLocation(9, 9, 9)); !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("SetItem = %v, want a not-supported projection", err)
	}
	if unchanged, _ := collection.Item(0); unchanged.Id() != 1 {
		t.Fatal("a rejected SetItem changed the collection")
	}
}

// TestTouchCollectionMutatorsAreUnconditionallyUnsupported pins that every
// IList<T> mutator throws with no validation at all, including on arguments
// that would otherwise be invalid.
func TestTouchCollectionMutatorsAreUnconditionallyUnsupported(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(1, 1, 1)})
	item := sampleLocation(5, 5, 5)

	if err := collection.Add(item); !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("Add = %v", err)
	}
	if err := collection.Clear(); !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("Clear = %v", err)
	}
	// A wildly invalid index still reports not-supported, not out-of-range:
	// the reference throws before looking at the argument.
	if err := collection.Insert(-99, item); !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("Insert = %v", err)
	}
	if err := collection.RemoveAt(-99); !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("RemoveAt = %v", err)
	}
	removed, err := collection.Remove(item)
	if removed || !errors.Is(err, errTouchNotSupported) {
		t.Fatalf("Remove = %t, %v", removed, err)
	}
	if collection.Count() != 1 {
		t.Fatalf("a rejected mutation changed Count to %d", collection.Count())
	}
	if !collection.IsReadOnly() {
		t.Fatal("IsReadOnly = false")
	}
	// IsReadOnly is a constant, so even a zero-valued collection reports true.
	var zero TouchCollection
	if !zero.IsReadOnly() {
		t.Fatal("zero-valued IsReadOnly = false")
	}
	if zero.IsConnected() || zero.Count() != 0 {
		t.Fatalf("zero-valued collection = connected %t, Count %d", zero.IsConnected(), zero.Count())
	}
}

// TestTouchCollectionSearchUsesOperatorEquality pins the search semantics: the
// reference compares with the equality operator, which weighs all seven fields
// including both state fields, not the looser Equals.
func TestTouchCollectionSearchUsesOperatorEquality(t *testing.T) {
	first := sampleLocation(1, 10, 20)
	second := sampleLocationWithPrevious(2, 30, 40, 3, 4)
	collection := mustCollection(t, []TouchLocation{first, second})

	stored0, _ := collection.Item(0)
	stored1, _ := collection.Item(1)
	if got := collection.IndexOf(stored0); got != 0 {
		t.Fatalf("IndexOf(first) = %d", got)
	}
	if got := collection.IndexOf(stored1); got != 1 {
		t.Fatalf("IndexOf(second) = %d", got)
	}
	if !collection.Contains(stored0) || !collection.Contains(stored1) {
		t.Fatal("Contains missed a stored location")
	}

	// Same id and position, different state: Equals would accept it but the
	// operator does not, so the search misses.
	sameIdDifferentState := NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		1, TouchLocationStateReleased, framework.Vector2{X: 10, Y: 20})
	if !stored0.EqualsByTouchLocation(sameIdDifferentState) {
		t.Fatal("the fixture no longer exercises the Equals/operator disagreement")
	}
	if got := collection.IndexOf(sameIdDifferentState); got != -1 {
		t.Fatalf("IndexOf ignored a state difference: %d", got)
	}
	if collection.Contains(sameIdDifferentState) {
		t.Fatal("Contains ignored a state difference")
	}
	if got := collection.IndexOf(sampleLocation(99, 0, 0)); got != -1 {
		t.Fatalf("IndexOf(absent) = %d", got)
	}
}

// TestTouchCollectionFindByIdComparesIdentifiersOnly pins that FindById
// ignores position and state entirely and yields a zero location on a miss.
func TestTouchCollectionFindByIdComparesIdentifiersOnly(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(4, 1, 2), sampleLocation(9, 3, 4)})

	found, location := collection.FindById(9)
	if !found || location.Id() != 9 || location.Position() != (framework.Vector2{X: 3, Y: 4}) {
		t.Fatalf("FindById(9) = %t, %+v", found, location)
	}
	found, location = collection.FindById(1234)
	if found {
		t.Fatal("FindById reported a missing id")
	}
	// The miss yields a zero location, not the Id -1 sentinel that
	// TouchLocation.TryGetPreviousLocation uses.
	if location != (TouchLocation{}) {
		t.Fatalf("FindById miss = %+v, want the zero location", location)
	}
	var zero TouchCollection
	if found, _ := zero.FindById(0); found {
		t.Fatal("an empty collection matched an id")
	}
}

// TestTouchCollectionCopyToValidatesInReferenceOrder pins all three checks and
// the 64-bit capacity arithmetic that keeps a near-maximal start index from
// wrapping into a false pass.
func TestTouchCollectionCopyToValidatesInReferenceOrder(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(1, 1, 1), sampleLocation(2, 2, 2)})

	if err := collection.CopyTo(nil, 0); !errors.Is(err, errTouchArgumentNull) {
		t.Fatalf("nil destination = %v", err)
	}
	destination := make([]TouchLocation, 4)
	if err := collection.CopyTo(destination, -1); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("negative start = %v", err)
	}
	if err := collection.CopyTo(destination, 3); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("insufficient destination = %v", err)
	}
	// 64-bit arithmetic: arrayIndex + Count overflows int32 here, and the
	// reference still reports the argument error rather than wrapping.
	const nearMax = int32(1<<31 - 1)
	if err := collection.CopyTo(destination, nearMax); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("overflowing start = %v", err)
	}

	if err := collection.CopyTo(destination, 2); err != nil {
		t.Fatalf("CopyTo = %v", err)
	}
	if destination[0] != (TouchLocation{}) || destination[1] != (TouchLocation{}) {
		t.Fatal("CopyTo wrote before its start index")
	}
	if destination[2].Id() != 1 || destination[3].Id() != 2 {
		t.Fatalf("CopyTo wrote %+v", destination)
	}
	// An exactly-sized destination is accepted: the check is `<`, not `<=`.
	exact := make([]TouchLocation, 2)
	if err := collection.CopyTo(exact, 0); err != nil {
		t.Fatalf("exactly-sized destination = %v", err)
	}
	// An empty collection copies nothing and accepts any valid start.
	var zero TouchCollection
	if err := zero.CopyTo(exact, 2); err != nil {
		t.Fatalf("empty CopyTo = %v", err)
	}
}

// TestTouchCollectionEnumeratorCursor pins the cursor's exact positions,
// including that Current reports an error before the first MoveNext and again
// after exhaustion.
func TestTouchCollectionEnumeratorCursor(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(1, 1, 1), sampleLocation(2, 2, 2)})
	enumerator := collection.GetEnumerator()

	// Position starts at -1, so Current fails before the first MoveNext.
	if _, err := enumerator.Current(); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("Current before MoveNext = %v", err)
	}

	var ids []int32
	for enumerator.MoveNext() {
		location, err := enumerator.Current()
		if err != nil {
			t.Fatalf("Current during enumeration = %v", err)
		}
		ids = append(ids, location.Id())
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("enumeration order = %v", ids)
	}

	// After exhaustion the position is clamped, so repeated MoveNext keeps
	// reporting false and Current keeps failing rather than drifting.
	for i := 0; i < 3; i++ {
		if enumerator.MoveNext() {
			t.Fatal("MoveNext reported true after exhaustion")
		}
	}
	if _, err := enumerator.Current(); !errors.Is(err, errTouchArgumentOutOfRange) {
		t.Fatalf("Current after exhaustion = %v", err)
	}
	// Dispose owns nothing and is safe to call repeatedly.
	enumerator.Dispose()
	enumerator.Dispose()

	// An empty collection yields a cursor that never advances.
	var zero TouchCollection
	emptyCursor := zero.GetEnumerator()
	if emptyCursor.MoveNext() {
		t.Fatal("an empty collection advanced")
	}
}

// TestTouchCollectionKeepsValueSemantics pins that the collection is a CLR
// value type: assigning it copies, and the enumerator holds its own copy.
func TestTouchCollectionKeepsValueSemantics(t *testing.T) {
	collection := mustCollection(t, []TouchLocation{sampleLocation(1, 1, 1)})
	copied := collection
	if copied.Count() != collection.Count() {
		t.Fatal("the copy disagrees about Count")
	}
	// Both are read-only, so the observable value-semantics claim is that a
	// copy is a distinct, equal snapshot rather than an alias.
	if copied != collection {
		t.Fatal("a copy is not equal to its source")
	}

	// The enumerator captures a copy, so rebinding the source variable leaves
	// the cursor's view intact.
	enumerator := collection.GetEnumerator()
	collection = mustCollection(t, []TouchLocation{
		sampleLocation(7, 7, 7), sampleLocation(8, 8, 8), sampleLocation(9, 9, 9),
	})
	count := 0
	for enumerator.MoveNext() {
		count++
	}
	if count != 1 {
		t.Fatalf("the cursor followed its source variable: saw %d", count)
	}
	if collection.Count() != 3 {
		t.Fatalf("the rebound collection = %d", collection.Count())
	}
}
