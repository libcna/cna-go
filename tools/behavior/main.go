package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	audio "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Audio"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
	input "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input"
	touch "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input/Touch"
	media "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Media"
)

type observation struct {
	ID         string `json:"id"`
	Group      string `json:"group"`
	Provenance string `json:"provenance"`
	Expected   any    `json:"expected"`
	Actual     any    `json:"actual"`
	Passed     bool   `json:"passed"`
}

type corpusReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Authority     string        `json:"authority"`
	Summary       corpusSummary `json:"summary"`
	Observations  []observation `json:"observations"`
}

type corpusSummary struct {
	Observations int `json:"OBSERVATIONS"`
	Assertions   int `json:"ASSERTIONS"`
	Failures     int `json:"FAILURES"`
}

func main() {
	output := flag.String("output", "docs/generated/behavior-corpus-report.json", "report path")
	flag.Parse()
	report := runCorpus()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("OBSERVATIONS=%d\nASSERTIONS=%d\nFAILURES=%d\n", report.Summary.Observations, report.Summary.Assertions, report.Summary.Failures)
	if report.Summary.Failures != 0 {
		os.Exit(1)
	}
}

func runCorpus() corpusReport {
	report := corpusReport{SchemaVersion: 1, Authority: "PURE_XNA_DERIVED retained XNA 4.0 reference observations; GO_LANGUAGE_PROJECTION observations are labeled individually"}
	checkWithProvenance := func(id, group, provenance string, expected, actual any) {
		passed := fmt.Sprint(expected) == fmt.Sprint(actual)
		report.Observations = append(report.Observations, observation{ID: id, Group: group, Provenance: provenance, Expected: expected, Actual: actual, Passed: passed})
		report.Summary.Observations++
		report.Summary.Assertions++
		if !passed {
			report.Summary.Failures++
		}
	}
	check := func(id, group string, expected, actual any) {
		checkWithProvenance(id, group, "PURE_XNA_DERIVED", expected, actual)
	}
	checkGoProjection := func(id, group string, expected, actual any) {
		checkWithProvenance(id, group, "GO_LANGUAGE_PROJECTION", expected, actual)
	}
	bits := func(value float32) string { return fmt.Sprintf("0x%08x", math.Float32bits(value)) }
	floatBits := func(values ...float32) string {
		result := make([]string, len(values))
		for i, value := range values {
			result[i] = bits(value)
		}
		return strings.Join(result, ",")
	}
	packed := func(value uint32) string { return fmt.Sprintf("0x%08x", value) }

	check("player-index.defined-raw-values", "PLAYER_INDEX", "0,1,2,3", fmt.Sprintf("%d,%d,%d,%d", framework.PlayerIndexOne, framework.PlayerIndexTwo, framework.PlayerIndexThree, framework.PlayerIndexFour))
	check("player-index.undefined-raw-value", "PLAYER_INDEX", int32(12345), int32(framework.PlayerIndex(12345)))

	noPlayerState, noPlayerError := input.KeyboardGetStateByNone()
	for _, fixture := range []struct {
		name  string
		value framework.PlayerIndex
	}{
		{"one", framework.PlayerIndexOne},
		{"two", framework.PlayerIndexTwo},
		{"three", framework.PlayerIndexThree},
		{"four", framework.PlayerIndexFour},
		{"undefined", framework.PlayerIndex(12345)},
	} {
		state, err := input.KeyboardGetStateByPlayerIndex(fixture.value)
		sameError := noPlayerError != nil && err != nil && noPlayerError.Error() == err.Error()
		check("keyboard.player-index."+fixture.name, "KEYBOARD_PLAYER_INDEX", "true,true", fmt.Sprintf("%t,%t", state == noPlayerState, sameError))
	}

	check("display-orientation.raw-values", "DISPLAY_ORIENTATION", "0,1,2,4", fmt.Sprintf("%d,%d,%d,%d", framework.DisplayOrientationDefault, framework.DisplayOrientationLandscapeLeft, framework.DisplayOrientationLandscapeRight, framework.DisplayOrientationPortrait))
	landscape := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationLandscapeRight
	leftPortrait := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	allOrientations := landscape | framework.DisplayOrientationPortrait
	check("display-orientation.flags-combinations", "DISPLAY_ORIENTATION", "3,5,7", fmt.Sprintf("%d,%d,%d", landscape, leftPortrait, allOrientations))
	check("display-orientation.unknown-raw-bit", "DISPLAY_ORIENTATION", int32(1<<20), int32(framework.DisplayOrientation(1<<20)))

	managerDirty := func(manager *framework.GraphicsDeviceManager) bool {
		return reflect.ValueOf(manager).Elem().FieldByName("isDeviceDirty").Bool()
	}
	manager := &framework.GraphicsDeviceManager{}
	check("graphics-manager-orientation.initial-state", "GRAPHICS_MANAGER_ORIENTATION", "0,false", fmt.Sprintf("%d,%t", manager.SupportedOrientations(), managerDirty(manager)))
	manager.SetSupportedOrientations(framework.DisplayOrientationDefault)
	check("graphics-manager-orientation.same-value-dirty", "GRAPHICS_MANAGER_ORIENTATION", "0,true", fmt.Sprintf("%d,%t", manager.SupportedOrientations(), managerDirty(manager)))
	changedManager := &framework.GraphicsDeviceManager{}
	changedManager.SetSupportedOrientations(leftPortrait)
	check("graphics-manager-orientation.changed-value-dirty", "GRAPHICS_MANAGER_ORIENTATION", "5,true", fmt.Sprintf("%d,%t", changedManager.SupportedOrientations(), managerDirty(changedManager)))
	changedManager.SetSupportedOrientations(framework.DisplayOrientationLandscapeRight)
	changedManager.SetSupportedOrientations(framework.DisplayOrientation(1 << 20))
	check("graphics-manager-orientation.multiple-assignment", "GRAPHICS_MANAGER_ORIENTATION", "1048576,true", fmt.Sprintf("%d,%t", changedManager.SupportedOrientations(), managerDirty(changedManager)))
	postDisposeManager := &framework.GraphicsDeviceManager{}
	_ = postDisposeManager.Dispose(true)
	postDisposeManager.SetSupportedOrientations(landscape)
	check("graphics-manager-orientation.post-disposal-managed-state", "GRAPHICS_MANAGER_ORIENTATION", "3,true", fmt.Sprintf("%d,%t", postDisposeManager.SupportedOrientations(), managerDirty(postDisposeManager)))

	check("buffer-usage.raw-values", "BUFFER_USAGE", "0,1", fmt.Sprintf("%d,%d", graphics.BufferUsageNone, graphics.BufferUsageWriteOnly))
	check("buffer-usage.underlying-int32", "BUFFER_USAGE", "int32", reflect.TypeOf(graphics.BufferUsageNone).Kind().String())
	var zeroBufferUsage graphics.BufferUsage
	checkGoProjection("buffer-usage.zero-value", "BUFFER_USAGE", graphics.BufferUsageNone, zeroBufferUsage)
	checkGoProjection("buffer-usage.arbitrary-raw-values", "BUFFER_USAGE", "2,3,1048576,-1", fmt.Sprintf("%d,%d,%d,%d", graphics.BufferUsage(2), graphics.BufferUsage(3), graphics.BufferUsage(1<<20), graphics.BufferUsage(-1)))
	checkGoProjection("buffer-usage.bitwise-composition", "BUFFER_USAGE", "1,1,3", fmt.Sprintf("%d,%d,%d", graphics.BufferUsageNone|graphics.BufferUsageWriteOnly, graphics.BufferUsageWriteOnly|graphics.BufferUsageWriteOnly, graphics.BufferUsage(2)|graphics.BufferUsageWriteOnly))

	check("clear-options.raw-values", "CLEAR_OPTIONS", "1,2,4", fmt.Sprintf("%d,%d,%d", graphics.ClearOptionsTarget, graphics.ClearOptionsDepthBuffer, graphics.ClearOptionsStencil))
	clearOptionsKind := reflect.TypeOf(graphics.ClearOptionsTarget).Kind()
	check("clear-options.underlying-system-int32", "CLEAR_OPTIONS", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[clearOptionsKind])
	clearOptionsPowersOfTwo := graphics.ClearOptionsTarget != 0 && graphics.ClearOptionsTarget&(graphics.ClearOptionsTarget-1) == 0 &&
		graphics.ClearOptionsDepthBuffer != 0 && graphics.ClearOptionsDepthBuffer&(graphics.ClearOptionsDepthBuffer-1) == 0 &&
		graphics.ClearOptionsStencil != 0 && graphics.ClearOptionsStencil&(graphics.ClearOptionsStencil-1) == 0
	check("clear-options.flags-metadata-shape", "CLEAR_OPTIONS", true, clearOptionsPowersOfTwo)
	check("clear-options.no-declared-zero-literal", "CLEAR_OPTIONS", true, graphics.ClearOptionsTarget != 0 && graphics.ClearOptionsDepthBuffer != 0 && graphics.ClearOptionsStencil != 0)
	var zeroClearOptions graphics.ClearOptions
	checkGoProjection("clear-options.unnamed-zero-value", "CLEAR_OPTIONS", int32(0), int32(zeroClearOptions))
	checkGoProjection("clear-options.declared-combinations", "CLEAR_OPTIONS", "3,5,6,7", fmt.Sprintf("%d,%d,%d,%d", graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer, graphics.ClearOptionsTarget|graphics.ClearOptionsStencil, graphics.ClearOptionsDepthBuffer|graphics.ClearOptionsStencil, graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer|graphics.ClearOptionsStencil))
	checkGoProjection("clear-options.arbitrary-raw-values", "CLEAR_OPTIONS", "0,8,1048576,-1", fmt.Sprintf("%d,%d,%d,%d", graphics.ClearOptions(0), graphics.ClearOptions(8), graphics.ClearOptions(1<<20), graphics.ClearOptions(-1)))
	checkGoProjection("clear-options.bitwise-or", "CLEAR_OPTIONS", "3,9", fmt.Sprintf("%d,%d", graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer, graphics.ClearOptions(8)|graphics.ClearOptionsTarget))
	checkGoProjection("clear-options.bitwise-and", "CLEAR_OPTIONS", "4,0", fmt.Sprintf("%d,%d", graphics.ClearOptions(7)&graphics.ClearOptionsStencil, graphics.ClearOptions(2)&graphics.ClearOptionsTarget))

	check("surface-format.complete-raw-table", "SURFACE_FORMAT", "Color=0,Bgr565=1,Bgra5551=2,Bgra4444=3,Dxt1=4,Dxt3=5,Dxt5=6,NormalizedByte2=7,NormalizedByte4=8,Rgba1010102=9,Rg32=10,Rgba64=11,Alpha8=12,Single=13,Vector2=14,Vector4=15,HalfSingle=16,HalfVector2=17,HalfVector4=18,HdrBlendable=19", fmt.Sprintf(
		"Color=%d,Bgr565=%d,Bgra5551=%d,Bgra4444=%d,Dxt1=%d,Dxt3=%d,Dxt5=%d,NormalizedByte2=%d,NormalizedByte4=%d,Rgba1010102=%d,Rg32=%d,Rgba64=%d,Alpha8=%d,Single=%d,Vector2=%d,Vector4=%d,HalfSingle=%d,HalfVector2=%d,HalfVector4=%d,HdrBlendable=%d",
		graphics.SurfaceFormatColor,
		graphics.SurfaceFormatBgr565,
		graphics.SurfaceFormatBgra5551,
		graphics.SurfaceFormatBgra4444,
		graphics.SurfaceFormatDxt1,
		graphics.SurfaceFormatDxt3,
		graphics.SurfaceFormatDxt5,
		graphics.SurfaceFormatNormalizedByte2,
		graphics.SurfaceFormatNormalizedByte4,
		graphics.SurfaceFormatRgba1010102,
		graphics.SurfaceFormatRg32,
		graphics.SurfaceFormatRgba64,
		graphics.SurfaceFormatAlpha8,
		graphics.SurfaceFormatSingle,
		graphics.SurfaceFormatVector2,
		graphics.SurfaceFormatVector4,
		graphics.SurfaceFormatHalfSingle,
		graphics.SurfaceFormatHalfVector2,
		graphics.SurfaceFormatHalfVector4,
		graphics.SurfaceFormatHdrBlendable,
	))
	surfaceFormatKind := reflect.TypeOf(graphics.SurfaceFormatColor).Kind()
	check("surface-format.underlying-system-int32", "SURFACE_FORMAT", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[surfaceFormatKind])
	check("surface-format.flags", "SURFACE_FORMAT", false, false)
	var zeroSurfaceFormat graphics.SurfaceFormat
	checkGoProjection("surface-format.zero-value-color", "SURFACE_FORMAT", graphics.SurfaceFormatColor, zeroSurfaceFormat)
	checkGoProjection("surface-format.arbitrary-positive-raw", "SURFACE_FORMAT", "20,12345", fmt.Sprintf("%d,%d", graphics.SurfaceFormat(20), graphics.SurfaceFormat(12345)))
	checkGoProjection("surface-format.negative-raw", "SURFACE_FORMAT", int32(-1), int32(graphics.SurfaceFormat(-1)))

	check("depth-format.complete-raw-table", "DEPTH_FORMAT", "None=0,Depth16=1,Depth24=2,Depth24Stencil8=3", fmt.Sprintf(
		"None=%d,Depth16=%d,Depth24=%d,Depth24Stencil8=%d",
		graphics.DepthFormatNone,
		graphics.DepthFormatDepth16,
		graphics.DepthFormatDepth24,
		graphics.DepthFormatDepth24Stencil8,
	))
	depthFormatKind := reflect.TypeOf(graphics.DepthFormatNone).Kind()
	check("depth-format.underlying-system-int32", "DEPTH_FORMAT", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[depthFormatKind])
	check("depth-format.flags", "DEPTH_FORMAT", false, false)
	var zeroDepthFormat graphics.DepthFormat
	checkGoProjection("depth-format.zero-value-none", "DEPTH_FORMAT", graphics.DepthFormatNone, zeroDepthFormat)
	checkGoProjection("depth-format.arbitrary-positive-raw", "DEPTH_FORMAT", "4,12345", fmt.Sprintf("%d,%d", graphics.DepthFormat(4), graphics.DepthFormat(12345)))
	checkGoProjection("depth-format.negative-raw", "DEPTH_FORMAT", int32(-1), int32(graphics.DepthFormat(-1)))

	check("graphics-profile.complete-raw-table", "GRAPHICS_PROFILE", "Reach=0,HiDef=1", fmt.Sprintf(
		"Reach=%d,HiDef=%d",
		graphics.GraphicsProfileReach,
		graphics.GraphicsProfileHiDef,
	))
	graphicsProfileKind := reflect.TypeOf(graphics.GraphicsProfileReach).Kind()
	check("graphics-profile.underlying-system-int32", "GRAPHICS_PROFILE", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[graphicsProfileKind])
	check("graphics-profile.flags", "GRAPHICS_PROFILE", false, false)
	var zeroGraphicsProfile graphics.GraphicsProfile
	checkGoProjection("graphics-profile.zero-value-reach", "GRAPHICS_PROFILE", graphics.GraphicsProfileReach, zeroGraphicsProfile)
	checkGoProjection("graphics-profile.arbitrary-positive-raw", "GRAPHICS_PROFILE", "2,12345", fmt.Sprintf("%d,%d", graphics.GraphicsProfile(2), graphics.GraphicsProfile(12345)))
	checkGoProjection("graphics-profile.negative-raw", "GRAPHICS_PROFILE", int32(-1), int32(graphics.GraphicsProfile(-1)))

	check("button-state.complete-raw-table", "BUTTON_STATE", "Released=0,Pressed=1", fmt.Sprintf(
		"Released=%d,Pressed=%d",
		input.ButtonStateReleased,
		input.ButtonStatePressed,
	))
	buttonStateKind := reflect.TypeOf(input.ButtonStateReleased).Kind()
	check("button-state.underlying-system-int32", "BUTTON_STATE", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[buttonStateKind])
	check("button-state.flags", "BUTTON_STATE", false, false)
	var zeroButtonState input.ButtonState
	checkGoProjection("button-state.zero-value-released", "BUTTON_STATE", input.ButtonStateReleased, zeroButtonState)
	checkGoProjection("button-state.arbitrary-positive-raw", "BUTTON_STATE", "2,12345", fmt.Sprintf("%d,%d", input.ButtonState(2), input.ButtonState(12345)))
	checkGoProjection("button-state.negative-raw", "BUTTON_STATE", int32(-1), int32(input.ButtonState(-1)))

	// Foundation 14 pure-managed batch: 25 ordinary and flags enums whose only
	// public-signature dependency is System.Int32. Every raw value asserted below
	// is pinned XNA 4.0 Windows metadata. Completing these enums is a metadata
	// fact only; it proves no renderer, device, audio, media, touch, or game pad
	// runtime capability, and no backend route exists for any of them.

	check("render-target-usage.complete-raw-table", "RENDER_TARGET_USAGE",
		"DiscardContents=0,PreserveContents=1,PlatformContents=2",
		fmt.Sprintf("DiscardContents=%d,PreserveContents=%d,PlatformContents=%d",
			graphics.RenderTargetUsageDiscardContents, graphics.RenderTargetUsagePreserveContents, graphics.RenderTargetUsagePlatformContents))
	check("render-target-usage.underlying-system-int32", "RENDER_TARGET_USAGE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.RenderTargetUsageDiscardContents).Kind()])
	var zeroRenderTargetUsage graphics.RenderTargetUsage
	checkGoProjection("render-target-usage.zero-value-discard-contents", "RENDER_TARGET_USAGE", graphics.RenderTargetUsageDiscardContents, zeroRenderTargetUsage)
	checkGoProjection("render-target-usage.arbitrary-positive-raw", "RENDER_TARGET_USAGE", "3,12345",
		fmt.Sprintf("%d,%d", graphics.RenderTargetUsage(3), graphics.RenderTargetUsage(12345)))
	checkGoProjection("render-target-usage.negative-raw", "RENDER_TARGET_USAGE", int32(-1), int32(graphics.RenderTargetUsage(-1)))

	check("cube-map-face.complete-raw-table", "CUBE_MAP_FACE",
		"PositiveX=0,NegativeX=1,PositiveY=2,NegativeY=3,PositiveZ=4,NegativeZ=5",
		fmt.Sprintf("PositiveX=%d,NegativeX=%d,PositiveY=%d,NegativeY=%d,PositiveZ=%d,NegativeZ=%d",
			graphics.CubeMapFacePositiveX, graphics.CubeMapFaceNegativeX, graphics.CubeMapFacePositiveY, graphics.CubeMapFaceNegativeY, graphics.CubeMapFacePositiveZ, graphics.CubeMapFaceNegativeZ))
	check("cube-map-face.underlying-system-int32", "CUBE_MAP_FACE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.CubeMapFacePositiveX).Kind()])
	var zeroCubeMapFace graphics.CubeMapFace
	checkGoProjection("cube-map-face.zero-value-positive-x", "CUBE_MAP_FACE", graphics.CubeMapFacePositiveX, zeroCubeMapFace)
	checkGoProjection("cube-map-face.arbitrary-positive-raw", "CUBE_MAP_FACE", "6,12345",
		fmt.Sprintf("%d,%d", graphics.CubeMapFace(6), graphics.CubeMapFace(12345)))
	checkGoProjection("cube-map-face.negative-raw", "CUBE_MAP_FACE", int32(-1), int32(graphics.CubeMapFace(-1)))

	check("audio-channels.complete-raw-table", "AUDIO_CHANNELS",
		"Mono=1,Stereo=2",
		fmt.Sprintf("Mono=%d,Stereo=%d",
			audio.AudioChannelsMono, audio.AudioChannelsStereo))
	check("audio-channels.underlying-system-int32", "AUDIO_CHANNELS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(audio.AudioChannelsMono).Kind()])
	var zeroAudioChannels audio.AudioChannels
	// The pinned contract declares no zero literal, so the Go zero value is
	// an ordinary undefined raw value rather than a named constant.
	checkGoProjection("audio-channels.zero-value-unnamed", "AUDIO_CHANNELS", "0,false,false",
		fmt.Sprintf("%d,%t,%t", int32(zeroAudioChannels), zeroAudioChannels == audio.AudioChannelsMono, zeroAudioChannels == audio.AudioChannelsStereo))
	checkGoProjection("audio-channels.arbitrary-positive-raw", "AUDIO_CHANNELS", "3,12345",
		fmt.Sprintf("%d,%d", audio.AudioChannels(3), audio.AudioChannels(12345)))
	checkGoProjection("audio-channels.negative-raw", "AUDIO_CHANNELS", int32(-1), int32(audio.AudioChannels(-1)))

	check("audio-stop-options.complete-raw-table", "AUDIO_STOP_OPTIONS",
		"AsAuthored=0,Immediate=1",
		fmt.Sprintf("AsAuthored=%d,Immediate=%d",
			audio.AudioStopOptionsAsAuthored, audio.AudioStopOptionsImmediate))
	check("audio-stop-options.underlying-system-int32", "AUDIO_STOP_OPTIONS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(audio.AudioStopOptionsAsAuthored).Kind()])
	var zeroAudioStopOptions audio.AudioStopOptions
	checkGoProjection("audio-stop-options.zero-value-as-authored", "AUDIO_STOP_OPTIONS", audio.AudioStopOptionsAsAuthored, zeroAudioStopOptions)
	checkGoProjection("audio-stop-options.arbitrary-positive-raw", "AUDIO_STOP_OPTIONS", "2,12345",
		fmt.Sprintf("%d,%d", audio.AudioStopOptions(2), audio.AudioStopOptions(12345)))
	checkGoProjection("audio-stop-options.negative-raw", "AUDIO_STOP_OPTIONS", int32(-1), int32(audio.AudioStopOptions(-1)))

	check("index-element-size.complete-raw-table", "INDEX_ELEMENT_SIZE",
		"SixteenBits=0,ThirtyTwoBits=1",
		fmt.Sprintf("SixteenBits=%d,ThirtyTwoBits=%d",
			graphics.IndexElementSizeSixteenBits, graphics.IndexElementSizeThirtyTwoBits))
	check("index-element-size.underlying-system-int32", "INDEX_ELEMENT_SIZE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.IndexElementSizeSixteenBits).Kind()])
	var zeroIndexElementSize graphics.IndexElementSize
	checkGoProjection("index-element-size.zero-value-sixteen-bits", "INDEX_ELEMENT_SIZE", graphics.IndexElementSizeSixteenBits, zeroIndexElementSize)
	checkGoProjection("index-element-size.arbitrary-positive-raw", "INDEX_ELEMENT_SIZE", "2,12345",
		fmt.Sprintf("%d,%d", graphics.IndexElementSize(2), graphics.IndexElementSize(12345)))
	checkGoProjection("index-element-size.negative-raw", "INDEX_ELEMENT_SIZE", int32(-1), int32(graphics.IndexElementSize(-1)))

	check("set-data-options.complete-raw-table", "SET_DATA_OPTIONS",
		"None=0,Discard=1,NoOverwrite=2",
		fmt.Sprintf("None=%d,Discard=%d,NoOverwrite=%d",
			graphics.SetDataOptionsNone, graphics.SetDataOptionsDiscard, graphics.SetDataOptionsNoOverwrite))
	check("set-data-options.underlying-system-int32", "SET_DATA_OPTIONS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.SetDataOptionsNone).Kind()])
	setDataOptionsUnion, setDataOptionsSingleBits := enumBitStructure([]int32{int32(graphics.SetDataOptionsNone), int32(graphics.SetDataOptionsDiscard), int32(graphics.SetDataOptionsNoOverwrite)})
	check("set-data-options.flags-union", "SET_DATA_OPTIONS", int32(3), setDataOptionsUnion)
	check("set-data-options.flags-disjoint-single-bits", "SET_DATA_OPTIONS", true, setDataOptionsSingleBits)
	var zeroSetDataOptions graphics.SetDataOptions
	checkGoProjection("set-data-options.zero-value-none", "SET_DATA_OPTIONS", graphics.SetDataOptionsNone, zeroSetDataOptions)
	checkGoProjection("set-data-options.arbitrary-positive-raw", "SET_DATA_OPTIONS", "3,12345",
		fmt.Sprintf("%d,%d", graphics.SetDataOptions(3), graphics.SetDataOptions(12345)))
	checkGoProjection("set-data-options.negative-raw", "SET_DATA_OPTIONS", int32(-1), int32(graphics.SetDataOptions(-1)))

	check("media-state.complete-raw-table", "MEDIA_STATE",
		"Stopped=0,Playing=1,Paused=2",
		fmt.Sprintf("Stopped=%d,Playing=%d,Paused=%d",
			media.MediaStateStopped, media.MediaStatePlaying, media.MediaStatePaused))
	check("media-state.underlying-system-int32", "MEDIA_STATE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(media.MediaStateStopped).Kind()])
	var zeroMediaState media.MediaState
	checkGoProjection("media-state.zero-value-stopped", "MEDIA_STATE", media.MediaStateStopped, zeroMediaState)
	checkGoProjection("media-state.arbitrary-positive-raw", "MEDIA_STATE", "3,12345",
		fmt.Sprintf("%d,%d", media.MediaState(3), media.MediaState(12345)))
	checkGoProjection("media-state.negative-raw", "MEDIA_STATE", int32(-1), int32(media.MediaState(-1)))

	check("effect-parameter-class.complete-raw-table", "EFFECT_PARAMETER_CLASS",
		"Scalar=0,Vector=1,Matrix=2,Object=3,Struct=4",
		fmt.Sprintf("Scalar=%d,Vector=%d,Matrix=%d,Object=%d,Struct=%d",
			graphics.EffectParameterClassScalar, graphics.EffectParameterClassVector, graphics.EffectParameterClassMatrix, graphics.EffectParameterClassObject, graphics.EffectParameterClassStruct))
	check("effect-parameter-class.underlying-system-int32", "EFFECT_PARAMETER_CLASS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.EffectParameterClassScalar).Kind()])
	var zeroEffectParameterClass graphics.EffectParameterClass
	checkGoProjection("effect-parameter-class.zero-value-scalar", "EFFECT_PARAMETER_CLASS", graphics.EffectParameterClassScalar, zeroEffectParameterClass)
	checkGoProjection("effect-parameter-class.arbitrary-positive-raw", "EFFECT_PARAMETER_CLASS", "5,12345",
		fmt.Sprintf("%d,%d", graphics.EffectParameterClass(5), graphics.EffectParameterClass(12345)))
	checkGoProjection("effect-parameter-class.negative-raw", "EFFECT_PARAMETER_CLASS", int32(-1), int32(graphics.EffectParameterClass(-1)))

	check("compare-function.complete-raw-table", "COMPARE_FUNCTION",
		"Always=0,Never=1,Less=2,LessEqual=3,Equal=4,GreaterEqual=5,Greater=6,NotEqual=7",
		fmt.Sprintf("Always=%d,Never=%d,Less=%d,LessEqual=%d,Equal=%d,GreaterEqual=%d,Greater=%d,NotEqual=%d",
			graphics.CompareFunctionAlways, graphics.CompareFunctionNever, graphics.CompareFunctionLess, graphics.CompareFunctionLessEqual, graphics.CompareFunctionEqual, graphics.CompareFunctionGreaterEqual, graphics.CompareFunctionGreater, graphics.CompareFunctionNotEqual))
	check("compare-function.underlying-system-int32", "COMPARE_FUNCTION", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.CompareFunctionAlways).Kind()])
	var zeroCompareFunction graphics.CompareFunction
	checkGoProjection("compare-function.zero-value-always", "COMPARE_FUNCTION", graphics.CompareFunctionAlways, zeroCompareFunction)
	checkGoProjection("compare-function.arbitrary-positive-raw", "COMPARE_FUNCTION", "8,12345",
		fmt.Sprintf("%d,%d", graphics.CompareFunction(8), graphics.CompareFunction(12345)))
	checkGoProjection("compare-function.negative-raw", "COMPARE_FUNCTION", int32(-1), int32(graphics.CompareFunction(-1)))

	check("effect-parameter-type.complete-raw-table", "EFFECT_PARAMETER_TYPE",
		"Void=0,Bool=1,Int32=2,Single=3,String=4,Texture=5,Texture1D=6,Texture2D=7,Texture3D=8,TextureCube=9",
		fmt.Sprintf("Void=%d,Bool=%d,Int32=%d,Single=%d,String=%d,Texture=%d,Texture1D=%d,Texture2D=%d,Texture3D=%d,TextureCube=%d",
			graphics.EffectParameterTypeVoid, graphics.EffectParameterTypeBool, graphics.EffectParameterTypeInt32, graphics.EffectParameterTypeSingle, graphics.EffectParameterTypeString, graphics.EffectParameterTypeTexture, graphics.EffectParameterTypeTexture1D, graphics.EffectParameterTypeTexture2D, graphics.EffectParameterTypeTexture3D, graphics.EffectParameterTypeTextureCube))
	check("effect-parameter-type.underlying-system-int32", "EFFECT_PARAMETER_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.EffectParameterTypeVoid).Kind()])
	var zeroEffectParameterType graphics.EffectParameterType
	checkGoProjection("effect-parameter-type.zero-value-void", "EFFECT_PARAMETER_TYPE", graphics.EffectParameterTypeVoid, zeroEffectParameterType)
	checkGoProjection("effect-parameter-type.arbitrary-positive-raw", "EFFECT_PARAMETER_TYPE", "10,12345",
		fmt.Sprintf("%d,%d", graphics.EffectParameterType(10), graphics.EffectParameterType(12345)))
	checkGoProjection("effect-parameter-type.negative-raw", "EFFECT_PARAMETER_TYPE", int32(-1), int32(graphics.EffectParameterType(-1)))

	check("gesture-type.complete-raw-table", "GESTURE_TYPE",
		"None=0,Tap=1,DoubleTap=2,Hold=4,HorizontalDrag=8,VerticalDrag=16,FreeDrag=32,Pinch=64,Flick=128,DragComplete=256,PinchComplete=512",
		fmt.Sprintf("None=%d,Tap=%d,DoubleTap=%d,Hold=%d,HorizontalDrag=%d,VerticalDrag=%d,FreeDrag=%d,Pinch=%d,Flick=%d,DragComplete=%d,PinchComplete=%d",
			touch.GestureTypeNone, touch.GestureTypeTap, touch.GestureTypeDoubleTap, touch.GestureTypeHold, touch.GestureTypeHorizontalDrag, touch.GestureTypeVerticalDrag, touch.GestureTypeFreeDrag, touch.GestureTypePinch, touch.GestureTypeFlick, touch.GestureTypeDragComplete, touch.GestureTypePinchComplete))
	check("gesture-type.underlying-system-int32", "GESTURE_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(touch.GestureTypeNone).Kind()])
	gestureTypeUnion, gestureTypeSingleBits := enumBitStructure([]int32{int32(touch.GestureTypeNone), int32(touch.GestureTypeTap), int32(touch.GestureTypeDoubleTap), int32(touch.GestureTypeHold), int32(touch.GestureTypeHorizontalDrag), int32(touch.GestureTypeVerticalDrag), int32(touch.GestureTypeFreeDrag), int32(touch.GestureTypePinch), int32(touch.GestureTypeFlick), int32(touch.GestureTypeDragComplete), int32(touch.GestureTypePinchComplete)})
	check("gesture-type.flags-union", "GESTURE_TYPE", int32(1023), gestureTypeUnion)
	check("gesture-type.flags-disjoint-single-bits", "GESTURE_TYPE", true, gestureTypeSingleBits)
	var zeroGestureType touch.GestureType
	checkGoProjection("gesture-type.zero-value-none", "GESTURE_TYPE", touch.GestureTypeNone, zeroGestureType)
	checkGoProjection("gesture-type.arbitrary-positive-raw", "GESTURE_TYPE", "513,12345",
		fmt.Sprintf("%d,%d", touch.GestureType(513), touch.GestureType(12345)))
	checkGoProjection("gesture-type.negative-raw", "GESTURE_TYPE", int32(-1), int32(touch.GestureType(-1)))

	check("buttons.complete-raw-table", "BUTTONS",
		"DPadUp=1,DPadDown=2,DPadLeft=4,DPadRight=8,Start=16,Back=32,LeftStick=64,RightStick=128,LeftShoulder=256,RightShoulder=512,BigButton=2048,A=4096,B=8192,X=16384,Y=32768,RightThumbstickUp=16777216,RightThumbstickDown=33554432,RightThumbstickRight=67108864,RightThumbstickLeft=134217728,LeftThumbstickUp=268435456,LeftThumbstickDown=536870912,LeftThumbstickRight=1073741824,LeftThumbstickLeft=2097152,LeftTrigger=8388608,RightTrigger=4194304",
		fmt.Sprintf("DPadUp=%d,DPadDown=%d,DPadLeft=%d,DPadRight=%d,Start=%d,Back=%d,LeftStick=%d,RightStick=%d,LeftShoulder=%d,RightShoulder=%d,BigButton=%d,A=%d,B=%d,X=%d,Y=%d,RightThumbstickUp=%d,RightThumbstickDown=%d,RightThumbstickRight=%d,RightThumbstickLeft=%d,LeftThumbstickUp=%d,LeftThumbstickDown=%d,LeftThumbstickRight=%d,LeftThumbstickLeft=%d,LeftTrigger=%d,RightTrigger=%d",
			input.ButtonsDPadUp, input.ButtonsDPadDown, input.ButtonsDPadLeft, input.ButtonsDPadRight, input.ButtonsStart, input.ButtonsBack, input.ButtonsLeftStick, input.ButtonsRightStick, input.ButtonsLeftShoulder, input.ButtonsRightShoulder, input.ButtonsBigButton, input.ButtonsA, input.ButtonsB, input.ButtonsX, input.ButtonsY, input.ButtonsRightThumbstickUp, input.ButtonsRightThumbstickDown, input.ButtonsRightThumbstickRight, input.ButtonsRightThumbstickLeft, input.ButtonsLeftThumbstickUp, input.ButtonsLeftThumbstickDown, input.ButtonsLeftThumbstickRight, input.ButtonsLeftThumbstickLeft, input.ButtonsLeftTrigger, input.ButtonsRightTrigger))
	check("buttons.underlying-system-int32", "BUTTONS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(input.ButtonsDPadUp).Kind()])
	buttonsUnion, buttonsSingleBits := enumBitStructure([]int32{int32(input.ButtonsDPadUp), int32(input.ButtonsDPadDown), int32(input.ButtonsDPadLeft), int32(input.ButtonsDPadRight), int32(input.ButtonsStart), int32(input.ButtonsBack), int32(input.ButtonsLeftStick), int32(input.ButtonsRightStick), int32(input.ButtonsLeftShoulder), int32(input.ButtonsRightShoulder), int32(input.ButtonsBigButton), int32(input.ButtonsA), int32(input.ButtonsB), int32(input.ButtonsX), int32(input.ButtonsY), int32(input.ButtonsRightThumbstickUp), int32(input.ButtonsRightThumbstickDown), int32(input.ButtonsRightThumbstickRight), int32(input.ButtonsRightThumbstickLeft), int32(input.ButtonsLeftThumbstickUp), int32(input.ButtonsLeftThumbstickDown), int32(input.ButtonsLeftThumbstickRight), int32(input.ButtonsLeftThumbstickLeft), int32(input.ButtonsLeftTrigger), int32(input.ButtonsRightTrigger)})
	check("buttons.flags-union", "BUTTONS", int32(2145451007), buttonsUnion)
	check("buttons.flags-disjoint-single-bits", "BUTTONS", true, buttonsSingleBits)
	var zeroButtons input.Buttons
	// The pinned contract declares no zero literal, so the Go zero value is
	// an ordinary undefined raw value rather than a named constant.
	checkGoProjection("buttons.zero-value-unnamed", "BUTTONS", "0,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false,false",
		fmt.Sprintf("%d,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t,%t", int32(zeroButtons), zeroButtons == input.ButtonsDPadUp, zeroButtons == input.ButtonsDPadDown, zeroButtons == input.ButtonsDPadLeft, zeroButtons == input.ButtonsDPadRight, zeroButtons == input.ButtonsStart, zeroButtons == input.ButtonsBack, zeroButtons == input.ButtonsLeftStick, zeroButtons == input.ButtonsRightStick, zeroButtons == input.ButtonsLeftShoulder, zeroButtons == input.ButtonsRightShoulder, zeroButtons == input.ButtonsBigButton, zeroButtons == input.ButtonsA, zeroButtons == input.ButtonsB, zeroButtons == input.ButtonsX, zeroButtons == input.ButtonsY, zeroButtons == input.ButtonsRightThumbstickUp, zeroButtons == input.ButtonsRightThumbstickDown, zeroButtons == input.ButtonsRightThumbstickRight, zeroButtons == input.ButtonsRightThumbstickLeft, zeroButtons == input.ButtonsLeftThumbstickUp, zeroButtons == input.ButtonsLeftThumbstickDown, zeroButtons == input.ButtonsLeftThumbstickRight, zeroButtons == input.ButtonsLeftThumbstickLeft, zeroButtons == input.ButtonsLeftTrigger, zeroButtons == input.ButtonsRightTrigger))
	checkGoProjection("buttons.arbitrary-positive-raw", "BUTTONS", "1073741825,12345",
		fmt.Sprintf("%d,%d", input.Buttons(1073741825), input.Buttons(12345)))
	checkGoProjection("buttons.negative-raw", "BUTTONS", int32(-1), int32(input.Buttons(-1)))

	check("microphone-state.complete-raw-table", "MICROPHONE_STATE",
		"Started=0,Stopped=1",
		fmt.Sprintf("Started=%d,Stopped=%d",
			audio.MicrophoneStateStarted, audio.MicrophoneStateStopped))
	check("microphone-state.underlying-system-int32", "MICROPHONE_STATE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(audio.MicrophoneStateStarted).Kind()])
	var zeroMicrophoneState audio.MicrophoneState
	checkGoProjection("microphone-state.zero-value-started", "MICROPHONE_STATE", audio.MicrophoneStateStarted, zeroMicrophoneState)
	checkGoProjection("microphone-state.arbitrary-positive-raw", "MICROPHONE_STATE", "2,12345",
		fmt.Sprintf("%d,%d", audio.MicrophoneState(2), audio.MicrophoneState(12345)))
	checkGoProjection("microphone-state.negative-raw", "MICROPHONE_STATE", int32(-1), int32(audio.MicrophoneState(-1)))

	check("fill-mode.complete-raw-table", "FILL_MODE",
		"Solid=0,WireFrame=1",
		fmt.Sprintf("Solid=%d,WireFrame=%d",
			graphics.FillModeSolid, graphics.FillModeWireFrame))
	check("fill-mode.underlying-system-int32", "FILL_MODE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.FillModeSolid).Kind()])
	var zeroFillMode graphics.FillMode
	checkGoProjection("fill-mode.zero-value-solid", "FILL_MODE", graphics.FillModeSolid, zeroFillMode)
	checkGoProjection("fill-mode.arbitrary-positive-raw", "FILL_MODE", "2,12345",
		fmt.Sprintf("%d,%d", graphics.FillMode(2), graphics.FillMode(12345)))
	checkGoProjection("fill-mode.negative-raw", "FILL_MODE", int32(-1), int32(graphics.FillMode(-1)))

	check("media-source-type.complete-raw-table", "MEDIA_SOURCE_TYPE",
		"LocalDevice=0,WindowsMediaConnect=4",
		fmt.Sprintf("LocalDevice=%d,WindowsMediaConnect=%d",
			media.MediaSourceTypeLocalDevice, media.MediaSourceTypeWindowsMediaConnect))
	check("media-source-type.underlying-system-int32", "MEDIA_SOURCE_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(media.MediaSourceTypeLocalDevice).Kind()])
	var zeroMediaSourceType media.MediaSourceType
	checkGoProjection("media-source-type.zero-value-local-device", "MEDIA_SOURCE_TYPE", media.MediaSourceTypeLocalDevice, zeroMediaSourceType)
	checkGoProjection("media-source-type.arbitrary-positive-raw", "MEDIA_SOURCE_TYPE", "5,12345",
		fmt.Sprintf("%d,%d", media.MediaSourceType(5), media.MediaSourceType(12345)))
	checkGoProjection("media-source-type.negative-raw", "MEDIA_SOURCE_TYPE", int32(-1), int32(media.MediaSourceType(-1)))

	check("sound-state.complete-raw-table", "SOUND_STATE",
		"Playing=0,Paused=1,Stopped=2",
		fmt.Sprintf("Playing=%d,Paused=%d,Stopped=%d",
			audio.SoundStatePlaying, audio.SoundStatePaused, audio.SoundStateStopped))
	check("sound-state.underlying-system-int32", "SOUND_STATE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(audio.SoundStatePlaying).Kind()])
	var zeroSoundState audio.SoundState
	checkGoProjection("sound-state.zero-value-playing", "SOUND_STATE", audio.SoundStatePlaying, zeroSoundState)
	checkGoProjection("sound-state.arbitrary-positive-raw", "SOUND_STATE", "3,12345",
		fmt.Sprintf("%d,%d", audio.SoundState(3), audio.SoundState(12345)))
	checkGoProjection("sound-state.negative-raw", "SOUND_STATE", int32(-1), int32(audio.SoundState(-1)))

	check("cull-mode.complete-raw-table", "CULL_MODE",
		"None=0,CullClockwiseFace=1,CullCounterClockwiseFace=2",
		fmt.Sprintf("None=%d,CullClockwiseFace=%d,CullCounterClockwiseFace=%d",
			graphics.CullModeNone, graphics.CullModeCullClockwiseFace, graphics.CullModeCullCounterClockwiseFace))
	check("cull-mode.underlying-system-int32", "CULL_MODE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.CullModeNone).Kind()])
	var zeroCullMode graphics.CullMode
	checkGoProjection("cull-mode.zero-value-none", "CULL_MODE", graphics.CullModeNone, zeroCullMode)
	checkGoProjection("cull-mode.arbitrary-positive-raw", "CULL_MODE", "3,12345",
		fmt.Sprintf("%d,%d", graphics.CullMode(3), graphics.CullMode(12345)))
	checkGoProjection("cull-mode.negative-raw", "CULL_MODE", int32(-1), int32(graphics.CullMode(-1)))

	check("graphics-device-status.complete-raw-table", "GRAPHICS_DEVICE_STATUS",
		"Normal=0,Lost=1,NotReset=2",
		fmt.Sprintf("Normal=%d,Lost=%d,NotReset=%d",
			graphics.GraphicsDeviceStatusNormal, graphics.GraphicsDeviceStatusLost, graphics.GraphicsDeviceStatusNotReset))
	check("graphics-device-status.underlying-system-int32", "GRAPHICS_DEVICE_STATUS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.GraphicsDeviceStatusNormal).Kind()])
	var zeroGraphicsDeviceStatus graphics.GraphicsDeviceStatus
	checkGoProjection("graphics-device-status.zero-value-normal", "GRAPHICS_DEVICE_STATUS", graphics.GraphicsDeviceStatusNormal, zeroGraphicsDeviceStatus)
	checkGoProjection("graphics-device-status.arbitrary-positive-raw", "GRAPHICS_DEVICE_STATUS", "3,12345",
		fmt.Sprintf("%d,%d", graphics.GraphicsDeviceStatus(3), graphics.GraphicsDeviceStatus(12345)))
	checkGoProjection("graphics-device-status.negative-raw", "GRAPHICS_DEVICE_STATUS", int32(-1), int32(graphics.GraphicsDeviceStatus(-1)))

	check("texture-address-mode.complete-raw-table", "TEXTURE_ADDRESS_MODE",
		"Wrap=0,Clamp=1,Mirror=2",
		fmt.Sprintf("Wrap=%d,Clamp=%d,Mirror=%d",
			graphics.TextureAddressModeWrap, graphics.TextureAddressModeClamp, graphics.TextureAddressModeMirror))
	check("texture-address-mode.underlying-system-int32", "TEXTURE_ADDRESS_MODE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.TextureAddressModeWrap).Kind()])
	var zeroTextureAddressMode graphics.TextureAddressMode
	checkGoProjection("texture-address-mode.zero-value-wrap", "TEXTURE_ADDRESS_MODE", graphics.TextureAddressModeWrap, zeroTextureAddressMode)
	checkGoProjection("texture-address-mode.arbitrary-positive-raw", "TEXTURE_ADDRESS_MODE", "3,12345",
		fmt.Sprintf("%d,%d", graphics.TextureAddressMode(3), graphics.TextureAddressMode(12345)))
	checkGoProjection("texture-address-mode.negative-raw", "TEXTURE_ADDRESS_MODE", int32(-1), int32(graphics.TextureAddressMode(-1)))

	check("game-pad-dead-zone.complete-raw-table", "GAME_PAD_DEAD_ZONE",
		"None=0,IndependentAxes=1,Circular=2",
		fmt.Sprintf("None=%d,IndependentAxes=%d,Circular=%d",
			input.GamePadDeadZoneNone, input.GamePadDeadZoneIndependentAxes, input.GamePadDeadZoneCircular))
	check("game-pad-dead-zone.underlying-system-int32", "GAME_PAD_DEAD_ZONE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(input.GamePadDeadZoneNone).Kind()])
	var zeroGamePadDeadZone input.GamePadDeadZone
	checkGoProjection("game-pad-dead-zone.zero-value-none", "GAME_PAD_DEAD_ZONE", input.GamePadDeadZoneNone, zeroGamePadDeadZone)
	checkGoProjection("game-pad-dead-zone.arbitrary-positive-raw", "GAME_PAD_DEAD_ZONE", "3,12345",
		fmt.Sprintf("%d,%d", input.GamePadDeadZone(3), input.GamePadDeadZone(12345)))
	checkGoProjection("game-pad-dead-zone.negative-raw", "GAME_PAD_DEAD_ZONE", int32(-1), int32(input.GamePadDeadZone(-1)))

	check("video-soundtrack-type.complete-raw-table", "VIDEO_SOUNDTRACK_TYPE",
		"Music=0,Dialog=1,MusicAndDialog=2",
		fmt.Sprintf("Music=%d,Dialog=%d,MusicAndDialog=%d",
			media.VideoSoundtrackTypeMusic, media.VideoSoundtrackTypeDialog, media.VideoSoundtrackTypeMusicAndDialog))
	check("video-soundtrack-type.underlying-system-int32", "VIDEO_SOUNDTRACK_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(media.VideoSoundtrackTypeMusic).Kind()])
	var zeroVideoSoundtrackType media.VideoSoundtrackType
	checkGoProjection("video-soundtrack-type.zero-value-music", "VIDEO_SOUNDTRACK_TYPE", media.VideoSoundtrackTypeMusic, zeroVideoSoundtrackType)
	checkGoProjection("video-soundtrack-type.arbitrary-positive-raw", "VIDEO_SOUNDTRACK_TYPE", "3,12345",
		fmt.Sprintf("%d,%d", media.VideoSoundtrackType(3), media.VideoSoundtrackType(12345)))
	checkGoProjection("video-soundtrack-type.negative-raw", "VIDEO_SOUNDTRACK_TYPE", int32(-1), int32(media.VideoSoundtrackType(-1)))

	check("present-interval.complete-raw-table", "PRESENT_INTERVAL",
		"Default=0,One=1,Two=2,Immediate=3",
		fmt.Sprintf("Default=%d,One=%d,Two=%d,Immediate=%d",
			graphics.PresentIntervalDefault, graphics.PresentIntervalOne, graphics.PresentIntervalTwo, graphics.PresentIntervalImmediate))
	check("present-interval.underlying-system-int32", "PRESENT_INTERVAL", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.PresentIntervalDefault).Kind()])
	var zeroPresentInterval graphics.PresentInterval
	checkGoProjection("present-interval.zero-value-default", "PRESENT_INTERVAL", graphics.PresentIntervalDefault, zeroPresentInterval)
	checkGoProjection("present-interval.arbitrary-positive-raw", "PRESENT_INTERVAL", "4,12345",
		fmt.Sprintf("%d,%d", graphics.PresentInterval(4), graphics.PresentInterval(12345)))
	checkGoProjection("present-interval.negative-raw", "PRESENT_INTERVAL", int32(-1), int32(graphics.PresentInterval(-1)))

	check("primitive-type.complete-raw-table", "PRIMITIVE_TYPE",
		"TriangleList=0,TriangleStrip=1,LineList=2,LineStrip=3",
		fmt.Sprintf("TriangleList=%d,TriangleStrip=%d,LineList=%d,LineStrip=%d",
			graphics.PrimitiveTypeTriangleList, graphics.PrimitiveTypeTriangleStrip, graphics.PrimitiveTypeLineList, graphics.PrimitiveTypeLineStrip))
	check("primitive-type.underlying-system-int32", "PRIMITIVE_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.PrimitiveTypeTriangleList).Kind()])
	var zeroPrimitiveType graphics.PrimitiveType
	checkGoProjection("primitive-type.zero-value-triangle-list", "PRIMITIVE_TYPE", graphics.PrimitiveTypeTriangleList, zeroPrimitiveType)
	checkGoProjection("primitive-type.arbitrary-positive-raw", "PRIMITIVE_TYPE", "4,12345",
		fmt.Sprintf("%d,%d", graphics.PrimitiveType(4), graphics.PrimitiveType(12345)))
	checkGoProjection("primitive-type.negative-raw", "PRIMITIVE_TYPE", int32(-1), int32(graphics.PrimitiveType(-1)))

	check("touch-location-state.complete-raw-table", "TOUCH_LOCATION_STATE",
		"Invalid=0,Released=1,Pressed=2,Moved=3",
		fmt.Sprintf("Invalid=%d,Released=%d,Pressed=%d,Moved=%d",
			touch.TouchLocationStateInvalid, touch.TouchLocationStateReleased, touch.TouchLocationStatePressed, touch.TouchLocationStateMoved))
	check("touch-location-state.underlying-system-int32", "TOUCH_LOCATION_STATE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(touch.TouchLocationStateInvalid).Kind()])
	var zeroTouchLocationState touch.TouchLocationState
	checkGoProjection("touch-location-state.zero-value-invalid", "TOUCH_LOCATION_STATE", touch.TouchLocationStateInvalid, zeroTouchLocationState)
	checkGoProjection("touch-location-state.arbitrary-positive-raw", "TOUCH_LOCATION_STATE", "4,12345",
		fmt.Sprintf("%d,%d", touch.TouchLocationState(4), touch.TouchLocationState(12345)))
	checkGoProjection("touch-location-state.negative-raw", "TOUCH_LOCATION_STATE", int32(-1), int32(touch.TouchLocationState(-1)))

	check("blend-function.complete-raw-table", "BLEND_FUNCTION",
		"Add=0,Subtract=1,ReverseSubtract=2,Min=3,Max=4",
		fmt.Sprintf("Add=%d,Subtract=%d,ReverseSubtract=%d,Min=%d,Max=%d",
			graphics.BlendFunctionAdd, graphics.BlendFunctionSubtract, graphics.BlendFunctionReverseSubtract, graphics.BlendFunctionMin, graphics.BlendFunctionMax))
	check("blend-function.underlying-system-int32", "BLEND_FUNCTION", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.BlendFunctionAdd).Kind()])
	var zeroBlendFunction graphics.BlendFunction
	checkGoProjection("blend-function.zero-value-add", "BLEND_FUNCTION", graphics.BlendFunctionAdd, zeroBlendFunction)
	checkGoProjection("blend-function.arbitrary-positive-raw", "BLEND_FUNCTION", "5,12345",
		fmt.Sprintf("%d,%d", graphics.BlendFunction(5), graphics.BlendFunction(12345)))
	checkGoProjection("blend-function.negative-raw", "BLEND_FUNCTION", int32(-1), int32(graphics.BlendFunction(-1)))

	// Foundation 15 pure-managed batch B: the last five safe leaf enums. The
	// same provenance split applies — pinned raw values are XNA metadata, the
	// zero-value and arbitrary-raw facts are Go language projection, and none of
	// these enums implies render state, sampling, blending, or game pad support.

	check("color-write-channels.complete-raw-table", "COLOR_WRITE_CHANNELS",
		"None=0,Red=1,Green=2,Blue=4,Alpha=8,All=15",
		fmt.Sprintf("None=%d,Red=%d,Green=%d,Blue=%d,Alpha=%d,All=%d",
			graphics.ColorWriteChannelsNone, graphics.ColorWriteChannelsRed, graphics.ColorWriteChannelsGreen, graphics.ColorWriteChannelsBlue, graphics.ColorWriteChannelsAlpha, graphics.ColorWriteChannelsAll))
	check("color-write-channels.underlying-system-int32", "COLOR_WRITE_CHANNELS", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.ColorWriteChannelsNone).Kind()])
	colorWriteChannelsUnion, colorWriteChannelsSingleBits := enumBitStructure([]int32{int32(graphics.ColorWriteChannelsNone), int32(graphics.ColorWriteChannelsRed), int32(graphics.ColorWriteChannelsGreen), int32(graphics.ColorWriteChannelsBlue), int32(graphics.ColorWriteChannelsAlpha)})
	check("color-write-channels.flags-union", "COLOR_WRITE_CHANNELS", int32(15), colorWriteChannelsUnion)
	check("color-write-channels.flags-disjoint-single-bits", "COLOR_WRITE_CHANNELS", true, colorWriteChannelsSingleBits)
	// All=15 is a pinned aggregate literal, not an invented convenience.
	check("color-write-channels.pinned-all-equals-channel-union", "COLOR_WRITE_CHANNELS", graphics.ColorWriteChannelsAll, graphics.ColorWriteChannelsRed|graphics.ColorWriteChannelsGreen|graphics.ColorWriteChannelsBlue|graphics.ColorWriteChannelsAlpha)
	var zeroColorWriteChannels graphics.ColorWriteChannels
	checkGoProjection("color-write-channels.zero-value-none", "COLOR_WRITE_CHANNELS", graphics.ColorWriteChannelsNone, zeroColorWriteChannels)
	checkGoProjection("color-write-channels.arbitrary-positive-raw", "COLOR_WRITE_CHANNELS", "16,12345",
		fmt.Sprintf("%d,%d", graphics.ColorWriteChannels(16), graphics.ColorWriteChannels(12345)))
	checkGoProjection("color-write-channels.negative-raw", "COLOR_WRITE_CHANNELS", int32(-1), int32(graphics.ColorWriteChannels(-1)))

	check("stencil-operation.complete-raw-table", "STENCIL_OPERATION",
		"Keep=0,Zero=1,Replace=2,Increment=3,Decrement=4,IncrementSaturation=5,DecrementSaturation=6,Invert=7",
		fmt.Sprintf("Keep=%d,Zero=%d,Replace=%d,Increment=%d,Decrement=%d,IncrementSaturation=%d,DecrementSaturation=%d,Invert=%d",
			graphics.StencilOperationKeep, graphics.StencilOperationZero, graphics.StencilOperationReplace, graphics.StencilOperationIncrement, graphics.StencilOperationDecrement, graphics.StencilOperationIncrementSaturation, graphics.StencilOperationDecrementSaturation, graphics.StencilOperationInvert))
	check("stencil-operation.underlying-system-int32", "STENCIL_OPERATION", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.StencilOperationKeep).Kind()])
	var zeroStencilOperation graphics.StencilOperation
	checkGoProjection("stencil-operation.zero-value-keep", "STENCIL_OPERATION", graphics.StencilOperationKeep, zeroStencilOperation)
	checkGoProjection("stencil-operation.arbitrary-positive-raw", "STENCIL_OPERATION", "8,12345",
		fmt.Sprintf("%d,%d", graphics.StencilOperation(8), graphics.StencilOperation(12345)))
	checkGoProjection("stencil-operation.negative-raw", "STENCIL_OPERATION", int32(-1), int32(graphics.StencilOperation(-1)))

	check("texture-filter.complete-raw-table", "TEXTURE_FILTER",
		"Linear=0,Point=1,Anisotropic=2,LinearMipPoint=3,PointMipLinear=4,MinLinearMagPointMipLinear=5,MinLinearMagPointMipPoint=6,MinPointMagLinearMipLinear=7,MinPointMagLinearMipPoint=8",
		fmt.Sprintf("Linear=%d,Point=%d,Anisotropic=%d,LinearMipPoint=%d,PointMipLinear=%d,MinLinearMagPointMipLinear=%d,MinLinearMagPointMipPoint=%d,MinPointMagLinearMipLinear=%d,MinPointMagLinearMipPoint=%d",
			graphics.TextureFilterLinear, graphics.TextureFilterPoint, graphics.TextureFilterAnisotropic, graphics.TextureFilterLinearMipPoint, graphics.TextureFilterPointMipLinear, graphics.TextureFilterMinLinearMagPointMipLinear, graphics.TextureFilterMinLinearMagPointMipPoint, graphics.TextureFilterMinPointMagLinearMipLinear, graphics.TextureFilterMinPointMagLinearMipPoint))
	check("texture-filter.underlying-system-int32", "TEXTURE_FILTER", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.TextureFilterLinear).Kind()])
	var zeroTextureFilter graphics.TextureFilter
	checkGoProjection("texture-filter.zero-value-linear", "TEXTURE_FILTER", graphics.TextureFilterLinear, zeroTextureFilter)
	checkGoProjection("texture-filter.arbitrary-positive-raw", "TEXTURE_FILTER", "9,12345",
		fmt.Sprintf("%d,%d", graphics.TextureFilter(9), graphics.TextureFilter(12345)))
	checkGoProjection("texture-filter.negative-raw", "TEXTURE_FILTER", int32(-1), int32(graphics.TextureFilter(-1)))

	check("game-pad-type.complete-raw-table", "GAME_PAD_TYPE",
		"Unknown=0,GamePad=1,Wheel=2,ArcadeStick=3,FlightStick=4,DancePad=5,Guitar=6,AlternateGuitar=7,DrumKit=8,BigButtonPad=768",
		fmt.Sprintf("Unknown=%d,GamePad=%d,Wheel=%d,ArcadeStick=%d,FlightStick=%d,DancePad=%d,Guitar=%d,AlternateGuitar=%d,DrumKit=%d,BigButtonPad=%d",
			input.GamePadTypeUnknown, input.GamePadTypeGamePad, input.GamePadTypeWheel, input.GamePadTypeArcadeStick, input.GamePadTypeFlightStick, input.GamePadTypeDancePad, input.GamePadTypeGuitar, input.GamePadTypeAlternateGuitar, input.GamePadTypeDrumKit, input.GamePadTypeBigButtonPad))
	check("game-pad-type.underlying-system-int32", "GAME_PAD_TYPE", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(input.GamePadTypeUnknown).Kind()])
	var zeroGamePadType input.GamePadType
	checkGoProjection("game-pad-type.zero-value-unknown", "GAME_PAD_TYPE", input.GamePadTypeUnknown, zeroGamePadType)
	checkGoProjection("game-pad-type.arbitrary-positive-raw", "GAME_PAD_TYPE", "769,12345",
		fmt.Sprintf("%d,%d", input.GamePadType(769), input.GamePadType(12345)))
	checkGoProjection("game-pad-type.negative-raw", "GAME_PAD_TYPE", int32(-1), int32(input.GamePadType(-1)))

	check("blend.complete-raw-table", "BLEND",
		"One=0,Zero=1,SourceColor=2,InverseSourceColor=3,SourceAlpha=4,InverseSourceAlpha=5,DestinationColor=6,InverseDestinationColor=7,DestinationAlpha=8,InverseDestinationAlpha=9,BlendFactor=10,InverseBlendFactor=11,SourceAlphaSaturation=12",
		fmt.Sprintf("One=%d,Zero=%d,SourceColor=%d,InverseSourceColor=%d,SourceAlpha=%d,InverseSourceAlpha=%d,DestinationColor=%d,InverseDestinationColor=%d,DestinationAlpha=%d,InverseDestinationAlpha=%d,BlendFactor=%d,InverseBlendFactor=%d,SourceAlphaSaturation=%d",
			graphics.BlendOne, graphics.BlendZero, graphics.BlendSourceColor, graphics.BlendInverseSourceColor, graphics.BlendSourceAlpha, graphics.BlendInverseSourceAlpha, graphics.BlendDestinationColor, graphics.BlendInverseDestinationColor, graphics.BlendDestinationAlpha, graphics.BlendInverseDestinationAlpha, graphics.BlendBlendFactor, graphics.BlendInverseBlendFactor, graphics.BlendSourceAlphaSaturation))
	check("blend.underlying-system-int32", "BLEND", "System.Int32",
		map[reflect.Kind]string{reflect.Int32: "System.Int32"}[reflect.TypeOf(graphics.BlendOne).Kind()])
	var zeroBlend graphics.Blend
	checkGoProjection("blend.zero-value-one", "BLEND", graphics.BlendOne, zeroBlend)
	checkGoProjection("blend.arbitrary-positive-raw", "BLEND", "13,12345",
		fmt.Sprintf("%d,%d", graphics.Blend(13), graphics.Blend(12345)))
	checkGoProjection("blend.negative-raw", "BLEND", int32(-1), int32(graphics.Blend(-1)))

	// Foundation 15 value-struct cluster. Every constant below was read from
	// the retained XNA assemblies' IL, not inferred. The clamping rules, hash
	// algorithms, and ToString layouts are PURE_XNA_DERIVED; Go copy semantics
	// and the permissive raw ButtonState domain are GO_LANGUAGE_PROJECTION.
	// Completing these values claims no game pad or mouse capability: CNA-Go
	// exposes no GamePad, GamePadState, or Mouse type and no input backend.

	clampedSticks := input.NewGamePadThumbSticks(
		framework.Vector2{X: 0.5, Y: -2},
		framework.Vector2{X: 3, Y: float32(math.NaN())},
	)
	check("game-pad-thumb-sticks.constructor-clamp", "GAME_PAD_THUMB_STICKS",
		"{X:0.5 Y:-1},{X:1 Y:1}",
		fmt.Sprintf("%s,%s", clampedSticks.Left().ToString(), clampedSticks.Right().ToString()))
	check("game-pad-thumb-sticks.nan-clamps-to-one", "GAME_PAD_THUMB_STICKS", float32(1), clampedSticks.Right().Y)
	check("game-pad-thumb-sticks.smart-hash", "GAME_PAD_THUMB_STICKS", int32(-2139095040), clampedSticks.GetHashCode())
	var zeroSticks input.GamePadThumbSticks
	check("game-pad-thumb-sticks.zero-hash-substitution", "GAME_PAD_THUMB_STICKS", int32(math.MaxInt32), zeroSticks.GetHashCode())
	check("game-pad-thumb-sticks.string", "GAME_PAD_THUMB_STICKS", "{Left:{X:0.5 Y:-1} Right:{X:1 Y:1}}", clampedSticks.ToString())

	clampedTriggers := input.NewGamePadTriggers(1.5, -0.25)
	check("game-pad-triggers.constructor-clamp", "GAME_PAD_TRIGGERS", "1,0",
		fmt.Sprintf("%g,%g", clampedTriggers.Left(), clampedTriggers.Right()))
	nanTriggers := input.NewGamePadTriggers(float32(math.NaN()), 0.5)
	check("game-pad-triggers.nan-propagates", "GAME_PAD_TRIGGERS", true, math.IsNaN(float64(nanTriggers.Left())))
	negativeZeroTriggers := input.NewGamePadTriggers(float32(math.Copysign(0, -1)), 0)
	check("game-pad-triggers.negative-zero-becomes-positive", "GAME_PAD_TRIGGERS", false, math.Signbit(float64(negativeZeroTriggers.Left())))
	quarterTriggers := input.NewGamePadTriggers(0.25, 0.75)
	check("game-pad-triggers.smart-hash", "GAME_PAD_TRIGGERS", int32(29360128), quarterTriggers.GetHashCode())
	check("game-pad-triggers.string", "GAME_PAD_TRIGGERS", "{Left:0.25 Right:0.75}", quarterTriggers.ToString())
	checkGoProjection("game-pad-triggers.nan-not-self-equal", "GAME_PAD_TRIGGERS", false,
		input.GamePadTriggersOperatorEqualityByGamePadTriggersAndGamePadTriggers(nanTriggers, nanTriggers))

	dpad := input.NewGamePadDPad(input.ButtonStatePressed, input.ButtonStateReleased, input.ButtonStatePressed, input.ButtonStateReleased)
	check("game-pad-dpad.constructor-parameter-order", "GAME_PAD_DPAD", "1,0,1,0",
		fmt.Sprintf("%d,%d,%d,%d", dpad.Up(), dpad.Down(), dpad.Left(), dpad.Right()))
	check("game-pad-dpad.string", "GAME_PAD_DPAD", "{DPad:Up Left}", dpad.ToString())
	var noDPad input.GamePadDPad
	check("game-pad-dpad.string-none", "GAME_PAD_DPAD", "{DPad:None}", noDPad.ToString())
	allDPad := input.NewGamePadDPad(input.ButtonStatePressed, input.ButtonStatePressed, input.ButtonStatePressed, input.ButtonStatePressed)
	check("game-pad-dpad.string-order", "GAME_PAD_DPAD", "{DPad:Up Down Left Right}", allDPad.ToString())
	singleDPad := input.NewGamePadDPad(input.ButtonStatePressed, input.ButtonStateReleased, input.ButtonStateReleased, input.ButtonStateReleased)
	check("game-pad-dpad.smart-hash", "GAME_PAD_DPAD", int32(1), singleDPad.GetHashCode())
	pairDPad := input.NewGamePadDPad(input.ButtonStatePressed, input.ButtonStateReleased, input.ButtonStateReleased, input.ButtonStatePressed)
	check("game-pad-dpad.compatible-hash-collision", "GAME_PAD_DPAD",
		fmt.Sprintf("%d,%d", math.MaxInt32, math.MaxInt32),
		fmt.Sprintf("%d,%d", pairDPad.GetHashCode(), noDPad.GetHashCode()))
	arbitraryDPad := input.NewGamePadDPad(input.ButtonState(12345), input.ButtonStateReleased, input.ButtonStateReleased, input.ButtonStateReleased)
	checkGoProjection("game-pad-dpad.arbitrary-raw-is-not-pressed", "GAME_PAD_DPAD", "{DPad:None}", arbitraryDPad.ToString())

	buttons := input.NewGamePadButtons(input.ButtonsA | input.ButtonsStart | input.ButtonsBigButton)
	check("game-pad-buttons.mask-derivation", "GAME_PAD_BUTTONS", "1,1,1,0,0",
		fmt.Sprintf("%d,%d,%d,%d,%d", buttons.A(), buttons.Start(), buttons.BigButton(), buttons.B(), buttons.Back()))
	strayButtons := input.NewGamePadButtons(input.ButtonsLeftThumbstickUp | input.ButtonsRightTrigger)
	check("game-pad-buttons.thumbstick-literals-have-no-field", "GAME_PAD_BUTTONS", "{Buttons:None}", strayButtons.ToString())
	orderedButtons := input.NewGamePadButtons(input.ButtonsBack | input.ButtonsA | input.ButtonsLeftStick | input.ButtonsY)
	check("game-pad-buttons.string-order", "GAME_PAD_BUTTONS", "{Buttons:A Y LeftStick Back}", orderedButtons.ToString())
	check("game-pad-buttons.smart-hash", "GAME_PAD_BUTTONS", int32(1), input.NewGamePadButtons(input.ButtonsA).GetHashCode())
	check("game-pad-buttons.compatible-hash-collision", "GAME_PAD_BUTTONS", int32(math.MaxInt32),
		input.NewGamePadButtons(input.ButtonsA|input.ButtonsStart).GetHashCode())

	mouse := input.NewMouseState(1, 2, 3,
		input.ButtonStatePressed, input.ButtonStateReleased, input.ButtonStatePressed,
		input.ButtonStateReleased, input.ButtonStatePressed)
	check("mouse-state.constructor-parameter-order", "MOUSE_STATE", "1,2,3,1,0,1,0,1",
		fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d", mouse.X(), mouse.Y(), mouse.ScrollWheelValue(),
			mouse.LeftButton(), mouse.MiddleButton(), mouse.RightButton(), mouse.XButton1(), mouse.XButton2()))
	// MouseState does not use Helpers.SmartGetHashCode, so it has no
	// Int32.MaxValue substitution and the zero value hashes to zero.
	var zeroMouse input.MouseState
	check("mouse-state.direct-xor-hash", "MOUSE_STATE", "1,0",
		fmt.Sprintf("%d,%d", mouse.GetHashCode(), zeroMouse.GetHashCode()))
	check("mouse-state.string", "MOUSE_STATE", "{X:1 Y:2 Buttons:Left Right XButton2 Wheel:3}", mouse.ToString())
	check("mouse-state.string-none", "MOUSE_STATE", "{X:0 Y:0 Buttons:None Wheel:0}", zeroMouse.ToString())
	movedMouse := mouse
	checkGoProjection("mouse-state.go-copy-semantics", "MOUSE_STATE", true,
		input.MouseStateOperatorEqualityByMouseStateAndMouseState(mouse, movedMouse))

	// Foundation 15 touch value cluster, read from the retained
	// Microsoft.Xna.Framework.Input.Touch assembly. Completing these values
	// claims no touch capability: CNA-Go exposes no TouchPanel and reads no
	// device.
	gestureTimestamp := framework.TimeSpanFromTicks(1234567)
	gesture := touch.NewGestureSample(
		touch.GestureTypeFreeDrag, gestureTimestamp,
		framework.Vector2{X: 1, Y: 2}, framework.Vector2{X: 3, Y: 4},
		framework.Vector2{X: 5, Y: 6}, framework.Vector2{X: 7, Y: 8},
	)
	check("gesture-sample.stores-components-unchanged", "GESTURE_SAMPLE",
		"32,1234567,{X:1 Y:2},{X:3 Y:4},{X:5 Y:6},{X:7 Y:8}",
		fmt.Sprintf("%d,%d,%s,%s,%s,%s", gesture.GestureType(), gesture.Timestamp().Ticks(),
			gesture.Position().ToString(), gesture.Position2().ToString(),
			gesture.Delta().ToString(), gesture.Delta2().ToString()))

	singleTouch := touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		7, touch.TouchLocationStateMoved, framework.Vector2{X: 1.5, Y: -2.5})
	check("touch-location.single-sample-constructor", "TOUCH_LOCATION", "7,3,{X:1.5 Y:-2.5}",
		fmt.Sprintf("%d,%d,%s", singleTouch.Id(), singleTouch.State(), singleTouch.Position().ToString()))
	noPrevious, emptyPrevious := singleTouch.TryGetPreviousLocation()
	check("touch-location.absent-previous-sample", "TOUCH_LOCATION", "false,-1,0,{X:0 Y:0}",
		fmt.Sprintf("%t,%d,%d,%s", noPrevious, emptyPrevious.Id(), emptyPrevious.State(), emptyPrevious.Position().ToString()))
	pairedTouch := touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2AndTouchLocationStateAndVector2(
		9, touch.TouchLocationStateMoved, framework.Vector2{X: 10, Y: 20},
		touch.TouchLocationStatePressed, framework.Vector2{X: 30, Y: 40})
	hasPrevious, promoted := pairedTouch.TryGetPreviousLocation()
	nestedPrevious, _ := promoted.TryGetPreviousLocation()
	check("touch-location.previous-sample-promotion", "TOUCH_LOCATION", "true,9,2,{X:30 Y:40},false",
		fmt.Sprintf("%t,%d,%d,%s,%t", hasPrevious, promoted.Id(), promoted.State(),
			promoted.Position().ToString(), nestedPrevious))

	// A genuine XNA asymmetry: the equality operator compares all seven
	// fields while Equals(TouchLocation) ignores both state fields.
	touchPosition := framework.Vector2{X: 1, Y: 2}
	pressedTouch := touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(3, touch.TouchLocationStatePressed, touchPosition)
	movedTouch := touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(3, touch.TouchLocationStateMoved, touchPosition)
	check("touch-location.equals-ignores-state", "TOUCH_LOCATION", true, pressedTouch.EqualsByTouchLocation(movedTouch))
	check("touch-location.operator-observes-state", "TOUCH_LOCATION", false,
		touch.TouchLocationOperatorEqualityByTouchLocationAndTouchLocation(pressedTouch, movedTouch))
	check("touch-location.hash", "TOUCH_LOCATION",
		int32(5)+int32(math.Float32bits(1))+int32(math.Float32bits(2)),
		touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(5, touch.TouchLocationStatePressed, framework.Vector2{X: 1, Y: 2}).GetHashCode())
	check("touch-location.signed-zero-hash-canonicalization", "TOUCH_LOCATION", int32(11),
		touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(11, touch.TouchLocationStatePressed,
			framework.Vector2{X: 0, Y: float32(math.Copysign(0, -1))}).GetHashCode())
	check("touch-location.string-omits-state", "TOUCH_LOCATION", "{Position:{X:1 Y:2}}", pressedTouch.ToString())

	// Foundation 16: GamePadState, unlocked by the Foundation-15 game pad
	// values. IsButtonDown reproduces the reference XInput packing and
	// IndependentAxes dead-zone arithmetic as pure managed code. Completing it
	// claims no game pad capability: CNA-Go exposes no GamePad type, polls
	// nothing, and reads no device. IsConnected reports what the constructor
	// stored.
	padState := input.NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: 0.5}, framework.Vector2{Y: 0.5}, 0.5, 0.1,
		[]input.Buttons{input.ButtonsA, input.ButtonsDPadUp},
	)
	check("game-pad-state.constructor-defaults", "GAME_PAD_STATE", "true,0",
		fmt.Sprintf("%t,%d", padState.IsConnected(), padState.PacketNumber()))
	check("game-pad-state.buttons-slice-combined", "GAME_PAD_STATE", "1,0,1,0",
		fmt.Sprintf("%d,%d,%d,%d", padState.Buttons().A(), padState.Buttons().B(),
			padState.DPad().Up(), padState.DPad().Down()))
	check("game-pad-state.normal-button-query", "GAME_PAD_STATE", "true,false,true",
		fmt.Sprintf("%t,%t,%t", padState.IsButtonDown(input.ButtonsA),
			padState.IsButtonDown(input.ButtonsB), padState.IsButtonUp(input.ButtonsB)))
	check("game-pad-state.all-requested-bits-required", "GAME_PAD_STATE", "true,false",
		fmt.Sprintf("%t,%t", padState.IsButtonDown(input.ButtonsA|input.ButtonsDPadUp),
			padState.IsButtonDown(input.ButtonsA|input.ButtonsB)))
	// A half-deflected stick quantizes to 16383, clearing the 7849 left-stick
	// dead zone; a tenth-deflected stick quantizes to 3276 and does not.
	check("game-pad-state.thumbstick-outside-dead-zone", "GAME_PAD_STATE", "true,false",
		fmt.Sprintf("%t,%t", padState.IsButtonDown(input.ButtonsLeftThumbstickRight),
			padState.IsButtonDown(input.ButtonsLeftThumbstickLeft)))
	nearState := input.NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{X: 0.1}, framework.Vector2{}, 0, 0, nil)
	check("game-pad-state.thumbstick-inside-dead-zone", "GAME_PAD_STATE", false,
		nearState.IsButtonDown(input.ButtonsLeftThumbstickRight))
	// A half-pulled trigger quantizes to 127, clearing the dead zone of 30; a
	// tenth-pulled trigger quantizes to 25 and does not.
	check("game-pad-state.trigger-dead-zone", "GAME_PAD_STATE", "true,false",
		fmt.Sprintf("%t,%t", padState.IsButtonDown(input.ButtonsLeftTrigger),
			padState.IsButtonDown(input.ButtonsRightTrigger)))
	// Asking about no button at all reports true: the empty mask is trivially
	// contained. This is reference behavior, not a Go artifact.
	check("game-pad-state.empty-mask-is-down", "GAME_PAD_STATE", true, padState.IsButtonDown(0))
	var zeroPadState input.GamePadState
	check("game-pad-state.hash-and-string", "GAME_PAD_STATE", "0,{IsConnected:False}",
		fmt.Sprintf("%d,%s", zeroPadState.GetHashCode(), zeroPadState.ToString()))
	defaultPadState := input.NewGamePadStateByVector2AndVector2AndSingleAndSingleAndSliceOfButtons(
		framework.Vector2{}, framework.Vector2{}, 0, 0, nil)
	check("game-pad-state.connected-hash-and-string", "GAME_PAD_STATE", "1,{IsConnected:True}",
		fmt.Sprintf("%d,%s", defaultPadState.GetHashCode(), defaultPadState.ToString()))
	checkGoProjection("game-pad-state.zero-value-is-disconnected", "GAME_PAD_STATE", false, zeroPadState.IsConnected())

	check("math.clamp.low", "MathHelper", bits(0), bits(framework.MathHelperClamp(-2, 0, 1)))
	check("math.clamp.inverted", "MathHelper", "0x40000000", bits(framework.MathHelperClamp(0, 2, 1)))
	check("math.lerp", "MathHelper", bits(4), bits(framework.MathHelperLerp(2, 10, 0.25)))
	check("math.barycentric", "MathHelper", bits(3.5), bits(framework.MathHelperBarycentric(1, 3, 5, 0.25, 0.5)))
	check("math.catmullrom", "MathHelper", "0xc1218313", bits(framework.MathHelperCatmullRom(-10, -10, -10, -7, 0.3)))
	check("math.hermite", "MathHelper", "0xc1351eba", bits(framework.MathHelperHermite(-10, -10, -10, -10, 1.1)))
	check("math.wrapangle.large", "MathHelper", "0xbfc2e06c", bits(framework.MathHelperWrapAngle(123456.789)))
	check("math.to_degrees", "MathHelper", bits(180), bits(framework.MathHelperToDegrees(framework.MathHelperPi)))
	check("math.negative_zero.distance", "MathHelper", "0x00000000", bits(framework.MathHelperDistance(float32(math.Copysign(0, -1)), 0)))
	check("math.nan.hermite", "MathHelper", true, math.IsNaN(float64(framework.MathHelperHermite(1, float32(math.Inf(1)), 2, 0, 0))))
	check("math.overflow.distance", "MathHelper", true, math.IsInf(float64(framework.MathHelperDistance(math.MaxFloat32, -math.MaxFloat32)), 1))
	check("math.subnormal.lerp", "MathHelper", bits(math.SmallestNonzeroFloat32), bits(framework.MathHelperLerp(0, math.SmallestNonzeroFloat32, 1)))
	check("math.underflow.lerp", "MathHelper", "0x00000000", bits(framework.MathHelperLerp(0, math.SmallestNonzeroFloat32, 0.5)))

	point := framework.NewPoint(1, 2)
	check("point.hash", "Point", int32(3), point.GetHashCode())
	check("point.string", "Point", "{X:1 Y:2}", point.ToString())
	check("point.value-equality", "Point", true, point.EqualsByPoint(framework.NewPoint(1, 2)))
	zero1, zero2 := framework.PointZero(), framework.PointZero()
	zero1.X = 9
	check("point.zero.fresh-value", "Point", int32(0), zero2.X)

	rectangle := framework.NewRectangle(0, 0, 10, 20)
	check("rectangle.contains.inclusive-min", "Rectangle", true, rectangle.ContainsByInt32AndInt32(0, 0))
	check("rectangle.contains.exclusive-max", "Rectangle", false, rectangle.ContainsByInt32AndInt32(10, 20))
	intersection := framework.RectangleIntersectByRectangleAndRectangle(framework.NewRectangle(0, 0, 10, 10), framework.NewRectangle(5, -5, 10, 10))
	check("rectangle.intersect", "Rectangle", "{X:5 Y:0 Width:5 Height:5}", intersection.ToString())
	union := framework.RectangleUnionByRectangleAndRectangle(framework.NewRectangle(0, 0, 10, 10), framework.NewRectangle(5, -5, 10, 10))
	check("rectangle.union", "Rectangle", "{X:0 Y:-5 Width:15 Height:15}", union.ToString())
	mutable := framework.NewRectangle(2, 3, 4, 5)
	mutable.Inflate(1, 2)
	check("rectangle.inflate", "Rectangle", "{X:1 Y:1 Width:6 Height:9}", mutable.ToString())
	mutable.OffsetByInt32AndInt32(-1, 2)
	check("rectangle.offset", "Rectangle", "{X:0 Y:3 Width:6 Height:9}", mutable.ToString())
	check("rectangle.hash", "Rectangle", int32(10), framework.NewRectangle(1, 2, 3, 4).GetHashCode())

	total := framework.TimeSpanFromTicks(math.MaxInt64)
	elapsed := framework.TimeSpanFromTicks(166667)
	gameTime := framework.NewGameTimeByTimeSpanAndTimeSpanAndBoolean(total, elapsed, true)
	check("gametime.total-ticks", "GameTime", int64(math.MaxInt64), gameTime.TotalGameTime().Ticks())
	check("gametime.elapsed-ticks", "GameTime", int64(166667), gameTime.ElapsedGameTime().Ticks())
	check("gametime.slow", "GameTime", true, gameTime.IsRunningSlowly())

	vector2Zero := framework.Vector2NormalizeByVector2(framework.Vector2Zero())
	check("vector2.normalize.zero", "VECTOR2", "0xffc00000,0xffc00000", floatBits(vector2Zero.X, vector2Zero.Y))
	check("vector2.divide.reciprocal-first", "VECTOR2", "0x3edb6db8", bits(framework.Vector2DivideByVector2AndSingle(framework.NewVector2BySingle(3), 7).X))
	vector2Overlap := []framework.Vector2{{X: 1}, {X: 2}, {X: 3}}
	translation := framework.MatrixCreateTranslationBySingleAndSingleAndSingle(10, 0, 0)
	framework.Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(vector2Overlap, 0, &translation, vector2Overlap, 1, 2)
	check("vector2.transform.forward-overlap", "VECTOR2", "0x41300000,0x41a80000", floatBits(vector2Overlap[1].X, vector2Overlap[2].X))
	check("vector2.nan.equality", "VECTOR2", false, framework.Vector2{X: float32(math.NaN())}.EqualsByVector2(framework.Vector2{X: float32(math.NaN())}))

	vector3Zero := framework.Vector3NormalizeByVector3(framework.Vector3Zero())
	check("vector3.normalize.zero", "VECTOR3", "0xffc00000,0xffc00000,0xffc00000", floatBits(vector3Zero.X, vector3Zero.Y, vector3Zero.Z))
	check("vector3.forward", "VECTOR3", "0x00000000,0x00000000,0xbf800000", floatBits(framework.Vector3Forward().X, framework.Vector3Forward().Y, framework.Vector3Forward().Z))
	cross := framework.Vector3CrossByVector3AndVector3(framework.Vector3Right(), framework.Vector3Up())
	check("vector3.cross.handedness", "VECTOR3", "0x00000000,0x00000000,0x3f800000", floatBits(cross.X, cross.Y, cross.Z))
	check("vector3.divide.reciprocal-first", "VECTOR3", "0x40155556", bits(framework.Vector3DivideByVector3AndSingle(framework.NewVector3BySingle(7), 3).X))
	check("vector3.hash", "VECTOR3", int32(-1077936128), (framework.Vector3{X: 1, Y: 2, Z: 3}).GetHashCode())
	xnaNaN := math.Float32frombits(0xffc00000)
	vector3MinNaN := framework.Vector3MinByVector3AndVector3(framework.Vector3{X: xnaNaN, Y: 1, Z: xnaNaN}, framework.Vector3{X: 7, Y: xnaNaN, Z: xnaNaN})
	check("vector3.min.nan-order", "VECTOR3", "0x40e00000,0xffc00000,0xffc00000", floatBits(vector3MinNaN.X, vector3MinNaN.Y, vector3MinNaN.Z))
	vector3ClampReversed := framework.Vector3ClampByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.NewVector3BySingle(2), framework.NewVector3BySingle(1))
	check("vector3.clamp.reversed", "VECTOR3", "0x40000000,0x40000000,0x40000000", floatBits(vector3ClampReversed.X, vector3ClampReversed.Y, vector3ClampReversed.Z))

	vector4Zero := framework.Vector4NormalizeByVector4(framework.Vector4Zero())
	check("vector4.normalize.zero", "VECTOR4", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(vector4Zero.X, vector4Zero.Y, vector4Zero.Z, vector4Zero.W))
	vector4Negated := framework.Vector4OperatorUnaryNegationByVector4(framework.Vector4Zero())
	check("vector4.negate.signed-zero", "VECTOR4", "0x80000000,0x80000000,0x80000000,0x80000000", floatBits(vector4Negated.X, vector4Negated.Y, vector4Negated.Z, vector4Negated.W))
	check("vector4.divide.reciprocal-first", "VECTOR4", "0x458099ca", bits(framework.Vector4DivideByVector4AndSingle(framework.NewVector4BySingle(12345.67), 3).X))

	quaternionZero := framework.QuaternionInverseByQuaternion(framework.Quaternion{})
	check("quaternion.inverse.zero", "QUATERNION", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(quaternionZero.X, quaternionZero.Y, quaternionZero.Z, quaternionZero.W))
	qa := framework.Quaternion{X: 45889.05859375, Y: -42412.4453125, Z: 96034.96875, W: -76386.84375}
	qb := framework.Quaternion{X: -16375.435546875, Y: 51428.1875, Z: -69603.09375, W: -2207.3798828125}
	quaternionProduct := framework.QuaternionMultiplyByQuaternionAndQuaternion(qa, qb)
	check("quaternion.multiply.order", "QUATERNION", "0xce47a05e,0xcf03edf7,0x4fc9c4dd,0x5011d115", floatBits(quaternionProduct.X, quaternionProduct.Y, quaternionProduct.Z, quaternionProduct.W))
	yaw := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Up(), 0.7)
	pitch := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Right(), -0.4)
	slerp := framework.QuaternionSlerpByQuaternionAndQuaternionAndSingle(yaw, pitch, 0.37)
	check("quaternion.slerp.branch", "QUATERNION", "0xbd9a16ec,0x3e60d7e7,0x00000000,0x3f79023d", floatBits(slerp.X, slerp.Y, slerp.Z, slerp.W))
	check("quaternion.concatenate.order", "QUATERNION", true, framework.QuaternionConcatenateByQuaternionAndQuaternion(yaw, pitch) == framework.QuaternionMultiplyByQuaternionAndQuaternion(pitch, yaw))
	largeAxis := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Up(), 123456.789)
	check("quaternion.axis-angle.large", "QUATERNION", "0x00000000,0x3f30464f,0x00000000,0xbf39a48f", floatBits(largeAxis.X, largeAxis.Y, largeAxis.Z, largeAxis.W))
	fromMatrix := framework.QuaternionCreateFromRotationMatrixByMatrix(framework.MatrixCreateRotationYBySingle(0.7))
	check("quaternion.from-matrix", "QUATERNION", "0x00000000,0x3eaf904c,0x00000000,0x3f707abb", floatBits(fromMatrix.X, fromMatrix.Y, fromMatrix.Z, fromMatrix.W))

	matrix := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4), framework.MatrixCreateRotationYBySingle(0.25))
	matrix = framework.MatrixMultiplyByMatrixAndMatrix(matrix, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	check("matrix.translation.row-vector", "MATRIX", "0x40a00000,0x40c00000,0x40e00000", floatBits(matrix.Translation().X, matrix.Translation().Y, matrix.Translation().Z))
	matrixProduct := framework.MatrixMultiplyByMatrixAndMatrix(matrix, framework.MatrixInvertByMatrix(matrix))
	check("matrix.inverse.product", "MATRIX", "0x3f800000,0x00000000,0xb2000000,0x00000000,0x00000000,0x3f800000,0x00000000,0x00000000,0x33000000,0x00000000,0x3f800000,0x00000000,0x34000000,0x00000000,0x00000000,0x3f800000", floatBits(matrixProduct.M11, matrixProduct.M12, matrixProduct.M13, matrixProduct.M14, matrixProduct.M21, matrixProduct.M22, matrixProduct.M23, matrixProduct.M24, matrixProduct.M31, matrixProduct.M32, matrixProduct.M33, matrixProduct.M34, matrixProduct.M41, matrixProduct.M42, matrixProduct.M43, matrixProduct.M44))
	singular := framework.MatrixInvertByMatrix(framework.Matrix{})
	check("matrix.invert.singular", "MATRIX", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(singular.M11, singular.M22, singular.M33, singular.M44))
	check("matrix.identity.hash", "MATRIX", int32(-33554432), framework.MatrixIdentity().GetHashCode())
	rotation := framework.MatrixCreateRotationYBySingle(123456.789)
	check("matrix.rotation.large", "MATRIX", "0x3d53e807,0xbf7fa83d", floatBits(rotation.M11, rotation.M31))
	perspectiveInfinity := framework.MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(4, 3, 0.1, float32(math.Inf(1)))
	check("matrix.perspective.infinity", "MATRIX", "0xffc00000,0xffc00000", floatBits(perspectiveInfinity.M33, perspectiveInfinity.M43))
	constrainedBillboard := framework.MatrixCreateConstrainedBillboardByVector3AndVector3AndVector3AndNullableOfVector3AndNullableOfVector3(framework.Vector3{Y: 10}, framework.Vector3Zero(), framework.Vector3{Y: 2}, nil, nil)
	check("matrix.billboard.axis", "MATRIX", "0xbf800000,0x40000000,0xbf800000", floatBits(constrainedBillboard.M11, constrainedBillboard.M22, constrainedBillboard.M33))
	zeroPlaneShadow := framework.MatrixCreateShadowByVector3AndPlane(framework.Vector3Forward(), framework.NewPlaneByVector3AndSingle(framework.Vector3Zero(), 0))
	check("matrix.shadow.zero-plane", "MATRIX", "true,true", fmt.Sprintf("%t,%t", math.IsNaN(float64(zeroPlaneShadow.M11)), math.IsNaN(float64(zeroPlaneShadow.M44))))
	degenerateLookAt := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.Vector3Zero(), framework.Vector3Up())
	check("matrix.lookat.degenerate", "MATRIX", "0xffc00000,0xffc00000,0xffc00000,0x00000000,0xffc00000,0xffc00000,0xffc00000,0x00000000,0xffc00000,0xffc00000,0xffc00000,0x00000000,0x7fc00000,0x7fc00000,0x7fc00000,0x3f800000", floatBits(degenerateLookAt.M11, degenerateLookAt.M12, degenerateLookAt.M13, degenerateLookAt.M14, degenerateLookAt.M21, degenerateLookAt.M22, degenerateLookAt.M23, degenerateLookAt.M24, degenerateLookAt.M31, degenerateLookAt.M32, degenerateLookAt.M33, degenerateLookAt.M34, degenerateLookAt.M41, degenerateLookAt.M42, degenerateLookAt.M43, degenerateLookAt.M44))
	infiniteTransformInput := framework.MatrixIdentity()
	infiniteTransformInput.M14 = float32(math.Inf(1))
	infiniteTransform := framework.MatrixTransformByMatrixAndQuaternion(infiniteTransformInput, framework.QuaternionIdentity())
	check("matrix.transform.infinity", "MATRIX", "0x3f800000,0x7f800000,false", fmt.Sprintf("%s,%s,%t", bits(infiniteTransform.M11), bits(infiniteTransform.M14), math.IsNaN(float64(infiniteTransform.M11))))
	mirrored := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(-2, 3, 4), framework.MatrixCreateRotationYBySingle(0.25))
	mirrored = framework.MatrixMultiplyByMatrixAndMatrix(mirrored, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	decomposed, scale, orientation, matrixTranslation := mirrored.Decompose()
	check("matrix.decompose.return", "MATRIX", true, decomposed)
	check("matrix.decompose.outputs", "MATRIX", "0x40000000,0x40400000,0xc0800000,0x00000000,0x3f7e00aa,0x00000000,0xbdff5579,0x40a00000,0x40c00000,0x40e00000", floatBits(scale.X, scale.Y, scale.Z, orientation.X, orientation.Y, orientation.Z, orientation.W, matrixTranslation.X, matrixTranslation.Y, matrixTranslation.Z))
	check("matrix.string", "MATRIX", "{ {M11:1 M12:0 M13:0 M14:0} {M21:0 M22:1 M23:0 M24:0} {M31:0 M32:0 M33:1 M34:0} {M41:0 M42:0 M43:0 M44:1} }", framework.MatrixIdentity().ToString())

	degeneratePlane := framework.NewPlaneByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.Vector3Zero(), framework.Vector3Zero())
	check("plane.points.degenerate", "PLANE", "0xffc00000,0xffc00000,0xffc00000,0x7fc00000", floatBits(degeneratePlane.Normal.X, degeneratePlane.Normal.Y, degeneratePlane.Normal.Z, degeneratePlane.D))
	nearUnitPlane := framework.PlaneNormalizeByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3{X: 0.6, Y: 0.79999995}, 2))
	check("plane.normalize.near-unit", "PLANE", "0x3f19999a,0x3f4ccccc,0x00000000,0x40000000", floatBits(nearUnitPlane.Normal.X, nearUnitPlane.Normal.Y, nearUnitPlane.Normal.Z, nearUnitPlane.D))
	unitBox := framework.NewBoundingBox(framework.NewVector3BySingle(-1), framework.NewVector3BySingle(1))
	check("plane.box.coplanar", "PLANE", framework.PlaneIntersectionTypeIntersecting, (framework.Plane{}).IntersectsByBoundingBox(unitBox))
	reflectionPlane := framework.NewPlaneByVector3AndSingle(framework.Vector3{X: 2}, 4)
	reflection := framework.MatrixCreateReflectionByRefPlaneAndOutMatrix(&reflectionPlane)
	check("plane.reflection.ref-normalization", "PLANE", "0x3f800000,0x40000000,0xbf800000,0xc0800000", floatBits(reflectionPlane.Normal.X, reflectionPlane.D, reflection.M11, reflection.M41))

	unitSphere := framework.NewBoundingSphere(framework.Vector3Zero(), 1)
	rayDistance, rayHit := framework.NewRay(framework.Vector3{X: -5, Y: 0.25}, framework.Vector3UnitX()).IntersectsByBoundingSphere(unitSphere)
	check("ray.sphere.nullable.has-value", "RAY", true, rayHit)
	check("ray.sphere.distance", "RAY", "0x40810421", bits(rayDistance))
	_, rayBoxHit := framework.NewRay(framework.Vector3{X: 2}, framework.Vector3{X: -5e-7}).IntersectsByBoundingBox(unitBox)
	check("ray.box.near-parallel-null", "RAY", false, rayBoxHit)
	_, rayPlaneHit := framework.NewRay(framework.Vector3Zero(), framework.Vector3{X: 5e-6, Y: 1}).IntersectsByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3UnitX(), -1))
	check("ray.plane.near-parallel-null", "RAY", false, rayPlaneHit)
	behindDistance, behindHit := framework.NewRay(framework.Vector3{X: 5e-6}, framework.Vector3UnitX()).IntersectsByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3UnitX(), 0))
	check("ray.plane.just-behind-clamped", "RAY", "true,0x00000000", fmt.Sprintf("%t,%s", behindHit, bits(behindDistance)))

	check("box.point.edge", "BOUNDING_BOX", framework.ContainmentTypeContains, unitBox.ContainsByVector3(framework.Vector3UnitX()))
	check("box.point.nan", "BOUNDING_BOX", framework.ContainmentTypeDisjoint, unitBox.ContainsByVector3(framework.Vector3{X: float32(math.NaN())}))
	nan := float32(math.NaN())
	nanBox := framework.NewBoundingBox(framework.Vector3{X: nan, Y: -1, Z: -1}, framework.Vector3{X: nan, Y: 1, Z: 1})
	check("box.box.nan-intersects", "BOUNDING_BOX", true, unitBox.IntersectsByBoundingBox(nanBox))
	boxCorners := framework.NewBoundingBox(framework.Vector3{X: 1, Y: 2, Z: 3}, framework.Vector3{X: 4, Y: 5, Z: 6}).GetCornersByNone()
	check("box.corner.order", "BOUNDING_BOX", "{X:1 Y:5 Z:6}|{X:4 Y:2 Z:3}|{X:1 Y:2 Z:3}", boxCorners[0].ToString()+"|"+boxCorners[6].ToString()+"|"+boxCorners[7].ToString())

	check("sphere.point.edge", "BOUNDING_SPHERE", framework.ContainmentTypeDisjoint, unitSphere.ContainsByVector3(framework.Vector3UnitX()))
	check("sphere.external-tangent", "BOUNDING_SPHERE", false, unitSphere.IntersectsByBoundingSphere(framework.NewBoundingSphere(framework.Vector3{X: 2}, 1)))
	pointsSphere := framework.BoundingSphereCreateFromPoints([]framework.Vector3{{X: -4, Y: 1}, {X: 6, Y: -2, Z: 3}, {Y: 8, Z: -5}, {X: 2, Z: 9}})
	check("sphere.create-points", "BOUNDING_SPHERE", "0x3f800000,0x40800000,0x40000000,0x4101fc10", floatBits(pointsSphere.Center.X, pointsSphere.Center.Y, pointsSphere.Center.Z, pointsSphere.Radius))

	projection := framework.MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(framework.MathHelperPiOver4, 4.0/3.0, 1, 10)
	view := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3{Z: 5}, framework.Vector3Zero(), framework.Vector3Up())
	frustum := framework.NewBoundingFrustum(framework.MatrixMultiplyByMatrixAndMatrix(view, projection))
	frustumNear := frustum.Near()
	check("frustum.near-plane", "BOUNDING_FRUSTUM", "0x80000000,0x80000000,0x3f800000,0xc0800000", floatBits(frustumNear.Normal.X, frustumNear.Normal.Y, frustumNear.Normal.Z, frustumNear.D))
	frustumTop := frustum.Top()
	check("frustum.top-plane", "BOUNDING_FRUSTUM", "0x00000000,0x3f6c835f,0x3ec3ef16,0xbff4eadb", floatBits(frustumTop.Normal.X, frustumTop.Normal.Y, frustumTop.Normal.Z, frustumTop.D))
	frustumCorners := frustum.GetCornersByNone()
	check("frustum.corner-order.first", "BOUNDING_FRUSTUM", "0xbf0d6289,0x3ed413cb,0x40800000", floatBits(frustumCorners[0].X, frustumCorners[0].Y, frustumCorners[0].Z))
	check("frustum.corner-order.seventh", "BOUNDING_FRUSTUM", "0x40b0bb28,0xc0848c5d,0xc09ffff8", floatBits(frustumCorners[6].X, frustumCorners[6].Y, frustumCorners[6].Z))
	check("frustum.box.gjk.inside", "BOUNDING_FRUSTUM", true, frustum.IntersectsByBoundingBox(framework.NewBoundingBox(framework.NewVector3BySingle(-0.5), framework.NewVector3BySingle(0.5))))
	check("frustum.box.gjk.outside", "BOUNDING_FRUSTUM", false, frustum.IntersectsByBoundingBox(framework.NewBoundingBox(framework.NewVector3BySingle(100), framework.NewVector3BySingle(101))))
	frustumDistance, frustumRayHit := frustum.IntersectsByRay(framework.NewRay(framework.Vector3{Z: 20}, framework.Vector3Forward()))
	check("frustum.ray.nullable", "BOUNDING_FRUSTUM", "true,0x41800000", fmt.Sprintf("%t,%s", frustumRayHit, bits(frustumDistance)))
	check("frustum.class.nil-equality", "BOUNDING_FRUSTUM", false, framework.BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(frustum, nil))

	floatColor := framework.NewColorBySingleAndSingleAndSingleAndSingle(0.5, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)))
	check("color.float-packing", "COLOR", "0x00ff0080", packed(floatColor.PackedValue()))
	check("color.lerp.midpoint", "COLOR", "0x7f7f7f7f", packed(framework.ColorLerp(framework.NewColorByInt32AndInt32AndInt32AndInt32(0, 0, 0, 0), framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255), 0.5).PackedValue()))
	check("color.multiply.midpoint", "COLOR", "0x7f7f7f7f", packed(framework.ColorMultiply(framework.ColorWhite(), 0.5).PackedValue()))
	check("color.transparent", "COLOR", "0x00ffffff", packed(framework.ColorTransparent().PackedValue()))
	check("color.cornflower-blue", "COLOR", "0xffed9564", packed(framework.ColorCornflowerBlue().PackedValue()))
	colorVector := framework.NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4).ToVector4()
	check("color.vector-roundtrip", "COLOR", true, framework.NewColorByVector4(colorVector) == framework.NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4))

	viewport := graphics.NewViewportByInt32AndInt32AndInt32AndInt32(11, 13, 640, 360)
	viewport.SetMinDepth(0.2)
	viewport.SetMaxDepth(0.9)
	viewportWorld := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(1.5, 0.75, 2), framework.MatrixCreateRotationYBySingle(0.31))
	viewportWorld = framework.MatrixMultiplyByMatrixAndMatrix(viewportWorld, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(2, -1, 0.5))
	viewportView := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3{X: 4, Y: 3, Z: 8}, framework.Vector3Zero(), framework.Vector3Up())
	viewportProjection := framework.MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(0.9, 16.0/9.0, 0.1, 100)
	projected := viewport.Project(framework.Vector3{X: 0.25, Y: -0.5, Z: 1.25}, viewportProjection, viewportView, viewportWorld)
	check("viewport.project", "VIEWPORT", "0x43d42808,0x43ac9f3c,0x3f63aff4", floatBits(projected.X, projected.Y, projected.Z))
	unprojected := viewport.Unproject(projected, viewportProjection, viewportView, viewportWorld)
	check("viewport.unproject", "VIEWPORT", "0x3e7ffe10,0xbefff906,0x3fa00111", floatBits(unprojected.X, unprojected.Y, unprojected.Z))
	singularUnproject := viewport.Unproject(framework.Vector3{X: 100, Y: 50, Z: 0.5}, framework.MatrixIdentity(), framework.MatrixIdentity(), framework.Matrix{})
	check("viewport.unproject.singular", "VIEWPORT", "0xffc00000,0xffc00000,0xffc00000", floatBits(singularUnproject.X, singularUnproject.Y, singularUnproject.Z))

	check("vertex-element.enums.format", "VERTEX_ELEMENT_ENUMS", "0,1,2,3,4,5,6,7,8,9,10,11", fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d",
		graphics.VertexElementFormatSingle,
		graphics.VertexElementFormatVector2,
		graphics.VertexElementFormatVector3,
		graphics.VertexElementFormatVector4,
		graphics.VertexElementFormatColor,
		graphics.VertexElementFormatByte4,
		graphics.VertexElementFormatShort2,
		graphics.VertexElementFormatShort4,
		graphics.VertexElementFormatNormalizedShort2,
		graphics.VertexElementFormatNormalizedShort4,
		graphics.VertexElementFormatHalfVector2,
		graphics.VertexElementFormatHalfVector4,
	))
	check("vertex-element.enums.usage", "VERTEX_ELEMENT_ENUMS", "0,1,2,3,4,5,6,7,8,9,10,11,12", fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d",
		graphics.VertexElementUsagePosition,
		graphics.VertexElementUsageColor,
		graphics.VertexElementUsageTextureCoordinate,
		graphics.VertexElementUsageNormal,
		graphics.VertexElementUsageBinormal,
		graphics.VertexElementUsageTangent,
		graphics.VertexElementUsageBlendIndices,
		graphics.VertexElementUsageBlendWeight,
		graphics.VertexElementUsageDepth,
		graphics.VertexElementUsageFog,
		graphics.VertexElementUsagePointSize,
		graphics.VertexElementUsageSample,
		graphics.VertexElementUsageTessellateFactor,
	))

	var zeroVertexElement graphics.VertexElement
	constructedZeroVertexElement := graphics.NewVertexElement(0, graphics.VertexElementFormatSingle, graphics.VertexElementUsagePosition, 0)
	check("vertex-element.zero.getters", "VERTEX_ELEMENT", "0,0,0,0", fmt.Sprintf("%d,%d,%d,%d", zeroVertexElement.Offset(), zeroVertexElement.VertexElementFormat(), zeroVertexElement.VertexElementUsage(), zeroVertexElement.UsageIndex()))
	check("vertex-element.zero.constructor-equivalence", "VERTEX_ELEMENT", "true,2147483647,{Offset:0 Format:Single Usage:Position UsageIndex:0}", fmt.Sprintf("%t,%d,%s", zeroVertexElement.Equals(constructedZeroVertexElement), zeroVertexElement.GetHashCode(), zeroVertexElement.ToString()))

	ordinaryVertexElement := graphics.NewVertexElement(12, graphics.VertexElementFormatVector3, graphics.VertexElementUsageTextureCoordinate, 7)
	check("vertex-element.constructor", "VERTEX_ELEMENT", "12,2,2,7", fmt.Sprintf("%d,%d,%d,%d", ordinaryVertexElement.Offset(), ordinaryVertexElement.VertexElementFormat(), ordinaryVertexElement.VertexElementUsage(), ordinaryVertexElement.UsageIndex()))
	offsetVertexElement := ordinaryVertexElement
	offsetVertexElement.SetOffset(13)
	check("vertex-element.copy.offset", "VERTEX_ELEMENT", "12,13", fmt.Sprintf("%d,%d", ordinaryVertexElement.Offset(), offsetVertexElement.Offset()))
	formatVertexElement := ordinaryVertexElement
	formatVertexElement.SetVertexElementFormat(graphics.VertexElementFormatHalfVector4)
	check("vertex-element.copy.format", "VERTEX_ELEMENT", "2,11", fmt.Sprintf("%d,%d", ordinaryVertexElement.VertexElementFormat(), formatVertexElement.VertexElementFormat()))
	usageVertexElement := ordinaryVertexElement
	usageVertexElement.SetVertexElementUsage(graphics.VertexElementUsageTangent)
	check("vertex-element.copy.usage", "VERTEX_ELEMENT", "2,5", fmt.Sprintf("%d,%d", ordinaryVertexElement.VertexElementUsage(), usageVertexElement.VertexElementUsage()))
	indexVertexElement := ordinaryVertexElement
	indexVertexElement.SetUsageIndex(8)
	check("vertex-element.copy.usage-index", "VERTEX_ELEMENT", "7,8", fmt.Sprintf("%d,%d", ordinaryVertexElement.UsageIndex(), indexVertexElement.UsageIndex()))

	unknownVertexElement := graphics.NewVertexElement(123, graphics.VertexElementFormat(12345), graphics.VertexElementUsage(-23456), -456)
	check("vertex-element.undefined.storage", "VERTEX_ELEMENT", "123,12345,-23456,-456", fmt.Sprintf("%d,%d,%d,%d", unknownVertexElement.Offset(), unknownVertexElement.VertexElementFormat(), unknownVertexElement.VertexElementUsage(), unknownVertexElement.UsageIndex()))
	boundaryVertexElement := graphics.NewVertexElement(math.MinInt32, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTessellateFactor, math.MaxInt32)
	check("vertex-element.int32-boundaries", "VERTEX_ELEMENT", "-2147483648,2147483647", fmt.Sprintf("%d,%d", boundaryVertexElement.Offset(), boundaryVertexElement.UsageIndex()))
	check("vertex-element.equals-object", "VERTEX_ELEMENT", "true,false,false,false", fmt.Sprintf("%t,%t,%t,%t", ordinaryVertexElement.Equals(ordinaryVertexElement), ordinaryVertexElement.Equals(&ordinaryVertexElement), ordinaryVertexElement.Equals(nil), ordinaryVertexElement.Equals(int32(12))))
	check("vertex-element.equals-field-differences", "VERTEX_ELEMENT", "false,false,false,false", fmt.Sprintf("%t,%t,%t,%t", ordinaryVertexElement.Equals(offsetVertexElement), ordinaryVertexElement.Equals(formatVertexElement), ordinaryVertexElement.Equals(usageVertexElement), ordinaryVertexElement.Equals(indexVertexElement)))
	unknownVertexElementCopy := unknownVertexElement
	check("vertex-element.equals-undefined", "VERTEX_ELEMENT", true, unknownVertexElement.Equals(unknownVertexElementCopy))
	check("vertex-element.operators", "VERTEX_ELEMENT", "true,true", fmt.Sprintf("%t,%t", graphics.VertexElementOperatorEqualityByVertexElementAndVertexElement(ordinaryVertexElement, ordinaryVertexElement), graphics.VertexElementOperatorInequalityByVertexElementAndVertexElement(ordinaryVertexElement, offsetVertexElement)))

	check("vertex-element.hash.zero-fallback", "VERTEX_ELEMENT", int32(math.MaxInt32), zeroVertexElement.GetHashCode())
	check("vertex-element.hash.ordinary", "VERTEX_ELEMENT", int32(11), ordinaryVertexElement.GetHashCode())
	check("vertex-element.hash.negative", "VERTEX_ELEMENT", int32(3), graphics.NewVertexElement(-16, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTangent, -3).GetHashCode())
	check("vertex-element.hash.undefined", "VERTEX_ELEMENT", int32(27162), unknownVertexElement.GetHashCode())
	check("vertex-element.hash.boundaries", "VERTEX_ELEMENT", int32(-8), boundaryVertexElement.GetHashCode())
	check("vertex-element.hash.nonzero-collision", "VERTEX_ELEMENT", int32(math.MaxInt32), graphics.NewVertexElement(1, graphics.VertexElementFormatVector3, graphics.VertexElementUsageNormal, 0).GetHashCode())

	check("vertex-element.string.zero", "VERTEX_ELEMENT", "{Offset:0 Format:Single Usage:Position UsageIndex:0}", zeroVertexElement.ToString())
	check("vertex-element.string.ordinary", "VERTEX_ELEMENT", "{Offset:12 Format:Vector3 Usage:TextureCoordinate UsageIndex:7}", ordinaryVertexElement.ToString())
	check("vertex-element.string.negative", "VERTEX_ELEMENT", "{Offset:-16 Format:HalfVector4 Usage:Tangent UsageIndex:-3}", graphics.NewVertexElement(-16, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTangent, -3).ToString())
	check("vertex-element.string.undefined", "VERTEX_ELEMENT", "{Offset:123 Format:12345 Usage:-23456 UsageIndex:-456}", unknownVertexElement.ToString())
	check("vertex-element.string.boundaries", "VERTEX_ELEMENT", "{Offset:-2147483648 Format:HalfVector4 Usage:TessellateFactor UsageIndex:2147483647}", boundaryVertexElement.ToString())

	check("curve.enums.continuity", "CURVE_ENUMS", "0,1", fmt.Sprintf("%d,%d", framework.CurveContinuitySmooth, framework.CurveContinuityStep))
	check("curve.enums.tangent", "CURVE_ENUMS", "0,1,2", fmt.Sprintf("%d,%d,%d", framework.CurveTangentFlat, framework.CurveTangentLinear, framework.CurveTangentSmooth))
	check("curve.enums.loop", "CURVE_ENUMS", "0,1,2,3,4", fmt.Sprintf("%d,%d,%d,%d,%d", framework.CurveLoopTypeConstant, framework.CurveLoopTypeCycle, framework.CurveLoopTypeCycleOffset, framework.CurveLoopTypeOscillate, framework.CurveLoopTypeLinear))

	curveKey := framework.NewCurveKeyBySingleAndSingle(1, 2)
	check("curve.key.defaults", "CURVE_KEY", "0x00000000,0x00000000,0", fmt.Sprintf("%s,%s,%d", bits(curveKey.TangentIn()), bits(curveKey.TangentOut()), curveKey.Continuity()))
	fullCurveKey := framework.NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(1, 2, 3, 4, framework.CurveContinuityStep)
	curveKeyClone := fullCurveKey.Clone()
	check("curve.key.clone.identity", "CURVE_KEY", true, curveKeyClone != fullCurveKey && curveKeyClone.EqualsByCurveKey(fullCurveKey))
	curveKeyClone.SetValue(9)
	check("curve.key.clone.independent", "CURVE_KEY", "0x40000000,0x41100000", floatBits(fullCurveKey.Value(), curveKeyClone.Value()))
	comparisonLess, comparisonLessError := fullCurveKey.CompareTo(framework.NewCurveKeyBySingleAndSingle(2, 0))
	comparisonEqual, comparisonEqualError := fullCurveKey.CompareTo(framework.NewCurveKeyBySingleAndSingle(1, 99))
	check("curve.key.compare.finite", "CURVE_KEY", "-1,0,true", fmt.Sprintf("%d,%d,%t", comparisonLess, comparisonEqual, comparisonLessError == nil && comparisonEqualError == nil))
	comparisonNaN, comparisonNaNError := framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1))
	check("curve.key.compare.nan", "CURVE_KEY", "0,true", fmt.Sprintf("%d,%t", comparisonNaN, comparisonNaNError == nil))
	comparisonNaNFinite, _ := framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(0, 1))
	comparisonFiniteNaN, _ := framework.NewCurveKeyBySingleAndSingle(0, 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1))
	check("curve.key.compare.nan-order", "CURVE_KEY", "-1,1", fmt.Sprintf("%d,%d", comparisonNaNFinite, comparisonFiniteNaN))
	check("curve.key.nan.equality", "CURVE_KEY", false, framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).EqualsByCurveKey(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1)))
	check("curve.key.hash", "CURVE_KEY", int32(4194305), fullCurveKey.GetHashCode())
	check("curve.key.operators", "CURVE_KEY", "true,true", fmt.Sprintf("%t,%t", framework.CurveKeyOperatorEqualityByCurveKeyAndCurveKey(fullCurveKey, fullCurveKey.Clone()), framework.CurveKeyOperatorInequalityByCurveKeyAndCurveKey(nil, fullCurveKey)))
	_, comparisonNilError := fullCurveKey.CompareTo(nil)
	check("curve.key.compare.nil-error", "CURVE_KEY", true, comparisonNilError != nil)

	mustAddCurveKey := func(collection *framework.CurveKeyCollection, key *framework.CurveKey) {
		if err := collection.Add(key); err != nil {
			panic(err)
		}
	}
	mustCurveKeyAt := func(collection *framework.CurveKeyCollection, index int32) *framework.CurveKey {
		key, err := collection.Item(index)
		if err != nil {
			panic(err)
		}
		return key
	}
	curveCollection := framework.NewCurveKeyCollection()
	duplicateA := framework.NewCurveKeyBySingleAndSingle(1, 10)
	duplicateB := framework.NewCurveKeyBySingleAndSingle(1, 20)
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(2, 30))
	mustAddCurveKey(curveCollection, duplicateA)
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(0, 0))
	mustAddCurveKey(curveCollection, duplicateB)
	check("curve.collection.order", "CURVE_COLLECTION", "0x00000000,0x3f800000,0x3f800000,0x40000000", floatBits(mustCurveKeyAt(curveCollection, 0).Position(), mustCurveKeyAt(curveCollection, 1).Position(), mustCurveKeyAt(curveCollection, 2).Position(), mustCurveKeyAt(curveCollection, 3).Position()))
	check("curve.collection.duplicate-identity", "CURVE_COLLECTION", true, mustCurveKeyAt(curveCollection, 1) == duplicateA && mustCurveKeyAt(curveCollection, 2) == duplicateB)
	duplicateA.SetValue(11)
	equalDuplicate := framework.NewCurveKeyBySingleAndSingle(1, 11)
	check("curve.collection.value-equality", "CURVE_COLLECTION", "true,1", fmt.Sprintf("%t,%d", curveCollection.Contains(equalDuplicate), curveCollection.IndexOf(equalDuplicate)))
	replacement := framework.NewCurveKeyBySingleAndSingle(3, 40)
	if err := curveCollection.SetItem(0, replacement); err != nil {
		panic(err)
	}
	check("curve.collection.item-reorders", "CURVE_COLLECTION", "0x3f800000,0x3f800000,0x40000000,0x40400000", floatBits(mustCurveKeyAt(curveCollection, 0).Position(), mustCurveKeyAt(curveCollection, 1).Position(), mustCurveKeyAt(curveCollection, 2).Position(), mustCurveKeyAt(curveCollection, 3).Position()))
	samePosition := framework.NewCurveKeyBySingleAndSingle(3, 41)
	if err := curveCollection.SetItem(3, samePosition); err != nil {
		panic(err)
	}
	check("curve.collection.item-identity", "CURVE_COLLECTION", true, mustCurveKeyAt(curveCollection, 3) == samePosition)
	_, negativeItemError := curveCollection.Item(-1)
	setEndError := curveCollection.SetItem(curveCollection.Count(), samePosition)
	removeNegativeError := curveCollection.RemoveAt(-1)
	check("curve.collection.index-errors", "CURVE_COLLECTION", "true,true,true", fmt.Sprintf("%t,%t,%t", negativeItemError != nil, setEndError != nil, removeNegativeError != nil))
	destination := make([]*framework.CurveKey, 6)
	copyError := curveCollection.CopyTo(destination, 1)
	check("curve.collection.copyto", "CURVE_COLLECTION", true, copyError == nil && destination[1] == duplicateA && destination[4] == samePosition)
	check("curve.collection.copyto-errors", "CURVE_COLLECTION", true, curveCollection.CopyTo(nil, 0) != nil && curveCollection.CopyTo(make([]*framework.CurveKey, 4), 1) != nil)
	collectionClone := curveCollection.Clone()
	check("curve.collection.clone-depth", "CURVE_COLLECTION", true, collectionClone != curveCollection && mustCurveKeyAt(collectionClone, 0) == mustCurveKeyAt(curveCollection, 0))
	iterator := curveCollection.GetEnumerator()
	firstEnumerated, firstEnumeratedOK, firstEnumeratedError := iterator.Next()
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(4, 50))
	_, _, invalidatedError := iterator.Next()
	check("curve.collection.enumerator-invalidation", "CURVE_COLLECTION", true, firstEnumerated == duplicateA && firstEnumeratedOK && firstEnumeratedError == nil && invalidatedError != nil)
	check("curve.collection.remove-equal", "CURVE_COLLECTION", true, curveCollection.Remove(equalDuplicate) && !curveCollection.Contains(equalDuplicate))
	curveCollection.Clear()
	check("curve.collection.clear-readonly", "CURVE_COLLECTION", "0,false", fmt.Sprintf("%d,%t", curveCollection.Count(), curveCollection.IsReadOnly()))

	tangentCurve := framework.NewCurve()
	for _, pair := range [][2]float32{{0, 0}, {1, 10}, {3, 30}} {
		mustAddCurveKey(tangentCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(pair[0], pair[1]))
	}
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangent(1, framework.CurveTangentFlat); err != nil {
		panic(err)
	}
	check("curve.tangent.flat", "CURVE_TANGENTS", "0x00000000,0x00000000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangent(1, framework.CurveTangentLinear); err != nil {
		panic(err)
	}
	check("curve.tangent.linear", "CURVE_TANGENTS", "0x41200000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	tangentCurve.ComputeTangentsByCurveTangent(framework.CurveTangentSmooth)
	check("curve.tangent.smooth-first", "CURVE_TANGENTS", "0x00000000,0x41200000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 0).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 0).TangentOut()))
	check("curve.tangent.smooth-middle", "CURVE_TANGENTS", "0x41200000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	check("curve.tangent.smooth-last", "CURVE_TANGENTS", "0x41a00000,0x00000000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 2).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 2).TangentOut()))
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangentAndCurveTangent(1, framework.CurveTangentFlat, framework.CurveTangentSmooth); err != nil {
		panic(err)
	}
	check("curve.tangent.mixed", "CURVE_TANGENTS", "0x00000000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	check("curve.tangent.invalid-index", "CURVE_TANGENTS", true, tangentCurve.ComputeTangentByInt32AndCurveTangent(-1, framework.CurveTangentFlat) != nil)

	emptyCurve := framework.NewCurve()
	check("curve.evaluate.empty", "CURVE_EVALUATE", "0x00000000", bits(emptyCurve.Evaluate(5)))
	check("curve.evaluate.defaults", "CURVE_EVALUATE", "0,0,true", fmt.Sprintf("%d,%d,%t", emptyCurve.PreLoop(), emptyCurve.PostLoop(), emptyCurve.IsConstant()))
	singleCurve := framework.NewCurve()
	mustAddCurveKey(singleCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(2, 7))
	check("curve.evaluate.single", "CURVE_EVALUATE", "0x40e00000,0x40e00000,0x40e00000", floatBits(singleCurve.Evaluate(-1), singleCurve.Evaluate(2), singleCurve.Evaluate(9)))
	hermiteCurve := framework.NewCurve()
	mustAddCurveKey(hermiteCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(0, 0))
	mustAddCurveKey(hermiteCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 10))
	check("curve.evaluate.hermite", "CURVE_EVALUATE", "0x3fc80000", bits(hermiteCurve.Evaluate(0.25)))
	asymmetricCurve := framework.NewCurve()
	mustAddCurveKey(asymmetricCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingle(0, 0, 99, 4))
	mustAddCurveKey(asymmetricCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingle(2, 10, -2, 77))
	check("curve.evaluate.hermite-asymmetric", "CURVE_EVALUATE", "0x40b80000", bits(asymmetricCurve.Evaluate(1)))
	stepCurve := framework.NewCurve()
	mustAddCurveKey(stepCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(0, 2, 0, 0, framework.CurveContinuityStep))
	mustAddCurveKey(stepCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 9))
	check("curve.evaluate.step", "CURVE_EVALUATE", "0x40000000,0x41100000", floatBits(stepCurve.Evaluate(0.999), stepCurve.Evaluate(1)))
	duplicateCurve := framework.NewCurve()
	mustAddCurveKey(duplicateCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 10))
	mustAddCurveKey(duplicateCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 20))
	check("curve.evaluate.duplicate", "CURVE_EVALUATE", "0x41200000", bits(duplicateCurve.Evaluate(1)))
	curveClone := hermiteCurve.Clone()
	check("curve.evaluate.clone-depth", "CURVE_EVALUATE", true, curveClone != hermiteCurve && curveClone.Keys() != hermiteCurve.Keys() && mustCurveKeyAt(curveClone.Keys(), 0) == mustCurveKeyAt(hermiteCurve.Keys(), 0))

	newLoopCurve := func() *framework.Curve {
		curve := framework.NewCurve()
		mustAddCurveKey(curve.Keys(), framework.NewCurveKeyBySingleAndSingle(5, 0))
		mustAddCurveKey(curve.Keys(), framework.NewCurveKeyBySingleAndSingle(7, 10))
		return curve
	}
	constantLoop := newLoopCurve()
	check("curve.loop.constant", "CURVE_LOOPS", "0x00000000,0x41200000", floatBits(constantLoop.Evaluate(4), constantLoop.Evaluate(8)))
	cycleLoop := newLoopCurve()
	cycleLoop.SetPreLoop(framework.CurveLoopTypeCycle)
	cycleLoop.SetPostLoop(framework.CurveLoopTypeCycle)
	check("curve.loop.cycle", "CURVE_LOOPS", "0x40a00000,0x40a00000", floatBits(cycleLoop.Evaluate(4), cycleLoop.Evaluate(8)))
	offsetLoop := newLoopCurve()
	offsetLoop.SetPreLoop(framework.CurveLoopTypeCycleOffset)
	offsetLoop.SetPostLoop(framework.CurveLoopTypeCycleOffset)
	check("curve.loop.cycle-offset", "CURVE_LOOPS", "0xc0a00000,0x41700000", floatBits(offsetLoop.Evaluate(4), offsetLoop.Evaluate(8)))
	oscillateLoop := newLoopCurve()
	oscillateLoop.SetPreLoop(framework.CurveLoopTypeOscillate)
	oscillateLoop.SetPostLoop(framework.CurveLoopTypeOscillate)
	check("curve.loop.oscillate", "CURVE_LOOPS", "0x40a00000,0x40a00000", floatBits(oscillateLoop.Evaluate(4), oscillateLoop.Evaluate(8)))
	linearLoop := newLoopCurve()
	mustCurveKeyAt(linearLoop.Keys(), 0).SetTangentIn(2)
	mustCurveKeyAt(linearLoop.Keys(), 1).SetTangentOut(3)
	linearLoop.SetPreLoop(framework.CurveLoopTypeLinear)
	linearLoop.SetPostLoop(framework.CurveLoopTypeLinear)
	check("curve.loop.linear", "CURVE_LOOPS", "0xc0000000,0x41800000", floatBits(linearLoop.Evaluate(4), linearLoop.Evaluate(9)))
	check("curve.loop.negative-cycle", "CURVE_LOOPS", "0x41200000", bits(cycleLoop.Evaluate(3)))
	check("curve.loop.negative-cycle-offset", "CURVE_LOOPS", "0xc1200000", bits(offsetLoop.Evaluate(3)))
	check("curve.loop.negative-oscillate", "CURVE_LOOPS", "0x41200000", bits(oscillateLoop.Evaluate(3)))
	check("curve.loop.multiple-offset", "CURVE_LOOPS", "0xc1a00000,0x41a00000", floatBits(offsetLoop.Evaluate(1), offsetLoop.Evaluate(9)))

	check("packed.alpha.zero", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(0).PackedValue())
	check("packed.alpha.ordinary", "PACKED_ALPHA", uint8(128), packedvector.NewAlpha8(0.5).PackedValue())
	check("packed.alpha.clamp-low", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(-1).PackedValue())
	check("packed.alpha.clamp-high", "PACKED_ALPHA", uint8(255), packedvector.NewAlpha8(2).PackedValue())
	check("packed.alpha.tie-even-zero", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(float32(0.5/255)).PackedValue())
	check("packed.alpha.tie-even-two", "PACKED_ALPHA", uint8(2), packedvector.NewAlpha8(float32(2.5/255)).PackedValue())
	check("packed.alpha.non-finite", "PACKED_ALPHA", "0,255,0", fmt.Sprintf("%d,%d,%d", packedvector.NewAlpha8(float32(math.NaN())).PackedValue(), packedvector.NewAlpha8(float32(math.Inf(1))).PackedValue(), packedvector.NewAlpha8(float32(math.Inf(-1))).PackedValue()))

	check("packed.bgr565.ordinary", "PACKED_16BIT_COLOR", uint16(17431), packedvector.NewBgr565BySingleAndSingleAndSingle(0.25, 0.5, 0.75).PackedValue())
	check("packed.bgr565.lanes", "PACKED_16BIT_COLOR", "63488,2016,31", fmt.Sprintf("%d,%d,%d", packedvector.NewBgr565BySingleAndSingleAndSingle(1, 0, 0).PackedValue(), packedvector.NewBgr565BySingleAndSingleAndSingle(0, 1, 0).PackedValue(), packedvector.NewBgr565BySingleAndSingleAndSingle(0, 0, 1).PackedValue()))
	check("packed.bgra4444.ordinary", "PACKED_16BIT_COLOR", uint16(50025), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0.2, 0.4, 0.6, 0.8).PackedValue())
	check("packed.bgra4444.lanes", "PACKED_16BIT_COLOR", "3840,240,15,61440", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))
	check("packed.bgra5551.ordinary", "PACKED_16BIT_COLOR", uint16(8727), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue())
	check("packed.bgra5551.alpha-threshold", "PACKED_16BIT_COLOR", "0,0,32768", fmt.Sprintf("%d,%d,%d", packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 0)).PackedValue(), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, 0.5).PackedValue(), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 1)).PackedValue()))

	check("packed.byte4.raw-domain", "PACKED_BYTE4", uint32(4286578944), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(-1, 1, 128, 256).PackedValue())
	check("packed.byte4.ties", "PACKED_BYTE4", uint32(67240448), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(0.5, 1.5, 2.5, 3.5).PackedValue())
	check("packed.byte4.non-finite", "PACKED_BYTE4", uint32(0x8000ff00), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 127.5).PackedValue())
	check("packed.byte4.decode", "PACKED_BYTE4", "0x00000000,0x3f800000,0x43000000,0x437f0000", func() string {
		value := packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(-1, 1, 128, 256).ToVector4()
		return floatBits(value.X, value.Y, value.Z, value.W)
	}())

	check("packed.half.single", "PACKED_HALF", uint16(13653), packedvector.NewHalfSingle(float32(1.0/3.0)).PackedValue())
	check("packed.half.vector2-order", "PACKED_HALF", uint32(3221240832), packedvector.NewHalfVector2BySingleAndSingle(1, -2).PackedValue())
	check("packed.half.vector4-order", "PACKED_HALF", uint64(9223433612727172096), packedvector.NewHalfVector4BySingleAndSingleAndSingleAndSingle(1, -2, 0.5, math.Float32frombits(0x80000000)).PackedValue())
	halfFixtures := []struct {
		id      string
		input   uint32
		packed  uint16
		decoded uint32
		text    string
	}{
		{"positive-zero", 0x00000000, 0x0000, 0x00000000, "0"},
		{"negative-zero", 0x80000000, 0x8000, 0x80000000, "0"},
		{"smallest-subnormal", 0x33800000, 0x0001, 0x33800000, "5.960464E-08"},
		{"largest-subnormal", 0x387fc000, 0x03ff, 0x387fc000, "6.097555E-05"},
		{"smallest-normal", 0x38800000, 0x0400, 0x38800000, "6.103516E-05"},
		{"tie-even-low", 0x3f801000, 0x3c00, 0x3f800000, "1"},
		{"tie-even-high", 0x3f803000, 0x3c02, 0x3f804000, "1.001953"},
		{"maximum-conventional-finite", 0x477fe000, 0x7bff, 0x477fe000, "65504"},
		{"exponent31-boundary", 0x477ff000, 0x7c00, 0x47800000, "65536"},
		{"positive-infinity", 0x7f800000, 0x7fff, 0x47ffe000, "131008"},
		{"negative-infinity", 0xff800000, 0xffff, 0xc7ffe000, "-131008"},
		{"positive-nan", 0x7fc12345, 0x7fff, 0x47ffe000, "131008"},
		{"negative-nan", 0xffc12345, 0xffff, 0xc7ffe000, "-131008"},
	}
	for _, fixture := range halfFixtures {
		value := packedvector.NewHalfSingle(math.Float32frombits(fixture.input))
		actual := fmt.Sprintf("%d,%08X,%s", value.PackedValue(), math.Float32bits(value.ToSingle()), value.ToString())
		expected := fmt.Sprintf("%d,%08X,%s", fixture.packed, fixture.decoded, fixture.text)
		check("packed.half."+fixture.id, "PACKED_HALF", expected, actual)
	}

	check("packed.normalized-byte2", "PACKED_NORMALIZED_BYTE", uint16(16513), packedvector.NewNormalizedByte2BySingleAndSingle(-1, 0.5).PackedValue())
	check("packed.normalized-byte4-endpoints", "PACKED_NORMALIZED_BYTE", uint32(2139029633), packedvector.NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue())
	check("packed.normalized-byte4-ties", "PACKED_NORMALIZED_BYTE", uint32(4261413376), packedvector.NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(float32(0.5/127), float32(1.5/127), float32(-0.5/127), float32(-1.5/127)).PackedValue())
	var normalizedByteMinimum packedvector.NormalizedByte2
	normalizedByteMinimum.SetPackedValue(0x0080)
	check("packed.normalized-byte-minimum-decodes-minus-one", "PACKED_NORMALIZED_BYTE", "0xbf800000", bits(normalizedByteMinimum.ToVector2().X))

	check("packed.normalized-short2", "PACKED_NORMALIZED_SHORT", uint32(1073774593), packedvector.NewNormalizedShort2BySingleAndSingle(-1, 0.5).PackedValue())
	check("packed.normalized-short4-endpoints", "PACKED_NORMALIZED_SHORT", uint64(9223231295071485953), packedvector.NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue())
	check("packed.normalized-short4-ties", "PACKED_NORMALIZED_SHORT", uint64(18446181123756261376), packedvector.NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(float32(0.5/32767), float32(1.5/32767), float32(-0.5/32767), float32(-1.5/32767)).PackedValue())
	var normalizedShortMinimum packedvector.NormalizedShort2
	normalizedShortMinimum.SetPackedValue(0x00008000)
	check("packed.normalized-short-minimum-decodes-minus-one", "PACKED_NORMALIZED_SHORT", "0xbf800000", bits(normalizedShortMinimum.ToVector2().X))

	check("packed.rg32.ordinary", "PACKED_RG_RGBA", uint32(3221176320), packedvector.NewRg32BySingleAndSingle(0.25, 0.75).PackedValue())
	check("packed.rg32.lanes", "PACKED_RG_RGBA", "65535,4294901760", fmt.Sprintf("%d,%d", packedvector.NewRg32BySingleAndSingle(1, 0).PackedValue(), packedvector.NewRg32BySingleAndSingle(0, 1).PackedValue()))
	check("packed.rgba1010102.ordinary", "PACKED_RG_RGBA", uint32(2952265984), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue())
	check("packed.rgba1010102.lanes", "PACKED_RG_RGBA", "1023,1047552,1072693248,3221225472", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))
	check("packed.rgba64.ordinary", "PACKED_RG_RGBA", uint64(18446673702817906688), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1).PackedValue())
	check("packed.rgba64.lanes", "PACKED_RG_RGBA", "65535,4294901760,281470681743360,18446462598732840960", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))

	check("packed.short2.endpoints", "PACKED_SHORT", uint32(2147450880), packedvector.NewShort2BySingleAndSingle(-32768, 32767).PackedValue())
	check("packed.short4.ordinary", "PACKED_SHORT", uint64(9223090574762868736), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(-32768, -1.5, 2.5, 32768).PackedValue())
	check("packed.short4.ties", "PACKED_SHORT", uint64(562954248257536), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(-0.5, -1.5, 0.5, 1.5).PackedValue())
	check("packed.short4.non-finite", "PACKED_SHORT", uint64(0x7fff80007fff0000), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 32768).PackedValue())

	interfaceInput := framework.Vector4{X: -2, Y: 0.5, Z: 2, W: 0.25}
	alphaInterface := packedvector.Alpha8{}
	var alphaPacked packedvector.IPackedVectorOfTPacked[uint8] = &alphaInterface
	alphaPacked.PackFromVector4(interfaceInput)
	check("packed.interface.alpha8", "PACKED_INTERFACE", "64,0x00000000,0x00000000,0x00000000,0x3e808081", fmt.Sprintf("%d,%s", alphaPacked.PackedValue(), floatBits(alphaPacked.ToVector4().X, alphaPacked.ToVector4().Y, alphaPacked.ToVector4().Z, alphaPacked.ToVector4().W)))
	bgrInterface := packedvector.Bgr565{}
	var bgrPacked packedvector.IPackedVectorOfTPacked[uint16] = &bgrInterface
	bgrPacked.PackFromVector4(interfaceInput)
	check("packed.interface.bgr565", "PACKED_INTERFACE", "1055,0x00000000,0x3f020821,0x3f800000,0x3f800000", fmt.Sprintf("%d,%s", bgrPacked.PackedValue(), floatBits(bgrPacked.ToVector4().X, bgrPacked.ToVector4().Y, bgrPacked.ToVector4().Z, bgrPacked.ToVector4().W)))
	halfInterface := packedvector.HalfSingle{}
	var halfPacked packedvector.IPackedVectorOfTPacked[uint16] = &halfInterface
	halfPacked.PackFromVector4(interfaceInput)
	check("packed.interface.halfsingle", "PACKED_INTERFACE", "49152,0xc0000000,0x00000000,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", halfPacked.PackedValue(), floatBits(halfPacked.ToVector4().X, halfPacked.ToVector4().Y, halfPacked.ToVector4().Z, halfPacked.ToVector4().W)))
	normalizedByteInterface := packedvector.NormalizedByte2{}
	var normalizedBytePacked packedvector.IPackedVectorOfTPacked[uint16] = &normalizedByteInterface
	normalizedBytePacked.PackFromVector4(interfaceInput)
	check("packed.interface.normalizedbyte2", "PACKED_INTERFACE", "16513,0xbf800000,0x3f010204,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", normalizedBytePacked.PackedValue(), floatBits(normalizedBytePacked.ToVector4().X, normalizedBytePacked.ToVector4().Y, normalizedBytePacked.ToVector4().Z, normalizedBytePacked.ToVector4().W)))
	rgInterface := packedvector.Rg32{}
	var rgPacked packedvector.IPackedVectorOfTPacked[uint32] = &rgInterface
	rgPacked.PackFromVector4(interfaceInput)
	check("packed.interface.rg32", "PACKED_INTERFACE", "2147483648,0x00000000,0x3f000080,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", rgPacked.PackedValue(), floatBits(rgPacked.ToVector4().X, rgPacked.ToVector4().Y, rgPacked.ToVector4().Z, rgPacked.ToVector4().W)))
	shortInterface := packedvector.Short2{}
	var shortPacked packedvector.IPackedVectorOfTPacked[uint32] = &shortInterface
	shortPacked.PackFromVector4(interfaceInput)
	check("packed.interface.short2", "PACKED_INTERFACE", "65534,0xc0000000,0x00000000,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", shortPacked.PackedValue(), floatBits(shortPacked.ToVector4().X, shortPacked.ToVector4().Y, shortPacked.ToVector4().Z, shortPacked.ToVector4().W)))

	var packedValue64 packedvector.HalfVector4
	packedValue64.SetPackedValue(0xfedcba9876543210)
	check("packed.value64.hash-string", "PACKED_INTERFACE", "-2004318072,{X:0.1894531 Y:25920 Z:-0.8242188 W:-112384}", fmt.Sprintf("%d,%s", packedValue64.GetHashCode(), packedValue64.ToString()))
	packedValue64Copy := packedValue64
	packedValue64Copy.SetPackedValue(0)
	check("packed.value64.copy-and-equality", "PACKED_INTERFACE", "true,true,false,true", fmt.Sprintf("%t,%t,%t,%t", packedValue64.EqualsByObject(packedValue64), packedvector.HalfVector4OperatorEqualityByHalfVector4AndHalfVector4(packedValue64, packedValue64), packedValue64.EqualsByHalfVector4(packedValue64Copy), packedvector.HalfVector4OperatorInequalityByHalfVector4AndHalfVector4(packedValue64, packedValue64Copy)))

	// --- Foundation 17: pure managed CLR audio descriptors ------------------
	//
	// Read from Microsoft.Xna.Framework.dll IL
	// (sha256 38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130).
	// Every accessor is one ldfld/stfld plus UnsafeNativeStructures::
	// FlipHandedness, which copies X and Y and stores `neg` of Z.
	vector3Bits := func(v framework.Vector3) string { return floatBits(v.X, v.Y, v.Z) }

	// The constructor stores Vector3.Zero unflipped into _Position/_Velocity
	// while the getter flips, so both read back with a negative-zero Z. It
	// stores FlipHandedness of Vector3.Forward/Vector3.Up, so those two read
	// back bit-exactly as the source constants.
	listener := audio.NewAudioListener()
	check("audio-listener.default-position", "AUDIO_DESCRIPTOR", "0x00000000,0x00000000,0x80000000", vector3Bits(listener.Position()))
	check("audio-listener.default-velocity", "AUDIO_DESCRIPTOR", "0x00000000,0x00000000,0x80000000", vector3Bits(listener.Velocity()))
	check("audio-listener.default-forward", "AUDIO_DESCRIPTOR", vector3Bits(framework.Vector3Forward()), vector3Bits(listener.Forward()))
	check("audio-listener.default-up", "AUDIO_DESCRIPTOR", vector3Bits(framework.Vector3Up()), vector3Bits(listener.Up()))
	// The negative zero is invisible to Vector3 equality, exactly as in the
	// reference: only the sign bit distinguishes it.
	check("audio-listener.default-position-equals-zero", "AUDIO_DESCRIPTOR", "true", fmt.Sprintf("%t", listener.Position() == framework.Vector3Zero()))

	emitter := audio.NewAudioEmitter()
	check("audio-emitter.default-position", "AUDIO_DESCRIPTOR", "0x00000000,0x00000000,0x80000000", vector3Bits(emitter.Position()))
	check("audio-emitter.default-velocity", "AUDIO_DESCRIPTOR", "0x00000000,0x00000000,0x80000000", vector3Bits(emitter.Velocity()))
	check("audio-emitter.default-forward", "AUDIO_DESCRIPTOR", vector3Bits(framework.Vector3Forward()), vector3Bits(emitter.Forward()))
	check("audio-emitter.default-up", "AUDIO_DESCRIPTOR", vector3Bits(framework.Vector3Up()), vector3Bits(emitter.Up()))
	check("audio-emitter.default-doppler-scale", "AUDIO_DESCRIPTOR", bits(1), bits(emitter.DopplerScale()))

	// FlipHandedness is a sign-bit flip, so applying it on write and again on
	// read is a bit-exact identity for every binary32 value.
	negativeZero := float32(math.Copysign(0, -1))
	for _, roundTrip := range []struct {
		name  string
		value framework.Vector3
	}{
		{"ordinary", framework.Vector3{X: 1, Y: 2, Z: 3}},
		{"positive-zero", framework.Vector3Zero()},
		{"negative-zero", framework.Vector3{Z: negativeZero}},
		{"infinities", framework.Vector3{X: float32(math.Inf(-1)), Y: float32(math.Inf(1)), Z: float32(math.Inf(1))}},
		{"denormal", framework.Vector3{X: math.SmallestNonzeroFloat32, Y: -math.SmallestNonzeroFloat32, Z: math.SmallestNonzeroFloat32}},
		{"nan", framework.Vector3{X: float32(math.NaN()), Y: float32(math.NaN()), Z: float32(math.NaN())}},
	} {
		fresh := audio.NewAudioListener()
		fresh.SetPosition(roundTrip.value)
		fresh.SetForward(roundTrip.value)
		check("audio-listener.round-trip."+roundTrip.name, "AUDIO_DESCRIPTOR",
			vector3Bits(roundTrip.value)+"|"+vector3Bits(roundTrip.value),
			vector3Bits(fresh.Position())+"|"+vector3Bits(fresh.Forward()))
		freshEmitter := audio.NewAudioEmitter()
		freshEmitter.SetVelocity(roundTrip.value)
		freshEmitter.SetUp(roundTrip.value)
		check("audio-emitter.round-trip."+roundTrip.name, "AUDIO_DESCRIPTOR",
			vector3Bits(roundTrip.value)+"|"+vector3Bits(roundTrip.value),
			vector3Bits(freshEmitter.Velocity())+"|"+vector3Bits(freshEmitter.Up()))
	}

	// AudioEmitter::set_DopplerScale guards its store with `bge.un.s`, which
	// branches to the store on an ordered value >= 0 *and* on an unordered
	// comparison. So every NaN is accepted and only negative-ordered values
	// throw. The XNA behavior is a thrown ArgumentOutOfRangeException; the
	// projected Go behavior is a non-nil error on the setter alone, which is
	// why the acceptance/rejection column is PURE_XNA_DERIVED while the error
	// channel itself is a GO_LANGUAGE_PROJECTION.
	for _, dopplerCase := range []struct {
		name     string
		value    float32
		accepted bool
	}{
		{"positive", 2.5, true},
		{"one", 1, true},
		{"positive-denormal", math.SmallestNonzeroFloat32, true},
		{"largest-finite", math.MaxFloat32, true},
		{"positive-zero", 0, true},
		{"negative-zero", negativeZero, true},
		{"positive-infinity", float32(math.Inf(1)), true},
		{"nan", float32(math.NaN()), true},
		{"negative", -2.5, false},
		{"negative-denormal", -math.SmallestNonzeroFloat32, false},
		{"lowest-finite", -math.MaxFloat32, false},
		{"negative-infinity", float32(math.Inf(-1)), false},
	} {
		fresh := audio.NewAudioEmitter()
		err := fresh.SetDopplerScale(dopplerCase.value)
		stored := bits(dopplerCase.value)
		if !dopplerCase.accepted {
			stored = bits(1) // a rejected assignment leaves the default in place
		}
		check("audio-emitter.doppler-scale."+dopplerCase.name, "AUDIO_DESCRIPTOR",
			fmt.Sprintf("%t,%s", dopplerCase.accepted, stored),
			fmt.Sprintf("%t,%s", err == nil, bits(fresh.DopplerScale())))
		checkGoProjection("audio-emitter.doppler-scale.error-channel."+dopplerCase.name, "AUDIO_DESCRIPTOR",
			fmt.Sprintf("%t", !dopplerCase.accepted), fmt.Sprintf("%t", err != nil))
	}

	// CLR reference semantics survive the pure-managed classification: two
	// variables referencing one instance observe the same mutations.
	shared := audio.NewAudioListener()
	sharedAlias := shared
	sharedAlias.SetVelocity(framework.Vector3{X: 5, Y: 6, Z: 7})
	check("audio-listener.reference-semantics", "AUDIO_DESCRIPTOR", "0x40a00000,0x40c00000,0x40e00000", vector3Bits(shared.Velocity()))
	sharedEmitter := audio.NewAudioEmitter()
	sharedEmitterAlias := sharedEmitter
	_ = sharedEmitterAlias.SetDopplerScale(9)
	check("audio-emitter.reference-semantics", "AUDIO_DESCRIPTOR", bits(9), bits(sharedEmitter.DopplerScale()))

	// --- Foundation 18: projected CLR interface contracts -------------------
	//
	// These contracts have no implementation in CNA-Go, so there is no XNA
	// runtime behavior to observe. What is observable, and worth recording, is
	// the Go language projection: the contracts are satisfiable, the
	// per-operation fallibility split survives into the method set, and a
	// source result stays a separate channel from the error.
	//
	// The boundary each split comes from is measured from reference
	// implementor IL and is recorded in the api-compat report's Foundation-18
	// closures, not here.
	var matrices graphics.IEffectMatrices = &interfaceProbe{}
	matrices.SetWorld(framework.MatrixIdentity())
	matrices.SetView(framework.MatrixCreateTranslationBySingleAndSingleAndSingle(1, 2, 3))
	checkGoProjection("effect-matrices.infallible-round-trip", "INTERFACE_CONTRACT",
		"true,true", fmt.Sprintf("%t,%t",
			matrices.World() == framework.MatrixIdentity(),
			matrices.View() == framework.MatrixCreateTranslationBySingleAndSingleAndSingle(1, 2, 3)))

	fogProbe := &interfaceProbe{}
	var fog graphics.IEffectFog = fogProbe
	fog.SetFogEnabled(true)
	fog.SetFogStart(1)
	fog.SetFogEnd(2)
	fogColorError := fog.SetFogColor(framework.Vector3{X: 1, Y: 2, Z: 3})
	fogColor, fogColorReadError := fog.FogColor()
	checkGoProjection("effect-fog.managed-operations-are-infallible", "INTERFACE_CONTRACT",
		"true,0x3f800000,0x40000000", fmt.Sprintf("%t,%s,%s", fog.FogEnabled(), bits(fog.FogStart()), bits(fog.FogEnd())))
	checkGoProjection("effect-fog.runtime-operations-carry-an-error", "INTERFACE_CONTRACT",
		"true,true,0x3f800000,0x40000000,0x40400000",
		fmt.Sprintf("%t,%t,%s", fogColorError == nil, fogColorReadError == nil, vector3Bits(fogColor)))
	fogProbe.fogColorFailure = errors.New("D3DX HRESULT")
	_, failedRead := fog.FogColor()
	checkGoProjection("effect-fog.runtime-failure-reaches-the-caller", "INTERFACE_CONTRACT",
		"true,true", fmt.Sprintf("%t,%t", fog.SetFogColor(framework.Vector3{}) != nil, failedRead != nil))

	var component framework.IGameComponent = &interfaceProbe{}
	var deviceManagerContract framework.IGraphicsDeviceManager = &interfaceProbe{proceed: true}
	proceed, beginError := deviceManagerContract.BeginDraw()
	declining := &interfaceProbe{proceed: false}
	declined, declineError := framework.IGraphicsDeviceManager(declining).BeginDraw()
	checkGoProjection("graphics-device-manager-contract.begin-draw-channels", "INTERFACE_CONTRACT",
		"true,true,false,true", fmt.Sprintf("%t,%t,%t,%t", proceed, beginError == nil, declined, declineError == nil))
	checkGoProjection("game-component-contract.initialize-channel", "INTERFACE_CONTRACT",
		"true,true", fmt.Sprintf("%t,%t", component.Initialize() == nil, deviceManagerContract.CreateDevice() == nil))

	// --- Foundation 19: pure managed swap-chain descriptor ------------------
	//
	// Read from Microsoft.Xna.Framework.Graphics.dll IL
	// (sha256 560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55).
	// Every accessor is one ldflda into the nested Settings value struct plus
	// one ldfld or stfld; Bounds is computed; Clone copies the whole struct.
	parameters := graphics.NewPresentationParameters()
	// The constructor's whole body after the base call is
	// `this.IsFullScreen = true`, so a fresh descriptor is full-screen and
	// every other member takes its CLR zero.
	check("presentation-parameters.constructor-defaults", "PRESENTATION_PARAMETERS",
		"0,0,0,0,0,0,0,0,0,true",
		fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%t",
			parameters.BackBufferWidth(), parameters.BackBufferHeight(),
			parameters.BackBufferFormat(), parameters.DepthStencilFormat(),
			parameters.MultiSampleCount(), parameters.DisplayOrientation(),
			parameters.PresentationInterval(), parameters.RenderTargetUsage(),
			parameters.DeviceWindowHandle(), parameters.IsFullScreen()))
	// Bounds is computed from the two live extents and always originates at
	// zero, so a fresh descriptor reports a degenerate rectangle.
	defaultBounds := parameters.Bounds()
	check("presentation-parameters.default-bounds", "PRESENTATION_PARAMETERS", "0,0,0,0",
		fmt.Sprintf("%d,%d,%d,%d", defaultBounds.X, defaultBounds.Y, defaultBounds.Width, defaultBounds.Height))
	parameters.SetBackBufferWidth(1280)
	parameters.SetBackBufferHeight(720)
	trackedBounds := parameters.Bounds()
	check("presentation-parameters.bounds-tracks-extents", "PRESENTATION_PARAMETERS", "0,0,1280,720",
		fmt.Sprintf("%d,%d,%d,%d", trackedBounds.X, trackedBounds.Y, trackedBounds.Width, trackedBounds.Height))
	// Nothing validates: a descriptor stores what it is given.
	degenerate := graphics.NewPresentationParameters()
	degenerate.SetBackBufferWidth(-5)
	degenerate.SetMultiSampleCount(-7)
	degenerate.SetBackBufferFormat(graphics.SurfaceFormat(12345))
	degenerate.SetDisplayOrientation(framework.DisplayOrientation(1 << 20))
	check("presentation-parameters.stores-without-validating", "PRESENTATION_PARAMETERS",
		"-5,-7,12345,1048576,-5",
		fmt.Sprintf("%d,%d,%d,%d,%d", degenerate.BackBufferWidth(), degenerate.MultiSampleCount(),
			degenerate.BackBufferFormat(), degenerate.DisplayOrientation(), degenerate.Bounds().Width))

	// System.IntPtr projects to the pointer-sized bit value it carries. The
	// value is opaque: it is stored and returned verbatim, never dereferenced,
	// and it is no evidence that a window exists.
	handles := graphics.NewPresentationParameters()
	handleResults := make([]string, 0, 4)
	for _, handle := range []uintptr{0, 1, ^uintptr(0), 0xdeadbeef} {
		handles.SetDeviceWindowHandle(handle)
		handleResults = append(handleResults, fmt.Sprintf("%#x", handles.DeviceWindowHandle()))
	}
	check("presentation-parameters.device-window-handle-round-trip", "PRESENTATION_PARAMETERS",
		fmt.Sprintf("%#x,%#x,%#x,%#x", uintptr(0), uintptr(1), ^uintptr(0), uintptr(0xdeadbeef)),
		strings.Join(handleResults, ","))

	// Clone copies the whole Settings value struct into a fresh instance, so
	// the source's IsFullScreen overwrites the clone constructor's true and
	// the two share nothing afterwards.
	source := graphics.NewPresentationParameters()
	source.SetBackBufferWidth(800)
	source.SetDeviceWindowHandle(0x1234)
	source.SetIsFullScreen(false)
	clone := source.Clone()
	clone.SetBackBufferWidth(1)
	check("presentation-parameters.clone-is-independent", "PRESENTATION_PARAMETERS",
		"800,1,0x1234,false,false",
		fmt.Sprintf("%d,%d,%#x,%t,%t", source.BackBufferWidth(), clone.BackBufferWidth(),
			clone.DeviceWindowHandle(), clone.IsFullScreen(), source.IsFullScreen()))
	// The descriptor itself keeps CLR reference semantics; only Clone copies.
	aliased := graphics.NewPresentationParameters()
	aliasedReference := aliased
	aliasedReference.SetBackBufferHeight(1080)
	check("presentation-parameters.reference-semantics", "PRESENTATION_PARAMETERS", "1080",
		fmt.Sprintf("%d", aliased.BackBufferHeight()))

	// --- Foundation 20: read-only touch collection --------------------------
	//
	// Read from Microsoft.Xna.Framework.Input.Touch.dll IL
	// (sha256 b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25).
	// TouchCollection is a System.ValueType with eight inline TouchLocation
	// slots. Building one claims no touch capability: CNA-Go has no TouchPanel
	// and the caller supplies every location.
	touchSample := func(id int32, x, y float32) touch.TouchLocation {
		return touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(
			id, touch.TouchLocationStateMoved, framework.Vector2{X: x, Y: y})
	}
	touchSamples := []touch.TouchLocation{touchSample(1, 10, 20), touchSample(2, 30, 40)}
	touches, touchesError := touch.NewTouchCollection(touchSamples)
	check("touch-collection.constructor", "TOUCH_COLLECTION", "true,2,true,true",
		fmt.Sprintf("%t,%d,%t,%t", touchesError == nil, touches.Count(), touches.IsConnected(), touches.IsReadOnly()))

	// The constructor validates nil first and the eight-slot capacity second;
	// exactly eight is accepted because the reference tests `> 8`.
	_, nilError := touch.NewTouchCollection(nil)
	_, nineError := touch.NewTouchCollection(make([]touch.TouchLocation, 9))
	eight, eightError := touch.NewTouchCollection(make([]touch.TouchLocation, 8))
	check("touch-collection.constructor-guards", "TOUCH_COLLECTION", "true,true,true,8",
		fmt.Sprintf("%t,%t,%t,%d", nilError != nil, nineError != nil, eightError == nil, eight.Count()))

	// The indexer getter validates; every IList<T> mutator throws
	// unconditionally, without looking at its argument at all.
	_, lowIndexError := touches.Item(-1)
	_, highIndexError := touches.Item(2)
	touchFirst, touchFirstError := touches.Item(0)
	check("touch-collection.indexer", "TOUCH_COLLECTION", "true,true,true,1",
		fmt.Sprintf("%t,%t,%t,%d", lowIndexError != nil, highIndexError != nil, touchFirstError == nil, touchFirst.Id()))
	touchMutable := touches
	_, removeError := touchMutable.Remove(touchFirst)
	check("touch-collection.mutators-are-unsupported", "TOUCH_COLLECTION", "true,true,true,true,true,true,2",
		fmt.Sprintf("%t,%t,%t,%t,%t,%t,%d",
			touchMutable.Add(touchFirst) != nil, touchMutable.Clear() != nil,
			touchMutable.Insert(-99, touchFirst) != nil, touchMutable.RemoveAt(-99) != nil,
			removeError != nil, touchMutable.SetItem(0, touchFirst) != nil, touchMutable.Count()))

	// Search compares with the TouchLocation equality operator, which weighs
	// both state fields, so a location that Equals would accept is missed.
	sameIDDifferentState := touch.NewTouchLocationByInt32AndTouchLocationStateAndVector2(
		1, touch.TouchLocationStateReleased, framework.Vector2{X: 10, Y: 20})
	check("touch-collection.search-uses-operator-equality", "TOUCH_COLLECTION", "0,true,-1,false,true",
		fmt.Sprintf("%d,%t,%d,%t,%t",
			touches.IndexOf(touchFirst), touches.Contains(touchFirst),
			touches.IndexOf(sameIDDifferentState), touches.Contains(sameIDDifferentState),
			touchFirst.EqualsByTouchLocation(sameIDDifferentState)))

	// FindById compares identifiers only and yields a zero location on a miss.
	foundById, touchById := touches.FindById(2)
	touchMissedById, touchMissed := touches.FindById(1234)
	check("touch-collection.find-by-id", "TOUCH_COLLECTION", "true,2,false,0,0",
		fmt.Sprintf("%t,%d,%t,%d,%d", foundById, touchById.Id(), touchMissedById, touchMissed.Id(), int32(touchMissed.State())))

	// CopyTo validates a nil touchDestination, a negative start, and an
	// insufficient touchDestination, the last in 64-bit arithmetic.
	touchDestination := make([]touch.TouchLocation, 4)
	check("touch-collection.copy-to-guards", "TOUCH_COLLECTION", "true,true,true,true",
		fmt.Sprintf("%t,%t,%t,%t",
			touches.CopyTo(nil, 0) != nil, touches.CopyTo(touchDestination, -1) != nil,
			touches.CopyTo(touchDestination, 3) != nil, touches.CopyTo(touchDestination, 1<<31-1) != nil))
	touchCopyError := touches.CopyTo(touchDestination, 2)
	check("touch-collection.copy-to", "TOUCH_COLLECTION", "true,0,0,1,2",
		fmt.Sprintf("%t,%d,%d,%d,%d", touchCopyError == nil,
			touchDestination[0].Id(), touchDestination[1].Id(), touchDestination[2].Id(), touchDestination[3].Id()))

	// The touchCursor starts before the first element and clamps on exhaustion, so
	// Current reports an error at both ends.
	touchCursor := touches.GetEnumerator()
	_, beforeStart := touchCursor.Current()
	touchVisited := make([]string, 0, 2)
	for touchCursor.MoveNext() {
		location, err := touchCursor.Current()
		touchVisited = append(touchVisited, fmt.Sprintf("%d/%t", location.Id(), err == nil))
	}
	afterEnd := touchCursor.MoveNext()
	_, afterEndError := touchCursor.Current()
	check("touch-collection.enumerator", "TOUCH_COLLECTION", "true,1/true,2/true,false,true",
		fmt.Sprintf("%t,%s,%t,%t", beforeStart != nil, strings.Join(touchVisited, ","), afterEnd, afterEndError != nil))

	// Value semantics: a copy is an equal touchSnapshot, and the touchCursor touchCaptured a
	// copy rather than following the source variable.
	touchSnapshot := touches
	touchCapturedCursor := touches.GetEnumerator()
	touches, _ = touch.NewTouchCollection([]touch.TouchLocation{touchSample(7, 7, 7), touchSample(8, 8, 8), touchSample(9, 9, 9)})
	touchCaptured := 0
	for touchCapturedCursor.MoveNext() {
		touchCaptured++
	}
	check("touch-collection.value-semantics", "TOUCH_COLLECTION", "true,2,3,2",
		fmt.Sprintf("%t,%d,%d,%d", touchSnapshot == touchMutable, touchSnapshot.Count(), touches.Count(), touchCaptured))

	// --- Foundation 21: pure managed service registry -----------------------
	//
	// Read from Microsoft.Xna.Framework.Game.dll IL
	// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0).
	// GameServiceContainer is one private Dictionary<Type, object>. Completing
	// it wires nothing up: CNA-Go's Game exposes no Services property, so
	// nothing in the binding populates or consults a container.
	services := framework.NewGameServiceContainer()
	serviceKey := reflect.TypeOf((*serviceCorpusProbe)(nil)).Elem()
	unrelatedKey := reflect.TypeOf(0)
	registered := &serviceCorpusProvider{}

	missingBefore, missingBeforeError := services.GetService(serviceKey)
	// A missing service is an absence, not a failure: `ldnull; ret`.
	check("game-service-container.missing-is-not-a-failure", "GAME_SERVICE_CONTAINER", "true,true",
		fmt.Sprintf("%t,%t", missingBefore == nil, missingBeforeError == nil))
	// Removing a service that was never registered is not an error either:
	// the reference discards the dictionary's Boolean result.
	check("game-service-container.absent-removal-is-not-a-failure", "GAME_SERVICE_CONTAINER", "true",
		fmt.Sprintf("%t", services.RemoveService(serviceKey) == nil))
	// All three methods reject a nil service type, and AddService also rejects
	// a nil provider.
	_, nilLookupError := services.GetService(nil)
	check("game-service-container.nil-arguments", "GAME_SERVICE_CONTAINER", "true,true,true,true",
		fmt.Sprintf("%t,%t,%t,%t",
			services.AddService(nil, registered) != nil,
			services.AddService(serviceKey, nil) != nil,
			services.RemoveService(nil) != nil,
			nilLookupError != nil))

	addError := services.AddService(serviceKey, registered)
	resolved, resolvedError := services.GetService(serviceKey)
	check("game-service-container.add-and-resolve", "GAME_SERVICE_CONTAINER", "true,true,true",
		fmt.Sprintf("%t,%t,%t", addError == nil, resolvedError == nil, resolved == any(registered)))
	// The duplicate check runs before the assignability check, so an
	// unassignable provider for an already-registered type reports the
	// duplicate.
	duplicateError := services.AddService(serviceKey, registered)
	unassignableError := services.AddService(unrelatedKey, registered)
	check("game-service-container.rejections", "GAME_SERVICE_CONTAINER", "true,true,true",
		fmt.Sprintf("%t,%t,%t", duplicateError != nil, unassignableError != nil,
			strings.Contains(fmt.Sprint(services.AddService(serviceKey, "not a probe")), "already contains")))

	// Reference semantics: two variables naming one container share
	// registrations, and a separate container shares nothing.
	servicesAlias := services
	_ = servicesAlias.RemoveService(serviceKey)
	afterAliasRemoval, _ := services.GetService(serviceKey)
	separate, _ := framework.NewGameServiceContainer().GetService(serviceKey)
	check("game-service-container.reference-semantics", "GAME_SERVICE_CONTAINER", "true,true",
		fmt.Sprintf("%t,%t", afterAliasRemoval == nil, separate == nil))

	// --- Foundation 22/23: the event architecture and its first contracts ---
	//
	// Read from Microsoft.Xna.Framework.Game.dll IL
	// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0).
	// GameComponent and DrawableGameComponent are the shipped implementors of
	// IUpdateable and IDrawable. Their event accessors are the standard
	// compiler-generated Delegate.Combine/Delegate.Remove pair guarded by
	// Interlocked.CompareExchange, and every raise site pushes the shared
	// System.EventArgs::Empty static field.
	//
	// Completing the two contracts wires nothing up: CNA-Go has no
	// GameComponent, no component collection, and no component loop.

	// EventArgs.Empty is one shared identity, and it is not nil.
	check("event-args.empty-is-one-shared-identity", "EVENT_ARCHITECTURE", "true,true",
		fmt.Sprintf("%t,%t", framework.EventArgsEmpty() == framework.EventArgsEmpty(), framework.EventArgsEmpty() != nil))

	var corpusEvent framework.EventSource[*framework.EventArgs]
	var corpusOrder []string
	corpusSender := &corpusComponent{}

	// Zero handlers is not a failure.
	check("event-source.raise-with-no-handlers", "EVENT_ARCHITECTURE", "true",
		fmt.Sprintf("%t", corpusEvent.Raise(corpusSender, framework.EventArgsEmpty()) == nil))

	corpusHandler := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, args *framework.EventArgs) error {
			corpusOrder = append(corpusOrder, name)
			if sender != any(corpusSender) || args != framework.EventArgsEmpty() {
				corpusOrder = append(corpusOrder, "identity-lost")
			}
			return nil
		}
	}
	sharedHandler := corpusHandler("shared")
	firstToken, _ := corpusEvent.Add(sharedHandler)
	secondToken, _ := corpusEvent.Add(sharedHandler)
	_, _ = corpusEvent.Add(corpusHandler("third"))
	_ = corpusEvent.Raise(corpusSender, framework.EventArgsEmpty())
	// Registration order is preserved, and one handler value added twice is two
	// registrations, because a token names the registration rather than the
	// handler. That last part is the Go language projection.
	check("event-source.registration-order", "EVENT_ARCHITECTURE", "shared,shared,third",
		strings.Join(corpusOrder, ","))
	checkGoProjection("event-source.duplicate-registrations-are-independent", "EVENT_ARCHITECTURE", "true",
		fmt.Sprintf("%t", firstToken != secondToken))

	// Removing one duplicate leaves the other; removing again, removing the
	// zero token, and removing a token owned by another event are all harmless,
	// mirroring Delegate.Remove of an absent delegate.
	var foreignEvent framework.EventSource[*framework.EventArgs]
	foreignToken, _ := foreignEvent.Add(corpusHandler("foreign"))
	corpusOrder = nil
	removals := []error{
		corpusEvent.Remove(firstToken),
		corpusEvent.Remove(firstToken),
		corpusEvent.Remove(framework.EventSubscription{}),
		corpusEvent.Remove(foreignToken),
	}
	_ = corpusEvent.Raise(corpusSender, framework.EventArgsEmpty())
	harmless := true
	for _, err := range removals {
		harmless = harmless && err == nil
	}
	check("event-source.absent-removal-is-harmless", "EVENT_ARCHITECTURE", "true,shared,third",
		fmt.Sprintf("%t,%s", harmless, strings.Join(corpusOrder, ",")))
	_ = secondToken

	// A nil handler registers nothing, mirroring Delegate.Combine(existing, null).
	nilToken, nilError := corpusEvent.Add(nil)
	corpusOrder = nil
	_ = corpusEvent.Raise(corpusSender, framework.EventArgsEmpty())
	check("event-source.nil-handler-registers-nothing", "EVENT_ARCHITECTURE", "true,true,shared,third",
		fmt.Sprintf("%t,%t,%s", nilError == nil, nilToken == framework.EventSubscription{}, strings.Join(corpusOrder, ",")))

	// The first handler failure propagates, stops the dispatch, and leaves the
	// registration list intact.
	var failingEvent framework.EventSource[*framework.EventArgs]
	corpusFailure := errors.New("corpus handler failure")
	corpusOrder = nil
	_, _ = failingEvent.Add(func(sender any, args *framework.EventArgs) error {
		corpusOrder = append(corpusOrder, "first")
		return nil
	})
	_, _ = failingEvent.Add(func(sender any, args *framework.EventArgs) error {
		corpusOrder = append(corpusOrder, "failing")
		return corpusFailure
	})
	_, _ = failingEvent.Add(func(sender any, args *framework.EventArgs) error {
		corpusOrder = append(corpusOrder, "never")
		return nil
	})
	raiseError := failingEvent.Raise(corpusSender, framework.EventArgsEmpty())
	firstPass := strings.Join(corpusOrder, ",")
	corpusOrder = nil
	secondError := failingEvent.Raise(corpusSender, framework.EventArgsEmpty())
	check("event-source.failure-stops-dispatch-and-keeps-the-list", "EVENT_ARCHITECTURE",
		"true,first,failing,true,first,failing",
		fmt.Sprintf("%t,%s,%t,%s", errors.Is(raiseError, corpusFailure), firstPass,
			errors.Is(secondError, corpusFailure), strings.Join(corpusOrder, ",")))

	// Mutation during a raise affects later raises, never the running snapshot.
	var snapshotEvent framework.EventSource[*framework.EventArgs]
	corpusOrder = nil
	var lateToken framework.EventSubscription
	_, _ = snapshotEvent.Add(func(sender any, args *framework.EventArgs) error {
		corpusOrder = append(corpusOrder, "mutator")
		lateToken, _ = snapshotEvent.Add(func(sender any, args *framework.EventArgs) error {
			corpusOrder = append(corpusOrder, "late")
			return nil
		})
		return nil
	})
	_ = snapshotEvent.Raise(corpusSender, framework.EventArgsEmpty())
	duringRaise := strings.Join(corpusOrder, ",")
	corpusOrder = nil
	_ = snapshotEvent.Raise(corpusSender, framework.EventArgsEmpty())
	check("event-source.dispatch-uses-a-snapshot", "EVENT_ARCHITECTURE", "mutator,mutator,late",
		fmt.Sprintf("%s,%s", duringRaise, strings.Join(corpusOrder, ",")))
	_ = lateToken

	// The two component contracts are satisfiable, infallible, and raise
	// through their projected accessors. set_Enabled raises only when the
	// stored value actually changes, which is what the reference IL does.
	var corpusUpdateable framework.IUpdateable = corpusSender
	var corpusDrawable framework.IDrawable = corpusSender
	corpusOrder = nil
	componentToken, componentAddError := corpusUpdateable.AddEnabledChangedHandler(
		func(sender any, args *framework.EventArgs) error {
			corpusOrder = append(corpusOrder, fmt.Sprintf("enabled=%t", sender.(*corpusComponent).Enabled()))
			return nil
		})
	changed := corpusSender.SetEnabled(true)
	unchanged := corpusSender.SetEnabled(true)
	corpusUpdateable.Update(framework.NewGameTimeByNone())
	corpusDrawable.Draw(framework.NewGameTimeByNone())
	check("component-contracts.raise-only-on-change", "EVENT_ARCHITECTURE", "true,true,true,enabled=true,1,1",
		fmt.Sprintf("%t,%t,%t,%s,%d,%d", componentAddError == nil, changed == nil, unchanged == nil,
			strings.Join(corpusOrder, ","), corpusSender.updates, corpusSender.draws))
	check("component-contracts.removal-through-the-interface", "EVENT_ARCHITECTURE", "true",
		fmt.Sprintf("%t", corpusUpdateable.RemoveEnabledChangedHandler(componentToken) == nil))

	// The three System.EventArgs carriers store what they are given and
	// validate nothing, and none of them is assignable to its CLR base.
	corpusCarrier := framework.NewGameComponentCollectionEventArgs(corpusSender)
	var carrierAsAny any = corpusCarrier
	_, carrierIsBase := carrierAsAny.(*framework.EventArgs)
	check("event-args-carriers.store-without-validating", "EVENT_ARCHITECTURE", "true,true,false",
		fmt.Sprintf("%t,%t,%t",
			corpusCarrier.GameComponent() == framework.IGameComponent(corpusSender),
			framework.NewGameComponentCollectionEventArgs(nil).GameComponent() == nil,
			carrierIsBase))

	// ---------------------------------------------------------------------
	// Foundation 26. The BCL base-class composition projection, proved by
	// GameComponentCollection over Collection<IGameComponent>.
	//
	// Behavior is PURE_XNA_DERIVED from two pinned binaries: the four
	// overrides from Microsoft.Xna.Framework.Game.dll, and the inherited
	// Collection<T>/List<T> surface from mscorlib 4.0.30319.1, the .NET
	// Framework 4.0 RTM assembly every pinned XNA assembly binds against.
	// ---------------------------------------------------------------------

	components := framework.NewGameComponentCollection()
	var collectionOrder []string
	collectionHandler := func(name string) framework.EventHandler[*framework.GameComponentCollectionEventArgs] {
		return func(sender any, args *framework.GameComponentCollectionEventArgs) error {
			// The sender is the collection itself and the args are freshly
			// allocated per raise, so both identities are recorded rather
			// than assumed.
			label := "<nil>"
			if args.GameComponent() != nil {
				label = "component"
			}
			owner, senderIsCollection := sender.(*framework.GameComponentCollection)
			if !senderIsCollection {
				collectionOrder = append(collectionOrder, name+":sender-lost")
				return nil
			}
			collectionOrder = append(collectionOrder, fmt.Sprintf("%s:%s:count=%d", name, label, owner.Count()))
			return nil
		}
	}
	_, addedHandlerError := components.AddComponentAddedHandler(collectionHandler("added"))
	_, removedHandlerError := components.AddComponentRemovedHandler(collectionHandler("removed"))

	// A fresh collection is empty and its inherited store is mutable.
	check("game-component-collection.starts-empty", "BCL_BASE_COMPOSITION", "0,true,true",
		fmt.Sprintf("%d,%t,%t", components.Count(), addedHandlerError == nil, removedHandlerError == nil))

	firstComponent, secondComponent := &corpusComponent{}, &corpusComponent{}

	// Add is InsertItem(Count, item): it mutates and only THEN announces, so
	// the handler already sees the new count.
	collectionOrder = nil
	addFirst := components.Add(firstComponent)
	addSecond := components.Add(secondComponent)
	check("game-component-collection.add-mutates-before-it-announces", "BCL_BASE_COMPOSITION",
		"true,true,2,added:component:count=1,added:component:count=2",
		fmt.Sprintf("%t,%t,%d,%s", addFirst == nil, addSecond == nil, components.Count(), strings.Join(collectionOrder, ",")))

	// The duplicate test runs before the insertion, so a rejected duplicate
	// neither mutates nor announces.
	collectionOrder = nil
	duplicate := components.Add(firstComponent)
	check("game-component-collection.duplicate-is-rejected-without-effect", "BCL_BASE_COMPOSITION",
		"true,2,",
		fmt.Sprintf("%t,%d,%s", duplicate != nil, components.Count(), strings.Join(collectionOrder, ",")))

	// The inherited readers.
	firstIndex := components.IndexOf(firstComponent)
	indexedComponent, indexError := components.Item(1)
	outOfRange := func() bool { _, err := components.Item(2); return err != nil }()
	check("game-component-collection.inherited-readers", "BCL_BASE_COMPOSITION", "0,true,true,true,true,-1",
		fmt.Sprintf("%d,%t,%t,%t,%t,%d", firstIndex, indexError == nil,
			indexedComponent == framework.IGameComponent(secondComponent), outOfRange,
			components.Contains(firstComponent), components.IndexOf(&corpusComponent{})))

	// The indexer setter always fails, and reports the range failure first.
	inRangeAssignment := components.SetItemProperty(0, secondComponent)
	outOfRangeAssignment := components.SetItemProperty(9, secondComponent)
	directOverride := components.SetItemMethod(0, secondComponent)
	check("game-component-collection.assignment-is-never-supported", "BCL_BASE_COMPOSITION", "true,true,true,2",
		fmt.Sprintf("%t,%t,%t,%d", inRangeAssignment != nil, outOfRangeAssignment != nil,
			directOverride != nil, components.Count()))

	// Enumeration is source order, and every mutation invalidates it.
	componentIterator := components.GetEnumerator()
	firstYield, firstOK, _ := componentIterator.Next()
	_ = components.Add(&corpusComponent{})
	_, staleOK, staleError := componentIterator.Next()
	check("game-component-collection.enumeration-fails-fast", "BCL_BASE_COMPOSITION", "true,true,false,true",
		fmt.Sprintf("%t,%t,%t,%t", firstOK, firstYield == framework.IGameComponent(firstComponent),
			staleOK, staleError != nil))

	// Remove mutates before it announces; removing an absent component
	// changes nothing.
	collectionOrder = nil
	removed, removeError := components.Remove(firstComponent)
	absent, absentError := components.Remove(&corpusComponent{})
	check("game-component-collection.remove-mutates-before-it-announces", "BCL_BASE_COMPOSITION",
		"true,true,false,true,2,removed:component:count=2",
		fmt.Sprintf("%t,%t,%t,%t,%d,%s", removed, removeError == nil, absent, absentError == nil,
			components.Count(), strings.Join(collectionOrder, ",")))

	// Clear announces the WHOLE collection before it mutates, which is the
	// opposite order from Insert and Remove.
	collectionOrder = nil
	clearError := components.Clear()
	check("game-component-collection.clear-announces-before-it-mutates", "BCL_BASE_COMPOSITION",
		"true,0,removed:component:count=2,removed:component:count=2",
		fmt.Sprintf("%t,%d,%s", clearError == nil, components.Count(), strings.Join(collectionOrder, ",")))

	// A nil component is stored and announces nothing on insert, is announced
	// by Clear, and duplicates as a nil.
	nilBearing := framework.NewGameComponentCollection()
	collectionOrder = nil
	_, _ = nilBearing.AddComponentAddedHandler(collectionHandler("added"))
	_, _ = nilBearing.AddComponentRemovedHandler(collectionHandler("removed"))
	nilAdd := nilBearing.Add(nil)
	nilDuplicate := nilBearing.Add(nil)
	afterInsert := strings.Join(collectionOrder, ",")
	collectionOrder = nil
	nilClear := nilBearing.Clear()
	check("game-component-collection.nil-component-asymmetry", "BCL_BASE_COMPOSITION",
		"true,true,,true,removed:<nil>:count=1",
		fmt.Sprintf("%t,%t,%s,%t,%s", nilAdd == nil, nilDuplicate != nil, afterInsert,
			nilClear == nil, strings.Join(collectionOrder, ",")))

	// Clearing an already empty collection still invalidates enumerators,
	// because List<T>.Clear increments its version unconditionally.
	emptyCollection := framework.NewGameComponentCollection()
	emptyIterator := emptyCollection.GetEnumerator()
	emptyClear := emptyCollection.Clear()
	_, emptyOK, emptyError := emptyIterator.Next()
	check("game-component-collection.empty-clear-still-invalidates", "BCL_BASE_COMPOSITION", "true,false,true",
		fmt.Sprintf("%t,%t,%t", emptyClear == nil, emptyOK, emptyError != nil))

	// Insert admits Count itself; RemoveAt and the indexer setter do not.
	guards := framework.NewGameComponentCollection()
	_ = guards.Add(&corpusComponent{})
	insertAtCount := guards.Insert(1, &corpusComponent{})
	insertPastCount := guards.Insert(3, &corpusComponent{})
	insertNegative := guards.Insert(-1, &corpusComponent{})
	removeAtCount := guards.RemoveAt(2)
	check("game-component-collection.index-guards-differ-per-operation", "BCL_BASE_COMPOSITION",
		"true,true,true,true,2",
		fmt.Sprintf("%t,%t,%t,%t,%d", insertAtCount == nil, insertPastCount != nil,
			insertNegative != nil, removeAtCount != nil, guards.Count()))

	// CopyTo reports Array.Copy's three failures and hands out a copy.
	copyTarget := make([]framework.IGameComponent, 3)
	nilDestination := guards.CopyTo(nil, 0)
	negativeIndex := guards.CopyTo(copyTarget, -1)
	tooSmall := guards.CopyTo(copyTarget, 2)
	copied := guards.CopyTo(copyTarget, 1)
	copyTarget[1] = nil
	survivor, _ := guards.Item(0)
	check("game-component-collection.copyto-validates-and-copies", "BCL_BASE_COMPOSITION",
		"true,true,true,true,true",
		fmt.Sprintf("%t,%t,%t,%t,%t", nilDestination != nil, negativeIndex != nil, tooSmall != nil,
			copied == nil, survivor != nil))

	// A handler failure leaves an applied mutation applied and an unapplied
	// Clear unapplied. That is the same asymmetry, observed through failure.
	failingCollection := framework.NewGameComponentCollection()
	collectionFailure := errors.New("corpus collection handler failure")
	_, _ = failingCollection.AddComponentAddedHandler(
		func(any, *framework.GameComponentCollectionEventArgs) error { return collectionFailure })
	failedAdd := failingCollection.Add(&corpusComponent{})
	clearedCollection := framework.NewGameComponentCollection()
	_ = clearedCollection.Add(&corpusComponent{})
	_, _ = clearedCollection.AddComponentRemovedHandler(
		func(any, *framework.GameComponentCollectionEventArgs) error { return collectionFailure })
	failedClear := clearedCollection.Clear()
	check("game-component-collection.handler-failure-preserves-the-mutation-order", "BCL_BASE_COMPOSITION",
		"true,1,true,1",
		fmt.Sprintf("%t,%d,%t,%d", errors.Is(failedAdd, collectionFailure), failingCollection.Count(),
			errors.Is(failedClear, collectionFailure), clearedCollection.Count()))

	// ---------------------------------------------------------------------
	// Foundation 27. ReadOnlyCollection<T> as a signature-position BCL
	// projection, proved by Media.VisualizationData.
	// ---------------------------------------------------------------------

	visualization := media.NewVisualizationData()
	frequencies, samples := visualization.Frequencies(), visualization.Samples()

	// The constructor allocates two 256-element Single arrays and wraps each.
	// Both getters are one ldfld, so each returns the same stored view every
	// time and the two views are distinct.
	check("visualization-data.two-stable-distinct-256-buffers", "BCL_SIGNATURE_ADAPTER", "256,256,true,true,false",
		fmt.Sprintf("%d,%d,%t,%t,%t", frequencies.Count(), samples.Count(),
			visualization.Frequencies() == frequencies, visualization.Samples() == samples,
			any(frequencies) == any(samples)))

	// Both buffers start as zeros; the constructor writes nothing.
	firstFrequency, frequencyError := frequencies.Item(0)
	lastSample, sampleError := samples.Item(0xFF)
	pastEnd := func() bool { _, err := samples.Item(0x100); return err != nil }()
	negative := func() bool { _, err := samples.Item(-1); return err != nil }()
	check("visualization-data.buffers-start-zeroed-and-bounded", "BCL_SIGNATURE_ADAPTER", "0,true,0,true,true,true",
		fmt.Sprintf("%v,%t,%v,%t,%t,%t", firstFrequency, frequencyError == nil,
			lastSample, sampleError == nil, pastEnd, negative))

	// ReadOnlyCollection<T> stores the list rather than copying it, so the
	// view is LIVE. CNA-Go has no media backend, so the corpus plays that part
	// by copying out, checking, and relying on the view reporting the current
	// contents rather than a snapshot taken at construction.
	visualizationCopy := make([]float32, 0x100)
	visualizationCopyError := frequencies.CopyTo(visualizationCopy, 0)
	visualizationCopy[0] = 1
	afterCopyMutation, _ := frequencies.Item(0)
	check("read-only-collection.copyto-hands-out-a-copy", "BCL_SIGNATURE_ADAPTER", "true,0",
		fmt.Sprintf("%t,%v", visualizationCopyError == nil, afterCopyMutation))

	// CopyTo carries Array.Copy's three failures in Array.Copy's order.
	viewNilDestination := frequencies.CopyTo(nil, 0)
	viewNegativeIndex := frequencies.CopyTo(visualizationCopy, -1)
	viewTooSmall := frequencies.CopyTo(visualizationCopy, 1)
	check("read-only-collection.copyto-validates", "BCL_SIGNATURE_ADAPTER", "true,true,true",
		fmt.Sprintf("%t,%t,%t", viewNilDestination != nil, viewNegativeIndex != nil, viewTooSmall != nil))

	// The whole buffer enumerates in source order, and an array-backed view is
	// NOT version-checked: SZGenericArrayEnumerator<T> holds only _array,
	// _index and _endIndex and compares the index against the end.
	visualizationIterator := samples.GetEnumerator()
	enumerated, enumerationFailed := 0, false
	for {
		_, ok, err := visualizationIterator.Next()
		if err != nil {
			enumerationFailed = true
			break
		}
		if !ok {
			break
		}
		enumerated++
	}
	check("read-only-collection.array-backed-enumeration-is-complete-and-unchecked", "BCL_SIGNATURE_ADAPTER",
		"256,false", fmt.Sprintf("%d,%t", enumerated, enumerationFailed))

	// Element equality is EqualityComparer<Single>.Default, which reaches
	// System.Single::Equals, so a NaN search finds a NaN element. Go's ==
	// would report false, which is why the comparer is projected rather than
	// inherited from the language.
	nanBuffer := []float32{1, float32(math.NaN()), 3}
	nanView := framework.NewReadOnlyCollectionOverSingles(nanBuffer)
	check("read-only-collection.single-comparer-treats-nan-as-equal", "BCL_SIGNATURE_ADAPTER", "1,true,2,-1",
		fmt.Sprintf("%d,%t,%d,%d", nanView.IndexOf(float32(math.NaN())),
			nanView.Contains(float32(math.NaN())), nanView.IndexOf(3), nanView.IndexOf(9)))

	// The view is live over the array it was handed: a later element write is
	// visible, while pointing the owner's own variable at a different array is
	// not, because the CLR view holds the array reference it was given.
	liveBuffer := []float32{1, 2, 3}
	liveView := framework.NewReadOnlyCollectionOverSingles(liveBuffer)
	liveBuffer[1] = 99
	liveElement, _ := liveView.Item(1)
	liveBuffer = []float32{7, 8}
	_ = liveBuffer
	check("read-only-collection.live-over-the-list-it-was-given", "BCL_SIGNATURE_ADAPTER", "99,3",
		fmt.Sprintf("%v,%d", liveElement, liveView.Count()))

	// ---------------------------------------------------------------------
	// Foundation 28. IGraphicsDeviceService, the device-publication contract.
	//
	// Its one non-event operation is infallible because the reference
	// implementor's accessor is `ldarg.0; ldfld device; ret` -- a stored
	// reference handed over, null before a device exists.
	// ---------------------------------------------------------------------

	var deviceServiceContract graphics.IGraphicsDeviceService = &corpusDeviceService{}
	check("graphics-device-service.accessor-hands-over-a-stored-reference", "EVENT_ARCHITECTURE", "true",
		fmt.Sprintf("%t", deviceServiceContract.GraphicsDevice() == nil))

	corpusService := &corpusDeviceService{}
	deviceServiceContract = corpusService
	var deviceOrder []string
	deviceHandler := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, args *framework.EventArgs) error {
			if sender != any(corpusService) || args != framework.EventArgsEmpty() {
				deviceOrder = append(deviceOrder, name+":identity-lost")
				return nil
			}
			deviceOrder = append(deviceOrder, name)
			return nil
		}
	}
	createdToken, createdError := deviceServiceContract.AddDeviceCreatedHandler(deviceHandler("created"))
	_, resettingError := deviceServiceContract.AddDeviceResettingHandler(deviceHandler("resetting"))
	_, resetError := deviceServiceContract.AddDeviceResetHandler(deviceHandler("reset"))
	_, disposingError := deviceServiceContract.AddDeviceDisposingHandler(deviceHandler("disposing"))
	corpusService.raiseAll()
	check("graphics-device-service.four-independent-events", "EVENT_ARCHITECTURE",
		"true,true,true,true,created,resetting,reset,disposing",
		fmt.Sprintf("%t,%t,%t,%t,%s", createdError == nil, resettingError == nil,
			resetError == nil, disposingError == nil, strings.Join(deviceOrder, ",")))

	deviceOrder = nil
	removalError := deviceServiceContract.RemoveDeviceCreatedHandler(createdToken)
	corpusService.raiseAll()
	check("graphics-device-service.removal-affects-one-event-only", "EVENT_ARCHITECTURE",
		"true,resetting,reset,disposing",
		fmt.Sprintf("%t,%s", removalError == nil, strings.Join(deviceOrder, ",")))

	// ------------------------------------------------------------------
	// Foundation 30. Game's two managed CLR objects.
	//
	// Microsoft.Xna.Framework.Game::get_Components and ::get_Services are one
	// `ldfld` each over fields the constructor assigns exactly once, so the
	// observable claims are identity, stability and reference semantics. The
	// component engine those two feed becomes observable in the next group,
	// through the explicit base calls.
	// ------------------------------------------------------------------
	corpusGame, corpusGameError := framework.NewGame(corpusCallbacks{})

	check("game-managed-state.constructor-allocates-both", "GAME_MANAGED_STATE", "true,true,true,0",
		fmt.Sprintf("%t,%t,%t,%d", corpusGameError == nil,
			corpusGame.Components() != nil, corpusGame.Services() != nil,
			corpusGame.Components().Count()))

	// One `ldfld` cannot allocate, so repeated reads are the same identity.
	check("game-managed-state.getters-are-stable-identities", "GAME_MANAGED_STATE", "true,true",
		fmt.Sprintf("%t,%t", corpusGame.Components() == corpusGame.Components(),
			corpusGame.Services() == corpusGame.Services()))

	// Two Games own two distinct sets of managed state.
	otherGame, _ := framework.NewGame(corpusCallbacks{})
	check("game-managed-state.each-game-owns-its-own", "GAME_MANAGED_STATE", "true,true",
		fmt.Sprintf("%t,%t", corpusGame.Components() != otherGame.Components(),
			corpusGame.Services() != otherGame.Services()))

	// CLR reference semantics: a mutation through the returned object is the
	// Game's own state, and every later read sees it.
	sharedComponent := &corpusComponent{}
	sharedAdd := corpusGame.Components().Add(sharedComponent)
	sharedService := corpusGame.Services().AddService(reflect.TypeOf((*serviceCorpusProbe)(nil)).Elem(), &serviceCorpusProvider{})
	resolved, resolveError := corpusGame.Services().GetService(reflect.TypeOf((*serviceCorpusProbe)(nil)).Elem())
	check("game-managed-state.mutation-through-the-getter-is-visible", "GAME_MANAGED_STATE", "true,true,1,true,true,true",
		fmt.Sprintf("%t,%t,%d,%t,%t,%t", sharedAdd == nil, sharedService == nil,
			corpusGame.Components().Count(), corpusGame.Components().Contains(sharedComponent),
			resolveError == nil, resolved != nil))

	// The Game constructor subscribes its own two handlers FIRST, so a
	// consumer's handler always runs after the engine has tracked the change.
	// The proof from outside is the order alone: the consumer's handler is the
	// only one it can see, and it must be the last to run.
	var subscriptionOrder []string
	_, _ = corpusGame.Components().AddComponentAddedHandler(func(sender any, args *framework.GameComponentCollectionEventArgs) error {
		subscriptionOrder = append(subscriptionOrder, "consumer")
		return nil
	})
	orderComponent := &corpusComponent{}
	orderAdd := corpusGame.Components().Add(orderComponent)
	check("game-managed-state.consumer-handler-runs-after-the-engine", "GAME_MANAGED_STATE", "true,consumer",
		fmt.Sprintf("%t,%s", orderAdd == nil, strings.Join(subscriptionOrder, ",")))

	// ------------------------------------------------------------------
	// Foundation 31. The explicit Game base-call family.
	//
	// These observations are what an outside caller can see, so they are the
	// component engine's real qualification: the ordering, the snapshot, the
	// Enabled/Visible gates and the mutation semantics all become visible
	// through the base calls and through nothing else.
	// ------------------------------------------------------------------
	var engineLog []string
	engineGame, _ := framework.NewGame(corpusCallbacks{})
	addEngineComponent := func(name string, updateOrder, drawOrder int32) *orderedCorpusComponent {
		component := newOrderedCorpusComponent(name, &engineLog, updateOrder, drawOrder)
		_ = engineGame.Components().Add(component)
		return component
	}
	// Added in an order that is neither the update order nor the draw order.
	engineMid := addEngineComponent("mid", 5, 5)
	engineLast := addEngineComponent("last", 9, 1)
	engineFirst := addEngineComponent("first", 1, 9)

	// Nothing has been initialized: inRun is false, so the add handler queued
	// each component instead of initializing it.
	check("game-base-call.adding-before-the-run-initializes-nothing", "GAME_BASE_CALL", "3,",
		fmt.Sprintf("%d,%s", engineGame.Components().Count(), strings.Join(engineLog, ",")))

	// base Initialize drains that queue, in ADD order: notYetInitialized is a
	// plain list, and only the update and draw lists are ordered.
	engineLog = nil
	initializeError := framework.GameBaseInitialize(engineGame)
	check("game-base-call.base-initialize-drains-in-add-order", "GAME_BASE_CALL",
		"true,init:mid,init:last,init:first",
		fmt.Sprintf("%t,%s", initializeError == nil, strings.Join(engineLog, ",")))

	// A second call has nothing left: each component is removed as it is
	// initialized, so the drain is not repeatable.
	engineLog = nil
	secondInitialize := framework.GameBaseInitialize(engineGame)
	check("game-base-call.base-initialize-is-not-repeatable", "GAME_BASE_CALL", "true,",
		fmt.Sprintf("%t,%s", secondInitialize == nil, strings.Join(engineLog, ",")))

	// base Update iterates by UpdateOrder; base Draw by DrawOrder, which here
	// is the reverse, so the two derived lists are provably independent.
	engineLog = nil
	updateError := framework.GameBaseUpdate(engineGame, framework.NewGameTimeByNone())
	drawError := framework.GameBaseDraw(engineGame, framework.NewGameTimeByNone())
	check("game-base-call.update-and-draw-orders-are-independent", "GAME_BASE_CALL",
		"true,true,update:first,update:mid,update:last,draw:last,draw:mid,draw:first",
		fmt.Sprintf("%t,%t,%s", updateError == nil, drawError == nil, strings.Join(engineLog, ",")))

	// Enabled and Visible are read at iteration time and gate one loop each.
	_ = engineMid.SetEnabled(false)
	_ = engineFirst.SetVisible(false)
	engineLog = nil
	_ = framework.GameBaseUpdate(engineGame, framework.NewGameTimeByNone())
	_ = framework.GameBaseDraw(engineGame, framework.NewGameTimeByNone())
	check("game-base-call.enabled-and-visible-gate-one-loop-each", "GAME_BASE_CALL",
		"update:first,update:last,draw:last,draw:mid",
		strings.Join(engineLog, ","))
	_ = engineMid.SetEnabled(true)
	_ = engineFirst.SetVisible(true)

	// An order change re-places the component, and lands it AFTER an existing
	// run of the same order because of the reference's forward walk.
	_ = engineFirst.SetUpdateOrder(9)
	engineLog = nil
	_ = framework.GameBaseUpdate(engineGame, framework.NewGameTimeByNone())
	check("game-base-call.order-change-lands-after-an-equal-order-run", "GAME_BASE_CALL",
		"update:mid,update:last,update:first",
		strings.Join(engineLog, ","))

	// Mutating the collection during the frame does not change that frame: the
	// first loop copies the ordered list and the second iterates the copy.
	engineLate := newOrderedCorpusComponent("late", &engineLog, -99, -99)
	engineMid.onUpdate = func() {
		_ = engineGame.Components().Add(engineLate)
		_, _ = engineGame.Components().Remove(engineLast)
	}
	engineLog = nil
	_ = framework.GameBaseUpdate(engineGame, framework.NewGameTimeByNone())
	thisFrame := strings.Join(engineLog, ",")
	engineMid.onUpdate = nil
	engineLog = nil
	_ = framework.GameBaseUpdate(engineGame, framework.NewGameTimeByNone())
	check("game-base-call.mid-frame-mutation-applies-to-the-next-frame", "GAME_BASE_CALL",
		"update:mid,update:last,update:first|update:late,update:mid,update:first",
		thisFrame+"|"+strings.Join(engineLog, ","))

	// The family's one Go-only failure. It has no CLR counterpart: a CLR Game
	// always ran its constructor, and Go can produce one that did not.
	unconstructed := &framework.Game{}
	check("game-base-call.an-unconstructed-game-is-refused", "GAME_BASE_CALL",
		"true,true,true,true,true",
		fmt.Sprintf("%t,%t,%t,%t,%t",
			framework.GameBaseInitialize(unconstructed) != nil,
			framework.GameBaseLoadContent(unconstructed) != nil,
			framework.GameBaseUnloadContent(unconstructed) != nil,
			framework.GameBaseUpdate(unconstructed, framework.NewGameTimeByNone()) != nil,
			framework.GameBaseDraw(unconstructed, framework.NewGameTimeByNone()) != nil))

	// Both no-op base bodies are a bare `ret` of code size 1 in the reference,
	// so they must do nothing at all -- not even to the pending queue.
	noOpGame, _ := framework.NewGame(corpusCallbacks{})
	var noOpLog []string
	_ = noOpGame.Components().Add(newOrderedCorpusComponent("untouched", &noOpLog, 0, 0))
	loadError := framework.GameBaseLoadContent(noOpGame)
	unloadError := framework.GameBaseUnloadContent(noOpGame)
	check("game-base-call.load-and-unload-content-are-faithful-no-ops", "GAME_BASE_CALL",
		"true,true,,1",
		fmt.Sprintf("%t,%t,%s,%d", loadError == nil, unloadError == nil,
			strings.Join(noOpLog, ","), noOpGame.Components().Count()))

	// ------------------------------------------------------------------
	// Foundation 32. GameComponent, the class the whole engine was built for.
	// ------------------------------------------------------------------
	componentGame, _ := framework.NewGame(corpusCallbacks{})
	gameComponent := framework.NewGameComponent(componentGame)

	// The 21-byte constructor: Enabled true, UpdateOrder zero, the Game stored
	// as handed over, and nothing validated.
	orphanComponent := framework.NewGameComponent(nil)
	check("game-gameComponent.constructor-defaults", "GAME_COMPONENT", "true,0,true,true,true",
		fmt.Sprintf("%t,%d,%t,%t,%t", gameComponent.Enabled(), gameComponent.UpdateOrder(),
			gameComponent.Game() == componentGame, orphanComponent.Game() == nil, orphanComponent.Enabled()))

	// Both settable properties suppress an unchanged value, store before they
	// announce, and raise with the COMPONENT as sender rather than whatever
	// sender the On... method was handed.
	var componentEvents []string
	_, _ = gameComponent.AddEnabledChangedHandler(func(sender any, args *framework.EventArgs) error {
		componentEvents = append(componentEvents, fmt.Sprintf("enabled=%t,self=%t,empty=%t",
			sender.(*framework.GameComponent).Enabled(), sender == any(gameComponent), args == framework.EventArgsEmpty()))
		return nil
	})
	_, _ = gameComponent.AddUpdateOrderChangedHandler(func(sender any, args *framework.EventArgs) error {
		componentEvents = append(componentEvents, fmt.Sprintf("order=%d,self=%t",
			sender.(*framework.GameComponent).UpdateOrder(), sender == any(gameComponent)))
		return nil
	})
	sameEnabled := gameComponent.SetEnabled(true)
	sameOrder := gameComponent.SetUpdateOrder(0)
	check("game-gameComponent.unchanged-assignment-announces-nothing", "GAME_COMPONENT", "true,true,",
		fmt.Sprintf("%t,%t,%s", sameEnabled == nil, sameOrder == nil, strings.Join(componentEvents, "|")))

	componentEvents = nil
	_ = gameComponent.SetEnabled(false)
	_ = gameComponent.SetUpdateOrder(7)
	check("game-gameComponent.changed-assignment-stores-then-announces", "GAME_COMPONENT",
		"enabled=false,self=true,empty=true|order=7,self=true",
		strings.Join(componentEvents, "|"))

	// The On... methods accept a sender, ignore it, and raise with `this`.
	// That is what makes Game's engine work, since its handler reads sender.
	componentEvents = nil
	decoy := framework.NewGameComponent(nil)
	_ = gameComponent.OnEnabledChanged(decoy, framework.EventArgsEmpty())
	check("game-gameComponent.on-methods-ignore-their-sender-argument", "GAME_COMPONENT",
		"enabled=false,self=true,empty=true",
		strings.Join(componentEvents, "|"))

	// Dispose removes from Game.Components BEFORE it announces, so a Disposed
	// handler observes a Game that no longer contains the gameComponent.
	disposeGame, _ := framework.NewGame(corpusCallbacks{})
	disposed := framework.NewGameComponent(disposeGame)
	addDisposed := disposeGame.Components().Add(disposed)
	observedDuringDispose := int32(-1)
	disposeRaises := 0
	_, _ = disposed.AddDisposedHandler(func(sender any, args *framework.EventArgs) error {
		disposeRaises++
		observedDuringDispose = disposeGame.Components().Count()
		return nil
	})
	disposeError := disposed.DisposeByNone()
	check("game-gameComponent.dispose-removes-then-announces", "GAME_COMPONENT", "true,true,0,0,1",
		fmt.Sprintf("%t,%t,%d,%d,%d", addDisposed == nil, disposeError == nil,
			observedDuringDispose, disposeGame.Components().Count(), disposeRaises))

	// It is NOT idempotent: the reference pops Remove's boolean and raises
	// Disposed unconditionally, so a second Dispose raises again.
	secondDispose := disposed.DisposeByNone()
	check("game-gameComponent.dispose-is-not-idempotent", "GAME_COMPONENT", "true,2",
		fmt.Sprintf("%t,%d", secondDispose == nil, disposeRaises))

	// Dispose(false) is the whole body behind `if (!disposing) return`, which
	// is also why Finalize is observably a no-op.
	quietGame, _ := framework.NewGame(corpusCallbacks{})
	quiet := framework.NewGameComponent(quietGame)
	_ = quietGame.Components().Add(quiet)
	quietRaises := 0
	_, _ = quiet.AddDisposedHandler(func(any, *framework.EventArgs) error { quietRaises++; return nil })
	quietDispose := quiet.DisposeByBoolean(false)
	quiet.Finalize()
	check("game-gameComponent.dispose-false-and-finalize-do-nothing", "GAME_COMPONENT", "true,0,1",
		fmt.Sprintf("%t,%d,%d", quietDispose == nil, quietRaises, quietGame.Components().Count()))

	// And the class drives the engine with no adapter between them: order
	// changes re-place it, base Update runs it, Dispose untracks it, and a
	// GameComponent is never drawn because it is not IDrawable.
	engineComponentGame, _ := framework.NewGame(corpusCallbacks{})
	early := framework.NewGameComponent(engineComponentGame)
	late := framework.NewGameComponent(engineComponentGame)
	_ = late.SetUpdateOrder(5)
	_ = engineComponentGame.Components().Add(early)
	_ = engineComponentGame.Components().Add(late)
	_, isDrawable := any(early).(framework.IDrawable)
	_, isUpdateable := any(early).(framework.IUpdateable)
	_, isComponent := any(early).(framework.IGameComponent)
	initializeComponents := framework.GameBaseInitialize(engineComponentGame)
	updateComponents := framework.GameBaseUpdate(engineComponentGame, framework.NewGameTimeByNone())
	drawComponents := framework.GameBaseDraw(engineComponentGame, framework.NewGameTimeByNone())
	disposeEarly := early.DisposeByNone()
	check("game-gameComponent.drives-the-engine", "GAME_COMPONENT", "false,true,true,true,true,true,true,1",
		fmt.Sprintf("%t,%t,%t,%t,%t,%t,%t,%d", isDrawable, isUpdateable, isComponent,
			initializeComponents == nil, updateComponents == nil, drawComponents == nil,
			disposeEarly == nil, engineComponentGame.Components().Count()))

	// ------------------------------------------------------------------
	// Foundation 34. Game's four events and the three protected raise sites.
	// ------------------------------------------------------------------
	eventGame, _ := framework.NewGame(corpusCallbacks{})
	eventSenderGame, _ := framework.NewGame(corpusCallbacks{})

	// OnActivated and OnDeactivated accept a sender, ignore it, and raise with
	// `this`. OnExiting is the outlier: its IL pushes `ldnull`, so the handler
	// receives no sender at all. The args are forwarded unchanged in all three.
	var eventSenders []string
	customArgs := framework.EventArgsEmpty()
	_, _ = eventGame.AddActivatedHandler(func(sender any, args *framework.EventArgs) error {
		eventSenders = append(eventSenders, fmt.Sprintf("activated:self=%t,args=%t", sender == any(eventGame), args == customArgs))
		return nil
	})
	_, _ = eventGame.AddDeactivatedHandler(func(sender any, args *framework.EventArgs) error {
		eventSenders = append(eventSenders, fmt.Sprintf("deactivated:self=%t,args=%t", sender == any(eventGame), args == customArgs))
		return nil
	})
	_, _ = eventGame.AddExitingHandler(func(sender any, args *framework.EventArgs) error {
		eventSenders = append(eventSenders, fmt.Sprintf("exiting:nil=%t,args=%t", sender == nil, args == customArgs))
		return nil
	})
	activatedRaise := eventGame.OnActivated(eventSenderGame, customArgs)
	deactivatedRaise := eventGame.OnDeactivated(eventSenderGame, customArgs)
	exitingRaise := eventGame.OnExiting(eventSenderGame, customArgs)
	check("game-event.on-methods-drop-their-sender", "GAME_EVENT",
		"activated:self=true,args=true|deactivated:self=true,args=true|exiting:nil=true,args=true,true,true,true",
		fmt.Sprintf("%s,%t,%t,%t", strings.Join(eventSenders, "|"),
			activatedRaise == nil, deactivatedRaise == nil, exitingRaise == nil))

	// The four events are four separate registration lists. A token from one
	// removes nothing from another, and a handler on one never runs for another.
	independentGame, _ := framework.NewGame(corpusCallbacks{})
	var independentLog []string
	independentHandler := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(any, *framework.EventArgs) error {
			independentLog = append(independentLog, name)
			return nil
		}
	}
	activatedToken, _ := independentGame.AddActivatedHandler(independentHandler("activated"))
	_, _ = independentGame.AddDeactivatedHandler(independentHandler("deactivated"))
	_, _ = independentGame.AddExitingHandler(independentHandler("exiting"))
	_, _ = independentGame.AddDisposedHandler(independentHandler("disposed"))
	crossRemovals := fmt.Sprintf("%t,%t,%t",
		independentGame.RemoveDeactivatedHandler(activatedToken) == nil,
		independentGame.RemoveExitingHandler(activatedToken) == nil,
		independentGame.RemoveDisposedHandler(activatedToken) == nil)
	_ = independentGame.OnActivated(nil, framework.EventArgsEmpty())
	_ = independentGame.OnDeactivated(nil, framework.EventArgsEmpty())
	_ = independentGame.OnExiting(nil, framework.EventArgsEmpty())
	check("game-event.the-four-lists-are-independent", "GAME_EVENT",
		"true,true,true,activated,deactivated,exiting",
		fmt.Sprintf("%s,%s", crossRemovals, strings.Join(independentLog, ",")))

	// Dispatch is registration order, which is what one native subscription per
	// event buys: CNA invokes multiple native registrations on one event in
	// reverse order, and none of that is visible here.
	orderGame, _ := framework.NewGame(corpusCallbacks{})
	var eventOrder []string
	for _, name := range []string{"first", "second", "third"} {
		label := name
		_, _ = orderGame.AddExitingHandler(func(any, *framework.EventArgs) error {
			eventOrder = append(eventOrder, label)
			return nil
		})
	}
	_ = orderGame.OnExiting(nil, framework.EventArgsEmpty())
	check("game-event.dispatch-is-registration-order", "GAME_EVENT", "first,second,third",
		strings.Join(eventOrder, ","))

	// The settled accessor projection, on a Game: one handler added twice is
	// two registrations, a token removes exactly one, and the zero token, an
	// already-removed token and a token owned by another Game are all inert.
	tokenGame, _ := framework.NewGame(corpusCallbacks{})
	tokenCalls := 0
	tokenHandler := func(any, *framework.EventArgs) error { tokenCalls++; return nil }
	firstDisposedToken, _ := tokenGame.AddDisposedHandler(tokenHandler)
	_, _ = tokenGame.AddDisposedHandler(tokenHandler)
	foreignGameToken, _ := eventSenderGame.AddDisposedHandler(tokenHandler)
	nilGameToken, nilGameError := tokenGame.AddActivatedHandler(nil)
	inertRemovals := fmt.Sprintf("%t,%t,%t,%t",
		tokenGame.RemoveDisposedHandler(firstDisposedToken) == nil,
		tokenGame.RemoveDisposedHandler(firstDisposedToken) == nil,
		tokenGame.RemoveDisposedHandler(framework.EventSubscription{}) == nil,
		tokenGame.RemoveDisposedHandler(foreignGameToken) == nil)
	tokenCalls = 0
	_ = tokenGame.OnActivated(nil, framework.EventArgsEmpty())
	disposedAfterRemoval := tokenCalls
	check("game-event.token-identity-and-inert-removals", "GAME_EVENT",
		"true,true,true,true,true,true,0",
		fmt.Sprintf("%s,%t,%t,%d", inertRemovals,
			nilGameToken == framework.EventSubscription{}, nilGameError == nil, disposedAfterRemoval))

	// A failing handler stops dispatch, propagates, and leaves the list intact.
	failGame, _ := framework.NewGame(corpusCallbacks{})
	failSentinel := errors.New("corpus game event failure")
	failRan, laterRan := 0, 0
	_, _ = failGame.AddExitingHandler(func(any, *framework.EventArgs) error { failRan++; return failSentinel })
	_, _ = failGame.AddExitingHandler(func(any, *framework.EventArgs) error { laterRan++; return nil })
	firstFailure := failGame.OnExiting(nil, framework.EventArgsEmpty())
	secondFailure := failGame.OnExiting(nil, framework.EventArgsEmpty())
	checkGoProjection("game-event.handler-failure-stops-dispatch", "GAME_EVENT", "true,true,2,0",
		fmt.Sprintf("%t,%t,%d,%d",
			errors.Is(firstFailure, failSentinel), errors.Is(secondFailure, failSentinel),
			failRan, laterRan))

	return report
}

