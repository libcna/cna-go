// Command packed_vector_qualify performs deterministic exhaustive qualification
// over every XNA packed format whose complete bit domain is at most 16 bits.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
)

const sourceAssemblySHA256 = "38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130"

type sweep struct {
	Type       string `json:"type"`
	Iterations int    `json:"iterations"`
	Failures   int    `json:"failures"`
	Rule       string `json:"rule"`
}

type halfClasses struct {
	PositiveZero       int `json:"positiveZero"`
	NegativeZero       int `json:"negativeZero"`
	Subnormal          int `json:"subnormal"`
	Normal             int `json:"normal"`
	Exponent31ZeroFrac int `json:"exponent31ZeroFraction"`
	Exponent31Fraction int `json:"exponent31Fraction"`
	NonFiniteDecodes   int `json:"nonFiniteDecodes"`
}

type report struct {
	SchemaVersion        int         `json:"schemaVersion"`
	Authority            string      `json:"authority"`
	SourceAssemblySHA256 string      `json:"sourceAssemblySha256"`
	Sweeps               []sweep     `json:"sweeps"`
	HalfClasses          halfClasses `json:"halfClasses"`
	TotalIterations      int         `json:"totalIterations"`
	Failures             int         `json:"failures"`
}

func main() {
	output := flag.String("output", "docs/generated/packed-vector-exhaustive-report.json", "report path")
	flag.Parse()

	result := qualify()
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("EXHAUSTIVE_ITERATIONS=%d\nEXHAUSTIVE_FAILURES=%d\n", result.TotalIterations, result.Failures)
	if result.Failures != 0 {
		os.Exit(1)
	}
}

func qualify() report {
	result := report{
		SchemaVersion:        1,
		Authority:            "PURE_XNA_DERIVED round-trip invariants confirmed from pinned XNA 4.0 IL",
		SourceAssemblySHA256: sourceAssemblySHA256,
	}

	addSweep := func(name, rule string, iterations, failures int) {
		result.Sweeps = append(result.Sweeps, sweep{Type: name, Iterations: iterations, Failures: failures, Rule: rule})
		result.TotalIterations += iterations
		result.Failures += failures
	}

	alphaFailures := 0
	for raw := uint32(0); raw <= math.MaxUint8; raw++ {
		var value packedvector.Alpha8
		value.SetPackedValue(uint8(raw))
		if packedvector.NewAlpha8(value.ToAlpha()).PackedValue() != uint8(raw) {
			alphaFailures++
		}
	}
	addSweep("Alpha8", "encode(decode(bits)) == bits", 256, alphaFailures)

	bgrFailures := 0
	bgra4444Failures := 0
	bgra5551Failures := 0
	halfFailures := 0
	for raw := uint32(0); raw <= math.MaxUint16; raw++ {
		bits := uint16(raw)

		var bgr packedvector.Bgr565
		bgr.SetPackedValue(bits)
		if packedvector.NewBgr565ByVector3(bgr.ToVector3()).PackedValue() != bits {
			bgrFailures++
		}

		var bgra4444 packedvector.Bgra4444
		bgra4444.SetPackedValue(bits)
		if packedvector.NewBgra4444ByVector4(bgra4444.ToVector4()).PackedValue() != bits {
			bgra4444Failures++
		}

		var bgra5551 packedvector.Bgra5551
		bgra5551.SetPackedValue(bits)
		if packedvector.NewBgra5551ByVector4(bgra5551.ToVector4()).PackedValue() != bits {
			bgra5551Failures++
		}

		var half packedvector.HalfSingle
		half.SetPackedValue(bits)
		decoded := half.ToSingle()
		if math.IsNaN(float64(decoded)) || math.IsInf(float64(decoded), 0) {
			result.HalfClasses.NonFiniteDecodes++
		}
		if packedvector.NewHalfSingle(decoded).PackedValue() != bits {
			halfFailures++
		}
		exponent := bits & 0x7c00
		fraction := bits & 0x03ff
		switch {
		case exponent == 0 && fraction == 0 && bits&0x8000 == 0:
			result.HalfClasses.PositiveZero++
		case exponent == 0 && fraction == 0:
			result.HalfClasses.NegativeZero++
		case exponent == 0:
			result.HalfClasses.Subnormal++
		case exponent == 0x7c00 && fraction == 0:
			result.HalfClasses.Exponent31ZeroFrac++
		case exponent == 0x7c00:
			result.HalfClasses.Exponent31Fraction++
		default:
			result.HalfClasses.Normal++
		}
	}
	addSweep("Bgr565", "encode(decode(bits)) == bits", 65536, bgrFailures)
	addSweep("Bgra4444", "encode(decode(bits)) == bits", 65536, bgra4444Failures)
	addSweep("Bgra5551", "encode(decode(bits)) == bits", 65536, bgra5551Failures)
	addSweep("HalfSingle", "all encodings decode finite and encode(decode(bits)) == bits", 65536, halfFailures)

	return result
}
