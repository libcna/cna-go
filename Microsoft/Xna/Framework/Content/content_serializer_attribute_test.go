package content

import (
	"reflect"
	"strings"
	"testing"
)

// TestContentSerializerAttributeDefaultsAllowNullToTrue is the constructor's
// only instruction before the base call. Five of the six fields take their zero
// value and allowNull does not, which is exactly what an implementation that
// zeroed all six would get wrong.
func TestContentSerializerAttributeDefaultsAllowNullToTrue(t *testing.T) {
	attribute := NewContentSerializerAttribute()
	if !attribute.AllowNull() {
		t.Fatal("AllowNull defaulted to false; the constructor stores ldc.i4.1")
	}
	if attribute.ElementName() != "" {
		t.Fatalf("ElementName defaulted to %q; want the empty string", attribute.ElementName())
	}
	if attribute.FlattenContent() || attribute.Optional() || attribute.SharedResource() {
		t.Fatal("a bool field other than allowNull defaulted to true")
	}
	if attribute.HasCollectionItemName() {
		t.Fatal("a fresh attribute reported a collection item name")
	}
}

// TestCollectionItemNameSubstitutesItemForAnEmptyField pins the one member of
// this type that is not a field read. The getter answers "Item" for an empty
// field, and HasCollectionItemName reads the FIELD, so the two disagree in
// exactly the case the reference makes them disagree.
func TestCollectionItemNameSubstitutesItemForAnEmptyField(t *testing.T) {
	attribute := NewContentSerializerAttribute()
	if got := attribute.CollectionItemName(); got != "Item" {
		t.Fatalf("CollectionItemName = %q; want the literal \"Item\"", got)
	}
	if attribute.HasCollectionItemName() {
		t.Fatal("HasCollectionItemName read the property rather than the field")
	}

	if err := attribute.SetCollectionItemName("Entry"); err != nil {
		t.Fatalf("SetCollectionItemName: %v", err)
	}
	if got := attribute.CollectionItemName(); got != "Entry" {
		t.Fatalf("CollectionItemName = %q; want the stored name", got)
	}
	if !attribute.HasCollectionItemName() {
		t.Fatal("HasCollectionItemName stayed false after a name was stored")
	}

	// Setting the name to "Item" explicitly is NOT the same state as leaving it
	// unset, which is the whole reason HasCollectionItemName exists.
	explicit := NewContentSerializerAttribute()
	if err := explicit.SetCollectionItemName("Item"); err != nil {
		t.Fatalf("SetCollectionItemName: %v", err)
	}
	if !explicit.HasCollectionItemName() {
		t.Fatal("an explicitly stored \"Item\" did not set the field")
	}
}

// TestCollectionItemNameSetterRefusesEmpty pins the guard AND the parameter
// name. The reference names "value" -- the compiler's name for a setter's
// argument -- not the property.
func TestCollectionItemNameSetterRefusesEmpty(t *testing.T) {
	attribute := NewContentSerializerAttribute()
	err := attribute.SetCollectionItemName("")
	if err == nil {
		t.Fatal("an empty collection item name was accepted")
	}
	if !strings.Contains(err.Error(), "value") {
		t.Fatalf("the refusal was %q; the reference names the setter's argument, \"value\"", err)
	}
	if strings.Contains(err.Error(), "collectionItemName") {
		t.Fatalf("the refusal was %q; that is the CONSTRUCTOR's parameter name on a different type", err)
	}
	// The refused setter stored nothing -- and the case that shows it is one
	// where the field ALREADY holds a name, because an empty field is what a
	// store-then-refuse implementation would leave behind anyway.
	if err = attribute.SetCollectionItemName("Entry"); err != nil {
		t.Fatalf("SetCollectionItemName: %v", err)
	}
	if err = attribute.SetCollectionItemName(""); err == nil {
		t.Fatal("an empty collection item name was accepted over a stored one")
	}
	if attribute.CollectionItemName() != "Entry" {
		t.Fatalf("the refused set left %q; it must store nothing", attribute.CollectionItemName())
	}
	if !attribute.HasCollectionItemName() {
		t.Fatal("the refused set cleared the stored name")
	}
}

// TestElementNameSetterHasNoGuard is the contrast that makes the previous test
// about a measured guard rather than about strings generally. set_ElementName
// is one stfld and accepts anything.
func TestElementNameSetterHasNoGuard(t *testing.T) {
	attribute := NewContentSerializerAttribute()
	attribute.SetElementName("Node")
	if attribute.ElementName() != "Node" {
		t.Fatal("the element name was not stored")
	}
	// Clearing it is the case a guard would refuse, and the reference has no
	// guard: set_ElementName is one stfld.
	attribute.SetElementName("")
	if attribute.ElementName() != "" {
		t.Fatal("the element name could not be cleared; set_ElementName has no guard")
	}
}

