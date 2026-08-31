package framework

import (
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// TestIsActiveReportsTheFieldTheNativeActivationSignalsMaintain measures the
// member against the state the reference's own host handlers keep, through the
// edge-triggered path Foundation 34 projected.
//
// The reference's get_IsActive is `isActive && !(GamerServicesDispatcher.IsInitialized
// && Guide.IsVisible)`, and the guide half is unreachable in CNA-Go: both
// statics live in Microsoft.Xna.Framework.GamerServices.dll, IsInitialized is
// `packetBuffer != null`, and packetBuffer's only assignment in that whole
// assembly is inside GamerServicesDispatcher.Initialize -- a method on a type
// CNA-Go projects no part of. See game_events.go.
func TestIsActiveReportsTheFieldTheNativeActivationSignalsMaintain(t *testing.T) {
	game := &Game{}
	if game.IsActive() {
		t.Fatal("a Game reports active before any activation signal")
	}
	if err := game.raiseNativeGameEvent(interop.GameEventActivated); err != nil {
		t.Fatalf("activation: %v", err)
	}
	if !game.IsActive() {
		t.Fatal("a Game does not report active after the activation signal")
	}
	// The edge-triggered guard means a repeated signal changes nothing, and
	// IsActive must not drift with it.
	if err := game.raiseNativeGameEvent(interop.GameEventActivated); err != nil {
		t.Fatalf("repeated activation: %v", err)
	}
	if !game.IsActive() {
		t.Fatal("a repeated activation signal cleared IsActive")
	}
	if err := game.raiseNativeGameEvent(interop.GameEventDeactivated); err != nil {
		t.Fatalf("deactivation: %v", err)
	}
	if game.IsActive() {
		t.Fatal("a Game still reports active after the deactivation signal")
	}
}

// TestIsActiveOnANilGameIsFalseRatherThanAPanic pins the Go-only guard. Go can
// produce a nil *Game where the CLR cannot produce a null `this`, and a
// property that panics would be a failure mode the reference has no equivalent
// for.
func TestIsActiveOnANilGameIsFalseRatherThanAPanic(t *testing.T) {
	var game *Game
	if game.IsActive() {
		t.Fatal("a nil Game reports active")
	}
}
