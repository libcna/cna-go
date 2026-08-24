package framework

import "fmt"

// Vector4 is XNA's four-dimensional binary32 value vector.
type Vector4 struct{ X, Y, Z, W float32 }

func NewVector4ByVector2AndSingleAndSingle(v Vector2, z, w float32) Vector4 {
	return Vector4{v.X, v.Y, z, w}
}
func NewVector4ByVector3AndSingle(v Vector3, w float32) Vector4 { return Vector4{v.X, v.Y, v.Z, w} }
func NewVector4BySingle(v float32) Vector4                      { return Vector4{v, v, v, v} }
func NewVector4BySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Vector4 {
	return Vector4{x, y, z, w}
}

func Vector4Zero() Vector4  { return Vector4{} }
func Vector4One() Vector4   { return Vector4{1, 1, 1, 1} }
func Vector4UnitX() Vector4 { return Vector4{X: 1} }
func Vector4UnitY() Vector4 { return Vector4{Y: 1} }
func Vector4UnitZ() Vector4 { return Vector4{Z: 1} }
func Vector4UnitW() Vector4 { return Vector4{W: 1} }

func (v Vector4) LengthSquared() float32 { return v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W }
func (v Vector4) Length() float32        { return sqrt32(v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W) }
func (v *Vector4) Normalize() {
	f := float32(1) / sqrt32(v.X*v.X+v.Y*v.Y+v.Z*v.Z+v.W*v.W)
	v.X *= f
	v.Y *= f
	v.Z *= f
	v.W *= f
}
func Vector4NormalizeByVector4(v Vector4) Vector4 { v.Normalize(); return v }
func Vector4NormalizeByRefVector4AndOutVector4(v *Vector4) Vector4 {
	return Vector4NormalizeByVector4(*v)
}

func Vector4DistanceByVector4AndVector4(a, b Vector4) float32 {
	return Vector4SubtractByVector4AndVector4(a, b).Length()
}
func Vector4DistanceByRefVector4AndRefVector4AndOutSingle(a, b *Vector4) float32 {
	return Vector4DistanceByVector4AndVector4(*a, *b)
}
func Vector4DistanceSquaredByVector4AndVector4(a, b Vector4) float32 {
	return Vector4SubtractByVector4AndVector4(a, b).LengthSquared()
}
func Vector4DistanceSquaredByRefVector4AndRefVector4AndOutSingle(a, b *Vector4) float32 {
	return Vector4DistanceSquaredByVector4AndVector4(*a, *b)
}
func Vector4DotByVector4AndVector4(a, b Vector4) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}
func Vector4DotByRefVector4AndRefVector4AndOutSingle(a, b *Vector4) float32 {
	return Vector4DotByVector4AndVector4(*a, *b)
}

