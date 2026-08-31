# Foundation 50 — SpriteBatch's Draw family, and a verifier that read half a claim

`SpriteBatch` gains six Draw overloads and its begin/end pair, `Texture2D` gains
`Bounds`, and `Game` gains `IsActive`. Eight missing members close,
`TOTAL_DIAGNOSTICS` drops from 242 to 234, and CNA-Go binds the second of CNA's
two sprite-submission routes.

Two verifiers found defects in the work of the milestones before them, and both
are recorded below rather than quietly fixed.

## Seven overloads, one private method, two native routes

Every one of the reference's seven `Draw` overloads is a short prologue over one
private call:

```csharp
InternalDraw(Texture2D texture, ref Vector4 destination, bool scaleDestination,
             ref Nullable<Rectangle> sourceRectangle, Color color, float rotation,
             ref Vector2 origin, SpriteEffects effects, float depth)
```

and that `Vector4` means two different things:

| overload family | destination | `scaleDestination` |
| -- | -- | -- |
| position | `(position.X, position.Y, scaleX, scaleY)` | **true** |
| rectangle | `(rect.X, rect.Y, rect.Width, rect.Height)` | **false** |

Read off the seven prologues, member by member:

| # | signature tail | destination | source | rotation | origin | effects | depth |
| -- | -- | -- | -- | -- | -- | -- | -- |
| 0 | `Vector2, Color` | `(x, y, 1, 1)` | `nullRectangle` | 0 | `Vector2.Zero` | 0 | 0 |
| 1 | `Vector2, Rect?, Color` | `(x, y, 1, 1)` | argument | 0 | `Vector2.Zero` | 0 | 0 |
| 2 | `…, Single scale, …` | `(x, y, s, s)` | argument | argument | argument | argument | argument |
| 3 | `…, Vector2 scale, …` | `(x, y, s.X, s.Y)` | argument | argument | argument | argument | argument |
| 4 | `Rectangle, Color` | the rectangle | `nullRectangle` | 0 | `Vector2.Zero` | 0 | 0 |
| 5 | `Rectangle, Rect?, Color` | the rectangle | argument | 0 | `Vector2.Zero` | 0 | 0 |
| 6 | `Rectangle, Rect?, Color, …` | the rectangle | argument | argument | argument | argument | argument |

Every default in that table is read from IL rather than chosen. Overload 2 was
the one CNA-Go already had; the other six are new.

CNA declares the same split from the other side, in two structures, with a
documented reason:

> `CNA_SpriteCommand` gives a destination rectangle; the canonical API also has
> a family of Draw overloads that take a **position and a scale** instead, and
> the two are not interchangeable: with a position, `origin` is measured in
> source-texture pixels and the scale applies after that offset, which a caller
> cannot reproduce by computing a rectangle without repeating the canonical
> arithmetic.

So the projection binds both routes, and `scaleDestination` chooses between
them, exactly as it does in the IL. Overload 6 takes **no** scale at all, which
is not an omission: a destination rectangle already states the size, so the
reference's parameter list is eight arguments where its position-based sibling
is nine.

## The pair, and the order of its guards

`InternalDraw` opens with two throws, and the order is observable:

```csharp
if (texture == null)                                   // IL_0003
    throw new ArgumentNullException("texture", FrameworkResources.NullNotAllowed);
if (!this.inBeginEndPair)                              // IL_0014
    throw new InvalidOperationException(FrameworkResources.BeginMustBeCalledBeforeDraw);
```

Both conditions can hold at once, and the IL decides: a null texture outside a
pair reports the **argument**. A test and a corpus row assert exactly that,
because swapping the two is a one-line rewrite that changes what a consumer
sees.

`inBeginEndPair` is tracked managed-side, as the reference tracks it. CNA refuses
the same states itself, with `CNA_RESULT_INVALID_STATE` — the same decision, and
its own sentence, where the reference reports Microsoft's. The flag is raised
only after CNA accepts a `Begin` and cleared only after CNA accepts an `End`, so
a refused call leaves the pair exactly as it was.

CNA-Go's own disposal check runs **after** both guards, and that is the
reference's shape rather than a convenience: a disposed `SpriteBatch` still holds
its `inBeginEndPair` field, and `InternalDraw` reads it before touching anything
a disposal could have taken away. Only a nil receiver is answered first, because
it has no reference counterpart — the CLR has no null `this` — and reading the
field would panic.

Three throw sites, three different sentences, and two of them read alike:

```text
NullNotAllowed               This method does not accept null for this parameter.
BeginMustBeCalledBeforeDraw  Begin must be called successfully before a Draw can be called.
BeginMustBeCalledBeforeEnd   Begin must be called successfully before End can be called.
EndMustBeCalledBeforeBegin   Begin cannot be called again until End has been successfully called.
```

