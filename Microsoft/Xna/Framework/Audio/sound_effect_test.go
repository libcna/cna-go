package audio

import (
	"errors"
	"math"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The audio family's managed half is measured here without a device: the four
// static scalars, the two sample conversions and every guard that runs before a
// native call. What needs a device is in the native-stress scenario.

// TestSoundEffectStaticDefaultsAreTheClassInitializers pins the five values the
// reference's .cctor writes, including the one that is not public surface.
func TestSoundEffectStaticDefaultsAreTheClassInitializers(t *testing.T) {
	resetSoundEffectStatics()
	if got := SoundEffectMasterVolume(); got != 1 {
		t.Fatalf("MasterVolume = %v, want 1", got)
	}
	if got := SoundEffectSpeedOfSound(); got != 343.5 {
		t.Fatalf("SpeedOfSound = %v, want 343.5", got)
	}
	if got := SoundEffectDopplerScale(); got != 1 {
		t.Fatalf("DopplerScale = %v, want 1", got)
	}
	if got := SoundEffectDistanceScale(); got != 1 {
		t.Fatalf("DistanceScale = %v, want 1", got)
	}
	// maxVelocityComponent is `assembly` and not public surface, but it is a
	// store the reference makes and set_SpeedOfSound recomputes it.
	if soundEffectMaxVelocityComponent != 343.499 {
		t.Fatalf("maxVelocityComponent = %v, want 343.499", soundEffectMaxVelocityComponent)
	}
}

// TestTheFourStaticSettersDoNotShareAValidationShape is the milestone's central
// managed claim. No two of them validate the same way and the differences are
// not guessable from the signatures, so every boundary is pinned.
func TestTheFourStaticSettersDoNotShareAValidationShape(t *testing.T) {
	nan := float32(math.NaN())

	// MasterVolume: the range is [0,1] and NaN is refused.
	for _, bad := range []float32{-0.001, 1.001, nan, float32(math.Inf(1)), float32(math.Inf(-1))} {
		resetSoundEffectStatics()
		if err := SetSoundEffectMasterVolume(bad); !errors.Is(err, errAudioArgumentOutOfRange) {
			t.Fatalf("SetMasterVolume(%v) = %v, want a range refusal", bad, err)
		}
		if SoundEffectMasterVolume() != 1 {
			t.Fatal("a refused MasterVolume changed the field")
		}
	}

	// SpeedOfSound: strictly positive, so ZERO is refused -- and NaN too.
	for _, bad := range []float32{0, -1, nan} {
		resetSoundEffectStatics()
		if err := SetSoundEffectSpeedOfSound(bad); !errors.Is(err, errAudioArgumentOutOfRange) {
			t.Fatalf("SetSpeedOfSound(%v) = %v, want a range refusal", bad, err)
		}
	}

	// DopplerScale: ZERO IS ALLOWED, which is the pair's one asymmetry. The
	// call still needs a running game, so the refusal it reaches is the
	// no-game one and NOT a range refusal -- which is what distinguishes the
	// two outcomes.
	resetSoundEffectStatics()
	if err := SetSoundEffectDopplerScale(0); errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatal("SetDopplerScale(0) was refused for its range; zero is legal there")
	}
	for _, bad := range []float32{-0.001, nan} {
		resetSoundEffectStatics()
		if err := SetSoundEffectDopplerScale(bad); !errors.Is(err, errAudioArgumentOutOfRange) {
			t.Fatalf("SetDopplerScale(%v) = %v, want a range refusal", bad, err)
		}
	}

	// DistanceScale: NaN is ACCEPTED here and only here, because the guard's
	// branch is `bge.un` and jumps PAST the throw when the comparison is
	// unordered.
	resetSoundEffectStatics()
	if err := SetSoundEffectDistanceScale(nan); errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatal("SetDistanceScale(NaN) was refused; `bge.un` accepts an unordered comparison")
	}
	if !math.IsNaN(float64(SoundEffectDistanceScale())) {
		t.Fatalf("DistanceScale = %v after a NaN; the clamp is an ORDERED `ble`, which NaN fails, so the NaN is stored",
			SoundEffectDistanceScale())
	}
	resetSoundEffectStatics()
	if err := SetSoundEffectDistanceScale(-0.001); !errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatalf("SetDistanceScale(-0.001) = %v, want a range refusal", err)
	}
}

