package audio

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The XACT family's managed half is measured here without an engine. What a
// unit test can reach is exactly the part that does not depend on XACT: the
// nil and disposal guards, the zero AudioCategory, the flattening order the
// bridge depends on, and the contract constant.
//
// Everything below those guards -- construction, category lookup, cue
// transport, the disposal callbacks -- needs a real engine over real fixtures
// and is qualified by the native stress slice instead.

func TestAudioEngineContentVersionIsTheContractsAndNotTheFileFormats(t *testing.T) {
	// 39 is what the pinned XNA constant says. CNA's own fixture generator
	// writes 46 into an .xgs header, and this project's fixture writer writes
	// 46 too -- because the two numbers answer different questions. A
	// projection that reported the file-format number would be reporting
	// something the member does not mean.
	if AudioEngineContentVersion != 39 {
		t.Fatalf("AudioEngine.ContentVersion = %d, want 39", AudioEngineContentVersion)
	}
}

func TestTheZeroAudioCategoryIsLegalAndAnswersWithoutAHandle(t *testing.T) {
	var left, right AudioCategory
	equal, err := left.EqualsByAudioCategory(right)
	if err != nil {
		t.Fatalf("comparing two zero categories: %v", err)
	}
	if !equal {
		t.Fatal("two zero AudioCategory values compared unequal")
	}
	// The equality answers BEFORE reaching a handle, which is what makes the
	// zero value usable at all: there is no handle to ask.
	notEqual, err := AudioCategoryOperatorInequalityByAudioCategoryAndAudioCategory(left, right)
	if err != nil {
		t.Fatalf("op_Inequality on two zero categories: %v", err)
	}
	if notEqual {
		t.Fatal("op_Inequality reported two zero categories as different")
	}
	// Equals(object) on a value of another type is false and not an error,
	// which is what the reference's `is` test gives.
	other, err := left.EqualsByObject("not a category")
	if err != nil {
		t.Fatalf("Equals(object) on a foreign value: %v", err)
	}
	if other {
		t.Fatal("Equals(object) matched a string")
	}
	if nilEqual, err := left.EqualsByObject(nil); err != nil || nilEqual {
		t.Fatalf("Equals(nil) = %v, %v; want false, nil", nilEqual, err)
	}
}

// TestEveryZeroAudioCategoryMemberThatNeedsAHandleRefuses pins BOTH halves of
// each guard: that the member refuses, and that it refuses as a NIL one rather
// than a disposed one.
//
// The second half is the part a reader would drop. Dropping the guard still
// produces a refusal -- interop.Resource.liveHandle answers ErrDisposed on a
// nil receiver -- so a test that only asserted "some error" would pass with no
// guard at all. But `default(AudioCategory)` was never constructed and never
// disposed, and telling a caller their category "has already been disposed" is
// a false statement about a value they just declared.
func TestEveryZeroAudioCategoryMemberThatNeedsAHandleRefuses(t *testing.T) {
	var category AudioCategory
	_, err := category.Name()
	if err == nil {
		t.Fatal("Name answered on a zero category")
	}
	if !errors.Is(err, errXactNil) {
		t.Fatalf("a zero category refused with %q; want the nil refusal, not a disposal one", err)
	}
	if _, err := category.ToString(); err == nil {
		t.Fatal("ToString answered on a zero category")
	}
	if _, err := category.GetHashCode(); err == nil {
		t.Fatal("GetHashCode answered on a zero category")
	}
	if err := category.SetVolume(0.5); err == nil {
		t.Fatal("SetVolume ran on a zero category")
	}
	if err := category.Pause(); err == nil {
		t.Fatal("Pause ran on a zero category")
	}
	if err := category.Resume(); err == nil {
		t.Fatal("Resume ran on a zero category")
	}
	if err := category.Stop(AudioStopOptionsImmediate); err == nil {
		t.Fatal("Stop ran on a zero category")
	}
}