The last two are the reason the next section exists.

## A verifier that proved only half of what it claimed

Foundation 49 built `tools/resource_strings` after a message was inferred from
its resource key. It searched the retained assembly's bytes for the sentence,
which catches an invented one — and cannot catch a **real sentence filed under
the wrong key**, which is the other half of the same defect, because the key is
what names the throw site.

It was hiding one. CNA-Go recorded the audio-emitter message under

```text
DopplerScaleMustBeGreaterThanOrEqualToZero
```

a key that exists in no assembly. The sentence is real and correct; its key is
`InvalidEmitterDopplerScale`. Four milestones of substring searches passed it,
because the search never looked at a key at all.

`tools/resource_strings/resources.go` now reads the assembly's resource sets the
way `System.Resources.ResourceReader` writes them — magic `0xBEEFCACE`, header
version 2, the name section's 7-bit-length-prefixed UTF-16 names each carrying a
4-byte data offset, and the data section's string entries — and every claim is
checked **key to value**. The eight-byte alignment is measured from the start of
the set rather than from the file offset, which is the one detail that turns a
working reader into one that finds the magic, accepts the header and then reads
names out of the middle of the hash table.

Two falsifiability proofs ship with it, and one of them is the defect itself: a
registry entry claiming the real doppler sentence under the invented key must be
reported as "is not a resource key". A third test reads four keys directly and
requires the two lookalike begin/end messages to come back **different**, because
a reader that confused them would make one throw site report the other's
sentence and every other test would still pass.

This is also what settled the four strings above. Two candidate sentences in the
assembly matched "Begin cannot be called again"; only the keyed read says which
one `EndMustBeCalledBeforeBegin` is.

## A route bound and never called, one milestone later

Foundation 49 added a reachability check to `tools/native_abi` — route →
trampoline → cgo call → reachable from outside `internal/interop` — after
`cna_graphics_device_manager_dispose` outlived its only call site. It caught this
milestone's new binding on the first run, twice over, in a way worth recording:

- `cna_sprite_batch_submit_many` and `cna_sprite_batch_submit_scaled_many` differ
  only in **which command pointer they take**, so binding one where the other
  belongs is a two-token edit. Two probe mutations plant exactly that, and both
  fail to compile.
- `CNA_SpriteCommand` and `CNA_SpriteScaledCommand` carry the same handle, source,
  colour, rotation, origin, effects and depth, and differ in one member: a
  destination `CNA_Rectangle` of four `int32` against a position `CNA_Vector2`
  plus a scale `CNA_Vector2` of four `float32`. **Both are 16 bytes at the same
  offset.** Renaming one to the other moves no measurement at all, so that
  mutation is a *bridge* control — `bridge.c` writes `command.destination.x`, and
  a member that is not there is not a silent conversion. Narrowing the rectangle
  to `int16` is the opposite case: C narrows the assignment without a word, and
  only `sizeof` and the offsets after it move, so it is a *layout* control.

Twelve new layout measurements pin `CNA_SpriteCommand`, which also repaired a
pre-existing control: `sprite-command-source-and-colour-swapped` matched the new
struct first and, until the measurements existed, swapped two fields nothing was
measuring.

## Game.IsActive, and a blocker that was measuring the right fact

`get_IsActive` is not the field read its name suggests. It is 30 bytes:

```csharp
bool guideVisible = false;
if (GamerServicesDispatcher.IsInitialized) guideVisible = Guide.IsVisible;
if (!this.isActive) return false;
return !guideVisible;
```

The guide overlay makes an otherwise-active game report inactive, which is the
whole point of the property: a game does not run its simulation under the Xbox
LIVE guide.

Foundation 42 deferred the member on "`Microsoft.Xna.Framework.GamerServices.dll`
— not one of the seven pinned assemblies". That was measuring the right fact and
stopping one step early. The assembly is **retained**, so the fact can be
finished:

```text
GamerServicesDispatcher::get_IsInitialized
  ldsfld UserPacketBuffer GamerServicesDispatcher::packetBuffer
  ldnull; ceq; ldc.i4.0; ceq; ret            // packetBuffer != null
```

and `packetBuffer` has exactly **one** `stsfld` in that entire assembly, inside
`GamerServicesDispatcher::Initialize(IServiceProvider)` — a public static method
on a type CNA-Go projects no part of. There is no expressible CNA-Go program in
which it has run, so `IsInitialized` is false for every one of them and the guide
branch is unreachable code rather than a simplification. What remains is
`ldfld isActive` — and that field is already maintained by the native activation
signals, through the reference's own edge-triggered host handlers, since
Foundation 34.

