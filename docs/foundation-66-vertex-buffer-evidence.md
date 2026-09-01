# Foundation 66 — VertexBuffer, IVertexType, and two measured narrowings

Vertices reach CNA and come back. Two types close, and the declaration
Foundation 64 left purely managed gets the native handle it was waiting for.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 164     162
MISSING_TYPE                                      120     118
COMPLETE_TYPES                                    133     135
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      137     139
TARGET_MEMBERS                                   2239    2262
BOUND_FUNCTIONS                                   110     119
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              271     292
behavior corpus                                   730     733
external canary tests                              85      86
native stress scenarios                            16      17
capability rows                                    65      66
resource strings                                   +5
```

## IVertexType is what makes the Type-keyed constructor real

```il
.class interface public abstract auto ansi IVertexType
  instance VertexDeclaration get_VertexDeclaration()
```

One read-only property. Five implementors ship in
`Microsoft.Xna.Framework.Graphics.dll` and every one is the same six bytes —
`ldsfld VertexDeclaration; ret` — so the contract is classified infallible on
measured agreement rather than on resemblance.

It exists here because `VertexDeclaration::FromType` — `assembly`, not public
surface — is the whole failure surface of `VertexBuffer`'s Type-keyed
constructor, and its second check is
`Activator.CreateInstance(type) as IVertexType`. Without a projected
`IVertexType`, **every Go type would fail that check** and the constructor would
be a member nothing could satisfy. With it, every step is expressible:

```text
!type.IsValueType                        -> reflect.Kind() != Struct
Activator.CreateInstance(type)           -> reflect.New(t).Elem().Interface()
... as IVertexType                       -> a Go type assertion
Marshal.SizeOf(type) != decl.VertexStride -> t.Size() != decl.VertexStride()
```

The exception identities are the reference's own and are pinned separately: the
first two checks throw `ArgumentException`, the last two `InvalidOperationException`.
A projection using one error for all four would look right and tell a consumer
the wrong kind of thing; the mutation that does so fails.

## The declaration's handle, created where a consumer appeared

Foundation 64 bound **none** of CNA's five declaration routes, on the
reachability rule: every one would have answered a question the type already
answered from the fields the reference itself reads. `cna_vertex_buffer_create`
takes a `CNA_VertexDeclarationHandle`, so this milestone is the consumer that
was missing, and the handle is created **lazily** — at the first buffer that
needs one, and shared by every later one.

```text
VertexDeclaration  OWNED, created lazily, cna_vertex_declaration_destroy
VertexBuffer       OWNED, generation-checked, cna_vertex_buffer_destroy
```

Deferring rather than creating eagerly is not a convenience:
`cna_vertex_declaration_create` needs no device and would work in the
constructor, but a declaration a consumer builds and never binds would then own
a CNA object for nothing. The lazy handle keeps the managed lifetime the
reference has and adds a native one only where a native consumer exists.

The stride is passed **explicitly, always**. CNA's stride-less route recomputes
one from the elements, which would silently turn
`new VertexDeclaration(32, elements)` over a 16-byte layout into a 16-byte
declaration. The native scenario is what catches that mutation: with the
recomputed stride the padded buffer becomes 32 bytes instead of 64 and the fill
is refused.

A buffer does **not** own its declaration. `get_VertexDeclaration` is one `ldfld`
handing back the caller's object, and disposing a buffer leaves it alive — the
reference disposes nothing it did not create.

## The RAW routes, and why they are the faithful ones

CNA publishes two upload families. The typed one packs a value of one of its
built-in `CNA_VertexType` layouts; the raw one takes bytes, a vertex count and
an explicit byte stride. XNA's `SetData<T>` is the second shape exactly: its T
is any struct and its size is `sizeof(T)`, which IS the stride. Using the typed
route would mean closing T to a set CNA named and the reference does not — so
the raw route is the faithful choice rather than the fallback.

`cna_vertex_buffer_set_data_raw_at`'s offset indexes **the buffer**, which is the
one offset in CNA's transfer family that does, and is what XNA's `offsetInBytes`
means. The index buffer's read route has no such offset, so `VertexBuffer` needs
no special case where `IndexBuffer` did.

## Two narrowings, both measured, both recorded

**A vertexStride LARGER than sizeof(T) is refused.** XNA's strided path copies
`sizeof(T)` bytes to `offsetInBytes + i*stride` and **leaves the gaps alone** —
which is the whole point of an interleaved buffer, where the gaps hold other
attributes. CNA publishes no route with a separate source and destination
stride: its `vertex_stride` describes the SOURCE vertex, so composing the padded
image on the Go side would write zeros into bytes the reference preserves.
Refusing with a message that names the reason is honest; silently zeroing a
consumer's other attributes would not be. This is distinct from the reference's
own `VertexStrideTooSmall`, which is reproduced separately and which the refusal
must not claim.

**A transfer that is not a whole number of the buffer's vertices is refused.**
This one was found by running the scenario, not by reading:

```text
cna_vertex_buffer_set_data_raw_at failed with CNA result 1:
  The vertex stride does not match this VertexBuffer's VertexDeclaration.
