# CNA-Go normative implementation plan

**Selected profile:** Microsoft XNA Framework 4.0 Windows runtime

**Native contract:** canonical CNA C ABI, admitted range major 0 with minor 21
or newer, qualified at 0.21.0 (Foundation 44 migrated this from Foundation 1's
exact 0.7.0 admission)

**Foundation 1 status:** qualified for the explicitly documented Linux amd64
HEADLESS closure; strict structural verification remains intentionally red for
genuine missing surface.

**Foundation 2 status:** qualified for the managed geometry/transform closure,
including its recursive Plane/Ray/bounds dependencies, then Color and Viewport.

**Foundation 3 status:** qualified for the complete managed Curve family,
including reference-class identity, the formal BCL collection projection, and
binary32 tangent/evaluation/loop behavior.

**Foundation 4 status:** qualified for the complete managed PackedVector
family and exhaustive raw-pattern behavior.

**Foundation 5 status:** qualified for the complete managed VertexElement
descriptor closure.

**Foundation 6 status:** qualified for the root PlayerIndex enum and the final
Keyboard overload through the unchanged process keyboard route.

**Foundation 7 status:** qualified for the DisplayOrientation flags enum and
the managed GraphicsDeviceManager.SupportedOrientations property slice.

**Foundation 8 status:** qualified for exactly the pure managed Graphics
BufferUsage flags enum, with no buffer class, device member, or ABI expansion.

**Foundation 9 status:** qualified for exactly the pure managed Graphics
ClearOptions flags enum, including the no-declared-zero mapping edge case,
with no GraphicsDevice member, clear behavior, or ABI expansion.

**Foundation 10 status:** qualified for exactly the pure managed Graphics
SurfaceFormat non-flags enum contract, with no texture/display/adapter/
presentation consumer, runtime pixel-format claim, or ABI expansion.

**Foundation 11 status:** qualified for exactly the pure managed Graphics
DepthFormat non-flags enum contract, with no adapter/presentation/render-target/
manager consumer, depth-stencil runtime claim, or ABI expansion.

**Foundation 12 status:** qualified for exactly the pure managed Graphics
GraphicsProfile non-flags enum contract, with no adapter/device/manager/
device-information consumer, Reach/HiDef hardware or feature-level claim, or
ABI expansion.

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

- [x] Admit and hash a Linux amd64 CNA artifact. Foundation 1 admitted an exact
      ABI 0.7 artifact; Foundation 44 migrated the boundary to the live CNA C
      ABI and now admits a stated range.
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
fallback. An ABI outside the admitted range -- major 0 with minor 21 or newer --
is rejected, and the rejection names the library path, the reported version and
the range. The compiler compares every manifest prototype against the canonical
declaration of the same name, and compares CNA-Go's private layouts against the
canonical ones by measuring both in separate translation units; the production
loader separately checks the artifact exports and confirms with `dladdr` that
each resolved address belongs to the symbol the manifest names.

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

## Foundation 3 managed Curve policy

Foundation 3 completes exactly `Curve`, `CurveKeyCollection`, `CurveKey`,
`CurveLoopType`, `CurveTangent`, and `CurveContinuity`. The three classes are
pointer facades over private managed state; the enums are explicit named
`int32` values. Curve keys remain references at every collection and clone
boundary required by XNA.

Generic BCL collection interfaces do not introduce fake public BCL packages.
`ICollection<T>` is measured as the concrete XNA collection method set.
`IEnumerable<T>` and `IEnumerator<T>` use the separately measured generic
`Iterator<T>` language adapter, whose `Next` result separates value presence
from mutation failure. Managed argument/index failures use Go `error`, never a
slice-bounds panic.

The reference algorithms retain sorted duplicate insertion, shallow collection
clone depth, binary32 tangent operation order, the double segment-position
intermediate, Hermite/Step interpolation, and the negative-cycle decrement
rule. Exact evidence is in `docs/curve-evidence.md`.

## Foundation 3 qualification evidence

- [x] Regenerate the exact six-type dependency closure and 56-member Go projection.
- [x] Complete all six types with local strict zero and no new partial type.
- [x] Formalize and mutation-test ICollection/IEnumerable/IEnumerator projection.
- [x] Qualify key constructors, identity, clone, equality, hash, operators, and NaN ordering.
- [x] Qualify sorted insertion, duplicate keys, indexer repositioning, CopyTo, shallow clone, and fail-fast iteration.
- [x] Qualify Curve defaults, Keys identity, IsConstant, and shallow-key Curve clone.
- [x] Qualify Flat/Linear/Smooth/mixed tangents and repeated whole-curve computation.
- [x] Qualify empty/single evaluation, segment boundaries, Step, Hermite, all loops, and negative cycles.
- [x] Grow the PURE_XNA_DERIVED corpus from 93 to 142 assertions with zero failures.
- [x] Preserve every global mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go tests, vet, race, build, trimpath, native stress, and the unchanged maintained template.
- [x] Produce an audited deterministic source artifact and qualify an isolated consumer from it.

The normal strict verifier remains red only for 222 genuinely missing types and
180 members on the same six deferred native/runtime partial types:

```text
FOUNDATION_MILESTONE_3_COMPLETE=true
```

## Foundation 4 managed PackedVector policy

Foundation 4 completes exactly the two PackedVector interfaces and all
seventeen concrete packed value structs. `IPackedVector<TPacked>` retains the
deterministic generic-collision name `IPackedVectorOfTPacked[TPacked]`, and the
general CLR owner-parameter mapping substitutes `!0` with `TPacked` instead of
leaking a metadata token into Go.

Pure managed interface methods gain no synthetic `error`. Mutable packed
structs remain Go values, while pointer method sets satisfy the exact generic
interface and its transitive base. Explicit CLR implementations needed by Go
structural interfaces are measured as 25 language witness projections, not as
ordinary XNA members or allowlist entries.

Each format stores one private fixed-width packed integer. Packing preserves
XNA binary32 scale/clamp/round order, ties-to-even, explicit non-finite
handling, exact lane masks, SNorm minimum decoding, raw integer behavior, and
XNA's non-IEEE exponent-31 half semantics. Exact mapping and behavior evidence
is in `docs/packed-vector-evidence.md`.

## Foundation 4 qualification evidence

- [x] Regenerate the exact 19-type, 171-source-member, 189-Go-member closure.
- [x] Generalize and mutation-test `!n` owner generic-parameter substitution.
- [x] Formalize pure managed interface no-error classification and generic inheritance.
- [x] Prove all seventeen exact `*T` generic/base interface conformances with `go/types`.
- [x] Measure 17 `PackFromVector4` and eight `ToVector4` witnesses separately.
- [x] Complete all seventeen canonical packed-bit structs with local strict zero.
- [x] Qualify every constructor, converter, direct setter, interface lane route, equality, hash, operator, and string identity.
- [x] Qualify UNorm, SNorm, raw integer, half, midpoint-adjacent, and non-finite behavior.
- [x] Exhaust all 256 Alpha8 and all 65,536 Bgr565, Bgra4444, Bgra5551, and HalfSingle bit patterns with zero failures.
- [x] Grow the PURE_XNA_DERIVED corpus from 142 to 201 assertions with zero failures.
- [x] Preserve every global mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go tests, vet, race, build, trimpath, native stress, and the unchanged maintained template.
- [x] Produce an audited deterministic source artifact and qualify an isolated consumer from it.

The normal strict verifier remains red only for 203 genuinely missing types and
180 members on the same six deferred native/runtime partial types:

```text
FOUNDATION_MILESTONE_4_COMPLETE=true
```

## Foundation 5 managed VertexElement policy

Foundation 5 completes exactly `Graphics.VertexElement`,
`VertexElementFormat`, and `VertexElementUsage`. The closure has 37 source
identities and 39 mapped Go identities. Both enums are explicit non-flags
named `int32` values; arbitrary underlying values remain representable and are
never constructor/setter validation failures.

`VertexElement` is a private-state Go value struct. Its natural zero value is
the Single/Position descriptor at offset and usage index zero, but no fake
parameterless XNA constructor is added. Four writable source properties expand
to four value-received getters and four pointer-received setters. Copies have
independent property state.

The unique object equality method maps to `Equals(any)`; there is no typed
`Equals(VertexElement)` source identity or Go alias. Both mapped operators
compare all four fields. `GetHashCode` reproduces XNA `SmartGetHashCode`: XOR
the four sequential 32-bit words and return `Int32.MaxValue` when the XOR is
zero. `ToString` preserves exact labels/order/punctuation, declared enum names,
and signed-decimal fallback for undefined non-flags enum values. Exact IL,
fixtures, and the local matrix are in `docs/vertex-element-evidence.md`.

## Foundation 5 qualification evidence

- [x] Regenerate the exact three-type, 37-source-member/39-Go-member closure.
- [x] Complete all three types with compiler-measured local strict zero.
- [x] Preserve all 25 PackedVector interface witnesses and exhaustive evidence.
- [x] Mutation-test property projection, constructor signatures, operators,
      enum values/flags, and the absence of typed `Equals(VertexElement)`.
- [x] Grow the PURE_XNA_DERIVED corpus from 201 to 227 assertions with zero failures.
- [x] Preserve every global mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go tests, vet, race, build, trimpath, native stress, and the unchanged maintained template.
- [x] Produce an audited deterministic source artifact and qualify an isolated consumer from it.

The normal strict verifier remains red only for 200 genuinely missing types and
180 members on the same six deferred native/runtime partial types. This
milestone adds no `IVertexType`, `VertexDeclaration`, vertex struct/buffer,
draw/GPU route, or CNA ABI function.

```text
FOUNDATION_MILESTONE_5_COMPLETE=true
```

## Foundation 6 PlayerIndex and Keyboard policy

Foundation 6 completes exactly `Microsoft.Xna.Framework.PlayerIndex` and the
sole missing `Input.Keyboard.GetState(PlayerIndex)` member. `PlayerIndex` is a
root-Framework, non-flags named `int32` enum with explicit `One=0`, `Two=1`,
`Three=2`, and `Four=3` constants. Arbitrary underlying values remain
representable and are not validated.

Direct IL from the hash-matched XNA 4.0 Windows assembly proves that the
player-index overload never loads or otherwise consumes its argument. Both Go
overloads therefore call the same private process keyboard-state helper and
retain the same active-Game, callback, owner-thread, native error, and value
snapshot semantics. No player-aware CNA route exists or was added.

## Foundation 6 qualification evidence

- [x] Independently verify the two-type, seven-source-identity/six-Go-identity closure.
- [x] Complete PlayerIndex and Keyboard with compiler-measured local strict zero.
- [x] Preserve undefined `int32` enum values and reject range-validation regressions.
- [x] Prove from direct IL that `GetState(PlayerIndex)` does not read its argument.
- [x] Prove identical native HEADLESS snapshots for all four values and raw `12345`.
- [x] Prove the same requirement before Run, during callbacks, and after shutdown.
- [x] Grow the PURE_XNA_DERIVED corpus from 227 to 234 assertions with zero failures.
- [x] Grow verifier mutations from 47 to 58 focused cases.
- [x] Preserve every mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go, native stress, unchanged template, artifact, and isolated-consumer gates.

Keyboard moves from partial to complete. Normal strict verification remains red
only for 199 genuinely missing types and 179 members on five explicitly
deferred native/runtime partial types. Exact evidence is in
`docs/player-index-keyboard-evidence.md`.

```text
FOUNDATION_MILESTONE_6_COMPLETE=true
```

## Foundation 7 DisplayOrientation and SupportedOrientations policy

Foundation 7 completes exactly `Microsoft.Xna.Framework.DisplayOrientation`
and the getter/setter projection of
`GraphicsDeviceManager.SupportedOrientations`. `DisplayOrientation` is a root
Framework `[Flags]` named `int32` enum with explicit `Default=0`,
`LandscapeLeft=1`, `LandscapeRight=2`, and `Portrait=4` values. Named
combinations and arbitrary raw bits remain representable without validation.

Hash-matched XNA Windows IL proves that the manager constructor leaves both
`supportedOrientations` and `isDeviceDirty` at CLR zero initialization. The
getter only loads the orientation field. The setter always stores the exact
value and always sets dirty true, including for a same-value assignment; it
contains no comparison, validation, call, eager window/device application, or
throw path. CNA-Go preserves those two private managed fields independently of
native resource lifetime and adds no synthetic accessor error.

## Foundation 7 qualification evidence

- [x] Complete the exact six-source-identity/six-Go-identity closure.
- [x] Complete DisplayOrientation with local strict zero and general flags policy.
- [x] Measure the selected GDM getter/setter slice at zero local diagnostics.
- [x] Preserve GDM as partial while reducing its missing members from 42 to 40.
- [x] Qualify default, exact-bit storage, combinations, unknown bits, same-value
      dirty behavior, changed-value dirty behavior, multiple setters, and
      post-disposal managed access.
