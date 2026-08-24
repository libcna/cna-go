package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
)

type observation struct {
	ID       string `json:"id"`
	Group    string `json:"group"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
	Passed   bool   `json:"passed"`
}

type corpusReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Authority     string        `json:"authority"`
	Summary       corpusSummary `json:"summary"`
	Observations  []observation `json:"observations"`
}

type corpusSummary struct {
	Observations int `json:"OBSERVATIONS"`
	Assertions   int `json:"ASSERTIONS"`
	Failures     int `json:"FAILURES"`
}

func main() {
	output := flag.String("output", "docs/generated/behavior-corpus-report.json", "report path")
	flag.Parse()
	report := runCorpus()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("OBSERVATIONS=%d\nASSERTIONS=%d\nFAILURES=%d\n", report.Summary.Observations, report.Summary.Assertions, report.Summary.Failures)
	if report.Summary.Failures != 0 {
		os.Exit(1)
	}
}

func runCorpus() corpusReport {
	report := corpusReport{SchemaVersion: 1, Authority: "PURE_XNA_DERIVED retained XNA 4.0 reference observations"}
	check := func(id, group string, expected, actual any) {
		passed := fmt.Sprint(expected) == fmt.Sprint(actual)
		report.Observations = append(report.Observations, observation{ID: id, Group: group, Expected: expected, Actual: actual, Passed: passed})
		report.Summary.Observations++
		report.Summary.Assertions++
		if !passed {
			report.Summary.Failures++
		}
	}
	bits := func(value float32) string { return fmt.Sprintf("0x%08x", math.Float32bits(value)) }
	floatBits := func(values ...float32) string {
		result := make([]string, len(values))
		for i, value := range values {
			result[i] = bits(value)
		}
		return strings.Join(result, ",")
	}
	packed := func(value uint32) string { return fmt.Sprintf("0x%08x", value) }

	check("math.clamp.low", "MathHelper", bits(0), bits(framework.MathHelperClamp(-2, 0, 1)))
	check("math.clamp.inverted", "MathHelper", "0x40000000", bits(framework.MathHelperClamp(0, 2, 1)))
	check("math.lerp", "MathHelper", bits(4), bits(framework.MathHelperLerp(2, 10, 0.25)))
	check("math.barycentric", "MathHelper", bits(3.5), bits(framework.MathHelperBarycentric(1, 3, 5, 0.25, 0.5)))
	check("math.catmullrom", "MathHelper", "0xc1218313", bits(framework.MathHelperCatmullRom(-10, -10, -10, -7, 0.3)))
	check("math.hermite", "MathHelper", "0xc1351eba", bits(framework.MathHelperHermite(-10, -10, -10, -10, 1.1)))
	check("math.wrapangle.large", "MathHelper", "0xbfc2e06c", bits(framework.MathHelperWrapAngle(123456.789)))
	check("math.to_degrees", "MathHelper", bits(180), bits(framework.MathHelperToDegrees(framework.MathHelperPi)))
	check("math.negative_zero.distance", "MathHelper", "0x00000000", bits(framework.MathHelperDistance(float32(math.Copysign(0, -1)), 0)))
	check("math.nan.hermite", "MathHelper", true, math.IsNaN(float64(framework.MathHelperHermite(1, float32(math.Inf(1)), 2, 0, 0))))
	check("math.overflow.distance", "MathHelper", true, math.IsInf(float64(framework.MathHelperDistance(math.MaxFloat32, -math.MaxFloat32)), 1))
	check("math.subnormal.lerp", "MathHelper", bits(math.SmallestNonzeroFloat32), bits(framework.MathHelperLerp(0, math.SmallestNonzeroFloat32, 1)))
	check("math.underflow.lerp", "MathHelper", "0x00000000", bits(framework.MathHelperLerp(0, math.SmallestNonzeroFloat32, 0.5)))

	point := framework.NewPoint(1, 2)
	check("point.hash", "Point", int32(3), point.GetHashCode())
	check("point.string", "Point", "{X:1 Y:2}", point.ToString())
	check("point.value-equality", "Point", true, point.EqualsByPoint(framework.NewPoint(1, 2)))
	zero1, zero2 := framework.PointZero(), framework.PointZero()
	zero1.X = 9
	check("point.zero.fresh-value", "Point", int32(0), zero2.X)

	rectangle := framework.NewRectangle(0, 0, 10, 20)
	check("rectangle.contains.inclusive-min", "Rectangle", true, rectangle.ContainsByInt32AndInt32(0, 0))
	check("rectangle.contains.exclusive-max", "Rectangle", false, rectangle.ContainsByInt32AndInt32(10, 20))
	intersection := framework.RectangleIntersectByRectangleAndRectangle(framework.NewRectangle(0, 0, 10, 10), framework.NewRectangle(5, -5, 10, 10))
	check("rectangle.intersect", "Rectangle", "{X:5 Y:0 Width:5 Height:5}", intersection.ToString())
	union := framework.RectangleUnionByRectangleAndRectangle(framework.NewRectangle(0, 0, 10, 10), framework.NewRectangle(5, -5, 10, 10))
	check("rectangle.union", "Rectangle", "{X:0 Y:-5 Width:15 Height:15}", union.ToString())
	mutable := framework.NewRectangle(2, 3, 4, 5)
	mutable.Inflate(1, 2)
	check("rectangle.inflate", "Rectangle", "{X:1 Y:1 Width:6 Height:9}", mutable.ToString())
	mutable.OffsetByInt32AndInt32(-1, 2)
	check("rectangle.offset", "Rectangle", "{X:0 Y:3 Width:6 Height:9}", mutable.ToString())
	check("rectangle.hash", "Rectangle", int32(10), framework.NewRectangle(1, 2, 3, 4).GetHashCode())

	total := framework.TimeSpanFromTicks(math.MaxInt64)
	elapsed := framework.TimeSpanFromTicks(166667)
	gameTime := framework.NewGameTimeByTimeSpanAndTimeSpanAndBoolean(total, elapsed, true)
	check("gametime.total-ticks", "GameTime", int64(math.MaxInt64), gameTime.TotalGameTime().Ticks())
	check("gametime.elapsed-ticks", "GameTime", int64(166667), gameTime.ElapsedGameTime().Ticks())
	check("gametime.slow", "GameTime", true, gameTime.IsRunningSlowly())

	vector2Zero := framework.Vector2NormalizeByVector2(framework.Vector2Zero())
	check("vector2.normalize.zero", "VECTOR2", "0xffc00000,0xffc00000", floatBits(vector2Zero.X, vector2Zero.Y))
	check("vector2.divide.reciprocal-first", "VECTOR2", "0x3edb6db8", bits(framework.Vector2DivideByVector2AndSingle(framework.NewVector2BySingle(3), 7).X))
	vector2Overlap := []framework.Vector2{{X: 1}, {X: 2}, {X: 3}}
	translation := framework.MatrixCreateTranslationBySingleAndSingleAndSingle(10, 0, 0)
	framework.Vector2TransformBySliceOfVector2AndInt32AndRefMatrixAndSliceOfVector2AndInt32AndInt32(vector2Overlap, 0, &translation, vector2Overlap, 1, 2)
	check("vector2.transform.forward-overlap", "VECTOR2", "0x41300000,0x41a80000", floatBits(vector2Overlap[1].X, vector2Overlap[2].X))
	check("vector2.nan.equality", "VECTOR2", false, framework.Vector2{X: float32(math.NaN())}.EqualsByVector2(framework.Vector2{X: float32(math.NaN())}))

	vector3Zero := framework.Vector3NormalizeByVector3(framework.Vector3Zero())
	check("vector3.normalize.zero", "VECTOR3", "0xffc00000,0xffc00000,0xffc00000", floatBits(vector3Zero.X, vector3Zero.Y, vector3Zero.Z))
	check("vector3.forward", "VECTOR3", "0x00000000,0x00000000,0xbf800000", floatBits(framework.Vector3Forward().X, framework.Vector3Forward().Y, framework.Vector3Forward().Z))
	cross := framework.Vector3CrossByVector3AndVector3(framework.Vector3Right(), framework.Vector3Up())
	check("vector3.cross.handedness", "VECTOR3", "0x00000000,0x00000000,0x3f800000", floatBits(cross.X, cross.Y, cross.Z))
	check("vector3.divide.reciprocal-first", "VECTOR3", "0x40155556", bits(framework.Vector3DivideByVector3AndSingle(framework.NewVector3BySingle(7), 3).X))
	check("vector3.hash", "VECTOR3", int32(-1077936128), (framework.Vector3{X: 1, Y: 2, Z: 3}).GetHashCode())
	xnaNaN := math.Float32frombits(0xffc00000)
	vector3MinNaN := framework.Vector3MinByVector3AndVector3(framework.Vector3{X: xnaNaN, Y: 1, Z: xnaNaN}, framework.Vector3{X: 7, Y: xnaNaN, Z: xnaNaN})
	check("vector3.min.nan-order", "VECTOR3", "0x40e00000,0xffc00000,0xffc00000", floatBits(vector3MinNaN.X, vector3MinNaN.Y, vector3MinNaN.Z))
	vector3ClampReversed := framework.Vector3ClampByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.NewVector3BySingle(2), framework.NewVector3BySingle(1))
	check("vector3.clamp.reversed", "VECTOR3", "0x40000000,0x40000000,0x40000000", floatBits(vector3ClampReversed.X, vector3ClampReversed.Y, vector3ClampReversed.Z))

	vector4Zero := framework.Vector4NormalizeByVector4(framework.Vector4Zero())
	check("vector4.normalize.zero", "VECTOR4", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(vector4Zero.X, vector4Zero.Y, vector4Zero.Z, vector4Zero.W))
	vector4Negated := framework.Vector4OperatorUnaryNegationByVector4(framework.Vector4Zero())
	check("vector4.negate.signed-zero", "VECTOR4", "0x80000000,0x80000000,0x80000000,0x80000000", floatBits(vector4Negated.X, vector4Negated.Y, vector4Negated.Z, vector4Negated.W))
	check("vector4.divide.reciprocal-first", "VECTOR4", "0x458099ca", bits(framework.Vector4DivideByVector4AndSingle(framework.NewVector4BySingle(12345.67), 3).X))

	quaternionZero := framework.QuaternionInverseByQuaternion(framework.Quaternion{})
	check("quaternion.inverse.zero", "QUATERNION", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(quaternionZero.X, quaternionZero.Y, quaternionZero.Z, quaternionZero.W))
	qa := framework.Quaternion{X: 45889.05859375, Y: -42412.4453125, Z: 96034.96875, W: -76386.84375}
	qb := framework.Quaternion{X: -16375.435546875, Y: 51428.1875, Z: -69603.09375, W: -2207.3798828125}
	quaternionProduct := framework.QuaternionMultiplyByQuaternionAndQuaternion(qa, qb)
	check("quaternion.multiply.order", "QUATERNION", "0xce47a05e,0xcf03edf7,0x4fc9c4dd,0x5011d115", floatBits(quaternionProduct.X, quaternionProduct.Y, quaternionProduct.Z, quaternionProduct.W))
	yaw := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Up(), 0.7)
	pitch := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Right(), -0.4)
	slerp := framework.QuaternionSlerpByQuaternionAndQuaternionAndSingle(yaw, pitch, 0.37)
	check("quaternion.slerp.branch", "QUATERNION", "0xbd9a16ec,0x3e60d7e7,0x00000000,0x3f79023d", floatBits(slerp.X, slerp.Y, slerp.Z, slerp.W))
	check("quaternion.concatenate.order", "QUATERNION", true, framework.QuaternionConcatenateByQuaternionAndQuaternion(yaw, pitch) == framework.QuaternionMultiplyByQuaternionAndQuaternion(pitch, yaw))
	largeAxis := framework.QuaternionCreateFromAxisAngleByVector3AndSingle(framework.Vector3Up(), 123456.789)
	check("quaternion.axis-angle.large", "QUATERNION", "0x00000000,0x3f30464f,0x00000000,0xbf39a48f", floatBits(largeAxis.X, largeAxis.Y, largeAxis.Z, largeAxis.W))
	fromMatrix := framework.QuaternionCreateFromRotationMatrixByMatrix(framework.MatrixCreateRotationYBySingle(0.7))
	check("quaternion.from-matrix", "QUATERNION", "0x00000000,0x3eaf904c,0x00000000,0x3f707abb", floatBits(fromMatrix.X, fromMatrix.Y, fromMatrix.Z, fromMatrix.W))

	matrix := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(2, 3, 4), framework.MatrixCreateRotationYBySingle(0.25))
	matrix = framework.MatrixMultiplyByMatrixAndMatrix(matrix, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	check("matrix.translation.row-vector", "MATRIX", "0x40a00000,0x40c00000,0x40e00000", floatBits(matrix.Translation().X, matrix.Translation().Y, matrix.Translation().Z))
	matrixProduct := framework.MatrixMultiplyByMatrixAndMatrix(matrix, framework.MatrixInvertByMatrix(matrix))
	check("matrix.inverse.product", "MATRIX", "0x3f800000,0x00000000,0xb2000000,0x00000000,0x00000000,0x3f800000,0x00000000,0x00000000,0x33000000,0x00000000,0x3f800000,0x00000000,0x34000000,0x00000000,0x00000000,0x3f800000", floatBits(matrixProduct.M11, matrixProduct.M12, matrixProduct.M13, matrixProduct.M14, matrixProduct.M21, matrixProduct.M22, matrixProduct.M23, matrixProduct.M24, matrixProduct.M31, matrixProduct.M32, matrixProduct.M33, matrixProduct.M34, matrixProduct.M41, matrixProduct.M42, matrixProduct.M43, matrixProduct.M44))
	singular := framework.MatrixInvertByMatrix(framework.Matrix{})
	check("matrix.invert.singular", "MATRIX", "0xffc00000,0xffc00000,0xffc00000,0xffc00000", floatBits(singular.M11, singular.M22, singular.M33, singular.M44))
	check("matrix.identity.hash", "MATRIX", int32(-33554432), framework.MatrixIdentity().GetHashCode())
	rotation := framework.MatrixCreateRotationYBySingle(123456.789)
	check("matrix.rotation.large", "MATRIX", "0x3d53e807,0xbf7fa83d", floatBits(rotation.M11, rotation.M31))
	perspectiveInfinity := framework.MatrixCreatePerspectiveBySingleAndSingleAndSingleAndSingle(4, 3, 0.1, float32(math.Inf(1)))
	check("matrix.perspective.infinity", "MATRIX", "0xffc00000,0xffc00000", floatBits(perspectiveInfinity.M33, perspectiveInfinity.M43))
	constrainedBillboard := framework.MatrixCreateConstrainedBillboardByVector3AndVector3AndVector3AndNullableOfVector3AndNullableOfVector3(framework.Vector3{Y: 10}, framework.Vector3Zero(), framework.Vector3{Y: 2}, nil, nil)
	check("matrix.billboard.axis", "MATRIX", "0xbf800000,0x40000000,0xbf800000", floatBits(constrainedBillboard.M11, constrainedBillboard.M22, constrainedBillboard.M33))
	zeroPlaneShadow := framework.MatrixCreateShadowByVector3AndPlane(framework.Vector3Forward(), framework.NewPlaneByVector3AndSingle(framework.Vector3Zero(), 0))
	check("matrix.shadow.zero-plane", "MATRIX", "true,true", fmt.Sprintf("%t,%t", math.IsNaN(float64(zeroPlaneShadow.M11)), math.IsNaN(float64(zeroPlaneShadow.M44))))
	degenerateLookAt := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.Vector3Zero(), framework.Vector3Up())
	check("matrix.lookat.degenerate", "MATRIX", "0xffc00000,0xffc00000,0xffc00000,0x00000000,0xffc00000,0xffc00000,0xffc00000,0x00000000,0xffc00000,0xffc00000,0xffc00000,0x00000000,0x7fc00000,0x7fc00000,0x7fc00000,0x3f800000", floatBits(degenerateLookAt.M11, degenerateLookAt.M12, degenerateLookAt.M13, degenerateLookAt.M14, degenerateLookAt.M21, degenerateLookAt.M22, degenerateLookAt.M23, degenerateLookAt.M24, degenerateLookAt.M31, degenerateLookAt.M32, degenerateLookAt.M33, degenerateLookAt.M34, degenerateLookAt.M41, degenerateLookAt.M42, degenerateLookAt.M43, degenerateLookAt.M44))
	infiniteTransformInput := framework.MatrixIdentity()
	infiniteTransformInput.M14 = float32(math.Inf(1))
	infiniteTransform := framework.MatrixTransformByMatrixAndQuaternion(infiniteTransformInput, framework.QuaternionIdentity())
	check("matrix.transform.infinity", "MATRIX", "0x3f800000,0x7f800000,false", fmt.Sprintf("%s,%s,%t", bits(infiniteTransform.M11), bits(infiniteTransform.M14), math.IsNaN(float64(infiniteTransform.M11))))
	mirrored := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(-2, 3, 4), framework.MatrixCreateRotationYBySingle(0.25))
	mirrored = framework.MatrixMultiplyByMatrixAndMatrix(mirrored, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(5, 6, 7))
	decomposed, scale, orientation, matrixTranslation := mirrored.Decompose()
	check("matrix.decompose.return", "MATRIX", true, decomposed)
	check("matrix.decompose.outputs", "MATRIX", "0x40000000,0x40400000,0xc0800000,0x00000000,0x3f7e00aa,0x00000000,0xbdff5579,0x40a00000,0x40c00000,0x40e00000", floatBits(scale.X, scale.Y, scale.Z, orientation.X, orientation.Y, orientation.Z, orientation.W, matrixTranslation.X, matrixTranslation.Y, matrixTranslation.Z))
	check("matrix.string", "MATRIX", "{ {M11:1 M12:0 M13:0 M14:0} {M21:0 M22:1 M23:0 M24:0} {M31:0 M32:0 M33:1 M34:0} {M41:0 M42:0 M43:0 M44:1} }", framework.MatrixIdentity().ToString())

	degeneratePlane := framework.NewPlaneByVector3AndVector3AndVector3(framework.Vector3Zero(), framework.Vector3Zero(), framework.Vector3Zero())
	check("plane.points.degenerate", "PLANE", "0xffc00000,0xffc00000,0xffc00000,0x7fc00000", floatBits(degeneratePlane.Normal.X, degeneratePlane.Normal.Y, degeneratePlane.Normal.Z, degeneratePlane.D))
	nearUnitPlane := framework.PlaneNormalizeByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3{X: 0.6, Y: 0.79999995}, 2))
	check("plane.normalize.near-unit", "PLANE", "0x3f19999a,0x3f4ccccc,0x00000000,0x40000000", floatBits(nearUnitPlane.Normal.X, nearUnitPlane.Normal.Y, nearUnitPlane.Normal.Z, nearUnitPlane.D))
	unitBox := framework.NewBoundingBox(framework.NewVector3BySingle(-1), framework.NewVector3BySingle(1))
	check("plane.box.coplanar", "PLANE", framework.PlaneIntersectionTypeIntersecting, (framework.Plane{}).IntersectsByBoundingBox(unitBox))
	reflectionPlane := framework.NewPlaneByVector3AndSingle(framework.Vector3{X: 2}, 4)
	reflection := framework.MatrixCreateReflectionByRefPlaneAndOutMatrix(&reflectionPlane)
	check("plane.reflection.ref-normalization", "PLANE", "0x3f800000,0x40000000,0xbf800000,0xc0800000", floatBits(reflectionPlane.Normal.X, reflectionPlane.D, reflection.M11, reflection.M41))

	unitSphere := framework.NewBoundingSphere(framework.Vector3Zero(), 1)
	rayDistance, rayHit := framework.NewRay(framework.Vector3{X: -5, Y: 0.25}, framework.Vector3UnitX()).IntersectsByBoundingSphere(unitSphere)
	check("ray.sphere.nullable.has-value", "RAY", true, rayHit)
	check("ray.sphere.distance", "RAY", "0x40810421", bits(rayDistance))
	_, rayBoxHit := framework.NewRay(framework.Vector3{X: 2}, framework.Vector3{X: -5e-7}).IntersectsByBoundingBox(unitBox)
	check("ray.box.near-parallel-null", "RAY", false, rayBoxHit)
	_, rayPlaneHit := framework.NewRay(framework.Vector3Zero(), framework.Vector3{X: 5e-6, Y: 1}).IntersectsByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3UnitX(), -1))
	check("ray.plane.near-parallel-null", "RAY", false, rayPlaneHit)
	behindDistance, behindHit := framework.NewRay(framework.Vector3{X: 5e-6}, framework.Vector3UnitX()).IntersectsByPlane(framework.NewPlaneByVector3AndSingle(framework.Vector3UnitX(), 0))
	check("ray.plane.just-behind-clamped", "RAY", "true,0x00000000", fmt.Sprintf("%t,%s", behindHit, bits(behindDistance)))

	check("box.point.edge", "BOUNDING_BOX", framework.ContainmentTypeContains, unitBox.ContainsByVector3(framework.Vector3UnitX()))
	check("box.point.nan", "BOUNDING_BOX", framework.ContainmentTypeDisjoint, unitBox.ContainsByVector3(framework.Vector3{X: float32(math.NaN())}))
	nan := float32(math.NaN())
	nanBox := framework.NewBoundingBox(framework.Vector3{X: nan, Y: -1, Z: -1}, framework.Vector3{X: nan, Y: 1, Z: 1})
	check("box.box.nan-intersects", "BOUNDING_BOX", true, unitBox.IntersectsByBoundingBox(nanBox))
	boxCorners := framework.NewBoundingBox(framework.Vector3{X: 1, Y: 2, Z: 3}, framework.Vector3{X: 4, Y: 5, Z: 6}).GetCornersByNone()
	check("box.corner.order", "BOUNDING_BOX", "{X:1 Y:5 Z:6}|{X:4 Y:2 Z:3}|{X:1 Y:2 Z:3}", boxCorners[0].ToString()+"|"+boxCorners[6].ToString()+"|"+boxCorners[7].ToString())

	check("sphere.point.edge", "BOUNDING_SPHERE", framework.ContainmentTypeDisjoint, unitSphere.ContainsByVector3(framework.Vector3UnitX()))
	check("sphere.external-tangent", "BOUNDING_SPHERE", false, unitSphere.IntersectsByBoundingSphere(framework.NewBoundingSphere(framework.Vector3{X: 2}, 1)))
	pointsSphere := framework.BoundingSphereCreateFromPoints([]framework.Vector3{{X: -4, Y: 1}, {X: 6, Y: -2, Z: 3}, {Y: 8, Z: -5}, {X: 2, Z: 9}})
	check("sphere.create-points", "BOUNDING_SPHERE", "0x3f800000,0x40800000,0x40000000,0x4101fc10", floatBits(pointsSphere.Center.X, pointsSphere.Center.Y, pointsSphere.Center.Z, pointsSphere.Radius))

	projection := framework.MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(framework.MathHelperPiOver4, 4.0/3.0, 1, 10)
	view := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3{Z: 5}, framework.Vector3Zero(), framework.Vector3Up())
	frustum := framework.NewBoundingFrustum(framework.MatrixMultiplyByMatrixAndMatrix(view, projection))
	frustumNear := frustum.Near()
	check("frustum.near-plane", "BOUNDING_FRUSTUM", "0x80000000,0x80000000,0x3f800000,0xc0800000", floatBits(frustumNear.Normal.X, frustumNear.Normal.Y, frustumNear.Normal.Z, frustumNear.D))
	frustumTop := frustum.Top()
	check("frustum.top-plane", "BOUNDING_FRUSTUM", "0x00000000,0x3f6c835f,0x3ec3ef16,0xbff4eadb", floatBits(frustumTop.Normal.X, frustumTop.Normal.Y, frustumTop.Normal.Z, frustumTop.D))
	frustumCorners := frustum.GetCornersByNone()
	check("frustum.corner-order.first", "BOUNDING_FRUSTUM", "0xbf0d6289,0x3ed413cb,0x40800000", floatBits(frustumCorners[0].X, frustumCorners[0].Y, frustumCorners[0].Z))
	check("frustum.corner-order.seventh", "BOUNDING_FRUSTUM", "0x40b0bb28,0xc0848c5d,0xc09ffff8", floatBits(frustumCorners[6].X, frustumCorners[6].Y, frustumCorners[6].Z))
	check("frustum.box.gjk.inside", "BOUNDING_FRUSTUM", true, frustum.IntersectsByBoundingBox(framework.NewBoundingBox(framework.NewVector3BySingle(-0.5), framework.NewVector3BySingle(0.5))))
	check("frustum.box.gjk.outside", "BOUNDING_FRUSTUM", false, frustum.IntersectsByBoundingBox(framework.NewBoundingBox(framework.NewVector3BySingle(100), framework.NewVector3BySingle(101))))
	frustumDistance, frustumRayHit := frustum.IntersectsByRay(framework.NewRay(framework.Vector3{Z: 20}, framework.Vector3Forward()))
	check("frustum.ray.nullable", "BOUNDING_FRUSTUM", "true,0x41800000", fmt.Sprintf("%t,%s", frustumRayHit, bits(frustumDistance)))
	check("frustum.class.nil-equality", "BOUNDING_FRUSTUM", false, framework.BoundingFrustumOperatorEqualityByBoundingFrustumAndBoundingFrustum(frustum, nil))

	floatColor := framework.NewColorBySingleAndSingleAndSingleAndSingle(0.5, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)))
	check("color.float-packing", "COLOR", "0x00ff0080", packed(floatColor.PackedValue()))
	check("color.lerp.midpoint", "COLOR", "0x7f7f7f7f", packed(framework.ColorLerp(framework.NewColorByInt32AndInt32AndInt32AndInt32(0, 0, 0, 0), framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255), 0.5).PackedValue()))
	check("color.multiply.midpoint", "COLOR", "0x7f7f7f7f", packed(framework.ColorMultiply(framework.ColorWhite(), 0.5).PackedValue()))
	check("color.transparent", "COLOR", "0x00ffffff", packed(framework.ColorTransparent().PackedValue()))
	check("color.cornflower-blue", "COLOR", "0xffed9564", packed(framework.ColorCornflowerBlue().PackedValue()))
	colorVector := framework.NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4).ToVector4()
	check("color.vector-roundtrip", "COLOR", true, framework.NewColorByVector4(colorVector) == framework.NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4))

	viewport := graphics.NewViewportByInt32AndInt32AndInt32AndInt32(11, 13, 640, 360)
	viewport.SetMinDepth(0.2)
	viewport.SetMaxDepth(0.9)
	viewportWorld := framework.MatrixMultiplyByMatrixAndMatrix(framework.MatrixCreateScaleBySingleAndSingleAndSingle(1.5, 0.75, 2), framework.MatrixCreateRotationYBySingle(0.31))
	viewportWorld = framework.MatrixMultiplyByMatrixAndMatrix(viewportWorld, framework.MatrixCreateTranslationBySingleAndSingleAndSingle(2, -1, 0.5))
	viewportView := framework.MatrixCreateLookAtByVector3AndVector3AndVector3(framework.Vector3{X: 4, Y: 3, Z: 8}, framework.Vector3Zero(), framework.Vector3Up())
	viewportProjection := framework.MatrixCreatePerspectiveFieldOfViewBySingleAndSingleAndSingleAndSingle(0.9, 16.0/9.0, 0.1, 100)
	projected := viewport.Project(framework.Vector3{X: 0.25, Y: -0.5, Z: 1.25}, viewportProjection, viewportView, viewportWorld)
	check("viewport.project", "VIEWPORT", "0x43d42808,0x43ac9f3c,0x3f63aff4", floatBits(projected.X, projected.Y, projected.Z))
	unprojected := viewport.Unproject(projected, viewportProjection, viewportView, viewportWorld)
	check("viewport.unproject", "VIEWPORT", "0x3e7ffe10,0xbefff906,0x3fa00111", floatBits(unprojected.X, unprojected.Y, unprojected.Z))
	singularUnproject := viewport.Unproject(framework.Vector3{X: 100, Y: 50, Z: 0.5}, framework.MatrixIdentity(), framework.MatrixIdentity(), framework.Matrix{})
	check("viewport.unproject.singular", "VIEWPORT", "0xffc00000,0xffc00000,0xffc00000", floatBits(singularUnproject.X, singularUnproject.Y, singularUnproject.Z))

	return report
}
