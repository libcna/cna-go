package media

import (
	"github.com/openeggbert/cna-go/internal/interop"
)

// The five entities a music library describes.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # None of them has a public constructor, and one has a static factory
//
// The pinned contract declares no constructor on any of the five. They reach a
// consumer from a MediaLibrary or from another entity -- an Album's Artist, an
// Artist's Songs -- which is why every factory below is unexported.
//
// Song::FromUri is the one exception and it IS declared, as a public static.

// Song is Microsoft.Xna.Framework.Media.Song.
type Song struct{ mediaObject }

// newSong wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newSong(handle uint64) *Song { return &Song{mediaObject{handle: handle}} }

// Dispose is Song::Dispose(), which releases the native object. The managed
// latch is set FIRST, so a member reached afterwards refuses without touching
// a handle CNA has already torn down.
func (s *Song) Dispose() error {
	if s == nil || s.disposed {
		return nil
	}
	s.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaSongDispose(s.handle); err != nil {
		return err
	}
	return runtime.MediaSongDestroy(s.handle)
}

// Finalize is Song::Finalize, the protected finalizer, which the reference
// implements as the same teardown Dispose performs.
//
// Nothing calls it: Go has no CLR finalization and this projection registers no
// runtime finalizer. It is projected because the pinned contract declares it --
// the same judgement SoundEffect's got in Foundation 87.
func (s *Song) Finalize() error { return s.Dispose() }

// IsDisposed is Song::get_IsDisposed. It reads the MANAGED latch and does not
// ask CNA, because a disposed object is exactly the one whose handle must not
// be touched.
func (s *Song) IsDisposed() bool {
	return s == nil || s.disposed
}

// Name is Song::get_Name.
func (s *Song) Name() (string, error) {
	if err := s.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaSongName(s.handle)
}

// ToString is Song::ToString(), which the reference implements as the name.
func (s *Song) ToString() (string, error) {
	if err := s.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaSongTypeName(s.handle)
}

// GetHashCode is Song::GetHashCode.
func (s *Song) GetHashCode() (int32, error) {
	if err := s.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaSongHashCode(s.handle)
}

// EqualsByObject is Song::Equals(Object), which answers false for anything
// that is not a Song -- including nil.
func (s *Song) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Song)
	if !ok {
		return false, nil
	}
	return s.EqualsBySong(other)
}

// EqualsBySong is Song::Equals(Song), the IEquatable<T> member.
func (s *Song) EqualsBySong(other *Song) (bool, error) {
	if err := s.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaSongEquals(s.handle, other.handle)
}

// SongOperatorEqualityBySongAndSong is Song::op_Equality. It is a package function because Go has
// no operator overloading, which is the settled spelling for every projected
// CLR operator.
//
// Two nils are EQUAL and one nil is not, which is what the reference's own
// reference comparison answers before it reaches the native call.
func SongOperatorEqualityBySongAndSong(left, right *Song) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsBySong(right)
}

// SongOperatorInequalityBySongAndSong is Song::op_Inequality, the negation of op_Equality.
func SongOperatorInequalityBySongAndSong(left, right *Song) (bool, error) {
	equal, err := SongOperatorEqualityBySongAndSong(left, right)
	return !equal, err
}

// Album is Microsoft.Xna.Framework.Media.Album.
type Album struct{ mediaObject }

// newAlbum wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newAlbum(handle uint64) *Album { return &Album{mediaObject{handle: handle}} }

// Dispose is Album::Dispose(), which releases the native object. The managed
// latch is set FIRST, so a member reached afterwards refuses without touching
// a handle CNA has already torn down.
func (a *Album) Dispose() error {
	if a == nil || a.disposed {
		return nil
	}
	a.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaAlbumDispose(a.handle); err != nil {
		return err
	}
	return runtime.MediaAlbumDestroy(a.handle)
}

// Finalize is Album::Finalize, the protected finalizer, which the reference
// implements as the same teardown Dispose performs.
//
// Nothing calls it: Go has no CLR finalization and this projection registers no
// runtime finalizer. It is projected because the pinned contract declares it --
// the same judgement SoundEffect's got in Foundation 87.
func (a *Album) Finalize() error { return a.Dispose() }

// IsDisposed is Album::get_IsDisposed. It reads the MANAGED latch and does not
// ask CNA, because a disposed object is exactly the one whose handle must not
// be touched.
func (a *Album) IsDisposed() bool {
	return a == nil || a.disposed
}

// Name is Album::get_Name.
func (a *Album) Name() (string, error) {
	if err := a.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaAlbumName(a.handle)
}

// ToString is Album::ToString(), which the reference implements as the name.
func (a *Album) ToString() (string, error) {
	if err := a.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaAlbumTypeName(a.handle)
}

// GetHashCode is Album::GetHashCode.
func (a *Album) GetHashCode() (int32, error) {
	if err := a.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaAlbumHashCode(a.handle)
}

