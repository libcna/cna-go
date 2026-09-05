package audio

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// AudioEngine is Microsoft.Xna.Framework.Audio.AudioEngine:
//
//	.class public auto ansi beforefieldinit AudioEngine
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//
// The root of XACT. It owns the authored settings -- the categories, the global
// variables and the renderer list -- and every bank and cue is created under
// it.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_audio_engine_destroy.
//
// Its banks and categories are registered as CHILD resources, which makes CNA's
// ordering rule structural rather than something a caller has to remember.
//
// # The settings file is a PATH, and this projection never reads it
//
// Both constructors take a file name and hand it to XACT. Foundation 98
// measured what CNA's parser accepts before binding anything: it reports a
// malformed file through its LOG rather than through the result code, and the
// smallest file it will open is 80 bytes beginning `XGSF`. A settings file with
// real categories and variables is 136 bytes and entirely authorable, which is
// what the qualification slice writes.
//
// # ContentVersion is 39, not 46
//
// The pinned contract's field is `public const int ContentVersion = 39`. CNA's
// own fixture generator writes 46 into the file header, and both are true of
// different things: 39 is what this XNA build's constant says, and 46 is what
// CNA's parser accepts in a file. The projection reports the CONTRACT's value,
// because that is what the member is.
type AudioEngine struct {
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the managed latch get_IsDisposed reads. It is a field and not
	// a native read for the reason SoundEffect's is: the reference reads a
	// field, and asking CNA would be a second answer to one question.
	disposed bool
	// disposing carries the Disposing event and its native registration.
	disposing xactDisposingEvent
	// children are the banks and categories created under this engine, released
	// in Dispose before the engine itself.
	children []interface{ disposeFromEngine() error }
}

// AudioEngineContentVersion is AudioEngine::ContentVersion, a public constant.
//
// It is a package-level constant rather than a method because the reference
// declares a `const` FIELD: a caller reads it without an instance, and a method
// would need one.
const AudioEngineContentVersion int32 = 39

// NewAudioEngineByString is AudioEngine::.ctor(String), the one-argument
// constructor. The reference forwards to the three-argument one with
// `TimeSpan.Zero` and a null renderer id; CNA has a separate route for exactly
// that shape, so the projection calls it rather than passing a zero and an
// empty string through the wider one.
func NewAudioEngineByString(settingsFile string) (*AudioEngine, error) {
	runtime, err := currentRuntime()
	if err != nil {
		return nil, err
	}
	resource, err := runtime.AudioEngineCreate(settingsFile)
	if err != nil {
		return nil, err
	}
	return finishAudioEngine(resource)
}

// NewAudioEngineByStringAndTimeSpanAndString is
// AudioEngine::.ctor(String, TimeSpan, String).
//
// The look-ahead is a TimeSpan and reaches CNA as TICKS, which is the TimeSpan's
// exact storage -- so no precision is lost on the way down, which would not be
// true of milliseconds.
func NewAudioEngineByStringAndTimeSpanAndString(settingsFile string, lookAheadTime framework.TimeSpan, rendererID string) (*AudioEngine, error) {
	runtime, err := currentRuntime()
	if err != nil {
		return nil, err
	}
	resource, err := runtime.AudioEngineCreateWithRenderer(settingsFile, lookAheadTime.Ticks(), rendererID)
	if err != nil {
		return nil, err
	}
	return finishAudioEngine(resource)
}

// finishAudioEngine is the shared tail of both constructors: wrap the handle and
// install the disposal notification.
//
// If the subscription fails the engine is destroyed rather than returned. A
// half-built engine whose Disposing event could never fire is worse than a
// constructor that refuses, because the failure would surface much later and
// somewhere else.
func finishAudioEngine(resource *interop.Resource) (*AudioEngine, error) {
	engine := &AudioEngine{resource: resource}
	if err := engine.disposing.subscribe(resource, interop.XactAudioEngine, engine); err != nil {
		return nil, errors.Join(err, resource.Dispose())
	}
	return engine, nil
}

// IsDisposed is AudioEngine::get_IsDisposed, one `ldfld`.
func (e *AudioEngine) IsDisposed() bool {
	return e == nil || e.disposed
}

// Update is AudioEngine::Update(), which XACT requires to be called regularly:
// it is what advances cue state, applies category changes and reclaims voices.
func (e *AudioEngine) Update() error {
	if e == nil {
		return errXactNil
	}
	if e.disposed {
		return xactObjectDisposed("AudioEngine")
	}
	return e.resource.AudioEngineUpdate()
}

// GetCategory is AudioEngine::GetCategory(String).
//
// # Every call produces a NEW handle
//
// The reference returns a `struct` built from an index into the engine's
// authored table, so two calls with one name produce two equal values. CNA
// returns an owned handle from each call, so two calls produce two DIFFERENT
// handles that denote the same category -- which is why AudioCategory's
// equality is cna_audio_category_equals and not a handle comparison.
//
// Each handle is registered under the engine, so a game that looks a category
// up every frame accumulates handles until the engine is disposed. That is a
// real cost and it is stated rather than hidden; the reference has the same
// shape without the handle, and a caller that minds it can hold the value.
//
// A name XACT does not know is a REFUSAL, not a zero value: CNA reports it as
// an invalid argument, which is the condition the reference's own
// InvalidOperationException marks.
func (e *AudioEngine) GetCategory(name string) (AudioCategory, error) {
	if e == nil {
		return AudioCategory{}, errXactNil
	}
	if e.disposed {
		return AudioCategory{}, xactObjectDisposed("AudioEngine")
	}
	resource, err := e.resource.AudioEngineGetCategory(name)
	if err != nil {
		return AudioCategory{}, err
	}
	category := AudioCategory{resource: resource}
	e.adopt(category)
	return category, nil
}

