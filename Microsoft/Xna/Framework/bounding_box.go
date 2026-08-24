package framework

import "fmt"

const BoundingBoxCornerCount int32 = 8

// BoundingBox stores XNA's unmodified minimum and maximum corners.
type BoundingBox struct{ Min, Max Vector3 }

func NewBoundingBox(min, max Vector3) BoundingBox { return BoundingBox{min, max} }
func (b BoundingBox) GetCornersByNone() []Vector3 {
	return []Vector3{{b.Min.X, b.Max.Y, b.Max.Z}, {b.Max.X, b.Max.Y, b.Max.Z}, {b.Max.X, b.Min.Y, b.Max.Z}, {b.Min.X, b.Min.Y, b.Max.Z}, {b.Min.X, b.Max.Y, b.Min.Z}, {b.Max.X, b.Max.Y, b.Min.Z}, {b.Max.X, b.Min.Y, b.Min.Z}, {b.Min.X, b.Min.Y, b.Min.Z}}
}
func (b BoundingBox) GetCornersBySliceOfVector3(corners []Vector3) {
	if corners == nil || len(corners) < 8 {
		panic("corners array too small")
	}
	copy(corners, b.GetCornersByNone())
}
func (b BoundingBox) ContainsByVector3(p Vector3) ContainmentType {
	if b.Min.X <= p.X && p.X <= b.Max.X && b.Min.Y <= p.Y && p.Y <= b.Max.Y && b.Min.Z <= p.Z && p.Z <= b.Max.Z {
		return ContainmentTypeContains
	}
	return ContainmentTypeDisjoint
}
func (b BoundingBox) ContainsByRefVector3AndOutContainmentType(p *Vector3) ContainmentType {
	return b.ContainsByVector3(*p)
}
func (b BoundingBox) ContainsByBoundingBox(o BoundingBox) ContainmentType {
	if b.Max.X < o.Min.X || b.Min.X > o.Max.X || b.Max.Y < o.Min.Y || b.Min.Y > o.Max.Y || b.Max.Z < o.Min.Z || b.Min.Z > o.Max.Z {
		return ContainmentTypeDisjoint
	}
	if b.Min.X <= o.Min.X && o.Max.X <= b.Max.X && b.Min.Y <= o.Min.Y && o.Max.Y <= b.Max.Y && b.Min.Z <= o.Min.Z && o.Max.Z <= b.Max.Z {
		return ContainmentTypeContains
	}
	return ContainmentTypeIntersects
}
func (b BoundingBox) ContainsByRefBoundingBoxAndOutContainmentType(o *BoundingBox) ContainmentType {
	return b.ContainsByBoundingBox(*o)
}
func (b BoundingBox) ContainsByBoundingSphere(s BoundingSphere) ContainmentType {
	closest := Vector3ClampByVector3AndVector3AndVector3(s.Center, b.Min, b.Max)
	distance := Vector3DistanceSquaredByVector3AndVector3(s.Center, closest)
	r := s.Radius
	if distance > r*r {
		return ContainmentTypeDisjoint
	}
	containsX := b.Min.X+r <= s.Center.X && s.Center.X <= b.Max.X-r && b.Max.X-b.Min.X > r
	containsY := b.Min.Y+r <= s.Center.Y && s.Center.Y <= b.Max.Y-r && b.Max.Y-b.Min.Y > r
	// XNA 4.0 observably repeats the X-width test in the Z clause.
	containsZ := b.Min.Z+r <= s.Center.Z && s.Center.Z <= b.Max.Z-r && b.Max.X-b.Min.X > r
	if containsX && containsY && containsZ {
		return ContainmentTypeContains
	}
	return ContainmentTypeIntersects
}
func (b BoundingBox) ContainsByRefBoundingSphereAndOutContainmentType(s *BoundingSphere) ContainmentType {
	return b.ContainsByBoundingSphere(*s)
}
func (b BoundingBox) ContainsByBoundingFrustum(f *BoundingFrustum) ContainmentType {
	if f == nil {
		panic("frustum is nil")
	}
	all := true
	for _, c := range f.GetCornersByNone() {
		if b.ContainsByVector3(c) == ContainmentTypeDisjoint {
			all = false
			break
		}
	}
	if all {
		return ContainmentTypeContains
	}
	if b.IntersectsByBoundingFrustum(f) {
		return ContainmentTypeIntersects
	}
	return ContainmentTypeDisjoint
}
func (b BoundingBox) IntersectsByBoundingBox(o BoundingBox) bool {
	return !(b.Max.X < o.Min.X || b.Min.X > o.Max.X || b.Max.Y < o.Min.Y || b.Min.Y > o.Max.Y || b.Max.Z < o.Min.Z || b.Min.Z > o.Max.Z)
}
func (b BoundingBox) IntersectsByRefBoundingBoxAndOutBoolean(o *BoundingBox) bool {
	return b.IntersectsByBoundingBox(*o)
}
func (b BoundingBox) IntersectsByBoundingSphere(s BoundingSphere) bool {
	closest := Vector3ClampByVector3AndVector3AndVector3(s.Center, b.Min, b.Max)
	return !(Vector3DistanceSquaredByVector3AndVector3(s.Center, closest) > s.Radius*s.Radius)
}
func (b BoundingBox) IntersectsByRefBoundingSphereAndOutBoolean(s *BoundingSphere) bool {
	return b.IntersectsByBoundingSphere(*s)
}
func (b BoundingBox) IntersectsByBoundingFrustum(f *BoundingFrustum) bool {
	if f == nil {
		panic("frustum is nil")
	}
	return f.IntersectsByBoundingBox(b)
}
func (b BoundingBox) IntersectsByPlane(p Plane) PlaneIntersectionType {
	return p.IntersectsByBoundingBox(b)
}
func (b BoundingBox) IntersectsByRefPlaneAndOutPlaneIntersectionType(p *Plane) PlaneIntersectionType {
	return b.IntersectsByPlane(*p)
}
func intersectSlab(position, direction, min, max float32, distance, maxDistance *float32) bool {
	if abs32(direction) < 1e-6 {
		return position >= min && position <= max
	}
	inv := float32(1) / direction
	near, far := (min-position)*inv, (max-position)*inv
	if near > far {
		near, far = far, near
	}
	*distance = MathHelperMax(near, *distance)
	*maxDistance = MathHelperMin(far, *maxDistance)
	return *distance <= *maxDistance
}
func (b BoundingBox) IntersectsByRay(r Ray) (float32, bool) {
	distance, maxDistance := float32(0), float32(3.4028234663852886e+38)
	if !intersectSlab(r.Position.X, r.Direction.X, b.Min.X, b.Max.X, &distance, &maxDistance) || !intersectSlab(r.Position.Y, r.Direction.Y, b.Min.Y, b.Max.Y, &distance, &maxDistance) || !intersectSlab(r.Position.Z, r.Direction.Z, b.Min.Z, b.Max.Z, &distance, &maxDistance) {
		return 0, false
	}
	return distance, true
}
func (b BoundingBox) IntersectsByRefRayAndOutNullableOfSingle(r *Ray) (float32, bool) {
	return b.IntersectsByRay(*r)
}

