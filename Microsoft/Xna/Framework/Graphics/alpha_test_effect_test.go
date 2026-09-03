package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// AlphaTestEffect and DualTextureEffect are managed state on both sides of the
// boundary, so everything below is measured without a device. What needs one --
// the two or three properties that cross, and OnApply's push -- is in the
// native-stress scenario.

// TestAlphaTestEffectFieldInitialisersAreTheConstructorsOwn pins the seven
// values the constructor stores and the four it does not.
func TestAlphaTestEffectFieldInitialisersAreTheConstructorsOwn(t *testing.T) {
	effect := newAlphaTestEffectState()
	identity := framework.MatrixIdentity()
	if effect.World() != identity || effect.View() != identity || effect.Projection() != identity {
		t.Fatal("the three matrices are not Matrix.Identity")
	}
	if effect.DiffuseColor() != framework.Vector3One() || effect.Alpha() != 1 {
		t.Fatalf("DiffuseColor/Alpha = %v/%v", effect.DiffuseColor(), effect.Alpha())
	}
	if effect.FogStart() != 0 || effect.FogEnd() != 1 {
		t.Fatalf("fog range = (%v, %v), want (0, 1)", effect.FogStart(), effect.FogEnd())
	}
	// `ldc.i4.6` -- CompareFunction.Greater, and the one default this type has
	// that BasicEffect does not.
	if effect.AlphaFunction() != CompareFunctionGreater {
		t.Fatalf("AlphaFunction = %v, want Greater", effect.AlphaFunction())
	}
	// ReferenceAlpha, FogEnabled and VertexColorEnabled are never assigned.
	if effect.ReferenceAlpha() != 0 || effect.FogEnabled() || effect.VertexColorEnabled() {
		t.Fatal("a field the constructor assigns nowhere has a non-zero default")
	}
	if effect.dirtyFlags != effectDirtyAll {
		t.Fatalf("dirtyFlags = %d, want -1", effect.dirtyFlags)
	}
}

// TestUnlitEffectsRaiseTwoMatrixFlagsWhereBasicEffectRaisesThree is the
// measurement that decided against sharing an accessor body across the stock
// effects, made falsifiable.
//
//	set_World   BasicEffect 19 = WorldViewProj|World|Fog
//	            AlphaTest   17 = WorldViewProj|Fog
//	set_View    BasicEffect 21 = WorldViewProj|EyePosition|Fog
//	            AlphaTest   17 = WorldViewProj|Fog
func TestUnlitEffectsRaiseTwoMatrixFlagsWhereBasicEffectRaisesThree(t *testing.T) {
	unlit := effectDirtyWorldViewProj | effectDirtyFog
	for name, probe := range map[string]struct {
		mutate func()
		read   func() effectDirtyFlags
		want   effectDirtyFlags
	}{
		"AlphaTestEffect.SetWorld": func() struct {
			mutate func()
			read   func() effectDirtyFlags
			want   effectDirtyFlags
		} {
			e := newAlphaTestEffectState()
			e.dirtyFlags = 0
			return struct {
				mutate func()
				read   func() effectDirtyFlags
				want   effectDirtyFlags
			}{func() { e.SetWorld(framework.MatrixIdentity()) }, func() effectDirtyFlags { return e.dirtyFlags }, unlit}
		}(),
		"AlphaTestEffect.SetView": func() struct {
			mutate func()
			read   func() effectDirtyFlags
			want   effectDirtyFlags
		} {
			e := newAlphaTestEffectState()
			e.dirtyFlags = 0
			return struct {
				mutate func()
				read   func() effectDirtyFlags
				want   effectDirtyFlags
			}{func() { e.SetView(framework.MatrixIdentity()) }, func() effectDirtyFlags { return e.dirtyFlags }, unlit}
		}(),
		"DualTextureEffect.SetWorld": func() struct {
			mutate func()
			read   func() effectDirtyFlags
			want   effectDirtyFlags
		} {
			e := newDualTextureEffectState()
			e.dirtyFlags = 0
			return struct {
				mutate func()
				read   func() effectDirtyFlags
				want   effectDirtyFlags
			}{func() { e.SetWorld(framework.MatrixIdentity()) }, func() effectDirtyFlags { return e.dirtyFlags }, unlit}
		}(),
		"DualTextureEffect.SetView": func() struct {
			mutate func()
			read   func() effectDirtyFlags
			want   effectDirtyFlags
		} {
			e := newDualTextureEffectState()
			e.dirtyFlags = 0
			return struct {
				mutate func()
				read   func() effectDirtyFlags
				want   effectDirtyFlags
			}{func() { e.SetView(framework.MatrixIdentity()) }, func() effectDirtyFlags { return e.dirtyFlags }, unlit}
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			probe.mutate()
			if got := probe.read(); got != probe.want {
				t.Fatalf("%s raised %d, want %d", name, got, probe.want)
			}
		})
	}
	// The contrast, on the same run: BasicEffect's two setters raise a third
	// flag each, and this is the assertion that fails if the two families are
	// ever collapsed into one body.
	basic := newBasicEffectState()
	basic.dirtyFlags = 0
	basic.SetWorld(framework.MatrixIdentity())
	if basic.dirtyFlags == unlit {
		t.Fatal("BasicEffect.SetWorld raised the unlit pair; it must also raise World")
	}
	basic.dirtyFlags = 0
	basic.SetView(framework.MatrixIdentity())
	if basic.dirtyFlags == unlit {
		t.Fatal("BasicEffect.SetView raised the unlit pair; it must also raise EyePosition")
	}
}

