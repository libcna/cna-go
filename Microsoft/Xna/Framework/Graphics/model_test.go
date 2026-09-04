package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// newTestModel builds the skeleton the reference's content reader would: bones
// in an order where a parent always precedes its children, which is what makes
// CopyAbsoluteBoneTransformsTo's single forward pass correct.
//
//	0 root          identity
//	1   child       translate(1,0,0)
//	2     grandchild translate(0,1,0)
func newTestModel() (*Model, []*ModelBone) {
	// The root's transform is deliberately NOT the identity. With an identity
	// root every bone's parent has the same local and absolute transform, and a
	// projection that read the parent's LOCAL matrix would pass every test.
	root := &ModelBone{name: "root", index: 0,
		transform: framework.MatrixCreateTranslationBySingleAndSingleAndSingle(10, 0, 0)}
	child := &ModelBone{name: "child", index: 1,
		transform: framework.MatrixCreateTranslationBySingleAndSingleAndSingle(1, 0, 0), parent: root}
	grandchild := &ModelBone{name: "grandchild", index: 2,
		transform: framework.MatrixCreateTranslationBySingleAndSingleAndSingle(0, 1, 0), parent: child}
	bones := []*ModelBone{root, child, grandchild}
	root.children = newModelBoneCollection([]*ModelBone{child})
	child.children = newModelBoneCollection([]*ModelBone{grandchild})
	grandchild.children = newModelBoneCollection(nil)

	mesh := &ModelMesh{name: "mesh", parentBone: child}
	mesh.meshParts = newModelMeshPartCollection(nil)
	mesh.effects = newModelEffectCollection(nil)

	model := &Model{
		root:   root,
		bones:  newModelBoneCollection(bones),
		meshes: newModelMeshCollection([]*ModelMesh{mesh}),
	}
	return model, bones
}

// TestModelBoneIsPureFieldAccess pins that every ModelBone member is one field
// access and that the setter validates nothing.
func TestModelBoneIsPureFieldAccess(t *testing.T) {
	_, bones := newTestModel()
	root, child := bones[0], bones[1]
	if root.Name() != "root" || child.Name() != "child" {
		t.Fatalf("names = %q, %q", root.Name(), child.Name())
	}
	if root.Index() != 0 || child.Index() != 1 {
		t.Fatalf("indices = %d, %d", root.Index(), child.Index())
	}
	if root.Parent() != nil {
		t.Fatal("the root bone reported a parent; the parent walk terminates there")
	}
	if child.Parent() != root {
		t.Fatal("the child did not report the root as its parent")
	}
	// set_Transform is one stfld: a zero matrix is stored unchanged.
	var zero framework.Matrix
	child.SetTransform(zero)
	if child.Transform() != zero {
		t.Fatal("the setter did not store a zero matrix; it validates nothing")
	}
}

// TestModelBoneCollectionInheritedSurface exercises the five members the
// composed ReadOnlyCollection base contributes.
func TestModelBoneCollectionInheritedSurface(t *testing.T) {
	model, bones := newTestModel()
	all := model.Bones()
	if got := all.Count(); got != 3 {
		t.Fatalf("Count = %d, want 3", got)
	}
	for index, want := range bones {
		got, err := all.ItemPropertySignature854B41ED(int32(index))
		if err != nil {
			t.Fatalf("Item(%d): %v", index, err)
		}
		if got != want {
			t.Fatalf("Item(%d) returned the wrong bone", index)
		}
	}
	if _, err := all.ItemPropertySignature854B41ED(3); err == nil {
		t.Fatal("Item accepted an index past the end")
	}
	if _, err := all.ItemPropertySignature854B41ED(-1); err == nil {
		t.Fatal("Item accepted a negative index")
	}
	if !all.Contains(bones[2]) {
		t.Fatal("Contains missed a bone the collection holds")
	}
	if all.Contains(&ModelBone{}) {
		t.Fatal("Contains matched a bone the collection does not hold; the comparer is identity")
	}
	if got := all.IndexOf(bones[1]); got != 1 {
		t.Fatalf("IndexOf = %d, want 1", got)
	}
	if got := all.IndexOf(&ModelBone{}); got != -1 {
		t.Fatalf("IndexOf of an absent bone = %d, want -1", got)
	}
	// CopyTo carries the three failures Array.Copy has, and accepts a
	// destination LONGER than the collection.
	destination := make([]*ModelBone, 5)
	if err := all.CopyTo(destination, 1); err != nil {
		t.Fatalf("CopyTo into a longer array: %v", err)
	}
	if destination[1] != bones[0] || destination[3] != bones[2] || destination[0] != nil {
		t.Fatal("CopyTo wrote the wrong slots")
	}
	if err := all.CopyTo(nil, 0); err == nil {
		t.Fatal("CopyTo accepted a nil destination")
	}
	if err := all.CopyTo(destination, -1); err == nil {
		t.Fatal("CopyTo accepted a negative index")
	}
	if err := all.CopyTo(make([]*ModelBone, 2), 0); err == nil {
		t.Fatal("CopyTo accepted a destination too small")
	}
}

