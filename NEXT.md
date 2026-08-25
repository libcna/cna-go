# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 12 are complete. Milestone 12 completes exactly:

```text
Microsoft.Xna.Framework.Graphics.GraphicsProfile
```

Foundation 11 was committed and synchronized at
`8d86ee1a9815b7c7a71f8b4c623df6580c3a45a5` before this milestone began, so
Foundation 12 started from a clean `develop` worktree. Foundation 12 is fully
qualified and is intentionally uncommitted. Preserve the established namespace,
enum, interop, and measured-absence rules.

## Foundation 12 projection

`GraphicsProfile` belongs directly to the Graphics package. It is a non-flags
named `int32` enum with two explicit constants:

```text
Reach=0
HiDef=1
```

The pinned contract contains three CLR field identities: two named literals
plus synthetic `value__`. The established enum-storage rule excludes `value__`,
leaving exactly two mapped Go identities.

There is no `iota`, `// xna:flags`, validation, Stringer, parser,
`IsReach`/`IsHiDef`/`SupportsHiDef`/`FeatureLevel` helper, native enum mirror,
or GPU capability mapping. Go zero equals `Reach`; raw `2`, `12345`, and `-1`
remain representable. `Reach | HiDef` is an ordinary Go integer expression with
no XNA meaning.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The formal mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=64
TARGET_MEMBERS=1359
TOTAL_DIAGNOSTICS=370
MISSING_TYPE=193
MISSING_MEMBER=177
COMPLETE_TYPES=59
PARTIAL_TYPES=5
MISSING_TYPES=193

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. The selected closure is:

```text
GraphicsProfile source=3 expected-go=2 target-go=2 local=0
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

The behavior corpus has 274 observations, 274 assertions, and zero failures.
`GRAPHICS_PROFILE=6`: three `PURE_XNA_DERIVED` facts cover the complete
two-value table, `System.Int32`, and `flags=false`; three
`GO_LANGUAGE_PROJECTION` facts cover Reach zero, arbitrary positive raw values,
and a negative raw value.

The verifier has 144 mutation cases. The 12 Foundation-12 cases reject a
missing or misplaced type, kind/underlying/flags errors, both wrong raw values,
missing `HiDef`, projected `value__`, an invented third constant, a renamed
`Hidef` spelling, and an exported String helper. Both expected literals are
generically raw-value checked. Manual allowlists and unmeasured categories are
zero. No new general mapping rule was needed, so `docs/xna-go-mapping.md` is
unchanged.

DepthFormat, SurfaceFormat, ClearOptions, BufferUsage,
DisplayOrientation/SupportedOrientations, PlayerIndex/Keyboard, VertexElement,
representative ordinary/flags enums, and the 262,400-pattern PackedVector sweep
remain green.

## Scope and native preservation

All four direct GraphicsProfile reverse dependents remain deferred:
GraphicsAdapter, GraphicsDevice, GraphicsDeviceInformation, and
GraphicsDeviceManager. `GraphicsDevice.GraphicsProfile` and the
`GraphicsDeviceManagerGraphicsProfile`/`SetGraphicsDeviceManagerGraphicsProfile`
pair were deliberately not implemented.

`Reach` and `HiDef` are metadata values, not runtime GPU capability claims. No
Reach support, HiDef support, profile selection, feature-level detection,
shader-model mapping, renderer capability mapping, texture/render-target limit,
native profile negotiation, native enum mirror, or CNA mapping exists. ABI
measurements remain 23 bound functions, 67 prototype type positions, 96 C/Go
measurements, 28 layouts, two callbacks, and five constants with zero missing
symbols or mismatches.

Native race stress retains 20 Game, recreation, texture, SpriteBatch,
callback-error, and callback-panic cycles, 80 GC points, and zero
crash/UAF/double-free observations. `GO_RACE_STATUS=PASS`; native sanitizers
remain `NOT_RUN`.

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
go run ./tools/api_compat --mode strict   # expected 370 deferred diagnostics
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

Source-artifact qualification archives the exact worktree, not bare HEAD, and
uses a fresh extracted consumer with `GOWORK=off` and `-buildvcs=false`.
Archive hash and isolated consumer results belong in the external final handoff
so the archive does not recursively describe its own hash.

## Worktree provenance

Foundation 12 started on clean `develop` with `HEAD` and `origin/develop` both
at `8d86ee1a9815b7c7a71f8b4c623df6580c3a45a5`, the committed and synchronized
Foundation-11 milestone. The supplied brief predicted an uncommitted
Foundation 11; the actual checkout was authoritative and that newer published
state was preserved. Foundation 12 was qualified without commit or push.
History was not rewritten.

## One next dependency-complete milestone

The regenerated 193-node missing inventory and public-signature graph contain
71 nodes with no missing XNA signature dependency. The highest raw reverse
fan-out belongs to `Design.MathTypeConverter` (12) and
`Graphics.GraphicsResource` (11), but neither is a pure managed public-data
leaf. `MathTypeConverter` derives from
`System.ComponentModel.ExpandableObjectConverter` and returns
`PropertyDescriptorCollection`, neither of which the BCL mapping table covers.
`GraphicsResource` is an `IDisposable` runtime base whose
public surface requires the explicitly partial `GraphicsDevice` plus a
finalizer and a disposal event. The effect interfaces `IEffectMatrices` and
`IEffectFog` (5 each) belong to the deferred Effects/3D family, and the
remaining higher-ranked nodes are deferred game/content/audio runtime types.

Among pure managed public-data leaves the highest current reverse fan-out is
three, held by `Input.ButtonState`, `Graphics.RenderTargetUsage`, and
`Graphics.CubeMapFace`. `ButtonState` wins the tie on both established
criteria: it has the smallest closure (three source identities against four and
seven), and its reverse reach is entirely managed — `GamePadButtons`,
`GamePadDPad`, and `MouseState` are all `System.ValueType` structs — whereas
the other two lead directly into the deferred native render-target, texture,
and device families. The single next selected closure is therefore:

```text
Microsoft.Xna.Framework.Input.ButtonState
```

`ButtonState` has no missing public-signature dependency; its only type
reference is `System.Int32`. Its package already hosts the qualified two-literal
ordinary enum `KeyState`, so no new mapping rule is expected. Selection does not
authorize any consumer, GamePad or Mouse surface, input polling, native route,
or another enum. Its pinned metadata and behavior must be independently
inspected in the next milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_12_COMPLETE=true
```
