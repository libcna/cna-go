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
	"runtime"
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
	if C.cna_go_abi_admits(C.uint32_t(version)) == 0 {
		C.cna_go_close()
		return fmt.Errorf(
			"%w: %s reports CNA C ABI %s (0x%08x); CNA-Go admits %s",
			ErrNativeUnavailable, path, FormatABIVersion(version), version, ABIAdmissionPolicy())
	}
	return nil
}

// ABIMajor, ABIMinimumMinor and ABIQualifiedVersion mirror bridge.h. The
// mirror is checked at compile time by the arrays below, exactly as the
// game-event and frame-hook mirrors are.
const (
	ABIMajor            = uint32(C.CNA_GO_ABI_MAJOR)
	ABIMinimumMinor     = uint32(C.CNA_GO_ABI_MINIMUM_MINOR)
	ABIQualifiedVersion = uint32(C.CNA_GO_ABI_QUALIFIED_VERSION)
)

// ABIAdmits reports whether an encoded CNA C ABI version satisfies CNA-Go's
// admission policy. The decision is made by bridge.c so the loader and every
// report answer from one implementation.
func ABIAdmits(version uint32) bool {
	return C.cna_go_abi_admits(C.uint32_t(version)) != 0
}

// FormatABIVersion decodes an encoded CNA ABI version into major.minor.patch.
func FormatABIVersion(version uint32) string {
	return fmt.Sprintf("%d.%d.%d",
		uint32(C.cna_go_abi_major_of(C.uint32_t(version))),
		uint32(C.cna_go_abi_minor_of(C.uint32_t(version))),
		uint32(C.cna_go_abi_patch_of(C.uint32_t(version))))
}

// ABIAdmissionPolicy states the admitted range in the same words the loader's
// rejection uses.
func ABIAdmissionPolicy() string {
	return fmt.Sprintf("major %d with minor %d or newer (qualified at %s)",
		ABIMajor, ABIMinimumMinor, FormatABIVersion(ABIQualifiedVersion))
}

// nativeSymbolIdentity proves every resolved pointer belongs to the symbol the
// manifest names. It returns the first disagreement, or the empty string.
func nativeSymbolIdentity() (bool, string) {
	buffer := make([]byte, 512)
	ok := C.cna_go_verify_symbol_identity((*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer))) != 0
	return ok, cStringFromBuffer(buffer)
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

func nativeGameCreate(context uintptr, title string, frameHooks FrameHookMask, timing TimingConfiguration) (uint64, error) {
	titleBytes := []byte(title)
	var titlePointer *C.char
	if len(titleBytes) > 0 {
		titlePointer = (*C.char)(unsafe.Pointer(&titleBytes[0]))
	}
	native := C.CnaGoGameTiming{
		target_elapsed_time_ticks: C.int64_t(timing.TargetElapsedTicks),
		inactive_sleep_time_ticks: C.int64_t(timing.InactiveSleepTicks),
		is_fixed_time_step:        C.uint8_t(boolToByte(timing.IsFixedTimeStep)),
		is_mouse_visible:          C.uint8_t(boolToByte(timing.IsMouseVisible)),
	}
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_game_create(C.uintptr_t(context), titlePointer, C.uint64_t(len(titleBytes)), C.uint32_t(frameHooks), &native, &handle))
	return uint64(handle), resultError("cna_game_create/cna_game_set_frame_hooks_ext/timing", code)
}

func boolToByte(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func nativeGameSetIsMouseVisible(game uint64, visible bool) error {
	return resultError("cna_game_set_is_mouse_visible", uint32(C.cna_go_game_set_is_mouse_visible(C.CnaGoHandle(game), C.uint8_t(boolToByte(visible)))))
}

func nativeGameSetIsFixedTimeStep(game uint64, fixed bool) error {
	return resultError("cna_game_set_is_fixed_time_step", uint32(C.cna_go_game_set_is_fixed_time_step(C.CnaGoHandle(game), C.uint8_t(boolToByte(fixed)))))
}

func nativeGameSetTargetElapsedTimeTicks(game uint64, ticks int64) error {
	return resultError("cna_game_set_target_elapsed_time_ticks", uint32(C.cna_go_game_set_target_elapsed_time_ticks(C.CnaGoHandle(game), C.int64_t(ticks))))
}

func nativeGameSetInactiveSleepTimeTicks(game uint64, ticks int64) error {
	return resultError("cna_game_set_inactive_sleep_time_ticks", uint32(C.cna_go_game_set_inactive_sleep_time_ticks(C.CnaGoHandle(game), C.int64_t(ticks))))
}

func nativeGameResetElapsedTime(game uint64) error {
	return resultError("cna_game_reset_elapsed_time", uint32(C.cna_go_game_reset_elapsed_time(C.CnaGoHandle(game))))
}

func nativeGameSuppressDraw(game uint64) error {
	return resultError("cna_game_suppress_draw", uint32(C.cna_go_game_suppress_draw(C.CnaGoHandle(game))))
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

func nativeTextureCreateEncodedSized(device uint64, data []byte, width, height uint32, zoom bool) (uint64, error) {
	var pointer *C.uint8_t
	if len(data) > 0 {
		pointer = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	crop := C.uint8_t(0)
	if zoom {
		crop = 1
	}
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_texture2d_create_from_encoded_memory_sized(
		C.CnaGoHandle(device), pointer, C.uint64_t(len(data)),
		C.uint32_t(width), C.uint32_t(height), crop, &handle))
	return uint64(handle), resultError("cna_texture2d_create_from_encoded_memory", code)
}

func nativeTextureEncodedByteCount(texture uint64, imageFormat, width, height uint32) (uint64, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_texture2d_get_encoded_byte_count(
		C.CnaGoHandle(texture), C.uint32_t(imageFormat), C.uint32_t(width), C.uint32_t(height), &count))
	return uint64(count), resultError("cna_texture2d_get_encoded_byte_count", code)
}

func nativeTextureCopyEncoded(texture uint64, imageFormat, width, height uint32, destination []byte) (uint64, error) {
	var pointer *C.uint8_t
	if len(destination) > 0 {
		pointer = (*C.uint8_t)(unsafe.Pointer(&destination[0]))
	}
	var written C.uint64_t
	code := uint32(C.cna_go_texture2d_copy_encoded(
		C.CnaGoHandle(texture), C.uint32_t(imageFormat), C.uint32_t(width), C.uint32_t(height),
		pointer, C.uint64_t(len(destination)), &written))
	return uint64(written), resultError("cna_texture2d_copy_encoded", code)
}

func nativeTextureSetData(texture uint64, dataType uint32, transfer TextureTransfer, data unsafe.Pointer, capacity uint64) error {
	rectangle := C.uint8_t(0)
	if transfer.HasRectangle {
		rectangle = 1
	}
	return resultError("cna_texture2d_set_data", uint32(C.cna_go_texture2d_set_data(
		C.CnaGoHandle(texture), C.uint32_t(dataType), C.int32_t(transfer.Level), rectangle,
		C.int32_t(transfer.X), C.int32_t(transfer.Y), C.int32_t(transfer.Width), C.int32_t(transfer.Height),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount), data, C.uint64_t(capacity))))
}

func nativeTextureGetData(texture uint64, dataType uint32, transfer TextureTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	rectangle := C.uint8_t(0)
	if transfer.HasRectangle {
		rectangle = 1
	}
	var required C.uint64_t
	code := uint32(C.cna_go_texture2d_get_data(
		C.CnaGoHandle(texture), C.uint32_t(dataType), C.int32_t(transfer.Level), rectangle,
		C.int32_t(transfer.X), C.int32_t(transfer.Y), C.int32_t(transfer.Width), C.int32_t(transfer.Height),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount), destination, C.uint64_t(capacity), &required))
	return uint64(required), resultError("cna_texture2d_get_data", code)
}

func nativeTextureInfo(texture uint64) (TextureInfo, error) {
	var width, height, levels, format C.uint32_t
	code := uint32(C.cna_go_texture2d_get_info(C.CnaGoHandle(texture), &width, &height, &levels, &format))
	return TextureInfo{Width: uint32(width), Height: uint32(height), Levels: uint32(levels), Format: uint32(format)}, resultError("cna_texture2d_get_info", code)
}

// ShaderStage is CNA_ShaderStage: which of the device's two sampler and texture
// collections a slot belongs to.
const (
	ShaderStagePixel  uint32 = 0
	ShaderStageVertex uint32 = 1
)

