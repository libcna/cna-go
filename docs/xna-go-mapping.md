# Normative XNA to Go language mapping

This document is normative for CNA-Go's strict XNA surface. The machine-readable
counterpart is `tools/api_compat/mapping-rules.json`. A binding implementation
must not introduce a new spelling or signature exception without updating both
and adding a verifier mutation test.

## Namespace and package identity

An XNA namespace maps to the same case-preserving import-path suffix under this
module. The local package identifier is the lowercase final namespace segment.

| XNA namespace | Go import path | Package declaration |
|---|---|---|
| `Microsoft.Xna.Framework` | `github.com/openeggbert/cna-go/Microsoft/Xna/Framework` | `framework` |
| `Microsoft.Xna.Framework.Graphics` | `github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics` | `graphics` |
| `Microsoft.Xna.Framework.Input` | `github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input` | `input` |
| `Microsoft.Xna.Framework.Content` | `github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Content` | `content` |

The rule continues recursively for other XNA namespaces. An import alias such
as `xna` is chosen by the consumer and is not an API identity. Namespace
hierarchy is never represented by nested Go objects. CNA-Go does not invent a
public `CNA/Framework` layer.

## Types and BCL values

An XNA top-level type keeps its PascalCase name. A public nested CLR type is
flattened by concatenating declaring names: `TouchCollection.Enumerator`
becomes `TouchCollectionEnumerator`. Generic arity markers are removed. If the
result collides with a non-generic or different-arity identity, the generic
identity appends `Of` and its source generic parameter names, for example
`ContentTypeReader<T>` becomes `ContentTypeReaderOfT[T]` while the non-generic
type remains `ContentTypeReader`. `IPackedVector<TPacked>` becomes
`IPackedVectorOfTPacked[TPacked]`.

CLR structs map to Go structs and retain value-copy semantics. CLR classes map
to pointer facades over unexported state. CLR interfaces map to structural Go
interfaces when their contract can be represented. CLR enums map to named
`int32` types because all enums in the selected profile have `System.Int32`
underlying storage. Every literal has an explicit raw constant; `iota` is not
used. Flags types remain named integers and the verifier checks both their raw
values and `[Flags]` classification. A flags enum does not require a declared
zero literal: Go's zero value remains representable as raw zero, but the
mapping never synthesizes `None`, `Default`, or another name absent from CLR
metadata. Delegates map to named function types.

The core BCL projection is fixed-width: `Single` is `float32`, `Double` is
`float64`, signed and unsigned integers preserve their width, `Boolean` is
`bool`, `String` is UTF-8 `string`, and `IntPtr` is `uintptr` only where the XNA
public contract genuinely exposes an opaque architecture-width value. Native
pointers are never a substitute for `IntPtr`.

`System.TimeSpan` maps to CNA-Go's `TimeSpan`, an immutable-style value storing
signed 100-nanosecond ticks in an `int64`. It is deliberately not
`time.Duration`: Duration has nanosecond units and a materially smaller range.

CLR owner generic-parameter tokens are substituted before a Go signature is
formed. `!0` means the owner's first declared generic parameter, `!1` the
second, and so on. Thus the `!0` storage type in
`IPackedVector<TPacked>.PackedValue` maps to `TPacked`; the raw CLR token can
never appear in Go. Substitution recurses through the supported array,
nullable, and generic shapes. A malformed token or an index with no declared
owner parameter is a `GENERIC_MAPPING_MISMATCH`, not an `any` escape hatch.

`System.Nullable<T>` is a source identity retained by the verifier rather than
a fake public `System` package type. A nullable value input maps to `*T`, where
`nil` means null and a non-nil pointer supplies a value. A nullable return maps
to `(T, bool)`, where the Boolean is `hasValue`; the zero value of `T` is
unspecified when `hasValue` is false and must not be interpreted as a sentinel.
The same `(T, bool)` pair is appended for an `out Nullable<T>`. Nullable never
uses NaN, infinity, or another distinguished value to represent null.

`System.IO.Stream` maps at the first raw-content boundary to `io.Reader`.
Reading begins at the reader's current position and continues to EOF. CNA-Go
does not close the reader. The interop layer copies bytes into C-owned or
C-call-duration storage before native dispatch. A later API whose observable
contract requires seeking must use `io.ReadSeeker`; it may not silently widen
this rule for existing functions.

### BCL collection interfaces

