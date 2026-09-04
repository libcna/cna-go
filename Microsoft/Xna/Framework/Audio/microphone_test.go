package audio

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// Microphone's managed half is one validated setter, one exception-kind
// surprise and a state mapping. Its CAPTURE members are projected and are never
// called here for the reason they are never called in the native scenario:
// recording from a physical microphone is not something a test suite does on
// someone's machine.

// TestMicrophoneBufferDurationHasThreeGuardsAndOneMessage pins set_BufferDuration's
// single `if` with three parts, and the bounds that are INCLUSIVE.
func TestMicrophoneBufferDurationHasThreeGuardsAndOneMessage(t *testing.T) {
	microphone := &Microphone{}
	milliseconds := func(value int64) framework.TimeSpan {
		return framework.TimeSpanFromTicks(value * 10000)
	}
	for _, row := range []struct {
		name  string
		value framework.TimeSpan
	}{
		{"below the floor", milliseconds(90)},
		{"above the ceiling", milliseconds(1010)},
		{"not 10ms aligned", milliseconds(105)},
		{"zero", milliseconds(0)},
		{"negative", milliseconds(-100)},
	} {
		err := microphone.SetBufferDuration(row.value)
		if !errors.Is(err, errAudioArgumentOutOfRange) {
			t.Fatalf("%s = %v, want a range refusal", row.name, err)
		}
		if !strings.Contains(err.Error(), invalidMicrophoneBufferDuration) {
			t.Fatalf("%s carried %q, not the reference's message", row.name, err)
		}
	}
	// Both bounds are INCLUSIVE and both are 10ms aligned, so all three of
	// these get past the guard and stop at the no-runtime refusal.
	// The alignment is to TEN milliseconds, not a hundred: 110 and 250 are
	// legal and a projection that aligned to 100 would refuse both while
	// accepting every round value above.
	for _, good := range []int64{100, 1000, 500, 110, 250, 990} {
		err := microphone.SetBufferDuration(milliseconds(good))
		if errors.Is(err, errAudioArgumentOutOfRange) {
			t.Fatalf("%dms was refused for its range; the bounds are inclusive", good)
		}
	}
}

// TestMicrophoneGetSampleDurationThrowsADifferentKindThanSoundEffects is the
// milestone's sharpest managed claim.
//
// Two members read the same and carry the SAME message, and they throw
// different exception KINDS:
//
//	Microphone::GetSampleDuration(int)      ArgumentException(InvalidBufferSize)
//	SoundEffect::GetSampleDuration(int,...) ArgumentOutOfRangeException("sizeInBytes", InvalidBufferSize)
//
// A projection that shared one helper between them would be wrong for one of
// the two, and nothing in the signatures says which.
func TestMicrophoneGetSampleDurationThrowsADifferentKindThanSoundEffects(t *testing.T) {
	microphone := &Microphone{}
	_, err := microphone.GetSampleDuration(-1)
	if !errors.Is(err, errAudioArgument) {
		t.Fatalf("Microphone.GetSampleDuration(-1) = %v, want a plain ArgumentException", err)
	}
	if errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatal("Microphone.GetSampleDuration threw the OUT OF RANGE kind; the reference throws ArgumentException")
	}
	if !strings.Contains(err.Error(), invalidBufferSize) {
		t.Fatalf("the refusal carried %q", err)
	}
	// SoundEffect's sibling, same message, other kind.
	_, err = SoundEffectGetSampleDuration(-1, 44100, AudioChannelsMono)
	if !errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatalf("SoundEffect.GetSampleDuration(-1) = %v, want the OUT OF RANGE kind", err)
	}
	if !strings.Contains(err.Error(), invalidBufferSize) {
		t.Fatal("the two members do not share the message they are supposed to share")
	}
}

// TestMicrophoneStateMappingFallsBackToStopped pins the branch a qualified
// artifact may never take, in its own function so the path is reachable.
func TestMicrophoneStateMappingFallsBackToStopped(t *testing.T) {
	if got := microphoneStateFromNative(nativeMicrophoneStateStarted); got != MicrophoneStateStarted {
		t.Fatalf("CNA_MICROPHONE_STATE_STARTED mapped to %v", got)
	}
	if got := microphoneStateFromNative(nativeMicrophoneStateStopped); got != MicrophoneStateStopped {
		t.Fatalf("CNA_MICROPHONE_STATE_STOPPED mapped to %v", got)
	}
	for _, unknown := range []uint32{2, 7, 255, 1 << 31} {
		if got := microphoneStateFromNative(unknown); got != MicrophoneStateStopped {
			t.Fatalf("an unrecognised state %d mapped to %v, want Stopped", unknown, got)
		}
	}
}

