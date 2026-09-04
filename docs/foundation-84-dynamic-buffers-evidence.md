# Foundation 84 — DynamicVertexBuffer and DynamicIndexBuffer

```text
COMPLETE_TYPES   182 -> 184        MISSING_TYPE       75 -> 73
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  304 -> 305        DELIBERATELY_UNBOUND_ROUTES  +1
COMPOSED_XNA_BASES 6 ->   8        IDENTITY_SITES      9 -> 15
```

The graphics namespace is now complete: 184 of the profile's 257 types, and
every remaining family is outside it.

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
```

## The two types add almost nothing, and the almost is the whole milestone

Each declares two constructors, two `SetData` overloads, one property and one
event. The property is a latch, the event has one raise site, and `SetData` is
its base's `CopyData` with one extra argument. Everything else is inherited.

What made the milestone real was not the surface but three measurements that
were not in the plan.

### 1. A successful upload CLEARS the content-lost latch

`VertexBuffer::CopyData` ends its **setting** path with four instructions that
no reading of the public contract would predict:

```text
IL_02a4:  ldarg.s   isSetting
IL_02a6:  brfalse.s IL_02ce
IL_02a8:  ldarg.0
IL_02a9:  isinst    IDynamicGraphicsResource
IL_02b7:  callvirt  IDynamicGraphicsResource::SetContentLost(bool)   // false
```

`IndexBuffer::CopyData`, `Texture2D::CopyData` and `TextureCube::CopyData` carry
the same tail — the texture ones followed by
`stfld Texture::renderTargetContentsDirty`, a private field with no public
reader and no CNA counterpart, which is not projected.

So writing new data to a dynamic resource declares its content no longer lost.
A consumer whose reaction to `IsContentLost` is to re-upload sees the flag go
down **because of the upload**, and this is the behaviour that makes the
property usable at all.

Two consequences were already latent and are now fixed:

- The clear is **after** the result check, so a refused upload leaves the latch
  as it was.
- `RenderTarget2D` and `RenderTargetCube` are `IDynamicGraphicsResource` too and
  carried the latch and the event since Foundations 58 and 73 **with nothing
  that could clear or raise them**, because the interface the reference
  dispatches through was not projected. Both now carry `setContentLost`.

`IDynamicGraphicsResource` is `assembly` and not in the pinned contract, so it
is projected as an unexported Go interface with the one member the reference
dispatches on. Leaving it out lost a behaviour; exporting it would have added a
type the contract does not declare.

### 2. `SetDataOptions` is converted by a BIT TEST, not an equality

`ConvertXnaSetDataOptionsToDx` is 27 bytes:

```text
if ((options & 1) == 1) return 0x2000;   // D3DLOCK_DISCARD
if ((options & 2) == 2) return 0x1000;   // D3DLOCK_NOOVERWRITE
return 0;
```

Three things follow that the enum does not suggest:

| input | reference | why |
| --- | --- | --- |
| `Discard \| NoOverwrite` | Discard | bit 0 is tested first and returns |
| `99` | Discard | `0b1100011` has bit 0 |
| `4` | None | neither bit, and **no refusal** |

CNA numbers its three options identically — `CNA_SET_DATA_NONE` 0, `_DISCARD` 1,
`_NO_OVERWRITE` 2 — so a cast would have been right for the three named values
and wrong for everything else: CNA answers `CNA_RESULT_INVALID_ARGUMENT` for
"an undefined option" and **the reference refuses nothing**. Handing CNA the
caller's raw value would refuse where XNA silently accepts, which is a
divergence in the direction that breaks working consumer code.

`nativeSetDataOptions` projects the bit test, and the native scenario uploads
with `3` and with `99` on both buffers so the conversion has a native witness:
without it CNA refuses those two calls.

### 3. The `dynamic` flag has exactly one observable, and it is not a getter

Nothing in the public contract reports whether a buffer was created dynamic, and
`IsContentLost` answers false either way. The flag is visible only through a
refusal, which a probe against the pinned artifact measured:

```text
buffer created with dynamic=false, uploads through the options overload
  options None          accepted
  options Discard       refused
  options NoOverwrite   refused
  offset + NoOverwrite  refused
