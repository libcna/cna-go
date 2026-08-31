# Foundation 52 — DisplayMode, and Texture2D's two constructors

`DisplayMode` is complete: six members, no constructor, a new `COMPLETE_TYPE`.
`GraphicsDevice.DisplayMode` reports one, and `Texture2D` gains the two
constructors that make an **empty** texture rather than decoding bytes. Nine
missing members close and one missing type does; `TOTAL_DIAGNOSTICS` drops from
219 to 215.

## Native-sourced and pure managed are different facts

`DisplayMode` is three private fields and six members over them:

```text
get_Width          ldfld _width                                  7 bytes
get_Height         ldfld _height                                 7 bytes
get_Format         ldfld _format                                 7 bytes
get_AspectRatio    38 bytes of arithmetic over two of the fields
get_TitleSafeArea  Viewport::GetTitleSafeArea(0, 0, _width, _height)
ToString           String.Format over the four, with CurrentCulture
```

Nothing in it reaches a device, an adapter or a handle, so it is registered in
`pureManagedTypes` and every member is **infallible**. That is not in tension
with the fact that every `DisplayMode` a consumer can hold was reported by a
member that asked CNA: the member carries the error, and the value it produces is
three numbers.

The contract declares **six members and no constructor**, so a consumer cannot
build one at all. Today the only source is `GraphicsDevice.DisplayMode`;
`GraphicsAdapter`'s two are still missing with the adapter.

### Two members that a natural rewrite gets wrong

**`AspectRatio` guards both dimensions.**

```csharp
if (_height == 0 || _width == 0) return 0f;
return (float)_width / (float)_height;
```

The width half looks redundant — a zero numerator divides cleanly to `0.0`
anyway — and it is what makes the answer **defined** rather than a floating point
result. Both branches are reproduced and a test measures all three zero cases.

**`TitleSafeArea` is the whole area.** It calls `Viewport::GetTitleSafeArea`,
which on this profile is ten bytes:

```text
ldarg.0; ldarg.1; ldarg.2; ldarg.3
newobj Rectangle::.ctor(int32, int32, int32, int32)
```

No inset. A projection that applied the Xbox 80% title-safe inset would be
reproducing a different platform, and this is the same static `Viewport.
TitleSafeArea` has returned since Foundation 1.

### ToString renders the enum by name

`String.Format` boxes the `SurfaceFormat`, so the CLR renders its **name**:

```text
{Width:800 Height:480 Format:Color AspectRatio:1.6666666}
```

Formatting the Go value with `%d` or `%v` would render a number in an otherwise
correct sentence. The name comes from an **unexported** `surfaceFormatString`,
following the `vertexElementFormatString` convention, because the profile's
`SurfaceFormat` declares no `ToString` and an exported one would be public API
the contract does not have. A value no member declares renders as its decimal,
which is what the CLR does for an undefined enum value, and a coverage test
requires all twenty declared formats to have a name.

### CNA reports an aspect ratio and the projection does not use it

`CNA_DisplayMode` carries `aspect_ratio` beside the two dimensions. The
projection computes the reference's arithmetic instead, because the reference has
a **defined** answer for a zero width and a second computed value could disagree
with it. The native field still crosses the boundary, so its layout stays
measured — and it is exactly the field one of this milestone's mutation controls
attacks.

## Texture2D's two constructors

Both are thirty bytes of IL over one private `CreateTexture`:

```csharp
.ctor(GraphicsDevice, int width, int height)
  CreateTexture(device, width, height, mipMap: false, usage: 0, pool: 1,
                format: SurfaceFormat.Color)      // both ldc.i4.0

.ctor(GraphicsDevice, int width, int height, bool mipMap, SurfaceFormat format)
  CreateTexture(device, width, height, mipMap, usage: 0, pool: 1, format)
```

so the three-argument overload is the five-argument one with `false` and
`SurfaceFormat.Color`, and those defaults are read off the IL rather than chosen.
`usage` and `pool` are D3D concepts the reference passes as constants and CNA has
no parameter for.

Both wrap the call in a `.try/fault` that calls `GraphicsResource::Dispose(true)`
if it throws — the CLR's way of not leaking a half-built object. Go has no
half-built object to leak: a refused creation returns `(nil, err)`, and if the
native texture was created and the read-back then failed, `Device.CreateTexture`
disposes it before returning.

**One guard is reproduced and the rest is CNA's**, which is a boundary worth
stating. The reference's first check is

```csharp
if (graphicsDevice == null)
    throw new ArgumentNullException("graphicsDevice",
                                    FrameworkResources.DeviceCannotBeNullOnResourceCreate);
```

