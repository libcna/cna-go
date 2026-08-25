package input

import "fmt"

// GamePadTriggers is a managed value describing both XNA game pad trigger
// positions. Constructing it claims no game pad capability.
type GamePadTriggers struct {
	left  float32
	right float32
}

// NewGamePadTriggers clamps each trigger into [0, 1] exactly as the reference
// constructor does: System.Math.Min against 1 followed by System.Math.Max
// against 0. Math.Min and Math.Max propagate NaN, so a NaN trigger stays NaN,
// and negative zero becomes positive zero because Max returns its second
// operand when the comparison is false.
func NewGamePadTriggers(leftTrigger, rightTrigger float32) GamePadTriggers {
	return GamePadTriggers{
		left:  clampTrigger(leftTrigger),
		right: clampTrigger(rightTrigger),
	}
}

func clampTrigger(value float32) float32 {
	return mathMaxSingle(mathMinSingle(value, 1), 0)
}

func (t GamePadTriggers) Left() float32  { return t.left }
func (t GamePadTriggers) Right() float32 { return t.right }

func (t GamePadTriggers) Equals(obj any) bool {
	other, ok := obj.(GamePadTriggers)
	return ok && GamePadTriggersOperatorEqualityByGamePadTriggersAndGamePadTriggers(t, other)
}

// GetHashCode reproduces Helpers.SmartGetHashCode over the two Single words of
// the sequential layout.
func (t GamePadTriggers) GetHashCode() int32 {
	return smartGetHashCode(singleWord(t.left), singleWord(t.right))
}

func (t GamePadTriggers) ToString() string {
	return fmt.Sprintf("{Left:%s Right:%s}", formatSingle(t.left), formatSingle(t.right))
}

func GamePadTriggersOperatorEqualityByGamePadTriggersAndGamePadTriggers(left, right GamePadTriggers) bool {
	return left.left == right.left && left.right == right.right
}

func GamePadTriggersOperatorInequalityByGamePadTriggersAndGamePadTriggers(left, right GamePadTriggers) bool {
	return left.left != right.left || left.right != right.right
}
