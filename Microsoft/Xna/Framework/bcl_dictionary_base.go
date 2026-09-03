package framework

import "errors"

// This file is CNA-Go language support, not XNA surface.
//
// It is the private Go adapter for one supported BCL base class,
// System.Collections.Generic.Dictionary<TKey,TValue>, together with the two
// public nested BCL types that base's projected surface returns. The adapter
// itself is unexported, so it is not a projected XNA type, not a projected XNA
// member, not a public base-class object, and not a native handle; it is
// declared in tools/api_compat/mapping-rules.json under bclBaseAdapters and is
// measured there.
//
// # Why composition rather than embedding, and why not a Go map
//
// The general rule is Foundation 26's: Go has no CLR class inheritance, so a
// supported BCL base is held in an unexported field and its inherited public
// surface is re-exposed one measured member at a time. Dictionary adds a second
// refusal that matters more than it looks:
//
//	type LaunchParameters map[string]string   // NEVER
//
// A Go map is not Dictionary<K,V>. Dictionary enumerates its entries in the
// order of the entries ARRAY, which is insertion order with removed slots
// reused from a free list; Go deliberately randomises map iteration order.
// Enumeration order is observable through GetEnumerator, Keys and Values, so a
// map projection would be wrong at the first `foreach`, not merely
// unidiomatic. The adapter below therefore reproduces the reference's real
// structure -- buckets, entries, free list, version -- rather than a lookup
// table that happens to answer the same questions.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// Every pinned XNA assembly declares `.assembly extern mscorlib 4.0.0.0` with
// public key token b77a5c561934e089, which is the identity of that binary. It
// is the .NET Framework 4.0 RTM Dictionary: its Insert has no collision
// counter, no IsWellKnownEqualityComparer test and no randomised-comparer
// switch, so `comparer` is assigned once by the constructor and never changes.
//
// # The null-key guard is statically dead in this profile
//
// Insert, FindEntry and Remove each open with `box !TKey; brfalse;
// ThrowHelper.ThrowArgumentNullException`. The profile's only consumer is
// LaunchParameters, whose base is Dictionary<string,string>, and CNA-Go maps
// System.String to Go string, which has no null. That branch is therefore
// unreachable for every key any consumer can supply, exactly as
// Collection<T>'s `items.IsReadOnly` guard is unreachable for every consumer of
// that base, and it is not projected as a failure mode. A future consumer over
// a nullable key type would reopen this, which is why it is written down here
// rather than merely omitted.

// The exact .NET Framework 4.0 BCL messages the reference's two reachable throw
// sites load, read from the pinned mscorlib
// (sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63).
const (
	argKeyNotFound          = "The given key was not present in the dictionary."
	argumentAddingDuplicate = "An item with the same key has already been added."
)

// dictionaryEntry is Dictionary<TKey,TValue>.Entry: the private nested struct
// the entries array holds.
//
// hashCode is the comparer's hash masked with 0x7FFFFFFF while the slot is
// live, and -1 while it is on the free list, which is what makes a freed slot
// distinguishable from a live one with hash zero. next is the index of the next
// entry in the same bucket chain, or -1 at the end of a chain, and doubles as
// the free-list link while the slot is free.
type dictionaryEntry[TKey any, TValue any] struct {
	hashCode int32
	next     int32
	key      TKey
	value    TValue
}

