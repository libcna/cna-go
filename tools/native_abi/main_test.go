package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The native game-event bridge is the first CNA-Go surface whose correctness
// depends on values the compiler cannot infer from a Go signature: an event
// identity, a registration-handle width, a callback shape and the position of
// the caller context in a five-parameter prototype. Each of those is pinned by
// a _Static_assert or an incompatible-pointer-types assignment, and each of the
// mutations below removes exactly one pin and requires the compile to fail.
//
// The mutations act on real source. Two translation units carry the pins:
//
//   - internal/interop/bridge.c compiled with CNA-Go's private manifest and no
//     canonical CNA header, which is exactly how the cgo build sees it;
//   - tools/native_abi/testdata/probe.c compiled WITH the canonical header,
//     which is the only place the private manifest and the canonical
//     declarations are compiled together.
//
// A mutation that still compiles is a hole in the ABI evidence, not a passing
// test.

type sourceMutation struct {
	name string
	file string
	old  string
	new  string
}

var bridgeMutations = []sourceMutation{
	{
		name: "wrong-event-constant",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED UINT32_C(0)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED UINT32_C(7)",
	},
	{
		name: "swapped-event-constants",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_DISPOSED UINT32_C(2)\n#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(3)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_DISPOSED UINT32_C(3)\n#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(2)",
	},
	{
		name: "registration-handle-narrowed",
		file: "abi_manifest.h",
		old:  "typedef CNA_Handle CNA_GameEventRegistrationHandle;",
		new:  "typedef uint32_t CNA_GameEventRegistrationHandle;",
	},
	{
		name: "event-identity-count-drift",
		file: "abi_manifest.h",
		old:  "#define CNA_GAME_EVENT_MAXIMUM CNA_GAME_EVENT_EXITING",
		new:  "#define CNA_GAME_EVENT_MAXIMUM UINT32_C(9)",
	},
	{
		name: "missing-unsubscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_unsubscribe) \\\n",
		new:  "",
	},
	{
		name: "missing-subscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_subscribe) \\\n",
		new:  "",
	},
	{
		name: "callback-returns-a-result",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef CNA_Result (*CNA_GameEventCallback)(void*);",
	},
	{
		name: "callback-drops-user-data",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef void (*CNA_GameEventCallback)(void);",
	},
	{
		name: "callback-takes-a-game-handle",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef void (*CNA_GameEventCallback)(CNA_Handle, void*);",
	},
	{
		name: "unsubscribe-takes-the-game",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);",
		new:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_Handle, CNA_GameEventRegistrationHandle);",
	},
	{
		name: "bridge-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_EVENT_DISPOSED = 2,",
		new:  "    CNA_GO_GAME_EVENT_DISPOSED = 3,",
	},
	{
		name: "bridge-mirror-count-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_EVENT_COUNT = 4",
		new:  "    CNA_GO_GAME_EVENT_COUNT = 5",
	},
	// The four optional frame-hook members. CNA-Go assigns each one behind a
	// declared capability, so their order and their types are both load-bearing
	// in a way they were not while only `initialize` was installed.
	{
		name: "frame-hook-member-order-drift",
		file: "abi_manifest.h",
		old:  "    CNA_GameLifecycleCallback begin_run;\n    CNA_GameLifecycleCallback end_run;\n    CNA_GameBeginDrawCallback begin_draw;",
		new:  "    CNA_GameBeginDrawCallback begin_draw;\n    CNA_GameLifecycleCallback begin_run;\n    CNA_GameLifecycleCallback end_run;",
	},
	{
		name: "begin-draw-member-narrowed-to-the-lifecycle-shape",
		file: "abi_manifest.h",
		old:  "    CNA_GameBeginDrawCallback begin_draw;",
		new:  "    CNA_GameLifecycleCallback begin_draw;",
	},
	{
		name: "begin-draw-callback-drops-the-out-parameter",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*CNA_GameBeginDrawCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_Bool*, CNA_CallbackError*);",
		new:  "typedef CNA_Result (*CNA_GameBeginDrawCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_CallbackError*);",
	},
	// The Foundation 42 timing setters, from the side the bridge translation
	// unit can see: a prototype that loses the game handle, gains a parameter,
	// or disappears from the required set.
	{
		name: "inactive-sleep-time-takes-no-game",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_inactive_sleep_time_ticks_fn)(CNA_Handle, int64_t);",
		new:  "typedef CNA_Result (*cna_game_set_inactive_sleep_time_ticks_fn)(int64_t);",
	},
	{
		name: "suppress-draw-takes-a-frame-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_suppress_draw_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_game_suppress_draw_fn)(CNA_Handle, uint32_t);",
	},
	{
		name: "missing-suppress-draw-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_suppress_draw) \\\n",
		new:  "",
	},
	{
		name: "missing-reset-elapsed-time-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_reset_elapsed_time) \\\n",
		new:  "",
	},
	{
		name: "frame-hook-mask-bit-collision",
		file: "bridge.h",
		old:  "    CNA_GO_FRAME_HOOK_BEGIN_DRAW = 1u << 2,",
		new:  "    CNA_GO_FRAME_HOOK_BEGIN_DRAW = 1u << 1,",
	},
	// Foundation 45. The window family is a SECOND identity space that also
	// starts at zero, so every one of these mutations produces a value that
	// looks entirely legitimate in the other family's table.
	{
		name: "window-event-identity-drift",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED UINT32_C(1)",
		new:  "#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED UINT32_C(2)",
	},
	{
		name: "bridge-window-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED = 2,",
		new:  "    CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED = 1,",
	},
	{
		name: "window-event-count-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_WINDOW_EVENT_COUNT = 3",
		new:  "    CNA_GO_GAME_WINDOW_EVENT_COUNT = 4",
	},
	{
		name: "missing-window-subscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_window_subscribe)",
		new:  "",
	},
	// Foundation 47's two frame steps. Both are CNA_Result(CNA_Handle), which
	// is the shape cna_game_run, cna_game_request_exit and cna_game_destroy
	// also have -- so a manifest that bound one where another belongs compiles
	// cleanly, and only the loader's dladdr identity check separates them.
	{
		name: "missing-tick-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_tick) \\\n",
		new:  "",
	},
	{
		name: "missing-run-one-frame-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_run_one_frame) \\\n",
		new:  "",
	},
	{
		name: "tick-takes-a-frame-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_tick_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_game_tick_fn)(CNA_Handle, uint32_t);",
	},
	// Foundation 48's GraphicsDeviceManager setters. Nine of the ten share the
	// shape CNA_Result(CNA_Handle, <one value>), so a manifest that bound one
	// where another belongs compiles whenever the value types match -- which
	// is why the width and height routes, both CNA_Result(CNA_Handle, int32_t),
	// are separated by the loader's dladdr identity check rather than by the
	// compiler.
	{
		name: "missing-manager-apply-changes-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_apply_changes) \\\n",
		new:  "",
	},
	{
		name: "missing-manager-full-screen-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_set_is_full_screen) \\\n",
		new:  "",
	},
	{
		name: "missing-window-title-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_set_window_title) \\\n",
		new:  "",
	},
	// The admission policy itself. The qualified encoded constant is what the
	// loader's rejection message reports, and the floor is what it enforces; a
	// policy that reported one range and enforced another would admit or refuse
	// libraries for reasons no diagnostic could explain.
	{
		name: "qualified-abi-constant-disagrees-with-the-floor",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_QUALIFIED_VERSION = 0x00001500u,",
		new:  "    CNA_GO_ABI_QUALIFIED_VERSION = 0x00001400u,",
	},
	{
		name: "abi-floor-raised-without-re-encoding",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MINIMUM_MINOR = 21,",
		new:  "    CNA_GO_ABI_MINIMUM_MINOR = 22,",
	},
	{
		name: "abi-decoding-mirror-drift",
		file: "bridge.h",
		old:  "#define CNA_GO_ABI_MINOR_OF(version) (((uint32_t)(version) >> 8) & UINT32_C(0xFF))",
		new:  "#define CNA_GO_ABI_MINOR_OF(version) (((uint32_t)(version) >> 4) & UINT32_C(0xFF))",
	},
}

