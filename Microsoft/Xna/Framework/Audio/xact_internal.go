package audio

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 98 -- XACT. Five types over 58 routes.
//
// AudioEngine, AudioCategory, SoundBank, WaveBank and Cue. Their reference
// authority is Microsoft.Xna.Framework.Xact.dll, the SECOND assembly this
// module projects from -- RendererDetail was the first, and it needed nothing
// native at all. These five are the opposite: almost every member reaches XACT.
// ---------------------------------------------------------------------------

// errXactNil is the Go-only refusal a member answers on a nil receiver. The
// reference has no such state: a CLR instance method on a null reference throws
// NullReferenceException before any body runs, and Go cannot project that.
var errXactNil = errors.New("the XACT object is nil")

// errXactNoRuntime is what a constructor answers with no loaded runtime. The
// reference reaches XACT directly and has no equivalent.
var errXactNoRuntime = fmt.Errorf("%w: this member needs a loaded runtime", errAudioInvalidOperation)

// xactObjectDisposed builds the ObjectDisposedException the four disposable
// XACT types raise, with the reference's own two arguments: GetType().Name and
// the fixed sentence. The NAME is what tells a caller which of the four refused.
func xactObjectDisposed(typeName string) error {
	return fmt.Errorf("%w: %s: %s", errAudioObjectDisposed, typeName, objectDisposedMessage)
}

// currentRuntime is the one place the XACT family reaches for the runtime, so
// the "no runtime" refusal reads the same from every constructor.
func currentRuntime() (*interop.Runtime, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errXactNoRuntime
	}
	return runtime, nil
}

// flattenListener and flattenEmitter produce the flat float shapes the bridge's
// fill helpers consume.
//
// The ORDER is the settled contract between this package and bridge.c, fixed in
// Foundation 87 by SoundEffectInstance::Apply3D and unchanged here: a listener
// is Forward, Position, Up, Velocity, and an emitter is DopplerScale followed
// by the same four. Reusing appendVector3 rather than writing a second flatten
// is what keeps XACT's Apply3D and SoundEffectInstance's from drifting apart.
func flattenListener(listener *AudioListener) []float32 {
	flat := make([]float32, 0, 12)
	flat = appendVector3(flat, listener.Forward())
	flat = appendVector3(flat, listener.Position())
	flat = appendVector3(flat, listener.Up())
	flat = appendVector3(flat, listener.Velocity())
	return flat
}

func flattenEmitter(emitter *AudioEmitter) []float32 {
	flat := make([]float32, 0, 13)
	flat = append(flat, emitter.DopplerScale())
	flat = appendVector3(flat, emitter.Forward())
	flat = appendVector3(flat, emitter.Position())
	flat = appendVector3(flat, emitter.Up())
	flat = appendVector3(flat, emitter.Velocity())
	return flat
}

// xactDisposingEvent is the shared half of the four Disposing events.
//
// All four types declare `event EventHandler<EventArgs> Disposing`, and all
// four raise it from the same place: CNA's disposal notification. Holding the
// source and the native registration together in one struct is what lets each
// type's three members -- add, remove, and the teardown that releases the
// registration -- be three forwards rather than three bodies.
type xactDisposingEvent struct {
	source       framework.EventSource[*framework.EventArgs]
	registration *interop.XactDisposingRegistration
}

// subscribe installs the native callback that raises the managed event.
//
// It is called from each constructor, not lazily from the first Add, because
// CNA raises the notification whether or not anything is listening: subscribing
// late would miss a disposal that happened before the first handler arrived.
//
// A raise error is DROPPED here and that is deliberate. CNA calls this
// synchronously from inside its own disposal, through a C frame that returns
// void, so there is nowhere for an error to go -- the same wall every callback
// in this module meets. What the runtime does instead is record it, and Run
// surfaces it.
func (e *xactDisposingEvent) subscribe(resource *interop.Resource, kind interop.XactKind, sender any) error {
	registration, err := resource.XactSubscribeDisposing(kind, func() {
		_ = e.source.Raise(sender, framework.EventArgsEmpty())
	})
	if err != nil {
		return err
	}
	e.registration = registration
	return nil
}

// release drops the native registration. It is idempotent.
func (e *xactDisposingEvent) release() error {
	if e.registration == nil {
		return nil
	}
	registration := e.registration
	e.registration = nil
	return registration.Release()
}
