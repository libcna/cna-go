# Foundation 72 — the Effect cluster, and the draws it unblocked

Nine types close, `SpriteBatch` closes with them, Foundation 67's recorded
blocker is removed with evidence rather than with a changed expectation, and
the six `DrawUser*` generics that blocker had been standing in front of close
too.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 142     123
MISSING_TYPE                                      112     103
MISSING_MEMBER                                     28      20
COMPLETE_TYPES                                    141     151
PARTIAL_TYPES                                       4       3
UNEXPECTED_MEMBER                                   0       0
ABI_MISMATCHES                                      0       0

BOUND_FUNCTIONS                                   155     220
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              389     426
DELIBERATELY_UNBOUND_ROUTES                        14      16
XNA_COMPOSED_DERIVED_TYPES_PROJECTED               14      15
external canary tests                              90      92
capability rows                                    70      71
```

The three types still partial are `Game` (2), `GraphicsDevice` (12) and
`GraphicsDeviceManager` (6). `SpriteBatch` left that list: its last two `Begin`
overloads named `Effect`, and faking them with a nil or a default effect would
have changed XNA semantics, so they waited for a real one.

`GraphicsDevice` went from 18 missing members to 12, because the six
`DrawUser*` generics belong to this milestone rather than the next one: they
are draw calls, and until an effect could be applied every one of them would
have been projected into the same refusal Foundation 67 recorded.

## The blocker, and how it was removed

Foundation 67 recorded:

```text
CNA result 12: GraphicsDevice::DrawPrimitives: no effect has been applied.
successful draw count = 0
refused draw count = 20
BACKEND_BLOCKED / EFFECT_DEPENDENCY
```

The `vertex-buffer` stress scenario now runs the same three draws **twice**:

```text
before any apply   DrawPrimitives -> CNA result 12, as Foundation 67 measured
after pass.Apply() DrawPrimitives, DrawIndexedPrimitives,
                   DrawInstancedPrimitives -> all three succeed
