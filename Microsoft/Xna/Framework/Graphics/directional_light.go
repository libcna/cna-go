package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// DirectionalLight is one of the three lights a stock effect publishes: a
// direction, a diffuse colour, a specular colour and an enable flag.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # The reference object is a CACHE in front of three EffectParameters
//
//	.class public auto ansi sealed beforefieldinit DirectionalLight
//	       extends [mscorlib]System.Object
//	  .field private EffectParameter directionParam, diffuseColorParam, specularColorParam
//	  .field private bool enabled
//	  .field private Vector3 cachedDirection, cachedDiffuseColor, cachedSpecularColor
//
// Every one of the four GETTERS is a single `ldfld` of a cached field -- so all
// four are INFALLIBLE, and none of them reads back through a parameter. Every
// one of the four SETTERS writes the cache AND, when its parameter is not null,
// writes through it with EffectParameter::SetValue, which ends in
// `calli unmanaged stdcall` -- so all four are FALLIBLE.
//
// That split is reproduced exactly, and it is the reason this type does not
// simply forward everything to CNA.
//
// # What CNA offers, and what the projection uses it for
//
// A stock effect published by CNA reports ZERO EffectParameters. The Foundation
// 79 probe loaded `{"cnjVersion":1,"type":"BasicEffect"}` through
// ContentManager on both qualified artifacts:
//
//	HEADLESS   PARAMETER_COUNT 0    TECHNIQUE_COUNT 1  ("Default", 1 pass)
//	SOFTWARE   PARAMETER_COUNT 0    TECHNIQUE_COUNT 1  ("Default", 1 pass)
//
// So the reference's write-through target does not exist here. CNA models the
// light itself instead -- `cna_effect_lights_get_directional_light` hands back a
// view of the light an effect owns and `cna_directional_light_set_*` writes
// through it -- and the projection uses THAT as the write-through target. The
// shape is the reference's, unchanged: cache on the way in, native write when
// there is somewhere to write.
//
// # A light the public constructor built writes nowhere
//
// The constructor takes the three EffectParameters, and with no parameters in
// existence a consumer can only pass nil. The reference's own setters guard
// each parameter with `brfalse` and skip the write, so a light with three null
// parameters is a pure cache -- which is exactly what this projection gives,
// and it is why `cna_directional_light_create` is NOT bound: a free-standing
// native light no effect reads would change nothing observable.
type DirectionalLight struct {
	// resource is the effect-owned light this object writes through, or nil for
	// a light the public constructor built. It stands in the three
	// EffectParameter fields' place: the reference has three null checks, this
	// has one, because CNA models the light as one object rather than as three
	// parameters.
	resource *interop.Resource
	// The three EffectParameters the constructor was given. The contract
	// declares them at the constructor's signature positions, so they are
	// stored; nothing reads them, because CNA publishes no parameter a caller
	// could have obtained one from.
	directionParameter     *EffectParameter
	diffuseColorParameter  *EffectParameter
	specularColorParameter *EffectParameter

	// The four cached fields, which are what every getter answers.
	enabled             bool
	cachedDirection     framework.Vector3
	cachedDiffuseColor  framework.Vector3
	cachedSpecularColor framework.Vector3
}

// NewDirectionalLight is
// DirectionalLight::.ctor(EffectParameter, EffectParameter, EffectParameter, DirectionalLight).
//
// The IL has two arms and they behave differently in a way that is easy to get
// wrong. With a cloneSource it copies the four fields with `stfld` and writes
// NOTHING through: a cloned light does not push its state into its own
// parameters. With no cloneSource it calls the three SETTERS with Vector3.Down,
// Vector3.One and Vector3.Zero -- so `enabled` stays false, the direction write
// reaches the parameter (set_Direction has no enabled guard) and the two colour
// writes do not (both are guarded by `enabled`).
//
// It is FALLIBLE, and that is the contract's rather than this projection's: the
// setter arm reaches EffectParameter::SetValue, which the reference's own
// constructor can therefore fail in. That a publicly-constructed light has no
// parameters to write in CNA-Go, and so cannot in fact fail, does not narrow
// the declared contract.
func NewDirectionalLight(
	directionParameter, diffuseColorParameter, specularColorParameter *EffectParameter,
	cloneSource *DirectionalLight,
) (*DirectionalLight, error) {
	light := &DirectionalLight{
		directionParameter:     directionParameter,
		diffuseColorParameter:  diffuseColorParameter,
		specularColorParameter: specularColorParameter,
	}
	if cloneSource != nil {
		// The `stfld` arm: four field copies, no write-through, and NOT the
		// clone source's native light -- the new object is a cache with no
		// effect behind it, exactly as the reference's is a cache in front of
		// the three parameters this constructor was given.
		light.enabled = cloneSource.enabled
		light.cachedDirection = cloneSource.cachedDirection
		light.cachedDiffuseColor = cloneSource.cachedDiffuseColor
		light.cachedSpecularColor = cloneSource.cachedSpecularColor
		return light, nil
	}
	// The setter arm, in the reference's order.
	if err := light.SetDirection(framework.Vector3Down()); err != nil {
		return nil, err
	}
	if err := light.SetDiffuseColor(framework.Vector3One()); err != nil {
		return nil, err
	}
	if err := light.SetSpecularColor(framework.Vector3Zero()); err != nil {
		return nil, err
	}
	return light, nil
}

