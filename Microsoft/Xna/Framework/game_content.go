package framework

import "github.com/openeggbert/cna-go/internal/servicebridge"

// ---------------------------------------------------------------------------
// Foundation 63 — Game::Content, the framework half.
// ---------------------------------------------------------------------------
//
// The PROJECTION of Game::get_Content and Game::set_Content lives in the
// Content package, because the settled cross-package cycle rule places an
// ancestor-namespace member whose type is a descendant-namespace type in the
// descendant package. The FIELD lives here, on Game, because that is where the
// reference keeps it and because its lifetime must be the Game's.
//
// These two closures are the whole of the connection. Neither names
// ContentManager, neither retains anything, and nothing here is exported: the
// Content package reaches them through internal/servicebridge, which is the
// same mechanism DrawableGameComponent's device-service resolution already
// uses, and which exists precisely so that a cross-package member needs no
// public API to be projectable.

func init() {
	servicebridge.SetGameContentAccessors(
		func(game any) (any, bool) {
			typed, ok := game.(*Game)
			if !ok || typed == nil {
				return nil, false
			}
			return typed.content, true
		},
		func(game any, value any) bool {
			typed, ok := game.(*Game)
			if !ok || typed == nil {
				return false
			}
			typed.content = value
			return true
		},
	)
}
