package media

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"strings"
	"testing"
)

// TestEveryMediaPlayerMemberRefusesWithoutARuntime pins that all twenty static
// members answer the Go-only refusal rather than reaching a game that is not
// there. Every route takes the game, so there is no member that could work.
func TestEveryMediaPlayerMemberRefusesWithoutARuntime(t *testing.T) {
	for name, call := range map[string]func() error{
		"Pause":        MediaPlayerPause,
		"Resume":       MediaPlayerResume,
		"Stop":         MediaPlayerStop,
		"MoveNext":     MediaPlayerMoveNext,
		"MovePrevious": MediaPlayerMovePrevious,
		"IsShuffled":   func() error { _, e := MediaPlayerIsShuffled(); return e },
		"IsRepeating":  func() error { _, e := MediaPlayerIsRepeating(); return e },
		"IsMuted":      func() error { _, e := MediaPlayerIsMuted(); return e },
		"Volume":       func() error { _, e := MediaPlayerVolume(); return e },
		"State":        func() error { _, e := MediaPlayerState(); return e },
		"PlayPosition": func() error { _, e := MediaPlayerPlayPosition(); return e },
		"Queue":        func() error { _, e := MediaPlayerQueue(); return e },
		"GameHasControl": func() error {
			_, e := MediaPlayerGameHasControl()
			return e
		},
		"IsVisualizationEnabled": func() error {
			_, e := MediaPlayerIsVisualizationEnabled()
			return e
		},
		"SetIsShuffled":  func() error { return SetMediaPlayerIsShuffled(true) },
		"SetIsRepeating": func() error { return SetMediaPlayerIsRepeating(true) },
		"SetIsMuted":     func() error { return SetMediaPlayerIsMuted(true) },
		"SetVolume":      func() error { return SetMediaPlayerVolume(0.5) },
		"SetIsVisualizationEnabled": func() error {
			return SetMediaPlayerIsVisualizationEnabled(true)
		},
		"GetVisualizationData": func() error {
			return MediaPlayerGetVisualizationData(NewVisualizationData())
		},
	} {
		if err := call(); !errors.Is(err, errMediaNoRuntime) {
			t.Errorf("MediaPlayer.%s with no runtime = %v; want the no-runtime refusal", name, err)
		}
	}
}

// TestMediaPlayerPlayChecksTheArgumentBeforeTheRuntime pins the guard order the
// whole family now shares: an invalid argument is invalid whether or not a game
// is running, so it is answered first.
//
// MediaPlayer is static, so there is no disposal latch to precede it -- which
// is the only difference from VideoPlayer.Play, where disposal comes first.
func TestMediaPlayerPlayChecksTheArgumentBeforeTheRuntime(t *testing.T) {
	for name, call := range map[string]func() error{
		"Play(Song)":                  func() error { return MediaPlayerPlayBySong(nil) },
		"Play(SongCollection)":        func() error { return MediaPlayerPlayBySongCollection(nil) },
		"Play(SongCollection, Int32)": func() error { return MediaPlayerPlayBySongCollectionAndInt32(nil, 0) },
		"GetVisualizationData":        func() error { return MediaPlayerGetVisualizationData(nil) },
	} {
		err := call()
		if !errors.Is(err, errMediaArgumentNull) {
			t.Errorf("MediaPlayer.%s(nil) = %v; want the argument-null refusal", name, err)
		}
		if errors.Is(err, errMediaNoRuntime) {
			t.Errorf("MediaPlayer.%s(nil) answered the no-runtime refusal; the argument is checked first", name)
		}
	}
	// A NON-nil argument gets past the guard and reaches the runtime check,
	// which is what says the guard is about the argument.
	if err := MediaPlayerPlayBySong(newSong(1)); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Play(song) with no runtime = %v", err)
	}
	if err := MediaPlayerGetVisualizationData(NewVisualizationData()); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("GetVisualizationData(data) with no runtime = %v", err)
	}
}

