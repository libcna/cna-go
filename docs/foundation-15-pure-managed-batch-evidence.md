# Foundation Milestone 15 pure-managed batch B evidence

## Authority and batch scope

Foundation Milestone 15 completes **12 public XNA types** carrying **122
mapped Go identities** in three clusters:

| Cluster | Theme | Types | Identities |
|---|---|---:|---:|
| A | the last safe pure-managed leaf enums | 5 | 46 |
| B | the GamePad and Mouse value structs | 5 | 57 |
| C | the Touch value structs | 2 | 19 |

Cluster A closes the safe pure-managed leaf-enum category entirely: after it,
no dependency-complete missing enum remains.

Two authorities are used, and they are kept distinct.

**Public surface** remains the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

**Reference behavior** for clusters B and C was read as IL directly from the
retained original assemblies — the same assemblies the VertexElement closure
used, re-verified by hash for this milestone:

| Assembly | SHA-256 |
|---|---|
| `Microsoft.Xna.Framework.dll` | `38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130` |
| `Microsoft.Xna.Framework.Input.Touch.dll` | `b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25` |

Every clamping rule, hash algorithm, ToString layout, and equality rule below
was read from that IL. Nothing was inferred, guessed, or copied from a
third-party reimplementation.

## Cluster A — the last leaf enums

Each entry is an ordinary or flags enum with base `System.Enum`,
`System.Int32` storage, no direct interfaces, and no member other than its
literals plus synthetic `value__`.

| # | XNA type | Flags | Source ids | Go ids | Fan-out |
|---:|---|---|---:|---:|---:|
| 1 | `Microsoft.Xna.Framework.Graphics.ColorWriteChannels` | true | 7 | 6 | 1 |
| 2 | `Microsoft.Xna.Framework.Graphics.StencilOperation` | false | 9 | 8 | 1 |
| 3 | `Microsoft.Xna.Framework.Graphics.TextureFilter` | false | 10 | 9 | 1 |
| 4 | `Microsoft.Xna.Framework.Input.GamePadType` | false | 11 | 10 | 1 |
| 5 | `Microsoft.Xna.Framework.Graphics.Blend` | false | 14 | 13 | 1 |

`ColorWriteChannels.All = 15` is a **pinned aggregate literal declared by the
contract**, not an invented convenience constant; the behavior corpus asserts
that it equals `Red|Green|Blue|Alpha`.

`GamePadType` has a sparse table: `BigButtonPad = 768` while every other
literal is 0-8. `Blend.One = 0` and `Blend.Zero = 1` are deliberately not
their own numeric values.

The incremental scoreboard for cluster A, regenerated after every type:

| # | Completed type | Ids | TARGET_TYPES | TARGET_MEMBERS | TOTAL_DIAGNOSTICS | MISSING_TYPE | MISSING_MEMBER | COMPLETE_TYPES | PARTIAL_TYPES |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 0 | *(Foundation 14 baseline)* | — | 90 | 1482 | 344 | 167 | 177 | 85 | 5 |
| 1 | `ColorWriteChannels` | 6 | 91 | 1488 | 343 | 166 | 177 | 86 | 5 |
| 2 | `StencilOperation` | 8 | 92 | 1496 | 342 | 165 | 177 | 87 | 5 |
| 3 | `TextureFilter` | 9 | 93 | 1505 | 341 | 164 | 177 | 88 | 5 |
| 4 | `GamePadType` | 10 | 94 | 1515 | 340 | 163 | 177 | 89 | 5 |
| 5 | `Blend` | 13 | 95 | 1528 | 339 | 162 | 177 | 90 | 5 |

## Cluster B — GamePad and Mouse value structs

All five are `System.ValueType` structs whose members are infallible managed
value work. **No member carries a synthetic Go error result**, which the
verifier measures explicitly.

| XNA type | Source ids | Go ids | Fan-out |
|---|---:|---:|---:|
| `Microsoft.Xna.Framework.Input.GamePadThumbSticks` | 8 | 8 | 1 |
| `Microsoft.Xna.Framework.Input.GamePadTriggers` | 8 | 8 | 1 |
| `Microsoft.Xna.Framework.Input.GamePadDPad` | 10 | 10 | 1 |
| `Microsoft.Xna.Framework.Input.GamePadButtons` | 17 | 17 | 1 |
| `Microsoft.Xna.Framework.Input.MouseState` | 14 | 14 | 1 |

