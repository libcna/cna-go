package framework

import "fmt"

// Matrix is XNA's row-major 4x4 binary32 value matrix. Vectors use v*M and
// translation occupies M41-M43.
type Matrix struct {
	M11, M12, M13, M14 float32
	M21, M22, M23, M24 float32
	M31, M32, M33, M34 float32
	M41, M42, M43, M44 float32
}

func NewMatrix(m11, m12, m13, m14, m21, m22, m23, m24, m31, m32, m33, m34, m41, m42, m43, m44 float32) Matrix {
	return Matrix{m11, m12, m13, m14, m21, m22, m23, m24, m31, m32, m33, m34, m41, m42, m43, m44}
}
func MatrixIdentity() Matrix { return Matrix{M11: 1, M22: 1, M33: 1, M44: 1} }

func (m Matrix) Translation() Vector3      { return Vector3{m.M41, m.M42, m.M43} }
func (m *Matrix) SetTranslation(v Vector3) { m.M41 = v.X; m.M42 = v.Y; m.M43 = v.Z }
func (m Matrix) Right() Vector3            { return Vector3{m.M11, m.M12, m.M13} }
func (m *Matrix) SetRight(v Vector3)       { m.M11 = v.X; m.M12 = v.Y; m.M13 = v.Z }
func (m Matrix) Left() Vector3             { return Vector3{-m.M11, -m.M12, -m.M13} }
func (m *Matrix) SetLeft(v Vector3)        { m.M11 = -v.X; m.M12 = -v.Y; m.M13 = -v.Z }
func (m Matrix) Up() Vector3               { return Vector3{m.M21, m.M22, m.M23} }
func (m *Matrix) SetUp(v Vector3)          { m.M21 = v.X; m.M22 = v.Y; m.M23 = v.Z }
func (m Matrix) Down() Vector3             { return Vector3{-m.M21, -m.M22, -m.M23} }
func (m *Matrix) SetDown(v Vector3)        { m.M21 = -v.X; m.M22 = -v.Y; m.M23 = -v.Z }
func (m Matrix) Forward() Vector3          { return Vector3{-m.M31, -m.M32, -m.M33} }
func (m *Matrix) SetForward(v Vector3)     { m.M31 = -v.X; m.M32 = -v.Y; m.M33 = -v.Z }
func (m Matrix) Backward() Vector3         { return Vector3{m.M31, m.M32, m.M33} }
func (m *Matrix) SetBackward(v Vector3)    { m.M31 = v.X; m.M32 = v.Y; m.M33 = v.Z }

func MatrixCreateTranslationByVector3(v Vector3) Matrix {
	return MatrixCreateTranslationBySingleAndSingleAndSingle(v.X, v.Y, v.Z)
}
func MatrixCreateTranslationByRefVector3AndOutMatrix(v *Vector3) Matrix {
	return MatrixCreateTranslationByVector3(*v)
}
func MatrixCreateTranslationBySingleAndSingleAndSingle(x, y, z float32) Matrix {
	r := MatrixIdentity()
	r.M41 = x
	r.M42 = y
	r.M43 = z
	return r
}
func MatrixCreateTranslationBySingleAndSingleAndSingleAndOutMatrix(x, y, z float32) Matrix {
	return MatrixCreateTranslationBySingleAndSingleAndSingle(x, y, z)
}
func MatrixCreateScaleBySingle(s float32) Matrix {
	return MatrixCreateScaleBySingleAndSingleAndSingle(s, s, s)
}
func MatrixCreateScaleBySingleAndOutMatrix(s float32) Matrix { return MatrixCreateScaleBySingle(s) }
func MatrixCreateScaleByVector3(v Vector3) Matrix {
	return MatrixCreateScaleBySingleAndSingleAndSingle(v.X, v.Y, v.Z)
}
func MatrixCreateScaleByRefVector3AndOutMatrix(v *Vector3) Matrix {
	return MatrixCreateScaleByVector3(*v)
}
func MatrixCreateScaleBySingleAndSingleAndSingle(x, y, z float32) Matrix {
	return Matrix{M11: x, M22: y, M33: z, M44: 1}
}
func MatrixCreateScaleBySingleAndSingleAndSingleAndOutMatrix(x, y, z float32) Matrix {
	return MatrixCreateScaleBySingleAndSingleAndSingle(x, y, z)
}
func MatrixCreateRotationXBySingle(radians float32) Matrix {
	c, s := cos32(radians), sin32(radians)
	return Matrix{M11: 1, M22: c, M23: s, M32: -s, M33: c, M44: 1}
}
func MatrixCreateRotationXBySingleAndOutMatrix(radians float32) Matrix {
	return MatrixCreateRotationXBySingle(radians)
}
func MatrixCreateRotationYBySingle(radians float32) Matrix {
	c, s := cos32(radians), sin32(radians)
	return Matrix{M11: c, M13: -s, M22: 1, M31: s, M33: c, M44: 1}
}
func MatrixCreateRotationYBySingleAndOutMatrix(radians float32) Matrix {
	return MatrixCreateRotationYBySingle(radians)
}
func MatrixCreateRotationZBySingle(radians float32) Matrix {
	c, s := cos32(radians), sin32(radians)
	return Matrix{M11: c, M12: s, M21: -s, M22: c, M33: 1, M44: 1}
}
func MatrixCreateRotationZBySingleAndOutMatrix(radians float32) Matrix {
	return MatrixCreateRotationZBySingle(radians)
}

