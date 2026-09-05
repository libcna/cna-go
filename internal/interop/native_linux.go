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

// nativeSpriteBatchBeginWithEffect is the same four descriptors plus the effect
// and the transform. It repeats the flattening rather than sharing it, because
// the two routes take DIFFERENT canonical prototypes and one shared builder
// would make a prototype change on either side invisible.
func nativeSpriteBatchBeginWithEffect(
	batch uint64, sortMode uint32, blend BlendStateValue, sampler SamplerStateValue,
	depth DepthStencilStateValue, rasterizer RasterizerStateValue,
	effect uint64, transform *[16]float32,
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
	var matrix [16]C.float
	var matrixPointer *C.float
	hasTransform := C.uint8_t(0)
	if transform != nil {
		for index := range transform {
			matrix[index] = C.float(transform[index])
		}
		matrixPointer = &matrix[0]
		hasTransform = 1
	}
	return resultError("cna_sprite_batch_begin_with_effect", uint32(C.cna_go_sprite_batch_begin_with_effect(
		C.CnaGoHandle(batch), C.uint32_t(sortMode),
		&blendWords[0], &blendMask[0], &blendFactor[0],
		&samplerWords[0], &samplerInts[0], C.float(sampler.MipMapLevelOfDetailBias),
		&depthFlags[0], &depthWords[0], &depthInts[0],
		C.uint32_t(rasterizer.CullMode), C.uint32_t(rasterizer.FillMode),
		C.float(rasterizer.DepthBias), C.float(rasterizer.SlopeScaleDepthBias),
		cnaBool(rasterizer.MultiSampleAntiAlias), cnaBool(rasterizer.ScissorTestEnable),
		C.CnaGoHandle(effect), hasTransform, matrixPointer)))
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

// The four cube render-target routes. Foundation 73.

func nativeRenderTargetCubeCreate(device uint64, size uint32, mipMap bool, format, depthFormat uint32, multiSampleCount int32, usage uint32) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_render_target_cube_create(
		C.CnaGoHandle(device), C.uint32_t(size), cnaBool(mipMap), C.uint32_t(format),
		C.uint32_t(depthFormat), C.int32_t(multiSampleCount), C.uint32_t(usage), &handle))
	if err := resultError("cna_render_target_cube_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeGraphicsDeviceSetRenderTargetCube(device, renderTarget uint64, face uint32) error {
	return resultError("cna_graphics_device_set_render_target_cube",
		uint32(C.cna_go_graphics_device_set_render_target_cube(
			C.CnaGoHandle(device), C.CnaGoHandle(renderTarget), C.uint32_t(face))))
}

func nativeGraphicsDeviceSetRenderTargets(device uint64, handles []uint64, faces []uint32) error {
	if len(handles) == 0 {
		return resultError("cna_graphics_device_set_render_targets",
			uint32(C.cna_go_graphics_device_set_render_targets(C.CnaGoHandle(device), nil, nil, 0)))
	}
	nativeHandles := make([]C.CnaGoHandle, len(handles))
	nativeFaces := make([]C.uint32_t, len(handles))
	for index := range handles {
		nativeHandles[index] = C.CnaGoHandle(handles[index])
		nativeFaces[index] = C.uint32_t(faces[index])
	}
	code := uint32(C.cna_go_graphics_device_set_render_targets(
		C.CnaGoHandle(device), &nativeHandles[0], &nativeFaces[0], C.uint64_t(len(handles))))
	runtime.KeepAlive(nativeHandles)
	runtime.KeepAlive(nativeFaces)
	return resultError("cna_graphics_device_set_render_targets", code)
}

func nativeGraphicsDeviceRenderTargetCount(device uint64) (uint64, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_graphics_device_get_render_target_count(C.CnaGoHandle(device), &count))
	return uint64(count), resultError("cna_graphics_device_get_render_target_count", code)
}

// Device reset, presentation parameters and back-buffer readback.
// Foundation 73.

// presentationIntCount is the number of int32-shaped presentation fields the
// bridge exchanges, in the order bridge.c's CNA_GO_PRESENTATION_* indices name.
const presentationIntCount = 8

// PresentationValue is CNA_PresentationParameters, flattened. The two CNA_Bools
// stay Go bools and everything else is an int32, which is what the eight-slot
// array the bridge takes carries.
type PresentationValue struct {
	BackBufferFormat     int32
	BackBufferWidth      int32
	BackBufferHeight     int32
	DepthStencilFormat   int32
	MultiSampleCount     int32
	PresentationInterval int32
	DisplayOrientation   int32
	RenderTargetUsage    int32
	IsFullScreen         bool
	Headless             bool
}

func (v PresentationValue) ints() [presentationIntCount]C.int32_t {
	return [presentationIntCount]C.int32_t{
		C.int32_t(v.BackBufferFormat), C.int32_t(v.BackBufferWidth), C.int32_t(v.BackBufferHeight),
		C.int32_t(v.DepthStencilFormat), C.int32_t(v.MultiSampleCount),
		C.int32_t(v.PresentationInterval), C.int32_t(v.DisplayOrientation),
		C.int32_t(v.RenderTargetUsage),
	}
}

func nativeGraphicsDeviceCreate(adapterIndex, profile uint32, value PresentationValue) (uint64, error) {
	ints := value.ints()
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_graphics_device_create(
		C.uint32_t(adapterIndex), C.uint32_t(profile), &ints[0],
		cnaBool(value.IsFullScreen), cnaBool(value.Headless), &handle))
	if err := resultError("cna_graphics_device_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeGraphicsDeviceDestroy(device uint64) error {
	return resultError("cna_graphics_device_destroy",
		uint32(C.cna_go_graphics_device_destroy(C.CnaGoHandle(device))))
}

func nativeGraphicsDeviceReset(device uint64) error {
	return resultError("cna_graphics_device_reset",
		uint32(C.cna_go_graphics_device_reset(C.CnaGoHandle(device))))
}

func nativeGraphicsDeviceResetWithParameters(device uint64, value PresentationValue, adapterIndex *uint32) error {
	ints := value.ints()
	hasAdapter := C.uint8_t(0)
	index := C.uint32_t(0)
	if adapterIndex != nil {
		hasAdapter = 1
		index = C.uint32_t(*adapterIndex)
	}
	return resultError("cna_graphics_device_reset_with_parameters",
		uint32(C.cna_go_graphics_device_reset_with_parameters(
			C.CnaGoHandle(device), &ints[0], cnaBool(value.IsFullScreen), cnaBool(value.Headless),
			hasAdapter, index)))
}

func nativeGraphicsDevicePresentationParameters(device uint64) (PresentationValue, error) {
	var ints [presentationIntCount]C.int32_t
	var fullScreen, headless C.uint8_t
	code := uint32(C.cna_go_graphics_device_get_presentation_parameters(
		C.CnaGoHandle(device), &ints[0], &fullScreen, &headless))
	if err := resultError("cna_graphics_device_get_presentation_parameters", code); err != nil {
		return PresentationValue{}, err
	}
	return PresentationValue{
		BackBufferFormat:     int32(ints[0]),
		BackBufferWidth:      int32(ints[1]),
		BackBufferHeight:     int32(ints[2]),
		DepthStencilFormat:   int32(ints[3]),
		MultiSampleCount:     int32(ints[4]),
		PresentationInterval: int32(ints[5]),
		DisplayOrientation:   int32(ints[6]),
		RenderTargetUsage:    int32(ints[7]),
		IsFullScreen:         fullScreen != 0,
		Headless:             headless != 0,
	}, nil
}

func nativeGraphicsDeviceBackBufferData(
	device uint64, hasRectangle bool, x, y, width, height int32,
	startIndex, elementCount uint64, destination unsafe.Pointer, capacity uint64,
) error {
	return resultError("cna_graphics_device_get_backbuffer_data_window",
		uint32(C.cna_go_graphics_device_get_backbuffer_data_window(
			C.CnaGoHandle(device), cnaBool(hasRectangle),
			C.int32_t(x), C.int32_t(y), C.int32_t(width), C.int32_t(height),
			C.uint64_t(startIndex), C.uint64_t(elementCount), destination, C.uint64_t(capacity))))
}

// The two user-primitive routes. Foundation 73.
//
// Both take a RAW vertex stream and an explicit declaration: CNA's four TYPED
// vertex sources name CNA_VertexPositionColor and its three siblings, which the
// Graphics package has no Go types for, and a raw stream with a declaration
// expresses every layout those four do and more.

func nativeDrawUserPrimitives(
	device uint64, primitiveType, vertexSource uint32, vertexData unsafe.Pointer,
	declaration uint64, vertexOffset, numVertices, primitiveCount int32,
) error {
	return resultError("cna_graphics_device_draw_user_primitives",
		uint32(C.cna_go_graphics_device_draw_user_primitives(
			C.CnaGoHandle(device), C.uint32_t(primitiveType), C.uint32_t(vertexSource),
			vertexData, C.CnaGoHandle(declaration),
			C.int32_t(vertexOffset), C.int32_t(numVertices), C.int32_t(primitiveCount))))
}

func nativeDrawUserIndexedPrimitives(
	device uint64, primitiveType, vertexSource uint32, vertexData unsafe.Pointer,
	declaration uint64, vertexOffset, numVertices, primitiveCount int32,
	indexElementSize uint32, indexOffset int32, indexData unsafe.Pointer,
) error {
	return resultError("cna_graphics_device_draw_user_indexed_primitives",
		uint32(C.cna_go_graphics_device_draw_user_indexed_primitives(
			C.CnaGoHandle(device), C.uint32_t(primitiveType), C.uint32_t(vertexSource),
			vertexData, C.CnaGoHandle(declaration),
			C.int32_t(vertexOffset), C.int32_t(numVertices), C.int32_t(primitiveCount),
			C.uint32_t(indexElementSize), C.int32_t(indexOffset), indexData)))
}

// The Effect cluster. Foundation 72.
//
// Every accessor in CNA's effect API returns a FRESH owned view handle -- the
// probe measured two cna_effect_get_parameters calls answering two different
// handles -- while the reference's get_Parameters is one `ldfld` that answers
// the same object forever. So the Graphics package caches each view exactly
// once and this layer never calls an accessor twice for one logical object.

func nativeEffectCreateCompiled(device uint64, code []byte) (uint64, error) {
	var handle C.CnaGoHandle
	var data *C.uint8_t
	if len(code) > 0 {
		data = (*C.uint8_t)(unsafe.Pointer(&code[0]))
	}
	code2 := uint32(C.cna_go_effect_create_compiled(
		C.CnaGoHandle(device), data, C.uint64_t(len(code)), &handle))
	runtime.KeepAlive(code)
	if err := resultError("cna_effect_create_compiled", code2); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeContentManagerLoadEffect(manager uint64, assetName string) (uint64, error) {
	var handle C.CnaGoHandle
	var data *C.char
	if len(assetName) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(assetName)))
	}
	code := uint32(C.cna_go_content_manager_load_effect(
		C.CnaGoHandle(manager), data, C.uint64_t(len(assetName)), &handle))
	runtime.KeepAlive(assetName)
	if err := resultError("cna_content_manager_load_effect", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

// The eight effect string reads, through one trampoline selected by kind. The
// route NAME reported on failure is the canonical one for that kind, so a
// caller reading an error learns which CNA route refused rather than learning
// the name of CNA-Go's own multiplexer.
const (
	effectStringTechniqueName uint32 = iota
	effectStringPassName
	effectStringParameterName
	effectStringParameterSemantic
	effectStringParameterValue
	effectStringAnnotationName
	effectStringAnnotationSemantic
	effectStringAnnotationValue
)

var effectStringRoutes = [...]string{
	"cna_effect_technique_copy_name",
	"cna_effect_pass_copy_name",
	"cna_effect_parameter_copy_name",
	"cna_effect_parameter_copy_semantic",
	"cna_effect_parameter_copy_value_string",
	"cna_effect_annotation_copy_name",
	"cna_effect_annotation_copy_semantic",
	"cna_effect_annotation_copy_value_string",
}

func nativeEffectString(kind uint32, handle uint64) (string, error) {
	route := "cna_effect_string"
	if int(kind) < len(effectStringRoutes) {
		route = effectStringRoutes[kind]
	}
	var byteCount C.uint64_t
	code := uint32(C.cna_go_effect_string(C.uint32_t(kind), C.CnaGoHandle(handle), nil, 0, &byteCount))
	if err := resultError(route, code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_effect_string(C.uint32_t(kind), C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError(route, code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

// handleOut is the shape most of this cluster takes: one input handle, one
// output handle. The route name is passed so a refusal names the CNA route.
func nativeEffectHandleOut(route string, call func(*C.CnaGoHandle) C.CnaGoResult) (uint64, error) {
	var handle C.CnaGoHandle
	if err := resultError(route, uint32(call(&handle))); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeEffectApply(effect uint64) error {
	return resultError("cna_effect_apply", uint32(C.cna_go_effect_apply(C.CnaGoHandle(effect))))
}

func nativeEffectDestroy(effect uint64) error {
	return resultError("cna_effect_destroy", uint32(C.cna_go_effect_destroy(C.CnaGoHandle(effect))))
}

func nativeEffectClone(effect uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_clone", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_clone(C.CnaGoHandle(effect), out)
	})
}

func nativeEffectParameters(effect uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_get_parameters", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_get_parameters(C.CnaGoHandle(effect), out)
	})
}

func nativeEffectTechniques(effect uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_get_techniques", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_get_techniques(C.CnaGoHandle(effect), out)
	})
}

func nativeEffectCurrentTechnique(effect uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_get_current_technique", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_get_current_technique(C.CnaGoHandle(effect), out)
	})
}

func nativeEffectSetCurrentTechnique(effect, technique uint64) error {
	return resultError("cna_effect_set_current_technique",
		uint32(C.cna_go_effect_set_current_technique(C.CnaGoHandle(effect), C.CnaGoHandle(technique))))
}

func nativeEffectCollectionCount(route string, kind uint32, collection uint64) (uint64, error) {
	var count C.uint64_t
	var code C.CnaGoResult
	switch kind {
	case effectCollectionTechnique:
		code = C.cna_go_effect_technique_collection_get_count(C.CnaGoHandle(collection), &count)
	case effectCollectionPass:
		code = C.cna_go_effect_pass_collection_get_count(C.CnaGoHandle(collection), &count)
	case effectCollectionParameter:
		code = C.cna_go_effect_parameter_collection_get_count(C.CnaGoHandle(collection), &count)
	default:
		code = C.cna_go_effect_annotation_collection_get_count(C.CnaGoHandle(collection), &count)
	}
	if err := resultError(route, uint32(code)); err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func nativeEffectCollectionAt(route string, kind uint32, collection, index uint64) (uint64, error) {
	var handle C.CnaGoHandle
	var code C.CnaGoResult
	switch kind {
	case effectCollectionTechnique:
		code = C.cna_go_effect_technique_collection_get_at(C.CnaGoHandle(collection), C.uint64_t(index), &handle)
	case effectCollectionPass:
		code = C.cna_go_effect_pass_collection_get_at(C.CnaGoHandle(collection), C.uint64_t(index), &handle)
	case effectCollectionParameter:
		code = C.cna_go_effect_parameter_collection_get_at(C.CnaGoHandle(collection), C.uint64_t(index), &handle)
	default:
		code = C.cna_go_effect_annotation_collection_get_at(C.CnaGoHandle(collection), C.uint64_t(index), &handle)
	}
	if err := resultError(route, uint32(code)); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeEffectTechniquePasses(technique uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_technique_get_passes", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_technique_get_passes(C.CnaGoHandle(technique), out)
	})
}

