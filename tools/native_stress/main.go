// Command native_stress runs each native lifetime scenario in a crash-isolated
// subprocess. It does not claim sanitizer or leak-detector coverage.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	input "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

type counters struct {
	GameCycles           int `json:"GAME_CYCLES"`
	GameRecreationCycles int `json:"GAME_RECREATION_CYCLES"`
	TextureCycles        int `json:"TEXTURE_CYCLES"`
	// The render-target semantic slice. Binds and BindRefusals are mutually
	// exclusive per cycle and are counted separately on purpose: a refusal is
	// CNA's documented answer on a backend with no off-screen storage, and a
	// run that only ever refused must not read as a run that only ever bound.
	RenderTargetCycles             int `json:"RENDER_TARGET_CYCLES"`
	RenderTargetCreations          int `json:"RENDER_TARGET_CREATIONS"`
	RenderTargetDescriptionChecks  int `json:"RENDER_TARGET_DESCRIPTION_CHECKS"`
	RenderTargetSubstitutionChecks int `json:"RENDER_TARGET_SUBSTITUTION_CHECKS"`
	RenderTargetBinds              int `json:"RENDER_TARGET_BINDS"`
	RenderTargetBindRefusals       int `json:"RENDER_TARGET_BIND_REFUSALS"`
	RenderTargetUnbinds            int `json:"RENDER_TARGET_UNBINDS"`
	RenderTargetPixelChecks        int `json:"RENDER_TARGET_PIXEL_CHECKS"`
	RenderTargetReadbackRefusals   int `json:"RENDER_TARGET_READBACK_REFUSALS"`
	RenderTargetSpriteDraws        int `json:"RENDER_TARGET_SPRITE_DRAWS"`
	RenderTargetDisposalChecks     int `json:"RENDER_TARGET_DISPOSAL_CHECKS"`
	// InheritedDisposeVirtualChecks counts the runs of the one control that can
	// tell the inherited Dispose() reaching the DERIVED override from it
	// reaching the composed base's slot. Every managed observable agrees for
	// both; only the native handle disagrees.
	InheritedDisposeVirtualChecks int `json:"INHERITED_DISPOSE_VIRTUAL_CHECKS"`
	SpriteBatchCycles             int `json:"SPRITEBATCH_CYCLES"`
	CallbackErrorCycles           int `json:"CALLBACK_ERROR_CYCLES"`
	CallbackPanicCycles           int `json:"CALLBACK_PANIC_CYCLES"`
	WrongThreadChecks             int `json:"WRONG_THREAD_CHECKS"`
	OwnerThreadRetries            int `json:"OWNER_THREAD_RETRIES"`
	GCStressPoints                int `json:"GC_STRESS_POINTS"`
	NativeCrashes                 int `json:"NATIVE_CRASHES"`
	ObservedUAF                   int `json:"OBSERVED_UAF"`
	ObservedDoubleFree            int `json:"OBSERVED_DOUBLE_FREE"`

	GameEventActivated   int `json:"GAME_EVENT_ACTIVATED_DELIVERIES"`
	GameEventDeactivated int `json:"GAME_EVENT_DEACTIVATED_DELIVERIES"`
	GameEventExiting     int `json:"GAME_EVENT_EXITING_DELIVERIES"`

	// The disposal counters are two different facts and are deliberately two
	// different counters. The native signal is CNA reporting native game
	// destruction from inside cna_game_destroy; the managed raise is
	// Game::Disposed, which the reference raises from Dispose(bool) and from
	// nowhere else. Foundation 39 stopped the first from driving the second.
	GameNativeDisposalSignals int `json:"GAME_NATIVE_DISPOSAL_SIGNALS"`
	GameDisposedDuringRun     int `json:"GAME_DISPOSED_RAISED_DURING_RUN"`
	GameDisposedByManagedCall int `json:"GAME_DISPOSED_RAISED_BY_MANAGED_DISPOSE"`
	GameDisposedRepeatChecks  int `json:"GAME_DISPOSED_REPEAT_CHECKS"`
	GameDisposeAfterRunCycles int `json:"GAME_DISPOSE_AFTER_RUN_CYCLES"`
	GameEventOrderChecks      int `json:"GAME_EVENT_ORDER_CHECKS"`
	GameEventRemovalChecks    int `json:"GAME_EVENT_REMOVAL_CHECKS"`
	GameEventOwnerThreadHits  int `json:"GAME_EVENT_OWNER_THREAD_CHECKS"`
	GameEventRerunCycles      int `json:"GAME_EVENT_RERUN_CYCLES"`
	GameEventPostRunChecks    int `json:"GAME_EVENT_POST_RUN_CHECKS"`

	// Foundation 50. Every Draw overload the profile declares, submitted to a
	// live native SpriteBatch inside a real draw callback, plus the two guards
	// InternalDraw applies before it queues anything.
	SpriteDrawCycles             int `json:"SPRITE_DRAW_CYCLES"`
	SpriteDrawScaledSubmits      int `json:"SPRITE_DRAW_SCALED_SUBMISSIONS"`
	SpriteDrawDestinationSubmits int `json:"SPRITE_DRAW_DESTINATION_SUBMISSIONS"`
	SpriteDrawNullTextureChecks  int `json:"SPRITE_DRAW_NULL_TEXTURE_CHECKS"`
	SpriteDrawOutsidePairChecks  int `json:"SPRITE_DRAW_OUTSIDE_PAIR_CHECKS"`
	SpriteDrawPairGuardChecks    int `json:"SPRITE_DRAW_PAIR_GUARD_CHECKS"`
	SpriteDrawBoundsChecks       int `json:"SPRITE_DRAW_TEXTURE_BOUNDS_CHECKS"`

	// Foundation 51. GraphicsDevice's render state, round-tripped through the
	// live device, plus the two masked Clear overloads and Present.
	DeviceStateCycles                 int `json:"DEVICE_STATE_CYCLES"`
	DeviceStateRoundTrips             int `json:"DEVICE_STATE_ROUND_TRIPS"`
	DeviceStateObjectRefusals         int `json:"DEVICE_STATE_OBJECT_REFUSALS"`
	DeviceStateObjectBinds            int `json:"DEVICE_STATE_OBJECT_BINDS"`
	SpriteBatchStateBegins            int `json:"SPRITE_BATCH_STATE_BEGINS"`
	DeviceCollectionIdentityChecks    int `json:"DEVICE_COLLECTION_IDENTITY_CHECKS"`
	DeviceCollectionRangeChecks       int `json:"DEVICE_COLLECTION_RANGE_CHECKS"`
	DeviceCollectionTextureRoundTrips int `json:"DEVICE_COLLECTION_TEXTURE_ROUND_TRIPS"`
	DeviceCollectionSamplerRoundTrips int `json:"DEVICE_COLLECTION_SAMPLER_ROUND_TRIPS"`
	DeviceEventSubscriptions          int `json:"DEVICE_EVENT_SUBSCRIPTIONS"`
	DeviceEventRegistrationChecks     int `json:"DEVICE_EVENT_REGISTRATION_CHECKS"`
	DeviceStateReadOnlyChecks         int `json:"DEVICE_STATE_READ_ONLY_CHECKS"`
	DeviceStateClearCalls             int `json:"DEVICE_STATE_CLEAR_CALLS"`
	DeviceStateClearRefusals          int `json:"DEVICE_STATE_CLEAR_REFUSALS"`
	DeviceStatePresentCalls           int `json:"DEVICE_STATE_PRESENT_CALLS"`
	DeviceStateStaleChecks            int `json:"DEVICE_STATE_STALE_CHECKS"`
	DeviceStateWrongThreadHits        int `json:"DEVICE_STATE_WRONG_THREAD_CHECKS"`

	// Foundation 52. The device's display mode, and an EMPTY texture created
	// from its dimensions and format rather than decoded from bytes.
	DeviceStateDisplayModeChecks int `json:"DEVICE_STATE_DISPLAY_MODE_CHECKS"`
	DeviceStateTextureCreations  int `json:"DEVICE_STATE_TEXTURE_CREATIONS"`
	DeviceStateTextureRefusals   int `json:"DEVICE_STATE_TEXTURE_REFUSALS"`

	// Foundation 53. A texture encoded to PNG and to JPEG, and decoded back at
	// a requested size through both zoom modes.
	DeviceStateEncodeChecks     int `json:"DEVICE_STATE_TEXTURE_ENCODE_CHECKS"`
	DeviceStateDecodeSizeChecks int `json:"DEVICE_STATE_TEXTURE_DECODE_SIZE_CHECKS"`
	DeviceStateEncodeRefusals   int `json:"DEVICE_STATE_TEXTURE_ENCODE_REFUSALS"`

	// Foundation 54. Typed transfers through the generic-method projection.
	DeviceStateTransferRoundTrips int `json:"DEVICE_STATE_TEXTURE_TRANSFER_ROUND_TRIPS"`
	DeviceStateTransferRefusals   int `json:"DEVICE_STATE_TEXTURE_TRANSFER_REFUSALS"`

	FrameHookOverrideCycles  int `json:"FRAME_HOOK_OVERRIDE_CYCLES"`
	FrameHookBeginRunHits    int `json:"FRAME_HOOK_BEGIN_RUN_DELIVERIES"`
	FrameHookEndRunHits      int `json:"FRAME_HOOK_END_RUN_DELIVERIES"`
	FrameHookBeginDrawHits   int `json:"FRAME_HOOK_BEGIN_DRAW_DELIVERIES"`
	FrameHookEndDrawHits     int `json:"FRAME_HOOK_END_DRAW_DELIVERIES"`
	FrameHookRefusedFrames   int `json:"FRAME_HOOK_REFUSED_FRAMES"`
	FrameHookAdmittedFrames  int `json:"FRAME_HOOK_ADMITTED_FRAMES"`
	FrameHookEndDrawExpected int `json:"FRAME_HOOK_END_DRAW_EXPECTED"`
	FrameHookSkipChecks      int `json:"FRAME_HOOK_REFUSED_FRAME_SKIP_CHECKS"`
	FrameHookBaseCallChecks  int `json:"FRAME_HOOK_EXPLICIT_BASE_CALL_CHECKS"`
	FrameHookOrderChecks     int `json:"FRAME_HOOK_ORDER_CHECKS"`
	FrameHookSubsetCycles    int `json:"FRAME_HOOK_SUBSET_CYCLES"`
	FrameHookUninstalledHits int `json:"FRAME_HOOK_UNINSTALLED_DELIVERIES"`

	TimingCycles            int `json:"GAME_TIMING_CYCLES"`
	TimingSettersApplied    int `json:"GAME_TIMING_SETTERS_APPLIED"`
	TimingWrongThreadChecks int `json:"GAME_TIMING_WRONG_THREAD_CHECKS"`
	TimingRangeChecks       int `json:"GAME_TIMING_RANGE_CHECKS"`
	TimingCreatedWithConfig int `json:"GAME_TIMING_CREATED_WITH_CONFIGURED_STEP"`

	WindowCycles              int `json:"GAME_WINDOW_CYCLES"`
	WindowIdentityChecks      int `json:"GAME_WINDOW_IDENTITY_CHECKS"`
	WindowGuardedFallbacks    int `json:"GAME_WINDOW_GUARDED_FALLBACK_CHECKS"`
	WindowUnguardedFailures   int `json:"GAME_WINDOW_UNGUARDED_FAILURE_CHECKS"`
	WindowLiveReads           int `json:"GAME_WINDOW_LIVE_READ_CHECKS"`
	WindowTitleSuppressions   int `json:"GAME_WINDOW_TITLE_SUPPRESSION_CHECKS"`
	WindowWrongThreadChecks   int `json:"GAME_WINDOW_WRONG_THREAD_CHECKS"`
	WindowScreenDeviceChanges int `json:"GAME_WINDOW_SCREEN_DEVICE_CHANGE_CYCLES"`
	WindowResizeRoundTrips    int `json:"GAME_WINDOW_RESIZE_ROUND_TRIPS"`
	// Whether the live window reported a positive client size. HEADLESS does
	// not, and that is a renderer fact rather than a binding one.
	WindowPositiveClientBounds int `json:"GAME_WINDOW_POSITIVE_CLIENT_BOUNDS"`
	// The three canonical window signals. HEADLESS never resizes, rotates or
	// changes screen, so these are expected to stay zero in this environment
	// and are recorded rather than asserted -- exactly as
	// GAME_EVENT_DEACTIVATED_DELIVERIES is.
	WindowEventClientSize   int `json:"GAME_WINDOW_EVENT_CLIENT_SIZE_DELIVERIES"`
	WindowEventOrientation  int `json:"GAME_WINDOW_EVENT_ORIENTATION_DELIVERIES"`
	WindowEventScreenDevice int `json:"GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_DELIVERIES"`

	FrameStepCycles            int `json:"GAME_FRAME_STEP_CYCLES"`
	FrameStepTicks             int `json:"GAME_FRAME_STEP_TICKS"`
	FrameStepRunOneFrames      int `json:"GAME_FRAME_STEP_RUN_ONE_FRAMES"`
	FrameStepInitializations   int `json:"GAME_FRAME_STEP_INITIALIZATIONS"`
	FrameStepTickInitChecks    int `json:"GAME_FRAME_STEP_TICK_DOES_NOT_INITIALIZE_CHECKS"`
	FrameStepUpdates           int `json:"GAME_FRAME_STEP_UPDATE_DELIVERIES"`
	FrameStepDraws             int `json:"GAME_FRAME_STEP_DRAW_DELIVERIES"`
	FrameStepSuppressChecks    int `json:"GAME_FRAME_STEP_SUPPRESS_DRAW_CHECKS"`
	FrameStepWrongThreadChecks int `json:"GAME_FRAME_STEP_WRONG_THREAD_CHECKS"`
	FrameStepCallbackRefusals  int `json:"GAME_FRAME_STEP_CALLBACK_REFUSAL_CHECKS"`
	FrameStepExitChecks        int `json:"GAME_FRAME_STEP_EXIT_CHECKS"`
	FrameStepSessionChecks     int `json:"GAME_FRAME_STEP_SESSION_LIFETIME_CHECKS"`
	FrameStepDisposeChecks     int `json:"GAME_FRAME_STEP_DISPOSE_CHECKS"`
	FrameStepRecreationChecks  int `json:"GAME_FRAME_STEP_RECREATION_CHECKS"`
	FrameStepRunAfterStepCycle int `json:"GAME_FRAME_STEP_RUN_ADOPTS_SESSION_CYCLES"`

	ManagerCycles           int `json:"GRAPHICS_MANAGER_CYCLES"`
	ManagerDefaultChecks    int `json:"GRAPHICS_MANAGER_DEFAULT_CHECKS"`
	ManagerSettersApplied   int `json:"GRAPHICS_MANAGER_SETTERS_APPLIED"`
	ManagerCrossPackageSets int `json:"GRAPHICS_MANAGER_CROSS_PACKAGE_SETTERS_APPLIED"`
	ManagerRangeChecks      int `json:"GRAPHICS_MANAGER_RANGE_CHECKS"`
	ManagerApplyChanges     int `json:"GRAPHICS_MANAGER_APPLY_CHANGES"`
	ManagerToggleChecks     int `json:"GRAPHICS_MANAGER_TOGGLE_FULL_SCREEN_CHECKS"`
	ManagerWrongThreadCheck int `json:"GRAPHICS_MANAGER_WRONG_THREAD_CHECKS"`

	ManagerServiceChecks       int `json:"GRAPHICS_MANAGER_SERVICE_REGISTRATION_CHECKS"`
	ManagerDuplicateChecks     int `json:"GRAPHICS_MANAGER_DUPLICATE_REGISTRATION_CHECKS"`
	ManagerGameDeviceChecks    int `json:"GRAPHICS_MANAGER_GAME_GRAPHICS_DEVICE_CHECKS"`
	ManagerDrawableChecks      int `json:"GRAPHICS_MANAGER_DRAWABLE_COMPONENT_CHECKS"`
	ManagerEventRaiseChecks    int `json:"GRAPHICS_MANAGER_EVENT_RAISE_CHECKS"`
	ManagerServiceRemovalCheck int `json:"GRAPHICS_MANAGER_SERVICE_REMOVAL_CHECKS"`
	// The five canonical manager signals. HEADLESS creates its device once and
	// never loses or resets it, so the reset and disposing counters are
	// expected to stay at zero and are recorded rather than asserted.
	ManagerSignalDeviceCreated   int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_CREATED_DELIVERIES"`
	ManagerSignalDeviceReset     int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_RESET_DELIVERIES"`
	ManagerSignalDeviceResetting int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_RESETTING_DELIVERIES"`
	ManagerSignalDeviceDisposing int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_DISPOSING_DELIVERIES"`
	ManagerSignalDisposed        int `json:"GRAPHICS_MANAGER_SIGNAL_DISPOSED_DELIVERIES"`
}

type stressReport struct {
	SchemaVersion         int      `json:"schema_version"`
	Isolation             string   `json:"isolation"`
	GoRaceStatus          string   `json:"GO_RACE_STATUS"`
	NativeSanitizerStatus string   `json:"NATIVE_SANITIZER_STATUS"`
	NativeLibrarySHA256   string   `json:"native_library_sha256,omitempty"`
	Counters              counters `json:"counters"`
}

type stressGame struct {
	scenario string
	index    int
	manager  *framework.GraphicsDeviceManager
	device   *graphics.GraphicsDevice
	data     []byte
	result   counters

	// eventOrder records every native game signal in delivery order, and
	// removedRan records whether a handler removed before Run ever fired.
	eventOrder      []string
	removedRan      bool
	ownerGoroutine  string
	eventGoroutines map[string]bool

	// The sprite-draw scenario's two live objects, created in LoadContent and
	// used from inside Draw, which is the only moment CNA has a render pass.
	spriteTexture *graphics.Texture2D
	spriteBatch   *graphics.SpriteBatch

	// runtime is captured from inside the first callback, which is the only
	// place it is reachable, and it survives the run. It is how the native
	// disposal signal is observed at all now that it raises no public event.
	runtime *interop.Runtime
	// disposedRaises counts public Game.Disposed raises. It must stay zero for
	// the whole run: the native signal no longer drives the event.
	disposedRaises int
}