// corpusDeviceService is a corpus-local conformer of the device-publication
// contract. Only it can raise; a consumer holding the contract can only
// subscribe. It publishes no device and reproduces no XNA implementor.
type corpusDeviceService struct {
	created   framework.EventSource[*framework.EventArgs]
	disposing framework.EventSource[*framework.EventArgs]
	reset     framework.EventSource[*framework.EventArgs]
	resetting framework.EventSource[*framework.EventArgs]
}

func (s *corpusDeviceService) GraphicsDevice() *graphics.GraphicsDevice { return nil }

func (s *corpusDeviceService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.created.Add(h)
}
func (s *corpusDeviceService) RemoveDeviceCreatedHandler(sub framework.EventSubscription) error {
	return s.created.Remove(sub)
}
func (s *corpusDeviceService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.disposing.Add(h)
}
func (s *corpusDeviceService) RemoveDeviceDisposingHandler(sub framework.EventSubscription) error {
	return s.disposing.Remove(sub)
}
func (s *corpusDeviceService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.reset.Add(h)
}
func (s *corpusDeviceService) RemoveDeviceResetHandler(sub framework.EventSubscription) error {
	return s.reset.Remove(sub)
}
func (s *corpusDeviceService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resetting.Add(h)
}
func (s *corpusDeviceService) RemoveDeviceResettingHandler(sub framework.EventSubscription) error {
	return s.resetting.Remove(sub)
}

