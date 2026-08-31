package framework

// This file projects Microsoft.Xna.Framework.Game's disposal surface:
//
//	.method public hidebysig newslot virtual final instance void Dispose()
//	.method family  hidebysig         virtual instance void Finalize()
//	.method family  hidebysig newslot virtual instance void Dispose(bool disposing)
//
// and it MOVES Game.Disposed to the raise site the reference actually uses.
//
// # The correction
//
// Foundation 34 bound CNA_GAME_EVENT_DISPOSED to the public Game.Disposed
// event, because Game::Dispose was not projected and the native signal was the
// only disposal fact CNA-Go had. That was a recorded divergence, and it is now
// corrected rather than preserved.
//
// The pinned IL is unambiguous. Game::Disposed has no On... method and exactly
// one raise site, at the tail of Dispose(bool):
//
//	IL_006d: ldarg.0
//	IL_006e: ldfld      EventHandler`1<EventArgs> Game::Disposed
//	IL_0073: brfalse.s  IL_0086
//	IL_0075: ldarg.0
//	IL_0076: ldfld      EventHandler`1<EventArgs> Game::Disposed
//	IL_007b: ldarg.0                                   // `this` is the sender
//	IL_007c: ldsfld     EventArgs EventArgs::Empty
//	IL_0081: callvirt   EventHandler`1::Invoke(object, !0)
//
// It is raised from MANAGED disposal, by the consumer's own Dispose call, and
// by nothing else. A consumer who never disposes never sees it -- that is the
// contract, not an omission.
//
// CNA's own signal is raised from native game destruction at the end of the
// runtime host's lifetime, which is a different observable moment: it fires
// whether or not anyone disposed the Game, and it does not fire when a consumer
// disposes without ever running. Those are different semantics, and XNA
// raise-site fidelity wins. The native signal stays bound and stays measured --
// it is what proves the subscription lifetime spans cna_game_destroy -- but it
// is a CNA lifecycle signal, not the XNA event's raise path, and it raises
// nothing public.
//
// # Native host destruction and managed disposal are separate concepts
//
// Runtime.Run destroys the native host when the run ends, because the native
// generation cannot outlive it. That is an implementation detail of running,
// not a statement that the XNA object was disposed: the reference's own
// RunGame does not dispose the Game either. A consumer may therefore call
// Dispose after Run returns and get the full managed semantics, on a Game with
// no native handle left. Nothing here reaches native code, acquires a handle,
// or pretends one is alive.
//
// # Dispose is NOT idempotent, and neither is the reference
//
// There is no disposed flag anywhere in Game. Every call re-runs the whole
// body: it re-copies the components, disposes each again, and raises Disposed
// AGAIN. GameComponent already carries exactly this behaviour for exactly this
// reason, and it is reproduced here rather than smoothed into the idempotent
// Dispose a reader might expect.

// gameComponentDisposable and gameComponentSimpleDisposable are the Go
// projection of the reference's `array[i] as IDisposable`.
//
// System.IDisposable contributes NO projected Go interface -- the settled BCL
// interface rule -- so there is no framework.IDisposable to assert against.
// What the `isinst` actually tests is whether the component declares the
// interface's single member, and the settled overload rule decides what that
// member is called in Go:
//
//	declares only Dispose()                -> Dispose() error
//	declares Dispose() and Dispose(bool)   -> DisposeByNone() error
//
// Microsoft's own two components, GameComponent and DrawableGameComponent,
// take the second spelling. A consumer's own component may legitimately take
// either, so both are accepted and nothing else is: a method of another name,
// or of another shape, is not the IDisposable member and the reference's
// `isinst` would not have matched it either.
//
// The two-overload spelling is tried first because it is the one the
// reference's own components have.
type gameComponentDisposable interface {
	DisposeByNone() error
}

type gameComponentSimpleDisposable interface {
	Dispose() error
}

