package framework

import "fmt"

// Plane is a value plane represented by a normal and distance coefficient.
type Plane struct {
	Normal Vector3
	D      float32
}

func NewPlaneByVector3AndVector3AndVector3(a, b, c Vector3) Plane {
	n := Vector3NormalizeByVector3(Vector3CrossByVector3AndVector3(Vector3SubtractByVector3AndVector3(b, a), Vector3SubtractByVector3AndVector3(c, a)))
	return Plane{n, -Vector3DotByVector3AndVector3(n, a)}
}
func NewPlaneByVector3AndSingle(normal Vector3, d float32) Plane { return Plane{normal, d} }
func NewPlaneByVector4(v Vector4) Plane                          { return Plane{Vector3{v.X, v.Y, v.Z}, v.W} }
func NewPlaneBySingleAndSingleAndSingleAndSingle(a, b, c, d float32) Plane {
	return Plane{Vector3{a, b, c}, d}
}

func (p Plane) DotByVector4(v Vector4) float32 {
	return p.Normal.X*v.X + p.Normal.Y*v.Y + p.Normal.Z*v.Z + p.D*v.W
}
func (p Plane) DotByRefVector4AndOutSingle(v *Vector4) float32 { return p.DotByVector4(*v) }
func (p Plane) DotCoordinateByVector3(v Vector3) float32 {
	return p.Normal.X*v.X + p.Normal.Y*v.Y + p.Normal.Z*v.Z + p.D
}
func (p Plane) DotCoordinateByRefVector3AndOutSingle(v *Vector3) float32 {
	return p.DotCoordinateByVector3(*v)
}
func (p Plane) DotNormalByVector3(v Vector3) float32 {
	return p.Normal.X*v.X + p.Normal.Y*v.Y + p.Normal.Z*v.Z
}
func (p Plane) DotNormalByRefVector3AndOutSingle(v *Vector3) float32 { return p.DotNormalByVector3(*v) }
func (p *Plane) Normalize() {
	length := p.Normal.X*p.Normal.X + p.Normal.Y*p.Normal.Y + p.Normal.Z*p.Normal.Z
	if !(abs32(length-1) < 1.1920929e-7) {
		f := float32(1) / sqrt32(length)
		p.Normal.X *= f
		p.Normal.Y *= f
		p.Normal.Z *= f
		p.D *= f
	}
}
func PlaneNormalizeByPlane(p Plane) Plane                { p.Normalize(); return p }
func PlaneNormalizeByRefPlaneAndOutPlane(p *Plane) Plane { return PlaneNormalizeByPlane(*p) }

func (p Plane) IntersectsByBoundingSphere(s BoundingSphere) PlaneIntersectionType {
	d := s.Center.X*p.Normal.X + s.Center.Y*p.Normal.Y + s.Center.Z*p.Normal.Z + p.D
	if d > s.Radius {
		return PlaneIntersectionTypeFront
	}
	if d < -s.Radius {
		return PlaneIntersectionTypeBack
	}
	return PlaneIntersectionTypeIntersecting
}
func (p Plane) IntersectsByRefBoundingSphereAndOutPlaneIntersectionType(s *BoundingSphere) PlaneIntersectionType {
	return p.IntersectsByBoundingSphere(*s)
}
func (p Plane) IntersectsByBoundingBox(b BoundingBox) PlaneIntersectionType {
	negative, positive := Vector3{}, Vector3{}
	if p.Normal.X >= 0 {
		negative.X = b.Min.X
		positive.X = b.Max.X
	} else {
		negative.X = b.Max.X
		positive.X = b.Min.X
	}
	if p.Normal.Y >= 0 {
		negative.Y = b.Min.Y
		positive.Y = b.Max.Y
	} else {
		negative.Y = b.Max.Y
		positive.Y = b.Min.Y
	}
	if p.Normal.Z >= 0 {
		negative.Z = b.Min.Z
		positive.Z = b.Max.Z
	} else {
		negative.Z = b.Max.Z
		positive.Z = b.Min.Z
	}
	d := p.Normal.X*negative.X + p.Normal.Y*negative.Y + p.Normal.Z*negative.Z + p.D
	if d > 0 {
		return PlaneIntersectionTypeFront
	}
	d = p.Normal.X*positive.X + p.Normal.Y*positive.Y + p.Normal.Z*positive.Z + p.D
	if d < 0 {
		return PlaneIntersectionTypeBack
	}
	return PlaneIntersectionTypeIntersecting
}
func (p Plane) IntersectsByRefBoundingBoxAndOutPlaneIntersectionType(b *BoundingBox) PlaneIntersectionType {
	return p.IntersectsByBoundingBox(*b)
}
func (p Plane) IntersectsByBoundingFrustum(f *BoundingFrustum) PlaneIntersectionType {
	if f == nil {
		panic("frustum is nil")
	}
	return f.IntersectsByPlane(p)
}

func PlaneTransformByPlaneAndMatrix(p Plane, m Matrix) Plane {
	return PlaneTransformByRefPlaneAndRefMatrixAndOutPlane(&p, &m)
}
func PlaneTransformByRefPlaneAndRefMatrixAndOutPlane(p *Plane, m *Matrix) Plane {
	inv := MatrixInvertByMatrix(*m)
	x, y, z, d := p.Normal.X, p.Normal.Y, p.Normal.Z, p.D
	return NewPlaneBySingleAndSingleAndSingleAndSingle(x*inv.M11+y*inv.M12+z*inv.M13+d*inv.M14, x*inv.M21+y*inv.M22+z*inv.M23+d*inv.M24, x*inv.M31+y*inv.M32+z*inv.M33+d*inv.M34, x*inv.M41+y*inv.M42+z*inv.M43+d*inv.M44)
}
func PlaneTransformByPlaneAndQuaternion(p Plane, q Quaternion) Plane {
	return PlaneTransformByRefPlaneAndRefQuaternionAndOutPlane(&p, &q)
}
func PlaneTransformByRefPlaneAndRefQuaternionAndOutPlane(p *Plane, q *Quaternion) Plane {
	n := Vector3TransformByRefVector3AndRefQuaternionAndOutVector3(&p.Normal, q)
	return Plane{n, p.D}
}

func (p Plane) EqualsByPlane(o Plane) bool { return p.Normal.EqualsByVector3(o.Normal) && p.D == o.D }
func (p Plane) EqualsByObject(value any) bool {
	o, ok := value.(Plane)
	return ok && p.EqualsByPlane(o)
}
func (p Plane) GetHashCode() int32 { return p.Normal.GetHashCode() + singleHashCode(p.D) }
func (p Plane) ToString() string {
	return fmt.Sprintf("{Normal:%s D:%s}", p.Normal.ToString(), formatSingle(p.D))
}
func PlaneOperatorEqualityByPlaneAndPlane(a, b Plane) bool   { return a.EqualsByPlane(b) }
func PlaneOperatorInequalityByPlaneAndPlane(a, b Plane) bool { return !a.EqualsByPlane(b) }
