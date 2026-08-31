package framework

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// This file projects Game's timing and presentation state:
//
//	InactiveSleepTime   TimeSpan   get/set
//	TargetElapsedTime   TimeSpan   get/set
//	IsFixedTimeStep     bool       get/set
//	IsMouseVisible      bool       get/set
//	SuppressDraw()
//	ResetElapsedTime()
//
// # Getters are field reads, setters reach the loop
//
// In the reference every one of these getters is a single `ldfld`, and the
// managed loop reads the SAME fields every frame. CNA-Go's loop is the native
// one, so the split falls out of that difference and is not a choice:
//
//   - the getter stays a field read of Game's own managed state, exactly as the
//     reference's is. It allocates nothing, validates nothing, reaches nothing,
//     and works before Run, during it and after it;
//   - the setter validates as the reference does, stores as the reference does,
//     and then pushes the value to the native loop -- because that loop is what
//     the reference's own loop would have read, and a stored value the loop
//     never sees would be a setter that appears to work and does not.
//
// # Nothing here was added to CNA
//
// All six canonical operations were already exported by the pinned artifact and
// had simply never been reached from Go:
//
//	cna_game_set_is_mouse_visible          cna_game_reset_elapsed_time
//	cna_game_set_is_fixed_time_step        cna_game_suppress_draw
//	cna_game_set_target_elapsed_time_ticks
//	cna_game_set_inactive_sleep_time_ticks
//
// The corresponding cna_game_get_* functions are deliberately NOT bound. The
// reference reads its own field, so binding the native getter would introduce a
// second source of truth that could disagree with the first.
//
// # Before Run, during Run, after Run
//
// With no live native game a setter stores and reports success, and the stored
// value is carried into cna_game_create -- which is why the create info no
// longer passes literals. That is the reference's behaviour too: a Game
// configured before Run runs with what it was configured with.
//
// During a run the setter reaches the loop on the owner thread. From another
// goroutine CNA answers CNA_RESULT_THREAD and the setter reports it, which is
// more useful than reproducing a value the loop will not honour.

// The XNA Game constructor's timing defaults, read from its IL:
//
//	maximumElapsedTime = TimeSpan.FromMilliseconds(500)
//	isFixedTimeStep    = true
//	targetElapsedTime  = TimeSpan.FromTicks(0x28b0b)      // 166667, 1/60 s
//	inactiveSleepTime  = TimeSpan.FromMilliseconds(20)    // 200000 ticks
//
// isMouseVisible is not assigned by the constructor and therefore defaults to
// false, which is Go's zero value for the field as well.
const (
	gameDefaultTargetElapsedTicks int64 = 0x28b0b
	gameDefaultInactiveSleepTicks int64 = 20 * 10_000
	gameMaximumElapsedTicks       int64 = 500 * 10_000
	// gameRunningSlowlyReset is int.MaxValue, which ResetElapsedTime and the
	// constructor both store into the two running-slowly counters.
	gameRunningSlowlyReset int32 = 0x7fffffff
)

// errTimeSpanOutOfRange projects System.ArgumentOutOfRangeException, which both
// TimeSpan setters throw. It is unexported because the XNA public contract
// declares no error type here, the same way the service container's are.
var errTimeSpanOutOfRange = errors.New("argument is out of range")

// The exact Resources strings the two throw sites load.
const (
	inactiveSleepTimeCannotBeZero = "The inactive sleep time cannot be zero."
	targetElapsedCannotBeZero     = "The target elapsed time cannot be zero or less than zero."
)

func timeSpanOutOfRangeError(parameter, message string) error {
	return fmt.Errorf("%w: %s: %s", errTimeSpanOutOfRange, parameter, message)
}

// InactiveSleepTime is Game::get_InactiveSleepTime, one `ldfld` over the field
// the constructor sets to 20 milliseconds.
func (g *Game) InactiveSleepTime() TimeSpan {
	return TimeSpanFromTicks(g.inactiveSleepTime)
}

// SetInactiveSleepTime is Game::set_InactiveSleepTime:
//
//	if (value < TimeSpan.Zero)
//	    throw new ArgumentOutOfRangeException("value",
//	        Resources.InactiveSleepTimeCannotBeZero);
//	this.inactiveSleepTime = value;
//
// Note the comparison: it is `op_LessThan`, so ZERO IS ACCEPTED even though the
// message the reference loads says it cannot be. The message is the reference's
// and is reproduced verbatim; the boundary is the IL's, and the IL admits zero.
func (g *Game) SetInactiveSleepTime(value TimeSpan) error {
	if value.Ticks() < 0 {
		return timeSpanOutOfRangeError("value", inactiveSleepTimeCannotBeZero)
	}
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.inactiveSleepTime = value.Ticks()
	_, err := g.runtime.SetInactiveSleepTimeTicks(value.Ticks())
	return err
}