// TestModelBoneCollectionByNameLookup pins TryGetValue's three measured
// details: the empty-name refusal, its exception KIND, and the ORDINAL
// comparison.
func TestModelBoneCollectionByNameLookup(t *testing.T) {
	model, bones := newTestModel()
	all := model.Bones()

	found, bone, err := all.TryGetValue("child")
	if err != nil || !found || bone != bones[1] {
		t.Fatalf("TryGetValue(\"child\") = %v, %v, %v", found, bone, err)
	}
	// Ordinal means CASE SENSITIVE. `ldc.i4.4` is StringComparison.Ordinal,
	// not OrdinalIgnoreCase.
	found, _, err = all.TryGetValue("CHILD")
	if err != nil {
		t.Fatalf("TryGetValue(\"CHILD\"): %v", err)
	}
	if found {
		t.Fatal("the name lookup was case-insensitive; the reference compares Ordinal")
	}
	// A miss is not an error.
	found, bone, err = all.TryGetValue("absent")
	if err != nil || found || bone != nil {
		t.Fatalf("a miss answered %v, %v, %v", found, bone, err)
	}
	// The EMPTY name is refused, and refused as ArgumentNullException.
	if _, _, err := all.TryGetValue(""); !errors.Is(err, errModelArgumentNull) {
		t.Fatalf("TryGetValue(\"\") = %v, want the argument-null refusal", err)
	}

	// The by-name indexer forwards, and turns a miss into KeyNotFoundException.
	got, err := all.ItemPropertySignatureC23A10DE("grandchild")
	if err != nil || got != bones[2] {
		t.Fatalf("indexer: %v, %v", got, err)
	}
	if _, err := all.ItemPropertySignatureC23A10DE("absent"); !errors.Is(err, errModelKeyNotFound) {
		t.Fatalf("the indexer answered %v for a missing name, want key-not-found", err)
	}
	if _, err := all.ItemPropertySignatureC23A10DE(""); !errors.Is(err, errModelArgumentNull) {
		t.Fatal("the indexer did not reach TryGetValue's own guard first")
	}
}

// TestModelBoneCollectionEnumerator walks the derived enumerator, which is the
// member that HIDES the inherited one.
func TestModelBoneCollectionEnumerator(t *testing.T) {
	model, bones := newTestModel()
	enumerator := model.Bones().GetEnumerator()
	// Current before the first step answers nothing.
	if enumerator.Current() != nil {
		t.Fatal("Current answered before the first MoveNext")
	}
	seen := 0
	for enumerator.MoveNext() {
		if enumerator.Current() != bones[seen] {
			t.Fatalf("step %d returned the wrong bone", seen)
		}
		seen++
	}
	if seen != len(bones) {
		t.Fatalf("enumerated %d bones, want %d", seen, len(bones))
	}
	// Past the end it stays past the end.
	if enumerator.MoveNext() {
		t.Fatal("MoveNext advanced past the end")
	}
	if enumerator.Current() != nil {
		t.Fatal("Current answered past the end")
	}
	enumerator.Dispose()
	// An empty collection enumerates zero times.
	empty := newModelBoneCollection(nil).GetEnumerator()
	if empty.MoveNext() {
		t.Fatal("an empty collection produced a step")
	}
}