func TestNilXactReceiversRefuseRatherThanPanicking(t *testing.T) {
	var engine *AudioEngine
	var soundBank *SoundBank
	var waveBank *WaveBank
	var cue *Cue

	// The four IsDisposed getters are the exception: they are `ldfld` in the
	// reference and infallible here, so a nil receiver answers TRUE rather
	// than refusing. A nil object is not usable, and saying so is the only
	// answer an infallible bool can give.
	if !engine.IsDisposed() || !soundBank.IsDisposed() || !waveBank.IsDisposed() || !cue.IsDisposed() {
		t.Fatal("a nil XACT object reported IsDisposed false")
	}

	if err := engine.Update(); err == nil {
		t.Fatal("Update ran on a nil engine")
	}
	if _, err := engine.GetCategory("Music"); err == nil {
		t.Fatal("GetCategory answered on a nil engine")
	}
	if _, err := engine.GetGlobalVariable("v"); err == nil {
		t.Fatal("GetGlobalVariable answered on a nil engine")
	}
	if err := engine.SetGlobalVariable("v", 1); err == nil {
		t.Fatal("SetGlobalVariable ran on a nil engine")
	}
	if _, err := engine.RendererDetails(); err == nil {
		t.Fatal("RendererDetails answered on a nil engine")
	}
	if _, err := soundBank.IsInUse(); err == nil {
		t.Fatal("IsInUse answered on a nil sound bank")
	}
	if _, err := soundBank.GetCue("Tone"); err == nil {
		t.Fatal("GetCue answered on a nil sound bank")
	}
	if err := soundBank.PlayCueByString("Tone"); err == nil {
		t.Fatal("PlayCue ran on a nil sound bank")
	}
	if _, err := waveBank.IsInUse(); err == nil {
		t.Fatal("IsInUse answered on a nil wave bank")
	}
	if _, err := waveBank.IsPrepared(); err == nil {
		t.Fatal("IsPrepared answered on a nil wave bank")
	}
	if _, err := cue.Name(); err == nil {
		t.Fatal("Name answered on a nil cue")
	}
	if err := cue.Play(); err == nil {
		t.Fatal("Play ran on a nil cue")
	}
	if _, err := cue.IsPlaying(); err == nil {
		t.Fatal("IsPlaying answered on a nil cue")
	}
}

// TestADisposedXactObjectRefusesAndNamesItsOwnType is the claim that separates
// the four disposal refusals from each other: the reference's
// ObjectDisposedException carries GetType().Name, so a caller can tell WHICH
// object refused. A shared message would lose that.
func TestADisposedXactObjectRefusesAndNamesItsOwnType(t *testing.T) {
	engine := &AudioEngine{disposed: true}
	soundBank := &SoundBank{disposed: true}
	waveBank := &WaveBank{disposed: true}
	cue := &Cue{disposed: true}

	for _, probe := range []struct {
		typeName string
		err      error
	}{
		{"AudioEngine", engine.Update()},
		{"SoundBank", errorOf2(soundBank.IsInUse())},
		{"WaveBank", errorOf2(waveBank.IsPrepared())},
		{"Cue", cue.Play()},
	} {
		if probe.err == nil {
			t.Fatalf("a disposed %s did not refuse", probe.typeName)
		}
		if !strings.Contains(probe.err.Error(), probe.typeName) {
			t.Fatalf("a disposed %s refused with %q, which does not name the type",
				probe.typeName, probe.err)
		}
		if !strings.Contains(probe.err.Error(), objectDisposedMessage) {
			t.Fatalf("a disposed %s refused with %q, which is not the reference's message",
				probe.typeName, probe.err)
		}
	}
}

func errorOf2(_ bool, err error) error { return err }

