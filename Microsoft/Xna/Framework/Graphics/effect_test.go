package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// ---------------------------------------------------------------------------
// Foundation 72 — the Effect cluster's managed half.
// ---------------------------------------------------------------------------
//
// Every object here is built from its fields directly, which is what a CNA load
// produces. The native half -- a real stock effect loaded, its technique's pass
// applied, and a draw that then SUCCEEDS where the same draw refused a moment
// earlier -- is the `vertex-buffer` stress scenario's job.

// testEffect builds a two-technique effect whose graph is the shape a real one
// has, with no CNA handle anywhere: nothing below reaches one.
func testEffect() *Effect {
	effect := &Effect{resource: &GraphicsResource{}}
	first := &EffectTechnique{effect: effect, name: "Default"}
	second := &EffectTechnique{effect: effect, name: "Other"}
	first.passes = &EffectPassCollection{passes: []*EffectPass{
		{technique: first, name: "P0"}, {technique: first, name: "P1"},
	}}
	second.passes = &EffectPassCollection{passes: []*EffectPass{{technique: second, name: "P0"}}}
	effect.techniques = &EffectTechniqueCollection{techniques: []*EffectTechnique{first, second}}
	effect.currentTechnique = first
	effect.parameters = &EffectParameterCollection{parameters: []*EffectParameter{
		{name: "World", semantic: "WORLD", parameterClass: EffectParameterClassMatrix, rowCount: 4, columnCount: 4},
		{name: "Alpha", semantic: "alpha", parameterClass: EffectParameterClassScalar, rowCount: 1, columnCount: 1},
	}}
	for _, parameter := range effect.parameters.parameters {
		parameter.elements = &EffectParameterCollection{}
		parameter.structureMembers = &EffectParameterCollection{}
		parameter.annotations = &EffectAnnotationCollection{}
	}
	return effect
}

// TestBothIndexersAnswerNilRatherThanThrowing pins the behaviour a reader would
// least expect and every one of the four collections shares:
//
//	get_Item(int32)   index < 0 || index >= Count  ->  ldnull; ret
//	get_Item(string)  nothing matches              ->  ldnull; ret
//
// A BCL List<T> throws for an out-of-range index; these collections check the
// range THEMSELVES first and answer null instead.
func TestBothIndexersAnswerNilRatherThanThrowing(t *testing.T) {
	effect := testEffect()
	techniques := effect.Techniques()
	if techniques.ItemPropertySignatureA5F4623F(-1) != nil || techniques.ItemPropertySignatureA5F4623F(2) != nil {
		t.Fatal("an out-of-range technique index answered a technique")
	}
	if techniques.ItemPropertySignatureDA594950("nope") != nil {
		t.Fatal("an unknown technique name answered a technique")
	}
	passes := techniques.ItemPropertySignatureA5F4623F(0).Passes()
	if passes.ItemPropertySignatureCA1DC5FC(-1) != nil || passes.ItemPropertySignatureCA1DC5FC(2) != nil ||
		passes.ItemPropertySignatureE281298D("nope") != nil {
		t.Fatal("an out-of-range or unknown pass answered a pass")
	}
	parameters := effect.Parameters()
	if parameters.ItemPropertySignatureC421CB56(-1) != nil || parameters.ItemPropertySignatureC421CB56(2) != nil ||
		parameters.ItemPropertySignatureD47463F7("nope") != nil {
		t.Fatal("an out-of-range or unknown parameter answered a parameter")
	}
	annotations := parameters.ItemPropertySignatureC421CB56(0).Annotations()
	if annotations.ItemPropertySignature771818B0(0) != nil || annotations.ItemPropertySignatureFA6B0951("nope") != nil {
		t.Fatal("an empty annotation collection answered an annotation")
	}
}

