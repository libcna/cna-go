package graphics

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 69 — SpriteFont's managed surface.
// ---------------------------------------------------------------------------
//
// Every test here builds a font from a glyph table directly, which is what CNA
// hands the projection at load time. The two CNA handles are left nil: no test
// in this file reaches one, because no member below reaches a runtime -- the
// three setters do, and each of those assertions says so.

// glyph builds one table entry. The kerning triple is (left bearing, width,
// right bearing), which is the meaning InternalMeasure gives Vector3's three
// components.
func glyph(character uint16, cropHeight int32, left, width, right float32) interop.SpriteFontGlyph {
	return interop.SpriteFontGlyph{
		Character:   character,
		GlyphBounds: interop.SpriteFontRectangle{Width: int32(width), Height: cropHeight},
		Cropping:    interop.SpriteFontRectangle{Width: int32(width), Height: cropHeight},
		KerningX:    left,
		KerningY:    width,
		KerningZ:    right,
	}
}

// testFont is the corpus font: 'A' (65), 'B' (66) and '?' (63), sorted
// ascending exactly as the reference's characterMap and CNA's `.cnj` reader
// both require. 'B' carries NEGATIVE bearings on both sides, which is what
// makes the two clamps in InternalMeasure observable.
func testFont() *SpriteFont {
	return &SpriteFont{
		glyphs: []interop.SpriteFontGlyph{
			glyph('?', 8, 1, 4, 2),
			glyph('A', 8, 0, 5, 0),
			glyph('B', 12, -3, 6, -2),
		},
		lineSpacing: 10,
		spacing:     1,
	}
}

func measure(t *testing.T, font *SpriteFont, text string) (float32, float32) {
	t.Helper()
	size, err := font.MeasureStringByString(text)
	if err != nil {
		t.Fatalf("MeasureStringByString(%q): %v", text, err)
	}
	return size.X, size.Y
}

// TestAnEmptyStringMeasuresZeroRatherThanTheLineSpacing pins InternalMeasure's
// first branch:
//
//	if (text.Length == 0) return Vector2.Zero;
//
// which returns BEFORE result.Y is set to the line spacing. A projection that
// fell through would answer (0, lineSpacing), which is a plausible height and
// not the reference's.
func TestAnEmptyStringMeasuresZeroRatherThanTheLineSpacing(t *testing.T) {
	x, y := measure(t, testFont(), "")
	if x != 0 || y != 0 {
		t.Fatalf(`MeasureString("") = (%v, %v), want Vector2.Zero`, x, y)
	}
}

// TestOneGlyphIsItsLeftBearingPlusItsWidth pins the single-glyph arithmetic,
// including the two things that do NOT happen: `spacing` is not added, because
// the glyph is the first on its line, and the right bearing IS added, by the
// clamped post-loop statement.
func TestOneGlyphIsItsLeftBearingPlusItsWidth(t *testing.T) {
	x, y := measure(t, testFont(), "A")
	if x != 5 || y != 10 {
		t.Fatalf(`MeasureString("A") = (%v, %v), want (0+5+max(0,0), max(10, 8))`, x, y)
	}
	// '?' has a positive right bearing, and the post-loop statement adds it.
	x, y = measure(t, testFont(), "?")
	if x != 1+4+2 || y != 10 {
		t.Fatalf(`MeasureString("?") = (%v, %v), want (1+4+2, 10)`, x, y)
	}
}

// TestTheFirstGlyphOnALineHasItsLEFTBearingClampedAtZero pins the branch that
// separates a first glyph from every other:
//
//	if (firstGlyphOnLine) k.X = Math.Max(k.X, 0f);
//
// 'B' has a left bearing of -3. Alone it measures 0+6 = 6, not 3.
func TestTheFirstGlyphOnALineHasItsLEFTBearingClampedAtZero(t *testing.T) {
	x, _ := measure(t, testFont(), "B")
	if x != 6 {
		t.Fatalf(`MeasureString("B") = %v, want max(-3,0)+6`, x)
	}
	// And its NEGATIVE right bearing is clamped by the post-loop statement, so
	// a single 'B' is not 6 + (-2).
	if x != 6 {
		t.Fatalf("the trailing right bearing was not clamped: %v", x)
	}
}

