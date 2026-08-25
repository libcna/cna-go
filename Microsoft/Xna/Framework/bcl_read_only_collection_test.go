package framework

import (
	"errors"
	"math"
	"testing"
)

func readOnlyOverInt32(items []int32) *ReadOnlyCollection[int32] {
	return newReadOnlyCollectionOverSlice(items, func(a, b int32) bool { return a == b })
}

func TestReadOnlyCollectionForwardsEveryRead(t *testing.T) {
	view := readOnlyOverInt32([]int32{10, 20, 30})
	if view.Count() != 3 {
		t.Fatalf("Count = %d, want 3", view.Count())
	}
	value, err := view.Item(1)
	if err != nil || value != 20 {
		t.Fatalf("Item(1) = %d, %v", value, err)
	}
	if view.IndexOf(30) != 2 || view.IndexOf(99) != -1 {
		t.Fatalf("IndexOf = %d / %d", view.IndexOf(30), view.IndexOf(99))
	}
	if !view.Contains(10) || view.Contains(99) {
		t.Fatal("Contains disagrees with IndexOf")
	}
}

func TestReadOnlyCollectionIsLiveOverTheListItWasGiven(t *testing.T) {
	// ReadOnlyCollection<T> STORES the list; it does not copy it. Read-only
	// means the caller cannot write through the view, not that the data is
	// frozen, so an owner's later write is visible.
	items := []int32{1, 2, 3}
	view := readOnlyOverInt32(items)
	items[1] = 99
	value, err := view.Item(1)
	if err != nil {
		t.Fatal(err)
	}
	if value != 99 {
		t.Fatalf("Item(1) = %d, want 99: the view must be live over the list", value)
	}
	if !view.Contains(99) {
		t.Fatal("a search must see the current contents")
	}
}

func TestReadOnlyCollectionIsBoundToTheListInstanceNotTheOwnersField(t *testing.T) {
	// In the CLR the view holds the array REFERENCE it was handed, so an owner
	// that later points its own field at a different array does not change
	// what an existing view shows. A captured Go slice header reproduces that;
	// a *[]T would be more live than the reference.
	items := []int32{1, 2, 3}
	view := readOnlyOverInt32(items)
	items = []int32{7, 8, 9, 10}
	_ = items
	if view.Count() != 3 {
		t.Fatalf("Count = %d, want 3: the view is bound to the original list", view.Count())
	}
	value, err := view.Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if value != 1 {
		t.Fatalf("Item(0) = %d, want 1", value)
	}
}

func TestReadOnlyCollectionBoundsMatchTheUnderlyingArray(t *testing.T) {
	view := readOnlyOverInt32([]int32{1, 2})
	for _, index := range []int32{-1, 2, 3, math.MinInt32} {
		if _, err := view.Item(index); !errors.Is(err, errCollectionArgumentOutOfRange) {
			t.Fatalf("Item(%d) = %v, want the range failure", index, err)
		}
	}
}

func TestReadOnlyCollectionCopyToCarriesArrayCopyFailures(t *testing.T) {
	view := readOnlyOverInt32([]int32{1, 2})
	if err := view.CopyTo(nil, 0); !errors.Is(err, errCollectionArgumentNull) {
		t.Fatalf("nil destination = %v", err)
	}
	destination := make([]int32, 3)
	if err := view.CopyTo(destination, -1); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("negative index = %v", err)
	}
	if err := view.CopyTo(destination, 2); !errors.Is(err, errCollectionArgument) {
		t.Fatalf("destination too small = %v", err)
	}
	if err := view.CopyTo(destination, 1); err != nil {
		t.Fatal(err)
	}
	if destination[0] != 0 || destination[1] != 1 || destination[2] != 2 {
		t.Fatalf("copied %v", destination)
	}
	// The copy is a copy.
	destination[1] = 99
	value, err := view.Item(0)
	if err != nil || value != 1 {
		t.Fatalf("CopyTo aliased the list: Item(0) = %d, %v", value, err)
	}
}

