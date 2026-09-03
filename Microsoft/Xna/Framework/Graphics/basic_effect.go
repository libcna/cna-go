package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// BasicEffect is XNA's stock effect: transform, material, lighting, fog and one
// texture, with no shader for a consumer to write.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # The reference is MANAGED STATE plus one push in OnApply
//
//	.class public auto ansi beforefieldinit BasicEffect
//	       extends Effect
//	       implements IEffectMatrices, IEffectLights, IEffectFog
//
// It declares twelve EffectParameter fields, sixteen state fields, three
// DirectionalLights and an EffectDirtyFlags word. Almost every property is one
// `ldfld` to read and one `stfld` plus a dirty-flag `or` to write; OnApply then
// computes the derived quantities -- WorldViewProj, the fog vector, the inverse
// transpose, the eye position, the shader index and the alpha-premultiplied
// material colours -- and writes them into the parameters it cached by name.
//
// Only FOUR properties read or write a parameter directly, and each of those
// four is therefore fallible where its siblings are not:
//
//	SpecularColor   specularColorParam.GetValueVector3() / SetValue
//	SpecularPower   specularPowerParam.GetValueSingle()  / SetValue
//	FogColor        fogColorParam.GetValueVector3()      / SetValue
//	Texture         textureParam.GetValueTexture2D()     / SetValue
//
// That split is the contract, and this projection reproduces it exactly. It is
// also what the three interfaces already pinned in Foundation 18 say: World is
// declared infallible on IEffectMatrices, FogColor fallible on IEffectFog.
//
// # Where the state goes, given that CNA publishes no parameters
//
// The reference's push target does not exist here. The Foundation 79 probe
// loaded CNA's stock BasicEffect through ContentManager on both qualified
// artifacts and got PARAMETER_COUNT 0 from each, with one technique named
// "Default" holding one pass. CNA keeps the same state natively instead --
// `cna_basic_effect_*` plus the three shared interface families
// `cna_effect_matrices_*`, `cna_effect_fog_*` and `cna_effect_lights_*` -- and
// applies it on `cna_effect_apply`.
//
// So the projection keeps the reference's managed state and its dirty flags,
// and OnApply pushes the dirty subset into CNA rather than into parameters.
// CNA derives WorldViewProj, the fog vector and the shader permutation itself,
// which is why the push is the RAW state and not the reference's computed
// values. Nothing here reads CNA back for a property whose reference getter is
// an `ldfld`: doing that would let CNA's clamping decide an XNA getter's answer,
// and the thirteen getter routes that would have served are recorded in the
// native-ABI registry as deliberately unbound under MANAGED_REFERENCE.
//
// # The base, composed
//
// Everything a caller uses that is not one of BasicEffect's own twenty-six
// members -- Parameters, Techniques, CurrentTechnique, GraphicsDevice, Name,
// Tag, IsDisposed, Dispose, the Disposing event -- is inherited public surface
// of Effect, re-exposed here as measured forwarding. Effect's two VIRTUAL
// members, Clone and OnApply, dispatch back to this type through
// bindDerivedEffect, so EffectPass::Apply reaches BasicEffect's OnApply and not
// the base's empty body.
type BasicEffect struct {
	// effect is the composed Effect, which carries the composed
	// GraphicsResource and the one native handle.
	effect *Effect

	// The three lights, built once so the identity a caller observes is stable:
	// the reference's are fields and `DirectionalLight0` is one `ldfld`.
	light0 *DirectionalLight
	light1 *DirectionalLight
	light2 *DirectionalLight

	// The reference's own state fields, in its declaration order.
	lightingEnabled        bool
	preferPerPixelLighting bool
	oneLight               bool
	fogEnabled             bool
	textureEnabled         bool
	vertexColorEnabled     bool
	world                  framework.Matrix
	view                   framework.Matrix
	projection             framework.Matrix
	diffuseColor           framework.Vector3
	emissiveColor          framework.Vector3
	ambientLightColor      framework.Vector3
	alpha                  float32
	fogStart               float32
	fogEnd                 float32
	dirtyFlags             effectDirtyFlags

	// texture is the one field the reference does NOT keep, and the reason is
	// recorded at Texture().
	texture *Texture2D
}

