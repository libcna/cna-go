package media

import (
	"bytes"
	"errors"
	"io"
	"time"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// MediaLibrary is Microsoft.Xna.Framework.Media.MediaLibrary, the entry point
// to everything else in this namespace.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It is what makes the rest reachable
//
// Foundation 95 projected ten metadata types with exactly one public entry
// point between them -- Song.FromUri -- because none of them declares a
// constructor. This one declares two, and its seven collection properties are
// how a consumer reaches songs, artists, albums, playlists, genres and
// pictures at all.
type MediaLibrary struct {
	mediaObject
	// source is the MediaSource this library was opened from, held because the
	// contract exposes it and CNA answers its type and name off the LIBRARY
	// rather than off a separate object.
	source *MediaSource
}

// NewMediaLibraryByNone is MediaLibrary::.ctor(), which opens the default library.
func NewMediaLibraryByNone() (*MediaLibrary, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errMediaNoRuntime
	}
	handle, err := runtime.MediaLibraryCreate()
	if err != nil {
		return nil, err
	}
	return &MediaLibrary{mediaObject: mediaObject{handle: handle}}, nil
}

// NewMediaLibraryByMediaSource is MediaLibrary::.ctor(MediaSource).
//
// CNA opens a library by the source's INDEX in the available list, because it
// has no media-source handle at all. So a source that came from
// MediaSourceGetAvailableMediaSources carries that index and works here, and
// one read back from an already-open library's MediaSource property does not.
//
// That is a real difference from the reference, where any MediaSource can be
// passed, and it is refused with a message that says which kind is which
// rather than silently opening the default library.
func NewMediaLibraryByMediaSource(source *MediaSource) (*MediaLibrary, error) {
	if source == nil {
		return nil, mediaArgumentNullError("mediaSource")
	}
	if !source.enumerated {
		return nil, errors.New(
			"this MediaSource was read back from an open library and carries no index; " +
				"CNA opens a library by the index a source has in MediaSourceGetAvailableMediaSources")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errMediaNoRuntime
	}
	handle, err := runtime.MediaLibraryCreateFromSource(source.index)
	if err != nil {
		return nil, err
	}
	return &MediaLibrary{mediaObject: mediaObject{handle: handle}, source: source}, nil
}

// Dispose is MediaLibrary::Dispose().
func (m *MediaLibrary) Dispose() error {
	if m == nil || m.disposed {
		return nil
	}
	m.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaLibraryDispose(m.handle); err != nil {
		return err
	}
	return runtime.MediaLibraryDestroy(m.handle)
}

// Finalize is MediaLibrary::Finalize, the protected finalizer.
func (m *MediaLibrary) Finalize() error { return m.Dispose() }

// IsDisposed is MediaLibrary::get_IsDisposed, read from the managed latch.
func (m *MediaLibrary) IsDisposed() bool { return m == nil || m.disposed }

// MediaSource is MediaLibrary::get_MediaSource.
//
// CNA answers the source's type and name off the LIBRARY -- there is no source
// handle -- so the value is built from those two reads. It carries no index and
// therefore cannot be handed back to the constructor; see that member.
func (m *MediaLibrary) MediaSource() (*MediaSource, error) {
	if err := m.usable(); err != nil {
		return nil, err
	}
	if m.source != nil {
		return m.source, nil
	}
	kind, err := runtimeFor().MediaLibraryMediaSourceType(m.handle)
	if err != nil {
		return nil, err
	}
	name, err := runtimeFor().MediaLibraryMediaSourceName(m.handle)
	if err != nil {
		return nil, err
	}
	return &MediaSource{sourceType: MediaSourceType(kind), name: name}, nil
}

// Songs is MediaLibrary::get_Songs.
func (m *MediaLibrary) Songs() (*SongCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibrarySongs)
	if err != nil {
		return nil, err
	}
	return newSongCollection(handle), nil
}

// Artists is MediaLibrary::get_Artists.
func (m *MediaLibrary) Artists() (*ArtistCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibraryArtists)
	if err != nil {
		return nil, err
	}
	return newArtistCollection(handle), nil
}

// Albums is MediaLibrary::get_Albums.
func (m *MediaLibrary) Albums() (*AlbumCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibraryAlbums)
	if err != nil {
		return nil, err
	}
	return newAlbumCollection(handle), nil
}

// Genres is MediaLibrary::get_Genres.
func (m *MediaLibrary) Genres() (*GenreCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibraryGenres)
	if err != nil {
		return nil, err
	}
	return newGenreCollection(handle), nil
}

// Playlists is MediaLibrary::get_Playlists.
func (m *MediaLibrary) Playlists() (*PlaylistCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibraryPlaylists)
	if err != nil {
		return nil, err
	}
	return newPlaylistCollection(handle), nil
}

// Pictures is MediaLibrary::get_Pictures.
func (m *MediaLibrary) Pictures() (*PictureCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibraryPictures)
	if err != nil {
		return nil, err
	}
	return newPictureCollection(handle), nil
}

// SavedPictures is MediaLibrary::get_SavedPictures, the pictures this
// application saved rather than every picture the device holds.
func (m *MediaLibrary) SavedPictures() (*PictureCollection, error) {
	handle, err := m.collection(runtimeFor().MediaLibrarySavedPictures)
	if err != nil {
		return nil, err
	}
	return newPictureCollection(handle), nil
}

