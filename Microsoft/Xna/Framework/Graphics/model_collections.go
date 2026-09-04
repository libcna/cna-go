package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The four Model collections, which share one shape:
//
//	.class public auto ansi sealed beforefieldinit ModelBoneCollection
//	       extends [mscorlib]System.Collections.ObjectModel.ReadOnlyCollection`1<ModelBone>
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # How the CLR base is carried
//
// Go has no inheritance, so each collection holds the base as an unexported
// field -- an instance of the framework language adapter that projects
// ReadOnlyCollection<T> -- and re-exposes the inherited public members as
// measured forwarding methods. The adapter is never embedded, never exported
// and never returned. This is the settled composition rule, and the same one
// GameComponentCollection already uses for Collection<T>.
//
// # Each collection keeps its OWN backing field beside the base's
//
// The reference does exactly this. Its constructor is
//
//	.ctor(ModelBone[] bones)
//	    base..ctor(bones);  this.wrappedArray = bones
//
// so the base and the derived class point at the SAME object, and the derived
// members -- TryGetValue, the by-name indexer, GetEnumerator -- read the
// derived field while the inherited members read the base. The projection
// keeps both, because collapsing them would hide that the reference can tell
// them apart.
//
// # Three wrap an ARRAY and one wraps a LIST, and it is observable
//
//	ModelBoneCollection      ModelBone[]
//	ModelMeshCollection      ModelMesh[]
//	ModelMeshPartCollection  ModelMeshPart[]
//	ModelEffectCollection    List<Effect>       <- the odd one
//
// ModelEffectCollection is the only one that MUTATES: it declares `assembly`
// Add and Remove, which is what ModelMeshPart.set_Effect calls. Two
// consequences follow, and both are reproduced:
//
//   - its view must be LIVE, because the CLR view holds the List reference and
//     sees every addition; the three array-backed views cannot change length.
//   - its nested Enumerator wraps List<Effect>.Enumerator, which is
//     VERSION-CHECKED and fails once the list is mutated. The array-backed
//     enumerators have no version and cannot detect anything.
//
// So enumerating a mesh's Effects while assigning a part's Effect fails in the
// reference, and doing the same to Bones, Meshes or MeshParts does not.
//
// # The hidden GetEnumerator
//
// All four declare `public hidebysig instance GetEnumerator()` returning their
// own nested Enumerator, with no `virtual` and no `newslot` -- C# `new`, which
// HIDES ReadOnlyCollection<T>.GetEnumerator() rather than overriding it.
//
// Reaching a hidden base member in C# requires a cast to the base. CNA-Go
// projects no base type to cast to, so the inherited GetEnumerator is
// unreachable and is not projected: each collection has exactly one Go method
// named GetEnumerator, the derived one. This is the rule the ReadOnlyCollection
// base blocker named as available-but-untested, and this milestone is what
// tests it.

// Sentinel errors these collections answer. They are unexported for the reason
// every other family's are: the XNA contract declares no error type here.
var (
	// errModelArgumentNull projects System.ArgumentNullException.
	errModelArgumentNull = errors.New("model argument is nil")
	// errModelEnumerationFailed projects the System.InvalidOperationException
	// List<T>.Enumerator.MoveNextRare throws once the list has changed. Only
	// ModelEffectCollection can answer it: its three sibling collections wrap
	// an array, which has no version and cannot change length.
	errModelEnumerationFailed = errors.New("model effect collection changed during enumeration")
	// errModelKeyNotFound projects System.Collections.Generic.KeyNotFoundException,
	// which the by-name indexers throw through its PARAMETERLESS constructor --
	// so the reference supplies no message of its own and neither does this.
	errModelKeyNotFound = errors.New("the given key was not present in the collection")
)

func modelArgumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errModelArgumentNull, parameter)
}

// ---------------------------------------------------------------------------
// ModelBoneCollection

// ModelBoneCollection is Model.Bones and ModelBone.Children.
type ModelBoneCollection struct {
	base         framework.ReadOnlyCollection[*ModelBone]
	wrappedArray []*ModelBone
}

func newModelBoneCollection(bones []*ModelBone) *ModelBoneCollection {
	return &ModelBoneCollection{
		base:         *framework.NewReadOnlyCollectionOverReferences(bones),
		wrappedArray: bones,
	}
}

// Count is the inherited ReadOnlyCollection<T>::get_Count.
func (c *ModelBoneCollection) Count() int32 { return c.base.Count() }

