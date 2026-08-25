# Foundation 29 — the `System.Exception` audit, and a measured frontier

This milestone completes no type. It does two things: it carries out the
directed audit of `System.Exception`, and it turns **DEFERRED** from a status
word into a measured claim across the whole base relationship table.

> A **DEFERRED** base must name what blocks it, down to the exact inherited
> member or the exact architecture decision. A deferred base that records
> nothing is a verifier failure.

Deferring with no recorded blocker is an unmeasured structural category wearing
a status word, which is precisely what the rest of this verifier exists to
prevent. Seven deferred bases now carry **21** named blockers, each classified
as `SUBSYSTEM` — one inherited member's type belongs to a .NET subsystem CNA-Go
has not mapped — or `ARCHITECTURE` — a cross-cutting public-API decision no
single member carries.

## Reference authority

```text
mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100)
              5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
```

Plus the pinned XNA assemblies, all read with `ikdasm`.

## The audit: what the eight derived types actually declare

Five types derive from `System.Exception`, three from
`System.Runtime.InteropServices.ExternalException`. **All eight declare only
constructors.**

| type | base | declared |
| --- | --- | --- |
| `Audio.NoMicrophoneConnectedException` | `Exception` | 3 ctors |
| `Content.ContentLoadException` | `Exception` | 3 ctors + a **protected serialization ctor** |
| `Graphics.DeviceLostException` | `Exception` | 3 ctors |
| `Graphics.DeviceNotResetException` | `Exception` | 3 ctors |
| `Graphics.NoSuitableGraphicsDeviceException` | `Exception` | 3 ctors |
| `Audio.InstancePlayLimitException` | `ExternalException` | 3 ctors |
| `Audio.NoAudioHardwareException` | `ExternalException` | 3 ctors |
| `Storage.StorageDeviceNotConnectedException` | `ExternalException` | 3 ctors + a **protected serialization ctor** |

This is the `GameComponentCollection` shape again, in a new domain: the declared
surface is constructors, and everything a caller would use is inherited. The
public-surface rule Foundation 26 established therefore applies — an inherited
public member is still public CLR surface and must not disappear — which is
exactly why a partial projection is not available as an escape.

Two further facts fall straight out of the table:

- Every one of the twenty-four public constructors takes `System.Exception` as
  its `innerException` parameter, so **`System.Exception` needs a
  signature-position projection before any derived type can be projected at
  all** — and its own surface carries the blockers below.
- `ContentLoadException` and `StorageDeviceNotConnectedException` declare a
  **protected** `.ctor(SerializationInfo, StreamingContext)`. Protected members
  are projected under settled policy, so those two need
  `System.Runtime.Serialization` for their *own declared* surface, before any
  inheritance question arises.

## The audit: what `System.Exception` would contribute

Eleven public instance members, which would project to thirteen Go identities.

| CLR member | type | mappable today? |
| --- | --- | --- |
| `Message` get | `string` | yes |
| `StackTrace` get | `string` | type yes, **semantics no** |
| `HelpLink` get/set | `string` | yes |
| `Source` get/set | `string` | yes |
| `ToString()` | `string` | yes |
| `GetType()` | `System.Type` | yes, `reflect.Type` is mapped |
| `InnerException` get | `System.Exception` | only if Exception itself maps |
| `GetBaseException()` | `System.Exception` | only if Exception itself maps |
| `Data` get | `System.Collections.IDictionary` | **no** |
| `TargetSite` get | `System.Reflection.MethodBase` | **no** |
| `GetObjectData(...)` | `SerializationInfo`, `StreamingContext` | **no** |

Plus protected, which settled policy projects: `HResult` get/set over `int32`
(fine), the serialization constructor, and
`add`/`remove_SerializeObjectState` over
`EventHandler<SafeSerializationEventArgs>` — two more serialization
dependencies.

Three distinct subsystems, none of them mapped:

1. **`System.Collections.IDictionary`** — the non-generic dictionary contract.
   Also the sole blocker of twelve `Design` type converters, so it is a shared
   frontier rather than an exception-only one.
2. **`System.Reflection.MethodBase`** — the reflection *member* model. CNA-Go
   maps `System.Type` to `reflect.Type` and nothing else.
3. **`System.Runtime.Serialization`** — `SerializationInfo` and
   `StreamingContext`. The same subsystem that blocks `Dictionary<K,V>`.

## The audit: the two architecture obstacles

These are the material ones, and neither is a missing type.

### Is an XNA exception type a Go `error`?

This is the decision, and it is **cross-cutting rather than local**.

