package input

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The Input statics reach CNA, so outside a lifecycle callback every one of
// them answers the no-running-game refusal rather than a value. That is the
// only branch a package test can reach; the native behaviour is covered by the
// stress slice, which runs inside a real game.

// TestGamePadStaticsRefuseWithoutAGame covers all four GamePad entry points,
// including the one-argument GetState overload, which must reach the same
// guard through its forwarding body rather than answering a zero value.
func TestGamePadStaticsRefuseWithoutAGame(t *testing.T) {
	if _, err := GamePadGetStateByPlayerIndex(framework.PlayerIndexOne); !errors.Is(err, errGamePadNoRunningGame) {
		t.Fatalf("GetState(PlayerIndex): err = %v, want the no-running-game refusal", err)
	}
	if _, err := GamePadGetStateByPlayerIndexAndGamePadDeadZone(
		framework.PlayerIndexTwo, GamePadDeadZoneCircular); !errors.Is(err, errGamePadNoRunningGame) {
		t.Fatalf("GetState(PlayerIndex,GamePadDeadZone): err = %v, want the no-running-game refusal", err)
	}
	if _, err := GamePadGetCapabilities(framework.PlayerIndexThree); !errors.Is(err, errGamePadNoRunningGame) {
		t.Fatalf("GetCapabilities: err = %v, want the no-running-game refusal", err)
	}
	if _, err := GamePadSetVibration(framework.PlayerIndexFour, 0.5, 0.5); !errors.Is(err, errGamePadNoRunningGame) {
		t.Fatalf("SetVibration: err = %v, want the no-running-game refusal", err)
	}
}

// TestMouseStaticsRefuseWithoutAGame covers the three Mouse members.
func TestMouseStaticsRefuseWithoutAGame(t *testing.T) {
	if _, err := MouseGetState(); !errors.Is(err, errMouseNoRunningGame) {
		t.Fatalf("GetState: err = %v, want the no-running-game refusal", err)
	}
	if err := MouseSetPosition(10, 20); !errors.Is(err, errMouseNoRunningGame) {
		t.Fatalf("SetPosition: err = %v, want the no-running-game refusal", err)
	}
	if _, err := MouseWindowHandle(); !errors.Is(err, errMouseNoRunningGame) {
		t.Fatalf("get_WindowHandle: err = %v, want the no-running-game refusal", err)
	}
	if err := SetMouseWindowHandle(1); !errors.Is(err, errMouseNoRunningGame) {
		t.Fatalf("set_WindowHandle: err = %v, want the no-running-game refusal", err)
	}
}

// TestGamePadOneArgumentGetStateForwardsIndependentAxes pins the `ldc.i4.1`
// in the one-argument overload's two-instruction body. Without a game neither
// call reaches native, so what the test can prove is that the forwarding
// happens at all -- the overload is not a separate implementation that could
// drift from the two-argument one.
func TestGamePadOneArgumentGetStateForwardsIndependentAxes(t *testing.T) {
	if GamePadDeadZoneIndependentAxes != 1 {
		t.Fatalf("GamePadDeadZone.IndependentAxes = %d, want 1, which is the literal the reference forwards",
			GamePadDeadZoneIndependentAxes)
	}
	one, errOne := GamePadGetStateByPlayerIndex(framework.PlayerIndexOne)
	two, errTwo := GamePadGetStateByPlayerIndexAndGamePadDeadZone(
		framework.PlayerIndexOne, GamePadDeadZoneIndependentAxes)
	if !errors.Is(errOne, errGamePadNoRunningGame) || !errors.Is(errTwo, errGamePadNoRunningGame) {
		t.Fatalf("both overloads should refuse: %v / %v", errOne, errTwo)
	}
	if one != two {
		t.Fatal("the two overloads answered differently for the dead zone the shorter one forwards")
	}
}

// TestGamePadCapabilityFlagOrder pins the index constants against the order the
// contract declares the properties in. The C side fills a flat byte array in
// exactly this order, so a disagreement here is what would silently swap two
// booleans -- HasAButton reporting the state of HasBackButton, say.
func TestGamePadCapabilityFlagOrder(t *testing.T) {
	if gamePadCapabilityFlagCount != 25 {
		t.Fatalf("flag count = %d, want 25: IsConnected plus the twenty-four booleans XNA declares",
			gamePadCapabilityFlagCount)
	}
	if gamePadCapabilityIsConnected != 0 {
		t.Fatalf("IsConnected index = %d, want 0", gamePadCapabilityIsConnected)
	}
	if gamePadCapabilityHasVoiceSupport != gamePadCapabilityFlagCount-1 {
		t.Fatalf("HasVoiceSupport index = %d, want the last slot %d",
			gamePadCapabilityHasVoiceSupport, gamePadCapabilityFlagCount-1)
	}
	// Every index distinct and inside the array: a duplicated constant would
	// make two properties read one slot and leave another slot unread.
	seen := map[int]string{}
	for index, name := range gamePadCapabilityNames {
		if index < 0 || index >= gamePadCapabilityFlagCount {
			t.Fatalf("%s has index %d, outside the array", name, index)
		}
		if other, duplicate := seen[index]; duplicate {
			t.Fatalf("%s and %s share index %d", name, other, index)
		}
		seen[index] = name
	}
	if len(seen) != gamePadCapabilityFlagCount {
		t.Fatalf("%d distinct indices for %d slots", len(seen), gamePadCapabilityFlagCount)
	}
}

