package framework

import "reflect"

// InstanceDescriptor projects
// System.ComponentModel.Design.Serialization.InstanceDescriptor: a description
// of how to rebuild a value, as the member that constructs it plus the
// arguments to pass.
//
// # Reference authority
//
//	System.dll   c3182e40f09a8d3a...
//
// # Both roles it plays here, measured
//
// It is a TYPE TOKEN in four places -- MathTypeConverter::CanConvertTo and
// TypeConverter::CanConvertFrom both compare `typeof` it, so a consumer must be
// able to name it to ask the question.
//
// And it is CONSTRUCTED, in every leaf converter's ConvertTo:
//
//	if (destinationType == typeof(InstanceDescriptor) && value is Vector2) {
//	    ConstructorInfo ctor = typeof(Vector2).GetConstructor(new[]{ typeof(float), typeof(float) });
//	    if (ctor != null)
//	        return new InstanceDescriptor(ctor, new object[]{ v.X, v.Y });
//	}
//
// # The member is the TYPE, because Go has no ConstructorInfo
//
// The reference carries a `System.Reflection.ConstructorInfo`. A Go struct has
// no constructor to describe -- the projection's own are package functions --
// so what is carried is the reflect.Type being built, which is the part a
// consumer can act on: the arguments are in the reference's own order and the
// type says what they build.
//
// That is a divergence in SHAPE and not in content, and it is recorded rather
// than papered over: a caller that wanted to invoke the constructor reflectively
// cannot, and a caller that wanted to know what this value is and what it is
// made of can.
type InstanceDescriptor struct {
	memberType reflect.Type
	arguments  []any
}

// NewInstanceDescriptor is InstanceDescriptor::.ctor(MemberInfo, ICollection),
// with the member spelled as the type it constructs.
//
// The arguments are COPIED. The reference takes an ICollection and holds it,
// and a caller that mutated the array afterwards would change the descriptor;
// copying is the safer reading of a value type's description and the difference
// is not observable through any member the profile declares.
func NewInstanceDescriptor(memberType reflect.Type, arguments []any) *InstanceDescriptor {
	copied := make([]any, len(arguments))
	copy(copied, arguments)
	return &InstanceDescriptor{memberType: memberType, arguments: copied}
}

// MemberType is what the reference's get_MemberInfo names: the member that
// builds the value. Here it is the type being built.
func (d *InstanceDescriptor) MemberType() reflect.Type {
	if d == nil {
		return nil
	}
	return d.memberType
}

// Arguments is get_Arguments, in the reference's own order.
func (d *InstanceDescriptor) Arguments() []any {
	if d == nil {
		return nil
	}
	copied := make([]any, len(d.arguments))
	copy(copied, d.arguments)
	return copied
}