func nativeEffectTechniqueAnnotations(technique uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_technique_get_annotations", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_technique_get_annotations(C.CnaGoHandle(technique), out)
	})
}

func nativeEffectPassAnnotations(pass uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_pass_get_annotations", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_pass_get_annotations(C.CnaGoHandle(pass), out)
	})
}

func nativeEffectPassApply(pass uint64) error {
	return resultError("cna_effect_pass_apply", uint32(C.cna_go_effect_pass_apply(C.CnaGoHandle(pass))))
}

func nativeEffectParameterElements(parameter uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_parameter_get_elements", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_parameter_get_elements(C.CnaGoHandle(parameter), out)
	})
}

func nativeEffectParameterStructureMembers(parameter uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_parameter_get_structure_members", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_parameter_get_structure_members(C.CnaGoHandle(parameter), out)
	})
}

func nativeEffectParameterAnnotations(parameter uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_parameter_get_annotations", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_parameter_get_annotations(C.CnaGoHandle(parameter), out)
	})
}

func nativeEffectParameterInfo(parameter uint64) (EffectMetadata, error) {
	var rows, columns C.int32_t
	var class, kind C.uint32_t
	code := uint32(C.cna_go_effect_parameter_get_info(
		C.CnaGoHandle(parameter), &rows, &columns, &class, &kind))
	if err := resultError("cna_effect_parameter_get_info", code); err != nil {
		return EffectMetadata{}, err
	}
	return EffectMetadata{
		RowCount: int32(rows), ColumnCount: int32(columns),
		ParameterClass: uint32(class), ParameterType: uint32(kind),
	}, nil
}

func nativeEffectAnnotationInfo(annotation uint64) (EffectMetadata, error) {
	var rows, columns C.int32_t
	var class, kind C.uint32_t
	code := uint32(C.cna_go_effect_annotation_get_info(
		C.CnaGoHandle(annotation), &rows, &columns, &class, &kind))
	if err := resultError("cna_effect_annotation_get_info", code); err != nil {
		return EffectMetadata{}, err
	}
	return EffectMetadata{
		RowCount: int32(rows), ColumnCount: int32(columns),
		ParameterClass: uint32(class), ParameterType: uint32(kind),
	}, nil
}

func nativeEffectParameterGetValue(parameter uint64, valueType uint32, out unsafe.Pointer) error {
	return resultError("cna_effect_parameter_get_value", uint32(C.cna_go_effect_parameter_get_value(
		C.CnaGoHandle(parameter), C.uint32_t(valueType), out)))
}

func nativeEffectParameterSetValue(parameter uint64, valueType uint32, value unsafe.Pointer) error {
	return resultError("cna_effect_parameter_set_value", uint32(C.cna_go_effect_parameter_set_value(
		C.CnaGoHandle(parameter), C.uint32_t(valueType), value)))
}

func nativeEffectParameterGetValues(parameter uint64, valueType uint32, requested uint64, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_effect_parameter_get_values(
		C.CnaGoHandle(parameter), C.uint32_t(valueType), C.uint64_t(requested),
		destination, C.uint64_t(capacity), &count))
	if err := resultError("cna_effect_parameter_get_values", code); err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func nativeEffectParameterSetValues(parameter uint64, valueType uint32, values unsafe.Pointer, count uint64) error {
	return resultError("cna_effect_parameter_set_values", uint32(C.cna_go_effect_parameter_set_values(
		C.CnaGoHandle(parameter), C.uint32_t(valueType), values, C.uint64_t(count))))
}

func nativeEffectParameterSetValueString(parameter uint64, value string) error {
	var data *C.char
	if len(value) > 0 {
		data = (*C.char)(unsafe.Pointer(unsafe.StringData(value)))
	}
	code := uint32(C.cna_go_effect_parameter_set_value_string(
		C.CnaGoHandle(parameter), data, C.uint64_t(len(value))))
	runtime.KeepAlive(value)
	return resultError("cna_effect_parameter_set_value_string", code)
}

func nativeEffectParameterSetValueTexture(parameter uint64, textureType uint32, texture uint64) error {
	return resultError("cna_effect_parameter_set_value_texture",
		uint32(C.cna_go_effect_parameter_set_value_texture(
			C.CnaGoHandle(parameter), C.uint32_t(textureType), C.CnaGoHandle(texture))))
}

func nativeEffectAnnotationBoolean(annotation uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_effect_annotation_get_value_boolean(C.CnaGoHandle(annotation), &value))
	return value != 0, resultError("cna_effect_annotation_get_value_boolean", code)
}

func nativeEffectAnnotationInt32(annotation uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_effect_annotation_get_value_int32(C.CnaGoHandle(annotation), &value))
	return int32(value), resultError("cna_effect_annotation_get_value_int32", code)
}

func nativeEffectAnnotationSingle(annotation uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_effect_annotation_get_value_single(C.CnaGoHandle(annotation), &value))
	return float32(value), resultError("cna_effect_annotation_get_value_single", code)
}

func nativeEffectAnnotationVector(annotation uint64, width uint32) ([4]float32, error) {
	var values [4]C.float
	code := uint32(C.cna_go_effect_annotation_get_value_vector(
		C.CnaGoHandle(annotation), C.uint32_t(width), &values[0]))
	route := "cna_effect_annotation_get_value_vector4"
	switch width {
	case 2:
		route = "cna_effect_annotation_get_value_vector2"
	case 3:
		route = "cna_effect_annotation_get_value_vector3"
	}
	var result [4]float32
	for index := range values {
		result[index] = float32(values[index])
	}
	return result, resultError(route, code)
}

func nativeEffectAnnotationMatrix(annotation uint64) ([16]float32, error) {
	var values [16]C.float
	code := uint32(C.cna_go_effect_annotation_get_value_matrix(C.CnaGoHandle(annotation), &values[0]))
	var result [16]float32
	for index := range values {
		result[index] = float32(values[index])
	}
	return result, resultError("cna_effect_annotation_get_value_matrix", code)
}

func nativeEffectViewDestroy(kind uint32, handle uint64) error {
	switch kind {
	case effectViewTechniqueCollection:
		return resultError("cna_effect_technique_collection_destroy",
			uint32(C.cna_go_effect_technique_collection_destroy(C.CnaGoHandle(handle))))
	case effectViewTechnique:
		return resultError("cna_effect_technique_destroy",
			uint32(C.cna_go_effect_technique_destroy(C.CnaGoHandle(handle))))
	case effectViewPassCollection:
		return resultError("cna_effect_pass_collection_destroy",
			uint32(C.cna_go_effect_pass_collection_destroy(C.CnaGoHandle(handle))))
	case effectViewPass:
		return resultError("cna_effect_pass_destroy",
			uint32(C.cna_go_effect_pass_destroy(C.CnaGoHandle(handle))))
	case effectViewParameterCollection:
		return resultError("cna_effect_parameter_collection_destroy",
			uint32(C.cna_go_effect_parameter_collection_destroy(C.CnaGoHandle(handle))))
	case effectViewParameter:
		return resultError("cna_effect_parameter_destroy",
			uint32(C.cna_go_effect_parameter_destroy(C.CnaGoHandle(handle))))
	case effectViewAnnotationCollection:
		return resultError("cna_effect_annotation_collection_destroy",
			uint32(C.cna_go_effect_annotation_collection_destroy(C.CnaGoHandle(handle))))
	default:
		return resultError("cna_effect_annotation_destroy",
			uint32(C.cna_go_effect_annotation_destroy(C.CnaGoHandle(handle))))
	}
}

// The ten volume/cube texture routes. Foundation 71.
//
// Both transfer routes take CNA_Color elements only, so the pointer crossing
// here is an unsafe.Pointer to a caller array of framework.Color -- the one
// element type CNA's cube and volume transfers can express. The Graphics
// package is where any other type is refused.

func nativeTexture3DCreate(device uint64, width, height, depth uint32, mipMap bool, format uint32) (uint64, error) {
	var handle C.CnaGoHandle
	flag := C.uint8_t(0)
	if mipMap {
		flag = 1
	}
	code := uint32(C.cna_go_texture3d_create(
		C.CnaGoHandle(device), C.uint32_t(width), C.uint32_t(height), C.uint32_t(depth),
		flag, C.uint32_t(format), &handle))
	if err := resultError("cna_texture3d_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeTexture3DDestroy(texture uint64) error {
	return resultError("cna_texture3d_destroy",
		uint32(C.cna_go_texture3d_destroy(C.CnaGoHandle(texture))))
}

func nativeTexture3DInfo(texture uint64) (Texture3DInfo, error) {
	var width, height, depth, levels, format C.uint32_t
	code := uint32(C.cna_go_texture3d_get_info(
		C.CnaGoHandle(texture), &width, &height, &depth, &levels, &format))
	if err := resultError("cna_texture3d_get_info", code); err != nil {
		return Texture3DInfo{}, err
	}
	return Texture3DInfo{
		Width: uint32(width), Height: uint32(height), Depth: uint32(depth),
		Levels: uint32(levels), Format: uint32(format),
	}, nil
}

func nativeTexture3DSetData(texture uint64, transfer Texture3DTransfer, data unsafe.Pointer, capacity uint64) error {
	return resultError("cna_texture3d_set_data", uint32(C.cna_go_texture3d_set_data(
		C.CnaGoHandle(texture), C.int32_t(transfer.Level),
		C.int32_t(transfer.Left), C.int32_t(transfer.Top), C.int32_t(transfer.Right),
		C.int32_t(transfer.Bottom), C.int32_t(transfer.Front), C.int32_t(transfer.Back),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount), data, C.uint64_t(capacity))))
}

func nativeTexture3DGetData(texture uint64, transfer Texture3DTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	var required C.uint64_t
	code := uint32(C.cna_go_texture3d_get_data(
		C.CnaGoHandle(texture), C.int32_t(transfer.Level),
		C.int32_t(transfer.Left), C.int32_t(transfer.Top), C.int32_t(transfer.Right),
		C.int32_t(transfer.Bottom), C.int32_t(transfer.Front), C.int32_t(transfer.Back),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount),
		destination, C.uint64_t(capacity), &required))
	return uint64(required), resultError("cna_texture3d_get_data", code)
}

func nativeTextureCubeCreate(device uint64, size uint32, mipMap bool, format uint32) (uint64, error) {
	var handle C.CnaGoHandle
	flag := C.uint8_t(0)
	if mipMap {
		flag = 1
	}
	code := uint32(C.cna_go_texturecube_create(
		C.CnaGoHandle(device), C.uint32_t(size), flag, C.uint32_t(format), &handle))
	if err := resultError("cna_texturecube_create", code); err != nil {
		return 0, err
	}
	return uint64(handle), nil
}

func nativeTextureCubeDestroy(texture uint64) error {
	return resultError("cna_texturecube_destroy",
		uint32(C.cna_go_texturecube_destroy(C.CnaGoHandle(texture))))
}

func nativeTextureCubeInfo(texture uint64) (TextureCubeInfo, error) {
	var size, levels, format C.uint32_t
	code := uint32(C.cna_go_texturecube_get_info(C.CnaGoHandle(texture), &size, &levels, &format))
	if err := resultError("cna_texturecube_get_info", code); err != nil {
		return TextureCubeInfo{}, err
	}
	return TextureCubeInfo{Size: uint32(size), Levels: uint32(levels), Format: uint32(format)}, nil
}

func nativeTextureCubeSetData(texture uint64, transfer TextureCubeTransfer, data unsafe.Pointer, capacity uint64) error {
	rectangle := C.uint8_t(0)
	if transfer.HasRectangle {
		rectangle = 1
	}
	return resultError("cna_texturecube_set_data", uint32(C.cna_go_texturecube_set_data(
		C.CnaGoHandle(texture), C.uint32_t(transfer.Face), C.int32_t(transfer.Level), rectangle,
		C.int32_t(transfer.X), C.int32_t(transfer.Y), C.int32_t(transfer.Width), C.int32_t(transfer.Height),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount), data, C.uint64_t(capacity))))
}

