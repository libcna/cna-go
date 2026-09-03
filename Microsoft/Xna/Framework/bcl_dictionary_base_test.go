package framework

import (
	"errors"
	"testing"
)

// newTestDictionary builds a bare dictionaryBase<string,string> the way
// Dictionary<string,string>'s parameterless constructor does: no buckets, the
// default string comparer, and default string equality for ContainsValue.
func newTestDictionary(t *testing.T) *dictionaryBase[string, string] {
	t.Helper()
	dictionary := &dictionaryBase[string, string]{}
	dictionary.init(defaultStringComparer, func(a, b string) bool { return a == b })
	return dictionary
}

// drain walks an iterator to exhaustion and reports the keys it yielded.
func drainKeys(t *testing.T, iterator Iterator[KeyValuePair[string, string]]) []string {
	t.Helper()
	var keys []string
	for {
		pair, ok, err := iterator.Next()
		if err != nil {
			t.Fatalf("enumeration failed: %v", err)
		}
		if !ok {
			return keys
		}
		keys = append(keys, pair.Key())
	}
}

func TestDictionaryBaseStartsEmptyAndUninitialised(t *testing.T) {
	dictionary := newTestDictionary(t)
	// `.ctor()` is `.ctor(0, null)`, and .ctor(int32, comparer) calls
	// Initialize only when capacity > 0, so the bucket array is genuinely
	// absent rather than empty.
	if dictionary.buckets != nil {
		t.Fatalf("parameterless construction allocated buckets: %v", dictionary.buckets)
	}
	if dictionary.countOf() != 0 {
		t.Fatalf("Count = %d, want 0", dictionary.countOf())
	}
	// FindEntry short-circuits on the null bucket array, so a lookup before the
	// first insertion must not panic.
	if dictionary.containsKey("absent") {
		t.Fatal("ContainsKey found a key in a dictionary that has no buckets")
	}
	if _, err := dictionary.item("absent"); !errors.Is(err, errDictionaryKeyNotFound) {
		t.Fatalf("get_Item on an empty dictionary = %v, want KeyNotFoundException", err)
	}
	if dictionary.remove("absent") {
		t.Fatal("Remove reported success on a dictionary that has no buckets")
	}
}

func TestDictionaryBaseAddRefusesADuplicateAndSetItemDoesNot(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("k", "first"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Insert(add: true) reaches ThrowHelper.ThrowArgumentException BEFORE it
	// writes, so the existing value must survive the refusal.
	if err := dictionary.add("k", "second"); !errors.Is(err, errDictionaryDuplicateKey) {
		t.Fatalf("duplicate Add = %v, want ArgumentException", err)
	}
	if value, err := dictionary.item("k"); err != nil || value != "first" {
		t.Fatalf("after a refused Add, Item = %q, %v", value, err)
	}
	// set_Item is Insert(add: false), which overwrites.
	dictionary.assign("k", "second")
	if value, err := dictionary.item("k"); err != nil || value != "second" {
		t.Fatalf("after set_Item, Item = %q, %v", value, err)
	}
	// and adds a key that is not there.
	dictionary.assign("new", "value")
	if !dictionary.containsKey("new") {
		t.Fatal("set_Item did not add a missing key")
	}
	if dictionary.countOf() != 2 {
		t.Fatalf("Count = %d, want 2", dictionary.countOf())
	}
}

// TestDictionaryBaseEnumerationIsEntriesOrderNotInsertionOrder is the test a Go
// map could not pass, and the reason the adapter reproduces the reference's
// structure instead of wrapping one.
//
// Remove pushes the freed slot onto the HEAD of the free list, and the next
// Insert takes it, so the re-added key appears at the REMOVED key's old
// position rather than at the end.
func TestDictionaryBaseEnumerationReusesTheFreedSlot(t *testing.T) {
	dictionary := newTestDictionary(t)
	for _, key := range []string{"alpha", "beta", "gamma", "delta"} {
		if err := dictionary.add(key, key+"-value"); err != nil {
			t.Fatalf("Add %q: %v", key, err)
		}
	}
	if got := drainKeys(t, dictionary.getEnumerator()); !equalStrings(got, []string{"alpha", "beta", "gamma", "delta"}) {
		t.Fatalf("initial enumeration = %v", got)
	}
	if !dictionary.remove("beta") {
		t.Fatal("Remove(beta) reported failure")
	}
	if got := drainKeys(t, dictionary.getEnumerator()); !equalStrings(got, []string{"alpha", "gamma", "delta"}) {
		t.Fatalf("enumeration after removal = %v", got)
	}
	if err := dictionary.add("epsilon", "epsilon-value"); err != nil {
		t.Fatalf("Add epsilon: %v", err)
	}
	// Slot 1, beta's, is at the head of the free list, so epsilon lands there
	// and enumerates SECOND rather than last.
	if got := drainKeys(t, dictionary.getEnumerator()); !equalStrings(got, []string{"alpha", "epsilon", "gamma", "delta"}) {
		t.Fatalf("enumeration after reinsertion = %v, want the freed slot reused in place", got)
	}
	if dictionary.countOf() != 4 {
		t.Fatalf("Count = %d, want 4", dictionary.countOf())
	}
}

// TestDictionaryBaseSurvivesAResizeWithOrderIntact pins the other half of the
// order claim: Resize copies the entries at the SAME indices, so growing past
// the first prime cannot reshuffle anything.
func TestDictionaryBaseSurvivesAResizeWithOrderIntact(t *testing.T) {
	dictionary := newTestDictionary(t)
	var expected []string
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		if err := dictionary.add(key, key); err != nil {
			t.Fatalf("Add %q: %v", key, err)
		}
		expected = append(expected, key)
	}
	// The first prime is 3, so ten insertions have forced at least two resizes.
	if len(dictionary.buckets) < 10 {
		t.Fatalf("bucket count = %d, want a grown table", len(dictionary.buckets))
	}
	if got := drainKeys(t, dictionary.getEnumerator()); !equalStrings(got, expected) {
		t.Fatalf("enumeration after resize = %v, want %v", got, expected)
	}
	for _, key := range expected {
		if value, err := dictionary.item(key); err != nil || value != key {
			t.Fatalf("after resize, Item(%q) = %q, %v", key, value, err)
		}
	}
}