Generic BCL collection interfaces do not create public `System.Collections`
packages and do not add XNA types. `ICollection<T>` projects to the ordinary
method set on the concrete XNA collection: `Count`, `IsReadOnly`, `Add`,
`Clear`, `Contains`, `CopyTo`, `Remove`, and the inherited enumeration route.
The verifier measures that method set against the source interface relationship.

`IEnumerable<T>`, non-generic `IEnumerable`, and `IEnumerator<T>` use one
measured Go-language adapter:

```go
type Iterator[T any] interface {
    Next() (value T, ok bool, err error)
}
```

An XNA `GetEnumerator()` therefore returns `Iterator[T]`. Each call creates a
fresh cursor in source order. The cursor does not expose or snapshot a backing
slice. If the collection changes after cursor creation, the next call returns
an error, reproducing versioned fail-fast enumeration without a Go panic. The
`bool` means that a value was produced; `error` remains a separate failure
channel. This cursor mapping preserves the module's Go 1.22 language baseline
while still representing mutation failure.

## Fields and properties

A public instance CLR field maps to an exported Go struct field where the type
is a value projection. Thus `Vector2.X` and `Vector2.Y` remain fields.

Go has no properties. An instance getter maps to a method with the property
name; a setter maps to `Set` plus the property name. A value-type setter has a
pointer receiver. A static getter maps to a zero-argument package function
prefixed by the declaring type, and a static setter adds `Set` before that
prefix. Examples are `(*Texture2D).Width()`, `(*SoundEffectInstance).Volume()`,
`(*SoundEffectInstance).SetVolume(float32) error`, `Vector2Zero() Vector2`, and
`ColorWhite() Color`. Functions, rather than exported mutable variables,
preserve read-only static-property semantics.

Static CLR constants map to package constants prefixed by the declaring type.
A non-constant static field uses getter/setter functions just like a static
property. The CLR enum implementation field `value__` has no Go member.

## Constructors, overloads, statics, and operators

A type with exactly one public constructor uses `NewType`. When a type has more
than one public constructor, every constructor—including the simplest—uses
`NewTypeBy<ParameterShape>`. This avoids an arbitrary privileged overload.
Parameter shapes concatenate mapped source types with `And`, use `None` for an
empty parameter list, and preserve `Ref`, `Out`, array, and generic shape.

The same rule applies to overloaded methods: every member in an overload group
adds `By<ParameterShape>`. A unique instance method keeps its source name. A
unique static method is a package function prefixed by its declaring type, such
as `MatrixCreateScale`. This rule is source-signature-derived and does not use
ad hoc semantic suffixes. It never uses `...any`.

Go uses one identifier namespace for fields and methods on a type and one for
all package declarations. If otherwise valid mapped identities collide, every
collider appends its source-kind word (`Field`, `Property`, `Method`,
`Constructor`, `Event`, or `Operator`). If that still collides, it appends
`Signature` plus the uppercase eight-digit FNV-1a-32 digest of the complete XNA
member identity. This final escape hatch is deterministic and verifier-derived;
it is never chosen manually.

Operators are distinct source members and never collapse into an ordinary XNA
method with the same semantic verb. They map to package functions named
`TypeOperator<OperatorName>By<OperandShape>`. For example, the two-vector
addition operator maps to
`Vector2OperatorAdditionByVector2AndVector2`. Ordinary XNA `Vector2.Add`
overloads map separately under `Vector2AddBy...`. This spelling is verbose by
design: it is stable, reconstructible, and one-to-one.

## Results, exceptions, ref, and out

Native resource/runtime failures return `error`; they never panic. A void-like
fallible operation returns `error`, and a value-producing fallible operation
returns `(T, error)`. Error is a language-added result and does not count as an
extra XNA member. Pure value operations do not add an error unless authoritative
XNA behavior exposes an argument failure that cannot be represented by the
input type. The same rule applies to completely managed class facades: an
invalid collection index, nil key insertion, insufficient `CopyTo` storage,
nil `CompareTo` operand, or invalid tangent key index returns `error`; it does
not leak a Go slice-bounds or nil-pointer panic. Panics are reserved for
violated internal invariants and are contained at every C callback boundary.

A native-backed class owner does not by itself make every property fallible.
When direct reference IL proves that a member only reads or writes managed
state, that member gains no synthetic `error` and does not enter the native
boundary. `GraphicsDeviceManager.SupportedOrientations` is the first measured
case: its getter returns one stored `DisplayOrientation`, while its setter
stores the exact bits and marks private future device configuration dirty.
Construction and disposal remain fallible native operations, but this stored
property remains managed before and after native resource disposal.

## Pure managed CLR classes

