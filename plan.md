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

## Deferred families

Content/XNB and LZX; effects, models, and 3D; audio/XACT; media/video; storage;
touch; design; GamerServices; and broad platform work are not
Foundation 1 count-filling opportunities. Windows, macOS, Android, iOS, and
Web/Wasm are unqualified even if the Go compiler can target them.

## Next milestone selection rule

After Foundation 13, regenerate the scoreboard and dependency graph before
selecting one next closure. Do not automatically choose a ButtonState,
GraphicsProfile, DepthFormat, or SurfaceFormat consumer, continue
GraphicsDeviceManager or GraphicsDevice, infer runtime support from a managed
enum, or combine independent families.

The dependency-complete node count is measured with direct interfaces included
as public-signature dependencies. That rule is authoritative; a historical
count taken without interface edges differs by exactly one node
(`Microsoft.Xna.Framework.GameComponent`) and never changes the ranking.
