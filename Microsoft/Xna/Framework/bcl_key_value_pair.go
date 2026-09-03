package framework

import (
	"fmt"
	"strings"
)

// This file is CNA-Go language support, not XNA surface.
//
// KeyValuePair is the projection of
// System.Collections.Generic.KeyValuePair<TKey,TValue>, which the pinned XNA
// public contract carries at a signature position: LaunchParameters inherits
// Dictionary<string,string>::GetEnumerator, whose enumerator's element type it
// is. It is a declared BCL signature adapter in
// tools/api_compat/mapping-rules.json and adds no XNA identity of its own.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// The CLR type is a struct with two private fields and exactly three public
// members besides its constructor -- get_Key, get_Value and ToString -- so it
// projects as a Go struct value with unexported fields and three methods. The
// fields are unexported because the CLR fields are private and the properties
// are get-only: a public Go field would publish a setter the reference does
// not declare.
type KeyValuePair[TKey any, TValue any] struct {
	key   TKey
	value TValue
}

// NewKeyValuePair is KeyValuePair<TKey,TValue>::.ctor, which is two stfld and
// validates nothing.
func NewKeyValuePair[TKey any, TValue any](key TKey, value TValue) KeyValuePair[TKey, TValue] {
	return KeyValuePair[TKey, TValue]{key: key, value: value}
}

// Key is KeyValuePair<TKey,TValue>::get_Key, one ldfld.
func (pair KeyValuePair[TKey, TValue]) Key() TKey { return pair.key }

// Value is KeyValuePair<TKey,TValue>::get_Value, one ldfld.
func (pair KeyValuePair[TKey, TValue]) Value() TValue { return pair.value }

// ToString is KeyValuePair<TKey,TValue>::ToString:
//
//	StringBuilder b = new StringBuilder();
//	b.Append('[');
//	if (Key   != null) b.Append(Key.ToString());
//	b.Append(", ");
//	if (Value != null) b.Append(Value.ToString());
//	b.Append(']');
//	return b.ToString();
//
// so a null half contributes nothing at all rather than a placeholder, and the
// separator is emitted unconditionally: `[, ]` is the reference's rendering of
// a pair of nulls.
//
// The two `constrained. !T; callvirt Object::ToString()` calls are the runtime
// type's virtual ToString, which clrToString projects.
func (pair KeyValuePair[TKey, TValue]) ToString() string {
	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(clrToString(any(pair.key)))
	builder.WriteString(", ")
	builder.WriteString(clrToString(any(pair.value)))
	builder.WriteByte(']')
	return builder.String()
}

// clrToString projects `constrained. !T; callvirt System.Object::ToString()`
// preceded by the `box !T; brfalse` null test that guards it.
//
// The three branches are the whole of what the profile can reach:
//
//   - a nil interface is the boxed null the reference skips, so it contributes
//     the empty string;
//   - a Go string is System.String, whose ToString is `ldarg.0; ret`;
//   - a projected CNA-Go type carries its own ToString, which IS the reference
//     virtual this call would dispatch to.
//
// The profile instantiates KeyValuePair exactly once, over
// <System.String,System.String>, so the first two branches are the reachable
// ones and both are exact. The final fallback is Go's default formatting,
// which is NOT a claim of CLR fidelity for an arbitrary element type: a future
// instantiation over a primitive whose CLR ToString differs from Go's -- a
// float32, whose CLR "R" rendering and Go's %v disagree -- must supply its own
// ToString rather than rely on it.
func clrToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case interface{ ToString() string }:
		return typed.ToString()
	default:
		return fmt.Sprint(typed)
	}
}