// TestCopyBoneTransformsRoundTrip covers the two plain Copy* members and the
// two guards all three share.
func TestCopyBoneTransformsRoundTrip(t *testing.T) {
	model, bones := newTestModel()
	transforms := make([]framework.Matrix, 3)
	if err := model.CopyBoneTransformsTo(transforms); err != nil {
		t.Fatalf("CopyBoneTransformsTo: %v", err)
	}
	for index, bone := range bones {
		if transforms[index] != bone.Transform() {
			t.Fatalf("bone %d was copied wrong", index)
		}
	}
	// A LONGER destination is accepted and its extra slots are left alone.
	longer := make([]framework.Matrix, 5)
	sentinel := framework.MatrixCreateTranslationBySingleAndSingleAndSingle(9, 9, 9)
	longer[4] = sentinel
	if err := model.CopyBoneTransformsTo(longer); err != nil {
		t.Fatalf("CopyBoneTransformsTo into a longer array: %v", err)
	}
	if longer[4] != sentinel {
		t.Fatal("the copy wrote past the bone count")
	}

	// Write different transforms back and read them again.
	replacement := []framework.Matrix{
		framework.MatrixCreateTranslationBySingleAndSingleAndSingle(5, 0, 0),
		framework.MatrixCreateTranslationBySingleAndSingleAndSingle(0, 5, 0),
		framework.MatrixCreateTranslationBySingleAndSingleAndSingle(0, 0, 5),
	}
	if err := model.CopyBoneTransformsFrom(replacement); err != nil {
		t.Fatalf("CopyBoneTransformsFrom: %v", err)
	}
	for index, bone := range bones {
		if bone.Transform() != replacement[index] {
			t.Fatalf("bone %d did not take the written transform", index)
		}
	}

	// The two guards, in the reference's order.
	for name, call := range map[string]func([]framework.Matrix) error{
		"CopyBoneTransformsTo":         model.CopyBoneTransformsTo,
		"CopyBoneTransformsFrom":       model.CopyBoneTransformsFrom,
		"CopyAbsoluteBoneTransformsTo": model.CopyAbsoluteBoneTransformsTo,
	} {
		if err := call(nil); !errors.Is(err, errModelArgumentNull) {
			t.Fatalf("%s accepted nil: %v", name, err)
		}
		if err := call(make([]framework.Matrix, 2)); !errors.Is(err, errModelArgumentOutOfRange) {
			t.Fatalf("%s accepted a destination shorter than the bone count: %v", name, err)
		}
	}
}

// TestCopyAbsoluteBoneTransformsAccumulatesDownTheChain is the test that fails
// if the parent walk, its multiplication ORDER, or its reliance on already
// written output is wrong.
func TestCopyAbsoluteBoneTransformsAccumulatesDownTheChain(t *testing.T) {
	model, bones := newTestModel()
	absolute := make([]framework.Matrix, 3)
	if err := model.CopyAbsoluteBoneTransformsTo(absolute); err != nil {
		t.Fatalf("CopyAbsoluteBoneTransformsTo: %v", err)
	}
	// The root has no parent, so its absolute transform is its local one.
	if absolute[0] != bones[0].Transform() {
		t.Fatal("the root's absolute transform is not its local transform")
	}
	// A child is local * parent's ABSOLUTE, in that order.
	wantChild := framework.MatrixMultiplyByMatrixAndMatrix(bones[1].Transform(), absolute[0])
	if absolute[1] != wantChild {
		t.Fatalf("child absolute = %v, want local*parent = %v", absolute[1], wantChild)
	}
	wantGrandchild := framework.MatrixMultiplyByMatrixAndMatrix(bones[2].Transform(), absolute[1])
	if absolute[2] != wantGrandchild {
		t.Fatal("the grandchild did not accumulate through the child")
	}
	// The accumulation is real and reaches all the way to the root: 10 from the
	// root, 1 from the child, 1 from the grandchild.
	if absolute[2].M41 != 11 || absolute[2].M42 != 1 {
		t.Fatalf("grandchild translation = (%v,%v), want (11,1) accumulated through root and child",
			absolute[2].M41, absolute[2].M42)
	}
	// And the parent's ABSOLUTE is what the grandchild multiplies by, not its
	// local: the two differ here precisely because the root is not identity.
	if absolute[1] == bones[1].Transform() {
		t.Fatal("the fixture's child has the same local and absolute transform, so this test is blind")
	}
}