CLR `class` is not evidence of native backing. A class is classified as *pure
managed* when authoritative Microsoft XNA IL proves that its selected public
behavior is backed entirely by managed fields and deterministic managed code,
and therefore owns no CNA native object and needs no FFI, no native allocation,
no renderer or device query, no native destruction, no callback registration,
no thread-affinity lifecycle, and no external hardware state. Anything short of
that proof keeps the fallible native facade, so `Game`, `GraphicsDeviceManager`,
`GraphicsDevice`, `SpriteBatch`, and `Texture2D` are deliberately excluded.

Classification changes fallibility, never semantics. An admitted class is still
a CLR reference type: its constructor returns `*T`, and two variables holding
one instance observe the same mutations. It never becomes a copied Go value.

`Audio.AudioListener` and `Audio.AudioEmitter` are the first classes admitted
on this general rule rather than because they are value types.

The classification spans both CLR kinds. Admitting a CLR *struct* changes only
its fallibility, never its value semantics: `TouchCollection` is admitted so
that its measured argument validation and its unconditional
`NotSupportedException` mutators can be projected honestly, and it remains a
copied Go value struct.

## System.IntPtr and the raw-handle rule

`System.IntPtr` projects to Go `uintptr`. That is a **language projection of an
opaque pointer-sized bit value** the public XNA contract carries at that
position, and nothing more. It does not mean the value may be dereferenced,
that it is a CNA or SDL handle, that a window exists, that `unsafe.Pointer` is
admissible anywhere, or that native device creation is authorized.
`IntPtr.Zero` is `uintptr(0)`. Because the values are opaque handles and bit
patterns for this binding, no signed numerical ordering is claimed for them
even though CLR `IntPtr` is signed.

The `RAW_HANDLE_LEAK` gate is narrowed to match, and only that far. A public
`uintptr` is admitted **only** at a signature position where the authoritative
XNA metadata declares `System.IntPtr`. Six CLR members do:
`GameWindow.Handle`, `GraphicsAdapter.MonitorHandle`, the `GraphicsDevice.Present`
overload taking `overrideWindowHandle`, `PresentationParameters.DeviceWindowHandle`,
`Mouse.WindowHandle`, and `TouchPanel.WindowHandle`.

Everything else still leaks: a `uintptr` the source never declared, one that has
drifted between parameter and result or between indices, one inside a slice or
pointer at an unadmitted position, one on a member the profile does not declare
at all, an exported named type defined over `uintptr`, a public
`unsafe.Pointer`, a leaked CNA or SDL handle, and any implementation-only
native pointer or internal type in public API.

## Fallibility is per operation

An `error` result belongs to one projected operation, not to a type and not to
a property. A constructor, an ordinary method, a property getter, and that same
property's setter are each classified independently, so a property may project
as

```go
func (x *AudioEmitter) DopplerScale() float32
func (x *AudioEmitter) SetDopplerScale(value float32) error
```

when the reference getter is one field read and only the setter validates. The
reverse pairing is equally expressible. A getter never gains an `error` because
its setter validates, and an unrelated member never gains one because a sibling
throws.

The mapping keys follow that scope: `constructor|Name`, `method|Name`,
`field|Name`, `property-get|Name`, `property-set|Name`, and `property|Name`.
The last one marks both accessors and is correct only where the reference IL
validates on read *and* on write, as `CurveKeyCollection`'s indexer does. Using
it for a property that only validates on assignment is a measured defect: the
verifier compares each accessor separately and names the accessor and the
direction of the disagreement.

An `out T` parameter is removed from the input list and appended to Go results.
For the conventional `TryX` pattern the value precedes the final `bool`. A
mutable `ref T` input maps to `*T`. Source direction remains in the overload
shape so `ref`, `out`, and value overloads cannot collide. No rule exposes an
unsafe or C pointer.

Nullable presence, source Boolean returns, `out` values, and Go failures remain
separate result channels. Thus a fallible nullable-returning projection, if one
is introduced by an authoritative runtime boundary, returns `(T, bool, error)`:
the first Boolean is nullable `hasValue`, while `error` reports genuine
failure. An `out Nullable<T>` still retains `OutNullableOfT` in its deterministic
overload shape even though its Go result expands to two values.

Interface kind alone does not make an operation fallible. Interface identities
are explicitly classified by their execution boundary, and the boundary is read
from the reference implementor IL in the assembly that declares the interface,
never from speculation about an implementor that does not exist. A
native/runtime interface may retain a final `error`; a pure managed value
interface does not gain one merely because its owner is an interface.

