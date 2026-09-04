# CNA-Go roadmap — what is done, and what remains

This file is the answer to "what is left?". It is written from **measured**
state, not from intent: every count below comes from a tool in this repository,
and every claim about what CNA can do comes from the canonical headers at
`~/deps/cna-c-abi-0.21.0/include/CNA/C/`.

Regenerate the numbers before trusting them:

```sh
go run ./tools/api_compat            # the scoreboard, docs/generated/, and the frontier
go run ./tools/native_abi -headers ~/deps/cna-c-abi-0.21.0/include \
                          -library ~/deps/cna-c-abi-0.21.0/libcna_c_api.so
go run ./tools/external_consumer -source .
```

## Scoreboard

<!-- cna-go:scoreboard -->
```text
TOTAL_DIAGNOSTICS               75
MISSING_TYPE                    75
MISSING_MEMBER                   0
COMPLETE_TYPES                 182
PARTIAL_TYPES                    0
UNEXPECTED_MEMBER                0
ALLOWLIST_ENTRIES                0
GLOBAL_ACTIONABLE_LOCAL         75
GLOBAL_UNREVIEWED                0
BOUND_FUNCTIONS                304
MANIFEST_LAYOUT_AGREEMENTS     457
ABI_MISMATCHES                   0
```
<!-- /cna-go:scoreboard -->

**This block is not maintained by hand.** Every line is checked against
`docs/generated/api-compat-report.json` and
`docs/generated/native-abi-report.json` by
`TestRoadmapScoreboardMatchesTheGeneratedReports`, which fails if a number here
disagrees with the last generated run, if a key is missing, or if a key nobody
generates is added. Update the reports, then copy their values here.

## Where the project stands

The **whole 2D graphics path is closed and proved against a live renderer**:
`GraphicsDevice`, `SpriteBatch`, `Effect` and its eight companion types,
`Texture2D`/`TextureCube`/`Texture3D`, `RenderTarget2D`/`RenderTargetCube`, the
four state objects, the vertex/index buffers, `SpriteFont`, `ContentManager`,
the `Game` loop and `GraphicsDeviceManager`.

Foundation 73 produced the first **`VERIFIED_PIXEL`** evidence in the project:
on the SOFTWARE artifact, a cleared back buffer reads back and every texel
equals the clear colour, 20 cycles out of 20. Everything before it was
`VERIFIED_NATIVE_DRAW` — CNA accepted the submission and nothing could read the
result.

Foundation 74 composed `System.Collections.Generic.Dictionary<K,V>` and with it
projected `LaunchParameters` and `Game.LaunchParameters`. It is also the
milestone that made this file's numbers generated rather than remembered.

Foundation 75 projected `GraphicsDeviceInformation` and
`PreparingDeviceSettingsEventArgs` and closed **`GraphicsDeviceManager`**: the
device enumeration, the eight-step ranking policy, `CanResetDevice` and the
`PreparingDeviceSettings` event are all projected.

Foundation 76 projected `System.Exception` and closed `Game`. Foundation 77
projected the four stock vertex structs, and the native stress run submits all
four to CNA on both the HEADLESS and SOFTWARE artifacts, 220 user-primitive
draws with no refusal. Foundation 78 composed `System.Exception` and
`ExternalException` and projected all eight XNA exception types.

Foundation 79 composed **`Effect`** — the sixth composed XNA base and the second
whose RETURNS widen — and projected `BasicEffect`, `DirectionalLight` and
`IEffectLights`. The design turned on one measurement: CNA's stock BasicEffect
publishes **no EffectParameters** on either qualified artifact, so the
reference's push target does not exist and the push goes into CNA's own
stock-effect state instead. The reference's managed state and dirty flags are
kept, because forwarding every property to CNA would have made fourteen
infallible members fallible and contradicted interface signatures Foundation 18
measured from the same assembly.

Foundation 80 closed **`AlphaTestEffect`**, **`DualTextureEffect`** and
**`EffectMaterial`** on that shape, and measured the place it does not
generalise: the two unlit effects' `set_World` and `set_View` raise TWO
dirty-flag bits where BasicEffect's raise three, so a shared accessor body would
have been wrong and each type declares its own.

Foundation 81 closed **`EnvironmentMapEffect`** and **`SkinnedEffect`**, and
with them the whole stock-effect family: all six of `Effect`'s derived types are
projected. Both implement `IEffectLights::LightingEnabled` EXPLICITLY, so the
pinned contract lists no such property on either and both accessors are
interface witnesses; `TextureCube` became the fourth substitutable base, because
`EnvironmentMapEffect::EnvironmentMap` is the only TextureCube-typed parameter
position in the profile.

Foundation 82 closed the last two **root types**, `FrameworkDispatcher` and
`TitleContainer`. Both are static in the reference and both CNA routes take a
game handle for thread affinity, so both refuse outside a running game --
recorded rather than silently succeeding. `TitleContainer.OpenStream` carries
CNA's documented narrowing: that ABI has no stream handle for title content and
delivers the whole file instead, so the returned reader is over bytes already in
memory.

