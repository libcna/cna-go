package input

import "fmt"

// GamePadDPad is a managed value describing the four XNA game pad directional
// pad buttons. Constructing it claims no game pad capability.
type GamePadDPad struct {
	up    ButtonState
	right ButtonState
	down  ButtonState
	left  ButtonState
}

// NewGamePadDPad stores the four directional states. The reference constructor
// declares its parameters in up, down, left, right order while its sequential
// fields are declared up, right, down, left; the parameter order below is the
// public contract.
func NewGamePadDPad(upValue, downValue, leftValue, rightValue ButtonState) GamePadDPad {
	return GamePadDPad{up: upValue, right: rightValue, down: downValue, left: leftValue}
}

func (d GamePadDPad) Up() ButtonState    { return d.up }
func (d GamePadDPad) Down() ButtonState  { return d.down }
func (d GamePadDPad) Left() ButtonState  { return d.left }
func (d GamePadDPad) Right() ButtonState { return d.right }

func (d GamePadDPad) Equals(obj any) bool {
	other, ok := obj.(GamePadDPad)
	return ok && GamePadDPadOperatorEqualityByGamePadDPadAndGamePadDPad(d, other)
}

// GetHashCode reproduces Helpers.SmartGetHashCode over the four Int32 words of
// the sequential layout. XOR is commutative, so the declared field order does
// not change the result.
func (d GamePadDPad) GetHashCode() int32 {
	return smartGetHashCode(int32(d.up), int32(d.right), int32(d.down), int32(d.left))
}

// ToString appends the name of every pressed direction in up, down, left,
// right order, separated by single spaces, and substitutes "None" when no
// direction is pressed.
func (d GamePadDPad) ToString() string {
	names := ""
	names = appendPressedName(names, d.up, "Up")
	names = appendPressedName(names, d.down, "Down")
	names = appendPressedName(names, d.left, "Left")
	names = appendPressedName(names, d.right, "Right")
	return fmt.Sprintf("{DPad:%s}", pressedNamesOrNone(names))
}

func GamePadDPadOperatorEqualityByGamePadDPadAndGamePadDPad(left, right GamePadDPad) bool {
	return left.up == right.up && left.down == right.down &&
		left.left == right.left && left.right == right.right
}

func GamePadDPadOperatorInequalityByGamePadDPadAndGamePadDPad(left, right GamePadDPad) bool {
	return !GamePadDPadOperatorEqualityByGamePadDPadAndGamePadDPad(left, right)
}
