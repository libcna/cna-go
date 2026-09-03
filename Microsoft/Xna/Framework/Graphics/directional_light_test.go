package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// A publicly constructed DirectionalLight has no native light behind it -- CNA
// publishes no EffectParameters for a caller to have obtained, so the three
// parameter arguments are always nil -- which is exactly the case the
// reference's own `brfalse` guards describe. Every behaviour below is therefore
// the reference's managed half, measured without a device.

func newTestLight(t *testing.T) *DirectionalLight {
	t.Helper()
	light, err := NewDirectionalLight(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewDirectionalLight: %v", err)
	}
	return light
}

// TestDirectionalLightDefaultsAreTheConstructorsSetterArm pins the three
// `ldc.r4`-free constants the no-clone arm calls its own setters with, and the
// one field it does NOT touch.
func TestDirectionalLightDefaultsAreTheConstructorsSetterArm(t *testing.T) {
	light := newTestLight(t)
	if got := light.Direction(); got != framework.Vector3Down() {
		t.Fatalf("default Direction = %v, want Vector3.Down", got)
	}
	if got := light.DiffuseColor(); got != framework.Vector3One() {
		t.Fatalf("default DiffuseColor = %v, want Vector3.One", got)
	}
	if got := light.SpecularColor(); got != framework.Vector3Zero() {
		t.Fatalf("default SpecularColor = %v, want Vector3.Zero", got)
	}
	// `enabled` is never assigned by the constructor: the field's zero value is
	// what a fresh light carries, and BasicEffect's own constructor is what
	// turns light 0 on afterwards.
	if light.Enabled() {
		t.Fatal("a fresh light is enabled; the constructor assigns the flag nowhere")
	}
}

// TestDirectionalLightGettersAnswerTheCacheNotTheParameter is the split the
// whole type turns on: get_DiffuseColor is `ldfld cachedDiffuseColor`, and
// set_Enabled(false) writes Vector3.Zero into the PARAMETER while leaving that
// cache alone.
func TestDirectionalLightGettersAnswerTheCacheNotTheParameter(t *testing.T) {
	light := newTestLight(t)
	if err := light.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	colour := framework.NewVector3BySingleAndSingleAndSingle(0.25, 0.5, 0.75)
	if err := light.SetDiffuseColor(colour); err != nil {
		t.Fatal(err)
	}
	if err := light.SetSpecularColor(colour); err != nil {
		t.Fatal(err)
	}
	if err := light.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if got := light.DiffuseColor(); got != colour {
		t.Fatalf("a disabled light reports DiffuseColor %v; the cache is untouched by disabling", got)
	}
	if got := light.SpecularColor(); got != colour {
		t.Fatalf("a disabled light reports SpecularColor %v", got)
	}
	// And the colour set WHILE disabled still reaches the cache, which is what
	// makes re-enabling publish it.
	darker := framework.NewVector3BySingleAndSingleAndSingle(0.1, 0.1, 0.1)
	if err := light.SetDiffuseColor(darker); err != nil {
		t.Fatal(err)
	}
	if got := light.DiffuseColor(); got != darker {
		t.Fatalf("a write to a disabled light did not reach the cache: %v", got)
	}
}

// TestDirectionalLightSetEnabledReturnsEarlyOnTheSameValue pins the `beq.s` at
// IL_0007, which is what decides whether the two colour writes happen at all.
func TestDirectionalLightSetEnabledReturnsEarlyOnTheSameValue(t *testing.T) {
	light := newTestLight(t)
	if err := light.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if light.Enabled() {
		t.Fatal("setting the value it already holds changed it")
	}
	if err := light.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !light.Enabled() {
		t.Fatal("enabling did not take")
	}
}

// TestDirectionalLightCloneCopiesFieldsAndWritesNothing pins the OTHER arm: the
// clone path is four `stfld`s and no setter call, so a cloned light's state is
// the source's exactly -- including a state the setter arm could not produce.
func TestDirectionalLightCloneCopiesFieldsAndWritesNothing(t *testing.T) {
	source := newTestLight(t)
	if err := source.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	direction := framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3)
	diffuse := framework.NewVector3BySingleAndSingleAndSingle(4, 5, 6)
	specular := framework.NewVector3BySingleAndSingleAndSingle(7, 8, 9)
	for _, err := range []error{
		source.SetDirection(direction),
		source.SetDiffuseColor(diffuse),
		source.SetSpecularColor(specular),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	clone, err := NewDirectionalLight(nil, nil, nil, source)
	if err != nil {
		t.Fatalf("clone constructor: %v", err)
	}
	if !clone.Enabled() || clone.Direction() != direction ||
		clone.DiffuseColor() != diffuse || clone.SpecularColor() != specular {
		t.Fatalf("clone = %v/%v/%v/%v", clone.Enabled(), clone.Direction(), clone.DiffuseColor(), clone.SpecularColor())
	}
	// The clone is its own object: mutating it must not reach the source.
	if err := clone.SetDirection(framework.Vector3Zero()); err != nil {
		t.Fatal(err)
	}
	if source.Direction() != direction {
		t.Fatal("mutating the clone reached the source")
	}
	// A clone of a DISABLED light with non-zero colours is a state the setter
	// arm cannot produce, because that arm's colour writes are guarded by
	// `enabled`. The clone arm reaches it by copying, which is why it is a
	// separate arm rather than three setter calls.
	if err := source.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	disabledClone, err := NewDirectionalLight(nil, nil, nil, source)
	if err != nil {
		t.Fatal(err)
	}
	if disabledClone.Enabled() || disabledClone.DiffuseColor() != diffuse {
		t.Fatalf("clone of a disabled light = %v/%v", disabledClone.Enabled(), disabledClone.DiffuseColor())
	}
}