func MatrixCreateFromAxisAngleByVector3AndSingle(axis Vector3, angle float32) Matrix {
	x, y, z := axis.X, axis.Y, axis.Z
	s, c := sin32(angle), cos32(angle)
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	return NewMatrix(
		xx+c*(1-xx), xy-c*xy+s*z, xz-c*xz-s*y, 0,
		xy-c*xy-s*z, yy+c*(1-yy), yz-c*yz+s*x, 0,
		xz-c*xz+s*y, yz-c*yz-s*x, zz+c*(1-zz), 0,
		0, 0, 0, 1)
}
func MatrixCreateFromAxisAngleByRefVector3AndSingleAndOutMatrix(axis *Vector3, angle float32) Matrix {
	return MatrixCreateFromAxisAngleByVector3AndSingle(*axis, angle)
}
func MatrixCreateFromQuaternionByQuaternion(q Quaternion) Matrix {
	xx, yy, zz := q.X*q.X, q.Y*q.Y, q.Z*q.Z
	xy, zw, zx := q.X*q.Y, q.Z*q.W, q.Z*q.X
	yw, yz, xw := q.Y*q.W, q.Y*q.Z, q.X*q.W
	return NewMatrix(1-2*(yy+zz), 2*(xy+zw), 2*(zx-yw), 0, 2*(xy-zw), 1-2*(zz+xx), 2*(yz+xw), 0, 2*(zx+yw), 2*(yz-xw), 1-2*(yy+xx), 0, 0, 0, 0, 1)
}
func MatrixCreateFromQuaternionByRefQuaternionAndOutMatrix(q *Quaternion) Matrix {
	return MatrixCreateFromQuaternionByQuaternion(*q)
}
func MatrixCreateFromYawPitchRollBySingleAndSingleAndSingle(yaw, pitch, roll float32) Matrix {
	return MatrixCreateFromQuaternionByQuaternion(QuaternionCreateFromYawPitchRollBySingleAndSingleAndSingle(yaw, pitch, roll))
}
func MatrixCreateFromYawPitchRollBySingleAndSingleAndSingleAndOutMatrix(yaw, pitch, roll float32) Matrix {
	return MatrixCreateFromYawPitchRollBySingleAndSingleAndSingle(yaw, pitch, roll)
}
func MatrixCreateLookAtByVector3AndVector3AndVector3(position, target, upVector Vector3) Matrix {
	backward := Vector3NormalizeByVector3(Vector3SubtractByVector3AndVector3(position, target))
	right := Vector3NormalizeByVector3(Vector3CrossByVector3AndVector3(upVector, backward))
	up := Vector3CrossByVector3AndVector3(backward, right)
	return NewMatrix(right.X, up.X, backward.X, 0, right.Y, up.Y, backward.Y, 0, right.Z, up.Z, backward.Z, 0, -Vector3DotByVector3AndVector3(right, position), -Vector3DotByVector3AndVector3(up, position), -Vector3DotByVector3AndVector3(backward, position), 1)
}
func MatrixCreateLookAtByRefVector3AndRefVector3AndRefVector3AndOutMatrix(position, target, up *Vector3) Matrix {
	return MatrixCreateLookAtByVector3AndVector3AndVector3(*position, *target, *up)
}
func MatrixCreateWorldByVector3AndVector3AndVector3(position, forward, up Vector3) Matrix {
	backward := Vector3NormalizeByVector3(Vector3NegateByVector3(forward))
	right := Vector3NormalizeByVector3(Vector3CrossByVector3AndVector3(up, backward))
	corrected := Vector3CrossByVector3AndVector3(backward, right)
	return NewMatrix(right.X, right.Y, right.Z, 0, corrected.X, corrected.Y, corrected.Z, 0, backward.X, backward.Y, backward.Z, 0, position.X, position.Y, position.Z, 1)
}
func MatrixCreateWorldByRefVector3AndRefVector3AndRefVector3AndOutMatrix(position, forward, up *Vector3) Matrix {
	return MatrixCreateWorldByVector3AndVector3AndVector3(*position, *forward, *up)
}

