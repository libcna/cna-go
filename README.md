# CNA-Go

> **Status:** early, measured binding foundation — functional for the qualified
> Foundation 1 runtime and Foundation 2/3/4/5/6/7/8/9/10/11/12/13 managed/API
> closures, far from full XNA compatibility.

CNA-Go maps Microsoft XNA Framework 4.0 namespaces to Go import paths and
executes native-backed APIs only through CNA's canonical C ABI:

```text
Go game
   ↓
Microsoft/Xna/Framework[/Graphics|Input|Content]
   ↓
internal/interop (the only cgo/native boundary)
   ↓
CNA C ABI 0.7.0
```

There is deliberately no invented public `CNA/Framework` layer. Pure XNA
values are implemented as Go values; public packages expose neither C types
nor native handles.

## What is qualified

The current structural scoreboard maps the authoritative XNA 4.0 Windows
runtime profile (257 types and 2,964 members) to 257 expected Go types and
3,255 expected Go members — 3,243 projected from XNA-declared members plus 12
projected from the public surface a supported BCL base contributes. The current
target has 122 types and 1,791 members: 117 types are complete, five
native/runtime types are partial, and 135 are missing. The strict verifier
remains red because most XNA surface is intentionally absent. Every mismatch,
leak, allowlist, and unmeasured-category gate is green.

Forty-one of the missing types inherit from another type in the profile whose
base relationship is deferred; they are recorded with classified blockers rather
than left silent, and none of them is selectable until XNA-to-XNA class
inheritance is decided.

Foundation 1 qualifies on **Linux amd64 desktop with cgo**:

- real CNA-driven `Game` lifecycle and tick-exact `GameTime`;
- locked owner OS thread, generation checks, `runtime/cgo.Handle` callbacks,
  and contained callback errors/panics;
- real GraphicsDeviceManager/device, native viewport and clear;
- PNG `Texture2D` creation from `io.Reader`;
- one exact real scaled SpriteBatch draw route;
- native keyboard snapshots with exact XNA `Keys` values;
- a complete first managed closure measured by the verifier and behavior
  corpus.

Foundation 2 additionally qualifies, as managed Go with no ABI expansion:

- complete `Vector2`, `Vector3`, `Vector4`, `Quaternion`, and `Matrix` values;
- the recursive `Plane`, `Ray`, bounding box/sphere/frustum, and containment
  dependency closure;
- explicit nullable `(value, hasValue)` intersection results;
- complete `Color`, including its 141 predefined static colors;
- managed `Viewport.Project` and `Viewport.Unproject`;
- a 93-observation PURE_XNA_DERIVED exact-bit behavior corpus.

Foundation 3 qualifies the complete managed Curve family with no ABI expansion:

- reference-class identity for `Curve`, `CurveKey`, and `CurveKeyCollection`;
- exact key constructors, equality, hash, clone, and `Single.CompareTo` order;
- sorted reference collection semantics, shallow clone, mapped index failures,
  and versioned fail-fast iteration;
- XNA binary32 tangents, Hermite/Step evaluation, all five loop modes, and
  negative-cycle behavior;
- a 142-observation `PURE_XNA_DERIVED` corpus with zero failures.

Foundation 4 qualifies the complete managed PackedVector family with no ABI
expansion:

- both managed interfaces, including general `!0 -> TPacked` substitution and
  `IPackedVectorOfTPacked[TPacked]` generic identity;
- exact pointer-method-set conformance for all seventeen mutable value structs;
- 25 formally measured explicit-interface witness methods without member-count
  inflation or allowlists;
- XNA-exact UNorm, SNorm, raw byte/short, and non-IEEE half packing behavior;
- 262,400 exhaustive packed-pattern round trips and a 201-observation
  `PURE_XNA_DERIVED` corpus with zero failures.

Foundation 5 qualifies the complete managed vertex-element descriptor closure
with no ABI expansion:

- `VertexElementFormat` and `VertexElementUsage` as exact non-flags `int32`
  enums, including undefined CLR enum values;
- `VertexElement` as a private-state Go value struct with exact mutable-property
  projection, zero-value and copy semantics;
- only `Equals(Object)`—no invented typed equality overload—and both exact
  mapped operator identities;
- XNA `SmartGetHashCode` word-XOR/fallback behavior and exact descriptor string
  formatting;
- a 227-observation `PURE_XNA_DERIVED` corpus with zero failures.

Foundation 6 completes the root PlayerIndex enum and Keyboard surface without
ABI expansion:

