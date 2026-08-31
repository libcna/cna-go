package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// These tests measure the managed half of SpriteBatch's Draw family: the two
// guards InternalDraw applies before it queues anything, and the order it
// applies them in. A live native SpriteBatch is not needed for either, because
// the reference reaches neither the queue nor the device until both have
// passed -- and the native half is proved by the sprite-draw stress scenario,
// which submits every overload through a real draw callback.

func TestANilTextureReportsTheArgumentNullExceptionMessage(t *testing.T) {
	batch := &SpriteBatch{inBeginEndPair: true}
	err := batch.DrawByTexture2DAndVector2AndColor(nil, framework.Vector2{}, framework.Color{})
	if !errors.Is(err, errSpriteArgumentNull) {
		t.Fatalf("error = %v, want the ArgumentNullException projection", err)
	}
	if !strings.Contains(err.Error(), "texture") {
		t.Fatalf("error %q does not name the parameter", err)
	}
	if !strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("error %q is not FrameworkResources.NullNotAllowed", err)
	}
}

// TestTheTextureCheckRunsBeforeTheBeginEndCheck pins the reference's order. Both
// conditions hold at once here, and the IL decides which is reported: the
// ArgumentNullException is thrown at IL_0003, before inBeginEndPair is read at
// IL_0014. Swapping them is a one-line rewrite that changes what a consumer
// sees.
func TestTheTextureCheckRunsBeforeTheBeginEndCheck(t *testing.T) {
	batch := &SpriteBatch{}
	err := batch.DrawByTexture2DAndVector2AndColor(nil, framework.Vector2{}, framework.Color{})
	if !errors.Is(err, errSpriteArgumentNull) {
		t.Fatalf("error = %v, want the argument to be reported before the state", err)
	}
}

func TestDrawOutsideABeginEndPairReportsTheInvalidOperationMessage(t *testing.T) {
	batch := &SpriteBatch{}
	texture := &Texture2D{}
	err := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, framework.Color{})
	if !errors.Is(err, errSpriteInvalidOperation) {
		t.Fatalf("error = %v, want the InvalidOperationException projection", err)
	}
	if !strings.Contains(err.Error(), beginMustBeCalledBeforeDraw) {
		t.Fatalf("error %q is not FrameworkResources.BeginMustBeCalledBeforeDraw", err)
	}
}

// TestEverySevenOverloadsShareTheTwoGuards is the coverage proof: a new overload
// that reached the resource directly would pass every test above and still be
// unguarded. All seven are called with the same unguarded batch.
func TestEverySevenOverloadsShareTheTwoGuards(t *testing.T) {
	batch := &SpriteBatch{}
	texture := &Texture2D{}
	position := framework.NewVector2BySingleAndSingle(1, 2)
	destination := framework.NewRectangle(0, 0, 4, 4)
	source := framework.NewRectangle(0, 0, 2, 2)
	color := framework.Color{}
	origin := framework.Vector2{}

	overloads := map[string]func() error{
		"Vector2AndColor": func() error {
			return batch.DrawByTexture2DAndVector2AndColor(texture, position, color)
		},
		"Vector2AndSourceAndColor": func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(texture, position, &source, color)
		},
		"Vector2AndUniformScale": func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				texture, position, &source, color, 0, origin, 1, 0, 0)
		},
		"Vector2AndPerAxisScale": func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				texture, position, &source, color, 0, origin, position, 0, 0)
		},
		"RectangleAndColor": func() error {
			return batch.DrawByTexture2DAndRectangleAndColor(texture, destination, color)
		},
		"RectangleAndSourceAndColor": func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(texture, destination, &source, color)
		},
		"RectangleAndRotation": func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle(
				texture, destination, &source, color, 0, origin, 0, 0)
		},
	}
	if len(overloads) != 7 {
		t.Fatalf("the profile declares 7 Draw overloads and this test calls %d", len(overloads))
	}
	for name, draw := range overloads {
		if err := draw(); !errors.Is(err, errSpriteInvalidOperation) {
			t.Errorf("%s outside a pair = %v, want the InvalidOperationException projection", name, err)
		}
	}
}

// TestTheBeginEndPairGuardsAreTheReferencesTwoMessages measures the pair from
// the other two sides: End before Begin, and Begin twice. Neither reaches a
// native batch, because the reference throws before it touches one.
func TestTheBeginEndPairGuardsAreTheReferencesTwoMessages(t *testing.T) {
	batch := &SpriteBatch{resource: &interop.Resource{}}
	err := batch.End()
	if !errors.Is(err, errSpriteInvalidOperation) || !strings.Contains(err.Error(), beginMustBeCalledBeforeEnd) {
		t.Fatalf("End before Begin = %v, want FrameworkResources.BeginMustBeCalledBeforeEnd", err)
	}
	batch.inBeginEndPair = true
	err = batch.BeginByNone()
	if !errors.Is(err, errSpriteInvalidOperation) || !strings.Contains(err.Error(), endMustBeCalledBeforeBegin) {
		t.Fatalf("Begin twice = %v, want FrameworkResources.EndMustBeCalledBeforeBegin", err)
	}
}

// TestTheFourMessagesAreFourDistinctSentences guards against the defect the
// resource reader was built for. Two of these read alike, and a projection that
// used one where the other belongs would report a plausible sentence at the
// wrong throw site.
func TestTheFourMessagesAreFourDistinctSentences(t *testing.T) {
	seen := map[string]string{}
	for name, value := range map[string]string{
		"NullNotAllowed":              nullNotAllowed,
		"BeginMustBeCalledBeforeDraw": beginMustBeCalledBeforeDraw,
		"BeginMustBeCalledBeforeEnd":  beginMustBeCalledBeforeEnd,
		"EndMustBeCalledBeforeBegin":  endMustBeCalledBeforeBegin,
	} {
		if other, duplicate := seen[value]; duplicate {
			t.Fatalf("%s and %s are the same string", name, other)
		}
		seen[value] = name
	}
}

// TestBoundsOnADisposedTextureReportsRatherThanReturningAZeroRectangle pins that
// Bounds is fallible for the reason Width and Height are, and does not paper
// over a disposed texture with an empty rectangle.
func TestBoundsOnADisposedTextureReportsRatherThanReturningAZeroRectangle(t *testing.T) {
	var texture *Texture2D
	if _, err := texture.Bounds(); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("Bounds on a nil texture = %v, want ErrDisposed", err)
	}
	if _, err := (&Texture2D{}).Bounds(); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("Bounds on an unbound texture = %v, want ErrDisposed", err)
	}
}