// dictionaryBase is the private Go projection of
// System.Collections.Generic.Dictionary<TKey,TValue>.
//
// The field set is the reference's, name for name, because every one of them
// has observable consequences: buckets and entries fix enumeration order,
// freeList and freeCount decide which index a new entry takes after a removal,
// and version is what makes a live enumerator fail fast.
//
// The zero value is not usable: a consumer must call init so the comparer is
// wired before any operation runs.
type dictionaryBase[TKey any, TValue any] struct {
	// buckets is the bucket heads, one int32 per bucket, -1 when empty. It is
	// nil until the first insertion, exactly as the parameterless constructor
	// leaves it: `.ctor()` is `.ctor(0, null)`, and `.ctor(int32, comparer)`
	// calls Initialize only when capacity > 0.
	buckets []int32
	// entries is the entry array. Enumeration walks it from 0 to count-1 and
	// skips slots whose hashCode is negative, so its order IS the enumeration
	// order of the dictionary, of Keys and of Values.
	entries []dictionaryEntry[TKey, TValue]
	// count is how many entry slots have ever been handed out, live or freed;
	// Count is count minus freeCount.
	count int32
	// freeList is the head of the free-slot chain, -1 when empty.
	freeList int32
	// freeCount is how many slots are on that chain.
	freeCount int32
	// version projects Dictionary._version. Insert, Remove and a non-empty
	// Clear increment it; an EMPTY Clear does not, which is the opposite of
	// List<T>.Clear and is reproduced rather than harmonised.
	version int32
	// comparer is the IEqualityComparer<TKey> the constructor stored. In this
	// profile it is always EqualityComparer<TKey>.Default, because the only
	// constructor a consumer reaches is the parameterless one.
	comparer IEqualityComparer[TKey]
	// valueEqual projects EqualityComparer<TValue>.Default, which ONE member
	// uses: ContainsValue compares values rather than keys. It is supplied per
	// consumer for the reason the collection base's element comparer is --
	// which comparer the BCL selects depends on the type argument.
	valueEqual func(a, b TValue) bool
	// keys and values are Dictionary's own two cached views. The reference
	// creates each at most once and returns the same object from then on, so
	// the identity a caller observes never changes, and that is reproduced.
	keys   *DictionaryKeyCollection[TKey, TValue]
	values *DictionaryValueCollection[TKey, TValue]
}

// Unexported sentinel errors projecting the exact CLR exceptions the pinned
// Dictionary<K,V> IL throws. They are unexported for the reason the collection
// base's are: the XNA public contract declares no error type for a dictionary
// operation, and telling one CLR exception from another would be the
// System.Exception public mapping decision.
var (
	// errDictionaryKeyNotFound projects System.Collections.Generic
	// .KeyNotFoundException, thrown by ThrowHelper.ThrowKeyNotFoundException
	// from get_Item. Its message is the pinned mscorlib's Arg_KeyNotFound,
	// verified byte for byte by tools/resource_strings.
	errDictionaryKeyNotFound = errors.New(argKeyNotFound)
	// errDictionaryDuplicateKey projects the System.ArgumentException
	// ThrowHelper.ThrowArgumentException(ExceptionResource.Argument_AddingDuplicate)
	// raises from Insert when add is true.
	errDictionaryDuplicateKey = errors.New(argumentAddingDuplicate)
)

func (d *dictionaryBase[TKey, TValue]) init(comparer IEqualityComparer[TKey], valueEqual func(a, b TValue) bool) {
	d.freeList = -1
	d.comparer = comparer
	d.valueEqual = valueEqual
}

// ---------------------------------------------------------------------------
// The private machinery: Initialize, FindEntry, Insert and Resize.
// ---------------------------------------------------------------------------

// initialize is Dictionary<K,V>::Initialize: take the next prime at or above
// capacity, allocate that many buckets set to -1 and that many entries, and
// reset the free list.
func (d *dictionaryBase[TKey, TValue]) initialize(capacity int32) {
	size := hashHelpersGetPrime(capacity)
	d.buckets = make([]int32, size)
	for i := range d.buckets {
		d.buckets[i] = -1
	}
	d.entries = make([]dictionaryEntry[TKey, TValue], size)
	d.freeList = -1
}

// findEntry is Dictionary<K,V>::FindEntry: walk the bucket chain comparing the
// stored hash first and only then the comparer, and return the entry index or
// -1.
//
// The hash test before the Equals call is not an optimisation that could be
// dropped: it is what the reference does, so a comparer whose Equals and
// GetHashCode disagree finds nothing here exactly as it finds nothing there.
func (d *dictionaryBase[TKey, TValue]) findEntry(key TKey) int32 {
	if d.buckets == nil {
		return -1
	}
	hashCode := d.comparer.GetHashCode(key) & 0x7FFFFFFF
	for i := d.buckets[int(hashCode)%len(d.buckets)]; i >= 0; i = d.entries[i].next {
		if d.entries[i].hashCode == hashCode && d.comparer.Equals(d.entries[i].key, key) {
			return i
		}
	}
	return -1
}