// TestGamePadCapabilitiesReadTheirOwnSlot walks all twenty-five accessors with
// exactly one flag raised and checks that only that accessor answers true.
// This is the test a swapped index cannot pass.
func TestGamePadCapabilitiesReadTheirOwnSlot(t *testing.T) {
	readers := gamePadCapabilityReaders()
	if len(readers) != gamePadCapabilityFlagCount {
		t.Fatalf("%d readers for %d slots", len(readers), gamePadCapabilityFlagCount)
	}
	for raised := 0; raised < gamePadCapabilityFlagCount; raised++ {
		var flags [gamePadCapabilityFlagCount]bool
		flags[raised] = true
		capabilities := newGamePadCapabilities(GamePadTypeGamePad, flags)
		for index, read := range readers {
			if got := read(capabilities); got != (index == raised) {
				t.Fatalf("with only slot %d raised, reader %d answered %v",
					raised, index, got)
			}
		}
	}
}

// TestGamePadCapabilitiesZeroValueIsTheDisconnectedAnswer pins that the empty
// value the ERROR_DEVICE_NOT_CONNECTED branch returns needs no construction.
func TestGamePadCapabilitiesZeroValueIsTheDisconnectedAnswer(t *testing.T) {
	var empty GamePadCapabilities
	if empty.IsConnected() {
		t.Fatal("the zero value reported a connected controller")
	}
	if got := empty.GamePadType(); got != GamePadTypeUnknown {
		t.Fatalf("zero-value GamePadType = %d, want Unknown (0)", got)
	}
	for index, read := range gamePadCapabilityReaders() {
		if read(empty) {
			t.Fatalf("zero-value reader %d answered true", index)
		}
	}
}

// TestMouseButtonMaskMapsEachBitOnce walks the five CNA_MOUSE_BUTTON_* bits
// through the same expansion MouseGetState uses. A swapped pair would report
// the right button as the middle one, and this is the test that sees it.
func TestMouseButtonMaskMapsEachBitOnce(t *testing.T) {
	bits := []uint32{
		cnaMouseButtonLeft, cnaMouseButtonMiddle, cnaMouseButtonRight,
		cnaMouseButtonX1, cnaMouseButtonX2,
	}
	readers := []func(MouseState) ButtonState{
		MouseState.LeftButton, MouseState.MiddleButton, MouseState.RightButton,
		MouseState.XButton1, MouseState.XButton2,
	}
	// The five bits must be distinct: two that collide would make one button
	// answer for another with no other symptom.
	var union uint32
	for _, bit := range bits {
		if bit == 0 {
			t.Fatal("a mouse button bit is zero")
		}
		if union&bit != 0 {
			t.Fatalf("bit %#x collides with an earlier one", bit)
		}
		union |= bit
	}
	for raised, bit := range bits {
		left, middle, right, x1, x2 := mouseButtonStates(bit)
		state := NewMouseState(0, 0, 0, left, middle, right, x1, x2)
		for index, read := range readers {
			want := ButtonStateReleased
			if index == raised {
				want = ButtonStatePressed
			}
			if got := read(state); got != want {
				t.Fatalf("bit %d raised: button %d = %v, want %v", raised, index, got, want)
			}
		}
	}
	// An empty mask releases everything, and a full one presses everything.
	left, middle, right, x1, x2 := mouseButtonStates(0)
	for index, got := range []ButtonState{left, middle, right, x1, x2} {
		if got != ButtonStateReleased {
			t.Fatalf("empty mask: button %d = %v", index, got)
		}
	}
	left, middle, right, x1, x2 = mouseButtonStates(union)
	for index, got := range []ButtonState{left, middle, right, x1, x2} {
		if got != ButtonStatePressed {
			t.Fatalf("full mask: button %d = %v", index, got)
		}
	}
}

