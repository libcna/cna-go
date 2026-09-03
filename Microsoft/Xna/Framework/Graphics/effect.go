package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 72 — Effect, EffectTechnique(Collection) and EffectPass(Collection).
// ---------------------------------------------------------------------------

// notCurrentTechnique is FrameworkResources.NotCurrentTechnique, the one
// message EffectPass::Apply throws.
const notCurrentTechnique = "Cannot Apply an EffectPass that is not from the CurrentTechnique."

// EffectPass is Microsoft.Xna.Framework.Graphics.EffectPass.
//
// # Apply is the member the whole cluster exists for
//
//	Helpers.CheckDisposed(effect, effect.pComPtr);
//	if (effect.CurrentTechnique != this._technique)
//	    throw new InvalidOperationException(FrameworkResources.NotCurrentTechnique);
//	effect.OnApply();
//	... BeginPass on the D3DX effect ...
//
// Both guards are reproduced managed-side because both are the reference's own
// and neither is CNA's: CNA's cna_effect_pass_apply accepts a pass from a
// technique that is not current.
type EffectPass struct {
	view        *interop.EffectView
	technique   *EffectTechnique
	name        string
	annotations *EffectAnnotationCollection
}

func newEffectPass(view *interop.EffectView, technique *EffectTechnique) (*EffectPass, error) {
	name, err := view.String(interop.EffectStringPassName)
	if err != nil {
		return nil, err
	}
	annotationView, err := view.PassAnnotations()
	if err != nil {
		return nil, err
	}
	annotations, err := newEffectAnnotationCollection(annotationView)
	if err != nil {
		return nil, err
	}
	return &EffectPass{view: view, technique: technique, name: name, annotations: annotations}, nil
}

// Name is EffectPass::get_Name, one `ldfld`.
func (p *EffectPass) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Annotations is EffectPass::get_Annotations, one `ldfld` over the collection
// the constructor built.
func (p *EffectPass) Annotations() *EffectAnnotationCollection {
	if p == nil {
		return nil
	}
	return p.annotations
}

// Apply is EffectPass::Apply.
func (p *EffectPass) Apply() error {
	if p == nil || p.technique == nil || p.technique.effect == nil {
		return errEffectNil
	}
	effect := p.technique.effect
	if effect.IsDisposed() {
		// Helpers::CheckDisposed(effect, effect.pComPtr) names the effect
		// object's RUNTIME type, which for a stock effect is the stock
		// effect's. The field holds the composed base half, so the name comes
		// through the CLR `this` rather than from it.
		return fmt.Errorf("%w: %s", errObjectDisposed, effect.resource.self().clrTypeName())
	}
	if effect.currentTechnique != p.technique {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, notCurrentTechnique)
	}
	if err := effect.OnApply(); err != nil {
		return err
	}
	return p.view.ApplyPass()
}

func (p *EffectPass) dispose() error {
	if p == nil {
		return nil
	}
	return errors.Join(p.annotations.dispose(), p.view.Dispose())
}

// EffectPassCollection is Microsoft.Xna.Framework.Graphics.EffectPassCollection,
// with the same null-answering indexers every collection in this cluster has.
type EffectPassCollection struct {
	view   *interop.EffectView
	passes []*EffectPass
}

func newEffectPassCollection(view *interop.EffectView, technique *EffectTechnique) (*EffectPassCollection, error) {
	count, err := view.Count(interop.EffectCollectionPass)
	if err != nil {
		return nil, err
	}
	collection := &EffectPassCollection{view: view}
	for index := uint64(0); index < count; index++ {
		element, err := view.At(interop.EffectCollectionPass, index, interop.EffectViewPass)
		if err != nil {
			return nil, err
		}
		pass, err := newEffectPass(element, technique)
		if err != nil {
			return nil, err
		}
		collection.passes = append(collection.passes, pass)
	}
	return collection, nil
}

// Count is EffectPassCollection::get_Count.
func (c *EffectPassCollection) Count() int32 {
	if c == nil {
		return 0
	}
	return int32(len(c.passes))
}

