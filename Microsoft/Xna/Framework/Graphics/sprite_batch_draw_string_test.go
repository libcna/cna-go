package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 70 — the six DrawString overloads' managed half.
// ---------------------------------------------------------------------------
//
// These measure the two guards each overload applies before anything is queued,
// the ORDER it applies them in, and the argument normalisation the six differ
// by. A live native SpriteBatch is not needed for any of it, because the
// reference reaches neither the queue nor the font until both guards have
// passed -- and the native half is proved by the sprite-font stress scenario,
// which submits all six through a real begin/end pair against a real CNA font.

// drawStringOverload names one of the six and calls it on a given batch with a
// given font. Every case below runs all six, because a new overload that
// reached the resource directly would pass a test that named only one.
type drawStringOverload struct {
	name string
	call func(*SpriteBatch, *SpriteFont) error
}

func drawStringOverloads(text *strings.Builder) []drawStringOverload {
	white := framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)
	position := framework.NewVector2BySingleAndSingle(4, 5)
	origin := framework.NewVector2BySingleAndSingle(1, 2)
	scale := framework.NewVector2BySingleAndSingle(2, 3)
	return []drawStringOverload{
		{"String", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringAndVector2AndColor(f, "AB", position, white)
		}},
		{"StringBuilder", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(f, text, position, white)
		}},
		{"String uniform scale", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				f, "AB", position, white, 0.25, origin, 2, SpriteEffectsFlipHorizontally, 0.5)
		}},
		{"StringBuilder uniform scale", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				f, text, position, white, 0.25, origin, 2, SpriteEffectsFlipHorizontally, 0.5)
		}},
		{"String per-axis scale", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				f, "AB", position, white, 0.25, origin, scale, SpriteEffectsNone, 0.5)
		}},
		{"StringBuilder per-axis scale", func(b *SpriteBatch, f *SpriteFont) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				f, text, position, white, 0.25, origin, scale, SpriteEffectsNone, 0.5)
		}},
	}
}

// TestANilSpriteFontIsReportedWithTheParameterNameAndNoResourceString pins the
// first guard AND the shape it takes. DrawString's throw is
//
//	ldstr "spriteFont"; newobj ArgumentNullException::.ctor(string)
//
// the ONE-argument constructor, where Draw's null-texture throw loads
// FrameworkResources.NullNotAllowed into the two-argument one. Same exception
// class, different message, and the difference is the reference's.
func TestANilSpriteFontIsReportedWithTheParameterNameAndNoResourceString(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("AB")
	for _, overload := range drawStringOverloads(&builder) {
		batch := &SpriteBatch{inBeginEndPair: true}
		err := overload.call(batch, nil)
		if !errors.Is(err, errSpriteArgumentNull) {
			t.Fatalf("%s: error = %v, want the ArgumentNullException projection", overload.name, err)
		}
		if !strings.Contains(err.Error(), "spriteFont") {
			t.Fatalf("%s: error %q does not name the parameter", overload.name, err)
		}
		if strings.Contains(err.Error(), nullNotAllowed) {
			t.Fatalf("%s: error %q carries a resource string DrawString's throw does not load", overload.name, err)
		}
	}
}

// TestTheSpriteFontCheckRunsBeforeTheBeginEndCheck pins the order. Both
// conditions hold at once here and the IL decides which is reported: the
// argument throw is at IL_0003 and the pair check happens later, inside
// SpriteBatch::InternalDraw, when the first glyph reaches it.
func TestTheSpriteFontCheckRunsBeforeTheBeginEndCheck(t *testing.T) {
	var builder strings.Builder
	for _, overload := range drawStringOverloads(&builder) {
		batch := &SpriteBatch{}
		if err := overload.call(batch, nil); !errors.Is(err, errSpriteArgumentNull) {
			t.Fatalf("%s: error = %v, want the argument to be reported before the state", overload.name, err)
		}
	}
}