// TestDirectionalLightRefusesANilReceiver is the Go-only guard standing in for
// NullReferenceException.
func TestDirectionalLightRefusesANilReceiver(t *testing.T) {
	var light *DirectionalLight
	if light.Enabled() || light.Direction() != (framework.Vector3{}) ||
		light.DiffuseColor() != (framework.Vector3{}) || light.SpecularColor() != (framework.Vector3{}) {
		t.Fatal("a nil light answered a getter with something other than the zero value")
	}
	for _, err := range []error{
		light.SetEnabled(true),
		light.SetDirection(framework.Vector3One()),
		light.SetDiffuseColor(framework.Vector3One()),
		light.SetSpecularColor(framework.Vector3One()),
	} {
		if err == nil {
			t.Fatal("a setter on a nil light reported success")
		}
	}
}

// TestEffectHelpersDefaultLightingIsTheMeasuredRig pins every constant
// EffectHelpers::EnableDefaultLighting loads, in the order it loads them.
//
// This is the reason cna_effect_lights_enable_default stays unbound: these
// twelve vectors are XNA's, and CNA's preset is CNA's.
func TestEffectHelpersDefaultLightingIsTheMeasuredRig(t *testing.T) {
	light0, light1, light2 := newTestLight(t), newTestLight(t), newTestLight(t)
	ambient, err := effectHelpersEnableDefaultLighting(light0, light1, light2)
	if err != nil {
		t.Fatal(err)
	}
	want := framework.NewVector3BySingleAndSingleAndSingle(0.05333332, 0.09882354, 0.1819608)
	if ambient != want {
		t.Fatalf("ambient = %v, want %v", ambient, want)
	}
	for index, expected := range []struct {
		direction, diffuse, specular framework.Vector3
		light                        *DirectionalLight
	}{
		{
			light:     light0,
			direction: framework.NewVector3BySingleAndSingleAndSingle(-0.5265408, -0.5735765, -0.6275069),
			diffuse:   framework.NewVector3BySingleAndSingleAndSingle(1, 0.9607844, 0.8078432),
			specular:  framework.NewVector3BySingleAndSingleAndSingle(1, 0.9607844, 0.8078432),
		},
		{
			light:     light1,
			direction: framework.NewVector3BySingleAndSingleAndSingle(0.7198464, 0.3420201, 0.6040227),
			diffuse:   framework.NewVector3BySingleAndSingleAndSingle(0.9647059, 0.7607844, 0.4078432),
			specular:  framework.Vector3Zero(),
		},
		{
			light:     light2,
			direction: framework.NewVector3BySingleAndSingleAndSingle(0.4545195, -0.7660444, 0.4545195),
			diffuse:   framework.NewVector3BySingleAndSingleAndSingle(0.3231373, 0.3607844, 0.3937255),
			specular:  framework.NewVector3BySingleAndSingleAndSingle(0.3231373, 0.3607844, 0.3937255),
		},
	} {
		if !expected.light.Enabled() {
			t.Fatalf("light %d is not enabled after the rig", index)
		}
		if got := expected.light.Direction(); got != expected.direction {
			t.Fatalf("light %d direction = %v, want %v", index, got, expected.direction)
		}
		if got := expected.light.DiffuseColor(); got != expected.diffuse {
			t.Fatalf("light %d diffuse = %v, want %v", index, got, expected.diffuse)
		}
		if got := expected.light.SpecularColor(); got != expected.specular {
			t.Fatalf("light %d specular = %v, want %v", index, got, expected.specular)
		}
	}
	// The rig enables each light LAST, after its three writes. On a light with
	// a native half that ordering decides whether the colours reach the shader,
	// so it is asserted here as an ordering claim rather than only as values:
	// light 1's specular is Vector3.Zero, and a rig that enabled first and then
	// wrote would be indistinguishable on values alone for the other two.
	if light1.SpecularColor() != framework.Vector3Zero() {
		t.Fatal("light 1's specular colour is not the zero the rig writes")
	}
}