Because fallibility is per operation, a single contract may legitimately mix
the two. `IEffectMatrices` is uniformly managed: all five shipped implementors
back `World`, `View`, and `Projection` with a managed field access plus a dirty
flag, so none of its six operations is fallible. `IEffectFog` is not: those
same implementors back `FogEnabled`, `FogStart`, and `FogEnd` the same managed
way but route `FogColor` through `EffectParameter`, which calls unmanaged D3DX
and throws on a failed HRESULT, so exactly its two `FogColor` operations are
fallible and its six siblings are not.

`IGameComponent` and `IGraphicsDeviceManager` are unclassified and keep the
runtime default in which every operation is fallible, which is correct for
contracts whose whole purpose is to reach the game and device runtime.
`IGraphicsDeviceManager.BeginDraw` shows the channel rule: its source Boolean
says whether drawing may proceed and stays distinct from the `error`, which
says whether the call itself failed.

Both PackedVector interfaces are pure managed value interfaces, so their mapped
contracts are:

```go
type IPackedVector interface {
    ToVector4() framework.Vector4
    PackFromVector4(framework.Vector4)
}

type IPackedVectorOfTPacked[TPacked any] interface {
    IPackedVector
    PackedValue() TPacked
    SetPackedValue(TPacked)
}
```

This is an explicit reusable classification; it does not remove error results
from unrelated interfaces.

## Collections and enumerators

`System.Collections.Generic.IList<T>` projects the same way as `ICollection<T>`:
a concrete Go method set on the XNA collection, with no fabricated BCL package.
It needs nothing extra, because the indexer and index methods it adds are
already declared public members of the XNA collection and map as ordinary
members.

A collection that declares its own public enumerator type projects that type
from `GetEnumerator`. The `Iterator<T>` adapter is for collections that declare
none. `CurveKeyCollection` uses the adapter; `TouchCollection` declares
`TouchCollection.Enumerator` and so projects
`GetEnumerator() TouchCollectionEnumerator`.

A read-only collection keeps its mutators. `TouchCollection` declares the whole
`IList<T>` surface, and the reference implements every mutator as an
unconditional `NotSupportedException`. Those throws project as errors rather
than being dropped: a caller that mutates a read-only view has a real bug, and
silently accepting the call would hide it.

A BCL interface whose members the XNA type already declares publicly adds no
projected surface and needs no separate Go interface. `TouchCollection`'s
`IList<TouchLocation>` and `GameServiceContainer`'s `System.IServiceProvider`
are both this shape: every member of the declared interface is already an
ordinary public member of the class, so the concrete method set is the whole
projection.

## Mutable struct interface projection

A mutable CLR struct remains a Go value struct. Its non-mutating operations use
value receivers, while property setters and other state-changing interface
operations use pointer receivers. Consequently the verifier requires `*T`, not
`T`, to implement the exact constructed mutable interface. For every concrete
PackedVector format, compiler `go/types` evidence proves that `*T` implements
`IPackedVectorOfTPacked[exact fixed-width type]` and transitively
`IPackedVector`, while `T` retains ordinary value-copy semantics and does not
implement the mutable method set.

CLR explicit interface implementations are not always ordinary public declared
members, but Go structural interfaces require exported witness methods. Such a
method is admitted only when an actual mapped direct interface forces its exact
name and signature and the source type does not already declare the public
member. Witnesses remain compiler-extracted and are reported separately; they
do not increase `REFERENCE_MEMBERS`, `EXPECTED_GO_MEMBERS`, or
`TARGET_MEMBERS`. A missing or wrong witness is an interface/signature
diagnostic. An arbitrary extra or a claimed witness not required by the mapped
interface remains `UNEXPECTED_MEMBER`; there is no witness allowlist.

## Inheritance and Game

Go embedding is not CLR inheritance and does not provide virtual override
dispatch. XNA class facades therefore do not claim inheritance through
embedding. Base and interface relationships remain verifier facets: the
verifier checks the selected kind, forbids exported implementation embedding,
and measures every source relationship under this adaptation.

`Game` uses the explicit host-plus-callback projection:

```go
type Game struct { /* private native host state */ }

type GameCallbacks interface {
    Initialize(*Game) error
    LoadContent(*Game) error
    Update(*Game, GameTime) error
    Draw(*Game, GameTime) error
    UnloadContent(*Game) error
}

func NewGame(callbacks GameCallbacks) (*Game, error)
func (g *Game) Run() error
func (g *Game) Exit() error
```

