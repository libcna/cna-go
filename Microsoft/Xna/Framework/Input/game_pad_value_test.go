package input

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func TestGamePadThumbSticksConstructorClamping(t *testing.T) {
	sticks := NewGamePadThumbSticks(
		framework.Vector2{X: 0.5, Y: -2},
		framework.Vector2{X: 3, Y: float32(math.NaN())},
	)
	if got := sticks.Left(); got.X != 0.5 || got.Y != -1 {
		t.Fatalf("Left = %v, want {0.5 -1}", got)
	}
	// The XNA Vector2 minimum keeps its second operand when the comparison is
	// false, so a NaN component clamps to 1 instead of staying NaN.
	if got := sticks.Right(); got.X != 1 || got.Y != 1 {
		t.Fatalf("Right = %v, want {1 1}", got)
	}
}

func TestGamePadThumbSticksHashEqualityAndString(t *testing.T) {
	sticks := NewGamePadThumbSticks(framework.Vector2{X: 0.5, Y: -2}, framework.Vector2{X: 1, Y: 1})
	if got := sticks.GetHashCode(); got != -2139095040 {
		t.Fatalf("GetHashCode = %d, want -2139095040", got)
	}
	var zero GamePadThumbSticks
	if got := zero.GetHashCode(); got != math.MaxInt32 {
		t.Fatalf("zero GetHashCode = %d, want the Int32.MaxValue substitution", got)
	}
	if got := sticks.ToString(); got != "{Left:{X:0.5 Y:-1} Right:{X:1 Y:1}}" {
		t.Fatalf("ToString = %q", got)
	}
	same := NewGamePadThumbSticks(framework.Vector2{X: 0.5, Y: -1}, framework.Vector2{X: 1, Y: 1})
	if !sticks.Equals(same) || !GamePadThumbSticksOperatorEqualityByGamePadThumbSticksAndGamePadThumbSticks(sticks, same) {
		t.Fatal("equal thumbsticks did not compare equal")
	}
	if GamePadThumbSticksOperatorInequalityByGamePadThumbSticksAndGamePadThumbSticks(sticks, same) {
		t.Fatal("equal thumbsticks compared unequal")
	}
	if sticks.Equals("not a thumbstick value") || sticks.Equals(nil) {
		t.Fatal("Equals accepted a foreign type")
	}
	different := NewGamePadThumbSticks(framework.Vector2{}, framework.Vector2{X: 1, Y: 1})
	if GamePadThumbSticksOperatorEqualityByGamePadThumbSticksAndGamePadThumbSticks(sticks, different) {
		t.Fatal("different thumbsticks compared equal")
	}
}

func TestGamePadTriggersConstructorClamping(t *testing.T) {
	triggers := NewGamePadTriggers(1.5, -0.25)
	if triggers.Left() != 1 || triggers.Right() != 0 {
		t.Fatalf("clamped triggers = %v/%v, want 1/0", triggers.Left(), triggers.Right())
	}
	// System.Math.Min and System.Math.Max propagate NaN, unlike the XNA
	// Vector2 component rule used by GamePadThumbSticks.
	nan := NewGamePadTriggers(float32(math.NaN()), 0.5)
	if !math.IsNaN(float64(nan.Left())) {
		t.Fatalf("NaN trigger = %v, want NaN", nan.Left())
	}
	// Math.Max(-0, 0) returns its second operand, so negative zero becomes
	// positive zero.
	negativeZero := NewGamePadTriggers(float32(math.Copysign(0, -1)), 0)
	if math.Signbit(float64(negativeZero.Left())) {
		t.Fatal("negative zero trigger kept its sign bit")
	}
}

func TestGamePadTriggersHashAndString(t *testing.T) {
	triggers := NewGamePadTriggers(0.25, 0.75)
	if got := triggers.GetHashCode(); got != 29360128 {
		t.Fatalf("GetHashCode = %d, want 29360128", got)
	}
	var zero GamePadTriggers
	if got := zero.GetHashCode(); got != math.MaxInt32 {
		t.Fatalf("zero GetHashCode = %d, want the Int32.MaxValue substitution", got)
	}
	if got := triggers.ToString(); got != "{Left:0.25 Right:0.75}" {
		t.Fatalf("ToString = %q", got)
	}
	if !triggers.Equals(NewGamePadTriggers(0.25, 0.75)) {
		t.Fatal("equal triggers did not compare equal")
	}
	if GamePadTriggersOperatorEqualityByGamePadTriggersAndGamePadTriggers(nan(), nan()) {
		t.Fatal("NaN triggers compared equal; IEEE equality must be preserved")
	}
	if !GamePadTriggersOperatorInequalityByGamePadTriggersAndGamePadTriggers(nan(), nan()) {
		t.Fatal("NaN triggers did not compare unequal")
	}
}

