# CNA-Go roadmap — what is done, and what remains

This file is the answer to "what is left?". It is written from **measured**
state, not from intent: every count below comes from a tool in this repository,
and every claim about what CNA can do comes from the canonical headers at
`~/deps/cna-c-abi-0.21.0/include/CNA/C/`.

Regenerate the numbers before trusting them:

```sh
go run ./tools/api_compat            # the scoreboard and docs/generated/
go run ./tools/native_abi -headers ~/deps/cna-c-abi-0.21.0/include \
                          -library ~/deps/cna-c-abi-0.21.0/libcna_c_api.so
go run ./tools/external_consumer -source .
```

Last measured at Foundation 73:

```text
TOTAL_DIAGNOSTICS   109       COMPLETE_TYPES        154
MISSING_TYPE        101       PARTIAL_TYPES           2
MISSING_MEMBER        8       UNEXPECTED_MEMBER       0
BOUND_FUNCTIONS     230       ABI_MISMATCHES          0
LAYOUT AGREEMENTS   457       external canary tests  97
```

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

## The 8 missing members

These are the only two incomplete types in the profile. Both are small and both
are blocked on a *type* rather than on CNA.

| type | members | what it needs first |
| --- | --- | --- |
| `Game` | `LaunchParameters`, `ShowMissingRequirementMessage(Exception)` | `LaunchParameters : Dictionary<string,string>` — a BCL projection decision |
| `GraphicsDeviceManager` | `FindBestDevice`, `CanResetDevice`, `RankDevices`, `OnPreparingDeviceSettings`, `PreparingDeviceSettings` add/remove | `GraphicsDeviceInformation` and `PreparingDeviceSettingsEventArgs` |

Closing these three small types closes both — and takes `PARTIAL_TYPES` to
zero, which is the cleanest single milestone left.

## The 101 missing types, by family

Route counts are `grep`ed from the canonical CNA headers. A family with routes
is `ACTIONABLE_LOCAL`; the classification column says what actually gates it.

| family | types | CNA routes | classification |
| --- | ---: | ---: | --- |
| **Stock effects** — `BasicEffect`, `AlphaTestEffect`, `DualTextureEffect`, `EnvironmentMapEffect`, `SkinnedEffect`, `EffectMaterial`, `IEffectLights`, `DirectionalLight` | 8 | 19 / 13 / 9 / 17 / 21 / — / — / 13 | **ACTIONABLE_LOCAL** |
| **Model family** — `Model`, `ModelBone`, `ModelMesh`, `ModelMeshPart` + 4 collections + 3 nested enumerators | 12 | 233 | **ACTIONABLE_LOCAL** |
| **Audio** — `SoundEffect`, `SoundEffectInstance`, `DynamicSoundEffectInstance`, `Microphone`, 3 exceptions, `RendererDetail` | 8 | 63 / 18 / 10 | **ACTIONABLE_LOCAL** (SDL3 present in `lib/`) |
| **XACT** — `AudioEngine`, `SoundBank`, `WaveBank`, `Cue`, `AudioCategory` | 5 | 32 / 12 / 10 / 17 | **ACTIONABLE_LOCAL** |
| **Media** — `MediaPlayer`, `MediaLibrary`, `MediaQueue`, `Song`, `Video`, `VideoPlayer`, `Album`, `Artist`, `Genre`, `Picture`, `Playlist` + 8 collections | 20 | 31 / 23 / 53 / 46 / 50 / 55 / 19 | **ACTIONABLE_LOCAL** |
| **Design converters** — `MathTypeConverter` + 12 subclasses | 13 | none needed | **ACTIONABLE_LOCAL** — pure managed; needs a `System.ComponentModel` projection decision |
| **Content plumbing** — `ContentReader`, `ContentTypeReader(+<T>)`, `ContentTypeReaderManager`, `ResourceContentManager`, `ContentLoadException`, 5 serializer attributes | 11 | 87 | **PARTLY DELIBERATE_OUT_OF_SCOPE** — CNA-Go parses no XNB container; the 5 attributes need a Go answer for CLR attributes |
| **Input** — `GamePad`, `GamePadCapabilities`, `Mouse`, `TouchPanel`, `TouchPanelCapabilities` | 5 | 68 / 41 / 21 / 59 | **ACTIONABLE_LOCAL** |
| **Storage** — `StorageDevice`, `StorageContainer`, `StorageDeviceNotConnectedException` | 3 | 65 / 12 | **ACTIONABLE_LOCAL** |
| **Graphics leftovers** — `OcclusionQuery`, `DynamicVertexBuffer`, `DynamicIndexBuffer`, `VertexPositionColor(+Texture)`, `VertexPositionTexture`, `VertexPositionNormalTexture`, 3 device exceptions | 10 | 12 / **0** / **0** | **MOSTLY ACTIONABLE_LOCAL** — see note |
| **Root types** — `LaunchParameters`, `GraphicsDeviceInformation`, `PreparingDeviceSettingsEventArgs`, `TitleContainer`, `FrameworkDispatcher` | 5 | — | **ACTIONABLE_LOCAL** |
| **GamerServices** — `GamerServicesComponent` | 1 | 53 (`guide_*`) | **ACTIONABLE_LOCAL** |

**The dynamic-buffer note.** `DynamicVertexBuffer` and `DynamicIndexBuffer`
have **no dedicated CNA routes**. They are almost certainly the existing
vertex/index buffer routes with a `BufferUsage`, plus a `ContentLost` event —
but that is a hypothesis, and the rule in `plan.md` is that a route is bound
only when it AGREES with the member's reference body. Probe before projecting.

## Suggested milestone order

The ordering rule is in `plan.md` under **Next milestone selection rule**; this
is that rule applied to the state above.

1. **`GraphicsDeviceInformation`, `PreparingDeviceSettingsEventArgs`,
   `LaunchParameters`** → closes `Game` and `GraphicsDeviceManager`, taking
   `PARTIAL_TYPES` and `MISSING_MEMBER` to **zero**. Smallest milestone with
   the largest scoreboard effect.
2. **The vertex structs** (`VertexPositionColor` and its three siblings) —
   four small value types, `IVertexType` is already projected, and they unblock
   realistic `DrawUser*` and Model work.
3. **The six stock effects + `DirectionalLight` + `IEffectLights` +
   `EffectMaterial`** — `Effect` is projected and every one of these composes
   it, so this is the composition rule applied to a family that is ready.
4. **`SoundEffect` / `SoundEffectInstance`** — this also grows
   `ContentManager.Load<T>`'s closed set, which is currently held back only by
   the missing Go type.
5. **The Model family** — largest single family, 233 routes, and it depends on
   3 and 2 above.
6. **Input**, then **Storage**, then **Media/XACT**, then **Design converters**.

`TitleContainer`, `FrameworkDispatcher` and the exception types can be folded
into whichever milestone first needs them.

## Two open decisions that gate whole families

Neither is a CNA question; both are Go projection questions, and both should be
settled with a measured argument the way `plan.md`'s BCL rule was.

1. **CLR attributes.** The five `ContentSerializer*Attribute` types have no Go
   counterpart — Go has struct tags, not attributes. Until this is decided,
   those five stay missing.
2. **`System.ComponentModel.TypeConverter`.** All 13 `Design` converters derive
   from it. It is not in the profile, so the composition rule has nothing to
   compose against.

## The gates every milestone must pass

Unchanged from Foundation 44 onward, and all of them must be green:

```sh
go test ./...                                    # unit + verifier tests
go run ./tools/behavior                          # behaviour corpus
go run ./tools/api_compat                        # strict API + inventory
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
