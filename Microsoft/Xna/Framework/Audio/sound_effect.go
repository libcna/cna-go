package audio

import (
	"errors"
	"fmt"
	"io"
	"math"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/dispatcher"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SoundEffect is Microsoft.Xna.Framework.Audio.SoundEffect:
//
//	.class public auto ansi sealed SoundEffect
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_sound_effect_destroy.
//
// Its INSTANCES are registered as child resources of the effect, which makes
// CNA's documented ordering -- "The returned effect must be destroyed after all
// instances created from it and before the game" -- a structural property
// rather than a rule a caller has to remember.
//
// # Four of its members never reach CNA
//
// `get_IsDisposed`, `get_Duration`, `get_Name` and `set_Name` are `ldfld` and
// `stfld` over managed fields the constructor seeds. CNA publishes a route for
// each, and each is recorded as unbound: reading CNA back would be a second
// answer to a question the managed field already holds, and `get_Name` in the
// reference has no disposal check at all, so it answers after Dispose where a
// native read could not.
//
// `Duration` is the exception that proves the rule. The reference stores it at
// construction, but what it stores is derived from the buffer the RENDERER
// received, so the projection reads CNA's ticks once at construction and stores
// those. That is the same choice VertexBuffer made for its stride: record what
// the backend built, not what was asked for.
type SoundEffect struct {
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the reference's own `disposed` field, which get_IsDisposed
	// reads directly.
	disposed bool
	// effectName is `effectName`, seeded with String.Empty by every
	// constructor. set_Name refuses the empty string, so the initial value is
	// unreachable through the setter.
	effectName string
	// duration is `duration`, read from CNA once at construction.
	duration framework.TimeSpan
	// children is the reference's `List<WeakReference>` of instances, kept as
	// direct references because Go has no weak reference and because Dispose
	// must reach every one of them. See Dispose for what that costs.
	children []*SoundEffectInstance
}

// errSoundEffectNil is the Go-only guard for a zero value.
var errSoundEffectNil = errors.New("SoundEffect is nil or uninitialized")

// NewSoundEffectBySliceOfByteAndInt32AndAudioChannels is
// SoundEffect::.ctor(Byte[], Int32, AudioChannels), 92 bytes:
//
//	effectName = String.Empty; syncObject = new object(); handle = -1;
//	instancePool = new Stack<SoundEffectInstance>();
//	children = new List<WeakReference>();
//	base();
//	if (buffer == null || buffer.Length == 0)
//	    throw new ArgumentException(FrameworkResources.InvalidAudioBuffer);
//	FromBuffer(buffer, 0, buffer.Length, sampleRate, channels, 0, 0);
//
// The buffer check here runs BEFORE FromBuffer's own checks, so this overload
// reports the BUFFER where the seven-argument one reports the SAMPLE RATE for
// the same pair of bad arguments. That divergence is the reference's and is
// pinned by a test.
//
// Note the message: `ArgumentException` with a sentence and NO parameter name,
// not `ArgumentNullException`.
func NewSoundEffectBySliceOfByteAndInt32AndAudioChannels(
	buffer []uint8, sampleRate int32, channels AudioChannels,
) (*SoundEffect, error) {
	if len(buffer) == 0 {
		return nil, argumentError(invalidAudioBuffer)
	}
	return newSoundEffectFromBuffer(buffer, 0, int32(len(buffer)), sampleRate, channels, 0, 0)
}

// NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32
// is SoundEffect::.ctor(Byte[], Int32, Int32, Int32, AudioChannels, Int32,
// Int32), 75 bytes, which is the field prologue and then FromBuffer with the
// caller's own arguments -- no check of its own at all.
func NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
	buffer []uint8, offset, count, sampleRate int32, channels AudioChannels, loopStart, loopLength int32,
) (*SoundEffect, error) {
	return newSoundEffectFromBuffer(buffer, offset, count, sampleRate, channels, loopStart, loopLength)
}

