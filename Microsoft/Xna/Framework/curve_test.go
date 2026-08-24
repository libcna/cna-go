package framework

import (
	"math"
	"testing"
)

func addCurveKey(t *testing.T, collection *CurveKeyCollection, key *CurveKey) {
	t.Helper()
	if err := collection.Add(key); err != nil {
		t.Fatalf("Add: %v", err)
	}
}

func curveKeyAt(t *testing.T, collection *CurveKeyCollection, index int32) *CurveKey {
	t.Helper()
	key, err := collection.Item(index)
	if err != nil {
		t.Fatalf("Item(%d): %v", index, err)
	}
	return key
}

func TestCurveEnumsConstructorsAndProperties(t *testing.T) {
	if CurveContinuitySmooth != 0 || CurveContinuityStep != 1 {
		t.Fatalf("continuity values = %d/%d", CurveContinuitySmooth, CurveContinuityStep)
	}
	if CurveTangentFlat != 0 || CurveTangentLinear != 1 || CurveTangentSmooth != 2 {
		t.Fatalf("tangent values = %d/%d/%d", CurveTangentFlat, CurveTangentLinear, CurveTangentSmooth)
	}
	if CurveLoopTypeConstant != 0 || CurveLoopTypeCycle != 1 || CurveLoopTypeCycleOffset != 2 ||
		CurveLoopTypeOscillate != 3 || CurveLoopTypeLinear != 4 {
		t.Fatalf("loop values = %d/%d/%d/%d/%d", CurveLoopTypeConstant, CurveLoopTypeCycle,
			CurveLoopTypeCycleOffset, CurveLoopTypeOscillate, CurveLoopTypeLinear)
	}

	short := NewCurveKeyBySingleAndSingle(1, 2)
	if short.Position() != 1 || short.Value() != 2 || short.TangentIn() != 0 ||
		short.TangentOut() != 0 || short.Continuity() != CurveContinuitySmooth {
		t.Fatalf("short constructor = %+v", short)
	}
	middle := NewCurveKeyBySingleAndSingleAndSingleAndSingle(1, 2, 3, 4)
	if middle.TangentIn() != 3 || middle.TangentOut() != 4 || middle.Continuity() != CurveContinuitySmooth {
		t.Fatalf("middle constructor = %+v", middle)
	}
	full := NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(1, 2, 3, 4, CurveContinuityStep)
	full.SetValue(20)
	full.SetTangentIn(30)
	full.SetTangentOut(40)
	full.SetContinuity(CurveContinuitySmooth)
	if full.Position() != 1 || full.Value() != 20 || full.TangentIn() != 30 ||
		full.TangentOut() != 40 || full.Continuity() != CurveContinuitySmooth {
		t.Fatalf("properties = %+v", full)
	}
}

func TestCurveKeyCloneEqualityHashAndOperators(t *testing.T) {
	original := NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(1, 2, 3, 4, CurveContinuityStep)
	clone := original.Clone()
	if clone == original || !clone.EqualsByCurveKey(original) || !clone.EqualsByObject(original) {
		t.Fatalf("clone identity/equality = cloneSame:%t typed:%t object:%t", clone == original,
			clone.EqualsByCurveKey(original), clone.EqualsByObject(original))
	}
	clone.SetValue(9)
	if original.Value() != 2 || clone.Value() != 9 {
		t.Fatalf("clone scalar independence = original %v clone %v", original.Value(), clone.Value())
	}
	if original.GetHashCode() != 4194305 {
		t.Fatalf("hash = %d, want 4194305", original.GetHashCode())
	}
	if original.EqualsByCurveKey(NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(1, 2, 3, 5, CurveContinuityStep)) {
		t.Fatal("one-field difference compared equal")
	}
	if !NewCurveKeyBySingleAndSingle(float32(0), 2).EqualsByCurveKey(NewCurveKeyBySingleAndSingle(float32(math.Copysign(0, -1)), 2)) {
		t.Fatal("signed zeros did not compare equal")
	}
	nan := float32(math.NaN())
	nanKey := NewCurveKeyBySingleAndSingle(nan, 1)
	if nanKey.EqualsByCurveKey(nanKey) || nanKey.EqualsByObject(nanKey) {
		t.Fatal("NaN field unexpectedly compared equal")
	}
	if !CurveKeyOperatorEqualityByCurveKeyAndCurveKey(nil, nil) ||
		CurveKeyOperatorEqualityByCurveKeyAndCurveKey(nil, original) ||
		!CurveKeyOperatorInequalityByCurveKeyAndCurveKey(nil, original) ||
		!CurveKeyOperatorEqualityByCurveKeyAndCurveKey(original, original.Clone()) {
		t.Fatal("operator equality/inequality semantics differ")
	}
}