`GamePadButtons` was unlocked by the Foundation-14 `Buttons` enum and is the
direct payoff of that milestone.

### Constructor semantics, from IL

- `GamePadThumbSticks` clamps each stick with a component-wise
  `Vector2.Min(v, One)` then `Vector2.Max(v, -One)`. The XNA Vector2 minimum
  keeps its **second** operand when the comparison is false, so a **NaN
  component clamps to 1** rather than staying NaN.
- `GamePadTriggers` clamps each trigger with `Math.Min(v, 1f)` then
  `Math.Max(v, 0f)`. `System.Math.Min`/`Max` **propagate NaN**, so a NaN
  trigger stays NaN — the opposite of the thumbstick rule. `Math.Max(-0, 0)`
  returns its second operand, so **negative zero becomes positive zero**.
- `GamePadDPad`'s parameter order is `(up, down, left, right)` while its
  sequential fields are declared `up, right, down, left`. The parameter order
  is the public contract and is asserted directly.
- `GamePadButtons` derives every state as
  `(buttons & mask) == mask ? Pressed : Released`. Each mask read from the IL
  matches the corresponding pinned `Buttons` literal exactly — an independent
  cross-validation of the Foundation-14 enum:
  A=0x1000, B=0x2000, X=0x4000, Y=0x8000, Start=16, Back=32, LeftStick=64,
  RightStick=0x80, LeftShoulder=0x100, RightShoulder=0x200, BigButton=0x800.
  Thumbstick-direction and trigger literals have **no** `GamePadButtons`
  field and cannot turn any button on.
- `MouseState` stores all eight components unchanged with no validation.

### Hashing, from IL

`GamePadThumbSticks`, `GamePadTriggers`, `GamePadDPad`, and `GamePadButtons`
all call `Microsoft.Xna.Framework.Helpers.SmartGetHashCode`. The helper pins
the boxed value, XORs every complete 32-bit word of the marshalled layout, and
substitutes `Int32.MaxValue` when the XOR is zero. XOR is commutative, so the
declared field order does not affect the result. The zero substitution is
intentional and creates compatible collisions; it must not be replaced with a
better-distributed combine function.

`MouseState` does **not** use that helper. It XORs the eight member hash codes
directly, and `Int32.GetHashCode` and the `Int32`-backed `ButtonState` hash
both return the value itself — so there is **no** `Int32.MaxValue`
substitution and a zero-valued snapshot hashes to **0**. The corpus asserts
that contrast directly.

Exact fixtures:

| Value | Hash |
|---|---:|
| `GamePadThumbSticks{(0.5,-1),(1,1)}` | -2139095040 |
| zero `GamePadThumbSticks` | 2147483647 |
| `GamePadTriggers(0.25, 0.75)` | 29360128 |
| `GamePadDPad(Up pressed only)` | 1 |
| `GamePadDPad(Up and Right pressed)` | 2147483647 *(compatible collision)* |
| `GamePadButtons(A)` | 1 |
| `GamePadButtons(A\|Start)` | 2147483647 *(compatible collision)* |
| `MouseState(1,2,3, L=P,M=R,R=P,X1=R,X2=P)` | 1 |
| zero `MouseState` | 0 *(no substitution)* |

### ToString, from IL

`GamePadDPad`, `GamePadButtons`, and `MouseState` accumulate the names of
pressed buttons separated by single spaces and substitute `None` when the
accumulator is empty. The reference compares against `Pressed` **exactly**, so
an arbitrary raw `ButtonState` such as `12345` contributes no name.

| Type | Format | Name order |
|---|---|---|
| `GamePadThumbSticks` | `{Left:{0} Right:{1}}` | — |
| `GamePadTriggers` | `{Left:{0} Right:{1}}` | — |
| `GamePadDPad` | `{DPad:{0}}` | Up, Down, Left, Right |
| `GamePadButtons` | `{Buttons:{0}}` | A, B, X, Y, LeftShoulder, RightShoulder, LeftStick, RightStick, Start, Back, BigButton |
| `MouseState` | `{X:{0} Y:{1} Buttons:{2} Wheel:{3}}` | Left, Right, Middle, XButton1, XButton2 |

## Cluster C — Touch value structs

| XNA type | Source ids | Go ids | Fan-out |
|---|---:|---:|---:|
| `Microsoft.Xna.Framework.Input.Touch.GestureSample` | 7 | 7 | 1 |
| `Microsoft.Xna.Framework.Input.Touch.TouchLocation` | 12 | 12 | 2 |

