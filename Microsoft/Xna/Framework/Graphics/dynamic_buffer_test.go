package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// The dynamic buffers add four things to their bases: a latching IsContentLost,
// a ContentLost event, a SetData that carries SetDataOptions, and the
// IDynamicGraphicsResource dispatch that CLEARS the latch after a successful
// upload. Three of the four are pure managed state and are measured here
// without a device; the upload is in the native-stress scenario.

// newManagedDynamicVertexBuffer is the object the constructor builds minus its
// native half, so the managed state machine can be exercised on its own. It
// installs the binding the constructor installs, because every claim below is
// about which object answers.
func newManagedDynamicVertexBuffer() *DynamicVertexBuffer {
	base := &VertexBuffer{resource: &GraphicsResource{}}
	buffer := &DynamicVertexBuffer{buffer: base}
	base.bindDerived(buffer)
	return buffer
}

// newManagedDynamicIndexBuffer is its index-side twin.
func newManagedDynamicIndexBuffer() *DynamicIndexBuffer {
	base := &IndexBuffer{resource: &GraphicsResource{}}
	buffer := &DynamicIndexBuffer{buffer: base}
	base.bindDerived(buffer)
	return buffer
}

// TestDynamicBufferConstructorGuardOrderIsTheReferences pins the one place the
// two constructors DIVERGE, which no reader would predict: the vertex side's
// declaration-keyed constructor checks its declaration first and the index
// side's Type-keyed constructor checks its COUNT first.
func TestDynamicBufferConstructorGuardOrderIsTheReferences(t *testing.T) {
	// Vertex, declaration-keyed: a nil declaration is refused ahead of a bad
	// count, so a caller passing both is told about the declaration.
	_, err := NewDynamicVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(nil, nil, -1, BufferUsageNone)
	if !errors.Is(err, errGraphicsResourceArgumentNull) {
		t.Fatalf("a nil declaration with a bad count = %v; the declaration check runs FIRST", err)
	}
	if !containsSubstring(err.Error(), "vertexDeclaration") {
		t.Fatalf("the refusal named %q, not vertexDeclaration", err)
	}
	// Index, Type-keyed: the COUNT is checked before the type is resolved, the
	// opposite of the vertex side's Type-keyed constructor.
	_, err = NewDynamicIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(nil, nil, 0, BufferUsageNone)
	if !errors.Is(err, errArgumentOutOfRange) {
		t.Fatalf("a nil type with a zero count = %v; the count check runs FIRST", err)
	}
	if !containsSubstring(err.Error(), "indexCount") {
		t.Fatalf("the refusal named %q, not indexCount", err)
	}
	if !containsSubstring(err.Error(), resourcesMustBeGreaterThanZeroSize) {
		t.Fatalf("the count refusal carried %q, not the reference's message", err)
	}
	// And the vertex side's Type-keyed constructor resolves the TYPE first,
	// which is the divergence stated from the other direction.
	_, err = NewDynamicVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(nil, nil, 0, BufferUsageNone)
	if errors.Is(err, errArgumentOutOfRange) && containsSubstring(err.Error(), "vertexCount") {
		t.Fatal("the vertex Type-keyed constructor reported the count; the reference resolves the declaration first")
	}
}

// TestDynamicBufferContentLostLatches pins that the field, once true, is
// answered without asking anything and never goes back on its own.
func TestDynamicBufferContentLostLatches(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	// A latched buffer answers true with no native read at all -- the composed
	// resource here holds no handle, so a read would fail rather than answer.
	buffer.contentLost = true
	lost, err := buffer.IsContentLost()
	if err != nil || !lost {
		t.Fatalf("a latched buffer answered (%v, %v); the latch short-circuits the device read", lost, err)
	}
	index := newManagedDynamicIndexBuffer()
	index.contentLost = true
	lost, err = index.IsContentLost()
	if err != nil || !lost {
		t.Fatalf("a latched index buffer answered (%v, %v)", lost, err)
	}
}

