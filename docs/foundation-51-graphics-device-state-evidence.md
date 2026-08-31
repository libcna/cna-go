# Foundation 51 — GraphicsDevice's render state

`GraphicsDevice` goes from 70 missing members to 55. Fifteen arrive: four
properties with both accessors, the `Viewport` **setter** whose getter shipped in
Foundation 1, three read-only properties, both masked `Clear` overloads and
`Present()`. `TOTAL_DIAGNOSTICS` drops from 234 to 219, and CNA-Go binds fourteen
new routes — the largest single-milestone ABI expansion in the project.

## Every accessor asks CNA, and that is measured rather than convenient

Five of the reference's getters are single field reads over a managed cache:

```text
get_BlendFactor        ldflda cachedBlendFactor; ldobj Color   12 bytes
get_MultiSampleMask    ldfld  cachedMultiSampleMask             7 bytes
get_ReferenceStencil   ldfld  cachedReferenceStencil            7 bytes
get_GraphicsProfile    ldfld  _graphicsProfile                  7 bytes
get_IsDisposed         ldfld  isDisposed                        7 bytes
```

and two reach D3D after a disposal check:

```text
get_ScissorRectangle      Helpers::CheckDisposed, then IDirect3DDevice9::GetScissorRect
get_GraphicsDeviceStatus  Helpers::CheckDisposed, then TestCooperativeLevel
```

Every setter opens with `Helpers::CheckDisposed(this, pComPtr)` and then reaches
the device.

CNA-Go projects **all fifteen** as native calls, and the reason is that the
managed cache is not reproducible here. The reference's constructor fills those
fields when it creates the D3D device. CNA-Go does not create the device — CNA
does, and `GraphicsDevice` is a borrowed, generation-checked facade over it. A
managed cache would therefore start at Go's zero values and disagree with the
live device until something wrote to it, which is exactly the second-source-of-
truth failure the settled rule forbids; it is the same reasoning that keeps the
ten `cna_graphics_device_manager_get_*` routes unbound, applied in the opposite
direction because here CNA is the only side that knows.

The consequence is stated rather than hidden: **five members that carry no error
in the reference carry one here.** A consumer writes `value, err :=` where C#
writes a property read.

## Three enum identities that cross unchanged

```text
XNA ClearOptions          Target 1  DepthBuffer 2  Stencil 4
CNA_CLEAR_OPTION_*        TARGET 1  DEPTH_BUFFER 2 STENCIL 4

XNA GraphicsProfile       Reach 0   HiDef 1
CNA_GRAPHICS_PROFILE_*    REACH 0   HI_DEF 1

XNA GraphicsDeviceStatus  Normal 0  Lost 1  NotReset 2
CNA_GRAPHICS_DEVICE_STATUS_*  NORMAL 0  LOST 1  NOT_RESET 2
```

The projection casts across all three, and a cast is correct only while the two
numbering schemes agree — which nothing in either type system says. Each pair is
therefore asserted **by value**, in an in-package test and in the behavior
corpus, so a change on either side is a failure rather than a silently different
device state.

## The two masked Clear overloads

`Clear(ClearOptions, Vector4, Single, Int32)` is twenty bytes of IL:

```csharp
Color local = new Color(color);              // Color::.ctor(Vector4)
this.Clear(options, local, depth, stencil);
```

so it converts and forwards, and the conversion is the `Color` constructor's own
rather than a second rounding rule invented here. The overload it forwards to is
543 bytes in the reference and one `cna_graphics_device_clear_options` call here.

CNA refuses a non-finite depth with `CNA_RESULT_INVALID_ARGUMENT`, and that
refusal surfaces: twenty stress cycles pass an infinity and require the call to
report. A projection that dropped the result would look identical from Go.

`Present()` is ten bytes forwarding to the three-pointer overload with three
nulls. That overload stays missing — it takes two `Nullable<Rectangle>` and an
`IntPtr` window handle, and CNA exposes no route that presents a sub-rectangle to
another window — which makes `Present` an overload **group**, so the no-argument
member is `PresentByNone` under the settled naming rule. The un-suffixed name was
tried first and the verifier reported it as an `UNEXPECTED_MEMBER` together with
two `OVERLOAD_MAPPING_MISMATCH`es.

## ABI: fourteen routes, and three controls about a conversion that says nothing

```text                                        before      after
BOUND_FUNCTIONS                               58      72
PROTOTYPE_TYPE_POSITIONS                     178     222
C_GO_MEASUREMENTS                            311     358
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         129     132
native ABI mutation controls                  79      83
ABI_MISMATCHES / FINDINGS                      0       0
```

