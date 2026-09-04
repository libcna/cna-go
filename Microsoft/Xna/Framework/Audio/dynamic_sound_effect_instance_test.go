package audio

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// DynamicSoundEffectInstance's managed half is its constructor guards and the
// two overridden accessors that REFUSE. Everything else forwards to a base the
// previous milestone already pins.

// newManagedDynamicInstance is the object the constructor builds minus its
// native half, with the binding the constructor installs -- because every claim
// below is about which object answers.
func newManagedDynamicInstance() *DynamicSoundEffectInstance {
	base := &SoundEffectInstance{currentVolume: 1}
	dynamic := &DynamicSoundEffectInstance{
		instance:   base,
		sampleRate: 44100,
		channels:   AudioChannelsMono,
	}
	base.bindDerived(dynamic)
	return dynamic
}

// TestDynamicInstanceConstructorGuardsAreTheReferences pins both ranges and the
// fact that they are the SAME bounds SoundEffect's FromBuffer uses.
func TestDynamicInstanceConstructorGuardsAreTheReferences(t *testing.T) {
	for _, bad := range []int32{7999, 48001, 0, -1} {
		_, err := NewDynamicSoundEffectInstance(bad, AudioChannelsMono)
		if err == nil || !strings.Contains(err.Error(), "sampleRate") {
			t.Fatalf("sampleRate %d = %v, want a sampleRate refusal", bad, err)
		}
	}
	for _, bad := range []AudioChannels{0, 3, -1} {
		_, err := NewDynamicSoundEffectInstance(44100, bad)
		if err == nil || !strings.Contains(err.Error(), "channels") {
			t.Fatalf("channels %d = %v, want a channels refusal", bad, err)
		}
	}
	// The sampleRate check runs FIRST, so a call with both wrong reports it.
	_, err := NewDynamicSoundEffectInstance(7999, AudioChannels(9))
	if err == nil || !strings.Contains(err.Error(), "sampleRate") {
		t.Fatalf("both wrong = %v, want sampleRate", err)
	}
	// Both bounds are INCLUSIVE, so a legal pair gets past the guards and
	// stops at the no-runtime refusal.
	for _, rate := range []int32{8000, 48000} {
		_, err := NewDynamicSoundEffectInstance(rate, AudioChannelsStereo)
		if err != nil && !errors.Is(err, errSoundEffectNoRunningGame) {
			t.Fatalf("sampleRate %d = %v, want only the no-runtime refusal", rate, err)
		}
	}
}

// TestDynamicInstanceCanNeverLoop is the type's central claim: it OVERRIDES
// both IsLooped accessors and both of them refuse to do what the base's do.
func TestDynamicInstanceCanNeverLoop(t *testing.T) {
	instance := newManagedDynamicInstance()
	looped, err := instance.IsLooped()
	if err != nil {
		t.Fatal(err)
	}
	if looped {
		t.Fatal("a fresh dynamic instance reports itself looped")
	}
	// TRUE is refused, with the type's own message.
	err = instance.SetIsLooped(true)
	if !errors.Is(err, errAudioInvalidOperation) ||
		!strings.Contains(err.Error(), invalidDynamicIsLoopedCall) {
		t.Fatalf("SetIsLooped(true) = %v, want the InvalidDynamicIsLoopedCall refusal", err)
	}
	// FALSE is accepted and stores NOTHING -- the reference's body returns.
	if err := instance.SetIsLooped(false); err != nil {
		t.Fatalf("SetIsLooped(false) = %v, want acceptance", err)
	}
	looped, err = instance.IsLooped()
	if err != nil || looped {
		t.Fatalf("IsLooped after setting false = %v, %v", looped, err)
	}
	// The BASE's flag is untouched, which is what "stores nothing" means.
	if instance.instance.looped {
		t.Fatal("SetIsLooped(false) wrote through to the base's field")
	}
	// And the getter does not READ that field either: a base whose flag is set
	// -- which only the base's own setter could do -- still reports false
	// through the override. A projection that forwarded to the base would
	// answer true here and false in every other row above.
	instance.instance.looped = true
	looped, err = instance.IsLooped()
	if err != nil {
		t.Fatal(err)
	}
	if looped {
		t.Fatal("IsLooped read the base's field; the override returns a CONSTANT false")
	}
	instance.instance.looped = false
	// The disposal check runs FIRST in both accessors.
	_ = instance.instance.DisposeByBoolean(true)
	if _, err := instance.IsLooped(); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatalf("IsLooped on a disposed instance = %v", err)
	}
	if err := instance.SetIsLooped(false); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatalf("SetIsLooped(false) on a disposed instance = %v; disposal is checked first", err)
	}
}

