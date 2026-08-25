package touch

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// GestureSample is a managed value describing one completed XNA touch
// gesture. Constructing it claims no touch capability: CNA-Go exposes no
// TouchPanel, and nothing here reads a device.
type GestureSample struct {
	gestureType GestureType
	timestamp   framework.TimeSpan
	position    framework.Vector2
	position2   framework.Vector2
	delta       framework.Vector2
	delta2      framework.Vector2
}

// NewGestureSample stores every component unchanged. The reference constructor
// performs no validation, normalization, or clamping.
func NewGestureSample(gestureType GestureType, timestamp framework.TimeSpan, position, position2, delta, delta2 framework.Vector2) GestureSample {
	return GestureSample{
		gestureType: gestureType,
		timestamp:   timestamp,
		position:    position,
		position2:   position2,
		delta:       delta,
		delta2:      delta2,
	}
}

func (g GestureSample) GestureType() GestureType      { return g.gestureType }
func (g GestureSample) Timestamp() framework.TimeSpan { return g.timestamp }
func (g GestureSample) Position() framework.Vector2   { return g.position }
func (g GestureSample) Position2() framework.Vector2  { return g.position2 }
func (g GestureSample) Delta() framework.Vector2      { return g.delta }
func (g GestureSample) Delta2() framework.Vector2     { return g.delta2 }
