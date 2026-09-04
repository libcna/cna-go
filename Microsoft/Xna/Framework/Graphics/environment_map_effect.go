package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// EnvironmentMapEffect is the stock effect for cube-map reflection: a base
// texture blended with an environment map, lit and fogged.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
//	.class public auto ansi beforefieldinit EnvironmentMapEffect
//	       extends Effect
//	       implements IEffectMatrices, IEffectLights, IEffectFog
//
// # Six of its twenty members cross, which is the most of any stock effect
//
//	FogColor                fogColorParam
//	Texture                 textureParam
//	EnvironmentMap          environmentMapParam        (a TextureCube)
//	EnvironmentMapAmount    environmentMapAmountParam
//	EnvironmentMapSpecular  environmentMapSpecularParam
//	FresnelFactor           fresnelFactorParam
//
// The last two are 59-byte setters rather than 13, and the extra bytes are the
// detail that decides a shader permutation: each writes its parameter AND
// stores a managed flag from the value it was given.
//
//	set_EnvironmentMapSpecular:  specularEnabled = value != Vector3.Zero
//	set_FresnelFactor:           fresnelEnabled  = value != 0
//
// Both then raise ShaderIndex. So the two are fallible -- they write a
// parameter -- and they also mutate managed state, which no other crossing
// accessor in the family does.
//
// # LightingEnabled is EXPLICIT, and the type has no such member
//
//	.method private ... instance bool
//	        Microsoft.Xna.Framework.Graphics.IEffectLights.get_LightingEnabled()
//	  IL_0000: ldc.i4.1
//	  IL_0001: ret
//
// Two bytes: lighting is ALWAYS on. The setter is 51 bytes and throws
// NotSupportedException for false. Because both are explicit implementations,
// the pinned contract lists NO LightingEnabled property on this type -- a
// consumer reaches it only through the interface, which is exactly what an
// explicit implementation means in C#.
//
// Go has no explicit interface implementation. The settled projection is the
// interface-witness rule: the members exist on the type so the interface can be
// satisfied, and the verifier is told they are witnesses rather than declared
// surface.
type EnvironmentMapEffect struct {
	effect *Effect

	light0, light1, light2 *DirectionalLight

	world, view, projection framework.Matrix
	diffuseColor            framework.Vector3
	emissiveColor           framework.Vector3
	ambientLightColor       framework.Vector3
	alpha                   float32
	fogEnabled              bool
	fogStart, fogEnd        float32
	oneLight                bool
	fresnelEnabled          bool
	specularEnabled         bool
	dirtyFlags              effectDirtyFlags

	texture        *Texture2D
	environmentMap *TextureCube
}

// errEnvironmentMapEffectNil is the Go-only guard for a zero value.
var errEnvironmentMapEffectNil = errors.New("environment map effect is nil or uninitialized")

// cantDisableLighting is FrameworkResources.CantDisableLighting, read from
// Microsoft.Xna.Framework.dll. String.Format's {0} carries the effect's class
// NAME -- typeof(T).Name, so the short name and not the namespaced one.
const cantDisableLighting = "%s does not support setting LightingEnabled to false."

