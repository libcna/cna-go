package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

// This file reads the .NET resource sets embedded in a managed assembly, so a
// claimed message can be checked against THE KEY THE REFERENCE THROWS UNDER
// rather than against the assembly's bytes as a whole.
//
// # Why the substring search was not enough
//
// Foundation 49 checked that a claimed message appeared somewhere in its
// assembly. That catches an invented sentence, which is what it was built for,
// and it cannot catch a sentence that is real but filed under the wrong key --
// the second half of the same defect, because the key is what names the throw
// site.
//
// It was hiding one. CNA-Go recorded the audio-emitter message under
// `DopplerScaleMustBeGreaterThanOrEqualToZero`, a key that does not exist. The
// string is real and correct; its key is `InvalidEmitterDopplerScale`. The
// substring search passed it for four milestones.
//
// # The format
//
// A resource set begins with the magic 0xBEEFCACE and is laid out as
// System.Resources.ResourceReader writes it:
//
//	magic            4  0xBEEFCACE
//	readerVersion    4
//	skipLength       4  bytes of reader/set type names to skip
//	<type names>
//	headerVersion    4  2 for every set in these assemblies
//	resourceCount    4
//	typeCount        4
//	<type names>        each 7-bit-length-prefixed
//	<pad to 8>          RELATIVE TO THE START OF THE SET, not to the file
//	nameHashes       4 * resourceCount
//	namePositions    4 * resourceCount
//	dataSectionStart 4  relative to the start of the set
//	<name section>      each entry: 7-bit byte length, UTF-16LE name, 4-byte data offset
//	<data section>      each entry: 7-bit type index, then the value
//
// Only type index 1, String, is read: every message this tool checks is one,
// and a reader that decoded more would be a larger thing that can be wrong.
//
// The eight-byte alignment is measured from the start of the set. Aligning to
// the file offset instead produces a reader that finds the magic, accepts the
// header, and then reads names out of the middle of the hash table -- it fails
// loudly here rather than silently, but only because the name lengths stop
// decoding.

const resourceSetMagic = 0xBEEFCACE

// resourceStrings reads every string resource in every resource set the blob
// contains, keyed by name.
func resourceStrings(blob []byte) (map[string]string, error) {
	strings := map[string]string{}
	sets := 0
	for offset := 0; offset+4 <= len(blob); offset++ {
		if binary.LittleEndian.Uint32(blob[offset:]) != resourceSetMagic {
			continue
		}
		set, err := readResourceSet(blob, offset)
		if err != nil {
			// A four-byte sequence can be the magic by accident. A set that
			// does not parse is not one, and the caller's own key lookup is
			// what reports a genuinely missing resource.
			continue
		}
		sets++
		for key, value := range set {
			strings[key] = value
		}
	}
	if sets == 0 {
		return nil, errors.New("no resource set found")
	}
	return strings, nil
}

func readResourceSet(blob []byte, base int) (map[string]string, error) {
	cursor := base + 4
	read32 := func() (int, error) {
		if cursor+4 > len(blob) {
			return 0, errors.New("truncated")
		}
		value := int(int32(binary.LittleEndian.Uint32(blob[cursor:])))
		cursor += 4
		return value, nil
	}

	if _, err := read32(); err != nil { // reader version
		return nil, err
	}
	skip, err := read32()
	if err != nil {
		return nil, err
	}
	if skip < 0 || cursor+skip > len(blob) {
		return nil, errors.New("implausible type-name length")
	}
	cursor += skip

	headerVersion, err := read32()
	if err != nil {
		return nil, err
	}
	if headerVersion != 2 {
		return nil, fmt.Errorf("unsupported resource header version %d", headerVersion)
	}
	resourceCount, err := read32()
	if err != nil {
		return nil, err
	}
	typeCount, err := read32()
	if err != nil {
		return nil, err
	}
	if resourceCount < 0 || resourceCount > 1<<20 || typeCount < 0 || typeCount > 1<<16 {
		return nil, errors.New("implausible counts")
	}
	for index := 0; index < typeCount; index++ {
		length, next, err := read7BitLength(blob, cursor)
		if err != nil {
			return nil, err
		}
		cursor = next + length
		if cursor > len(blob) {
			return nil, errors.New("truncated type name")
		}
	}

	// Relative to the START OF THE SET.
	cursor = base + ((cursor-base)+7)&^7

	cursor += 4 * resourceCount // name hashes
	if cursor+4*resourceCount+4 > len(blob) {
		return nil, errors.New("truncated name positions")
	}
	positions := make([]int, resourceCount)
	for index := range positions {
		positions[index] = int(int32(binary.LittleEndian.Uint32(blob[cursor:])))
		cursor += 4
	}
	dataSection, err := read32()
	if err != nil {
		return nil, err
	}
	nameSection := cursor

	set := make(map[string]string, resourceCount)
	for _, position := range positions {
		at := nameSection + position
		if position < 0 || at >= len(blob) {
			return nil, errors.New("name position out of range")
		}
		length, next, err := read7BitLength(blob, at)
		if err != nil {
			return nil, err
		}
		if length < 0 || length%2 != 0 || next+length+4 > len(blob) {
			return nil, errors.New("implausible name length")
		}
		name := decodeUTF16(blob[next : next+length])
		valueOffset := int(int32(binary.LittleEndian.Uint32(blob[next+length:])))

		at = base + dataSection + valueOffset
		if at < 0 || at >= len(blob) {
			return nil, errors.New("data position out of range")
		}
		kind, next, err := read7BitLength(blob, at)
		if err != nil {
			return nil, err
		}
		if kind != 1 { // String
			continue
		}
		length, next, err = read7BitLength(blob, next)
		if err != nil {
			return nil, err
		}
		if length < 0 || next+length > len(blob) {
			return nil, errors.New("implausible value length")
		}
		set[name] = string(blob[next : next+length])
	}
	return set, nil
}

// read7BitLength decodes the 7-bit-encoded length BinaryWriter writes.
func read7BitLength(blob []byte, at int) (int, int, error) {
	value := 0
	shift := 0
	for {
		if at >= len(blob) {
			return 0, 0, errors.New("truncated 7-bit length")
		}
		if shift > 28 {
			return 0, 0, errors.New("overlong 7-bit length")
		}
		current := blob[at]
		at++
		value |= int(current&0x7f) << shift
		if current&0x80 == 0 {
			return value, at, nil
		}
		shift += 7
	}
}

func decodeUTF16(raw []byte) string {
	units := make([]uint16, len(raw)/2)
	for index := range units {
		units[index] = binary.LittleEndian.Uint16(raw[2*index:])
	}
	return string(utf16.Decode(units))
}