func nativeTextureCubeGetData(texture uint64, transfer TextureCubeTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	rectangle := C.uint8_t(0)
	if transfer.HasRectangle {
		rectangle = 1
	}
	var required C.uint64_t
	code := uint32(C.cna_go_texturecube_get_data(
		C.CnaGoHandle(texture), C.uint32_t(transfer.Face), C.int32_t(transfer.Level), rectangle,
		C.int32_t(transfer.X), C.int32_t(transfer.Y), C.int32_t(transfer.Width), C.int32_t(transfer.Height),
		C.uint64_t(transfer.StartIndex), C.uint64_t(transfer.ElementCount),
		destination, C.uint64_t(capacity), &required))
	return uint64(required), resultError("cna_texturecube_get_data", code)
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

// ---------------------------------------------------------------------------
// Foundation 89 -- the Input family's static readers.
//
// Every route takes the GAME handle and nothing owns anything: a gamepad, a
// mouse and a touch panel are properties of the running game rather than
// objects with lifetimes, which is the reference's shape too -- all five
// projected types are `abstract sealed` static classes or value structs.
// ---------------------------------------------------------------------------

// GamePadCapabilitiesFlagCount is the number of boolean capabilities XNA
// declares. CNA carries eleven more `_ext` flags with no XNA counterpart and
// they are not copied.
const GamePadCapabilitiesFlagCount = 25

// nativeGamePadCapabilities is cna_gamepad_get_capabilities. The flags cross as
// a flat byte array in the order GamePadCapabilities declares its properties,
// so a thirty-eight-argument cgo signature never exists.
func nativeGamePadCapabilities(game uint64, playerIndex uint32) (uint32, [GamePadCapabilitiesFlagCount]byte, error) {
	var padType C.uint32_t
	var flags [GamePadCapabilitiesFlagCount]C.uint8_t
	code := uint32(C.cna_go_gamepad_get_capabilities(C.CnaGoHandle(game),
		C.uint32_t(playerIndex), &padType, &flags[0]))
	var out [GamePadCapabilitiesFlagCount]byte
	for index := range flags {
		out[index] = byte(flags[index])
	}
	return uint32(padType), out, resultError("cna_gamepad_get_capabilities", code)
}

func nativeGamePadSetVibration(game uint64, playerIndex uint32, left, right float32) (bool, error) {
	var applied C.uint8_t
	code := uint32(C.cna_go_gamepad_set_vibration(C.CnaGoHandle(game), C.uint32_t(playerIndex),
		C.float(left), C.float(right), &applied))
	return applied != 0, resultError("cna_gamepad_set_vibration", code)
}

// GamePadStateValues is CNA_GamePadState reduced to what XNA's GamePadState
// carries. CNA's packet_number has no XNA counterpart and is kept because the
// reference's XINPUT_STATE has one too -- it is what a caller would use to tell
// a repeated poll from a new one, and nothing public reports it.
type GamePadStateValues struct {
	IsConnected    bool
	PacketNumber   int32
	PressedButtons uint32
	LeftThumbX     float32
	LeftThumbY     float32
	RightThumbX    float32
	RightThumbY    float32
	LeftTrigger    float32
	RightTrigger   float32
}

// nativeGamePadState is cna_gamepad_get_state or its dead-zone-carrying
// sibling. ONE wrapper serves both, because XNA's two GetState overloads differ
// only in whether the caller names a dead zone -- the one-argument overload
// forwards with GamePadDeadZone.IndependentAxes.
func nativeGamePadState(game uint64, playerIndex uint32, deadZone uint32, hasDeadZone bool) (GamePadStateValues, error) {
	var connected C.uint8_t
	var packet C.int32_t
	var buttons C.uint32_t
	var analog [6]C.float
	flag := C.uint8_t(0)
	if hasDeadZone {
		flag = 1
	}
	route := "cna_gamepad_get_state"
	if hasDeadZone {
		route = "cna_gamepad_get_state_with_dead_zone"
	}
	code := uint32(C.cna_go_gamepad_get_state(C.CnaGoHandle(game), C.uint32_t(playerIndex),
		flag, C.uint32_t(deadZone), &connected, &packet, &buttons, &analog[0]))
	return GamePadStateValues{
		IsConnected:    connected != 0,
		PacketNumber:   int32(packet),
		PressedButtons: uint32(buttons),
		LeftThumbX:     float32(analog[0]),
		LeftThumbY:     float32(analog[1]),
		RightThumbX:    float32(analog[2]),
		RightThumbY:    float32(analog[3]),
		LeftTrigger:    float32(analog[4]),
		RightTrigger:   float32(analog[5]),
	}, resultError(route, code)
}

// MouseStateValues is CNA_MouseState reduced to the five fields it carries.
type MouseStateValues struct {
	X                     int32
	Y                     int32
	ScrollWheel           int32
	HorizontalScrollWheel int32
	PressedButtons        uint32
}

func nativeMouseState(game uint64) (MouseStateValues, error) {
	var ints [4]C.int32_t
	var buttons C.uint32_t
	code := uint32(C.cna_go_mouse_get_state(C.CnaGoHandle(game), &ints[0], &buttons))
	return MouseStateValues{
		X:                     int32(ints[0]),
		Y:                     int32(ints[1]),
		ScrollWheel:           int32(ints[2]),
		HorizontalScrollWheel: int32(ints[3]),
		PressedButtons:        uint32(buttons),
	}, resultError("cna_mouse_get_state", code)
}

func nativeMouseSetPosition(game uint64, x, y int32) error {
	return resultError("cna_mouse_set_position",
		uint32(C.cna_go_mouse_set_position(C.CnaGoHandle(game), C.int32_t(x), C.int32_t(y))))
}

func nativeMouseWindowHandle(game uint64) (uint64, error) {
	var window C.uint64_t
	code := uint32(C.cna_go_mouse_get_window_handle(C.CnaGoHandle(game), &window))
	return uint64(window), resultError("cna_mouse_get_window_handle", code)
}

func nativeMouseSetWindowHandle(game, window uint64) error {
	return resultError("cna_mouse_set_window_handle",
		uint32(C.cna_go_mouse_set_window_handle(C.CnaGoHandle(game), C.uint64_t(window))))
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

// ---------------------------------------------------------------------------
// Foundation 79 -- the stock-effect routes.
//
// Every one of these is a single CNA call over an effect handle or a
// directional-light handle. A Vector3 crosses as three floats and a Matrix as
// sixteen, which is the same flat-array marshalling the effect annotation
// readers already use: the Go side never depends on a C struct layout, and the
// bridge does the one memcpy.
// ---------------------------------------------------------------------------

func nativeBasicEffectSetVertexColorEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_basic_effect_set_vertex_color_enabled", uint32(C.cna_go_basic_effect_set_vertex_color_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeBasicEffectSetPreferPerPixelLighting(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_basic_effect_set_prefer_per_pixel_lighting", uint32(C.cna_go_basic_effect_set_prefer_per_pixel_lighting(C.CnaGoHandle(handle), raw)))
}

func nativeBasicEffectSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_basic_effect_set_diffuse_color", uint32(C.cna_go_basic_effect_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeBasicEffectSetEmissiveColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_basic_effect_set_emissive_color", uint32(C.cna_go_basic_effect_set_emissive_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeBasicEffectSpecularColor(handle uint64) ([3]float32, error) {
	var values [3]C.float
	code := uint32(C.cna_go_basic_effect_get_specular_color(C.CnaGoHandle(handle), &values[0]))
	var result [3]float32
	for index := range values {
		result[index] = float32(values[index])
	}
	return result, resultError("cna_basic_effect_get_specular_color", code)
}

func nativeBasicEffectSetSpecularColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_basic_effect_set_specular_color", uint32(C.cna_go_basic_effect_set_specular_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeBasicEffectSpecularPower(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_basic_effect_get_specular_power(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_basic_effect_get_specular_power", code)
}

func nativeBasicEffectSetSpecularPower(handle uint64, value float32) error {
	return resultError("cna_basic_effect_set_specular_power", uint32(C.cna_go_basic_effect_set_specular_power(C.CnaGoHandle(handle), C.float(value))))
}

func nativeBasicEffectSetAlpha(handle uint64, value float32) error {
	return resultError("cna_basic_effect_set_alpha", uint32(C.cna_go_basic_effect_set_alpha(C.CnaGoHandle(handle), C.float(value))))
}

func nativeBasicEffectSetTextureEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_basic_effect_set_texture_enabled", uint32(C.cna_go_basic_effect_set_texture_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeEffectMatricesSetWorld(handle uint64, value [16]float32) error {
	var values [16]C.float
	for index := range value {
		values[index] = C.float(value[index])
	}
	return resultError("cna_effect_matrices_set_world", uint32(C.cna_go_effect_matrices_set_world(C.CnaGoHandle(handle), &values[0])))
}

func nativeEffectMatricesSetView(handle uint64, value [16]float32) error {
	var values [16]C.float
	for index := range value {
		values[index] = C.float(value[index])
	}
	return resultError("cna_effect_matrices_set_view", uint32(C.cna_go_effect_matrices_set_view(C.CnaGoHandle(handle), &values[0])))
}

func nativeEffectMatricesSetProjection(handle uint64, value [16]float32) error {
	var values [16]C.float
	for index := range value {
		values[index] = C.float(value[index])
	}
	return resultError("cna_effect_matrices_set_projection", uint32(C.cna_go_effect_matrices_set_projection(C.CnaGoHandle(handle), &values[0])))
}

func nativeEffectFogColor(handle uint64) ([3]float32, error) {
	var values [3]C.float
	code := uint32(C.cna_go_effect_fog_get_color(C.CnaGoHandle(handle), &values[0]))
	var result [3]float32
	for index := range values {
		result[index] = float32(values[index])
	}
	return result, resultError("cna_effect_fog_get_color", code)
}

func nativeEffectFogSetColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_effect_fog_set_color", uint32(C.cna_go_effect_fog_set_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeEffectFogSetEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_effect_fog_set_enabled", uint32(C.cna_go_effect_fog_set_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeEffectFogSetStart(handle uint64, value float32) error {
	return resultError("cna_effect_fog_set_start", uint32(C.cna_go_effect_fog_set_start(C.CnaGoHandle(handle), C.float(value))))
}

func nativeEffectFogSetEnd(handle uint64, value float32) error {
	return resultError("cna_effect_fog_set_end", uint32(C.cna_go_effect_fog_set_end(C.CnaGoHandle(handle), C.float(value))))
}

func nativeEffectLightsSetAmbientColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_effect_lights_set_ambient_color", uint32(C.cna_go_effect_lights_set_ambient_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeEffectLightsSetEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_effect_lights_set_enabled", uint32(C.cna_go_effect_lights_set_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeDirectionalLightSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_directional_light_set_diffuse_color", uint32(C.cna_go_directional_light_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeDirectionalLightSetDirection(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_directional_light_set_direction", uint32(C.cna_go_directional_light_set_direction(C.CnaGoHandle(handle), &values[0])))
}

func nativeDirectionalLightSetSpecularColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_directional_light_set_specular_color", uint32(C.cna_go_directional_light_set_specular_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeDirectionalLightSetEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_directional_light_set_enabled", uint32(C.cna_go_directional_light_set_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeBasicEffectCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_basic_effect_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_basic_effect_create(C.CnaGoHandle(device), out)
	})
}

func nativeBasicEffectSetTexture(effect, texture uint64) error {
	return resultError("cna_basic_effect_set_texture",
		uint32(C.cna_go_basic_effect_set_texture(C.CnaGoHandle(effect), C.CnaGoHandle(texture))))
}

// Foundation 80 -- AlphaTestEffect, DualTextureEffect and EffectMaterial.

func nativeAlphaTestEffectCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_alpha_test_effect_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_alpha_test_effect_create(C.CnaGoHandle(device), out)
	})
}

func nativeAlphaTestEffectSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_alpha_test_effect_set_diffuse_color", uint32(C.cna_go_alpha_test_effect_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeAlphaTestEffectSetAlpha(handle uint64, value float32) error {
	return resultError("cna_alpha_test_effect_set_alpha", uint32(C.cna_go_alpha_test_effect_set_alpha(C.CnaGoHandle(handle), C.float(value))))
}

func nativeAlphaTestEffectSetTexture(effect, texture uint64) error {
	return resultError("cna_alpha_test_effect_set_texture",
		uint32(C.cna_go_alpha_test_effect_set_texture(C.CnaGoHandle(effect), C.CnaGoHandle(texture))))
}

func nativeAlphaTestEffectSetVertexColorEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_alpha_test_effect_set_vertex_color_enabled", uint32(C.cna_go_alpha_test_effect_set_vertex_color_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeAlphaTestEffectSetAlphaFunction(handle uint64, value uint32) error {
	return resultError("cna_alpha_test_effect_set_alpha_function", uint32(C.cna_go_alpha_test_effect_set_alpha_function(C.CnaGoHandle(handle), C.uint32_t(value))))
}

func nativeAlphaTestEffectSetReferenceAlpha(handle uint64, value int32) error {
	return resultError("cna_alpha_test_effect_set_reference_alpha", uint32(C.cna_go_alpha_test_effect_set_reference_alpha(C.CnaGoHandle(handle), C.int32_t(value))))
}

func nativeDualTextureEffectCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_dual_texture_effect_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_dual_texture_effect_create(C.CnaGoHandle(device), out)
	})
}

func nativeDualTextureEffectSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_dual_texture_effect_set_diffuse_color", uint32(C.cna_go_dual_texture_effect_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeDualTextureEffectSetAlpha(handle uint64, value float32) error {
	return resultError("cna_dual_texture_effect_set_alpha", uint32(C.cna_go_dual_texture_effect_set_alpha(C.CnaGoHandle(handle), C.float(value))))
}

func nativeDualTextureEffectSetTexture(effect uint64, index uint32, texture uint64) error {
	return resultError("cna_dual_texture_effect_set_texture",
		uint32(C.cna_go_dual_texture_effect_set_texture(C.CnaGoHandle(effect), C.uint32_t(index), C.CnaGoHandle(texture))))
}

func nativeDualTextureEffectSetVertexColorEnabled(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_dual_texture_effect_set_vertex_color_enabled", uint32(C.cna_go_dual_texture_effect_set_vertex_color_enabled(C.CnaGoHandle(handle), raw)))
}

func nativeEffectMaterialCreate(cloneSource uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_material_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_material_create(C.CnaGoHandle(cloneSource), out)
	})
}

// Foundation 81 -- EnvironmentMapEffect and SkinnedEffect.

func nativeEnvironmentMapEffectCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_environment_map_effect_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_environment_map_effect_create(C.CnaGoHandle(device), out)
	})
}

func nativeEnvironmentMapEffectSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_environment_map_effect_set_diffuse_color", uint32(C.cna_go_environment_map_effect_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeEnvironmentMapEffectSetEmissiveColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_environment_map_effect_set_emissive_color", uint32(C.cna_go_environment_map_effect_set_emissive_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeEnvironmentMapEffectSetAlpha(handle uint64, value float32) error {
	return resultError("cna_environment_map_effect_set_alpha", uint32(C.cna_go_environment_map_effect_set_alpha(C.CnaGoHandle(handle), C.float(value))))
}

func nativeEnvironmentMapEffectSetTexture(effect, handle uint64) error {
	return resultError("cna_environment_map_effect_set_texture",
		uint32(C.cna_go_environment_map_effect_set_texture(C.CnaGoHandle(effect), C.CnaGoHandle(handle))))
}

func nativeEnvironmentMapEffectSetEnvironmentMap(effect, handle uint64) error {
	return resultError("cna_environment_map_effect_set_environment_map",
		uint32(C.cna_go_environment_map_effect_set_environment_map(C.CnaGoHandle(effect), C.CnaGoHandle(handle))))
}

func nativeEnvironmentMapEffectAmount(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_environment_map_effect_get_amount(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_environment_map_effect_get_amount", code)
}

func nativeEnvironmentMapEffectSetAmount(handle uint64, value float32) error {
	return resultError("cna_environment_map_effect_set_amount", uint32(C.cna_go_environment_map_effect_set_amount(C.CnaGoHandle(handle), C.float(value))))
}

func nativeEnvironmentMapEffectSpecular(handle uint64) ([3]float32, error) {
	var values [3]C.float
	code := uint32(C.cna_go_environment_map_effect_get_specular(C.CnaGoHandle(handle), &values[0]))
	return [3]float32{float32(values[0]), float32(values[1]), float32(values[2])},
		resultError("cna_environment_map_effect_get_specular", code)
}

func nativeEnvironmentMapEffectSetSpecular(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_environment_map_effect_set_specular", uint32(C.cna_go_environment_map_effect_set_specular(C.CnaGoHandle(handle), &values[0])))
}

func nativeEnvironmentMapEffectFresnelFactor(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_environment_map_effect_get_fresnel_factor(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_environment_map_effect_get_fresnel_factor", code)
}

func nativeEnvironmentMapEffectSetFresnelFactor(handle uint64, value float32) error {
	return resultError("cna_environment_map_effect_set_fresnel_factor", uint32(C.cna_go_environment_map_effect_set_fresnel_factor(C.CnaGoHandle(handle), C.float(value))))
}

func nativeSkinnedEffectCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_skinned_effect_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_skinned_effect_create(C.CnaGoHandle(device), out)
	})
}

func nativeSkinnedEffectSetDiffuseColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_skinned_effect_set_diffuse_color", uint32(C.cna_go_skinned_effect_set_diffuse_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeSkinnedEffectSetEmissiveColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_skinned_effect_set_emissive_color", uint32(C.cna_go_skinned_effect_set_emissive_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeSkinnedEffectSpecularColor(handle uint64) ([3]float32, error) {
	var values [3]C.float
	code := uint32(C.cna_go_skinned_effect_get_specular_color(C.CnaGoHandle(handle), &values[0]))
	return [3]float32{float32(values[0]), float32(values[1]), float32(values[2])},
		resultError("cna_skinned_effect_get_specular_color", code)
}

func nativeSkinnedEffectSetSpecularColor(handle uint64, value [3]float32) error {
	values := [3]C.float{C.float(value[0]), C.float(value[1]), C.float(value[2])}
	return resultError("cna_skinned_effect_set_specular_color", uint32(C.cna_go_skinned_effect_set_specular_color(C.CnaGoHandle(handle), &values[0])))
}

func nativeSkinnedEffectSpecularPower(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_skinned_effect_get_specular_power(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_skinned_effect_get_specular_power", code)
}

func nativeSkinnedEffectSetSpecularPower(handle uint64, value float32) error {
	return resultError("cna_skinned_effect_set_specular_power", uint32(C.cna_go_skinned_effect_set_specular_power(C.CnaGoHandle(handle), C.float(value))))
}

func nativeSkinnedEffectSetAlpha(handle uint64, value float32) error {
	return resultError("cna_skinned_effect_set_alpha", uint32(C.cna_go_skinned_effect_set_alpha(C.CnaGoHandle(handle), C.float(value))))
}

func nativeSkinnedEffectSetPreferPerPixelLighting(handle uint64, value bool) error {
	var raw C.uint8_t
	if value {
		raw = 1
	}
	return resultError("cna_skinned_effect_set_prefer_per_pixel_lighting", uint32(C.cna_go_skinned_effect_set_prefer_per_pixel_lighting(C.CnaGoHandle(handle), raw)))
}

func nativeSkinnedEffectSetTexture(effect, handle uint64) error {
	return resultError("cna_skinned_effect_set_texture",
		uint32(C.cna_go_skinned_effect_set_texture(C.CnaGoHandle(effect), C.CnaGoHandle(handle))))
}

func nativeSkinnedEffectSetWeightsPerVertex(handle uint64, value int32) error {
	return resultError("cna_skinned_effect_set_weights_per_vertex", uint32(C.cna_go_skinned_effect_set_weights_per_vertex(C.CnaGoHandle(handle), C.int32_t(value))))
}

func nativeSkinnedEffectSetBoneTransforms(handle uint64, transforms []float32) error {
	if len(transforms) == 0 {
		return resultError("cna_skinned_effect_set_bone_transforms",
			uint32(C.cna_go_skinned_effect_set_bone_transforms(C.CnaGoHandle(handle), nil, 0)))
	}
	values := make([]C.float, len(transforms))
	for index := range transforms {
		values[index] = C.float(transforms[index])
	}
	return resultError("cna_skinned_effect_set_bone_transforms",
		uint32(C.cna_go_skinned_effect_set_bone_transforms(C.CnaGoHandle(handle), &values[0],
			C.uint64_t(len(transforms)/16))))
}

func nativeSkinnedEffectCopyBoneTransforms(handle uint64, count int) ([]float32, error) {
	values := make([]C.float, count*16)
	var written C.uint64_t
	code := uint32(C.cna_go_skinned_effect_copy_bone_transforms(C.CnaGoHandle(handle),
		C.uint64_t(count), &values[0], C.uint64_t(count), &written))
	if err := resultError("cna_skinned_effect_copy_bone_transforms", code); err != nil {
		return nil, err
	}
	out := make([]float32, int(written)*16)
	for index := range out {
		out[index] = float32(values[index])
	}
	return out, nil
}

// Foundation 82 -- the two root types.

func nativeFrameworkDispatcherUpdate(game uint64) error {
	return resultError("cna_framework_dispatcher_update",
		uint32(C.cna_go_framework_dispatcher_update(C.CnaGoHandle(game))))
}

// nativeTitleContainerRead is the size-then-copy pair CNA's title reader has.
// The route always reports the file's byte count, so a first call with a zero
// capacity is a SIZE query and the second is the copy.
func nativeTitleContainerRead(game uint64, name string) ([]byte, error) {
	view := C.CString(name)
	defer C.free(unsafe.Pointer(view))
	var size C.uint64_t
	code := uint32(C.cna_go_title_container_read(C.CnaGoHandle(game), view,
		C.uint64_t(len(name)), nil, 0, &size))
	// A zero capacity is how this route is SIZED, and it answers
	// CNA_RESULT_BUFFER_TOO_SMALL rather than success -- the header says
	// out_bytes "always receives the file's byte count", including then. Every
	// other size query in this binding is a separate `_size` route that
	// succeeds, so treating 14 as a failure here is the mistake to avoid: the
	// first run of the title probe reported a file that had just been written
	// as not found.
	if code != cnaResultBufferTooSmall {
		if err := resultError("cna_title_container_read_ext", code); err != nil {
			return nil, err
		}
	}
	if size == 0 {
		return []byte{}, nil
	}
	buffer := make([]byte, int(size))
	var written C.uint64_t
	code = uint32(C.cna_go_title_container_read(C.CnaGoHandle(game), view,
		C.uint64_t(len(name)), (*C.uint8_t)(unsafe.Pointer(&buffer[0])),
		C.uint64_t(len(buffer)), &written))
	if err := resultError("cna_title_container_read_ext", code); err != nil {
		return nil, err
	}
	return buffer[:int(written)], nil
}

// Foundation 83 -- OcclusionQuery.

func nativeOcclusionQueryCreate(device uint64) (uint64, error) {
	return nativeEffectHandleOut("cna_occlusion_query_create", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_occlusion_query_create(C.CnaGoHandle(device), out)
	})
}

func nativeOcclusionQueryDestroy(query uint64) error {
	return resultError("cna_occlusion_query_destroy",
		uint32(C.cna_go_occlusion_query_destroy(C.CnaGoHandle(query))))
}

func nativeOcclusionQueryBegin(query uint64) error {
	return resultError("cna_occlusion_query_begin",
		uint32(C.cna_go_occlusion_query_begin(C.CnaGoHandle(query))))
}

func nativeOcclusionQueryEnd(query uint64) error {
	return resultError("cna_occlusion_query_end",
		uint32(C.cna_go_occlusion_query_end(C.CnaGoHandle(query))))
}

func nativeOcclusionQueryIsComplete(query uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_occlusion_query_get_is_complete(C.CnaGoHandle(query), &value))
	return value != 0, resultError("cna_occlusion_query_get_is_complete", code)
}

func nativeOcclusionQueryPixelCount(query uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_occlusion_query_get_pixel_count(C.CnaGoHandle(query), &value))
	return int32(value), resultError("cna_occlusion_query_get_pixel_count", code)
}

// Foundation 84 -- the dynamic buffers' options-carrying upload.
//
// ONE route covers both DynamicVertexBuffer.SetData overloads, because the
// reference's four-argument one forwards to the six-argument one with an offset
// of zero. The no-offset CNA route is recorded as SUBSUMED for the same reason.
func nativeVertexBufferSetDataRawAtWithOptions(buffer, offset uint64, data unsafe.Pointer, byteCount, vertexCount uint64, stride, options uint32) error {
	return resultError("cna_vertex_buffer_set_data_raw_at_with_options",
		uint32(C.cna_go_vertex_buffer_set_data_raw_at_with_options(C.CnaGoHandle(buffer),
			C.uint64_t(offset), data, C.uint64_t(byteCount), C.uint64_t(vertexCount),
			C.uint32_t(stride), C.uint32_t(options))))
}

// ---------------------------------------------------------------------------
// Foundation 87 -- SoundEffect and SoundEffectInstance.
// ---------------------------------------------------------------------------

// nativeSoundEffectCreatePCM16Range is cna_sound_effect_create_pcm16_range_ext,
// which CNA calls "the canonical seven-argument constructor". BOTH of XNA's
// constructors reach it: the three-argument one forwards to FromBuffer with an
// offset of zero, the whole length and a zero loop region, so one route serves
// both and cna_sound_effect_create_pcm16 is recorded as SUBSUMED.
func nativeSoundEffectCreatePCM16Range(game uint64, sampleRate, channels uint32, data unsafe.Pointer,
	byteCount uint64, offset, count, loopStart, loopLength int32) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_sound_effect_create_pcm16_range(C.CnaGoHandle(game),
		C.uint32_t(sampleRate), C.uint32_t(channels), (*C.uint8_t)(data), C.uint64_t(byteCount),
		C.int32_t(offset), C.int32_t(count), C.int32_t(loopStart), C.int32_t(loopLength), &handle))
	return uint64(handle), resultError("cna_sound_effect_create_pcm16_range_ext", code)
}

// nativeSoundEffectCreateFromEncoded is cna_sound_effect_create_from_encoded_ext,
// which backs SoundEffect::FromStream. CNA's note explains the shape: "The
// canonical operation takes a C++ stream and reads it to the end, so C takes the
// bytes it would have read."
func nativeSoundEffectCreateFromEncoded(game uint64, data unsafe.Pointer, byteCount uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_sound_effect_create_from_encoded(C.CnaGoHandle(game),
		(*C.uint8_t)(data), C.uint64_t(byteCount), &handle))
	return uint64(handle), resultError("cna_sound_effect_create_from_encoded_ext", code)
}

func nativeSoundEffectDurationTicks(effect uint64) (int64, error) {
	var ticks C.int64_t
	code := uint32(C.cna_go_sound_effect_get_duration_ticks(C.CnaGoHandle(effect), &ticks))
	return int64(ticks), resultError("cna_sound_effect_get_duration_ticks", code)
}

func nativeSoundEffectCreateInstance(effect uint64) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_sound_effect_create_instance(C.CnaGoHandle(effect), &handle))
	return uint64(handle), resultError("cna_sound_effect_create_instance", code)
}

func nativeSoundEffectDestroy(effect uint64) error {
	return resultError("cna_sound_effect_destroy",
		uint32(C.cna_go_sound_effect_destroy(C.CnaGoHandle(effect))))
}

// nativeSoundEffectPlay is cna_sound_effect_play. The bool it reports is not
// "did it make a sound": CNA answers CNA_FALSE for a disposed effect and
// CNA_RESULT_INVALID_STATE when too many instances are already playing, and it
// is the latter that XNA's Play converts into a false return.
func nativeSoundEffectPlay(effect uint64) (bool, error) {
	var played C.uint8_t
	code := uint32(C.cna_go_sound_effect_play(C.CnaGoHandle(effect), &played))
	return played != 0, resultError("cna_sound_effect_play", code)
}

func nativeSoundEffectPlayWithSettings(effect uint64, volume, pitch, pan float32) (bool, error) {
	var played C.uint8_t
	code := uint32(C.cna_go_sound_effect_play_with_settings(C.CnaGoHandle(effect),
		C.float(volume), C.float(pitch), C.float(pan), &played))
	return played != 0, resultError("cna_sound_effect_play_with_settings", code)
}

// The four process-wide scalars. Only the SETTERS are bound: the reference's
// getters are one `ldsfld` over a static field the setter maintains, so reading
// CNA back would be a second answer to a question the managed field already
// holds. Their read routes are recorded as REDUNDANT_READ.
func nativeSoundEffectSetMasterVolume(game uint64, value float32) error {
	return resultError("cna_sound_effect_set_master_volume",
		uint32(C.cna_go_sound_effect_set_master_volume(C.CnaGoHandle(game), C.float(value))))
}

func nativeSoundEffectSetDistanceScale(game uint64, value float32) error {
	return resultError("cna_sound_effect_set_distance_scale",
		uint32(C.cna_go_sound_effect_set_distance_scale(C.CnaGoHandle(game), C.float(value))))
}

func nativeSoundEffectSetDopplerScale(game uint64, value float32) error {
	return resultError("cna_sound_effect_set_doppler_scale",
		uint32(C.cna_go_sound_effect_set_doppler_scale(C.CnaGoHandle(game), C.float(value))))
}

func nativeSoundEffectSetSpeedOfSound(game uint64, value float32) error {
	return resultError("cna_sound_effect_set_speed_of_sound",
		uint32(C.cna_go_sound_effect_set_speed_of_sound(C.CnaGoHandle(game), C.float(value))))
}

func nativeSoundInstancePlay(instance uint64) error {
	return resultError("cna_sound_effect_instance_play",
		uint32(C.cna_go_sound_effect_instance_play(C.CnaGoHandle(instance))))
}

func nativeSoundInstancePause(instance uint64) error {
	return resultError("cna_sound_effect_instance_pause",
		uint32(C.cna_go_sound_effect_instance_pause(C.CnaGoHandle(instance))))
}

func nativeSoundInstanceResume(instance uint64) error {
	return resultError("cna_sound_effect_instance_resume",
		uint32(C.cna_go_sound_effect_instance_resume(C.CnaGoHandle(instance))))
}

func nativeSoundInstanceStop(instance uint64, immediate bool) error {
	flag := C.uint8_t(0)
	if immediate {
		flag = 1
	}
	return resultError("cna_sound_effect_instance_stop",
		uint32(C.cna_go_sound_effect_instance_stop(C.CnaGoHandle(instance), flag)))
}

// SoundEffectInstanceInfo is CNA_SoundEffectInstanceInfo, reduced to the five
// fields it carries that mean anything.
type SoundEffectInstanceInfo struct {
	State    uint32
	IsLooped bool
	Volume   float32
	Pitch    float32
	Pan      float32
}

func nativeSoundInstanceInfo(instance uint64) (SoundEffectInstanceInfo, error) {
	var state C.uint32_t
	var looped C.uint8_t
	var scalars [3]C.float
	code := uint32(C.cna_go_sound_effect_instance_get_info(C.CnaGoHandle(instance),
		&state, &looped, &scalars[0]))
	info := SoundEffectInstanceInfo{
		State:    uint32(state),
		IsLooped: looped != 0,
		Volume:   float32(scalars[0]),
		Pitch:    float32(scalars[1]),
		Pan:      float32(scalars[2]),
	}
	return info, resultError("cna_sound_effect_instance_get_info", code)
}

func nativeSoundInstanceSetVolume(instance uint64, value float32) error {
	return resultError("cna_sound_effect_instance_set_volume",
		uint32(C.cna_go_sound_effect_instance_set_volume(C.CnaGoHandle(instance), C.float(value))))
}

func nativeSoundInstanceSetPitch(instance uint64, value float32) error {
	return resultError("cna_sound_effect_instance_set_pitch",
		uint32(C.cna_go_sound_effect_instance_set_pitch(C.CnaGoHandle(instance), C.float(value))))
}

func nativeSoundInstanceSetPan(instance uint64, value float32) error {
	return resultError("cna_sound_effect_instance_set_pan",
		uint32(C.cna_go_sound_effect_instance_set_pan(C.CnaGoHandle(instance), C.float(value))))
}

func nativeSoundInstanceSetIsLooped(instance uint64, value bool) error {
	flag := C.uint8_t(0)
	if value {
		flag = 1
	}
	return resultError("cna_sound_effect_instance_set_is_looped",
		uint32(C.cna_go_sound_effect_instance_set_is_looped(C.CnaGoHandle(instance), flag)))
}

// nativeSoundInstanceApply3D is cna_sound_effect_instance_apply_3d_multi_ext.
//
// ONE route covers both of XNA's Apply3D overloads, because the reference's
// single-listener overload builds a one-element array and forwards -- so the
// multi-listener CNA route with a count of one IS the single-listener case, and
// cna_sound_effect_instance_apply_3d is recorded as SUBSUMED.
func nativeSoundInstanceApply3D(instance uint64, listeners []float32, count uint64, emitter []float32) error {
	if len(listeners) == 0 || len(emitter) == 0 {
		return resultError("cna_sound_effect_instance_apply_3d_multi_ext", resultInvalidArgument)
	}
	return resultError("cna_sound_effect_instance_apply_3d_multi_ext",
		uint32(C.cna_go_sound_effect_instance_apply_3d(C.CnaGoHandle(instance),
			(*C.float)(&listeners[0]), C.uint64_t(count), (*C.float)(&emitter[0]))))
}

