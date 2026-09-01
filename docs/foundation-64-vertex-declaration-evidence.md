# Foundation 64 — VertexDeclaration, and the validator that is its whole failure surface

The first vertex type arrives, and it arrives with no native half at all.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 166     165
MISSING_TYPE                                      122     121
COMPLETE_TYPES                                    131     132
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      135     136
TARGET_MEMBERS                                   2202    2217
behavior corpus                                   720     727
external canary tests                              83      84
resource strings                                   +5
capability rows                                    63      64
```

## It reaches nothing native, and that is the reference's shape

```il
.ctor(int32 vertexStride, VertexElement[] elements)
  if (elements == null || elements.Length == 0)
      throw new ArgumentNullException("elements", FrameworkResources.NullNotAllowed);
  _elements     = (VertexElement[])elements.Clone();
  _vertexStride = vertexStride;
  VertexElementValidator.Validate(vertexStride, _elements);
```

A clone, a store and a validation. `GraphicsResource::_parent` is assigned by
`Bind`, which is `assembly` and which only the internal draw path calls, so a
declaration a consumer constructs has a **null GraphicsDevice** — and CNA-Go
answers null rather than inventing a device it never saw.

CNA publishes five declaration routes: `cna_vertex_declaration_create`,
`_create_with_stride`, `_get_stride`, `_copy_elements` and `_destroy`. **None is
bound.** Every one would answer a question this type already answers from the
fields the reference itself reads, so binding them would add routes whose only
consumer is a member that does not need them — the reachability rule, applied
before the routes exist rather than after. They become necessary the moment a
`VertexBuffer` is created FROM a declaration, and that is where they will be
bound and consumed.

```text
Ownership: MANAGED. No CNA handle, nothing to destroy, no owner thread.
```

The composed `GraphicsResource` carries a nil resource, which is exactly what
`_internalHandle == 0` means in the reference.

## The validator, check for check and in order

`VertexElementValidator` is `.class private abstract sealed` with every member
`assembly`. It is not public surface there and nothing here exports it. It is
reproduced rather than approximated because it is the **entire** failure surface
of both constructors: a projection that accepted a declaration the reference
rejects would hand a later CNA route a layout XNA never had, and one that
rejected a declaration the reference accepts would refuse a legal program.

```text
stride > 0                             ArgumentOutOfRangeException("vertexStride")
stride % 4 == 0                        VertexElementOffsetNotMultipleFour
per element, in array order:
  0 <= usage <= 12                     VertexElementBadUsage
  0 <= offset && offset+size <= stride  VertexElementOutsideStride
  offset % 4 == 0                      VertexElementOffsetNotMultipleFour
  no EARLIER element shares
    its (usage, usageIndex)            DuplicateVertexElement
  no byte it occupies is
    already claimed                    VertexElementsOverlap
```

**The order is load-bearing.** An element that is both badly-used and outside the
stride reports the usage, because that check comes first — and a caller who fixed
only the second failure would still be refused. A test pins it with an element
that is both.

**The occupancy map is the reference's own** `int[vertexStride]` initialised to
`-1` and filled with the INDEX of the element that claimed each byte. A boolean
map would report the same failures and could not say what they collided with;
the reference's message names both elements, earlier one first, and so does
this.

Five FrameworkResources strings join the registry, all in
`Microsoft.Xna.Framework.dll` rather than the Graphics assembly the type lives
in. The first is thrown at **two** sites for **two** quantities — the stride and
one element's offset — and its sentence names both, which is why one message
covers both throws rather than the projection inventing a second.

## Two clones, not one

```il
GetVertexElements
  ldarg.0; ldfld _elements; callvirt Array.Clone(); castclass; ret
```

The clone IS the member. The constructor clones on the way in so a caller cannot
invalidate a declaration the validator already accepted; `GetVertexElements`
clones on the way out so they cannot do it afterwards. Both are pinned, and both
have a mutation that fails.

## Three measured details worth stating

**An empty array takes the null branch.** `new VertexDeclaration(16)` throws
`ArgumentNullException` in C# rather than producing an element-less declaration:
the IL's `brtrue` on the reference and its `ldlen` both fall through to the same
throw. Go has no null slice distinct from an empty one and does not need one —
both are refused, which is what the reference does with both.

**The computed stride is a MAXIMUM, not a sum.** `GetVertexStride` returns the
largest END offset, so a layout with a gap strides over the gap. A projection
that summed the sizes would give 16 where the reference gives 32.

**An unknown element format occupies zero bytes.** `GetTypeSize`'s default is
`ldc.i4.0`, and that is not a failure: the element takes no space, so it neither
overflows the stride nor overlaps anything. Reproducing it as a rejection would
refuse a declaration the reference accepts.

**`params` projects to a slice, not a Go variadic.** `params` is a call-site
convenience over a parameter whose CLR type is an array, and the settled array
rule projects that to a slice. A variadic would spell this member differently
from every other array position in the profile.

## Evidence

Thirteen in-package tests, seven of them the validator's individual refusals,
plus seven behavior-corpus observations and one external-consumer test.

Twelve mutations were planted and **all twelve fail**:

```text
CAUGHT  the computed stride sums the sizes instead of taking the maximum
CAUGHT  the stride multiple-of-four check is dropped
CAUGHT  the element-offset multiple-of-four check is dropped
CAUGHT  the fits-within-stride bound is off by one
CAUGHT  the overlap check never fires
CAUGHT  the overlap message names the new element twice
CAUGHT  the duplicate-usage check never fires
CAUGHT  Vector3 is sixteen bytes instead of twelve
CAUGHT  an unknown format occupies four bytes instead of none
CAUGHT  the constructor stores the caller's array without cloning
CAUGHT  GetVertexElements returns the stored array
CAUGHT  the constructor never installs the CLR `this`
```

The fourth is the one that needed a test written for it. Offsets and sizes are
otherwise multiples of four, so no legal layout ever lands exactly one byte over
the stride — but the offset check runs AFTER the fit check, so an element at
offset 1 reaches the fit check unfiltered. A loose bound lets it through and
reports the **wrong** failure. The control is that element, and the assertion is
on which sentence comes back.

## What this milestone does not claim

- **Nothing is bound to CNA.** The five declaration routes exist and are
  deliberately unbound; the type is complete without them because the reference
  is too.
- **`Bind`, `Unbind` and `FromType` are not projected.** All three are
  `assembly`. `Unbind` is why `Dispose(bool)` has an override at all, and its one
  branch is unreachable for every declaration a consumer can construct — which is
  the state the reference is in for the same object.
- **No vertex data moves.** `VertexBuffer` and `IndexBuffer` are still missing
  types, and the eleven `GraphicsDevice` members that consume a declaration are
  still missing members.
- **`VertexDeclaration` cannot yet reach a device.** Its `GraphicsDevice` is null
  by construction, and will stay null until something binds it.
