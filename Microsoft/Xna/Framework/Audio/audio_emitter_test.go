package audio

import (
	"math"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestAudioEmitterConstructorDefaults pins the exact constructed state,
// including the same negative-zero Z asymmetry the listener has and the four
// members the reference constructor initializes to 1.
func TestAudioEmitterConstructorDefaults(t *testing.T) {
	emitter := NewAudioEmitter()
	const negativeZero = uint32(0x80000000)

	if got := vectorBits(emitter.Position()); got != [3]uint32{0, 0, negativeZero} {
		t.Fatalf("Position default bits = %v", got)
	}
	if got := vectorBits(emitter.Velocity()); got != [3]uint32{0, 0, negativeZero} {
		t.Fatalf("Velocity default bits = %v", got)
	}
	if got := vectorBits(emitter.Forward()); got != vectorBits(framework.Vector3Forward()) {
		t.Fatalf("Forward default bits = %v", got)
	}
	if got := vectorBits(emitter.Up()); got != vectorBits(framework.Vector3Up()) {
		t.Fatalf("Up default bits = %v", got)
	}
	if emitter.DopplerScale() != 1 {
		t.Fatalf("DopplerScale default = %v", emitter.DopplerScale())
	}
	if emitter.channelCount != 1 || emitter.channelRadius != 1 || emitter.curveDistanceScaler != 1 {
		t.Fatalf("unprojected emitterData defaults = %d %v %v",
			emitter.channelCount, emitter.channelRadius, emitter.curveDistanceScaler)
	}
}

// TestAudioEmitterRoundTripsEveryAssignedValue mirrors the listener: the double
// flip is invisible for every bit pattern.
func TestAudioEmitterRoundTripsEveryAssignedValue(t *testing.T) {
	inputs := []framework.Vector3{
		{X: 1, Y: 2, Z: 3},
		framework.Vector3Zero(),
		{X: 0, Y: 0, Z: float32(math.Copysign(0, -1))},
		{X: float32(math.Inf(-1)), Y: float32(math.Inf(1)), Z: float32(math.Inf(1))},
		{X: float32(math.NaN()), Y: float32(math.NaN()), Z: float32(math.NaN())},
	}
	for _, input := range inputs {
		emitter := NewAudioEmitter()
		emitter.SetPosition(input)
		emitter.SetVelocity(input)
		emitter.SetForward(input)
		emitter.SetUp(input)
		for name, got := range map[string]framework.Vector3{
			"Position": emitter.Position(),
			"Velocity": emitter.Velocity(),
			"Forward":  emitter.Forward(),
			"Up":       emitter.Up(),
		} {
			if vectorBits(got) != vectorBits(input) {
				t.Fatalf("%s round-trip bits = %v, want %v", name, vectorBits(got), vectorBits(input))
			}
		}
	}
}

// TestAudioEmitterDopplerScaleValidation is the authoritative table for the
// `bge.un.s` guard: the store is reached whenever the ordered comparison
// value >= 0 succeeds or the comparison is unordered, so exactly the
// negative-ordered values throw.
func TestAudioEmitterDopplerScaleValidation(t *testing.T) {
	negativeZero := float32(math.Copysign(0, -1))
	for _, testCase := range []struct {
		name     string
		value    float32
		accepted bool
	}{
		{"ordinary-positive", 2.5, true},
		{"one", 1, true},
		{"smallest-positive-denormal", math.SmallestNonzeroFloat32, true},
		{"largest-finite", math.MaxFloat32, true},
		{"positive-zero", 0, true},
		{"negative-zero", negativeZero, true},
		{"positive-infinity", float32(math.Inf(1)), true},
		{"nan", float32(math.NaN()), true},
		{"ordinary-negative", -2.5, false},
		{"smallest-negative-denormal", -math.SmallestNonzeroFloat32, false},
		{"lowest-finite", -math.MaxFloat32, false},
		{"negative-infinity", float32(math.Inf(-1)), false},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			emitter := NewAudioEmitter()
			err := emitter.SetDopplerScale(testCase.value)
			if testCase.accepted {
				if err != nil {
					t.Fatalf("SetDopplerScale(%v) = %v, want nil", testCase.value, err)
				}
				if math.Float32bits(emitter.DopplerScale()) != math.Float32bits(testCase.value) {
					t.Fatalf("SetDopplerScale(%v) stored bits 0x%08x",
						testCase.value, math.Float32bits(emitter.DopplerScale()))
				}
				return
			}
			if err == nil {
				t.Fatalf("SetDopplerScale(%v) = nil, want an error", testCase.value)
			}
			if !strings.Contains(err.Error(), invalidEmitterDopplerScale) {
				t.Fatalf("SetDopplerScale(%v) error = %q", testCase.value, err)
			}
			// A rejected assignment leaves the stored value untouched.
			if emitter.DopplerScale() != 1 {
				t.Fatalf("a rejected assignment changed DopplerScale to %v", emitter.DopplerScale())
			}
		})
	}
}

// TestAudioEmitterDopplerScaleGetterIsInfallible states the accessor-level
// claim directly: the compiler rejects a second result here, so the getter
// signature carries no error even though its own setter does.
func TestAudioEmitterDopplerScaleGetterIsInfallible(t *testing.T) {
	emitter := NewAudioEmitter()
	var value float32 = emitter.DopplerScale()
	if value != 1 {
		t.Fatalf("DopplerScale = %v", value)
	}
	// Reading back a NaN the setter accepted must also not fail.
	if err := emitter.SetDopplerScale(float32(math.NaN())); err != nil {
		t.Fatalf("NaN assignment = %v", err)
	}
	if !math.IsNaN(float64(emitter.DopplerScale())) {
		t.Fatalf("stored NaN read back as %v", emitter.DopplerScale())
	}
}

// TestAudioEmitterKeepsReferenceSemantics mirrors the listener check.
func TestAudioEmitterKeepsReferenceSemantics(t *testing.T) {
	emitter := NewAudioEmitter()
	alias := emitter
	if err := alias.SetDopplerScale(4); err != nil {
		t.Fatal(err)
	}
	alias.SetVelocity(framework.Vector3{X: 1, Y: 2, Z: 3})
	if emitter.DopplerScale() != 4 {
		t.Fatalf("aliased DopplerScale mutation was not observed: %v", emitter.DopplerScale())
	}
	if emitter.Velocity() != (framework.Vector3{X: 1, Y: 2, Z: 3}) {
		t.Fatalf("aliased Velocity mutation was not observed: %v", emitter.Velocity())
	}
	if NewAudioEmitter().DopplerScale() != 1 {
		t.Fatal("a separately constructed emitter shares state")
	}
}

// TestAudioEmitterUnrelatedSettersStayInfallible records that one validating
// setter does not make its siblings fallible: the four positional setters take
// no error result at all, which the compiler enforces here.
func TestAudioEmitterUnrelatedSettersStayInfallible(t *testing.T) {
	emitter := NewAudioEmitter()
	degenerate := framework.Vector3Zero()
	emitter.SetPosition(degenerate)
	emitter.SetVelocity(degenerate)
	emitter.SetForward(degenerate)
	emitter.SetUp(degenerate)
	if emitter.Forward() != degenerate || emitter.Up() != degenerate {
		t.Fatalf("degenerate orientation was altered: %v %v", emitter.Forward(), emitter.Up())
	}
}
