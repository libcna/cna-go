# Foundation 22 — the general CLR event architecture

Foundation 22 takes the decision the Foundation-21 handoff named as the
highest-value remaining blocker: the **event handler type**. It maps
`System.EventArgs` and `System.EventHandler<T>` as general BCL/language
projections, materialises the previously documented-only `EventSubscription`
adapter, adds a public `EventSource[T]` support primitive, and makes the CLR
base relationship a measured facet instead of a silently dropped one.

It completes **no XNA type**. Its whole product is language machinery plus
measurement, which is why every XNA counter is unchanged.

## Reference authority

```text
Microsoft.Xna.Framework.dll             38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
Microsoft.Xna.Framework.Graphics.dll    560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.Game.dll        b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
Microsoft.Xna.Framework.Input.Touch.dll b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25
Microsoft.Xna.Framework.Xact.dll        a14d5364dca7cf49fb90639e87ba04d52b59a700dc9198efa5707ce8eae28f0a
Microsoft.Xna.Framework.Video.dll       17538b1ca9d48a993e2cd88c96b436df08e7abb4aec5d4758eb21feb580d6e06
```

All six re-verified by hash this session and read with `ikdasm`. Public surface
remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## Why one decision covers the whole event surface

Every public CLR event in the profile — all **49** of them — is declared
`System.EventHandler`1<T>`, and only five distinct generic arguments occur:

```text
44  System.EventHandler`1[System.EventArgs]
 2  System.EventHandler`1[Microsoft.Xna.Framework.GameComponentCollectionEventArgs]
 1  System.EventHandler`1[Microsoft.Xna.Framework.Graphics.ResourceCreatedEventArgs]
 1  System.EventHandler`1[Microsoft.Xna.Framework.Graphics.ResourceDestroyedEventArgs]
 1  System.EventHandler`1[Microsoft.Xna.Framework.PreparingDeviceSettingsEventArgs]
```

There is no second delegate shape and no non-generic `EventHandler` anywhere in
the profile, so two adapters close the entire event surface.

## The two mapped BCL shapes

```text
System.EventArgs        -> *framework.EventArgs
System.EventHandler<T>  -> framework.EventHandler[T]
```

`EventHandler[T]` is `func(sender any, args T) error`. The generic argument is
carried exactly; the previous behavior — an undeclared `mapType` fall-through to
`any` — is gone, and degrading a handler is now a named event defect.

### Why the handler returns error

A CLR event handler returns `void`, but invoking one can throw, and CNA-Go
reports runtime failure through an `error` result rather than a panic. The
`error` is therefore a **`GO_LANGUAGE_PROJECTION` of the CLR exception
channel**, not an additional XNA return identity: no XNA event handler produces
a value.

### Why EventArgs is a pointer and Empty is shared

`System.EventArgs` is a CLR class, so it keeps reference semantics. `nil` is not
its `Empty`, and neither `any`, `struct{}`, nor `map[string]any` is admitted.

The reference never carries information in a bare `EventArgs`; every raise site
pushes the one shared static field, e.g. in `GameComponent::set_Enabled`:

```text
ldsfld   [mscorlib]System.EventArgs [mscorlib]System.EventArgs::Empty
callvirt instance void GameComponent::OnEnabledChanged(object, EventArgs)
```

So `System.EventArgs.Empty` projects as `framework.EventArgsEmpty()`, returning
one stable private singleton. It is a **function, not an exported variable**,
because an exported variable would let one consumer replace the shared identity
for every other consumer in the process.

`EventArgs` carries one unexported byte. That is a Go language necessity, not
invented CLR state: the Go specification permits two distinct zero-size
variables to share an address, and the gc runtime does exactly that. Measured on
go1.24.4 linux/amd64, four separate `new(struct{})` allocations all returned
`runtime.zerobase` — 6 collisions out of 6 pairs — while a one-byte struct gave
0. A zero-size `EventArgs` would make every instance pointer-equal to every
other and destroy the reference identity `Empty` depends on.

