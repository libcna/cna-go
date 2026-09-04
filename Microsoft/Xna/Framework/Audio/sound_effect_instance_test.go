package audio

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/dispatcher"
	"github.com/openeggbert/cna-go/internal/interop"
)

// SoundEffectInstance's managed half is its guards, and the pair that decides
// the type's behaviour is set_Pan against Apply3D: they are the two halves of
// one mode latch and each refuses when the other has won.

// newManagedInstance is the object CreateInstance builds minus its native half,
// so the guards that run before any native call can be exercised on their own.
func newManagedInstance() *SoundEffectInstance {
	return &SoundEffectInstance{currentVolume: 1}
}

// TestInstanceScalarDefaultsAreTheConstructorsStores pins the three values the
// reference's constructor writes through its own public setters.
func TestInstanceScalarDefaultsAreTheConstructorsStores(t *testing.T) {
	instance := newSoundEffectInstance(nil, nil)
	if instance.Volume() != 1 || instance.Pitch() != 0 || instance.Pan() != 0 {
		t.Fatalf("a fresh instance = %v/%v/%v, want 1/0/0",
			instance.Volume(), instance.Pitch(), instance.Pan())
	}
	if instance.IsLooped() || instance.IsDisposed() {
		t.Fatal("a fresh instance is looped or disposed")
	}
}

// TestInstanceScalarRangesAreTheReferences pins all three ranges and the NaN
// behaviour they share, which is not the behaviour the static setters share.
func TestInstanceScalarRangesAreTheReferences(t *testing.T) {
	nan := float32(math.NaN())
	for _, row := range []struct {
		name string
		set  func(*SoundEffectInstance, float32) error
		bad  []float32
		good []float32
	}{
		{"Volume", (*SoundEffectInstance).SetVolume,
			[]float32{-0.001, 1.001, nan}, []float32{0, 1, 0.5}},
		{"Pitch", (*SoundEffectInstance).SetPitch,
			[]float32{-1.001, 1.001, nan}, []float32{-1, 1, 0}},
		{"Pan", (*SoundEffectInstance).SetPan,
			[]float32{-1.001, 1.001, nan}, []float32{-1, 1, 0}},
	} {
		for _, bad := range row.bad {
			instance := newManagedInstance()
			if err := row.set(instance, bad); !errors.Is(err, errAudioArgumentOutOfRange) {
				t.Fatalf("Set%s(%v) = %v, want a range refusal", row.name, bad, err)
			}
		}
		// A value INSIDE the range gets past the range check and reaches the
		// nil-resource refusal instead, which is what distinguishes the two.
		for _, good := range row.good {
			instance := newManagedInstance()
			if err := row.set(instance, good); errors.Is(err, errAudioArgumentOutOfRange) {
				t.Fatalf("Set%s(%v) was refused for its range", row.name, good)
			}
		}
	}
}

// TestPanAndApply3DAreTheTwoHalvesOfOneModeLatch is the type's central claim.
//
//	set_Pan:   if (!isPacketSubmitted) is3d = false;  if (is3d) throw InvalidPanCall
//	Apply3D:   if (!isPacketSubmitted) is3d = true;   if (!is3d) throw InvalidApply3DCall
//
// Mirror images, and the `isPacketSubmitted` guard is what lets the mode be
// changed before the first Play and fixes it afterwards.
func TestPanAndApply3DAreTheTwoHalvesOfOneModeLatch(t *testing.T) {
	// Before a packet is submitted, Apply3D SETS the 3D mode.
	instance := newManagedInstance()
	if instance.is3d {
		t.Fatal("a fresh instance is already 3D")
	}
	_ = instance.Apply3DByAudioListenerAndAudioEmitter(NewAudioListener(), NewAudioEmitter())
	if !instance.is3d {
		t.Fatal("Apply3D did not set the 3D mode before a packet was submitted")
	}
	// And set_Pan CLEARS it again, because no packet has been submitted.
	if err := instance.SetPan(0); errors.Is(err, errAudioInvalidOperation) {
		t.Fatal("SetPan was refused on an instance no packet has been submitted for")
	}
	if instance.is3d {
		t.Fatal("SetPan did not clear the 3D mode")
	}

	// Once a packet HAS been submitted the mode is fixed, and the member for
	// the other mode refuses -- with its own message.
	instance = newManagedInstance()
	instance.isPacketSubmitted = true
	instance.is3d = true
	err := instance.SetPan(0)
	if !errors.Is(err, errAudioInvalidOperation) || !strings.Contains(err.Error(), invalidPanCall) {
		t.Fatalf("SetPan on a submitted 3D instance = %v, want the InvalidPanCall refusal", err)
	}

	instance = newManagedInstance()
	instance.isPacketSubmitted = true
	instance.is3d = false
	err = instance.Apply3DByAudioListenerAndAudioEmitter(NewAudioListener(), NewAudioEmitter())
	if !errors.Is(err, errAudioInvalidOperation) || !strings.Contains(err.Error(), invalidApply3DCall) {
		t.Fatalf("Apply3D on a submitted 2D instance = %v, want the InvalidApply3DCall refusal", err)
	}
}