// TestGetVisualizationDataNamesTheParameterItRefuses pins the refusal's name,
// which is what tells a caller WHICH argument was wrong.
func TestGetVisualizationDataNamesTheParameterItRefuses(t *testing.T) {
	err := MediaPlayerGetVisualizationData(nil)
	if err == nil {
		t.Fatal("a nil visualization target was accepted")
	}
	if !strings.Contains(err.Error(), "visualizationData") {
		t.Fatalf("the refusal was %q; it must name the parameter", err)
	}
}

// TestVisualizationDataViewsStayLiveOverTheBuffers is the property the whole
// type turns on, and Foundation 97 is the first milestone where anything
// writes into it.
func TestVisualizationDataViewsStayLiveOverTheBuffers(t *testing.T) {
	data := NewVisualizationData()
	frequencies := data.Frequencies()
	samples := data.Samples()
	if frequencies == nil || samples == nil {
		t.Fatal("a fresh VisualizationData has no views")
	}

	// Fill it the way MediaPlayer::GetVisualizationData does.
	data.setFrequencies([]float32{1, 2, 3})
	data.setSamples([]float32{4, 5})

	// The views the caller ALREADY holds see the new data, because the write
	// copies into the arrays the views were built over.
	first, err := frequencies.Item(0)
	if err != nil || first != 1 {
		t.Fatalf("the frequency view answered %v, %v; want the value just written", first, err)
	}
	second, err := samples.Item(1)
	if err != nil || second != 5 {
		t.Fatalf("the sample view answered %v, %v; want the value just written", second, err)
	}

	// A SHORTER fill zeroes the rest, so nothing stale is visible.
	data.setFrequencies([]float32{9})
	stale, err := frequencies.Item(1)
	if err != nil || stale != 0 {
		t.Fatalf("a shorter fill left %v at index 1; the remainder must be zeroed", stale)
	}
	// And the length never changes: both buffers are fixed at 256.
	if frequencies.Count() != visualizationBufferLength {
		t.Fatalf("the frequency view is %d long; the buffers are fixed", frequencies.Count())
	}
}

// TestAVideoPlayerLatchesLikeTheRestOfTheFamily pins the one playback type that
// owns a handle.
func TestAVideoPlayerLatchesLikeTheRestOfTheFamily(t *testing.T) {
	if _, err := NewVideoPlayer(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("NewVideoPlayer with no runtime = %v", err)
	}
	player := &VideoPlayer{handle: 1}
	if player.IsDisposed() {
		t.Fatal("a fresh player reported itself disposed")
	}
	if err := player.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if !player.IsDisposed() {
		t.Fatal("IsDisposed stayed false after Dispose")
	}
	// Every member now refuses with the DISPOSED error, not the runtime one.
	for name, call := range map[string]func() error{
		"Play":         func() error { return player.Play(nil) },
		"Pause":        player.Pause,
		"Resume":       player.Resume,
		"Stop":         player.Stop,
		"GetTexture":   func() error { _, e := player.GetTexture(); return e },
		"Video":        func() error { _, e := player.Video(); return e },
		"State":        func() error { _, e := player.State(); return e },
		"IsLooped":     func() error { _, e := player.IsLooped(); return e },
		"Volume":       func() error { _, e := player.Volume(); return e },
		"PlayPosition": func() error { _, e := player.PlayPosition(); return e },
		"SetIsLooped":  func() error { return player.SetIsLooped(true) },
		"SetVolume":    func() error { return player.SetVolume(0.5) },
	} {
		if err := call(); !errors.Is(err, errMediaDisposed) {
			t.Errorf("VideoPlayer.%s on a disposed player = %v", name, err)
		}
	}
	// A nil receiver reports disposed rather than panicking.
	if !(*VideoPlayer)(nil).IsDisposed() {
		t.Fatal("a nil VideoPlayer did not report itself disposed")
	}
	// Finalize is the same teardown.
	other := &VideoPlayer{handle: 1}
	if err := other.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !other.IsDisposed() {
		t.Fatal("Finalize did not move the latch")
	}
}