func validatePerspective(near, far float32) {
	if near <= 0 || far <= 0 || near >= far {
		panic("perspective plane out of range")
	}
}
func MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(fov, aspect, near, far float32) Matrix {
	if fov <= 0 || fov >= MathHelperPi {
		panic("field of view out of range")
	}
	validatePerspective(near, far)
	yScale := float32(1) / tan32(fov*0.5)
	xScale := yScale / aspect
	depth := far / (near - far)
	return NewMatrix(xScale, 0, 0, 0, 0, yScale, 0, 0, 0, 0, depth, -1, 0, 0, (near*far)/(near-far), 0)
}
func MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingleAndOutMatrix(fov, aspect, near, far float32) Matrix {
	return MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(fov, aspect, near, far)
}
func MatrixCreateOrthographicBySingleAndSingleAndSingleAndSingle(width, height, near, far float32) Matrix {
	r := MatrixIdentity()
	r.M11 = 2 / width
	r.M22 = 2 / height
	r.M33 = 1 / (near - far)
	r.M43 = near / (near - far)
	return r
}
func MatrixCreateOrthographicBySingleAndSingleAndSingleAndSingleAndOutMatrix(width, height, near, far float32) Matrix {
	return MatrixCreateOrthographicBySingleAndSingleAndSingleAndSingle(width, height, near, far)
}
func MatrixCreateOrthographicOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingle(left, right, bottom, top, near, far float32) Matrix {
	r := MatrixIdentity()
	r.M11 = 2 / (right - left)
	r.M22 = 2 / (top - bottom)
	r.M33 = 1 / (near - far)
	r.M41 = (left + right) / (left - right)
	r.M42 = (top + bottom) / (bottom - top)
	r.M43 = near / (near - far)
	return r
}
func MatrixCreateOrthographicOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingleAndOutMatrix(left, right, bottom, top, near, far float32) Matrix {
	return MatrixCreateOrthographicOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingle(left, right, bottom, top, near, far)
}
func MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(width, height, near, far float32) Matrix {
	validatePerspective(near, far)
	return NewMatrix((2*near)/width, 0, 0, 0, 0, (2*near)/height, 0, 0, 0, 0, far/(near-far), -1, 0, 0, (near*far)/(near-far), 0)
}
func MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingleAndOutMatrix(width, height, near, far float32) Matrix {
	return MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(width, height, near, far)
}
func MatrixCreatePerspectiveOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingle(left, right, bottom, top, near, far float32) Matrix {
	validatePerspective(near, far)
	return NewMatrix((2*near)/(right-left), 0, 0, 0, 0, (2*near)/(top-bottom), 0, 0, (left+right)/(right-left), (top+bottom)/(top-bottom), far/(near-far), -1, 0, 0, (near*far)/(near-far), 0)
}
func MatrixCreatePerspectiveOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingleAndOutMatrix(left, right, bottom, top, near, far float32) Matrix {
	return MatrixCreatePerspectiveOffCenterBySingleAndSingleAndSingleAndSingleAndSingleAndSingle(left, right, bottom, top, near, far)
}

