package cna

import "errors"

// ErrNativeUnavailable reports that the upstream CNA C ABI has not shipped.
var ErrNativeUnavailable = errors.New("cna: native C ABI is not available yet")

// GameTime describes timing for one update or draw callback.
type GameTime struct {
	TotalSeconds   float64
	ElapsedSeconds float64
	RunningSlowly  bool
}

// Game receives lifecycle callbacks from CNA. Go uses an interface instead of
// emulating the class inheritance used by XNA.
type Game interface {
	Initialize(*GameContext) error
	LoadContent(*GameContext) error
	Update(*GameContext, GameTime) error
	Draw(*GameContext, GameTime) error
	UnloadContent(*GameContext) error
}

// GameAdapter supplies no-op lifecycle methods for games that only need to
// implement a subset of Game.
type GameAdapter struct{}

func (GameAdapter) Initialize(*GameContext) error              { return nil }
func (GameAdapter) LoadContent(*GameContext) error             { return nil }
func (GameAdapter) Update(*GameContext, GameTime) error        { return nil }
func (GameAdapter) Draw(*GameContext, GameTime) error          { return nil }
func (GameAdapter) UnloadContent(*GameContext) error           { return nil }

// GameContext will expose graphics, content, input, and exit control once the
// C ABI is available. It intentionally has no native handle in the scaffold.
type GameContext struct{}

// Run will hand a Game to CNA's native loop. It currently returns an explicit
// availability error instead of pretending that a native runtime exists.
func Run(game Game) error {
	if game == nil {
		return errors.New("cna: game must not be nil")
	}
	return ErrNativeUnavailable
}
