package touch

// GestureType identifies XNA touch gesture kinds.
// xna:flags
type GestureType int32

const (
	GestureTypeNone           GestureType = 0
	GestureTypeTap            GestureType = 1
	GestureTypeDoubleTap      GestureType = 2
	GestureTypeHold           GestureType = 4
	GestureTypeHorizontalDrag GestureType = 8
	GestureTypeVerticalDrag   GestureType = 16
	GestureTypeFreeDrag       GestureType = 32
	GestureTypePinch          GestureType = 64
	GestureTypeFlick          GestureType = 128
	GestureTypeDragComplete   GestureType = 256
	GestureTypePinchComplete  GestureType = 512
)