var callbackSentinel = errors.New("native stress callback sentinel")

func main() {
	child := flag.String("child", "", "internal isolated scenario")
	index := flag.Int("index", 0, "internal scenario index")
	output := flag.String("output", "docs/generated/native-stress-report.json", "parent JSON report; empty disables writing")
	raceStatus := flag.String("race-status", "NOT_RUN", "PASS only when this invocation was built with -race")
	flag.Parse()
	if *child != "" {
		if err := runChild(*child, *index); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	result, err := runParent()
	if writeErr := writeStressReport(*output, *raceStatus, result); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(2)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeStressReport(path, raceStatus string, result counters) error {
	if path == "" {
		return nil
	}
	if raceStatus != "PASS" && raceStatus != "NOT_RUN" {
		return fmt.Errorf("invalid race status %q", raceStatus)
	}
	// The artifact identity, by CONTENT. Foundation 58 runs this against TWO
	// qualified artifacts -- a HEADLESS one and a SOFTWARE one -- and their
	// counters legitimately differ: only the software renderer can read a
	// render target's colour attachment back to the CPU. A report that did not
	// say which artifact produced it could not be read at all.
	report := stressReport{
		SchemaVersion:         1,
		Isolation:             "one native Game generation per subprocess",
		GoRaceStatus:          raceStatus,
		NativeSanitizerStatus: "NOT_RUN",
		NativeLibrarySHA256:   nativeLibraryDigest(),
		Counters:              result,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// nativeLibraryDigest is the SHA-256 of the artifact CNA_NATIVE_LIBRARY names.
// It is empty when the variable is unset, which is the case where the platform
// loader chose the library and CNA-Go cannot say which one it found.
func nativeLibraryDigest() string {
	path := os.Getenv("CNA_NATIVE_LIBRARY")
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func runParent() (counters, error) {
	executable, err := os.Executable()
	if err != nil {
		return counters{}, err
	}
	var total counters
	for _, scenario := range []string{"success", "callback-error", "callback-panic", "event-rerun", "frame-hook-override", "frame-hook-subset", "timing", "window", "frame-step", "frame-step-run", "graphics-manager", "sprite-draw", "device-state", "render-target"} {
		for index := 0; index < 20; index++ {
			command := exec.Command(executable, "--child", scenario, "--index", fmt.Sprint(index))
			command.Env = os.Environ()
			output, runErr := command.CombinedOutput()
			if runErr != nil {
				total.NativeCrashes++
				return total, fmt.Errorf("isolated %s cycle %d failed: %w\n%s", scenario, index, runErr, output)
			}
			var one counters
			if err := decodeLastJSONLine(output, &one); err != nil {
				return total, fmt.Errorf("decode isolated %s cycle %d: %w: %q", scenario, index, err, output)
			}
			addCounters(&total, one)
		}
	}
	if total.GameCycles < 20 || total.GameRecreationCycles < 20 || total.TextureCycles < 20 || total.SpriteBatchCycles < 20 || total.CallbackErrorCycles < 20 || total.CallbackPanicCycles < 20 {
		return total, errors.New("native stress minimum was not met")
	}
	// One clean run per success cycle delivers exactly one Activated, one
	// Exiting and one Disposed, each proved in order and on the owner
	// goroutine. Deactivated has no minimum: HEADLESS cannot produce a focus
	// transition away from the game, and the counter records that honestly by
	// staying at zero.
	if total.GameEventActivated < 20 || total.GameEventExiting < 20 || total.GameNativeDisposalSignals < 20 {
		return total, errors.New("native game-event delivery minimum was not met")
	}
	// Twenty sprite-draw cycles, each submitting five position commands and
	// four destination commands through a live native SpriteBatch. The two
	// families are required SEPARATELY: a projection that sent every overload
	// down one route would still submit, and would place three of the seven
	// wrong.
	if total.SpriteDrawCycles < 20 || total.SpriteDrawScaledSubmits < 100 || total.SpriteDrawDestinationSubmits < 80 {
		return total, errors.New("native sprite-draw submission minimum was not met")
	}
	if total.SpriteDrawNullTextureChecks < 20 || total.SpriteDrawOutsidePairChecks < 40 ||
		total.SpriteDrawPairGuardChecks < 40 || total.SpriteDrawBoundsChecks < 20 {
		return total, errors.New("a sprite-draw guard or bounds proof did not run in every cycle")
	}
	// Five round trips per cycle -- blend factor, multisample mask, reference
	// stencil, scissor rectangle and viewport -- each written to the live
	// device and read back from it. A getter that answered from a managed cache
	// would pass these and would be a second source of truth, so the read is
	// required to come back from the device that was written.
	if total.DeviceStateCycles < 20 || total.DeviceStateRoundTrips < 100 {
		return total, errors.New("native device-state round-trip minimum was not met")
	}
	// Two display-mode checks and three texture creations per cycle, plus the
	// two refusals the projection makes before it reaches CNA.
	if total.DeviceStateDisplayModeChecks < 40 || total.DeviceStateTextureCreations < 60 ||
		total.DeviceStateTextureRefusals < 40 {
		return total, errors.New("a device-state display-mode or texture proof did not run in every cycle")
	}
	// Two encodes and two sized decodes per cycle, plus the one refusal the
	// projection makes before it reaches CNA.
	if total.DeviceStateEncodeChecks < 40 || total.DeviceStateDecodeSizeChecks < 40 ||
		total.DeviceStateEncodeRefusals < 20 {
		return total, errors.New("a texture encode or sized-decode proof did not run in every cycle")
	}
	// Three typed round trips per cycle -- a full-surface Color transfer, a
	// windowed one, and a rectangle one -- each written to the live texture and
	// read back from it, plus the two refusals the projection makes itself.
	if total.DeviceStateTransferRoundTrips < 60 || total.DeviceStateTransferRefusals < 40 {
		return total, errors.New("a texture transfer proof did not run in every cycle")
	}
	if total.DeviceStateReadOnlyChecks < 60 || total.DeviceStateClearCalls < 40 ||
		total.DeviceStateClearRefusals < 20 || total.DeviceStatePresentCalls < 20 ||
		total.DeviceStateStaleChecks < 20 || total.DeviceStateWrongThreadHits < 20 {
		return total, errors.New("a device-state proof did not run in every cycle")
	}
	// Foundation 39. The native disposal signal never raises the public event,
	// and the public event is raised only by a managed Dispose call.
	if total.GameDisposedDuringRun != 0 {
		return total, fmt.Errorf("Game.Disposed was raised %d times from a run; its only reference raise site is managed Dispose(bool)", total.GameDisposedDuringRun)
	}
	if total.GameDisposeAfterRunCycles < 20 || total.GameDisposedRepeatChecks < 20 {
		return total, errors.New("managed disposal after the run was not proved in every cycle")
	}
	// Two raises per cycle: Dispose is not idempotent, so the second call
	// raises again. A projection that had invented a disposed flag would
	// report exactly half this.
	if total.GameDisposedByManagedCall != 2*total.GameDisposeAfterRunCycles {
		return total, fmt.Errorf("%d managed Dispose calls raised Disposed %d times, want two per cycle",
			2*total.GameDisposeAfterRunCycles, total.GameDisposedByManagedCall)
	}
	if total.GameEventOrderChecks < 20 || total.GameEventOwnerThreadHits < 20 {
		return total, errors.New("native game-event ordering or owner-goroutine minimum was not met")
	}
	if total.GameEventRemovalChecks < 80 {
		return total, errors.New("native game-event removal was not proved in every isolated cycle")
	}
	// The rerun scenario is the lifetime proof: a second Run on the same Go
	// Game installs four fresh registrations after the first four were
	// released, and Add/Remove work with no native game alive at all.
	if total.GameEventRerunCycles < 20 || total.GameEventPostRunChecks < 20 {
		return total, errors.New("native game-event rerun minimum was not met")
	}
	// The optional frame hooks. Each override cycle delivers begin_run and
	// end_run exactly once, so those two counters pin the cycle count; the
	// draw hooks fire per frame and have no fixed total, but every refused
	// frame must be proved to have skipped BOTH draw and end_draw, and every
	// override call must be proved to have reached the base exactly once.
	if total.FrameHookOverrideCycles < 20 || total.FrameHookSubsetCycles < 20 {
		return total, errors.New("native frame-hook override minimum was not met")
	}
	if total.FrameHookBeginRunHits != 20 || total.FrameHookEndRunHits != 20 {
		return total, fmt.Errorf("begin_run delivered %d times and end_run %d, want exactly one per override cycle",
			total.FrameHookBeginRunHits, total.FrameHookEndRunHits)
	}
	if total.FrameHookRefusedFrames < 20 || total.FrameHookAdmittedFrames < 20 {
		return total, errors.New("the frame-hook scenario produced no refused or no admitted frames, so neither branch was exercised")
	}
	// end_draw is compared against the admitted frames of the runs that
	// actually installed it. The subset scenario declares no EndDraw at all,
	// so its admitted frames deliver draw and no end_draw hook -- which is the
	// uninstalled-member behaviour, not a skipped frame.
	if total.FrameHookEndDrawHits != total.FrameHookEndDrawExpected {
		return total, fmt.Errorf("end_draw arrived %d times for %d admitted frames on a Game that installed it; a refused frame must skip it",
			total.FrameHookEndDrawHits, total.FrameHookEndDrawExpected)
	}
	if total.FrameHookEndDrawExpected >= total.FrameHookAdmittedFrames {
		return total, errors.New("every admitted frame installed end_draw, so the uninstalled branch was never exercised")
	}
	if total.FrameHookBeginDrawHits != total.FrameHookAdmittedFrames+total.FrameHookRefusedFrames {
		return total, fmt.Errorf("begin_draw arrived %d times for %d admitted and %d refused frames",
			total.FrameHookBeginDrawHits, total.FrameHookAdmittedFrames, total.FrameHookRefusedFrames)
	}
	if total.FrameHookSkipChecks < 20 || total.FrameHookOrderChecks < 20 || total.FrameHookBaseCallChecks < 20 {
		return total, errors.New("the refused-frame, ordering or explicit-base-call proof did not run in every cycle")
	}
	// The whole point of the capability being per hook: a Game that declares
	// only BeginDraw must never receive the other three.
	if total.FrameHookUninstalledHits != 0 {
		return total, fmt.Errorf("%d hooks were delivered for capabilities the callback object never declared", total.FrameHookUninstalledHits)
	}
	// Foundation 42. Six timing and presentation settings reach the live native
	// loop, one is refused from a non-owner goroutine, and a Game configured
	// before Run is created with what it was configured with.
	if total.TimingCycles < 20 || total.TimingCreatedWithConfig < 20 {
		return total, errors.New("native timing minimum was not met")
	}
	if total.TimingSettersApplied != 6*total.TimingCycles {
		return total, fmt.Errorf("%d timing settings reached a live native game across %d cycles, want six per cycle",
			total.TimingSettersApplied, total.TimingCycles)
	}
	if total.TimingWrongThreadChecks < 20 || total.TimingRangeChecks < 20 {
		return total, errors.New("the timing thread or range proof did not run in every cycle")
	}
	// Foundation 45. Each window cycle measures the same members three times:
	// before Run with no native window, during Run with one, and after Run
	// with none again. The guarded and unguarded families must BOTH have been
	// exercised in every cycle, because the split between them is the whole
	// claim.
	if total.WindowCycles < 20 {
		return total, errors.New("native window minimum was not met")
	}
	if total.WindowIdentityChecks != 3*total.WindowCycles {
		return total, fmt.Errorf("Game.Window identity was checked %d times across %d cycles, want three per cycle",
			total.WindowIdentityChecks, total.WindowCycles)
	}
	if total.WindowGuardedFallbacks != 6*total.WindowCycles {
		return total, fmt.Errorf("%d guarded-fallback checks across %d cycles, want six per cycle",
			total.WindowGuardedFallbacks, total.WindowCycles)
	}
	if total.WindowUnguardedFailures != 4*total.WindowCycles {
		return total, fmt.Errorf("%d unguarded-failure checks across %d cycles, want four per cycle",
			total.WindowUnguardedFailures, total.WindowCycles)
	}
	if total.WindowLiveReads != 6*total.WindowCycles {
		return total, fmt.Errorf("%d live window reads across %d cycles, want six per cycle",
			total.WindowLiveReads, total.WindowCycles)
	}
	if total.WindowTitleSuppressions < 20 || total.WindowWrongThreadChecks < 20 {
		return total, errors.New("the window title suppression or thread proof did not run in every cycle")
	}
	if total.WindowScreenDeviceChanges < 20 {
		return total, errors.New("the screen-device-change pair did not run in every cycle")
	}
	// Foundation 47. The frame-step lifecycle: a native game that exists
	// without a loop, driven a frame at a time, and destroyed by Dispose.
	if total.FrameStepCycles < 20 {
		return total, errors.New("native frame-step minimum was not met")
	}
	// Six steps per cycle: two Ticks, two RunOneFrames, one suppressed Tick
	// and one post-Exit Tick.
	if total.FrameStepTicks != 4*total.FrameStepCycles || total.FrameStepRunOneFrames != 2*total.FrameStepCycles {
		return total, fmt.Errorf("%d ticks and %d run-one-frames across %d cycles, want four and two per cycle",
			total.FrameStepTicks, total.FrameStepRunOneFrames, total.FrameStepCycles)
	}
	// Initialization happens exactly ONCE per session however many frames are
	// stepped, and the two Ticks that precede the first RunOneFrame deliver
	// none of it.
	if total.FrameStepInitializations != total.FrameStepCycles {
		return total, fmt.Errorf("%d initializations across %d frame-step cycles, want exactly one per session",
			total.FrameStepInitializations, total.FrameStepCycles)
	}
	if total.FrameStepTickInitChecks < 20 {
		return total, errors.New("the proof that a tick does not initialize did not run in every cycle")
	}
	// Every step delivers exactly one Update; only the suppressed one and the
	// post-Exit one skip Draw.
	if total.FrameStepUpdates != 6*total.FrameStepCycles {
		return total, fmt.Errorf("%d updates across %d cycles, want six per cycle", total.FrameStepUpdates, total.FrameStepCycles)
	}
	if total.FrameStepDraws >= total.FrameStepUpdates {
		return total, fmt.Errorf("%d draws for %d updates; SuppressDraw skipped none", total.FrameStepDraws, total.FrameStepUpdates)
	}
	if total.FrameStepSuppressChecks < 20 || total.FrameStepExitChecks < 20 {
		return total, errors.New("the suppress-draw or exit proof did not run in every cycle")
	}
	if total.FrameStepWrongThreadChecks < 20 || total.FrameStepCallbackRefusals < 20 {
		return total, errors.New("the owner-thread or in-callback refusal proof did not run in every cycle")
	}
	if total.FrameStepSessionChecks < 60 || total.FrameStepDisposeChecks < 20 || total.FrameStepRecreationChecks < 20 {
		return total, errors.New("the session lifetime, disposal or recreation proof did not run in every cycle")
	}
	if total.FrameStepRunAfterStepCycle < 20 {
		return total, errors.New("Run did not adopt a standalone session in every cycle")
	}
	// Foundation 48. GraphicsDeviceManager's nine configuration setters push to
	// CNA's own manager, which is the object ApplyChanges reads.
	if total.ManagerCycles < 20 || total.ManagerDefaultChecks < 20 {
		return total, errors.New("native graphics-manager minimum was not met")
	}
	if total.ManagerSettersApplied != 6*total.ManagerCycles {
		return total, fmt.Errorf("%d framework-typed settings across %d cycles, want six per cycle",
			total.ManagerSettersApplied, total.ManagerCycles)
	}
	if total.ManagerCrossPackageSets != 3*total.ManagerCycles {
		return total, fmt.Errorf("%d cross-package settings across %d cycles, want three per cycle",
			total.ManagerCrossPackageSets, total.ManagerCycles)
	}
	if total.ManagerRangeChecks < 20 || total.ManagerApplyChanges < 20 ||
		total.ManagerToggleChecks < 20 || total.ManagerWrongThreadCheck < 20 {
		return total, errors.New("a graphics-manager range, apply, toggle or thread proof did not run in every cycle")
	}
	// Foundation 49. The manager registers itself under both service contracts,
	// which is what finally makes Game.GraphicsDevice and
	// DrawableGameComponent.Initialize work with no consumer-supplied service.
	if total.ManagerServiceChecks != 2*total.ManagerCycles {
		return total, fmt.Errorf("%d service-registration checks across %d cycles, want two per cycle",
			total.ManagerServiceChecks, total.ManagerCycles)
	}
	if total.ManagerDuplicateChecks < 20 || total.ManagerServiceRemovalCheck < 20 {
		return total, errors.New("the duplicate-registration or service-removal proof did not run in every cycle")
	}
	if total.ManagerGameDeviceChecks < 20 || total.ManagerDrawableChecks < 20 {
		return total, errors.New("Game.GraphicsDevice or DrawableGameComponent did not resolve the published service in every cycle")
	}
	if total.ManagerEventRaiseChecks != 4*total.ManagerCycles {
		return total, fmt.Errorf("%d manager raiser checks across %d cycles, want four per cycle",
			total.ManagerEventRaiseChecks, total.ManagerCycles)
	}
	// The device-created signal is the one HEADLESS actually produces, and it
	// must arrive at least once per cycle: it is what proves the native
	// subscription is installed and routed rather than merely accepted.
	if total.ManagerSignalDeviceCreated < total.ManagerCycles {
		return total, fmt.Errorf("%d device-created signals across %d cycles, want at least one per cycle",
			total.ManagerSignalDeviceCreated, total.ManagerCycles)
	}
	return total, nil
}

func decodeLastJSONLine(output []byte, value any) error {
	lines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
	for index := len(lines) - 1; index >= 0; index-- {
		line := bytes.TrimSpace(lines[index])
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		if err := json.Unmarshal(line, value); err == nil {
			return nil
		}
	}
	return errors.New("no JSON result line found")
}

func runChild(scenario string, index int) error {
	if scenario == "frame-hook-override" || scenario == "frame-hook-subset" {
		return runFrameHookChild(scenario)
	}
	if scenario == "timing" {
		return runTimingChild()
	}
	if scenario == "window" {
		return runWindowChild()
	}
	if scenario == "frame-step" {
		return runFrameStepChild()
	}
	if scenario == "frame-step-run" {
		return runFrameStepRunChild()
	}
	if scenario == "graphics-manager" {
		return runGraphicsManagerChild()
	}
	game := &stressGame{scenario: scenario, index: index, data: encodedPNG()}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	if err := game.subscribeGameEvents(host); err != nil {
		return err
	}
	if err := verifyKeyboardUnavailable("before Game.Run"); err != nil {
		return err
	}
	err = host.Run()
	game.recordGameEventDeliveries()
	if scenario == "event-rerun" {
		return runEventRerunChild(game, host, err)
	}
	if unavailableErr := verifyKeyboardUnavailable("after Game shutdown"); unavailableErr != nil {
		return unavailableErr
	}
	switch scenario {
	case "success":
		game.result.GameCycles = 1
		game.result.GameRecreationCycles = 1
		if err != nil {
			return err
		}
		if verifyErr := game.verifyGameEventDelivery(); verifyErr != nil {
			return verifyErr
		}
		if disposeErr := game.verifyManagedDisposalAfterRun(host); disposeErr != nil {
			return disposeErr
		}
		if _, staleErr := game.device.Viewport(); !errors.Is(staleErr, interop.ErrStaleGeneration) {
			game.result.ObservedUAF++
			return fmt.Errorf("stale graphics device was not rejected by generation: %w", staleErr)
		}
	case "device-state":
		game.result.DeviceStateCycles = 1
		if err != nil {
			return err
		}
		// Every facade from a finished run is stale, and the state members must
		// report that rather than reaching a device that is gone.
		if _, staleErr := game.device.BlendFactor(); !errors.Is(staleErr, interop.ErrStaleGeneration) {
			game.result.ObservedUAF++
			return fmt.Errorf("stale device BlendFactor was not rejected by generation: %w", staleErr)
		}
		game.result.DeviceStateStaleChecks++
	case "sprite-draw":
		game.result.SpriteDrawCycles = 1
		if err != nil {
			return err
		}
		if game.result.SpriteDrawScaledSubmits == 0 || game.result.SpriteDrawDestinationSubmits == 0 {
			return errors.New("the sprite-draw scenario submitted nothing")
		}
	case "render-target":
		game.result.RenderTargetCycles = 1
		if err != nil {
			return err
		}
		if game.result.RenderTargetCreations == 0 {
			return errors.New("the render-target scenario created nothing")
		}
		// Exactly one of the two outcomes, every cycle. A run reporting neither
		// never reached the bind, and one reporting both would mean the
		// scenario ran twice in a process that should host one cycle.
		if game.result.RenderTargetBinds+game.result.RenderTargetBindRefusals != game.result.RenderTargetCreations {
			return fmt.Errorf("render-target binds %d and refusals %d do not account for %d creations",
				game.result.RenderTargetBinds, game.result.RenderTargetBindRefusals, game.result.RenderTargetCreations)
		}
		// A renderer that BOUND must have produced the pixels, drawn them
		// through the Texture2D position and disposed cleanly. A renderer that
		// refused proves the description and the substitution and nothing more,
		// which is what BLOCKED_RENDERER means here.
		if game.result.RenderTargetBinds > 0 {
			if game.result.RenderTargetSpriteDraws == 0 || game.result.RenderTargetDisposalChecks == 0 {
				return errors.New("the render target bound and the semantic slice did not complete")
			}
			// The pixel check is the one step a renderer may be unable to
			// perform, so it is required to be accounted for rather than
			// required to have happened.
			if game.result.RenderTargetPixelChecks+game.result.RenderTargetReadbackRefusals != game.result.RenderTargetBinds {
				return fmt.Errorf("render-target pixel checks %d and readback refusals %d do not account for %d binds",
					game.result.RenderTargetPixelChecks, game.result.RenderTargetReadbackRefusals, game.result.RenderTargetBinds)
			}
		}
	case "callback-error":
		game.result.CallbackErrorCycles = 1
		if !errors.Is(err, callbackSentinel) {
			return fmt.Errorf("callback error was not returned from Run: %w", err)
		}
	case "callback-panic":
		game.result.CallbackPanicCycles = 1
		if err == nil {
			return errors.New("callback panic was not contained and returned")
		}
	default:
		return fmt.Errorf("unknown child scenario %q", scenario)
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// subscribeGameEvents registers on all four projected Game events BEFORE the
// native game exists, which is the point of the exercise: a Go consumer never
// waits for a native subscription, because the bridge installs exactly one per
// event when the native host is created and the Go registration list is
// entirely managed state.
//
// It also registers and immediately removes a fifth handler. That handler must
// never run, which is what proves removal reaches the delivery path rather than
// only the registration list.
func (g *stressGame) subscribeGameEvents(host *framework.Game) error {
	g.eventGoroutines = map[string]bool{}
	record := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, args *framework.EventArgs) error {
			g.eventOrder = append(g.eventOrder, name)
			if args != framework.EventArgsEmpty() {
				return fmt.Errorf("%s carried args that are not EventArgs.Empty", name)
			}
			// Exiting is the one event the reference raises with a null
			// sender; the other three raise with the Game.
			if name == "Exiting" {
				if sender != nil {
					return fmt.Errorf("Exiting sender = %v, want nil", sender)
				}
			} else if sender != any(host) {
				return fmt.Errorf("%s sender = %v, want the Game", name, sender)
			}
			g.eventGoroutines[currentGoroutineLabel()] = true
			return nil
		}
	}
	if _, err := host.AddActivatedHandler(record("Activated")); err != nil {
		return err
	}
	if _, err := host.AddDeactivatedHandler(record("Deactivated")); err != nil {
		return err
	}
	if _, err := host.AddExitingHandler(record("Exiting")); err != nil {
		return err
	}
	// Disposed does NOT join the ordered log. Its reference raise site is
	// managed Dispose(bool), so during a run it must never fire at all; this
	// handler exists to prove exactly that, and to fire when the run is over
	// and the consumer disposes on purpose.
	if _, err := host.AddDisposedHandler(func(sender any, args *framework.EventArgs) error {
		g.disposedRaises++
		if args != framework.EventArgsEmpty() {
			return fmt.Errorf("Disposed carried args that are not EventArgs.Empty")
		}
		if sender != any(host) {
			return fmt.Errorf("Disposed sender = %v, want the Game", sender)
		}
		g.eventGoroutines[currentGoroutineLabel()] = true
		return nil
	}); err != nil {
		return err
	}
	removed, err := host.AddExitingHandler(func(any, *framework.EventArgs) error {
		g.removedRan = true
		return nil
	})
	if err != nil {
		return err
	}
	if err := host.RemoveExitingHandler(removed); err != nil {
		return err
	}
	// Removing the same token twice, and removing it from a different event,
	// must both be inert.
	if err := host.RemoveExitingHandler(removed); err != nil {
		return err
	}
	return host.RemoveDisposedHandler(removed)
}

func (g *stressGame) recordGameEventDeliveries() {
	for _, name := range g.eventOrder {
		switch name {
		case "Activated":
			g.result.GameEventActivated++
		case "Deactivated":
			g.result.GameEventDeactivated++
		case "Exiting":
			g.result.GameEventExiting++
		}
	}
	// The native disposal signal is read from the internal runtime, which is
	// the only place it is observable now that it raises nothing public.
	if g.runtime != nil {
		g.result.GameNativeDisposalSignals = g.runtime.GameEventDeliveries()[interop.GameEventDisposed]
	}
	g.result.GameDisposedDuringRun = g.disposedRaises
	if !g.removedRan {
		g.result.GameEventRemovalChecks++
	}
	if len(g.eventGoroutines) == 1 && g.eventGoroutines[g.ownerGoroutine] {
		g.result.GameEventOwnerThreadHits++
	}
}

// verifyGameEventDelivery holds the exact ordering the pinned runtime produces
// for a clean run, measured against libcna_c_api.so:
//
//	initialize -> load_content -> begin_run -> ACTIVATED -> update/draw...
//	-> exiting callback -> EXITING -> end_run -> [cna_game_run returns]
//	-> unload_content -> DISPOSED -> [cna_game_destroy returns]
//
// The two facts that matter to the projection are that Exiting precedes
// Disposed and that each is delivered exactly once. Deactivated is NOT asserted:
// the qualification artifact runs a HEADLESS renderer with no window manager,
// so no focus transition away from the game can be produced, and inventing one
// to make a counter move would be fabricating evidence.
func (g *stressGame) verifyGameEventDelivery() error {
	if g.result.GameEventActivated != 1 {
		return fmt.Errorf("Activated delivered %d times, want exactly 1", g.result.GameEventActivated)
	}
	if g.result.GameEventExiting != 1 {
		return fmt.Errorf("Exiting delivered %d times, want exactly 1", g.result.GameEventExiting)
	}
	// The native disposal signal still arrives exactly once per run, from
	// inside cna_game_destroy, and that is what proves the four registrations
	// outlive native destruction. It raises nothing public.
	if g.result.GameNativeDisposalSignals != 1 {
		return fmt.Errorf("the native disposal signal arrived %d times, want exactly 1", g.result.GameNativeDisposalSignals)
	}
	if g.result.GameDisposedDuringRun != 0 {
		return fmt.Errorf("Game.Disposed was raised %d times during a run; its only reference raise site is managed Dispose(bool)", g.result.GameDisposedDuringRun)
	}
	exiting := -1
	for i, name := range g.eventOrder {
		if name == "Exiting" {
			exiting = i
		}
	}
	if exiting < 0 {
		return fmt.Errorf("delivery order %v contains no Exiting", g.eventOrder)
	}
	if g.eventOrder[0] != "Activated" {
		return fmt.Errorf("delivery order %v does not start with Activated", g.eventOrder)
	}
	g.result.GameEventOrderChecks++
	if g.removedRan {
		return errors.New("a handler removed before Run was still delivered")
	}
	if g.result.GameEventOwnerThreadHits != 1 {
		return fmt.Errorf("game events were delivered on %d distinct goroutines, want the owner goroutine only", len(g.eventGoroutines))
	}
	return nil
}

// verifyManagedDisposalAfterRun is the other half of the Foundation 39
// correction, proved against a Game whose native generation is already gone.
//
// Three facts, none of which the old native-signal binding could have produced:
//
//  1. Disposing AFTER the run raises Game.Disposed. The reference's raise site
//     is managed Dispose(bool), and managed state outlives the native host, so
//     a consumer who disposes when the run is over still gets the event.
//  2. It raises with the Game as sender and EventArgs.Empty as args, checked
//     by the handler itself.
//  3. Dispose is NOT idempotent. A second call raises again, because Game
//     carries no disposed flag anywhere. A projection that invented one would
//     report one raise here instead of two.
//
// Nothing here reaches native code. There is no live handle left and none is
// fabricated: the whole body is managed component and event work.
func (g *stressGame) verifyManagedDisposalAfterRun(host *framework.Game) error {
	before := g.disposedRaises
	if before != 0 {
		return fmt.Errorf("Game.Disposed had already been raised %d times before any Dispose call", before)
	}
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose after the run: %w", err)
	}
	if g.disposedRaises != 1 {
		return fmt.Errorf("one managed Dispose raised Game.Disposed %d times, want 1", g.disposedRaises)
	}
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("second Dispose after the run: %w", err)
	}
	if g.disposedRaises != 2 {
		return fmt.Errorf("a second Dispose raised Game.Disposed %d times in total, want 2; Game has no disposed flag", g.disposedRaises)
	}
	// Dispose(false) is the finalizer path and does nothing at all.
	if err := host.DisposeByBoolean(false); err != nil {
		return fmt.Errorf("Dispose(false) after the run: %w", err)
	}
	if err := host.Finalize(); err != nil {
		return fmt.Errorf("Finalize after the run: %w", err)
	}
	if g.disposedRaises != 2 {
		return fmt.Errorf("Dispose(false) or Finalize raised Game.Disposed; total is %d, want 2", g.disposedRaises)
	}
	// The native disposal signal count did not move: managed disposal reaches
	// no native code at all.
	if g.runtime != nil {
		if signals := g.runtime.GameEventDeliveries()[interop.GameEventDisposed]; signals != g.result.GameNativeDisposalSignals {
			return fmt.Errorf("managed disposal changed the native disposal signal count from %d to %d",
				g.result.GameNativeDisposalSignals, signals)
		}
	}
	g.result.GameDisposedByManagedCall = g.disposedRaises
	g.result.GameDisposedRepeatChecks++
	g.result.GameDisposeAfterRunCycles++
	return nil
}

// currentGoroutineLabel identifies the delivering goroutine without exposing
// any runtime identity beyond this file. Every native game event must arrive on
// the same locked owner goroutine that entered cna_game_run.
func currentGoroutineLabel() string {
	buffer := make([]byte, 64)
	buffer = buffer[:runtime.Stack(buffer, false)]
	line := string(buffer)
	if index := bytes.IndexByte([]byte(line), '['); index > 0 {
		return line[:index]
	}
	return line
}

// runEventRerunChild proves the native subscription lifetime claim end to end:
// the four registrations installed for one run are released when that run ends,
// a SECOND Run on the same Go Game installs four fresh ones, and the projected
// accessors keep working across both without a consumer resubscribing.
//
// It also records the one behavior a second run makes visible. Game::isActive
// is a private field the reference never resets, and the two activation events
// are edge-triggered on it, so a second run's activation signal is SUPPRESSED --
// the game was already active as far as the managed half is concerned. That is
// what HostActivated does in CLR too, and it is recorded here rather than
// smoothed over.
func runEventRerunChild(game *stressGame, host *framework.Game, firstErr error) error {
	if firstErr != nil {
		return fmt.Errorf("first run: %w", firstErr)
	}
	first := game.result
	if first.GameEventActivated != 1 || first.GameEventExiting != 1 || first.GameNativeDisposalSignals != 1 {
		return fmt.Errorf("first run delivered %v and %d native disposal signals", game.eventOrder, first.GameNativeDisposalSignals)
	}
	if first.GameDisposedDuringRun != 0 {
		return fmt.Errorf("the first run raised Game.Disposed %d times", first.GameDisposedDuringRun)
	}

	// Add and remove handlers with no native game alive at all. Both are pure
	// managed list work, so both must be ordinary successes.
	postRun, err := host.AddExitingHandler(func(any, *framework.EventArgs) error { return nil })
	if err != nil {
		return fmt.Errorf("subscribe after the run ended: %w", err)
	}
	if err := host.RemoveExitingHandler(postRun); err != nil {
		return fmt.Errorf("unsubscribe after the run ended: %w", err)
	}
	game.result.GameEventPostRunChecks++

	game.eventOrder = nil
	game.eventGoroutines = map[string]bool{}
	game.manager = nil
	game.device = nil
	if err := host.Run(); err != nil {
		return fmt.Errorf("second run: %w", err)
	}
	second := game.eventOrder
	activated, exiting := 0, 0
	for _, name := range second {
		switch name {
		case "Activated":
			activated++
		case "Exiting":
			exiting++
		}
	}
	if exiting != 1 {
		return fmt.Errorf("second run delivered %v; Exiting must arrive exactly once", second)
	}
	// Two runs, two native destructions, two disposal signals -- and still no
	// public Disposed raise, because nobody disposed anything.
	signals := 0
	if game.runtime != nil {
		signals = game.runtime.GameEventDeliveries()[interop.GameEventDisposed]
	}
	if signals != 2 {
		return fmt.Errorf("two runs produced %d native disposal signals, want 2", signals)
	}
	if game.disposedRaises != 0 {
		return fmt.Errorf("two runs raised Game.Disposed %d times with no Dispose call", game.disposedRaises)
	}
	// The edge-trigger guard is the whole point: the managed half never saw a
	// deactivation, so the second run's activation signal raises nothing.
	if activated != 0 {
		return fmt.Errorf("second run raised Activated %d times; isActive was already true, so the guard must suppress it", activated)
	}
	if len(game.eventGoroutines) != 1 || !game.eventGoroutines[game.ownerGoroutine] {
		return fmt.Errorf("second run delivered on %d goroutines", len(game.eventGoroutines))
	}
	// The counter is refreshed to the two-run total BEFORE managed disposal is
	// proved, because that proof asserts managed disposal does not move it.
	game.result.GameNativeDisposalSignals = signals
	if disposeErr := game.verifyManagedDisposalAfterRun(host); disposeErr != nil {
		return disposeErr
	}
	game.result.GameEventRerunCycles++
	game.result.GameEventActivated = first.GameEventActivated + activated
	game.result.GameEventExiting = first.GameEventExiting + exiting
	game.result.GameDisposedDuringRun = 0
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

func (g *stressGame) Initialize(host *framework.Game) error {
	g.ownerGoroutine = currentGoroutineLabel()
	if current, ok := interop.CurrentRuntime(); ok {
		g.runtime = current
	}
	manager, err := framework.NewGraphicsDeviceManager(host)
	if err != nil {
		return err
	}
	g.manager = manager
	if got := manager.SupportedOrientations(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("initial SupportedOrientations = %d, want Default", got)
	}
	configured := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	manager.SetSupportedOrientations(configured)
	if got := manager.SupportedOrientations(); got != configured {
		return fmt.Errorf("Initialize SupportedOrientations = %d, want %d", got, configured)
	}
	return nil
}

func (g *stressGame) LoadContent(_ *framework.Game) error {
	configured := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	if got := g.manager.SupportedOrientations(); got != configured {
		return fmt.Errorf("LoadContent SupportedOrientations = %d, want %d", got, configured)
	}
	g.manager.SetSupportedOrientations(configured)
	unknown := framework.DisplayOrientation(1 << 20)
	g.manager.SetSupportedOrientations(unknown)
	if got := g.manager.SupportedOrientations(); got != unknown {
		return fmt.Errorf("raw SupportedOrientations = %d, want %d", got, unknown)
	}
	if err := verifyKeyboardPlayerIndexSnapshots(); err != nil {
		return err
	}
	device, err := graphics.GraphicsDeviceManagerGraphicsDevice(g.manager)
	if err != nil {
		return err
	}
	g.device = device
	if g.scenario == "sprite-draw" {
		texture, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
		if err != nil {
			return err
		}
		batch, err := graphics.NewSpriteBatch(device)
		if err != nil {
			return err
		}
		g.spriteTexture = texture
		g.spriteBatch = batch
		// The two guards are checked HERE, outside any begin/end pair, which is
		// the state InternalDraw's second throw is about. A nil texture is
		// checked first because the reference checks it first: its
		// ArgumentNullException is thrown before the pair is even read, so this
		// call is outside a pair AND has a nil texture, and must report the
		// argument.
		nullErr := batch.DrawByTexture2DAndVector2AndColor(nil, framework.Vector2{}, framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255))
		if nullErr == nil || !strings.Contains(nullErr.Error(), "This method does not accept null for this parameter.") {
			return fmt.Errorf("nil-texture Draw outside a pair = %v, want the ArgumentNullException message", nullErr)
		}
		g.result.SpriteDrawNullTextureChecks++
		outsideErr := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255))
		if outsideErr == nil || !strings.Contains(outsideErr.Error(), "Begin must be called successfully before a Draw can be called.") {
			return fmt.Errorf("Draw outside a pair = %v, want the InvalidOperationException message", outsideErr)
		}
		g.result.SpriteDrawOutsidePairChecks++
		endErr := batch.End()
		if endErr == nil || !strings.Contains(endErr.Error(), "Begin must be called successfully before End can be called.") {
			return fmt.Errorf("End outside a pair = %v, want the InvalidOperationException message", endErr)
		}
		g.result.SpriteDrawPairGuardChecks++
		bounds := texture.Bounds()
		width := texture.Width()
		height := texture.Height()
		if bounds.X != 0 || bounds.Y != 0 || bounds.Width != width || bounds.Height != height {
			return fmt.Errorf("Bounds = %+v, want (0,0,%d,%d)", bounds, width, height)
		}
		g.result.SpriteDrawBoundsChecks++
		return nil
	}
	if g.scenario != "success" || g.index != 0 {
		return nil
	}
	for cycle := 0; cycle < 20; cycle++ {
		texture, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
		if err != nil {
			return err
		}
		batch, err := graphics.NewSpriteBatch(device)
		if err != nil {
			return err
		}
		if cycle == 0 {
			if err := g.manager.Dispose(true); !errors.Is(err, interop.ErrChildrenAlive) {
				return fmt.Errorf("parent-before-child result: %w", err)
			}
			wrongThread := make(chan error, 1)
			go func() { wrongThread <- texture.DisposeByBoolean(true) }()
			if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
				return fmt.Errorf("wrong-thread texture disposal result: %w", err)
			}
			g.result.WrongThreadChecks++
			if err := texture.DisposeByBoolean(true); err != nil {
				return fmt.Errorf("owner-thread retry: %w", err)
			}
			g.result.OwnerThreadRetries++
		} else if cycle == 1 {
			// # The inherited Dispose() must reach the DERIVED body
			//
			// GraphicsResource::Dispose() is `callvirt Dispose(bool)`, which
			// dispatches to Texture2D's override and releases the native
			// texture. A Go forwarding that called the COMPOSED BASE's
			// DisposeByNone instead would set the flag, raise Disposing and
			// leak the CNA texture -- and every managed observable would agree
			// with the correct version, because the flag and the event are the
			// base's either way.
			//
			// The one observable that disagrees is the native handle. After the
			// derived body runs, the resource is destroyed and a later
			// operation reports ErrDisposed; after the base body runs, the
			// handle is still live and the same operation SUCCEEDS. That is why
			// this control lives here and not in a managed unit test.
			if err := texture.DisposeByNone(); err != nil {
				return fmt.Errorf("inherited Dispose(): %w", err)
			}
			if !texture.IsDisposed() {
				return errors.New("inherited Dispose() left IsDisposed false; the base half runs in a finally")
			}
			var sink bytes.Buffer
			if err := texture.SaveAsPng(&sink, 4, 4); !errors.Is(err, interop.ErrDisposed) {
				return fmt.Errorf("SaveAsPng after the inherited Dispose() = %v, want ErrDisposed: "+
					"the native texture is still alive, so Dispose() reached the composed base's slot instead of Texture2D's override", err)
			}
			g.result.InheritedDisposeVirtualChecks++
		} else if err := texture.DisposeByBoolean(true); err != nil {
			return err
		}
		if err := texture.DisposeByBoolean(true); err != nil {
			g.result.ObservedDoubleFree++
			return fmt.Errorf("double texture Dispose was not idempotent: %w", err)
		}
		if err := batch.DisposeByBoolean(true); err != nil {
			return err
		}
		if err := batch.DisposeByBoolean(true); err != nil {
			g.result.ObservedDoubleFree++
			return fmt.Errorf("double SpriteBatch Dispose was not idempotent: %w", err)
		}
		g.result.TextureCycles++
		g.result.SpriteBatchCycles++
		runtime.GC()
		g.result.GCStressPoints++
	}
	return nil
}

