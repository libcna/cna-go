# Foundation Milestone 9 ClearOptions evidence

## Authority and exact one-type closure

Foundation Milestone 9 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Graphics.ClearOptions
```

The public authority remains the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The independently inspected entry has CLR kind `enum`, base `System.Enum`,
`System.Int32` underlying storage, `[Flags]` metadata, no direct interfaces,
and exactly four CLR field identities:

| CLR identity | Role | Raw value |
|---|---|---:|
| `value__` | synthetic enum storage | — |
| `Target` | declared literal | 1 |
| `DepthBuffer` | declared literal | 2 |
| `Stencil` | declared literal | 4 |

The established enum-storage rule excludes `value__`. Four source identities
therefore map to exactly three Go identities. There is no declared constructor,
method, property, event, or operator.

## Flags projection and the unnamed zero value

`ClearOptions` belongs directly to `Microsoft.Xna.Framework.Graphics` and maps
to package `graphics` under `Microsoft/Xna/Framework/Graphics`. Its complete
public shape is a named `int32` with the exact `// xna:flags` directive and
three explicit constants:

```go
type ClearOptions int32

const (
    ClearOptionsTarget      ClearOptions = 1
    ClearOptionsDepthBuffer ClearOptions = 2
    ClearOptionsStencil     ClearOptions = 4
)
```

There is deliberately no `ClearOptionsNone`, `ClearOptionsDefault`,
`ClearOptionsNoClear`, or `ClearOptionsAll`. XNA metadata declares no named
zero literal and no named all-bits literal. Go nevertheless represents raw
zero naturally:

```go
var options ClearOptions
int32(options) == 0
```

That language zero value is unnamed; representability does not create a new
XNA member. The general mapping rule and its generic verifier fixture now state
this explicitly. Flags enums with three nonzero powers of two and no named
zero are valid, while an invented named zero remains `UNEXPECTED_MEMBER`.

## Raw domain and bitwise projection

Ordinary typed Go expressions preserve every declared combination:

| Expression | Raw value |
|---|---:|
| `Target | DepthBuffer` | 3 |
| `Target | Stencil` | 5 |
| `DepthBuffer | Stencil` | 6 |
| `Target | DepthBuffer | Stencil` | 7 |

Raw `0`, `8`, `1<<20`, and `-1` remain representable. Unknown bits are not
masked to `0x7` and negative values are not rejected. Representative OR/AND
results are `ClearOptions(8) | Target == 9`, `ClearOptions(7) & Stencil == 4`,
and `ClearOptions(2) & Target == 0`. These are Go language projections, not
claims about XNA renderer execution.

There is no validation, normalization, `iota`, `String`, `MarshalText`,
`Names`, `Format`, `ToString`, `Or`, `And`, `HasFlag`, `Contains`, or other
convenience surface.

## Structural and negative evidence

The dedicated generated closure measurement is:

| Source identities | Expected Go identities | Target Go identities | Local diagnostics | Kind | Underlying | Flags | Named zero |
|---:|---:|---:|---:|---|---|---|---|
| 4 | 3 | 3 | 0 | enum / named integer | `int32` | true | false |

It also records `Target=1`, `DepthBuffer=2`, `Stencil=4`,
`valueStorageExcluded=true`, and explicit false measurements for
`ClearOptionsNone`, `ClearOptionsDefault`, and `ClearOptionsAll` presence.
The whole-profile scoreboard is 61 target types, 1,333 target members, 373
genuine missing-surface diagnostics, 196 missing types, 177 missing members,
56 complete types, and the same five partial types. All 25 PackedVector
interface witnesses remain unchanged.

The verifier inventory grows from 85 to 99 cases. The 14 focused mutations
reject a missing type, wrong package, wrong kind, wrong underlying type,
missing marker, false flags classification, each wrong literal, missing
`Stencil`, projected `value__`, invented `ClearOptionsNone=0`, invented
`ClearOptionsAll=7`, and an exported helper method. The general no-named-zero
fixture proves that absence of a zero literal is valid without any
`ClearOptions` allowlist or mapping exception.

The local closure has zero diagnostics in every measured category: kind,
base/interface, field/property, method/signature, parameter/return/error,
overload/generic, enum/flags, event/operator/ref-out, and language mapping.
Globally, every unexpected-surface, mismatch, native-leak, allowlist, and
unmeasured counter is zero.

## Behavior provenance

The corpus grows from 247 to 256 observations/assertions with zero failures.
`CLEAR_OPTIONS` contributes nine observations:

- four `PURE_XNA_DERIVED` facts for exact literals, `System.Int32`, flags bit
  shape, and the absence of a declared zero literal;
- five `GO_LANGUAGE_PROJECTION` observations for unnamed raw zero,
  combinations 3/5/6/7, arbitrary signed raw values, OR, and AND.

Focused package tests and all ClearOptions corpus observations pass with
`CNA_NATIVE_LIBRARY` unset.

## ABI and strict scope boundary

This milestone adds no CNA constant, function, prototype position, C/Go
measurement, layout, callback, symbol, handle, cgo route, or
`internal/interop` change. ABI evidence remains 23 bound functions, 67
prototype type positions, 96 C/Go measurements, 28 layouts, two callbacks,
and five constants, with zero missing symbols or mismatches.

The two reverse-dependent overloads remain absent:

```text
GraphicsDevice.Clear(ClearOptions, Color, Single, Int32)
GraphicsDevice.Clear(ClearOptions, Vector4, Single, Int32)
```

No other GraphicsDevice member, render-target clearing, depth clearing,
stencil clearing, state mutation, renderer command, or native ClearOptions
route was added. Foundation 1's pre-existing `ClearByColor` route is unchanged
and does not implement either selected XNA overload. GraphicsDevice remains at
70 missing members, and combined `MISSING_MEMBER` remains 177.