func vector3Points(value any) []Vector3 {
	points, ok := value.([]Vector3)
	if !ok || points == nil {
		panic("points must be []Vector3")
	}
	return points
}
func BoundingBoxCreateFromPoints(points any) BoundingBox {
	values := vector3Points(points)
	if len(values) == 0 {
		panic("points are empty")
	}
	min, max := NewVector3BySingle(3.4028234663852886e+38), NewVector3BySingle(-3.4028234663852886e+38)
	for _, p := range values {
		min = Vector3MinByVector3AndVector3(min, p)
		max = Vector3MaxByVector3AndVector3(max, p)
	}
	return BoundingBox{min, max}
}
func BoundingBoxCreateMergedByBoundingBoxAndBoundingBox(a, b BoundingBox) BoundingBox {
	return BoundingBox{Vector3MinByVector3AndVector3(a.Min, b.Min), Vector3MaxByVector3AndVector3(a.Max, b.Max)}
}
func BoundingBoxCreateMergedByRefBoundingBoxAndRefBoundingBoxAndOutBoundingBox(a, b *BoundingBox) BoundingBox {
	return BoundingBoxCreateMergedByBoundingBoxAndBoundingBox(*a, *b)
}
func BoundingBoxCreateFromSphereByBoundingSphere(s BoundingSphere) BoundingBox {
	r := NewVector3BySingle(s.Radius)
	return BoundingBox{Vector3SubtractByVector3AndVector3(s.Center, r), Vector3AddByVector3AndVector3(s.Center, r)}
}
func BoundingBoxCreateFromSphereByRefBoundingSphereAndOutBoundingBox(s *BoundingSphere) BoundingBox {
	return BoundingBoxCreateFromSphereByBoundingSphere(*s)
}
func (b BoundingBox) support(direction Vector3) Vector3 {
	r := Vector3{}
	if direction.X >= 0 {
		r.X = b.Max.X
	} else {
		r.X = b.Min.X
	}
	if direction.Y >= 0 {
		r.Y = b.Max.Y
	} else {
		r.Y = b.Min.Y
	}
	if direction.Z >= 0 {
		r.Z = b.Max.Z
	} else {
		r.Z = b.Min.Z
	}
	return r
}
func (b BoundingBox) EqualsByBoundingBox(o BoundingBox) bool {
	return b.Min.EqualsByVector3(o.Min) && b.Max.EqualsByVector3(o.Max)
}
func (b BoundingBox) EqualsByObject(value any) bool {
	o, ok := value.(BoundingBox)
	return ok && b.EqualsByBoundingBox(o)
}
func (b BoundingBox) GetHashCode() int32 { return b.Min.GetHashCode() + b.Max.GetHashCode() }
func (b BoundingBox) ToString() string {
	return fmt.Sprintf("{Min:%s Max:%s}", b.Min.ToString(), b.Max.ToString())
}
func BoundingBoxOperatorEqualityByBoundingBoxAndBoundingBox(a, b BoundingBox) bool {
	return a.EqualsByBoundingBox(b)
}
func BoundingBoxOperatorInequalityByBoundingBoxAndBoundingBox(a, b BoundingBox) bool {
	return !a.EqualsByBoundingBox(b)
}
