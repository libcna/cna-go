package framework

import "fmt"

// BoundingSphere is an XNA center/radius value; negative radii are rejected.
type BoundingSphere struct {
	Center Vector3
	Radius float32
}

func NewBoundingSphere(center Vector3, radius float32) BoundingSphere {
	if radius < 0 {
		panic("sphere radius is negative")
	}
	return BoundingSphere{center, radius}
}
func (s BoundingSphere) ContainsByVector3(p Vector3) ContainmentType {
	if Vector3DistanceSquaredByVector3AndVector3(s.Center, p) < s.Radius*s.Radius {
		return ContainmentTypeContains
	}
	return ContainmentTypeDisjoint
}
func (s BoundingSphere) ContainsByRefVector3AndOutContainmentType(p *Vector3) ContainmentType {
	return s.ContainsByVector3(*p)
}
func (s BoundingSphere) ContainsByBoundingSphere(o BoundingSphere) ContainmentType {
	d := Vector3DistanceByVector3AndVector3(s.Center, o.Center)
	if !(s.Radius+o.Radius >= d) {
		return ContainmentTypeDisjoint
	}
	if s.Radius-o.Radius >= d {
		return ContainmentTypeContains
	}
	return ContainmentTypeIntersects
}
func (s BoundingSphere) ContainsByRefBoundingSphereAndOutContainmentType(o *BoundingSphere) ContainmentType {
	return s.ContainsByBoundingSphere(*o)
}
func (s BoundingSphere) ContainsByBoundingBox(b BoundingBox) ContainmentType {
	if !b.IntersectsByBoundingSphere(s) {
		return ContainmentTypeDisjoint
	}
	r2 := s.Radius * s.Radius
	for _, corner := range b.GetCornersByNone() {
		if Vector3SubtractByVector3AndVector3(s.Center, corner).LengthSquared() > r2 {
			return ContainmentTypeIntersects
		}
	}
	return ContainmentTypeContains
}
func (s BoundingSphere) ContainsByRefBoundingBoxAndOutContainmentType(b *BoundingBox) ContainmentType {
	return s.ContainsByBoundingBox(*b)
}
func (s BoundingSphere) ContainsByBoundingFrustum(f *BoundingFrustum) ContainmentType {
	if f == nil {
		panic("frustum is nil")
	}
	all := true
	for _, corner := range f.GetCornersByNone() {
		if s.ContainsByVector3(corner) == ContainmentTypeDisjoint {
			all = false
			break
		}
	}
	if all {
		return ContainmentTypeContains
	}
	if s.IntersectsByBoundingFrustum(f) {
		return ContainmentTypeIntersects
	}
	return ContainmentTypeDisjoint
}
func (s BoundingSphere) IntersectsByBoundingBox(b BoundingBox) bool {
	closest := Vector3ClampByVector3AndVector3AndVector3(s.Center, b.Min, b.Max)
	return !(Vector3DistanceSquaredByVector3AndVector3(s.Center, closest) > s.Radius*s.Radius)
}
func (s BoundingSphere) IntersectsByRefBoundingBoxAndOutBoolean(b *BoundingBox) bool {
	return s.IntersectsByBoundingBox(*b)
}
func (s BoundingSphere) IntersectsByBoundingSphere(o BoundingSphere) bool {
	d := Vector3DistanceSquaredByVector3AndVector3(s.Center, o.Center)
	return s.Radius*s.Radius+2*s.Radius*o.Radius+o.Radius*o.Radius > d
}
func (s BoundingSphere) IntersectsByRefBoundingSphereAndOutBoolean(o *BoundingSphere) bool {
	return s.IntersectsByBoundingSphere(*o)
}
func (s BoundingSphere) IntersectsByBoundingFrustum(f *BoundingFrustum) bool {
	if f == nil {
		panic("frustum is nil")
	}
	return f.IntersectsByBoundingSphere(s)
}
func (s BoundingSphere) IntersectsByPlane(p Plane) PlaneIntersectionType {
	return p.IntersectsByBoundingSphere(s)
}
func (s BoundingSphere) IntersectsByRefPlaneAndOutPlaneIntersectionType(p *Plane) PlaneIntersectionType {
	return s.IntersectsByPlane(*p)
}
func (s BoundingSphere) IntersectsByRay(r Ray) (float32, bool) {
	return r.IntersectsByBoundingSphere(s)
}
func (s BoundingSphere) IntersectsByRefRayAndOutNullableOfSingle(r *Ray) (float32, bool) {
	return s.IntersectsByRay(*r)
}

