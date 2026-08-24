# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 6 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestones 2 through 5 completed the
geometry/transform, Curve, PackedVector, and VertexElement managed closures.
Milestone 6 completes exactly:

```text
Microsoft.Xna.Framework.PlayerIndex
Microsoft.Xna.Framework.Input.Keyboard.GetState(PlayerIndex)
```

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 6 projection and behavior

`PlayerIndex` is in the root Framework package. It is a non-flags named
`int32` enum with explicit constants `One=0`, `Two=1`, `Three=2`, and `Four=3`.
The implementation uses no `iota`, validator, count constant, alias, duplicate
Input type, or native metadata. Raw values such as `PlayerIndex(12345)` remain
representable.

The Input package safely imports the root package; the root does not import
Input. The final mapped Keyboard member is:

```go
func KeyboardGetStateByPlayerIndex(
    playerIndex framework.PlayerIndex,
) (KeyboardState, error)
```

Both overloads call one private process keyboard-state helper. Direct IL from
the hash-matched XNA 4.0 Windows assembly proves that
`GetState(PlayerIndex)` contains no argument-load instruction and never reads,
validates, or routes on `playerIndex`. The no-argument XNA overload calls it
with raw `255`. CNA-Go therefore preserves the actual process-global semantics
through the already-qualified `cna_keyboard_get_state` route.

All four declared player values and raw `12345` produce value-equal snapshots
under the deterministic HEADLESS fixture. Both overloads retain the same
active-Game/callback/owner-thread requirement and the same before-Run and
after-shutdown failure path. Exact evidence is in
`docs/player-index-keyboard-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=58
TARGET_MEMBERS=1322
TOTAL_DIAGNOSTICS=378
MISSING_TYPE=199
MISSING_MEMBER=179
COMPLETE_TYPES=53
PARTIAL_TYPES=5
MISSING_TYPES=199

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. The dedicated closure measurement is:

```text
PlayerIndex source=5 expected-go=4 target-go=4 kind=named-int32-enum local=0
Keyboard    source=2 expected-go=2 target-go=2 kind=static-class     local=0
```

Keyboard has moved from partial with one missing member to complete. The exact
remaining partial types are:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (42)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count is 179.

## Behavioral and verifier evidence

The managed corpus is `PURE_XNA_DERIVED` with 234 observations, 234
assertions, and zero failures. Foundation 6 contributes:

```text
PLAYER_INDEX=2
KEYBOARD_PLAYER_INDEX=5
```

The verifier has 58 mutation cases. The eleven Foundation 6 cases cover wrong
PlayerIndex kind and underlying type, accidental flags, wrong endpoint values,
missing `Four`, missing Keyboard overload, wrong parameter type, wrong return,
missing error, and wrong overload name. `ALLOWLIST_ENTRIES=0` and
`UNMEASURED_STRUCTURAL_CATEGORY=0`.

All KeyboardState regression tests cover `Keys`, `KeyState`, construction,
queries/indexer, pressed-key ordering, equality, hash, operators, and both
overloads. Pure tests pass with `CNA_NATIVE_LIBRARY` unset.

## ABI, native, and template regression

Milestone 6 adds no native function, cgo function, manifest entry, layout,
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

Ordinary and Go-race native stress retain 20 Game, recreation, texture,
SpriteBatch, callback-error, and callback-panic cycles, at least 80 forced-GC
points, and zero crash/UAF/double-free observations. `GO_RACE_STATUS=PASS`;
native sanitizers remain `NOT_RUN`.

The maintained sibling `cna-go-template` source remains unchanged at commit
`65254848d9fac02ace934db3879106834bafca97`. Its test, vet, race, build,
trimpath, exact 60-Draw, and exact 600-Draw gates pass against the admitted
library. No multiplayer-keyboard example or new platform/hardware capability
was added.

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
go run ./tools/api_compat --mode strict       # expected 378 missing diagnostics
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
milestone handoff records the archive filename, SHA-256, entry count, two-pass
determinism, audit counters, and isolated 60/600 results so the hash is not
recursively embedded in the archive it describes.

## Worktree provenance

The actual checkout at the start of Foundation 6 differed from the supplied
handoff: it was clean at `2d572ef`, and `origin/develop` matched. Foundation 5
had already been committed as `2d572ef feat: complete vertex element
foundation milestone 5`; it was not uncommitted. Foundation 4 remains committed
as `15c26a2 feat: complete packed vector foundation milestone 4`. Foundation 6
is intentionally left uncommitted.

## One next dependency-complete milestone

The regenerated missing inventory selects exactly this next narrow closure:

```text
Microsoft.Xna.Framework.DisplayOrientation
Microsoft.Xna.Framework.GraphicsDeviceManager.SupportedOrientations
```

`DisplayOrientation` is a root, flags, `Int32` enum and is the sole missing
public-signature dependency of the existing GraphicsDeviceManager getter and
setter. Reference code shows that the property stores its value and marks the
manager dirty; it does not require GamePad or another input family. Recompute
the exact closure, direct IL behavior, defaults, lifecycle consequences, and
native-boundary needs before implementation.

Do not mix that milestone with GamePad merely because PlayerIndex now exists,
or with Mouse, Touch, vertex declarations/buffers, another GraphicsDeviceManager
property, Content, Effects, or another family. No Foundation 7 code has been
started here.

```text
FOUNDATION_MILESTONE_6_COMPLETE=true
```
