package framework

import "fmt"

// Vector3 is XNA's three-dimensional binary32 value vector.
type Vector3 struct {
	X float32
	Y float32
	Z float32
}

func NewVector3ByVector2AndSingle(value Vector2, z float32) Vector3 {
	return Vector3{X: value.X, Y: value.Y, Z: z}
}
func NewVector3BySingle(value float32) Vector3 { return Vector3{X: value, Y: value, Z: value} }
func NewVector3BySingleAndSingleAndSingle(x, y, z float32) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

func Vector3Zero() Vector3     { return Vector3{} }
func Vector3One() Vector3      { return Vector3{X: 1, Y: 1, Z: 1} }
func Vector3UnitX() Vector3    { return Vector3{X: 1} }
func Vector3UnitY() Vector3    { return Vector3{Y: 1} }
func Vector3UnitZ() Vector3    { return Vector3{Z: 1} }
func Vector3Up() Vector3       { return Vector3{Y: 1} }
func Vector3Down() Vector3     { return Vector3{Y: -1} }
func Vector3Right() Vector3    { return Vector3{X: 1} }
func Vector3Left() Vector3     { return Vector3{X: -1} }
func Vector3Forward() Vector3  { return Vector3{Z: -1} }
func Vector3Backward() Vector3 { return Vector3{Z: 1} }

func (v Vector3) LengthSquared() float32 { return v.X*v.X + v.Y*v.Y + v.Z*v.Z }
func (v Vector3) Length() float32        { return sqrt32(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
func (v *Vector3) Normalize() {
	factor := float32(1) / sqrt32(v.X*v.X+v.Y*v.Y+v.Z*v.Z)
	v.X *= factor
	v.Y *= factor
	v.Z *= factor
}

func Vector3NormalizeByVector3(value Vector3) Vector3 { value.Normalize(); return value }
func Vector3NormalizeByRefVector3AndOutVector3(value *Vector3) Vector3 {
	return Vector3NormalizeByVector3(*value)
}
func Vector3DistanceByVector3AndVector3(a, b Vector3) float32 {
	return Vector3SubtractByVector3AndVector3(a, b).Length()
}
func Vector3DistanceByRefVector3AndRefVector3AndOutSingle(a, b *Vector3) float32 {
	return Vector3DistanceByVector3AndVector3(*a, *b)
}
func Vector3DistanceSquaredByVector3AndVector3(a, b Vector3) float32 {
	return Vector3SubtractByVector3AndVector3(a, b).LengthSquared()
}
func Vector3DistanceSquaredByRefVector3AndRefVector3AndOutSingle(a, b *Vector3) float32 {
	return Vector3DistanceSquaredByVector3AndVector3(*a, *b)
}
func Vector3DotByVector3AndVector3(a, b Vector3) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}
func Vector3DotByRefVector3AndRefVector3AndOutSingle(a, b *Vector3) float32 {
	return Vector3DotByVector3AndVector3(*a, *b)
}
func Vector3CrossByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
func Vector3CrossByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3CrossByVector3AndVector3(*a, *b)
}
func Vector3ReflectByVector3AndVector3(vector, normal Vector3) Vector3 {
	dot := vector.X*normal.X + vector.Y*normal.Y + vector.Z*normal.Z
	return Vector3{X: vector.X - 2*dot*normal.X, Y: vector.Y - 2*dot*normal.Y, Z: vector.Z - 2*dot*normal.Z}
}
func Vector3ReflectByRefVector3AndRefVector3AndOutVector3(vector, normal *Vector3) Vector3 {
	return Vector3ReflectByVector3AndVector3(*vector, *normal)
}

