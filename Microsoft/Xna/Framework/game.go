package framework

import (
	"errors"

	"github.com/openeggbert/cna-go/internal/interop"
)

// GameTime describes timing for one update or draw callback.
type GameTime struct {
	TotalSeconds    float64
	ElapsedSeconds  float64
	IsRunningSlowly bool
}

// Game receives lifecycle callbacks from CNA.
type Game interface {
	Initialize() error
	LoadContent() error
	Update(GameTime) error
	Draw(GameTime) error
	UnloadContent() error
}

// Run hands an XNA-compatible game to CNA's native loop.
func Run(game Game) error {
	if game == nil {
		return errors.New("game must not be nil")
	}
	return interop.ErrNativeUnavailable
}
