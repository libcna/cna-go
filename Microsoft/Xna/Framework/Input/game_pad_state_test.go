package input

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestGamePadStateXInputBitsMatchPinnedButtons measures, rather than assumes,
// that every XInput bit the reference packs equals the pinned Buttons literal
// of the same name.
func TestGamePadStateXInputBitsMatchPinnedButtons(t *testing.T) {
	for _, item := range []struct {
		name   string
		xinput uint16
		pinned Buttons
	}{
		{"DPadUp", xinputDPadUp, ButtonsDPadUp},
		{"DPadDown", xinputDPadDown, ButtonsDPadDown},
		{"DPadLeft", xinputDPadLeft, ButtonsDPadLeft},
		{"DPadRight", xinputDPadRight, ButtonsDPadRight},
		{"Start", xinputStart, ButtonsStart},
		{"Back", xinputBack, ButtonsBack},
		{"LeftStick", xinputLeftStick, ButtonsLeftStick},
		{"RightStick", xinputRightStick, ButtonsRightStick},
		{"LeftShoulder", xinputLeftShoulder, ButtonsLeftShoulder},
		{"RightShoulder", xinputRightShoulder, ButtonsRightShoulder},
		{"BigButton", xinputBigButton, ButtonsBigButton},
		{"A", xinputA, ButtonsA},
		{"B", xinputB, ButtonsB},
		{"X", xinputX, ButtonsX},
		{"Y", xinputY, ButtonsY},
	} {
		if Buttons(item.xinput) != item.pinned {
			t.Fatalf("%s: XInput bit 0x%04X != pinned Buttons %d", item.name, item.xinput, item.pinned)
		}
	}
	if normalButtonMask != 0xFBFF {
		t.Fatalf("normalButtonMask = 0x%X", normalButtonMask)
	}
}

func TestGamePadStateComponentConstructor(t *testing.T) {
	sticks := NewGamePadThumbSticks(framework.Vector2{X: 0.5}, framework.Vector2{})
	triggers := NewGamePadTriggers(0.5, 0)
	buttons := NewGamePadButtons(ButtonsA | ButtonsStart)
	dpad := NewGamePadDPad(ButtonStatePressed, ButtonStateReleased, ButtonStateReleased, ButtonStateReleased)

	state := NewGamePadStateByGamePadThumbSticksAndGamePadTriggersAndGamePadButtonsAndGamePadDPad(sticks, triggers, buttons, dpad)
	if !state.IsConnected() {
		t.Fatal("both public constructors mark the state connected")
	}
	if state.PacketNumber() != 0 {
		t.Fatalf("PacketNumber = %d, want 0", state.PacketNumber())
	}
	if state.ThumbSticks() != sticks || state.Triggers() != triggers ||
		state.Buttons() != buttons || state.DPad() != dpad {
		t.Fatal("components were not stored unchanged")
	}
}

func TestGamePadStateButtonsSliceConstructor(t *testing.T) {
	state := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: 2}, framework.Vector2{}, 1.5, -1,
		[]Buttons{ButtonsA, ButtonsDPadUp, ButtonsDPadLeft},
	)
	// The slice is combined with a bitwise OR.
	if state.Buttons().A() != ButtonStatePressed || state.Buttons().B() != ButtonStateReleased {
		t.Fatal("the Buttons slice was not combined")
	}
	if state.DPad().Up() != ButtonStatePressed || state.DPad().Left() != ButtonStatePressed ||
		state.DPad().Down() != ButtonStateReleased || state.DPad().Right() != ButtonStateReleased {
		t.Fatalf("dpad = %+v", state.DPad())
	}
	// The thumbstick and trigger values go through their own clamping
	// constructors.
	if state.ThumbSticks().Left().X != 1 {
		t.Fatalf("thumbstick was not clamped: %v", state.ThumbSticks().Left())
	}
	if state.Triggers().Left() != 1 || state.Triggers().Right() != 0 {
		t.Fatalf("triggers were not clamped: %v/%v", state.Triggers().Left(), state.Triggers().Right())
	}
	// A nil slice combines to zero.
	empty := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0, nil)
	if empty.Buttons() != (GamePadButtons{}) || empty.DPad() != (GamePadDPad{}) {
		t.Fatal("a nil Buttons slice did not combine to zero")
	}
}

func TestGamePadStateIsButtonDownForNormalButtons(t *testing.T) {
	state := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0,
		[]Buttons{ButtonsA, ButtonsBigButton, ButtonsDPadDown},
	)
	for _, down := range []Buttons{ButtonsA, ButtonsBigButton, ButtonsDPadDown} {
		if !state.IsButtonDown(down) || state.IsButtonUp(down) {
			t.Fatalf("button %d should be down", down)
		}
	}
	for _, up := range []Buttons{ButtonsB, ButtonsX, ButtonsY, ButtonsStart, ButtonsDPadUp} {
		if state.IsButtonDown(up) || !state.IsButtonUp(up) {
			t.Fatalf("button %d should be up", up)
		}
	}
	// Every requested bit must be present.
	if !state.IsButtonDown(ButtonsA | ButtonsBigButton) {
		t.Fatal("a combination of two pressed buttons should be down")
	}
	if state.IsButtonDown(ButtonsA | ButtonsB) {
		t.Fatal("a combination containing a released button must not be down")
	}
	// Asking about no button at all reports true, exactly as the reference
	// does: the empty mask is trivially contained.
	if !state.IsButtonDown(0) {
		t.Fatal("IsButtonDown(0) must report true")
	}
}

