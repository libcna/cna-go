package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// EffectReference is the Go projection of a position whose CLR type is Effect.
//
// # Why Effect widens at RETURNS and not only at parameters
//
// The settled substitutable-base rule widens a base-typed PARAMETER and leaves
// a base-typed RETURN as the concrete pointer, recording the lost downcast.
// Foundation 76 made one exception, for System.Exception, on the ground that a
// base whose DERIVED TYPES are the point must widen at returns too or the
// derived identity is erased at every hand-back.
//
// Effect is the second such base, and the measurement is stronger than the
// exception hierarchy's. `Effect::Clone()` is declared to return Effect and is
// OVERRIDDEN by all five stock effects, each returning a new instance of its
// own class:
//
//	.method public hidebysig virtual instance class Effect Clone()
//	  newobj instance void BasicEffect::.ctor(class BasicEffect)
//
// A C# consumer writes `(BasicEffect)basicEffect.Clone()`, and that cast is the
// whole point of the member -- cloning a BasicEffect to get an Effect you can
// no longer set DiffuseColor on is not a useful operation. Returning the
// composed *Effect would hand back the BASE HALF of a BasicEffect with no path
// to the object that owns it, so the downcast would not merely be lost, it
// would be impossible. The rule's default is therefore unavailable here rather
// than merely inconvenient.
//
// # Why a consumer cannot forge one
//
// effectBase is unexported, so only this module can satisfy the interface. It
// is spelled with a `Base` suffix for the reason textureBase is: `*Effect` and
// every type that composes one already hold a FIELD named `effect`, and Go has
// one identifier namespace for fields and methods.
//
// The rest of the interface is Effect's whole public surface, so a consumer who
// takes an EffectReference can USE it without asserting -- and can still assert
// back to *BasicEffect when they want the derived half.
type EffectReference interface {
	// The Effect half of whatever the value is. Unexported, so the interface is
	// unsatisfiable outside this module.
	effectBase() *Effect

	// Effect's own declared surface.
	Parameters() *EffectParameterCollection
	Techniques() *EffectTechniqueCollection
	CurrentTechnique() *EffectTechnique
	SetCurrentTechnique(value *EffectTechnique) error
	Clone() (EffectReference, error)
	OnApply() error

	// GraphicsResource's, inherited by Effect and re-exposed by every type that
	// composes one.
	GraphicsDevice() *GraphicsDevice
	Name() string
	SetName(value string)
	Tag() any
	SetTag(value any)
	IsDisposed() bool
	ToString() string
	AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDisposingHandler(subscription framework.EventSubscription) error
}

// Disposal is deliberately NOT on the interface, and the reason is the
// contract's rather than a choice.
//
// Effect DECLARES the protected `Dispose(Boolean)` override, so it projects
// both DisposeByNone and DisposeByBoolean; BasicEffect declares no Dispose at
// all, so its inherited PUBLIC surface carries one Dispose that takes no
// argument. The two types therefore spell disposal differently because the
// pinned metadata spells it differently, and an interface member would have to
// pick one spelling and impose it on the other type.
//
// A consumer disposes a cloned effect through the type assertion they already
// perform to use it, which is the same assertion a C# consumer's cast is.

// effectBase makes an Effect its own reference.
func (e *Effect) effectBase() *Effect { return e }

// effectBase answers with the BasicEffect's composed base.
func (e *BasicEffect) effectBase() *Effect {
	if e == nil {
		return nil
	}
	return e.effect
}

// resolveEffect is the `ldarg` a CLR call site does for free. It answers nil for
// a nil interface AND for an interface holding a typed nil, because the
// reference sees one null either way.
func resolveEffect(reference EffectReference) *Effect {
	if reference == nil {
		return nil
	}
	return reference.effectBase()
}

// The compiler is the proof that both halves of the family satisfy the
// interface. A derived effect that forgot one forwarding member fails here.
var (
	_ EffectReference = (*Effect)(nil)
	_ EffectReference = (*BasicEffect)(nil)
)
