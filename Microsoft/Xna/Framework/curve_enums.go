package framework

// CurveContinuity describes interpolation after a curve key.
type CurveContinuity int32

const (
	CurveContinuitySmooth CurveContinuity = 0
	CurveContinuityStep   CurveContinuity = 1
)

// CurveLoopType describes curve evaluation outside the key range.
type CurveLoopType int32

const (
	CurveLoopTypeConstant    CurveLoopType = 0
	CurveLoopTypeCycle       CurveLoopType = 1
	CurveLoopTypeCycleOffset CurveLoopType = 2
	CurveLoopTypeOscillate   CurveLoopType = 3
	CurveLoopTypeLinear      CurveLoopType = 4
)

// CurveTangent describes how a curve-key tangent is computed.
type CurveTangent int32

const (
	CurveTangentFlat   CurveTangent = 0
	CurveTangentLinear CurveTangent = 1
	CurveTangentSmooth CurveTangent = 2
)
