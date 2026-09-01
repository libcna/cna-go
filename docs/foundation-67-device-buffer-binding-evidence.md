# Foundation 67 — binding buffers to the device, and the three draw calls

`GraphicsDevice` goes from 28 missing members to 19 — the largest single move on
that type since Foundation 62 — and the buffers of the last three milestones
finally reach it.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 162     152
MISSING_TYPE                                      118     117
MISSING_MEMBER                                     44      35
COMPLETE_TYPES                                    135     136
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      139     140
TARGET_MEMBERS                                   2262    2278
BOUND_FUNCTIONS                                   119     124
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              292     297
behavior corpus                                   733     735
external canary tests                              86      87
capability rows                                    66      67
resource strings                                   +2
```

## The bound objects are MANAGED state, and they have to be

```il
get_Indices        ldarg.0; ldfld _currentIB; ret
GetVertexBuffers   new VertexBufferBinding[currentVertexBufferCount];
                   Array.Copy(currentVertexBuffers, copy, count); ret
```

Both answer from managed fields the setters maintain, and CNA-Go keeps the same
two fields — not as a convenience, but because it is the only way it could.
**CNA hands back a handle**, and a handle cannot be turned into the Go object a
consumer is holding without a registry that would retain every buffer for the
life of the process. The reference keeps the same fields for the same reason: it
must give back the object the caller bound, not an equivalent one.

So the two readers join `managedStoredMembers` — the first entries
`GraphicsDevice` gets that are not about a value it asked CNA for — and the
**setters** stay fallible, because binding is what actually changes the device.

`GetVertexBuffers` returns a **copy**. A caller who could mutate the array they
were handed could rewrite the device's binding table without going through a
setter, and the mutation that returns the stored slice fails.

## The null branch, which is the one a reader gets wrong

```il
SetVertexBuffer(VertexBuffer vertexBuffer)
  if (vertexBuffer != null) { binding = new VertexBufferBinding(vertexBuffer);
                              SetVertexBuffers(&binding, 1); }
  else                        SetVertexBuffers(null, 0);
```

A **null buffer unbinds everything**. It does not build a binding, it does not
throw, and in the two-argument overload it does not look at the offset either. A
non-null buffer with a bad offset is refused by the *binding constructor*, which
is why `SetVertexBuffer` has no offset validation of its own.

`cna_graphics_device_set_vertex_buffer_offset` exists and is **not bound**: the
reference reaches the array route for both overloads, so binding the
single-buffer-with-offset route would add one with no consumer. The reachability
verifier caught exactly that when it was bound speculatively, which is what that
verifier is for.

## VertexBufferBinding, and its two orderings

```il
.ctor(VertexBuffer, int32 vertexOffset, int32 instanceFrequency)
  vertexBuffer == null                        ArgumentNullException(NullNotAllowed)
  vertexOffset < 0 || >= _vertexCount         ArgumentOutOfRangeException("vertexOffset")
  instanceFrequency < 0                       ArgumentOutOfRangeException("instanceFrequency")
```

The offset bound is **exclusive** — `vertexOffset == VertexCount` is refused — and
the frequency check runs **after** both offset tests, so a caller with two bad
arguments is told about the offset. Both orderings have a mutation that fails.

Neither range refusal carries a message: the reference uses the one-argument
`ArgumentOutOfRangeException` constructor at both sites, so the parameter name is
all a caller gets, and inventing a sentence would be inventing evidence.

The type joins `pureManagedTypes` on the same native-SOURCED / pure-managed split
`DisplayMode` has: its constructors read `VertexBuffer::_vertexCount`, which is a
managed field on a native-backed type. The **buffer** carries the native error; a
struct describing where in it to start does not.

## The draw guards: three reproduced, two recorded

```text
Helpers.CheckDisposed                        reproduced
primitiveCount <= 0    MustDrawSomething     reproduced
> ProfileCapabilities.MaxPrimitiveCount      NOT reproduced
VerifyCanDraw(false, false)                  NOT reproduced
instanceStreamMask != 0                      reproduced
```

The profile cap has no public XNA type to read — `ProfileCapabilities` is
internal and CNA-Go projects no part of it — so there is no measured maximum to
compare against, and inventing one would be asserting a support decision CNA-Go
did not make.

`VerifyCanDraw` requires an effect to be applied, and that is exactly what CNA
refuses on. The **instance-frequency** refusal is reproduced, because the
projection holds the bindings it applied and can see a non-zero frequency among
them — and it is the one draw guard a consumer can trip without a shader.

`DrawInstancedPrimitives` deliberately does **not** carry it: an instanced draw
is precisely the call a non-zero frequency is for. The mutation that gives it
that refusal is caught only by the native scenario, where a real instanced
binding is applied.

## Evidence, and an honest BACKEND_BLOCKED

The vertex-buffer scenario grows, still twenty isolated cycles, against **both**
qualified artifacts with identical results:

```text
VERTEX_BUFFER_BIND_CHECKS        20   VERTEX_BUFFER_DRAWS               0
VERTEX_BUFFER_INDEX_BIND_CHECKS  20   VERTEX_BUFFER_DRAW_REFUSALS      20
VERTEX_BUFFER_DRAW_GUARD_CHECKS  20   VERTEX_BUFFER_UNBIND_CHECKS      20
```

**Zero draws and twenty refusals, and that is reported as what it is.** CNA
answers every draw with

```text
CNA result 12: GraphicsDevice::DrawPrimitives: no effect has been applied.
```

on both artifacts. That is the same requirement XNA's own `VerifyCanDraw`
imposes, and CNA-Go projects no `Effect` surface to satisfy it with — so the
capability row is `BACKEND_BLOCKED`, not `VERIFIED_NATIVE`, and the counters keep
the two outcomes apart. Everything **up to** the draw call is verified natively:
the binds reach CNA, the buffers really are bound, and the bound objects come
back as the same Go objects.

Nine mutations were planted; seven fail in-package and two fail in the native
scenario:

```text
CAUGHT           the binding offset bound is inclusive
CAUGHT           the frequency check runs before the offset check
CAUGHT           the one-argument binding stores a non-zero frequency
CAUGHT           GetVertexBuffers returns the stored table
CAUGHT           a nil buffer is refused instead of unbinding
CAUGHT           the non-instanced draws stop refusing a non-zero frequency
CAUGHT           the primitive-count guard runs after the frequency guard
CAUGHT (native)  the instanced draw gains the non-instanced refusal
CAUGHT (native)  the managed binding table is never updated
```

One earlier candidate was **withdrawn rather than counted**: giving the
one-argument binding constructor the range check the three-argument one has
cannot change behaviour for any buffer a consumer can construct, because
`VertexCount` is always positive and the offset is always zero. It is a
non-evidence mutation for the same reason dropping a padding field is, and the
control that replaced it asserts what the constructor stores instead.

`CNA_VertexBufferBinding` is the one CNA struct **array** that crosses on the
device's own surface, and its layout is pinned: a 64-bit handle followed by two
`int32`s, so moving the handle behind them is byte-identical to C and makes CNA
read a buffer token out of the offset and frequency words.

## What this milestone does not claim

- **Nothing was rendered.** Zero successful draws on either artifact, for the
  measured reason above. No pixel claim is made and none is implied.
- **The six `DrawUser*` overloads are not projected.** They take caller-supplied
  vertex arrays through `CNA_UserPrimitives`, which is a separate route family.
- **`SetRenderTargets`, `GetRenderTargets`, `Reset`, `Present` and the adapter
  members remain missing.** `GraphicsDevice` is still PARTIAL at 19.
- **No profile maximum is enforced**, because no public XNA type publishes one.