// TextureSlot is CNA_TextureSlotInfo, flattened.
//
// Handle is deliberately not a *Resource. CNA states plainly that there is NO
// route from a native object back to a handle, so a slot filled by CNA's own
// code -- a SpriteBatch flush, for example -- reports Bound with an INVALID
// handle. A binding that answered from a cache alone could not tell that case
// from an empty slot, which is exactly what this field is for.
type TextureSlot struct {
	Bound  bool
	Handle uint64
}

func nativeGraphicsDeviceGetTexture(device uint64, stage, slot uint32) (TextureSlot, error) {
	var bound C.uint8_t
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_get_texture(
		C.CnaGoHandle(device), C.uint32_t(stage), C.uint32_t(slot), &bound, &handle))
	return TextureSlot{Bound: bound != 0, Handle: uint64(handle)},
		resultError("cna_graphics_device_get_texture", code)
}

func nativeGraphicsDeviceSetTexture(device uint64, stage, slot uint32, texture uint64) error {
	return resultError("cna_graphics_device_set_texture", uint32(C.cna_go_graphics_device_set_texture(
		C.CnaGoHandle(device), C.uint32_t(stage), C.uint32_t(slot), C.CnaGoHandle(texture))))
}

func nativeGraphicsDeviceGetSamplerState(device uint64, stage, slot uint32) (SamplerStateValue, error) {
	var words [4]C.uint32_t
	var ints [2]C.int32_t
	var bias C.float
	code := uint32(C.cna_go_graphics_device_get_sampler_state(
		C.CnaGoHandle(device), C.uint32_t(stage), C.uint32_t(slot), &words[0], &ints[0], &bias))
	return SamplerStateValue{
		AddressU: uint32(words[0]), AddressV: uint32(words[1]), AddressW: uint32(words[2]),
		Filter: uint32(words[3]), MaxAnisotropy: int32(ints[0]), MaxMipLevel: int32(ints[1]),
		MipMapLevelOfDetailBias: float32(bias),
	}, resultError("cna_graphics_device_get_sampler_state", code)
}

func nativeGraphicsDeviceSetSamplerState(device uint64, stage, slot uint32, value SamplerStateValue) error {
	words := [4]C.uint32_t{
		C.uint32_t(value.AddressU), C.uint32_t(value.AddressV),
		C.uint32_t(value.AddressW), C.uint32_t(value.Filter)}
	ints := [2]C.int32_t{C.int32_t(value.MaxAnisotropy), C.int32_t(value.MaxMipLevel)}
	return resultError("cna_graphics_device_set_sampler_state", uint32(C.cna_go_graphics_device_set_sampler_state(
		C.CnaGoHandle(device), C.uint32_t(stage), C.uint32_t(slot),
		&words[0], &ints[0], C.float(value.MipMapLevelOfDetailBias))))
}

// The four state routes. Each flattens its descriptor into scalars; the bridge
// builds CNA's versioned POD on the C side.

func nativeGraphicsDeviceSetBlendState(device uint64, value BlendStateValue) error {
	return resultError("cna_graphics_device_set_blend_state", uint32(C.cna_go_graphics_device_set_blend_state(
		C.CnaGoHandle(device),
		C.uint32_t(value.AlphaBlendFunction), C.uint32_t(value.AlphaDestinationBlend), C.uint32_t(value.AlphaSourceBlend),
		C.uint32_t(value.ColorBlendFunction), C.uint32_t(value.ColorDestinationBlend), C.uint32_t(value.ColorSourceBlend),
		C.uint32_t(value.ColorWriteChannels), C.uint32_t(value.ColorWriteChannels1),
		C.uint32_t(value.ColorWriteChannels2), C.uint32_t(value.ColorWriteChannels3),
		C.uint8_t(value.BlendFactorR), C.uint8_t(value.BlendFactorG),
		C.uint8_t(value.BlendFactorB), C.uint8_t(value.BlendFactorA),
		C.int32_t(value.MultiSampleMask))))
}

func nativeGraphicsDeviceSetDepthStencilState(device uint64, value DepthStencilStateValue) error {
	return resultError("cna_graphics_device_set_depth_stencil_state", uint32(C.cna_go_graphics_device_set_depth_stencil_state(
		C.CnaGoHandle(device),
		cnaBool(value.DepthBufferEnable), cnaBool(value.DepthBufferWriteEnable),
		cnaBool(value.StencilEnable), cnaBool(value.TwoSidedStencilMode),
		C.uint32_t(value.DepthBufferFunction), C.uint32_t(value.StencilFunction),
		C.int32_t(value.StencilMask), C.int32_t(value.StencilWriteMask), C.int32_t(value.ReferenceStencil),
		C.uint32_t(value.StencilFail), C.uint32_t(value.StencilDepthBufferFail), C.uint32_t(value.StencilPass),
		C.uint32_t(value.CounterClockwiseStencilFunction), C.uint32_t(value.CounterClockwiseStencilFail),
		C.uint32_t(value.CounterClockwiseStencilDepthBufferFail), C.uint32_t(value.CounterClockwiseStencilPass))))
}

func nativeGraphicsDeviceSetRasterizerState(device uint64, value RasterizerStateValue) error {
	return resultError("cna_graphics_device_set_rasterizer_state", uint32(C.cna_go_graphics_device_set_rasterizer_state(
		C.CnaGoHandle(device), C.uint32_t(value.CullMode), C.uint32_t(value.FillMode),
		C.float(value.DepthBias), C.float(value.SlopeScaleDepthBias),
		cnaBool(value.MultiSampleAntiAlias), cnaBool(value.ScissorTestEnable))))
}

func nativeSpriteBatchBeginWithStates(
	batch uint64, sortMode uint32, blend BlendStateValue, sampler SamplerStateValue,
	depth DepthStencilStateValue, rasterizer RasterizerStateValue,
) error {
	blendWords := [10]C.uint32_t{
		C.uint32_t(blend.AlphaBlendFunction), C.uint32_t(blend.AlphaDestinationBlend), C.uint32_t(blend.AlphaSourceBlend),
		C.uint32_t(blend.ColorBlendFunction), C.uint32_t(blend.ColorDestinationBlend), C.uint32_t(blend.ColorSourceBlend),
		C.uint32_t(blend.ColorWriteChannels), C.uint32_t(blend.ColorWriteChannels1),
		C.uint32_t(blend.ColorWriteChannels2), C.uint32_t(blend.ColorWriteChannels3)}
	blendMask := [1]C.int32_t{C.int32_t(blend.MultiSampleMask)}
	blendFactor := [4]C.uint8_t{
		C.uint8_t(blend.BlendFactorR), C.uint8_t(blend.BlendFactorG),
		C.uint8_t(blend.BlendFactorB), C.uint8_t(blend.BlendFactorA)}
	samplerWords := [4]C.uint32_t{
		C.uint32_t(sampler.AddressU), C.uint32_t(sampler.AddressV),
		C.uint32_t(sampler.AddressW), C.uint32_t(sampler.Filter)}
	samplerInts := [2]C.int32_t{C.int32_t(sampler.MaxAnisotropy), C.int32_t(sampler.MaxMipLevel)}
	depthFlags := [4]C.uint8_t{
		cnaBool(depth.DepthBufferEnable), cnaBool(depth.DepthBufferWriteEnable),
		cnaBool(depth.StencilEnable), cnaBool(depth.TwoSidedStencilMode)}
	depthWords := [9]C.uint32_t{
		C.uint32_t(depth.DepthBufferFunction), C.uint32_t(depth.StencilFunction),
		C.uint32_t(depth.StencilFail), C.uint32_t(depth.StencilDepthBufferFail), C.uint32_t(depth.StencilPass),
		C.uint32_t(depth.CounterClockwiseStencilFunction), C.uint32_t(depth.CounterClockwiseStencilFail),
		C.uint32_t(depth.CounterClockwiseStencilDepthBufferFail), C.uint32_t(depth.CounterClockwiseStencilPass)}
	depthInts := [3]C.int32_t{
		C.int32_t(depth.StencilMask), C.int32_t(depth.StencilWriteMask), C.int32_t(depth.ReferenceStencil)}
	return resultError("cna_sprite_batch_begin_with_states", uint32(C.cna_go_sprite_batch_begin_with_states(
		C.CnaGoHandle(batch), C.uint32_t(sortMode),
		&blendWords[0], &blendMask[0], &blendFactor[0],
		&samplerWords[0], &samplerInts[0], C.float(sampler.MipMapLevelOfDetailBias),
		&depthFlags[0], &depthWords[0], &depthInts[0],
		C.uint32_t(rasterizer.CullMode), C.uint32_t(rasterizer.FillMode),
		C.float(rasterizer.DepthBias), C.float(rasterizer.SlopeScaleDepthBias),
		cnaBool(rasterizer.MultiSampleAntiAlias), cnaBool(rasterizer.ScissorTestEnable))))
}

