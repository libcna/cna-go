package framework

// PropertyDescriptor projects System.ComponentModel.PropertyDescriptor as far
// as the profile reaches it: a NAMED, READABLE component of a value.
//
// # Reference authority
//
//	System.dll                    c3182e40f09a8d3a...
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Why it is not a reflect.StructField
//
// Reading only the vector converters says it could be. Vector2Converter builds
// its descriptors with
//
//	new FieldPropertyDescriptor(typeof(Vector2).GetField("X"))
//
// and a FieldInfo is a reflect.StructField. But ColorConverter builds its with
//
//	new PropertyPropertyDescriptor(typeof(Color).GetProperty("R"))
//
// because XNA's Color exposes R, G, B and A as PROPERTIES over a packed value
// rather than as fields -- and this projection follows it, so Color's
// components are methods and have no StructField at all.
//
// Both `FieldPropertyDescriptor` and `PropertyPropertyDescriptor` are `private`
// in the reference and neither is in the contract. What IS in the contract is
// the collection they end up in, so what the projection has to carry is the
// thing both kinds have in common: a name, and a way to read that component off
// a value. That is what this is.
type PropertyDescriptor struct {
	name string
	read func(component any) (any, bool)
}

// NewPropertyDescriptor builds one. It is exported because a consumer holding a
// collection from GetProperties may want to build a comparable one; the
// reference's two descriptor classes are private, so this is the projection's
// single spelling of both.
func NewPropertyDescriptor(name string, read func(component any) (any, bool)) PropertyDescriptor {
	return PropertyDescriptor{name: name, read: read}
}

// Name is PropertyDescriptor::get_Name, which for both reference subclasses is
// the reflected member's name.
func (d PropertyDescriptor) Name() string { return d.name }

// GetValue is PropertyDescriptor::GetValue(Object), reading the component off
// the value it describes.
//
// The second result reports whether the component was READ, which the reference
// signals by throwing for a value of the wrong type. A projection that returned
// a zero would be indistinguishable from a component that really is zero.
func (d PropertyDescriptor) GetValue(component any) (any, bool) {
	if d.read == nil {
		return nil, false
	}
	return d.read(component)
}