`Game` is the XNA class identity and owns the native lifecycle. `GameCallbacks`
is a measured Go language adapter for protected virtual overrides, not a second
XNA type. A consumer does not embed `Game` to obtain fake dynamic dispatch.
`Run` locks its goroutine to one OS thread before native creation and retains
that lock through callback unregistration and destruction. Callbacks receive
the host explicitly so `Exit`, later services, window state, and parent-owned
facades have one unambiguous runtime.

The root XNA namespace and a child package can otherwise form an illegal Go
import cycle: graphics methods consume root values while `Game` or
`GraphicsDeviceManager` returns a graphics type. Such descendant returns use a
single cycle-cut rule: the getter is projected in the descendant package as a
package function named `OwnerTypeMember`, taking the ancestor facade. Private
interop registries transfer the association; no handle appears publicly. This
is reported as a language mapping limitation, not disguised as CLR inheritance.

## Events and callback identity

An XNA event maps to `Add<EventName>Handler` and
`Remove<EventName>Handler`. Addition returns an opaque `EventSubscription`;
removal accepts that token. Static events prefix the declaring type. This
preserves duplicate subscriptions, deterministic removal, callback order, and
self-removal without comparing Go function values. Named CLR delegate types map
to named Go function types. Native registration remains private and uses
`runtime/cgo.Handle`; an event token is never a native handle.

