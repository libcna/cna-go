package audio

// AudioStopOptions identifies how an XNA audio stop request is authored.
type AudioStopOptions int32

const (
	AudioStopOptionsAsAuthored AudioStopOptions = 0
	AudioStopOptionsImmediate  AudioStopOptions = 1
)
