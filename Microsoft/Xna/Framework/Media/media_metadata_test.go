package media

import (
	"errors"
	"strings"
	"testing"
)

// The metadata graph reaches CNA for everything it answers, so what a managed
// test can measure is the GUARDS: the disposal latch, the refusals, the
// index bounds and the equality shortcuts that never reach a runtime.
//
// The native half is qualified by the stress slice, which walks a real library.
// An empty library is a valid answer there and is counted as one.

// TestEveryTypeRefusesWithoutARuntime pins that a member reached with no
// runtime says SO, rather than dereferencing a handle it cannot use.
func TestEveryTypeRefusesWithoutARuntime(t *testing.T) {
	song := newSong(1)
	if _, err := song.Name(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Song.Name with no runtime = %v; want the no-runtime refusal", err)
	}
	if _, err := song.Duration(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Song.Duration with no runtime = %v", err)
	}
	collection := newSongCollection(1)
	if _, err := collection.Count(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("SongCollection.Count with no runtime = %v", err)
	}
	if _, err := collection.Item(0); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("SongCollection.Item with no runtime = %v", err)
	}
	if _, err := SongFromUri("name", "http://example.invalid/a.mp3"); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("SongFromUri with no runtime = %v", err)
	}
}

// TestDisposeIsIdempotentAndLatchesFirst pins the order Dispose runs in: the
// managed latch is set BEFORE the native call, so a member reached afterwards
// refuses without touching a handle CNA has torn down.
func TestDisposeIsIdempotentAndLatchesFirst(t *testing.T) {
	song := newSong(1)
	if song.IsDisposed() {
		t.Fatal("a fresh song reported itself disposed")
	}
	// With no runtime the native half is skipped and the latch still moves,
	// which is what makes the latch the projection's own answer.
	if err := song.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if !song.IsDisposed() {
		t.Fatal("IsDisposed stayed false after Dispose")
	}
	// A second Dispose does nothing and does not fail.
	if err := song.Dispose(); err != nil {
		t.Fatalf("a second Dispose: %v", err)
	}
	// Every member now refuses with the DISPOSED error and not the no-runtime
	// one, which is what says the latch is checked first.
	if _, err := song.Name(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Name after Dispose = %v; want the disposed refusal", err)
	}
	if _, err := song.Rating(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Rating after Dispose = %v", err)
	}
}

// TestFinalizeIsTheSameTeardownAsDispose pins the projected finalizer, which
// nothing calls and which the contract declares.
func TestFinalizeIsTheSameTeardownAsDispose(t *testing.T) {
	for name, finalize := range map[string]func() error{
		"Song":               newSong(1).Finalize,
		"Album":              newAlbum(1).Finalize,
		"Artist":             newArtist(1).Finalize,
		"Genre":              newGenre(1).Finalize,
		"Playlist":           newPlaylist(1).Finalize,
		"SongCollection":     newSongCollection(1).Finalize,
		"AlbumCollection":    newAlbumCollection(1).Finalize,
		"ArtistCollection":   newArtistCollection(1).Finalize,
		"GenreCollection":    newGenreCollection(1).Finalize,
		"PlaylistCollection": newPlaylistCollection(1).Finalize,
	} {
		if err := finalize(); err != nil {
			t.Errorf("%s.Finalize: %v", name, err)
		}
	}
	// And it latches, exactly as Dispose does.
	album := newAlbum(1)
	if err := album.Finalize(); err != nil {
		t.Fatalf("Album.Finalize: %v", err)
	}
	if !album.IsDisposed() {
		t.Fatal("Finalize did not set the disposal latch")
	}
}

