package framework

import (
	"errors"
	"sync"

	"github.com/openeggbert/cna-go/internal/interop"
)

// Game is the concrete Go facade for the XNA Game class. Protected virtual
// overrides are supplied separately through GameCallbacks because Go has no
// inheritance or virtual methods.
//
// # Game is a hybrid host
//
// The division of responsibility is deliberate and is read from the reference
// per member, not assumed per type:
//
//   - The native CNA runtime owns the real host, the frame loop, the window,
//     the device and the platform. That is GameHost's and WindowsGameHost's
//     role in Microsoft.Xna.Framework.Game.dll, and CNA-Go does not reimplement
//     it in Go.
//   - Go owns the managed CLR state. Game::get_Components and
//     Game::get_Services are one `ldfld` each over fields the Game constructor
//     assigns, and the component engine that maintains them is ordinary managed
//     list work. Routing that through a C ABI would invent native ownership and
//     a native failure mode the reference does not have, so it stays in Go.
//   - GameCallbacks stays the language adapter for the protected virtual
//     overrides, and base behavior is reached explicitly through the
//     GameBase... functions rather than being run automatically.
//
// # The managed state below
//
// The five unexported lists are Game's own private fields in the reference and
// are not exposed by any projected member. See game_component_engine.go for the
// exact derivation of how they are maintained.
type Game struct {
	callbacks GameCallbacks
	runtime   *interop.Runtime

	// gameComponents and gameServices are the two managed CLR objects
	// Game::get_Components and Game::get_Services hand out. Each is created
	// exactly once, at the reference's construction point, and the getter is
	// one field read, so the identity a caller observes never changes.
	gameComponents *GameComponentCollection
	gameServices   *GameServiceContainer

	// The five private derived lists. updateableComponents and
	// drawableComponents are kept in order incrementally; the two `currently`
	// lists are the per-frame snapshots base Update and base Draw copy into;
	// notYetInitialized is the pending-initialization queue.
	updateableComponents        []updateableEntry
	currentlyUpdatingComponents []IUpdateable
	drawableComponents          []drawableEntry
	currentlyDrawingComponents  []IDrawable
	notYetInitialized           []IGameComponent

	// doneFirstUpdate is Game::doneFirstUpdate, which base Update assigns.
	// Its readers in the reference -- Paint, Tick and DrawFrame -- are not
	// projected yet, so nothing observes it; it is kept because base Update
	// genuinely assigns it and omitting the assignment would make the
	// projected base body incomplete.
	doneFirstUpdate bool

	// inRun is Game::inRun. The reference sets it true in RunGame after
	// Initialize() returns and back to false in RunGame's finally; CNA-Go sets
	// it at the equivalent boundary of the native run sequence, because the
	// native host plays GameHost's part. It decides whether a newly added
	// component is initialized immediately or queued.
	inRun bool

	// The four CLR events Game declares, each a private registration list
	// exactly as the reference keeps a private multicast delegate field. See
	// game_events.go for the raise paths and the native bridge.
	activated   EventSource[*EventArgs]
	deactivated EventSource[*EventArgs]
	exiting     EventSource[*EventArgs]
	disposed    EventSource[*EventArgs]

	// The four optional frame-boundary overrides, captured once from the
	// callback object in NewGame. Each is nil unless that object declares the
	// corresponding exported method; a nil one means the matching native hook
	// is never installed, so the native frame position keeps its own
	// behaviour. See game_frame_hook_overrides.go.
	beginRunOverride  gameBeginRunOverride
	endRunOverride    gameEndRunOverride
	beginDrawOverride gameBeginDrawOverride
	endDrawOverride   gameEndDrawOverride

	// disposeLock projects the `lock (this)` that wraps the whole body of
	// Dispose(bool). See DisposeByBoolean in game_disposal.go for the exact
	// concurrency projection and its one deliberate divergence.
	disposeLock sync.Mutex

	// isActive is Game::isActive, the private bool HostActivated and
	// HostDeactivated maintain. It is NOT Game::IsActive: that getter also
	// consults GamerServices' Guide and stays a missing member. This field
	// exists only because it is what makes the two activation events
	// edge-triggered, and the reference leaves it false at construction.
	isActive bool
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
//
// The managed construction below reproduces the reference constructor's order
// for the part CNA-Go owns. Microsoft.Xna.Framework.Game::.ctor assigns
// gameServices from a field initializer -- so it exists before anything else
// runs -- and then, after the base constructor, allocates gameComponents and
// immediately subscribes Game's own two handlers to it:
//
//	gameServices   = new GameServiceContainer();          // field initializer
//	...
//	gameComponents = new GameComponentCollection();
//	gameComponents.ComponentAdded   += GameComponentAdded;
//	gameComponents.ComponentRemoved += GameComponentRemoved;
//
// The subscription order is observable and is preserved: Game's handlers are
// registered first, so a consumer's later handler always runs after the engine
// has already tracked or untracked the component. A consumer therefore observes
// a consistent Game during their own ComponentAdded, exactly as in CLR, where
// the multicast invocation list runs in subscription order.
func NewGame(callbacks GameCallbacks) (*Game, error) {
	if callbacks == nil {
		return nil, errors.New("Game callbacks must not be nil")
	}
	game := &Game{callbacks: callbacks}
	// The optional frame-hook capabilities are discovered here, at the same
	// boundary where the callback object becomes associated with the Game, and
	// never again. A Go object's method set is fixed for its lifetime, so a
	// later re-check could not produce a different answer, and there is no
	// registration operation to change it.
	game.captureFrameHookOverrides(callbacks)
	game.gameServices = NewGameServiceContainer()
	game.gameComponents = NewGameComponentCollection()
	// Neither accessor can fail: EventSource.Add reports no failure of its own
	// and both handlers are non-nil. The results are discarded because Game
	// never unsubscribes from its own collection -- the reference holds no
	// token either, and the collection cannot outlive the Game that owns it.
	if _, err := game.gameComponents.AddComponentAddedHandler(game.gameComponentAdded); err != nil {
		return nil, err
	}
	if _, err := game.gameComponents.AddComponentRemovedHandler(game.gameComponentRemoved); err != nil {
		return nil, err
	}
	game.runtime = interop.NewRuntime(gameRuntimeCallbacks{game: game})
	interop.RegisterOwner(game, game.runtime, nil)
	return game, nil
}

// Components is Game::get_Components, whose whole body is
//
//	ldarg.0; ldfld GameComponentCollection Game::gameComponents; ret
//
// It is a field read of an object the constructor allocated once, so it cannot
// fail, never allocates, and returns the same collection every time. The
// collection keeps CLR reference semantics: a caller who mutates what this
// returns mutates the Game's components, and every other caller sees it.
func (g *Game) Components() *GameComponentCollection {
	return g.gameComponents
}

// Services is Game::get_Services, the same one-`ldfld` shape over the container
// the constructor allocated. It cannot fail and returns one stable identity.
//
// The container is the reference's own service registry and is genuinely
// public: anything may register into it, and Game itself reads it during the
// run sequence to find its IGraphicsDeviceManager.
func (g *Game) Services() *GameServiceContainer {
	return g.gameServices
}

// Run creates and runs the admitted CNA ABI 0.7 Game on a locked OS thread.
//
// The `inRun` reset reproduces RunGame's finally block, which clears the flag
// once a blocking run has returned:
//
//	finally { if (!endRunRequired) inRun = false; }
//
// endRunRequired is set only on the non-blocking StartGameLoop path, which is
// GameHost's, so for a blocking Run the reset always happens.
func (g *Game) Run() error {
	if g == nil || g.runtime == nil {
		return errors.New("Game is nil or uninitialized")
	}
	err := g.runtime.Run()
	g.inRun = false
	return err
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

// Initialize runs the consumer's Initialize override and then raises `inRun`.
//
// The reference sequences exactly this in RunGame:
//
//	this.Initialize();      // the virtual, so the derived override runs
//	this.inRun = true;
//
// so a component added from inside the override is still queued, and one added
// after it is initialized on the spot. CNA-Go raises the flag at the same point
// in the sequence, on the native `initialize` frame hook, because the native
// CNA host plays GameHost's part. A failing override leaves the flag down, as
// the reference's exception leaves the assignment unreached.
func (c gameRuntimeCallbacks) Initialize() error {
	if err := c.game.callbacks.Initialize(c.game); err != nil {
		return err
	}
	c.game.inRun = true
	return nil
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

// GameEvent is the private end of the native game-event bridge. It is NOT a
// GameCallbacks member: the five-member override contract is unchanged, and a
// consumer implements nothing new to receive these signals.
func (c gameRuntimeCallbacks) GameEvent(event uint32) error {
	return c.game.raiseNativeGameEvent(event)
}

// FrameHookOverrides reports exactly the hooks the callback object supplied an
// override for. A bit that is clear leaves that CNA_GameFrameHooks member NULL,
// which the canonical header defines as simply not called.
//
// It is derived from the fields NewGame captured, so the mask and the four
// dispatch methods below cannot disagree: the same nil-ness decides both.
func (c gameRuntimeCallbacks) FrameHookOverrides() interop.FrameHookMask {
	var mask interop.FrameHookMask
	if c.game.beginRunOverride != nil {
		mask |= interop.FrameHookBeginRun
	}
	if c.game.endRunOverride != nil {
		mask |= interop.FrameHookEndRun
	}
	if c.game.beginDrawOverride != nil {
		mask |= interop.FrameHookBeginDraw
	}
	if c.game.endDrawOverride != nil {
		mask |= interop.FrameHookEndDraw
	}
	return mask
}

// The four optional frame-hook dispatchers. Each is reached only from the
// native hook its own mask bit installed, so the nil branch is unreachable by
// construction; it is reported rather than quietly running the base, because
// running a base the consumer did not ask for is exactly what this whole
// mechanism exists to avoid.
func (c gameRuntimeCallbacks) BeginRun() error {
	if c.game.beginRunOverride == nil {
		return errFrameHookWithoutOverride
	}
	return c.game.beginRunOverride.BeginRun(c.game)
}

func (c gameRuntimeCallbacks) EndRun() error {
	if c.game.endRunOverride == nil {
		return errFrameHookWithoutOverride
	}
	return c.game.endRunOverride.EndRun(c.game)
}

// BeginDraw forwards the override's two channels unchanged. A refusal is
// (false, nil) and is never promoted to an error, and an error never decides
// the frame.
func (c gameRuntimeCallbacks) BeginDraw() (bool, error) {
	if c.game.beginDrawOverride == nil {
		return false, errFrameHookWithoutOverride
	}
	return c.game.beginDrawOverride.BeginDraw(c.game)
}

func (c gameRuntimeCallbacks) EndDraw() error {
	if c.game.endDrawOverride == nil {
		return errFrameHookWithoutOverride
	}
	return c.game.endDrawOverride.EndDraw(c.game)
}

// errFrameHookWithoutOverride reports a native frame hook that arrived with no
// override behind it. It has no CLR counterpart and is unreachable while the
// mask and the captured fields are derived from each other; it exists so that
// a future divergence between them fails loudly instead of silently running a
// base body at a position CNA-Go picked.
var errFrameHookWithoutOverride = errors.New("a native frame hook was delivered for a Game with no override for it")

func gameTimeFromInterop(value interop.FrameTime) GameTime {
	return NewGameTimeByTimeSpanAndTimeSpanAndBoolean(
		TimeSpanFromTicks(value.TotalTicks),
		TimeSpanFromTicks(value.ElapsedTicks),
		value.IsRunningSlowly,
	)
}

// GraphicsDeviceManager owns the Game's canonical native graphics manager.
type GraphicsDeviceManager struct {
	runtime               *interop.Runtime
	resource              *interop.Resource
	supportedOrientations DisplayOrientation
	isDeviceDirty         bool
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
	manager := &GraphicsDeviceManager{
		runtime:               game.runtime,
		resource:              resource,
		supportedOrientations: DisplayOrientationDefault,
		isDeviceDirty:         false,
	}
	interop.RegisterOwner(manager, manager.runtime, manager.resource)
	return manager, nil
}

// SupportedOrientations returns the exact stored XNA orientation flags.
func (m *GraphicsDeviceManager) SupportedOrientations() DisplayOrientation {
	return m.supportedOrientations
}

// SetSupportedOrientations stores the exact flags and always marks future
// device configuration dirty, matching the XNA setter.
func (m *GraphicsDeviceManager) SetSupportedOrientations(value DisplayOrientation) {
	m.supportedOrientations = value
	m.isDeviceDirty = true
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
