package graphics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// positionColourVertex is a consumer's own vertex type: a struct that
// implements IVertexType, exactly as a C# one does. It is what makes the
// Type-keyed constructor reachable at all, and every FromType test below turns
// on one property of it.
type positionColourVertex struct {
	Position framework.Vector3
	Colour   framework.Color
}

var positionColourDeclaration = mustDeclaration(16, []VertexElement{
	NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
	NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
})

func (positionColourVertex) VertexDeclaration() *VertexDeclaration { return positionColourDeclaration }

func mustDeclaration(stride int32, elements []VertexElement) *VertexDeclaration {
	declaration, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(stride, elements)
	if err != nil {
		panic(err)
	}
	return declaration
}

// notAVertexType is a struct with no VertexDeclaration method.
type notAVertexType struct{ A, B, C, D int32 }

// wrongSizeVertex implements IVertexType and lies: its declaration's stride is
// not its own size.
type wrongSizeVertex struct{ A int32 }

func (wrongSizeVertex) VertexDeclaration() *VertexDeclaration { return positionColourDeclaration }

// nullDeclarationVertex implements IVertexType and returns nil.
type nullDeclarationVertex struct{ A, B, C, D int32 }

func (nullDeclarationVertex) VertexDeclaration() *VertexDeclaration { return nil }

var _ IVertexType = positionColourVertex{}

// TestFromTypeReproducesEachCheckInOrder pins VertexDeclaration::FromType,
// which is `assembly` in the reference and is the whole failure surface of the
// Type-keyed constructor. Each case is one of its four checks, with the
// FrameworkResources sentence its throw site loads and the type named in it.
func TestFromTypeReproducesEachCheckInOrder(t *testing.T) {
	declaration, err := vertexDeclarationFromType(reflect.TypeOf(positionColourVertex{}))
	if err != nil {
		t.Fatalf("a valid vertex type was refused: %v", err)
	}
	if declaration != positionColourDeclaration {
		t.Fatal("FromType did not return the declaration the type publishes")
	}
	for _, testCase := range []struct {
		name    string
		goType  reflect.Type
		message string
	}{
		{"a non-struct", reflect.TypeOf(int32(0)), "is not a value type."},
		{"a struct that is not an IVertexType", reflect.TypeOf(notAVertexType{}), "does not implement the IVertexType interface."},
		{"an IVertexType with a nil declaration", reflect.TypeOf(nullDeclarationVertex{}), "returned a null VertexDeclaration."},
		{"an IVertexType whose size is not its stride", reflect.TypeOf(wrongSizeVertex{}), "does not match the stride of its vertex declaration."},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := vertexDeclarationFromType(testCase.goType)
			if err == nil {
				t.Fatal("the type was accepted")
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("%v, want %q", err, testCase.message)
			}
			if !strings.Contains(err.Error(), testCase.goType.String()) {
				t.Fatalf("%v, want the message to name the type", err)
			}
		})
	}
	if _, err := vertexDeclarationFromType(nil); err == nil {
		t.Fatal("a nil vertex type was accepted")
	}
}

// TestFromTypeSplitsArgumentAndInvalidOperation pins the reference's own split:
// its first two checks throw ArgumentException and its last two throw
// InvalidOperationException. A projection that used one error for all four
// would look right and tell a consumer the wrong kind of thing.
func TestFromTypeSplitsArgumentAndInvalidOperation(t *testing.T) {
	_, notValue := vertexDeclarationFromType(reflect.TypeOf(int32(0)))
	_, notInterface := vertexDeclarationFromType(reflect.TypeOf(notAVertexType{}))
	_, nullDeclaration := vertexDeclarationFromType(reflect.TypeOf(nullDeclarationVertex{}))
	_, wrongSize := vertexDeclarationFromType(reflect.TypeOf(wrongSizeVertex{}))
	if !errors.Is(notValue, errArgument) || !errors.Is(notInterface, errArgument) {
		t.Fatalf("the first two checks are not ArgumentException: %v / %v", notValue, notInterface)
	}
	if !errors.Is(nullDeclaration, errVertexTypeInvalidOperation) || !errors.Is(wrongSize, errVertexTypeInvalidOperation) {
		t.Fatalf("the last two checks are not InvalidOperationException: %v / %v", nullDeclaration, wrongSize)
	}
}

