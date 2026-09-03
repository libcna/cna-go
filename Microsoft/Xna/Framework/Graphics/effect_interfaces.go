package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// IEffectMatrices is XNA's transform contract for a built-in effect. It is a
// pure managed contract: in Microsoft.Xna.Framework.Graphics.dll all five
// shipped implementors -- AlphaTestEffect, BasicEffect, DualTextureEffect,
// EnvironmentMapEffect, and SkinnedEffect -- back every one of these six
// operations with a managed field read or write plus a managed dirty-flag OR.
// None of them touches a device, so none of them can fail, and no operation
// here carries an error result.
//
// Foundation 79 gave the interface its first implementor: BasicEffect satisfies
// it with exactly these six infallible operations, and the compiler checks that
// in basic_effect.go. What the doc above says about the five shipped
// implementors is unchanged and is what made the signatures infallible.
type IEffectMatrices interface {
	World() framework.Matrix
	SetWorld(value framework.Matrix)
	View() framework.Matrix
	SetView(value framework.Matrix)
	Projection() framework.Matrix
	SetProjection(value framework.Matrix)
}

// IEffectFog is XNA's fog contract for a built-in effect. Unlike
// IEffectMatrices it is not uniformly managed, and the split is measured
// rather than assumed.
//
// The same five shipped implementors back FogEnabled, FogStart, and FogEnd
// with a managed field read or write plus a managed dirty-flag OR, so those
// six operations cannot fail. All five route FogColor through
// EffectParameter::GetValueVector3 and EffectParameter::SetValue, which end in
// `calli unmanaged stdcall` into ID3DXBaseEffect and throw
// GraphicsHelpers::GetExceptionFromResult on a negative HRESULT. Both FogColor
// accessors therefore cross a qualified runtime boundary and both carry an
// error result, while their six siblings do not.
//
// As with IEffectMatrices, BasicEffect is the first implementor, and the split
// is what its projection has: six field accesses and one pair that crosses.
type IEffectFog interface {
	FogEnabled() bool
	SetFogEnabled(value bool)
	FogStart() float32
	SetFogStart(value float32)
	FogEnd() float32
	SetFogEnd(value float32)
	FogColor() (framework.Vector3, error)
	SetFogColor(value framework.Vector3) error
}

// IEffectLights is XNA's lighting contract for a built-in effect, and the third
// of the three the stock effects share.
//
//	.class interface public abstract auto ansi IEffectLights
//	  DirectionalLight0/1/2   get only
//	  AmbientLightColor       get and set
//	  LightingEnabled         get and set
//	  EnableDefaultLighting()
//
// Its fallibility split is measured the same way the other two were, and it is
// the least uniform of the three.
//
// The three light accessors and both LightingEnabled accessors are one `ldfld`
// or one `stfld` plus a dirty-flag OR in all five shipped implementors, so
// none can fail. AmbientLightColor is the same in BasicEffect, SkinnedEffect
// and EnvironmentMapEffect -- the only three that declare it.
//
// EnableDefaultLighting is not. Every implementor routes it through
// EffectHelpers::EnableDefaultLighting, which calls twelve DirectionalLight
// setters, and each of those writes through an EffectParameter with
// `callvirt EffectParameter::SetValue`, ending in `calli unmanaged stdcall`.
// It therefore crosses a qualified runtime boundary and carries an error
// result, alone among the eight operations.
//
// DirectionalLight's OWN accessors are the mirror image and are declared on the
// class rather than here: all four getters are `ldfld` of a cached field and
// all four setters write through a parameter, so the type reads infallibly and
// writes fallibly.
type IEffectLights interface {
	DirectionalLight0() *DirectionalLight
	DirectionalLight1() *DirectionalLight
	DirectionalLight2() *DirectionalLight
	AmbientLightColor() framework.Vector3
	SetAmbientLightColor(value framework.Vector3)
	LightingEnabled() bool
	SetLightingEnabled(value bool)
	EnableDefaultLighting() error
}
