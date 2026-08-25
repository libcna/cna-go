package graphics

// BlendFunction identifies how XNA combines source and destination blend terms.
type BlendFunction int32

const (
	BlendFunctionAdd             BlendFunction = 0
	BlendFunctionSubtract        BlendFunction = 1
	BlendFunctionReverseSubtract BlendFunction = 2
	BlendFunctionMin             BlendFunction = 3
	BlendFunctionMax             BlendFunction = 4
)