// cnaBool is CNA_Bool's one-byte convention, written once so seventeen call
// sites cannot disagree about it.
func cnaBool(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}

// The four render-target routes. CNA's create and info structures are
// VERSIONED, so the bridge fills struct_size and struct_version on the C side
// and flattens every other field into scalars: no CNA structure crosses cgo,
// which is the same rule the texture and sprite families already follow.

func nativeRenderTarget2DCreate(device uint64, width, height uint32, mipMap bool, format, depthFormat uint32, multiSampleCount int32, usage uint32) (uint64, error) {
	var handle C.CnaGoHandle
	var mip C.uint8_t
	if mipMap {
		mip = 1
	}
	code := uint32(C.cna_go_render_target2d_create(
		C.CnaGoHandle(device), C.uint32_t(width), C.uint32_t(height), mip,
		C.uint32_t(format), C.uint32_t(depthFormat), C.int32_t(multiSampleCount), C.uint32_t(usage), &handle))
	return uint64(handle), resultError("cna_render_target2d_create", code)
}

func nativeRenderTargetInfo(renderTarget uint64) (RenderTargetInfo, error) {
	var kind, width, height, levelCount, format, depthFormat, usage C.uint32_t
	var multiSampleCount C.int32_t
	var contentLost, rendererAvailable C.uint8_t
	code := uint32(C.cna_go_render_target_get_info(
		C.CnaGoHandle(renderTarget), &kind, &width, &height, &levelCount,
		&format, &depthFormat, &multiSampleCount, &usage, &contentLost, &rendererAvailable))
	return RenderTargetInfo{
		Kind: uint32(kind), Width: uint32(width), Height: uint32(height),
		LevelCount: uint32(levelCount), Format: uint32(format), DepthFormat: uint32(depthFormat),
		MultiSampleCount: int32(multiSampleCount), Usage: uint32(usage),
		IsContentLost: contentLost != 0, RendererAvailable: rendererAvailable != 0,
	}, resultError("cna_render_target_get_info", code)
}

func nativeRenderTargetDestroy(renderTarget uint64) error {
	return resultError("cna_render_target_destroy",
		uint32(C.cna_go_render_target_destroy(C.CnaGoHandle(renderTarget))))
}

func nativeGraphicsDeviceSetRenderTarget2D(device, renderTarget uint64) error {
	return resultError("cna_graphics_device_set_render_target2d",
		uint32(C.cna_go_graphics_device_set_render_target2d(C.CnaGoHandle(device), C.CnaGoHandle(renderTarget))))
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

func nativeGraphicsDeviceBlendFactor(device uint64) (uint8, uint8, uint8, uint8, error) {
	var r, g, b, a C.uint8_t
	code := uint32(C.cna_go_graphics_device_get_blend_factor(C.CnaGoHandle(device), &r, &g, &b, &a))
	return uint8(r), uint8(g), uint8(b), uint8(a), resultError("cna_graphics_device_get_blend_factor", code)
}

func nativeGraphicsDeviceSetBlendFactor(device uint64, r, g, b, a uint8) error {
	return resultError("cna_graphics_device_set_blend_factor", uint32(C.cna_go_graphics_device_set_blend_factor(
		C.CnaGoHandle(device), C.uint8_t(r), C.uint8_t(g), C.uint8_t(b), C.uint8_t(a))))
}

func nativeGraphicsDeviceMultiSampleMask(device uint64) (int32, error) {
	var mask C.int32_t
	code := uint32(C.cna_go_graphics_device_get_multi_sample_mask(C.CnaGoHandle(device), &mask))
	return int32(mask), resultError("cna_graphics_device_get_multi_sample_mask", code)
}

func nativeGraphicsDeviceSetMultiSampleMask(device uint64, mask int32) error {
	return resultError("cna_graphics_device_set_multi_sample_mask",
		uint32(C.cna_go_graphics_device_set_multi_sample_mask(C.CnaGoHandle(device), C.int32_t(mask))))
}

func nativeGraphicsDeviceReferenceStencil(device uint64) (int32, error) {
	var stencil C.int32_t
	code := uint32(C.cna_go_graphics_device_get_reference_stencil(C.CnaGoHandle(device), &stencil))
	return int32(stencil), resultError("cna_graphics_device_get_reference_stencil", code)
}

func nativeGraphicsDeviceSetReferenceStencil(device uint64, stencil int32) error {
	return resultError("cna_graphics_device_set_reference_stencil",
		uint32(C.cna_go_graphics_device_set_reference_stencil(C.CnaGoHandle(device), C.int32_t(stencil))))
}

func nativeGraphicsDeviceScissorRectangle(device uint64) (ScissorRectangle, error) {
	var x, y, width, height C.int32_t
	code := uint32(C.cna_go_graphics_device_get_scissor_rectangle(C.CnaGoHandle(device), &x, &y, &width, &height))
	return ScissorRectangle{X: int32(x), Y: int32(y), Width: int32(width), Height: int32(height)},
		resultError("cna_graphics_device_get_scissor_rectangle", code)
}

func nativeGraphicsDeviceSetScissorRectangle(device uint64, rectangle ScissorRectangle) error {
	return resultError("cna_graphics_device_set_scissor_rectangle", uint32(C.cna_go_graphics_device_set_scissor_rectangle(
		C.CnaGoHandle(device), C.int32_t(rectangle.X), C.int32_t(rectangle.Y),
		C.int32_t(rectangle.Width), C.int32_t(rectangle.Height))))
}

func nativeGraphicsDeviceSetViewport(device uint64, viewport Viewport) error {
	return resultError("cna_graphics_device_set_viewport", uint32(C.cna_go_graphics_device_set_viewport(
		C.CnaGoHandle(device), C.int32_t(viewport.X), C.int32_t(viewport.Y),
		C.int32_t(viewport.Width), C.int32_t(viewport.Height),
		C.float(viewport.MinDepth), C.float(viewport.MaxDepth))))
}

func nativeGraphicsDeviceGraphicsProfile(device uint64) (uint32, error) {
	var profile C.uint32_t
	code := uint32(C.cna_go_graphics_device_get_graphics_profile(C.CnaGoHandle(device), &profile))
	return uint32(profile), resultError("cna_graphics_device_get_graphics_profile", code)
}

func nativeGraphicsDeviceStatus(device uint64) (uint32, error) {
	var status C.uint32_t
	code := uint32(C.cna_go_graphics_device_get_status(C.CnaGoHandle(device), &status))
	return uint32(status), resultError("cna_graphics_device_get_status", code)
}

func nativeGraphicsDeviceIsDisposed(device uint64) (bool, error) {
	var disposed C.uint8_t
	code := uint32(C.cna_go_graphics_device_get_is_disposed(C.CnaGoHandle(device), &disposed))
	return disposed != 0, resultError("cna_graphics_device_get_is_disposed", code)
}

func nativeGraphicsDeviceClearOptions(device uint64, options uint32, r, g, b, a uint8, depth float32, stencil int32) error {
	return resultError("cna_graphics_device_clear_options", uint32(C.cna_go_graphics_device_clear_options(
		C.CnaGoHandle(device), C.uint32_t(options),
		C.uint8_t(r), C.uint8_t(g), C.uint8_t(b), C.uint8_t(a),
		C.float(depth), C.int32_t(stencil))))
}

func nativeTextureCreate(device uint64, width, height uint32, mipMap bool, format uint32) (uint64, error) {
	var handle C.CnaGoHandle
	mip := C.uint8_t(0)
	if mipMap {
		mip = 1
	}
	code := uint32(C.cna_go_texture2d_create(C.CnaGoHandle(device), C.uint32_t(width), C.uint32_t(height), mip, C.uint32_t(format), &handle))
	return uint64(handle), resultError("cna_texture2d_create", code)
}

func nativeGraphicsDeviceDisplayMode(device uint64) (DisplayMode, error) {
	var width, height C.int32_t
	var aspect C.float
	var format C.uint32_t
	code := uint32(C.cna_go_graphics_device_get_display_mode(C.CnaGoHandle(device), &width, &height, &aspect, &format))
	return DisplayMode{
		Width: int32(width), Height: int32(height),
		AspectRatio: float32(aspect), Format: uint32(format),
	}, resultError("cna_graphics_device_get_display_mode", code)
}

func nativeGraphicsDevicePresent(device uint64) error {
	return resultError("cna_graphics_device_present", uint32(C.cna_go_graphics_device_present(C.CnaGoHandle(device))))
}

func nativeSpriteBatchDrawDestination(batch, texture uint64, command SpriteDestinationCommand) error {
	return resultError("cna_sprite_batch_submit_many", uint32(C.cna_go_sprite_batch_draw_destination(
		C.CnaGoHandle(batch), C.CnaGoHandle(texture),
		C.int32_t(command.DestinationX), C.int32_t(command.DestinationY),
		C.int32_t(command.DestinationWidth), C.int32_t(command.DestinationHeight),
		C.int32_t(command.SourceX), C.int32_t(command.SourceY), C.int32_t(command.SourceWidth), C.int32_t(command.SourceHeight),
		C.uint8_t(command.Red), C.uint8_t(command.Green), C.uint8_t(command.Blue), C.uint8_t(command.Alpha),
		C.float(command.Rotation), C.float(command.OriginX), C.float(command.OriginY),
		C.uint32_t(command.Effects), C.float(command.LayerDepth))))
}

func nativeSpriteBatchEnd(batch uint64) error {
	return resultError("cna_sprite_batch_end", uint32(C.cna_go_sprite_batch_end(C.CnaGoHandle(batch))))
}

func nativeSpriteBatchDestroy(batch uint64) error {
	return resultError("cna_sprite_batch_destroy", uint32(C.cna_go_sprite_batch_destroy(C.CnaGoHandle(batch))))
}

func nativeManagerCreateDevice(manager uint64) error {
	return resultError("cna_graphics_device_manager_create_device",
		uint32(C.cna_go_graphics_device_manager_create_device(C.CnaGoHandle(manager))))
}

func nativeManagerBeginDraw(manager uint64) (bool, error) {
	var shouldDraw C.uint8_t
	code := uint32(C.cna_go_graphics_device_manager_begin_draw(C.CnaGoHandle(manager), &shouldDraw))
	return shouldDraw != 0, resultError("cna_graphics_device_manager_begin_draw", code)
}

func nativeManagerEndDraw(manager uint64) error {
	return resultError("cna_graphics_device_manager_end_draw",
		uint32(C.cna_go_graphics_device_manager_end_draw(C.CnaGoHandle(manager))))
}

func nativeManagerSubscribeEvents(manager uint64, context uintptr) ([managerEventCount]uint64, error) {
	var registrations [managerEventCount]C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_manager_subscribe_events(C.CnaGoHandle(manager), C.uintptr_t(context), &registrations[0]))
	var result [managerEventCount]uint64
	for i := range result {
		result[i] = uint64(registrations[i])
	}
	return result, resultError("cna_graphics_device_manager_subscribe", code)
}

