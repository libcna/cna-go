package input

// GamePadCapabilities is Microsoft.Xna.Framework.Input.GamePadCapabilities:
//
//	.class public sequential ansi sealed beforefieldinit GamePadCapabilities
//	       extends [mscorlib]System.ValueType
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Twenty-six properties, twenty-four of which are one boolean each
//
// It has no public constructor: `GamePad.GetCapabilities` produces one, and a
// disconnected controller produces the ZERO value -- whose `IsConnected` is
// false and whose every capability is false, which is exactly what a caller
// should see for a controller that is not there.
//
// # Pure managed
//
// Every member reads a field. The values arrive once, from
// `cna_gamepad_get_capabilities`, and nothing here reaches CNA again.
//
// # CNA carries eleven capabilities XNA does not
//
// Its structure adds a light bar, trigger vibration motors, a misc button, four
// paddles, a touchpad, a gyro and an accelerometer. None is in the pinned
// contract, so none is projected: this type is XNA's twenty-six and not CNA's
// thirty-seven.
type GamePadCapabilities struct {
	gamePadType GamePadType
	// flags holds the twenty-five booleans in the order the contract declares
	// them, which is the order the bridge fills its array in. One field rather
	// than twenty-five keeps the two sides' agreement in one place.
	flags [gamePadCapabilityFlagCount]bool
}

// The flag positions, in the order GamePadCapabilities declares its properties.
// The bridge fills its array in this order and the two must agree; naming the
// indices is what makes a disagreement a compile error rather than a wrong
// answer.
const (
	gamePadCapabilityIsConnected = iota
	gamePadCapabilityHasAButton
	gamePadCapabilityHasBButton
	gamePadCapabilityHasXButton
	gamePadCapabilityHasYButton
	gamePadCapabilityHasBackButton
	gamePadCapabilityHasStartButton
	gamePadCapabilityHasBigButton
	gamePadCapabilityHasDPadUpButton
	gamePadCapabilityHasDPadDownButton
	gamePadCapabilityHasDPadLeftButton
	gamePadCapabilityHasDPadRightButton
	gamePadCapabilityHasLeftShoulderButton
	gamePadCapabilityHasRightShoulderButton
	gamePadCapabilityHasLeftStickButton
	gamePadCapabilityHasRightStickButton
	gamePadCapabilityHasLeftXThumbStick
	gamePadCapabilityHasLeftYThumbStick
	gamePadCapabilityHasRightXThumbStick
	gamePadCapabilityHasRightYThumbStick
	gamePadCapabilityHasLeftTrigger
	gamePadCapabilityHasRightTrigger
	gamePadCapabilityHasLeftVibrationMotor
	gamePadCapabilityHasRightVibrationMotor
	gamePadCapabilityHasVoiceSupport
	gamePadCapabilityFlagCount
)

// newGamePadCapabilities builds one from the values CNA reported. It is
// unexported because the reference's constructor is too: a consumer receives
// one from GetCapabilities and never builds one.
func newGamePadCapabilities(padType GamePadType, flags [gamePadCapabilityFlagCount]bool) GamePadCapabilities {
	return GamePadCapabilities{gamePadType: padType, flags: flags}
}

// GamePadType is GamePadCapabilities::get_GamePadType.
func (c GamePadCapabilities) GamePadType() GamePadType { return c.gamePadType }

// IsConnected is GamePadCapabilities::get_IsConnected, and it is the one a
// caller must read first: every other property is false on a disconnected
// controller, so a capability that answers false says nothing on its own.
func (c GamePadCapabilities) IsConnected() bool { return c.flags[gamePadCapabilityIsConnected] }

// The twenty-four capability reads, each one field.

func (c GamePadCapabilities) HasAButton() bool { return c.flags[gamePadCapabilityHasAButton] }
func (c GamePadCapabilities) HasBButton() bool { return c.flags[gamePadCapabilityHasBButton] }
func (c GamePadCapabilities) HasXButton() bool { return c.flags[gamePadCapabilityHasXButton] }
func (c GamePadCapabilities) HasYButton() bool { return c.flags[gamePadCapabilityHasYButton] }

func (c GamePadCapabilities) HasBackButton() bool {
	return c.flags[gamePadCapabilityHasBackButton]
}

func (c GamePadCapabilities) HasStartButton() bool {
	return c.flags[gamePadCapabilityHasStartButton]
}

func (c GamePadCapabilities) HasBigButton() bool { return c.flags[gamePadCapabilityHasBigButton] }

func (c GamePadCapabilities) HasDPadUpButton() bool {
	return c.flags[gamePadCapabilityHasDPadUpButton]
}

func (c GamePadCapabilities) HasDPadDownButton() bool {
	return c.flags[gamePadCapabilityHasDPadDownButton]
}

func (c GamePadCapabilities) HasDPadLeftButton() bool {
	return c.flags[gamePadCapabilityHasDPadLeftButton]
}

func (c GamePadCapabilities) HasDPadRightButton() bool {
	return c.flags[gamePadCapabilityHasDPadRightButton]
}

func (c GamePadCapabilities) HasLeftShoulderButton() bool {
	return c.flags[gamePadCapabilityHasLeftShoulderButton]
}

func (c GamePadCapabilities) HasRightShoulderButton() bool {
	return c.flags[gamePadCapabilityHasRightShoulderButton]
}

func (c GamePadCapabilities) HasLeftStickButton() bool {
	return c.flags[gamePadCapabilityHasLeftStickButton]
}

func (c GamePadCapabilities) HasRightStickButton() bool {
	return c.flags[gamePadCapabilityHasRightStickButton]
}

func (c GamePadCapabilities) HasLeftXThumbStick() bool {
	return c.flags[gamePadCapabilityHasLeftXThumbStick]
}

func (c GamePadCapabilities) HasLeftYThumbStick() bool {
	return c.flags[gamePadCapabilityHasLeftYThumbStick]
}

func (c GamePadCapabilities) HasRightXThumbStick() bool {
	return c.flags[gamePadCapabilityHasRightXThumbStick]
}

func (c GamePadCapabilities) HasRightYThumbStick() bool {
	return c.flags[gamePadCapabilityHasRightYThumbStick]
}

func (c GamePadCapabilities) HasLeftTrigger() bool {
	return c.flags[gamePadCapabilityHasLeftTrigger]
}

func (c GamePadCapabilities) HasRightTrigger() bool {
	return c.flags[gamePadCapabilityHasRightTrigger]
}

func (c GamePadCapabilities) HasLeftVibrationMotor() bool {
	return c.flags[gamePadCapabilityHasLeftVibrationMotor]
}

func (c GamePadCapabilities) HasRightVibrationMotor() bool {
	return c.flags[gamePadCapabilityHasRightVibrationMotor]
}

func (c GamePadCapabilities) HasVoiceSupport() bool {
	return c.flags[gamePadCapabilityHasVoiceSupport]
}
