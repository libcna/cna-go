# Foundation Milestone 8 BufferUsage evidence

## Authority and exact one-type closure

Foundation Milestone 8 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Graphics.BufferUsage
```

The public authority remains the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, whose
retained SHA-256 is
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The independently inspected entry has CLR kind `enum`, base `System.Enum`,
`System.Int32` underlying storage, `[Flags]` metadata, no direct interfaces,
and exactly these three CLR field identities:

| CLR identity | Role | Raw value |
|---|---|---:|
| `value__` | synthetic enum storage | — |
| `None` | declared literal | 0 |
| `WriteOnly` | declared literal | 1 |

The established Go enum-storage rule excludes `value__`. The closure therefore
contains three source identities and exactly two mapped Go identities. There
is no declared constructor, property, method, event, or operator.

## Go flags projection

`BufferUsage` belongs directly to `Microsoft.Xna.Framework.Graphics` and maps
to package `graphics` under `Microsoft/Xna/Framework/Graphics`. Its exact
public shape is a named `int32` with the established exact `// xna:flags`
directive and explicit constants:

```go
type BufferUsage int32

const (
    BufferUsageNone      BufferUsage = 0
    BufferUsageWriteOnly BufferUsage = 1
)
```

The implementation uses neither `iota` nor an automatically generated bit
position. A zero Go value equals `BufferUsageNone`. Like the already qualified
`SpriteEffects` and `DisplayOrientation` projections, the complete signed
32-bit raw domain remains representable: values such as `2`, `3`, `1<<20`,
and `-1` are neither rejected nor normalized.

Ordinary typed Go bitwise operations qualify representative compositions:

```text
None | WriteOnly            = WriteOnly (1)
WriteOnly | WriteOnly       = WriteOnly (1)
BufferUsage(2) | WriteOnly  = BufferUsage(3)
```

No validation, mask restriction, `String`, `Or`, `And`, `HasFlag`,
`Contains`, `IsWriteOnly`, constructor, count constant, or other convenience
surface exists.

## Structural and negative evidence

The dedicated generated closure measurement is:

| Source identities | Expected Go identities | Target Go identities | Local diagnostics | Kind | Underlying | Flags |
|---:|---:|---:|---:|---|---|---|
| 3 | 2 | 2 | 0 | enum / named integer | `int32` | true |

It also records `None=0`, `WriteOnly=1`, `valueStorageExcluded=true`, and
`status=PASS`. The whole-profile scoreboard moves exactly as predicted to 60
target types, 1,330 target members, 374 missing-surface diagnostics, 197
missing types, 177 missing members, 55 complete types, and the same five
partial types. Every mismatch, unexpected-surface, leak, allowlist, and
unmeasured counter remains zero. All 25 PackedVector interface witnesses are
unchanged.

The verifier mutation inventory grows from 73 to 85 cases. The 12 focused
additions reject a missing type, wrong package, wrong kind, non-`int32`
underlying type, absent flags marker, false flags classification, wrong `None`
or `WriteOnly` values, missing `WriteOnly`, projected `value__`, an extra public
constant, and an exported helper method. Exact directive parsing also proves
that `// xna:flags=false` cannot masquerade as the required marker. The rules
are general enum/flags and unexpected-surface rules; there is no BufferUsage
allowlist or special verifier exception.

## Behavior provenance

The managed corpus grows from 242 to 247 observations/assertions with zero
failures. The five-observation `BUFFER_USAGE` group separates provenance:

- `PURE_XNA_DERIVED`: exact `None=0`, `WriteOnly=1`, and `Int32` storage
  projection;
- `GO_LANGUAGE_PROJECTION`: zero value, arbitrary raw signed values, and
  representative bitwise composition.

The `[Flags]` classification itself is verified from pinned metadata by the
structural verifier. Generic Go integer behavior is not mislabeled as an XNA
runtime observation. Focused package tests and the behavior tool pass with
`CNA_NATIVE_LIBRARY` unset and make no native call.

## ABI and scope boundary

This closure is pure managed enum metadata. It adds no CNA constant, function,
prototype position, measurement, layout, callback, symbol, native handle,
cgo route, or `internal/interop` change. The canonical ABI evidence therefore
remains exactly 23 bound functions, 67 prototype type positions, 96 C/Go
measurements, 28 layouts, two callbacks, and five constants, with zero missing
symbols or mismatches.

The dependency direction is intentionally one-way: future buffer APIs may use
`BufferUsage`, but this enum does not imply buffer support. The following
remain explicitly out of scope and unimplemented:

- `VertexBuffer` and `DynamicVertexBuffer`;
- `IndexBuffer` and `DynamicIndexBuffer`;
- `VertexDeclaration` and `IVertexType`;
- `SetData`, `GetData`, buffer creation, binding, or mapping;
- every GraphicsDevice buffer/draw member.

`Game`, `GraphicsDeviceManager`, `GraphicsDevice`, `SpriteBatch`, and
`Texture2D` are unchanged. In particular, Foundation 7's managed
`SupportedOrientations` behavior remains intact and GraphicsDeviceManager
still has exactly 40 missing members.