// TestASecondGlyphAddsSpacingAndThePreviousRightBearing pins the else branch,
// which is the one place `spacing` enters the measurement:
//
//	result.X += spacing + rightBearing;   // then k.X + k.Y
//
// "AB" is 0+5 for 'A', then +1 spacing +0 A's right bearing, then B's UNCLAMPED
// -3 + 6, then the post-loop max(-2, 0) = 0.
func TestASecondGlyphAddsSpacingAndThePreviousRightBearing(t *testing.T) {
	x, y := measure(t, testFont(), "AB")
	want := float32(5 + 1 + 0 + (-3) + 6)
	if x != want {
		t.Fatalf(`MeasureString("AB") = %v, want %v`, x, want)
	}
	// The height is the LARGEST cropping height on the line, and 'B' is 12.
	if y != 12 {
		t.Fatalf(`MeasureString("AB").Y = %v, want max(lineSpacing, 8, 12)`, y)
	}
}

// TestASecondGlyphsLeftBearingIsNOTClamped is the negative control for the
// clamp above: 'B' after 'A' contributes -3, so the two-glyph width is smaller
// than the sum of the two one-glyph widths.
func TestASecondGlyphsLeftBearingIsNOTClamped(t *testing.T) {
	font := testFont()
	pair, _ := measure(t, font, "AB")
	a, _ := measure(t, font, "A")
	b, _ := measure(t, font, "B")
	if pair >= a+b {
		t.Fatalf("AB=%v is not narrower than A=%v plus B=%v; the second glyph's bearing was clamped", pair, a, b)
	}
}

// TestACarriageReturnIsSkippedEntirely pins the first test in the loop:
//
//	if (c == '\r') continue;
//
// It is not looked up, adds nothing, and does not end a line -- so "A\r\nB" and
// "A\nB" measure identically, and a string of carriage returns measures as the
// single empty line it leaves.
func TestACarriageReturnIsSkippedEntirely(t *testing.T) {
	font := testFont()
	withReturn, heightWithReturn := measure(t, font, "A\r\nB")
	without, heightWithout := measure(t, font, "A\nB")
	if withReturn != without || heightWithReturn != heightWithout {
		t.Fatalf("A\\r\\nB = (%v,%v) and A\\nB = (%v,%v); the carriage return was not skipped",
			withReturn, heightWithReturn, without, heightWithout)
	}
	// A lone carriage return is a non-empty string that reaches no glyph.
	x, y := measure(t, font, "\r")
	if x != 0 || y != 10 {
		t.Fatalf(`MeasureString("\r") = (%v, %v), want (0, lineSpacing)`, x, y)
	}
}

// TestANewlineStartsALineAndAddsOneLineSpacing pins the newline branch and the
// post-loop height statement together. Two lines is `lineSpacing` for the last
// line plus `1 * lineSpacing` for the one before it -- and the WIDTH is the
// widest line, carried in maxWidth.
func TestANewlineStartsALineAndAddsOneLineSpacing(t *testing.T) {
	font := testFont()
	x, y := measure(t, font, "A\nA")
	if y != 20 {
		t.Fatalf(`MeasureString("A\nA").Y = %v, want lineSpacing + 1*lineSpacing`, y)
	}
	if x != 5 {
		t.Fatalf(`MeasureString("A\nA").X = %v, want the widest line`, x)
	}
	// A wider first line survives in maxWidth, which is the only reason the
	// reference keeps that local.
	x, _ = measure(t, font, "AB\nA")
	if x != 9 {
		t.Fatalf(`MeasureString("AB\nA").X = %v, want the FIRST line's width`, x)
	}
}

// TestTheHeightAddsAnIntegerMultipleOfLineSpacing pins the post-loop statement
//
//	result.Y += (float)(lineCount * lineSpacing);
//
// as the INTEGER multiply it is. Three newlines is three lines counted, and the
// last line's own height is the Max already in result.Y.
func TestTheHeightAddsAnIntegerMultipleOfLineSpacing(t *testing.T) {
	_, y := measure(t, testFont(), "A\nA\nA\nB")
	if y != 12+30 {
		t.Fatalf(`MeasureString("A\nA\nA\nB").Y = %v, want max(10,12) + 3*10`, y)
	}
}

// TestAnUnknownCharacterFallsBackToTheDefaultCharacter pins
// GetIndexForCharacter's fallback, and pins that it is the DEFAULT glyph's
// metrics that are used -- not a zero-width skip.
func TestAnUnknownCharacterFallsBackToTheDefaultCharacter(t *testing.T) {
	font := testFont()
	fallback := uint16('?')
	font.defaultCharacter = &fallback
	unknown, _ := measure(t, font, "Z")
	question, _ := measure(t, font, "?")
	if unknown != question {
		t.Fatalf("an unknown character measured %v and the default measured %v", unknown, question)
	}
}

