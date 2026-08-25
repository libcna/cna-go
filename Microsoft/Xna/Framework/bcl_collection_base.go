package framework

import (
	"errors"
	"fmt"
	"reflect"
)

// This file is CNA-Go language support, not XNA surface.
//
// It is the private Go adapter for one supported BCL base class,
// System.Collections.ObjectModel.Collection<T>. Nothing declared here is
// exported, so nothing here is a projected XNA type, a projected XNA member, a
// public base-class object, or a native handle. The adapter is declared in
// tools/api_compat/mapping-rules.json under bclBaseAdapters and is measured
// there; it adds no XNA identity of its own.
//
// # Why composition rather than embedding
//
// Go has no CLR class inheritance. Simulating it with an exported embedded
// field
//
//	type GameComponentCollection struct {
//	    Collection[IGameComponent]     // NEVER
//	}
//
// would publish a Go type the XNA contract never declares, promote its whole
// method set onto the derived type, make the derived type assignable and
// type-assertable in ways CLR inheritance does not imply, and expose the
// support implementation as public API. CNA-Go therefore holds the base as an
// unexported field and forwards the inherited public surface explicitly, one
// measured member at a time.
//
// # Reference authority
//
// The behavior below is read from the exact .NET Framework 4.0 BCL that the
// pinned XNA assemblies bind against, not from modern .NET and not from Go
// convention:
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// Every pinned XNA assembly declares `.assembly extern mscorlib 4.0.0.0` with
// public key token b77a5c561934e089, which is the identity of that binary.
//
// Collection<T> stores its elements in an IList<T> field named `items`. The
// parameterless constructor -- the only one any pinned XNA subclass calls --
// assigns `new List<T>()`, so the backing store is always List<T> and its
// IsReadOnly is always false. That is why the `items.IsReadOnly` guard that
// opens Add, Clear, Insert, Remove, RemoveAt and set_Item is statically dead
// for every XNA consumer and is not projected as a failure mode.
//
// Collection<T> itself validates only the index, and it validates it
// differently per operation. List<T> then validates again and owns the
// version counter that invalidates enumerators.

// Unexported sentinel errors projecting the exact CLR exceptions the pinned
// Collection<T> and List<T> IL throws. They are unexported because the XNA
// public contract declares no error type for a collection operation, and
// because giving a consumer a way to tell one CLR exception from another would
// be the System.Exception public mapping decision, which remains open. A
// consumer can tell success from failure, which is what the contract needs.
var (
	// errCollectionArgumentOutOfRange projects
	// System.ArgumentOutOfRangeException, thrown by
	// System.ThrowHelper::ThrowArgumentOutOfRangeException.
	errCollectionArgumentOutOfRange = errors.New("collection index is out of range")
	// errCollectionArgumentNull projects System.ArgumentNullException.
	errCollectionArgumentNull = errors.New("collection argument is nil")
	// errCollectionArgument projects System.ArgumentException.
	errCollectionArgument = errors.New("collection argument is invalid")
	// errCollectionNotSupported projects System.NotSupportedException.
	errCollectionNotSupported = errors.New("collection operation is not supported")
	// errCollectionEnumerationFailed projects the
	// System.InvalidOperationException that List<T>.Enumerator.MoveNextRare
	// throws when the collection changed after the enumerator was taken.
	errCollectionEnumerationFailed = errors.New("collection changed during enumeration")
)

func collectionIndexError(parameter string) error {
	return fmt.Errorf("%w: %s", errCollectionArgumentOutOfRange, parameter)
}

func collectionNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errCollectionArgumentNull, parameter)
}

func collectionArgumentError(message string) error {
	return fmt.Errorf("%w: %s", errCollectionArgument, message)
}

func collectionNotSupportedError(message string) error {
	return fmt.Errorf("%w: %s", errCollectionNotSupported, message)
}

