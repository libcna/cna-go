# Foundation 24 — System.IDisposable as a measured BCL relationship

Foundation 24 takes the second decision the Foundation-21 handoff named, and
takes it as narrowly as the evidence allows: `System.IDisposable` is registered
as a **measured relationship that adds no projected Go surface**. It creates no
type, no member, and no ownership model, and it completes no XNA type.

The same pass makes the whole non-XNA interface set measured rather than
silently skipped, closing a real hole: `buildMappedInterfacesAndWitnesses`
dropped any direct interface absent from the 257-type contract with a bare
`continue`, so a BCL interface could neither be accounted for nor noticed.

## Reference authority

```text
Microsoft.Xna.Framework.Game.dll  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
```

Re-verified by hash this session and read with `ikdasm`. Public surface remains
the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## The rule

> A non-XNA CLR interface an XNA type declares contributes **no projected Go
> surface of its own**.

This is not an assumption. In the pinned profile every such interface is
satisfied one of exactly two ways, and neither produces a Go identity:

1. **The XNA type already declares the interface's members publicly**, so the
   concrete method set is the whole projection. This is the rule Foundation 20
   settled for `IList<T>` and Foundation 21 for `IServiceProvider`, and it also
   covers `IEquatable<T>`, `IComparable<T>`, and the collection interfaces.
2. **The type implements the interface explicitly**, so the CLR member is
   private and is not public surface at all.

## The proof case

`GraphicsDeviceManager` is the one type in the profile that implements
`System.IDisposable` without publicly declaring `Dispose()`:

```text
.method private hidebysig newslot virtual final
        instance void  System.IDisposable.Dispose() cil managed
{
  .override [mscorlib]System.IDisposable::Dispose
```

It is an explicit interface implementation, so the public contract carries only
the protected `Dispose(bool)`. Had `System.IDisposable` been treated as adding
surface, `GraphicsDeviceManager` would have gained a parameterless `Dispose()`
the reference does not expose. `TestIDisposableAddsNoProjectedSurface` asserts
exactly that: the type projects exactly one `Dispose` identity and it takes the
`bool`.

The other 28 declaring types declare `Dispose()` publicly in their own right, so
it maps as an ordinary member under the existing overload and per-operation
fallibility rules.

## What is explicitly not created

```text
no Disposable / IDisposable Go interface
no Close alias
no io.Closer adaptation
no runtime.SetFinalizer, and no implicit GC disposal
no ownership wrapper
no Dispose synthesized from CLR ancestry
```

`inventedDisposalNames` names each of these, and the verifier rejects any of
them appearing as an exported member of, or an exported embedding in, a
projected type. None of the six names is an XNA identity anywhere in the
profile, which the positive test also asserts.

## The exhaustive interface table

Eight distinct non-XNA interfaces appear as direct interfaces in the profile.
All eight are declared `MAPPED_NO_SURFACE`, and an undeclared one is
`INTERFACE_MAPPING_MISMATCH`:

```text
                                        clrMembers  projected  declaring  projected
System.IEquatable`1                          1          0         40         32
System.IDisposable                           1          0         29          3
System.Collections.Generic.IEnumerable`1     1          0         12          0
System.Collections.Generic.IEnumerator`1     4          0          5          1
System.Collections.Generic.ICollection`1     7          0          1          1
System.Collections.Generic.IList`1           4          0          1          1
System.IComparable`1                         1          0          1          1
System.IServiceProvider                      1          0          1          1
```

`projectedMembers` is the measured claim, and it is zero for every row.

## What this does and does not unblock

Mapping the interface makes a dependency **syntactically** complete. It does not
make any type safe to implement. Regenerating the frontier moves three types out
of "blocked on an unmapped BCL shape", and all three remain blocked on runtime
behavior CNA-Go does not have:

```text
Audio.Cue                   BCL shape resolved; still XACT native
Audio.SoundEffectInstance   BCL shape resolved; still the native error-code boundary
Audio.AudioCategory         BCL shape resolved; still AudioEngine + XACT
```

Ownership and lifetime for `GraphicsResource`, `GraphicsDevice`, `Texture2D`,
`SpriteBatch`, `Effect`, and the audio types remain a per-type question this
relationship deliberately does not answer.

## GameWindow — inspected and deferred with evidence

The regenerated frontier surfaced exactly one candidate not already on the
deferral list. It was re-derived rather than assumed, and it is blocked:

```text
.class public abstract auto ansi beforefieldinit Microsoft.Xna.Framework.GameWindow
.method assembly hidebysig specialname rtspecialname instance void .ctor()
```

`GameWindow` is an **abstract** class with an `assembly` constructor and eight
public abstract members that have no body at all — `Handle`,
`AllowUserResizing` (both accessors), `ClientBounds`, `ScreenDeviceName`,
`CurrentOrientation`, `BeginScreenDeviceChange`, and
`SetSupportedOrientations` — plus a protected abstract `SetTitle` that the one
concrete accessor, `set_Title`, delegates to after its null check.

There is therefore no managed behavior to derive for the members that matter,
`Handle` is a real platform window handle, and CNA-Go has no window: `Game`
remains a protected partial with no `Window` property. Implementing it would
mean fabricating a window. Deferred in the same category as `Mouse` and
`DisplayMode`.

## Verification

Positive:

- `TestBCLInterfaceRelationshipsAreExhaustive` — all eight declared, each with a
  declaring type, `System.IDisposable` at exactly 29.
- `TestIDisposableAddsNoProjectedSurface` — every projected `Dispose` identity
  traces to a declared XNA member; `GraphicsDeviceManager` projects exactly one
  and it takes the `bool`; no forbidden disposal name is an adapter, a type, or
  a member.
- `TestBCLInterfaceMeasurementIsReported` — every row reaches the report with
  `projectedMembers == 0`.

Negative — 9 new mutation fixtures, inventory **437 → 446**: an invented `Close`
alias, an invented `Disposable` embedding, a framework-qualified `IDisposable`
embedding, an invented finalizer surface, an invented ownership wrapper, an
undeclared BCL interface, a dropped declared `Dispose`, a changed declared
`Dispose` fallibility, and a `Dispose` synthesized onto `Curve`, which declares
none and implements no `IDisposable`.

## Scoreboard — unchanged, deliberately

```text                        before    after
TARGET_TYPES                   118      118
TARGET_MEMBERS                1722     1722
TOTAL_DIAGNOSTICS              316      316
MISSING_TYPE                   139      139
MISSING_MEMBER                 177      177
COMPLETE_TYPES                 113      113
PARTIAL_TYPES                    5        5

mutation inventory             437      446
behavior corpus                575      575
```

Every mismatch, leak, allowlist, and unmeasured counter remains zero. CNA ABI is
unchanged and nothing was rebuilt.
