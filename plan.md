# CNA-Go normative implementation plan

**Selected profile:** Microsoft XNA Framework 4.0 Windows runtime

**Native contract:** canonical CNA C ABI, exact experimental version 0.7.0

**Foundation 1 status:** qualified for the explicitly documented Linux amd64
HEADLESS closure; strict structural verification remains intentionally red for
genuine missing surface.

**Foundation 2 status:** qualified for the managed geometry/transform closure,
including its recursive Plane/Ray/bounds dependencies, then Color and Viewport.

This file is normative. `NEXT.md` is the resumable handoff; generated reports
are evidence and must not replace the rules here.

## Non-negotiable architecture

1. XNA namespaces map to case-preserving Go import paths below
   `Microsoft/Xna/Framework`; consumers choose import aliases.
2. `internal/interop` is the only C/cgo, symbol, handle, result, callback,
   string, ownership, version, and native-thread boundary.
3. Public XNA packages never expose C types, native handles, or implementation
   helpers. No public package is invented for convenience.
4. Pure XNA values remain Go values. Native resources use the CNA C ABI only;
   CNA's C++ ABI and other bindings are forbidden dependencies.
5. Missing members are absent and measured. No variadic `any` overload sink,
   synthetic native object, default-success result, no-op, or guessed
   capability may conceal incompleteness.

## Foundation 1 completed architecture

- [x] Audit every exported baseline symbol and decide keep/change/remove.
- [x] Pin the 257-type/2,964-member authoritative reference snapshot and hash.
- [x] Formalize namespace, type-kind, field, property, constructor, overload,
      static, operator, enum/flags, exception/error, ref/out, generic collision,
      nested-type, event/delegate, TimeSpan, Stream, IntPtr, and Game mappings.
- [x] Derive 257 expected Go XNA types and 3,243 mapped XNA members.
- [x] Extract the actual exported Go surface with parser/AST/types evidence.
- [x] Measure every requested structural/mapping/leak category.
- [x] Prove verifier negatives with mutation fixtures; leak-only is green.
- [x] Establish exact-version dynamic loading and a single typed C manifest.
- [x] Confine cgo to `internal/interop` and keep C pointers out of Go APIs.
- [x] Establish locked-owner-OS-thread, generation, ownership, and
      `runtime/cgo.Handle` callback architecture.
- [x] Contain returned callback errors and panics at the C boundary.
- [x] Complete the first locally closed managed set: MathHelper, Point,
      Rectangle, GameTime, Keys, KeyboardState, KeyState, SpriteEffects, and
      SpriteSortMode.
- [x] Add a XNA-derived managed behavior corpus and exact float32 edge checks.
- [x] Implement real Game, GraphicsDeviceManager/device/viewport/clear,
      encoded PNG Texture2D, selected SpriteBatch draw, and Keyboard routes.
- [x] Rewrite the maintained template as a Linux desktop 2D canary; remove
      Content/XNB, effect/3D, Android, Web/Wasm, and fake capabilities.

## Foundation 1 qualification evidence

- [x] Admit and hash an exact ABI 0.7 Linux amd64 CNA artifact.
- [x] Make the compiler-backed ABI report green with zero missing symbols and
      zero mismatches.
- [x] Qualify native Game lifecycle, tick data, exit, recreation, PNG info,
      viewport, clear, SpriteBatch, and keyboard with real runs.
- [x] Run maintained template at exact 60 and 600 native Draw callbacks.
- [x] Run crash-isolated 20-cycle ownership/callback/GC stress.
- [x] Run gofmt, vet, test, race, build, trimpath, behavior, strict-red,
      leak-only-green, native ABI, capabilities, and both-repository diff gates.
- [x] Create and audit a deterministic exact worktree source artifact.
- [x] Extract it into a fresh directory and qualify a fresh consumer with no
      development-checkout dependency at build/test and 60/600 frames.
- [x] Finalize evidence docs, capabilities, README, and `NEXT.md`.

The strict verifier is red because Foundation 1 is not the full binding. That
expected incompleteness does not negate the qualified foundation result:

```text
FOUNDATION_MILESTONE_1_COMPLETE=true
```

This is not a visible-rendering or broad-platform claim. The admitted artifact
is HEADLESS/NULL-audio; visible rendering is `BACKEND_BLOCKED`, native
sanitizers are `NOT_RUN`, and all unimplemented families remain explicit in the
capability inventory.