// EqualsByObject is Album::Equals(Object), which answers false for anything
// that is not a Album -- including nil.
func (a *Album) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Album)
	if !ok {
		return false, nil
	}
	return a.EqualsByAlbum(other)
}

// EqualsByAlbum is Album::Equals(Album), the IEquatable<T> member.
func (a *Album) EqualsByAlbum(other *Album) (bool, error) {
	if err := a.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaAlbumEquals(a.handle, other.handle)
}

// AlbumOperatorEqualityByAlbumAndAlbum is Album::op_Equality. It is a package function because Go has
// no operator overloading, which is the settled spelling for every projected
// CLR operator.
//
// Two nils are EQUAL and one nil is not, which is what the reference's own
// reference comparison answers before it reaches the native call.
func AlbumOperatorEqualityByAlbumAndAlbum(left, right *Album) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByAlbum(right)
}

// AlbumOperatorInequalityByAlbumAndAlbum is Album::op_Inequality, the negation of op_Equality.
func AlbumOperatorInequalityByAlbumAndAlbum(left, right *Album) (bool, error) {
	equal, err := AlbumOperatorEqualityByAlbumAndAlbum(left, right)
	return !equal, err
}

// Artist is Microsoft.Xna.Framework.Media.Artist.
type Artist struct{ mediaObject }

// newArtist wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newArtist(handle uint64) *Artist { return &Artist{mediaObject{handle: handle}} }

// Dispose is Artist::Dispose(), which releases the native object. The managed
// latch is set FIRST, so a member reached afterwards refuses without touching
// a handle CNA has already torn down.
func (a *Artist) Dispose() error {
	if a == nil || a.disposed {
		return nil
	}
	a.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaArtistDispose(a.handle); err != nil {
		return err
	}
	return runtime.MediaArtistDestroy(a.handle)
}

// Finalize is Artist::Finalize, the protected finalizer, which the reference
// implements as the same teardown Dispose performs.
//
// Nothing calls it: Go has no CLR finalization and this projection registers no
// runtime finalizer. It is projected because the pinned contract declares it --
// the same judgement SoundEffect's got in Foundation 87.
func (a *Artist) Finalize() error { return a.Dispose() }

// IsDisposed is Artist::get_IsDisposed. It reads the MANAGED latch and does not
// ask CNA, because a disposed object is exactly the one whose handle must not
// be touched.
func (a *Artist) IsDisposed() bool {
	return a == nil || a.disposed
}

// Name is Artist::get_Name.
func (a *Artist) Name() (string, error) {
	if err := a.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaArtistName(a.handle)
}

// ToString is Artist::ToString(), which the reference implements as the name.
func (a *Artist) ToString() (string, error) {
	if err := a.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaArtistTypeName(a.handle)
}

// GetHashCode is Artist::GetHashCode.
func (a *Artist) GetHashCode() (int32, error) {
	if err := a.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaArtistHashCode(a.handle)
}

// EqualsByObject is Artist::Equals(Object), which answers false for anything
// that is not a Artist -- including nil.
func (a *Artist) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Artist)
	if !ok {
		return false, nil
	}
	return a.EqualsByArtist(other)
}

// EqualsByArtist is Artist::Equals(Artist), the IEquatable<T> member.
func (a *Artist) EqualsByArtist(other *Artist) (bool, error) {
	if err := a.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaArtistEquals(a.handle, other.handle)
}

// ArtistOperatorEqualityByArtistAndArtist is Artist::op_Equality. It is a package function because Go has
// no operator overloading, which is the settled spelling for every projected
// CLR operator.
//
// Two nils are EQUAL and one nil is not, which is what the reference's own
// reference comparison answers before it reaches the native call.
func ArtistOperatorEqualityByArtistAndArtist(left, right *Artist) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByArtist(right)
}

// ArtistOperatorInequalityByArtistAndArtist is Artist::op_Inequality, the negation of op_Equality.
func ArtistOperatorInequalityByArtistAndArtist(left, right *Artist) (bool, error) {
	equal, err := ArtistOperatorEqualityByArtistAndArtist(left, right)
	return !equal, err
}

// Genre is Microsoft.Xna.Framework.Media.Genre.
type Genre struct{ mediaObject }

// newGenre wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newGenre(handle uint64) *Genre { return &Genre{mediaObject{handle: handle}} }

// Dispose is Genre::Dispose(), which releases the native object. The managed
// latch is set FIRST, so a member reached afterwards refuses without touching
// a handle CNA has already torn down.
func (g *Genre) Dispose() error {
	if g == nil || g.disposed {
		return nil
	}
	g.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaGenreDispose(g.handle); err != nil {
		return err
	}
	return runtime.MediaGenreDestroy(g.handle)
}

// Finalize is Genre::Finalize, the protected finalizer, which the reference
// implements as the same teardown Dispose performs.
//
// Nothing calls it: Go has no CLR finalization and this projection registers no
// runtime finalizer. It is projected because the pinned contract declares it --
// the same judgement SoundEffect's got in Foundation 87.
func (g *Genre) Finalize() error { return g.Dispose() }