- [x] Grow the PURE_XNA_DERIVED corpus from 234 to 242 assertions with zero failures.
- [x] Grow verifier mutations from 58 to 73 focused cases.
- [x] Preserve every mismatch, leak, allowlist, and unmeasured counter at zero.
- [x] Keep all CNA ABI measurements and symbols unchanged.
- [x] Requalify Go, native stress, unchanged template, artifact, and isolated-consumer gates.

GraphicsDeviceManager remains honestly partial with 40 missing members. No
other property, `ApplyChanges`, GameWindow orientation behavior, or platform
rotation claim is included. Exact IL and lifecycle evidence is in
`docs/display-orientation-evidence.md`.

```text
FOUNDATION_MILESTONE_7_COMPLETE=true
```

## Foundation 8 BufferUsage policy

Foundation 8 completes exactly
`Microsoft.Xna.Framework.Graphics.BufferUsage`. It is a `[Flags]` named
`int32` enum with explicit `None=0` and `WriteOnly=1` constants. The synthetic
CLR `value__` field is excluded, so three source identities project to exactly
two Go identities. The complete signed `int32` domain remains representable;
there is no validation, normalization, `String`, flags helper, constructor, or
other convenience surface.

The type is pure managed metadata in the root Graphics package. It adds no CNA
constant or operation and does not implement VertexBuffer, DynamicVertexBuffer,
IndexBuffer, DynamicIndexBuffer, VertexDeclaration, IVertexType, SetData,
GetData, binding, creation, or draw support. The five existing native/runtime
partial types are untouched.

## Foundation 8 qualification evidence

- [x] Independently confirm the exact three-source-identity/two-Go-identity contract.
- [x] Complete BufferUsage with compiler-measured local strict zero.
- [x] Preserve explicit values, exact flags metadata, zero value, arbitrary raw bits, and bitwise composition.
- [x] Grow verifier mutations from 73 to 85 focused cases without an allowlist or special case.
- [x] Grow the behavior corpus from 242 to 247 assertions with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 197 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/buffer-usage-evidence.md`.

```text
FOUNDATION_MILESTONE_8_COMPLETE=true
```

## Foundation 9 ClearOptions policy

Foundation 9 completes exactly
`Microsoft.Xna.Framework.Graphics.ClearOptions`. It is a `[Flags]` named
`int32` enum with explicit `Target=1`, `DepthBuffer=2`, and `Stencil=4`
constants. The synthetic CLR `value__` field is excluded, so four source
identities project to exactly three Go identities.

Unlike previously selected flags enums, the pinned metadata declares no
zero-valued literal. Raw zero remains the unnamed natural Go zero value; the
binding does not synthesize `None`, `Default`, `All`, or any other constant.
The complete signed `int32` domain remains representable without validation or
masking, and ordinary Go bitwise operations preserve declared and unknown bits.

The type is pure managed metadata in the root Graphics package. It does not
implement either `GraphicsDevice.Clear(ClearOptions, ...)` overload, any other
GraphicsDevice member, render-target/depth/stencil clearing, renderer state,
or a CNA/native route. The five native/runtime partial types are untouched.

## Foundation 9 qualification evidence

- [x] Independently confirm the exact four-source-identity/three-Go-identity contract.
- [x] Complete ClearOptions with compiler-measured local strict zero.
- [x] Preserve the unnamed zero value, explicit 1/2/4 literals, combinations,
      arbitrary signed raw bits, and typed OR/AND without helper surface.
- [x] Harden the general flags rule and reject invented named zero/all constants.
- [x] Grow verifier mutations from 85 to 99 without an allowlist or special case.
- [x] Grow the behavior corpus from 247 to 256 assertions with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 196 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/clear-options-evidence.md`.

```text
FOUNDATION_MILESTONE_9_COMPLETE=true
```

## Foundation 10 SurfaceFormat policy

Foundation 10 completes exactly
`Microsoft.Xna.Framework.Graphics.SurfaceFormat`. It is a non-flags named
`int32` enum with 20 explicit named constants from `Color=0` through
`HdrBlendable=19`. The synthetic CLR `value__` field is excluded. The milestone
request's stated 20-source/19-Go arithmetic is corrected by the pinned contract
and its own 20-literal table: the exact closure contains 21 source field
identities and 20 mapped Go identities.

All raw assignments are explicit and production contains no `iota`. The zero
value is `Color`; arbitrary signed `int32` values remain representable without
validation. There is no flags marker, Stringer, parser, format-size/helper API,
PackedVector dependency, native enum mirror, or GPU format mapping.

Completion of the enum contract does not claim runtime support for DXT, HDR,
half/floating-point, render-target, upload, or any other listed format. All
eleven reverse dependents remain deferred, including display/adapter/
presentation types, render targets, the texture family, the Texture2D format
constructor, and GraphicsDeviceManager format properties.

## Foundation 10 qualification evidence

- [x] Independently confirm the exact 21-source-identity/20-Go-identity contract.
- [x] Complete SurfaceFormat with compiler-measured local strict zero.
- [x] Retain every one of the 20 exact raw values and exclude `value__`.
- [x] Qualify Color zero value and arbitrary positive/negative raw values.
- [x] Grow verifier mutations from 99 to 117 without an allowlist or special mapping exception.
- [x] Grow the behavior corpus from 256 to 262 assertions with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 195 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/surface-format-evidence.md`.

```text
FOUNDATION_MILESTONE_10_COMPLETE=true
```

## Foundation 11 DepthFormat policy

Foundation 11 completes exactly
`Microsoft.Xna.Framework.Graphics.DepthFormat`. It is a non-flags named
`int32` enum with explicit `None=0`, `Depth16=1`, `Depth24=2`, and
`Depth24Stencil8=3` constants. The synthetic CLR `value__` field is excluded,
so five source identities map to exactly four Go identities.

All raw assignments are explicit and production contains no `iota`. The zero
value is `None`; arbitrary signed `int32` values remain representable without
validation. There is no flags marker, Stringer, parser, depth/stencil-bit or
size helper, SurfaceFormat conversion, native enum mirror, or GPU format
mapping. `Depth24Stencil8` is one ordinary enum value, not a flags expression.

Completion of the enum contract does not claim runtime support for Depth16,
Depth24, stencil, depth-stencil targets, backbuffer depth, depth/stencil testing,
allocation, or clearing. All five direct reverse dependents remain deferred:
GraphicsAdapter, PresentationParameters, RenderTarget2D, RenderTargetCube, and
GraphicsDeviceManager.

## Foundation 11 qualification evidence

- [x] Independently confirm the exact five-source-identity/four-Go-identity contract.
- [x] Complete DepthFormat with compiler-measured local strict zero.
- [x] Retain all four exact raw values and exclude `value__`.
- [x] Qualify None zero value and arbitrary positive/negative raw values.
- [x] Grow verifier mutations from 117 to 132 without an allowlist or mapping exception.
- [x] Grow the behavior corpus from 262 to 268 with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 194 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/depth-format-evidence.md`.

```text
FOUNDATION_MILESTONE_11_COMPLETE=true
```

## Foundation 12 GraphicsProfile policy

Foundation 12 completes exactly
`Microsoft.Xna.Framework.Graphics.GraphicsProfile`. It is a non-flags named
`int32` enum with explicit `Reach=0` and `HiDef=1` constants. The synthetic CLR
`value__` field is excluded, so three source identities map to exactly two Go
identities.

Both raw assignments are explicit and production contains no `iota`. The zero
value is `Reach`; arbitrary signed `int32` values remain representable without
validation, clamping, panic, or error. There is no flags marker, Stringer,
parser, `IsReach`/`IsHiDef`/`SupportsHiDef`/`FeatureLevel` helper, native enum
mirror, or GPU capability mapping. `Reach | HiDef` is an ordinary Go integer
expression, not an XNA flags composition.

Completion of the enum contract claims managed metadata only. `Reach` and
`HiDef` are metadata values, not runtime GPU capability claims: no actual Reach
support, HiDef support, profile selection, feature-level detection, shader-model
support, renderer capability mapping, or native profile negotiation is claimed.
All four direct reverse dependents remain deferred: GraphicsAdapter,
GraphicsDevice, GraphicsDeviceInformation, and GraphicsDeviceManager. A
completed parameter or property type never authorizes its consumer.

## Foundation 12 qualification evidence

- [x] Independently confirm the exact three-source-identity/two-Go-identity contract.
- [x] Complete GraphicsProfile with compiler-measured local strict zero.
- [x] Retain both exact raw values and exclude `value__`.
- [x] Qualify Reach zero value and arbitrary positive/negative raw values.
- [x] Grow verifier mutations from 132 to 144 without an allowlist or mapping exception.
- [x] Grow the behavior corpus from 268 to 274 with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 193 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/graphics-profile-evidence.md`.

```text
FOUNDATION_MILESTONE_12_COMPLETE=true
```

## Foundation 13 ButtonState policy

Foundation 13 completes exactly `Microsoft.Xna.Framework.Input.ButtonState`. It
is a non-flags named `int32` enum with explicit `Released=0` and `Pressed=1`
constants. The synthetic CLR `value__` field is excluded, so three source
identities map to exactly two Go identities. It belongs to the `Input` package,
never to `Graphics` or the `Framework` root.

Both raw assignments are explicit and production contains no `iota`. The zero
value is `Released`; arbitrary signed `int32` values remain representable
without validation, clamping, panic, or error. There is no flags marker,
Stringer, parser, `IsPressed`/`IsReleased`/`Bool`/`FromBool`/`Toggle` helper,
native enum mirror, or input backend mapping. `Released | Pressed` is an
ordinary Go integer expression, not an XNA flags composition.

Completion of the enum contract claims managed metadata only. `Released` and
`Pressed` are metadata values, not runtime input capability claims: no mouse
button behavior, gamepad button behavior, D-pad behavior, button polling,
device enumeration, connection state, dead zone, or vibration is claimed. All
three direct reverse dependents remain deferred: GamePadButtons, GamePadDPad,
and MouseState. A completed property or parameter type never authorizes its
consumer.

## Foundation 13 qualification evidence

- [x] Independently confirm the exact three-source-identity/two-Go-identity contract.
- [x] Complete ButtonState in the Input package with compiler-measured local strict zero.
- [x] Retain both exact raw values and exclude `value__`.
- [x] Qualify Released zero value and arbitrary positive/negative raw values.
- [x] Grow verifier mutations from 144 to 157 without an allowlist or mapping exception.
- [x] Add a source-level self-test that rejects an accidental `xna:flags` directive at the declaration site.
- [x] Grow the behavior corpus from 274 to 280 with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 192 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/button-state-evidence.md`.

```text
FOUNDATION_MILESTONE_13_COMPLETE=true
```

## Foundation 14 pure-managed batch A policy

Foundation 14 is the first multi-type autonomous batch. It completes exactly
25 public XNA types carrying 121 mapped Go identities, every one an ordinary or
flags enum whose only public-signature dependency is `System.Int32`.

Types were consumed by the established ranking — highest reverse fan-out first,
ties broken by the smallest source closure — with the dependency graph
regenerated and reranked after every completed type. Batch membership required
all sixteen safety conditions: mapped dependencies, pure managed logic, no CNA
ABI or cgo expansion, no renderer, `GraphicsDevice`, `Texture2D`, or
`SpriteBatch` work, no ownership/lifetime architecture, no callbacks, no thread
affinity, no hardware-positive behavior, no platform service, no new BCL
projection subsystem, an exactly known pinned surface, deterministic semantics
established from retained authoritative XNA evidence, and no stub behavior.

Every enum reuses the established projection unchanged: a named `int32` type,
explicit raw values, no `iota`, the `// xna:flags` directive present exactly for
the pinned flags enums, the synthetic CLR `value__` field excluded, and no
validation, Stringer, parser, helper API, native mirror, or backend mapping.
Two enums (`AudioChannels`, `Buttons`) declare no zero literal, so their Go zero
value is an ordinary undefined raw value and equals no named constant. No new
general mapping rule was required, so `docs/xna-go-mapping.md` is unchanged.

The batch removed `MISSING_TYPE` diagnostics only. The five partial runtime
types were off limits: `MISSING_MEMBER` stayed at 177 and `PARTIAL_TYPES` at 5
through all 25 steps, even though `CubeMapFace`, `GraphicsDeviceStatus`, and
`PrimitiveType` have `GraphicsDevice` among their deferred reverse consumers.

Three namespaces gained their first Go package — `Audio`, `Media`, and nested
`Input/Touch` — each by the deterministic `packagePathForNamespace` rule and
each carrying only a `doc.go` and enum files. This is a namespace fact and no
audio, media, or touch capability claim.

Completion of these enum contracts claims managed metadata only. No render
target, cube map, primitive draw, render state, sampler, buffer, presentation,
device-loss, effect, audio, microphone, media, video, touch panel, or game pad
capability is claimed, and no completed literal authorizes its consumer.

## Foundation 14 qualification evidence