// insert is Dictionary<K,V>::Insert(key, value, add), the one body behind both
// Add and the indexer setter.
//
// An existing key either fails (add, which is Add) or overwrites and bumps the
// version (not add, which is set_Item). A new key takes the head of the free
// list when one exists and otherwise the next unused slot, resizing first when
// the entry array is full.
func (d *dictionaryBase[TKey, TValue]) insert(key TKey, value TValue, add bool) error {
	if d.buckets == nil {
		d.initialize(0)
	}
	hashCode := d.comparer.GetHashCode(key) & 0x7FFFFFFF
	targetBucket := int(hashCode) % len(d.buckets)
	for i := d.buckets[targetBucket]; i >= 0; i = d.entries[i].next {
		if d.entries[i].hashCode != hashCode || !d.comparer.Equals(d.entries[i].key, key) {
			continue
		}
		if add {
			return errDictionaryDuplicateKey
		}
		d.entries[i].value = value
		d.version++
		return nil
	}
	var index int32
	if d.freeCount > 0 {
		index = d.freeList
		d.freeList = d.entries[index].next
		d.freeCount--
	} else {
		if d.count == int32(len(d.entries)) {
			d.resize()
			targetBucket = int(hashCode) % len(d.buckets)
		}
		index = d.count
		d.count++
	}
	d.entries[index].hashCode = hashCode
	d.entries[index].next = d.buckets[targetBucket]
	d.entries[index].key = key
	d.entries[index].value = value
	d.buckets[targetBucket] = index
	d.version++
	return nil
}

// resize is Dictionary<K,V>::Resize: a fresh prime-sized bucket and entry pair,
// with the live entries copied across at the SAME indices and rehashed.
//
// Copying at the same indices is why a resize does not disturb enumeration
// order, and it is also why the rehash loop can walk 0..count-1 without
// checking for freed slots: Insert only reaches Resize when freeCount is zero,
// so every slot below count is live and no entry carries the -1 hash that would
// index a bucket out of range.
func (d *dictionaryBase[TKey, TValue]) resize() {
	newSize := hashHelpersGetPrime(d.count * 2)
	buckets := make([]int32, newSize)
	for i := range buckets {
		buckets[i] = -1
	}
	entries := make([]dictionaryEntry[TKey, TValue], newSize)
	copy(entries, d.entries[:d.count])
	for i := int32(0); i < d.count; i++ {
		bucket := int(entries[i].hashCode) % int(newSize)
		entries[i].next = buckets[bucket]
		buckets[bucket] = i
	}
	d.buckets = buckets
	d.entries = entries
}

// ---------------------------------------------------------------------------
// The inherited public surface.
//
// Thirteen of Dictionary<K,V>'s fourteen public members are here. The
// fourteenth, GetObjectData, is recorded as a measured external-closure
// exclusion in the adapter registry: its two parameter types reach
// System.Runtime.Serialization, and reproducing SerializationInfo's pinned
// public inventory would drag System.Decimal and System.DateTime in behind it,
// none of which the XNA 4.0 Windows profile names anywhere else.
//
// Constructors are not here because the CLR does not inherit them; the private
// helpers above are not public; and everything else Dictionary declares is a
// private explicit implementation of IDictionary, IDictionary<K,V>,
// ICollection, ICollection<KVP> or IEnumerable, which the settled
// BCL-interface rule already excludes.
// ---------------------------------------------------------------------------

// comparerOf is Dictionary<K,V>::get_Comparer, one ldfld of the comparer the
// constructor stored.
func (d *dictionaryBase[TKey, TValue]) comparerOf() IEqualityComparer[TKey] { return d.comparer }

// countOf is Dictionary<K,V>::get_Count, which is `count - freeCount` and not
// the length of anything.
func (d *dictionaryBase[TKey, TValue]) countOf() int32 { return d.count - d.freeCount }

