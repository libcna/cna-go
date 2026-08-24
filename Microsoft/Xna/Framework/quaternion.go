package framework

import (
	"fmt"
	"math"
)

// Quaternion is XNA's four-component rotation value.
type Quaternion struct{ X, Y, Z, W float32 }

func NewQuaternionByVector3AndSingle(vector Vector3, scalar float32) Quaternion {
	return Quaternion{vector.X, vector.Y, vector.Z, scalar}
}
func NewQuaternionBySingleAndSingleAndSingleAndSingle(x, y, z, w float32) Quaternion {
	return Quaternion{x, y, z, w}
}
func QuaternionIdentity() Quaternion { return Quaternion{W: 1} }

func (q Quaternion) LengthSquared() float32 { return q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W }
func (q Quaternion) Length() float32        { return sqrt32(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W) }
func (q *Quaternion) Normalize() {
	f := float32(1) / sqrt32(q.LengthSquared())
	q.X *= f
	q.Y *= f
	q.Z *= f
	q.W *= f
}
func (q *Quaternion) Conjugate() { q.X = -q.X; q.Y = -q.Y; q.Z = -q.Z }

func QuaternionCreateFromAxisAngleByVector3AndSingle(axis Vector3, angle float32) Quaternion {
	half := angle * 0.5
	s, c := sin32(half), cos32(half)
	return Quaternion{axis.X * s, axis.Y * s, axis.Z * s, c}
}
func QuaternionCreateFromAxisAngleByRefVector3AndSingleAndOutQuaternion(axis *Vector3, angle float32) Quaternion {
	return QuaternionCreateFromAxisAngleByVector3AndSingle(*axis, angle)
}
func QuaternionCreateFromYawPitchRollBySingleAndSingleAndSingle(yaw, pitch, roll float32) Quaternion {
	hr := roll * 0.5
	sr, cr := sin32(hr), cos32(hr)
	hp := pitch * 0.5
	sp, cp := sin32(hp), cos32(hp)
	hy := yaw * 0.5
	sy, cy := sin32(hy), cos32(hy)
	return Quaternion{
		X: cy*sp*cr + sy*cp*sr,
		Y: sy*cp*cr - cy*sp*sr,
		Z: cy*cp*sr - sy*sp*cr,
		W: cy*cp*cr + sy*sp*sr,
	}
}
func QuaternionCreateFromYawPitchRollBySingleAndSingleAndSingleAndOutQuaternion(yaw, pitch, roll float32) Quaternion {
	return QuaternionCreateFromYawPitchRollBySingleAndSingleAndSingle(yaw, pitch, roll)
}
func QuaternionCreateFromRotationMatrixByMatrix(m Matrix) Quaternion {
	trace := m.M11 + m.M22 + m.M33
	if trace > 0 {
		root := sqrt32(trace + 1)
		factor := float32(0.5) / root
		return Quaternion{(m.M23 - m.M32) * factor, (m.M31 - m.M13) * factor, (m.M12 - m.M21) * factor, root * 0.5}
	}
	if m.M11 >= m.M22 && m.M11 >= m.M33 {
		root := sqrt32(1 + m.M11 - m.M22 - m.M33)
		factor := float32(0.5) / root
		return Quaternion{0.5 * root, (m.M12 + m.M21) * factor, (m.M13 + m.M31) * factor, (m.M23 - m.M32) * factor}
	}
	if m.M22 > m.M33 {
		root := sqrt32(1 + m.M22 - m.M11 - m.M33)
		factor := float32(0.5) / root
		return Quaternion{(m.M21 + m.M12) * factor, 0.5 * root, (m.M32 + m.M23) * factor, (m.M31 - m.M13) * factor}
	}
	root := sqrt32(1 + m.M33 - m.M11 - m.M22)
	factor := float32(0.5) / root
	return Quaternion{(m.M31 + m.M13) * factor, (m.M32 + m.M23) * factor, 0.5 * root, (m.M12 - m.M21) * factor}
}
func QuaternionCreateFromRotationMatrixByRefMatrixAndOutQuaternion(m *Matrix) Quaternion {
	return QuaternionCreateFromRotationMatrixByMatrix(*m)
}

