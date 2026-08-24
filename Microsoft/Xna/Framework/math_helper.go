package framework

import "math"

// MathHelper is the XNA static math-helper type identity. Its members map to
// type-prefixed package declarations because Go has no static methods.
type MathHelper struct{}

const (
	MathHelperE       float32 = 2.71828175
	MathHelperLog2E   float32 = 1.442695
	MathHelperLog10E  float32 = 0.434294492
	MathHelperPi      float32 = 3.14159274
	MathHelperTwoPi   float32 = 6.28318548
	MathHelperPiOver2 float32 = 1.57079637
	MathHelperPiOver4 float32 = 0.785398185
)

func MathHelperToRadians(degrees float32) float32 {
	return degrees * 0.0174532924
}

func MathHelperToDegrees(radians float32) float32 {
	return radians * 57.29578
}

func MathHelperDistance(value1, value2 float32) float32 {
	return float32(math.Abs(float64(value1 - value2)))
}

func MathHelperMin(value1, value2 float32) float32 {
	if math.IsNaN(float64(value1)) || value1 < value2 {
		return value1
	}
	return value2
}

func MathHelperMax(value1, value2 float32) float32 {
	if math.IsNaN(float64(value1)) || value1 > value2 {
		return value1
	}
	return value2
}

func MathHelperClamp(value, min, max float32) float32 {
	if value > max {
		return max
	}
	if value < min {
		return min
	}
	return value
}

func MathHelperLerp(value1, value2, amount float32) float32 {
	return value1 + (value2-value1)*amount
}

func MathHelperBarycentric(value1, value2, value3, amount1, amount2 float32) float32 {
	return value1 + amount1*(value2-value1) + amount2*(value3-value1)
}

func MathHelperSmoothStep(value1, value2, amount float32) float32 {
	amount = MathHelperClamp(amount, 0, 1)
	return MathHelperLerp(value1, value2, amount*amount*(3-2*amount))
}

func MathHelperCatmullRom(value1, value2, value3, value4, amount float32) float32 {
	squared := amount * amount
	cubed := amount * squared
	return 0.5 * (2*value2 + (-value1+value3)*amount +
		(2*value1-5*value2+4*value3-value4)*squared +
		(-value1+3*value2-3*value3+value4)*cubed)
}

func MathHelperHermite(value1, tangent1, value2, tangent2, amount float32) float32 {
	squared := amount * amount
	cubed := amount * squared
	first := 2*cubed - 3*squared + 1
	second := -2*cubed + 3*squared
	third := cubed - 2*squared + amount
	fourth := cubed - squared
	return value1*first + value2*second + tangent1*third + tangent2*fourth
}

func MathHelperWrapAngle(angle float32) float32 {
	angle = ieeeRemainderFloat32(angle, MathHelperTwoPi)
	if angle <= -MathHelperPi {
		return angle + MathHelperTwoPi
	}
	if angle > MathHelperPi {
		return angle - MathHelperTwoPi
	}
	return angle
}

func ieeeRemainderFloat32(value, divisor float32) float32 {
	if math.IsInf(float64(value), 0) || divisor == 0 || math.IsNaN(float64(divisor)) {
		return float32(math.NaN())
	}
	value64 := float64(value)
	divisor64 := float64(divisor)
	quotient := value64 / divisor64
	floor := math.Floor(quotient)
	fraction := quotient - floor
	nearest := floor
	if fraction > 0.5 || (fraction == 0.5 && math.Mod(floor, 2) != 0) {
		nearest = floor + 1
	}
	result := float32(value64 - divisor64*nearest)
	if result == 0 {
		return float32(math.Copysign(0, value64))
	}
	return result
}