func (s *corpusDeviceService) raiseAll() {
	_ = s.created.Raise(s, framework.EventArgsEmpty())
	_ = s.resetting.Raise(s, framework.EventArgsEmpty())
	_ = s.reset.Raise(s, framework.EventArgsEmpty())
	_ = s.disposing.Raise(s, framework.EventArgsEmpty())
}

// corpusComponent is a corpus-local conformer of both component contracts. It
// is the shape an external package must be able to write: private EventSource
// fields behind the projected accessors. It reproduces no XNA component.
// corpusCallbacks is a do-nothing GameCallbacks so the corpus can construct a
// Game. It runs no override body: every observation below reaches Game's
// managed state directly or through an explicit base call.
type corpusCallbacks struct{}

func (corpusCallbacks) Initialize(*framework.Game) error                 { return nil }
func (corpusCallbacks) LoadContent(*framework.Game) error                { return nil }
func (corpusCallbacks) Update(*framework.Game, framework.GameTime) error { return nil }
func (corpusCallbacks) Draw(*framework.Game, framework.GameTime) error   { return nil }
func (corpusCallbacks) UnloadContent(*framework.Game) error              { return nil }

// orderedCorpusComponent is a corpus-local component whose update and draw
// orders differ, which is what makes the two derived lists distinguishable. It
// reproduces GameComponent's setter shape -- suppress when unchanged, store,
// then raise with itself as sender -- because the engine's handlers read the
// sender.
type orderedCorpusComponent struct {
	name        string
	log         *[]string
	enabled     bool
	visible     bool
	updateOrder int32
	drawOrder   int32
	onUpdate    func()

	enabledChanged     framework.EventSource[*framework.EventArgs]
	updateOrderChanged framework.EventSource[*framework.EventArgs]
	visibleChanged     framework.EventSource[*framework.EventArgs]
	drawOrderChanged   framework.EventSource[*framework.EventArgs]
}