// newPublishedDirectionalLight is the light a stock effect publishes: the same
// object shape with a native light behind it, and with the effect's own
// defaults read once so the cache starts where CNA's light is.
//
// The reference's BasicEffect gets its three lights from CacheEffectParameters,
// which passes the effect's own DirLight<n>* parameters and a clone source; the
// defaults then come from the constructor's setter arm. Here the defaults come
// from the same setter arm, and they are pushed INTO CNA rather than read out
// of it, so a caller sees XNA's documented defaults rather than CNA's.
func newPublishedDirectionalLight(resource *interop.Resource, cloneSource *DirectionalLight) (*DirectionalLight, error) {
	light := &DirectionalLight{resource: resource}
	if cloneSource != nil {
		light.enabled = cloneSource.enabled
		light.cachedDirection = cloneSource.cachedDirection
		light.cachedDiffuseColor = cloneSource.cachedDiffuseColor
		light.cachedSpecularColor = cloneSource.cachedSpecularColor
		// A clone copies fields and writes nothing through in the reference,
		// because the cloned D3DX effect already carries the values. CNA's
		// cna_effect_clone does the same for the native light, so nothing is
		// pushed here either.
		return light, nil
	}
	if err := light.SetDirection(framework.Vector3Down()); err != nil {
		return nil, err
	}
	if err := light.SetDiffuseColor(framework.Vector3One()); err != nil {
		return nil, err
	}
	if err := light.SetSpecularColor(framework.Vector3Zero()); err != nil {
		return nil, err
	}
	return light, nil
}

// Enabled is DirectionalLight::get_Enabled -- `ldarg.0; ldfld enabled; ret`.
func (l *DirectionalLight) Enabled() bool {
	if l == nil {
		return false
	}
	return l.enabled
}

// SetEnabled is DirectionalLight::set_Enabled, whose 124 bytes are one early
// return and two symmetric arms:
//
//	if (enabled == value) return;
//	enabled = value;
//	if (enabled) { diffuseColorParam?.SetValue(cachedDiffuseColor);
//	               specularColorParam?.SetValue(cachedSpecularColor); }
//	else         { diffuseColorParam?.SetValue(Vector3.Zero);
//	               specularColorParam?.SetValue(Vector3.Zero); }
//
// Disabling a light is expressed to the shader as zero colours while the
// CACHED colours are untouched, which is why get_DiffuseColor still answers the
// stored colour on a disabled light. CNA performs the same substitution behind
// cna_directional_light_set_enabled, so one route stands for the whole body --
// and the early return is reproduced managed-side because it is the reference's
// and it decides whether the two writes happen at all.
func (l *DirectionalLight) SetEnabled(value bool) error {
	if l == nil {
		return errEffectNil
	}
	if l.enabled == value {
		return nil
	}
	l.enabled = value
	if l.resource == nil {
		return nil
	}
	return l.resource.DirectionalLightSetEnabled(value)
}

// Direction is DirectionalLight::get_Direction, one `ldfld`.
func (l *DirectionalLight) Direction() framework.Vector3 {
	if l == nil {
		return framework.Vector3{}
	}
	return l.cachedDirection
}

// SetDirection is DirectionalLight::set_Direction:
//
//	directionParam?.SetValue(value);
//	cachedDirection = value;
//
// The parameter write comes FIRST and the cache second, and there is no
// `enabled` guard -- direction is written whether the light is on or off.
func (l *DirectionalLight) SetDirection(value framework.Vector3) error {
	if l == nil {
		return errEffectNil
	}
	if l.resource != nil {
		if err := l.resource.DirectionalLightSetDirection(vector3Triple(value)); err != nil {
			return err
		}
	}
	l.cachedDirection = value
	return nil
}

// DiffuseColor is DirectionalLight::get_DiffuseColor, one `ldfld` of the CACHE
// -- so a disabled light still reports the colour it was given, not the zero
// its parameter holds.
func (l *DirectionalLight) DiffuseColor() framework.Vector3 {
	if l == nil {
		return framework.Vector3{}
	}
	return l.cachedDiffuseColor
}

// SetDiffuseColor is DirectionalLight::set_DiffuseColor:
//
//	if (enabled) diffuseColorParam?.SetValue(value);
//	cachedDiffuseColor = value;
//
// The write is skipped on a disabled light, and the cache is updated either
// way, so enabling the light later publishes the colour set while it was off.
func (l *DirectionalLight) SetDiffuseColor(value framework.Vector3) error {
	if l == nil {
		return errEffectNil
	}
	if l.enabled && l.resource != nil {
		if err := l.resource.DirectionalLightSetDiffuseColor(vector3Triple(value)); err != nil {
			return err
		}
	}
	l.cachedDiffuseColor = value
	return nil
}

// SpecularColor is DirectionalLight::get_SpecularColor, cached the same way.
func (l *DirectionalLight) SpecularColor() framework.Vector3 {
	if l == nil {
		return framework.Vector3{}
	}
	return l.cachedSpecularColor
}

// SetSpecularColor is DirectionalLight::set_SpecularColor, guarded by `enabled`
// exactly as the diffuse setter is.
func (l *DirectionalLight) SetSpecularColor(value framework.Vector3) error {
	if l == nil {
		return errEffectNil
	}
	if l.enabled && l.resource != nil {
		if err := l.resource.DirectionalLightSetSpecularColor(vector3Triple(value)); err != nil {
			return err
		}
	}
	l.cachedSpecularColor = value
	return nil
}

// dispose destroys the member-view handle this object holds. The canonical
// header calls cna_effect_lights_get_directional_light's result "an owned
// stable member-view handle" and cna_directional_light_destroy "destroys a
// standalone or NESTED DirectionalLight view handle", so the VIEW is the
// caller's to release while the light behind it stays the effect's.
func (l *DirectionalLight) dispose() error {
	if l == nil || l.resource == nil {
		return nil
	}
	return l.resource.Dispose()
}

// vector3Triple is the three floats CNA_Vector3 declares, in its order.
func vector3Triple(value framework.Vector3) [3]float32 {
	return [3]float32{value.X, value.Y, value.Z}
}