// TestDynamicBufferSetContentLostRaisesOnlyOnTrue pins SetContentLost's whole
// 23-byte body: it always stores, and raises ONLY when the flag is true.
func TestDynamicBufferSetContentLostRaisesOnlyOnTrue(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	raised := 0
	var sender any
	var args *framework.EventArgs
	if _, err := buffer.AddContentLostHandler(func(s any, a *framework.EventArgs) error {
		raised++
		sender, args = s, a
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// False stores and returns.
	buffer.setContentLost(false)
	if buffer.contentLost || raised != 0 {
		t.Fatalf("setContentLost(false) stored %v and raised %d times", buffer.contentLost, raised)
	}
	// True stores and raises, with the object as sender and EventArgs.Empty.
	buffer.setContentLost(true)
	if !buffer.contentLost || raised != 1 {
		t.Fatalf("setContentLost(true) stored %v and raised %d times", buffer.contentLost, raised)
	}
	if sender != any(buffer) {
		t.Fatalf("ContentLost's sender is %T; the reference pushes `ldarg.0`, the buffer itself", sender)
	}
	if args != framework.EventArgsEmpty() {
		t.Fatal("ContentLost carried something other than EventArgs.Empty")
	}
	// And back to false: stores, does not raise.
	buffer.setContentLost(false)
	if buffer.contentLost || raised != 1 {
		t.Fatalf("setContentLost(false) after a raise stored %v and raised %d times", buffer.contentLost, raised)
	}
}

// TestDynamicIndexBufferSetContentLostRaisesOnlyOnTrue is the same claim on the
// index side, which has its own 23-byte copy of the body.
func TestDynamicIndexBufferSetContentLostRaisesOnlyOnTrue(t *testing.T) {
	buffer := newManagedDynamicIndexBuffer()
	raised := 0
	if _, err := buffer.AddContentLostHandler(func(any, *framework.EventArgs) error { raised++; return nil }); err != nil {
		t.Fatal(err)
	}
	buffer.setContentLost(false)
	if raised != 0 {
		t.Fatalf("setContentLost(false) raised %d times", raised)
	}
	buffer.setContentLost(true)
	if !buffer.contentLost || raised != 1 {
		t.Fatalf("setContentLost(true) stored %v and raised %d times", buffer.contentLost, raised)
	}
}

// TestNoteContentRestoredResolvesTheWholeObject is the identity claim, and it
// is the one a bare receiver would silently break: CopyData tests `ldarg.0`
// with `isinst IDynamicGraphicsResource`, so the composed base must resolve the
// OBJECT before asking. A VertexBuffer half is not dynamic and would answer no
// for every buffer in the profile.
func TestNoteContentRestoredResolvesTheWholeObject(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	buffer.contentLost = true
	buffer.buffer.noteContentRestored()
	if buffer.contentLost {
		t.Fatal("a successful upload did not clear the latch; noteContentRestored answered with the base half")
	}
	index := newManagedDynamicIndexBuffer()
	index.contentLost = true
	index.buffer.noteContentRestored()
	if index.contentLost {
		t.Fatal("the index buffer's latch survived an upload")
	}
	// A PLAIN buffer is not an IDynamicGraphicsResource, and the same call must
	// do nothing rather than fail.
	plain := &VertexBuffer{resource: &GraphicsResource{}}
	plain.resource.bindDerived(plain)
	plain.noteContentRestored()
	plainIndex := &IndexBuffer{resource: &GraphicsResource{}}
	plainIndex.resource.bindDerived(plainIndex)
	plainIndex.noteContentRestored()
}

// TestNoteContentRestoredDoesNotRaise pins the half of SetContentLost the clear
// path depends on: it is called with FALSE, so clearing a latch must be silent.
// A consumer subscribed to ContentLost would otherwise be told the content was
// lost every time they uploaded.
func TestNoteContentRestoredDoesNotRaise(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	raised := 0
	if _, err := buffer.AddContentLostHandler(func(any, *framework.EventArgs) error { raised++; return nil }); err != nil {
		t.Fatal(err)
	}
	buffer.contentLost = true
	buffer.buffer.noteContentRestored()
	if raised != 0 {
		t.Fatalf("clearing the latch raised ContentLost %d times", raised)
	}
}

// TestRenderTargetsCarryTheSameDynamicMember pins that the interface has four
// implementers and not two: Texture2D::CopyData and TextureCube::CopyData carry
// the same `isinst` tail, and the render targets are what it finds.
func TestRenderTargetsCarryTheSameDynamicMember(t *testing.T) {
	target := &RenderTarget2D{texture: &Texture2D{texture: &Texture{resource: &GraphicsResource{}}}}
	target.texture.texture.resource.bindDerived(target)
	target.contentLost = true
	target.texture.noteContentRestored()
	if target.contentLost {
		t.Fatal("a successful upload to a RenderTarget2D did not clear its content-lost latch")
	}
	cube := &RenderTargetCube{cube: &TextureCube{texture: &Texture{resource: &GraphicsResource{}}}}
	cube.cube.texture.resource.bindDerived(cube)
	cube.contentLost = true
	cube.cube.noteContentRestored()
	if cube.contentLost {
		t.Fatal("a successful upload to a RenderTargetCube did not clear its content-lost latch")
	}
	// A plain texture is not dynamic and the same call does nothing.
	plain := &Texture2D{texture: &Texture{resource: &GraphicsResource{}}}
	plain.texture.resource.bindDerived(plain)
	plain.noteContentRestored()

	// And the raise rule holds on the texture half too: SetContentLost is ONE
	// body with four implementers, so a render target that stored without
	// raising, or raised without storing, would be a different body.
	for name, target := range map[string]dynamicGraphicsResource{
		"RenderTarget2D":   target,
		"RenderTargetCube": cube,
	} {
		raised := 0
		var subscribe func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
		var latched func() bool
		switch typed := target.(type) {
		case *RenderTarget2D:
			subscribe, latched = typed.AddContentLostHandler, func() bool { return typed.contentLost }
		case *RenderTargetCube:
			subscribe, latched = typed.AddContentLostHandler, func() bool { return typed.contentLost }
		}
		if _, err := subscribe(func(any, *framework.EventArgs) error { raised++; return nil }); err != nil {
			t.Fatal(err)
		}
		target.setContentLost(false)
		if raised != 0 || latched() {
			t.Fatalf("%s.setContentLost(false) raised %d times and latched %v", name, raised, latched())
		}
		target.setContentLost(true)
		if raised != 1 || !latched() {
			t.Fatalf("%s.setContentLost(true) raised %d times and latched %v", name, raised, latched())
		}
	}
}

// TestDynamicBuffersAreAcceptedWhereTheirBasesAre is the substitutability claim
// in the form a consumer meets it: every position that names the base takes the
// derived value. It compiles or it does not.
func TestDynamicBuffersAreAcceptedWhereTheirBasesAre(t *testing.T) {
	var vertex VertexBufferReference = newManagedDynamicVertexBuffer()
	if resolveVertexBuffer(vertex) == nil {
		t.Fatal("a DynamicVertexBuffer resolved to no VertexBuffer")
	}
	var index IndexBufferReference = newManagedDynamicIndexBuffer()
	if resolveIndexBuffer(index) == nil {
		t.Fatal("a DynamicIndexBuffer resolved to no IndexBuffer")
	}
	// A nil interface and an interface holding a typed nil are ONE null to the
	// reference, and must be one here.
	if resolveVertexBuffer(nil) != nil {
		t.Fatal("a nil VertexBufferReference resolved to something")
	}
	var typedNil *DynamicVertexBuffer
	if resolveVertexBuffer(typedNil) != nil {
		t.Fatal("a typed-nil DynamicVertexBuffer resolved to something")
	}
	var typedNilIndex *DynamicIndexBuffer
	if resolveIndexBuffer(typedNilIndex) != nil {
		t.Fatal("a typed-nil DynamicIndexBuffer resolved to something")
	}
}

// TestDisposedDynamicBufferNamesItself is the other identity site: the
// ObjectDisposedException CopyData throws names the OBJECT, so a disposed
// DynamicVertexBuffer must not report itself as a VertexBuffer.
func TestDisposedDynamicBufferNamesItself(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	buffer.buffer.resource.isDisposed = true
	err := buffer.buffer.checkDisposed()
	if err == nil {
		t.Fatal("a disposed buffer was not refused")
	}
	if !containsSubstring(err.Error(), "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer") {
		t.Fatalf("the disposal refusal said %q; the reference names the object's own type", err)
	}
	index := newManagedDynamicIndexBuffer()
	index.buffer.resource.isDisposed = true
	err = index.buffer.checkDisposed()
	if err == nil || !containsSubstring(err.Error(), "Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer") {
		t.Fatalf("the index disposal refusal said %v", err)
	}
	// And an undisposed buffer is not refused.
	fresh := newManagedDynamicVertexBuffer()
	if err := fresh.buffer.checkDisposed(); err != nil {
		t.Fatalf("an undisposed buffer was refused: %v", err)
	}
}

// TestDynamicBufferForwardsItsInheritedSurface pins that the derived type
// re-exposes what it inherits, and answers from the SAME state its base does
// rather than from a copy.
func TestDynamicBufferForwardsItsInheritedSurface(t *testing.T) {
	buffer := newManagedDynamicVertexBuffer()
	buffer.buffer.vertexCount = 17
	buffer.buffer.bufferUsage = BufferUsageWriteOnly
	if buffer.VertexCount() != 17 || buffer.BufferUsage() != BufferUsageWriteOnly {
		t.Fatalf("the derived buffer reported %d/%v, the base holds 17/WriteOnly",
			buffer.VertexCount(), buffer.BufferUsage())
	}
	// Unnamed, ToString falls back to the OBJECT's type -- the derived one,
	// through the composed base's identity chain.
	if buffer.ToString() != "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer" {
		t.Fatalf("ToString = %q; GraphicsResource::ToString falls back to the object's type", buffer.ToString())
	}
	buffer.SetName("streamed")
	if buffer.Name() != "streamed" || buffer.buffer.Name() != "streamed" {
		t.Fatal("Name is not the base's field")
	}
	if buffer.ToString() != "streamed" {
		t.Fatalf("a named resource's ToString = %q; the reference answers with _name when it is set", buffer.ToString())
	}
	index := newManagedDynamicIndexBuffer()
	index.buffer.indexCount = 9
	index.buffer.indexElementSize = IndexElementSizeThirtyTwoBits
	if index.IndexCount() != 9 || index.IndexElementSize() != IndexElementSizeThirtyTwoBits {
		t.Fatalf("the derived index buffer reported %d/%v", index.IndexCount(), index.IndexElementSize())
	}
	if index.ToString() != "Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer" {
		t.Fatalf("ToString = %q", index.ToString())
	}
	// A zero value answers rather than panicking, on every forwarded member.
	var zero *DynamicVertexBuffer
	if zero.VertexCount() != 0 || zero.Name() != "" || !zero.IsDisposed() || zero.ToString() != "" {
		t.Fatal("a nil DynamicVertexBuffer did not answer its zero values")
	}
	var zeroIndex *DynamicIndexBuffer
	if zeroIndex.IndexCount() != 0 || !zeroIndex.IsDisposed() {
		t.Fatal("a nil DynamicIndexBuffer did not answer its zero values")
	}
}

// TestARefusedUploadLeavesTheLatchAlone pins the half of CopyData's tail that
// its PLACEMENT decides: the clear is after the result check, so an upload CNA
// refuses must leave the content-lost latch exactly as it was.
//
// It needs an upload that gets PAST every managed guard and then fails, which
// no device-free test can produce by accident -- so the buffer is given a
// native resource that is real enough to be reached and wrong enough to refuse:
// a zero interop.Resource, whose kind is not a vertex buffer's, so liveHandle
// refuses it. That is the only failure mode reachable without a device, and it
// is exactly the one this claim needs.
func TestARefusedUploadLeavesTheLatchAlone(t *testing.T) {
	base := &VertexBuffer{
		resource:    &GraphicsResource{resource: &interop.Resource{}},
		declaration: &VertexDeclaration{},
		vertexCount: 4,
		// The stride the transfer arithmetic is measured in. Without it
		// prepareVertexTransfer refuses before the upload and the claim would
		// pass over a defect.
		vertexStride: 4,
	}
	buffer := &DynamicVertexBuffer{buffer: base}
	base.bindDerived(buffer)
	buffer.contentLost = true

	err := DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		buffer, []uint32{1, 2, 3, 4}, 0, 4, SetDataOptionsDiscard)
	if err == nil {
		t.Fatal("an upload through a resource of the wrong kind was accepted; the test's premise is gone")
	}
	if !buffer.contentLost {
		t.Fatal("a REFUSED upload cleared the content-lost latch; CopyData clears only after the result check")
	}
	// The same claim on the index side, whose two branches both end the same
	// way.
	indexBase := &IndexBuffer{
		resource:         &GraphicsResource{resource: &interop.Resource{}},
		indexCount:       4,
		indexElementSize: IndexElementSizeSixteenBits,
	}
	indexBuffer := &DynamicIndexBuffer{buffer: indexBase}
	indexBase.bindDerived(indexBuffer)
	indexBuffer.contentLost = true
	err = DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		indexBuffer, []uint16{1, 2, 3, 4}, 0, 4, SetDataOptionsDiscard)
	if err == nil {
		t.Fatal("an index upload through a resource of the wrong kind was accepted")
	}
	if !indexBuffer.contentLost {
		t.Fatal("a REFUSED index upload cleared the content-lost latch")
	}
}