// TestDistanceScaleClampsToSingleEpsilon pins the silent rewrite: a consumer who
// sets zero and reads the getter back does NOT get zero.
func TestDistanceScaleClampsToSingleEpsilon(t *testing.T) {
	for _, value := range []float32{0, float32Epsilon / 2, float32Epsilon} {
		resetSoundEffectStatics()
		_ = SetSoundEffectDistanceScale(value)
		if got := SoundEffectDistanceScale(); got != float32Epsilon {
			t.Fatalf("SetDistanceScale(%v) stored %v, want Single.Epsilon %v", value, got, float32Epsilon)
		}
	}
	// A value ABOVE the epsilon is stored unchanged.
	resetSoundEffectStatics()
	_ = SetSoundEffectDistanceScale(2)
	if got := SoundEffectDistanceScale(); got != 2 {
		t.Fatalf("SetDistanceScale(2) stored %v", got)
	}
}

// TestSpeedOfSoundRecomputesMaxVelocity pins the store the reference makes that
// no public member reports, and the ORDER that makes it observable: the field
// is written BEFORE the native call, so a call with no running game still
// leaves both fields updated.
func TestSpeedOfSoundRecomputesMaxVelocity(t *testing.T) {
	resetSoundEffectStatics()
	_ = SetSoundEffectSpeedOfSound(1000)
	if got := SoundEffectSpeedOfSound(); got != 1000 {
		t.Fatalf("SpeedOfSound = %v, want 1000 -- the store runs BEFORE the native call", got)
	}
	if want := float32(1000 - 1000.0/1000); soundEffectMaxVelocityComponent != want {
		t.Fatalf("maxVelocityComponent = %v, want %v", soundEffectMaxVelocityComponent, want)
	}
}

// TestMasterVolumeStoresAfterTheNativeCall is the mirror claim, and it is the
// pair's second asymmetry: MasterVolume calls native FIRST, so a call that
// cannot reach a runtime leaves the field ALONE.
func TestMasterVolumeStoresAfterTheNativeCall(t *testing.T) {
	resetSoundEffectStatics()
	err := SetSoundEffectMasterVolume(0.25)
	if err == nil {
		t.Skip("a runtime is available; this claim is about the no-runtime path")
	}
	if got := SoundEffectMasterVolume(); got != 1 {
		t.Fatalf("MasterVolume = %v after a failed set, want the previous 1", got)
	}
}

