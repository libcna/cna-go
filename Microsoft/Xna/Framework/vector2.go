package framework

import "fmt"

// Vector2 is a two-dimensional XNA vector implemented entirely in Go.
type Vector2 struct {
	X float32
	Y float32
}

func NewVector2BySingle(value float32) Vector2 {
	return Vector2{X: value, Y: value}
}

func NewVector2BySingleAndSingle(x, y float32) Vector2 {
	return Vector2{X: x, Y: y}
}

func Vector2AddByVector2AndVector2(value1, value2 Vector2) Vector2 {
	return Vector2{X: value1.X + value2.X, Y: value1.Y + value2.Y}
}

func Vector2AddByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2AddByVector2AndVector2(*value1, *value2)
}

// LengthSquared returns the squared Euclidean length of v.
func (v Vector2) LengthSquared() float32 {
	return v.X*v.X + v.Y*v.Y
}

func (v Vector2) Length() float32 { return sqrt32(v.X*v.X + v.Y*v.Y) }

func (v *Vector2) Normalize() {
	factor := float32(1) / sqrt32(v.X*v.X+v.Y*v.Y)
	v.X *= factor
	v.Y *= factor
}

func Vector2Zero() Vector2  { return Vector2{} }
func Vector2One() Vector2   { return Vector2{X: 1, Y: 1} }
func Vector2UnitX() Vector2 { return Vector2{X: 1} }
func Vector2UnitY() Vector2 { return Vector2{Y: 1} }

func Vector2DistanceByVector2AndVector2(value1, value2 Vector2) float32 {
	return Vector2SubtractByVector2AndVector2(value1, value2).Length()
}

func Vector2DistanceByRefVector2AndRefVector2AndOutSingle(value1, value2 *Vector2) float32 {
	return Vector2DistanceByVector2AndVector2(*value1, *value2)
}

func Vector2DistanceSquaredByVector2AndVector2(value1, value2 Vector2) float32 {
	return Vector2SubtractByVector2AndVector2(value1, value2).LengthSquared()
}

func Vector2DistanceSquaredByRefVector2AndRefVector2AndOutSingle(value1, value2 *Vector2) float32 {
	return Vector2DistanceSquaredByVector2AndVector2(*value1, *value2)
}

func Vector2DotByVector2AndVector2(value1, value2 Vector2) float32 {
	return value1.X*value2.X + value1.Y*value2.Y
}

func Vector2DotByRefVector2AndRefVector2AndOutSingle(value1, value2 *Vector2) float32 {
	return Vector2DotByVector2AndVector2(*value1, *value2)
}

func Vector2ReflectByVector2AndVector2(vector, normal Vector2) Vector2 {
	factor := 2 * Vector2DotByVector2AndVector2(vector, normal)
	return Vector2{X: vector.X - factor*normal.X, Y: vector.Y - factor*normal.Y}
}

func Vector2ReflectByRefVector2AndRefVector2AndOutVector2(vector, normal *Vector2) Vector2 {
	return Vector2ReflectByVector2AndVector2(*vector, *normal)
}

func Vector2MinByVector2AndVector2(value1, value2 Vector2) Vector2 {
	result := value2
	if value1.X < value2.X {
		result.X = value1.X
	}
	if value1.Y < value2.Y {
		result.Y = value1.Y
	}
	return result
}

func Vector2MinByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2MinByVector2AndVector2(*value1, *value2)
}

func Vector2MaxByVector2AndVector2(value1, value2 Vector2) Vector2 {
	result := value2
	if value1.X > value2.X {
		result.X = value1.X
	}
	if value1.Y > value2.Y {
		result.Y = value1.Y
	}
	return result
}

func Vector2MaxByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2MaxByVector2AndVector2(*value1, *value2)
}

func Vector2ClampByVector2AndVector2AndVector2(value, min, max Vector2) Vector2 {
	return Vector2{X: MathHelperClamp(value.X, min.X, max.X), Y: MathHelperClamp(value.Y, min.Y, max.Y)}
}

