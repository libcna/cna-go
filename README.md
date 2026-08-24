# CNA-Go

> **Status:** early, measured binding foundation — functional for the qualified
> Foundation 1 runtime and Foundation 2/3/4/5/6/7/8/9/10 managed/API closures, far from full
> XNA compatibility.

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
3,243 expected Go members. The current target has 62 types and 1,353 members:
57 types are complete, five native/runtime types are partial, and 195 are
missing. The strict verifier remains red because most XNA surface is
intentionally absent. Every mismatch, leak, allowlist, and unmeasured-category
gate is green.

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

The admitted qualification artifact uses CNA ABI 0.7.0, the HEADLESS renderer,
and NULL audio. Native draw execution is proven, but visible rendering is not.
Windows, macOS, Android, iOS, and Web/Wasm are not qualified. Content/XNB,
Effects/3D, Audio, Media, Storage, Touch, and most of XNA remain unimplemented.

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
surface exists; its 372 missing-surface diagnostics are the work queue, not a
compatibility claim.
The native ABI and stress commands require the qualified native environment.

The normative rules are in [plan.md](plan.md), the language projection in
[docs/xna-go-mapping.md](docs/xna-go-mapping.md), the native boundary in
[docs/native-abi.md](docs/native-abi.md), and the resumable handoff in
[NEXT.md](NEXT.md).

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
