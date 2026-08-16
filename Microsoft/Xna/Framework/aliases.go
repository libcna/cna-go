package framework

import cnaframework "github.com/openeggbert/cna-go/CNA/Framework"

// Vector2 preserves the XNA value type name at the compatibility import path.
type Vector2 = cnaframework.Vector2

// Color preserves the XNA value type name at the compatibility import path.
type Color = cnaframework.Color

// GameTime preserves the XNA callback value name.
type GameTime = cnaframework.GameTime

// Game preserves the XNA lifecycle concept using a Go interface.
type Game = cnaframework.Game

// GameContext is the CNA-backed lifecycle context.
type GameContext = cnaframework.GameContext

var (
	// CornflowerBlue is the traditional XNA clear color.
	CornflowerBlue = cnaframework.CornflowerBlue
	// White is opaque white.
	White = cnaframework.White
)

// Run hands an XNA-compatible game to CNA's native loop.
func Run(game Game) error {
	return cnaframework.Run(game)
}