Every one of the 49 public CLR events in the profile is declared
`System.EventHandler`1<T>`, so two BCL shapes carry the whole event surface:

```text
System.EventArgs        -> *framework.EventArgs
System.EventHandler<T>  -> framework.EventHandler[T]
```

`EventHandler[T]` is `func(sender any, args T) error`. The generic argument is
mapped exactly — degrading it to `any`, to a bare `func`, to a channel, or to a
callback word is `EVENT_MAPPING_MISMATCH`. The `error` result is a Go language
projection of the CLR exception channel, not an extra XNA return identity: no
XNA event handler produces a value.

`System.EventArgs` is a CLR class, so it keeps reference semantics and projects
as a pointer. `nil` is not its `Empty`. `System.EventArgs.Empty` projects as
`framework.EventArgsEmpty()`, one stable private singleton behind a function so
no consumer can reassign the shared identity — the reference raises it as
`ldsfld System.EventArgs::Empty`, so shared object identity is the faithful
projection. `EventArgs` carries one unexported byte because Go permits distinct
zero-size variables to share an address, and the gc runtime does: four separate
`new(struct{})` allocations measured on go1.24.4 all returned
`runtime.zerobase`.

`EventSource[T]` is public language support with `Add`, `Remove`, and `Raise`,
so a type declared **outside** CNA-Go can implement an event-bearing XNA
contract without inventing its own incompatible token. Its semantics are pinned:
registration order is preserved, duplicates are allowed, dispatch runs over a
snapshot taken under the lock so mutation during a raise affects only later
raises, no internal lock is held while a consumer handler runs, the first
non-nil handler error propagates and stops the dispatch, and the registration
list survives that failure.

Two behaviors are read from the reference accessors rather than invented.
`add_X` calls `Delegate.Combine(existing, value)`, which returns the other
operand when one is null, so `Add(nil)` registers nothing and returns the zero
token instead of panicking. `remove_X` calls `Delegate.Remove`, which returns
the list unchanged when the delegate is absent, so the zero token, an
already-removed token, and a token owned by another event are all harmless.

One difference is a deliberate Go language projection and is recorded as such:
Go function values are not comparable, so a token names the **registration**
rather than the handler. Adding one handler twice therefore produces two
registrations and two distinct tokens, where CLR `Delegate.Remove` would match
by delegate identity.

### A native signal behind a projected event

Three of `Game`'s four events are raised by `GameHost` in the reference, and the
native CNA runtime plays `GameHost`'s part. Those signals are bound to the
already-published canonical C API rather than re-raised in Go, and the binding
follows one rule:

> exactly **one** native subscription per event per Game, never one per
> consumer handler.

Ordering is why. CNA invokes multiple native registrations on one event in
REVERSE registration order, measured against the pinned artifact, so registering
one native callback per Go handler would silently invert the dispatch order
`EventSource` promises. Routing every handler through one `EventSource` makes the
question moot, and it also removes the native API's owner-thread affinity from
the projected accessors: `Add` and `Remove` touch a mutex-guarded Go list and
never reach C, so a consumer may subscribe from any goroutine even though
`cna_game_subscribe` itself answers `CNA_RESULT_THREAD` from any thread but the
owner.

The subscription is installed eagerly when the native game is created — the
point at which the reference's `EnsureHost` subscribes — and released only after
the native game is destroyed, because the disposal signal is raised from inside
that destruction and a registration handle stays valid across it.

`CNA_GameEventCallback` returns `void`, so this boundary has no result channel:
a handler failure cannot stop the game the way a lifecycle callback can. It is
recorded through the same callback-failure path every lifecycle failure uses and
surfaces from `Game.Run`, so nothing is discarded; a panic is recovered in the
trampoline and recorded the same way. Nothing crosses the C frame.

The sender a raise pushes is read from IL and is not uniform.
`Game::OnActivated` and `OnDeactivated` accept a sender, ignore it, and raise
with `this`; `Game::OnExiting` pushes `ldnull` instead, so an `Exiting` handler
receives a **nil** sender. `Game::Disposed` has no protected raise method at
all — `Dispose(bool)` invokes the delegate field directly.

### Protected virtuals outside the callback contract

`GameCallbacks` projects exactly five protected virtuals. Every other protected
virtual projects as an ordinary method on its declaring type whose body is the
reference base body — the same rule that makes `GameComponent.OnEnabledChanged`,
`GameComponent.Initialize` and `Game.BeginRun` methods rather than contract
members.

`Game.BeginDraw` is the one with a value channel:

```go
func (g *Game) BeginDraw() (bool, error)
```

The Boolean is the frame's drawing decision, not a success flag: `DrawFrame`
runs `if (BeginDraw()) { Draw(); EndDraw(); }`, so a false answer skips both.
Collapsing it into the error would destroy a channel the reference has, so the
two stay separate and a refused call answers `(false, err)`.

## CLR interfaces from the BCL

A non-XNA interface an XNA type declares contributes **no projected Go surface
of its own**. In the pinned profile each is satisfied one of two ways, and
neither produces a new Go identity: either the XNA type already declares the
interface's members publicly, so the concrete method set is the whole
projection, or it implements the interface explicitly, so the CLR member is
`private ... .override` and is not public surface at all.

The table is exhaustive over the profile's eight non-XNA direct interfaces —
`IDisposable`, `IEquatable<T>`, `IComparable<T>`, `IServiceProvider`,
`IEnumerable<T>`, `IEnumerator<T>`, `ICollection<T>`, `IList<T>` — and an
undeclared one is `INTERFACE_MAPPING_MISMATCH`.

`System.IDisposable` is the case that most invites invention, so the rule is
stated explicitly: it creates **no** `Disposable` or `IDisposable` Go interface,
**no** `Close` alias, **no** `io.Closer` adaptation, **no**
`runtime.SetFinalizer`, and **no** ownership wrapper. Twenty-eight of its
twenty-nine declaring types declare `Dispose` publicly in their own right, and
it maps as an ordinary member with its own fallibility.
`GraphicsDeviceManager` is the twenty-ninth and the proof: it implements the
interface explicitly, so its `Dispose()` is not public surface and nothing is
projected for it — its only projected `Dispose` is the `Dispose(bool)` its
public contract declares. A `Dispose` is never synthesized from CLR ancestry.

Mapping the interface makes a dependency syntactically complete. It does not
decide native ownership or lifetime for `GraphicsResource`, `GraphicsDevice`,
`Texture2D`, `SpriteBatch`, `Effect`, or any audio type; that stays a per-type
question this relationship does not answer.

The same accounting covers XNA-namespaced interfaces that are **not** public
contract types. `Graphics.IGraphicsResource` and
`Graphics.IDynamicGraphicsResource` are both `.class interface private` —
assembly-visible device-loss and content-loss plumbing declared by seven and
four public graphics types. They contribute no public surface for a stronger
reason than a BCL interface does: they have no public member to project at all.
They are declared `INTERNAL_NO_SURFACE` rather than skipped, so the dependency
frontier does not count them as unmapped names, and an XNA interface that is
neither a public contract type nor a declared internal one is
`INTERFACE_MAPPING_MISMATCH`.

## CLR base types

Go has no CLR inheritance and CNA-Go never fakes one with exported embedding,
which would promote members the XNA contract does not declare. A non-XNA CLR
base survives as a **measured relationship**: the derived class projects as its
own pointer reference type and the base contributes no Go member identity.

The relationship table is exhaustive over the profile, so a base nobody has
decided about is `BASE_MAPPING_MISMATCH` rather than a silent omission. Three
CLR roots are implied by the existing projections (`System.Object`,
`System.ValueType`, `System.Enum`); `System.EventArgs` is mapped to the
`framework.EventArgs` adapter; `Collection<T>` is **composed** (below); and
seven remain deferred as open public-API decisions — `System.Exception`,
`ExternalException`, `System.Attribute`, `System.IO.BinaryReader`,
`ExpandableObjectConverter`, `ReadOnlyCollection<T>`, and `Dictionary<K,V>`. A
**deferred** base means no derived type may be projected yet, and projecting
one anyway is a diagnostic.

### BCL types at signature positions

A BCL type the contract carries in a **public signature** needs a public Go
spelling, or the member that returns it cannot be projected at all. That is the
footing `System.TimeSpan` and `System.EventHandler<T>` already have, and
`System.Collections.ObjectModel.ReadOnlyCollection<T>` joins them as
`*framework.ReadOnlyCollection[T]`.

This is a **different role** from base composition and neither implies the
other. A base adapter is private machinery a derived type composes and
forwards; a signature adapter is public because a projected member returns one.
`ReadOnlyCollection<T>` holds both roles independently: supported as a
signature adapter, still deferred as a base.

A signature adapter's exported surface is pinned to the exact public CLR member
inventory, so it is not a hole in the unexpected-member scan. For
`ReadOnlyCollection<T>` that is six members, and read-only needed no new
decision: every mutator is a private explicit implementation, which the settled
BCL-interface rule already excludes.

Two semantics are the list's rather than the view's, and both are measured. The
view **stores** the list rather than copying it, so read-only means no public
mutation through that surface and not frozen data. And enumeration forwards, so
an array-backed view is deliberately **not** version-checked — the reference
array enumerator has no version to check — while a `List<T>`-backed one keeps
its fail-fast behavior.

### BCL base-class composition

A **supported BCL collection base** is the one exception to "the base
contributes no Go member identity", and it is an exception for a reason the
other statuses do not face: a public member inherited from such a base is
**still public CLR surface**. Projecting only the members the XNA assembly
declares would leave `GameComponentCollection` with a constructor, four
protected overrides and two events — a collection nothing can be added to.

The projection is composition plus measured forwarding:

```go
type GameComponentCollection struct {
    base collectionBase[IGameComponent]   // unexported, not embedded
    ...
}
```

The adapter is implementation machinery. It is not an XNA type, not an exported
field, not a public base-class object, not an embedded public API, and not a
native handle, and the verifier rejects each of those shapes — including
embedding the unexported adapter, whose promotion would publish forwarding
nobody measured. The base's protected virtuals become an **unexported** Go
interface, so only a type declared in this module can supply or reach a hook,
and every mutating public operation routes through it so a subclass override
always runs.

Which inherited members are projected is measured, not guessed. Constructors
are not inherited; `family` members are not public; and an explicitly
implemented interface member is not public surface at all, so
`Collection<T>`'s `IsReadOnly`, `IsSynchronized`, `SyncRoot` and `IsFixedSize`
are **absent by rule** and the verifier checks their absence. Inherited members
run through the identical naming, overload, direction and fallibility
machinery, so they take part in ordinary collision resolution: the declared
protected `SetItem` override and the inherited `Item` setter both spell
`SetItem`, and the settled rule resolves them to `SetItemMethod` and
`SetItemProperty`.

Inherited behavior is read from the exact .NET Framework BCL the pinned XNA
assemblies bind against — mscorlib 4.0.30319.1, SHA-256
`5634668d…d98acc63` — never from modern .NET and never from Go convention. A
family whose exact behavior cannot be established stays deferred.

Provenance is tracked per member. `REFERENCE_MEMBERS` keeps naming exactly what
the Microsoft assemblies declare and is never inflated with inherited mscorlib
members; inherited projections are counted separately, and the two partitions
are disjoint and exhaustive, so no member is ever counted twice.

### XNA-to-XNA base composition

An XNA base is a type **inside** the contract that CNA-Go already projects, and
whose public surface the contract does not redeclare on the derived type. Three
are composed: `GameComponent`, `GraphicsResource`, and `Texture`.

The shape is the BCL one plus three rules the BCL case never needed.

**The inherited set is decided by CLR SLOT, not by name.** A derived class
excludes an inherited member only when it declares one with the same kind, name,
generic arity and parameter list. `DrawableGameComponent` and `Texture2D` both
override the **protected** `Dispose(bool)` and both inherit the **public**
`Dispose()`; a name-keyed exclusion deleted that member from their projected
surface, and in `DrawableGameComponent`'s case it was the member `Game.Dispose`
looks for.

**Inherited and declared members share one overload namespace.** A derived class
that declares one overload of an inherited name does not own the name, so the
group is sized over the effective method set and every member of a group larger
than one carries `By<ParameterShape>`. That is why `Texture2D` spells
`DisposeByNone`/`DisposeByBoolean` while `Texture` — which overrides nothing —
spells the same inherited member `Dispose`.

**Fallibility follows the DECLARING type.** Whether a member reaches a runtime
boundary is a property of its own body, so a base member classified managed
stored stays managed stored on every derived type, without being registered once
per derived type.

#### The CLR `this` under composition

`ldarg.0` in a base body is the **whole** object. Composition splits it, so a
composed base holds an unexported reference to the outermost derived object,
installed by the derived constructor and read only through an unexported
accessor. It is used wherever the reference uses `ldarg.0` as an **object** —
an event sender, a collection identity, `ToString`'s runtime type — and never
where `ldarg.0` merely reaches a field.

Nothing else changes: still no embedding, no exported base field, and no public
`Base`, `Parent` or `As<Base>` accessor. The mechanism is what makes private
composition **correct** rather than merely private, and the verifier holds it
from the Go syntax tree: every recorded identity site must reach the accessor
exactly as many times as the reference pushes `ldarg.0` as an object there, and
every projected derived constructor must install it. A base that is itself
composed over another — `Texture` — holds no copy and **forwards** instead:
one CLR `this`, one place that answers with it.

A derived override must forward to **its own** body, not the base's, wherever
the reference uses `callvirt`. `GraphicsResource::Dispose()` is
`callvirt Dispose(bool)`, so `Texture2D.DisposeByNone` calls
`Texture2D.DisposeByBoolean`; calling the composed base's would reproduce the
base's slot and leak the native texture. No managed observable distinguishes the
two, so that one is proved natively.

### CLR base substitutability at a parameter position

A **parameter** position whose CLR type is a class with a **LIVE**
substitutability requirement projects to an exported interface named
`<GoName>Reference` with an **unexported** method. Nameable by a consumer,
satisfiable only inside the module.

A requirement is LIVE when both ends exist: a projected carrier names the base
in a public signature, and at least one derived type is projected. Foundation 40
measured every family LATENT or NONE; Foundation 58 projected `RenderTarget2D`
and `Texture2D` became the first LIVE one, at seven positions — all
`SpriteBatch.Draw`'s `texture`. The registry and the measurement are
cross-checked in both directions.

Returns and property getters keep the concrete pointer.
`Texture2D::FromStream` returns a Texture2D and a caller uses every Texture2D
member on it; an interface there would take those members away to solve a
problem returns do not have.

An inherited member whose Go projection is already a **package function** — a
static, or a generic instance method under the generic-method rule — is not
re-projected on the derived type. Its Go identity names its declaring type, and
its receiver-first parameter is itself a parameter position carrying the
interface, so `Texture2DSetDataBySliceOfT(renderTarget, pixels)` already works.

Go has two nil shapes at an interface position — a nil interface, and a non-nil
interface holding a typed nil — and the CLR has one null. Both reach the same
`ArgumentNullException`.

## Structural counts

The 257 reference types map one-to-one to 257 expected Go XNA types. Member
projection removes 49 enum backing fields, expands 840 properties into 1,119
accessors, and expands 49 events into 98 add/remove accessors:

```text
XNA-declared        = 2964 - 49 - 840 + 1119 - 49 + 98 = 3243
BCL-inherited                                          =   12
EXPECTED_GO_MEMBERS                                    = 3255
```

The XNA-declared total is pinned and does not move. The BCL-inherited total is
the surface the composition projection makes representable: eleven public CLR
members of `Collection<IGameComponent>`, of which the indexer contributes two
accessors.

Language adapter types (`EventArgs`, `EventHandler<T>`, `EventSource<T>`,
`EventSubscription`, `GameCallbacks`, `Iterator<T>`, and `TimeSpan`) and error
results are measured separately and do not inflate these XNA counts. Every
adapter is declared in the framework package and takes the same package
qualification as any other framework-package name, so a graphics-package event
reads `framework.EventHandler[*framework.EventArgs]`.

The BCL base adapters are a separate, **unexported** family and are not
language adapters in this sense: `collectionBase[T]` is named by no projected
signature and cannot be reached, referenced, or type-asserted from outside the
module. When a composed base gains a consumer outside the framework package the
adapter moves to an internal package rather than becoming exported.
