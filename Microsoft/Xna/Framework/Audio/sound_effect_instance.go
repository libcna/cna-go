package audio

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SoundEffectInstance is Microsoft.Xna.Framework.Audio.SoundEffectInstance:
//
//	.class public auto ansi beforefieldinit SoundEffectInstance
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//	  .field private SoundEffect effect
//	  .field private bool disposed, looped, is3d, isFireAndForget
//	  .field private float32 currentVolume, currentPitch, currentPan
//	  .field assembly bool isPacketSubmitted
//	  .field private uint32 voiceHandle
//	  .field private object voiceHandleLock
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It has no public constructor
//
// The pinned contract declares none: an instance comes from
// `SoundEffect.CreateInstance` or from `SoundEffect.Play`'s pool. So this type
// and SoundEffect had to land in the same milestone.
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_sound_effect_instance_destroy,
//	and registered as a CHILD of its effect.
//
// CNA states the ordering: "The returned effect must be destroyed after all
// instances created from it and before the game." Making the instance a child
// resource is what turns that into a structural property.
//
// # Every member locks, and every member checks disposal first
//
// The shape is uniform and worth stating once:
//
//	lock (voiceHandleLock) {
//	    if (IsDisposed)
//	        throw new ObjectDisposedException(GetType().Name, FrameworkResources.ObjectDisposedException);
//	    ... the member's own work ...
//	}
//
// The Go projection needs no lock -- a *SoundEffectInstance is not shared
// across goroutines by anything this module does, and CNA's own routes carry
// their thread affinity -- but the disposal check is reproduced at every one of
// them, because it is the reference's observable behaviour rather than its
// threading strategy.
//
// # Where the Go-only guards sit, and why it matters
//
// A nil RECEIVER is refused first, because nothing can be asked of it. A nil
// RESOURCE is refused LAST, immediately before the native call -- not with the
// receiver. The reference has no nil-resource concept at all, so putting that
// guard first would answer it where the reference answers a disposal or a
// range refusal, and a consumer would get the wrong reason.
type SoundEffectInstance struct {
	// effect is `effect`, the SoundEffect this came from. It is held so the
	// instance can be released when the effect is.
	effect *SoundEffect
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the reference's own flag.
	disposed bool
	// The three cached scalars and the loop flag, which the reference stores
	// AFTER the native call so a failed set leaves the previous value readable.
	currentVolume float32
	currentPitch  float32
	currentPan    float32
	looped        bool
	// is3d is set by Apply3D and read by set_Pan, which refuses once it is set.
	is3d bool
	// isPacketSubmitted is `assembly` in the reference and is read by set_Pan's
	// guard prefix, which CLEARS is3d when no packet has been submitted.
	isPacketSubmitted bool
	// derived is the CLR `this`: the outermost object this instance is the base
	// of. Foundation 88 added it with DynamicSoundEffectInstance, which is the
	// contract's only derived type -- and the note that used to stand here,
	// saying the disposal literal would become an identity site the moment that
	// type arrived, is what this field answers.
	derived soundEffectInstanceObject
}

// soundEffectInstanceObject is the CLR `this` a composed SoundEffectInstance
// needs. One member, because one thing needs it: every disposal refusal in the
// family carries `GetType().Name`, and a DynamicSoundEffectInstance must name
// itself.
type soundEffectInstanceObject interface {
	// clrTypeName is System.Object::ToString's answer for this object.
	clrTypeName() string
}

// clrTypeName makes a SoundEffectInstance its own `this` when nothing composes it.
func (i *SoundEffectInstance) clrTypeName() string {
	return "SoundEffectInstance"
}

// self resolves the CLR `this`. It is nil-safe for the reason
// GraphicsResource.self is: a base half whose derived object was never bound is
// still a legal receiver, and answering with itself is what the reference does
// for an object of its own class.
func (i *SoundEffectInstance) self() soundEffectInstanceObject {
	if i != nil && i.derived != nil {
		return i.derived
	}
	return i
}

// bindDerived installs the CLR `this`. Every constructor of a type that
// composes a SoundEffectInstance calls it, and nothing else does.
func (i *SoundEffectInstance) bindDerived(derived soundEffectInstanceObject) {
	if i == nil {
		return
	}
	i.derived = derived
}