```

Across twenty isolated cycles on the qualified artifact:

```text
VERTEX_BUFFER_DRAW_REFUSALS_BEFORE_APPLY   20    the control, required
VERTEX_BUFFER_EFFECT_LOADS                 20
VERTEX_BUFFER_EFFECT_APPLIES               20
VERTEX_BUFFER_DRAWS                        20    was 0
VERTEX_BUFFER_DRAW_REFUSALS                 0    was 20
USER_PRIMITIVE_DRAWS                      120    six overloads x twenty cycles
NATIVE_CRASHES / OBSERVED_UAF / DOUBLE_FREE 0
```

The first half is a **control**, and it is required in every cycle rather than
merely accounted for. A draw that succeeded either way would prove nothing
about the effect, so the scenario fails if the pre-apply draw ever stops
refusing.

The claim is `VERIFIED_NATIVE_DRAW`, **not** `VERIFIED_PIXEL`. No qualified
artifact can read the back buffer back, so nothing is claimed about what was
rasterised.

## Which door to an Effect actually opens

`Effect`'s one public constructor takes compiled Direct3D 9 Effect Framework
bytecode, and `cna_effect_create_compiled` takes exactly that. It is gated by
`CNA_GRAPHICS_CAPABILITY_COMPILED_EFFECTS`, and the probe measured that
capability on every artifact CNA publishes here:

```text
HEADLESS   custom_effects=1  compiled_effects=0
SOFTWARE   custom_effects=1  compiled_effects=0
OPENGL33   custom_effects=1  compiled_effects=0
```

So the constructor is `BLOCKED_RENDERER` everywhere CNA-Go can be qualified,
and it reports CNA's own refusal:

```text
Compiled effect bytecode does not contain a structurally valid XNA Direct3D 9
Effect Framework header. (Parameter 'effectCode')
```

`ContentManager.Load<Effect>` is the door that opens. CNA's loader reads three
shapes — a compiled `.xnb`, a `.cnj` naming a stock effect, and a `.cnj`
carrying shader source — and **only the first is gated by that capability**. So
the stress scenario writes CNA's own descriptor:

```json
{"cnjVersion":1,"type":"BasicEffect"}
```

and gets a real, applicable effect on the qualified artifact.

## Every accessor hands out a FRESH handle, and the reference hands out one object

This is the fact the whole cluster's design rests on, and it was measured
rather than assumed:

```text
PARAMETERS view1=4294967300 view2=4294967301 same=0
ELEMENT    view1=4294967302 view2=4294967303 same=0
current1=4294967309 current2=4294967310 current_same=0
```

Two `cna_effect_get_parameters` calls answer two different owned handles. The
reference's `get_Parameters` is one `ldfld` over a `List<EffectParameter>` its
constructor built, and answers the same object forever.

So the whole reflected graph is materialised **once**, at construction, exactly
as the reference materialises its Lists — and every accessor is then a field
read. That is both the identity XNA has and the only shape that does not leak
one CNA handle per property read.

It cost one measured bug to get right. `cna_effect_get_current_technique` was
read for its handle and the handle dropped, which leaked one view per effect
and surfaced at teardown as:

```text
cna_game_destroy: All owned C child resources must be destroyed before the game.
```

Each of the eight view kinds is now its own resource kind, with its own destroy
route, so the runtime's own teardown releases them the way it releases every
other owned handle.

```text
the effect   OWNED, cna_effect_destroy
every view   OWNED, its own cna_effect_*_destroy
```

A view **outlives** its effect: the probe destroyed an effect with views alive
and the views still answered `get_count`. So there is no destruction order
between an effect and its graph, and views register under the effect's own
parent rather than under the effect.

## Two divergences, both recorded

### The three texture getters refuse

`EffectParameter::GetValueTexture2D` returns the **same** `Texture2D` object the
setter was given: the reference stores the managed reference beside the D3DX
handle. `cna_effect_parameter_get_value_texture` reports "the retained handle or
invalid handle for null" — a handle, not an object. Building a fresh facade over
it would hand back a different object with the same native half, and

```csharp
if (p.GetValueTexture2D() == myTexture)
```

would silently become false. So all three getters refuse with a message that
says so, the route is unbound under `REPRESENTATION`, and the **setter** —
the direction that loses nothing — is bound and used.

### The InvalidCastException is reproduced managed-side

Every value member of `EffectParameter` begins with

```csharp
if (_paramClass != <the class this overload reads> &&
    pElementCollection.Count == 0)
    throw new InvalidCastException();
