package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// Model is Microsoft.Xna.Framework.Graphics.Model:
//
//	.class public auto ansi sealed beforefieldinit Model
//	       extends [mscorlib]System.Object
//
// A loaded model: a bone hierarchy and the meshes hanging from it.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # The constructor is not projected
//
// It is `assembly`, and so is the static factory beside it. A Model reaches a
// consumer only through ContentManager.Load<Model>.
type Model struct {
	root   *ModelBone
	bones  *ModelBoneCollection
	meshes *ModelMeshCollection
	tag    any
}

// The messages the family throws, verified byte for byte against the retained
// assembly.
const (
	modelHasNoEffect = "ModelMeshPart has a null Effect."
	// modelHasNoIEffectMatrices is what Model.Draw throws for a custom effect
	// that cannot be given a world/view/projection. ModelMesh.Draw does NOT
	// throw it -- it applies passes without touching transforms -- which is
	// exactly what the message tells a consumer to do instead.
	modelHasNoIEffectMatrices = "This model contains a custom effect which does not implement the IEffectMatrices interface, so it cannot be drawn using Model.Draw. Instead, call ModelMesh.Draw after setting the appropriate effect parameters."
)

// errModelInvalidOperation projects System.InvalidOperationException.
var errModelInvalidOperation = errors.New("model operation is invalid")

// errModelArgumentOutOfRange projects System.ArgumentOutOfRangeException, which
// all three Copy* members throw for a destination that cannot hold every bone.
var errModelArgumentOutOfRange = errors.New("model argument is out of range")

func modelArgumentOutOfRangeError(parameter string) error {
	return fmt.Errorf("%w: %s", errModelArgumentOutOfRange, parameter)
}

// sharedDrawBoneMatrices is Model::sharedDrawBoneMatrices, and it really is
// what its name says: a `private static Matrix[]` that EVERY Model in the
// process draws through.
//
// Model.Draw grows it when the model it is drawing has more bones than it
// holds, and never shrinks it, so its length is the high-water mark of every
// model drawn so far. It is scratch space, not state: Draw fills it completely
// before reading it.
//
// It is reproduced rather than replaced by a per-model buffer because the
// sharing is observable. Two goroutines drawing two models at once corrupt each
// other's transforms here exactly as two CLR threads do -- Model.Draw is not
// thread-safe in the reference, and a projection that quietly made it safe
// would be describing a different runtime.
var sharedDrawBoneMatrices []framework.Matrix

// Root is Model::get_Root, one `ldfld`.
func (m *Model) Root() *ModelBone { return m.root }

// Bones is Model::get_Bones, one `ldfld`. Its order is the order bone indices
// refer to, and a parent always precedes its children -- which is what makes
// CopyAbsoluteBoneTransformsTo's single forward pass correct.
func (m *Model) Bones() *ModelBoneCollection { return m.bones }

// Meshes is Model::get_Meshes, one `ldfld`.
func (m *Model) Meshes() *ModelMeshCollection { return m.meshes }

// Tag is Model::get_Tag, `System.Object` and therefore `any`.
func (m *Model) Tag() any { return m.tag }

// SetTag is Model::set_Tag, one `stfld` with no validation.
func (m *Model) SetTag(value any) { m.tag = value }

// CopyBoneTransformsTo is Model::CopyBoneTransformsTo(Matrix[]), measured at
// 95 bytes:
//
//	if (destinationBoneTransforms == null)
//	    throw new ArgumentNullException("destinationBoneTransforms");
//	if (destinationBoneTransforms.Length < bones.Count)
//	    throw new ArgumentOutOfRangeException("destinationBoneTransforms");
//	for (int i = 0; i < bones.Count; i++)
//	    destinationBoneTransforms[i] = bones[i].transform;
//
// A destination LONGER than the bone count is accepted, and the extra elements
// are left untouched. The copy reads the bone's private `transform` field
// rather than its getter, which for this type is the same `ldfld`.
func (m *Model) CopyBoneTransformsTo(destinationBoneTransforms []framework.Matrix) error {
	count, err := m.checkBoneDestination(destinationBoneTransforms, "destinationBoneTransforms")
	if err != nil {
		return err
	}
	for index := int32(0); index < count; index++ {
		bone, err := m.bones.ItemPropertySignature854B41ED(index)
		if err != nil {
			return err
		}
		destinationBoneTransforms[index] = bone.transform
	}
	return nil
}

// CopyBoneTransformsFrom is Model::CopyBoneTransformsFrom(Matrix[]), the same
// 95-byte shape in reverse:
//
//	guards on "sourceBoneTransforms"
//	for (int i = 0; i < bones.Count; i++)
//	    bones[i].transform = sourceBoneTransforms[i];
//
// It writes the bone's private FIELD, not through set_Transform. For this type
// the two are the same single `stfld`, so nothing is bypassed -- but the
// projection writes the field for the same reason it reads it above: to match
// what the reference reaches.
func (m *Model) CopyBoneTransformsFrom(sourceBoneTransforms []framework.Matrix) error {
	count, err := m.checkBoneDestination(sourceBoneTransforms, "sourceBoneTransforms")
	if err != nil {
		return err
	}
	for index := int32(0); index < count; index++ {
		bone, err := m.bones.ItemPropertySignature854B41ED(index)
		if err != nil {
			return err
		}
		bone.transform = sourceBoneTransforms[index]
	}
	return nil
}

