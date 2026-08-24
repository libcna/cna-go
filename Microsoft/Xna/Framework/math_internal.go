package framework

import (
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

func formatSingle(value float32) string {
	switch {
	case math.IsNaN(float64(value)):
		return "NaN"
	case math.IsInf(float64(value), 1):
		return "Infinity"
	case math.IsInf(float64(value), -1):
		return "-Infinity"
	default:
		return strconv.FormatFloat(float64(value), 'g', -1, 32)
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
