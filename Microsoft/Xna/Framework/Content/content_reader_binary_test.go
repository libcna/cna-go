package content

import (
	"fmt"
	"testing"
)

// cannedBytes is a contentByteSource over a fixed slice. It exists because the
// decoding below is the part of the inherited BinaryReader surface that is
// PURE: readExact is the only member that reaches CNA, so substituting it
// leaves every byte-order, length and encoding decision under test while the
// handle, the disposal latch and the short-read refusal stay where they are.
type cannedBytes struct {
	data []byte
	at   int
	// names records the readerName each call passed, which is what the
	// short-read message would report; a decoder that asks for the wrong
	// length is visible here as well as in the answer.
	lengths []int32
}

func (c *cannedBytes) readExact(count int32, readerName string) ([]byte, error) {
	c.lengths = append(c.lengths, count)
	if c.at+int(count) > len(c.data) {
		return nil, fmt.Errorf("%w: %s ran past the canned bytes", errContentLoad, readerName)
	}
	slice := c.data[c.at : c.at+int(count)]
	c.at += int(count)
	return slice, nil
}

func readerOver(data ...byte) (*binaryReaderBase, *cannedBytes) {
	source := &cannedBytes{data: data}
	return &binaryReaderBase{owner: source}, source
}

// TestTheFixedWidthReadsAreLittleEndian pins the byte order against a value
// whose bytes differ in every position, so a big-endian decode cannot agree by
// accident. The order is the FORMAT's, not the host's.
func TestTheFixedWidthReadsAreLittleEndian(t *testing.T) {
	reader, source := readerOver(0x01, 0x02)
	got16, err := reader.ReadInt16()
	if err != nil || got16 != 0x0201 {
		t.Fatalf("ReadInt16 = %#x, %v; want 0x0201", got16, err)
	}
	if len(source.lengths) != 1 || source.lengths[0] != 2 {
		t.Fatalf("ReadInt16 asked for %v bytes; want [2]", source.lengths)
	}

	reader, _ = readerOver(0x01, 0x02, 0x03, 0x04)
	got32, err := reader.ReadInt32()
	if err != nil || got32 != 0x04030201 {
		t.Fatalf("ReadInt32 = %#x, %v; want 0x04030201", got32, err)
	}

	reader, _ = readerOver(0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08)
	got64, err := reader.ReadInt64()
	if err != nil || got64 != 0x0807060504030201 {
		t.Fatalf("ReadInt64 = %#x, %v; want 0x0807060504030201", got64, err)
	}

	reader, _ = readerOver(0x01, 0x02)
	gotU16, err := reader.ReadUInt16()
	if err != nil || gotU16 != 0x0201 {
		t.Fatalf("ReadUInt16 = %#x, %v; want 0x0201", gotU16, err)
	}

	reader, _ = readerOver(0x01, 0x02, 0x03, 0x04)
	gotU32, err := reader.ReadUInt32()
	if err != nil || gotU32 != 0x04030201 {
		t.Fatalf("ReadUInt32 = %#x, %v; want 0x04030201", gotU32, err)
	}

	reader, _ = readerOver(0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08)
	gotU64, err := reader.ReadUInt64()
	if err != nil || gotU64 != 0x0807060504030201 {
		t.Fatalf("ReadUInt64 = %#x, %v; want 0x0807060504030201", gotU64, err)
	}
}

// TestReadSingleAndDoubleDecodeIEEE754 pins the float decodes against bit
// patterns that are exactly representable, so the comparison is not a
// tolerance question.
//
// These two are ContentReader's own declared members, not inherited ones; what
// is tested here is the shared decode their bodies use.
func TestReadSingleAndDoubleDecodeIEEE754(t *testing.T) {
	// 1.0f is 0x3F800000.
	reader, _ := readerOver(0x00, 0x00, 0x80, 0x3F)
	single, err := reader.readSingle()
	if err != nil || single != 1.0 {
		t.Fatalf("ReadSingle = %v, %v; want 1", single, err)
	}
	// -2.0f is 0xC0000000.
	reader, _ = readerOver(0x00, 0x00, 0x00, 0xC0)
	if single, err = reader.readSingle(); err != nil || single != -2.0 {
		t.Fatalf("ReadSingle = %v, %v; want -2", single, err)
	}
	// 1.0 is 0x3FF0000000000000.
	reader, _ = readerOver(0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F)
	double, err := reader.readDouble()
	if err != nil || double != 1.0 {
		t.Fatalf("ReadDouble = %v, %v; want 1", double, err)
	}
}

