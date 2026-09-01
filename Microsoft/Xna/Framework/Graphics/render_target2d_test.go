package graphics

import (
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// TestARenderTargetSatisfiesEveryTexture2DParameterPosition is the
// substitutability claim, stated as the compiler checks it.
//
// Every assignment below fails to compile if the rule is broken, which is the
// strongest form the claim can take in Go: a `*Texture2D` parameter would make
// the render-target lines a type error rather than a test failure.
func TestARenderTargetSatisfiesEveryTexture2DParameterPosition(t *testing.T) {
	var fromTexture Texture2DReference = &Texture2D{}
	var fromRenderTarget Texture2DReference = &RenderTarget2D{}
	if fromTexture == nil || fromRenderTarget == nil {
		t.Fatal("a Texture2D and a RenderTarget2D must both satisfy Texture2DReference")
	}
	// The seven live positions and the six generic-method receivers take the
	// interface, so both values reach them. These are compile-time facts; the
	// call results are not what is under test.
	batch := &SpriteBatch{}
	_ = batch.DrawByTexture2DAndVector2AndColor
	_ = Texture2DSetDataBySliceOfT[framework.Color]
	_ = Texture2DGetDataBySliceOfT[framework.Color]
}

// TestResolveTexture2DTreatsBothNilShapesAsOneNull pins the guard the interface
// makes necessary.
//
// The reference sees ONE null and reports one ArgumentNullException. Go has two
// shapes -- a nil interface, and a non-nil interface holding a typed nil -- and
// a consumer can write either. Both must reach the same message.
func TestResolveTexture2DTreatsBothNilShapesAsOneNull(t *testing.T) {
	if resolveTexture2D(nil) != nil {
		t.Fatal("a nil interface resolved to a texture")
	}
	var typedNilTexture *Texture2D
	if resolveTexture2D(typedNilTexture) != nil {
		t.Fatal("an interface holding a nil *Texture2D resolved to a texture")
	}
	var typedNilTarget *RenderTarget2D
	if resolveTexture2D(typedNilTarget) != nil {
		t.Fatal("an interface holding a nil *RenderTarget2D resolved to a texture")
	}
	// And both spellings reach the reference's own sentence, through the same
	// guard, on a batch that is not in a begin/end pair -- which is the order
	// InternalDraw applies its two checks in.
	batch := &SpriteBatch{graphicsResource: newGraphicsResource(nil, nil)}
	for name, reference := range map[string]Texture2DReference{
		"nil interface":       nil,
		"typed nil Texture2D": typedNilTexture,
		"typed nil target":    typedNilTarget,
	} {
		err := batch.DrawByTexture2DAndVector2AndColor(reference, framework.Vector2{},
			framework.NewColorByInt32AndInt32AndInt32AndInt32(0, 0, 0, 255))
		if err == nil || !strings.Contains(err.Error(), nullNotAllowed) {
			t.Fatalf("%s: Draw = %v, want FrameworkResources.NullNotAllowed", name, err)
		}
	}
}

// TestARenderTargetIsItsOwnRuntimeType pins the CLR `this` across the deepest
// chain in the profile.
//
//	RenderTarget2D -> Texture2D -> Texture -> GraphicsResource
//
// GraphicsResource::ToString falls back to System.Object::ToString, which is the
// RUNTIME type's full name. Three forwarding links stand between the raise site
// and the object, and every one of them has to pass the binding on.
func TestARenderTargetIsItsOwnRuntimeType(t *testing.T) {
	target := &RenderTarget2D{texture: newTexture2D(nil, nil, textureInfo(4, 4), nil)}
	target.texture.bindDerived(target)
	if got := target.ToString(); got != "Microsoft.Xna.Framework.Graphics.RenderTarget2D" {
		t.Fatalf("ToString = %q; the CLR `this` must survive three forwarding links", got)
	}
	target.SetName("shadow-map")
	if got := target.ToString(); got != "shadow-map" {
		t.Fatalf("named ToString = %q", got)
	}
}

// TestARenderTargetAnnouncesItselfWhenDisposed is the other identity site over
// the same chain.
func TestARenderTargetAnnouncesItselfWhenDisposed(t *testing.T) {
	target := &RenderTarget2D{texture: newTexture2D(nil, nil, textureInfo(2, 2), nil)}
	target.texture.bindDerived(target)
	var sender any
	raises := 0
	if _, err := target.AddDisposingHandler(func(source any, args *framework.EventArgs) error {
		sender = source
		raises++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := target.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if err := target.DisposeByNone(); err != nil {
		t.Fatalf("second Dispose: %v", err)
	}
	if raises != 1 {
		t.Fatalf("Disposing raised %d times; GraphicsResource's isDisposed flag makes this idempotent", raises)
	}
	if sender != any(target) {
		t.Fatalf("Disposing announced %T, want the RenderTarget2D", sender)
	}
	if !target.IsDisposed() {
		t.Fatal("IsDisposed is false after disposal")
	}
}

// TestRenderTargetDescriptionDefaultsComeFromTheConstructorIL pins the five
// `ldc.i4.0` the three-argument constructor passes.
func TestRenderTargetDescriptionDefaultsComeFromTheConstructorIL(t *testing.T) {
	// A device is required, so the defaults are checked through the refusal the
	// reference's own first guard produces, and through the zero values the
	// projection reports for an unbound target.
	if _, err := NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32(nil, 4, 4); err == nil {
		t.Fatal("a nil device was accepted")
	} else if !strings.Contains(err.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatalf("nil device = %v, want FrameworkResources.DeviceCannotBeNullOnResourceCreate", err)
	}
	var zero *RenderTarget2D
	if zero.RenderTargetUsage() != RenderTargetUsageDiscardContents {
		t.Fatalf("the constructor default usage is DiscardContents")
	}
	if zero.DepthStencilFormat() != DepthFormatNone || zero.MultiSampleCount() != 0 {
		t.Fatal("the constructor defaults are DepthFormat.None and no multisampling")
	}
}

func textureInfo(width, height uint32) interop.TextureInfo {
	return interop.TextureInfo{Width: width, Height: height, Levels: 1}
}