// TestAnUnknownCharacterWithNoDefaultIsRefusedWithTheReferencesSentence pins
// the throw, its exact message and the character it names. The message is
// formatted from FrameworkResources.CharacterNotInFont with the SAME character
// at both placeholders -- as a char at {0} and as four lowercase hex digits at
// {1:x4}.
func TestAnUnknownCharacterWithNoDefaultIsRefusedWithTheReferencesSentence(t *testing.T) {
	_, err := testFont().MeasureStringByString("Z")
	if err == nil {
		t.Fatal("an unknown character with no default character was measured")
	}
	if !errors.Is(err, errGraphicsResourceArgument) {
		t.Fatalf("MeasureString refused with %v, want an ArgumentException", err)
	}
	if !strings.Contains(err.Error(), "The character 'Z' (0x005a) is not available in this SpriteFont.") {
		t.Fatalf("the refusal does not carry the reference's sentence: %v", err)
	}
	// GetIndexForCharacter's throw names the parameter; set_DefaultCharacter's
	// does not, and the two are different exceptions.
	if !strings.Contains(err.Error(), "character:") {
		t.Fatalf("the refusal does not name the `character` parameter: %v", err)
	}
}

// TestADefaultCharacterOutsideTheFontIsRefusedNamingTHATCharacter pins the
// recursion's terminating branch. A font whose default character is not in its
// own map throws for the DEFAULT, because the recursive call is what reaches
// the throw.
func TestADefaultCharacterOutsideTheFontIsRefusedNamingTHATCharacter(t *testing.T) {
	font := testFont()
	missing := uint16('#')
	font.defaultCharacter = &missing
	_, err := font.MeasureStringByString("Z")
	if err == nil {
		t.Fatal("a default character outside the font measured successfully")
	}
	if !strings.Contains(err.Error(), "'#'") {
		t.Fatalf("the refusal names the asked-for character rather than the default: %v", err)
	}
}

// TestSetDefaultCharacterRefusesACharacterTheFontDoesNotHave pins
// set_DefaultCharacter's own guard, which is a DIFFERENT throw site from
// GetIndexForCharacter's: same sentence, no parameter name.
//
// It also pins that the refusal happens BEFORE CNA is reached: this font has no
// native handle, so a projection that pushed first would report ErrDisposed.
func TestSetDefaultCharacterRefusesACharacterTheFontDoesNotHave(t *testing.T) {
	font := testFont()
	missing := uint16('#')
	err := font.SetDefaultCharacter(&missing)
	if err == nil {
		t.Fatal("set_DefaultCharacter accepted a character outside the font")
	}
	if errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("the guard did not run before CNA was reached: %v", err)
	}
	if !strings.Contains(err.Error(), "The character '#' (0x0023) is not available in this SpriteFont.") {
		t.Fatalf("the refusal does not carry the reference's sentence: %v", err)
	}
	if strings.Contains(err.Error(), "character:") {
		t.Fatalf("set_DefaultCharacter's throw names a parameter the reference does not: %v", err)
	}
	if _, ok := font.DefaultCharacter(); ok {
		t.Fatal("the refused value was stored")
	}
}

// TestClearingTheDefaultCharacterIsAlwaysAllowed pins that the guard is
// `HasValue && !Contains`: a null skips it entirely, so clearing never throws
// even on a font whose map is empty.
//
// The store happens before the native push, which is why the managed field is
// cleared here even though this font has no CNA handle to push to.
func TestClearingTheDefaultCharacterIsAlwaysAllowed(t *testing.T) {
	font := testFont()
	present := uint16('A')
	font.defaultCharacter = &present
	err := font.SetDefaultCharacter(nil)
	if !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("clearing reported %v; with no CNA handle the native push is what must fail", err)
	}
	if _, ok := font.DefaultCharacter(); ok {
		t.Fatal("clearing did not store null before pushing")
	}
}

// TestDefaultCharacterIsReturnedByValue pins the nullable return: the reference
// copies a value type out of a field, so nothing a caller does with the result
// can reach the font.
func TestDefaultCharacterIsReturnedByValue(t *testing.T) {
	font := testFont()
	stored := uint16('A')
	font.defaultCharacter = &stored
	value, ok := font.DefaultCharacter()
	if !ok || value != 'A' {
		t.Fatalf("DefaultCharacter() = (%v, %v), want ('A', true)", value, ok)
	}
	value = 'B'
	again, _ := font.DefaultCharacter()
	if again != 'A' {
		t.Fatalf("writing the returned value changed the font: %v", again)
	}
	_ = value
}