// ---------------------------------------------------------------------------
// Foundation 88 -- DynamicSoundEffectInstance.
//
// Its handle is a SoundEffectInstance handle as far as the transport routes are
// concerned: CNA's play, pause, resume, stop, get_info and the four setters all
// take it. What the five routes below add is the streaming half -- creation
// without a SoundEffect, the pending-buffer count, the submission, and the two
// sample conversions, which unlike SoundEffect's STATIC pair are instance
// members and therefore really do take a handle.
// ---------------------------------------------------------------------------

func nativeDynamicSoundInstanceCreate(game uint64, sampleRate int32, channels uint32) (uint64, error) {
	var handle C.CnaGoHandle
	code := uint32(C.cna_go_dynamic_sound_effect_instance_create(C.CnaGoHandle(game),
		C.int32_t(sampleRate), C.uint32_t(channels), &handle))
	return uint64(handle), resultError("cna_dynamic_sound_effect_instance_create", code)
}

func nativeDynamicSoundInstancePendingBufferCount(instance uint64) (int32, error) {
	var count C.int32_t
	code := uint32(C.cna_go_dynamic_sound_effect_instance_get_pending_buffer_count(
		C.CnaGoHandle(instance), &count))
	return int32(count), resultError("cna_dynamic_sound_effect_instance_get_pending_buffer_count", code)
}

func nativeDynamicSoundInstanceSubmitBuffer(instance uint64, data unsafe.Pointer, byteCount uint64, offset, count int32) error {
	return resultError("cna_dynamic_sound_effect_instance_submit_buffer",
		uint32(C.cna_go_dynamic_sound_effect_instance_submit_buffer(C.CnaGoHandle(instance),
			(*C.uint8_t)(data), C.uint64_t(byteCount), C.int32_t(offset), C.int32_t(count))))
}

// ---------------------------------------------------------------------------
// Foundation 88 -- Microphone.
//
// The whole family is INDEX-addressed: a microphone is a POSITION in the
// machine's list rather than an owned handle, so nothing here is a Resource and
// nothing here is destroyed. That is CNA's shape and the reference's too --
// XNA's Microphone has an `assembly initonly int32 Id` and no disposal at all.
//
// # start_at and get_data_at are bound and NEVER exercised
//
// Both are projected because the pinned contract declares Start and GetData.
// Neither is called by the native scenario, and the reason is a standing
// constraint rather than a technical one: recording from a physical microphone
// is not something a test suite does on someone's machine. The enumeration and
// state surface is exercised in full; capture is not.
// ---------------------------------------------------------------------------

func nativeMicrophoneCount(game uint64) (uint64, error) {
	var count C.uint64_t
	code := uint32(C.cna_go_microphone_get_count(C.CnaGoHandle(game), &count))
	return uint64(count), resultError("cna_microphone_get_count", code)
}

// nativeMicrophoneDefaultIndex reports the default microphone's position AND
// whether there is one. CNA's route carries both because a machine with
// microphones need not have a DEFAULT: "Receives the zero-based index; left
// unchanged when there is no default."
func nativeMicrophoneDefaultIndex(game uint64) (uint64, bool, error) {
	var index C.uint64_t
	var available C.uint8_t
	code := uint32(C.cna_go_microphone_get_default_index(C.CnaGoHandle(game), &index, &available))
	return uint64(index), available != 0, resultError("cna_microphone_get_default_index_ext", code)
}

func nativeMicrophoneName(game, index uint64) (string, error) {
	var size C.uint64_t
	code := uint32(C.cna_go_microphone_get_name_size_at(C.CnaGoHandle(game), C.uint64_t(index), &size))
	if err := resultError("cna_microphone_get_name_size_at", code); err != nil {
		return "", err
	}
	if size == 0 {
		return "", nil
	}
	buffer := make([]byte, int(size))
	var written C.uint64_t
	code = uint32(C.cna_go_microphone_copy_name_at(C.CnaGoHandle(game), C.uint64_t(index),
		(*C.char)(unsafe.Pointer(&buffer[0])), C.uint64_t(len(buffer)), &written))
	if err := resultError("cna_microphone_copy_name_at", code); err != nil {
		return "", err
	}
	return string(buffer[:int(written)]), nil
}

func nativeMicrophoneBufferDurationTicks(game, index uint64) (int64, error) {
	var ticks C.int64_t
	code := uint32(C.cna_go_microphone_get_buffer_duration_ticks_at(C.CnaGoHandle(game),
		C.uint64_t(index), &ticks))
	return int64(ticks), resultError("cna_microphone_get_buffer_duration_ticks_at", code)
}

func nativeMicrophoneSetBufferDurationTicks(game, index uint64, ticks int64) error {
	return resultError("cna_microphone_set_buffer_duration_ticks_at",
		uint32(C.cna_go_microphone_set_buffer_duration_ticks_at(C.CnaGoHandle(game),
			C.uint64_t(index), C.int64_t(ticks))))
}

func nativeMicrophoneIsHeadset(game, index uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_microphone_get_is_headset_at(C.CnaGoHandle(game), C.uint64_t(index), &value))
	return value != 0, resultError("cna_microphone_get_is_headset_at", code)
}

func nativeMicrophoneSampleRate(game, index uint64) (int32, error) {
	var rate C.int32_t
	code := uint32(C.cna_go_microphone_get_sample_rate_at(C.CnaGoHandle(game), C.uint64_t(index), &rate))
	return int32(rate), resultError("cna_microphone_get_sample_rate_at", code)
}

func nativeMicrophoneState(game, index uint64) (uint32, error) {
	var state C.uint32_t
	code := uint32(C.cna_go_microphone_get_state_at(C.CnaGoHandle(game), C.uint64_t(index), &state))
	return uint32(state), resultError("cna_microphone_get_state_at", code)
}

func nativeMicrophoneStart(game, index uint64) error {
	return resultError("cna_microphone_start_at",
		uint32(C.cna_go_microphone_start_at(C.CnaGoHandle(game), C.uint64_t(index))))
}

func nativeMicrophoneStop(game, index uint64) error {
	return resultError("cna_microphone_stop_at",
		uint32(C.cna_go_microphone_stop_at(C.CnaGoHandle(game), C.uint64_t(index))))
}

func nativeMicrophoneGetData(game, index uint64, destination []byte) (uint64, error) {
	if len(destination) == 0 {
		return 0, resultError("cna_microphone_get_data_at", resultInvalidArgument)
	}
	var written C.uint64_t
	code := uint32(C.cna_go_microphone_get_data_at(C.CnaGoHandle(game), C.uint64_t(index),
		(*C.uint8_t)(unsafe.Pointer(&destination[0])), C.uint64_t(len(destination)), &written))
	return uint64(written), resultError("cna_microphone_get_data_at", code)
}

func nativeSoundInstanceDestroy(instance uint64) error {
	return resultError("cna_sound_effect_instance_destroy",
		uint32(C.cna_go_sound_effect_instance_destroy(C.CnaGoHandle(instance))))
}

func nativeEffectLightsDirectionalLight(effect uint64, index uint32) (uint64, error) {
	return nativeEffectHandleOut("cna_effect_lights_get_directional_light", func(out *C.CnaGoHandle) C.CnaGoResult {
		return C.cna_go_effect_lights_get_directional_light(C.CnaGoHandle(effect), C.uint32_t(index), out)
	})
}

func nativeDirectionalLightDestroy(light uint64) error {
	return resultError("cna_directional_light_destroy",
		uint32(C.cna_go_directional_light_destroy(C.CnaGoHandle(light))))
}

// ---------------------------------------------------------------------------
// Foundation 91. The Storage family.
//
// Every route here takes an opaque handle or a string view, so there is no
// struct to mirror and nothing for MANIFEST_LAYOUT_AGREEMENTS to prove; what
// has to agree is the parameter lists, which the route census checks against
// the canonical headers.
//
// None of these takes a game handle. Storage is not a graphics resource and CNA
// does not tie it to a window's event state, so unlike the Input family these
// are callable outside a lifecycle callback -- which is what lets a save happen
// from wherever a game decides to save.

// storageStringArg is the pointer half of a CNA_StringView. An empty Go string
// has no data pointer, and CNA reads the length first, so nil is correct rather
// than merely tolerated.
func storageStringArg(value string) *C.char {
	if len(value) == 0 {
		return nil
	}
	return (*C.char)(unsafe.Pointer(unsafe.StringData(value)))
}

func nativeStorageShowSelector() (uint64, error) {
	var device C.CnaGoHandle
	code := uint32(C.cna_go_storage_device_show_selector(&device))
	return uint64(device), resultError("cna_storage_device_show_selector", code)
}

func nativeStorageShowSelectorForPlayer(player uint32) (uint64, error) {
	var device C.CnaGoHandle
	code := uint32(C.cna_go_storage_device_show_selector_for_player(C.uint32_t(player), &device))
	return uint64(device), resultError("cna_storage_device_show_selector_for_player", code)
}

func nativeStorageShowSelectorWithSpace(sizeInBytes, directoryCount int32) (uint64, error) {
	var device C.CnaGoHandle
	code := uint32(C.cna_go_storage_device_show_selector_with_space(
		C.int32_t(sizeInBytes), C.int32_t(directoryCount), &device))
	return uint64(device), resultError("cna_storage_device_show_selector_with_space", code)
}

func nativeStorageShowSelectorForPlayerWithSpace(player uint32, sizeInBytes, directoryCount int32) (uint64, error) {
	var device C.CnaGoHandle
	code := uint32(C.cna_go_storage_device_show_selector_for_player_with_space(
		C.uint32_t(player), C.int32_t(sizeInBytes), C.int32_t(directoryCount), &device))
	return uint64(device), resultError("cna_storage_device_show_selector_for_player_with_space", code)
}

func nativeStorageDeviceFreeSpace(device uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_storage_device_get_free_space(C.CnaGoHandle(device), &value))
	return int64(value), resultError("cna_storage_device_get_free_space", code)
}

func nativeStorageDeviceTotalSpace(device uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_storage_device_get_total_space(C.CnaGoHandle(device), &value))
	return int64(value), resultError("cna_storage_device_get_total_space", code)
}

func nativeStorageDeviceIsConnected(device uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_storage_device_get_is_connected(C.CnaGoHandle(device), &value))
	return value != 0, resultError("cna_storage_device_get_is_connected", code)
}

func nativeStorageDeviceDeleteContainer(device uint64, titleName string) error {
	data := storageStringArg(titleName)
	code := uint32(C.cna_go_storage_device_delete_container(
		C.CnaGoHandle(device), data, C.uint64_t(len(titleName))))
	runtime.KeepAlive(titleName)
	return resultError("cna_storage_device_delete_container", code)
}

func nativeStorageDeviceDestroy(device uint64) error {
	return resultError("cna_storage_device_destroy",
		uint32(C.cna_go_storage_device_destroy(C.CnaGoHandle(device))))
}

func nativeStorageContainerOpen(device uint64, displayName string) (uint64, error) {
	data := storageStringArg(displayName)
	var container C.CnaGoHandle
	code := uint32(C.cna_go_storage_container_open(
		C.CnaGoHandle(device), data, C.uint64_t(len(displayName)), &container))
	runtime.KeepAlive(displayName)
	return uint64(container), resultError("cna_storage_container_open", code)
}

func nativeStorageContainerDisplayName(container uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_storage_container_get_display_name_size(C.CnaGoHandle(container), &byteCount))
	if err := resultError("cna_storage_container_get_display_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_storage_container_copy_display_name(
		C.CnaGoHandle(container), (*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_storage_container_copy_display_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeStorageContainerIsDisposed(container uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_storage_container_get_is_disposed(C.CnaGoHandle(container), &value))
	return value != 0, resultError("cna_storage_container_get_is_disposed", code)
}

func nativeStorageContainerDevice(container uint64) (uint64, error) {
	var device C.CnaGoHandle
	code := uint32(C.cna_go_storage_container_get_storage_device(C.CnaGoHandle(container), &device))
	return uint64(device), resultError("cna_storage_container_get_storage_device", code)
}

func nativeStorageContainerDispose(container uint64) error {
	return resultError("cna_storage_container_dispose",
		uint32(C.cna_go_storage_container_dispose(C.CnaGoHandle(container))))
}

func nativeStorageContainerDestroy(container uint64) error {
	return resultError("cna_storage_container_destroy",
		uint32(C.cna_go_storage_container_destroy(C.CnaGoHandle(container))))
}

func nativeStorageContainerCreateDirectory(container uint64, directory string) error {
	data := storageStringArg(directory)
	code := uint32(C.cna_go_storage_container_create_directory(
		C.CnaGoHandle(container), data, C.uint64_t(len(directory))))
	runtime.KeepAlive(directory)
	return resultError("cna_storage_container_create_directory", code)
}

func nativeStorageContainerDeleteDirectory(container uint64, directory string) error {
	data := storageStringArg(directory)
	code := uint32(C.cna_go_storage_container_delete_directory(
		C.CnaGoHandle(container), data, C.uint64_t(len(directory))))
	runtime.KeepAlive(directory)
	return resultError("cna_storage_container_delete_directory", code)
}

func nativeStorageContainerDirectoryExists(container uint64, directory string) (bool, error) {
	data := storageStringArg(directory)
	var value C.uint8_t
	code := uint32(C.cna_go_storage_container_directory_exists(
		C.CnaGoHandle(container), data, C.uint64_t(len(directory)), &value))
	runtime.KeepAlive(directory)
	return value != 0, resultError("cna_storage_container_directory_exists", code)
}

func nativeStorageContainerFileExists(container uint64, file string) (bool, error) {
	data := storageStringArg(file)
	var value C.uint8_t
	code := uint32(C.cna_go_storage_container_file_exists(
		C.CnaGoHandle(container), data, C.uint64_t(len(file)), &value))
	runtime.KeepAlive(file)
	return value != 0, resultError("cna_storage_container_file_exists", code)
}

func nativeStorageContainerDeleteFile(container uint64, file string) error {
	data := storageStringArg(file)
	code := uint32(C.cna_go_storage_container_delete_file(
		C.CnaGoHandle(container), data, C.uint64_t(len(file))))
	runtime.KeepAlive(file)
	return resultError("cna_storage_container_delete_file", code)
}

// nativeStorageContainerNames is the two-step enumeration BOTH GetDirectoryNames
// and GetFileNames use: a count, then one copy per index. The count and the
// copies are separate CNA calls, so a directory that changes between them would
// be seen half-changed -- which is exactly what the reference's own
// Directory.GetFiles cannot promise either.
func nativeStorageContainerNames(container uint64, pattern string, directories bool) ([]string, error) {
	data := storageStringArg(pattern)
	var count C.uint64_t
	var code uint32
	if directories {
		code = uint32(C.cna_go_storage_container_get_directory_name_count(
			C.CnaGoHandle(container), data, C.uint64_t(len(pattern)), &count))
	} else {
		code = uint32(C.cna_go_storage_container_get_file_name_count(
			C.CnaGoHandle(container), data, C.uint64_t(len(pattern)), &count))
	}
	route := "cna_storage_container_get_file_name_count"
	if directories {
		route = "cna_storage_container_get_directory_name_count"
	}
	if err := resultError(route, code); err != nil {
		runtime.KeepAlive(pattern)
		return nil, err
	}
	names := make([]string, 0, int(count))
	for index := C.uint64_t(0); index < count; index++ {
		buffer := make([]byte, 512)
		var copied C.uint64_t
		if directories {
			code = uint32(C.cna_go_storage_container_copy_directory_name(
				C.CnaGoHandle(container), data, C.uint64_t(len(pattern)), index,
				(*C.char)(unsafe.Pointer(&buffer[0])), C.uint64_t(len(buffer)), &copied))
		} else {
			code = uint32(C.cna_go_storage_container_copy_file_name(
				C.CnaGoHandle(container), data, C.uint64_t(len(pattern)), index,
				(*C.char)(unsafe.Pointer(&buffer[0])), C.uint64_t(len(buffer)), &copied))
		}
		copyRoute := "cna_storage_container_copy_file_name"
		if directories {
			copyRoute = "cna_storage_container_copy_directory_name"
		}
		if err := resultError(copyRoute, code); err != nil {
			runtime.KeepAlive(pattern)
			return nil, err
		}
		names = append(names, string(buffer[:int(copied)]))
	}
	runtime.KeepAlive(pattern)
	return names, nil
}

func nativeStorageContainerCreateFile(container uint64, file string) (uint64, error) {
	data := storageStringArg(file)
	var stream C.CnaGoHandle
	code := uint32(C.cna_go_storage_container_create_file(
		C.CnaGoHandle(container), data, C.uint64_t(len(file)), &stream))
	runtime.KeepAlive(file)
	return uint64(stream), resultError("cna_storage_container_create_file", code)
}

// nativeStorageContainerOpenFile serves all THREE OpenFile overloads. The
// reference's shorter two forward to the longest with defaults, so one Go
// wrapper with a mode selector keeps the three CNA routes reachable without
// pretending they are one.
func nativeStorageContainerOpenFile(container uint64, file string, mode uint32, access uint32, share uint32, arity int) (uint64, error) {
	data := storageStringArg(file)
	var stream C.CnaGoHandle
	var code uint32
	var route string
	switch arity {
	case 1:
		route = "cna_storage_container_open_file"
		code = uint32(C.cna_go_storage_container_open_file(
			C.CnaGoHandle(container), data, C.uint64_t(len(file)), C.uint32_t(mode), &stream))
	case 2:
		route = "cna_storage_container_open_file_access"
		code = uint32(C.cna_go_storage_container_open_file_access(
			C.CnaGoHandle(container), data, C.uint64_t(len(file)),
			C.uint32_t(mode), C.uint32_t(access), &stream))
	default:
		route = "cna_storage_container_open_file_share"
		code = uint32(C.cna_go_storage_container_open_file_share(
			C.CnaGoHandle(container), data, C.uint64_t(len(file)),
			C.uint32_t(mode), C.uint32_t(access), C.uint32_t(share), &stream))
	}
	runtime.KeepAlive(file)
	return uint64(stream), resultError(route, code)
}

func nativeStorageStreamRead(stream uint64, destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	var read C.uint64_t
	code := uint32(C.cna_go_storage_stream_read(C.CnaGoHandle(stream),
		(*C.uint8_t)(unsafe.Pointer(&destination[0])), C.uint64_t(len(destination)), &read))
	return int(read), resultError("cna_storage_stream_read", code)
}

func nativeStorageStreamWrite(stream uint64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return resultError("cna_storage_stream_write",
		uint32(C.cna_go_storage_stream_write(C.CnaGoHandle(stream),
			(*C.uint8_t)(unsafe.Pointer(&data[0])), C.uint64_t(len(data)))))
}

func nativeStorageStreamSeek(stream uint64, offset int64, origin uint32) (int64, error) {
	var position C.int64_t
	code := uint32(C.cna_go_storage_stream_seek(C.CnaGoHandle(stream),
		C.int64_t(offset), C.uint32_t(origin), &position))
	return int64(position), resultError("cna_storage_stream_seek", code)
}

func nativeStorageStreamPosition(stream uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_storage_stream_get_position(C.CnaGoHandle(stream), &value))
	return int64(value), resultError("cna_storage_stream_get_position", code)
}