func MatrixCreateBillboardByVector3AndVector3AndVector3AndNullableOfVector3(object, camera, up Vector3, cameraForward *Vector3) Matrix {
	facing := Vector3SubtractByVector3AndVector3(object, camera)
	length := facing.LengthSquared()
	if length < 0.0001 {
		if cameraForward != nil {
			facing = Vector3NegateByVector3(*cameraForward)
		} else {
			facing = Vector3Forward()
		}
	} else {
		facing = Vector3MultiplyByVector3AndSingle(facing, 1/sqrt32(length))
	}
	right := Vector3CrossByVector3AndVector3(up, facing)
	right.Normalize()
	corrected := Vector3CrossByVector3AndVector3(facing, right)
	return NewMatrix(right.X, right.Y, right.Z, 0, corrected.X, corrected.Y, corrected.Z, 0, facing.X, facing.Y, facing.Z, 0, object.X, object.Y, object.Z, 1)
}
func MatrixCreateBillboardByRefVector3AndRefVector3AndRefVector3AndNullableOfVector3AndOutMatrix(object, camera, up, cameraForward *Vector3) Matrix {
	return MatrixCreateBillboardByVector3AndVector3AndVector3AndNullableOfVector3(*object, *camera, *up, cameraForward)
}
func MatrixCreateConstrainedBillboardByVector3AndVector3AndVector3AndNullableOfVector3AndNullableOfVector3(object, camera, axis Vector3, cameraForward, objectForward *Vector3) Matrix {
	facing := Vector3SubtractByVector3AndVector3(object, camera)
	length := facing.LengthSquared()
	if length < 0.0001 {
		if cameraForward != nil {
			facing = Vector3NegateByVector3(*cameraForward)
		} else {
			facing = Vector3Forward()
		}
	} else {
		facing = Vector3MultiplyByVector3AndSingle(facing, 1/sqrt32(length))
	}
	up := axis
	alignment := Vector3DotByVector3AndVector3(axis, facing)
	var forward, right Vector3
	if abs32(alignment) > 0.99825466 {
		if objectForward != nil {
			forward = *objectForward
			alignment = Vector3DotByVector3AndVector3(axis, forward)
			if abs32(alignment) > 0.99825466 {
				alignment = axis.X*Vector3Forward().X + axis.Y*Vector3Forward().Y + axis.Z*Vector3Forward().Z
				if abs32(alignment) > 0.99825466 {
					forward = Vector3Right()
				} else {
					forward = Vector3Forward()
				}
			}
		} else {
			alignment = axis.X*Vector3Forward().X + axis.Y*Vector3Forward().Y + axis.Z*Vector3Forward().Z
			if abs32(alignment) > 0.99825466 {
				forward = Vector3Right()
			} else {
				forward = Vector3Forward()
			}
		}
		right = Vector3CrossByVector3AndVector3(axis, forward)
		right.Normalize()
		forward = Vector3CrossByVector3AndVector3(right, axis)
		forward.Normalize()
	} else {
		right = Vector3CrossByVector3AndVector3(axis, facing)
		right.Normalize()
		forward = Vector3CrossByVector3AndVector3(right, up)
		forward.Normalize()
	}
	return NewMatrix(right.X, right.Y, right.Z, 0, up.X, up.Y, up.Z, 0, forward.X, forward.Y, forward.Z, 0, object.X, object.Y, object.Z, 1)
}
func MatrixCreateConstrainedBillboardByRefVector3AndRefVector3AndRefVector3AndNullableOfVector3AndNullableOfVector3AndOutMatrix(object, camera, axis, cameraForward, objectForward *Vector3) Matrix {
	return MatrixCreateConstrainedBillboardByVector3AndVector3AndVector3AndNullableOfVector3AndNullableOfVector3(*object, *camera, *axis, cameraForward, objectForward)
}

