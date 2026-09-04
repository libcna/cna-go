package content

import "reflect"

// attributeBase is the private System.Attribute adapter the five content
// serializer attributes compose.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # It holds nothing, and that is measured
//
// System.Attribute declares NO instance field. Its only state is the object
// header every CLR object has, so the adapter is an empty struct and each
// inherited member answers from the DERIVED object's own data -- which is why
// every one of them takes the owner rather than reading the adapter.
//
// # What Go cannot do, stated once
//
// A CLR attribute is applied to a declaration and read back with reflection.
// Go has no attribute metadata, so a projected attribute can be CONSTRUCTED and
// READ but never attached to anything. That is a real limitation and it is
// recorded on each of the five types rather than hidden.
//
// It is also narrower than it sounds. The runtime's own readers of these
// attributes are ReflectiveReader`1 and ReflectiveReaderMemberHelper, both
// `private` and neither public surface, and CNA performs the type-reader
// dispatch itself. So nothing a consumer can reach through this projection
// would have read an applied attribute even if Go could apply one.
type attributeBase struct{}

// attributeObject is what an inherited member needs from the attribute that
// composes it: the derived value, so field-by-field comparison and the runtime
// type both see the real object.
//
// A NIL owner is not a case any of these functions guards, and that is
// deliberate: every caller is a method on one of the five attributes, which
// passes its own receiver. A nil receiver arrives here as a non-nil interface
// holding a nil pointer, which reflect handles -- it is only a literal nil
// interface that would panic, and no path in this package produces one.
type attributeObject interface {
	// attributeSelf answers the derived attribute as an interface value. It is
	// unexported because the CLR declares no accessor for the base object.
	attributeSelf() any
}

// attributeEquals is System.Attribute::Equals(Object).
//
// The reference does NOT compare references. It checks the runtime types match
// and then compares every field through reflection, so two separately
// constructed attributes with equal fields are equal -- which Go's == on two
// pointers would report as false.
func attributeEquals(owner attributeObject, obj any) bool {
	// ONE call, and that is not a shortcut. The reference performs three steps
	// -- a null check, a runtime-type comparison, and a field-by-field walk --
	// and reflect.DeepEqual performs all three: it answers false for a nil
	// against a non-nil, false for two different concrete types, and otherwise
	// compares field by field.
	//
	// Writing the first two out separately was the first attempt here, and the
	// mutation run is what showed they were dead: neutering either changed no
	// answer, because DeepEqual had already decided. A guard no input can
	// reach is not documentation, it is a line that looks tested and is not.
	return reflect.DeepEqual(owner.attributeSelf(), obj)
}

// attributeGetHashCode is System.Attribute::GetHashCode.
//
// The reference builds the hash from the runtime TYPE, not from the field
// values, so every instance of one attribute type shares a hash and Equals is
// what separates them. Reproducing that keeps the pair consistent: equal
// objects hash equal, which is the contract a hash has to keep.
func attributeGetHashCode(owner attributeObject) int32 {
	return clrTypeNameHash(reflect.TypeOf(owner.attributeSelf()).String())
}

// clrTypeNameHash is a stable string hash. It is the project's own, because the
// CLR's type hash is a runtime handle value no projection can reproduce and
// pretending otherwise would be inventing a number.
func clrTypeNameHash(name string) int32 {
	var hash int32 = 17
	for index := 0; index < len(name); index++ {
		hash = hash*31 + int32(name[index])
	}
	return hash
}

// attributeMatch is System.Attribute::Match(Object), whose base body is
// `return this.Equals(obj)`. A derived attribute may narrow it and none of
// these five does.
func attributeMatch(owner attributeObject, obj any) bool {
	return attributeEquals(owner, obj)
}

// attributeIsDefaultAttribute is System.Attribute::IsDefaultAttribute, whose
// base body is `ldc.i4.0; ret`. None of the five overrides it, so all five
// answer false -- including a default-constructed ContentSerializerAttribute,
// which is the case a plausible implementation would get wrong.
func attributeIsDefaultAttribute(owner attributeObject) bool {
	return false
}

// attributeTypeID is System.Attribute::get_TypeId, whose default returns
// GetType(). It answers the RUNTIME type through an object-typed property,
// which is why the projection's result is `any` holding a reflect.Type rather
// than a reflect.Type directly: narrowing it would be a different member.
func attributeTypeID(owner attributeObject) any {
	return reflect.TypeOf(owner.attributeSelf())
}