// A few pins cannot live in the bridge translation unit at all, and the reason
// is worth stating rather than working around. GNU C converts silently between
// void* and a function pointer, so swapping the callback and the caller context
// in cna_game_subscribe's prototype still compiles against a manifest that
// declares the same swap -- the manifest would simply be self-consistently
// wrong. The pin has to compare the manifest with the canonical declaration,
// which only the probe translation unit does.
var probeMutations = []sourceMutation{
	{
		name: "subscribe-user-data-before-callback",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, void*, CNA_GameEventCallback, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "subscribe-drops-the-out-registration",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*);",
	},
	{
		name: "subscribe-returns-the-registration",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_GameEventRegistrationHandle (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "unsubscribe-takes-a-narrowed-handle",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);",
		new:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(uint32_t);",
	},
	{
		name: "manifest-disagrees-with-canonical-header",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(3)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(2)",
	},
	// The two Foundation 42 mutations the BRIDGE translation unit cannot catch,
	// and the reason is worth stating. C converts silently between integer
	// widths and between a narrow unsigned return and a wider one, so a
	// manifest that narrowed a tick count to 32 bits -- capping the target step
	// at about 3.6 minutes -- or that turned a result code into a CNA_Bool
	// would still compile against itself. Only comparing the manifest with the
	// CANONICAL declaration catches them, which is what the probe's assignment
	// pins do.
	{
		name: "target-elapsed-time-narrowed-to-32-bits",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_target_elapsed_time_ticks_fn)(CNA_Handle, int64_t);",
		new:  "typedef CNA_Result (*cna_game_set_target_elapsed_time_ticks_fn)(CNA_Handle, int32_t);",
	},
	// Foundation 47. C narrows a CNA_Result to a CNA_Bool silently, so a
	// manifest that turned a frame step's result code into a Boolean would
	// compile against itself and report success for every failing frame.
	{
		name: "manager-back-buffer-width-narrowed",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_width_fn)(CNA_Handle, int32_t);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_width_fn)(CNA_Handle, int16_t);",
	},
	{
		name: "manager-full-screen-takes-an-int",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_set_is_full_screen_fn)(CNA_Handle, CNA_Bool);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_set_is_full_screen_fn)(CNA_Handle, int32_t);",
	},
	{
		name: "manager-apply-changes-takes-a-flag",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_apply_changes_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_apply_changes_fn)(CNA_Handle, CNA_Bool);",
	},
	{
		name: "run-one-frame-returns-a-bool",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_run_one_frame_fn)(CNA_Handle);",
		new:  "typedef CNA_Bool (*cna_game_run_one_frame_fn)(CNA_Handle);",
	},
	{
		name: "mouse-visibility-returns-the-previous-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_is_mouse_visible_fn)(CNA_Handle, CNA_Bool);",
		new:  "typedef CNA_Bool (*cna_game_set_is_mouse_visible_fn)(CNA_Handle, CNA_Bool);",
	},
	{
		name: "probe-callback-returns-a-result",
		file: "probe.c",
		old:  "static void cna_go_probe_game_event(void* context) { (void)context; }",
		new:  "static CNA_Result cna_go_probe_game_event(void* context) { (void)context; return 0; }",
	},
	{
		name: "probe-callback-takes-a-game-handle",
		file: "probe.c",
		old:  "static void cna_go_probe_game_event(void* context) { (void)context; }",
		new:  "static void cna_go_probe_game_event(CNA_Handle game, void* context) { (void)game; (void)context; }",
	},
	// The begin_draw hook is the one CNA-Go installs that carries a value
	// channel, so its out-parameter is pinned from three directions: dropping
	// it, moving it past the error, and narrowing it to the result channel.
	{
		name: "probe-begin-draw-drops-the-out-parameter",
		file: "probe.c",
		old:  "    CNA_Bool* out_should_draw,\n    CNA_CallbackError* out_error) {\n    (void)game; (void)game_time; (void)context; (void)out_error;",
		new:  "    CNA_CallbackError* out_error) {\n    (void)game; (void)game_time; (void)context; (void)out_error;\n    CNA_Bool* out_should_draw = NULL;",
	},
	{
		name: "probe-begin-draw-swaps-the-out-parameter-and-the-error",
		file: "probe.c",
		old:  "    CNA_Bool* out_should_draw,\n    CNA_CallbackError* out_error) {",
		new:  "    CNA_CallbackError* out_error,\n    CNA_Bool* out_should_draw) {",
	},
	{
		name: "probe-begin-draw-returns-the-drawing-decision",
		file: "probe.c",
		old:  "static CNA_Result cna_go_probe_begin_draw(",
		new:  "static CNA_Bool cna_go_probe_begin_draw(",
	},
	// And the lifecycle shape the other three hooks install.
	{
		name: "probe-lifecycle-drops-the-game-time",
		file: "probe.c",
		old:  "static CNA_Result cna_go_probe_lifecycle(\n    CNA_Handle game,\n    const CNA_GameTime* game_time,\n    void* context,",
		new:  "static CNA_Result cna_go_probe_lifecycle(\n    CNA_Handle game,\n    void* context,",
	},
	// The ABI ADMISSION POLICY, from the side only this translation unit can
	// check: the policy's own numbers against the canonical header's.
	{
		name: "admission-floor-above-the-canonical-minor",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MINIMUM_MINOR = 21,",
		new:  "    CNA_GO_ABI_MINIMUM_MINOR = 99,",
	},
	{
		name: "admission-major-outside-the-canonical-major",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MAJOR = 0,",
		new:  "    CNA_GO_ABI_MAJOR = 1,",
	},
	{
		name: "encoded-version-mirror-shifts-the-minor-wrongly",
		file: "bridge.h",
		old:  "     (((uint32_t)(minor) & UINT32_C(0xFF)) << 8) | \\",
		new:  "     (((uint32_t)(minor) & UINT32_C(0xFF)) << 4) | \\",
	},
	{
		name: "bridge-callback-result-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_RESULT_CALLBACK = 9,",
		new:  "    CNA_GO_RESULT_CALLBACK = 10,",
	},
	{
		name: "bridge-success-result-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_RESULT_SUCCESS = 0,",
		new:  "    CNA_GO_RESULT_SUCCESS = 1,",
	},
	// A required symbol that CNA no longer declares. The bridge translation
	// unit cannot see this: it resolves symbols by string at run time, so a
	// stale entry compiles and fails only when a consumer loads a library.
	// Foundation 45's window prototypes, from the side only the probe can
	// check. C converts silently between integer widths and between a struct
	// pointer and another struct pointer of the same size, so a manifest that
	// narrowed a client dimension or wrote a Viewport where a Rectangle
	// belongs would compile happily against itself.
	{
		name: "window-subscribe-swaps-the-callback-and-the-context",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_subscribe_fn)(CNA_Handle, CNA_GameWindowEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_window_subscribe_fn)(CNA_Handle, CNA_GameWindowEvent, void*, CNA_GameEventCallback, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "window-title-route-takes-a-bare-pointer",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_window_title_fn)(CNA_Handle, CNA_StringView);",
		new:  "typedef CNA_Result (*cna_game_set_window_title_fn)(CNA_Handle, const char*);",
	},
	{
		name: "end-screen-device-change-narrows-the-client-size",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_end_screen_device_change_fn)(CNA_Handle, CNA_StringView, int32_t, int32_t);",
		new:  "typedef CNA_Result (*cna_game_window_end_screen_device_change_fn)(CNA_Handle, CNA_StringView, int16_t, int16_t);",
	},
	{
		name: "client-bounds-writes-a-viewport",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_get_client_bounds_fn)(CNA_Handle, CNA_Rectangle*);",
		new:  "typedef CNA_Result (*cna_game_window_get_client_bounds_fn)(CNA_Handle, CNA_Viewport*);",
	},
	{
		name: "native-window-handle-narrowed-to-32-bits",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_get_native_handle_ext_fn)(CNA_Handle, uint64_t*);",
		new:  "typedef CNA_Result (*cna_game_window_get_native_handle_ext_fn)(CNA_Handle, uint32_t*);",
	},
	{
		name: "screen-device-name-copy-drops-its-capacity",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_copy_screen_device_name_fn)(CNA_Handle, char*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_game_window_copy_screen_device_name_fn)(CNA_Handle, char*, uint64_t*);",
	},
	{
		name: "stale-required-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_keyboard_get_state)",
		new:  "    X(cna_game_removed_route_ext) \\\n    X(cna_keyboard_get_state)",
	},
}

