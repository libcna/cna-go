package graphics

import (
	"fmt"
	"io"

	"github.com/openeggbert/cna-go/internal/interop"
)

// Texture2D's stream surface: the sized FromStream, and the two SaveAs
// overloads.
//
// # One private method behind three public doors, again
//
// SaveAsPng and SaveAsJpeg are eleven bytes of IL each, over one private
// SaveAsImage(Stream, XnaImageFormat, int, int):
//
//	SaveAsPng   ldc.i4.2   // XnaImageFormat.Png
//	SaveAsJpeg  ldc.i4.0   // XnaImageFormat.Jpeg
//
// and CNA numbers its own image formats the OTHER way -- PNG 0, JPEG 1. This is
// one of the few identities in the profile that does not cross unchanged, so the
// mapping is made once, here, where both names are visible, instead of being
// carried as a shared constant that looks like it agrees.
//
// The five-argument FromStream is twenty-three bytes:
//
//	XnaImageOperation op = (zoom == false) ? 1 : 3;
//	return new Texture2D(graphicsDevice, stream, width, height, op);
//
// and CNA's decode info carries the same distinction as a bool with the same
// meaning: cover-and-crop when true, fit while preserving the aspect ratio when
// false.

// Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean is
// Texture2D::FromStream(GraphicsDevice, Stream, Int32, Int32, Boolean).
//
// The reader is read from its CURRENT position to EOF, and CNA-Go does not close
// it -- the same contract the two-argument overload has had since Foundation 1,
// and the one XNA has, because a Stream a caller opened is a Stream the caller
// closes.
func Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(
	graphicsDevice *GraphicsDevice, stream io.Reader, width, height int32, zoom bool,
) (*Texture2D, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if stream == nil {
		return nil, fmt.Errorf("%w: stream: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if width < 0 || height < 0 {
		return nil, fmt.Errorf("%w: a requested texture dimension is negative: %dx%d",
			errGraphicsResourceArgument, width, height)
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	resource, info, err := graphicsDevice.device.CreateTextureFromEncodedSized(
		data, uint32(width), uint32(height), zoom)
	if err != nil {
		return nil, err
	}
	return &Texture2D{resource: resource, info: info}, nil
}

// SaveAsPng is
// Texture2D::SaveAsPng(Stream, Int32, Int32).
func (t *Texture2D) SaveAsPng(stream io.Writer, width, height int32) error {
	return t.saveAsImage(stream, interop.TextureImageFormatPNG, width, height)
}

// SaveAsJpeg is
// Texture2D::SaveAsJpeg(Stream, Int32, Int32).
func (t *Texture2D) SaveAsJpeg(stream io.Writer, width, height int32) error {
	return t.saveAsImage(stream, interop.TextureImageFormatJPEG, width, height)
}

// saveAsImage is the projection of the private SaveAsImage's shared prologue.
//
// The reference's three guards are:
//
//	if (stream == null)      throw new ArgumentNullException("stream", FrameworkResources.NullNotAllowed);
//	if (!stream.CanWrite)    throw new ArgumentException("stream");
//	if (format != 0 && format != 2) throw new ArgumentException("format");
//
// The FIRST is reproduced. The second has no Go counterpart and is recorded
// rather than invented: an io.Writer has no CanWrite, and every one of them
// either writes or reports an error when asked, so there is no state to test
// before writing. The third is unreachable from either public entry point --
// both pass a constant -- and it stays unreachable here for the same reason.
//
// Note the second guard's shape while it is in view: `new ArgumentException(string)`
// takes a MESSAGE, not a parameter name, so the reference's exception reads
// literally "stream". That is the CLR's own quirk and is recorded because a
// projection that turned it into a parameter name would be reporting something
// the reference does not say.
func (t *Texture2D) saveAsImage(stream io.Writer, imageFormat uint32, width, height int32) error {
	if stream == nil {
		return fmt.Errorf("%w: stream: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if t == nil || t.resource == nil {
		return interop.ErrDisposed
	}
	if width < 0 || height < 0 {
		return fmt.Errorf("%w: an encoded dimension is negative: %dx%d",
			errGraphicsResourceArgument, width, height)
	}
	encoded, err := t.resource.EncodeTexture(imageFormat, uint32(width), uint32(height))
	if err != nil {
		return err
	}
	// The reference writes the whole encoded image and returns; a short write is
	// an io.Writer failure, which Go reports and the CLR's Stream.Write would
	// have thrown.
	if _, err := stream.Write(encoded); err != nil {
		return err
	}
	return nil
}
