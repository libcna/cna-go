package framework

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// These tests drive the optional per-hook override mechanism through the exact
// private boundary the native hooks reach, gameRuntimeCallbacks, so what they
// prove is what a real frame delivers rather than a rehearsal of it.

// bareCallbacks is a five-member GameCallbacks implementation with no optional
// override at all. It is the shape every consumer written before this
// mechanism existed already has.
type bareCallbacks struct{ initialized int }

func (c *bareCallbacks) Initialize(*Game) error       { c.initialized++; return nil }
func (c *bareCallbacks) LoadContent(*Game) error      { return nil }
func (c *bareCallbacks) Update(*Game, GameTime) error { return nil }
func (c *bareCallbacks) Draw(*Game, GameTime) error   { return nil }
func (c *bareCallbacks) UnloadContent(*Game) error    { return nil }

// beginDrawOnlyCallbacks declares exactly one optional override. Its BeginDraw
// calls the base explicitly, which is the Go projection of base.BeginDraw().
type beginDrawOnlyCallbacks struct {
	bareCallbacks
	calls      int
	baseCalls  int
	baseAnswer bool
	baseError  error
	refuse     bool
}

func (c *beginDrawOnlyCallbacks) BeginDraw(game *Game) (bool, error) {
	c.calls++
	answer, err := game.BeginDraw()
	c.baseCalls++
	c.baseAnswer, c.baseError = answer, err
	if c.refuse {
		return false, nil
	}
	return answer, err
}

// allFourCallbacks declares every optional override and records the order they
// arrive in.
type allFourCallbacks struct {
	bareCallbacks
	order []string
}

func (c *allFourCallbacks) BeginRun(*Game) error { c.order = append(c.order, "BeginRun"); return nil }
func (c *allFourCallbacks) EndRun(*Game) error   { c.order = append(c.order, "EndRun"); return nil }
func (c *allFourCallbacks) EndDraw(*Game) error  { c.order = append(c.order, "EndDraw"); return nil }
func (c *allFourCallbacks) BeginDraw(*Game) (bool, error) {
	c.order = append(c.order, "BeginDraw")
	return true, nil
}

// endDrawOnlyCallbacks declares a single override that is NOT the first one, so
// the four capabilities are proved independent rather than ordered.
type endDrawOnlyCallbacks struct {
	bareCallbacks
	calls int
}

func (c *endDrawOnlyCallbacks) EndDraw(*Game) error { c.calls++; return nil }

// TestNoOverrideInstallsNoHook is the compatibility claim. A callback object
// written before this mechanism existed reports an empty mask, so every
// optional CNA_GameFrameHooks member stays NULL and the native frame positions
// behave exactly as they did.
func TestNoOverrideInstallsNoHook(t *testing.T) {
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	bridge := gameRuntimeCallbacks{game: game}
	if mask := bridge.FrameHookOverrides(); mask != 0 {
		t.Fatalf("a five-member callback object reported mask %#x, want 0", mask)
	}
	if game.beginRunOverride != nil || game.endRunOverride != nil ||
		game.beginDrawOverride != nil || game.endDrawOverride != nil {
		t.Fatal("a five-member callback object was captured as supplying an override")
	}
}

// TestOneOverrideInstallsExactlyThatHook proves the capabilities are
// independent: declaring BeginDraw installs begin_draw and nothing else.
func TestOneOverrideInstallsExactlyThatHook(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		callbacks GameCallbacks
		want      interop.FrameHookMask
	}{
		{"begin draw only", &beginDrawOnlyCallbacks{}, interop.FrameHookBeginDraw},
		{"end draw only", &endDrawOnlyCallbacks{}, interop.FrameHookEndDraw},
		{"all four", &allFourCallbacks{}, interop.FrameHookBeginRun | interop.FrameHookEndRun | interop.FrameHookBeginDraw | interop.FrameHookEndDraw},
		{"none", &bareCallbacks{}, 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			game, err := NewGame(testCase.callbacks)
			if err != nil {
				t.Fatalf("NewGame: %v", err)
			}
			if mask := (gameRuntimeCallbacks{game: game}).FrameHookOverrides(); mask != testCase.want {
				t.Fatalf("mask = %#x, want %#x", mask, testCase.want)
			}
		})
	}
}

