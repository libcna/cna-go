package framework

// The exact Resources strings the reference throw sites load, read from the
// Microsoft.Xna.Framework.Resources.resources stream of the retained
// Microsoft.Xna.Framework.Game.dll
// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0).
const (
	cannotAddSameComponentMultipleTimes       = "Cannot add the same game component to a game component collection multiple times."
	cannotSetItemsIntoGameComponentCollection = "Cannot set a value using operator[] on GameComponentCollection.  Use Add/Remove instead."
)

// GameComponentCollection is XNA's component list: the collection Game.Components
// exposes, which raises ComponentAdded and ComponentRemoved as it changes.
//
// # The base-class composition
//
// The CLR class is
//
//	.class public auto ansi sealed GameComponentCollection
//	    extends [mscorlib]System.Collections.ObjectModel.Collection`1<IGameComponent>
//
// and it declares only seven members: one public constructor, four protected
// overrides, and two events. Everything a caller actually uses -- Add, Remove,
// Clear, Count, the indexer, IndexOf, Insert, RemoveAt, Contains, CopyTo and
// GetEnumerator -- is inherited public surface of Collection<IGameComponent>.
//
// Go has no CLR inheritance, so the base is held as the unexported `base`
// field, an instance of the private collectionBase[T] adapter, and the eleven
// inherited public members are re-exposed as measured forwarding methods. The
// adapter is never embedded, never exported, and never returned: no caller can
// name it, reach the backing store, or mutate the collection without passing
// through the subclass hooks below.
//
// # The four overrides, and the asymmetry between them
//
// The reference IL is deliberately not symmetric, and the projection preserves
// it exactly:
//
//	InsertItem   rejects a duplicate FIRST, then mutates, then raises
//	             ComponentAdded -- but only for a non-null component
//	RemoveItem   reads the element, mutates, then raises ComponentRemoved --
//	             but only for a non-null component
//	SetItem      throws NotSupportedException unconditionally, having read
//	             neither of its arguments
//	ClearItems   raises ComponentRemoved for EVERY element, index 0 upward,
//	             with NO null check, and only THEN mutates
//
// So Insert and Remove mutate before they announce, while Clear announces the
// whole collection before it changes anything; and Clear is the one hook that
// will hand a handler a GameComponentCollectionEventArgs whose GameComponent
// is nil. A handler that fails therefore leaves an Insert or a Remove already
// applied, and leaves a Clear not applied at all.
//
// # What is projected, and what is not
//
// The four overrides are `family` in the reference metadata and the class is
// sealed, so no CLR caller can reach them. They are still projected, because
// the pinned contract declares them and CNA-Go projects a declared protected
// member like any other; they are the same code the collection reaches through
// the private hook interface, so calling one directly behaves identically.
//
// The CLR spells the protected override and the inherited indexer setter the
// same, `SetItem`. The settled collision rule resolves that mechanically by
// appending each collider's source kind, which is why the override is
// SetItemMethod and the indexer setter is SetItemProperty. Neither name is
// invented for this type.
//
// # What drives this collection
//
// As of Foundation 30 this is no longer an unused projection. Game allocates
// exactly one of these in its constructor and immediately subscribes its own
// two private handlers to both events, so every mutation announced here is what
// keeps Game's derived update and draw lists correct. The asymmetries above are
// therefore load-bearing rather than curiosities: because Insert mutates before
// it announces, Game's handler already sees the component in the collection,
// and because Clear announces everything before it mutates, Game untracks every
// component while the collection is still full.
type GameComponentCollection struct {
	// base is the private Collection<IGameComponent> adapter. It is
	// unexported, not embedded, and never handed out.
	base             collectionBase[IGameComponent]
	componentAdded   EventSource[*GameComponentCollectionEventArgs]
	componentRemoved EventSource[*GameComponentCollectionEventArgs]
}

// NewGameComponentCollection projects the one public constructor, whose whole
// body is `call Collection`1<IGameComponent>::.ctor()`. The base constructor
// assigns a fresh List<IGameComponent>, so a new collection is empty and its
// backing store is never read-only.
func NewGameComponentCollection() *GameComponentCollection {
	collection := &GameComponentCollection{}
	collection.base.init(collection, referenceIdentityEqual[IGameComponent])
	return collection
}