// TestButtonsFromMaskExpandsEveryDeclaredLiteral walks the Buttons literals one
// at a time. A dropped entry is invisible to a round-trip check over a full
// mask -- the survivors still OR back together -- so each literal is expanded
// alone and must come back alone.
func TestButtonsFromMaskExpandsEveryDeclaredLiteral(t *testing.T) {
	// Written out longhand: a list derived from the same table the function
	// uses would agree with it by construction.
	declared := []Buttons{
		ButtonsDPadUp, ButtonsDPadDown, ButtonsDPadLeft, ButtonsDPadRight,
		ButtonsStart, ButtonsBack, ButtonsLeftStick, ButtonsRightStick,
		ButtonsLeftShoulder, ButtonsRightShoulder, ButtonsBigButton,
		ButtonsA, ButtonsB, ButtonsX, ButtonsY,
		ButtonsLeftThumbstickLeft, ButtonsLeftThumbstickRight,
		ButtonsLeftThumbstickUp, ButtonsLeftThumbstickDown,
		ButtonsRightThumbstickLeft, ButtonsRightThumbstickRight,
		ButtonsRightThumbstickUp, ButtonsRightThumbstickDown,
		ButtonsLeftTrigger, ButtonsRightTrigger,
	}
	for _, button := range declared {
		got := buttonsFromMask(uint32(button))
		if len(got) != 1 || got[0] != button {
			t.Fatalf("buttonsFromMask(%d) = %v, want exactly [%d]", button, got, button)
		}
	}
	// The full expansion must carry every one of them and nothing with a zero
	// value.
	all := buttonsFromMask(0xffffffff)
	if len(all) != len(declared) {
		t.Fatalf("a full mask expanded to %d buttons, want the %d declared literals",
			len(all), len(declared))
	}
	present := map[Buttons]bool{}
	for _, button := range all {
		if button == 0 {
			t.Fatal("the expansion produced Buttons(0), which names no button")
		}
		if present[button] {
			t.Fatalf("%d appears twice in the expansion", button)
		}
		present[button] = true
	}
	for _, button := range declared {
		if !present[button] {
			t.Fatalf("%d was dropped by the expansion", button)
		}
	}
	if len(buttonsFromMask(0)) != 0 {
		t.Fatal("an empty mask expanded to a non-empty list")
	}
}

// gamePadCapabilityNames pairs each index constant with the accessor that must
// read it. It is written out longhand on purpose: a generated table would be
// derived from the same constants it is meant to check.
var gamePadCapabilityNames = map[int]string{
	gamePadCapabilityIsConnected:            "IsConnected",
	gamePadCapabilityHasAButton:             "HasAButton",
	gamePadCapabilityHasBButton:             "HasBButton",
	gamePadCapabilityHasXButton:             "HasXButton",
	gamePadCapabilityHasYButton:             "HasYButton",
	gamePadCapabilityHasBackButton:          "HasBackButton",
	gamePadCapabilityHasStartButton:         "HasStartButton",
	gamePadCapabilityHasBigButton:           "HasBigButton",
	gamePadCapabilityHasDPadUpButton:        "HasDPadUpButton",
	gamePadCapabilityHasDPadDownButton:      "HasDPadDownButton",
	gamePadCapabilityHasDPadLeftButton:      "HasDPadLeftButton",
	gamePadCapabilityHasDPadRightButton:     "HasDPadRightButton",
	gamePadCapabilityHasLeftShoulderButton:  "HasLeftShoulderButton",
	gamePadCapabilityHasRightShoulderButton: "HasRightShoulderButton",
	gamePadCapabilityHasLeftStickButton:     "HasLeftStickButton",
	gamePadCapabilityHasRightStickButton:    "HasRightStickButton",
	gamePadCapabilityHasLeftXThumbStick:     "HasLeftXThumbStick",
	gamePadCapabilityHasLeftYThumbStick:     "HasLeftYThumbStick",
	gamePadCapabilityHasRightXThumbStick:    "HasRightXThumbStick",
	gamePadCapabilityHasRightYThumbStick:    "HasRightYThumbStick",
	gamePadCapabilityHasLeftTrigger:         "HasLeftTrigger",
	gamePadCapabilityHasRightTrigger:        "HasRightTrigger",
	gamePadCapabilityHasLeftVibrationMotor:  "HasLeftVibrationMotor",
	gamePadCapabilityHasRightVibrationMotor: "HasRightVibrationMotor",
	gamePadCapabilityHasVoiceSupport:        "HasVoiceSupport",
}

// gamePadCapabilityReaders lists the twenty-five accessors in the order the
// contract declares them, which is the order the C side fills the byte array.
// Reader i must answer slot i and nothing else.
func gamePadCapabilityReaders() []func(GamePadCapabilities) bool {
	return []func(GamePadCapabilities) bool{
		GamePadCapabilities.IsConnected,
		GamePadCapabilities.HasAButton,
		GamePadCapabilities.HasBButton,
		GamePadCapabilities.HasXButton,
		GamePadCapabilities.HasYButton,
		GamePadCapabilities.HasBackButton,
		GamePadCapabilities.HasStartButton,
		GamePadCapabilities.HasBigButton,
		GamePadCapabilities.HasDPadUpButton,
		GamePadCapabilities.HasDPadDownButton,
		GamePadCapabilities.HasDPadLeftButton,
		GamePadCapabilities.HasDPadRightButton,
		GamePadCapabilities.HasLeftShoulderButton,
		GamePadCapabilities.HasRightShoulderButton,
		GamePadCapabilities.HasLeftStickButton,
		GamePadCapabilities.HasRightStickButton,
		GamePadCapabilities.HasLeftXThumbStick,
		GamePadCapabilities.HasLeftYThumbStick,
		GamePadCapabilities.HasRightXThumbStick,
		GamePadCapabilities.HasRightYThumbStick,
		GamePadCapabilities.HasLeftTrigger,
		GamePadCapabilities.HasRightTrigger,
		GamePadCapabilities.HasLeftVibrationMotor,
		GamePadCapabilities.HasRightVibrationMotor,
		GamePadCapabilities.HasVoiceSupport,
	}
}
