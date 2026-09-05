package media

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// MediaPlayer is Microsoft.Xna.Framework.Media.MediaPlayer, and every one of
// its twenty members is STATIC.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # A static class projects to package functions, not to a type
//
// The contract declares no constructor and no instance member. So there is no
// MediaPlayer VALUE to project -- what a consumer reaches is a set of package
// functions, which is the settled spelling for a CLR static in this projection.
//
// The MediaPlayer prefix is what keeps them apart from the Song and Library
// members that share their names, and the empty marker type carries the CLR
// identity -- the settled shape TitleContainer and FrameworkDispatcher have.

// MediaPlayer is the XNA static media-player type identity, `public abstract
// sealed` in the reference. It holds nothing: all twenty of its members are
// static and project to type-prefixed package declarations.
type MediaPlayer struct{}

// MediaPlayerPlayBySong is MediaPlayer::Play(Song).
func MediaPlayerPlayBySong(song *Song) error {
	// The ARGUMENT first. MediaPlayer is static, so there is no disposal latch
	// to precede it, and a nil song is a programming error that is true
	// whether or not a game is running -- the rule Foundation 95 and 96 both
	// had to learn.
	if song == nil {
		return mediaArgumentNullError("song")
	}
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	if err := song.usable(); err != nil {
		return err
	}
	return runtime.MediaPlayerPlaySong(song.handle)
}

// MediaPlayerPlayBySongCollection is MediaPlayer::Play(SongCollection).
func MediaPlayerPlayBySongCollection(collection *SongCollection) error {
	if collection == nil {
		return mediaArgumentNullError("collection")
	}
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	if err := collection.usable(); err != nil {
		return err
	}
	return runtime.MediaPlayerPlaySongs(collection.handle)
}

// MediaPlayerPlayBySongCollectionAndInt32 is
// MediaPlayer::Play(SongCollection, Int32), which starts at a given index.
func MediaPlayerPlayBySongCollectionAndInt32(collection *SongCollection, index int32) error {
	if collection == nil {
		return mediaArgumentNullError("collection")
	}
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	if err := collection.usable(); err != nil {
		return err
	}
	count, err := collection.Count()
	if err != nil {
		return err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return err
	}
	return runtime.MediaPlayerPlaySongsFrom(collection.handle, index)
}

// MediaPlayerPause is MediaPlayer::Pause().
func MediaPlayerPause() error { return mediaPlayerCall((*interop.Runtime).MediaPlayerPause) }

// MediaPlayerResume is MediaPlayer::Resume().
func MediaPlayerResume() error { return mediaPlayerCall((*interop.Runtime).MediaPlayerResume) }

// MediaPlayerStop is MediaPlayer::Stop().
func MediaPlayerStop() error { return mediaPlayerCall((*interop.Runtime).MediaPlayerStop) }

// MediaPlayerMoveNext is MediaPlayer::MoveNext().
func MediaPlayerMoveNext() error { return mediaPlayerCall((*interop.Runtime).MediaPlayerMoveNext) }

// MediaPlayerMovePrevious is MediaPlayer::MovePrevious().
func MediaPlayerMovePrevious() error {
	return mediaPlayerCall((*interop.Runtime).MediaPlayerMovePrevious)
}

// mediaPlayerCall is the guard the five argument-free transport members share.
func mediaPlayerCall(call func(*interop.Runtime) error) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return call(runtime)
}

// mediaPlayerRuntime answers the loaded runtime or the Go-only refusal. Every
// member here needs one, because every route takes the game.
func mediaPlayerRuntime() (*interop.Runtime, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errMediaNoRuntime
	}
	return runtime, nil
}

// MediaPlayerIsShuffled is MediaPlayer::get_IsShuffled.
func MediaPlayerIsShuffled() (bool, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return false, err
	}
	return runtime.MediaPlayerIsShuffled()
}

// SetMediaPlayerIsShuffled is MediaPlayer::set_IsShuffled.
func SetMediaPlayerIsShuffled(value bool) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return runtime.MediaPlayerSetIsShuffled(value)
}

// MediaPlayerIsRepeating is MediaPlayer::get_IsRepeating.
func MediaPlayerIsRepeating() (bool, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return false, err
	}
	return runtime.MediaPlayerIsRepeating()
}

// SetMediaPlayerIsRepeating is MediaPlayer::set_IsRepeating.
func SetMediaPlayerIsRepeating(value bool) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return runtime.MediaPlayerSetIsRepeating(value)
}

// MediaPlayerIsMuted is MediaPlayer::get_IsMuted.
func MediaPlayerIsMuted() (bool, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return false, err
	}
	return runtime.MediaPlayerIsMuted()
}

