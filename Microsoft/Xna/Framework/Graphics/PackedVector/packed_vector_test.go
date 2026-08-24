package packedvector

import (
	"fmt"
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func requireEqual[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
}

func requireVectorBits(t *testing.T, got framework.Vector4, want [4]uint32) {
	t.Helper()
	gotBits := [4]uint32{
		math.Float32bits(got.X),
		math.Float32bits(got.Y),
		math.Float32bits(got.Z),
		math.Float32bits(got.W),
	}
	if gotBits != want {
		t.Fatalf("vector bits: got %08X, want %08X", gotBits, want)
	}
}

func vectorBits(x, y, z, w float32) [4]uint32 {
	return [4]uint32{
		math.Float32bits(x),
		math.Float32bits(y),
		math.Float32bits(z),
		math.Float32bits(w),
	}
}

func exerciseInterface[TPacked comparable](
	t *testing.T,
	name string,
	value IPackedVectorOfTPacked[TPacked],
	input framework.Vector4,
	wantPacked TPacked,
	wantVector [4]uint32,
) {
	t.Helper()
	value.PackFromVector4(input)
	requireEqual(t, name+" packed", value.PackedValue(), wantPacked)
	var base IPackedVector = value
	requireVectorBits(t, base.ToVector4(), wantVector)
}

func TestReferencePackedGoldens(t *testing.T) {
	t.Run("Alpha8", func(t *testing.T) {
		fixtures := []struct {
			value float32
			want  uint8
		}{
			{0, 0}, {0.5, 128}, {-1, 0}, {2, 255},
			{float32(0.5 / 255), 0}, {float32(2.5 / 255), 2},
			{float32(math.NaN()), 0}, {float32(math.Inf(1)), 255}, {float32(math.Inf(-1)), 0},
		}
		for _, fixture := range fixtures {
			requireEqual(t, "packed", NewAlpha8(fixture.value).PackedValue(), fixture.want)
		}
	})

	t.Run("color layouts", func(t *testing.T) {
		requireEqual(t, "Bgr565 ordinary", NewBgr565BySingleAndSingleAndSingle(0.25, 0.5, 0.75).PackedValue(), uint16(17431))
		requireEqual(t, "Bgr565 X", NewBgr565BySingleAndSingleAndSingle(1, 0, 0).PackedValue(), uint16(0xf800))
		requireEqual(t, "Bgr565 Y", NewBgr565BySingleAndSingleAndSingle(0, 1, 0).PackedValue(), uint16(0x07e0))
		requireEqual(t, "Bgr565 Z", NewBgr565BySingleAndSingleAndSingle(0, 0, 1).PackedValue(), uint16(0x001f))

		requireEqual(t, "Bgra4444 ordinary", NewBgra4444BySingleAndSingleAndSingleAndSingle(0.2, 0.4, 0.6, 0.8).PackedValue(), uint16(50025))
		requireEqual(t, "Bgra4444 X", NewBgra4444BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), uint16(0x0f00))
		requireEqual(t, "Bgra4444 Y", NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), uint16(0x00f0))
		requireEqual(t, "Bgra4444 Z", NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), uint16(0x000f))
		requireEqual(t, "Bgra4444 W", NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue(), uint16(0xf000))

		requireEqual(t, "Bgra5551 ordinary", NewBgra5551BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue(), uint16(8727))
		requireEqual(t, "Bgra5551 alpha below", NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 0)).PackedValue(), uint16(0))
		requireEqual(t, "Bgra5551 alpha tie", NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, 0.5).PackedValue(), uint16(0))
		requireEqual(t, "Bgra5551 alpha above", NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 1)).PackedValue(), uint16(0x8000))
	})

	t.Run("Byte4", func(t *testing.T) {
		requireEqual(t, "raw", NewByte4BySingleAndSingleAndSingleAndSingle(-1, 1, 128, 256).PackedValue(), uint32(4286578944))
		requireEqual(t, "ties", NewByte4BySingleAndSingleAndSingleAndSingle(0.5, 1.5, 2.5, 3.5).PackedValue(), uint32(67240448))
		requireEqual(t, "non-finite", NewByte4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 127.5).PackedValue(), uint32(0x8000ff00))
	})

	t.Run("half layouts", func(t *testing.T) {
		requireEqual(t, "single", NewHalfSingle(float32(1.0/3.0)).PackedValue(), uint16(13653))
		requireEqual(t, "vector2", NewHalfVector2BySingleAndSingle(1, -2).PackedValue(), uint32(3221240832))
		requireEqual(t, "vector4", NewHalfVector4BySingleAndSingleAndSingleAndSingle(1, -2, 0.5, math.Float32frombits(0x80000000)).PackedValue(), uint64(9223433612727172096))
		requireEqual(t, "vector4 X", NewHalfVector4BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), uint64(15360))
		requireEqual(t, "vector4 Y", NewHalfVector4BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), uint64(1006632960))
		requireEqual(t, "vector4 Z", NewHalfVector4BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), uint64(65970697666560))
		requireEqual(t, "vector4 W", NewHalfVector4BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue(), uint64(4323455642275676160))
	})

	t.Run("normalized", func(t *testing.T) {
		requireEqual(t, "byte2", NewNormalizedByte2BySingleAndSingle(-1, 0.5).PackedValue(), uint16(16513))
		requireEqual(t, "byte4 endpoints", NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue(), uint32(2139029633))
		requireEqual(t, "byte4 ties", NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(float32(0.5/127), float32(1.5/127), float32(-0.5/127), float32(-1.5/127)).PackedValue(), uint32(4261413376))
		requireEqual(t, "short2", NewNormalizedShort2BySingleAndSingle(-1, 0.5).PackedValue(), uint32(1073774593))
		requireEqual(t, "short4 endpoints", NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue(), uint64(9223231295071485953))
		requireEqual(t, "short4 ties", NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(float32(0.5/32767), float32(1.5/32767), float32(-0.5/32767), float32(-1.5/32767)).PackedValue(), uint64(18446181123756261376))
	})

	t.Run("RG and RGBA", func(t *testing.T) {
		requireEqual(t, "Rg32 ordinary", NewRg32BySingleAndSingle(0.25, 0.75).PackedValue(), uint32(3221176320))
		requireEqual(t, "Rg32 X", NewRg32BySingleAndSingle(1, 0).PackedValue(), uint32(65535))
		requireEqual(t, "Rg32 Y", NewRg32BySingleAndSingle(0, 1).PackedValue(), uint32(4294901760))
		requireEqual(t, "Rgba1010102 ordinary", NewRgba1010102BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue(), uint32(2952265984))
		requireEqual(t, "Rgba1010102 X", NewRgba1010102BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), uint32(1023))
		requireEqual(t, "Rgba1010102 Y", NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), uint32(1047552))
		requireEqual(t, "Rgba1010102 Z", NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), uint32(1072693248))
		requireEqual(t, "Rgba1010102 W", NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue(), uint32(3221225472))
		requireEqual(t, "Rgba64 ordinary", NewRgba64BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1).PackedValue(), uint64(18446673702817906688))
		requireEqual(t, "Rgba64 X", NewRgba64BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), uint64(65535))
		requireEqual(t, "Rgba64 Y", NewRgba64BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), uint64(4294901760))
		requireEqual(t, "Rgba64 Z", NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), uint64(281470681743360))
		requireEqual(t, "Rgba64 W", NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue(), uint64(18446462598732840960))
	})

	t.Run("Short", func(t *testing.T) {
		requireEqual(t, "Short2 endpoints", NewShort2BySingleAndSingle(-32768, 32767).PackedValue(), uint32(2147450880))
		requireEqual(t, "Short4 ordinary", NewShort4BySingleAndSingleAndSingleAndSingle(-32768, -1.5, 2.5, 32768).PackedValue(), uint64(9223090574762868736))
		requireEqual(t, "Short4 ties", NewShort4BySingleAndSingleAndSingleAndSingle(-0.5, -1.5, 0.5, 1.5).PackedValue(), uint64(562954248257536))
		requireEqual(t, "Short non-finite", NewShort4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 32768).PackedValue(), uint64(0x7fff80007fff0000))
	})
}