// TestTheConstructorGuardsRunInTheReferencesOrder pins that a null declaration
// is refused BEFORE a bad count, and that the Type-keyed constructor resolves
// the type BEFORE either -- so a caller with both a bad type and a bad count is
// told about the type.
func TestTheConstructorGuardsRunInTheReferencesOrder(t *testing.T) {
	_, err := NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		nil, nil, 0, BufferUsageNone)
	if err == nil || !strings.Contains(err.Error(), "vertexDeclaration") {
		t.Fatalf("a null declaration with a zero count = %v, want the declaration refusal first", err)
	}
	if !strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("%v, want the reference's NullNotAllowed message", err)
	}
	_, err = NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		nil, positionColourDeclaration, 0, BufferUsageNone)
	if err == nil || !strings.Contains(err.Error(), resourcesMustBeGreaterThanZeroSize) {
		t.Fatalf("a zero count = %v, want the reference's size message", err)
	}
	_, err = NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		nil, reflect.TypeOf(notAVertexType{}), 0, BufferUsageNone)
	if err == nil || !strings.Contains(err.Error(), "does not implement the IVertexType interface.") {
		t.Fatalf("a bad type with a bad count = %v, want the type refusal first", err)
	}
}

// TestANilDeviceIsRefusedInGosOwnWordsForVertexBuffers pins the same measured
// divergence IndexBuffer carries: VertexBuffer's constructors have no device
// check either.
func TestANilDeviceIsRefusedInGosOwnWordsForVertexBuffers(t *testing.T) {
	_, err := NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		nil, positionColourDeclaration, 4, BufferUsageNone)
	if !errors.Is(err, errVertexBufferDeviceRequired) {
		t.Fatalf("%v, want the Go-only device refusal", err)
	}
	if strings.Contains(err.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatal("the refusal borrowed Texture2D's message; VertexBuffer's constructors have no such throw site")
	}
}

// corpusVertexBuffer builds a buffer with no native half, which is every guard
// that runs before CNA.
func corpusVertexBuffer(t *testing.T, vertexCount, stride int32, usage BufferUsage) *VertexBuffer {
	t.Helper()
	buffer := &VertexBuffer{
		resource:     newGraphicsResource(nil, nil),
		declaration:  positionColourDeclaration,
		vertexCount:  vertexCount,
		bufferUsage:  usage,
		vertexStride: stride,
	}
	buffer.resource.bindDerived(buffer)
	return buffer
}

// TestEveryVertexTransferGuardRunsBeforeTheDevice covers CopyData's whole guard
// prefix, in its order.
func TestEveryVertexTransferGuardRunsBeforeTheDevice(t *testing.T) {
	buffer := corpusVertexBuffer(t, 4, 16, BufferUsageNone)
	if err := VertexBufferSetDataBySliceOfT(buffer, []positionColourVertex{}); err == nil ||
		!strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("an empty array = %v", err)
	}
	if err := VertexBufferSetDataBySliceOfTAndInt32AndInt32(buffer,
		make([]positionColourVertex, 2), 0, 3); err == nil ||
		!strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("a window off the end = %v", err)
	}
	// Five vertices do not fit a four-vertex buffer.
	if err := VertexBufferSetDataBySliceOfT(buffer, make([]positionColourVertex, 5)); err == nil ||
		!strings.Contains(err.Error(), resourceDataMustBeCorrectSize) {
		t.Fatalf("an oversized transfer = %v", err)
	}
	// A stride SMALLER than the element is the reference's own refusal.
	if err := VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		buffer, 0, make([]positionColourVertex, 1), 0, 1, 8); err == nil ||
		!strings.Contains(err.Error(), vertexStrideTooSmall) {
		t.Fatalf("a stride below sizeof(T) = %v, want the reference's refusal", err)
	}
	if err := VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		buffer, -4, make([]positionColourVertex, 1), 0, 1, 0); err == nil ||
		!strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("a negative offset = %v", err)
	}
}