// TestAlphaTestEffectAlphaTestPairRaisesItsOwnFlag pins the flag no other stock
// effect uses, and pins that neither setter validates.
func TestAlphaTestEffectAlphaTestPairRaisesItsOwnFlag(t *testing.T) {
	effect := newAlphaTestEffectState()
	effect.dirtyFlags = 0
	effect.SetAlphaFunction(CompareFunctionLessEqual)
	if effect.dirtyFlags != effectDirtyAlphaTest {
		t.Fatalf("set_AlphaFunction raised %d, want the AlphaTest flag", effect.dirtyFlags)
	}
	effect.dirtyFlags = 0
	effect.SetReferenceAlpha(128)
	if effect.dirtyFlags != effectDirtyAlphaTest {
		t.Fatalf("set_ReferenceAlpha raised %d", effect.dirtyFlags)
	}
	// Neither setter validates: an undeclared CompareFunction and an alpha
	// outside 0..255 are stored and reported back, because the reference's
	// bodies are a store and an `or` with no guard.
	effect.SetAlphaFunction(CompareFunction(99))
	effect.SetReferenceAlpha(-7)
	if effect.AlphaFunction() != CompareFunction(99) || effect.ReferenceAlpha() != -7 {
		t.Fatalf("a setter validated: %v / %v", effect.AlphaFunction(), effect.ReferenceAlpha())
	}
}

// TestAlphaTestEffectRemainingFlagsMatchTheMeasuredWords covers the setters
// this type shares with BasicEffect, whose words DO agree.
func TestAlphaTestEffectRemainingFlagsMatchTheMeasuredWords(t *testing.T) {
	for name, probe := range map[string]struct {
		mutate func(*AlphaTestEffect)
		want   effectDirtyFlags
	}{
		"Projection":         {func(e *AlphaTestEffect) { e.SetProjection(framework.MatrixIdentity()) }, effectDirtyWorldViewProj},
		"DiffuseColor":       {func(e *AlphaTestEffect) { e.SetDiffuseColor(framework.Vector3Zero()) }, effectDirtyMaterialColor},
		"Alpha":              {func(e *AlphaTestEffect) { e.SetAlpha(0.5) }, effectDirtyMaterialColor},
		"FogStart":           {func(e *AlphaTestEffect) { e.SetFogStart(3) }, effectDirtyFog},
		"FogEnd":             {func(e *AlphaTestEffect) { e.SetFogEnd(4) }, effectDirtyFog},
		"FogEnabled":         {func(e *AlphaTestEffect) { e.SetFogEnabled(true) }, effectDirtyFogEnable | effectDirtyShaderIndex},
		"VertexColorEnabled": {func(e *AlphaTestEffect) { e.SetVertexColorEnabled(true) }, effectDirtyShaderIndex},
	} {
		t.Run(name, func(t *testing.T) {
			effect := newAlphaTestEffectState()
			effect.dirtyFlags = 0
			probe.mutate(effect)
			if effect.dirtyFlags != probe.want {
				t.Fatalf("set_%s raised %d, want %d", name, effect.dirtyFlags, probe.want)
			}
		})
	}
	// The two boolean setters return early on the value they already hold.
	for name, mutate := range map[string]func(*AlphaTestEffect){
		"FogEnabled":         func(e *AlphaTestEffect) { e.SetFogEnabled(false) },
		"VertexColorEnabled": func(e *AlphaTestEffect) { e.SetVertexColorEnabled(false) },
	} {
		effect := newAlphaTestEffectState()
		effect.dirtyFlags = 0
		mutate(effect)
		if effect.dirtyFlags != 0 {
			t.Fatalf("set_%s raised %d for the value it already held", name, effect.dirtyFlags)
		}
	}
}