// Item is the inherited ReadOnlyCollection<T>::get_Item(Int32).
func (c *ModelBoneCollection) ItemPropertySignature854B41ED(index int32) (*ModelBone, error) {
	return c.base.Item(index)
}

// Contains is the inherited ReadOnlyCollection<T>::Contains.
func (c *ModelBoneCollection) Contains(value *ModelBone) bool { return c.base.Contains(value) }

// IndexOf is the inherited ReadOnlyCollection<T>::IndexOf.
func (c *ModelBoneCollection) IndexOf(value *ModelBone) int32 { return c.base.IndexOf(value) }

// CopyTo is the inherited ReadOnlyCollection<T>::CopyTo.
func (c *ModelBoneCollection) CopyTo(array []*ModelBone, index int32) error {
	return c.base.CopyTo(array, index)
}

// ItemByName is ModelBoneCollection::get_Item(String), the by-name indexer,
// whose whole body is
//
//	if (!TryGetValue(boneName, out bone)) throw new KeyNotFoundException();
//	return bone;
//
// The exception is the PARAMETERLESS KeyNotFoundException, so it carries no
// message naming the bone. A null or empty name reaches TryGetValue's own
// guard first and answers that refusal instead.
func (c *ModelBoneCollection) ItemPropertySignatureC23A10DE(boneName string) (*ModelBone, error) {
	found, bone, err := c.TryGetValue(boneName)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errModelKeyNotFound
	}
	return bone, nil
}

// TryGetValue is ModelBoneCollection::TryGetValue, measured at 81 bytes:
//
//	if (String.IsNullOrEmpty(boneName))
//	    throw new ArgumentNullException("boneName");
//	for (int i = 0; i < Items.Count; i++)
//	    if (String.Compare(Items[i].Name, boneName, StringComparison.Ordinal) == 0) {
//	        value = Items[i]; return true;
//	    }
//	value = null; return false;
//
// Three measured details.
//
// The guard is IsNullOrEmpty, so the EMPTY string is refused too -- and it is
// refused with ArgumentNullException rather than ArgumentException, which only
// the IL settles. Go has no nil string, so empty is the whole guard.
//
// The comparison is `ldc.i4.4`, StringComparison.Ordinal: CASE SENSITIVE, not
// the OrdinalIgnoreCase a reader might assume from a name lookup.
//
// And the scan reads the BASE's protected `Items`, not the derived
// `wrappedArray`. The two hold the same object here, but the projection reads
// what the reference reads, so a future divergence would show up rather than
// be papered over. GetEnumerator is the member that genuinely reads
// wrappedArray.
//
// The CLR `out` parameter projects as a second result, which is the settled
// TryGetValue shape; a not-found lookup is not an error and answers false.
func (c *ModelBoneCollection) TryGetValue(boneName string) (bool, *ModelBone, error) {
	if boneName == "" {
		return false, nil, modelArgumentNullError("boneName")
	}
	count := c.base.Count()
	for index := int32(0); index < count; index++ {
		bone, err := c.base.Item(index)
		if err != nil {
			return false, nil, err
		}
		if bone != nil && bone.Name() == boneName {
			return true, bone, nil
		}
	}
	return false, nil, nil
}

// GetEnumerator is ModelBoneCollection::GetEnumerator, which HIDES the
// inherited one and returns the collection's own nested Enumerator over the
// wrapped array.
func (c *ModelBoneCollection) GetEnumerator() ModelBoneCollectionEnumerator {
	return ModelBoneCollectionEnumerator{wrappedArray: c.wrappedArray, position: -1}
}

// ModelBoneCollectionEnumerator is ModelBoneCollection+Enumerator:
//
//	.class sequential ansi sealed nested public beforefieldinit Enumerator
//	       extends System.ValueType
//	    .field private ModelBone[] wrappedArray
//	    .field private int32 position
//
// A value struct over the array and a cursor, with NO version field -- the
// reference has none, because an array's length cannot change.
type ModelBoneCollectionEnumerator struct {
	wrappedArray []*ModelBone
	position     int32
}

// Current is Enumerator::get_Current.
func (e ModelBoneCollectionEnumerator) Current() *ModelBone {
	if e.position < 0 || int(e.position) >= len(e.wrappedArray) {
		return nil
	}
	return e.wrappedArray[e.position]
}

// MoveNext is Enumerator::MoveNext.
func (e *ModelBoneCollectionEnumerator) MoveNext() bool {
	if int(e.position)+1 >= len(e.wrappedArray) {
		e.position = int32(len(e.wrappedArray))
		return false
	}
	e.position++
	return true
}

