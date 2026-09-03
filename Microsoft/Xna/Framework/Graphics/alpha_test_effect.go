package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// AlphaTestEffect is the stock effect for alpha-tested rendering: one texture,
// no lighting, and a comparison the pixel's alpha must pass.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # It is BasicEffect's shape with the lighting removed
//
//	.class public auto ansi beforefieldinit AlphaTestEffect
//	       extends Effect
//	       implements IEffectMatrices, IEffectFog
//
// Foundation 79 settled that shape against BasicEffect and it is unchanged
// here: managed state with the reference's dirty flags, a push in OnApply, and
// only the properties the reference backs with an EffectParameter reaching CNA
// directly. Two of the sixteen members do:
//
//	FogColor   fogColorParam.GetValueVector3() / SetValue    12 and 13 bytes
//	Texture    textureParam.GetValueTexture2D() / SetValue   12 and 13 bytes
//
// The other fourteen are `ldarg.0; ldfld <field>; ret` to read and a store plus
// a dirty-flag `or` to write.
//
// # The two flag words that are NOT BasicEffect's
//
// Sharing an accessor body across the stock effects would have been wrong, and
// the reason is measured rather than suspected:
//
//	                 BasicEffect            AlphaTestEffect
//	set_World        19 = WorldViewProj|World|Fog    17 = WorldViewProj|Fog
//	set_View         21 = WorldViewProj|EyePosition|Fog   17 = WorldViewProj|Fog
//
// AlphaTestEffect has no world parameter and no eye position, because it has no
// lighting, so its two setters raise two flags where BasicEffect's raise three.
// Every other flag word in the type agrees with BasicEffect's.
//
// # Its own pair
//
// AlphaFunction and ReferenceAlpha both raise `64`, the AlphaTest flag, which no
// other stock effect uses. The default AlphaFunction is `ldc.i4.6` --
// CompareFunction.Greater -- and ReferenceAlpha's default is the field's zero.
type AlphaTestEffect struct {
	effect *Effect

	world, view, projection framework.Matrix
	diffuseColor            framework.Vector3
	alpha                   float32
	fogEnabled              bool
	fogStart, fogEnd        float32
	vertexColorEnabled      bool
	alphaFunction           CompareFunction
	referenceAlpha          int32
	dirtyFlags              effectDirtyFlags

	// texture is held for the reason BasicEffect's is: CNA reports an effect
	// texture as a handle and the property's value is an object, and this
	// effect is the sole writer of it.
	texture *Texture2D
}

// errAlphaTestEffectNil is the Go-only guard for a zero value.
var errAlphaTestEffectNil = errors.New("alpha test effect is nil or uninitialized")

