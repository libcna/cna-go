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
TOTAL_DIAGNOSTICS               51
MISSING_TYPE                    51
MISSING_MEMBER                   0
COMPLETE_TYPES                 206
PARTIAL_TYPES                    0
UNEXPECTED_MEMBER                0
ALLOWLIST_ENTRIES                0
GLOBAL_ACTIONABLE_LOCAL         51
GLOBAL_UNREVIEWED                0
BOUND_FUNCTIONS                350
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
the `ContentLost` event, and `SetDataOptions` in the transfer descriptors.

Foundation 84 closed **`DynamicVertexBuffer`** and **`DynamicIndexBuffer`**, and
with them the whole `Graphics` namespace. Three measurements decided it and none
was in the plan:

- **A successful `SetData` clears the content-lost latch.** Every `CopyData` in
  the family ends its setting path with
  `ldarg.0; isinst IDynamicGraphicsResource; ... SetContentLost(false)`, after
  the result check. `IDynamicGraphicsResource` is `assembly` and not in the
  contract, so it is projected as an unexported interface with the one member
  the reference dispatches on — and `RenderTarget2D` and `RenderTargetCube`,
  which had carried the latch and the event since Foundations 58 and 73 with
  nothing able to clear or raise them, now carry it too.
- **`SetDataOptions` is converted by a BIT TEST.** `ConvertXnaSetDataOptionsToDx`
  tests bit 0, then bit 1, and returns zero otherwise, so `Discard|NoOverwrite`
  is Discard and an undefined value is mapped rather than refused. CNA refuses an
  undefined option by name, so handing it the caller's raw value would refuse
  where the reference accepts.
- **The `dynamic` flag has exactly one observable.** Nothing in the contract
  reports it and `IsContentLost` answers false either way; CNA refuses a non-None
  `SetDataOptions` on a buffer created static, so a refusal on a buffer created
  dynamic is a defect rather than a capability, and the native scenario treats it
  that way.

`IsContentLost` still answers false on both qualified artifacts and the
`ContentLost` event still cannot fire there, because CNA documents the state as
"currently always false" — a limitation recorded rather than a defect, and the
boundary a renderer that can lose a device would move.

Foundation 85 raised the STRENGTH of evidence rather than the type count: the
first **`VERIFIED_PIXEL`** draw. Every draw proof from Foundation 60 to 84 was
`VERIFIED_NATIVE_DRAW` — CNA accepted the submission and nothing could read the
result back. The SOFTWARE artifact reads the back buffer and a `BasicEffect`
with lighting, texturing, fog and vertex colour off is a known solid material,
so the texels can now be checked:

- The **default `RasterizerState` culls a counter-clockwise triangle**, proved
  two-sided with the same three corners in the opposite winding order.
- The constructor's `DiffuseColor` default, `Vector3.One`, is seen as white.
- A half-screen triangle covers 192080 of 384000 texels with a measured corner
  pattern, so the GEOMETRY reaches the rasteriser rather than the draw acting
  as a clear.
- `DiffuseColor` decides the texel, and `Alpha` premultiplies it into both the
  colour and the alpha channel: `(0,1,0)` at `Alpha 0.5` comes back
  `(0,127,0,127)`.
- `VertexColorEnabled` and `EnableDefaultLighting` both reach CNA and this
  renderer ignores both, which is recorded in two counters each rather than
  asserted. The SOFTWARE artifact evaluates a FLAT MATERIAL -- `DiffuseColor`
  and `Alpha` -- and nothing per-vertex or per-light, which is the boundary any
  later pixel claim has to be written against.

Eleven planted defects, all killed, in a class no earlier suite could score:
deleting `BasicEffectSetDiffuseColor` from `OnApply` was invisible to the whole
project before this milestone. Foundation 80 named this in advance -- its two
unkilled push defects were "the same boundary that makes `VERIFIED_PIXEL` the
next thing worth building" -- and the boundary has moved for `BasicEffect` only,
so those two remain unkilled and are now killable in principle. Extending the
pixel slice to the other five stock effects is the obvious next lever.

HEADLESS has no readback path and records a refusal, pinned by the parent
accounting to the back buffer's own refusal count so the slice cannot quietly
stop running where it should.