// effectDirtyFlags is Microsoft.Xna.Framework.Graphics.EffectDirtyFlags, a
// private `[Flags] enum EffectDirtyFlags : int` in the reference. It is not in
// the pinned contract -- it is not public -- so it is projected unexported, as
// the closure of a private helper the reference's own body reaches.
//
// The bit values are read from the IL's own constants: the OnApply body tests
// 8 for MaterialColor and 0x80 for ShaderIndex, and clears MaterialColor with
// `and -9`. The constructors seed the word with `ldc.i4.m1`, every bit set, so
// the first OnApply writes everything.
type effectDirtyFlags int32

const (
	effectDirtyWorldViewProj effectDirtyFlags = 1
	effectDirtyWorld         effectDirtyFlags = 2
	effectDirtyEyePosition   effectDirtyFlags = 4
	effectDirtyMaterialColor effectDirtyFlags = 8
	effectDirtyFog           effectDirtyFlags = 16
	effectDirtyFogEnable     effectDirtyFlags = 32
	effectDirtyAlphaTest     effectDirtyFlags = 64
	effectDirtyShaderIndex   effectDirtyFlags = 128
	effectDirtyAll           effectDirtyFlags = -1
)

// errBasicEffectNil is the Go-only guard for a zero value, which the reference
// answers with NullReferenceException.
var errBasicEffectNil = errors.New("basic effect is nil or uninitialized")

