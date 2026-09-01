package graphics

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 69 — SpriteFont.
// ---------------------------------------------------------------------------

// SpriteFont is Microsoft.Xna.Framework.Graphics.SpriteFont:
//
//	.class public auto ansi sealed beforefieldinit SpriteFont
//	       extends [mscorlib]System.Object
//
// # It is NOT a GraphicsResource
//
// The class extends System.Object, implements nothing, and declares no Dispose.
// So there is no inherited surface to project, no Name, no Tag, no Disposing
// event and no IsDisposed -- and a consumer never disposes one. Its lifetime is
// the ContentManager's, which is the only thing that can produce one: its own
// `.ctor` is `assembly`, so the type has no public constructor in either
// runtime and CNA-Go exports none.
//
// # Ownership
//
//	the font   OWNED, destroyed with cna_sprite_font_destroy
//	the atlas  OWNED, destroyed with cna_texture2d_destroy, AFTER the font
//
// One asset, two owned CNA handles. The atlas is `textureValue` in the
// reference -- a private field with no public accessor in the pinned XNA 4.0
// contract -- so CNA-Go holds it privately for the same reason: it must be
// released, and nothing may reach it. CNA retains it while the font lives and
// `cna_texture2d_destroy` refuses with INVALID_STATE until the font is gone, so
// release() takes that order.
//
// # Its four private Lists are CNA's glyph table
//
// The reference holds
//
//	List<Rectangle> glyphData      List<Rectangle> croppingData
//	List<char>      characterMap   List<Vector3>   kerning
//
// four PARALLEL lists indexed by one glyph index, and every member reads them.
// `cna_sprite_font_copy_glyphs` returns exactly that table, one element per
// glyph carrying all four values, and its documentation says element i
// describes the character at index i. The projection reads it ONCE, at
// construction, into one slice of glyphs -- which preserves the parallel-index
// invariant by construction rather than by two reads that could disagree.
type SpriteFont struct {
	// font and texture are the two owned CNA handles.
	font    *interop.Resource
	texture *interop.Resource

	// glyphs is the reference's four parallel Lists, held as one table. It is
	// sorted ascending by character, which is what GetIndexForCharacter's
	// binary search requires and what CNA's `.cnj` reader enforces.
	glyphs []interop.SpriteFontGlyph

	// The three mutable fields, held managed because the reference holds them
	// managed and every reader of them is a managed body.
	lineSpacing      int32
	spacing          float32
	defaultCharacter *uint16

	// characters is get_Characters's lazily built, then CACHED, view. The
	// reference builds it on first read and stores it, so the second call
	// returns the SAME object; this field is that store.
	characters *framework.ReadOnlyCollection[uint16]
}

// errSpriteFontNil is the Go-only guard for a zero SpriteFont.
var errSpriteFontNil = errors.New("SpriteFont is nil or uninitialized")

// characterNotInFont is FrameworkResources.CharacterNotInFont, the one message
// SpriteFont's own members throw. It is thrown from TWO sites with two
// different exception shapes:
//
//	set_DefaultCharacter   ArgumentException(message)
//	GetIndexForCharacter   ArgumentException(message, "character")
//
// and both format it with the SAME char twice -- boxed as Char at {0} and as
// Int32 at {1}. The CLR spelling is kept in the constant and substituted
// positionally, for the reason every other multiply-substituted message here
// is: `%s` would make the two placeholders indistinguishable, and `{1:x4}`
// carries a format specifier Go has no equivalent for.
const characterNotInFont = "The character '{0}' (0x{1:x4}) is not available in this SpriteFont. If applicable, adjust the font's start and end CharacterRegions to include this character."

// formatCharacterNotInFont substitutes the two placeholders the way
// String.Format(CurrentCulture, ...) does for a boxed Char at {0} and a boxed
// Int32 at {1}. `{1:x4}` is Int32.ToString("x4"): lowercase hex, at least four
// digits.
func formatCharacterNotInFont(character uint16) string {
	message := strings.ReplaceAll(characterNotInFont, "{0}", string(utf16.Decode([]uint16{character})))
	return strings.ReplaceAll(message, "{1:x4}", fmt.Sprintf("%04x", character))
}

