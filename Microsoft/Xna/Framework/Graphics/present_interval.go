package graphics

// PresentInterval identifies how XNA relates presentation to screen refresh.
type PresentInterval int32

const (
	PresentIntervalDefault   PresentInterval = 0
	PresentIntervalOne       PresentInterval = 1
	PresentIntervalTwo       PresentInterval = 2
	PresentIntervalImmediate PresentInterval = 3
)
