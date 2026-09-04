package graphics

// SetDataOptions specifies how an XNA buffer data write relates to data the
// buffer already holds.
// xna:flags
type SetDataOptions int32

const (
	SetDataOptionsNone        SetDataOptions = 0
	SetDataOptionsDiscard     SetDataOptions = 1
	SetDataOptionsNoOverwrite SetDataOptions = 2
)

// nativeSetDataOptions is the projection of
// `ConvertXnaSetDataOptionsToDx(SetDataOptions)`, 27 bytes, which is a BITWISE
// test rather than an equality and therefore accepts values the enum does not
// name:
//
//	IL_0000:  ldarg.0; ldc.i4.1; and; ldc.i4.1; bne.un.s IL_000c
//	IL_0006:  ldc.i4 0x2000; ret                 // D3DLOCK_DISCARD
//	IL_000c:  ldarg.0; ldc.i4.2; and; ldc.i4.2; beq.s IL_0015
//	IL_0012:  ldc.i4.0; br.s IL_001a             // no lock flag
//	IL_0015:  ldc.i4 0x1000; ret                 // D3DLOCK_NOOVERWRITE
//
// Three consequences a reader would not predict from the enum, and all three
// are the reference's:
//
//   - Bit 0 WINS. `Discard | NoOverwrite` is Discard, because the first test
//     returns before the second runs.
//   - An UNDEFINED value is never refused. 99 has bit 0 set and is Discard; 4
//     has neither bit and is None. The reference has no validation here at all.
//   - Only two bits are ever looked at, so every other bit is ignored.
//
// # Why this is a conversion and not a cast
//
// CNA numbers its three options the same way XNA does -- CNA_SET_DATA_NONE 0,
// _DISCARD 1, _NO_OVERWRITE 2 -- so a cast would be right for the three named
// values and WRONG for everything else: CNA documents that it answers
// `CNA_RESULT_INVALID_ARGUMENT` for "an undefined option", and the reference
// refuses nothing. Passing the caller's raw value through would turn a silent
// bit test into a refusal, which is a divergence in the direction that breaks
// working consumer code.
//
// A shared numbering is a coincidence to be checked, not a rule to rely on --
// the same position nativeIndexElementSize and nativeBufferUsage take.
func nativeSetDataOptions(options SetDataOptions) uint32 {
	if options&1 == 1 {
		return 1 // CNA_SET_DATA_DISCARD, the reference's D3DLOCK_DISCARD
	}
	if options&2 == 2 {
		return 2 // CNA_SET_DATA_NO_OVERWRITE, the reference's D3DLOCK_NOOVERWRITE
	}
	return 0 // CNA_SET_DATA_NONE, the reference's zero
}
