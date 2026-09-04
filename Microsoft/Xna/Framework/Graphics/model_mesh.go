package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// ModelMesh is Microsoft.Xna.Framework.Graphics.ModelMesh:
//
//	.class public auto ansi sealed beforefieldinit ModelMesh
//	       extends [mscorlib]System.Object
//
// One drawable piece of a model: a bone it hangs from, a bounding sphere, the
// parts it is built from and the distinct effects those parts use.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
type ModelMesh struct {
	name           string
	parentBone     *ModelBone
	boundingSphere framework.BoundingSphere
	tag            any
	meshParts      *ModelMeshPartCollection
	effects        *ModelEffectCollection
}

// Name is ModelMesh::get_Name, one `ldfld`.
func (m *ModelMesh) Name() string { return m.name }

// ParentBone is ModelMesh::get_ParentBone, one `ldfld`. Its Index is what
// Model.Draw uses to pick this mesh's absolute transform out of the shared
// bone-matrix array.
func (m *ModelMesh) ParentBone() *ModelBone { return m.parentBone }

// BoundingSphere is ModelMesh::get_BoundingSphere, one `ldfld`. It is in the
// mesh's own space, so a consumer transforms it by the absolute bone transform
// before culling.
func (m *ModelMesh) BoundingSphere() framework.BoundingSphere { return m.boundingSphere }

// Tag is ModelMesh::get_Tag, `System.Object` and therefore `any`.
func (m *ModelMesh) Tag() any { return m.tag }

// SetTag is ModelMesh::set_Tag, one `stfld` with no validation.
func (m *ModelMesh) SetTag(value any) { m.tag = value }

// MeshParts is ModelMesh::get_MeshParts, one `ldfld`.
func (m *ModelMesh) MeshParts() *ModelMeshPartCollection { return m.meshParts }

// Effects is ModelMesh::get_Effects, one `ldfld`. The collection holds the
// DISTINCT effects this mesh's parts use, maintained by
// ModelMeshPart.SetEffect as parts change.
func (m *ModelMesh) Effects() *ModelEffectCollection { return m.effects }

// Draw is ModelMesh::Draw(), measured at 120 bytes:
//
//	foreach part in MeshParts:
//	    Effect effect = part.Effect;
//	    if (effect == null)
//	        throw new InvalidOperationException(FrameworkResources.ModelHasNoEffect);
//	    foreach pass in effect.CurrentTechnique.Passes:
//	        pass.Apply();
//	        part.Draw();
//
// Two measured details the projection keeps.
//
// `CurrentTechnique.Passes` is RE-FETCHED on every iteration -- IL_0049 repeats
// the whole `get_CurrentTechnique; get_Passes; get_Item` chain rather than
// hoisting it -- so a pass that changes CurrentTechnique changes what the loop
// goes on to iterate. Hoisting would be a behaviour change, not an
// optimisation.
//
// The null-effect check is PER PART and throws before that part draws, but
// after every earlier part already has. A mesh whose third part has no effect
// draws two parts and then fails; it does not fail cleanly having drawn
// nothing.
//
// Unlike the reference this projection reports the refusal rather than
// throwing, and the device calls a part makes can refuse too, so Draw carries
// an error.
func (m *ModelMesh) Draw() error {
	if m.meshParts == nil {
		return nil
	}
	for _, part := range m.meshParts.wrappedArray {
		effect := concreteEffect(part.Effect())
		if effect == nil {
			return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
		}
		// The pass COUNT is read once per part, exactly as the reference reads
		// it into V_4 before the inner loop.
		technique := effect.CurrentTechnique()
		if technique == nil {
			return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
		}
		passCount := technique.Passes().Count()
		for index := int32(0); index < passCount; index++ {
			// Re-fetched every iteration, which is what the reference does: it
			// repeats the whole get_CurrentTechnique/get_Passes/get_Item chain
			// at IL_0049 rather than hoisting it.
			current := effect.CurrentTechnique()
			if current == nil {
				return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
			}
			pass := current.Passes().ItemPropertySignatureCA1DC5FC(index)
			if pass == nil {
				return fmt.Errorf("%w: %s", errModelInvalidOperation, modelHasNoEffect)
			}
			if err := pass.Apply(); err != nil {
				return err
			}
			if err := part.draw(); err != nil {
				return err
			}
		}
	}
	return nil
}