func Vector4MinByVector4AndVector4(a, b Vector4) Vector4 {
	r := b
	if a.X < b.X {
		r.X = a.X
	}
	if a.Y < b.Y {
		r.Y = a.Y
	}
	if a.Z < b.Z {
		r.Z = a.Z
	}
	if a.W < b.W {
		r.W = a.W
	}
	return r
}
func Vector4MinByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4MinByVector4AndVector4(*a, *b)
}
func Vector4MaxByVector4AndVector4(a, b Vector4) Vector4 {
	r := b
	if a.X > b.X {
		r.X = a.X
	}
	if a.Y > b.Y {
		r.Y = a.Y
	}
	if a.Z > b.Z {
		r.Z = a.Z
	}
	if a.W > b.W {
		r.W = a.W
	}
	return r
}
func Vector4MaxByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4MaxByVector4AndVector4(*a, *b)
}
func Vector4ClampByVector4AndVector4AndVector4(v, min, max Vector4) Vector4 {
	return Vector4{MathHelperClamp(v.X, min.X, max.X), MathHelperClamp(v.Y, min.Y, max.Y), MathHelperClamp(v.Z, min.Z, max.Z), MathHelperClamp(v.W, min.W, max.W)}
}
func Vector4ClampByRefVector4AndRefVector4AndRefVector4AndOutVector4(v, min, max *Vector4) Vector4 {
	return Vector4ClampByVector4AndVector4AndVector4(*v, *min, *max)
}
func Vector4LerpByVector4AndVector4AndSingle(a, b Vector4, amount float32) Vector4 {
	return Vector4{a.X + (b.X-a.X)*amount, a.Y + (b.Y-a.Y)*amount, a.Z + (b.Z-a.Z)*amount, a.W + (b.W-a.W)*amount}
}
func Vector4LerpByRefVector4AndRefVector4AndSingleAndOutVector4(a, b *Vector4, amount float32) Vector4 {
	return Vector4LerpByVector4AndVector4AndSingle(*a, *b, amount)
}
func Vector4BarycentricByVector4AndVector4AndVector4AndSingleAndSingle(a, b, c Vector4, x, y float32) Vector4 {
	return Vector4{MathHelperBarycentric(a.X, b.X, c.X, x, y), MathHelperBarycentric(a.Y, b.Y, c.Y, x, y), MathHelperBarycentric(a.Z, b.Z, c.Z, x, y), MathHelperBarycentric(a.W, b.W, c.W, x, y)}
}
func Vector4BarycentricByRefVector4AndRefVector4AndRefVector4AndSingleAndSingleAndOutVector4(a, b, c *Vector4, x, y float32) Vector4 {
	return Vector4BarycentricByVector4AndVector4AndVector4AndSingleAndSingle(*a, *b, *c, x, y)
}
func Vector4SmoothStepByVector4AndVector4AndSingle(a, b Vector4, t float32) Vector4 {
	return Vector4{MathHelperSmoothStep(a.X, b.X, t), MathHelperSmoothStep(a.Y, b.Y, t), MathHelperSmoothStep(a.Z, b.Z, t), MathHelperSmoothStep(a.W, b.W, t)}
}
func Vector4SmoothStepByRefVector4AndRefVector4AndSingleAndOutVector4(a, b *Vector4, t float32) Vector4 {
	return Vector4SmoothStepByVector4AndVector4AndSingle(*a, *b, t)
}
func Vector4CatmullRomByVector4AndVector4AndVector4AndVector4AndSingle(a, b, c, d Vector4, t float32) Vector4 {
	return Vector4{MathHelperCatmullRom(a.X, b.X, c.X, d.X, t), MathHelperCatmullRom(a.Y, b.Y, c.Y, d.Y, t), MathHelperCatmullRom(a.Z, b.Z, c.Z, d.Z, t), MathHelperCatmullRom(a.W, b.W, c.W, d.W, t)}
}
func Vector4CatmullRomByRefVector4AndRefVector4AndRefVector4AndRefVector4AndSingleAndOutVector4(a, b, c, d *Vector4, t float32) Vector4 {
	return Vector4CatmullRomByVector4AndVector4AndVector4AndVector4AndSingle(*a, *b, *c, *d, t)
}
func Vector4HermiteByVector4AndVector4AndVector4AndVector4AndSingle(a, ta, b, tb Vector4, t float32) Vector4 {
	return Vector4{MathHelperHermite(a.X, ta.X, b.X, tb.X, t), MathHelperHermite(a.Y, ta.Y, b.Y, tb.Y, t), MathHelperHermite(a.Z, ta.Z, b.Z, tb.Z, t), MathHelperHermite(a.W, ta.W, b.W, tb.W, t)}
}
func Vector4HermiteByRefVector4AndRefVector4AndRefVector4AndRefVector4AndSingleAndOutVector4(a, ta, b, tb *Vector4, t float32) Vector4 {
	return Vector4HermiteByVector4AndVector4AndVector4AndVector4AndSingle(*a, *ta, *b, *tb, t)
}