// removeDeviceHandler is the removal half of the event loop above, split out
// because Go has no way to name a method value's receiver twice in a map.
func removeDeviceHandler(device *graphics.GraphicsDevice, name string, subscription framework.EventSubscription) error {
	switch name {
	case "Disposing":
		return device.RemoveDisposingHandler(subscription)
	case "DeviceLost":
		return device.RemoveDeviceLostHandler(subscription)
	case "DeviceReset":
		return device.RemoveDeviceResetHandler(subscription)
	case "DeviceResetting":
		return device.RemoveDeviceResettingHandler(subscription)
	}
	return fmt.Errorf("unknown device event %q", name)
}

func verifyKeyboardPlayerIndexSnapshots() error {
	baseline, err := input.KeyboardGetStateByNone()
	if err != nil {
		return fmt.Errorf("read process keyboard baseline: %w", err)
	}
	for _, playerIndex := range playerIndexFixtures() {
		state, err := input.KeyboardGetStateByPlayerIndex(playerIndex)
		if err != nil {
			return fmt.Errorf("read process keyboard state for PlayerIndex(%d): %w", playerIndex, err)
		}
		if !input.KeyboardStateOperatorEqualityByKeyboardStateAndKeyboardState(baseline, state) {
			return fmt.Errorf("PlayerIndex(%d) selected a different keyboard snapshot", playerIndex)
		}
	}
	return nil
}

