package framework

// GameTime is a tick-exact value snapshot supplied by the native game loop.
// XNA defines a sealed class; CNA-Go deliberately uses value semantics for
// callback snapshots so a retained value cannot alias mutable native state.
type GameTime struct {
	totalGameTime   TimeSpan
	elapsedGameTime TimeSpan
	runningSlowly   bool
}

func NewGameTimeByNone() GameTime {
	return GameTime{}
}

func NewGameTimeByTimeSpanAndTimeSpan(totalGameTime, elapsedGameTime TimeSpan) GameTime {
	return GameTime{totalGameTime: totalGameTime, elapsedGameTime: elapsedGameTime}
}

func NewGameTimeByTimeSpanAndTimeSpanAndBoolean(totalGameTime, elapsedGameTime TimeSpan, isRunningSlowly bool) GameTime {
	return GameTime{totalGameTime: totalGameTime, elapsedGameTime: elapsedGameTime, runningSlowly: isRunningSlowly}
}

func (g GameTime) TotalGameTime() TimeSpan {
	return g.totalGameTime
}

func (g GameTime) ElapsedGameTime() TimeSpan {
	return g.elapsedGameTime
}

func (g GameTime) IsRunningSlowly() bool {
	return g.runningSlowly
}