// GetGlobalVariable is AudioEngine::GetGlobalVariable(String).
func (e *AudioEngine) GetGlobalVariable(name string) (float32, error) {
	if e == nil {
		return 0, errXactNil
	}
	if e.disposed {
		return 0, xactObjectDisposed("AudioEngine")
	}
	return e.resource.AudioEngineGlobalVariable(name)
}

// SetGlobalVariable is AudioEngine::SetGlobalVariable(String, Single).
//
// A variable authored READ-ONLY silently ignores the write, and both XACT and
// CNA report success for it. That is the authored file's decision rather than
// this projection's, and it is worth knowing about: a settings file whose
// variable byte is 0x03 rather than 0x01 makes every set here a no-op that
// reads back as the initial value.
func (e *AudioEngine) SetGlobalVariable(name string, value float32) error {
	if e == nil {
		return errXactNil
	}
	if e.disposed {
		return xactObjectDisposed("AudioEngine")
	}
	return e.resource.AudioEngineSetGlobalVariable(name, value)
}

// RendererDetails is AudioEngine::get_RendererDetails, whose CLR type is
// `ReadOnlyCollection<RendererDetail>`.
//
// The projection answers a Go slice, which is the settled mapping for a
// ReadOnlyCollection: a fresh slice per call, so a caller mutating it changes
// nothing the engine holds -- the same guarantee the CLR collection's
// read-only-ness gives.
//
// Only the two STRINGS come from CNA. RendererDetail's ToString, GetHashCode
// and equality are pure managed and already projected, and CNA's routes for
// those three are deliberately unbound: its hash cannot reproduce the pinned
// mscorlib string hash, and its equality compares collection INDICES where the
// reference compares the two strings.
func (e *AudioEngine) RendererDetails() (*framework.ReadOnlyCollection[RendererDetail], error) {
	if e == nil {
		return nil, errXactNil
	}
	if e.disposed {
		return nil, xactObjectDisposed("AudioEngine")
	}
	count, err := e.resource.AudioEngineRendererCount()
	if err != nil {
		return nil, err
	}
	details := make([]RendererDetail, 0, count)
	for index := int32(0); index < count; index++ {
		detail, err := e.resource.AudioEngineRendererDetail(index)
		if err != nil {
			return nil, err
		}
		details = append(details, newRendererDetail(detail.FriendlyName, detail.RendererID))
	}
	return framework.NewReadOnlyCollectionOverValues(details), nil
}

// AddDisposingHandler is AudioEngine::add_Disposing.
func (e *AudioEngine) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if e == nil {
		return framework.EventSubscription{}, errXactNil
	}
	return e.disposing.source.Add(handler)
}

// RemoveDisposingHandler is AudioEngine::remove_Disposing.
func (e *AudioEngine) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if e == nil {
		return errXactNil
	}
	return e.disposing.source.Remove(subscription)
}

// DisposeByNone is AudioEngine::Dispose(), `Dispose(true); GC.SuppressFinalize(this)`.
//
// The order is the one CNA's header requires and the reference's own
// Dispose(bool) follows: the banks and categories created under this engine go
// first, then the engine. Releasing the disposal REGISTRATION happens last of
// all, after the native destroy, because destroying the engine is what raises
// the event -- releasing first would drop the notification the caller
// subscribed for.
func (e *AudioEngine) DisposeByNone() error {
	if e == nil {
		return errXactNil
	}
	if e.disposed {
		return nil
	}
	e.disposed = true
	var failures []error
	for _, child := range e.children {
		if err := child.disposeFromEngine(); err != nil {
			failures = append(failures, err)
		}
	}
	e.children = nil
	if err := e.resource.Dispose(); err != nil {
		failures = append(failures, err)
	}
	if err := e.disposing.release(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// DisposeByBoolean is AudioEngine::Dispose(Boolean), the protected overload.
//
// The reference's two branches differ only in whether managed objects are
// touched, and both release the native engine. A `false` here is the finalizer
// path, which in the CLR must not touch other managed objects because they may
// already be collected -- so it skips the children and releases only the
// engine, which is exactly what the reference does.
func (e *AudioEngine) DisposeByBoolean(disposing bool) error {
	if e == nil {
		return errXactNil
	}
	if disposing {
		return e.DisposeByNone()
	}
	if e.disposed {
		return nil
	}
	e.disposed = true
	e.children = nil
	return e.resource.Dispose()
}

// Finalize is AudioEngine::Finalize, `Dispose(false)`.
//
// Nothing calls it: Go has no CLR finalization and CNA-Go registers no runtime
// finalizer. It is projected because the pinned contract declares it.
func (e *AudioEngine) Finalize() error {
	return e.DisposeByBoolean(false)
}

// adopt registers a bank as a child of the engine so the engine's teardown
// releases it. It is unexported because the reference's own tracking is
// private.
func (e *AudioEngine) adopt(child interface{ disposeFromEngine() error }) {
	e.children = append(e.children, child)
}

// usable is the guard the bank constructors share: an engine that is nil or
// disposed cannot author anything.
func (e *AudioEngine) usable() error {
	if e == nil {
		return fmt.Errorf("%w: audioEngine", errAudioArgumentNull)
	}
	if e.disposed {
		return xactObjectDisposed("AudioEngine")
	}
	return nil
}
