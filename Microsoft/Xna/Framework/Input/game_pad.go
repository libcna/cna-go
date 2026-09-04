package input

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// GamePad is Microsoft.Xna.Framework.Input.GamePad:
//
//	.class public abstract auto ansi sealed beforefieldinit GamePad
//
// `abstract sealed` is C# for a static class, so the type carries the identity
// and its four members are type-prefixed package functions -- the settled
// projection every static XNA class gets.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # The three readers share one body, and its middle branch is the interesting one
//
//	if (ThrottleDisconnectedRetries(index)) return <empty>;
//	ErrorCodes result = UnsafeNativeMethods.GetState/GetCaps(index, ...);
//	ResetThrottleState(index, result);
//	if (result == 0x48f) return <empty>;              // ERROR_DEVICE_NOT_CONNECTED
//	if (result != success)
//	    throw new InvalidOperationException(FrameworkResources.InvalidController);
//	return new GamePadState(...);
//
// A DISCONNECTED controller is NOT an error. `0x48f` is 1167, Win32's
// ERROR_DEVICE_NOT_CONNECTED, and the reference answers an empty value whose
// IsConnected is false. Only some OTHER failure throws.
//
// So a consumer polls four player indices every frame and gets four answers,
// three of them empty, without a single exception. A projection that turned a
// missing controller into an error would break the loop every game writes.
//
// # The retry throttle is private and is not projected
//
// XInput is slow to answer for an absent controller, so the reference
// rate-limits the retry with two private members and a static table. Nothing
// public reports it, CNA performs its own polling, and reproducing a throttle
// over a different backend would be reproducing a workaround rather than a
// behaviour.
type GamePad struct{}

// errGamePadNoRunningGame is the Go-only refusal these statics can answer. The
// reference's are static and reach XInput directly; CNA's routes read the
// window's own event state and need a lifecycle callback.
var errGamePadNoRunningGame = errors.New("this member needs a running game")

// GamePadGetStateByPlayerIndex is GamePad::GetState(PlayerIndex), which the
// reference implements as `GetState(index, GamePadDeadZone.IndependentAxes)` --
// the `ldc.i4.1` in its two-instruction body.
func GamePadGetStateByPlayerIndex(playerIndex framework.PlayerIndex) (GamePadState, error) {
	return GamePadGetStateByPlayerIndexAndGamePadDeadZone(playerIndex, GamePadDeadZoneIndependentAxes)
}

// GamePadGetStateByPlayerIndexAndGamePadDeadZone is
// GamePad::GetState(PlayerIndex, GamePadDeadZone).
//
// A disconnected controller answers the ZERO GamePadState and no error, which
// is what the reference's ERROR_DEVICE_NOT_CONNECTED branch returns.
func GamePadGetStateByPlayerIndexAndGamePadDeadZone(
	playerIndex framework.PlayerIndex, deadZone GamePadDeadZone,
) (GamePadState, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return GamePadState{}, errGamePadNoRunningGame
	}
	values, err := runtime.GamePadState(uint32(playerIndex), uint32(deadZone), true)
	if err != nil {
		return GamePadState{}, fmt.Errorf("%w: %s", err, invalidController)
	}
	if !values.IsConnected {
		// The reference's empty state: IsConnected false and every value zero.
		return GamePadState{}, nil
	}
	// The five-argument constructor is the one that does exactly this: it
	// combines the button list with a bitwise OR, builds the thumbsticks and
	// triggers through their own CLAMPING constructors, derives the DPad from
	// the combined value with a non-zero bit test, and marks the state
	// connected. Reproducing any of that here would be reproducing a body the
	// projection already has.
	state := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.NewVector2BySingleAndSingle(values.LeftThumbX, values.LeftThumbY),
		framework.NewVector2BySingleAndSingle(values.RightThumbX, values.RightThumbY),
		values.LeftTrigger, values.RightTrigger,
		buttonsFromMask(values.PressedButtons))
	// The PUBLIC constructors all set the packet number to zero; the one the
	// reference's GetState actually calls is
	// `GamePadState::.ctor(XINPUT_STATE&, ...)`, which reads it from the
	// XInput snapshot. CNA reports the same number, so it is stored here rather
	// than left at zero.
	//
	// This is not cosmetic. PacketNumber participates in GamePadState's Equals
	// and GetHashCode, and its whole purpose is to let a consumer tell "the
	// controller has not moved" from "the controller reports the same values
	// again" -- which a projection that always answered zero would collapse.
	state.packet = values.PacketNumber
	return state, nil
}

// GamePadGetCapabilities is
// GamePad::GetCapabilities(PlayerIndex), on the same three-branch body.
func GamePadGetCapabilities(playerIndex framework.PlayerIndex) (GamePadCapabilities, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return GamePadCapabilities{}, errGamePadNoRunningGame
	}
	padType, flags, err := runtime.GamePadCapabilities(uint32(playerIndex))
	if err != nil {
		return GamePadCapabilities{}, fmt.Errorf("%w: %s", err, invalidController)
	}
	var projected [gamePadCapabilityFlagCount]bool
	for index := range projected {
		projected[index] = flags[index] != 0
	}
	return newGamePadCapabilities(GamePadType(padType), projected), nil
}

// GamePadSetVibration is
// GamePad::SetVibration(PlayerIndex, Single, Single), whose Boolean return says
// whether the vibration was APPLIED -- false for a controller that is not
// there or has no motors, and not an error either way.
//
// CNA's route reports the same boolean under the same name.
func GamePadSetVibration(
	playerIndex framework.PlayerIndex, leftMotor, rightMotor float32,
) (bool, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errGamePadNoRunningGame
	}
	return runtime.GamePadSetVibration(uint32(playerIndex), leftMotor, rightMotor)
}

// invalidController is the FrameworkResources message the reference's
// InvalidOperationException carries when the native call fails for a reason
// other than a missing controller.
const invalidController = "An invalid operation was performed. Is your PlayerIndex correct?"

// buttonsFromMask expands CNA's pressed-button mask into the Buttons values the
// GamePadButtons constructor takes. The mask's bit positions are XNA's own
// Buttons literals, which the enum's projection already pins.
func buttonsFromMask(mask uint32) []Buttons {
	all := []Buttons{
		ButtonsDPadUp, ButtonsDPadDown, ButtonsDPadLeft, ButtonsDPadRight,
		ButtonsStart, ButtonsBack, ButtonsLeftStick, ButtonsRightStick,
		ButtonsLeftShoulder, ButtonsRightShoulder, ButtonsBigButton,
		ButtonsA, ButtonsB, ButtonsX, ButtonsY,
		ButtonsLeftThumbstickLeft, ButtonsRightTrigger, ButtonsLeftTrigger,
		ButtonsRightThumbstickUp, ButtonsRightThumbstickDown,
		ButtonsRightThumbstickRight, ButtonsRightThumbstickLeft,
		ButtonsLeftThumbstickUp, ButtonsLeftThumbstickDown, ButtonsLeftThumbstickRight,
	}
	pressed := make([]Buttons, 0, len(all))
	for _, button := range all {
		if mask&uint32(button) != 0 {
			pressed = append(pressed, button)
		}
	}
	return pressed
}