// TestIsLoopedRefusesAfterThePacketIsSubmitted pins the THIRD "before the first
// Play" precondition, and the flag that makes all three reachable.
//
// The projection was missing both when it was first written: SetIsLooped had no
// guard at all, and isPacketSubmitted was never set, so the mode latch could
// never latch either. The native scenario found it because CNA enforces the
// same precondition -- INVALID_STATE "after playback has begun" -- and these
// rows are what stop it coming back.
func TestIsLoopedRefusesAfterThePacketIsSubmitted(t *testing.T) {
	instance := newManagedInstance()
	// Before the packet: the guard passes and the call reaches the nil-resource
	// refusal, which is what distinguishes "allowed" from "refused" here.
	if err := instance.SetIsLooped(true); errors.Is(err, errAudioInvalidOperation) {
		t.Fatalf("SetIsLooped before any Play = %v; the packet has not been submitted", err)
	}
	instance.isPacketSubmitted = true
	err := instance.SetIsLooped(true)
	if !errors.Is(err, errAudioInvalidOperation) || !strings.Contains(err.Error(), invalidIsLoopedCall) {
		t.Fatalf("SetIsLooped after the packet = %v, want the InvalidIsLoopedCall refusal", err)
	}
	// And the disposal check runs FIRST, so a disposed instance reports
	// disposal rather than the packet.
	disposed := newManagedInstance()
	disposed.isPacketSubmitted = true
	_ = disposed.DisposeByNone()
	if err := disposed.SetIsLooped(true); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatalf("SetIsLooped on a disposed instance = %v", err)
	}
}

// TestPlaySubmitsThePacket pins the store that fixes the instance's mode. It is
// the one place isPacketSubmitted becomes true, and CNA names the same moment:
// "the reference implementation submits its audio packet on the first ..._play".
func TestPlaySubmitsThePacket(t *testing.T) {
	instance := newManagedInstance()
	if instance.isPacketSubmitted {
		t.Fatal("a fresh instance has already submitted its packet")
	}
	// With no resource the Play fails, and the flag must NOT be set -- the
	// reference submits the packet as part of a successful play.
	if err := instance.Play(); err == nil {
		t.Fatal("Play succeeded with no resource")
	}
	if instance.isPacketSubmitted {
		t.Fatal("a FAILED Play submitted the packet")
	}
}

// TestPanChecksItsModeBeforeItsRange pins the guard ORDER: an instance that is
// latched 3D refuses a pan of 5 for being 3D, not for being out of range.
func TestPanChecksItsModeBeforeItsRange(t *testing.T) {
	instance := newManagedInstance()
	instance.isPacketSubmitted = true
	instance.is3d = true
	err := instance.SetPan(5)
	if !strings.Contains(err.Error(), invalidPanCall) {
		t.Fatalf("SetPan(5) on a 3D instance = %v; the MODE guard runs before the range check", err)
	}
}