func TestRoundingAndNonFiniteByNumericalFamily(t *testing.T) {
	for _, fixture := range []struct {
		name string
		mask float32
	}{
		{"UNorm1", 1},
		{"UNorm2", 3},
		{"UNorm4", 15},
		{"UNorm5", 31},
		{"UNorm6", 63},
		{"UNorm8", 255},
		{"UNorm10", 1023},
		{"UNorm16", 65535},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			mask := uint32(fixture.mask)
			midpoint := float32(0.5 / fixture.mask)
			requireEqual(t, "below 0.5", packUNorm(fixture.mask, math.Nextafter32(midpoint, 0)), uint32(0))
			requireEqual(t, "tie at 0.5", packUNorm(fixture.mask, midpoint), uint32(0))
			requireEqual(t, "above 0.5", packUNorm(fixture.mask, math.Nextafter32(midpoint, 1)), uint32(1))
			wantHighTie := uint32(2)
			if mask < wantHighTie {
				wantHighTie = mask
			}
			requireEqual(t, "tie at 2.5", packUNorm(fixture.mask, float32(2.5/fixture.mask)), wantHighTie)
			requireEqual(t, "NaN", packUNorm(fixture.mask, float32(math.NaN())), uint32(0))
			requireEqual(t, "+Inf", packUNorm(fixture.mask, float32(math.Inf(1))), mask)
			requireEqual(t, "-Inf", packUNorm(fixture.mask, float32(math.Inf(-1))), uint32(0))
		})
	}

	requireEqual(t, "unsigned NaN", packUnsigned(255, float32(math.NaN())), uint32(0))
	requireEqual(t, "unsigned +Inf", packUnsigned(255, float32(math.Inf(1))), uint32(255))
	requireEqual(t, "unsigned -Inf", packUnsigned(255, float32(math.Inf(-1))), uint32(0))
	requireEqual(t, "signed NaN", packSigned(0xffff, float32(math.NaN())), uint32(0))
	requireEqual(t, "signed +Inf", packSigned(0xffff, float32(math.Inf(1))), uint32(0x7fff))
	requireEqual(t, "signed -Inf", packSigned(0xffff, float32(math.Inf(-1))), uint32(0x8000))
	requireEqual(t, "SNorm8 NaN", packSNorm(0xff, float32(math.NaN())), uint32(0))
	requireEqual(t, "SNorm8 +Inf", packSNorm(0xff, float32(math.Inf(1))), uint32(0x7f))
	requireEqual(t, "SNorm8 -Inf", packSNorm(0xff, float32(math.Inf(-1))), uint32(0x81))
	requireEqual(t, "SNorm16 +Inf", packSNorm(0xffff, float32(math.Inf(1))), uint32(0x7fff))
	requireEqual(t, "SNorm16 -Inf", packSNorm(0xffff, float32(math.Inf(-1))), uint32(0x8001))

	requireEqual(t, "unsigned below tie", packUnsigned(255, math.Nextafter32(0.5, 0)), uint32(0))
	requireEqual(t, "unsigned tie", packUnsigned(255, 0.5), uint32(0))
	requireEqual(t, "unsigned above tie", packUnsigned(255, math.Nextafter32(0.5, 1)), uint32(1))
	requireEqual(t, "signed below tie", packSigned(0xffff, math.Nextafter32(0.5, 0)), uint32(0))
	requireEqual(t, "signed tie", packSigned(0xffff, 0.5), uint32(0))
	requireEqual(t, "signed above tie", packSigned(0xffff, math.Nextafter32(0.5, 1)), uint32(1))

	snorm8Midpoint := float32(0.5 / 127)
	requireEqual(t, "SNorm8 below tie", packSNorm(0xff, math.Nextafter32(snorm8Midpoint, 0)), uint32(0))
	requireEqual(t, "SNorm8 tie", packSNorm(0xff, snorm8Midpoint), uint32(0))
	requireEqual(t, "SNorm8 above tie", packSNorm(0xff, math.Nextafter32(snorm8Midpoint, 1)), uint32(1))
	snorm16Midpoint := float32(0.5 / 32767)
	requireEqual(t, "SNorm16 below tie", packSNorm(0xffff, math.Nextafter32(snorm16Midpoint, 0)), uint32(0))
	requireEqual(t, "SNorm16 tie", packSNorm(0xffff, snorm16Midpoint), uint32(0))
	requireEqual(t, "SNorm16 above tie", packSNorm(0xffff, math.Nextafter32(snorm16Midpoint, 1)), uint32(1))

	requireEqual(t, "half below tie", packHalf(math.Float32frombits(0x3f800fff)), uint16(0x3c00))
	requireEqual(t, "half exact tie", packHalf(math.Float32frombits(0x3f801000)), uint16(0x3c00))
	requireEqual(t, "half above tie", packHalf(math.Float32frombits(0x3f801001)), uint16(0x3c01))
}

