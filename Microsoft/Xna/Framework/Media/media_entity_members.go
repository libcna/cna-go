package media

import (
	"bytes"
	"io"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// The members that differ between the five entities: what each one describes.
//
// # An optional reference is a NIL, not an error
//
// Three of CNA's getters -- a song's artist, album and genre, and an album's
// artist and genre -- report AVAILABILITY beside the handle. A song whose file
// carries no album tag is not a failure; it is a song with no album, and the
// reference answers null there. So an unavailable reference projects to a nil
// pointer and no error, and only a real native failure is an error.

// Artist is Song::get_Artist.
func (s *Song) Artist() (*Artist, error) {
	if err := s.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaSongArtist(s.handle)
	if err != nil || !available {
		return nil, err
	}
	return newArtist(handle), nil
}

// Album is Song::get_Album.
func (s *Song) Album() (*Album, error) {
	if err := s.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaSongAlbum(s.handle)
	if err != nil || !available {
		return nil, err
	}
	return newAlbum(handle), nil
}

// Genre is Song::get_Genre.
func (s *Song) Genre() (*Genre, error) {
	if err := s.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaSongGenre(s.handle)
	if err != nil || !available {
		return nil, err
	}
	return newGenre(handle), nil
}

// Duration is Song::get_Duration. CNA reports TICKS, which is the CLR's own
// unit, so the projection converts nothing.
func (s *Song) Duration() (framework.TimeSpan, error) {
	if err := s.usable(); err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtimeFor().MediaSongDurationTicks(s.handle)
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}

// IsRated is Song::get_IsRated.
func (s *Song) IsRated() (bool, error) {
	if err := s.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaSongIsRated(s.handle)
}

// Rating is Song::get_Rating.
func (s *Song) Rating() (int32, error) {
	if err := s.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaSongRating(s.handle)
}

// PlayCount is Song::get_PlayCount.
func (s *Song) PlayCount() (int32, error) {
	if err := s.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaSongPlayCount(s.handle)
}

// TrackNumber is Song::get_TrackNumber.
func (s *Song) TrackNumber() (int32, error) {
	if err := s.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaSongTrackNumber(s.handle)
}

// IsProtected is Song::get_IsProtected.
func (s *Song) IsProtected() (bool, error) {
	if err := s.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaSongIsProtected(s.handle)
}

// SongFromUri is Song::FromUri(String, Uri), the one public static in the
// family.
//
// System.Uri projects to a Go STRING, and that is measured rather than chosen:
// the reference's whole body is `new Song(name, uri)`, whose constructor
// compares the Uri against null and then hands it straight to the P/Invoke,
// which marshals it as a string. Nothing reads a scheme, a host or a path.
//
// The null comparison becomes an empty-string check, which is the same refusal
// for the only value a Go caller can pass in its place.
func SongFromUri(name, uri string) (*Song, error) {
	// The ARGUMENT check comes first, because the reference's does: FromUri's
	// whole body is `new Song(name, uri)` and that constructor compares the Uri
	// against null BEFORE it reaches the P/Invoke. A projection that checked
	// for a runtime first would answer the wrong refusal on a machine with no
	// game running.
	if uri == "" {
		return nil, mediaArgumentNullError("uri")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errMediaNoRuntime
	}
	handle, err := runtime.MediaSongFromURI(name, uri)
	if err != nil {
		return nil, err
	}
	return newSong(handle), nil
}

// Artist is Album::get_Artist.
func (a *Album) Artist() (*Artist, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaAlbumArtist(a.handle)
	if err != nil || !available {
		return nil, err
	}
	return newArtist(handle), nil
}

// Genre is Album::get_Genre.
func (a *Album) Genre() (*Genre, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaAlbumGenre(a.handle)
	if err != nil || !available {
		return nil, err
	}
	return newGenre(handle), nil
}

// Songs is Album::get_Songs.
func (a *Album) Songs() (*SongCollection, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaAlbumSongs(a.handle)
	if err != nil {
		return nil, err
	}
	return newSongCollection(handle), nil
}

// Duration is Album::get_Duration.
func (a *Album) Duration() (framework.TimeSpan, error) {
	if err := a.usable(); err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtimeFor().MediaAlbumDurationTicks(a.handle)
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}

// HasArt is Album::get_HasArt.
func (a *Album) HasArt() (bool, error) {
	if err := a.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaAlbumHasArt(a.handle)
}

// GetAlbumArt is Album::GetAlbumArt(), which answers a Stream over the art.
//
// The bytes are copied into Go memory and handed back as a reader. That is what
// System.IO.Stream maps to in this profile, and it is also the honest lifetime:
// the native buffer belongs to CNA and a reader over it would outlive the call.
//
// An album with NO art answers an empty reader rather than an error, which is
// what HasArt is for -- the reference returns a zero-length stream there too.
func (a *Album) GetAlbumArt() (io.Reader, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	data, err := runtimeFor().MediaAlbumArt(a.handle)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// GetThumbnail is Album::GetThumbnail(), the same shape over the thumbnail.
func (a *Album) GetThumbnail() (io.Reader, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	data, err := runtimeFor().MediaAlbumThumbnail(a.handle)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// Songs is Artist::get_Songs.
func (a *Artist) Songs() (*SongCollection, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaArtistSongs(a.handle)
	if err != nil {
		return nil, err
	}
	return newSongCollection(handle), nil
}

// Albums is Artist::get_Albums.
func (a *Artist) Albums() (*AlbumCollection, error) {
	if err := a.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaArtistAlbums(a.handle)
	if err != nil {
		return nil, err
	}
	return newAlbumCollection(handle), nil
}

// Songs is Genre::get_Songs.
func (g *Genre) Songs() (*SongCollection, error) {
	if err := g.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaGenreSongs(g.handle)
	if err != nil {
		return nil, err
	}
	return newSongCollection(handle), nil
}

// Albums is Genre::get_Albums.
func (g *Genre) Albums() (*AlbumCollection, error) {
	if err := g.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaGenreAlbums(g.handle)
	if err != nil {
		return nil, err
	}
	return newAlbumCollection(handle), nil
}

// Songs is Playlist::get_Songs.
func (p *Playlist) Songs() (*SongCollection, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPlaylistSongs(p.handle)
	if err != nil {
		return nil, err
	}
	return newSongCollection(handle), nil
}

// Duration is Playlist::get_Duration.
func (p *Playlist) Duration() (framework.TimeSpan, error) {
	if err := p.usable(); err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtimeFor().MediaPlaylistDurationTicks(p.handle)
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}