// TestReadBooleanIsNonZeroNotOne is the reference's own comparison. A file
// holding 2 in a boolean field reads as true, and an implementation that
// compares against 1 would answer false.
func TestReadBooleanIsNonZeroNotOne(t *testing.T) {
	for _, value := range []byte{1, 2, 0x80, 0xFF} {
		reader, _ := readerOver(value)
		got, err := reader.ReadBoolean()
		if err != nil || !got {
			t.Fatalf("ReadBoolean(%#x) = %v, %v; want true", value, got, err)
		}
	}
	reader, _ := readerOver(0)
	if got, err := reader.ReadBoolean(); err != nil || got {
		t.Fatalf("ReadBoolean(0) = %v, %v; want false", got, err)
	}
}

// TestReadByteAndSByteShareTheByte pins that the signed read reinterprets
// rather than re-reads.
func TestReadByteAndSByteShareTheByte(t *testing.T) {
	reader, source := readerOver(0xFF)
	signed, err := reader.ReadSByte()
	if err != nil || signed != -1 {
		t.Fatalf("ReadSByte(0xFF) = %d, %v; want -1", signed, err)
	}
	if len(source.lengths) != 1 {
		t.Fatalf("ReadSByte took %d reads; want 1", len(source.lengths))
	}
	reader, _ = readerOver(0xFF)
	if unsigned, err := reader.ReadByte(); err != nil || unsigned != 0xFF {
		t.Fatalf("ReadByte(0xFF) = %d, %v; want 255", unsigned, err)
	}
}

// TestSevenBitEncodedIntMatchesTheReference walks the four details a plausible
// implementation gets wrong: the continuation bit is 0x80, only the low seven
// bits carry payload, the shift is masked with 31, and the refusal fires after
// FIVE bytes rather than four.
func TestSevenBitEncodedIntMatchesTheReference(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		want  int32
	}{
		{"one byte", []byte{0x00}, 0},
		{"one byte, high payload bit", []byte{0x7F}, 127},
		// 0x80 0x01 is 0 in the low group and 1 in the next: 1 << 7.
		{"two bytes", []byte{0x80, 0x01}, 128},
		// 0x40 has bit 6 set, which is PAYLOAD and not continuation.
		{"bit six is payload", []byte{0x40}, 0x40},
		{"five bytes reaching the sign bit", []byte{0x80, 0x80, 0x80, 0x80, 0x08}, -2147483648},
		{"maximum positive", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x07}, 2147483647},
	}
	for _, one := range cases {
		reader, _ := readerOver(one.bytes...)
		got, err := reader.read7BitEncodedInt()
		if err != nil {
			t.Fatalf("%s: read7BitEncodedInt: %v", one.name, err)
		}
		if got != one.want {
			t.Fatalf("%s: read7BitEncodedInt = %d; want %d", one.name, got, one.want)
		}
	}

	// The refusal is at shift == 35, so a fifth byte is still consumed and only
	// a sixth is refused. That the boundary is 35 and not 28 is exactly what
	// the maximum-positive case above proves from the other side.
	reader, _ := readerOver(0x80, 0x80, 0x80, 0x80, 0x80, 0x01)
	if _, err := reader.read7BitEncodedInt(); err == nil {
		t.Fatal("a six-byte 7-bit integer was accepted")
	}
}

