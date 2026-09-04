package framework

// TimeSpan is the exact-tick Go projection used for System.TimeSpan in XNA
// signatures. One tick is 100 nanoseconds.
type TimeSpan struct {
	ticks int64
}

func TimeSpanFromTicks(ticks int64) TimeSpan {
	return TimeSpan{ticks: ticks}
}

func (t TimeSpan) Ticks() int64 {
	return t.ticks
}

func (t TimeSpan) TotalSeconds() float64 {
	return float64(t.ticks) / 10_000_000
}

// The two BCL members the audio family's measured arithmetic reaches. Both are
// read from the pinned mscorlib IL rather than assumed, because both have a
// clamp or a rounding step a plausible implementation would leave out.

// timeSpanMaxTotal is the saturation bound both members share:
//
//	0x2710 ticks per millisecond, and ±922337203685477 milliseconds
//
// which is Int64.MaxValue ticks expressed in milliseconds, truncated.
const timeSpanMaxTotal = 922337203685477.0

// TotalMilliseconds is System.TimeSpan::get_TotalMilliseconds, 64 bytes:
//
//	double temp = (double)_ticks * 0.0001;
//	if (!(temp <= 922337203685477))  return  922337203685477;
//	if (!(temp >= -922337203685477)) return -922337203685477;
//	return temp;
//
// It SATURATES rather than overflowing, and the two comparisons are unordered
// (`ble.un` / `bge.un`), so a NaN -- which no TimeSpan can hold, since the field
// is an int64 -- would take the first branch. The multiply is by 0.0001 rather
// than a divide by 10000, which is a different float64 operation and can give a
// different last bit.
func (t TimeSpan) TotalMilliseconds() float64 {
	temp := float64(t.ticks) * 0.0001
	if !(temp <= timeSpanMaxTotal) {
		return timeSpanMaxTotal
	}
	if !(temp >= -timeSpanMaxTotal) {
		return -timeSpanMaxTotal
	}
	return temp
}
