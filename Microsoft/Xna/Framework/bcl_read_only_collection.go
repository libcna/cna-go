package framework

// This file is CNA-Go language support, not XNA surface.
//
// ReadOnlyCollection is the projection of
// System.Collections.ObjectModel.ReadOnlyCollection<T>, which the pinned XNA
// public contract carries at signature positions. It is a declared language
// adapter in tools/api_compat/mapping-rules.json and adds no XNA identity, on
// exactly the footing System.TimeSpan and System.EventHandler<T> already have:
// a BCL type the contract names in a public signature needs a public Go
// spelling, or the member that returns it cannot be projected at all.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # What read-only means here
//
// ReadOnlyCollection<T> holds one private `IList<T> list` field and forwards
// every read to it:
//
//	get_Count   ldfld list; callvirt ICollection`1::get_Count
//	get_Item    ldfld list; ldarg.1; callvirt IList`1::get_Item
//	Contains    ldfld list; ldarg.1; callvirt ICollection`1::Contains
//	CopyTo      ldfld list; ldarg.1; ldarg.2; callvirt ICollection`1::CopyTo
//	IndexOf     ldfld list; ldarg.1; callvirt IList`1::IndexOf
//	GetEnumerator  ldfld list; callvirt IEnumerable`1::GetEnumerator
//
// It stores the list, it does not copy it. Read-only therefore means **no
// public mutation through this surface**, not immutable storage forever: the
// owner keeps writing to the underlying list and every change is visible
// through the view. Freezing a copy would be a different type.
//
// Every mutator is a private explicit implementation --
// `ICollection<T>.Add`, `.Clear`, `.Remove`, `IList<T>.Insert`, `.RemoveAt`,
// `IList<T>.set_Item` and the whole non-generic IList set -- so the settled
// BCL-interface rule already excludes them and no new decision was needed to
// leave them out. `Items` is `family`, and the one constructor is not
// inherited by anything.
//
// The public surface is therefore exactly six members, and every type in it
// was already decided: int32, T, []T, and the settled Iterator[T].
type ReadOnlyCollection[T any] struct {
	source readOnlyListSource[T]
}

// readOnlyListSource is the private projection of the IList<T> a
// ReadOnlyCollection<T> wraps. It is unexported, so a consumer can neither
// supply a source nor reach the one a view holds.
//
// It exists because the underlying list decides real behavior that differs by
// source, most importantly whether enumeration is version-checked. The view
// forwards; it does not impose.
type readOnlyListSource[T any] interface {
	count() int32
	item(index int32) (T, error)
	indexOf(item T) int32
	copyTo(destination []T, arrayIndex int32) error
	getEnumerator() Iterator[T]
}

// newReadOnlyCollectionOverSlice wraps a Go slice, which is the projection of
// a `T[]` used as an IList<T>.
//
// The slice header is captured by value on purpose. In the CLR the view holds
// the array *reference* it was handed, so an owner that later replaces its own
// field with a different array does not change what an existing view shows,
// while writes into the captured array are visible immediately. A captured Go
// slice header reproduces both halves exactly; a *[]T would be more live than
// the reference, and a copy would be less.
func newReadOnlyCollectionOverSlice[T any](items []T, equal func(a, b T) bool) *ReadOnlyCollection[T] {
	return &ReadOnlyCollection[T]{source: &arrayListSource[T]{items: items, equal: equal}}
}

// Count is ReadOnlyCollection<T>::get_Count, one forwarded `list.Count`. It
// cannot fail.
func (c *ReadOnlyCollection[T]) Count() int32 { return c.source.count() }

// Item is ReadOnlyCollection<T>::get_Item, one forwarded `list[index]`.
//
// The view performs no bounds check of its own; the failure is the underlying
// list's, and for an array-backed list it is the array's own range failure.
func (c *ReadOnlyCollection[T]) Item(index int32) (T, error) { return c.source.item(index) }