// TestSetDataOptionsConversionIsTheBitTest pins ConvertXnaSetDataOptionsToDx's
// whole 27-byte body, including the two things the enum does not suggest: bit 0
// wins over bit 1, and an undefined value is mapped rather than refused.
//
// It matters because CNA refuses an undefined option by name, so a projection
// that passed the caller's raw value through would refuse where the reference
// silently accepts -- a divergence in the direction that breaks working code.
func TestSetDataOptionsConversionIsTheBitTest(t *testing.T) {
	for _, row := range []struct {
		options SetDataOptions
		want    uint32
		why     string
	}{
		{SetDataOptionsNone, 0, "None has neither bit"},
		{SetDataOptionsDiscard, 1, "bit 0"},
		{SetDataOptionsNoOverwrite, 2, "bit 1"},
		{SetDataOptionsDiscard | SetDataOptionsNoOverwrite, 1, "bit 0 is tested FIRST and returns, so Discard wins"},
		{4, 0, "an undefined value with neither bit is None, not a refusal"},
		{99, 1, "99 is 0b1100011 and has bit 0, so it is Discard"},
		{6, 2, "6 is 0b110: no bit 0, bit 1 set"},
		{-1, 1, "every bit set, and bit 0 is tested first"},
	} {
		if got := nativeSetDataOptions(row.options); got != row.want {
			t.Fatalf("nativeSetDataOptions(%d) = %d, want %d (%s)", row.options, got, row.want, row.why)
		}
	}
}