Foundation 83 closed **`OcclusionQuery`** and ran the probe the dynamic buffers
were waiting on. The hypothesis holds: CNA models a dynamic buffer through the
SAME routes as a static one, with a `dynamic` flag in the create info, an
`is_content_lost` field in the info snapshot, subscribe/unsubscribe routes for
the `ContentLost` event, and `SetDataOptions` in the transfer descriptors. CNA
documents the content-loss state as "currently always false", so `IsContentLost`
will always answer false and the event will never fire — a limitation to record
rather than a defect. What remains is composing the `VertexBuffer` and
`IndexBuffer` bases.

## No partial types, and no missing members

**`PARTIAL_TYPES` and `MISSING_MEMBER` are both zero.** Every type CNA-Go
projects, it projects completely; what remains is 83 types it does not project
at all, and every one of them is classified.

## What is left, and why

The per-family breakdown lives in **`docs/generated/remaining-work.md`**, which
is generated from the frontier registry in `tools/api_compat/frontier.go`. That
registry partitions the live missing-type set: every missing type belongs to
exactly one family, every family carries a classification, and a family that
names no blocker or claims a type that is no longer missing is a verifier
failure. `GLOBAL_UNREVIEWED` counts the missing types nobody has classified,
and it is zero.

The suggested order is dependency order, which is the order the families appear
in that registry:

1. **`SoundEffect` / `SoundEffectInstance`** and the audio family, which also
   grow `ContentManager.Load<T>`'s closed set. The stock effects are done, so
   upgrading `DrawUser*` from `VERIFIED_NATIVE_DRAW` to `VERIFIED_PIXEL` is now
   unblocked and not yet done: it needs the SOFTWARE back-buffer readback taken
   with a known material, which any of the six effects can now supply.
2. **The Model family**, which depends on the stock effects and now has them.
3. Then **Input**, **Storage**, **Media/XACT**, the **content plumbing** and
   the **Design converters**.

**The dynamic-buffer note, resolved.** Foundation 83 probed it against the
canonical headers and the hypothesis was right: `CNA_VertexBufferCreateInfo`
carries a `dynamic` flag documented as "True to construct DynamicVertexBuffer",
the info snapshot carries `dynamic` and `is_content_lost`, both buffers have
`subscribe_content_lost`/`unsubscribe_content_lost`, and the transfer
descriptors carry `SetDataOptions` whose non-None values "require a supported
dynamic-buffer overload". The creation flag and the info fields were already
bound in Foundation 65 and 66. What remains is composing the two bases and
binding the options-carrying uploads and the two subscription pairs.

## Two decisions that were open, and where they now stand

1. **CLR attributes.** The five `ContentSerializer*Attribute` types have no Go
   counterpart for *application*, but the question of whether the TYPES can
   exist is separate from whether Go can attach them to declarations, and only
   the second is a candidate language limitation. The frontier registry records
   the family as `ACTIONABLE_LOCAL` for that reason.
2. **`System.ComponentModel.TypeConverter`.** All 13 `Design` converters derive
   from it. It is not in the profile, so it needs the same treatment
   `Dictionary<K,V>` got in Foundation 74: the minimal measured closure its IL
   actually reaches, and nothing more.

## The gates every milestone must pass

Unchanged from Foundation 44 onward, and all of them must be green:

```sh
go test ./...                                    # unit + verifier tests
go run ./tools/behavior                          # behaviour corpus
go run ./tools/api_compat                        # strict API + inventory + frontier
go run ./tools/native_abi -headers ... -library ...   # ABI_MISMATCHES must be 0
go run ./tools/external_consumer -source .       # must actually COMPILE
DISPLAY=:77 CNA_NATIVE_LIBRARY=.../libcna_c_api.so go run ./tools/native_stress
```

Three rules that are easy to lose and expensive to relearn:

- **The external consumer is a real gate.** It compiles the public surface from
  outside the module with `GOWORK=off`. A name that only works inside the
  repository has not shipped.
- **No speculative route binding.** Every bound CNA route needs a production
  call site *today*, and `tools/native_abi/reachability_test.go` enforces the
  whole chain. A route with no caller belongs in the deliberately-unbound
  registry with its measured reason.
- **ABI prototypes come from the canonical headers**, compiler-backed via
  `tools/native_abi/testdata/probe.c`. Never from a manifest alone.

## Environment

- **Never build in the scratchpad or `/tmp`.** Use `build/`, `build-probe/`,
  `build-consumer/` — the closed list in `openeggbert/CLAUDE.md`.
- **Every native run needs a virtual display.** `DISPLAY=:77` (Xvfb). The
  OpenGL artifacts open real windows otherwise.
- Qualified artifacts: `~/deps/cna-c-abi-0.21.0` (HEADLESS), `-software`,
  `-opengl33`, `-opengles3-fx`. The first two are what the gates run on.
- The Go toolchain this repository is qualified with is `~/deps/go1.24.4`.