// TestCloneCopiesFieldsAndNotProperties is the detail Clone turns on. It copies
// collectionItemName as the FIELD, so a clone of an attribute with no name set
// keeps the empty field -- and HasCollectionItemName stays false on both.
// Copying through the property would store "Item" and flip it to true.
func TestCloneCopiesFieldsAndNotProperties(t *testing.T) {
	source := NewContentSerializerAttribute()
	clone := source.Clone()
	if clone == source {
		t.Fatal("Clone answered the same instance; the reference constructs a new one")
	}
	if clone.HasCollectionItemName() {
		t.Fatal("the clone reported a collection item name the source did not have")
	}
	if got := clone.CollectionItemName(); got != "Item" {
		t.Fatalf("the clone's CollectionItemName = %q; want the same \"Item\" fallback", got)
	}

	// Every one of the six fields round-trips, with values that differ from
	// both the zero value and the constructor's default.
	// The three remaining bools are given DIFFERENT values, so a copy that
	// reads the wrong source field is visible. All-true would hide it.
	source.SetElementName("Node")
	source.SetFlattenContent(false)
	source.SetOptional(true)
	source.SetAllowNull(false)
	source.SetSharedResource(true)
	if err := source.SetCollectionItemName("Entry"); err != nil {
		t.Fatalf("SetCollectionItemName: %v", err)
	}
	full := source.Clone()
	switch {
	case full.ElementName() != "Node":
		t.Fatal("Clone dropped elementName")
	case full.FlattenContent():
		t.Fatal("Clone set flattenContent from another field")
	case !full.Optional():
		t.Fatal("Clone dropped optional")
	case full.AllowNull():
		t.Fatal("Clone dropped allowNull; the constructor sets it true and the copy must overwrite it")
	case !full.SharedResource():
		t.Fatal("Clone dropped sharedResource")
	case full.CollectionItemName() != "Entry":
		t.Fatal("Clone dropped collectionItemName")
	}

	// The clone is independent.
	full.SetElementName("Other")
	if source.ElementName() != "Node" {
		t.Fatal("mutating the clone reached the source")
	}

	// The SECOND half of the bool coverage. The case above gave flattenContent
	// false and optional true; this one swaps them, so a Clone that drops
	// either copy is visible in one of the two -- with only one arrangement, a
	// dropped copy of a false field looks identical to a correct one.
	swapped := NewContentSerializerAttribute()
	swapped.SetFlattenContent(true)
	swapped.SetOptional(false)
	swapped.SetAllowNull(true)
	swapped.SetSharedResource(false)
	swappedClone := swapped.Clone()
	switch {
	case !swappedClone.FlattenContent():
		t.Fatal("Clone dropped flattenContent")
	case swappedClone.Optional():
		t.Fatal("Clone set optional from another field")
	case !swappedClone.AllowNull():
		t.Fatal("Clone dropped allowNull")
	case swappedClone.SharedResource():
		t.Fatal("Clone set sharedResource from another field")
	}
}

// TestTheInheritedEqualsComparesFieldsNotReferences is System.Attribute's own
// behaviour, and it is the opposite of what Go's == on two pointers answers.
func TestTheInheritedEqualsComparesFieldsNotReferences(t *testing.T) {
	first := NewContentSerializerAttribute()
	second := NewContentSerializerAttribute()
	if first == second {
		t.Fatal("the fixture is broken: two constructions produced one pointer")
	}
	if !first.Equals(second) {
		t.Fatal("two separately built attributes with equal fields are not equal; the reference compares FIELD BY FIELD")
	}
	if first.GetHashCode() != second.GetHashCode() {
		t.Fatal("equal attributes hashed differently")
	}
	// The hash comes from the runtime TYPE, so two attribute types must not
	// share one -- which is what separates it from a constant.
	if first.GetHashCode() == NewContentSerializerIgnoreAttribute().GetHashCode() {
		t.Fatal("two attribute types share a hash; the reference hashes the runtime type")
	}
	if first.GetHashCode() == 0 {
		t.Fatal("the hash is zero, which a constant implementation would also answer")
	}

	second.SetOptional(true)
	if first.Equals(second) {
		t.Fatal("attributes differing in one field compared equal")
	}

	// A different attribute TYPE is never equal, even with no fields to
	// disagree about.
	if first.Equals(NewContentSerializerIgnoreAttribute()) {
		t.Fatal("two different attribute types compared equal")
	}
	if first.Equals(nil) {
		t.Fatal("an attribute equalled nil")
	}

	// Match's base body is `return this.Equals(obj)`.
	if !first.Match(NewContentSerializerAttribute()) {
		t.Fatal("Match disagreed with Equals")
	}
	if first.Match(nil) {
		t.Fatal("Match accepted nil")
	}
}

