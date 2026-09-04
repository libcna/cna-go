package graphics

import (
	"errors"
	"fmt"
	"strconv"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SkinnedEffect is the stock effect for skinned animation: a lit, textured
// material plus an array of bone transforms the vertex shader blends.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
//	.class public auto ansi beforefieldinit SkinnedEffect
//	       extends Effect
//	       implements IEffectMatrices, IEffectLights, IEffectFog
//
// # It is the only stock effect with a public FIELD, a validating setter and a
// pair of array methods
//
// Everything else follows the settled shape. What is new here is all in three
// places:
//
//  1. `MaxBones` is `public static literal int32 = 0x48` -- 72, a compile-time
//     constant with no storage, and the only public field in the whole stock
//     effect family.
//  2. `set_WeightsPerVertex` is 54 bytes and VALIDATES: anything but 1, 2 or 4
//     is an ArgumentOutOfRangeException carrying
//     FrameworkResources.SkinnedEffectWeightsPerVertex. It is therefore the one
//     managed setter in the family that is fallible, and its error channel
//     carries the reference's own refusal rather than an invented one.
//  3. `SetBoneTransforms` and `GetBoneTransforms` cross, and each guards
//     differently -- which is measured below rather than assumed symmetric.
//
// # LightingEnabled is EXPLICIT here too
//
// The same two bodies EnvironmentMapEffect has: a two-byte getter that answers
// true and a 51-byte setter that throws for false. The pinned contract lists no
// LightingEnabled property on this type either, so both are interface witnesses.
type SkinnedEffect struct {
	effect *Effect

	light0, light1, light2 *DirectionalLight

	world, view, projection framework.Matrix
	diffuseColor            framework.Vector3
	emissiveColor           framework.Vector3
	ambientLightColor       framework.Vector3
	alpha                   float32
	preferPerPixelLighting  bool
	oneLight                bool
	fogEnabled              bool
	fogStart, fogEnd        float32
	weightsPerVertex        int32
	dirtyFlags              effectDirtyFlags

	texture *Texture2D
}

// SkinnedEffectMaxBones is SkinnedEffect::MaxBones, `public static literal
// int32 = int32(0x00000048)`. It is a compile-time constant in the reference
// and a Go constant here, which is the same thing: no storage, and a value a
// consumer can use in an array length.
const SkinnedEffectMaxBones int32 = 72

// errSkinnedEffectNil is the Go-only guard for a zero value.
var errSkinnedEffectNil = errors.New("skinned effect is nil or uninitialized")

// The two Microsoft messages this type's guards carry, read from
// Microsoft.Xna.Framework.dll. SkinnedEffectMaxBonesMessage's {0} carries the
// bone limit, which the reference passes as the same 72.
const (
	skinnedEffectWeightsPerVertex = "SkinnedEffect.WeightsPerVertex must be 1, 2, or 4."
	// The bone limit crosses String.Format's {0} as a NUMBER in the reference
	// and as its rendered text here, which is what `%s` spells: the settled
	// placeholder rule maps {0} to %s, and 72 renders identically either way.
	skinnedEffectMaxBonesMessage = "SkinnedEffect supports a maximum of %s bones."
)

// NewSkinnedEffectByGraphicsDevice is SkinnedEffect::.ctor(GraphicsDevice), 201
// bytes. Its field initialisers are EnvironmentMapEffect's plus one --
// `weightsPerVertex = 4` -- and its tail, unlike the other lit effects', builds
// an identity bone array and pushes it:
//
//	Matrix[] identity = new Matrix[72];
//	for (int i = 0; i < 72; i++) identity[i] = Matrix.Identity;
//	SetBoneTransforms(identity);
func NewSkinnedEffectByGraphicsDevice(device *GraphicsDevice) (*SkinnedEffect, error) {
	if device == nil || device.device == nil {
		return nil, fmt.Errorf("%w: device: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := device.device.CreateSkinnedEffect()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(device, resource)
	if err != nil {
		return nil, err
	}
	skinned := newSkinnedEffectState()
	skinned.effect = effect
	effect.bindDerived(skinned)
	effect.bindDerivedEffect(skinned)
	if err := skinned.cacheEffectParameters(nil); err != nil {
		return nil, err
	}
	identity := make([]framework.Matrix, SkinnedEffectMaxBones)
	for index := range identity {
		identity[index] = framework.MatrixIdentity()
	}
	if err := skinned.SetBoneTransforms(identity); err != nil {
		return nil, err
	}
	return skinned, nil
}

// NewSkinnedEffectBySkinnedEffect is the PROTECTED clone constructor. It copies
// TWELVE values: preferPerPixelLighting, fogEnabled, the three matrices,
// diffuseColor, emissiveColor, ambientLightColor, alpha, fogStart, fogEnd and
// weightsPerVertex.
//
// The BONE TRANSFORMS are absent from that list, because they live in the
// cloned effect and come across with it.
func NewSkinnedEffectBySkinnedEffect(cloneSource *SkinnedEffect) (*SkinnedEffect, error) {
	if cloneSource == nil || cloneSource.effect == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	cloned, err := cloneSource.effect.cloneBase()
	if err != nil {
		return nil, err
	}
	skinned := newSkinnedEffectState()
	skinned.effect = cloned
	cloned.bindDerived(skinned)
	cloned.bindDerivedEffect(skinned)
	if err := skinned.cacheEffectParameters(cloneSource); err != nil {
		return nil, err
	}
	skinned.preferPerPixelLighting = cloneSource.preferPerPixelLighting
	skinned.fogEnabled = cloneSource.fogEnabled
	skinned.world = cloneSource.world
	skinned.view = cloneSource.view
	skinned.projection = cloneSource.projection
	skinned.diffuseColor = cloneSource.diffuseColor
	skinned.emissiveColor = cloneSource.emissiveColor
	skinned.ambientLightColor = cloneSource.ambientLightColor
	skinned.alpha = cloneSource.alpha
	skinned.fogStart = cloneSource.fogStart
	skinned.fogEnd = cloneSource.fogEnd
	skinned.weightsPerVertex = cloneSource.weightsPerVertex
	skinned.texture = cloneSource.texture
	return skinned, nil
}

func newSkinnedEffectState() *SkinnedEffect {
	return &SkinnedEffect{
		world:             framework.MatrixIdentity(),
		view:              framework.MatrixIdentity(),
		projection:        framework.MatrixIdentity(),
		diffuseColor:      framework.Vector3One(),
		emissiveColor:     framework.Vector3Zero(),
		ambientLightColor: framework.Vector3Zero(),
		alpha:             1,
		fogEnd:            1,
		weightsPerVertex:  4,
		dirtyFlags:        effectDirtyAll,
	}
}

func (e *SkinnedEffect) cacheEffectParameters(cloneSource *SkinnedEffect) error {
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

func (e *SkinnedEffect) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.SkinnedEffect"
}

func (e *SkinnedEffect) nativeResource() *interop.Resource {
	if e == nil || e.effect == nil {
		return nil
	}
	return e.effect.nativeResource()
}

// ---------------------------------------------------------------------------
// IEffectMatrices, with the lit words.
// ---------------------------------------------------------------------------

// World is SkinnedEffect::get_World.
func (e *SkinnedEffect) World() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.world
}

// SetWorld is SkinnedEffect::set_World.
func (e *SkinnedEffect) SetWorld(value framework.Matrix) {
	if e == nil {
		return
	}
	e.world = value
	e.dirtyFlags |= effectDirtyWorld | effectDirtyWorldViewProj | effectDirtyFog
}

// View is SkinnedEffect::get_View.
func (e *SkinnedEffect) View() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.view
}

// SetView is SkinnedEffect::set_View.
func (e *SkinnedEffect) SetView(value framework.Matrix) {
	if e == nil {
		return
	}
	e.view = value
	e.dirtyFlags |= effectDirtyWorldViewProj | effectDirtyEyePosition | effectDirtyFog
}

// Projection is SkinnedEffect::get_Projection.
func (e *SkinnedEffect) Projection() framework.Matrix {
	if e == nil {
		return framework.Matrix{}
	}
	return e.projection
}

// SetProjection is SkinnedEffect::set_Projection.
func (e *SkinnedEffect) SetProjection(value framework.Matrix) {
	if e == nil {
		return
	}
	e.projection = value
	e.dirtyFlags |= effectDirtyWorldViewProj
}

// ---------------------------------------------------------------------------
// IEffectFog.
// ---------------------------------------------------------------------------

// FogEnabled is SkinnedEffect::get_FogEnabled.
func (e *SkinnedEffect) FogEnabled() bool {
	if e == nil {
		return false
	}
	return e.fogEnabled
}

// SetFogEnabled is SkinnedEffect::set_FogEnabled.
func (e *SkinnedEffect) SetFogEnabled(value bool) {
	if e == nil || e.fogEnabled == value {
		return
	}
	e.fogEnabled = value
	e.dirtyFlags |= effectDirtyFogEnable | effectDirtyShaderIndex
}

// FogStart is SkinnedEffect::get_FogStart.
func (e *SkinnedEffect) FogStart() float32 {
	if e == nil {
		return 0
	}
	return e.fogStart
}

// SetFogStart is SkinnedEffect::set_FogStart.
func (e *SkinnedEffect) SetFogStart(value float32) {
	if e == nil {
		return
	}
	e.fogStart = value
	e.dirtyFlags |= effectDirtyFog
}

// FogEnd is SkinnedEffect::get_FogEnd.
func (e *SkinnedEffect) FogEnd() float32 {
	if e == nil {
		return 0
	}
	return e.fogEnd
}

// SetFogEnd is SkinnedEffect::set_FogEnd.
func (e *SkinnedEffect) SetFogEnd(value float32) {
	if e == nil {
		return
	}
	e.fogEnd = value
	e.dirtyFlags |= effectDirtyFog
}

// FogColor is SkinnedEffect::get_FogColor.
func (e *SkinnedEffect) FogColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errSkinnedEffectNil
	}
	values, err := resource.EffectFogColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetFogColor is SkinnedEffect::set_FogColor.