// TestSampleConversionsValidateInTheReferenceOrder pins both statics' guards,
// including the two that carry different exception shapes.
func TestSampleConversionsValidateInTheReferenceOrder(t *testing.T) {
	oneSecond := framework.TimeSpanFromTicks(10_000_000)

	// GetSampleSizeInBytes: duration, then sampleRate, then channels.
	if _, err := SoundEffectGetSampleSizeInBytes(framework.TimeSpanFromTicks(-1), 44100, AudioChannelsMono); !errors.Is(err, errAudioArgumentOutOfRange) {
		t.Fatalf("a negative duration = %v", err)
	}
	if _, err := SoundEffectGetSampleSizeInBytes(oneSecond, 7999, AudioChannelsMono); err == nil ||
		!strings.Contains(err.Error(), "sampleRate") {
		t.Fatalf("sampleRate 7999 = %v", err)
	}
	if _, err := SoundEffectGetSampleSizeInBytes(oneSecond, 48001, AudioChannelsMono); err == nil {
		t.Fatal("sampleRate 48001 was accepted")
	}
	// Both channel bounds, and ZERO in particular: the reference's test is
	// `channels < 1`, so a zero channel count is refused. A projection that
	// wrote `< 0` would accept it and then divide by a BlockAlign of zero.
	for _, bad := range []AudioChannels{0, 3, -1} {
		if _, err := SoundEffectGetSampleSizeInBytes(oneSecond, 44100, bad); err == nil ||
			!strings.Contains(err.Error(), "channels") {
			t.Fatalf("channels %d = %v, want a channels refusal", bad, err)
		}
	}
	// The bounds themselves are INCLUSIVE.
	for _, rate := range []int32{8000, 48000} {
		if _, err := SoundEffectGetSampleSizeInBytes(oneSecond, rate, AudioChannelsMono); err != nil {
			t.Fatalf("sampleRate %d was refused: %v", rate, err)
		}
	}
	// A zero duration returns zero -- AFTER all three range checks, so a zero
	// duration with a bad rate is still refused for the rate.
	if got, err := SoundEffectGetSampleSizeInBytes(framework.TimeSpan{}, 44100, AudioChannelsMono); err != nil || got != 0 {
		t.Fatalf("a zero duration = %d, %v", got, err)
	}
	if _, err := SoundEffectGetSampleSizeInBytes(framework.TimeSpan{}, 7999, AudioChannelsMono); err == nil {
		t.Fatal("a zero duration with a bad sample rate was accepted; the range checks run FIRST")
	}

	// GetSampleDuration: sizeInBytes carries a MESSAGE where the others carry
	// only a parameter name.
	if _, err := SoundEffectGetSampleDuration(-1, 44100, AudioChannelsMono); err == nil ||
		!strings.Contains(err.Error(), invalidBufferSize) {
		t.Fatalf("a negative size = %v, want the InvalidBufferSize message", err)
	}
	if _, err := SoundEffectGetSampleDuration(0, 7999, AudioChannelsMono); err == nil {
		t.Fatal("a bad sample rate was accepted")
	}
}

// TestSampleConversionsRoundTripAtTheMeasuredArithmetic pins the AudioFormat
// bodies, including the two details a plausible implementation drops: the
// float32 arithmetic and the millisecond rounding.
func TestSampleConversionsRoundTripAtTheMeasuredArithmetic(t *testing.T) {
	// The MEASURED byte counts, which are not the arithmetic ones -- and the
	// difference is the whole reason the float32 conversions are written out.
	//
	//	samples = (int)(TotalMilliseconds * (double)((float)SampleRate / 1000f))
	//
	// `(float)44100 / 1000f` is 44.099998474121094, not 44.1, because 44.1 has
	// no binary32 representation. One second therefore scales to 44099.998...
	// and TRUNCATES to 44099 samples rather than 44100.
	//
	// A rate whose thousandth IS representable -- 8000 and 48000 both are --
	// gives the exact count. 44100 and 22050 do not.
	oneSecond := framework.TimeSpanFromTicks(10_000_000)
	for _, row := range []struct {
		rate     int32
		channels AudioChannels
		want     int32
		why      string
	}{
		{8000, AudioChannelsMono, 16000, "8000/1000 is 8.0, exact in binary32"},
		{48000, AudioChannelsMono, 96000, "48000/1000 is 48.0, exact"},
		{44100, AudioChannelsMono, 88198, "44100/1000 truncates to 44099 samples, so 2 bytes short of 88200"},
		{22050, AudioChannelsMono, 44098, "22050/1000 truncates to 22049 samples"},
		// Stereo at 44100 comes back to the round number, and NOT because the
		// truncation went away: 44099 is odd, so `samples % Channels` is 1 and
		// the alignment step adds it. That is what makes the step an addition
		// rather than a round-up -- for mono the remainder is always zero.
		{44100, AudioChannelsStereo, 176400, "44099 is odd, so the channel remainder adds one sample back"},
	} {
		got, err := SoundEffectGetSampleSizeInBytes(oneSecond, row.rate, row.channels)
		if err != nil {
			t.Fatal(err)
		}
		if got != row.want {
			t.Fatalf("one second at %dHz/%d channels = %d bytes, want %d (%s)",
				row.rate, row.channels, got, row.want, row.why)
		}
		// And back: every one of them round-trips to exactly one second,
		// because TimeSpan.FromMilliseconds ROUNDS to a whole millisecond.
		duration, err := SoundEffectGetSampleDuration(got, row.rate, row.channels)
		if err != nil {
			t.Fatal(err)
		}
		if duration.Ticks() != 10_000_000 {
			t.Fatalf("%d bytes at %dHz/%d channels = %d ticks, want 10000000",
				got, row.rate, row.channels, duration.Ticks())
		}
		if duration.Ticks()%10000 != 0 {
			t.Fatalf("%d ticks is not a whole number of milliseconds", duration.Ticks())
		}
	}
	// The float32 scale factor decides the DURATION too, and here it differs by
	// a whole MILLISECOND rather than a rounding bit.
	//
	//	269462 bytes at 11025Hz mono
	//	    float32:  (float)134731 * 1000f / (float)11025  -> 12221.0 ms
	//	    float64:  134731.0 * 1000 / 11025.0             -> 12220.99...  -> 12220 ms
	//
	// A projection that did the arithmetic in float64 would be ten thousand
	// ticks short. Every other size in this test agrees between the two, which
	// is exactly why one that does not is needed.
	for _, row := range []struct {
		size, rate int32
		want       int64
	}{
		{269462, 11025, 122210000},
		{272990, 11025, 123810000},
		{276518, 11025, 125410000},
	} {
		got, err := SoundEffectGetSampleDuration(row.size, row.rate, AudioChannelsMono)
		if err != nil {
			t.Fatal(err)
		}
		if got.Ticks() != row.want {
			t.Fatalf("%d bytes at %dHz = %d ticks, want %d -- the scale factor is FLOAT32",
				row.size, row.rate, got.Ticks(), row.want)
		}
	}

	// A size that is not a whole frame is truncated by the int32 division,
	// which is the reference's own `sizeInBytes / BlockAlign`.
	duration, err := SoundEffectGetSampleDuration(88198, 44100, AudioChannelsMono)
	if err != nil {
		t.Fatal(err)
	}
	odd, err := SoundEffectGetSampleDuration(88199, 44100, AudioChannelsMono)
	if err != nil {
		t.Fatal(err)
	}
	if odd.Ticks() != duration.Ticks() {
		t.Fatalf("an extra byte changed the duration: %d against %d", odd.Ticks(), duration.Ticks())
	}
}

