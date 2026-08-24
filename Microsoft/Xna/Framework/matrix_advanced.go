package framework

func (m Matrix) Decompose() (bool, Vector3, Quaternion, Vector3) {
	translation := Vector3{m.M41, m.M42, m.M43}
	basis := [3]Vector3{{m.M11, m.M12, m.M13}, {m.M21, m.M22, m.M23}, {m.M31, m.M32, m.M33}}
	scale := [3]float32{basis[0].Length(), basis[1].Length(), basis[2].Length()}
	var largest, middle, smallest int
	if scale[0] < scale[1] {
		if scale[1] < scale[2] {
			largest, middle, smallest = 2, 1, 0
		} else {
			largest = 1
			if scale[0] < scale[2] {
				middle, smallest = 2, 0
			} else {
				middle, smallest = 0, 2
			}
		}
	} else if scale[0] < scale[2] {
		largest, middle, smallest = 2, 0, 1
	} else {
		largest = 0
		if scale[1] < scale[2] {
			middle, smallest = 2, 1
		} else {
			middle, smallest = 1, 2
		}
	}
	canonical := [3]Vector3{Vector3UnitX(), Vector3UnitY(), Vector3UnitZ()}
	if scale[largest] < 0.0001 {
		basis[largest] = canonical[largest]
	}
	basis[largest].Normalize()
	if scale[middle] < 0.0001 {
		x, y, z := abs32(basis[largest].X), abs32(basis[largest].Y), abs32(basis[largest].Z)
		least := 0
		if x < y {
			if !(y < z || x < z) {
				least = 2
			}
		} else {
			if x < z || y < z {
				least = 1
			} else {
				least = 2
			}
		}
		canonical[least] = Vector3CrossByVector3AndVector3(basis[middle], basis[largest])
	}
	basis[middle].Normalize()
	if scale[smallest] < 0.0001 {
		basis[middle] = Vector3CrossByVector3AndVector3(basis[smallest], basis[largest])
	}
	basis[smallest].Normalize()
	rotationMatrix := NewMatrix(basis[0].X, basis[0].Y, basis[0].Z, 0, basis[1].X, basis[1].Y, basis[1].Z, 0, basis[2].X, basis[2].Y, basis[2].Z, 0, 0, 0, 0, 1)
	determinant := rotationMatrix.Determinant()
	if determinant < 0 {
		scale[largest] = -scale[largest]
		basis[largest] = Vector3NegateByVector3(basis[largest])
		determinant = -determinant
		rotationMatrix = NewMatrix(basis[0].X, basis[0].Y, basis[0].Z, 0, basis[1].X, basis[1].Y, basis[1].Z, 0, basis[2].X, basis[2].Y, basis[2].Z, 0, 0, 0, 0, 1)
	}
	determinant -= 1
	determinant *= determinant
	if 0.0001 < determinant {
		return false, Vector3{scale[0], scale[1], scale[2]}, QuaternionIdentity(), translation
	}
	return true, Vector3{scale[0], scale[1], scale[2]}, QuaternionCreateFromRotationMatrixByMatrix(rotationMatrix), translation
}

func MatrixInvertByMatrix(m Matrix) Matrix { return MatrixInvertByRefMatrixAndOutMatrix(&m) }
func MatrixInvertByRefMatrixAndOutMatrix(m *Matrix) Matrix {
	n1, n2, n3, n4 := m.M11, m.M12, m.M13, m.M14
	n5, n6, n7, n8 := m.M21, m.M22, m.M23, m.M24
	n9, n10, n11, n12 := m.M31, m.M32, m.M33, m.M34
	n13, n14, n15, n16 := m.M41, m.M42, m.M43, m.M44
	n17 := n11*n16 - n12*n15
	n18 := n10*n16 - n12*n14
	n19 := n10*n15 - n11*n14
	n20 := n9*n16 - n12*n13
	n21 := n9*n15 - n11*n13
	n22 := n9*n14 - n10*n13
	n23 := n6*n17 - n7*n18 + n8*n19
	n24 := -(n5*n17 - n7*n20 + n8*n21)
	n25 := n5*n18 - n6*n20 + n8*n22
	n26 := -(n5*n19 - n6*n21 + n7*n22)
	n27 := float32(1) / (n1*n23 + n2*n24 + n3*n25 + n4*n26)
	var r Matrix
	r.M11 = n23 * n27
	r.M21 = n24 * n27
	r.M31 = n25 * n27
	r.M41 = n26 * n27
	r.M12 = -(n2*n17 - n3*n18 + n4*n19) * n27
	r.M22 = (n1*n17 - n3*n20 + n4*n21) * n27
	r.M32 = -(n1*n18 - n2*n20 + n4*n22) * n27
	r.M42 = (n1*n19 - n2*n21 + n3*n22) * n27
	n28 := n7*n16 - n8*n15
	n29 := n6*n16 - n8*n14
	n30 := n6*n15 - n7*n14
	n31 := n5*n16 - n8*n13
	n32 := n5*n15 - n7*n13
	n33 := n5*n14 - n6*n13
	r.M13 = (n2*n28 - n3*n29 + n4*n30) * n27
	r.M23 = -(n1*n28 - n3*n31 + n4*n32) * n27
	r.M33 = (n1*n29 - n2*n31 + n4*n33) * n27
	r.M43 = -(n1*n30 - n2*n32 + n3*n33) * n27
	n34 := n7*n12 - n8*n11
	n35 := n6*n12 - n8*n10
	n36 := n6*n11 - n7*n10
	n37 := n5*n12 - n8*n9
	n38 := n5*n11 - n7*n9
	n39 := n5*n10 - n6*n9
	r.M14 = -(n2*n34 - n3*n35 + n4*n36) * n27
	r.M24 = (n1*n34 - n3*n37 + n4*n38) * n27
	r.M34 = -(n1*n35 - n2*n37 + n4*n39) * n27
	r.M44 = (n1*n36 - n2*n38 + n3*n39) * n27
	return r
}