// TestDisposalIsIdempotentAndTheFinalizerPathSkipsChildren pins both halves of
// the Dispose(bool) contract: `true` walks the managed children, `false` -- the
// CLR finalizer path -- must not touch them, because in the reference they may
// already have been collected.
func TestDisposalIsIdempotentAndTheFinalizerPathSkipsChildren(t *testing.T) {
	walked := 0
	engine := &AudioEngine{disposed: true}
	engine.children = []interface{ disposeFromEngine() error }{
		countingChild{count: &walked},
	}
	// Already disposed: neither branch may walk anything.
	if err := engine.DisposeByNone(); err != nil {
		t.Fatalf("a second DisposeByNone: %v", err)
	}
	if err := engine.DisposeByBoolean(false); err != nil {
		t.Fatalf("a second DisposeByBoolean(false): %v", err)
	}
	if walked != 0 {
		t.Fatalf("a disposed engine walked %d children", walked)
	}

	// Fresh, finalizer path: the children are dropped WITHOUT being walked.
	live := &AudioEngine{}
	live.children = []interface{ disposeFromEngine() error }{countingChild{count: &walked}}
	if err := live.DisposeByBoolean(false); err != nil {
		t.Fatalf("the finalizer path: %v", err)
	}
	if walked != 0 {
		t.Fatalf("the finalizer path walked %d children; it must touch none", walked)
	}
	if !live.IsDisposed() {
		t.Fatal("the finalizer path left the engine undisposed")
	}
}

type countingChild struct{ count *int }

func (c countingChild) disposeFromEngine() error {
	*c.count++
	return nil
}

// TestFlatteningMatchesTheOrderTheBridgeFills is the contract between this
// package and bridge.c, and it is the one thing here a wrong answer would make
// silently WRONG rather than refused: a listener whose Up and Forward were
// swapped would still play, from the wrong direction.
func TestFlatteningMatchesTheOrderTheBridgeFills(t *testing.T) {
	listener := NewAudioListener()
	listener.SetForward(framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3))
	listener.SetPosition(framework.NewVector3BySingleAndSingleAndSingle(4, 5, 6))
	listener.SetUp(framework.NewVector3BySingleAndSingleAndSingle(7, 8, 9))
	listener.SetVelocity(framework.NewVector3BySingleAndSingleAndSingle(10, 11, 12))
	flat := flattenListener(listener)
	if len(flat) != 12 {
		t.Fatalf("a flattened listener is %d floats, want 12", len(flat))
	}
	// The values reach the bridge UNCHANGED. AudioListener stores each vector
	// with its Z negated and negates it back on the way out -- flipHandedness
	// is its own inverse -- so the public accessor pair is an identity round
	// trip and what CNA receives is the right-handed XNA vector the caller set.
	// This is the same thing SoundEffectInstance::Apply3D sends, which is what
	// keeps the two positional families from disagreeing.
	want := []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	for index, value := range want {
		if flat[index] != value {
			t.Fatalf("flattened listener[%d] = %v, want %v (full: %v)", index, flat[index], value, flat)
		}
	}

	emitter := NewAudioEmitter()
	emitter.SetForward(framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3))
	emitter.SetPosition(framework.NewVector3BySingleAndSingleAndSingle(4, 5, 6))
	emitter.SetUp(framework.NewVector3BySingleAndSingleAndSingle(7, 8, 9))
	emitter.SetVelocity(framework.NewVector3BySingleAndSingleAndSingle(10, 11, 12))
	if err := emitter.SetDopplerScale(2); err != nil {
		t.Fatalf("setting the doppler scale: %v", err)
	}
	emitterFlat := flattenEmitter(emitter)
	if len(emitterFlat) != 13 {
		t.Fatalf("a flattened emitter is %d floats, want 13", len(emitterFlat))
	}
	// The DopplerScale leads, which is what the emitter's C struct declares and
	// what makes the emitter thirteen floats where a listener is twelve.
	if emitterFlat[0] != 2 {
		t.Fatalf("flattened emitter[0] = %v, want the doppler scale 2", emitterFlat[0])
	}
	for index, value := range want {
		if emitterFlat[index+1] != value {
			t.Fatalf("flattened emitter[%d] = %v, want %v", index+1, emitterFlat[index+1], value)
		}
	}
}