func (e *SkinnedEffect) SetFogColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
	}
	return resource.EffectFogSetColor(vector3Triple(value))
}

// ---------------------------------------------------------------------------
// IEffectLights. LightingEnabled is the witness pair.
// ---------------------------------------------------------------------------

// LightingEnabled is IEffectLights::LightingEnabled's getter, implemented
// EXPLICITLY as `ldc.i4.1; ret`. Lighting is always on for a skinned effect.
func (e *SkinnedEffect) LightingEnabled() bool { return true }

// SetLightingEnabled is the explicit setter: `true` is a no-op and `false`
// throws NotSupportedException carrying FrameworkResources.CantDisableLighting
// formatted with this type's short name. The refusal has nowhere to go through
// IEffectLights' measured infallible signature, which is recorded here for the
// reason it is recorded on EnvironmentMapEffect.
func (e *SkinnedEffect) SetLightingEnabled(value bool) {
	if value {
		return
	}
	_ = fmt.Sprintf(cantDisableLighting, "SkinnedEffect")
}

// AmbientLightColor is SkinnedEffect::get_AmbientLightColor.
func (e *SkinnedEffect) AmbientLightColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.ambientLightColor
}

// SetAmbientLightColor is SkinnedEffect::set_AmbientLightColor.
func (e *SkinnedEffect) SetAmbientLightColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.ambientLightColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// DirectionalLight0 is SkinnedEffect::get_DirectionalLight0.
func (e *SkinnedEffect) DirectionalLight0() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light0
}

