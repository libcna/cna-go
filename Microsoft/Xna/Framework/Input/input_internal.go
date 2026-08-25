package input

import (
	"math"
	"strconv"
)

// smartGetHashCode reproduces the XNA Framework helper
// `Microsoft.Xna.Framework.Helpers.SmartGetHashCode`. The helper pins the
// boxed value, reads every complete 32-bit word of its marshalled layout,
// XORs them, and substitutes Int32.MaxValue when the XOR is zero. The zero
// substitution is intentional and creates compatible collisions; it must not
// be replaced with a better-distributed combine function.
//
// XOR is commutative, so the declared field order of a sequential layout does
// not affect the result — only the set of 32-bit words does.
func smartGetHashCode(words ...int32) int32 {
	var hash int32
	for _, word := range words {
		hash ^= word
	}
	if hash == 0 {
		return math.MaxInt32
	}
	return hash
}

// singleWord reinterprets a Single as the 32-bit word SmartGetHashCode reads
// out of the pinned struct. Unlike System.Single.GetHashCode it does not
// canonicalize signed zero, because the helper reads raw storage.
func singleWord(value float32) int32 { return int32(math.Float32bits(value)) }

// mathMinSingle reproduces System.Math.Min(Single, Single), including its NaN
// propagation, which differs from the XNA Vector2.Min component rule.
func mathMinSingle(value1, value2 float32) float32 {
	if value1 < value2 {
		return value1
	}
	if math.IsNaN(float64(value1)) {
		return value1
	}
	return value2
}

// mathMaxSingle reproduces System.Math.Max(Single, Single). Max(-0, 0)
// returns positive zero because the comparison is false and the second
// operand is returned unchanged.
func mathMaxSingle(value1, value2 float32) float32 {
	if value1 > value2 {
		return value1
	}
	if math.IsNaN(float64(value1)) {
		return value1
	}
	return value2
}

// formatSingle reproduces the default CLR Single formatting the reference
// ToString implementations observe. It matches the framework package helper of
// the same name; the two are kept identical by
// TestInputFormatSingleMatchesFrameworkFormatting.
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

// appendPressedName reproduces the reference button-name accumulator: a
// pressed button appends its name, separated from any earlier name by exactly
// one space. The reference compares against Pressed exactly, so an arbitrary
// raw ButtonState that is neither Released nor Pressed contributes nothing.
func appendPressedName(accumulated string, state ButtonState, name string) string {
	if state != ButtonStatePressed {
		return accumulated
	}
	if accumulated == "" {
		return name
	}
	return accumulated + " " + name
}

// pressedNamesOrNone reproduces the reference "None" substitution applied when
// no button contributed a name.
func pressedNamesOrNone(accumulated string) string {
	if accumulated == "" {
		return "None"
	}
	return accumulated
}

// buttonStateFromMask reproduces the reference
// `(buttons & mask) == mask ? Pressed : Released` derivation.
func buttonStateFromMask(buttons Buttons, mask Buttons) ButtonState {
	if buttons&mask == mask {
		return ButtonStatePressed
	}
	return ButtonStateReleased
}