func verifyKeyboardUnavailable(stage string) error {
	baseline, baselineError := input.KeyboardGetStateByNone()
	if baselineError == nil {
		return fmt.Errorf("KeyboardGetStateByNone unexpectedly succeeded %s", stage)
	}
	for _, playerIndex := range playerIndexFixtures() {
		state, err := input.KeyboardGetStateByPlayerIndex(playerIndex)
		if err == nil {
			return fmt.Errorf("KeyboardGetStateByPlayerIndex(%d) unexpectedly succeeded %s", playerIndex, stage)
		}
		if state != baseline || err.Error() != baselineError.Error() {
			return fmt.Errorf("KeyboardGetStateByPlayerIndex(%d) used a different unavailable-runtime path %s", playerIndex, stage)
		}
	}
	return nil
}

func playerIndexFixtures() []framework.PlayerIndex {
	return []framework.PlayerIndex{
		framework.PlayerIndexOne,
		framework.PlayerIndexTwo,
		framework.PlayerIndexThree,
		framework.PlayerIndexFour,
		framework.PlayerIndex(12345),
	}
}

func (g *stressGame) Update(_ *framework.Game, _ framework.GameTime) error {
	switch g.scenario {
	case "callback-error":
		return callbackSentinel
	case "callback-panic":
		panic("native stress callback panic")
	default:
		return nil
	}
}

func (g *stressGame) Draw(host *framework.Game, _ framework.GameTime) error {
	if g.scenario == "sprite-draw" {
		if err := g.drawEverySpriteOverload(); err != nil {
			return err
		}
	}
	if g.scenario == "device-state" {
		if err := g.exerciseDeviceState(); err != nil {
			return err
		}
	}
	if g.scenario == "render-target" {
		if err := g.exerciseRenderTarget(); err != nil {
			return err
		}
	}
	return host.Exit()
}

// isNativeRefusal reports whether an error is CNA answering "this renderer
// cannot do that" rather than the binding getting something wrong.
//
// It matches on the CNA result code, not on the message: the message is
// documentation and may be reworded, while CNA_RESULT_NOT_SUPPORTED is part of
// the ABI. A refusal is recorded as a renderer limitation; anything else is a
// defect and fails the scenario.
func isNativeRefusal(err error) bool {
	var native *interop.NativeError
	return errors.As(err, &native) && native.Code == 6
}

// renderTargetClearColor is the deterministic content the render-target
// semantic test writes and reads back. It is a colour no other value in the
// scenario produces, so a readback that matched by accident would have to
// produce these exact four bytes.
var renderTargetClearColor = framework.NewColorByInt32AndInt32AndInt32AndInt32(203, 67, 21, 255)

// exerciseRenderTarget is the render-target semantic slice, end to end, inside
// a draw callback -- the only moment CNA lends a device handle out.
//
//	create -> bind -> clear -> unbind -> read back THROUGH THE Texture2D SURFACE
//
// The last step is the point. `Texture2DGetDataBySliceOfT` takes a
// Texture2DReference, and passing a *RenderTarget2D to it is the Go spelling of
// C#'s `renderTarget` flowing into a Texture2D position. It proves the
// substitution and the render-target contents with one call, because the pixels
// come back through the base's member.
//
// # Two renderers, two outcomes, both recorded
//
// CNA permits a render target to be CREATED on a backend with no real
// off-screen storage: creation succeeds, RendererAvailable is false, and
// binding reports NOT_SUPPORTED. The HEADLESS artifact is such a backend and the
// SOFTWARE one is not, so this scenario asserts what each can actually do and
// counts them separately. A binding that failed on a renderer that HAS storage
// is a defect; one that failed on a renderer that does not is the documented
// contract.
func (g *stressGame) exerciseRenderTarget() error {
	device := g.device
	const size = 8

	target, err := graphics.NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32(device, size, size)
	if err != nil {
		return fmt.Errorf("NewRenderTarget2D: %w", err)
	}
	g.result.RenderTargetCreations++

	// The Texture2D half of its surface answers before anything is bound, from
	// the description CNA applied.
	if target.Width() != size || target.Height() != size {
		return fmt.Errorf("render target is %dx%d, want %dx%d", target.Width(), target.Height(), size, size)
	}
	if target.Bounds() != framework.NewRectangle(0, 0, size, size) {
		return fmt.Errorf("render target Bounds = %+v", target.Bounds())
	}
	if target.LevelCount() < 1 {
		return fmt.Errorf("render target LevelCount = %d", target.LevelCount())
	}
	if got := target.ToString(); got != "Microsoft.Xna.Framework.Graphics.RenderTarget2D" {
		return fmt.Errorf("render target ToString = %q; the CLR `this` must reach the outermost object across three composition links", got)
	}
	if target.RenderTargetUsage() != graphics.RenderTargetUsageDiscardContents {
		return fmt.Errorf("render target usage = %v, want the constructor's DiscardContents default", target.RenderTargetUsage())
	}
	g.result.RenderTargetDescriptionChecks++

	// It satisfies the Texture2D parameter position, which is the substitution
	// under test. The assignment is the proof; it does not compile otherwise.
	var asTexture graphics.Texture2DReference = target
	if asTexture == nil {
		return errors.New("a RenderTarget2D does not satisfy Texture2DReference")
	}
	g.result.RenderTargetSubstitutionChecks++

	bindErr := device.SetRenderTargetByRenderTarget2D(target)
	if bindErr != nil {
		// The documented refusal on a backend with no off-screen storage. It is
		// recorded rather than treated as a pass or a failure.
		g.result.RenderTargetBindRefusals++
		fmt.Fprintf(os.Stderr, "render-target bind refused: %v\n", bindErr)
		return target.DisposeByNone()
	}
	g.result.RenderTargetBinds++

	if err := device.ClearByColor(renderTargetClearColor); err != nil {
		return fmt.Errorf("Clear into the render target: %w", err)
	}
	if err := device.SetRenderTargetByRenderTarget2D(nil); err != nil {
		return fmt.Errorf("restore the back buffer: %w", err)
	}
	g.result.RenderTargetUnbinds++

	// The readback, through the BASE's member. It is the only step that needs
	// the renderer to be able to copy a colour attachment back to the CPU, and
	// that is a per-renderer capability rather than a per-binding one: the
	// HEADLESS artifact binds and clears and then refuses this with
	//
	//	Texture2D::GetData: this graphics renderer cannot read a render
	//	target's colour attachment back to the CPU
	//
	// so the refusal is counted and the slice continues. A pixel check is
	// evidence only where the renderer can produce one.
	pixels := make([]framework.Color, size*size)
	readErr := graphics.Texture2DGetDataBySliceOfT(target, pixels)
	switch {
	case readErr == nil:
		for index, pixel := range pixels {
			if pixel != renderTargetClearColor {
				return fmt.Errorf("render target pixel %d = %+v, want the cleared %+v", index, pixel, renderTargetClearColor)
			}
		}
		g.result.RenderTargetPixelChecks++
	case isNativeRefusal(readErr):
		g.result.RenderTargetReadbackRefusals++
		fmt.Fprintf(os.Stderr, "render-target readback refused: %v\n", readErr)
	default:
		return fmt.Errorf("GetData through the Texture2D surface: %w", readErr)
	}

	// And a SpriteBatch draws it, which is the seven live substitutability
	// positions exercised rather than only asserted.
	batch, batchErr := graphics.NewSpriteBatch(device)
	if batchErr != nil {
		return fmt.Errorf("NewSpriteBatch: %w", batchErr)
	}
	if err := batch.BeginByNone(); err != nil {
		return fmt.Errorf("Begin: %w", err)
	}
	if err := batch.DrawByTexture2DAndVector2AndColor(target, framework.Vector2{X: 1, Y: 1},
		framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)); err != nil {
		return fmt.Errorf("Draw a render target as a texture: %w", err)
	}
	if err := batch.End(); err != nil {
		return fmt.Errorf("End: %w", err)
	}
	g.result.RenderTargetSpriteDraws++
	if err := batch.DisposeByNone(); err != nil {
		return fmt.Errorf("SpriteBatch disposal: %w", err)
	}

	// Disposal through the INHERITED member, and the idempotence the
	// GraphicsResource flag gives it.
	disposals := 0
	if _, err := target.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		disposals++
		if sender != any(target) {
			return errors.New("Disposing announced something other than the render target")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("AddDisposingHandler: %w", err)
	}
	if err := target.DisposeByNone(); err != nil {
		return fmt.Errorf("render target disposal: %w", err)
	}
	if err := target.DisposeByNone(); err != nil {
		return fmt.Errorf("second render target disposal: %w", err)
	}
	if disposals != 1 || !target.IsDisposed() {
		return fmt.Errorf("Disposing raised %d times, IsDisposed=%t", disposals, target.IsDisposed())
	}
	g.result.RenderTargetDisposalChecks++
	return nil
}