func Vector3MinByVector3AndVector3(a, b Vector3) Vector3 {
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
	return r
}
func Vector3MinByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3MinByVector3AndVector3(*a, *b)
}
func Vector3MaxByVector3AndVector3(a, b Vector3) Vector3 {
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
	return r
}
func Vector3MaxByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3MaxByVector3AndVector3(*a, *b)
}
func Vector3ClampByVector3AndVector3AndVector3(value, min, max Vector3) Vector3 {
	return Vector3{X: MathHelperClamp(value.X, min.X, max.X), Y: MathHelperClamp(value.Y, min.Y, max.Y), Z: MathHelperClamp(value.Z, min.Z, max.Z)}
}
func Vector3ClampByRefVector3AndRefVector3AndRefVector3AndOutVector3(value, min, max *Vector3) Vector3 {
	return Vector3ClampByVector3AndVector3AndVector3(*value, *min, *max)
}
func Vector3LerpByVector3AndVector3AndSingle(a, b Vector3, amount float32) Vector3 {
	return Vector3{X: a.X + (b.X-a.X)*amount, Y: a.Y + (b.Y-a.Y)*amount, Z: a.Z + (b.Z-a.Z)*amount}
}
func Vector3LerpByRefVector3AndRefVector3AndSingleAndOutVector3(a, b *Vector3, amount float32) Vector3 {
	return Vector3LerpByVector3AndVector3AndSingle(*a, *b, amount)
}
func Vector3BarycentricByVector3AndVector3AndVector3AndSingleAndSingle(a, b, c Vector3, amount1, amount2 float32) Vector3 {
	return Vector3{X: MathHelperBarycentric(a.X, b.X, c.X, amount1, amount2), Y: MathHelperBarycentric(a.Y, b.Y, c.Y, amount1, amount2), Z: MathHelperBarycentric(a.Z, b.Z, c.Z, amount1, amount2)}
}
func Vector3BarycentricByRefVector3AndRefVector3AndRefVector3AndSingleAndSingleAndOutVector3(a, b, c *Vector3, amount1, amount2 float32) Vector3 {
	return Vector3BarycentricByVector3AndVector3AndVector3AndSingleAndSingle(*a, *b, *c, amount1, amount2)
}
func Vector3SmoothStepByVector3AndVector3AndSingle(a, b Vector3, amount float32) Vector3 {
	return Vector3{X: MathHelperSmoothStep(a.X, b.X, amount), Y: MathHelperSmoothStep(a.Y, b.Y, amount), Z: MathHelperSmoothStep(a.Z, b.Z, amount)}
}
func Vector3SmoothStepByRefVector3AndRefVector3AndSingleAndOutVector3(a, b *Vector3, amount float32) Vector3 {
	return Vector3SmoothStepByVector3AndVector3AndSingle(*a, *b, amount)
}
func Vector3CatmullRomByVector3AndVector3AndVector3AndVector3AndSingle(a, b, c, d Vector3, amount float32) Vector3 {
	return Vector3{X: MathHelperCatmullRom(a.X, b.X, c.X, d.X, amount), Y: MathHelperCatmullRom(a.Y, b.Y, c.Y, d.Y, amount), Z: MathHelperCatmullRom(a.Z, b.Z, c.Z, d.Z, amount)}
}
func Vector3CatmullRomByRefVector3AndRefVector3AndRefVector3AndRefVector3AndSingleAndOutVector3(a, b, c, d *Vector3, amount float32) Vector3 {
	return Vector3CatmullRomByVector3AndVector3AndVector3AndVector3AndSingle(*a, *b, *c, *d, amount)
}
func Vector3HermiteByVector3AndVector3AndVector3AndVector3AndSingle(value1, tangent1, value2, tangent2 Vector3, amount float32) Vector3 {
	return Vector3{X: MathHelperHermite(value1.X, tangent1.X, value2.X, tangent2.X, amount), Y: MathHelperHermite(value1.Y, tangent1.Y, value2.Y, tangent2.Y, amount), Z: MathHelperHermite(value1.Z, tangent1.Z, value2.Z, tangent2.Z, amount)}
}
func Vector3HermiteByRefVector3AndRefVector3AndRefVector3AndRefVector3AndSingleAndOutVector3(value1, tangent1, value2, tangent2 *Vector3, amount float32) Vector3 {
	return Vector3HermiteByVector3AndVector3AndVector3AndVector3AndSingle(*value1, *tangent1, *value2, *tangent2, amount)
}