// TestCharactersIsTheSortedMapAndIsCachedByIdentity pins get_Characters: the
// view is built once and STORED, so the second call returns the same object.
func TestCharactersIsTheSortedMapAndIsCachedByIdentity(t *testing.T) {
	font := testFont()
	first := font.Characters()
	if first == nil {
		t.Fatal("Characters() is nil")
	}
	if got := first.Count(); got != 3 {
		t.Fatalf("Characters().Count() = %v, want 3", got)
	}
	if second := font.Characters(); second != first {
		t.Fatal("Characters() built a second view; the reference caches the first")
	}
	// The order is the glyph table's, which CNA guarantees ascending.
	for index, want := range []uint16{'?', 'A', 'B'} {
		got, err := first.Item(int32(index))
		if err != nil || got != want {
			t.Fatalf("Characters()[%d] = (%v, %v), want %v", index, got, err, want)
		}
	}
}

// TestTheStringBuilderOverloadMeasuresTheBuildersCurrentText pins that the
// StringBuilder overload reads the builder rather than being a second name for
// the string one, and that a nil builder is the reference's null.
func TestTheStringBuilderOverloadMeasuresTheBuildersCurrentText(t *testing.T) {
	font := testFont()
	var builder strings.Builder
	builder.WriteString("AB")
	built, err := font.MeasureStringByStringBuilder(&builder)
	if err != nil {
		t.Fatalf("MeasureStringByStringBuilder: %v", err)
	}
	direct, _ := measure(t, font, "AB")
	if built.X != direct || built.Y != 12 {
		t.Fatalf("the builder measured %v and the string measured %v", built, direct)
	}

	_, err = font.MeasureStringByStringBuilder(nil)
	if err == nil {
		t.Fatal("a nil StringBuilder was measured")
	}
	if !errors.Is(err, errGraphicsResourceArgumentNull) || !strings.Contains(err.Error(), "text") {
		t.Fatalf("a nil builder reported %v, want ArgumentNullException(\"text\")", err)
	}
}

// TestASurrogatePairIsTwoLookupsRatherThanOne pins the UTF-16 unit rule.
// StringProxy indexes code units, so a non-BMP rune is two characters to the
// reference -- and a projection that ranged over Go runes would perform one
// lookup of a value no font's 16-bit character map can hold.
func TestASurrogatePairIsTwoLookupsRatherThanOne(t *testing.T) {
	font := testFont()
	fallback := uint16('A')
	font.defaultCharacter = &fallback
	// U+1D400 is one rune and two UTF-16 code units.
	pair, err := font.MeasureStringByString("\U0001D400")
	if err != nil {
		t.Fatalf("measuring a surrogate pair: %v", err)
	}
	two, _ := measure(t, font, "AA")
	if pair.X != two {
		t.Fatalf("a surrogate pair measured %v and two fallback glyphs measured %v", pair.X, two)
	}
}

// TestTheThreeSettersReachCNA pins that each setter is fallible because it
// pushes, and that the managed field is written FIRST -- which is the order
// that keeps MeasureString reading the reference's own state whatever CNA
// answers. This font has no handle, so the push is what fails.
func TestTheThreeSettersReachCNA(t *testing.T) {
	font := testFont()
	if err := font.SetLineSpacing(24); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("SetLineSpacing reported %v, want the native push to fail", err)
	}
	if font.LineSpacing() != 24 {
		t.Fatalf("LineSpacing() = %d; the managed store must precede the push", font.LineSpacing())
	}
	if err := font.SetSpacing(2.5); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("SetSpacing reported %v, want the native push to fail", err)
	}
	if font.Spacing() != 2.5 {
		t.Fatalf("Spacing() = %v; the managed store must precede the push", font.Spacing())
	}
	// And the stored values are what MeasureString then uses.
	x, y := measure(t, font, "AB")
	if y != 24 {
		t.Fatalf("the new line spacing did not reach MeasureString: %v", y)
	}
	if x != 5+2.5+0+(-3)+6 {
		t.Fatalf("the new spacing did not reach MeasureString: %v", x)
	}
}

// TestMaxSingleFollowsMathMaxOnNaN pins the one place Go's obvious comparison
// and System.Math.Max disagree: Math.Max returns NaN when either argument is
// NaN, and `a > b` is false for every NaN comparison, so a naive `if a > b`
// would silently answer b.
func TestMaxSingleFollowsMathMaxOnNaN(t *testing.T) {
	nan := float32(math.NaN())
	if got := maxSingle(nan, 0); !math.IsNaN(float64(got)) {
		t.Fatalf("maxSingle(NaN, 0) = %v, want NaN", got)
	}
	if got := maxSingle(0, nan); !math.IsNaN(float64(got)) {
		t.Fatalf("maxSingle(0, NaN) = %v, want NaN", got)
	}
	if got := maxSingle(3, 7); got != 7 {
		t.Fatalf("maxSingle(3, 7) = %v", got)
	}
}
