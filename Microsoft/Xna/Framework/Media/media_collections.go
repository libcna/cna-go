package media

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

	"github.com/openeggbert/cna-go/internal/interop"
)

// The five collections a music library hands out.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Six members each, and the enumerator is the interesting one
//
// Each collection declares Finalize, Dispose, GetEnumerator, IsDisposed, Item
// and Count. The first four and the indexer are direct; GetEnumerator is what
// IEnumerable<T> requires, and Go has no enumerator interface -- so it projects
// to the settled Iterator adapter, the same one every other projected CLR
// enumerator in this profile uses.
//
// # An EMPTY collection is a valid answer, not a missing one
//
// A host with no music library answers zero for Count and refuses every index.
// That is the library being empty, which is a state XNA has too -- it is not
// evidence that the family cannot be projected, and the tests treat it as the
// measurement it is.

// SongCollection is Microsoft.Xna.Framework.Media.SongCollection.
type SongCollection struct{ mediaObject }

// newSongCollection wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newSongCollection(handle uint64) *SongCollection {
	return &SongCollection{mediaObject{handle: handle}}
}

// Dispose is SongCollection::Dispose().
func (s *SongCollection) Dispose() error {
	if s == nil || s.disposed {
		return nil
	}
	s.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaSongCollectionDispose(s.handle); err != nil {
		return err
	}
	return runtime.MediaSongCollectionDestroy(s.handle)
}

// Finalize is SongCollection::Finalize, the protected finalizer. Nothing calls it; see
// the entities' own note for why it is projected anyway.
func (s *SongCollection) Finalize() error { return s.Dispose() }

// IsDisposed is SongCollection::get_IsDisposed, read from the managed latch.
func (s *SongCollection) IsDisposed() bool { return s == nil || s.disposed }

// Count is SongCollection::get_Count.
func (s *SongCollection) Count() (int32, error) {
	if err := s.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaSongCollectionCount(s.handle)
}

