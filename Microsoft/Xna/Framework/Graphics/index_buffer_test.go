package graphics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// Every test below reaches the guards that run BEFORE the device, which is the
// whole managed half of this type: creation itself needs a callback-scoped CNA
// device, and the native half is proved in tools/native_stress.

// TestBothConstructorsRefuseANonPositiveCount pins the check the reference
// performs FIRST, before it stores the device:
//
//	if (indexCount <= 0)
//	    throw new ArgumentOutOfRangeException("indexCount",
//	        FrameworkResources.ResourcesMustBeGreaterThanZeroSize);
//
// The order is observable: a bad count is refused with a nil device too.
func TestBothConstructorsRefuseANonPositiveCount(t *testing.T) {
	for _, count := range []int32{0, -1} {
		_, err := NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
			nil, IndexElementSizeSixteenBits, count, BufferUsageNone)
		if err == nil {
			t.Fatalf("count %d was accepted", count)
		}
		if !strings.Contains(err.Error(), resourcesMustBeGreaterThanZeroSize) {
			t.Fatalf("count %d = %v, want the reference's message", count, err)
		}
		if !strings.Contains(err.Error(), "indexCount") {
			t.Fatalf("count %d = %v, want the reference's parameter name", count, err)
		}
		_, err = NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
			nil, reflect.TypeOf(uint16(0)), count, BufferUsageNone)
		if err == nil || !strings.Contains(err.Error(), resourcesMustBeGreaterThanZeroSize) {
			t.Fatalf("the Type constructor accepted count %d: %v", count, err)
		}
	}
}

// TestTheTypeConstructorClosesTheElementSet pins the one place CNA-Go is
// NARROWER than the reference, and the reason. `Marshal.SizeOf` accepts any
// blittable type; CNA stores 16-bit or 32-bit indices and nothing else, so a
// type outside the four is refused BY NAME rather than producing a buffer whose
// width XNA's own IndexElementSize cannot name.
func TestTheTypeConstructorClosesTheElementSet(t *testing.T) {
	for _, accepted := range []struct {
		goType reflect.Type
		want   IndexElementSize
	}{
		{reflect.TypeOf(int16(0)), IndexElementSizeSixteenBits},
		{reflect.TypeOf(uint16(0)), IndexElementSizeSixteenBits},
		{reflect.TypeOf(int32(0)), IndexElementSizeThirtyTwoBits},
		{reflect.TypeOf(uint32(0)), IndexElementSizeThirtyTwoBits},
	} {
		got, err := indexElementSizeForType(accepted.goType)
		if err != nil {
			t.Fatalf("%s was refused: %v", accepted.goType, err)
		}
		if got != accepted.want {
			t.Fatalf("%s mapped to %d, want %d", accepted.goType, got, accepted.want)
		}
	}
	for _, refused := range []reflect.Type{
		reflect.TypeOf(int64(0)), reflect.TypeOf(float32(0)), reflect.TypeOf(byte(0)),
		reflect.TypeOf(struct{ A, B uint16 }{}), reflect.TypeOf(""),
	} {
		if _, err := indexElementSizeForType(refused); err == nil {
			t.Fatalf("%s was accepted as an index element type", refused)
		} else if !errors.Is(err, errUnsupportedIndexType) {
			t.Fatalf("%s = %v, want the unsupported-type refusal", refused, err)
		} else if !strings.Contains(err.Error(), refused.String()) {
			t.Fatalf("%s = %v, want the refusal to name the type", refused, err)
		}
	}
	if _, err := indexElementSizeForType(nil); err == nil {
		t.Fatal("a nil index type was accepted")
	}
}

// TestANilDeviceIsRefusedInGosOwnWords pins the measured divergence. The
// reference has NO device check here -- it stores the argument and dereferences
// it two statements later, so C# gets a NullReferenceException. Go cannot
// project that, and this refusal must NOT borrow the message Texture2D's
// constructor really does throw.
func TestANilDeviceIsRefusedInGosOwnWords(t *testing.T) {
	_, err := NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		nil, IndexElementSizeSixteenBits, 16, BufferUsageNone)
	if err == nil {
		t.Fatal("a nil device was accepted")
	}
	if !errors.Is(err, errIndexBufferDeviceRequired) {
		t.Fatalf("%v, want the Go-only device refusal", err)
	}
	if strings.Contains(err.Error(), deviceCannotBeNullOnResourceCreate) {
		t.Fatal("the refusal borrowed Texture2D's DeviceCannotBeNullOnResourceCreate; IndexBuffer's constructors have no such throw site")
	}
}

