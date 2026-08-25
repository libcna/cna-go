package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
	input "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input"
)

type observation struct {
	ID         string `json:"id"`
	Group      string `json:"group"`
	Provenance string `json:"provenance"`
	Expected   any    `json:"expected"`
	Actual     any    `json:"actual"`
	Passed     bool   `json:"passed"`
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
	report := corpusReport{SchemaVersion: 1, Authority: "PURE_XNA_DERIVED retained XNA 4.0 reference observations; GO_LANGUAGE_PROJECTION observations are labeled individually"}
	checkWithProvenance := func(id, group, provenance string, expected, actual any) {
		passed := fmt.Sprint(expected) == fmt.Sprint(actual)
		report.Observations = append(report.Observations, observation{ID: id, Group: group, Provenance: provenance, Expected: expected, Actual: actual, Passed: passed})
		report.Summary.Observations++
		report.Summary.Assertions++
		if !passed {
			report.Summary.Failures++
		}
	}
	check := func(id, group string, expected, actual any) {
		checkWithProvenance(id, group, "PURE_XNA_DERIVED", expected, actual)
	}
	checkGoProjection := func(id, group string, expected, actual any) {
		checkWithProvenance(id, group, "GO_LANGUAGE_PROJECTION", expected, actual)
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

	check("player-index.defined-raw-values", "PLAYER_INDEX", "0,1,2,3", fmt.Sprintf("%d,%d,%d,%d", framework.PlayerIndexOne, framework.PlayerIndexTwo, framework.PlayerIndexThree, framework.PlayerIndexFour))
	check("player-index.undefined-raw-value", "PLAYER_INDEX", int32(12345), int32(framework.PlayerIndex(12345)))

	noPlayerState, noPlayerError := input.KeyboardGetStateByNone()
	for _, fixture := range []struct {
		name  string
		value framework.PlayerIndex
	}{
		{"one", framework.PlayerIndexOne},
		{"two", framework.PlayerIndexTwo},
		{"three", framework.PlayerIndexThree},
		{"four", framework.PlayerIndexFour},
		{"undefined", framework.PlayerIndex(12345)},
	} {
		state, err := input.KeyboardGetStateByPlayerIndex(fixture.value)
		sameError := noPlayerError != nil && err != nil && noPlayerError.Error() == err.Error()
		check("keyboard.player-index."+fixture.name, "KEYBOARD_PLAYER_INDEX", "true,true", fmt.Sprintf("%t,%t", state == noPlayerState, sameError))
	}

	check("display-orientation.raw-values", "DISPLAY_ORIENTATION", "0,1,2,4", fmt.Sprintf("%d,%d,%d,%d", framework.DisplayOrientationDefault, framework.DisplayOrientationLandscapeLeft, framework.DisplayOrientationLandscapeRight, framework.DisplayOrientationPortrait))
	landscape := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationLandscapeRight
	leftPortrait := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	allOrientations := landscape | framework.DisplayOrientationPortrait
	check("display-orientation.flags-combinations", "DISPLAY_ORIENTATION", "3,5,7", fmt.Sprintf("%d,%d,%d", landscape, leftPortrait, allOrientations))
	check("display-orientation.unknown-raw-bit", "DISPLAY_ORIENTATION", int32(1<<20), int32(framework.DisplayOrientation(1<<20)))

	managerDirty := func(manager *framework.GraphicsDeviceManager) bool {
		return reflect.ValueOf(manager).Elem().FieldByName("isDeviceDirty").Bool()
	}
	manager := &framework.GraphicsDeviceManager{}
	check("graphics-manager-orientation.initial-state", "GRAPHICS_MANAGER_ORIENTATION", "0,false", fmt.Sprintf("%d,%t", manager.SupportedOrientations(), managerDirty(manager)))
	manager.SetSupportedOrientations(framework.DisplayOrientationDefault)
	check("graphics-manager-orientation.same-value-dirty", "GRAPHICS_MANAGER_ORIENTATION", "0,true", fmt.Sprintf("%d,%t", manager.SupportedOrientations(), managerDirty(manager)))
	changedManager := &framework.GraphicsDeviceManager{}
	changedManager.SetSupportedOrientations(leftPortrait)
	check("graphics-manager-orientation.changed-value-dirty", "GRAPHICS_MANAGER_ORIENTATION", "5,true", fmt.Sprintf("%d,%t", changedManager.SupportedOrientations(), managerDirty(changedManager)))
	changedManager.SetSupportedOrientations(framework.DisplayOrientationLandscapeRight)
	changedManager.SetSupportedOrientations(framework.DisplayOrientation(1 << 20))
	check("graphics-manager-orientation.multiple-assignment", "GRAPHICS_MANAGER_ORIENTATION", "1048576,true", fmt.Sprintf("%d,%t", changedManager.SupportedOrientations(), managerDirty(changedManager)))
	postDisposeManager := &framework.GraphicsDeviceManager{}
	_ = postDisposeManager.Dispose(true)
	postDisposeManager.SetSupportedOrientations(landscape)
	check("graphics-manager-orientation.post-disposal-managed-state", "GRAPHICS_MANAGER_ORIENTATION", "3,true", fmt.Sprintf("%d,%t", postDisposeManager.SupportedOrientations(), managerDirty(postDisposeManager)))

	check("buffer-usage.raw-values", "BUFFER_USAGE", "0,1", fmt.Sprintf("%d,%d", graphics.BufferUsageNone, graphics.BufferUsageWriteOnly))
	check("buffer-usage.underlying-int32", "BUFFER_USAGE", "int32", reflect.TypeOf(graphics.BufferUsageNone).Kind().String())
	var zeroBufferUsage graphics.BufferUsage
	checkGoProjection("buffer-usage.zero-value", "BUFFER_USAGE", graphics.BufferUsageNone, zeroBufferUsage)
	checkGoProjection("buffer-usage.arbitrary-raw-values", "BUFFER_USAGE", "2,3,1048576,-1", fmt.Sprintf("%d,%d,%d,%d", graphics.BufferUsage(2), graphics.BufferUsage(3), graphics.BufferUsage(1<<20), graphics.BufferUsage(-1)))
	checkGoProjection("buffer-usage.bitwise-composition", "BUFFER_USAGE", "1,1,3", fmt.Sprintf("%d,%d,%d", graphics.BufferUsageNone|graphics.BufferUsageWriteOnly, graphics.BufferUsageWriteOnly|graphics.BufferUsageWriteOnly, graphics.BufferUsage(2)|graphics.BufferUsageWriteOnly))

	check("clear-options.raw-values", "CLEAR_OPTIONS", "1,2,4", fmt.Sprintf("%d,%d,%d", graphics.ClearOptionsTarget, graphics.ClearOptionsDepthBuffer, graphics.ClearOptionsStencil))
	clearOptionsKind := reflect.TypeOf(graphics.ClearOptionsTarget).Kind()
	check("clear-options.underlying-system-int32", "CLEAR_OPTIONS", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[clearOptionsKind])
	clearOptionsPowersOfTwo := graphics.ClearOptionsTarget != 0 && graphics.ClearOptionsTarget&(graphics.ClearOptionsTarget-1) == 0 &&
		graphics.ClearOptionsDepthBuffer != 0 && graphics.ClearOptionsDepthBuffer&(graphics.ClearOptionsDepthBuffer-1) == 0 &&
		graphics.ClearOptionsStencil != 0 && graphics.ClearOptionsStencil&(graphics.ClearOptionsStencil-1) == 0
	check("clear-options.flags-metadata-shape", "CLEAR_OPTIONS", true, clearOptionsPowersOfTwo)
	check("clear-options.no-declared-zero-literal", "CLEAR_OPTIONS", true, graphics.ClearOptionsTarget != 0 && graphics.ClearOptionsDepthBuffer != 0 && graphics.ClearOptionsStencil != 0)
	var zeroClearOptions graphics.ClearOptions
	checkGoProjection("clear-options.unnamed-zero-value", "CLEAR_OPTIONS", int32(0), int32(zeroClearOptions))
	checkGoProjection("clear-options.declared-combinations", "CLEAR_OPTIONS", "3,5,6,7", fmt.Sprintf("%d,%d,%d,%d", graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer, graphics.ClearOptionsTarget|graphics.ClearOptionsStencil, graphics.ClearOptionsDepthBuffer|graphics.ClearOptionsStencil, graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer|graphics.ClearOptionsStencil))
	checkGoProjection("clear-options.arbitrary-raw-values", "CLEAR_OPTIONS", "0,8,1048576,-1", fmt.Sprintf("%d,%d,%d,%d", graphics.ClearOptions(0), graphics.ClearOptions(8), graphics.ClearOptions(1<<20), graphics.ClearOptions(-1)))
	checkGoProjection("clear-options.bitwise-or", "CLEAR_OPTIONS", "3,9", fmt.Sprintf("%d,%d", graphics.ClearOptionsTarget|graphics.ClearOptionsDepthBuffer, graphics.ClearOptions(8)|graphics.ClearOptionsTarget))
	checkGoProjection("clear-options.bitwise-and", "CLEAR_OPTIONS", "4,0", fmt.Sprintf("%d,%d", graphics.ClearOptions(7)&graphics.ClearOptionsStencil, graphics.ClearOptions(2)&graphics.ClearOptionsTarget))

	check("surface-format.complete-raw-table", "SURFACE_FORMAT", "Color=0,Bgr565=1,Bgra5551=2,Bgra4444=3,Dxt1=4,Dxt3=5,Dxt5=6,NormalizedByte2=7,NormalizedByte4=8,Rgba1010102=9,Rg32=10,Rgba64=11,Alpha8=12,Single=13,Vector2=14,Vector4=15,HalfSingle=16,HalfVector2=17,HalfVector4=18,HdrBlendable=19", fmt.Sprintf(
		"Color=%d,Bgr565=%d,Bgra5551=%d,Bgra4444=%d,Dxt1=%d,Dxt3=%d,Dxt5=%d,NormalizedByte2=%d,NormalizedByte4=%d,Rgba1010102=%d,Rg32=%d,Rgba64=%d,Alpha8=%d,Single=%d,Vector2=%d,Vector4=%d,HalfSingle=%d,HalfVector2=%d,HalfVector4=%d,HdrBlendable=%d",
		graphics.SurfaceFormatColor,
		graphics.SurfaceFormatBgr565,
		graphics.SurfaceFormatBgra5551,
		graphics.SurfaceFormatBgra4444,
		graphics.SurfaceFormatDxt1,
		graphics.SurfaceFormatDxt3,
		graphics.SurfaceFormatDxt5,
		graphics.SurfaceFormatNormalizedByte2,
		graphics.SurfaceFormatNormalizedByte4,
		graphics.SurfaceFormatRgba1010102,
		graphics.SurfaceFormatRg32,
		graphics.SurfaceFormatRgba64,
		graphics.SurfaceFormatAlpha8,
		graphics.SurfaceFormatSingle,
		graphics.SurfaceFormatVector2,
		graphics.SurfaceFormatVector4,
		graphics.SurfaceFormatHalfSingle,
		graphics.SurfaceFormatHalfVector2,
		graphics.SurfaceFormatHalfVector4,
		graphics.SurfaceFormatHdrBlendable,
	))
	surfaceFormatKind := reflect.TypeOf(graphics.SurfaceFormatColor).Kind()
	check("surface-format.underlying-system-int32", "SURFACE_FORMAT", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[surfaceFormatKind])
	check("surface-format.flags", "SURFACE_FORMAT", false, false)
	var zeroSurfaceFormat graphics.SurfaceFormat
	checkGoProjection("surface-format.zero-value-color", "SURFACE_FORMAT", graphics.SurfaceFormatColor, zeroSurfaceFormat)
	checkGoProjection("surface-format.arbitrary-positive-raw", "SURFACE_FORMAT", "20,12345", fmt.Sprintf("%d,%d", graphics.SurfaceFormat(20), graphics.SurfaceFormat(12345)))
	checkGoProjection("surface-format.negative-raw", "SURFACE_FORMAT", int32(-1), int32(graphics.SurfaceFormat(-1)))

	check("depth-format.complete-raw-table", "DEPTH_FORMAT", "None=0,Depth16=1,Depth24=2,Depth24Stencil8=3", fmt.Sprintf(
		"None=%d,Depth16=%d,Depth24=%d,Depth24Stencil8=%d",
		graphics.DepthFormatNone,
		graphics.DepthFormatDepth16,
		graphics.DepthFormatDepth24,
		graphics.DepthFormatDepth24Stencil8,
	))
	depthFormatKind := reflect.TypeOf(graphics.DepthFormatNone).Kind()
	check("depth-format.underlying-system-int32", "DEPTH_FORMAT", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[depthFormatKind])
	check("depth-format.flags", "DEPTH_FORMAT", false, false)
	var zeroDepthFormat graphics.DepthFormat
	checkGoProjection("depth-format.zero-value-none", "DEPTH_FORMAT", graphics.DepthFormatNone, zeroDepthFormat)
	checkGoProjection("depth-format.arbitrary-positive-raw", "DEPTH_FORMAT", "4,12345", fmt.Sprintf("%d,%d", graphics.DepthFormat(4), graphics.DepthFormat(12345)))
	checkGoProjection("depth-format.negative-raw", "DEPTH_FORMAT", int32(-1), int32(graphics.DepthFormat(-1)))

	check("graphics-profile.complete-raw-table", "GRAPHICS_PROFILE", "Reach=0,HiDef=1", fmt.Sprintf(
		"Reach=%d,HiDef=%d",
		graphics.GraphicsProfileReach,
		graphics.GraphicsProfileHiDef,
	))
	graphicsProfileKind := reflect.TypeOf(graphics.GraphicsProfileReach).Kind()
	check("graphics-profile.underlying-system-int32", "GRAPHICS_PROFILE", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[graphicsProfileKind])
	check("graphics-profile.flags", "GRAPHICS_PROFILE", false, false)
	var zeroGraphicsProfile graphics.GraphicsProfile
	checkGoProjection("graphics-profile.zero-value-reach", "GRAPHICS_PROFILE", graphics.GraphicsProfileReach, zeroGraphicsProfile)
	checkGoProjection("graphics-profile.arbitrary-positive-raw", "GRAPHICS_PROFILE", "2,12345", fmt.Sprintf("%d,%d", graphics.GraphicsProfile(2), graphics.GraphicsProfile(12345)))
	checkGoProjection("graphics-profile.negative-raw", "GRAPHICS_PROFILE", int32(-1), int32(graphics.GraphicsProfile(-1)))

	check("button-state.complete-raw-table", "BUTTON_STATE", "Released=0,Pressed=1", fmt.Sprintf(
		"Released=%d,Pressed=%d",
		input.ButtonStateReleased,
		input.ButtonStatePressed,
	))
	buttonStateKind := reflect.TypeOf(input.ButtonStateReleased).Kind()
	check("button-state.underlying-system-int32", "BUTTON_STATE", "System.Int32", map[reflect.Kind]string{reflect.Int32: "System.Int32"}[buttonStateKind])
	check("button-state.flags", "BUTTON_STATE", false, false)
	var zeroButtonState input.ButtonState
	checkGoProjection("button-state.zero-value-released", "BUTTON_STATE", input.ButtonStateReleased, zeroButtonState)
	checkGoProjection("button-state.arbitrary-positive-raw", "BUTTON_STATE", "2,12345", fmt.Sprintf("%d,%d", input.ButtonState(2), input.ButtonState(12345)))
	checkGoProjection("button-state.negative-raw", "BUTTON_STATE", int32(-1), int32(input.ButtonState(-1)))

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

	check("vertex-element.enums.format", "VERTEX_ELEMENT_ENUMS", "0,1,2,3,4,5,6,7,8,9,10,11", fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d",
		graphics.VertexElementFormatSingle,
		graphics.VertexElementFormatVector2,
		graphics.VertexElementFormatVector3,
		graphics.VertexElementFormatVector4,
		graphics.VertexElementFormatColor,
		graphics.VertexElementFormatByte4,
		graphics.VertexElementFormatShort2,
		graphics.VertexElementFormatShort4,
		graphics.VertexElementFormatNormalizedShort2,
		graphics.VertexElementFormatNormalizedShort4,
		graphics.VertexElementFormatHalfVector2,
		graphics.VertexElementFormatHalfVector4,
	))
	check("vertex-element.enums.usage", "VERTEX_ELEMENT_ENUMS", "0,1,2,3,4,5,6,7,8,9,10,11,12", fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d",
		graphics.VertexElementUsagePosition,
		graphics.VertexElementUsageColor,
		graphics.VertexElementUsageTextureCoordinate,
		graphics.VertexElementUsageNormal,
		graphics.VertexElementUsageBinormal,
		graphics.VertexElementUsageTangent,
		graphics.VertexElementUsageBlendIndices,
		graphics.VertexElementUsageBlendWeight,
		graphics.VertexElementUsageDepth,
		graphics.VertexElementUsageFog,
		graphics.VertexElementUsagePointSize,
		graphics.VertexElementUsageSample,
		graphics.VertexElementUsageTessellateFactor,
	))

	var zeroVertexElement graphics.VertexElement
	constructedZeroVertexElement := graphics.NewVertexElement(0, graphics.VertexElementFormatSingle, graphics.VertexElementUsagePosition, 0)
	check("vertex-element.zero.getters", "VERTEX_ELEMENT", "0,0,0,0", fmt.Sprintf("%d,%d,%d,%d", zeroVertexElement.Offset(), zeroVertexElement.VertexElementFormat(), zeroVertexElement.VertexElementUsage(), zeroVertexElement.UsageIndex()))
	check("vertex-element.zero.constructor-equivalence", "VERTEX_ELEMENT", "true,2147483647,{Offset:0 Format:Single Usage:Position UsageIndex:0}", fmt.Sprintf("%t,%d,%s", zeroVertexElement.Equals(constructedZeroVertexElement), zeroVertexElement.GetHashCode(), zeroVertexElement.ToString()))

	ordinaryVertexElement := graphics.NewVertexElement(12, graphics.VertexElementFormatVector3, graphics.VertexElementUsageTextureCoordinate, 7)
	check("vertex-element.constructor", "VERTEX_ELEMENT", "12,2,2,7", fmt.Sprintf("%d,%d,%d,%d", ordinaryVertexElement.Offset(), ordinaryVertexElement.VertexElementFormat(), ordinaryVertexElement.VertexElementUsage(), ordinaryVertexElement.UsageIndex()))
	offsetVertexElement := ordinaryVertexElement
	offsetVertexElement.SetOffset(13)
	check("vertex-element.copy.offset", "VERTEX_ELEMENT", "12,13", fmt.Sprintf("%d,%d", ordinaryVertexElement.Offset(), offsetVertexElement.Offset()))
	formatVertexElement := ordinaryVertexElement
	formatVertexElement.SetVertexElementFormat(graphics.VertexElementFormatHalfVector4)
	check("vertex-element.copy.format", "VERTEX_ELEMENT", "2,11", fmt.Sprintf("%d,%d", ordinaryVertexElement.VertexElementFormat(), formatVertexElement.VertexElementFormat()))
	usageVertexElement := ordinaryVertexElement
	usageVertexElement.SetVertexElementUsage(graphics.VertexElementUsageTangent)
	check("vertex-element.copy.usage", "VERTEX_ELEMENT", "2,5", fmt.Sprintf("%d,%d", ordinaryVertexElement.VertexElementUsage(), usageVertexElement.VertexElementUsage()))
	indexVertexElement := ordinaryVertexElement
	indexVertexElement.SetUsageIndex(8)
	check("vertex-element.copy.usage-index", "VERTEX_ELEMENT", "7,8", fmt.Sprintf("%d,%d", ordinaryVertexElement.UsageIndex(), indexVertexElement.UsageIndex()))

	unknownVertexElement := graphics.NewVertexElement(123, graphics.VertexElementFormat(12345), graphics.VertexElementUsage(-23456), -456)
	check("vertex-element.undefined.storage", "VERTEX_ELEMENT", "123,12345,-23456,-456", fmt.Sprintf("%d,%d,%d,%d", unknownVertexElement.Offset(), unknownVertexElement.VertexElementFormat(), unknownVertexElement.VertexElementUsage(), unknownVertexElement.UsageIndex()))
	boundaryVertexElement := graphics.NewVertexElement(math.MinInt32, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTessellateFactor, math.MaxInt32)
	check("vertex-element.int32-boundaries", "VERTEX_ELEMENT", "-2147483648,2147483647", fmt.Sprintf("%d,%d", boundaryVertexElement.Offset(), boundaryVertexElement.UsageIndex()))
	check("vertex-element.equals-object", "VERTEX_ELEMENT", "true,false,false,false", fmt.Sprintf("%t,%t,%t,%t", ordinaryVertexElement.Equals(ordinaryVertexElement), ordinaryVertexElement.Equals(&ordinaryVertexElement), ordinaryVertexElement.Equals(nil), ordinaryVertexElement.Equals(int32(12))))
	check("vertex-element.equals-field-differences", "VERTEX_ELEMENT", "false,false,false,false", fmt.Sprintf("%t,%t,%t,%t", ordinaryVertexElement.Equals(offsetVertexElement), ordinaryVertexElement.Equals(formatVertexElement), ordinaryVertexElement.Equals(usageVertexElement), ordinaryVertexElement.Equals(indexVertexElement)))
	unknownVertexElementCopy := unknownVertexElement
	check("vertex-element.equals-undefined", "VERTEX_ELEMENT", true, unknownVertexElement.Equals(unknownVertexElementCopy))
	check("vertex-element.operators", "VERTEX_ELEMENT", "true,true", fmt.Sprintf("%t,%t", graphics.VertexElementOperatorEqualityByVertexElementAndVertexElement(ordinaryVertexElement, ordinaryVertexElement), graphics.VertexElementOperatorInequalityByVertexElementAndVertexElement(ordinaryVertexElement, offsetVertexElement)))

	check("vertex-element.hash.zero-fallback", "VERTEX_ELEMENT", int32(math.MaxInt32), zeroVertexElement.GetHashCode())
	check("vertex-element.hash.ordinary", "VERTEX_ELEMENT", int32(11), ordinaryVertexElement.GetHashCode())
	check("vertex-element.hash.negative", "VERTEX_ELEMENT", int32(3), graphics.NewVertexElement(-16, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTangent, -3).GetHashCode())
	check("vertex-element.hash.undefined", "VERTEX_ELEMENT", int32(27162), unknownVertexElement.GetHashCode())
	check("vertex-element.hash.boundaries", "VERTEX_ELEMENT", int32(-8), boundaryVertexElement.GetHashCode())
	check("vertex-element.hash.nonzero-collision", "VERTEX_ELEMENT", int32(math.MaxInt32), graphics.NewVertexElement(1, graphics.VertexElementFormatVector3, graphics.VertexElementUsageNormal, 0).GetHashCode())

	check("vertex-element.string.zero", "VERTEX_ELEMENT", "{Offset:0 Format:Single Usage:Position UsageIndex:0}", zeroVertexElement.ToString())
	check("vertex-element.string.ordinary", "VERTEX_ELEMENT", "{Offset:12 Format:Vector3 Usage:TextureCoordinate UsageIndex:7}", ordinaryVertexElement.ToString())
	check("vertex-element.string.negative", "VERTEX_ELEMENT", "{Offset:-16 Format:HalfVector4 Usage:Tangent UsageIndex:-3}", graphics.NewVertexElement(-16, graphics.VertexElementFormatHalfVector4, graphics.VertexElementUsageTangent, -3).ToString())
	check("vertex-element.string.undefined", "VERTEX_ELEMENT", "{Offset:123 Format:12345 Usage:-23456 UsageIndex:-456}", unknownVertexElement.ToString())
	check("vertex-element.string.boundaries", "VERTEX_ELEMENT", "{Offset:-2147483648 Format:HalfVector4 Usage:TessellateFactor UsageIndex:2147483647}", boundaryVertexElement.ToString())

	check("curve.enums.continuity", "CURVE_ENUMS", "0,1", fmt.Sprintf("%d,%d", framework.CurveContinuitySmooth, framework.CurveContinuityStep))
	check("curve.enums.tangent", "CURVE_ENUMS", "0,1,2", fmt.Sprintf("%d,%d,%d", framework.CurveTangentFlat, framework.CurveTangentLinear, framework.CurveTangentSmooth))
	check("curve.enums.loop", "CURVE_ENUMS", "0,1,2,3,4", fmt.Sprintf("%d,%d,%d,%d,%d", framework.CurveLoopTypeConstant, framework.CurveLoopTypeCycle, framework.CurveLoopTypeCycleOffset, framework.CurveLoopTypeOscillate, framework.CurveLoopTypeLinear))

	curveKey := framework.NewCurveKeyBySingleAndSingle(1, 2)
	check("curve.key.defaults", "CURVE_KEY", "0x00000000,0x00000000,0", fmt.Sprintf("%s,%s,%d", bits(curveKey.TangentIn()), bits(curveKey.TangentOut()), curveKey.Continuity()))
	fullCurveKey := framework.NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(1, 2, 3, 4, framework.CurveContinuityStep)
	curveKeyClone := fullCurveKey.Clone()
	check("curve.key.clone.identity", "CURVE_KEY", true, curveKeyClone != fullCurveKey && curveKeyClone.EqualsByCurveKey(fullCurveKey))
	curveKeyClone.SetValue(9)
	check("curve.key.clone.independent", "CURVE_KEY", "0x40000000,0x41100000", floatBits(fullCurveKey.Value(), curveKeyClone.Value()))
	comparisonLess, comparisonLessError := fullCurveKey.CompareTo(framework.NewCurveKeyBySingleAndSingle(2, 0))
	comparisonEqual, comparisonEqualError := fullCurveKey.CompareTo(framework.NewCurveKeyBySingleAndSingle(1, 99))
	check("curve.key.compare.finite", "CURVE_KEY", "-1,0,true", fmt.Sprintf("%d,%d,%t", comparisonLess, comparisonEqual, comparisonLessError == nil && comparisonEqualError == nil))
	comparisonNaN, comparisonNaNError := framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1))
	check("curve.key.compare.nan", "CURVE_KEY", "0,true", fmt.Sprintf("%d,%t", comparisonNaN, comparisonNaNError == nil))
	comparisonNaNFinite, _ := framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(0, 1))
	comparisonFiniteNaN, _ := framework.NewCurveKeyBySingleAndSingle(0, 1).CompareTo(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1))
	check("curve.key.compare.nan-order", "CURVE_KEY", "-1,1", fmt.Sprintf("%d,%d", comparisonNaNFinite, comparisonFiniteNaN))
	check("curve.key.nan.equality", "CURVE_KEY", false, framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1).EqualsByCurveKey(framework.NewCurveKeyBySingleAndSingle(float32(math.NaN()), 1)))
	check("curve.key.hash", "CURVE_KEY", int32(4194305), fullCurveKey.GetHashCode())
	check("curve.key.operators", "CURVE_KEY", "true,true", fmt.Sprintf("%t,%t", framework.CurveKeyOperatorEqualityByCurveKeyAndCurveKey(fullCurveKey, fullCurveKey.Clone()), framework.CurveKeyOperatorInequalityByCurveKeyAndCurveKey(nil, fullCurveKey)))
	_, comparisonNilError := fullCurveKey.CompareTo(nil)
	check("curve.key.compare.nil-error", "CURVE_KEY", true, comparisonNilError != nil)

	mustAddCurveKey := func(collection *framework.CurveKeyCollection, key *framework.CurveKey) {
		if err := collection.Add(key); err != nil {
			panic(err)
		}
	}
	mustCurveKeyAt := func(collection *framework.CurveKeyCollection, index int32) *framework.CurveKey {
		key, err := collection.Item(index)
		if err != nil {
			panic(err)
		}
		return key
	}
	curveCollection := framework.NewCurveKeyCollection()
	duplicateA := framework.NewCurveKeyBySingleAndSingle(1, 10)
	duplicateB := framework.NewCurveKeyBySingleAndSingle(1, 20)
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(2, 30))
	mustAddCurveKey(curveCollection, duplicateA)
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(0, 0))
	mustAddCurveKey(curveCollection, duplicateB)
	check("curve.collection.order", "CURVE_COLLECTION", "0x00000000,0x3f800000,0x3f800000,0x40000000", floatBits(mustCurveKeyAt(curveCollection, 0).Position(), mustCurveKeyAt(curveCollection, 1).Position(), mustCurveKeyAt(curveCollection, 2).Position(), mustCurveKeyAt(curveCollection, 3).Position()))
	check("curve.collection.duplicate-identity", "CURVE_COLLECTION", true, mustCurveKeyAt(curveCollection, 1) == duplicateA && mustCurveKeyAt(curveCollection, 2) == duplicateB)
	duplicateA.SetValue(11)
	equalDuplicate := framework.NewCurveKeyBySingleAndSingle(1, 11)
	check("curve.collection.value-equality", "CURVE_COLLECTION", "true,1", fmt.Sprintf("%t,%d", curveCollection.Contains(equalDuplicate), curveCollection.IndexOf(equalDuplicate)))
	replacement := framework.NewCurveKeyBySingleAndSingle(3, 40)
	if err := curveCollection.SetItem(0, replacement); err != nil {
		panic(err)
	}
	check("curve.collection.item-reorders", "CURVE_COLLECTION", "0x3f800000,0x3f800000,0x40000000,0x40400000", floatBits(mustCurveKeyAt(curveCollection, 0).Position(), mustCurveKeyAt(curveCollection, 1).Position(), mustCurveKeyAt(curveCollection, 2).Position(), mustCurveKeyAt(curveCollection, 3).Position()))
	samePosition := framework.NewCurveKeyBySingleAndSingle(3, 41)
	if err := curveCollection.SetItem(3, samePosition); err != nil {
		panic(err)
	}
	check("curve.collection.item-identity", "CURVE_COLLECTION", true, mustCurveKeyAt(curveCollection, 3) == samePosition)
	_, negativeItemError := curveCollection.Item(-1)
	setEndError := curveCollection.SetItem(curveCollection.Count(), samePosition)
	removeNegativeError := curveCollection.RemoveAt(-1)
	check("curve.collection.index-errors", "CURVE_COLLECTION", "true,true,true", fmt.Sprintf("%t,%t,%t", negativeItemError != nil, setEndError != nil, removeNegativeError != nil))
	destination := make([]*framework.CurveKey, 6)
	copyError := curveCollection.CopyTo(destination, 1)
	check("curve.collection.copyto", "CURVE_COLLECTION", true, copyError == nil && destination[1] == duplicateA && destination[4] == samePosition)
	check("curve.collection.copyto-errors", "CURVE_COLLECTION", true, curveCollection.CopyTo(nil, 0) != nil && curveCollection.CopyTo(make([]*framework.CurveKey, 4), 1) != nil)
	collectionClone := curveCollection.Clone()
	check("curve.collection.clone-depth", "CURVE_COLLECTION", true, collectionClone != curveCollection && mustCurveKeyAt(collectionClone, 0) == mustCurveKeyAt(curveCollection, 0))
	iterator := curveCollection.GetEnumerator()
	firstEnumerated, firstEnumeratedOK, firstEnumeratedError := iterator.Next()
	mustAddCurveKey(curveCollection, framework.NewCurveKeyBySingleAndSingle(4, 50))
	_, _, invalidatedError := iterator.Next()
	check("curve.collection.enumerator-invalidation", "CURVE_COLLECTION", true, firstEnumerated == duplicateA && firstEnumeratedOK && firstEnumeratedError == nil && invalidatedError != nil)
	check("curve.collection.remove-equal", "CURVE_COLLECTION", true, curveCollection.Remove(equalDuplicate) && !curveCollection.Contains(equalDuplicate))
	curveCollection.Clear()
	check("curve.collection.clear-readonly", "CURVE_COLLECTION", "0,false", fmt.Sprintf("%d,%t", curveCollection.Count(), curveCollection.IsReadOnly()))

	tangentCurve := framework.NewCurve()
	for _, pair := range [][2]float32{{0, 0}, {1, 10}, {3, 30}} {
		mustAddCurveKey(tangentCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(pair[0], pair[1]))
	}
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangent(1, framework.CurveTangentFlat); err != nil {
		panic(err)
	}
	check("curve.tangent.flat", "CURVE_TANGENTS", "0x00000000,0x00000000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangent(1, framework.CurveTangentLinear); err != nil {
		panic(err)
	}
	check("curve.tangent.linear", "CURVE_TANGENTS", "0x41200000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	tangentCurve.ComputeTangentsByCurveTangent(framework.CurveTangentSmooth)
	check("curve.tangent.smooth-first", "CURVE_TANGENTS", "0x00000000,0x41200000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 0).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 0).TangentOut()))
	check("curve.tangent.smooth-middle", "CURVE_TANGENTS", "0x41200000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	check("curve.tangent.smooth-last", "CURVE_TANGENTS", "0x41a00000,0x00000000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 2).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 2).TangentOut()))
	if err := tangentCurve.ComputeTangentByInt32AndCurveTangentAndCurveTangent(1, framework.CurveTangentFlat, framework.CurveTangentSmooth); err != nil {
		panic(err)
	}
	check("curve.tangent.mixed", "CURVE_TANGENTS", "0x00000000,0x41a00000", floatBits(mustCurveKeyAt(tangentCurve.Keys(), 1).TangentIn(), mustCurveKeyAt(tangentCurve.Keys(), 1).TangentOut()))
	check("curve.tangent.invalid-index", "CURVE_TANGENTS", true, tangentCurve.ComputeTangentByInt32AndCurveTangent(-1, framework.CurveTangentFlat) != nil)

	emptyCurve := framework.NewCurve()
	check("curve.evaluate.empty", "CURVE_EVALUATE", "0x00000000", bits(emptyCurve.Evaluate(5)))
	check("curve.evaluate.defaults", "CURVE_EVALUATE", "0,0,true", fmt.Sprintf("%d,%d,%t", emptyCurve.PreLoop(), emptyCurve.PostLoop(), emptyCurve.IsConstant()))
	singleCurve := framework.NewCurve()
	mustAddCurveKey(singleCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(2, 7))
	check("curve.evaluate.single", "CURVE_EVALUATE", "0x40e00000,0x40e00000,0x40e00000", floatBits(singleCurve.Evaluate(-1), singleCurve.Evaluate(2), singleCurve.Evaluate(9)))
	hermiteCurve := framework.NewCurve()
	mustAddCurveKey(hermiteCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(0, 0))
	mustAddCurveKey(hermiteCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 10))
	check("curve.evaluate.hermite", "CURVE_EVALUATE", "0x3fc80000", bits(hermiteCurve.Evaluate(0.25)))
	asymmetricCurve := framework.NewCurve()
	mustAddCurveKey(asymmetricCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingle(0, 0, 99, 4))
	mustAddCurveKey(asymmetricCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingle(2, 10, -2, 77))
	check("curve.evaluate.hermite-asymmetric", "CURVE_EVALUATE", "0x40b80000", bits(asymmetricCurve.Evaluate(1)))
	stepCurve := framework.NewCurve()
	mustAddCurveKey(stepCurve.Keys(), framework.NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(0, 2, 0, 0, framework.CurveContinuityStep))
	mustAddCurveKey(stepCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 9))
	check("curve.evaluate.step", "CURVE_EVALUATE", "0x40000000,0x41100000", floatBits(stepCurve.Evaluate(0.999), stepCurve.Evaluate(1)))
	duplicateCurve := framework.NewCurve()
	mustAddCurveKey(duplicateCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 10))
	mustAddCurveKey(duplicateCurve.Keys(), framework.NewCurveKeyBySingleAndSingle(1, 20))
	check("curve.evaluate.duplicate", "CURVE_EVALUATE", "0x41200000", bits(duplicateCurve.Evaluate(1)))
	curveClone := hermiteCurve.Clone()
	check("curve.evaluate.clone-depth", "CURVE_EVALUATE", true, curveClone != hermiteCurve && curveClone.Keys() != hermiteCurve.Keys() && mustCurveKeyAt(curveClone.Keys(), 0) == mustCurveKeyAt(hermiteCurve.Keys(), 0))

	newLoopCurve := func() *framework.Curve {
		curve := framework.NewCurve()
		mustAddCurveKey(curve.Keys(), framework.NewCurveKeyBySingleAndSingle(5, 0))
		mustAddCurveKey(curve.Keys(), framework.NewCurveKeyBySingleAndSingle(7, 10))
		return curve
	}
	constantLoop := newLoopCurve()
	check("curve.loop.constant", "CURVE_LOOPS", "0x00000000,0x41200000", floatBits(constantLoop.Evaluate(4), constantLoop.Evaluate(8)))
	cycleLoop := newLoopCurve()
	cycleLoop.SetPreLoop(framework.CurveLoopTypeCycle)
	cycleLoop.SetPostLoop(framework.CurveLoopTypeCycle)
	check("curve.loop.cycle", "CURVE_LOOPS", "0x40a00000,0x40a00000", floatBits(cycleLoop.Evaluate(4), cycleLoop.Evaluate(8)))
	offsetLoop := newLoopCurve()
	offsetLoop.SetPreLoop(framework.CurveLoopTypeCycleOffset)
	offsetLoop.SetPostLoop(framework.CurveLoopTypeCycleOffset)
	check("curve.loop.cycle-offset", "CURVE_LOOPS", "0xc0a00000,0x41700000", floatBits(offsetLoop.Evaluate(4), offsetLoop.Evaluate(8)))
	oscillateLoop := newLoopCurve()
	oscillateLoop.SetPreLoop(framework.CurveLoopTypeOscillate)
	oscillateLoop.SetPostLoop(framework.CurveLoopTypeOscillate)
	check("curve.loop.oscillate", "CURVE_LOOPS", "0x40a00000,0x40a00000", floatBits(oscillateLoop.Evaluate(4), oscillateLoop.Evaluate(8)))
	linearLoop := newLoopCurve()
	mustCurveKeyAt(linearLoop.Keys(), 0).SetTangentIn(2)
	mustCurveKeyAt(linearLoop.Keys(), 1).SetTangentOut(3)
	linearLoop.SetPreLoop(framework.CurveLoopTypeLinear)
	linearLoop.SetPostLoop(framework.CurveLoopTypeLinear)
	check("curve.loop.linear", "CURVE_LOOPS", "0xc0000000,0x41800000", floatBits(linearLoop.Evaluate(4), linearLoop.Evaluate(9)))
	check("curve.loop.negative-cycle", "CURVE_LOOPS", "0x41200000", bits(cycleLoop.Evaluate(3)))
	check("curve.loop.negative-cycle-offset", "CURVE_LOOPS", "0xc1200000", bits(offsetLoop.Evaluate(3)))
	check("curve.loop.negative-oscillate", "CURVE_LOOPS", "0x41200000", bits(oscillateLoop.Evaluate(3)))
	check("curve.loop.multiple-offset", "CURVE_LOOPS", "0xc1a00000,0x41a00000", floatBits(offsetLoop.Evaluate(1), offsetLoop.Evaluate(9)))

	check("packed.alpha.zero", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(0).PackedValue())
	check("packed.alpha.ordinary", "PACKED_ALPHA", uint8(128), packedvector.NewAlpha8(0.5).PackedValue())
	check("packed.alpha.clamp-low", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(-1).PackedValue())
	check("packed.alpha.clamp-high", "PACKED_ALPHA", uint8(255), packedvector.NewAlpha8(2).PackedValue())
	check("packed.alpha.tie-even-zero", "PACKED_ALPHA", uint8(0), packedvector.NewAlpha8(float32(0.5/255)).PackedValue())
	check("packed.alpha.tie-even-two", "PACKED_ALPHA", uint8(2), packedvector.NewAlpha8(float32(2.5/255)).PackedValue())
	check("packed.alpha.non-finite", "PACKED_ALPHA", "0,255,0", fmt.Sprintf("%d,%d,%d", packedvector.NewAlpha8(float32(math.NaN())).PackedValue(), packedvector.NewAlpha8(float32(math.Inf(1))).PackedValue(), packedvector.NewAlpha8(float32(math.Inf(-1))).PackedValue()))

	check("packed.bgr565.ordinary", "PACKED_16BIT_COLOR", uint16(17431), packedvector.NewBgr565BySingleAndSingleAndSingle(0.25, 0.5, 0.75).PackedValue())
	check("packed.bgr565.lanes", "PACKED_16BIT_COLOR", "63488,2016,31", fmt.Sprintf("%d,%d,%d", packedvector.NewBgr565BySingleAndSingleAndSingle(1, 0, 0).PackedValue(), packedvector.NewBgr565BySingleAndSingleAndSingle(0, 1, 0).PackedValue(), packedvector.NewBgr565BySingleAndSingleAndSingle(0, 0, 1).PackedValue()))
	check("packed.bgra4444.ordinary", "PACKED_16BIT_COLOR", uint16(50025), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0.2, 0.4, 0.6, 0.8).PackedValue())
	check("packed.bgra4444.lanes", "PACKED_16BIT_COLOR", "3840,240,15,61440", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewBgra4444BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))
	check("packed.bgra5551.ordinary", "PACKED_16BIT_COLOR", uint16(8727), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue())
	check("packed.bgra5551.alpha-threshold", "PACKED_16BIT_COLOR", "0,0,32768", fmt.Sprintf("%d,%d,%d", packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 0)).PackedValue(), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, 0.5).PackedValue(), packedvector.NewBgra5551BySingleAndSingleAndSingleAndSingle(0, 0, 0, math.Nextafter32(0.5, 1)).PackedValue()))

	check("packed.byte4.raw-domain", "PACKED_BYTE4", uint32(4286578944), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(-1, 1, 128, 256).PackedValue())
	check("packed.byte4.ties", "PACKED_BYTE4", uint32(67240448), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(0.5, 1.5, 2.5, 3.5).PackedValue())
	check("packed.byte4.non-finite", "PACKED_BYTE4", uint32(0x8000ff00), packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 127.5).PackedValue())
	check("packed.byte4.decode", "PACKED_BYTE4", "0x00000000,0x3f800000,0x43000000,0x437f0000", func() string {
		value := packedvector.NewByte4BySingleAndSingleAndSingleAndSingle(-1, 1, 128, 256).ToVector4()
		return floatBits(value.X, value.Y, value.Z, value.W)
	}())

	check("packed.half.single", "PACKED_HALF", uint16(13653), packedvector.NewHalfSingle(float32(1.0/3.0)).PackedValue())
	check("packed.half.vector2-order", "PACKED_HALF", uint32(3221240832), packedvector.NewHalfVector2BySingleAndSingle(1, -2).PackedValue())
	check("packed.half.vector4-order", "PACKED_HALF", uint64(9223433612727172096), packedvector.NewHalfVector4BySingleAndSingleAndSingleAndSingle(1, -2, 0.5, math.Float32frombits(0x80000000)).PackedValue())
	halfFixtures := []struct {
		id      string
		input   uint32
		packed  uint16
		decoded uint32
		text    string
	}{
		{"positive-zero", 0x00000000, 0x0000, 0x00000000, "0"},
		{"negative-zero", 0x80000000, 0x8000, 0x80000000, "0"},
		{"smallest-subnormal", 0x33800000, 0x0001, 0x33800000, "5.960464E-08"},
		{"largest-subnormal", 0x387fc000, 0x03ff, 0x387fc000, "6.097555E-05"},
		{"smallest-normal", 0x38800000, 0x0400, 0x38800000, "6.103516E-05"},
		{"tie-even-low", 0x3f801000, 0x3c00, 0x3f800000, "1"},
		{"tie-even-high", 0x3f803000, 0x3c02, 0x3f804000, "1.001953"},
		{"maximum-conventional-finite", 0x477fe000, 0x7bff, 0x477fe000, "65504"},
		{"exponent31-boundary", 0x477ff000, 0x7c00, 0x47800000, "65536"},
		{"positive-infinity", 0x7f800000, 0x7fff, 0x47ffe000, "131008"},
		{"negative-infinity", 0xff800000, 0xffff, 0xc7ffe000, "-131008"},
		{"positive-nan", 0x7fc12345, 0x7fff, 0x47ffe000, "131008"},
		{"negative-nan", 0xffc12345, 0xffff, 0xc7ffe000, "-131008"},
	}
	for _, fixture := range halfFixtures {
		value := packedvector.NewHalfSingle(math.Float32frombits(fixture.input))
		actual := fmt.Sprintf("%d,%08X,%s", value.PackedValue(), math.Float32bits(value.ToSingle()), value.ToString())
		expected := fmt.Sprintf("%d,%08X,%s", fixture.packed, fixture.decoded, fixture.text)
		check("packed.half."+fixture.id, "PACKED_HALF", expected, actual)
	}

	check("packed.normalized-byte2", "PACKED_NORMALIZED_BYTE", uint16(16513), packedvector.NewNormalizedByte2BySingleAndSingle(-1, 0.5).PackedValue())
	check("packed.normalized-byte4-endpoints", "PACKED_NORMALIZED_BYTE", uint32(2139029633), packedvector.NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue())
	check("packed.normalized-byte4-ties", "PACKED_NORMALIZED_BYTE", uint32(4261413376), packedvector.NewNormalizedByte4BySingleAndSingleAndSingleAndSingle(float32(0.5/127), float32(1.5/127), float32(-0.5/127), float32(-1.5/127)).PackedValue())
	var normalizedByteMinimum packedvector.NormalizedByte2
	normalizedByteMinimum.SetPackedValue(0x0080)
	check("packed.normalized-byte-minimum-decodes-minus-one", "PACKED_NORMALIZED_BYTE", "0xbf800000", bits(normalizedByteMinimum.ToVector2().X))

	check("packed.normalized-short2", "PACKED_NORMALIZED_SHORT", uint32(1073774593), packedvector.NewNormalizedShort2BySingleAndSingle(-1, 0.5).PackedValue())
	check("packed.normalized-short4-endpoints", "PACKED_NORMALIZED_SHORT", uint64(9223231295071485953), packedvector.NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(-2, 0, 1, 2).PackedValue())
	check("packed.normalized-short4-ties", "PACKED_NORMALIZED_SHORT", uint64(18446181123756261376), packedvector.NewNormalizedShort4BySingleAndSingleAndSingleAndSingle(float32(0.5/32767), float32(1.5/32767), float32(-0.5/32767), float32(-1.5/32767)).PackedValue())
	var normalizedShortMinimum packedvector.NormalizedShort2
	normalizedShortMinimum.SetPackedValue(0x00008000)
	check("packed.normalized-short-minimum-decodes-minus-one", "PACKED_NORMALIZED_SHORT", "0xbf800000", bits(normalizedShortMinimum.ToVector2().X))

	check("packed.rg32.ordinary", "PACKED_RG_RGBA", uint32(3221176320), packedvector.NewRg32BySingleAndSingle(0.25, 0.75).PackedValue())
	check("packed.rg32.lanes", "PACKED_RG_RGBA", "65535,4294901760", fmt.Sprintf("%d,%d", packedvector.NewRg32BySingleAndSingle(1, 0).PackedValue(), packedvector.NewRg32BySingleAndSingle(0, 1).PackedValue()))
	check("packed.rgba1010102.ordinary", "PACKED_RG_RGBA", uint32(2952265984), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 0.5).PackedValue())
	check("packed.rgba1010102.lanes", "PACKED_RG_RGBA", "1023,1047552,1072693248,3221225472", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewRgba1010102BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))
	check("packed.rgba64.ordinary", "PACKED_RG_RGBA", uint64(18446673702817906688), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1).PackedValue())
	check("packed.rgba64.lanes", "PACKED_RG_RGBA", "65535,4294901760,281470681743360,18446462598732840960", fmt.Sprintf("%d,%d,%d,%d", packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(1, 0, 0, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 1, 0, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 1, 0).PackedValue(), packedvector.NewRgba64BySingleAndSingleAndSingleAndSingle(0, 0, 0, 1).PackedValue()))

	check("packed.short2.endpoints", "PACKED_SHORT", uint32(2147450880), packedvector.NewShort2BySingleAndSingle(-32768, 32767).PackedValue())
	check("packed.short4.ordinary", "PACKED_SHORT", uint64(9223090574762868736), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(-32768, -1.5, 2.5, 32768).PackedValue())
	check("packed.short4.ties", "PACKED_SHORT", uint64(562954248257536), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(-0.5, -1.5, 0.5, 1.5).PackedValue())
	check("packed.short4.non-finite", "PACKED_SHORT", uint64(0x7fff80007fff0000), packedvector.NewShort4BySingleAndSingleAndSingleAndSingle(float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 32768).PackedValue())

	interfaceInput := framework.Vector4{X: -2, Y: 0.5, Z: 2, W: 0.25}
	alphaInterface := packedvector.Alpha8{}
	var alphaPacked packedvector.IPackedVectorOfTPacked[uint8] = &alphaInterface
	alphaPacked.PackFromVector4(interfaceInput)
	check("packed.interface.alpha8", "PACKED_INTERFACE", "64,0x00000000,0x00000000,0x00000000,0x3e808081", fmt.Sprintf("%d,%s", alphaPacked.PackedValue(), floatBits(alphaPacked.ToVector4().X, alphaPacked.ToVector4().Y, alphaPacked.ToVector4().Z, alphaPacked.ToVector4().W)))
	bgrInterface := packedvector.Bgr565{}
	var bgrPacked packedvector.IPackedVectorOfTPacked[uint16] = &bgrInterface
	bgrPacked.PackFromVector4(interfaceInput)
	check("packed.interface.bgr565", "PACKED_INTERFACE", "1055,0x00000000,0x3f020821,0x3f800000,0x3f800000", fmt.Sprintf("%d,%s", bgrPacked.PackedValue(), floatBits(bgrPacked.ToVector4().X, bgrPacked.ToVector4().Y, bgrPacked.ToVector4().Z, bgrPacked.ToVector4().W)))
	halfInterface := packedvector.HalfSingle{}
	var halfPacked packedvector.IPackedVectorOfTPacked[uint16] = &halfInterface
	halfPacked.PackFromVector4(interfaceInput)
	check("packed.interface.halfsingle", "PACKED_INTERFACE", "49152,0xc0000000,0x00000000,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", halfPacked.PackedValue(), floatBits(halfPacked.ToVector4().X, halfPacked.ToVector4().Y, halfPacked.ToVector4().Z, halfPacked.ToVector4().W)))
	normalizedByteInterface := packedvector.NormalizedByte2{}
	var normalizedBytePacked packedvector.IPackedVectorOfTPacked[uint16] = &normalizedByteInterface
	normalizedBytePacked.PackFromVector4(interfaceInput)
	check("packed.interface.normalizedbyte2", "PACKED_INTERFACE", "16513,0xbf800000,0x3f010204,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", normalizedBytePacked.PackedValue(), floatBits(normalizedBytePacked.ToVector4().X, normalizedBytePacked.ToVector4().Y, normalizedBytePacked.ToVector4().Z, normalizedBytePacked.ToVector4().W)))
	rgInterface := packedvector.Rg32{}
	var rgPacked packedvector.IPackedVectorOfTPacked[uint32] = &rgInterface
	rgPacked.PackFromVector4(interfaceInput)
	check("packed.interface.rg32", "PACKED_INTERFACE", "2147483648,0x00000000,0x3f000080,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", rgPacked.PackedValue(), floatBits(rgPacked.ToVector4().X, rgPacked.ToVector4().Y, rgPacked.ToVector4().Z, rgPacked.ToVector4().W)))
	shortInterface := packedvector.Short2{}
	var shortPacked packedvector.IPackedVectorOfTPacked[uint32] = &shortInterface
	shortPacked.PackFromVector4(interfaceInput)
	check("packed.interface.short2", "PACKED_INTERFACE", "65534,0xc0000000,0x00000000,0x00000000,0x3f800000", fmt.Sprintf("%d,%s", shortPacked.PackedValue(), floatBits(shortPacked.ToVector4().X, shortPacked.ToVector4().Y, shortPacked.ToVector4().Z, shortPacked.ToVector4().W)))

	var packedValue64 packedvector.HalfVector4
	packedValue64.SetPackedValue(0xfedcba9876543210)
	check("packed.value64.hash-string", "PACKED_INTERFACE", "-2004318072,{X:0.1894531 Y:25920 Z:-0.8242188 W:-112384}", fmt.Sprintf("%d,%s", packedValue64.GetHashCode(), packedValue64.ToString()))
	packedValue64Copy := packedValue64
	packedValue64Copy.SetPackedValue(0)
	check("packed.value64.copy-and-equality", "PACKED_INTERFACE", "true,true,false,true", fmt.Sprintf("%t,%t,%t,%t", packedValue64.EqualsByObject(packedValue64), packedvector.HalfVector4OperatorEqualityByHalfVector4AndHalfVector4(packedValue64, packedValue64), packedValue64.EqualsByHalfVector4(packedValue64Copy), packedvector.HalfVector4OperatorInequalityByHalfVector4AndHalfVector4(packedValue64, packedValue64Copy)))

	return report
}
