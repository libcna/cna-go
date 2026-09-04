package audio

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// DynamicSoundEffectInstance is
// Microsoft.Xna.Framework.Audio.DynamicSoundEffectInstance:
//
//	.class public auto ansi sealed DynamicSoundEffectInstance
//	       extends Microsoft.Xna.Framework.Audio.SoundEffectInstance
//	  .field private static Dictionary`2<uint32,WeakReference> allInstances
//	  .field private AudioFormat format
//	  .field private EventHandler`1<EventArgs> BufferNeeded
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It is a streaming instance with NO source effect
//
// Every other SoundEffectInstance comes from a SoundEffect. This one has a
// public constructor and allocates its own voice, which is why CNA gives it a
// creation route that takes a GAME rather than an effect handle, and why the
// projection registers it with no parent resource.
//
// # It OVERRIDES four members of its base, and two of them REFUSE
//
// The two IsLooped accessors are the surprise, and they are 32 and 45 bytes of
// pure refusal:
//
//	get_IsLooped   if (IsDisposed) throw ObjectDisposedException(GetType().Name, ...);
//	               return false;                     // ALWAYS, `ldc.i4.0`
//
//	set_IsLooped   if (IsDisposed) throw ObjectDisposedException(...);
//	               if (value) throw new InvalidOperationException(
//	                              FrameworkResources.InvalidDynamicIsLoopedCall);
//	               // setting FALSE stores nothing and returns
//
// So a streaming instance can never loop: the getter always answers false and
// the setter accepts only the value the getter already reports. A consumer who
// sets false and reads it back sees no change because there was none to make.
//
// That is the composed-base virtual-dispatch problem in its plainest form, and
// it is why this type re-declares both accessors rather than inheriting them.
type DynamicSoundEffectInstance struct {
	// instance is the composed SoundEffectInstance, which carries the one
	// native handle and every inherited member.
	instance *SoundEffectInstance
	// sampleRate and channels are what `format` holds in the reference. They
	// are kept because the two sample conversions need them and because the
	// constructor validated them.
	sampleRate int32
	channels   AudioChannels
	// bufferNeededEvent is the `BufferNeeded` delegate field.
	bufferNeededEvent framework.EventSource[*framework.EventArgs]
}

// errDynamicSoundEffectInstanceNil is the Go-only guard for a zero value.
var errDynamicSoundEffectInstanceNil = errors.New("DynamicSoundEffectInstance is nil or uninitialized")

// NewDynamicSoundEffectInstance is
// DynamicSoundEffectInstance::.ctor(Int32, AudioChannels), 74 bytes:
//
//	base();                                          // the BASE runs FIRST
//	if (sampleRate < 8000 || sampleRate > 48000)
//	    throw new ArgumentOutOfRangeException("sampleRate");
//	if (channels < 1 || channels > 2)
//	    throw new ArgumentOutOfRangeException("channels");
//	format = AudioFormat.Create(sampleRate, channels, 16);
//	AllocateVoice();
//
// The base constructor runs BEFORE either guard, so a refused construction has
// already run the base's own initialisation. That ordering is the reference's
// and is why the projection builds the composed instance only after the guards
// pass -- a Go constructor that returns an error has no half-built object to
// leave behind, which is a narrowing in the direction that cannot surprise.
//
// The two ranges are the same 8000..48000 and 1..2 that SoundEffect's
// FromBuffer checks, and both throw with a PARAMETER NAME and no message.
func NewDynamicSoundEffectInstance(
	sampleRate int32, channels AudioChannels,
) (*DynamicSoundEffectInstance, error) {
	if err := checkAudioSampleRate(sampleRate); err != nil {
		return nil, err
	}
	if err := checkAudioChannels(channels); err != nil {
		return nil, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errSoundEffectNoRunningGame
	}
	resource, err := runtime.CreateDynamicSoundEffectInstance(sampleRate, uint32(channels))
	if err != nil {
		return nil, err
	}
	base := newSoundEffectInstance(nil, resource)
	dynamic := &DynamicSoundEffectInstance{
		instance:   base,
		sampleRate: sampleRate,
		channels:   channels,
	}
	base.bindDerived(dynamic)
	return dynamic, nil
}