// newSpriteFont builds the projection over the two handles CNA's loader
// reported and the snapshot it took of them.
func newSpriteFont(font, texture *interop.Resource, info interop.SpriteFontInfo, glyphs []interop.SpriteFontGlyph) *SpriteFont {
	value := &SpriteFont{
		font:        font,
		texture:     texture,
		glyphs:      glyphs,
		lineSpacing: info.LineSpacing,
		spacing:     info.Spacing,
	}
	if info.HasDefaultCharacter {
		character := info.DefaultCharacter
		value.defaultCharacter = &character
	}
	return value
}

// nativeFont is the owned CNA font handle, for this package's own operations.
// Unexported; it never escapes.
func (f *SpriteFont) nativeFont() *interop.Resource {
	if f == nil {
		return nil
	}
	return f.font
}

// # Nothing here releases either handle, and that is the reference's shape
//
// XNA's SpriteFont declares no Dispose and implements no IDisposable, so a
// consumer never releases one and this projection exposes no way to. Both CNA
// handles are owned by the runtime's resource table and released by the game's
// own teardown -- in the order CNA requires, because
// interop.LoadContentSpriteFont registers the atlas BEFORE the font and the
// teardown releases in reverse. The atlas field exists so the handle is
// reachable at all; nothing reads it.

// LineSpacing is SpriteFont::get_LineSpacing:
//
//	ldarg.0; ldfld lineSpacing; ret
//
// seven bytes over a managed field.
func (f *SpriteFont) LineSpacing() int32 {
	if f == nil {
		return 0
	}
	return f.lineSpacing
}

// SetLineSpacing is SpriteFont::set_LineSpacing, whose whole reference body is
//
//	ldarg.0; ldarg.1; stfld lineSpacing; ret
//
// eight bytes with NO validation: a negative line spacing is stored, and
// MeasureString then answers with it.
//
// It is fallible here for the reason Game's `stfld` setters are: the value has
// to reach CNA as well, because cna_sprite_batch_draw_string lays text out from
// the NATIVE font's spacing and a managed-only store would make a drawn string
// disagree with a measured one. The managed field is written first, so the
// reference's own state is what MeasureString reads whatever CNA answers.
func (f *SpriteFont) SetLineSpacing(value int32) error {
	if f == nil {
		return errSpriteFontNil
	}
	f.lineSpacing = value
	return f.font.SetSpriteFontLineSpacing(value)
}

// Spacing is SpriteFont::get_Spacing, one `ldfld`.
func (f *SpriteFont) Spacing() float32 {
	if f == nil {
		return 0
	}
	return f.spacing
}

// SetSpacing is SpriteFont::set_Spacing, one `stfld` with no validation, and
// fallible here for the same reason SetLineSpacing is.
//
// CNA narrows it: cna_sprite_font_set_spacing documents "Must be finite" and
// refuses a NaN or an infinity the reference would store. That is a CNA ABI
// restriction rather than an XNA one, and it is reported as CNA's own refusal
// rather than reworded into an XNA message the reference never throws.
func (f *SpriteFont) SetSpacing(value float32) error {
	if f == nil {
		return errSpriteFontNil
	}
	f.spacing = value
	return f.font.SetSpriteFontSpacing(value)
}

// DefaultCharacter is SpriteFont::get_DefaultCharacter, one `ldfld` of a
// Nullable<char>.
//
// A nullable RETURN projects as the value then a has-value bool, on the settled
// nullable rule, and it is by VALUE for the reason the reference's is: a
// Nullable<char> is a value type its getter copies, so nothing a caller does
// with the result can change the font's state.
func (f *SpriteFont) DefaultCharacter() (uint16, bool) {
	if f == nil || f.defaultCharacter == nil {
		return 0, false
	}
	return *f.defaultCharacter, true
}