- [x] Independently confirm all 25 pinned enum contracts, admitting kind, flags, underlying storage, base type, interfaces, `value__`, exact literal names, and exact raw values.
- [x] Complete 25 types and 121 identities with compiler-measured local strict zero for every one.
- [x] Move the scoreboard by exactly the predicted delta at every one of the 25 steps.
- [x] Add one table-driven `foundation14EnumClosures` verifier category covering all 25 types with no allowlist or mapping exception.
- [x] Add 304 exhaustive negative verifier cases across 14 structural defects, each with an asserted clean baseline.
- [x] Grow the declared mutation inventory from 157 to 177 cases.
- [x] Add per-package source-level `xna:flags` self-tests and a negative fixture for the detector itself.
- [x] Grow the behavior corpus from 280 to 411 with explicit XNA/Go provenance and zero failures.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Preserve every mismatch, leak, allowlist, unmeasured, interface-witness, and CNA ABI counter.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 167 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/foundation-14-pure-managed-batch-evidence.md`.

```text
FOUNDATION_MILESTONE_14_COMPLETE=true
BATCH_NAME=PURE_MANAGED_BATCH_A
```

## Foundation 15 pure-managed batch B policy

Foundation 15 completes 12 public XNA types carrying 122 mapped Go identities
in three clusters: the last five safe pure-managed leaf enums, the five
GamePad and Mouse value structs, and the two Touch value structs. Cluster A
closes the safe leaf-enum category — no dependency-complete missing enum
remains.

Public surface authority remains the pinned XNA 4.0 Windows contract.
Reference *behavior* for the value structs is read as IL from the retained
original assemblies, whose hashes are recorded in the milestone evidence. Every
clamping rule, hash algorithm, ToString layout, and equality rule is taken from
that IL. Nothing is inferred from a third-party reimplementation.

Pure managed value structs project as Go value types with no synthetic error
result on any member, because their reference implementations are infallible
managed value work. The verifier measures that claim directly as
`errorResults` in the value-struct closure category, and a dedicated negative
fixture rejects any member that gains an error.

Two reference asymmetries are reproduced deliberately and must not be
"corrected":

- `GamePadThumbSticks` clamps NaN to 1 because the XNA Vector2 minimum keeps
  its second operand, while `GamePadTriggers` propagates NaN because
  `System.Math.Min`/`Max` do; and `Math.Max(-0, 0)` turns negative zero into
  positive zero.
- `TouchLocation.op_Equality` compares all seven fields including both
  `TouchLocationState` values, while `Equals(TouchLocation)` ignores both. The
  two therefore disagree, exactly as the reference does.

`MouseState` uses its own XOR hash with no `Int32.MaxValue` substitution,
while the four game pad values use `Helpers.SmartGetHashCode`, which does
substitute. The resulting compatible collisions are intentional.

A verifier defect was fixed: `System.TimeSpan` is now package-qualified as
`framework.TimeSpan` outside the framework package, matching what
`mapping-rules.json` has always declared and the qualification rule already
applied to XNA types and `Iterator`. Expected type and member counts are
unchanged.

Completing these types claims managed metadata and managed value behavior
only. No game pad, mouse, or touch capability is claimed; CNA-Go still exposes
no GamePad, GamePadState, GamePadCapabilities, Mouse, or TouchPanel type and no
input backend.

Types whose only public surface is read-only device capability with no public
constructor — `TouchPanelCapabilities`, `GamePadCapabilities`, `RendererDetail`,
`DisplayMode` — remain deferred. Implementing them would expose getters that
can only ever report a fabricated capability.

## Foundation 15 qualification evidence

- [x] Independently confirm all 12 pinned contracts and read cluster B/C behavior from hash-verified retained assemblies.
- [x] Complete 12 types and 122 identities with compiler-measured local strict zero for every one.
- [x] Add a `foundation15ValueStructClosures` category that measures the zero-synthetic-error claim directly.
- [x] Generalise the enum closure and defect machinery across milestones; 366 enum negative cases and 70 value-struct negative cases.
- [x] Grow the declared mutation inventory from 177 to 210 cases.
- [x] Grow the behavior corpus from 411 to 476 with explicit XNA/Go provenance and zero failures.
- [x] Fix the `System.TimeSpan` package-qualification defect without moving any expected count.
- [x] Preserve all five partial types and their combined 177 missing members.
- [x] Re-verify the exact Foundation-11 pinned native library by hash and reproduce both native reports against it.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Normal strict verification remains red only for 155 genuinely missing types
and 177 members on five explicitly deferred native/runtime partial types.
Exact evidence is in `docs/foundation-15-pure-managed-batch-evidence.md`.

```text
FOUNDATION_MILESTONE_15_COMPLETE=true
BATCH_NAME=PURE_MANAGED_BATCH_B
```

## Foundation 16 GamePadState policy

Foundation 16 completes exactly one type, `Microsoft.Xna.Framework.Input.GamePadState`,
which became dependency-complete when the Foundation-15 game pad values landed.
Its closure is 15 source identities and 15 mapped Go identities.

Its private `XINPUT_STATE` snapshot is reproduced as an unexported Go struct
because `IsButtonDown` reads the packed form rather than the public values. The
reference `FillInternalState` is pure managed bit packing with no P/Invoke, no
marshalling, and no device access, and the Go reproduction keeps that property:
the snapshot is never exposed, marshalled, or handed to a native library.

Every XInput bit the reference packs coincides with the pinned `Buttons`
literal of the same name, and that coincidence is measured by a test rather
than assumed.

Only the `IndependentAxes` dead-zone mode is reachable from the completed
public surface, because `IsButtonDown` hard-codes it. The Circular and None
modes are reachable only through the unimplemented `GamePad.GetState` and are
deliberately absent rather than written as unreachable code.

Two reference behaviors are asserted rather than smoothed over:
`IsButtonDown(0)` reports true because the empty mask is trivially contained,
and a multi-bit query requires every requested bit to be present.

ECMA-335 leaves float-to-integer conversion of NaN unspecified, and so does Go.
The CIL `conv.i2` and `conv.u1` conversions the reference uses are therefore
spelled out explicitly, deriving NaN to zero from the x86/x64 integer
indefinite value rather than leaving it to the compiler.

`IsConnected` reports what the constructor stored. Completing this type claims
no game pad capability: CNA-Go still exposes no `GamePad` type and performs no
polling, device enumeration, connection detection, or vibration.

## Foundation 16 qualification evidence

- [x] Read the full closure — constructors, FillInternalState, dead-zone utilities, IsButtonDown, hashing, formatting, equality — from the hash-verified retained assembly.
- [x] Complete the type with compiler-measured local strict zero and `errorResults: 0`.
- [x] Measure, rather than assume, that every packed XInput bit equals its pinned `Buttons` literal.
- [x] Assert both dead-zone boundaries, the NaN conversion rule, the empty-mask rule, and the all-bits-required rule.
- [x] Extend the value-struct defect matrix to 8 types, 91 identities, and 80 negative cases; grow the mutation inventory from 210 to 217.
- [x] Grow the behavior corpus from 476 to 487 with zero failures.
- [x] Reproduce both native reports against the exact Foundation-11 pinned library.
- [x] Requalify Go, native stress, unchanged template, deterministic artifact, and isolated-consumer gates.

Exact evidence is in `docs/foundation-16-game-pad-state-evidence.md`.

```text
FOUNDATION_MILESTONE_16_COMPLETE=true
```

## Safe pure-managed seam exhausted at the old rules

After Foundation 16 no dependency-complete missing type could be completed
without either a public-API policy decision or a fabricated device capability.
Search was no longer the bottleneck; the blocked groups were recorded in the
Foundation-16 evidence and in NEXT.md. Foundation 17 settles the first two of
those decisions.

## Foundation 17 pure managed class and per-operation fallibility policy

Two mapping rules were too coarse and are replaced.

CLR `class` is not evidence of native backing. A class is classified as pure
managed only when authoritative Microsoft XNA IL proves the selected public
behavior is backed entirely by managed fields and deterministic managed code,
so it owns no CNA native object and needs no FFI, native allocation, device
query, native destruction, callback registration, thread-affinity lifecycle, or
external hardware state. `Game`, `GraphicsDeviceManager`, `GraphicsDevice`,
`SpriteBatch`, and `Texture2D` are excluded and keep their fallible facades.
Classification changes fallibility only: an admitted class is still a CLR
reference type and still projects as `*T`.

Fallibility belongs to one projected operation, never to a type or a property.
A constructor, a method, a property getter, and that same property's setter are
each classified independently, keyed as `constructor|Name`, `method|Name`,
`field|Name`, `property-get|Name`, `property-set|Name`, and `property|Name`.
The whole-property key stays supported for properties that genuinely validate
on read and on write, and using it for an assignment-only validation is a
measured defect.

Foundation 17 closes `Audio.AudioListener` (9 identities, no error result) and
`Audio.AudioEmitter` (11 identities, exactly one error result, on
`SetDopplerScale` alone). `set_DopplerScale` guards its store with `bge.un.s`,
so every NaN and both zeros are accepted and only negative-ordered values
throw; that is reproduced rather than corrected. Both constructors leave
`Position` and `Velocity` reading back with a negative-zero Z because the
constructor stores `Vector3.Zero` unflipped while the getter flips; that is
also reproduced rather than normalized.

## Foundation 17 qualification evidence

- [x] Read both classes, `UnsafeNativeStructures::FlipHandedness`, the `Vector3` static constructor, and the thrown `FrameworkResources` string from the hash-verified retained assembly.
- [x] Generalize the classification and fallibility tables without hard-coding any type into verifier logic.
- [x] Add the `managedClassClosure` category measuring reference projection, accessor pairs, and per-accessor fallibility.
- [x] Name the accessor and the direction in every `ERROR_MAPPING_MISMATCH`, covering all four accessor cases.
- [x] Add 26 target-side and 6 classification-side negative cases; grow the mutation inventory from 217 to 249.
- [x] Grow the behavior corpus from 487 to 535 with zero failures.
- [x] Keep `MISSING_MEMBER` at 177 and `PARTIAL_TYPES` at 5, with every mismatch, leak, allowlist, and unmeasured counter at zero.

Exact evidence is in `docs/foundation-17-managed-class-evidence.md`.

```text
FOUNDATION_MILESTONE_17_COMPLETE=true
```

## Foundation 18 managed interface projection policy

An interface contributes no fallibility of its own. Each projected operation is
classified by the execution boundary it crosses, and the boundary is read from
the reference implementor IL in the assembly that declares the interface, never
from a guess about an implementor that does not exist. Where every shipped
implementor agrees, that agreement is the contract's measured behavior; where
no implementor's behavior can be read, the interface stays deferred.

Because Foundation 17 made fallibility per operation, one contract may mix the
two boundaries honestly, and `IEffectFog` is the first that does.

Foundation 18 closes four contracts and implements none of them:

```text
Graphics.IEffectMatrices   3 -> 6   uniformly managed, 0 errors
Graphics.IEffectFog        4 -> 8   mixed, 2 errors on FogColor alone
IGameComponent             1 -> 1   runtime boundary, 1 error
IGraphicsDeviceManager     3 -> 3   runtime boundary, 3 errors
```

`IUpdateable`, `IDrawable`, and `IGraphicsDeviceService` remain deferred on one
specific mapping gap: their events are typed `System.EventHandler<System.EventArgs>`,
a BCL generic delegate with no declared mapping that currently degrades to
`any`. That gap is recorded rather than papered over.

## Foundation 18 qualification evidence

- [x] Read both interface declarations and all five shipped effect implementors, plus both game-component implementors and the device manager, from hash-verified retained assemblies.
- [x] Prove the `IEffectFog` split rather than assume it: `FogColor` reaches `EffectParameter`, which calls unmanaged D3DX and throws on a failed HRESULT, while its six siblings are managed field access.
- [x] Add the `managedInterfaceClosure` category with a pinned per-operation fallibility table that the mapping tables cannot silently override.
- [x] Add 37 applied and 7 accounted-for skipped target-side cases plus 4 classification-side cases; grow the mutation inventory from 249 to 290.
- [x] Add compile-time conformance doubles in both packages and grow the behavior corpus from 535 to 541 with zero failures.
- [x] Keep `MISSING_MEMBER` at 177 and `PARTIAL_TYPES` at 5, and assert that `GraphicsDeviceManager` is *not* bound to `IGraphicsDeviceManager`.

Exact evidence is in `docs/foundation-18-interface-evidence.md`.

```text
FOUNDATION_MILESTONE_18_COMPLETE=true
```

## Foundation 19 System.IntPtr and descriptor policy

`System.IntPtr` projects to Go `uintptr`. That is the opaque pointer-sized bit
value the public XNA contract carries at that position and nothing more: it may
not be dereferenced, it is not a CNA or SDL handle, it is no evidence that a
window exists, and it authorizes neither `unsafe.Pointer` nor native device
creation. `IntPtr.Zero` is `uintptr(0)`, and no signed numerical ordering is
claimed for these values even though CLR `IntPtr` is signed.

`RAW_HANDLE_LEAK` is narrowed by exactly that much: a public `uintptr` is
admitted only at a signature position where authoritative XNA metadata declares
`System.IntPtr`. Six CLR members do. Every other route to a pointer-sized word
in public surface still leaks, and `unsafe.Pointer` remains a
`PUBLIC_NATIVE_FFI_LEAK` rather than collapsing into the same category.

Foundation 19 then closes `Graphics.PresentationParameters`: 13 source members,
23 Go identities, no error result. It is a pure managed descriptor over one
nested `Settings` value struct. **A descriptor is not a device.** Storing a
platform window handle authorizes no `GraphicsDevice` creation or reset, no
`GraphicsAdapter`, no display enumeration, no native window lookup, no SDL
call, and no presentation.

Two reference facts are reproduced rather than corrected: the constructor's
only statement is `IsFullScreen = true`, so a fresh descriptor is full-screen
on a zero-sized back buffer; and XNA 4.0 has **no `Clear` method** on this type,
so none was invented and no MonoGame or FNA default was copied.

## Foundation 19 qualification evidence

- [x] Enumerate every `System.IntPtr` position in the pinned profile, not only the one implemented, and pin each member's count.
- [x] Narrow `RAW_HANDLE_LEAK` positionally against the expected surface so no separate allowlist can drift.
- [x] Fixture both directions: 2 cases that must stay clean and 10 that must leak, including slice, pointer, index-drift, invented-member, and named-type routes, plus one that must stay a `PUBLIC_NATIVE_FFI_LEAK`.
- [x] Derive the complete `PresentationParameters` contract from IL: constructor, `Clone`, ten read/write properties, computed `Bounds`, no validation, no `Clear`.
- [x] Make two shared managed-class defects owner-relative so the matrix applies unchanged to a Graphics-package type.
- [x] Grow the mutation inventory from 290 to 314 and the behavior corpus from 541 to 548, both with zero failures.
- [x] Keep `MISSING_MEMBER` at 177, `PARTIAL_TYPES` at 5, and all three leak counters at zero.

Exact evidence is in `docs/foundation-19-intptr-presentation-parameters-evidence.md`.

```text
FOUNDATION_MILESTONE_19_COMPLETE=true
```

## Foundation 20 read-only collection policy

`TouchCollection` was previously grouped with the touch types blocked on device
capability. The IL says otherwise: two public constructors, no `calli`, no
P/Invoke, no device access, and eight private inline `TouchLocation` fields
rather than an array. Completing it claims no touch capability; CNA-Go still
has no `TouchPanel` and reads no device.

It is the first cluster that is simultaneously a CLR value type and fallible,
which only Foundation 17's per-operation fallibility makes expressible.
Admitting a CLR struct to the pure-managed classification changes its
fallibility and nothing else: it stays a copied Go value, and the closure
asserts the constructor projects `TouchCollection` rather than a pointer.

Two mapping rules are settled. `System.Collections.Generic.IList<T>` projects
like `ICollection<T>` — a concrete Go method set, no fabricated BCL package —
because the indexer and index methods it adds are already declared public
members of the XNA collection. And a collection that declares its own public
enumerator type projects that type from `GetEnumerator`; the `Iterator<T>`
adapter is for collections that declare none.

A read-only collection keeps its mutators. Every `IList<T>` mutator here is an
unconditional `NotSupportedException` that validates nothing first, and those
throws project as errors rather than being dropped.

## Foundation 20 qualification evidence

- [x] Re-derive the whole closure from the hash-verified retained assembly, including the private eight-slot storage and the post-increment slot selection.
- [x] Preserve the guard order, the `> 8` capacity boundary, and `CopyTo`'s 64-bit overflow arithmetic.
- [x] Preserve the search asymmetry: `IndexOf` uses the equality operator, so a location `Equals` accepts is missed.
- [x] Preserve the cursor's -1 start, its clamp on exhaustion, and the errors at both ends.
- [x] Generalize the managed closure across both CLR kinds and derive the expected constructor projection from pinned metadata.
- [x] Add `dropped_error` and shape predicates: 61 applied and 9 accounted-for skipped cases across five types; mutation inventory 314 to 336.
- [x] Grow the behavior corpus from 548 to 558 with zero failures.

Exact evidence is in `docs/foundation-20-touch-collection-evidence.md`.

```text
FOUNDATION_MILESTONE_20_COMPLETE=true
```

## Foundation 21 service registry and the exhausted frontier

`GameServiceContainer`'s only unsatisfied dependency was the declared direct
interface `System.IServiceProvider`, whose single member `GetService(Type)` is
already an ordinary public member of the class. That generalizes the rule
Foundation 20 established for `IList<T>`: a BCL interface whose members the XNA
type already declares publicly adds no projected surface and needs no separate
Go interface.

The type is one private `Dictionary<Type, object>`: 4 source members, 4 Go
identities, 3 with an error. Two reference behaviors are preserved that a
reimplementation would likely get wrong. `AddService` checks for a duplicate
**before** checking assignability, so an unassignable provider for an
already-registered type reports the duplicate. And a miss is an absence rather
than a failure: `RemoveService` on an unregistered type succeeds, and
`GetService` on one returns nil with no error.

Completing it wires nothing up. CNA-Go's `Game` exposes no `Services` property,
so nothing in the binding populates or consults a container.

## Foundation 21 qualification evidence

- [x] Read the whole type and its four `Resources` strings from the hash-verified retained assembly.
- [x] Preserve the duplicate-before-assignability ordering and the absence-is-not-failure rule.
- [x] Document the one Go/CLR boundary honestly: a non-nil interface holding a typed nil pointer has no CLR counterpart.
- [x] Extend the shared type closure to 6 types: 72 applied and 12 accounted-for skipped cases; mutation inventory 336 to 347.
- [x] Grow the behavior corpus from 558 to 564 with zero failures.

Exact evidence is in `docs/foundation-21-game-service-container-evidence.md`.

```text
FOUNDATION_MILESTONE_21_COMPLETE=true
```

## Foundation 22 the general CLR event architecture

Every public CLR event in the profile is `System.EventHandler`1<T>`, so two BCL
shapes close the whole event surface. Both are general, not per-type.

