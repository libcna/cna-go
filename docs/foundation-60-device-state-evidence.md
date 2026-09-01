# Foundation 60 — applying state objects, and the two Begin overloads

The four state objects reach a device. Eight members close on two partial types
and four CNA routes are bound.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 198     190
MISSING_MEMBER                                     73      65
COMPLETE_TYPES                                    128     128
UNEXPECTED_MEMBER                                   0       0

BOUND_FUNCTIONS                                    82      86
LAYOUTS                                           185     234
behavior corpus                                   704     707
capability rows                                    59      60
```

`GraphicsDevice` goes from 54 missing members to 48; `SpriteBatch` from 10 to 8.

## The getter answers with the OBJECT

```text
get_BlendState  ldarg.0; ldfld cachedBlendState; ret
InitializeDeviceState  cachedBlendState = null; cachedDepthStencilState = null;
                       cachedRasterizerState = null;
```

Two facts follow, and they decide the whole design.

**A device answers nil until something sets one.** Not `BlendState.Opaque`, not
a device default — null, from `InitializeDeviceState`.

**The getter cannot read CNA back.** CNA holds the VALUES —
`CNA_BlendState` is a POD — and a fresh object built from them would fail the
first `device.BlendState() == myState` a consumer writes. So the cache lives on
the `GraphicsDevice` facade, which Foundation 49 already made one object per
manager per native generation, and the identity is as stable as the reference's
field.

It still refuses a **stale** facade, which the reference's device object cannot
be: `Device.Live` checks the generation before answering from managed state.

## The setter is where the freeze happens

```text
set_BlendState(value)
  if (value == null)
      throw new ArgumentNullException("value", FrameworkResources.NullNotAllowed);
  if (value == cachedBlendState && !blendStateDirty) return;
  ... end an active effect pass carrying the blend flag ...
  value.Apply(this);            // ObjectDisposedException first, then:
                                //   _parent = this;  isBound = true;
  cachedBlendState = value;
```

All four steps are reproduced. `Apply`'s two stores are why a state object
handed to **any** device is read-only from then on, which is exactly the freeze
rule Foundation 59 pinned from the other side.

The effect-pass step has no counterpart and is recorded rather than skipped
silently: CNA-Go maps no effect subsystem, so there is no active pass to end.

## Begin's nulls are defaults, and they reach the device

Every `Begin` overload funnels into the seven-argument one, which stores its
arguments and calls `SetRenderState`. That is where the nulls resolve, and its
IL names each substitute:

```text
blendState        ?? BlendState.AlphaBlend
depthStencilState ?? DepthStencilState.None
rasterizerState   ?? RasterizerState.CullCounterClockwise
samplerState      ?? SamplerState.LinearClamp     (as SamplerStates[0])
```

CNA documents the same four for its own null arguments, which is a useful
agreement and not the authority.

`SetRenderState` calls `set_BlendState` and its siblings, so after
`Begin(mode, myBlend)` the device answers `myBlend` and `myBlend` is frozen.
That is observable and is reproduced. CNA takes all four descriptors on one
route, `cna_sprite_batch_begin_with_states`, so the values reach the renderer
**once** rather than through four device calls followed by a fifth: the managed
half is performed here and the native half by the one route that needs it.

The two `Begin` overloads that also take an `Effect` stay blocked, on a shader
subsystem CNA-Go maps no part of.

## The boundary carries numbers, not structures

Four CNA state descriptors are now mirrored in CNA-Go's private manifest and
filled on the C side from flat scalars, so no CNA structure crosses cgo — the
rule the texture and sprite families already follow. `LAYOUTS` moves 185 -> 234,
every one agreeing between the canonical headers and the private manifest.

`CNA_BlendState::blend_factor` is a `CNA_Color` — four bytes inside a run of
`uint32`s — and its offset is what decides whether the constant blend colour
lands on the colour or on the multisample mask. It is pinned.

## Falsification

Four mutations, each compiling, run against the software renderer:

| mutation | caught by |
| -------- | --------- |
| the setter does not freeze | `a bound BlendState accepted a write; Apply raises isBound` |
| the getter builds a fresh object | `BlendState returned a different object` |
| `Begin` substitutes `Default` for a null depth state | `Begin did not substitute DepthStencilState.None for a null` |
| a null state is accepted as a default | `SetBlendState(nil) = <nil>, want the ArgumentNullException` |

## Native evidence, on both qualified artifacts

```text                                    HEADLESS  SOFTWARE
DEVICE_STATE_OBJECT_REFUSALS                  20        20
DEVICE_STATE_OBJECT_BINDS                     60        60
SPRITE_BATCH_STATE_BEGINS                     40        40
NATIVE_CRASHES / UAF / DOUBLE_FREE             0         0
```

Both artifacts accept every descriptor, so the state family needs no renderer
capability the headless build lacks — unlike the render-target readback.

## Qualification

```text
gofmt / go vet / go test ./... / -race    clean
api_compat report                         190 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           707 observations, 0 failures
runtime capabilities                      60 rows, PASS
native ABI                                86 bound, 234 layouts, 0 mismatches
native stress, both artifacts             0 crashes
```
