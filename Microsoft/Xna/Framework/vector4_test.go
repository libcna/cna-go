package framework

import "testing"

func TestVector4UnaryNegationPreservesNegativeZero(t *testing.T) {
	got := Vector4OperatorUnaryNegationByVector4(Vector4Zero())
	requireFloatBits(t, got.X, 0x80000000)
}

func TestVectorScalarDivisionUsesReciprocalFirst(t *testing.T) {
	requireFloatBits(t, Vector2DivideByVector2AndSingle(NewVector2BySingle(3), 7).X, 0x3EDB6DB8)
	requireFloatBits(t, Vector3DivideByVector3AndSingle(NewVector3BySingle(7), 3).X, 0x40155556)
	requireFloatBits(t, Vector4DivideByVector4AndSingle(NewVector4BySingle(12345.67), 3).X, 0x458099CA)
}
