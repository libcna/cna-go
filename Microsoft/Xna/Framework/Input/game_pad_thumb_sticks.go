package input

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// GamePadThumbSticks is a managed value describing both XNA game pad
// thumbstick positions. Constructing it claims no game pad capability: it is
// an ordinary value holder with no polling, device, or backend route.
type GamePadThumbSticks struct {
	left  framework.Vector2
	right framework.Vector2
}

// NewGamePadThumbSticks clamps each thumbstick into the unit square exactly as
// the reference constructor does: a component-wise Vector2 minimum against
// Vector2.One followed by a component-wise maximum against -Vector2.One.
// Because the XNA Vector2 minimum keeps the second operand when the comparison
// is false, a NaN component clamps to 1 rather than staying NaN.
func NewGamePadThumbSticks(leftThumbstick, rightThumbstick framework.Vector2) GamePadThumbSticks {
	return GamePadThumbSticks{
		left:  clampThumbstick(leftThumbstick),
		right: clampThumbstick(rightThumbstick),
	}
}

func clampThumbstick(value framework.Vector2) framework.Vector2 {
	value = framework.Vector2MinByVector2AndVector2(value, framework.Vector2One())
	return framework.Vector2MaxByVector2AndVector2(
		value,
		framework.Vector2OperatorUnaryNegationByVector2(framework.Vector2One()),
	)
}

func (s GamePadThumbSticks) Left() framework.Vector2  { return s.left }
func (s GamePadThumbSticks) Right() framework.Vector2 { return s.right }

func (s GamePadThumbSticks) Equals(obj any) bool {
	other, ok := obj.(GamePadThumbSticks)
	return ok && GamePadThumbSticksOperatorEqualityByGamePadThumbSticksAndGamePadThumbSticks(s, other)
}

// GetHashCode reproduces Helpers.SmartGetHashCode over the four Single words
// of the sequential layout.
func (s GamePadThumbSticks) GetHashCode() int32 {
	return smartGetHashCode(
		singleWord(s.left.X), singleWord(s.left.Y),
		singleWord(s.right.X), singleWord(s.right.Y),
	)
}

func (s GamePadThumbSticks) ToString() string {
	return fmt.Sprintf("{Left:%s Right:%s}", s.left.ToString(), s.right.ToString())
}

func GamePadThumbSticksOperatorEqualityByGamePadThumbSticksAndGamePadThumbSticks(left, right GamePadThumbSticks) bool {
	return framework.Vector2OperatorEqualityByVector2AndVector2(left.left, right.left) &&
		framework.Vector2OperatorEqualityByVector2AndVector2(left.right, right.right)
}

func GamePadThumbSticksOperatorInequalityByGamePadThumbSticksAndGamePadThumbSticks(left, right GamePadThumbSticks) bool {
	return framework.Vector2OperatorInequalityByVector2AndVector2(left.left, right.left) ||
		framework.Vector2OperatorInequalityByVector2AndVector2(left.right, right.right)
}
