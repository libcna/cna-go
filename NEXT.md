# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 14 are complete. Milestone 14 is the first
multi-type autonomous batch, `PURE_MANAGED_BATCH_A`. It completes exactly 25
public XNA types carrying 121 mapped Go identities, every one an ordinary or
flags enum whose only public-signature dependency is `System.Int32`.

Foundation 13 was committed and synchronized at
`9dbc4dfafd80c4065d0533d0483ee8efcd52672e` before this milestone began, so
Foundation 14 started from a clean `develop` worktree. Foundation 14 is fully
qualified and is committed as exactly one batch commit. Preserve the
established namespace, enum, interop, and measured-absence rules.

## Foundation 14 batch

The 25 completed types, in the order they were consumed:

```text
 1 Graphics.RenderTargetUsage      3    14 Graphics.FillMode              2
 2 Graphics.CubeMapFace            6    15 Media.MediaSourceType          2
 3 Audio.AudioChannels             2    16 Audio.SoundState               3
 4 Audio.AudioStopOptions          2    17 Graphics.CullMode              3
 5 Graphics.IndexElementSize       2    18 Graphics.GraphicsDeviceStatus  3
 6 Graphics.SetDataOptions         3    19 Graphics.TextureAddressMode    3
 7 Media.MediaState                3    20 Input.GamePadDeadZone          3
 8 Graphics.EffectParameterClass   5    21 Media.VideoSoundtrackType      3
 9 Graphics.CompareFunction        8    22 Graphics.PresentInterval       4
10 Graphics.EffectParameterType   10    23 Graphics.PrimitiveType         4
11 Input.Touch.GestureType        11    24 Input.Touch.TouchLocationState 4
12 Input.Buttons                  25    25 Graphics.BlendFunction         5
13 Audio.MicrophoneState           2
```

`SetDataOptions`, `GestureType`, and `Buttons` are flags enums and carry the
`// xna:flags` directive; the other 22 are ordinary enums and must not.
`AudioChannels` and `Buttons` declare no zero literal, so their Go zero value
is an ordinary undefined raw value that equals no named constant.

Three namespaces gained their first Go package: `Audio`, `Media`, and nested
`Input/Touch`. Each package path is the deterministic result of the established
`packagePathForNamespace` rule and carries only a `doc.go` and enum files.

## Structural scoreboard

The pinned contract remains 257 types / 2,964 members at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The formal mapping remains 257 expected Go types / 3,243 members.

```text
TARGET_TYPES=90
TARGET_MEMBERS=1482
TOTAL_DIAGNOSTICS=344
MISSING_TYPE=167
MISSING_MEMBER=177
COMPLETE_TYPES=85
PARTIAL_TYPES=5
MISSING_TYPES=167

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. All 25 batch closures report `PASS` with 121 target
Go identities and zero local diagnostics.

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game` (39)
- `Microsoft.Xna.Framework.GraphicsDeviceManager` (40)
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` (70)
- `Microsoft.Xna.Framework.Graphics.SpriteBatch` (16)
- `Microsoft.Xna.Framework.Graphics.Texture2D` (12)

Their combined missing-member count remains 177. `CubeMapFace`,
`GraphicsDeviceStatus`, and `PrimitiveType` have `GraphicsDevice` among their
deferred reverse consumers; no `GraphicsDevice` member was added.

## Behavior and verifier evidence

The behavior corpus has 411 observations, 411 assertions, and zero failures,
grown from 280. Each completed type contributes a `PURE_XNA_DERIVED` complete
raw-value table and a `System.Int32` storage fact, plus `GO_LANGUAGE_PROJECTION`
zero-value, arbitrary-positive-raw, and negative-raw facts. The three flags
enums add two more `PURE_XNA_DERIVED` facts each: the exact literal union
(`SetDataOptions`=3, `GestureType`=1023, `Buttons`=2145451007) and that every
non-zero literal is a distinct single bit.

The verifier gained one table-driven category, `foundation14EnumClosures`,
covering all 25 types rather than 25 near-identical bespoke closures. The table
is admitted against the pinned contract by `TestFoundation14EnumMappedContracts`.
`TestFoundation14EnumDefectsRejectedForEveryType` applies 14 structural defects
to every applicable type — 304 negative cases, each with an asserted clean
baseline. The declared mutation inventory grew from 157 to 177 cases. Each
owning package carries a source-level `flagsDirectiveAt` self-test plus
`TestFlagsDirectiveDetector`, the negative fixture for the detector itself.
Manual allowlists and unmeasured categories are zero. No new general mapping
rule was needed, so `docs/xna-go-mapping.md` is unchanged.

ButtonState, GraphicsProfile, DepthFormat, SurfaceFormat, ClearOptions,
BufferUsage, DisplayOrientation/SupportedOrientations, PlayerIndex/Keyboard,
VertexElement, and the 262,400-pattern PackedVector sweep remain green.

## Scope and native preservation

Completing these enums proves managed metadata only. No render target, cube
map, primitive draw, render state, sampler, buffer, presentation, device-loss,
effect, audio, microphone, media, video, touch panel, or game pad capability is
claimed. `go run ./tools/capabilities --check` still reports
`RUNTIME_CAPABILITIES=38 STATUS=PASS` and `docs/runtime-capabilities.json` is
unmodified.

No CNA source, C binding, cgo route, native mirror constant, layout, or
callback was added. ABI measurements remain 23 bound functions, 67 prototype
type positions, 96 C/Go measurements, 28 layouts, two callbacks, and five
constants with zero missing symbols or mismatches.

Native race stress retains 20 Game, recreation, texture, SpriteBatch,
callback-error, and callback-panic cycles, one wrong-thread check, one owner
retry, 80 GC points, and zero crash/UAF/double-free observations.
`GO_RACE_STATUS=PASS`; native sanitizers remain `NOT_RUN`.

The unchanged maintained `cna-go-template` remains clean at
`65254848d9fac02ace934db3879106834bafca97`; test, vet, race, build, trimpath,
and exact 60/600 Draw gates pass.

The exact ABI-0.7 library binary admitted in Foundation 11
(`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`) is no
longer present on the development machine. Foundation 14 verified the ABI and
the stress suite against the shared ABI-0.7 build at
`~/deps/cna-c-abi-0.7.0/libcna_c_api.so`
(`c62949d23d3745964f5e557a06665875621ed4cb6e2930e3f282afd5911f2dcb`), which
reproduces every count, the full function list, and every stress counter
exactly. `docs/generated/native-abi-report.json` still records the
Foundation-11 hash and was not rewritten.

## Re-running gates

Use Go 1.24.4, Linux amd64, cgo, GCC, and an exact admitted ABI-0.7 library:

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict   # expected 344 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
git diff --check
```

