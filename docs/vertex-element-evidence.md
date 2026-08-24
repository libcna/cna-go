# Foundation Milestone 5 VertexElement evidence

## Authority and exact closure

Foundation Milestone 5 completes exactly these Microsoft XNA Framework 4.0
Windows runtime identities:

```text
Microsoft.Xna.Framework.Graphics.VertexElement
Microsoft.Xna.Framework.Graphics.VertexElementFormat
Microsoft.Xna.Framework.Graphics.VertexElementUsage
```

The pinned public contract remains
`tools/api_compat/reference/xna40-windows-runtime-contract.json` at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The reference IL was read directly from the retained original assemblies:

| Assembly | SHA-256 |
|---|---|
| `Microsoft.Xna.Framework.dll` | `38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130` |
| `Microsoft.Xna.Framework.Graphics.dll` | `560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55` |

The closure contains 37 source identities: 10 on `VertexElement`, 13 on
`VertexElementFormat`, and 14 on `VertexElementUsage`. Enum `value__` storage
is omitted and the four writable properties expand into eight Go accessors,
so the compiler-derived projection has 39 Go identities: 14, 12, and 13
respectively. No mapping correction was needed.

No selected public signature references `IVertexType`, `VertexDeclaration`, a
built-in vertex struct, a vertex/index buffer, or `GraphicsDevice`. Enum names
such as `Byte4` and `HalfVector4` are literals and do not reference the
PackedVector types with the same names. The exact dependency closure therefore
remains three types.

## Enum projection and raw domains

Both enums are non-flags named `int32` types. Every constant uses an explicit
raw value; the implementation contains no `iota`.

| `VertexElementFormat` | Raw value |
|---|---:|
| `Single` | 0 |
| `Vector2` | 1 |
| `Vector3` | 2 |
| `Vector4` | 3 |
| `Color` | 4 |
| `Byte4` | 5 |
| `Short2` | 6 |
| `Short4` | 7 |
| `NormalizedShort2` | 8 |
| `NormalizedShort4` | 9 |
| `HalfVector2` | 10 |
| `HalfVector4` | 11 |

| `VertexElementUsage` | Raw value |
|---|---:|
| `Position` | 0 |
| `Color` | 1 |
| `TextureCoordinate` | 2 |
| `Normal` | 3 |
| `Binormal` | 4 |
| `Tangent` | 5 |
| `BlendIndices` | 6 |
| `BlendWeight` | 7 |
| `Depth` | 8 |
| `Fog` | 9 |
| `PointSize` | 10 |
| `Sample` | 11 |
| `TessellateFactor` | 12 |

CLR enums can hold undefined underlying values. The constructor and setters
therefore store every `int32` value without membership validation. Retained
fixtures use format `12345` and usage `-23456` through construction, getters,
setters, equality, hashing, and formatting. The enum types expose no public Go
`String`/`ToString` convenience surface. Private formatting helpers return the
exact name for declared values and the signed decimal underlying value for an
undefined non-flags value, matching the `System.Enum` path used by XNA.

## VertexElement value and property semantics

`VertexElement` is a Go value struct with four private fields. Its only source
constructor projects without an error result:

```go
func NewVertexElement(
    offset int32,
    elementFormat VertexElementFormat,
    elementUsage VertexElementUsage,
    usageIndex int32,
) VertexElement
```

The constructor IL performs four raw field stores. It does not reject negative
offsets, negative usage indexes, integer boundaries, or undefined enums.

The four source properties map to these eight identities:

```go
func (v VertexElement) Offset() int32
func (v *VertexElement) SetOffset(int32)
func (v VertexElement) VertexElementFormat() VertexElementFormat
func (v *VertexElement) SetVertexElementFormat(VertexElementFormat)
func (v VertexElement) VertexElementUsage() VertexElementUsage
func (v *VertexElement) SetVertexElementUsage(VertexElementUsage)
func (v VertexElement) UsageIndex() int32
func (v *VertexElement) SetUsageIndex(int32)
```

Getters use value receivers and setters use pointer receivers. Copying a value
copies all state; mutating each property on a copy leaves the original
unchanged. No state is shared and no backing field is exported.

Go's natural zero value is observable as offset `0`, format `Single`, usage
`Position`, and usage index `0`. It equals the four-argument constructor with
those values and has the same hash and string. This language-level zero value
does not add a parameterless XNA constructor or a `NewVertexElement()` member.

## Equality and operators

The public metadata declares only `Equals(Object)`. Because it is unique, the
deterministic mapping is:

```go
func (v VertexElement) Equals(value any) bool
```

There is deliberately no `EqualsByVertexElement`, typed `Equals`, alias, or
other convenience member. A Go `VertexElement` value is the mapped boxed-value
case; `nil`, pointers, and unrelated dynamic types compare false.

The XNA operator IL compares offset, usage index, usage, and format. The exact
mapped operator identities are:

```go
VertexElementOperatorEqualityByVertexElementAndVertexElement
VertexElementOperatorInequalityByVertexElementAndVertexElement
```

One-field-difference fixtures cover every property. Undefined enum values are
compared by their actual signed `int32` values. Inequality is the exact logical
negation of equality.

## XNA GetHashCode

`VertexElement.GetHashCode` boxes the 16-byte sequential value and calls the
XNA Framework helper `Helpers.SmartGetHashCode`. The helper pins the boxed
value, reads each complete 32-bit word, XORs them, and returns
`Int32.MaxValue` when the XOR is zero. For this layout the exact operation is:

```text
hash = Offset XOR int32(Format) XOR int32(Usage) XOR UsageIndex
if hash == 0: hash = 2147483647
```

The zero substitution is intentional and creates compatible collisions. It
must not be replaced with a better-distributed combine function.

| Fixture `(offset, format, usage, usageIndex)` | Exact `Int32` hash |
|---|---:|
| `(0, Single, Position, 0)` | 2147483647 |
| `(12, Vector3, TextureCoordinate, 7)` | 11 |
| `(-16, HalfVector4, Tangent, -3)` | 3 |
| `(123, 12345, -23456, -456)` | 27162 |
| `(MinInt32, HalfVector4, TessellateFactor, MaxInt32)` | -8 |
| `(1, Vector3, Normal, 0)` | 2147483647 |

The last row is a nonzero value whose four words XOR to zero; it proves that
the fallback is not limited to the all-zero struct.

## XNA ToString

The reference IL calls `String.Format` with `CultureInfo.CurrentCulture` and
the exact composite format:

```text
{{Offset:{0} Format:{1} Usage:{2} UsageIndex:{3}}}
```

The retained ordinary Go/XNA projection is therefore:

```text
{Offset:<offset> Format:<format> Usage:<usage> UsageIndex:<usageIndex>}
```

Known enums use their exact case-sensitive names. Undefined non-flags values
use their signed decimal raw value. CNA-Go does not introduce a `CultureInfo`
surface in this closure; the decimal representation matches the retained
invariant/default-culture reference observations and the established managed
value-string policy.

| Fixture | Exact string |
|---|---|
| zero | `{Offset:0 Format:Single Usage:Position UsageIndex:0}` |
| ordinary | `{Offset:12 Format:Vector3 Usage:TextureCoordinate UsageIndex:7}` |
| negative | `{Offset:-16 Format:HalfVector4 Usage:Tangent UsageIndex:-3}` |
| undefined enums | `{Offset:123 Format:12345 Usage:-23456 UsageIndex:-456}` |
| boundaries | `{Offset:-2147483648 Format:HalfVector4 Usage:TessellateFactor UsageIndex:2147483647}` |

## Behavioral and verifier evidence

The `PURE_XNA_DERIVED` corpus grows from 201 to 227 observations/assertions
with zero failures:

| Group | Observations |
|---|---:|
| `VERTEX_ELEMENT_ENUMS` | 2 |
| `VERTEX_ELEMENT` | 24 |

The groups cover every raw enum value, zero and constructor state, all getters
and setters, copy semantics, complete `int32` storage, undefined enums,
`Equals(Object)`, both operators, six exact hashes, and five exact strings.
Focused package tests exercise the same behavior with finer per-property
assertions and require no CNA library.

The verifier mutation inventory grows from 32 to 47 fixtures. The 15 new
negative cases cover wrong type kind; property-as-field; missing setters;
wrong property types; constructor order and enum type; accidental typed
`Equals(VertexElement)`; missing equality/inequality operators; wrong values
on both enums; and accidental flags classification. Unexpected receiver
members are attributed to their owning type in local measurements, so an
invented typed `Equals` makes both the global unexpected-member gate and the
VertexElement local matrix red. No allowlist or unmeasured category is used.

## Compiler-measured local strict-zero matrix

The generated API report retains this dedicated closure measurement:

| Type | Source members | Expected Go | Target Go | Local diagnostics | Kind |
|---|---:|---:|---:|---:|---|
| `VertexElement` | 10 | 14 | 14 | 0 | struct |
| `VertexElementFormat` | 13 | 12 | 12 | 0 | named `int32` enum, non-flags |
| `VertexElementUsage` | 14 | 13 | 13 | 0 | named `int32` enum, non-flags |

Every global unexpected-symbol, kind, base/interface, field/property,
method/signature/error, overload/generic, enum/flags, event/operator/ref-out/
language, native-leak, allowlist, and unmeasured counter remains zero. The
normal strict report remains red only for 200 genuinely missing types and the
unchanged 180 missing members on six native/runtime partial types.

This milestone adds no CNA function, cgo, native handle, interop route,
`VertexDeclaration`, buffer, draw path, GPU support, or hardware capability.