// errSoundEffectInstanceNil is the Go-only guard for a zero value.
var errSoundEffectInstanceNil = errors.New("SoundEffectInstance is nil or uninitialized")

// newSoundEffectInstance is SoundEffectInstance::.ctor(SoundEffect, bool), seen
// from the side that matters here: the caller supplies the effect and the
// native handle, and the three scalars start at the reference's own defaults.
func newSoundEffectInstance(effect *SoundEffect, resource *interop.Resource) *SoundEffectInstance {
	return &SoundEffectInstance{
		effect:   effect,
		resource: resource,
		// The reference's constructor sets Volume, Pitch and Pan through the
		// public setters with 1, 0 and 0 -- so these are the values those
		// setters stored, not invented defaults.
		currentVolume: 1,
		currentPitch:  0,
		currentPan:    0,
	}
}

// IsDisposed is SoundEffectInstance::get_IsDisposed, one `ldfld`.
func (i *SoundEffectInstance) IsDisposed() bool {
	if i == nil {
		return true
	}
	return i.disposed
}

// Volume is SoundEffectInstance::get_Volume, one `ldfld` over the cached value.
func (i *SoundEffectInstance) Volume() float32 {
	if i == nil {
		return 0
	}
	return i.currentVolume
}

// SetVolume is SoundEffectInstance::set_Volume, 113 bytes, and it is the shape
// the other two scalars share:
//
//	lock (voiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    if (!(value >= 0) || !(value <= 1))
//	        throw new ArgumentOutOfRangeException("value");
//	    SetVolume(voiceHandle, value);    // native FIRST
//	    currentVolume = value;            // store SECOND
//	}
//
// Two things a reader would get wrong. The comparisons are `blt.un`/`bgt.un`,
// so a NaN takes the throw. And the native call runs BEFORE the store, so a
// failed native call leaves the getter reporting what is actually in effect.
//
// CNA does NOT clamp volume -- its header says it is "passed through by CNA
// without clamping" -- so XNA's own range check is what keeps the member's
// contract, and it stays in front of the route.
func (i *SoundEffectInstance) SetVolume(value float32) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if !(value >= 0) || !(value <= 1) {
		return argumentOutOfRangeError("value", "")
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	if err := i.resource.SoundInstanceSetVolume(value); err != nil {
		return err
	}
	i.currentVolume = value
	return nil
}

// Pitch is SoundEffectInstance::get_Pitch.
func (i *SoundEffectInstance) Pitch() float32 {
	if i == nil {
		return 0
	}
	return i.currentPitch
}

// SetPitch is SoundEffectInstance::set_Pitch, the same body with the range
// [-1, 1].
//
// CNA CLAMPS pitch rather than refusing it, so a value outside the range would
// be accepted natively and silently changed. XNA refuses it, so the check stays
// in front of the route and the clamp is never reached.
func (i *SoundEffectInstance) SetPitch(value float32) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if !(value >= -1) || !(value <= 1) {
		return argumentOutOfRangeError("value", "")
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	if err := i.resource.SoundInstanceSetPitch(value); err != nil {
		return err
	}
	i.currentPitch = value
	return nil
}

// Pan is SoundEffectInstance::get_Pan.
func (i *SoundEffectInstance) Pan() float32 {
	if i == nil {
		return 0
	}
	return i.currentPan
}

// SetPan is SoundEffectInstance::set_Pan, and it is the one scalar setter with
// an extra guard -- two extra statements, in fact:
//
//	lock (voiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    if (!isPacketSubmitted) is3d = false;                  // a WRITE, in a guard prefix
//	    if (is3d) throw new InvalidOperationException(FrameworkResources.InvalidPanCall);
//	    if (!(value >= -1) || !(value <= 1))
//	        throw new ArgumentOutOfRangeException("value");
//	    SetPan(voiceHandle, value); currentPan = value;
//	}
//
// So panning an instance that Apply3D has touched is refused -- and the `is3d`
// flag is CLEARED first when no packet has been submitted, which means an
// instance that was made 3D and then never fed can be panned again. Nothing in
// the signature hints at either statement.
func (i *SoundEffectInstance) SetPan(value float32) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if !i.isPacketSubmitted {
		i.is3d = false
	}
	if i.is3d {
		return fmt.Errorf("%w: %s", errAudioInvalidOperation, invalidPanCall)
	}
	if !(value >= -1) || !(value <= 1) {
		return argumentOutOfRangeError("value", "")
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	if err := i.resource.SoundInstanceSetPan(value); err != nil {
		return err
	}
	i.currentPan = value
	return nil
}