- [x] `System.EventArgs -> *framework.EventArgs`, keeping CLR reference
      semantics; `nil` is not its `Empty`.
- [x] `System.EventArgs.Empty -> framework.EventArgsEmpty()`, one stable private
      singleton behind a function so no consumer can reassign it.
- [x] `System.EventHandler<T> -> framework.EventHandler[T]`, the generic
      argument carried exactly and never degraded to `any`.
- [x] Handler results `error`, recorded as a Go projection of the CLR exception
      channel rather than an XNA return identity.
- [x] `EventSubscription` materialised as a real opaque token;
      `EventSource[T]` added as public language support.
- [x] Non-XNA CLR bases made a measured relationship, exhaustive over the
      profile, with `System.EventArgs` mapped and eight bases deferred.
- [x] Preserve the settled two-accessor event projection and every XNA counter.

## Foundation 22 qualification evidence

Exact evidence is in `docs/foundation-22-event-architecture-evidence.md`.

```text
FOUNDATION_MILESTONE_22_COMPLETE=true
```

## Foundation 23 the first event contracts

- [x] `IUpdateable` and `IDrawable`, re-derived from IL and classified
      `PURE_MANAGED` on their own implementor evidence.
- [x] The three dependency-complete `System.EventArgs` carriers, preserving
      public and `assembly` construction exactly.
- [x] External conformance canary as its own module, `GOWORK=off`, no sibling
      checkout.

## Foundation 23 qualification evidence

Exact evidence is in `docs/foundation-23-component-contracts-evidence.md`.

```text
FOUNDATION_MILESTONE_23_COMPLETE=true
```

## Foundation 24 and 25 BCL relationships and the frontier

- [x] `System.IDisposable` registered as a measured relationship that adds no
      projected Go surface: no `Disposable` type, no `Close`, no `io.Closer`,
      no finalizer, no ownership wrapper, no synthesized `Dispose`.
- [x] All eight non-XNA direct interfaces declared and measured with
      `projectedMembers == 0`.
- [x] The two internal XNA interfaces declared `INTERNAL_NO_SURFACE` rather
      than skipped.
- [x] Frontier regenerated; `GameComponentCollection`, `LaunchParameters`, and
      `GameWindow` re-derived and deferred with recorded evidence.

## Foundation 24 and 25 qualification evidence

Exact evidence is in `docs/foundation-24-idisposable-relationship-evidence.md`
and `docs/foundation-25-frontier-closure-evidence.md`.

```text
FOUNDATION_MILESTONE_24_COMPLETE=true
FOUNDATION_MILESTONE_25_COMPLETE=true
```

## Foundation 26 the general BCL base-class composition projection

- [x] A CLR class inheriting a supported BCL collection base projects as a
      concrete Go reference type CONTAINING a private generic adapter, and
      re-exposes the base's public surface through measured forwarding.
- [x] A public member inherited from such a base is still public CLR surface
      and must not disappear because the XNA metadata does not declare it.
- [x] The adapter is never an XNA type, an exported field, a public base-class
      object, an embedded public API, or a handle; exported embedding AND
      embedding the unexported adapter are both rejected.
- [x] The base's protected virtuals are an unexported Go interface, so only a
      type declared in this module can supply or reach a hook, and every
      mutating public operation routes through it.
- [x] `Collection`1` behavior read from the pinned mscorlib 4.0.30319.1, the
      binary every XNA assemblyref names, never from modern .NET or Go habit.
- [x] `GameComponentCollection` completed: 7 declared members to 9 identities,
      11 inherited CLR members to 12, 21 total.
- [x] Three provenance classes kept distinct; `REFERENCE_MEMBERS` unchanged at
      2964 and the pinned 3243 XNA-declared projection count unmoved.
- [x] 22 negative controls; the collection proved from outside the repository.

## Foundation 26 qualification evidence

Exact evidence is in `docs/foundation-26-bcl-base-composition-evidence.md`.

```text
FOUNDATION_MILESTONE_26_COMPLETE=true
```

## Foundation 27 BCL types at signature positions

- [x] A BCL type the contract carries at a public signature position needs a
      public Go spelling, on the `System.TimeSpan` and `System.EventHandler<T>`
      footing; `ReadOnlyCollection`1` maps to `*framework.ReadOnlyCollection[T]`.
