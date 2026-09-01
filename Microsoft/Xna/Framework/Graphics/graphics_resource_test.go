package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// TestGraphicsResourceNameAndTagAreManagedStorage pins the two members whose
// reference bodies look like they reach a device and do not.
//
//	get_Name  if (_internalHandle != 0)
//	              return _parent.Resources.GetCachedName(_internalHandle);
//	          return _localName;
//
// DeviceResourceManager::GetCachedName is a `Dictionary<ulong, ResourceData>`
// under a Monitor, answering String.Empty for an absent key. Nothing native, no
// throw site, so neither accessor carries an error and an unnamed resource
// answers with the empty string rather than reporting.
func TestGraphicsResourceNameAndTagAreManagedStorage(t *testing.T) {
	resource := newGraphicsResource(nil, nil)
	if got := resource.Name(); got != "" {
		t.Fatalf("Name on a new resource = %q, want the empty string GetCachedName answers with", got)
	}
	if got := resource.Tag(); got != nil {
		t.Fatalf("Tag on a new resource = %v, want nil", got)
	}
	resource.SetName("logo")
	resource.SetTag(42)
	if resource.Name() != "logo" || resource.Tag() != 42 {
		t.Fatalf("Name/Tag = %q/%v after setting", resource.Name(), resource.Tag())
	}
	// The reference validates nothing and stores null the same as anything
	// else, so clearing is not a special case.
	resource.SetName("")
	resource.SetTag(nil)
	if resource.Name() != "" || resource.Tag() != nil {
		t.Fatalf("Name/Tag = %q/%v after clearing", resource.Name(), resource.Tag())
	}
}

// TestGraphicsResourceToStringFallsBackToTheRuntimeType pins the identity site.
//
//	ToString()
//	  local = name; if (!IsNullOrEmpty(local)) return local;
//	  return base.ToString();      // System.Object -> the RUNTIME type's name
//
// The fallback is the object's own type, which under composition is the
// outermost object rather than the base half.
func TestGraphicsResourceToStringFallsBackToTheRuntimeType(t *testing.T) {
	bare := newGraphicsResource(nil, nil)
	if got := bare.ToString(); got != "Microsoft.Xna.Framework.Graphics.GraphicsResource" {
		t.Fatalf("bare ToString = %q", got)
	}
	bare.SetName("named")
	if got := bare.ToString(); got != "named" {
		t.Fatalf("named ToString = %q, want the name", got)
	}

	texture := newTexture2D(nil, nil, interop.TextureInfo{Width: 4, Height: 2, Levels: 1}, nil)
	if got := texture.ToString(); got != "Microsoft.Xna.Framework.Graphics.Texture2D" {
		t.Fatalf("Texture2D ToString = %q; the fallback is the RUNTIME type, resolved through the CLR `this` across TWO composition links", got)
	}
	texture.SetName("logo")
	if got := texture.ToString(); got != "logo" {
		t.Fatalf("named Texture2D ToString = %q", got)
	}
}

// TestGraphicsResourceDisposalIsIdempotentAndRaisesOnce pins the flag that
// distinguishes this family from GameComponent's.
//
//	~GraphicsResource()  if (!isDisposed) { isDisposed = true;
//	                                        Disposing(this, EventArgs.Empty); }
//
// GameComponent has no such flag and re-runs on every call. This one raises
// exactly once, ever, and the flag is set BEFORE the event, so a handler
// observes IsDisposed == true.
func TestGraphicsResourceDisposalIsIdempotentAndRaisesOnce(t *testing.T) {
	resource := newGraphicsResource(nil, nil)
	raises := 0
	observed := false
	if _, err := resource.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		raises++
		observed = resource.IsDisposed()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if resource.IsDisposed() {
		t.Fatal("a new resource reports disposed")
	}
	if err := resource.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raises != 1 || !observed {
		t.Fatalf("Disposing raised %d times, handler observed IsDisposed=%t", raises, observed)
	}
	if err := resource.DisposeByNone(); err != nil {
		t.Fatalf("second Dispose: %v", err)
	}
	if raises != 1 {
		t.Fatalf("Disposing raised %d times over two calls; the isDisposed guard makes this family's disposal idempotent", raises)
	}
}

// TestGraphicsResourceDisposeFalseSetsTheFlagAndRaisesNothing pins the other
// branch: `!GraphicsResource()` sets the flag, and only `~GraphicsResource()`
// announces.
func TestGraphicsResourceDisposeFalseSetsTheFlagAndRaisesNothing(t *testing.T) {
	resource := newGraphicsResource(nil, nil)
	raises := 0
	if _, err := resource.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		raises++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := resource.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !resource.IsDisposed() {
		t.Fatal("Dispose(false) left IsDisposed false; !GraphicsResource() sets the flag on both paths")
	}
	if raises != 0 {
		t.Fatalf("Dispose(false) raised Disposing %d times; only ~GraphicsResource() announces", raises)
	}
	// And the flag it set now suppresses the announcement a later Dispose(true)
	// would otherwise make, which is the reference's behaviour and not an
	// artefact: both halves read the same field.
	if err := resource.DisposeByNone(); err != nil {
		t.Fatalf("Dispose(true) after Finalize: %v", err)
	}
	if raises != 0 {
		t.Fatalf("Disposing raised %d times after the flag was already set", raises)
	}
}

