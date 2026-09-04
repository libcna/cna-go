package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// EnvironmentMapEffect and SkinnedEffect close the stock-effect family. What is
// new in them -- and what these tests hold -- is the explicit LightingEnabled
// pair, the two flag-storing setters, the validating setter and the two
// bone-transform guards.

// TestTheLitStockEffectsAlwaysReportLightingEnabled pins the two-byte getter:
// `ldc.i4.1; ret`, whatever the object's state.
func TestTheLitStockEffectsAlwaysReportLightingEnabled(t *testing.T) {
	environment := newEnvironmentMapEffectState()
	skinned := newSkinnedEffectState()
	if !environment.LightingEnabled() || !skinned.LightingEnabled() {
		t.Fatal("a lit stock effect reported LightingEnabled false")
	}
	// Setting true is the reference's no-op; setting false is its
	// NotSupportedException. Neither stores ANYTHING, which is a stronger claim
	// than "the getter still answers true" -- the getter is a constant, so it
	// would answer true whatever the setter did to the object.
	environment.dirtyFlags = 0
	skinned.dirtyFlags = 0
	// The FALSE call comes first and the TRUE one last on purpose. A setter
	// that stored its argument would end holding `true`, and a check run after
	// the false call would read the zero value and see nothing wrong.
	environment.SetLightingEnabled(false)
	environment.SetLightingEnabled(true)
	skinned.SetLightingEnabled(false)
	skinned.SetLightingEnabled(true)
	if !environment.LightingEnabled() || !skinned.LightingEnabled() {
		t.Fatal("a setter changed a value the getter returns as a constant")
	}
	if environment.dirtyFlags != 0 || skinned.dirtyFlags != 0 {
		t.Fatal("an explicit LightingEnabled setter raised a dirty flag")
	}
	if environment.oneLight || skinned.oneLight ||
		environment.fresnelEnabled || environment.specularEnabled {
		t.Fatal("an explicit LightingEnabled setter stored state; the reference's stores none")
	}
	// The pair reaches these types only through the interface, which is what an
	// explicit implementation means -- and is why they are witnesses.
	var asLights IEffectLights = environment
	if !asLights.LightingEnabled() {
		t.Fatal("IEffectLights did not carry the constant getter")
	}
}

// TestEnvironmentMapEffectFlagStoringSettersNeedANativeHalf covers the two
// 59-byte setters. Each writes a parameter AND stores a managed flag, so
// neither can run without the effect -- and the flag it would store is what a
// shader permutation turns on.
func TestEnvironmentMapEffectFlagStoringSettersNeedANativeHalf(t *testing.T) {
	effect := newEnvironmentMapEffectState()
	if err := effect.SetEnvironmentMapSpecular(framework.Vector3One()); err == nil {
		t.Fatal("SetEnvironmentMapSpecular answered with no native effect")
	}
	if err := effect.SetFresnelFactor(1); err == nil {
		t.Fatal("SetFresnelFactor answered with no native effect")
	}
	// The two above refuse at the top, before either the write or the store, so
	// they say nothing about the ORDER of the two. This one does: an effect
	// whose native half is GONE reaches the write and fails there, and the
	// reference's body writes the parameter FIRST -- so a failed write must
	// leave the flag alone.
	released := newEnvironmentMapEffectState()
	released.effect = &Effect{resource: newGraphicsResource(nil, &interop.Resource{})}
	// The constructor seeds every bit, so the flag word is cleared first --
	// otherwise ShaderIndex is already set and the assertion below proves
	// nothing.
	released.dirtyFlags = 0
	if err := released.SetEnvironmentMapSpecular(framework.Vector3One()); err == nil {
		t.Fatal("SetEnvironmentMapSpecular answered over a released native effect")
	}
	if err := released.SetFresnelFactor(1); err == nil {
		t.Fatal("SetFresnelFactor answered over a released native effect")
	}
	if released.specularEnabled || released.fresnelEnabled {
		t.Fatal("a setter whose parameter write failed still stored its flag; the write comes first")
	}
	if released.dirtyFlags&effectDirtyShaderIndex != 0 {
		t.Fatal("a setter whose parameter write failed still raised ShaderIndex")
	}
	// The refusal is BEFORE the flag store, so neither flag moved.
	if effect.specularEnabled || effect.fresnelEnabled {
		t.Fatal("a refused setter still stored its flag")
	}
	for _, refuse := range []func() error{
		func() error { _, err := effect.EnvironmentMapAmount(); return err },
		func() error { _, err := effect.EnvironmentMapSpecular(); return err },
		func() error { _, err := effect.FresnelFactor(); return err },
		func() error { return effect.SetEnvironmentMapAmount(1) },
		func() error { return effect.SetEnvironmentMap(nil) },
		func() error { return effect.SetTexture(nil) },
		func() error { return effect.OnApply() },
	} {
		if err := refuse(); err == nil {
			t.Fatal("a native-backed member reported success with no native effect")
		}
	}
}

