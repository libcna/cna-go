package framework

import "fmt"

const BoundingFrustumCornerCount int32 = 8

// BoundingFrustum is a managed reference facade, matching XNA class identity semantics.
type BoundingFrustum struct {
	matrix  Matrix
	planes  [6]Plane
	corners [8]Vector3
	gjk     *gjk
}

func NewBoundingFrustum(value Matrix) *BoundingFrustum {
	f := &BoundingFrustum{}
	f.setMatrix(value)
	return f
}
func (f *BoundingFrustum) Near() Plane            { return f.planes[0] }
func (f *BoundingFrustum) Far() Plane             { return f.planes[1] }
func (f *BoundingFrustum) Left() Plane            { return f.planes[2] }
func (f *BoundingFrustum) Right() Plane           { return f.planes[3] }
func (f *BoundingFrustum) Top() Plane             { return f.planes[4] }
func (f *BoundingFrustum) Bottom() Plane          { return f.planes[5] }
func (f *BoundingFrustum) Matrix() Matrix         { return f.matrix }
func (f *BoundingFrustum) SetMatrix(value Matrix) { f.setMatrix(value) }
func (f *BoundingFrustum) GetCornersByNone() []Vector3 {
	r := make([]Vector3, 8)
	copy(r, f.corners[:])
	return r
}
func (f *BoundingFrustum) GetCornersBySliceOfVector3(corners []Vector3) {
	if corners == nil || len(corners) < 8 {
		panic("corners array too small")
	}
	copy(corners, f.corners[:])
}
func (f *BoundingFrustum) ContainsByVector3(point Vector3) ContainmentType {
	for _, p := range f.planes {
		d := p.Normal.X*point.X + p.Normal.Y*point.Y + p.Normal.Z*point.Z + p.D
		if d > 1e-5 {
			return ContainmentTypeDisjoint
		}
	}
	return ContainmentTypeContains
}
func (f *BoundingFrustum) ContainsByRefVector3AndOutContainmentType(point *Vector3) ContainmentType {
	return f.ContainsByVector3(*point)
}
func (f *BoundingFrustum) ContainsByBoundingBox(box BoundingBox) ContainmentType {
	intersects := false
	for _, p := range f.planes {
		switch box.IntersectsByPlane(p) {
		case PlaneIntersectionTypeFront:
			return ContainmentTypeDisjoint
		case PlaneIntersectionTypeIntersecting:
			intersects = true
		}
	}
	if intersects {
		return ContainmentTypeIntersects
	}
	return ContainmentTypeContains
}
func (f *BoundingFrustum) ContainsByRefBoundingBoxAndOutContainmentType(box *BoundingBox) ContainmentType {
	return f.ContainsByBoundingBox(*box)
}
func (f *BoundingFrustum) ContainsByBoundingSphere(s BoundingSphere) ContainmentType {
	inside := 0
	for _, p := range f.planes {
		d := p.Normal.X*s.Center.X + p.Normal.Y*s.Center.Y + p.Normal.Z*s.Center.Z + p.D
		if d > s.Radius {
			return ContainmentTypeDisjoint
		}
		if d < -s.Radius {
			inside++
		}
	}
	if inside == 6 {
		return ContainmentTypeContains
	}
	return ContainmentTypeIntersects
}
func (f *BoundingFrustum) ContainsByRefBoundingSphereAndOutContainmentType(s *BoundingSphere) ContainmentType {
	return f.ContainsByBoundingSphere(*s)
}
func (f *BoundingFrustum) ContainsByBoundingFrustum(o *BoundingFrustum) ContainmentType {
	if o == nil {
		panic("frustum is nil")
	}
	if !f.IntersectsByBoundingFrustum(o) {
		return ContainmentTypeDisjoint
	}
	for _, c := range o.corners {
		if f.ContainsByVector3(c) == ContainmentTypeDisjoint {
			return ContainmentTypeIntersects
		}
	}
	return ContainmentTypeContains
}

