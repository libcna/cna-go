# Foundation 23 — the first event contracts and the EventArgs carriers

Foundation 23 spends the Foundation-22 event decision. It completes the five
types that decision unblocked and nothing else: no dependency rule was relaxed
and no protected partial was touched.

```text
Microsoft.Xna.Framework.IUpdateable                          7 Go identities
Microsoft.Xna.Framework.IDrawable                            7
Microsoft.Xna.Framework.GameComponentCollectionEventArgs     2
Microsoft.Xna.Framework.Graphics.ResourceCreatedEventArgs    1
Microsoft.Xna.Framework.Graphics.ResourceDestroyedEventArgs  2
```

Five types, 19 mapped Go identities.

## Reference authority

```text
Microsoft.Xna.Framework.Game.dll      b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
Microsoft.Xna.Framework.Graphics.dll  560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
```

Both re-verified by hash this session and read with `ikdasm`. Public surface
remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## The two component contracts

Re-derived from `Microsoft.Xna.Framework.Game.dll` rather than from memory. Both
properties in each contract are **get-only**: `GameComponent` and
`DrawableGameComponent` declare setters, but those are members of the class, not
requirements of the contract.

```go
type IUpdateable interface {
    Enabled() bool
    UpdateOrder() int32
    Update(gameTime GameTime)
    AddEnabledChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
    RemoveEnabledChangedHandler(subscription EventSubscription) error
    AddUpdateOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
    RemoveUpdateOrderChangedHandler(subscription EventSubscription) error
}
```

`IDrawable` is the same shape over `Visible`, `DrawOrder`, and `Draw`.

`GameTime` passes by value, from the existing deliberate exception that keeps a
callback snapshot from aliasing mutable native state.

### Why both contracts are infallible

The `managedInterfaces` evidence rule reads the boundary from the reference
implementor IL in the assembly that declares the interface. `Game.dll` declares
both and ships exactly one implementor of each:

```text
GameComponent::get_Enabled            ldarg.0; ldfld enabled; ret
GameComponent::get_UpdateOrder        ldarg.0; ldfld updateOrder; ret
GameComponent::Update(GameTime)       ret                      // code size 1
DrawableGameComponent::get_Visible    ldarg.0; ldfld visible; ret
DrawableGameComponent::get_DrawOrder  ldarg.0; ldfld drawOrder; ret
DrawableGameComponent::Draw(GameTime) ret                      // code size 1
```

Nothing in either contract reaches a device, an allocation, or native code, so
neither gains a synthetic error. Both are admitted to `classifiedInterfaces`
with boundary `PURE_MANAGED`.

That is deliberately a **different verdict from `IGameComponent`**, which stays
unclassified and keeps a fallible `Initialize`, because
`DrawableGameComponent::Initialize` resolves `IGraphicsDeviceService` out of
`Game.Services` and throws `InvalidOperationException` when it is absent. The
class does reach the graphics runtime — through `get_GraphicsDevice` and
`Initialize`, which these two contracts do not declare. The boundary is read per
contract, not inherited from a neighbouring contract on the same class.

The four event accessors per contract still carry an error. That error is the
settled event accessor projection, not evidence of a runtime boundary, so the
closure measurement records them in a separate `eventAccessors` count and leaves
`fallibleOperations` at zero. Reporting them as boundary operations would have
misrepresented an infallible contract as a runtime one.

## The three System.EventArgs carriers

Each is one or two managed fields behind get-only accessors, and each is
admitted to `pureManagedTypes`, so no accessor is fallible.

```go
// framework
func NewGameComponentCollectionEventArgs(gameComponent IGameComponent) *GameComponentCollectionEventArgs
func (a *GameComponentCollectionEventArgs) GameComponent() IGameComponent

// graphics — no constructor, deliberately
func (a *ResourceCreatedEventArgs) Resource() any
func (a *ResourceDestroyedEventArgs) Name() string
func (a *ResourceDestroyedEventArgs) Tag() any
```

### Construction visibility is preserved exactly

`GameComponentCollectionEventArgs`'s constructor is `public` in the reference,
so CNA-Go projects it. Its IL is `call EventArgs::.ctor` plus one `stfld` with
no validation, so a nil component is stored exactly as a null one is, and the
constructor cannot fail.

Both graphics carriers declare their constructors **`assembly`**:

```text
.method assembly hidebysig specialname rtspecialname
        instance void .ctor(object resource) cil managed
```