// TestDisposedInstanceRefusesEveryMember pins that the disposal check comes
// first everywhere, and that disposal is idempotent.
func TestDisposedInstanceRefusesEveryMember(t *testing.T) {
	instance := newManagedInstance()
	if err := instance.DisposeByNone(); err != nil {
		t.Fatal(err)
	}
	if !instance.IsDisposed() {
		t.Fatal("the instance is not disposed after Dispose")
	}
	if err := instance.DisposeByNone(); err != nil {
		t.Fatalf("a second Dispose = %v; the reference's flag makes it idempotent", err)
	}
	for name, call := range map[string]func() error{
		"Play":        instance.Play,
		"Pause":       instance.Pause,
		"Resume":      instance.Resume,
		"Stop":        instance.StopByNone,
		"SetVolume":   func() error { return instance.SetVolume(1) },
		"SetPitch":    func() error { return instance.SetPitch(0) },
		"SetPan":      func() error { return instance.SetPan(0) },
		"SetIsLooped": func() error { return instance.SetIsLooped(true) },
		"Apply3D": func() error {
			return instance.Apply3DByAudioListenerAndAudioEmitter(NewAudioListener(), NewAudioEmitter())
		},
	} {
		if err := call(); !errors.Is(err, errAudioObjectDisposed) {
			t.Fatalf("%s on a disposed instance = %v, want the disposal refusal", name, err)
		}
	}
	if _, err := instance.State(); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatal("State answered on a disposed instance")
	}
	// The message is the reference's, and the type name is the object's.
	err := instance.Play()
	if !strings.Contains(err.Error(), objectDisposedMessage) ||
		!strings.Contains(err.Error(), "SoundEffectInstance") {
		t.Fatalf("the disposal refusal said %q", err)
	}
}

// TestNilInstanceAnswersItsZeroValues pins that every member on a zero value
// answers rather than panicking.
func TestNilInstanceAnswersItsZeroValues(t *testing.T) {
	var instance *SoundEffectInstance
	if !instance.IsDisposed() || instance.Volume() != 0 || instance.Pitch() != 0 ||
		instance.Pan() != 0 || instance.IsLooped() {
		t.Fatal("a nil instance did not answer its zero values")
	}
	for name, call := range map[string]func() error{
		"Play":    instance.Play,
		"Pause":   instance.Pause,
		"Dispose": instance.DisposeByNone,
	} {
		if err := call(); !errors.Is(err, errSoundEffectInstanceNil) {
			t.Fatalf("%s on a nil instance = %v", name, err)
		}
	}
}

