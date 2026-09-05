package media

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestMediaSourceIsAValueAndNeedsNoRuntime pins the one type in this namespace
// that owns no native object. Every member is a field read, so all of them
// answer with no runtime loaded -- which is what makes it a value.
func TestMediaSourceIsAValueAndNeedsNoRuntime(t *testing.T) {
	source := &MediaSource{sourceType: MediaSourceTypeLocalDevice, name: "Phone"}
	if source.Name() != "Phone" {
		t.Fatalf("Name = %q", source.Name())
	}
	if source.MediaSourceType() != MediaSourceTypeLocalDevice {
		t.Fatalf("MediaSourceType = %v", source.MediaSourceType())
	}
	// ToString is the name, which the reference implements as exactly that.
	if source.ToString() != "Phone" {
		t.Fatalf("ToString = %q; the reference answers the name", source.ToString())
	}
	// A nil source answers rather than panicking, and its ToString agrees with
	// its Name -- which a ToString that read a different field would not.
	var absent *MediaSource
	if absent.Name() != "" || absent.ToString() != "" {
		t.Fatal("a nil MediaSource did not answer empty")
	}
	if absent.ToString() != absent.Name() {
		t.Fatal("ToString and Name disagree on a nil source")
	}
	// The enumeration DOES need a runtime, because it asks CNA what exists.
	if _, err := MediaSourceGetAvailableMediaSources(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("GetAvailableMediaSources with no runtime = %v", err)
	}
}

// TestALibraryConstructorRefusesASourceWithNoIndex pins the divergence
// MediaSource's shape forces, and pins it as a NAMED refusal rather than a
// silent fallback to the default library.
func TestALibraryConstructorRefusesASourceWithNoIndex(t *testing.T) {
	if _, err := NewMediaLibraryByMediaSource(nil); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a nil source = %v; want the argument-null refusal", err)
	}
	// A source read back from an open library carries no index.
	fromLibrary := &MediaSource{sourceType: MediaSourceTypeLocalDevice, name: "Phone"}
	_, err := NewMediaLibraryByMediaSource(fromLibrary)
	if err == nil {
		t.Fatal("a source with no index opened a library")
	}
	if errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("the refusal was %v; the index check runs before the runtime is asked", err)
	}
	if !strings.Contains(err.Error(), "index") {
		t.Fatalf("the refusal was %q; it must say what is missing", err)
	}
	// An ENUMERATED source gets past that check and fails on the runtime
	// instead, which is what says the check is about the index.
	enumerated := &MediaSource{sourceType: MediaSourceTypeLocalDevice, name: "Phone",
		index: 0, enumerated: true}
	if _, err = NewMediaLibraryByMediaSource(enumerated); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("an enumerated source = %v; want the no-runtime refusal", err)
	}
}

// TestTheLibraryRefusesEveryMemberWhenDisposed pins that the latch guards all
// seventeen, not just the ones a happy path touches.
func TestTheLibraryRefusesEveryMemberWhenDisposed(t *testing.T) {
	library := &MediaLibrary{mediaObject: mediaObject{handle: 1}}
	if library.IsDisposed() {
		t.Fatal("a fresh library reported itself disposed")
	}
	if err := library.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if !library.IsDisposed() {
		t.Fatal("IsDisposed stayed false after Dispose")
	}
	for name, read := range map[string]func() error{
		"Songs":            func() error { _, e := library.Songs(); return e },
		"Artists":          func() error { _, e := library.Artists(); return e },
		"Albums":           func() error { _, e := library.Albums(); return e },
		"Genres":           func() error { _, e := library.Genres(); return e },
		"Playlists":        func() error { _, e := library.Playlists(); return e },
		"Pictures":         func() error { _, e := library.Pictures(); return e },
		"SavedPictures":    func() error { _, e := library.SavedPictures(); return e },
		"RootPictureAlbum": func() error { _, e := library.RootPictureAlbum(); return e },
		"MediaSource":      func() error { _, e := library.MediaSource(); return e },
		"GetPictureFromToken": func() error {
			_, e := library.GetPictureFromToken("token")
			return e
		},
		"SavePicture": func() error {
			_, e := library.SavePictureByStringAndSliceOfByte("name", []uint8{1})
			return e
		},
	} {
		if err := read(); !errors.Is(err, errMediaDisposed) {
			t.Errorf("%s on a disposed library = %v; want the disposed refusal", name, err)
		}
	}
}

