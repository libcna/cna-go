package graphics

import (
	"errors"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

func bindableBuffer(vertexCount int32) *VertexBuffer {
	return newVertexBuffer(nil, nil, positionColourDeclaration, interop.VertexBufferInfo{
		VertexCount: vertexCount, VertexStride: 16,
	})
}

// TestTheBindingConstructorsRefuseInTheReferencesOrder pins all three, and the
// order between the two range checks: the offset is validated before the
// instance frequency, so a caller with both wrong is told about the offset.
func TestTheBindingConstructorsRefuseInTheReferencesOrder(t *testing.T) {
	buffer := bindableBuffer(4)
	if _, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(nil, 0, 0); err == nil ||
		!strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("a nil buffer = %v, want the reference's NullNotAllowed message", err)
	}
	for _, offset := range []int32{-1, 4, 5} {
		if _, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, offset, 0); err == nil ||
			!strings.Contains(err.Error(), "vertexOffset") {
			t.Fatalf("offset %d = %v, want the vertexOffset refusal", offset, err)
		}
	}
	// The bound is EXCLUSIVE: the last valid offset is VertexCount-1.
	if _, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, 3, 0); err != nil {
		t.Fatalf("offset 3 of a four-vertex buffer was refused: %v", err)
	}
	if _, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, -1, -1); err == nil ||
		!strings.Contains(err.Error(), "vertexOffset") {
		t.Fatalf("a bad offset AND a bad frequency = %v, want the offset reported first", err)
	}
	if _, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, 0, -1); err == nil ||
		!strings.Contains(err.Error(), "instanceFrequency") {
		t.Fatalf("a negative frequency = %v", err)
	}
	// Neither range refusal carries a MESSAGE: the reference uses the
	// one-argument ArgumentOutOfRangeException constructor at both sites.
	_, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, 9, 0)
	if strings.Contains(err.Error(), nullNotAllowed) || strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("%v borrowed a message the reference's throw site does not carry", err)
	}
}

// TestTheOneArgumentBindingKeepsOnlyTheNullCheck pins that the shortest
// constructor validates nothing but the buffer -- with a zero offset there is
// nothing to range-check, so it accepts a buffer the two-argument one would
// refuse an offset into.
func TestTheOneArgumentBindingKeepsOnlyTheNullCheck(t *testing.T) {
	binding, err := NewVertexBufferBindingByVertexBuffer(bindableBuffer(1))
	if err != nil {
		t.Fatalf("a one-vertex buffer was refused: %v", err)
	}
	// Both stored values are ZERO. The reference's shortest constructor stores
	// `ldc.i4.0` into each, and a projection that delegated to a longer one
	// with anything else would be binding a stream the caller did not ask for.
	if binding.VertexOffset() != 0 || binding.InstanceFrequency() != 0 {
		t.Fatalf("the one-argument binding stored %d/%d, want zeros", binding.VertexOffset(), binding.InstanceFrequency())
	}
	if binding.VertexBuffer() == nil {
		t.Fatal("the one-argument binding stored no buffer")
	}
	if _, err := NewVertexBufferBindingByVertexBuffer(nil); err == nil {
		t.Fatal("a nil buffer was accepted")
	}
	// op_Implicit is `newobj .ctor(VertexBuffer); ret`, so it carries exactly
	// that constructor's one failure and no other.
	implicit, err := VertexBufferBindingOperatorImplicitByVertexBuffer(bindableBuffer(2))
	if err != nil {
		t.Fatalf("op_Implicit: %v", err)
	}
	if implicit.VertexBuffer() == nil || implicit.VertexOffset() != 0 {
		t.Fatal("op_Implicit did not build the one-argument binding")
	}
	if _, err := VertexBufferBindingOperatorImplicitByVertexBuffer(nil); err == nil {
		t.Fatal("op_Implicit accepted a nil buffer")
	}
}

// TestTheZeroBindingIsAnEmptySlot pins that Go's zero value means what
// `default(VertexBufferBinding)` means in C#: a slot that binds nothing, whose
// three getters answer without failing.
func TestTheZeroBindingIsAnEmptySlot(t *testing.T) {
	var empty VertexBufferBinding
	if empty.VertexBuffer() != nil || empty.VertexOffset() != 0 || empty.InstanceFrequency() != 0 {
		t.Fatal("the zero binding is not empty")
	}
}