func nativeManagerUnsubscribeEvents(registrations *[managerEventCount]uint64) error {
	var native [managerEventCount]C.CnaGoHandle
	for i, handle := range registrations {
		native[i] = C.CnaGoHandle(handle)
	}
	code := uint32(C.cna_go_graphics_device_manager_unsubscribe_events(&native[0]))
	for i := range registrations {
		registrations[i] = uint64(native[i])
	}
	return resultError("cna_game_unsubscribe", code)
}

// cnaGoGraphicsDeviceManagerEvent is the manager family's own trampoline. Its
// context is a per-MANAGER cgo.Handle rather than the runtime's, because the
// signal belongs to one manager object.
//
//export cnaGoGraphicsDeviceManagerEvent
func cnaGoGraphicsDeviceManagerEvent(event C.uint32_t, context C.uintptr_t) {
	var signals *ManagerSignals
	defer func() {
		if recovered := recover(); recovered != nil && signals != nil && signals.runtime != nil {
			signals.runtime.recordCallbackFailure(fmt.Errorf("panic in native manager-event trampoline: %v", recovered))
		}
	}()
	handle := cgo.Handle(context)
	signals = handle.Value().(*ManagerSignals)
	signals.deliver(uint32(event))
}

func nativeDeviceSubscribeEvents(device uint64, context uintptr) ([deviceEventCount]uint64, error) {
	var registrations [deviceEventCount]C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_subscribe_events(
		C.CnaGoHandle(device), C.uintptr_t(context), &registrations[0]))
	var result [deviceEventCount]uint64
	for i := range registrations {
		result[i] = uint64(registrations[i])
	}
	return result, resultError("cna_graphics_device_subscribe_event", code)
}

func nativeDeviceUnsubscribeEvents(registrations *[deviceEventCount]uint64) error {
	var native [deviceEventCount]C.CnaGoHandle
	for i, registration := range registrations {
		native[i] = C.CnaGoHandle(registration)
	}
	code := uint32(C.cna_go_graphics_device_unsubscribe_events(&native[0]))
	for i := range registrations {
		registrations[i] = uint64(native[i])
	}
	return resultError("cna_graphics_device_unsubscribe", code)
}

func nativeGraphicsDeviceDispose(device uint64) error {
	return resultError("cna_graphics_device_dispose",
		uint32(C.cna_go_graphics_device_dispose(C.CnaGoHandle(device))))
}

// The nine content-manager routes. Every string crosses as a pointer and a
// length, which the bridge wraps in a CNA_StringView on its own side, and every
// read is the two-call length-then-copy shape CNA's string reads take.

func nativeContentManagerCreate(device uint64, rootDirectory string) (uint64, error) {
	var handle C.CnaGoHandle
	var data *C.char
	if len(rootDirectory) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(rootDirectory)))
	}
	code := uint32(C.cna_go_content_manager_create(
		C.CnaGoHandle(device), data, C.uint64_t(len(rootDirectory)), &handle))
	runtime.KeepAlive(rootDirectory)
	return uint64(handle), resultError("cna_content_manager_create", code)
}

func nativeContentManagerDestroy(manager uint64) error {
	return resultError("cna_content_manager_destroy",
		uint32(C.cna_go_content_manager_destroy(C.CnaGoHandle(manager))))
}

func nativeContentManagerRootDirectory(manager uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_content_manager_get_root_directory_size(C.CnaGoHandle(manager), &byteCount))
	if err := resultError("cna_content_manager_get_root_directory_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_content_manager_copy_root_directory(
		C.CnaGoHandle(manager), (*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_content_manager_copy_root_directory", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeContentManagerSetRootDirectory(manager uint64, rootDirectory string) error {
	var data *C.char
	if len(rootDirectory) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(rootDirectory)))
	}
	code := uint32(C.cna_go_content_manager_set_root_directory(
		C.CnaGoHandle(manager), data, C.uint64_t(len(rootDirectory))))
	runtime.KeepAlive(rootDirectory)
	return resultError("cna_content_manager_set_root_directory", code)
}

func nativeContentManagerUnload(manager uint64) error {
	return resultError("cna_content_manager_unload",
		uint32(C.cna_go_content_manager_unload(C.CnaGoHandle(manager))))
}

func nativeContentManagerLoadTexture2D(manager uint64, assetName string) (uint64, error) {
	var handle C.CnaGoHandle
	var data *C.char
	if len(assetName) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(assetName)))
	}
	code := uint32(C.cna_go_content_manager_load_texture2d(
		C.CnaGoHandle(manager), data, C.uint64_t(len(assetName)), &handle))
	runtime.KeepAlive(assetName)
	return uint64(handle), resultError("cna_content_manager_load_texture2d", code)
}

// The eight SpriteFont routes. Foundation 69.
//
// The loader is the only route in this file that produces TWO owned handles
// from one call, and the pair is reported together for the reason CNA reports
// it together: the atlas is retained while the font lives, so a caller that
// received only the font would hold a texture it could neither reach nor
// release.

func nativeContentManagerLoadSpriteFont(manager uint64, assetName string) (uint64, uint64, error) {
	var font, texture C.CnaGoHandle
	var data *C.char
	if len(assetName) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(assetName)))
	}
	code := uint32(C.cna_go_content_manager_load_sprite_font(
		C.CnaGoHandle(manager), data, C.uint64_t(len(assetName)), &font, &texture))
	runtime.KeepAlive(assetName)
	if err := resultError("cna_content_manager_load_sprite_font", code); err != nil {
		return 0, 0, err
	}
	return uint64(font), uint64(texture), nil
}