// TestTheStringIndexerIsCaseSensitiveAndGetParameterBySemanticIsNot pins the
// reference's own inconsistency, which lives three hundred lines apart in one
// assembly:
//
//	get_Item(string)        String.op_Equality               ORDINAL
//	GetParameterBySemantic  String.Compare(a, b, Ordinal-
//	                        IgnoreCase)                      case-INSENSITIVE
//
// A projection that used one comparison for both would be wrong for exactly one
// of them, and nothing else in the type would show it.
func TestTheStringIndexerIsCaseSensitiveAndGetParameterBySemanticIsNot(t *testing.T) {
	parameters := testEffect().Parameters()
	if parameters.ItemPropertySignatureD47463F7("World") == nil {
		t.Fatal("the exact name did not match")
	}
	if parameters.ItemPropertySignatureD47463F7("world") != nil {
		t.Fatal("the string indexer matched a different case; op_Equality is ordinal")
	}
	if parameters.GetParameterBySemantic("WORLD") == nil || parameters.GetParameterBySemantic("world") == nil {
		t.Fatal("GetParameterBySemantic did not match ignoring case")
	}
	// And its counterpart: the semantic stored in lower case is found upper.
	if parameters.GetParameterBySemantic("ALPHA") == nil {
		t.Fatal("GetParameterBySemantic did not match a lower-case semantic")
	}
}

// TestOrdinalIgnoreCaseFoldsOnlyASCII pins the comparison itself. Go's
// strings.EqualFold folds by UNICODE rules, so it makes 'K' equal to the Kelvin
// sign U+212A; StringComparison.OrdinalIgnoreCase folds only A-Z and does not.
func TestOrdinalIgnoreCaseFoldsOnlyASCII(t *testing.T) {
	if !equalsOrdinalIgnoreCase("Alpha", "aLPHA") {
		t.Fatal("ASCII case folding failed")
	}
	if equalsOrdinalIgnoreCase("K", "K") {
		t.Fatal("the Kelvin sign folded to K; OrdinalIgnoreCase folds only ASCII")
	}
	if equalsOrdinalIgnoreCase("a", "ab") {
		t.Fatal("different lengths compared equal")
	}
}

// TestApplyRefusesAPassFromAnotherTechnique pins EffectPass::Apply's second
// guard and its exact message.
func TestApplyRefusesAPassFromAnotherTechnique(t *testing.T) {
	effect := testEffect()
	other := effect.Techniques().ItemPropertySignatureA5F4623F(1)
	err := other.Passes().ItemPropertySignatureCA1DC5FC(0).Apply()
	if err == nil {
		t.Fatal("a pass from a technique that is not current was applied")
	}
	if !strings.Contains(err.Error(), notCurrentTechnique) {
		t.Fatalf("%v, want FrameworkResources.NotCurrentTechnique", err)
	}
	if !errors.Is(err, errSpriteInvalidOperation) {
		t.Fatalf("%v, want the InvalidOperationException projection", err)
	}
}

// TestApplyChecksDisposalBeforeTheTechnique pins the order: CheckDisposed is at
// IL_001c and the technique comparison at IL_0022, so a pass from the wrong
// technique on a DISPOSED effect reports the disposal.
func TestApplyChecksDisposalBeforeTheTechnique(t *testing.T) {
	effect := testEffect()
	_ = effect.resource.DisposeByBoolean(true)
	err := effect.Techniques().ItemPropertySignatureA5F4623F(1).Passes().ItemPropertySignatureCA1DC5FC(0).Apply()
	if !errors.Is(err, errObjectDisposed) {
		t.Fatalf("%v, want the disposal reported before the technique", err)
	}
}

// TestSetCurrentTechniqueReturnsBeforeTheParentCheckForTheCurrentOne pins the
// early return at IL_002d, which is BEFORE the `value._parent != this` check --
// so assigning the technique the effect already holds never reaches the parent
// comparison.
func TestSetCurrentTechniqueReturnsBeforeTheParentCheckForTheCurrentOne(t *testing.T) {
	effect := testEffect()
	current := effect.CurrentTechnique()
	if err := effect.SetCurrentTechnique(current); err != nil {
		t.Fatalf("assigning the current technique reported %v; the reference returns", err)
	}
	// A technique from ANOTHER effect is refused with a parameterless
	// InvalidOperationException: the reference loads no resource string here.
	foreign := testEffect().Techniques().ItemPropertySignatureA5F4623F(0)
	err := effect.SetCurrentTechnique(foreign)
	if !errors.Is(err, errSpriteInvalidOperation) {
		t.Fatalf("%v, want InvalidOperationException for a foreign technique", err)
	}
	if strings.Contains(err.Error(), notCurrentTechnique) {
		t.Fatalf("%v carries a resource string the reference's throw does not load", err)
	}
}

