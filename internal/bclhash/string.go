// SPDX-License-Identifier: MS-PL

// Package bclhash holds the BCL hash algorithms CNA-Go reproduces from pinned
// .NET Framework 4.0 metadata.
//
// It exists because more than one projected namespace needs the same algorithm
// and Go has no way to share an unexported function across packages. The
// framework package needs it for Dictionary<string,V>'s default comparer;
// Microsoft.Xna.Framework.Audio needs it for RendererDetail.GetHashCode, which
// XORs the hashes of its two strings.
//
// Nothing here is public XNA surface. The package is `internal`, so a consumer
// of the module cannot reach it and the contract gains nothing it does not
// declare.
package bclhash

import "unicode/utf16"

// String is System.String::GetHashCode, read from the pinned mscorlib
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
func String(value string) int32 {
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