func nan() GamePadTriggers { return NewGamePadTriggers(float32(math.NaN()), 0) }

func TestGamePadDPadConstructorOrderAccessorsAndString(t *testing.T) {
	// The reference constructor parameter order is up, down, left, right.
	dpad := NewGamePadDPad(ButtonStatePressed, ButtonStateReleased, ButtonStatePressed, ButtonStateReleased)
	if dpad.Up() != ButtonStatePressed || dpad.Down() != ButtonStateReleased ||
		dpad.Left() != ButtonStatePressed || dpad.Right() != ButtonStateReleased {
		t.Fatalf("accessors = %d/%d/%d/%d", dpad.Up(), dpad.Down(), dpad.Left(), dpad.Right())
	}
	if got := dpad.ToString(); got != "{DPad:Up Left}" {
		t.Fatalf("ToString = %q, want {DPad:Up Left}", got)
	}
	var none GamePadDPad
	if got := none.ToString(); got != "{DPad:None}" {
		t.Fatalf("empty ToString = %q", got)
	}
	all := NewGamePadDPad(ButtonStatePressed, ButtonStatePressed, ButtonStatePressed, ButtonStatePressed)
	if got := all.ToString(); got != "{DPad:Up Down Left Right}" {
		t.Fatalf("full ToString = %q", got)
	}
	// The reference compares against Pressed exactly, so an arbitrary raw
	// ButtonState contributes no name.
	arbitrary := NewGamePadDPad(ButtonState(12345), ButtonStateReleased, ButtonStateReleased, ButtonStateReleased)
	if got := arbitrary.ToString(); got != "{DPad:None}" {
		t.Fatalf("arbitrary raw ToString = %q", got)
	}
}

func TestGamePadDPadHashAndEquality(t *testing.T) {
	single := NewGamePadDPad(ButtonStatePressed, ButtonStateReleased, ButtonStateReleased, ButtonStateReleased)
	if got := single.GetHashCode(); got != 1 {
		t.Fatalf("single-pressed GetHashCode = %d, want 1", got)
	}
	// Two pressed directions XOR to zero, which the helper substitutes with
	// Int32.MaxValue. The collision is intentional and compatible.
	pair := NewGamePadDPad(ButtonStatePressed, ButtonStateReleased, ButtonStateReleased, ButtonStatePressed)
	var none GamePadDPad
	if pair.GetHashCode() != math.MaxInt32 || none.GetHashCode() != math.MaxInt32 {
		t.Fatalf("substituted hashes = %d/%d", pair.GetHashCode(), none.GetHashCode())
	}
	if !single.Equals(NewGamePadDPad(ButtonStatePressed, ButtonStateReleased, ButtonStateReleased, ButtonStateReleased)) {
		t.Fatal("equal dpads did not compare equal")
	}
	if !GamePadDPadOperatorInequalityByGamePadDPadAndGamePadDPad(single, pair) {
		t.Fatal("different dpads did not compare unequal")
	}
	if single.Equals(pair) || single.Equals(nil) {
		t.Fatal("Equals accepted a different value or a nil object")
	}
}

func TestGamePadButtonsMaskDerivation(t *testing.T) {
	buttons := NewGamePadButtons(ButtonsA | ButtonsStart | ButtonsBigButton)
	if buttons.A() != ButtonStatePressed || buttons.Start() != ButtonStatePressed ||
		buttons.BigButton() != ButtonStatePressed {
		t.Fatal("set masks did not derive Pressed")
	}
	for name, state := range map[string]ButtonState{
		"B": buttons.B(), "X": buttons.X(), "Y": buttons.Y(),
		"LeftStick": buttons.LeftStick(), "RightStick": buttons.RightStick(),
		"LeftShoulder": buttons.LeftShoulder(), "RightShoulder": buttons.RightShoulder(),
		"Back": buttons.Back(),
	} {
		if state != ButtonStateReleased {
			t.Fatalf("%s = %d, want Released", name, state)
		}
	}
	// Thumbstick-direction and trigger literals have no GamePadButtons field
	// and must not turn any face button on.
	stray := NewGamePadButtons(ButtonsLeftThumbstickUp | ButtonsRightTrigger)
	if stray != (GamePadButtons{}) {
		t.Fatalf("stray literals produced %v", stray)
	}
}