// DirectionalLight1 is SkinnedEffect::get_DirectionalLight1.
func (e *SkinnedEffect) DirectionalLight1() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light1
}

// DirectionalLight2 is SkinnedEffect::get_DirectionalLight2.
func (e *SkinnedEffect) DirectionalLight2() *DirectionalLight {
	if e == nil {
		return nil
	}
	return e.light2
}

// EnableDefaultLighting is SkinnedEffect::EnableDefaultLighting, 30 bytes --
// the rig and the ambient colour, with no LightingEnabled store.
func (e *SkinnedEffect) EnableDefaultLighting() error {
	if e == nil {
		return errSkinnedEffectNil
	}
	ambient, err := effectHelpersEnableDefaultLighting(e.light0, e.light1, e.light2)
	if err != nil {
		return err
	}
	e.SetAmbientLightColor(ambient)
	return nil
}

// ---------------------------------------------------------------------------
// SkinnedEffect's own surface.
// ---------------------------------------------------------------------------

// DiffuseColor is SkinnedEffect::get_DiffuseColor.
func (e *SkinnedEffect) DiffuseColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.diffuseColor
}

// SetDiffuseColor is SkinnedEffect::set_DiffuseColor.
func (e *SkinnedEffect) SetDiffuseColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.diffuseColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// EmissiveColor is SkinnedEffect::get_EmissiveColor.
func (e *SkinnedEffect) EmissiveColor() framework.Vector3 {
	if e == nil {
		return framework.Vector3{}
	}
	return e.emissiveColor
}

