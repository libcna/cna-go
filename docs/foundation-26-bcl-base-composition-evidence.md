# Foundation 26 — the general BCL base-class composition projection

Foundation 26 takes the decision Foundation 25 named as the highest-value open
one: **how a BCL collection base projects**. The answer is a general
architecture, not a one-type exception, and `GameComponentCollection` is its
first proof.

> A CLR class that inherits a supported BCL collection base projects as a
> concrete Go reference type that **contains a private generic adapter** for
> that base and re-exposes the base's **public** surface through measured
> forwarding members.

The two rules that fall out of it:

- **A public member inherited from a supported BCL base is still public CLR
  surface**, so it must not disappear merely because the XNA assembly metadata
  does not declare it.
- **The adapter is implementation machinery.** It is not an XNA type, not an
  exported field, not a public base-class object, not an embedded public API,
  and not a native handle.

Foundation 25 recorded four alternatives and chose none. This is alternative 1
and 3 combined, with the part that made 3 unattractive — "adds a public non-XNA
support type" — removed by keeping the adapter unexported.

## Reference authority

Two pinned binaries, because the behavior comes from two places.

```text
Microsoft.Xna.Framework.Game.dll  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
mscorlib.dll                      5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
```

The XNA assembly is the pinned Foundation-1 reference, re-verified by hash this
session. The mscorlib is the **.NET Framework 4.0 RTM** implementation
assembly:

```text
Assembly      mscorlib, Version 4.0.0.0
FileVersion   4.0.30319.1 (RTMRel.030319-0100)
ModuleRefs    QCall, kernel32.dll, advapi32.dll, clr.dll, ...
```

It is the right binary and not merely a plausible one. Every pinned XNA
assembly declares

```text
AssemblyRef  Name=mscorlib  Version=4.0.0.0  PublicKey=B7 7A 5C 56 19 34 E0 89
```

which is that identity. It is an implementation assembly rather than a
reference assembly, so `Collection<T>`, `List<T>` and `Dictionary<K,V>` carry
real IL. Both were read with `ikdasm`; the resource strings below were read
from the `Microsoft.Xna.Framework.Resources.resources` stream with `monodis
--mresources`.

Modern .NET was **not** consulted. Mono's mscorlib was **not** consulted. Where
this document states a behavior, the IL for it is quoted.

## What `Collection<T>` actually is

```text
.class public auto ansi serializable beforefieldinit
       System.Collections.ObjectModel.Collection`1<T>
       extends System.Object
       implements IList`1<!T>, ICollection`1<!T>, IEnumerable`1<!T>,
                  IList, ICollection, IEnumerable
{
  .field private class System.Collections.Generic.IList`1<!T> items
  .field private notserialized object _syncRoot
```

The parameterless constructor — the only one any pinned XNA subclass calls —
is

```text
IL_0007:  newobj  instance void class List`1<!T>::.ctor()
IL_000c:  stfld   IList`1<!0> Collection`1<!T>::items
```

so the backing store is always `List<T>`, and `items.IsReadOnly` is always
false. That matters: six public mutators open with