// clrTypeName is GetType().Name's answer, which every disposal refusal in the
// family carries.
func (d *DynamicSoundEffectInstance) clrTypeName() string {
	return "DynamicSoundEffectInstance"
}

// IsLooped is DynamicSoundEffectInstance::get_IsLooped, 32 bytes, and it
// ALWAYS answers false. See the type's own note.
//
// It is FALLIBLE because the reference's disposal check throws, and infallible
// otherwise -- the `return false` reaches nothing.
func (d *DynamicSoundEffectInstance) IsLooped() (bool, error) {
	if d == nil || d.instance == nil {
		return false, errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return false, d.instance.objectDisposed()
	}
	return false, nil
}

// SetIsLooped is DynamicSoundEffectInstance::set_IsLooped, 45 bytes, which
// accepts only the value the getter already reports.
func (d *DynamicSoundEffectInstance) SetIsLooped(value bool) error {
	if d == nil || d.instance == nil {
		return errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return d.instance.objectDisposed()
	}
	if value {
		return fmt.Errorf("%w: %s", errAudioInvalidOperation, invalidDynamicIsLoopedCall)
	}
	// Setting FALSE stores nothing: the reference's body returns here.
	return nil
}

// PendingBufferCount is DynamicSoundEffectInstance::get_PendingBufferCount:
//
//	lock (VoiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    return GetPendingBufferCount();
//	}
//
// A LIVE native read, unlike every cached scalar on the base.
func (d *DynamicSoundEffectInstance) PendingBufferCount() (int32, error) {
	if d == nil || d.instance == nil {
		return 0, errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return 0, d.instance.objectDisposed()
	}
	if d.instance.resource == nil {
		return 0, errDynamicSoundEffectInstanceNil
	}
	return d.instance.resource.DynamicSoundInstancePendingBufferCount()
}

// Play is DynamicSoundEffectInstance::Play(), which OVERRIDES the base's:
//
//	lock (VoiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    Play(VoiceHandle);
//	    isPacketSubmitted = true;
//	}
//
// The base's Play does the same two things in the same order, so the override
// exists in the reference to skip the base's fire-and-forget bookkeeping rather
// than to change what a caller sees. The projection routes to the base for that
// reason: one body, and the packet flag set on success either way.
func (d *DynamicSoundEffectInstance) Play() error {
	if d == nil || d.instance == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.Play()
}

// SubmitBufferBySliceOfByte is
// DynamicSoundEffectInstance::SubmitBuffer(Byte[]), 12 bytes:
// `SubmitBuffer(buffer, 0, buffer.Length)`.
func (d *DynamicSoundEffectInstance) SubmitBufferBySliceOfByte(buffer []uint8) error {
	return d.SubmitBufferBySliceOfByteAndInt32AndInt32(buffer, 0, int32(len(buffer)))
}

// SubmitBufferBySliceOfByteAndInt32AndInt32 is
// DynamicSoundEffectInstance::SubmitBuffer(Byte[], Int32, Int32), whose four
// checks are SoundEffect::FromBuffer's four, with the same four messages:
//
//	buffer null, empty or MISALIGNED             -> InvalidAudioBuffer
//	offset negative, past the end or MISALIGNED  -> InvalidAudioBufferOffset
//	offset+count overflow                        -> InvalidOffsetCountLength
//	count <= 0, past the end, or MISALIGNED      -> InvalidOffsetCountLength
//
// `IsAligned` is `v % BlockAlign == 0` and BlockAlign is `channels * 2`, so a
// submitted buffer must be a whole number of frames -- the same rule the
// constructor applies to a SoundEffect's PCM.
func (d *DynamicSoundEffectInstance) SubmitBufferBySliceOfByteAndInt32AndInt32(
	buffer []uint8, offset, count int32,
) error {
	if d == nil || d.instance == nil {
		return errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return d.instance.objectDisposed()
	}
	if err := checkAudioBufferWindow(buffer, offset, count, int32(d.channels)); err != nil {
		return err
	}
	if d.instance.resource == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.resource.DynamicSoundInstanceSubmitBuffer(buffer, offset, count)
}

