package framework

import (
	"github.com/openeggbert/cna-go/internal/bclhash"
	"reflect"
)

// This file is CNA-Go language support, not XNA surface.
//
// IEqualityComparer is the projection of
// System.Collections.Generic.IEqualityComparer<T>, which the pinned XNA public
// contract carries at a signature position: LaunchParameters inherits
// Dictionary<string,string>::get_Comparer, whose return type it is. It is a
// declared BCL signature adapter in tools/api_compat/mapping-rules.json and
// adds no XNA identity of its own, on the same footing as System.TimeSpan and
// System.EventHandler<T>.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// The interface declares exactly two members:
//
//	.method public hidebysig newslot abstract virtual instance bool
//	        Equals(!T x, !T y)
//	.method public hidebysig newslot abstract virtual instance int32
//	        GetHashCode(!T obj)
//
// Both are projected. Neither is fallible: an implementor's contract is to
// answer, and the two implementations in the profile are field reads and
// arithmetic.
type IEqualityComparer[T any] interface {
	// Equals is IEqualityComparer<T>::Equals.
	Equals(x, y T) bool
	// GetHashCode is IEqualityComparer<T>::GetHashCode.
	GetHashCode(obj T) int32
}

// stringEqualityComparer projects EqualityComparer<System.String>.Default.
//
// The selection is the CLR's, not a choice: EqualityComparer<T>.CreateComparer
// tests `typeof(IEquatable<T>).IsAssignableFrom(typeof(T))` first, and
// System.String implements IEquatable<string>, so the default comparer for
// string is GenericEqualityComparer<string>, whose two members are
//
//	Equals(x, y)     x != null ? x.Equals(y) : y == null
//	GetHashCode(obj) obj == null ? 0 : obj.GetHashCode()
//
// String::Equals(string) is ordinal, character-by-character, which is exactly
// Go's ==. String::GetHashCode is reproduced by stringHashCode below.
//
// The type is unexported and the value is a singleton, because
// EqualityComparer<T>.Default is a cached static: two dictionaries built by the
// parameterless constructor hand back the SAME comparer object, and a Go
// consumer comparing two Comparer() results must see that.
type stringEqualityComparer struct{}

// defaultStringComparer is the one instance, so Comparer() has CLR's cached
// static identity rather than allocating per call.
var defaultStringComparer IEqualityComparer[string] = stringEqualityComparer{}

// Equals is GenericEqualityComparer<string>::Equals. The null branches are
// unreachable here: CNA-Go maps System.String to Go string, which has no null,
// so the whole body reduces to String::Equals(string), which is ordinal.
func (stringEqualityComparer) Equals(x, y string) bool { return x == y }

// GetHashCode is GenericEqualityComparer<string>::GetHashCode, which is one
// forwarded String::GetHashCode.
func (stringEqualityComparer) GetHashCode(obj string) int32 { return stringHashCode(obj) }

// stringHashCode is System.String::GetHashCode. The algorithm and the three
// load-bearing details it reproduces are documented on bclhash.String, which is
// where it lives so the audio package can reach the same body -- Go has no way
// to share an unexported function across packages, and RendererDetail.GetHashCode
// needs exactly this hash.
func stringHashCode(value string) int32 { return bclhash.String(value) }

// referenceIdentityHashCode projects System.Object::GetHashCode() for a CLR
// reference.
//
// The CLR derives that value from the object's sync-block index, which is
// unspecified, differs between runs of the same program, and is documented as
// such. It is therefore not reproducible by construction -- not by CNA-Go, and
// not by the reference against itself. What IS part of the contract is the two
// properties every identity hash has: it is stable for the object's lifetime,
// and two distinct live objects usually differ. This supplies both from the Go
// pointer the reference facade already is.
//
// A nil reference answers zero. The CLR would throw NullReferenceException
// there, because `null.GetHashCode()` is a virtual call on nothing; the one
// caller in the profile -- GraphicsDeviceInformation::GetHashCode -- is
// infallible in the contract, so zero is the answer rather than a panic from
// inside a hash.
func referenceIdentityHashCode(value any) int32 {
	if value == nil {
		return 0
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice:
		if reflected.IsNil() {
			return 0
		}
		address := uint64(reflected.Pointer())
		return int32(uint32(address>>4) ^ uint32(address>>32))
	default:
		// A value type reached this position, which no CLR reference can be.
		// Zero is the honest answer: there is no object identity to hash.
		return 0
	}
}