`GestureSample` stores six components unchanged and validates nothing.

`TouchLocation` stores `id`, `state`, `x`, `y`, `prevState`, `prevX`, `prevY`.
Its single-sample constructor leaves the previous fields zero, so the previous
state is the zero literal `Invalid`.

`TryGetPreviousLocation` maps to the established out-parameter projection
`() (bool, TouchLocation)`. When the previous state is `Invalid` it reports
`false` and yields a location whose `Id` is **-1** and whose other fields are
zero; otherwise it reports `true` and yields the previous sample promoted into
a location that itself carries no previous sample.

### A genuine XNA asymmetry

The IL shows that `TouchLocation`'s equality operator and its `IEquatable`
implementation **deliberately disagree**:

- `op_Equality` compares all seven fields, **including** `state` and
  `prevState`;
- `Equals(TouchLocation)` compares only `id`, `x`, `y`, `prevX`, and `prevY`,
  **ignoring both state fields**.

So two locations that differ only in state satisfy `Equals` while the operator
reports inequality. This is reproduced exactly and is asserted by both a
focused test and a behavior observation. It was **not** "corrected".

`TouchLocation.GetHashCode` sums `Int32.GetHashCode(id)` and two
`Single.GetHashCode` values with wrapping `Int32` addition; both signed zeros
hash to zero. `ToString` is `{Position:{0}}` and deliberately omits the state.

## A mapping-table defect fixed

`GestureSample.Timestamp` is the first `System.TimeSpan` in the mapped surface
to appear **outside** the `Microsoft.Xna.Framework` package. The Go `bclTypes`
table in `tools/api_compat/mapping.go` returned a bare `TimeSpan` for every
owner, while `tools/api_compat/mapping-rules.json` has always declared
`"System.TimeSpan": "framework.TimeSpan"`.

The declared rule is authoritative, and the same package-qualification pattern
already exists for both XNA types and the `Iterator` projection. `mapType` now
qualifies `TimeSpan` when the owner package is not the framework package. This
is a defect fix that aligns the verifier with the mapping rules it is meant to
enforce, not a new policy: `EXPECTED_GO_TYPES` and `EXPECTED_GO_MEMBERS` are
unchanged at 257 / 3,243 and no diagnostic category moved.

## Structural and mutation evidence

Two report categories were added:

- `foundation15EnumClosures` reuses the Foundation-14 table-driven enum
  closure — all 5 report `PASS`.
- `foundation15ValueStructClosures` is a new category for pure managed value
  structs. Beyond identity arithmetic it records the family's central semantic
  claim, `errorResults`, which must be **0**. All 7 report `PASS` with
  `errorResults: 0` and zero local diagnostics.

The enum mapped-contract and exhaustive-defect tests were generalised across
milestones (`TestBatchEnumMappedContracts`,
`TestBatchEnumDefectsRejectedForEveryType`) and now cover all 30 pinned enums
and 167 enum identities, producing **366 enum negative cases**.

`TestFoundation15ValueStructDefectsRejectedForEveryType` applies 10 structural
defects to each of the 7 value structs — **70 value-struct negative cases** —
each with an asserted clean baseline:

| Defect | Raised category |
|---|---|
| `missing_type` | `MISSING_TYPE` |
| `wrong_package` | `MISSING_TYPE` |
| `projected_as_class` | `TYPE_KIND_MISMATCH` |
| `missing_last_member` | `MISSING_MEMBER` |
| `missing_first_member` | `MISSING_MEMBER` |
| `renamed_last_member` | `MISSING_MEMBER` |
| `synthetic_error_result` | `RETURN_MAPPING_MISMATCH` |
| `wrong_result_type` | `RETURN_MAPPING_MISMATCH` |
| `wrong_constructor_parameters` | `PARAMETER_MAPPING_MISMATCH` |
| `unexpected_mutator` | `UNEXPECTED_MEMBER` |

`projected_as_class` guards value semantics: projecting a `System.ValueType`
as a Go reference type would silently change copy behavior.
`synthetic_error_result` guards the infallibility claim.
`unexpected_mutator` guards immutability — these values expose no setters.

The declared mutation inventory grows from **177 to 210** cases.

## Behavior provenance

The corpus grows from **411 to 476** observations / 476 assertions with zero
failures.

