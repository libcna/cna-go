package graphics

// DepthFormat identifies the format of depth-stencil data.
type DepthFormat int32

const (
	DepthFormatNone            DepthFormat = 0
	DepthFormatDepth16         DepthFormat = 1
	DepthFormatDepth24         DepthFormat = 2
	DepthFormatDepth24Stencil8 DepthFormat = 3
)