// NewEnvironmentMapEffectByGraphicsDevice is
// EnvironmentMapEffect::.ctor(GraphicsDevice), 160 bytes:
//
//	world = view = projection = Matrix.Identity;
//	diffuseColor = Vector3.One;
//	emissiveColor = ambientLightColor = Vector3.Zero;
//	alpha = 1;  fogEnd = 1;  dirtyFlags = -1;
//	base(device, EnvironmentMapEffectCode.Code);
//	CacheEffectParameters(null);
func NewEnvironmentMapEffectByGraphicsDevice(device *GraphicsDevice) (*EnvironmentMapEffect, error) {
	if device == nil || device.device == nil {
		return nil, fmt.Errorf("%w: device: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := device.device.CreateEnvironmentMapEffect()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(device, resource)
	if err != nil {
		return nil, err
	}
	environment := newEnvironmentMapEffectState()
	environment.effect = effect
	effect.bindDerived(environment)
	effect.bindDerivedEffect(environment)
	if err := environment.cacheEffectParameters(nil); err != nil {
		return nil, err
	}
	return environment, nil
}

// NewEnvironmentMapEffectByEnvironmentMapEffect is the PROTECTED clone
// constructor. It copies TWELVE values: fogEnabled, fresnelEnabled,
// specularEnabled, the three matrices, diffuseColor, emissiveColor,
// ambientLightColor, alpha, fogStart and fogEnd.
//
// The two ENABLED flags are copied and the values behind them are not, because
// those live in the cloned effect -- which is why the flags are fields at all.
func NewEnvironmentMapEffectByEnvironmentMapEffect(cloneSource *EnvironmentMapEffect) (*EnvironmentMapEffect, error) {
	if cloneSource == nil || cloneSource.effect == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned, err := cloneSource.effect.cloneBase()
	if err != nil {
		return nil, err
	}
	environment := newEnvironmentMapEffectState()
	environment.effect = cloned
	cloned.bindDerived(environment)
	cloned.bindDerivedEffect(environment)
	if err := environment.cacheEffectParameters(cloneSource); err != nil {
		return nil, err
	}
	environment.fogEnabled = cloneSource.fogEnabled
	environment.fresnelEnabled = cloneSource.fresnelEnabled
	environment.specularEnabled = cloneSource.specularEnabled
	environment.world = cloneSource.world
	environment.view = cloneSource.view
	environment.projection = cloneSource.projection
	environment.diffuseColor = cloneSource.diffuseColor
	environment.emissiveColor = cloneSource.emissiveColor
	environment.ambientLightColor = cloneSource.ambientLightColor
	environment.alpha = cloneSource.alpha
	environment.fogStart = cloneSource.fogStart
	environment.fogEnd = cloneSource.fogEnd
	environment.texture = cloneSource.texture
	environment.environmentMap = cloneSource.environmentMap
	return environment, nil
}

func newEnvironmentMapEffectState() *EnvironmentMapEffect {
	return &EnvironmentMapEffect{
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

// cacheEffectParameters is the type's CacheEffectParameters, of which the three
// DirectionalLight constructions are what survives -- the fourteen parameter
// lookups have nothing to find, as Foundation 79 measured.
func (e *EnvironmentMapEffect) cacheEffectParameters(cloneSource *EnvironmentMapEffect) error {
	var sources [3]*DirectionalLight
	if cloneSource != nil {
		sources = [3]*DirectionalLight{cloneSource.light0, cloneSource.light1, cloneSource.light2}
	}
	lights, err := publishDirectionalLights(e.effect, sources)
	if err != nil {
		return err
	}
	e.light0, e.light1, e.light2 = lights[0], lights[1], lights[2]
	return nil
}

func (e *EnvironmentMapEffect) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect"
}

func (e *EnvironmentMapEffect) nativeResource() *interop.Resource {
	if e == nil || e.effect == nil {
		return nil
	}
	return e.effect.nativeResource()
}

// ---------------------------------------------------------------------------
// IEffectMatrices. This effect IS lit, so its two matrix setters raise
// BasicEffect's three-flag words rather than the unlit pair.
// ---------------------------------------------------------------------------

// World is EnvironmentMapEffect::get_World.
func (e *EnvironmentMapEffect) World() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.world
}

// SetWorld is EnvironmentMapEffect::set_World.
func (e *EnvironmentMapEffect) SetWorld(value framework.Matrix) {
	if e == nil {
		return
	}
	e.world = value
	e.dirtyFlags |= effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyFog
}

// View is EnvironmentMapEffect::get_View.
func (e *EnvironmentMapEffect) View() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.view
}

// SetView is EnvironmentMapEffect::set_View.
func (e *EnvironmentMapEffect) SetView(value framework.Matrix) {
	if e == nil {
		return
	}
	e.view = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyEyePosition | effectDirtyFog
}

// Projection is EnvironmentMapEffect::get_Projection.
func (e *EnvironmentMapEffect) Projection() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.projection
}

