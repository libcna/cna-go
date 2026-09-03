package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// DualTextureEffect is the stock effect for two-layer texturing: a base texture
// modulated by a second, with no lighting.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # It is AlphaTestEffect's shape with a second texture instead of the test
//
//	.class public auto ansi beforefieldinit DualTextureEffect
//	       extends Effect
//	       implements IEffectMatrices, IEffectFog
//
// Its thirteen members split the way every stock effect's do. THREE cross:
// FogColor, Texture and Texture2, each 12 bytes to read and 13 to write through
// an EffectParameter. The other ten are one `ldfld` and one store plus a
// dirty-flag `or`, and every flag word agrees with AlphaTestEffect's --
// including `set_World` and `set_View` raising WorldViewProj|Fog rather than
// BasicEffect's three-flag words, because this effect has no lighting either.
//
// # One CNA route backs two properties
//
// The contract declares Texture and Texture2 as two properties;
// `cna_dual_texture_effect_set_texture` takes a LAYER INDEX, "zero or one". So
// one route backs both setters, at index 0 and index 1, which is a shape no
// other stock effect has.
type DualTextureEffect struct {
	effect *Effect

	world, view, projection framework.Matrix
	diffuseColor            framework.Vector3
	alpha                   float32
	fogEnabled              bool
	fogStart, fogEnd        float32
	vertexColorEnabled      bool
	dirtyFlags              effectDirtyFlags

	// The two layers, held for the reason every stock effect's texture is.
	texture, texture2 *Texture2D
}

// errDualTextureEffectNil is the Go-only guard for a zero value.
var errDualTextureEffectNil = errors.New("dual texture effect is nil or uninitialized")

// NewDualTextureEffectByGraphicsDevice is
// DualTextureEffect::.ctor(GraphicsDevice), 92 bytes -- AlphaTestEffect's
// constructor without the `alphaFunction` store:
//
//	world = view = projection = Matrix.Identity;
//	diffuseColor = Vector3.One;  alpha = 1;  fogEnd = 1;  dirtyFlags = -1;
//	base(device, DualTextureEffectCode.Code);
//	CacheEffectParameters();
func NewDualTextureEffectByGraphicsDevice(device *GraphicsDevice) (*DualTextureEffect, error) {
	if device == nil || device.device == nil {
		return nil, fmt.Errorf("%w: device: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := device.device.CreateDualTextureEffect()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(device, resource)
	if err != nil {
		return nil, err
	}
	dual := newDualTextureEffectState()
	dual.effect = effect
	effect.bindDerived(dual)
	effect.bindDerivedEffect(dual)
	return dual, nil
}

// NewDualTextureEffectByDualTextureEffect is
// DualTextureEffect::.ctor(DualTextureEffect cloneSource), the PROTECTED clone
// constructor. It copies NINE values: fogEnabled, vertexColorEnabled, the three
// matrices, diffuseColor, alpha, fogStart and fogEnd.
func NewDualTextureEffectByDualTextureEffect(cloneSource *DualTextureEffect) (*DualTextureEffect, error) {
	if cloneSource == nil || cloneSource.effect == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned, err := cloneSource.effect.cloneBase()
	if err != nil {
		return nil, err
	}
	dual := newDualTextureEffectState()
	dual.effect = cloned
	cloned.bindDerived(dual)
	cloned.bindDerivedEffect(dual)
	dual.fogEnabled = cloneSource.fogEnabled
	dual.vertexColorEnabled = cloneSource.vertexColorEnabled
	dual.world = cloneSource.world
	dual.view = cloneSource.view
	dual.projection = cloneSource.projection
	dual.diffuseColor = cloneSource.diffuseColor
	dual.alpha = cloneSource.alpha
	dual.fogStart = cloneSource.fogStart
	dual.fogEnd = cloneSource.fogEnd
	dual.texture = cloneSource.texture
	dual.texture2 = cloneSource.texture2
	return dual, nil
}

func newDualTextureEffectState() *DualTextureEffect {
	return &DualTextureEffect{
		world:        framework.MatrixIdentity(),
		view:         framework.MatrixIdentity(),
		projection:   framework.MatrixIdentity(),
		diffuseColor: framework.Vector3One(),
		alpha:        1,
		fogEnd:       1,
		dirtyFlags:   effectDirtyAll,
	}
}

func (e *DualTextureEffect) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.DualTextureEffect"
}

func (e *DualTextureEffect) nativeResource() *interop.Resource {
	if e == nil || e.effect == nil {
		return nil
	}
	return e.effect.nativeResource()
}

// ---------------------------------------------------------------------------
// IEffectMatrices.
// ---------------------------------------------------------------------------

// World is DualTextureEffect::get_World.
func (e *DualTextureEffect) World() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.world
}