// TestAnOmittedCapabilityNeverReceivesItsHook is the other half of the same
// claim, seen from the dispatch side. A hook that was never installed cannot
// arrive; if the bridge is asked for one anyway it reports rather than quietly
// running the base at a position CNA-Go picked.
func TestAnOmittedCapabilityNeverReceivesItsHook(t *testing.T) {
	game, err := NewGame(&endDrawOnlyCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	bridge := gameRuntimeCallbacks{game: game}
	if err := bridge.BeginRun(); !errors.Is(err, errFrameHookWithoutOverride) {
		t.Fatalf("BeginRun with no override = %v", err)
	}
	if err := bridge.EndRun(); !errors.Is(err, errFrameHookWithoutOverride) {
		t.Fatalf("EndRun with no override = %v", err)
	}
	shouldDraw, err := bridge.BeginDraw()
	if !errors.Is(err, errFrameHookWithoutOverride) {
		t.Fatalf("BeginDraw with no override = %v", err)
	}
	if shouldDraw {
		t.Fatal("a refused BeginDraw dispatch also admitted the frame")
	}
	if err := bridge.EndDraw(); err != nil {
		t.Fatalf("EndDraw with an override = %v", err)
	}
}

// TestExplicitBaseCallDoesNotRedispatch is the recursion proof.
//
// Inside an override, game.BeginDraw() is the Go projection of
// base.BeginDraw(). Game.BeginDraw reads only Game's own state and never
// consults the callback object, so the override cannot re-enter itself. If it
// could, the counter below would exceed one and this test would not return.
func TestExplicitBaseCallDoesNotRedispatch(t *testing.T) {
	callbacks := &beginDrawOnlyCallbacks{}
	game, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	shouldDraw, err := (gameRuntimeCallbacks{game: game}).BeginDraw()
	if err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if !shouldDraw {
		t.Fatal("the base admitted the frame but the override reported false")
	}
	if callbacks.calls != 1 {
		t.Fatalf("the override ran %d times for one hook delivery; an explicit base call redispatched", callbacks.calls)
	}
	if callbacks.baseCalls != 1 || !callbacks.baseAnswer || callbacks.baseError != nil {
		t.Fatalf("base call: ran=%d answer=%t err=%v", callbacks.baseCalls, callbacks.baseAnswer, callbacks.baseError)
	}
}

// TestBaseIsRunZeroOnceOrTwiceExactlyAsWritten holds the call-site rule. The
// derived override controls base calls exactly as CLR does: not calling it does
// not run it, calling it twice runs it twice, and nothing deduplicates or
// suppresses a repeated explicit call.
func TestBaseIsRunZeroOnceOrTwiceExactlyAsWritten(t *testing.T) {
	for _, want := range []int{0, 1, 2, 3} {
		counted := &countingBaseCallbacks{baseCalls: want}
		game, err := NewGame(counted)
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}
		if _, err := (gameRuntimeCallbacks{game: game}).BeginDraw(); err != nil {
			t.Fatalf("BeginDraw: %v", err)
		}
		if counted.ran != want {
			t.Fatalf("an override that calls the base %d times ran it %d times", want, counted.ran)
		}
	}
}

type countingBaseCallbacks struct {
	bareCallbacks
	baseCalls int
	ran       int
}

func (c *countingBaseCallbacks) BeginDraw(game *Game) (bool, error) {
	for i := 0; i < c.baseCalls; i++ {
		if _, err := game.BeginDraw(); err != nil {
			return false, err
		}
		c.ran++
	}
	return true, nil
}