// TestTheElementWidthCheckIsByRepresentationNotBySize is the same rule the
// texture transfer carries, on the other side of the boundary: CNA identifies
// an index by its WIDTH, so a T whose layout is not that width would stride
// wrong and nothing would report it.
func TestTheElementWidthCheckIsByRepresentationNotBySize(t *testing.T) {
	if width, size, err := indexElementWidth[uint16](); err != nil || width != 2 || size != IndexElementSizeSixteenBits {
		t.Fatalf("uint16 = %d,%d,%v", width, size, err)
	}
	if width, size, err := indexElementWidth[int32](); err != nil || width != 4 || size != IndexElementSizeThirtyTwoBits {
		t.Fatalf("int32 = %d,%d,%v", width, size, err)
	}
	if _, _, err := indexElementWidth[uint64](); !errors.Is(err, errUnsupportedIndexElement) {
		t.Fatalf("uint64 = %v, want the unsupported-element refusal", err)
	}
	if _, _, err := indexElementWidth[struct{ A uint16 }](); !errors.Is(err, errUnsupportedIndexElement) {
		t.Fatalf("a struct = %v, want the unsupported-element refusal", err)
	}
}

// TestValidateCopyParametersReproducesTheHelpersOrderAndNames pins
// Helpers::ValidateCopyParameters, which every buffer transfer in the reference
// shares. The three throws carry three different parameter names and one
// sentence, and the ORDER decides which a caller is told about.
func TestValidateCopyParametersReproducesTheHelpersOrderAndNames(t *testing.T) {
	if err := validateCopyParameters(4, 0, 4); err != nil {
		t.Fatalf("a whole-array window was refused: %v", err)
	}
	if err := validateCopyParameters(4, 1, 3); err != nil {
		t.Fatalf("a valid window was refused: %v", err)
	}
	for _, testCase := range []struct {
		name                            string
		length, dataIndex, elementCount int32
		parameter                       string
	}{
		{"a negative index", 4, -1, 1, "dataIndex"},
		{"an index past the end", 4, 5, 1, "dataIndex"},
		{"a window running off the end", 4, 2, 3, "elementCount"},
		{"a zero count", 4, 0, 0, "elementCount"},
		{"a negative count", 4, 0, -1, "elementCount"},
		// dataIndex == dataLength passes the FIRST check and is caught by the
		// second, because the count is required positive by the third.
		{"an empty window at the end", 4, 4, 1, "elementCount"},
	} {
		err := validateCopyParameters(testCase.length, testCase.dataIndex, testCase.elementCount)
		if err == nil {
			t.Fatalf("%s was accepted", testCase.name)
		}
		if !strings.Contains(err.Error(), testCase.parameter) {
			t.Fatalf("%s = %v, want the parameter named %q", testCase.name, err, testCase.parameter)
		}
		if !strings.Contains(err.Error(), mustBeValidIndex) {
			t.Fatalf("%s = %v, want the reference's message", testCase.name, err)
		}
	}
}

// TestEveryTransferGuardRunsBeforeTheDevice covers the whole CopyData prefix on
// a buffer with no native half. Each of these must be refused by the
// PROJECTION, because a transfer that reached CNA with a bad window would be
// reported in CNA's words rather than XNA's.
func TestEveryTransferGuardRunsBeforeTheDevice(t *testing.T) {
	buffer := &IndexBuffer{
		resource:         newGraphicsResource(nil, nil),
		indexCount:       8,
		indexElementSize: IndexElementSizeSixteenBits,
		bufferUsage:      BufferUsageNone,
	}
	buffer.resource.bindDerived(buffer)

	if err := IndexBufferSetDataBySliceOfT(buffer, []uint16{}); err == nil ||
		!strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("an empty array = %v, want the reference's data refusal", err)
	}
	if err := IndexBufferSetDataBySliceOfTAndInt32AndInt32(buffer, []uint16{1, 2}, 0, 3); err == nil ||
		!strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("a window off the end = %v", err)
	}
	if err := IndexBufferSetDataBySliceOfT(buffer, []uint64{1, 2}); !errors.Is(err, errUnsupportedIndexElement) {
		t.Fatalf("an unsupported element = %v", err)
	}
	// Nine 16-bit indices do not fit an eight-index buffer, which the reference
	// measures against the width it CREATED the buffer with.
	tooMany := make([]uint16, 9)
	if err := IndexBufferSetDataBySliceOfT(buffer, tooMany); err == nil ||
		!strings.Contains(err.Error(), resourceDataMustBeCorrectSize) {
		t.Fatalf("an oversized transfer = %v, want the reference's size refusal", err)
	}
	// And so do eight 32-bit ones, for the same reason: the buffer is sixteen
	// bytes, not eight elements of whatever T is.
	if err := IndexBufferSetDataBySliceOfT(buffer, make([]uint32, 8)); err == nil ||
		!strings.Contains(err.Error(), resourceDataMustBeCorrectSize) {
		t.Fatalf("a 32-bit transfer into a 16-bit buffer = %v", err)
	}
	if err := IndexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32(buffer, -4, []uint16{1}, 0, 1); err == nil ||
		!strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("a negative byte offset = %v", err)
	}
}