// newSoundEffectFromBuffer is SoundEffect::FromBuffer, whose seven checks are
// the whole managed failure surface of both constructors. The order is the
// reference's and each message is its own:
//
//	sampleRate range                              -> AOORE("sampleRate")
//	channels range                                -> AOORE("channels")
//	buffer null, empty or MISALIGNED              -> InvalidAudioBuffer
//	offset negative, past the end or MISALIGNED   -> InvalidAudioBufferOffset
//	offset+count overflow, past the end,
//	    count <= 0, or count MISALIGNED           -> InvalidOffsetCountLength
//	loopStart+loopLength overflow, either
//	    negative, or the end past the frame count -> InvalidLoopRegion
//
// `IsAligned(v)` is `v % BlockAlign == 0` and BlockAlign is `channels * 2`, so
// a buffer, an offset and a count must each be a whole number of frames. Three
// separate tests with three separate messages, which is what makes them worth
// writing out.
func newSoundEffectFromBuffer(
	buffer []uint8, offset, count, sampleRate int32, channels AudioChannels, loopStart, loopLength int32,
) (*SoundEffect, error) {
	if err := checkAudioSampleRate(sampleRate); err != nil {
		return nil, err
	}
	if err := checkAudioChannels(channels); err != nil {
		return nil, err
	}
	blockAlign := audioBlockAlign(int32(channels))
	length := int32(len(buffer))
	if len(buffer) == 0 || length%blockAlign != 0 {
		return nil, argumentError(invalidAudioBuffer)
	}
	if offset < 0 || offset >= length || offset%blockAlign != 0 {
		return nil, argumentError(invalidAudioBufferOffset)
	}
	// `checked(offset + count)` -- the reference catches the OverflowException
	// and reports it with the SAME message the range test uses.
	total := int64(offset) + int64(count)
	if total > math.MaxInt32 || total < math.MinInt32 {
		return nil, argumentError(invalidOffsetCountLength)
	}
	if int32(total) > length || count <= 0 || count%blockAlign != 0 {
		return nil, argumentError(invalidOffsetCountLength)
	}
	loopEnd := int64(loopStart) + int64(loopLength)
	if loopEnd > math.MaxInt32 || loopEnd < math.MinInt32 {
		return nil, argumentError(invalidLoopRegion)
	}
	frames := count / blockAlign
	if loopStart < 0 || loopLength < 0 || int32(loopEnd) > frames {
		return nil, argumentError(invalidLoopRegion)
	}
	// A zero loopLength is NOT "no loop": the reference rewrites it to the
	// whole range. CNA documents the same rule from the other side -- "zero
	// loops the whole range" -- so the caller's zero crosses unchanged and the
	// rewrite happens once, on the side that acts on it.
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errSoundEffectNoRunningGame
	}
	resource, err := runtime.CreateSoundEffectFromPCM16(uint32(sampleRate), uint32(channels),
		buffer, offset, count, loopStart, loopLength)
	if err != nil {
		return nil, err
	}
	return newSoundEffect(resource)
}

// SoundEffectFromStream is SoundEffect::FromStream(Stream), 22 bytes:
//
//	if (stream == null) throw new ArgumentNullException("stream");
//	return new SoundEffect(stream);
//
// # A measured WIDENING, recorded rather than hidden
//
// The private constructor it calls opens a `WavFile`, so the reference accepts
// WAV and nothing else on this path. CNA's route is wider: "Whatever the audio
// backend can decode is accepted, which is more than the raw PCM the other
// creation routes take."
//
// So a consumer handing this a WAV gets the reference's behaviour, and one
// handing it an Ogg gets an effect where the reference would have thrown. That
// is a widening in the direction that accepts more, which cannot break working
// code, and it is stated here rather than papered over.
func SoundEffectFromStream(stream io.Reader) (*SoundEffect, error) {
	if stream == nil {
		return nil, argumentNullError("stream")
	}
	encoded, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 {
		return nil, argumentError(invalidAudioBuffer)
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errSoundEffectNoRunningGame
	}
	resource, err := runtime.CreateSoundEffectFromEncoded(encoded)
	if err != nil {
		return nil, err
	}
	return newSoundEffect(resource)
}

// newSoundEffect is the tail every creation path shares: it reads the duration
// CNA computed and seeds the managed fields the reference's constructors seed.
func newSoundEffect(resource *interop.Resource) (*SoundEffect, error) {
	ticks, err := resource.SoundEffectDurationTicks()
	if err != nil {
		_ = resource.Dispose()
		return nil, err
	}
	return &SoundEffect{
		resource: resource,
		// String.Empty, which set_Name would refuse.
		effectName: "",
		duration:   framework.TimeSpanFromTicks(ticks),
	}, nil
}

// IsDisposed is SoundEffect::get_IsDisposed, one `ldfld` over the managed flag.
func (e *SoundEffect) IsDisposed() bool {
	if e == nil {
		return true
	}
	return e.disposed
}

// Duration is SoundEffect::get_Duration, one `ldfld`. It answers AFTER disposal,
// because the reference's getter has no disposal check.
func (e *SoundEffect) Duration() framework.TimeSpan {
	if e == nil {
		return framework.TimeSpan{}
	}
	return e.duration
}

// Name is SoundEffect::get_Name, one `ldfld` with NO disposal check -- so a
// disposed effect still answers its name.
func (e *SoundEffect) Name() string {
	if e == nil {
		return ""
	}
	return e.effectName
}

// SetName is SoundEffect::set_Name, 57 bytes:
//
//	if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	if (String.IsNullOrEmpty(value)) throw new ArgumentNullException("value");
//	effectName = value;
//
// The guard is IsNullOrEmpty, so the EMPTY string is refused with
// ArgumentNullException -- and the constructor seeds the field with exactly
// that value, so the initial name cannot be set again through this member.
func (e *SoundEffect) SetName(value string) error {
	if e == nil {
		return errSoundEffectNil
	}
	if e.disposed {
		return e.objectDisposed()
	}
	if value == "" {
		return argumentNullError("value")
	}
	e.effectName = value
	return nil
}