// TargetElapsedTime is Game::get_TargetElapsedTime, one `ldfld` over the field
// the constructor sets to 166,667 ticks -- one sixtieth of a second.
func (g *Game) TargetElapsedTime() TimeSpan {
	return TimeSpanFromTicks(g.targetElapsedTime)
}

// SetTargetElapsedTime is Game::set_TargetElapsedTime:
//
//	if (value <= TimeSpan.Zero)
//	    throw new ArgumentOutOfRangeException("value",
//	        Resources.TargetElaspedCannotBeZero);
//	this.targetElapsedTime = value;
//
// The comparison here is `op_LessThanOrEqual`, so unlike its neighbour this one
// really does reject zero. The two setters look symmetrical and are not, and
// the difference is one IL instruction.
func (g *Game) SetTargetElapsedTime(value TimeSpan) error {
	if value.Ticks() <= 0 {
		return timeSpanOutOfRangeError("value", targetElapsedCannotBeZero)
	}
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.targetElapsedTime = value.Ticks()
	_, err := g.runtime.SetTargetElapsedTimeTicks(value.Ticks())
	return err
}

// IsFixedTimeStep is Game::get_IsFixedTimeStep, one `ldfld` over the field the
// constructor sets to true.
func (g *Game) IsFixedTimeStep() bool { return g.isFixedTimeStep }

// SetIsFixedTimeStep is Game::set_IsFixedTimeStep, whose whole reference body is
// one `stfld` with no validation and no suppression -- assigning the value it
// already has stores it again and does nothing else.
//
// It is fallible here for a reason the reference does not have: the value has to
// reach the native loop, and that call can be refused.
func (g *Game) SetIsFixedTimeStep(value bool) error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.isFixedTimeStep = value
	_, err := g.runtime.SetIsFixedTimeStep(value)
	return err
}

// IsMouseVisible is Game::get_IsMouseVisible, one `ldfld`. The constructor does
// not assign it, so it starts false.
func (g *Game) IsMouseVisible() bool { return g.isMouseVisible }

// SetIsMouseVisible is Game::set_IsMouseVisible:
//
//	this.isMouseVisible = value;
//	if (this.Window != null) this.Window.IsMouseVisible = value;
//
// Store first, then propagate to the window -- and the propagation is guarded,
// because get_Window returns null when the Game has no host.
//
// CNA-Go has no GameWindow and no GameHost: the native runtime owns the window,
// and CNA publishes the same effect as cna_game_set_is_mouse_visible. So the
// guarded branch projects exactly: with a live native game the value reaches the
// window, and with none there is no window to reach, which is the state the
// reference itself is in before Run.
func (g *Game) SetIsMouseVisible(value bool) error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.isMouseVisible = value
	_, err := g.runtime.SetIsMouseVisible(value)
	return err
}

// SuppressDraw is Game::SuppressDraw, whose whole reference body is
//
//	this.suppressDraw = true;
//
// eight bytes of IL and no validation. The field's only readers are Tick and
// the run loop, which decide whether the next frame draws -- and in CNA-Go that
// loop is the native one, so the flag is pushed to it. Setting a flag the loop
// never reads would be a member that appears to work and does not.
//
// The managed field is kept because it is the reference's own state, and it is
// what makes the projection's effect observable from Go.
func (g *Game) SuppressDraw() error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.suppressDraw = true
	_, err := g.runtime.SuppressDraw()
	return err
}

// ResetElapsedTime is Game::ResetElapsedTime:
//
//	this.forceElapsedTimeToZero = true;
//	this.drawRunningSlowly = false;
//	this.updatesSinceRunningSlowly1 = int.MaxValue;
//	this.updatesSinceRunningSlowly2 = int.MaxValue;
//
// Four managed assignments and nothing else. Every one of them is state the
// reference's own timing loop reads, and CNA's cna_game_reset_elapsed_time is
// that loop's counterpart -- its documentation says the same thing the member's
// does, that the next frame must not be reported as running slowly after a long
// blocking operation.
//
// All four fields are assigned here as well as pushed, because they are Game's
// own state and omitting them would leave the projected object half-reset.
func (g *Game) ResetElapsedTime() error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	g.forceElapsedTimeToZero = true
	g.drawRunningSlowly = false
	g.updatesSinceRunningSlowly1 = gameRunningSlowlyReset
	g.updatesSinceRunningSlowly2 = gameRunningSlowlyReset
	_, err := g.runtime.ResetElapsedTime()
	return err
}

// timingConfiguration is what the native game is created with. It is read once,
// on the owner thread, immediately before cna_game_create.
func (c gameRuntimeCallbacks) TimingConfiguration() interop.TimingConfiguration {
	return interop.TimingConfiguration{
		TargetElapsedTicks: c.game.targetElapsedTime,
		InactiveSleepTicks: c.game.inactiveSleepTime,
		IsFixedTimeStep:    c.game.isFixedTimeStep,
		IsMouseVisible:     c.game.isMouseVisible,
	}
}