// exerciseDeviceState round-trips GraphicsDevice's render state through the
// LIVE device, inside a draw callback, which is the only moment CNA lends a
// device handle out.
//
// Every write is read back from the device rather than from anything CNA-Go
// holds, which is the whole point: the reference answers these getters from a
// managed cache its own constructor fills, CNA-Go cannot fill one because it
// does not create the device, and a projection that cached anyway would pass a
// test that only compared what it had just been given.
func (g *stressGame) exerciseDeviceState() error {
	device := g.device

	factor := framework.NewColorByInt32AndInt32AndInt32AndInt32(12, 34, 56, 78)
	if err := device.SetBlendFactor(factor); err != nil {
		return fmt.Errorf("SetBlendFactor: %w", err)
	}
	readFactor, err := device.BlendFactor()
	if err != nil {
		return fmt.Errorf("BlendFactor: %w", err)
	}
	if readFactor != factor {
		return fmt.Errorf("BlendFactor round trip = %+v, want %+v", readFactor, factor)
	}
	g.result.DeviceStateRoundTrips++

	// Foundation 60. The three state-object properties, through the LIVE
	// device: a null is the reference's ArgumentNullException, a set pushes the
	// descriptor to CNA and FREEZES the object, and the getter answers with the
	// very object that was set rather than with a fresh one built from values.
	if _, initial := device.BlendState(); initial != nil {
		return fmt.Errorf("BlendState on a live device: %w", initial)
	}
	if refusal := device.SetBlendState(nil); refusal == nil ||
		!strings.Contains(refusal.Error(), "This method does not accept null for this parameter.") {
		return fmt.Errorf("SetBlendState(nil) = %v, want the ArgumentNullException message", refusal)
	}
	g.result.DeviceStateObjectRefusals++

	ownBlend := graphics.NewBlendState()
	if err := ownBlend.SetColorSourceBlend(graphics.BlendSourceAlpha); err != nil {
		return fmt.Errorf("a fresh BlendState refused a write: %w", err)
	}
	if err := device.SetBlendState(ownBlend); err != nil {
		return fmt.Errorf("SetBlendState: %w", err)
	}
	readBlend, readErr := device.BlendState()
	if readErr != nil {
		return fmt.Errorf("BlendState: %w", readErr)
	}
	if readBlend != ownBlend {
		return errors.New("BlendState returned a different object; the getter answers with the one the setter was given")
	}
	if err := ownBlend.SetColorSourceBlend(graphics.BlendOne); err == nil {
		return errors.New("a bound BlendState accepted a write; Apply raises isBound")
	}
	if ownBlend.GraphicsDevice() != device {
		return errors.New("Apply did not store the device as the state's parent")
	}
	g.result.DeviceStateObjectBinds++

	if err := device.SetDepthStencilState(graphics.DepthStencilStateDepthRead()); err != nil {
		return fmt.Errorf("SetDepthStencilState: %w", err)
	}
	if err := device.SetRasterizerState(graphics.RasterizerStateCullNone()); err != nil {
		return fmt.Errorf("SetRasterizerState: %w", err)
	}
	readDepth, _ := device.DepthStencilState()
	readRaster, _ := device.RasterizerState()
	if readDepth != graphics.DepthStencilStateDepthRead() || readRaster != graphics.RasterizerStateCullNone() {
		return errors.New("the device did not cache the preset objects it was given")
	}
	g.result.DeviceStateObjectBinds += 2

	// And the two state-carrying Begin overloads, which reach CNA through one
	// route carrying all four descriptors and then perform SetRenderState's
	// managed half.
	stateBatch, stateBatchErr := graphics.NewSpriteBatch(device)
	if stateBatchErr != nil {
		return fmt.Errorf("NewSpriteBatch: %w", stateBatchErr)
	}
	if err := stateBatch.BeginBySpriteSortModeAndBlendState(
		graphics.SpriteSortModeDeferred, graphics.BlendStateAdditive()); err != nil {
		return fmt.Errorf("Begin(sortMode, blendState): %w", err)
	}
	if bound, _ := device.BlendState(); bound != graphics.BlendStateAdditive() {
		return errors.New("Begin did not apply its blend state to the device")
	}
	// SetRenderState substitutes the reference's defaults for the three nulls.
	if depth, _ := device.DepthStencilState(); depth != graphics.DepthStencilStateNone() {
		return errors.New("Begin did not substitute DepthStencilState.None for a null")
	}
	if raster, _ := device.RasterizerState(); raster != graphics.RasterizerStateCullCounterClockwise() {
		return errors.New("Begin did not substitute RasterizerState.CullCounterClockwise for a null")
	}
	if err := stateBatch.End(); err != nil {
		return fmt.Errorf("End after a state Begin: %w", err)
	}
	g.result.SpriteBatchStateBegins++

	if err := stateBatch.BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerState(
		graphics.SpriteSortModeDeferred, graphics.BlendStateOpaque(), graphics.SamplerStatePointClamp(),
		graphics.DepthStencilStateDefault(), graphics.RasterizerStateCullClockwise()); err != nil {
		return fmt.Errorf("Begin with four states: %w", err)
	}
	if err := stateBatch.End(); err != nil {
		return fmt.Errorf("End after the four-state Begin: %w", err)
	}
	g.result.SpriteBatchStateBegins++
	if err := stateBatch.DisposeByNone(); err != nil {
		return fmt.Errorf("state SpriteBatch disposal: %w", err)
	}

	// Foundation 61. The four collections, through the LIVE device.
	textures, texturesErr := device.Textures()
	if texturesErr != nil {
		return fmt.Errorf("Textures: %w", texturesErr)
	}
	again, _ := device.Textures()
	if textures != again {
		return errors.New("Textures returned two objects; the reference holds one per device")
	}
	vertexTextures, _ := device.VertexTextures()
	if vertexTextures == textures {
		return errors.New("Textures and VertexTextures are the same collection")
	}
	g.result.DeviceCollectionIdentityChecks++

	// The refusal must be the PROJECTION's ArgumentOutOfRangeException, not
	// CNA's own out-of-range answer: an off-by-one bound would still refuse,
	// through the wrong guard, with the wrong identity.
	for _, index := range []int32{-1, 16} {
		_, readErr := textures.Item(index)
		writeErr := textures.SetItem(index, nil)
		for name, err := range map[string]error{"get": readErr, "set": writeErr} {
			if err == nil {
				return fmt.Errorf("Textures[%d] %s was accepted; the indexer refuses out of range", index, name)
			}
			if !strings.Contains(err.Error(), "index is out of range") {
				return fmt.Errorf("Textures[%d] %s = %v, want the projection's ArgumentOutOfRangeException", index, name, err)
			}
		}
	}
	g.result.DeviceCollectionRangeChecks++

	// An empty slot answers nil, a bound one answers the texture that was
	// bound, and unbinding empties it again.
	slotTexture, slotErr := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
	if slotErr != nil {
		return fmt.Errorf("slot texture: %w", slotErr)
	}
	if err := textures.SetItem(0, slotTexture); err != nil {
		return fmt.Errorf("Textures[0] = texture: %w", err)
	}
	readSlot, readSlotErr := textures.Item(0)
	if readSlotErr != nil {
		return fmt.Errorf("Textures[0]: %w", readSlotErr)
	}
	if readSlot == nil {
		return errors.New("Textures[0] answered nil for a slot it had just bound")
	}
	if readSlot.Format() != slotTexture.Format() || readSlot.LevelCount() != slotTexture.LevelCount() {
		return errors.New("Textures[0] answered with a different texture")
	}
	if err := textures.SetItem(0, nil); err != nil {
		return fmt.Errorf("Textures[0] = nil: %w", err)
	}
	if empty, err := textures.Item(0); err != nil || empty != nil {
		return fmt.Errorf("Textures[0] after unbinding = %v, %v; want nil and no error", empty, err)
	}
	g.result.DeviceCollectionTextureRoundTrips++
	if err := slotTexture.DisposeByNone(); err != nil {
		return fmt.Errorf("slot texture disposal: %w", err)
	}

	samplers, samplersErr := device.SamplerStates()
	if samplersErr != nil {
		return fmt.Errorf("SamplerStates: %w", samplersErr)
	}
	// A slot nothing has written answers with what CNA reports, materialised.
	reported, reportedErr := samplers.Item(0)
	if reportedErr != nil || reported == nil {
		return fmt.Errorf("SamplerStates[0] = %v, %v", reported, reportedErr)
	}
	if err := samplers.SetItem(0, graphics.SamplerStatePointClamp()); err != nil {
		return fmt.Errorf("SamplerStates[0] = PointClamp: %w", err)
	}
	if bound, err := samplers.Item(0); err != nil || bound != graphics.SamplerStatePointClamp() {
		return fmt.Errorf("SamplerStates[0] after setting = %v, %v; the getter answers with the object that was set", bound, err)
	}
	if refusal := samplers.SetItem(0, nil); refusal == nil {
		return errors.New("SamplerStates[0] = nil was accepted")
	}
	g.result.DeviceCollectionSamplerRoundTrips++

	// Foundation 62. The six device events, subscribed against the LIVE device.
	// Registration is what is proved here: CNA raises DeviceLost, DeviceReset
	// and DeviceResetting only when a renderer really loses or resets a device,
	// which neither qualified artifact can be made to do, and it raises
	// Disposing from a disposal this scenario must not perform on a device the
	// Game owns and goes on using.
	deviceRaises := 0
	subscriptions := 0
	for name, add := range map[string]func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error){
		"Disposing": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDisposingHandler(h)
		},
		"DeviceLost": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceLostHandler(h)
		},
		"DeviceReset": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceResetHandler(h)
		},
		"DeviceResetting": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceResettingHandler(h)
		},
	} {
		subscription, err := add(func(sender any, args *framework.EventArgs) error {
			deviceRaises++
			return nil
		})
		if err != nil {
			return fmt.Errorf("Add%sHandler: %w", name, err)
		}
		subscriptions++
		if err := removeDeviceHandler(device, name, subscription); err != nil {
			return fmt.Errorf("Remove%sHandler: %w", name, err)
		}
	}
	createdSubscription, createdErr := device.AddResourceCreatedHandler(
		func(sender any, args *graphics.ResourceCreatedEventArgs) error { return nil })
	if createdErr != nil {
		return fmt.Errorf("AddResourceCreatedHandler: %w", createdErr)
	}
	destroyedSubscription, destroyedErr := device.AddResourceDestroyedHandler(
		func(sender any, args *graphics.ResourceDestroyedEventArgs) error { return nil })
	if destroyedErr != nil {
		return fmt.Errorf("AddResourceDestroyedHandler: %w", destroyedErr)
	}
	subscriptions += 2
	if err := device.RemoveResourceCreatedHandler(createdSubscription); err != nil {
		return fmt.Errorf("RemoveResourceCreatedHandler: %w", err)
	}
	if err := device.RemoveResourceDestroyedHandler(destroyedSubscription); err != nil {
		return fmt.Errorf("RemoveResourceDestroyedHandler: %w", err)
	}
	if subscriptions != 6 {
		return fmt.Errorf("%d device event subscriptions, want six", subscriptions)
	}
	g.result.DeviceEventSubscriptions += subscriptions
	// Every registration must be released before cna_game_destroy succeeds, and
	// the manager's disposal is what does it. A leaked registration would make
	// the whole isolated cycle fail at teardown, which is the strongest form
	// this control can take.
	g.result.DeviceEventRegistrationChecks++

	if err := device.SetMultiSampleMask(0x0f0f0f0f); err != nil {
		return fmt.Errorf("SetMultiSampleMask: %w", err)
	}
	mask, err := device.MultiSampleMask()
	if err != nil {
		return fmt.Errorf("MultiSampleMask: %w", err)
	}
	if mask != 0x0f0f0f0f {
		return fmt.Errorf("MultiSampleMask round trip = %#x, want 0x0f0f0f0f", mask)
	}
	g.result.DeviceStateRoundTrips++

	if err := device.SetReferenceStencil(37); err != nil {
		return fmt.Errorf("SetReferenceStencil: %w", err)
	}
	stencil, err := device.ReferenceStencil()
	if err != nil {
		return fmt.Errorf("ReferenceStencil: %w", err)
	}
	if stencil != 37 {
		return fmt.Errorf("ReferenceStencil round trip = %d, want 37", stencil)
	}
	g.result.DeviceStateRoundTrips++

	scissor := framework.NewRectangle(3, 5, 64, 48)
	if err := device.SetScissorRectangle(scissor); err != nil {
		return fmt.Errorf("SetScissorRectangle: %w", err)
	}
	readScissor, err := device.ScissorRectangle()
	if err != nil {
		return fmt.Errorf("ScissorRectangle: %w", err)
	}
	if readScissor != scissor {
		return fmt.Errorf("ScissorRectangle round trip = %+v, want %+v", readScissor, scissor)
	}
	g.result.DeviceStateRoundTrips++

	// The viewport's getter has been projected since Foundation 1 and its
	// setter arrives now, so this round trip crosses two milestones' work.
	viewport := graphics.NewViewportByInt32AndInt32AndInt32AndInt32(2, 4, 320, 200)
	viewport.SetMinDepth(0.25)
	viewport.SetMaxDepth(0.75)
	if err := device.SetViewport(viewport); err != nil {
		return fmt.Errorf("SetViewport: %w", err)
	}
	readViewport, err := device.Viewport()
	if err != nil {
		return fmt.Errorf("Viewport: %w", err)
	}
	if readViewport.X() != 2 || readViewport.Y() != 4 || readViewport.Width() != 320 || readViewport.Height() != 200 ||
		readViewport.MinDepth() != 0.25 || readViewport.MaxDepth() != 0.75 {
		return fmt.Errorf("Viewport round trip = %s", readViewport.ToString())
	}
	g.result.DeviceStateRoundTrips++

	// Three read-only members. Their values come from the device rather than
	// from a constant here, so what is asserted is that each is a value the
	// enum actually declares -- not a number CNA-Go chose.
	profile, err := device.GraphicsProfile()
	if err != nil {
		return fmt.Errorf("GraphicsProfile: %w", err)
	}
	if profile != graphics.GraphicsProfileReach && profile != graphics.GraphicsProfileHiDef {
		return fmt.Errorf("GraphicsProfile = %d, which is not a declared value", profile)
	}
	g.result.DeviceStateReadOnlyChecks++

	status, err := device.GraphicsDeviceStatus()
	if err != nil {
		return fmt.Errorf("GraphicsDeviceStatus: %w", err)
	}
	if status != graphics.GraphicsDeviceStatusNormal && status != graphics.GraphicsDeviceStatusLost &&
		status != graphics.GraphicsDeviceStatusNotReset {
		return fmt.Errorf("GraphicsDeviceStatus = %d, which is not a declared value", status)
	}
	g.result.DeviceStateReadOnlyChecks++

	disposed, err := device.IsDisposed()
	if err != nil {
		return fmt.Errorf("IsDisposed: %w", err)
	}
	if disposed {
		return errors.New("the live device reports itself disposed from inside a draw callback")
	}
	g.result.DeviceStateReadOnlyChecks++

	// Both masked Clear overloads, through the same route, with the same mask.
	options := graphics.ClearOptionsTarget | graphics.ClearOptionsDepthBuffer
	if err := device.ClearByClearOptionsAndColorAndSingleAndInt32(options, factor, 1, 0); err != nil {
		return fmt.Errorf("Clear(ClearOptions, Color, ...): %w", err)
	}
	g.result.DeviceStateClearCalls++
	if err := device.ClearByClearOptionsAndVector4AndSingleAndInt32(
		options, framework.NewVector4BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1), 1, 0); err != nil {
		return fmt.Errorf("Clear(ClearOptions, Vector4, ...): %w", err)
	}
	g.result.DeviceStateClearCalls++

	// CNA refuses a non-finite depth with CNA_RESULT_INVALID_ARGUMENT, and the
	// refusal must surface rather than be swallowed. A projection that ignored
	// the result would look identical from Go.
	nonFinite := float32(math.Inf(1))
	if refusal := device.ClearByClearOptionsAndColorAndSingleAndInt32(options, factor, nonFinite, 0); refusal == nil {
		return errors.New("a non-finite clear depth was accepted")
	}
	g.result.DeviceStateClearRefusals++

	if err := device.PresentByNone(); err != nil {
		return fmt.Errorf("Present: %w", err)
	}
	g.result.DeviceStatePresentCalls++

	// The device's display mode. Its two computed members are reproduced from
	// the dimensions rather than taken from CNA's own aspect ratio, so what is
	// checked here is that they agree with the dimensions CNA reported.
	mode, err := device.DisplayMode()
	if err != nil {
		return fmt.Errorf("DisplayMode: %w", err)
	}
	if mode.Width() <= 0 || mode.Height() <= 0 {
		return fmt.Errorf("DisplayMode = %s, which has a non-positive dimension", mode.ToString())
	}
	g.result.DeviceStateDisplayModeChecks++
	safe := mode.TitleSafeArea()
	wantAspect := float32(mode.Width()) / float32(mode.Height())
	if safe.X != 0 || safe.Y != 0 || safe.Width != mode.Width() || safe.Height != mode.Height() ||
		mode.AspectRatio() != wantAspect {
		return fmt.Errorf("DisplayMode computed members disagree with its dimensions: %s", mode.ToString())
	}
	g.result.DeviceStateDisplayModeChecks++

	// An EMPTY texture, created from a stated size and format rather than
	// decoded. Three per cycle: the three-argument constructor, the
	// five-argument one with the same defaults, and one with a mip chain.
	for index, create := range []func() (*graphics.Texture2D, error){
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 32, 16)
		},
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
				device, 32, 16, false, graphics.SurfaceFormatColor)
		},
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
				device, 64, 64, true, graphics.SurfaceFormatColor)
		},
	} {
		created, createErr := create()
		if createErr != nil {
			return fmt.Errorf("empty texture %d: %w", index, createErr)
		}
		width, height := created.Width(), created.Height()
		wantWidth, wantHeight := int32(32), int32(16)
		if index == 2 {
			wantWidth, wantHeight = 64, 64
		}
		if width != wantWidth || height != wantHeight {
			return fmt.Errorf("empty texture %d = %dx%d, want %dx%d", index, width, height, wantWidth, wantHeight)
		}
		if err := created.DisposeByBoolean(true); err != nil {
			return fmt.Errorf("empty texture %d disposal: %w", index, err)
		}
		g.result.DeviceStateTextureCreations++
	}

	// A texture encoded to PNG and to JPEG, then decoded back at a REQUESTED
	// size through both of the reference's zoom modes.
	//
	// The encoded bytes are checked for their format signature rather than for
	// a length, because a length proves only that something was written: PNG
	// begins with the eight-byte magic and JPEG with SOI. A projection that sent
	// XNA's own format identity through -- SaveAsPng passes 2, and CNA's PNG is
	// 0 while its JPEG is 1 -- would encode a JPEG under the PNG member and this
	// is what catches it.
	source, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
	if err != nil {
		return fmt.Errorf("source texture: %w", err)
	}
	var png, jpeg bytes.Buffer
	if err := source.SaveAsPng(&png, 32, 32); err != nil {
		return fmt.Errorf("SaveAsPng: %w", err)
	}
	if got := png.Bytes(); len(got) < 8 || string(got[1:4]) != "PNG" {
		return fmt.Errorf("SaveAsPng wrote %d bytes that do not begin with the PNG signature", len(got))
	}
	g.result.DeviceStateEncodeChecks++
	if err := source.SaveAsJpeg(&jpeg, 32, 32); err != nil {
		return fmt.Errorf("SaveAsJpeg: %w", err)
	}
	if got := jpeg.Bytes(); len(got) < 3 || got[0] != 0xff || got[1] != 0xd8 {
		return fmt.Errorf("SaveAsJpeg wrote %d bytes that do not begin with the JPEG SOI marker", len(got))
	}
	g.result.DeviceStateEncodeChecks++

	for _, zoom := range []bool{false, true} {
		decoded, decodeErr := graphics.Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(
			device, bytes.NewReader(png.Bytes()), 24, 24, zoom)
		if decodeErr != nil {
			return fmt.Errorf("sized decode (zoom=%t): %w", zoom, decodeErr)
		}
		width, height := decoded.Width(), decoded.Height()
		if width != 24 || height != 24 {
			return fmt.Errorf("sized decode (zoom=%t) = %dx%d, want 24x24", zoom, width, height)
		}
		if err := decoded.DisposeByBoolean(true); err != nil {
			return fmt.Errorf("sized decode disposal: %w", err)
		}
		g.result.DeviceStateDecodeSizeChecks++
	}

	// The one guard SaveAsImage's prologue has that Go can express: a nil
	// destination carries Microsoft's own sentence.
	if refusal := source.SaveAsPng(nil, 8, 8); refusal == nil ||
		!strings.Contains(refusal.Error(), "This method does not accept null for this parameter.") {
		return fmt.Errorf("SaveAsPng to a nil writer = %v, want the reference's message", refusal)
	}
	g.result.DeviceStateEncodeRefusals++
	if err := source.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("source texture disposal: %w", err)
	}

	// Typed transfers, through the generic-method projection. Each writes a
	// pattern to the live texture and reads it back FROM the texture, so a
	// projection that kept a managed copy would pass a test that compared its
	// own input.
	transferTexture, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 4, 4)
	if err != nil {
		return fmt.Errorf("transfer texture: %w", err)
	}
	written := make([]framework.Color, 16)
	for index := range written {
		written[index] = framework.NewColorByInt32AndInt32AndInt32AndInt32(
			int32(index*16%256), int32(index*8%256), int32(index*4%256), 255)
	}
	if err := graphics.Texture2DSetDataBySliceOfT(transferTexture, written); err != nil {
		return fmt.Errorf("SetData: %w", err)
	}
	readBack := make([]framework.Color, 16)
	if err := graphics.Texture2DGetDataBySliceOfT(transferTexture, readBack); err != nil {
		return fmt.Errorf("GetData: %w", err)
	}
	for index := range written {
		if readBack[index] != written[index] {
			return fmt.Errorf("texel %d round-tripped as %v, want %v", index, readBack[index], written[index])
		}
	}
	g.result.DeviceStateTransferRoundTrips++

	// A WINDOW of the same array, through the three-argument overload.
	windowed := make([]framework.Color, 16)
	if err := graphics.Texture2DGetDataBySliceOfTAndInt32AndInt32(transferTexture, windowed, 0, 16); err != nil {
		return fmt.Errorf("windowed GetData: %w", err)
	}
	if windowed[0] != written[0] || windowed[15] != written[15] {
		return errors.New("a windowed transfer did not reproduce the full surface")
	}
	g.result.DeviceStateTransferRoundTrips++

	// A RECTANGLE, through the five-argument overload the other two funnel
	// into. Two by two at the origin is four texels of the sixteen.
	region := framework.NewRectangle(0, 0, 2, 2)
	corner := make([]framework.Color, 4)
	if err := graphics.Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		transferTexture, 0, &region, corner, 0, 4); err != nil {
		return fmt.Errorf("rectangle GetData: %w", err)
	}
	if corner[0] != written[0] {
		return fmt.Errorf("the rectangle transfer's first texel is %v, want %v", corner[0], written[0])
	}
	g.result.DeviceStateTransferRoundTrips++

	// The two refusals the projection makes before CNA is reached: an element
	// type outside the eighteen CNA declares, and a transfer window that leaves
	// the array.
	if refusal := graphics.Texture2DSetDataBySliceOfT(transferTexture, []int64{1, 2, 3, 4}); refusal == nil ||
		!strings.Contains(refusal.Error(), "is not one of the eighteen element types") {
		return fmt.Errorf("an unsupported element type produced %v", refusal)
	}
	g.result.DeviceStateTransferRefusals++
	if refusal := graphics.Texture2DSetDataBySliceOfTAndInt32AndInt32(transferTexture, written, 8, 16); refusal == nil {
		return errors.New("a transfer window past the end of the array was accepted")
	}
	g.result.DeviceStateTransferRefusals++
	if err := transferTexture.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("transfer texture disposal: %w", err)
	}

	// The two refusals the projection makes itself, before CNA is reached: a
	// nil device carries Microsoft's own sentence, and a negative dimension is
	// refused rather than converted into an enormous uint32.
	if _, refusal := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(nil, 4, 4); refusal == nil ||
		!strings.Contains(refusal.Error(), "The GraphicsDevice must not be null when creating new resources.") {
		return fmt.Errorf("nil-device texture creation = %v, want the reference's message", refusal)
	}
	g.result.DeviceStateTextureRefusals++
	if _, refusal := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, -1, 4); refusal == nil {
		return errors.New("a negative texture width was accepted")
	}
	g.result.DeviceStateTextureRefusals++

	// The owner-thread policy reaches these members too. A second goroutine
	// must be refused before it can touch the device.
	wrongThread := make(chan error, 1)
	go func() { _, callErr := wrongThread2Read(device); wrongThread <- callErr }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("device state from a second goroutine = %v, want ErrWrongThread", err)
	}
	g.result.DeviceStateWrongThreadHits++
	return nil
}