func (f *BoundingFrustum) ensureGJK() *gjk {
	if f.gjk == nil {
		f.gjk = &gjk{}
	}
	f.gjk.reset()
	return f.gjk
}
func (f *BoundingFrustum) support(direction Vector3) Vector3 {
	index := 0
	selected := Vector3DotByVector3AndVector3(f.corners[0], direction)
	for i := 1; i < 8; i++ {
		dot := Vector3DotByVector3AndVector3(f.corners[i], direction)
		if dot > selected {
			index = i
			selected = dot
		}
	}
	return f.corners[index]
}
func (f *BoundingFrustum) intersectsSupport(otherPoint Vector3, otherSupport func(Vector3) Vector3) bool {
	g := f.ensureGJK()
	closest := Vector3SubtractByVector3AndVector3(f.corners[0], otherPoint)
	previous := float32(3.4028234663852886e+38)
	threshold := float32(0)
	for {
		direction := Vector3NegateByVector3(closest)
		p1 := f.support(direction)
		p2 := otherSupport(closest)
		support := Vector3SubtractByVector3AndVector3(p1, p2)
		if Vector3DotByVector3AndVector3(closest, support) > 0 {
			return false
		}
		g.addSupportPoint(support)
		closest = g.closest
		old := previous
		previous = closest.LengthSquared()
		if old-previous <= 1e-5*old {
			return false
		}
		threshold = 4e-5 * g.maxLengthSquared
		if g.fullSimplex() || previous < threshold {
			return true
		}
	}
}
func (f *BoundingFrustum) IntersectsByBoundingBox(box BoundingBox) bool {
	closest := Vector3SubtractByVector3AndVector3(f.corners[0], box.Min)
	if closest.LengthSquared() < 1e-5 {
		closest = Vector3SubtractByVector3AndVector3(f.corners[0], box.Max)
	}
	g := f.ensureGJK()
	previous := float32(3.4028234663852886e+38)
	threshold := float32(0)
	for {
		direction := Vector3NegateByVector3(closest)
		support := Vector3SubtractByVector3AndVector3(f.support(direction), box.support(closest))
		if Vector3DotByVector3AndVector3(closest, support) > 0 {
			return false
		}
		g.addSupportPoint(support)
		closest = g.closest
		old := previous
		previous = closest.LengthSquared()
		if old-previous <= 1e-5*old {
			return false
		}
		threshold = 4e-5 * g.maxLengthSquared
		if g.fullSimplex() || previous < threshold {
			return true
		}
	}
}
func (f *BoundingFrustum) IntersectsByRefBoundingBoxAndOutBoolean(box *BoundingBox) bool {
	return f.IntersectsByBoundingBox(*box)
}
func (f *BoundingFrustum) IntersectsByBoundingSphere(s BoundingSphere) bool {
	closest := Vector3SubtractByVector3AndVector3(f.corners[0], s.Center)
	if closest.LengthSquared() < 1e-5 {
		closest = Vector3UnitX()
	}
	g := f.ensureGJK()
	previous := float32(3.4028234663852886e+38)
	threshold := float32(0)
	for {
		direction := Vector3NegateByVector3(closest)
		support := Vector3SubtractByVector3AndVector3(f.support(direction), s.support(closest))
		if Vector3DotByVector3AndVector3(closest, support) > 0 {
			return false
		}
		g.addSupportPoint(support)
		closest = g.closest
		old := previous
		previous = closest.LengthSquared()
		if old-previous <= 1e-5*old {
			return false
		}
		threshold = 4e-5 * g.maxLengthSquared
		if g.fullSimplex() || previous < threshold {
			return true
		}
	}
}
func (f *BoundingFrustum) IntersectsByRefBoundingSphereAndOutBoolean(s *BoundingSphere) bool {
	return f.IntersectsByBoundingSphere(*s)
}
func (f *BoundingFrustum) IntersectsByBoundingFrustum(o *BoundingFrustum) bool {
	if o == nil {
		panic("frustum is nil")
	}
	closest := Vector3SubtractByVector3AndVector3(f.corners[0], o.corners[0])
	if closest.LengthSquared() < 1e-5 {
		closest = Vector3SubtractByVector3AndVector3(f.corners[0], o.corners[1])
	}
	g := f.ensureGJK()
	previous := float32(3.4028234663852886e+38)
	threshold := float32(0)
	for {
		direction := Vector3NegateByVector3(closest)
		support := Vector3SubtractByVector3AndVector3(f.support(direction), o.support(closest))
		if Vector3DotByVector3AndVector3(closest, support) > 0 {
			return false
		}
		g.addSupportPoint(support)
		closest = g.closest
		old := previous
		previous = closest.LengthSquared()
		threshold = 4e-5 * g.maxLengthSquared
		if old-previous <= 1e-5*old {
			return false
		}
		if g.fullSimplex() || previous < threshold {
			return true
		}
	}
}
func (f *BoundingFrustum) IntersectsByPlane(p Plane) PlaneIntersectionType {
	mask := 0
	for _, c := range f.corners {
		dot := Vector3DotByVector3AndVector3(c, p.Normal)
		if dot+p.D > 0 {
			mask |= 1
		} else {
			mask |= 2
		}
		if mask == 3 {
			return PlaneIntersectionTypeIntersecting
		}
	}
	if mask == 1 {
		return PlaneIntersectionTypeFront
	}
	return PlaneIntersectionTypeBack
}
func (f *BoundingFrustum) IntersectsByRefPlaneAndOutPlaneIntersectionType(p *Plane) PlaneIntersectionType {
	return f.IntersectsByPlane(*p)
}
func (f *BoundingFrustum) IntersectsByRay(r Ray) (float32, bool) {
	if f.ContainsByVector3(r.Position) == ContainmentTypeContains {
		return 0, true
	}
	entry, exit := float32(-3.4028234663852886e+38), float32(3.4028234663852886e+38)
	for _, p := range f.planes {
		directionDot := Vector3DotByVector3AndVector3(r.Direction, p.Normal)
		positionDot := Vector3DotByVector3AndVector3(r.Position, p.Normal) + p.D
		if abs32(directionDot) < 1e-5 {
			if positionDot > 0 {
				return 0, false
			}
			continue
		}
		distance := -positionDot / directionDot
		if directionDot < 0 {
			if distance > exit {
				return 0, false
			}
			if distance > entry {
				entry = distance
			}
		} else {
			if distance < entry {
				return 0, false
			}
			if distance < exit {
				exit = distance
			}
		}
	}
	result := exit
	if entry >= 0 {
		result = entry
	}
	if result >= 0 {
		return result, true
	}
	return 0, false
}
func (f *BoundingFrustum) IntersectsByRefRayAndOutNullableOfSingle(r *Ray) (float32, bool) {
	return f.IntersectsByRay(*r)
}