// Contains is ReadOnlyCollection<T>::Contains, `list.Contains(value)`, a
// forward linear scan over EqualityComparer<T>.Default that stops at the first
// match.
func (c *ReadOnlyCollection[T]) Contains(value T) bool { return c.source.indexOf(value) >= 0 }

// IndexOf is ReadOnlyCollection<T>::IndexOf, `list.IndexOf(value)`: the first
// match, or -1.
func (c *ReadOnlyCollection[T]) IndexOf(value T) int32 { return c.source.indexOf(value) }

// CopyTo is ReadOnlyCollection<T>::CopyTo, `list.CopyTo(array, index)`, which
// for an array-backed list is Array.Copy and carries its three failures in its
// order: a nil destination, a negative index, and a destination too small to
// hold Count elements from that index.
func (c *ReadOnlyCollection[T]) CopyTo(array []T, index int32) error {
	return c.source.copyTo(array, index)
}

// GetEnumerator is ReadOnlyCollection<T>::GetEnumerator, which forwards to the
// underlying list's enumerator, so enumeration semantics are the list's rather
// than the view's.
//
// That distinction is load-bearing and measured. A List<T>-backed view
// enumerates fail-fast against List<T>._version. An ARRAY-backed view does
// not: SZArrayHelper.SZGenericArrayEnumerator<T> holds only _array, _index and
// _endIndex, and its MoveNext compares the index against the end and nothing
// else. An array cannot change length, so there is no version to check, and an
// element written during enumeration is simply observed. CNA-Go reproduces
// that rather than adding an invalidation the reference does not have.
func (c *ReadOnlyCollection[T]) GetEnumerator() Iterator[T] { return c.source.getEnumerator() }

// arrayListSource is a `T[]` used as an IList<T>, which is what every pinned
// XNA consumer of ReadOnlyCollection<T> hands it.
type arrayListSource[T any] struct {
	items []T
	// equal projects EqualityComparer<T>.Default for this element type. It is
	// supplied per consumer because the comparer the BCL selects depends on T,
	// and for several primitives it is NOT Go's ==.
	equal func(a, b T) bool
}

func (s *arrayListSource[T]) count() int32 { return int32(len(s.items)) }

func (s *arrayListSource[T]) item(index int32) (T, error) {
	var zero T
	if uint32(index) >= uint32(len(s.items)) {
		return zero, collectionIndexError("index")
	}
	return s.items[int(index)], nil
}

func (s *arrayListSource[T]) indexOf(item T) int32 {
	for i := range s.items {
		if s.equal(s.items[i], item) {
			return int32(i)
		}
	}
	return -1
}

func (s *arrayListSource[T]) copyTo(destination []T, arrayIndex int32) error {
	if destination == nil {
		return collectionNullError("destination")
	}
	if arrayIndex < 0 {
		return collectionIndexError("arrayIndex")
	}
	if int64(arrayIndex)+int64(len(s.items)) > int64(len(destination)) {
		return collectionArgumentError("destination is too small")
	}
	copy(destination[int(arrayIndex):], s.items)
	return nil
}

func (s *arrayListSource[T]) getEnumerator() Iterator[T] {
	return &arrayListIterator[T]{source: s}
}

// arrayListIterator projects SZArrayHelper.SZGenericArrayEnumerator<T>.
//
// It has no version field because the reference has none. The reference's
// other observable difference from List<T>.Enumerator -- its get_Current
// throws InvalidOperationException before the first step and after the last,
// where List<T>.Enumerator.get_Current is one unvalidated ldfld -- is
// unrepresentable through Iterator[T], which fuses MoveNext and Current, so
// neither state can be reached and neither is invented.
type arrayListIterator[T any] struct {
	source *arrayListSource[T]
	index  int
}

func (i *arrayListIterator[T]) Next() (T, bool, error) {
	var zero T
	if i.index >= len(i.source.items) {
		return zero, false, nil
	}
	value := i.source.items[i.index]
	i.index++
	return value, true, nil
}