// TestApply3DRefusesANilArgumentBeforeReachingTheRuntime pins the Go-only
// guards. The reference dereferences both arguments unguarded and throws
// NullReferenceException; a Go nil cannot produce that, and the guard is what
// keeps it from being a panic instead.
func TestApply3DRefusesANilArgumentBeforeReachingTheRuntime(t *testing.T) {
	cue := &Cue{}
	if err := cue.Apply3D(nil, NewAudioEmitter()); err == nil {
		t.Fatal("Apply3D accepted a nil listener")
	}
	if err := cue.Apply3D(NewAudioListener(), nil); err == nil {
		t.Fatal("Apply3D accepted a nil emitter")
	}
	bank := &SoundBank{}
	if err := bank.PlayCueByStringAndAudioListenerAndAudioEmitter("Tone", nil, NewAudioEmitter()); err == nil {
		t.Fatal("the positional PlayCue accepted a nil listener")
	}
	if err := bank.PlayCueByStringAndAudioListenerAndAudioEmitter("Tone", NewAudioListener(), nil); err == nil {
		t.Fatal("the positional PlayCue accepted a nil emitter")
	}
}

// TestABankConstructorRefusesADisposedEngineByName pins the ORDER the
// reference's constructor checks in: the engine first, before the filename is
// looked at.
func TestABankConstructorRefusesADisposedEngineByName(t *testing.T) {
	if _, err := NewSoundBank(nil, "sounds.xsb"); err == nil {
		t.Fatal("NewSoundBank accepted a nil engine")
	}
	disposed := &AudioEngine{disposed: true}
	_, err := NewSoundBank(disposed, "sounds.xsb")
	if err == nil {
		t.Fatal("NewSoundBank accepted a disposed engine")
	}
	if !strings.Contains(err.Error(), "AudioEngine") {
		t.Fatalf("the refusal %q does not name AudioEngine", err)
	}
	if _, err := NewWaveBankByAudioEngineAndString(nil, "waves.xwb"); err == nil {
		t.Fatal("the non-streaming WaveBank constructor accepted a nil engine")
	}
	if _, err := NewWaveBankByAudioEngineAndStringAndInt32AndInt16(nil, "waves.xwb", 0, 0); err == nil {
		t.Fatal("the streaming WaveBank constructor accepted a nil engine")
	}
}

// TestTheDisposingEventAcceptsAndRemovesHandlersWithoutAnEngine proves the
// managed half of the four events works before any native registration exists,
// which is what lets a caller subscribe and unsubscribe on an object whose
// runtime is gone.
func TestTheDisposingEventAcceptsAndRemovesHandlersWithoutAnEngine(t *testing.T) {
	engine := &AudioEngine{}
	raised := 0
	subscription, err := engine.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		raised++
		return nil
	})
	if err != nil {
		t.Fatalf("adding a Disposing handler: %v", err)
	}
	if err := engine.disposing.source.Raise(engine, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("raising Disposing: %v", err)
	}
	if raised != 1 {
		t.Fatalf("the handler ran %d times, want 1", raised)
	}
	if err := engine.RemoveDisposingHandler(subscription); err != nil {
		t.Fatalf("removing a Disposing handler: %v", err)
	}
	if err := engine.disposing.source.Raise(engine, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("raising Disposing after removal: %v", err)
	}
	if raised != 1 {
		t.Fatalf("a removed handler ran again: %d", raised)
	}
	// Releasing a subscription that was never made is harmless, which is what
	// lets Dispose call it unconditionally.
	if err := engine.disposing.release(); err != nil {
		t.Fatalf("releasing an unsubscribed event: %v", err)
	}
}
