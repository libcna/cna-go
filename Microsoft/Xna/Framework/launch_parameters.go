package framework

import (
	"os"
	"strings"
)

// LaunchParameters is XNA's parsed command line: the Dictionary<string,string>
// Game.LaunchParameters hands out, filled once at construction from the
// process's own arguments.
//
// # The base-class composition
//
// The CLR class is
//
//	.class public auto ansi beforefieldinit LaunchParameters
//	    extends [mscorlib]System.Collections.Generic.Dictionary`2<string,string>
//
// and it declares exactly one public member -- the parameterless constructor.
// Everything a caller uses is inherited public surface of
// Dictionary<string,string>, so the composition rule Foundation 26 settled
// applies in full: the base is the unexported `base` field holding the private
// dictionaryBase[string, string] adapter, and the inherited public members are
// re-exposed as measured forwarding methods. The adapter is never embedded,
// never exported and never returned, and the class is emphatically NOT
// projected as a Go map -- see bcl_dictionary_base.go for why enumeration
// order makes that not a stylistic choice.
//
// # What the constructor does
//
//	.ctor()
//	  call Dictionary`2<string,string>::.ctor()
//	  ParseCommandLineArguments(Environment.GetCommandLineArgs())
//
// The base constructor is `.ctor(0, null)`, which allocates nothing and takes
// EqualityComparer<string>.Default, so a LaunchParameters starts with a null
// bucket array and the ordinal string comparer.
//
// # ParseCommandLineArguments, exactly
//
// `assembly`, so it is not projected surface; it is the constructor's whole
// remaining body and its behaviour is entirely observable through the
// dictionary it fills:
//
//	separators = { '/', '-' }
//	if (args.Length <= 1) return;              // args[0] is the executable
//	for (i = 1; i < args.Length; i++) {
//	    argument = args[i].TrimStart(separators);
//	    ParseKeyValuePair(argument, out key, out value);
//	    if (!ContainsKey(key) && key != string.Empty)
//	        Add(key, value);
//	}
//
// and ParseKeyValuePair, which is private:
//
//	key = argument; value = string.Empty;
//	index = argument.IndexOf(':');
//	if (index != -1) {
//	    key   = argument.Substring(0, index);
//	    value = argument.Substring(index + 1);
//	}
//
// Four details are load-bearing and are reproduced rather than tidied:
//
//   - args[0] is skipped, because Environment.GetCommandLineArgs puts the
//     executable there;
//   - only LEADING '/' and '-' are trimmed, and TrimStart removes any run of
//     them, so `--x` and `/-/x` both yield the key `x`;
//   - the FIRST colon splits, so `a:b:c` is key `a` and value `b:c`, and a
//     trailing colon yields an empty value;
//   - a duplicate key keeps the FIRST occurrence, because ContainsKey is
//     tested before Add, and an empty key is dropped -- which is what happens
//     to an argument that is nothing but separators, or one beginning with a
//     colon.
type LaunchParameters struct {
	// base is the private Dictionary<string,string> adapter. It is
	// unexported, not embedded, and never handed out.
	base dictionaryBase[string, string]
}

// NewLaunchParameters projects LaunchParameters::.ctor, whose two statements
// are the base constructor and the command-line parse.
//
// The arguments are the running process's, read from os.Args, which is Go's
// spelling of System.Environment::GetCommandLineArgs(): both hand back the
// executable followed by the arguments, which is why both skip index 0.
func NewLaunchParameters() *LaunchParameters {
	parameters := &LaunchParameters{}
	parameters.base.init(defaultStringComparer, func(a, b string) bool { return a == b })
	parameters.parseCommandLineArguments(os.Args)
	return parameters
}

// parseCommandLineArguments is LaunchParameters::ParseCommandLineArguments. It
// is `assembly` in the reference, so it is not projected surface; it is
// unexported here for the same reason.
func (l *LaunchParameters) parseCommandLineArguments(arguments []string) {
	if len(arguments) <= 1 {
		return
	}
	for _, argument := range arguments[1:] {
		key, value := parseLaunchKeyValuePair(strings.TrimLeft(argument, "/-"))
		if !l.base.containsKey(key) && key != "" {
			// Add cannot fail here: the duplicate it refuses is exactly what
			// ContainsKey has just excluded, and the key is not empty.
			_ = l.base.add(key, value)
		}
	}
}