// collection is the guard the seven collection properties share. It is a
// helper because the guard is identical in all seven and the only thing that
// differs is which route answers.
func (m *MediaLibrary) collection(read func(uint64) (uint64, error)) (uint64, error) {
	if err := m.usable(); err != nil {
		return 0, err
	}
	return read(m.handle)
}

// RootPictureAlbum is MediaLibrary::get_RootPictureAlbum.
//
// A device with no picture library has no root album, which CNA reports as an
// availability flag and the reference answers null for.
func (m *MediaLibrary) RootPictureAlbum() (*PictureAlbum, error) {
	if err := m.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaLibraryRootPictureAlbum(m.handle)
	if err != nil || !available {
		return nil, err
	}
	return newPictureAlbum(handle), nil
}

// GetPictureFromToken is MediaLibrary::GetPictureFromToken(String).
//
// A token that names no picture answers NIL rather than an error: the token
// comes from outside the application and not finding one is an ordinary
// outcome, which CNA reports as an availability flag.
func (m *MediaLibrary) GetPictureFromToken(token string) (*Picture, error) {
	if err := m.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaLibraryPictureFromToken(m.handle, token)
	if err != nil || !available {
		return nil, err
	}
	return newPicture(handle), nil
}

// SavePictureByStringAndSliceOfByte is
// MediaLibrary::SavePicture(String, Byte[]).
func (m *MediaLibrary) SavePictureByStringAndSliceOfByte(name string, imageBuffer []uint8) (*Picture, error) {
	if err := m.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaLibrarySavePicture(m.handle, name, imageBuffer)
	if err != nil {
		return nil, err
	}
	return newPicture(handle), nil
}

// SavePictureByStringAndStream is MediaLibrary::SavePicture(String, Stream).
//
// The stream is READ INTO MEMORY and handed to the byte-array route, rather
// than passed to CNA's own stream overload. CNA's takes a native stream handle
// and a Go io.Reader has none -- the same wall Foundation 92 met with
// ContentReader -- and the reference's two overloads differ only in where the
// bytes come from.
//
// So the divergence is in the buffering, not in the outcome: a caller handing a
// very large stream pays for it in memory here where the reference would not.
// That is recorded rather than hidden.
func (m *MediaLibrary) SavePictureByStringAndStream(name string, source io.Reader) (*Picture, error) {
	// Three checks in a measured order. DISPOSAL first, because that is the
	// reference's own `ThrowIfDisposed()` at the top of every instance member --
	// and because a disposed library must not consume a caller's stream.
	//
	// Then the ARGUMENT, before the runtime: a nil stream is a programming
	// error that is true whether or not a game is running, and answering
	// "no runtime" there would send the caller looking in the wrong place.
	if m == nil || m.disposed {
		return nil, errMediaDisposed
	}
	if source == nil {
		return nil, mediaArgumentNullError("source")
	}
	if err := m.usable(); err != nil {
		return nil, err
	}
	buffer, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	return m.SavePictureByStringAndSliceOfByte(name, buffer)
}

// Album is Picture::get_Album.
func (p *Picture) Album() (*PictureAlbum, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaPictureAlbumOf(p.handle)
	if err != nil || !available {
		return nil, err
	}
	return newPictureAlbum(handle), nil
}

// Width is Picture::get_Width.
func (p *Picture) Width() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureWidth(p.handle)
}

// Height is Picture::get_Height.
func (p *Picture) Height() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureHeight(p.handle)
}

// Date is Picture::get_Date.
//
// CNA reports UNIX ticks -- seconds since 1970 -- and System.DateTime counts
// 100-nanosecond intervals since year 1. The conversion is done here rather
// than left to the caller, because the contract's type is a DateTime and a
// caller handed a raw number could not know which epoch it was in.
func (p *Picture) Date() (time.Time, error) {
	if err := p.usable(); err != nil {
		return time.Time{}, err
	}
	unix, err := runtimeFor().MediaPictureDateUnixTicks(p.handle)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(unix, 0).UTC(), nil
}

// GetImage is Picture::GetImage(), a Stream over the full image.
//
// The bytes are copied into Go memory and handed back as a reader, for the
// reason Album.GetAlbumArt's are: the native buffer belongs to CNA and a reader
// over it would outlive the call.
func (p *Picture) GetImage() (io.Reader, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	data, err := runtimeFor().MediaPictureImage(p.handle)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// GetThumbnail is Picture::GetThumbnail(), the same shape over the thumbnail.
func (p *Picture) GetThumbnail() (io.Reader, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	data, err := runtimeFor().MediaPictureThumbnail(p.handle)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// Albums is PictureAlbum::get_Albums, the child albums.
func (p *PictureAlbum) Albums() (*PictureAlbumCollection, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPictureAlbumAlbums(p.handle)
	if err != nil {
		return nil, err
	}
	return newPictureAlbumCollection(handle), nil
}

// Pictures is PictureAlbum::get_Pictures.
func (p *PictureAlbum) Pictures() (*PictureCollection, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPictureAlbumPictures(p.handle)
	if err != nil {
		return nil, err
	}
	return newPictureCollection(handle), nil
}

// Parent is PictureAlbum::get_Parent. The ROOT album has none, which CNA
// reports as an availability flag and the reference answers null for.
func (p *PictureAlbum) Parent() (*PictureAlbum, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	handle, available, err := runtimeFor().MediaPictureAlbumParent(p.handle)
	if err != nil || !available {
		return nil, err
	}
	return newPictureAlbum(handle), nil
}

var _ = framework.TimeSpan{}