// CreateInstance is SoundEffect::CreateInstance, 121 bytes:
//
//	lock (syncObject) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    instance = new SoundEffectInstance(this, false);
//	    lock (children) { children.Add(new WeakReference(instance)); }
//	    return instance;
//	}
//
// `false` is isFireAndForget: Play's path builds a pooled instance with true,
// and that flag is the only difference between the two.
func (e *SoundEffect) CreateInstance() (*SoundEffectInstance, error) {
	if e == nil {
		return nil, errSoundEffectNil
	}
	if e.disposed {
		return nil, e.objectDisposed()
	}
	// The nil-resource guard sits immediately before the native call rather
	// than with the receiver: the reference has no such concept, so answering
	// it earlier would answer where the reference answers a disposal refusal.
	if e.resource == nil {
		return nil, errSoundEffectNil
	}
	resource, err := e.resource.CreateSoundEffectInstance()
	if err != nil {
		return nil, err
	}
	instance := newSoundEffectInstance(e, resource)
	e.children = append(e.children, instance)
	return instance, nil
}

// PlayByNone is SoundEffect::Play(), 22 bytes: `Play(1, 0, 0)`.
func (e *SoundEffect) PlayByNone() (bool, error) {
	return e.PlayBySingleAndSingleAndSingle(1, 0, 0)
}

// PlayBySingleAndSingleAndSingle is SoundEffect::Play(Single, Single, Single),
// 270 bytes, and TWO things about it are worth stating before the body.
//
// # It refuses until FrameworkDispatcher.Update has run
//
// The FIRST statement, before any lock and before the disposal check:
//
//	if (!FrameworkDispatcher.UpdateCalledAtLeastOnce)
//	    throw new InvalidOperationException(String.Format(CultureInfo.CurrentCulture,
//	        FrameworkResources.CallFrameworkDispatcherUpdate));
//
// The flag is `assembly` on FrameworkDispatcher and not in the pinned contract,
// so it is tracked in this module rather than exposed: see
// internal/dispatcher. A fire-and-forget sound needs the
// dispatcher because the dispatcher is what reclaims its instance.
//
// # The bool is NOT "did it make a sound"
//
//	try { ... start a pooled instance ...; return true; }
//	catch (InstancePlayLimitException) { pool it back; return false; }
//
// False means the voice limit was hit and nothing else. CNA answers
// `CNA_RESULT_INVALID_STATE` "when too many instances are already playing",
// which is the same condition, so that ONE result maps to a false return and
// every other failure stays an error.
func (e *SoundEffect) PlayBySingleAndSingleAndSingle(volume, pitch, pan float32) (bool, error) {
	if e == nil {
		return false, errSoundEffectNil
	}
	if !dispatcher.HasRun() {
		return false, fmt.Errorf("%w: %s", errAudioInvalidOperation, callFrameworkDispatcherUpdate)
	}
	if e.disposed {
		return false, e.objectDisposed()
	}
	if e.resource == nil {
		return false, errSoundEffectNil
	}
	played, err := e.resource.SoundEffectPlayWithSettings(volume, pitch, pan)
	if interop.IsInvalidState(err) {
		// The voice limit, which the reference catches as
		// InstancePlayLimitException and converts into a false return.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return played, nil
}

// Dispose is SoundEffect::Dispose(), which the reference implements as
// `Dispose(true); GC.SuppressFinalize(this)`.
//
// The private Dispose(bool) stops and releases every instance the effect
// created before releasing the effect, which is what CNA's ordering rule
// requires too. The projection walks its own children rather than a
// WeakReference list: Go has no weak reference, so an instance the consumer has
// dropped is still reachable here, and the difference is that this projection
// keeps instances alive until their effect is disposed where the reference lets
// the collector take them first. That is a lifetime widening with no observable
// difference through the contract -- an instance the consumer cannot reach
// cannot be asked anything -- and it is what makes the release ordering
// deterministic.
func (e *SoundEffect) Dispose() error {
	if e == nil {
		return errSoundEffectNil
	}
	if e.disposed {
		// Disposal is idempotent, as the reference's flag makes it.
		return nil
	}
	e.disposed = true
	var failures []error
	for _, instance := range e.children {
		if err := instance.disposeFromEffect(); err != nil {
			failures = append(failures, err)
		}
	}
	e.children = nil
	if e.resource != nil {
		if err := e.resource.Dispose(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// Finalize is SoundEffect::Finalize, the protected finalizer, which the
// reference implements as `Dispose(false)`.
//
// Nothing calls it: Go has no CLR finalization and CNA-Go registers no runtime
// finalizer. It is projected because the pinned contract declares it, and it
// reaches the same teardown Dispose does -- the reference's Dispose(bool)
// releases the native object on both branches and differs only in whether it
// touches other managed objects.
func (e *SoundEffect) Finalize() error {
	return e.Dispose()
}

// objectDisposed is Helpers' ObjectDisposedException(GetType().Name, message).
//
// The type name is a LITERAL here and not an identity resolution, because
// SoundEffect is `sealed` in the reference: no derived type can exist, so
// `GetType().Name` has exactly one possible answer. SoundEffectInstance's is a
// different matter -- see its own note.
func (e *SoundEffect) objectDisposed() error {
	return fmt.Errorf("%w: SoundEffect: %s", errAudioObjectDisposed, objectDisposedMessage)
}
