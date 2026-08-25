package graphics

// RenderTargetUsage identifies how the contents of an XNA render target are
// treated when that render target is set on a device.
type RenderTargetUsage int32

const (
	RenderTargetUsageDiscardContents  RenderTargetUsage = 0
	RenderTargetUsagePreserveContents RenderTargetUsage = 1
	RenderTargetUsagePlatformContents RenderTargetUsage = 2
)
