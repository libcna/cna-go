# CNA-Go normative implementation plan

**Selected profile:** Microsoft XNA Framework 4.0 Windows runtime

**Native contract:** canonical CNA C ABI, exact experimental version 0.7.0

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

## Deferred families

Content/XNB and LZX; effects, models, and 3D; audio/XACT; media/video; storage;
touch; design; GamerServices; and broad platform work are not
Foundation 1 count-filling opportunities. Windows, macOS, Android, iOS, and
Web/Wasm are unqualified even if the Go compiler can target them.

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
count taken without interface edges differs by exactly one node
(`Microsoft.Xna.Framework.GameComponent`) and never changes the ranking.