// NewBasicEffectByGraphicsDevice is BasicEffect::.ctor(GraphicsDevice):
//
//	world = view = projection = Matrix.Identity;
//	diffuseColor = Vector3.One;  emissiveColor = ambientLightColor = Vector3.Zero;
//	alpha = 1;  fogEnd = 1;  dirtyFlags = -1;
//	base(device, BasicEffectCode.Code);
//	CacheEffectParameters(null);
//	DirectionalLight0.Enabled = true;
//	SpecularColor = Vector3.One;
//	SpecularPower = 16;
//
// Two of those lines cannot be reproduced as written and both are stated here
// rather than hidden. `BasicEffectCode.Code` is Microsoft's compiled shader and
// CNA-Go cannot ship it; `cna_basic_effect_create` makes CNA's own stock
// BasicEffect instead, which is the same trade every native-backed projection
// in this binding makes. And CacheEffectParameters looks up fifteen parameters
// by name -- "Texture", "DiffuseColor", "DirLight0Direction" and the rest -- in
// a collection CNA reports as empty, so the three lights are built over CNA's
// own light views rather than over parameter triples.
//
// Everything else is the reference's, in the reference's order, including the
// detail that DirectionalLight0 is enabled while LightingEnabled is still
// false.
func NewBasicEffectByGraphicsDevice(device *GraphicsDevice) (*BasicEffect, error) {
	if device == nil || device.device == nil {
		return nil, fmt.Errorf("%w: device: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := device.device.CreateBasicEffect()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(device, resource)
	if err != nil {
		return nil, err
	}
	basic := newBasicEffectState()
	basic.effect = effect
	effect.bindDerived(basic)
	effect.bindDerivedEffect(basic)
	if err := basic.cacheEffectParameters(nil); err != nil {
		return nil, err
	}
	if err := basic.light0.SetEnabled(true); err != nil {
		return nil, err
	}
	if err := basic.SetSpecularColor(framework.Vector3One()); err != nil {
		return nil, err
	}
	if err := basic.SetSpecularPower(16); err != nil {
		return nil, err
	}
	return basic, nil
}

// NewBasicEffectByBasicEffect is BasicEffect::.ctor(BasicEffect cloneSource),
// the PROTECTED constructor Clone() uses.
//
// It seeds the same field initialisers the public constructor does, chains to
// `base(cloneSource)` -- which clones the effect -- calls
// CacheEffectParameters(cloneSource), and then copies THIRTEEN values across:
// the five bools, the three matrices, the three Vector3s, alpha, fogStart and
// fogEnd.
//
// What it does NOT copy is as informative as what it does. SpecularColor,
// SpecularPower, FogColor and Texture are absent because they live in the
// cloned effect and come across with it; `oneLight` is absent because OnApply
// recomputes it; and the three lights are absent because
// CacheEffectParameters(cloneSource) builds each new light with the source's as
// its clone source.
func NewBasicEffectByBasicEffect(cloneSource *BasicEffect) (*BasicEffect, error) {
	if cloneSource == nil || cloneSource.effect == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned, err := cloneSource.effect.cloneBase()
	if err != nil {
		return nil, err
	}
	basic := newBasicEffectState()
	basic.effect = cloned
	cloned.bindDerived(basic)
	cloned.bindDerivedEffect(basic)
	if err := basic.cacheEffectParameters(cloneSource); err != nil {
		return nil, err
	}
	basic.lightingEnabled = cloneSource.lightingEnabled
	basic.preferPerPixelLighting = cloneSource.preferPerPixelLighting
	basic.fogEnabled = cloneSource.fogEnabled
	basic.textureEnabled = cloneSource.textureEnabled
	basic.vertexColorEnabled = cloneSource.vertexColorEnabled
	basic.world = cloneSource.world
	basic.view = cloneSource.view
	basic.projection = cloneSource.projection
	basic.diffuseColor = cloneSource.diffuseColor
	basic.emissiveColor = cloneSource.emissiveColor
	basic.ambientLightColor = cloneSource.ambientLightColor
	basic.alpha = cloneSource.alpha
	basic.fogStart = cloneSource.fogStart
	basic.fogEnd = cloneSource.fogEnd
	// The texture is the one value the reference reads back out of the cloned
	// effect rather than copying. Here it is copied, for the reason recorded at
	// Texture(): the projection holds the managed object the setter was given,
	// and CNA's clone carries the native half.
	basic.texture = cloneSource.texture
	return basic, nil
}

// newBasicEffectState is the block of field initialisers both constructors run
// before they chain to the base.
func newBasicEffectState() *BasicEffect {
	return &BasicEffect{
		world:             framework.MatrixIdentity(),
		view:              framework.MatrixIdentity(),
		projection:        framework.MatrixIdentity(),
		diffuseColor:      framework.Vector3One(),
		emissiveColor:     framework.Vector3Zero(),
		ambientLightColor: framework.Vector3Zero(),
		alpha:             1,
		fogEnd:            1,
		dirtyFlags:        effectDirtyAll,
	}
}

// cacheEffectParameters is BasicEffect::CacheEffectParameters, whose 478 bytes
// are twelve lookups by name and three DirectionalLight constructions.
//
// The twelve lookups have nothing to find: CNA publishes no parameters, so the
// twelve fields would all be null and every write-through the reference does
// through them would be skipped by its own `brfalse` guards. The three light
// constructions DO have a counterpart, because CNA models the lights
// themselves, so this is what survives of the member.
func (e *BasicEffect) cacheEffectParameters(cloneSource *BasicEffect) error {
	resource := e.effect.nativeResource()
	lights := [3]**DirectionalLight{&e.light0, &e.light1, &e.light2}
	var sources [3]*DirectionalLight
	if cloneSource != nil {
		sources = [3]*DirectionalLight{cloneSource.light0, cloneSource.light1, cloneSource.light2}
	}
	for index, target := range lights {
		view, err := resource.EffectLightsDirectionalLight(uint32(index))
		if err != nil {
			return err
		}
		light, err := newPublishedDirectionalLight(view, sources[index])
		if err != nil {
			return err
		}
		*target = light
	}
	return nil
}

// clrTypeName is System.Object::ToString's answer for a BasicEffect, which is
// what GraphicsResource::ToString falls back to for an unnamed one.
func (e *BasicEffect) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.BasicEffect" }

func (e *BasicEffect) nativeResource() *interop.Resource {
	if e == nil || e.effect == nil {
		return nil
	}
	return e.effect.nativeResource()
}

// ---------------------------------------------------------------------------
// IEffectMatrices. Three `ldfld` getters and three `stfld` setters, each with
// the exact dirty-flag word the IL ORs in.
// ---------------------------------------------------------------------------

// World is BasicEffect::get_World.
func (e *BasicEffect) World() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.world
}

// SetWorld is BasicEffect::set_World, which raises World, WorldViewProj and
// Fog -- the fog vector is computed from the world-view, so it is stale too.
func (e *BasicEffect) SetWorld(value framework.Matrix) {
	if e == nil {
		return
	}
	e.world = value
	e.dirtyFlags |= effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyFog
}

// View is BasicEffect::get_View.
func (e *BasicEffect) View() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.view
}