func newOrderedCorpusComponent(name string, log *[]string, updateOrder, drawOrder int32) *orderedCorpusComponent {
	return &orderedCorpusComponent{name: name, log: log, enabled: true, visible: true,
		updateOrder: updateOrder, drawOrder: drawOrder}
}

func (c *orderedCorpusComponent) record(entry string) {
	if c.log != nil {
		*c.log = append(*c.log, entry)
	}
}

func (c *orderedCorpusComponent) Initialize() error  { c.record("init:" + c.name); return nil }
func (c *orderedCorpusComponent) Enabled() bool      { return c.enabled }
func (c *orderedCorpusComponent) Visible() bool      { return c.visible }
func (c *orderedCorpusComponent) UpdateOrder() int32 { return c.updateOrder }
func (c *orderedCorpusComponent) DrawOrder() int32   { return c.drawOrder }

func (c *orderedCorpusComponent) Update(framework.GameTime) {
	c.record("update:" + c.name)
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

func (c *orderedCorpusComponent) Draw(framework.GameTime) { c.record("draw:" + c.name) }

func (c *orderedCorpusComponent) SetEnabled(value bool) error {
	if c.enabled == value {
		return nil
	}
	c.enabled = value
	return c.enabledChanged.Raise(c, framework.EventArgsEmpty())
}

func (c *orderedCorpusComponent) SetVisible(value bool) error {
	if c.visible == value {
		return nil
	}
	c.visible = value
	return c.visibleChanged.Raise(c, framework.EventArgsEmpty())
}

func (c *orderedCorpusComponent) SetUpdateOrder(value int32) error {
	if c.updateOrder == value {
		return nil
	}
	c.updateOrder = value
	return c.updateOrderChanged.Raise(c, framework.EventArgsEmpty())
}

func (c *orderedCorpusComponent) SetDrawOrder(value int32) error {
	if c.drawOrder == value {
		return nil
	}
	c.drawOrder = value
	return c.drawOrderChanged.Raise(c, framework.EventArgsEmpty())
}

func (c *orderedCorpusComponent) AddEnabledChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.enabledChanged.Add(h)
}
func (c *orderedCorpusComponent) RemoveEnabledChangedHandler(s framework.EventSubscription) error {
	return c.enabledChanged.Remove(s)
}
func (c *orderedCorpusComponent) AddUpdateOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.updateOrderChanged.Add(h)
}
func (c *orderedCorpusComponent) RemoveUpdateOrderChangedHandler(s framework.EventSubscription) error {
	return c.updateOrderChanged.Remove(s)
}
func (c *orderedCorpusComponent) AddVisibleChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.visibleChanged.Add(h)
}
func (c *orderedCorpusComponent) RemoveVisibleChangedHandler(s framework.EventSubscription) error {
	return c.visibleChanged.Remove(s)
}
func (c *orderedCorpusComponent) AddDrawOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.drawOrderChanged.Add(h)
}
func (c *orderedCorpusComponent) RemoveDrawOrderChangedHandler(s framework.EventSubscription) error {
	return c.drawOrderChanged.Remove(s)
}

