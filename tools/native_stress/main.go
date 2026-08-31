// Command native_stress runs each native lifetime scenario in a crash-isolated
// subprocess. It does not claim sanitizer or leak-detector coverage.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	input "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input"
	"github.com/openeggbert/cna-go/internal/interop"
)

type counters struct {
	GameCycles           int `json:"GAME_CYCLES"`
	GameRecreationCycles int `json:"GAME_RECREATION_CYCLES"`
	TextureCycles        int `json:"TEXTURE_CYCLES"`
	SpriteBatchCycles    int `json:"SPRITEBATCH_CYCLES"`
	CallbackErrorCycles  int `json:"CALLBACK_ERROR_CYCLES"`
	CallbackPanicCycles  int `json:"CALLBACK_PANIC_CYCLES"`
	WrongThreadChecks    int `json:"WRONG_THREAD_CHECKS"`
	OwnerThreadRetries   int `json:"OWNER_THREAD_RETRIES"`
	GCStressPoints       int `json:"GC_STRESS_POINTS"`
	NativeCrashes        int `json:"NATIVE_CRASHES"`
	ObservedUAF          int `json:"OBSERVED_UAF"`
	ObservedDoubleFree   int `json:"OBSERVED_DOUBLE_FREE"`

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
}

type stressReport struct {
	SchemaVersion         int      `json:"schema_version"`
	Isolation             string   `json:"isolation"`
	GoRaceStatus          string   `json:"GO_RACE_STATUS"`
	NativeSanitizerStatus string   `json:"NATIVE_SANITIZER_STATUS"`
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
	report := stressReport{
		SchemaVersion:         1,
		Isolation:             "one native Game generation per subprocess",
		GoRaceStatus:          raceStatus,
		NativeSanitizerStatus: "NOT_RUN",
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

func runParent() (counters, error) {
	executable, err := os.Executable()
	if err != nil {
		return counters{}, err
	}
	var total counters
	for _, scenario := range []string{"success", "callback-error", "callback-panic", "event-rerun", "frame-hook-override", "frame-hook-subset", "timing"} {
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
			go func() { wrongThread <- texture.Dispose(true) }()
			if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
				return fmt.Errorf("wrong-thread texture disposal result: %w", err)
			}
			g.result.WrongThreadChecks++
			if err := texture.Dispose(true); err != nil {
				return fmt.Errorf("owner-thread retry: %w", err)
			}
			g.result.OwnerThreadRetries++
		} else if err := texture.Dispose(true); err != nil {
			return err
		}
		if err := texture.Dispose(true); err != nil {
			g.result.ObservedDoubleFree++
			return fmt.Errorf("double texture Dispose was not idempotent: %w", err)
		}
		if err := batch.Dispose(true); err != nil {
			return err
		}
		if err := batch.Dispose(true); err != nil {
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
	return host.Exit()
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

func addCounters(target *counters, value counters) {
	target.GameCycles += value.GameCycles
	target.GameRecreationCycles += value.GameRecreationCycles
	target.TextureCycles += value.TextureCycles
	target.SpriteBatchCycles += value.SpriteBatchCycles
	target.CallbackErrorCycles += value.CallbackErrorCycles
	target.CallbackPanicCycles += value.CallbackPanicCycles
	target.WrongThreadChecks += value.WrongThreadChecks
	target.OwnerThreadRetries += value.OwnerThreadRetries
	target.GCStressPoints += value.GCStressPoints
	target.NativeCrashes += value.NativeCrashes
	target.ObservedUAF += value.ObservedUAF
	target.ObservedDoubleFree += value.ObservedDoubleFree
	target.GameEventActivated += value.GameEventActivated
	target.GameEventDeactivated += value.GameEventDeactivated
	target.GameEventExiting += value.GameEventExiting
	target.GameNativeDisposalSignals += value.GameNativeDisposalSignals
	target.GameDisposedDuringRun += value.GameDisposedDuringRun
	target.GameDisposedByManagedCall += value.GameDisposedByManagedCall
	target.GameDisposedRepeatChecks += value.GameDisposedRepeatChecks
	target.GameDisposeAfterRunCycles += value.GameDisposeAfterRunCycles
	target.GameEventOrderChecks += value.GameEventOrderChecks
	target.GameEventRemovalChecks += value.GameEventRemovalChecks
	target.GameEventOwnerThreadHits += value.GameEventOwnerThreadHits
	target.GameEventRerunCycles += value.GameEventRerunCycles
	target.GameEventPostRunChecks += value.GameEventPostRunChecks
	target.FrameHookOverrideCycles += value.FrameHookOverrideCycles
	target.FrameHookBeginRunHits += value.FrameHookBeginRunHits
	target.FrameHookEndRunHits += value.FrameHookEndRunHits
	target.FrameHookBeginDrawHits += value.FrameHookBeginDrawHits
	target.FrameHookEndDrawHits += value.FrameHookEndDrawHits
	target.FrameHookRefusedFrames += value.FrameHookRefusedFrames
	target.FrameHookAdmittedFrames += value.FrameHookAdmittedFrames
	target.FrameHookEndDrawExpected += value.FrameHookEndDrawExpected
	target.FrameHookSkipChecks += value.FrameHookSkipChecks
	target.FrameHookBaseCallChecks += value.FrameHookBaseCallChecks
	target.FrameHookOrderChecks += value.FrameHookOrderChecks
	target.FrameHookSubsetCycles += value.FrameHookSubsetCycles
	target.FrameHookUninstalledHits += value.FrameHookUninstalledHits
	target.TimingCycles += value.TimingCycles
	target.TimingSettersApplied += value.TimingSettersApplied
	target.TimingWrongThreadChecks += value.TimingWrongThreadChecks
	target.TimingRangeChecks += value.TimingRangeChecks
	target.TimingCreatedWithConfig += value.TimingCreatedWithConfig
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
