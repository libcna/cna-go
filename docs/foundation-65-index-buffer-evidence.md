# Foundation 65 — IndexBuffer, and the copy validator every buffer shares

The first buffer type, and the first index that travels to CNA and comes back.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 165     164
MISSING_TYPE                                      121     120
COMPLETE_TYPES                                    132     133
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      136     137
TARGET_MEMBERS                                   2217    2239
BOUND_FUNCTIONS                                   104     110
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              251     271
behavior corpus                                   727     730
external canary tests                              84      85
native stress scenarios                            15      16
capability rows                                    64      65
resource strings                                   +4
```

## Ownership, and where creation can happen

```text
IndexBuffer   OWNED, generation-checked, cna_index_buffer_destroy
```

`cna_index_buffer_create` takes a **callback-scoped** device handle, so a buffer
can only be created from inside a lifecycle callback, on the device's owner OS
thread. That is where the native scenario creates both of its buffers. The
composed `GraphicsResource` holds the one CNA handle, exactly as it does for a
texture: there is one native owner per logical object and the derived wrapper
keeps only what the reference keeps on it.

## The constructors, in the reference's order

```il
.ctor(GraphicsDevice, IndexElementSize, int32 indexCount, BufferUsage)
  if (indexCount <= 0)
      throw new ArgumentOutOfRangeException("indexCount",
          FrameworkResources.ResourcesMustBeGreaterThanZeroSize);
  _parent = graphicsDevice;
  CreateBuffer(indexCount, indexElementSize == SixteenBits ? 2 : 4, ...);
