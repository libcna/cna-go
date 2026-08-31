package framework

import "github.com/openeggbert/cna-go/internal/interop"

// This file projects the four CLR events Microsoft.Xna.Framework.Game declares,
// and the three protected virtual raise sites that go with them.
//
//	.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Activated
//	.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Deactivated
//	.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Exiting
//	.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Disposed
//
// All four are System.EventHandler`1<System.EventArgs>, so all four take the
// settled two-accessor projection with EventHandler[*EventArgs] and
// EventSubscription. Every add_/remove_ accessor is the ordinary compiler-
// generated Delegate.Combine/Delegate.Remove pair with an Interlocked
// compare-exchange loop, so there is no per-event behavior in the accessors
// themselves.
//
// # Three of them are raised by the host, and the host is native
//
// The reference's Game subscribes six of its own private methods to GameHost in
// EnsureHost, which the constructor calls before it even allocates
// gameComponents:
//
//	host.Activated   += HostActivated
//	host.Deactivated += HostDeactivated
//	host.Exiting     += HostExiting
//	(plus Suspend, Resume and Idle)
//
// GameHost is exactly the role the native CNA runtime plays in CNA-Go, and CNA
// publishes those same three signals plus a disposal signal as canonical,
// already-shipped C API surface. So the projection binds the existing signals
// rather than inventing a Go-side raise:
//
//	CNA_GAME_EVENT_ACTIVATED   -> OnActivated(game, EventArgs.Empty)
//	CNA_GAME_EVENT_DEACTIVATED -> OnDeactivated(game, EventArgs.Empty)
//	CNA_GAME_EVENT_EXITING     -> OnExiting(game, EventArgs.Empty)
//
// All three go through the projected protected virtual, because that is what
// HostActivated, HostDeactivated and HostExiting do -- each is a `callvirt` to
// the On... method, not a direct delegate invoke.
//
// # The fourth signal is deliberately NOT an event raise
//
//	CNA_GAME_EVENT_DISPOSED    -> (nothing public)
//
// Game::Disposed is the one event of the four the host does not raise. Its only
// raise site in the whole class is the tail of Dispose(bool), it has no On...
// method, and a consumer who never disposes never sees it. CNA raises its own
// disposal signal from native game destruction at the end of the runtime host's
// lifetime, which is a different observable moment: it fires whether or not
// anyone disposed the Game, and it does not fire when a consumer disposes
// without ever running.
//
// Foundation 34 bound the two together because Dispose was not projected and
// the native signal was the only disposal fact CNA-Go had. Foundation 39
// projects Dispose and corrects the binding: the public event now follows the
// reference's managed raise site, and the native signal stays bound and
// measured as a CNA LIFECYCLE signal only -- it is what proves the subscription
// outlives cna_game_destroy, and it raises nothing public.
//
// # Where CNA-Go subscribes, and why it is eager
//
// The reference subscribes at construction because that is where the host is
// created. CNA-Go subscribes where the native host is created, which is inside
// Run, immediately after cna_game_create returns. It is not deferred until the
// first Go handler arrives: CNA enforces owner-thread affinity on
// cna_game_subscribe itself and answers CNA_RESULT_THREAD for a call from any
// other thread, while a Go consumer may add a handler from any goroutine at any
// time. Eager installation on the owner thread is therefore the only point at
// which the native call is guaranteed to be legal.
//
// # Exactly one native subscription per event, never one per handler
//
// A Go consumer's handlers live in an EventSource, and the bridge raises that
// source once per native signal. Registering one native callback per Go handler
// was measured and rejected: CNA invokes multiple registrations on one event in
// REVERSE registration order, which would silently invert the dispatch order
// the event projection promises. One native subscription per event makes the
// question moot -- ordering is decided entirely by EventSource, in registration
// order, exactly as a CLR multicast invocation list runs.
//
// It also makes Add and Remove ordinary Go calls with no thread affinity: they
// touch a mutex-guarded list and never reach C, so a consumer may subscribe and
// unsubscribe from any goroutine even though the native API underneath could
// not.
//
// # No native identity is reachable from here
//
// CNA_GameEventRegistrationHandle, the cgo.Handle, the callback context pointer
// and the CNA_GAME_EVENT_* constants all stay inside internal/interop. The
// public surface is EventHandler[*EventArgs] and EventSubscription, the same
// two names every other projected XNA event uses.

// AddActivatedHandler registers a handler for Game::Activated.
//
// The event fires when the game becomes the active application. The reference
// raise path is
//
//	HostActivated(sender, e)
//	  if (isActive) return;              // edge-triggered
//	  isActive = true;                   // state FIRST
//	  OnActivated(this, EventArgs.Empty) // then announce
//
// so a handler always observes a game that has already recorded itself active.
func (g *Game) AddActivatedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return g.activated.Add(handler)
}

// RemoveActivatedHandler removes the registration the token names and leaves
// every other registration in place. The native subscription is unaffected:
// it is per Game, not per handler, and outlives every Add and Remove.
func (g *Game) RemoveActivatedHandler(subscription EventSubscription) error {
	return g.activated.Remove(subscription)
}

// AddDeactivatedHandler registers a handler for Game::Deactivated, whose raise
// path is HostDeactivated and is the exact mirror of HostActivated: it returns
// immediately unless the game is currently active, lowers the flag, and then
// announces.
func (g *Game) AddDeactivatedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return g.deactivated.Add(handler)
}