Strict mode rewrites `docs/generated/api-compat-report.json` with
`"mode": "strict"`; rerun `--mode report` last so the committed evidence keeps
its report mode.

Verify the native ABI and stress counters out of tree with an explicit
`-output` under the scratchpad so a locally rebuilt ABI-0.7 library or an
absolute developer header path never rewrites the committed evidence.

Source-artifact qualification archives the exact worktree, not bare HEAD, and
uses a fresh extracted consumer with `GOWORK=off` and `-buildvcs=false`.
Archive hash and isolated consumer results belong in the external final handoff
so the archive does not recursively describe its own hash.

## Worktree provenance

Foundation 14 started on clean `develop` with `HEAD` and `origin/develop` both
at `9dbc4dfafd80c4065d0533d0483ee8efcd52672e`, the committed and synchronized
Foundation-13 milestone. The whole batch was accumulated without per-type
commits and landed as exactly one commit. History was not rewritten.

## Dependency-graph counting rule

Dependency completeness counts a node's base type, its declared direct
interfaces, and every member signature type as public-signature dependencies.
BCL types covered by the mapping table are satisfied; the five partial runtime
types count as present. Regenerating the Foundation-13 baseline under this rule
reproduces exactly 71 dependency-complete missing nodes, matching the published
Foundation-13 handoff. The count is now 55.

The historical "71/72" nuance is not reopened.
`Microsoft.Xna.Framework.GameComponent` remains dependency-incomplete because
its declared direct interfaces `IGameComponent` and `IUpdateable` are still
missing. The interface-inclusive rule is authoritative and never changes the
ranking.

The batch unlocked genuinely new dependency-complete nodes, including
`Input.GamePadButtons` (from `Buttons`), `Input.Touch.TouchLocation` and
`GestureSample` (from `TouchLocationState` and `GestureType`),
`Graphics.PresentationParameters` (from `RenderTargetUsage` and
`PresentInterval`), `Media.MediaSource` and `Media.Video` (from
`MediaSourceType` and `VideoSoundtrackType`), and
`Graphics.IGraphicsDeviceService`.

## Remaining safe pure-managed candidates

Five pure-managed enums remain dependency-complete and safe under the same
rules, all at reverse fan-out 1: `ColorWriteChannels` (flags, 6 literals),
`StencilOperation` (8), `TextureFilter` (9), `GamePadType` (10), and `Blend`
(13) — 46 further identities. They were not skipped for cause; the 25-type
ceiling was reached first. Eight value structs are also dependency-complete
with no unmapped BCL dependency: `TouchPanelCapabilities`, `RendererDetail`,
`GestureSample`, `GamePadThumbSticks`, `GamePadTriggers`, `GamePadDPad`,
`MouseState`, and `GamePadButtons`.

## One next selected boundary

The highest raw reverse fan-out among the 55 dependency-complete nodes still
belongs to `Design.MathTypeConverter` (12), which remains excluded on the
established ground that `System.ComponentModel.ExpandableObjectConverter`,
`ITypeDescriptorContext`, and `PropertyDescriptorCollection` are outside the
BCL mapping table.

The next selected boundary is therefore the highest genuinely reachable node:

```text
Microsoft.Xna.Framework.Graphics.GraphicsResource
```

`GraphicsResource` has reverse fan-out 11 and is the direct CLR base of
`BlendState`, `DepthStencilState`, `Effect`, `IndexBuffer`, `OcclusionQuery`,
`RasterizerState`, `SamplerState`, `SpriteBatch`, `Texture`, `VertexBuffer`,
and `VertexDeclaration`. Every remaining Graphics resource type is gated behind
it, including the already-partial `SpriteBatch` and the `Texture` base of
`Texture2D`.

It is deliberately not another pure-managed leaf. Its nine source members are
`ToString`, `Dispose()`, `Dispose(Boolean)`, `Finalize`, `IsDisposed`, `Tag`,
`Name`, `GraphicsDevice`, and the `Disposing` event, and it declares
`System.IDisposable`. Completing it therefore requires the deferred ownership
and lifetime architecture: a Go disposal projection, a finalizer decision, a
disposal event projection, and the `GraphicsDevice` property, which touches one
of the five partial runtime types. That architecture is the real next
milestone, not a count-filling opportunity.

Selection authorizes nothing else. None of the eleven derived types is
authorized, no renderer, GPU, texture, buffer, effect, or presentation route is
authorized, and the five remaining safe enums are not authorized as part of it.
Its pinned metadata and disposal semantics must be independently inspected in
the next milestone.

```text
SELECTED_ONLY=true
STARTED=false
FOUNDATION_MILESTONE_14_COMPLETE=true
BATCH_NAME=PURE_MANAGED_BATCH_A
```
