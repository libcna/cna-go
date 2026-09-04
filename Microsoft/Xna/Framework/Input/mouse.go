package input

import (
	"errors"

	"github.com/openeggbert/cna-go/internal/interop"
)

// Mouse is Microsoft.Xna.Framework.Input.Mouse:
//
//	.class public abstract auto ansi sealed beforefieldinit Mouse
//
// A static class, so its three members are type-prefixed package functions and
// the identity is an empty struct.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # The reference body is Win32, so CNA is the behaviour authority here
//
//	get_WindowHandle  ldsfld hHookedHandle; IntPtr(void*)
//	set_WindowHandle  stores the handle AND re-targets the MouseMessageHooker
//	GetState          GetCursorPos, ScreenToClient, GetAsyncKeyState x5
//
// There is no managed contract in that body to reproduce beyond its SHAPE:
// the cursor position is reported in the hooked window's CLIENT coordinates,
// and the button values come from the OS rather than from a message queue the
// projection owns. Reproducing `GetAsyncKeyState` on Linux would be
// reproducing a platform call, not a behaviour, so the corresponding CNA
// routes answer instead and their results are projected unchanged.
//
// # The scroll wheel is cumulative, not per-frame
//
// Both sides report a running total in 120-unit notches, so a consumer
// subtracts the previous frame's value. The projection does not reset it.
//
// # CNA reports a horizontal wheel that XNA does not declare
//
// `CNA_MouseState.horizontal_scroll_wheel` has no XNA counterpart: XNA 4.0's
// MouseState declares X, Y, ScrollWheelValue and five buttons and nothing
// else. It is read across the bridge with the rest of the structure and then
// DROPPED, because projecting it would add public surface the contract does
// not declare.
type Mouse struct{}

// errMouseNoRunningGame is the Go-only refusal these statics can answer. The
// reference reaches the OS directly; CNA's mouse routes read the window's own
// event state, which exists only inside a lifecycle callback.
var errMouseNoRunningGame = errors.New("this member needs a running game")

// MouseGetState is Mouse::GetState().
func MouseGetState() (MouseState, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return MouseState{}, errMouseNoRunningGame
	}
	values, err := runtime.MouseState()
	if err != nil {
		return MouseState{}, err
	}
	left, middle, right, x1, x2 := mouseButtonStates(values.PressedButtons)
	return NewMouseState(values.X, values.Y, values.ScrollWheel,
		left, middle, right, x1, x2), nil
}

// MouseSetPosition is Mouse::SetPosition(Int32, Int32), which
// takes CLIENT coordinates of the hooked window -- the same space GetState
// reports in, not screen space.
func MouseSetPosition(x, y int32) error {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errMouseNoRunningGame
	}
	return runtime.MouseSetPosition(x, y)
}

// MouseWindowHandle is get_WindowHandle. System.IntPtr projects to uintptr,
// which the mapping already declares, so no raw pointer crosses the surface.
func MouseWindowHandle() (uintptr, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errMouseNoRunningGame
	}
	handle, err := runtime.MouseWindowHandle()
	if err != nil {
		return 0, err
	}
	return uintptr(handle), nil
}

// SetMouseWindowHandle is set_WindowHandle, which re-targets which window's
// client area GetState and SetPosition are relative to.
func SetMouseWindowHandle(value uintptr) error {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errMouseNoRunningGame
	}
	return runtime.MouseSetWindowHandle(uint64(value))
}

// The CNA_MOUSE_BUTTON_* bits, mirrored from CNA/C/input.h. They are private:
// XNA declares no mouse button flags enum, and the projection reports each
// button as its own ButtonState property, exactly as MouseState does.
const (
	cnaMouseButtonLeft   uint32 = 1 << 0
	cnaMouseButtonMiddle uint32 = 1 << 1
	cnaMouseButtonRight  uint32 = 1 << 2
	cnaMouseButtonX1     uint32 = 1 << 3
	cnaMouseButtonX2     uint32 = 1 << 4
)

// mouseButtonStates expands one pressed-button mask into the five ButtonStates
// MouseState's constructor takes, in its declared parameter order.
//
// It exists as a named function rather than five inline calls so that the
// WIRING is testable without a game: a swapped pair here would report the right
// button as the middle one, and that is exactly the kind of defect no managed
// test could reach while the expansion lived inside MouseGetState.
func mouseButtonStates(mask uint32) (left, middle, right, x1, x2 ButtonState) {
	return mouseButtonState(mask, cnaMouseButtonLeft),
		mouseButtonState(mask, cnaMouseButtonMiddle),
		mouseButtonState(mask, cnaMouseButtonRight),
		mouseButtonState(mask, cnaMouseButtonX1),
		mouseButtonState(mask, cnaMouseButtonX2)
}

// mouseButtonState maps one pressed bit onto the ButtonState MouseState holds.
func mouseButtonState(mask, bit uint32) ButtonState {
	if mask&bit != 0 {
		return ButtonStatePressed
	}
	return ButtonStateReleased
}
