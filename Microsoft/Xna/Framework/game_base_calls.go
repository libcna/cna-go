package framework

import "errors"

// This file is CNA-Go language support, not XNA surface.
//
// The five exported functions below are declared in
// tools/api_compat/mapping-rules.json under gameBaseCallAdapters and are
// measured there. None of them is an XNA identity, none is counted in
// REFERENCE_MEMBERS, and none is a member of Game or of GameCallbacks.
//
// # Why they have to exist
//
// In CLR, a derived Game decides whether and when the base runs:
//
//	protected override void Update(GameTime t)
//	{
//	    this.spawner.Tick(t);
//	    base.Update(t);          // <- the derived class chose this, and here
//	}
//
// Omitting `base.Update(t)` is legal and means the component loop does not run
// that frame. Putting it first instead of last changes what runs before what.
// The call site is the semantics.
//
// GameCallbacks is CNA-Go's projection of those protected virtual overrides,
// and Go has neither inheritance nor a `base` keyword. If CNA-Go ran the base
// body automatically around every callback, every override would be a
// mandatory `base.Update(t)` at a position CNA-Go picked -- which is not
// XNA's contract but a different one that happens to resemble it.
//
// So base behavior is never automatic. It is reached explicitly, by calling
// one of these, and a callback that never calls one gets no base behavior at
// all. That is the whole point of the family.
//
// # What they are not
//
// They are package-level functions, not methods on Game, so the projected XNA
// member surface of Game is not polluted with names Microsoft never declared.
// They take the Game as their first parameter, which is the `this` a CLR base
// call passes implicitly.
//
// They never invoke GameCallbacks. A base body in the reference never calls
// the virtual it is the base of -- that would be unbounded recursion -- and
// neither does any of these. Game::Initialize does end in a `callvirt` to
// Game::LoadContent, which IS a virtual dispatch to the derived override, and
// that one step is measured as a deferral for a separate reason recorded
// below rather than being silently reproduced as a callback invocation.
//
// # Why every one of them is fallible
//
// Each takes a *Game the consumer supplies, and Go -- unlike CLR, where a
// constructor always ran -- lets a consumer write `&framework.Game{}` and get
// an object whose managed state was never allocated. CNA-Go's settled answer
// at a public entry point that takes such a Game is an error result: Game.Run
// and Game.Exit already report exactly this. That guard is the only failure
// GameBaseLoadContent, GameBaseUpdate, GameBaseDraw and GameBaseUnloadContent
// have; GameBaseInitialize additionally propagates a component's own
// Initialize failure, which is real reference behavior.

// errGameNotConstructed is the guard shared by the five base-call helpers. It
// is the same condition Game.Run and Game.Exit already report, phrased the same
// way, and it has no CLR counterpart: it exists because Go can produce a Game
// value whose constructor never ran.
var errGameNotConstructed = errors.New("Game is nil or uninitialized")

// gameManagedState reports whether the Game carries the managed state its
// constructor allocates. A Game from NewGame always does.
func gameManagedState(game *Game) bool {
	return game != nil && game.gameComponents != nil && game.gameServices != nil
}

// GameBaseInitialize runs the base body of Microsoft.Xna.Framework.Game's
// protected virtual Initialize, which is the pending-component drain:
//
//	protected virtual void Initialize()
//	{
//	    this.HookDeviceEvents();
//	    while (this.notYetInitialized.Count > 0)
//	    {
//	        this.notYetInitialized[0].Initialize();
//	        this.notYetInitialized.RemoveAt(0);
//	    }
//	    if (this.graphicsDeviceService != null &&
//	        this.graphicsDeviceService.GraphicsDevice != null)
//	        this.LoadContent();
//	}
//
// # The drain
//
// It always takes index 0, initializes, and only THEN removes, so a component
// whose Initialize fails stays at the head of the queue and the drain stops
// where it failed. It also re-reads Count every iteration, so a component that
// adds another component from inside its own Initialize extends the same drain
// and the new component is initialized before this call returns -- which is
// consistent, because `inRun` is still false at this point and the add handler
// therefore queues rather than initializes.
//
// # Two reference steps are deferred, and neither is faked
//
// HookDeviceEvents resolves
// Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService out of Services,
// stores it, and if it is non-null subscribes four device handlers to it. It
// is not reproduced here, for two independent reasons:
//
//   - The contract lives in the GRAPHICS package, which imports this one. The
//     settled cross-package rule already projects Game's device-typed members
//     into the descendant package for exactly this reason; the framework
//     package cannot name the type.
//   - Nothing in CNA-Go can publish that service. The reference's own
//     registrar is GraphicsDeviceManager, and CNA-Go's GraphicsDeviceManager
//     is a partial native-backed facade that satisfies neither
//     IGraphicsDeviceService nor IGraphicsDeviceManager, so
//     Services.GetService would find nothing to store even if the type were
//     nameable.
//
// The conditional LoadContent call is deferred with it, because its condition
// is the field HookDeviceEvents assigns: with no device service, the reference
// itself does not call LoadContent from Initialize either. CNA-Go's
// LoadContent arrives instead from the native CNA `load_content` callback,
// whose documented order is `initialize`, then the runtime's own component and
// device setup, then `load_content` -- so the observable "content loads after
// initialization, once a device exists" ordering is supplied by the native
// host rather than invented in Go.
//
// Neither deferral is observable from the managed component part. Both steps
// follow or precede the drain without touching the queue, the collection, or
// either derived list, and CNA-Go exposes no way to count an event's
// subscribers, so no component's Initialize can tell whether they ran.
func GameBaseInitialize(game *Game) error {
	if !gameManagedState(game) {
		return errGameNotConstructed
	}
	for len(game.notYetInitialized) > 0 {
		if err := game.notYetInitialized[0].Initialize(); err != nil {
			return err
		}
		game.notYetInitialized = removeAt(game.notYetInitialized, 0)
	}
	return nil
}