// TestCopyAbsoluteBoneTransformsOrderIsLoadBearing proves the multiplication
// ORDER rather than assuming it. Two translations commute, so the chain above
// cannot tell `local * parent` from `parent * local`; a ROTATION and a
// translation do not.
func TestCopyAbsoluteBoneTransformsOrderIsLoadBearing(t *testing.T) {
	root := &ModelBone{name: "root", index: 0,
		transform: framework.MatrixCreateRotationZBySingle(1.5707963)}
	child := &ModelBone{name: "child", index: 1,
		transform: framework.MatrixCreateTranslationBySingleAndSingleAndSingle(1, 0, 0), parent: root}
	model := &Model{
		root:   root,
		bones:  newModelBoneCollection([]*ModelBone{root, child}),
		meshes: newModelMeshCollection(nil),
	}
	absolute := make([]framework.Matrix, 2)
	if err := model.CopyAbsoluteBoneTransformsTo(absolute); err != nil {
		t.Fatalf("CopyAbsoluteBoneTransformsTo: %v", err)
	}
	want := framework.MatrixMultiplyByMatrixAndMatrix(child.Transform(), absolute[0])
	reversed := framework.MatrixMultiplyByMatrixAndMatrix(absolute[0], child.Transform())
	if want == reversed {
		t.Fatal("the fixture's transforms commute, so this test cannot see the order")
	}
	if absolute[1] != want {
		t.Fatalf("child absolute = %v; the reference emits local*parent, not parent*local", absolute[1])
	}
}

// newTestMesh builds a mesh with the given number of parts, all sharing no
// effect yet, wired so SetEffect can reach its siblings.
func newTestMesh(partCount int) (*ModelMesh, []*ModelMeshPart) {
	parts := make([]*ModelMeshPart, partCount)
	mesh := &ModelMesh{name: "mesh"}
	for index := range parts {
		parts[index] = &ModelMeshPart{parent: mesh}
	}
	mesh.meshParts = newModelMeshPartCollection(parts)
	mesh.effects = newModelEffectCollection(nil)
	return mesh, parts
}

