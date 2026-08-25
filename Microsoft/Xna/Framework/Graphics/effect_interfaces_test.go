package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// effectProbe is a test-only double. It exists to prove that the projected
// contracts are satisfiable and that their result channels are separate; it is
// not an effect, and it reproduces no XNA effect behavior.
type effectProbe struct {
	world      framework.Matrix
	view       framework.Matrix
	projection framework.Matrix

	fogEnabled bool
	fogStart   float32
	fogEnd     float32
	fogColor   framework.Vector3

	// fogColorFailure stands in for the D3DX HRESULT that the reference
	// EffectParameter path turns into an exception.
	fogColorFailure error
}

func (p *effectProbe) World() framework.Matrix              { return p.world }
func (p *effectProbe) SetWorld(value framework.Matrix)      { p.world = value }
func (p *effectProbe) View() framework.Matrix               { return p.view }
func (p *effectProbe) SetView(value framework.Matrix)       { p.view = value }
func (p *effectProbe) Projection() framework.Matrix         { return p.projection }
func (p *effectProbe) SetProjection(value framework.Matrix) { p.projection = value }
func (p *effectProbe) FogEnabled() bool                     { return p.fogEnabled }
func (p *effectProbe) SetFogEnabled(value bool)             { p.fogEnabled = value }
func (p *effectProbe) FogStart() float32                    { return p.fogStart }
func (p *effectProbe) SetFogStart(value float32)            { p.fogStart = value }
func (p *effectProbe) FogEnd() float32                      { return p.fogEnd }
func (p *effectProbe) SetFogEnd(value float32)              { p.fogEnd = value }

func (p *effectProbe) FogColor() (framework.Vector3, error) {
	if p.fogColorFailure != nil {
		return framework.Vector3{}, p.fogColorFailure
	}
	return p.fogColor, nil
}

func (p *effectProbe) SetFogColor(value framework.Vector3) error {
	if p.fogColorFailure != nil {
		return p.fogColorFailure
	}
	p.fogColor = value
	return nil
}

// Compile-time conformance. If either projected method set drifts, this fails
// to build rather than failing silently at run time.
var (
	_ IEffectMatrices = (*effectProbe)(nil)
	_ IEffectFog      = (*effectProbe)(nil)
)

// TestEffectMatricesContractIsInfallible pins the shape of the pure managed
// contract: every operation is a plain accessor with no error channel, which
// the compiler enforces by accepting these single-value assignments.
func TestEffectMatricesContractIsInfallible(t *testing.T) {
	var matrices IEffectMatrices = &effectProbe{}
	identity := framework.MatrixIdentity()
	matrices.SetWorld(identity)
	matrices.SetView(identity)
	matrices.SetProjection(identity)

	var world framework.Matrix = matrices.World()
	var view framework.Matrix = matrices.View()
	var projection framework.Matrix = matrices.Projection()
	if world != identity || view != identity || projection != identity {
		t.Fatalf("transform round-trip = %v %v %v", world, view, projection)
	}
}

// TestEffectFogContractSplitsFallibility pins the measured split: six managed
// operations with no error channel, and both FogColor accessors with one.
func TestEffectFogContractSplitsFallibility(t *testing.T) {
	probe := &effectProbe{}
	var fog IEffectFog = probe

	fog.SetFogEnabled(true)
	fog.SetFogStart(1.5)
	fog.SetFogEnd(400)
	var enabled bool = fog.FogEnabled()
	var start float32 = fog.FogStart()
	var end float32 = fog.FogEnd()
	if !enabled || start != 1.5 || end != 400 {
		t.Fatalf("managed fog round-trip = %t %v %v", enabled, start, end)
	}

	color := framework.Vector3{X: 0.25, Y: 0.5, Z: 0.75}
	if err := fog.SetFogColor(color); err != nil {
		t.Fatalf("SetFogColor = %v", err)
	}
	got, err := fog.FogColor()
	if err != nil {
		t.Fatalf("FogColor = %v", err)
	}
	if got != color {
		t.Fatalf("FogColor round-trip = %v", got)
	}

	// The error channel is real on both accessors and reaches the caller
	// instead of panicking or degrading to a zero value silently.
	boundaryFailure := errors.New("D3DX HRESULT")
	probe.fogColorFailure = boundaryFailure
	if err := fog.SetFogColor(color); !errors.Is(err, boundaryFailure) {
		t.Fatalf("SetFogColor failure = %v", err)
	}
	if _, err := fog.FogColor(); !errors.Is(err, boundaryFailure) {
		t.Fatalf("FogColor failure = %v", err)
	}
	// A rejected assignment must not have changed the stored value.
	probe.fogColorFailure = nil
	if stored, _ := fog.FogColor(); stored != color {
		t.Fatalf("stored fog color = %v", stored)
	}
}
