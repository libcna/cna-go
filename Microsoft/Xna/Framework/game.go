package framework

import (
	"errors"

	"github.com/openeggbert/cna-go/internal/interop"
)

// Game is the concrete Go facade for the XNA Game class. Protected virtual
// overrides are supplied separately through GameCallbacks because Go has no
// inheritance or virtual methods.
type Game struct {
	callbacks GameCallbacks
	runtime   *interop.Runtime
}

// GameCallbacks is the measured Go language adapter for XNA Game lifecycle
// overrides. The native CNA loop invokes these methods on the Game owner OS
// thread.
type GameCallbacks interface {
	Initialize(*Game) error
	LoadContent(*Game) error
	Update(*Game, GameTime) error
	Draw(*Game, GameTime) error
	UnloadContent(*Game) error
}

// NewGame associates lifecycle overrides with one not-yet-running Game host.
func NewGame(callbacks GameCallbacks) (*Game, error) {
	if callbacks == nil {
		return nil, errors.New("Game callbacks must not be nil")
	}
	game := &Game{callbacks: callbacks}
	game.runtime = interop.NewRuntime(gameRuntimeCallbacks{game: game})
	interop.RegisterOwner(game, game.runtime, nil)
	return game, nil
}

// Run creates and runs the admitted CNA ABI 0.7 Game on a locked OS thread.
func (g *Game) Run() error {
	if g == nil || g.runtime == nil {
		return errors.New("Game is nil or uninitialized")
	}
	return g.runtime.Run()
}

// Exit asks the native CNA loop to stop at its next safe point.
func (g *Game) Exit() error {
	if g == nil || g.runtime == nil {
		return errors.New("Game is nil or uninitialized")
	}
	return g.runtime.Exit()
}

type gameRuntimeCallbacks struct {
	game *Game
}

func (c gameRuntimeCallbacks) Initialize() error {
	return c.game.callbacks.Initialize(c.game)
}

func (c gameRuntimeCallbacks) LoadContent() error {
	return c.game.callbacks.LoadContent(c.game)
}

func (c gameRuntimeCallbacks) Update(value interop.FrameTime) error {
	return c.game.callbacks.Update(c.game, gameTimeFromInterop(value))
}

func (c gameRuntimeCallbacks) Draw(value interop.FrameTime) error {
	return c.game.callbacks.Draw(c.game, gameTimeFromInterop(value))
}

func (c gameRuntimeCallbacks) UnloadContent() error {
	return c.game.callbacks.UnloadContent(c.game)
}

func gameTimeFromInterop(value interop.FrameTime) GameTime {
	return NewGameTimeByTimeSpanAndTimeSpanAndBoolean(
		TimeSpanFromTicks(value.TotalTicks),
		TimeSpanFromTicks(value.ElapsedTicks),
		value.IsRunningSlowly,
	)
}

// GraphicsDeviceManager owns the Game's canonical native graphics manager.
type GraphicsDeviceManager struct {
	runtime  *interop.Runtime
	resource *interop.Resource
}

// NewGraphicsDeviceManager maps the XNA constructor and must be called from a
// lifecycle callback, normally Initialize.
func NewGraphicsDeviceManager(game *Game) (*GraphicsDeviceManager, error) {
	if game == nil || game.runtime == nil {
		return nil, errors.New("Game is nil or uninitialized")
	}
	resource, err := game.runtime.CreateGraphicsDeviceManager()
	if err != nil {
		return nil, err
	}
	manager := &GraphicsDeviceManager{runtime: game.runtime, resource: resource}
	interop.RegisterOwner(manager, manager.runtime, manager.resource)
	return manager, nil
}

// Dispose releases the owned manager. The native handle is retained if CNA
// refuses destruction, allowing an owner-thread retry.
func (m *GraphicsDeviceManager) Dispose(disposing bool) error {
	_ = disposing
	if m == nil || m.resource == nil {
		return nil
	}
	if err := m.resource.Dispose(); err != nil {
		return err
	}
	interop.UnregisterOwner(m)
	return nil
}
