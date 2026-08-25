package media

// MediaSourceType identifies the origin of an XNA media source.
type MediaSourceType int32

const (
	MediaSourceTypeLocalDevice         MediaSourceType = 0
	MediaSourceTypeWindowsMediaConnect MediaSourceType = 4
)
