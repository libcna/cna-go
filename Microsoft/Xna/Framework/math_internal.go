package framework

import (
	"fmt"
	"math"
	"strconv"
)

func sqrt32(value float32) float32 { return float32(math.Sqrt(float64(value))) }
func sin32(value float32) float32  { return float32(math.Sin(float64(value))) }
func cos32(value float32) float32  { return float32(math.Cos(float64(value))) }
func tan32(value float32) float32  { return float32(math.Tan(float64(value))) }
func acos32(value float32) float32 { return float32(math.Acos(float64(value))) }
func asin32(value float32) float32 { return float32(math.Asin(float64(value))) }
func atan232(y, x float32) float32 { return float32(math.Atan2(float64(y), float64(x))) }
func abs32(value float32) float32  { return math.Float32frombits(math.Float32bits(value) & 0x7fffffff) }
func copysign32(x, y float32) float32 {
	return math.Float32frombits((math.Float32bits(x) & 0x7fffffff) | (math.Float32bits(y) & 0x80000000))
}

// singleHashCode reproduces CLR Single.GetHashCode for the XNA-era runtime.
func singleHashCode(value float32) int32 {
	if value == 0 { // CLR canonicalizes both signed zeros.
		return 0
	}
	return int32(math.Float32bits(value))
}

// compareSingle reproduces System.Single.CompareTo, including its total-order
// treatment of NaN as preceding every non-NaN value and equal to another NaN.
func compareSingle(left, right float32) int32 {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	case left == right:
		return 0
	case math.IsNaN(float64(left)):
		if math.IsNaN(float64(right)) {
			return 0
		}
		return -1
	default:
		return 1
	}
}

// singleEquals reproduces System.Single::Equals(Single), which is what
// EqualityComparer<Single>.Default reaches: Single implements
// IEquatable<Single>, so the BCL selects GenericEqualityComparer<Single> and
// that comparer calls the strongly typed Equals.
//
// It is deliberately NOT Go's ==. The reference IL is
//
//	ldarg.1; ldarg.0; ldind.r4; bne.un.s L      // equal -> true
//	ldarg.1; call IsNaN; brfalse.s L2
//	ldarg.0; ldind.r4; call IsNaN; ret          // both NaN -> true
//
// so NaN equals NaN, where Go's == reports false. A collection search over
// System.Single therefore finds a NaN element, and CNA-Go reproduces that
// rather than inheriting Go's IEEE comparison. Signed zeros stay equal in both
// languages, so no special case is needed for them.
//
// This is the equality counterpart of compareSingle, which already records the
// same NaN treatment for System.Single::CompareTo.
func singleEquals(left, right float32) bool {
	if left == right {
		return true
	}
	return math.IsNaN(float64(left)) && math.IsNaN(float64(right))
}

func curveArgumentError(message string) error {
	return fmt.Errorf("curve argument: %s", message)
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

func validateTransformSlices[T any](source []T, sourceIndex int32, destination []T, destinationIndex, length int32) {
	if source == nil || destination == nil {
		panic("array is nil")
	}
	if int64(len(source)) < int64(sourceIndex)+int64(length) {
		panic("source array is too small")
	}
	if int64(len(destination)) < int64(destinationIndex)+int64(length) {
		panic("destination array is too small")
	}
}
