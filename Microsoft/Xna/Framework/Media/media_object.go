package media

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// The machinery every media metadata type shares.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # All ten types are the same object with a different name
//
// Album, Artist, Genre, Song and Playlist each own a native handle, implement
// IDisposable and IEquatable<T>, and answer a Name and a ToString. The five
// collections each own a handle, implement IDisposable and IEnumerable<T>, and
// answer a Count and an indexer.
//
// So the disposal latch, the equality, the string conversion and the guard are
// written ONCE here and each type forwards to them. That is not a shortcut: the
// reference's own types are the same shape, generated from the same template
// against a different native object.
//
// # The two disposal routes are both bound, and both are needed
//
// CNA gives every one of these types a `_dispose` and a `_destroy`. They are
// not a duplication. `_dispose` is XNA's IDisposable: a consumer calls it, and
// the object STAYS QUERYABLE -- IsDisposed answers true afterwards, which the
// contract requires. `_destroy` releases the native memory, which the
// projection does when it is finished with the handle. Binding only one would
// make either IsDisposed unanswerable or the memory unreleasable.

// errMediaDisposed projects the ObjectDisposedException every member raises
// after Dispose.
var errMediaDisposed = errors.New("the media object is disposed")

// errMediaNoRuntime is the Go-only refusal a member answers with no runtime
// loaded. The reference reaches a media library that is always present.
var errMediaNoRuntime = errors.New("this member needs a loaded runtime")

// mediaObject is the state every one of the ten types holds.
type mediaObject struct {
	handle uint64
	// disposed is the MANAGED latch. CNA answers IsDisposed too, and the
	// projection keeps its own because a member must refuse BEFORE it reaches a
	// handle the consumer has already disposed -- asking CNA about a disposed
	// object is what the latch exists to avoid.
	disposed bool
}

// usable is the guard every member shares.
func (o *mediaObject) usable() error {
	if o == nil || o.disposed {
		return errMediaDisposed
	}
	if _, ok := interop.CurrentRuntime(); !ok {
		return errMediaNoRuntime
	}
	return nil
}

// runtimeFor answers the loaded runtime, after usable has admitted the object.
func runtimeFor() *interop.Runtime {
	runtime, _ := interop.CurrentRuntime()
	return runtime
}

// mediaDisposedError names the type, the way each reference type's
// ObjectDisposedException does.
func mediaDisposedError(typeName string) error {
	return fmt.Errorf("%w: %s", errMediaDisposed, typeName)
}

// errMediaArgumentNull projects System.ArgumentNullException, which carries a
// parameter name and no message of its own.
var errMediaArgumentNull = errors.New("argument is null")

func mediaArgumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errMediaArgumentNull, parameter)
}

// errMediaArgumentOutOfRange projects System.ArgumentOutOfRangeException, which
// every collection indexer raises for an index outside its bounds.
var errMediaArgumentOutOfRange = errors.New("argument is out of range")

// checkCollectionIndex is the bound every collection indexer applies.
//
// It is a named function rather than four lines repeated in five indexers, and
// that is what makes it REACHABLE: the indexers sit behind a runtime guard, so
// a test with no runtime cannot get to the comparison through them. The bound
// itself needs no runtime -- it is a comparison against a count -- and the
// count is what the runtime supplies.
//
// The refusal is ArgumentOutOfRangeException, which is what the reference's
// indexer raises, and it names both the index and the length so a caller can
// tell an out-of-range read from an empty collection.
func checkCollectionIndex(index, count int32) error {
	if index < 0 || index >= count {
		return fmt.Errorf("%w: index %d is outside a collection of %d",
			errMediaArgumentOutOfRange, index, count)
	}
	return nil
}

// The ten nil-receiver guards.
//
// mediaObject.usable() checks `o == nil`, and that check CANNOT fire through a
// nil outer pointer: reaching the embedded field is itself the dereference, so
// the panic happens before the guard runs. Each type therefore shadows it with
// one that tests its OWN pointer first.
//
// Ten near-identical methods are the price of embedding a shared guard behind a
// typed receiver, and it is paid here rather than by asking every member to
// remember the check.

func (s *Song) usable() error {
	if s == nil {
		return errMediaDisposed
	}
	return s.mediaObject.usable()
}

func (a *Album) usable() error {
	if a == nil {
		return errMediaDisposed
	}
	return a.mediaObject.usable()
}

func (a *Artist) usable() error {
	if a == nil {
		return errMediaDisposed
	}
	return a.mediaObject.usable()
}

func (g *Genre) usable() error {
	if g == nil {
		return errMediaDisposed
	}
	return g.mediaObject.usable()
}

func (p *Playlist) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (s *SongCollection) usable() error {
	if s == nil {
		return errMediaDisposed
	}
	return s.mediaObject.usable()
}

func (a *AlbumCollection) usable() error {
	if a == nil {
		return errMediaDisposed
	}
	return a.mediaObject.usable()
}

func (a *ArtistCollection) usable() error {
	if a == nil {
		return errMediaDisposed
	}
	return a.mediaObject.usable()
}

func (g *GenreCollection) usable() error {
	if g == nil {
		return errMediaDisposed
	}
	return g.mediaObject.usable()
}

func (p *PlaylistCollection) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (p *Picture) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (p *PictureAlbum) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (p *PictureCollection) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (p *PictureAlbumCollection) usable() error {
	if p == nil {
		return errMediaDisposed
	}
	return p.mediaObject.usable()
}

func (m *MediaLibrary) usable() error {
	if m == nil {
		return errMediaDisposed
	}
	return m.mediaObject.usable()
}
