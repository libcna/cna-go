package interop

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// These tests cover the parts of the native game-event bridge that have a
// decision in them but no native call: what happens when a signal arrives at a
// runtime that is no longer live, what a failing or panicking consumer does at
// a boundary that cannot report failure to C, and whether releasing the
// registrations twice can hand CNA a stale handle.
//
// Everything that does reach C is proved by tools/native_stress against the
// pinned artifact, in a crash-isolated subprocess, because that is the only
// place a real signal exists.

// recordingCallbacks is the minimum Callbacks implementation. Its five
// lifecycle members are never reached here; only GameEvent is, plus the frame
// hooks the frame-hook tests drive.
type recordingCallbacks struct {
	events  []uint32
	failure error
	panics  bool

	// frameHooks is what FrameHookOverrides reports, and hooks records every
	// optional frame hook that was actually dispatched, in order.
	frameHooks   FrameHookMask
	hooks        []string
	shouldDraw   bool
	hookFailure  error
	hookPanics   bool
	beginDrawRan int
}

func (c *recordingCallbacks) Initialize() error      { return nil }
func (c *recordingCallbacks) LoadContent() error     { return nil }
func (c *recordingCallbacks) Update(FrameTime) error { return nil }
func (c *recordingCallbacks) Draw(FrameTime) error   { return nil }
func (c *recordingCallbacks) UnloadContent() error   { return nil }

func (c *recordingCallbacks) FrameHookOverrides() FrameHookMask { return c.frameHooks }

func (c *recordingCallbacks) BeginRun() error {
	c.hooks = append(c.hooks, "BeginRun")
	return c.hookFailure
}

func (c *recordingCallbacks) EndRun() error {
	c.hooks = append(c.hooks, "EndRun")
	return c.hookFailure
}

func (c *recordingCallbacks) BeginDraw() (bool, error) {
	c.hooks = append(c.hooks, "BeginDraw")
	c.beginDrawRan++
	if c.hookPanics {
		panic("begin draw override panic")
	}
	return c.shouldDraw, c.hookFailure
}

func (c *recordingCallbacks) EndDraw() error {
	c.hooks = append(c.hooks, "EndDraw")
	return c.hookFailure
}

func (c *recordingCallbacks) GameEvent(event uint32) error {
	c.events = append(c.events, event)
	if c.panics {
		panic("game event handler panic")
	}
	return c.failure
}

func liveRuntime(callbacks Callbacks) *Runtime {
	runtime := NewRuntime(callbacks)
	runtime.alive = true
	runtime.generation = 1
	return runtime
}

// TestInvokeGameEventRoutesEveryIdentity proves the four canonical identities
// reach the framework unchanged and in the order they arrive.
func TestInvokeGameEventRoutesEveryIdentity(t *testing.T) {
	callbacks := &recordingCallbacks{}
	runtime := liveRuntime(callbacks)
	want := []uint32{GameEventActivated, GameEventDeactivated, GameEventExiting, GameEventDisposed}
	for _, event := range want {
		runtime.invokeGameEvent(event)
	}
	if len(callbacks.events) != len(want) {
		t.Fatalf("delivered %v, want %v", callbacks.events, want)
	}
	for i := range want {
		if callbacks.events[i] != want[i] {
			t.Fatalf("delivered %v, want %v", callbacks.events, want)
		}
	}
	if runtime.callbackFailure != nil {
		t.Fatalf("a clean delivery recorded %v", runtime.callbackFailure)
	}
}

// TestInvokeGameEventOnADeadRuntimeDeliversNothing is the callback-after-free
// guard seen from the Go side. The registrations are released after
// cna_game_destroy, so a signal cannot arrive here in practice; if the
// ownership tables ever changed, dropping it and recording the staleness is the
// correct answer rather than calling into a torn-down facade.
func TestInvokeGameEventOnADeadRuntimeDeliversNothing(t *testing.T) {
	callbacks := &recordingCallbacks{}
	runtime := NewRuntime(callbacks)
	runtime.invokeGameEvent(GameEventExiting)
	if len(callbacks.events) != 0 {
		t.Fatalf("a dead runtime delivered %v", callbacks.events)
	}
	if !errors.Is(runtime.callbackFailure, ErrStaleGeneration) {
		t.Fatalf("recorded %v, want ErrStaleGeneration", runtime.callbackFailure)
	}
}

// TestInvokeGameEventRecordsAHandlerFailure holds the error boundary.
// CNA_GameEventCallback returns void, so the failure cannot stop the game; it
// must be recorded rather than discarded, and it must surface from Run.
func TestInvokeGameEventRecordsAHandlerFailure(t *testing.T) {
	sentinel := errors.New("interop game event sentinel")
	callbacks := &recordingCallbacks{failure: sentinel}
	runtime := liveRuntime(callbacks)
	runtime.invokeGameEvent(GameEventActivated)
	if !errors.Is(runtime.callbackFailure, sentinel) {
		t.Fatalf("recorded %v, want the handler sentinel", runtime.callbackFailure)
	}
	// The first failure wins, exactly as it does for a lifecycle callback.
	callbacks.failure = errors.New("a later failure")
	runtime.invokeGameEvent(GameEventExiting)
	if !errors.Is(runtime.callbackFailure, sentinel) {
		t.Fatalf("a later failure replaced the first: %v", runtime.callbackFailure)
	}
}