func TestCurveKeyCompareToSingleSemantics(t *testing.T) {
	nan := float32(math.NaN())
	cases := []struct {
		name        string
		left, right float32
		want        int32
	}{
		{"finite-less", -2, 3, -1},
		{"finite-equal", 3, 3, 0},
		{"finite-greater", 3, -2, 1},
		{"positive-negative-zero", 0, float32(math.Copysign(0, -1)), 0},
		{"nan-finite", nan, 1, -1},
		{"finite-nan", 1, nan, 1},
		{"nan-nan", nan, nan, 0},
		{"negative-infinity", float32(math.Inf(-1)), -100, -1},
		{"positive-infinity", float32(math.Inf(1)), 100, 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewCurveKeyBySingleAndSingle(test.left, 0).CompareTo(NewCurveKeyBySingleAndSingle(test.right, 0))
			if err != nil || got != test.want {
				t.Fatalf("CompareTo = %d, %v; want %d, nil", got, err, test.want)
			}
		})
	}
	if _, err := NewCurveKeyBySingleAndSingle(0, 0).CompareTo(nil); err == nil {
		t.Fatal("CompareTo(nil) succeeded")
	}
}

func TestCurveKeyCollectionSortedReferenceAndEqualitySemantics(t *testing.T) {
	keys := NewCurveKeyCollection()
	middleA := NewCurveKeyBySingleAndSingle(1, 10)
	middleB := NewCurveKeyBySingleAndSingle(1, 20)
	addCurveKey(t, keys, NewCurveKeyBySingleAndSingle(2, 30))
	addCurveKey(t, keys, middleA)
	addCurveKey(t, keys, NewCurveKeyBySingleAndSingle(0, 0))
	addCurveKey(t, keys, middleB)

	wantPositions := []float32{0, 1, 1, 2}
	for index, want := range wantPositions {
		if got := curveKeyAt(t, keys, int32(index)).Position(); got != want {
			t.Fatalf("position[%d] = %v, want %v", index, got, want)
		}
	}
	if curveKeyAt(t, keys, 1) != middleA || curveKeyAt(t, keys, 2) != middleB {
		t.Fatal("equal-position insertion did not retain after-equals order and identity")
	}
	middleA.SetValue(11)
	equalDistinct := NewCurveKeyBySingleAndSingle(1, 11)
	if !keys.Contains(equalDistinct) || keys.IndexOf(equalDistinct) != 1 {
		t.Fatal("Contains/IndexOf did not use CurveKey value equality")
	}
	if !keys.Remove(equalDistinct) || keys.Contains(middleA) || keys.Remove(equalDistinct) {
		t.Fatal("Remove did not remove the first value-equal key exactly once")
	}
	addCurveKey(t, keys, middleB)
	if keys.Count() != 4 {
		t.Fatalf("same reference could not be stored twice: count %d", keys.Count())
	}
}

