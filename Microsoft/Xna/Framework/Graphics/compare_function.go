package graphics

// CompareFunction identifies an XNA comparison test.
type CompareFunction int32

const (
	CompareFunctionAlways       CompareFunction = 0
	CompareFunctionNever        CompareFunction = 1
	CompareFunctionLess         CompareFunction = 2
	CompareFunctionLessEqual    CompareFunction = 3
	CompareFunctionEqual        CompareFunction = 4
	CompareFunctionGreaterEqual CompareFunction = 5
	CompareFunctionGreater      CompareFunction = 6
	CompareFunctionNotEqual     CompareFunction = 7
)