func MatrixCreateShadowByVector3AndPlane(light Vector3, plane Plane) Matrix {
	n := PlaneNormalizeByPlane(plane)
	dot := n.Normal.X*light.X + n.Normal.Y*light.Y + n.Normal.Z*light.Z
	x, y, z, d := -n.Normal.X, -n.Normal.Y, -n.Normal.Z, -n.D
	return NewMatrix(x*light.X+dot, x*light.Y, x*light.Z, 0, y*light.X, y*light.Y+dot, y*light.Z, 0, z*light.X, z*light.Y, z*light.Z+dot, 0, d*light.X, d*light.Y, d*light.Z, dot)
}
func MatrixCreateShadowByRefVector3AndRefPlaneAndOutMatrix(light *Vector3, plane *Plane) Matrix {
	return MatrixCreateShadowByVector3AndPlane(*light, *plane)
}
func matrixReflectionNormalized(p Plane) Matrix {
	x, y, z := p.Normal.X, p.Normal.Y, p.Normal.Z
	dx, dy, dz := -2*x, -2*y, -2*z
	return NewMatrix(dx*x+1, dy*x, dz*x, 0, dx*y, dy*y+1, dz*y, 0, dx*z, dy*z, dz*z+1, 0, dx*p.D, dy*p.D, dz*p.D, 1)
}
func MatrixCreateReflectionByPlane(p Plane) Matrix {
	p.Normalize()
	return matrixReflectionNormalized(p)
}
func MatrixCreateReflectionByRefPlaneAndOutMatrix(p *Plane) Matrix {
	normalized := PlaneNormalizeByPlane(*p)
	p.Normalize()
	return matrixReflectionNormalized(normalized)
}

func (m Matrix) Determinant() float32 {
	a := (m.M33 * m.M44) - (m.M34 * m.M43)
	b := (m.M32 * m.M44) - (m.M34 * m.M42)
	c := (m.M32 * m.M43) - (m.M33 * m.M42)
	d := (m.M31 * m.M44) - (m.M34 * m.M41)
	e := (m.M31 * m.M43) - (m.M33 * m.M41)
	f := (m.M31 * m.M42) - (m.M32 * m.M41)
	return m.M11*((m.M22*a-m.M23*b)+m.M24*c) - m.M12*((m.M21*a-m.M23*d)+m.M24*e) + m.M13*((m.M21*b-m.M22*d)+m.M24*f) - m.M14*((m.M21*c-m.M22*e)+m.M23*f)
}
func MatrixTransposeByMatrix(m Matrix) Matrix {
	return NewMatrix(m.M11, m.M21, m.M31, m.M41, m.M12, m.M22, m.M32, m.M42, m.M13, m.M23, m.M33, m.M43, m.M14, m.M24, m.M34, m.M44)
}
func MatrixTransposeByRefMatrixAndOutMatrix(m *Matrix) Matrix { return MatrixTransposeByMatrix(*m) }