- exact non-flags `int32` values `One=0`, `Two=1`, `Three=2`, and `Four=3`;
- arbitrary raw enum values remain representable and are never validated;
- `KeyboardGetStateByPlayerIndex(framework.PlayerIndex)` uses the same process
  keyboard route and runtime/error requirements as `KeyboardGetStateByNone`;
- direct XNA IL proves the player argument is never read;
- Keyboard moves from partial to complete, with a 234-observation corpus and
  zero failures.

Foundation 7 completes DisplayOrientation and one exact managed
GraphicsDeviceManager property slice without ABI expansion:

- `[Flags]` `DisplayOrientation` uses exact explicit `int32` values 0, 1, 2,
  and 4 while preserving combinations and unknown raw bits;
- `SupportedOrientations()` and `SetSupportedOrientations(DisplayOrientation)`
  store managed configuration with no synthetic error or native call;
- direct XNA IL proves constructor defaults and that every setter stores the
  exact value and marks private device state dirty, including same-value sets;
- GraphicsDeviceManager remains partial with 40 missing members, and the
  managed corpus reaches 242 observations with zero failures.

Foundation 8 completes exactly the managed Graphics `BufferUsage` enum without
ABI expansion:

- exact `[Flags]` named-`int32` metadata with explicit `None=0` and
  `WriteOnly=1` constants;
- the synthetic CLR `value__` field is excluded, leaving exactly two mapped Go
  identities;
- zero value, arbitrary signed raw bits, and typed bitwise composition remain
  available without validation or helper API;
- no buffer type, buffer operation, GraphicsDevice member, or GPU capability is
  added, and the corpus reaches 247 observations with zero failures.

Foundation 9 completes exactly the managed Graphics `ClearOptions` enum
without ABI expansion:

- exact `[Flags]` named-`int32` metadata with explicit `Target=1`,
  `DepthBuffer=2`, and `Stencil=4` constants;
- the synthetic CLR `value__` field is excluded, leaving exactly three mapped
  Go identities;
- XNA declares no named zero member, so raw zero remains representable but
  unnamed—there is no invented `None`, `Default`, or `All` constant;
- combinations, unknown bits, negative values, OR, and AND remain available
  without validation or helper API;
- no GraphicsDevice.Clear overload or native clear behavior is added, and the
  corpus reaches 256 observations with zero failures.

Foundation 10 completes exactly the managed Graphics `SurfaceFormat` enum
contract without ABI expansion:

- exact non-flags named-`int32` metadata with all 20 explicit literals from
  `Color=0` through `HdrBlendable=19`;
- the synthetic CLR `value__` field is excluded, so the pinned contract's 21
  field identities map to exactly 20 Go identities;
- zero value equals `Color`, while arbitrary positive and negative raw values
  remain representable without validation;
- no `iota`, flags marker, Stringer/helper surface, or PackedVector dependency
  is added;
- no pixel-format support, texture/render-target consumer, GPU/native format
  mapping, or CNA ABI route is claimed, and the corpus reaches 262 observations
  with zero failures.

Foundation 11 completes exactly the managed Graphics `DepthFormat` enum
contract without ABI expansion:

- exact non-flags named-`int32` metadata with `None=0`, `Depth16=1`,
  `Depth24=2`, and `Depth24Stencil8=3`;
- the synthetic CLR `value__` field is excluded, so five source identities
  map to exactly four Go identities;
- zero value equals `None`, while arbitrary positive and negative raw values
  remain representable without validation;
- no `iota`, flags marker, Stringer/helper surface, SurfaceFormat conversion,
  consumer API, GPU/native depth format mapping, or CNA ABI route is added;
- no actual depth/stencil surface support is claimed, and the corpus reaches
  268 observations with zero failures.

Foundation 12 completes exactly the managed Graphics `GraphicsProfile` enum
contract without ABI expansion:

- exact non-flags named-`int32` metadata with `Reach=0` and `HiDef=1`;
- the synthetic CLR `value__` field is excluded, so three source identities map
  to exactly two Go identities;
- zero value equals `Reach`, while arbitrary positive and negative raw values
  remain representable without validation;
- no `iota`, flags marker, Stringer/helper surface, consumer API, GPU/native
  profile mapping, or CNA ABI route is added;
- `Reach` and `HiDef` are metadata values, not hardware capability claims: no
  profile selection, feature-level detection, or renderer support is claimed,
  and the corpus reaches 274 observations with zero failures.

