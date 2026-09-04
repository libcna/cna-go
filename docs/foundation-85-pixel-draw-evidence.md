# Foundation 85 — the first `VERIFIED_PIXEL` draw

```text
COMPLETE_TYPES   184 -> 184        MISSING_TYPE       73 -> 73
BOUND_FUNCTIONS  305 -> 305        DELIBERATELY_UNBOUND_ROUTES  +0
```

**No type changed.** This milestone raises the STRENGTH of evidence the project
already had: every draw proof from Foundation 60 to Foundation 84 was
`VERIFIED_NATIVE_DRAW` — CNA accepted the submission and nothing could read the
result back. This one checks the texels.

## Why it could not be done before, and why it can now

Two things were missing and both arrived:

- **A readback.** Foundation 73 read the back buffer for the first time, but
  only to check a CLEAR colour. The qualified HEADLESS artifact has no readback
  path at all; the SOFTWARE artifact does.
- **A predictable output colour.** A draw only proves something if you know what
  it should look like. Foundations 79–81 projected the stock effects, and a
  `BasicEffect` with lighting, texturing, fog and vertex colour all off is a
  known solid material.

The ROADMAP has carried "upgrading `DrawUser*` from `VERIFIED_NATIVE_DRAW` to
`VERIFIED_PIXEL` is now unblocked and not yet done" since Foundation 81. It is
done.

## What the texels say

Every row below is measured on the SOFTWARE artifact, over an 800×480 back
buffer cleared to `(17, 34, 51, 255)` — a marker colour nothing else in the
process uses, so a texel that still holds it was not drawn to.

| fixture | result |
| --- | --- |
| counter-clockwise triangle, default `RasterizerState` | **nothing drawn**, 384000/384000 still the marker |
| the SAME three corners clockwise, same state | **`(255,255,255,255)`**, all 384000 |
| half-screen triangle, `CullNone` | 192080 material / 191920 marker |
| `DiffuseColor = (0,1,0)` | `(0,255,0,255)`, and no white left |
| `Alpha = 0.5` over that | **`(0,127,0,127)`** |
| `VertexColorEnabled = true`, yellow vertices | still `(0,255,0,255)` |
| `EnableDefaultLighting()` over the same material | still `(0,255,0,255)` |

Five of those are assertions and two are records. Taken one at a time:

### The default rasterizer state culls counter-clockwise, and the proof is two-sided

An empty buffer proves nothing on its own — a renderer that draws nothing ever
would give the same answer. So the claim is made with the SAME three corners in
the opposite winding order, through the same effect and the same state: one
fills the buffer and one leaves it untouched. That is the first falsifiable
statement about a default state object anywhere in the suite.

### The default material is `Vector3.One`, seen as a colour

The winding check draws with the constructor's own `DiffuseColor` and gets white.
The reference sets it with three `ldc.r4 1`, so this pins a constructor default
as a pixel fact at no extra cost: a constructor that left the colour at zero
draws black and fails.

### The geometry reaches the rasteriser

A full-screen triangle cannot be told apart from a clear. The half-screen one
covers 192080 of 384000 texels — 50.02% — and the four corner samples follow a
measured pattern: both LEFT corners inside the triangle, both RIGHT corners
outside. A draw that ignored its vertices and filled everything passes the
colour checks and fails here.

The corner pattern is *measured*, not derived. A renderer's clip-space Y
direction is its own business, and asserting a predicted orientation would have
been asserting a guess.

### `Alpha` premultiplies the colour and lands in the alpha channel

`DiffuseColor = (0,1,0)` at `Alpha = 0.5` comes back `(0,127,0,127)`. Both
numbers are `255 × 0.5` truncated, in the colour channel AND the alpha channel.
This is the strongest single assertion in the slice: nothing but a real shader
evaluation of the reference's `diffuseColor * alpha` produces that pair.

### What the renderer honours, and what it ignores

Two claims are recorded rather than asserted, and together they say something
about the artifact worth writing down.

**`VertexColorEnabled`.** The fixture's vertices are yellow and the material is
green. XNA's `BasicEffect` selects a shader that reads the vertex colour when
the flag is on, so a faithful renderer would come back yellow. CNA's software
renderer comes back **green**, and the flag does reach it: `OnApply` pushes
`cna_basic_effect_set_vertex_color_enabled` on the shader-index dirty bit and
the push succeeds.

**Lighting.** `EnableDefaultLighting()` installs the reference's three measured
rigs and turns lighting on. The same geometry and the same `DiffuseColor` come
back **unchanged**.

So the SOFTWARE artifact evaluates a FLAT MATERIAL — `DiffuseColor` and `Alpha`
— and nothing per-vertex or per-light. That is the boundary of what any pixel
claim in this project can currently assert, and it is stated here so a later
milestone does not rediscover it by writing an assertion that cannot hold.