func Vector2ClampByRefVector2AndRefVector2AndRefVector2AndOutVector2(value, min, max *Vector2) Vector2 {
	return Vector2ClampByVector2AndVector2AndVector2(*value, *min, *max)
}

func Vector2LerpByVector2AndVector2AndSingle(value1, value2 Vector2, amount float32) Vector2 {
	return Vector2{X: value1.X + (value2.X-value1.X)*amount, Y: value1.Y + (value2.Y-value1.Y)*amount}
}

func Vector2LerpByRefVector2AndRefVector2AndSingleAndOutVector2(value1, value2 *Vector2, amount float32) Vector2 {
	return Vector2LerpByVector2AndVector2AndSingle(*value1, *value2, amount)
}

func Vector2BarycentricByVector2AndVector2AndVector2AndSingleAndSingle(value1, value2, value3 Vector2, amount1, amount2 float32) Vector2 {
	return Vector2{
		X: MathHelperBarycentric(value1.X, value2.X, value3.X, amount1, amount2),
		Y: MathHelperBarycentric(value1.Y, value2.Y, value3.Y, amount1, amount2),
	}
}

func Vector2BarycentricByRefVector2AndRefVector2AndRefVector2AndSingleAndSingleAndOutVector2(value1, value2, value3 *Vector2, amount1, amount2 float32) Vector2 {
	return Vector2BarycentricByVector2AndVector2AndVector2AndSingleAndSingle(*value1, *value2, *value3, amount1, amount2)
}

func Vector2SmoothStepByVector2AndVector2AndSingle(value1, value2 Vector2, amount float32) Vector2 {
	return Vector2{
		X: MathHelperSmoothStep(value1.X, value2.X, amount),
		Y: MathHelperSmoothStep(value1.Y, value2.Y, amount),
	}
}

func Vector2SmoothStepByRefVector2AndRefVector2AndSingleAndOutVector2(value1, value2 *Vector2, amount float32) Vector2 {
	return Vector2SmoothStepByVector2AndVector2AndSingle(*value1, *value2, amount)
}

func Vector2CatmullRomByVector2AndVector2AndVector2AndVector2AndSingle(value1, value2, value3, value4 Vector2, amount float32) Vector2 {
	return Vector2{
		X: MathHelperCatmullRom(value1.X, value2.X, value3.X, value4.X, amount),
		Y: MathHelperCatmullRom(value1.Y, value2.Y, value3.Y, value4.Y, amount),
	}
}

func Vector2CatmullRomByRefVector2AndRefVector2AndRefVector2AndRefVector2AndSingleAndOutVector2(value1, value2, value3, value4 *Vector2, amount float32) Vector2 {
	return Vector2CatmullRomByVector2AndVector2AndVector2AndVector2AndSingle(*value1, *value2, *value3, *value4, amount)
}

func Vector2HermiteByVector2AndVector2AndVector2AndVector2AndSingle(value1, tangent1, value2, tangent2 Vector2, amount float32) Vector2 {
	return Vector2{
		X: MathHelperHermite(value1.X, tangent1.X, value2.X, tangent2.X, amount),
		Y: MathHelperHermite(value1.Y, tangent1.Y, value2.Y, tangent2.Y, amount),
	}
}

func Vector2HermiteByRefVector2AndRefVector2AndRefVector2AndRefVector2AndSingleAndOutVector2(value1, tangent1, value2, tangent2 *Vector2, amount float32) Vector2 {
	return Vector2HermiteByVector2AndVector2AndVector2AndVector2AndSingle(*value1, *tangent1, *value2, *tangent2, amount)
}

func Vector2NormalizeByVector2(value Vector2) Vector2 {
	value.Normalize()
	return value
}

func Vector2NormalizeByRefVector2AndOutVector2(value *Vector2) Vector2 {
	return Vector2NormalizeByVector2(*value)
}

func Vector2NegateByVector2(value Vector2) Vector2 {
	return Vector2{X: -value.X, Y: -value.Y}
}