// ItemPropertySignatureCA1DC5FC is EffectPassCollection::get_Item(Int32); out of range is nil.
func (c *EffectPassCollection) ItemPropertySignatureCA1DC5FC(index int32) *EffectPass {
	if c == nil || index < 0 || int(index) >= len(c.passes) {
		return nil
	}
	return c.passes[index]
}

// ItemPropertySignatureE281298D is EffectPassCollection::get_Item(String); no match is nil.
func (c *EffectPassCollection) ItemPropertySignatureE281298D(name string) *EffectPass {
	if c == nil {
		return nil
	}
	for _, pass := range c.passes {
		if pass.name == name {
			return pass
		}
	}
	return nil
}

// GetEnumerator is EffectPassCollection::GetEnumerator.
func (c *EffectPassCollection) GetEnumerator() framework.Iterator[*EffectPass] {
	if c == nil {
		return &effectIterator[*EffectPass]{}
	}
	return &effectIterator[*EffectPass]{items: c.passes}
}

func (c *EffectPassCollection) dispose() error {
	if c == nil {
		return nil
	}
	var failures []error
	for _, pass := range c.passes {
		failures = append(failures, pass.dispose())
	}
	failures = append(failures, c.view.Dispose())
	return errors.Join(failures...)
}

// EffectTechnique is Microsoft.Xna.Framework.Graphics.EffectTechnique. Three
// members, all `ldfld` over what the constructor built.
type EffectTechnique struct {
	view        *interop.EffectView
	effect      *Effect
	name        string
	passes      *EffectPassCollection
	annotations *EffectAnnotationCollection
}

func newEffectTechnique(view *interop.EffectView, effect *Effect) (*EffectTechnique, error) {
	name, err := view.String(interop.EffectStringTechniqueName)
	if err != nil {
		return nil, err
	}
	technique := &EffectTechnique{view: view, effect: effect, name: name}
	passView, err := view.Passes()
	if err != nil {
		return nil, err
	}
	if technique.passes, err = newEffectPassCollection(passView, technique); err != nil {
		return nil, err
	}
	annotationView, err := view.TechniqueAnnotations()
	if err != nil {
		return nil, err
	}
	if technique.annotations, err = newEffectAnnotationCollection(annotationView); err != nil {
		return nil, err
	}
	return technique, nil
}

// Name is EffectTechnique::get_Name.
func (t *EffectTechnique) Name() string {
	if t == nil {
		return ""
	}
	return t.name
}

// Passes is EffectTechnique::get_Passes.
func (t *EffectTechnique) Passes() *EffectPassCollection {
	if t == nil {
		return nil
	}
	return t.passes
}

// Annotations is EffectTechnique::get_Annotations.
func (t *EffectTechnique) Annotations() *EffectAnnotationCollection {
	if t == nil {
		return nil
	}
	return t.annotations
}

func (t *EffectTechnique) dispose() error {
	if t == nil {
		return nil
	}
	return errors.Join(t.passes.dispose(), t.annotations.dispose(), t.view.Dispose())
}

// EffectTechniqueCollection is
// Microsoft.Xna.Framework.Graphics.EffectTechniqueCollection.
type EffectTechniqueCollection struct {
	view       *interop.EffectView
	techniques []*EffectTechnique
}

func newEffectTechniqueCollection(view *interop.EffectView, effect *Effect) (*EffectTechniqueCollection, error) {
	count, err := view.Count(interop.EffectCollectionTechnique)
	if err != nil {
		return nil, err
	}
	collection := &EffectTechniqueCollection{view: view}
	for index := uint64(0); index < count; index++ {
		element, err := view.At(interop.EffectCollectionTechnique, index, interop.EffectViewTechnique)
		if err != nil {
			return nil, err
		}
		technique, err := newEffectTechnique(element, effect)
		if err != nil {
			return nil, err
		}
		collection.techniques = append(collection.techniques, technique)
	}
	return collection, nil
}

