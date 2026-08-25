# Foundation Milestone 13 ButtonState evidence

## Authority and exact one-type closure

Foundation Milestone 13 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Input.ButtonState
```

The public authority is the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The entry was inspected independently of the supplied brief and matches it
exactly. It has CLR kind `enum`, `sealed=true`, base `System.Enum`,
`System.Int32` underlying storage, `flags=false`, no direct interfaces, no
generic parameters, and no declared constructor, method, property, event, or
operator. The inherited `System.Enum` interface list (`IComparable`,
`IConvertible`, `IFormattable`) is CLR base surface and contributes nothing to
the direct Go public projection.

The exact closure contains three source field identities: synthetic `value__`
plus two named literals. The established enum-storage mapping excludes only
`value__`, producing exactly two expected and two target Go identities.

`ButtonState` references no other XNA type. Its only signature type is
`System.Int32`, so its public-signature dependency set is empty.

## Complete raw-value table

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Released` | `ButtonStateReleased` | 0 |
| `Pressed` | `ButtonStatePressed` | 1 |

Both constants are explicitly assigned. Production contains no `iota`, alias,
or alternate spelling. There is no invented `None`, `Up`, `Down`, `Default`,
`Unknown`, `Held`, or `Release`/`Press` literal.

## Go projection and raw domain

`ButtonState` belongs directly to namespace `Microsoft.Xna.Framework.Input` and
package `input`, in the dedicated file
`Microsoft/Xna/Framework/Input/button_state.go`. It is neither placed under
`Graphics` nor at the `Framework` root. It is a named `int32` type with no
`// xna:flags` directive. The Go zero value equals `ButtonStateReleased`.

The complete signed `int32` domain remains representable. Focused tests retain
`ButtonState(2)`, `ButtonState(12345)`, and `ButtonState(-1)` exactly. There is
no membership validation, clamping, normalization, constructor, default helper,
panic, or error-return conversion. Ruby-style and Swift-style unknown-value
enum semantics are engineering comparators only and are deliberately not
copied. This permissive raw domain is `GO_LANGUAGE_PROJECTION` evidence, never
a claim about XNA runtime behavior.

`ButtonState` is not a flags enum. `Released | Pressed` is an ordinary Go
integer expression with no XNA meaning and is neither documented nor qualified
as a composition. There is no `HasFlag`, `Contains`, `Or`, `And`, or `Mask`
API. `Released=0` and `Pressed=1` are ordinary enum alternatives, not bits.

There is no exported `String`, `MarshalText`, `UnmarshalText`,
`ParseButtonState`, `Name`, `IsPressed`, `IsReleased`, `Bool`, `FromBool`,
`Toggle`, or other helper API. No `ButtonStateValue` or `value__` identity is
exposed.

`ButtonState` required no new general mapping rule. It reuses the established
ordinary non-flags enum projection unchanged — the same projection its own
package already applies to the two-literal `KeyState` enum — so
`docs/xna-go-mapping.md` is unmodified.

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

The whole-profile scoreboard moves from 64 to 65 target types and from 1,359 to
1,361 target members, so genuine deferred diagnostics fall from 370 to 369 and
missing types from 193 to 192 while complete types rise from 59 to 60. Missing
members remain 177 across the unchanged five partial types. Every global
structural preservation, leak, allowlist, and unmeasured counter remains zero;
all 25 PackedVector interface witnesses (17 `PackFromVector4`, eight
`ToVector4`) remain unchanged.

The verifier inventory grows from 144 to 157 cases. The 13 Foundation-13
mutations reject a missing type, a wrongly packaged type, a wrong Go kind, a
wrong underlying type, an accidental `xna:flags` classification,
`Released != 0`, `Pressed != 1`, a missing `Pressed`, a projected `value__`, an
invented third `None` constant, a renamed `Release` spelling, a renamed `Press`
spelling, and an exported `IsPressed` helper method. Both constants are also
measured by the generic ordinary-enum raw-value verifier; nothing about
`ButtonState` is special-cased and no allowlist or unmeasured structural
category exists.

The brief's fourteen requested negative concerns map onto thirteen mutation
fixtures because `flags=true` and an accidental `xna:flags` marker are the same
measured defect in this verifier: the `// xna:flags` doc directive is the sole
channel through which a Go type is classified as a flags enum, and both are
rejected by `button-state-accidentally-flags` as `FLAGS_MAPPING_MISMATCH`. The
fourteenth concern is covered instead by a distinct source-level self-test,
`TestButtonStateSourceCarriesNoFlagsDirective`, which parses the real
`button_state.go` declaration with `go/parser`, asserts no `xna:flags`
directive is attached, and then proves the detector rejects a mutated
declaration that carries one. That test measures the actual declaration site
rather than an extractor model field.

## Behavior provenance

The corpus grows from 274 to 280 observations/assertions with zero failures.
`BUTTON_STATE` contributes six observations:

- `PURE_XNA_DERIVED` (three): the complete two-value raw table `Released=0` and
  `Pressed=1`, `System.Int32` storage, and `flags=false`;
- `GO_LANGUAGE_PROJECTION` (three): zero value equals `Released`, arbitrary
  positive raw values remain representable, and a negative raw value remains
  representable.

The two provenance categories are never conflated. Go's permissive named-`int32`
raw domain is labeled `GO_LANGUAGE_PROJECTION`, never as XNA runtime behavior.

