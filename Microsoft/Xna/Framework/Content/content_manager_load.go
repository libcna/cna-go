package content

import (
	"errors"
	"fmt"

	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// ---------------------------------------------------------------------------
// Foundation 63 — the two generic ContentManager members.
// ---------------------------------------------------------------------------

// # Both are package functions, on the settled generic-method rule
//
// Go methods cannot declare type parameters, so a CLR generic instance method
// projects as a package-level generic FUNCTION taking the receiver first. These
// are the first two members in the profile whose RETURN is the type parameter,
// which is what the verifier fix below exists for.
//
// # What T may be, and why the set is closed
//
// The reference's Load<T> reads an `.xnb` and hands back whatever the content
// reader produced, so its T is open. CNA's content pipeline exposes one route
// per asset kind, and CNA-Go binds the one whose asset type it projects:
//
//	*graphics.Texture2D    cna_content_manager_load_texture2d
//	*graphics.SpriteFont   cna_content_manager_load_sprite_font
//	*graphics.Effect       cna_content_manager_load_effect
//
// A T outside that set is refused BY NAME, which is the same shape the texture
// transfer rule takes: the CLR would fail at runtime for an asset whose reader
// does not match, and reporting the type is more useful than reporting a cast.
//
// SoundEffect has a CNA route too and is absent here for one reason: CNA-Go
// does not project that type yet. It is a missing TYPE rather than a missing
// loader, and adding a loader for a type that has no Go identity would be a
// route with nothing to return. SpriteFont left that list in Foundation 69 and
// Effect in Foundation 72, each when its type became real -- which is the rule
// working, not an exception to it.
//
// TextureCube became real in Foundation 71 and is still absent, for a DIFFERENT
// reason worth naming: cna_content_manager_load_texture_cube exists, but the
// type's own transfers are refused by every qualified renderer, so a loader
// would hand back an object whose data members cannot be reached. That is a
// renderer limitation rather than a missing type, and it is recorded rather
// than papered over with a loader nothing can use.

// errContentUnsupportedAsset projects the refusal a T outside the closed set
// gets. The reference has no counterpart -- its Load<T> would throw
// ContentLoadException from the reader -- and the difference is recorded: the
// projection refuses BEFORE the load, on the Go type, rather than after it on
// the asset's declared reader.
var errContentUnsupportedAsset = errors.New("the content manager cannot load this asset type")

// ContentManagerLoad is ContentManager::Load<T>(String).
//
//	if (assetName == null) throw new ArgumentNullException("assetName");
//	if (disposed) throw new ObjectDisposedException(typeof(ContentManager).Name);
//	... cache lookup, then ReadAsset<T>, then cache store ...
//
// The cache is CNA's: `cna_content_manager_load_texture2d` goes through the
// native content manager, which caches by normalized key, so a second Load of
// the same name answers from it. That is why the projection keeps no cache of
// its own -- two caches would be two answers.
func ContentManagerLoad[T any](reference ContentManagerReference, assetName string) (T, error) {
	var zero T
	// The position widens to ContentManagerReference because ContentManager is
	// a substitutable base: a ResourceContentManager must be able to reach this
	// function, and in C# it simply does.
	manager := contentManagerOf(reference)
	if manager == nil || manager.disposed {
		// An IDENTITY SITE: ObjectDisposedException(this.ToString()).
		return zero, manager.disposedError()
	}
	if assetName == "" {
		return zero, fmt.Errorf("%w: assetName", errContentArgumentNull)
	}
	// The type is decided BEFORE the device is. A T outside the closed set
	// reaches no runtime at all, so a consumer who asks for an asset kind
	// CNA-Go does not project learns exactly that, rather than learning that no
	// graphics device service is registered -- which would be true, unhelpful,
	// and would change once they registered one.
	switch any(zero).(type) {
	case *graphics.Texture2D:
		resource, err := manager.native()
		if err != nil {
			return zero, err
		}
		asset, loadErr := servicebridge.LoadContentTexture2D(resource, assetName)
		if loadErr != nil {
			return zero, loadErr
		}
		loaded, ok := asset.(T)
		if !ok {
			return zero, errContentUnsupportedAsset
		}
		return loaded, nil
	case *graphics.Effect:
		resource, err := manager.native()
		if err != nil {
			return zero, err
		}
		asset, loadErr := servicebridge.LoadContentEffect(resource, assetName)
		if loadErr != nil {
			return zero, loadErr
		}
		loaded, ok := asset.(T)
		if !ok {
			return zero, errContentUnsupportedAsset
		}
		return loaded, nil
	case *graphics.SpriteFont:
		// The SpriteFont route reports the font AND its glyph atlas, and the
		// projected font holds both. Nothing here sees the texture: XNA's
		// SpriteFont keeps it in a private field with no public accessor, and
		// CNA-Go keeps it the same way.
		resource, err := manager.native()
		if err != nil {
			return zero, err
		}
		asset, loadErr := servicebridge.LoadContentSpriteFont(resource, assetName)
		if loadErr != nil {
			return zero, loadErr
		}
		loaded, ok := asset.(T)
		if !ok {
			return zero, errContentUnsupportedAsset
		}
		return loaded, nil
	default:
		return zero, fmt.Errorf("%w: %T", errContentUnsupportedAsset, zero)
	}
}

// ContentManagerReadAsset is ContentManager::ReadAsset<T>(String,
// Action<IDisposable>), the protected member Load<T> delegates the actual read
// to:
//
//	Stream stream = OpenStream(assetName);
//	using (ContentReader reader = ContentReader.Create(this, stream, assetName,
//	                                                   recordDisposableObject))
//	    return reader.ReadAsset<T>();
//
// # The recordDisposableObject action has no counterpart
//
// The reference passes it so a disposable the reader creates is registered with
// the manager and released by Unload. CNA's content manager owns that
// relationship itself -- its Unload disposes what it cached -- so there is
// nothing for a Go callback to record and the parameter is accepted and unused.
// `System.Action<IDisposable>` projects to `any` from the settled BCL table,
// and the projection does not invent a delegate type for a value it cannot use.
//
// # It does NOT bypass the cache
//
// The reference's ReadAsset reads without consulting the cache; CNA offers one
// load route and it is cached. So this differs from the reference in exactly
// that way, and it is recorded rather than papered over: a second ReadAsset of
// one name answers from CNA's cache where the reference would read again.
func ContentManagerReadAsset[T any](reference ContentManagerReference, assetName string, recordDisposableObject func(any)) (T, error) {
	manager := contentManagerOf(reference)
	if manager == nil || manager.disposed {
		// The THIRD identity site. ReadAsset carries its own disposal check in
		// the reference rather than inheriting Load's, so it is its own site.
		var zero T
		return zero, manager.disposedError()
	}
	// The Action<IDisposable> the reference passes is how a loader hands the
	// manager each disposable it creates, so Unload can release them. CNA owns
	// that bookkeeping -- its content manager tracks what it created -- so the
	// callback is accepted and not invoked, and calling it would announce
	// objects this projection did not create.
	_ = recordDisposableObject
	// The REFERENCE is forwarded, not the narrowed base: the CLR `this` does
	// not change across a call, so Load must see the object the caller handed
	// in. This is currently UNOBSERVABLE -- the guard above fires before the
	// delegation and Load's only identity site is the same disposal check --
	// and it is written this way regardless, because the alternative is right
	// only for as long as that stays true.
	return ContentManagerLoad[T](reference, assetName)
}

// contentManagerOf narrows a substitutable reference to the base half the
// package functions operate on. A nil interface and an interface holding a nil
// pointer are both the CLR's null, which is what the disposal guards compare
// against.
func contentManagerOf(reference ContentManagerReference) *ContentManager {
	if reference == nil {
		return nil
	}
	return reference.contentManager()
}