// IsLooped is SoundEffectInstance::get_IsLooped, one `ldfld`.
func (i *SoundEffectInstance) IsLooped() bool {
	if i == nil {
		return false
	}
	return i.looped
}

// SetIsLooped is SoundEffectInstance::set_IsLooped, 57 bytes, and it is the
// THIRD member with a "before the first Play" precondition:
//
//	if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	if (isPacketSubmitted)
//	    throw new InvalidOperationException(FrameworkResources.InvalidIsLoopedCall);
//	looped = value;
//
// # The reference STORES and this projection PUSHES, and CNA agrees anyway
//
// The reference's body makes no native call at all: it stores the flag, and the
// flag is read when the packet is submitted at the first Play. CNA has no
// packet-submission step a caller can reach, so the projection pushes at set
// time -- and CNA enforces the SAME precondition from its own side, refusing
// with CNA_RESULT_INVALID_STATE "after playback has begun".
//
// So the two agree on when the member is legal and differ only in when the
// value travels. That is why the route is bound rather than left managed: a
// stored flag nothing pushed would never reach the mixer.
func (i *SoundEffectInstance) SetIsLooped(value bool) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if i.isPacketSubmitted {
		return fmt.Errorf("%w: %s", errAudioInvalidOperation, invalidIsLoopedCall)
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	if err := i.resource.SoundInstanceSetIsLooped(value); err != nil {
		return err
	}
	i.looped = value
	return nil
}

// State is SoundEffectInstance::get_State, 104 bytes, and it is a DERIVED value
// rather than a field:
//
//	VoiceState voice = (VoiceState)4;         // the value if the call fails
//	GetState(voiceHandle, out voice);
//	SoundState result = SoundState.Stopped;   // the DEFAULT
//	if ((voice & 8) != 0) result = SoundState.Paused;
//	if (voice == 1 || voice == 2) result = SoundState.Playing;
//	return result;
//
// Bit 3 means Paused, voice states 1 and 2 mean Playing, and EVERYTHING ELSE --
// including the 4 the local is seeded with -- reads as Stopped. The two tests
// run in that order and the second WINS.
//
// CNA reports a CNA_SoundState directly, so the projection reads its answer
// rather than reproducing a bit test over a VoiceState this ABI does not
// expose. What it keeps is the FALLBACK: a state CNA reports that XNA has no
// name for reads as Stopped, exactly as an unrecognised VoiceState does there.
func (i *SoundEffectInstance) State() (SoundState, error) {
	if i == nil {
		return SoundStateStopped, errSoundEffectInstanceNil
	}
	if i.disposed {
		return SoundStateStopped, i.objectDisposed()
	}
	if i.resource == nil {
		return SoundStateStopped, errSoundEffectInstanceNil
	}
	info, err := i.resource.SoundInstanceInfo()
	if err != nil {
		return SoundStateStopped, err
	}
	return soundStateFromNative(info.State), nil
}

// soundStateFromNative is get_State's mapping, kept as its own function so the
// FALLBACK branch can be reached by a test. On both qualified artifacts CNA
// reports only its three defined identities, so a mutation of that branch would
// otherwise be a path that never executes -- which is not a killed mutation.
//
// The fallback is the reference's own: its `SoundState result = Stopped` is the
// value that survives when neither the paused bit nor the two playing voice
// states match, including the 4 the local is seeded with when the native call
// fails outright.
func soundStateFromNative(state uint32) SoundState {
	switch state {
	case nativeSoundStatePlaying:
		return SoundStatePlaying
	case nativeSoundStatePaused:
		return SoundStatePaused
	default:
		return SoundStateStopped
	}
}

// The three CNA_SOUND_STATE_* identities. They happen to match XNA's SoundState
// literals, and the projection maps them explicitly anyway: a shared numbering
// is a coincidence to be checked, not a rule to rely on.
const (
	nativeSoundStatePlaying uint32 = 0
	nativeSoundStatePaused  uint32 = 1
	nativeSoundStateStopped uint32 = 2
)