// GetSampleDuration is
// DynamicSoundEffectInstance::GetSampleDuration(Int32), which reads the
// instance's OWN `format` field:
//
//	lock (VoiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    if (sizeInBytes < 0)
//	        throw new ArgumentOutOfRangeException("sizeInBytes", InvalidBufferSize);
//	    return format.DurationFromSize(sizeInBytes);
//	}
//
// # It is MANAGED, and the measurement is why
//
// CNA publishes cna_dynamic_sound_effect_instance_get_sample_size_in_bytes and
// its sibling, and this milestone bound both and then reverted them. The native
// scenario asserted the number the reference produces and CNA answered a
// different one:
//
//	one second at 22050Hz mono
//	    the reference  44098 bytes   (float32 scale factor, 22049 samples)
//	    CNA            44100 bytes   (the exact arithmetic)
//
// Both are "one second at 22050Hz mono" and they differ by one frame, because
// XNA computes its scale factor in float32 and CNA does not. The reference's
// answer is the contract, so the conversion is the same managed arithmetic
// SoundEffect's statics use and the two CNA routes are recorded unbound.
func (d *DynamicSoundEffectInstance) GetSampleDuration(sizeInBytes int32) (framework.TimeSpan, error) {
	if d == nil || d.instance == nil {
		return framework.TimeSpan{}, errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return framework.TimeSpan{}, d.instance.objectDisposed()
	}
	if sizeInBytes < 0 {
		return framework.TimeSpan{}, argumentOutOfRangeError("sizeInBytes", invalidBufferSize)
	}
	return durationFromSize(sizeInBytes, d.sampleRate, int32(d.channels))
}

// GetSampleSizeInBytes is
// DynamicSoundEffectInstance::GetSampleSizeInBytes(TimeSpan).
func (d *DynamicSoundEffectInstance) GetSampleSizeInBytes(duration framework.TimeSpan) (int32, error) {
	if d == nil || d.instance == nil {
		return 0, errDynamicSoundEffectInstanceNil
	}
	if d.instance.disposed {
		return 0, d.instance.objectDisposed()
	}
	total := duration.TotalMilliseconds()
	if !(total >= 0) || !(total <= 2147483647) {
		return 0, argumentOutOfRangeError("duration", "")
	}
	size, err := sizeFromDuration(duration, d.sampleRate, int32(d.channels))
	if err != nil {
		// The reference catches the OverflowException its checked arithmetic
		// raises and rethrows it naming the DURATION.
		return 0, argumentOutOfRangeError("duration", "")
	}
	return size, nil
}

// AddBufferNeededHandler is add_BufferNeeded, on the settled two-accessor event
// projection.
//
// # The event's raise site is CNA's, and the projection does not reach it
//
// The reference raises BufferNeeded from its streaming callback when a
// submitted buffer has been consumed. CNA has the counterpart --
// `cna_dynamic_sound_effect_instance_subscribe_buffer_needed` -- and binding it
// would mean holding a Go callback across a native boundary for the instance's
// lifetime, which is the shape every other CNA event in this project is
// deliberately not bound in.
//
// So the accessors are projected because the contract declares them and the
// registration list is real, and the route is recorded unbound. What a consumer
// can do instead is poll PendingBufferCount, which is a live read and answers
// the same question the event announces.
func (d *DynamicSoundEffectInstance) AddBufferNeededHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if d == nil {
		return framework.EventSubscription{}, errDynamicSoundEffectInstanceNil
	}
	return d.bufferNeededEvent.Add(handler)
}

// RemoveBufferNeededHandler is remove_BufferNeeded.
func (d *DynamicSoundEffectInstance) RemoveBufferNeededHandler(subscription framework.EventSubscription) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.bufferNeededEvent.Remove(subscription)
}

// ---------------------------------------------------------------------------
// The inherited public surface of SoundEffectInstance, forwarded. IsLooped is
// absent because this type OVERRIDES both its accessors above.
// ---------------------------------------------------------------------------

