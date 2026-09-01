# Foundation 58 — RenderTarget2D, and the substitutability question it makes live

`RenderTarget2D` is complete, and with it the profile's first **LIVE** CLR base
substitutability requirement. Foundation 40 measured every family LATENT or NONE
and concluded private composition was exactly sufficient; that conclusion was
correct and it expired the moment a derived graphics type existed.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 204     202
MISSING_TYPE                                      130     129
MISSING_MEMBER                                     74      73
COMPLETE_TYPES                                    123     124
PARTIAL_TYPES                                       4       5
UNEXPECTED_MEMBER                                   0       0
ABI_MISMATCHES                                      0       0

BOUND_FUNCTIONS                                    78      82
XNA_BASE_SUBSTITUTABILITY_LIVE                      0       1
XNA_BASE_SUBSTITUTABILITY_REGISTERED                -       1
XNA_COMPOSED_BASE_RELATIONSHIPS                     3       4
XNA_COMPOSED_IDENTITY_FORWARDS                      1       2
behavior corpus                                   694     698
capability rows                                    56      58
```

## The substitutability question, measured

```text
base                        requirement  positions  on projected carriers
Microsoft...Graphics.Texture2D   LIVE        17            9
Microsoft...Graphics.Effect      LATENT      11            2
Microsoft...Graphics.VertexBuffer LATENT      8            2
... eight more, LATENT or NONE
```

Seven of Texture2D's nine live positions are `SpriteBatch.Draw`'s `texture`
parameter; the other two are `Texture2D::FromStream`'s **return**, which needs
nothing. So the requirement is precisely: **seven parameter positions must
accept a RenderTarget2D**.

### The mechanism, and the two it is not

```go
type Texture2DReference interface {
    texture2D() *Texture2D    // unexported
}
```

**Not a conversion.** `renderTarget.AsTexture2D()` is refused for the reason the
composition rule already refuses `Base` and `Parent`: it hands a consumer the
base object, and it makes the Go call site stop looking like the C# one. With
the interface the call site is identical:

```go
spriteBatch.DrawByTexture2DAndVector2AndColor(renderTarget, position, color)
```

**Not embedding.** Embedding promotes the base's whole method set, so a member
the derived class overrides would silently keep the base's body. The verifier has
refused it since Foundation 41 and still does.

The method is unexported, so only this module can satisfy the interface: a
consumer cannot hand a SpriteBatch an object CNA never made. That is the same
guarantee the concrete `*Texture2D` gave, kept while widening the position.

### Where the rule does NOT apply

Returns and property getters keep the concrete pointer. `Texture2D::FromStream`
returns a Texture2D and a caller uses every Texture2D member on it.

An inherited member whose Go projection is already a **package function** is not
re-projected on the derived type. Two shapes qualify and both already name their
declaring type:

```text
Texture2D::FromStream          static          Texture2DFromStreamBy...
Texture2D::SetData<T>(T[])     generic instance Texture2DSetDataBySliceOfT
```

In CLR, `RenderTarget2D.FromStream(...)` and `Texture2D.FromStream(...)` are the
same member reached through two names. `renderTarget.SetData(...)` is likewise
the same member, and the generic-method rule already turned it into a function
whose first parameter is the receiver — which is a parameter position, so the
substitutable-base rule widens it and one function serves every derived value.
Projecting them again would be duplication under a second name, which the
overload and static rules exist to prevent. That removed six would-be
`RenderTarget2D...` identities and two static ones.

### Two nil shapes, one null

Go has a nil interface and a non-nil interface holding a typed nil; the CLR has
one null and one `ArgumentNullException`. `resolveTexture2D` answers nil for
both, so `Draw(nil, ...)` and `var rt *RenderTarget2D; Draw(rt, ...)` reach the
same message.

### The registry is cross-checked

`substitutableBases` decides whether a parameter widens. It is hand-written; the
requirement is derived from the pinned contract and the projected surface. A base
that becomes LIVE and is not registered would leave seven positions no derived
value can reach; one registered without being LIVE would widen a signature for a
substitution nothing can perform. Both are now `BASE_MAPPING_MISMATCH`.

## RenderTarget2D

```text
.class public RenderTarget2D extends Texture2D
       implements IDynamicGraphicsResource, IGraphicsResource, IDisposable
  .field assembly RenderTargetHelper helper
  .field assembly bool _contentLost
```

Three constructors, four declared members, sixteen inherited. The chain is four
deep and every link but the root forwards its binding, so a render target's
`ToString` answers with **its own** runtime type after three hops.

### The arguments are preferences

```text
CreateRenderTarget(...)
  if (graphicsDevice == null) throw ArgumentNullException(...)
  graphicsDevice.Adapter.QueryFormat(..., preferredFormat, preferredDepthFormat,
                                     preferredMultiSampleCount,
                                     out selected, out selectedDepth, out selectedSamples);
  Texture2D.ValidateCreationParameters(..., selected, mipMap);
  this.helper = new RenderTargetHelper(this, width, height, selected, selectedDepth,
                                       selectedSamples, usage, ...);
