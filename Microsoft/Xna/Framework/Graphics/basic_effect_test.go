package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// BasicEffect's state half is managed in the reference and managed here, so
// every test below measures it without a device. What needs one -- OnApply's
// push, the four CNA-backed properties and the three published lights -- is in
// the native-stress scenario instead, and is named at the end of this file.

// newManagedBasicEffect is the block of field initialisers both constructors
// run before they chain to the base. It is the object with its native half
// missing, which is exactly what the managed claims are about.
func newManagedBasicEffect() *BasicEffect { return newBasicEffectState() }

// TestBasicEffectFieldInitialisersAreTheConstructorsOwn pins the nine values
// both constructors store before `call Effect::.ctor`, and the five they do
// NOT: lightingEnabled, preferPerPixelLighting, fogEnabled, textureEnabled and
// vertexColorEnabled are never assigned, so their zero value is the default.
func TestBasicEffectFieldInitialisersAreTheConstructorsOwn(t *testing.T) {
	effect := newManagedBasicEffect()
	identity := framework.MatrixIdentity()
	if effect.World() != identity || effect.View() != identity || effect.Projection() != identity {
		t.Fatal("a fresh BasicEffect's three matrices are not Matrix.Identity")
	}
	if effect.DiffuseColor() != framework.Vector3One() {
		t.Fatalf("DiffuseColor = %v, want Vector3.One", effect.DiffuseColor())
	}
	if effect.EmissiveColor() != framework.Vector3Zero() || effect.AmbientLightColor() != framework.Vector3Zero() {
		t.Fatal("EmissiveColor or AmbientLightColor is not Vector3.Zero")
	}
	if effect.Alpha() != 1 {
		t.Fatalf("Alpha = %v, want 1", effect.Alpha())
	}
	// fogEnd is 1 and fogStart is NOT assigned, so the pair is (0, 1) rather
	// than the (0, 0) a symmetric initialiser would give.
	if effect.FogStart() != 0 || effect.FogEnd() != 1 {
		t.Fatalf("fog range = (%v, %v), want (0, 1)", effect.FogStart(), effect.FogEnd())
	}
	for name, got := range map[string]bool{
		"LightingEnabled":        effect.LightingEnabled(),
		"PreferPerPixelLighting": effect.PreferPerPixelLighting(),
		"FogEnabled":             effect.FogEnabled(),
		"TextureEnabled":         effect.TextureEnabled(),
		"VertexColorEnabled":     effect.VertexColorEnabled(),
	} {
		if got {
			t.Fatalf("%s defaults to true; the constructor assigns it nowhere", name)
		}
	}
	// `ldc.i4.m1` -- every bit set, so the first OnApply writes everything.
	if effect.dirtyFlags != effectDirtyAll {
		t.Fatalf("dirtyFlags = %d, want -1", effect.dirtyFlags)
	}
}