// TestManifestProbeRefusesACanonicalHeader proves the guard that makes the
// manifest probe worth running. abi_manifest.h suppresses its own definitions
// whenever a CNA header is present, so a canonical header leaking into this
// translation unit would turn it into a second canonical probe: it would report
// perfect agreement while measuring the same types twice.
func TestManifestProbeRefusesACanonicalHeader(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror",
		"-I"+headers, "-I"+directory, "-include", "CNA/C/cna.h",
		"-c", filepath.Join(directory, "manifest_probe.c"),
		"-o", filepath.Join(directory, "leaked.o"))
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("the manifest probe compiled with a canonical CNA header in scope")
	}
	if !strings.Contains(string(output), "must not see a canonical CNA header") {
		t.Fatalf("the manifest probe failed for the wrong reason:\n%s", output)
	}
}

// TestBridgeTranslationUnitCompilesUnmutated is the control. Every mutation
// below is only evidence if the unmutated source compiles cleanly under the
// same flags.
func TestBridgeTranslationUnitCompilesUnmutated(t *testing.T) {
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageInteropSources(t, root, directory, nil)
	if output, err := compileBridge(directory); err != nil {
		t.Fatalf("unmutated bridge translation unit did not compile: %v\n%s", err, output)
	}
}

// TestProbeTranslationUnitCompilesUnmutated is the same control for the probe,
// which is the only translation unit that sees the canonical CNA header and
// CNA-Go's private manifest at once.
func TestProbeTranslationUnitCompilesUnmutated(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	if output, err := compileProbe(directory, headers); err != nil {
		t.Fatalf("unmutated probe translation unit did not compile: %v\n%s", err, output)
	}
}

