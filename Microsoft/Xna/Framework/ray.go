package framework

import "fmt"

// Ray has an origin and direction; intersection absence is returned as hasValue=false.
type Ray struct{ Position, Direction Vector3 }

func NewRay(position, direction Vector3) Ray                          { return Ray{position, direction} }
func (r Ray) IntersectsByBoundingBox(box BoundingBox) (float32, bool) { return box.IntersectsByRay(r) }
func (r Ray) IntersectsByRefBoundingBoxAndOutNullableOfSingle(box *BoundingBox) (float32, bool) {
	return r.IntersectsByBoundingBox(*box)
}
func (r Ray) IntersectsByBoundingSphere(s BoundingSphere) (float32, bool) {
	x, y, z := s.Center.X-r.Position.X, s.Center.Y-r.Position.Y, s.Center.Z-r.Position.Z
	distance := x*x + y*y + z*z
	radius := s.Radius * s.Radius
	if distance <= radius {
		return 0, true
	}
	projection := x*r.Direction.X + y*r.Direction.Y + z*r.Direction.Z
	if projection < 0 {
		return 0, false
	}
	closest := distance - projection*projection
	if closest > radius {
		return 0, false
	}
	return projection - sqrt32(radius-closest), true
}
func (r Ray) IntersectsByRefBoundingSphereAndOutNullableOfSingle(s *BoundingSphere) (float32, bool) {
	return r.IntersectsByBoundingSphere(*s)
}
func (r Ray) IntersectsByPlane(p Plane) (float32, bool) {
	denominator := p.Normal.X*r.Direction.X + p.Normal.Y*r.Direction.Y + p.Normal.Z*r.Direction.Z
	if abs32(denominator) < 1e-5 {
		return 0, false
	}
	positionDot := p.Normal.X*r.Position.X + p.Normal.Y*r.Position.Y + p.Normal.Z*r.Position.Z
	d := (-p.D - positionDot) / denominator
	if d < 0 {
		if d < -1e-5 {
			return 0, false
		}
		return 0, true
	}
	return d, true
}
func (r Ray) IntersectsByRefPlaneAndOutNullableOfSingle(p *Plane) (float32, bool) {
	return r.IntersectsByPlane(*p)
}
func (r Ray) IntersectsByBoundingFrustum(f *BoundingFrustum) (float32, bool) {
	if f == nil {
		panic("frustum is nil")
	}
	return f.IntersectsByRay(r)
}
func (r Ray) EqualsByRay(o Ray) bool {
	return r.Position.EqualsByVector3(o.Position) && r.Direction.EqualsByVector3(o.Direction)
}
func (r Ray) EqualsByObject(value any) bool { o, ok := value.(Ray); return ok && r.EqualsByRay(o) }
func (r Ray) GetHashCode() int32            { return r.Position.GetHashCode() + r.Direction.GetHashCode() }
func (r Ray) ToString() string {
	return fmt.Sprintf("{Position:%s Direction:%s}", r.Position.ToString(), r.Direction.ToString())
}
func RayOperatorEqualityByRayAndRay(a, b Ray) bool   { return a.EqualsByRay(b) }
func RayOperatorInequalityByRayAndRay(a, b Ray) bool { return !a.EqualsByRay(b) }
