package framework

// This file is the OPTIONAL per-hook override mechanism for Game's four
// frame-boundary protected virtuals. It is CNA-Go language support, not XNA
// surface: nothing here is an XNA identity, nothing here is counted in
// REFERENCE_MEMBERS, and none of the four interfaces below is exported.
//
// # The problem it solves
//
// In CLR a derived Game may override ANY SUBSET of BeginRun, EndRun, BeginDraw
// and EndDraw. Foundation 35 projected all four as base bodies on Game and
// Foundation 36 measured the four canonical CNA hooks that sit at the same
// frame positions, but nothing could reach them: GameCallbacks is the language
// adapter for the FIVE protected virtuals it declares, and it keeps exactly
// those five.
//
// # The shape, and why it is this one
//
// Each hook gets its own single-method unexported interface. A callback object
// satisfies one simply by declaring the corresponding EXPORTED method with the
// measured signature; Go interfaces are structural, so a consumer never names
// -- and never needs access to -- the interface type itself.
//
//	func (c *Callbacks) BeginDraw(game *framework.Game) (bool, error) {
//	    // derived work before base
//	    return game.BeginDraw()
//	}
//
// Four independent capabilities preserve the actual override set. One bundled
// interface carrying all four would force a consumer who overrides only
// BeginDraw to write three no-op methods, and a no-op override is not the same
// thing as no override: it installs a native hook and takes the base's place.
//
// Everything this shape avoids is deliberate:
//
//   - GameCallbacks is untouched and still has exactly five members, so every
//     existing external implementation keeps compiling;
//   - no new EXPORTED framework interface identity is published;
//   - there is no registration or unregistration operation and no mutable
//     per-Game callback state, because a Go object's method set is fixed for
//     its whole lifetime and there is nothing to mutate;
//   - a hook with no override is not installed at all, so the native frame
//     position keeps the behaviour it had before this mechanism existed.
//
// # Base calls
//
// There is deliberately no GameBaseBeginRun, GameBaseEndRun, GameBaseBeginDraw
// or GameBaseEndDraw. The projected methods on Game ARE the base bodies, so
// calling game.BeginDraw() from inside an override is the Go projection of
// base.BeginDraw() -- and it is the whole projection: zero calls means the base
// does not run, one call runs it once at that source position, two calls run it
// twice. The call site is the semantics, exactly as it is in CLR.
//
// Those methods read Game's own state and never consult the callback object,
// so an explicit base call cannot re-enter the override. That is proved rather
// than asserted; see game_frame_hooks_test.go.
//
// # Discovery
//
// Discovery happens exactly once, in NewGame, at the same boundary where
// GameCallbacks itself becomes associated with the Game. The results are
// private fields. Nothing type-asserts per frame and nothing can change after
// construction.

// gameBeginRunOverride is satisfied by a callback object that declares
//
//	BeginRun(*Game) error
//
// It replaces Game::BeginRun at the native begin_run position, which the
// pinned runtime delivers once, after initialize and load_content and before
// the first update -- exactly where RunGame calls BeginRun.
type gameBeginRunOverride interface {
	BeginRun(*Game) error
}

// gameEndRunOverride is satisfied by a callback object that declares
//
//	EndRun(*Game) error
//
// It replaces Game::EndRun at the native end_run position, delivered once
// after the last frame and before cna_game_run returns -- where RunGame calls
// EndRun, immediately after the blocking host.Run() returns.
type gameEndRunOverride interface {
	EndRun(*Game) error
}

// gameBeginDrawOverride is satisfied by a callback object that declares
//
//	BeginDraw(*Game) (bool, error)
//
// The Boolean is the frame's drawing decision and stays a channel separate
// from the error, exactly as the base body's is:
//
//	(false, nil) -- skip this frame's Draw AND EndDraw
//	(true,  nil) -- proceed
//	(_,     err) -- the established callback-failure path; the runtime's own
//	                decision is left untouched and the failure surfaces from
//	                Game.Run
//
// A false answer is not an error and is never turned into one. Skipping EndDraw
// on a refused frame is the native runtime's own behaviour, measured against
// the pinned artifact, and it is the same shape as DrawFrame's
// `if (BeginDraw()) { Draw(); EndDraw(); }`.
type gameBeginDrawOverride interface {
	BeginDraw(*Game) (bool, error)
}

// gameEndDrawOverride is satisfied by a callback object that declares
//
//	EndDraw(*Game) error
//
// It replaces Game::EndDraw at the native end_draw position, which fires after
// each admitted draw and is skipped entirely on a frame begin_draw refused.
type gameEndDrawOverride interface {
	EndDraw(*Game) error
}

// captureFrameHookOverrides performs the one and only capability discovery, at
// the construction boundary. Storing the results means no frame pays for a
// type assertion, and -- more importantly -- the override set a Game runs with
// is decided once, from the object it was constructed with, and can never
// change underneath a running frame loop.
func (g *Game) captureFrameHookOverrides(callbacks GameCallbacks) {
	if override, ok := callbacks.(gameBeginRunOverride); ok {
		g.beginRunOverride = override
	}
	if override, ok := callbacks.(gameEndRunOverride); ok {
		g.endRunOverride = override
	}
	if override, ok := callbacks.(gameBeginDrawOverride); ok {
		g.beginDrawOverride = override
	}
	if override, ok := callbacks.(gameEndDrawOverride); ok {
		g.endDrawOverride = override
	}
}
