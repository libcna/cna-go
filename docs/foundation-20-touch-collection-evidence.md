# Foundation 20 — the read-only touch collection

Foundation 20 closes `Microsoft.Xna.Framework.Input.Touch.TouchCollection` and
its nested public enumerator. It is the first cluster that is **simultaneously
a CLR value type and fallible**, which is only expressible because Foundation 17
made fallibility per operation and Foundation 19 generalized the pure-managed
classification.

## Reference authority

```text
Microsoft.Xna.Framework.Input.Touch.dll
sha256 b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25
```

Read with `ikdasm`. Public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## Why this was reachable

`TouchCollection` was previously grouped with the touch types blocked on device
capability. The IL says otherwise: it has **two public constructors**, no
`calli`, no P/Invoke, and no device access. Its storage is eight private inline
`TouchLocation` fields — `location0` through `location7`, not an array — plus a
count and a connection flag, so a collection allocates nothing and can never
hold a ninth touch.

Completing it claims **no touch capability**. CNA-Go still has no `TouchPanel`,
never produces a collection itself, and reads no device. A caller supplies the
locations, exactly as with `TouchLocation` in Foundation 15.

## A value type that can fail

`TouchCollection` is a `System.ValueType`, so it keeps copy semantics: assigning
it copies, and `GetEnumerator` hands the cursor a copy via `ldobj`. Yet nine of
its sixteen projected operations carry an error, because the reference either
validates arguments or throws unconditionally.

Admitting it to `pureManagedTypes` changes only that fallibility. It does not
make it a reference type, and the closure asserts the constructor projects
`TouchCollection`, not `*TouchCollection`.

## Exact contract

CLR: `public sealed struct TouchCollection : IList<TouchLocation>`,
15 source members, 16 projected Go identities, 9 with an error result.

```go
func NewTouchCollection(touches []TouchLocation) (TouchCollection, error)

func (c TouchCollection) IsConnected() bool
func (c TouchCollection) Count() int32
func (c TouchCollection) IsReadOnly() bool
func (c TouchCollection) Item(index int32) (TouchLocation, error)
func (c *TouchCollection) SetItem(index int32, value TouchLocation) error
func (c *TouchCollection) Add(item TouchLocation) error
func (c *TouchCollection) Clear() error
func (c *TouchCollection) Insert(index int32, item TouchLocation) error
func (c *TouchCollection) RemoveAt(index int32) error
func (c *TouchCollection) Remove(item TouchLocation) (bool, error)
func (c TouchCollection) Contains(item TouchLocation) bool
func (c TouchCollection) IndexOf(item TouchLocation) int32
func (c TouchCollection) CopyTo(array []TouchLocation, arrayIndex int32) error
func (c TouchCollection) FindById(id int32) (bool, TouchLocation)
func (c TouchCollection) GetEnumerator() TouchCollectionEnumerator
```

### Constructor

```text
IL_0000: ldarg.1; brtrue.s      -> null check
IL_0003: ldstr "touches"; newobj ArgumentNullException; throw
IL_000e: ldarg.1; ldlen; ldc.i4.8; ble.s  -> capacity check
IL_0014: ldstr "touches"; newobj ArgumentOutOfRangeException; throw
IL_001f: isConnected = true; locationCount = 0; eight slots initobj
         for each touch:
           TryGetPreviousLocation(out prev)
             true  -> AddTouchLocation(Id, State, X, Y, prev.State, prev.X, prev.Y)
             false -> AddTouchLocation(Id, State, X, Y, 0, 0f, 0f)
```

The guards run in that order, nil first. The capacity test is `> 8`, so exactly
eight is accepted. An empty but non-nil slice yields a **connected** empty
collection, which is the only way `IsConnected` differs from a zero value.

The private `AddTouchLocation` increments the count *first* and uses the
previous count to select a slot, so a ninth call would increment the count while
storing nothing. The constructor's capacity check makes that unreachable from
public API; the helper reproduces it rather than guarding it, so it matches the
reference exactly.

### The read side

| member        | reference behavior                                                        |
| ------------- | -------------------------------------------------------------------------- |
| `IsConnected` | stored flag; true after the public constructor, false on a zero value      |
| `Count`       | stored `locationCount`                                                     |
| `IsReadOnly`  | constant `ldc.i4.1; ret` — **always true**, including on a zero value      |
| `Item` get    | `index < 0 \|\| index >= Count` → `ArgumentOutOfRangeException("index")`, then a switch over the eight slots |
| `IndexOf`     | linear scan comparing with **`TouchLocation.op_Equality`**, or -1           |
| `Contains`    | `IndexOf(item) >= 0`                                                       |
| `FindById`    | linear scan on **Id alone**; a miss yields a zero `TouchLocation`          |
| `GetEnumerator` | `ldobj` a copy, then construct the cursor at position -1                 |

`IndexOf` uses the **equality operator**, which weighs all seven fields
including both state fields — not the looser `Equals(TouchLocation)`, which
ignores them. That distinction is the Foundation-15 `TouchLocation` quirk
reaching into a consumer, and the fixtures exercise a location that `Equals`
accepts and `IndexOf` misses.

`FindById`'s miss yields a plain zero `TouchLocation`, **not** the `Id`-of-`-1`
sentinel that `TouchLocation.TryGetPreviousLocation` returns. The two misses
look different by design.

### The write side is unconditional

`set_Item`, `Add`, `Clear`, `Insert`, `RemoveAt`, and `Remove` each have exactly
this body:

```text
IL_0000: newobj instance void [mscorlib]System.NotSupportedException::.ctor()
IL_0005: throw
```

They validate nothing first. `Insert(-99, item)` and `RemoveAt(-99)` report
not-supported, not out-of-range, and a rejected call changes nothing.

Those throws are projected as errors rather than dropped. A caller that mutates
a read-only view has a real bug, and silently accepting the call would hide it.

### `CopyTo`

Three checks, in the reference's order: a nil destination throws
`ArgumentNullException("array")`; `arrayIndex < 0` throws
`ArgumentOutOfRangeException("arrayIndex")`; and an insufficient destination
throws `ArgumentOutOfRangeException("arrayIndex")` too.

The capacity test is done in **64-bit arithmetic** — `conv.i8` on both sides —
so a start index near the top of the int32 range reports the argument error
instead of wrapping into a false pass. That is reproduced with `int64`, and a
fixture drives `arrayIndex = int32(1<<31 - 1)` to prove it.

The comparison is `array.Length < arrayIndex + Count`, so an exactly-sized
destination is accepted.

## `TouchCollection.Enumerator`

CLR: `public sealed struct Enumerator : IEnumerator<TouchLocation>`,
3 source members, 3 projected identities, 1 with an error result.

```go
func (e TouchCollectionEnumerator) Current() (TouchLocation, error)
func (e *TouchCollectionEnumerator) MoveNext() bool
func (e *TouchCollectionEnumerator) Dispose()
```

- The cursor starts at position **-1**.
- `Current` forwards to `TouchCollection.get_Item`, so it inherits that
  indexer's validation and reports an error **before the first `MoveNext`** and
  again once the cursor is exhausted.
- `MoveNext` increments, and on exhaustion **clamps** the position to `Count`
  rather than letting it run past, so repeated calls keep reporting false
  without drifting.
- `Dispose` is a bare `ret`. It owns nothing and is safe to call repeatedly.

The explicit `IEnumerator.Reset` and `IEnumerator.Current` implementations are
private in the reference and are not in the public contract, so neither is
projected.

## Mapping rules this settles

`System.Collections.Generic.IList<T>` projects the same way as `ICollection<T>`:
a concrete Go method set on the XNA collection, with no fabricated BCL package.
It needs nothing extra, because the indexer and index methods it adds are
already declared public members of the XNA collection.

A collection that declares its own public enumerator type projects that type
from `GetEnumerator`; the `Iterator<T>` adapter is for collections that declare
none. `CurveKeyCollection` uses the adapter, `TouchCollection` does not.

## Verifier coverage

The Foundation-17 managed-class closure is generalized into a shared
pure-managed **type** closure spanning both CLR kinds. It now records the source
kind and derives the expected constructor projection from the pinned metadata:
`*T` for a class, `T` for a struct. Projecting either one backwards silently
changes whether two variables share mutations, and `wrong_constructor_semantics`
attacks whichever direction is correct for the owner.

```text
AudioListener               class   value=false   5 ->  9   0 errors  ref *AudioListener           PASS
AudioEmitter                class   value=false   6 -> 11   1 error   ref *AudioEmitter            PASS
PresentationParameters      class   value=false  13 -> 23   0 errors  ref *PresentationParameters  PASS
TouchCollection             struct  value=true   15 -> 16   9 errors  ref TouchCollection          PASS
TouchCollection+Enumerator  struct  value=true    3 ->  3   1 error   ref (no constructor)         PASS
```

### Negative fixtures

The matrix grew from 13 defects to 14 — `dropped_error` was added as the mirror
of the artificial-error defects — and gained shape predicates so a defect that
cannot be expressed on a given type is **counted as skipped** rather than
silently dropped. Across five types: **61 applied, 9 skipped**, accounting for
all 70.

```text
AudioListener               1 skipped  nothing can fail, so dropped_error has no target
AudioEmitter                0 skipped  its one fallible setter expresses all 14
PresentationParameters      1 skipped  nothing can fail
TouchCollection             2 skipped  its only setter and only constructor are both fallible
TouchCollection+Enumerator  5 skipped  no constructor, no setter, no infallible getter
```

The isolated-surface helper now also supplies the module-wide `Iterator<T>`
adapter when the owner declares a BCL collection interface, so a correct
`TouchCollection` baseline is clean instead of failing on an adapter that
belongs to the module rather than to the type under test.

The mutation inventory grows from 314 to 336.

## Behavior corpus

Ten new observations in group `TOUCH_COLLECTION`, taking the corpus from 548 to
558 with zero failures: construction and its two guards including the
exactly-eight boundary, indexer validation, all six unconditional mutators with
deliberately invalid arguments, the operator-versus-`Equals` search
disagreement, `FindById`'s identifier-only match and zero-location miss, the
four `CopyTo` guards including the 64-bit overflow case, a full `CopyTo` with
its untouched prefix, the cursor at both ends, and value semantics including a
cursor that keeps its captured copy after its source variable is rebound.

## Structural effect

```text                       before   after
TARGET_TYPES                    110     112
TARGET_MEMBERS                 1680    1699
TOTAL_DIAGNOSTICS               324     322
MISSING_TYPE                    147     145
MISSING_MEMBER                  177     177
COMPLETE_TYPES                  105     107
PARTIAL_TYPES                     5       5
```

Every mismatch, leak, allowlist, and unmeasured counter stays at zero.

## No ABI change

Nothing here reaches native code. The CNA-Go ABI is unchanged at
`23 / 67 / 96 / 28 / 2 / 5` with no missing symbols and no mismatches. CNA was
not rebuilt.
