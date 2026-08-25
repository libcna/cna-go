package touch

import "errors"

// touchLocationSlots is the fixed inline capacity of a TouchCollection. The
// reference declares eight private TouchLocation fields, `location0` through
// `location7`, rather than an array, so a collection allocates nothing and
// can never hold a ninth touch.
const touchLocationSlots = 8

// Sentinel errors projecting the exact CLR exceptions the reference throws.
// They are unexported: the XNA public contract declares no error type here, so
// exporting one would be public surface XNA does not have. Package tests match
// on them with errors.Is.
var (
	// errTouchArgumentNull projects System.ArgumentNullException.
	errTouchArgumentNull = errors.New("touch argument is nil")
	// errTouchArgumentOutOfRange projects System.ArgumentOutOfRangeException.
	errTouchArgumentOutOfRange = errors.New("touch argument is out of range")
	// errTouchNotSupported projects System.NotSupportedException, which every
	// TouchCollection mutator throws unconditionally.
	errTouchNotSupported = errors.New("touch collection is read-only")
)

func touchArgumentNullError(parameter string) error {
	return &touchArgumentError{sentinel: errTouchArgumentNull, parameter: parameter}
}

func touchArgumentOutOfRangeError(parameter string) error {
	return &touchArgumentError{sentinel: errTouchArgumentOutOfRange, parameter: parameter}
}

func touchNotSupportedError(operation string) error {
	return &touchArgumentError{sentinel: errTouchNotSupported, parameter: operation}
}

// touchArgumentError carries the exact parameter name the reference passes to
// the CLR exception constructor, which is the part of the thrown value the IL
// actually pins.
type touchArgumentError struct {
	sentinel  error
	parameter string
}

func (e *touchArgumentError) Error() string {
	return e.sentinel.Error() + ": " + e.parameter
}

func (e *touchArgumentError) Unwrap() error { return e.sentinel }

// TouchCollection is XNA's read-only view of the touch points in one sample.
// It is a System.ValueType with eight inline TouchLocation slots, a count, and
// a connection flag; it allocates nothing, reads no device, and reaches no
// native code.
//
// Constructing one claims no touch capability. CNA-Go exposes no TouchPanel
// and never produces a collection itself; a caller supplies the locations.
//
// It declares IList<TouchLocation>, but only to expose the read side. Every
// mutating operation of that contract throws System.NotSupportedException
// unconditionally in the reference, and IsReadOnly is a constant true. Those
// throws are reproduced as errors rather than silently accepted, because a
// caller that mutates a read-only view has a real bug to hear about.
type TouchCollection struct {
	isConnected   bool
	locationCount int32
	locations     [touchLocationSlots]TouchLocation
}

// NewTouchCollection reproduces the public TouchCollection constructor. It
// validates in the reference's order -- nil first, then capacity -- and marks
// the collection connected.
//
// For each supplied location it stores the location together with its previous
// sample when TryGetPreviousLocation reports one, and with a zeroed previous
// sample when it does not; the zeroed previous state is TouchLocationState's
// zero literal, Invalid.
func NewTouchCollection(touches []TouchLocation) (TouchCollection, error) {
	if touches == nil {
		return TouchCollection{}, touchArgumentNullError("touches")
	}
	if len(touches) > touchLocationSlots {
		return TouchCollection{}, touchArgumentOutOfRangeError("touches")
	}
	collection := TouchCollection{isConnected: true}
	for _, touch := range touches {
		position := touch.Position()
		if hasPrevious, previous := touch.TryGetPreviousLocation(); hasPrevious {
			previousPosition := previous.Position()
			collection.addTouchLocation(
				touch.Id(), touch.State(), position.X, position.Y,
				previous.State(), previousPosition.X, previousPosition.Y,
			)
			continue
		}
		collection.addTouchLocation(
			touch.Id(), touch.State(), position.X, position.Y,
			TouchLocationState(0), 0, 0,
		)
	}
	return collection, nil
}

// addTouchLocation reproduces the private AddTouchLocation, including its
// ordering: the count is incremented first and the *previous* count selects
// the slot, so a ninth call increments the count while storing nothing. The
// public constructor's capacity check makes that branch unreachable from
// public API, and it is reproduced rather than guarded so the helper matches
// the reference exactly.
func (c *TouchCollection) addTouchLocation(
	id int32, state TouchLocationState, x, y float32,
	previousState TouchLocationState, previousX, previousY float32,
) {
	slot := c.locationCount
	c.locationCount++
	if slot < 0 || slot >= touchLocationSlots {
		return
	}
	c.locations[slot] = TouchLocation{
		id: id, state: state, x: x, y: y,
		prevState: previousState, prevX: previousX, prevY: previousY,
	}
}

// IsConnected reports the stored connection flag. The public constructor sets
// it true, so only a zero-valued collection reports false.
func (c TouchCollection) IsConnected() bool { return c.isConnected }

// Count is the number of stored touch locations.
func (c TouchCollection) Count() int32 { return c.locationCount }

// IsReadOnly is the reference's constant `ldc.i4.1; ret`. It is always true,
// including for a zero-valued collection.
func (c TouchCollection) IsReadOnly() bool { return true }

// Item reproduces the indexer getter, which validates `index < 0 ||
// index >= Count` and throws System.ArgumentOutOfRangeException("index")
// before reading a slot.
func (c TouchCollection) Item(index int32) (TouchLocation, error) {
	if index < 0 || index >= c.locationCount {
		return TouchLocation{}, touchArgumentOutOfRangeError("index")
	}
	if index >= touchLocationSlots {
		// Unreachable while Count never exceeds the slot capacity. The
		// reference's switch falls through to its last slot here rather than
		// throwing, and that is what is reproduced.
		return c.locations[touchLocationSlots-1], nil
	}
	return c.locations[index], nil
}