func Vector3AddByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}
func Vector3AddByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3AddByVector3AndVector3(*a, *b)
}
func Vector3SubtractByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}
func Vector3SubtractByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3SubtractByVector3AndVector3(*a, *b)
}
func Vector3NegateByVector3(v Vector3) Vector3                  { return Vector3{X: -v.X, Y: -v.Y, Z: -v.Z} }
func Vector3NegateByRefVector3AndOutVector3(v *Vector3) Vector3 { return Vector3NegateByVector3(*v) }
func Vector3MultiplyByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3{X: a.X * b.X, Y: a.Y * b.Y, Z: a.Z * b.Z}
}
func Vector3MultiplyByVector3AndSingle(v Vector3, scale float32) Vector3 {
	return Vector3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}
func Vector3MultiplyByRefVector3AndSingleAndOutVector3(v *Vector3, scale float32) Vector3 {
	return Vector3MultiplyByVector3AndSingle(*v, scale)
}
func Vector3MultiplyByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3MultiplyByVector3AndVector3(*a, *b)
}
func Vector3DivideByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3{X: a.X / b.X, Y: a.Y / b.Y, Z: a.Z / b.Z}
}
func Vector3DivideByVector3AndSingle(v Vector3, divisor float32) Vector3 {
	factor := float32(1) / divisor
	return Vector3{X: v.X * factor, Y: v.Y * factor, Z: v.Z * factor}
}
func Vector3DivideByRefVector3AndSingleAndOutVector3(v *Vector3, divisor float32) Vector3 {
	return Vector3DivideByVector3AndSingle(*v, divisor)
}
func Vector3DivideByRefVector3AndRefVector3AndOutVector3(a, b *Vector3) Vector3 {
	return Vector3DivideByVector3AndVector3(*a, *b)
}

func Vector3TransformByVector3AndMatrix(position Vector3, matrix Matrix) Vector3 {
	return Vector3TransformByRefVector3AndRefMatrixAndOutVector3(&position, &matrix)
}
func Vector3TransformByRefVector3AndRefMatrixAndOutVector3(position *Vector3, matrix *Matrix) Vector3 {
	return Vector3{
		X: position.X*matrix.M11 + position.Y*matrix.M21 + position.Z*matrix.M31 + matrix.M41,
		Y: position.X*matrix.M12 + position.Y*matrix.M22 + position.Z*matrix.M32 + matrix.M42,
		Z: position.X*matrix.M13 + position.Y*matrix.M23 + position.Z*matrix.M33 + matrix.M43,
	}
}
func Vector3TransformByVector3AndQuaternion(value Vector3, rotation Quaternion) Vector3 {
	return Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(&value, &rotation)
}
func Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(value *Vector3, rotation *Quaternion) Vector3 {
	x2, y2, z2 := rotation.X+rotation.X, rotation.Y+rotation.Y, rotation.Z+rotation.Z
	wx2, wy2, wz2 := rotation.W*x2, rotation.W*y2, rotation.W*z2
	xx2, xy2, xz2 := rotation.X*x2, rotation.X*y2, rotation.X*z2
	yy2, yz2, zz2 := rotation.Y*y2, rotation.Y*z2, rotation.Z*z2
	return Vector3{
		X: value.X*(1-yy2-zz2) + value.Y*(xy2-wz2) + value.Z*(xz2+wy2),
		Y: value.X*(xy2+wz2) + value.Y*(1-xx2-zz2) + value.Z*(yz2-wx2),
		Z: value.X*(xz2-wy2) + value.Y*(yz2+wx2) + value.Z*(1-xx2-yy2),
	}
}
func Vector3TransformNormalByVector3AndMatrix(normal Vector3, matrix Matrix) Vector3 {
	return Vector3TransformNormalByRefVector3AndRefMatrixAndOutVector3(&normal, &matrix)
}
func Vector3TransformNormalByRefVector3AndRefMatrixAndOutVector3(normal *Vector3, matrix *Matrix) Vector3 {
	return Vector3{
		X: normal.X*matrix.M11 + normal.Y*matrix.M21 + normal.Z*matrix.M31,
		Y: normal.X*matrix.M12 + normal.Y*matrix.M22 + normal.Z*matrix.M32,
		Z: normal.X*matrix.M13 + normal.Y*matrix.M23 + normal.Z*matrix.M33,
	}
}