// TestMicrophoneGetDataGuardsRunBeforeAnyCapture pins that the three window
// checks are managed, so a consumer's bad arguments are refused WITHOUT the
// projection ever reaching a capture route.
func TestMicrophoneGetDataGuardsRunBeforeAnyCapture(t *testing.T) {
	microphone := &Microphone{}
	buffer := make([]uint8, 8)
	for _, row := range []struct {
		name          string
		buffer        []uint8
		offset, count int32
		want          string
	}{
		{"empty buffer", nil, 0, 0, invalidAudioBuffer},
		{"negative offset", buffer, -1, 4, invalidAudioBufferOffset},
		{"offset at the end", buffer, 8, 1, invalidAudioBufferOffset},
		{"zero count", buffer, 0, 0, invalidOffsetCountLength},
		{"count past the end", buffer, 4, 6, invalidOffsetCountLength},
	} {
		_, err := microphone.GetDataBySliceOfByteAndInt32AndInt32(row.buffer, row.offset, row.count)
		if err == nil || !strings.Contains(err.Error(), row.want) {
			t.Fatalf("%s = %v, want %q", row.name, err, row.want)
		}
	}
}

// TestMicrophoneNameIsAField pins the projection shape the pinned contract
// forces: `Name` is `kind: field` and not a property, so it is an exported Go
// FIELD with no getter.
//
// The `initonly` half cannot be projected -- Go has no readonly field -- so a
// consumer can assign to it where the reference would not compile. That is a
// language limitation, and this test states it rather than hiding it: the
// assignment below is legal Go and is the divergence.
func TestMicrophoneNameIsAField(t *testing.T) {
	microphone := &Microphone{Name: "Front Microphone"}
	if microphone.Name != "Front Microphone" {
		t.Fatalf("Name = %q", microphone.Name)
	}
	// The divergence, written down: `initonly` has no Go counterpart.
	microphone.Name = "reassigned"
	if microphone.Name != "reassigned" {
		t.Fatal("the field is not assignable, which contradicts the recorded limitation")
	}
}

// TestNilMicrophoneMethodsRefuse pins the nil-receiver guard on every method,
// and states the one place the field projection cannot have one.
//
// Every METHOD on a nil *Microphone answers or refuses. `Name` is a FIELD --
// because the pinned contract declares it `kind: field` -- so reading it
// through a nil pointer panics, where a getter would have returned "". That is
// the second consequence of the contract's field kind, after the missing
// `initonly`, and it is recorded here rather than papered over with an
// accessor the contract does not declare.
func TestNilMicrophoneMethodsRefuse(t *testing.T) {
	var microphone *Microphone
	if got := microphone.BufferDuration(); got.Ticks() != 0 {
		t.Fatalf("a nil Microphone's BufferDuration = %d ticks", got.Ticks())
	}
	for name, call := range map[string]func() error{
		"Start":             microphone.Start,
		"Stop":              microphone.Stop,
		"Finalize":          microphone.Finalize,
		"SetBufferDuration": func() error { return microphone.SetBufferDuration(framework.TimeSpan{}) },
	} {
		if err := call(); !errors.Is(err, errMicrophoneNil) {
			t.Fatalf("%s on a nil Microphone = %v", name, err)
		}
	}
	if _, err := microphone.SampleRate(); !errors.Is(err, errMicrophoneNil) {
		t.Fatalf("SampleRate on a nil Microphone = %v", err)
	}
	if _, err := microphone.IsHeadset(); !errors.Is(err, errMicrophoneNil) {
		t.Fatalf("IsHeadset on a nil Microphone = %v", err)
	}
	if _, err := microphone.State(); !errors.Is(err, errMicrophoneNil) {
		t.Fatalf("State on a nil Microphone = %v", err)
	}
	if _, err := microphone.GetSampleDuration(0); !errors.Is(err, errMicrophoneNil) {
		t.Fatalf("GetSampleDuration on a nil Microphone = %v", err)
	}
	if _, err := microphone.GetSampleSizeInBytes(framework.TimeSpan{}); !errors.Is(err, errMicrophoneNil) {
		t.Fatalf("GetSampleSizeInBytes on a nil Microphone = %v", err)
	}
}
