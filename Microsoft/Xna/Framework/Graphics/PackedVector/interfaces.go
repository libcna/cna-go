package packedvector

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// IPackedVector is the managed value-packing protocol shared by every XNA
// packed-vector format.
type IPackedVector interface {
	ToVector4() framework.Vector4
	PackFromVector4(framework.Vector4)
}

// IPackedVectorOfTPacked preserves XNA's generic packed storage identity.
type IPackedVectorOfTPacked[TPacked any] interface {
	IPackedVector
	PackedValue() TPacked
	SetPackedValue(TPacked)
}

var (
	_ IPackedVectorOfTPacked[uint8]  = (*Alpha8)(nil)
	_ IPackedVectorOfTPacked[uint16] = (*Bgr565)(nil)
	_ IPackedVectorOfTPacked[uint16] = (*Bgra4444)(nil)
	_ IPackedVectorOfTPacked[uint16] = (*Bgra5551)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*Byte4)(nil)
	_ IPackedVectorOfTPacked[uint16] = (*HalfSingle)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*HalfVector2)(nil)
	_ IPackedVectorOfTPacked[uint64] = (*HalfVector4)(nil)
	_ IPackedVectorOfTPacked[uint16] = (*NormalizedByte2)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*NormalizedByte4)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*NormalizedShort2)(nil)
	_ IPackedVectorOfTPacked[uint64] = (*NormalizedShort4)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*Rg32)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*Rgba1010102)(nil)
	_ IPackedVectorOfTPacked[uint64] = (*Rgba64)(nil)
	_ IPackedVectorOfTPacked[uint32] = (*Short2)(nil)
	_ IPackedVectorOfTPacked[uint64] = (*Short4)(nil)
)
