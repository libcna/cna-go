package media

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// Video is Microsoft.Xna.Framework.Media.Video: five read-only properties over
// a decoded video asset.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # A consumer CANNOT obtain one, and that is measured
//
// The contract declares no constructor and no static factory. In XNA a Video
// arrives from `ContentManager.Load<Video>`, and that specialization cannot be
// bound: CNA has load routes for textures, effects, sprite fonts, sound
// effects, texture cubes and models, and NONE for video.
//
// CNA can build one -- `cna_video_create(graphics_device, file_name, ...)` --
// but exposing that would be inventing a constructor the contract does not
// declare, which is the same line the Model family holds.
//
// So the type is complete and unreachable, exactly as Model is, and the reason
// is recorded here rather than papered over with a factory nothing measured.
//
// It also has NO disposal: the contract declares no Dispose, no Finalize and no
// IsDisposed, so the projection has no latch. The handle belongs to whatever
// produced it.
type Video struct {
	handle uint64
}

// Duration is Video::get_Duration.
func (v *Video) Duration() (framework.TimeSpan, error) {
	runtime, err := v.usableVideo()
	if err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtime.VideoDurationTicks(v.handle)
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}

// Width is Video::get_Width.
func (v *Video) Width() (int32, error) {
	runtime, err := v.usableVideo()
	if err != nil {
		return 0, err
	}
	return runtime.VideoWidth(v.handle)
}

// Height is Video::get_Height.
func (v *Video) Height() (int32, error) {
	runtime, err := v.usableVideo()
	if err != nil {
		return 0, err
	}
	return runtime.VideoHeight(v.handle)
}

// FramesPerSecond is Video::get_FramesPerSecond.
func (v *Video) FramesPerSecond() (float32, error) {
	runtime, err := v.usableVideo()
	if err != nil {
		return 0, err
	}
	return runtime.VideoFramesPerSecond(v.handle)
}

// VideoSoundtrackType is Video::get_VideoSoundtrackType.
func (v *Video) VideoSoundtrackType() (VideoSoundtrackType, error) {
	runtime, err := v.usableVideo()
	if err != nil {
		return VideoSoundtrackTypeMusic, err
	}
	value, err := runtime.VideoSoundtrackType(v.handle)
	return VideoSoundtrackType(value), err
}

func (v *Video) usableVideo() (*interop.Runtime, error) {
	if v == nil {
		return nil, mediaArgumentNullError("video")
	}
	return mediaPlayerRuntime()
}

// VideoPlayer is Microsoft.Xna.Framework.Media.VideoPlayer.
//
// Unlike Video it DOES declare a constructor, so a consumer can build one --
// and then cannot give it anything to play, for the reason recorded on Video.
// Every member below works on a player with no video; Play is the one that
// needs one.
type VideoPlayer struct {
	handle   uint64
	disposed bool
}

// NewVideoPlayer is VideoPlayer::.ctor().
func NewVideoPlayer() (*VideoPlayer, error) {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return nil, err
	}
	handle, err := runtime.VideoPlayerCreate()
	if err != nil {
		return nil, err
	}
	return &VideoPlayer{handle: handle}, nil
}

// Dispose is VideoPlayer::Dispose(), latching before the native call.
func (p *VideoPlayer) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.VideoPlayerDispose(p.handle); err != nil {
		return err
	}
	return runtime.VideoPlayerDestroy(p.handle)
}

// Finalize is VideoPlayer::Finalize, the protected finalizer.
func (p *VideoPlayer) Finalize() error { return p.Dispose() }

// IsDisposed is VideoPlayer::get_IsDisposed, read from the managed latch.
func (p *VideoPlayer) IsDisposed() bool { return p == nil || p.disposed }

// Play is VideoPlayer::Play(Video).
func (p *VideoPlayer) Play(video *Video) error {
	// DISPOSAL first, because the reference's ThrowIfDisposed opens every
	// instance member; then the ARGUMENT, before the runtime.
	if p == nil || p.disposed {
		return errMediaDisposed
	}
	if video == nil {
		return mediaArgumentNullError("video")
	}
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerPlay(p.handle, video.handle)
}

// Pause is VideoPlayer::Pause().
func (p *VideoPlayer) Pause() error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerPause(p.handle)
}

// Resume is VideoPlayer::Resume().
func (p *VideoPlayer) Resume() error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerResume(p.handle)
}

// Stop is VideoPlayer::Stop().
func (p *VideoPlayer) Stop() error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerStop(p.handle)
}

// GetTexture is VideoPlayer::GetTexture(), the current frame.
//
// A player that has never been given a video has no frame, which CNA reports
// as an availability flag. The reference throws there; this answers nil,
// because a Go caller can test a nil and the alternative would be inventing
// which exception it threw.
func (p *VideoPlayer) GetTexture() (*graphics.Texture2D, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return nil, err
	}
	resource, info, available, err := runtime.VideoPlayerFrame(p.handle)
	if err != nil || !available {
		return nil, err
	}
	frame, err := servicebridge.AdoptVideoFrame(resource, info)
	if err != nil {
		return nil, err
	}
	texture, ok := frame.(*graphics.Texture2D)
	if !ok {
		return nil, errMediaNoRuntime
	}
	return texture, nil
}

// Video is VideoPlayer::get_Video.
func (p *VideoPlayer) Video() (*Video, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return nil, err
	}
	handle, available, err := runtime.VideoPlayerVideo(p.handle)
	if err != nil || !available {
		return nil, err
	}
	return &Video{handle: handle}, nil
}

// State is VideoPlayer::get_State.
func (p *VideoPlayer) State() (MediaState, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return MediaStateStopped, err
	}
	value, err := runtime.VideoPlayerState(p.handle)
	return MediaState(value), err
}

// IsLooped is VideoPlayer::get_IsLooped.
func (p *VideoPlayer) IsLooped() (bool, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return false, err
	}
	return runtime.VideoPlayerIsLooped(p.handle)
}

// SetIsLooped is VideoPlayer::set_IsLooped.
func (p *VideoPlayer) SetIsLooped(value bool) error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerSetIsLooped(p.handle, value)
}

// IsMuted is VideoPlayer::get_IsMuted.
func (p *VideoPlayer) IsMuted() (bool, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return false, err
	}
	return runtime.VideoPlayerIsMuted(p.handle)
}

// SetIsMuted is VideoPlayer::set_IsMuted.
func (p *VideoPlayer) SetIsMuted(value bool) error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerSetIsMuted(p.handle, value)
}

// Volume is VideoPlayer::get_Volume.
func (p *VideoPlayer) Volume() (float32, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return 0, err
	}
	return runtime.VideoPlayerVolume(p.handle)
}

// SetVolume is VideoPlayer::set_Volume.
func (p *VideoPlayer) SetVolume(value float32) error {
	runtime, err := p.usablePlayer()
	if err != nil {
		return err
	}
	return runtime.VideoPlayerSetVolume(p.handle, value)
}

// PlayPosition is VideoPlayer::get_PlayPosition.
func (p *VideoPlayer) PlayPosition() (framework.TimeSpan, error) {
	runtime, err := p.usablePlayer()
	if err != nil {
		return framework.TimeSpan{}, err
	}
	ticks, err := runtime.VideoPlayerPlayPositionTicks(p.handle)
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return framework.TimeSpanFromTicks(ticks), nil
}

// usablePlayer is the guard every member shares: the managed latch first, then
// the runtime.
func (p *VideoPlayer) usablePlayer() (*interop.Runtime, error) {
	if p == nil || p.disposed {
		return nil, errMediaDisposed
	}
	return mediaPlayerRuntime()
}
