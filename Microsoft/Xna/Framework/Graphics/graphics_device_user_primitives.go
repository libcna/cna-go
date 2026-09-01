package graphics

import (
	"fmt"
	"reflect"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 73 — the six DrawUser* generic methods.
// ---------------------------------------------------------------------------
//
// # Package functions, on the settled generic-method rule
//
// Go methods cannot declare type parameters, so a CLR generic instance method
// projects as a package-level function whose first parameter is the receiver.
// These six are the profile's largest single group under that rule, and the
// suffix names the type parameter the member declares: `SliceOfT`.
//
// # Two families, four shapes, one native pair
//
//	DrawUserPrimitives<T>(type, T[], vertexOffset, primitiveCount)
//	DrawUserPrimitives<T>(type, T[], vertexOffset, primitiveCount, VertexDeclaration)
//	DrawUserIndexedPrimitives<T>(type, T[], vertexOffset, numVertices,
//	                             Int16[]|Int32[], indexOffset, primitiveCount)
//	DrawUserIndexedPrimitives<T>(... , VertexDeclaration)
//
// The declaration-less overloads are eleven bytes of IL that call the
// declaration-bearing one with
//
//	VertexDeclarationFactory<T>.VertexDeclaration
//
// which is the same cached declaration `VertexDeclaration.FromType(typeof(T))`
// answers. So they are argument normalisers and the projection treats them as
// exactly that. Their CLR constraint is `valuetype .ctor IVertexType`; the
// declaration-bearing pair's is only `valuetype .ctor`, which is why only the
// short pair can resolve a declaration from T at all.
//
// # CNA takes a RAW stream and a declaration
//
// `CNA_UserPrimitives` names five vertex sources: a raw byte stream at the
// declared stride, and four TYPED arrays -- CNA_VertexPositionColor and its
// three siblings. CNA-Go projects none of those four value types, and it always
// has a declaration -- the caller's or T's -- so every call here uses
// CNA_USER_VERTEX_SOURCE_RAW_STREAM with an explicit declaration, which
// expresses every layout the four typed sources do and more.

// The two FrameworkResources strings these six throw that no other member here
// already claims.
const (
	offsetNotValid = "The offset must be within the valid range for this resource."
)

// GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32 is
// GraphicsDevice::DrawUserPrimitives<T>(PrimitiveType, T[], Int32, Int32).
//
// It resolves T's declaration and forwards, which is the whole of its
// seventeen-byte body.
func GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, primitiveCount int32,
) error {
	declaration, err := userVertexDeclaration[T]()
	if err != nil {
		return err
	}
	return GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
		device, primitiveType, vertexData, vertexOffset, primitiveCount, declaration)
}

// GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration is
// GraphicsDevice::DrawUserPrimitives<T>(PrimitiveType, T[], Int32, Int32,
// VertexDeclaration) -- the overload the short one funnels into, and the one
// that reaches CNA.
//
// Its guards, in the reference's order:
//
//	Helpers.CheckDisposed(this, pComPtr);
//	if (vertexData == null)        ArgumentNullException("vertexData", NullNotAllowed)
//	if (vertexDeclaration == null) ArgumentNullException("vertexDeclaration", NullNotAllowed)
//	if (primitiveCount <= 0)       ArgumentOutOfRangeException("primitiveCount", MustDrawSomething)
//	if (primitiveCount > profile max)                      NotSupportedException  -- NOT reproduced
//	if (vertexOffset + vertexCount > vertexData.Length)
//	                               ArgumentOutOfRangeException("primitiveCount", MustBeValidIndex)
//	if (vertexOffset < 0 || vertexOffset >= vertexData.Length)
//	                               ArgumentOutOfRangeException("vertexOffset", OffsetNotValid)
//
// The profile cap is not reproduced for the reason DrawPrimitives' is not:
// ProfileCapabilities is not a public XNA type and CNA-Go projects no part of
// it, so there is no measured maximum to compare against and CNA refuses a
// count its backend cannot draw in its own words.
func GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, primitiveCount int32, vertexDeclaration *VertexDeclaration,
) error {
	native, declaration, err := prepareUserPrimitives(
		device, primitiveType, len(vertexData), vertexOffset, primitiveCount, vertexDeclaration)
	if err != nil {
		return err
	}
	return native.DrawUserPrimitives(uint32(primitiveType), sliceStart(vertexData),
		declaration, vertexOffset, primitiveCount)
}

// GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32 is
// GraphicsDevice::DrawUserIndexedPrimitives<T>(PrimitiveType, T[], Int32,
// Int32, Int16[], Int32, Int32).
func GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, numVertices int32, indexData []int16, indexOffset, primitiveCount int32,
) error {
	declaration, err := userVertexDeclaration[T]()
	if err != nil {
		return err
	}
	return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32AndVertexDeclaration(
		device, primitiveType, vertexData, vertexOffset, numVertices,
		indexData, indexOffset, primitiveCount, declaration)
}

// GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32 is the 32-bit index twin.
func GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, numVertices int32, indexData []int32, indexOffset, primitiveCount int32,
) error {
	declaration, err := userVertexDeclaration[T]()
	if err != nil {
		return err
	}
	return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32AndVertexDeclaration(
		device, primitiveType, vertexData, vertexOffset, numVertices,
		indexData, indexOffset, primitiveCount, declaration)
}

// GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32AndVertexDeclaration is
// GraphicsDevice::DrawUserIndexedPrimitives<T>(PrimitiveType, T[], Int32,
// Int32, Int16[], Int32, Int32, VertexDeclaration).
//
// Its guards add the reference's own index checks to the vertex ones:
//
//	if (indexData == null)  ArgumentNullException("indexData", NullNotAllowed)
//	if (numVertices <= 0)   ArgumentOutOfRangeException("numVertices",
//	                            ResourcesMustBeGreaterThanZero)
//
// and the index-window check, whose message is MustBeValidIndex on
// "primitiveCount" -- the same parameter the vertex-window check names, because
// both are about how much the primitive count asks for.
func GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32AndVertexDeclaration[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, numVertices int32, indexData []int16, indexOffset, primitiveCount int32,
	vertexDeclaration *VertexDeclaration,
) error {
	native, declaration, err := prepareUserIndexedPrimitives(
		device, primitiveType, len(vertexData), vertexOffset, numVertices,
		len(indexData), indexOffset, primitiveCount, vertexDeclaration)
	if err != nil {
		return err
	}
	return native.DrawUserIndexedPrimitives(uint32(primitiveType), sliceStart(vertexData),
		declaration, vertexOffset, numVertices, primitiveCount,
		interop.IndexElementSizeSixteenBits, indexOffset, sliceStart(indexData))
}

// GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32AndVertexDeclaration is the
// 32-bit index twin, and the ONLY difference is the element identity CNA is
// given: the reference's two overloads differ by one `ldc.i4` selecting
// D3DFMT_INDEX16 or D3DFMT_INDEX32.
func GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32AndVertexDeclaration[T any](
	device *GraphicsDevice, primitiveType PrimitiveType, vertexData []T,
	vertexOffset, numVertices int32, indexData []int32, indexOffset, primitiveCount int32,
	vertexDeclaration *VertexDeclaration,
) error {
	native, declaration, err := prepareUserIndexedPrimitives(
		device, primitiveType, len(vertexData), vertexOffset, numVertices,
		len(indexData), indexOffset, primitiveCount, vertexDeclaration)
	if err != nil {
		return err
	}
	return native.DrawUserIndexedPrimitives(uint32(primitiveType), sliceStart(vertexData),
		declaration, vertexOffset, numVertices, primitiveCount,
		interop.IndexElementSizeThirtyTwoBits, indexOffset, sliceStart(indexData))
}

// userVertexDeclaration is `VertexDeclarationFactory<T>.VertexDeclaration`, the
// cached declaration the four short overloads resolve from T.
//
// It is VertexDeclaration.FromType's own resolution, which Foundation 66
// projected: T must be a value type that implements IVertexType and whose size
// matches its declaration's stride. A T that fails any of those gets the
// reference's exact message.
func userVertexDeclaration[T any]() (*VertexDeclaration, error) {
	return vertexDeclarationFromType(reflect.TypeOf(*new(T)))
}

// prepareUserPrimitives is the guard prefix and the native handles both
// non-indexed overloads share.
//
// `device.live()` comes FIRST because the reference's first statement is
// `Helpers.CheckDisposed(this, pComPtr)`. It is a wider check here than there:
// the CLR cannot have a null `this`, so CNA-Go's version also reports a device
// that never had a native half, and a consumer with no device sees that rather
// than an argument message. The argument guards themselves are in
// verifyUserPrimitives, which is where their ORDER is pinned.
func prepareUserPrimitives(
	device *GraphicsDevice, primitiveType PrimitiveType, vertexCount int,
	vertexOffset, primitiveCount int32, vertexDeclaration *VertexDeclaration,
) (*interop.Device, *interop.Resource, error) {
	native, err := device.live()
	if err != nil {
		return nil, nil, err
	}
	if err := verifyUserPrimitives(primitiveType, vertexCount, vertexOffset, primitiveCount, vertexDeclaration); err != nil {
		return nil, nil, err
	}
	declaration, err := vertexDeclaration.nativeDeclaration(native.Runtime())
	if err != nil {
		return nil, nil, err
	}
	return native, declaration, nil
}

