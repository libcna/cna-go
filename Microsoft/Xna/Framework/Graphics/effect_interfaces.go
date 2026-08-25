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
// Declaring the contract implements no effect. CNA-Go has no effect runtime
// and no implementor of this interface; the type exists so that a consumer can
// name the contract and so that a future effect implementation has an exact
// signature to satisfy.
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
// As with IEffectMatrices, declaring the contract implements no effect.
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