// SetProjection is EnvironmentMapEffect::set_Projection.
func (e *EnvironmentMapEffect) SetProjection(value framework.Matrix) {
	if e == nil {
		return
	}
	e.projection = value
	e.dirtyFlags |= effectDirtyWorldViewProj
}

// ---------------------------------------------------------------------------
// IEffectFog.
// ---------------------------------------------------------------------------

// FogEnabled is EnvironmentMapEffect::get_FogEnabled.
func (e *EnvironmentMapEffect) FogEnabled() bool {
	if e == nil {
		return false
	}
	return e.fogEnabled
}

// SetFogEnabled is EnvironmentMapEffect::set_FogEnabled.
func (e *EnvironmentMapEffect) SetFogEnabled(value bool) {
	if e == nil || e.fogEnabled == value {
		return
	}
	e.fogEnabled = value
	e.dirtyFlags |= effectDirtyFogEnable | effectDirtyShaderIndex
}

// FogStart is EnvironmentMapEffect::get_FogStart.
func (e *EnvironmentMapEffect) FogStart() float32 {
	if e == nil {
		return 0
	}
	return e.fogStart
}

// SetFogStart is EnvironmentMapEffect::set_FogStart.
func (e *EnvironmentMapEffect) SetFogStart(value float32) {
	if e == nil {
		return
	}
	e.fogStart = value
	e.dirtyFlags |= effectDirtyFog
}

// FogEnd is EnvironmentMapEffect::get_FogEnd.
func (e *EnvironmentMapEffect) FogEnd() float32 {
	if e == nil {
		return 0
	}
	return e.fogEnd
}

// SetFogEnd is EnvironmentMapEffect::set_FogEnd.
func (e *EnvironmentMapEffect) SetFogEnd(value float32) {
	if e == nil {
		return
	}
	e.fogEnd = value
	e.dirtyFlags |= effectDirtyFog
}

// FogColor is EnvironmentMapEffect::get_FogColor.
func (e *EnvironmentMapEffect) FogColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errEnvironmentMapEffectNil
	}
	values, err := resource.EffectFogColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetFogColor is EnvironmentMapEffect::set_FogColor.
func (e *EnvironmentMapEffect) SetFogColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	return resource.EffectFogSetColor(vector3Triple(value))
}

// ---------------------------------------------------------------------------
// IEffectLights. LightingEnabled is the WITNESS pair; the rest are declared.
// ---------------------------------------------------------------------------

// LightingEnabled is IEffectLights::LightingEnabled's getter, implemented
// EXPLICITLY by this type as `ldc.i4.1; ret` -- two bytes, always true.
//
// It is an interface witness rather than declared surface: the pinned contract
// lists no LightingEnabled property on EnvironmentMapEffect, because an
// explicit implementation is reachable only through the interface in C#. Go has
// no such distinction, so the member exists and the verifier is told what it is.
func (e *EnvironmentMapEffect) LightingEnabled() bool { return true }

// SetLightingEnabled is the explicit setter, 51 bytes:
//
//	if (!value) throw new NotSupportedException(string.Format(
//	    CultureInfo.CurrentCulture, FrameworkResources.CantDisableLighting,
//	    typeof(EnvironmentMapEffect).Name));
//
// So `true` is a no-op and `false` is a refusal. It is INFALLIBLE in the
// interface's signature, because IEffectLights' measured contract says so, and
// the refusal it must carry has nowhere to go -- which is recorded here rather
// than solved by widening a signature the interface pins.
//
// The projection therefore answers the way the reference does for the value
// that is legal and DROPS the other, and this is the one place in the stock
// effects where a projected member cannot report a refusal the reference makes.
func (e *EnvironmentMapEffect) SetLightingEnabled(value bool) {
	if value {
		return
	}
	// The message the reference formats, kept as evidence that the refusal was
	// measured rather than forgotten. Go has no way to raise it from an
	// infallible setter.
	_ = fmt.Sprintf(cantDisableLighting, "EnvironmentMapEffect")
}

