# Foundation 61 — the device's texture and sampler collections

`TextureCollection` and `SamplerStateCollection` are complete, four
`GraphicsDevice` members close, and the substitutable-base rule reaches its
second family.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 190     184
MISSING_TYPE                                      125     123
MISSING_MEMBER                                     65      61
COMPLETE_TYPES                                    128     130
UNEXPECTED_MEMBER                                   0       0

BOUND_FUNCTIONS                                    86      90
LAYOUTS                                           234     238
XNA_BASE_SUBSTITUTABILITY_LIVE                      1       2
behavior corpus                                   707     709
capability rows                                    60      61
```

`GraphicsDevice` goes from 47 missing members to 43.

## Two types, one member each

```text
.class public sealed TextureCollection extends System.Object
  .field private GraphicsDevice _parent
  .field private int32 _textureOffset
  .field assembly int32 _maxTextures
  public Texture Item[int32]        { get; set; }

.class public sealed SamplerStateCollection extends System.Object
  public SamplerState Item[int32]   { get; set; }
```

Both constructors are `assembly`, so CNA-Go declares none: only a
`GraphicsDevice` hands one out, which is the nonpublic-construction rule
`ResourceCreatedEventArgs` already follows. The device answers with the **same**
collection object every call, as its fields promise, so the facade builds each
lazily and keeps it.

`_textureOffset` is the reference's spelling of which of the two collections
this is; CNA spells the same distinction as `CNA_SHADER_STAGE_PIXEL` and
`CNA_SHADER_STAGE_VERTEX`. Both collections hold sixteen slots
(`CNA_TEXTURE_COLLECTION_MAX_TEXTURES` and `CNA_MAX_SAMPLERS`).

## The refusal must be the projection's, not CNA's

```text
if (index < 0 || index >= _maxTextures)
    throw new ArgumentOutOfRangeException("index");
```

`newobj ArgumentOutOfRangeException::.ctor(string)` on the literal `"index"` —
the parameter name and no message.

CNA refuses an out-of-range slot too, which is what makes this worth a control:
an off-by-one bound would **still refuse**, through the wrong guard, with CNA's
message. The native scenario therefore requires the projection's own sentence,
and the planted off-by-one produces

```text
Textures[16] get = cna_graphics_device_get_texture failed with CNA result 1:
The texture sampler slot is out of range., want the projection's
ArgumentOutOfRangeException
```

## The one measured divergence, and CNA's own instructions for it

XNA's texture getter reads the D3D slot and recovers the managed wrapper with
`Texture3D::GetManagedObject` and its two siblings — a pointer-to-object map.
CNA states plainly that it has none:

> There is deliberately no route from a native object back to a handle, here or
> anywhere else in this ABI. […] A slot filled by canonical CNA code — a
> SpriteBatch flush, for example — reports `bound` as `CNA_TRUE` with an invalid
> handle. […] cache what you bind and answer from the cache, and use `bound` to
> tell "something else owns this slot now" from "the slot is empty".

CNA-Go does exactly that, and the third case is **reported**:

```text
empty slot            -> nil, no error         (the reference's null)
slot CNA-Go bound     -> the texture it bound
slot CNA filled       -> an error naming the situation
```

Answering nil there would say "empty", which is false; answering the stale cache
would name the wrong texture. The refusal has no reference counterpart and is
recorded as the divergence it is.

## A base-typed return, and the downcast Go cannot express

The getter's declared type is `Texture`, the BASE. A C# consumer who bound a
`Texture2D` gets a `Texture`-typed reference and must downcast to use it as one.
CNA-Go answers with the composed `*Texture` — the same logical object, carrying
the two members `Texture` declares, which is exactly the member set the CLR's
static type gives — and a Go consumer has no downcast to recover the Texture2D.

That is a **GO_LANGUAGE_LIMITATION** at a base-typed RETURN, recorded rather
than worked around. Widening the return to `TextureReference` would take those
two members away to buy an assertion the CLR spells as a cast.

## The second live substitutability family

The indexer's SETTER takes a `Texture`, and projecting the collection put that
position on a carrier CNA-Go projects. `Texture` already had a projected derived
type, so its requirement went LIVE for exactly the reason `Texture2D`'s did in
Foundation 58, and `TextureReference` follows the same rule with no new
architecture.

The rule grew by one clause: a **property setter's value** is a parameter
position and widens with the rest. It is spelled `textureBase()` rather than
`texture()` because `Texture2D` and `RenderTarget2D` both hold a FIELD of that
name and Go has one identifier namespace for fields and methods.

```text
XNA_BASE_SUBSTITUTABILITY_LIVE        1 -> 2
XNA_BASE_SUBSTITUTABILITY_REGISTERED  1 -> 2
XNA_BASE_SUBSTITUTABILITY_LATENT      8 -> 7
```

## Falsification

Four mutations, each compiling, against the software renderer:

| mutation | caught by |
| -------- | --------- |
| the collection is rebuilt per call | `Textures returned two objects` |
| the upper bound is off by one | CNA's message where the projection's belongs |
| both stages share one collection | `Textures and VertexTextures are the same collection` |
| the sampler getter ignores its cache | `the getter answers with the object that was set` |

## Native evidence, on both qualified artifacts

```text                                       HEADLESS  SOFTWARE
DEVICE_COLLECTION_IDENTITY_CHECKS               20        20
DEVICE_COLLECTION_RANGE_CHECKS                  20        20
DEVICE_COLLECTION_TEXTURE_ROUND_TRIPS           20        20
DEVICE_COLLECTION_SAMPLER_ROUND_TRIPS           20        20
```

## Qualification

```text
gofmt / go vet / go test ./... / -race    clean
api_compat report                         184 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           709 observations, 0 failures
runtime capabilities                      61 rows, PASS
native ABI                                90 bound, 238 layouts, 0 mismatches
native stress, both artifacts             0 crashes
```