// NewAlphaTestEffectByGraphicsDevice is AlphaTestEffect::.ctor(GraphicsDevice),
// 99 bytes:
//
//	world = view = projection = Matrix.Identity;
//	diffuseColor = Vector3.One;  alpha = 1;  fogEnd = 1;
//	alphaFunction = CompareFunction.Greater;  dirtyFlags = -1;
//	base(device, AlphaTestEffectCode.Code);
//	CacheEffectParameters();
//
// It has no tail after CacheEffectParameters, unlike BasicEffect's three
// statements, so every default this type has is a field initialiser.
//
// `AlphaTestEffectCode.Code` is Microsoft's compiled shader and cannot ship
// here; `cna_alpha_test_effect_create` makes CNA's own. CacheEffectParameters
// looks up seven parameters by name in a collection the Foundation 79 probe
// measured EMPTY, so nothing survives of it.
func NewAlphaTestEffectByGraphicsDevice(device *GraphicsDevice) (*AlphaTestEffect, error) {
	if device == nil || device.device == nil {
		return nil, fmt.Errorf("%w: device: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := device.device.CreateAlphaTestEffect()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(device, resource)
	if err != nil {
		return nil, err
	}
	alphaTest := newAlphaTestEffectState()
	alphaTest.effect = effect
	effect.bindDerived(alphaTest)
	effect.bindDerivedEffect(alphaTest)
	return alphaTest, nil
}

// NewAlphaTestEffectByAlphaTestEffect is
// AlphaTestEffect::.ctor(AlphaTestEffect cloneSource), the PROTECTED
// constructor Clone() uses.
//
// It seeds the same initialisers, chains to `base(cloneSource)` and then copies
// ELEVEN values: fogEnabled, vertexColorEnabled, the three matrices,
// diffuseColor, alpha, fogStart, fogEnd, alphaFunction and referenceAlpha.
// FogColor and Texture are absent because they live in the cloned effect.
func NewAlphaTestEffectByAlphaTestEffect(cloneSource *AlphaTestEffect) (*AlphaTestEffect, error) {
	if cloneSource == nil || cloneSource.effect == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned, err := cloneSource.effect.cloneBase()
	if err != nil {
		return nil, err
	}
	alphaTest := newAlphaTestEffectState()
	alphaTest.effect = cloned
	cloned.bindDerived(alphaTest)
	cloned.bindDerivedEffect(alphaTest)
	alphaTest.fogEnabled = cloneSource.fogEnabled
	alphaTest.vertexColorEnabled = cloneSource.vertexColorEnabled
	alphaTest.world = cloneSource.world
	alphaTest.view = cloneSource.view
	alphaTest.projection = cloneSource.projection
	alphaTest.diffuseColor = cloneSource.diffuseColor
	alphaTest.alpha = cloneSource.alpha
	alphaTest.fogStart = cloneSource.fogStart
	alphaTest.fogEnd = cloneSource.fogEnd
	alphaTest.alphaFunction = cloneSource.alphaFunction
	alphaTest.referenceAlpha = cloneSource.referenceAlpha
	// The texture is copied for the reason BasicEffect's is: the projection
	// holds the managed object and CNA's clone carries the native half.
	alphaTest.texture = cloneSource.texture
	return alphaTest, nil
}

func newAlphaTestEffectState() *AlphaTestEffect {
	return &AlphaTestEffect{
		world:         framework.MatrixIdentity(),
		view:          framework.MatrixIdentity(),
		projection:    framework.MatrixIdentity(),
		diffuseColor:  framework.Vector3One(),
		alpha:         1,
		fogEnd:        1,
		alphaFunction: CompareFunctionGreater,
		dirtyFlags:    effectDirtyAll,
	}
}

func (e *AlphaTestEffect) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.AlphaTestEffect"
}

func (e *AlphaTestEffect) nativeResource() *interop.Resource {
	if e == nil || e.effect == nil {
		return nil
	}
	return e.effect.nativeResource()
}

// ---------------------------------------------------------------------------
// IEffectMatrices.
// ---------------------------------------------------------------------------

// World is AlphaTestEffect::get_World.
func (e *AlphaTestEffect) World() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.world
}

// SetWorld is AlphaTestEffect::set_World, which raises WorldViewProj and Fog --
// and NOT World, which BasicEffect's raises, because this effect has no world
// parameter to write.
func (e *AlphaTestEffect) SetWorld(value framework.Matrix) {
	if e == nil {
		return
	}
	e.world = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyFog
}

// View is AlphaTestEffect::get_View.
func (e *AlphaTestEffect) View() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.view
}

// SetView is AlphaTestEffect::set_View, which raises the same two -- and NOT
// EyePosition, because an unlit effect has no eye position.
func (e *AlphaTestEffect) SetView(value framework.Matrix) {
	if e == nil {
		return
	}
	e.view = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyFog
}

// Projection is AlphaTestEffect::get_Projection.
func (e *AlphaTestEffect) Projection() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.projection
}

// SetProjection is AlphaTestEffect::set_Projection.
func (e *AlphaTestEffect) SetProjection(value framework.Matrix) {
	if e == nil {
		return
	}
	e.projection = value
	e.dirtyFlags |= effectDirtyWorldViewProj
}

// ---------------------------------------------------------------------------
// IEffectFog.
// ---------------------------------------------------------------------------

// FogEnabled is AlphaTestEffect::get_FogEnabled.
func (e *AlphaTestEffect) FogEnabled() bool {
	if e == nil {
		return false
	}
	return e.fogEnabled
}

// SetFogEnabled is AlphaTestEffect::set_FogEnabled, 35 bytes: an early return
// then FogEnable|ShaderIndex.
func (e *AlphaTestEffect) SetFogEnabled(value bool) {
	if e == nil || e.fogEnabled == value {
		return
	}
	e.fogEnabled = value
	e.dirtyFlags |= effectDirtyFogEnable | effectDirtyShaderIndex
}

// FogStart is AlphaTestEffect::get_FogStart.
func (e *AlphaTestEffect) FogStart() float32 {
	if e == nil {
		return 0
	}
	return e.fogStart
}

// SetFogStart is AlphaTestEffect::set_FogStart.
func (e *AlphaTestEffect) SetFogStart(value float32) {
	if e == nil {
		return
	}
	e.fogStart = value
	e.dirtyFlags |= effectDirtyFog
}