// TestBridgeABIMutationsFailToCompile removes one ABI pin at a time from the
// translation unit the cgo build actually compiles.
func TestBridgeABIMutationsFailToCompile(t *testing.T) {
	root := repositoryRoot(t)
	for _, mutation := range bridgeMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageInteropSources(t, root, directory, &mutation)
			output, err := compileBridge(directory)
			if err == nil {
				t.Fatalf("mutation %q compiled cleanly; the ABI pin it removes is not enforced", mutation.name)
			}
			_ = output
		})
	}
}

// TestProbeABIMutationsFailToCompile does the same for the canonical-header
// probe, including the one mutation that makes CNA-Go's private manifest
// disagree with the shipped CNA declarations.
func TestProbeABIMutationsFailToCompile(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	for _, mutation := range probeMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageProbeSources(t, root, directory, &mutation)
			if _, err := compileProbe(directory, headers); err == nil {
				t.Fatalf("mutation %q compiled cleanly; the ABI pin it removes is not enforced", mutation.name)
			}
		})
	}
}

// TestManifestRoutesAreSelfDescribing keeps the measured prototype table and
// the bridge's required-symbol list from drifting apart. They cannot drift any
// more: the table IS the manifest now, parsed from the same X-macro list the
// cgo build resolves, so this proves the parser reads it rather than that two
// hand-maintained copies still agree.
func TestManifestRoutesAreSelfDescribing(t *testing.T) {
	manifest := readSource(t, filepath.Join(repositoryRoot(t), "internal", "interop", "abi_manifest.h"))
	routes, err := parseManifestRoutes(manifest)
	if err != nil {
		t.Fatalf("parse the manifest: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("the manifest declares no bound routes")
	}
	seen := map[string]bool{}
	for _, entry := range routes {
		if seen[entry.name] {
			t.Fatalf("%s appears twice in the required-symbol list", entry.name)
		}
		seen[entry.name] = true
		if !strings.Contains(manifest, "X("+entry.name+")") {
			t.Fatalf("%s is measured but the bridge never resolves it", entry.name)
		}
	}
	for _, required := range []string{"cna_game_subscribe", "cna_game_unsubscribe", "cna_get_abi_version"} {
		if !seen[required] {
			t.Fatalf("%s is bound by the bridge but has no measured prototype", required)
		}
	}
}

// TestManifestRoutesRejectAStaleEntry proves the parser refuses a
// required-symbol entry with no prototype of its own rather than counting it as
// a route with zero parameters.
func TestManifestRoutesRejectAStaleEntry(t *testing.T) {
	manifest := readSource(t, filepath.Join(repositoryRoot(t), "internal", "interop", "abi_manifest.h"))
	mutated := strings.Replace(manifest,
		"    X(cna_keyboard_get_state)",
		"    X(cna_game_removed_route_ext) \\\n    X(cna_keyboard_get_state)", 1)
	if mutated == manifest {
		t.Fatal("the required-symbol list no longer ends with cna_keyboard_get_state")
	}
	if _, err := parseManifestRoutes(mutated); err == nil {
		t.Fatal("a required symbol with no typedef was accepted")
	}
}

// TestManifestProbeCompilesUnmutated is the control for the manifest-only
// translation unit: the one that measures CNA-Go's private declarations in the
// same environment cgo gives bridge.c.
func TestManifestProbeCompilesUnmutated(t *testing.T) {
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	if output, err := compileManifestProbe(directory); err != nil {
		t.Fatalf("unmutated manifest probe did not compile: %v\n%s", err, output)
	}
}

// TestUnmutatedProbesAgreeOnEveryMeasurement is the control for the comparison
// the layout mutations below falsify.
func TestUnmutatedProbesAgreeOnEveryMeasurement(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	canonical, manifest := measureStagedProbes(t, directory, headers)
	shared := 0
	for key, value := range canonical {
		other, ok := manifest[key]
		if !ok {
			continue
		}
		shared++
		if other != value {
			t.Fatalf("%s: canonical %d, manifest %d", key, value, other)
		}
	}
	if shared == 0 {
		t.Fatal("the two probes share no measurement, so comparing them proves nothing")
	}
}

// layoutMutations change CNA-Go's private manifest in ways that COMPILE
// cleanly everywhere -- in the bridge, in the canonical probe, and in the
// manifest probe -- because C aggregates are written by field name. Each one
// still changes the bytes the shipped binding would pass to CNA. They are the
// class the ABI evidence could not see until the manifest's own declarations
// were measured, and each must produce a divergence.
var layoutMutations = []sourceMutation{
	{
		name: "sprite-command-source-and-colour-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_Rectangle source;\n    CNA_Color color;",
		new:  "    CNA_Color color;\n    CNA_Rectangle source;",
	},
	{
		name: "keyboard-state-word-count-doubled",
		file: "abi_manifest.h",
		old:  "    uint64_t pressed_key_words[4];",
		new:  "    uint64_t pressed_key_words[8];",
	},
	{
		name: "viewport-depth-widened-to-double",
		file: "abi_manifest.h",
		old:  "    float min_depth;\n    float max_depth;",
		new:  "    double min_depth;\n    double max_depth;",
	},
	{
		name: "texture-info-struct-size-widened",
		file: "abi_manifest.h",
		old:  "typedef struct CNA_Texture2DInfo {\n    uint32_t struct_size;",
		new:  "typedef struct CNA_Texture2DInfo {\n    uint64_t struct_size;",
	},
	{
		name: "game-time-tick-count-narrowed",
		file: "abi_manifest.h",
		old:  "    int64_t total_game_time_ticks;\n    int64_t elapsed_game_time_ticks;",
		new:  "    int32_t total_game_time_ticks;\n    int64_t elapsed_game_time_ticks;",
	},
	{
		name: "string-view-length-narrowed",
		file: "abi_manifest.h",
		old:  "typedef struct CNA_StringView { const char* data; uint64_t byte_length; } CNA_StringView;",
		new:  "typedef struct CNA_StringView { const char* data; uint32_t byte_length; } CNA_StringView;",
	},
}

// TestLayoutMutationsDiverge is the falsifiability proof for the manifest-side
// measurement. Every mutation compiles; the comparison is what catches it.
func TestLayoutMutationsDiverge(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	for _, mutation := range layoutMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageProbeSources(t, root, directory, &mutation)
			canonical, manifest := measureStagedProbes(t, directory, headers)
			for key, value := range canonical {
				if other, ok := manifest[key]; ok && other != value {
					return
				}
			}
			t.Fatalf("mutation %q changed no measured value; the manifest layout is not pinned", mutation.name)
		})
	}
}

