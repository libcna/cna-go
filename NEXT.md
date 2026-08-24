# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 and 2 are complete. Milestone 1 established the real
CNA C-ABI runtime architecture. Milestone 2 added no C functions and completes
the managed XNA geometry/transform closure plus the two existing partials it
directly unblocked.

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only cgo/C/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 2 closure

The pinned public signatures prove this exact recursive closure:

```text
Vector2
Vector3
Vector4
Quaternion
Matrix
Plane
Ray
BoundingBox
BoundingSphere
BoundingFrustum
ContainmentType
PlaneIntersectionType
```

The original five math types were insufficient: `Matrix` introduces `Plane`;
`Plane` introduces the three bounds/frustum types and
`PlaneIntersectionType`; the bounds introduce `Ray`, `ContainmentType`, and
their mutual references. After all twelve became locally strict-clean,
`Color` and `Graphics.Viewport` were completed. No unrelated family or native
route was added. Exact counts, direct dependencies, conventions, and the local
matrix are in `docs/geometry-transform-evidence.md`.

`System.Nullable<T>` is now formal: an input is `*T` with nil meaning null; a
return or out value is `(T, bool)` with the Boolean meaning `hasValue`. It is
distinct from source Boolean returns, ordinary `out T`, and a final Go `error`.
The verifier has a dedicated self-test.

## Structural scoreboard

The retained XNA contract is SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc` and
contains 257 types / 2,964 members. The formal Go projection remains 257 types /
3,243 members.

```text
TARGET_TYPES=29
TARGET_MEMBERS=1033
TOTAL_DIAGNOSTICS=408
MISSING_TYPE=228
MISSING_MEMBER=180
COMPLETE_TYPES=23
PARTIAL_TYPES=6
MISSING_TYPES=228
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. All twelve closure types plus Color and Viewport have zero
local diagnostics.

The exact remaining partial types are:

- `Microsoft.Xna.Framework.Game`
- `Microsoft.Xna.Framework.GraphicsDeviceManager`
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice`
- `Microsoft.Xna.Framework.Graphics.SpriteBatch`
- `Microsoft.Xna.Framework.Graphics.Texture2D`
- `Microsoft.Xna.Framework.Input.Keyboard`

Their 180 missing members are unchanged native/runtime work. In particular,
`Keyboard.GetState(PlayerIndex)` was not added.

## Behavioral evidence

The managed PURE_XNA_DERIVED corpus now has 93 observations, 93 assertions, and
zero failures. It covers all requested vector, quaternion, matrix, plane, ray,
bounds/frustum, Color, and Viewport groups. Focused package tests independently
cover the full 141-entry predefined Color table, array overlap/order and invalid
ranges, nullable intersections, class identity, corner ordering, GJK results,
binary32 edge behavior, hash, and string identities.

The Go 1.24.4 Linux amd64 cgo qualification passes gofmt, vet, test, race,
build, and trimpath. Pure managed tests do not require a native library.

## ABI and native regression

The admitted library remains the exact HEADLESS/NULL-audio ABI 0.7.0 artifact
with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`.
Milestone 2 leaves every ABI count unchanged:

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

Both ordinary and Go-race native stress pass 20 Game, recreation, texture,
SpriteBatch, callback-error, and callback-panic cycles, 80 forced-GC points,
and zero crash/UAF/double-free observations. Native sanitizers remain
`NOT_RUN`.

The maintained `cna-go-template` source was not changed. Its test, vet, race,
build, trimpath, exact 60-Draw, and exact 600-Draw gates pass against the same
admitted library. Visible rendering remains backend-blocked; no capability
claim changed.

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
go run ./tools/api_compat --mode strict       # expected 408 missing diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Source-artifact and isolated-consumer qualification use a deterministic archive
of the exact worktree, not the development checkout. The external milestone
handoff records that archive's filename, SHA-256, entry count, and two-pass
determinism result so the hash is not recursively embedded in the archive it
describes.

## One next dependency-complete milestone

The freshly regenerated missing inventory selects this managed curve closure:

```text
Curve
CurveKeyCollection
CurveKey
CurveLoopType
CurveTangent
CurveContinuity
```

`Curve` directly introduces `CurveKeyCollection`, `CurveLoopType`, and
`CurveTangent`; the collection introduces `CurveKey`; the key introduces
`CurveContinuity`. Recompute the closure from the pinned contract when work
begins. Do not combine it with native partial cleanup, Content/XNB, LZX,
Effect/3D, Model, Audio/XACT, Media/Video, Storage, Touch, Design,
PackedVector, or GamerServices.

```text
FOUNDATION_MILESTONE_2_COMPLETE=true
```
