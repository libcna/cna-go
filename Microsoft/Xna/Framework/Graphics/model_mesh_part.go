package graphics

// ModelMeshPart is Microsoft.Xna.Framework.Graphics.ModelMeshPart:
//
//	.class public auto ansi sealed beforefieldinit ModelMeshPart
//	       extends [mscorlib]System.Object
//
// One draw call's worth of a mesh: a slice of a vertex buffer, a slice of an
// index buffer, and the effect to draw it with.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
// # Draw is NOT contract surface
//
// `ModelMeshPart::Draw()` is `assembly` in the reference metadata and the
// pinned contract does not declare it, so it is projected as an unexported
// method. Its only caller is ModelMesh.Draw, which is the public entry point.
type ModelMeshPart struct {
	vertexOffset   int32
	numVertices    int32
	startIndex     int32
	primitiveCount int32
	vertexBuffer   *VertexBuffer
	indexBuffer    *IndexBuffer
	effect         *Effect
	tag            any
	// parent is the mesh this part belongs to. It is the `assembly` field
	// SetEffect walks to reach its siblings, and it has no public accessor --
	// the reference declares none either.
	parent *ModelMesh
}

// StartIndex is ModelMeshPart::get_StartIndex, one `ldfld`.
func (p *ModelMeshPart) StartIndex() int32 { return p.startIndex }

// PrimitiveCount is ModelMeshPart::get_PrimitiveCount, one `ldfld`.
func (p *ModelMeshPart) PrimitiveCount() int32 { return p.primitiveCount }

// VertexOffset is ModelMeshPart::get_VertexOffset, one `ldfld`.
func (p *ModelMeshPart) VertexOffset() int32 { return p.vertexOffset }

// NumVertices is ModelMeshPart::get_NumVertices, one `ldfld`.
func (p *ModelMeshPart) NumVertices() int32 { return p.numVertices }

// IndexBuffer is ModelMeshPart::get_IndexBuffer, one `ldfld`.
func (p *ModelMeshPart) IndexBuffer() *IndexBuffer { return p.indexBuffer }

// VertexBuffer is ModelMeshPart::get_VertexBuffer, one `ldfld`.
func (p *ModelMeshPart) VertexBuffer() *VertexBuffer { return p.vertexBuffer }

// Effect is ModelMeshPart::get_Effect, one `ldfld`, widened to EffectReference
// because Effect is a substitutable base at return positions too.
func (p *ModelMeshPart) Effect() EffectReference {
	if p.effect == nil {
		return nil
	}
	return p.effect
}

// Tag is ModelMeshPart::get_Tag, `System.Object` and therefore `any`.
func (p *ModelMeshPart) Tag() any { return p.tag }

// SetTag is ModelMeshPart::set_Tag, one `stfld` with no validation.
func (p *ModelMeshPart) SetTag(value any) { p.tag = value }

