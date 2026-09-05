package audio

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// WaveBank is Microsoft.Xna.Framework.Audio.WaveBank:
//
//	.class public auto ansi beforefieldinit WaveBank
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//
// One compiled `.xwb`: the WAVE DATA a sound bank's cues play. It has no method
// of its own -- a game creates it, keeps it alive while the cues that need it
// are playing, and disposes it.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_wave_bank_destroy,
//	registered as a CHILD of its engine.
//
// # IsPrepared is the streaming constructor's whole point
//
// A non-streaming bank is fully in memory when the constructor returns, so it
// is prepared immediately. A STREAMING one is not: the constructor returns
// before the data is ready, and `get_IsPrepared` is how a caller learns it may
// play. It is the one member that distinguishes the two constructors.
type WaveBank struct {
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the managed latch get_IsDisposed reads.
	disposed bool
	// disposing carries the Disposing event and its native registration.
	disposing xactDisposingEvent
}

// NewWaveBankByAudioEngineAndString is
// WaveBank::.ctor(AudioEngine, String), the non-streaming constructor. Its
// parameter is named `nonStreamingWaveBankFilename` in the reference, which is
// the clearest statement of what separates the two.
func NewWaveBankByAudioEngineAndString(audioEngine *AudioEngine, nonStreamingWaveBankFilename string) (*WaveBank, error) {
	if err := audioEngine.usable(); err != nil {
		return nil, err
	}
	resource, err := audioEngine.resource.WaveBankCreate(nonStreamingWaveBankFilename)
	if err != nil {
		return nil, err
	}
	return finishWaveBank(audioEngine, resource)
}

// NewWaveBankByAudioEngineAndStringAndInt32AndInt16 is
// WaveBank::.ctor(AudioEngine, String, Int32, Int16), the streaming
// constructor.
//
// `packetsize` is an Int16 in the reference and an int16_t in CNA, so it stays
// one here rather than widening: a caller that passes a value a short cannot
// hold has made an error the reference would also refuse.
func NewWaveBankByAudioEngineAndStringAndInt32AndInt16(audioEngine *AudioEngine, streamingWaveBankFilename string, offset int32, packetsize int16) (*WaveBank, error) {
	if err := audioEngine.usable(); err != nil {
		return nil, err
	}
	resource, err := audioEngine.resource.WaveBankCreateStreaming(streamingWaveBankFilename, offset, packetsize)
	if err != nil {
		return nil, err
	}
	return finishWaveBank(audioEngine, resource)
}

// finishWaveBank is the shared tail of both constructors.
func finishWaveBank(audioEngine *AudioEngine, resource *interop.Resource) (*WaveBank, error) {
	bank := &WaveBank{resource: resource}
	if err := bank.disposing.subscribe(resource, interop.XactWaveBank, bank); err != nil {
		return nil, errors.Join(err, resource.Dispose())
	}
	audioEngine.adopt(bank)
	return bank, nil
}

// IsDisposed is WaveBank::get_IsDisposed, one `ldfld`.
func (b *WaveBank) IsDisposed() bool {
	return b == nil || b.disposed
}

// IsInUse is WaveBank::get_IsInUse: whether any wave in this bank is sounding.
func (b *WaveBank) IsInUse() (bool, error) {
	if b == nil {
		return false, errXactNil
	}
	if b.disposed {
		return false, xactObjectDisposed("WaveBank")
	}
	return b.resource.WaveBankIsInUse()
}

// IsPrepared is WaveBank::get_IsPrepared: whether the bank's data has finished
// loading. A non-streaming bank answers true from the moment it exists.
func (b *WaveBank) IsPrepared() (bool, error) {
	if b == nil {
		return false, errXactNil
	}
	if b.disposed {
		return false, xactObjectDisposed("WaveBank")
	}
	return b.resource.WaveBankIsPrepared()
}

// AddDisposingHandler is WaveBank::add_Disposing.
func (b *WaveBank) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if b == nil {
		return framework.EventSubscription{}, errXactNil
	}
	return b.disposing.source.Add(handler)
}

// RemoveDisposingHandler is WaveBank::remove_Disposing.
func (b *WaveBank) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if b == nil {
		return errXactNil
	}
	return b.disposing.source.Remove(subscription)
}

// Dispose is WaveBank::Dispose(). The bank owns no managed children, so it is
// the native destroy and then the registration -- in that order, because the
// destroy is what raises the event.
func (b *WaveBank) DisposeByNone() error {
	if b == nil {
		return errXactNil
	}
	if b.disposed {
		return nil
	}
	b.disposed = true
	var failures []error
	if err := b.resource.Dispose(); err != nil {
		failures = append(failures, err)
	}
	if err := b.disposing.release(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// DisposeByBoolean is WaveBank::Dispose(Boolean), the protected overload.
func (b *WaveBank) DisposeByBoolean(disposing bool) error {
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
	return b.resource.Dispose()
}

// Finalize is WaveBank::Finalize, `Dispose(false)`.
func (b *WaveBank) Finalize() error {
	return b.DisposeByBoolean(false)
}

// disposeFromEngine is what the engine's teardown calls.
func (b *WaveBank) disposeFromEngine() error {
	return b.DisposeByNone()
}