```

`preferredFormat` and `preferredDepthFormat` are Microsoft's own parameter names,
and the properties report the **selected** values. CNA makes the same distinction
from the other side — `CNA_RenderTargetInfo` carries the APPLIED format, depth
format, sample count and usage — so the projection reads them from there rather
than echoing the arguments back.

The three-argument constructor passes five `ldc.i4.0`, so every default is read
off the IL: no mip chain, `SurfaceFormat.Color`, `DepthFormat.None`, no
multisampling, `RenderTargetUsage.DiscardContents`.

### Ownership: a distinct CNA kind

`cna_texture2d_destroy` is documented as destroying a Texture2D **but not** a
render target. The composed GraphicsResource therefore carries the
render-target kind and destroys it through `cna_render_target_destroy` — one
native owner, the right destroy. Everything else about the chain is unchanged.

`cna_texture_*` accepts "a Texture2D or matching render-target handle", which is
the native fact the whole Go question rests on: **to CNA, a render target is a
texture handle**. `liveTextureHandle` admits both kinds so the binding does not
refuse at the boundary what CNA accepts at the ABI.

### `IsContentLost` latches, and is the one fallible property

```text
if (!_contentLost) _contentLost = _parent.IsDeviceLost;
return _contentLost;
```

`GraphicsDevice::get_IsDeviceLost` reaches D3D, so this one asks CNA and can be
refused; the other three are two field reads each and are managed stored.

## A second qualified artifact, and a real renderer

The pinned HEADLESS artifact cannot copy a colour attachment back to the CPU. A
SOFTWARE-renderer build of the same ABI was already present in `cnanext`'s build
tree, and is now retained beside the headless one:

```text
HEADLESS  ~/deps/cna-c-abi-0.21.0/libcna_c_api.so           c32bfbd3…eb731b
SOFTWARE  ~/deps/cna-c-abi-0.21.0-software/libcna_c_api.so  fe353c3b…c13f3c
```

Both report ABI 0.21.0, resolve all 82 bound symbols and produce zero ABI
mismatches. The software renderer needs no display and no Xvfb.

## The render-target semantic slice

One draw callback, twenty isolated cycles, against each artifact:

```text
create -> bind -> clear -> unbind -> READ BACK THROUGH THE Texture2D SURFACE
       -> draw through SpriteBatch -> dispose twice
```

The readback is `Texture2DGetDataBySliceOfT(renderTarget, pixels)` — the Go
spelling of a `renderTarget` flowing into a Texture2D position. It proves the
substitution and the render-target contents with one call, because the pixels
come back through the base's member.

```text                                   HEADLESS  SOFTWARE
RENDER_TARGET_CYCLES                          20        20
RENDER_TARGET_CREATIONS                       20        20
RENDER_TARGET_DESCRIPTION_CHECKS              20        20
RENDER_TARGET_SUBSTITUTION_CHECKS             20        20
RENDER_TARGET_BINDS                           20        20
RENDER_TARGET_UNBINDS                         20        20
RENDER_TARGET_PIXEL_CHECKS                     0        20
RENDER_TARGET_READBACK_REFUSALS               20         0
RENDER_TARGET_SPRITE_DRAWS                    20        20
RENDER_TARGET_DISPOSAL_CHECKS                 20        20
NATIVE_CRASHES / UAF / DOUBLE_FREE              0         0
```

Each software pixel check reads 64 pixels and requires every one to be the exact
cleared colour. The headless refusals carry CNA's own sentence:

```text
Texture2D::GetData: this graphics renderer cannot read a render target's
colour attachment back to the CPU
```

which is `CNA_RESULT_NOT_SUPPORTED`. The scenario matches on the RESULT CODE
rather than the message — the message is documentation and may be reworded,
the code is ABI — and requires binds to equal pixel checks plus readback
refusals, so a renderer that silently stopped doing either would be caught.

The stress report now records `native_library_sha256`, because two artifacts
legitimately produce different counters and a report that did not say which one
produced it could not be read at all.

## Falsification

| mutation | caught by |
| -------- | --------- |
| `SetRenderTarget` binds nothing | `render target pixel 0 = {0 0 0 0}, want the cleared {203 67 21 255}` |
| `texture2D()` yields a different Texture2D | `GetData through the Texture2D surface: CNA resource is disposed` |
| the parameter takes `*Texture2D` again | 7 `PARAMETER_MAPPING_MISMATCH`, and the compile-time test stops BUILDING |
| a LIVE base left out of `substitutableBases` | `BASE_MAPPING_MISMATCH` |
| a registered base that is not LIVE | `BASE_MAPPING_MISMATCH` |
| Texture2D's identity link swallows the binding | `BASE_MAPPING_MISMATCH` |
| RenderTarget2D's constructor does not install the CLR `this` | `BASE_MAPPING_MISMATCH` |

The first mutation is the one worth reading: without the bind, the clear reaches
the back buffer and the render target is transparent black. Nothing managed
distinguishes the two versions.

## One verifier correction the milestone forced

`writtenStreamParameters` is keyed by CLR member identity, and the inherited
`SaveAsPng` arrived under `RenderTarget2D::SaveAsPng` rather than
`Texture2D::SaveAsPng`. The registry missed it and the projection would have
declared an `io.Reader` a consumer cannot write to — Foundation 53's exact
defect, on an inherited member. Direction is now keyed by the **declaring**
member, for the same reason fallibility already is: which way a stream is used
is a property of the body.

## Qualification

```text
gofmt / go vet / go test ./... / -race    clean
api_compat report                         202 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           698 observations, 0 failures
runtime capabilities                      58 rows, PASS
native ABI (pinned headers)               82 bound, 12 deliberately unbound, 0 findings
native stress, HEADLESS                   20 render-target cycles, 0 crashes
native stress, SOFTWARE                   20 render-target cycles with pixels, 0 crashes
cna-go-template 60 frames                 PASS on both artifacts
```