and CNA-Go reports that exact sentence. Everything after it — a zero dimension, a
format the renderer does not have — is validated by CNA and reported as CNA
states it. Reproducing those messages would mean reproducing D3D9's
format-capability tables, and CNA-Go would then be asserting a support decision
it did not make.

One refusal is the projection's own: a **negative** dimension. CNA takes both as
`uint32`, so `-1` would arrive as 4294967295, and the check exists so a
conversion cannot silently invent a dimension.

## ABI: four controls, and one that was deleted for not being evidence

```text                                        before      after
BOUND_FUNCTIONS                               72      74
PROTOTYPE_TYPE_POSITIONS                     222     229
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         132     146
native ABI mutation controls                  83      87
ABI_MISMATCHES / FINDINGS                      0       0
```

Two probe controls pass a versioned structure the wrong way — the create info by
value where CNA takes a pointer, and the display mode by value where the callee
**fills** it, which would silently discard every field written.

Two layout controls attack what no prototype check can see:

- **`display-mode-aspect-ratio-and-format-swapped`** — two neighbours of the same
  width and a different kind, a `float` and a `uint32`. Swapping them moves no
  size and no later offset, and the bridge's assignments convert silently in both
  directions: an aspect ratio of 1.666 would arrive as the format identity 1,
  which is `Bgr565`, and a format of 0 would arrive as an aspect ratio of 0.0.
  Only the two offsets move.
- **`texture-create-info-reserved-bytes-widened`** — a fourth reserved byte,
  which pushes `format` from offset 20 to 24 and grows the structure.

The obvious version of that last one — **dropping** `reserved[3]` entirely — was
written first and **removed**, because it is not observable: the compiler inserts
exactly those three bytes of padding to align a `uint32` after a `uint8`, so the
declared and the implied layouts are identical. A control that cannot fail is not
evidence, and it is recorded in the table rather than left as a passing test
nobody re-derives.

## Evidence

The device-state scenario grows, still 20 isolated cycles:

```text
DEVICE_STATE_DISPLAY_MODE_CHECKS   40   two per cycle
DEVICE_STATE_TEXTURE_CREATIONS     60   three per cycle
DEVICE_STATE_TEXTURE_REFUSALS      40   two per cycle
```

The second display-mode check is the one that matters: the two computed members
are required to agree with the **dimensions CNA reported**, so a projection that
took CNA's aspect ratio, or applied a title-safe inset, would fail against the
live device rather than against a constant.

The three creations are the three-argument constructor, the five-argument one
with the same defaults, and one with a mip chain at a different size; each is
read back for its dimensions and disposed. The two refusals are the nil device
carrying Microsoft's sentence and the negative dimension.

**Every counter from the other twelve scenarios is byte-identical**, which is
what makes this a strictly additive milestone.

## Scoreboard

```text                                      before    after
TARGET_TYPES                                 124      125
TARGET_MEMBERS                              1919     1928
MISSING_TYPE                                 133      132
MISSING_MEMBER                                86       83
TOTAL_DIAGNOSTICS                            219      215
COMPLETE_TYPES                               119      120
PARTIAL_TYPES                                  5        5
UNEXPECTED_MEMBER                              0        0

behavior corpus                              677      681
external canary tests                         78       79
native stress scenarios                       13       13
native ABI mutation controls                  83       87
resource strings verified                     19       20
runtime capability rows                       52       54
```

## What this milestone does not claim

- **No adapter is projected.** `GraphicsAdapter` is still missing, so
  `GraphicsAdapter.CurrentDisplayMode` and `SupportedDisplayModes` — the
  reference's other two sources of a `DisplayMode` — are missing with it, and so
  are `GraphicsDevice.Adapter` and the three `Reset` overloads.
- **`DisplayMode` reports the DEVICE's mode, not the adapter's.** The reference's
  `GraphicsDevice.DisplayMode` reads its adapter; CNA-Go asks CNA for the
  device's, through the route CNA provides. On the qualified HEADLESS artifact
  they are the same answer, and on a machine with several monitors they need not
  be.
- **`Texture2D` is still partial**, with nine members left: the five-argument
  `FromStream`, `SaveAsPng`/`SaveAsJpeg`, and the six generic `SetData`/`GetData`
  overloads that wait on the generic-method projection rule.
- **Nothing here writes a pixel into the new textures.** They are created at a
  stated size and format, read back for their dimensions, and disposed;
  `SetData` is the member that would fill one and it is not projected.