func TestGamePadStateThumbstickDeadZone(t *testing.T) {
	// 0.5 quantizes to 16383, which clears the 7849 left-stick dead zone.
	far := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: 0.5}, framework.Vector2{Y: 0.5}, 0, 0, nil)
	if !far.IsButtonDown(ButtonsLeftThumbstickRight) {
		t.Fatal("a half-deflected left stick should report LeftThumbstickRight")
	}
	if far.IsButtonDown(ButtonsLeftThumbstickLeft) || far.IsButtonDown(ButtonsLeftThumbstickUp) {
		t.Fatal("the opposite and orthogonal directions must stay up")
	}
	if !far.IsButtonDown(ButtonsRightThumbstickUp) {
		t.Fatal("a half-deflected right stick should report RightThumbstickUp")
	}

	// 0.1 quantizes to 3276, which is inside the dead zone and collapses to
	// zero, so no direction is reported.
	near := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: 0.1}, framework.Vector2{Y: 0.1}, 0, 0, nil)
	for _, direction := range []Buttons{
		ButtonsLeftThumbstickLeft, ButtonsLeftThumbstickRight,
		ButtonsLeftThumbstickUp, ButtonsLeftThumbstickDown,
		ButtonsRightThumbstickLeft, ButtonsRightThumbstickRight,
		ButtonsRightThumbstickUp, ButtonsRightThumbstickDown,
	} {
		if near.IsButtonDown(direction) {
			t.Fatalf("direction %d is inside the dead zone and must stay up", direction)
		}
	}

	negative := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: -0.5, Y: -0.5}, framework.Vector2{}, 0, 0, nil)
	if !negative.IsButtonDown(ButtonsLeftThumbstickLeft) || !negative.IsButtonDown(ButtonsLeftThumbstickDown) {
		t.Fatal("a negatively deflected left stick should report Left and Down")
	}
}

func TestGamePadStateTriggerDeadZone(t *testing.T) {
	// 0.5 quantizes to 127, clearing the trigger dead zone of 30.
	pressed := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0.5, 0.5, nil)
	if !pressed.IsButtonDown(ButtonsLeftTrigger) || !pressed.IsButtonDown(ButtonsRightTrigger) {
		t.Fatal("half-pulled triggers should report down")
	}
	// 0.1 quantizes to 25, inside the dead zone.
	released := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0.1, 0.1, nil)
	if released.IsButtonDown(ButtonsLeftTrigger) || released.IsButtonDown(ButtonsRightTrigger) {
		t.Fatal("triggers inside the dead zone must stay up")
	}
	// A NaN trigger converts to zero and stays inside the dead zone.
	nanTrigger := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, float32(math.NaN()), 0, nil)
	if nanTrigger.IsButtonDown(ButtonsLeftTrigger) {
		t.Fatal("a NaN trigger must not report down")
	}
}

func TestGamePadStateHashStringAndEquality(t *testing.T) {
	var zero GamePadState
	if got := zero.GetHashCode(); got != 0 {
		t.Fatalf("zero GetHashCode = %d, want 0", got)
	}
	if got := zero.ToString(); got != "{IsConnected:False}" {
		t.Fatalf("zero ToString = %q", got)
	}
	connected := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0, nil)
	if got := connected.GetHashCode(); got != 1 {
		t.Fatalf("default connected GetHashCode = %d, want 1", got)
	}
	if got := connected.ToString(); got != "{IsConnected:True}" {
		t.Fatalf("connected ToString = %q", got)
	}
	same := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0, []Buttons{})
	if !connected.Equals(same) ||
		!GamePadStateOperatorEqualityByGamePadStateAndGamePadState(connected, same) ||
		GamePadStateOperatorInequalityByGamePadStateAndGamePadState(connected, same) {
		t.Fatal("equal states did not compare equal")
	}
	different := NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0, []Buttons{ButtonsA})
	if GamePadStateOperatorEqualityByGamePadStateAndGamePadState(connected, different) {
		t.Fatal("different states compared equal")
	}
	if connected.Equals(nil) || connected.Equals("not a state") {
		t.Fatal("Equals accepted a foreign type")
	}
	// The zero value is not connected, so it never equals a constructed one.
	if GamePadStateOperatorEqualityByGamePadStateAndGamePadState(zero, connected) {
		t.Fatal("the zero value must not equal a constructed state")
	}
}

func TestGamePadDeadZoneLinearRule(t *testing.T) {
	if got := applyLinearDeadZone(100, 255, 30); got != 70.0/225.0 {
		t.Fatalf("applyLinearDeadZone(100,255,30) = %v", got)
	}
	if got := applyLinearDeadZone(-100, 255, 30); got != -70.0/225.0 {
		t.Fatalf("negative branch = %v", got)
	}
	for _, inside := range []float32{0, 30, -30, 29.5, float32(math.NaN())} {
		if got := applyLinearDeadZone(inside, 255, 30); got != 0 {
			t.Fatalf("applyLinearDeadZone(%v,255,30) = %v, want 0", inside, got)
		}
	}
	// The result is clamped into [-1, 1].
	if got := applyLinearDeadZone(1e9, 255, 30); got != 1 {
		t.Fatalf("clamp high = %v", got)
	}
	if got := applyLinearDeadZone(-1e9, 255, 30); got != -1 {
		t.Fatalf("clamp low = %v", got)
	}
	// NaN conversions used by the internal snapshot resolve to zero.
	if clrConvertToInt16(float32(math.NaN())) != 0 || clrConvertToUInt8(float32(math.NaN())) != 0 {
		t.Fatal("NaN conversions must resolve to zero")
	}
	if clrConvertToInt16(0.5*32767) != 16383 || clrConvertToUInt8(0.5*255) != 127 {
		t.Fatal("truncation toward zero is wrong")
	}
}
