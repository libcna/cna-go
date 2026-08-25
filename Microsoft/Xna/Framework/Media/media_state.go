package media

// MediaState identifies the transport state of XNA media playback.
type MediaState int32

const (
	MediaStateStopped MediaState = 0
	MediaStatePlaying MediaState = 1
	MediaStatePaused  MediaState = 2
)
