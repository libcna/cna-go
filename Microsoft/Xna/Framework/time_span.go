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