// TestDynamicInstanceNamesItselfWhenDisposed is the IDENTITY claim, and it is
// the one a bare literal would silently break: every disposal refusal in the
// family carries GetType().Name, so a disposed DynamicSoundEffectInstance must
// name ITSELF and not its base.
func TestDynamicInstanceNamesItselfWhenDisposed(t *testing.T) {
	instance := newManagedDynamicInstance()
	_ = instance.instance.DisposeByBoolean(true)
	_, err := instance.IsLooped()
	if err == nil {
		t.Fatal("a disposed instance was not refused")
	}
	if !strings.Contains(err.Error(), "DynamicSoundEffectInstance") {
		t.Fatalf("the disposal refusal said %q; the reference names the object's own type", err)
	}
	// The BASE alone still names itself, which is what makes the resolution a
	// resolution rather than a rename.
	base := &SoundEffectInstance{}
	_ = base.DisposeByBoolean(true)
	if err := base.Play(); err == nil || !strings.Contains(err.Error(), "SoundEffectInstance") {
		t.Fatalf("a disposed base said %v", err)
	}
	if strings.Contains(base.objectDisposed().Error(), "Dynamic") {
		t.Fatal("the base named a derived type it does not have")
	}
}

// TestDynamicInstanceSubmitBufferSharesFromBuffersChecks pins that the four
// window checks are the SAME four, with the same four messages.
func TestDynamicInstanceSubmitBufferSharesFromBuffersChecks(t *testing.T) {
	instance := newManagedDynamicInstance()
	aligned := make([]uint8, 8)
	for _, row := range []struct {
		name          string
		buffer        []uint8
		offset, count int32
		want          string
	}{
		{"empty buffer", nil, 0, 0, invalidAudioBuffer},
		{"odd buffer length", make([]uint8, 7), 0, 6, invalidAudioBuffer},
		{"odd offset", aligned, 1, 4, invalidAudioBufferOffset},
		{"offset at the end", aligned, 8, 2, invalidAudioBufferOffset},
		{"odd count", aligned, 0, 5, invalidOffsetCountLength},
		{"count past the end", aligned, 4, 6, invalidOffsetCountLength},
		{"zero count", aligned, 0, 0, invalidOffsetCountLength},
	} {
		err := instance.SubmitBufferBySliceOfByteAndInt32AndInt32(row.buffer, row.offset, row.count)
		if err == nil || !strings.Contains(err.Error(), row.want) {
			t.Fatalf("%s = %v, want %q", row.name, err, row.want)
		}
	}
	// A STEREO instance aligns to FOUR bytes, not two, so a buffer that a mono
	// instance accepts is refused here. That is what proves the channel count
	// reaches the check rather than a constant.
	stereo := &DynamicSoundEffectInstance{
		instance:   &SoundEffectInstance{currentVolume: 1},
		sampleRate: 44100,
		channels:   AudioChannelsStereo,
	}
	stereo.instance.bindDerived(stereo)
	twoBytes := make([]uint8, 2)
	if err := instance.SubmitBufferBySliceOfByte(twoBytes); err != nil &&
		strings.Contains(err.Error(), invalidAudioBuffer) {
		t.Fatal("a two-byte buffer was refused by a MONO instance; its BlockAlign is 2")
	}
	if err := stereo.SubmitBufferBySliceOfByte(twoBytes); err == nil ||
		!strings.Contains(err.Error(), invalidAudioBuffer) {
		t.Fatalf("a two-byte buffer on a STEREO instance = %v, want the alignment refusal", err)
	}

	// The disposal check runs before all four.
	_ = instance.instance.DisposeByBoolean(true)
	if err := instance.SubmitBufferBySliceOfByte(nil); !errors.Is(err, errAudioObjectDisposed) {
		t.Fatalf("SubmitBuffer on a disposed instance = %v", err)
	}
}

// TestDynamicInstanceConversionGuards pins the two members' own checks, which
// run before anything reaches CNA.
func TestDynamicInstanceConversionGuards(t *testing.T) {
	instance := newManagedDynamicInstance()
	if _, err := instance.GetSampleDuration(-1); !errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatalf("GetSampleDuration(-1) = %v", err)
	}
	if _, err := instance.GetSampleSizeInBytes(framework.TimeSpanFromTicks(-1)); !errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatalf("a negative duration = %v", err)
	}
	// A zero value answers rather than panicking.
	var zero *DynamicSoundEffectInstance
	if !zero.IsDisposed() || zero.Volume() != 0 {
		t.Fatal("a nil DynamicSoundEffectInstance did not answer its zero values")
	}
	if _, err := zero.PendingBufferCount(); !errors.Is(err, errDynamicSoundEffectInstanceNil) {
		t.Fatalf("PendingBufferCount on a nil instance = %v", err)
	}
}