// collectionOverrides is the private equivalent of the four protected virtual
// members a CLR subclass of Collection<T> may override.
//
// The methods are unexported, so only a type declared in this package can
// satisfy the interface: a consumer outside CNA-Go can neither implement the
// hooks nor call them, exactly as a consumer of a sealed CLR class cannot
// reach its protected members. They are also not projected XNA surface --
// GameComponentCollection's four overrides are `family` in the reference
// metadata and the class is sealed, so no CLR caller can reach them either.
//
// Every mutating public operation routes through this interface, so a subclass
// hook always runs. Nothing appends to the backing store directly and then
// fakes the subclass's observable effect separately.
type collectionOverrides[T any] interface {
	insertItem(index int32, item T) error
	removeItem(index int32) error
	setItem(index int32, item T) error
	clearItems() error
}

// collectionBase is the private Go projection of
// System.Collections.ObjectModel.Collection<T> over its default List<T>
// backing store.
//
// The zero value is not usable: a consumer must call init so the hook
// dispatcher and the element equality projection are wired before any
// operation runs.
type collectionBase[T any] struct {
	// items is the List<T> backing store. It is never returned, never
	// aliased into a caller's hands, and never exposed by any projected
	// member, so no caller can mutate the store without passing through a
	// subclass hook.
	items []T
	// version projects List<T>._version. Every List<T> mutator increments it
	// and every enumerator captures it, which is what makes enumeration
	// fail fast after a change.
	version uint64
	// overrides is the subclass whose hooks run in place of the base
	// implementations. It is never nil after init.
	overrides collectionOverrides[T]
	// equal projects EqualityComparer<T>.Default for this element type. It
	// is supplied per consumer rather than assumed, because the comparer the
	// BCL selects depends on T.
	equal func(a, b T) bool
}

func (c *collectionBase[T]) init(overrides collectionOverrides[T], equal func(a, b T) bool) {
	c.overrides = overrides
	c.equal = equal
}

// ---------------------------------------------------------------------------
// The base implementations of the four protected virtual hooks.
//
// These are what `base.InsertItem(...)` reaches in a CLR override. Each is one
// List<T> mutation and the version increment that mutation performs.
// ---------------------------------------------------------------------------

// baseInsertItem is Collection<T>::InsertItem, which is `items.Insert(index,
// item)`. The index is already validated by the caller that reached the hook.
func (c *collectionBase[T]) baseInsertItem(index int32, item T) {
	position := int(index)
	var zero T
	c.items = append(c.items, zero)
	copy(c.items[position+1:], c.items[position:])
	c.items[position] = item
	c.version++
}

// baseRemoveItem is Collection<T>::RemoveItem, which is
// `items.RemoveAt(index)`. List<T>.RemoveAt clears the vacated slot so the
// removed element is not retained, which matters because T is a reference type
// for every pinned consumer.
func (c *collectionBase[T]) baseRemoveItem(index int32) {
	position := int(index)
	var zero T
	copy(c.items[position:], c.items[position+1:])
	c.items[len(c.items)-1] = zero
	c.items = c.items[:len(c.items)-1]
	c.version++
}

// baseSetItem is Collection<T>::SetItem, which is `items[index] = value`.
func (c *collectionBase[T]) baseSetItem(index int32, item T) {
	c.items[int(index)] = item
	c.version++
}

// baseClearItems is Collection<T>::ClearItems, which is `items.Clear()`.
//
// List<T>.Clear increments the version unconditionally: its IL skips only the
// Array.Clear and the `_size = 0` store when the list is already empty, and
// falls through to `_version++` in every case. Clearing an empty collection
// therefore still invalidates every live enumerator, and that is reproduced
// here rather than optimised away.
func (c *collectionBase[T]) baseClearItems() {
	var zero T
	for i := range c.items {
		c.items[i] = zero
	}
	c.items = c.items[:0]
	c.version++
}

// ---------------------------------------------------------------------------
// Hook dispatch. Every mutating public operation goes through one of these,
// which is the `callvirt` in the reference IL.
// ---------------------------------------------------------------------------

func (c *collectionBase[T]) dispatchInsertItem(index int32, item T) error {
	return c.overrides.insertItem(index, item)
}

func (c *collectionBase[T]) dispatchRemoveItem(index int32) error {
	return c.overrides.removeItem(index)
}

