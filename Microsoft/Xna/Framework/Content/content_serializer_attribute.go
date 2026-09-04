package content

// ContentSerializerAttribute is
// Microsoft.Xna.Framework.Content.ContentSerializerAttribute:
//
//	.class public auto ansi sealed beforefieldinit ContentSerializerAttribute
//	       extends [mscorlib]System.Attribute
//
// The attribute the intermediate serializer reads to decide how one member is
// written and read back.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It can be built and read, and it cannot be ATTACHED
//
// Go has no attribute metadata. Every member below is exactly what the pinned
// contract declares, and a consumer can construct one, set it and read it back;
// what a consumer cannot do is annotate a declaration with it, because Go has
// no syntax for that and no metadata to hold it. See attributeBase for why that
// is narrower than it sounds -- the runtime's own readers are private, and CNA
// does the type-reader dispatch.
type ContentSerializerAttribute struct {
	// base is the private System.Attribute adapter, held by the settled
	// composition rule.
	base attributeBase
	// The six fields the reference declares, in its own order.
	elementName        string
	flattenContent     bool
	optional           bool
	allowNull          bool
	sharedResource     bool
	collectionItemName string
}

// NewContentSerializerAttribute is ContentSerializerAttribute::.ctor(), whose
// body is TWO instructions before the base call:
//
//	ldarg.0; ldc.i4.1; stfld allowNull
//	ldarg.0; call System.Attribute::.ctor()
//
// So AllowNull defaults to TRUE. Every other field takes its zero value, which
// is what a plausible implementation would give all six.
func NewContentSerializerAttribute() *ContentSerializerAttribute {
	return &ContentSerializerAttribute{allowNull: true}
}

// attributeSelf installs the CLR `this` for the inherited members.
func (a *ContentSerializerAttribute) attributeSelf() any { return a }

// ElementName is get_ElementName, one ldfld.
func (a *ContentSerializerAttribute) ElementName() string { return a.elementName }

// SetElementName is set_ElementName, one stfld with NO guard -- unlike
// CollectionItemName's setter, which refuses an empty value.
func (a *ContentSerializerAttribute) SetElementName(value string) { a.elementName = value }

// FlattenContent is get_FlattenContent.
func (a *ContentSerializerAttribute) FlattenContent() bool { return a.flattenContent }

// SetFlattenContent is set_FlattenContent.
func (a *ContentSerializerAttribute) SetFlattenContent(value bool) { a.flattenContent = value }

// Optional is get_Optional.
func (a *ContentSerializerAttribute) Optional() bool { return a.optional }

// SetOptional is set_Optional.
func (a *ContentSerializerAttribute) SetOptional(value bool) { a.optional = value }

// AllowNull is get_AllowNull. It is the one field the constructor sets.
func (a *ContentSerializerAttribute) AllowNull() bool { return a.allowNull }

// SetAllowNull is set_AllowNull.
func (a *ContentSerializerAttribute) SetAllowNull(value bool) { a.allowNull = value }

// SharedResource is get_SharedResource.
func (a *ContentSerializerAttribute) SharedResource() bool { return a.sharedResource }

// SetSharedResource is set_SharedResource.
func (a *ContentSerializerAttribute) SetSharedResource(value bool) { a.sharedResource = value }

// CollectionItemName is get_CollectionItemName, which is NOT a field read:
//
//	if (String.IsNullOrEmpty(this.collectionItemName)) return "Item";
//	return this.collectionItemName;
//
// So an unset name answers the literal "Item" rather than the empty string, and
// HasCollectionItemName is what tells the two apart.
func (a *ContentSerializerAttribute) CollectionItemName() string {
	if a.collectionItemName == "" {
		return "Item"
	}
	return a.collectionItemName
}

// SetCollectionItemName is set_CollectionItemName, which REFUSES an empty
// value -- `String.IsNullOrEmpty(value)` throws ArgumentNullException("value").
// The parameter it names is "value", the compiler's name for a setter's
// argument, not the property's.
func (a *ContentSerializerAttribute) SetCollectionItemName(value string) error {
	if value == "" {
		return contentArgumentNullError("value")
	}
	a.collectionItemName = value
	return nil
}

// HasCollectionItemName is get_HasCollectionItemName, which reads the FIELD
// rather than the property:
//
//	return !String.IsNullOrEmpty(this.collectionItemName);
//
// Reading the property instead would answer true always, because the property
// substitutes "Item" for an empty field.
func (a *ContentSerializerAttribute) HasCollectionItemName() bool {
	return a.collectionItemName != ""
}

// Clone is ContentSerializerAttribute::Clone(), which copies all six FIELDS
// into a fresh instance:
//
//	var copy = new ContentSerializerAttribute();
//	copy.elementName = this.elementName; ... copy.collectionItemName = this.collectionItemName;
//
// The collection item name is copied as the FIELD, so a clone of an attribute
// with no name set keeps the empty field and still answers "Item" from the
// property -- and HasCollectionItemName stays false on both. Copying through
// the property would set the clone's field to "Item" and flip that to true.
//
// The fresh instance is constructed rather than memberwise-copied, so allowNull
// is set to true first and then overwritten by the source's value. That is
// invisible here and it is why the projection constructs one too.
func (a *ContentSerializerAttribute) Clone() *ContentSerializerAttribute {
	clone := NewContentSerializerAttribute()
	clone.elementName = a.elementName
	clone.flattenContent = a.flattenContent
	clone.optional = a.optional
	clone.allowNull = a.allowNull
	clone.sharedResource = a.sharedResource
	clone.collectionItemName = a.collectionItemName
	return clone
}

// Equals is the inherited System.Attribute::Equals(Object).
func (a *ContentSerializerAttribute) Equals(obj any) bool { return attributeEquals(a, obj) }

// GetHashCode is the inherited System.Attribute::GetHashCode.
func (a *ContentSerializerAttribute) GetHashCode() int32 { return attributeGetHashCode(a) }

// Match is the inherited System.Attribute::Match(Object).
func (a *ContentSerializerAttribute) Match(obj any) bool { return attributeMatch(a, obj) }

// IsDefaultAttribute is the inherited System.Attribute::IsDefaultAttribute.
func (a *ContentSerializerAttribute) IsDefaultAttribute() bool {
	return attributeIsDefaultAttribute(a)
}

// TypeId is the inherited System.Attribute::get_TypeId.
func (a *ContentSerializerAttribute) TypeId() any { return attributeTypeID(a) }
