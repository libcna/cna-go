package graphics

import (
	"fmt"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 70 — SpriteBatch's six DrawString overloads.
// ---------------------------------------------------------------------------
//
// # Six public doors, one private call, exactly as Draw has
//
// Every one of the six is the same eleven IL instructions of guard followed by
// one call:
//
//	if (spriteFont == null) throw new ArgumentNullException("spriteFont");
//	if (text == null)       throw new ArgumentNullException("text");
//	StringProxy proxy = new StringProxy(text);
//	spriteFont.InternalDraw(ref proxy, this, position, color, rotation, origin,
//	                        ref scale, effects, depth);
//
// and the six differ only in which transform fields they leave at a default:
//
//	(font, text, position, color)                rotation 0, origin Zero,
//	                                             scale Vector2.One, effects 0,
//	                                             depth 0
//	(..., float scale, ...)                      scale = (scale, scale)
//	(..., Vector2 scale, ...)                    scale as given
//
// each in a String and a StringBuilder flavour. Those defaults are read off the
// prologues -- `Vector2::get_One` at IL_0029 of the four-argument pair, and the
// two `stfld`s that splat a float into both components in the uniform-scale
// pair -- rather than chosen.
//
// # The layout is the runtime's, which is the settled SpriteBatch decision
//
// Foundation 50 settled that SpriteBatch::InternalDraw is the RUNTIME's job:
// the seven Draw overloads are argument normalisers and CNA lays the sprite
// out. DrawString has exactly that shape one level up -- SpriteFont::InternalDraw
// walks the string, places each glyph and calls SpriteBatch::InternalDraw per
// glyph -- and CNA offers cna_sprite_batch_draw_string, whose command carries
// the same nine values InternalDraw takes, in the same meaning. So the six
// overloads normalise and CNA expands.
//
// # One recorded consequence
//
// SpriteFont::InternalDraw consults InternalMeasure ONLY when a flip is
// requested: FlipHorizontally reads its X to mirror the block and
// FlipVertically its Y. Foundation 69 measured cna_sprite_font_measure_utf8
// disagreeing with InternalMeasure by the last glyph's negative right bearing,
// so a FLIPPED string whose final glyph has a negative right bearing may be
// offset by that amount from where the reference would put it. Unflipped text
// is unaffected, and MeasureString -- the value a consumer can see -- is the
// reference's own arithmetic either way. This is CNA's layout differing from
// the reference's, recorded rather than hidden, and it is the same measured
// divergence rather than a second one.

// spriteDrawString is the projection of the shared prologue. It takes the
// already-normalised scale as two floats, which is the `ref Vector2` the
// reference passes.
func (b *SpriteBatch) spriteDrawString(
	spriteFont *SpriteFont,
	text string,
	textIsNil bool,
	position framework.Vector2,
	color framework.Color,
	rotation float32,
	origin framework.Vector2,
	scaleX, scaleY float32,
	effects SpriteEffects,
	layerDepth float32,
) error {
	// A nil receiver is the one case with no reference counterpart -- the CLR
	// has no null `this` -- and it is answered first because every check after
	// it reads a field.
	if b == nil {
		return interop.ErrDisposed
	}
	// The reference's order: spriteFont at IL_0003, text at IL_0011. Both come
	// BEFORE the begin/end check, which SpriteBatch::InternalDraw performs when
	// the first glyph reaches it -- so a null argument outside a begin/end pair
	// reports the ARGUMENT, exactly as Draw's null texture does.
	//
	// Neither throw loads FrameworkResources.NullNotAllowed. Both are
	// `ldstr "spriteFont"` / `ldstr "text"` into the one-argument
	// ArgumentNullException constructor, which is a different shape from
	// Draw's -- and the difference is preserved rather than smoothed over.
	if spriteFont == nil {
		return fmt.Errorf("%w: spriteFont", errSpriteArgumentNull)
	}
	if textIsNil {
		return fmt.Errorf("%w: text", errSpriteArgumentNull)
	}
	if !b.inBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, beginMustBeCalledBeforeDraw)
	}
	if b.resource() == nil || spriteFont.nativeFont() == nil {
		return interop.ErrDisposed
	}
	return b.resource().DrawString(spriteFont.nativeFont(), text, interop.SpriteTextCommand{
		PositionX: position.X, PositionY: position.Y,
		Red: color.R(), Green: color.G(), Blue: color.B(), Alpha: color.A(),
		Rotation: rotation, OriginX: origin.X, OriginY: origin.Y,
		ScaleX: scaleX, ScaleY: scaleY,
		Effects: uint32(effects), LayerDepth: layerDepth,
	})
}

