// Command native_abi compiles CNA-Go's private ABI manifest against the
// canonical CNA headers and independently admits a selected shared library.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/openeggbert/cna-go/internal/interop"
)

type report struct {
	SchemaVersion          int            `json:"schema_version"`
	Status                 string         `json:"status"`
	HeaderRoot             string         `json:"header_root"`
	NativeLibrary          string         `json:"native_library"`
	NativeLibrarySHA256    string         `json:"native_library_sha256"`
	ABIVersion             string         `json:"abi_version"`
	BoundFunctions         int            `json:"BOUND_FUNCTIONS"`
	PrototypeTypePositions int            `json:"PROTOTYPE_TYPE_POSITIONS"`
	CGoMeasurements        int            `json:"C_GO_MEASUREMENTS"`
	Layouts                int            `json:"LAYOUTS"`
	Callbacks              int            `json:"CALLBACKS"`
	Constants              int            `json:"CONSTANTS"`
	MissingHeaderSymbols   []string       `json:"MISSING_HEADER_SYMBOLS"`
	MissingLibrarySymbols  []string       `json:"MISSING_LIBRARY_SYMBOLS"`
	ABIMismatches          []string       `json:"ABI_MISMATCHES"`
	Measurements           map[string]int `json:"measurements"`
	Functions              []string       `json:"functions"`
}

var parameterCounts = map[string]int{
	"cna_get_abi_version": 0, "cna_error_get_last_message_size": 1,
	"cna_error_copy_last_message": 3, "cna_game_create": 2,
	"cna_game_set_frame_hooks_ext": 2, "cna_game_run": 1,
	"cna_game_request_exit": 1, "cna_game_destroy": 1,
	"cna_game_subscribe": 5, "cna_game_unsubscribe": 1,
	// Foundation 42: the Game timing and presentation setters, and the two
	// frame commands. All six were already exported by the pinned artifact and
	// had never been reached from Go.
	"cna_game_set_is_mouse_visible": 2, "cna_game_set_is_fixed_time_step": 2,
	"cna_game_set_target_elapsed_time_ticks": 2,
	"cna_game_set_inactive_sleep_time_ticks": 2,
	"cna_game_reset_elapsed_time":            1, "cna_game_suppress_draw": 1,
	"cna_graphics_device_manager_create":              2,
	"cna_graphics_device_manager_get_graphics_device": 2,
	"cna_graphics_device_manager_destroy":             1, "cna_game_get_graphics_device": 2,
	"cna_graphics_device_get_viewport": 2, "cna_graphics_device_clear_rgba": 5,
	"cna_texture2d_create_from_encoded_memory": 5, "cna_texture2d_get_info": 2,
	"cna_texture2d_destroy": 1, "cna_sprite_batch_create": 2,
	"cna_sprite_batch_begin": 2, "cna_sprite_batch_submit_scaled_many": 3,
	"cna_sprite_batch_end": 1, "cna_sprite_batch_destroy": 1,
	"cna_keyboard_get_state": 2,
}

func main() {
	headers := flag.String("headers", "../../cna/modules/c-api/include", "canonical CNA C include root")
	library := flag.String("library", "", "absolute path to the ABI 0.7 shared library")
	output := flag.String("output", "docs/generated/native-abi-report.json", "JSON report path")
	flag.Parse()
	result, err := verify(*headers, *library)
	if writeErr := writeReport(*output, result); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("BOUND_FUNCTIONS=%d ABI_MISMATCHES=%d MISSING_LIBRARY_SYMBOLS=%d\n", result.BoundFunctions, len(result.ABIMismatches), len(result.MissingLibrarySymbols))
}