// SetView is BasicEffect::set_View, which raises WorldViewProj, EyePosition and
// Fog: the eye position is the view's inverse translation.
func (e *BasicEffect) SetView(value framework.Matrix) {
	if e == nil {
		return
	}
	e.view = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyEyePosition | effectDirtyFog
}

// Projection is BasicEffect::get_Projection.
func (e *BasicEffect) Projection() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.projection
}

// SetProjection is BasicEffect::set_Projection, which raises WorldViewProj
// alone -- 22 bytes against the other two setters' 23, because it ORs one flag
// fewer.
func (e *BasicEffect) SetProjection(value framework.Matrix) {
	if e == nil {
		return
	}
	e.projection = value
	e.dirtyFlags |= effectDirtyWorldViewProj
}

// ---------------------------------------------------------------------------
// IEffectFog.
// ---------------------------------------------------------------------------

// FogEnabled is BasicEffect::get_FogEnabled.
func (e *BasicEffect) FogEnabled() bool {
	if e == nil {
		return false
	}
	return e.fogEnabled
}

// SetFogEnabled is BasicEffect::set_FogEnabled, one of the five setters whose
// 35 bytes are a `beq` early return: assigning the value it already holds
// raises no flag at all. It selects a shader permutation, so it raises
// ShaderIndex and FogEnable.
func (e *BasicEffect) SetFogEnabled(value bool) {
	if e == nil || e.fogEnabled == value {
		return
	}
	e.fogEnabled = value
	e.dirtyFlags |= effectDirtyFogEnable | effectDirtyShaderIndex
}

// FogStart is BasicEffect::get_FogStart.
func (e *BasicEffect) FogStart() float32 {
	if e == nil {
		return 0
	}
	return e.fogStart
}

// SetFogStart is BasicEffect::set_FogStart.
func (e *BasicEffect) SetFogStart(value float32) {
	if e == nil {
		return
	}
	e.fogStart = value
	e.dirtyFlags |= effectDirtyFog
}

// FogEnd is BasicEffect::get_FogEnd.
func (e *BasicEffect) FogEnd() float32 {
	if e == nil {
		return 0
	}
	return e.fogEnd
}

// SetFogEnd is BasicEffect::set_FogEnd.
func (e *BasicEffect) SetFogEnd(value float32) {
	if e == nil {
		return
	}
	e.fogEnd = value
	e.dirtyFlags |= effectDirtyFog
}

// FogColor is BasicEffect::get_FogColor -- `fogColorParam.GetValueVector3()`,
// one of the four properties that really do cross into the effect, and one of
// the four that are therefore fallible.
func (e *BasicEffect) FogColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errBasicEffectNil
	}
	values, err := resource.EffectFogColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetFogColor is BasicEffect::set_FogColor -- `fogColorParam.SetValue(value)`,
// with no cache and no dirty flag, because the effect itself is the store.
func (e *BasicEffect) SetFogColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errBasicEffectNil
	}
	return resource.EffectFogSetColor(vector3Triple(value))
}

// ---------------------------------------------------------------------------
// IEffectLights.
// ---------------------------------------------------------------------------

// LightingEnabled is BasicEffect::get_LightingEnabled.
func (e *BasicEffect) LightingEnabled() bool {
	if e == nil {
		return false
	}
	return e.lightingEnabled
}