// SetDefaultCharacter is SpriteFont::set_DefaultCharacter:
//
//	if (value.HasValue && !characterMap.Contains(value.Value))
//	    throw new ArgumentException(String.Format(CurrentCulture,
//	        FrameworkResources.CharacterNotInFont, value.Value, (int)value.Value));
//	this.defaultCharacter = value;
//
// Two details the IL settles that a reader would otherwise guess wrong.
//
// Clearing it is ALWAYS legal: the guard is `HasValue && !Contains`, so a null
// skips the check entirely. And the containment test is `List<char>.Contains`,
// a LINEAR scan over the character map -- not the binary search
// GetIndexForCharacter uses -- so it finds a character wherever it sits, which
// for the sorted map CNA produces is the same answer.
//
// The refusal is raised HERE, before CNA is reached, because CNA's own
// cna_sprite_font_set_default_character refuses the same case with
// CNA_RESULT_INVALID_ARGUMENT and no message. The reference's exact sentence is
// the projected behaviour; CNA's refusal would be a different one.
func (f *SpriteFont) SetDefaultCharacter(value *uint16) error {
	if f == nil {
		return errSpriteFontNil
	}
	if value != nil && !f.containsCharacter(*value) {
		return fmt.Errorf("%w: %s", errGraphicsResourceArgument, formatCharacterNotInFont(*value))
	}
	if value == nil {
		f.defaultCharacter = nil
	} else {
		character := *value
		f.defaultCharacter = &character
	}
	return f.font.SetSpriteFontDefaultCharacter(value != nil, characterOrZero(value))
}

// characterOrZero is the pair cna_sprite_font_set_default_character takes: a
// has-value flag and a value CNA ignores when the flag is false.
func characterOrZero(value *uint16) uint16 {
	if value == nil {
		return 0
	}
	return *value
}

// containsCharacter is `characterMap.Contains(value)`, the linear scan
// set_DefaultCharacter really performs.
func (f *SpriteFont) containsCharacter(value uint16) bool {
	for index := range f.glyphs {
		if f.glyphs[index].Character == value {
			return true
		}
	}
	return false
}

// Characters is SpriteFont::get_Characters:
//
//	if (characters == null)
//	    characters = new ReadOnlyCollection<char>(characterMap);
//	return characters;
//
// A lazily built view, CACHED in a field, so the second call returns the SAME
// object -- which is an identity a consumer can observe and the projection
// preserves.
//
// The reference's view wraps the LIVE characterMap. Nothing public mutates it,
// so a live view and a snapshot are indistinguishable here; the slice this
// wraps is the glyph table's character column, built once alongside the view it
// backs.
func (f *SpriteFont) Characters() *framework.ReadOnlyCollection[uint16] {
	if f == nil {
		return nil
	}
	if f.characters == nil {
		characters := make([]uint16, len(f.glyphs))
		for index := range f.glyphs {
			characters[index] = f.glyphs[index].Character
		}
		f.characters = framework.NewReadOnlyCollectionOverCharacters(characters)
	}
	return f.characters
}

// MeasureStringByString is SpriteFont::MeasureString(String):
//
//	if (text == null) throw new ArgumentNullException("text");
//	StringProxy proxy = new StringProxy(text);
//	return InternalMeasure(ref proxy);
//
// The proxy exists in the reference only to give the String and StringBuilder
// overloads one body; both construct one and both call the same private
// measure. See internalMeasure for the algorithm.
//
// A Go string cannot be null, so the reference's ArgumentNullException has no
// reachable counterpart on this overload: the empty string is a REAL empty
// string, which InternalMeasure answers Vector2.Zero for. That is recorded
// rather than turned into a refusal the reference does not make for "".
func (f *SpriteFont) MeasureStringByString(text string) (framework.Vector2, error) {
	if f == nil {
		return framework.Vector2{}, errSpriteFontNil
	}
	return f.internalMeasure(utf16.Encode([]rune(text)))
}

