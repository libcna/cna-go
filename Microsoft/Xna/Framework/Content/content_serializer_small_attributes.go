package content

// The four smaller content serializer attributes. Each is a sealed class over
// System.Attribute with one constructor, at most one get-only property, and no
// behaviour beyond a constructor guard -- so they share a file rather than four
// nearly empty ones.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// Every one of them can be constructed and read and none can be ATTACHED to a
// declaration; see attributeBase for why, stated once for all five.

// ContentSerializerCollectionItemNameAttribute is
// Microsoft.Xna.Framework.Content.ContentSerializerCollectionItemNameAttribute.
//
// It is the standalone form of ContentSerializerAttribute's own
// CollectionItemName, and the two differ in a way worth keeping: this one has
// NO "Item" fallback. Its getter is one ldfld, so an instance always answers
// exactly what it was built with -- which the constructor guarantees is not
// empty.
type ContentSerializerCollectionItemNameAttribute struct {
	base               attributeBase
	collectionItemName string
}

// NewContentSerializerCollectionItemNameAttribute is the .ctor(String), which
// calls the base constructor FIRST and only then checks its argument:
//
//	base..ctor();
//	if (String.IsNullOrEmpty(collectionItemName))
//	    throw new ArgumentNullException("collectionItemName");
//
// The parameter it names is the constructor's own, not "value".
func NewContentSerializerCollectionItemNameAttribute(collectionItemName string) (*ContentSerializerCollectionItemNameAttribute, error) {
	if collectionItemName == "" {
		return nil, contentArgumentNullError("collectionItemName")
	}
	return &ContentSerializerCollectionItemNameAttribute{collectionItemName: collectionItemName}, nil
}

func (a *ContentSerializerCollectionItemNameAttribute) attributeSelf() any { return a }

// CollectionItemName is get_CollectionItemName, one ldfld and no fallback.
func (a *ContentSerializerCollectionItemNameAttribute) CollectionItemName() string {
	return a.collectionItemName
}

// Equals is the inherited System.Attribute::Equals(Object).
func (a *ContentSerializerCollectionItemNameAttribute) Equals(obj any) bool {
	return attributeEquals(a, obj)
}

// GetHashCode is the inherited System.Attribute::GetHashCode.
func (a *ContentSerializerCollectionItemNameAttribute) GetHashCode() int32 {
	return attributeGetHashCode(a)
}

// Match is the inherited System.Attribute::Match(Object).
func (a *ContentSerializerCollectionItemNameAttribute) Match(obj any) bool {
	return attributeMatch(a, obj)
}

// IsDefaultAttribute is the inherited System.Attribute::IsDefaultAttribute.
func (a *ContentSerializerCollectionItemNameAttribute) IsDefaultAttribute() bool {
	return attributeIsDefaultAttribute(a)
}

// TypeId is the inherited System.Attribute::get_TypeId.
func (a *ContentSerializerCollectionItemNameAttribute) TypeId() any { return attributeTypeID(a) }

// ContentSerializerIgnoreAttribute is
// Microsoft.Xna.Framework.Content.ContentSerializerIgnoreAttribute, whose whole
// body is the base constructor call. It carries no state at all: its PRESENCE
// on a member is the entire message, which is also the one thing Go cannot
// express.
type ContentSerializerIgnoreAttribute struct {
	base attributeBase
}

// NewContentSerializerIgnoreAttribute is the .ctor(), two instructions.
func NewContentSerializerIgnoreAttribute() *ContentSerializerIgnoreAttribute {
	return &ContentSerializerIgnoreAttribute{}
}

func (a *ContentSerializerIgnoreAttribute) attributeSelf() any { return a }

// Equals is the inherited System.Attribute::Equals(Object). With no fields,
// every instance of this type equals every other -- which is the reference's
// answer too, because its field walk finds nothing to disagree about.
func (a *ContentSerializerIgnoreAttribute) Equals(obj any) bool { return attributeEquals(a, obj) }

// GetHashCode is the inherited System.Attribute::GetHashCode.
func (a *ContentSerializerIgnoreAttribute) GetHashCode() int32 { return attributeGetHashCode(a) }

// Match is the inherited System.Attribute::Match(Object).
func (a *ContentSerializerIgnoreAttribute) Match(obj any) bool { return attributeMatch(a, obj) }

// IsDefaultAttribute is the inherited System.Attribute::IsDefaultAttribute.
func (a *ContentSerializerIgnoreAttribute) IsDefaultAttribute() bool {
	return attributeIsDefaultAttribute(a)
}