func TestCurveKeyCollectionItemReordersAndValidates(t *testing.T) {
	keys := NewCurveKeyCollection()
	for _, position := range []float32{0, 2, 4} {
		addCurveKey(t, keys, NewCurveKeyBySingleAndSingle(position, position))
	}
	replacement := NewCurveKeyBySingleAndSingle(3, 30)
	if err := keys.SetItem(0, replacement); err != nil {
		t.Fatal(err)
	}
	for index, want := range []float32{2, 3, 4} {
		if curveKeyAt(t, keys, int32(index)).Position() != want {
			t.Fatalf("reordered positions differ at %d", index)
		}
	}
	samePosition := NewCurveKeyBySingleAndSingle(3, 300)
	if err := keys.SetItem(1, samePosition); err != nil {
		t.Fatal(err)
	}
	if curveKeyAt(t, keys, 1) != samePosition {
		t.Fatal("same-position replacement lost object identity")
	}

	for _, index := range []int32{-1, keys.Count()} {
		if _, err := keys.Item(index); err == nil {
			t.Fatalf("Item(%d) succeeded", index)
		}
		if err := keys.SetItem(index, replacement); err == nil {
			t.Fatalf("SetItem(%d) succeeded", index)
		}
		if err := keys.RemoveAt(index); err == nil {
			t.Fatalf("RemoveAt(%d) succeeded", index)
		}
	}
	if err := keys.SetItem(0, nil); err == nil || keys.Add(nil) == nil {
		t.Fatal("nil key mutation succeeded")
	}
}

func TestCurveKeyCollectionCopyRemoveClearAndClone(t *testing.T) {
	keys := NewCurveKeyCollection()
	first := NewCurveKeyBySingleAndSingle(0, 1)
	second := NewCurveKeyBySingleAndSingle(2, 3)
	addCurveKey(t, keys, first)
	addCurveKey(t, keys, second)

	if keys.CopyTo(nil, 0) == nil || keys.CopyTo(make([]*CurveKey, 2), -1) == nil ||
		keys.CopyTo(make([]*CurveKey, 2), 3) == nil || keys.CopyTo(make([]*CurveKey, 2), 1) == nil {
		t.Fatal("CopyTo accepted invalid destination/index/capacity")
	}
	destination := make([]*CurveKey, 4)
	if err := keys.CopyTo(destination, 1); err != nil {
		t.Fatal(err)
	}
	if destination[1] != first || destination[2] != second {
		t.Fatal("CopyTo did not preserve references and caller storage")
	}
	empty := NewCurveKeyCollection()
	if err := empty.CopyTo(make([]*CurveKey, 0), 0); err != nil {
		t.Fatalf("empty CopyTo exact capacity: %v", err)
	}

	clone := keys.Clone()
	if clone == keys || curveKeyAt(t, clone, 0) != first {
		t.Fatal("collection Clone was not a new shallow collection")
	}
	curveKeyAt(t, clone, 0).SetValue(42)
	if curveKeyAt(t, keys, 0).Value() != 42 {
		t.Fatal("collection Clone unexpectedly cloned CurveKey objects")
	}
	if err := clone.RemoveAt(0); err != nil {
		t.Fatal(err)
	}
	if clone.Count() != 1 || keys.Count() != 2 {
		t.Fatal("collection Clone storage was not independent")
	}

	if err := keys.RemoveAt(0); err != nil || keys.Count() != 1 || curveKeyAt(t, keys, 0) != second {
		t.Fatal("RemoveAt first failed")
	}
	keys.Clear()
	if keys.Count() != 0 || keys.IsReadOnly() {
		t.Fatalf("Clear/IsReadOnly = count %d readOnly %t", keys.Count(), keys.IsReadOnly())
	}
	keys.Clear()
	if keys.Count() != 0 {
		t.Fatal("clearing an empty collection changed count")
	}
}

