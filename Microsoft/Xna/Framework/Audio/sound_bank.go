package audio

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SoundBank is Microsoft.Xna.Framework.Audio.SoundBank:
//
//	.class public auto ansi beforefieldinit SoundBank
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//
// One compiled `.xsb`: the authored CUES, each naming a sound, a category and a
// wave in some wave bank.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_sound_bank_destroy,
//	registered as a CHILD of its engine.
//
// # IsInUse is CNA's, IsDisposed is the projection's
//
// The two look alike and are not. `get_IsDisposed` reads a managed field in the
// reference -- a bank knows it was disposed without asking XACT -- so this
// projection reads its own latch. `get_IsInUse` asks whether any cue from this
// bank is still playing, which only XACT knows, so it reaches CNA.
type SoundBank struct {
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the managed latch get_IsDisposed reads.
	disposed bool
	// disposing carries the Disposing event and its native registration.
	disposing xactDisposingEvent
	// cues are the Cue objects GetCue produced, released before the bank.
	cues []*Cue
}

// NewSoundBank is SoundBank::.ctor(AudioEngine, String), the type's ONLY
// constructor -- which is why it takes no parameter-shape suffix where
// WaveBank's two do.
//
// The engine is checked BEFORE the filename reaches CNA, in the reference's own
// order: its first statement is the null check on audioEngine.
func NewSoundBank(audioEngine *AudioEngine, filename string) (*SoundBank, error) {
	if err := audioEngine.usable(); err != nil {
		return nil, err
	}
	resource, err := audioEngine.resource.SoundBankCreate(filename)
	if err != nil {
		return nil, err
	}
	bank := &SoundBank{resource: resource}
	if err := bank.disposing.subscribe(resource, interop.XactSoundBank, bank); err != nil {
		return nil, errors.Join(err, resource.Dispose())
	}
	audioEngine.adopt(bank)
	return bank, nil
}

// IsDisposed is SoundBank::get_IsDisposed, one `ldfld`.
func (b *SoundBank) IsDisposed() bool {
	return b == nil || b.disposed
}

// IsInUse is SoundBank::get_IsInUse: whether any cue from this bank is still
// sounding. It reaches CNA because XACT is the only thing that knows.
func (b *SoundBank) IsInUse() (bool, error) {
	if b == nil {
		return false, errXactNil
	}
	if b.disposed {
		return false, xactObjectDisposed("SoundBank")
	}
	return b.resource.SoundBankIsInUse()
}

// GetCue is SoundBank::GetCue(String), which builds a Cue a caller then plays,
// positions and stops.
//
// The cue is registered as a child of the BANK, not of the engine: it is the
// bank that authored it, and it must not outlive the bank.
func (b *SoundBank) GetCue(name string) (*Cue, error) {
	if b == nil {
		return nil, errXactNil
	}
	if b.disposed {
		return nil, xactObjectDisposed("SoundBank")
	}
	resource, err := b.resource.SoundBankGetCue(name)
	if err != nil {
		return nil, err
	}
	cue := &Cue{resource: resource}
	if err := cue.disposing.subscribe(resource, interop.XactCue, cue); err != nil {
		return nil, errors.Join(err, resource.Dispose())
	}
	b.cues = append(b.cues, cue)
	return cue, nil
}

// PlayCueByString is SoundBank::PlayCue(String), the fire-and-forget overload:
// it starts a cue the caller never receives and therefore cannot stop.
func (b *SoundBank) PlayCueByString(name string) error {
	if b == nil {
		return errXactNil
	}
	if b.disposed {
		return xactObjectDisposed("SoundBank")
	}
	return b.resource.SoundBankPlayCue(name)
}

// PlayCueByStringAndAudioListenerAndAudioEmitter is
// SoundBank::PlayCue(String, AudioListener, AudioEmitter), the positional
// fire-and-forget.
//
// The reference dereferences both arguments with no null check, so it throws
// NullReferenceException; Go cannot project that and answers the argument-null
// refusal in its own words instead -- the same position Apply3D is in.
func (b *SoundBank) PlayCueByStringAndAudioListenerAndAudioEmitter(name string, listener *AudioListener, emitter *AudioEmitter) error {
	if b == nil {
		return errXactNil
	}
	if b.disposed {
		return xactObjectDisposed("SoundBank")
	}
	if listener == nil {
		return argumentNullError("listener")
	}
	if emitter == nil {
		return argumentNullError("emitter")
	}
	return b.resource.SoundBankPlayCue3D(name, flattenListener(listener), flattenEmitter(emitter))
}

// AddDisposingHandler is SoundBank::add_Disposing.
func (b *SoundBank) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errXactNil
	}
	return b.disposing.source.Add(handler)
}

// RemoveDisposingHandler is SoundBank::remove_Disposing.
func (b *SoundBank) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errXactNil
	}
	return b.disposing.source.Remove(subscription)
}

// Dispose is SoundBank::Dispose(). Its cues go first, then the bank, then the
// registration -- the same order the engine's teardown follows and for the same
// reason: destroying the bank is what raises the event.
func (b *SoundBank) DisposeByNone() error {
	if b == nil {
		return errXactNil
	}
	if b.disposed {
		return nil
	}
	b.disposed = true
	var failures []error
	for _, cue := range b.cues {
		if err := cue.disposeFromBank(); err != nil {
			failures = append(failures, err)
		}
	}
	b.cues = nil
	if err := b.resource.Dispose(); err != nil {
		failures = append(failures, err)
	}
	if err := b.disposing.release(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// DisposeByBoolean is SoundBank::Dispose(Boolean), the protected overload. The
// `false` branch is the finalizer path and touches no other managed object.
func (b *SoundBank) DisposeByBoolean(disposing bool) error {
	if b == nil {
		return errXactNil
	}
	if disposing {
		return b.DisposeByNone()
	}
	if b.disposed {
		return nil
	}
	b.disposed = true
	b.cues = nil
	return b.resource.Dispose()
}

// Finalize is SoundBank::Finalize, `Dispose(false)`.
func (b *SoundBank) Finalize() error {
	return b.DisposeByBoolean(false)
}

// disposeFromEngine is what the engine's teardown calls.
func (b *SoundBank) disposeFromEngine() error {
	return b.DisposeByNone()
}