Foundation 86 closed **`RendererDetail`**, the first type whose authority is the
Xact assembly rather than `Microsoft.Xna.Framework.dll` -- two strings and five
members with no native dependency, so projecting it needs no audio engine, no
bank and no device. Its `GetHashCode` needed the pinned mscorlib string hash,
which moved from the framework package into **`internal/bclhash`** so two
projected namespaces can share one body without adding public surface.

It also corrected a **stale blocker**. `xnaBaseRelationships` recorded that "the
qualification artifact pins a NULL audio renderer, so nothing behind it would
play"; `cna_audio_get_capabilities` reports `is_playback_available = TRUE` on
BOTH qualified artifacts, which `docs/generated/runtime-capabilities.md` had
already measured from a direct C probe. The audio family's remaining blocker is
CNA-Go's own surface and nothing else.

The capability route was bound end to end during that milestone and then
**reverted**: reading `Helpers::GetExceptionFromErrorCode` settled that XNA does
not probe for audio hardware at all -- it makes the native call and maps the
returned error code, and `NoAudioHardwareException` comes out of that switch. A
route with no faithful call site does not get bound, so the measurement is kept
and the binding is not.

Foundation 87 closed **`SoundEffect`** and **`SoundEffectInstance`** over 22
bound routes. They landed together because the pinned contract declares NO
public constructor for the instance: one comes from `CreateInstance` or from
`Play`'s pool and nowhere else.

The measurement that decided it is that **`GetSampleSizeInBytes` does not return
round numbers**. Its scale factor is computed in float32, and `(float)44100 /
1000f` is 44.099998474121094 -- so one second at 44.1kHz mono truncates to 44099
samples and **88198** bytes, not 88200. At 8000Hz and 48000Hz, whose thousandths
ARE representable, the counts are exact. At 44100 STEREO the round number comes
back, and not because the truncation went away: 44099 is odd, so
`samples % Channels` adds one sample back -- which is what makes the alignment
step an addition rather than a round-up.

Three members carry a "before the first Play" precondition -- `set_Pan`,
`Apply3D` and `set_IsLooped` -- and one private flag decides all three. The
native run found that the projection had dropped `set_IsLooped`'s guard entirely
AND never set the flag, so the pan/Apply3D mode latch could not latch either;
CNA surfaced it by refusing the call after playback exactly as the reference
does.

Every native fixture is SILENT PCM. The qualified artifacts open a real playback
device, so a fixture with signal in it would make audible noise on the machine
running the suite twenty times per cycle for no evidence gained. Silence
exercises creation, lifetime, transport and state identically; what it cannot
prove is audibility, and the scenario does not claim it.

`internal/dispatcher` joins `internal/bclhash` as the second internal package
this session: `FrameworkDispatcher.UpdateCalledAtLeastOnce` is `assembly`, not in
the contract, and read from another namespace, so Go leaves no other way to
share it without adding public surface.

Foundation 88 closed **`DynamicSoundEffectInstance`** and **`Microphone`**, and
with them the whole **`Microsoft.Xna.Framework.Audio`** namespace.

A streaming instance can NEVER loop, and the reference says so twice: its
`get_IsLooped` returns a constant false and its `set_IsLooped` accepts only the
value the getter already reports. Both are overrides, which is what made
`SoundEffectInstance` the first COMPOSED base outside the graphics namespace --
and its identity site is the disposal refusal every member in the family
reaches.

The native scenario found a defect again, the third time this session. It
asserted the byte count the REFERENCE produces and CNA answered a different one:
one second at 22050Hz mono is **44098** bytes to XNA, whose scale factor is
computed in float32, and **44100** to CNA, which computes the exact arithmetic.
All four sample-conversion routes were bound and then reverted; both types now
use the managed `AudioFormat` body the reference reads. The fixture's rate is
load-bearing and the scenario says so -- 8000Hz would have agreed with the wrong
projection, because its thousandth is exact in binary32.

`Microphone.Start` and `Microphone.GetData` are projected because the contract
declares them and the suite calls NEITHER. `MICROPHONE_CAPTURE_CALLS` exists to
be zero and the parent accounting FAILS the run on any other value: a non-zero
value is a run that began recording on someone's machine.