func QuaternionNormalizeByQuaternion(q Quaternion) Quaternion { q.Normalize(); return q }
func QuaternionNormalizeByRefQuaternionAndOutQuaternion(q *Quaternion) Quaternion {
	return QuaternionNormalizeByQuaternion(*q)
}
func QuaternionConjugateByQuaternion(q Quaternion) Quaternion {
	return Quaternion{-q.X, -q.Y, -q.Z, q.W}
}
func QuaternionConjugateByRefQuaternionAndOutQuaternion(q *Quaternion) Quaternion {
	return QuaternionConjugateByQuaternion(*q)
}
func QuaternionInverseByQuaternion(q Quaternion) Quaternion {
	f := float32(1) / q.LengthSquared()
	return Quaternion{-q.X * f, -q.Y * f, -q.Z * f, q.W * f}
}
func QuaternionInverseByRefQuaternionAndOutQuaternion(q *Quaternion) Quaternion {
	return QuaternionInverseByQuaternion(*q)
}
func QuaternionDotByQuaternionAndQuaternion(a, b Quaternion) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}
func QuaternionDotByRefQuaternionAndRefQuaternionAndOutSingle(a, b *Quaternion) float32 {
	return QuaternionDotByQuaternionAndQuaternion(*a, *b)
}
func QuaternionConcatenateByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return quaternionProduct(b, a)
}
func QuaternionConcatenateByRefQuaternionAndRefQuaternionAndOutQuaternion(a, b *Quaternion) Quaternion {
	return QuaternionConcatenateByQuaternionAndQuaternion(*a, *b)
}
func QuaternionLerpByQuaternionAndQuaternionAndSingle(a, b Quaternion, amount float32) Quaternion {
	inv := float32(1) - amount
	var r Quaternion
	if QuaternionDotByQuaternionAndQuaternion(a, b) >= 0 {
		r = Quaternion{inv*a.X + amount*b.X, inv*a.Y + amount*b.Y, inv*a.Z + amount*b.Z, inv*a.W + amount*b.W}
	} else {
		r = Quaternion{inv*a.X - amount*b.X, inv*a.Y - amount*b.Y, inv*a.Z - amount*b.Z, inv*a.W - amount*b.W}
	}
	return QuaternionNormalizeByQuaternion(r)
}
func QuaternionLerpByRefQuaternionAndRefQuaternionAndSingleAndOutQuaternion(a, b *Quaternion, amount float32) Quaternion {
	return QuaternionLerpByQuaternionAndQuaternionAndSingle(*a, *b, amount)
}
func QuaternionSlerpByQuaternionAndQuaternionAndSingle(a, b Quaternion, amount float32) Quaternion {
	cosOmega := QuaternionDotByQuaternionAndQuaternion(a, b)
	flip := false
	if cosOmega < 0 {
		flip = true
		cosOmega = -cosOmega
	}
	var w1, w2 float32
	if cosOmega > 0.999999 {
		w1 = 1 - amount
		if flip {
			w2 = -amount
		} else {
			w2 = amount
		}
	} else {
		omega := acos32(cosOmega)
		inverseSin := float32(1.0 / math.Sin(float64(omega)))
		w1 = sin32((1-amount)*omega) * inverseSin
		if flip {
			w2 = -sin32(amount*omega) * inverseSin
		} else {
			w2 = sin32(amount*omega) * inverseSin
		}
	}
	return Quaternion{w1*a.X + w2*b.X, w1*a.Y + w2*b.Y, w1*a.Z + w2*b.Z, w1*a.W + w2*b.W}
}
func QuaternionSlerpByRefQuaternionAndRefQuaternionAndSingleAndOutQuaternion(a, b *Quaternion, amount float32) Quaternion {
	return QuaternionSlerpByQuaternionAndQuaternionAndSingle(*a, *b, amount)
}