```text
callvirt instance bool ICollection`1<!T>::get_IsReadOnly()
brfalse.s <continue>
ldc.i4.s 28
call void System.ThrowHelper::ThrowNotSupportedException(...)
```

and for every consumer in the pinned profile that guard is **statically dead**.
It is not projected as a failure mode, and that is a measured claim rather than
an omission.

### The public surface is eleven members, not thirty-two

`Collection<T>` declares thirty-two members. Only eleven are public surface:

| CLR member | kind | projected Go member |
| --- | --- | --- |
| `Count` | property, get | `Count() int32` |
| `Item` | property, get | `Item(int32) (IGameComponent, error)` |
| `Item` | property, set | `SetItemProperty(int32, IGameComponent) error` |
| `Add` | method | `Add(IGameComponent) error` |
| `Clear` | method | `Clear() error` |
| `Contains` | method | `Contains(IGameComponent) bool` |
| `CopyTo` | method | `CopyTo([]IGameComponent, int32) error` |
| `GetEnumerator` | method | `GetEnumerator() Iterator[IGameComponent]` |
| `IndexOf` | method | `IndexOf(IGameComponent) int32` |
| `Insert` | method | `Insert(int32, IGameComponent) error` |
| `Remove` | method | `Remove(IGameComponent) (bool, error)` |
| `RemoveAt` | method | `RemoveAt(int32) error` |

Eleven CLR members, twelve Go identities: the indexer's two accessors are two
projections, exactly as the settled property rule requires.

The other twenty-one are excluded, each for a recorded reason:

- **two constructors** — the CLR does not inherit constructors;
- **five `family` members** — `Items` and the four virtual hooks. `Items`
  returns the backing store, so projecting it would hand out storage;
- **fourteen private explicit implementations** — every `IList`, `ICollection`,
  `IEnumerable` member plus `ICollection<T>.IsReadOnly`.

That last group is the load-bearing one. `IsReadOnly` is

```text
.method private hidebysig newslot specialname virtual final
        instance bool 'System.Collections.Generic.ICollection<T>.get_IsReadOnly'()
```

`private`, name-mangled — an explicit implementation. `new Collection<int>()
.IsReadOnly` does not compile in C# either. The settled Foundation-24 rule
already says an explicitly implemented interface member is not public surface,
so **no `IsReadOnly`, `IsSynchronized`, `SyncRoot`, or `IsFixedSize` is
projected**, and the verifier checks each is absent rather than merely not
required. This corrected a real defect: the existing collection-interface check
demanded `IsReadOnly` unconditionally, which would have forced CNA-Go to invent
a member the CLR does not expose.

### Validation is per operation, and deliberately inconsistent

Every guard below is quoted from the pinned IL. They do not agree with each
other, and the projection does not tidy them up.

| operation | guard | then |
| --- | --- | --- |
| `get_Item` | none of its own | `items[index]`, so **List<T>**'s `(uint)index >= (uint)_size` |
| `set_Item` | `index < 0 \|\| index >= Count` | `callvirt SetItem` |
| `Insert` | `index < 0 \|\| index > Count` | `callvirt InsertItem` |
| `RemoveAt` | `index < 0 \|\| index >= Count` | `callvirt RemoveItem` |
| `Add` | none | `callvirt InsertItem(Count, item)` |
| `Remove` | none | `IndexOf`; `-1` returns false; else `callvirt RemoveItem` |
| `Clear` | none | `callvirt ClearItems` |
| `CopyTo` | none of its own | `items.CopyTo`, so `Array.Copy`'s three failures |

`Insert` uses `ble.un.s` where `set_Item` and `RemoveAt` use `blt.un.s`: its
guard **admits `Count` itself**, which is why `Add` can be defined as
`InsertItem(Count, item)`. The comparisons are unsigned, so a negative index
fails the same test as an index past the end.

Note also that `set_Item` validates its index *before* it reaches `SetItem`.
On a subclass whose `SetItem` always throws, an out-of-range assignment
therefore reports the **range** failure, not the unsupported one.

### Enumeration is `List<T>`'s, including its version

```text
Collection`1::GetEnumerator
  ldfld    items
  callvirt IEnumerator`1<!0> IEnumerable`1<!T>::GetEnumerator()
```

So the enumerator a caller observes is `List<T>.Enumerator`. Its `MoveNext`
compares the version **first**:

```text
ldfld int32 Enumerator<!T>::version
ldfld int32 List`1<!T>::_version
bne.un.s  IL_004a          // -> MoveNextRare, which throws
ldfld int32 Enumerator<!T>::index
ldfld int32 List`1<!T>::_size
bge.un.s  IL_004a
```

Two consequences the projection preserves and the tests pin:

- a mutation is reported even when the cursor is already past the end, because
  the version check precedes the bounds test;