// item is Dictionary<K,V>::get_Item: FindEntry, then either the stored value or
// KeyNotFoundException. It is the ONE inherited read that fails.
func (d *dictionaryBase[TKey, TValue]) item(key TKey) (TValue, error) {
	if i := d.findEntry(key); i >= 0 {
		return d.entries[i].value, nil
	}
	var zero TValue
	return zero, errDictionaryKeyNotFound
}

// assign is Dictionary<K,V>::set_Item, which is `Insert(key, value, false)`:
// it adds a missing key rather than refusing, and cannot fail.
func (d *dictionaryBase[TKey, TValue]) assign(key TKey, value TValue) {
	_ = d.insert(key, value, false)
}

// add is Dictionary<K,V>::Add, which is `Insert(key, value, true)` and refuses
// a key that is already present.
func (d *dictionaryBase[TKey, TValue]) add(key TKey, value TValue) error {
	return d.insert(key, value, true)
}

// clear is Dictionary<K,V>::Clear.
//
// The whole body is guarded by `count > 0`, so clearing an ALREADY EMPTY
// dictionary changes nothing and does NOT increment the version -- a live
// enumerator survives it. List<T>.Clear falls through to `_version++`
// unconditionally and invalidates one. The two collections disagree, and each
// is projected as it is.
func (d *dictionaryBase[TKey, TValue]) clear() {
	if d.count <= 0 {
		return
	}
	for i := range d.buckets {
		d.buckets[i] = -1
	}
	var zero dictionaryEntry[TKey, TValue]
	for i := int32(0); i < d.count; i++ {
		d.entries[i] = zero
	}
	d.freeList = -1
	d.count = 0
	d.freeCount = 0
	d.version++
}

// containsKey is Dictionary<K,V>::ContainsKey, `FindEntry(key) >= 0`.
func (d *dictionaryBase[TKey, TValue]) containsKey(key TKey) bool { return d.findEntry(key) >= 0 }

// containsValue is Dictionary<K,V>::ContainsValue: a forward scan of the live
// entries using EqualityComparer<TValue>.Default. It is the one member whose
// comparer is the VALUE's rather than the key's, which is why the equality is
// supplied separately by the consumer.
func (d *dictionaryBase[TKey, TValue]) containsValue(value TValue) bool {
	for i := int32(0); i < d.count; i++ {
		if d.entries[i].hashCode >= 0 && d.valueEqual(d.entries[i].value, value) {
			return true
		}
	}
	return false
}

// remove is Dictionary<K,V>::Remove: unlink the entry from its bucket chain,
// mark it free, clear both halves so neither is retained, and push it onto the
// free list. It returns false without touching anything when the key is absent.
//
// The freed slot goes to the HEAD of the free list, so the next insertion
// reuses the most recently removed index. That is observable through
// enumeration order and is the reason a Go map could not stand in.
func (d *dictionaryBase[TKey, TValue]) remove(key TKey) bool {
	if d.buckets == nil {
		return false
	}
	hashCode := d.comparer.GetHashCode(key) & 0x7FFFFFFF
	bucket := int(hashCode) % len(d.buckets)
	last := int32(-1)
	for i := d.buckets[bucket]; i >= 0; i = d.entries[i].next {
		if d.entries[i].hashCode != hashCode || !d.comparer.Equals(d.entries[i].key, key) {
			last = i
			continue
		}
		if last < 0 {
			d.buckets[bucket] = d.entries[i].next
		} else {
			d.entries[last].next = d.entries[i].next
		}
		var zeroKey TKey
		var zeroValue TValue
		d.entries[i].hashCode = -1
		d.entries[i].next = d.freeList
		d.entries[i].key = zeroKey
		d.entries[i].value = zeroValue
		d.freeList = i
		d.freeCount++
		d.version++
		return true
	}
	return false
}

// tryGetValue is Dictionary<K,V>::TryGetValue.
//
// The settled direction rule REMOVES an out parameter from the inputs and
// APPENDS it to the results, so the declared bool return comes first and the
// value second. This is the profile's first member with both a non-void return
// and an out parameter -- every earlier one returns void, where appending is
// indistinguishable from prepending -- so the order is the rule applied rather
// than a new choice, and it is deliberately not reshuffled into Go's
// value-then-ok idiom, which would make one member disagree with the rule every
// other member follows.
//
// A miss yields default(TValue), which the reference writes explicitly with
// initobj rather than leaving the caller's variable alone.
func (d *dictionaryBase[TKey, TValue]) tryGetValue(key TKey) (bool, TValue) {
	if i := d.findEntry(key); i >= 0 {
		return true, d.entries[i].value
	}
	var zero TValue
	return false, zero
}

