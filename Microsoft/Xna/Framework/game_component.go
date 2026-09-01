package framework

import "sync"

// GameComponent is XNA's one concrete game component: the class a consumer
// derives from to be initialized once and updated every frame, and the only
// implementor of IUpdateable the reference ships.
//
// # Why it could not be projected before
//
// Its whole public surface is managed field work, but one member was not
// reachable. Dispose(bool) runs
//
//	get_Game().get_Components().Remove(this)
//
// so the type could not exist until Game exposed Components. Foundation 30 did
// that; nothing else about the class changed.
//
// # Every default is from the constructor, and there is no validation
//
//	.ctor(Game game)
//	  ldarg.0; ldc.i4.1; stfld enabled      // field initializer: Enabled = true
//	  ldarg.0; call Object::.ctor()
//	  ldarg.0; ldarg.1; stfld game
//	  ret                                    // code size 21
//
// Enabled defaults to true, UpdateOrder to zero, and the Game reference is
// stored exactly as handed over. **The constructor does not null-check the
// Game**, so a component with no Game is legal and simply skips the removal in
// Dispose -- which is the one place the reference reads the field back.
//
// # The two settable properties suppress an unchanged value
//
//	set_Enabled(bool value)
//	  ldarg.0; ldfld enabled; ldarg.1; beq.s RET   // unchanged -> nothing at all
//	  ldarg.0; ldarg.1; stfld enabled              // store FIRST
//	  ldarg.0; ldarg.0; ldsfld EventArgs::Empty
//	  callvirt OnEnabledChanged(object, EventArgs) // then announce
//
// Store then announce, and only when the value actually changed. UpdateOrder
// is byte-for-byte the same shape. That suppression is load-bearing for Game's
// component engine: an assignment that changes nothing must not re-place the
// component in the ordered list.
//
// # The two On... methods ignore their sender argument
//
//	OnEnabledChanged(object sender, EventArgs args)
//	  ldarg.0; ldfld EnabledChanged; brfalse.s RET
//	  ldarg.0; ldfld EnabledChanged
//	  ldarg.0                                      // <- `this`, NOT `sender`
//	  ldarg.2                                      // the args, forwarded
//	  callvirt EventHandler`1::Invoke(object, EventArgs)
//
// The `sender` parameter is accepted, ignored, and replaced by `this` at the
// raise. That is not a curiosity: Game's UpdateableUpdateOrderChanged handler
// reads the SENDER to decide which component to re-place, so this is what makes
// the engine work. Passing the argument through would break it.
//
// # Nothing here is native
//
// Every member above is one field access, one comparison, or one delegate
// invoke. The class reaches no device, no host, and no CNA handle. It is
// classified pure managed for exactly that reason, and its three fallible
// members are fallible for three specific reasons rather than because it is a
// class.
type GameComponent struct {
	enabled     bool
	updateOrder int32
	game        *Game

	enabledChanged     EventSource[*EventArgs]
	updateOrderChanged EventSource[*EventArgs]
	disposed           EventSource[*EventArgs]

	// disposeLock projects the `lock (this)` that wraps the whole body of
	// Dispose(bool). See DisposeByBoolean for the exact concurrency
	// projection and its one deliberate divergence.
	disposeLock sync.Mutex

	// derived is the CLR `this`: the outermost object this component is the
	// base of, or nil when the component IS the outermost object.
	//
	// # Why a composed base needs it
	//
	// In CLR, `ldarg.0` inside GameComponent's body is the WHOLE object -- a
	// DrawableGameComponent when that is what was constructed. Private named
	// composition splits that object in two, and the base half has no way back
	// to the whole one, so every reference site that uses `ldarg.0` as an
	// OBJECT rather than as a path to a field silently means the wrong thing:
	//
	//	Game.Components.Remove(this)     removes the base, which is not in the
	//	                                 collection -- the derived object is
	//	Disposed(this, EventArgs.Empty)  hands a consumer the base object as
	//	                                 the sender of the derived one's event
	//
	// Both were live: Milestone 55 measured a DrawableGameComponent surviving
	// Game.Dispose in Game.Components, and its three events announcing a sender
	// no consumer can match against the component it registered on.
	//
	// The reference is installed by the derived constructor through
	// bindDerived, is never exported, and is read only through self(). It is
	// the settled projection of CLR object identity under composition and the
	// GraphicsResource chain uses the same one.
	derived IGameComponent
}

// bindDerived installs the CLR `this` for a composed base. It is called by the
// constructor of every type that composes a GameComponent, and by nothing else.
func (c *GameComponent) bindDerived(derived IGameComponent) { c.derived = derived }