func TestVectorConstructorsMatchScalarConstructors(t *testing.T) {
	requireEqual(t, "Bgr565", NewBgr565ByVector3(framework.Vector3{X: 0.25, Y: 0.5, Z: 0.75}), NewBgr565BySingleAndSingleAndSingle(0.25, 0.5, 0.75))
	vector4 := framework.Vector4{X: -0.25, Y: 0.5, Z: 1.25, W: 0.75}
	vector2 := framework.Vector2{X: -0.25, Y: 0.5}
	requireEqual(t, "Bgra4444", NewBgra4444ByVector4(vector4), NewBgra4444BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "Bgra5551", NewBgra5551ByVector4(vector4), NewBgra5551BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "Byte4", NewByte4ByVector4(vector4), NewByte4BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "HalfVector2", NewHalfVector2ByVector2(vector2), NewHalfVector2BySingleAndSingle(vector2.X, vector2.Y))
	requireEqual(t, "HalfVector4", NewHalfVector4ByVector4(vector4), NewHalfVector4BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "NormalizedByte2", NewNormalizedByte2ByVector2(vector2), NewNormalizedByte2BySingleAndSingle(vector2.X, vector2.Y))
	requireEqual(t, "NormalizedByte4", NewNormalizedByte4ByVector4(vector4), NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "NormalizedShort2", NewNormalizedShort2ByVector2(vector2), NewNormalizedShort2BySingleAndSingle(vector2.X, vector2.Y))
	requireEqual(t, "NormalizedShort4", NewNormalizedShort4ByVector4(vector4), NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "Rg32", NewRg32ByVector2(vector2), NewRg32BySingleAndSingle(vector2.X, vector2.Y))
	requireEqual(t, "Rgba1010102", NewRgba1010102ByVector4(vector4), NewRgba1010102BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "Rgba64", NewRgba64ByVector4(vector4), NewRgba64BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
	requireEqual(t, "Short2", NewShort2ByVector2(vector2), NewShort2BySingleAndSingle(vector2.X, vector2.Y))
	requireEqual(t, "Short4", NewShort4ByVector4(vector4), NewShort4BySingleAndSingleAndSingleAndSingle(vector4.X, vector4.Y, vector4.Z, vector4.W))
}

func TestPackedInterfacesConsumeAndFillExactLanes(t *testing.T) {
	input := framework.Vector4{X: -2, Y: 0.5, Z: 2, W: 0.25}
	exerciseInterface(t, "Alpha8", &Alpha8{}, input, uint8(64), vectorBits(0, 0, 0, float32(64)/255))
	exerciseInterface(t, "Bgr565", &Bgr565{}, input, uint16(1055), vectorBits(0, float32(32)/63, 1, 1))
	exerciseInterface(t, "Bgra4444", &Bgra4444{}, input, uint16(0x408f), vectorBits(0, float32(8)/15, 1, float32(4)/15))
	exerciseInterface(t, "Bgra5551", &Bgra5551{}, input, uint16(0x021f), vectorBits(0, float32(16)/31, 1, 0))
	exerciseInterface(t, "Byte4", &Byte4{}, input, uint32(0x00020000), vectorBits(0, 0, 2, 0))
	exerciseInterface(t, "HalfSingle", &HalfSingle{}, input, uint16(0xc000), vectorBits(-2, 0, 0, 1))
	exerciseInterface(t, "HalfVector2", &HalfVector2{}, input, uint32(0x3800c000), vectorBits(-2, 0.5, 0, 1))
	exerciseInterface(t, "HalfVector4", &HalfVector4{}, input, uint64(0x340040003800c000), vectorBits(-2, 0.5, 2, 0.25))
	exerciseInterface(t, "NormalizedByte2", &NormalizedByte2{}, input, uint16(0x4081), vectorBits(-1, float32(64)/127, 0, 1))
	exerciseInterface(t, "NormalizedByte4", &NormalizedByte4{}, input, uint32(0x207f4081), vectorBits(-1, float32(64)/127, 1, float32(32)/127))
	exerciseInterface(t, "NormalizedShort2", &NormalizedShort2{}, input, uint32(0x40008001), vectorBits(-1, float32(16384)/32767, 0, 1))
	exerciseInterface(t, "NormalizedShort4", &NormalizedShort4{}, input, uint64(0x20007fff40008001), vectorBits(-1, float32(16384)/32767, 1, float32(8192)/32767))
	exerciseInterface(t, "Rg32", &Rg32{}, input, uint32(0x80000000), vectorBits(0, float32(32768)/65535, 0, 1))
	exerciseInterface(t, "Rgba1010102", &Rgba1010102{}, input, uint32(0x7ff80000), vectorBits(0, float32(512)/1023, 1, float32(1)/3))
	exerciseInterface(t, "Rgba64", &Rgba64{}, input, uint64(0x4000ffff80000000), vectorBits(0, float32(32768)/65535, 1, float32(16384)/65535))
	exerciseInterface(t, "Short2", &Short2{}, input, uint32(0x0000fffe), vectorBits(-2, 0, 0, 1))
	exerciseInterface(t, "Short4", &Short4{}, input, uint64(0x000000020000fffe), vectorBits(-2, 0, 2, 0))
}

func TestHalfReferenceSpecialValues(t *testing.T) {
	fixtures := []struct {
		name        string
		inputBits   uint32
		packed      uint16
		decodedBits uint32
		text        string
	}{
		{"positive zero", 0x00000000, 0x0000, 0x00000000, "0"},
		{"negative zero", 0x80000000, 0x8000, 0x80000000, "0"},
		{"smallest subnormal", 0x33800000, 0x0001, 0x33800000, "5.960464E-08"},
		{"largest subnormal", 0x387fc000, 0x03ff, 0x387fc000, "6.097555E-05"},
		{"smallest normal", 0x38800000, 0x0400, 0x38800000, "6.103516E-05"},
		{"tie even low", 0x3f801000, 0x3c00, 0x3f800000, "1"},
		{"tie even high", 0x3f803000, 0x3c02, 0x3f804000, "1.001953"},
		{"maximum conventional finite", 0x477fe000, 0x7bff, 0x477fe000, "65504"},
		{"exponent 31 boundary", 0x477ff000, 0x7c00, 0x47800000, "65536"},
		{"positive infinity saturates", 0x7f800000, 0x7fff, 0x47ffe000, "131008"},
		{"negative infinity saturates", 0xff800000, 0xffff, 0xc7ffe000, "-131008"},
		{"positive NaN saturates", 0x7fc12345, 0x7fff, 0x47ffe000, "131008"},
		{"negative NaN saturates", 0xffc12345, 0xffff, 0xc7ffe000, "-131008"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			value := NewHalfSingle(math.Float32frombits(fixture.inputBits))
			requireEqual(t, "packed", value.PackedValue(), fixture.packed)
			requireEqual(t, "decoded", math.Float32bits(value.ToSingle()), fixture.decodedBits)
			requireEqual(t, "text", value.ToString(), fixture.text)
		})
	}
}