// TestComposedResourcesAnnounceTheWholeObject is the identity site measured
// behaviourally, over both derived families.
func TestComposedResourcesAnnounceTheWholeObject(t *testing.T) {
	texture := newTexture2D(nil, nil, interop.TextureInfo{Width: 8, Height: 8, Levels: 1}, nil)
	var textureSender any
	if _, err := texture.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		textureSender = sender
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := texture.DisposeByNone(); err != nil {
		t.Fatalf("Texture2D Dispose: %v", err)
	}
	if textureSender != any(texture) {
		t.Fatalf("Texture2D announced %T; the reference pushes `ldarg.0`, which is the whole object", textureSender)
	}

	batch := &SpriteBatch{graphicsResource: newGraphicsResource(nil, nil)}
	batch.graphicsResource.bindDerived(batch)
	var batchSender any
	if _, err := batch.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		batchSender = sender
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := batch.DisposeByNone(); err != nil {
		t.Fatalf("SpriteBatch Dispose: %v", err)
	}
	if batchSender != any(batch) {
		t.Fatalf("SpriteBatch announced %T, want the batch", batchSender)
	}
	if got := batch.ToString(); got != "Microsoft.Xna.Framework.Graphics.SpriteBatch" {
		t.Fatalf("SpriteBatch ToString = %q", got)
	}
}

// TestTextureDescriptionIsCachedAtConstruction pins the split
// Texture2D::InitializeDescription makes between the requested format and the
// created surface's.
//
//	if (!format.HasValue) format = ConvertWindowsFormatToXna(desc.Format);
//	_width = desc.Width; _height = desc.Height;
//	Texture::InitializeDescription(format.Value);
//
// A CONSTRUCTOR passes the format it was asked for; FromStream passes no value
// and takes the decoder's. Width, height and level count always come from the
// created surface.
func TestTextureDescriptionIsCachedAtConstruction(t *testing.T) {
	info := interop.TextureInfo{Width: 64, Height: 32, Levels: 7, Format: uint32(SurfaceFormatBgr565)}

	requested := SurfaceFormatDxt5
	fromConstructor := newTexture2D(nil, nil, info, &requested)
	if got := fromConstructor.Format(); got != SurfaceFormatDxt5 {
		t.Fatalf("constructor Format = %v, want the REQUESTED format the reference stores", got)
	}

	fromStream := newTexture2D(nil, nil, info, nil)
	if got := fromStream.Format(); got != SurfaceFormatBgr565 {
		t.Fatalf("FromStream Format = %v, want the created surface's; the reference passes no value there", got)
	}

	for _, texture := range []*Texture2D{fromConstructor, fromStream} {
		if texture.Width() != 64 || texture.Height() != 32 {
			t.Fatalf("dimensions = %dx%d, want the created surface's 64x32", texture.Width(), texture.Height())
		}
		if texture.LevelCount() != 7 {
			t.Fatalf("LevelCount = %d, want the created texture's 7", texture.LevelCount())
		}
		if got := texture.Bounds(); got != framework.NewRectangle(0, 0, 64, 32) {
			t.Fatalf("Bounds = %+v", got)
		}
	}

	// Every one of them answers after disposal, because none checks it: the
	// reference's getters are `ldfld` and CNA-Go's read the same cached values.
	if err := fromStream.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if !fromStream.IsDisposed() {
		t.Fatal("IsDisposed is false after Dispose")
	}
	if fromStream.Width() != 64 || fromStream.LevelCount() != 7 || fromStream.Format() != SurfaceFormatBgr565 {
		t.Fatal("a disposed texture stopped answering with its cached description")
	}
}

// TestDerivedDisposeReachesTheDerivedBody pins the `callvirt` the inherited
// Dispose() carries, from the other end: a Texture2D disposed through the
// INHERITED member must run Texture2D's override, which releases the native
// texture, and not GraphicsResource's, which only sets a flag.
func TestDerivedDisposeReachesTheDerivedBody(t *testing.T) {
	// A resource with no runtime cannot be destroyed, so the observable here is
	// the ORDER: the derived override releases before the base sets the flag,
	// so a handler sees a resource whose native half is already released.
	texture := newTexture2D(nil, nil, interop.TextureInfo{Width: 1, Height: 1, Levels: 1}, nil)
	order := []string{}
	if _, err := texture.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		order = append(order, "disposing")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := texture.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if len(order) != 1 || order[0] != "disposing" {
		t.Fatalf("order = %v", order)
	}
	if !texture.IsDisposed() {
		t.Fatal("the base half never ran; DisposeByNone must reach the derived override, which calls it in a finally")
	}
}