// measureStagedProbes compiles and runs both probes over one staged source
// tree and returns their measurements.
func measureStagedProbes(t *testing.T, directory, headers string) (map[string]int, map[string]int) {
	t.Helper()
	canonicalBinary := filepath.Join(directory, "canonical-probe")
	arguments := []string{
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I" + headers, "-I" + directory, "-DCNA_GO_LAYOUT_ONLY",
		filepath.Join(directory, "probe.c"), "-o", canonicalBinary,
	}
	if output, err := exec.Command("gcc", arguments...).CombinedOutput(); err != nil {
		t.Fatalf("canonical probe did not compile: %v\n%s", err, output)
	}
	manifestBinary := filepath.Join(directory, "manifest-probe")
	arguments = []string{
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I" + directory,
		filepath.Join(directory, "manifest_probe.c"), "-o", manifestBinary,
	}
	if output, err := exec.Command("gcc", arguments...).CombinedOutput(); err != nil {
		t.Fatalf("manifest probe did not compile: %v\n%s", err, output)
	}
	return runStagedProbe(t, canonicalBinary), runStagedProbe(t, manifestBinary)
}

func runStagedProbe(t *testing.T, binary string) map[string]int {
	t.Helper()
	output, err := exec.Command(binary).Output()
	if err != nil {
		t.Fatalf("run %s: %v", binary, err)
	}
	measurements := map[string]int{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		if value, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
			measurements[parts[0]] = value
		}
	}
	return measurements
}

