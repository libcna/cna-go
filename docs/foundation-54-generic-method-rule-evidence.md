# Foundation 54 — the generic-method projection rule

`Texture2D`'s six `SetData`/`GetData` overloads arrive, and with them the rule
that made them projectable at all. Six missing members close and two appear:
the type's declared surface is now complete, which is exactly what exposed a
second gap the verifier had been carrying.

`TOTAL_DIAGNOSTICS` drops from 212 to 208, and the profile's `!!0` tokens stop
being named by their position.

## Go methods cannot declare type parameters

```go
func (t *Texture2D) SetData[T any](data []T) error   // does not compile
```

That is a Go language rule, not a limitation of this binding, and no arrangement
of receivers gets around it. A CLR generic **instance** method therefore has no
method-shaped projection at all.

The settled response to "this member cannot be a method here" already exists in
the profile: the cross-package cycle rule turns such a member into a
package-level function whose **first parameter is the receiver**. A generic
method becomes the same shape:

```text
Texture2D::SetData<T>(T[])  ->  Texture2DSetDataBySliceOfT[T any](*Texture2D, []T) error
```

A generic **static** method already had that shape and keeps it, so the rule adds
one condition to an existing branch rather than a second convention.

### `!!0` is a position, not a name

The pinned contract spells a method type parameter as the IL token `!!0`, meaning
"this method's first type parameter". Its **name** is in the member's
`genericParameters`, and nothing resolved it: the overload-suffix builder
stripped the punctuation and produced

```text
Texture2D.SetDataBySliceOf0
```

a name for a position, on a member Go cannot declare, that no consumer could have
guessed. The suffix builder and the type mapper now both resolve the token
through the member's own declaration, and a token whose position the member does
not declare is left unresolved rather than renamed — so a contract defect stays
reportable instead of being disguised as a type called `0`.

## What Go cannot say about T

The CLR constrains it as `valuetype .ctor T` — a struct with a default
constructor. Go has no such constraint, so the projection is `[T any]` and the
element type is checked at **runtime**.

That is not a loosening of the contract. The CLR also fails at runtime for a
struct whose layout does not match the surface format, and CNA validates
type/format compatibility itself. What Go loses is the compile-time refusal of a
reference type, and it is recorded rather than papered over.

## The element mapping, and the size check that is the point of it

CNA declares **eighteen** `CNA_TEXTURE_DATA_*` element identities and every one
has a Go type — `framework.Color`, `byte`, `float32`, `uint16`,
`framework.Vector2`/`Vector4`, and the eleven `PackedVector` types. The mapping
is closed and total, so there is no XNA transfer this projection cannot express,
and a test asserts all eighteen are covered with no two sharing an identity.

The **size check** is the load-bearing half:

> CNA identifies an element by what it REPRESENTS, never by how large it is.

So a Go type whose layout drifted — a packed vector that gained a field, a
`Color` that stopped being four bytes — would be copied wholesale into a buffer
CNA reads with a different stride, and **nothing on either side would report
it**. Every transfer therefore checks `unsafe.Sizeof(T)` against the width its
identity means, and refuses by name rather than transferring. It is the same
class of defect the ABI layout probe exists for, on the Go side of the boundary
where that probe cannot see; four of CNA's packed-storage aliases are now
measured there too, so both sides of the width are pinned.

## A type may not be COMPLETE while members it INHERITS are absent

Closing `Texture2D`'s last declared members made it report **COMPLETE**, and a
Foundation 33 invariant caught it immediately:

```text
BASE_MAPPING_MISMATCH  Texture2D derives from a DEFERRED XNA base and is reported
                       COMPLETE, but it cannot be: the 2 public members it
                       inherits are not projected
```

Completeness was computed from a type's **declared** surface alone. `Texture2D`
inherits `LevelCount` and `Format` from `Texture`, whose relationship is DEFERRED
because `Texture` extends `GraphicsResource`, whose ownership is undecided —
so those two are projected nowhere.