// ---------------------------------------------------------------------------
// The four XNA-declared protected overrides.
//
// These are the projected members of the pinned contract AND the bodies the
// private hook interface dispatches to, so there is exactly one implementation
// of each and no path can reach a mutation without running it.
// ---------------------------------------------------------------------------

// InsertItem is GameComponentCollection::InsertItem.
//
//	IndexOf(item) != -1 -> throw ArgumentException(CannotAddSameComponentMultipleTimes)
//	base.InsertItem(index, item)
//	item != null -> OnComponentAdded(new GameComponentCollectionEventArgs(item))
//
// The duplicate test runs before the insertion, so a rejected duplicate leaves
// the collection untouched and raises nothing. The event args are freshly
// allocated per raise -- this is not the shared EventArgs.Empty identity that
// XNA's argument-free events use -- and the sender is the collection.
//
// A nil component is inserted like any other, because IndexOf(nil) finds a nil
// element only when one is already present; it simply announces nothing. That
// is also why a second nil insertion fails as a duplicate.
func (c *GameComponentCollection) InsertItem(index int32, item IGameComponent) error {
	if c.base.indexOf(item) != -1 {
		return collectionArgumentError(cannotAddSameComponentMultipleTimes)
	}
	c.base.baseInsertItem(index, item)
	if item == nil {
		return nil
	}
	return c.componentAdded.Raise(c, NewGameComponentCollectionEventArgs(item))
}

// RemoveItem is GameComponentCollection::RemoveItem.
//
//	IGameComponent removed = base[index]
//	base.RemoveItem(index)
//	removed != null -> OnComponentRemoved(new GameComponentCollectionEventArgs(removed))
//
// The element is read before it is removed and announced after, so a handler
// observing the collection during ComponentRemoved sees it already gone.
func (c *GameComponentCollection) RemoveItem(index int32) error {
	removed, err := c.base.item(index)
	if err != nil {
		return err
	}
	c.base.baseRemoveItem(index)
	if removed == nil {
		return nil
	}
	return c.componentRemoved.Raise(c, NewGameComponentCollectionEventArgs(removed))
}

// SetItemMethod is GameComponentCollection::SetItem, whose entire body is
//
//	newobj NotSupportedException(CannotSetItemsIntoGameComponentCollection); throw
//
// It reads neither argument and has no success path. Reaching it through the
// inherited indexer setter is different only in that the setter validates the
// index first, so an out-of-range assignment reports the range failure instead.
//
// The Method suffix is the settled collision rule resolving this override
// against the inherited Item setter, which the same rules also spell SetItem.
func (c *GameComponentCollection) SetItemMethod(index int32, item IGameComponent) error {
	return collectionNotSupportedError(cannotSetItemsIntoGameComponentCollection)
}

// ClearItems is GameComponentCollection::ClearItems.
//
//	for (int i = 0; i < base.Count; i++)
//	    OnComponentRemoved(new GameComponentCollectionEventArgs(base[i]));
//	base.ClearItems();
//
// Three details are load-bearing and none of them is tidied up here. There is
// no null check, so a nil element is announced too, unlike RemoveItem. Count is
// re-read on every iteration, so a handler that adds a component extends the
// loop. And the mutation happens only after every announcement, so a handler
// that fails leaves the collection fully intact.
func (c *GameComponentCollection) ClearItems() error {
	for i := int32(0); i < c.base.count(); i++ {
		item, err := c.base.item(i)
		if err != nil {
			return err
		}
		if err := c.componentRemoved.Raise(c, NewGameComponentCollectionEventArgs(item)); err != nil {
			return err
		}
	}
	c.base.baseClearItems()
	return nil
}

// The private hook interface the adapter dispatches through. Each is the
// `callvirt` in Collection<T>'s IL landing on this class's override.
func (c *GameComponentCollection) insertItem(index int32, item IGameComponent) error {
	return c.InsertItem(index, item)
}

func (c *GameComponentCollection) removeItem(index int32) error { return c.RemoveItem(index) }

func (c *GameComponentCollection) setItem(index int32, item IGameComponent) error {
	return c.SetItemMethod(index, item)
}

func (c *GameComponentCollection) clearItems() error { return c.ClearItems() }

// ---------------------------------------------------------------------------
// The inherited public surface of Collection<IGameComponent>.
// ---------------------------------------------------------------------------

