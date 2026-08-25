# Foundation Milestone 12 GraphicsProfile evidence

## Authority and exact one-type closure

Foundation Milestone 12 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Graphics.GraphicsProfile
```

The public authority is the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The independently inspected entry has CLR kind `enum`, `sealed=true`, base
`System.Enum`, `System.Int32` underlying storage, `flags=false`, no direct
interfaces, and no declared constructor, method, property, event, or operator.
The inherited `System.Enum` interface list (`IComparable`, `IConvertible`,
`IFormattable`) is CLR base surface and contributes nothing to the direct Go
public projection.

The exact closure contains three source field identities: synthetic `value__`
plus two named literals. The established enum-storage mapping excludes only
`value__`, producing exactly two expected and two target Go identities.

## Complete raw-value table

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Reach` | `GraphicsProfileReach` | 0 |
| `HiDef` | `GraphicsProfileHiDef` | 1 |

Both constants are explicitly assigned. Production contains no `iota`, alias,
or alternate spelling. There is no invented `Default`, `Low`, `High`,
`ReachProfile`, or `HiDefProfile` literal.

## Go projection and raw domain

`GraphicsProfile` belongs directly to namespace
`Microsoft.Xna.Framework.Graphics` and package `graphics`, in the dedicated
file `Microsoft/Xna/Framework/Graphics/graphics_profile.go`. It is a named
`int32` type with no `// xna:flags` directive. The Go zero value equals
`GraphicsProfileReach`.

The complete signed `int32` domain remains representable. Focused tests retain
`GraphicsProfile(2)`, `GraphicsProfile(12345)`, and `GraphicsProfile(-1)`
exactly. There is no membership validation, clamping, normalization,
constructor, default helper, panic, or error-return conversion. Ruby-style and
Swift-style unknown-value enum semantics are engineering comparators only and
are deliberately not copied.

`GraphicsProfile` is not a flags enum. `Reach | HiDef` is an ordinary Go
integer expression with no XNA meaning and is neither documented nor qualified
as a composition. There is no `HasFlag`, `Contains`, `Or`, `And`, or `Mask`
API.

There is no exported `String`, `MarshalText`, `UnmarshalText`,
`ParseGraphicsProfile`, `Name`, `IsReach`, `IsHiDef`, `FeatureLevel`,
`SupportsHiDef`, or other helper API. No `GraphicsProfileValue` or `value__`
identity is exposed.

## Structural and mutation evidence

The compiler-extracted local closure is:

| Source identities | Expected Go identities | Target Go identities | Local diagnostics | Kind | Go kind | Underlying | Flags | `value__` excluded |
|---:|---:|---:|---:|---|---|---|---|---|
| 3 | 2 | 2 | 0 | enum | named | `System.Int32` / `int32` | false | true |

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

The whole-profile scoreboard moves from 63 to 64 target types and from 1,357 to
1,359 target members, so genuine deferred diagnostics fall from 371 to 370 and
missing types from 194 to 193 while complete types rise from 58 to 59. Missing
members remain 177 across the unchanged five partial types. Every global
structural preservation, leak, allowlist, and unmeasured counter remains zero;
all 25 PackedVector interface witnesses (17 `PackFromVector4`, eight
`ToVector4`) remain unchanged.

The verifier inventory grows from 132 to 144 cases. The 12 Foundation-12
mutations reject a missing type, a wrongly packaged type, a wrong Go kind, a
wrong underlying type, an accidental `xna:flags` classification, `Reach != 0`,
`HiDef != 1`, a missing `HiDef`, a projected `value__`, an invented third
constant, a renamed `Hidef` spelling, and an exported `String` helper method.
Both constants are also measured by the generic ordinary-enum raw-value
verifier; nothing about `GraphicsProfile` is special-cased and no allowlist or
unmeasured structural category exists.

`GraphicsProfile` required no new general mapping rule. It reuses the
established ordinary non-flags enum projection unchanged, so
`docs/xna-go-mapping.md` is unmodified.

## Behavior provenance

The corpus grows from 268 to 274 observations/assertions with zero failures.
`GRAPHICS_PROFILE` contributes six observations:

- `PURE_XNA_DERIVED` (three): the complete two-value raw table `Reach=0` and
  `HiDef=1`, `System.Int32` storage, and `flags=false`;
- `GO_LANGUAGE_PROJECTION` (three): zero value equals `Reach`, arbitrary
  positive raw values remain representable, and a negative raw value remains
  representable.

Go's permissive named-`int32` raw domain is labeled `GO_LANGUAGE_PROJECTION`,
never as XNA runtime behavior.

The focused package tests and behavior corpus pass with `CNA_NATIVE_LIBRARY`
unset and instantiate no Game, GraphicsDevice, GraphicsDeviceManager,
GraphicsAdapter, GraphicsDeviceInformation, native loader, or cgo route. This
is pure managed metadata.

## Runtime and ABI boundary

Completing this enum proves managed metadata only. `Reach` and `HiDef` are
metadata values, not runtime GPU capability claims. Nothing here claims actual
Reach support, actual HiDef support, graphics profile selection, feature-level
detection, shader-model support, renderer capability mapping, texture or
render-target limits, or native profile negotiation. No hardware or shader
model was inspected in order to assign a profile.

No native mirror (`CNA_GRAPHICS_PROFILE_REACH`, `CNA_GRAPHICS_PROFILE_HIDEF`,
or any other), C function, native enum conversion, manifest entry, layout, or
constant measurement was added, and no Foundation-12 selected API crosses the C
ABI. No CNA source was changed.

The canonical ABI remains exactly 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants,
with zero missing header symbols, zero missing library symbols, and zero
mismatches.

## Deferred direct reverse dependents

The regenerated public-signature graph records exactly four current direct
reverse dependents of `GraphicsProfile`:

| Consumer | Referencing signature |
|---|---|
| `Microsoft.Xna.Framework.Graphics.GraphicsAdapter` | `QueryBackBufferFormat`, `QueryRenderTargetFormat`, `IsProfileSupported` parameters |
| `Microsoft.Xna.Framework.Graphics.GraphicsDevice` | `.ctor` parameter and the `GraphicsProfile` property |
| `Microsoft.Xna.Framework.GraphicsDeviceInformation` | the `GraphicsProfile` property |
| `Microsoft.Xna.Framework.GraphicsDeviceManager` | the `GraphicsProfile` property |

All four remain deferred. A completed parameter or property type does not
authorize its consumer:

- `GraphicsAdapter` is not implemented — no adapter enumeration, profile
  support query, current display mode, or device capability API exists.
- `GraphicsDeviceInformation` is not implemented — no adapter/profile pairing,
  presentation configuration, or `PreparingDeviceSettings` expansion exists.
- `GraphicsDevice.GraphicsProfile` is not implemented. No fake `Reach` or
  `HiDef` is returned, no profile state is cached, and CNA is not asked.
  `GraphicsDevice` remains partial with exactly 70 missing members, including
  its `GraphicsProfile` getter and its three-argument constructor.
- `GraphicsDeviceManager.GraphicsProfile` is not implemented. Its two mapped Go
  identities, `GraphicsDeviceManagerGraphicsProfile` and
  `SetGraphicsDeviceManagerGraphicsProfile`, remain deferred with no
  dirty-state behavior, `ApplyChanges` integration, or preferred-profile state.
  `GraphicsDeviceManager` remains partial with exactly 40 missing members.

`PresentationParameters`, `DisplayMode`, device creation and profile selection,
and Reach/HiDef feature detection likewise remain unimplemented.