// TestReadStringCountsBytesNotCharacters is the detail the length prefix turns
// on. Two-byte UTF-8 characters make the byte count and the character count
// differ, so an implementation reading the wrong one desynchronises the stream.
func TestReadStringCountsBytesNotCharacters(t *testing.T) {
	// "ab" is 2 bytes; "áé" is 4 bytes but 2 characters.
	text := "áé"
	if len(text) != 4 {
		t.Fatalf("the fixture is %d bytes; the test needs 4", len(text))
	}
	data := append([]byte{byte(len(text))}, text...)
	// A trailing byte the decode must NOT consume.
	data = append(data, 0x2A)
	reader, source := readerOver(data...)
	got, err := reader.ReadString()
	if err != nil || got != text {
		t.Fatalf("ReadString = %q, %v; want %q", got, err, text)
	}
	if source.lengths[len(source.lengths)-1] != 4 {
		t.Fatalf("ReadString asked for %d bytes; want 4", source.lengths[len(source.lengths)-1])
	}
	// The stream is positioned exactly after the string.
	if next, err := reader.ReadByte(); err != nil || next != 0x2A {
		t.Fatalf("the byte after the string was %#x, %v; want 0x2A", next, err)
	}
}

// TestReadStringTakesTheEmptyCaseWithoutReading pins that a zero length is
// answered from the length alone. Asking CNA for zero bytes is not the same
// thing: the reference returns String.Empty before it reads.
func TestReadStringTakesTheEmptyCaseWithoutReading(t *testing.T) {
	reader, source := readerOver(0x00)
	got, err := reader.ReadString()
	if err != nil || got != "" {
		t.Fatalf("ReadString = %q, %v; want the empty string", got, err)
	}
	if len(source.lengths) != 1 {
		t.Fatalf("the empty string took %d reads; want 1, the length byte alone", len(source.lengths))
	}
}

// TestReadCharDecodesThroughTheEncoding pins that a character is not a byte.
func TestReadCharDecodesThroughTheEncoding(t *testing.T) {
	reader, source := readerOver('A')
	got, err := reader.ReadChar()
	if err != nil || got != 'A' {
		t.Fatalf("ReadChar = %#x, %v; want 'A'", got, err)
	}
	if len(source.lengths) != 1 {
		t.Fatalf("an ASCII character took %d reads; want 1", len(source.lengths))
	}

	// U+00E1 is two bytes in UTF-8, so the decode must read a second one.
	reader, source = readerOver(0xC3, 0xA1)
	if got, err = reader.ReadChar(); err != nil || got != 0x00E1 {
		t.Fatalf("ReadChar = %#x, %v; want 0x00E1", got, err)
	}
	if len(source.lengths) != 2 {
		t.Fatalf("a two-byte character took %d reads; want 2", len(source.lengths))
	}

	// ReadChars runs the same decode count times.
	reader, _ = readerOver(0xC3, 0xA1, 'z')
	chars, err := reader.ReadChars(2)
	if err != nil || len(chars) != 2 || chars[0] != 0x00E1 || chars[1] != 'z' {
		t.Fatalf("ReadChars = %#x, %v; want [0xE1 0x7A]", chars, err)
	}
	if _, err := reader.ReadChars(-1); err == nil {
		t.Fatal("ReadChars accepted a negative count")
	}
}

// TestReadBytesRefusesANegativeCount pins the guard, and that zero is answered
// without a read.
func TestReadBytesRefusesANegativeCount(t *testing.T) {
	reader, source := readerOver(1, 2, 3)
	if _, err := reader.ReadBytes(-1); err == nil {
		t.Fatal("ReadBytes accepted a negative count")
	}
	if len(source.lengths) != 0 {
		t.Fatal("the refused read still reached the byte source")
	}
	empty, err := reader.ReadBytes(0)
	if err != nil || len(empty) != 0 {
		t.Fatalf("ReadBytes(0) = %v, %v; want an empty slice", empty, err)
	}
	if len(source.lengths) != 0 {
		t.Fatal("a zero-length read still reached the byte source")
	}
	got, err := reader.ReadBytes(3)
	if err != nil || len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("ReadBytes(3) = %v, %v; want [1 2 3]", got, err)
	}
}