Foundation 13 completes exactly the managed Input `ButtonState` enum contract
without ABI expansion:

- exact non-flags named-`int32` metadata with `Released=0` and `Pressed=1`;
- the synthetic CLR `value__` field is excluded, so three source identities map
  to exactly two Go identities;
- zero value equals `Released`, while arbitrary positive and negative raw
  values remain representable without validation;
- no `iota`, flags marker, Stringer/helper surface, consumer API, input backend
  or native enum mapping, or CNA ABI route is added;
- `Released` and `Pressed` are metadata values, not hardware capability claims:
  no mouse, gamepad, D-pad, or button-polling behavior is claimed, and the
  corpus reaches 280 observations with zero failures.

See [geometry and transform evidence](docs/geometry-transform-evidence.md) for
the computed closure, mapping decisions, conventions, and local strict-zero
matrix.

See [Curve evidence](docs/curve-evidence.md) for the collection-language
projection, class/clone semantics, binary32 interpolation, and local matrix.

See [PackedVector evidence](docs/packed-vector-evidence.md) for generic and
interface projection, bit layouts, rounding/non-finite rules, half semantics,
exhaustive sweeps, and the 19-type local strict-zero matrix.

See [VertexElement evidence](docs/vertex-element-evidence.md) for the exact
three-type closure, property expansion, undefined-enum behavior, hash/string
fixtures, and local strict-zero matrix.

See [PlayerIndex and Keyboard evidence](docs/player-index-keyboard-evidence.md)
for the enum table, direct unused-argument IL proof, shared runtime route, and
local strict-zero matrix.

See [DisplayOrientation evidence](docs/display-orientation-evidence.md) for the
flags contract, constructor/getter/setter IL, private dirty-state behavior,
lifecycle evidence, and selected property-slice measurement.

See [BufferUsage evidence](docs/buffer-usage-evidence.md) for the exact
one-type closure, named-int32 flags projection, raw-bit qualification, focused
verifier negatives, and explicit buffer/runtime scope boundary.

See [ClearOptions evidence](docs/clear-options-evidence.md) for the exact
one-type closure, unnamed-zero flags edge case, bitwise qualification, focused
verifier negatives, and explicit GraphicsDevice/native scope boundary.

See [SurfaceFormat evidence](docs/surface-format-evidence.md) for the complete
20-literal raw table, corrected source/mapped identity arithmetic, non-flags
projection, focused verifier negatives, and strict runtime-support boundary.

See [DepthFormat evidence](docs/depth-format-evidence.md) for the exact
four-literal raw table, non-flags projection, focused verifier negatives,
deferred reverse dependents, and strict runtime-support boundary.

See [GraphicsProfile evidence](docs/graphics-profile-evidence.md) for the
exact two-literal raw table, non-flags projection, focused verifier negatives,
the four deferred reverse dependents, and the strict metadata-only Reach/HiDef
boundary.

See [ButtonState evidence](docs/button-state-evidence.md) for the exact
two-literal raw table, non-flags projection, focused verifier negatives, the
three deferred `System.ValueType` reverse consumers, and the strict
metadata-only input boundary.

See [Foundation 14 pure-managed batch A evidence](docs/foundation-14-pure-managed-batch-evidence.md)
for the 25 completed enums and their 121 mapped identities, the per-type pinned
raw tables and deferred reverse consumers, the table-driven verifier closure
category, the 304 exhaustive negative cases, the ranked skip list with exact
reasons, and the strict no-capability-inflation boundary for the new `Audio`,
`Media`, and `Input/Touch` namespaces.

See [Foundation 15 pure-managed batch B evidence](docs/foundation-15-pure-managed-batch-evidence.md)
for the last five leaf enums, the GamePad/Mouse/Touch value-struct clusters and
their IL-derived clamping, hashing, and formatting rules, the deliberate
`TouchLocation` equality asymmetry, the zero-synthetic-error measurement, and
the re-verified native ABI provenance.

See [Foundation 16 GamePadState evidence](docs/foundation-16-game-pad-state-evidence.md)
for the managed XInput packing, the IndependentAxes dead-zone reproduction, the
measured bit-for-bit agreement with the pinned `Buttons` literals, and why the
safe pure-managed seam was exhausted at the old mapping rules.

See [Foundation 17 managed class evidence](docs/foundation-17-managed-class-evidence.md)
for the general pure-managed CLR class rule, per-operation fallibility and its
accessor-level keys, the bit-exact `FlipHandedness` involution, the
negative-zero constructor defaults, the exact `bge.un.s` validation that accepts
NaN, and the 32 new negative fixtures. `AudioListener` and `AudioEmitter` are
pure managed descriptors: completing them claims no audio runtime capability
and creates no XACT state.