The fix reports the gap where a reader looks for it: as the type's **own** missing
members. `Texture2D` stays PARTIAL and its inventory now names

```text
Microsoft.Xna.Framework.Graphics.Texture::Format()
Microsoft.Xna.Framework.Graphics.Texture::LevelCount()
```

The rule is deliberately narrow — it fires only for a type that would otherwise
be reported complete, because that is the only case where the omission misleads.
A type already PARTIAL is already reported as partial, and its inherited gap is
counted in `XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED` and named in its base's
recorded blockers. The hard failure that caught this stays in place; this is what
makes it unreachable.

## Evidence

The device-state scenario grows, still 20 isolated cycles:

```text
DEVICE_STATE_TEXTURE_TRANSFER_ROUND_TRIPS   60   three per cycle
DEVICE_STATE_TEXTURE_TRANSFER_REFUSALS      40   two per cycle
```

Each round trip **writes a pattern to the live texture and reads it back from
it**, texel by texel: the full surface through the one-argument overload, the
same surface through the windowed one, and a 2×2 rectangle through the
five-argument overload the other two funnel into. A projection that kept a
managed copy would pass a test that compared its own input; this compares what
CNA gives back.

The two refusals are an element type outside the eighteen, and a transfer window
that leaves the array — the bounds check that runs before the pointer is taken,
because handing CNA a start and a count that leave the slice is an address it may
read past.

Three verifier controls ship with the rule: the generic member's expected
identity is a package-level func whose first parameter is the receiver and whose
array is `[]T`; `!!0[]` resolves to `SliceOfT` **and** is left alone with no
declaration in scope; and `Texture2D` reports its two inherited members as
unprojected while a non-deferred base contributes none.

## ABI

```text                                        before      after
BOUND_FUNCTIONS                               76      78
PROTOTYPE_TYPE_POSITIONS                     243     256
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         152     164
native ABI mutation controls                  91      95
ABI_MISMATCHES / FINDINGS                      0       0
```

Four new controls. Two are prototype defects — the transfer passed by value where
CNA takes a pointer, and a `get_data` that drops the required-element count,
which would make an undersized destination unreportable. Two are layout defects
C cannot see: the transfer's `start_index` and `element_count` are **both
`uint64` and adjacent**, so swapping them turns "sixteen elements from zero" into
"zero elements from sixteen" — a transfer that succeeds and copies nothing — and
a widened `CNA_PackedBgr565` would stride twice as far through the caller's
array.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1931     1937
MISSING_MEMBER                                80       76
TOTAL_DIAGNOSTICS                            212      208
COMPLETE_TYPES                               120      120
PARTIAL_TYPES                                  5        5
UNEXPECTED_MEMBER                              0        0

behavior corpus                              684      687
external canary tests                         80       81
native stress scenarios                       13       13
native ABI mutation controls                  91       95
runtime capability rows                       54       55
```

`MISSING_MEMBER` falls by four rather than six: the six generic members close and
the two inherited ones are now reported, which is the milestone's second half
doing its job.

## What this milestone does not claim

- **The rule is settled; its other users are not projected.** Nine generic
  members remain in the profile — `GraphicsDevice`'s three `GetBackBufferData`
  and six `DrawUser*` overloads — and they wait on vertex buffers and a vertex
  declaration, not on this rule.
- **`T` is `any` and the CLR's constraint is not expressible.** A reference type
  is refused at runtime, by name, where C# refuses it at compile time.
- **Nothing here proves a rendered pixel.** What is proved is that a Color array
  written through the projection comes back from CNA's own texture unchanged,
  including through a window and a sub-rectangle.
- **`Texture2D` is still PARTIAL, and now says why**: the two public members it
  inherits from the deferred `Texture` are unprojected, and `Texture` waits on
  `GraphicsResource`, whose ownership is undecided.