// TestANilStringBuilderIsTheReferencesNullText pins the second guard on the
// three overloads that can reach it. A Go string cannot be null, so only the
// StringBuilder family has a reachable counterpart -- which is exactly why the
// two CLR overloads must stay two Go members.
func TestANilStringBuilderIsTheReferencesNullText(t *testing.T) {
	font := testFont()
	for _, call := range []func(*SpriteBatch) error{
		func(b *SpriteBatch) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(
				font, nil, framework.Vector2{}, framework.Color{})
		},
		func(b *SpriteBatch) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				font, nil, framework.Vector2{}, framework.Color{}, 0, framework.Vector2{}, 1, 0, 0)
		},
		func(b *SpriteBatch) error {
			return b.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				font, nil, framework.Vector2{}, framework.Color{}, 0, framework.Vector2{}, framework.Vector2{}, 0, 0)
		},
	} {
		batch := &SpriteBatch{inBeginEndPair: true}
		err := call(batch)
		if !errors.Is(err, errSpriteArgumentNull) {
			t.Fatalf("error = %v, want ArgumentNullException for a nil StringBuilder", err)
		}
		if !strings.Contains(err.Error(), "text") {
			t.Fatalf("error %q does not name the `text` parameter", err)
		}
	}
}

// TestTheFontCheckRunsBeforeTheTextCheck pins their relative order: IL_0003
// against IL_0011. Both arguments are null here.
func TestTheFontCheckRunsBeforeTheTextCheck(t *testing.T) {
	batch := &SpriteBatch{inBeginEndPair: true}
	err := batch.DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(
		nil, nil, framework.Vector2{}, framework.Color{})
	if !strings.Contains(err.Error(), "spriteFont") {
		t.Fatalf("error %q reports the text before the font", err)
	}
}

// TestEverySixOverloadsShareTheBeginEndGuard is the coverage proof: with both
// arguments valid and no begin/end pair, all six must report the reference's
// InvalidOperationException rather than reaching a resource.
func TestEverySixOverloadsShareTheBeginEndGuard(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("AB")
	font := testFont()
	for _, overload := range drawStringOverloads(&builder) {
		batch := &SpriteBatch{}
		err := overload.call(batch, font)
		if !errors.Is(err, errSpriteInvalidOperation) {
			t.Fatalf("%s: error = %v, want the InvalidOperationException projection", overload.name, err)
		}
		if !strings.Contains(err.Error(), beginMustBeCalledBeforeDraw) {
			t.Fatalf("%s: error %q is not FrameworkResources.BeginMustBeCalledBeforeDraw", overload.name, err)
		}
	}
}

// TestInsideAPairEverySixOverloadsReachTheResource is the other half of the
// coverage proof. With both guards satisfied and a batch with no native half,
// every overload must report the DISPOSED sentinel -- which is only reachable
// past both guards, so an overload that skipped one would report a different
// error here.
func TestInsideAPairEverySixOverloadsReachTheResource(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("AB")
	font := testFont()
	for _, overload := range drawStringOverloads(&builder) {
		batch := &SpriteBatch{inBeginEndPair: true}
		if err := overload.call(batch, font); !errors.Is(err, interop.ErrDisposed) {
			t.Fatalf("%s: error = %v, want the guards passed and the resource reached", overload.name, err)
		}
	}
}

// TestTheStringBuilderOverloadsReadTheBuildersCurrentText pins that the builder
// is READ at the call, which is what StringProxy does: it stores the builder
// and consults get_Length and get_Chars, so text appended after the overload
// was chosen is still drawn.
func TestTheStringBuilderOverloadsReadTheBuildersCurrentText(t *testing.T) {
	var builder strings.Builder
	if got := builderText(&builder); got != "" {
		t.Fatalf("an empty builder read %q", got)
	}
	builder.WriteString("AB")
	if got := builderText(&builder); got != "AB" {
		t.Fatalf("builderText = %q, want the builder's current text", got)
	}
	builder.WriteString("C")
	if got := builderText(&builder); got != "ABC" {
		t.Fatalf("builderText = %q after a second append", got)
	}
	if got := builderText(nil); got != "" {
		t.Fatalf("a nil builder read %q; its refusal is the caller's", got)
	}
}
