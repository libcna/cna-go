package graphics

import (
	"errors"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Foundation 64 — Microsoft.Xna.Framework.Graphics.VertexElementValidator.
// ---------------------------------------------------------------------------
//
// The validator is `.class private abstract auto ansi sealed` and every one of
// its members is `assembly`. It is not public surface in the reference and it
// is not public surface here: nothing below is exported, and no projected type
// names it.
//
// It is reproduced rather than approximated because it is the ENTIRE failure
// surface of VertexDeclaration's two constructors. A projection that accepted a
// declaration the reference rejects would hand CNA a layout XNA never had, and
// one that rejected a declaration the reference accepts would refuse a legal
// program. Both are visible only if the algorithm is the reference's own.
//
// Six FrameworkResources strings appear here, all in
// Microsoft.Xna.Framework.dll rather than the Graphics assembly the type lives
// in, which is where FrameworkResources is.

// vertexElementOffsetNotMultipleFour is thrown TWICE by Validate, at two
// different sites, for two different quantities: the stride and one element's
// offset. The reference uses one message for both, and the sentence names both.
const vertexElementOffsetNotMultipleFour = "Invalid VertexDeclaration. Vertex stride and VertexElement.Offset must be multiples of four."

// The four formatted messages. Each carries `{0}{1}` -- a usage immediately
// followed by a usage index, with no separator -- and the overlap one carries
// four placeholders naming two elements.
const (
	vertexElementBadUsage      = "Invalid VertexDeclaration. Usage {0}{1} is out of range."
	vertexElementOutsideStride = "Invalid VertexDeclaration. Element {0}{1} does not fit within the specified vertex stride."
	duplicateVertexElement     = "Invalid VertexDeclaration. Duplicate element {0}{1}."
	vertexElementsOverlap      = "Invalid VertexDeclaration. Elements {0}{1} and {2}{3} are overlapping."
)

// vertexElementTypeSize is VertexElementValidator::GetTypeSize, a `switch` over
// the twelve declared VertexElementFormat values with a `ldc.i4.0` default:
//
//	Single 4   Vector2 8   Vector3 12  Vector4 16
//	Color  4   Byte4   4   Short2   4  Short4   8
//	NormalizedShort2 4     NormalizedShort4 8
//	HalfVector2      4     HalfVector4      8
//
// A format outside the twelve answers ZERO, which is not a failure: it makes
// the element occupy no bytes, so the stride and overlap checks treat it as
// empty. That is the reference's behaviour and it is reproduced as written.
func vertexElementTypeSize(format VertexElementFormat) int32 {
	switch format {
	case VertexElementFormatSingle, VertexElementFormatColor, VertexElementFormatByte4,
		VertexElementFormatShort2, VertexElementFormatNormalizedShort2, VertexElementFormatHalfVector2:
		return 4
	case VertexElementFormatVector2, VertexElementFormatShort4,
		VertexElementFormatNormalizedShort4, VertexElementFormatHalfVector4:
		return 8
	case VertexElementFormatVector3:
		return 12
	case VertexElementFormatVector4:
		return 16
	default:
		return 0
	}
}

// vertexStrideForElements is VertexElementValidator::GetVertexStride:
//
//	int max = 0;
//	for (int i = 0; i < elements.Length; i++)
//	{
//	    int end = elements[i].Offset + GetTypeSize(elements[i].VertexElementFormat);
//	    if (max < end) max = end;
//	}
//	return max;
//
// It is the largest END offset, NOT the sum of the sizes: a declaration with a
// gap in it strides over the gap, and one whose elements overlap does not
// double-count. Validate is what rejects the overlap; this only measures.
func vertexStrideForElements(elements []VertexElement) int32 {
	var stride int32
	for index := range elements {
		end := elements[index].Offset() + vertexElementTypeSize(elements[index].VertexElementFormat())
		if stride < end {
			stride = end
		}
	}
	return stride
}

// validateVertexElements is VertexElementValidator::Validate(int32,
// VertexElement[]), 603 bytes of IL, reproduced check for check and in order.
//
// The order is load-bearing: an element that is both badly-used and outside the
// stride reports the usage, because that check comes first, and a caller who
// fixed the second failure first would still be refused. The five checks, in
// the order the reference performs them:
//
//	stride > 0                            ArgumentOutOfRangeException("vertexStride")
//	stride % 4 == 0                       ArgumentException(OffsetNotMultipleFour)
//	then, per element, in array order:
//	  0 <= usage <= 12                    ArgumentException(BadUsage)
//	  0 <= offset && offset+size <= stride ArgumentException(OutsideStride)
//	  offset % 4 == 0                     ArgumentException(OffsetNotMultipleFour)
//	  no EARLIER element shares its
//	    (usage, usageIndex) pair          ArgumentException(DuplicateVertexElement)
//	  no byte it occupies is already
//	    claimed by an earlier element     ArgumentException(ElementsOverlap)
//
// The occupancy map is the reference's own `int[vertexStride]` initialised to
// -1 and filled with the index of the element that claimed each byte, which is
// what lets the overlap message name BOTH elements. A boolean map would report
// the same failures and could not say what they collided with.
func validateVertexElements(vertexStride int32, elements []VertexElement) error {
	if vertexStride <= 0 {
		return fmt.Errorf("%w: vertexStride", errArgumentOutOfRange)
	}
	if vertexStride&3 != 0 {
		return fmt.Errorf("%w: %s", errArgument, vertexElementOffsetNotMultipleFour)
	}
	occupiedBy := make([]int32, vertexStride)
	for index := range occupiedBy {
		occupiedBy[index] = -1
	}
	for index := range elements {
		offset := elements[index].Offset()
		size := vertexElementTypeSize(elements[index].VertexElementFormat())
		usage := elements[index].VertexElementUsage()
		if usage < 0 || usage > VertexElementUsageTessellateFactor {
			// `{1}` is String.Empty at this site, and it is passed anyway. The
			// reference formats a two-argument message with an empty second
			// argument, so the sentence ends "Usage Position is out of range."
			return fmt.Errorf("%w: %s", errArgument,
				formatVertexElementMessage(vertexElementBadUsage, vertexElementUsageString(usage), ""))
		}
		if offset < 0 || offset+size > vertexStride {
			return fmt.Errorf("%w: %s", errArgument,
				formatVertexElementMessage(vertexElementOutsideStride,
					vertexElementUsageString(usage), fmt.Sprint(elements[index].UsageIndex())))
		}
		if offset&3 != 0 {
			return fmt.Errorf("%w: %s", errArgument, vertexElementOffsetNotMultipleFour)
		}
		for earlier := 0; earlier < index; earlier++ {
			if elements[index].VertexElementUsage() == elements[earlier].VertexElementUsage() &&
				elements[index].UsageIndex() == elements[earlier].UsageIndex() {
				return fmt.Errorf("%w: %s", errArgument,
					formatVertexElementMessage(duplicateVertexElement,
						vertexElementUsageString(usage), fmt.Sprint(elements[index].UsageIndex())))
			}
		}
		for at := offset; at < offset+size; at++ {
			if occupiedBy[at] >= 0 {
				owner := elements[occupiedBy[at]]
				return fmt.Errorf("%w: %s", errArgument,
					formatVertexElementMessage(vertexElementsOverlap,
						vertexElementUsageString(owner.VertexElementUsage()), fmt.Sprint(owner.UsageIndex()),
						vertexElementUsageString(usage), fmt.Sprint(elements[index].UsageIndex())))
			}
			occupiedBy[at] = int32(index)
		}
	}
	return nil
}

// formatVertexElementMessage substitutes `{0}`, `{1}`, ... positionally.
//
// The CLR's String.Format placeholders are kept in the constants rather than
// rewritten to Go verbs, for the reason the state-object freeze message is: the
// resource-string verifier compares the constant against the exact bytes the
// retained assembly holds, and a rewritten placeholder would not be those
// bytes.
func formatVertexElementMessage(message string, arguments ...string) string {
	for index, argument := range arguments {
		message = strings.ReplaceAll(message, fmt.Sprintf("{%d}", index), argument)
	}
	return message
}

// errArgument and errArgumentOutOfRange project the two BCL exception
// identities Validate throws. They are unexported: the XNA public contract
// declares no error type on these members, and CNA-Go invents none.
var (
	errArgument           = errors.New("argument is not valid")
	errArgumentOutOfRange = errors.New("argument is out of range")
)
