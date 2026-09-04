package framework

import (
	"testing"

	"github.com/openeggbert/cna-go/internal/dispatcher"
)

// TestUpdateMarksTheFlagBeforeItCanRefuse pins the ORDER of
// FrameworkDispatcher.Update's two statements.
//
// `UpdateCalledAtLeastOnce = true` is the reference's FIRST statement, ahead of
// everything else it does. The projection's refusal when no game is running
// comes after it, so a call that fails has still been MADE -- and
// SoundEffect.Play's guard asks whether Update was called, not whether it
// succeeded.
//
// The flag is process-wide and nothing resets it, exactly as the reference's
// static field is, so this test asserts only that the flag is set after a call.
// A projection that marked it after the runtime lookup would leave it false
// here and refuse every fire-and-forget Play in a process with no game.
func TestUpdateMarksTheFlagBeforeItCanRefuse(t *testing.T) {
	// The call is expected to fail with no running game; what matters is the
	// flag it set on the way.
	_ = FrameworkDispatcherUpdate()
	if !dispatcher.HasRun() {
		t.Fatal("FrameworkDispatcher.Update did not mark the flag; the store is its FIRST statement")
	}
}