func TestHalfAllBitPatternsRoundTrip(t *testing.T) {
	classCounts := map[string]int{}
	for raw := uint32(0); raw <= math.MaxUint16; raw++ {
		bits := uint16(raw)
		var value HalfSingle
		value.SetPackedValue(bits)
		decoded := value.ToSingle()
		if math.IsNaN(float64(decoded)) || math.IsInf(float64(decoded), 0) {
			t.Fatalf("XNA half %04X decoded non-finite", bits)
		}
		if got := NewHalfSingle(decoded).PackedValue(); got != bits {
			t.Fatalf("half round trip %04X -> %08X -> %04X", bits, math.Float32bits(decoded), got)
		}
		exponent := bits & 0x7c00
		fraction := bits & 0x03ff
		switch {
		case exponent == 0 && fraction == 0:
			classCounts["zero"]++
		case exponent == 0:
			classCounts["subnormal"]++
		case exponent == 0x7c00 && fraction == 0:
			classCounts["exponent31_zero_fraction"]++
		case exponent == 0x7c00:
			classCounts["exponent31_fraction"]++
		default:
			classCounts["normal"]++
		}
	}
	want := map[string]int{
		"zero":                     2,
		"subnormal":                2046,
		"normal":                   61440,
		"exponent31_zero_fraction": 2,
		"exponent31_fraction":      2046,
	}
	for class, count := range want {
		requireEqual(t, class, classCounts[class], count)
	}
}