- `List<T>.Clear` runs `_version++` **unconditionally** — its IL skips only the
  `Array.Clear` and the `_size = 0` store when the list is already empty — so
  **clearing an already empty collection still invalidates live enumerators**.

`CNA-Go` reuses the settled `Iterator[T]` adapter rather than inventing a
second enumeration shape. Fusing `MoveNext` and `Current` into one `Next` makes
two CLR states unrepresentable rather than reinvented: `IEnumerator<T>.Current`
returns `default(T)` before the first step and after the last, because
`Enumerator.get_Current` is one `ldfld` with no validation. `Reset` and the
non-generic `Current` are private explicit implementations and project nothing.

### Element equality

`Contains` and `IndexOf` end in `EqualityComparer<T>.Default`. For an element
type whose CLR form is a reference type that does not override `Object.Equals`,
that comparer reduces to reference identity — and **no type in
`Microsoft.Xna.Framework.Game.dll` overrides `Equals`**, so it reduces to
reference identity for every component XNA ships.

Equality is therefore supplied per consumer rather than assumed, and for
`IGameComponent` it is Go's `==` on the interface value, which is pointer
identity for the pointer facades CNA-Go projects CLR classes as.

One guard is a Go language necessity rather than CLR behavior. Go permits an
interface to be satisfied by a *value* type, and `==` panics when two such
values share a dynamic type that is not comparable. No CLR implementor can be
in that state, so there is no reference identity to project for one; CNA-Go
reports "not the same element" rather than panicking from inside a collection
operation.

`List<T>.Contains` also splits on a null sought item, scanning for a null
element instead of consulting the comparer. A stored nil element is found.

## The architecture

```go
type GameComponentCollection struct {
    base             collectionBase[IGameComponent]   // unexported, not embedded
    componentAdded   EventSource[*GameComponentCollectionEventArgs]
    componentRemoved EventSource[*GameComponentCollectionEventArgs]
}
```

### No exported embedding

```go
type GameComponentCollection struct { Collection[IGameComponent] }   // REFUSED
```

is rejected by the verifier, for four separate reasons: it publishes a Go type
the XNA contract never declares; it promotes a whole method set the derived
type never declared; it implies a Go subtype relationship CLR inheritance does
not create; and it exposes support implementation to type assertions. Embedding
the *unexported* adapter is refused too — promotion would publish forwarding
nobody measured.

### The hooks

`Collection<T>`'s four protected virtuals are projected as an **unexported** Go
interface:

```go
type collectionOverrides[T any] interface {
    insertItem(index int32, item T) error
    removeItem(index int32) error
    setItem(index int32, item T) error
    clearItems() error
}
```

Unexported methods mean only a type declared in this module can satisfy it or
reach it — the same reach a CLR consumer has to a sealed class's protected
members, which is none. Every mutating public operation routes through it, so a
subclass override always runs; nothing appends to the store and then separately
fakes the override's observable effect. `TestEveryMutatorRoutesThroughItsHook`
proves the routing generically, independent of any XNA type.

No reflection is used for dispatch, and no runtime type assertion: the wiring
is a single field assignment in the constructor.

### The storage

The backing slice is never returned, never aliased into a caller's hands, and
never an exported field. `CopyTo` hands out a copy, and the tests mutate that
copy to prove it does not reach the collection.

## `GameComponentCollection`'s four overrides

The reference is deliberately not symmetric. All four are quoted below and none
is tidied.

```text
InsertItem   IndexOf(item) != -1 -> throw ArgumentException(CannotAdd...)
             base.InsertItem(index, item)                   <- mutates
             item != null -> OnComponentAdded(new ...EventArgs(item))

RemoveItem   IGameComponent removed = base[index]
             base.RemoveItem(index)                         <- mutates
             removed != null -> OnComponentRemoved(new ...EventArgs(removed))

SetItem      newobj NotSupportedException(CannotSet...); throw

ClearItems   for (int i = 0; i < base.Count; i++)
                 OnComponentRemoved(new ...EventArgs(base[i]));
             base.ClearItems()                              <- mutates LAST
```