// TestABeginDrawOverrideRefusalIsNotAnError keeps the Boolean and the error
// apart at the projection boundary, which is where the native runtime reads
// them: a refusal must reach CNA as (CNA_FALSE, success), never as a failure.
func TestABeginDrawOverrideRefusalIsNotAnError(t *testing.T) {
	callbacks := &beginDrawOnlyCallbacks{refuse: true}
	game, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	shouldDraw, err := (gameRuntimeCallbacks{game: game}).BeginDraw()
	if err != nil {
		t.Fatalf("a refusal was reported as a failure: %v", err)
	}
	if shouldDraw {
		t.Fatal("the override refused the frame and the bridge admitted it")
	}
	// The base still answered true; the override's answer is what reaches CNA.
	if !callbacks.baseAnswer {
		t.Fatal("the base refused the frame, which it cannot do with no manager registered")
	}
}

// TestAnOverrideFailureSurfacesUnchanged proves the failure channel is the
// established one and that a failing override does not also decide the frame.
func TestAnOverrideFailureSurfacesUnchanged(t *testing.T) {
	sentinel := errors.New("override failure")
	game, err := NewGame(&failingOverrideCallbacks{failure: sentinel})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	bridge := gameRuntimeCallbacks{game: game}
	if err := bridge.BeginRun(); !errors.Is(err, sentinel) {
		t.Fatalf("BeginRun failure = %v", err)
	}
	if err := bridge.EndRun(); !errors.Is(err, sentinel) {
		t.Fatalf("EndRun failure = %v", err)
	}
	if err := bridge.EndDraw(); !errors.Is(err, sentinel) {
		t.Fatalf("EndDraw failure = %v", err)
	}
	if _, err := bridge.BeginDraw(); !errors.Is(err, sentinel) {
		t.Fatalf("BeginDraw failure = %v", err)
	}
}

type failingOverrideCallbacks struct {
	bareCallbacks
	failure error
}

func (c *failingOverrideCallbacks) BeginRun(*Game) error { return c.failure }
func (c *failingOverrideCallbacks) EndRun(*Game) error   { return c.failure }
func (c *failingOverrideCallbacks) EndDraw(*Game) error  { return c.failure }
func (c *failingOverrideCallbacks) BeginDraw(*Game) (bool, error) {
	return true, c.failure
}

// TestTheCapabilitiesAreFourPrivateSingleMethodContracts pins the shape of the
// mechanism itself: four distinct unexported identities, one method each, and
// the exact signature a consumer has to declare to satisfy one.
func TestTheCapabilitiesAreFourPrivateSingleMethodContracts(t *testing.T) {
	for name, want := range map[string]struct {
		contract  reflect.Type
		method    string
		signature string
	}{
		"gameBeginRunOverride":  {reflect.TypeOf((*gameBeginRunOverride)(nil)).Elem(), "BeginRun", "func(*framework.Game) error"},
		"gameEndRunOverride":    {reflect.TypeOf((*gameEndRunOverride)(nil)).Elem(), "EndRun", "func(*framework.Game) error"},
		"gameBeginDrawOverride": {reflect.TypeOf((*gameBeginDrawOverride)(nil)).Elem(), "BeginDraw", "func(*framework.Game) (bool, error)"},
		"gameEndDrawOverride":   {reflect.TypeOf((*gameEndDrawOverride)(nil)).Elem(), "EndDraw", "func(*framework.Game) error"},
	} {
		if want.contract.NumMethod() != 1 {
			t.Fatalf("%s declares %d methods; each capability is exactly one", name, want.contract.NumMethod())
		}
		method := want.contract.Method(0)
		if method.Name != want.method {
			t.Fatalf("%s declares %q, want %q", name, method.Name, want.method)
		}
		if got := method.Type.String(); got != want.signature {
			t.Fatalf("%s.%s has signature %s, want %s", name, method.Name, got, want.signature)
		}
		if !strings.HasPrefix(name, "game") || strings.ToUpper(name[:1]) == name[:1] {
			t.Fatalf("%s is not an unexported identity", name)
		}
	}
	// The four are distinct: satisfying one must not satisfy another.
	only := reflect.TypeOf(&endDrawOnlyCallbacks{})
	if only.Implements(reflect.TypeOf((*gameBeginDrawOverride)(nil)).Elem()) {
		t.Fatal("an EndDraw-only conformer also satisfied the BeginDraw capability")
	}
	if !only.Implements(reflect.TypeOf((*gameEndDrawOverride)(nil)).Elem()) {
		t.Fatal("an EndDraw conformer did not satisfy the EndDraw capability")
	}
}