// TestSizeFromDurationAddsTheChannelRemainder pins the alignment step that is
// NOT a round-up: `samples + samples % Channels`.
func TestSizeFromDurationAddsTheChannelRemainder(t *testing.T) {
	// A duration that lands on an ODD sample count at stereo. 8000Hz for 1ms
	// is 8 samples, which is even; 8001 is not a legal rate, so the odd case is
	// reached with a duration that scales to an odd number.
	for _, row := range []struct {
		milliseconds float64
		rate         int32
		channels     AudioChannels
	}{
		{1, 8000, AudioChannelsStereo},
		{3, 44100, AudioChannelsStereo},
		{7, 22050, AudioChannelsStereo},
		{3, 44100, AudioChannelsMono},
	} {
		duration, err := timeSpanFromMilliseconds(row.milliseconds)
		if err != nil {
			t.Fatal(err)
		}
		size, err := sizeFromDuration(duration, row.rate, int32(row.channels))
		if err != nil {
			t.Fatal(err)
		}
		samples := int32(duration.TotalMilliseconds() * float64(float32(row.rate)/1000))
		want := (samples + samples%int32(row.channels)) * audioBlockAlign(int32(row.channels))
		if size != want {
			t.Fatalf("%vms at %dHz/%d channels = %d, want %d", row.milliseconds, row.rate, row.channels, size, want)
		}
		// For MONO the remainder is always zero, so nothing is added.
		if row.channels == AudioChannelsMono && size != samples*2 {
			t.Fatalf("mono added a remainder: %d against %d", size, samples*2)
		}
	}
}

// resetSoundEffectStatics puts the four scalars and the internal one back to
// the class initializer's values. The reference has no way to do this and needs
// none; a test that mutates process-wide state does.
func resetSoundEffectStatics() {
	soundEffectMasterVolume = 1
	soundEffectSpeedOfSound = 343.5
	soundEffectDopplerScale = 1
	soundEffectDistanceScale = 1
	soundEffectMaxVelocityComponent = 343.499
}
