# Foundation Milestone 10 SurfaceFormat evidence

## Authority and exact one-type closure

Foundation Milestone 10 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Graphics.SurfaceFormat
```

The public authority is the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
Its retained Graphics assembly provenance is SHA-256
`560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55`.
The independently inspected entry has CLR kind `enum`, base `System.Enum`,
`System.Int32` underlying storage, `flags=false`, no direct interfaces, and no
declared constructor, method, property, event, or operator.

The milestone request described 20 CLR identities and 19 mapped Go identities,
but its own required table contains 20 named literals (`0` through `19`). The
pinned contract therefore contains 21 field identities: synthetic `value__`
plus 20 named literals. The established enum-storage rule excludes only
`value__`, producing 20 mapped Go identities. Fresh verifier evidence is
authoritative, so the closure is retained as source=21 / expected-go=20 rather
than omitting a declared literal or forcing the requested arithmetic.

## Complete raw-value table

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Color` | `SurfaceFormatColor` | 0 |
| `Bgr565` | `SurfaceFormatBgr565` | 1 |
| `Bgra5551` | `SurfaceFormatBgra5551` | 2 |
| `Bgra4444` | `SurfaceFormatBgra4444` | 3 |
| `Dxt1` | `SurfaceFormatDxt1` | 4 |
| `Dxt3` | `SurfaceFormatDxt3` | 5 |
| `Dxt5` | `SurfaceFormatDxt5` | 6 |
| `NormalizedByte2` | `SurfaceFormatNormalizedByte2` | 7 |
| `NormalizedByte4` | `SurfaceFormatNormalizedByte4` | 8 |
| `Rgba1010102` | `SurfaceFormatRgba1010102` | 9 |
| `Rg32` | `SurfaceFormatRg32` | 10 |
| `Rgba64` | `SurfaceFormatRgba64` | 11 |
| `Alpha8` | `SurfaceFormatAlpha8` | 12 |
| `Single` | `SurfaceFormatSingle` | 13 |
| `Vector2` | `SurfaceFormatVector2` | 14 |
| `Vector4` | `SurfaceFormatVector4` | 15 |
| `HalfSingle` | `SurfaceFormatHalfSingle` | 16 |
| `HalfVector2` | `SurfaceFormatHalfVector2` | 17 |
| `HalfVector4` | `SurfaceFormatHalfVector4` | 18 |
| `HdrBlendable` | `SurfaceFormatHdrBlendable` | 19 |

Every production constant is explicitly assigned. There is no `iota` and no
spelling normalization. In particular, `Bgr565`, `Bgra5551`, `Bgra4444`,
`Dxt1`, `Dxt3`, `Dxt5`, `Rgba1010102`, `Rg32`, `Rgba64`, and `HdrBlendable`
retain their exact mapped identities.

## Go projection and raw domain

`SurfaceFormat` belongs directly to `Microsoft.Xna.Framework.Graphics` and is
implemented in package `graphics` as a named `int32` type. It has no
`// xna:flags` directive because the pinned metadata says `flags=false`.
Sequential values are ordinary enum literals, not composable flags.

The Go zero value is `SurfaceFormatColor` because `Color=0`. The complete
signed `int32` domain remains representable, including `SurfaceFormat(20)`,
`SurfaceFormat(12345)`, and `SurfaceFormat(-1)`. There is no membership
validation, clamping, masking, parsing, or normalization and no invented
`Default` or `None` constant.

There is no exported `String`, `MarshalText`, `UnmarshalText`, `Format`,
`Name`, size, component-count, compression, HDR, pixel-size, or native-format
helper. Names shared with `Graphics.PackedVector` are literals only. The
implementation has no PackedVector import, dependency, association, or
conversion helper.

## Structural and mutation evidence

The compiler-extracted local closure is:

| Source identities | Expected Go identities | Target Go identities | Local diagnostics | Kind | Underlying | Flags |
|---:|---:|---:|---:|---|---|---|
| 21 | 20 | 20 | 0 | enum / named integer | `int32` | false |

The generated report retains a PASS row for every one of the 20 named values
and confirms `valueStorageExcluded=true`. The whole-profile scoreboard is 62
target types, 1,353 target members, 372 genuine deferred diagnostics, 195
missing types, 177 missing members, 57 complete types, and the unchanged five
partial types. All 25 PackedVector interface witnesses remain unchanged.

The mutation inventory grows from 99 to 117 cases. The 18 SurfaceFormat cases
reject a missing or misplaced type, wrong kind, wrong underlying type,
accidental flags classification, wrong values at representative early/middle/
late positions, missing `Dxt3`, missing `HdrBlendable`, projected `value__`, an
extra constant, an exported `String` helper, and renamed `BGR565` spelling.
The generic enum verifier measures all 20 expected raw values. There is no
allowlist and no unmeasured structural category.

Every global unexpected-surface, kind, base/interface, field/property,
signature/parameter/return/error, overload/generic, enum/flags,
event/operator/ref-out/language, native-leak, allowlist, and unmeasured counter
is zero.

## Behavior provenance

The corpus grows from 256 to 262 observations/assertions with zero failures.
`SURFACE_FORMAT` contributes six observations:

- `PURE_XNA_DERIVED`: the complete named raw table, `System.Int32` storage,
  and `flags=false` contract fact;
- `GO_LANGUAGE_PROJECTION`: zero value equals `Color`, arbitrary positive raw
  values remain representable, and a negative raw value remains representable.

The focused package tests and behavior corpus run with `CNA_NATIVE_LIBRARY`
unset. They create no Game, GraphicsDevice, Texture2D, renderer, loader, or
interop state.

## Runtime and ABI boundary

Enum completion proves the managed XNA contract only. It does not claim that
CNA can upload, create, render, sample, compress, or target every listed
pixel/texture format. DXT compression, HDR, half-vector, floating-point target,
`Rgba1010102`, and any other backend format support remain unclaimed. There is
no GPU format negotiation, native mapping, CNA enum mirror, C manifest entry,
or ABI change.

The canonical ABI remains exactly 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants,
with zero missing header/library symbols and zero mismatches.

## Deferred reverse dependents

The regenerated graph records the same eleven reverse dependents, all still
deferred:

- `DisplayMode` and `DisplayModeCollection`;
- `GraphicsAdapter`;
- `PresentationParameters`;
- `RenderTarget2D` and `RenderTargetCube`;
- `Texture`, `Texture2D`, `Texture3D`, and `TextureCube`;
- `GraphicsDeviceManager` format properties.

No `Texture2D` format constructor, data-transfer path, render target, display
or adapter discovery, presentation state, GraphicsDevice format query, or
GraphicsDeviceManager format member was added. The five partial counts remain
Game=39, GraphicsDeviceManager=40, GraphicsDevice=70, SpriteBatch=16, and
Texture2D=12, totaling 177 missing members.