The day GamerServices is projected, the member gains the branch with it.

## Texture2D.Bounds

```csharp
get_Bounds()
  ldarg.0; call get_Width
  ldarg.0; call get_Height
  newobj Rectangle::.ctor(0, 0, width, height)
```

A fresh rectangle at the origin, every call. It is **fallible** for the reason
`Width` and `Height` are — both read a disposed-checked native texture — and it
reports rather than papering over a disposed texture with an empty rectangle.

## Evidence

A twelfth native stress scenario, 20 isolated cycles, each submitting through a
real draw callback:

```text
SPRITE_DRAW_CYCLES                    20
SPRITE_DRAW_SCALED_SUBMISSIONS       100   five per cycle
SPRITE_DRAW_DESTINATION_SUBMISSIONS   80   four per cycle
SPRITE_DRAW_NULL_TEXTURE_CHECKS       20
SPRITE_DRAW_OUTSIDE_PAIR_CHECKS       40   before Begin and after End
SPRITE_DRAW_PAIR_GUARD_CHECKS         40   End before Begin, and Begin twice
SPRITE_DRAW_TEXTURE_BOUNDS_CHECKS     20
```

The two submission counters are separate, and the parent requires both: a
projection that sent every overload down one route would still submit, and would
place three of the seven wrong — which no return value reports.

Six counters from the other eleven scenarios move, each by exactly 20, and they
are the six a new 20-cycle scenario contributes: one Activated, one Exiting, one
native disposal signal, one owner-thread check, one removal check and one GC
point per cycle. **Every other counter in the report is byte-identical.**

The guards are also measured where they reach no native code at all: six
in-package tests, five behavior-corpus observations and two external-canary
tests. The canary is where the overload family is pinned by **signature**, which
an in-package test cannot do usefully — an overload family is the one place a
binding can look complete while missing a member, because every name is the same
word and only the argument list differs.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1896     1904
MISSING_MEMBER                               109      101
TOTAL_DIAGNOSTICS                            242      234
COMPLETE_TYPES                               119      119
PARTIAL_TYPES                                  5        5
UNEXPECTED_MEMBER                              0        0

behavior corpus                              667      673
external canary tests                         75       77
native stress scenarios                       11       12
native ABI mutation controls                  75       79
resource strings verified                     15       19
runtime capability rows                       49       51

BOUND_FUNCTIONS                               57       58
PROTOTYPE_TYPE_POSITIONS                     174      178
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         117      129
ABI_MISMATCHES / FINDINGS                      0        0
```

`Game` has three members left and they are two blockers: `Content` ×2 needs
`ContentManager` and the generic-method projection rule, and `LaunchParameters`
needs a BCL `Dictionary<string,string>`.

`ShowMissingRequirementMessage` is the fourth, and it is **not** projected on
purpose. Its body is a type test on `NoSuitableGraphicsDeviceException` and
`NoAudioHardwareException` — both on the `System.Exception` frontier Foundation
29 measured — and the only branch CNA-Go could express today is the one that
returns false without showing anything. That is a fake implementation of a member
whose whole job is the two branches it cannot take. CNA does have the routes it
would need (`cna_message_box_show_simple_ext`, with a test backend that records
requests instead of blocking on a person), so the member becomes buildable the day
those two exception types are.

## What this milestone does not claim

- **Nothing here proves a visible sprite.** The artifact is HEADLESS. What is
  proved is that every overload is accepted by a live native SpriteBatch inside a
  real draw callback, on the route the IL says it belongs on.
- **`SpriteBatch` is still partial.** Ten members remain: four `Begin` overloads
  needing `BlendState`, `SamplerState`, `DepthStencilState`, `RasterizerState` and
  `Effect`, and six `DrawString` overloads needing `SpriteFont`.
- **`Texture2D` is still partial.** Eleven members remain: two constructors, the
  five-argument `FromStream`, `SaveAsPng`/`SaveAsJpeg`, and the six generic
  `SetData`/`GetData` overloads, which need the generic-method projection rule
  that is not settled yet.
- **`IsActive` has no guide branch.** It cannot be wrong for any program CNA-Go
  can express, and it is not the reference's complete body. The difference is
  recorded here and in the capability inventory rather than hidden behind a
  member that looks finished.
- **The `int32`-to-`float32`-and-back round trip in the rectangle overloads is
  the reference's own.** XNA stores a destination rectangle's four `int32` fields
  into a `Vector4` with `conv.r4`; a coordinate whose magnitude exceeds 2^24 loses
  precision there, in XNA, before CNA-Go is involved.