// RemoveDeactivatedHandler removes the registration the token names.
func (g *Game) RemoveDeactivatedHandler(subscription EventSubscription) error {
	return g.deactivated.Remove(subscription)
}

// AddExitingHandler registers a handler for Game::Exiting.
//
// Note the sender a handler receives. OnExiting raises with a NULL sender --
// `ldnull`, not `ldarg.0` -- which is the one place in this family where the
// reference does not pass the Game. That is preserved: an Exiting handler is
// called with a nil sender.
func (g *Game) AddExitingHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return g.exiting.Add(handler)
}

// RemoveExitingHandler removes the registration the token names.
func (g *Game) RemoveExitingHandler(subscription EventSubscription) error {
	return g.exiting.Remove(subscription)
}

// AddDisposedHandler registers a handler for Game::Disposed.
//
// Disposed is the one event of the four with no protected raise method: the
// reference invokes the delegate field directly from the end of Dispose(bool),
// with `this` as the sender, and CNA-Go raises it from exactly there.
//
// It therefore fires when a consumer disposes the Game and at no other time. A
// consumer who never calls Dispose never sees it, a consumer who disposes twice
// sees it twice, and ending a Run does not raise it -- all three are the
// reference's behaviour, and none of them is what the native disposal signal
// does. See game_disposal.go for the correction and its evidence.
func (g *Game) AddDisposedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return g.disposed.Add(handler)
}

// RemoveDisposedHandler removes the registration the token names.
func (g *Game) RemoveDisposedHandler(subscription EventSubscription) error {
	return g.disposed.Remove(subscription)
}

// OnActivated is Game::OnActivated, the protected virtual raise site:
//
//	OnActivated(object sender, EventArgs args)
//	  ldarg.0; ldfld Activated; brfalse.s RET
//	  ldarg.0; ldfld Activated
//	  ldarg.0            // <- `this`, NOT the sender parameter
//	  ldarg.2            // the args, forwarded unchanged
//	  callvirt EventHandler`1::Invoke(object, EventArgs)
//
// The sender argument is accepted, ignored, and replaced by the Game itself,
// the same shape GameComponent::OnEnabledChanged has. Both parameters are still
// projected because the pinned contract declares them.
//
// It is fallible because raising runs consumer handlers and a handler failure
// is not discarded; a Game with no Activated registrations cannot fail.
func (g *Game) OnActivated(sender any, args *EventArgs) error {
	return g.activated.Raise(g, args)
}

// OnDeactivated is Game::OnDeactivated, byte-for-byte the same shape over the
// Deactivated field, and it ignores its sender the same way.
func (g *Game) OnDeactivated(sender any, args *EventArgs) error {
	return g.deactivated.Raise(g, args)
}

// OnExiting is Game::OnExiting, and it is NOT the same shape as the two above:
//
//	OnExiting(object sender, EventArgs args)
//	  ldarg.0; ldfld Exiting; brfalse.s RET
//	  ldarg.0; ldfld Exiting
//	  ldnull             // <- NULL sender, not `this` and not the parameter
//	  ldarg.2
//	  callvirt EventHandler`1::Invoke(object, EventArgs)
//
// One IL instruction is the whole difference and it is observable, so it is
// reproduced exactly: an Exiting handler receives a nil sender. Nothing in the
// reference ever passes a non-null sender to this event.
func (g *Game) OnExiting(sender any, args *EventArgs) error {
	return g.exiting.Raise(nil, args)
}

// raiseNativeGameEvent is the private end of the native bridge. It is not a
// projected member and is not reachable from outside this package: the
// interop.Callbacks implementation in game.go is its only caller.
//
// Each identity is routed to the reference's own raise path for that event, and
// the two activation signals go through the edge-trigger guard the reference's
// host handlers apply. That guard is deliberate rather than defensive: CLR
// raises Activated only on a false-to-true transition, so a runtime that
// repeated a signal would still produce exactly one CLR-shaped raise here.
func (g *Game) raiseNativeGameEvent(event uint32) error {
	if g == nil {
		return nil
	}
	switch event {
	case interop.GameEventActivated:
		// HostActivated: if (isActive) return; isActive = true; OnActivated(...)
		if g.isActive {
			return nil
		}
		g.isActive = true
		return g.OnActivated(g, EventArgsEmpty())
	case interop.GameEventDeactivated:
		// HostDeactivated: if (!isActive) return; isActive = false; OnDeactivated(...)
		if !g.isActive {
			return nil
		}
		g.isActive = false
		return g.OnDeactivated(g, EventArgsEmpty())
	case interop.GameEventExiting:
		// HostExiting: OnExiting(this, EventArgs.Empty) -- unguarded, and the
		// On... method drops the sender for a null one.
		return g.OnExiting(g, EventArgsEmpty())
	case interop.GameEventDisposed:
		// Deliberately raises nothing. CNA delivers this from inside
		// cna_game_destroy; Game::Disposed is raised from Dispose(bool) and
		// from nowhere else, so driving the public event from here would put
		// the raise at a moment the reference has no raise at. The signal is
		// still bound, still delivered and still counted -- internal/interop
		// records the delivery, which is what proves the registrations outlive
		// native destruction -- and the public event is left to Dispose.
		return nil
	default:
		return nil
	}
}