// TestEnvironmentMapEffectDefaultsAndFlags pins the field initialisers and the
// LIT matrix words -- this effect raises three flags where the unlit two raise
// two.
func TestEnvironmentMapEffectDefaultsAndFlags(t *testing.T) {
	effect := newEnvironmentMapEffectState()
	identity := framework.MatrixIdentity()
	if effect.World() != identity || effect.DiffuseColor() != framework.Vector3One() ||
		effect.EmissiveColor() != framework.Vector3Zero() ||
		effect.AmbientLightColor() != framework.Vector3Zero() ||
		effect.Alpha() != 1 || effect.FogEnd() != 1 {
		t.Fatal("a field initialiser disagrees with the constructor")
	}
	effect.dirtyFlags = 0
	effect.SetWorld(identity)
	if effect.dirtyFlags != effectDirtyWorld|effectDirtyWorldViewProj|effectDirtyFog {
		t.Fatalf("set_World raised %d; a LIT effect raises three flags", effect.dirtyFlags)
	}
	effect.dirtyFlags = 0
	effect.SetView(identity)
	if effect.dirtyFlags != effectDirtyWorldViewProj|effectDirtyEyePosition|effectDirtyFog {
		t.Fatalf("set_View raised %d", effect.dirtyFlags)
	}
}

// TestSkinnedEffectWeightsPerVertexIsTheOneValidatingSetter pins the 54-byte
// body, its message, and the absence of an early return.
func TestSkinnedEffectWeightsPerVertexIsTheOneValidatingSetter(t *testing.T) {
	effect := newSkinnedEffectState()
	// `ldc.i4.4` -- the constructor's own default.
	if effect.WeightsPerVertex() != 4 {
		t.Fatalf("WeightsPerVertex = %d, want 4", effect.WeightsPerVertex())
	}
	for _, legal := range []int32{1, 2, 4} {
		effect.dirtyFlags = 0
		if err := effect.SetWeightsPerVertex(legal); err != nil {
			t.Fatalf("SetWeightsPerVertex(%d): %v", legal, err)
		}
		if effect.WeightsPerVertex() != legal || effect.dirtyFlags != effectDirtyShaderIndex {
			t.Fatalf("SetWeightsPerVertex(%d) left %d/%d", legal, effect.WeightsPerVertex(), effect.dirtyFlags)
		}
	}
	// No early return: assigning the value it already holds still raises the
	// flag, unlike every other boolean permutation setter in the family.
	effect.dirtyFlags = 0
	if err := effect.SetWeightsPerVertex(4); err != nil {
		t.Fatal(err)
	}
	if effect.dirtyFlags != effectDirtyShaderIndex {
		t.Fatal("SetWeightsPerVertex returned early for the value it already held")
	}
	for _, illegal := range []int32{0, 3, -1, 5, 8} {
		err := effect.SetWeightsPerVertex(illegal)
		if err == nil {
			t.Fatalf("SetWeightsPerVertex(%d) was accepted", illegal)
		}
		if !errors.Is(err, errArgumentOutOfRange) ||
			!containsSubstring(err.Error(), skinnedEffectWeightsPerVertex) {
			t.Fatalf("SetWeightsPerVertex(%d) = %v, want the reference's own refusal", illegal, err)
		}
	}
	// A refused value leaves the stored one alone.
	if effect.WeightsPerVertex() != 4 {
		t.Fatalf("a refused SetWeightsPerVertex changed the stored value to %d", effect.WeightsPerVertex())
	}
}