// Dispose is Enumerator::Dispose, whose body is one `ret`.
func (e *ModelBoneCollectionEnumerator) Dispose() {}

// ---------------------------------------------------------------------------
// ModelMeshCollection
//
// The same shape as ModelBoneCollection, including the by-name lookup, over
// ModelMesh.

// ModelMeshCollection is Model.Meshes.
type ModelMeshCollection struct {
	base         framework.ReadOnlyCollection[*ModelMesh]
	wrappedArray []*ModelMesh
}

func newModelMeshCollection(meshes []*ModelMesh) *ModelMeshCollection {
	return &ModelMeshCollection{
		base:         *framework.NewReadOnlyCollectionOverReferences(meshes),
		wrappedArray: meshes,
	}
}

// Count is the inherited ReadOnlyCollection<T>::get_Count.
func (c *ModelMeshCollection) Count() int32 { return c.base.Count() }

// Item is the inherited ReadOnlyCollection<T>::get_Item(Int32).
func (c *ModelMeshCollection) ItemPropertySignature854B41ED(index int32) (*ModelMesh, error) {
	return c.base.Item(index)
}

// Contains is the inherited ReadOnlyCollection<T>::Contains.
func (c *ModelMeshCollection) Contains(value *ModelMesh) bool { return c.base.Contains(value) }

// IndexOf is the inherited ReadOnlyCollection<T>::IndexOf.
func (c *ModelMeshCollection) IndexOf(value *ModelMesh) int32 { return c.base.IndexOf(value) }

// CopyTo is the inherited ReadOnlyCollection<T>::CopyTo.
func (c *ModelMeshCollection) CopyTo(array []*ModelMesh, index int32) error {
	return c.base.CopyTo(array, index)
}

// ItemByName is ModelMeshCollection::get_Item(String); see
// ModelBoneCollection.ItemByName, whose body it matches instruction for
// instruction over a different element type.
func (c *ModelMeshCollection) ItemPropertySignature69F0C25F(meshName string) (*ModelMesh, error) {
	found, mesh, err := c.TryGetValue(meshName)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errModelKeyNotFound
	}
	return mesh, nil
}

// TryGetValue is ModelMeshCollection::TryGetValue, the same measured body as
// ModelBoneCollection's: IsNullOrEmpty guarded with ArgumentNullException, and
// an ORDINAL, case-sensitive name comparison over the base's Items.
func (c *ModelMeshCollection) TryGetValue(meshName string) (bool, *ModelMesh, error) {
	if meshName == "" {
		return false, nil, modelArgumentNullError("meshName")
	}
	count := c.base.Count()
	for index := int32(0); index < count; index++ {
		mesh, err := c.base.Item(index)
		if err != nil {
			return false, nil, err
		}
		if mesh != nil && mesh.Name() == meshName {
			return true, mesh, nil
		}
	}
	return false, nil, nil
}

// GetEnumerator is ModelMeshCollection::GetEnumerator, which hides the
// inherited one.
func (c *ModelMeshCollection) GetEnumerator() ModelMeshCollectionEnumerator {
	return ModelMeshCollectionEnumerator{wrappedArray: c.wrappedArray, position: -1}
}

// ModelMeshCollectionEnumerator is ModelMeshCollection+Enumerator, a value
// struct over the array and a cursor with no version field.
type ModelMeshCollectionEnumerator struct {
	wrappedArray []*ModelMesh
	position     int32
}

// Current is Enumerator::get_Current.
func (e ModelMeshCollectionEnumerator) Current() *ModelMesh {
	if e.position < 0 || int(e.position) >= len(e.wrappedArray) {
		return nil
	}
	return e.wrappedArray[e.position]
}

// MoveNext is Enumerator::MoveNext.
func (e *ModelMeshCollectionEnumerator) MoveNext() bool {
	if int(e.position)+1 >= len(e.wrappedArray) {
		e.position = int32(len(e.wrappedArray))
		return false
	}
	e.position++
	return true
}

// Dispose is Enumerator::Dispose, one `ret`.
func (e *ModelMeshCollectionEnumerator) Dispose() {}

// ---------------------------------------------------------------------------
// ModelMeshPartCollection
//
// No by-name lookup: a mesh part has no Name.