The admitted qualification artifact uses CNA ABI 0.7.0, the HEADLESS renderer,
and NULL audio. Native draw execution is proven, but visible rendering is not.
Windows, macOS, Android, iOS, and Web/Wasm are not qualified. Content/XNB,
Effects/3D, Audio, Media, Storage, Touch, and most of XNA remain unimplemented.
See [Foundation 18 interface evidence](docs/foundation-18-interface-evidence.md)
for the managed-interface projection rule, the measured `IEffectFog` split in
which only `FogColor` reaches D3DX, the two runtime-boundary contracts, the
separate Boolean and error channels of `BeginDraw`, and the exact event-mapping
gap that keeps `IUpdateable`, `IDrawable`, and `IGraphicsDeviceService`
deferred. `IUpdateable`, `IDrawable` and `IGameComponent` became live in
Foundations 30-32: `Game` keeps ordered lists of every `IUpdateable` and
`IDrawable` in `Components`, and `GameComponent` is a shipped implementor. The
other contracts remain declarations only: CNA-Go has no effect runtime and no
device manager bound to `IGraphicsDeviceManager`.

See [Foundation 19 IntPtr and PresentationParameters evidence](docs/foundation-19-intptr-presentation-parameters-evidence.md)
for the `System.IntPtr` to `uintptr` projection and exactly what it does not
authorize, the narrowed `RAW_HANDLE_LEAK` rule with its two-clean and ten-leak
fixtures, the complete descriptor contract, the `IsFullScreen` constructor
quirk, and why XNA 4.0 has no `Clear` here. `PresentationParameters` is a
descriptor, not a device: it stores a platform window handle and creates,
resets, enumerates, and presents nothing.

See [Foundation 20 touch collection evidence](docs/foundation-20-touch-collection-evidence.md)
for why `TouchCollection` was reachable after all, the first cluster that is at
once a CLR value type and fallible, the unconditional `NotSupportedException`
write side, `CopyTo`'s 64-bit overflow arithmetic, the operator-versus-`Equals`
search asymmetry, and the cursor's behavior at both ends. Completing it claims
no touch capability: CNA-Go has no `TouchPanel` and reads no device.

See [Foundation 21 service container evidence](docs/foundation-21-game-service-container-evidence.md)
for the general rule that a BCL interface whose members the XNA type already
declares publicly adds no projected surface, the duplicate-before-assignability
check order, and why a missing or absent service is an absence rather than a
failure. As of Foundation 30, `Game` exposes it and hands back one stable
container per Game; nothing in the binding registers into it, because the
reference's only registrar is `GraphicsDeviceManager` and CNA-Go's partial one
satisfies neither service contract.

Foundations 30 through 33 qualify the managed Game component slice, with no
ABI expansion and no CNA change:

- `Game.Components` and `Game.Services`, each one stable managed identity;
- the component engine the reference keeps around them: two incrementally
  ordered derived lists, a pending-initialization queue, and the collection and
  order-changed handlers that maintain them;
- five explicit base-call functions, so an override decides whether and when
  base behavior runs, exactly as `base.Update(t)` does in CLR;
- complete `GameComponent`, satisfying `IGameComponent` and `IUpdateable` under
  a compiler-checked conformance rule;
- 34 external-consumer tests proving the whole loop from outside the module.

Nothing there renders. Component `Update` and `Draw` are called on the owner
thread with a tick-exact `GameTime`; what a component does there is its own.

See [Foundation 30 Game managed state evidence](docs/foundation-30-game-managed-state-evidence.md)
for why `Game::get_Components` and `::get_Services` are seven bytes of `ldfld`
each and therefore belong in Go rather than behind the C ABI, the component
engine derived from IL rather than from memory — Game does **not** sort its
components, both order comparers return 0 only for reference identity, and ties
are stable through an explicit forward walk rather than a stable sort — and the
`inRun` guard that decides whether an added component is queued or initialized
on the spot.

See [Foundation 31 base-call evidence](docs/foundation-31-game-base-call-evidence.md)
for why base behavior is never automatic: in CLR the override decides whether
and when the base runs, so CNA-Go supplies five package-level `GameBase...`
functions a callback may call, or not, or call in a different place. They are
measured language support and add nothing to any XNA identity counter. The
native callback order is audited against XNA's there.