func TestArrayBackedEnumerationIsNotVersionChecked(t *testing.T) {
	// This is the measured difference between the two sources. A List<T>
	// -backed view enumerates fail-fast; an ARRAY-backed one does not, because
	// SZArrayHelper.SZGenericArrayEnumerator<T> holds only _array, _index and
	// _endIndex and compares the index against the end and nothing else. An
	// array cannot change length, so there is no version to check.
	items := []int32{1, 2, 3}
	view := readOnlyOverInt32(items)
	iterator := view.GetEnumerator()
	first, ok, err := iterator.Next()
	if !ok || err != nil || first != 1 {
		t.Fatalf("first step = %d ok %v err %v", first, ok, err)
	}
	items[1] = 99
	second, ok, err := iterator.Next()
	if !ok || err != nil {
		t.Fatalf("an element write must not invalidate an array-backed enumerator: ok %v err %v", ok, err)
	}
	if second != 99 {
		t.Fatalf("second step = %d, want the current element 99", second)
	}
	third, ok, err := iterator.Next()
	if !ok || err != nil || third != 3 {
		t.Fatalf("third step = %d ok %v err %v", third, ok, err)
	}
	if _, ok, err := iterator.Next(); ok || err != nil {
		t.Fatalf("exhausted enumerator = ok %v err %v", ok, err)
	}
}

func TestListBackedViewKeepsItsVersionCheck(t *testing.T) {
	// The other half of the same claim: the view forwards, so a List<T>
	// -backed source keeps the fail-fast enumeration the array source lacks.
	recorder := newHookRecorder()
	if err := recorder.base.add("a"); err != nil {
		t.Fatal(err)
	}
	view := &ReadOnlyCollection[string]{source: &recorder.base}
	iterator := view.GetEnumerator()
	if err := recorder.base.add("b"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("a List<T>-backed view must stay version-checked, got %v", err)
	}
	if view.Count() != 2 {
		t.Fatalf("Count = %d, want 2: the view is live over the collection", view.Count())
	}
}

func TestReadOnlyCollectionOverSinglesUsesTheCLRComparer(t *testing.T) {
	nan := float32(math.NaN())
	view := NewReadOnlyCollectionOverSingles([]float32{1, nan, -0})
	// System.Single::Equals returns true for two NaNs, so a NaN search finds
	// a NaN element. Go's == would report false.
	if view.IndexOf(nan) != 1 {
		t.Fatalf("IndexOf(NaN) = %d, want 1: EqualityComparer<Single>.Default treats NaN as equal", view.IndexOf(nan))
	}
	if !view.Contains(nan) {
		t.Fatal("Contains(NaN) must be true")
	}
	// Signed zeros stay equal in both languages, so the search finds -0 at 2
	// by matching +0 first at... no element is +0, so index 2 is the match.
	if view.IndexOf(0) != 2 {
		t.Fatalf("IndexOf(0) = %d, want 2", view.IndexOf(0))
	}
	if view.IndexOf(2) != -1 {
		t.Fatal("an absent value must report -1")
	}
}

func TestSingleEqualsMatchesTheReference(t *testing.T) {
	nan, other := float32(math.NaN()), float32(math.Float32frombits(0x7fc00001))
	cases := []struct {
		left, right float32
		want        bool
	}{
		{1, 1, true},
		{1, 2, false},
		{0, float32(math.Copysign(0, -1)), true},
		{nan, nan, true},
		{nan, other, true},
		{nan, 1, false},
		{1, nan, false},
		{float32(math.Inf(1)), float32(math.Inf(1)), true},
		{float32(math.Inf(1)), float32(math.Inf(-1)), false},
	}
	for _, testCase := range cases {
		if got := singleEquals(testCase.left, testCase.right); got != testCase.want {
			t.Fatalf("singleEquals(%v, %v) = %v, want %v", testCase.left, testCase.right, got, testCase.want)
		}
	}
}

func TestEmptyReadOnlyCollection(t *testing.T) {
	view := readOnlyOverInt32(nil)
	if view.Count() != 0 || view.IndexOf(1) != -1 || view.Contains(1) {
		t.Fatal("an empty view reported contents")
	}
	if _, err := view.Item(0); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("Item(0) on an empty view = %v", err)
	}
	if _, ok, err := view.GetEnumerator().Next(); ok || err != nil {
		t.Fatalf("empty enumeration = ok %v err %v", ok, err)
	}
	if err := view.CopyTo(make([]int32, 0), 0); err != nil {
		t.Fatalf("copying nothing into nothing = %v", err)
	}
}