// collectionBase satisfies readOnlyListSource, so a future
// ReadOnlyCollection<T> over a live Collection<T> keeps that list's
// version-checked enumeration rather than the array source's unchecked one.
var _ readOnlyListSource[int] = (*collectionBase[int])(nil)

// liveListSource is the projection of a `List<T>` used as the IList<T> a
// ReadOnlyCollection<T> wraps, as opposed to the `T[]` arrayListSource covers.
//
// The distinction is the CLR's, not a convenience. ReadOnlyCollection<T> stores
// the IList REFERENCE it was handed, so when the owner is a List<T> that the
// owner keeps ADDING TO, every addition is visible through the view
// immediately. An array-backed view cannot observe that, because an array's
// length never changes; a List-backed one must.
//
// The source is a function rather than a captured slice for exactly that
// reason: a captured slice header would freeze the length at construction, and
// a *[]T would expose the owner's storage to anything holding the view. A
// closure over the owner is what the CLR reference is -- a way to ask the
// owner for its list, each time, without being able to replace it.
type liveListSource[T any] struct {
	list  func() []T
	equal func(a, b T) bool
}

func (s *liveListSource[T]) count() int32 { return int32(len(s.list())) }

func (s *liveListSource[T]) item(index int32) (T, error) {
	items := s.list()
	if index < 0 || int(index) >= len(items) {
		var zero T
		return zero, collectionIndexError("index")
	}
	return items[index], nil
}

func (s *liveListSource[T]) indexOf(item T) int32 {
	for i, candidate := range s.list() {
		if s.equal(candidate, item) {
			return int32(i)
		}
	}
	return -1
}

func (s *liveListSource[T]) copyTo(destination []T, arrayIndex int32) error {
	items := s.list()
	if destination == nil {
		return collectionNullError("destination")
	}
	if arrayIndex < 0 {
		return collectionIndexError("arrayIndex")
	}
	if int64(arrayIndex)+int64(len(items)) > int64(len(destination)) {
		return collectionArgumentError("destination is too small")
	}
	copy(destination[int(arrayIndex):], items)
	return nil
}

func (s *liveListSource[T]) getEnumerator() Iterator[T] {
	return &liveListIterator[T]{source: s, version: len(s.list())}
}

// liveListIterator projects List<T>.Enumerator, which is version-checked: the
// reference captures List<T>._version at construction and throws
// InvalidOperationException from MoveNext once the list has been mutated.
//
// CNA-Go has no _version to read, so the check is over the LENGTH, which is
// what every mutation of the lists this source backs changes -- the only
// mutators any consumer of this source declares are Add and Remove. A
// same-length replacement would go unnoticed here and would be caught there;
// no consumer performs one.
type liveListIterator[T any] struct {
	source  *liveListSource[T]
	version int
	index   int
}

func (i *liveListIterator[T]) Next() (T, bool, error) {
	var zero T
	items := i.source.list()
	if len(items) != i.version {
		return zero, false, errCollectionEnumerationFailed
	}
	if i.index >= len(items) {
		return zero, false, nil
	}
	value := items[i.index]
	i.index++
	return value, true, nil
}

// NewReadOnlyCollectionOverLiveReferences builds a read-only view over a
// List<T> of CLR REFERENCES that its owner keeps mutating.
//
// It exists because Microsoft.Xna.Framework.Graphics.ModelEffectCollection
// wraps a `List<Effect>` rather than an array, and ModelMeshPart.set_Effect
// adds to and removes from that list while consumers hold the view. It lives in
// a different Go package and cannot reach the unexported source types.
//
// The element comparer is Go's `==`, which for a pointer is identity --
// EqualityComparer<T>.Default for a CLR class that overrides no Equals.
func NewReadOnlyCollectionOverLiveReferences[T comparable](list func() []T) *ReadOnlyCollection[T] {
	return &ReadOnlyCollection[T]{source: &liveListSource[T]{
		list:  list,
		equal: func(a, b T) bool { return a == b },
	}}
}

