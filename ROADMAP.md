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
TOTAL_DIAGNOSTICS               86
MISSING_TYPE                    86
MISSING_MEMBER                   0
COMPLETE_TYPES                 171
PARTIAL_TYPES                    0
UNEXPECTED_MEMBER                0
ALLOWLIST_ENTRIES                0
GLOBAL_ACTIONABLE_LOCAL         86
GLOBAL_UNREVIEWED                0
BOUND_FUNCTIONS                230
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

## No partial types, and no missing members

**`PARTIAL_TYPES` and `MISSING_MEMBER` are both zero.** Every type CNA-Go
projects, it projects completely; what remains is 98 types it does not project
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

1. **The stock effects**, which compose the already complete `Effect` and give
   `DrawUser*` an independently predictable output colour — the one thing
   standing between the vertex structs and `VERIFIED_PIXEL` evidence for a
   drawn triangle.
2. **`SoundEffect` / `SoundEffectInstance`**, which also grow
   `ContentManager.Load<T>`'s closed set.
3. **The Model family**, then **Input**, **Storage**, **Media/XACT**, the
   **content plumbing** and the **Design converters**.

**The dynamic-buffer note.** `DynamicVertexBuffer` and `DynamicIndexBuffer`
have **no dedicated CNA routes**. They are almost certainly the existing
vertex/index buffer routes with a `BufferUsage`, plus a `ContentLost` event —
but that is a hypothesis, and the rule in `plan.md` is that a route is bound
only when it AGREES with the member's reference body. Probe before projecting.

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
