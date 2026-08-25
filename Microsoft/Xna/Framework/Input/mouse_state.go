package input

import "fmt"

// MouseState is a managed value describing an XNA mouse snapshot.
// Constructing it claims no mouse capability: no cursor, polling, device, or
// backend route exists, and CNA-Go exposes no Mouse type.
type MouseState struct {
	x            int32
	y            int32
	leftButton   ButtonState
	rightButton  ButtonState
	middleButton ButtonState
	xb1          ButtonState
	xb2          ButtonState
	wheel        int32
}

// NewMouseState stores the snapshot exactly as the reference constructor does.
// No value is validated, clamped, or normalized.
func NewMouseState(x, y, scrollWheel int32, leftButton, middleButton, rightButton, xButton1, xButton2 ButtonState) MouseState {
	return MouseState{
		x: x, y: y, wheel: scrollWheel,
		leftButton: leftButton, rightButton: rightButton, middleButton: middleButton,
		xb1: xButton1, xb2: xButton2,
	}
}

func (m MouseState) X() int32                  { return m.x }
func (m MouseState) Y() int32                  { return m.y }
func (m MouseState) LeftButton() ButtonState   { return m.leftButton }
func (m MouseState) MiddleButton() ButtonState { return m.middleButton }
func (m MouseState) RightButton() ButtonState  { return m.rightButton }
func (m MouseState) XButton1() ButtonState     { return m.xb1 }
func (m MouseState) XButton2() ButtonState     { return m.xb2 }
func (m MouseState) ScrollWheelValue() int32   { return m.wheel }

func (m MouseState) Equals(obj any) bool {
	other, ok := obj.(MouseState)
	return ok && MouseStateOperatorEqualityByMouseStateAndMouseState(m, other)
}

// GetHashCode reproduces the reference implementation exactly. Unlike the
// game pad values it does not use Helpers.SmartGetHashCode: it XORs the eight
// member hash codes directly, and Int32.GetHashCode and the Int32-backed
// ButtonState hash both return the value itself. There is therefore no
// Int32.MaxValue substitution — a zero-valued snapshot hashes to zero.
func (m MouseState) GetHashCode() int32 {
	return m.x ^ m.y ^ int32(m.leftButton) ^ int32(m.rightButton) ^
		int32(m.middleButton) ^ int32(m.xb1) ^ int32(m.xb2) ^ m.wheel
}

// ToString appends the name of every pressed button in left, right, middle,
// XButton1, XButton2 order and substitutes "None" when nothing is pressed.
func (m MouseState) ToString() string {
	names := ""
	names = appendPressedName(names, m.leftButton, "Left")
	names = appendPressedName(names, m.rightButton, "Right")
	names = appendPressedName(names, m.middleButton, "Middle")
	names = appendPressedName(names, m.xb1, "XButton1")
	names = appendPressedName(names, m.xb2, "XButton2")
	return fmt.Sprintf("{X:%d Y:%d Buttons:%s Wheel:%d}", m.x, m.y, pressedNamesOrNone(names), m.wheel)
}

func MouseStateOperatorEqualityByMouseStateAndMouseState(left, right MouseState) bool {
	return left.x == right.x && left.y == right.y &&
		left.leftButton == right.leftButton && left.rightButton == right.rightButton &&
		left.middleButton == right.middleButton &&
		left.xb1 == right.xb1 && left.xb2 == right.xb2 &&
		left.wheel == right.wheel
}

func MouseStateOperatorInequalityByMouseStateAndMouseState(left, right MouseState) bool {
	return !MouseStateOperatorEqualityByMouseStateAndMouseState(left, right)
}