// Play is SoundEffectInstance::Play(), and it is what SUBMITS THE PACKET.
//
// That one store is what fixes the instance's mode. Before it, SetPan, Apply3D
// and SetIsLooped may each be called and each changes what the instance is;
// after it, all three are decided -- SetPan refuses a 3D instance, Apply3D
// refuses a 2D one, and SetIsLooped refuses outright.
//
// CNA describes the same moment in its own words: "the reference implementation
// submits its audio packet on the first `..._play`". So the flag is set here,
// on success, and the three preconditions above become reachable.
func (i *SoundEffectInstance) Play() error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	if err := i.resource.SoundInstancePlay(); err != nil {
		return err
	}
	i.isPacketSubmitted = true
	return nil
}

// Pause is SoundEffectInstance::Pause().
func (i *SoundEffectInstance) Pause() error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	return i.resource.SoundInstancePause()
}

// Resume is SoundEffectInstance::Resume().
func (i *SoundEffectInstance) Resume() error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	return i.resource.SoundInstanceResume()
}

// StopByNone is SoundEffectInstance::Stop(), which the reference implements as
// `Stop(true)` -- the IMMEDIATE stop.
func (i *SoundEffectInstance) StopByNone() error {
	return i.StopByBoolean(true)
}

// StopByBoolean is SoundEffectInstance::Stop(Boolean immediate). CNA's route
// describes the argument in the same words the reference does: "CNA_TRUE to cut
// off now or CNA_FALSE to stop looping and finish naturally."
func (i *SoundEffectInstance) StopByBoolean(immediate bool) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	return i.resource.SoundInstanceStop(immediate)
}

// DisposeByNone is SoundEffectInstance::Dispose(), which the reference
// implements as `Dispose(true); GC.SuppressFinalize(this)`.
func (i *SoundEffectInstance) DisposeByNone() error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	return i.DisposeByBoolean(true)
}

// Finalize is SoundEffectInstance::Finalize, the protected finalizer:
// `Dispose(false)`. Nothing calls it, for the reason every other projected
// finalizer in this module is uncalled: Go has no CLR finalization.
func (i *SoundEffectInstance) Finalize() error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	return i.DisposeByBoolean(false)
}

// DisposeByBoolean is SoundEffectInstance::Dispose(Boolean), which the contract
// declares `protected` and which therefore projects alongside the public one.
func (i *SoundEffectInstance) DisposeByBoolean(disposing bool) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return nil
	}
	i.disposed = true
	if i.resource == nil {
		return nil
	}
	return i.resource.Dispose()
}

// disposeFromEffect is the release SoundEffect.Dispose performs over its
// children. It is separate from DisposeByBoolean only so the effect's own
// teardown reads as what it is.
func (i *SoundEffectInstance) disposeFromEffect() error {
	return i.DisposeByBoolean(true)
}

// objectDisposed is Helpers' ObjectDisposedException(GetType().Name, message),
// and it is this type's IDENTITY SITE.
//
// The reference pushes `GetType().Name`, so a disposed
// DynamicSoundEffectInstance must name ITSELF. Until Foundation 88 the literal
// here had exactly one possible answer and was right by accident; that
// milestone projected the derived type and the answer now comes through self().
//
// Foundation 84 met the same situation with VertexBuffer and IndexBuffer, and
// the note this comment replaces is what stopped it being rediscovered.
func (i *SoundEffectInstance) objectDisposed() error {
	return fmt.Errorf("%w: %s: %s", errAudioObjectDisposed, i.self().clrTypeName(), objectDisposedMessage)
}

// Apply3DByAudioListenerAndAudioEmitter is
// SoundEffectInstance::Apply3D(AudioListener, AudioEmitter), 20 bytes:
//
//	AudioListener[] one = new AudioListener[1];
//	one[0] = listener;
//	SafeApply3D(one, emitter);
//
// A one-element array and a forward, which is why ONE native route serves both
// overloads.
func (i *SoundEffectInstance) Apply3DByAudioListenerAndAudioEmitter(listener *AudioListener, emitter *AudioEmitter) error {
	return i.Apply3DBySliceOfAudioListenerAndAudioEmitter([]*AudioListener{listener}, emitter)
}