// SetEmissiveColor is SkinnedEffect::set_EmissiveColor.
func (e *SkinnedEffect) SetEmissiveColor(value framework.Vector3) {
	if e == nil {
		return
	}
	e.emissiveColor = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// SpecularColor is SkinnedEffect::get_SpecularColor, a parameter read.
func (e *SkinnedEffect) SpecularColor() (framework.Vector3, error) {
	resource := e.nativeResource()
	if resource == nil {
		return framework.Vector3{}, errSkinnedEffectNil
	}
	values, err := resource.SkinnedEffectSpecularColor()
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// SetSpecularColor is SkinnedEffect::set_SpecularColor.
func (e *SkinnedEffect) SetSpecularColor(value framework.Vector3) error {
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
	}
	return resource.SkinnedEffectSetSpecularColor(vector3Triple(value))
}

// SpecularPower is SkinnedEffect::get_SpecularPower, a parameter read.
func (e *SkinnedEffect) SpecularPower() (float32, error) {
	resource := e.nativeResource()
	if resource == nil {
		return 0, errSkinnedEffectNil
	}
	return resource.SkinnedEffectSpecularPower()
}

// SetSpecularPower is SkinnedEffect::set_SpecularPower.
func (e *SkinnedEffect) SetSpecularPower(value float32) error {
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
	}
	return resource.SkinnedEffectSetSpecularPower(value)
}

// Alpha is SkinnedEffect::get_Alpha.
func (e *SkinnedEffect) Alpha() float32 {
	if e == nil {
		return 0
	}
	return e.alpha
}

// SetAlpha is SkinnedEffect::set_Alpha.
func (e *SkinnedEffect) SetAlpha(value float32) {
	if e == nil {
		return
	}
	e.alpha = value
	e.dirtyFlags |= effectDirtyMaterialColor
}

// PreferPerPixelLighting is SkinnedEffect::get_PreferPerPixelLighting.
func (e *SkinnedEffect) PreferPerPixelLighting() bool {
	if e == nil {
		return false
	}
	return e.preferPerPixelLighting
}

// SetPreferPerPixelLighting is SkinnedEffect::set_PreferPerPixelLighting, a
// preference that selects a shader permutation and reports back what it stored.
func (e *SkinnedEffect) SetPreferPerPixelLighting(value bool) {
	if e == nil || e.preferPerPixelLighting == value {
		return
	}
	e.preferPerPixelLighting = value
	e.dirtyFlags |= effectDirtyShaderIndex
}

// WeightsPerVertex is SkinnedEffect::get_WeightsPerVertex, whose default the
// constructor stores as `ldc.i4.4`.
func (e *SkinnedEffect) WeightsPerVertex() int32 {
	if e == nil {
		return 0
	}
	return e.weightsPerVertex
}