func matrixProduct(a, b Matrix) Matrix {
	return NewMatrix(
		a.M11*b.M11+a.M12*b.M21+a.M13*b.M31+a.M14*b.M41, a.M11*b.M12+a.M12*b.M22+a.M13*b.M32+a.M14*b.M42, a.M11*b.M13+a.M12*b.M23+a.M13*b.M33+a.M14*b.M43, a.M11*b.M14+a.M12*b.M24+a.M13*b.M34+a.M14*b.M44,
		a.M21*b.M11+a.M22*b.M21+a.M23*b.M31+a.M24*b.M41, a.M21*b.M12+a.M22*b.M22+a.M23*b.M32+a.M24*b.M42, a.M21*b.M13+a.M22*b.M23+a.M23*b.M33+a.M24*b.M43, a.M21*b.M14+a.M22*b.M24+a.M23*b.M34+a.M24*b.M44,
		a.M31*b.M11+a.M32*b.M21+a.M33*b.M31+a.M34*b.M41, a.M31*b.M12+a.M32*b.M22+a.M33*b.M32+a.M34*b.M42, a.M31*b.M13+a.M32*b.M23+a.M33*b.M33+a.M34*b.M43, a.M31*b.M14+a.M32*b.M24+a.M33*b.M34+a.M34*b.M44,
		a.M41*b.M11+a.M42*b.M21+a.M43*b.M31+a.M44*b.M41, a.M41*b.M12+a.M42*b.M22+a.M43*b.M32+a.M44*b.M42, a.M41*b.M13+a.M42*b.M23+a.M43*b.M33+a.M44*b.M43, a.M41*b.M14+a.M42*b.M24+a.M43*b.M34+a.M44*b.M44)
}
func matrixMap2(a, b Matrix, f func(float32, float32) float32) Matrix {
	return NewMatrix(f(a.M11, b.M11), f(a.M12, b.M12), f(a.M13, b.M13), f(a.M14, b.M14), f(a.M21, b.M21), f(a.M22, b.M22), f(a.M23, b.M23), f(a.M24, b.M24), f(a.M31, b.M31), f(a.M32, b.M32), f(a.M33, b.M33), f(a.M34, b.M34), f(a.M41, b.M41), f(a.M42, b.M42), f(a.M43, b.M43), f(a.M44, b.M44))
}
func matrixScale(m Matrix, s float32) Matrix {
	return NewMatrix(m.M11*s, m.M12*s, m.M13*s, m.M14*s, m.M21*s, m.M22*s, m.M23*s, m.M24*s, m.M31*s, m.M32*s, m.M33*s, m.M34*s, m.M41*s, m.M42*s, m.M43*s, m.M44*s)
}
func MatrixAddByMatrixAndMatrix(a, b Matrix) Matrix {
	return matrixMap2(a, b, func(x, y float32) float32 { return x + y })
}
func MatrixAddByRefMatrixAndRefMatrixAndOutMatrix(a, b *Matrix) Matrix {
	return MatrixAddByMatrixAndMatrix(*a, *b)
}
func MatrixSubtractByMatrixAndMatrix(a, b Matrix) Matrix {
	return matrixMap2(a, b, func(x, y float32) float32 { return x - y })
}
func MatrixSubtractByRefMatrixAndRefMatrixAndOutMatrix(a, b *Matrix) Matrix {
	return MatrixSubtractByMatrixAndMatrix(*a, *b)
}
func MatrixNegateByMatrix(m Matrix) Matrix                 { return matrixScale(m, -1) }
func MatrixNegateByRefMatrixAndOutMatrix(m *Matrix) Matrix { return MatrixNegateByMatrix(*m) }
func MatrixMultiplyByMatrixAndMatrix(a, b Matrix) Matrix   { return matrixProduct(a, b) }
func MatrixMultiplyByRefMatrixAndRefMatrixAndOutMatrix(a, b *Matrix) Matrix {
	return matrixProduct(*a, *b)
}
func MatrixMultiplyByMatrixAndSingle(m Matrix, s float32) Matrix { return matrixScale(m, s) }
func MatrixMultiplyByRefMatrixAndSingleAndOutMatrix(m *Matrix, s float32) Matrix {
	return matrixScale(*m, s)
}
func MatrixDivideByMatrixAndMatrix(a, b Matrix) Matrix {
	return matrixMap2(a, b, func(x, y float32) float32 { return x / y })
}
func MatrixDivideByRefMatrixAndRefMatrixAndOutMatrix(a, b *Matrix) Matrix {
	return MatrixDivideByMatrixAndMatrix(*a, *b)
}
func MatrixDivideByMatrixAndSingle(m Matrix, d float32) Matrix { return matrixScale(m, 1/d) }
func MatrixDivideByRefMatrixAndSingleAndOutMatrix(m *Matrix, d float32) Matrix {
	return MatrixDivideByMatrixAndSingle(*m, d)
}
func MatrixLerpByMatrixAndMatrixAndSingle(a, b Matrix, t float32) Matrix {
	return matrixMap2(a, b, func(x, y float32) float32 { return x + (y-x)*t })
}
func MatrixLerpByRefMatrixAndRefMatrixAndSingleAndOutMatrix(a, b *Matrix, t float32) Matrix {
	return MatrixLerpByMatrixAndMatrixAndSingle(*a, *b, t)
}