// wrongThread2Read is the one device read the wrong-thread proof makes, kept
// separate so the goroutine body stays a single call.
func wrongThread2Read(device *graphics.GraphicsDevice) (framework.Color, error) {
	return device.BlendFactor()
}

// drawEverySpriteOverload submits one command through each of the profile's
// seven Draw overloads, inside one begin/end pair, in a live draw callback.
//
// Four of the seven reach cna_sprite_batch_submit_scaled_many and three reach
// cna_sprite_batch_submit_many, and the two counters below are what separate
// them: a projection that routed a rectangle overload through the scaled family
// would draw something, and would draw it in the wrong place, which no return
// value reports.
func (g *stressGame) drawEverySpriteOverload() error {
	batch := g.spriteBatch
	texture := g.spriteTexture
	white := framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)
	source := framework.NewRectangle(0, 0, 16, 16)
	destination := framework.NewRectangle(4, 8, 32, 24)
	origin := framework.NewVector2BySingleAndSingle(2, 3)

	// A second Begin inside a pair is the reference's EndMustBeCalledBeforeBegin
	// throw, and it is checked while a pair is genuinely open rather than
	// simulated.
	if err := batch.BeginByNone(); err != nil {
		return err
	}
	if again := batch.BeginByNone(); again == nil ||
		!strings.Contains(again.Error(), "Begin cannot be called again until End has been successfully called.") {
		return fmt.Errorf("second Begin = %v, want the InvalidOperationException message", again)
	}
	g.result.SpriteDrawPairGuardChecks++

	scaled := []func() error{
		func() error {
			return batch.DrawByTexture2DAndVector2AndColor(texture, framework.NewVector2BySingleAndSingle(1, 2), white)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(texture, framework.NewVector2BySingleAndSingle(3, 4), &source, white)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				texture, framework.NewVector2BySingleAndSingle(5, 6), &source, white, 0.5, origin, 2, graphics.SpriteEffectsFlipHorizontally, 0.25)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				texture, framework.NewVector2BySingleAndSingle(7, 8), &source, white, 0.5, origin, framework.NewVector2BySingleAndSingle(2, 3), graphics.SpriteEffectsFlipVertically, 0.75)
		},
	}
	for index, draw := range scaled {
		if err := draw(); err != nil {
			return fmt.Errorf("scaled Draw overload %d: %w", index, err)
		}
		g.result.SpriteDrawScaledSubmits++
	}

	destinations := []func() error{
		func() error { return batch.DrawByTexture2DAndRectangleAndColor(texture, destination, white) },
		func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(texture, destination, &source, white)
		},
		func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle(
				texture, destination, &source, white, 0.5, origin, graphics.SpriteEffectsFlipHorizontally, 0.5)
		},
	}
	for index, draw := range destinations {
		if err := draw(); err != nil {
			return fmt.Errorf("destination Draw overload %d: %w", index, err)
		}
		g.result.SpriteDrawDestinationSubmits++
	}

	// A nil source rectangle is the static nullRectangle the three-argument
	// overloads pass, and it must reach the same route rather than being
	// refused. Both families are exercised with one.
	if err := batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(texture, framework.NewVector2BySingleAndSingle(9, 10), nil, white); err != nil {
		return fmt.Errorf("scaled Draw with a nil source: %w", err)
	}
	g.result.SpriteDrawScaledSubmits++
	if err := batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(texture, destination, nil, white); err != nil {
		return fmt.Errorf("destination Draw with a nil source: %w", err)
	}
	g.result.SpriteDrawDestinationSubmits++

	if err := batch.End(); err != nil {
		return err
	}
	// The pair is closed, so the guard is armed again -- which is what proves
	// End cleared the flag rather than only flushing.
	if after := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, white); after == nil ||
		!strings.Contains(after.Error(), "Begin must be called successfully before a Draw can be called.") {
		return fmt.Errorf("Draw after End = %v, want the InvalidOperationException message", after)
	}
	g.result.SpriteDrawOutsidePairChecks++
	return nil
}

