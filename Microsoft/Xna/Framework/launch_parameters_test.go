package framework

import (
	"errors"
	"os"
	"testing"
)

// newParsedLaunchParameters builds a LaunchParameters over a supplied argument
// vector rather than the process's own, so the parse can be measured without
// rewriting os.Args. It runs exactly the constructor's two statements.
func newParsedLaunchParameters(arguments []string) *LaunchParameters {
	parameters := &LaunchParameters{}
	parameters.base.init(defaultStringComparer, func(a, b string) bool { return a == b })
	parameters.parseCommandLineArguments(arguments)
	return parameters
}

func TestLaunchParametersParsesTheReferenceCommandLine(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		want      map[string]string
		order     []string
	}{
		{
			name:      "no arguments beyond the executable",
			arguments: []string{"game.exe"},
			want:      map[string]string{},
		},
		{
			name:      "the executable itself is never a parameter",
			arguments: []string{"/notakey:notavalue"},
			want:      map[string]string{},
		},
		{
			name:      "a slash-prefixed key/value pair",
			arguments: []string{"game.exe", "/windowed:true"},
			want:      map[string]string{"windowed": "true"},
			order:     []string{"windowed"},
		},
		{
			name:      "a dash prefix, and a run of prefixes, trim the same way",
			arguments: []string{"game.exe", "-a:1", "--b:2", "/-/c:3"},
			want:      map[string]string{"a": "1", "b": "2", "c": "3"},
			order:     []string{"a", "b", "c"},
		},
		{
			name:      "an argument with no colon is a key with an empty value",
			arguments: []string{"game.exe", "/fullscreen"},
			want:      map[string]string{"fullscreen": ""},
			order:     []string{"fullscreen"},
		},
		{
			name:      "the FIRST colon splits, so the value may contain more",
			arguments: []string{"game.exe", "/path:C:/games/x"},
			want:      map[string]string{"path": "C:/games/x"},
			order:     []string{"path"},
		},
		{
			name:      "a trailing colon yields an empty value",
			arguments: []string{"game.exe", "/key:"},
			want:      map[string]string{"key": ""},
			order:     []string{"key"},
		},
		{
			name:      "a duplicate key keeps the FIRST occurrence",
			arguments: []string{"game.exe", "/k:first", "/k:second"},
			want:      map[string]string{"k": "first"},
			order:     []string{"k"},
		},
		{
			name:      "an empty key is dropped, so separators alone add nothing",
			arguments: []string{"game.exe", "///", "-", ":value", "/real:1"},
			want:      map[string]string{"real": "1"},
			order:     []string{"real"},
		},
		{
			name:      "only LEADING separators are trimmed",
			arguments: []string{"game.exe", "/a-b:c/d"},
			want:      map[string]string{"a-b": "c/d"},
			order:     []string{"a-b"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			parameters := newParsedLaunchParameters(testCase.arguments)
			if int(parameters.Count()) != len(testCase.want) {
				t.Fatalf("Count = %d, want %d", parameters.Count(), len(testCase.want))
			}
			for key, want := range testCase.want {
				got, err := parameters.Item(key)
				if err != nil || got != want {
					t.Fatalf("Item(%q) = %q, %v; want %q", key, got, err, want)
				}
			}
			if testCase.order != nil {
				var got []string
				iterator := parameters.GetEnumerator()
				for {
					pair, ok, err := iterator.Next()
					if err != nil {
						t.Fatalf("enumeration: %v", err)
					}
					if !ok {
						break
					}
					got = append(got, pair.Key())
				}
				if !equalStrings(got, testCase.order) {
					t.Fatalf("enumeration order = %v, want %v", got, testCase.order)
				}
			}
		})
	}
}

