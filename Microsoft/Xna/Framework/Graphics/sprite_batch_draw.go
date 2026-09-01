package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SpriteBatch's Draw family, and the begin/end pair it is guarded by.
//
// # One private method, seven public doors
//
// Every one of the reference's seven Draw overloads is a short prologue over
// one private call:
//
//	InternalDraw(Texture2D texture, ref Vector4 destination, bool scaleDestination,
//	             ref Nullable<Rectangle> sourceRectangle, Color color, float rotation,
//	             ref Vector2 origin, SpriteEffects effects, float depth)
//
// and `destination` is a Vector4 that means two different things:
//
//	position overloads   (position.X, position.Y, scaleX, scaleY)   scaleDestination = true
//	rectangle overloads  (rect.X, rect.Y, rect.Width, rect.Height)  scaleDestination = false
//
// The three-argument overloads pass 1 for both scale components, the static
// `nullRectangle` for the source, 0 for rotation, the static `vector2Zero` for
// the origin, 0 for effects and 0 for depth. Every default here is read off
// those prologues rather than chosen.
//
// CNA declares the same split from the other side, in two structures with a
// documented reason: with a position, the origin is measured in source-texture
// pixels and the scale applies after that offset, so a caller cannot reproduce
// one family by computing a rectangle for the other. So the projection is two
// routes -- cna_sprite_batch_submit_scaled_many and cna_sprite_batch_submit_many
// -- and the choice between them is `scaleDestination`, exactly as in the IL.
//
// # The two guards
//
// InternalDraw begins with two throws, in this order:
//
//	if (texture == null)
//	    throw new ArgumentNullException("texture", FrameworkResources.NullNotAllowed);
//	if (!this.inBeginEndPair)
//	    throw new InvalidOperationException(FrameworkResources.BeginMustBeCalledBeforeDraw);
//
// Both are reproduced HERE, managed-side, rather than left to CNA. CNA refuses a
// submission outside a begin/end interval with CNA_RESULT_INVALID_STATE, which
// is the same decision -- but it is CNA's sentence, and a consumer who reads the
// message gets Microsoft's from the reference. The flag is this object's own for
// the same reason it is the reference's own: it is what the messages are about.

// Sentinel errors projecting the exact CLR exceptions SpriteBatch throws.
var (
	// errSpriteArgumentNull projects System.ArgumentNullException.
	errSpriteArgumentNull = errors.New("sprite batch argument is nil")
	// errSpriteInvalidOperation projects System.InvalidOperationException.
	errSpriteInvalidOperation = errors.New("sprite batch operation is not valid")
)

// The exact FrameworkResources strings the reference's throw sites load, read
// by tools/resource_strings out of the retained Microsoft.Xna.Framework.dll
// under the keys the IL names.
const (
	nullNotAllowed              = "This method does not accept null for this parameter."
	beginMustBeCalledBeforeDraw = "Begin must be called successfully before a Draw can be called."
	beginMustBeCalledBeforeEnd  = "Begin must be called successfully before End can be called."
	endMustBeCalledBeforeBegin  = "Begin cannot be called again until End has been successfully called."
)

// spriteDraw is the projection of InternalDraw's shared body. It is unexported
// and takes the destination as four floats plus the scaleDestination flag,
// which is the Vector4-and-bool the reference passes by reference.
func (b *SpriteBatch) spriteDraw(
	reference Texture2DReference,
	destinationX, destinationY, destinationZ, destinationW float32,
	scaleDestination bool,
	sourceRectangle *framework.Rectangle,
	color framework.Color,
	rotation float32,
	origin framework.Vector2,
	effects SpriteEffects,
	layerDepth float32,
) error {
	// The reference's order, preserved exactly: the texture check is at IL_0003
	// and the pair check at IL_0014, so a null texture outside a begin/end pair
	// reports the ARGUMENT, not the state.
	//
	// CNA-Go's own disposal check comes AFTER both, which is also the
	// reference's shape rather than a convenience: a disposed SpriteBatch in
	// XNA still holds its inBeginEndPair field and still throws the
	// InvalidOperationException, because InternalDraw reads the field before it
	// touches anything a disposal could have taken away.
	//
	// A nil receiver is the one case with no reference counterpart -- the CLR
	// has no null `this` -- and it is answered first because reading the field
	// would panic.
	if b == nil {
		return interop.ErrDisposed
	}
	texture := resolveTexture2D(reference)
	if texture == nil {
		return fmt.Errorf("%w: texture: %s", errSpriteArgumentNull, nullNotAllowed)
	}
	if !b.inBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, beginMustBeCalledBeforeDraw)
	}
	if b.resource() == nil || texture.nativeResource() == nil {
		return interop.ErrDisposed
	}

	if scaleDestination {
		command := interop.SpriteCommand{
			PositionX: destinationX, PositionY: destinationY,
			Red: color.R(), Green: color.G(), Blue: color.B(), Alpha: color.A(),
			Rotation: rotation, OriginX: origin.X, OriginY: origin.Y,
			ScaleX: destinationZ, ScaleY: destinationW,
			Effects: uint32(effects), LayerDepth: layerDepth,
		}
		if sourceRectangle != nil {
			command.SourceX = sourceRectangle.X
			command.SourceY = sourceRectangle.Y
			command.SourceWidth = sourceRectangle.Width
			command.SourceHeight = sourceRectangle.Height
		}
		return b.resource().DrawSprite(texture.nativeResource(), command)
	}

	command := interop.SpriteDestinationCommand{
		DestinationX: int32(destinationX), DestinationY: int32(destinationY),
		DestinationWidth: int32(destinationZ), DestinationHeight: int32(destinationW),
		Red: color.R(), Green: color.G(), Blue: color.B(), Alpha: color.A(),
		Rotation: rotation, OriginX: origin.X, OriginY: origin.Y,
		Effects: uint32(effects), LayerDepth: layerDepth,
	}
	if sourceRectangle != nil {
		command.SourceX = sourceRectangle.X
		command.SourceY = sourceRectangle.Y
		command.SourceWidth = sourceRectangle.Width
		command.SourceHeight = sourceRectangle.Height
	}
	return b.resource().DrawSpriteToDestination(texture.nativeResource(), command)
}