func Vector4AddByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4{a.X + b.X, a.Y + b.Y, a.Z + b.Z, a.W + b.W}
}
func Vector4AddByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4AddByVector4AndVector4(*a, *b)
}
func Vector4SubtractByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4{a.X - b.X, a.Y - b.Y, a.Z - b.Z, a.W - b.W}
}
func Vector4SubtractByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4SubtractByVector4AndVector4(*a, *b)
}
func Vector4NegateByVector4(v Vector4) Vector4                  { return Vector4{-v.X, -v.Y, -v.Z, -v.W} }
func Vector4NegateByRefVector4AndOutVector4(v *Vector4) Vector4 { return Vector4NegateByVector4(*v) }
func Vector4MultiplyByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4{a.X * b.X, a.Y * b.Y, a.Z * b.Z, a.W * b.W}
}
func Vector4MultiplyByVector4AndSingle(v Vector4, s float32) Vector4 {
	return Vector4{v.X * s, v.Y * s, v.Z * s, v.W * s}
}
func Vector4MultiplyByRefVector4AndSingleAndOutVector4(v *Vector4, s float32) Vector4 {
	return Vector4MultiplyByVector4AndSingle(*v, s)
}
func Vector4MultiplyByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4MultiplyByVector4AndVector4(*a, *b)
}
func Vector4DivideByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4{a.X / b.X, a.Y / b.Y, a.Z / b.Z, a.W / b.W}
}
func Vector4DivideByVector4AndSingle(v Vector4, d float32) Vector4 {
	f := float32(1) / d
	return Vector4{v.X * f, v.Y * f, v.Z * f, v.W * f}
}
func Vector4DivideByRefVector4AndSingleAndOutVector4(v *Vector4, d float32) Vector4 {
	return Vector4DivideByVector4AndSingle(*v, d)
}
func Vector4DivideByRefVector4AndRefVector4AndOutVector4(a, b *Vector4) Vector4 {
	return Vector4DivideByVector4AndVector4(*a, *b)
}

func Vector4TransformByVector2AndMatrix(v Vector2, m Matrix) Vector4 {
	return Vector4TransformByRefVector2AndRefMatrixAndOutVector4(&v, &m)
}
func Vector4TransformByRefVector2AndRefMatrixAndOutVector4(v *Vector2, m *Matrix) Vector4 {
	return Vector4{v.X*m.M11 + v.Y*m.M21 + m.M41, v.X*m.M12 + v.Y*m.M22 + m.M42, v.X*m.M13 + v.Y*m.M23 + m.M43, v.X*m.M14 + v.Y*m.M24 + m.M44}
}
func Vector4TransformByVector3AndMatrix(v Vector3, m Matrix) Vector4 {
	return Vector4TransformByRefVector3AndRefMatrixAndOutVector4(&v, &m)
}
func Vector4TransformByRefVector3AndRefMatrixAndOutVector4(v *Vector3, m *Matrix) Vector4 {
	return Vector4{v.X*m.M11 + v.Y*m.M21 + v.Z*m.M31 + m.M41, v.X*m.M12 + v.Y*m.M22 + v.Z*m.M32 + m.M42, v.X*m.M13 + v.Y*m.M23 + v.Z*m.M33 + m.M43, v.X*m.M14 + v.Y*m.M24 + v.Z*m.M34 + m.M44}
}
func Vector4TransformByVector4AndMatrix(v Vector4, m Matrix) Vector4 {
	return Vector4TransformByRefVector4AndRefMatrixAndOutVector4(&v, &m)
}
func Vector4TransformByRefVector4AndRefMatrixAndOutVector4(v *Vector4, m *Matrix) Vector4 {
	return Vector4{v.X*m.M11 + v.Y*m.M21 + v.Z*m.M31 + v.W*m.M41, v.X*m.M12 + v.Y*m.M22 + v.Z*m.M32 + v.W*m.M42, v.X*m.M13 + v.Y*m.M23 + v.Z*m.M33 + v.W*m.M43, v.X*m.M14 + v.Y*m.M24 + v.Z*m.M34 + v.W*m.M44}
}
func Vector4TransformByVector2AndQuaternion(v Vector2, q Quaternion) Vector4 {
	return Vector4TransformByRefVector2AndRefQuaternionAndOutVector4(&v, &q)
}
func Vector4TransformByRefVector2AndRefQuaternionAndOutVector4(v *Vector2, q *Quaternion) Vector4 {
	r := Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(&Vector3{X: v.X, Y: v.Y}, q)
	return Vector4{r.X, r.Y, r.Z, 1}
}
func Vector4TransformByVector3AndQuaternion(v Vector3, q Quaternion) Vector4 {
	return Vector4TransformByRefVector3AndRefQuaternionAndOutVector4(&v, &q)
}
func Vector4TransformByRefVector3AndRefQuaternionAndOutVector4(v *Vector3, q *Quaternion) Vector4 {
	r := Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(v, q)
	return Vector4{r.X, r.Y, r.Z, 1}
}
func Vector4TransformByVector4AndQuaternion(v Vector4, q Quaternion) Vector4 {
	return Vector4TransformByRefVector4AndRefQuaternionAndOutVector4(&v, &q)
}
func Vector4TransformByRefVector4AndRefQuaternionAndOutVector4(v *Vector4, q *Quaternion) Vector4 {
	p := Vector3{v.X, v.Y, v.Z}
	r := Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(&p, q)
	return Vector4{r.X, r.Y, r.Z, v.W}
}

