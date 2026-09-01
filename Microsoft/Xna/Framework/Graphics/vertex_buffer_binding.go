package graphics

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// Foundation 67 — VertexBufferBinding.
// ---------------------------------------------------------------------------

// VertexBufferBinding is Microsoft.Xna.Framework.Graphics.VertexBufferBinding:
//
//	.class public sequential ansi sealed beforefieldinit VertexBufferBinding
//	  .field private VertexBuffer _vertexBuffer
//	  .field private int32        _vertexOffset
//	  .field private int32        _instanceFrequency
//
// A value struct with three read-only properties and one implicit conversion.
// It is what `GraphicsDevice::SetVertexBuffers` takes and what `GetVertexBuffers`
// hands back, and the two `SetVertexBuffer` overloads build one internally.
//
// # It is a STRUCT, and its zero value is meaningful
//
// The reference's `default(VertexBufferBinding)` has a null buffer and two
// zeros, and `SetVertexBuffers` treats such an entry as an empty slot. Go's
// zero value is the same thing, so nothing is invented: a binding a consumer
// never initialised binds nothing, exactly as in C#.
//
// Its three getters are field reads with no validation, so all three are
// infallible -- including on the zero value.
type VertexBufferBinding struct {
	vertexBuffer      *VertexBuffer
	vertexOffset      int32
	instanceFrequency int32
}

// NewVertexBufferBindingByVertexBufferAndInt32AndInt32 is
// VertexBufferBinding::.ctor(VertexBuffer, int32, int32):
//
//	if (vertexBuffer == null)
//	    throw new ArgumentNullException("vertexBuffer", FrameworkResources.NullNotAllowed);
//	if (vertexOffset < 0 || vertexOffset >= vertexBuffer._vertexCount)
//	    throw new ArgumentOutOfRangeException("vertexOffset");
//	if (instanceFrequency < 0)
//	    throw new ArgumentOutOfRangeException("instanceFrequency");
//
// Two details the IL settles that a reader would otherwise guess.
//
// The offset bound is EXCLUSIVE: `vertexOffset == VertexCount` is refused, so a
// binding cannot start past the last vertex. And the instance-frequency check
// runs AFTER both offset checks, so a caller with a bad offset and a bad
// frequency is told about the offset.
//
// Neither ArgumentOutOfRangeException carries a message. The reference uses the
// one-argument constructor at both sites, so the parameter name is all a caller
// gets -- and inventing a sentence here would be inventing evidence.
func NewVertexBufferBindingByVertexBufferAndInt32AndInt32(vertexBuffer *VertexBuffer, vertexOffset, instanceFrequency int32) (VertexBufferBinding, error) {
	if vertexBuffer == nil {
		return VertexBufferBinding{}, fmt.Errorf("%w: vertexBuffer: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if vertexOffset < 0 || vertexOffset >= vertexBuffer.VertexCount() {
		return VertexBufferBinding{}, fmt.Errorf("%w: vertexOffset", errArgumentOutOfRange)
	}
	if instanceFrequency < 0 {
		return VertexBufferBinding{}, fmt.Errorf("%w: instanceFrequency", errArgumentOutOfRange)
	}
	return VertexBufferBinding{
		vertexBuffer:      vertexBuffer,
		vertexOffset:      vertexOffset,
		instanceFrequency: instanceFrequency,
	}, nil
}

// NewVertexBufferBindingByVertexBufferAndInt32 is
// VertexBufferBinding::.ctor(VertexBuffer, int32), which is the three-argument
// body with the frequency check dropped and a zero stored.
func NewVertexBufferBindingByVertexBufferAndInt32(vertexBuffer *VertexBuffer, vertexOffset int32) (VertexBufferBinding, error) {
	return NewVertexBufferBindingByVertexBufferAndInt32AndInt32(vertexBuffer, vertexOffset, 0)
}

// NewVertexBufferBindingByVertexBuffer is
// VertexBufferBinding::.ctor(VertexBuffer), which keeps ONLY the null check:
// with a zero offset there is nothing to validate, so a buffer of any size is
// accepted -- including one the two-argument constructor would refuse an offset
// into.
func NewVertexBufferBindingByVertexBuffer(vertexBuffer *VertexBuffer) (VertexBufferBinding, error) {
	if vertexBuffer == nil {
		return VertexBufferBinding{}, fmt.Errorf("%w: vertexBuffer: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	return VertexBufferBinding{vertexBuffer: vertexBuffer}, nil
}

// VertexBufferBindingOperatorImplicitByVertexBuffer is
// VertexBufferBinding::op_Implicit(VertexBuffer):
//
//	newobj VertexBufferBinding::.ctor(VertexBuffer); ret
//
// Go has no implicit conversion, so the settled operator rule spells it as a
// package-level function -- and it is FALLIBLE for the same reason the
// constructor it calls is: a null buffer is refused.
func VertexBufferBindingOperatorImplicitByVertexBuffer(vertexBuffer *VertexBuffer) (VertexBufferBinding, error) {
	return NewVertexBufferBindingByVertexBuffer(vertexBuffer)
}

// VertexBuffer is VertexBufferBinding::get_VertexBuffer, one field read.
func (b VertexBufferBinding) VertexBuffer() *VertexBuffer { return b.vertexBuffer }

// VertexOffset is VertexBufferBinding::get_VertexOffset.
func (b VertexBufferBinding) VertexOffset() int32 { return b.vertexOffset }

// InstanceFrequency is VertexBufferBinding::get_InstanceFrequency.
func (b VertexBufferBinding) InstanceFrequency() int32 { return b.instanceFrequency }
