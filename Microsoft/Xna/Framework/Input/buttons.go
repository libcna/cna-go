package input

// Buttons identifies XNA game pad buttons and thumbstick directions.
// xna:flags
type Buttons int32

const (
	ButtonsDPadUp               Buttons = 1
	ButtonsDPadDown             Buttons = 2
	ButtonsDPadLeft             Buttons = 4
	ButtonsDPadRight            Buttons = 8
	ButtonsStart                Buttons = 16
	ButtonsBack                 Buttons = 32
	ButtonsLeftStick            Buttons = 64
	ButtonsRightStick           Buttons = 128
	ButtonsLeftShoulder         Buttons = 256
	ButtonsRightShoulder        Buttons = 512
	ButtonsBigButton            Buttons = 2048
	ButtonsA                    Buttons = 4096
	ButtonsB                    Buttons = 8192
	ButtonsX                    Buttons = 16384
	ButtonsY                    Buttons = 32768
	ButtonsRightThumbstickUp    Buttons = 16777216
	ButtonsRightThumbstickDown  Buttons = 33554432
	ButtonsRightThumbstickRight Buttons = 67108864
	ButtonsRightThumbstickLeft  Buttons = 134217728
	ButtonsLeftThumbstickUp     Buttons = 268435456
	ButtonsLeftThumbstickDown   Buttons = 536870912
	ButtonsLeftThumbstickRight  Buttons = 1073741824
	ButtonsLeftThumbstickLeft   Buttons = 2097152
	ButtonsLeftTrigger          Buttons = 8388608
	ButtonsRightTrigger         Buttons = 4194304
)