// self is the CLR `ldarg.0` as an OBJECT: the outermost object of the
// composition chain, which is the component itself when nothing composes it.
func (c *GameComponent) self() IGameComponent {
	if c.derived != nil {
		return c.derived
	}
	return c
}

// NewGameComponent projects the one public constructor. It validates nothing
// and cannot fail: a nil Game is stored like any other, exactly as the
// reference stores a null one.
func NewGameComponent(game *Game) *GameComponent {
	return &GameComponent{enabled: true, game: game}
}

// Game is get_Game, one `ldfld`. It cannot fail and returns whatever the
// constructor was handed, including nil.
func (c *GameComponent) Game() *Game { return c.game }

// Enabled is get_Enabled, one `ldfld`. IUpdateable declares it get-only and
// infallible, and the class agrees.
func (c *GameComponent) Enabled() bool { return c.enabled }

// SetEnabled is set_Enabled: suppress an unchanged value, store, then announce.
//
// It is fallible only because announcing runs consumer handlers, and a handler
// failure is not discarded. A suppressed assignment announces nothing and
// therefore cannot fail.
func (c *GameComponent) SetEnabled(value bool) error {
	if c.enabled == value {
		return nil
	}
	c.enabled = value
	return c.OnEnabledChanged(c.self(), EventArgsEmpty())
}

// UpdateOrder is get_UpdateOrder, one `ldfld`.
func (c *GameComponent) UpdateOrder() int32 { return c.updateOrder }

// SetUpdateOrder is set_UpdateOrder, the same shape as SetEnabled.
//
// When the component belongs to a Game, the announcement reaches Game's engine
// and re-places the component in the ordered update list, so a changed order
// takes effect from the next base Update.
func (c *GameComponent) SetUpdateOrder(value int32) error {
	if c.updateOrder == value {
		return nil
	}
	c.updateOrder = value
	return c.OnUpdateOrderChanged(c.self(), EventArgsEmpty())
}

// Initialize is GameComponent::Initialize, whose body is
//
//	IL_0000: ret            // code size 1
//
// so the base class does nothing. It keeps an error result because
// IGameComponent declares one: the contract is fallible on the evidence of its
// OTHER implementor, DrawableGameComponent, whose Initialize resolves
// IGraphicsDeviceService out of Game.Services and throws when it is absent. Go
// requires an exact signature to satisfy an interface, so a member that does
// nothing still carries the contract's channel. It never uses it.
func (c *GameComponent) Initialize() error { return nil }

// Update is GameComponent::Update, also a bare `ret` of code size 1. IUpdateable
// is infallible, which Foundation 23 read from this very body, so the projection
// has no error result to carry.
func (c *GameComponent) Update(gameTime GameTime) {}

// OnEnabledChanged is the protected virtual raise site.
//
//	if (EnabledChanged != null) EnabledChanged(this, args);
//
// It ignores `sender` and raises with the component itself, which is what
// Game's engine reads. Both parameters are still projected because the pinned
// contract declares them.
func (c *GameComponent) OnEnabledChanged(sender any, args *EventArgs) error {
	return c.enabledChanged.Raise(c.self(), args)
}

// OnUpdateOrderChanged is the same shape over UpdateOrderChanged, and ignores
// its sender argument the same way.
func (c *GameComponent) OnUpdateOrderChanged(sender any, args *EventArgs) error {
	return c.updateOrderChanged.Raise(c.self(), args)
}

// DisposeByNone is Dispose(), the sealed IDisposable member:
//
//	Dispose(true);
//	GC.SuppressFinalize(this);
//
// It is `newslot virtual final` in the reference -- declared on the interface
// slot and sealed -- so no CLR subclass can change it, and the projection is a
// plain method for the same reason.
//
// GC.SuppressFinalize has nothing to suppress here. CNA-Go registers no Go
// finalizer on a GameComponent, so there is no finalization to cancel; the call
// is a no-op rather than an unreproduced step.
//
// It is NOT idempotent, and the reference is not either. A second Dispose finds
// the component already gone from Components, so the removal reports false and
// is discarded, and the Disposed event is raised AGAIN.
func (c *GameComponent) DisposeByNone() error {
	return c.DisposeByBoolean(true)
}

// Finalize is the protected finalizer, which the reference declares as
//
//	try { Dispose(false); } finally { base.Finalize(); }
//
// Dispose(false) returns immediately, so the whole method is observably a
// no-op and this faithfully does nothing. It cannot fail: the one branch it
// takes has no failure in it.
//
// Go has no CLR finalization, and CNA-Go registers no runtime finalizer for a
// GameComponent, so nothing calls this on its own. It is projected because the
// pinned contract declares it.
func (c *GameComponent) Finalize() {}