```

The count check runs **before** the device is stored, which is observable: a bad
count is refused whatever the device is, and a test pins it with a nil one.

**There is no device null check.** The reference stores the argument and
dereferences it two statements later, so C# gets a `NullReferenceException`. Go
cannot project that, so a nil device is refused here by a Go-only guard **in its
own words** — not by borrowing `DeviceCannotBeNullOnResourceCreate`, which
`Texture2D`'s constructor really does throw and this one does not. A mutation
that borrows it fails.

The `Type` constructor differs in one statement — `Marshal.SizeOf(indexType)` —
and that is where CNA-Go is deliberately **narrower**:

```text
int16, uint16   -> SixteenBits
int32, uint32   -> ThirtyTwoBits
anything else   -> refused BY NAME
```

`Marshal.SizeOf` accepts any blittable type and would happily produce a buffer
whose element width XNA's own `IndexElementSize` cannot name. CNA stores 16-bit
or 32-bit indices and nothing else, so the set is closed and the refusal names
the Go type — which is more useful than letting CNA refuse a width it was never
told about.

## The three properties record what CNA APPLIED

```il
get_IndexCount        ldarg.0; ldfld _indexCount; ret
get_IndexElementSize  _indexSize == 2 ? SixteenBits : ThirtyTwoBits
get_BufferUsage       ConvertDxBufferUsageToXna(_usage)
```

None reaches D3D, none checks disposal, and all three answer after `Dispose` —
in the reference and here. That is exactly why the projection reads
`cna_index_buffer_get_info` **once**, at construction, and answers from the
recorded fields: a getter that asked CNA per call would be fallible, and the
reference's is not. The reference does the same thing for the same reason —
`_indexSize` is what `CreateBuffer` stored, not what the caller asked for — so a
renderer that widened an index is visible rather than hidden. Two mutations that
ignore CNA's report fail.

## Six generic members, one body

All six transfers are CLR generic **instance** methods, so the settled rule
projects each as a package-level generic function taking the receiver first.
Five of them are literally `call SetData(0, ...)` or `call GetData(0, ...)` in
the IL, so the projection funnels the same way.

```text
SetData<T>(T[])                 -> IndexBufferSetDataBySliceOfT
SetData<T>(T[], int, int)       -> ...BySliceOfTAndInt32AndInt32
SetData<T>(int, T[], int, int)  -> ...ByInt32AndSliceOfTAndInt32AndInt32
GetData<T>(...)                    the same three
```

`CopyData`'s guard prefix, in its order:

```text
Helpers.CheckDisposed                    ObjectDisposedException
data == null || data.Length == 0         ArgumentNullException("data")
the buffer is the device's Indices       NOT REPRODUCED -- see below
getting from a WriteOnly buffer          NotSupportedException
Helpers.ValidateCopyParameters           ArgumentOutOfRangeException x3
sizeof(T)*count + offset <= _bufferSize  InvalidOperationException
```

**One check is not reproduced, and it is recorded rather than skipped.** The
third asks whether this buffer is the device's currently bound `Indices`.
`GraphicsDevice::Indices` is not projected yet, so there is no way to ask. When
that member arrives the check has a place to live; until then a transfer to a
bound buffer reaches CNA, which answers for itself.

`Helpers::ValidateCopyParameters` is 67 bytes shared by every buffer transfer in
the reference, and it is reproduced with its **three different parameter names**
and its order. One consequence a reader would otherwise guess wrong:
`dataIndex == dataLength` passes the first check and is caught by the second,
because the third requires a positive count — so an empty window at the end of
the array is refused, not accepted.

The size check measures against the width the buffer was **created** with, not
the width of `T`. So eight 32-bit indices into a 16-bit buffer of eight are
refused by size, before CNA is asked, and a mutation that measures in the
transfer's width fails.

## The element width rule, again

The same rule the texture transfer carries, on the other side of the boundary:

> CNA identifies an index by its WIDTH, never by the Go type's size.

A `T` whose layout was not the width its identity means would be copied into a
buffer CNA reads with a different stride, and **nothing on either side would
report it**. So each transfer checks `unsafe.Sizeof(T)` against the width its
identity means, and refuses by name.

Two mutations pin it, and the pair is instructive. Widening the accepted case
list alone is **absorbed** by the size check — that is the check doing its job,
not a hole. The falsifying mutation is widening the list *and* removing the
size check, and the other is a 32-bit identity that reports a two-byte width,
which would stride half as far through CNA's buffer and still succeed.

## Evidence

A sixteenth native scenario, twenty isolated cycles, against **both** qualified
artifacts, identical on each:

```text
INDEX_BUFFER_CYCLES              20    INDEX_BUFFER_ROUND_TRIPS         20
INDEX_BUFFER_CREATIONS           20    INDEX_BUFFER_READBACK_REFUSALS    0
INDEX_BUFFER_DESCRIPTION_CHECKS  20    INDEX_BUFFER_WINDOW_ROUND_TRIPS  20
INDEX_BUFFER_REFUSALS            20    INDEX_BUFFER_WRITE_ONLY_CHECKS   20
INDEX_BUFFER_DISPOSAL_CHECKS     20
```

Each cycle writes six indices to a live CNA buffer and **reads them back from
it**, then reads two of them again through the windowed overload — which indexes
the CALLER'S array, so the first two buffer indices land at positions 2 and 3 of
the destination and the rest of the destination must be untouched. A projection
that kept a managed copy would pass a test comparing its own input; this
compares what CNA gives back.

Thirteen managed mutations were planted and **all thirteen fail**:

```text
CAUGHT  the count check admits zero
CAUGHT  the Type constructor admits a 64-bit index
CAUGHT  the nil-device refusal borrows Texture2D's message
CAUGHT  the dataIndex bound is inclusive instead of exclusive
CAUGHT  the positive-count check is dropped
CAUGHT  the buffer size is measured in the transfer's width, not the buffer's
CAUGHT  GetData no longer refuses a WriteOnly buffer
CAUGHT  the disposal check never fires
CAUGHT  the accepted element set widens AND the size check is removed
CAUGHT  a thirty-two-bit identity reports a two-byte width
CAUGHT  the constructor never installs the CLR `this`
CAUGHT  the projection ignores the width CNA applied
CAUGHT  the projection ignores the usage CNA applied
```

Three ABI prototype controls and three layout controls ship with the six routes.
The layout ones are invisible to C: the create-info's `index_element_size` and
`buffer_usage` are adjacent `uint32`s whose swap turns a 16-bit read-write buffer
into a 32-bit write-only one; the info struct's three `CNA_Bool`s are adjacent
single bytes whose swap makes a live buffer report itself content-lost; and the
transfer's `start_index` and `element_count` are adjacent `uint64`s whose swap
turns "six indices from zero" into "zero indices from six" — a transfer that
succeeds and moves nothing.

Four FrameworkResources strings join the registry. `MustBeValidIndex` is the
shared one: `ValidateCopyParameters` throws it three times with three parameter
names and one sentence, and every buffer transfer in the reference goes through
that helper — so it is already paid for when `VertexBuffer` arrives.

## What this milestone does not claim

- **Nothing is drawn.** `DrawIndexedPrimitives` and `GraphicsDevice::Indices`
  are still missing members; an index buffer here is written, read and destroyed,
  never bound.
- **`DynamicIndexBuffer` is not projected.** CNA's create-info has a `dynamic`
  flag and this projection always passes false, because the derived type is a
  separate CLR identity CNA-Go does not project. The streaming `SetDataOptions`
  belong to that type's own overloads and never cross.
- **The still-bound check is absent**, for the measured reason above.
- **`cna_index_buffer_subscribe_content_lost` is not bound.** It belongs to
  `DynamicIndexBuffer`'s `ContentLost` event, and CNA raises it only on renderer
  families that can lose a device — neither of which is a qualified artifact.
- **No `.xnb` index data was read.** The indices in the scenario are the
  scenario's own.