// SetItem reproduces the indexer setter, whose entire body is
// `newobj NotSupportedException; throw`. It validates nothing first and never
// stores.
func (c *TouchCollection) SetItem(index int32, value TouchLocation) error {
	return touchNotSupportedError("set_Item")
}

// Add is an unconditional System.NotSupportedException in the reference.
func (c *TouchCollection) Add(item TouchLocation) error {
	return touchNotSupportedError("Add")
}

// Clear is an unconditional System.NotSupportedException in the reference.
func (c *TouchCollection) Clear() error {
	return touchNotSupportedError("Clear")
}

// Insert is an unconditional System.NotSupportedException in the reference. It
// does not validate index first.
func (c *TouchCollection) Insert(index int32, item TouchLocation) error {
	return touchNotSupportedError("Insert")
}

// RemoveAt is an unconditional System.NotSupportedException in the reference.
// It does not validate index first.
func (c *TouchCollection) RemoveAt(index int32) error {
	return touchNotSupportedError("RemoveAt")
}

// Remove is an unconditional System.NotSupportedException in the reference. It
// never reports a Boolean result, so the projected bool is always false when
// the error is non-nil.
func (c *TouchCollection) Remove(item TouchLocation) (bool, error) {
	return false, touchNotSupportedError("Remove")
}

// IndexOf scans the stored range in order and compares with the TouchLocation
// **equality operator**, which weighs all seven fields including both state
// fields -- not the looser EqualsByTouchLocation, which ignores them. It
// returns -1 when nothing matches.
func (c TouchCollection) IndexOf(item TouchLocation) int32 {
	for index := int32(0); index < c.locationCount; index++ {
		location, err := c.Item(index)
		if err != nil {
			return -1
		}
		if TouchLocationOperatorEqualityByTouchLocationAndTouchLocation(location, item) {
			return index
		}
	}
	return -1
}

// Contains is the reference's `IndexOf(item) >= 0`, so it inherits the strict
// operator comparison.
func (c TouchCollection) Contains(item TouchLocation) bool {
	return c.IndexOf(item) >= 0
}

// CopyTo reproduces the reference's three checks in order: a nil destination
// throws ArgumentNullException("array"); a negative start throws
// ArgumentOutOfRangeException("arrayIndex"); and an insufficient destination
// throws ArgumentOutOfRangeException("arrayIndex") too.
//
// The capacity check is performed in 64-bit arithmetic exactly as the
// reference does, so a start index near the top of the int32 range reports the
// argument error instead of wrapping into a false pass.
func (c TouchCollection) CopyTo(array []TouchLocation, arrayIndex int32) error {
	if array == nil {
		return touchArgumentNullError("array")
	}
	if arrayIndex < 0 {
		return touchArgumentOutOfRangeError("arrayIndex")
	}
	if int64(len(array)) < int64(arrayIndex)+int64(c.locationCount) {
		return touchArgumentOutOfRangeError("arrayIndex")
	}
	for index := int32(0); index < c.locationCount; index++ {
		location, err := c.Item(index)
		if err != nil {
			return err
		}
		array[int64(arrayIndex)+int64(index)] = location
	}
	return nil
}

// FindById scans the stored range for the first location whose Id matches and
// reports it. It compares identifiers only, ignoring position and state
// entirely, and yields a zero TouchLocation when nothing matches -- not the
// Id -1 sentinel that TouchLocation.TryGetPreviousLocation uses.
func (c TouchCollection) FindById(id int32) (bool, TouchLocation) {
	for index := int32(0); index < c.locationCount; index++ {
		location, err := c.Item(index)
		if err != nil {
			break
		}
		if location.Id() == id {
			return true, location
		}
	}
	return false, TouchLocation{}
}

// GetEnumerator returns a cursor over a copy of this collection, exactly as
// the reference's `ldobj; newobj Enumerator::.ctor` does. Because the
// collection is a value type, the cursor is unaffected by later changes to the
// variable it came from.
//
// The declared public signature returns the concrete nested enumerator rather
// than the Iterator<T> adapter, because XNA declares a concrete public
// enumerator here; the adapter projection applies to collections that declare
// none.
func (c TouchCollection) GetEnumerator() TouchCollectionEnumerator {
	return TouchCollectionEnumerator{collection: c, position: -1}
}

// TouchCollectionEnumerator is the cursor XNA nests inside TouchCollection. It
// holds a copy of the collection and a position that starts before the first
// element.
type TouchCollectionEnumerator struct {
	collection TouchCollection
	position   int32
}

// Current forwards to the collection indexer, so it carries that indexer's
// argument validation: it reports an error before the first MoveNext, when the
// position is still -1, and again once MoveNext has exhausted the cursor and
// clamped the position to Count.
func (e TouchCollectionEnumerator) Current() (TouchLocation, error) {
	return e.collection.Item(e.position)
}

// MoveNext advances the cursor. On exhaustion the reference clamps the
// position to Count rather than letting it run past, so repeated calls after
// the end keep reporting false without drifting.
func (e *TouchCollectionEnumerator) MoveNext() bool {
	e.position++
	if e.position < e.collection.locationCount {
		return true
	}
	e.position = e.collection.locationCount
	return false
}

// Dispose is a bare `ret` in the reference. It releases nothing because the
// cursor owns nothing.
func (e *TouchCollectionEnumerator) Dispose() {}