// keysOf is Dictionary<K,V>::get_Keys, which allocates the view once, caches
// it, and returns the same object forever after.
func (d *dictionaryBase[TKey, TValue]) keysOf() *DictionaryKeyCollection[TKey, TValue] {
	if d.keys == nil {
		d.keys = &DictionaryKeyCollection[TKey, TValue]{dictionary: d}
	}
	return d.keys
}

// valuesOf is Dictionary<K,V>::get_Values, cached the same way.
func (d *dictionaryBase[TKey, TValue]) valuesOf() *DictionaryValueCollection[TKey, TValue] {
	if d.values == nil {
		d.values = &DictionaryValueCollection[TKey, TValue]{dictionary: d}
	}
	return d.values
}

// getEnumerator is Dictionary<K,V>::GetEnumerator.
//
// The reference returns the nested Dictionary<K,V>.Enumerator STRUCT, not a
// boxed IEnumerator<KeyValuePair<K,V>>. The settled rule for that shape is the
// one Foundation 72 wrote down for List<T>.Enumerator: the nested struct IS an
// IEnumerator<T>, and IEnumerator<T> is Iterator[T], so the projection is
// Iterator[KeyValuePair[TKey, TValue]]. The struct copy semantics a C# caller
// can use -- taking an independent cursor by assigning the enumerator -- are
// not expressible through a Go interface and are recorded as a language
// limitation rather than reinvented.
func (d *dictionaryBase[TKey, TValue]) getEnumerator() Iterator[KeyValuePair[TKey, TValue]] {
	return &dictionaryIterator[TKey, TValue]{dictionary: d, version: d.version}
}

// onDeserialization is Dictionary<K,V>::OnDeserialization, whose entire body is
//
//	if (m_siInfo == null) return;
//	... rebuild from the SerializationInfo ...
//
// and whose first branch is the only reachable one here. m_siInfo has exactly
// one non-null writer in the pinned IL: the `family` constructor
// .ctor(SerializationInfo, StreamingContext). The CLR does not inherit
// constructors, LaunchParameters declares no serialization constructor of its
// own, and CNA-Go deserialises nothing, so no state this adapter can reach has
// m_siInfo set. The member is therefore projected as the empty body it
// provably is, rather than omitted or faked, and it cannot fail.
//
// The sender parameter is System.Object, which the reference reads not once.
func (d *dictionaryBase[TKey, TValue]) onDeserialization(sender any) { _ = sender }

// ---------------------------------------------------------------------------
// Enumeration.
// ---------------------------------------------------------------------------

// dictionaryIterator projects Dictionary<K,V>.Enumerator onto Iterator[T].
//
// MoveNext compares the captured version FIRST, before the bounds test, so a
// dictionary mutated after enumeration has already run off the end still fails
// a further step. It then advances past freed slots -- hashCode < 0 -- and
// yields a KeyValuePair built from the live entry.
//
// The enumerator's default(T) states before the first MoveNext and after the
// last are not representable through Iterator[T], which fuses MoveNext and
// Current into one Next, so neither is invented; and IEnumerator.Reset is a
// private explicit implementation, which the settled BCL-interface rule already
// excludes.
type dictionaryIterator[TKey any, TValue any] struct {
	dictionary *dictionaryBase[TKey, TValue]
	version    int32
	index      int32
}

func (i *dictionaryIterator[TKey, TValue]) Next() (KeyValuePair[TKey, TValue], bool, error) {
	var zero KeyValuePair[TKey, TValue]
	if i.version != i.dictionary.version {
		return zero, false, errCollectionEnumerationFailed
	}
	for i.index < i.dictionary.count {
		entry := i.dictionary.entries[i.index]
		i.index++
		if entry.hashCode >= 0 {
			return NewKeyValuePair(entry.key, entry.value), true, nil
		}
	}
	i.index = i.dictionary.count + 1
	return zero, false, nil
}