func (c *collectionBase[T]) dispatchSetItem(index int32, item T) error {
	return c.overrides.setItem(index, item)
}

func (c *collectionBase[T]) dispatchClearItems() error {
	return c.overrides.clearItems()
}

// ---------------------------------------------------------------------------
// The inherited public surface.
//
// These are the eleven public members of Collection<T> that a derived type
// forwards. Constructors are not inherited in the CLR and are not here; the
// protected Items property and the four protected hooks are not public surface
// and are not here; every remaining member of Collection<T> is a private
// explicit interface implementation, which the settled BCL-interface rule
// already excludes from the projected surface.
// ---------------------------------------------------------------------------

// count is Collection<T>::get_Count, `items.Count`, which cannot fail.
func (c *collectionBase[T]) count() int32 { return int32(len(c.items)) }

// item is Collection<T>::get_Item, which is one `items[index]` with no
// validation of its own. The validation comes from List<T>::get_Item, whose
// guard is the unsigned comparison `(uint)index >= (uint)_size`, so a negative
// index fails the same test as an index past the end.
func (c *collectionBase[T]) item(index int32) (T, error) {
	var zero T
	if uint32(index) >= uint32(len(c.items)) {
		return zero, collectionIndexError("index")
	}
	return c.items[int(index)], nil
}

// assign is Collection<T>::set_Item.
//
// It validates the index itself, with `index < 0 || index >= Count`, and only
// then reaches the SetItem hook. The order matters: on a collection whose
// SetItem always fails, an out-of-range index still reports the range failure
// rather than the hook's.
func (c *collectionBase[T]) assign(index int32, value T) error {
	if index < 0 || index >= c.count() {
		return collectionIndexError("index")
	}
	return c.dispatchSetItem(index, value)
}

// add is Collection<T>::Add, which is `InsertItem(Count, item)`. It performs
// no validation of its own, so everything an Add can report comes from the
// subclass hook.
func (c *collectionBase[T]) add(item T) error {
	return c.dispatchInsertItem(c.count(), item)
}

// clear is Collection<T>::Clear, which is `ClearItems()`.
func (c *collectionBase[T]) clear() error {
	return c.dispatchClearItems()
}

// contains is Collection<T>::Contains, which is `items.Contains(item)`.
//
// List<T>.Contains splits on whether the sought item is null: a null item is
// found by scanning for a null element, and a non-null item is compared with
// EqualityComparer<T>.Default. Both branches are a forward linear scan that
// stops at the first match, and neither touches the version.
func (c *collectionBase[T]) contains(item T) bool {
	return c.indexOf(item) >= 0
}

// indexOf is Collection<T>::IndexOf, which is `items.IndexOf(item)` and ends
// in Array.IndexOf<T> over the same EqualityComparer<T>.Default. It returns
// the first match and -1 when there is none.
func (c *collectionBase[T]) indexOf(item T) int32 {
	for i := range c.items {
		if c.equal(c.items[i], item) {
			return int32(i)
		}
	}
	return -1
}

// copyTo is Collection<T>::CopyTo, which is `items.CopyTo(array, index)` and
// then Array.Copy(_items, 0, array, index, _size). The three failures are
// Array.Copy's, in its order: a null destination, a negative destination
// index, and a destination that cannot hold Count elements from that index.
func (c *collectionBase[T]) copyTo(destination []T, arrayIndex int32) error {
	if destination == nil {
		return collectionNullError("destination")
	}
	if arrayIndex < 0 {
		return collectionIndexError("arrayIndex")
	}
	if int64(arrayIndex)+int64(len(c.items)) > int64(len(destination)) {
		return collectionArgumentError("destination is too small")
	}
	copy(destination[int(arrayIndex):], c.items)
	return nil
}

// insert is Collection<T>::Insert.
//
// Its guard is `index < 0 || index > Count`, which admits Count itself, unlike
// the guards on set_Item and RemoveAt. Appending through Insert(Count, item)
// is legal exactly because Add is defined as that call.
func (c *collectionBase[T]) insert(index int32, item T) error {
	if index < 0 || index > c.count() {
		return collectionIndexError("index")
	}
	return c.dispatchInsertItem(index, item)
}