func TestExhaustiveSmallPackedRoundTrips(t *testing.T) {
	for raw := uint32(0); raw <= math.MaxUint8; raw++ {
		var value Alpha8
		value.SetPackedValue(uint8(raw))
		if got := NewAlpha8(value.ToAlpha()).PackedValue(); got != uint8(raw) {
			t.Fatalf("Alpha8 round trip %02X -> %02X", raw, got)
		}
	}

	for raw := uint32(0); raw <= math.MaxUint16; raw++ {
		bits := uint16(raw)
		var bgr Bgr565
		bgr.SetPackedValue(bits)
		v3 := bgr.ToVector3()
		if got := NewBgr565ByVector3(v3).PackedValue(); got != bits {
			t.Fatalf("Bgr565 round trip %04X -> %04X", bits, got)
		}

		var bgra4 Bgra4444
		bgra4.SetPackedValue(bits)
		if got := NewBgra4444ByVector4(bgra4.ToVector4()).PackedValue(); got != bits {
			t.Fatalf("Bgra4444 round trip %04X -> %04X", bits, got)
		}

		var bgra5 Bgra5551
		bgra5.SetPackedValue(bits)
		if got := NewBgra5551ByVector4(bgra5.ToVector4()).PackedValue(); got != bits {
			t.Fatalf("Bgra5551 round trip %04X -> %04X", bits, got)
		}
	}
}

func checkCopy[T any, TPacked comparable](
	t *testing.T,
	name string,
	zero T,
	mutate func(*T),
	packed func(T) TPacked,
	want TPacked,
) {
	t.Helper()
	copyValue := zero
	mutate(&copyValue)
	var zeroPacked TPacked
	requireEqual(t, name+" original", packed(zero), zeroPacked)
	requireEqual(t, name+" copy", packed(copyValue), want)
}

