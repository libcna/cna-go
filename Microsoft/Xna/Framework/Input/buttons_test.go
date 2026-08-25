package input

import (
	"reflect"
	"testing"
)

func TestButtonsCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value Buttons
		want  int32
	}{
		{"DPadUp", ButtonsDPadUp, 1},
		{"DPadDown", ButtonsDPadDown, 2},
		{"DPadLeft", ButtonsDPadLeft, 4},
		{"DPadRight", ButtonsDPadRight, 8},
		{"Start", ButtonsStart, 16},
		{"Back", ButtonsBack, 32},
		{"LeftStick", ButtonsLeftStick, 64},
		{"RightStick", ButtonsRightStick, 128},
		{"LeftShoulder", ButtonsLeftShoulder, 256},
		{"RightShoulder", ButtonsRightShoulder, 512},
		{"BigButton", ButtonsBigButton, 2048},
		{"A", ButtonsA, 4096},
		{"B", ButtonsB, 8192},
		{"X", ButtonsX, 16384},
		{"Y", ButtonsY, 32768},
		{"RightThumbstickUp", ButtonsRightThumbstickUp, 16777216},
		{"RightThumbstickDown", ButtonsRightThumbstickDown, 33554432},
		{"RightThumbstickRight", ButtonsRightThumbstickRight, 67108864},
		{"RightThumbstickLeft", ButtonsRightThumbstickLeft, 134217728},
		{"LeftThumbstickUp", ButtonsLeftThumbstickUp, 268435456},
		{"LeftThumbstickDown", ButtonsLeftThumbstickDown, 536870912},
		{"LeftThumbstickRight", ButtonsLeftThumbstickRight, 1073741824},
		{"LeftThumbstickLeft", ButtonsLeftThumbstickLeft, 2097152},
		{"LeftTrigger", ButtonsLeftTrigger, 8388608},
		{"RightTrigger", ButtonsRightTrigger, 4194304},
	}
	if len(values) != 25 {
		t.Fatalf("Buttons literal count = %d, want 25", len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item.name] {
			t.Fatalf("Buttons%s appears twice in the pinned table", item.name)
		}
		seen[item.name] = true
		if got := int32(item.value); got != item.want {
			t.Errorf("Buttons%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(ButtonsDPadUp).Kind(); got != reflect.Int32 {
		t.Fatalf("Buttons underlying kind = %s, want int32", got)
	}
}

func TestButtonsZeroAndArbitraryRawValues(t *testing.T) {
	var zero Buttons
	// The pinned contract declares no zero literal for this enum, so the Go
	// zero value is an ordinary undefined raw value, not a named constant.
	if int32(zero) != 0 {
		t.Fatalf("zero Buttons = %d, want 0", int32(zero))
	}
	if zero == ButtonsDPadUp {
		t.Fatal("zero Buttons unexpectedly equals DPadUp")
	}
	if zero == ButtonsDPadDown {
		t.Fatal("zero Buttons unexpectedly equals DPadDown")
	}
	if zero == ButtonsDPadLeft {
		t.Fatal("zero Buttons unexpectedly equals DPadLeft")
	}
	if zero == ButtonsDPadRight {
		t.Fatal("zero Buttons unexpectedly equals DPadRight")
	}
	if zero == ButtonsStart {
		t.Fatal("zero Buttons unexpectedly equals Start")
	}
	if zero == ButtonsBack {
		t.Fatal("zero Buttons unexpectedly equals Back")
	}
	if zero == ButtonsLeftStick {
		t.Fatal("zero Buttons unexpectedly equals LeftStick")
	}
	if zero == ButtonsRightStick {
		t.Fatal("zero Buttons unexpectedly equals RightStick")
	}
	if zero == ButtonsLeftShoulder {
		t.Fatal("zero Buttons unexpectedly equals LeftShoulder")
	}
	if zero == ButtonsRightShoulder {
		t.Fatal("zero Buttons unexpectedly equals RightShoulder")
	}
	if zero == ButtonsBigButton {
		t.Fatal("zero Buttons unexpectedly equals BigButton")
	}
	if zero == ButtonsA {
		t.Fatal("zero Buttons unexpectedly equals A")
	}
	if zero == ButtonsB {
		t.Fatal("zero Buttons unexpectedly equals B")
	}
	if zero == ButtonsX {
		t.Fatal("zero Buttons unexpectedly equals X")
	}
	if zero == ButtonsY {
		t.Fatal("zero Buttons unexpectedly equals Y")
	}
	if zero == ButtonsRightThumbstickUp {
		t.Fatal("zero Buttons unexpectedly equals RightThumbstickUp")
	}
	if zero == ButtonsRightThumbstickDown {
		t.Fatal("zero Buttons unexpectedly equals RightThumbstickDown")
	}
	if zero == ButtonsRightThumbstickRight {
		t.Fatal("zero Buttons unexpectedly equals RightThumbstickRight")
	}
	if zero == ButtonsRightThumbstickLeft {
		t.Fatal("zero Buttons unexpectedly equals RightThumbstickLeft")
	}
	if zero == ButtonsLeftThumbstickUp {
		t.Fatal("zero Buttons unexpectedly equals LeftThumbstickUp")
	}
	if zero == ButtonsLeftThumbstickDown {
		t.Fatal("zero Buttons unexpectedly equals LeftThumbstickDown")
	}
	if zero == ButtonsLeftThumbstickRight {
		t.Fatal("zero Buttons unexpectedly equals LeftThumbstickRight")
	}
	if zero == ButtonsLeftThumbstickLeft {
		t.Fatal("zero Buttons unexpectedly equals LeftThumbstickLeft")
	}
	if zero == ButtonsLeftTrigger {
		t.Fatal("zero Buttons unexpectedly equals LeftTrigger")
	}
	if zero == ButtonsRightTrigger {
		t.Fatal("zero Buttons unexpectedly equals RightTrigger")
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(Buttons(raw)); got != raw {
			t.Fatalf("Buttons(%d) = %d", raw, got)
		}
	}
}

func TestButtonsSourceFlagsDirective(t *testing.T) {
	if got := flagsDirectiveAt(t, "buttons.go", "Buttons"); got != true {
		t.Fatalf("Buttons xna:flags directive = %t, want true", got)
	}
}
