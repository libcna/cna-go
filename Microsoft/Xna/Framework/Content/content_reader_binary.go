package content

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"
)

// The inherited System.IO.BinaryReader surface.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # Every read goes through CNA, and that is not a choice
//
// CNA's content reader BORROWS the stream for its whole lifetime. Reading the
// same stream from the Go side as well would advance one position twice and
// desynchronise the two readers, so every member below is built on
// `cna_content_reader_read_bytes_exact` rather than on the io.Reader the
// container handed back.
//
// That is also why three inherited members are excluded rather than projected,
// each for its own measured reason -- see the adapter registry.
//
// # The byte order is the FORMAT's, not the host's
//
// A compiled asset is little-endian wherever it is read, so every decode below
// names the order explicitly instead of taking the platform's.

// Read7BitEncodedInt is BinaryReader::Read7BitEncodedInt, measured in full:
//
//	int count = 0, shift = 0;
//	byte b;
//	do {
//	    if (shift == 35) throw new FormatException(Format_Bad7BitInt32);
//	    b = ReadByte();
//	    count |= (b & 0x7F) << (shift & 31);
//	    shift += 7;
//	} while ((b & 0x80) != 0);
//	return count;
//
// Two details a plausible implementation would get wrong. The shift is MASKED
// with 31 before use, and the refusal fires at `shift == 35` -- which is after
// FIVE bytes, not four. The member is `protected` in the reference, so it is
// not inherited public surface; it is unexported here for the same reason, and
// ReadString is what needs it.
func (b *binaryReaderBase) read7BitEncodedInt() (int32, error) {
	var count int32
	var shift uint32
	for {
		if shift == 35 {
			return 0, fmt.Errorf("%w: the 7-bit encoded length is malformed", errContentLoad)
		}
		b, err := b.ReadByte()
		if err != nil {
			return 0, err
		}
		count |= int32(b&0x7F) << (shift & 31)
		shift += 7
		if b&0x80 == 0 {
			return count, nil
		}
	}
}

// binaryReaderBase is the private adapter that models System.IO.BinaryReader.
//
// Its state is the CNA reader handle, which is what "the stream the base holds"
// is here: BinaryReader owns a Stream and CNA's content reader is what owns it
// in this projection. ContentReader holds one of these in an unexported field
// and forwards the inherited members, which is the settled composition rule.
type binaryReaderBase struct {
	// owner is what the reads pull bytes from. It is narrowed to that single
	// role rather than held as *ContentReader, because fetching bytes is the
	// ONLY thing the adapter ever asks of the derived reader -- the disposal
	// latch and the handle both live behind readExact, which is why the
	// adapter does not hold either.
	owner contentByteSource
}

// contentByteSource is the one thing binaryReaderBase needs from the reader it
// belongs to: an exact-length read that has already applied the disposal latch
// and the short-read refusal. *ContentReader is its only production
// implementation.
type contentByteSource interface {
	readExact(count int32, readerName string) ([]byte, error)
}

// readExact reads exactly count bytes, which is what CNA's read_bytes_exact
// promises: a short read is a failure rather than a partial answer.
func (b *binaryReaderBase) readExact(count int32, readerName string) ([]byte, error) {
	return b.owner.readExact(count, readerName)
}

// The three `Read` overloads -- Read(), Read(Byte[],Int32,Int32) and
// Read(Char[],Int32,Int32) -- are NOT projected, and the reason is a mechanism
// limitation rather than a measured one: the inherited-projection model has no
// way to express an overloaded INHERITED member, so the registry cannot name
// them and the verifier rejects a Go type that declares them anyway.
//
// That is recorded rather than worked around. Every other inherited member is
// here, and a future milestone that teaches the model about inherited overloads
// would gain three more without changing anything measured.

// ReadBoolean is BinaryReader::ReadBoolean, one byte, and anything non-zero is
// true -- the reference compares against zero rather than against one.
func (b *binaryReaderBase) ReadBoolean() (bool, error) {
	buffer, err := b.readExact(1, "ReadBoolean")
	if err != nil {
		return false, err
	}
	return buffer[0] != 0, nil
}