// TestSetCurrentTechniqueRefusesNullWithMicrosoftsSentence pins the null check,
// which uses the TWO-argument ArgumentNullException and therefore does carry
// NullNotAllowed -- unlike DrawString's, which does not.
func TestSetCurrentTechniqueRefusesNullWithMicrosoftsSentence(t *testing.T) {
	err := testEffect().SetCurrentTechnique(nil)
	if err == nil {
		t.Fatal("a nil technique was accepted")
	}
	if !strings.Contains(err.Error(), "value") || !strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("%v, want ArgumentNullException(\"value\", NullNotAllowed)", err)
	}
}

// TestTheCastGuardAdmitsAnArrayParameterWhateverItsClassIs pins the second half
// of every value member's guard:
//
//	if (_paramClass != expected && pElementCollection.Count == 0) throw;
//
// which is what lets a Vector parameter be read through a scalar-array
// overload. A projection that checked only the class would refuse a call the
// reference accepts.
func TestTheCastGuardAdmitsAnArrayParameterWhateverItsClassIs(t *testing.T) {
	matrix := testEffect().Parameters().ItemPropertySignatureD47463F7("World")
	// A Matrix parameter with NO elements is refused through a scalar getter.
	if err := matrix.castGuard(EffectParameterClassScalar); !errors.Is(err, errInvalidCast) {
		t.Fatalf("%v, want InvalidCastException for a Matrix read as a Scalar", err)
	}
	// Give it one element and the same guard admits it.
	matrix.elements = &EffectParameterCollection{parameters: []*EffectParameter{{}}}
	if err := matrix.castGuard(EffectParameterClassScalar); err != nil {
		t.Fatalf("%v; a parameter with elements is admitted whatever its class", err)
	}
	// And its own class is always admitted.
	if err := matrix.castGuard(EffectParameterClassMatrix); err != nil {
		t.Fatalf("%v; a parameter is always readable as its own class", err)
	}
}

// TestTheArrayGettersRefuseANonPositiveCountBeforeTheCastGuard pins
//
//	if (count <= 0) throw new ArgumentOutOfRangeException();
//
// which is the array getters' FIRST statement and names no parameter.
func TestTheArrayGettersRefuseANonPositiveCountBeforeTheCastGuard(t *testing.T) {
	scalar := testEffect().Parameters().ItemPropertySignatureD47463F7("Alpha")
	for _, count := range []int32{0, -1} {
		if _, err := scalar.GetValueSingleArray(count); !errors.Is(err, errArgumentOutOfRange) {
			t.Fatalf("count %d reported %v, want ArgumentOutOfRangeException", count, err)
		}
	}
	// And a count of zero on a parameter of the WRONG class still reports the
	// count, because the count check comes first.
	matrix := testEffect().Parameters().ItemPropertySignatureD47463F7("World")
	if _, err := matrix.GetValueSingleArray(0); !errors.Is(err, errArgumentOutOfRange) {
		t.Fatalf("%v, want the count reported before the cast", err)
	}
}