// ModelMeshPartCollection is ModelMesh.MeshParts.
type ModelMeshPartCollection struct {
	base         framework.ReadOnlyCollection[*ModelMeshPart]
	wrappedArray []*ModelMeshPart
}

func newModelMeshPartCollection(parts []*ModelMeshPart) *ModelMeshPartCollection {
	return &ModelMeshPartCollection{
		base:         *framework.NewReadOnlyCollectionOverReferences(parts),
		wrappedArray: parts,
	}
}

// Count is the inherited ReadOnlyCollection<T>::get_Count.
func (c *ModelMeshPartCollection) Count() int32 { return c.base.Count() }

// Item is the inherited ReadOnlyCollection<T>::get_Item(Int32).
func (c *ModelMeshPartCollection) Item(index int32) (*ModelMeshPart, error) {
	return c.base.Item(index)
}

// Contains is the inherited ReadOnlyCollection<T>::Contains.
func (c *ModelMeshPartCollection) Contains(value *ModelMeshPart) bool {
	return c.base.Contains(value)
}

// IndexOf is the inherited ReadOnlyCollection<T>::IndexOf.
func (c *ModelMeshPartCollection) IndexOf(value *ModelMeshPart) int32 {
	return c.base.IndexOf(value)
}

// CopyTo is the inherited ReadOnlyCollection<T>::CopyTo.
func (c *ModelMeshPartCollection) CopyTo(array []*ModelMeshPart, index int32) error {
	return c.base.CopyTo(array, index)
}

// GetEnumerator is ModelMeshPartCollection::GetEnumerator, which hides the
// inherited one.
func (c *ModelMeshPartCollection) GetEnumerator() ModelMeshPartCollectionEnumerator {
	return ModelMeshPartCollectionEnumerator{wrappedArray: c.wrappedArray, position: -1}
}

// ModelMeshPartCollectionEnumerator is ModelMeshPartCollection+Enumerator.
type ModelMeshPartCollectionEnumerator struct {
	wrappedArray []*ModelMeshPart
	position     int32
}

// Current is Enumerator::get_Current.
func (e ModelMeshPartCollectionEnumerator) Current() *ModelMeshPart {
	if e.position < 0 || int(e.position) >= len(e.wrappedArray) {
		return nil
	}
	return e.wrappedArray[e.position]
}

// MoveNext is Enumerator::MoveNext.
func (e *ModelMeshPartCollectionEnumerator) MoveNext() bool {
	if int(e.position)+1 >= len(e.wrappedArray) {
		e.position = int32(len(e.wrappedArray))
		return false
	}
	e.position++
	return true
}

// Dispose is Enumerator::Dispose, one `ret`.
func (e *ModelMeshPartCollectionEnumerator) Dispose() {}

// ---------------------------------------------------------------------------
// ModelEffectCollection
//
// The one collection that mutates, and the one backed by a List rather than an
// array.

// ModelEffectCollection is ModelMesh.Effects.
//
// Its backing field is `List<Effect> wrappedList`, and its `assembly` Add and
// Remove are what ModelMeshPart.set_Effect calls as parts change which effect
// they use. The base view is therefore LIVE over the list -- the CLR view holds
// the List reference and sees every addition -- which is why it is built with
// the live constructor rather than the array one.
type ModelEffectCollection struct {
	base        framework.ReadOnlyCollection[*Effect]
	wrappedList []*Effect
}

func newModelEffectCollection(effects []*Effect) *ModelEffectCollection {
	collection := &ModelEffectCollection{wrappedList: effects}
	// The closure is what the CLR's stored List REFERENCE is: a way to ask the
	// owner for its current list every time, so an Add or a Remove is visible
	// through Count and the indexer immediately.
	collection.base = *framework.NewReadOnlyCollectionOverLiveReferences(
		func() []*Effect { return collection.wrappedList })
	return collection
}

// Count is the inherited ReadOnlyCollection<T>::get_Count, live over the list.
func (c *ModelEffectCollection) Count() int32 { return c.base.Count() }

// Item is the inherited ReadOnlyCollection<T>::get_Item(Int32).
//
// The result widens to EffectReference because Effect is a substitutable base
// at RETURN positions too: a C# caller assigns the Effect this hands back to a
// variable of a derived type only after a cast, but every member of Effect must
// stay reachable, and the interface is what carries them.
func (c *ModelEffectCollection) Item(index int32) (EffectReference, error) {
	effect, err := c.base.Item(index)
	if err != nil {
		return nil, err
	}
	if effect == nil {
		return nil, nil
	}
	return effect, nil
}

