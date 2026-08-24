package framework

const (
	curveTangentEpsilon    float32 = 1.1920929e-7
	smallestPositiveSingle         = 0x1p-149
)

// Curve is a managed reference-class facade for an XNA scalar curve.
type Curve struct {
	keys     *CurveKeyCollection
	preLoop  CurveLoopType
	postLoop CurveLoopType
}

func NewCurve() *Curve {
	return &Curve{keys: NewCurveKeyCollection()}
}

func (c *Curve) Clone() *Curve {
	return &Curve{
		keys:     c.keys.Clone(),
		preLoop:  c.preLoop,
		postLoop: c.postLoop,
	}
}

func (c *Curve) ComputeTangentByInt32AndCurveTangent(keyIndex int32, tangentType CurveTangent) error {
	return c.ComputeTangentByInt32AndCurveTangentAndCurveTangent(keyIndex, tangentType, tangentType)
}

func (c *Curve) ComputeTangentByInt32AndCurveTangentAndCurveTangent(
	keyIndex int32,
	tangentInType, tangentOutType CurveTangent,
) error {
	index, err := curveCollectionIndex(keyIndex, len(c.keys.keys))
	if err != nil {
		return err
	}
	c.computeTangent(index, tangentInType, tangentOutType)
	return nil
}

func (c *Curve) ComputeTangentsByCurveTangent(tangentType CurveTangent) {
	c.ComputeTangentsByCurveTangentAndCurveTangent(tangentType, tangentType)
}

func (c *Curve) ComputeTangentsByCurveTangentAndCurveTangent(tangentInType, tangentOutType CurveTangent) {
	for index := range c.keys.keys {
		c.computeTangent(index, tangentInType, tangentOutType)
	}
}

func (c *Curve) Evaluate(position float32) float32 {
	count := len(c.keys.keys)
	if count == 0 {
		return 0
	}
	if count == 1 {
		return c.keys.keys[0].value
	}

	first := c.keys.keys[0]
	last := c.keys.keys[count-1]
	virtualPosition := position
	var valueOffset float32

	if virtualPosition < first.position {
		switch c.preLoop {
		case CurveLoopTypeConstant:
			return first.value
		case CurveLoopTypeLinear:
			return first.value - first.tangentIn*(first.position-virtualPosition)
		default:
			timeRange, inverseTimeRange := c.keys.ranges()
			cycle := calculateCurveCycle(virtualPosition, first.position, inverseTimeRange)
			cyclePosition := virtualPosition - (first.position + cycle*timeRange)
			switch c.preLoop {
			case CurveLoopTypeCycle:
				virtualPosition = first.position + cyclePosition
			case CurveLoopTypeCycleOffset:
				virtualPosition = first.position + cyclePosition
				valueOffset = (last.value - first.value) * cycle
			default:
				if int32(cycle)&1 == 0 {
					virtualPosition = first.position + cyclePosition
				} else {
					virtualPosition = last.position - cyclePosition
				}
			}
		}
	} else if last.position < virtualPosition {
		switch c.postLoop {
		case CurveLoopTypeConstant:
			return last.value
		case CurveLoopTypeLinear:
			return last.value - last.tangentOut*(last.position-virtualPosition)
		default:
			timeRange, inverseTimeRange := c.keys.ranges()
			cycle := calculateCurveCycle(virtualPosition, first.position, inverseTimeRange)
			cyclePosition := virtualPosition - (first.position + cycle*timeRange)
			switch c.postLoop {
			case CurveLoopTypeCycle:
				virtualPosition = first.position + cyclePosition
			case CurveLoopTypeCycleOffset:
				virtualPosition = first.position + cyclePosition
				valueOffset = (last.value - first.value) * cycle
			default:
				if int32(cycle)&1 == 0 {
					virtualPosition = first.position + cyclePosition
				} else {
					virtualPosition = last.position - cyclePosition
				}
			}
		}
	}

	start, end, amount := c.findSegment(virtualPosition)
	return valueOffset + curveHermite(start, end, amount)
}

func (c *Curve) PreLoop() CurveLoopType { return c.preLoop }

func (c *Curve) SetPreLoop(value CurveLoopType) { c.preLoop = value }

func (c *Curve) PostLoop() CurveLoopType { return c.postLoop }

func (c *Curve) SetPostLoop(value CurveLoopType) { c.postLoop = value }

func (c *Curve) Keys() *CurveKeyCollection { return c.keys }

func (c *Curve) IsConstant() bool { return len(c.keys.keys) <= 1 }

func (c *Curve) computeTangent(index int, tangentInType, tangentOutType CurveTangent) {
	key := c.keys.keys[index]
	previousPosition, nextPosition := key.position, key.position
	previousValue, nextValue := key.value, key.value
	if index > 0 {
		previousPosition = c.keys.keys[index-1].position
		previousValue = c.keys.keys[index-1].value
	}
	if index+1 < len(c.keys.keys) {
		nextPosition = c.keys.keys[index+1].position
		nextValue = c.keys.keys[index+1].value
	}

	switch tangentInType {
	case CurveTangentSmooth:
		positionSpan := nextPosition - previousPosition
		valueSpan := nextValue - previousValue
		if abs32(valueSpan) < curveTangentEpsilon {
			key.tangentIn = 0
		} else {
			key.tangentIn = valueSpan * abs32(previousPosition-key.position) / positionSpan
		}
	case CurveTangentLinear:
		key.tangentIn = key.value - previousValue
	default:
		key.tangentIn = 0
	}

	switch tangentOutType {
	case CurveTangentSmooth:
		positionSpan := nextPosition - previousPosition
		valueSpan := nextValue - previousValue
		if abs32(valueSpan) < curveTangentEpsilon {
			key.tangentOut = 0
		} else {
			key.tangentOut = valueSpan * abs32(nextPosition-key.position) / positionSpan
		}
	case CurveTangentLinear:
		key.tangentOut = nextValue - key.value
	default:
		key.tangentOut = 0
	}
}

func (c *Curve) findSegment(position float32) (*CurveKey, *CurveKey, float32) {
	amount := position
	start := c.keys.keys[0]
	var end *CurveKey
	for index := 1; index < len(c.keys.keys); index++ {
		end = c.keys.keys[index]
		if end.position >= position {
			startPosition := float64(start.position)
			endPosition := float64(end.position)
			targetPosition := float64(position)
			positionSpan := endPosition - startPosition
			amount = 0
			if positionSpan > 1e-10 {
				amount = float32((targetPosition - startPosition) / positionSpan)
			}
			return start, end, amount
		}
		start = end
	}
	return start, end, amount
}

func curveHermite(start, end *CurveKey, amount float32) float32 {
	if start.continuity == CurveContinuityStep {
		if amount < 1 {
			return start.value
		}
		return end.value
	}
	amountSquared := amount * amount
	amountCubed := amountSquared * amount
	first := ((2 * amountCubed) - (3 * amountSquared)) + 1
	second := (-2 * amountCubed) + (3 * amountSquared)
	third := (amountCubed - (2 * amountSquared)) + amount
	fourth := amountCubed - amountSquared
	return start.value*first + end.value*second + start.tangentOut*third + end.tangentIn*fourth
}

func calculateCurveCycle(position, firstPosition, inverseTimeRange float32) float32 {
	cycle := (position - firstPosition) * inverseTimeRange
	if cycle < 0 {
		cycle -= 1
	}
	return float32(int32(cycle))
}