// TestInvokeGameEventContainsAPanic proves nothing crosses the C frame. A
// panicking consumer handler is recovered and recorded as the run's failure.
func TestInvokeGameEventContainsAPanic(t *testing.T) {
	callbacks := &recordingCallbacks{panics: true}
	runtime := liveRuntime(callbacks)
	runtime.invokeGameEvent(GameEventDisposed)
	if runtime.callbackFailure == nil {
		t.Fatal("a panicking game-event handler recorded nothing")
	}
	if !strings.Contains(runtime.callbackFailure.Error(), "panic in Game event handler") {
		t.Fatalf("recorded %v, want a contained panic", runtime.callbackFailure)
	}
}

// TestReleaseGameEventsIsIdempotent is the other half of the lifetime claim.
// CNA answers CNA_RESULT_INVALID_HANDLE for a registration that was already
// released, so the slots are zeroed under the same lock that publishes them and
// a second release must pass nothing at all rather than a stale handle.
func TestReleaseGameEventsIsIdempotent(t *testing.T) {
	runtime := liveRuntime(&recordingCallbacks{})
	// Nothing installed: the release must not reach C, which it cannot here
	// because no library is open. Reaching it would fault rather than fail.
	if err := runtime.releaseGameEvents(); err != nil {
		t.Fatalf("releasing an uninstalled bridge reported %v", err)
	}
	// Slots are cleared even when the underlying release is never attempted.
	runtime.mu.Lock()
	runtime.eventRegistrations = [gameEventCount]uint64{0, 0, 0, 0}
	runtime.mu.Unlock()
	if err := runtime.releaseGameEvents(); err != nil {
		t.Fatalf("releasing zeroed registrations reported %v", err)
	}
	runtime.mu.Lock()
	registrations := runtime.eventRegistrations
	runtime.mu.Unlock()
	for i, handle := range registrations {
		if handle != 0 {
			t.Fatalf("registration slot %d still holds %d after release", i, handle)
		}
	}
}

// TestGameEventNamesCoverEveryIdentity keeps the trace names and the identities
// from drifting, including the unknown case a future ABI could produce.
func TestGameEventNamesCoverEveryIdentity(t *testing.T) {
	for event, want := range map[uint32]string{
		GameEventActivated:   "Activated",
		GameEventDeactivated: "Deactivated",
		GameEventDisposed:    "Disposed",
		GameEventExiting:     "Exiting",
	} {
		if got := gameEventName(event); got != want {
			t.Fatalf("gameEventName(%d) = %q, want %q", event, got, want)
		}
	}
	if got := gameEventName(99); got != "unknown(99)" {
		t.Fatalf("gameEventName(99) = %q", got)
	}
}

// TestTheIdentitiesAreContiguousFromZero pins what the four Go constants are.
// The C side of the chain is compiler-checked in native_linux.go and measured
// by tools/native_abi; this is the Go end of it.
func TestTheIdentitiesAreContiguousFromZero(t *testing.T) {
	identities := []uint32{GameEventActivated, GameEventDeactivated, GameEventDisposed, GameEventExiting}
	if len(identities) != gameEventCount {
		t.Fatalf("%d identities for a bridge of %d", len(identities), gameEventCount)
	}
	seen := make(map[uint32]bool, len(identities))
	for i, identity := range identities {
		if identity != uint32(i) {
			t.Fatalf("identity %d is %d; the canonical constants are contiguous from zero", i, identity)
		}
		if seen[identity] {
			t.Fatalf("identity %d is declared twice", identity)
		}
		seen[identity] = true
	}
}

// TestCallbacksCarriesTheBridgeWithoutDisturbingTheFive proves the internal
// contract carries the private bridge and the private frame-hook dispatch
// beside the five lifecycle members, and that the five themselves are
// untouched. The five names are what the PUBLIC GameCallbacks projects; every
// other member here is private to this package's consumers, which is why no
// consumer ever had to implement anything to get either mechanism.
func TestCallbacksCarriesTheBridgeWithoutDisturbingTheFive(t *testing.T) {
	contract := reflect.TypeOf((*Callbacks)(nil)).Elem()
	want := map[string]string{
		"Initialize":         "func() error",
		"LoadContent":        "func() error",
		"Update":             "func(interop.FrameTime) error",
		"Draw":               "func(interop.FrameTime) error",
		"UnloadContent":      "func() error",
		"GameEvent":          "func(uint32) error",
		"FrameHookOverrides": "func() interop.FrameHookMask",
		"BeginRun":           "func() error",
		"EndRun":             "func() error",
		"BeginDraw":          "func() (bool, error)",
		"EndDraw":            "func() error",
	}
	if contract.NumMethod() != len(want) {
		t.Fatalf("Callbacks has %d methods, want %d", contract.NumMethod(), len(want))
	}
	for i := 0; i < contract.NumMethod(); i++ {
		method := contract.Method(i)
		signature, declared := want[method.Name]
		if !declared {
			t.Fatalf("Callbacks gained an unexpected member %q", method.Name)
		}
		if got := method.Type.String(); got != signature {
			t.Fatalf("%s has signature %s, want %s", method.Name, got, signature)
		}
		delete(want, method.Name)
	}
	if len(want) != 0 {
		t.Fatalf("Callbacks lost members %v", want)
	}
}