## The event accessor projection is unchanged

One CLR event still becomes exactly two projected Go accessors:

```go
Add<EventName>Handler(handler framework.EventHandler[T]) (framework.EventSubscription, error)
Remove<EventName>Handler(subscription framework.EventSubscription) error
```

Static events keep the declaring-type prefix. Nothing became a property, a
slice, a channel, or a `+=` helper, and the member-count formula is untouched.

One latent defect was fixed on the way: the projected `EventSubscription` was
spelled unqualified in **every** package. No event-bearing type is complete yet,
so it had never been observable, but a graphics-package event must read
`framework.EventSubscription`. Adapter qualification is now one helper used by
`EventArgs`, `EventHandler`, `EventSubscription`, and `Iterator<T>` alike.

## EventSubscription and EventSource

`EventSubscription` is an opaque token with no exported field, no native handle,
no `cgo.Handle`, and no `unsafe.Pointer`. `EventSource[T]` is **public** language
support with `Add`, `Remove`, and `Raise`, so a type declared outside CNA-Go can
implement an event-bearing XNA contract without inventing its own incompatible
token. The zero value is ready to use.

### Semantics read from the reference, not invented

The compiler-generated accessors in `GameComponent` and `DrawableGameComponent`
are the authority for two behaviors:

```text
add_EnabledChanged     -> Delegate.Combine(existing, value) + Interlocked.CompareExchange
remove_EnabledChanged  -> Delegate.Remove (existing, value) + Interlocked.CompareExchange
```

`Delegate.Combine` with a null operand returns the other operand, so adding a
null delegate to a CLR event is a no-op that does not throw — therefore
`Add(nil)` registers nothing and returns the zero token instead of leaving a nil
function to panic at dispatch. `Delegate.Remove` returns the invocation list
unchanged when the delegate is absent, so the zero token, an already-removed
token, and a token owned by another `EventSource` are all harmless.

### The one deliberate divergence

Go function values are not comparable, so CLR `Delegate.Remove` identity
matching cannot be reproduced. A token therefore names the **registration**, not
the handler: adding one handler twice produces two registrations and two
distinct tokens, and each is removed separately. This is recorded as a
`GO_LANGUAGE_PROJECTION` rather than presented as CLR-equivalent.

### Pinned dispatch semantics

| property | behavior |
| -------- | -------- |
| ordering | registration order |
| duplicates | permitted, independent identities |
| dispatch | over a snapshot taken under the lock |
| mutation during raise | affects later raises only, never the running one |
| lock scope | no internal lock is held while a consumer handler runs |
| failure | first non-nil error propagates, no later handler runs |
| after failure | registration list intact, next raise sees the same handlers |
| token lifetime | explicit removal only; no finalizer, no auto-unsubscribe |

A handler failure is never swallowed.

## CLR base types are now measured

Go has no CLR inheritance and CNA-Go never fakes one with exported embedding,
which would promote members the XNA contract does not declare. A non-XNA CLR
base survives as a measured relationship, and the table is **exhaustive over the
profile**: an undeclared base is `BASE_MAPPING_MISMATCH`, so nothing can be
dropped in silence.