- [x] The signature-adapter role and the base-adapter role are independent, and
      one CLR type may hold both; `ReadOnlyCollection`1` is SUPPORTED as the
      first and DEFERRED as the second.
- [x] A signature adapter's exported surface is pinned to the exact public CLR
      member inventory, so an adapter type is not a hole in the
      unexpected-member scan.
- [x] Read-only needed no new decision: every mutator is a private explicit
      implementation the settled BCL-interface rule already excludes.
- [x] Enumeration semantics belong to the underlying list; an array-backed view
      is deliberately not version-checked.
- [x] `Media.VisualizationData` completed; `singleEquals` reproduces
      `System.Single::Equals`, which treats NaN as equal.
- [x] 9 negative controls, three of which attack read-only directly.

## Foundation 27 qualification evidence

Exact evidence is in `docs/foundation-27-read-only-collection-evidence.md`.

```text
FOUNDATION_MILESTONE_27_COMPLETE=true
```

## Foundation 28 the device-publication contract

- [x] `Graphics.IGraphicsDeviceService` completed; every operation infallible
      on the reference implementor's one-`ldfld` accessor.
- [x] The per-contract boundary rule sharpened: the same
      `GraphicsDeviceManager` implements an infallible contract and a fallible
      one, so the boundary is read per contract and never per class.
- [x] Frontier re-derived; three types the type-level graph calls reachable
      shown from IL not to be, each blocker named to the exact member.
- [x] `GameComponent` blocked on `Game::Components`, a protected-partial
      member; `GraphicsResource` blocked on native ownership; `Microphone`
      device-blocked despite its signature being unblocked.

## Foundation 28 qualification evidence

Exact evidence is in
`docs/foundation-28-device-service-and-frontier-evidence.md`.

```text
FOUNDATION_MILESTONE_28_COMPLETE=true
```

## Foundation 29 the audited frontier

- [x] A DEFERRED base must NAME what blocks it, to the exact inherited member
      or the exact architecture decision; one that records nothing fails.
- [x] 21 blockers across seven deferred bases, each SUBSYSTEM or ARCHITECTURE.
- [x] `System.Exception` audited: all eight derived types declare only
      constructors, three inherited members need three distinct unmapped
      subsystems, and two architecture obstacles are the material ones.
- [x] `System.Attribute` audited and found no easier.
- [x] `addsProjectedSurface` corrected to be true exactly for COMPOSED.

## Foundation 29 qualification evidence

Exact evidence is in `docs/foundation-29-exception-frontier-evidence.md`.

```text
FOUNDATION_MILESTONE_29_COMPLETE=true
```

## Foundation 30 Game's managed Components and Services

- [x] `Game::get_Components` and `::get_Services` derived from IL as seven
      bytes each — one `ldfld` over a field the constructor assigns once — so
      both are managed CLR state, infallible, and stable in identity.
- [x] Both are Go-owned. Routing either through the C ABI would invent a
      native owner and a native failure mode the reference does not have.
- [x] `Game` allocates one collection and one container at the reference's
      construction point and subscribes its own two handlers to the collection
      before any consumer can, so a consumer's handler always runs second.
- [x] The component engine derived, not remembered: Game does NOT sort; both
      order comparers return 0 only for reference identity; ties are stable
      through an explicit forward walk, not a stable sort; the order-changed
      handlers read the SENDER; `inRun` is raised only after `Initialize`
      returns; initialization precedes both placements.
- [x] Mapper defect fixed: a property accessor's fallibility flag was
      inherited rather than recomputed, so an accessor-level classification
      could raise fallibility but never lower it. Guarded by a structural
      invariant over all 3255 expected members.

## Foundation 30 qualification evidence

Exact evidence is in `docs/foundation-30-game-managed-state-evidence.md`.

```text
FOUNDATION_MILESTONE_30_COMPLETE=true
```

## Foundation 31 base behavior is never automatic

- [x] `GameCallbacks` projects CLR protected virtual OVERRIDES, and in CLR the
      override decides whether and when the base runs. Running the base body
      around a callback would be a different contract that resembles XNA's.
- [x] Five package-level base-call functions, so `Game`'s projected member
      surface gains no name Microsoft never declared.
- [x] Measured language support, not XNA identity: each adapter pinned to a
      real protected virtual, no arbitrary `GameBase*` helper admitted, one per
      callback member required, and the names admitted FROM the registry rather
      than by an allowlist. `REFERENCE_MEMBERS` and `EXPECTED_GO_MEMBERS`
      unmoved.
- [x] Four deferred reference steps, each classified and each proved
      unobservable from the managed component surface.
- [x] Native callback order audited against XNA's; the relative order of
      everything CNA-Go owns is preserved, and the `GameHost` substitution is
      recorded rather than hidden.

## Foundation 31 qualification evidence

Exact evidence is in `docs/foundation-31-game-base-call-evidence.md`.

```text
FOUNDATION_MILESTONE_31_COMPLETE=true
```

## Foundation 32 GameComponent

- [x] Completed; pure managed on IL that is field access, comparison and
      delegate invoke throughout.
- [x] `Initialize` is the first member in the profile whose fallibility comes
      from a CONTRACT IT IMPLEMENTS rather than its own body.
- [x] Four load-bearing quirks preserved: no constructor validation; the
      `On...` methods ignore their sender and raise with `this`; the setters
      suppress an unchanged value; `Dispose` removes before it announces and is
      not idempotent.
- [x] `lock (this)` projected with `TryLock`, because CLR's Monitor is
      reentrant and reentry is reachable, so a plain Lock would deadlock where
      the reference recurses. The divergence is recorded.
- [x] New general rule: a COMPLETE projected class must satisfy the Go
      projection of every XNA interface its CLR metadata declares, on the
      pointer method set, and `go/types` must say so.

## Foundation 32 qualification evidence

Exact evidence is in `docs/foundation-32-game-component-evidence.md`.

```text
FOUNDATION_MILESTONE_32_COMPLETE=true
```

## Foundation 33 the XNA base frontier

- [x] A second base frontier existed and was silent: `Texture2D` inherits nine
      public members from `Texture` and `GraphicsResource` that CNA-Go does not
      project, and nothing recorded it.
- [x] Twelve XNA-to-XNA relationships, 41 derived types, 25 classified
      blockers, 245 unprojected inherited public members.
- [x] The substantive rule: no derived type of a DEFERRED XNA base may be
      reported COMPLETE. `Texture2D` and `SpriteBatch` are partial for the
      right reason, and that is now checked.
- [x] The open architecture decision stated exactly: a composition rule for an
      XNA-identity base, a third provenance class, and an override adapter for
      the base's protected virtuals.
- [x] `GraphicsDeviceManager` service publication audited in full and blocked:
      the partial manager satisfies neither service contract, and the four
      device events have no raise path in CNA at all.

## Foundation 33 qualification evidence

Exact evidence is in `docs/foundation-33-xna-base-frontier-evidence.md`.

```text
FOUNDATION_MILESTONE_33_COMPLETE=true
```

## Foundation 34 the native Game event bridge

- [x] `Game::Activated`, `Deactivated`, `Exiting` and `Disposed` are eight
      accessors plus the three protected raise sites Microsoft declares --
      eleven fewer missing members.
- [x] The four signals were ALREADY published by the pinned CNA C API and had
      never been reached from Go. No CNA C++ changed, CNA was not rebuilt, and
      the artifact is still `e912cd1d...b116f`; the ABI counters moved because
      CNA-Go binds more of that unchanged binary.
- [x] `OnActivated`/`OnDeactivated` accept a sender, ignore it, and raise with
      `this`. `OnExiting` pushes `ldnull` instead -- one instruction, and
      observable from every handler. `Disposed` has no `On...` method at all.
- [x] Exactly ONE native subscription per event per Game. CNA invokes multiple
      native registrations on one event in REVERSE order, measured, so
      per-handler registration would have inverted the promised dispatch order.
- [x] Installed eagerly at native game creation, because CNA answers
      `CNA_RESULT_THREAD` for `cna_game_subscribe` from any other thread.
      Released after `cna_game_destroy`, because the disposal signal is raised
      from inside it. The `cgo.Handle` is deleted last.
- [x] `CNA_GameEventCallback` returns void, so a handler failure cannot stop the
      game. It is recorded through the existing callback-failure path and
      surfaces from `Run` rather than being discarded, and `inCallback` is
      deliberately not raised.
- [x] `GameCallbacks` untouched: the bridge method went on the INTERNAL
      `interop.Callbacks`, so every existing five-member implementation still
      compiles.
- [x] `Activated`, `Exiting` and `Disposed` are VERIFIED_NATIVE at 60 isolated
      deliveries each. `Deactivated` is NOT_RUN_ENVIRONMENT: HEADLESS has no
      window manager and cannot lose focus, and the counter stays at zero.

## Foundation 34 qualification evidence

Exact evidence is in `docs/foundation-34-game-event-bridge-evidence.md`.

```text
FOUNDATION_MILESTONE_34_COMPLETE=true
```

## Foundation 35 the frame-boundary virtuals

- [x] `Game::BeginRun`, `EndRun`, `BeginDraw` and `EndDraw` complete as methods
      on `Game`, by the rule that already made `GameComponent` complete: the
      mapper redirects exactly the five protected virtuals `GameCallbacks`
      declares, and every other protected virtual is a method whose body is the
      reference base body.
- [x] The two run hooks are `IL_0000: ret`. `inRun` is NOT raised by `BeginRun`:
      `RunGame` raises it before it calls the virtual.
- [x] `Game::graphicsDeviceManager` has one assignment in the whole class, and
      the statement after it calls `CreateDevice`. The native runtime owns the
      device and Foundation 30 already audited that nothing can register into
      `IGraphicsDeviceManager`, so the field is permanently null -- a state the
      reference itself has whenever no manager is registered.
- [x] `BeginDraw`'s Boolean stays a value channel: `DrawFrame` runs
      `if (BeginDraw()) { Draw(); EndDraw(); }`, and the measured native
      `begin_draw` hook does the same. A refused call answers `(false, err)`.
- [x] CNA's `begin_run`, `end_run`, `begin_draw` and `end_draw` correspond
      position for position and are deliberately NOT installed. Base behavior is
      never automatic; forwarding would run the base where CNA-Go picked, make
      it mandatory, and prejudge the override design.
- [x] The override mechanism is STOPPED and reported, not guessed:
      `GameCallbacks` keeps five members, no `GameBase*` helper was added, and
      three materially different public designs remain plausible.

## Foundation 35 qualification evidence

Exact evidence is in `docs/foundation-35-game-frame-hook-evidence.md`.

```text
FOUNDATION_MILESTONE_35_COMPLETE=true
```

## Foundation 36 the signal and frame-hook registries

- [x] Two decisions that lived only in prose became closed registries the strict
      run enforces.
- [x] `gameNativeSignals`: 4 signals, 3 raise sites, 1 runtime-deferred. Three
      is not two short of four -- `Disposed` genuinely has no `On...` method, and
      declaring one now fails. `NOT_RUN_ENVIRONMENT` requires a reason and
      `VERIFIED_NATIVE` forbids one.
- [x] `gameFrameHooks`: 4 hooks, 0 installed, 4 deferred steps. The zero is a
      measured zero with four recorded reasons behind it.
- [x] The base-call closure stated from the other side: no `GameBaseBeginRun`
      may exist, and `BeginDraw`'s Boolean is pinned as a channel separate from
      its error in both the registry and `mapping-rules.json`.
- [x] 31 negative controls from one shared table, plus two accounting controls
      proving neither registry is an XNA identity. Mutation inventory 500 -> 531.

## Foundation 36 qualification evidence

Exact evidence is in `docs/foundation-36-signal-registry-evidence.md`.

```text
FOUNDATION_MILESTONE_36_COMPLETE=true
```

## Foundation 37 the bridge lifetime, proved

- [x] Foundation 34's three lifetime claims were supported by a C probe and
      prose. A probe proves what the C API does, not what the binding does with
      it.
- [x] A fourth `native_stress` scenario runs the SAME Go `Game` twice: the first
      run's four registrations are released and four fresh ones installed, with
      `Exiting` and `Disposed` arriving exactly once per run over 20 isolated
      cycles and zero crashes, use-after-free or double frees.
- [x] The second run's activation is SUPPRESSED, and the scenario asserts it.
      `Game::isActive` is never reset by the reference, so the edge-trigger
      guard sees no transition -- exactly as `HostActivated` does in CLR. That
      is why the activation total is 80 and not 100.
- [x] `Add` and `Remove` with no native game alive are ordinary successes, proved
      20 times.
- [x] `internal/interop` gained its first tests: eight, over the decisions that
      do not reach C -- routing, a dead runtime, first-failure-wins, panic
      containment, release idempotency, name coverage, the Go end of the
      constant chain, and `Callbacks` gaining exactly one internal member.

## Foundation 37 qualification evidence

Exact evidence is in `docs/foundation-37-bridge-lifetime-evidence.md`.

```text
FOUNDATION_MILESTONE_37_COMPLETE=true
```

## Foundation 38 qualification evidence

Exact evidence is in `docs/foundation-38-frame-hook-override-evidence.md`.

The frame-hook override branch that Foundation 36 stopped and reported is
closed. Of the three candidate shapes it recorded, the second -- per-hook
capability interfaces -- was selected and then narrowed: the four contracts are
UNEXPORTED and satisfied structurally, so the mechanism publishes no new public
framework identity, adds no member to `GameCallbacks`, and introduces no mutable
per-Game registration state.

```text
FOUNDATION_MILESTONE_38_COMPLETE=true
```

## Foundation 39 qualification evidence

Exact evidence is in `docs/foundation-39-game-disposal-evidence.md`.

The second branch Foundation 36 stopped and reported is closed, and it is closed
by CORRECTING a shipped divergence rather than preserving it. Of the two options
that milestone recorded, neither was taken as written: the native signal is
neither dropped nor left driving the public event. It stays bound and measured
as a CNA lifecycle signal -- which is what proves a registration outlives
cna_game_destroy -- while the public event moves to the reference's own managed
raise site.

The registry rule moved with it. "Every event is bound to one canonical signal
through the raise path the reference uses" is satisfied by a binding whose
semantics do not align; the rule is now "every projected XNA event must have its
AUTHORITATIVE XNA raise path, and a native signal may implement that path only
when the semantics align".

```text
FOUNDATION_MILESTONE_39_COMPLETE=true
```

## Foundation 40 qualification evidence

Exact evidence is in `docs/foundation-40-base-substitutability-evidence.md`.

The XNA-to-XNA inheritance architecture's largest unknown is measured rather
than argued. Fifty-one public signature positions in the profile name a class
that another class derives from; three families -- GameComponent,
GraphicsResource and MathTypeConverter, carrying 25 of the 41 derived types --
are named in NONE of them, and no family is LIVE.

For a family named in zero positions, private named composition with explicit
forwarding is not a compromise: no public reference abstraction can be justified
by anything in the contract, because there is no position for a derived value to
flow through.

```text
FOUNDATION_MILESTONE_40_COMPLETE=true
```

## Foundation 41 qualification evidence

Exact evidence is in `docs/foundation-41-xna-inheritance-evidence.md`.

The architecture decision Foundation 33 named and eight milestones deferred is
made: PRIVATE NAMED COMPOSITION plus EXPLICIT MEASURED FORWARDING, never Go
embedding, with no public accessor for the base object. The third provenance
class, XNA_INHERITED, joins XNA_DECLARED and BCL_INHERITED, and the three are
asserted disjoint and exhaustive.

GameComponent is the first COMPOSED relationship because Foundation 40 measured
that nothing in the profile names it. COMPOSED is about inheritance, not
completeness: neither derived type is projected, and DrawableGameComponent's
remaining blocker -- resolving IGraphicsDeviceService from a framework-package
method body, with two materially different designs available and neither
selected by precedent -- is recorded as the open fork it is.

```text
FOUNDATION_MILESTONE_41_COMPLETE=true
```

## Foundation 42 qualification evidence

Exact evidence is in `docs/foundation-42-game-timing-evidence.md`.

Ten of Game's nineteen remaining members are complete, and CNA was not rebuilt:
all six canonical operations behind them were already exported by the pinned
artifact and had never been reached from Go. The getters project as the field
reads they are in the reference and are infallible; the setters push to the
native loop, because that loop is what the reference's own loop would have read.

The Game's configured values are now what cna_game_create is handed, replacing
the literals it used to pass.

```text
FOUNDATION_MILESTONE_42_COMPLETE=true
```

## Foundation 43 qualification evidence

Exact evidence is in `docs/foundation-43-game-graphics-device-evidence.md`.

Game.GraphicsDevice is complete, projected into the Graphics package by the
cross-package cycle rule. It is reachable because the reference body FALLS BACK
to resolving IGraphicsDeviceService out of Services when its cached field is
null -- which is the state CNA-Go is always in, and the state the reference
itself is in before Initialize runs.

The milestone also corrects two resource strings Foundation 42 derived from
resource KEYS instead of reading from the resource VALUES.

```text
FOUNDATION_MILESTONE_43_COMPLETE=true
```

## Foundation 44 through 61 — the milestone index and the rules they settled

Foundation 1 through 43 are recorded above, one policy section and one
qualification section each. From Foundation 44 the same material lives in one
evidence document per milestone under `docs/`, because each milestone's
argument is long enough to deserve its own file and short enough that a shared
section would only index it. This section IS that index, and it records the
NORMATIVE rules those milestones added; the evidence for each is in its file.

| #  | milestone | evidence |
| -- | --------- | -------- |
| 44 | the live CNA C ABI migration | `foundation-44-abi-migration-evidence.md` |
| 45 | GameWindow and Game.Window | `foundation-45-game-window-evidence.md` |
| 46 | DrawableGameComponent and device-service resolution | `foundation-46-...` |
| 47 | the native session split out of Run | `foundation-47-frame-step-evidence.md` |
| 48 | GraphicsDeviceManager's configuration surface | `foundation-48-...` |
| 49 | the device service the manager publishes | `foundation-49-...` |
| 50 | every SpriteBatch.Draw overload, both native routes | `foundation-50-...` |
| 51 | GraphicsDevice render state | `foundation-51-...` |
| 52 | DisplayMode, and Texture2D's two constructors | `foundation-52-...` |
| 53 | Texture2D's stream surface, both directions | `foundation-53-...` |
| 54 | the generic-method projection rule | `foundation-54-...` |
| 55 | the inherited slot, and the CLR `this` a composed base lost | `foundation-55-composition-identity-evidence.md` |
| 56 | the graphics resource base chain, composed | `foundation-56-graphics-resource-chain-evidence.md` |
| 57 | the routes CNA-Go deliberately does not bind | `foundation-57-unbound-routes-evidence.md` |
| 58 | RenderTarget2D and the substitutability rule | `foundation-58-render-target-evidence.md` |
| 59 | the four graphics state objects and their freeze rule | `foundation-59-state-objects-evidence.md` |
| 60 | applying state objects to the device | `foundation-60-device-state-evidence.md` |
| 61 | the device's texture and sampler collections | `foundation-61-device-collections-evidence.md` |
| 62 | GraphicsDevice's six events, and its disposal | `foundation-62-device-events-evidence.md` |
| 63 | the Content subsystem, and Game::Content | `foundation-63-content-manager-evidence.md` |
| 64 | VertexDeclaration and its element validator | `foundation-64-vertex-declaration-evidence.md` |
| 65 | IndexBuffer, and the shared copy validator | `foundation-65-index-buffer-evidence.md` |
| 66 | VertexBuffer, IVertexType, and two narrowings | `foundation-66-vertex-buffer-evidence.md` |
| 67 | binding buffers to the device, and the draw calls | `foundation-67-device-buffer-binding-evidence.md` |
| 68 | GraphicsAdapter, and a narrowing that belongs to CNA | `foundation-68-graphics-adapter-evidence.md` |
| 69 | SpriteFont, and a measurement that is the reference's | `foundation-69-sprite-font-evidence.md` |
| 70 | SpriteBatch's six DrawString overloads | `foundation-70-draw-string-evidence.md` |
| 71 | TextureCube, Texture3D, and a blocker that was false | `foundation-71-volume-cube-texture-evidence.md` |
| 72 | the Effect cluster, and the draws it unblocked | `foundation-72-effect-cluster-evidence.md` |
| 73 | the rest of GraphicsDevice, and the profile's first pixels | `foundation-73-presentation-evidence.md` |
| 74 | the Dictionary base, LaunchParameters, and a generated roadmap | `foundation-74-dictionary-base-evidence.md` |
| 75 | GraphicsDeviceInformation, and the rest of GraphicsDeviceManager | `foundation-75-device-selection-evidence.md` |
| 76 | System.Exception, and the profile's last missing member | `foundation-76-exception-projection-evidence.md` |
| 77 | the four stock vertex structs | `foundation-77-vertex-structs-evidence.md` |
| 78 | the eight XNA exception types | `foundation-78-exception-family-evidence.md` |
| 79 | BasicEffect, DirectionalLight and IEffectLights | `foundation-79-stock-effect-evidence.md` |
| 80 | AlphaTestEffect, DualTextureEffect and EffectMaterial | `foundation-80-unlit-stock-effects-evidence.md` |
| 81 | EnvironmentMapEffect and SkinnedEffect, closing the stock effects | `foundation-81-lit-stock-effects-evidence.md` |
| 82 | FrameworkDispatcher and TitleContainer, the last root types | `foundation-82-root-types-evidence.md` |
| 83 | OcclusionQuery, and the dynamic-buffer probe | `foundation-83-occlusion-query-evidence.md` |
| 84 | DynamicVertexBuffer and DynamicIndexBuffer, closing the Graphics namespace | `foundation-84-dynamic-buffers-evidence.md` |
| 85 | the first VERIFIED_PIXEL draw | `foundation-85-pixel-draw-evidence.md` |
| 86 | RendererDetail, and the stale audio blocker | `foundation-86-renderer-detail-evidence.md` |
| 87 | SoundEffect and SoundEffectInstance | `foundation-87-sound-effect-evidence.md` |
| 88 | DynamicSoundEffectInstance, Microphone, and the end of the Audio namespace | `foundation-88-audio-namespace-evidence.md` |
| 89 | GamePad, Mouse and TouchPanel, closing the Input namespace | `foundation-89-input-evidence.md` |
| 90 | the Model family, closing the Graphics namespace | `foundation-90-model-family-evidence.md` |
| 91 | StorageDevice and StorageContainer, closing the Storage namespace | `foundation-91-storage-evidence.md` |

**A recorded blocker is a claim, and claims get re-measured** (91).
System.IO.BinaryReader had been DEFERRED since Foundation 29 partly because "the
reader's inherited surface depends on seeking". ContentReader's IL contains ZERO
`Stream::Seek` calls -- the one hit was the substring of `get_CanSeek`, inside a
`private static` helper whose check is skipped when the stream cannot seek. Half
the blocker was wrong for four milestones because nobody re-measured it. A
blocker inherited from an earlier session is evidence about that session, not
about today.

**Prove where a test writes; do not assume it** (91). The Storage slice's root
was measured before anything was written, and it was `~/.local/share/<app>` --
outside the project, in the user's home. CNA builds it from XDG_DATA_HOME, so
the harness redirects it; but the slice does not trust the redirect either. It
reads the root back and refuses to continue unless it lies under the permitted
path. A safety constraint that only a comment enforces is not enforced.

**Fake async is a shape, not a simplification** (91). XNA's storage
BeginXxx/EndXxx invokes the callback and returns a completed result before Begin
exits. So IsCompleted, CompletedSynchronously and a signalled wait handle are
constants -- and a projection that returned false from IsCompleted "to be safe"
would be describing an implementation the reference does not have. CNA's header
says the same, but the evidence is the IL.

### The rules these milestones settled

**A HIDDEN inherited member is unreachable, not renamed** (90). All four Model
collections declare their own `GetEnumerator` with neither `virtual` nor
`newslot` -- C# `new`, which hides `ReadOnlyCollection<T>`'s. Reaching a hidden
base member in C# requires a cast to the base; CNA-Go projects no base type to
cast to, so the inherited member cannot be reached by any consumer and is
EXCLUDED from the adapter rather than given a hashed name by the collision
rule. Each collection keeps exactly one `GetEnumerator`, the derived one. This
is what unblocked a base that had been DEFERRED since Foundation 29.

**A read-only view over a LIST is live; over an ARRAY it is frozen** (90).
`ReadOnlyCollection<T>` stores the `IList` reference, so what the underlying
list can do, the view does. `ModelEffectCollection` wraps a `List<Effect>` its
owner keeps mutating, so its view must see every addition and its enumerator
must be version-checked; its three array-backed siblings can be neither, because
an array's length never changes. One adapter, two sources, and the signatures
carry the difference -- only the List-backed `MoveNext` returns an error.

**A family with no public constructor has no native slice until its factory
lands** (90). The Model family's twelve types declare ZERO public constructors:
a Model arrives only from `ContentManager.Load<Model>`. Its draw path reaches
the device and is still recorded VERIFIED_MANAGED, because nothing outside the
package can build one to draw. Naming that as unclaimed is the honest report;
adding a constructor the reference does not have to make a test possible would
not be.


**A reference body that reaches nothing is still the contract** (89). The
pinned `Microsoft.Xna.Framework.Input.Touch.dll` declares no p/invoke anywhere:
`GetCaps()` is `initobj` and `ret`, `GetState()` updates from a state it just
zeroed, and `ReadGesture()` has no `ret` instruction at all. CNA implements all
fourteen touch routes for real, and binding them was still WRONG -- a projection
that answered real touches would diverge from the runtime this binding targets
on the first `GetState` a game makes. When the reference reaches nothing, the
faithful projection reaches nothing, and the working native routes are recorded
as CONTRACT_DIVERGENCE. The counter that proves it, `TOUCH_PANEL_NATIVE_CALLS`,
is exercised from INSIDE a live game so that its zero means something.

**A mutation that needs hardware is named, not hidden** (89). Three of the
family's forty-four planted defects sit on the CONNECTED-controller branch,
which a build machine with no controller never enters. They are reported as
unkilled with the reason, and the slice is written so an attached controller
would kill them. `GAMEPADS_CONNECTED` reports zero as a measured result rather
than a skipped test, because the reference's own
`ERROR_DEVICE_NOT_CONNECTED` branch -- an empty state and NO exception -- is
exactly what that zero exercises.

**Inheritance is excluded by CLR SIGNATURE, not by name** (55). A derived class
excludes an inherited member only when it declares one with the same kind, name,
generic arity and parameter list. The old name-keyed rule deleted
`DrawableGameComponent`'s inherited public `Dispose()`, which is the member
`Game.Dispose` looks for.

**Inherited and declared members share one overload namespace** (55). A group
larger than one carries `By<ParameterShape>` whichever half declared it.

**Fallibility follows the DECLARING type** (56), and so does stream direction
(58). Both are properties of a member's own body.

**A composed base holds the CLR `this`** (55, 56, 58). It is unexported,
installed by the derived constructor, and used wherever the reference uses
`ldarg.0` as an OBJECT rather than as a path to a field. A base that is itself
composed forwards instead of holding a second copy. The verifier holds this from
the Go syntax tree, counting reaches against `ldarg.0` object uses.

**A base-typed PARAMETER position widens to a reference interface when
substitutability is LIVE** (58, 61). Exported so a consumer can name it,
unexported method so only this module satisfies it. Returns and property getters
keep the concrete pointer; a property setter's value is a parameter position. A
base-typed return leaves a downcast Go cannot express, which is recorded as a
language limitation rather than worked around.

**An inherited member whose projection is a PACKAGE FUNCTION is not
re-projected** (58). A static, or a generic instance method: its Go identity
already names its declaring type, and its receiver-first parameter carries the
reference interface.

**A route CNA offers for a PROJECTED member and CNA-Go does not bind is
recorded, with the measured reason** (57). Four classes, checked against the
canonical headers and against the manifest.

**A BCL type at a public signature position takes the standard-library Go
type whose ROLE it is, chosen from what the profile's positions MEASURABLY do
with it** (53, 69). `System.IO.Stream` is `io.Reader` because every stream
position in the profile is read, with the two written ones named in a registry;
`System.Text.StringBuilder` is `*strings.Builder` because all four of its
positions are inputs whose only reads are `get_Length` and `get_Chars`. This is
not the same rule as `bclSignatureAdapters`, which reproduces a BCL type's own
measured member inventory: an adapter is needed when a projected member RETURNS
the BCL object and a consumer must use its surface.

**A native route whose XNA member is projected is bound only if it AGREES with
the member's reference body** (57, 69). Foundation 69 added the fifth class,
`REDUNDANT_READ`, and the first entry justified by a measured numeric
disagreement rather than by a shape: `cna_sprite_font_measure_utf8` adds the
last glyph's right bearing unclamped where `InternalMeasure` clamps it.

**A substitutable-base position is a PARAMETER position** (73). The
substitutable-base rule widens a base-typed parameter to an exported reference
interface; a return and a read-only getter keep the concrete pointer. The
verifier now counts only `parameter:*`, `property-set` and `field-type` when it
computes a projected carrier's LIVE positions, which is that rule as it was
already written down rather than a new one.

**A handle's OWNERSHIP KIND is recorded, never inferred from the handle** (73).
`GraphicsDevice` is the one type with two kinds -- BORROWED from a Game, and
OWNED from the public constructor -- and a destroyed OWNED device must report
itself disposed rather than falling back to the borrowed path and answering as
the running game's device.

**A COMPOSED base whose consumers span PACKAGES moves to an internal package**
(78). The adapter must stay unexported, and eight exception types in four
packages cannot reach an unexported framework type. `internal/` keeps it
unreachable from outside the module, which is the property the unexported field
had.

**A reference interface whose implementors span packages exposes ONE language
accessor, admitted by name** (78). Go has no way for one package to satisfy
another package's unexported method. The accessor is registered on the base
adapter, admitted only on the adapter's own type and on types that compose that
base, and it must record a reason -- and it keeps the interface closed because
its result type is declared in an internal package.

**A DECLARED member blocked by an external BCL closure is recorded and COUNTED,
never dropped** (78). `blockedDeclaredMembers` is the declared-member
counterpart of a base adapter's `BCL_PROJECTION_BLOCKED_EXTERNAL` exclusion. The
identity must exist in the pinned contract, the entry must state its kind, needs
and reason, and `BLOCKED_DECLARED_MEMBERS` is reported separately from
`MISSING_MEMBER` so it can never read as surface that is present. The pinned
3243 declared projections is checked as `declared + blocked`.

**A CLR EXCEPTION OBJECT and a Go operation error are DIFFERENT CONTRACTS**
(76). Foundation 29 framed this as a dilemma -- either every settled
per-operation fallibility decision reopens, or the eight derived types are
inert. Neither is what the projection does: an operation reports failure through
a Go error exactly as before, and System.Exception is a value the profile
constructs and passes. The type therefore has no Error() method, because the
pinned CLR type declares none.

**A base whose DERIVED TYPES are the point widens at RETURNS as well as at
parameters** (76). The substitutable-base rule leaves a base-typed return as the
concrete pointer and records the lost downcast. For an exception hierarchy that
would erase which of the eight kinds a consumer chained, so System.Exception
widens at every position and the downcast is a Go type assertion.

**The pinned .NET Framework 4.0 mscorlib is a retained, hash-admitted message
authority** (76). Every "read from the pinned mscorlib" claim named its sha256
and nothing could check one; the binary is now retained at
`~/deps/bcl-4.0-pinned` and `tools/resource_strings` verifies BCL messages
against it, refusing a binary whose hash does not match.

**A CLR interface a class declares that Go cannot put on the class has a
REGISTERED CARRIER, compiler-checked** (75). The cross-package rule relocates a
class member whose type belongs to a descendant package; an INTERFACE member has
nowhere to move to, because its signature is part of the interface. The
conformance is then carried by a named adapter in the interface's own package --
the object the class really publishes -- and `crossPackageInterfaceCarriers`
records the pair, names the exact blocking member, and makes the verifier check
that the carrier satisfies the interface. A class that fails without an entry is
still `INTERFACE_MAPPING_MISMATCH`.

**A private helper whose every operand is a descendant-package type lives in
that package** (75). `GraphicsDeviceManager`'s AddDevices and its device
comparer are `private` in the reference and are projected in the Graphics
package, reached through `internal/servicebridge`. What stays on the manager is
what the reference DECLARES there: FindBestDevice, RankDevices, CanResetDevice,
the raise helper and the event.

**A reference defect is reproduced, not corrected** (75).
`GraphicsDeviceInformation::set_Adapter` guards on `this.adapter` rather than on
`value`, so assigning null succeeds and the refusal fires only when the current
adapter is already null. Correcting it would make CNA-Go refuse an assignment
the reference accepts, which is a different API.

**An exclusion of a base member states WHICH KIND of exclusion it is** (74).
`NOT_PUBLIC_SURFACE` -- a constructor, a `family` member, a private explicit
implementation -- claims nothing is missing. Every other kind claims something
public IS missing and must name the exact external closure it waits on, which
the verifier counts separately and enforces. An admission of a hole is not a
permission to have one.

**A dead guard in a BCL base is not projected as a failure mode, and the death
is written down** (26, 74). `Collection<T>`'s `items.IsReadOnly` and
`Dictionary<K,V>`'s null-key `ThrowArgumentNullException` are both unreachable
for every consumer in the profile, the second because `System.String` projects
to Go `string`, which has no null. Each is recorded in the adapter registry so
a future consumer that could reach it reopens the question rather than
inheriting the omission.

**An `out` parameter is APPENDED to the declared return, not prepended** (74).
`Dictionary<K,V>::TryGetValue` is the profile's first member with both, so the
settled `remove-input-and-append-return` rule finally has an observable
consequence: the projection is `(bool, string)` and is deliberately not
reshuffled into Go's value-then-ok idiom.

**ROADMAP.md's scoreboard is generated-or-validated, never remembered** (74).
One marked block of `KEY VALUE` lines, checked against the generated reports by
a test that is itself proved with planted stale numbers. Historical evidence
documents quote the scoreboard of the milestone they record and are deliberately
out of the guard's scope.

**Every missing type belongs to exactly one frontier family, and every family
names what it is stopped on** (74). `tools/api_compat/frontier.go` partitions
the live missing-type set; `GLOBAL_UNREVIEWED` is assigned by the exhaustiveness
check rather than written by hand, so it can only be nonzero because something
was forgotten.

**A graphics state object freezes** (59). Every setter is fallible because
`ThrowIfBound` is a refusal the reference really throws; every getter is one
`ldfld`. The static presets are frozen from construction.

## Deferred families

This section is a STATEMENT OF CURRENT STATE, not a permanent list, and it is
rewritten whenever a milestone makes one of its entries false.

Still deferred: audio/XACT; media/video; storage; touch; design;
GamerServices; and broad platform work. Every one of them is now classified in
`tools/api_compat/frontier.go`, which partitions the live missing-type set and
generates `docs/generated/remaining-work.md`; that generated table is the
authority for what each family is stopped on, and this paragraph is its
summary. The 3D family is no longer deferred as
a whole: Foundation 71 projected TextureCube and Texture3D and Foundation 72
the whole Effect cluster, so what remains under it is the six stock effects,
EffectMaterial and the Model family. Windows, macOS, Android, iOS, and
Web/Wasm are unqualified even if the Go compiler can target them.

`ROADMAP.md` carries the scoreboard -- validated against the generated reports
rather than typed by hand since Foundation 74 -- and points at
`docs/generated/remaining-work.md` for the per-family breakdown, with the CNA
route counts each family has to work with. Those two are the files to read when
selecting the next milestone; this section is the summary.

No longer deferred, and the milestone that ended each:

```text
Dictionary`2   Foundation 74 composed it, which was five missing Go spellings
               and one measured external blocker rather than the six subsystem
               blockers Foundation 29 recorded. LaunchParameters and
               Game::LaunchParameters followed.
Content/XNB    Foundation 63 projected ContentManager over CNA's own content
               pipeline, and Foundation 69 grew its closed Load<T> set to a
               second asset kind. LZX remains CNA's business, not the
               binding's: CNA-Go parses no container.
SpriteFont     Foundation 69.
effects        Foundation 72 projected Effect, EffectTechnique(Collection),
               EffectPass(Collection), EffectParameter(Collection) and
               EffectAnnotation(Collection), and with them removed the
               EFFECT_DEPENDENCY that had blocked every GraphicsDevice draw.
cube/volume    Foundation 71 projected TextureCube and Texture3D. Their
               RENDERER support is a separate question and is measured per
               artifact rather than claimed.
GraphicsDevice Foundation 73 closed the type: Reset, Present,
               GetBackBufferData, the render-target members, RenderTargetCube,
               RenderTargetBinding and the public constructor. It also
               produced the profile's first VERIFIED_PIXEL evidence -- the
               SOFTWARE artifact reads the back buffer back and every texel
               equals the colour it was cleared to.
```