```

CNA's header says as much — "non-None values require a supported dynamic-buffer
overload". So the native scenario treats a refusal of a non-None option on a
*dynamic* buffer as a **defect rather than a capability**, and that assertion is
the only thing in the suite that can see the flag at all. Planting
`dynamic_flag_is_never_set_on_the_vertex_buffer` before the assertion existed
produced a green run; after it, the mutation dies.

## Constructor guard order, which differs between the two types

Measured one constructor at a time, because no reader would predict the split:

```text
DynamicVertexBuffer(GraphicsDevice, VertexDeclaration, Int32, BufferUsage)
    null declaration FIRST, then vertexCount <= 0

DynamicVertexBuffer(GraphicsDevice, Type, Int32, BufferUsage)
    resolves the DECLARATION first, so a bad type is reported before a bad count

DynamicIndexBuffer(GraphicsDevice, IndexElementSize, Int32, BufferUsage)
DynamicIndexBuffer(GraphicsDevice, Type, Int32, BufferUsage)
    indexCount <= 0 FIRST, in both -- the opposite of the vertex Type overload
```

A caller passing both a bad type and a bad count is told about the **type** on
the vertex side and about the **count** on the index side.

## Two composed bases, four identity sites

`VertexBuffer` and `IndexBuffer` were `DEFERRED` with a `TRANSITIVE` blocker that
stopped being true at Foundations 65 and 66. What actually kept them deferred was
that nothing derived from them. Both are now `COMPOSED` with empty blocker lists —
the third and fourth entries in the registry with nothing left to record.

Both are **middle links with sites**, and both sites were literals that were right
by accident until this milestone:

| site | reference | what a bare receiver would do |
| --- | --- | --- |
| `checkDisposed` | `ldarg.0; ldfld pComPtr; ldarg.0; call Helpers::CheckDisposed(object, native int)` | name the base in every `ObjectDisposedException`, so a disposed `DynamicVertexBuffer` would report itself as a `VertexBuffer` |
| `noteContentRestored` | `ldarg.0; isinst IDynamicGraphicsResource` | answer "not dynamic" for **every** buffer in the profile, and the latch would never clear |

`Texture2D` and `TextureCube` gained the second site for the same reason, which
is why the identity-site count moved 9 → 15 rather than 9 → 11.

## Substitutability went LIVE for both buffer bases

Both had their positions from the day they were projected —
`GraphicsDevice::SetVertexBuffer`'s two overloads, `VertexBufferBinding`'s three
constructors and its `op_Implicit`, `GraphicsDevice::set_Indices` — and were
measured `LATENT` for eighteen foundations because no projected type derived from
them. The dynamic buffers are the first, so `VertexBufferReference` and
`IndexBufferReference` join `Texture2DReference`, `TextureReference`,
`EffectReference` and `TextureCubeReference`.

The **receiver** of a generic instance method widens with them, so
`VertexBufferSetDataBySliceOfT(dynamicBuffer, data)` compiles — a
`DynamicVertexBuffer` reaches its base's `SetData` exactly as a C# caller reaches
it through inheritance.

Neither widens at RETURNS. `get_Indices` and `get_VertexBuffer` hand back a value
whose buffer members a consumer reads, not a derived identity the member exists
to produce, so the settled default holds and the lost C# downcast is recorded on
the interfaces.

## One route bound, one recorded unbound

```text
BOUND     cna_vertex_buffer_set_data_raw_at_with_options
UNBOUND   cna_vertex_buffer_set_data_raw_with_options    SUBSUMED
```

The index side needed **no new route**: `cna_index_buffer_set_data` and
`_set_data_at` already carry an `options` argument, and the static overloads pass
a hardcoded zero because the reference hardcodes `SetDataOptions.None` there. The
asymmetry is CNA's, not a projection choice.

The four `subscribe_content_lost` / `unsubscribe_content_lost` routes stay
unbound for the reason `cna_render_target_subscribe_content_lost` does, recorded
in the Go sources where that precedent lives: CNA raises them only on DirectX9,
Direct2D and Skia, and the qualified artifacts are HEADLESS and SOFTWARE.

## Planted defects

40 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   40      KILLED   34      SKIPPED   0
```

29 are killed by the in-package tests and 5 by the vertex-buffer stress child.
Three needed assertions the first pass did not have, and each is a distinct
lesson:

| defect | why the first assertion missed it |
| --- | --- |
| the dynamic flag never reaches the creation | the scenario counted an option refusal as a renderer capability, which is exactly what a non-dynamic buffer produces. The probe showed CNA refuses non-None options on a static buffer, so the refusal is the defect's signature and not a limitation |
| the buffer offset is ignored | the offset upload wrote the SAME vertices the whole-buffer upload had written, so writing them at the wrong place looked identical. A distinct marker vertex at the offset, and a readback that checks BOTH slots, is the only shape that can tell them apart |
| CNA gets the caller's raw option | every option the scenario used was a value CNA also defines, so passing it through worked. Uploading with `3` and `99` — which the reference maps by its bit test and CNA refuses by name — is what makes the conversion visible |

### Six are not killed, and each is named

| defect | why |
| --- | --- |
| `vertex_is_content_lost_does_not_latch` | the store only matters if CNA ever reports content lost, and CNA documents the field as "currently always false". Unfalsifiable without a device that can lose itself |
| `is_content_lost_never_asks_cna` | same measurement from the other side: a zero struct and CNA's answer are both false |
| `vertex_upload_drops_the_caller_options` | equivalent on a dynamic buffer. CNA accepts all three named options and the bytes that arrive are identical; the options are performance hints and D3D's own difference is a lock flag |
| `index_upload_drops_the_caller_options` | the same, on the same evidence |
| `static_upload_forgets_the_copy_data_tail` | the clear's only observable is a latch nothing in the qualified environment can set: the reference's own setter is `assembly`, and CNA never reports content lost. The tail's IDENTITY behaviour is killed in the managed suite, where the member is reachable |
| `index_upload_forgets_the_copy_data_tail` | the same |

The last two are the boundary a device that can lose itself would move — the
same shape as Foundation 83's unkilled defect, and recorded rather than argued
away.

## What a device-free test could not reach, and how it was reached anyway

`vertex_upload_clears_the_latch_before_the_result_check` needs an upload that
passes every managed guard and then **fails**, which no device-free test produces
by accident. `TestARefusedUploadLeavesTheLatchAlone` gives the buffer a native
resource that is real enough to be reached and wrong enough to refuse — a zero
`interop.Resource`, whose kind is not a vertex buffer's, so `liveHandle` refuses
it. That is the only failure mode reachable without a device, and it is exactly
the one the claim needs.

## Native qualification

```text
                                    HEADLESS   SOFTWARE
DYNAMIC_BUFFER_CREATIONS                  20         20
DYNAMIC_BUFFER_CREATION_REFUSALS           0          0
DYNAMIC_BUFFER_OPTION_UPLOADS            160        160
DYNAMIC_BUFFER_OPTION_REFUSALS             0          0
DYNAMIC_BUFFER_ROUND_TRIPS                80         80
DYNAMIC_BUFFER_CONTENT_LOST_READS         40         40
DYNAMIC_BUFFER_LATCH_CLEARS               20         20
DYNAMIC_BUFFER_BIND_CHECKS                20         20
DYNAMIC_BUFFER_DISPOSAL_CHECKS            20         20
```

Both artifacts create both buffers, accept all eight uploads per cycle including
the two undefined option values, read back what was written, answer the
content-lost read with the documented `false`, and bind and unbind a dynamic
buffer through the positions that name its base.

## What is proved, and where

```text
dynamic_buffer_test.go       11 tests: both constructor guard orders and the
                             place they diverge, the latch's short circuit,
                             SetContentLost's store-always / raise-only-on-true
                             on all four implementing types, the isinst that
                             must resolve the whole object, a refused upload
                             leaving the latch alone, the disposal message
                             naming the derived type, the bit-test conversion
                             including 3, 99 and -1, and the reference
                             interfaces including typed nils
native_stress vertex-buffer  20 cycles on two artifacts: the dynamic flag seen
                             through CNA's option refusal, every named and two
                             undefined SetDataOptions values, a windowed upload
                             proved by a marker vertex and a two-slot readback,
                             the content-lost read, binding and unbinding
                             through the base's positions, and a disposal proved
                             through a SetData that names the derived type
external canary              1 test compiling both types from outside, the two
                             reference interfaces at every position that takes
                             them, the widened generic receivers, and the one
                             Dispose an inherited public surface carries
```

```text
FOUNDATION_MILESTONE_84_COMPLETE=true
```