func TestDictionaryBaseEnumeratorFailsFastOnEveryMutation(t *testing.T) {
	mutations := map[string]func(d *dictionaryBase[string, string]){
		"add":     func(d *dictionaryBase[string, string]) { _ = d.add("new", "value") },
		"setItem": func(d *dictionaryBase[string, string]) { d.assign("present", "other") },
		"remove":  func(d *dictionaryBase[string, string]) { d.remove("present") },
		"clear":   func(d *dictionaryBase[string, string]) { d.clear() },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			dictionary := newTestDictionary(t)
			if err := dictionary.add("present", "value"); err != nil {
				t.Fatal(err)
			}
			iterator := dictionary.getEnumerator()
			mutate(dictionary)
			if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
				t.Fatalf("Next after %s = %v, want the version failure", name, err)
			}
		})
	}
}

// TestDictionaryBaseEmptyClearDoesNotInvalidate is the asymmetry with
// List<T>.Clear, and it is the reference's, not a simplification: Clear's whole
// body -- including `version++` -- is guarded by `count > 0`.
func TestDictionaryBaseEmptyClearDoesNotInvalidate(t *testing.T) {
	dictionary := newTestDictionary(t)
	iterator := dictionary.getEnumerator()
	dictionary.clear()
	if _, ok, err := iterator.Next(); err != nil || ok {
		t.Fatalf("Next after clearing an empty dictionary = %v, %v; want a clean end", ok, err)
	}
	// A non-empty Clear does invalidate.
	if err := dictionary.add("k", "v"); err != nil {
		t.Fatal(err)
	}
	live := dictionary.getEnumerator()
	dictionary.clear()
	if _, _, err := live.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("Next after a non-empty Clear = %v, want the version failure", err)
	}
}

// TestDictionaryBaseEnumeratorChecksVersionBeforeBounds pins the ORDER of
// MoveNext's two tests: the version comparison runs first, so a dictionary
// mutated after enumeration finished still fails a further step.
func TestDictionaryBaseEnumeratorChecksVersionBeforeBounds(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("k", "v"); err != nil {
		t.Fatal(err)
	}
	iterator := dictionary.getEnumerator()
	if _, ok, err := iterator.Next(); !ok || err != nil {
		t.Fatalf("first step = %v, %v", ok, err)
	}
	if _, ok, err := iterator.Next(); ok || err != nil {
		t.Fatalf("second step = %v, %v; want a clean end", ok, err)
	}
	if err := dictionary.add("later", "value"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := iterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("step past the end after a mutation = %v, want the version failure", err)
	}
}