See [Foundation 32 GameComponent evidence](docs/foundation-32-game-component-evidence.md)
for the class the engine was built for, the four load-bearing quirks it must
keep — including `On...` methods that ignore their sender and raise with `this`,
which is what makes the engine work — the `TryLock` projection of a reentrant
CLR `Monitor`, and the new general rule that a complete projected class must
satisfy every XNA interface its metadata declares, checked by `go/types`.

See [Foundation 33 XNA base frontier evidence](docs/foundation-33-xna-base-frontier-evidence.md)
for the second base frontier, which was silent until now: `Texture2D` inherits
nine public members from `Texture` and `GraphicsResource` that CNA-Go does not
project. Twelve relationships over 41 derived types are recorded with classified
blockers, and no derived type of a deferred XNA base may be reported complete.
That decision — XNA-to-XNA class inheritance — is the next architectural one.

See [Foundation 34 game event bridge evidence](docs/foundation-34-game-event-bridge-evidence.md)
for `Game.Activated`, `Deactivated`, `Exiting` and `Disposed`, bound to CNA
signals that were already published and had never been reached from Go. No CNA
C++ changed and CNA was not rebuilt; the ABI counters moved because CNA-Go binds
more of the same unchanged binary. `OnExiting` raises with a **null** sender
where its two siblings raise with the Game, and that one IL instruction is
preserved. Exactly one native subscription per event per Game, because CNA
invokes multiple registrations on one event in reverse order. `Deactivated` is
structurally complete but recorded as `NOT_RUN_ENVIRONMENT`: a HEADLESS artifact
has no window manager and can never lose focus.

See [Foundation 35 frame hook evidence](docs/foundation-35-game-frame-hook-evidence.md)
for `Game.BeginRun`, `EndRun`, `BeginDraw` and `EndDraw`. `BeginDraw`'s Boolean
stays a value channel separate from the error, because `DrawFrame` runs
`if (BeginDraw()) { Draw(); EndDraw(); }` and a false answer skips both. CNA's
four canonical frame hooks correspond position for position and are deliberately
**not** installed: base behavior is never automatic, so forwarding them would
prejudge an override design that has not been made. `GameCallbacks` still has
exactly five members.

See [Foundation 36 signal registry evidence](docs/foundation-36-signal-registry-evidence.md)
for the two registries that turn those decisions into measured facts: four
signals with three raise sites and one honest runtime deferral, and four frame
hooks with a measured zero installed and four recorded reasons behind it.

The `Media` package contains enum metadata only and carries no media runtime
capability claim. The `Input/Touch` package adds `TouchLocation`,
`GestureSample`, and the read-only `TouchCollection` alongside its enums, and
still carries no touch runtime capability claim: nothing there polls a panel,
reads a device, or recognizes a gesture. The `Audio` package adds two pure
managed positional descriptors alongside its enums and still carries no audio
runtime capability claim: nothing there opens a device, creates XACT state, or
plays a sound.

See the generated [runtime capability inventory](docs/generated/runtime-capabilities.md)
for evidence and limitations by capability.

## Native runtime

The Go build uses cgo but does not link a developer CNA build at compile time.
Supply an exact CNA C ABI 0.7 shared library at runtime:

```sh
export CNA_NATIVE_LIBRARY=/absolute/path/to/libcna_c_api.so
```

The override must be an absolute regular-file path. Without it, the Linux
loader searches for `libcna_c_api.so`. Wrong ABI versions and missing required
symbols fail before Game creation. CNA-Go contains no checkout-relative native
library fallback and does not distribute CNA binaries.

## Development and verification

The maintained sibling `cna-go-template` uses a `go.work` file for local
development. A published module version is not claimed. Final consumer
qualification instead extracts the audited CNA-Go source archive and uses a
temporary `replace` to that exact source tree.

Useful gates are:

```sh
go test ./...
go vet ./...
go test -race ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library /absolute/path/to/libcna_c_api.so
go run ./tools/native_stress
```

Normal structural strict mode is expected to exit nonzero until all mapped XNA
surface exists; its 295 missing-surface diagnostics are the work queue, not a
compatibility claim.
The native ABI and stress commands require the qualified native environment.

The normative rules are in [plan.md](plan.md), the language projection in
[docs/xna-go-mapping.md](docs/xna-go-mapping.md), the native boundary in
[docs/native-abi.md](docs/native-abi.md), and the resumable handoff in
[NEXT.md](NEXT.md).

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