// SetLightingEnabled is BasicEffect::set_LightingEnabled: an early return, then
// MaterialColor and ShaderIndex. MaterialColor is in the set because the
// material push multiplies by alpha differently depending on whether lighting
// is on.
func (e *BasicEffect) SetLightingEnabled(value bool) {
	if e == nil || e.lightingEnabled == value {
		return
	}
	e.lightingEnabled = value
	e.dirtyFlags |= effectDirtyMaterialColor | effectDirtyShaderIndex
}

// PreferPerPixelLighting is BasicEffect::get_PreferPerPixelLighting.
func (e *BasicEffect) PreferPerPixelLighting() bool {
	if e == nil {
		return false
	}
	return e.preferPerPixelLighting
}

// SetPreferPerPixelLighting is BasicEffect::set_PreferPerPixelLighting, a
// PREFERENCE: it selects a shader permutation and the reference reports back
// what it STORED, never what the device could do.
func (e *BasicEffect) SetPreferPerPixelLighting(value bool) {
	if e == nil || e.preferPerPixelLighting == value {
		return
	}
	e.preferPerPixelLighting = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// AmbientLightColor is BasicEffect::get_AmbientLightColor.
func (e *BasicEffect) AmbientLightColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.ambientLightColor
}

// SetAmbientLightColor is BasicEffect::set_AmbientLightColor.
func (e *BasicEffect) SetAmbientLightColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.ambientLightColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// DirectionalLight0 is BasicEffect::get_DirectionalLight0, one `ldfld`: the
// same object on every call.
func (e *BasicEffect) DirectionalLight0() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light0
}

// DirectionalLight1 is BasicEffect::get_DirectionalLight1.
func (e *BasicEffect) DirectionalLight1() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light1
}

// DirectionalLight2 is BasicEffect::get_DirectionalLight2.
func (e *BasicEffect) DirectionalLight2() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light2
}

// EnableDefaultLighting is BasicEffect::EnableDefaultLighting:
//
//	LightingEnabled = true;
//	AmbientLightColor = EffectHelpers.EnableDefaultLighting(light0, light1, light2);
//
// The rig EffectHelpers applies is a measured constant, read straight from the
// helper's IL, and it is reproduced here rather than delegated to
// `cna_effect_lights_enable_default`. CNA's preset is CNA's; making it answer
// for XNA's would turn a native default into an XNA behaviour golden, which the
// authority rule forbids. That route is recorded as deliberately unbound under
// CONTRACT_DIVERGENCE.
//
// It is fallible because the light setters are: each writes through to the
// light CNA owns.
func (e *BasicEffect) EnableDefaultLighting() error {
	if e == nil {
		return errBasicEffectNil
	}
	e.SetLightingEnabled(true)
	ambient, err := effectHelpersEnableDefaultLighting(e.light0, e.light1, e.light2)
	if err != nil {
		return err
	}
	e.SetAmbientLightColor(ambient)
	return nil
}

// effectHelpersEnableDefaultLighting is
// EffectHelpers::EnableDefaultLighting(DirectionalLight, DirectionalLight,
// DirectionalLight), an `assembly` static whose 261 bytes are thirteen calls
// with `ldc.r4` operands and one returned Vector3. Every constant below is that
// operand, exactly as the IL spells it.
//
// The order matters and is preserved: for each light Direction, DiffuseColor,
// SpecularColor and only THEN Enabled. Because the two colour setters are
// guarded by `enabled` and each light starts disabled, the colour writes reach
// the cache and not the native light; enabling the light afterwards is what
// publishes them.
func effectHelpersEnableDefaultLighting(light0, light1, light2 *DirectionalLight) (framework.Vector3, error) {
	steps := []struct {
		light                        *DirectionalLight
		direction, diffuse, specular framework.Vector3
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
	}
	for _, step := range steps {
		if err := step.light.SetDirection(step.direction); err != nil {
			return framework.Vector3{}, err
		}
		if err := step.light.SetDiffuseColor(step.diffuse); err != nil {
			return framework.Vector3{}, err
		}
		if err := step.light.SetSpecularColor(step.specular); err != nil {
			return framework.Vector3{}, err
		}
		if err := step.light.SetEnabled(true); err != nil {
			return framework.Vector3{}, err
		}
	}
	return framework.NewVector3BySingleAndSingleAndSingle(0.05333332, 0.09882354, 0.1819608), nil
}