// TestTheDrawGuardsRunWithoutADevice pins the two guards the projection makes
// itself, which are reachable on an unconstructed device because they come
// first.
func TestTheDrawGuardsRunWithoutADevice(t *testing.T) {
	device := &GraphicsDevice{}
	// With no live device every draw is refused, and NOT with the primitive
	// count message -- the device check comes first in the projection because
	// CheckDisposed does in the reference.
	if err := device.DrawPrimitives(PrimitiveTypeTriangleList, 0, 0); err == nil {
		t.Fatal("a draw on an unconstructed device was accepted")
	} else if strings.Contains(err.Error(), mustDrawSomething) {
		t.Fatal("the primitive-count guard ran before the disposal check")
	}
	if device.Indices() != nil {
		t.Fatal("an unconstructed device reports a bound index buffer")
	}
	if device.GetVertexBuffers() != nil {
		t.Fatal("an unconstructed device reports bound vertex buffers")
	}
}

// TestTheNonInstancedDrawsRefuseANonZeroInstanceFrequency pins the one draw
// guard the projection CAN reproduce without a shader, and pins that the
// instanced member does not carry it -- which is the whole point of that
// member.
func TestTheNonInstancedDrawsRefuseANonZeroInstanceFrequency(t *testing.T) {
	device := &GraphicsDevice{}
	binding, err := NewVertexBufferBindingByVertexBufferAndInt32AndInt32(bindableBuffer(4), 0, 1)
	if err != nil {
		t.Fatalf("a binding with frequency 1: %v", err)
	}
	device.vertexBuffers = []VertexBufferBinding{binding}
	if err := device.verifyNonInstancedDraw(1); err == nil ||
		!strings.Contains(err.Error(), nonZeroInstanceFrequency) {
		t.Fatalf("%v, want the reference's non-instanced refusal", err)
	}
	if !errors.Is(device.verifyNonInstancedDraw(1), errDrawInvalidOperation) {
		t.Fatal("the refusal is not InvalidOperationException")
	}
	// A zero primitive count is reported first, because the reference checks it
	// before it looks at the streams.
	if err := device.verifyNonInstancedDraw(0); err == nil ||
		!strings.Contains(err.Error(), mustDrawSomething) {
		t.Fatalf("%v, want the primitive-count refusal first", err)
	}
	// With every frequency zero the guard passes.
	zero, err := NewVertexBufferBindingByVertexBuffer(bindableBuffer(4))
	if err != nil {
		t.Fatalf("a zero-frequency binding: %v", err)
	}
	device.vertexBuffers = []VertexBufferBinding{zero}
	if err := device.verifyNonInstancedDraw(1); err != nil {
		t.Fatalf("a zero-frequency stream was refused: %v", err)
	}
}

// TestGetVertexBuffersReturnsACopy pins `Array.Copy` into a fresh array: a
// caller must not be able to rewrite the device's binding table by mutating
// what they were handed.
func TestGetVertexBuffersReturnsACopy(t *testing.T) {
	device := &GraphicsDevice{}
	first, err := NewVertexBufferBindingByVertexBuffer(bindableBuffer(4))
	if err != nil {
		t.Fatalf("a binding: %v", err)
	}
	device.vertexBuffers = []VertexBufferBinding{first}
	returned := device.GetVertexBuffers()
	if len(returned) != 1 || returned[0].VertexBuffer() != first.VertexBuffer() {
		t.Fatal("GetVertexBuffers did not answer the bound binding")
	}
	returned[0] = VertexBufferBinding{}
	if device.GetVertexBuffers()[0].VertexBuffer() == nil {
		t.Fatal("mutating the returned array changed the device's table")
	}
}

// TestSetVertexBufferWithANilBufferUnbindsRatherThanRefusing pins the branch a
// reader would most easily get wrong: the reference's null path calls
// SetVertexBuffers(null, 0), it does not build a binding and it does not throw.
func TestSetVertexBufferWithANilBufferUnbindsRatherThanRefusing(t *testing.T) {
	device := &GraphicsDevice{}
	// With no live device the unbind still reaches CNA and is refused there --
	// what matters is that it is NOT refused by the binding constructor's null
	// check, which would report "vertexBuffer".
	err := device.SetVertexBufferByVertexBuffer(nil)
	if err != nil && strings.Contains(err.Error(), "vertexBuffer") {
		t.Fatalf("%v: a nil buffer must unbind, not fail the binding constructor", err)
	}
	err = device.SetVertexBufferByVertexBufferAndInt32(nil, 99)
	if err != nil && (strings.Contains(err.Error(), "vertexBuffer") || strings.Contains(err.Error(), "vertexOffset")) {
		t.Fatalf("%v: a nil buffer must unbind without looking at the offset", err)
	}
	// A non-nil buffer with a bad offset IS refused, by the binding
	// constructor, before the device is reached.
	err = device.SetVertexBufferByVertexBufferAndInt32(bindableBuffer(4), 4)
	if err == nil || !strings.Contains(err.Error(), "vertexOffset") {
		t.Fatalf("%v, want the binding constructor's offset refusal", err)
	}
}