var (
	_ framework.IGameComponent = (*orderedCorpusComponent)(nil)
	_ framework.IUpdateable    = (*orderedCorpusComponent)(nil)
	_ framework.IDrawable      = (*orderedCorpusComponent)(nil)
)

type corpusComponent struct {
	enabled bool
	updates int
	draws   int

	enabledChanged     framework.EventSource[*framework.EventArgs]
	updateOrderChanged framework.EventSource[*framework.EventArgs]
	visibleChanged     framework.EventSource[*framework.EventArgs]
	drawOrderChanged   framework.EventSource[*framework.EventArgs]
}

func (c *corpusComponent) Initialize() error         { return nil }
func (c *corpusComponent) Enabled() bool             { return c.enabled }
func (c *corpusComponent) UpdateOrder() int32        { return 0 }
func (c *corpusComponent) Visible() bool             { return false }
func (c *corpusComponent) DrawOrder() int32          { return 0 }
func (c *corpusComponent) Update(framework.GameTime) { c.updates++ }
func (c *corpusComponent) Draw(framework.GameTime)   { c.draws++ }

func (c *corpusComponent) SetEnabled(value bool) error {
	if c.enabled == value {
		return nil
	}
	c.enabled = value
	return c.enabledChanged.Raise(c, framework.EventArgsEmpty())
}

