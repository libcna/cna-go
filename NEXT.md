# CNA-Go resumable handoff

## Current state

Foundation Milestones 1, 2, and 3 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestone 2 completed managed geometry,
Color, and Viewport. Milestone 3 adds no C function and completes exactly the
managed Curve family:

```text
Curve
CurveKeyCollection
CurveKey
CurveLoopType
CurveTangent
CurveContinuity
```

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 3 semantics

`Curve`, `CurveKey`, and `CurveKeyCollection` are pointer facades over private
managed state. `CurveKey` has all three overload-derived constructor names,
read-only Position, mutable scalar/continuity properties, value equality,
additive XNA hash, both operators, independent Clone, and Position-only
`Single.CompareTo` ordering including NaN.

The collection stores key references privately and remains position-sorted.
Ordinary equal positions are inserted after existing equals. Item replacement
repositions when Position differs. Contains/IndexOf/Remove use key value
equality. CopyTo retains caller storage and key pointers. Collection and Curve
clones have new collection storage but share key objects.

The BCL collection boundary is formal: no `System.Collections` Go package is
created. `ICollection<T>` maps to the concrete method set;
`IEnumerable<T>`/`IEnumerator<T>` map through the measured `Iterator<T>`
adapter with `Next() (T, bool, error)`. Enumeration is source-ordered and
versioned/fail-fast. Managed argument/index failures return Go errors rather
than slice or nil panics.

Curve defaults, stable Keys identity, exact IsConstant, Flat/Linear/Smooth and
mixed tangents, double-intermediate segment selection, binary32 Hermite, Step,
all five loop modes, and negative cycle decrement/parity are qualified. Exact
semantics and the local matrix are in `docs/curve-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=35
TARGET_MEMBERS=1089
TOTAL_DIAGNOSTICS=402
MISSING_TYPE=222
MISSING_MEMBER=180
COMPLETE_TYPES=29
PARTIAL_TYPES=6
MISSING_TYPES=222
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. All six Curve types have zero local diagnostics at 13, 19, 14,
2, 5, and 3 target members respectively.

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game`
- `Microsoft.Xna.Framework.GraphicsDeviceManager`
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice`
- `Microsoft.Xna.Framework.Graphics.SpriteBatch`
- `Microsoft.Xna.Framework.Graphics.Texture2D`
- `Microsoft.Xna.Framework.Input.Keyboard`

Their 180 missing members are unchanged native/runtime work.
`Keyboard.GetState(PlayerIndex)` remains absent.

## Behavioral evidence

The managed `PURE_XNA_DERIVED` corpus now has 142 observations, 142 assertions,
and zero failures. Foundation 3 contributes 49 observations:

```text
CURVE_ENUMS=3
CURVE_KEY=10
CURVE_COLLECTION=12
CURVE_TANGENTS=7
CURVE_EVALUATE=8
CURVE_LOOPS=9
```

Focused package tests additionally cover negative/end indexes, all CopyTo
ranges, stable/fresh/fail-fast cursors, pointer identity, shallow clone depth,
NaN comparison/equality/hash distinctions, exact key boundaries, mixed tangent
modes, repeated tangent passes, equal-value IsConstant, and positive/negative
loop multiples.

## ABI and native regression

The admitted native library remains the exact HEADLESS/NULL-audio ABI 0.7.0
artifact with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`.
Milestone 3 leaves every ABI count unchanged:

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
points, and zero crash/UAF/double-free observations. Native sanitizers remain
`NOT_RUN`.

The maintained `cna-go-template` source is unchanged. Its test, vet, race,
build, trimpath, exact 60-Draw, and exact 600-Draw gates pass against the same
admitted library. Visible rendering remains backend-blocked; no native runtime
capability claim changed.

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
go run ./tools/api_compat --mode strict       # expected 402 missing diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Source-artifact and isolated-consumer qualification use a deterministic archive
of the exact worktree, not the development checkout. The external milestone
handoff records the new archive filename, SHA-256, entry count, and two-pass
determinism result so the hash is not recursively embedded in the archive it
describes.

## One next dependency-complete milestone

The freshly regenerated missing inventory selects exactly the managed
`Microsoft.Xna.Framework.Graphics.PackedVector` closure: both packed-vector
interfaces and all seventeen concrete packed value types. Recompute its public
signature closure from the pinned contract before implementation. Do not mix it
with Design, Content/XNB, Effects/3D, native partial cleanup, or another family.

```text
FOUNDATION_MILESTONE_3_COMPLETE=true
```