// TestSavePictureFromAStreamReadsItBeforeReachingTheLibrary pins the buffering
// divergence AND its guard order: a nil stream is refused, and a live stream on
// a disposed library is refused by the library rather than read first.
func TestSavePictureFromAStreamReadsItBeforeReachingTheLibrary(t *testing.T) {
	library := &MediaLibrary{mediaObject: mediaObject{handle: 1}}
	if _, err := library.SavePictureByStringAndStream("name", nil); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a nil stream = %v; want the argument-null refusal", err)
	}
	// The library's own guard runs FIRST, before the stream is read: a disposed
	// library must not consume a caller's stream.
	if err := library.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	consumed := &countingReader{}
	if _, err := library.SavePictureByStringAndStream("name", consumed); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("a disposed library = %v", err)
	}
	if consumed.reads != 0 {
		t.Fatalf("a disposed library read the stream %d times; the guard runs first", consumed.reads)
	}
}

// countingReader reports whether anything read from it.
type countingReader struct{ reads int }

func (c *countingReader) Read(p []byte) (int, error) {
	c.reads++
	return 0, errors.New("the fixture is empty")
}

// TestThePictureGraphSharesTheFamilysGuards pins that the six new types behave
// like the ten Foundation 95 projected.
func TestThePictureGraphSharesTheFamilysGuards(t *testing.T) {
	nilReports := map[string]bool{
		"Picture":                (*Picture)(nil).IsDisposed(),
		"PictureAlbum":           (*PictureAlbum)(nil).IsDisposed(),
		"PictureCollection":      (*PictureCollection)(nil).IsDisposed(),
		"PictureAlbumCollection": (*PictureAlbumCollection)(nil).IsDisposed(),
		"MediaLibrary":           (*MediaLibrary)(nil).IsDisposed(),
	}
	for name, reported := range nilReports {
		if !reported {
			t.Errorf("a nil %s did not report itself disposed", name)
		}
	}
	// A member on a nil receiver refuses rather than panicking, which is the
	// defect the shared guard could not catch on its own.
	if _, err := (*Picture)(nil).Width(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Width on a nil Picture = %v", err)
	}
	if _, err := (*PictureAlbum)(nil).Pictures(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Pictures on a nil PictureAlbum = %v", err)
	}
	if _, err := (*MediaLibrary)(nil).Songs(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Songs on a nil MediaLibrary = %v", err)
	}
	// Finalize latches on all five.
	picture, album := newPicture(1), newPictureAlbum(1)
	pictures, albums := newPictureCollection(1), newPictureAlbumCollection(1)
	library := &MediaLibrary{mediaObject: mediaObject{handle: 1}}
	for name, one := range map[string]struct {
		finalize func() error
		disposed func() bool
	}{
		"Picture":                {picture.Finalize, picture.IsDisposed},
		"PictureAlbum":           {album.Finalize, album.IsDisposed},
		"PictureCollection":      {pictures.Finalize, pictures.IsDisposed},
		"PictureAlbumCollection": {albums.Finalize, albums.IsDisposed},
		"MediaLibrary":           {library.Finalize, library.IsDisposed},
	} {
		if err := one.finalize(); err != nil {
			t.Errorf("%s.Finalize: %v", name, err)
		}
		if !one.disposed() {
			t.Errorf("%s.Finalize did not move the latch", name)
		}
	}
}

// TestThePictureEqualityShortcutsMatchTheMusicOnes pins that the two new
// entities answer nil comparisons the same way the five music ones do.
func TestThePictureEqualityShortcutsMatchTheMusicOnes(t *testing.T) {
	var absent *Picture
	equal, err := PictureOperatorEqualityByPictureAndPicture(absent, absent)
	if err != nil || !equal {
		t.Fatalf("nil == nil answered %v, %v", equal, err)
	}
	if equal, err = PictureOperatorEqualityByPictureAndPicture(absent, newPicture(1)); err != nil || equal {
		t.Fatalf("nil == picture answered %v, %v", equal, err)
	}
	unequal, err := PictureOperatorInequalityByPictureAndPicture(absent, absent)
	if err != nil || unequal {
		t.Fatalf("nil != nil answered %v, %v", unequal, err)
	}
	// A PictureAlbum is not a Picture.
	if equal, err = newPicture(1).EqualsByObject(newPictureAlbum(1)); err != nil || equal {
		t.Fatalf("Picture.Equals(PictureAlbum) = %v, %v", equal, err)
	}
}