// TestGetDataRefusesAWriteOnlyBuffer pins the NotSupportedException the
// reference throws only on the READ path, and only for BufferUsage.WriteOnly.
func TestGetDataRefusesAWriteOnlyBuffer(t *testing.T) {
	writeOnly := &IndexBuffer{
		resource:         newGraphicsResource(nil, nil),
		indexCount:       8,
		indexElementSize: IndexElementSizeSixteenBits,
		bufferUsage:      BufferUsageWriteOnly,
	}
	writeOnly.resource.bindDerived(writeOnly)
	if err := IndexBufferGetDataBySliceOfT(writeOnly, make([]uint16, 4)); err == nil ||
		!strings.Contains(err.Error(), writeOnlyGetNotSupported) {
		t.Fatalf("GetData on a WriteOnly buffer = %v, want the reference's refusal", err)
	}
	// SetData on the same buffer passes the guard prefix and is refused only
	// because there is no native half here -- which is the point: the
	// WriteOnly check is on the read path alone.
	err := IndexBufferSetDataBySliceOfT(writeOnly, make([]uint16, 4))
	if err != nil && strings.Contains(err.Error(), writeOnlyGetNotSupported) {
		t.Fatal("SetData was refused for WriteOnly; the reference checks it only when getting")
	}
}

// TestADisposedBufferRefusesEveryTransfer pins Helpers.CheckDisposed, which is
// CopyData's FIRST statement -- before the data null check.
func TestADisposedBufferRefusesEveryTransfer(t *testing.T) {
	buffer := &IndexBuffer{
		resource:         newGraphicsResource(nil, nil),
		indexCount:       8,
		indexElementSize: IndexElementSizeSixteenBits,
	}
	buffer.resource.bindDerived(buffer)
	if err := buffer.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if err := IndexBufferSetDataBySliceOfT(buffer, []uint16{1}); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("SetData after Dispose = %v, want ObjectDisposedException", err)
	}
	if err := IndexBufferGetDataBySliceOfT(buffer, make([]uint16, 1)); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("GetData after Dispose = %v", err)
	}
	// A disposed buffer STILL answers its three properties, because none of the
	// reference's getters checks disposal.
	if buffer.IndexCount() != 8 || buffer.IndexElementSize() != IndexElementSizeSixteenBits {
		t.Fatal("a disposed buffer stopped answering its properties")
	}
	// The empty-array refusal comes AFTER the disposal check, which is the
	// order CopyData has.
	if err := IndexBufferSetDataBySliceOfT(buffer, []uint16{}); !errors.Is(err, interop.ErrDisposed) {
		t.Fatalf("an empty array on a disposed buffer = %v, want the disposal refusal first", err)
	}
}

// TestTheEnumMappingsAreExplicitAndTotal pins the two CNA correspondences.
// Their numberings agree with XNA's today, and a cast would make the agreement
// invisible; these are the one place it is stated.
func TestTheEnumMappingsAreExplicitAndTotal(t *testing.T) {
	if nativeIndexElementSize(IndexElementSizeSixteenBits) != interop.IndexElementSizeSixteenBits ||
		nativeIndexElementSize(IndexElementSizeThirtyTwoBits) != interop.IndexElementSizeThirtyTwoBits {
		t.Fatal("the index element size mapping disagrees with CNA's identities")
	}
	// An out-of-range literal narrows to sixteen bits here and WIDENS to
	// thirty-two in the reference, whose test is `== SixteenBits ? 2 : 4`.
	// The difference is deliberate: CNA takes an identity rather than a width,
	// and inventing ThirtyTwoBits for a value that is neither would tell CNA
	// something the caller never said.
	if nativeIndexElementSize(IndexElementSize(7)) != interop.IndexElementSizeSixteenBits {
		t.Fatal("an unknown IndexElementSize did not take the sixteen-bit branch")
	}
	if nativeBufferUsage(BufferUsageNone) != 0 || nativeBufferUsage(BufferUsageWriteOnly) != 1 {
		t.Fatal("the buffer usage mapping disagrees with CNA's identities")
	}
	if indexElementWidthFor(IndexElementSizeSixteenBits) != 2 || indexElementWidthFor(IndexElementSizeThirtyTwoBits) != 4 {
		t.Fatal("the stored element width is wrong")
	}
}

