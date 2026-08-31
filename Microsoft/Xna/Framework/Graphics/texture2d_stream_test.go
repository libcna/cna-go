package graphics

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// The managed half of Texture2D's stream surface is the prologue: the guards
// that run before CNA is reached, and the format identity the two SaveAs
// members choose. The encode itself is proved by the device-state stress
// scenario, which checks the PNG signature and the JPEG SOI marker on bytes CNA
// produced.

func TestSaveRefusesANilWriterWithTheReferenceMessage(t *testing.T) {
	texture := &Texture2D{}
	for name, save := range map[string]func() error{
		"SaveAsPng":  func() error { return texture.SaveAsPng(nil, 8, 8) },
		"SaveAsJpeg": func() error { return texture.SaveAsJpeg(nil, 8, 8) },
	} {
		err := save()
		if !errors.Is(err, errGraphicsResourceArgumentNull) {
			t.Errorf("%s(nil) = %v, want the ArgumentNullException projection", name, err)
		}
		if err == nil || !strings.Contains(err.Error(), nullNotAllowed) {
			t.Errorf("%s(nil) = %v, want FrameworkResources.NullNotAllowed", name, err)
		}
	}
}

// TestTheNilWriterCheckRunsBeforeTheDisposalCheck pins the reference's order:
// the stream test is the first statement of SaveAsImage, at IL_0000, before
// anything about the texture is read.
func TestTheNilWriterCheckRunsBeforeTheDisposalCheck(t *testing.T) {
	var texture *Texture2D
	if err := texture.SaveAsPng(nil, 8, 8); !errors.Is(err, errGraphicsResourceArgumentNull) {
		t.Fatalf("a nil texture and a nil writer reported %v, want the argument", err)
	}
	if err := (&Texture2D{}).SaveAsPng(&bytes.Buffer{}, 8, 8); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("an unbound texture with a real writer reported %v, want ErrDisposed", err)
	}
}

// TestTheTwoImageFormatIdentitiesAreCnaAndNotXna is the one identity in this
// milestone that does NOT cross unchanged, and the test exists because it looks
// like it should.
//
//	XNA SharedConstants.XnaImageFormat   Jpeg 0   Png 2
//	CNA_TEXTURE_IMAGE_FORMAT_*           PNG  0   JPEG 1
//
// Passing XNA's own constant through would encode a JPEG under SaveAsPng, and
// nothing about the call would report it.
func TestTheTwoImageFormatIdentitiesAreCnaAndNotXna(t *testing.T) {
	if interop.TextureImageFormatPNG != 0 || interop.TextureImageFormatJPEG != 1 {
		t.Fatalf("CNA image formats are PNG=%d JPEG=%d, want 0 and 1",
			interop.TextureImageFormatPNG, interop.TextureImageFormatJPEG)
	}
	// XNA's own numbering, stated here so the difference is visible rather than
	// implied: a projection that forwarded these would be wrong in both members.
	const xnaJpeg, xnaPng = 0, 2
	if interop.TextureImageFormatPNG == xnaPng || interop.TextureImageFormatJPEG == xnaJpeg {
		t.Fatal("the CNA identities coincide with XNA's, so this test no longer measures the difference")
	}
}

func TestSizedFromStreamRefusesItsArgumentsBeforeReadingTheStream(t *testing.T) {
	reader := bytes.NewReader([]byte{1, 2, 3})
	if _, err := Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(
		nil, reader, 4, 4, false); !errors.Is(err, errGraphicsResourceArgumentNull) {
		t.Fatalf("a nil device reported %v", err)
	}
	if reader.Len() != 3 {
		t.Fatal("the stream was read before the device was checked")
	}
	// The DEVICE is checked before the stream, which is the reference's order:
	// FromStream forwards to the constructor and CreateTexture's first throw is
	// the device one. A device-less facade with a nil stream must therefore
	// report the device, and this asserts the MESSAGE so it cannot pass because
	// both errors share a sentinel.
	_, bothNil := Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(nil, nil, 4, 4, false)
	if bothNil == nil || !strings.Contains(bothNil.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatalf("a nil device and a nil stream reported %v, want the device message", bothNil)
	}
	_, nilStream := Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(
		&GraphicsDevice{device: nil}, nil, 4, 4, false)
	if nilStream == nil || !strings.Contains(nilStream.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatalf("a device-less facade reported %v, want the device message", nilStream)
	}
}