// remove is Collection<T>::Remove: find the item, return false without
// touching the collection when it is absent, and otherwise reach the
// RemoveItem hook and return true.
//
// The bool is reported even when the hook fails, because the reference reaches
// `ldc.i4.1; ret` only after RemoveItem returns normally; a hook that throws
// propagates instead. The Go projection therefore returns false with the error.
func (c *collectionBase[T]) remove(item T) (bool, error) {
	index := c.indexOf(item)
	if index < 0 {
		return false, nil
	}
	if err := c.dispatchRemoveItem(index); err != nil {
		return false, err
	}
	return true, nil
}

// removeAt is Collection<T>::RemoveAt, whose guard is
// `index < 0 || index >= Count`, and which then reaches the RemoveItem hook.
func (c *collectionBase[T]) removeAt(index int32) error {
	if index < 0 || index >= c.count() {
		return collectionIndexError("index")
	}
	return c.dispatchRemoveItem(index)
}

// getEnumerator is Collection<T>::GetEnumerator, which returns
// `items.GetEnumerator()` boxed as IEnumerator<T>, so the enumerator a caller
// observes is List<T>.Enumerator and its semantics are List<T>'s.
func (c *collectionBase[T]) getEnumerator() Iterator[T] {
	return &collectionBaseIterator[T]{collection: c, version: c.version}
}

// collectionBaseIterator projects List<T>.Enumerator onto the settled
// Iterator[T] language adapter.
//
// List<T>.Enumerator captures the list's version when it is created and
// MoveNext compares it first, before the bounds test: a version change is
// reported even when the cursor is already past the end, so a collection
// mutated after enumeration finished still fails a further step. Next
// reproduces that order.
//
// The CLR enumerator's other two states are not representable through
// Iterator[T] and are not invented here. IEnumerator<T>.Current returns
// default(T) before the first MoveNext and after the last, because
// List<T>.Enumerator.get_Current is one ldfld with no validation; Iterator[T]
// fuses MoveNext and Current into one Next, so neither state can be observed.
// IEnumerator.Reset is a private explicit implementation on the enumerator,
// which the settled BCL-interface rule already excludes from projected
// surface, so no Reset is projected either.
type collectionBaseIterator[T any] struct {
	collection *collectionBase[T]
	version    uint64
	index      int
}

func (i *collectionBaseIterator[T]) Next() (T, bool, error) {
	var zero T
	if i.version != i.collection.version {
		return zero, false, errCollectionEnumerationFailed
	}
	if i.index >= len(i.collection.items) {
		return zero, false, nil
	}
	value := i.collection.items[i.index]
	i.index++
	return value, true, nil
}

// referenceIdentityEqual projects EqualityComparer<T>.Default for an element
// type whose CLR form is a reference type that does not override
// Object.Equals.
//
// For such a T the BCL selects ObjectEqualityComparer<T>, whose Equals reaches
// the virtual Object.Equals, which is reference identity. Go's == on an
// interface value compares the dynamic type and the dynamic value, which is
// pointer identity for the pointer facades CNA-Go projects every CLR class as,
// so == is the faithful spelling.
//
// The comparability guard is a Go language necessity rather than CLR behavior.
// Go permits an interface to be satisfied by a value type, and == panics when
// two such values share a dynamic type that is not comparable -- a struct with
// a slice, map, or func field. No CLR implementor can be in that state, since
// every CLR implementor is a reference type, so there is no reference identity
// to project for one; reporting "not the same element" is the honest answer
// and is strictly better than a panic from inside a collection operation.
func referenceIdentityEqual[T comparable](a, b T) bool {
	dynamicA, dynamicB := reflect.TypeOf(a), reflect.TypeOf(b)
	if dynamicA != dynamicB {
		return false
	}
	if dynamicA == nil {
		// Both operands are the nil interface, which the CLR sees as two
		// null references.
		return true
	}
	if !dynamicA.Comparable() {
		return false
	}
	return a == b
}