// ---------------------------------------------------------------------------
// BasicEffect's own material surface.
// ---------------------------------------------------------------------------

// DiffuseColor is BasicEffect::get_DiffuseColor, an `ldfld` of the field the
// setter stored -- NOT the alpha-premultiplied value OnApply pushes.
func (e *BasicEffect) DiffuseColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.diffuseColor
}

// SetDiffuseColor is BasicEffect::set_DiffuseColor.
func (e *BasicEffect) SetDiffuseColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.diffuseColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// EmissiveColor is BasicEffect::get_EmissiveColor.
func (e *BasicEffect) EmissiveColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.emissiveColor
}

// SetEmissiveColor is BasicEffect::set_EmissiveColor.
func (e *BasicEffect) SetEmissiveColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.emissiveColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// SpecularColor is BasicEffect::get_SpecularColor -- 12 bytes, a parameter read
// and no cache at all, which is why it is fallible and why it answers whatever
// the effect stored rather than what the caller last passed.
func (e *BasicEffect) SpecularColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errBasicEffectNil
	}
	values, err := resource.BasicEffectSpecularColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetSpecularColor is BasicEffect::set_SpecularColor -- 13 bytes, a parameter
// write and no dirty flag.
func (e *BasicEffect) SetSpecularColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errBasicEffectNil
	}
	return resource.BasicEffectSetSpecularColor(vector3Triple(value))
}

// SpecularPower is BasicEffect::get_SpecularPower, the same shape one scalar
// down.
func (e *BasicEffect) SpecularPower() (float32, error) {
	resource := e.nativeResource()
	if resource == nil {
		return 0, errBasicEffectNil
	}
	return resource.BasicEffectSpecularPower()
}

// SetSpecularPower is BasicEffect::set_SpecularPower.
func (e *BasicEffect) SetSpecularPower(value float32) error {
	resource := e.nativeResource()
	if resource == nil {
		return errBasicEffectNil
	}
	return resource.BasicEffectSetSpecularPower(value)
}

// Alpha is BasicEffect::get_Alpha.
func (e *BasicEffect) Alpha() float32 {
	if e == nil {
		return 0
	}
	return e.alpha
}

