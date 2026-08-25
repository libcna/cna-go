package framework

import (
	"errors"
	"testing"
)

// hookRecorder is a minimal collectionOverrides[T] that records every hook
// call and then performs the base behavior.
//
// It exists to prove the routing claim independently of any XNA type: every
// mutating public operation of Collection<T> must reach the equivalent
// overridable hook, so a subclass override always runs and nothing appends to
// the backing store behind its back.
type hookRecorder struct {
	base  collectionBase[string]
	calls []string
}

func newHookRecorder() *hookRecorder {
	recorder := &hookRecorder{}
	recorder.base.init(recorder, func(a, b string) bool { return a == b })
	return recorder
}

func (h *hookRecorder) insertItem(index int32, item string) error {
	h.calls = append(h.calls, "insertItem")
	h.base.baseInsertItem(index, item)
	return nil
}

func (h *hookRecorder) removeItem(index int32) error {
	h.calls = append(h.calls, "removeItem")
	h.base.baseRemoveItem(index)
	return nil
}

func (h *hookRecorder) setItem(index int32, item string) error {
	h.calls = append(h.calls, "setItem")
	h.base.baseSetItem(index, item)
	return nil
}

func (h *hookRecorder) clearItems() error {
	h.calls = append(h.calls, "clearItems")
	h.base.baseClearItems()
	return nil
}

func (h *hookRecorder) lastCall() string {
	if len(h.calls) == 0 {
		return ""
	}
	return h.calls[len(h.calls)-1]
}

func TestEveryMutatorRoutesThroughItsHook(t *testing.T) {
	cases := []struct {
		name string
		run  func(*hookRecorder) error
		hook string
	}{
		{"Add", func(h *hookRecorder) error { return h.base.add("x") }, "insertItem"},
		{"Insert", func(h *hookRecorder) error { return h.base.insert(0, "x") }, "insertItem"},
		{"Clear", func(h *hookRecorder) error { return h.base.clear() }, "clearItems"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := newHookRecorder()
			if err := testCase.run(recorder); err != nil {
				t.Fatal(err)
			}
			if recorder.lastCall() != testCase.hook {
				t.Fatalf("%s reached %q, want %q", testCase.name, recorder.lastCall(), testCase.hook)
			}
		})
	}

	// The operations that need an element present first.
	populated := []struct {
		name string
		run  func(*hookRecorder) error
		hook string
	}{
		{"RemoveAt", func(h *hookRecorder) error { return h.base.removeAt(0) }, "removeItem"},
		{"Remove", func(h *hookRecorder) error { _, err := h.base.remove("seed"); return err }, "removeItem"},
		{"Assign", func(h *hookRecorder) error { return h.base.assign(0, "y") }, "setItem"},
	}
	for _, testCase := range populated {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := newHookRecorder()
			if err := recorder.base.add("seed"); err != nil {
				t.Fatal(err)
			}
			recorder.calls = nil
			if err := testCase.run(recorder); err != nil {
				t.Fatal(err)
			}
			if recorder.lastCall() != testCase.hook {
				t.Fatalf("%s reached %q, want %q", testCase.name, recorder.lastCall(), testCase.hook)
			}
		})
	}
}

func TestAddIsDefinedAsInsertAtCount(t *testing.T) {
	recorder := newHookRecorder()
	var indices []int32
	recorder.base.init(&indexCapturingOverrides{base: &recorder.base, indices: &indices}, func(a, b string) bool { return a == b })
	for _, value := range []string{"a", "b", "c"} {
		if err := recorder.base.add(value); err != nil {
			t.Fatal(err)
		}
	}
	// Collection<T>.Add is `InsertItem(Count, item)`, so the hook sees the
	// growing index rather than a separate append path.
	if len(indices) != 3 || indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Fatalf("hook indices = %v, want [0 1 2]", indices)
	}
}

type indexCapturingOverrides struct {
	base    *collectionBase[string]
	indices *[]int32
}

func (o *indexCapturingOverrides) insertItem(index int32, item string) error {
	*o.indices = append(*o.indices, index)
	o.base.baseInsertItem(index, item)
	return nil
}
func (o *indexCapturingOverrides) removeItem(index int32) error {
	o.base.baseRemoveItem(index)
	return nil
}
func (o *indexCapturingOverrides) setItem(index int32, item string) error {
	o.base.baseSetItem(index, item)
	return nil
}
func (o *indexCapturingOverrides) clearItems() error {
	o.base.baseClearItems()
	return nil
}

func TestAHookThatRefusesLeavesTheStoreUntouched(t *testing.T) {
	refused := errors.New("refused")
	recorder := newHookRecorder()
	recorder.base.init(&refusingOverrides{err: refused}, func(a, b string) bool { return a == b })
	if err := recorder.base.add("x"); !errors.Is(err, refused) {
		t.Fatalf("Add = %v, want the hook's refusal", err)
	}
	if recorder.base.count() != 0 {
		t.Fatalf("Count = %d: a refusing hook must not mutate", recorder.base.count())
	}
	// No mutation means no version change, so a live enumerator survives.
	iterator := recorder.base.getEnumerator()
	if _, ok, err := iterator.Next(); ok || err != nil {
		t.Fatalf("enumeration = ok %v err %v", ok, err)
	}
}

type refusingOverrides struct{ err error }

func (o *refusingOverrides) insertItem(int32, string) error { return o.err }
func (o *refusingOverrides) removeItem(int32) error         { return o.err }
func (o *refusingOverrides) setItem(int32, string) error    { return o.err }
func (o *refusingOverrides) clearItems() error              { return o.err }