func TestPackedValueSettersAreBitTransparentAndCopiesAreIndependent(t *testing.T) {
	checkCopy(t, "Alpha8", Alpha8{}, func(v *Alpha8) { v.SetPackedValue(0xa5) }, func(v Alpha8) uint8 { return v.PackedValue() }, uint8(0xa5))
	checkCopy(t, "Bgr565", Bgr565{}, func(v *Bgr565) { v.SetPackedValue(0xa55a) }, func(v Bgr565) uint16 { return v.PackedValue() }, uint16(0xa55a))
	checkCopy(t, "Bgra4444", Bgra4444{}, func(v *Bgra4444) { v.SetPackedValue(0xf00f) }, func(v Bgra4444) uint16 { return v.PackedValue() }, uint16(0xf00f))
	checkCopy(t, "Bgra5551", Bgra5551{}, func(v *Bgra5551) { v.SetPackedValue(0x8001) }, func(v Bgra5551) uint16 { return v.PackedValue() }, uint16(0x8001))
	checkCopy(t, "Byte4", Byte4{}, func(v *Byte4) { v.SetPackedValue(0x80ff00a5) }, func(v Byte4) uint32 { return v.PackedValue() }, uint32(0x80ff00a5))
	checkCopy(t, "HalfSingle", HalfSingle{}, func(v *HalfSingle) { v.SetPackedValue(0x7e35) }, func(v HalfSingle) uint16 { return v.PackedValue() }, uint16(0x7e35))
	checkCopy(t, "HalfVector2", HalfVector2{}, func(v *HalfVector2) { v.SetPackedValue(0x7e358001) }, func(v HalfVector2) uint32 { return v.PackedValue() }, uint32(0x7e358001))
	checkCopy(t, "HalfVector4", HalfVector4{}, func(v *HalfVector4) { v.SetPackedValue(0xff00123480017e35) }, func(v HalfVector4) uint64 { return v.PackedValue() }, uint64(0xff00123480017e35))
	checkCopy(t, "NormalizedByte2", NormalizedByte2{}, func(v *NormalizedByte2) { v.SetPackedValue(0x8081) }, func(v NormalizedByte2) uint16 { return v.PackedValue() }, uint16(0x8081))
	checkCopy(t, "NormalizedByte4", NormalizedByte4{}, func(v *NormalizedByte4) { v.SetPackedValue(0xff80817f) }, func(v NormalizedByte4) uint32 { return v.PackedValue() }, uint32(0xff80817f))
	checkCopy(t, "NormalizedShort2", NormalizedShort2{}, func(v *NormalizedShort2) { v.SetPackedValue(0x80008001) }, func(v NormalizedShort2) uint32 { return v.PackedValue() }, uint32(0x80008001))
	checkCopy(t, "NormalizedShort4", NormalizedShort4{}, func(v *NormalizedShort4) { v.SetPackedValue(0xffff800080018000) }, func(v NormalizedShort4) uint64 { return v.PackedValue() }, uint64(0xffff800080018000))
	checkCopy(t, "Rg32", Rg32{}, func(v *Rg32) { v.SetPackedValue(0xffff0001) }, func(v Rg32) uint32 { return v.PackedValue() }, uint32(0xffff0001))
	checkCopy(t, "Rgba1010102", Rgba1010102{}, func(v *Rgba1010102) { v.SetPackedValue(0xc0100801) }, func(v Rgba1010102) uint32 { return v.PackedValue() }, uint32(0xc0100801))
	checkCopy(t, "Rgba64", Rgba64{}, func(v *Rgba64) { v.SetPackedValue(0xffff800000010000) }, func(v Rgba64) uint64 { return v.PackedValue() }, uint64(0xffff800000010000))
	checkCopy(t, "Short2", Short2{}, func(v *Short2) { v.SetPackedValue(0x7fff8000) }, func(v Short2) uint32 { return v.PackedValue() }, uint32(0x7fff8000))
	checkCopy(t, "Short4", Short4{}, func(v *Short4) { v.SetPackedValue(0xffff800000017fff) }, func(v Short4) uint64 { return v.PackedValue() }, uint64(0xffff800000017fff))
}