// SetWorld is DualTextureEffect::set_World: WorldViewProj|Fog, the unlit pair.
func (e *DualTextureEffect) SetWorld(value framework.Matrix) {
	if e == nil {
		return
	}
	e.world = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyFog
}

// View is DualTextureEffect::get_View.
func (e *DualTextureEffect) View() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.view
}

// SetView is DualTextureEffect::set_View.
func (e *DualTextureEffect) SetView(value framework.Matrix) {
	if e == nil {
		return
	}
	e.view = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyFog
}

// Projection is DualTextureEffect::get_Projection.
func (e *DualTextureEffect) Projection() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.projection
}

// SetProjection is DualTextureEffect::set_Projection.
func (e *DualTextureEffect) SetProjection(value framework.Matrix) {
	if e == nil {
		return
	}
	e.projection = value
	e.dirtyFlags |= effectDirtyWorldViewProj
}

// ---------------------------------------------------------------------------
// IEffectFog.
// ---------------------------------------------------------------------------

// FogEnabled is DualTextureEffect::get_FogEnabled.
func (e *DualTextureEffect) FogEnabled() bool {
	if e == nil {
		return false
	}
	return e.fogEnabled
}

// SetFogEnabled is DualTextureEffect::set_FogEnabled.
func (e *DualTextureEffect) SetFogEnabled(value bool) {
	if e == nil || e.fogEnabled == value {
		return
	}
	e.fogEnabled = value
	e.dirtyFlags |= effectDirtyFogEnable | effectDirtyShaderIndex
}

// FogStart is DualTextureEffect::get_FogStart.
func (e *DualTextureEffect) FogStart() float32 {
	if e == nil {
		return 0
	}
	return e.fogStart
}

// SetFogStart is DualTextureEffect::set_FogStart.
func (e *DualTextureEffect) SetFogStart(value float32) {
	if e == nil {
		return
	}
	e.fogStart = value
	e.dirtyFlags |= effectDirtyFog
}

// FogEnd is DualTextureEffect::get_FogEnd.
func (e *DualTextureEffect) FogEnd() float32 {
	if e == nil {
		return 0
	}
	return e.fogEnd
}

// SetFogEnd is DualTextureEffect::set_FogEnd.
func (e *DualTextureEffect) SetFogEnd(value float32) {
	if e == nil {
		return
	}
	e.fogEnd = value
	e.dirtyFlags |= effectDirtyFog
}

// FogColor is DualTextureEffect::get_FogColor.
func (e *DualTextureEffect) FogColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errDualTextureEffectNil
	}
	values, err := resource.EffectFogColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetFogColor is DualTextureEffect::set_FogColor.
func (e *DualTextureEffect) SetFogColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errDualTextureEffectNil
	}
	return resource.EffectFogSetColor(vector3Triple(value))
}

// ---------------------------------------------------------------------------
// DualTextureEffect's own surface.
// ---------------------------------------------------------------------------

// DiffuseColor is DualTextureEffect::get_DiffuseColor.
func (e *DualTextureEffect) DiffuseColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.diffuseColor
}

// SetDiffuseColor is DualTextureEffect::set_DiffuseColor.
func (e *DualTextureEffect) SetDiffuseColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.diffuseColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// Alpha is DualTextureEffect::get_Alpha.
func (e *DualTextureEffect) Alpha() float32 {
	if e == nil {
		return 0
	}
	return e.alpha
}