// TestTheOverrideSetIsFixedAtConstruction records why there is no registration
// operation: the answer is decided once, from the object NewGame was handed,
// and a second Game built from the same object gets the same answer.
func TestTheOverrideSetIsFixedAtConstruction(t *testing.T) {
	callbacks := &allFourCallbacks{}
	first, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	second, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	firstMask := (gameRuntimeCallbacks{game: first}).FrameHookOverrides()
	secondMask := (gameRuntimeCallbacks{game: second}).FrameHookOverrides()
	if firstMask != secondMask {
		t.Fatalf("the same callback object produced masks %#x and %#x", firstMask, secondMask)
	}
	if firstMask != interop.FrameHookBeginRun|interop.FrameHookEndRun|interop.FrameHookBeginDraw|interop.FrameHookEndDraw {
		t.Fatalf("a four-override object reported mask %#x", firstMask)
	}
}

// TestOverridesReceiveTheOwningGame proves the Game a hook hands the override
// is the Game the hook belongs to, which is what makes an explicit base call
// reach the right object.
func TestOverridesReceiveTheOwningGame(t *testing.T) {
	callbacks := &identityCallbacks{}
	game, err := NewGame(callbacks)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	bridge := gameRuntimeCallbacks{game: game}
	if err := bridge.BeginRun(); err != nil {
		t.Fatalf("BeginRun: %v", err)
	}
	if _, err := bridge.BeginDraw(); err != nil {
		t.Fatalf("BeginDraw: %v", err)
	}
	if err := bridge.EndDraw(); err != nil {
		t.Fatalf("EndDraw: %v", err)
	}
	if err := bridge.EndRun(); err != nil {
		t.Fatalf("EndRun: %v", err)
	}
	if len(callbacks.seen) != 4 {
		t.Fatalf("saw %d hooks, want 4", len(callbacks.seen))
	}
	for i, seen := range callbacks.seen {
		if seen != game {
			t.Fatalf("hook %d received %p, want the owning Game %p", i, seen, game)
		}
	}
}

type identityCallbacks struct {
	bareCallbacks
	seen []*Game
}

func (c *identityCallbacks) BeginRun(game *Game) error { c.seen = append(c.seen, game); return nil }
func (c *identityCallbacks) EndRun(game *Game) error   { c.seen = append(c.seen, game); return nil }
func (c *identityCallbacks) EndDraw(game *Game) error  { c.seen = append(c.seen, game); return nil }
func (c *identityCallbacks) BeginDraw(game *Game) (bool, error) {
	c.seen = append(c.seen, game)
	return true, nil
}

// TestNoPublicRegistrationSurfaceExists holds the negative half of the design.
// There is no exported way to install, replace or remove an override, and no
// exported capability interface: the only way in is to declare the method.
func TestNoPublicRegistrationSurfaceExists(t *testing.T) {
	game := reflect.TypeOf((*Game)(nil))
	for i := 0; i < game.NumMethod(); i++ {
		name := game.Method(i).Name
		if strings.Contains(name, "Override") || strings.HasSuffix(name, "Hook") {
			t.Fatalf("Game exposes %q; the override set is fixed at construction and has no registration API", name)
		}
	}
	// GameCallbacks is still exactly the five. This duplicates the structural
	// test next door on purpose: it is the promise the whole mechanism was
	// shaped around, and it should fail here too if it is ever broken.
	if contract := reflect.TypeOf((*GameCallbacks)(nil)).Elem(); contract.NumMethod() != 5 {
		t.Fatalf("GameCallbacks has %d methods, want exactly 5", contract.NumMethod())
	}
}