// IsDisposed is Genre::get_IsDisposed. It reads the MANAGED latch and does not
// ask CNA, because a disposed object is exactly the one whose handle must not
// be touched.
func (g *Genre) IsDisposed() bool {
	return g == nil || g.disposed
}

// Name is Genre::get_Name.
func (g *Genre) Name() (string, error) {
	if err := g.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaGenreName(g.handle)
}

// ToString is Genre::ToString(), which the reference implements as the name.
func (g *Genre) ToString() (string, error) {
	if err := g.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaGenreTypeName(g.handle)
}

// GetHashCode is Genre::GetHashCode.
func (g *Genre) GetHashCode() (int32, error) {
	if err := g.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaGenreHashCode(g.handle)
}

// EqualsByObject is Genre::Equals(Object), which answers false for anything
// that is not a Genre -- including nil.
func (g *Genre) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Genre)
	if !ok {
		return false, nil
	}
	return g.EqualsByGenre(other)
}

// EqualsByGenre is Genre::Equals(Genre), the IEquatable<T> member.
func (g *Genre) EqualsByGenre(other *Genre) (bool, error) {
	if err := g.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaGenreEquals(g.handle, other.handle)
}

// GenreOperatorEqualityByGenreAndGenre is Genre::op_Equality. It is a package function because Go has
// no operator overloading, which is the settled spelling for every projected
// CLR operator.
//
// Two nils are EQUAL and one nil is not, which is what the reference's own
// reference comparison answers before it reaches the native call.
func GenreOperatorEqualityByGenreAndGenre(left, right *Genre) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByGenre(right)
}

// GenreOperatorInequalityByGenreAndGenre is Genre::op_Inequality, the negation of op_Equality.
func GenreOperatorInequalityByGenreAndGenre(left, right *Genre) (bool, error) {
	equal, err := GenreOperatorEqualityByGenreAndGenre(left, right)
	return !equal, err
}

// Playlist is Microsoft.Xna.Framework.Media.Playlist.
type Playlist struct{ mediaObject }

// newPlaylist wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newPlaylist(handle uint64) *Playlist { return &Playlist{mediaObject{handle: handle}} }

// Dispose is Playlist::Dispose(), which releases the native object. The managed
// latch is set FIRST, so a member reached afterwards refuses without touching
// a handle CNA has already torn down.
func (p *Playlist) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPlaylistDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPlaylistDestroy(p.handle)
}

// Finalize is Playlist::Finalize, the protected finalizer, which the reference
// implements as the same teardown Dispose performs.
//
// Nothing calls it: Go has no CLR finalization and this projection registers no
// runtime finalizer. It is projected because the pinned contract declares it --
// the same judgement SoundEffect's got in Foundation 87.
func (p *Playlist) Finalize() error { return p.Dispose() }

// IsDisposed is Playlist::get_IsDisposed. It reads the MANAGED latch and does not
// ask CNA, because a disposed object is exactly the one whose handle must not
// be touched.
func (p *Playlist) IsDisposed() bool {
	return p == nil || p.disposed
}

// Name is Playlist::get_Name.
func (p *Playlist) Name() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPlaylistName(p.handle)
}

// ToString is Playlist::ToString(), which the reference implements as the name.
func (p *Playlist) ToString() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPlaylistTypeName(p.handle)
}

// GetHashCode is Playlist::GetHashCode.
func (p *Playlist) GetHashCode() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPlaylistHashCode(p.handle)
}

// EqualsByObject is Playlist::Equals(Object), which answers false for anything
// that is not a Playlist -- including nil.
func (p *Playlist) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Playlist)
	if !ok {
		return false, nil
	}
	return p.EqualsByPlaylist(other)
}

// EqualsByPlaylist is Playlist::Equals(Playlist), the IEquatable<T> member.
func (p *Playlist) EqualsByPlaylist(other *Playlist) (bool, error) {
	if err := p.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaPlaylistEquals(p.handle, other.handle)
}

// PlaylistOperatorEqualityByPlaylistAndPlaylist is Playlist::op_Equality. It is a package function because Go has
// no operator overloading, which is the settled spelling for every projected
// CLR operator.
//
// Two nils are EQUAL and one nil is not, which is what the reference's own
// reference comparison answers before it reaches the native call.
func PlaylistOperatorEqualityByPlaylistAndPlaylist(left, right *Playlist) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByPlaylist(right)
}

// PlaylistOperatorInequalityByPlaylistAndPlaylist is Playlist::op_Inequality, the negation of op_Equality.
func PlaylistOperatorInequalityByPlaylistAndPlaylist(left, right *Playlist) (bool, error) {
	equal, err := PlaylistOperatorEqualityByPlaylistAndPlaylist(left, right)
	return !equal, err
}