func MatrixTransformByMatrixAndQuaternion(m Matrix, q Quaternion) Matrix {
	return MatrixTransformByRefMatrixAndRefQuaternionAndOutMatrix(&m, &q)
}
func MatrixTransformByRefMatrixAndRefQuaternionAndOutMatrix(v *Matrix, q *Quaternion) Matrix {
	x2, y2, z2 := q.X+q.X, q.Y+q.Y, q.Z+q.Z
	wx2, wy2, wz2 := q.W*x2, q.W*y2, q.W*z2
	xx2, xy2, xz2 := q.X*x2, q.X*y2, q.X*z2
	yy2, yz2, zz2 := q.Y*y2, q.Y*z2, q.Z*z2
	a, b, c := 1-yy2-zz2, xy2-wz2, xz2+wy2
	d, e, f := xy2+wz2, 1-xx2-zz2, yz2-wx2
	g, h, i := xz2-wy2, yz2+wx2, 1-xx2-yy2
	return NewMatrix(v.M11*a+v.M12*b+v.M13*c, v.M11*d+v.M12*e+v.M13*f, v.M11*g+v.M12*h+v.M13*i, v.M14, v.M21*a+v.M22*b+v.M23*c, v.M21*d+v.M22*e+v.M23*f, v.M21*g+v.M22*h+v.M23*i, v.M24, v.M31*a+v.M32*b+v.M33*c, v.M31*d+v.M32*e+v.M33*f, v.M31*g+v.M32*h+v.M33*i, v.M34, v.M41*a+v.M42*b+v.M43*c, v.M41*d+v.M42*e+v.M43*f, v.M41*g+v.M42*h+v.M43*i, v.M44)
}

func (m Matrix) EqualsByMatrix(o Matrix) bool { return m == o }
func (m Matrix) EqualsByObject(value any) bool {
	o, ok := value.(Matrix)
	return ok && m.EqualsByMatrix(o)
}
func (m Matrix) GetHashCode() int32 {
	return singleHashCode(m.M11) + singleHashCode(m.M12) + singleHashCode(m.M13) + singleHashCode(m.M14) + singleHashCode(m.M21) + singleHashCode(m.M22) + singleHashCode(m.M23) + singleHashCode(m.M24) + singleHashCode(m.M31) + singleHashCode(m.M32) + singleHashCode(m.M33) + singleHashCode(m.M34) + singleHashCode(m.M41) + singleHashCode(m.M42) + singleHashCode(m.M43) + singleHashCode(m.M44)
}
func (m Matrix) ToString() string {
	return fmt.Sprintf("{ {M11:%s M12:%s M13:%s M14:%s} {M21:%s M22:%s M23:%s M24:%s} {M31:%s M32:%s M33:%s M34:%s} {M41:%s M42:%s M43:%s M44:%s} }", formatSingle(m.M11), formatSingle(m.M12), formatSingle(m.M13), formatSingle(m.M14), formatSingle(m.M21), formatSingle(m.M22), formatSingle(m.M23), formatSingle(m.M24), formatSingle(m.M31), formatSingle(m.M32), formatSingle(m.M33), formatSingle(m.M34), formatSingle(m.M41), formatSingle(m.M42), formatSingle(m.M43), formatSingle(m.M44))
}

func MatrixOperatorAdditionByMatrixAndMatrix(a, b Matrix) Matrix {
	return MatrixAddByMatrixAndMatrix(a, b)
}
func MatrixOperatorSubtractionByMatrixAndMatrix(a, b Matrix) Matrix {
	return MatrixSubtractByMatrixAndMatrix(a, b)
}
func MatrixOperatorUnaryNegationByMatrix(m Matrix) Matrix                { return MatrixNegateByMatrix(m) }
func MatrixOperatorMultiplyByMatrixAndMatrix(a, b Matrix) Matrix         { return matrixProduct(a, b) }
func MatrixOperatorMultiplyByMatrixAndSingle(m Matrix, s float32) Matrix { return matrixScale(m, s) }
func MatrixOperatorMultiplyBySingleAndMatrix(s float32, m Matrix) Matrix { return matrixScale(m, s) }
func MatrixOperatorDivisionByMatrixAndMatrix(a, b Matrix) Matrix {
	return MatrixDivideByMatrixAndMatrix(a, b)
}
func MatrixOperatorDivisionByMatrixAndSingle(m Matrix, d float32) Matrix {
	return MatrixDivideByMatrixAndSingle(m, d)
}
func MatrixOperatorEqualityByMatrixAndMatrix(a, b Matrix) bool   { return a.EqualsByMatrix(b) }
func MatrixOperatorInequalityByMatrixAndMatrix(a, b Matrix) bool { return !a.EqualsByMatrix(b) }