// GameBaseLoadContent runs the base body of Game's protected virtual
// LoadContent, which is
//
//	IL_0000: ret            // code size 1
//
// a true authoritative no-op. The reference declares the virtual so a derived
// class has something to override; the base itself does nothing at all, and
// this faithfully does nothing at all.
func GameBaseLoadContent(game *Game) error {
	if !gameManagedState(game) {
		return errGameNotConstructed
	}
	return nil
}

// GameBaseUnloadContent runs the base body of Game's protected virtual
// UnloadContent, which is also a bare `ret` of code size 1 and is therefore
// also faithfully a no-op.
func GameBaseUnloadContent(game *Game) error {
	if !gameManagedState(game) {
		return errGameNotConstructed
	}
	return nil
}

// GameBaseUpdate runs the base body of Game's protected virtual Update, which
// is the component update loop:
//
//	protected virtual void Update(GameTime gameTime)
//	{
//	    Logger.BeginLogEvent(LoggingEvent.Update, "");
//	    for (int i = 0; i < this.updateableComponents.Count; i++)
//	        this.currentlyUpdatingComponents.Add(this.updateableComponents[i]);
//	    for (int j = 0; j < this.currentlyUpdatingComponents.Count; j++)
//	    {
//	        IUpdateable u = this.currentlyUpdatingComponents[j];
//	        if (u.Enabled) u.Update(gameTime);
//	    }
//	    this.currentlyUpdatingComponents.Clear();
//	    FrameworkDispatcher.Update();
//	    this.doneFirstUpdate = true;
//	    Logger.EndLogEvent(LoggingEvent.Update, "");
//	}
//
// # The snapshot, and what it does and does not freeze
//
// The first loop copies the ordered list into a second list and the second
// loop iterates the COPY, so adding or removing components during the frame
// does not change which components this frame updates. It is not a full
// freeze: `Enabled` is read at iteration time, not at snapshot time, so a
// component disabled earlier in the same frame is skipped when its turn comes.
//
// # No try/finally
//
// The reference has no exception handler, so a component that throws leaves
// currentlyUpdatingComponents populated and the next frame's first loop
// APPENDS to it. The Go projection reproduces the structure exactly -- a
// straight-line Clear, never a deferred one -- so a component that panics
// leaves exactly the same debris. It cannot be reached through the error
// channel, because IUpdateable.Update is infallible: the one shipped
// implementor's Update is a bare `ret` of code size 1, which is why
// Foundation 23 classified the contract infallible.
//
// # Two deferred reference steps
//
// FrameworkDispatcher.Update() pumps the media and audio subsystems, which
// CNA-Go does not have: there is no media backend and no audio backend, so
// there is nothing to pump and nothing is invented in its place.
// Logger.BeginLogEvent/EndLogEvent are XNA's private profiling channel, are
// not public surface, and produce no observable effect.
//
// doneFirstUpdate IS assigned, because it is Game's own managed state. Its
// only readers in the reference -- Paint, Tick and DrawFrame -- are not yet
// projected, so nothing observes it today.
func GameBaseUpdate(game *Game, gameTime GameTime) error {
	if !gameManagedState(game) {
		return errGameNotConstructed
	}
	for i := 0; i < len(game.updateableComponents); i++ {
		game.currentlyUpdatingComponents = append(game.currentlyUpdatingComponents, game.updateableComponents[i].component)
	}
	for j := 0; j < len(game.currentlyUpdatingComponents); j++ {
		updateable := game.currentlyUpdatingComponents[j]
		if updateable.Enabled() {
			updateable.Update(gameTime)
		}
	}
	game.currentlyUpdatingComponents = game.currentlyUpdatingComponents[:0]
	game.doneFirstUpdate = true
	return nil
}

// GameBaseDraw runs the base body of Game's protected virtual Draw, which is
// the same shape as base Update over IDrawable and Visible:
//
//	protected virtual void Draw(GameTime gameTime)
//	{
//	    for (int i = 0; i < this.drawableComponents.Count; i++)
//	        this.currentlyDrawingComponents.Add(this.drawableComponents[i]);
//	    for (int j = 0; j < this.currentlyDrawingComponents.Count; j++)
//	    {
//	        IDrawable d = this.currentlyDrawingComponents[j];
//	        if (d.Visible) d.Draw(gameTime);
//	    }
//	    this.currentlyDrawingComponents.Clear();
//	}
//
// It is 107 bytes of IL and there is nothing else in it. In particular it
// **touches no device**: base Draw does not clear, present, or reach
// GraphicsDevice at all. The device work lives in Game::BeginDraw and
// Game::EndDraw, which delegate to IGraphicsDeviceManager and are separate
// protected virtuals that CNA-Go does not project. Each IDrawable is
// responsible for its own drawing.
//
// It also has no logging and no dispatcher call -- those belong to BeginDraw
// and EndDraw -- and it assigns no field, so unlike base Update it has no
// deferred step at all.
func GameBaseDraw(game *Game, gameTime GameTime) error {
	if !gameManagedState(game) {
		return errGameNotConstructed
	}
	for i := 0; i < len(game.drawableComponents); i++ {
		game.currentlyDrawingComponents = append(game.currentlyDrawingComponents, game.drawableComponents[i].component)
	}
	for j := 0; j < len(game.currentlyDrawingComponents); j++ {
		drawable := game.currentlyDrawingComponents[j]
		if drawable.Visible() {
			drawable.Draw(gameTime)
		}
	}
	game.currentlyDrawingComponents = game.currentlyDrawingComponents[:0]
	return nil
}
