# Foundation Milestone 11 DepthFormat evidence

## Authority and exact one-type closure

Foundation Milestone 11 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Graphics.DepthFormat
```

The public authority is the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The independently inspected entry has CLR kind `enum`, base `System.Enum`,
`System.Int32` underlying storage, `flags=false`, no direct interfaces, and no
declared constructor, method, property, event, or operator.

The exact closure contains five source field identities: synthetic `value__`
plus four named literals. The established enum-storage mapping excludes only
`value__`, producing exactly four expected and four target Go identities.

## Complete raw-value table

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `None` | `DepthFormatNone` | 0 |
| `Depth16` | `DepthFormatDepth16` | 1 |
| `Depth24` | `DepthFormatDepth24` | 2 |
| `Depth24Stencil8` | `DepthFormatDepth24Stencil8` | 3 |

Every constant is explicitly assigned. Production contains no `iota`, alias,
alternate spelling, or `Stencil8` literal. `Depth24Stencil8` is one ordinary
enum alternative with raw value 3; it is not a bitwise composition.

## Go projection and raw domain

`DepthFormat` belongs directly to namespace
`Microsoft.Xna.Framework.Graphics` and package `graphics`. It is a named
`int32` type with no `// xna:flags` directive. The Go zero value equals
`DepthFormatNone`.

The complete signed `int32` domain remains representable. Focused tests retain
`DepthFormat(4)`, `DepthFormat(12345)`, and `DepthFormat(-1)` exactly. There is
no membership validation, clamping, normalization, constructor, panic, or
error-return conversion.

There is no exported `String`, `MarshalText`, `UnmarshalText`, parser, `Name`,
`Format`, `HasStencil`, `DepthBits`, `StencilBits`, `BytesPerPixel`,
`IsDepthOnly`, `IsDepthStencil`, `NativeFormat`, `ToSurfaceFormat`, or other
helper API.

## Structural and mutation evidence

The compiler-extracted local closure is:

| Source identities | Expected Go identities | Target Go identities | Local diagnostics | Kind | Go kind | Underlying | Flags | `value__` excluded |
|---:|---:|---:|---:|---|---|---|---|---|
| 5 | 4 | 4 | 0 | enum | named | `System.Int32` / `int32` | false | true |

The local strict-zero matrix is:

```text
MISSING_MEMBER=0
TYPE_KIND_MISMATCH=0
BASE_MAPPING_MISMATCH=0
INTERFACE_MAPPING_MISMATCH=0
FIELD_MAPPING_MISMATCH=0
PROPERTY_MAPPING_MISMATCH=0
METHOD_SIGNATURE_MAPPING_MISMATCH=0
PARAMETER_MAPPING_MISMATCH=0
RETURN_MAPPING_MISMATCH=0
ERROR_MAPPING_MISMATCH=0
OVERLOAD_MAPPING_MISMATCH=0
GENERIC_MAPPING_MISMATCH=0
ENUM_VALUE_MISMATCH=0
FLAGS_MAPPING_MISMATCH=0
EVENT_MAPPING_MISMATCH=0
OPERATOR_MAPPING_MISMATCH=0
REF_OUT_MAPPING_MISMATCH=0
LANGUAGE_MAPPING_MISMATCH=0
```

The whole-profile scoreboard is 63 target types, 1,357 target members, 371
genuine deferred diagnostics, 194 missing types, 177 missing members, 58
complete types, and the unchanged five partial types. Every global structural
preservation, leak, allowlist, and unmeasured counter remains zero; all 25
PackedVector interface witnesses remain unchanged.

The verifier inventory grows from 117 to 132 cases. The 15 Foundation-11
mutations reject a missing or misplaced type, wrong kind, wrong underlying
type, accidental flags classification, each of the four wrong literal values,
missing `Depth24`, missing `Depth24Stencil8`, projected `value__`, an extra
constant, an exported `String` helper, and renamed `Depth24Stencil08` spelling.
All four constants are also measured by the generic enum raw-value verifier.
There is no allowlist or unmeasured structural category.

## Behavior provenance

The corpus grows from 262 to 268 observations/assertions with zero failures.
`DEPTH_FORMAT` contributes six observations:

- `PURE_XNA_DERIVED`: the complete four-value raw table, `System.Int32`
  storage, and `flags=false`;
- `GO_LANGUAGE_PROJECTION`: zero value equals `None`, arbitrary positive raw
  values remain representable, and a negative raw value remains representable.

The focused package tests and behavior corpus pass with `CNA_NATIVE_LIBRARY`
unset and instantiate no Game, GraphicsDevice, GraphicsDeviceManager,
GraphicsAdapter, PresentationParameters, or render target.

## Runtime and ABI boundary

Completing this enum proves managed metadata only. It does not claim Depth16
GPU support, Depth24 support, stencil support, depth-stencil target support,
backbuffer depth support, allocation, testing, clearing, format selection, or
renderer behavior. No native format mirror, conversion table, interop route,
CNA source change, or ABI expansion exists.

The canonical ABI remains exactly 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants,
with zero missing header/library symbols and zero mismatches.

## Deferred direct reverse dependents

The regenerated public-signature graph records exactly five current direct
reverse dependents:

- `Microsoft.Xna.Framework.Graphics.GraphicsAdapter`;
- `Microsoft.Xna.Framework.Graphics.PresentationParameters`;
- `Microsoft.Xna.Framework.Graphics.RenderTarget2D`;
- `Microsoft.Xna.Framework.Graphics.RenderTargetCube`;
- `Microsoft.Xna.Framework.GraphicsDeviceManager`.

All five remain deferred. No GraphicsAdapter, PresentationParameters,
RenderTarget2D, RenderTargetCube, `PreferredDepthStencilFormat`,
DepthStencilState, depth/stencil runtime, or native API was implemented.
SurfaceFormat and DepthFormat remain independent enums with no public
conversion or combined abstraction.
