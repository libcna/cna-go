package input

import "fmt"

// GamePadButtons is a managed value describing the eleven XNA game pad face,
// shoulder, stick, and system buttons. Constructing it claims no game pad
// capability: it derives its states arithmetically from a Buttons value and
// has no polling, device, or backend route.
type GamePadButtons struct {
	a             ButtonState
	b             ButtonState
	x             ButtonState
	y             ButtonState
	leftStick     ButtonState
	rightStick    ButtonState
	leftShoulder  ButtonState
	rightShoulder ButtonState
	back          ButtonState
	start         ButtonState
	bigButton     ButtonState
}

// NewGamePadButtons derives each button state from the supplied flags value
// with the reference rule `(buttons & mask) == mask ? Pressed : Released`.
// Each mask is the pinned Buttons literal of the same name.
func NewGamePadButtons(buttons Buttons) GamePadButtons {
	return GamePadButtons{
		a:             buttonStateFromMask(buttons, ButtonsA),
		b:             buttonStateFromMask(buttons, ButtonsB),
		x:             buttonStateFromMask(buttons, ButtonsX),
		y:             buttonStateFromMask(buttons, ButtonsY),
		start:         buttonStateFromMask(buttons, ButtonsStart),
		back:          buttonStateFromMask(buttons, ButtonsBack),
		leftStick:     buttonStateFromMask(buttons, ButtonsLeftStick),
		rightStick:    buttonStateFromMask(buttons, ButtonsRightStick),
		leftShoulder:  buttonStateFromMask(buttons, ButtonsLeftShoulder),
		rightShoulder: buttonStateFromMask(buttons, ButtonsRightShoulder),
		bigButton:     buttonStateFromMask(buttons, ButtonsBigButton),
	}
}

func (b GamePadButtons) A() ButtonState             { return b.a }
func (b GamePadButtons) B() ButtonState             { return b.b }
func (b GamePadButtons) X() ButtonState             { return b.x }
func (b GamePadButtons) Y() ButtonState             { return b.y }
func (b GamePadButtons) LeftStick() ButtonState     { return b.leftStick }
func (b GamePadButtons) RightStick() ButtonState    { return b.rightStick }
func (b GamePadButtons) LeftShoulder() ButtonState  { return b.leftShoulder }
func (b GamePadButtons) RightShoulder() ButtonState { return b.rightShoulder }
func (b GamePadButtons) Back() ButtonState          { return b.back }
func (b GamePadButtons) Start() ButtonState         { return b.start }
func (b GamePadButtons) BigButton() ButtonState     { return b.bigButton }

func (b GamePadButtons) Equals(obj any) bool {
	other, ok := obj.(GamePadButtons)
	return ok && GamePadButtonsOperatorEqualityByGamePadButtonsAndGamePadButtons(b, other)
}

// GetHashCode reproduces Helpers.SmartGetHashCode over the eleven Int32 words
// of the sequential layout.
func (b GamePadButtons) GetHashCode() int32 {
	return smartGetHashCode(
		int32(b.a), int32(b.b), int32(b.x), int32(b.y),
		int32(b.leftStick), int32(b.rightStick),
		int32(b.leftShoulder), int32(b.rightShoulder),
		int32(b.back), int32(b.start), int32(b.bigButton),
	)
}

// ToString appends the name of every pressed button in the reference order,
// separated by single spaces, and substitutes "None" when nothing is pressed.
func (b GamePadButtons) ToString() string {
	names := ""
	names = appendPressedName(names, b.a, "A")
	names = appendPressedName(names, b.b, "B")
	names = appendPressedName(names, b.x, "X")
	names = appendPressedName(names, b.y, "Y")
	names = appendPressedName(names, b.leftShoulder, "LeftShoulder")
	names = appendPressedName(names, b.rightShoulder, "RightShoulder")
	names = appendPressedName(names, b.leftStick, "LeftStick")
	names = appendPressedName(names, b.rightStick, "RightStick")
	names = appendPressedName(names, b.start, "Start")
	names = appendPressedName(names, b.back, "Back")
	names = appendPressedName(names, b.bigButton, "BigButton")
	return fmt.Sprintf("{Buttons:%s}", pressedNamesOrNone(names))
}

func GamePadButtonsOperatorEqualityByGamePadButtonsAndGamePadButtons(left, right GamePadButtons) bool {
	return left.a == right.a && left.b == right.b && left.x == right.x && left.y == right.y &&
		left.leftShoulder == right.leftShoulder && left.rightShoulder == right.rightShoulder &&
		left.leftStick == right.leftStick && left.rightStick == right.rightStick &&
		left.back == right.back && left.start == right.start && left.bigButton == right.bigButton
}

func GamePadButtonsOperatorInequalityByGamePadButtonsAndGamePadButtons(left, right GamePadButtons) bool {
	return !GamePadButtonsOperatorEqualityByGamePadButtonsAndGamePadButtons(left, right)
}
