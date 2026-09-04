package graphics

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 67 — GraphicsDevice's vertex and index binding, and the three
// non-user draw members.
// ---------------------------------------------------------------------------
//
// # The bound objects are MANAGED state, and that is the reference's shape
//
//	get_Indices        ldarg.0; ldfld _currentIB; ret
//	GetVertexBuffers   copy currentVertexBuffers[0..currentVertexBufferCount]
//
// Both answer from managed fields the setters maintain, so a consumer gets back
// the SAME objects they bound rather than something reconstructed. CNA-Go keeps
// the same fields for the same reason, and it is the only way it could: CNA
// hands back a handle, and a handle cannot be turned into the Go object a
// consumer is holding without a registry that would retain every buffer for the
// life of the process.
//
// The SETTERS reach CNA, because binding is what actually changes the device.

// The two FrameworkResources strings the draw members throw.
const (
	mustDrawSomething        = "When drawing, at least one primitive must be drawn."
	nonZeroInstanceFrequency = "Non-instanced draw calls are not valid when a vertex buffer is bound with a non-zero instance frequency."
)

// errDrawInvalidOperation projects System.InvalidOperationException.
var errDrawInvalidOperation = errors.New("operation is not valid")

// SetVertexBufferByVertexBuffer is GraphicsDevice::SetVertexBuffer(VertexBuffer):
//
//	if (vertexBuffer != null) { binding = new VertexBufferBinding(vertexBuffer);
//	                            SetVertexBuffers(&binding, 1); }
//	else                        SetVertexBuffers(null, 0);
//
// A NULL buffer unbinds everything -- it is not an error, and it does not build
// a binding at all. That is the branch a reader would most easily get wrong.
func (d *GraphicsDevice) SetVertexBufferByVertexBuffer(vertexBuffer VertexBufferReference) error {
	if resolveVertexBuffer(vertexBuffer) == nil {
		return d.SetVertexBuffers(nil)
	}
	binding, err := NewVertexBufferBindingByVertexBuffer(vertexBuffer)
	if err != nil {
		return err
	}
	return d.SetVertexBuffers([]VertexBufferBinding{binding})
}

// SetVertexBufferByVertexBufferAndInt32 is
// GraphicsDevice::SetVertexBuffer(VertexBuffer, int32), the same body with the
// two-argument binding constructor -- so a bad offset is refused by THAT
// constructor's check, which is why this member has no offset validation of its
// own. A null buffer still unbinds, and the offset is not looked at.
func (d *GraphicsDevice) SetVertexBufferByVertexBufferAndInt32(vertexBuffer VertexBufferReference, vertexOffset int32) error {
	if resolveVertexBuffer(vertexBuffer) == nil {
		return d.SetVertexBuffers(nil)
	}
	binding, err := NewVertexBufferBindingByVertexBufferAndInt32(vertexBuffer, vertexOffset)
	if err != nil {
		return err
	}
	return d.SetVertexBuffers([]VertexBufferBinding{binding})
}

// SetVertexBuffers is GraphicsDevice::SetVertexBuffers(VertexBufferBinding[]),
// which validates every binding BEFORE applying any -- and CNA says the same of
// its own route: "Every binding is validated before any is applied."
//
// An empty or nil array unbinds every vertex buffer, which is what the
// reference's `SetVertexBuffers(null, 0)` does and what CNA documents for an
// empty array.
func (d *GraphicsDevice) SetVertexBuffers(vertexBuffers []VertexBufferBinding) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	flattened := make([]int64, 0, len(vertexBuffers)*3)
	for index := range vertexBuffers {
		buffer := vertexBuffers[index].VertexBuffer()
		if buffer == nil {
			// A default(VertexBufferBinding) is an empty slot in the reference
			// and unbinds that stream; CNA takes CNA_INVALID_HANDLE for it.
			flattened = append(flattened, 0, 0, 0)
			continue
		}
		handle, err := device.HandleOf(buffer.nativeResource())
		if err != nil {
			return err
		}
		flattened = append(flattened, int64(handle),
			int64(vertexBuffers[index].VertexOffset()),
			int64(vertexBuffers[index].InstanceFrequency()))
	}
	if err := device.SetVertexBuffers(flattened); err != nil {
		return err
	}
	// The managed copy the getter answers from, taken only after CNA accepted
	// the whole array -- so a refused bind leaves the previous state readable,
	// which is what "validated before any is applied" means for the getter too.
	d.vertexBuffers = append([]VertexBufferBinding(nil), vertexBuffers...)
	return nil
}

// GetVertexBuffers is GraphicsDevice::GetVertexBuffers:
//
//	VertexBufferBinding[] copy = new VertexBufferBinding[currentVertexBufferCount];
//	Array.Copy(currentVertexBuffers, copy, currentVertexBufferCount);
//	return copy;
//
// A COPY, so a caller cannot rewrite the device's binding table by mutating
// what they were handed. It is infallible: the reference reads two managed
// fields and allocates.
func (d *GraphicsDevice) GetVertexBuffers() []VertexBufferBinding {
	if d == nil {
		return nil
	}
	return append([]VertexBufferBinding(nil), d.vertexBuffers...)
}