func Vector3TransformBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(source []Vector3, sourceIndex int32, matrix *Matrix, destination []Vector3, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector3TransformByRefVector3AndRefMatrixAndOutVector3(&source[sourceIndex+i], matrix)
	}
}
func Vector3TransformBySliceOfVector3AndRefMatrixAndSliceOfVector3(source []Vector3, matrix *Matrix, destination []Vector3) {
	if source == nil {
		panic("source array is nil")
	}
	Vector3TransformBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(source, 0, matrix, destination, 0, int32(len(source)))
}
func Vector3TransformBySliceOfVector3AndInt32AndRefQuaternionAndSliceOfVector3AndInt32AndInt32(source []Vector3, sourceIndex int32, rotation *Quaternion, destination []Vector3, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(&source[sourceIndex+i], rotation)
	}
}
func Vector3TransformBySliceOfVector3AndRefQuaternionAndSliceOfVector3(source []Vector3, rotation *Quaternion, destination []Vector3) {
	if source == nil {
		panic("source array is nil")
	}
	Vector3TransformBySliceOfVector3AndInt32AndRefQuaternionAndSliceOfVector3AndInt32AndInt32(source, 0, rotation, destination, 0, int32(len(source)))
}
func Vector3TransformNormalBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(source []Vector3, sourceIndex int32, matrix *Matrix, destination []Vector3, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector3TransformNormalByRefVector3AndRefMatrixAndOutVector3(&source[sourceIndex+i], matrix)
	}
}
func Vector3TransformNormalBySliceOfVector3AndRefMatrixAndSliceOfVector3(source []Vector3, matrix *Matrix, destination []Vector3) {
	if source == nil {
		panic("source array is nil")
	}
	Vector3TransformNormalBySliceOfVector3AndInt32AndRefMatrixAndSliceOfVector3AndInt32AndInt32(source, 0, matrix, destination, 0, int32(len(source)))
}

func (v Vector3) EqualsByVector3(other Vector3) bool {
	return v.X == other.X && v.Y == other.Y && v.Z == other.Z
}
func (v Vector3) EqualsByObject(value any) bool {
	other, ok := value.(Vector3)
	return ok && v.EqualsByVector3(other)
}
func (v Vector3) GetHashCode() int32 {
	return singleHashCode(v.X) + singleHashCode(v.Y) + singleHashCode(v.Z)
}
func (v Vector3) ToString() string {
	return fmt.Sprintf("{X:%s Y:%s Z:%s}", formatSingle(v.X), formatSingle(v.Y), formatSingle(v.Z))
}

func Vector3OperatorAdditionByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3AddByVector3AndVector3(a, b)
}
func Vector3OperatorSubtractionByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3SubtractByVector3AndVector3(a, b)
}
func Vector3OperatorUnaryNegationByVector3(v Vector3) Vector3 { return Vector3NegateByVector3(v) }
func Vector3OperatorMultiplyByVector3AndSingle(v Vector3, scale float32) Vector3 {
	return Vector3MultiplyByVector3AndSingle(v, scale)
}
func Vector3OperatorMultiplyBySingleAndVector3(scale float32, v Vector3) Vector3 {
	return Vector3MultiplyByVector3AndSingle(v, scale)
}
func Vector3OperatorMultiplyByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3MultiplyByVector3AndVector3(a, b)
}
func Vector3OperatorDivisionByVector3AndSingle(v Vector3, divisor float32) Vector3 {
	return Vector3DivideByVector3AndSingle(v, divisor)
}
func Vector3OperatorDivisionByVector3AndVector3(a, b Vector3) Vector3 {
	return Vector3DivideByVector3AndVector3(a, b)
}
func Vector3OperatorEqualityByVector3AndVector3(a, b Vector3) bool   { return a.EqualsByVector3(b) }
func Vector3OperatorInequalityByVector3AndVector3(a, b Vector3) bool { return !a.EqualsByVector3(b) }