// DisposeByBoolean is Dispose(bool), the protected virtual:
//
//	if (!disposing) return;
//	lock (this)
//	{
//	    if (this.Game != null)
//	        this.Game.Components.Remove(this);
//	    if (this.Disposed != null)
//	        this.Disposed(this, EventArgs.Empty);
//	}
//
// # The order is removal first, announcement second
//
// A Disposed handler therefore observes a Game whose Components no longer
// contains the component, and whose update and draw lists have already
// untracked it -- because Components.Remove reaches RemoveItem, which raises
// ComponentRemoved, which Game's own engine handler consumes before this method
// gets to its own event.
//
// # Fallibility
//
// Two of the three statements can fail, and neither failure is invented.
// Components.Remove is fallible on the settled collection projection, and
// raising Disposed runs consumer handlers whose failure is not discarded. The
// reference discards the BOOLEAN result of Remove -- there is a `pop` -- and so
// does this; the error is a different channel and is not discarded.
//
// # The Go-only Game guard
//
// The reference tests `Game != null`. In Go a consumer can hold a Game whose
// constructor never ran, which has no Components at all, so the guard is
// "does this component's Game carry its managed state" -- the same question in
// the only form Go can ask it.
//
// # The concurrency projection, and its one divergence
//
// CLR's Monitor is REENTRANT per thread, and reentry here is reachable: a
// Disposed handler, or a ComponentRemoved handler, may call Dispose again. Go
// has no reentrant mutex and no supported thread identity, so a plain Lock
// would deadlock on a path the reference merely recurses on -- strictly worse
// than the reference.
//
// The projection therefore takes the lock with TryLock: an uncontended call
// enters and holds it for the whole critical section, exactly as Monitor does,
// and a reentrant call proceeds without re-acquiring, which is what Monitor's
// recursion does. The divergence is that a genuinely CONCURRENT second
// disposer would proceed instead of blocking. That is deliberate and is
// recorded rather than hidden: CNA-Go's Game and component state is owner-
// thread state, the binding promises no cross-goroutine safety for it, and a
// guaranteed deadlock on a reachable single-threaded path is the worse of the
// two errors.
func (c *GameComponent) DisposeByBoolean(disposing bool) error {
	if !disposing {
		return nil
	}
	if c.disposeLock.TryLock() {
		defer c.disposeLock.Unlock()
	}
	if gameManagedState(c.game) {
		if _, err := c.game.Components().Remove(c.self()); err != nil {
			return err
		}
	}
	return c.disposed.Raise(c.self(), EventArgsEmpty())
}

// ---------------------------------------------------------------------------
// The three events, on the settled two-accessor projection.
// ---------------------------------------------------------------------------

// AddEnabledChangedHandler registers a handler for EnabledChanged, which
// set_Enabled raises through OnEnabledChanged when the value actually changes.
func (c *GameComponent) AddEnabledChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.enabledChanged.Add(handler)
}

// RemoveEnabledChangedHandler removes the registration the token names.
func (c *GameComponent) RemoveEnabledChangedHandler(subscription EventSubscription) error {
	return c.enabledChanged.Remove(subscription)
}

// AddUpdateOrderChangedHandler registers a handler for UpdateOrderChanged.
//
// A Game this component belongs to has already registered its own handler
// here, added when the component joined Components, so a consumer's handler
// runs after the engine has re-placed the component.
func (c *GameComponent) AddUpdateOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.updateOrderChanged.Add(handler)
}

// RemoveUpdateOrderChangedHandler removes the registration the token names.
func (c *GameComponent) RemoveUpdateOrderChangedHandler(subscription EventSubscription) error {
	return c.updateOrderChanged.Remove(subscription)
}

// AddDisposedHandler registers a handler for Disposed, which Dispose(true)
// raises AFTER the component has left Game.Components.
func (c *GameComponent) AddDisposedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.disposed.Add(handler)
}

// RemoveDisposedHandler removes the registration the token names.
func (c *GameComponent) RemoveDisposedHandler(subscription EventSubscription) error {
	return c.disposed.Remove(subscription)
}

// The reference declares IGameComponent, IUpdateable and IDisposable on this
// class, so the Go projection must satisfy the Go projections of the two that
// CNA-Go projects. These are compiler-level witnesses: if either contract
// stopped being satisfied, this file would not build.
//
// System.IDisposable is not an XNA type and CNA-Go projects no Go interface for
// it; the settled IDisposable relationship is a measured one rather than a Go
// contract.
var (
	_ IGameComponent = (*GameComponent)(nil)
	_ IUpdateable    = (*GameComponent)(nil)
)