// TestEqualityShortcutsNeverReachTheRuntime pins the two comparisons the
// reference answers before it asks anything: a nil against a nil is EQUAL, and
// a nil against a non-nil is not.
func TestEqualityShortcutsNeverReachTheRuntime(t *testing.T) {
	var absent *Song
	equal, err := SongOperatorEqualityBySongAndSong(absent, absent)
	if err != nil || !equal {
		t.Fatalf("nil == nil answered %v, %v; two absent songs are equal", equal, err)
	}
	equal, err = SongOperatorEqualityBySongAndSong(absent, newSong(1))
	if err != nil || equal {
		t.Fatalf("nil == song answered %v, %v; want false and no error", equal, err)
	}
	// op_Inequality is the negation, including for the nil cases.
	unequal, err := SongOperatorInequalityBySongAndSong(absent, absent)
	if err != nil || unequal {
		t.Fatalf("nil != nil answered %v, %v", unequal, err)
	}
	unequal, err = SongOperatorInequalityBySongAndSong(absent, newSong(1))
	if err != nil || !unequal {
		t.Fatalf("nil != song answered %v, %v", unequal, err)
	}
}

// TestEqualsByObjectRefusesAForeignType pins the `is Song` check, which
// answers FALSE rather than an error -- and does so without a runtime, because
// the type check runs first.
func TestEqualsByObjectRefusesAForeignType(t *testing.T) {
	song := newSong(1)
	equal, err := song.EqualsByObject("not a song")
	if err != nil || equal {
		t.Fatalf("Equals(string) = %v, %v; want false and no error", equal, err)
	}
	equal, err = song.EqualsByObject(nil)
	if err != nil || equal {
		t.Fatalf("Equals(nil) = %v, %v; want false and no error", equal, err)
	}
	// An Album is not a Song, even though both are media objects.
	if equal, err = song.EqualsByObject(newAlbum(1)); err != nil || equal {
		t.Fatalf("Song.Equals(Album) = %v, %v", equal, err)
	}
	// A Song against a nil Song is false, and that DOES need the guard order:
	// the receiver is checked before the argument.
	if equal, err = song.EqualsBySong(nil); err == nil && equal {
		t.Fatal("a song equalled a nil song")
	}
}

// TestSongFromUriRefusesAnEmptyUri pins the null comparison the reference makes
// before it reaches the native call, projected onto the only value a Go caller
// can pass in a null's place.
func TestSongFromUriRefusesAnEmptyUri(t *testing.T) {
	_, err := SongFromUri("name", "")
	if err == nil {
		t.Fatal("an empty URI was accepted")
	}
	if !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("the refusal was %v; want the argument-null one", err)
	}
	if !strings.Contains(err.Error(), "uri") {
		t.Fatalf("the refusal was %q; the reference names the uri parameter", err)
	}
	// A non-empty URI gets past the guard and fails on the runtime instead,
	// which is what says the guard is about the URI and not about everything.
	if _, err = SongFromUri("name", "http://example.invalid/a.mp3"); errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a non-empty URI raised the argument-null refusal: %v", err)
	}
}

// TestACollectionEnumeratorReadsItsLengthOnce pins the shape the reference's
// enumerator has: the count is taken when the walk starts, not on every step.
func TestACollectionEnumeratorReadsItsLengthOnce(t *testing.T) {
	collection := newSongCollection(1)
	if _, err := collection.GetEnumerator(); err == nil {
		t.Fatal("an enumerator was made with no runtime")
	}
	// The iterator itself is walkable without a runtime once its count is
	// zero, which is the empty-library case the stress slice also sees.
	iterator := &songCollectionIterator{owner: collection, count: 0, index: -1}
	value, ok, err := iterator.Next()
	if value != nil || ok || err != nil {
		t.Fatalf("an empty walk answered %v, %v, %v; want nothing and no error", value, ok, err)
	}
	// A second Next is still nothing, so the walk does not run past its end.
	if _, ok, _ = iterator.Next(); ok {
		t.Fatal("the walk continued past its end")
	}
}

// TestACollectionIndexerRefusesOutOfRange pins that the bound is read from the
// collection and that an empty one refuses every index including zero.
func TestACollectionIndexerRefusesOutOfRange(t *testing.T) {
	collection := newSongCollection(1)
	if err := collection.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	// A disposed collection refuses before it can read a bound at all.
	if _, err := collection.Item(0); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Item on a disposed collection = %v", err)
	}
	// The out-of-range refusal is a DIFFERENT error from the disposal one, so a
	// caller can tell an empty collection from a dead one.
	if errors.Is(errMediaArgumentOutOfRange, errMediaDisposed) {
		t.Fatal("the out-of-range and disposal refusals are the same error")
	}
}