The projection is not masking anything: both flags cross and the renderer
ignores them. Each outcome goes into two counters rather than an assertion, the
same shape Foundation 83's inside-pair `IsComplete` answer has, so a run in
which the behaviour changed moves a count instead of passing in silence. The
asserted half of the lighting check survives — the lit draw must still cover its
half of the buffer, so a rig that made the geometry vanish is still caught.

## HEADLESS cannot run it, and says so

```text
                                    HEADLESS   SOFTWARE
BACK_BUFFER_READS                          0         20
BACK_BUFFER_READ_REFUSALS                 20          0
PIXEL_DRAW_REFUSALS                       20          0
PIXEL_DRAW_WINDING_CHECKS                  0         20
PIXEL_DRAW_GEOMETRY_CHECKS                 0         20
PIXEL_DRAW_MATERIAL_CHECKS                 0         20
PIXEL_DRAW_ALPHA_CHECKS                    0         20
PIXEL_DRAW_VERTEX_COLOR_HONOURED           0          0
PIXEL_DRAW_VERTEX_COLOR_IGNORED            0         20
PIXEL_DRAW_LIGHTING_CHECKS                 0          0
PIXEL_DRAW_LIGHTING_IGNORED                0         20
```

The parent accounting pins `PIXEL_DRAW_REFUSALS` to equal
`BACK_BUFFER_READ_REFUSALS`, so the slice cannot quietly stop running on an
artifact that CAN read back.

## Planted defects

11 distinct defects, each a real way to get this wrong and each compiling. Every
one is scored against the SOFTWARE artifact, because it is the only qualified
artifact that can see a texel — a harness that took the ambient library would
have scored all eleven as surviving.

```text
PLANTED   11      KILLED   11      SKIPPED   0
```

All eleven are in a CLASS earlier milestones could not score at all. Foundation
79 wrote that its stock-effect pushes' "only observable is what the shader
draws", and Foundations 80 and 81 repeated it; none of them planted a
push-deletion defect, because there was nothing to score one against. Deleting
`BasicEffectSetDiffuseColor` from `OnApply` was invisible to every suite in the
project before this milestone and dies immediately under it.

To be exact about what did NOT change. Seven defects are still unkilled across
Foundations 79, 80 and 81, and they split three ways:

| milestone | unkilled | still unkilled here, because |
| --- | ---: | --- |
| 79 | 4 | three are equivalent mutants and one is a light-view leak with no detector; none is a push |
| 80 | 2 | **both ARE pushes** whose only observable is what the shader draws — but they are `DualTextureEffect`'s layer index and `AlphaTestEffect`'s alpha-test push, and this slice draws with a `BasicEffect` |
| 81 | 1 | CNA normalises the fourth matrix row itself, so the correction is redundant on this artifact rather than unobservable |

Foundation 80's evidence named this milestone in advance: "the same boundary
that makes `VERIFIED_PIXEL` the next thing worth building". The boundary has
moved for `BasicEffect` and not for the other five stock effects, so those two
defects are now killable in principle and are not killed here. Extending the
pixel slice to the rest of the family is the obvious next lever.

The eleven defects, by what they break:

| group | defects |
| --- | --- |
| the material pushes | `diffuse_color_is_never_pushed`, `alpha_is_never_pushed`, `diffuse_push_sends_the_emissive_colour`, `alpha_push_sends_a_constant_one` |
| the dirty flags that decide whether a push happens | `the_diffuse_setter_does_not_mark_the_material_dirty`, `on_apply_clears_the_material_flag_without_pushing` |
| the constructor's default material | `the_constructor_leaves_the_diffuse_colour_black` |
| the state objects the winding claim rests on | `cull_clockwise_and_counter_clockwise_are_swapped`, `cull_counter_clockwise_is_really_cull_none`, `the_rasterizer_state_never_reaches_cna` |
| `EffectPass.Apply`'s own contract | `apply_does_not_run_on_apply` |

## What is proved, and where

```text
native_stress presentation   20 cycles on SOFTWARE: the default rasterizer state
                             culling a counter-clockwise triangle and drawing
                             its clockwise twin, the constructor's Vector3.One
                             diffuse colour seen as white, a half-screen triangle
                             covering half the buffer with a measured corner
                             pattern, DiffuseColor deciding the texel, Alpha
                             premultiplying it into both the colour and the
                             alpha channel, and VertexColorEnabled and lighting
                             each recorded in two buckets
                             20 cycles on HEADLESS: one refusal per cycle, pinned
                             to the back buffer's own refusal count
```

```text
FOUNDATION_MILESTONE_85_COMPLETE=true
```