// SetWeightsPerVertex is SkinnedEffect::set_WeightsPerVertex, 54 bytes and the
// ONE managed setter in the stock-effect family that validates:
//
//	if (value != 1 && value != 2 && value != 4)
//	    throw new ArgumentOutOfRangeException("value",
//	        FrameworkResources.SkinnedEffectWeightsPerVertex);
//	weightsPerVertex = value;
//	dirtyFlags |= ShaderIndex;
//
// It is therefore fallible, and its error channel carries the reference's own
// refusal. Note it has NO early return: assigning the value it already holds
// stores it again and raises the flag again.
func (e *SkinnedEffect) SetWeightsPerVertex(value int32) error {
	if e == nil {
		return errSkinnedEffectNil
	}
	if value != 1 && value != 2 && value != 4 {
		return fmt.Errorf("%w: value: %s", errArgumentOutOfRange, skinnedEffectWeightsPerVertex)
	}
	e.weightsPerVertex = value
	e.dirtyFlags |= effectDirtyShaderIndex
	return nil
}

// Texture is SkinnedEffect::get_Texture, the managed field.
func (e *SkinnedEffect) Texture() (*Texture2D, error) {
	if e == nil {
		return nil, errSkinnedEffectNil
	}
	return e.texture, nil
}

// SetTexture is SkinnedEffect::set_Texture.
func (e *SkinnedEffect) SetTexture(value Texture2DReference) error {
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
	}
	texture := resolveTexture2D(value)
	if err := resource.SkinnedEffectSetTexture(texture.nativeResource()); err != nil {
		return err
	}
	e.texture = texture
	return nil
}

// SetBoneTransforms is SkinnedEffect::SetBoneTransforms(Matrix[]), 83 bytes:
//
//	if (boneTransforms == null || boneTransforms.Length == 0)
//	    throw new ArgumentNullException("boneTransforms", FrameworkResources.NullNotAllowed);
//	if (boneTransforms.Length > 72)
//	    throw new ArgumentException(string.Format(CultureInfo.CurrentCulture,
//	        FrameworkResources.SkinnedEffectMaxBones, 72));
//	bonesParam.SetValue(boneTransforms);
//
// The first guard is worth reading twice: an EMPTY array raises
// ArgumentNullException, not ArgumentException. That is the reference's choice
// and it is reproduced rather than tidied.
func (e *SkinnedEffect) SetBoneTransforms(boneTransforms []framework.Matrix) error {
	if e == nil {
		return errSkinnedEffectNil
	}
	if len(boneTransforms) == 0 {
		return fmt.Errorf("%w: boneTransforms: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if int32(len(boneTransforms)) > SkinnedEffectMaxBones {
		return fmt.Errorf("%w: %s", errGraphicsResourceArgument,
			fmt.Sprintf(skinnedEffectMaxBonesMessage, strconv.Itoa(int(SkinnedEffectMaxBones))))
	}
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
	}
	values := make([]float32, 0, len(boneTransforms)*16)
	for _, transform := range boneTransforms {
		row := matrixToRowMajor(transform)
		values = append(values, row[:]...)
	}
	return resource.SkinnedEffectSetBoneTransforms(values)
}