// SetMediaPlayerIsMuted is MediaPlayer::set_IsMuted.
func SetMediaPlayerIsMuted(value bool) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return runtime.MediaPlayerSetIsMuted(value)
}

// MediaPlayerIsVisualizationEnabled is
// MediaPlayer::get_IsVisualizationEnabled.
func MediaPlayerIsVisualizationEnabled() (bool, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return false, err
	}
	return runtime.MediaPlayerIsVisualizationEnabled()
}

// SetMediaPlayerIsVisualizationEnabled is
// MediaPlayer::set_IsVisualizationEnabled.
func SetMediaPlayerIsVisualizationEnabled(value bool) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return runtime.MediaPlayerSetIsVisualizationEnabled(value)
}

// MediaPlayerVolume is MediaPlayer::get_Volume.
func MediaPlayerVolume() (float32, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return 0, err
	}
	return runtime.MediaPlayerVolume()
}

// SetMediaPlayerVolume is MediaPlayer::set_Volume.
func SetMediaPlayerVolume(value float32) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	return runtime.MediaPlayerSetVolume(value)
}

// MediaPlayerGameHasControl is MediaPlayer::get_GameHasControl, which reports
// whether the game owns the background music or the host does.
func MediaPlayerGameHasControl() (bool, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return false, err
	}
	return runtime.MediaPlayerGameHasControl()
}

// MediaPlayerState is MediaPlayer::get_State.
func MediaPlayerState() (MediaState, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return MediaStateStopped, err
	}
	value, err := runtime.MediaPlayerState()
	return MediaState(value), err
}

// MediaPlayerPlayPosition is MediaPlayer::get_PlayPosition. CNA reports TICKS,
// which is the CLR's own unit.
func MediaPlayerPlayPosition() (framework.TimeSpan, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtime.MediaPlayerPlayPositionTicks()
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}

// MediaPlayerQueue is MediaPlayer::get_Queue.
//
// The handle CNA answers is BORROWED, not owned: it is a view of the
// process-wide queue rather than an object the caller may destroy. So the
// projected MediaQueue holds it without a disposal latch, which is also what
// its contract says -- MediaQueue declares no Dispose and no IsDisposed.
func MediaPlayerQueue() (*MediaQueue, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return nil, err
	}
	handle, err := runtime.MediaPlayerQueue()
	if err != nil {
		return nil, err
	}
	return &MediaQueue{handle: handle}, nil
}

// MediaPlayerGetVisualizationData is
// MediaPlayer::GetVisualizationData(VisualizationData), which FILLS the object
// the caller hands it rather than returning one.
//
// It answers false when there is nothing to report -- the reference's own
// return is void and it leaves the object untouched, so the projection reports
// whether it wrote instead of leaving a caller unable to tell.
func MediaPlayerGetVisualizationData(data *VisualizationData) error {
	if data == nil {
		return mediaArgumentNullError("visualizationData")
	}
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	frequencies, samples, err := runtime.MediaPlayerVisualizationData()
	if err != nil {
		return err
	}
	data.setFrequencies(frequencies)
	data.setSamples(samples)
	return nil
}

// The two events, on the settled two-accessor projection: a CLR event's add
// and remove accessors are two Go members, and a subscription is a value the
// caller keeps in order to remove it.
//
// Both are STATIC, so both are package functions -- and both reach the SAME
// native registration pair, because CNA's callback carries only a context and
// the identity comes from which route was subscribed.

// MediaPlayerAddActiveSongChangedHandler is
// MediaPlayer::add_ActiveSongChanged.
func MediaPlayerAddActiveSongChangedHandler(
	handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return mediaPlayerAddHandler(mediaPlayerActiveSongChanged, handler)
}

// MediaPlayerRemoveActiveSongChangedHandler is
// MediaPlayer::remove_ActiveSongChanged.
func MediaPlayerRemoveActiveSongChangedHandler(subscription framework.EventSubscription) error {
	return mediaPlayerRemoveHandler(mediaPlayerActiveSongChanged, subscription)
}

// MediaPlayerAddMediaStateChangedHandler is
// MediaPlayer::add_MediaStateChanged.
func MediaPlayerAddMediaStateChangedHandler(
	handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return mediaPlayerAddHandler(mediaPlayerMediaStateChanged, handler)
}

// MediaPlayerRemoveMediaStateChangedHandler is
// MediaPlayer::remove_MediaStateChanged.
func MediaPlayerRemoveMediaStateChangedHandler(subscription framework.EventSubscription) error {
	return mediaPlayerRemoveHandler(mediaPlayerMediaStateChanged, subscription)
}