func TestLaunchParametersReadsTheProcessCommandLine(t *testing.T) {
	// NewLaunchParameters is the constructor, and its argument source is
	// os.Args -- Go's spelling of Environment.GetCommandLineArgs(). The test
	// binary's own arguments are whatever `go test` passed, so the claim under
	// test is that the constructor agrees with parsing THOSE, not that any
	// particular key is present.
	constructed := NewLaunchParameters()
	parsed := newParsedLaunchParameters(os.Args)
	if constructed.Count() != parsed.Count() {
		t.Fatalf("NewLaunchParameters parsed %d entries, os.Args yields %d", constructed.Count(), parsed.Count())
	}
	iterator := parsed.GetEnumerator()
	for {
		pair, ok, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		got, err := constructed.Item(pair.Key())
		if err != nil || got != pair.Value() {
			t.Fatalf("constructed[%q] = %q, %v; want %q", pair.Key(), got, err, pair.Value())
		}
	}
}

func TestLaunchParametersInheritedSurfaceIsTheDictionarys(t *testing.T) {
	parameters := newParsedLaunchParameters([]string{"game.exe", "/a:1", "/b:2"})

	if _, err := parameters.Item("absent"); !errors.Is(err, errDictionaryKeyNotFound) {
		t.Fatalf("Item(absent) = %v, want KeyNotFoundException", err)
	}
	if err := parameters.Add("a", "other"); !errors.Is(err, errDictionaryDuplicateKey) {
		t.Fatalf("duplicate Add = %v, want ArgumentException", err)
	}
	if err := parameters.Add("c", "3"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	parameters.SetItem("a", "overwritten")
	if value, err := parameters.Item("a"); err != nil || value != "overwritten" {
		t.Fatalf("after SetItem, Item(a) = %q, %v", value, err)
	}
	if !parameters.ContainsKey("b") || parameters.ContainsKey("absent") {
		t.Fatal("ContainsKey disagrees with the contents")
	}
	if !parameters.ContainsValue("2") || parameters.ContainsValue("b") {
		t.Fatal("ContainsValue is scanning the wrong half")
	}
	if found, value := parameters.TryGetValue("b"); !found || value != "2" {
		t.Fatalf("TryGetValue(b) = %v, %q", found, value)
	}
	if found, value := parameters.TryGetValue("absent"); found || value != "" {
		t.Fatalf("TryGetValue(absent) = %v, %q", found, value)
	}
	if !parameters.Remove("b") || parameters.Remove("b") {
		t.Fatal("Remove did not report presence then absence")
	}
	if parameters.Count() != 2 {
		t.Fatalf("Count = %d, want 2", parameters.Count())
	}
	if parameters.Keys().Count() != 2 || parameters.Values().Count() != 2 {
		t.Fatalf("views report %d/%d", parameters.Keys().Count(), parameters.Values().Count())
	}
	if parameters.Keys() != parameters.Keys() {
		t.Fatal("Keys is not cached")
	}
	if parameters.Comparer() != defaultStringComparer {
		t.Fatal("Comparer is not EqualityComparer<string>.Default")
	}
	// OnDeserialization's reachable body is empty, so it changes nothing.
	before := parameters.Count()
	parameters.OnDeserialization(nil)
	parameters.OnDeserialization(parameters)
	if parameters.Count() != before {
		t.Fatal("OnDeserialization changed the dictionary")
	}
	parameters.Clear()
	if parameters.Count() != 0 {
		t.Fatalf("after Clear, Count = %d", parameters.Count())
	}
}

func TestGameLaunchParametersIsAStableMutableIdentity(t *testing.T) {
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	parameters := game.LaunchParameters()
	if parameters == nil {
		t.Fatal("Game.LaunchParameters is nil; the constructor did not allocate it")
	}
	// get_LaunchParameters is one ldfld, so the identity never changes.
	if game.LaunchParameters() != parameters {
		t.Fatal("Game.LaunchParameters returned a second object")
	}
	// XNA hands out the instance, not a copy or a view, so a consumer's
	// mutation is visible through the Game's own field.
	if err := parameters.Add("cna-go-test", "visible"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	value, err := game.LaunchParameters().Item("cna-go-test")
	if err != nil || value != "visible" {
		t.Fatalf("mutation through Game.LaunchParameters = %q, %v", value, err)
	}
	// Two Games have their own dictionaries.
	second, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if second.LaunchParameters() == parameters {
		t.Fatal("two Games share one LaunchParameters object")
	}
	if second.LaunchParameters().ContainsKey("cna-go-test") {
		t.Fatal("a second Game observed the first Game's mutation")
	}
}