func TestCurveKeyCollectionIteratorOrderFreshnessAndInvalidation(t *testing.T) {
	keys := NewCurveKeyCollection()
	first := NewCurveKeyBySingleAndSingle(0, 0)
	second := NewCurveKeyBySingleAndSingle(1, 1)
	addCurveKey(t, keys, first)
	addCurveKey(t, keys, second)
	var _ Iterator[*CurveKey] = keys.GetEnumerator()

	iterator := keys.GetEnumerator()
	value, ok, err := iterator.Next()
	if err != nil || !ok || value != first {
		t.Fatalf("first Next = %p, %t, %v", value, ok, err)
	}
	addCurveKey(t, keys, NewCurveKeyBySingleAndSingle(2, 2))
	if _, _, err := iterator.Next(); err == nil {
		t.Fatal("mutation during enumeration was not reported")
	}

	fresh := keys.GetEnumerator()
	for index, want := range []*CurveKey{first, second, curveKeyAt(t, keys, 2)} {
		value, ok, err = fresh.Next()
		if err != nil || !ok || value != want {
			t.Fatalf("fresh Next[%d] = %p, %t, %v", index, value, ok, err)
		}
	}
	if value, ok, err = fresh.Next(); err != nil || ok || value != nil {
		t.Fatalf("completed Next = %p, %t, %v", value, ok, err)
	}

	nonMutating := keys.GetEnumerator()
	if err := keys.CopyTo(make([]*CurveKey, 3), 0); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := nonMutating.Next(); err != nil || !ok {
		t.Fatalf("CopyTo invalidated iterator: ok %t err %v", ok, err)
	}
	nonMutating = keys.GetEnumerator()
	if keys.Remove(NewCurveKeyBySingleAndSingle(99, 99)) {
		t.Fatal("absent key removed")
	}
	if _, ok, err := nonMutating.Next(); err != nil || !ok {
		t.Fatalf("failed Remove invalidated iterator: ok %t err %v", ok, err)
	}

	replaced := keys.GetEnumerator()
	if err := keys.SetItem(0, NewCurveKeyBySingleAndSingle(0, 10)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := replaced.Next(); err == nil {
		t.Fatal("Item replacement did not invalidate iterator")
	}
}

func TestCurveDefaultsKeysIdentityIsConstantAndClone(t *testing.T) {
	curve := NewCurve()
	if curve.PreLoop() != CurveLoopTypeConstant || curve.PostLoop() != CurveLoopTypeConstant ||
		!curve.IsConstant() || curve.Evaluate(5) != 0 {
		t.Fatalf("defaults = pre %d post %d constant %t evaluate %v", curve.PreLoop(), curve.PostLoop(), curve.IsConstant(), curve.Evaluate(5))
	}
	if curve.Keys() != curve.Keys() {
		t.Fatal("Keys identity was not stable")
	}
	curve.SetPreLoop(CurveLoopTypeCycle)
	curve.SetPostLoop(CurveLoopTypeLinear)
	key := NewCurveKeyBySingleAndSingle(0, 1)
	addCurveKey(t, curve.Keys(), key)
	if !curve.IsConstant() {
		t.Fatal("one-key curve was not constant")
	}
	addCurveKey(t, curve.Keys(), NewCurveKeyBySingleAndSingle(1, 1))
	if curve.IsConstant() {
		t.Fatal("two-key equal-value curve was reported constant")
	}
	clone := curve.Clone()
	if clone == curve || clone.Keys() == curve.Keys() || clone.Keys().Count() != 2 ||
		clone.PreLoop() != CurveLoopTypeCycle || clone.PostLoop() != CurveLoopTypeLinear {
		t.Fatal("Curve Clone identity/state differs")
	}
	if curveKeyAt(t, clone.Keys(), 0) != key {
		t.Fatal("Curve Clone did not retain shallow key identity")
	}
	curveKeyAt(t, clone.Keys(), 0).SetValue(42)
	if curveKeyAt(t, curve.Keys(), 0).Value() != 42 {
		t.Fatal("Curve Clone unexpectedly deep-cloned keys")
	}
	if err := clone.Keys().RemoveAt(1); err != nil || curve.Keys().Count() != 2 || clone.Keys().Count() != 1 {
		t.Fatal("Curve Clone collection storage was not independent")
	}
}

func TestCurveTangentsFlatLinearSmoothAndMixed(t *testing.T) {
	curve := NewCurve()
	for _, pair := range [][2]float32{{0, 0}, {1, 10}, {3, 30}} {
		addCurveKey(t, curve.Keys(), NewCurveKeyBySingleAndSingle(pair[0], pair[1]))
	}
	if err := curve.ComputeTangentByInt32AndCurveTangent(1, CurveTangentFlat); err != nil {
		t.Fatal(err)
	}
	middle := curveKeyAt(t, curve.Keys(), 1)
	if middle.TangentIn() != 0 || middle.TangentOut() != 0 {
		t.Fatal("Flat tangent was nonzero")
	}
	if err := curve.ComputeTangentByInt32AndCurveTangent(1, CurveTangentLinear); err != nil {
		t.Fatal(err)
	}
	if middle.TangentIn() != 10 || middle.TangentOut() != 20 {
		t.Fatalf("Linear tangent = %v/%v", middle.TangentIn(), middle.TangentOut())
	}
	curve.ComputeTangentsByCurveTangent(CurveTangentSmooth)
	want := [][2]float32{{0, 10}, {10, 20}, {20, 0}}
	for index, tangents := range want {
		key := curveKeyAt(t, curve.Keys(), int32(index))
		if key.TangentIn() != tangents[0] || key.TangentOut() != tangents[1] {
			t.Fatalf("Smooth[%d] = %v/%v, want %v/%v", index, key.TangentIn(), key.TangentOut(), tangents[0], tangents[1])
		}
	}
	if err := curve.ComputeTangentByInt32AndCurveTangentAndCurveTangent(1, CurveTangentFlat, CurveTangentLinear); err != nil {
		t.Fatal(err)
	}
	if middle.TangentIn() != 0 || middle.TangentOut() != 20 {
		t.Fatalf("mixed Flat/Linear = %v/%v", middle.TangentIn(), middle.TangentOut())
	}
	if err := curve.ComputeTangentByInt32AndCurveTangentAndCurveTangent(1, CurveTangentLinear, CurveTangentSmooth); err != nil {
		t.Fatal(err)
	}
	if middle.TangentIn() != 10 || middle.TangentOut() != 20 {
		t.Fatalf("mixed Linear/Smooth = %v/%v", middle.TangentIn(), middle.TangentOut())
	}
	curve.ComputeTangentsByCurveTangentAndCurveTangent(CurveTangentSmooth, CurveTangentFlat)
	firstPass := [][2]float32{
		{curveKeyAt(t, curve.Keys(), 0).TangentIn(), curveKeyAt(t, curve.Keys(), 0).TangentOut()},
		{middle.TangentIn(), middle.TangentOut()},
		{curveKeyAt(t, curve.Keys(), 2).TangentIn(), curveKeyAt(t, curve.Keys(), 2).TangentOut()},
	}
	curve.ComputeTangentsByCurveTangentAndCurveTangent(CurveTangentSmooth, CurveTangentFlat)
	for index, previous := range firstPass {
		key := curveKeyAt(t, curve.Keys(), int32(index))
		if key.TangentIn() != previous[0] || key.TangentOut() != previous[1] {
			t.Fatalf("repeated ComputeTangents changed key %d", index)
		}
	}
	for _, index := range []int32{-1, curve.Keys().Count()} {
		if err := curve.ComputeTangentByInt32AndCurveTangent(index, CurveTangentFlat); err == nil {
			t.Fatalf("ComputeTangent(%d) succeeded", index)
		}
	}

	single := NewCurve()
	addCurveKey(t, single.Keys(), NewCurveKeyBySingleAndSingle(2, 7))
	single.ComputeTangentsByCurveTangent(CurveTangentSmooth)
	if curveKeyAt(t, single.Keys(), 0).TangentIn() != 0 || curveKeyAt(t, single.Keys(), 0).TangentOut() != 0 {
		t.Fatal("single-key smooth tangents were nonzero")
	}
}

func TestCurveEvaluateEmptySingleSegmentsStepAndHermite(t *testing.T) {
	empty := NewCurve()
	for _, position := range []float32{-1, 0, 1, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		if got := empty.Evaluate(position); got != 0 {
			t.Fatalf("empty Evaluate(%v) = %v", position, got)
		}
	}
	single := NewCurve()
	addCurveKey(t, single.Keys(), NewCurveKeyBySingleAndSingle(2, 7))
	for _, position := range []float32{-1, 2, 9, float32(math.NaN()), float32(math.Inf(1))} {
		if got := single.Evaluate(position); got != 7 {
			t.Fatalf("single Evaluate(%v) = %v", position, got)
		}
	}

	ordinary := NewCurve()
	addCurveKey(t, ordinary.Keys(), NewCurveKeyBySingleAndSingle(0, 0))
	addCurveKey(t, ordinary.Keys(), NewCurveKeyBySingleAndSingle(1, 10))
	addCurveKey(t, ordinary.Keys(), NewCurveKeyBySingleAndSingle(3, 20))
	if got := math.Float32bits(ordinary.Evaluate(0.25)); got != 0x3fc80000 {
		t.Fatalf("Hermite quarter bits = %#08x", got)
	}
	below := math.Float32frombits(math.Float32bits(1) - 1)
	above := math.Float32frombits(math.Float32bits(1) + 1)
	belowStart, belowEnd, _ := ordinary.findSegment(below)
	aboveStart, aboveEnd, _ := ordinary.findSegment(above)
	if ordinary.Evaluate(0) != 0 || ordinary.Evaluate(1) != 10 || ordinary.Evaluate(3) != 20 ||
		belowStart != curveKeyAt(t, ordinary.Keys(), 0) || belowEnd != curveKeyAt(t, ordinary.Keys(), 1) ||
		aboveStart != curveKeyAt(t, ordinary.Keys(), 1) || aboveEnd != curveKeyAt(t, ordinary.Keys(), 2) {
		t.Fatal("segment boundary selection differs")
	}

	asymmetric := NewCurve()
	addCurveKey(t, asymmetric.Keys(), NewCurveKeyBySingleAndSingleAndSingleAndSingle(0, 0, 99, 4))
	addCurveKey(t, asymmetric.Keys(), NewCurveKeyBySingleAndSingleAndSingleAndSingle(2, 10, -2, 77))
	if got := math.Float32bits(asymmetric.Evaluate(1)); got != 0x40b80000 {
		t.Fatalf("asymmetric Hermite bits = %#08x", got)
	}
	curveKeyAt(t, asymmetric.Keys(), 0).SetTangentOut(float32(math.NaN()))
	if !math.IsNaN(float64(asymmetric.Evaluate(1))) {
		t.Fatal("NaN tangent did not propagate through Hermite")
	}

	step := NewCurve()
	addCurveKey(t, step.Keys(), NewCurveKeyBySingleAndSingleAndSingleAndSingleAndCurveContinuity(0, 2, 0, 0, CurveContinuityStep))
	addCurveKey(t, step.Keys(), NewCurveKeyBySingleAndSingle(1, 9))
	if step.Evaluate(0) != 2 || step.Evaluate(0.999) != 2 || step.Evaluate(math.Float32frombits(math.Float32bits(1)-1)) != 2 || step.Evaluate(1) != 9 {
		t.Fatal("Step continuity threshold differs")
	}

	duplicates := NewCurve()
	addCurveKey(t, duplicates.Keys(), NewCurveKeyBySingleAndSingle(1, 10))
	addCurveKey(t, duplicates.Keys(), NewCurveKeyBySingleAndSingle(1, 20))
	if duplicates.Evaluate(1) != 10 {
		t.Fatalf("duplicate exact evaluation = %v", duplicates.Evaluate(1))
	}
}

func TestCurveLoopModesAndNegativeCycles(t *testing.T) {
	newTwoKey := func() *Curve {
		curve := NewCurve()
		addCurveKey(t, curve.Keys(), NewCurveKeyBySingleAndSingle(5, 0))
		addCurveKey(t, curve.Keys(), NewCurveKeyBySingleAndSingle(7, 10))
		return curve
	}
	tests := []struct {
		mode              CurveLoopType
		preWant, postWant float32
	}{
		{CurveLoopTypeConstant, 0, 10},
		{CurveLoopTypeCycle, 5, 5},
		{CurveLoopTypeCycleOffset, -5, 15},
		{CurveLoopTypeOscillate, 5, 5},
	}
	for _, test := range tests {
		curve := newTwoKey()
		curve.SetPreLoop(test.mode)
		curve.SetPostLoop(test.mode)
		if got := curve.Evaluate(4); got != test.preWant {
			t.Fatalf("pre mode %d = %v, want %v", test.mode, got, test.preWant)
		}
		if got := curve.Evaluate(8); got != test.postWant {
			t.Fatalf("post mode %d = %v, want %v", test.mode, got, test.postWant)
		}
	}
	linear := newTwoKey()
	curveKeyAt(t, linear.Keys(), 0).SetTangentIn(2)
	curveKeyAt(t, linear.Keys(), 1).SetTangentOut(3)
	linear.SetPreLoop(CurveLoopTypeLinear)
	linear.SetPostLoop(CurveLoopTypeLinear)
	if linear.Evaluate(4) != -2 || linear.Evaluate(9) != 16 {
		t.Fatalf("linear loop = %v/%v", linear.Evaluate(4), linear.Evaluate(9))
	}

	cycle := newTwoKey()
	cycle.SetPreLoop(CurveLoopTypeCycle)
	offset := newTwoKey()
	offset.SetPreLoop(CurveLoopTypeCycleOffset)
	oscillate := newTwoKey()
	oscillate.SetPreLoop(CurveLoopTypeOscillate)
	if cycle.Evaluate(3) != 10 || offset.Evaluate(3) != -10 || oscillate.Evaluate(3) != 10 {
		t.Fatalf("negative exact cycle = %v/%v/%v", cycle.Evaluate(3), offset.Evaluate(3), oscillate.Evaluate(3))
	}
	postOffset := newTwoKey()
	postOffset.SetPostLoop(CurveLoopTypeCycleOffset)
	if postOffset.Evaluate(9) != 20 {
		t.Fatalf("positive multiple CycleOffset = %v", postOffset.Evaluate(9))
	}
	negativeMultiple := newTwoKey()
	negativeMultiple.SetPreLoop(CurveLoopTypeCycleOffset)
	if negativeMultiple.Evaluate(1) != -20 {
		t.Fatalf("negative multiple CycleOffset = %v", negativeMultiple.Evaluate(1))
	}
	equalValues := NewCurve()
	addCurveKey(t, equalValues.Keys(), NewCurveKeyBySingleAndSingle(5, 4))
	addCurveKey(t, equalValues.Keys(), NewCurveKeyBySingleAndSingle(7, 4))
	equalValues.SetPreLoop(CurveLoopTypeCycleOffset)
	equalValues.SetPostLoop(CurveLoopTypeCycleOffset)
	if equalValues.Evaluate(-1) != 4 || equalValues.Evaluate(13) != 4 {
		t.Fatal("equal-endpoint CycleOffset introduced an offset")
	}

	unit := NewCurve()
	addCurveKey(t, unit.Keys(), NewCurveKeyBySingleAndSingle(0, 0))
	addCurveKey(t, unit.Keys(), NewCurveKeyBySingleAndSingle(1, 10))
	unit.SetPreLoop(CurveLoopTypeCycle)
	if unit.Evaluate(-1) != 10 || unit.Evaluate(-2) != 10 ||
		math.Float32bits(unit.Evaluate(-0.1)) != math.Float32bits(unit.Evaluate(-1.1)) {
		t.Fatalf("negative cycle decrement = -0.1 %v -1 %v -1.1 %v -2 %v",
			unit.Evaluate(-0.1), unit.Evaluate(-1), unit.Evaluate(-1.1), unit.Evaluate(-2))
	}
	unit.SetPreLoop(CurveLoopTypeOscillate)
	if unit.Evaluate(-1) != 10 || unit.Evaluate(-2) != 0 {
		t.Fatalf("negative oscillation parity = -1 %v -2 %v", unit.Evaluate(-1), unit.Evaluate(-2))
	}
}
