package framework

// IUpdateable is XNA's per-frame update contract: the shape Game's component
// loop calls once per Update, ordered by UpdateOrder and skipped when Enabled
// is false.
//
// Every operation is infallible. Microsoft.Xna.Framework.Game.dll declares the
// contract and ships exactly one implementor, GameComponent, whose IL is
// managed field work throughout: get_Enabled and get_UpdateOrder are one ldfld
// each, and Update(GameTime) is a bare `ret` of code size 1. Nothing in the
// contract reaches a device, an allocation, or native code, so nothing here
// gains a synthetic error result.
//
// That is deliberately a different verdict from IGameComponent, which keeps a
// fallible Initialize because DrawableGameComponent.Initialize resolves
// IGraphicsDeviceService out of Game.Services and throws when it is absent. The
// boundary is read per contract from its own implementor IL, not inherited from
// a neighbouring contract on the same class.
//
// Both properties are get-only in the CLR interface. GameComponent declares
// setters, but they are members of the class, not requirements of the contract.
//
// The two events are the only fallible-looking operations, and their error
// belongs to the settled event accessor projection rather than to this
// contract's boundary. A conformer satisfies them by delegating to an
// EventSource, which is why EventSource is public language support:
//
//	type Spinner struct {
//	    enabledChanged EventSource[*EventArgs]
//	    enabled        bool
//	}
//
//	func (s *Spinner) AddEnabledChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
//	    return s.enabledChanged.Add(h)
//	}
//
//	func (s *Spinner) RemoveEnabledChangedHandler(sub EventSubscription) error {
//	    return s.enabledChanged.Remove(sub)
//	}
//
// As of Foundation 32 this contract is live: GameComponent satisfies it, Game
// keeps an ordered list of every IUpdateable in Components, and base Update
// calls Update on each Enabled one in UpdateOrder.
type IUpdateable interface {
	Enabled() bool
	UpdateOrder() int32
	Update(gameTime GameTime)
	AddEnabledChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveEnabledChangedHandler(subscription EventSubscription) error
	AddUpdateOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveUpdateOrderChangedHandler(subscription EventSubscription) error
}

// IDrawable is XNA's per-frame draw contract, the same shape as IUpdateable
// over Visible and DrawOrder.
//
// Every operation is infallible for the same reason and on the same evidence:
// in Microsoft.Xna.Framework.Game.dll the one shipped implementor,
// DrawableGameComponent, backs get_Visible and get_DrawOrder with one ldfld
// each and implements Draw(GameTime) as a bare `ret` of code size 1. The class
// reaches the graphics runtime through get_GraphicsDevice and Initialize, which
// this contract does not declare.
//
// Game keeps an ordered list of every IDrawable in Components and base Draw
// calls Draw on each Visible one in DrawOrder. The reference's own implementor,
// DrawableGameComponent, is still missing -- it crosses the protected graphics
// runtime -- so the live implementors are consumers' own types.
type IDrawable interface {
	Visible() bool
	DrawOrder() int32
	Draw(gameTime GameTime)
	AddVisibleChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveVisibleChangedHandler(subscription EventSubscription) error
	AddDrawOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveDrawOrderChangedHandler(subscription EventSubscription) error
}

// GameComponentCollectionEventArgs carries the component a
// GameComponentCollection change concerns.
//
// Its CLR base is System.EventArgs, which CNA-Go models as a measured
// relationship rather than as Go embedding: this is its own reference type and
// inherits no member. The whole type is one private IGameComponent field.
// Microsoft.Xna.Framework.Game.dll shows the constructor as
// `call EventArgs::.ctor` followed by a single stfld with no validation, and
// get_GameComponent as one ldfld, so neither operation can fail and a nil
// component is stored exactly as the reference stores a null one.
//
// XNA declares the constructor public, so CNA-Go projects it.
// GameComponentCollection raises it on every insertion and removal, and Game's
// component engine is its first consumer.
type GameComponentCollectionEventArgs struct {
	gameComponent IGameComponent
}

// NewGameComponentCollectionEventArgs stores the component and validates
// nothing, exactly as the reference constructor does.
func NewGameComponentCollectionEventArgs(gameComponent IGameComponent) *GameComponentCollectionEventArgs {
	return &GameComponentCollectionEventArgs{gameComponent: gameComponent}
}

// GameComponent returns the component the change concerns.
func (a *GameComponentCollectionEventArgs) GameComponent() IGameComponent {
	return a.gameComponent
}