func quaternionProduct(a, b Quaternion) Quaternion {
	crossX := a.Y*b.Z - a.Z*b.Y
	crossY := a.Z*b.X - a.X*b.Z
	crossZ := a.X*b.Y - a.Y*b.X
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z
	return Quaternion{a.X*b.W + b.X*a.W + crossX, a.Y*b.W + b.Y*a.W + crossY, a.Z*b.W + b.Z*a.W + crossZ, a.W*b.W - dot}
}
func QuaternionAddByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return Quaternion{a.X + b.X, a.Y + b.Y, a.Z + b.Z, a.W + b.W}
}
func QuaternionAddByRefQuaternionAndRefQuaternionAndOutQuaternion(a, b *Quaternion) Quaternion {
	return QuaternionAddByQuaternionAndQuaternion(*a, *b)
}
func QuaternionSubtractByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return Quaternion{a.X - b.X, a.Y - b.Y, a.Z - b.Z, a.W - b.W}
}
func QuaternionSubtractByRefQuaternionAndRefQuaternionAndOutQuaternion(a, b *Quaternion) Quaternion {
	return QuaternionSubtractByQuaternionAndQuaternion(*a, *b)
}
func QuaternionNegateByQuaternion(q Quaternion) Quaternion { return Quaternion{-q.X, -q.Y, -q.Z, -q.W} }
func QuaternionNegateByRefQuaternionAndOutQuaternion(q *Quaternion) Quaternion {
	return QuaternionNegateByQuaternion(*q)
}
func QuaternionMultiplyByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return quaternionProduct(a, b)
}
func QuaternionMultiplyByRefQuaternionAndRefQuaternionAndOutQuaternion(a, b *Quaternion) Quaternion {
	return quaternionProduct(*a, *b)
}
func QuaternionMultiplyByQuaternionAndSingle(q Quaternion, s float32) Quaternion {
	return Quaternion{q.X * s, q.Y * s, q.Z * s, q.W * s}
}
func QuaternionMultiplyByRefQuaternionAndSingleAndOutQuaternion(q *Quaternion, s float32) Quaternion {
	return QuaternionMultiplyByQuaternionAndSingle(*q, s)
}
func QuaternionDivideByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return quaternionProduct(a, QuaternionInverseByQuaternion(b))
}
func QuaternionDivideByRefQuaternionAndRefQuaternionAndOutQuaternion(a, b *Quaternion) Quaternion {
	return QuaternionDivideByQuaternionAndQuaternion(*a, *b)
}

func (q Quaternion) EqualsByQuaternion(o Quaternion) bool {
	return q.X == o.X && q.Y == o.Y && q.Z == o.Z && q.W == o.W
}
func (q Quaternion) EqualsByObject(value any) bool {
	o, ok := value.(Quaternion)
	return ok && q.EqualsByQuaternion(o)
}
func (q Quaternion) GetHashCode() int32 {
	return singleHashCode(q.X) + singleHashCode(q.Y) + singleHashCode(q.Z) + singleHashCode(q.W)
}
func (q Quaternion) ToString() string {
	return fmt.Sprintf("{X:%s Y:%s Z:%s W:%s}", formatSingle(q.X), formatSingle(q.Y), formatSingle(q.Z), formatSingle(q.W))
}

func QuaternionOperatorAdditionByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return QuaternionAddByQuaternionAndQuaternion(a, b)
}
func QuaternionOperatorSubtractionByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return QuaternionSubtractByQuaternionAndQuaternion(a, b)
}
func QuaternionOperatorUnaryNegationByQuaternion(q Quaternion) Quaternion {
	return QuaternionNegateByQuaternion(q)
}
func QuaternionOperatorMultiplyByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return quaternionProduct(a, b)
}
func QuaternionOperatorMultiplyByQuaternionAndSingle(q Quaternion, s float32) Quaternion {
	return QuaternionMultiplyByQuaternionAndSingle(q, s)
}
func QuaternionOperatorDivisionByQuaternionAndQuaternion(a, b Quaternion) Quaternion {
	return QuaternionDivideByQuaternionAndQuaternion(a, b)
}
func QuaternionOperatorEqualityByQuaternionAndQuaternion(a, b Quaternion) bool {
	return a.EqualsByQuaternion(b)
}
func QuaternionOperatorInequalityByQuaternionAndQuaternion(a, b Quaternion) bool {
	return !a.EqualsByQuaternion(b)
}
