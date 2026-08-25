package input

// GamePadType identifies the reported kind of an XNA game pad device.
type GamePadType int32

const (
	GamePadTypeUnknown         GamePadType = 0
	GamePadTypeGamePad         GamePadType = 1
	GamePadTypeWheel           GamePadType = 2
	GamePadTypeArcadeStick     GamePadType = 3
	GamePadTypeFlightStick     GamePadType = 4
	GamePadTypeDancePad        GamePadType = 5
	GamePadTypeGuitar          GamePadType = 6
	GamePadTypeAlternateGuitar GamePadType = 7
	GamePadTypeDrumKit         GamePadType = 8
	GamePadTypeBigButtonPad    GamePadType = 768
)
