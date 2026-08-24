# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 9 are complete. Milestone 9 completes exactly:

```text
Microsoft.Xna.Framework.Graphics.ClearOptions
```

Foundation 8 is committed at `68f2a7e9e556c6b5ef0d4d8b360fa1b46673133d`.
Foundation 9 is qualified on top and intentionally uncommitted. Preserve the
current worktree and the established namespace, value/class, interop, and
measured-absence rules.

## Foundation 9 projection

`ClearOptions` belongs directly to the Graphics package. It is a `[Flags]`
named `int32` with exactly these declared constants:

```text
ClearOptionsTarget=1
ClearOptionsDepthBuffer=2
ClearOptionsStencil=4
```

The synthetic CLR `value__` storage is excluded, so four source identities map
to exactly three Go identities. Most importantly, XNA declares no zero-valued
field. Raw zero is the unnamed Go zero value; `ClearOptionsNone`,
`ClearOptionsDefault`, and `ClearOptionsAll` do not exist.

Declared combinations produce 3, 5, 6, and 7. Raw `0`, `8`, `1<<20`, and `-1`
remain representable without validation or masking. Typed OR and AND are
ordinary Go operations. No `String`, flags helper, constructor, validator, or
other convenience API was added.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=61
TARGET_MEMBERS=1333
TOTAL_DIAGNOSTICS=373
MISSING_TYPE=196
MISSING_MEMBER=177
COMPLETE_TYPES=56
PARTIAL_TYPES=5
MISSING_TYPES=196

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. The selected closure is:

```text
ClearOptions source=4 expected-go=3 target-go=3 local=0
kind=enum actual=named underlying=int32 flags=true
Target=1 DepthBuffer=2 Stencil=4 value__-excluded=true
named-zero=false None=false Default=false All=false status=PASS
```

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177.

## Behavior and verifier evidence

The behavior corpus has 256 observations, 256 assertions, and zero failures.
`CLEAR_OPTIONS=9`: four `PURE_XNA_DERIVED` metadata/value facts and five
explicitly labeled `GO_LANGUAGE_PROJECTION` observations for zero, raw values,
combinations, OR, and AND.

The verifier has 99 mutation cases. The 14 Foundation-9 cases reject missing
or misplaced type, kind/underlying/flags errors, all three wrong literals,
missing Stencil, projected `value__`, invented `None`, invented `All`, and an
exported helper. A generic fixture proves that flags metadata with three
nonzero bits and no named zero is valid. Manual allowlists and unmeasured
categories remain zero.

BufferUsage, DisplayOrientation/SupportedOrientations, PlayerIndex/Keyboard,
ordinary enums, and the 262,400-pattern PackedVector sweep remain green.

## Scope and native preservation

Neither reverse-dependent GraphicsDevice overload was implemented:

```text
GraphicsDevice.Clear(ClearOptions, Color, Single, Int32)
GraphicsDevice.Clear(ClearOptions, Vector4, Single, Int32)
```

No other GraphicsDevice member, render-target/depth/stencil clear behavior,
renderer command, CNA constant, or ABI route was added. ABI measurements remain
23 bound functions, 67 prototype type positions, 96 C/Go measurements, 28
layouts, two callbacks, and five constants with zero missing symbols or
mismatches.

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
go run ./tools/api_compat --mode strict       # expected 373 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Source-artifact qualification archives the exact worktree, not bare HEAD, and
uses a fresh extracted consumer with `GOWORK=off`. Archive hash and isolated
consumer results belong in the external final handoff so the archive does not
recursively describe its own hash.

## Worktree provenance

Foundation 9 started on clean `develop` with `HEAD` and `origin/develop` both
at `68f2a7e9e556c6b5ef0d4d8b360fa1b46673133d`, the externally committed
Foundation-8 milestone. Foundation 9 remains intentionally uncommitted after
qualification. History was not rewritten.

## One next dependency-complete milestone

The regenerated 196-node missing inventory and public-signature dependency
graph contain 72 nodes with no missing XNA signature dependency. Among pure
managed public-data leaves, the maximal reverse fan-out is 11 consumers. The
single next selected closure is therefore:

```text
Microsoft.Xna.Framework.Graphics.SurfaceFormat
```

Selection is dependency-driven: SurfaceFormat is referenced by DisplayMode,
DisplayModeCollection, GraphicsAdapter, PresentationParameters, RenderTarget2D,
RenderTargetCube, Texture, Texture2D, Texture3D, TextureCube, and
GraphicsDeviceManager. This selection does not authorize any consumer,
GraphicsDevice.Clear overload, render-target operation, texture constructor,
buffer API, VertexDeclaration, or another enum. Its pinned contract and
behavior must be independently inspected in the next milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_9_COMPLETE=true
```