// Pause is SoundEffectInstance::Pause().
func (d *DynamicSoundEffectInstance) Pause() error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.Pause()
}

// Resume is SoundEffectInstance::Resume().
func (d *DynamicSoundEffectInstance) Resume() error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.Resume()
}

// StopByNone is SoundEffectInstance::Stop().
func (d *DynamicSoundEffectInstance) StopByNone() error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.StopByNone()
}

// StopByBoolean is SoundEffectInstance::Stop(Boolean).
func (d *DynamicSoundEffectInstance) StopByBoolean(immediate bool) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.StopByBoolean(immediate)
}

// State is SoundEffectInstance::get_State.
func (d *DynamicSoundEffectInstance) State() (SoundState, error) {
	if d == nil {
		return SoundStateStopped, errDynamicSoundEffectInstanceNil
	}
	return d.instance.State()
}

// IsDisposed is SoundEffectInstance::get_IsDisposed.
func (d *DynamicSoundEffectInstance) IsDisposed() bool {
	if d == nil {
		return true
	}
	return d.instance.IsDisposed()
}

// Volume is SoundEffectInstance::get_Volume.
func (d *DynamicSoundEffectInstance) Volume() float32 {
	if d == nil {
		return 0
	}
	return d.instance.Volume()
}

// SetVolume is SoundEffectInstance::set_Volume.
func (d *DynamicSoundEffectInstance) SetVolume(value float32) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.SetVolume(value)
}

// Pitch is SoundEffectInstance::get_Pitch.
func (d *DynamicSoundEffectInstance) Pitch() float32 {
	if d == nil {
		return 0
	}
	return d.instance.Pitch()
}

// SetPitch is SoundEffectInstance::set_Pitch.
func (d *DynamicSoundEffectInstance) SetPitch(value float32) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.SetPitch(value)
}

// Pan is SoundEffectInstance::get_Pan.
func (d *DynamicSoundEffectInstance) Pan() float32 {
	if d == nil {
		return 0
	}
	return d.instance.Pan()
}

// SetPan is SoundEffectInstance::set_Pan.
func (d *DynamicSoundEffectInstance) SetPan(value float32) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.SetPan(value)
}

// Apply3DByAudioListenerAndAudioEmitter is
// SoundEffectInstance::Apply3D(AudioListener, AudioEmitter).
func (d *DynamicSoundEffectInstance) Apply3DByAudioListenerAndAudioEmitter(listener *AudioListener, emitter *AudioEmitter) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.Apply3DByAudioListenerAndAudioEmitter(listener, emitter)
}

// Apply3DBySliceOfAudioListenerAndAudioEmitter is
// SoundEffectInstance::Apply3D(AudioListener[], AudioEmitter).
func (d *DynamicSoundEffectInstance) Apply3DBySliceOfAudioListenerAndAudioEmitter(listeners []*AudioListener, emitter *AudioEmitter) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.Apply3DBySliceOfAudioListenerAndAudioEmitter(listeners, emitter)
}

// DisposeByNone is SoundEffectInstance::Dispose(), inherited unchanged. The
// derived type declares the PROTECTED overload and not this one, so both appear
// and the overload suffixes distinguish them -- the same split Effect and its
// stock effects have.
func (d *DynamicSoundEffectInstance) DisposeByNone() error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.DisposeByNone()
}

// DisposeByBoolean is DynamicSoundEffectInstance::Dispose(Boolean), which the
// reference OVERRIDES to remove itself from the static `allInstances` map
// before reaching the base's teardown. That map is private bookkeeping for the
// reference's own streaming pump, which CNA performs itself, so the override
// adds nothing here and forwards.
func (d *DynamicSoundEffectInstance) DisposeByBoolean(disposing bool) error {
	if d == nil {
		return errDynamicSoundEffectInstanceNil
	}
	return d.instance.DisposeByBoolean(disposing)
}

// Finalize is deliberately ABSENT. SoundEffectInstance declares it `family`,
// so it is not part of the inherited PUBLIC surface a derived type carries --
// the same enumeration rule that gives the stock effects one Dispose and Effect
// two.