The Load<T> set is closed by PROJECTED TYPE IDENTITY, which is the mechanism
that keeps the two facts above from drifting apart: CNA has native loaders for
`SoundEffect`, `TextureCube` and `Effect` today, and each is absent from the
set because CNA-Go projects no Go type for it. Each is a missing TYPE, not a
missing loader, and each becomes actionable the milestone its type does.

**A base whose derived types override a RETURNING member widens at returns**
(79). Foundation 76 made the first exception to "a base-typed return keeps the
concrete pointer", for the exception hierarchy. Effect is the second, and its
case is stronger: `Clone` is declared to return `Effect` and all five stock
effects override it to return their own class, and the composed base half is
unreachable from outside its package — so the downcast is not lost, it is
impossible. `returnWideningBases` records the closed set, and a name in it that
is not also in `substitutableBases` is a verifier failure: a return cannot widen
to an interface no parameter position declares.

**A composition hook is installed only for a virtual Go cannot already
dispatch** (79). `Effect` declares `Clone` and `OnApply` virtual and only
`OnApply` gets one. `Clone` is a member of the widened reference interface, so a
consumer holding a derived value reaches the derived body through Go's own
method set; `OnApply` is reached through the base object `EffectPass` holds in a
field, where nothing in the language recovers it. The first draft hooked both,
and the planted-defect run is what found the second hook unkillable — a branch
no production path executes.

