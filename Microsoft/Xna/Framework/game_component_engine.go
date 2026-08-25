package framework

// This file is the private component engine of Microsoft.Xna.Framework.Game.
//
// Nothing declared here is exported and nothing here is projected XNA surface.
// Every name below is a `private` member of the reference Game class, read from
// Microsoft.Xna.Framework.Game.dll
// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0),
// with the inherited List<T> behavior read from the admitted
//
//	mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # Why Game keeps derived lists at all
//
// Game does NOT sort Components on every frame. The reference declares five
// private lists beside the public collection:
//
//	List<IUpdateable>   updateableComponents
//	List<IUpdateable>   currentlyUpdatingComponents
//	List<IDrawable>     drawableComponents
//	List<IDrawable>     currentlyDrawingComponents
//	List<IGameComponent> notYetInitialized
//
// and keeps the first and third sorted incrementally: a component is placed at
// its ordered position when it is ADDED and re-placed when its order CHANGES.
// The per-frame cost is a copy, not a sort. The relationship to Components is
// explicit and one-directional: Components is the only public collection, and
// these lists are maintained purely by the two collection handlers Game
// subscribes in its own constructor.
//
// # Insertion ordering, and why ties are stable without a stable sort
//
// The reference does not use a stable sort; it does not sort at all. It uses
// List<T>.BinarySearch with UpdateOrderComparer.Default / DrawOrderComparer.Default
// and then walks forward past every equal-order element. Both comparers have the
// same deliberately non-total shape:
//
//	Compare(x, y):
//	    x == null && y == null -> 0
//	    x == null              -> 1      // a null sorts AFTER a non-null
//	    y == null              -> -1
//	    x.Equals(y)            -> 0      // Object.Equals: reference identity
//	    x.Order < y.Order      -> -1
//	    otherwise              -> 1      // EQUAL orders report 1, never 0
//
// Two distinct components with the same order therefore never compare equal.
// The only way BinarySearch returns a non-negative index is that the searched
// component is ALREADY in the list, which is exactly what the `if (index < 0)`
// guard in the add handler tests: it is a "not already present" guard, not an
// "equal order found" guard.
//
// Because equal-order elements report 1 -- "the list element sorts after the
// searched value" -- the binary search always converges left of the run of
// equal-order elements, and the explicit forward walk then steps past all of
// them. A new component with an existing order lands AFTER every component that
// already has that order, so ties keep insertion order. That is derived, not
// assumed: Array.BinarySearch's insertion point is not otherwise ordered among
// equal elements, and the walk is what makes the result deterministic.

// updateableEntry pairs one tracked IUpdateable with the subscription token
// that removes Game's UpdateOrderChanged handler from it.
//
// The token is an implementation necessity of the settled event-accessor
// projection rather than a reference identity. CLR's
//
//	updateable.UpdateOrderChanged -= this.UpdateableUpdateOrderChanged
//
// finds the entry by delegate equality; CNA-Go names the registration instead
// of the handler, so the registration has to travel with the component. It
// travels with the list entry, which is also why the order-changed handler's
// remove-then-reinsert keeps the same subscription: the reference re-inserts
// without re-subscribing, and so does this.
type updateableEntry struct {
	component    IUpdateable
	subscription EventSubscription
}

type drawableEntry struct {
	component    IDrawable
	subscription EventSubscription
}

// compareUpdateOrder is Microsoft.Xna.Framework.UpdateOrderComparer::Compare,
// a private sealed IComparer<IUpdateable> whose single instance is the static
// Default field. See the shape derived in this file's header comment.
func compareUpdateOrder(x, y IUpdateable) int32 {
	if x == nil && y == nil {
		return 0
	}
	if x == nil {
		return 1
	}
	if y == nil {
		return -1
	}
	if referenceIdentityEqual(x, y) {
		return 0
	}
	if x.UpdateOrder() < y.UpdateOrder() {
		return -1
	}
	return 1
}

// compareDrawOrder is Microsoft.Xna.Framework.DrawOrderComparer::Compare. The
// IL is byte-for-byte the same shape over IDrawable and DrawOrder.
func compareDrawOrder(x, y IDrawable) int32 {
	if x == nil && y == nil {
		return 0
	}
	if x == nil {
		return 1
	}
	if y == nil {
		return -1
	}
	if referenceIdentityEqual(x, y) {
		return 0
	}
	if x.DrawOrder() < y.DrawOrder() {
		return -1
	}
	return 1
}

// binarySearchEntries is
// System.Collections.Generic.ArraySortHelper<T>::InternalBinarySearch, which
// List<T>.BinarySearch reaches through Array.BinarySearch. The midpoint is
// `lo + ((hi - lo) >> 1)` and a miss returns the bitwise complement of the
// insertion point, both exactly as the reference computes them.
func binarySearchEntries[E any](entries []E, value E, compare func(a, b E) int32) int32 {
	lo, hi := int32(0), int32(len(entries))-1
	for lo <= hi {
		i := lo + ((hi - lo) >> 1)
		order := compare(entries[i], value)
		if order == 0 {
			return i
		}
		if order < 0 {
			lo = i + 1
		} else {
			hi = i - 1
		}
	}
	return ^lo
}