// dictionaryKeyIterator and dictionaryValueIterator project
// KeyCollection.Enumerator and ValueCollection.Enumerator, which are the same
// cursor over the same entries array yielding one half of each live entry.
type dictionaryKeyIterator[TKey any, TValue any] struct {
	dictionary *dictionaryBase[TKey, TValue]
	version    int32
	index      int32
}

func (i *dictionaryKeyIterator[TKey, TValue]) Next() (TKey, bool, error) {
	var zero TKey
	if i.version != i.dictionary.version {
		return zero, false, errCollectionEnumerationFailed
	}
	for i.index < i.dictionary.count {
		entry := i.dictionary.entries[i.index]
		i.index++
		if entry.hashCode >= 0 {
			return entry.key, true, nil
		}
	}
	i.index = i.dictionary.count + 1
	return zero, false, nil
}

type dictionaryValueIterator[TKey any, TValue any] struct {
	dictionary *dictionaryBase[TKey, TValue]
	version    int32
	index      int32
}

func (i *dictionaryValueIterator[TKey, TValue]) Next() (TValue, bool, error) {
	var zero TValue
	if i.version != i.dictionary.version {
		return zero, false, errCollectionEnumerationFailed
	}
	for i.index < i.dictionary.count {
		entry := i.dictionary.entries[i.index]
		i.index++
		if entry.hashCode >= 0 {
			return entry.value, true, nil
		}
	}
	i.index = i.dictionary.count + 1
	return zero, false, nil
}

// ---------------------------------------------------------------------------
// The two public nested views.
// ---------------------------------------------------------------------------

// DictionaryKeyCollection is the projection of
// System.Collections.Generic.Dictionary<TKey,TValue>.KeyCollection, which the
// pinned contract carries at a signature position because
// Dictionary<K,V>::get_Keys returns it. It is a declared BCL signature adapter
// and adds no XNA identity of its own; the flattened Go name is the settled
// nested-name rule, declaring type concatenated with nested type.
//
// It is a VIEW, not a copy: it holds the dictionary it was made from and reads
// through to it, so every later insertion or removal is visible through an
// already-obtained Keys.
//
// Its public CLR surface is exactly Count, CopyTo and GetEnumerator.
// Everything else it declares is a private explicit implementation of
// ICollection<TKey>, ICollection or IEnumerable -- including every mutator,
// which is what makes the view read-only without a new decision.
type DictionaryKeyCollection[TKey any, TValue any] struct {
	dictionary *dictionaryBase[TKey, TValue]
}

// Count is KeyCollection::get_Count, one forwarded Dictionary::get_Count.
func (c *DictionaryKeyCollection[TKey, TValue]) Count() int32 { return c.dictionary.countOf() }

// CopyTo is KeyCollection::CopyTo, whose three failures are its own and are
// checked in this order: a null destination, an index outside
// [0, array.Length], and a destination whose remaining room is smaller than
// Count. Note that `index > array.Length` is admitted by the second test and
// caught by the third only when the dictionary is non-empty, which is the
// reference's own asymmetry.
func (c *DictionaryKeyCollection[TKey, TValue]) CopyTo(array []TKey, index int32) error {
	if array == nil {
		return collectionNullError("array")
	}
	if index < 0 || index > int32(len(array)) {
		return collectionIndexError("index")
	}
	if int32(len(array))-index < c.dictionary.countOf() {
		return collectionArgumentError("destination array is not long enough")
	}
	for i := int32(0); i < c.dictionary.count; i++ {
		if c.dictionary.entries[i].hashCode >= 0 {
			array[index] = c.dictionary.entries[i].key
			index++
		}
	}
	return nil
}

// GetEnumerator is KeyCollection::GetEnumerator, projected through the same
// settled nested-enumerator rule Dictionary's own GetEnumerator uses.
func (c *DictionaryKeyCollection[TKey, TValue]) GetEnumerator() Iterator[TKey] {
	return &dictionaryKeyIterator[TKey, TValue]{dictionary: c.dictionary, version: c.dictionary.version}
}

