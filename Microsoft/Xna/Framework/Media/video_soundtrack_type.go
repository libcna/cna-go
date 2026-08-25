package media

// VideoSoundtrackType identifies the soundtrack content of an XNA video.
type VideoSoundtrackType int32

const (
	VideoSoundtrackTypeMusic          VideoSoundtrackType = 0
	VideoSoundtrackTypeDialog         VideoSoundtrackType = 1
	VideoSoundtrackTypeMusicAndDialog VideoSoundtrackType = 2
)