// Count is the inherited Count property, `items.Count`.
func (c *GameComponentCollection) Count() int32 { return c.base.count() }

// Item is the inherited indexer getter. Collection<T> forwards it unvalidated
// to List<T>, whose guard is the unsigned `(uint)index >= (uint)_size`, so a
// negative index fails exactly as an index past the end does.
func (c *GameComponentCollection) Item(index int32) (IGameComponent, error) {
	return c.base.item(index)
}

// SetItemProperty is the inherited indexer setter. It validates the index
// itself and then reaches SetItem, which always fails, so the only thing this
// member can vary is which failure it reports: an out-of-range index reports
// the range failure, and an in-range one reports the unsupported assignment.
// It never mutates and never raises.
func (c *GameComponentCollection) SetItemProperty(index int32, value IGameComponent) error {
	return c.base.assign(index, value)
}

// Add is the inherited Add, defined as `InsertItem(Count, item)`. It validates
// nothing itself, so a duplicate component is the only way it fails.
func (c *GameComponentCollection) Add(item IGameComponent) error { return c.base.add(item) }

// Clear is the inherited Clear, defined as `ClearItems()`.
func (c *GameComponentCollection) Clear() error { return c.base.clear() }

// Contains is the inherited Contains, a forward linear scan that stops at the
// first match.
func (c *GameComponentCollection) Contains(item IGameComponent) bool {
	return c.base.contains(item)
}

// CopyTo is the inherited CopyTo, which ends in Array.Copy. Its three failures
// are Array.Copy's, in Array.Copy's order: a nil destination, a negative index,
// and a destination too small to hold Count elements from that index.
func (c *GameComponentCollection) CopyTo(array []IGameComponent, index int32) error {
	return c.base.copyTo(array, index)
}

// GetEnumerator is the inherited GetEnumerator. Collection<T> returns the
// backing List<T>'s enumerator boxed as IEnumerator<T>, so enumeration is
// List<T>'s: source order, and fail-fast against the version the list bumps on
// every mutation, including a Clear of an already empty collection.
func (c *GameComponentCollection) GetEnumerator() Iterator[IGameComponent] {
	return c.base.getEnumerator()
}

// IndexOf is the inherited IndexOf: the first match, or -1.
func (c *GameComponentCollection) IndexOf(item IGameComponent) int32 {
	return c.base.indexOf(item)
}

// Insert is the inherited Insert. Its guard admits Count itself, unlike the
// indexer setter's and RemoveAt's, and it validates the index before the
// duplicate test in InsertItem runs.
func (c *GameComponentCollection) Insert(index int32, item IGameComponent) error {
	return c.base.insert(index, item)
}

// Remove is the inherited Remove: false and no change when the component is
// absent, otherwise the RemoveItem hook and true.
func (c *GameComponentCollection) Remove(item IGameComponent) (bool, error) {
	return c.base.remove(item)
}

// RemoveAt is the inherited RemoveAt, whose guard rejects Count itself.
func (c *GameComponentCollection) RemoveAt(index int32) error { return c.base.removeAt(index) }

// ---------------------------------------------------------------------------
// The two events, on the settled two-accessor projection.
// ---------------------------------------------------------------------------

// AddComponentAddedHandler registers a handler for ComponentAdded, which
// InsertItem raises after a non-nil component has been inserted.
func (c *GameComponentCollection) AddComponentAddedHandler(handler EventHandler[*GameComponentCollectionEventArgs]) (EventSubscription, error) {
	return c.componentAdded.Add(handler)
}

// RemoveComponentAddedHandler removes the registration the token names.
func (c *GameComponentCollection) RemoveComponentAddedHandler(subscription EventSubscription) error {
	return c.componentAdded.Remove(subscription)
}

// AddComponentRemovedHandler registers a handler for ComponentRemoved, which
// RemoveItem raises after a non-nil component has been removed and ClearItems
// raises for every element -- nil ones included -- before it removes any.
func (c *GameComponentCollection) AddComponentRemovedHandler(handler EventHandler[*GameComponentCollectionEventArgs]) (EventSubscription, error) {
	return c.componentRemoved.Add(handler)
}

// RemoveComponentRemovedHandler removes the registration the token names.
func (c *GameComponentCollection) RemoveComponentRemovedHandler(subscription EventSubscription) error {
	return c.componentRemoved.Remove(subscription)
}