**A COMPOSED middle link holds no state and may still have SITES** (79). The
object-identity registry used to require a link that forwards its binding to
record no derived field, no self accessor AND no identity site. Those are two
claims, not one: `Effect` forwards and holds nothing, and two of its own members
still report a CLR runtime type. A middle link with sites names the accessor it
reaches through the link it forwards to, and the verifier checks that it does
not declare one of its own.

**A projection keeps the reference's MANAGED state even when the runtime offers
to hold it** (79). CNA models a whole stock effect natively, and forwarding every
property to it would have been shorter. It would also have made fourteen
infallible members fallible, contradicting interface signatures measured three
years of milestones earlier, and would have let CNA's clamping answer getters
whose reference bodies are one `ldfld`. The managed state stays, the reference's
dirty flags stay, and `OnApply` pushes — which is the reference's own shape with
a different push target.

**A control is what turns a native success into evidence** (79). The BasicEffect
draw succeeds on both artifacts, and the control taken immediately before it
also succeeds — so the success is NOT evidence that applying the effect unblocked
the draw. Without the control the twenty successes would have read as a result
they are not.

**A shared body across a family needs the family's IL to AGREE, member by
member** (80). The five stock effects' `set_World` is 23 bytes in every one of
them, and the flag it ORs is not the same: BasicEffect raises
WorldViewProj|World|Fog, and the two unlit effects raise WorldViewProj|Fog,
because they have no world parameter and no eye position. Each type therefore
declares its own accessors with its own measured word, and only the write that
carries no flags -- the three-matrix push -- is shared. The test holds both
halves of that table, so a later collapse into one body fails.

**An inherited surface is the PUBLIC surface, and that decides what an interface
may require** (80). Effect declares `OnApply` `protected internal`; a derived
type that OVERRIDES it projects it, and one that does not -- EffectMaterial --
has no such member at all. `EffectReference` dropped `OnApply` for that reason:
an interface member would have required a member the pinned contract does not
give that type. The same rule already decides that a derived effect has one
`Dispose()` where Effect has two.

**A composition hook is installed only where there is a body to dispatch to**
(80). EffectMaterial calls `bindDerived`, which carries the CLR `this` for
ToString, and NOT `bindDerivedEffect`, which installs an override -- it
overrides nothing. Its Clone answers an Effect, exactly as the reference's does,
and the native run asserts that positively rather than only asserting it is not
an EffectMaterial.

**An EXPLICIT interface implementation is a witness, and its refusal may have
nowhere to go** (81). EnvironmentMapEffect and SkinnedEffect implement
`IEffectLights::LightingEnabled` explicitly -- a two-byte getter that always
answers true and a 51-byte setter that throws for false -- so the pinned
contract lists no such property on either type and both accessors are interface
witnesses. The setter's refusal cannot be reported: `IEffectLights`' measured
contract makes the accessor INFALLIBLE, because every other implementor's is one
`stfld`. The projection therefore answers the way the reference does for the
legal value and records the refusal it drops, rather than widening a signature
the interface pins. It is the one place in the stock effects where a projected
member cannot carry a refusal the reference makes.

