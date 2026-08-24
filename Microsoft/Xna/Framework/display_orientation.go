package framework

// DisplayOrientation describes the orientations supported by a display.
// xna:flags
type DisplayOrientation int32

const (
	DisplayOrientationDefault        DisplayOrientation = 0
	DisplayOrientationLandscapeLeft  DisplayOrientation = 1
	DisplayOrientationLandscapeRight DisplayOrientation = 2
	DisplayOrientationPortrait       DisplayOrientation = 4
)
