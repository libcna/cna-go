//go:build linux && cgo

package interop

/*
#cgo LDFLAGS: -ldl -pthread
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"runtime/cgo"
	"unsafe"
)

// The Go game-event identities must equal the C ones exactly. These arrays are
// sized by the difference, so any drift between runtime.go's constants and
// bridge.h's mirror of CNA_GAME_EVENT_* is a compile error rather than a
// silently mis-routed signal.
var (
	_ [C.CNA_GO_GAME_EVENT_ACTIVATED - GameEventActivated]struct{}
	_ [C.CNA_GO_GAME_EVENT_DEACTIVATED - GameEventDeactivated]struct{}
	_ [C.CNA_GO_GAME_EVENT_DISPOSED - GameEventDisposed]struct{}
	_ [C.CNA_GO_GAME_EVENT_EXITING - GameEventExiting]struct{}
	_ [C.CNA_GO_GAME_EVENT_COUNT - gameEventCount]struct{}
)

// The optional frame-hook mask must equal bridge.h's mirror exactly. A drift
// would install the wrong CNA_GameFrameHooks member for a declared override --
// the same silent mis-routing the game-event assertions above exist to stop.
var (
	_ [C.CNA_GO_FRAME_HOOK_BEGIN_RUN - FrameHookBeginRun]struct{}
	_ [C.CNA_GO_FRAME_HOOK_END_RUN - FrameHookEndRun]struct{}
	_ [C.CNA_GO_FRAME_HOOK_BEGIN_DRAW - FrameHookBeginDraw]struct{}
	_ [C.CNA_GO_FRAME_HOOK_END_DRAW - FrameHookEndDraw]struct{}
	_ [C.CNA_GO_FRAME_HOOK_ALL - (FrameHookBeginRun | FrameHookEndRun | FrameHookBeginDraw | FrameHookEndDraw)]struct{}
)

func nativeOpen(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	buffer := make([]byte, 1024)
	if C.cna_go_open(cPath, (*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer))) == 0 {
		return fmt.Errorf("%w: %s", ErrNativeUnavailable, cStringFromBuffer(buffer))
	}
	version := uint32(C.cna_go_abi_version())
	if version != uint32(C.CNA_GO_ABI_VERSION) {
		C.cna_go_close()
		return fmt.Errorf("%w: CNA C ABI version 0x%08x is not admitted; require 0.7.0 (0x%08x)", ErrNativeUnavailable, version, uint32(C.CNA_GO_ABI_VERSION))
	}
	return nil
}

func nativeClose() { C.cna_go_close() }

func nativeABIVersion() uint32 { return uint32(C.cna_go_abi_version()) }

func nativeOwnerThreadID() uint64 { return uint64(C.cna_go_owner_thread_id()) }

func nativeBoundSymbols() []string {
	count := uint32(C.cna_go_bound_function_count())
	result := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		name := C.cna_go_bound_function_name(C.uint32_t(i))
		if name != nil {
			result = append(result, C.GoString(name))
		}
	}
	return result
}

func nativeHasLoadedSymbol(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return C.cna_go_has_loaded_symbol(cName) != 0
}

func nativeLastErrorMessage() string {
	required := int(C.cna_go_last_error_message(nil, 0))
	if required <= 0 {
		return ""
	}
	buffer := make([]byte, required)
	C.cna_go_last_error_message((*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer)))
	return string(buffer)
}

func nativeGameCreate(context uintptr, title string, frameHooks FrameHookMask) (uint64, error) {
	titleBytes := []byte(title)
	var titlePointer *C.char
	if len(titleBytes) > 0 {
		titlePointer = (*C.char)(unsafe.Pointer(&titleBytes[0]))
	}
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_game_create(C.uintptr_t(context), titlePointer, C.uint64_t(len(titleBytes)), C.uint32_t(frameHooks), &handle))
	return uint64(handle), resultError("cna_game_create/cna_game_set_frame_hooks_ext", code)
}

func nativeGameRun(game uint64) error {
	return resultError("cna_game_run", uint32(C.cna_go_game_run(C.CnaGoHandle(game))))
}

func nativeGameRequestExit(game uint64) error {
	return resultError("cna_game_request_exit", uint32(C.cna_go_game_request_exit(C.CnaGoHandle(game))))
}

func nativeGameDestroy(game uint64) error {
	return resultError("cna_game_destroy", uint32(C.cna_go_game_destroy(C.CnaGoHandle(game))))
}

// nativeGameSubscribeEvents installs exactly one native subscription per
// canonical game event and returns the four owned registration handles. CNA
// enforces owner-thread affinity on cna_game_subscribe itself: a call from any
// other thread reports CNA_RESULT_THREAD (8) and installs nothing.
func nativeGameSubscribeEvents(game uint64, context uintptr) ([gameEventCount]uint64, error) {
	var registrations [gameEventCount]C.CnaGoHandle
	code := uint32(C.cna_go_game_subscribe_events(C.CnaGoHandle(game), C.uintptr_t(context), &registrations[0]))
	var result [gameEventCount]uint64
	for i := range result {
		result[i] = uint64(registrations[i])
	}
	return result, resultError("cna_game_subscribe", code)
}

// nativeGameUnsubscribeEvents releases every non-zero registration and zeroes
// the caller's slot, so a second release cannot pass a stale handle back to
// CNA -- which answers CNA_RESULT_INVALID_HANDLE (2) for one, not success.
func nativeGameUnsubscribeEvents(registrations *[gameEventCount]uint64) error {
	var handles [gameEventCount]C.CnaGoHandle
	for i, value := range registrations {
		handles[i] = C.CnaGoHandle(value)
	}
	code := uint32(C.cna_go_game_unsubscribe_events(&handles[0]))
	for i := range registrations {
		registrations[i] = uint64(handles[i])
	}
	return resultError("cna_game_unsubscribe", code)
}

func nativeGraphicsDeviceManagerCreate(game uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_manager_create(C.CnaGoHandle(game), &handle))
	return uint64(handle), resultError("cna_graphics_device_manager_create", code)
}

func nativeGraphicsDeviceManagerGetDevice(manager uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_manager_get_device(C.CnaGoHandle(manager), &handle))
	return uint64(handle), resultError("cna_graphics_device_manager_get_graphics_device", code)
}

func nativeGraphicsDeviceManagerDestroy(manager uint64) error {
	return resultError("cna_graphics_device_manager_destroy", uint32(C.cna_go_graphics_device_manager_destroy(C.CnaGoHandle(manager))))
}

func nativeGameGetGraphicsDevice(game uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_game_get_graphics_device(C.CnaGoHandle(game), &handle))
	return uint64(handle), resultError("cna_game_get_graphics_device", code)
}

func nativeGraphicsDeviceViewport(device uint64) (Viewport, error) {
	var x, y, width, height C.int32_t
	var minDepth, maxDepth C.float
	code := uint32(C.cna_go_graphics_device_get_viewport(C.CnaGoHandle(device), &x, &y, &width, &height, &minDepth, &maxDepth))
	return Viewport{X: int32(x), Y: int32(y), Width: int32(width), Height: int32(height), MinDepth: float32(minDepth), MaxDepth: float32(maxDepth)}, resultError("cna_graphics_device_get_viewport", code)
}

func nativeGraphicsDeviceClear(device uint64, red, green, blue, alpha float32) error {
	return resultError("cna_graphics_device_clear_rgba", uint32(C.cna_go_graphics_device_clear_rgba(C.CnaGoHandle(device), C.float(red), C.float(green), C.float(blue), C.float(alpha))))
}

func nativeTextureCreateEncoded(device uint64, data []byte) (uint64, error) {
	var pointer *C.uint8_t
	if len(data) > 0 {
		pointer = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_texture2d_create_from_encoded_memory(C.CnaGoHandle(device), pointer, C.uint64_t(len(data)), &handle))
	return uint64(handle), resultError("cna_texture2d_create_from_encoded_memory", code)
}

func nativeTextureInfo(texture uint64) (TextureInfo, error) {
	var width, height, levels, format C.uint32_t
	code := uint32(C.cna_go_texture2d_get_info(C.CnaGoHandle(texture), &width, &height, &levels, &format))
	return TextureInfo{Width: uint32(width), Height: uint32(height), Levels: uint32(levels), Format: uint32(format)}, resultError("cna_texture2d_get_info", code)
}

func nativeTextureDestroy(texture uint64) error {
	return resultError("cna_texture2d_destroy", uint32(C.cna_go_texture2d_destroy(C.CnaGoHandle(texture))))
}

func nativeSpriteBatchCreate(device uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_sprite_batch_create(C.CnaGoHandle(device), &handle))
	return uint64(handle), resultError("cna_sprite_batch_create", code)
}

func nativeSpriteBatchBegin(batch uint64) error {
	return resultError("cna_sprite_batch_begin", uint32(C.cna_go_sprite_batch_begin(C.CnaGoHandle(batch))))
}

func nativeSpriteBatchDrawScaled(batch, texture uint64, command SpriteCommand) error {
	return resultError("cna_sprite_batch_submit_scaled_many", uint32(C.cna_go_sprite_batch_draw_scaled(
		C.CnaGoHandle(batch), C.CnaGoHandle(texture),
		C.float(command.PositionX), C.float(command.PositionY),
		C.int32_t(command.SourceX), C.int32_t(command.SourceY), C.int32_t(command.SourceWidth), C.int32_t(command.SourceHeight),
		C.uint8_t(command.Red), C.uint8_t(command.Green), C.uint8_t(command.Blue), C.uint8_t(command.Alpha),
		C.float(command.Rotation), C.float(command.OriginX), C.float(command.OriginY),
		C.float(command.ScaleX), C.float(command.ScaleY), C.uint32_t(command.Effects), C.float(command.LayerDepth))))
}

func nativeSpriteBatchEnd(batch uint64) error {
	return resultError("cna_sprite_batch_end", uint32(C.cna_go_sprite_batch_end(C.CnaGoHandle(batch))))
}

func nativeSpriteBatchDestroy(batch uint64) error {
	return resultError("cna_sprite_batch_destroy", uint32(C.cna_go_sprite_batch_destroy(C.CnaGoHandle(batch))))
}

func nativeKeyboardState(game uint64) ([4]uint64, error) {
	var result [4]uint64
	var words [4]C.uint64_t
	code := uint32(C.cna_go_keyboard_get_state(C.CnaGoHandle(game), &words[0], &words[1], &words[2], &words[3]))
	for i := range result {
		result[i] = uint64(words[i])
	}
	return result, resultError("cna_keyboard_get_state", code)
}

func cStringFromBuffer(buffer []byte) string {
	for i, value := range buffer {
		if value == 0 {
			return string(buffer[:i])
		}
	}
	return string(buffer)
}

//export cnaGoGameEvent
func cnaGoGameEvent(event C.uint32_t, context C.uintptr_t) {
	// A CNA_GameEventCallback returns void, so nothing may cross the C frame:
	// neither a Go panic nor an error. Both are recorded on the Runtime and
	// surface from Run instead.
	var state *Runtime
	defer func() {
		if recovered := recover(); recovered != nil && state != nil {
			state.recordCallbackFailure(fmt.Errorf("panic in native game-event trampoline: %v", recovered))
		}
	}()
	handle := cgo.Handle(context)
	state = handle.Value().(*Runtime)
	state.invokeGameEvent(uint32(event))
}

// cnaGoBeginDraw is the begin_draw trampoline. It is separate from
// cnaGoLifecycle because CNA_GameBeginDrawCallback carries an out-parameter,
// and that Boolean is a value channel rather than a second success flag: the
// slot is written only on success, and a refused frame is (0, success).
//
//export cnaGoBeginDraw
func cnaGoBeginDraw(game C.uint64_t, totalTicks C.int64_t, elapsedTicks C.int64_t, runningSlowly C.uint8_t, context C.uintptr_t, outShouldDraw *C.uint8_t) (result C.uint32_t) {
	var state *Runtime
	defer func() {
		if recovered := recover(); recovered != nil {
			if state != nil {
				state.recordCallbackFailure(fmt.Errorf("panic in native begin_draw trampoline: %v", recovered))
			}
			result = C.uint32_t(resultCallback)
		}
	}()
	handle := cgo.Handle(context)
	state = handle.Value().(*Runtime)
	shouldDraw, err := state.invokeBeginDrawCallback(uint64(game), FrameTime{TotalTicks: int64(totalTicks), ElapsedTicks: int64(elapsedTicks), IsRunningSlowly: runningSlowly != 0})
	if err != nil {
		return C.uint32_t(resultCallback)
	}
	if outShouldDraw != nil {
		if shouldDraw {
			*outShouldDraw = 1
		} else {
			*outShouldDraw = 0
		}
	}
	return C.uint32_t(resultSuccess)
}

//export cnaGoLifecycle
func cnaGoLifecycle(kind C.uint32_t, game C.uint64_t, totalTicks C.int64_t, elapsedTicks C.int64_t, runningSlowly C.uint8_t, context C.uintptr_t) (result C.uint32_t) {
	var state *Runtime
	defer func() {
		if recovered := recover(); recovered != nil {
			if state != nil {
				state.recordCallbackFailure(fmt.Errorf("panic in native callback trampoline: %v", recovered))
			}
			result = C.uint32_t(resultCallback)
		}
	}()
	handle := cgo.Handle(context)
	state = handle.Value().(*Runtime)
	err := state.invokeCallback(uint32(kind), uint64(game), FrameTime{TotalTicks: int64(totalTicks), ElapsedTicks: int64(elapsedTicks), IsRunningSlowly: runningSlowly != 0})
	if err != nil {
		return C.uint32_t(resultCallback)
	}
	return C.uint32_t(resultSuccess)
}