// TestAStrideLargerThanTheElementIsRefusedWithItsReason is the milestone's one
// recorded narrowing. XNA writes sizeof(T) bytes per element and LEAVES THE
// GAPS ALONE; CNA publishes no route with a separate source and destination
// stride, so composing the padded image here would zero bytes the reference
// preserves. The refusal must say that rather than pretend the stride is
// invalid.
func TestAStrideLargerThanTheElementIsRefusedWithItsReason(t *testing.T) {
	buffer := corpusVertexBuffer(t, 8, 32, BufferUsageNone)
	err := VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		buffer, 0, make([]positionColourVertex, 2), 0, 2, 32)
	if !errors.Is(err, errVertexStrideUnsupported) {
		t.Fatalf("%v, want the recorded upstream refusal", err)
	}
	if strings.Contains(err.Error(), vertexStrideTooSmall) {
		t.Fatal("the refusal claimed the reference's too-small message, which is a different failure")
	}
	// A stride EQUAL to the element size is the tightly-packed case and is
	// accepted by the guard prefix -- it reaches the device, which is absent
	// here, so the error must not be the stride one.
	err = VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		buffer, 0, make([]positionColourVertex, 2), 0, 2, 16)
	if errors.Is(err, errVertexStrideUnsupported) {
		t.Fatal("a stride equal to sizeof(T) was refused; zero padding is the packed case")
	}
}

// TestGetDataRefusesAWriteOnlyVertexBuffer pins the NotSupportedException on
// the read path only.
func TestGetDataRefusesAWriteOnlyVertexBuffer(t *testing.T) {
	writeOnly := corpusVertexBuffer(t, 4, 16, BufferUsageWriteOnly)
	if err := VertexBufferGetDataBySliceOfT(writeOnly, make([]positionColourVertex, 2)); err == nil ||
		!strings.Contains(err.Error(), writeOnlyGetNotSupported) {
		t.Fatalf("GetData on a WriteOnly buffer = %v", err)
	}
	err := VertexBufferSetDataBySliceOfT(writeOnly, make([]positionColourVertex, 2))
	if err != nil && strings.Contains(err.Error(), writeOnlyGetNotSupported) {
		t.Fatal("SetData was refused for WriteOnly; the reference checks it only when getting")
	}
}

// TestADisposedVertexBufferRefusesEveryTransfer pins CheckDisposed as the FIRST
// statement, before the data check.
func TestADisposedVertexBufferRefusesEveryTransfer(t *testing.T) {
	buffer := corpusVertexBuffer(t, 4, 16, BufferUsageNone)
	if err := buffer.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if err := VertexBufferSetDataBySliceOfT(buffer, []positionColourVertex{{}}); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("SetData after Dispose = %v", err)
	}
	if err := VertexBufferSetDataBySliceOfT(buffer, []positionColourVertex{}); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("an empty array on a disposed buffer = %v, want the disposal refusal first", err)
	}
	if buffer.VertexCount() != 4 || buffer.VertexDeclaration() == nil {
		t.Fatal("a disposed buffer stopped answering its properties")
	}
}

// TestTheBufferDoesNotOwnItsDeclaration pins that get_VertexDeclaration hands
// back the CALLER'S object and that disposing the buffer leaves it alive --
// the reference disposes nothing it did not create.
func TestTheBufferDoesNotOwnItsDeclaration(t *testing.T) {
	declaration := mustDeclaration(16, []VertexElement{
		NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
	})
	buffer := &VertexBuffer{
		resource:     newGraphicsResource(nil, nil),
		declaration:  declaration,
		vertexCount:  4,
		vertexStride: 16,
	}
	buffer.resource.bindDerived(buffer)
	if buffer.VertexDeclaration() != declaration {
		t.Fatal("VertexDeclaration did not answer the object the constructor was given")
	}
	if err := buffer.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if declaration.IsDisposed() {
		t.Fatal("disposing the buffer disposed its declaration; the buffer does not own it")
	}
	if declaration.VertexStride() != 16 {
		t.Fatal("the declaration stopped answering after the buffer was disposed")
	}
}

// TestTheProjectionRecordsTheStrideCNAApplied pins that the fit check is
// measured against CNA's reported stride rather than the declaration's own. A
// renderer that padded a vertex would otherwise be invisible, and every
// transfer would be measured against a size the buffer does not have.
func TestTheProjectionRecordsTheStrideCNAApplied(t *testing.T) {
	padded := newVertexBuffer(nil, nil, positionColourDeclaration, interop.VertexBufferInfo{
		VertexCount:  4,
		VertexStride: 32,
	})
	if padded.bufferByteSize() != 128 {
		t.Fatalf("buffer size = %d, want 4 vertices of the 32 bytes CNA applied", padded.bufferByteSize())
	}
	// Eight 16-byte vertices fit 128 bytes, and would not fit the
	// declaration's own 4x16.
	if err := VertexBufferSetDataBySliceOfT(padded, make([]positionColourVertex, 8)); err != nil &&
		strings.Contains(err.Error(), resourceDataMustBeCorrectSize) {
		t.Fatal("the fit check used the declaration's stride rather than CNA's")
	}
	if got := padded.ToString(); got != "Microsoft.Xna.Framework.Graphics.VertexBuffer" {
		t.Fatalf("ToString = %q; the CLR `this` must reach the outermost object", got)
	}
	if padded.BufferUsage() != BufferUsageNone {
		t.Fatalf("BufferUsage = %d", padded.BufferUsage())
	}
	applied := newVertexBuffer(nil, nil, positionColourDeclaration, interop.VertexBufferInfo{
		VertexCount: 2, VertexStride: 16, BufferUsage: 1,
	})
	if applied.BufferUsage() != BufferUsageWriteOnly {
		t.Fatal("the usage CNA applied was ignored")
	}
}