func (g *stressGame) UnloadContent(_ *framework.Game) error {
	if g.manager != nil {
		if err := g.manager.Dispose(true); err != nil {
			return err
		}
		postDispose := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationLandscapeRight
		g.manager.SetSupportedOrientations(postDispose)
		if got := g.manager.SupportedOrientations(); got != postDispose {
			return fmt.Errorf("post-disposal SupportedOrientations = %d, want %d", got, postDispose)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// The optional per-hook frame-boundary overrides, against the pinned runtime.
// ---------------------------------------------------------------------------

// frameHookGame is a consumer whose callback object declares optional frame
// overrides. Nothing here names an unexported binding type: the opt-in is the
// exported method, and which methods this type declares is decided by the
// `subset` flag at construction, exactly as a consumer decides it by writing
// them or not.
//
// The claim it proves is the one no in-process test can: that the CNA runtime
// really delivers the installed hooks at the measured frame positions, really
// skips draw AND end_draw on a frame the override refused, and really never
// delivers a hook whose capability the object does not declare.
type frameHookGame struct {
	// declaresEndDraw records whether THIS object declares the EndDraw
	// override, which is what decides whether an admitted frame should deliver
	// the end_draw hook at all.
	declaresEndDraw bool
	order           []string
	updates         int
	baseAnswers     []bool
	baseCalls       int
	result          counters
	ownerGoroutine  string
	goroutines      map[string]bool
	uninstalled     []string
}

func (g *frameHookGame) record(entry string) {
	g.order = append(g.order, entry)
	g.goroutines[currentGoroutineLabel()] = true
}

func (g *frameHookGame) Initialize(*framework.Game) error {
	g.ownerGoroutine = currentGoroutineLabel()
	g.goroutines = map[string]bool{}
	return nil
}

func (g *frameHookGame) LoadContent(*framework.Game) error { return nil }

// Update ends the run after enough frames that both the refused and the
// admitted branch are exercised several times.
func (g *frameHookGame) Update(host *framework.Game, _ framework.GameTime) error {
	g.updates++
	g.record("update")
	if g.updates >= 8 {
		return host.Exit()
	}
	return nil
}

func (g *frameHookGame) Draw(*framework.Game, framework.GameTime) error {
	g.record("draw")
	g.result.FrameHookAdmittedFrames++
	if g.declaresEndDraw {
		g.result.FrameHookEndDrawExpected++
	}
	return nil
}

func (g *frameHookGame) UnloadContent(*framework.Game) error { return nil }

// BeginDraw is the one optional override both scenarios declare. It calls the
// base explicitly -- the Go projection of base.BeginDraw() -- records what the
// base answered, and then refuses every other frame with its OWN answer.
//
// The base always admits the frame here, because no IGraphicsDeviceManager is
// registered. So a skipped draw is positive proof that the override's answer
// and not the base's is what reached CNA.
func (g *frameHookGame) BeginDraw(host *framework.Game) (bool, error) {
	g.record("begin_draw")
	g.result.FrameHookBeginDrawHits++
	answer, err := host.BeginDraw()
	if err != nil {
		return false, err
	}
	g.baseCalls++
	g.baseAnswers = append(g.baseAnswers, answer)
	if g.result.FrameHookBeginDrawHits%2 == 0 {
		g.result.FrameHookRefusedFrames++
		g.record("refused")
		return false, nil
	}
	return true, nil
}

// frameHookAllGame adds the other three overrides. The subset scenario uses
// frameHookGame directly and therefore declares only BeginDraw, which is what
// makes "an omitted capability never receives its hook" measurable.
type frameHookAllGame struct{ frameHookGame }

func (g *frameHookAllGame) BeginRun(host *framework.Game) error {
	g.record("begin_run")
	g.result.FrameHookBeginRunHits++
	return host.BeginRun()
}

func (g *frameHookAllGame) EndRun(host *framework.Game) error {
	g.record("end_run")
	g.result.FrameHookEndRunHits++
	return host.EndRun()
}

func (g *frameHookAllGame) EndDraw(host *framework.Game) error {
	g.record("end_draw")
	g.result.FrameHookEndDrawHits++
	return host.EndDraw()
}

// frameHookSubsetGame declares only BeginDraw, and additionally records any of
// the other three arriving. They cannot: their CNA_GameFrameHooks members are
// left NULL because no capability was declared for them, and a null member is
// one the canonical header says is simply not called. The recording exists so
// that if they ever did arrive, the run would fail loudly rather than quietly
// gaining behaviour.
type frameHookSubsetGame struct{ frameHookGame }

func runFrameHookChild(scenario string) error {
	var callbacks framework.GameCallbacks
	var state *frameHookGame
	switch scenario {
	case "frame-hook-override":
		every := &frameHookAllGame{frameHookGame{declaresEndDraw: true}}
		callbacks, state = every, &every.frameHookGame
	case "frame-hook-subset":
		subset := &frameHookSubsetGame{}
		callbacks, state = subset, &subset.frameHookGame
	default:
		return fmt.Errorf("unknown frame-hook scenario %q", scenario)
	}
	host, err := framework.NewGame(callbacks)
	if err != nil {
		return err
	}
	if err := host.Run(); err != nil {
		return err
	}
	if err := state.verify(scenario); err != nil {
		return err
	}
	runtime.GC()
	state.result.GCStressPoints++
	data, _ := json.Marshal(state.result)
	fmt.Println(string(data))
	return nil
}

// verify holds every claim the scenario exists to prove.
func (g *frameHookGame) verify(scenario string) error {
	if len(g.uninstalled) != 0 {
		g.result.FrameHookUninstalledHits += len(g.uninstalled)
		return fmt.Errorf("hooks %v arrived for capabilities that were never declared", g.uninstalled)
	}
	// Every override call reached the base exactly once, and the base admitted
	// every frame -- which is what makes a skipped draw the override's doing.
	if g.baseCalls != g.result.FrameHookBeginDrawHits {
		return fmt.Errorf("the override ran %d times and called the base %d times", g.result.FrameHookBeginDrawHits, g.baseCalls)
	}
	for i, answer := range g.baseAnswers {
		if !answer {
			return fmt.Errorf("base BeginDraw refused frame %d with no manager registered", i)
		}
	}
	if g.result.FrameHookBeginDrawHits == 0 {
		return errors.New("begin_draw was never delivered, so the hook was not installed")
	}
	g.result.FrameHookBaseCallChecks++

	// A refused frame delivers neither draw nor end_draw; an admitted one
	// delivers draw and then end_draw, in that order.
	refused, admitted := 0, 0
	for i := 0; i < len(g.order); i++ {
		if g.order[i] != "begin_draw" {
			continue
		}
		next := g.order[i+1:]
		if len(next) > 0 && next[0] == "refused" {
			refused++
			for _, entry := range next[1:] {
				if entry == "begin_draw" {
					break
				}
				if entry == "draw" || entry == "end_draw" {
					return fmt.Errorf("a refused frame still delivered %q; order=%v", entry, g.order)
				}
			}
			continue
		}
		admitted++
		if len(next) == 0 || next[0] != "draw" {
			return fmt.Errorf("an admitted frame did not deliver draw next; order=%v", g.order)
		}
		if scenario == "frame-hook-override" && (len(next) < 2 || next[1] != "end_draw") {
			return fmt.Errorf("an admitted frame did not deliver end_draw after draw; order=%v", g.order)
		}
	}
	if refused == 0 || admitted == 0 {
		return fmt.Errorf("scenario exercised %d refused and %d admitted frames; both branches are required", refused, admitted)
	}
	g.result.FrameHookSkipChecks++

	if len(g.goroutines) != 1 || !g.goroutines[g.ownerGoroutine] {
		return fmt.Errorf("frame hooks were delivered on %d goroutines, want the owner goroutine only", len(g.goroutines))
	}

	switch scenario {
	case "frame-hook-override":
		// begin_run before every frame, end_run after every frame, once each.
		if g.result.FrameHookBeginRunHits != 1 || g.result.FrameHookEndRunHits != 1 {
			return fmt.Errorf("begin_run fired %d times and end_run %d, want exactly once each",
				g.result.FrameHookBeginRunHits, g.result.FrameHookEndRunHits)
		}
		if g.order[0] != "begin_run" {
			return fmt.Errorf("delivery order %v does not start with begin_run", g.order)
		}
		if g.order[len(g.order)-1] != "end_run" {
			return fmt.Errorf("delivery order %v does not end with end_run", g.order)
		}
		g.result.FrameHookOrderChecks++
		g.result.FrameHookOverrideCycles++
	case "frame-hook-subset":
		// The three undeclared capabilities installed nothing, so none of
		// their hooks can appear anywhere in the order.
		for _, entry := range g.order {
			switch entry {
			case "begin_run", "end_run", "end_draw":
				g.result.FrameHookUninstalledHits++
				return fmt.Errorf("hook %q was delivered to a callback object that declares only BeginDraw", entry)
			}
		}
		g.result.FrameHookOrderChecks++
		g.result.FrameHookSubsetCycles++
	}
	return nil
}

func encodedPNG() []byte {
	value := image.NewRGBA(image.Rect(0, 0, 2, 2))
	value.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	value.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	value.SetRGBA(0, 1, color.RGBA{B: 255, A: 255})
	value.SetRGBA(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var output bytes.Buffer
	if err := png.Encode(&output, value); err != nil {
		panic(err)
	}
	return output.Bytes()
}

// addCounters sums every counter mechanically.
//
// It used to be a hand-written list of one line per field, and that list was a
// silent single point of failure: a counter added to the struct but not to the
// list stayed at zero in the aggregate report while its scenario ran perfectly,
// so the evidence would read "this was never measured" for something that was.
// Foundation 45 hit exactly that. Reflection cannot drift, and the panic below
// is a programmer invariant -- every counter is an int by construction.
func addCounters(target *counters, value counters) {
	destination := reflect.ValueOf(target).Elem()
	source := reflect.ValueOf(value)
	for index := 0; index < destination.NumField(); index++ {
		field := destination.Field(index)
		if field.Kind() != reflect.Int {
			panic(fmt.Sprintf("counters.%s is %s; every counter must be an int", destination.Type().Field(index).Name, field.Kind()))
		}
		field.SetInt(field.Int() + source.Field(index).Int())
	}
}

// ---------------------------------------------------------------------------
// Game's timing and presentation state, against the pinned runtime.
// ---------------------------------------------------------------------------

// timingGame configures a Game before Run and then, from inside a lifecycle
// callback on the owner thread, drives every timing setter against the live
// native loop.
//
// The claim it proves is the one no in-process test can: these are not stored
// values that look like they work. They reach CNA, CNA accepts them, and the
// values a Game was configured with BEFORE Run are what cna_game_create was
// handed -- a rejected target step would have failed creation outright.
type timingGame struct {
	result counters
}

func (g *timingGame) Initialize(host *framework.Game) error {
	// Inside a lifecycle callback on the owner thread: every setter must reach
	// the live native game.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(100000)); err != nil {
		return fmt.Errorf("SetTargetElapsedTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(0)); err != nil {
		return fmt.Errorf("SetInactiveSleepTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetIsFixedTimeStep(false); err != nil {
		return fmt.Errorf("SetIsFixedTimeStep during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetIsMouseVisible(true); err != nil {
		return fmt.Errorf("SetIsMouseVisible during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SuppressDraw(); err != nil {
		return fmt.Errorf("SuppressDraw during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.ResetElapsedTime(); err != nil {
		return fmt.Errorf("ResetElapsedTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++

	// The managed field is what the getter reads, and it holds what was stored
	// regardless of where the value went afterwards.
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("TargetElapsedTime = %d after setting it to 100000", got)
	}
	if host.IsFixedTimeStep() || !host.IsMouseVisible() {
		return fmt.Errorf("flags = %t/%t after setting them false/true", host.IsFixedTimeStep(), host.IsMouseVisible())
	}

	// From another goroutine CNA answers CNA_RESULT_THREAD, and the projection
	// reports it rather than pretending the loop was told.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- host.SetIsFixedTimeStep(true) }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a timing setter from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.TimingWrongThreadChecks++

	// The argument checks are the reference's own and come before anything
	// native: TargetElapsedTime rejects zero, InactiveSleepTime accepts it.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(0)); err == nil {
		return errors.New("TargetElapsedTime accepted zero; the reference compares with op_LessThanOrEqual")
	}
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(0)); err != nil {
		return fmt.Errorf("InactiveSleepTime rejected zero: %w", err)
	}
	g.result.TimingSettersApplied-- // the second accepted set is not a seventh setting
	g.result.TimingRangeChecks++
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("a rejected TargetElapsedTime still stored: %d", got)
	}
	// Restore the accounting: exactly six settings reached the loop.
	g.result.TimingSettersApplied++
	return nil
}

func (g *timingGame) LoadContent(*framework.Game) error                       { return nil }
func (g *timingGame) Update(host *framework.Game, _ framework.GameTime) error { return host.Exit() }
func (g *timingGame) Draw(*framework.Game, framework.GameTime) error          { return nil }
func (g *timingGame) UnloadContent(*framework.Game) error                     { return nil }

// runTimingChild configures the Game BEFORE Run, which is the state a real
// consumer sets a frame rate in, and proves the configured values are what the
// native game is created with.
func runTimingChild() error {
	game := &timingGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	// A non-default, valid step. If cna_game_create did not accept it -- or if
	// the create path still passed a literal -- this run would not start.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(83333)); err != nil {
		return fmt.Errorf("configure before Run: %w", err)
	}
	if err := host.SetIsMouseVisible(true); err != nil {
		return fmt.Errorf("configure mouse visibility before Run: %w", err)
	}
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(1000)); err != nil {
		return fmt.Errorf("configure inactive sleep before Run: %w", err)
	}
	if err := host.Run(); err != nil {
		return err
	}
	game.result.TimingCreatedWithConfig++
	game.result.TimingCycles++
	// After the run the managed state is still readable, and still says what
	// the last successful set stored.
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("TargetElapsedTime after the run = %d, want the value Initialize stored", got)
	}
	// With no live native game the setters store and report success again.
	if err := host.SetIsFixedTimeStep(true); err != nil {
		return fmt.Errorf("SetIsFixedTimeStep after the run: %w", err)
	}
	if !host.IsFixedTimeStep() {
		return errors.New("a post-run set did not reach the managed field")
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// windowGame proves GameWindow against a LIVE native game, which is the half
// the pure-Go tests structurally cannot reach: with no native game every
// guarded member answers its fallback, so only a run can show that the same
// member reads a real window when there is one.
type windowGame struct {
	result   counters
	window   *framework.GameWindow
	deviceOK bool
}

func (g *windowGame) Initialize(host *framework.Game) error {
	window := host.Window()
	// The identity a consumer captured BEFORE Run is the identity the callback
	// sees. A projection that allocated per call would hand out a second
	// object here and silently orphan every subscription made before Run.
	if window != g.window {
		return errors.New("Game.Window returned a different object inside a callback than before Run")
	}
	g.result.WindowIdentityChecks++

	// The UNGUARDED member now succeeds. Before Run it reported the
	// reference's NullReferenceException; the difference between the two is
	// the whole point of the measured guard split.
	bounds, err := window.ClientBounds()
	if err != nil {
		return fmt.Errorf("ClientBounds with a live window: %w", err)
	}
	g.result.WindowLiveReads++
	// The measured SIZE is the renderer's, not the binding's. The HEADLESS
	// artifact reports a 0x0 client rectangle while its graphics device
	// reports an 800x480 viewport, so a positive size is COUNTED rather than
	// required: requiring it would make this scenario a renderer test that
	// fails for a reason CNA-Go cannot fix, and asserting 0x0 would bake a
	// headless artifact's answer into the contract.
	if bounds.Width > 0 && bounds.Height > 0 {
		g.result.WindowPositiveClientBounds++
	}

	// The guarded members answer without failing. Their VALUES are the
	// platform's and are not asserted -- a headless window legitimately has no
	// screen device name and a zero handle -- but a failure would be a real
	// defect and is.
	if _, err := window.Handle(); err != nil {
		return fmt.Errorf("Handle with a live window: %w", err)
	}
	if _, err := window.ScreenDeviceName(); err != nil {
		return fmt.Errorf("ScreenDeviceName with a live window: %w", err)
	}
	g.result.WindowLiveReads += 2

	// AllowUserResizing round-trips only if the platform honours it. Both
	// calls must report success; whether the value came back is COUNTED rather
	// than required, because a headless window has no resize grip to grant.
	before, err := window.AllowUserResizing()
	if err != nil {
		return fmt.Errorf("AllowUserResizing with a live window: %w", err)
	}
	if err := window.SetAllowUserResizing(!before); err != nil {
		return fmt.Errorf("SetAllowUserResizing with a live window: %w", err)
	}
	after, err := window.AllowUserResizing()
	if err != nil {
		return fmt.Errorf("AllowUserResizing after assignment: %w", err)
	}
	if after == !before {
		g.result.WindowResizeRoundTrips++
	}
	g.result.WindowLiveReads++

	// Title: the managed field is authoritative, and the assignment reaches
	// the live loop.
	if got := window.Title(); got != "" {
		return fmt.Errorf("Title = %q at the start of a run, want the constructor's String.Empty", got)
	}
	if err := window.SetTitleProperty("cna-go window stress"); err != nil {
		return fmt.Errorf("SetTitleProperty with a live window: %w", err)
	}
	if got := window.Title(); got != "cna-go window stress" {
		return fmt.Errorf("Title = %q after assignment", got)
	}
	g.result.WindowLiveReads++

	// The suppression proof, and it needs a live run to exist at all.
	//
	// set_Title's guard is `if (this.title != value)`, so an UNCHANGED
	// assignment performs no platform call. From a non-owner goroutine that is
	// observable and nothing else is: an unchanged assignment succeeds because
	// it never reaches the boundary, while a changed one is refused for being
	// off the owner thread. A projection that dropped the guard would fail the
	// first of the two.
	unchanged := make(chan error, 1)
	go func() { unchanged <- window.SetTitleProperty("cna-go window stress") }()
	if err := <-unchanged; err != nil {
		return fmt.Errorf("an unchanged title from a non-owner goroutine reported %v; the guard should have stopped it before the boundary", err)
	}
	g.result.WindowTitleSuppressions++
	changed := make(chan error, 1)
	go func() { changed <- window.SetTitleProperty("a different title") }()
	if err := <-changed; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a changed title from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.WindowWrongThreadChecks++
	// The refused assignment still stored the managed field before it reached
	// the boundary, exactly as the reference's own order does: the store
	// precedes the SetTitle call.
	if got := window.Title(); got != "a different title" {
		return fmt.Errorf("Title = %q after a refused native call; the reference stores before it calls", got)
	}

	// The screen-device-change pair, which is the other unguarded family.
	name, err := window.ScreenDeviceName()
	if err != nil {
		return fmt.Errorf("ScreenDeviceName before a device change: %w", err)
	}
	if err := window.BeginScreenDeviceChange(false); err != nil {
		return fmt.Errorf("BeginScreenDeviceChange with a live window: %w", err)
	}
	if err := window.EndScreenDeviceChangeByString(name); err != nil {
		return fmt.Errorf("EndScreenDeviceChange with a live window: %w", err)
	}
	g.result.WindowScreenDeviceChanges++

	// CurrentOrientation is the reference's constant and stays it even with a
	// live window: the reference never asks the platform in this profile.
	if got := window.CurrentOrientation(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("CurrentOrientation = %v with a live window, want the reference's constant Default", got)
	}
	window.SetSupportedOrientations(framework.DisplayOrientationPortrait)
	if got := window.CurrentOrientation(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("CurrentOrientation = %v after SetSupportedOrientations; the reference's body is one `ret`", got)
	}
	g.result.WindowLiveReads++

	if runtime, ok := interop.CurrentRuntime(); ok {
		deliveries := runtime.GameWindowEventDeliveries()
		g.result.WindowEventClientSize += deliveries[interop.GameWindowEventClientSizeChanged]
		g.result.WindowEventOrientation += deliveries[interop.GameWindowEventOrientationChanged]
		g.result.WindowEventScreenDevice += deliveries[interop.GameWindowEventScreenDeviceNameChanged]
		g.deviceOK = true
	}
	return nil
}

func (g *windowGame) LoadContent(*framework.Game) error                       { return nil }
func (g *windowGame) Update(host *framework.Game, _ framework.GameTime) error { return host.Exit() }
func (g *windowGame) Draw(*framework.Game, framework.GameTime) error          { return nil }
func (g *windowGame) UnloadContent(*framework.Game) error                     { return nil }

// runWindowChild measures the window before, during and after one run. The
// before and after halves are what prove the guard split is a split: the same
// member answers a fallback with no native window and a real value with one.
func runWindowChild() error {
	game := &windowGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	window := host.Window()
	if window == nil {
		return errors.New("Game.Window is nil after construction; the reference's EnsureHost runs in the constructor")
	}
	if host.Window() != window {
		return errors.New("Game.Window is not a stable identity before Run")
	}
	game.window = window
	game.result.WindowIdentityChecks++

	// Before Run: the five guarded members answer the reference's own
	// fallbacks and report nothing.
	handle, err := window.Handle()
	if err != nil || handle != 0 {
		return fmt.Errorf("Handle before Run = %#x, %v; want IntPtr.Zero and no failure", handle, err)
	}
	allow, err := window.AllowUserResizing()
	if err != nil || allow {
		return fmt.Errorf("AllowUserResizing before Run = %t, %v; want false and no failure", allow, err)
	}
	if err := window.SetAllowUserResizing(true); err != nil {
		return fmt.Errorf("SetAllowUserResizing before Run: %w", err)
	}
	name, err := window.ScreenDeviceName()
	if err != nil || name != "" {
		return fmt.Errorf("ScreenDeviceName before Run = %q, %v; want String.Empty and no failure", name, err)
	}
	if err := window.SetTitleMethod("before"); err != nil {
		return fmt.Errorf("SetTitle before Run: %w", err)
	}
	game.result.WindowGuardedFallbacks += 5

	// And the three unguarded ones report the reference's failure.
	if _, err := window.ClientBounds(); err == nil {
		return errors.New("ClientBounds succeeded before Run; the reference dereferences a null form")
	}
	if err := window.BeginScreenDeviceChange(true); err == nil {
		return errors.New("BeginScreenDeviceChange succeeded before Run")
	}
	if err := window.EndScreenDeviceChangeByStringAndInt32AndInt32("screen", 320, 240); err == nil {
		return errors.New("EndScreenDeviceChange succeeded before Run")
	}
	game.result.WindowUnguardedFailures += 3

	if err := host.Run(); err != nil {
		return err
	}
	if !game.deviceOK {
		return errors.New("the window scenario never reached a live runtime")
	}
	game.result.WindowCycles++

	// After the run the window is the same object, the managed title survives,
	// and every member is back to its no-native-window behaviour.
	if host.Window() != window {
		return errors.New("Game.Window changed identity after the run")
	}
	game.result.WindowIdentityChecks++
	if got := window.Title(); got != "a different title" {
		return fmt.Errorf("Title after the run = %q; the managed field outlives the native window", got)
	}
	if _, err := window.ClientBounds(); err == nil {
		return errors.New("ClientBounds succeeded after the run; the native window is gone")
	}
	game.result.WindowUnguardedFailures++
	handle, err = window.Handle()
	if err != nil || handle != 0 {
		return fmt.Errorf("Handle after the run = %#x, %v; want IntPtr.Zero and no failure", handle, err)
	}
	game.result.WindowGuardedFallbacks++

	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// frameStepGame drives a native game a frame at a time, with no loop anywhere.
//
// It declares BeginDraw and EndDraw so the optional frame hooks are installed
// too: a frame step has to reach the same hook positions a looped frame does,
// and a projection that only worked inside cna_game_run would not.
type frameStepGame struct {
	result counters

	initializes, loads, updates, draws, unloads int
	beginDraws, endDraws                        int

	// suppressNextDraw asks BeginDraw to refuse exactly one frame, which is
	// how SuppressDraw's effect is observed without a loop.
	host             *framework.Game
	tickFromCallback error
	callbackAsked    bool
}

func (g *frameStepGame) Initialize(host *framework.Game) error {
	g.initializes++
	// A frame step from INSIDE a lifecycle callback must be refused: CNA
	// answers CNA_RESULT_INVALID_STATE because a frame step called from within
	// a frame would re-enter the loop it is part of.
	if !g.callbackAsked {
		g.callbackAsked = true
		g.tickFromCallback = host.Tick()
	}
	return nil
}

func (g *frameStepGame) LoadContent(*framework.Game) error { g.loads++; return nil }

func (g *frameStepGame) Update(*framework.Game, framework.GameTime) error {
	g.updates++
	return nil
}

func (g *frameStepGame) Draw(*framework.Game, framework.GameTime) error {
	g.draws++
	return nil
}

func (g *frameStepGame) UnloadContent(*framework.Game) error {
	g.unloads++
	return nil
}

func (g *frameStepGame) BeginDraw(host *framework.Game) (bool, error) {
	g.beginDraws++
	return host.BeginDraw()
}

func (g *frameStepGame) EndDraw(host *framework.Game) error {
	g.endDraws++
	return host.EndDraw()
}

// runFrameStepChild is the whole lifecycle in one isolated process: create by
// stepping, step, dispose, and step again.
func runFrameStepChild() error {
	game := &frameStepGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	game.host = host

	runtimeOf := func() (*interop.Runtime, bool) { return interop.CurrentRuntime() }

	// Before the first step there is no native game at all.
	if _, live := runtimeOf(); live {
		return errors.New("a native runtime was current before the first frame step")
	}
	game.result.FrameStepSessionChecks++

	// TICK ONE. It creates the session and delivers exactly one Update and one
	// Draw -- and NO Initialize and no LoadContent, because Game::Tick has no
	// initialization step and CNA's does not either.
	if err := host.Tick(); err != nil {
		return fmt.Errorf("first Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.initializes != 0 || game.loads != 0 {
		return fmt.Errorf("a first Tick delivered %d initializes and %d loads; Tick does not initialize",
			game.initializes, game.loads)
	}
	if game.updates != 1 {
		return fmt.Errorf("a first Tick delivered %d updates, want one", game.updates)
	}
	game.result.FrameStepTickInitChecks++
	current, live := runtimeOf()
	if !live || !current.HasStandaloneSession() {
		return errors.New("the first Tick did not create a standalone session")
	}
	game.result.FrameStepSessionChecks++

	if err := host.Tick(); err != nil {
		return fmt.Errorf("second Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.initializes != 0 {
		return fmt.Errorf("a second Tick initialized %d times", game.initializes)
	}

	// RUN ONE FRAME. CNA initializes on first use, which the reference does
	// NOT do -- recorded as a measured upstream difference rather than hidden.
	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("first RunOneFrame: %w", err)
	}
	game.result.FrameStepRunOneFrames++
	if game.initializes != 1 || game.loads != 1 {
		return fmt.Errorf("a first RunOneFrame delivered %d initializes and %d loads, want one each",
			game.initializes, game.loads)
	}
	// The in-callback refusal was taken during that Initialize.
	if game.tickFromCallback == nil {
		return errors.New("a Tick from inside a lifecycle callback succeeded; CNA refuses a re-entrant frame step")
	}
	game.result.FrameStepCallbackRefusals++

	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("second RunOneFrame: %w", err)
	}
	game.result.FrameStepRunOneFrames++
	if game.initializes != 1 {
		return fmt.Errorf("initialization ran %d times across a session, want exactly one", game.initializes)
	}
	game.result.FrameStepInitializations = game.initializes

	// A step from another goroutine is refused: the goroutine that took the
	// first step owns the session's OS thread for its whole life.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- host.Tick() }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a Tick from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	game.result.FrameStepWrongThreadChecks++

	// SUPPRESS DRAW. The next step updates and does not draw.
	drawsBefore := game.draws
	if err := host.SuppressDraw(); err != nil {
		return fmt.Errorf("SuppressDraw: %w", err)
	}
	if err := host.Tick(); err != nil {
		return fmt.Errorf("suppressed Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.draws != drawsBefore {
		return fmt.Errorf("a suppressed Tick drew %d times", game.draws-drawsBefore)
	}
	game.result.FrameStepSuppressChecks++

	// EXIT, from outside a lifecycle callback -- a state that could not exist
	// before a session could outlive a call. CNA's request-exit also suppresses
	// the next draw, so the step after it updates and does not draw.
	if err := host.Exit(); err != nil {
		return fmt.Errorf("Exit outside a callback: %w", err)
	}
	drawsBefore = game.draws
	if err := host.Tick(); err != nil {
		return fmt.Errorf("Tick after Exit: %w", err)
	}
	game.result.FrameStepTicks++
	if game.draws != drawsBefore {
		return fmt.Errorf("the step after Exit drew %d times; CNA's request-exit suppresses the next draw", game.draws-drawsBefore)
	}
	game.result.FrameStepExitChecks++

	game.result.FrameStepUpdates = game.updates
	game.result.FrameStepDraws = game.draws

	// DISPOSE ends the session, and CNA delivers UnloadContent and the exiting
	// signal from inside cna_game_destroy.
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	if game.unloads != 1 {
		return fmt.Errorf("Dispose delivered %d UnloadContent callbacks, want one", game.unloads)
	}
	if _, stillLive := runtimeOf(); stillLive {
		return errors.New("a native runtime was still current after Dispose")
	}
	game.result.FrameStepDisposeChecks++
	game.result.FrameStepSessionChecks++

	// And a step AFTER Dispose starts a fresh session, because Game keeps no
	// disposed flag -- the reference does not either, which is why Dispose is
	// not idempotent anywhere in this profile.
	if err := host.Tick(); err != nil {
		return fmt.Errorf("Tick after Dispose: %w", err)
	}
	if _, revived := runtimeOf(); !revived {
		return errors.New("a Tick after Dispose created no session")
	}
	game.result.FrameStepRecreationChecks++
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("second Dispose: %w", err)
	}

	game.result.FrameStepCycles++
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// runFrameStepRunChild proves the ownership rule: Run ADOPTS a session a frame
// step created and does not destroy it, because whoever created the native game
// destroys it.
//
// This is the reference's own shape. XNA's Run calls host.Run() on a host the
// constructor already made, and CNA's Game::Run skips DoInitialize when
// hasInitialized_ is set -- so a stepped-then-run Game keeps one native game
// and one initialization.
type frameStepRunGame struct {
	initializes, updates int
	exitAfter            int
}

func (g *frameStepRunGame) Initialize(*framework.Game) error { g.initializes++; return nil }
func (g *frameStepRunGame) LoadContent(*framework.Game) error {
	return nil
}
func (g *frameStepRunGame) Update(host *framework.Game, _ framework.GameTime) error {
	g.updates++
	if g.updates >= g.exitAfter {
		return host.Exit()
	}
	return nil
}
func (g *frameStepRunGame) Draw(*framework.Game, framework.GameTime) error { return nil }
func (g *frameStepRunGame) UnloadContent(*framework.Game) error            { return nil }

func runFrameStepRunChild() error {
	game := &frameStepRunGame{exitAfter: 3}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	var result counters

	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("RunOneFrame: %w", err)
	}
	if game.initializes != 1 {
		return fmt.Errorf("RunOneFrame initialized %d times, want one", game.initializes)
	}
	current, live := interop.CurrentRuntime()
	if !live || !current.HasStandaloneSession() {
		return errors.New("RunOneFrame created no standalone session")
	}

	// Run adopts it. The session is NOT re-created, so initialization does not
	// happen again, and Run does not destroy what it did not create.
	if err := host.Run(); err != nil {
		return fmt.Errorf("Run after a frame step: %w", err)
	}
	if game.initializes != 1 {
		return fmt.Errorf("Run re-initialized the adopted session: %d initializations", game.initializes)
	}
	current, stillLive := interop.CurrentRuntime()
	if !stillLive || !current.HasStandaloneSession() {
		return errors.New("Run destroyed a session it did not create")
	}
	result.FrameStepRunAfterStepCycle++

	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	if _, after := interop.CurrentRuntime(); after {
		return errors.New("Dispose left the adopted session alive")
	}
	runtime.GC()
	result.GCStressPoints++
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
	return nil
}

// graphicsManagerGame proves GraphicsDeviceManager's configuration surface
// against a LIVE native manager, which is the half the managed tests
// structurally cannot reach: with no native manager every setter stores and
// pushes nothing, so only a run can show that the value arrives.
type graphicsManagerGame struct {
	result counters
}

func (g *graphicsManagerGame) Initialize(host *framework.Game) error {
	manager, err := framework.NewGraphicsDeviceManager(host)
	if err != nil {
		return fmt.Errorf("NewGraphicsDeviceManager: %w", err)
	}

	// The constructor's own field initializers, on a manager that now has a
	// native one behind it.
	if manager.PreferredBackBufferWidth() != framework.GraphicsDeviceManagerDefaultBackBufferWidth() ||
		manager.PreferredBackBufferHeight() != framework.GraphicsDeviceManagerDefaultBackBufferHeight() ||
		!manager.SynchronizeWithVerticalRetrace() || manager.IsFullScreen() || manager.PreferMultiSampling() {
		return fmt.Errorf("constructor defaults: %dx%d vsync=%t fullscreen=%t multisample=%t",
			manager.PreferredBackBufferWidth(), manager.PreferredBackBufferHeight(),
			manager.SynchronizeWithVerticalRetrace(), manager.IsFullScreen(), manager.PreferMultiSampling())
	}
	if graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager) != graphics.DepthFormatDepth24 {
		return fmt.Errorf("PreferredDepthStencilFormat = %v, want Depth24",
			graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager))
	}
	g.result.ManagerDefaultChecks++

	// The six framework-typed setters. Each stores AND reaches CNA's manager;
	// a refused push would surface here rather than being swallowed.
	type setting struct {
		name  string
		apply func() error
		check func() bool
	}
	for _, s := range []setting{
		{"PreferredBackBufferWidth", func() error { return manager.SetPreferredBackBufferWidth(1024) },
			func() bool { return manager.PreferredBackBufferWidth() == 1024 }},
		{"PreferredBackBufferHeight", func() error { return manager.SetPreferredBackBufferHeight(576) },
			func() bool { return manager.PreferredBackBufferHeight() == 576 }},
		{"IsFullScreen", func() error { return manager.SetIsFullScreen(false) },
			func() bool { return !manager.IsFullScreen() }},
		{"SynchronizeWithVerticalRetrace", func() error { return manager.SetSynchronizeWithVerticalRetrace(false) },
			func() bool { return !manager.SynchronizeWithVerticalRetrace() }},
		{"PreferMultiSampling", func() error { return manager.SetPreferMultiSampling(true) },
			func() bool { return manager.PreferMultiSampling() }},
		{"SupportedOrientations", func() error {
			return manager.SetSupportedOrientations(framework.DisplayOrientationLandscapeLeft)
		}, func() bool { return manager.SupportedOrientations() == framework.DisplayOrientationLandscapeLeft }},
	} {
		if err := s.apply(); err != nil {
			return fmt.Errorf("Set%s against a live manager: %w", s.name, err)
		}
		if !s.check() {
			return fmt.Errorf("Set%s did not reach the managed field", s.name)
		}
		g.result.ManagerSettersApplied++
	}

	// The three Graphics-typed ones, which travel through internal/servicebridge
	// because the framework package cannot name their enums.
	if err := graphics.SetGraphicsDeviceManagerGraphicsProfile(manager, graphics.GraphicsProfileReach); err != nil {
		return fmt.Errorf("SetGraphicsProfile: %w", err)
	}
	if graphics.GraphicsDeviceManagerGraphicsProfile(manager) != graphics.GraphicsProfileReach {
		return errors.New("GraphicsProfile did not round-trip across the package boundary")
	}
	g.result.ManagerCrossPackageSets++
	if err := graphics.SetGraphicsDeviceManagerPreferredBackBufferFormat(manager, graphics.SurfaceFormatColor); err != nil {
		return fmt.Errorf("SetPreferredBackBufferFormat: %w", err)
	}
	if graphics.GraphicsDeviceManagerPreferredBackBufferFormat(manager) != graphics.SurfaceFormatColor {
		return errors.New("PreferredBackBufferFormat did not round-trip")
	}
	g.result.ManagerCrossPackageSets++
	if err := graphics.SetGraphicsDeviceManagerPreferredDepthStencilFormat(manager, graphics.DepthFormatDepth16); err != nil {
		return fmt.Errorf("SetPreferredDepthStencilFormat: %w", err)
	}
	if graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager) != graphics.DepthFormatDepth16 {
		return errors.New("PreferredDepthStencilFormat did not round-trip")
	}
	g.result.ManagerCrossPackageSets++

	// The one validation, at its exact boundary, against a live manager: a
	// rejected value never reaches CNA either.
	if err := manager.SetPreferredBackBufferWidth(0); err == nil {
		return errors.New("SetPreferredBackBufferWidth(0) was accepted; the IL compares with bgt on zero")
	}
	if manager.PreferredBackBufferWidth() != 1024 {
		return fmt.Errorf("a rejected width stored: %d", manager.PreferredBackBufferWidth())
	}
	g.result.ManagerRangeChecks++

	// A setter from another goroutine is refused, exactly as the timing
	// setters and the window members are.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- manager.SetPreferMultiSampling(false) }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a manager setter from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.ManagerWrongThreadCheck++

	if err := manager.ApplyChanges(); err != nil {
		return fmt.Errorf("ApplyChanges: %w", err)
	}
	g.result.ManagerApplyChanges++

	// ToggleFullScreen flips through the projected setter, so the managed flag
	// follows the device.
	before := manager.IsFullScreen()
	if err := manager.ToggleFullScreen(); err != nil {
		return fmt.Errorf("ToggleFullScreen: %w", err)
	}
	if manager.IsFullScreen() == before {
		return errors.New("ToggleFullScreen did not flip the managed flag")
	}
	if err := manager.ToggleFullScreen(); err != nil {
		return fmt.Errorf("second ToggleFullScreen: %w", err)
	}
	if manager.IsFullScreen() != before {
		return errors.New("two ToggleFullScreen calls did not return to the starting state")
	}
	g.result.ManagerToggleChecks++

	// ------------------------------------------------------------------
	// Foundation 49. The two service registrations the constructor makes, and
	// what they unlock.
	// ------------------------------------------------------------------

	// Registration one: the manager itself, under the framework-package
	// IGraphicsDeviceManager contract. It is the manager object, not an
	// adapter, because that contract is nameable from the framework package.
	registeredManager, err := host.Services().GetService(
		reflect.TypeOf((*framework.IGraphicsDeviceManager)(nil)).Elem())
	if err != nil || registeredManager == nil {
		return fmt.Errorf("IGraphicsDeviceManager is not registered: %v %v", registeredManager, err)
	}
	if registeredManager != any(manager) {
		return errors.New("IGraphicsDeviceManager resolves to something other than the manager itself")
	}
	g.result.ManagerServiceChecks++

	// Registration two: an adapter over the manager, under the
	// Graphics-package IGraphicsDeviceService contract. It is an adapter
	// because no framework-package type can declare that contract's device
	// accessor, and its identity is stable across resolutions.
	registeredService, err := host.Services().GetService(
		reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if err != nil || registeredService == nil {
		return fmt.Errorf("IGraphicsDeviceService is not registered: %v %v", registeredService, err)
	}
	service, ok := registeredService.(graphics.IGraphicsDeviceService)
	if !ok {
		return errors.New("the registered device service does not satisfy the contract")
	}
	again, _ := host.Services().GetService(reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if again != registeredService {
		return errors.New("the device service adapter is not a stable identity")
	}
	g.result.ManagerServiceChecks++

	// A second manager is refused with the reference's own ArgumentException.
	if _, duplicateErr := framework.NewGraphicsDeviceManager(host); duplicateErr == nil {
		return errors.New("a second GraphicsDeviceManager was accepted")
	} else if !strings.Contains(duplicateErr.Error(), "A graphics device manager is already registered.") {
		return fmt.Errorf("a second manager reported %v, want the reference's message", duplicateErr)
	}
	g.result.ManagerDuplicateChecks++

	// THE PAYOFF. Game.GraphicsDevice has reported the reference's
	// InvalidOperationException since Foundation 43, because CNA-Go published
	// no service of its own and only a consumer could register one. It now
	// resolves the manager's.
	gameDevice, gameDeviceErr := graphics.GameGraphicsDevice(host)
	if gameDeviceErr != nil {
		return fmt.Errorf("Game.GraphicsDevice with a published service: %w", gameDeviceErr)
	}
	if gameDevice != service.GraphicsDevice() {
		return errors.New("Game.GraphicsDevice answered a device other than the published service's")
	}
	g.result.ManagerGameDeviceChecks++

	// And DrawableGameComponent.Initialize, which threw
	// MissingGraphicsDeviceService for the same reason, now resolves and
	// subscribes.
	component := framework.NewDrawableGameComponent(host)
	if err := component.Initialize(); err != nil {
		return fmt.Errorf("DrawableGameComponent.Initialize with a published service: %w", err)
	}
	componentDevice, componentErr := graphics.DrawableGameComponentGraphicsDevice(component)
	if componentErr != nil {
		return fmt.Errorf("DrawableGameComponent.GraphicsDevice: %w", componentErr)
	}
	if componentDevice != service.GraphicsDevice() {
		return errors.New("the component resolved a different device from the published service's")
	}
	if err := component.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("component Dispose: %w", err)
	}
	g.result.ManagerDrawableChecks++

	// The four protected raisers reach a consumer's handlers on the live
	// manager, which is the surface the device service publishes.
	raises := map[string]int{}
	for name, add := range map[string]func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error){
		"created":   manager.AddDeviceCreatedHandler,
		"resetting": manager.AddDeviceResettingHandler,
		"reset":     manager.AddDeviceResetHandler,
		"disposing": manager.AddDeviceDisposingHandler,
	} {
		key := name
		if _, err := add(func(sender any, args *framework.EventArgs) error {
			raises[key]++
			return nil
		}); err != nil {
			return fmt.Errorf("subscribe %s: %w", name, err)
		}
	}
	for name, raise := range map[string]func(any, *framework.EventArgs) error{
		"created":   manager.OnDeviceCreated,
		"resetting": manager.OnDeviceResetting,
		"reset":     manager.OnDeviceReset,
		"disposing": manager.OnDeviceDisposing,
	} {
		if err := raise(manager, framework.EventArgsEmpty()); err != nil {
			return fmt.Errorf("On%s: %w", name, err)
		}
		if raises[name] != 1 {
			return fmt.Errorf("On%s reached its handler %d times", name, raises[name])
		}
		g.result.ManagerEventRaiseChecks++
	}

	// CreateDevice is the operation that would raise DeviceCreated. It is one
	// of the three IGraphicsDeviceManager witnesses and is called here for two
	// reasons: it is the only way to exercise that witness against a live
	// manager, and it is the raise site the signal counters below measure.
	if err := manager.CreateDevice(); err != nil {
		return fmt.Errorf("CreateDevice: %w", err)
	}
	shouldDraw, beginErr := manager.BeginDraw()
	if beginErr != nil {
		return fmt.Errorf("BeginDraw: %w", beginErr)
	}
	if shouldDraw {
		if err := manager.EndDraw(); err != nil {
			return fmt.Errorf("EndDraw: %w", err)
		}
	}

	// The native signals CNA actually delivered. The counter is read through
	// internal/servicebridge rather than from the manager, because an exported
	// accessor for it would be public API the XNA contract does not declare.
	deliveries, haveDeliveries := servicebridge.ReadManagerSignalDeliveries(manager)
	if !haveDeliveries || len(deliveries) != interop.ManagerEventCount {
		return fmt.Errorf("manager signal deliveries unavailable: %v %d", haveDeliveries, len(deliveries))
	}
	g.result.ManagerSignalDisposed += deliveries[interop.ManagerEventDisposed]
	g.result.ManagerSignalDeviceCreated += deliveries[interop.ManagerEventDeviceCreated]
	g.result.ManagerSignalDeviceDisposing += deliveries[interop.ManagerEventDeviceDisposing]
	g.result.ManagerSignalDeviceReset += deliveries[interop.ManagerEventDeviceReset]
	g.result.ManagerSignalDeviceResetting += deliveries[interop.ManagerEventDeviceResetting]

	if err := manager.Dispose(true); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	// Dispose unregisters both services, and only its own: the reference's own
	// `if (GetService(...) == this)` guard.
	afterManager, _ := host.Services().GetService(reflect.TypeOf((*framework.IGraphicsDeviceManager)(nil)).Elem())
	afterService, _ := host.Services().GetService(reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if afterManager != nil || afterService != nil {
		return fmt.Errorf("Dispose left registrations behind: manager=%v service=%v", afterManager != nil, afterService != nil)
	}
	g.result.ManagerServiceRemovalCheck++

	g.result.ManagerCycles++
	return nil
}

func (g *graphicsManagerGame) LoadContent(*framework.Game) error { return nil }
func (g *graphicsManagerGame) Update(host *framework.Game, _ framework.GameTime) error {
	return host.Exit()
}
func (g *graphicsManagerGame) Draw(*framework.Game, framework.GameTime) error { return nil }
func (g *graphicsManagerGame) UnloadContent(*framework.Game) error            { return nil }

func runGraphicsManagerChild() error {
	game := &graphicsManagerGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	if err := host.Run(); err != nil {
		return err
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}