func (c *corpusComponent) AddEnabledChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.enabledChanged.Add(h)
}
func (c *corpusComponent) RemoveEnabledChangedHandler(s framework.EventSubscription) error {
	return c.enabledChanged.Remove(s)
}
func (c *corpusComponent) AddUpdateOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.updateOrderChanged.Add(h)
}
func (c *corpusComponent) RemoveUpdateOrderChangedHandler(s framework.EventSubscription) error {
	return c.updateOrderChanged.Remove(s)
}
func (c *corpusComponent) AddVisibleChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.visibleChanged.Add(h)
}
func (c *corpusComponent) RemoveVisibleChangedHandler(s framework.EventSubscription) error {
	return c.visibleChanged.Remove(s)
}
func (c *corpusComponent) AddDrawOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return c.drawOrderChanged.Add(h)
}
func (c *corpusComponent) RemoveDrawOrderChangedHandler(s framework.EventSubscription) error {
	return c.drawOrderChanged.Remove(s)
}

var (
	_ framework.IGameComponent = (*corpusComponent)(nil)
	_ framework.IUpdateable    = (*corpusComponent)(nil)
	_ framework.IDrawable      = (*corpusComponent)(nil)
)

// serviceCorpusProbe and serviceCorpusProvider are corpus-local doubles used to
// exercise GameServiceContainer's assignability rule. They reproduce no XNA
// service.
type serviceCorpusProbe interface{ CorpusPing() }