// TestSoundEffectGuardsRunWithoutADevice pins the constructor guard order --
// including the place the two constructors DIVERGE -- and Play's dispatcher
// precondition, all without reaching a runtime.
func TestSoundEffectGuardsRunWithoutADevice(t *testing.T) {
	// The three-argument constructor checks the BUFFER first, so a call with
	// both a bad buffer and a bad sample rate reports the buffer.
	_, err := NewSoundEffectBySliceOfByteAndInt32AndAudioChannels(nil, 7999, AudioChannelsMono)
	if err == nil || !strings.Contains(err.Error(), invalidAudioBuffer) {
		t.Fatalf("a nil buffer with a bad rate = %v, want the buffer message", err)
	}
	// The seven-argument one has NO check of its own, so the same pair reports
	// the SAMPLE RATE. That divergence is the reference's.
	_, err = NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
		nil, 0, 0, 7999, AudioChannelsMono, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "sampleRate") {
		t.Fatalf("the seven-argument constructor with the same pair = %v, want sampleRate", err)
	}

	// The three ALIGNMENT refusals, each with its own message. BlockAlign is
	// channels * 2, so an odd length, an odd offset and an odd count are each
	// refused separately.
	aligned := make([]byte, 8)
	for _, row := range []struct {
		name                  string
		buffer                []byte
		offset, count         int32
		loopStart, loopLength int32
		want                  string
	}{
		{"odd buffer length", make([]byte, 7), 0, 6, 0, 0, invalidAudioBuffer},
		{"odd offset", aligned, 1, 4, 0, 0, invalidAudioBufferOffset},
		{"offset at the end", aligned, 8, 4, 0, 0, invalidAudioBufferOffset},
		{"odd count", aligned, 0, 5, 0, 0, invalidOffsetCountLength},
		{"count past the end", aligned, 4, 6, 0, 0, invalidOffsetCountLength},
		{"zero count", aligned, 0, 0, 0, 0, invalidOffsetCountLength},
		{"negative loop start", aligned, 0, 8, -1, 0, invalidLoopRegion},
		{"negative loop length", aligned, 0, 8, 0, -1, invalidLoopRegion},
		{"loop past the frames", aligned, 0, 8, 0, 5, invalidLoopRegion},
	} {
		_, err := NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
			row.buffer, row.offset, row.count, 44100, AudioChannelsMono, row.loopStart, row.loopLength)
		if err == nil || !strings.Contains(err.Error(), row.want) {
			t.Fatalf("%s = %v, want %q", row.name, err, row.want)
		}
	}
	// A four-frame buffer with a loop covering exactly the frames is accepted
	// as far as the guards go, and stops at the no-runtime refusal.
	_, err = NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
		aligned, 0, 8, 44100, AudioChannelsMono, 0, 4)
	if err != nil && !errors.Is(err, errSoundEffectNoRunningGame) {
		t.Fatalf("a valid call = %v, want only the no-runtime refusal", err)
	}
}

// TestFromStreamRefusesANilStream pins FromStream's one managed guard.
func TestFromStreamRefusesANilStream(t *testing.T) {
	if _, err := SoundEffectFromStream(nil); !errors.Is(err, errAudioArgumentNull) {
		t.Fatalf("FromStream(nil) = %v", err)
	}
}

// TestSoundEffectNameRefusesTheEmptyString pins set_Name's IsNullOrEmpty guard,
// and the consequence: the constructor's own String.Empty cannot be set again.
func TestSoundEffectNameRefusesTheEmptyString(t *testing.T) {
	effect := &SoundEffect{}
	if effect.Name() != "" {
		t.Fatal("a fresh effect has a name")
	}
	if err := effect.SetName(""); !errors.Is(err, errAudioArgumentNull) {
		t.Fatalf("SetName(\"\") = %v; the guard is IsNullOrEmpty", err)
	}
	if err := effect.SetName("thunder"); err != nil {
		t.Fatal(err)
	}
	if effect.Name() != "thunder" {
		t.Fatalf("Name = %q", effect.Name())
	}
	// get_Name has NO disposal check, so a disposed effect still answers.
	effect.disposed = true
	if effect.Name() != "thunder" {
		t.Fatal("a disposed effect stopped answering its name")
	}
	if err := effect.SetName("other"); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatalf("SetName on a disposed effect = %v", err)
	}
	// Duration and IsDisposed answer after disposal too.
	if !effect.IsDisposed() {
		t.Fatal("IsDisposed did not answer")
	}
	_ = effect.Duration()
}

// TestSoundStateMappingHasStoppedAsItsFallback pins the branch CNA never takes
// on a qualified artifact. The reference's get_State seeds a local with
// SoundState.Stopped and only two tests can move it, so every value the runtime
// might report that XNA has no name for -- including the 4 the local starts at
// when the native call fails -- reads as Stopped.
func TestSoundStateMappingHasStoppedAsItsFallback(t *testing.T) {
	if got := soundStateFromNative(nativeSoundStatePlaying); got != SoundStatePlaying {
		t.Fatalf("CNA_SOUND_STATE_PLAYING mapped to %v", got)
	}
	if got := soundStateFromNative(nativeSoundStatePaused); got != SoundStatePaused {
		t.Fatalf("CNA_SOUND_STATE_PAUSED mapped to %v", got)
	}
	if got := soundStateFromNative(nativeSoundStateStopped); got != SoundStateStopped {
		t.Fatalf("CNA_SOUND_STATE_STOPPED mapped to %v", got)
	}
	// Everything else, which is the branch this test exists for.
	for _, unknown := range []uint32{3, 4, 8, 255, 1 << 31} {
		if got := soundStateFromNative(unknown); got != SoundStateStopped {
			t.Fatalf("an unrecognised state %d mapped to %v, want Stopped", unknown, got)
		}
	}
}

