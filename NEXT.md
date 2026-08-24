# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 10 are complete. Milestone 10 completes exactly:

```text
Microsoft.Xna.Framework.Graphics.SurfaceFormat
```

Foundation 9 is committed and synchronized at
`e402a85bac00bb82ea429a2a6704e9f5d8b0058f`. Foundation 10 is fully qualified
on top and intentionally uncommitted. Preserve the current worktree and the
established namespace, enum, interop, and measured-absence rules.

## Foundation 10 projection

`SurfaceFormat` belongs directly to the Graphics package. It is a non-flags
named `int32` enum with 20 explicit constants:

```text
Color=0
Bgr565=1
Bgra5551=2
Bgra4444=3
Dxt1=4
Dxt3=5
Dxt5=6
NormalizedByte2=7
NormalizedByte4=8
Rgba1010102=9
Rg32=10
Rgba64=11
Alpha8=12
Single=13
Vector2=14
Vector4=15
HalfSingle=16
HalfVector2=17
HalfVector4=18
HdrBlendable=19
```

The milestone request said 20 CLR identities and 19 Go identities, but the
pinned contract and the required table contain 20 named literals plus
synthetic `value__`. The authoritative closure is therefore 21 source
identities and 20 mapped Go identities. Omitting a literal to force 20/19 would
violate the contract.

There is no `iota`, `// xna:flags`, validation, Stringer, parser, size/format
helper, PackedVector dependency, native enum mirror, or GPU mapping. Go zero
equals `Color`; raw `20`, `12345`, and `-1` remain representable.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The formal mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=62
TARGET_MEMBERS=1353
TOTAL_DIAGNOSTICS=372
MISSING_TYPE=195
MISSING_MEMBER=177
COMPLETE_TYPES=57
PARTIAL_TYPES=5
MISSING_TYPES=195

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. The selected closure is:

```text
SurfaceFormat source=21 expected-go=20 target-go=20 local=0
kind=enum actual=named underlying=int32 flags=false
value__-excluded=true all-20-raw-values=PASS status=PASS
```

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177.

## Behavior and verifier evidence

The behavior corpus has 262 observations, 262 assertions, and zero failures.
`SURFACE_FORMAT=6`: three `PURE_XNA_DERIVED` facts cover the complete 20-value
table, `System.Int32`, and `flags=false`; three `GO_LANGUAGE_PROJECTION` facts
cover Color zero, arbitrary positive raw values, and a negative raw value.

The verifier has 117 mutation cases. The 18 Foundation-10 cases reject missing
or misplaced type, kind/underlying/flags errors, representative early/middle/
late raw-value errors, missing middle/last values, projected `value__`, extra
constant, exported helper, and renamed spelling. Every expected literal is
generically raw-value checked. Manual allowlists and unmeasured categories are
zero.

ClearOptions, BufferUsage, DisplayOrientation/SupportedOrientations,
PlayerIndex/Keyboard, VertexElement, representative ordinary/flags enums, and
the 262,400-pattern PackedVector sweep remain green.

## Scope and native preservation

All eleven reverse dependents remain deferred: DisplayMode,
DisplayModeCollection, GraphicsAdapter, PresentationParameters, RenderTarget2D,
RenderTargetCube, Texture, Texture2D's format constructor, Texture3D,
TextureCube, and GraphicsDeviceManager format properties.

No actual pixel-format support is claimed. Compression, HDR, floating-point
targets, upload/data paths, GPU negotiation, renderer mapping, and CNA mapping
remain unimplemented. ABI measurements remain 23 bound functions, 67
prototype type positions, 96 C/Go measurements, 28 layouts, two callbacks,
and five constants with zero missing symbols or mismatches.

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
go run ./tools/api_compat --mode strict       # expected 372 deferred diagnostics
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

Foundation 10 started on clean `develop` with `HEAD` and `origin/develop` both
at `e402a85bac00bb82ea429a2a6704e9f5d8b0058f`, the committed and synchronized
Foundation-9 milestone. Foundation 10 remains intentionally uncommitted after
qualification. History was not rewritten.

## One next dependency-complete milestone

The regenerated 195-node missing inventory and public-signature graph still
contain 72 nodes with no missing XNA signature dependency. Runtime/design
classes and effect interfaces rank higher by raw reverse fan-out but are not
pure managed public-data leaves. Among such leaves, the highest current reverse
fan-out is five. The single next selected closure is therefore:

```text
Microsoft.Xna.Framework.Graphics.DepthFormat
```

`DepthFormat` has no missing public-signature dependency and is referenced by
GraphicsAdapter, PresentationParameters, RenderTarget2D, RenderTargetCube, and
GraphicsDeviceManager. Selection does not authorize any of those consumers,
any SurfaceFormat consumer, depth-buffer runtime behavior, or another enum.
Its pinned metadata and behavior must be independently inspected in the next
milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_10_COMPLETE=true
```
