# Foundation Milestone 3 Curve evidence

## Authority and exact closure

Foundation Milestone 3 completes exactly the six-type Curve family from the
pinned Microsoft XNA Framework 4.0 Windows runtime contract:

```text
Curve
CurveKey
CurveKeyCollection
CurveContinuity
CurveLoopType
CurveTangent
```

The retained metadata independently yields 11, 15, 13, 3, 6, and 4 source
members respectively. The formal Go projection removes enum `value__` fields
and expands properties into accessors, yielding 13, 19, 14, 2, 5, and 3 mapped
members: 56 total. No other XNA type, CNA function, interop route, native
ownership category, or runtime partial was added.

`Curve`, `CurveKey`, and `CurveKeyCollection` are XNA classes and therefore Go
pointer facades with private managed state. Their identity is ordinary Go
pointer identity. They have no native handle, `Dispose`, runtime generation,
finalizer, or cgo dependency. The three enums are named `int32` types with
explicit non-flags values:

| Enum | Values |
|---|---|
| `CurveContinuity` | `Smooth=0`, `Step=1` |
| `CurveTangent` | `Flat=0`, `Linear=1`, `Smooth=2` |
| `CurveLoopType` | `Constant=0`, `Cycle=1`, `CycleOffset=2`, `Oscillate=3`, `Linear=4` |

## CurveKey

All three constructors retain the overload-derived spelling:

```text
NewCurveKeyBySingleAndSingle
NewCurveKeyBySingleAndSingleAndSingleAndSingle
NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity
```

The shorter constructors use `TangentIn=0`, `TangentOut=0`, and
`Continuity=Smooth`. `Position` is read-only. `Value`, `TangentIn`,
`TangentOut`, and `Continuity` have paired getters/setters and private backing
state. Enum setters preserve the source raw enum domain; they do not invent a
second set of legal Go values.

`Clone` returns a distinct key with independent scalar state. Equality compares
all five properties with XNA `Single` equality, so equal-but-distinct ordinary
keys compare equal, signed zeros compare equal, and a NaN field prevents value
equality. Both operators retain their one-to-one mapped identities and handle
nil operands before dispatching to value equality.

`CompareTo` compares `Position` only through the private XNA-era
`Single.CompareTo` projection: finite values and infinities have numeric order,
both signed zeros compare equal, NaN precedes every non-NaN value, and two NaNs
compare equal. A nil operand is a mapped managed argument error. `GetHashCode`
adds, with `int32` wraparound, the existing exact XNA-era `Single` hashes for
`Position`, `Value`, `TangentIn`, and `TangentOut`, followed by the continuity
raw value. Both zero signs hash as zero; other single bit patterns, including
NaN, retain the established scalar hash behavior.

## BCL collection projection

`CurveKeyCollection` directly implements `ICollection<CurveKey>` and
transitively implements generic and non-generic `IEnumerable`. CNA-Go does not
invent `System.Collections` packages. `ICollection<T>` maps to the concrete
collection's normal Go method set, while `IEnumerable<T>` and `IEnumerator<T>`
use the measured language adapter:

```go
type Iterator[T any] interface {
    Next() (value T, ok bool, err error)
}
```

Thus `GetEnumerator() Iterator[*CurveKey]` creates a fresh, source-ordered
cursor. It exposes no backing slice and takes no snapshot. Every structural
mutation increments a private version; the next cursor operation reports an
error after a mutation. `CopyTo` and an unsuccessful `Remove` do not invalidate
a cursor because neither changes collection contents. The verifier measures
the source interface relationship, the complete concrete method set, the
generic adapter shape, and its `(T, bool, error)` result order. A negative
mutation fixture proves the interface category is active.

The `Item[int]` indexer maps to `Item(int32) (*CurveKey, error)` and
`SetItem(int32, *CurveKey) error`. `Count` returns `int32`; `IsReadOnly` is
false. Invalid negative or end indexes, nil additions/replacements, nil
`CopyTo` storage, and insufficient destination length return errors rather
than leaking a Go panic.

Storage is a private `[]*CurveKey`, so collection boundaries retain key
references. `Add` uses the reference binary-search insertion algorithm. For
ordinary duplicate positions it advances past every existing equal position,
preserving insertion order. `SetItem` replaces in place when old and new
positions compare equal with `Single` equality; otherwise it removes the old
entry and reinserts the replacement in sorted position. `Contains`, `IndexOf`,
and `Remove` use full `CurveKey` value equality, not pointer identity. `CopyTo`
writes references into caller-owned storage. `RemoveAt` and `Clear` preserve
the remaining order.

