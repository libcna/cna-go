package framework

import "testing"

// This file is the falsifiability gate for Foundation 74.
//
// Each test below builds a deliberately WRONG version of one behaviour the
// projection claims, and asserts that the very assertion the real code passes
// rejects the mutant. A property nothing could fail is not evidence, and these
// four are the mutations most likely to be introduced by someone "simplifying"
// the adapter: a Go map instead of the entries array, last-wins duplicate
// handling, an enumerator that does not capture the version, and a
// LaunchParameters accessor that hands back a copy.

// TestMutantGoMapBackingIsRejectedByTheOrderAssertion kills the mutation the
// whole adapter exists to prevent.
//
// The real dictionary reuses the freed slot, so removing beta and adding
// epsilon yields alpha, epsilon, gamma, delta. Any structure that appends
// instead -- which is what a slice-or-map projection does -- yields
// alpha, gamma, delta, epsilon, and the pinned order rejects it.
func TestMutantGoMapBackingIsRejectedByTheOrderAssertion(t *testing.T) {
	const wanted = "alpha,epsilon,gamma,delta"

	real := newTestDictionary(t)
	appendOnly := newAppendOnlyMutant()
	for _, key := range []string{"alpha", "beta", "gamma", "delta"} {
		if err := real.add(key, key); err != nil {
			t.Fatal(err)
		}
		appendOnly.add(key)
	}
	real.remove("beta")
	appendOnly.remove("beta")
	if err := real.add("epsilon", "epsilon"); err != nil {
		t.Fatal(err)
	}
	appendOnly.add("epsilon")

	if got := joinStrings(drainKeys(t, real.getEnumerator())); got != wanted {
		t.Fatalf("the real dictionary enumerated %q, want %q", got, wanted)
	}
	if got := joinStrings(appendOnly.keys); got == wanted {
		t.Fatal("the append-only mutant survived the enumeration-order assertion")
	}
}

// TestMutantLastWinsDuplicateIsRejected kills "wrong Dictionary duplicate
// order": ParseCommandLineArguments tests ContainsKey BEFORE it adds, so the
// FIRST occurrence of a repeated key wins. A parser that assigned through the
// indexer instead would keep the last.
func TestMutantLastWinsDuplicateIsRejected(t *testing.T) {
	arguments := []string{"game.exe", "/k:first", "/k:second"}

	real := newParsedLaunchParameters(arguments)
	if value, err := real.Item("k"); err != nil || value != "first" {
		t.Fatalf("the real parser kept %q, want the FIRST occurrence", value)
	}

	// The mutant: `this[key] = value` instead of the guarded Add.
	mutant := &LaunchParameters{}
	mutant.base.init(defaultStringComparer, func(a, b string) bool { return a == b })
	for _, argument := range arguments[1:] {
		key, value := parseLaunchKeyValuePair(trimLaunchSeparators(argument))
		if key != "" {
			mutant.base.assign(key, value)
		}
	}
	if value, _ := mutant.Item("k"); value == "first" {
		t.Fatal("the last-wins mutant survived the duplicate-order assertion")
	}
}

// TestMutantEnumeratorWithoutAVersionIsRejected kills "wrong enumerator
// invalidation".
func TestMutantEnumeratorWithoutAVersionIsRejected(t *testing.T) {
	dictionary := newTestDictionary(t)
	if err := dictionary.add("present", "value"); err != nil {
		t.Fatal(err)
	}

	real := dictionary.getEnumerator()
	mutant := &unversionedIterator{dictionary: dictionary}
	if err := dictionary.add("later", "value"); err != nil {
		t.Fatal(err)
	}

	// One assertion, applied to both: "a step after a mutation reports the
	// enumeration failure". The real enumerator satisfies it; the mutant does
	// not, which is what makes the assertion evidence rather than decoration.
	if _, _, err := real.Next(); err == nil {
		t.Fatal("the real enumerator did not fail fast after a mutation")
	}
	if _, _, err := mutant.Next(); err == nil {
		return
	}
	t.Fatal("the unversioned mutant reported a failure, so the version assertion is not what kills it")
}

// TestMutantLaunchParametersCopyIsRejected kills "LaunchParameters returns a
// copy". XNA's getter is one ldfld, so a consumer's mutation reaches the Game.
func TestMutantLaunchParametersCopyIsRejected(t *testing.T) {
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if err := game.LaunchParameters().Add("planted", "value"); err != nil {
		t.Fatal(err)
	}
	if !game.launchParameters.ContainsKey("planted") {
		t.Fatal("the real accessor did not hand out the Game's own dictionary")
	}

	// The mutant: a getter that copies the entries into a fresh object.
	copyOf := func(source *LaunchParameters) *LaunchParameters {
		clone := &LaunchParameters{}
		clone.base.init(defaultStringComparer, func(a, b string) bool { return a == b })
		iterator := source.GetEnumerator()
		for {
			pair, ok, iterErr := iterator.Next()
			if iterErr != nil || !ok {
				return clone
			}
			clone.base.assign(pair.Key(), pair.Value())
		}
	}
	if err := copyOf(game.launchParameters).Add("only-in-the-copy", "value"); err != nil {
		t.Fatal(err)
	}
	if game.launchParameters.ContainsKey("only-in-the-copy") {
		t.Fatal("the copying mutant somehow reached the Game, so the assertion proves nothing")
	}
}

// trimLaunchSeparators is the mutant parser's copy of the TrimStart the real
// one performs, kept here so the mutation is the duplicate handling and
// nothing else.
func trimLaunchSeparators(argument string) string {
	for len(argument) > 0 && (argument[0] == '/' || argument[0] == '-') {
		argument = argument[1:]
	}
	return argument
}

// appendOnlyMutant models a projection whose removal compacts and whose
// insertion appends, which is what a slice or a Go map produces.
type appendOnlyMutant struct{ keys []string }

func newAppendOnlyMutant() *appendOnlyMutant { return &appendOnlyMutant{} }

func (m *appendOnlyMutant) add(key string) { m.keys = append(m.keys, key) }

func (m *appendOnlyMutant) remove(key string) {
	for i, existing := range m.keys {
		if existing == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			return
		}
	}
}

// unversionedIterator is dictionaryIterator with the version check removed.
type unversionedIterator struct {
	dictionary *dictionaryBase[string, string]
	index      int32
}

func (i *unversionedIterator) Next() (KeyValuePair[string, string], bool, error) {
	var zero KeyValuePair[string, string]
	for i.index < i.dictionary.count {
		entry := i.dictionary.entries[i.index]
		i.index++
		if entry.hashCode >= 0 {
			return NewKeyValuePair(entry.key, entry.value), true, nil
		}
	}
	return zero, false, nil
}

func joinStrings(values []string) string {
	result := ""
	for i, value := range values {
		if i > 0 {
			result += ","
		}
		result += value
	}
	return result
}