// TestSetEffectReferenceCounting is the test the whole family turns on. It
// walks every branch of the 175-byte body.
func TestSetEffectReferenceCounting(t *testing.T) {
	mesh, parts := newTestMesh(3)
	first := &Effect{}
	second := &Effect{}

	// Assigning to one part adds the effect once.
	parts[0].SetEffect(first)
	if got := mesh.Effects().Count(); got != 1 {
		t.Fatalf("Effects.Count = %d after one assignment, want 1", got)
	}
	if mesh.Effects().IndexOf(first) != 0 {
		t.Fatal("the effect was not added")
	}

	// A SECOND part taking the same effect must NOT add it again: the scan
	// finds a sibling already using it and suppresses the Add.
	parts[1].SetEffect(first)
	if got := mesh.Effects().Count(); got != 1 {
		t.Fatalf("Effects.Count = %d after a second part took the same effect, want 1", got)
	}

	// Moving one of the two parts off it must NOT remove it: the other part
	// still uses it, so the Remove is suppressed and the Add happens.
	parts[0].SetEffect(second)
	if got := mesh.Effects().Count(); got != 2 {
		t.Fatalf("Effects.Count = %d, want 2: the old effect is still in use", got)
	}
	if mesh.Effects().IndexOf(first) < 0 {
		t.Fatal("the old effect was removed while a sibling still used it")
	}

	// Moving the LAST user off it removes it.
	parts[1].SetEffect(second)
	if got := mesh.Effects().Count(); got != 1 {
		t.Fatalf("Effects.Count = %d after the last user left, want 1", got)
	}
	if mesh.Effects().IndexOf(first) >= 0 {
		t.Fatal("the old effect stayed after its last user left")
	}

	// Assigning the SAME effect again is the early return: no scan, no churn.
	parts[0].SetEffect(second)
	if got := mesh.Effects().Count(); got != 1 {
		t.Fatalf("Effects.Count = %d after a redundant assignment, want 1", got)
	}

	// Assigning nil removes the effect from the last user without adding
	// anything, because the Add is guarded on a non-nil value.
	parts[0].SetEffect(nil)
	parts[1].SetEffect(nil)
	if got := mesh.Effects().Count(); got != 0 {
		t.Fatalf("Effects.Count = %d after every part dropped its effect, want 0", got)
	}
	if parts[0].Effect() != nil {
		t.Fatal("the part kept an effect after being assigned nil")
	}
}

// TestSetEffectComparesIdentityNotEquality pins that the scan uses
// ReferenceEquals. Two distinct Effect objects are never the same effect, no
// matter what they hold.
func TestSetEffectComparesIdentityNotEquality(t *testing.T) {
	mesh, parts := newTestMesh(2)
	first := &Effect{}
	second := &Effect{}
	parts[0].SetEffect(first)
	parts[1].SetEffect(second)
	if got := mesh.Effects().Count(); got != 2 {
		t.Fatalf("two distinct effects produced Count = %d, want 2", got)
	}
	if mesh.Effects().IndexOf(first) == mesh.Effects().IndexOf(second) {
		t.Fatal("two distinct effects were treated as one")
	}
}

// TestModelEffectCollectionIsLive pins the difference that made this collection
// need its own constructor: the view is over a LIST the owner keeps changing,
// so a consumer holding it sees every addition.
func TestModelEffectCollectionIsLive(t *testing.T) {
	mesh, parts := newTestMesh(2)
	effects := mesh.Effects()
	if got := effects.Count(); got != 0 {
		t.Fatalf("a fresh collection reported Count = %d", got)
	}
	parts[0].SetEffect(&Effect{})
	// The SAME collection object must now report one, without being re-fetched.
	if got := effects.Count(); got != 1 {
		t.Fatalf("the view did not see the addition: Count = %d, want 1", got)
	}
	item, err := effects.Item(0)
	if err != nil || item == nil {
		t.Fatalf("the view could not read the added effect: %v, %v", item, err)
	}
	parts[0].SetEffect(nil)
	if got := effects.Count(); got != 0 {
		t.Fatalf("the view did not see the removal: Count = %d", got)
	}
}

// TestModelEffectCollectionEnumeratorIsVersionChecked pins the second
// consequence of the List backing: unlike its three array-backed siblings, this
// enumerator fails once the list has changed.
func TestModelEffectCollectionEnumeratorIsVersionChecked(t *testing.T) {
	mesh, parts := newTestMesh(2)
	parts[0].SetEffect(&Effect{})

	enumerator := mesh.Effects().GetEnumerator()
	ok, err := enumerator.MoveNext()
	if err != nil || !ok {
		t.Fatalf("first step: %v, %v", ok, err)
	}
	// Mutate the list under the enumerator.
	parts[1].SetEffect(&Effect{})
	if _, err := enumerator.MoveNext(); !errors.Is(err, errModelEnumerationFailed) {
		t.Fatalf("MoveNext after a mutation = %v, want the enumeration-failed refusal", err)
	}

	// An UNDISTURBED enumeration walks every element and then stops.
	fresh := mesh.Effects().GetEnumerator()
	seen := 0
	for {
		ok, err := fresh.MoveNext()
		if err != nil {
			t.Fatalf("undisturbed enumeration failed: %v", err)
		}
		if !ok {
			break
		}
		if fresh.Current() == nil {
			t.Fatal("Current answered nil mid-enumeration")
		}
		seen++
	}
	if seen != 2 {
		t.Fatalf("enumerated %d effects, want 2", seen)
	}
	fresh.Dispose()
}