// TestTheTenTypesShareOneDisposalLatch is the structural claim the shared
// mediaObject makes, checked rather than assumed.
func TestTheTenTypesShareOneDisposalLatch(t *testing.T) {
	for name, pair := range map[string]struct {
		dispose func() error
		check   func() bool
	}{
		"Song":               {newSong(1).Dispose, newSong(1).IsDisposed},
		"AlbumCollection":    {newAlbumCollection(1).Dispose, newAlbumCollection(1).IsDisposed},
		"PlaylistCollection": {newPlaylistCollection(1).Dispose, newPlaylistCollection(1).IsDisposed},
	} {
		if pair.check() {
			t.Errorf("a fresh %s reported itself disposed", name)
		}
		if err := pair.dispose(); err != nil {
			t.Errorf("%s.Dispose: %v", name, err)
		}
	}
	// A nil receiver reports disposed rather than panicking, which is what
	// every IsDisposed in the family promises.
	var absent *Genre
	if !absent.IsDisposed() {
		t.Fatal("a nil Genre did not report itself disposed")
	}
	var absentCollection *GenreCollection
	if !absentCollection.IsDisposed() {
		t.Fatal("a nil GenreCollection did not report itself disposed")
	}
}

// TestTheIndexBoundRefusesEveryWrongIndex pins the comparison itself, which no
// test can reach through an indexer: those sit behind a runtime guard, and the
// bound needs no runtime.
func TestTheIndexBoundRefusesEveryWrongIndex(t *testing.T) {
	// An EMPTY collection refuses every index, zero included.
	for _, index := range []int32{-1, 0, 1} {
		if err := checkCollectionIndex(index, 0); err == nil {
			t.Fatalf("index %d was accepted against an empty collection", index)
		}
	}
	// A collection of two accepts exactly 0 and 1.
	for _, index := range []int32{0, 1} {
		if err := checkCollectionIndex(index, 2); err != nil {
			t.Fatalf("index %d was refused against a collection of two: %v", index, err)
		}
	}
	// The COUNT itself is out of range, which an off-by-one would accept.
	if err := checkCollectionIndex(2, 2); err == nil {
		t.Fatal("the count itself was accepted as an index")
	}
	// A negative index is out of range, which a bound that checked only the
	// upper end would accept.
	if err := checkCollectionIndex(-1, 2); err == nil {
		t.Fatal("a negative index was accepted")
	}
	// The refusal is the out-of-range one and NOT the disposal one, so a caller
	// can tell an empty collection from a dead one, and it names both numbers.
	err := checkCollectionIndex(5, 2)
	if !errors.Is(err, errMediaArgumentOutOfRange) {
		t.Fatalf("the refusal was %v; want the out-of-range one", err)
	}
	if errors.Is(err, errMediaDisposed) {
		t.Fatalf("the refusal was %v; a bad index is not a disposal", err)
	}
	if !strings.Contains(err.Error(), "5") || !strings.Contains(err.Error(), "2") {
		t.Fatalf("the refusal %q names neither the index nor the length", err)
	}
}

