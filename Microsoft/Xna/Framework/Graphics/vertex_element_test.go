package graphics

import (
	"math"
	"testing"
)

func TestVertexElementEnumRawValues(t *testing.T) {
	formats := []VertexElementFormat{
		VertexElementFormatSingle,
		VertexElementFormatVector2,
		VertexElementFormatVector3,
		VertexElementFormatVector4,
		VertexElementFormatColor,
		VertexElementFormatByte4,
		VertexElementFormatShort2,
		VertexElementFormatShort4,
		VertexElementFormatNormalizedShort2,
		VertexElementFormatNormalizedShort4,
		VertexElementFormatHalfVector2,
		VertexElementFormatHalfVector4,
	}
	for raw, value := range formats {
		if value != VertexElementFormat(raw) {
			t.Fatalf("format value %d = %d", raw, value)
		}
	}

	usages := []VertexElementUsage{
		VertexElementUsagePosition,
		VertexElementUsageColor,
		VertexElementUsageTextureCoordinate,
		VertexElementUsageNormal,
		VertexElementUsageBinormal,
		VertexElementUsageTangent,
		VertexElementUsageBlendIndices,
		VertexElementUsageBlendWeight,
		VertexElementUsageDepth,
		VertexElementUsageFog,
		VertexElementUsagePointSize,
		VertexElementUsageSample,
		VertexElementUsageTessellateFactor,
	}
	for raw, value := range usages {
		if value != VertexElementUsage(raw) {
			t.Fatalf("usage value %d = %d", raw, value)
		}
	}
}

func TestVertexElementZeroAndConstructor(t *testing.T) {
	var zero VertexElement
	constructed := NewVertexElement(0, VertexElementFormatSingle, VertexElementUsagePosition, 0)
	if zero.Offset() != 0 || zero.VertexElementFormat() != VertexElementFormatSingle ||
		zero.VertexElementUsage() != VertexElementUsagePosition || zero.UsageIndex() != 0 {
		t.Fatalf("zero getters = %d,%d,%d,%d", zero.Offset(), zero.VertexElementFormat(), zero.VertexElementUsage(), zero.UsageIndex())
	}
	if !zero.Equals(constructed) ||
		!VertexElementOperatorEqualityByVertexElementAndVertexElement(zero, constructed) ||
		zero.GetHashCode() != constructed.GetHashCode() || zero.ToString() != constructed.ToString() {
		t.Fatalf("zero and constructor differ: %#v %#v", zero, constructed)
	}
}

func TestVertexElementPropertiesPreserveInt32AndUndefinedEnums(t *testing.T) {
	const unknownFormat VertexElementFormat = 12345
	const unknownUsage VertexElementUsage = -23456
	values := []int32{0, 1, -1, math.MinInt32, math.MaxInt32}
	for _, value := range values {
		element := NewVertexElement(value, unknownFormat, unknownUsage, value)
		if element.Offset() != value || element.UsageIndex() != value ||
			element.VertexElementFormat() != unknownFormat || element.VertexElementUsage() != unknownUsage {
			t.Fatalf("constructor did not preserve %d: %#v", value, element)
		}
		element.SetOffset(-value - 1)
		element.SetUsageIndex(-value - 1)
		element.SetVertexElementFormat(VertexElementFormat(int32(unknownFormat) + value))
		element.SetVertexElementUsage(VertexElementUsage(int32(unknownUsage) + value))
		if element.Offset() != -value-1 || element.UsageIndex() != -value-1 ||
			element.VertexElementFormat() != VertexElementFormat(int32(unknownFormat)+value) ||
			element.VertexElementUsage() != VertexElementUsage(int32(unknownUsage)+value) {
			t.Fatalf("setters did not preserve int32 domain for %d: %#v", value, element)
		}
	}
}

func TestVertexElementCopySemanticsForEveryProperty(t *testing.T) {
	original := NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageTextureCoordinate, 7)

	offsetCopy := original
	offsetCopy.SetOffset(13)
	formatCopy := original
	formatCopy.SetVertexElementFormat(VertexElementFormatHalfVector4)
	usageCopy := original
	usageCopy.SetVertexElementUsage(VertexElementUsageTangent)
	indexCopy := original
	indexCopy.SetUsageIndex(8)

	if original.Offset() != 12 || original.VertexElementFormat() != VertexElementFormatVector3 ||
		original.VertexElementUsage() != VertexElementUsageTextureCoordinate || original.UsageIndex() != 7 {
		t.Fatalf("copy mutation changed original: %#v", original)
	}
	if offsetCopy.Offset() != 13 || formatCopy.VertexElementFormat() != VertexElementFormatHalfVector4 ||
		usageCopy.VertexElementUsage() != VertexElementUsageTangent || indexCopy.UsageIndex() != 8 {
		t.Fatalf("copy mutations were not retained")
	}
}