// AmbientLightColor is EnvironmentMapEffect::get_AmbientLightColor.
func (e *EnvironmentMapEffect) AmbientLightColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.ambientLightColor
}

// SetAmbientLightColor is EnvironmentMapEffect::set_AmbientLightColor.
func (e *EnvironmentMapEffect) SetAmbientLightColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.ambientLightColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// DirectionalLight0 is EnvironmentMapEffect::get_DirectionalLight0.
func (e *EnvironmentMapEffect) DirectionalLight0() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light0
}

// DirectionalLight1 is EnvironmentMapEffect::get_DirectionalLight1.
func (e *EnvironmentMapEffect) DirectionalLight1() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light1
}

// DirectionalLight2 is EnvironmentMapEffect::get_DirectionalLight2.
func (e *EnvironmentMapEffect) DirectionalLight2() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light2
}

// EnableDefaultLighting is EnvironmentMapEffect::EnableDefaultLighting, 30
// bytes -- BasicEffect's body WITHOUT the LightingEnabled store, because
// lighting is always on here.
func (e *EnvironmentMapEffect) EnableDefaultLighting() error {
	if e == nil {
		return errEnvironmentMapEffectNil
	}
	ambient, err := effectHelpersEnableDefaultLighting(e.light0, e.light1, e.light2)
	if err != nil {
		return err
	}
	e.SetAmbientLightColor(ambient)
	return nil
}

// ---------------------------------------------------------------------------
// EnvironmentMapEffect's own surface.
// ---------------------------------------------------------------------------

// DiffuseColor is EnvironmentMapEffect::get_DiffuseColor.
func (e *EnvironmentMapEffect) DiffuseColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.diffuseColor
}

// SetDiffuseColor is EnvironmentMapEffect::set_DiffuseColor.
func (e *EnvironmentMapEffect) SetDiffuseColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.diffuseColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// EmissiveColor is EnvironmentMapEffect::get_EmissiveColor.
func (e *EnvironmentMapEffect) EmissiveColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.emissiveColor
}

// SetEmissiveColor is EnvironmentMapEffect::set_EmissiveColor.
func (e *EnvironmentMapEffect) SetEmissiveColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.emissiveColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// Alpha is EnvironmentMapEffect::get_Alpha.
func (e *EnvironmentMapEffect) Alpha() float32 {
	if e == nil {
		return 0
	}
	return e.alpha
}

// SetAlpha is EnvironmentMapEffect::set_Alpha.
func (e *EnvironmentMapEffect) SetAlpha(value float32) {
	if e == nil {
		return
	}
	e.alpha = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// EnvironmentMapAmount is EnvironmentMapEffect::get_EnvironmentMapAmount, a
// parameter read.
func (e *EnvironmentMapEffect) EnvironmentMapAmount() (float32, error) {
	resource := e.nativeResource()
	if resource == nil {
		return 0, errEnvironmentMapEffectNil
	}
	return resource.EnvironmentMapEffectAmount()
}

// SetEnvironmentMapAmount is
// EnvironmentMapEffect::set_EnvironmentMapAmount, a parameter write with no
// cache and no dirty flag.
func (e *EnvironmentMapEffect) SetEnvironmentMapAmount(value float32) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	return resource.EnvironmentMapEffectSetAmount(value)
}

// EnvironmentMapSpecular is
// EnvironmentMapEffect::get_EnvironmentMapSpecular, a parameter read.
func (e *EnvironmentMapEffect) EnvironmentMapSpecular() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errEnvironmentMapEffectNil
	}
	values, err := resource.EnvironmentMapEffectSpecular()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetEnvironmentMapSpecular is one of the two 59-byte setters: it writes the