type serviceCorpusProvider struct{}

func (*serviceCorpusProvider) CorpusPing() {}

// interfaceProbe is a corpus-local double for the Foundation-18 contracts. It
// exists to observe the Go projection of the contracts -- that they are
// satisfiable and that their channels are separate -- and reproduces no XNA
// effect, component, or device behavior. Every observation it takes part in is
// labeled GO_LANGUAGE_PROJECTION for that reason.
type interfaceProbe struct {
	world           framework.Matrix
	view            framework.Matrix
	projection      framework.Matrix
	fogEnabled      bool
	fogStart        float32
	fogEnd          float32
	fogColor        framework.Vector3
	fogColorFailure error
	proceed         bool
}

func (p *interfaceProbe) World() framework.Matrix              { return p.world }
func (p *interfaceProbe) SetWorld(value framework.Matrix)      { p.world = value }
func (p *interfaceProbe) View() framework.Matrix               { return p.view }
func (p *interfaceProbe) SetView(value framework.Matrix)       { p.view = value }
func (p *interfaceProbe) Projection() framework.Matrix         { return p.projection }
func (p *interfaceProbe) SetProjection(value framework.Matrix) { p.projection = value }
func (p *interfaceProbe) FogEnabled() bool                     { return p.fogEnabled }
func (p *interfaceProbe) SetFogEnabled(value bool)             { p.fogEnabled = value }
func (p *interfaceProbe) FogStart() float32                    { return p.fogStart }
func (p *interfaceProbe) SetFogStart(value float32)            { p.fogStart = value }
func (p *interfaceProbe) FogEnd() float32                      { return p.fogEnd }
func (p *interfaceProbe) SetFogEnd(value float32)              { p.fogEnd = value }

