package graphics

// ClearOptions specifies which graphics buffers are cleared.
// xna:flags
type ClearOptions int32

const (
	ClearOptionsTarget      ClearOptions = 1
	ClearOptionsDepthBuffer ClearOptions = 2
	ClearOptionsStencil     ClearOptions = 4
)