```text
IMPLIED   System.Object                                      75 derived
IMPLIED   System.ValueType                                   56
IMPLIED   System.Enum                                        49
MAPPED    System.EventArgs            -> framework.EventArgs  4
DEFERRED  System.Exception                                    5
DEFERRED  System.Attribute                                    5
DEFERRED  System.Collections.ObjectModel.ReadOnlyCollection`1 4
DEFERRED  System.Runtime.InteropServices.ExternalException    3
DEFERRED  System.IO.BinaryReader                              1
DEFERRED  System.ComponentModel.ExpandableObjectConverter     1
DEFERRED  System.Collections.ObjectModel.Collection`1         1
DEFERRED  System.Collections.Generic.Dictionary`2             1
```

All 205 non-XNA-based types are accounted for by exactly one row. **DEFERRED**
means no derived type may be projected yet; projecting one anyway is a
diagnostic, which is what stops a type from shipping under a base relationship
nobody has decided about.

## What this unblocks

Regenerating the dependency frontier with the two BCL shapes mapped moves six
missing types from "blocked on an unmapped BCL shape" to dependency-complete:

```text
Microsoft.Xna.Framework.IUpdateable                         ready
Microsoft.Xna.Framework.IDrawable                           ready
Microsoft.Xna.Framework.GameComponentCollectionEventArgs    ready
Microsoft.Xna.Framework.Graphics.ResourceCreatedEventArgs   ready
Microsoft.Xna.Framework.Graphics.ResourceDestroyedEventArgs ready
Microsoft.Xna.Framework.GameWindow                          shape only
```

`PreparingDeviceSettingsEventArgs` remains blocked on
`GraphicsDeviceInformation`, and `IGraphicsDeviceService` remains blocked by the
protected partial `GraphicsDevice` its property names. Neither dependency rule
was weakened to improve the count.

## Verification

Positive:

- `TestEventProjectionIsMeasuredExactly` pins the projected signatures on both
  sides of the package-qualification rule, and sweeps the whole profile to prove
  49 events produce 98 accessors and that no handler anywhere degrades to `any`.
- `TestBCLBaseRelationshipsAreExhaustive` proves the base table covers every
  non-XNA base in the profile and that every declared base has a derived type.
- `TestBCLBaseRelationshipMeasurementIsReported` proves all 205 derived types
  are covered by exactly one relationship row.
- `TestEventAdapterSurfaceIsDeclaredLanguageSupport` proves the four adapters
  are registered, framework-package scoped, and collide with no XNA identity.

Negative — 39 new mutation fixtures, taking the inventory from **347 to 386**:

- 34 event fixtures: 17 defects × 2 owners, one in the framework package and one
  in a descendant package, so the qualification half of the rule is attacked as
  hard as the shape half. The defects are handler degraded to `EventHandler[any]`,
  handler erased to `any`, wrong generic argument, `EventArgs` by value, handler
  as a raw `func`, as a channel, as `unsafe.Pointer`, subscription token dropped,
  token as a native `uintptr`, removal taking the handler, removal taking a
  native handle, either error channel dropped, either accessor missing, a leaked
  CLR accessor name, and an accessor projected as a field.
- 5 base fixtures: exported embedding, framework-qualified embedding, a derived
  class projected as an interface, a deferred base projected anyway, and an
  undeclared base.

Runtime behavior — 11 direct tests, green under `-race`: shared `Empty`
identity, zero/one/many handlers, registration order, duplicate registrations
with independent tokens, removing the first and the second duplicate, repeated
removal, foreign-token removal, add and remove during dispatch, snapshot
behavior, failure in first/middle/last position with the list intact afterwards,
sender identity, exact args pointer identity, the `Add(nil)` no-op, and 1,600
concurrent add/raise/remove cycles across 8 goroutines.

## Scoreboard — unchanged, deliberately

```text                      before    after
TARGET_TYPES                  113      113
TARGET_MEMBERS               1703     1703
TOTAL_DIAGNOSTICS             321      321
MISSING_TYPE                  144      144
MISSING_MEMBER                177      177
COMPLETE_TYPES                108      108
PARTIAL_TYPES                   5        5

mutation inventory            347      386
behavior corpus               564      564
```

Every mismatch, leak, allowlist, and unmeasured counter remains zero, including
the now load-bearing `EVENT_MAPPING_MISMATCH` and `BASE_MAPPING_MISMATCH`. The
five protected partial runtime types are untouched, and `MISSING_MEMBER` stays
at 177 because no protected partial gained an event it cannot actually raise.

CNA ABI is unchanged: this milestone adds no native symbol, no callback, and no
layout. Nothing here was rebuilt.