func nativeStorageStreamLength(stream uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_storage_stream_get_length(C.CnaGoHandle(stream), &value))
	return int64(value), resultError("cna_storage_stream_get_length", code)
}

func nativeStorageStreamSetLength(stream uint64, length int64) error {
	return resultError("cna_storage_stream_set_length",
		uint32(C.cna_go_storage_stream_set_length(C.CnaGoHandle(stream), C.int64_t(length))))
}

func nativeStorageStreamCanRead(stream uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_storage_stream_get_can_read(C.CnaGoHandle(stream), &value))
	return value != 0, resultError("cna_storage_stream_get_can_read", code)
}

func nativeStorageStreamCanWrite(stream uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_storage_stream_get_can_write(C.CnaGoHandle(stream), &value))
	return value != 0, resultError("cna_storage_stream_get_can_write", code)
}

func nativeStorageStreamCanSeek(stream uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_storage_stream_get_can_seek(C.CnaGoHandle(stream), &value))
	return value != 0, resultError("cna_storage_stream_get_can_seek", code)
}

func nativeStorageStreamFlush(stream uint64) error {
	return resultError("cna_storage_stream_flush",
		uint32(C.cna_go_storage_stream_flush(C.CnaGoHandle(stream))))
}

func nativeStorageStreamClose(stream uint64) error {
	return resultError("cna_storage_stream_close",
		uint32(C.cna_go_storage_stream_close(C.CnaGoHandle(stream))))
}

// The three `_ext` routes, which are NOT XNA surface. They exist so a test can
// isolate itself in a project-controlled root and then PROVE it.

func nativeStorageSetAppName(name string) error {
	data := storageStringArg(name)
	code := uint32(C.cna_go_storage_set_app_name_ext(data, C.uint64_t(len(name))))
	runtime.KeepAlive(name)
	return resultError("cna_storage_set_app_name_ext", code)
}

func nativeStorageRoot() (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_storage_get_root_size_ext(&byteCount))
	if err := resultError("cna_storage_get_root_size_ext", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_storage_copy_root_ext(
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_storage_copy_root_ext", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

// ---------------------------------------------------------------------------
// Foundation 92. The content reader.
//
// Fifteen of CNA's twenty routes have an XNA counterpart. The other five --
// read_bounding_sphere, get_version, get_platform and the two check_* helpers
// -- have none and are recorded as deliberately unbound.

// ContentReaderCreateValues is CNA_ContentReaderCreateInfo reduced to what the
// projection supplies. struct_size and struct_version are the bridge's; the
// reserved bytes are zeroed there too.
type ContentReaderCreateValues struct {
	ContentManager uint64
	Stream         uint64
	AssetName      string
	Version        int32
	Platform       uint8
}

func nativeContentReaderCreate(values ContentReaderCreateValues) (uint64, error) {
	name := storageStringArg(values.AssetName)
	var reader C.CnaGoHandle
	code := uint32(C.cna_go_content_reader_create(
		C.CnaGoHandle(values.ContentManager), C.CnaGoHandle(values.Stream),
		name, C.uint64_t(len(values.AssetName)),
		C.int32_t(values.Version), C.uint8_t(values.Platform), &reader))
	runtime.KeepAlive(values.AssetName)
	return uint64(reader), resultError("cna_content_reader_create", code)
}

func nativeContentReaderAssetName(reader uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_content_reader_get_asset_name_size(C.CnaGoHandle(reader), &byteCount))
	if err := resultError("cna_content_reader_get_asset_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_content_reader_copy_asset_name(
		C.CnaGoHandle(reader), (*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_content_reader_copy_asset_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

// nativeContentReaderReadFloats serves the five value reads that hand back
// floats. The count is the route's own, so a caller cannot ask for a Vector2
// and be given four values.
func nativeContentReaderReadFloats(reader uint64, kind int) ([]float32, error) {
	var values [16]C.float
	var code uint32
	var route string
	var count int
	switch kind {
	case 2:
		route, count = "cna_content_reader_read_vector2", 2
		code = uint32(C.cna_go_content_reader_read_vector2(C.CnaGoHandle(reader), &values[0]))
	case 3:
		route, count = "cna_content_reader_read_vector3", 3
		code = uint32(C.cna_go_content_reader_read_vector3(C.CnaGoHandle(reader), &values[0]))
	case 4:
		route, count = "cna_content_reader_read_vector4", 4
		code = uint32(C.cna_go_content_reader_read_vector4(C.CnaGoHandle(reader), &values[0]))
	case 5:
		route, count = "cna_content_reader_read_quaternion", 4
		code = uint32(C.cna_go_content_reader_read_quaternion(C.CnaGoHandle(reader), &values[0]))
	default:
		route, count = "cna_content_reader_read_matrix", 16
		code = uint32(C.cna_go_content_reader_read_matrix(C.CnaGoHandle(reader), &values[0]))
	}
	if err := resultError(route, code); err != nil {
		return nil, err
	}
	result := make([]float32, count)
	for i := 0; i < count; i++ {
		result[i] = float32(values[i])
	}
	return result, nil
}

func nativeContentReaderReadColor(reader uint64) ([4]byte, error) {
	var channels [4]C.uint8_t
	code := uint32(C.cna_go_content_reader_read_color(C.CnaGoHandle(reader), &channels[0]))
	var result [4]byte
	for i := range result {
		result[i] = byte(channels[i])
	}
	return result, resultError("cna_content_reader_read_color", code)
}

func nativeContentReaderReadObjectTag(reader uint64) (bool, error) {
	var hasValue C.uint8_t
	code := uint32(C.cna_go_content_reader_read_object_tag(C.CnaGoHandle(reader), &hasValue))
	return hasValue != 0, resultError("cna_content_reader_read_object_tag", code)
}

func nativeContentReaderInitializeTypeReaders(reader uint64) error {
	return resultError("cna_content_reader_initialize_type_readers",
		uint32(C.cna_go_content_reader_initialize_type_readers(C.CnaGoHandle(reader))))
}

func nativeContentReaderReadSharedResources(reader uint64) error {
	return resultError("cna_content_reader_read_shared_resources",
		uint32(C.cna_go_content_reader_read_shared_resources(C.CnaGoHandle(reader))))
}

func nativeContentReaderReadBytesExact(reader uint64, count int32, readerName string, destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	name := storageStringArg(readerName)
	var read C.uint64_t
	code := uint32(C.cna_go_content_reader_read_bytes_exact(
		C.CnaGoHandle(reader), C.int32_t(count), name, C.uint64_t(len(readerName)),
		(*C.uint8_t)(unsafe.Pointer(&destination[0])), C.uint64_t(len(destination)), &read))
	runtime.KeepAlive(readerName)
	return int(read), resultError("cna_content_reader_read_bytes_exact", code)
}

func nativeContentReaderDestroy(reader uint64) error {
	return resultError("cna_content_reader_destroy",
		uint32(C.cna_go_content_reader_destroy(C.CnaGoHandle(reader))))
}

// ---------------------------------------------------------------------------
// Foundation 95. The media metadata graph.
//
// Ten types over 105 routes, and every one of them is regular: an owned handle,
// a disposal pair, a name, a type name, and whatever that type describes.
//
// The `_dispose` and `_destroy` routes are BOTH bound and they are not a
// duplication. `_dispose` is XNA's IDisposable -- a consumer calls it and the
// object stays queryable, answering true from IsDisposed -- and `_destroy`
// releases the native memory, which the projection does when it is finished
// with the handle. Binding only one would make either IsDisposed unanswerable
// or the memory unreleasable.
// ---------------------------------------------------------------------------

func nativeSongDispose(handle uint64) error {
	return resultError("cna_song_dispose",
		uint32(C.cna_go_song_dispose(C.CnaGoHandle(handle))))
}

func nativeSongDestroy(handle uint64) error {
	return resultError("cna_song_destroy",
		uint32(C.cna_go_song_destroy(C.CnaGoHandle(handle))))
}

func nativeSongIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_song_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_song_get_is_disposed", code)
}

func nativeSongHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_song_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_song_get_hash_code", code)
}

func nativeSongName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_song_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_song_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_song_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_song_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeSongTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_song_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_song_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_song_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_song_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeSongEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_song_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_song_equals", code)
}

func nativeAlbumDispose(handle uint64) error {
	return resultError("cna_album_dispose",
		uint32(C.cna_go_album_dispose(C.CnaGoHandle(handle))))
}

func nativeAlbumDestroy(handle uint64) error {
	return resultError("cna_album_destroy",
		uint32(C.cna_go_album_destroy(C.CnaGoHandle(handle))))
}

func nativeAlbumIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_album_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_album_get_is_disposed", code)
}

func nativeAlbumHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_album_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_album_get_hash_code", code)
}

func nativeAlbumName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_album_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_album_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_album_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_album_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeAlbumTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_album_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_album_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_album_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_album_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeAlbumEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_album_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_album_equals", code)
}

func nativeArtistDispose(handle uint64) error {
	return resultError("cna_artist_dispose",
		uint32(C.cna_go_artist_dispose(C.CnaGoHandle(handle))))
}

func nativeArtistDestroy(handle uint64) error {
	return resultError("cna_artist_destroy",
		uint32(C.cna_go_artist_destroy(C.CnaGoHandle(handle))))
}

func nativeArtistIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_artist_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_artist_get_is_disposed", code)
}

func nativeArtistHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_artist_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_artist_get_hash_code", code)
}

func nativeArtistName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_artist_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_artist_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_artist_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_artist_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeArtistTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_artist_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_artist_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_artist_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_artist_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeArtistEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_artist_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_artist_equals", code)
}

func nativeGenreDispose(handle uint64) error {
	return resultError("cna_genre_dispose",
		uint32(C.cna_go_genre_dispose(C.CnaGoHandle(handle))))
}

func nativeGenreDestroy(handle uint64) error {
	return resultError("cna_genre_destroy",
		uint32(C.cna_go_genre_destroy(C.CnaGoHandle(handle))))
}

func nativeGenreIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_genre_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_genre_get_is_disposed", code)
}

func nativeGenreHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_genre_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_genre_get_hash_code", code)
}

func nativeGenreName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_genre_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_genre_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_genre_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_genre_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeGenreTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_genre_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_genre_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_genre_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_genre_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeGenreEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_genre_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_genre_equals", code)
}

func nativePlaylistDispose(handle uint64) error {
	return resultError("cna_playlist_dispose",
		uint32(C.cna_go_playlist_dispose(C.CnaGoHandle(handle))))
}

func nativePlaylistDestroy(handle uint64) error {
	return resultError("cna_playlist_destroy",
		uint32(C.cna_go_playlist_destroy(C.CnaGoHandle(handle))))
}

func nativePlaylistIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_playlist_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_playlist_get_is_disposed", code)
}

func nativePlaylistHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_playlist_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_playlist_get_hash_code", code)
}

func nativePlaylistName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_playlist_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_playlist_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_playlist_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_playlist_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePlaylistTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_playlist_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_playlist_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_playlist_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_playlist_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePlaylistEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_playlist_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_playlist_equals", code)
}