*If yes*: `DeviceLostException` implements `Error() string`, `errors.As` works,
and `InnerException` chains to `errors.Unwrap`. But then every fallible
projected operation's error contract changes from "an opaque error" to
"possibly a typed CLR exception", and every settled per-operation fallibility
decision in the binding is reopened. Does `collectionIndexError` become an
`ArgumentOutOfRangeException` value? Does `curveArgumentError`? Does
`serviceArgumentError`? Roughly thirty deliberate, individually evidenced
decisions would have to be retaken together, and the four unexported sentinel
error families that exist today would all become public API.

*If no*: the eight types are inert objects. Nothing constructs them, nothing
returns them, nothing catches them. That is strictly worse than the
"collection nothing can be added to" that Foundation 25 rejected — a collection
at least works standalone, whereas an exception that is never thrown is pure
decoration.

Neither branch is available without a decision that is bigger than this
milestone, and the prompt's own instruction applies: do not invent a general
exception hierarchy merely to keep the session moving.

### What does `StackTrace` return?

The CLR captures a stack trace **at throw time**. A Go value built by
`NewDeviceLostException("...")` has never been thrown and has no throw site, so
the member would be projected with no faithful value to return. Returning the
empty string is an observable divergence from a reference that returns the
frames; returning a Go stack is a different thing wearing the same name.

## Verdict

`System.Exception` and `ExternalException` remain **DEFERRED**, and this is a
legitimate stop condition for those families: a material unresolved public API
decision remains, and it is now stated precisely enough that taking it is a
scoped piece of work rather than an open question.

The eight types are the second-largest blocked cluster in the profile.

## The other clusters, for completeness

`System.Attribute` was audited too, since it is the next-largest at five types
and the audit is cheap once the machinery exists. It is **not** easier:

- The five `ContentSerializer*` types declare real surface of their own —
  `ContentSerializerAttribute` has a constructor, `Clone`, and six `string`/
  `bool` properties — all trivially mappable.
- But Go has **no attribute metadata**. A projected attribute could be
  constructed and read back and never applied to anything, so the five types
  would be inert data objects whose entire purpose — annotating content-pipeline
  members — is unrepresentable.
- The inherited static lookups (`GetCustomAttribute`, `IsDefined`) take
  `System.Reflection` `Assembly`, `MemberInfo`, `Module` and `ParameterInfo`.
  They also raise a question that has never arisen before: **are inherited
  statics part of a derived type's projected surface at all?** `Collection<T>`
  had none.

The full measured table, all twelve non-XNA bases:

```text
IMPLIED   System.Object                        75 derived   15 projected
IMPLIED   System.ValueType                     56 derived   42 projected
IMPLIED   System.Enum                          49 derived   49 projected
MAPPED    System.EventArgs                      4 derived    3 projected
COMPOSED  Collection`1                          1 derived    1 projected   adds 12 identities
DEFERRED  Dictionary`2                          1 derived    6 blockers
DEFERRED  ReadOnlyCollection`1 (as a base)      4 derived    2 blockers
DEFERRED  System.Exception                      5 derived    5 blockers
DEFERRED  ExternalException                     3 derived    1 blocker
DEFERRED  System.Attribute                      5 derived    3 blockers
DEFERRED  ExpandableObjectConverter             1 derived    3 blockers
DEFERRED  System.IO.BinaryReader                1 derived    1 blocker
```

`COMPOSED` is the one status under which a base contributes Go member
identities, and the verifier now asserts that it is the only one — a
correctness fix, since Foundation 26 left `addsProjectedSurface` reading
`false` for `Collection<T>` while its adapter measurement reported twelve.

## Verifier

- `bclBaseRelationship` gains `Blockers`, each with a kind, the exact CLR
  member where one carries it, what it needs, and why.
- A `DEFERRED` base with no blockers is `BASE_MAPPING_MISMATCH`.
- `addsProjectedSurface` is now `true` exactly for `COMPOSED`.
- Three new self-tests: every deferred base names its blockers; a deferred base
  stripped of its blockers is rejected; and the `System.Exception` audit's
  specific findings are pinned, including that all eight derived types declare
  only constructors, so a later edit cannot quietly soften them.

```text
BCL_DEFERRED_BASE_BLOCKERS      21
ALLOWLIST_ENTRIES                0
UNMEASURED_STRUCTURAL_CATEGORY   0
```

## Scoreboard

Unchanged, deliberately — this milestone completes no type.

```text
TARGET_TYPES        121
TARGET_MEMBERS     1755
COMPLETE_TYPES      116
MISSING_TYPE        136
TOTAL_DIAGNOSTICS   313
MISSING_MEMBER      177
EXPECTED_GO_MEMBERS 3255
```
