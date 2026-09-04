package audio

import (
	"github.com/openeggbert/cna-go/internal/bclhash"
)

// RendererDetail is Microsoft.Xna.Framework.Audio.RendererDetail:
//
//	.class public sequential ansi sealed beforefieldinit RendererDetail
//	       extends [mscorlib]System.ValueType
//	  .field private string _name
//	  .field private string _id
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// It is the FIRST type in this project whose authority is the Xact assembly
// rather than Microsoft.Xna.Framework.dll, and it is worth saying why it is not
// therefore XACT work: `AudioEngine.RendererDetails` is what produces one, but
// the TYPE is two strings and five members with no native dependency at all.
// Projecting it needs no audio engine, no bank and no device.
//
// # It has no public constructor, and that is the whole shape
//
// The only constructor is `assembly`:
//
//	.ctor(string name, string id)   15 bytes, two stores, nothing else
//
// So a consumer never builds one; they receive it from a collection this
// project does not yet project. The projection therefore exposes the zero value
// and the five public members, which is exactly what the pinned contract
// declares -- and a zero RendererDetail is a legal, reachable CLR value
// (`default(RendererDetail)`), so it is not a degenerate case invented here.
//
// # Pure managed
//
// Every member reads the two fields. Nothing reaches CNA, and no route in the
// canonical headers corresponds to any of them.
type RendererDetail struct {
	// name is `_name`, the field FriendlyName reads.
	name string
	// id is `_id`, the field RendererId reads.
	id string
}

// newRendererDetail is the assembly-internal constructor, unexported for the
// reason the reference makes it `assembly`: it is how the framework builds one,
// not how a consumer does. It exists so the type can be constructed at all
// inside this module once a producing collection arrives.
func newRendererDetail(name, id string) RendererDetail {
	return RendererDetail{name: name, id: id}
}

// FriendlyName is RendererDetail::get_FriendlyName, one `ldfld`.
func (d RendererDetail) FriendlyName() string { return d.name }

// RendererId is RendererDetail::get_RendererId, one `ldfld`.
func (d RendererDetail) RendererId() string { return d.id }

// GetHashCode is RendererDetail::GetHashCode, 60 bytes:
//
//	int idHash   = String.IsNullOrEmpty(_id)   ? 0 : _id.GetHashCode();
//	int nameHash = String.IsNullOrEmpty(_name) ? 0 : _name.GetHashCode();
//	return nameHash ^ idHash;
//
// Two details are load-bearing. The EMPTY string hashes to zero here rather
// than to String.GetHashCode("") -- the guard is `IsNullOrEmpty`, not a null
// check -- so a detail with an empty name and one with a null name have the
// same hash, which they would not if the guard were narrower. And Go has no
// null string, so `IsNullOrEmpty` collapses to a length test, which is what the
// CLR's own guard does for the empty case anyway.
//
// The string hash is the pinned mscorlib algorithm, not Go's: see bclhash.String.
func (d RendererDetail) GetHashCode() int32 {
	nameHash := int32(0)
	if d.name != "" {
		nameHash = bclhash.String(d.name)
	}
	idHash := int32(0)
	if d.id != "" {
		idHash = bclhash.String(d.id)
	}
	return nameHash ^ idHash
}

// ToString is RendererDetail::ToString, 17 bytes:
//
//	ldarg.0; ldobj RendererDetail; box RendererDetail
//	call instance string [mscorlib]System.ValueType::ToString()
//
// It boxes and calls ValueType::ToString, which answers `GetType().ToString()`
// -- the full CLR type name and NOTHING about the two fields. The override
// exists only because the struct is `sequential` and the compiler emitted one;
// it adds no information a caller could use.
//
// So the answer is the type name, for every value including the zero one. A
// reader who expected the friendly name here would be reading a member that
// does not report it, and this is the one place to say so.
func (d RendererDetail) ToString() string {
	return "Microsoft.Xna.Framework.Audio.RendererDetail"
}

// RendererDetailOperatorEqualityByRendererDetailAndRendererDetail is
// RendererDetail::op_Equality, 43 bytes:
//
//	return left._name == right._name && left._id == right._id;
//
// It SHORT-CIRCUITS: the id comparison runs only when the names match, which is
// what the `brfalse` after the first String::op_Equality does. Go's && has the
// same shape, so the projection is the reference's expression rather than a
// two-field struct comparison -- and the distinction matters because the
// reference compares STRINGS by value, exactly as Go's == does.
func RendererDetailOperatorEqualityByRendererDetailAndRendererDetail(left, right RendererDetail) bool {
	return left.name == right.name && left.id == right.id
}

// RendererDetailOperatorInequalityByRendererDetailAndRendererDetail is
// RendererDetail::op_Inequality, 11 bytes: `!op_Equality(left, right)`.
func RendererDetailOperatorInequalityByRendererDetailAndRendererDetail(left, right RendererDetail) bool {
	return !RendererDetailOperatorEqualityByRendererDetailAndRendererDetail(left, right)
}

// Equals is RendererDetail::Equals(object), 54 bytes, and its middle guard is
// the one a reader would drop:
//
//	if (obj == null) return false;
//	if (obj.GetType() != this.GetType()) return false;
//	return this == (RendererDetail)obj;
//
// The TYPE test is exact -- `Type::op_Inequality` on the two runtime types, not
// an `isinst` -- so a boxed value of any other type answers false rather than
// throwing at the unbox. Go's type assertion with the two-value form has
// exactly that behaviour: a non-RendererDetail answers false, and a nil `any`
// answers false through the same path the CLR's null check takes.
func (d RendererDetail) Equals(obj any) bool {
	other, ok := obj.(RendererDetail)
	return ok && RendererDetailOperatorEqualityByRendererDetailAndRendererDetail(d, other)
}
