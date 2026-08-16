package framework

import "errors"

// ErrNativeUnavailable reports that the upstream CNA C ABI has not shipped.
var ErrNativeUnavailable = errors.New("CNA native C ABI is not available yet")

// GameTime describes timing for one update or draw callback.
type GameTime struct {
	TotalSeconds   float64
	ElapsedSeconds float64
	IsRunningSlowly bool
}

// Game receives lifecycle callbacks from CNA.
type Game interface {
	Initialize(*GameContext) error
	LoadContent(*GameContext) error
	Update(*GameContext, GameTime) error
	Draw(*GameContext, GameTime) error
	UnloadContent(*GameContext) error
}

// GameContext will expose graphics, content, input, and exit control.
type GameContext struct{}

// Run hands a Game to CNA's native loop.
func Run(game Game) error {
	if game == nil {
		return errors.New("game must not be nil")
	}
	return ErrNativeUnavailable
}