// Count is EffectTechniqueCollection::get_Count.
func (c *EffectTechniqueCollection) Count() int32 {
	if c == nil {
		return 0
	}
	return int32(len(c.techniques))
}

// ItemPropertySignatureA5F4623F is EffectTechniqueCollection::get_Item(Int32); out of range is nil.
func (c *EffectTechniqueCollection) ItemPropertySignatureA5F4623F(index int32) *EffectTechnique {
	if c == nil || index < 0 || int(index) >= len(c.techniques) {
		return nil
	}
	return c.techniques[index]
}

// ItemPropertySignatureDA594950 is EffectTechniqueCollection::get_Item(String); no match is nil.
func (c *EffectTechniqueCollection) ItemPropertySignatureDA594950(name string) *EffectTechnique {
	if c == nil {
		return nil
	}
	for _, technique := range c.techniques {
		if technique.name == name {
			return technique
		}
	}
	return nil
}

// GetEnumerator is EffectTechniqueCollection::GetEnumerator.
func (c *EffectTechniqueCollection) GetEnumerator() framework.Iterator[*EffectTechnique] {
	if c == nil {
		return &effectIterator[*EffectTechnique]{}
	}
	return &effectIterator[*EffectTechnique]{items: c.techniques}
}

func (c *EffectTechniqueCollection) dispose() error {
	if c == nil {
		return nil
	}
	var failures []error
	for _, technique := range c.techniques {
		failures = append(failures, technique.dispose())
	}
	failures = append(failures, c.view.Dispose())
	return errors.Join(failures...)
}

// Effect is Microsoft.Xna.Framework.Graphics.Effect:
//
//	.class public auto ansi beforefieldinit Effect
//	       extends Microsoft.Xna.Framework.Graphics.GraphicsResource
//	       implements Microsoft.Xna.Framework.Graphics.IGraphicsResource
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_effect_destroy.
//
// The whole reflected graph -- parameters, techniques, passes, annotations and
// their nested collections -- is materialised at construction, exactly as the
// reference's constructor materialises its Lists, and every accessor is then a
// field read that answers the SAME object.
//
// # Its one public constructor is the compiled-bytecode one
//
//	.ctor(GraphicsDevice graphicsDevice, byte[] effectCode)
//
// and CNA's counterpart is cna_effect_create_compiled, which takes the same
// Direct3D 9 Effect Framework binary. Whether the running renderer accepts it
// is CNA_GRAPHICS_CAPABILITY_COMPILED_EFFECTS, and the Foundation 72 probe
// measured that capability FALSE on all three published artifacts -- HEADLESS,
// SOFTWARE and OPENGL33 -- so this constructor reports CNA's refusal on every
// environment CNA-Go can qualify against. That is a renderer property, stated
// as one, and it is not the type's only door: ContentManager.Load<Effect> reads
// a `.cnj` descriptor naming a stock effect and needs no such capability.
type Effect struct {
	resource         *GraphicsResource
	parameters       *EffectParameterCollection
	techniques       *EffectTechniqueCollection
	currentTechnique *EffectTechnique
	// derived is the CLR `this` for Effect's TWO virtual members. Foundation 79
	// needed it: EffectPass::Apply calls OnApply through the virtual slot, and
	// on a BasicEffect that slot holds BasicEffect's override, not the empty
	// base body. Composition has no vtable, so the derived object installs
	// itself here and the base's virtual entry points dispatch through it --
	// the same shape graphicsResourceObject already has for ToString's
	// GetType() call, one level up and with two members instead of one.
	derived effectVirtuals
}