Four new controls, and what separates them is which translation unit can see the
defect at all:

- **`clear-options-depth-and-stencil-swapped`** — the route is
  `(handle, options, color, float depth, int32 stencil)`, and the last two are a
  float and an int of the same width. Swapping them lets the bridge pass a depth
  where a stencil belongs, and C converts **both ways** without a word: a depth
  of `1.0f` arrives as the integer 1 and a stencil of 0 arrives as `0.0f`, so a
  clear still happens and clears the wrong thing. Only the probe's comparison
  with the canonical prototype catches it.
- **`blend-factor-passed-by-pointer`** and **`viewport-passed-by-pointer`** —
  a struct passed by address where CNA takes it by value. The viewport is the
  largest by-value struct CNA-Go passes: 24 bytes across six members, which the
  calling convention splits across registers.
- **`clear-options-mask-narrowed`** — a `uint16_t` mask. Target, DepthBuffer and
  Stencil are 1, 2 and 4, so every declared bit survives sixteen and nothing
  observable changes until CNA adds a bit above 65535. It is a **layout** control
  rather than a probe control for a structural reason worth recording: the
  manifest declares `CNA_ClearOptions` inside `#ifndef CNA_C_GRAPHICS_DEVICE_H`,
  so in the probe — which includes the canonical header — the narrowed typedef is
  never compiled at all. Only the manifest-only probe sees it, and only `sizeof`
  reports it. Three new alias measurements exist for exactly that.

## Evidence

A thirteenth native stress scenario, 20 isolated cycles, every call made from
inside a draw callback because that is the only moment CNA lends a device handle
out:

```text
DEVICE_STATE_CYCLES                20
DEVICE_STATE_ROUND_TRIPS          100   five per cycle
DEVICE_STATE_READ_ONLY_CHECKS      60   three per cycle
DEVICE_STATE_CLEAR_CALLS           40   both masked overloads
DEVICE_STATE_CLEAR_REFUSALS        20   a non-finite depth
DEVICE_STATE_PRESENT_CALLS         20
DEVICE_STATE_STALE_CHECKS          20
DEVICE_STATE_WRONG_THREAD_CHECKS   20
```

A round trip is a **write to the live device followed by a read back from it**,
not a comparison with what CNA-Go was just handed. That distinction is the whole
argument of the first section: a getter answering from a managed cache would pass
a test that only compared its own input, and would be a second source of truth.

Three read-only members are asserted to be values their enum actually
**declares**, rather than against a constant chosen here — the device decides what
profile and status it has, and what is being measured is that the identity
survived the boundary.

The stale check runs after the run has finished: every facade from a completed
generation must report `ErrStaleGeneration` rather than reach a device that is
gone. The wrong-thread check runs during it, from a second goroutine.

The three-member `GraphicsProfile`/`GraphicsDeviceStatus`/`IsDisposed` group and
the fifteen-member nil-facade guard are measured where they reach no native code:
three in-package tests, four behavior-corpus observations and one external-canary
test that compiles every member against its exact signature.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1904     1919
MISSING_MEMBER                               101       86
TOTAL_DIAGNOSTICS                            234      219
COMPLETE_TYPES                               119      119
PARTIAL_TYPES                                  5        5
UNEXPECTED_MEMBER                              0        0

behavior corpus                              673      677
external canary tests                         77       78
native stress scenarios                       12       13
native ABI mutation controls                  79       83
runtime capability rows                       51       52
```

## What this milestone does not claim

- **Nothing here proves a rendered pixel changed.** The artifact is HEADLESS.
  What is proved is that each value was accepted by CNA's real device and read
  back from it unchanged, and that a refused argument is reported.
- **`GraphicsDevice` is still partial**, with 55 members left. They are four
  families: the state-object properties (`BlendState`, `DepthStencilState`,
  `RasterizerState`, `SamplerStates`, `Textures`, …) which need those types; the
  buffer and render-target surface (`SetVertexBuffer`, `SetRenderTarget`,
  `GetRenderTargets`, …) which needs `VertexBuffer`, `IndexBuffer` and the render
  targets; the six drawing calls, which need a vertex declaration and a bound
  buffer to be meaningful; and `Adapter`/`DisplayMode`/`Reset`, which need
  `GraphicsAdapter`.
- **`IsDisposed` reports what CNA's device says about itself**, which is not the
  same question as whether this Go facade is usable. A facade from a previous
  native generation is rejected by the generation check before the call is made,
  and reports that instead.
- **The generic `GetBackBufferData` overloads are untouched.** They are `!!0[]`
  members and wait on the generic-method projection rule, like `Texture2D`'s
  `SetData`/`GetData`.