// TestBasicEffectSettersRaiseTheMeasuredDirtyFlags is the falsifiable half of
// the dirty-flag claim: each setter's flag word is read from the `or` operand
// in its own IL, and a setter that raised the wrong set would still round-trip
// its value.
func TestBasicEffectSettersRaiseTheMeasuredDirtyFlags(t *testing.T) {
	for name, probe := range map[string]struct {
		mutate func(*BasicEffect)
		want   effectDirtyFlags
	}{
		"World":                  {func(e *BasicEffect) { e.SetWorld(framework.MatrixIdentity()) }, effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyFog},
		"View":                   {func(e *BasicEffect) { e.SetView(framework.MatrixIdentity()) }, effectDirtyWorldViewProj | effectDirtyEyePosition | effectDirtyFog},
		"Projection":             {func(e *BasicEffect) { e.SetProjection(framework.MatrixIdentity()) }, effectDirtyWorldViewProj},
		"DiffuseColor":           {func(e *BasicEffect) { e.SetDiffuseColor(framework.Vector3Zero()) }, effectDirtyMaterialColor},
		"EmissiveColor":          {func(e *BasicEffect) { e.SetEmissiveColor(framework.Vector3One()) }, effectDirtyMaterialColor},
		"AmbientLightColor":      {func(e *BasicEffect) { e.SetAmbientLightColor(framework.Vector3One()) }, effectDirtyMaterialColor},
		"Alpha":                  {func(e *BasicEffect) { e.SetAlpha(0.5) }, effectDirtyMaterialColor},
		"FogStart":               {func(e *BasicEffect) { e.SetFogStart(3) }, effectDirtyFog},
		"FogEnd":                 {func(e *BasicEffect) { e.SetFogEnd(4) }, effectDirtyFog},
		"FogEnabled":             {func(e *BasicEffect) { e.SetFogEnabled(true) }, effectDirtyFogEnable | effectDirtyShaderIndex},
		"LightingEnabled":        {func(e *BasicEffect) { e.SetLightingEnabled(true) }, effectDirtyMaterialColor | effectDirtyShaderIndex},
		"PreferPerPixelLighting": {func(e *BasicEffect) { e.SetPreferPerPixelLighting(true) }, effectDirtyShaderIndex},
		"VertexColorEnabled":     {func(e *BasicEffect) { e.SetVertexColorEnabled(true) }, effectDirtyShaderIndex},
		"TextureEnabled":         {func(e *BasicEffect) { e.SetTextureEnabled(true) }, effectDirtyShaderIndex},
	} {
		t.Run(name, func(t *testing.T) {
			effect := newManagedBasicEffect()
			effect.dirtyFlags = 0
			probe.mutate(effect)
			if effect.dirtyFlags != probe.want {
				t.Fatalf("set_%s raised %d, want %d", name, effect.dirtyFlags, probe.want)
			}
		})
	}
}

// TestBasicEffectBooleanSettersReturnEarlyOnTheSameValue pins the `beq` the
// five permutation setters open with, and pins that the ten OTHER setters do
// not have it.
func TestBasicEffectBooleanSettersReturnEarlyOnTheSameValue(t *testing.T) {
	for name, mutate := range map[string]func(*BasicEffect){
		"FogEnabled":             func(e *BasicEffect) { e.SetFogEnabled(false) },
		"LightingEnabled":        func(e *BasicEffect) { e.SetLightingEnabled(false) },
		"PreferPerPixelLighting": func(e *BasicEffect) { e.SetPreferPerPixelLighting(false) },
		"VertexColorEnabled":     func(e *BasicEffect) { e.SetVertexColorEnabled(false) },
		"TextureEnabled":         func(e *BasicEffect) { e.SetTextureEnabled(false) },
	} {
		t.Run(name, func(t *testing.T) {
			effect := newManagedBasicEffect()
			effect.dirtyFlags = 0
			mutate(effect)
			if effect.dirtyFlags != 0 {
				t.Fatalf("set_%s raised %d for the value it already held", name, effect.dirtyFlags)
			}
		})
	}
	// The contrast: a Vector3 setter has no early return, so assigning the same
	// value still raises its flag.
	effect := newManagedBasicEffect()
	effect.dirtyFlags = 0
	effect.SetDiffuseColor(framework.Vector3One())
	if effect.dirtyFlags != effectDirtyMaterialColor {
		t.Fatal("set_DiffuseColor returned early for the value it already held; only the five boolean setters do")
	}
}