// parameter AND stores `specularEnabled = value != Vector3.Zero`, then raises
// ShaderIndex. The flag decides a shader permutation, so a projection that only
// wrote the parameter would draw the wrong thing.
func (e *EnvironmentMapEffect) SetEnvironmentMapSpecular(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	if err := resource.EnvironmentMapEffectSetSpecular(vector3Triple(value)); err != nil {
		return err
	}
	e.specularEnabled = value != framework.Vector3Zero()
	e.dirtyFlags |= effectDirtyShaderIndex
	return nil
}

// FresnelFactor is EnvironmentMapEffect::get_FresnelFactor, a parameter read.
func (e *EnvironmentMapEffect) FresnelFactor() (float32, error) {
	resource := e.nativeResource()
	if resource == nil {
		return 0, errEnvironmentMapEffectNil
	}
	return resource.EnvironmentMapEffectFresnelFactor()
}

// SetFresnelFactor is the other 59-byte setter, with the scalar counterpart of
// the same flag: `fresnelEnabled = value != 0`.
func (e *EnvironmentMapEffect) SetFresnelFactor(value float32) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	if err := resource.EnvironmentMapEffectSetFresnelFactor(value); err != nil {
		return err
	}
	e.fresnelEnabled = value != 0
	e.dirtyFlags |= effectDirtyShaderIndex
	return nil
}

// Texture is EnvironmentMapEffect::get_Texture, the managed field.
func (e *EnvironmentMapEffect) Texture() (*Texture2D, error) {
	if e == nil {
		return nil, errEnvironmentMapEffectNil
	}
	return e.texture, nil
}

// SetTexture is EnvironmentMapEffect::set_Texture.
func (e *EnvironmentMapEffect) SetTexture(value Texture2DReference) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	texture := resolveTexture2D(value)
	if err := resource.EnvironmentMapEffectSetTexture(texture.nativeResource()); err != nil {
		return err
	}
	e.texture = texture
	return nil
}

// EnvironmentMap is EnvironmentMapEffect::get_EnvironmentMap, the one texture
// position in the stock effects whose value is a TextureCube.
func (e *EnvironmentMapEffect) EnvironmentMap() (*TextureCube, error) {
	if e == nil {
		return nil, errEnvironmentMapEffectNil
	}
	return e.environmentMap, nil
}

// SetEnvironmentMap is EnvironmentMapEffect::set_EnvironmentMap. Its parameter
// does NOT widen: TextureCube is a registered substitutable base and its
// derived type RenderTargetCube is projected, so the position takes the
// reference interface.
func (e *EnvironmentMapEffect) SetEnvironmentMap(value TextureCubeReference) error {
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	cube := resolveTextureCube(value)
	if err := resource.EnvironmentMapEffectSetEnvironmentMap(cube.nativeResource()); err != nil {
		return err
	}
	e.environmentMap = cube
	return nil
}

// ---------------------------------------------------------------------------
// The two virtual members, and the push.
// ---------------------------------------------------------------------------

// Clone is EnvironmentMapEffect::Clone.
func (e *EnvironmentMapEffect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errEnvironmentMapEffectNil
	}
	return NewEnvironmentMapEffectByEnvironmentMapEffect(e)
}

