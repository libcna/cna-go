package graphics

// ColorWriteChannels specifies which colour channels an XNA render target
// write affects.
// xna:flags
type ColorWriteChannels int32

const (
	ColorWriteChannelsNone  ColorWriteChannels = 0
	ColorWriteChannelsRed   ColorWriteChannels = 1
	ColorWriteChannelsGreen ColorWriteChannels = 2
	ColorWriteChannelsBlue  ColorWriteChannels = 4
	ColorWriteChannelsAlpha ColorWriteChannels = 8
	ColorWriteChannelsAll   ColorWriteChannels = 15
)