// ReadByte is BinaryReader::ReadByte.
func (b *binaryReaderBase) ReadByte() (uint8, error) {
	buffer, err := b.readExact(1, "ReadByte")
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

// ReadSByte is BinaryReader::ReadSByte, the same byte read as signed.
func (b *binaryReaderBase) ReadSByte() (int8, error) {
	value, err := b.ReadByte()
	return int8(value), err
}

// ReadInt16 is BinaryReader::ReadInt16.
func (b *binaryReaderBase) ReadInt16() (int16, error) {
	buffer, err := b.readExact(2, "ReadInt16")
	if err != nil {
		return 0, err
	}
	return int16(binary.LittleEndian.Uint16(buffer)), nil
}

// ReadUInt16 is BinaryReader::ReadUInt16.
func (b *binaryReaderBase) ReadUInt16() (uint16, error) {
	buffer, err := b.readExact(2, "ReadUInt16")
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buffer), nil
}

// ReadInt32 is BinaryReader::ReadInt32.
func (b *binaryReaderBase) ReadInt32() (int32, error) {
	buffer, err := b.readExact(4, "ReadInt32")
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(buffer)), nil
}

// ReadUInt32 is BinaryReader::ReadUInt32.
func (b *binaryReaderBase) ReadUInt32() (uint32, error) {
	buffer, err := b.readExact(4, "ReadUInt32")
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buffer), nil
}

// ReadInt64 is BinaryReader::ReadInt64.
func (b *binaryReaderBase) ReadInt64() (int64, error) {
	buffer, err := b.readExact(8, "ReadInt64")
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(buffer)), nil
}

// ReadUInt64 is BinaryReader::ReadUInt64.
func (b *binaryReaderBase) ReadUInt64() (uint64, error) {
	buffer, err := b.readExact(8, "ReadUInt64")
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buffer), nil
}

// ReadChar is BinaryReader::ReadChar, which decodes ONE character through the
// reader's encoding -- UTF-8 for every reader the profile builds.
//
// A character is not a byte: the reference reads bytes until the decoder emits
// one, which for UTF-8 is one to four. `System.Char` is a UTF-16 code unit, so
// a rune outside the basic plane does not fit -- and the reference has the same
// problem, answering the first surrogate.
func (b *binaryReaderBase) ReadChar() (uint16, error) {
	buffer := make([]byte, 0, 4)
	for i := 0; i < 4; i++ {
		b, err := b.ReadByte()
		if err != nil {
			return 0, err
		}
		buffer = append(buffer, b)
		if value, size := utf8.DecodeRune(buffer); value != utf8.RuneError || size > 1 {
			if value > 0xFFFF {
				// Outside the basic plane. UTF-16 needs a surrogate pair and a
				// single System.Char holds only the first, which is what the
				// reference's own decoder produces here.
				value -= 0x10000
				return uint16(0xD800 + (value >> 10)), nil
			}
			return uint16(value), nil
		}
	}
	return 0, fmt.Errorf("%w: the character encoding is malformed", errContentLoad)
}

// ReadChars is BinaryReader::ReadChars(Int32), which reads exactly count
// characters and returns fewer only at the end of the stream.
func (b *binaryReaderBase) ReadChars(count int32) ([]uint16, error) {
	if count < 0 {
		return nil, contentArgumentNullError("count")
	}
	result := make([]uint16, 0, count)
	for i := int32(0); i < count; i++ {
		value, err := b.ReadChar()
		if err != nil {
			return result, err
		}
		result = append(result, value)
	}
	return result, nil
}

// ReadString is BinaryReader::ReadString, a 7-bit-encoded BYTE length followed
// by that many bytes decoded through the encoding.
//
// The length counts BYTES, not characters, which is why it is read before the
// decode rather than after.
func (b *binaryReaderBase) ReadString() (string, error) {
	length, err := b.read7BitEncodedInt()
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", fmt.Errorf("%w: the string length is negative", errContentLoad)
	}
	if length == 0 {
		return "", nil
	}
	buffer, err := b.readExact(length, "ReadString")
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