`PURE_XNA_DERIVED` covers every pinned raw value, the flags unions and
single-bit structure of `ColorWriteChannels`, the clamping rules, every hash
fixture above, every ToString layout, the `TouchLocation` equality asymmetry,
and the signed-zero hash canonicalization.

`GO_LANGUAGE_PROJECTION` covers zero values, arbitrary positive and negative
raw enum values, Go copy semantics for `MouseState`, the fact that a NaN
`GamePadTriggers` is not equal to itself under IEEE equality, and that an
arbitrary raw `ButtonState` is not `Pressed`.

The two categories are never conflated, and the corpus runs with
`CNA_NATIVE_LIBRARY` unset.

## Runtime boundary — no capability inflation

Completing these types claims managed metadata and managed value behavior
only. Explicitly: **no game pad, mouse, or touch capability is claimed.**
CNA-Go exposes no `GamePad`, `GamePadState`, `GamePadCapabilities`, `Mouse`,
or `TouchPanel` type; there is no polling loop, device enumeration, connection
state, dead-zone handling, vibration, cursor, SDL route, or input backend.
`Blend`, `StencilOperation`, `TextureFilter`, and `ColorWriteChannels` imply
no render state, blending, stencilling, or sampling.

`go run ./tools/capabilities --check` still reports `RUNTIME_CAPABILITIES=38
STATUS=PASS` and `docs/runtime-capabilities.json` is unmodified. No CNA
source, C binding, cgo route, native mirror, layout, or callback was added.

## Skipped candidates

| Candidate | Fan-out | Category | Exact reason |
|---|---:|---|---|
| `Design.MathTypeConverter` | 12 | `SKIPPED` — unmapped BCL | `ExpandableObjectConverter`, `ITypeDescriptorContext`, and `PropertyDescriptorCollection` are outside the BCL mapping table. |
| `Graphics.GraphicsResource` | 11 | `SKIPPED` — lifetime architecture | `IDisposable` base with `Dispose`, `Dispose(bool)`, `Finalize`, `IsDisposed`, a `Disposing` event, and a `GraphicsDevice` property. Selected as the next milestone instead. |
| `Graphics.IEffectMatrices`, `Graphics.IEffectFog` | 5, 5 | `SKIPPED` — new mapping policy | Projecting settable-property interfaces needs a general managed-interface decision this batch must not invent; deferred Effects/3D family with no implementor in scope. |
| `IGameComponent` | 3 | `SKIPPED` — Game lifecycle | Not a pure managed data leaf. |
| `Audio.AudioListener` | 3 | `SKIPPED` — mapping classification | **Blocker partially removed.** The IL now establishes the semantics exactly: `Position`/`Velocity` default to `Vector3.Zero`, `Forward`/`Up` to `Vector3.Forward`/`Vector3.Up`, and the internal `FlipHandedness` (X, Y, −Z) is an involution applied on both read and write, so public round-trips are identity. What remains is a **mapping decision**, not a semantic gap: as a CLR `class` the verifier projects every member as fallible, which would invent nine failure modes that the IL shows do not exist. Correcting it means classifying the type as pure managed. That is a public-API policy change and is surfaced rather than taken unilaterally. |
| `Audio.AudioEmitter` | 3 | `SKIPPED` — new mapping rule required | Same as `AudioListener`, plus `DopplerScale`, which defaults to 1 and whose setter throws `ArgumentOutOfRangeException` when the value is `< 0` (the IL uses `bge.un`, so **NaN does not throw**). Only the **setter** is fallible; the getter is pure field access. The existing `managedFallibleMembers` rule marks a property fallible on *both* accessors, so an honest projection needs a **new setter-only fallibility rule**. Inventing one to fit a single candidate is exactly what the batch rules forbid. |
| `Graphics.DisplayMode` | 3 | `SKIPPED` — fake adapter capability | Read-only properties, no public constructor; only a real adapter can produce one. |
| `Content.ContentManager` | 3 | `SKIPPED` — deferred family, unmapped BCL | Needs `IServiceProvider`, `Action\`1`, `IDisposable`. |
| `Input.Touch.TouchPanelCapabilities` | 1 | `SKIPPED` — fake device capability | Two read-only properties, no public constructor. Implementing it would expose getters that can only ever return the zero value — a fabricated capability report. |
| `Input.GamePadCapabilities` | 1 | `SKIPPED` — fake device capability | 26 read-only properties, no public constructor. Same reasoning. |
| `Audio.RendererDetail` | 1 | `SKIPPED` — fake device capability | Read-only `FriendlyName`/`RendererId`, no public constructor; produced only by audio renderer enumeration. |

