package graphics

// CullMode identifies which primitive winding XNA culls.
type CullMode int32

const (
	CullModeNone                     CullMode = 0
	CullModeCullClockwiseFace        CullMode = 1
	CullModeCullCounterClockwiseFace CullMode = 2
)
