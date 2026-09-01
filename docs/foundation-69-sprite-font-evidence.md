# Foundation 69 — SpriteFont, and a measurement that is the reference's

One type closes, and it closes on a decision that had to be MEASURED rather
than chosen: **CNA offers a measurement route for the type's only real method,
and the route disagrees with XNA.**

```text                                          before   after
TOTAL_DIAGNOSTICS                                 149     148
MISSING_TYPE                                      115     114
MISSING_MEMBER                                     34      34
COMPLETE_TYPES                                    138     139
PARTIAL_TYPES                                       4       4
UNEXPECTED_MEMBER                                   0       0
ABI_MISMATCHES                                      0       0

TARGET_TYPES                                      142     142
TARGET_MEMBERS                                   2301    2301
BOUND_FUNCTIONS                                   137     144
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              316     335
DELIBERATELY_UNBOUND_ROUTES                        12      14
behavior corpus                                   737     737
external canary tests                              88      89
native stress scenarios                            18      19
capability rows                                    68      69
```

`MISSING_MEMBER` is unchanged on purpose. SpriteFont's arrival makes
SpriteBatch's six `DrawString` overloads *implementable*; it does not implement
them, and reporting a member the binding does not have would be the one thing
this project refuses.

## What SpriteFont is, and what it is not

```text
.class public auto ansi sealed beforefieldinit SpriteFont
       extends [mscorlib]System.Object
```

It extends `System.Object`. It implements nothing. It declares no `Dispose`,
and it is **not** a `GraphicsResource` — so there is no inherited surface, no
`Name`, no `Tag`, no `Disposing` event and no `IsDisposed`. Six members, all
declared:

```text
MeasureString(String)             MeasureString(StringBuilder)
LineSpacing get/set               Spacing get/set
DefaultCharacter get/set          Characters get
```

Its `.ctor` is `assembly`. **Neither runtime has a public constructor for one**,
so a consumer obtains a font from `ContentManager.Load<SpriteFont>` and from
nowhere else, and CNA-Go exports no constructor either.

## Ownership: one asset, two owned handles

```text
the font    OWNED, cna_sprite_font_destroy
the atlas   OWNED, cna_texture2d_destroy, AFTER the font
```

`cna_content_manager_load_sprite_font` is the only CNA content route that
reports two owned handles from one call. XNA's SpriteFont holds the atlas in a
private `textureValue` field with no public accessor on this profile, so
CNA-Go holds it privately for the same reason: it must be released and nothing
may reach it.

The order is CNA's rule, and the probe confirmed it rather than trusting the
header:

```text
DESTROY atlas_first=3      CNA_RESULT_INVALID_STATE
DESTROY font=0             success
DESTROY atlas_after=0      success
```

Nothing in CNA-Go disposes a font, because XNA's SpriteFont is not
`IDisposable`. Both handles are released by the runtime's own teardown — which
is why `interop.LoadContentSpriteFont` registers the **atlas before the font**:
`disposeAllResources` releases in reverse registration order, so the font goes
first whatever else was registered afterwards. Registered in the natural
reading order, every game teardown that had loaded a font would report CNA's
refusal.

## The measurement, and why it is not CNA's

`cna_sprite_font_measure_utf8` exists, takes the font, takes UTF-8 and returns
a `CNA_Vector2`. Binding it would have been one line. `build-probe/f69-spritefont.c`
ran both against the same glyph table, over a three-glyph `.cnj` font whose
`'B'` carries a negative bearing on each side:

```text
'?' kerning (1, 4, 2)    crop height 8
'A' kerning (0, 5, 0)    crop height 8
'B' kerning (-3, 6, -2)  crop height 12      lineSpacing 10, spacing 1
```

```text
text      cna_sprite_font_measure_utf8   InternalMeasure from the IL
"A"       (5, 10)                        (5, 10)      agree
"?"       (7, 10)                        (7, 10)      agree
"B"       (4, 12)                        (6, 12)      DIVERGE
"AB"      (7, 12)                        (9, 12)      DIVERGE
"BA"      (10, 12)                       (10, 12)     agree
"AA"      (11, 10)                       (11, 10)     agree
"A\nA"    (5, 20)                        (5, 20)      agree
"A\r\nA"  (5, 20)                        (5, 20)      agree
"AB\nA"   (7, 20)                        (9, 20)      DIVERGE
"Z"       CNA result 1                   ArgumentException
```

Every divergence is the same two pixels and the same cause. The reference's
last statement over the width is

```csharp
result.X += Math.Max(rightBearing, 0f);
```