```

CNA describes a raw transfer in whole vertices of the buffer's **own** stride
and rejects any other stride by name. So the stride handed to CNA is always the
one CNA reported for the buffer, and the vertex count is the byte count divided
by it — four 16-byte values fill two 32-byte vertices, which is exactly what the
reference's single `memcpy` does with the same bytes. A byte count or offset
that lands part-way into a vertex has no expression in this ABI at all, and is
refused rather than rounded.

Both refusals name which side imposes the limit. Neither is a claim about XNA.

## Evidence

A seventeenth native scenario, twenty isolated cycles, against **both** qualified
artifacts, identical on each:

```text
VERTEX_BUFFER_CYCLES              20   VERTEX_BUFFER_ROUND_TRIPS         20
VERTEX_BUFFER_CREATIONS           20   VERTEX_BUFFER_READBACK_REFUSALS    0
VERTEX_BUFFER_DECLARATION_SHARES  20   VERTEX_BUFFER_OFFSET_ROUND_TRIPS  20
VERTEX_BUFFER_DESCRIPTION_CHECKS  20   VERTEX_BUFFER_FROM_TYPE_CHECKS    20
VERTEX_BUFFER_REFUSALS            20   VERTEX_BUFFER_STRIDE_CHECKS       20
VERTEX_BUFFER_DISPOSAL_CHECKS     20
```

Each cycle writes four vertices to a live CNA buffer and reads them back from
it, then reads two more **from byte 32** and requires the third and fourth. A
second buffer over the same declaration proves the handle is reused rather than
rebuilt — a second handle would be a second native owner for one managed object.
A third is built from a consumer's own `IVertexType`. A fourth, over a
declaration with an explicit 32-byte stride, proves the stride CNA applied is
what every fit check is measured in.

Thirteen mutations were planted; twelve fail in-package and the thirteenth fails
in the native scenario:

```text
CAUGHT           FromType admits a non-struct
CAUGHT           FromType admits a null declaration
CAUGHT           FromType stops comparing the size with the stride
CAUGHT           the null-declaration check throws the wrong exception identity
CAUGHT           the declaration check runs after the count check
CAUGHT           a stride below the element size is accepted
CAUGHT           a padded stride is silently sent to CNA as a packed one
CAUGHT           the buffer fit check never fires
CAUGHT           GetData no longer refuses a WriteOnly buffer
CAUGHT           the fit check uses the declaration's stride, not CNA's
CAUGHT           the constructor never installs the CLR `this`
CAUGHT           a partial-vertex transfer is sent to CNA anyway
CAUGHT (native)  the declaration lets CNA recompute the stride
```

The last is worth naming: it is invisible to every managed test, because the
recomputed and stored strides agree for any declaration whose elements fill it.
It shows up only against a declaration deliberately built wider than its
elements, on a live CNA buffer — which is what the scenario's padded case exists
for.

Three ABI prototype controls and three layout controls ship with the nine routes.
`CNA_VertexElement` is the one structure in this family that crosses as an
**array**, and its four fields are all four bytes wide, so any permutation is
byte-identical to C and describes a different layout entirely.

## What this milestone does not claim

- **Nothing is drawn.** `SetVertexBuffer`, `GetVertexBuffers` and the six
  `DrawUser*` overloads are still missing members; a vertex buffer here is
  written, read and destroyed, never bound.
- **`DynamicVertexBuffer` is not projected**, so the `dynamic` flag is always
  false and the streaming `SetDataOptions` never cross.
- **The bound-buffer check is absent**, for the same measured reason
  `IndexBuffer`'s is: `GetVertexBuffers` is not projected, so there is no way to
  ask which buffers are bound.
- **No built-in vertex struct is projected.** `VertexPositionColor` and its four
  siblings remain missing types; what this milestone provides is the interface a
  consumer's own struct implements, which is what `FromType` actually needs.
- **`Marshal.SizeOf` and `unsafe.Sizeof` are not the same function.** They agree
  for the packed structs a vertex type is, and where they could disagree Go's is
  the one that decides how many bytes move, so it is the one checked.
