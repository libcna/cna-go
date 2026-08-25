package audio

// MicrophoneState identifies the capture state of an XNA microphone.
type MicrophoneState int32

const (
	MicrophoneStateStarted MicrophoneState = 0
	MicrophoneStateStopped MicrophoneState = 1
)