// NewReadOnlyCollectionOverSingles builds the live read-only view a
// System.Single buffer is wrapped in, which is the one shape the pinned XNA
// contract constructs.
//
// It is exported because the XNA type that owns such a buffer,
// Microsoft.Xna.Framework.Media.VisualizationData, lives in a different Go
// package and must be able to build the view its projected members return. It
// is the language adapter's constructor, not an XNA identity, and matches the
// reference's own public ReadOnlyCollection<T>(IList<T>) constructor rather
// than inventing a capability the BCL withholds.
//
// The element comparer is System.Single's, so a NaN element is found by a NaN
// search, exactly as EqualityComparer<Single>.Default behaves and unlike Go's
// ==.
func NewReadOnlyCollectionOverSingles(items []float32) *ReadOnlyCollection[float32] {
	return newReadOnlyCollectionOverSlice(items, singleEquals)
}

// NewReadOnlyCollectionOverReferences is the same language-adapter constructor
// for a collection of CLR REFERENCES, and it exists for the same reason: the
// XNA type that owns one -- Microsoft.Xna.Framework.Graphics.GraphicsAdapter,
// whose static Adapters property returns
// ReadOnlyCollection<GraphicsAdapter> -- lives in a different Go package and
// must be able to build the view its projected member returns.
//
// The element comparer is Go's `==`, which for a pointer is identity. That is
// EqualityComparer<T>.Default for a CLR class that overrides no Equals, which
// GraphicsAdapter does not.
func NewReadOnlyCollectionOverReferences[T comparable](items []T) *ReadOnlyCollection[T] {
	return newReadOnlyCollectionOverSlice(items, func(a, b T) bool { return a == b })
}

// NewReadOnlyCollectionOverValues is the fifth element-kind constructor, for a
// collection whose elements are CLR STRUCTS.
//
// It exists for AudioEngine::get_RendererDetails, whose CLR type is
// ReadOnlyCollection<RendererDetail>: the element is a two-string struct that
// lives in a different Go package, so the view has to be built from there.
//
// The element comparer is Go's `==` on the struct. That is
// EqualityComparer<T>.Default for a value type whose Equals is a field-wise
// comparison -- which is exactly what RendererDetail::op_Equality is, two
// String::op_Equality calls over the same two fields Go compares. It is a fifth
// constructor rather than a use of the reference one for the reason the
// character constructor is a third: these elements are VALUES, and a name
// saying "over references" would describe the opposite of what the collection
// holds.
//
// A struct with a float field would NOT be admissible here, because Go's ==
// leaves NaN unequal to itself where EqualityComparer<T>.Default finds it. The
// constraint cannot express that, so it is stated: the caller is choosing Go's
// comparer and must know it matches.
func NewReadOnlyCollectionOverValues[T comparable](items []T) *ReadOnlyCollection[T] {
	return newReadOnlyCollectionOverSlice(items, func(a, b T) bool { return a == b })
}

// NewReadOnlyCollectionOverCharacters is the third element-kind constructor,
// for System.Char. It exists for the reason the second does -- the XNA type
// that owns one, Microsoft.Xna.Framework.Graphics.SpriteFont, whose Characters
// property returns ReadOnlyCollection<char>, lives in a different Go package --
// and it is its own name rather than a use of the reference constructor because
// a font's characters are VALUES, and a constructor named "over references"
// would say the opposite of what the collection holds.
//
// The element comparer is Go's `==` on the UTF-16 code unit System.Char
// projects to, which is EqualityComparer<Char>.Default.
func NewReadOnlyCollectionOverCharacters(items []uint16) *ReadOnlyCollection[uint16] {
	return newReadOnlyCollectionOverSlice(items, func(a, b uint16) bool { return a == b })
}
