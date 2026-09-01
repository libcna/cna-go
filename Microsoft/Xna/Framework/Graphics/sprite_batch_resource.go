package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The ten members SpriteBatch inherits from GraphicsResource, plus the disposal
// override it declares itself.
//
// SpriteBatch derives from GraphicsResource directly. Composing the base
// therefore closes seven CLR members here at the same time as it closes them on
// Texture2D, and that is the point of composing a base rather than a type: one
// projection, every derived type.

// clrTypeName is System.Object::ToString's answer for a SpriteBatch, which is
// what GraphicsResource::ToString falls back to for an unnamed resource.
func (b *SpriteBatch) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.SpriteBatch" }

// GraphicsDevice is GraphicsResource::get_GraphicsDevice, the device the batch
// was created on.
func (b *SpriteBatch) GraphicsDevice() *GraphicsDevice {
	if b == nil {
		return nil
	}
	return b.graphicsResource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (b *SpriteBatch) Name() string {
	if b == nil {
		return ""
	}
	return b.graphicsResource.Name()
}

// SetName is GraphicsResource::set_Name.
func (b *SpriteBatch) SetName(value string) {
	if b == nil {
		return
	}
	b.graphicsResource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (b *SpriteBatch) Tag() any {
	if b == nil {
		return nil
	}
	return b.graphicsResource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (b *SpriteBatch) SetTag(value any) {
	if b == nil {
		return
	}
	b.graphicsResource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed. SpriteBatch's own
// Dispose(bool) reads it too, as its guard.
func (b *SpriteBatch) IsDisposed() bool {
	if b == nil {
		return true
	}
	return b.graphicsResource.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (b *SpriteBatch) ToString() string {
	if b == nil {
		return ""
	}
	return b.graphicsResource.ToString()
}

// AddDisposingHandler is add_Disposing, on the base's registration list.
func (b *SpriteBatch) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errGraphicsResourceNil
	}
	return b.graphicsResource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (b *SpriteBatch) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errGraphicsResourceNil
	}
	return b.graphicsResource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), whose `callvirt Dispose(bool)`
// reaches THIS type's override.
func (b *SpriteBatch) DisposeByNone() error {
	return b.DisposeByBoolean(true)
}

// DisposeByBoolean is SpriteBatch::Dispose(bool), reproduced from its IL:
//
//	try {
//	    if (disposing && !IsDisposed) {
//	        if (spriteEffect != null) spriteEffect.Dispose();
//	        DisposePlatformData();          // the vertex and index buffers
//	    }
//	} finally {
//	    base.Dispose(disposing);
//	}
//
// # The guard is not Texture2D's, and that is deliberate
//
// This override releases only when `disposing` is TRUE. Texture2D's releases on
// both paths, because its `!Texture2D()` passes a hardcoded `ldc.i4.1` to
// ReleaseNativeObject regardless of the flag. The two ILs genuinely disagree,
// and each is reproduced as written: a SpriteBatch reaching this through the
// finalizer path leaves its platform data to the CLR, and a Texture2D does not.
//
// # What CNA-Go releases here
//
// The reference's SpriteBatch owns no handle of its own -- what it disposes are
// its effect and its two dynamic buffers, each a GraphicsResource in its own
// right. CNA's sprite batch IS one native object, so the one owned CNA handle
// stands in for all three, and it is released at the same moment and under the
// same guard.
//
// The base call is in a `finally`, so it runs even when the release fails.
func (b *SpriteBatch) DisposeByBoolean(disposing bool) error {
	if b == nil {
		return errGraphicsResourceNil
	}
	var released error
	if disposing && !b.graphicsResource.IsDisposed() {
		released = b.graphicsResource.releaseNativeObject()
	}
	baseErr := b.graphicsResource.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