// Item is SongCollection::get_Item(Int32), the indexer.
//
// An index outside the collection is a refusal rather than a nil, which is what
// the reference's ArgumentOutOfRangeException is. The bound is read from CNA
// first, so an empty collection refuses every index including zero.
func (s *SongCollection) Item(index int32) (*Song, error) {
	if err := s.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaSongCollectionCount(s.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaSongCollectionAt(s.handle, index)
	if err != nil {
		return nil, err
	}
	return newSong(handle), nil
}

// GetEnumerator is SongCollection::GetEnumerator(), the IEnumerable<Song> member.
func (s *SongCollection) GetEnumerator() (framework.Iterator[*Song], error) {
	if err := s.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaSongCollectionCount(s.handle)
	if err != nil {
		return nil, err
	}
	return &songCollectionIterator{owner: s, count: count, index: -1}, nil
}

// songCollectionIterator walks the collection by index, which is what the
// reference's own enumerator does: it holds the collection and a position and
// reads through the indexer.
//
// The COUNT is read once, when the enumerator is made, and not on every step.
// That is the reference's shape and it matters: a library that gained a song
// mid-walk would otherwise change the length of a walk already in progress.
type songCollectionIterator struct {
	owner *SongCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member. It reports the value, whether
// there was one, and any failure -- so an index that fails against CNA is a
// visible error rather than a walk that quietly stops early.
func (i *songCollectionIterator) Next() (*Song, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

// AlbumCollection is Microsoft.Xna.Framework.Media.AlbumCollection.
type AlbumCollection struct{ mediaObject }

// newAlbumCollection wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newAlbumCollection(handle uint64) *AlbumCollection {
	return &AlbumCollection{mediaObject{handle: handle}}
}

// Dispose is AlbumCollection::Dispose().
func (a *AlbumCollection) Dispose() error {
	if a == nil || a.disposed {
		return nil
	}
	a.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaAlbumCollectionDispose(a.handle); err != nil {
		return err
	}
	return runtime.MediaAlbumCollectionDestroy(a.handle)
}

// Finalize is AlbumCollection::Finalize, the protected finalizer. Nothing calls it; see
// the entities' own note for why it is projected anyway.
func (a *AlbumCollection) Finalize() error { return a.Dispose() }

// IsDisposed is AlbumCollection::get_IsDisposed, read from the managed latch.
func (a *AlbumCollection) IsDisposed() bool { return a == nil || a.disposed }

// Count is AlbumCollection::get_Count.
func (a *AlbumCollection) Count() (int32, error) {
	if err := a.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaAlbumCollectionCount(a.handle)
}

// Item is AlbumCollection::get_Item(Int32), the indexer.
//
// An index outside the collection is a refusal rather than a nil, which is what
// the reference's ArgumentOutOfRangeException is. The bound is read from CNA
// first, so an empty collection refuses every index including zero.
func (a *AlbumCollection) Item(index int32) (*Album, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaAlbumCollectionCount(a.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaAlbumCollectionAt(a.handle, index)
	if err != nil {
		return nil, err
	}
	return newAlbum(handle), nil
}

// GetEnumerator is AlbumCollection::GetEnumerator(), the IEnumerable<Album> member.
func (a *AlbumCollection) GetEnumerator() (framework.Iterator[*Album], error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaAlbumCollectionCount(a.handle)
	if err != nil {
		return nil, err
	}
	return &albumCollectionIterator{owner: a, count: count, index: -1}, nil
}

// albumCollectionIterator walks the collection by index, which is what the
// reference's own enumerator does: it holds the collection and a position and
// reads through the indexer.
//
// The COUNT is read once, when the enumerator is made, and not on every step.
// That is the reference's shape and it matters: a library that gained a song
// mid-walk would otherwise change the length of a walk already in progress.
type albumCollectionIterator struct {
	owner *AlbumCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member. It reports the value, whether
// there was one, and any failure -- so an index that fails against CNA is a
// visible error rather than a walk that quietly stops early.
func (i *albumCollectionIterator) Next() (*Album, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

// ArtistCollection is Microsoft.Xna.Framework.Media.ArtistCollection.
type ArtistCollection struct{ mediaObject }

// newArtistCollection wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newArtistCollection(handle uint64) *ArtistCollection {
	return &ArtistCollection{mediaObject{handle: handle}}
}

// Dispose is ArtistCollection::Dispose().
func (a *ArtistCollection) Dispose() error {
	if a == nil || a.disposed {
		return nil
	}
	a.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaArtistCollectionDispose(a.handle); err != nil {
		return err
	}
	return runtime.MediaArtistCollectionDestroy(a.handle)
}

// Finalize is ArtistCollection::Finalize, the protected finalizer. Nothing calls it; see
// the entities' own note for why it is projected anyway.
func (a *ArtistCollection) Finalize() error { return a.Dispose() }

// IsDisposed is ArtistCollection::get_IsDisposed, read from the managed latch.
func (a *ArtistCollection) IsDisposed() bool { return a == nil || a.disposed }

// Count is ArtistCollection::get_Count.
func (a *ArtistCollection) Count() (int32, error) {
	if err := a.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaArtistCollectionCount(a.handle)
}

// Item is ArtistCollection::get_Item(Int32), the indexer.
//
// An index outside the collection is a refusal rather than a nil, which is what
// the reference's ArgumentOutOfRangeException is. The bound is read from CNA
// first, so an empty collection refuses every index including zero.
func (a *ArtistCollection) Item(index int32) (*Artist, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaArtistCollectionCount(a.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaArtistCollectionAt(a.handle, index)
	if err != nil {
		return nil, err
	}
	return newArtist(handle), nil
}

// GetEnumerator is ArtistCollection::GetEnumerator(), the IEnumerable<Artist> member.
func (a *ArtistCollection) GetEnumerator() (framework.Iterator[*Artist], error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaArtistCollectionCount(a.handle)
	if err != nil {
		return nil, err
	}
	return &artistCollectionIterator{owner: a, count: count, index: -1}, nil
}

// artistCollectionIterator walks the collection by index, which is what the
// reference's own enumerator does: it holds the collection and a position and
// reads through the indexer.
//
// The COUNT is read once, when the enumerator is made, and not on every step.
// That is the reference's shape and it matters: a library that gained a song
// mid-walk would otherwise change the length of a walk already in progress.
type artistCollectionIterator struct {
	owner *ArtistCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member. It reports the value, whether
// there was one, and any failure -- so an index that fails against CNA is a
// visible error rather than a walk that quietly stops early.
func (i *artistCollectionIterator) Next() (*Artist, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

// GenreCollection is Microsoft.Xna.Framework.Media.GenreCollection.
type GenreCollection struct{ mediaObject }

// newGenreCollection wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newGenreCollection(handle uint64) *GenreCollection {
	return &GenreCollection{mediaObject{handle: handle}}
}

// Dispose is GenreCollection::Dispose().
func (g *GenreCollection) Dispose() error {
	if g == nil || g.disposed {
		return nil
	}
	g.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaGenreCollectionDispose(g.handle); err != nil {
		return err
	}
	return runtime.MediaGenreCollectionDestroy(g.handle)
}

// Finalize is GenreCollection::Finalize, the protected finalizer. Nothing calls it; see
// the entities' own note for why it is projected anyway.
func (g *GenreCollection) Finalize() error { return g.Dispose() }

// IsDisposed is GenreCollection::get_IsDisposed, read from the managed latch.
func (g *GenreCollection) IsDisposed() bool { return g == nil || g.disposed }

// Count is GenreCollection::get_Count.
func (g *GenreCollection) Count() (int32, error) {
	if err := g.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaGenreCollectionCount(g.handle)
}

// Item is GenreCollection::get_Item(Int32), the indexer.
//
// An index outside the collection is a refusal rather than a nil, which is what
// the reference's ArgumentOutOfRangeException is. The bound is read from CNA
// first, so an empty collection refuses every index including zero.
func (g *GenreCollection) Item(index int32) (*Genre, error) {
	if err := g.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaGenreCollectionCount(g.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaGenreCollectionAt(g.handle, index)
	if err != nil {
		return nil, err
	}
	return newGenre(handle), nil
}

// GetEnumerator is GenreCollection::GetEnumerator(), the IEnumerable<Genre> member.
func (g *GenreCollection) GetEnumerator() (framework.Iterator[*Genre], error) {
	if err := g.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaGenreCollectionCount(g.handle)
	if err != nil {
		return nil, err
	}
	return &genreCollectionIterator{owner: g, count: count, index: -1}, nil
}

// genreCollectionIterator walks the collection by index, which is what the
// reference's own enumerator does: it holds the collection and a position and
// reads through the indexer.
//
// The COUNT is read once, when the enumerator is made, and not on every step.
// That is the reference's shape and it matters: a library that gained a song
// mid-walk would otherwise change the length of a walk already in progress.
type genreCollectionIterator struct {
	owner *GenreCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member. It reports the value, whether
// there was one, and any failure -- so an index that fails against CNA is a
// visible error rather than a walk that quietly stops early.
func (i *genreCollectionIterator) Next() (*Genre, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

// PlaylistCollection is Microsoft.Xna.Framework.Media.PlaylistCollection.
type PlaylistCollection struct{ mediaObject }

// newPlaylistCollection wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newPlaylistCollection(handle uint64) *PlaylistCollection {
	return &PlaylistCollection{mediaObject{handle: handle}}
}

// Dispose is PlaylistCollection::Dispose().
func (p *PlaylistCollection) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPlaylistCollectionDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPlaylistCollectionDestroy(p.handle)
}

// Finalize is PlaylistCollection::Finalize, the protected finalizer. Nothing calls it; see
// the entities' own note for why it is projected anyway.
func (p *PlaylistCollection) Finalize() error { return p.Dispose() }

// IsDisposed is PlaylistCollection::get_IsDisposed, read from the managed latch.
func (p *PlaylistCollection) IsDisposed() bool { return p == nil || p.disposed }

// Count is PlaylistCollection::get_Count.
func (p *PlaylistCollection) Count() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPlaylistCollectionCount(p.handle)
}

// Item is PlaylistCollection::get_Item(Int32), the indexer.
//
// An index outside the collection is a refusal rather than a nil, which is what
// the reference's ArgumentOutOfRangeException is. The bound is read from CNA
// first, so an empty collection refuses every index including zero.
func (p *PlaylistCollection) Item(index int32) (*Playlist, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPlaylistCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPlaylistCollectionAt(p.handle, index)
	if err != nil {
		return nil, err
	}
	return newPlaylist(handle), nil
}

// GetEnumerator is PlaylistCollection::GetEnumerator(), the IEnumerable<Playlist> member.
func (p *PlaylistCollection) GetEnumerator() (framework.Iterator[*Playlist], error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPlaylistCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	return &playlistCollectionIterator{owner: p, count: count, index: -1}, nil
}

// playlistCollectionIterator walks the collection by index, which is what the
// reference's own enumerator does: it holds the collection and a position and
// reads through the indexer.
//
// The COUNT is read once, when the enumerator is made, and not on every step.
// That is the reference's shape and it matters: a library that gained a song
// mid-walk would otherwise change the length of a walk already in progress.
type playlistCollectionIterator struct {
	owner *PlaylistCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member. It reports the value, whether
// there was one, and any failure -- so an index that fails against CNA is a
// visible error rather than a walk that quietly stops early.
func (i *playlistCollectionIterator) Next() (*Playlist, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}