func (f *BoundingFrustum) EqualsByBoundingFrustum(o *BoundingFrustum) bool {
	return f != nil && o != nil && f.matrix.EqualsByMatrix(o.matrix)
}
func (f *BoundingFrustum) EqualsByObject(value any) bool {
	o, ok := value.(*BoundingFrustum)
	return ok && f.EqualsByBoundingFrustum(o)
}
func (f *BoundingFrustum) GetHashCode() int32 { return f.matrix.GetHashCode() }
func (f *BoundingFrustum) ToString() string {
	return fmt.Sprintf("{Near:%s Far:%s Left:%s Right:%s Top:%s Bottom:%s}", f.Near().ToString(), f.Far().ToString(), f.Left().ToString(), f.Right().ToString(), f.Top().ToString(), f.Bottom().ToString())
}
func BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(a, b *BoundingFrustum) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.matrix.EqualsByMatrix(b.matrix)
}
func BoundingFrustumOperatorInequalityByBoundingFrustumAndBoundingFrustum(a, b *BoundingFrustum) bool {
	return !BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(a, b)
}

func (f *BoundingFrustum) setMatrix(v Matrix) {
	f.matrix = v
	f.planes[2] = Plane{Vector3{-v.M14 - v.M11, -v.M24 - v.M21, -v.M34 - v.M31}, -v.M44 - v.M41}
	f.planes[3] = Plane{Vector3{-v.M14 + v.M11, -v.M24 + v.M21, -v.M34 + v.M31}, -v.M44 + v.M41}
	f.planes[4] = Plane{Vector3{-v.M14 + v.M12, -v.M24 + v.M22, -v.M34 + v.M32}, -v.M44 + v.M42}
	f.planes[5] = Plane{Vector3{-v.M14 - v.M12, -v.M24 - v.M22, -v.M34 - v.M32}, -v.M44 - v.M42}
	f.planes[0] = Plane{Vector3{-v.M13, -v.M23, -v.M33}, -v.M43}
	f.planes[1] = Plane{Vector3{-v.M14 + v.M13, -v.M24 + v.M23, -v.M34 + v.M33}, -v.M44 + v.M43}
	for i := range f.planes {
		length := f.planes[i].Normal.Length()
		f.planes[i].Normal = Vector3DivideByVector3AndSingle(f.planes[i].Normal, length)
		f.planes[i].D /= length
	}
	ray := frustumIntersectionLine(f.planes[0], f.planes[2])
	f.corners[0] = frustumIntersection(f.planes[4], ray)
	f.corners[3] = frustumIntersection(f.planes[5], ray)
	ray = frustumIntersectionLine(f.planes[3], f.planes[0])
	f.corners[1] = frustumIntersection(f.planes[4], ray)
	f.corners[2] = frustumIntersection(f.planes[5], ray)
	ray = frustumIntersectionLine(f.planes[2], f.planes[1])
	f.corners[4] = frustumIntersection(f.planes[4], ray)
	f.corners[7] = frustumIntersection(f.planes[5], ray)
	ray = frustumIntersectionLine(f.planes[1], f.planes[3])
	f.corners[5] = frustumIntersection(f.planes[4], ray)
	f.corners[6] = frustumIntersection(f.planes[5], ray)
}
func frustumIntersectionLine(a, b Plane) Ray {
	direction := Vector3CrossByVector3AndVector3(a.Normal, b.Normal)
	position := Vector3DivideByVector3AndSingle(Vector3CrossByVector3AndVector3(Vector3AddByVector3AndVector3(Vector3MultiplyByVector3AndSingle(b.Normal, -a.D), Vector3MultiplyByVector3AndSingle(a.Normal, b.D)), direction), direction.LengthSquared())
	return Ray{position, direction}
}
func frustumIntersection(p Plane, r Ray) Vector3 {
	distance := (-p.D - Vector3DotByVector3AndVector3(p.Normal, r.Position)) / Vector3DotByVector3AndVector3(p.Normal, r.Direction)
	return Vector3AddByVector3AndVector3(r.Position, Vector3MultiplyByVector3AndSingle(r.Direction, distance))
}