// TestDualTextureEffectHasTwoLayersAndNoAlphaTest pins what distinguishes it
// from AlphaTestEffect: two texture properties and no AlphaTest flag anywhere.
func TestDualTextureEffectHasTwoLayersAndNoAlphaTest(t *testing.T) {
	effect := newDualTextureEffectState()
	identity := framework.MatrixIdentity()
	if effect.World() != identity || effect.DiffuseColor() != framework.Vector3One() ||
		effect.Alpha() != 1 || effect.FogEnd() != 1 {
		t.Fatal("a field initialiser disagrees with the constructor")
	}
	// Both layers answer the managed field and start empty.
	first, err := effect.Texture()
	if err != nil || first != nil {
		t.Fatalf("Texture = %v, %v", first, err)
	}
	second, err := effect.Texture2()
	if err != nil || second != nil {
		t.Fatalf("Texture2 = %v, %v", second, err)
	}
	// Both setters need a native effect, so both refuse here -- and refusing is
	// what proves they reach CNA rather than only caching.
	if err := effect.SetTexture(nil); err == nil {
		t.Fatal("SetTexture answered with no native effect")
	}
	if err := effect.SetTexture2(nil); err == nil {
		t.Fatal("SetTexture2 answered with no native effect")
	}
	// No setter on this type raises the AlphaTest flag.
	for _, mutate := range []func(){
		func() { effect.SetWorld(identity) },
		func() { effect.SetView(identity) },
		func() { effect.SetProjection(identity) },
		func() { effect.SetDiffuseColor(framework.Vector3Zero()) },
		func() { effect.SetAlpha(0.5) },
		func() { effect.SetFogStart(1) },
		func() { effect.SetFogEnd(2) },
		func() { effect.SetFogEnabled(true) },
		func() { effect.SetVertexColorEnabled(true) },
	} {
		effect.dirtyFlags = 0
		mutate()
		if effect.dirtyFlags&effectDirtyAlphaTest != 0 {
			t.Fatal("a DualTextureEffect setter raised the AlphaTest flag")
		}
	}
}

// TestTheUnlitEffectsRefuseWithNoNativeHalf covers the members that really do
// reach CNA on both types.
func TestTheUnlitEffectsRefuseWithNoNativeHalf(t *testing.T) {
	alphaTest := newAlphaTestEffectState()
	if _, err := alphaTest.FogColor(); err == nil {
		t.Fatal("AlphaTestEffect.FogColor answered with no native effect")
	}
	for _, err := range []error{
		alphaTest.SetFogColor(framework.Vector3One()),
		alphaTest.SetTexture(nil),
		alphaTest.OnApply(),
	} {
		if err == nil {
			t.Fatal("an AlphaTestEffect native-backed member reported success")
		}
	}
	dual := newDualTextureEffectState()
	if _, err := dual.FogColor(); err == nil {
		t.Fatal("DualTextureEffect.FogColor answered with no native effect")
	}
	if err := dual.OnApply(); err == nil {
		t.Fatal("DualTextureEffect.OnApply reported success with no native effect")
	}
	// Texture answers the managed field on both, and a nil receiver refuses.
	if texture, err := alphaTest.Texture(); err != nil || texture != nil {
		t.Fatalf("AlphaTestEffect.Texture = %v, %v", texture, err)
	}
	var nilAlphaTest *AlphaTestEffect
	if _, err := nilAlphaTest.Texture(); err == nil {
		t.Fatal("Texture on a nil receiver reported success")
	}
	var nilDual *DualTextureEffect
	if _, err := nilDual.Texture2(); err == nil {
		t.Fatal("Texture2 on a nil receiver reported success")
	}
}