// DrawByTexture2DAndVector2AndColor is
// SpriteBatch::Draw(Texture2D, Vector2, Color).
//
//	destination = (position.X, position.Y, 1, 1), scaleDestination = true
//	source = nullRectangle, rotation = 0, origin = Vector2.Zero,
//	effects = 0, depth = 0
func (b *SpriteBatch) DrawByTexture2DAndVector2AndColor(
	texture Texture2DReference, position framework.Vector2, color framework.Color,
) error {
	return b.spriteDraw(texture, position.X, position.Y, 1, 1, true,
		nil, color, 0, framework.Vector2{}, 0, 0)
}

// DrawByTexture2DAndVector2AndNullableOfRectangleAndColor is
// SpriteBatch::Draw(Texture2D, Vector2, Nullable<Rectangle>, Color).
//
// It differs from the overload above by one IL instruction: the source
// rectangle is the caller's argument instead of the static nullRectangle.
func (b *SpriteBatch) DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(
	texture Texture2DReference, position framework.Vector2,
	sourceRectangle *framework.Rectangle, color framework.Color,
) error {
	return b.spriteDraw(texture, position.X, position.Y, 1, 1, true,
		sourceRectangle, color, 0, framework.Vector2{}, 0, 0)
}

// DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle
// is SpriteBatch::Draw(Texture2D, Vector2, Nullable<Rectangle>, Color, Single,
// Vector2, Vector2, SpriteEffects, Single) -- the PER-AXIS scale overload.
//
// It is the only Draw whose scale components can differ. Its uniform-scale
// sibling stores the one float into both Z and W, which is what makes them the
// same route and different members.
func (b *SpriteBatch) DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
	texture Texture2DReference, position framework.Vector2,
	sourceRectangle *framework.Rectangle, color framework.Color,
	rotation float32, origin framework.Vector2, scale framework.Vector2,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDraw(texture, position.X, position.Y, scale.X, scale.Y, true,
		sourceRectangle, color, rotation, origin, effects, layerDepth)
}

// DrawByTexture2DAndRectangleAndColor is
// SpriteBatch::Draw(Texture2D, Rectangle, Color).
//
//	destination = (rect.X, rect.Y, rect.Width, rect.Height)
//	scaleDestination = FALSE
//
// The rectangle's four int32 fields are stored into a Vector4 with `conv.r4`,
// so the reference converts to float and CNA takes int32 back. The round trip is
// exact for every rectangle the CLR can hold: an int32 whose magnitude exceeds
// 2^24 loses precision as a float32, and that is the reference's own loss, not
// a projection artefact -- XNA's own SpriteInfo is a Vector4.
func (b *SpriteBatch) DrawByTexture2DAndRectangleAndColor(
	texture Texture2DReference, destinationRectangle framework.Rectangle, color framework.Color,
) error {
	return b.spriteDraw(texture,
		float32(destinationRectangle.X), float32(destinationRectangle.Y),
		float32(destinationRectangle.Width), float32(destinationRectangle.Height), false,
		nil, color, 0, framework.Vector2{}, 0, 0)
}

// DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor is
// SpriteBatch::Draw(Texture2D, Rectangle, Nullable<Rectangle>, Color).
func (b *SpriteBatch) DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(
	texture Texture2DReference, destinationRectangle framework.Rectangle,
	sourceRectangle *framework.Rectangle, color framework.Color,
) error {
	return b.spriteDraw(texture,
		float32(destinationRectangle.X), float32(destinationRectangle.Y),
		float32(destinationRectangle.Width), float32(destinationRectangle.Height), false,
		sourceRectangle, color, 0, framework.Vector2{}, 0, 0)
}

// DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle
// is SpriteBatch::Draw(Texture2D, Rectangle, Nullable<Rectangle>, Color, Single,
// Vector2, SpriteEffects, Single).
//
// It takes NO scale, and that is not an omission: a destination rectangle
// already states the size, so the reference's parameter list has eight
// arguments where its position-based sibling has nine.
func (b *SpriteBatch) DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle(
	texture Texture2DReference, destinationRectangle framework.Rectangle,
	sourceRectangle *framework.Rectangle, color framework.Color,
	rotation float32, origin framework.Vector2,
	effects SpriteEffects, layerDepth float32,
) error {
	return b.spriteDraw(texture,
		float32(destinationRectangle.X), float32(destinationRectangle.Y),
		float32(destinationRectangle.Width), float32(destinationRectangle.Height), false,
		sourceRectangle, color, rotation, origin, effects, layerDepth)
}