// DictionaryValueCollection is the projection of
// System.Collections.Generic.Dictionary<TKey,TValue>.ValueCollection. It is
// KeyCollection's mirror in every respect, over the other half of each entry.
type DictionaryValueCollection[TKey any, TValue any] struct {
	dictionary *dictionaryBase[TKey, TValue]
}

// Count is ValueCollection::get_Count.
func (c *DictionaryValueCollection[TKey, TValue]) Count() int32 { return c.dictionary.countOf() }

// CopyTo is ValueCollection::CopyTo, with KeyCollection::CopyTo's three checks
// in the same order.
func (c *DictionaryValueCollection[TKey, TValue]) CopyTo(array []TValue, index int32) error {
	if array == nil {
		return collectionNullError("array")
	}
	if index < 0 || index > int32(len(array)) {
		return collectionIndexError("index")
	}
	if int32(len(array))-index < c.dictionary.countOf() {
		return collectionArgumentError("destination array is not long enough")
	}
	for i := int32(0); i < c.dictionary.count; i++ {
		if c.dictionary.entries[i].hashCode >= 0 {
			array[index] = c.dictionary.entries[i].value
			index++
		}
	}
	return nil
}

// GetEnumerator is ValueCollection::GetEnumerator.
func (c *DictionaryValueCollection[TKey, TValue]) GetEnumerator() Iterator[TValue] {
	return &dictionaryValueIterator[TKey, TValue]{dictionary: c.dictionary, version: c.dictionary.version}
}

// ---------------------------------------------------------------------------
// System.Collections.HashHelpers.
// ---------------------------------------------------------------------------

// hashHelpersPrimes is System.Collections.HashHelpers::primes, the 72-entry
// int32 table the pinned mscorlib initialises from a 288-byte static blob at
// I_0042E144. It is reproduced from that blob rather than regenerated, because
// "the primes .NET uses" is a fact about this binary and not a derivable one.
var hashHelpersPrimes = [...]int32{
	3, 7, 11, 17, 23, 29, 37, 47, 59, 71, 89, 107, 131, 163, 197, 239, 293, 353,
	431, 521, 631, 761, 919, 1103, 1327, 1597, 1931, 2333, 2801, 3371, 4049,
	4861, 5839, 7013, 8419, 10103, 12143, 14591, 17519, 21023, 25229, 30293,
	36353, 43627, 52361, 62851, 75431, 90523, 108631, 130363, 156437, 187751,
	225307, 270371, 324449, 389357, 467237, 560689, 672827, 807403, 968897,
	1162687, 1395263, 1674319, 2009191, 2411033, 2893249, 3471899, 4166287,
	4999559, 5999471, 7199369,
}

// hashHelpersGetPrime is HashHelpers::GetPrime: the first table entry at or
// above min, or -- past the end of the table -- the first odd candidate at or
// above min that IsPrime accepts.
//
// Nothing public observes which prime is chosen: capacity is not part of
// Dictionary's public surface, and enumeration order is the entries array's
// index order regardless of how many buckets there are. It is reproduced
// anyway, because the bucket count is the modulus every lookup divides by, and
// a table that merely resembled this one would make the structure resemble
// rather than reproduce the reference's.
func hashHelpersGetPrime(min int32) int32 {
	for _, prime := range hashHelpersPrimes {
		if prime >= min {
			return prime
		}
	}
	for candidate := min | 1; candidate < 0x7FFFFFFF; candidate += 2 {
		if hashHelpersIsPrime(candidate) {
			return candidate
		}
	}
	return min
}

// hashHelpersIsPrime is HashHelpers::IsPrime: even numbers are prime only when
// they are 2, and an odd candidate is tested against odd divisors up to its
// square root.
func hashHelpersIsPrime(candidate int32) bool {
	if candidate&1 == 0 {
		return candidate == 2
	}
	limit := int32(0)
	for limit*limit <= candidate {
		limit++
	}
	limit--
	for divisor := int32(3); divisor <= limit; divisor += 2 {
		if candidate%divisor == 0 {
			return false
		}
	}
	return true
}