func TestPackedEqualityHashStringAndOperators(t *testing.T) {
	var alpha Alpha8
	alpha.SetPackedValue(0xa5)
	requireEqual(t, "Alpha8 string", alpha.ToString(), "A5")
	requireEqual(t, "Alpha8 hash", alpha.GetHashCode(), int32(0xa5))
	requireEqual(t, "Alpha8 object equality", alpha.EqualsByObject(alpha), true)
	requireEqual(t, "Alpha8 object type", alpha.EqualsByObject(uint8(0xa5)), false)
	requireEqual(t, "Alpha8 equality operator", Alpha8OperatorEqualityByAlpha8AndAlpha8(alpha, alpha), true)
	requireEqual(t, "Alpha8 inequality operator", Alpha8OperatorInequalityByAlpha8AndAlpha8(alpha, Alpha8{}), true)

	var byte4 Byte4
	byte4.SetPackedValue(0x89abcdef)
	requireEqual(t, "Byte4 string", byte4.ToString(), "89ABCDEF")
	requireEqual(t, "Byte4 hash", byte4.GetHashCode(), int32(-1985229329))
	requireEqual(t, "Byte4 packed equality", byte4.EqualsByByte4(byte4), true)

	var half HalfSingle
	half.SetPackedValue(0x3555)
	requireEqual(t, "HalfSingle string", half.ToString(), "0.333252")
	var halfA, halfB HalfSingle
	halfA.SetPackedValue(0x7e00)
	halfB.SetPackedValue(0x7e01)
	requireEqual(t, "Half bit equality", halfA.EqualsByHalfSingle(halfB), false)
	requireEqual(t, "Half operator inequality", HalfSingleOperatorInequalityByHalfSingleAndHalfSingle(halfA, halfB), true)

	var value64 HalfVector4
	value64.SetPackedValue(0xfedcba9876543210)
	requireEqual(t, "64-bit hash", value64.GetHashCode(), int32(-2004318072))
	requireEqual(t, "HalfVector4 string", value64.ToString(), "{X:0.1894531 Y:25920 Z:-0.8242188 W:-112384}")
	requireEqual(t, "HalfVector4 object equality", value64.EqualsByObject(value64), true)
	requireEqual(t, "HalfVector4 operator equality", HalfVector4OperatorEqualityByHalfVector4AndHalfVector4(value64, value64), true)
	requireEqual(t, "HalfVector4 operator inequality", HalfVector4OperatorInequalityByHalfVector4AndHalfVector4(value64, HalfVector4{}), true)

	var rgba Rgba64
	rgba.SetPackedValue(0x0123456789abcdef)
	requireEqual(t, "Rgba64 string", rgba.ToString(), "0123456789ABCDEF")
	requireEqual(t, "Rgba64 hash", rgba.GetHashCode(), int32(-2004318072))
	requireEqual(t, "Rgba64 operator equality", rgba.EqualsByObject(rgba) && rgba.EqualsByRgba64(rgba) && Rgba64OperatorEqualityByRgba64AndRgba64(rgba, rgba), true)
	requireEqual(t, "Rgba64 operator inequality", Rgba64OperatorInequalityByRgba64AndRgba64(rgba, Rgba64{}), true)

	var bgr Bgr565
	bgr.SetPackedValue(0xa55a)
	requireEqual(t, "Bgr565 string", bgr.ToString(), "A55A")
	requireEqual(t, "Bgr565 hash", bgr.GetHashCode(), int32(42330))
	requireEqual(t, "Bgr565 equality", bgr.EqualsByObject(bgr) && bgr.EqualsByBgr565(bgr) && Bgr565OperatorEqualityByBgr565AndBgr565(bgr, bgr), true)
	requireEqual(t, "Bgr565 inequality", Bgr565OperatorInequalityByBgr565AndBgr565(bgr, Bgr565{}), true)

	var bgra4444 Bgra4444
	bgra4444.SetPackedValue(0xa55a)
	requireEqual(t, "Bgra4444 string/hash", bgra4444.ToString()+fmt.Sprint(bgra4444.GetHashCode()), "A55A42330")
	requireEqual(t, "Bgra4444 equality", bgra4444.EqualsByObject(bgra4444) && bgra4444.EqualsByBgra4444(bgra4444) && Bgra4444OperatorEqualityByBgra4444AndBgra4444(bgra4444, bgra4444) && Bgra4444OperatorInequalityByBgra4444AndBgra4444(bgra4444, Bgra4444{}), true)

	var bgra5551 Bgra5551
	bgra5551.SetPackedValue(0xa55a)
	requireEqual(t, "Bgra5551 string/hash", bgra5551.ToString()+fmt.Sprint(bgra5551.GetHashCode()), "A55A42330")
	requireEqual(t, "Bgra5551 equality", bgra5551.EqualsByObject(bgra5551) && bgra5551.EqualsByBgra5551(bgra5551) && Bgra5551OperatorEqualityByBgra5551AndBgra5551(bgra5551, bgra5551) && Bgra5551OperatorInequalityByBgra5551AndBgra5551(bgra5551, Bgra5551{}), true)

	requireEqual(t, "Byte4 operators", Byte4OperatorEqualityByByte4AndByte4(byte4, byte4) && Byte4OperatorInequalityByByte4AndByte4(byte4, Byte4{}) && byte4.EqualsByObject(byte4), true)

	var halfVector2 HalfVector2
	halfVector2.SetPackedValue(0xc0003c00)
	requireEqual(t, "HalfVector2 string", halfVector2.ToString(), "{X:1 Y:-2}")
	requireEqual(t, "HalfVector2 hash", halfVector2.GetHashCode(), int32(-1073726464))
	requireEqual(t, "HalfVector2 equality", halfVector2.EqualsByObject(halfVector2) && halfVector2.EqualsByHalfVector2(halfVector2) && HalfVector2OperatorEqualityByHalfVector2AndHalfVector2(halfVector2, halfVector2) && HalfVector2OperatorInequalityByHalfVector2AndHalfVector2(halfVector2, HalfVector2{}), true)

	var normalizedByte2 NormalizedByte2
	normalizedByte2.SetPackedValue(0xa55a)
	requireEqual(t, "NormalizedByte2 string/hash", normalizedByte2.ToString()+fmt.Sprint(normalizedByte2.GetHashCode()), "A55A42330")
	requireEqual(t, "NormalizedByte2 equality", normalizedByte2.EqualsByObject(normalizedByte2) && normalizedByte2.EqualsByNormalizedByte2(normalizedByte2) && NormalizedByte2OperatorEqualityByNormalizedByte2AndNormalizedByte2(normalizedByte2, normalizedByte2) && NormalizedByte2OperatorInequalityByNormalizedByte2AndNormalizedByte2(normalizedByte2, NormalizedByte2{}), true)

	var normalizedByte4 NormalizedByte4
	normalizedByte4.SetPackedValue(0x89abcdef)
	requireEqual(t, "NormalizedByte4 string/hash", normalizedByte4.ToString()+fmt.Sprint(normalizedByte4.GetHashCode()), "89ABCDEF-1985229329")
	requireEqual(t, "NormalizedByte4 equality", normalizedByte4.EqualsByObject(normalizedByte4) && normalizedByte4.EqualsByNormalizedByte4(normalizedByte4) && NormalizedByte4OperatorEqualityByNormalizedByte4AndNormalizedByte4(normalizedByte4, normalizedByte4) && NormalizedByte4OperatorInequalityByNormalizedByte4AndNormalizedByte4(normalizedByte4, NormalizedByte4{}), true)

	var normalizedShort2 NormalizedShort2
	normalizedShort2.SetPackedValue(0x89abcdef)
	requireEqual(t, "NormalizedShort2 string/hash", normalizedShort2.ToString()+fmt.Sprint(normalizedShort2.GetHashCode()), "89ABCDEF-1985229329")
	requireEqual(t, "NormalizedShort2 equality", normalizedShort2.EqualsByObject(normalizedShort2) && normalizedShort2.EqualsByNormalizedShort2(normalizedShort2) && NormalizedShort2OperatorEqualityByNormalizedShort2AndNormalizedShort2(normalizedShort2, normalizedShort2) && NormalizedShort2OperatorInequalityByNormalizedShort2AndNormalizedShort2(normalizedShort2, NormalizedShort2{}), true)

	var normalizedShort4 NormalizedShort4
	normalizedShort4.SetPackedValue(0x0123456789abcdef)
	requireEqual(t, "NormalizedShort4 string/hash", normalizedShort4.ToString()+fmt.Sprint(normalizedShort4.GetHashCode()), "0123456789ABCDEF-2004318072")
	requireEqual(t, "NormalizedShort4 equality", normalizedShort4.EqualsByObject(normalizedShort4) && normalizedShort4.EqualsByNormalizedShort4(normalizedShort4) && NormalizedShort4OperatorEqualityByNormalizedShort4AndNormalizedShort4(normalizedShort4, normalizedShort4) && NormalizedShort4OperatorInequalityByNormalizedShort4AndNormalizedShort4(normalizedShort4, NormalizedShort4{}), true)

	var rg Rg32
	rg.SetPackedValue(0x89abcdef)
	requireEqual(t, "Rg32 string/hash", rg.ToString()+fmt.Sprint(rg.GetHashCode()), "89ABCDEF-1985229329")
	requireEqual(t, "Rg32 equality", rg.EqualsByObject(rg) && rg.EqualsByRg32(rg) && Rg32OperatorEqualityByRg32AndRg32(rg, rg) && Rg32OperatorInequalityByRg32AndRg32(rg, Rg32{}), true)

	var rgba1010102 Rgba1010102
	rgba1010102.SetPackedValue(0x89abcdef)
	requireEqual(t, "Rgba1010102 string/hash", rgba1010102.ToString()+fmt.Sprint(rgba1010102.GetHashCode()), "89ABCDEF-1985229329")
	requireEqual(t, "Rgba1010102 equality", rgba1010102.EqualsByObject(rgba1010102) && rgba1010102.EqualsByRgba1010102(rgba1010102) && Rgba1010102OperatorEqualityByRgba1010102AndRgba1010102(rgba1010102, rgba1010102) && Rgba1010102OperatorInequalityByRgba1010102AndRgba1010102(rgba1010102, Rgba1010102{}), true)

	var short2 Short2
	short2.SetPackedValue(0x89abcdef)
	requireEqual(t, "Short2 string/hash", short2.ToString()+fmt.Sprint(short2.GetHashCode()), "89ABCDEF-1985229329")
	requireEqual(t, "Short2 equality", short2.EqualsByObject(short2) && short2.EqualsByShort2(short2) && Short2OperatorEqualityByShort2AndShort2(short2, short2) && Short2OperatorInequalityByShort2AndShort2(short2, Short2{}), true)

	var short4 Short4
	short4.SetPackedValue(0x0123456789abcdef)
	requireEqual(t, "Short4 string/hash", short4.ToString()+fmt.Sprint(short4.GetHashCode()), "0123456789ABCDEF-2004318072")
	requireEqual(t, "Short4 equality", short4.EqualsByObject(short4) && short4.EqualsByShort4(short4) && Short4OperatorEqualityByShort4AndShort4(short4, short4) && Short4OperatorInequalityByShort4AndShort4(short4, Short4{}), true)
}