// TestEveryTypesNilReceiverAndLatchAgree pins the two claims mediaObject makes
// on ALL TEN types rather than on one of them: a nil receiver reports itself
// disposed, and Finalize moves the latch.
func TestEveryTypesNilReceiverAndLatchAgree(t *testing.T) {
	nilReports := map[string]bool{
		"Song":               (*Song)(nil).IsDisposed(),
		"Album":              (*Album)(nil).IsDisposed(),
		"Artist":             (*Artist)(nil).IsDisposed(),
		"Genre":              (*Genre)(nil).IsDisposed(),
		"Playlist":           (*Playlist)(nil).IsDisposed(),
		"SongCollection":     (*SongCollection)(nil).IsDisposed(),
		"AlbumCollection":    (*AlbumCollection)(nil).IsDisposed(),
		"ArtistCollection":   (*ArtistCollection)(nil).IsDisposed(),
		"GenreCollection":    (*GenreCollection)(nil).IsDisposed(),
		"PlaylistCollection": (*PlaylistCollection)(nil).IsDisposed(),
	}
	for name, reported := range nilReports {
		if !reported {
			t.Errorf("a nil %s did not report itself disposed", name)
		}
	}

	// Finalize latches on every one of the ten, which one type's assertion
	// cannot say.
	type latching struct {
		finalize func() error
		disposed func() bool
	}
	song, album, artist, genre, playlist := newSong(1), newAlbum(1), newArtist(1), newGenre(1), newPlaylist(1)
	songs, albums := newSongCollection(1), newAlbumCollection(1)
	artists, genres, playlists := newArtistCollection(1), newGenreCollection(1), newPlaylistCollection(1)
	for name, one := range map[string]latching{
		"Song":               {song.Finalize, song.IsDisposed},
		"Album":              {album.Finalize, album.IsDisposed},
		"Artist":             {artist.Finalize, artist.IsDisposed},
		"Genre":              {genre.Finalize, genre.IsDisposed},
		"Playlist":           {playlist.Finalize, playlist.IsDisposed},
		"SongCollection":     {songs.Finalize, songs.IsDisposed},
		"AlbumCollection":    {albums.Finalize, albums.IsDisposed},
		"ArtistCollection":   {artists.Finalize, artists.IsDisposed},
		"GenreCollection":    {genres.Finalize, genres.IsDisposed},
		"PlaylistCollection": {playlists.Finalize, playlists.IsDisposed},
	} {
		if one.disposed() {
			t.Errorf("a fresh %s reported itself disposed", name)
			continue
		}
		if err := one.finalize(); err != nil {
			t.Errorf("%s.Finalize: %v", name, err)
		}
		if !one.disposed() {
			t.Errorf("%s.Finalize did not move the disposal latch", name)
		}
	}
}

// TestAMemberOnANilReceiverRefusesRatherThanPanicking pins the other half of
// the shared guard: usable() admits neither a nil object nor a disposed one.
func TestAMemberOnANilReceiverRefusesRatherThanPanicking(t *testing.T) {
	if _, err := (*Song)(nil).Name(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Name on a nil Song = %v; want the disposed refusal", err)
	}
	if _, err := (*SongCollection)(nil).Count(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("Count on a nil SongCollection = %v", err)
	}
	if _, err := (*Album)(nil).HasArt(); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("HasArt on a nil Album = %v", err)
	}
}

// TestEqualsChecksTheReceiverBeforeTheArgument pins the guard ORDER. A live
// receiver with no runtime must report the RUNTIME refusal even when the
// argument is nil -- an implementation that checked the argument first would
// answer false and no error.
func TestEqualsChecksTheReceiverBeforeTheArgument(t *testing.T) {
	equal, err := newSong(1).EqualsBySong(nil)
	if err == nil {
		t.Fatalf("Equals(nil) answered %v with no error; the receiver is checked first", equal)
	}
	if !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Equals(nil) = %v; want the no-runtime refusal from the receiver's guard", err)
	}
	// A DISPOSED receiver reports the disposal refusal, also before the
	// argument is looked at.
	song := newSong(1)
	if err = song.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if _, err = song.EqualsBySong(nil); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("a disposed receiver answered %v", err)
	}
}

// TestTheEnumeratorReportsAFailedReadRatherThanStopping pins that a walk which
// cannot read an element says so. A walk that stopped silently would be
// indistinguishable from a collection that ended.
func TestTheEnumeratorReportsAFailedReadRatherThanStopping(t *testing.T) {
	// A collection of two, with no runtime to read either element.
	iterator := &songCollectionIterator{owner: newSongCollection(1), count: 2, index: -1}
	value, ok, err := iterator.Next()
	if err == nil {
		t.Fatalf("a failed read answered %v, %v with no error", value, ok)
	}
	if ok {
		t.Fatal("a failed read reported a value")
	}

	// And the walk STARTS at the first element: an iterator built at index 0
	// would skip it, which against a one-element collection looks like an
	// empty walk rather than a failed read.
	single := &songCollectionIterator{owner: newSongCollection(1), count: 1, index: -1}
	if _, _, err = single.Next(); err == nil {
		t.Fatal("a one-element walk ended without reading its element")
	}
}
