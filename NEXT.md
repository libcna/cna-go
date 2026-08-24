# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 11 are complete. Milestone 11 completes exactly:

```text
Microsoft.Xna.Framework.Graphics.DepthFormat
```

Foundation 10 is committed and synchronized at
`06d084afbd2414af826c8219116e74aaec96a4a4`. Foundation 11 is fully
qualified on top. Publication was explicitly authorized after qualification.
Preserve the established namespace, enum, interop, and measured-absence rules.

## Foundation 11 projection

`DepthFormat` belongs directly to the Graphics package. It is a non-flags
named `int32` enum with four explicit constants:

```text
None=0
Depth16=1
Depth24=2
Depth24Stencil8=3
```

The pinned contract contains five CLR field identities: four named literals
plus synthetic `value__`. The established enum-storage rule excludes
`value__`, leaving exactly four mapped Go identities.

There is no `iota`, `// xna:flags`, validation, Stringer, parser,
depth/stencil-bit helper, SurfaceFormat conversion, native enum mirror, or GPU
mapping. Go zero equals `None`; raw `4`, `12345`, and `-1` remain
representable.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The formal mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=63
TARGET_MEMBERS=1357
TOTAL_DIAGNOSTICS=371
MISSING_TYPE=194
MISSING_MEMBER=177
COMPLETE_TYPES=58
PARTIAL_TYPES=5
MISSING_TYPES=194

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. The selected closure is:

```text
DepthFormat source=5 expected-go=4 target-go=4 local=0
kind=enum actual=named underlying=int32 flags=false
value__-excluded=true all-4-raw-values=PASS status=PASS
```

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177.

## Behavior and verifier evidence

The behavior corpus has 268 observations, 268 assertions, and zero failures.
`DEPTH_FORMAT=6`: three `PURE_XNA_DERIVED` facts cover the complete
four-value table, `System.Int32`, and `flags=false`; three
`GO_LANGUAGE_PROJECTION` facts cover None zero, arbitrary positive raw values,
and a negative raw value.

The verifier has 132 mutation cases. The 15 Foundation-11 cases reject missing
or misplaced type, kind/underlying/flags errors, every wrong raw value, missing
middle/last values, projected `value__`, an extra constant, an exported
String helper, and renamed spelling. Every expected literal is generically
raw-value checked. Manual allowlists and unmeasured categories are zero.

SurfaceFormat, ClearOptions, BufferUsage,
DisplayOrientation/SupportedOrientations, PlayerIndex/Keyboard, VertexElement,
representative ordinary/flags enums, and the 262,400-pattern PackedVector sweep
remain green.

## Scope and native preservation

All five direct DepthFormat reverse dependents remain deferred:
GraphicsAdapter, PresentationParameters, RenderTarget2D, RenderTargetCube, and
GraphicsDeviceManager.

No actual depth/stencil format support is claimed. Depth/stencil allocation,
attachments, render targets, backbuffers, presentation configuration, testing,
clearing, GPU negotiation, renderer mapping, and CNA mapping remain
unimplemented. ABI measurements remain 23 bound functions, 67 prototype type
positions, 96 C/Go measurements, 28 layouts, two callbacks, and five constants
with zero missing symbols or mismatches.

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
go run ./tools/api_compat --mode strict       # expected 371 deferred diagnostics
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

Foundation 11 started on clean `develop` with `HEAD` and
`origin/develop` both at
`06d084afbd2414af826c8219116e74aaec96a4a4`, the committed and synchronized
Foundation-10 milestone. Foundation 11 was qualified without commit or push;
publication was subsequently explicitly authorized. History was not rewritten.

## One next dependency-complete milestone

The regenerated 194-node missing inventory and public-signature graph contain
71 nodes with no missing XNA signature dependency. Runtime/design classes and
effect interfaces rank higher by raw reverse fan-out but are not small pure
managed public-data leaves. Among those leaves, the highest current reverse
fan-out is four. The single next selected closure is therefore:

```text
Microsoft.Xna.Framework.Graphics.GraphicsProfile
```

`GraphicsProfile` has no missing public-signature dependency and is
referenced by GraphicsAdapter, GraphicsDevice, GraphicsDeviceInformation, and
GraphicsDeviceManager. Selection does not authorize any consumer, device
profile behavior, capability query, native mapping, or another enum. Its pinned
metadata and behavior must be independently inspected in the next milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_11_COMPLETE=true
```