// ReadBytes is BinaryReader::ReadBytes(Int32).
func (b *binaryReaderBase) ReadBytes(count int32) ([]uint8, error) {
	if count < 0 {
		return nil, contentArgumentNullError("count")
	}
	if count == 0 {
		return []uint8{}, nil
	}
	return b.readExact(count, "ReadBytes")
}

// Close is BinaryReader::Close, which the reference implements as
// `Dispose(true)`. It releases the native reader, and CNA's reader closes the
// stream it borrowed.
func (r *ContentReader) Close() error { return r.close() }

// Dispose is BinaryReader::Dispose(), which forwards to Close.
func (r *ContentReader) Dispose() error { return r.close() }

// ReadBoolean is the inherited BinaryReader::ReadBoolean.
func (r *ContentReader) ReadBoolean() (bool, error) { return r.base.ReadBoolean() }

// ReadByte is the inherited BinaryReader::ReadByte.
func (r *ContentReader) ReadByte() (uint8, error) { return r.base.ReadByte() }

// ReadSByte is the inherited BinaryReader::ReadSByte.
func (r *ContentReader) ReadSByte() (int8, error) { return r.base.ReadSByte() }

// ReadInt16 is the inherited BinaryReader::ReadInt16.
func (r *ContentReader) ReadInt16() (int16, error) { return r.base.ReadInt16() }

// ReadUInt16 is the inherited BinaryReader::ReadUInt16.
func (r *ContentReader) ReadUInt16() (uint16, error) { return r.base.ReadUInt16() }

// readSingle and readDouble are the four- and eight-byte IEEE-754 decodes.
//
// They are UNEXPORTED, and that is the whole point: `ReadSingle` and
// `ReadDouble` are declared on ContentReader by the pinned contract -- XNA
// re-declares both rather than inheriting them -- so the public members belong
// there and not to this adapter. What lives here is only the shared decode the
// adapter's other reads already use, which keeps one answer about byte order in
// one place.
//
// CNA has no float route -- its content reader hands back floats only for the
// five framework-typed reads -- so the projection reads the bytes and decodes
// them. The byte order is the FORMAT's, not the host's.
func (b *binaryReaderBase) readSingle() (float32, error) {
	buffer, err := b.readExact(4, "ReadSingle")
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(buffer)), nil
}

func (b *binaryReaderBase) readDouble() (float64, error) {
	buffer, err := b.readExact(8, "ReadDouble")
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(buffer)), nil
}

// ReadInt32 is the inherited BinaryReader::ReadInt32.
func (r *ContentReader) ReadInt32() (int32, error) { return r.base.ReadInt32() }

// ReadSingle is ContentReader::ReadSingle(), which the contract DECLARES on
// ContentReader rather than leaving inherited. The body is the adapter's shared
// decode, so this file holds one answer about byte order.
func (r *ContentReader) ReadSingle() (float32, error) { return r.base.readSingle() }

// ReadDouble is ContentReader::ReadDouble(), declared the same way.
func (r *ContentReader) ReadDouble() (float64, error) { return r.base.readDouble() }

// ReadUInt32 is the inherited BinaryReader::ReadUInt32.
func (r *ContentReader) ReadUInt32() (uint32, error) { return r.base.ReadUInt32() }

// ReadInt64 is the inherited BinaryReader::ReadInt64.
func (r *ContentReader) ReadInt64() (int64, error) { return r.base.ReadInt64() }

// ReadUInt64 is the inherited BinaryReader::ReadUInt64.
func (r *ContentReader) ReadUInt64() (uint64, error) { return r.base.ReadUInt64() }

// ReadChar is the inherited BinaryReader::ReadChar.
func (r *ContentReader) ReadChar() (uint16, error) { return r.base.ReadChar() }

// ReadString is the inherited BinaryReader::ReadString.
func (r *ContentReader) ReadString() (string, error) { return r.base.ReadString() }

// ReadChars is the inherited BinaryReader::ReadChars(Int32).
func (r *ContentReader) ReadChars(count int32) ([]uint16, error) { return r.base.ReadChars(count) }

// ReadBytes is the inherited BinaryReader::ReadBytes(Int32).
func (r *ContentReader) ReadBytes(count int32) ([]uint8, error) { return r.base.ReadBytes(count) }
