package input

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// xinputButtonBits are the XInput button bits the reference packs into its
// private snapshot. Each one coincides with the pinned Buttons literal of the
// same name; TestGamePadStateXInputBitsMatchPinnedButtons measures that
// coincidence rather than assuming it.
const (
	xinputDPadUp        uint16 = 0x0001
	xinputDPadDown      uint16 = 0x0002
	xinputDPadLeft      uint16 = 0x0004
	xinputDPadRight     uint16 = 0x0008
	xinputStart         uint16 = 0x0010
	xinputBack          uint16 = 0x0020
	xinputLeftStick     uint16 = 0x0040
	xinputRightStick    uint16 = 0x0080
	xinputLeftShoulder  uint16 = 0x0100
	xinputRightShoulder uint16 = 0x0200
	xinputBigButton     uint16 = 0x0800
	xinputA             uint16 = 0x1000
	xinputB             uint16 = 0x2000
	xinputX             uint16 = 0x4000
	xinputY             uint16 = 0x8000
)

// normalButtonMask is the reference `_normalButtonMask` literal. It clears
// bit 0x0400, which XInput does not define.
const normalButtonMask Buttons = 0xFBFF

// xinputGamePad mirrors the private XINPUT_GAMEPAD snapshot the reference
// keeps inside GamePadState. It is an ordinary managed Go struct: it is never
// marshalled, never handed to a native library, and never exposed publicly.
// It exists only because IsButtonDown reads its packed form rather than the
// public button values.
type xinputGamePad struct {
	buttons      uint16
	leftTrigger  uint8
	rightTrigger uint8
	thumbLX      int16
	thumbLY      int16
	thumbRX      int16
	thumbRY      int16
}

// GamePadState is a managed value describing one XNA game pad snapshot.
// Constructing it claims no game pad capability: CNA-Go exposes no GamePad
// type, performs no polling, and reads no device. IsConnected reports what the
// constructor stored, which both public constructors set to true.
type GamePadState struct {
	connected bool
	packet    int32
	thumbs    GamePadThumbSticks
	triggers  GamePadTriggers
	buttons   GamePadButtons
	dpad      GamePadDPad
	state     xinputGamePad
}

// NewGamePadStateByGamePadThumbSticksAndGamePadTriggersAndGamePadButtonsAndGamePadDPad
// stores the four supplied values, sets the packet number to zero, marks the
// state connected, and fills the private XInput snapshot.
func NewGamePadStateByGamePadThumbSticksAndGamePadTriggersAndGamePadButtonsAndGamePadDPad(thumbSticks GamePadThumbSticks, triggers GamePadTriggers, buttons GamePadButtons, dPad GamePadDPad) GamePadState {
	state := GamePadState{
		connected: true,
		thumbs:    thumbSticks,
		triggers:  triggers,
		buttons:   buttons,
		dpad:      dPad,
	}
	state.fillInternalState()
	return state
}

// NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons
// combines the supplied button values with a bitwise OR, builds the thumbstick
// and trigger values through their own clamping constructors, and derives the
// directional pad from the combined value with a non-zero bit test.
func NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(leftThumbStick, rightThumbStick framework.Vector2, leftTrigger, rightTrigger float32, buttons []Buttons) GamePadState {
	var combined Buttons
	for _, button := range buttons {
		combined |= button
	}
	state := GamePadState{
		connected: true,
		thumbs:    NewGamePadThumbSticks(leftThumbStick, rightThumbStick),
		triggers:  NewGamePadTriggers(leftTrigger, rightTrigger),
		buttons:   NewGamePadButtons(combined),
		dpad: GamePadDPad{
			up:    dpadStateFromCombined(combined, ButtonsDPadUp),
			down:  dpadStateFromCombined(combined, ButtonsDPadDown),
			left:  dpadStateFromCombined(combined, ButtonsDPadLeft),
			right: dpadStateFromCombined(combined, ButtonsDPadRight),
		},
	}
	state.fillInternalState()
	return state
}