// TestTheMatrixRoundTripIsPositionalAndRowMajor pins the one place CNA_Matrix's
// element order is stated. A transposed mapping would compile and would produce
// a silently wrong transform.
func TestTheMatrixRoundTripIsPositionalAndRowMajor(t *testing.T) {
	value := framework.NewMatrix(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)
	raw := matrixToRowMajor(value)
	for index := range raw {
		if raw[index] != float32(index+1) {
			t.Fatalf("row-major element %d is %v, want %d", index, raw[index], index+1)
		}
	}
	if matrixFromRowMajor(raw) != value {
		t.Fatal("the matrix did not survive the round trip")
	}
	// The M12/M21 pair is what a transposition would swap.
	if raw[1] != value.M12 || raw[4] != value.M21 {
		t.Fatal("the mapping is transposed")
	}
}

// TestEveryAccessorAnswersTheSameObject pins the identity the whole cluster is
// designed around: the reference's accessors are `ldfld` over collections its
// constructor built, and CNA hands out a FRESH view handle from every call, so
// a projection that asked CNA per call would answer a different object each
// time AND leak a handle each time.
func TestEveryAccessorAnswersTheSameObject(t *testing.T) {
	effect := testEffect()
	if effect.Parameters() != effect.Parameters() || effect.Techniques() != effect.Techniques() ||
		effect.CurrentTechnique() != effect.CurrentTechnique() {
		t.Fatal("an Effect accessor answered two different objects")
	}
	technique := effect.Techniques().ItemPropertySignatureA5F4623F(0)
	if technique.Passes() != technique.Passes() || technique.Annotations() != technique.Annotations() {
		t.Fatal("an EffectTechnique accessor answered two different objects")
	}
	if effect.Techniques().ItemPropertySignatureA5F4623F(0) != technique {
		t.Fatal("the indexer answered two different techniques for one index")
	}
	parameter := effect.Parameters().ItemPropertySignatureC421CB56(0)
	if parameter.Elements() != parameter.Elements() ||
		parameter.StructureMembers() != parameter.StructureMembers() ||
		parameter.Annotations() != parameter.Annotations() {
		t.Fatal("an EffectParameter accessor answered two different objects")
	}
}

// TestTheEnumeratorsWalkEveryElementInOrder pins the four GetEnumerator
// projections, which are the BCL list's own enumerator rather than a
// collection-declared one.
func TestTheEnumeratorsWalkEveryElementInOrder(t *testing.T) {
	effect := testEffect()
	var names []string
	iterator := effect.Techniques().GetEnumerator()
	for {
		technique, ok, err := iterator.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if !ok {
			break
		}
		names = append(names, technique.Name())
	}
	if strings.Join(names, ",") != "Default,Other" {
		t.Fatalf("the enumerator walked %v", names)
	}
	// An empty collection's enumerator ends immediately, and a nil receiver's
	// does too.
	var empty *EffectAnnotationCollection
	if _, ok, err := empty.GetEnumerator().Next(); ok || err != nil {
		t.Fatalf("a nil collection enumerated (%v, %v)", ok, err)
	}
}

// TestTheTextureGettersRefuseRatherThanInventingAnObject pins the one recorded
// divergence in the cluster. The reference returns the SAME Texture2D the
// setter was given; CNA stores a handle and no object identity, so returning a
// fresh facade over that handle would make `p.GetValueTexture2D() == myTexture`
// silently false. The refusal says exactly that.
func TestTheTextureGettersRefuseRatherThanInventingAnObject(t *testing.T) {
	parameter := &EffectParameter{
		parameterClass: EffectParameterClassObject,
		elements:       &EffectParameterCollection{},
	}
	_, err := parameter.GetValueTexture2D()
	if err == nil {
		t.Fatal("a texture getter answered")
	}
	if !errors.Is(err, errEffectTextureIdentity) {
		t.Fatalf("%v, want the recorded identity refusal", err)
	}
	if !strings.Contains(err.Error(), "CNA") {
		t.Fatalf("%v does not attribute the limitation", err)
	}
	// And a parameter of the wrong class reports the CAST first.
	scalar := &EffectParameter{parameterClass: EffectParameterClassScalar, elements: &EffectParameterCollection{}}
	if _, err := scalar.GetValueTexture2D(); !errors.Is(err, errInvalidCast) {
		t.Fatalf("%v, want the cast reported before the identity limitation", err)
	}
}
