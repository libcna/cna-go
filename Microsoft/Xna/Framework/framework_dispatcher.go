package framework

import (
	"errors"

	"github.com/openeggbert/cna-go/internal/interop"
)

// FrameworkDispatcher is the XNA static dispatcher type identity. The
// reference declares it `public abstract sealed`, which is C# for a static
// class; Go has no static methods, so its one member maps to a type-prefixed
// package declaration and the type itself carries the identity.
//
// It has no state and no constructor, which is what `abstract sealed` means:
// the reference's own fields -- UpdateCalledAtLeastOnce, the pending-call queue
// and its copy buffer -- are all private and none is in the pinned contract.
type FrameworkDispatcher struct{}

// FrameworkDispatcherUpdate is
// Microsoft.Xna.Framework.FrameworkDispatcher::Update(), the only member of
// `public abstract sealed class FrameworkDispatcher`.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # What the reference's 290 bytes do
//
//	UpdateCalledAtLeastOnce = true;
//	PollForEvents();
//	lock (pendingCalls) {
//	    foreach (var call in pendingCalls) pendingCallsCopy.Add(call);
//	    pendingCalls.Clear();
//	}
//	foreach (var call in pendingCallsCopy) ... dispatch by ManagedCallType ...
//
// Two halves: it polls the platform for events, and it drains a queue of
// managed calls other subsystems -- media, audio, gamer services -- posted from
// whatever thread they ran on. The queue, its lock, its copy buffer and the
// ManagedCallType switch are all private state of an assembly-internal design,
// and none of it is in the pinned contract.
//
// CNA has the same member and the same shape behind it:
// `cna_framework_dispatcher_update` "pumps the framework-wide per-frame work
// the game loop normally drives". So the projection is that route and nothing
// else -- reproducing the queue managed-side would be a second dispatcher for
// work CNA has already done, and the two would disagree.
//
// # Why a static member needs a running game
//
// The reference's dispatcher is static and needs nothing. CNA's route takes a
// game handle, and the canonical header says exactly why: "The canonical
// dispatcher is static and exists for applications that do not run the game
// loop; a game handle is taken here only for thread affinity."
//
// So this member answers a refusal outside a game where the reference answers
// nothing at all. That is the same shape GraphicsAdapter's two statics already
// have, and it is recorded rather than hidden: the alternative is doing nothing
// and reporting success, which would let a consumer believe media and audio had
// been pumped when they had not.
//
// # It is FALLIBLE, and the reference's is not
//
// `Update` returns void and its body has no throw site a caller can reach. The
// error result here is the native boundary's, which is the settled rule for
// every projected member that crosses one.
func FrameworkDispatcherUpdate() error {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errNoRunningGame
	}
	return runtime.FrameworkDispatcherUpdate()
}

// errNoRunningGame is the refusal both root-type statics give outside a game.
// It is the projection's own, not a borrowed XNA message: the reference has no
// such failure, so there is no message to reproduce.
var errNoRunningGame = errors.New(
	"no game is running on this thread, and CNA takes a game handle for thread affinity even on a static member")
