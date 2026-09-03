package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// EffectMaterial is the effect the content pipeline builds for a model mesh
// part: an Effect with a different class name and nothing else.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # Its whole body is eight bytes
//
//	.class public auto ansi beforefieldinit EffectMaterial
//	       extends Effect
//	  .ctor(Effect cloneSource)
//	    IL_0000: ldarg.0
//	    IL_0001: ldarg.1
//	    IL_0002: call instance void Effect::.ctor(class Effect)
//	    IL_0007: ret
//
// It declares ONE member, that constructor, and overrides nothing -- not
// `Clone`, not `OnApply`. So the entire type is Effect's public surface plus a
// class name, and the only thing a consumer can observe that distinguishes it
// is `GetType()` and the `ToString()` that answers from it.
//
// That makes it the cheapest type in the profile to project and the one most
// easily got wrong by projecting nothing: its inherited surface is real, a
// consumer reaches Parameters and Techniques through it, and `Clone` on one
// answers an Effect rather than an EffectMaterial because the reference does
// not override it.
//
// # A hazard CNA documents and this type does not yet answer
//
// `cna_effect_material_retain_parameter_texture_ext` exists because "a
// parameter holds a raw pointer nothing owns", and retaining is the mechanism
// that removes the hazard. It backs no member of the pinned contract -- XNA's
// EffectMaterial declares none -- so nothing here calls it, and a consumer who
// writes a texture into one of this material's parameters and then disposes
// that texture leaves CNA holding a dangling pointer. That is inherited from
// Foundation 72's `cna_effect_parameter_set_value_texture` binding rather than
// introduced here, and it is recorded rather than papered over.
type EffectMaterial struct {
	// effect is the composed Effect, and it is the whole of this type's state.
	effect *Effect
}

// errEffectMaterialNil is the Go-only guard for a zero value.
var errEffectMaterialNil = errors.New("effect material is nil or uninitialized")

// NewEffectMaterial is EffectMaterial::.ctor(Effect cloneSource).
//
// The base constructor it chains to is the one that clones, and its first
// statement is the null check with FrameworkResources.NullNotAllowed -- so a
// null source is refused here with the same message, before anything native.
//
// CNA's counterpart takes the source effect rather than a device, for the same
// reason: `cna_effect_material_create(clone_source, out_effect)` "creates an
// EffectMaterial using the source effect's graphics device".
func NewEffectMaterial(cloneSource EffectReference) (*EffectMaterial, error) {
	source := resolveEffect(cloneSource)
	if source == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	resource, err := source.nativeResource().CreateEffectMaterial()
	if err != nil {
		return nil, err
	}
	effect, err := newEffect(source.GraphicsDevice(), resource)
	if err != nil {
		return nil, err
	}
	material := &EffectMaterial{effect: effect}
	// bindDerived only. bindDerivedEffect installs an OVERRIDE, and this type
	// overrides nothing -- its OnApply is the base's four-byte `ret` and its
	// Clone is the base's, which builds an Effect. Installing a hook here would
	// be a dispatch to a body that does not exist.
	effect.bindDerived(material)
	return material, nil
}

func (e *EffectMaterial) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.EffectMaterial"
}

func (e *EffectMaterial) effectBase() *Effect {
	if e == nil {
		return nil
	}
	return e.effect
}

// ---------------------------------------------------------------------------
// The inherited public surface of Effect, forwarded. Every member of this type
// is one of these: it declares nothing but a constructor.
// ---------------------------------------------------------------------------

// Parameters is Effect::get_Parameters.
func (e *EffectMaterial) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.effect.Parameters()
}

// Techniques is Effect::get_Techniques.
func (e *EffectMaterial) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.effect.Techniques()
}

// CurrentTechnique is Effect::get_CurrentTechnique.
func (e *EffectMaterial) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.effect.CurrentTechnique()
}

// SetCurrentTechnique is Effect::set_CurrentTechnique.
func (e *EffectMaterial) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errEffectMaterialNil
	}
	return e.effect.SetCurrentTechnique(value)
}

// Clone is Effect::Clone, INHERITED rather than overridden. It answers an
// Effect and not an EffectMaterial, because the reference declares no override
// and `Effect::Clone` builds an Effect.
func (e *EffectMaterial) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errEffectMaterialNil
	}
	return e.effect.cloneBase()
}

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *EffectMaterial) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.effect.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *EffectMaterial) Name() string {
	if e == nil {
		return ""
	}
	return e.effect.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *EffectMaterial) SetName(value string) {
	if e == nil {
		return
	}
	e.effect.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *EffectMaterial) Tag() any {
	if e == nil {
		return nil
	}
	return e.effect.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *EffectMaterial) SetTag(value any) {
	if e == nil {
		return
	}
	e.effect.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *EffectMaterial) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.effect.IsDisposed()
}

// ToString is GraphicsResource::ToString. It is the one member that
// distinguishes this type from its base at runtime, because the fallback names
// the RUNTIME type.
func (e *EffectMaterial) ToString() string {
	if e == nil {
		return ""
	}
	return e.effect.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *EffectMaterial) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errEffectMaterialNil
	}
	return e.effect.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *EffectMaterial) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errEffectMaterialNil
	}
	return e.effect.RemoveDisposingHandler(subscription)
}

// Dispose is GraphicsResource::Dispose(), this type's only dispose member.
func (e *EffectMaterial) Dispose() error {
	if e == nil {
		return errEffectMaterialNil
	}
	return e.effect.DisposeByNone()
}