// TestIsDefaultAttributeIsAlwaysFalse pins the base's `ldc.i4.0`. None of the
// five overrides it, so even a default-constructed attribute answers false --
// which is what an implementation reading the name would get wrong.
func TestIsDefaultAttributeIsAlwaysFalse(t *testing.T) {
	if NewContentSerializerAttribute().IsDefaultAttribute() {
		t.Fatal("a default-constructed ContentSerializerAttribute called itself the default")
	}
	if NewContentSerializerIgnoreAttribute().IsDefaultAttribute() {
		t.Fatal("a stateless attribute called itself the default")
	}
	if NewContentSerializerTypeVersionAttribute(0).IsDefaultAttribute() {
		t.Fatal("a zero type version called itself the default")
	}
}

// TestTypeIdAnswersTheRuntimeType pins that the object-typed property really
// carries the runtime type, which is what the default implementation returns.
func TestTypeIdAnswersTheRuntimeType(t *testing.T) {
	attribute := NewContentSerializerAttribute()
	id := attribute.TypeId()
	typed, ok := id.(reflect.Type)
	if !ok {
		t.Fatalf("TypeId answered %T; the default returns GetType()", id)
	}
	if typed != reflect.TypeOf(attribute) {
		t.Fatalf("TypeId answered %v; want the runtime type", typed)
	}
	// Two attribute types answer differently, which is what makes it an
	// identity rather than a constant.
	if NewContentSerializerIgnoreAttribute().TypeId() == id {
		t.Fatal("two attribute types share a TypeId")
	}
}

// TestTheFourSmallAttributesGuardExactlyWhatTheReferenceGuards is the family's
// asymmetry, measured: the two string constructors refuse an empty value and
// the Int32 one does not.
func TestTheFourSmallAttributesGuardExactlyWhatTheReferenceGuards(t *testing.T) {
	name, err := NewContentSerializerCollectionItemNameAttribute("Entry")
	if err != nil {
		t.Fatalf("NewContentSerializerCollectionItemNameAttribute: %v", err)
	}
	if name.CollectionItemName() != "Entry" {
		t.Fatalf("CollectionItemName = %q", name.CollectionItemName())
	}
	if _, err = NewContentSerializerCollectionItemNameAttribute(""); err == nil {
		t.Fatal("an empty collection item name was accepted")
	} else if !strings.Contains(err.Error(), "collectionItemName") {
		t.Fatalf("the refusal was %q; the reference names the constructor's parameter", err)
	}

	runtimeType, err := NewContentSerializerRuntimeTypeAttribute("Some.Type")
	if err != nil {
		t.Fatalf("NewContentSerializerRuntimeTypeAttribute: %v", err)
	}
	if runtimeType.RuntimeType() != "Some.Type" {
		t.Fatalf("RuntimeType = %q", runtimeType.RuntimeType())
	}
	if _, err = NewContentSerializerRuntimeTypeAttribute(""); err == nil {
		t.Fatal("an empty runtime type was accepted")
	} else if !strings.Contains(err.Error(), "runtimeType") {
		t.Fatalf("the refusal was %q; the reference names the constructor's parameter", err)
	}

	// The Int32 constructor has NO guard, so a negative version is stored.
	version := NewContentSerializerTypeVersionAttribute(-3)
	if version.TypeVersion() != -3 {
		t.Fatalf("TypeVersion = %d; the reference stores the argument without checking it", version.TypeVersion())
	}

	// The standalone collection item name attribute has NO "Item" fallback --
	// the constructor guarantees the field is non-empty, so the getter is one
	// field read. That is the difference from ContentSerializerAttribute's.
	if _, err = NewContentSerializerCollectionItemNameAttribute(""); err == nil {
		t.Fatal("the fixture is broken")
	}
}

// TestTheStatelessAttributeEqualsEveryOtherOfItsType is the degenerate case of
// the field walk: with nothing to compare, every instance agrees.
func TestTheStatelessAttributeEqualsEveryOtherOfItsType(t *testing.T) {
	first, second := NewContentSerializerIgnoreAttribute(), NewContentSerializerIgnoreAttribute()
	if !first.Equals(second) {
		t.Fatal("two stateless attributes of one type were not equal")
	}
	if first.GetHashCode() != second.GetHashCode() {
		t.Fatal("two stateless attributes hashed differently")
	}
	// And two attributes of different types that BOTH carry one string field
	// are still not equal, because the type check comes first.
	name, _ := NewContentSerializerCollectionItemNameAttribute("Entry")
	runtimeType, _ := NewContentSerializerRuntimeTypeAttribute("Entry")
	if name.Equals(runtimeType) {
		t.Fatal("two different attribute types with equal field values compared equal")
	}
	// Two of the SAME type with equal fields are equal.
	other, _ := NewContentSerializerCollectionItemNameAttribute("Entry")
	if !name.Equals(other) {
		t.Fatal("two collection item name attributes with the same name were not equal")
	}
	different, _ := NewContentSerializerCollectionItemNameAttribute("Other")
	if name.Equals(different) {
		t.Fatal("two collection item name attributes with different names compared equal")
	}
}