func TestGamePadButtonsHashAndString(t *testing.T) {
	buttons := NewGamePadButtons(ButtonsA)
	if got := buttons.GetHashCode(); got != 1 {
		t.Fatalf("GetHashCode = %d, want 1", got)
	}
	// A and Start XOR to zero and collide on the substitution, as does the
	// all-released value.
	pair := NewGamePadButtons(ButtonsA | ButtonsStart)
	var none GamePadButtons
	if pair.GetHashCode() != math.MaxInt32 || none.GetHashCode() != math.MaxInt32 {
		t.Fatalf("substituted hashes = %d/%d", pair.GetHashCode(), none.GetHashCode())
	}
	if got := none.ToString(); got != "{Buttons:None}" {
		t.Fatalf("empty ToString = %q", got)
	}
	// The reference name order is A, B, X, Y, LeftShoulder, RightShoulder,
	// LeftStick, RightStick, Start, Back, BigButton.
	ordered := NewGamePadButtons(ButtonsBack | ButtonsA | ButtonsLeftStick | ButtonsY)
	if got := ordered.ToString(); got != "{Buttons:A Y LeftStick Back}" {
		t.Fatalf("ToString = %q, want {Buttons:A Y LeftStick Back}", got)
	}
	if !buttons.Equals(NewGamePadButtons(ButtonsA)) ||
		!GamePadButtonsOperatorInequalityByGamePadButtonsAndGamePadButtons(buttons, pair) {
		t.Fatal("GamePadButtons comparison is wrong")
	}
}

func TestMouseStateStorageHashAndString(t *testing.T) {
	state := NewMouseState(1, 2, 3,
		ButtonStatePressed, ButtonStateReleased, ButtonStatePressed,
		ButtonStateReleased, ButtonStatePressed)
	if state.X() != 1 || state.Y() != 2 || state.ScrollWheelValue() != 3 {
		t.Fatalf("position/wheel = %d/%d/%d", state.X(), state.Y(), state.ScrollWheelValue())
	}
	if state.LeftButton() != ButtonStatePressed || state.MiddleButton() != ButtonStateReleased ||
		state.RightButton() != ButtonStatePressed || state.XButton1() != ButtonStateReleased ||
		state.XButton2() != ButtonStatePressed {
		t.Fatal("button parameters were stored in the wrong order")
	}
	// MouseState does not use Helpers.SmartGetHashCode, so there is no
	// Int32.MaxValue substitution and the zero value hashes to zero.
	if got := state.GetHashCode(); got != 1 {
		t.Fatalf("GetHashCode = %d, want 1", got)
	}
	var zero MouseState
	if got := zero.GetHashCode(); got != 0 {
		t.Fatalf("zero GetHashCode = %d, want 0", got)
	}
	if got := state.ToString(); got != "{X:1 Y:2 Buttons:Left Right XButton2 Wheel:3}" {
		t.Fatalf("ToString = %q", got)
	}
	if got := zero.ToString(); got != "{X:0 Y:0 Buttons:None Wheel:0}" {
		t.Fatalf("zero ToString = %q", got)
	}
	if !state.Equals(NewMouseState(1, 2, 3, ButtonStatePressed, ButtonStateReleased, ButtonStatePressed, ButtonStateReleased, ButtonStatePressed)) {
		t.Fatal("equal mouse states did not compare equal")
	}
	moved := NewMouseState(1, 2, 4, ButtonStatePressed, ButtonStateReleased, ButtonStatePressed, ButtonStateReleased, ButtonStatePressed)
	if MouseStateOperatorEqualityByMouseStateAndMouseState(state, moved) ||
		!MouseStateOperatorInequalityByMouseStateAndMouseState(state, moved) {
		t.Fatal("wheel difference was not observed")
	}
	if state.Equals(nil) || state.Equals(int32(1)) {
		t.Fatal("Equals accepted a foreign type")
	}
}

// TestInputFormatSingleMatchesFrameworkFormatting keeps the input package's
// Single formatting identical to the framework package helper it mirrors.
func TestInputFormatSingleMatchesFrameworkFormatting(t *testing.T) {
	for _, value := range []float32{
		0, 1, -1, 0.25, 0.75, 1.5, -0.5, 1234567, 0.1,
		float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)),
		float32(math.Copysign(0, -1)),
	} {
		want := framework.Vector2{X: value}.ToString()
		got := "{X:" + formatSingle(value) + " Y:0}"
		if got != want {
			t.Fatalf("formatSingle(%v) = %q, framework renders %q", value, got, want)
		}
	}
}