// Indices is GraphicsDevice::get_Indices, one `ldfld` over the managed field
// set_Indices maintains.
func (d *GraphicsDevice) Indices() *IndexBuffer {
	if d == nil {
		return nil
	}
	return d.indices
}

// SetIndices is GraphicsDevice::set_Indices, which binds the buffer on the
// device and stores it. A nil buffer unbinds, exactly as the reference's null
// does and as CNA's CNA_INVALID_HANDLE means.
func (d *GraphicsDevice) SetIndices(value IndexBufferReference) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	// Resolved before the null test, so a DynamicIndexBuffer that is itself
	// nil unbinds exactly as a null IndexBuffer does.
	buffer := resolveIndexBuffer(value)
	var handle uint64
	if buffer != nil {
		handle, err = device.HandleOf(buffer.nativeResource())
		if err != nil {
			return err
		}
	}
	if err := device.SetIndexBuffer(handle); err != nil {
		return err
	}
	d.indices = buffer
	return nil
}

// DrawPrimitives is GraphicsDevice::DrawPrimitives(PrimitiveType, int32, int32):
//
//	Helpers.CheckDisposed(this, pComPtr);
//	if (primitiveCount <= 0)
//	    throw new ArgumentOutOfRangeException("primitiveCount",
//	        FrameworkResources.MustDrawSomething);
//	if (primitiveCount > _profileCapabilities.MaxPrimitiveCount)
//	    ThrowNotSupportedException(ProfileMaxPrimitiveCount, max);   // NOT reproduced
//	VerifyCanDraw(false, false);
//	if (instanceStreamMask != 0)
//	    throw new InvalidOperationException(FrameworkResources.NonZeroInstanceFrequency);
//	if (!_insideScene) { BeginScene(); _insideScene = true; }
//	...DrawPrimitive...
//
// Three of the five are reproduced and two are recorded.
//
// The PROFILE cap is not: `ProfileCapabilities` is not a public XNA type and
// CNA-Go projects no part of it, so there is no measured maximum to compare
// against. CNA refuses a count its backend cannot draw, in its own words.
//
// `VerifyCanDraw` checks that a vertex buffer and an effect are bound, which
// CNA-Go cannot ask about for effects -- no effect surface is projected -- so
// CNA's own refusal stands for it.
//
// The instance-frequency check IS reproduced, because the projection holds the
// bindings it applied and can see a non-zero frequency among them. It is the
// one guard here that a consumer can trip without a shader.
func (d *GraphicsDevice) DrawPrimitives(primitiveType PrimitiveType, startVertex, primitiveCount int32) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	if err := d.verifyNonInstancedDraw(primitiveCount); err != nil {
		return err
	}
	return device.DrawPrimitives(uint32(primitiveType), startVertex, primitiveCount)
}

// DrawIndexedPrimitives is
// GraphicsDevice::DrawIndexedPrimitives(PrimitiveType, int32, int32, int32, int32, int32),
// whose guard prefix is DrawPrimitives' -- including the instance-frequency
// refusal, which is what makes it a NON-instanced call.
func (d *GraphicsDevice) DrawIndexedPrimitives(
	primitiveType PrimitiveType, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount int32,
) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	if err := d.verifyNonInstancedDraw(primitiveCount); err != nil {
		return err
	}
	return device.DrawIndexedPrimitives(uint32(primitiveType),
		baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount)
}

// DrawInstancedPrimitives is
// GraphicsDevice::DrawInstancedPrimitives(PrimitiveType, int32 x 6).
//
// It does NOT carry the instance-frequency refusal, and that is the whole point
// of the member: an instanced draw is exactly the call a non-zero frequency is
// for. It keeps the primitive-count check and adds one of its own on the
// instance count.
func (d *GraphicsDevice) DrawInstancedPrimitives(
	primitiveType PrimitiveType, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount, instanceCount int32,
) error {
	device, err := d.live()
	if err != nil {
		return err
	}
	if primitiveCount <= 0 {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustDrawSomething)
	}
	if instanceCount <= 0 {
		return fmt.Errorf("%w: instanceCount: %s", errArgumentOutOfRange, mustDrawSomething)
	}
	return device.DrawInstancedPrimitives(uint32(primitiveType),
		baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount, instanceCount)
}

// verifyNonInstancedDraw is the guard prefix the two non-instanced draw members
// share, in the reference's order.
func (d *GraphicsDevice) verifyNonInstancedDraw(primitiveCount int32) error {
	if primitiveCount <= 0 {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustDrawSomething)
	}
	for _, binding := range d.vertexBuffers {
		if binding.InstanceFrequency() != 0 {
			return fmt.Errorf("%w: %s", errDrawInvalidOperation, nonZeroInstanceFrequency)
		}
	}
	return nil
}

// nativeResourceHandle is the unexported reach a bind needs. It stays here
// rather than on interop.Resource's public surface because the raw-handle rule
// forbids publishing one.
var _ = interop.ErrDisposed
