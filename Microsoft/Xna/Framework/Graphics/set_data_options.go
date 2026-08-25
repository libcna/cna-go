package graphics

// SetDataOptions specifies how an XNA buffer data write relates to data the
// buffer already holds.
// xna:flags
type SetDataOptions int32

const (
	SetDataOptionsNone        SetDataOptions = 0
	SetDataOptionsDiscard     SetDataOptions = 1
	SetDataOptionsNoOverwrite SetDataOptions = 2
)