func nativeSpriteFontInfo(font uint64) (SpriteFontInfo, error) {
	var characterCount C.uint64_t
	var lineSpacing C.int32_t
	var spacing C.float
	var defaultCharacter C.uint16_t
	var hasDefault C.uint8_t
	code := uint32(C.cna_go_sprite_font_get_info(
		C.CnaGoHandle(font), &characterCount, &lineSpacing, &spacing, &defaultCharacter, &hasDefault))
	if err := resultError("cna_sprite_font_get_info", code); err != nil {
		return SpriteFontInfo{}, err
	}
	return SpriteFontInfo{
		CharacterCount:      uint64(characterCount),
		LineSpacing:         int32(lineSpacing),
		Spacing:             float32(spacing),
		DefaultCharacter:    uint16(defaultCharacter),
		HasDefaultCharacter: hasDefault != 0,
	}, nil
}

// nativeSpriteFontGlyphs reads the whole glyph table in one call. capacity is
// the count cna_sprite_font_get_info already reported, so the two-call
// size-then-copy shape the string reads use is not needed here.
func nativeSpriteFontGlyphs(font uint64, capacity uint64) ([]SpriteFontGlyph, error) {
	if capacity == 0 {
		var count C.uint64_t
		code := uint32(C.cna_go_sprite_font_copy_glyphs(
			C.CnaGoHandle(font), 0, nil, nil, nil, &count))
		if err := resultError("cna_sprite_font_copy_glyphs", code); err != nil {
			return nil, err
		}
		return nil, nil
	}
	characters := make([]uint16, capacity)
	rectangles := make([]int32, capacity*8)
	kerning := make([]float32, capacity*3)
	var count C.uint64_t
	code := uint32(C.cna_go_sprite_font_copy_glyphs(
		C.CnaGoHandle(font), C.uint64_t(capacity),
		(*C.uint16_t)(unsafe.Pointer(&characters[0])),
		(*C.int32_t)(unsafe.Pointer(&rectangles[0])),
		(*C.float)(unsafe.Pointer(&kerning[0])),
		&count))
	runtime.KeepAlive(characters)
	runtime.KeepAlive(rectangles)
	runtime.KeepAlive(kerning)
	if err := resultError("cna_sprite_font_copy_glyphs", code); err != nil {
		return nil, err
	}
	written := uint64(count)
	if written > capacity {
		written = capacity
	}
	glyphs := make([]SpriteFontGlyph, written)
	for at := uint64(0); at < written; at++ {
		glyphs[at] = SpriteFontGlyph{
			Character: characters[at],
			GlyphBounds: SpriteFontRectangle{
				X: rectangles[at*8+0], Y: rectangles[at*8+1],
				Width: rectangles[at*8+2], Height: rectangles[at*8+3],
			},
			Cropping: SpriteFontRectangle{
				X: rectangles[at*8+4], Y: rectangles[at*8+5],
				Width: rectangles[at*8+6], Height: rectangles[at*8+7],
			},
			KerningX: kerning[at*3+0],
			KerningY: kerning[at*3+1],
			KerningZ: kerning[at*3+2],
		}
	}
	return glyphs, nil
}

func nativeSpriteFontSetDefaultCharacter(font uint64, hasValue bool, value uint16) error {
	flag := C.uint8_t(0)
	if hasValue {
		flag = 1
	}
	return resultError("cna_sprite_font_set_default_character",
		uint32(C.cna_go_sprite_font_set_default_character(C.CnaGoHandle(font), flag, C.uint16_t(value))))
}

func nativeSpriteFontSetLineSpacing(font uint64, lineSpacing int32) error {
	return resultError("cna_sprite_font_set_line_spacing",
		uint32(C.cna_go_sprite_font_set_line_spacing(C.CnaGoHandle(font), C.int32_t(lineSpacing))))
}

func nativeSpriteFontSetSpacing(font uint64, spacing float32) error {
	return resultError("cna_sprite_font_set_spacing",
		uint32(C.cna_go_sprite_font_set_spacing(C.CnaGoHandle(font), C.float(spacing))))
}

func nativeSpriteFontDestroy(font uint64) error {
	return resultError("cna_sprite_font_destroy",
		uint32(C.cna_go_sprite_font_destroy(C.CnaGoHandle(font))))
}

func nativeSpriteBatchDrawString(batch, font uint64, text string, command SpriteTextCommand) error {
	var data *C.char
	if len(text) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(text)))
	}
	code := uint32(C.cna_go_sprite_batch_draw_string(
		C.CnaGoHandle(batch), C.CnaGoHandle(font), data, C.uint64_t(len(text)),
		C.float(command.PositionX), C.float(command.PositionY),
		C.uint8_t(command.Red), C.uint8_t(command.Green), C.uint8_t(command.Blue), C.uint8_t(command.Alpha),
		C.float(command.Rotation), C.float(command.OriginX), C.float(command.OriginY),
		C.float(command.ScaleX), C.float(command.ScaleY),
		C.uint32_t(command.Effects), C.float(command.LayerDepth)))
	runtime.KeepAlive(text)
	return resultError("cna_sprite_batch_draw_string", code)
}

