# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 13 are complete. Milestone 13 completes exactly:

```text
Microsoft.Xna.Framework.Input.ButtonState
```

Foundation 12 was committed and synchronized at
`033f3a34a03b1229584e2f357c3d7b2295f04eee` before this milestone began, so
Foundation 13 started from a clean `develop` worktree. The Foundation-13 brief
predicted an uncommitted Foundation 12; the actual checkout was authoritative
and that newer published state was preserved. Foundation 13 is fully qualified
and is intentionally uncommitted. Preserve the established namespace, enum,
interop, and measured-absence rules.

## Foundation 13 projection

`ButtonState` belongs directly to the Input package, alongside the already
qualified two-literal ordinary enum `KeyState`. It is a non-flags named `int32`
enum with two explicit constants:

```text
Released=0
Pressed=1
```

The pinned contract contains three CLR field identities: two named literals
plus synthetic `value__`. The established enum-storage rule excludes `value__`,
leaving exactly two mapped Go identities.

There is no `iota`, `// xna:flags`, validation, Stringer, parser,
`IsPressed`/`IsReleased`/`Bool`/`FromBool`/`Toggle` helper, native enum mirror,
or input backend mapping. Go zero equals `Released`; raw `2`, `12345`, and `-1`
remain representable. `Released | Pressed` is an ordinary Go integer expression
with no XNA meaning.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The formal mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=65
TARGET_MEMBERS=1361
TOTAL_DIAGNOSTICS=369
MISSING_TYPE=192
MISSING_MEMBER=177
COMPLETE_TYPES=60
PARTIAL_TYPES=5
MISSING_TYPES=192

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. The selected closure is:

```text
ButtonState source=3 expected-go=2 target-go=2 local=0
kind=enum actual=named underlying=int32 flags=false
value__-excluded=true both-raw-values=PASS status=PASS
```

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177.

## Behavior and verifier evidence

The behavior corpus has 280 observations, 280 assertions, and zero failures.
`BUTTON_STATE=6`: three `PURE_XNA_DERIVED` facts cover the complete two-value
table, `System.Int32`, and `flags=false`; three `GO_LANGUAGE_PROJECTION` facts
cover Released zero, arbitrary positive raw values, and a negative raw value.

The verifier has 157 mutation cases. The 13 Foundation-13 cases reject a
missing or misplaced type, kind/underlying/flags errors, both wrong raw values,
missing `Pressed`, projected `value__`, an invented third `None` constant,
renamed `Release` and `Press` spellings, and an exported `IsPressed` helper.
Both expected literals are generically raw-value checked. A separate
source-level self-test parses the real `button_state.go` declaration and
rejects an accidental `xna:flags` directive at the declaration site. Manual
allowlists and unmeasured categories are zero. No new general mapping rule was
needed, so `docs/xna-go-mapping.md` is unchanged.

GraphicsProfile, DepthFormat, SurfaceFormat, ClearOptions, BufferUsage,
DisplayOrientation/SupportedOrientations, PlayerIndex/Keyboard, VertexElement,
representative ordinary/flags enums, and the 262,400-pattern PackedVector sweep
remain green. Mouse and GamePad have no Go surface at all and therefore have no
regression to run.

## Scope and native preservation

All three direct ButtonState reverse consumers remain deferred: GamePadButtons,
GamePadDPad, and MouseState. All three are `System.ValueType` structs, so this
closure's reverse reach is entirely managed; the transitive reach adds
`GamePadState`, `GamePad`, and `Mouse`. No button property, constructor,
equality/hash, operator, polling route, or device enumeration was added.

`Released` and `Pressed` are metadata values, not runtime input capability
claims. No mouse behavior, gamepad behavior, D-pad behavior, button polling,
connection state, dead zone, vibration, SDL mapping, native enum mirror, or CNA
mapping exists. ABI measurements remain 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants
with zero missing symbols or mismatches.

Native race stress retains 20 Game, recreation, texture, SpriteBatch,
callback-error, and callback-panic cycles, one wrong-thread check, one owner
retry, 80 GC points, and zero crash/UAF/double-free observations.
`GO_RACE_STATUS=PASS`; native sanitizers remain `NOT_RUN`.

