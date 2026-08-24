package framework

// CurveKey is a managed reference-class facade for one curve control point.
type CurveKey struct {
	position   float32
	value      float32
	tangentIn  float32
	tangentOut float32
	continuity CurveContinuity
}

func NewCurveKeyBySingleAndSingle(position, value float32) *CurveKey {
	return NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(
		position, value, 0, 0, CurveContinuitySmooth,
	)
}

func NewCurveKeyBySingleAndSingleAndSingleAndSingle(position, value, tangentIn, tangentOut float32) *CurveKey {
	return NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(
		position, value, tangentIn, tangentOut, CurveContinuitySmooth,
	)
}

func NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(
	position, value, tangentIn, tangentOut float32,
	continuity CurveContinuity,
) *CurveKey {
	return &CurveKey{
		position:   position,
		value:      value,
		tangentIn:  tangentIn,
		tangentOut: tangentOut,
		continuity: continuity,
	}
}

func (k *CurveKey) Position() float32 { return k.position }

func (k *CurveKey) Value() float32 { return k.value }

func (k *CurveKey) SetValue(value float32) { k.value = value }

func (k *CurveKey) TangentIn() float32 { return k.tangentIn }

func (k *CurveKey) SetTangentIn(value float32) { k.tangentIn = value }

func (k *CurveKey) TangentOut() float32 { return k.tangentOut }

func (k *CurveKey) SetTangentOut(value float32) { k.tangentOut = value }

func (k *CurveKey) Continuity() CurveContinuity { return k.continuity }

func (k *CurveKey) SetContinuity(value CurveContinuity) { k.continuity = value }

func (k *CurveKey) Clone() *CurveKey {
	return &CurveKey{
		position:   k.position,
		value:      k.value,
		tangentIn:  k.tangentIn,
		tangentOut: k.tangentOut,
		continuity: k.continuity,
	}
}

func (k *CurveKey) EqualsByCurveKey(other *CurveKey) bool {
	return k != nil && other != nil &&
		k.position == other.position &&
		k.value == other.value &&
		k.tangentIn == other.tangentIn &&
		k.tangentOut == other.tangentOut &&
		k.continuity == other.continuity
}

func (k *CurveKey) EqualsByObject(value any) bool {
	other, ok := value.(*CurveKey)
	return ok && k.EqualsByCurveKey(other)
}

func (k *CurveKey) GetHashCode() int32 {
	hash := singleHashCode(k.position)
	hash += singleHashCode(k.value)
	hash += singleHashCode(k.tangentIn)
	hash += singleHashCode(k.tangentOut)
	hash += int32(k.continuity)
	return hash
}

func (k *CurveKey) CompareTo(other *CurveKey) (int32, error) {
	if other == nil {
		return 0, curveArgumentError("other is nil")
	}
	return compareSingle(k.position, other.position), nil
}

func CurveKeyOperatorEqualityByCurveKeyAndCurveKey(left, right *CurveKey) bool {
	if left == nil {
		return right == nil
	}
	return left.EqualsByCurveKey(right)
}

func CurveKeyOperatorInequalityByCurveKeyAndCurveKey(left, right *CurveKey) bool {
	return !CurveKeyOperatorEqualityByCurveKeyAndCurveKey(left, right)
}