## Checkpoints

| Point | Gates | Result |
|---|---|---|
| After each cluster-A enum | gofmt, focused package tests, full `api_compat --mode report`, dependency-graph regeneration | 5/5 `PASS`, delta exactly as predicted every time |
| After each cluster-B and cluster-C type | gofmt, package build, focused package tests | `PASS` |
| After each cluster | full `api_compat --mode report`, closure measurements | `PASS`; all 12 closures `PASS` |
| After the verifier work | `TestBatchEnum*` (366 cases), `TestFoundation15ValueStruct*` (70 cases), `TestMutationFixtures` (210 cases) | `PASS` |
| After the behavior work | `go run ./tools/behavior` | 476/476, 0 failures |
| Batch-complete | `gofmt -l .`, `go vet`, `go build`, `go build -trimpath`, `go test ./...`, `go test -race ./...`, leak-only, PackedVector, capabilities | all `PASS` |
| Native | `native_abi` and `native_stress` against the **exact** Foundation-11 pinned library | reproduce the committed evidence exactly |

## Native ABI provenance

The Foundation-14 handoff recorded that the exact ABI-0.7 library admitted in
Foundation 11 was "no longer present on the development machine". **That was
wrong and is corrected here.** A complete search of all 47 `libcna_c_api.so*`
files on the machine found the admitted binary
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f` in two
locations, together with the canonical header root it was measured against.

Re-running `tools/native_abi` against that exact binary and header root
reproduces the committed `docs/generated/native-abi-report.json` **key for
key**, with the sole difference being `header_root`, which the committed
evidence stores in normalized relative form. Re-running the native stress
suite against it reproduces every counter identically with
`GO_RACE_STATUS=PASS`.

The four distinctions are therefore:

| Property | Status |
|---|---|
| Exact binary provenance | **VERIFIED** — the admitted binary is present and hash-matched |
| ABI compatibility | **VERIFIED** — 23 / 67 / 96 / 28 / 2 / 5, identical function list, 0 missing header symbols, 0 missing library symbols, 0 mismatches |
| Behavioral equivalence | **VERIFIED on the exercised surface** — the 20-cycle stress suite reproduces every counter with 0 crashes, 0 UAF, 0 double-free |
| Reproduced build output | **NOT ESTABLISHED** — CNA was not rebuilt from source in this milestone, and no reproducible-build procedure was run |

Separately, the shared checkout `~/deps/cna-c-abi-0.7.0/libcna_c_api.so`
(`c62949d23d3745964f5e557a06665875621ed4cb6e2930e3f282afd5911f2dcb`) is a
**different, more recent build** of the same ABI. It also passes every ABI and
stress gate identically, so it is ABI- and behaviorally-compatible on the
exercised surface, but it is **not** the admitted binary and must not be
described as byte-identical. `docs/generated/native-abi-report.json` was not
rewritten by any run in this milestone.

One operational caveat: the admitted binary currently lives only under `/tmp`,
which is not durable. Preserving it in a stable location would protect the
pinned provenance.

## Final structural scoreboard

| Counter | Foundation 14 | Foundation 15 | Delta |
|---|---:|---:|---:|
| `REFERENCE_TYPES` | 257 | 257 | 0 |
| `REFERENCE_MEMBERS` | 2964 | 2964 | 0 |
| `EXPECTED_GO_TYPES` | 257 | 257 | 0 |
| `EXPECTED_GO_MEMBERS` | 3243 | 3243 | 0 |
| `TARGET_TYPES` | 90 | 102 | +12 |
| `TARGET_MEMBERS` | 1482 | 1604 | +122 |
| `TOTAL_DIAGNOSTICS` | 344 | 332 | −12 |
| `MISSING_TYPE` | 167 | 155 | −12 |
| `MISSING_MEMBER` | 177 | 177 | 0 |
| `COMPLETE_TYPES` | 85 | 97 | +12 |
| `PARTIAL_TYPES` | 5 | 5 | 0 |
| `MISSING_TYPES` | 167 | 155 | −12 |

Every preservation, mismatch, leak, allowlist, and unmeasured counter is zero
before and after. Interface witnesses remain 25 / 17 / 8. The five partial
types are unchanged at 39 / 40 / 70 / 16 / 12 for a combined 177 missing
members.

The dependency-complete node count moved 55 → 47.
