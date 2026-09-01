# Foundation 70 — SpriteBatch's six DrawString overloads

The milestone Foundation 69 made possible, and it is deliberately small: six
argument normalisers over one native route.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 148     142
MISSING_MEMBER                                     34      28
MISSING_TYPE                                      114     114
COMPLETE_TYPES                                    139     139
PARTIAL_TYPES                                       4       4
UNEXPECTED_MEMBER                                   0       0

BOUND_FUNCTIONS                                   144     145
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              335     346
```

`SpriteBatch` stays partial. Its last two `Begin` overloads name `Effect`, which
does not exist yet, and faking them with a nil or a default effect would change
XNA semantics — so they remain missing rather than wrong.

## Six doors, one call, exactly as Draw has

Every one of the six is the same prologue:

```csharp
if (spriteFont == null) throw new ArgumentNullException("spriteFont");
if (text == null)       throw new ArgumentNullException("text");
StringProxy proxy = new StringProxy(text);
spriteFont.InternalDraw(ref proxy, this, position, color, rotation, origin,
                        ref scale, effects, depth);
```

and they differ only in which transform fields they leave at a default:

```text
(font, text, position, color)      rotation 0, origin Zero, scale Vector2.One,
                                   effects 0, depth 0
(..., float scale, ...)            scale = (scale, scale), splatted by two stflds
(..., Vector2 scale, ...)          scale as given
```

each in a `String` and a `StringBuilder` flavour. `Vector2::get_One` at IL_0029
is where the first pair's scale comes from; the two `stfld`s at IL_0035 and
IL_003e are where the uniform pair's does.

## The throw shape is NOT Draw's, and the difference is preserved

```text
Draw        ldstr "texture";    FrameworkResources.NullNotAllowed
            newobj ArgumentNullException::.ctor(string, string)
DrawString  ldstr "spriteFont"; ldstr "text"
            newobj ArgumentNullException::.ctor(string)
```

Same exception class, different constructor, no resource string. A projection
that reused Draw's message would put a Microsoft sentence on a throw site that
does not load one — which is the same class of defect Foundation 65 recorded
when `IndexBuffer` was tempted to borrow `Texture2D`'s device message.

Both argument checks come **before** the begin/end check, because that one lives
in `SpriteBatch::InternalDraw` and is not reached until the first glyph arrives.
So a null argument outside a begin/end pair reports the ARGUMENT — exactly as
`Draw`'s null texture does, for exactly the same structural reason.

## The layout is the runtime's, which is the settled SpriteBatch decision

Foundation 50 settled that `SpriteBatch::InternalDraw` is the runtime's job: the
seven `Draw` overloads normalise and CNA lays the sprite out. `DrawString` has
that shape one level up — `SpriteFont::InternalDraw` walks the string, places
each glyph and calls `SpriteBatch::InternalDraw` per glyph — and CNA offers
`cna_sprite_batch_draw_string`, whose `CNA_SpriteTextCommand` carries exactly the
nine values `InternalDraw` takes, in the same meaning:

```text
sprite_font  text  position  color  rotation  origin  scale  effects  layer_depth
```

This is a *different* answer from Foundation 69's, and the difference is
principled rather than convenient. `MeasureString` **returns a value the
consumer reads**, and the reference computes it managed-side from fields the
consumer can change, so the arithmetic must be the reference's.
`DrawString` returns nothing; its effect is pixels, and pixels are the runtime's.

### The one consequence, recorded

`SpriteFont::InternalDraw` consults `InternalMeasure` in exactly two places, and
both are conditional:

```text
SpriteEffects.FlipHorizontally   reads InternalMeasure(...).X to mirror the block
SpriteEffects.FlipVertically     reads InternalMeasure(...).Y - lineSpacing
```

Foundation 69 measured `cna_sprite_font_measure_utf8` disagreeing with
`InternalMeasure` by the last glyph's unclamped negative right bearing. So a
**flipped** string whose final glyph has a negative right bearing may be placed
up to that many pixels from where the reference would put it. Unflipped text is
unaffected, and `MeasureString` — the value a consumer can actually observe — is
the reference's own arithmetic in every case. This is the same measured
divergence, not a second one.

## What is proved, and where

```text
sprite_batch_draw_string_test.go   7 tests, every one running ALL SIX overloads:
                                   the null-font guard and its exact shape, the
                                   font-before-text order, the argument-before-
                                   state order, the nil StringBuilder on the
                                   three overloads that can reach it, the shared
                                   begin/end guard, and the coverage control
                                   that every overload reaches the resource once
                                   both guards pass
native_stress sprite-font          120 real submissions -- all six overloads, 20
                                   cycles -- through a live begin/end pair
                                   against a real CNA font, plus the three
                                   guards proved outside the pair
external canary                    all six compiled at their exact shapes from
                                   outside, including the three *strings.Builder
                                   ones
```

The HEADLESS artifact accepted every submission: `SPRITE_FONT_DRAW_STRING_SUBMITS`
is 120 and `SPRITE_FONT_DRAW_STRING_REFUSALS` is 0. That is
`VERIFIED_NATIVE_SUBMISSION`, not `VERIFIED_PIXEL`: no readback path exists on
this artifact, so no claim is made about what was rasterised.

```text
FOUNDATION_MILESTONE_70_COMPLETE=true
```
