package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
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

	return report
}
