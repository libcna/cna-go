package graphics

// GraphicsDeviceStatus identifies the reported status of an XNA graphics device.
type GraphicsDeviceStatus int32

const (
	GraphicsDeviceStatusNormal   GraphicsDeviceStatus = 0
	GraphicsDeviceStatusLost     GraphicsDeviceStatus = 1
	GraphicsDeviceStatusNotReset GraphicsDeviceStatus = 2
)