// TestBasicEffectGettersAnswerTheStoredValue is the claim the thirteen unbound
// getter routes rest on: the reference reads its own field, not the effect, and
// what OnApply pushes is not always what the getter reports.
func TestBasicEffectGettersAnswerTheStoredValue(t *testing.T) {
	effect := newManagedBasicEffect()
	colour := framework.NewVector3BySingleAndSingleAndSingle(0.2, 0.4, 0.6)
	effect.SetDiffuseColor(colour)
	effect.SetAlpha(0.5)
	// OnApply pushes diffuseColor * alpha; get_DiffuseColor answers the stored
	// colour. A projection that read the pushed value back would report
	// (0.1, 0.2, 0.3) here.
	if got := effect.DiffuseColor(); got != colour {
		t.Fatalf("DiffuseColor = %v after SetAlpha(0.5), want the stored %v", got, colour)
	}
	world := framework.NewMatrix(
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)
	effect.SetWorld(world)
	effect.SetProjection(framework.MatrixIdentity())
	// The reference pushes world*view*projection into one parameter and keeps
	// the three matrices apart; get_World answers the world.
	if got := effect.World(); got != world {
		t.Fatalf("World = %v, want the stored matrix", got)
	}
}

// TestBasicEffectEnableDefaultLightingSetsLightingAndAmbient pins the two
// statements of the member's own body, over lights with no native half.
func TestBasicEffectEnableDefaultLightingSetsLightingAndAmbient(t *testing.T) {
	effect := newManagedBasicEffect()
	var err error
	if effect.light0, err = NewDirectionalLight(nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if effect.light1, err = NewDirectionalLight(nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if effect.light2, err = NewDirectionalLight(nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	effect.dirtyFlags = 0
	if err := effect.EnableDefaultLighting(); err != nil {
		t.Fatal(err)
	}
	if !effect.LightingEnabled() {
		t.Fatal("EnableDefaultLighting left LightingEnabled false")
	}
	want := framework.NewVector3BySingleAndSingleAndSingle(0.05333332, 0.09882354, 0.1819608)
	if got := effect.AmbientLightColor(); got != want {
		t.Fatalf("AmbientLightColor = %v, want the helper's returned %v", got, want)
	}
	// LightingEnabled raises MaterialColor|ShaderIndex and AmbientLightColor
	// raises MaterialColor, so the union is what the member leaves behind.
	if effect.dirtyFlags != effectDirtyMaterialColor|effectDirtyShaderIndex {
		t.Fatalf("EnableDefaultLighting left dirtyFlags = %d", effect.dirtyFlags)
	}
	if !effect.light0.Enabled() || !effect.light1.Enabled() || !effect.light2.Enabled() {
		t.Fatal("the rig left a light disabled")
	}
}

// TestBasicEffectOneLightIsRecomputedOnlyWhileLightingIsOn pins the bracket the
// IL puts around the `oneLight` recomputation -- `if (lightingEnabled)` -- and
// the ShaderIndex flag its TRANSITION raises.
func TestBasicEffectOneLightIsRecomputedOnlyWhileLightingIsOn(t *testing.T) {
	effect := newManagedBasicEffect()
	var err error
	if effect.light1, err = NewDirectionalLight(nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if effect.light2, err = NewDirectionalLight(nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	// Lighting off: the recomputation is skipped entirely, so oneLight stays
	// false even though neither of the two lights is enabled.
	effect.dirtyFlags = 0
	effect.recomputeOneLight()
	if effect.oneLight {
		t.Fatal("oneLight was recomputed while lighting was off")
	}
	if effect.dirtyFlags != 0 {
		t.Fatalf("the skipped recomputation still raised %d", effect.dirtyFlags)
	}
	// Lighting on, both extra lights off: oneLight becomes true and the
	// transition raises ShaderIndex.
	effect.lightingEnabled = true
	effect.dirtyFlags = 0
	effect.recomputeOneLight()
	if !effect.oneLight || effect.dirtyFlags != effectDirtyShaderIndex {
		t.Fatalf("oneLight = %v, dirtyFlags = %d", effect.oneLight, effect.dirtyFlags)
	}
	// A second pass with nothing changed must raise nothing: the flag follows
	// the TRANSITION, not the value.
	effect.dirtyFlags = 0
	effect.recomputeOneLight()
	if effect.dirtyFlags != 0 {
		t.Fatalf("an unchanged oneLight raised %d", effect.dirtyFlags)
	}
	// Enabling light 1 flips it back and raises the flag again.
	if err := effect.light1.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	effect.recomputeOneLight()
	if effect.oneLight || effect.dirtyFlags != effectDirtyShaderIndex {
		t.Fatalf("oneLight = %v, dirtyFlags = %d after enabling light 1", effect.oneLight, effect.dirtyFlags)
	}
}

// TestBasicEffectRefusesAnEffectWithNoNativeHalf covers the four properties
// that really do reach CNA, plus the two virtual members.
func TestBasicEffectRefusesAnEffectWithNoNativeHalf(t *testing.T) {
	effect := newManagedBasicEffect()
	if _, err := effect.SpecularColor(); err == nil {
		t.Fatal("SpecularColor answered without a native effect")
	}
	if _, err := effect.SpecularPower(); err == nil {
		t.Fatal("SpecularPower answered without a native effect")
	}
	if _, err := effect.FogColor(); err == nil {
		t.Fatal("FogColor answered without a native effect")
	}
	for _, err := range []error{
		effect.SetSpecularColor(framework.Vector3One()),
		effect.SetSpecularPower(16),
		effect.SetFogColor(framework.Vector3One()),
		effect.SetTexture(nil),
		effect.OnApply(),
	} {
		if err == nil {
			t.Fatal("a native-backed member reported success with no native effect")
		}
	}
	// Texture is the exception among the four: its value is a managed field, so
	// it answers rather than refusing even though the contract declares it
	// fallible.
	texture, err := effect.Texture()
	if err != nil || texture != nil {
		t.Fatalf("Texture = %v, %v; the getter answers the managed field", texture, err)
	}
	var nilEffect *BasicEffect
	if _, err := nilEffect.Texture(); err == nil {
		t.Fatal("Texture on a nil receiver reported success")
	}
}

// TestBasicEffectSatisfiesTheThreeDeclaredInterfaces is the compiler claim made
// executable: the pinned metadata says BasicEffect implements all three, and a
// consumer reaches it through each.
func TestBasicEffectSatisfiesTheThreeDeclaredInterfaces(t *testing.T) {
	effect := newManagedBasicEffect()
	var matrices IEffectMatrices = effect
	matrices.SetWorld(framework.MatrixIdentity())
	if matrices.World() != framework.MatrixIdentity() {
		t.Fatal("IEffectMatrices did not round-trip World")
	}
	var fog IEffectFog = effect
	fog.SetFogStart(2)
	if fog.FogStart() != 2 {
		t.Fatal("IEffectFog did not round-trip FogStart")
	}
	var lights IEffectLights = effect
	lights.SetLightingEnabled(true)
	if !lights.LightingEnabled() {
		t.Fatal("IEffectLights did not round-trip LightingEnabled")
	}
	// The interface's one fallible operation is fallible through the interface
	// too, which is what makes the split observable from a consumer's side.
	if err := lights.EnableDefaultLighting(); err == nil {
		t.Fatal("EnableDefaultLighting reported success over lights that do not exist")
	}
}

// TestEffectReferenceCarriesTheDerivedIdentity is the whole reason Effect
// widens at RETURNS: a value that came back as an EffectReference must assert
// to the derived type, and a value that is really an Effect must not.
func TestEffectReferenceCarriesTheDerivedIdentity(t *testing.T) {
	effect := newManagedBasicEffect()
	effect.effect = &Effect{}
	effect.effect.bindDerived(effect)
	effect.effect.bindDerivedEffect(effect)

	var reference EffectReference = effect
	if _, ok := reference.(*BasicEffect); !ok {
		t.Fatal("an EffectReference holding a BasicEffect did not assert back to one")
	}
	// The base half satisfies the interface too, and asserting IT to the
	// derived type must fail -- otherwise the interface would be carrying a
	// claim it cannot support.
	var base EffectReference = effect.effect
	if _, ok := base.(*BasicEffect); ok {
		t.Fatal("the composed base half asserted to the derived type")
	}
	// resolveEffect is the `ldarg` a CLR call site does for free, and it must
	// answer the same base half either way.
	if resolveEffect(reference) != effect.effect || resolveEffect(base) != effect.effect {
		t.Fatal("resolveEffect did not reach the one composed base")
	}
	if resolveEffect(nil) != nil {
		t.Fatal("resolveEffect(nil) is not nil")
	}
	var typedNil *BasicEffect
	if resolveEffect(typedNil) != nil {
		t.Fatal("an interface holding a typed nil did not resolve to nil")
	}
}

// TestComposedEffectReportsTheDerivedTypeName is the identity site made
// observable: Effect's refusals name the RUNTIME type, so a BasicEffect must
// not be reported as an Effect.
func TestComposedEffectReportsTheDerivedTypeName(t *testing.T) {
	effect := newManagedBasicEffect()
	effect.effect = &Effect{resource: newGraphicsResource(nil, nil)}
	effect.effect.bindDerived(effect)
	effect.effect.bindDerivedEffect(effect)
	// The disposed branch is the one both identity sites are on.
	effect.effect.resource.isDisposed = true
	err := effect.effect.SetCurrentTechnique(&EffectTechnique{})
	if err == nil {
		t.Fatal("SetCurrentTechnique on a disposed effect reported success")
	}
	if got := err.Error(); !containsSubstring(got, "Microsoft.Xna.Framework.Graphics.BasicEffect") {
		t.Fatalf("SetCurrentTechnique named %q; Helpers::CheckDisposed reports the runtime type", got)
	}
	if _, err := effect.effect.cloneBase(); err == nil ||
		!containsSubstring(err.Error(), "Microsoft.Xna.Framework.Graphics.BasicEffect") {
		t.Fatalf("cloneBase named %v", err)
	}
	// An Effect that nothing composes still names itself. newEffect binds the
	// Effect as its own CLR `this`, and a derived constructor replaces that
	// binding afterwards -- so an uncomposed one answers Effect.
	plain := &Effect{resource: newGraphicsResource(nil, nil)}
	plain.bindDerived(plain)
	plain.resource.isDisposed = true
	if _, err := plain.cloneBase(); err == nil ||
		!containsSubstring(err.Error(), "Microsoft.Xna.Framework.Graphics.Effect") {
		t.Fatalf("an uncomposed Effect named %v", err)
	}
}

// TestEffectVirtualsDispatchToTheDerivedBody is the composition counterpart of
// `callvirt`: EffectPass::Apply calls OnApply through the base, and on a
// BasicEffect that must reach BasicEffect's body rather than the base's empty
// one.
func TestEffectVirtualsDispatchToTheDerivedBody(t *testing.T) {
	// The base with nothing composed runs the four-byte `ret`.
	plain := &Effect{}
	if err := plain.OnApply(); err != nil {
		t.Fatalf("the base's OnApply is not the empty body: %v", err)
	}
	effect := newManagedBasicEffect()
	effect.effect = &Effect{}
	effect.effect.bindDerived(effect)
	effect.effect.bindDerivedEffect(effect)
	// BasicEffect's OnApply needs a native effect, so reaching it is observable
	// as a refusal where the base's body would have answered nil.
	if err := effect.effect.OnApply(); err == nil {
		t.Fatal("OnApply through the base reached the base's empty body, not the derived override")
	}
	// Clone has no such installer and needs none: it is a member of
	// EffectReference, so a consumer holding a BasicEffect reaches
	// BasicEffect::Clone through Go's own method set. The claim is that the
	// interface value dispatches to the derived body, and it is checked here
	// rather than through the base -- which no consumer can obtain.
	var reference EffectReference = effect
	if _, err := reference.Clone(); err == nil {
		t.Fatal("Clone through the interface reached the base's body, not the derived one")
	}
	if _, err := effect.effect.Clone(); err == nil {
		t.Fatal("the base's own Clone answered on an effect with no native half")
	}
}

func containsSubstring(haystack, needle string) bool {
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if haystack[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
