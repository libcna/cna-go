# Foundation 71 — TextureCube and Texture3D, and a blocker that was false

Two types close, and the milestone starts by correcting a claim this repository
had been carrying for fifteen milestones.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 142     142
MISSING_TYPE                                      114     112
MISSING_MEMBER                                     28      28
COMPLETE_TYPES                                    139     141
PARTIAL_TYPES                                       4       4
UNEXPECTED_MEMBER                                   0       0

BOUND_FUNCTIONS                                   145     155
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              346     389
XNA_COMPOSED_DERIVED_TYPES_PROJECTED               12      14
external canary tests                              89      90
native stress scenarios                            19      20
capability rows                                    69      70
```

## The blocker that was false

`xnaBaseRelationships` recorded, against `Texture`:

> Texture3D and TextureCube need CNA volume and cube texture routes, which the
> pinned 0.21.0 ABI does not expose to this binding yet

That was written without reading `CNA/C/texture_volume.h`, which the pinned
0.21.0 header tree has always contained and which declares sixteen routes. Ten
of them are now bound. The `Texture` entry's blocker list is EMPTY — the only
empty one in the registry — because the inheritance is projected and all three
derived types are complete, and an empty list is the only honest state for a
family with nothing left to record.

The same audit found a second stale entry and corrected it: `ContentManager`'s
blocker still said "CNA-Go maps no Content/XNB subsystem", which stopped being
true in Foundation 63. Its real blocker is `ResourceContentManager`, which loads
from a `System.Resources.ResourceManager`.

## The two types

```text
TextureCube   OWNED, cna_texturecube_destroy
Texture3D     OWNED, cna_texture3d_destroy
```

Both compose `Texture`, both rebind the CLR `this` through the whole chain, and
both are usable at a `Texture`-typed parameter position — which takes the
substitutable-base rule from one live derived type to three.

`cna_texturecube_destroy` is documented as destroying a TextureCube but **not** a
RenderTargetCube, which is the same per-kind split `cna_texture2d_destroy`
already has against a render target, so each is its own resource kind with its
own destroy.

Their four geometry readers are `ldfld` over fields `InitializeDescription`
filled once from the **created** surface:

```csharp
get_Size    ldarg.0; ldfld _size;   ret
get_Width   ldarg.0; ldfld _width;  ret
get_Height  ldarg.0; ldfld _height; ret
get_Depth   ldarg.0; ldfld _depth;  ret
```

so all four are infallible and all four answer after `Dispose`, on exactly the
evidence `Texture2D.Width` already rests on.

## The element set is ONE type wide, and it is CNA's limit

```c
cna_texture2d_set_data(handle, CNA_TextureDataType, transfer, const void*, n)
cna_texturecube_set_data(handle,                    transfer, const CNA_Color*, n)
cna_texture3d_set_data  (handle,                    transfer, const CNA_Color*, n)
```

The 2D route carries a data-type identity and eighteen representations behind
it. The cube and volume routes have **no data-type parameter at all**: there is
no way to tell CNA that an array holds `Bgr565` or `HalfVector4`.

The reference accepts any `valuetype .ctor T` for all three. So the twelve
generic transfers accept `framework.Color` and refuse everything else BY NAME,
with a message that says whose limit it is:

```text
TextureCube.SetData accepts only framework.Color elements, and got
packedvector.Bgr565: CNA's cube and volume transfer routes take CNA_Color and
carry no data-type identity, unlike cna_texture2d_set_data
```

A test asserts the message names CNA and **does not** name XNA, because the one
thing this narrowing must never become is a claim about the reference. It is
the same class of limitation Foundation 66 recorded for a vertex buffer's
partial-vertex transfers.

## The whole-volume box, read off the IL

```csharp
SetData<T>(T[] data)
  SetData(0, 0, 0, _width, _height, 0, _depth, data, 0,
          data == null ? 0 : data.Length)
```

Seven int32s in the order `level, left, top, right, bottom, front, back` — with
`_height` sitting immediately before the front literal, which is the one place
the argument order is easy to get wrong. A projection that swapped `front` and
`bottom` would compile and would transfer a different box, so a test pins the
constructed transfer field by field.

The cube's short overloads pass `(face, 0, null, data, 0, length)`, and the
FACE is part of the transfer rather than of the handle — a cube is one CNA
handle and every transfer names a face, which is why all six cube members carry
a `CubeMapFace` and the type itself does not.

## What the renderers actually do

Three artifacts, one scenario, measured rather than assumed:

```text
                      cube create   cube round trip   volume create   volume round trip
HEADLESS              yes           CNA result 6      CNA result 6    --
SOFTWARE              yes           yes               CNA result 6    --
OPENGL33 (Mesa 4.6)   yes           yes               yes             yes
```

CNA's own words for the two refusals:

```text
Texture3D: this renderer does not support real volume (3D) texture storage
TextureCube::SetData: this graphics renderer did not store the complete
requested cube face region -- the face, mip level or region is not supported here
```

So the capability row is `BACKEND_BLOCKED`, and the reason is exact. It is not
an upstream gap and not a binding defect: the same ABI's OPENGL33 build creates
and round-trips both, which is what makes the measurement about the renderer
rather than about the projection. The OPENGL33 run is scenario-scoped rather
than a third full qualification, because that artifact and the pinned HEADLESS
one disagree on an unrelated scenario — its render-target readback returns
zeros under Xvfb's llvmpipe — and promoting it to a qualification artifact
needs that difference explained first, not assumed away.

## ABI

Six new manifest structures, compiler-verified against the canonical headers
with zero mismatches and 43 new layout agreements.

`CNA_TextureCubeCreateInfo` puts a `CNA_Bool` and three reserved bytes between
the size and the format, so a manifest that dropped the run would read the
format from the mip flag's address. `CNA_Texture3DTransfer` is the widest
transfer in this manifest: seven int32 box coordinates and a reserved word
before two uint64 array-window values, and the reserved word is what keeps
`start_index` eight-byte aligned. `CNA_TextureCubeTransfer` carries **two**
independent reserved runs around a `CNA_Rectangle`.

```text
FOUNDATION_MILESTONE_71_COMPLETE=true
```
