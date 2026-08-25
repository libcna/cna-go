package touch

// TouchLocationState identifies the state of an XNA touch location.
type TouchLocationState int32

const (
	TouchLocationStateInvalid  TouchLocationState = 0
	TouchLocationStateReleased TouchLocationState = 1
	TouchLocationStatePressed  TouchLocationState = 2
	TouchLocationStateMoved    TouchLocationState = 3
)