func nativeSongArtist(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_song_get_artist(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_song_get_artist", code)
}

func nativeSongAlbum(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_song_get_album(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_song_get_album", code)
}

func nativeSongGenre(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_song_get_genre(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_song_get_genre", code)
}

func nativeSongDurationTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_song_get_duration(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_song_get_duration", code)
}

func nativeSongIsRated(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_song_get_is_rated(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_song_get_is_rated", code)
}

func nativeSongRating(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_song_get_rating(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_song_get_rating", code)
}

func nativeSongPlayCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_song_get_play_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_song_get_play_count", code)
}

func nativeSongTrackNumber(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_song_get_track_number(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_song_get_track_number", code)
}

func nativeSongIsProtected(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_song_get_is_protected(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_song_get_is_protected", code)
}

func nativeSongFromURI(game uint64, name, uri string) (uint64, error) {
	nameArg := storageStringArg(name)
	uriArg := storageStringArg(uri)
	var value C.CnaGoHandle
	code := uint32(C.cna_go_song_create_from_uri(C.CnaGoHandle(game),
		nameArg, C.uint64_t(len(name)),
		uriArg, C.uint64_t(len(uri)), &value))
	runtime.KeepAlive(name)
	runtime.KeepAlive(uri)
	return uint64(value), resultError("cna_song_create_from_uri", code)
}

func nativeAlbumArtist(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_album_get_artist(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_album_get_artist", code)
}

func nativeAlbumGenre(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_album_get_genre(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_album_get_genre", code)
}

func nativeAlbumSongs(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_album_get_songs(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_album_get_songs", code)
}

func nativeAlbumDurationTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_album_get_duration(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_album_get_duration", code)
}

func nativeAlbumHasArt(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_album_get_has_art(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_album_get_has_art", code)
}

func nativeAlbumArt(handle uint64) ([]byte, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_album_get_art_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_album_get_art_size", code); err != nil {
		return nil, err
	}
	if byteCount == 0 {
		return nil, nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_album_copy_art(C.CnaGoHandle(handle),
		(*C.uint8_t)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_album_copy_art", code); err != nil {
		return nil, err
	}
	return buffer[:int(copied)], nil
}

func nativeAlbumThumbnail(handle uint64) ([]byte, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_album_get_thumbnail_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_album_get_thumbnail_size", code); err != nil {
		return nil, err
	}
	if byteCount == 0 {
		return nil, nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_album_copy_thumbnail(C.CnaGoHandle(handle),
		(*C.uint8_t)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_album_copy_thumbnail", code); err != nil {
		return nil, err
	}
	return buffer[:int(copied)], nil
}

func nativeArtistSongs(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_artist_get_songs(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_artist_get_songs", code)
}

func nativeArtistAlbums(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_artist_get_albums(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_artist_get_albums", code)
}

func nativeGenreSongs(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_genre_get_songs(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_genre_get_songs", code)
}

func nativeGenreAlbums(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_genre_get_albums(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_genre_get_albums", code)
}

func nativePlaylistSongs(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_playlist_get_songs(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_playlist_get_songs", code)
}

func nativePlaylistDurationTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_playlist_get_duration(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_playlist_get_duration", code)
}

func nativeSongCollectionDispose(handle uint64) error {
	return resultError("cna_song_collection_dispose",
		uint32(C.cna_go_song_collection_dispose(C.CnaGoHandle(handle))))
}

func nativeSongCollectionDestroy(handle uint64) error {
	return resultError("cna_song_collection_destroy",
		uint32(C.cna_go_song_collection_destroy(C.CnaGoHandle(handle))))
}

func nativeSongCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_song_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_song_collection_get_is_disposed", code)
}

func nativeSongCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_song_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_song_collection_get_count", code)
}

func nativeSongCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_song_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_song_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_song_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_song_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeSongCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_song_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_song_collection_get_at", code)
}

func nativeAlbumCollectionDispose(handle uint64) error {
	return resultError("cna_album_collection_dispose",
		uint32(C.cna_go_album_collection_dispose(C.CnaGoHandle(handle))))
}

func nativeAlbumCollectionDestroy(handle uint64) error {
	return resultError("cna_album_collection_destroy",
		uint32(C.cna_go_album_collection_destroy(C.CnaGoHandle(handle))))
}

func nativeAlbumCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_album_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_album_collection_get_is_disposed", code)
}

func nativeAlbumCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_album_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_album_collection_get_count", code)
}

func nativeAlbumCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_album_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_album_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_album_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_album_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeAlbumCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_album_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_album_collection_get_at", code)
}

func nativeArtistCollectionDispose(handle uint64) error {
	return resultError("cna_artist_collection_dispose",
		uint32(C.cna_go_artist_collection_dispose(C.CnaGoHandle(handle))))
}

func nativeArtistCollectionDestroy(handle uint64) error {
	return resultError("cna_artist_collection_destroy",
		uint32(C.cna_go_artist_collection_destroy(C.CnaGoHandle(handle))))
}

func nativeArtistCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_artist_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_artist_collection_get_is_disposed", code)
}

func nativeArtistCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_artist_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_artist_collection_get_count", code)
}

func nativeArtistCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_artist_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_artist_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_artist_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_artist_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeArtistCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_artist_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_artist_collection_get_at", code)
}

func nativeGenreCollectionDispose(handle uint64) error {
	return resultError("cna_genre_collection_dispose",
		uint32(C.cna_go_genre_collection_dispose(C.CnaGoHandle(handle))))
}

func nativeGenreCollectionDestroy(handle uint64) error {
	return resultError("cna_genre_collection_destroy",
		uint32(C.cna_go_genre_collection_destroy(C.CnaGoHandle(handle))))
}

func nativeGenreCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_genre_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_genre_collection_get_is_disposed", code)
}

func nativeGenreCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_genre_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_genre_collection_get_count", code)
}

func nativeGenreCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_genre_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_genre_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_genre_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_genre_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeGenreCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_genre_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_genre_collection_get_at", code)
}

func nativePlaylistCollectionDispose(handle uint64) error {
	return resultError("cna_playlist_collection_dispose",
		uint32(C.cna_go_playlist_collection_dispose(C.CnaGoHandle(handle))))
}

func nativePlaylistCollectionDestroy(handle uint64) error {
	return resultError("cna_playlist_collection_destroy",
		uint32(C.cna_go_playlist_collection_destroy(C.CnaGoHandle(handle))))
}

func nativePlaylistCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_playlist_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_playlist_collection_get_is_disposed", code)
}

func nativePlaylistCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_playlist_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_playlist_collection_get_count", code)
}

func nativePlaylistCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_playlist_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_playlist_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_playlist_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_playlist_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePlaylistCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_playlist_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_playlist_collection_get_at", code)
}

// ---------------------------------------------------------------------------
// Foundation 96. The media library and the picture graph.
//
// The same shapes Foundation 95 established, over six more types. Two are new:
// a MediaSource is addressed by INDEX rather than by a handle, and a picture's
// date arrives as UNIX ticks rather than CLR ones.
// ---------------------------------------------------------------------------

func nativePictureDispose(handle uint64) error {
	return resultError("cna_picture_dispose",
		uint32(C.cna_go_picture_dispose(C.CnaGoHandle(handle))))
}

func nativePictureDestroy(handle uint64) error {
	return resultError("cna_picture_destroy",
		uint32(C.cna_go_picture_destroy(C.CnaGoHandle(handle))))
}

func nativePictureIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_picture_get_is_disposed", code)
}

func nativePictureHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_get_hash_code", code)
}

func nativePictureName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_picture_equals", code)
}

func nativePictureAlbumDispose(handle uint64) error {
	return resultError("cna_picture_album_dispose",
		uint32(C.cna_go_picture_album_dispose(C.CnaGoHandle(handle))))
}

func nativePictureAlbumDestroy(handle uint64) error {
	return resultError("cna_picture_album_destroy",
		uint32(C.cna_go_picture_album_destroy(C.CnaGoHandle(handle))))
}

func nativePictureAlbumIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_album_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_picture_album_get_is_disposed", code)
}

func nativePictureAlbumHashCode(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_album_get_hash_code(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_album_get_hash_code", code)
}

func nativePictureAlbumName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_album_get_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_album_get_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_album_copy_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_album_copy_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureAlbumTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_album_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_album_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_album_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_album_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureAlbumEquals(left, right uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_album_equals(C.CnaGoHandle(left), C.CnaGoHandle(right), &value))
	return value != 0, resultError("cna_picture_album_equals", code)
}

func nativePictureAlbumOf(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_picture_get_album(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_picture_get_album", code)
}

func nativePictureWidth(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_get_width(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_get_width", code)
}

func nativePictureHeight(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_get_height(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_get_height", code)
}

func nativePictureDateUnixTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_picture_get_date_unix_ticks(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_picture_get_date_unix_ticks", code)
}

func nativePictureImage(handle uint64) ([]byte, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_get_image_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_get_image_size", code); err != nil {
		return nil, err
	}
	if byteCount == 0 {
		return nil, nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_copy_image(C.CnaGoHandle(handle),
		(*C.uint8_t)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_copy_image", code); err != nil {
		return nil, err
	}
	return buffer[:int(copied)], nil
}

func nativePictureThumbnail(handle uint64) ([]byte, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_get_thumbnail_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_get_thumbnail_size", code); err != nil {
		return nil, err
	}
	if byteCount == 0 {
		return nil, nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_copy_thumbnail(C.CnaGoHandle(handle),
		(*C.uint8_t)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_copy_thumbnail", code); err != nil {
		return nil, err
	}
	return buffer[:int(copied)], nil
}

func nativePictureAlbumAlbums(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_picture_album_get_albums(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_picture_album_get_albums", code)
}

func nativePictureAlbumPictures(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_picture_album_get_pictures(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_picture_album_get_pictures", code)
}

func nativePictureAlbumParent(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_picture_album_get_parent(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_picture_album_get_parent", code)
}

func nativePictureCollectionDispose(handle uint64) error {
	return resultError("cna_picture_collection_dispose",
		uint32(C.cna_go_picture_collection_dispose(C.CnaGoHandle(handle))))
}

func nativePictureCollectionDestroy(handle uint64) error {
	return resultError("cna_picture_collection_destroy",
		uint32(C.cna_go_picture_collection_destroy(C.CnaGoHandle(handle))))
}

func nativePictureCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_picture_collection_get_is_disposed", code)
}

func nativePictureCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_collection_get_count", code)
}

func nativePictureCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_picture_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_picture_collection_get_at", code)
}

func nativePictureAlbumCollectionDispose(handle uint64) error {
	return resultError("cna_picture_album_collection_dispose",
		uint32(C.cna_go_picture_album_collection_dispose(C.CnaGoHandle(handle))))
}

func nativePictureAlbumCollectionDestroy(handle uint64) error {
	return resultError("cna_picture_album_collection_destroy",
		uint32(C.cna_go_picture_album_collection_destroy(C.CnaGoHandle(handle))))
}

func nativePictureAlbumCollectionIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_picture_album_collection_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_picture_album_collection_get_is_disposed", code)
}

func nativePictureAlbumCollectionCount(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_picture_album_collection_get_count(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_picture_album_collection_get_count", code)
}

func nativePictureAlbumCollectionTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_picture_album_collection_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_picture_album_collection_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_picture_album_collection_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_picture_album_collection_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativePictureAlbumCollectionAt(handle uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_picture_album_collection_get_at(C.CnaGoHandle(handle), C.int32_t(index), &value))
	return uint64(value), resultError("cna_picture_album_collection_get_at", code)
}

func nativeMediaLibraryDispose(handle uint64) error {
	return resultError("cna_media_library_dispose",
		uint32(C.cna_go_media_library_dispose(C.CnaGoHandle(handle))))
}

func nativeMediaLibraryDestroy(handle uint64) error {
	return resultError("cna_media_library_destroy",
		uint32(C.cna_go_media_library_destroy(C.CnaGoHandle(handle))))
}

func nativeMediaLibraryIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_library_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_media_library_get_is_disposed", code)
}

func nativeMediaLibraryTypeName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_media_library_get_type_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_media_library_get_type_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_media_library_copy_type_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_media_library_copy_type_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeMediaLibraryMediaSourceName(handle uint64) (string, error) {
	var byteCount C.uint64_t
	code := uint32(C.cna_go_media_library_get_media_source_name_size(C.CnaGoHandle(handle), &byteCount))
	if err := resultError("cna_media_library_get_media_source_name_size", code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	code = uint32(C.cna_go_media_library_copy_media_source_name(C.CnaGoHandle(handle),
		(*C.char)(unsafe.Pointer(&buffer[0])), byteCount, &copied))
	if err := resultError("cna_media_library_copy_media_source_name", code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

func nativeMediaLibraryMediaSourceType(handle uint64) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_media_library_get_media_source_type(C.CnaGoHandle(handle), &value))
	return uint32(value), resultError("cna_media_library_get_media_source_type", code)
}

func nativeMediaLibrarySongs(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_songs(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_songs", code)
}

func nativeMediaLibraryArtists(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_artists(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_artists", code)
}

func nativeMediaLibraryAlbums(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_albums(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_albums", code)
}

func nativeMediaLibraryGenres(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_genres(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_genres", code)
}

func nativeMediaLibraryPlaylists(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_playlists(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_playlists", code)
}

func nativeMediaLibraryPictures(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_pictures(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_pictures", code)
}

func nativeMediaLibrarySavedPictures(handle uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_get_saved_pictures(C.CnaGoHandle(handle), &value))
	return uint64(value), resultError("cna_media_library_get_saved_pictures", code)
}

func nativeMediaLibraryRootPictureAlbum(handle uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_media_library_get_root_picture_album(C.CnaGoHandle(handle), &value, &available))
	return uint64(value), available != 0, resultError("cna_media_library_get_root_picture_album", code)
}

func nativeMediaLibraryCreate(game uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_create(C.CnaGoHandle(game), &value))
	return uint64(value), resultError("cna_media_library_create", code)
}

func nativeMediaLibraryCreateFromSource(game uint64, sourceIndex uint32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_create_from_source(
		C.CnaGoHandle(game), C.uint32_t(sourceIndex), &value))
	return uint64(value), resultError("cna_media_library_create_from_source", code)
}

func nativeMediaLibraryPictureFromToken(library uint64, token string) (uint64, bool, error) {
	arg := storageStringArg(token)
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_media_library_get_picture_from_token(
		C.CnaGoHandle(library), arg, C.uint64_t(len(token)), &value, &available))
	runtime.KeepAlive(token)
	return uint64(value), available != 0, resultError("cna_media_library_get_picture_from_token", code)
}

func nativeMediaLibrarySavePicture(library uint64, name string, image []byte) (uint64, error) {
	arg := storageStringArg(name)
	var data *C.uint8_t
	if len(image) > 0 {
		data = (*C.uint8_t)(unsafe.Pointer(&image[0]))
	}
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_library_save_picture(
		C.CnaGoHandle(library), arg, C.uint64_t(len(name)),
		data, C.uint64_t(len(image)), &value))
	runtime.KeepAlive(name)
	runtime.KeepAlive(image)
	return uint64(value), resultError("cna_media_library_save_picture", code)
}

// The media sources, which CNA addresses by INDEX rather than by handle: there
// is no CNA_MediaSourceHandle at all. Every route takes the GAME plus the
// index, so the enumeration is scoped to the running game the way every other
// create route is.

func nativeMediaSourceAvailableCount(game uint64) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_media_source_get_available_count(C.CnaGoHandle(game), &value))
	return uint32(value), resultError("cna_media_source_get_available_count", code)
}

func nativeMediaSourceTypeAt(game uint64, index uint32) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_media_source_get_type_at(
		C.CnaGoHandle(game), C.uint32_t(index), &value))
	return uint32(value), resultError("cna_media_source_get_type_at", code)
}

func nativeMediaSourceNameAt(game uint64, index uint32) (string, error) {
	return mediaSourceTextAt(game, index,
		"cna_media_source_get_name_size_at", "cna_media_source_copy_name_at")
}

