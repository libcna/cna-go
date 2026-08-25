package audio

import (
	"math"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func vectorBits(v framework.Vector3) [3]uint32 {
	return [3]uint32{
		math.Float32bits(v.X),
		math.Float32bits(v.Y),
		math.Float32bits(v.Z),
	}
}

// TestFlipHandednessIsABitExactInvolution pins the property that makes every
// public accessor pair on both descriptors an identity round-trip: CIL `neg`
// and Go's unary minus are the same sign-bit flip, so applying it twice
// restores the exact bits for every interesting binary32 value.
func TestFlipHandednessIsABitExactInvolution(t *testing.T) {
	values := []float32{
		0,
		float32(math.Copysign(0, -1)),
		1,
		-1,
		math.SmallestNonzeroFloat32,
		-math.SmallestNonzeroFloat32,
		math.MaxFloat32,
		float32(math.Inf(1)),
		float32(math.Inf(-1)),
		float32(math.NaN()),
	}
	for _, z := range values {
		input := framework.Vector3{X: 3, Y: -4, Z: z}
		once := flipHandedness(input)
		twice := flipHandedness(once)
		if vectorBits(twice) != vectorBits(input) {
			t.Fatalf("flipHandedness is not an involution for Z bits 0x%08x: %v",
				math.Float32bits(z), vectorBits(twice))
		}
		if math.Float32bits(once.Z) != math.Float32bits(z)^0x80000000 {
			t.Fatalf("flipHandedness did not flip only the Z sign bit for 0x%08x", math.Float32bits(z))
		}
		if once.X != input.X || once.Y != input.Y {
			t.Fatalf("flipHandedness altered X or Y: %v", once)
		}
	}
}

// TestAudioListenerConstructorDefaults pins the exact constructed state,
// including the negative-zero Z that the reference constructor produces because
// it stores Vector3.Zero unflipped while the getter flips.
func TestAudioListenerConstructorDefaults(t *testing.T) {
	listener := NewAudioListener()
	const negativeZero = uint32(0x80000000)

	if got := vectorBits(listener.Position()); got != [3]uint32{0, 0, negativeZero} {
		t.Fatalf("Position default bits = %v", got)
	}
	if got := vectorBits(listener.Velocity()); got != [3]uint32{0, 0, negativeZero} {
		t.Fatalf("Velocity default bits = %v", got)
	}
	if got := vectorBits(listener.Forward()); got != vectorBits(framework.Vector3Forward()) {
		t.Fatalf("Forward default bits = %v", got)
	}
	if got := vectorBits(listener.Up()); got != vectorBits(framework.Vector3Up()) {
		t.Fatalf("Up default bits = %v", got)
	}

	// The negative zero compares equal to Vector3.Zero, exactly as it does in
	// the reference: it is observable only through the sign bit.
	if listener.Position() != framework.Vector3Zero() {
		t.Fatal("Position default is not equal to Vector3.Zero")
	}
}

// TestAudioListenerRoundTripsEveryAssignedValue proves the double flip is
// invisible to callers for every bit pattern, including both zeros and NaN.
func TestAudioListenerRoundTripsEveryAssignedValue(t *testing.T) {
	inputs := []framework.Vector3{
		{X: 1, Y: 2, Z: 3},
		framework.Vector3Zero(),
		{X: 0, Y: 0, Z: float32(math.Copysign(0, -1))},
		{X: float32(math.Inf(-1)), Y: float32(math.Inf(1)), Z: float32(math.Inf(1))},
		{X: math.SmallestNonzeroFloat32, Y: -math.MaxFloat32, Z: math.MaxFloat32},
		{X: float32(math.NaN()), Y: float32(math.NaN()), Z: float32(math.NaN())},
	}
	for _, input := range inputs {
		listener := NewAudioListener()
		listener.SetPosition(input)
		listener.SetVelocity(input)
		listener.SetForward(input)
		listener.SetUp(input)
		for name, got := range map[string]framework.Vector3{
			"Position": listener.Position(),
			"Velocity": listener.Velocity(),
			"Forward":  listener.Forward(),
			"Up":       listener.Up(),
		} {
			if vectorBits(got) != vectorBits(input) {
				t.Fatalf("%s round-trip bits = %v, want %v", name, vectorBits(got), vectorBits(input))
			}
		}
	}
}

// TestAudioListenerKeepsReferenceSemantics pins the CLR reference behavior that
// the pure-managed classification must not weaken.
func TestAudioListenerKeepsReferenceSemantics(t *testing.T) {
	listener := NewAudioListener()
	alias := listener
	alias.SetPosition(framework.Vector3{X: 7, Y: 8, Z: 9})
	if listener.Position() != (framework.Vector3{X: 7, Y: 8, Z: 9}) {
		t.Fatalf("aliased mutation was not observed: %v", listener.Position())
	}
	if listener != alias {
		t.Fatal("two references to one instance are not identical")
	}

	other := NewAudioListener()
	if other.Position() == listener.Position() {
		t.Fatal("a separately constructed listener shares state")
	}
}

// TestAudioListenerSettersAcceptEveryValue records that no accessor validates:
// the reference stores whatever it is given, with no normalization and no
// orthogonality requirement between Forward and Up.
func TestAudioListenerSettersAcceptEveryValue(t *testing.T) {
	listener := NewAudioListener()
	degenerate := framework.Vector3Zero()
	listener.SetForward(degenerate)
	listener.SetUp(degenerate)
	if listener.Forward() != degenerate || listener.Up() != degenerate {
		t.Fatalf("degenerate orientation was altered: %v %v", listener.Forward(), listener.Up())
	}
}