// SetAlpha is BasicEffect::set_Alpha, which raises MaterialColor because every
// pushed material colour is multiplied by it.
func (e *BasicEffect) SetAlpha(value float32) {
	if e == nil {
		return
	}
	e.alpha = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// VertexColorEnabled is BasicEffect::get_VertexColorEnabled.
func (e *BasicEffect) VertexColorEnabled() bool {
	if e == nil {
		return false
	}
	return e.vertexColorEnabled
}

// SetVertexColorEnabled is BasicEffect::set_VertexColorEnabled, a shader
// permutation and nothing else.
func (e *BasicEffect) SetVertexColorEnabled(value bool) {
	if e == nil || e.vertexColorEnabled == value {
		return
	}
	e.vertexColorEnabled = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// TextureEnabled is BasicEffect::get_TextureEnabled.
func (e *BasicEffect) TextureEnabled() bool {
	if e == nil {
		return false
	}
	return e.textureEnabled
}

// SetTextureEnabled is BasicEffect::set_TextureEnabled, also a permutation.
func (e *BasicEffect) SetTextureEnabled(value bool) {
	if e == nil || e.textureEnabled == value {
		return
	}
	e.textureEnabled = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// Texture is BasicEffect::get_Texture -- `textureParam.GetValueTexture2D()`,
// declared fallible because the reference's crosses into D3DX.
//
// # Why this one answers a MANAGED field
//
// Foundation 72 recorded that CNA reports an effect texture as a HANDLE and not
// as an object, so `EffectParameter::GetValueTexture2D` cannot hand back the
// Texture2D the setter was given and refuses instead: rebuilding a facade over
// the handle would make `p.GetValueTexture2D() == myTexture` silently false.
//
// The same obstacle stands here and the same answer would be wrong, because the
// two members are not in the same position. An EffectParameter is a VIEW of
// state anything holding another view can write, so a cache in the view could
// go stale. BasicEffect::Texture is a property of the effect and nothing else
// can write it -- CNA publishes no parameters at all -- so the object the
// setter was given IS the current value, and holding it reproduces the
// reference's observable exactly where refusing would not.
//
// `cna_basic_effect_get_texture` is therefore recorded as deliberately unbound
// under REPRESENTATION, and this getter cannot in fact fail; the error result
// is the contract's, not a prediction.
func (e *BasicEffect) Texture() (*Texture2D, error) {
	if e == nil {
		return nil, errBasicEffectNil
	}
	return e.texture, nil
}

// SetTexture is BasicEffect::set_Texture. The parameter is a
// Texture2DReference, because the substitutable-base rule widens a Texture2D
// PARAMETER position and a property setter is one: `basicEffect.Texture =
// myRenderTarget` compiles in C#, and RenderTarget2D is a projected type.
//
// A null texture crosses as the invalid handle, which CNA documents as
// selecting no texture.
func (e *BasicEffect) SetTexture(value Texture2DReference) error {
	resource := e.nativeResource()
	if resource == nil {
		return errBasicEffectNil
	}
	texture := resolveTexture2D(value)
	if err := resource.BasicEffectSetTexture(texture.nativeResource()); err != nil {
		return err
	}
	e.texture = texture
	return nil
}

// ---------------------------------------------------------------------------
// The two virtual members, and the push.
// ---------------------------------------------------------------------------

// Clone is BasicEffect::Clone, seven bytes: `newobj BasicEffect::.ctor(BasicEffect)`.
//
// It returns EffectReference rather than *Effect for the reason recorded on
// that interface: the value really is a BasicEffect and the downcast a C#
// consumer writes is the whole point of the member.
func (e *BasicEffect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errBasicEffectNil
	}
	return NewBasicEffectByBasicEffect(e)
}

// OnApply is BasicEffect::OnApply, the `famorassem virtual` hook EffectPass::Apply
// calls before it begins the pass.
//
// # What the reference's 388 bytes do, and what survives here
//
// The reference computes the DERIVED quantities and writes them into the
// parameters: SetWorldViewProjAndFog builds world*view*projection and the fog
// vector, SetMaterialColor premultiplies the diffuse and emissive colours by
// alpha, SetLightingMatrices writes the world, its inverse transpose and the
// eye position, and the tail computes a shader index from five booleans.
//
// CNA derives all of that itself from the raw state, so what this pushes is the
// raw state -- and the dirty-flag word decides which parts of it, exactly as it
// decides which parameters the reference writes. The one piece of the
// reference's own arithmetic that IS reproduced is `oneLight`, because it is
// state the reference keeps and its transition raises ShaderIndex.
func (e *BasicEffect) OnApply() error {
	if e == nil {
		return errBasicEffectNil
	}
	resource := e.nativeResource()
	if resource == nil {
		return errBasicEffectNil
	}
	if e.dirtyFlags&(effectDirtyWorld|effectDirtyWorldViewProj) != 0 {
		for _, write := range []struct {
			value framework.Matrix
			apply func([16]float32) error
		}{
			{e.world, resource.EffectMatricesSetWorld},
			{e.view, resource.EffectMatricesSetView},
			{e.projection, resource.EffectMatricesSetProjection},
		} {
			if err := write.apply(matrixToRowMajor(write.value)); err != nil {
				return err
			}
		}
		e.dirtyFlags &^= effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyEyePosition
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
	// The reference clears MaterialColor here with `and -9` and nothing else,
	// which is why the clear below is one flag rather than the group above's.
	if e.dirtyFlags&effectDirtyMaterialColor != 0 {
		if err := resource.BasicEffectSetDiffuseColor(vector3Triple(e.diffuseColor)); err != nil {
			return err
		}
		if err := resource.BasicEffectSetEmissiveColor(vector3Triple(e.emissiveColor)); err != nil {
			return err
		}
		if err := resource.BasicEffectSetAlpha(e.alpha); err != nil {
			return err
		}
		if err := resource.EffectLightsSetAmbientColor(vector3Triple(e.ambientLightColor)); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyMaterialColor
	}
	e.recomputeOneLight()
	if e.dirtyFlags&effectDirtyShaderIndex != 0 {
		for _, write := range []struct {
			value bool
			apply func(bool) error
		}{
			{e.lightingEnabled, resource.EffectLightsSetEnabled},
			{e.preferPerPixelLighting, resource.BasicEffectSetPreferPerPixelLighting},
			{e.vertexColorEnabled, resource.BasicEffectSetVertexColorEnabled},
			{e.textureEnabled, resource.BasicEffectSetTextureEnabled},
		} {
			if err := write.apply(write.value); err != nil {
				return err
			}
		}
		e.dirtyFlags &^= effectDirtyShaderIndex
	}
	return nil
}

// recomputeOneLight is the one piece of OnApply's own arithmetic the projection
// reproduces, because it is STATE the reference keeps rather than a value it
// derives for a parameter:
//
//	if (lightingEnabled) {
//	    bool newOneLight = !light1.Enabled && !light2.Enabled;
//	    if (oneLight != newOneLight) { oneLight = newOneLight; dirtyFlags |= ShaderIndex; }
//	}
//
// Two details a reader would otherwise guess wrong. The recomputation is
// bracketed by `lightingEnabled`, so `oneLight` is frozen while lighting is
// off. And the flag follows the TRANSITION, not the value: a second pass with
// nothing changed raises nothing.
//
// It is a member of its own so the managed claim can be measured without a
// device, which is the whole of what this projection adds to OnApply.
func (e *BasicEffect) recomputeOneLight() {
	if !e.lightingEnabled {
		return
	}
	oneLight := !e.light1.Enabled() && !e.light2.Enabled()
	if e.oneLight != oneLight {
		e.oneLight = oneLight
		e.dirtyFlags |= effectDirtyShaderIndex
	}
}

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *BasicEffect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *BasicEffect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *BasicEffect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *BasicEffect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errBasicEffectNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *BasicEffect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *BasicEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *BasicEffect) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *BasicEffect) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *BasicEffect) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *BasicEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString, which falls back to the RUNTIME
// type's name -- BasicEffect's, not Effect's, which is what bindDerived
// installs.
func (e *BasicEffect) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *BasicEffect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errBasicEffectNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *BasicEffect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errBasicEffectNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), and it is BasicEffect's ONLY dispose
// member -- unlike Effect's pair.
//
// The reason is the contract's, not a choice: Effect DECLARES the protected
// `Dispose(Boolean)` override and so projects it, while BasicEffect declares no
// Dispose at all. A protected member is projected on the type that declares it,
// and the inherited surface a derived type re-exposes is the PUBLIC surface, so
// BasicEffect inherits one Dispose and it takes no argument.
//
// What BasicEffect does have to release is the three light VIEWS, which are
// handles this object owns and which the reference has no counterpart for
// because its lights are managed objects. That release runs from the base's
// disposal through releaseDerivedNativeObjects rather than from a public
// override the contract does not declare.
func (e *BasicEffect) Dispose() error {
	if e == nil {
		return errBasicEffectNil
	}
	return e.effect.DisposeByNone()
}

// releaseDerivedNativeObjects is the derived half of Effect's disposal: the
// three light views, released before the effect behind them goes.
func (e *BasicEffect) releaseDerivedNativeObjects() error {
	if e == nil {
		return nil
	}
	return errors.Join(e.light0.dispose(), e.light1.dispose(), e.light2.dispose())
}

// The compiler is the proof that BasicEffect satisfies the three contracts the
// pinned metadata says it implements.
var (
	_ IEffectMatrices = (*BasicEffect)(nil)
	_ IEffectFog      = (*BasicEffect)(nil)
	_ IEffectLights   = (*BasicEffect)(nil)
)