// TestArrayBackedEnumeratorsHaveNoVersionCheck is the other half of the same
// finding: the three array-backed collections cannot detect a change, because
// the reference's enumerator over an array has no version field.
func TestArrayBackedEnumeratorsHaveNoVersionCheck(t *testing.T) {
	// The signature itself is the evidence: MoveNext on these three returns a
	// bool alone, with no failure to report, while ModelEffectCollection's
	// returns a bool and an error.
	var _ func(*ModelBoneCollectionEnumerator) bool = (*ModelBoneCollectionEnumerator).MoveNext
	var _ func(*ModelMeshCollectionEnumerator) bool = (*ModelMeshCollectionEnumerator).MoveNext
	var _ func(*ModelMeshPartCollectionEnumerator) bool = (*ModelMeshPartCollectionEnumerator).MoveNext
	var _ func(*ModelEffectCollectionEnumerator) (bool, error) = (*ModelEffectCollectionEnumerator).MoveNext
}

// TestModelMeshDrawRefusesAPartWithNoEffect pins the per-part null check, which
// is the only branch of ModelMesh.Draw reachable without a device.
func TestModelMeshDrawRefusesAPartWithNoEffect(t *testing.T) {
	mesh, _ := newTestMesh(1)
	if err := mesh.Draw(); !errors.Is(err, errModelInvalidOperation) {
		t.Fatalf("Draw with a null-effect part = %v, want the invalid-operation refusal", err)
	}
	if err := mesh.Draw(); err == nil || !contains(err.Error(), modelHasNoEffect) {
		t.Fatalf("the refusal carried %v, want the ModelHasNoEffect message", err)
	}
	// A mesh with NO parts draws nothing and refuses nothing.
	empty := &ModelMesh{meshParts: newModelMeshPartCollection(nil),
		effects: newModelEffectCollection(nil)}
	if err := empty.Draw(); err != nil {
		t.Fatalf("an empty mesh refused: %v", err)
	}
}

// TestModelDrawGrowsTheSharedBoneArray pins the static scratch buffer, which is
// the reference's own design and is observable: it is never shrunk, so its
// length is the high-water mark of every model drawn in the process.
func TestModelDrawGrowsTheSharedBoneArray(t *testing.T) {
	sharedDrawBoneMatrices = nil
	small, _ := newTestModel() // three bones
	// Draw refuses at the first mesh because its parts have no effect, but the
	// bone array is filled BEFORE any mesh is reached, which is what this test
	// watches.
	_ = small.Draw(framework.MatrixIdentity(), framework.MatrixIdentity(), framework.MatrixIdentity())
	if len(sharedDrawBoneMatrices) < 3 {
		t.Fatalf("the shared array holds %d matrices after a three-bone model, want at least 3",
			len(sharedDrawBoneMatrices))
	}
	// Sizing it is not enough: Draw must FILL it, and with the ABSOLUTE
	// transforms. A Draw that skipped the fill would leave zero matrices here.
	want := make([]framework.Matrix, 3)
	if err := small.CopyAbsoluteBoneTransformsTo(want); err != nil {
		t.Fatalf("CopyAbsoluteBoneTransformsTo: %v", err)
	}
	for index := range want {
		if sharedDrawBoneMatrices[index] != want[index] {
			t.Fatalf("shared bone %d = %v, want the absolute transform %v",
				index, sharedDrawBoneMatrices[index], want[index])
		}
	}
	grown := len(sharedDrawBoneMatrices)

	// A SMALLER model must not shrink it.
	tiny := &Model{
		root:   &ModelBone{name: "only", index: 0, transform: framework.MatrixIdentity()},
		meshes: newModelMeshCollection(nil),
	}
	tiny.bones = newModelBoneCollection([]*ModelBone{tiny.root})
	if err := tiny.Draw(framework.MatrixIdentity(), framework.MatrixIdentity(),
		framework.MatrixIdentity()); err != nil {
		t.Fatalf("drawing a one-bone model with no meshes: %v", err)
	}
	if len(sharedDrawBoneMatrices) != grown {
		t.Fatalf("the shared array changed length to %d for a smaller model; the reference never shrinks it",
			len(sharedDrawBoneMatrices))
	}
	sharedDrawBoneMatrices = nil
}

