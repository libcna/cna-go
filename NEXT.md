# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 8 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestones 2 through 7 completed the
geometry/transform, Curve, PackedVector, VertexElement, PlayerIndex/Keyboard,
and DisplayOrientation/SupportedOrientations closures. Milestone 8 completes
exactly:

```text
Microsoft.Xna.Framework.Graphics.BufferUsage
```

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 8 projection and behavior

`BufferUsage` is directly in the Graphics package. It is a `[Flags]` named
`int32` enum with explicit `None=0` and `WriteOnly=1` constants. The synthetic
CLR enum-storage field `value__` is excluded, so its three source identities
map to exactly two Go identities.

The Go zero value equals `BufferUsageNone`. The complete signed `int32` domain,
including values such as `2`, `3`, `1<<20`, and `-1`, remains representable.
Ordinary typed bitwise operations compose both declared and unknown bits. There
is no `iota`, validator, normalization, `String`, flags helper, count constant,
constructor, or convenience member.

This is pure managed enum metadata. No `VertexBuffer`, `DynamicVertexBuffer`,
`IndexBuffer`, `DynamicIndexBuffer`, `VertexDeclaration`, `IVertexType`, data
transfer, buffer creation/binding, draw operation, GraphicsDevice member, CNA
constant, or native ABI route was added. Exact evidence is in
`docs/buffer-usage-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=60
TARGET_MEMBERS=1330
TOTAL_DIAGNOSTICS=374
MISSING_TYPE=197
MISSING_MEMBER=177
COMPLETE_TYPES=55
PARTIAL_TYPES=5
MISSING_TYPES=197

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. The dedicated selected-closure measurement is:

```text
BufferUsage source=3 expected-go=2 target-go=2 local=0
kind=enum actual=named underlying=int32 flags=true
None=0 WriteOnly=1 value__-excluded=true status=PASS
```

The exact remaining partial types are:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177.

## Behavioral and verifier evidence

The managed corpus retains `PURE_XNA_DERIVED` as the XNA authority and labels
Go-only projection evidence per observation. It has 247 observations, 247
assertions, and zero failures. Foundation 8 contributes:

```text
BUFFER_USAGE=5
PURE_XNA_DERIVED=2
GO_LANGUAGE_PROJECTION=3
```

The XNA-derived observations cover exact raw literals and `Int32` projection.
The Go-projection observations cover zero value, arbitrary raw signed values,
and bitwise composition. Generic Go behavior is not labeled as an XNA runtime
observation.

The verifier has 85 mutation cases. The 12 Foundation 8 additions reject a
missing type, wrong package, wrong kind, wrong underlying type, absent marker,
false flags classification, wrong values, missing `WriteOnly`, projected
`value__`, an extra constant, and an exported helper. Exact directive parsing
also rejects `// xna:flags=false`. `ALLOWLIST_ENTRIES=0` and
`UNMEASURED_STRUCTURAL_CATEGORY=0`.

Foundation 7's default, same-value setter, changed setter, raw-bit storage,
dirty-state, and post-disposal behavior remains green. Foundation 6's four
PlayerIndex values, undefined raw value, and both Keyboard routes remain green.

## ABI, native, and template regression

Milestone 8 adds no native function, cgo function, manifest entry, layout,
callback, or constant. The admitted HEADLESS/NULL-audio ABI 0.7.0 artifact
retains SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`:

```text
BOUND_FUNCTIONS=23
PROTOTYPE_TYPE_POSITIONS=67
C_GO_MEASUREMENTS=96
LAYOUTS=28
CALLBACKS=2
CONSTANTS=5
MISSING_HEADER_SYMBOLS=0
MISSING_LIBRARY_SYMBOLS=0
ABI_MISMATCHES=0
```

Go-race native stress retains 20 Game, recreation, texture, SpriteBatch,
callback-error, and callback-panic cycles, 80 forced-GC points, and zero
crash/UAF/double-free observations. `GO_RACE_STATUS=PASS`; native sanitizers
remain `NOT_RUN`.

The maintained sibling `cna-go-template` source remains unchanged at commit
`65254848d9fac02ace934db3879106834bafca97`. Its test, vet, race, build,
trimpath, exact 60-Draw, and exact 600-Draw gates pass. No BufferUsage sample,
buffer allocation, or 3D change was added.

## Re-running gates

Use Go 1.24.4 with Linux amd64 cgo and GCC. Set `CNA_NATIVE_LIBRARY` to an
absolute exact ABI-0.7 library path for native gates.

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict       # expected 374 missing diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Source-artifact and isolated-consumer qualification use a deterministic archive
of the exact worktree, not bare HEAD or the development checkout. The external
milestone handoff records archive name, hash, entry count, two-pass
determinism, audit counters, and isolated 60/600 results so the hash is not
recursively embedded in the archive it describes.

## Worktree provenance

The actual checkout at the start of Foundation 8 was clean on `develop` at
`36ba8b01f5c87ffb1ebf3aa84e4291736b6b0efe`; `origin/develop` matched.
Foundation 7 is committed at that revision as
`feat: complete display orientation foundation milestone 7`. Foundation 8 is
intentionally left uncommitted after qualification. History was not rewritten.

## One next dependency-complete milestone

The regenerated 197-type missing inventory and public-signature dependency
graph select exactly this independent next managed closure:

```text
Microsoft.Xna.Framework.Graphics.ClearOptions
```

`ClearOptions` is a dependency-complete Graphics enum leaf in the regenerated
graph. A future milestone must independently inspect its pinned metadata,
values, flags behavior, and mapping before implementation. Selection does not
authorize another GraphicsDevice overload or any buffer/draw expansion.

This selection deliberately does not choose VertexBuffer, IndexBuffer, dynamic
buffers, VertexDeclaration, SetDataOptions, another BufferUsage consumer,
GraphicsDeviceManager work, GamePad, Mouse, Touch, Content, Effects, or a
combined enum batch.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_8_COMPLETE=true
```
