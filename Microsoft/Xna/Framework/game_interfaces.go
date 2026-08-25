package framework

// IGameComponent is XNA's one-method game component initialization contract.
//
// It keeps a final error result because it is a runtime-boundary contract, not
// because interfaces are fallible by default. In Microsoft.Xna.Framework.Game.dll
// the two shipped implementors disagree about how much they do:
// GameComponent.Initialize is a bare `ret`, while DrawableGameComponent.Initialize
// resolves IGraphicsDeviceService out of Game.Services, throws
// System.InvalidOperationException when the service is absent, subscribes to
// four device events, reads GraphicsDevice, and calls LoadContent. The contract
// therefore genuinely reaches the game and graphics runtime, and an
// implementation that cannot initialize needs somewhere to say so.
//
// As of Foundation 32 the contract has a shipped implementor: GameComponent
// satisfies it, and Game's component engine calls Initialize through it -- from
// the pending-queue drain in base Initialize, and directly from the collection
// handler for a component added while the game is running. GameComponent's own
// body is a bare `ret`, so it carries the channel and never uses it.
type IGameComponent interface {
	Initialize() error
}

// IGraphicsDeviceManager is XNA's device lifecycle contract, the one Game calls
// once per device creation and twice per frame.
//
// Every operation crosses a qualified runtime boundary, which is why every one
// of them is fallible. In the reference, CreateDevice delegates to
// GraphicsDeviceManager.ChangeDevice, BeginDraw to EnsureDevice, and EndDraw
// calls GraphicsDevice.Present inside a catch for
// Graphics.DeviceLostException. BeginDraw's Boolean is the source result --
// whether drawing may proceed -- and stays a separate channel from the error,
// exactly as the nullable and out-parameter rules keep their channels separate.
//
// Declaring the contract binds no implementor. CNA-Go's GraphicsDeviceManager
// remains a partial native-backed facade and is not projected as implementing
// this interface.
type IGraphicsDeviceManager interface {
	CreateDevice() error
	BeginDraw() (bool, error)
	EndDraw() error
}
