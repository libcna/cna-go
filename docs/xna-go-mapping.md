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
are explicitly classified by their execution boundary. A native/runtime
interface may retain a final `error`; a pure managed value interface does not
gain one merely because its owner is an interface. Both PackedVector
interfaces are pure managed value interfaces, so their mapped contracts are:

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
`Remove<EventName>Handler`. Addition returns an opaque, generation-checked
`EventSubscription`; removal accepts that token. Static events prefix the
declaring type. This preserves duplicate subscriptions, deterministic removal,
callback order, and self-removal without comparing Go function values. Named
CLR delegate types map to named Go function types. Native registration remains
private and uses `runtime/cgo.Handle`; an event token is never a native handle.

## Structural counts

The 257 reference types map one-to-one to 257 expected Go XNA types. Member
projection removes 49 enum backing fields, expands 840 properties into 1,119
accessors, and expands 49 events into 98 add/remove accessors:

```text
EXPECTED_GO_MEMBERS = 2964 - 49 - 840 + 1119 - 49 + 98 = 3243
```

Language adapter types (`EventSubscription`, `GameCallbacks`, `Iterator<T>`,
and `TimeSpan`) and error results are measured separately and do not
inflate these XNA counts.