// MeasureStringByStringBuilder is SpriteFont::MeasureString(StringBuilder).
//
// # Why *strings.Builder
//
// System.Text.StringBuilder appears at four public signature positions in the
// pinned contract -- this one and SpriteBatch's three DrawString shapes -- and
// every one of them READS it: StringProxy stores the builder and consults only
// get_Length and get_Chars. That is the same measurement System.IO.Stream's
// mapping rests on, and it takes the same answer: the position projects to the
// standard-library Go type whose ROLE it is, not to a reimplemented BCL class.
//
// The overload identity is preserved: `string` and `*strings.Builder` are
// different Go types, so the two CLR overloads stay two Go functions, which is
// what the CLR overload set means.
//
// A nil builder IS the reference's null, and it takes the reference's own
// ArgumentNullException("text").
func (f *SpriteFont) MeasureStringByStringBuilder(text *strings.Builder) (framework.Vector2, error) {
	if f == nil {
		return framework.Vector2{}, errSpriteFontNil
	}
	if text == nil {
		return framework.Vector2{}, fmt.Errorf("%w: text", errGraphicsResourceArgumentNull)
	}
	return f.internalMeasure(utf16.Encode([]rune(text.String())))
}

// internalMeasure is SpriteFont::InternalMeasure(ref StringProxy), 420 bytes of
// IL, reproduced statement for statement:
//
//	if (text.Length == 0) return Vector2.Zero;
//	Vector2 result = Vector2.Zero;
//	result.Y = lineSpacing;
//	float maxWidth = 0f; int lineCount = 0; float rightBearing = 0f;
//	bool firstGlyphOnLine = true;
//	for (int i = 0; i < text.Length; i++) {
//	    char c = text[i];
//	    if (c == '\r') continue;
//	    if (c == '\n') {
//	        result.X += Math.Max(rightBearing, 0f);
//	        rightBearing = 0f;
//	        maxWidth = Math.Max(result.X, maxWidth);
//	        result = Vector2.Zero; result.Y = lineSpacing;
//	        firstGlyphOnLine = true; lineCount++;
//	        continue;
//	    }
//	    Vector3 k = kerning[GetIndexForCharacter(c)];
//	    if (firstGlyphOnLine) k.X = Math.Max(k.X, 0f);
//	    else                  result.X += spacing + rightBearing;
//	    result.X += k.X + k.Y;
//	    rightBearing = k.Z;
//	    result.Y = Math.Max(result.Y, croppingData[GetIndexForCharacter(c)].Height);
//	    firstGlyphOnLine = false;
//	}
//	result.X += Math.Max(rightBearing, 0f);
//	result.Y += lineCount * lineSpacing;
//	result.X = Math.Max(result.X, maxWidth);
//	return result;
//
// # Five details worth naming, because each is invisible from the signature
//
// A carriage return is SKIPPED ENTIRELY: it advances nothing, ends no line and
// is not looked up, so "a\r\nb" and "a\nb" measure identically and a lone "\r"
// measures as the empty line it leaves.
//
// The FIRST glyph on a line has its left bearing clamped at zero, and no other
// glyph does. A negative-bearing glyph therefore measures differently at the
// start of a line than in the middle of one.
//
// `spacing` is added BETWEEN glyphs only, from the second onwards, and it is
// added together with the PREVIOUS glyph's right bearing.
//
// The height is `Max(lineSpacing, every cropping height)` for the LAST line,
// plus `lineCount * lineSpacing` for the lines before it -- an INTEGER multiply
// converted to float once, not a float accumulation.
//
// The trailing right bearing is clamped at zero and added after the loop, so a
// final glyph with a negative right bearing does not shorten the string.
//
// # The units are UTF-16 code units
//
// StringProxy indexes System.String and System.Text.StringBuilder, both of
// which are sequences of UTF-16 code units. A character outside the Basic
// Multilingual Plane is therefore TWO lookups of two surrogate code units, not
// one lookup of a rune -- which is why both overloads encode to []uint16 before
// reaching here rather than ranging over a Go string.
func (f *SpriteFont) internalMeasure(text []uint16) (framework.Vector2, error) {
	if len(text) == 0 {
		return framework.Vector2{}, nil
	}
	result := framework.Vector2{Y: float32(f.lineSpacing)}
	maxWidth := float32(0)
	lineCount := int32(0)
	rightBearing := float32(0)
	firstGlyphOnLine := true
	for _, character := range text {
		if character == '\r' {
			continue
		}
		if character == '\n' {
			result.X += maxSingle(rightBearing, 0)
			rightBearing = 0
			maxWidth = maxSingle(result.X, maxWidth)
			result = framework.Vector2{Y: float32(f.lineSpacing)}
			firstGlyphOnLine = true
			lineCount++
			continue
		}
		index, err := f.indexForCharacter(character)
		if err != nil {
			return framework.Vector2{}, err
		}
		glyph := f.glyphs[index]
		kerningX := glyph.KerningX
		if firstGlyphOnLine {
			kerningX = maxSingle(kerningX, 0)
		} else {
			result.X += f.spacing + rightBearing
		}
		result.X += kerningX + glyph.KerningY
		rightBearing = glyph.KerningZ
		result.Y = maxSingle(result.Y, float32(glyph.Cropping.Height))
		firstGlyphOnLine = false
	}
	result.X += maxSingle(rightBearing, 0)
	result.Y += float32(lineCount * f.lineSpacing)
	result.X = maxSingle(result.X, maxWidth)
	return result, nil
}