```

a **parameterless** exception; all 49 throw sites in the type are
`newobj InvalidCastException::.ctor()` with no message. The guard is reproduced
here because the probe measured CNA **accepting** a mismatched write: setting a
`Single` on `AlphaTestEffect.DiffuseColor`, a four-column Vector parameter,
returned success and read back zero. A projection that forwarded would turn the
reference's refusal into a silent no-op.

The second half of that guard is what a reader would get wrong: a parameter with
ELEMENTS is admitted whatever its own class is, which is how
`p.GetValueSingleArray(n)` works on a Vector parameter.

## The reference's own inconsistency, preserved

```text
get_Item(string)          String.op_Equality                       ORDINAL, case SENSITIVE
GetParameterBySemantic    String.Compare(a, b, OrdinalIgnoreCase)  case INSENSITIVE
```

Three hundred lines apart in one assembly, and each is projected as it is.
`equalsOrdinalIgnoreCase` is spelled out rather than taken from
`strings.EqualFold`, which folds by UNICODE rules and would make `K` equal the
Kelvin sign U+212A — a pair `OrdinalIgnoreCase` does not fold.

Both indexers on all four collections answer **null** rather than throwing,
including for an out-of-range index. A BCL `List<T>` throws; these collections
range-check first and `ldnull; ret`.

## `List<T>.Enumerator`, mapped

The four collections return `List<T>.Enumerator` directly rather than boxing it
as `IEnumerator<T>` the way `DisplayModeCollection` does, and the verifier had
been degrading that position to `any`. It is the same contract —
`List<T>.Enumerator` **is** an `IEnumerator<T>` — so it takes the same
`Iterator[T]` projection the settled collection rule already gives.

The concrete-enumerator rule does not apply: it is for a collection that
declares its OWN public enumerator type, as `TouchCollection` does, and none of
these four declares one. One difference is recorded: `List<T>.Enumerator` is a
struct a C# caller can copy for an independent cursor, which a Go interface
value cannot express.

## `cna_sprite_batch_begin_with_states` is now SUBSUMED

All four `Begin` overloads reach `cna_sprite_batch_begin_with_effect`, because
the reference has ONE `Begin` body and CNA documents `CNA_INVALID_HANDLE` as
the stock sprite effect and a null transform as the identity — precisely what
the two short overloads supply. Keeping the narrower route bound would give one
reference path two native paths that could drift, so it joins the deliberately
unbound registry under a new class, `SUBSUMED`.

## The six DrawUser* generics

They are package functions on the settled generic-method rule, and the two
declaration-less families are eleven bytes of IL over
`VertexDeclarationFactory<T>.VertexDeclaration` — which is
`VertexDeclaration.FromType(typeof(T))`'s own cached answer, already projected
in Foundation 66. So four of the six are argument normalisers and two reach CNA.

`CNA_UserPrimitives` names five vertex sources: a raw stream at the declared
stride, and four TYPED arrays — `CNA_VertexPositionColor` and its three
siblings. CNA-Go projects none of those four value types and always has a
declaration, so every call uses `CNA_USER_VERTEX_SOURCE_RAW_STREAM` with an
explicit declaration, which expresses every layout the typed sources do and
more.

The guards are reproduced in the reference's order, and the order is
observable:

```csharp
Helpers.CheckDisposed(this, pComPtr);              // IL_0010
if (vertexData == null)         "vertexData",        NullNotAllowed
if (vertexDeclaration == null)  "vertexDeclaration", NullNotAllowed
if (primitiveCount <= 0)        "primitiveCount",    MustDrawSomething
if (primitiveCount > profile max)                    NotSupportedException  -- NOT reproduced
if (vertexOffset + vertexCount > vertexData.Length)
                                "primitiveCount",    MustBeValidIndex
if (vertexOffset out of range)  "vertexOffset",      OffsetNotValid
```

A call that fails both window checks reports `primitiveCount`, not
`vertexOffset`, and a test pins that.

`GetElementCountArray` is written out rather than asked of CNA — the route
`cna_graphics_device_get_vertex_count_for_primitives` exists — because the
reference computes it managed-side and the answer decides which of the
reference's own messages a caller sees. Asking CNA would make a managed refusal
depend on a native call.

The profile cap is not reproduced, for the reason `DrawPrimitives`' is not:
`ProfileCapabilities` is not a public XNA type, CNA-Go projects no part of it,
and there is no measured maximum to compare against.

## What is proved, and where

```text
effect_test.go              13 tests: both null-answering indexers on all four
                            collections, the case-sensitivity disagreement,
                            OrdinalIgnoreCase folding only ASCII, Apply's two
                            guards and their order, set_CurrentTechnique's
                            early return before the parent check, the cast
                            guard's element clause, the array getters' count
                            check, the row-major matrix round trip, the
                            same-object identity of every accessor, and the
                            texture getters' recorded refusal
native_stress vertex-buffer 20 cycles: the control refusal, a real stock effect
                            loaded and applied, three draws that then SUCCEED,
                            all six DrawUser* overloads submitted with the
                            effect still applied, four DrawUser guards proved,
                            and the effect disposed with its whole view graph
graphics_device_user_primitives_test.go
                            6 tests: the topology table, each guard and its
                            message and parameter, the window-before-offset
                            order, the indexed family's two extra guards, and
                            the coverage control over all six overloads
external canary             1 test compiling all nine types from outside
build-probe/f72-effect.c    the capability matrix and what each creation route
                            answers on three artifacts
build-probe/f72-lifetime.c  the fresh-handle-per-accessor measurement, the
                            view-outlives-effect measurement, and the
                            mismatched-write measurement
```

```text
FOUNDATION_MILESTONE_72_COMPLETE=true
```