They are therefore not part of the public contract and CNA-Go invents no
constructor for them. Only an unexported construction path exists, which is what
a `GraphicsDevice` raising `ResourceCreated`/`ResourceDestroyed` would use. This
is the same nonpublic-construction rule `TouchCollection+Enumerator` already
follows, and it is why the two carriers are complete with 1 and 2 identities
rather than 2 and 3.

`ResourceDestroyedEventArgs` reports a destroyed resource's **name and tag**
rather than the resource, because by the time the event is raised the object is
gone. Its constructor's parameters are `(name, tag)` but its IL stores the tag
first; the observable result is identical and the order is mirrored anyway.

### The base relationship, not Go structure

All three derive from `System.EventArgs`. None of them embeds
`framework.EventArgs`: the CLR base survives as the Foundation-22 measured
relationship, so each is its own reference type, inherits no member, and is not
assignable to its base.

Completing them also generalised a latent defect in the class closure
measurement, which had hardcoded `System.Object` as the only acceptable base for
a CLR class. A class may now derive from any **MAPPED** BCL base, and the
closure records which relationship carried it:

```text
DIRECT     AudioListener, AudioEmitter, PresentationParameters,
           TouchCollection, TouchCollection+Enumerator, GameServiceContainer
EventArgs  GameComponentCollectionEventArgs, ResourceCreatedEventArgs,
           ResourceDestroyedEventArgs
```

An `UNDECIDED` base fails the closure, which is what stops a type from shipping
under a base relationship nobody has taken.

## What these types do not do

Completing them wires nothing up. CNA-Go has no `GameComponent`, no
`GameComponentCollection`, and no component loop, so nothing in the binding
calls `Update` or `Draw` or raises any of these events. `GraphicsDevice` remains
a protected partial that raises nothing, so nothing constructs either resource
carrier. The contracts are declarations that an external type can satisfy, and
the carriers are values an external type can receive.

## External conformance canary

The mandatory proof that the architecture works from outside the binding.
`tools/external_consumer` materialises `tools/external_consumer/testdata/eventcanary`
as **its own Go module** whose only requirement is
`github.com/openeggbert/cna-go`, replaced by an extracted source root, and runs
it with `GOWORK=off` and `GOFLAGS=-mod=mod` so no workspace file and no sibling
checkout can satisfy the import.

The canary declares `Rotator`, an external type that owns four private
`EventSource` fields, exposes only the projected accessors, and raises its own
events. Seven tests, zero failures:

```text
TestExternalTypeSatisfiesBothComponentContracts
TestExternalTypeRaisesItsOwnEvents
TestExternalDuplicateRegistrationsAreRemovedSeparately
TestExternalTokensDoNotCrossInstances
TestExternalHandlerFailurePropagates
TestExternalConsumerCannotRaiseThroughTheContract
TestExternalUseOfTheEventArgsCarriers
```

They prove interface satisfaction from outside, handler ordering, sender
identity, exact `EventArgs.Empty` pointer identity, duplicate subscription with
independent tokens, removing each duplicate separately, repeated and zero-token
removal, a token from one instance not disturbing another, handler failure
propagating to the external raiser with no later handler running and the state
change intact, and — the capability boundary — that a consumer holding only
`IUpdateable` has no projected way to raise, because the `EventSource` fields
are unexported.

## Verification

- 6 interface closures and 9 managed class closures, all `PASS`.
- 51 new mutation fixtures, inventory **386 → 437**: 22 interface fixtures
  (11 defects × 2 contracts, none skipped) and 29 class fixtures (14 defects ×
  3 carriers, 13 skipped as inexpressible and each counted).
- 11 new behavior corpus observations, **564 → 575**, zero failures.
- 5 new in-repo tests over the contracts and carriers, plus 7 external ones.

## Scoreboard

```text                        before    after
TARGET_TYPES                   113      118
TARGET_MEMBERS                1703     1722
TOTAL_DIAGNOSTICS              321      316
MISSING_TYPE                   144      139
MISSING_MEMBER                 177      177
COMPLETE_TYPES                 108      113
PARTIAL_TYPES                    5        5

mutation inventory             386      437
behavior corpus                564      575
```

Every mismatch, leak, allowlist, and unmeasured counter remains zero.
`MISSING_MEMBER` stays at 177: the five protected partial runtime types gained
no event, because an event that never fires is not implemented.

CNA ABI is unchanged and nothing was rebuilt.