func compileBridge(directory string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-c", filepath.Join(directory, "bridge.c"),
		"-o", filepath.Join(directory, "bridge.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

func compileProbe(directory, headers string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I"+headers,
		"-c", filepath.Join(directory, "probe.c"),
		"-o", filepath.Join(directory, "probe.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

// compileManifestProbe deliberately passes NO canonical include root. That is
// the whole point of the translation unit: it must measure CNA-Go's private
// declarations, and it refuses to compile if a CNA header reaches it.
func compileManifestProbe(directory string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-c", filepath.Join(directory, "manifest_probe.c"),
		"-o", filepath.Join(directory, "manifest_probe.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

func stageInteropSources(t *testing.T, root, directory string, mutation *sourceMutation) {
	t.Helper()
	for _, name := range []string{"bridge.c", "bridge.h", "abi_manifest.h"} {
		content := readSource(t, filepath.Join(root, "internal", "interop", name))
		if mutation != nil && mutation.file == name {
			content = applyMutation(t, content, *mutation)
		}
		writeSource(t, filepath.Join(directory, name), content)
	}
}

func stageProbeSources(t *testing.T, root, directory string, mutation *sourceMutation) {
	t.Helper()
	// The staged copy is flat, so both probes' relative includes of the private
	// headers become sibling includes. Nothing else about them changes.
	flatten := func(source string) string {
		source = strings.Replace(source, `#include "../../../internal/interop/abi_manifest.h"`, `#include "abi_manifest.h"`, 1)
		return strings.Replace(source, `#include "../../../internal/interop/bridge.h"`, `#include "bridge.h"`, 1)
	}
	staged := map[string]string{
		"probe.c":          flatten(readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "probe.c"))),
		"manifest_probe.c": flatten(readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "manifest_probe.c"))),
		"measurements.inc": readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "measurements.inc")),
		"abi_manifest.h":   readSource(t, filepath.Join(root, "internal", "interop", "abi_manifest.h")),
		"bridge.h":         readSource(t, filepath.Join(root, "internal", "interop", "bridge.h")),
	}
	if mutation != nil {
		content, ok := staged[mutation.file]
		if !ok {
			t.Fatalf("probe mutation %q names an unstaged file %q", mutation.name, mutation.file)
		}
		staged[mutation.file] = applyMutation(t, content, *mutation)
	}
	for name, content := range staged {
		writeSource(t, filepath.Join(directory, name), content)
	}
}