func verify(headerRoot, library string) (report, error) {
	result := report{
		SchemaVersion: 1, Status: "FAIL", HeaderRoot: headerRoot, NativeLibrary: library,
		ABIVersion: "0.7.0", Measurements: map[string]int{},
		MissingHeaderSymbols: []string{}, MissingLibrarySymbols: []string{}, ABIMismatches: []string{},
	}
	for name := range parameterCounts {
		result.Functions = append(result.Functions, name)
		result.PrototypeTypePositions += 1 + parameterCounts[name]
	}
	sort.Strings(result.Functions)
	result.BoundFunctions = len(result.Functions)
	// CNA_GameLifecycleCallback, CNA_GameBeginDrawCallback and
	// CNA_GameEventCallback: every callback typedef the bridge installs is
	// pinned by an incompatible-pointer-types assignment in probe.c.
	result.Callbacks = 3
	// The _Static_asserts in the compiled ABI probe: the five original ones,
	// the five that compare the four canonical game-event identities and
	// CNA_GAME_EVENT_MAXIMUM against CNA-Go's private manifest copy, and the
	// five that pin the frame-hook table's member order.
	//
	// bridge.c carries its own asserts, including the same five member-order
	// ones and the frame-hook mask, and is not counted here: it is compiled by
	// cgo against the MANIFEST rather than the canonical header, which is what
	// makes the pair of them pin the layout from both sides.
	result.Constants = 15
	root, err := filepath.Abs(headerRoot)
	if err != nil {
		return result, err
	}
	temporary, err := os.MkdirTemp("", "cna-go-native-abi-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(temporary)
	binary := filepath.Join(temporary, "probe")
	commonCompilerArgs := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types", "-I" + root}
	objectArgs := append(append([]string{}, commonCompilerArgs...), "-c", "tools/native_abi/testdata/probe.c", "-o", filepath.Join(temporary, "probe.o"))
	command := exec.Command("gcc", objectArgs...)
	if output, compileErr := command.CombinedOutput(); compileErr != nil {
		result.ABIMismatches = append(result.ABIMismatches, "canonical-header compile failed: "+strings.TrimSpace(string(output)))
		return result, compileErr
	}
	layoutArgs := append(append([]string{}, commonCompilerArgs...), "-DCNA_GO_LAYOUT_ONLY", "tools/native_abi/testdata/probe.c", "-o", binary)
	if output, compileErr := exec.Command("gcc", layoutArgs...).CombinedOutput(); compileErr != nil {
		result.ABIMismatches = append(result.ABIMismatches, "canonical-header layout probe failed: "+strings.TrimSpace(string(output)))
		return result, compileErr
	}
	probeOutput, err := exec.Command(binary).Output()
	if err != nil {
		result.ABIMismatches = append(result.ABIMismatches, "compiled ABI probe did not run")
		return result, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(probeOutput)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		value, parseErr := strconv.Atoi(parts[1])
		if parseErr == nil {
			result.Measurements[parts[0]] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	result.Layouts = len(result.Measurements) - 1
	result.CGoMeasurements = len(result.Measurements) + result.PrototypeTypePositions
	if library == "" {
		return result, errors.New("-library is required")
	}
	absLibrary, err := filepath.Abs(library)
	if err != nil {
		return result, err
	}
	// Reports retain the content identity, not an ephemeral qualification path.
	// Consumers select their own absolute runtime path with CNA_NATIVE_LIBRARY.
	result.NativeLibrary = filepath.Base(absLibrary)
	data, err := os.ReadFile(absLibrary)
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256(data)
	result.NativeLibrarySHA256 = hex.EncodeToString(hash[:])
	verification, err := interop.VerifyNativeLibrary(absLibrary)
	if err != nil {
		result.ABIMismatches = append(result.ABIMismatches, err.Error())
		return result, err
	}
	result.MissingLibrarySymbols = append(result.MissingLibrarySymbols, verification.MissingSymbols...)
	if verification.ABIVersion != 0x00000700 {
		result.ABIMismatches = append(result.ABIMismatches, fmt.Sprintf("loaded ABI is 0x%08x", verification.ABIVersion))
	}
	if len(verification.BoundSymbols) != result.BoundFunctions {
		result.ABIMismatches = append(result.ABIMismatches, fmt.Sprintf("bridge bound %d symbols; manifest has %d", len(verification.BoundSymbols), result.BoundFunctions))
	}
	if len(result.ABIMismatches) == 0 && len(result.MissingHeaderSymbols) == 0 && len(result.MissingLibrarySymbols) == 0 {
		result.Status = "PASS"
		return result, nil
	}
	return result, errors.New("native ABI verification failed")
}

func writeReport(path string, result report) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