// DisposeByNone is Game::Dispose(), the sealed IDisposable member:
//
//	Dispose(true);
//	GC.SuppressFinalize(this);
//
// It is `newslot virtual final` -- declared on the interface slot and sealed --
// so no CLR subclass can change it, and the projection is a plain method for
// the same reason GameComponent's is.
//
// GC.SuppressFinalize has nothing to suppress. CNA-Go registers no Go finalizer
// on a Game, so there is no finalization to cancel; the call is a no-op rather
// than an unreproduced step.
func (g *Game) DisposeByNone() error {
	return g.DisposeByBoolean(true)
}

// Finalize is Game::Finalize, the protected finalizer:
//
//	try { Dispose(false); } finally { base.Finalize(); }
//
// The body is reproduced exactly, which is why it is one line: Dispose(false)
// is the whole try block, and System.Object::Finalize is a bare `ret` that Go
// has no counterpart for and needs none.
//
// Dispose(false) returns at its first instruction, so this is observably a
// no-op on any Game -- including one whose constructor never ran, because the
// disposing check comes BEFORE the state guard. It carries an error because
// Game is a native-backed facade and every member of one does; it never uses
// it, and the value it forwards is Dispose(false)'s, not an invented nil.
//
// GameComponent::Finalize projects without an error result and that is not an
// inconsistency: GameComponent is classified pure managed, so its members start
// from infallible, while Game's start from fallible. The classification is per
// type and is read from the reference, not chosen per member.
//
// Go has no CLR finalization and CNA-Go registers no runtime finalizer for a
// Game, so nothing calls this on its own. It is projected because the pinned
// contract declares it.
func (g *Game) Finalize() error {
	return g.DisposeByBoolean(false)
}