// dpadStateFromCombined reproduces the reference non-zero bit test. It differs
// in form from the GamePadButtons mask rule, which compares against the whole
// mask, but the two agree for the single-bit directional literals.
func dpadStateFromCombined(combined, mask Buttons) ButtonState {
	if combined&mask != 0 {
		return ButtonStatePressed
	}
	return ButtonStateReleased
}

// fillInternalState reproduces the reference FillInternalState. It is pure
// managed bit packing over the public values: no native call, no marshalling,
// and no device access.
func (s *GamePadState) fillInternalState() {
	s.state = xinputGamePad{}
	var packed uint16
	for _, entry := range []struct {
		state ButtonState
		bit   uint16
	}{
		{s.buttons.A(), xinputA},
		{s.buttons.B(), xinputB},
		{s.buttons.X(), xinputX},
		{s.buttons.Y(), xinputY},
		{s.buttons.Back(), xinputBack},
		{s.buttons.LeftShoulder(), xinputLeftShoulder},
		{s.buttons.LeftStick(), xinputLeftStick},
		{s.buttons.RightShoulder(), xinputRightShoulder},
		{s.buttons.RightStick(), xinputRightStick},
		{s.buttons.Start(), xinputStart},
		{s.buttons.BigButton(), xinputBigButton},
		{s.dpad.Up(), xinputDPadUp},
		{s.dpad.Down(), xinputDPadDown},
		{s.dpad.Right(), xinputDPadRight},
		{s.dpad.Left(), xinputDPadLeft},
	} {
		if entry.state == ButtonStatePressed {
			packed |= entry.bit
		}
	}
	s.state.buttons = packed
	s.state.leftTrigger = clrConvertToUInt8(s.triggers.Left() * 255)
	s.state.rightTrigger = clrConvertToUInt8(s.triggers.Right() * 255)
	s.state.thumbLX = clrConvertToInt16(s.thumbs.Left().X * 32767)
	s.state.thumbLY = clrConvertToInt16(s.thumbs.Left().Y * 32767)
	s.state.thumbRX = clrConvertToInt16(s.thumbs.Right().X * 32767)
	s.state.thumbRY = clrConvertToInt16(s.thumbs.Right().Y * 32767)
}

func (s GamePadState) IsConnected() bool               { return s.connected }
func (s GamePadState) PacketNumber() int32             { return s.packet }
func (s GamePadState) ThumbSticks() GamePadThumbSticks { return s.thumbs }
func (s GamePadState) Triggers() GamePadTriggers       { return s.triggers }
func (s GamePadState) Buttons() GamePadButtons         { return s.buttons }
func (s GamePadState) DPad() GamePadDPad               { return s.dpad }