## Structural policy

The pinned reference contract is
`tools/api_compat/reference/xna40-windows-runtime-contract.json`; its admitted
SHA-256 is fixed in the verifier test and provenance README. Mapping rules are
machine-readable in `tools/api_compat/mapping-rules.json` and explained in
`docs/xna-go-mapping.md`.

Normal strict verification must remain nonzero until the entire mapped profile
exists. Foundation work instead requires that diagnostics describe real missing
surface only: no mismatch, unexpected, leak, allowlist, or unmeasured category.
Pure-value types selected as complete must have zero local diagnostics. Runtime
types may remain explicitly partial where completion requires deferred systems.

## Native and lifetime policy

Linux loads `CNA_NATIVE_LIBRARY` when it names an absolute regular file, or lets
the platform search `libcna_c_api.so`. Production contains no checkout-relative
fallback. ABI versions other than exactly 0.7.0 are rejected. The compiler
compares every manifest prototype and selected layout/constant/callback
measurement against canonical CNA headers; the production loader separately
checks the artifact exports.

The Run goroutine locks its OS thread before native creation and unlocks only
after callbacks are dead, children and Game have been destroyed, the generation
is invalid, and the `cgo.Handle` is deleted. Wrong-thread destroy preserves the
handle. Finalizers are not correctness mechanisms and Foundation 1 uses none.

## Pure-value growth policy

Do not chase type count. Add one dependency-complete closure at a time from the
fresh missing inventory and finish every selected type. `float32` follows XNA
operation order; calls through Go's float64 math library require explicit
rounding boundaries and bit-level golden cases for signed zero, NaN, infinity,
subnormal, overflow, and underflow.

Foundation 2 completes `Vector2`, `Vector3`, `Vector4`, `Quaternion`, `Matrix`,
`Plane`, `Ray`, `BoundingBox`, `BoundingSphere`, `BoundingFrustum`,
`ContainmentType`, and `PlaneIntersectionType` as one public-signature closure.
It then completes the directly unblocked `Color` and `Graphics.Viewport`
partials. `BoundingFrustum` remains a class/pointer facade even though all of
its state and algorithms are managed. `Nullable<T>` inputs map to `*T` and
nullable returns/out values map to `(T, bool)` without a sentinel.

All fourteen selected types have zero local structural diagnostics. The
PURE_XNA_DERIVED corpus has 93 observations and no failures; exact geometry
evidence is in `docs/geometry-transform-evidence.md`.

## Foundation 2 qualification evidence

- [x] Regenerate the public-signature dependency graph from the pinned contract.
- [x] Complete all twelve primary geometry/dependency types with local strict zero.
- [x] Formalize and verifier-test nullable input, return, out, and error separation.
- [x] Preserve binary32 operation ordering and add exact-bit behavior fixtures.
- [x] Complete Color, including all 141 predefined static colors and an independent golden table.
- [x] Complete managed Viewport Project/Unproject without a CNA route.
- [x] Preserve every global mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go tests, vet, race, build, trimpath, native stress, and the unchanged maintained template.
- [x] Produce and independently regenerate an audited deterministic source artifact and qualify an isolated consumer from it.

The normal strict verifier remains red only for 228 genuinely missing types and
180 members on six explicitly deferred native/runtime partial types:

```text
FOUNDATION_MILESTONE_2_COMPLETE=true
```

## Deferred families

Content/XNB and LZX; effects, models, and 3D; audio/XACT; media/video; storage;
touch; design; packed vectors; GamerServices; and broad platform work are not
Foundation 1 count-filling opportunities. Windows, macOS, Android, iOS, and
Web/Wasm are unqualified even if the Go compiler can target them.

## Next milestone selection rule

After Foundation 2 qualification, the next scoreboard-selected managed closure
is exactly `Curve`, `CurveKeyCollection`, `CurveKey`, `CurveLoopType`,
`CurveTangent`, and `CurveContinuity`. `Curve` directly introduces its
collection and two enums; the collection introduces `CurveKey`; the key
introduces `CurveContinuity`. Recompute this choice from the then-current pinned
contract before implementation. Do not mix it with Content, Effects/3D,
PackedVector, Design, or native runtime cleanup.
