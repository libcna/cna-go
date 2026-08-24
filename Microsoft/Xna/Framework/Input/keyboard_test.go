package input

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func TestKeyboardStateConstructionAndQueries(t *testing.T) {
	state := NewKeyboardState([]Keys{KeysA, KeysSpace, KeysA, KeysChatPadOrange, Keys(-1), Keys(256)})
	wantPressed := []Keys{KeysSpace, KeysA, KeysChatPadOrange}
	if got := state.GetPressedKeys(); len(got) != len(wantPressed) {
		t.Fatalf("pressed keys = %v, want %v", got, wantPressed)
	} else {
		for index := range wantPressed {
			if got[index] != wantPressed[index] {
				t.Fatalf("pressed keys = %v, want %v", got, wantPressed)
			}
		}
	}
	for _, key := range wantPressed {
		if !state.IsKeyDown(key) || state.IsKeyUp(key) || state.Item(key) != KeyStateDown {
			t.Fatalf("pressed key %d was not down", key)
		}
	}
	for _, key := range []Keys{KeysB, Keys(-1), Keys(256)} {
		if state.IsKeyDown(key) || !state.IsKeyUp(key) || state.Item(key) != KeyStateUp {
			t.Fatalf("unpressed key %d was not up", key)
		}
	}
}

func TestKeyboardStateEqualityHashAndOperators(t *testing.T) {
	left := NewKeyboardState([]Keys{KeysA, KeysF24, KeysOemClear})
	right := NewKeyboardState([]Keys{KeysOemClear, KeysA, KeysF24})
	different := NewKeyboardState([]Keys{KeysA})

	if !left.Equals(right) || left.Equals(&right) || left.Equals(nil) || left.Equals(int32(0)) {
		t.Fatal("KeyboardState object equality failed")
	}
	if !KeyboardStateOperatorEqualityByKeyboardStateAndKeyboardState(left, right) ||
		KeyboardStateOperatorInequalityByKeyboardStateAndKeyboardState(left, right) ||
		KeyboardStateOperatorEqualityByKeyboardStateAndKeyboardState(left, different) ||
		!KeyboardStateOperatorInequalityByKeyboardStateAndKeyboardState(left, different) {
		t.Fatal("KeyboardState operators failed")
	}
	if left.GetHashCode() != right.GetHashCode() {
		t.Fatalf("equal hashes differ: %d and %d", left.GetHashCode(), right.GetHashCode())
	}
}

func TestKeyboardOverloadsShareNoActiveGameRequirementForEveryPlayerIndex(t *testing.T) {
	noneState, noneError := KeyboardGetStateByNone()
	if noneError == nil {
		t.Fatal("KeyboardGetStateByNone succeeded without an active Game")
	}

	values := []framework.PlayerIndex{
		framework.PlayerIndexOne,
		framework.PlayerIndexTwo,
		framework.PlayerIndexThree,
		framework.PlayerIndexFour,
		framework.PlayerIndex(12345),
	}
	for _, playerIndex := range values {
		state, err := KeyboardGetStateByPlayerIndex(playerIndex)
		if err == nil {
			t.Fatalf("KeyboardGetStateByPlayerIndex(%d) succeeded without an active Game", playerIndex)
		}
		if state != noneState || err.Error() != noneError.Error() {
			t.Fatalf("player %d used a different no-Game path: state=%#v error=%q, want state=%#v error=%q", playerIndex, state, err, noneState, noneError)
		}
	}
}