// IsButtonDown reproduces the reference exactly. It starts from the packed
// XInput button field masked with the normal-button mask, then adds virtual
// bits for any thumbstick direction or trigger the caller actually asked
// about — each derived through the IndependentAxes dead zone — and finally
// requires every requested bit to be present. Asking about no button at all
// therefore reports true, as it does in the reference.
func (s GamePadState) IsButtonDown(button Buttons) bool {
	effective := Buttons(s.state.buttons) & normalButtonMask

	if button&ButtonsLeftThumbstickLeft == ButtonsLeftThumbstickLeft &&
		applyLeftStickDeadZone(int32(s.state.thumbLX), int32(s.state.thumbLY)).X < 0 {
		effective |= ButtonsLeftThumbstickLeft
	}
	if button&ButtonsLeftThumbstickRight == ButtonsLeftThumbstickRight &&
		applyLeftStickDeadZone(int32(s.state.thumbLX), int32(s.state.thumbLY)).X > 0 {
		effective |= ButtonsLeftThumbstickRight
	}
	if button&ButtonsLeftThumbstickDown == ButtonsLeftThumbstickDown &&
		applyLeftStickDeadZone(int32(s.state.thumbLX), int32(s.state.thumbLY)).Y < 0 {
		effective |= ButtonsLeftThumbstickDown
	}
	if button&ButtonsLeftThumbstickUp == ButtonsLeftThumbstickUp &&
		applyLeftStickDeadZone(int32(s.state.thumbLX), int32(s.state.thumbLY)).Y > 0 {
		effective |= ButtonsLeftThumbstickUp
	}
	if button&ButtonsRightThumbstickLeft == ButtonsRightThumbstickLeft &&
		applyRightStickDeadZone(int32(s.state.thumbRX), int32(s.state.thumbRY)).X < 0 {
		effective |= ButtonsRightThumbstickLeft
	}
	if button&ButtonsRightThumbstickRight == ButtonsRightThumbstickRight &&
		applyRightStickDeadZone(int32(s.state.thumbRX), int32(s.state.thumbRY)).X > 0 {
		effective |= ButtonsRightThumbstickRight
	}
	if button&ButtonsRightThumbstickDown == ButtonsRightThumbstickDown &&
		applyRightStickDeadZone(int32(s.state.thumbRX), int32(s.state.thumbRY)).Y < 0 {
		effective |= ButtonsRightThumbstickDown
	}
	if button&ButtonsRightThumbstickUp == ButtonsRightThumbstickUp &&
		applyRightStickDeadZone(int32(s.state.thumbRX), int32(s.state.thumbRY)).Y > 0 {
		effective |= ButtonsRightThumbstickUp
	}
	if button&ButtonsLeftTrigger == ButtonsLeftTrigger &&
		applyTriggerDeadZone(int32(s.state.leftTrigger)) > 0 {
		effective |= ButtonsLeftTrigger
	}
	if button&ButtonsRightTrigger == ButtonsRightTrigger &&
		applyTriggerDeadZone(int32(s.state.rightTrigger)) > 0 {
		effective |= ButtonsRightTrigger
	}
	return button&effective == button
}

// IsButtonUp is the exact logical negation of IsButtonDown.
func (s GamePadState) IsButtonUp(button Buttons) bool { return !s.IsButtonDown(button) }

func (s GamePadState) Equals(obj any) bool {
	other, ok := obj.(GamePadState)
	return ok && GamePadStateOperatorEqualityByGamePadStateAndGamePadState(s, other)
}

// GetHashCode reproduces the reference XOR of the six stored member hash
// codes. Boolean.GetHashCode returns 1 for true and 0 for false, and
// Int32.GetHashCode returns the value itself. The derived XInput snapshot does
// not participate.
func (s GamePadState) GetHashCode() int32 {
	connected := int32(0)
	if s.connected {
		connected = 1
	}
	return s.thumbs.GetHashCode() ^ s.triggers.GetHashCode() ^
		s.buttons.GetHashCode() ^ connected ^
		s.dpad.GetHashCode() ^ s.packet
}

// ToString reports only the connection flag, formatted the way the CLR formats
// a Boolean.
func (s GamePadState) ToString() string {
	return fmt.Sprintf("{IsConnected:%s}", formatBoolean(s.connected))
}

func formatBoolean(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

// GamePadStateOperatorEqualityByGamePadStateAndGamePadState compares the six
// stored members. The derived XInput snapshot is not compared, because it is a
// pure function of them.
func GamePadStateOperatorEqualityByGamePadStateAndGamePadState(left, right GamePadState) bool {
	return left.connected == right.connected &&
		left.packet == right.packet &&
		GamePadThumbSticksOperatorEqualityByGamePadThumbSticksAndGamePadThumbSticks(left.thumbs, right.thumbs) &&
		GamePadTriggersOperatorEqualityByGamePadTriggersAndGamePadTriggers(left.triggers, right.triggers) &&
		GamePadButtonsOperatorEqualityByGamePadButtonsAndGamePadButtons(left.buttons, right.buttons) &&
		GamePadDPadOperatorEqualityByGamePadDPadAndGamePadDPad(left.dpad, right.dpad)
}

func GamePadStateOperatorInequalityByGamePadStateAndGamePadState(left, right GamePadState) bool {
	return !GamePadStateOperatorEqualityByGamePadStateAndGamePadState(left, right)
}
