package content

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

type gameCallbacks struct{}

func (gameCallbacks) Initialize(*framework.Game) error                 { return nil }
func (gameCallbacks) LoadContent(*framework.Game) error                { return nil }
func (gameCallbacks) Update(*framework.Game, framework.GameTime) error { return nil }
func (gameCallbacks) Draw(*framework.Game, framework.GameTime) error   { return nil }
func (gameCallbacks) UnloadContent(*framework.Game) error              { return nil }

func newGame(t *testing.T) *framework.Game {
	t.Helper()
	game, err := framework.NewGame(gameCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

// TestGameContentExistsFromConstruction pins the constructor statement
//
//	this.content = new ContentManager(this.Services);
//
// A Game that has done nothing but be constructed already has its manager, and
// a consumer never sees nil from this member.
func TestGameContentExistsFromConstruction(t *testing.T) {
	game := newGame(t)
	manager := GameContent(game)
	if manager == nil {
		t.Fatal("GameContent on a freshly constructed Game returned nil; the constructor creates it")
	}
	provider, err := manager.ServiceProvider()
	if err != nil {
		t.Fatalf("ServiceProvider: %v", err)
	}
	if provider != any(game.Services()) {
		t.Fatal("the Game's ContentManager was built over some other service provider than the Game's own")
	}
}

// TestGameContentIsTheSameObjectEveryCall pins `ldfld`. A getter that built a
// fresh manager per call would pass every value test and break every consumer
// that assigned a RootDirectory through it.
func TestGameContentIsTheSameObjectEveryCall(t *testing.T) {
	game := newGame(t)
	first, second := GameContent(game), GameContent(game)
	if first == nil || first != second {
		t.Fatal("GameContent returned a different object on a second call; the reference reads one field")
	}
	if err := first.SetRootDirectory("Assets"); err != nil {
		t.Fatalf("SetRootDirectory: %v", err)
	}
	root, err := GameContent(game).RootDirectory()
	if err != nil {
		t.Fatalf("RootDirectory: %v", err)
	}
	if root != "Assets" {
		t.Fatalf("RootDirectory = %q through a second GameContent call, want the value assigned through the first", root)
	}
}

// TestEachGameHasItsOwnContentManager pins that the field is Game's rather than
// anything shared. A single package-level manager would satisfy the identity
// test above and be wrong for every program with two Games.
func TestEachGameHasItsOwnContentManager(t *testing.T) {
	first, second := newGame(t), newGame(t)
	if GameContent(first) == GameContent(second) {
		t.Fatal("two Games share one ContentManager; the reference's field is per-instance")
	}
}

// TestSetGameContentRefusesNil pins set_Content's whole failure surface:
//
//	if (value == null) throw new ArgumentNullException();
//
// and pins that the refusal leaves the existing manager in place, because the
// reference throws BEFORE its `stfld`.
func TestSetGameContentRefusesNil(t *testing.T) {
	game := newGame(t)
	before := GameContent(game)
	err := SetGameContent(game, nil)
	if err == nil {
		t.Fatal("SetGameContent(game, nil) reported no error")
	}
	if !errors.Is(err, errGameContentArgumentNull) {
		t.Fatalf("SetGameContent(game, nil) = %v, want the argument-null refusal", err)
	}
	if GameContent(game) != before {
		t.Fatal("the refused assignment still changed the field; the reference throws before its stfld")
	}
}

// TestSetGameContentReplacesTheManager pins the `stfld`, which is the only
// thing set_Content does after its guard.
func TestSetGameContentReplacesTheManager(t *testing.T) {
	game := newGame(t)
	replacement, err := NewContentManagerByIServiceProviderAndString(game.Services(), "Replaced")
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProviderAndString: %v", err)
	}
	if err := SetGameContent(game, replacement); err != nil {
		t.Fatalf("SetGameContent: %v", err)
	}
	if GameContent(game) != replacement {
		t.Fatal("GameContent did not answer the manager that was assigned")
	}
	root, err := GameContent(game).RootDirectory()
	if err != nil {
		t.Fatalf("RootDirectory: %v", err)
	}
	if root != "Replaced" {
		t.Fatalf("RootDirectory = %q, want the assigned manager's root", root)
	}
}

// TestGameContentOnANonGameAnswersNothing pins the Go-only guard. A zero Game
// is expressible in Go and is not a constructed one; it holds no manager, and
// an assignment to it is refused rather than silently kept somewhere else.
func TestGameContentOnANonGameAnswersNothing(t *testing.T) {
	if GameContent(nil) != nil {
		t.Fatal("GameContent(nil) returned a manager")
	}
	zero := &framework.Game{}
	if GameContent(zero) != nil {
		t.Fatal("GameContent on an unconstructed Game returned a manager")
	}
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	// The zero Game HAS the field, so the write succeeds and is observable --
	// the guard that refuses is the one for something that is not a Game at
	// all, which only the bridge can be handed.
	if err := SetGameContent(zero, manager); err != nil {
		t.Fatalf("SetGameContent on a zero Game: %v", err)
	}
	if GameContent(zero) != manager {
		t.Fatal("the assignment to a zero Game was not observable")
	}
	if err := SetGameContent(nil, manager); !errors.Is(err, errNotAGame) {
		t.Fatalf("SetGameContent(nil, manager) = %v, want the uninitialized-Game refusal", err)
	}
}
