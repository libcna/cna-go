package audio

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// This file projects SoundEffect's STATIC surface: the four process-wide
// scalars and the two sample conversions.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # The four scalars are static FIELDS with native write-through
//
// The class initializer seeds them:
//
//	speedOfSound         = 343.5
//	dopplerScale         = 1
//	distanceScale        = 1
//	maxVelocityComponent = 343.499        (assembly-only, not public surface)
//	currentVolume        = 1              (MasterVolume's backing field)
//
// Every GETTER is one `ldsfld`, so the projection reads its own field and binds
// no route; every SETTER validates, calls native and stores. CNA's four read
// routes are recorded as REDUNDANT_READ for exactly that reason.
//
// # No two setters validate the same way, and the differences are measured
//
//	set_MasterVolume   blt.un 0 / bgt.un 1 -> throw   range [0,1], NaN THROWS
//	                   native FIRST, then store
//	set_SpeedOfSound   ble.un 0 -> throw              must be > 0, NaN THROWS
//	                   store, recompute maxVelocityComponent, then native
//	set_DopplerScale   blt.un 0 -> throw              ZERO IS ALLOWED, NaN THROWS
//	set_DistanceScale  bge.un 0 -> CONTINUE           NaN is ACCEPTED here alone,
//	                   then value = (value <= float.Epsilon) ? float.Epsilon : value
//
// Three of the four refuse NaN and the fourth stores it. Zero is legal for
// DopplerScale, illegal for SpeedOfSound, and silently clamped to float.Epsilon
// for DistanceScale. None of that is guessable from the signatures.

// The five static fields, seeded exactly as the class initializer does.
var (
	soundEffectMasterVolume  float32 = 1
	soundEffectSpeedOfSound  float32 = 343.5
	soundEffectDopplerScale  float32 = 1
	soundEffectDistanceScale float32 = 1
	// maxVelocityComponent is `assembly` in the reference and is not public
	// surface. It is kept because set_SpeedOfSound RECOMPUTES it, and dropping
	// it would drop a store the reference makes.
	soundEffectMaxVelocityComponent float32 = 343.499
)

// errSoundEffectNoRunningGame is the Go-only refusal every static setter can
// answer.
//
// The reference's setters are static and reach a process-wide native mixer with
// no game in sight. CNA's routes take a GAME handle, so a projection outside a
// running game has nothing to call and says so rather than storing the value
// and reporting success -- which would let a consumer believe the mixer had
// been told when it had not. That is the same position FrameworkDispatcher's
// static member is in.
var errSoundEffectNoRunningGame = errors.New("this member needs a running game")

// float32Epsilon is System.Single.Epsilon, the smallest positive subnormal
// binary32 value: 1.401298E-45. set_DistanceScale clamps to it.
const float32Epsilon = float32(1.401298464324817e-45)

// SoundEffectMasterVolume is SoundEffect::get_MasterVolume, one `ldsfld`.
func SoundEffectMasterVolume() float32 { return soundEffectMasterVolume }