The focused package tests and behavior corpus pass with `CNA_NATIVE_LIBRARY`
unset and instantiate no Game, GraphicsDevice, Mouse, GamePad, native loader,
cgo route, or hardware input device. This is pure managed metadata.

## Runtime and ABI boundary

Completing this enum proves managed metadata only. `Released` and `Pressed` are
metadata values, not runtime input capability claims. Nothing here claims mouse
button behavior, gamepad button behavior, D-pad behavior, button polling, input
device enumeration, connection state, dead-zone handling, vibration, or any
other hardware observation. Existing Keyboard hardware claims are unchanged and
unrelated; no Mouse or GamePad capability claim exists in the inventory because
neither type is implemented.

No native mirror (`CNA_BUTTON_STATE_RELEASED`, `CNA_BUTTON_STATE_PRESSED`, or
any other), C function, SDL mapping, native enum conversion, manifest entry,
layout, or constant measurement was added, and no Foundation-13 selected API
crosses the C ABI. No CNA source was changed.

The canonical ABI remains exactly 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants,
with zero missing header symbols, zero missing library symbols, and zero
mismatches.

## Deferred direct reverse dependents

The regenerated public-signature graph records exactly three current direct
reverse consumers of `ButtonState`, with a direct fan-out of three:

| Consumer | CLR base | Referencing signature |
|---|---|---|
| `Microsoft.Xna.Framework.Input.GamePadButtons` | `System.ValueType` | the `A`, `B`, `Back`, `X`, `Y`, `Start`, `LeftShoulder`, `LeftStick`, `RightShoulder`, `RightStick`, and `BigButton` properties |
| `Microsoft.Xna.Framework.Input.GamePadDPad` | `System.ValueType` | the `.ctor` parameters and the `Up`, `Down`, `Right`, `Left` properties |
| `Microsoft.Xna.Framework.Input.MouseState` | `System.ValueType` | the `.ctor` parameters and the `LeftButton`, `RightButton`, `MiddleButton`, `XButton1`, `XButton2` properties |

All three are `System.ValueType` structs, which is why this closure's reverse
reach is entirely managed. The transitive reverse reach is six types: the three
direct consumers plus `GamePadState`, `GamePad`, and `Mouse`.

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

None of these were implemented. A completed parameter or property type never
authorizes its consumer:

- `GamePadButtons` is not implemented — no button property, constructor,
  equality, hash, or operator surface exists.
- `GamePadDPad` is not implemented — no `Up`/`Down`/`Left`/`Right` property,
  constructor, or equality surface exists.
- `MouseState` is not implemented — no `LeftButton`, `MiddleButton`,
  `RightButton`, `XButton1`, `XButton2`, position, or scroll-wheel surface
  exists.
- `GamePad`, `GamePadState`, `GamePadCapabilities`, `GamePadThumbSticks`,
  `GamePadTriggers`, `GamePadDeadZone`, `GamePadType`, `Buttons`, and `Mouse`
  likewise remain unimplemented. No input backend, SDL route, polling loop, or
  device enumeration was added.

## Dependency-graph counting nuance

The Foundation-12 handoff recorded 71 missing nodes with no missing public
signature dependency, and the Foundation-13 brief cited "approximately 71/72".
Regenerating under the current methodology gives an exact count of **70**.

The single differing node is `Microsoft.Xna.Framework.GameComponent`. It is
dependency-complete when only its base type and member signatures are treated
as public-signature dependencies, and not dependency-complete when its declared
direct interfaces are also treated as dependencies, because
`Microsoft.Xna.Framework.IGameComponent` and
`Microsoft.Xna.Framework.IUpdateable` are themselves still missing. The
interface-inclusive rule is the stricter and correct one — an implemented Go
type cannot honestly project a CLR interface list whose interfaces do not
exist — so 70 is the reported count and 71 is the same graph measured without
interface edges. No count of 72 is reproducible under any measured variant
(base+interfaces 70, without interfaces 71, without base 88, members only 90;
treating the five partial types as unsatisfying gives 68, 68, 83, and 83
respectively).

The selection semantics were not altered to reproduce the older headline
number. The ranking is unaffected: `GameComponent` has a reverse fan-out of two
and therefore sits below the fan-out-three tie group in every variant.

## Selection provenance

`ButtonState` was selected by the established rule. The highest raw reverse
fan-out among dependency-complete missing nodes belongs to
`Design.MathTypeConverter` (12) and `Graphics.GraphicsResource` (11), and then
`IEffectMatrices` and `IEffectFog` (5 each); all four were excluded on the
established grounds recorded in the Foundation-12 handoff — unmapped
`System.ComponentModel` BCL surface, an `IDisposable` runtime base tied to the
explicitly partial `GraphicsDevice`, and the deliberately deferred effect
family. Among small pure-managed leaves the top fan-out is three, held by
`Input.ButtonState`, `Graphics.RenderTargetUsage`, and `Graphics.CubeMapFace`
(alongside deferred runtime nodes `IGameComponent`, `AudioListener`,
`AudioEmitter`, `DisplayMode`, and `ContentManager`). `ButtonState` won the tie
on both established criteria: the smallest closure (three source identities
against four and seven) and a reverse reach that is entirely managed value
structures, whereas the other two lead directly into the deferred
render-target, texture, and device families.