// verifyUserPrimitives is the reference's argument guards for the non-indexed
// family, in the reference's order.
func verifyUserPrimitives(
	primitiveType PrimitiveType, vertexCount int,
	vertexOffset, primitiveCount int32, vertexDeclaration *VertexDeclaration,
) error {
	if vertexCount == 0 {
		// A nil or empty Go slice is the reference's null array: it has no
		// element to point at, so nothing can be read from it.
		return fmt.Errorf("%w: vertexData: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if vertexDeclaration == nil {
		return fmt.Errorf("%w: vertexDeclaration: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if primitiveCount <= 0 {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustDrawSomething)
	}
	needed, err := verticesForPrimitives(primitiveType, primitiveCount)
	if err != nil {
		return err
	}
	if int64(vertexOffset)+int64(needed) > int64(vertexCount) {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if vertexOffset < 0 || int(vertexOffset) >= vertexCount {
		return fmt.Errorf("%w: vertexOffset: %s", errArgumentOutOfRange, offsetNotValid)
	}
	return nil
}

// prepareUserIndexedPrimitives adds the index guards to the vertex ones, in the
// reference's order.
func prepareUserIndexedPrimitives(
	device *GraphicsDevice, primitiveType PrimitiveType, vertexCount int,
	vertexOffset, numVertices int32, indexCount int, indexOffset, primitiveCount int32,
	vertexDeclaration *VertexDeclaration,
) (*interop.Device, *interop.Resource, error) {
	native, err := device.live()
	if err != nil {
		return nil, nil, err
	}
	if err := verifyUserIndexedPrimitives(primitiveType, vertexCount, vertexOffset, numVertices,
		indexCount, indexOffset, primitiveCount, vertexDeclaration); err != nil {
		return nil, nil, err
	}
	declaration, err := vertexDeclaration.nativeDeclaration(native.Runtime())
	if err != nil {
		return nil, nil, err
	}
	return native, declaration, nil
}

// verifyUserIndexedPrimitives adds the reference's index guards to the vertex
// ones, in the reference's order.
func verifyUserIndexedPrimitives(
	primitiveType PrimitiveType, vertexCount int,
	vertexOffset, numVertices int32, indexCount int, indexOffset, primitiveCount int32,
	vertexDeclaration *VertexDeclaration,
) error {
	if vertexCount == 0 {
		return fmt.Errorf("%w: vertexData: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if indexCount == 0 {
		return fmt.Errorf("%w: indexData: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if vertexDeclaration == nil {
		return fmt.Errorf("%w: vertexDeclaration: %s", errGraphicsResourceArgumentNull, nullNotAllowed)
	}
	if numVertices <= 0 {
		return fmt.Errorf("%w: numVertices: %s", errArgumentOutOfRange, resourcesMustBeGreaterThanZeroSize)
	}
	if primitiveCount <= 0 {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustDrawSomething)
	}
	needed, err := verticesForPrimitives(primitiveType, primitiveCount)
	if err != nil {
		return err
	}
	if int64(indexOffset)+int64(needed) > int64(indexCount) {
		return fmt.Errorf("%w: primitiveCount: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if vertexOffset < 0 || int(vertexOffset) >= vertexCount {
		return fmt.Errorf("%w: vertexOffset: %s", errArgumentOutOfRange, offsetNotValid)
	}
	if int64(vertexOffset)+int64(numVertices) > int64(vertexCount) {
		return fmt.Errorf("%w: numVertices: %s", errArgumentOutOfRange, mustBeValidIndex)
	}
	if indexOffset < 0 || int(indexOffset) >= indexCount {
		return fmt.Errorf("%w: indexOffset: %s", errArgumentOutOfRange, offsetNotValid)
	}
	return nil
}

// verticesForPrimitives is GraphicsDevice::GetElementCountArray, the topology
// table both families' window checks are computed from:
//
//	PointList      n
//	LineList      2n
//	LineStrip     n + 1
//	TriangleList  3n
//	TriangleStrip n + 2
//
// It is written out rather than asked of CNA -- cna_graphics_device_get_vertex_count_for_primitives
// exists -- because the reference computes it MANAGED-side and the answer
// decides which of the reference's own messages a caller sees. Asking CNA would
// make a managed refusal depend on a native call.
func verticesForPrimitives(primitiveType PrimitiveType, primitiveCount int32) (int32, error) {
	switch primitiveType {
	case PrimitiveTypeLineList:
		return primitiveCount * 2, nil
	case PrimitiveTypeLineStrip:
		return primitiveCount + 1, nil
	case PrimitiveTypeTriangleList:
		return primitiveCount * 3, nil
	case PrimitiveTypeTriangleStrip:
		return primitiveCount + 2, nil
	default:
		// XNA 4.0's PrimitiveType has exactly these four members, so a value
		// outside them is not a topology the reference can draw either. Its own
		// GetElementCountArray falls through a switch and returns zero, which
		// makes every window check pass and hands D3D a topology it refuses;
		// this refuses by NAME instead, which is a Go-only decision and is
		// recorded as one.
		return 0, fmt.Errorf("%w: primitiveType: %d is not one of XNA's four topologies",
			errGraphicsResourceArgument, int32(primitiveType))
	}
}
