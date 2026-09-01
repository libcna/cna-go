package content

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file projects Microsoft.Xna.Framework.Game::get_Content and
// Game::set_Content.
//
// Both live in the CONTENT package rather than on Game because of the settled
// cross-package cycle rule: an ancestor-namespace member whose type is a
// descendant-namespace type projects as a descendant-package function named
// OwnerTypeMember, with the receiver first. The Content package imports the
// framework package, so the framework package cannot name ContentManager; here,
// both are nameable.
//
// # The reference bodies, exactly
//
//	get_Content:  ldarg.0; ldfld ContentManager Game::content; ret
//
//	set_Content:  if (value == null) throw new ArgumentNullException();
//	              this.content = value;
//
// The getter is one field read and cannot fail. The setter has exactly one
// failure, and it is NOT the usual named-argument form: the throw site is
// `newobj ArgumentNullException::.ctor()` with no argument name at all.
//
// # Where the value comes from
//
// Game's constructor creates it -- `this.content = new ContentManager(this.Services)`
// -- and this package supplies that constructor through the same bridge, from
// the init below. So the manager exists from construction, its identity never
// changes unless a consumer assigns one, and the Game owns it: nothing here
// keeps a map of Games, and the field dies with the object that holds it.

func init() {
	// Game's constructor calls this with its own GameServiceContainer. The
	// one-argument ContentManager constructor is what the reference calls, and
	// it refuses a nil provider -- which cannot happen from here, because
	// NewGame allocates the container before this runs.
	servicebridge.SetGameContentCreator(func(services any) (any, error) {
		return NewContentManagerByIServiceProvider(services)
	})
}

// errGameContentArgumentNull projects the argument-less
// System.ArgumentNullException set_Content throws.
var errGameContentArgumentNull = errors.New("value must not be nil")

// errNotAGame is the Go-only guard every projected Game member carries: Go can
// produce a Game whose constructor never ran, and such an object has no field
// to read.
var errNotAGame = errors.New("Game is nil or uninitialized")

// GameContent is Game::get_Content.
//
// It answers the ContentManager Game's constructor created, or the one a
// consumer assigned, and it is the SAME object on every call -- the reference's
// `ldfld` cannot be anything else. A Game whose constructor never ran has no
// content and answers nil, which is what a Go zero Game genuinely holds.
func GameContent(game *framework.Game) *ContentManager {
	stored, ok := servicebridge.ReadGameContent(game)
	if !ok {
		return nil
	}
	manager, typed := stored.(*ContentManager)
	if !typed {
		return nil
	}
	return manager
}

// SetGameContent is Game::set_Content, whose whole body is the null guard and
// the store.
func SetGameContent(game *framework.Game, value *ContentManager) error {
	if value == nil {
		return fmt.Errorf("%w", errGameContentArgumentNull)
	}
	if !servicebridge.WriteGameContent(game, value) {
		return errNotAGame
	}
	return nil
}