func (p *interfaceProbe) FogColor() (framework.Vector3, error) {
	if p.fogColorFailure != nil {
		return framework.Vector3{}, p.fogColorFailure
	}
	return p.fogColor, nil
}

func (p *interfaceProbe) SetFogColor(value framework.Vector3) error {
	if p.fogColorFailure != nil {
		return p.fogColorFailure
	}
	p.fogColor = value
	return nil
}

func (p *interfaceProbe) Initialize() error        { return nil }
func (p *interfaceProbe) CreateDevice() error      { return nil }
func (p *interfaceProbe) BeginDraw() (bool, error) { return p.proceed, nil }
func (p *interfaceProbe) EndDraw() error           { return nil }

var (
	_ graphics.IEffectMatrices         = (*interfaceProbe)(nil)
	_ graphics.IEffectFog              = (*interfaceProbe)(nil)
	_ framework.IGameComponent         = (*interfaceProbe)(nil)
	_ framework.IGraphicsDeviceManager = (*interfaceProbe)(nil)
)

// enumBitStructure reports the union of a pinned flags enum's literals and
// whether every non-zero literal is a distinct single bit. Both facts come
// from the pinned XNA metadata values, not from any runtime behavior.
func enumBitStructure(values []int32) (int32, bool) {
	var union int32
	seen := make(map[int32]bool, len(values))
	singleBits := true
	for _, value := range values {
		union |= value
		if value == 0 {
			continue
		}
		if value&(value-1) != 0 || seen[value] {
			singleBits = false
		}
		seen[value] = true
	}
	return union, singleBits
}
