package main

import (
	"os"
	"os/exec"
	"path/filepath"
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

// TestBoundFunctionsCoverTheEventSurface keeps the measured prototype table and
// the bridge's required-symbol list from drifting apart: a symbol the bridge
// resolves but the table does not describe would silently leave
// PROTOTYPE_TYPE_POSITIONS short.
func TestBoundFunctionsCoverTheEventSurface(t *testing.T) {
	for _, name := range []string{"cna_game_subscribe", "cna_game_unsubscribe"} {
		if _, ok := parameterCounts[name]; !ok {
			t.Fatalf("%s is bound by the bridge but has no measured prototype", name)
		}
	}
	manifest := readSource(t, filepath.Join(repositoryRoot(t), "internal", "interop", "abi_manifest.h"))
	for name := range parameterCounts {
		if !strings.Contains(manifest, "X("+name+")") {
			t.Fatalf("%s is measured but the bridge never resolves it", name)
		}
	}
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
	probe := readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "probe.c"))
	// The staged copy is flat, so the probe's relative include of the private
	// manifest becomes a sibling include. Nothing else about it changes.
	probe = strings.Replace(probe, `#include "../../../internal/interop/abi_manifest.h"`, `#include "abi_manifest.h"`, 1)
	manifest := readSource(t, filepath.Join(root, "internal", "interop", "abi_manifest.h"))
	if mutation != nil {
		switch mutation.file {
		case "probe.c":
			probe = applyMutation(t, probe, *mutation)
		case "abi_manifest.h":
			manifest = applyMutation(t, manifest, *mutation)
		default:
			t.Fatalf("probe mutation %q names an unstaged file %q", mutation.name, mutation.file)
		}
	}
	writeSource(t, filepath.Join(directory, "probe.c"), probe)
	writeSource(t, filepath.Join(directory, "abi_manifest.h"), manifest)
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
		candidates = append(candidates, filepath.Join(home, "deps", "cna-c-abi-0.7.0", "include"))
	}
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
