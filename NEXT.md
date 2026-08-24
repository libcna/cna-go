# CNA-Go resumable handoff

## Current state

Foundation Milestone 1 has turned the initial honest scaffold into a measured,
real CNA C-ABI foundation. The baseline CNA-Go HEAD was
`1adec27d421745fb176e6cd2cf5f849b3226f891`; the template baseline was
`4d836f5de7efbab16b3f509cbd0f7bc398d67c7c`. Work remains uncommitted unless a
later handoff explicitly records a commit.

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only cgo/C/symbol/handle/callback/ownership/thread
  boundary;
- public XNA packages contain no C types or raw/native handles;
- pure XNA values remain Go values;
- native resources route only through canonical CNA C ABI 0.7;
- missing API stays absent and structurally measured—never fake or no-op.

## Evidence scoreboard

The authoritative retained XNA contract is
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`, with
257 types and 2,964 members. The formal Go projection derives 257 types and
3,243 members.

The current generated report measures:

```text
TARGET_TYPES=18
TARGET_MEMBERS=310
TOTAL_DIAGNOSTICS=648
MISSING_TYPE=239
MISSING_MEMBER=409
COMPLETE_TYPES=9
PARTIAL_TYPES=9
MISSING_TYPES=239
```

Every mismatch, unexpected-symbol, public-native-leak, allowlist, and
unmeasured-category counter is zero. Strict mode is intentionally red for the
648 genuine missing diagnostics; leak-only is green. The nine complete types
are `GameTime`, `KeyState`, `KeyboardState`, `Keys`, `MathHelper`, `Point`,
`Rectangle`, `SpriteEffects`, and `SpriteSortMode`. Exact partial-member lists
are generated in `docs/generated/missing-type-inventory.md`.

The managed behavior corpus currently has 27 observations, 27 assertions, and
zero failures. It covers exact float bits and edge classes plus the completed
MathHelper/Point/Rectangle/GameTime closure.

## Native admission and runtime

The qualification library was built in an isolated temporary source tree from
CNA revision `a09196a6477f69a7a57c8364f990658d31531a5b` (the selected ABI-0.7
source), sharp-runtime revision
`54578590b328aa9612fe38bfddca9fd8ca795144`, and the CNA tree's exact SDL,
SDL_image, SDL_mixer, and Draco submodule revisions. CNA itself was not edited.
The artifact is HEADLESS/NULL-audio Linux amd64, reports ABI 0.7.0, has SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`,
and is not distributed or sanitizer-instrumented.

The compiler-backed gate measures 23 bound functions, 67 prototype positions,
96 aggregate C/Go measurements, 28 layouts, two callbacks, and five constants,
with zero missing header symbols, missing library symbols, or ABI mismatches.

`Game.Run` locks its OS thread, admits the library/version/symbol table, creates
a `runtime/cgo.Handle`, lets CNA drive the lifecycle, contains callback errors
and panics, performs callback-safe two-phase resource destruction, invalidates
the generation, then deletes the handle. Native stress passes 20 Game and
recreation cycles, 20 texture and SpriteBatch cycles, 20 callback-error cycles,
20 callback-panic cycles, 80 forced-GC points, and zero observed crashes, UAF,
or double-free. The same suite passes under Go's race detector. Native
sanitizers were not run, so do not claim native leak freedom.

The maintained `cna-go-template` is now an honest desktop entry point with no
Android/Wasm, fake ContentManager, BasicEffect/cube, capability guessing, or
renderer-name fiction. It loads the real 128x128 `Content/logo.png`, reads the
native viewport, clears, polls keyboard, and draws native-driven movement,
rotation, and scale. Exact 60- and 600-Draw modes pass on the qualified HEADLESS
artifact. Visible rendering remains backend-blocked and must not be claimed.

## Re-running gates

Use a Go 1.22-or-newer toolchain with Linux cgo and GCC. Set
`CNA_NATIVE_LIBRARY` to an absolute exact ABI-0.7 library path for native gates.

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict       # expected nonzero
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

In the template, use its `go.work` only for development. Run `go test ./...`,
`go build ./...`, and `go run ./cmd/desktop --frames 60|600`. Isolated consumer
qualification must instead point `replace github.com/openeggbert/cna-go => ...`
only at an extracted audited source archive, never at this checkout.

## One next dependency-complete milestone

Select the pure math closure:

```text
Vector2 -> Vector3 -> Vector4 -> Quaternion -> Matrix
```

Complete every mapped member of those five types together, extend the
PURE_XNA_DERIVED behavior corpus with operation-order and float32 bit cases,
then finish the Vector3/Vector4-dependent Color members and its exact predefined
color set. Do not start Effect/3D runtime APIs merely because Matrix becomes
available. This one closure completes the existing partial Vector2, unlocks
Color conversion work and future graphics values, and avoids the deferred
Content, audio, media, storage, touch, design, and packed-vector families.

Re-run the generated scoreboard before implementation; if authoritative
dependencies changed, update this closure explicitly rather than broadening it
ad hoc.