func nativeContentManagerAssetPath(manager uint64, assetName string) (string, error) {
	var data *C.char
	if len(assetName) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(assetName)))
	}
	var byteCount C.uint64_t
	code := uint32(C.cna_go_content_manager_get_asset_path_size(
		C.CnaGoHandle(manager), data, C.uint64_t(len(assetName)), &byteCount))
	if err := resultError("cna_content_manager_get_asset_path_size", code); err != nil {
		runtime.KeepAlive(assetName)
		return "", err
	}
	if byteCount == 0 {
		runtime.KeepAlive(assetName)
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_content_manager_copy_asset_path(
		C.CnaGoHandle(manager), data, C.uint64_t(len(assetName)),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	runtime.KeepAlive(assetName)
	if err := resultError("cna_content_manager_copy_asset_path", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

// The three device trampolines. Their context is a per-DEVICE-FACADE cgo.Handle:
// a game holds one device, but a facade is per manager per generation and a
// signal has to reach the one that subscribed.

//export cnaGoGraphicsDeviceEvent
func cnaGoGraphicsDeviceEvent(event C.uint32_t, context C.uintptr_t) {
	deliverDeviceSignal(context, uint32(event), DeviceSignalPayload{})
}

//export cnaGoGraphicsDeviceResourceCreated
func cnaGoGraphicsDeviceResourceCreated(hasResource C.uint8_t, context C.uintptr_t) {
	deliverDeviceSignal(context, DeviceEventResourceCreated,
		DeviceSignalPayload{HasResource: hasResource != 0})
}

//export cnaGoGraphicsDeviceResourceDestroyed
func cnaGoGraphicsDeviceResourceDestroyed(
	hasTag C.uint8_t, name *C.char, nameLength C.uint64_t, context C.uintptr_t,
) {
	// The name borrows bytes that expire when the callback returns, so it is
	// COPIED into Go memory here rather than retained.
	payload := DeviceSignalPayload{HasTag: hasTag != 0}
	if name != nil && nameLength > 0 {
		payload.Name = C.GoStringN(name, C.int(nameLength))
	}
	deliverDeviceSignal(context, DeviceEventResourceDestroyed, payload)
}

func deliverDeviceSignal(context C.uintptr_t, event uint32, payload DeviceSignalPayload) {
	var signals *DeviceSignals
	defer func() {
		if recovered := recover(); recovered != nil && signals != nil && signals.runtime != nil {
			signals.runtime.recordCallbackFailure(
				fmt.Errorf("panic in native device-event trampoline: %v", recovered))
		}
	}()
	handle := cgo.Handle(uintptr(context))
	signals = handle.Value().(*DeviceSignals)
	signals.deliver(event, payload)
}

// The GraphicsDeviceManager configuration setters. Each is a store CNA's own
// manager applies at ChangeDevice time, so a value that never reached it would
// be a setting that appears to work and does not.
func nativeManagerSetGraphicsProfile(manager uint64, profile uint32) error {
	return resultError("cna_graphics_device_manager_set_graphics_profile",
		uint32(C.cna_go_graphics_device_manager_set_graphics_profile(C.CnaGoHandle(manager), C.uint32_t(profile))))
}

func nativeManagerSetIsFullScreen(manager uint64, value bool) error {
	return resultError("cna_graphics_device_manager_set_is_full_screen",
		uint32(C.cna_go_graphics_device_manager_set_is_full_screen(C.CnaGoHandle(manager), C.uint8_t(boolToByte(value)))))
}

func nativeManagerSetPreferMultiSampling(manager uint64, value bool) error {
	return resultError("cna_graphics_device_manager_set_prefer_multi_sampling",
		uint32(C.cna_go_graphics_device_manager_set_prefer_multi_sampling(C.CnaGoHandle(manager), C.uint8_t(boolToByte(value)))))
}

func nativeManagerSetPreferredBackBufferFormat(manager uint64, format uint32) error {
	return resultError("cna_graphics_device_manager_set_preferred_back_buffer_format",
		uint32(C.cna_go_graphics_device_manager_set_preferred_back_buffer_format(C.CnaGoHandle(manager), C.uint32_t(format))))
}

func nativeManagerSetPreferredBackBufferWidth(manager uint64, width int32) error {
	return resultError("cna_graphics_device_manager_set_preferred_back_buffer_width",
		uint32(C.cna_go_graphics_device_manager_set_preferred_back_buffer_width(C.CnaGoHandle(manager), C.int32_t(width))))
}

func nativeManagerSetPreferredBackBufferHeight(manager uint64, height int32) error {
	return resultError("cna_graphics_device_manager_set_preferred_back_buffer_height",
		uint32(C.cna_go_graphics_device_manager_set_preferred_back_buffer_height(C.CnaGoHandle(manager), C.int32_t(height))))
}

func nativeManagerSetPreferredDepthStencilFormat(manager uint64, format uint32) error {
	return resultError("cna_graphics_device_manager_set_preferred_depth_stencil_format",
		uint32(C.cna_go_graphics_device_manager_set_preferred_depth_stencil_format(C.CnaGoHandle(manager), C.uint32_t(format))))
}

func nativeManagerSetSynchronizeWithVerticalRetrace(manager uint64, value bool) error {
	return resultError("cna_graphics_device_manager_set_synchronize_with_vertical_retrace",
		uint32(C.cna_go_graphics_device_manager_set_synchronize_with_vertical_retrace(C.CnaGoHandle(manager), C.uint8_t(boolToByte(value)))))
}

func nativeManagerSetSupportedOrientations(manager uint64, orientations uint32) error {
	return resultError("cna_graphics_device_manager_set_supported_orientations",
		uint32(C.cna_go_graphics_device_manager_set_supported_orientations(C.CnaGoHandle(manager), C.uint32_t(orientations))))
}

func nativeManagerApplyChanges(manager uint64) error {
	return resultError("cna_graphics_device_manager_apply_changes",
		uint32(C.cna_go_graphics_device_manager_apply_changes(C.CnaGoHandle(manager))))
}

func nativeGameTick(game uint64) error {
	return resultError("cna_game_tick", uint32(C.cna_go_game_tick(C.CnaGoHandle(game))))
}

func nativeGameRunOneFrame(game uint64) error {
	return resultError("cna_game_run_one_frame", uint32(C.cna_go_game_run_one_frame(C.CnaGoHandle(game))))
}

// The GameWindow routes. Every one takes the game handle; CNA models the
// window as a property of the game, so nothing here owns a native lifetime.
func nativeGameWindowAllowUserResizing(game uint64) (bool, error) {
	var allowed C.uint8_t
	code := uint32(C.cna_go_game_window_get_allow_user_resizing(C.CnaGoHandle(game), &allowed))
	return allowed != 0, resultError("cna_game_window_get_allow_user_resizing", code)
}

func nativeGameWindowSetAllowUserResizing(game uint64, value bool) error {
	code := uint32(C.cna_go_game_window_set_allow_user_resizing(C.CnaGoHandle(game), C.uint8_t(boolToByte(value))))
	return resultError("cna_game_window_set_allow_user_resizing", code)
}

func nativeGameWindowClientBounds(game uint64) (int32, int32, int32, int32, error) {
	var x, y, width, height C.int32_t
	code := uint32(C.cna_go_game_window_get_client_bounds(C.CnaGoHandle(game), &x, &y, &width, &height))
	return int32(x), int32(y), int32(width), int32(height), resultError("cna_game_window_get_client_bounds", code)
}

func nativeGameWindowNativeHandle(game uint64) (uint64, error) {
	var handle C.uint64_t
	code := uint32(C.cna_go_game_window_get_native_handle(C.CnaGoHandle(game), &handle))
	return uint64(handle), resultError("cna_game_window_get_native_handle_ext", code)
}

// nativeGameWindowScreenDeviceName is the two-call string read. The size call
// reports the byte count CNA would write, Go allocates that, and the copy call
// fills it; no native string pointer is retained.
func nativeGameWindowScreenDeviceName(game uint64) (string, error) {
	var required C.uint64_t
	code := uint32(C.cna_go_game_window_get_screen_device_name_size(C.CnaGoHandle(game), &required))
	if err := resultError("cna_game_window_get_screen_device_name_size", code); err != nil {
		return "", err
	}
	if required == 0 {
		return "", nil
	}
	buffer := make([]byte, int(required))
	var written C.uint64_t
	code = uint32(C.cna_go_game_window_copy_screen_device_name(
		C.CnaGoHandle(game),
		(*C.char)(unsafe.Pointer(&buffer[0])),
		C.uint64_t(len(buffer)),
		&written))
	if err := resultError("cna_game_window_copy_screen_device_name", code); err != nil {
		return "", err
	}
	if int(written) > len(buffer) {
		return "", fmt.Errorf("%w: CNA reported %d screen-device-name bytes into a %d byte buffer", ErrNativeUnavailable, int(written), len(buffer))
	}
	return string(buffer[:int(written)]), nil
}

func nativeGameWindowBeginScreenDeviceChange(game uint64, willBeFullScreen bool) error {
	code := uint32(C.cna_go_game_window_begin_screen_device_change(C.CnaGoHandle(game), C.uint8_t(boolToByte(willBeFullScreen))))
	return resultError("cna_game_window_begin_screen_device_change", code)
}

func nativeGameWindowEndScreenDeviceChange(game uint64, name string, width, height int32) error {
	nameBytes := []byte(name)
	var namePointer *C.char
	if len(nameBytes) > 0 {
		namePointer = (*C.char)(unsafe.Pointer(&nameBytes[0]))
	}
	code := uint32(C.cna_go_game_window_end_screen_device_change(
		C.CnaGoHandle(game), namePointer, C.uint64_t(len(nameBytes)), C.int32_t(width), C.int32_t(height)))
	return resultError("cna_game_window_end_screen_device_change", code)
}

func nativeGameSetWindowTitle(game uint64, title string) error {
	titleBytes := []byte(title)
	var titlePointer *C.char
	if len(titleBytes) > 0 {
		titlePointer = (*C.char)(unsafe.Pointer(&titleBytes[0]))
	}
	code := uint32(C.cna_go_game_set_window_title(C.CnaGoHandle(game), titlePointer, C.uint64_t(len(titleBytes))))
	return resultError("cna_game_set_window_title", code)
}

func nativeGameWindowSubscribeEvents(game uint64, context uintptr) ([gameWindowEventCount]uint64, error) {
	var registrations [gameWindowEventCount]C.CnaGoHandle
	code := uint32(C.cna_go_game_window_subscribe_events(C.CnaGoHandle(game), C.uintptr_t(context), &registrations[0]))
	var result [gameWindowEventCount]uint64
	for i := range result {
		result[i] = uint64(registrations[i])
	}
	return result, resultError("cna_game_window_subscribe", code)
}

func nativeGameWindowUnsubscribeEvents(registrations *[gameWindowEventCount]uint64) error {
	var native [gameWindowEventCount]C.CnaGoHandle
	for i, handle := range registrations {
		native[i] = C.CnaGoHandle(handle)
	}
	code := uint32(C.cna_go_game_window_unsubscribe_events(&native[0]))
	for i := range registrations {
		registrations[i] = uint64(native[i])
	}
	return resultError("cna_game_unsubscribe", code)
}

// cnaGoGameWindowEvent is the window family's own trampoline. Keeping it
// separate from cnaGoGameEvent is what stops a window signal from arriving as
// a game signal: both identity spaces start at zero.
//
//export cnaGoGameWindowEvent
func cnaGoGameWindowEvent(event C.uint32_t, context C.uintptr_t) {
	var state *Runtime
	defer func() {
		if recovered := recover(); recovered != nil && state != nil {
			state.recordCallbackFailure(fmt.Errorf("panic in native window-event trampoline: %v", recovered))
		}
	}()
	handle := cgo.Handle(context)
	state = handle.Value().(*Runtime)
	state.invokeGameWindowEvent(uint32(event))
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

// The six index-buffer routes. Every string is absent here: an index buffer
// carries only numbers, so nothing crosses but scalars and one caller pointer.

func nativeIndexBufferCreate(device uint64, indexCount int32, elementSize, usage uint32, dynamic bool) (uint64, error) {
	var handle C.CnaGoHandle
	flag := C.uint8_t(0)
	if dynamic {
		flag = 1
	}
	code := uint32(C.cna_go_index_buffer_create(
		C.CnaGoHandle(device), C.int32_t(indexCount), C.uint32_t(elementSize), C.uint32_t(usage), flag, &handle))
	if err := resultError("cna_index_buffer_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeIndexBufferDestroy(buffer uint64) error {
	return resultError("cna_index_buffer_destroy",
		uint32(C.cna_go_index_buffer_destroy(C.CnaGoHandle(buffer))))
}

func nativeIndexBufferInfo(buffer uint64) (IndexBufferInfo, error) {
	var count C.int32_t
	var elementSize, usage C.uint32_t
	var dynamic, contentLost, hasRenderer C.uint8_t
	code := uint32(C.cna_go_index_buffer_get_info(
		C.CnaGoHandle(buffer), &count, &elementSize, &usage, &dynamic, &contentLost, &hasRenderer))
	if err := resultError("cna_index_buffer_get_info", code); err != nil {
		return IndexBufferInfo{}, err
	}
	return IndexBufferInfo{
		IndexCount:       int32(count),
		IndexElementSize: uint32(elementSize),
		BufferUsage:      uint32(usage),
		Dynamic:          dynamic != 0,
		IsContentLost:    contentLost != 0,
		HasRenderer:      hasRenderer != 0,
	}, nil
}

func nativeIndexBufferSetData(buffer uint64, elementSize, options uint32, startIndex, elementCount uint64, data unsafe.Pointer, capacity uint64) error {
	return resultError("cna_index_buffer_set_data", uint32(C.cna_go_index_buffer_set_data(
		C.CnaGoHandle(buffer), C.uint32_t(elementSize), C.uint32_t(options),
		C.uint64_t(startIndex), C.uint64_t(elementCount), data, C.uint64_t(capacity))))
}

func nativeIndexBufferSetDataAt(buffer uint64, bufferOffsetInBytes uint64, elementSize, options uint32, startIndex, elementCount uint64, data unsafe.Pointer, capacity uint64) error {
	return resultError("cna_index_buffer_set_data_at", uint32(C.cna_go_index_buffer_set_data_at(
		C.CnaGoHandle(buffer), C.uint64_t(bufferOffsetInBytes), C.uint32_t(elementSize), C.uint32_t(options),
		C.uint64_t(startIndex), C.uint64_t(elementCount), data, C.uint64_t(capacity))))
}

func nativeIndexBufferGetData(buffer uint64, elementSize uint32, startIndex, elementCount uint64, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	var required C.uint64_t
	code := uint32(C.cna_go_index_buffer_get_data(
		C.CnaGoHandle(buffer), C.uint32_t(elementSize),
		C.uint64_t(startIndex), C.uint64_t(elementCount), destination, C.uint64_t(capacity), &required))
	return uint64(required), resultError("cna_index_buffer_get_data", code)
}

// The nine vertex routes. Elements cross as a FLAT int32 array of four fields
// each rather than as a struct, which is the settled boundary rule: no CNA
// struct crosses cgo, and bridge.c is where the four scalars become one
// CNA_VertexElement.

func nativeVertexDeclarationCreate(stride int32, hasStride bool, elements []int32) (uint64, error) {
	var handle C.CnaGoHandle
	flag := C.uint8_t(0)
	if hasStride {
		flag = 1
	}
	var first *C.int32_t
	if len(elements) > 0 {
		first = (*C.int32_t)(unsafe.Pointer(&elements[0]))
	}
	code := uint32(C.cna_go_vertex_declaration_create(
		C.int32_t(stride), flag, first, C.uint64_t(len(elements)/4), &handle))
	runtime.KeepAlive(elements)
	if err := resultError("cna_vertex_declaration_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeVertexDeclarationDestroy(declaration uint64) error {
	return resultError("cna_vertex_declaration_destroy",
		uint32(C.cna_go_vertex_declaration_destroy(C.CnaGoHandle(declaration))))
}

func nativeVertexDeclarationStride(declaration uint64) (int32, error) {
	var stride C.int32_t
	code := uint32(C.cna_go_vertex_declaration_get_stride(C.CnaGoHandle(declaration), &stride))
	return int32(stride), resultError("cna_vertex_declaration_get_stride", code)
}

func nativeVertexBufferCreate(device, declaration uint64, vertexCount int32, usage uint32, dynamic bool) (uint64, error) {
	var handle C.CnaGoHandle
	flag := C.uint8_t(0)
	if dynamic {
		flag = 1
	}
	code := uint32(C.cna_go_vertex_buffer_create(
		C.CnaGoHandle(device), C.CnaGoHandle(declaration), C.int32_t(vertexCount), C.uint32_t(usage), flag, &handle))
	if err := resultError("cna_vertex_buffer_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeVertexBufferDestroy(buffer uint64) error {
	return resultError("cna_vertex_buffer_destroy",
		uint32(C.cna_go_vertex_buffer_destroy(C.CnaGoHandle(buffer))))
}

func nativeVertexBufferInfo(buffer uint64) (VertexBufferInfo, error) {
	var count, stride C.int32_t
	var usage C.uint32_t
	var dynamic, contentLost, hasRenderer C.uint8_t
	var elementCount C.uint64_t
	code := uint32(C.cna_go_vertex_buffer_get_info(
		C.CnaGoHandle(buffer), &count, &usage, &dynamic, &contentLost, &hasRenderer, &stride, &elementCount))
	if err := resultError("cna_vertex_buffer_get_info", code); err != nil {
		return VertexBufferInfo{}, err
	}
	return VertexBufferInfo{
		VertexCount:        int32(count),
		BufferUsage:        uint32(usage),
		Dynamic:            dynamic != 0,
		IsContentLost:      contentLost != 0,
		HasRenderer:        hasRenderer != 0,
		VertexStride:       int32(stride),
		VertexElementCount: uint64(elementCount),
	}, nil
}

func nativeVertexBufferSetDataRawAt(buffer, offset uint64, data unsafe.Pointer, byteCount, vertexCount uint64, stride uint32) error {
	return resultError("cna_vertex_buffer_set_data_raw_at", uint32(C.cna_go_vertex_buffer_set_data_raw_at(
		C.CnaGoHandle(buffer), C.uint64_t(offset), data,
		C.uint64_t(byteCount), C.uint64_t(vertexCount), C.uint32_t(stride))))
}

func nativeVertexBufferGetDataRaw(buffer, offset uint64, destination unsafe.Pointer, byteCount, vertexCount uint64, stride uint32) error {
	return resultError("cna_vertex_buffer_get_data_raw", uint32(C.cna_go_vertex_buffer_get_data_raw(
		C.CnaGoHandle(buffer), C.uint64_t(offset), destination,
		C.uint64_t(byteCount), C.uint64_t(vertexCount), C.uint32_t(stride))))
}

// The six device binding and draw routes.

func nativeDeviceSetVertexBuffers(device uint64, bindings []int64) error {
	var first *C.int64_t
	if len(bindings) > 0 {
		first = (*C.int64_t)(unsafe.Pointer(&bindings[0]))
	}
	code := uint32(C.cna_go_graphics_device_set_vertex_buffers(
		C.CnaGoHandle(device), first, C.uint64_t(len(bindings)/3)))
	runtime.KeepAlive(bindings)
	return resultError("cna_graphics_device_set_vertex_buffers", code)
}

func nativeDeviceSetIndexBuffer(device, buffer uint64) error {
	return resultError("cna_graphics_device_set_index_buffer",
		uint32(C.cna_go_graphics_device_set_index_buffer(C.CnaGoHandle(device), C.CnaGoHandle(buffer))))
}

func nativeDeviceDrawPrimitives(device uint64, primitiveType uint32, vertexStart, primitiveCount int32) error {
	return resultError("cna_graphics_device_draw_primitives",
		uint32(C.cna_go_graphics_device_draw_primitives(
			C.CnaGoHandle(device), C.uint32_t(primitiveType), C.int32_t(vertexStart), C.int32_t(primitiveCount))))
}

func nativeDeviceDrawIndexedPrimitives(device uint64, primitiveType uint32, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount int32) error {
	return resultError("cna_graphics_device_draw_indexed_primitives",
		uint32(C.cna_go_graphics_device_draw_indexed_primitives(
			C.CnaGoHandle(device), C.uint32_t(primitiveType), C.int32_t(baseVertex), C.int32_t(minVertexIndex),
			C.int32_t(numVertices), C.int32_t(startIndex), C.int32_t(primitiveCount))))
}

func nativeDeviceDrawInstancedPrimitives(device uint64, primitiveType uint32, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount, instanceCount int32) error {
	return resultError("cna_graphics_device_draw_instanced_primitives",
		uint32(C.cna_go_graphics_device_draw_instanced_primitives(
			C.CnaGoHandle(device), C.uint32_t(primitiveType), C.int32_t(baseVertex), C.int32_t(minVertexIndex),
			C.int32_t(numVertices), C.int32_t(startIndex), C.int32_t(primitiveCount), C.int32_t(instanceCount))))
}

// The twelve adapter routes. Every one takes the callback-scoped device handle
// CNA requires, so none is reachable outside a lifecycle callback.

func nativeAdapterCount(device uint64) (uint64, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_graphics_adapter_get_count(C.CnaGoHandle(device), &count))
	return uint64(count), resultError("cna_graphics_adapter_get_count", code)
}

func nativeAdapterInfo(device uint64, index uint32) (AdapterInfo, error) {
	var reported C.uint32_t
	var isDefault, wide, nullDevice, referenceDevice C.uint8_t
	var vendor, deviceID, revision, subsystem C.int32_t
	var descriptionBytes, deviceNameBytes C.uint64_t
	code := uint32(C.cna_go_graphics_adapter_get_info(
		C.CnaGoHandle(device), C.uint32_t(index), &reported, &isDefault, &wide,
		&nullDevice, &referenceDevice, &vendor, &deviceID, &revision, &subsystem,
		&descriptionBytes, &deviceNameBytes))
	if err := resultError("cna_graphics_adapter_get_info", code); err != nil {
		return AdapterInfo{}, err
	}
	return AdapterInfo{
		Index:              uint32(reported),
		IsDefaultAdapter:   isDefault != 0,
		IsWideScreen:       wide != 0,
		UseNullDevice:      nullDevice != 0,
		UseReferenceDevice: referenceDevice != 0,
		VendorID:           int32(vendor),
		DeviceID:           int32(deviceID),
		Revision:           int32(revision),
		SubSystemID:        int32(subsystem),
	}, nil
}

func nativeAdapterDescription(device uint64, index uint32, byteCount uint64) (string, error) {
	return nativeAdapterString(device, index, true, byteCount)
}

func nativeAdapterDeviceName(device uint64, index uint32, byteCount uint64) (string, error) {
	return nativeAdapterString(device, index, false, byteCount)
}

// nativeAdapterString takes the byte count from CNA_GraphicsAdapterInfo rather
// than asking the copy route with a zero capacity: that route answers a zero
// capacity with CNA_RESULT 14 rather than with the required count, which is
// what the info structure's two length fields are for.
func nativeAdapterString(device uint64, index uint32, description bool, byteCount uint64) (string, error) {
	name := "cna_graphics_adapter_copy_device_name"
	copyOne := func(buffer *C.char, capacity C.uint64_t, out *C.uint64_t) C.CnaGoResult {
		return C.cna_go_graphics_adapter_copy_device_name(
			C.CnaGoHandle(device), C.uint32_t(index), buffer, capacity, out)
	}
	if description {
		name = "cna_graphics_adapter_copy_description"
		copyOne = func(buffer *C.char, capacity C.uint64_t, out *C.uint64_t) C.CnaGoResult {
			return C.cna_go_graphics_adapter_copy_description(
				C.CnaGoHandle(device), C.uint32_t(index), buffer, capacity, out)
		}
	}
	if byteCount == 0 {
		return "", nil
	}
	required := C.uint64_t(byteCount)
	buffer := make([]byte, int(required))
	code := uint32(copyOne((*C.char)(unsafe.Pointer(&buffer[0])), required, &required))
	if err := resultError(name, code); err != nil {
		return "", err
	}
	return string(buffer[:int(required)]), nil
}

func nativeAdapterCurrentDisplayMode(device uint64, index uint32) (DisplayModeValue, error) {
	var width, height C.int32_t
	var format C.uint32_t
	code := uint32(C.cna_go_graphics_adapter_get_current_display_mode(
		C.CnaGoHandle(device), C.uint32_t(index), &width, &height, &format))
	if err := resultError("cna_graphics_adapter_get_current_display_mode", code); err != nil {
		return DisplayModeValue{}, err
	}
	return DisplayModeValue{Width: int32(width), Height: int32(height), Format: uint32(format)}, nil
}

func nativeAdapterDisplayModes(device uint64, index uint32) ([]DisplayModeValue, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_graphics_adapter_get_display_mode_count(
		C.CnaGoHandle(device), C.uint32_t(index), &count))
	if err := resultError("cna_graphics_adapter_get_display_mode_count", code); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	flattened := make([]int32, int(count)*3)
	var reported C.uint64_t
	code = uint32(C.cna_go_graphics_adapter_copy_display_modes(
		C.CnaGoHandle(device), C.uint32_t(index),
		(*C.int32_t)(unsafe.Pointer(&flattened[0])), count, &reported))
	if err := resultError("cna_graphics_adapter_copy_display_modes", code); err != nil {
		return nil, err
	}
	modes := make([]DisplayModeValue, 0, int(reported))
	for at := 0; at < int(reported) && at*3+2 < len(flattened); at++ {
		modes = append(modes, DisplayModeValue{
			Width:  flattened[at*3+0],
			Height: flattened[at*3+1],
			Format: uint32(flattened[at*3+2]),
		})
	}
	runtime.KeepAlive(flattened)
	return modes, nil
}

func nativeAdapterSetDevicePreferences(device uint64, index uint32, nullDevice, referenceDevice bool) error {
	nullFlag, referenceFlag := C.uint8_t(0), C.uint8_t(0)
	if nullDevice {
		nullFlag = 1
	}
	if referenceDevice {
		referenceFlag = 1
	}
	return resultError("cna_graphics_adapter_set_device_preferences",
		uint32(C.cna_go_graphics_adapter_set_device_preferences(
			C.CnaGoHandle(device), C.uint32_t(index), nullFlag, referenceFlag)))
}

func nativeAdapterIsProfileSupported(device uint64, index, profile uint32) (bool, error) {
	var supported C.uint8_t
	code := uint32(C.cna_go_graphics_adapter_is_profile_supported(
		C.CnaGoHandle(device), C.uint32_t(index), C.uint32_t(profile), &supported))
	return supported != 0, resultError("cna_graphics_adapter_is_profile_supported", code)
}

func nativeAdapterQueryFormat(device uint64, index uint32, renderTarget bool, profile, format, depthFormat uint32, multiSampleCount int32) (FormatSelection, error) {
	target := C.uint8_t(0)
	if renderTarget {
		target = 1
	}
	var exact C.uint8_t
	var selectedFormat, selectedDepth C.uint32_t
	var selectedSamples C.int32_t
	code := uint32(C.cna_go_graphics_adapter_query_format(
		C.CnaGoHandle(device), C.uint32_t(index), target, C.uint32_t(profile), C.uint32_t(format),
		C.uint32_t(depthFormat), C.int32_t(multiSampleCount),
		&exact, &selectedFormat, &selectedDepth, &selectedSamples))
	if err := resultError("cna_graphics_adapter_query_format", code); err != nil {
		return FormatSelection{}, err
	}
	return FormatSelection{
		ExactMatch:       exact != 0,
		Format:           uint32(selectedFormat),
		DepthFormat:      uint32(selectedDepth),
		MultiSampleCount: int32(selectedSamples),
	}, nil
}

func nativeAdapterMonitorHandle(device uint64, index uint32) (uint64, error) {
	var value C.uint64_t
	code := uint32(C.cna_go_graphics_adapter_get_native_monitor_handle(
		C.CnaGoHandle(device), C.uint32_t(index), &value))
	return uint64(value), resultError("cna_graphics_adapter_get_native_monitor_handle", code)
}

func nativeDeviceAdapterIndex(device uint64) (uint32, error) {
	var index C.uint32_t
	code := uint32(C.cna_go_graphics_device_get_adapter_index(C.CnaGoHandle(device), &index))
	return uint32(index), resultError("cna_graphics_device_get_adapter_index", code)
}