// SetSoundEffectMasterVolume is SoundEffect::set_MasterVolume, 45 bytes:
//
//	if (!(value >= 0) || !(value <= 1)) throw new ArgumentOutOfRangeException("value");
//	SetMasterVolume(value);            // native FIRST
//	currentVolume = value;             // store SECOND
//
// The order is load-bearing: a native call that fails leaves the managed field
// at its previous value, so a consumer who reads the getter back after a failed
// set sees what is actually in effect.
func SetSoundEffectMasterVolume(value float32) error {
	if !(value >= 0) || !(value <= 1) {
		return argumentOutOfRangeError("value", "")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	if err := runtime.SetMasterVolume(value); err != nil {
		return err
	}
	soundEffectMasterVolume = value
	return nil
}

// SoundEffectSpeedOfSound is SoundEffect::get_SpeedOfSound.
func SoundEffectSpeedOfSound() float32 { return soundEffectSpeedOfSound }

// SetSoundEffectSpeedOfSound is SoundEffect::set_SpeedOfSound, 58 bytes:
//
//	if (!(value > 0)) throw new ArgumentOutOfRangeException("value");
//	speedOfSound = value;
//	maxVelocityComponent = speedOfSound - speedOfSound / 1000;
//	SetSpeedOfSound(speedOfSound);
//
// Note the ORDER, which is the opposite of MasterVolume's: the store happens
// FIRST and the native call last, so a failed native call leaves the managed
// field ALREADY UPDATED. The two setters disagree and both are reproduced.
//
// `ble.un 0` sends both zero and NaN to the throw, so the value must be
// strictly positive.
func SetSoundEffectSpeedOfSound(value float32) error {
	if !(value > 0) {
		return argumentOutOfRangeError("value", "")
	}
	soundEffectSpeedOfSound = value
	soundEffectMaxVelocityComponent = soundEffectSpeedOfSound - soundEffectSpeedOfSound/1000
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	return runtime.SetSpeedOfSound(soundEffectSpeedOfSound)
}

// SoundEffectDopplerScale is SoundEffect::get_DopplerScale.
func SoundEffectDopplerScale() float32 { return soundEffectDopplerScale }

// SetSoundEffectDopplerScale is SoundEffect::set_DopplerScale, 36 bytes:
//
//	if (!(value >= 0)) throw new ArgumentOutOfRangeException("value");
//	dopplerScale = value;
//	SetDopplerScale(dopplerScale);
//
// `blt.un 0` refuses a negative value and a NaN. ZERO IS ACCEPTED here and
// refused by SpeedOfSound, which is the pair's one asymmetry.
func SetSoundEffectDopplerScale(value float32) error {
	if !(value >= 0) {
		return argumentOutOfRangeError("value", "")
	}
	soundEffectDopplerScale = value
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	return runtime.SetDopplerScale(soundEffectDopplerScale)
}

// SoundEffectDistanceScale is SoundEffect::get_DistanceScale.
func SoundEffectDistanceScale() float32 { return soundEffectDistanceScale }

// SetSoundEffectDistanceScale is SoundEffect::set_DistanceScale, 54 bytes, and
// it is the odd one out twice over:
//
//	if (!(value >= 0))                                  // bge.un -> NaN CONTINUES
//	    throw new ArgumentOutOfRangeException("value");
//	value = (value <= Single.Epsilon) ? Single.Epsilon : value;   // ble, ORDERED
//	distanceScale = value;
//	SetDistanceScale(distanceScale);
//
// The guard's branch is `bge.un`, which jumps PAST the throw when the
// comparison is unordered -- so a NaN is accepted here and refused by the other
// three. The clamp that follows is an ORDERED `ble`, which a NaN fails, so a
// NaN survives the clamp too and is stored.
//
// Zero and every positive value below Single.Epsilon become Single.Epsilon,
// silently. A consumer who sets zero and reads the getter back does not get
// zero.
func SetSoundEffectDistanceScale(value float32) error {
	if !(value >= 0) && !isNaN32(value) {
		return argumentOutOfRangeError("value", "")
	}
	if value <= float32Epsilon {
		value = float32Epsilon
	}
	soundEffectDistanceScale = value
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	return runtime.SetDistanceScale(soundEffectDistanceScale)
}

// isNaN32 is the unordered half of the CIL comparisons above, spelled once.
func isNaN32(value float32) bool { return value != value }

// SoundEffectGetSampleSizeInBytes is
// SoundEffect::GetSampleSizeInBytes(TimeSpan, Int32, AudioChannels), 136 bytes.
//
// It is PURE MANAGED: the whole body is validation plus AudioFormat arithmetic,
// and it reaches no runtime at all. CNA publishes
// cna_sound_effect_get_sample_size_in_bytes, but that route takes a SOUND
// EFFECT handle and this member is static with no effect in sight, so binding
// it would answer a different question.
//
//	if (!(duration.TotalMilliseconds >= 0) || !(duration.TotalMilliseconds <= Int32.MaxValue))
//	    throw new ArgumentOutOfRangeException("duration");
//	if (sampleRate < 8000 || sampleRate > 48000) throw ... ("sampleRate");
//	if (channels < 1 || channels > 2)            throw ... ("channels");
//	if (duration == TimeSpan.Zero) return 0;     // an EARLY return, before any maths
//	try { return AudioHelper.GetSampleSizeInBytes(...); }
//	catch (OverflowException) { throw new ArgumentOutOfRangeException("duration"); }
//
// The zero check runs AFTER all three range checks, so a zero duration with a
// bad sample rate is still refused for the sample rate.
func SoundEffectGetSampleSizeInBytes(duration framework.TimeSpan, sampleRate int32, channels AudioChannels) (int32, error) {
	total := duration.TotalMilliseconds()
	if !(total >= 0) || !(total <= 2147483647) {
		return 0, argumentOutOfRangeError("duration", "")
	}
	if err := checkAudioSampleRate(sampleRate); err != nil {
		return 0, err
	}
	if err := checkAudioChannels(channels); err != nil {
		return 0, err
	}
	if duration.Ticks() == 0 {
		return 0, nil
	}
	size, err := sizeFromDuration(duration, sampleRate, int32(channels))
	if err != nil {
		// The reference catches OverflowException and rethrows it as an
		// ArgumentOutOfRangeException naming the DURATION, not the arithmetic.
		return 0, argumentOutOfRangeError("duration", "")
	}
	return size, nil
}

// SoundEffectGetSampleDuration is
// SoundEffect::GetSampleDuration(Int32, Int32, AudioChannels), 84 bytes, and
// its first guard is the only one in the pair that carries a MESSAGE:
//
//	if (sizeInBytes < 0)
//	    throw new ArgumentOutOfRangeException("sizeInBytes", FrameworkResources.InvalidBufferSize);
//	if (sampleRate < 8000 || sampleRate > 48000) throw ... ("sampleRate");
//	if (channels < 1 || channels > 2)            throw ... ("channels");
//	return AudioHelper.GetSampleDuration(sizeInBytes, sampleRate, channels);
//
// There is no overflow catch here: the conversion divides rather than
// multiplies, so it cannot overflow an int32.
func SoundEffectGetSampleDuration(sizeInBytes, sampleRate int32, channels AudioChannels) (framework.TimeSpan, error) {
	if sizeInBytes < 0 {
		return framework.TimeSpan{}, argumentOutOfRangeError("sizeInBytes", invalidBufferSize)
	}
	if err := checkAudioSampleRate(sampleRate); err != nil {
		return framework.TimeSpan{}, err
	}
	if err := checkAudioChannels(channels); err != nil {
		return framework.TimeSpan{}, err
	}
	return durationFromSize(sizeInBytes, sampleRate, int32(channels))
}

// The two range checks four members share, with the reference's exact bounds:
// 0x1f40..0xbb80 for the sample rate and 1..2 for the channel count.
func checkAudioSampleRate(sampleRate int32) error {
	if sampleRate < 8000 || sampleRate > 48000 {
		return argumentOutOfRangeError("sampleRate", "")
	}
	return nil
}

func checkAudioChannels(channels AudioChannels) error {
	if channels < 1 || channels > 2 {
		return argumentOutOfRangeError("channels", "")
	}
	return nil
}