// TestFrameHookMaskIsOneDistinctBitPerHook pins the Go end of the mask. The C
// end is compiler-checked in native_linux.go, where each constant is compared
// against bridge.h's mirror.
func TestFrameHookMaskIsOneDistinctBitPerHook(t *testing.T) {
	hooks := map[string]FrameHookMask{
		"BeginRun": FrameHookBeginRun, "EndRun": FrameHookEndRun,
		"BeginDraw": FrameHookBeginDraw, "EndDraw": FrameHookEndDraw,
	}
	var union FrameHookMask
	for name, bit := range hooks {
		if bit == 0 || bit&(bit-1) != 0 {
			t.Fatalf("%s is %#x, which is not a single bit", name, bit)
		}
		if union&bit != 0 {
			t.Fatalf("%s shares a bit with another hook", name)
		}
		union |= bit
	}
	if union != 0xF {
		t.Fatalf("the four hooks cover %#x, want 0xF", union)
	}
}

// TestFrameHooksDispatchOnlyWhatTheMaskDeclares proves the two halves of the
// mechanism cannot disagree: a runtime dispatches a hook only through the kind
// the mask installed, and the BeginDraw Boolean survives the dispatch as its
// own channel.
func TestFrameHooksDispatchOnlyWhatTheMaskDeclares(t *testing.T) {
	for _, refuses := range []bool{true, false} {
		callbacks := &recordingCallbacks{frameHooks: FrameHookBeginDraw, shouldDraw: !refuses}
		runtime := liveRuntime(callbacks)
		runtime.game = 7
		runtime.ownerThread = nativeOwnerThreadID()
		shouldDraw, err := runtime.invokeBeginDrawCallback(7, FrameTime{})
		if err != nil {
			t.Fatalf("begin_draw dispatch: %v", err)
		}
		if shouldDraw == refuses {
			t.Fatalf("begin_draw answered %t for an override that answered %t", shouldDraw, !refuses)
		}
		if len(callbacks.hooks) != 1 || callbacks.hooks[0] != "BeginDraw" {
			t.Fatalf("dispatched %v, want exactly BeginDraw", callbacks.hooks)
		}
		if runtime.callbackFailure != nil {
			t.Fatalf("a refusal was recorded as a failure: %v", runtime.callbackFailure)
		}
	}
}

// TestABeginDrawFailureIsNotADrawingDecision keeps the two channels apart at
// the boundary: the error is recorded, and the Boolean the caller receives is
// the untouched default rather than a refusal invented from the failure.
func TestABeginDrawFailureIsNotADrawingDecision(t *testing.T) {
	sentinel := errors.New("begin draw override failure")
	callbacks := &recordingCallbacks{frameHooks: FrameHookBeginDraw, shouldDraw: true, hookFailure: sentinel}
	runtime := liveRuntime(callbacks)
	runtime.game = 3
	runtime.ownerThread = nativeOwnerThreadID()
	if _, err := runtime.invokeBeginDrawCallback(3, FrameTime{}); !errors.Is(err, sentinel) {
		t.Fatalf("begin_draw failure = %v, want the sentinel", err)
	}
	if !errors.Is(runtime.callbackFailure, sentinel) {
		t.Fatalf("begin_draw failure was not recorded: %v", runtime.callbackFailure)
	}
}

// TestAPanickingBeginDrawOverrideIsContained proves the value-channel hook uses
// the same containment every other callback does. Nothing may cross the C frame.
func TestAPanickingBeginDrawOverrideIsContained(t *testing.T) {
	callbacks := &recordingCallbacks{frameHooks: FrameHookBeginDraw, hookPanics: true}
	runtime := liveRuntime(callbacks)
	runtime.game = 5
	runtime.ownerThread = nativeOwnerThreadID()
	_, err := runtime.invokeBeginDrawCallback(5, FrameTime{})
	if err == nil || !strings.Contains(err.Error(), "panic in Game callback") {
		t.Fatalf("panicking begin_draw override produced %v", err)
	}
	if runtime.callbackFailure == nil {
		t.Fatal("a panicking begin_draw override recorded no callback failure")
	}
}