**A substitutable base goes live the milestone a POSITION appears, not the
milestone a derived type does** (81). TextureCube was composed in Foundation 71
and had a projected derived type from Foundation 73, and its requirement stayed
LATENT for ten milestones because nothing in the profile named a TextureCube at
a parameter position. `EnvironmentMapEffect::EnvironmentMap`'s setter is the
only one there is, and projecting that effect is what made the requirement live
-- which the verifier reported rather than the author noticing.

**A retention CNA takes is measured through the member that reveals it** (81).
Assigning a TextureCube to an EnvironmentMapEffect makes
`cna_texturecube_destroy` answer "The TextureCube is retained by an
EffectParameter", where XNA's Dispose on the same texture is legal. The stress
asserts the refusal AND asserts that clearing the property releases it, so the
divergence is held from both sides instead of being avoided by disposing in a
convenient order.

**An assertion over a CONSTANT, a repeated call, or a coincident input holds
nothing** (81). Five of this milestone's twenty-five planted defects survived a
first pass, and every one of them was the test's fault. A setter's effect was
checked through a getter that returns `ldc.i4.1`; a native release was checked
by repeating a dispose CNA documents as idempotent; a reordering was checked on
a path that refuses before reaching either statement; and a correction that
forces `M44 = 1` was checked over identity input whose M44 is already 1. The
lesson is one question to ask of every assertion: *if the thing I am claiming
were absent, would this line change?*

**A STATIC member may still need a running game, and saying so beats answering
nothing** (82). `FrameworkDispatcher::Update` and `TitleContainer::OpenStream`
are static and need nothing in the reference; both CNA routes take a game
handle, and the canonical header says why -- "a game handle is taken here only
for thread affinity". So both projections refuse outside a game where the
reference would have worked. The alternative was doing nothing and reporting
success, which would let a consumer believe media and audio had been pumped
when they had not.

**A route that both sizes and copies reports BUFFER_TOO_SMALL as its size
answer** (82). Every other size query in this binding is a separate `_size`
route that succeeds; `cna_title_container_read_ext` sizes and copies through one
entry point, and a zero capacity answers CNA_RESULT_BUFFER_TOO_SMALL with the
byte count filled in. Treating that as a failure made a file that had just been
written report as not found, and only the device-backed run could find it --
which is what the native stress is for.

**Two assertions can contradict each other through the code under test** (83).
`OcclusionQuery.IsComplete` arms `Begin` as its first statement, so a scenario
that checked "a query inside its own pair is not complete" and then "a second
Begin is refused" made the first assertion break the second -- on correct code.
The claims cannot share a Begin/End pair. Where an observation MUTATES, the
order of assertions is part of the test's meaning.

**A physical assumption about a backend is a hypothesis, not a premise** (83).
"A query inside its own pair cannot be complete" is true of XNA's Direct3D 9
GetData and false of CNA, whose route answers whether A result can be read
rather than whether THIS one is ready. The scenario records both outcomes in
separate counters instead of asserting one, and the two qualified artifacts
land in different columns.

**An `assembly` interface the reference DISPATCHES ON is projected unexported**
(84). `IDynamicGraphicsResource` is not in the pinned contract, and leaving it
out lost a real behaviour: every `CopyData` ends its setting path with
`ldarg.0; isinst IDynamicGraphicsResource; ... SetContentLost(false)`, so a
successful upload clears a dynamic resource's content-lost latch. Exporting it
would have added a type the contract does not declare. The rule is the same one
the `clrTypeName` identity mechanism follows: an internal member the reference
needs is projected with an unexported name, and only the members the reference
actually dispatches on are projected at all.

**A CLR enum converter is measured, never assumed to be an equality** (84).
`ConvertXnaSetDataOptionsToDx` tests bit 0, then bit 1, and returns zero
otherwise, so `Discard|NoOverwrite` is Discard and an undefined value is mapped
rather than refused. CNA numbers its three options identically and answers
`INVALID_ARGUMENT` for an undefined one, so a cast would have been right for the
three named values and wrong for everything else -- refusing where the reference
silently accepts, which is the direction that breaks working consumer code. A
shared numbering is a coincidence to be checked, and the CHECK is the converter's
IL rather than the enum's declaration.

**A flag with no getter is observed through the REFUSAL it changes** (84).
Nothing in the contract reports whether a buffer was created dynamic, and
`IsContentLost` answers false either way, so the flag looked unobservable and the
scenario counted a refused upload as a renderer capability -- which is exactly
what a non-dynamic buffer produces. A probe showed CNA refuses a non-None
`SetDataOptions` on a static buffer and accepts every option on a dynamic one, so
the refusal IS the flag's signature: the scenario now fails on it, and the
mutation that never sets the flag dies. Before concluding a native argument is
unobservable, look for the call it makes fail.

**A milestone may raise EVIDENCE STRENGTH rather than the type count** (85).
Foundation 85 added no type and bound no route: it turned every draw proof since
Foundation 60 from `VERIFIED_NATIVE_DRAW` into `VERIFIED_PIXEL` by checking the
texels a known material produces. A whole CLASS of defect that Foundation 80
had called unobservable -- "the only observable is what the shader draws" -- is
now scored for BasicEffect, and deleting a push from `OnApply` dies immediately.
When a family's unkilled mutations all name the same missing observable,
building that observable is a milestone.

**Record the boundary the new observable has, not just the claims it enables**
(85). The SOFTWARE artifact evaluates a FLAT MATERIAL: `DiffuseColor` and
`Alpha` reach the texels, and `VertexColorEnabled` and `EnableDefaultLighting`
both cross to CNA and change nothing. Writing that down is what stops a later
milestone from spending itself on an assertion the renderer cannot satisfy.

**A one-sided pixel claim proves nothing** (85). "The counter-clockwise triangle
drew nothing" is satisfied by a renderer that never draws at all. The claim is
made with the SAME three corners in the opposite winding order, through the same
effect and the same state, so one fills the buffer and one leaves it untouched.
A full-screen triangle has the same weakness against a CLEAR, which is why the
geometry claim uses a half-screen one and checks the corners that must stay
marked.

**Scoring a mutation needs the artifact that can SEE it** (85). Eleven pixel
defects scored against HEADLESS would all have survived, because HEADLESS
refuses the readback and the slice skips. The harness pins the SOFTWARE library
rather than taking the ambient one, and the parent accounting pins the slice's
refusal count to the back buffer's so a skipped claim cannot look like a passing
one.

**A stale blocker is a measurement waiting to be redone** (86). The audio
family carried "the qualification artifact pins a NULL audio renderer, so
nothing behind it would play" long after it stopped being true; the generated
runtime-capabilities report already contradicted it. Before treating a recorded
blocker as a reason to defer a family, re-run the measurement that produced it.

**A route with no faithful call site does not get bound, even when it is the
obvious one** (86). `cna_audio_get_capabilities` was bound end to end and then
reverted: XNA never probes for audio hardware, it maps the error code the
creation call returns, so an up-front probe would be a call the reference does
not make. The measurement it produced is kept in the evidence; the binding is
not.

**An EQUIVALENT mutant is a finding, not a test gap** (90). Widening the second
`case` of a Go `switch` cannot change behaviour, because Go takes the first
matching case and a value the first case matched never reaches the second. The
switch's own semantics were already enforcing the exclusivity the reference's
`else if` does. Two of milestone 90's survivors were of this kind; the answer is
to replace the mutation with a defect that IS distinguishable -- two independent
`if`s -- and to record why the first one could never have been killed, rather
than to weaken a test until something fails.

**A survivor can mean the code is UNREACHABLE from where the test sits** (90).
Three planted defects in the live-list source survived every Model test, and the
reason was structural: `ModelEffectCollection` HIDES `GetEnumerator` and
forwards only some inherited members, so the base's iterator and half its bounds
checks cannot be reached through that family at all. The fix is to test the
adapter from its own package, not to route a Model test through machinery no
consumer can reach.

**A test that reads the same constant as the code follows it** (91). The
selector sentinel test passed `framework.PlayerIndex(storageNoPlayer)`, so a
mutation that changed the constant moved the test with it and survived. The
literal is written out now, with a separate assertion that the constant IS 255.
This is the value half of the anchor rule below: an anchor must not match prose,
and an expectation must not be computed from the thing it checks.

**A mutation anchor that can match PROSE is not a mutation** (86). This project
quotes reference bodies in doc comments, so `return nameHash ^ idHash` appears
twice in one file -- once in the comment and once in the code. Replacing the
first occurrence scored a defect as unkillable while the projection was
untouched. Anchor on surrounding code, and treat an unexpected survivor as a
harness bug before treating it as a finding.

**A round trip that writes back what it read proves nothing** (89). Both mouse
window-handle mutations -- dropping the write, and answering a constant --
survived a stress check that read the handle, assigned it back and compared.
The artifact starts with no hooked window, so the value is `0` and a dropped
write reads back the same `0` an honest one does. The same shape had already
bitten a managed test that assigned ONE value to both DisplayWidth and
DisplayHeight. A state round trip must carry a value the state did not already
hold, and where the live value cannot be varied safely, a sentinel written and
then withdrawn is the way to do it.

**Reproduce the reference's FLOAT WIDTH, not its formula** (87). XNA's
`GetSampleSizeInBytes` computes its scale factor in float32, so
`(float)44100 / 1000f` is 44.099998474121094 and one second truncates to 44099
samples rather than 44100. Doing the same arithmetic in float64 gives a
different byte count at 44100 and 22050 and the same one at 8000 and 48000 --
so a test that used only the exact rates would have agreed with the wrong
projection.

**A mutation of a path that never executes is not killed, and the fix is to
make the path reachable** (87). CNA reports only its three defined sound states,
so `get_State`'s fallback branch never ran and a mutation of it survived
correctly. Extracting the mapping into its own function let a test reach the
branch. Extracting is the answer; scoring it as unfalsifiable would have been
giving up on a claim that was one refactor away.

**Two members that agree everywhere except one state need that state** (87).
Resume and Play produce the same observable on a playing, paused or fresh
instance; they differ only on a STOPPED one, where Resume leaves it stopped. The
assertion is a Stop-then-Resume pair, and without it a projection that routed
Resume to the play route passed every other transport check.

**A native scenario that can make sound must make SILENCE** (87). The qualified
artifacts open a real playback device, so the audio fixtures are all-zero PCM.
Structure -- creation, lifetime, transport, state -- is exercised identically and
nothing audible reaches the machine running the suite. What silence cannot prove
is audibility, and the scenario says so rather than implying it.

**A conversion CNA also offers is not automatically CNA's to answer** (88). The
four audio sample conversions were bound and then reverted: one second at
22050Hz mono is 44098 bytes to the reference, whose scale factor is float32, and
44100 to CNA, which computes it exactly. Both answer the same question and
differ by one frame. When the reference computes something itself, binding a
route that computes it differently is a divergence dressed as a binding.

**Choose the fixture value whose answer is DISTINCTIVE** (88). The same
scenario at 8000Hz would have passed against the wrong projection, because
8000/1000 is exact in binary32 and both sides answer 16000 there. A fixture is
evidence only when a wrong implementation would give a different number, and
picking the rate that separates them is part of writing the assertion.

**A counter can exist to be ZERO** (88). MICROPHONE_CAPTURE_CALLS is asserted to
be zero by the parent accounting, and the run FAILS on any other value: the
suite projects Microphone.Start and GetData and calls neither, because starting
capture opens a real recording device on whatever machine the suite runs on. An
assertion that something did not happen is worth as much as one that it did,
and it is the only way to keep a constraint from eroding.

**A contract FIELD is projected as a field, and both consequences are recorded**
(88). Microphone.Name is `kind: field`, so it is an exported Go field: the
`initonly` half has no counterpart, and a field cannot carry a nil guard, so
reading it through a nil pointer panics where every method answers. Hiding it
behind a getter would have projected a property the contract does not declare,
which is the larger divergence -- so both limitations are written down and
pinned by a test.

## Next milestone selection rule

After Foundation 16, regenerate the scoreboard and dependency graph before
selecting the next closure or batch. Do not automatically choose a consumer of
any completed enum, continue GraphicsDeviceManager or GraphicsDevice, infer
runtime support from a managed enum, or combine independent families.

A multi-type batch is selected the same way as a single closure: rank the
dependency-complete missing nodes, then consume only candidates that satisfy
every safety condition, skipping and recording any that do not rather than
stopping the batch.

The dependency-complete node count is measured with direct interfaces included
as public-signature dependencies. That rule is authoritative; a historical
count taken without interface edges differed by exactly one node
(`Microsoft.Xna.Framework.GameComponent`, completed in Foundation 32) and never
changed the ranking.

From Foundation 33 the ranking has a second gate. A missing type whose CLR base
is another type in the profile is NOT selectable while that base is DEFERRED in
`xnaBaseRelationships`, however dependency-complete its own signature looks:
its inherited public surface is unprojectable, so completing it would report a
whole surface that is not there. Forty-one of the missing types are in that
state, and every one of them waits on the same architecture decision.
