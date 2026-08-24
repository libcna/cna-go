# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 7 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestones 2 through 6 completed the
geometry/transform, Curve, PackedVector, VertexElement, PlayerIndex, and
Keyboard closures. Milestone 7 completes exactly:

```text
Microsoft.Xna.Framework.DisplayOrientation
Microsoft.Xna.Framework.GraphicsDeviceManager.SupportedOrientations
```

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 7 projection and behavior

`DisplayOrientation` is in the root Framework package. It is a flags named
`int32` enum with explicit `Default=0`, `LandscapeLeft=1`,
`LandscapeRight=2`, and `Portrait=4` constants. Combinations and arbitrary raw
bits remain representable. There is no `iota`, validator, normalization, count
constant, or flags-helper API.

The exact selected property projection is:

```go
func (m *GraphicsDeviceManager) SupportedOrientations() DisplayOrientation
func (m *GraphicsDeviceManager) SetSupportedOrientations(value DisplayOrientation)
```

Direct IL from the hash-matched XNA 4.0 Windows assemblies proves that the
constructor leaves `supportedOrientations=0` and `isDeviceDirty=false` through
CLR zero initialization. The getter only loads the stored enum. The setter
always stores the exact value and always sets dirty true, including when the
value is unchanged. It performs no validation, branch, call, window/device
operation, `ApplyChanges`, or eager platform application.

CNA-Go retains these as private managed fields. Both accessors have no
synthetic error and no native call. The value remains accessible after native
manager disposal, matching its managed state character. Exact evidence is in
`docs/display-orientation-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=59
TARGET_MEMBERS=1328
TOTAL_DIAGNOSTICS=375
MISSING_TYPE=198
MISSING_MEMBER=177
COMPLETE_TYPES=54
PARTIAL_TYPES=5
MISSING_TYPES=198

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. The dedicated selected-closure measurement is:

```text
DisplayOrientation source=5 expected-go=4 target-go=4 local=0
SupportedOrientations source=1 expected-go=2 target-go=2 local=0
```

GraphicsDeviceManager remains partial. The exact remaining partial types are:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count is 177.

## Behavioral and verifier evidence

The managed corpus is `PURE_XNA_DERIVED` with 242 observations, 242
assertions, and zero failures. Foundation 7 contributes:

```text
DISPLAY_ORIENTATION=3
GRAPHICS_MANAGER_ORIENTATION=5
```

The verifier has 73 mutation cases. The 15 Foundation 7 additions cover enum
kind/underlying/flags/value/missing-literal failures; missing, read-only,
wrong-type, synthetic-error, static, and wrong-package property projections;
and a public dirty-helper leak. `ALLOWLIST_ENTRIES=0` and
`UNMEASURED_STRUCTURAL_CATEGORY=0`.

Package tests qualify initial private dirty state, same-value and changed-value
setters, multiple assignments, combinations, raw bits, and post-disposal
managed access. Native stress separately qualifies the real constructor
default and property persistence during Initialize, a later callback, and
after native disposal. Foundation 6 PlayerIndex and both Keyboard overloads
remain green.

## ABI, native, and template regression

Milestone 7 adds no native function, cgo function, manifest entry, layout,
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
trimpath, exact 60-Draw, and exact 600-Draw gates pass. No orientation example
or platform rotation claim was added.

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
go run ./tools/api_compat --mode strict       # expected 375 missing diagnostics
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

The actual checkout at the start of Foundation 7 was clean at `778b75b`, and
`origin/develop` matched. This differs from the supplied older handoff because
the user explicitly requested that Foundation 6 be committed and pushed in the
intervening turn. Foundation 6 is committed as `778b75b feat: complete player
index keyboard foundation milestone 6`. Foundation 7 is intentionally left
uncommitted. History was not rewritten.

## One next dependency-complete milestone

The regenerated missing inventory selects exactly this independent next
managed closure:

```text
Microsoft.Xna.Framework.Graphics.BufferUsage
```

`BufferUsage` is a two-literal root Graphics `[Flags]` `Int32` enum with no
missing public-signature dependency. A future milestone must independently
verify its pinned metadata and behavior before implementation. It must not add
VertexBuffer, IndexBuffer, VertexDeclaration, or another buffer family merely
because this enum is used by those deferred APIs.

This selection deliberately does not continue GraphicsDeviceManager and does
not start ApplyChanges, GameWindow, GamePad, Mouse, Touch, Content, Effects, or
another family. No Foundation 8 code has been started here.

```text
FOUNDATION_MILESTONE_7_COMPLETE=true
```