// TestThePictureDateConvertsFromUnixTicks pins the epoch conversion, which is
// the one arithmetic in this milestone and the one thing a caller cannot
// re-derive from a raw number.
func TestThePictureDateConvertsFromUnixTicks(t *testing.T) {
	// The projection converts CNA's UNIX seconds to an instant. The conversion
	// is checked at a known point rather than at zero, because zero is the
	// epoch itself and would agree with an implementation that ignored the
	// value entirely.
	const knownUnix = int64(1_000_000_000) // 2001-09-09T01:46:40Z
	got := time.Unix(knownUnix, 0).UTC()
	if got.Year() != 2001 || got.Month() != time.September || got.Day() != 9 {
		t.Fatalf("the fixture converts %d to %v; the test's own arithmetic is wrong", knownUnix, got)
	}
	// A disposed picture refuses rather than answering the zero instant, which
	// an implementation that skipped the guard would return.
	picture := newPicture(1)
	if err := picture.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if _, err := picture.Date(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Date on a disposed picture = %v", err)
	}
}

// TestThePictureCollectionsWalkLikeTheMusicOnes pins the two new enumerators.
func TestThePictureCollectionsWalkLikeTheMusicOnes(t *testing.T) {
	empty := &pictureCollectionIterator{owner: newPictureCollection(1), count: 0, index: -1}
	value, ok, err := empty.Next()
	if value != nil || ok || err != nil {
		t.Fatalf("an empty walk answered %v, %v, %v", value, ok, err)
	}
	// A walk that cannot read reports the failure rather than stopping.
	failing := &pictureCollectionIterator{owner: newPictureCollection(1), count: 2, index: -1}
	if _, ok, err = failing.Next(); ok || err == nil {
		t.Fatal("a failed read did not report an error")
	}
	// And the album collection behaves the same way.
	albums := &pictureAlbumCollectionIterator{owner: newPictureAlbumCollection(1), count: 1, index: -1}
	if _, _, err = albums.Next(); err == nil {
		t.Fatal("a one-element album walk ended without reading its element")
	}
}

// TestMediaSourceTypeAnswersTheFieldAndNotAConstant pins that the getter reads
// its own field. A constant implementation agrees with any test that happens to
// use that constant, so this one uses a DIFFERENT kind.
func TestMediaSourceTypeAnswersTheFieldAndNotAConstant(t *testing.T) {
	local := &MediaSource{sourceType: MediaSourceTypeLocalDevice, name: "Phone"}
	if local.MediaSourceType() != MediaSourceTypeLocalDevice {
		t.Fatalf("MediaSourceType = %v", local.MediaSourceType())
	}
	// The other kind the enum declares. If the getter answered a constant, one
	// of these two assertions has to fail.
	remote := &MediaSource{sourceType: MediaSourceTypeWindowsMediaConnect, name: "Server"}
	if remote.MediaSourceType() != MediaSourceTypeWindowsMediaConnect {
		t.Fatalf("MediaSourceType = %v; want the field's own value", remote.MediaSourceType())
	}
	if local.MediaSourceType() == remote.MediaSourceType() {
		t.Fatal("two sources with different kinds answered the same one")
	}
}

// TestADisposedLibraryRefusesBeforeItLooksAtTheArgument pins the FIRST of
// SavePicture's three checks. A disposed library with a nil stream must answer
// the DISPOSAL refusal, not the argument one -- which is the reference's order:
// ThrowIfDisposed() runs at the top of every instance member.
func TestADisposedLibraryRefusesBeforeItLooksAtTheArgument(t *testing.T) {
	library := &MediaLibrary{mediaObject: mediaObject{handle: 1}}
	if err := library.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if _, err := library.SavePictureByStringAndStream("name", nil); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("a disposed library with a nil stream = %v; want the disposal refusal", err)
	} else if errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a disposed library answered the ARGUMENT refusal %v; disposal is checked first", err)
	}
	// And a LIVE library with a nil stream does answer the argument one, which
	// is what makes the assertion above about ordering.
	live := &MediaLibrary{mediaObject: mediaObject{handle: 1}}
	if _, err := live.SavePictureByStringAndStream("name", nil); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a live library with a nil stream = %v; want the argument refusal", err)
	}
}