// FogEnd is AlphaTestEffect::get_FogEnd.
func (e *AlphaTestEffect) FogEnd() float32 {
	if e == nil {
		return 0
	}
	return e.fogEnd
}

// SetFogEnd is AlphaTestEffect::set_FogEnd.
func (e *AlphaTestEffect) SetFogEnd(value float32) {
	if e == nil {
		return
	}
	e.fogEnd = value
	e.dirtyFlags |= effectDirtyFog
}

// FogColor is AlphaTestEffect::get_FogColor, one of the two members that really
// do reach the effect.
func (e *AlphaTestEffect) FogColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errAlphaTestEffectNil
	}
	values, err := resource.EffectFogColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetFogColor is AlphaTestEffect::set_FogColor, a parameter write with no cache
// and no dirty flag.
func (e *AlphaTestEffect) SetFogColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errAlphaTestEffectNil
	}
	return resource.EffectFogSetColor(vector3Triple(value))
}

// ---------------------------------------------------------------------------
// AlphaTestEffect's own surface.
// ---------------------------------------------------------------------------

// DiffuseColor is AlphaTestEffect::get_DiffuseColor, the STORED colour rather
// than the alpha-premultiplied one OnApply pushes.
func (e *AlphaTestEffect) DiffuseColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.diffuseColor
}

// SetDiffuseColor is AlphaTestEffect::set_DiffuseColor.
func (e *AlphaTestEffect) SetDiffuseColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.diffuseColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// Alpha is AlphaTestEffect::get_Alpha.
func (e *AlphaTestEffect) Alpha() float32 {
	if e == nil {
		return 0
	}
	return e.alpha
}