func applyMutation(t *testing.T, content string, mutation sourceMutation) string {
	t.Helper()
	if !strings.Contains(content, mutation.old) {
		t.Fatalf("mutation %q no longer matches %s; the evidence it carries is stale", mutation.name, mutation.file)
	}
	return strings.Replace(content, mutation.old, mutation.new, 1)
}

func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeSource(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	t.Fatal("could not locate the module root")
	return ""
}

// canonicalHeaderRoot resolves the canonical CNA include tree the way every
// other native gate does: from CNA_C_API_INCLUDE when it is set, and otherwise
// from the path the qualification artifact documents. A checkout without the
// headers skips rather than fails, because the probe mutations measure the
// canonical declarations and cannot be evaluated without them.
func canonicalHeaderRoot(t *testing.T) string {
	t.Helper()
	candidates := []string{os.Getenv("CNA_C_API_INCLUDE")}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "deps", "cna-c-abi-0.21.0", "include"))
	}
	candidates = append(candidates, filepath.Join(repositoryRoot(t), "..", "..", "cnanext", "modules", "c-api", "include"))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(candidate, "CNA", "C", "cna.h")); err == nil {
			return candidate
		}
	}
	t.Skip("canonical CNA C headers are not available; set CNA_C_API_INCLUDE")
	return ""
}