// effectVirtuals is the pair of members Effect declares `virtual` and every
// stock effect overrides. A type that composes an Effect installs itself with
// bindDerivedEffect, and the base's Clone and OnApply then resolve to the
// override exactly as `callvirt` would.
type effectVirtuals interface {
	// OnApply is the derived body EffectPass::Apply reaches.
	//
	// Clone is deliberately NOT here, although the reference declares both
	// virtual. Go's interface dispatch already supplies Clone's virtual
	// behaviour: Clone is a member of EffectReference, so a consumer holding
	// one calls the DERIVED method directly, and the composed *Effect half is
	// unexported -- effectBase is lowercase -- so no consumer can obtain it and
	// reach the base's Clone at all. A dispatch here would be a branch nothing
	// executes, which is the shape this project refuses everywhere else.
	//
	// OnApply is different, and that difference is the whole reason this
	// interface exists: EffectPass holds the base object in a field, calls
	// OnApply on it, and Go dispatches to the base's method. Nothing in the
	// language recovers the derived body there.
	OnApply() error
	// releaseDerivedNativeObjects is not a CLR member. It exists because a
	// derived effect can own CNA handles the reference's derived class has no
	// counterpart for -- BasicEffect's three light views are the first -- and
	// those must go when the effect does. It is unexported, so nothing outside
	// this package can install or observe it.
	releaseDerivedNativeObjects() error
}

// errObjectDisposed projects System.ObjectDisposedException, which
// Helpers.CheckDisposed throws with the disposed object's type name.
var errObjectDisposed = errors.New("cannot access a disposed object")

// NewEffectByGraphicsDeviceAndSliceOfByte is Effect::.ctor(GraphicsDevice, Byte[]).
//
// The reference's first guard is GraphicsResource's own device check; every
// check after it belongs to D3DX's effect compiler, which CNA-Go does not
// reproduce for the reason Texture2D's format checks are not reproduced.
func NewEffectByGraphicsDeviceAndSliceOfByte(graphicsDevice *GraphicsDevice, effectCode []uint8) (*Effect, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := graphicsDevice.device.CreateCompiledEffect(effectCode)
	if err != nil {
		return nil, err
	}
	return newEffect(graphicsDevice, resource)
}