// TestPlayDoesNotSubmitThePacketWhenTheNativeCallFails needs an upload that
// gets past every managed guard and then fails, which no device-free test
// produces by accident. A zero interop.Resource is real enough to be reached
// and wrong enough to refuse -- its kind is not an instance's, so liveHandle
// rejects it -- and that is exactly the failure this claim needs.
func TestPlayDoesNotSubmitThePacketWhenTheNativeCallFails(t *testing.T) {
	instance := &SoundEffectInstance{currentVolume: 1, resource: &interop.Resource{}}
	if err := instance.Play(); err == nil {
		t.Fatal("Play succeeded through a resource of the wrong kind; the test's premise is gone")
	}
	if instance.isPacketSubmitted {
		t.Fatal("a FAILED Play submitted the packet; the reference submits it as part of a successful play")
	}
	// The same ordering claim for the scalars: the store follows the native
	// write, so a refused set leaves the getter reporting what is in effect.
	if err := instance.SetVolume(0.25); err == nil {
		t.Fatal("SetVolume succeeded through a bad resource")
	}
	if got := instance.Volume(); got != 1 {
		t.Fatalf("Volume = %v after a refused set, want the previous 1", got)
	}
	if err := instance.SetPitch(0.5); err == nil {
		t.Fatal("SetPitch succeeded through a bad resource")
	}
	if got := instance.Pitch(); got != 0 {
		t.Fatalf("Pitch = %v after a refused set, want the previous 0", got)
	}
	if err := instance.SetPan(0.5); err == nil {
		t.Fatal("SetPan succeeded through a bad resource")
	}
	if got := instance.Pan(); got != 0 {
		t.Fatalf("Pan = %v after a refused set", got)
	}
	if err := instance.SetIsLooped(true); err == nil {
		t.Fatal("SetIsLooped succeeded through a bad resource")
	}
	if instance.IsLooped() {
		t.Fatal("IsLooped stored after a refused set")
	}
}

// TestPlayRefusesUntilTheDispatcherHasRun pins SoundEffect::Play's FIRST
// statement, which runs ahead of its lock and its disposal check:
//
//	if (!FrameworkDispatcher.UpdateCalledAtLeastOnce)
//	    throw new InvalidOperationException(FrameworkResources.CallFrameworkDispatcherUpdate);
//
// A fire-and-forget sound needs the dispatcher because the dispatcher is what
// reclaims its instance, so the guard is load-bearing rather than ceremonial.
//
// The flag is process-wide and nothing resets it, so this test depends on
// nothing in this package having called FrameworkDispatcher.Update -- which
// nothing does, because the projected dispatcher lives in another package and
// this one never pumps it.
func TestPlayRefusesUntilTheDispatcherHasRun(t *testing.T) {
	if dispatcher.HasRun() {
		t.Skip("the dispatcher has already been pumped in this process")
	}
	effect := &SoundEffect{}
	_, err := effect.PlayByNone()
	if !errors.Is(err, errAudioInvalidOperation) ||
		!strings.Contains(err.Error(), callFrameworkDispatcherUpdate) {
		t.Fatalf("Play before any dispatcher update = %v, want the CallFrameworkDispatcherUpdate refusal", err)
	}
	// It runs ahead of the DISPOSAL check too, so a disposed effect reports the
	// dispatcher rather than its disposal.
	effect.disposed = true
	_, err = effect.PlayBySingleAndSingleAndSingle(1, 0, 0)
	if !strings.Contains(err.Error(), callFrameworkDispatcherUpdate) {
		t.Fatalf("Play on a disposed effect = %v; the dispatcher guard is the FIRST statement", err)
	}
}