// SetEffect is ModelMeshPart::set_Effect, and it is the only member in the
// Model family with real behaviour. Measured at 175 bytes:
//
//	if (value == this.effect) return;              // IDENTITY, and an early out
//
//	bool otherStillUsesOld = false;
//	bool otherAlreadyUsesNew = false;
//	for (int i = 0; i < parent.MeshParts.Count; i++) {
//	    ModelMeshPart part = parent.MeshParts[i];
//	    if (ReferenceEquals(part, this)) continue;
//	    Effect other = part.Effect;
//	    if (ReferenceEquals(other, this.effect))  otherStillUsesOld  = true;
//	    else if (ReferenceEquals(other, value))   otherAlreadyUsesNew = true;
//	}
//	if (!otherStillUsesOld  && this.effect != null) parent.Effects.Remove(this.effect);
//	if (!otherAlreadyUsesNew && value != null)      parent.Effects.Add(value);
//	this.effect = value;
//
// It is a REFERENCE COUNT over the parent mesh's Effects collection: the old
// effect leaves that collection only when no sibling still uses it, and the new
// one joins only when no sibling already has. So `mesh.Effects` stays the set
// of distinct effects its parts use, which is what makes ModelMesh.Draw able to
// set the transforms once per effect instead of once per part.
//
// Three things the projection must not simplify:
//
//   - every comparison is System.Object::ReferenceEquals, never Effect equality.
//     Go pointer identity is the faithful counterpart.
//   - the early return tests the FIELD, so assigning the same effect twice does
//     nothing at all -- no scan, no collection churn.
//   - the `else if` is exclusive. A sibling on the old effect suppresses only
//     the Remove; a sibling already on the new one suppresses only the Add.
//
// A part with no parent mesh cannot reach the collection. The reference would
// throw NullReferenceException; this projection stores the effect and leaves
// the collection alone, because a NullReferenceException is not contract
// behaviour a binding reproduces.
func (p *ModelMeshPart) SetEffect(value EffectReference) {
	// The parameter widens to EffectReference because Effect is a substitutable
	// base -- a C# caller assigns a BasicEffect here. The body works in
	// CONCRETE pointers, because every comparison the reference makes is
	// System.Object::ReferenceEquals over the stored field, and a nil interface
	// holding a nil pointer must be the same thing as a nil field.
	effect := concreteEffect(value)
	if effect == p.effect {
		return
	}
	if p.parent == nil {
		p.effect = effect
		return
	}
	otherStillUsesOld := false
	otherAlreadyUsesNew := false
	for _, part := range p.parent.meshParts.wrappedArray {
		if part == p {
			continue
		}
		other := concreteEffect(part.Effect())
		switch {
		case other == p.effect:
			otherStillUsesOld = true
		case other == effect:
			otherAlreadyUsesNew = true
		}
	}
	if !otherStillUsesOld && p.effect != nil {
		p.parent.effects.remove(p.effect)
	}
	if !otherAlreadyUsesNew && effect != nil {
		p.parent.effects.add(effect)
	}
	p.effect = effect
}

// concreteEffect narrows a substitutable Effect reference back to the pointer
// the field holds. A nil interface and an interface holding a nil pointer are
// both the CLR's null, which is what the reference's null tests compare against.
func concreteEffect(value EffectReference) *Effect {
	if value == nil {
		return nil
	}
	if effect, ok := any(value).(*Effect); ok {
		return effect
	}
	return nil
}

// draw is ModelMeshPart::Draw, `assembly` in the reference and therefore
// unexported. Measured at 79 bytes:
//
//	if (NumVertices <= 0) return;
//	GraphicsDevice device = vertexBuffer.GraphicsDevice;
//	device.SetVertexBuffer(vertexBuffer, vertexOffset);
//	device.Indices = indexBuffer;
//	device.DrawIndexedPrimitives(PrimitiveType.TriangleList, 0, 0,
//	                             numVertices, startIndex, primitiveCount);
//
// Three measured details. The device comes from the VERTEX BUFFER rather than
// from the mesh or a parameter. The primitive type is the `ldc.i4.0` literal,
// TriangleList, and is not configurable. And an empty part returns before
// touching the device at all, so it does not disturb the bindings a previous
// part left.
func (p *ModelMeshPart) draw() error {
	if p.numVertices <= 0 {
		return nil
	}
	if p.vertexBuffer == nil {
		return nil
	}
	device := p.vertexBuffer.GraphicsDevice()
	if device == nil {
		return nil
	}
	if err := device.SetVertexBufferByVertexBufferAndInt32(p.vertexBuffer, p.vertexOffset); err != nil {
		return err
	}
	if err := device.SetIndices(p.indexBuffer); err != nil {
		return err
	}
	return device.DrawIndexedPrimitives(PrimitiveTypeTriangleList, 0, 0,
		p.numVertices, p.startIndex, p.primitiveCount)
}
