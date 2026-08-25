package graphics

// StencilOperation identifies how XNA updates a stencil buffer entry.
type StencilOperation int32

const (
	StencilOperationKeep                StencilOperation = 0
	StencilOperationZero                StencilOperation = 1
	StencilOperationReplace             StencilOperation = 2
	StencilOperationIncrement           StencilOperation = 3
	StencilOperationDecrement           StencilOperation = 4
	StencilOperationIncrementSaturation StencilOperation = 5
	StencilOperationDecrementSaturation StencilOperation = 6
	StencilOperationInvert              StencilOperation = 7
)
