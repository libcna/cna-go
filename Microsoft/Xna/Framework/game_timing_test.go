package framework

import (
	"errors"
	"reflect"
	"testing"
)

func newTimingGame(t *testing.T) *Game {
	t.Helper()
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

// TestTimingDefaultsAreTheConstructorsOwn pins the four values Game::.ctor
// assigns, read from its IL. They matter beyond tidiness: they are what the
// native game is created with, so a wrong default is a wrong frame rate.
func TestTimingDefaultsAreTheConstructorsOwn(t *testing.T) {
	game := newTimingGame(t)
	if got := game.TargetElapsedTime().Ticks(); got != 166667 {
		t.Fatalf("TargetElapsedTime = %d ticks, want 166667 (0x28b0b, one sixtieth of a second)", got)
	}
	if got := game.InactiveSleepTime().Ticks(); got != 200000 {
		t.Fatalf("InactiveSleepTime = %d ticks, want 200000 (20 milliseconds)", got)
	}
	if !game.IsFixedTimeStep() {
		t.Fatal("IsFixedTimeStep = false; the constructor stores true")
	}
	// isMouseVisible is NOT assigned by the constructor, so it starts false.
	if game.IsMouseVisible() {
		t.Fatal("IsMouseVisible = true; the constructor does not assign it")
	}
	if got := game.maximumElapsedTime; got != 5_000_000 {
		t.Fatalf("maximumElapsedTime = %d ticks, want 5000000 (500 milliseconds)", got)
	}
	if game.updatesSinceRunningSlowly1 != gameRunningSlowlyReset || game.updatesSinceRunningSlowly2 != gameRunningSlowlyReset {
		t.Fatalf("running-slowly counters = %d/%d, want int.MaxValue in both",
			game.updatesSinceRunningSlowly1, game.updatesSinceRunningSlowly2)
	}
}

// TestTheTwoTimeSpanSettersHaveDifferentBoundaries is the difference of one IL
// instruction, and it is observable.
//
//	set_InactiveSleepTime  op_LessThan          -> zero is ACCEPTED
//	set_TargetElapsedTime  op_LessThanOrEqual   -> zero is REJECTED
//
// The two look symmetrical and are not. Only the resource KEY is called
// InactiveSleepTimeCannotBeZero; the string it names says "greater than or
// equal to zero", which is exactly what op_LessThan admits.
func TestTheTwoTimeSpanSettersHaveDifferentBoundaries(t *testing.T) {
	game := newTimingGame(t)
	if err := game.SetInactiveSleepTime(TimeSpanFromTicks(0)); err != nil {
		t.Fatalf("InactiveSleepTime = 0 was rejected: %v; the comparison is op_LessThan", err)
	}
	if got := game.InactiveSleepTime().Ticks(); got != 0 {
		t.Fatalf("InactiveSleepTime = %d after storing zero", got)
	}
	if err := game.SetInactiveSleepTime(TimeSpanFromTicks(-1)); !errors.Is(err, errTimeSpanOutOfRange) {
		t.Fatalf("InactiveSleepTime = -1 reported %v, want an out-of-range error", err)
	}
	if got := game.InactiveSleepTime().Ticks(); got != 0 {
		t.Fatalf("a rejected InactiveSleepTime still stored: %d", got)
	}

	if err := game.SetTargetElapsedTime(TimeSpanFromTicks(0)); !errors.Is(err, errTimeSpanOutOfRange) {
		t.Fatalf("TargetElapsedTime = 0 reported %v, want an out-of-range error; the comparison is op_LessThanOrEqual", err)
	}
	if err := game.SetTargetElapsedTime(TimeSpanFromTicks(-1)); !errors.Is(err, errTimeSpanOutOfRange) {
		t.Fatalf("TargetElapsedTime = -1 reported %v", err)
	}
	if got := game.TargetElapsedTime().Ticks(); got != 166667 {
		t.Fatalf("a rejected TargetElapsedTime still stored: %d", got)
	}
	if err := game.SetTargetElapsedTime(TimeSpanFromTicks(1)); err != nil {
		t.Fatalf("TargetElapsedTime = 1 was rejected: %v", err)
	}
	if got := game.TargetElapsedTime().Ticks(); got != 1 {
		t.Fatalf("TargetElapsedTime = %d after storing 1", got)
	}
}

// TestTimingSettersStoreWithNoNativeGame is the state a consumer configures a
// Game in before Run. Every setter succeeds, every getter reports what was
// stored, and the values are what cna_game_create is then handed.
func TestTimingSettersStoreWithNoNativeGame(t *testing.T) {
	game := newTimingGame(t)
	if err := game.SetIsFixedTimeStep(false); err != nil {
		t.Fatalf("SetIsFixedTimeStep: %v", err)
	}
	if err := game.SetIsMouseVisible(true); err != nil {
		t.Fatalf("SetIsMouseVisible: %v", err)
	}
	if err := game.SetTargetElapsedTime(TimeSpanFromTicks(333333)); err != nil {
		t.Fatalf("SetTargetElapsedTime: %v", err)
	}
	if err := game.SetInactiveSleepTime(TimeSpanFromTicks(50_000)); err != nil {
		t.Fatalf("SetInactiveSleepTime: %v", err)
	}
	if game.IsFixedTimeStep() || !game.IsMouseVisible() {
		t.Fatalf("flags = %t/%t, want false/true", game.IsFixedTimeStep(), game.IsMouseVisible())
	}
	// And that is exactly what the native game would be created with.
	timing := (gameRuntimeCallbacks{game: game}).TimingConfiguration()
	if timing.TargetElapsedTicks != 333333 || timing.InactiveSleepTicks != 50_000 {
		t.Fatalf("creation timing = %+v", timing)
	}
	if timing.IsFixedTimeStep || !timing.IsMouseVisible {
		t.Fatalf("creation flags = %+v", timing)
	}
}

// TestSuppressDrawAndResetElapsedTimeAssignTheirOwnFields holds both reference
// bodies. SuppressDraw is one `stfld`; ResetElapsedTime is four assignments and
// nothing else.
func TestSuppressDrawAndResetElapsedTimeAssignTheirOwnFields(t *testing.T) {
	game := newTimingGame(t)
	if game.suppressDraw {
		t.Fatal("suppressDraw starts true")
	}
	if err := game.SuppressDraw(); err != nil {
		t.Fatalf("SuppressDraw: %v", err)
	}
	if !game.suppressDraw {
		t.Fatal("SuppressDraw did not raise the flag")
	}
	// There is no way to lower it from the public surface, and the reference
	// has none either: only the loop clears it.
	game.drawRunningSlowly = true
	game.updatesSinceRunningSlowly1 = 3
	game.updatesSinceRunningSlowly2 = 7
	game.forceElapsedTimeToZero = false
	if err := game.ResetElapsedTime(); err != nil {
		t.Fatalf("ResetElapsedTime: %v", err)
	}
	if !game.forceElapsedTimeToZero {
		t.Fatal("ResetElapsedTime did not raise forceElapsedTimeToZero")
	}
	if game.drawRunningSlowly {
		t.Fatal("ResetElapsedTime did not lower drawRunningSlowly")
	}
	if game.updatesSinceRunningSlowly1 != gameRunningSlowlyReset || game.updatesSinceRunningSlowly2 != gameRunningSlowlyReset {
		t.Fatalf("running-slowly counters = %d/%d, want int.MaxValue in both",
			game.updatesSinceRunningSlowly1, game.updatesSinceRunningSlowly2)
	}
	// It does not touch suppressDraw, and SuppressDraw does not touch these.
	if !game.suppressDraw {
		t.Fatal("ResetElapsedTime cleared suppressDraw; its body has four assignments and that is not one of them")
	}
}

// TestTimingSurfaceIsTheProjectedShape holds the ten members' signatures, and
// in particular that the four getters are INFALLIBLE. Each is one `ldfld` in
// the reference -- no validation, no host, no device, no throw site -- so a
// synthetic error would be an invented failure mode.
func TestTimingSurfaceIsTheProjectedShape(t *testing.T) {
	game := reflect.TypeOf((*Game)(nil))
	for name, signature := range map[string]string{
		"InactiveSleepTime":    "func(*framework.Game) framework.TimeSpan",
		"SetInactiveSleepTime": "func(*framework.Game, framework.TimeSpan) error",
		"TargetElapsedTime":    "func(*framework.Game) framework.TimeSpan",
		"SetTargetElapsedTime": "func(*framework.Game, framework.TimeSpan) error",
		"IsFixedTimeStep":      "func(*framework.Game) bool",
		"SetIsFixedTimeStep":   "func(*framework.Game, bool) error",
		"IsMouseVisible":       "func(*framework.Game) bool",
		"SetIsMouseVisible":    "func(*framework.Game, bool) error",
		"SuppressDraw":         "func(*framework.Game) error",
		"ResetElapsedTime":     "func(*framework.Game) error",
	} {
		method, ok := game.MethodByName(name)
		if !ok {
			t.Fatalf("Game has no method %q", name)
		}
		if got := method.Type.String(); got != signature {
			t.Fatalf("Game.%s has signature %s, want %s", name, got, signature)
		}
	}
}

// TestTimingMembersRefuseAnUnconstructedGame covers the family's Go-only
// failure. The two TimeSpan setters validate BEFORE the guard, exactly as the
// reference validates before touching any field.
func TestTimingMembersRefuseAnUnconstructedGame(t *testing.T) {
	zero := &Game{}
	for name, call := range map[string]func() error{
		"SetIsFixedTimeStep": func() error { return zero.SetIsFixedTimeStep(true) },
		"SetIsMouseVisible":  func() error { return zero.SetIsMouseVisible(true) },
		"SuppressDraw":       zero.SuppressDraw,
		"ResetElapsedTime":   zero.ResetElapsedTime,
		"SetInactiveSleepTime": func() error {
			return zero.SetInactiveSleepTime(TimeSpanFromTicks(1))
		},
		"SetTargetElapsedTime": func() error {
			return zero.SetTargetElapsedTime(TimeSpanFromTicks(1))
		},
	} {
		if err := call(); !errors.Is(err, errGameNotConstructed) {
			t.Fatalf("%s on an unconstructed Game = %v", name, err)
		}
	}
	// The argument check comes first, so a bad value is reported as a bad
	// value even on a Game whose constructor never ran.
	if err := zero.SetTargetElapsedTime(TimeSpanFromTicks(0)); !errors.Is(err, errTimeSpanOutOfRange) {
		t.Fatalf("SetTargetElapsedTime(0) on an unconstructed Game = %v, want the range error", err)
	}
	if err := zero.SetInactiveSleepTime(TimeSpanFromTicks(-1)); !errors.Is(err, errTimeSpanOutOfRange) {
		t.Fatalf("SetInactiveSleepTime(-1) on an unconstructed Game = %v, want the range error", err)
	}
	// The four getters are field reads and work on anything.
	if zero.IsFixedTimeStep() || zero.IsMouseVisible() {
		t.Fatal("an unconstructed Game reports a raised timing flag")
	}
	if zero.TargetElapsedTime().Ticks() != 0 || zero.InactiveSleepTime().Ticks() != 0 {
		t.Fatal("an unconstructed Game reports a non-zero timing value")
	}
}