The unchanged maintained `cna-go-template` remains clean at
`65254848d9fac02ace934db3879106834bafca97`; test, vet, race, build, trimpath,
and exact 60/600 Draw gates pass.

## Re-running gates

Use Go 1.24.4, Linux amd64, cgo, GCC, and the exact admitted ABI-0.7 library:

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict   # expected 369 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Strict mode rewrites `docs/generated/api-compat-report.json` with
`"mode": "strict"`; rerun `--mode report` last so the committed evidence keeps
its report mode.

`docs/generated/native-abi-report.json` records a normalized `canonical-cna`
header root and the hash of the library admitted in Foundation 11. Verify the
ABI counts out of tree with an explicit `-output` under the scratchpad so a
locally rebuilt ABI-0.7 library or an absolute developer header path never
rewrites that committed evidence.

Source-artifact qualification archives the exact worktree, not bare HEAD, and
uses a fresh extracted consumer with `GOWORK=off` and `-buildvcs=false`.
Archive hash and isolated consumer results belong in the external final handoff
so the archive does not recursively describe its own hash.

## Worktree provenance

Foundation 13 started on clean `develop` with `HEAD` and `origin/develop` both
at `033f3a34a03b1229584e2f357c3d7b2295f04eee`, the committed and synchronized
Foundation-12 milestone. Foundation 13 was qualified without commit or push.
History was not rewritten.

## Dependency-graph counting rule

Dependency completeness counts a node's base type, its declared direct
interfaces, and every member signature type as public-signature dependencies.
Under that rule the regenerated graph now contains exactly 71 missing nodes
with no missing dependency. `GamePadDPad` and `MouseState` became
dependency-complete when `ButtonState` landed; `GamePadButtons` did not,
because it still needs the `Buttons` flags enum.

The historical "71/72" figure came from measuring the same graph without
interface edges. The single node that differs is
`Microsoft.Xna.Framework.GameComponent`, whose direct interfaces
`IGameComponent` and `IUpdateable` are still missing. The interface-inclusive
rule is authoritative. No count of 72 is reproducible under any measured
variant, and the ranking is unaffected in every variant.

## One next dependency-complete milestone

The highest raw reverse fan-out among the 71 dependency-complete nodes still
belongs to `Design.MathTypeConverter` (12) and `Graphics.GraphicsResource`
(11), and then `IEffectMatrices` and `IEffectFog` (5 each). All four remain
excluded on the established grounds: `MathTypeConverter` derives from
`System.ComponentModel.ExpandableObjectConverter` and returns
`PropertyDescriptorCollection`, neither of which the BCL mapping table covers;
`GraphicsResource` is an `IDisposable` runtime base whose public surface
requires the explicitly partial `GraphicsDevice` plus a finalizer and a
disposal event; and the effect interfaces belong to the deferred Effects/3D
family.

The fan-out-three tie group is `IGameComponent` (1 identity),
`Graphics.RenderTargetUsage` (4), `Audio.AudioListener` (5),
`Audio.AudioEmitter` (6), `Graphics.DisplayMode` (6),
`Graphics.CubeMapFace` (7), and `Content.ContentManager` (10).
`IGameComponent` is a Game-runtime lifecycle interface, not a pure managed
public-data leaf, and `AudioListener`, `AudioEmitter`, `DisplayMode`, and
`ContentManager` belong to the deferred audio, device, and content families.
That leaves the two pure managed ordinary enums, and the established
smallest-closure criterion selects `RenderTargetUsage` (four source identities
against seven). The single next selected closure is therefore:

```text
Microsoft.Xna.Framework.Graphics.RenderTargetUsage
```

`RenderTargetUsage` has no missing public-signature dependency; its only type
reference is `System.Int32`. Its package already hosts qualified ordinary
`int32` enums, so no new mapping rule is expected. Selection does not authorize
any consumer — `PresentationParameters`, `RenderTarget2D`, and
`RenderTargetCube` all remain deferred — nor any render-target, texture,
presentation, GPU, or native route, nor another enum. Its pinned metadata and
behavior must be independently inspected in the next milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_13_COMPLETE=true
```
