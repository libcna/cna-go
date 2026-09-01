# Foundation 73 — the rest of `GraphicsDevice`, and the profile's first pixels

`GraphicsDevice` closes. `RenderTargetCube` and `RenderTargetBinding` arrive
with it, the type gains its one public constructor, and the back-buffer
readback turns out to work on one of the qualified artifacts — which makes this
the first milestone that can say **`VERIFIED_PIXEL`** instead of
`VERIFIED_NATIVE_DRAW`.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 123     109
MISSING_TYPE                                      103     101
MISSING_MEMBER                                     20       8
COMPLETE_TYPES                                    151     154
PARTIAL_TYPES                                       3       2
UNEXPECTED_MEMBER                                   0       0
ABI_MISMATCHES                                      0       0

BOUND_FUNCTIONS                                   220     230
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              426     457
XNA_COMPOSED_BASE_RELATIONSHIPS                     4       5
XNA_COMPOSED_DERIVED_TYPES_PROJECTED               15      16
external canary tests                              92      97
capability rows                                    71      71
```

The **eight** members still missing in the whole profile are `Game` (2) and
`GraphicsDeviceManager` (6). Nothing else in the pinned XNA 4.0 surface that
CNA-Go admits is partial.

## What closed

| member | shape |
| --- | --- |
| `PresentationParameters` | fallible getter; reads `cna_graphics_device_get_presentation_parameters` and builds a fresh managed object |
| `Reset()` / `Reset(pp)` / `Reset(pp, adapter)` | three overloads, one native route |
| `Present(Rectangle?, Rectangle?, IntPtr)` | **refuses**, see below |
| `GetBackBufferData<T>` ×3 | package functions on the generic-method rule |
| `.ctor(GraphicsAdapter, GraphicsProfile, PresentationParameters)` | the profile's first **OWNED** device |
| `SetRenderTarget(RenderTargetCube, CubeMapFace)` | |
| `SetRenderTargets(RenderTargetBinding[])` | an empty array is the back buffer, not a refusal |
| `GetRenderTargets()` | infallible; answers a **copy** |
| `RenderTargetCube` | composes `TextureCube`, the fifth `COMPOSED` base |
| `RenderTargetBinding` | value struct, two `.ctor`s and `op_Implicit` |

## The one member that refuses, and why it is not a renderer limit

`Present(Nullable<Rectangle>, Nullable<Rectangle>, IntPtr)` is projected and
always fails, with `errPresentRectangles`. CNA's whole C ABI has **one** present
route:

```c
CNA_Result cna_graphics_device_present(CNA_GraphicsDevice device);
```

It carries no source rectangle, no destination rectangle and no window handle.
Binding this overload to it would present the *whole* back buffer into the
device's *own* window under a name that promises a sub-rectangle and a foreign
window — a silently wrong answer rather than a missing one. So the projection
refuses and says exactly what the route lacks.

This is `BLOCKED_UPSTREAM`, not `BLOCKED_RENDERER`: no artifact can change it,
because the limit is in the ABI's prototype rather than in a backend. The stress
scenario asserts the refusal is **not** a CNA result code — the call must never
reach CNA — in all 20 cycles on both artifacts.

## The first pixel readback in the profile

Every draw proof up to Foundation 72 was classified `VERIFIED_NATIVE_DRAW`: CNA
accepted the submission, and no qualified artifact could read the result back to
show *what* had been drawn. `GetBackBufferData` changes that on one artifact.

The scenario clears the back buffer to a marker colour nothing else in the
process uses — `(17, 34, 51, 255)` — reads every texel back, and compares
**exactly**. An approximate comparison would hide the one failure the check
exists to catch: a readback that returns a plausible buffer that is not the one
that was drawn.

```text                       HEADLESS   SOFTWARE
BACK_BUFFER_READS                 0         20
BACK_BUFFER_READ_REFUSALS        20          0
BACK_BUFFER_PIXEL_CHECKS          0         20
```

HEADLESS refuses in its own words:

```text
cna_graphics_device_get_backbuffer_data_window failed with CNA result 6:
GraphicsDevice::GetBackBufferData: the Headless renderer does not rasterize
and has no backbuffer pixel storage to read back.
```

That is a `BLOCKED_RENDERER` for HEADLESS and a **closed** capability on
SOFTWARE. The claim being made is narrow and worth stating precisely: 20 of 20
SOFTWARE cycles read back a full-size buffer in which every texel equalled the
clear colour. It proves the transfer, the element layout and the row stride. It
does **not** yet prove a *drawn* image reads back correctly — that would need a
draw followed by a readback, which is the next milestone's evidence to produce,
not this one's.

## The two renderers disagree about cube render targets, in the other direction

```text                                HEADLESS   SOFTWARE
RENDER_TARGET_CUBE_CREATIONS             20         20
RENDER_TARGET_CUBE_BINDS                 20          0
RENDER_TARGET_CUBE_BIND_REFUSALS          0         20
```

Both artifacts *create* one. HEADLESS binds it and SOFTWARE refuses the bind, so
between the two every branch of `SetRenderTarget(cube, face)` is exercised —
which is the reason both are run rather than only the more capable one.

## A defect this milestone found in itself

The stress run failed on its first HEADLESS pass with:

```text
a disposed owned device does not report itself disposed
```

`DestroyOwnedDevice` zeroed `Device.owned` after destroying the handle, and
`nativeHandle()` treats a zero `owned` as "this is the borrowed device a Game
publishes" — so a **destroyed** consumer-owned device silently became a facade
over the running game's device and answered its questions. `IsDisposed` reported
`false` because it was asking a different, very much alive device.

The fix is a `wasOwned` flag that survives destruction, so `nativeHandle()`
reports `ErrDisposed` instead of falling through to the borrowed path. The kind
a `Device` was created as never changes, so it is recorded rather than inferred
from whether a handle happens to be zero.

This is worth naming because of *how* it was found: no unit test could have
caught it. It needs a live game, a consumer-owned device created inside it, and
the owned one destroyed while the game's device is still running — which is
precisely the interleaving the stress scenario was written to produce.

## Ownership

`GraphicsDevice` is the one type in the profile with **two** ownership kinds:

- **BORROWED** — the device a `Game`'s manager publishes. Every one that existed
  before this milestone. CNA-Go never releases it on its own.
- **OWNED** — one from the public constructor, released with
  `cna_graphics_device_destroy`.

CNA tells the two apart itself: the probe confirmed `cna_graphics_device_destroy`
accepts a caller-created handle and **refuses** a `Game`'s. So the two kinds take
the two routes CNA declares for them rather than sharing one and hoping.

The stress scenario creates an owned device *inside* a running game, destroys
it, and then requires the game's device to still answer — 20 of 20 cycles on both
artifacts, no crash, no double free.

## Verifier repairs

**`substitutablePositionIsGoverned`.** Making `TextureCube` a `COMPOSED` base
surfaced a `BASE_MAPPING_MISMATCH` that was a verifier bug rather than a mapping
one. The substitutable-base rule governs *parameter* positions — a base-typed
parameter widens to a reference interface; returns and read-only getters keep the
concrete pointer. The verifier was computing LIVE from **all** positions on
projected carriers, so `EffectParameter::GetValueTextureCube`'s *return*
counted. It now counts only `parameter:*`, `property-set` and `field-type`,
which is the rule as already written down.

**`property-set`.** `mapping.go` recorded `property-type` for publicly settable
properties, which is not a position the rule names. It records `property-set`.

**`checkNoActiveRenderTarget`.** `GetBackBufferData`'s render-target guard sits
behind `Helpers.CheckDisposed`, so no test without a renderer could reach it
through the exported member. Extracted as a free function — the same shape
Foundation 72's `verifyUserPrimitives` took — so the message and the exception
kind are pinned without a device, while the exported member's *order* stays
pinned by the coverage control that calls it.

## Runs

```text
go test ./...                                                      PASS
go run ./tools/behavior                     ASSERTIONS=737   FAILURES=0
go run ./tools/native_abi   BOUND_FUNCTIONS=230  LAYOUTS=457  ABI_MISMATCHES=0
go run ./tools/external_consumer -source .    TESTS=97      FAILURES=0
native_stress  HEADLESS  20/20 cycles   0 failures
native_stress  SOFTWARE  20/20 cycles   0 failures
```

Every native run used `DISPLAY=:77` (Xvfb), never the real display.