func TestVersionMovesExactlyWhereListMovesIt(t *testing.T) {
	recorder := newHookRecorder()
	before := recorder.base.version

	// Reads never move it.
	recorder.base.count()
	_, _ = recorder.base.item(0)
	recorder.base.indexOf("a")
	recorder.base.contains("a")
	recorder.base.getEnumerator()
	if recorder.base.version != before {
		t.Fatalf("a read moved the version from %d to %d", before, recorder.base.version)
	}

	// Every List<T> mutator moves it by exactly one.
	steps := []struct {
		name string
		run  func() error
	}{
		{"insert", func() error { return recorder.base.add("a") }},
		{"insert again", func() error { return recorder.base.add("b") }},
		{"assign", func() error { return recorder.base.assign(0, "c") }},
		{"removeAt", func() error { return recorder.base.removeAt(0) }},
		{"clear", func() error { return recorder.base.clear() }},
		// List<T>.Clear increments unconditionally, so clearing an already
		// empty list moves the version again.
		{"clear when empty", func() error { return recorder.base.clear() }},
	}
	for _, step := range steps {
		previous := recorder.base.version
		if err := step.run(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if recorder.base.version != previous+1 {
			t.Fatalf("%s moved the version from %d to %d, want exactly one step", step.name, previous, recorder.base.version)
		}
	}
}

func TestCopyToDoesNotAliasTheStore(t *testing.T) {
	recorder := newHookRecorder()
	for _, value := range []string{"a", "b"} {
		if err := recorder.base.add(value); err != nil {
			t.Fatal(err)
		}
	}
	destination := make([]string, 4)
	if err := recorder.base.copyTo(destination, 1); err != nil {
		t.Fatal(err)
	}
	destination[1] = "hijacked"
	first, err := recorder.base.item(0)
	if err != nil {
		t.Fatal(err)
	}
	if first != "a" {
		t.Fatalf("store observed %q after the copy was mutated", first)
	}
}

func TestInsertGuardAdmitsCountAndRemoveAtDoesNot(t *testing.T) {
	recorder := newHookRecorder()
	if err := recorder.base.add("a"); err != nil {
		t.Fatal(err)
	}
	// Insert's guard is `index > Count`; the indexer setter's and RemoveAt's
	// are `index >= Count`.
	if err := recorder.base.insert(1, "b"); err != nil {
		t.Fatalf("Insert at Count must be legal: %v", err)
	}
	if err := recorder.base.removeAt(2); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("RemoveAt at Count must fail, got %v", err)
	}
	if err := recorder.base.assign(2, "c"); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("assignment at Count must fail, got %v", err)
	}
}

func TestNegativeIndexFailsTheSameUnsignedGuard(t *testing.T) {
	recorder := newHookRecorder()
	if err := recorder.base.add("a"); err != nil {
		t.Fatal(err)
	}
	// List<T> compares `(uint)index >= (uint)_size`, so -1 becomes a huge
	// unsigned value and fails the same test as an index past the end.
	if _, err := recorder.base.item(-1); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("Item(-1) = %v", err)
	}
	if _, err := recorder.base.item(int32(-2147483648)); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatal("the most negative index must fail too")
	}
}

func TestEnumeratorChecksVersionBeforeBounds(t *testing.T) {
	recorder := newHookRecorder()
	iterator := recorder.base.getEnumerator()
	if err := recorder.base.add("a"); err != nil {
		t.Fatal(err)
	}
	// The collection was empty when the enumerator was taken, so a bounds-first
	// implementation would report a clean finish. MoveNext compares the
	// version first, so this must fail instead.
	if _, ok, err := iterator.Next(); ok || !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("enumeration = ok %v err %v, want the version failure", ok, err)
	}
}

func TestEnumerationYieldsSourceOrder(t *testing.T) {
	recorder := newHookRecorder()
	for _, value := range []string{"a", "b", "c"} {
		if err := recorder.base.add(value); err != nil {
			t.Fatal(err)
		}
	}
	var seen []string
	iterator := recorder.base.getEnumerator()
	for {
		value, ok, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		seen = append(seen, value)
	}
	if len(seen) != 3 || seen[0] != "a" || seen[1] != "b" || seen[2] != "c" {
		t.Fatalf("enumeration = %v, want [a b c]", seen)
	}
}

func TestContainsFindsANilElementByTheNullBranch(t *testing.T) {
	// List<T>.Contains splits on whether the sought item is null: it scans for
	// a null element rather than consulting the comparer.
	collection := &nilBearingCollection{}
	collection.base.init(collection, referenceIdentityEqual[IGameComponent])
	if err := collection.base.add(nil); err != nil {
		t.Fatal(err)
	}
	if !collection.base.contains(nil) {
		t.Fatal("a stored nil element must be found")
	}
	if collection.base.indexOf(nil) != 0 {
		t.Fatalf("IndexOf(nil) = %d, want 0", collection.base.indexOf(nil))
	}
}

type nilBearingCollection struct {
	base collectionBase[IGameComponent]
}

func (c *nilBearingCollection) insertItem(index int32, item IGameComponent) error {
	c.base.baseInsertItem(index, item)
	return nil
}
func (c *nilBearingCollection) removeItem(index int32) error {
	c.base.baseRemoveItem(index)
	return nil
}
func (c *nilBearingCollection) setItem(index int32, item IGameComponent) error {
	c.base.baseSetItem(index, item)
	return nil
}
func (c *nilBearingCollection) clearItems() error {
	c.base.baseClearItems()
	return nil
}
