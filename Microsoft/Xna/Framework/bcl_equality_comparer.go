package framework

import (
	"reflect"
	"unicode/utf16"
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

// stringHashCode is System.String::GetHashCode, read from the pinned mscorlib
// IL:
//
//	hash1 = hash2 = 0x15051505
//	pint  = (int*)chars                 // two UTF-16 units per int32
//	len   = this.Length                 // in CHARS
//	while (len > 0) {
//	    hash1 = ((hash1 << 5) + hash1 + (hash1 >> 27)) ^ pint[0];
//	    if (len <= 2) break;
//	    hash2 = ((hash2 << 5) + hash2 + (hash2 >> 27)) ^ pint[1];
//	    pint += 2; len -= 4;
//	}
//	return hash1 + (hash2 * 0x5d588b65);
//
// Three details are load-bearing and are reproduced rather than approximated.
//
//  1. The unit is a UTF-16 code unit, not a byte and not a rune, so a Go
//     string is converted before hashing. A Go string that is not valid UTF-8
//     has no UTF-16 counterpart at all; utf16.Encode over its runes yields
//     U+FFFD for each invalid byte, which is what any UTF-16 view of it must
//     do, and no CLR string can be in that state to disagree with.
//  2. `pint[0]` is a little-endian 32-bit load of two adjacent code units, so
//     the low half is chars[i] and the high half is chars[i+1]. The pinned
//     profile is Windows x86/x64, which is little-endian; this reproduces that
//     load rather than the host's.
//  3. A CLR string is null-terminated in memory, so the loop may read one unit
//     past the last character and always reads a zero there. The odd-length
//     tail below supplies that zero explicitly.
//
// `>> 27` is an arithmetic shift of a signed int32 and every other operation is
// unchecked int32 arithmetic, so the whole body runs on int32 and wraps.
//
// This binary is the .NET Framework 4.0 RTM mscorlib, whose GetHashCode has no
// randomized-hashing branch at all: the algorithm above is the entire method.
// Randomized string hashing arrived after it, and reading the pinned binary is
// what settles that rather than assuming either way.
func stringHashCode(value string) int32 {
	units := utf16.Encode([]rune(value))
	hash1 := int32(0x15051505)
	hash2 := hash1
	length := len(units)
	// unit reads units[index], or the terminating zero one past the end.
	unit := func(index int) int32 {
		if index >= len(units) {
			return 0
		}
		return int32(units[index])
	}
	position := 0
	for length > 0 {
		hash1 = ((hash1 << 5) + hash1 + (hash1 >> 27)) ^ (unit(position) | unit(position+1)<<16)
		if length <= 2 {
			break
		}
		hash2 = ((hash2 << 5) + hash2 + (hash2 >> 27)) ^ (unit(position+2) | unit(position+3)<<16)
		position += 4
		length -= 4
	}
	return hash1 + hash2*0x5d588b65
}

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