// TestTheUnlitEffectsSatisfyTheirTwoInterfaces is the compiler claim made
// executable. Neither implements IEffectLights, and asserting that is what
// keeps a future edit from adding lighting to an effect that has none.
func TestTheUnlitEffectsSatisfyTheirTwoInterfaces(t *testing.T) {
	var matrices IEffectMatrices = newAlphaTestEffectState()
	matrices.SetProjection(framework.MatrixIdentity())
	if matrices.Projection() != framework.MatrixIdentity() {
		t.Fatal("IEffectMatrices did not round-trip through AlphaTestEffect")
	}
	var fog IEffectFog = newDualTextureEffectState()
	fog.SetFogEnd(9)
	if fog.FogEnd() != 9 {
		t.Fatal("IEffectFog did not round-trip through DualTextureEffect")
	}
	if _, isLights := any(newAlphaTestEffectState()).(IEffectLights); isLights {
		t.Fatal("AlphaTestEffect satisfies IEffectLights; the pinned metadata says it implements two interfaces, not three")
	}
	if _, isLights := any(newDualTextureEffectState()).(IEffectLights); isLights {
		t.Fatal("DualTextureEffect satisfies IEffectLights")
	}
}

// TestEffectMaterialIsEffectWithADifferentName pins the whole of what the type
// adds, which is a class name.
func TestEffectMaterialIsEffectWithADifferentName(t *testing.T) {
	material := &EffectMaterial{effect: &Effect{resource: newGraphicsResource(nil, nil)}}
	material.effect.bindDerived(material)
	if got := material.clrTypeName(); got != "Microsoft.Xna.Framework.Graphics.EffectMaterial" {
		t.Fatalf("clrTypeName = %q", got)
	}
	// The identity site: a refusal from the base names the RUNTIME type.
	material.effect.resource.isDisposed = true
	err := material.SetCurrentTechnique(&EffectTechnique{})
	if err == nil || !containsSubstring(err.Error(), "Microsoft.Xna.Framework.Graphics.EffectMaterial") {
		t.Fatalf("SetCurrentTechnique named %v", err)
	}
	// A null clone source is refused with the base constructor's own refusal --
	// ArgumentNullException("cloneSource", FrameworkResources.NullNotAllowed) --
	// before anything native. Asserting only that it errored would pass for a
	// projection that dropped the guard and let the native call fail instead.
	_, nullErr := NewEffectMaterial(nil)
	if nullErr == nil {
		t.Fatal("NewEffectMaterial accepted a null source")
	}
	if !errors.Is(nullErr, errGraphicsResourceArgumentNull) ||
		!containsSubstring(nullErr.Error(), "cloneSource") ||
		!containsSubstring(nullErr.Error(), nullNotAllowed) {
		t.Fatalf("NewEffectMaterial(nil) = %v, want the base constructor's ArgumentNullException", nullErr)
	}
	// It satisfies the widened Effect positions, and asserts back.
	var reference EffectReference = material
	if _, ok := reference.(*EffectMaterial); !ok {
		t.Fatal("an EffectReference holding an EffectMaterial did not assert back")
	}
	if resolveEffect(reference) != material.effect {
		t.Fatal("resolveEffect did not reach the composed base")
	}
	// And it has NO OnApply: the reference declares none and the inherited
	// surface a derived type re-exposes is the PUBLIC surface, of which a
	// `protected internal` member is not part.
	if _, hasOnApply := any(material).(interface{ OnApply() error }); hasOnApply {
		t.Fatal("EffectMaterial exposes OnApply; the contract declares it none")
	}
}
