package audio

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// Cue is Microsoft.Xna.Framework.Audio.Cue:
//
//	.class public auto ansi sealed beforefieldinit Cue
//	       extends [mscorlib]System.Object
//	       implements [mscorlib]System.IDisposable
//
// One playable authored sound from a sound bank, with a name, a variable set, a
// position and SEVEN state predicates.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_cue_destroy,
//	registered as a CHILD of the sound bank that produced it.
//
// # The eight predicates come from ONE observation
//
// XNA declares IsCreated, IsPreparing, IsPrepared, IsPlaying, IsStopping,
// IsStopped, IsPaused and IsDisposed as eight separate properties, and CNA
// answers all eight from a single `cna_cue_get_info`. Reading them one at a
// time would let a caller see a combination that never existed -- a cue
// reporting IsPlaying and IsStopped together because it stopped between two
// reads -- so each projected property below is one call to that route, and a
// caller that wants a consistent set of several reads them from one Info.
//
// IsDisposed is the exception: the reference reads a managed field for it, and
// asking CNA about a cue this projection has already destroyed is not a
// question that has an answer.
type Cue struct {
	// resource is the owned CNA handle.
	resource *interop.Resource
	// disposed is the managed latch get_IsDisposed reads.
	disposed bool
	// disposing carries the Disposing event and its native registration.
	disposing xactDisposingEvent
}

// Name is Cue::get_Name, the authored cue name.
func (c *Cue) Name() (string, error) {
	if err := c.usable(); err != nil {
		return "", err
	}
	return c.resource.CueName()
}

// IsCreated is Cue::get_IsCreated.
func (c *Cue) IsCreated() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsCreated })
}

// IsPreparing is Cue::get_IsPreparing.
func (c *Cue) IsPreparing() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsPreparing })
}

// IsPrepared is Cue::get_IsPrepared.
func (c *Cue) IsPrepared() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsPrepared })
}

// IsPlaying is Cue::get_IsPlaying.
func (c *Cue) IsPlaying() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsPlaying })
}

// IsStopping is Cue::get_IsStopping.
func (c *Cue) IsStopping() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsStopping })
}

// IsStopped is Cue::get_IsStopped.
func (c *Cue) IsStopped() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsStopped })
}

// IsPaused is Cue::get_IsPaused.
func (c *Cue) IsPaused() (bool, error) {
	return c.state(func(s interop.CueStates) bool { return s.IsPaused })
}

// IsDisposed is Cue::get_IsDisposed, one `ldfld`.
func (c *Cue) IsDisposed() bool {
	return c == nil || c.disposed
}

// state is the shared body of the seven native predicates.
func (c *Cue) state(read func(interop.CueStates) bool) (bool, error) {
	if err := c.usable(); err != nil {
		return false, err
	}
	states, err := c.resource.CueInfo()
	if err != nil {
		return false, err
	}
	return read(states), nil
}

// Play is Cue::Play().
func (c *Cue) Play() error {
	if err := c.usable(); err != nil {
		return err
	}
	return c.resource.CuePlay()
}

// Pause is Cue::Pause().
func (c *Cue) Pause() error {
	if err := c.usable(); err != nil {
		return err
	}
	return c.resource.CuePause()
}

// Resume is Cue::Resume().
func (c *Cue) Resume() error {
	if err := c.usable(); err != nil {
		return err
	}
	return c.resource.CueResume()
}

// Stop is Cue::Stop(AudioStopOptions).
func (c *Cue) Stop(options AudioStopOptions) error {
	if err := c.usable(); err != nil {
		return err
	}
	return c.resource.CueStop(uint32(options))
}

// GetVariable is Cue::GetVariable(String), which reads a cue-scoped variable.
// A cue variable is distinct from the engine's global of the same name: XACT
// keeps a per-cue copy, which is what makes per-instance pitch and volume
// possible.
func (c *Cue) GetVariable(name string) (float32, error) {
	if err := c.usable(); err != nil {
		return 0, err
	}
	return c.resource.CueVariable(name)
}

// SetVariable is Cue::SetVariable(String, Single).
func (c *Cue) SetVariable(name string, value float32) error {
	if err := c.usable(); err != nil {
		return err
	}
	return c.resource.CueSetVariable(name, value)
}

// Apply3D is Cue::Apply3D(AudioListener, AudioEmitter). It takes no name
// suffix because Cue declares ONE Apply3D -- unlike SoundEffectInstance, which
// declares two and whose projections are therefore suffixed.
//
// Unlike SoundEffectInstance's Apply3D there is NO mode latch here: a cue is 3D
// or not by how it was authored, and calling this on a cue that was not
// authored positional is XACT's decision to refuse rather than the projection's.
//
// The reference dereferences both arguments unguarded, so it throws
// NullReferenceException; the two guards below answer in Go's own words.
func (c *Cue) Apply3D(listener *AudioListener, emitter *AudioEmitter) error {
	if err := c.usable(); err != nil {
		return err
	}
	if listener == nil {
		return argumentNullError("listener")
	}
	if emitter == nil {
		return argumentNullError("emitter")
	}
	return c.resource.CueApply3D(flattenListener(listener), flattenEmitter(emitter))
}

// AddDisposingHandler is Cue::add_Disposing.
func (c *Cue) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if c == nil {
		return framework.EventSubscription{}, errXactNil
	}
	return c.disposing.source.Add(handler)
}

// RemoveDisposingHandler is Cue::remove_Disposing.
func (c *Cue) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if c == nil {
		return errXactNil
	}
	return c.disposing.source.Remove(subscription)
}

// Dispose is Cue::Dispose().
//
// The contract declares only the PUBLIC Dispose here -- Cue is `sealed` and has
// no protected Dispose(Boolean), which is the one place the four disposable
// XACT types differ from each other.
func (c *Cue) Dispose() error {
	if c == nil {
		return errXactNil
	}
	if c.disposed {
		return nil
	}
	c.disposed = true
	var failures []error
	if err := c.resource.Dispose(); err != nil {
		failures = append(failures, err)
	}
	if err := c.disposing.release(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// Finalize is Cue::Finalize, which the reference implements as `Dispose(false)`
// over a private Dispose(bool) the contract does not declare.
func (c *Cue) Finalize() error {
	return c.Dispose()
}

// usable is the guard every native member shares.
func (c *Cue) usable() error {
	if c == nil {
		return errXactNil
	}
	if c.disposed {
		return xactObjectDisposed("Cue")
	}
	return nil
}

// disposeFromBank is what the sound bank's teardown calls.
func (c *Cue) disposeFromBank() error {
	return c.Dispose()
}