// SetAlpha is DualTextureEffect::set_Alpha.
func (e *DualTextureEffect) SetAlpha(value float32) {
	if e == nil {
		return
	}
	e.alpha = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// VertexColorEnabled is DualTextureEffect::get_VertexColorEnabled.
func (e *DualTextureEffect) VertexColorEnabled() bool {
	if e == nil {
		return false
	}
	return e.vertexColorEnabled
}

// SetVertexColorEnabled is DualTextureEffect::set_VertexColorEnabled.
func (e *DualTextureEffect) SetVertexColorEnabled(value bool) {
	if e == nil || e.vertexColorEnabled == value {
		return
	}
	e.vertexColorEnabled = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// Texture is DualTextureEffect::get_Texture, layer zero.
func (e *DualTextureEffect) Texture() (*Texture2D, error) {
	if e == nil {
		return nil, errDualTextureEffectNil
	}
	return e.texture, nil
}

// SetTexture is DualTextureEffect::set_Texture, which reaches CNA at index 0.
func (e *DualTextureEffect) SetTexture(value Texture2DReference) error {
	return e.setLayer(0, value, &e.texture)
}

// Texture2 is DualTextureEffect::get_Texture2, layer one.
func (e *DualTextureEffect) Texture2() (*Texture2D, error) {
	if e == nil {
		return nil, errDualTextureEffectNil
	}
	return e.texture2, nil
}

// SetTexture2 is DualTextureEffect::set_Texture2, index 1 of the same route.
func (e *DualTextureEffect) SetTexture2(value Texture2DReference) error {
	return e.setLayer(1, value, &e.texture2)
}

// setLayer is the shared body of the two setters, which are the same 13 bytes
// in the reference over two different parameters and the same route here over
// two different indices.
func (e *DualTextureEffect) setLayer(index uint32, value Texture2DReference, field **Texture2D) error {
	resource := e.nativeResource()
	if resource == nil {
		return errDualTextureEffectNil
	}
	texture := resolveTexture2D(value)
	if err := resource.DualTextureEffectSetTexture(index, texture.nativeResource()); err != nil {
		return err
	}
	*field = texture
	return nil
}

// ---------------------------------------------------------------------------
// The two virtual members, and the push.
// ---------------------------------------------------------------------------

// Clone is DualTextureEffect::Clone.
func (e *DualTextureEffect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errDualTextureEffectNil
	}
	return NewDualTextureEffectByDualTextureEffect(e)
}

// OnApply is DualTextureEffect::OnApply, 206 bytes -- the shortest of the five,
// because there is no lighting to push and no alpha test to encode.
func (e *DualTextureEffect) OnApply() error {
	if e == nil {
		return errDualTextureEffectNil
	}
	resource := e.nativeResource()
	if resource == nil {
		return errDualTextureEffectNil
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
		if err := resource.DualTextureEffectSetDiffuseColor(vector3Triple(e.diffuseColor)); err != nil {
			return err
		}
		if err := resource.DualTextureEffectSetAlpha(e.alpha); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyMaterialColor
	}
	if e.dirtyFlags&effectDirtyShaderIndex != 0 {
		if err := resource.DualTextureEffectSetVertexColorEnabled(e.vertexColorEnabled); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyShaderIndex
	}
	return nil
}

// releaseDerivedNativeObjects has nothing to release: this effect publishes no
// views of its own.
func (e *DualTextureEffect) releaseDerivedNativeObjects() error { return nil }

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *DualTextureEffect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *DualTextureEffect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *DualTextureEffect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *DualTextureEffect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errDualTextureEffectNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *DualTextureEffect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *DualTextureEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *DualTextureEffect) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *DualTextureEffect) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *DualTextureEffect) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *DualTextureEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (e *DualTextureEffect) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *DualTextureEffect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errDualTextureEffectNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *DualTextureEffect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errDualTextureEffectNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), this type's only dispose member.
func (e *DualTextureEffect) Dispose() error {
	if e == nil {
		return errDualTextureEffectNil
	}
	return e.effect.DisposeByNone()
}

var (
	_ IEffectMatrices = (*DualTextureEffect)(nil)
	_ IEffectFog      = (*DualTextureEffect)(nil)
)