// OnApply is EnvironmentMapEffect::OnApply, 345 bytes.
func (e *EnvironmentMapEffect) OnApply() error {
	if e == nil {
		return errEnvironmentMapEffectNil
	}
	resource := e.nativeResource()
	if resource == nil {
		return errEnvironmentMapEffectNil
	}
	if e.dirtyFlags&(effectDirtyWorld|effectDirtyWorldViewProj) != 0 {
		if err := pushEffectMatrices(resource, e.world, e.view, e.projection); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyEyePosition
	}
	if err := pushEffectFog(resource, &e.dirtyFlags, e.fogEnabled, e.fogStart, e.fogEnd); err != nil {
		return err
	}
	if e.dirtyFlags&effectDirtyMaterialColor != 0 {
		if err := resource.EnvironmentMapEffectSetDiffuseColor(vector3Triple(e.diffuseColor)); err != nil {
			return err
		}
		if err := resource.EnvironmentMapEffectSetEmissiveColor(vector3Triple(e.emissiveColor)); err != nil {
			return err
		}
		if err := resource.EnvironmentMapEffectSetAlpha(e.alpha); err != nil {
			return err
		}
		if err := resource.EffectLightsSetAmbientColor(vector3Triple(e.ambientLightColor)); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyMaterialColor
	}
	// `oneLight` is recomputed unconditionally here, unlike BasicEffect's,
	// because lighting is always on for this effect.
	oneLight := !e.light1.Enabled() && !e.light2.Enabled()
	if e.oneLight != oneLight {
		e.oneLight = oneLight
		e.dirtyFlags |= effectDirtyShaderIndex
	}
	// The ShaderIndex branch CLEARS the flag and pushes nothing, and that is
	// not an omission. This effect's four permutation inputs -- fogEnabled,
	// fresnelEnabled, specularEnabled and oneLight -- all reach CNA already:
	// the fog enable through cna_effect_fog_set_enabled above, the two flags
	// through the setters that store them, and oneLight through the lights
	// themselves. The reference computes an index from them and writes it into
	// shaderIndexParam; CNA derives its own, so there is nothing left to send
	// and the bookkeeping is what remains.
	e.dirtyFlags &^= effectDirtyShaderIndex
	return nil
}

// releaseDerivedNativeObjects releases the three light views.
func (e *EnvironmentMapEffect) releaseDerivedNativeObjects() error {
	if e == nil {
		return nil
	}
	return errors.Join(e.light0.dispose(), e.light1.dispose(), e.light2.dispose())
}

// pushEffectFog is the fog half of every stock effect's OnApply, which IS
// identical across the family: the same two routes gated by the same flag, and
// the enable gated by its own.
func pushEffectFog(resource *interop.Resource, flags *effectDirtyFlags, enabled bool, start, end float32) error {
	if *flags&effectDirtyFog != 0 {
		if err := resource.EffectFogSetStart(start); err != nil {
			return err
		}
		if err := resource.EffectFogSetEnd(end); err != nil {
			return err
		}
		*flags &^= effectDirtyFog
	}
	if *flags&effectDirtyFogEnable != 0 {
		if err := resource.EffectFogSetEnabled(enabled); err != nil {
			return err
		}
		*flags &^= effectDirtyFogEnable
	}
	return nil
}

// publishDirectionalLights builds the three lights an effect owns, which every
// lit stock effect's CacheEffectParameters does the same way.
func publishDirectionalLights(effect *Effect, sources [3]*DirectionalLight) ([3]*DirectionalLight, error) {
	var lights [3]*DirectionalLight
	resource := effect.nativeResource()
	for index := range lights {
		view, err := resource.EffectLightsDirectionalLight(uint32(index))
		if err != nil {
			return lights, err
		}
		light, err := newPublishedDirectionalLight(view, sources[index])
		if err != nil {
			return lights, err
		}
		lights[index] = light
	}
	return lights, nil
}

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *EnvironmentMapEffect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *EnvironmentMapEffect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *EnvironmentMapEffect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *EnvironmentMapEffect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errEnvironmentMapEffectNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *EnvironmentMapEffect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *EnvironmentMapEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *EnvironmentMapEffect) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *EnvironmentMapEffect) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *EnvironmentMapEffect) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *EnvironmentMapEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (e *EnvironmentMapEffect) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *EnvironmentMapEffect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errEnvironmentMapEffectNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *EnvironmentMapEffect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errEnvironmentMapEffectNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), this type's only dispose member.
func (e *EnvironmentMapEffect) Dispose() error {
	if e == nil {
		return errEnvironmentMapEffectNil
	}
	return e.effect.DisposeByNone()
}

var (
	_ IEffectMatrices = (*EnvironmentMapEffect)(nil)
	_ IEffectFog      = (*EnvironmentMapEffect)(nil)
	_ IEffectLights   = (*EnvironmentMapEffect)(nil)
)
