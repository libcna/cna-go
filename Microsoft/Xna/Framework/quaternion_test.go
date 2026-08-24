package framework

import "testing"

func TestQuaternionXnaGoldenBranchesAndOrdering(t *testing.T) {
	zero := QuaternionInverseByQuaternion(Quaternion{})
	for _, value := range []float32{zero.X, zero.Y, zero.Z, zero.W} {
		requireFloatBits(t, value, 0xFFC00000)
	}

	a := Quaternion{45889.05859375, -42412.4453125, 96034.96875, -76386.84375}
	b := Quaternion{-16375.435546875, 51428.1875, -69603.09375, -2207.3798828125}
	product := QuaternionMultiplyByQuaternionAndQuaternion(a, b)
	want := [4]uint32{0xCE47A05E, 0xCF03EDF7, 0x4FC9C4DD, 0x5011D115}
	for i, value := range []float32{product.X, product.Y, product.Z, product.W} {
		requireFloatBits(t, value, want[i])
	}

	yaw := QuaternionCreateFromAxisAngleByVector3AndSingle(Vector3Up(), 0.7)
	pitch := QuaternionCreateFromAxisAngleByVector3AndSingle(Vector3Right(), -0.4)
	slerp := QuaternionSlerpByQuaternionAndQuaternionAndSingle(yaw, pitch, 0.37)
	for i, value := range []float32{slerp.X, slerp.Y, slerp.Z, slerp.W} {
		requireFloatBits(t, value, [4]uint32{0xBD9A16EC, 0x3E60D7E7, 0, 0x3F79023D}[i])
	}
	if QuaternionConcatenateByQuaternionAndQuaternion(yaw, pitch) != QuaternionMultiplyByQuaternionAndQuaternion(pitch, yaw) {
		t.Fatal("Concatenate did not apply value1 then value2")
	}
}

func TestQuaternionCreationGoldens(t *testing.T) {
	large := QuaternionCreateFromAxisAngleByVector3AndSingle(Vector3Up(), 123456.789)
	for i, value := range []float32{large.X, large.Y, large.Z, large.W} {
		requireFloatBits(t, value, [4]uint32{0, 0x3F30464F, 0, 0xBF39A48F}[i])
	}
	fromMatrix := QuaternionCreateFromRotationMatrixByMatrix(MatrixCreateRotationYBySingle(0.7))
	for i, value := range []float32{fromMatrix.X, fromMatrix.Y, fromMatrix.Z, fromMatrix.W} {
		requireFloatBits(t, value, [4]uint32{0, 0x3EAF904C, 0, 0x3F707ABB}[i])
	}
}