// TestTheComposedBaseAnswersWithThisTypesIdentity pins the CLR `this` the
// constructor installs. Without it GraphicsResource::ToString would fall back
// to the BASE's runtime name, which is the exact failure composition invites.
func TestTheComposedBaseAnswersWithThisTypesIdentity(t *testing.T) {
	// Through the REAL builder, which is what installs the `this`.
	buffer := newIndexBuffer(nil, nil, interop.IndexBufferInfo{IndexCount: 4})
	if got := buffer.ToString(); got != "Microsoft.Xna.Framework.Graphics.IndexBuffer" {
		t.Fatalf("ToString = %q, want this type's runtime CLR name", got)
	}
	// And the Disposing sender is the buffer, not the composed base.
	var sender any
	if _, err := buffer.AddDisposingHandler(func(s any, _ *framework.EventArgs) error {
		sender = s
		return nil
	}); err != nil {
		t.Fatalf("AddDisposingHandler: %v", err)
	}
	if err := buffer.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if sender != any(buffer) {
		t.Fatal("the Disposing sender is not the buffer")
	}
}

// TestTheProjectionRecordsWhatCNAApplied pins that the three properties come
// from CNA's own report rather than from the request. The reference reads its
// own fields back the same way -- `_indexSize` is what CreateBuffer stored --
// and a renderer that widened an index would otherwise be invisible.
func TestTheProjectionRecordsWhatCNAApplied(t *testing.T) {
	applied := newIndexBuffer(nil, nil, interop.IndexBufferInfo{
		IndexCount:       12,
		IndexElementSize: interop.IndexElementSizeThirtyTwoBits,
		BufferUsage:      1,
	})
	if applied.IndexCount() != 12 {
		t.Fatalf("IndexCount = %d, want CNA's 12", applied.IndexCount())
	}
	if applied.IndexElementSize() != IndexElementSizeThirtyTwoBits {
		t.Fatalf("IndexElementSize = %d, want the width CNA applied", applied.IndexElementSize())
	}
	if applied.BufferUsage() != BufferUsageWriteOnly {
		t.Fatalf("BufferUsage = %d, want the usage CNA applied", applied.BufferUsage())
	}
	narrow := newIndexBuffer(nil, nil, interop.IndexBufferInfo{
		IndexCount:       12,
		IndexElementSize: interop.IndexElementSizeSixteenBits,
	})
	if narrow.IndexElementSize() != IndexElementSizeSixteenBits || narrow.BufferUsage() != BufferUsageNone {
		t.Fatalf("the sixteen-bit report produced %d/%d", narrow.IndexElementSize(), narrow.BufferUsage())
	}
}

// TestAWidenedTransferElementIsRefusedByTheSizeCheck is the falsification of
// the width rule, and it needs BOTH halves of the mutation to be interesting:
// widening the accepted case list alone is absorbed by the size check, which is
// the check doing its job. Widening the list AND removing the size check would
// hand CNA an eight-byte element under a two-byte identity, so the test asserts
// the outcome rather than the mechanism.
func TestAWidenedTransferElementIsRefusedByTheSizeCheck(t *testing.T) {
	buffer := &IndexBuffer{
		resource:         newGraphicsResource(nil, nil),
		indexCount:       8,
		indexElementSize: IndexElementSizeSixteenBits,
	}
	buffer.resource.bindDerived(buffer)
	if err := IndexBufferSetDataBySliceOfT(buffer, make([]uint64, 4)); err == nil {
		t.Fatal("an eight-byte element was accepted for a sixteen-bit buffer")
	}
	// And the width a supported element reports must be the width its identity
	// MEANS, not the Go type's size by coincidence: a thirty-two-bit identity
	// paired with a two-byte width would stride half as far through CNA's
	// buffer and the transfer would still succeed.
	width, size, err := indexElementWidth[uint32]()
	if err != nil {
		t.Fatalf("uint32: %v", err)
	}
	if size != IndexElementSizeThirtyTwoBits || width != 4 {
		t.Fatalf("uint32 reported width %d for identity %d", width, size)
	}
	if width != uint32(indexElementWidthFor(size)) {
		t.Fatalf("the transfer width %d and the identity's width %d disagree", width, indexElementWidthFor(size))
	}
}

// TestAZeroIndexBufferIsRefusedRatherThanPanicking covers the Go-only guard.
func TestAZeroIndexBufferIsRefusedRatherThanPanicking(t *testing.T) {
	var absent *IndexBuffer
	if absent.IndexCount() != 0 || !absent.IsDisposed() {
		t.Fatal("a nil buffer answered with values")
	}
	if err := absent.DisposeByNone(); !errors.Is(err, errIndexBufferNil) {
		t.Fatalf("DisposeByNone on nil = %v", err)
	}
	if err := IndexBufferSetDataBySliceOfT(absent, []uint16{1}); !errors.Is(err, errIndexBufferNil) {
		t.Fatalf("SetData on nil = %v", err)
	}
}
