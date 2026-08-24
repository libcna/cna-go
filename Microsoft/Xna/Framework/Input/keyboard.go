package input

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// KeyState is the XNA up/down value returned by the KeyboardState indexer.
type KeyState int32

const (
	KeyStateUp   KeyState = 0
	KeyStateDown KeyState = 1
)

// Keyboard is the XNA static-class identity. Its operations are type-prefixed
// package functions.
type Keyboard struct{}

// KeyboardState is a copyable immutable 256-key snapshot.
type KeyboardState struct {
	pressed [4]uint64
}

func NewKeyboardState(keys []Keys) KeyboardState {
	var state KeyboardState
	for _, key := range keys {
		value := int32(key)
		if value < 0 || value >= 256 {
			continue
		}
		state.pressed[value/64] |= uint64(1) << uint32(value%64)
	}
	return state
}

func KeyboardGetStateByNone() (KeyboardState, error) {
	return keyboardGetState()
}

func KeyboardGetStateByPlayerIndex(_ framework.PlayerIndex) (KeyboardState, error) {
	return keyboardGetState()
}

func keyboardGetState() (KeyboardState, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return KeyboardState{}, errors.New("KeyboardGetState requires an active Game")
	}
	words, err := runtime.KeyboardState()
	if err != nil {
		return KeyboardState{}, err
	}
	return KeyboardState{pressed: words}, nil
}

func (s KeyboardState) IsKeyDown(key Keys) bool {
	value := int32(key)
	if value < 0 || value >= 256 {
		return false
	}
	return s.pressed[value/64]&(uint64(1)<<uint32(value%64)) != 0
}

func (s KeyboardState) IsKeyUp(key Keys) bool {
	return !s.IsKeyDown(key)
}

func (s KeyboardState) GetPressedKeys() []Keys {
	result := make([]Keys, 0)
	for key := int32(0); key < 256; key++ {
		if s.IsKeyDown(Keys(key)) {
			result = append(result, Keys(key))
		}
	}
	return result
}

func (s KeyboardState) GetHashCode() int32 {
	value := uint32(0)
	for _, word := range s.pressed {
		value ^= uint32(word)
		value ^= uint32(word >> 32)
	}
	return int32(value)
}

func (s KeyboardState) Equals(value any) bool {
	other, ok := value.(KeyboardState)
	return ok && s == other
}

func KeyboardStateOperatorEqualityByKeyboardStateAndKeyboardState(left, right KeyboardState) bool {
	return left == right
}

func KeyboardStateOperatorInequalityByKeyboardStateAndKeyboardState(left, right KeyboardState) bool {
	return left != right
}

func (s KeyboardState) Item(key Keys) KeyState {
	if s.IsKeyDown(key) {
		return KeyStateDown
	}
	return KeyStateUp
}
