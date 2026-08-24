# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 5 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestone 2 completed managed geometry,
Color, and Viewport. Milestone 3 completed Curve and the collection/iterator
projection. Milestone 4 completed the two PackedVector interfaces and all
seventeen concrete packed value structs. Milestone 5 adds no C function and
completes exactly this managed descriptor closure:

```text
Microsoft.Xna.Framework.Graphics.VertexElement
Microsoft.Xna.Framework.Graphics.VertexElementFormat
Microsoft.Xna.Framework.Graphics.VertexElementUsage
```

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 5 projection and behavior

The exact closure has three types, 37 XNA source identities, and 39 mapped Go
identities. `VertexElementFormat` and `VertexElementUsage` are non-flags named
`int32` types with explicit raw constants. No `iota`, validation, or public enum
string methods were added. Arbitrary CLR enum values remain representable.

`VertexElement` is a private-state Go value struct. Its one public constructor
stores the four input values without validation. Four writable CLR properties
expand to four value-received getters and four pointer-received setters. The
natural Go zero value is the descriptor `(0, Single, Position, 0)`, but there
is no invented parameterless XNA constructor. Struct copies have independent
mutable state.

The unique `Equals(Object)` identity maps to `Equals(any)`; no typed
`Equals(VertexElement)` member exists. Equality and both named operators
compare all four stored values. The exact XNA hash is the XOR of the four
sequential 32-bit words, with `Int32.MaxValue` substituted when the XOR is
zero. Exact string form is:

```text
{Offset:<offset> Format:<format> Usage:<usage> UsageIndex:<usageIndex>}
```

Known enums render their XNA names; undefined values render signed decimal.
Exact IL evidence, fixtures, enum tables, and the local strict-zero matrix are
in `docs/vertex-element-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=57
TARGET_MEMBERS=1317
TOTAL_DIAGNOSTICS=380
MISSING_TYPE=200
MISSING_MEMBER=180
COMPLETE_TYPES=51
PARTIAL_TYPES=6
MISSING_TYPES=200

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. All three Foundation 5 types have zero local diagnostics:

```text
VertexElement       source=10 expected-go=14 target-go=14 kind=struct
VertexElementFormat source=13 expected-go=12 target-go=12 kind=named-int32-enum
VertexElementUsage  source=14 expected-go=13 target-go=13 kind=named-int32-enum
```

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (42)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)
- `Microsoft.Xna.Framework.Input.Keyboard` (1)

Their combined 180 missing members are unchanged.

## Behavioral and verifier evidence

The managed corpus is `PURE_XNA_DERIVED` with 227 observations, 227
assertions, and zero failures. Foundation 5 contributes:

```text
VERTEX_ELEMENT_ENUMS=2
VERTEX_ELEMENT=24
```

The verifier has 47 mutation cases. Foundation 5 adds 15 cases spanning the
wrong struct kind, property-vs-field projection, missing/wrong setters,
constructor order and enum types, unexpected typed equality, both operators,
enum values, and non-flags status. Property expansion is compiler-measured;
`ALLOWLIST_ENTRIES=0` and `UNMEASURED_STRUCTURAL_CATEGORY=0`.

The PackedVector exhaustive report remains 262,400 iterations with zero
failures, and all 25 compiler-measured interface witnesses remain present.

## ABI, native, and template regression

The admitted native library remains the exact HEADLESS/NULL-audio ABI 0.7.0
artifact with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`.
Milestone 5 leaves every ABI count unchanged:

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
SpriteBatch, callback-error, and callback-panic cycles, 80 forced-GC points,
and zero crash/UAF/double-free observations. `GO_RACE_STATUS=PASS`; native
sanitizers remain `NOT_RUN`.

The maintained `cna-go-template` source is unchanged at commit
`65254848d9fac02ace934db3879106834bafca97`. Its test, vet, race, build,
trimpath, exact 60-Draw, and exact 600-Draw gates pass against the admitted
library. No renderer, GPU, vertex declaration/buffer, or draw capability was
claimed. The capability inventory records only the narrow managed descriptor
classification.

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
go run ./tools/api_compat --mode strict       # expected 380 missing diagnostics
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

## One next dependency-complete milestone

The freshly regenerated inventory and pinned public signatures select exactly:

```text
Microsoft.Xna.Framework.PlayerIndex
Microsoft.Xna.Framework.Input.Keyboard.GetState(PlayerIndex)
```

`PlayerIndex` is a standalone non-flags `Int32` enum (`One=0`, `Two=1`,
`Three=2`, `Four=3`). It is the sole missing public-signature dependency of the
sole remaining `Keyboard` member. Direct XNA IL confirms that the player-index
overload does not consume its argument; it reads the same process keyboard
state used by the existing no-argument path. This next milestone can therefore
complete `Keyboard` without adding a CNA ABI function. Recompute its exact
source/Go closure before implementation.

Do not mix that milestone with GamePad, `IVertexType`, `VertexDeclaration`,
vertex structs/buffers, native GraphicsDevice expansion, Design, Content/XNB,
Effects/3D, or another family. No next-milestone code has been started here.

```text
FOUNDATION_MILESTONE_5_COMPLETE=true
```