func BoundingSphereCreateFromBoundingBoxByBoundingBox(b BoundingBox) BoundingSphere {
	center := Vector3LerpByVector3AndVector3AndSingle(b.Min, b.Max, 0.5)
	return NewBoundingSphere(center, Vector3DistanceByVector3AndVector3(b.Min, b.Max)*0.5)
}
func BoundingSphereCreateFromBoundingBoxByRefBoundingBoxAndOutBoundingSphere(b *BoundingBox) BoundingSphere {
	return BoundingSphereCreateFromBoundingBoxByBoundingBox(*b)
}
func BoundingSphereCreateMergedByBoundingSphereAndBoundingSphere(a, b BoundingSphere) BoundingSphere {
	difference := Vector3SubtractByVector3AndVector3(b.Center, a.Center)
	distance := difference.Length()
	if a.Radius+b.Radius >= distance {
		if a.Radius-b.Radius >= distance {
			return a
		}
		if b.Radius-a.Radius >= distance {
			return b
		}
	}
	direction := Vector3MultiplyByVector3AndSingle(difference, 1/distance)
	min := MathHelperMin(-a.Radius, distance-b.Radius)
	max := MathHelperMax(a.Radius, distance+b.Radius)
	radius := (max - min) * 0.5
	return NewBoundingSphere(Vector3AddByVector3AndVector3(a.Center, Vector3MultiplyByVector3AndSingle(direction, radius+min)), radius)
}
func BoundingSphereCreateMergedByRefBoundingSphereAndRefBoundingSphereAndOutBoundingSphere(a, b *BoundingSphere) BoundingSphere {
	return BoundingSphereCreateMergedByBoundingSphereAndBoundingSphere(*a, *b)
}
func BoundingSphereCreateFromPoints(points any) BoundingSphere {
	values := vector3Points(points)
	if len(values) == 0 {
		panic("points are empty")
	}
	minX, maxX, minY, maxY, minZ, maxZ := values[0], values[0], values[0], values[0], values[0], values[0]
	for _, p := range values {
		if p.X < minX.X {
			minX = p
		}
		if p.X > maxX.X {
			maxX = p
		}
		if p.Y < minY.Y {
			minY = p
		}
		if p.Y > maxY.Y {
			maxY = p
		}
		if p.Z < minZ.Z {
			minZ = p
		}
		if p.Z > maxZ.Z {
			maxZ = p
		}
	}
	dx, dy, dz := Vector3DistanceByVector3AndVector3(maxX, minX), Vector3DistanceByVector3AndVector3(maxY, minY), Vector3DistanceByVector3AndVector3(maxZ, minZ)
	var center Vector3
	var radius float32
	if dx > dy {
		if dx > dz {
			center = Vector3LerpByVector3AndVector3AndSingle(maxX, minX, 0.5)
			radius = dx * 0.5
		} else {
			center = Vector3LerpByVector3AndVector3AndSingle(maxZ, minZ, 0.5)
			radius = dz * 0.5
		}
	} else if dy > dz {
		center = Vector3LerpByVector3AndVector3AndSingle(maxY, minY, 0.5)
		radius = dy * 0.5
	} else {
		center = Vector3LerpByVector3AndVector3AndSingle(maxZ, minZ, 0.5)
		radius = dz * 0.5
	}
	for _, p := range values {
		offset := Vector3SubtractByVector3AndVector3(p, center)
		distance := offset.Length()
		if distance > radius {
			radius = (radius + distance) * 0.5
			center = Vector3AddByVector3AndVector3(center, Vector3MultiplyByVector3AndSingle(offset, 1-radius/distance))
		}
	}
	return NewBoundingSphere(center, radius)
}
func BoundingSphereCreateFromFrustum(f *BoundingFrustum) BoundingSphere {
	if f == nil {
		panic("frustum is nil")
	}
	return BoundingSphereCreateFromPoints(f.GetCornersByNone())
}
func (s BoundingSphere) TransformByMatrix(m Matrix) BoundingSphere {
	r1 := m.M11*m.M11 + m.M12*m.M12 + m.M13*m.M13
	r2 := m.M21*m.M21 + m.M22*m.M22 + m.M23*m.M23
	r3 := m.M31*m.M31 + m.M32*m.M32 + m.M33*m.M33
	maximum := r2
	if r3 > r2 {
		maximum = r3
	}
	if r1 > maximum {
		maximum = r1
	}
	return NewBoundingSphere(Vector3TransformByVector3AndMatrix(s.Center, m), s.Radius*sqrt32(maximum))
}
func (s BoundingSphere) TransformByRefMatrixAndOutBoundingSphere(m *Matrix) BoundingSphere {
	return s.TransformByMatrix(*m)
}
func (s BoundingSphere) support(direction Vector3) Vector3 {
	length := direction.Length()
	scale := s.Radius / length
	return Vector3{s.Center.X + direction.X*scale, s.Center.Y + direction.Y*scale, s.Center.Z + direction.Z*scale}
}
func (s BoundingSphere) EqualsByBoundingSphere(o BoundingSphere) bool {
	return s.Center.EqualsByVector3(o.Center) && s.Radius == o.Radius
}
func (s BoundingSphere) EqualsByObject(value any) bool {
	o, ok := value.(BoundingSphere)
	return ok && s.EqualsByBoundingSphere(o)
}
func (s BoundingSphere) GetHashCode() int32 { return s.Center.GetHashCode() + singleHashCode(s.Radius) }
func (s BoundingSphere) ToString() string {
	return fmt.Sprintf("{Center:%s Radius:%s}", s.Center.ToString(), formatSingle(s.Radius))
}
func BoundingSphereOperatorEqualityByBoundingSphereAndBoundingSphere(a, b BoundingSphere) bool {
	return a.EqualsByBoundingSphere(b)
}
func BoundingSphereOperatorInequalityByBoundingSphereAndBoundingSphere(a, b BoundingSphere) bool {
	return !a.EqualsByBoundingSphere(b)
}
