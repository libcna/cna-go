package packedvector

import (
	"fmt"
	"math"
	"strconv"
)

func clampAndRound(value, minimum, maximum float32) float64 {
	switch {
	case math.IsNaN(float64(value)):
		return 0
	case math.IsInf(float64(value), 1):
		return float64(maximum)
	case math.IsInf(float64(value), -1):
		return float64(minimum)
	case value < minimum:
		return float64(minimum)
	case value > maximum:
		return float64(maximum)
	default:
		return math.RoundToEven(float64(value))
	}
}

func packUnsigned(bitmask, value float32) uint32 {
	return uint32(clampAndRound(value, 0, bitmask))
}

func packSigned(bitmask uint32, value float32) uint32 {
	maximum := float32(bitmask >> 1)
	minimum := -maximum - 1
	return uint32(int32(clampAndRound(value, minimum, maximum))) & bitmask
}

func packUNorm(bitmask, value float32) uint32 {
	scaled := value * bitmask
	return uint32(clampAndRound(scaled, 0, bitmask))
}

func unpackUNorm(bitmask, value uint32) float32 {
	return float32(value&bitmask) / float32(bitmask)
}

func packSNorm(bitmask uint32, value float32) uint32 {
	maximum := float32(bitmask >> 1)
	scaled := value * maximum
	return uint32(int32(clampAndRound(scaled, -maximum, maximum))) & bitmask
}

func unpackSNorm(bitmask, value uint32) float32 {
	signBit := (bitmask + 1) >> 1
	bits := value & bitmask
	if bits&signBit != 0 {
		if bits == signBit {
			return -1
		}
		return float32(int32(bits|^bitmask)) / float32(bitmask>>1)
	}
	return float32(bits) / float32(bitmask>>1)
}

// packHalf and unpackHalf reproduce XNA 4.0 HalfUtils. Its exponent-31
// encodings are extended finite values rather than IEEE half infinities/NaNs.
func packHalf(value float32) uint16 {
	word := math.Float32bits(value)
	sign := (word & 0x80000000) >> 16
	magnitude := word & 0x7fffffff
	if magnitude > 0x47ffefff {
		return uint16(sign | 0x7fff)
	}
	if magnitude < 0x38800000 {
		fraction := (magnitude & 0x007fffff) | 0x00800000
		shift := int32(113 - (magnitude >> 23))
		var reduced uint32
		if shift <= 31 {
			reduced = fraction >> (uint32(shift) & 31)
		}
		rounded := reduced + 0x0fff + ((reduced >> 13) & 1)
		return uint16(sign | (rounded >> 13))
	}
	biased := magnitude + uint32(0xc8000000)
	rounded := biased + 0x0fff + ((magnitude >> 13) & 1)
	return uint16(sign | (rounded >> 13))
}

func unpackHalf(value uint16) float32 {
	sign := uint32(value&0x8000) << 16
	exponent := uint32(value>>10) & 0x1f
	fraction := uint32(value & 0x03ff)
	var word uint32
	if exponent == 0 {
		if fraction == 0 {
			word = sign
		} else {
			exponentValue := int32(-14)
			for fraction&0x0400 == 0 {
				exponentValue--
				fraction <<= 1
			}
			fraction &= 0x03ff
			word = sign | uint32(exponentValue+127)<<23 | fraction<<13
		}
	} else {
		word = sign | uint32(int32(exponent)-15+127)<<23 | fraction<<13
	}
	return math.Float32frombits(word)
}

func hashUint64(value uint64) int32 {
	return int32(uint32(value)) ^ int32(uint32(value>>32))
}

func formatPacked(value uint64, digits int) string {
	return fmt.Sprintf("%0*X", digits, value)
}

func formatSingle(value float32) string {
	switch {
	case value == 0:
		return "0"
	case math.IsNaN(float64(value)):
		return "NaN"
	case math.IsInf(float64(value), 1):
		return "Infinity"
	case math.IsInf(float64(value), -1):
		return "-Infinity"
	default:
		return strconv.FormatFloat(float64(value), 'G', 7, 32)
	}
}