// GetBoneTransforms is SkinnedEffect::GetBoneTransforms(Int32), 110 bytes, and
// its two guards are NOT the setter's:
//
//	if (count <= 0) throw new ArgumentOutOfRangeException("count");
//	if (count > 72) throw new ArgumentOutOfRangeException("count",
//	    string.Format(CultureInfo.CurrentCulture,
//	                  FrameworkResources.SkinnedEffectMaxBones, 72));
//	Matrix[] result = bonesParam.GetValueMatrixArray(count);
//	for (int i = 0; i < count; i++) result[i].M44 = 1;
//	return result;
//
// Two details a reader would guess wrong. The zero-count refusal carries NO
// message and the over-limit one does. And the loop after the read forces M44
// to 1 on every returned matrix, because the shader stores a bone as three rows
// and the fourth comes back undefined -- so a projection that skipped it would
// hand back matrices the reference never returns.
func (e *SkinnedEffect) GetBoneTransforms(count int32) ([]framework.Matrix, error) {
	if e == nil {
		return nil, errSkinnedEffectNil
	}
	if count <= 0 {
		return nil, fmt.Errorf("%w: count", errArgumentOutOfRange)
	}
	if count > SkinnedEffectMaxBones {
		return nil, fmt.Errorf("%w: count: %s", errArgumentOutOfRange,
			fmt.Sprintf(skinnedEffectMaxBonesMessage, strconv.Itoa(int(SkinnedEffectMaxBones))))
	}
	resource := e.nativeResource()
	if resource == nil {
		return nil, errSkinnedEffectNil
	}
	values, err := resource.SkinnedEffectCopyBoneTransforms(int(count))
	if err != nil {
		return nil, err
	}
	transforms := make([]framework.Matrix, len(values)/16)
	for index := range transforms {
		var row [16]float32
		copy(row[:], values[index*16:(index+1)*16])
		transform := matrixFromRowMajor(row)
		// The M44 = 1 the reference writes after every read.
		transform.M44 = 1
		transforms[index] = transform
	}
	return transforms, nil
}

// ---------------------------------------------------------------------------
// The two virtual members, and the push.
// ---------------------------------------------------------------------------

// Clone is SkinnedEffect::Clone.
func (e *SkinnedEffect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errSkinnedEffectNil
	}
	return NewSkinnedEffectBySkinnedEffect(e)
}

// OnApply is SkinnedEffect::OnApply, 364 bytes.
func (e *SkinnedEffect) OnApply() error {
	if e == nil {
		return errSkinnedEffectNil
	}
	resource := e.nativeResource()
	if resource == nil {
		return errSkinnedEffectNil
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
		if err := resource.SkinnedEffectSetDiffuseColor(vector3Triple(e.diffuseColor)); err != nil {
			return err
		}
		if err := resource.SkinnedEffectSetEmissiveColor(vector3Triple(e.emissiveColor)); err != nil {
			return err
		}
		if err := resource.SkinnedEffectSetAlpha(e.alpha); err != nil {
			return err
		}
		if err := resource.EffectLightsSetAmbientColor(vector3Triple(e.ambientLightColor)); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyMaterialColor
	}
	// Lighting is always on, so the recomputation is unbracketed.
	oneLight := !e.light1.Enabled() && !e.light2.Enabled()
	if e.oneLight != oneLight {
		e.oneLight = oneLight
		e.dirtyFlags |= effectDirtyShaderIndex
	}
	if e.dirtyFlags&effectDirtyShaderIndex != 0 {
		if err := resource.SkinnedEffectSetPreferPerPixelLighting(e.preferPerPixelLighting); err != nil {
			return err
		}
		if err := resource.SkinnedEffectSetWeightsPerVertex(e.weightsPerVertex); err != nil {
			return err
		}
		e.dirtyFlags &^= effectDirtyShaderIndex
	}
	return nil
}

// releaseDerivedNativeObjects releases the three light views.
func (e *SkinnedEffect) releaseDerivedNativeObjects() error {
	if e == nil {
		return nil
	}
	return errors.Join(e.light0.dispose(), e.light1.dispose(), e.light2.dispose())
}

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *SkinnedEffect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *SkinnedEffect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *SkinnedEffect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *SkinnedEffect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errSkinnedEffectNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *SkinnedEffect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *SkinnedEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *SkinnedEffect) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *SkinnedEffect) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *SkinnedEffect) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *SkinnedEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (e *SkinnedEffect) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *SkinnedEffect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errSkinnedEffectNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *SkinnedEffect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errSkinnedEffectNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), this type's only dispose member.
func (e *SkinnedEffect) Dispose() error {
	if e == nil {
		return errSkinnedEffectNil
	}
	return e.effect.DisposeByNone()
}

var (
	_ IEffectMatrices = (*SkinnedEffect)(nil)
	_ IEffectFog      = (*SkinnedEffect)(nil)
	_ IEffectLights   = (*SkinnedEffect)(nil)
)
