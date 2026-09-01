# Foundation 59 — the four graphics state objects, and the rule they share

`BlendState`, `DepthStencilState`, `RasterizerState` and `SamplerState` are
complete. Four types, 41 value properties, 16 static presets, and one
architecture — which is why they are one milestone rather than four.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 202     198
MISSING_TYPE                                      129     125
MISSING_MEMBER                                     73      73
COMPLETE_TYPES                                    124     128
UNEXPECTED_MEMBER                                   0       0

behavior corpus                                   698     704
capability rows                                    58      59
resource strings                                   20      21
```

`MISSING_MEMBER` does not move, and should not: nothing on a partial type
closed. What closed is four whole types, and what they unblock —
`GraphicsDevice`'s state properties and `SpriteBatch`'s state-carrying `Begin`
overloads — is the next milestone's work, not this one's.

## They are GraphicsResources with no native handle

```text
.class public BlendState extends Microsoft.Xna.Framework.Graphics.GraphicsResource
```

All four derive from `GraphicsResource`, so all four compose one and inherit its
nine public members. None of them holds a native object: the reference keeps its
values in `cached*` managed fields and pushes them to D3D only when the device
applies the state, and **CNA models them the same way** — `CNA_BlendState` and
its three siblings are versioned C PODs passed by value, not handles.

```text
ownership: MANAGED_VALUE
```

The composed `GraphicsResource` therefore carries a nil resource, and that is
not a degenerate case: it is exactly the reference's own `_internalHandle == 0`
branch, where `Name` and `Tag` answer from the local fields rather than from the
device's cache. The same code path serves both, for the same reason.

## They freeze, and the reference's word for it is `isBound`

```text
set_ColorSourceBlend(Blend value)
  ThrowIfBound();
  cachedColorSourceBlend = value;

ThrowIfBound()
  if (isBound)
      throw new InvalidOperationException(String.Format(CurrentCulture,
          FrameworkResources.BoundStateObject, typeof(BlendState).Name));
```

**Every** setter calls it — including the thirteen inside `SetDefaults` and the
ones inside the private constructors. So every setter is FALLIBLE and carries a
refusal the reference really throws; every getter is one `ldfld` and carries
none.

### The static presets are frozen from construction

This is the part the documented sentence does not say. It reads "State objects
become read-only the first time they are bound to a GraphicsDevice", and each
static field is built by a **private** constructor whose last two statements are

```text
set_Name(name);
isBound = TRUE;
```

so `BlendState.AlphaBlend.ColorSourceBlend = ...` throws immediately, on an
object no device has ever seen. The public parameterless constructor ends with
`isBound = false` instead, so a consumer's own instance is mutable until bound.
Both halves are pinned.

### The exact sentence

`FrameworkResources.BoundStateObject`, read key-to-value out of
`Microsoft.Xna.Framework.dll`:

```text
Cannot change read-only {0}. State objects become read-only the first time they
are bound to a GraphicsDevice. To change property values, create a new {0}
instance.
```

`{0}` appears **twice** and takes the same argument, `typeof(T).Name`. The
resource registry's `Placeholders` flag converts Go's `%s` into `{0}`, `{1}`,
which would be wrong here, so the Go constant keeps the CLR's own spelling and
the registry compares the exact bytes the assembly holds.

## Every default and every preset, read off the IL

```text
BlendState            SetDefaults
  Color/AlphaSourceBlend       One          Color/AlphaDestinationBlend  Zero
  Color/AlphaBlendFunction     Add          ColorWriteChannels{,1,2,3}   All (15)
  BlendFactor                  Color.White  MultiSampleMask              -1

DepthStencilState     SetDefaults
  DepthBufferEnable            TRUE         DepthBufferWriteEnable       TRUE
  DepthBufferFunction          LessEqual    StencilEnable                false
  Stencil/CCW Function         Always       Stencil/CCW Pass/Fail/DepthFail  Keep
  TwoSidedStencilMode          false        StencilMask/WriteMask        -1
  ReferenceStencil             0

RasterizerState       SetDefaults
  CullMode           CullCounterClockwiseFace   FillMode         Solid
  ScissorTestEnable  false                      MultiSampleAntiAlias  TRUE
  DepthBias          0.0                        SlopeScaleDepthBias   0.0

SamplerState          SetDefaults
  Filter             Linear      AddressU/V/W        Wrap
  MaxAnisotropy      4           MaxMipLevel         0
  MipMapLevelOfDetailBias  0.0
```

Two of those are the ones a reader is most likely to assume wrongly: culling
defaults to **counter-clockwise** rather than none, and multisample antialiasing
defaults to **on**.

The sixteen presets, with the arguments each class initializer passes:

```text
BlendState(source, destination)  -- applied to BOTH channels
  Opaque           (One,         Zero)                 AlphaBlend  (One,         InverseSourceAlpha)
  Additive         (SourceAlpha, One)                  NonPremultiplied (SourceAlpha, InverseSourceAlpha)

DepthStencilState(depthEnable, depthWriteEnable)
  None (false,false)   Default (true,true)   DepthRead (true,false)

RasterizerState(cullMode)
  CullNone (None)   CullClockwise (CullClockwiseFace)   CullCounterClockwise (CullCounterClockwiseFace)

SamplerState(filter, address)  -- the ONE address goes to ALL THREE coordinates
  PointWrap (Point,Wrap)      PointClamp (Point,Clamp)
  LinearWrap (Linear,Wrap)    LinearClamp (Linear,Clamp)
  AnisotropicWrap (Anisotropic,Wrap)  AnisotropicClamp (Anisotropic,Clamp)
```

## A static field projects to a Go function

The settled static rule spells a static member as a package function prefixed by
its declaring type, so `BlendState.Opaque` is `BlendStateOpaque()`. What a
`public static initonly` FIELD promises — one object for the life of the process
— is kept by answering with the same pointer every time, so two reads compare
equal exactly as two reads of a static field do.

## The composition detail that cost four duplicated fields

Each state object holds its `*GraphicsResource` as a field **of its own**, and
`isBound` beside it, rather than through a shared holder struct. A shared holder
would put the base one level further down, where the composition verifier —
correctly — cannot see it. The rule that keeps the base private and measurable
is worth more than the four duplicated fields, and `throwIfBound` takes the flag
and the type identity as arguments so nothing else is duplicated.

## Falsification

Six mutations, each compiling:

| mutation | caught by |
| -------- | --------- |
| multisample antialiasing defaults off | `MultiSampleAntiAlias defaults to TRUE (ldc.i4.1)` |
| `MaxAnisotropy` defaults to 1 | `MaxAnisotropy = 1, want 4 (ldc.i4.4)` |
| the preset constructor does not freeze | `BlendState.AlphaBlend accepted a write` |
| `Additive` gets the reversed blend pair | `Additive source = 0/0, want 4 on both channels` |
| the refusal substitutes a constant name | the formatted-sentence test |
| `LinearClamp` wraps instead of clamping | `addresses = 0/0/0, want 1 on all three` |

## What this does NOT claim

Applying a state to a device. CNA has `cna_graphics_device_set_blend_state` and
its siblings and CNA-Go binds none of them yet, so no state reaches a renderer
in this milestone. That is `GraphicsDevice`'s work and it is next.

## Qualification

```text
gofmt / go vet / go test ./... / -race    clean
api_compat report                         198 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           704 observations, 0 failures
resource strings                          21 claimed, 21 verified, 0 findings
runtime capabilities                      59 rows, PASS
```
