package input

// GamePadDeadZone identifies how XNA applies a game pad thumbstick dead zone.
type GamePadDeadZone int32

const (
	GamePadDeadZoneNone            GamePadDeadZone = 0
	GamePadDeadZoneIndependentAxes GamePadDeadZone = 1
	GamePadDeadZoneCircular        GamePadDeadZone = 2
)
