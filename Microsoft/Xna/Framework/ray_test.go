package framework

import "testing"

func TestRayNullableIntersections(t *testing.T) {
	sphere := NewBoundingSphere(Vector3Zero(), 1)
	distance, ok := NewRay(Vector3{-5, 0.25, 0}, Vector3UnitX()).IntersectsByBoundingSphere(sphere)
	if !ok {
		t.Fatal("sphere hit reported null")
	}
	requireFloatBits(t, distance, 0x40810421)
	if _, ok := NewRay(Vector3{2, 0, 0}, Vector3{-5e-7, 0, 0}).IntersectsByBoundingBox(NewBoundingBox(NewVector3BySingle(-1), NewVector3BySingle(1))); ok {
		t.Fatal("near-parallel box ray reported hit")
	}
	if _, ok := NewRay(Vector3Zero(), Vector3{5e-6, 1, 0}).IntersectsByPlane(NewPlaneByVector3AndSingle(Vector3UnitX(), -1)); ok {
		t.Fatal("near-parallel plane ray reported hit")
	}
	if distance, ok := NewRay(Vector3{5e-6, 0, 0}, Vector3UnitX()).IntersectsByPlane(NewPlaneByVector3AndSingle(Vector3UnitX(), 0)); !ok || distance != 0 {
		t.Fatalf("just-behind plane = %v,%t", distance, ok)
	}
}

func TestRayInsideTangentBehindAndZeroDirection(t *testing.T) {
	sphere := NewBoundingSphere(Vector3Zero(), 1)
	if distance, ok := NewRay(Vector3Zero(), Vector3UnitX()).IntersectsByBoundingSphere(sphere); !ok || distance != 0 {
		t.Fatalf("inside sphere = %v,%t", distance, ok)
	}
	if distance, ok := NewRay(Vector3{X: -2, Y: 1}, Vector3UnitX()).IntersectsByBoundingSphere(sphere); !ok || distance != 2 {
		t.Fatalf("tangent sphere = %v,%t", distance, ok)
	}
	if _, ok := NewRay(Vector3{X: 2}, Vector3UnitX()).IntersectsByBoundingSphere(sphere); ok {
		t.Fatal("intersection behind ray reported a value")
	}
	if _, ok := NewRay(Vector3{X: 2}, Vector3Zero()).IntersectsByBoundingSphere(sphere); ok {
		t.Fatal("zero direction outside sphere reported a value")
	}
	box := NewBoundingBox(NewVector3BySingle(-1), NewVector3BySingle(1))
	if distance, ok := NewRay(Vector3Zero(), Vector3Zero()).IntersectsByBoundingBox(box); !ok || distance != 0 {
		t.Fatalf("zero direction inside box = %v,%t", distance, ok)
	}
	if _, ok := NewRay(Vector3{X: 2}, Vector3Zero()).IntersectsByBoundingBox(box); ok {
		t.Fatal("zero direction outside box reported a value")
	}
}