Foundation 89 closed the **`Microsoft.Xna.Framework.Input`** namespace with
`GamePad`, `GamePadCapabilities`, `Mouse`, `TouchPanel` and
`TouchPanelCapabilities` -- and it is the milestone that bound the fewest
routes it planned to.

`Microsoft.Xna.Framework.Input.Touch.dll` **declares no p/invoke anywhere**.
`TouchPanelCapabilities::GetCaps()` is ten bytes that zero a local and return
it; `TouchPanel::GetState()` updates from a state it just zeroed; and
`ReadGesture()` has two branches, both `throw`, and no `ret` instruction at all.
XNA 4.0's touch surface shipped for Windows Phone and the Windows build kept the
API with the device half removed. CNA implements all fourteen touch routes for
real -- they were bound, measured, and **reverted** under
`CONTRACT_DIVERGENCE`, because the pinned IL is the behaviour authority and a
projection answering real touches would diverge on the first `GetState` a game
makes. `TOUCH_PANEL_NATIVE_CALLS` is asserted to be zero from inside a live game
with a working runtime available, which is what makes the zero mean something.

GamePad's readers share one body whose middle branch is the one that matters:
`0x48f` is `ERROR_DEVICE_NOT_CONNECTED`, and a disconnected controller answers
an EMPTY state with no exception. Only some other failure throws. This machine
has no controller, so that is the branch the stress run takes and
`GAMEPADS_CONNECTED` is a measured 0 rather than a skip. `SetVibration` is
called only with two zeros.

Foundation 90 closed the **Model family** and with it the whole
**`Microsoft.Xna.Framework.Graphics`** namespace: twelve types, the largest
single family in the profile.

It also settled a blocker that had stood since Foundation 29.
`ReadOnlyCollection<T>` was DEFERRED **as a base** because all four Model
collections declare their own `GetEnumerator`, which HIDES the inherited one,
and the collision rule would have hashed both names. The answer adopted here is
that a hidden base member is UNREACHABLE -- reaching one in C# needs a cast to
the base, and CNA-Go projects no base type to cast to -- so it is excluded
rather than renamed, and each collection keeps exactly one `GetEnumerator`.

Three of the four collections wrap an ARRAY and one wraps a `List<Effect>`, and
the difference is observable twice: only the List-backed view is LIVE, and only
its enumerator is version-checked. `ModelMeshPart.set_Effect` is what mutates
it, and it is a reference count -- the old effect leaves the mesh's Effects only
when no sibling still uses it.

`Model.Draw` draws every model in the process through ONE private static
`Matrix[]`, grown to fit and never shrunk. That is reproduced rather than made
safe: the reference is not thread-safe here, and hiding it would describe a
different runtime.

What is not claimed: no native slice exercises the draw path, because the
contract declares ZERO public constructors across all twelve types and a Model
reaches a consumer only through `ContentManager.Load<Model>`. The content-reader
family unblocks it.

## No partial types, and no missing members

**`PARTIAL_TYPES` and `MISSING_MEMBER` are both zero.** Every type CNA-Go
projects, it projects completely; what remains is 51 types it does not project
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
   grow `ContentManager.Load<T>`'s closed set.
2. Then **Storage**, **Media/XACT**, the **content plumbing** and
   the **Design converters**. **Input** closed in Foundation 89 and the
   **Model family** in Foundation 90, which finished the Graphics namespace.

**The dynamic-buffer note, closed.** Foundation 83 probed it, Foundation 84
acted on it, and the outcome was smaller than the note expected: **one** new
route, `cna_vertex_buffer_set_data_raw_at_with_options`. The index side needed
none — `cna_index_buffer_set_data` and `_set_data_at` already carried an
`options` argument, and the static overloads pass a hardcoded zero because the
reference hardcodes `SetDataOptions.None` there. The four
`subscribe_content_lost` / `unsubscribe_content_lost` routes stay unbound for
the reason `cna_render_target_subscribe_content_lost` does: CNA raises them only
on DirectX9, Direct2D and Skia, and the qualified artifacts are HEADLESS and
SOFTWARE.

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