// TypeId is the inherited System.Attribute::get_TypeId.
func (a *ContentSerializerIgnoreAttribute) TypeId() any { return attributeTypeID(a) }

// ContentSerializerRuntimeTypeAttribute is
// Microsoft.Xna.Framework.Content.ContentSerializerRuntimeTypeAttribute, which
// names the runtime type an intermediate type deserializes into. The name is a
// STRING and not a Type: the runtime type may not be loaded when the attribute
// is read, which is the whole reason it is spelled rather than referenced.
type ContentSerializerRuntimeTypeAttribute struct {
	base        attributeBase
	runtimeType string
}

// NewContentSerializerRuntimeTypeAttribute is the .ctor(String), with the same
// base-first, guard-second shape as the collection item name attribute.
func NewContentSerializerRuntimeTypeAttribute(runtimeType string) (*ContentSerializerRuntimeTypeAttribute, error) {
	if runtimeType == "" {
		return nil, contentArgumentNullError("runtimeType")
	}
	return &ContentSerializerRuntimeTypeAttribute{runtimeType: runtimeType}, nil
}

func (a *ContentSerializerRuntimeTypeAttribute) attributeSelf() any { return a }

// RuntimeType is get_RuntimeType, one ldfld.
func (a *ContentSerializerRuntimeTypeAttribute) RuntimeType() string { return a.runtimeType }

// Equals is the inherited System.Attribute::Equals(Object).
func (a *ContentSerializerRuntimeTypeAttribute) Equals(obj any) bool { return attributeEquals(a, obj) }

// GetHashCode is the inherited System.Attribute::GetHashCode.
func (a *ContentSerializerRuntimeTypeAttribute) GetHashCode() int32 {
	return attributeGetHashCode(a)
}

// Match is the inherited System.Attribute::Match(Object).
func (a *ContentSerializerRuntimeTypeAttribute) Match(obj any) bool { return attributeMatch(a, obj) }

// IsDefaultAttribute is the inherited System.Attribute::IsDefaultAttribute.
func (a *ContentSerializerRuntimeTypeAttribute) IsDefaultAttribute() bool {
	return attributeIsDefaultAttribute(a)
}

// TypeId is the inherited System.Attribute::get_TypeId.
func (a *ContentSerializerRuntimeTypeAttribute) TypeId() any { return attributeTypeID(a) }

// ContentSerializerTypeVersionAttribute is
// Microsoft.Xna.Framework.Content.ContentSerializerTypeVersionAttribute.
//
// Its constructor takes an Int32 and does NOT guard it -- unlike the two string
// constructors, which refuse an empty value. A negative version is accepted,
// and that is the reference's own behaviour rather than an omission here.
type ContentSerializerTypeVersionAttribute struct {
	base        attributeBase
	typeVersion int32
}

// NewContentSerializerTypeVersionAttribute is the .ctor(Int32), four
// instructions and no refusal.
func NewContentSerializerTypeVersionAttribute(typeVersion int32) *ContentSerializerTypeVersionAttribute {
	return &ContentSerializerTypeVersionAttribute{typeVersion: typeVersion}
}

func (a *ContentSerializerTypeVersionAttribute) attributeSelf() any { return a }

// TypeVersion is get_TypeVersion, one ldfld.
func (a *ContentSerializerTypeVersionAttribute) TypeVersion() int32 { return a.typeVersion }

// Equals is the inherited System.Attribute::Equals(Object).
func (a *ContentSerializerTypeVersionAttribute) Equals(obj any) bool { return attributeEquals(a, obj) }

// GetHashCode is the inherited System.Attribute::GetHashCode.
func (a *ContentSerializerTypeVersionAttribute) GetHashCode() int32 {
	return attributeGetHashCode(a)
}

// Match is the inherited System.Attribute::Match(Object).
func (a *ContentSerializerTypeVersionAttribute) Match(obj any) bool { return attributeMatch(a, obj) }

// IsDefaultAttribute is the inherited System.Attribute::IsDefaultAttribute.
func (a *ContentSerializerTypeVersionAttribute) IsDefaultAttribute() bool {
	return attributeIsDefaultAttribute(a)
}

// TypeId is the inherited System.Attribute::get_TypeId.
func (a *ContentSerializerTypeVersionAttribute) TypeId() any { return attributeTypeID(a) }