func TestDictionaryBaseClearResetsTheFreeListAndCount(t *testing.T) {
	dictionary := newTestDictionary(t)
	for _, key := range []string{"a", "b", "c"} {
		if err := dictionary.add(key, key); err != nil {
			t.Fatal(err)
		}
	}
	dictionary.remove("b")
	dictionary.clear()
	if dictionary.count != 0 || dictionary.freeCount != 0 || dictionary.freeList != -1 {
		t.Fatalf("after Clear: count=%d freeCount=%d freeList=%d", dictionary.count, dictionary.freeCount, dictionary.freeList)
	}
	// Insertion after a Clear starts at slot 0 again.
	if err := dictionary.add("z", "z"); err != nil {
		t.Fatal(err)
	}
	if got := drainKeys(t, dictionary.getEnumerator()); !equalStrings(got, []string{"z"}) {
		t.Fatalf("enumeration after Clear = %v", got)
	}
}

func TestDictionaryBaseTryGetValueReturnsFoundFirst(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("k", "v"); err != nil {
		t.Fatal(err)
	}
	// The settled direction rule APPENDS the out parameter to the declared
	// return, so the bool comes first.
	if found, value := dictionary.tryGetValue("k"); !found || value != "v" {
		t.Fatalf("TryGetValue(k) = %v, %q", found, value)
	}
	if found, value := dictionary.tryGetValue("absent"); found || value != "" {
		t.Fatalf("TryGetValue(absent) = %v, %q; want false and default(string)", found, value)
	}
}

func TestDictionaryBaseContainsValueScansValuesNotKeys(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("key", "value"); err != nil {
		t.Fatal(err)
	}
	if !dictionary.containsValue("value") {
		t.Fatal("ContainsValue did not find a present value")
	}
	if dictionary.containsValue("key") {
		t.Fatal("ContainsValue matched a KEY, so it is scanning the wrong half")
	}
	// A freed slot must not be scanned: Remove clears both halves and marks the
	// hash negative.
	dictionary.remove("key")
	if dictionary.containsValue("value") {
		t.Fatal("ContainsValue found a value in a freed entry")
	}
}

func TestDictionaryBaseKeysAndValuesAreCachedLiveViews(t *testing.T) {
	dictionary := newTestDictionary(t)
	keys, values := dictionary.keysOf(), dictionary.valuesOf()
	// get_Keys caches, so the identity a caller observes never changes.
	if dictionary.keysOf() != keys || dictionary.valuesOf() != values {
		t.Fatal("get_Keys/get_Values allocated a second view")
	}
	if keys.Count() != 0 || values.Count() != 0 {
		t.Fatalf("empty views report %d/%d", keys.Count(), values.Count())
	}
	if err := dictionary.add("k", "v"); err != nil {
		t.Fatal(err)
	}
	// The views read THROUGH to the dictionary, so a later insertion is
	// visible through a view obtained before it.
	if keys.Count() != 1 || values.Count() != 1 {
		t.Fatalf("views are snapshots: %d/%d", keys.Count(), values.Count())
	}
	gotKey, ok, err := keys.GetEnumerator().Next()
	if err != nil || !ok || gotKey != "k" {
		t.Fatalf("key enumeration = %q, %v, %v", gotKey, ok, err)
	}
	gotValue, ok, err := values.GetEnumerator().Next()
	if err != nil || !ok || gotValue != "v" {
		t.Fatalf("value enumeration = %q, %v, %v", gotValue, ok, err)
	}
}

func TestDictionaryViewEnumeratorsFailFastAgainstTheDictionary(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("k", "v"); err != nil {
		t.Fatal(err)
	}
	keyIterator := dictionary.keysOf().GetEnumerator()
	valueIterator := dictionary.valuesOf().GetEnumerator()
	if err := dictionary.add("second", "value"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := keyIterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("key iterator = %v, want the version failure", err)
	}
	if _, _, err := valueIterator.Next(); !errors.Is(err, errCollectionEnumerationFailed) {
		t.Fatalf("value iterator = %v, want the version failure", err)
	}
}