// NewEffectByEffect is Effect::.ctor(Effect cloneSource), the PROTECTED
// constructor Clone() and every derived stock effect use:
//
//	if (cloneSource == null)
//	    throw new ArgumentNullException("cloneSource", FrameworkResources.NullNotAllowed);
//	... clone the D3DX effect and rebuild the reflected graph ...
//
// It is projected because the settled surface rule is "declared public AND
// protected", and it is exported for the reason every projected protected
// member is: Go has no protected, and a consumer deriving from Effect -- which
// is what this constructor is for -- must be able to name it.
func NewEffectByEffect(cloneSource EffectReference) (*Effect, error) {
	source := resolveEffect(cloneSource)
	if source == nil {
		return nil, fmt.Errorf("%w: cloneSource: %s",
			errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	// cloneBase, not Clone: the reference's constructor clones the D3DX effect
	// itself rather than calling the virtual, so a BasicEffect passed here
	// produces an Effect and not a second BasicEffect. That is the reference's
	// behaviour and it is why the derived classes each declare their OWN clone
	// constructor rather than reusing this one.
	return source.cloneBase()
}

// newEffect materialises the whole reflected graph, which is what the
// reference's constructor does after the D3DX effect exists.
func newEffect(device *GraphicsDevice, resource *interop.Resource) (*Effect, error) {
	effect := &Effect{resource: newGraphicsResource(device, resource)}
	effect.resource.bindDerived(effect)

	parameterView, err := resource.EffectParameters()
	if err != nil {
		return nil, err
	}
	if effect.parameters, err = newEffectParameterCollection(parameterView); err != nil {
		return nil, err
	}
	techniqueView, err := resource.EffectTechniques()
	if err != nil {
		return nil, err
	}
	if effect.techniques, err = newEffectTechniqueCollection(techniqueView, effect); err != nil {
		return nil, err
	}
	// The reference's constructor selects the FIRST technique, which is what
	// D3DX's GetTechnique(0) answers and what CNA reports through
	// cna_effect_get_current_technique. CNA hands back a fresh view of it, so
	// the projection matches that handle against the techniques it already
	// holds rather than keeping a second object for the same technique.
	current, err := resource.EffectCurrentTechnique()
	if err != nil {
		return nil, err
	}
	if current != nil {
		handle, handleErr := current.Handle()
		if handleErr != nil {
			return nil, handleErr
		}
		effect.currentTechnique = effect.techniqueForHandle(handle)
		// The view is CNA's own fresh handle for a technique this effect
		// already holds, so it is released as soon as it has been matched.
		// Keeping it would be one owned handle per effect that nothing ever
		// destroys, which CNA reports at game teardown.
		if err := current.Dispose(); err != nil {
			return nil, err
		}
	}
	return effect, nil
}

// techniqueForHandle matches a fresh CNA technique view against the ones this
// effect already holds. It is how CurrentTechnique keeps XNA's object identity
// over an ABI that reports handles.
func (e *Effect) techniqueForHandle(handle uint64) *EffectTechnique {
	if handle == 0 || e.techniques == nil {
		return nil
	}
	for _, technique := range e.techniques.techniques {
		if candidate, err := technique.view.Handle(); err == nil && candidate == handle {
			return technique
		}
	}
	// CNA answered a handle none of this effect's technique views carries,
	// which happens because every accessor mints a fresh handle. The FIRST
	// technique is what D3DX selects and what the reference's constructor
	// stores, so that is the identity used.
	if len(e.techniques.techniques) > 0 {
		return e.techniques.techniques[0]
	}
	return nil
}

// Parameters is Effect::get_Parameters, one `ldfld`.
func (e *Effect) Parameters() *EffectParameterCollection {
	if e == nil {
		return nil
	}
	return e.parameters
}

// Techniques is Effect::get_Techniques, one `ldfld`.
func (e *Effect) Techniques() *EffectTechniqueCollection {
	if e == nil {
		return nil
	}
	return e.techniques
}

// CurrentTechnique is Effect::get_CurrentTechnique, one `ldfld`.
func (e *Effect) CurrentTechnique() *EffectTechnique {
	if e == nil {
		return nil
	}
	return e.currentTechnique
}

// SetCurrentTechnique is Effect::set_CurrentTechnique:
//
//	Helpers.CheckDisposed(this, pComPtr);
//	if (value == null)
//	    throw new ArgumentNullException("value", FrameworkResources.NullNotAllowed);
//	if (value == _currentTechnique) return;
//	if (value._parent != this) throw new InvalidOperationException();
//	... end the active pass, select natively, store ...
//
// Three details a reader would otherwise guess wrong. Assigning the technique
// it already holds returns BEFORE the parent check, so a technique from another
// effect that happens to be current is not refused. The parent check throws a
// PARAMETERLESS InvalidOperationException -- no message and no resource string.
// And the null check loads NullNotAllowed into the two-argument
// ArgumentNullException, which DrawString's does not.
func (e *Effect) SetCurrentTechnique(value *EffectTechnique) error {
	if e == nil {
		return errEffectNil
	}
	if e.IsDisposed() {
		// self(), not clrTypeName(): the reference pushes `ldarg.0` into
		// Helpers::CheckDisposed, which names the RUNTIME type, and on a
		// BasicEffect that is BasicEffect. The bare receiver here is the base
		// half of a composed object and would always say Effect.
		return fmt.Errorf("%w: %s", errObjectDisposed, e.resource.self().clrTypeName())
	}
	if value == nil {
		return fmt.Errorf("%w: value: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if value == e.currentTechnique {
		return nil
	}
	if value.effect != e {
		return errSpriteInvalidOperation
	}
	handle, err := value.view.Handle()
	if err != nil {
		return err
	}
	if err := e.resource.nativeResource().SetEffectCurrentTechnique(handle); err != nil {
		return err
	}
	e.currentTechnique = value
	return nil
}

// Clone is Effect::Clone(), which builds a NEW Effect over a cloned D3DX effect
// and gives it its own reflected graph. CNA's cna_effect_clone does the same
// and the probe measured it answering an independent handle whose techniques
// and apply both work.
func (e *Effect) Clone() (EffectReference, error) {
	if e == nil {
		return nil, errEffectNil
	}
	// No dispatch to the derived half, for the reason recorded on
	// effectVirtuals: a consumer holding a BasicEffect calls BasicEffect::Clone
	// through Go's own method set, and the composed base is unreachable from
	// outside this package.
	return e.cloneBase()
}

// cloneBase is Effect::Clone's OWN body, reached non-virtually. It is what the
// protected clone constructor runs and what an Effect with no derived half
// answers.
func (e *Effect) cloneBase() (*Effect, error) {
	if e == nil {
		return nil, errEffectNil
	}
	if e.IsDisposed() {
		// The reference checks `ldarg.1`, the clone SOURCE, and the source is
		// this receiver -- so the type it names is the source's runtime type.
		return nil, fmt.Errorf("%w: %s", errObjectDisposed, e.resource.self().clrTypeName())
	}
	resource, err := e.resource.nativeResource().CloneEffect()
	if err != nil {
		return nil, err
	}
	return newEffect(e.resource.GraphicsDevice(), resource)
}

// OnApply is Effect::OnApply, the `protected internal virtual` hook whose base
// body is EMPTY -- `ret`, four bytes. EffectPass::Apply calls it before it
// begins the pass, and the six stock effects override it to push their state.
//
// It is projected as a method rather than an override point because CNA-Go
// projects no derived effect yet; when one arrives it will need the frame-hook
// rule's shape, and this is the base body that rule would run.
func (e *Effect) OnApply() error {
	if e == nil {
		return errEffectNil
	}
	if e.derived != nil {
		return e.derived.OnApply()
	}
	return nil
}

// clrTypeName is System.Object::ToString's answer for an Effect.
func (e *Effect) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.Effect" }

// bindDerivedEffect installs the CLR `this` for Effect's two virtual members.
// Every constructor of a type that composes an Effect calls it, and nothing
// else does.
func (e *Effect) bindDerivedEffect(derived effectVirtuals) {
	if e == nil {
		return
	}
	e.derived = derived
}

// bindDerived forwards the CLR `this` to the composed base.
func (e *Effect) bindDerived(derived graphicsResourceObject) {
	if e == nil || e.resource == nil {
		return
	}
	e.resource.bindDerived(derived)
}

// nativeResource is the one owned CNA effect handle. Unexported.
func (e *Effect) nativeResource() *interop.Resource {
	if e == nil || e.resource == nil {
		return nil
	}
	return e.resource.nativeResource()
}

// The nine inherited GraphicsResource members.

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (e *Effect) GraphicsDevice() *GraphicsDevice {
	if e == nil {
		return nil
	}
	return e.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (e *Effect) Name() string {
	if e == nil {
		return ""
	}
	return e.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (e *Effect) SetName(value string) {
	if e == nil {
		return
	}
	e.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (e *Effect) Tag() any {
	if e == nil {
		return nil
	}
	return e.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (e *Effect) SetTag(value any) {
	if e == nil {
		return
	}
	e.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (e *Effect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (e *Effect) ToString() string {
	if e == nil {
		return ""
	}
	return e.resource.ToString()
}

// AddDisposingHandler is add_Disposing.
func (e *Effect) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errEffectNil
	}
	return e.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (e *Effect) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errEffectNil
	}
	return e.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (e *Effect) DisposeByNone() error {
	return e.DisposeByBoolean(true)
}

// DisposeByBoolean is Effect::Dispose(bool). The reference releases the D3DX
// effect and its whole reflected graph goes with it; here the graph is a set of
// CNA view handles this object owns, so each is destroyed too -- children
// first, which is the order CNA does NOT require (a view outlives its effect,
// measured) and which is used anyway because it is the order that leaks nothing
// if CNA ever does.
func (e *Effect) DisposeByBoolean(disposing bool) error {
	if e == nil {
		return errEffectNil
	}
	var released error
	if !e.resource.IsDisposed() {
		var derived error
		if e.derived != nil {
			derived = e.derived.releaseDerivedNativeObjects()
		}
		released = errors.Join(
			derived,
			e.parameters.dispose(),
			e.techniques.dispose(),
			e.resource.releaseNativeObject(),
		)
	}
	baseErr := e.resource.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