// parseLaunchKeyValuePair is LaunchParameters::ParseKeyValuePair, whose two
// `out` parameters the settled direction rule turns into two results.
func parseLaunchKeyValuePair(argument string) (key, value string) {
	index := strings.IndexByte(argument, ':')
	if index < 0 {
		return argument, ""
	}
	return argument[:index], argument[index+1:]
}

// ---------------------------------------------------------------------------
// The thirteen inherited public members of Dictionary<string,string>.
//
// Each forwards to the private adapter and nothing else. The fourteenth,
// GetObjectData, is a measured external-closure exclusion recorded in the
// adapter registry; see docs/foundation-74-dictionary-base-evidence.md.
// ---------------------------------------------------------------------------

// Comparer is Dictionary<string,string>::get_Comparer. It hands back
// EqualityComparer<string>.Default, which is a cached CLR static, so every
// LaunchParameters answers with the same object.
func (l *LaunchParameters) Comparer() IEqualityComparer[string] { return l.base.comparerOf() }

// Count is Dictionary<string,string>::get_Count.
func (l *LaunchParameters) Count() int32 { return l.base.countOf() }

// Keys is Dictionary<string,string>::get_Keys, a cached view over this
// dictionary: the same object every call, and live rather than a snapshot.
func (l *LaunchParameters) Keys() *DictionaryKeyCollection[string, string] {
	return l.base.keysOf()
}

// Values is Dictionary<string,string>::get_Values, cached the same way.
func (l *LaunchParameters) Values() *DictionaryValueCollection[string, string] {
	return l.base.valuesOf()
}

// Item is Dictionary<string,string>::get_Item. It is the ONE inherited read
// that fails: an absent key is KeyNotFoundException, not an empty string.
func (l *LaunchParameters) Item(key string) (string, error) { return l.base.item(key) }

// SetItem is Dictionary<string,string>::set_Item, which is Insert(..., false):
// it overwrites an existing key and adds a missing one, and cannot fail.
func (l *LaunchParameters) SetItem(key string, value string) { l.base.assign(key, value) }

// Add is Dictionary<string,string>::Add, which is Insert(..., true) and
// refuses a key that is already present.
func (l *LaunchParameters) Add(key string, value string) error { return l.base.add(key, value) }

// Clear is Dictionary<string,string>::Clear. Clearing an already empty
// dictionary is a no-op that does not even bump the version.
func (l *LaunchParameters) Clear() { l.base.clear() }

// ContainsKey is Dictionary<string,string>::ContainsKey.
func (l *LaunchParameters) ContainsKey(key string) bool { return l.base.containsKey(key) }

// ContainsValue is Dictionary<string,string>::ContainsValue, a forward scan of
// the live entries.
func (l *LaunchParameters) ContainsValue(value string) bool { return l.base.containsValue(value) }

// GetEnumerator is Dictionary<string,string>::GetEnumerator. Enumeration is in
// entries-array order, which is insertion order with removed slots reused, and
// it fails fast against the dictionary's version.
func (l *LaunchParameters) GetEnumerator() Iterator[KeyValuePair[string, string]] {
	return l.base.getEnumerator()
}

// Remove is Dictionary<string,string>::Remove, which reports whether the key
// was there.
func (l *LaunchParameters) Remove(key string) bool { return l.base.remove(key) }

// TryGetValue is Dictionary<string,string>::TryGetValue. The declared bool
// return comes first and the out parameter is appended after it, which is the
// settled direction rule rather than Go's value-then-ok idiom.
func (l *LaunchParameters) TryGetValue(key string) (bool, string) { return l.base.tryGetValue(key) }

// OnDeserialization is Dictionary<string,string>::OnDeserialization, whose
// reachable body here is empty; see the adapter for why m_siInfo can never be
// non-null in this projection.
func (l *LaunchParameters) OnDeserialization(sender any) { l.base.onDeserialization(sender) }