func TestDictionaryViewCopyToChecksInReferenceOrder(t *testing.T) {
	dictionary := newTestDictionary(t)
	for _, key := range []string{"a", "b", "c"} {
		if err := dictionary.add(key, key+"!"); err != nil {
			t.Fatal(err)
		}
	}
	dictionary.remove("b")
	keys := dictionary.keysOf()

	if err := keys.CopyTo(nil, 0); !errors.Is(err, errCollectionArgumentNull) {
		t.Fatalf("CopyTo(nil) = %v, want ArgumentNullException", err)
	}
	if err := keys.CopyTo(make([]string, 4), -1); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("CopyTo(index -1) = %v, want ArgumentOutOfRangeException", err)
	}
	if err := keys.CopyTo(make([]string, 4), 5); !errors.Is(err, errCollectionArgumentOutOfRange) {
		t.Fatalf("CopyTo(index past the end) = %v, want ArgumentOutOfRangeException", err)
	}
	if err := keys.CopyTo(make([]string, 2), 1); !errors.Is(err, errCollectionArgument) {
		t.Fatalf("CopyTo into a short destination = %v, want ArgumentException", err)
	}

	// A successful copy writes only the LIVE entries, in entries order, at the
	// offset it was given, and leaves the rest of the destination alone.
	destination := []string{"x", "x", "x", "x"}
	if err := keys.CopyTo(destination, 1); err != nil {
		t.Fatalf("CopyTo: %v", err)
	}
	if !equalStrings(destination, []string{"x", "a", "c", "x"}) {
		t.Fatalf("CopyTo wrote %v", destination)
	}
	valueDestination := make([]string, 2)
	if err := dictionary.valuesOf().CopyTo(valueDestination, 0); err != nil {
		t.Fatalf("value CopyTo: %v", err)
	}
	if !equalStrings(valueDestination, []string{"a!", "c!"}) {
		t.Fatalf("value CopyTo wrote %v", valueDestination)
	}
}

func TestDictionaryBaseComparerIsTheCachedDefault(t *testing.T) {
	first, second := newTestDictionary(t), newTestDictionary(t)
	// EqualityComparer<string>.Default is a cached CLR static, so two
	// dictionaries hand back the same object.
	if first.comparerOf() != second.comparerOf() {
		t.Fatal("two dictionaries reported different default comparers")
	}
	comparer := first.comparerOf()
	if !comparer.Equals("a", "a") || comparer.Equals("a", "b") {
		t.Fatal("the default string comparer is not ordinal equality")
	}
	if comparer.GetHashCode("abc") != stringHashCode("abc") {
		t.Fatal("the default string comparer does not forward to String::GetHashCode")
	}
}

// TestStringHashCodeMatchesThePinnedAlgorithm pins values computed from the
// mscorlib IL listing rather than from this implementation, so a change to the
// Go body is caught rather than absorbed. The lengths are chosen to exercise
// every exit of the loop: the empty string skips it, 1 and 2 take the
// `len <= 2` break with and without a terminator read, 3 and 4 run the second
// accumulator once, and 5 iterates twice.
func TestStringHashCodeMatchesThePinnedAlgorithm(t *testing.T) {
	cases := []struct {
		value string
		want  int32
	}{
		{"", 757602046},
		{"a", -842352705},
		{"ab", -840386625},
		{"abc", 536991770},
		{"abcd", 1594742810},
		{"abcde", 398757997},
		{"Microsoft.Xna.Framework", 241716342},
		{"windowed", -484238069},
		// Non-ASCII is hashed as UTF-16 code units, not as UTF-8 bytes: a
		// byte-wise implementation disagrees on both of these.
		{"é", -842352825},
		{"\U0001F600", -2083885165},
	}
	for _, testCase := range cases {
		if got := stringHashCode(testCase.value); got != testCase.want {
			t.Fatalf("stringHashCode(%q) = %d, want %d", testCase.value, got, testCase.want)
		}
	}
}

func TestKeyValuePairProjectsBothHalvesAndTheReferenceToString(t *testing.T) {
	pair := NewKeyValuePair("k", "v")
	if pair.Key() != "k" || pair.Value() != "v" {
		t.Fatalf("pair = %q/%q", pair.Key(), pair.Value())
	}
	if got := pair.ToString(); got != "[k, v]" {
		t.Fatalf("ToString = %q", got)
	}
	// A null half contributes nothing, and the separator is emitted anyway.
	empty := NewKeyValuePair[any, any](nil, nil)
	if got := empty.ToString(); got != "[, ]" {
		t.Fatalf("null pair ToString = %q, want %q", got, "[, ]")
	}
}

func TestHashHelpersGetPrimeReproducesThePinnedTable(t *testing.T) {
	cases := []struct{ min, want int32 }{
		{0, 3}, {3, 3}, {4, 7}, {7, 7}, {8, 11}, {200, 239}, {7199369, 7199369},
	}
	for _, testCase := range cases {
		if got := hashHelpersGetPrime(testCase.min); got != testCase.want {
			t.Fatalf("GetPrime(%d) = %d, want %d", testCase.min, got, testCase.want)
		}
	}
	// Past the table, GetPrime falls back to the odd-candidate search.
	if got := hashHelpersGetPrime(7199370); got != 7199371 {
		t.Fatalf("GetPrime past the table = %d", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