and CNA adds the final glyph's right bearing **unclamped**. The two agree
wherever the last glyph's right bearing is non-negative, which is exactly why
`"BA"` agrees and `"AB"` does not. CNA *does* clamp the first glyph's LEFT
bearing, so half the algorithm matches and the other half does not — which is
the shape a reader would be least likely to guess.

So the route is recorded as deliberately unbound, class `CONTRACT_DIVERGENCE`,
and `MeasureString` runs `SpriteFont::InternalMeasure` from the pinned IL over
the glyph table `cna_sprite_font_copy_glyphs` reports. **The data is CNA's and
the arithmetic is the reference's.**

### The algorithm, statement for statement

```csharp
if (text.Length == 0) return Vector2.Zero;
Vector2 result = Vector2.Zero;
result.Y = lineSpacing;
float maxWidth = 0f; int lineCount = 0; float rightBearing = 0f;
bool firstGlyphOnLine = true;
for (int i = 0; i < text.Length; i++) {
    char c = text[i];
    if (c == '\r') continue;
    if (c == '\n') {
        result.X += Math.Max(rightBearing, 0f);
        rightBearing = 0f;
        maxWidth = Math.Max(result.X, maxWidth);
        result = Vector2.Zero; result.Y = lineSpacing;
        firstGlyphOnLine = true; lineCount++;
        continue;
    }
    Vector3 k = kerning[GetIndexForCharacter(c)];
    if (firstGlyphOnLine) k.X = Math.Max(k.X, 0f);
    else                  result.X += spacing + rightBearing;
    result.X += k.X + k.Y;
    rightBearing = k.Z;
    result.Y = Math.Max(result.Y, croppingData[GetIndexForCharacter(c)].Height);
    firstGlyphOnLine = false;
}
result.X += Math.Max(rightBearing, 0f);
result.Y += lineCount * lineSpacing;
result.X = Math.Max(result.X, maxWidth);
return result;
```

Five details that are invisible from the signature and each have a pinning
test:

- **A carriage return is skipped entirely.** It advances nothing, ends no line
  and is never looked up, so `"a\r\nb"` and `"a\nb"` measure identically and a
  lone `"\r"` measures the empty line it leaves.
- **Only the first glyph on a line has its left bearing clamped.** A
  negative-bearing glyph measures differently at the start of a line than in
  the middle of one, which is why `"AB"` is narrower than `"A"` plus `"B"`.
- **`spacing` is added between glyphs only**, from the second onwards, together
  with the *previous* glyph's right bearing.
- **The height is an integer multiply.** `lineCount * lineSpacing` in `int`,
  converted to float once, added to the last line's own `Max`.
- **The trailing right bearing is clamped.** This is the statement CNA does not
  have.

### The units are UTF-16 code units

`StringProxy` indexes `System.String` and `System.Text.StringBuilder`, both
sequences of UTF-16 code units. A character outside the Basic Multilingual
Plane is **two** lookups of two surrogate code units, not one lookup of a rune,
so both overloads encode to `[]uint16` before measuring. A projection ranging
over Go runes would look up a value no 16-bit character map can hold.

## `System.Text.StringBuilder`, mapped the way `System.IO.Stream` was

The pinned contract carries `StringBuilder` at exactly four public signature
positions — this `MeasureString` and SpriteBatch's three `DrawString` shapes —
and **every one is an input the reference only reads**. `StringProxy` is the
proof: it stores the builder and consults `get_Length` and `get_Chars`, and
nothing else in the profile touches one.

That is the measurement `System.IO.Stream -> io.Reader` already rests on, so it
takes the same answer:

```json
"System.Text.StringBuilder": "*strings.Builder"
```

The standard-library Go type whose ROLE it is, not a reimplemented BCL class.
`Len` and `String` are the two reads `StringProxy` makes, and `nil` is the
reference's null. The overload identity survives: `string` and
`*strings.Builder` are different Go types, so the CLR's two-member overload set
stays two Go members — which is what the overload set means. Before this the
verifier mapped the position to `any`, which is an unmeasured degradation that
would have collapsed the two.

## The two throw sites, one sentence

`FrameworkResources.CharacterNotInFont`, read from
`Microsoft.Xna.Framework.dll`:

```text
The character '{0}' (0x{1:x4}) is not available in this SpriteFont. If
applicable, adjust the font's start and end CharacterRegions to include this
character.
```

Both placeholders take the SAME character — boxed as `Char` at `{0}` and as
`Int32` at `{1}`, whose `x4` specifier Go has no `%`-verb for — so the CLR
spelling stays in the source constant and is substituted positionally, exactly
as `BoundStateObject` and the vertex-element messages are.

It is thrown from two sites with two different exception shapes, and the
projection keeps them apart:

```csharp
set_DefaultCharacter    ArgumentException(message)
GetIndexForCharacter    ArgumentException(message, "character")
```

`GetIndexForCharacter` recurses once into the default character and the
`character != defaultCharacter.Value` test refuses a second recursion, so a
font whose default character is not in its own map throws naming **the
default**, not the character asked for.

## Three fallible setters, and the reason

`set_LineSpacing` and `set_Spacing` are `ldarg.0; ldarg.1; stfld; ret` — eight
bytes, no validation. `set_DefaultCharacter` validates and then stores. All
three are fallible in Go, for the reason Foundation 42's `Game` setters are:
the value has to reach CNA, because `cna_sprite_batch_draw_string` lays text
out from the **native** font's values and a managed-only store would let a
drawn string disagree with a measured one. The managed field is written first,
so `MeasureString` reads the reference's own state whatever CNA answers.

Two CNA narrowings are recorded rather than reworded:

```text
cna_sprite_font_set_spacing(NaN)                 CNA result 1
cna_sprite_font_set_default_character('#')       CNA result 1
```

The reference stores a non-finite spacing; CNA documents "Must be finite" and
refuses it. And CNA refuses a default character outside the font with no
message at all, which is why the projection raises the reference's own sentence
*before* reaching CNA.

## `cna_sprite_font_copy_characters`, and why one read

The probe measured it reporting exactly the character column of
`cna_sprite_font_copy_glyphs`, in the same order and the same count:

```text
CHARACTERS result=0 count=3 identical_to_glyph_column=1
```

The reference's `characterMap` is ONE list that `get_Characters` views and
`GetIndexForCharacter` binary-searches, and its index correspondence with
`kerning` and `croppingData` is the invariant every member depends on. Reading
it from two routes could produce two lists whose indices no longer correspond,
so the route is recorded unbound under a new class:

```text
REDUNDANT_READ   the route re-reports data an already bound route carries
```

`cna_sprite_font_create` is NOT in the registry, and correctly: the registry is
for a route whose XNA member IS projected, and `SpriteFont::.ctor` is
`assembly`. There is nothing to be more faithful than.

## `get_Characters` is cached by identity

```csharp
if (characters == null) characters = new ReadOnlyCollection<char>(characterMap);
return characters;
```

Built on first read and **stored**, so the second call returns the same object.
That identity is observable and the projection preserves it.
`ReadOnlyCollection<char>` needed a third element-kind constructor —
`NewReadOnlyCollectionOverCharacters` — because a font's characters are VALUES
and a constructor named "over references" would describe the opposite of what
the collection holds.

## ABI

Two new manifest structures, both compiler-verified against the canonical
headers with zero mismatches.

`CNA_SpriteFontGlyph` is the first array element in this manifest whose fields
have three different widths in one struct: two 16-byte rectangles, a `uint16`
character, a `uint16` reserved, and three floats. **The reserved half-word is
what puts the kerning vector on a four-byte boundary** — a manifest that
dropped it would read the kerning from the character's address and every glyph
after the first from the wrong offset. `CNA_SpriteFontInfo` puts a `uint16` and
a `CNA_Bool` before a five-byte reserved run, so the run's width decides the
structure's size and a manifest one byte short would fail `struct_size` on
every call.

The glyph table crosses cgo as three flat arrays — characters, eight int32s of
rectangle, three floats of kerning — because no CNA struct crosses cgo, and
element *i* of each describes the same glyph, which is the parallel-index
invariant the reference's four Lists have.

## What is proved, and where

```text
sprite_font_test.go        17 tests over a glyph table built directly:
                           the algorithm statement by statement, both clamps,
                           the carriage return, the integer line multiply, the
                           two throw sites, the cached view, the nullable
                           return, the surrogate pair, Math.Max on NaN
native_stress sprite-font  20 isolated cycles over a REAL CNA font: the load,
                           the glyph table, eight reference measurements, the
                           THREE cases CNA's own measure disagrees about, the
                           unknown-character refusal, all three setters round
                           tripped, CNA's cache answering with the descriptor
                           deleted, and a clean teardown of both handles
external canary            1 test compiling every shape from outside, including
                           the *strings.Builder overload
build-probe/f69-spritefont.c   the divergence itself, both algorithms, one table
```

The behavior corpus gains no rows, and the reason is the type's own shape:
`SpriteFont::.ctor` is `assembly`, so the corpus — which creates no device and
therefore no content manager — cannot obtain one. Adding a public constructor
to make it reachable would be a test helper on the shipped surface, which is
the refusal the index-buffer transfer guards already took.

```text
FOUNDATION_MILESTONE_69_COMPLETE=true
```
