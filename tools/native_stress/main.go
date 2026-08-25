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

	GameEventActivated       int `json:"GAME_EVENT_ACTIVATED_DELIVERIES"`
	GameEventDeactivated     int `json:"GAME_EVENT_DEACTIVATED_DELIVERIES"`
	GameEventExiting         int `json:"GAME_EVENT_EXITING_DELIVERIES"`
	GameEventDisposed        int `json:"GAME_EVENT_DISPOSED_DELIVERIES"`
	GameEventOrderChecks     int `json:"GAME_EVENT_ORDER_CHECKS"`
	GameEventRemovalChecks   int `json:"GAME_EVENT_REMOVAL_CHECKS"`
	GameEventOwnerThreadHits int `json:"GAME_EVENT_OWNER_THREAD_CHECKS"`
	GameEventRerunCycles     int `json:"GAME_EVENT_RERUN_CYCLES"`
	GameEventPostRunChecks   int `json:"GAME_EVENT_POST_RUN_CHECKS"`
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
	for _, scenario := range []string{"success", "callback-error", "callback-panic", "event-rerun"} {
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
	if total.GameEventActivated < 20 || total.GameEventExiting < 20 || total.GameEventDisposed < 20 {
		return total, errors.New("native game-event delivery minimum was not met")
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
	if _, err := host.AddDisposedHandler(record("Disposed")); err != nil {
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
		case "Disposed":
			g.result.GameEventDisposed++
		}
	}
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
	if g.result.GameEventDisposed != 1 {
		return fmt.Errorf("Disposed delivered %d times, want exactly 1", g.result.GameEventDisposed)
	}
	exiting, disposed := -1, -1
	for i, name := range g.eventOrder {
		switch name {
		case "Exiting":
			exiting = i
		case "Disposed":
			disposed = i
		}
	}
	if exiting < 0 || disposed < 0 || exiting >= disposed {
		return fmt.Errorf("delivery order %v does not place Exiting before Disposed", g.eventOrder)
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
	if first.GameEventActivated != 1 || first.GameEventExiting != 1 || first.GameEventDisposed != 1 {
		return fmt.Errorf("first run delivered %v", game.eventOrder)
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
	activated, exiting, disposed := 0, 0, 0
	for _, name := range second {
		switch name {
		case "Activated":
			activated++
		case "Exiting":
			exiting++
		case "Disposed":
			disposed++
		}
	}
	if exiting != 1 || disposed != 1 {
		return fmt.Errorf("second run delivered %v; Exiting and Disposed must each arrive exactly once", second)
	}
	// The edge-trigger guard is the whole point: the managed half never saw a
	// deactivation, so the second run's activation signal raises nothing.
	if activated != 0 {
		return fmt.Errorf("second run raised Activated %d times; isActive was already true, so the guard must suppress it", activated)
	}
	if len(game.eventGoroutines) != 1 || !game.eventGoroutines[game.ownerGoroutine] {
		return fmt.Errorf("second run delivered on %d goroutines", len(game.eventGoroutines))
	}
	game.result.GameEventRerunCycles++
	game.result.GameEventActivated = first.GameEventActivated + activated
	game.result.GameEventExiting = first.GameEventExiting + exiting
	game.result.GameEventDisposed = first.GameEventDisposed + disposed
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

func (g *stressGame) Initialize(host *framework.Game) error {
	g.ownerGoroutine = currentGoroutineLabel()
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
	target.GameEventDisposed += value.GameEventDisposed
	target.GameEventOrderChecks += value.GameEventOrderChecks
	target.GameEventRemovalChecks += value.GameEventRemovalChecks
	target.GameEventOwnerThreadHits += value.GameEventOwnerThreadHits
	target.GameEventRerunCycles += value.GameEventRerunCycles
	target.GameEventPostRunChecks += value.GameEventPostRunChecks
}