// SetAlpha is AlphaTestEffect::set_Alpha.
func (e *AlphaTestEffect) SetAlpha(value float32) {
	if e == nil {
		return
	}
	e.alpha = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// VertexColorEnabled is AlphaTestEffect::get_VertexColorEnabled.
func (e *AlphaTestEffect) VertexColorEnabled() bool {
	if e == nil {
		return false
	}
	return e.vertexColorEnabled
}

// SetVertexColorEnabled is AlphaTestEffect::set_VertexColorEnabled, a shader
// permutation with an early return.
func (e *AlphaTestEffect) SetVertexColorEnabled(value bool) {
	if e == nil || e.vertexColorEnabled == value {
		return
	}
	e.vertexColorEnabled = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// AlphaFunction is AlphaTestEffect::get_AlphaFunction. The default is
// CompareFunction.Greater, stored by the constructor as `ldc.i4.6`.
func (e *AlphaTestEffect) AlphaFunction() CompareFunction {
	if e == nil {
		return CompareFunctionAlways
	}
	return e.alphaFunction
}

// SetAlphaFunction is AlphaTestEffect::set_AlphaFunction, 23 bytes: a store
// plus the AlphaTest flag, which no other stock effect raises. It does NOT
// validate the value -- an undeclared CompareFunction is stored and reported
// back, exactly as the reference stores it.
func (e *AlphaTestEffect) SetAlphaFunction(value CompareFunction) {
	if e == nil {
		return
	}
	e.alphaFunction = value
	e.dirtyFlags |= effectDirtyAlphaTest
}

// ReferenceAlpha is AlphaTestEffect::get_ReferenceAlpha, whose default is the
// field's zero rather than a stored constant.
func (e *AlphaTestEffect) ReferenceAlpha() int32 {
	if e == nil {
		return 0
	}
	return e.referenceAlpha
}

// SetReferenceAlpha is AlphaTestEffect::set_ReferenceAlpha, which raises the
// same AlphaTest flag and validates nothing.
func (e *AlphaTestEffect) SetReferenceAlpha(value int32) {
	if e == nil {
		return
	}
	e.referenceAlpha = value
	e.dirtyFlags |= effectDirtyAlphaTest
}

// Texture is AlphaTestEffect::get_Texture. It answers the managed field for the
// reason recorded on BasicEffect::Texture: CNA reports a handle, the property's
// value is an object, and this effect is the sole writer of it.
func (e *AlphaTestEffect) Texture() (*Texture2D, error) {
	if e == nil {
		return nil, errAlphaTestEffectNil
	}
	return e.texture, nil
}

// SetTexture is AlphaTestEffect::set_Texture. The parameter widens because a
// property setter is a parameter position and RenderTarget2D is a Texture2D.
func (e *AlphaTestEffect) SetTexture(value Texture2DReference) error {
	resource := e.nativeResource()
	if resource == nil {
		return errAlphaTestEffectNil
	}
	texture := resolveTexture2D(value)
	if err := resource.AlphaTestEffectSetTexture(texture.nativeResource()); err != nil {
		return err
	}
	e.texture = texture
	return nil
}

// ---------------------------------------------------------------------------
// The two virtual members, and the push.
// ---------------------------------------------------------------------------

// Clone is AlphaTestEffect::Clone, seven bytes:
// `newobj AlphaTestEffect::.ctor(AlphaTestEffect)`.
func (e *AlphaTestEffect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errAlphaTestEffectNil
	}
	return NewAlphaTestEffectByAlphaTestEffect(e)
}

// OnApply is AlphaTestEffect::OnApply, 686 bytes in the reference -- the
// longest of the five, because the alpha test is expressed to the shader as a
// pair of vectors whose four components the body selects from a table keyed on
// the comparison.
//
// CNA computes that itself from the comparison and the reference alpha, so what
// this pushes is the raw pair, gated by the same AlphaTest flag the reference
// gates its own computation with.
func (e *AlphaTestEffect) OnApply() error {
	if e == nil {
		return errAlphaTestEffectNil
	}
	resource := e.nativeResource()
	if resource == nil {
		return errAlphaTestEffectNil
	}
	if e.dirtyFlags&effectDirtyWorldViewProj != 0 {
		if err := pushEffectMatrices(resource, e.world, e.view, e.projection); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyWorldViewProj
	}
	if e.dirtyFlags&effectDirtyFog != 0 {
		if err := resource.EffectFogSetStart(e.fogStart); err != nil {
			return err
		}
		if err := resource.EffectFogSetEnd(e.fogEnd); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyFog
	}
	if e.dirtyFlags&effectDirtyFogEnable != 0 {
		if err := resource.EffectFogSetEnabled(e.fogEnabled); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyFogEnable
	}
	if e.dirtyFlags&effectDirtyMaterialColor != 0 {
		if err := resource.AlphaTestEffectSetDiffuseColor(vector3Triple(e.diffuseColor)); err != nil {
			return err
		}
		if err := resource.AlphaTestEffectSetAlpha(e.alpha); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyMaterialColor
	}
	if e.dirtyFlags&effectDirtyAlphaTest != 0 {
		if err := resource.AlphaTestEffectSetAlphaFunction(uint32(e.alphaFunction)); err != nil {
			return err
		}
		if err := resource.AlphaTestEffectSetReferenceAlpha(e.referenceAlpha); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyAlphaTest
	}
	if e.dirtyFlags&effectDirtyShaderIndex != 0 {
		if err := resource.AlphaTestEffectSetVertexColorEnabled(e.vertexColorEnabled); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyShaderIndex
	}
	return nil
}

// releaseDerivedNativeObjects is the derived half of Effect's disposal. This
// effect publishes no views of its own, so it has nothing to release -- the
// member exists because effectVirtuals requires it, and answering nil is the
// honest answer rather than a missing implementation.
func (e *AlphaTestEffect) releaseDerivedNativeObjects() error { return nil }

// pushEffectMatrices is the three-matrix push every stock effect makes, in the
// order the CNA_Matrix declaration takes them. It is shared because the WRITE
// is identical across the five; the flag words that decide WHEN to write are
// not, and stay on each type.
func pushEffectMatrices(resource *interop.Resource, world, view, projection framework.Matrix) error {
	for _, write := range []struct {
		value framework.Matrix
		apply func([16]float32) error
	}{
		{world, resource.EffectMatricesSetWorld},
		{view, resource.EffectMatricesSetView},
		{projection, resource.EffectMatricesSetProjection},
	} {
		if err := write.apply(matrixToRowMajor(write.value)); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *AlphaTestEffect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *AlphaTestEffect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *AlphaTestEffect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *AlphaTestEffect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errAlphaTestEffectNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *AlphaTestEffect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *AlphaTestEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *AlphaTestEffect) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *AlphaTestEffect) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *AlphaTestEffect) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *AlphaTestEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString, which answers this type's name.
func (e *AlphaTestEffect) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *AlphaTestEffect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errAlphaTestEffectNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *AlphaTestEffect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errAlphaTestEffectNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), and this type's ONLY dispose member:
// it declares no Dispose of its own, so its inherited PUBLIC surface carries
// one that takes no argument.
func (e *AlphaTestEffect) Dispose() error {
	if e == nil {
		return errAlphaTestEffectNil
	}
	return e.effect.DisposeByNone()
}

var (
	_ IEffectMatrices = (*AlphaTestEffect)(nil)
	_ IEffectFog      = (*AlphaTestEffect)(nil)
)
