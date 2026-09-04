package audio

import (
	"errors"
	"math"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// This file projects the arithmetic Microsoft.Xna.Framework.Audio.AudioFormat
// performs, WITHOUT projecting the type: `AudioFormat` is `private` in the
// reference and not in the pinned contract, so nothing here is public surface.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//	mscorlib (pinned .NET Framework 4.0)  for TimeSpan::Interval
//
// # What an AudioFormat actually is, on the paths SoundEffect reaches
//
// `AudioHelper.MakeFormat(sampleRate, channels, bitDepth)` writes a
// WAVEFORMATEX and `AudioFormat.Create` reads it straight back:
//
//	FormatTag      = 1                                    (PCM)
//	Channels       = (int16)channels
//	SampleRate     = sampleRate
//	AvgBytesPerSec = sampleRate * (channels * bitDepth / 8)
//	BlockAlign     = (int16)(channels * bitDepth / 8)
//	BitsPerSample  = bitDepth
//	cbSize         = 0
//
// Every path SoundEffect takes passes `bitDepth = 16`, so
//
//	BlockAlign = channels * 2
//
// and the whole round trip through the byte buffer collapses to three numbers.
// Reproducing the buffer would be reproducing a serialisation nothing reads.

// audioBlockAlign is the reference's BlockAlign for the only bit depth
// SoundEffect uses. It is a function rather than a constant so the `* 2` has
// one place to be wrong in.
func audioBlockAlign(channels int32) int32 { return channels * 2 }

// durationFromSize is AudioFormat::DurationFromSize(int32), 32 bytes:
//
//	int frames = sizeInBytes / BlockAlign;
//	return TimeSpan.FromMilliseconds((double)((float)frames * 1000f / (float)SampleRate));
//
// The multiply and the divide are FLOAT32 and only the result is widened to
// float64 for FromMilliseconds. Doing the arithmetic in float64 gives a
// different answer for most inputs, which is why the conversions are written
// out rather than folded.
func durationFromSize(sizeInBytes, sampleRate, channels int32) (framework.TimeSpan, error) {
	frames := sizeInBytes / audioBlockAlign(channels)
	milliseconds := float64(float32(frames) * 1000 / float32(sampleRate))
	return timeSpanFromMilliseconds(milliseconds)
}

// sizeFromDuration is AudioFormat::SizeFromDuration(TimeSpan), 44 bytes, and
// every arithmetic step in it is CHECKED:
//
//	int samples = checked((int)(duration.TotalMilliseconds * (double)((float)SampleRate / 1000f)));
//	int aligned = checked(samples + samples % Channels);
//	return checked(aligned * BlockAlign);
//
// `samples % Channels` is ADDED, which is not a round-up: for two channels and
// an odd sample count it adds one, and for one channel it always adds zero. A
// projection that rounded up to a multiple of BlockAlign would agree on every
// even input and differ on the odd ones.
//
// The three `conv.ovf.i4` / `add.ovf` / `mul.ovf` are what produce the
// OverflowException that SoundEffect.GetSampleSizeInBytes catches and converts
// into ArgumentOutOfRangeException("duration"), so the overflow is reported
// here rather than silently wrapping.
func sizeFromDuration(duration framework.TimeSpan, sampleRate, channels int32) (int32, error) {
	scaled := duration.TotalMilliseconds() * float64(float32(sampleRate)/1000)
	if math.IsNaN(scaled) || scaled > math.MaxInt32 || scaled < math.MinInt32 {
		return 0, errAudioOverflow
	}
	samples := int32(scaled)
	aligned := int64(samples) + int64(samples%channels)
	if aligned > math.MaxInt32 || aligned < math.MinInt32 {
		return 0, errAudioOverflow
	}
	total := aligned * int64(audioBlockAlign(channels))
	if total > math.MaxInt32 || total < math.MinInt32 {
		return 0, errAudioOverflow
	}
	return int32(total), nil
}

// errAudioOverflow stands for the System.OverflowException the reference's
// checked arithmetic raises. It never escapes the audio package: every caller
// converts it into the ArgumentOutOfRangeException the reference's own catch
// block produces.
var errAudioOverflow = errors.New("audio arithmetic overflowed")

// timeSpanFromMilliseconds is System.TimeSpan::FromMilliseconds, which is
// `Interval(value, 1)` in the pinned mscorlib, and Interval ROUNDS to a whole
// millisecond before it scales:
//
//	if (Double.IsNaN(value)) throw new ArgumentException(Arg_CannotBeNaN);
//	double millis = value + (value >= 0 ? 0.5 : -0.5);
//	if (millis > 922337203685477 || millis < -922337203685477)
//	    throw new OverflowException(Overflow_TimeSpanTooLong);
//	return new TimeSpan((long)millis * 10000);
//
// The rounding is what decides the answer: a TimeSpan built from milliseconds
// can never carry sub-millisecond ticks, so every duration this file produces
// has a tick count that is a multiple of 10000. A projection that multiplied by
// 10000 without the ±0.5 would differ on almost every input.
//
// It lives here rather than on framework.TimeSpan because a package-level
// constructor there would be public surface the pinned contract does not
// declare. framework.TimeSpanFromTicks, which the contract does carry, is what
// this hands its result to.
func timeSpanFromMilliseconds(value float64) (framework.TimeSpan, error) {
	if math.IsNaN(value) {
		return framework.TimeSpan{}, errAudioOverflow
	}
	millis := value
	if value >= 0 {
		millis += 0.5
	} else {
		millis -= 0.5
	}
	// 922337203685477 is Int64.MaxValue ticks expressed in milliseconds,
	// truncated -- the same bound TimeSpan::get_TotalMilliseconds saturates at.
	const maxTotalMilliseconds = 922337203685477.0
	if millis > maxTotalMilliseconds || millis < -maxTotalMilliseconds {
		return framework.TimeSpan{}, errAudioOverflow
	}
	return framework.TimeSpanFromTicks(int64(millis) * 10000), nil
}
