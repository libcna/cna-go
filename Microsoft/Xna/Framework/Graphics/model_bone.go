package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// ModelBone is Microsoft.Xna.Framework.Graphics.ModelBone:
//
//	.class public auto ansi sealed beforefieldinit ModelBone
//	       extends [mscorlib]System.Object
//
// One node of a model's skeleton: a name, an index, a local transform, a
// parent and a child collection.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # Pure managed, and every member is one field access
//
// Nothing here reaches a device. The five contract members are `ldfld` reads
// and, for Transform, one `stfld` write with no validation of any kind: a
// non-invertible matrix, a zero matrix and a NaN matrix are all stored
// unchanged. So none of them carries an error.
//
// # Transform is LOCAL, and that is the whole reason CopyAbsoluteBoneTransformsTo exists
//
// The matrix a bone holds is relative to its parent. A consumer that wants
// world-space bone matrices asks the Model for them, and Model walks the parent
// chain; a bone cannot do it alone because it does not know the array the walk
// accumulates into.
//
// # The constructor and SetParentAndChildren are not projected
//
// Both are `assembly` in the reference metadata, so no CLR caller outside
// Microsoft.Xna.Framework.Graphics.dll can reach them, and the pinned contract
// declares neither. A ModelBone reaches a consumer only through a Model, which
// arrives from ContentManager.Load<Model>.
type ModelBone struct {
	name      string
	index     int32
	transform framework.Matrix
	parent    *ModelBone
	children  *ModelBoneCollection
}

// Name is ModelBone::get_Name, one `ldfld`.
func (b *ModelBone) Name() string { return b.name }

// Index is ModelBone::get_Index, one `ldfld`. It is the bone's position in the
// owning Model's Bones collection, and it is what
// CopyAbsoluteBoneTransformsTo uses to find a parent's already-computed matrix.
func (b *ModelBone) Index() int32 { return b.index }

// Transform is ModelBone::get_Transform, one `ldfld`. The value is the bone's
// transform RELATIVE TO ITS PARENT.
func (b *ModelBone) Transform() framework.Matrix { return b.transform }

// SetTransform is ModelBone::set_Transform, one `stfld`. It validates nothing.
func (b *ModelBone) SetTransform(value framework.Matrix) { b.transform = value }

// Parent is ModelBone::get_Parent, one `ldfld`. It is nil for the root bone,
// which is the terminating case of the parent walk.
func (b *ModelBone) Parent() *ModelBone { return b.parent }

// Children is ModelBone::get_Children, one `ldfld`.
func (b *ModelBone) Children() *ModelBoneCollection { return b.children }