`CurveKeyCollection.Clone` creates independent collection storage while
retaining the same key pointers. Mutating a shared key is visible through both
collections; adding or removing a collection entry is not. The copied
time-range cache follows the reference clone state independently.

## Curve state and clone

`NewCurve` creates one stable `Keys` collection and defaults both loop modes to
`Constant`. Repeated `Keys()` calls return that same collection facade.
`IsConstant` is exactly `Keys.Count <= 1`; it is true for zero or one key and
false for two keys even when their values are equal.

`Curve.Clone` returns a new Curve and a new `CurveKeyCollection`, copies both
loop modes, and retains the original key pointers through the collection's
shallow clone. Curve collection mutations are independent, while key-property
mutations remain shared.

## Tangents

Both `ComputeTangent` and both `ComputeTangents` overloads retain their complete
parameter-shape suffixes. An invalid individual key index returns an error.
Whole-curve computation iterates valid indexes and needs no error result.

- `Flat` writes zero on the selected side.
- `Linear` writes `current.Value-previous.Value` on input and
  `next.Value-current.Value` on output; it is not divided by position spacing.
- `Smooth` uses the previous/next endpoint substitution from XNA. For each side
  it multiplies the surrounding value span by that side's absolute position
  distance and divides by the full surrounding position span. A surrounding
  value span with absolute magnitude below `1.1920929e-7` produces zero.

Input and output modes are evaluated independently. Formulas depend only on
positions and values, so repeated whole-curve tangent computation is stable and
earlier tangent writes do not affect later keys. All stored and arithmetic
values remain `float32`; duplicate-position division retains XNA non-finite
behavior.

## Evaluate and loop modes

An empty curve evaluates to positive zero and a single-key curve always returns
that key's value. Multi-key segment search proceeds in collection order and
selects the first end key whose position is greater than or equal to the
requested position. The interpolation fraction uses the reference double
position intermediate and narrows once to `float32`; spans at or below
`1e-10` use fraction zero. This preserves exact keys and deterministic duplicate
positions.

`Step` returns the segment start value while the interpolation amount is below
one and switches at the exact next key. Smooth continuity uses the XNA cubic
Hermite bases and binary32 accumulation order with `start.TangentOut` and
`end.TangentIn`. Focused and corpus fixtures cover zero/asymmetric/negative/NaN
tangents, non-unit spacing, exact segment boundaries, and exact result bits.

Outside the key range:

- `Constant` returns the applicable endpoint value.
- `Linear` extrapolates with the first input or last output tangent and the
  signed distance from the endpoint.
- `Cycle` maps the position back into the key range.
- `CycleOffset` additionally adds `(last.Value-first.Value)*cycle`.
- `Oscillate` uses even cycles in forward order and odd cycles reflected from
  the last position.

Cycle calculation is deliberately not Go remainder arithmetic. It computes
`(position-first.Position)*inverseTimeRange` in binary32, subtracts one whenever
that value is negative, then truncates to the XNA integer cycle. Consequently a
negative exact multiple belongs to the preceding cycle; negative offset signs
and negative oscillation parity are explicitly qualified.

## Behavior and strict evidence

The `PURE_XNA_DERIVED` corpus grows from 93 to 142 observations/assertions with
zero failures:

| Group | Observations |
|---|---:|
| `CURVE_ENUMS` | 3 |
| `CURVE_KEY` | 10 |
| `CURVE_COLLECTION` | 12 |
| `CURVE_TANGENTS` | 7 |
| `CURVE_EVALUATE` | 8 |
| `CURVE_LOOPS` | 9 |

The compiler-measured local matrix is:

| Type | Source members | Expected Go members | Target Go members | Local diagnostics | Kind | Behavior |
|---|---:|---:|---:|---:|---|---|
| `Curve` | 11 | 13 | 13 | 0 | class pointer facade | PASS |
| `CurveKey` | 15 | 19 | 19 | 0 | class pointer facade | PASS |
| `CurveKeyCollection` | 13 | 14 | 14 | 0 | class pointer facade | PASS |
| `CurveContinuity` | 3 | 2 | 2 | 0 | named `int32` enum | PASS |
| `CurveLoopType` | 6 | 5 | 5 | 0 | named `int32` enum | PASS |
| `CurveTangent` | 4 | 3 | 3 | 0 | named `int32` enum | PASS |

The global target is 35 types / 1,089 members: 29 complete types, the same six
native/runtime partials, and 222 missing types. The remaining 402 diagnostics
are exactly 222 missing types plus the unchanged 180 missing members. Every
mismatch, unexpected-symbol, leak, allowlist, and unmeasured category is zero.