// DisposeByBoolean is Game::Dispose(bool), the protected virtual that is the
// whole of Game's disposal behaviour:
//
//	if (!disposing) return;
//	lock (this)
//	{
//	    IGameComponent[] array = new IGameComponent[this.gameComponents.Count];
//	    this.gameComponents.CopyTo(array, 0);
//	    for (int i = 0; i < array.Length; i++)
//	        if (array[i] is IDisposable d) d.Dispose();
//	    if (this.graphicsDeviceManager is IDisposable m) m.Dispose();
//	    this.UnhookDeviceEvents();
//	    if (this.Disposed != null) this.Disposed(this, EventArgs.Empty);
//	}
//
// # The snapshot is load-bearing
//
// The copy is not defensive style; it is required. GameComponent::Dispose(bool)
// runs `Game.Components.Remove(this)`, so disposing a component MUTATES the
// collection being walked. The reference copies first and walks the copy, and
// so does this. A component that removes a *different* component from inside
// its own disposal is therefore still disposed here, exactly as in CLR.
//
// # No exception handler, so a failing component stops the rest
//
// There is no try/catch. A component whose Dispose throws propagates straight
// out of Game.Dispose, leaving every later component undisposed and the
// Disposed event unraised. The projection returns that error at the same point
// and leaves exactly the same debris.
//
// # The two device steps
//
// `graphicsDeviceManager as IDisposable` reads Game::graphicsDeviceManager,
// which has exactly one assignment in the whole class, at the head of RunGame,
// and CNA-Go does not perform that resolution -- the already-recorded
// architecture deferral the two draw hooks carry. The field is permanently
// null, which is a state the reference itself has whenever no manager is
// registered, so the guarded branch is simply not taken.
//
// UnhookDeviceEvents is reproduced as written and its guard is likewise always
// false. Its whole body is
//
//	if (graphicsDeviceService != null) { remove four device handlers }
//
// and graphicsDeviceService has exactly one assignment in the class, inside
// HookDeviceEvents, which base Initialize calls and which CNA-Go records as
// deferred. Nothing here has to assume that: the unhook step removes handlers
// that a hook step added, so with the hook step unreached there is provably
// nothing to remove, and the absence is unobservable at THIS member. What a
// consumer could observe -- their own IGraphicsDeviceService never being
// subscribed to -- belongs to Initialize and is recorded there.
//
// # Fallibility
//
// Two of the steps can fail and neither failure is invented. A component's own
// disposal is fallible on the settled contract, and raising Disposed runs
// consumer handlers whose failure is not discarded. The Go-only guard is the
// same one every Game member carries.
//
// # The lock
//
// `lock (this)` takes the Game's own monitor, and CLR monitors are REENTRANT
// per thread. Reentry is reachable: a Disposed handler, or a ComponentRemoved
// handler raised from a component's own disposal, may call Dispose again. Go
// has no reentrant mutex and no supported thread identity, so this takes the
// lock with TryLock -- an uncontended call holds it for the whole critical
// section exactly as Monitor does, and a reentrant call proceeds without
// re-acquiring, which is what Monitor's recursion does. The one divergence is
// that a genuinely CONCURRENT second disposer proceeds instead of blocking.
// That is the settled projection GameComponent::Dispose(bool) already carries,
// and it is recorded rather than hidden: a guaranteed deadlock on a reachable
// single-threaded path is the worse of the two errors.
func (g *Game) DisposeByBoolean(disposing bool) error {
	// IL_0000: ldarg.1; brfalse IL_0094 -- before anything else, including
	// any state check, which is why Dispose(false) is safe on any Game.
	if !disposing {
		return nil
	}
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	if g.disposeLock.TryLock() {
		defer g.disposeLock.Unlock()
	}
	components := make([]IGameComponent, g.gameComponents.Count())
	if err := g.gameComponents.CopyTo(components, 0); err != nil {
		return err
	}
	for i := 0; i < len(components); i++ {
		if err := disposeGameComponent(components[i]); err != nil {
			return err
		}
	}
	// if (graphicsDeviceManager is IDisposable d) d.Dispose();
	// The field is permanently null here; the branch is not taken.
	//
	// UnhookDeviceEvents();
	// Its guard is graphicsDeviceService, which only HookDeviceEvents assigns
	// and which is never reached, so there is nothing subscribed to remove.
	raiseErr := g.disposed.Raise(g, EventArgsEmpty())

	// The reference's body ends here, and for a Game that only ever used Run
	// so does this one: Run destroys the native game it created, so by the
	// time Dispose runs there is nothing native left and the step below does
	// nothing at all.
	//
	// Foundation 47 added one state the reference cannot have. A Game whose
	// frames were stepped with Tick or RunOneFrame has a live native game that
	// no Run will ever return from and destroy, and CNA admits exactly ONE
	// C-owned game per process -- so a standalone session nothing ended would
	// make the next Game impossible to create.
	//
	// It runs AFTER the managed body on purpose. Every component's own Dispose
	// and every Disposed handler therefore observes the same live device the
	// reference's would, and the managed order the corpus and the canary
	// measure is byte-for-byte unchanged.
	//
	// It is not a divergence from the reference's Dispose. The reference's host
	// is created by the CONSTRUCTOR and outlives Dispose because the process
	// owns it; CNA-Go's is created by a frame step, so ending it is the
	// disposal of a resource whose creation moved, not the disposal of one the
	// reference keeps.
	sessionErr := endGameStandaloneSession(g)
	if raiseErr != nil {
		return raiseErr
	}
	return sessionErr
}

// endGameStandaloneSession destroys a native game a frame step created, and
// does nothing when Run created it or when there is none.
func endGameStandaloneSession(g *Game) error {
	if g == nil || g.runtime == nil {
		return nil
	}
	return g.runtime.EndStandaloneSession()
}

// disposeGameComponent is the reference's `isinst IDisposable` followed by the
// guarded `callvirt IDisposable::Dispose()`. A component that is not disposable
// is skipped with no error, exactly as the null branch does.
func disposeGameComponent(component IGameComponent) error {
	switch disposable := component.(type) {
	case gameComponentDisposable:
		return disposable.DisposeByNone()
	case gameComponentSimpleDisposable:
		return disposable.Dispose()
	default:
		return nil
	}
}