func Vector2NegateByRefVector2AndOutVector2(value *Vector2) Vector2 {
	return Vector2NegateByVector2(*value)
}

func Vector2SubtractByVector2AndVector2(value1, value2 Vector2) Vector2 {
	return Vector2{X: value1.X - value2.X, Y: value1.Y - value2.Y}
}

func Vector2SubtractByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2SubtractByVector2AndVector2(*value1, *value2)
}

func Vector2MultiplyByVector2AndVector2(value1, value2 Vector2) Vector2 {
	return Vector2{X: value1.X * value2.X, Y: value1.Y * value2.Y}
}

func Vector2MultiplyByVector2AndSingle(value Vector2, scale float32) Vector2 {
	return Vector2{X: value.X * scale, Y: value.Y * scale}
}

func Vector2MultiplyByRefVector2AndSingleAndOutVector2(value *Vector2, scale float32) Vector2 {
	return Vector2MultiplyByVector2AndSingle(*value, scale)
}

func Vector2MultiplyByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2MultiplyByVector2AndVector2(*value1, *value2)
}

func Vector2DivideByVector2AndVector2(value1, value2 Vector2) Vector2 {
	return Vector2{X: value1.X / value2.X, Y: value1.Y / value2.Y}
}

func Vector2DivideByVector2AndSingle(value Vector2, divider float32) Vector2 {
	factor := float32(1) / divider
	return Vector2{X: value.X * factor, Y: value.Y * factor}
}

func Vector2DivideByRefVector2AndSingleAndOutVector2(value *Vector2, divider float32) Vector2 {
	return Vector2DivideByVector2AndSingle(*value, divider)
}

func Vector2DivideByRefVector2AndRefVector2AndOutVector2(value1, value2 *Vector2) Vector2 {
	return Vector2DivideByVector2AndVector2(*value1, *value2)
}

func Vector2TransformByVector2AndMatrix(position Vector2, matrix Matrix) Vector2 {
	return Vector2TransformByRefVector2AndRefMatrixAndOutVector2(&position, &matrix)
}

func Vector2TransformByRefVector2AndRefMatrixAndOutVector2(position *Vector2, matrix *Matrix) Vector2 {
	x := position.X*matrix.M11 + position.Y*matrix.M21 + matrix.M41
	y := position.X*matrix.M12 + position.Y*matrix.M22 + matrix.M42
	return Vector2{X: x, Y: y}
}

func Vector2TransformByVector2AndQuaternion(value Vector2, rotation Quaternion) Vector2 {
	return Vector2TransformByRefVector2AndRefQuaternionAndOutVector2(&value, &rotation)
}

func Vector2TransformByRefVector2AndRefQuaternionAndOutVector2(value *Vector2, rotation *Quaternion) Vector2 {
	x2, y2, z2 := rotation.X+rotation.X, rotation.Y+rotation.Y, rotation.Z+rotation.Z
	wz2, xx2, xy2 := rotation.W*z2, rotation.X*x2, rotation.X*y2
	yy2, zz2 := rotation.Y*y2, rotation.Z*z2
	return Vector2{
		X: value.X*(1-yy2-zz2) + value.Y*(xy2-wz2),
		Y: value.X*(xy2+wz2) + value.Y*(1-xx2-zz2),
	}
}

func Vector2TransformNormalByVector2AndMatrix(normal Vector2, matrix Matrix) Vector2 {
	return Vector2TransformNormalByRefVector2AndRefMatrixAndOutVector2(&normal, &matrix)
}

func Vector2TransformNormalByRefVector2AndRefMatrixAndOutVector2(normal *Vector2, matrix *Matrix) Vector2 {
	return Vector2{
		X: normal.X*matrix.M11 + normal.Y*matrix.M21,
		Y: normal.X*matrix.M12 + normal.Y*matrix.M22,
	}
}

func Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(source []Vector2, sourceIndex int32, matrix *Matrix, destination []Vector2, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector2TransformByRefVector2AndRefMatrixAndOutVector2(&source[sourceIndex+i], matrix)
	}
}