func TestVertexElementEqualityAndOperators(t *testing.T) {
	base := NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageTextureCoordinate, 7)
	if !base.Equals(base) || base.Equals(&base) || base.Equals(nil) || base.Equals(int32(12)) {
		t.Fatalf("object equality type behavior failed")
	}

	differences := []VertexElement{base, base, base, base}
	differences[0].SetOffset(13)
	differences[1].SetVertexElementFormat(VertexElementFormatVector4)
	differences[2].SetVertexElementUsage(VertexElementUsageNormal)
	differences[3].SetUsageIndex(8)
	for index, different := range differences {
		if base.Equals(different) || VertexElementOperatorEqualityByVertexElementAndVertexElement(base, different) ||
			!VertexElementOperatorInequalityByVertexElementAndVertexElement(base, different) {
			t.Fatalf("field difference %d was ignored", index)
		}
	}

	unknownA := NewVertexElement(1, 12345, -23456, 2)
	unknownB := NewVertexElement(1, 12345, -23456, 2)
	unknownDifferent := NewVertexElement(1, 12346, -23456, 2)
	if !unknownA.Equals(unknownB) || !VertexElementOperatorEqualityByVertexElementAndVertexElement(unknownA, unknownB) ||
		!VertexElementOperatorInequalityByVertexElementAndVertexElement(unknownA, unknownDifferent) {
		t.Fatalf("undefined enum equality failed")
	}
}

func TestVertexElementXNAHashFixtures(t *testing.T) {
	fixtures := []struct {
		name     string
		element  VertexElement
		expected int32
	}{
		{"zero-fallback", VertexElement{}, math.MaxInt32},
		{"ordinary", NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageTextureCoordinate, 7), 11},
		{"negative", NewVertexElement(-16, VertexElementFormatHalfVector4, VertexElementUsageTangent, -3), 3},
		{"undefined-enums", NewVertexElement(123, 12345, -23456, -456), 27162},
		{"int32-boundaries", NewVertexElement(math.MinInt32, VertexElementFormatHalfVector4, VertexElementUsageTessellateFactor, math.MaxInt32), -8},
		{"nonzero-collision-fallback", NewVertexElement(1, VertexElementFormatVector3, VertexElementUsageNormal, 0), math.MaxInt32},
	}
	for _, fixture := range fixtures {
		if actual := fixture.element.GetHashCode(); actual != fixture.expected {
			t.Errorf("%s hash = %d, want %d", fixture.name, actual, fixture.expected)
		}
	}
}

func TestVertexElementXNAStringFixtures(t *testing.T) {
	fixtures := []struct {
		name     string
		element  VertexElement
		expected string
	}{
		{"zero", VertexElement{}, "{Offset:0 Format:Single Usage:Position UsageIndex:0}"},
		{"ordinary", NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageTextureCoordinate, 7), "{Offset:12 Format:Vector3 Usage:TextureCoordinate UsageIndex:7}"},
		{"negative", NewVertexElement(-16, VertexElementFormatHalfVector4, VertexElementUsageTangent, -3), "{Offset:-16 Format:HalfVector4 Usage:Tangent UsageIndex:-3}"},
		{"undefined-enums", NewVertexElement(123, 12345, -23456, -456), "{Offset:123 Format:12345 Usage:-23456 UsageIndex:-456}"},
		{"int32-boundaries", NewVertexElement(math.MinInt32, VertexElementFormatHalfVector4, VertexElementUsageTessellateFactor, math.MaxInt32), "{Offset:-2147483648 Format:HalfVector4 Usage:TessellateFactor UsageIndex:2147483647}"},
	}
	for _, fixture := range fixtures {
		if actual := fixture.element.ToString(); actual != fixture.expected {
			t.Errorf("%s string = %q, want %q", fixture.name, actual, fixture.expected)
		}
	}
}
