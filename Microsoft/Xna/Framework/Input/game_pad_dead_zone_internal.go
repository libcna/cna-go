package input

import (
	"math"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The reference dead-zone sizes, read from the internal XNA helper
// GamePadDeadZoneUtils. They are XInput's standard constants.
const (
	leftStickDeadZoneSize  = 0x1EA9 // 7849
	rightStickDeadZoneSize = 0x21F1 // 8689
	triggerDeadZoneSize    = 0x1E   // 30
)

// applyLinearDeadZone reproduces GamePadDeadZoneUtils.ApplyLinearDeadZone.
// A magnitude inside the dead zone collapses to zero, anything outside is
// shifted toward zero by the dead-zone size and then rescaled over the
// remaining range. Both reference comparisons are unordered, so a NaN input
// takes neither branch and returns zero.
func applyLinearDeadZone(value, maxValue, deadZoneSize float32) float32 {
	switch {
	case value < -deadZoneSize:
		value += deadZoneSize
	case value > deadZoneSize:
		value -= deadZoneSize
	default:
		return 0
	}
	return framework.MathHelperClamp(value/(maxValue-deadZoneSize), -1, 1)
}

// applyLeftStickDeadZone and applyRightStickDeadZone reproduce
// GamePadDeadZoneUtils.ApplyStickDeadZone in its IndependentAxes mode, which
// is the only mode reachable from the completed public surface:
// GamePadState.IsButtonDown hard-codes GamePadDeadZone.IndependentAxes. The
// Circular and None modes are reachable only through GamePad.GetState, which
// is not implemented, so they are deliberately absent rather than written as
// unreachable code.
func applyLeftStickDeadZone(x, y int32) framework.Vector2 {
	return framework.Vector2{
		X: applyLinearDeadZone(float32(x), 32767, leftStickDeadZoneSize),
		Y: applyLinearDeadZone(float32(y), 32767, leftStickDeadZoneSize),
	}
}

func applyRightStickDeadZone(x, y int32) framework.Vector2 {
	return framework.Vector2{
		X: applyLinearDeadZone(float32(x), 32767, rightStickDeadZoneSize),
		Y: applyLinearDeadZone(float32(y), 32767, rightStickDeadZoneSize),
	}
}

// applyTriggerDeadZone reproduces GamePadDeadZoneUtils.ApplyTriggerDeadZone
// for every mode other than None, which is again the only case reachable from
// the completed public surface.
func applyTriggerDeadZone(value int32) float32 {
	return applyLinearDeadZone(float32(value), 255, triggerDeadZoneSize)
}

// clrConvertToInt16 and clrConvertToUInt8 reproduce the CIL conv.i2 and
// conv.u1 conversions from Single that the reference uses when it packs its
// internal XInput snapshot: convert toward zero and keep the low bits.
//
// ECMA-335 leaves the conversion of NaN unspecified. On the x86 and x64
// runtimes the float-to-int32 conversion yields the integer indefinite value
// 0x80000000, whose low 16 and low 8 bits are both zero, so NaN maps to 0.
// Go leaves the same conversion undefined, so it is spelled out here rather
// than left to the compiler.
func clrConvertToInt16(value float32) int16 {
	if math.IsNaN(float64(value)) {
		return 0
	}
	return int16(int32(value))
}

func clrConvertToUInt8(value float32) uint8 {
	if math.IsNaN(float64(value)) {
		return 0
	}
	return uint8(int32(value))
}