// orderedInsertionIndex is the shared body of the reference's four identical
// placement sites: BinarySearch, and on a miss, walk forward past every element
// whose order equals the new element's order.
//
// It returns -1 when BinarySearch found the element, which is the reference's
// `if (index < 0)` guard failing and the whole placement being skipped.
func orderedInsertionIndex[E any](entries []E, value E, compare func(a, b E) int32, order func(E) int32) int32 {
	index := binarySearchEntries(entries, value, compare)
	if index >= 0 {
		return -1
	}
	index = ^index
	for index < int32(len(entries)) && order(entries[index]) == order(value) {
		index++
	}
	return index
}

func updateableEntryOrder(e updateableEntry) int32 { return e.component.UpdateOrder() }

func drawableEntryOrder(e drawableEntry) int32 { return e.component.DrawOrder() }

func compareUpdateableEntries(a, b updateableEntry) int32 {
	return compareUpdateOrder(a.component, b.component)
}

func compareDrawableEntries(a, b drawableEntry) int32 {
	return compareDrawOrder(a.component, b.component)
}

// indexOfUpdateable is List<IUpdateable>.IndexOf, which reaches
// Array.IndexOf<T> and therefore EqualityComparer<T>.Default: for an interface
// element type the BCL selects ObjectEqualityComparer<T>, whose Equals is the
// virtual Object.Equals, which is reference identity for every CLR component.
// It returns the FIRST match, or -1.
func indexOfUpdateable(entries []updateableEntry, component IUpdateable) int32 {
	for i := range entries {
		if referenceIdentityEqual(entries[i].component, component) {
			return int32(i)
		}
	}
	return -1
}

func indexOfDrawable(entries []drawableEntry, component IDrawable) int32 {
	for i := range entries {
		if referenceIdentityEqual(entries[i].component, component) {
			return int32(i)
		}
	}
	return -1
}