func Vector4TransformBySliceOfVector4AndInt32AndRefMatrixAndSliceOfVector4AndInt32AndInt32(src []Vector4, si int32, m *Matrix, dst []Vector4, di, n int32) {
	validateTransformSlices(src, si, dst, di, n)
	for i := int32(0); i < n; i++ {
		dst[di+i] = Vector4TransformByRefVector4AndRefMatrixAndOutVector4(&src[si+i], m)
	}
}
func Vector4TransformBySliceOfVector4AndRefMatrixAndSliceOfVector4(src []Vector4, m *Matrix, dst []Vector4) {
	if src == nil {
		panic("source array is nil")
	}
	Vector4TransformBySliceOfVector4AndInt32AndRefMatrixAndSliceOfVector4AndInt32AndInt32(src, 0, m, dst, 0, int32(len(src)))
}
func Vector4TransformBySliceOfVector4AndInt32AndRefQuaternionAndSliceOfVector4AndInt32AndInt32(src []Vector4, si int32, q *Quaternion, dst []Vector4, di, n int32) {
	validateTransformSlices(src, si, dst, di, n)
	for i := int32(0); i < n; i++ {
		dst[di+i] = Vector4TransformByRefVector4AndRefQuaternionAndOutVector4(&src[si+i], q)
	}
}
func Vector4TransformBySliceOfVector4AndRefQuaternionAndSliceOfVector4(src []Vector4, q *Quaternion, dst []Vector4) {
	if src == nil {
		panic("source array is nil")
	}
	Vector4TransformBySliceOfVector4AndInt32AndRefQuaternionAndSliceOfVector4AndInt32AndInt32(src, 0, q, dst, 0, int32(len(src)))
}

func (v Vector4) EqualsByVector4(o Vector4) bool {
	return v.X == o.X && v.Y == o.Y && v.Z == o.Z && v.W == o.W
}
func (v Vector4) EqualsByObject(value any) bool {
	o, ok := value.(Vector4)
	return ok && v.EqualsByVector4(o)
}
func (v Vector4) GetHashCode() int32 {
	return singleHashCode(v.X) + singleHashCode(v.Y) + singleHashCode(v.Z) + singleHashCode(v.W)
}
func (v Vector4) ToString() string {
	return fmt.Sprintf("{X:%s Y:%s Z:%s W:%s}", formatSingle(v.X), formatSingle(v.Y), formatSingle(v.Z), formatSingle(v.W))
}

func Vector4OperatorAdditionByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4AddByVector4AndVector4(a, b)
}
func Vector4OperatorSubtractionByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4SubtractByVector4AndVector4(a, b)
}
func Vector4OperatorUnaryNegationByVector4(v Vector4) Vector4 { return Vector4NegateByVector4(v) }
func Vector4OperatorMultiplyByVector4AndSingle(v Vector4, s float32) Vector4 {
	return Vector4MultiplyByVector4AndSingle(v, s)
}
func Vector4OperatorMultiplyBySingleAndVector4(s float32, v Vector4) Vector4 {
	return Vector4MultiplyByVector4AndSingle(v, s)
}
func Vector4OperatorMultiplyByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4MultiplyByVector4AndVector4(a, b)
}
func Vector4OperatorDivisionByVector4AndSingle(v Vector4, d float32) Vector4 {
	return Vector4DivideByVector4AndSingle(v, d)
}
func Vector4OperatorDivisionByVector4AndVector4(a, b Vector4) Vector4 {
	return Vector4DivideByVector4AndVector4(a, b)
}
func Vector4OperatorEqualityByVector4AndVector4(a, b Vector4) bool   { return a.EqualsByVector4(b) }
func Vector4OperatorInequalityByVector4AndVector4(a, b Vector4) bool { return !a.EqualsByVector4(b) }