// TestModelDrawRefusesAnEffectWithoutIEffectMatrices pins the second of
// Model.Draw's two refusals, and the message that names the way out.
func TestModelDrawRefusesAnEffectWithoutIEffectMatrices(t *testing.T) {
	sharedDrawBoneMatrices = nil
	bone := &ModelBone{name: "root", index: 0, transform: framework.MatrixIdentity()}
	mesh, parts := newTestMesh(1)
	mesh.parentBone = bone
	// A bare Effect implements no IEffectMatrices; the stock effects do.
	parts[0].SetEffect(&Effect{})
	model := &Model{
		root:   bone,
		bones:  newModelBoneCollection([]*ModelBone{bone}),
		meshes: newModelMeshCollection([]*ModelMesh{mesh}),
	}
	err := model.Draw(framework.MatrixIdentity(), framework.MatrixIdentity(), framework.MatrixIdentity())
	if !errors.Is(err, errModelInvalidOperation) {
		t.Fatalf("Draw = %v, want the invalid-operation refusal", err)
	}
	if !contains(err.Error(), modelHasNoIEffectMatrices) {
		t.Fatalf("the refusal carried %q, want the IEffectMatrices message", err.Error())
	}
	sharedDrawBoneMatrices = nil
}

func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestSetEffectFlagsAreExclusive covers the branch two parts cannot reach: the
// scan's `else if` means one sibling sets at most ONE flag. It takes three
// parts to see it -- one on the old effect, one already on the new one.
func TestSetEffectFlagsAreExclusive(t *testing.T) {
	mesh, parts := newTestMesh(3)
	old := &Effect{}
	next := &Effect{}
	parts[0].SetEffect(old)  // the part that will move
	parts[1].SetEffect(old)  // a sibling still on the old effect
	parts[2].SetEffect(next) // a sibling already on the new one
	if got := mesh.Effects().Count(); got != 2 {
		t.Fatalf("setup produced Count = %d, want 2", got)
	}

	// Moving part 0 from old to next must do NEITHER: the Remove is suppressed
	// because part 1 still uses old, and the Add is suppressed because part 2
	// already uses next. A scan whose flags were not exclusive would get one of
	// them wrong.
	parts[0].SetEffect(next)
	if got := mesh.Effects().Count(); got != 2 {
		t.Fatalf("Count = %d after a move between two in-use effects, want 2", got)
	}
	if mesh.Effects().IndexOf(old) < 0 {
		t.Fatal("the old effect was removed while part 1 still used it")
	}
	if mesh.Effects().IndexOf(next) < 0 {
		t.Fatal("the new effect went missing")
	}
	// And it must appear exactly ONCE, not twice.
	seen := 0
	enumerator := mesh.Effects().GetEnumerator()
	for {
		ok, err := enumerator.MoveNext()
		if err != nil {
			t.Fatalf("enumeration: %v", err)
		}
		if !ok {
			break
		}
		if enumerator.Current() == EffectReference(next) {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("the new effect appears %d times, want 1", seen)
	}
}

// TestSetEffectNeverAddsNil pins the Add's non-nil guard. A part dropping its
// effect must not put a nil into the mesh's Effects, which ModelMesh.Draw
// would then refuse on.
func TestSetEffectNeverAddsNil(t *testing.T) {
	mesh, parts := newTestMesh(2)
	parts[0].SetEffect(&Effect{})
	parts[0].SetEffect(nil)
	if got := mesh.Effects().Count(); got != 0 {
		t.Fatalf("Count = %d after dropping the only effect, want 0", got)
	}
	// A part that never had an effect assigning nil is the early return, and
	// must not add one either.
	parts[1].SetEffect(nil)
	if got := mesh.Effects().Count(); got != 0 {
		t.Fatalf("Count = %d after a no-op nil assignment, want 0", got)
	}

	// The case above cannot see a missing `value != nil` guard, and the reason
	// is faithful rather than accidental: the scan finds the OTHER part also
	// holding nil, so ReferenceEquals(other, value) is true for it and the Add
	// is suppressed anyway -- which is what the reference does too.
	//
	// To see the guard, every other part must hold a NON-nil effect, so nothing
	// suppresses the Add and only the guard itself can.
	guarded, guardedParts := newTestMesh(2)
	guardedParts[0].SetEffect(&Effect{})
	guardedParts[1].SetEffect(&Effect{})
	if got := guarded.Effects().Count(); got != 2 {
		t.Fatalf("setup produced Count = %d, want 2", got)
	}
	guardedParts[0].SetEffect(nil)
	if got := guarded.Effects().Count(); got != 1 {
		t.Fatalf("Count = %d after one part dropped its effect, want 1: a nil was added", got)
	}
	item, err := guarded.Effects().Item(0)
	if err != nil {
		t.Fatalf("Item(0): %v", err)
	}
	if item == nil {
		t.Fatal("a nil effect reached the collection")
	}
	enumerator := mesh.Effects().GetEnumerator()
	for {
		ok, err := enumerator.MoveNext()
		if err != nil {
			t.Fatalf("enumeration: %v", err)
		}
		if !ok {
			break
		}
		if enumerator.Current() == nil {
			t.Fatal("a nil effect reached the mesh's Effects collection")
		}
	}
}

// TestModelEffectCollectionRemoveDropsExactlyOne pins List<T>.Remove: the FIRST
// match by identity, and nothing else. It reaches the unexported mutators
// directly because no public member can add the same effect twice.
func TestModelEffectCollectionRemoveDropsExactlyOne(t *testing.T) {
	first := &Effect{}
	second := &Effect{}
	collection := newModelEffectCollection(nil)
	collection.add(first)
	collection.add(second)
	collection.add(first) // a duplicate, which the public path never creates
	if got := collection.Count(); got != 3 {
		t.Fatalf("Count = %d, want 3", got)
	}
	collection.remove(first)
	if got := collection.Count(); got != 2 {
		t.Fatalf("Count = %d after removing one duplicate, want 2", got)
	}
	// The SECOND element must now be the survivor of the pair, in order.
	item, err := collection.Item(0)
	if err != nil || item != EffectReference(second) {
		t.Fatalf("element 0 = %v, want the second effect: Remove drops the FIRST match", item)
	}
	item, err = collection.Item(1)
	if err != nil || item != EffectReference(first) {
		t.Fatalf("element 1 = %v, want the surviving duplicate", item)
	}
	// Removing something absent leaves the list alone.
	collection.remove(&Effect{})
	if got := collection.Count(); got != 2 {
		t.Fatalf("Count = %d after removing an absent effect, want 2", got)
	}
}

// TestModelBoneCollectionTryGetValueAnswersTheFirstMatch pins that the scan
// stops at the first name match rather than running on.
func TestModelBoneCollectionTryGetValueAnswersTheFirstMatch(t *testing.T) {
	first := &ModelBone{name: "same", index: 0}
	second := &ModelBone{name: "same", index: 1}
	collection := newModelBoneCollection([]*ModelBone{first, second})
	found, bone, err := collection.TryGetValue("same")
	if err != nil || !found {
		t.Fatalf("TryGetValue: %v, %v", found, err)
	}
	if bone != first {
		t.Fatal("TryGetValue answered the last match; the reference returns at the first")
	}
}