func indexOfGameComponent(items []IGameComponent, component IGameComponent) int32 {
	for i := range items {
		if referenceIdentityEqual(items[i], component) {
			return int32(i)
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// The two collection handlers Game subscribes in its own constructor.
// ---------------------------------------------------------------------------

// gameComponentAdded is Microsoft.Xna.Framework.Game::GameComponentAdded, the
// private handler the Game constructor attaches to Components.ComponentAdded:
//
//	if (inRun) e.GameComponent.Initialize();
//	else       notYetInitialized.Add(e.GameComponent);
//
//	if (e.GameComponent is IUpdateable u) {
//	    int i = updateableComponents.BinarySearch(u, UpdateOrderComparer.Default);
//	    if (i < 0) {
//	        i = ~i;
//	        while (i < Count && list[i].UpdateOrder == u.UpdateOrder) i++;
//	        updateableComponents.Insert(i, u);
//	        u.UpdateOrderChanged += UpdateableUpdateOrderChanged;
//	    }
//	}
//	... the same again over IDrawable / DrawOrder ...
//
// Three details are load-bearing.
//
// First, initialization is decided by `inRun`, not by whether Initialize has
// ever run: `inRun` becomes true only after Game.Initialize() has returned
// inside the run sequence, so a component added at any point before that is
// queued and a component added afterwards is initialized on the spot.
//
// Second, the two placements are independent. A component that is both
// IUpdateable and IDrawable is placed in both lists, and a component that is
// neither is still initialized or queued.
//
// Third, the initialization step comes FIRST. A component whose Initialize
// fails is therefore never placed in either derived list and never subscribed
// to, because the reference's exception leaves the rest of the handler
// unreached -- and because Raise stops at the first failing handler, a user's
// own ComponentAdded handler does not run either.
func (g *Game) gameComponentAdded(sender any, e *GameComponentCollectionEventArgs) error {
	component := e.GameComponent()
	if g.inRun {
		if err := component.Initialize(); err != nil {
			return err
		}
	} else {
		g.notYetInitialized = append(g.notYetInitialized, component)
	}
	if updateable, ok := component.(IUpdateable); ok && updateable != nil {
		entry := updateableEntry{component: updateable}
		index := orderedInsertionIndex(g.updateableComponents, entry, compareUpdateableEntries, updateableEntryOrder)
		if index >= 0 {
			subscription, err := updateable.AddUpdateOrderChangedHandler(g.updateableUpdateOrderChanged)
			if err != nil {
				return err
			}
			entry.subscription = subscription
			g.updateableComponents = insertAt(g.updateableComponents, index, entry)
		}
	}
	if drawable, ok := component.(IDrawable); ok && drawable != nil {
		entry := drawableEntry{component: drawable}
		index := orderedInsertionIndex(g.drawableComponents, entry, compareDrawableEntries, drawableEntryOrder)
		if index >= 0 {
			subscription, err := drawable.AddDrawOrderChangedHandler(g.drawableDrawOrderChanged)
			if err != nil {
				return err
			}
			entry.subscription = subscription
			g.drawableComponents = insertAt(g.drawableComponents, index, entry)
		}
	}
	return nil
}

// gameComponentRemoved is Microsoft.Xna.Framework.Game::GameComponentRemoved:
//
//	if (!inRun) notYetInitialized.Remove(e.GameComponent);
//	if (e.GameComponent is IUpdateable u) {
//	    updateableComponents.Remove(u);
//	    u.UpdateOrderChanged -= UpdateableUpdateOrderChanged;
//	}
//	... the same again over IDrawable ...
//
// The `inRun` guard is inverted from the add handler's and guards only the
// pending-initialization queue: while the game is running there is no queue to
// clean, because every added component was initialized immediately.
//
// The handler is deliberately null-tolerant. ClearItems announces every element
// with no null check, so this handler is the one place a nil component reaches
// the engine; `List.Remove(null)` finds nothing and the two type tests fail, so
// a nil element removes nothing and unsubscribes nothing.
func (g *Game) gameComponentRemoved(sender any, e *GameComponentCollectionEventArgs) error {
	component := e.GameComponent()
	if !g.inRun {
		if index := indexOfGameComponent(g.notYetInitialized, component); index >= 0 {
			g.notYetInitialized = removeAt(g.notYetInitialized, index)
		}
	}
	if updateable, ok := component.(IUpdateable); ok && updateable != nil {
		if index := indexOfUpdateable(g.updateableComponents, updateable); index >= 0 {
			subscription := g.updateableComponents[index].subscription
			g.updateableComponents = removeAt(g.updateableComponents, index)
			if err := updateable.RemoveUpdateOrderChangedHandler(subscription); err != nil {
				return err
			}
		}
	}
	if drawable, ok := component.(IDrawable); ok && drawable != nil {
		if index := indexOfDrawable(g.drawableComponents, drawable); index >= 0 {
			subscription := g.drawableComponents[index].subscription
			g.drawableComponents = removeAt(g.drawableComponents, index)
			if err := drawable.RemoveDrawOrderChangedHandler(subscription); err != nil {
				return err
			}
		}
	}
	return nil
}

// updateableUpdateOrderChanged is
// Microsoft.Xna.Framework.Game::UpdateableUpdateOrderChanged, the handler Game
// attaches to every tracked IUpdateable:
//
//	IUpdateable u = sender as IUpdateable;
//	updateableComponents.Remove(u);
//	int i = updateableComponents.BinarySearch(u, UpdateOrderComparer.Default);
//	if (i < 0) { i = ~i; while (...) i++; updateableComponents.Insert(i, u); }
//
// It reads the SENDER, not the event args, which is why the reference's
// component raises its order-changed event with itself as sender rather than
// forwarding the sender it was handed.
//
// The removal comes first, so the search can never find the component and the
// re-insertion always happens; and the component is re-placed by the same
// forward walk as a fresh add, so a component whose order changes to an
// existing order moves to the END of that order's run rather than back to its
// old position. The subscription is not renewed, exactly as the reference does
// not re-attach the handler.
func (g *Game) updateableUpdateOrderChanged(sender any, args *EventArgs) error {
	updateable, ok := sender.(IUpdateable)
	if !ok || updateable == nil {
		return nil
	}
	entry := updateableEntry{component: updateable}
	if index := indexOfUpdateable(g.updateableComponents, updateable); index >= 0 {
		entry = g.updateableComponents[index]
		g.updateableComponents = removeAt(g.updateableComponents, index)
	}
	index := orderedInsertionIndex(g.updateableComponents, entry, compareUpdateableEntries, updateableEntryOrder)
	if index >= 0 {
		g.updateableComponents = insertAt(g.updateableComponents, index, entry)
	}
	return nil
}

// drawableDrawOrderChanged is
// Microsoft.Xna.Framework.Game::DrawableDrawOrderChanged. The IL is the same
// shape over IDrawable and DrawOrder.
func (g *Game) drawableDrawOrderChanged(sender any, args *EventArgs) error {
	drawable, ok := sender.(IDrawable)
	if !ok || drawable == nil {
		return nil
	}
	entry := drawableEntry{component: drawable}
	if index := indexOfDrawable(g.drawableComponents, drawable); index >= 0 {
		entry = g.drawableComponents[index]
		g.drawableComponents = removeAt(g.drawableComponents, index)
	}
	index := orderedInsertionIndex(g.drawableComponents, entry, compareDrawableEntries, drawableEntryOrder)
	if index >= 0 {
		g.drawableComponents = insertAt(g.drawableComponents, index, entry)
	}
	return nil
}

// insertAt and removeAt are List<T>.Insert and List<T>.RemoveAt over a Go
// slice. Both callers have already validated the index against the same bounds
// the reference validates, so neither reproduces a guard here.
func insertAt[E any](items []E, index int32, value E) []E {
	var zero E
	items = append(items, zero)
	copy(items[index+1:], items[index:])
	items[index] = value
	return items
}

func removeAt[E any](items []E, index int32) []E {
	var zero E
	copy(items[index:], items[index+1:])
	items[len(items)-1] = zero
	return items[:len(items)-1]
}