// TestAVideoAndAQueueHaveNoDisposal pins the two types whose contract declares
// none -- which is what makes their shape and their contract agree.
func TestAVideoAndAQueueHaveNoDisposal(t *testing.T) {
	// A nil receiver refuses by NAME rather than panicking.
	if _, err := (*Video)(nil).Width(); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("Width on a nil Video = %v", err)
	}
	if _, err := (*MediaQueue)(nil).Count(); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("Count on a nil MediaQueue = %v", err)
	}
	// A live one with no runtime reaches the runtime refusal, which is what
	// says the nil check is about the receiver.
	if _, err := (&Video{handle: 1}).Width(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Width on a live Video = %v", err)
	}
	if _, err := (&MediaQueue{handle: 1}).Count(); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Count on a live MediaQueue = %v", err)
	}
}

// TestTheEventAccessorsRefuseANilHandler pins the add accessor's one argument
// check.
func TestTheEventAccessorsRefuseANilHandler(t *testing.T) {
	// With no runtime the runtime check wins on both, which is what says the
	// accessors reach it at all -- a static event with no game has nothing to
	// subscribe to.
	// A nil handler is refused BEFORE the runtime, the family's settled order.
	if _, err := MediaPlayerAddActiveSongChangedHandler(nil); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("adding a nil ActiveSongChanged handler = %v; want the argument refusal", err)
	}
	if _, err := MediaPlayerAddMediaStateChangedHandler(nil); !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("adding a nil MediaStateChanged handler = %v", err)
	}
	// A real handler gets past it and reaches the runtime check.
	handler := func(sender any, args *framework.EventArgs) error { return nil }
	if _, err := MediaPlayerAddActiveSongChangedHandler(handler); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("adding a real handler = %v; want the no-runtime refusal", err)
	}
	if err := MediaPlayerRemoveActiveSongChangedHandler(framework.EventSubscription{}); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("removing an ActiveSongChanged handler = %v", err)
	}
}

// TestTheQueueIndexerRefusesOutOfRange pins that MediaQueue shares the family's
// bound even though it shares none of its disposal.
func TestTheQueueIndexerRefusesOutOfRange(t *testing.T) {
	queue := &MediaQueue{handle: 1}
	// With no runtime the count cannot be read, so the refusal is the runtime
	// one -- the bound is checked AFTER the count, which is the only order
	// that can work.
	if _, err := queue.Item(0); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("Item on a live queue with no runtime = %v", err)
	}
	if err := queue.SetActiveSongIndex(0); !errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("SetActiveSongIndex with no runtime = %v", err)
	}
	// The bound itself is the family's, and it refuses an empty queue's zero.
	if err := checkCollectionIndex(0, 0); err == nil {
		t.Fatal("an empty queue accepted index zero")
	}
	if !strings.Contains(checkCollectionIndex(3, 2).Error(), "3") {
		t.Fatal("the queue's refusal does not name the index")
	}
}

// TestVideoPlayerPlayChecksDisposalThenTheArgument pins the two-step order on a
// LIVE player, which a disposed one cannot show: with the latch satisfied, a
// nil video must answer the ARGUMENT refusal rather than reaching the runtime.
func TestVideoPlayerPlayChecksDisposalThenTheArgument(t *testing.T) {
	live := &VideoPlayer{handle: 1}
	err := live.Play(nil)
	if !errors.Is(err, errMediaArgumentNull) {
		t.Fatalf("a live player with a nil video = %v; want the argument refusal", err)
	}
	if errors.Is(err, errMediaNoRuntime) {
		t.Fatalf("a live player with a nil video answered %v; the argument is checked first", err)
	}
	if !strings.Contains(err.Error(), "video") {
		t.Fatalf("the refusal was %q; it must name the parameter", err)
	}
	// And a DISPOSED player answers the disposal refusal even with a nil video,
	// which is what makes disposal first.
	disposed := &VideoPlayer{handle: 1}
	if err = disposed.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if err = disposed.Play(nil); !errors.Is(err, errMediaDisposed) {
		t.Fatalf("a disposed player with a nil video = %v; want the disposal refusal", err)
	}
}