// DrawStringBySpriteFontAndStringAndVector2AndColor is
// SpriteBatch::DrawString(SpriteFont, String, Vector2, Color).
//
//	rotation = 0, origin = Vector2.Zero, scale = Vector2.One,
//	effects = 0, depth = 0
//
// A Go string cannot be null, so the reference's ArgumentNullException("text")
// has no reachable counterpart on the three String overloads: the empty string
// is a REAL empty string, which draws nothing. That is recorded rather than
// turned into a refusal the reference does not make for "".
func (b *SpriteBatch) DrawStringBySpriteFontAndStringAndVector2AndColor(
	spriteFont *SpriteFont, text string, position framework.Vector2, color framework.Color,
) error {
	return b.spriteDrawString(spriteFont, text, false, position, color, 0, framework.Vector2{}, 1, 1, 0, 0)
}

// DrawStringBySpriteFontAndStringBuilderAndVector2AndColor is
// SpriteBatch::DrawString(SpriteFont, StringBuilder, Vector2, Color).
//
// The parameter is `*strings.Builder` on the settled BCL signature rule: all
// four StringBuilder positions in the profile are inputs whose only reads are
// get_Length and get_Chars, which is the measurement System.IO.Stream's
// io.Reader mapping rests on. A nil builder IS the reference's null.
func (b *SpriteBatch) DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(
	spriteFont *SpriteFont, text *strings.Builder, position framework.Vector2, color framework.Color,
) error {
	return b.spriteDrawString(spriteFont, builderText(text), text == nil,
		position, color, 0, framework.Vector2{}, 1, 1, 0, 0)
}

// DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle
// is the UNIFORM-scale String overload. The reference splats the one float into
// both components of a local Vector2 with two `stfld`s, which is what makes it
// the same call as the per-axis overload and a different member.
func (b *SpriteBatch) DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
	spriteFont *SpriteFont, text string, position framework.Vector2, color framework.Color,
	rotation float32, origin framework.Vector2, scale float32,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDrawString(spriteFont, text, false, position, color,
		rotation, origin, scale, scale, effects, layerDepth)
}

// DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle
// is its StringBuilder twin.
func (b *SpriteBatch) DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
	spriteFont *SpriteFont, text *strings.Builder, position framework.Vector2, color framework.Color,
	rotation float32, origin framework.Vector2, scale float32,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDrawString(spriteFont, builderText(text), text == nil, position, color,
		rotation, origin, scale, scale, effects, layerDepth)
}

// DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle
// is the PER-AXIS scale String overload -- the only DrawString whose scale
// components can differ.
func (b *SpriteBatch) DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
	spriteFont *SpriteFont, text string, position framework.Vector2, color framework.Color,
	rotation float32, origin framework.Vector2, scale framework.Vector2,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDrawString(spriteFont, text, false, position, color,
		rotation, origin, scale.X, scale.Y, effects, layerDepth)
}

// DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle
// is its StringBuilder twin.
func (b *SpriteBatch) DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
	spriteFont *SpriteFont, text *strings.Builder, position framework.Vector2, color framework.Color,
	rotation float32, origin framework.Vector2, scale framework.Vector2,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDrawString(spriteFont, builderText(text), text == nil, position, color,
		rotation, origin, scale.X, scale.Y, effects, layerDepth)
}

// builderText reads a StringBuilder's CURRENT text, which is what
// SpriteFont/StringProxy indexes: it stores the builder and consults get_Length
// and get_Chars, so the value drawn is whatever the builder holds at the call.
// A nil builder answers the empty string and the caller reports the reference's
// ArgumentNullException instead.
func builderText(text *strings.Builder) string {
	if text == nil {
		return ""
	}
	return text.String()
}