// Apply3DBySliceOfAudioListenerAndAudioEmitter is
// SoundEffectInstance::Apply3D(AudioListener[], AudioEmitter), which forwards
// through SafeApply3D to UnsafeApply3D -- 256 bytes whose guard prefix is the
// exact MIRROR of set_Pan's:
//
//	lock (voiceHandleLock) {
//	    if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
//	    if (!isPacketSubmitted) is3d = true;      // set_Pan CLEARS it here
//	    if (!is3d)
//	        throw new InvalidOperationException(FrameworkResources.InvalidApply3DCall);
//	    ... copy each listener's data into a cached array, then Apply3D natively ...
//	}
//
// # The pair is a MODE LATCH, and the two members are its two halves
//
// Before a packet has been submitted -- that is, before the instance has played
// -- whichever member is called sets the mode: Apply3D makes it 3D and set_Pan
// makes it not. Once a packet HAS been submitted the mode is fixed, and the
// member for the other mode refuses.
//
// CNA describes the same rule from its own side, under the heading "Aim before
// you play": its route "refuses with CNA_RESULT_INVALID_STATE on an instance
// that is playing and was never positioned", and once positioned "the spatial
// volume, pan and pitch it computes are latched ... and set_pan stops reaching
// the output."
//
// # Two GO-ONLY refusals, and what the reference does instead
//
// UnsafeApply3D has NO null check on either argument: it dereferences
// `listeners[index].listenerData` and `emitter.emitterData`, so the reference
// throws NullReferenceException. Go cannot project that, so the two guards
// below answer in their own words -- the same position IndexBuffer's device
// check is in.
//
// An EMPTY array is a different matter and is left to CNA. The reference
// reaches its native call with a count of zero and surfaces whatever XACT
// returns; CNA refuses it and says why -- "an outcome not established here" --
// so the refusal a consumer meets is the renderer's rather than one invented
// on its behalf.
//
// # How several listeners combine is CNA's approximation, and it is recorded
//
// XACT computes per-listener output matrices. CNA's mixer has a single stereo
// gain pair and no equivalent, so it evaluates every listener and lets the
// NEAREST one decide. That is a documented divergence in what the sound
// becomes, not in what the member accepts, and nothing in the projection can
// narrow it.
func (i *SoundEffectInstance) Apply3DBySliceOfAudioListenerAndAudioEmitter(listeners []*AudioListener, emitter *AudioEmitter) error {
	if i == nil {
		return errSoundEffectInstanceNil
	}
	if i.disposed {
		return i.objectDisposed()
	}
	if !i.isPacketSubmitted {
		i.is3d = true
	}
	if !i.is3d {
		return fmt.Errorf("%w: %s", errAudioInvalidOperation, invalidApply3DCall)
	}
	if emitter == nil {
		return argumentNullError("emitter")
	}
	flat := make([]float32, 0, len(listeners)*12)
	for index, listener := range listeners {
		if listener == nil {
			return argumentNullError(fmt.Sprintf("listeners[%d]", index))
		}
		flat = appendVector3(flat, listener.Forward())
		flat = appendVector3(flat, listener.Position())
		flat = appendVector3(flat, listener.Up())
		flat = appendVector3(flat, listener.Velocity())
	}
	emitterData := make([]float32, 0, 13)
	emitterData = append(emitterData, emitter.DopplerScale())
	emitterData = appendVector3(emitterData, emitter.Forward())
	emitterData = appendVector3(emitterData, emitter.Position())
	emitterData = appendVector3(emitterData, emitter.Up())
	emitterData = appendVector3(emitterData, emitter.Velocity())
	if len(flat) == 0 {
		// CNA refuses a zero count by name; reaching it with an empty slice
		// would be a nil pointer, so the refusal is produced here in CNA's
		// terms rather than by dereferencing nothing.
		return fmt.Errorf("%w: listeners", errAudioArgument)
	}
	if i.resource == nil {
		return errSoundEffectInstanceNil
	}
	return i.resource.SoundInstanceApply3D(flat, uint64(len(listeners)), emitterData)
}

// appendVector3 flattens one vector in the order CNA's structures declare their
// fields. The ORDER is the contract between this file and the bridge, and it is
// written once so the two cannot disagree.
func appendVector3(into []float32, vector framework.Vector3) []float32 {
	return append(into, vector.X, vector.Y, vector.Z)
}