// TestSkinnedEffectBoneGuardsAreNotSymmetric pins the four guards, which use
// three different exception types between them.
func TestSkinnedEffectBoneGuardsAreNotSymmetric(t *testing.T) {
	effect := newSkinnedEffectState()
	// SetBoneTransforms: null OR EMPTY is ArgumentNullException, which is the
	// reference's own choice and not a tidy one.
	for name, transforms := range map[string][]framework.Matrix{
		"nil":   nil,
		"empty": {},
	} {
		err := effect.SetBoneTransforms(transforms)
		if err == nil || !errors.Is(err, errGraphicsResourceArgumentNull) ||
			!containsSubstring(err.Error(), "boneTransforms") {
			t.Fatalf("SetBoneTransforms(%s) = %v, want ArgumentNullException", name, err)
		}
	}
	// Over the limit is ArgumentException carrying the formatted message.
	tooMany := make([]framework.Matrix, SkinnedEffectMaxBones+1)
	err := effect.SetBoneTransforms(tooMany)
	if err == nil || !errors.Is(err, errGraphicsResourceArgument) ||
		!containsSubstring(err.Error(), "maximum of 72 bones") {
		t.Fatalf("SetBoneTransforms(73) = %v", err)
	}
	// Exactly the limit passes the guard and reaches the effect, which is not
	// there -- so the refusal that follows is the NATIVE one, not a guard.
	exactly := make([]framework.Matrix, SkinnedEffectMaxBones)
	err = effect.SetBoneTransforms(exactly)
	if err == nil || errors.Is(err, errGraphicsResourceArgument) {
		t.Fatalf("SetBoneTransforms(72) = %v, want the missing-effect refusal", err)
	}
	// GetBoneTransforms: a non-positive count is ArgumentOutOfRangeException
	// with NO message, and an over-limit one carries the same formatted text.
	for _, count := range []int32{0, -1} {
		_, err := effect.GetBoneTransforms(count)
		if err == nil || !errors.Is(err, errArgumentOutOfRange) {
			t.Fatalf("GetBoneTransforms(%d) = %v", count, err)
		}
		if containsSubstring(err.Error(), "maximum of") {
			t.Fatalf("GetBoneTransforms(%d) carried the max-bones message; the reference's zero-count throw has none", count)
		}
	}
	if _, err := effect.GetBoneTransforms(SkinnedEffectMaxBones + 1); err == nil ||
		!errors.Is(err, errArgumentOutOfRange) || !containsSubstring(err.Error(), "maximum of 72 bones") {
		t.Fatalf("GetBoneTransforms(73) = %v", err)
	}
}

// TestSkinnedEffectMaxBonesIsTheReferenceConstant pins the value and the fact
// that it is a constant rather than a field read.
func TestSkinnedEffectMaxBonesIsTheReferenceConstant(t *testing.T) {
	if SkinnedEffectMaxBones != 72 {
		t.Fatalf("SkinnedEffectMaxBones = %d, want 72 (0x48)", SkinnedEffectMaxBones)
	}
	// A constant can size an array, which is what a `public static literal` in
	// the reference is for.
	transforms := make([]framework.Matrix, SkinnedEffectMaxBones)
	if len(transforms) != 72 {
		t.Fatal("the constant did not size an array")
	}
}

// TestTheLitStockEffectsSatisfyAllThreeInterfaces closes the family: every one
// of Effect's six derived types is projected, and the three that declare
// IEffectLights satisfy it.
func TestTheLitStockEffectsSatisfyAllThreeInterfaces(t *testing.T) {
	for name, effect := range map[string]any{
		"BasicEffect":          newBasicEffectState(),
		"EnvironmentMapEffect": newEnvironmentMapEffectState(),
		"SkinnedEffect":        newSkinnedEffectState(),
	} {
		if _, ok := effect.(IEffectMatrices); !ok {
			t.Fatalf("%s does not satisfy IEffectMatrices", name)
		}
		if _, ok := effect.(IEffectFog); !ok {
			t.Fatalf("%s does not satisfy IEffectFog", name)
		}
		if _, ok := effect.(IEffectLights); !ok {
			t.Fatalf("%s does not satisfy IEffectLights", name)
		}
	}
	// And the two unlit ones still do NOT, which is what keeps the split real.
	for name, effect := range map[string]any{
		"AlphaTestEffect":   newAlphaTestEffectState(),
		"DualTextureEffect": newDualTextureEffectState(),
	} {
		if _, ok := effect.(IEffectLights); ok {
			t.Fatalf("%s satisfies IEffectLights", name)
		}
	}
}

// TestTextureCubeReferenceCarriesBothHalves is the fourth substitutable family,
// live since EnvironmentMapEffect gave TextureCube its only parameter position.
func TestTextureCubeReferenceCarriesBothHalves(t *testing.T) {
	var fromCube TextureCubeReference = &TextureCube{}
	var fromTarget TextureCubeReference = &RenderTargetCube{}
	if fromCube == nil || fromTarget == nil {
		t.Fatal("a TextureCube and a RenderTargetCube must both satisfy TextureCubeReference")
	}
	cube := &TextureCube{}
	if resolveTextureCube(cube) != cube {
		t.Fatal("a TextureCube did not resolve to itself")
	}
	target := &RenderTargetCube{cube: cube}
	if resolveTextureCube(target) != cube {
		t.Fatal("a RenderTargetCube did not resolve to its composed base")
	}
	if resolveTextureCube(nil) != nil {
		t.Fatal("resolveTextureCube(nil) is not nil")
	}
	var typedNil *RenderTargetCube
	if resolveTextureCube(typedNil) != nil {
		t.Fatal("an interface holding a typed nil did not resolve to nil")
	}
}