**Insert and Remove mutate before they announce. Clear announces the whole
collection before it changes anything.** A handler that fails therefore leaves
an `Add` or a `Remove` already applied, and leaves a `Clear` not applied at
all. Both directions are in the behavior corpus.

Three further details the previous session's summary did not record, all read
directly from the IL:

- **`ClearItems` has no null check.** `InsertItem` and `RemoveItem` both guard
  their raise with `brfalse.s`; `ClearItems` does not. It announces a nil
  element too, with a `GameComponentCollectionEventArgs` whose `GameComponent`
  is nil. That asymmetry is preserved and tested.
- **`ClearItems` re-reads `base.Count` every iteration** (`IL_001a`–`IL_0021`
  is the loop condition), so a handler that adds a component extends the loop.
- **`InsertItem` calls `IndexOf` non-virtually** (`call`, not `callvirt`), so
  the duplicate test is `Collection<T>`'s and cannot be intercepted.

A nil component is inserted like any other and announces nothing, because
`IndexOf(null)` finds a nil element only when one is already present — which is
also why a *second* nil insertion fails as a duplicate.

The two exception messages are the exact reference strings:

```text
CannotAddSameComponentMultipleTimes
  "Cannot add the same game component to a game component collection multiple times."
CannotSetItemsIntoGameComponentCollection
  "Cannot set a value using operator[] on GameComponentCollection.  Use Add/Remove instead."
```

### The `SetItem` collision

`GameComponentCollection` declares a protected `SetItem` override, and inherits
a public `Item` setter. The settled member rules spell both `SetItem`. The
established collision rule resolves it mechanically — *append each collider's
source kind* — with no bespoke exception:

```text
declared protected override   SetItem   -> SetItemMethod
inherited indexer setter      Item set  -> SetItemProperty
```

The inherited getter keeps the plain name `Item`, because nothing collides with
it. Both `SetItem*` members always fail; they differ only in that the setter
validates the index first.

## Identity accounting

Three provenance classes are kept distinct, and **`REFERENCE_MEMBERS` is not
falsified**: it still names exactly what the Microsoft assemblies declare.

```text
REFERENCE_XNA_MEMBERS             2964      unchanged
BCL_INHERITED_PUBLIC_MEMBERS        11      new counter
BCL_INHERITED_MEMBER_PROJECTIONS    12      new counter
EXPECTED_GO_MEMBERS               3255      3243 + 12
```

The 3243 XNA-declared projection count **did not move**. The count admission in
`main.go` now checks the two classes separately and re-derives the inherited
count straight from the registry, so a change in either is attributed rather
than absorbed. `TestPinnedContractAndMappedCounts` additionally partitions the
whole expected surface by provenance and requires the two halves to be disjoint
and exhaustive: **every expected Go member has exactly one provenance class, so
nothing is counted twice.**

For `GameComponentCollection`:

```text
XNA-declared members        7   ->   9 Go identities
BCL-inherited CLR members  11   ->  12 Go identities
                                    21 total
```

Every inherited projection is attributable to an exact BCL base, an exact CLR
member, and an exact projected Go member; the report carries the whole table
under `bclBaseAdapters[].inheritedMemberInventory`.

## Fallibility

Per operation, as the settled rule requires, and mixing both provenance classes
in one table because fallibility belongs to the operation rather than to where
it was declared.

| infallible | why |
| --- | --- |
| `Count`, `Contains`, `IndexOf`, `GetEnumerator` | read only |
| `NewGameComponentCollection` | one `call base..ctor` |

| fallible | why |
| --- | --- |
| `Item` get | List<T>'s bounds check |
| `SetItemProperty`, `SetItemMethod` | range, then an unconditional throw |
| `Add`, `Insert` | duplicate rejection, plus a raise that can fail |
| `Remove`, `RemoveAt`, `Clear` | a raise that can fail, plus a range check on the two indexed forms |
| `CopyTo` | `Array.Copy`'s three argument failures |
| `InsertItem`, `RemoveItem`, `ClearItems` | the same, reached directly |

