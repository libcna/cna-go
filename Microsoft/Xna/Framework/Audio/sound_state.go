package audio

// SoundState identifies the transport state of an XNA sound effect instance.
type SoundState int32

const (
	SoundStatePlaying SoundState = 0
	SoundStatePaused  SoundState = 1
	SoundStateStopped SoundState = 2
)
