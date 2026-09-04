// SPDX-License-Identifier: MS-PL

// Package dispatcher holds the one piece of FrameworkDispatcher's private state
// that a projected member outside its package has to read.
//
// # Why it is not a field on the projected type
//
// `FrameworkDispatcher.UpdateCalledAtLeastOnce` is `assembly` in the reference
// and is not in the pinned contract, so it must not become public surface. It is
// also read from ANOTHER namespace: SoundEffect::Play's very first statement,
// ahead of its lock and its disposal check, is
//
//	if (!FrameworkDispatcher.UpdateCalledAtLeastOnce)
//	    throw new InvalidOperationException(FrameworkResources.CallFrameworkDispatcherUpdate);
//
// Go cannot share an unexported symbol across packages, and the two alternatives
// were worse: exporting an accessor would add a member the contract does not
// declare, and duplicating the flag in the audio package would give one process
// two answers to one question.
//
// So it lives here, `internal`, where a consumer of the module cannot reach it
// and both projected packages can. That is the same shape internal/bclhash has
// and for the same reason.
//
// # There is one dispatcher per process
//
// The reference's flag is a STATIC field, so this is a package-level variable
// rather than per-object state. Nothing resets it: once Update has been called,
// the reference never sets it back to false.
package dispatcher

// updated is UpdateCalledAtLeastOnce.
var updated bool

// MarkUpdated is what FrameworkDispatcher.Update does with it, and it is the
// method's FIRST statement -- so a dispatcher call that goes on to fail has
// still been made, and the guard below asks whether Update was CALLED rather
// than whether it succeeded.
func MarkUpdated() { updated = true }

// HasRun answers the flag for SoundEffect.Play's guard.
func HasRun() bool { return updated }