// CopyAbsoluteBoneTransformsTo is Model::CopyAbsoluteBoneTransformsTo(Matrix[]),
// measured at 154 bytes:
//
//	the same two guards, then
//	for (int i = 0; i < bones.Count; i++) {
//	    ModelBone bone = bones[i];
//	    if (bone.Parent == null)
//	        destination[i] = bone.transform;
//	    else
//	        destination[i] = bone.transform * destination[bone.Parent.Index];
//	}
//
// This is a SINGLE FORWARD PASS that reads its own output: a bone's absolute
// transform is built from its parent's absolute transform, which must therefore
// already have been written. That is correct only because the content pipeline
// orders Bones so a parent always precedes its children, and it is the reason
// this member exists at all -- a ModelBone cannot compute it alone.
//
// The multiplication order is `bone.transform * destination[parentIndex]`, not
// the reverse. Matrix multiplication does not commute, so the order is the
// behaviour.
func (m *Model) CopyAbsoluteBoneTransformsTo(destinationBoneTransforms []framework.Matrix) error {
	count, err := m.checkBoneDestination(destinationBoneTransforms, "destinationBoneTransforms")
	if err != nil {
		return err
	}
	for index := int32(0); index < count; index++ {
		bone, err := m.bones.ItemPropertySignature854B41ED(index)
		if err != nil {
			return err
		}
		parent := bone.Parent()
		if parent == nil {
			destinationBoneTransforms[index] = bone.transform
			continue
		}
		parentIndex := parent.Index()
		if parentIndex < 0 || int(parentIndex) >= len(destinationBoneTransforms) {
			return modelArgumentOutOfRangeError("destinationBoneTransforms")
		}
		destinationBoneTransforms[index] = framework.MatrixMultiplyByMatrixAndMatrix(
			bone.transform, destinationBoneTransforms[parentIndex])
	}
	return nil
}

// checkBoneDestination is the two-guard prologue all three Copy* members share,
// in the reference's order: null first, then too-small.
func (m *Model) checkBoneDestination(transforms []framework.Matrix, parameter string) (int32, error) {
	if transforms == nil {
		return 0, modelArgumentNullError(parameter)
	}
	if m.bones == nil {
		return 0, nil
	}
	count := m.bones.Count()
	if int32(len(transforms)) < count {
		return 0, modelArgumentOutOfRangeError(parameter)
	}
	return count, nil
}

// Draw is Model::Draw(Matrix, Matrix, Matrix), measured at 236 bytes:
//
//	Matrix[] shared = Model.sharedDrawBoneMatrices;
//	if (shared == null || shared.Length < bones.Count)
//	    shared = Model.sharedDrawBoneMatrices = new Matrix[bones.Count];
//	CopyAbsoluteBoneTransformsTo(shared);
//	foreach mesh in Meshes:
//	    int parentIndex = mesh.ParentBone.Index;
//	    foreach effect in mesh.Effects:
//	        if (effect == null)
//	            throw new InvalidOperationException(ModelHasNoEffect);
//	        IEffectMatrices matrices = effect as IEffectMatrices;
//	        if (matrices == null)
//	            throw new InvalidOperationException(ModelHasNoIEffectMatrices);
//	        matrices.World      = shared[parentIndex] * world;
//	        matrices.View       = view;
//	        matrices.Projection = projection;
//	    mesh.Draw();
//
// The transforms are set once per EFFECT, not once per part, which is what the
// Effects collection's reference counting exists to make possible.
//
// `shared[parentIndex] * world` is the order the reference emits, and it means
// the bone transform is applied before the world transform.
//
// A custom effect that does not implement IEffectMatrices is refused with its
// own message, which tells the consumer to call ModelMesh.Draw instead -- and
// that really is a way out, because ModelMesh.Draw sets no transforms.
func (m *Model) Draw(world, view, projection framework.Matrix) error {
	if m.meshes == nil || m.bones == nil {
		return nil
	}
	boneCount := m.bones.Count()
	if sharedDrawBoneMatrices == nil || int32(len(sharedDrawBoneMatrices)) < boneCount {
		sharedDrawBoneMatrices = make([]framework.Matrix, boneCount)
	}
	if err := m.CopyAbsoluteBoneTransformsTo(sharedDrawBoneMatrices); err != nil {
		return err
	}
	for _, mesh := range m.meshes.wrappedArray {
		parentBone := mesh.ParentBone()
		if parentBone == nil {
			return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
		}
		parentIndex := parentBone.Index()
		if parentIndex < 0 || int(parentIndex) >= len(sharedDrawBoneMatrices) {
			return modelArgumentOutOfRangeError("sharedDrawBoneMatrices")
		}
		absolute := sharedDrawBoneMatrices[parentIndex]
		for _, effect := range mesh.effects.wrappedList {
			if effect == nil {
				return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
			}
			matrices, ok := any(effect).(IEffectMatrices)
			if !ok {
				return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoIEffectMatrices)
			}
			matrices.SetWorld(framework.MatrixMultiplyByMatrixAndMatrix(absolute, world))
			matrices.SetView(view)
			matrices.SetProjection(projection)
		}
		if err := mesh.Draw(); err != nil {
			return err
		}
	}
	return nil
}