// Contains is the inherited ReadOnlyCollection<T>::Contains.
func (c *ModelEffectCollection) Contains(value EffectReference) bool {
	return c.IndexOf(value) >= 0
}

// IndexOf is the inherited ReadOnlyCollection<T>::IndexOf.
func (c *ModelEffectCollection) IndexOf(value EffectReference) int32 {
	// The parameter widens to EffectReference because Effect is a substitutable
	// base: a C# caller passes a BasicEffect where an Effect is wanted, and the
	// settled rule gives that position the interface. The comparison is still
	// EqualityComparer<Effect>.Default, which for a class overriding no Equals
	// is reference identity -- so a derived effect matches only the very object
	// the collection holds.
	for index, candidate := range c.wrappedList {
		if effectReferenceIdentity(candidate, value) {
			return int32(index)
		}
	}
	return -1
}

// effectReferenceIdentity is System.Object::ReferenceEquals across the
// substitutable-base interface: two references are the same object when the
// concrete Effect behind them is the same pointer.
func effectReferenceIdentity(candidate *Effect, value EffectReference) bool {
	if value == nil {
		return candidate == nil
	}
	other, ok := any(value).(*Effect)
	if ok {
		return candidate == other
	}
	return false
}

// CopyTo is the inherited ReadOnlyCollection<T>::CopyTo.
func (c *ModelEffectCollection) CopyTo(array []*Effect, index int32) error {
	return c.base.CopyTo(array, index)
}

// GetEnumerator is ModelEffectCollection::GetEnumerator, which hides the
// inherited one and returns an enumerator over List<Effect>.Enumerator -- so
// unlike its three siblings this one is VERSION-CHECKED and fails once the
// list has been mutated.
func (c *ModelEffectCollection) GetEnumerator() ModelEffectCollectionEnumerator {
	return ModelEffectCollectionEnumerator{
		collection: c,
		version:    len(c.wrappedList),
		position:   -1,
	}
}

// add is ModelEffectCollection::Add, `assembly` in the reference and therefore
// unexported here: no consumer may grow a mesh's effect list directly, and the
// only caller is ModelMeshPart.SetEffect.
func (c *ModelEffectCollection) add(effect *Effect) {
	c.wrappedList = append(c.wrappedList, effect)
}

// remove is ModelEffectCollection::Remove, `assembly` for the same reason. The
// reference's List<T>.Remove drops the FIRST match by
// EqualityComparer<Effect>.Default, which for a class that overrides no Equals
// is reference identity, and leaves the list unchanged when nothing matches.
func (c *ModelEffectCollection) remove(effect *Effect) {
	for index, candidate := range c.wrappedList {
		if candidate == effect {
			c.wrappedList = append(c.wrappedList[:index], c.wrappedList[index+1:]...)
			return
		}
	}
}

// ModelEffectCollectionEnumerator is ModelEffectCollection+Enumerator:
//
//	.field private List`1/Enumerator<Effect> internalEnumerator
//
// It wraps the LIST's enumerator rather than an array, which is what makes it
// fail fast. The projection carries the length the list had when the
// enumerator was taken, because every mutator this collection declares -- Add
// and Remove, and nothing else -- changes it.
type ModelEffectCollectionEnumerator struct {
	collection *ModelEffectCollection
	version    int
	position   int32
}

// Current is Enumerator::get_Current, widened to EffectReference for the reason
// the indexer is.
func (e ModelEffectCollectionEnumerator) Current() EffectReference {
	if e.collection == nil || e.position < 0 || int(e.position) >= len(e.collection.wrappedList) {
		return nil
	}
	effect := e.collection.wrappedList[e.position]
	if effect == nil {
		return nil
	}
	return effect
}

// MoveNext is Enumerator::MoveNext, which forwards to
// List<Effect>.Enumerator.MoveNext and therefore throws
// InvalidOperationException once the list has changed.
func (e *ModelEffectCollectionEnumerator) MoveNext() (bool, error) {
	if e.collection == nil {
		return false, nil
	}
	if len(e.collection.wrappedList) != e.version {
		return false, errModelEnumerationFailed
	}
	if int(e.position)+1 >= len(e.collection.wrappedList) {
		e.position = int32(len(e.collection.wrappedList))
		return false, nil
	}
	e.position++
	return true, nil
}

// Dispose is Enumerator::Dispose, one `ret`.
func (e *ModelEffectCollectionEnumerator) Dispose() {}