// TestAPartialVertexTransferIsRefusedWithItsReason pins the second measured
// narrowing. CNA describes a raw transfer in WHOLE vertices of the buffer's own
// stride and refuses any other stride by name, so a byte count that is not a
// multiple of it -- or an offset that is not -- has no expression in this ABI.
// XNA's is a byte memcpy and has no such rule, so the refusal must say which
// side imposes it.
func TestAPartialVertexTransferIsRefusedWithItsReason(t *testing.T) {
	// A 32-byte-stride buffer of two vertices: 64 bytes.
	padded := corpusVertexBuffer(t, 2, 32, BufferUsageNone)
	// Four 16-byte values are 64 bytes -- two whole vertices -- and are fine.
	if err := VertexBufferSetDataBySliceOfT(padded, make([]positionColourVertex, 4)); errors.Is(err, errVertexPartialVertexUnsupported) {
		t.Fatalf("a whole number of vertices was refused: %v", err)
	}
	// Three are 48 bytes, which is one and a half.
	err := VertexBufferSetDataBySliceOfT(padded, make([]positionColourVertex, 3))
	if !errors.Is(err, errVertexPartialVertexUnsupported) {
		t.Fatalf("%v, want the whole-vertex refusal", err)
	}
	// And an offset that lands part-way into a vertex.
	err = VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		padded, 16, make([]positionColourVertex, 2), 0, 2, 0)
	if !errors.Is(err, errVertexPartialVertexUnsupported) {
		t.Fatalf("a mid-vertex offset = %v, want the whole-vertex refusal", err)
	}
}

// TestTheDeclarationHandleIsCreatedOnceAndCarriesTheStoredStride pins the lazy
// native handle and the reason it passes the stride EXPLICITLY: CNA's
// stride-less route would recompute one from the elements, silently replacing
// an explicit stride the two-argument constructor was given.
func TestTheDeclarationHandleIsCreatedOnceAndCarriesTheStoredStride(t *testing.T) {
	declaration := mustDeclaration(32, []VertexElement{
		NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
	})
	// The stored stride is the caller's 32, not the 16 the elements imply --
	// which is what makes the recomputing route wrong for this declaration.
	if declaration.VertexStride() != 32 {
		t.Fatalf("VertexStride = %d, want the explicit 32", declaration.VertexStride())
	}
	if vertexStrideForElements(declaration.GetVertexElements()) != 16 {
		t.Fatal("the computed stride is not 16; this control needs the two to differ")
	}
	// No handle until something needs one.
	if declaration.native != nil {
		t.Fatal("the constructor created a CNA declaration; it is deferred to the first consumer")
	}
	// With no runtime there is nothing to create one from, and that is a
	// refusal rather than a silent nil.
	if _, err := declaration.nativeDeclaration(nil); err == nil {
		t.Fatal("a declaration handle was created with no runtime")
	}
}

// TestAZeroVertexBufferIsRefusedRatherThanPanicking covers the Go-only guard.
func TestAZeroVertexBufferIsRefusedRatherThanPanicking(t *testing.T) {
	var absent *VertexBuffer
	if absent.VertexCount() != 0 || absent.VertexDeclaration() != nil || !absent.IsDisposed() {
		t.Fatal("a nil buffer answered with values")
	}
	if err := absent.DisposeByNone(); !errors.Is(err, errVertexBufferNil) {
		t.Fatalf("DisposeByNone on nil = %v", err)
	}
	if err := VertexBufferSetDataBySliceOfT(absent, []positionColourVertex{{}}); !errors.Is(err, errVertexBufferNil) {
		t.Fatalf("SetData on nil = %v", err)
	}
}