func Vector2TransformBySliceOfVector2AndRefMatrixAndSliceOfVector2(source []Vector2, matrix *Matrix, destination []Vector2) {
	if source == nil {
		panic("source array is nil")
	}
	Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(source, 0, matrix, destination, 0, int32(len(source)))
}

func Vector2TransformBySliceOfVector2AndInt32AndRefQuaternionAndSliceOfVector2AndInt32AndInt32(source []Vector2, sourceIndex int32, rotation *Quaternion, destination []Vector2, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector2TransformByRefVector2AndRefQuaternionAndOutVector2(&source[sourceIndex+i], rotation)
	}
}

func Vector2TransformBySliceOfVector2AndRefQuaternionAndSliceOfVector2(source []Vector2, rotation *Quaternion, destination []Vector2) {
	if source == nil {
		panic("source array is nil")
	}
	Vector2TransformBySliceOfVector2AndInt32AndRefQuaternionAndSliceOfVector2AndInt32AndInt32(source, 0, rotation, destination, 0, int32(len(source)))
}

func Vector2TransformNormalBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(source []Vector2, sourceIndex int32, matrix *Matrix, destination []Vector2, destinationIndex, length int32) {
	validateTransformSlices(source, sourceIndex, destination, destinationIndex, length)
	for i := int32(0); i < length; i++ {
		destination[destinationIndex+i] = Vector2TransformNormalByRefVector2AndRefMatrixAndOutVector2(&source[sourceIndex+i], matrix)
	}
}

func Vector2TransformNormalBySliceOfVector2AndRefMatrixAndSliceOfVector2(source []Vector2, matrix *Matrix, destination []Vector2) {
	if source == nil {
		panic("source array is nil")
	}
	Vector2TransformNormalBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(source, 0, matrix, destination, 0, int32(len(source)))
}

func (v Vector2) EqualsByVector2(other Vector2) bool { return v.X == other.X && v.Y == other.Y }
func (v Vector2) EqualsByObject(value any) bool {
	other, ok := value.(Vector2)
	return ok && v.EqualsByVector2(other)
}
func (v Vector2) GetHashCode() int32 { return singleHashCode(v.X) + singleHashCode(v.Y) }
func (v Vector2) ToString() string {
	return fmt.Sprintf("{X:%s Y:%s}", formatSingle(v.X), formatSingle(v.Y))
}

func Vector2OperatorAdditionByVector2AndVector2(a, b Vector2) Vector2 {
	return Vector2AddByVector2AndVector2(a, b)
}
func Vector2OperatorSubtractionByVector2AndVector2(a, b Vector2) Vector2 {
	return Vector2SubtractByVector2AndVector2(a, b)
}
func Vector2OperatorUnaryNegationByVector2(value Vector2) Vector2 {
	return Vector2NegateByVector2(value)
}
func Vector2OperatorMultiplyByVector2AndSingle(value Vector2, scale float32) Vector2 {
	return Vector2MultiplyByVector2AndSingle(value, scale)
}
func Vector2OperatorMultiplyBySingleAndVector2(scale float32, value Vector2) Vector2 {
	return Vector2MultiplyByVector2AndSingle(value, scale)
}
func Vector2OperatorMultiplyByVector2AndVector2(a, b Vector2) Vector2 {
	return Vector2MultiplyByVector2AndVector2(a, b)
}
func Vector2OperatorDivisionByVector2AndSingle(value Vector2, divider float32) Vector2 {
	return Vector2DivideByVector2AndSingle(value, divider)
}
func Vector2OperatorDivisionByVector2AndVector2(a, b Vector2) Vector2 {
	return Vector2DivideByVector2AndVector2(a, b)
}
func Vector2OperatorEqualityByVector2AndVector2(a, b Vector2) bool { return a.EqualsByVector2(b) }
func Vector2OperatorInequalityByVector2AndVector2(a, b Vector2) bool {
	return !a.EqualsByVector2(b)
}