func nativeMediaSourceTypeNameAt(game uint64, index uint32) (string, error) {
	return mediaSourceTextAt(game, index,
		"cna_media_source_get_type_name_size_at", "cna_media_source_copy_type_name_at")
}

// mediaSourceTextAt is the shared size-then-copy for the two string reads. It
// dispatches on the route NAME because cgo will not take a function value for a
// C symbol, and two four-line duplicates would be the alternative.
func mediaSourceTextAt(game uint64, index uint32, sizeRoute, copyRoute string) (string, error) {
	var byteCount C.uint64_t
	var code uint32
	if sizeRoute == "cna_media_source_get_name_size_at" {
		code = uint32(C.cna_go_media_source_get_name_size_at(
			C.CnaGoHandle(game), C.uint32_t(index), &byteCount))
	} else {
		code = uint32(C.cna_go_media_source_get_type_name_size_at(
			C.CnaGoHandle(game), C.uint32_t(index), &byteCount))
	}
	if err := resultError(sizeRoute, code); err != nil {
		return "", err
	}
	if byteCount == 0 {
		return "", nil
	}
	buffer := make([]byte, int(byteCount))
	var copied C.uint64_t
	destination := (*C.char)(unsafe.Pointer(&buffer[0]))
	if copyRoute == "cna_media_source_copy_name_at" {
		code = uint32(C.cna_go_media_source_copy_name_at(
			C.CnaGoHandle(game), C.uint32_t(index), destination, byteCount, &copied))
	} else {
		code = uint32(C.cna_go_media_source_copy_type_name_at(
			C.CnaGoHandle(game), C.uint32_t(index), destination, byteCount, &copied))
	}
	if err := resultError(copyRoute, code); err != nil {
		return "", err
	}
	return string(buffer[:int(copied)]), nil
}

// ---------------------------------------------------------------------------
// Foundation 97. Media playback.
//
// MediaPlayer is entirely STATIC in the contract, and every route takes the
// game -- so each wrapper resolves it from activeGame the way the media-source
// enumeration does, and none takes one from the caller.
// ---------------------------------------------------------------------------

func nativeMediaPlayerPause(game uint64) error {
	return resultError("cna_media_player_pause",
		uint32(C.cna_go_media_player_pause(C.CnaGoHandle(game))))
}

func nativeMediaPlayerResume(game uint64) error {
	return resultError("cna_media_player_resume",
		uint32(C.cna_go_media_player_resume(C.CnaGoHandle(game))))
}

func nativeMediaPlayerStop(game uint64) error {
	return resultError("cna_media_player_stop",
		uint32(C.cna_go_media_player_stop(C.CnaGoHandle(game))))
}

func nativeMediaPlayerMoveNext(game uint64) error {
	return resultError("cna_media_player_move_next",
		uint32(C.cna_go_media_player_move_next(C.CnaGoHandle(game))))
}

func nativeMediaPlayerMovePrevious(game uint64) error {
	return resultError("cna_media_player_move_previous",
		uint32(C.cna_go_media_player_move_previous(C.CnaGoHandle(game))))
}

func nativeMediaPlayerIsShuffled(game uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_player_get_is_shuffled(C.CnaGoHandle(game), &value))
	return value != 0, resultError("cna_media_player_get_is_shuffled", code)
}

func nativeMediaPlayerIsRepeating(game uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_player_get_is_repeating(C.CnaGoHandle(game), &value))
	return value != 0, resultError("cna_media_player_get_is_repeating", code)
}

func nativeMediaPlayerIsMuted(game uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_player_get_is_muted(C.CnaGoHandle(game), &value))
	return value != 0, resultError("cna_media_player_get_is_muted", code)
}

func nativeMediaPlayerIsVisualizationEnabled(game uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_player_get_is_visualization_enabled(C.CnaGoHandle(game), &value))
	return value != 0, resultError("cna_media_player_get_is_visualization_enabled", code)
}

func nativeMediaPlayerGameHasControl(game uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_media_player_get_game_has_control(C.CnaGoHandle(game), &value))
	return value != 0, resultError("cna_media_player_get_game_has_control", code)
}

func nativeMediaPlayerSetIsShuffled(game uint64, value bool) error {
	return resultError("cna_media_player_set_is_shuffled",
		uint32(C.cna_go_media_player_set_is_shuffled(C.CnaGoHandle(game), cnaBool(value))))
}

func nativeMediaPlayerSetIsRepeating(game uint64, value bool) error {
	return resultError("cna_media_player_set_is_repeating",
		uint32(C.cna_go_media_player_set_is_repeating(C.CnaGoHandle(game), cnaBool(value))))
}

func nativeMediaPlayerSetIsMuted(game uint64, value bool) error {
	return resultError("cna_media_player_set_is_muted",
		uint32(C.cna_go_media_player_set_is_muted(C.CnaGoHandle(game), cnaBool(value))))
}

func nativeMediaPlayerSetIsVisualizationEnabled(game uint64, value bool) error {
	return resultError("cna_media_player_set_is_visualization_enabled",
		uint32(C.cna_go_media_player_set_is_visualization_enabled(C.CnaGoHandle(game), cnaBool(value))))
}

func nativeMediaPlayerVolume(game uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_media_player_get_volume(C.CnaGoHandle(game), &value))
	return float32(value), resultError("cna_media_player_get_volume", code)
}

func nativeMediaPlayerSetVolume(game uint64, value float32) error {
	return resultError("cna_media_player_set_volume",
		uint32(C.cna_go_media_player_set_volume(C.CnaGoHandle(game), C.float(value))))
}

func nativeMediaPlayerState(game uint64) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_media_player_get_state(C.CnaGoHandle(game), &value))
	return uint32(value), resultError("cna_media_player_get_state", code)
}

func nativeMediaPlayerPlayPositionTicks(game uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_media_player_get_play_position_ticks(C.CnaGoHandle(game), &value))
	return int64(value), resultError("cna_media_player_get_play_position_ticks", code)
}

func nativeMediaPlayerQueue(game uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_player_get_queue(C.CnaGoHandle(game), &value))
	return uint64(value), resultError("cna_media_player_get_queue", code)
}

func nativeMediaPlayerPlaySong(game, song uint64) error {
	return resultError("cna_media_player_play_song",
		uint32(C.cna_go_media_player_play_song(C.CnaGoHandle(game), C.CnaGoHandle(song))))
}

func nativeMediaPlayerPlaySongs(game, songs uint64) error {
	return resultError("cna_media_player_play_songs",
		uint32(C.cna_go_media_player_play_songs(C.CnaGoHandle(game), C.CnaGoHandle(songs))))
}

func nativeMediaPlayerPlaySongsFrom(game, songs uint64, index int32) error {
	return resultError("cna_media_player_play_songs_from",
		uint32(C.cna_go_media_player_play_songs_from(
			C.CnaGoHandle(game), C.CnaGoHandle(songs), C.int32_t(index))))
}

// nativeMediaPlayerVisualizationData fills two 256-value buffers. The bridge
// owns the CNA struct, including its size/version prologue, so nothing about
// that layout crosses into Go.
func nativeMediaPlayerVisualizationData(game uint64) ([]float32, []float32, error) {
	const size = 256
	frequencies := make([]float32, size)
	samples := make([]float32, size)
	code := uint32(C.cna_go_media_player_get_visualization_data(C.CnaGoHandle(game),
		(*C.float)(unsafe.Pointer(&frequencies[0])),
		(*C.float)(unsafe.Pointer(&samples[0])), C.uint64_t(size)))
	if err := resultError("cna_media_player_get_visualization_data", code); err != nil {
		return nil, nil, err
	}
	return frequencies, samples, nil
}

// The two media-player event subscriptions. They take no game -- the events are
// process-wide -- and answer a registration handle the caller releases.
func nativeMediaPlayerSubscribeActiveSongChanged(context uintptr) (uint64, error) {
	var registration C.CnaGoHandle
	code := uint32(C.cna_go_media_player_subscribe_active_song_changed(
		C.uintptr_t(context), &registration))
	return uint64(registration), resultError("cna_media_player_subscribe_active_song_changed_ext", code)
}

func nativeMediaPlayerSubscribeMediaStateChanged(context uintptr) (uint64, error) {
	var registration C.CnaGoHandle
	code := uint32(C.cna_go_media_player_subscribe_media_state_changed(
		C.uintptr_t(context), &registration))
	return uint64(registration), resultError("cna_media_player_subscribe_media_state_changed_ext", code)
}

func nativeMediaPlayerUnsubscribe(registration uint64) error {
	return resultError("cna_media_player_unsubscribe_ext",
		uint32(C.cna_go_media_player_unsubscribe_ext(C.CnaGoHandle(registration))))
}

// The media queue, whose handle is BORROWED from the player rather than owned.
func nativeMediaQueueCount(queue uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_media_queue_get_count(C.CnaGoHandle(queue), &value))
	return int32(value), resultError("cna_media_queue_get_count", code)
}

func nativeMediaQueueActiveSongIndex(queue uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_media_queue_get_active_song_index(C.CnaGoHandle(queue), &value))
	return int32(value), resultError("cna_media_queue_get_active_song_index", code)
}

func nativeMediaQueueActiveSong(queue uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_media_queue_get_active_song(C.CnaGoHandle(queue), &value, &available))
	return uint64(value), available != 0, resultError("cna_media_queue_get_active_song", code)
}

func nativeMediaQueueAt(queue uint64, index int32) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_media_queue_get_at(C.CnaGoHandle(queue), C.int32_t(index), &value))
	return uint64(value), resultError("cna_media_queue_get_at", code)
}

func nativeMediaQueueDestroy(queue uint64) error {
	return resultError("cna_media_queue_destroy",
		uint32(C.cna_go_media_queue_destroy(C.CnaGoHandle(queue))))
}

func nativeVideoDurationTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_video_get_duration(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_video_get_duration", code)
}

func nativeVideoWidth(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_video_get_width(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_video_get_width", code)
}

func nativeVideoHeight(handle uint64) (int32, error) {
	var value C.int32_t
	code := uint32(C.cna_go_video_get_height(C.CnaGoHandle(handle), &value))
	return int32(value), resultError("cna_video_get_height", code)
}

func nativeVideoFramesPerSecond(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_video_get_frames_per_second(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_video_get_frames_per_second", code)
}

func nativeVideoSoundtrackType(handle uint64) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_video_get_soundtrack_type(C.CnaGoHandle(handle), &value))
	return uint32(value), resultError("cna_video_get_soundtrack_type", code)
}

func nativeVideoDestroy(handle uint64) error {
	return resultError("cna_video_destroy",
		uint32(C.cna_go_video_destroy(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerDispose(handle uint64) error {
	return resultError("cna_video_player_dispose",
		uint32(C.cna_go_video_player_dispose(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerDestroy(handle uint64) error {
	return resultError("cna_video_player_destroy",
		uint32(C.cna_go_video_player_destroy(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerPause(handle uint64) error {
	return resultError("cna_video_player_pause",
		uint32(C.cna_go_video_player_pause(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerResume(handle uint64) error {
	return resultError("cna_video_player_resume",
		uint32(C.cna_go_video_player_resume(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerStop(handle uint64) error {
	return resultError("cna_video_player_stop",
		uint32(C.cna_go_video_player_stop(C.CnaGoHandle(handle))))
}

func nativeVideoPlayerIsDisposed(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_video_player_get_is_disposed(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_video_player_get_is_disposed", code)
}

func nativeVideoPlayerIsLooped(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_video_player_get_is_looped(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_video_player_get_is_looped", code)
}

func nativeVideoPlayerIsMuted(handle uint64) (bool, error) {
	var value C.uint8_t
	code := uint32(C.cna_go_video_player_get_is_muted(C.CnaGoHandle(handle), &value))
	return value != 0, resultError("cna_video_player_get_is_muted", code)
}

func nativeVideoPlayerVolume(handle uint64) (float32, error) {
	var value C.float
	code := uint32(C.cna_go_video_player_get_volume(C.CnaGoHandle(handle), &value))
	return float32(value), resultError("cna_video_player_get_volume", code)
}

func nativeVideoPlayerState(handle uint64) (uint32, error) {
	var value C.uint32_t
	code := uint32(C.cna_go_video_player_get_state(C.CnaGoHandle(handle), &value))
	return uint32(value), resultError("cna_video_player_get_state", code)
}

func nativeVideoPlayerPlayPositionTicks(handle uint64) (int64, error) {
	var value C.int64_t
	code := uint32(C.cna_go_video_player_get_play_position_ticks(C.CnaGoHandle(handle), &value))
	return int64(value), resultError("cna_video_player_get_play_position_ticks", code)
}

func nativeVideoPlayerCreate(game uint64) (uint64, error) {
	var value C.CnaGoHandle
	code := uint32(C.cna_go_video_player_create(C.CnaGoHandle(game), &value))
	return uint64(value), resultError("cna_video_player_create", code)
}

func nativeVideoPlayerPlay(player, video uint64) error {
	return resultError("cna_video_player_play",
		uint32(C.cna_go_video_player_play(C.CnaGoHandle(player), C.CnaGoHandle(video))))
}

func nativeVideoPlayerSetIsLooped(player uint64, value bool) error {
	return resultError("cna_video_player_set_is_looped",
		uint32(C.cna_go_video_player_set_is_looped(C.CnaGoHandle(player), cnaBool(value))))
}

func nativeVideoPlayerSetIsMuted(player uint64, value bool) error {
	return resultError("cna_video_player_set_is_muted",
		uint32(C.cna_go_video_player_set_is_muted(C.CnaGoHandle(player), cnaBool(value))))
}

func nativeVideoPlayerSetVolume(player uint64, value float32) error {
	return resultError("cna_video_player_set_volume",
		uint32(C.cna_go_video_player_set_volume(C.CnaGoHandle(player), C.float(value))))
}

// nativeVideoPlayerVideo and nativeVideoPlayerTexture both report AVAILABILITY:
// a player that has never been given a video has neither.
func nativeVideoPlayerVideo(player uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_video_player_get_video(C.CnaGoHandle(player), &value, &available))
	return uint64(value), available != 0, resultError("cna_video_player_get_video", code)
}

func nativeVideoPlayerTexture(player uint64) (uint64, bool, error) {
	var value C.CnaGoHandle
	var available C.uint8_t
	code := uint32(C.cna_go_video_player_get_texture(C.CnaGoHandle(player), &value, &available))
	return uint64(value), available != 0, resultError("cna_video_player_get_texture", code)
}

//export cnaGoMediaPlayerEvent
func cnaGoMediaPlayerEvent(event C.uint32_t, context C.uintptr_t) {
	// A CNA_MediaPlayerEventCallback returns void, so nothing may cross the C
	// frame -- the same constraint the game-event trampoline works under, and
	// the same answer: record on the Runtime and surface it from Run.
	var state *Runtime
	defer func() {
		if recovered := recover(); recovered != nil && state != nil {
			state.recordCallbackFailure(
				fmt.Errorf("panic in native media-player-event trampoline: %v", recovered))
		}
	}()
	handle := cgo.Handle(context)
	state = handle.Value().(*Runtime)
	state.invokeMediaPlayerEvent(uint32(event))
}

func nativeMediaQueueSetActiveSongIndex(queue uint64, index int32) error {
	return resultError("cna_media_queue_set_active_song_index",
		uint32(C.cna_go_media_queue_set_active_song_index(C.CnaGoHandle(queue), C.int32_t(index))))
}