The error values are **unexported** sentinels wrapped with `%w`, matching the
`GameServiceContainer` precedent. That is deliberate and is the one recorded
sub-boundary of this milestone: see below.

## Recorded sub-boundary — distinguishing CLR exception types

`Collection<T>` and `List<T>` throw four distinct CLR exception types:
`ArgumentOutOfRangeException`, `ArgumentNullException`, `ArgumentException` and
`NotSupportedException`. CNA-Go reports failure through `error` results and has
no exception hierarchy, and `System.Exception` remains a **DEFERRED** base.

CNA-Go therefore projects **which operations fail and in which order**, which
is what the collection contract needs to be usable, but **does not give a
consumer a public way to tell one CLR exception type from another**. Making
`errors.Is` part of the public contract would be the `System.Exception` public
mapping decision, which is not this milestone's to take.

No collection behavior was blocked by this. Nothing here forced the exception
frontier open.

## Verifier

New measurement:

- a **BCL base adapter registry**, keyed by CLR base identity, recording the
  known base identity, the Go internal adapter family, the inherited public
  member inventory with a rationale per member, the supported behavior level,
  the pinned BCL authority and its hash, the deliberate exclusions with
  reasons, and the concrete XNA consumers;
- a new base status **`COMPOSED`**, which is the one status under which a base
  contributes projected Go identities;
- per-consumer measurement of the private adapter field, the two provenance
  classes, and exported embeddings.

New negative controls — 22 mutations, all in `testdata/mutations.json`,
bringing the inventory from 447 to 469:

```text
base_silently_dropped              exported_embedding
unexported_embedding               raw_slice_projection
exported_raw_slice_field           exported_raw_map_field
missing_adapter_field              exported_adapter_field
wrong_adapter_generic              wrong_adapter_family
excluded_member_projected          inherited_member_missing
inherited_indexer_setter_missing   declared_override_missing
extra_inherited_member             wrong_count_projection
wrong_indexer_projection           wrong_enumerator_type
enumerator_adapter_absent          infallible_mutator
infallible_indexer_setter          fallible_reader
```

`TestBCLCompositionFixtureIsCleanBeforeMutation` is the control they depend on:
the unmutated fixture produces zero diagnostics, so each mutation's diagnostic
is caused by the mutation.

The behavior-level controls are Go tests rather than verifier mutations,
because they are claims about runtime behavior: hook routing, version movement,
storage aliasing, and the mutation/announcement order in both directions.

```text
ALLOWLIST_ENTRIES               0
UNMEASURED_STRUCTURAL_CATEGORY  0
```

## Scoreboard

```text                          before   after
TARGET_TYPES                       118     119
TARGET_MEMBERS                    1722    1743
COMPLETE_TYPES                     113     114
MISSING_TYPE                       139     138
TOTAL_DIAGNOSTICS                  316     315
MISSING_MEMBER                     177     177     protected partials untouched
EXPECTED_GO_MEMBERS               3243    3255
mutation inventory                 447     469
behavior corpus                    575     588
```

Every mismatch, leak, allowlist and unmeasured counter is zero, including the
now load-bearing `BASE_MAPPING_MISMATCH`.

## What this does not do

Completing `GameComponentCollection` wires nothing up. CNA-Go has no
`GameComponent`, no `DrawableGameComponent`, and no component loop, and `Game`
remains a partial native-backed facade with no `Components` property, so
nothing in the binding constructs or drives a collection. The type is usable by
an external consumer; it is not used by CNA-Go.

`ReadOnlyCollection<T>` and `Dictionary<K,V>` remain **DEFERRED**. The registry
is built to hold them, and the architecture is general, but a family becomes
`COMPOSED` only once its exact pinned behavior has been read — not because the
shape looks similar.