// maxSingle is System.Math.Max(float32, float32) at the five sites
// InternalMeasure calls it. It is spelled out rather than borrowed so the NaN
// rule is stated once: Math.Max returns NaN when EITHER argument is NaN, and Go
// has no float32 max in the standard library that says so.
func maxSingle(a, b float32) float32 {
	if a != a || b != b {
		return a - a + (b - b) // NaN, whichever side produced it
	}
	if a > b {
		return a
	}
	return b
}

// indexForCharacter is SpriteFont::GetIndexForCharacter(char), a binary search
// over the sorted character map with a default-character fallback:
//
//	int low = 0, high = characterMap.Count - 1;
//	while (low <= high) {
//	    int mid = low + ((high - low) >> 1);
//	    char at = characterMap[mid];
//	    if (at == character) return mid;
//	    if (at < character) low = mid + 1; else high = mid - 1;
//	}
//	if (defaultCharacter.HasValue && character != defaultCharacter.Value)
//	    return GetIndexForCharacter(defaultCharacter.Value);
//	throw new ArgumentException(String.Format(CurrentCulture,
//	    FrameworkResources.CharacterNotInFont, character, (int)character),
//	    "character");
//
// The recursion terminates because the second call passes the default
// character, and the `character != defaultCharacter.Value` test refuses to
// recurse on it a second time. A font whose default character is not in its own
// map therefore throws for it -- naming the DEFAULT character, not the one
// asked for, which is what the recursive call makes the message say.
//
// The shift is an ARITHMETIC one on `high - low`, so a map larger than
// int32.MaxValue/2 cannot overflow the midpoint; the projection keeps the same
// arithmetic rather than Go's more obvious `(low+high)/2`.
func (f *SpriteFont) indexForCharacter(character uint16) (int, error) {
	low, high := 0, len(f.glyphs)-1
	for low <= high {
		middle := low + ((high - low) >> 1)
		at := f.glyphs[middle].Character
		if at == character {
			return middle, nil
		}
		if at < character {
			low = middle + 1
		} else {
			high = middle - 1
		}
	}
	if f.defaultCharacter != nil && character != *f.defaultCharacter {
		return f.indexForCharacter(*f.defaultCharacter)
	}
	return 0, fmt.Errorf("%w: character: %s", errGraphicsResourceArgument, formatCharacterNotInFont(character))
}
