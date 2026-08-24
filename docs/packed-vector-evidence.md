# PackedVector Foundation evidence

## Authority and closure

Foundation Milestone 4 completes exactly the XNA 4.0 Windows runtime namespace
`Microsoft.Xna.Framework.Graphics.PackedVector`. The authoritative public
contract is the pinned 257-type snapshot in
`tools/api_compat/reference/xna40-windows-runtime-contract.json`; the original
assembly used for behavior and IL evidence has SHA-256
`38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130`.
Other CNA bindings were not behavioral authority.

The closure contains two interfaces and seventeen concrete value structs:

```text
IPackedVector                 IPackedVector<TPacked>
Alpha8                        Bgr565
Bgra4444                      Bgra5551
Byte4                         HalfSingle
HalfVector2                   HalfVector4
NormalizedByte2              NormalizedByte4
NormalizedShort2             NormalizedShort4
Rg32                          Rgba1010102
Rgba64                        Short2
Short4
```

The contract contains 171 declared XNA member identities. Property expansion
maps those to 189 ordinary Go identities. The 25 explicit-interface witness
methods described below are measured separately and do not inflate either
declared-member total.

## Generic and managed-interface mapping

The deterministic generic collision rule retains the non-generic
`IPackedVector` name and maps `IPackedVector<TPacked>` to
`IPackedVectorOfTPacked[TPacked]`. The general CLR owner-parameter substitution
maps `!0` to the first declared parameter (`TPacked` here), `!1` to the second,
and so on. Invalid or out-of-range tokens produce
`GENERIC_MAPPING_MISMATCH`; no raw `!0` or fallback `any` is admitted.

Both interfaces are explicitly classified as pure managed value interfaces.
They do not gain a synthetic `error` merely because their source owner is an
interface:

```go
type IPackedVector interface {
    ToVector4() framework.Vector4
    PackFromVector4(framework.Vector4)
}

type IPackedVectorOfTPacked[TPacked any] interface {
    IPackedVector
    PackedValue() TPacked
    SetPackedValue(TPacked)
}
```

`IPackedVectorOfTPacked` directly embeds `IPackedVector`. The verifier measures
the generic parameter list, both interface kinds, inheritance, exact method
sets, getter/setter expansion, and absence of synthetic errors.

## Mutable structs and exact interface identity

Every concrete packed type remains a Go value struct with one private
fixed-width packed integer as canonical state. Getters and conversion methods
have value receivers. `SetPackedValue` and `PackFromVector4` mutate the value
and therefore have pointer receivers. Direct packed-bit assignment performs no
clamping, conversion, or canonicalization.

Compiler `go/types` evidence proves all seventeen exact relationships below.
For each row, `*T` implements the constructed generic interface and its
transitive `IPackedVector` base, while `T` does not implement the mutable
interface method set:

| Concrete type | Exact constructed interface |
|---|---|
| `Alpha8` | `IPackedVectorOfTPacked[uint8]` |
| `Bgr565` | `IPackedVectorOfTPacked[uint16]` |
| `Bgra4444` | `IPackedVectorOfTPacked[uint16]` |
| `Bgra5551` | `IPackedVectorOfTPacked[uint16]` |
| `Byte4` | `IPackedVectorOfTPacked[uint32]` |
| `HalfSingle` | `IPackedVectorOfTPacked[uint16]` |
| `HalfVector2` | `IPackedVectorOfTPacked[uint32]` |
| `HalfVector4` | `IPackedVectorOfTPacked[uint64]` |
| `NormalizedByte2` | `IPackedVectorOfTPacked[uint16]` |
| `NormalizedByte4` | `IPackedVectorOfTPacked[uint32]` |
| `NormalizedShort2` | `IPackedVectorOfTPacked[uint32]` |
| `NormalizedShort4` | `IPackedVectorOfTPacked[uint64]` |
| `Rg32` | `IPackedVectorOfTPacked[uint32]` |
| `Rgba1010102` | `IPackedVectorOfTPacked[uint32]` |
| `Rgba64` | `IPackedVectorOfTPacked[uint64]` |
| `Short2` | `IPackedVectorOfTPacked[uint32]` |
| `Short4` | `IPackedVectorOfTPacked[uint64]` |

Compile-time assertions in the implementation and verifier mutations cover a
wrong packed type, a missing mutator, a wrong setter type, and an incorrectly
value-received mutable interface.

## Explicit-interface witnesses

The pinned metadata and IL show that every concrete struct explicitly
implements `PackFromVector4`, although it is absent from the type's ordinary
public declared member list. Eight reduced-converter formats also require an
explicit `ToVector4`. The formal witness total is therefore 25:

| Owner | Language-added witnesses |
|---|---|
| `Alpha8` | `PackFromVector4`, `ToVector4` |
| `Bgr565` | `PackFromVector4`, `ToVector4` |
| `Bgra4444` | `PackFromVector4` |
| `Bgra5551` | `PackFromVector4` |
| `Byte4` | `PackFromVector4` |
| `HalfSingle` | `PackFromVector4`, `ToVector4` |
| `HalfVector2` | `PackFromVector4`, `ToVector4` |
| `HalfVector4` | `PackFromVector4` |
| `NormalizedByte2` | `PackFromVector4`, `ToVector4` |
| `NormalizedByte4` | `PackFromVector4` |
| `NormalizedShort2` | `PackFromVector4`, `ToVector4` |
| `NormalizedShort4` | `PackFromVector4` |
| `Rg32` | `PackFromVector4`, `ToVector4` |
| `Rgba1010102` | `PackFromVector4` |
| `Rgba64` | `PackFromVector4` |
| `Short2` | `PackFromVector4`, `ToVector4` |
| `Short4` | `PackFromVector4` |

The generated API report contains owner, source interface/member, reason,
exact signature, and status for every witness. Admission requires an actual
mapped direct interface and an exact required signature. Missing and malformed
witness mutations fail interface/signature checks. Arbitrary and bogus public
methods remain `UNEXPECTED_MEMBER`; this is not an allowlist.

`PackFromVector4` consumes W for `Alpha8`, X for `HalfSingle`, XYZ for
`Bgr565`, XY for every two-component format, and XYZW for every four-component
format. Reduced `ToVector4` results are exactly `(0,0,0,A)` for `Alpha8`,
`(X,Y,Z,1)` for `Bgr565`, `(X,0,0,1)` for `HalfSingle`, and `(X,Y,0,1)` for
all two-component formats. Interface-dispatch fixtures qualify these values;
the fill lanes were not inferred from convention.

## Shared numeric rules

All public inputs are binary32 `Single` values. Scaling occurs in `float32`
before the reference helper widens the already-rounded value to `double` for
`System.Math.Round`. Finite values are clamped before conversion and rounding
is midpoint-to-even. Adjacent binary32 values immediately below and above
midpoints are covered for every lane-width family.

The private managed helper layer implements four reference-derived paths:

- UNorm: binary32 `value * max`, clamp to `[0,max]`, round-to-even, then mask;
  decode is `bits/max` in binary32.
- SNorm: binary32 `value * positiveMax`, clamp to
  `[-positiveMax,positiveMax]`, round-to-even, and store two's-complement bits.
  The otherwise unreachable packed signed minimum decodes specially to `-1`.
- Raw unsigned: no scale; clamp to `[0,max]`, then round-to-even.
- Raw signed: no scale; clamp to the complete signed integer range, then
  round-to-even and store exact two's-complement bits.

For all four numeric helper paths, NaN maps to zero, positive infinity maps to
the upper clamp endpoint, and negative infinity maps to the lower endpoint.
No Go conversion from a non-finite float to an integer is used.

## Per-format packing matrix

Bit ranges are inclusive and numeric; no representation depends on host byte
order.

| Type | Input domain and rule | Packed layout | Decode |
|---|---|---|---|
| `Alpha8` | UNorm `[0,1]` alpha | `A[0:7]` in `uint8` | `A/255` |
| `Bgr565` | three UNorm channels | `Z[0:4], Y[5:10], X[11:15]` | divide by 31, 63, 31 |
| `Bgra4444` | four UNorm channels | `Z[0:3], Y[4:7], X[8:11], W[12:15]` | each lane `/15` |
| `Bgra5551` | four UNorm channels | `Z[0:4], Y[5:9], X[10:14], W[15]` | colors `/31`, W `/1` |
| `Byte4` | raw unsigned byte `[0,255]` | X, Y, Z, W in successive 8-bit lanes, X low | raw `0..255` floats |
| `HalfSingle` | XNA half conversion | one 16-bit lane | exact XNA half decode |
| `HalfVector2` | XNA half conversion | X low 16, Y high 16 | two exact half decodes |
| `HalfVector4` | XNA half conversion | X, Y, Z, W in successive 16-bit lanes, X low | four exact half decodes |
| `NormalizedByte2` | SNorm8 | X low 8, Y high 8 | signed lane `/127`; `0x80 -> -1` |
| `NormalizedByte4` | SNorm8 | X, Y, Z, W in successive 8-bit lanes | same SNorm8 rule |
| `NormalizedShort2` | SNorm16 | X low 16, Y high 16 | signed lane `/32767`; `0x8000 -> -1` |
| `NormalizedShort4` | SNorm16 | X, Y, Z, W in successive 16-bit lanes | same SNorm16 rule |
| `Rg32` | two UNorm16 channels | X low 16, Y high 16 | each lane `/65535` |
| `Rgba1010102` | four UNorm channels | X `[0:9]`, Y `[10:19]`, Z `[20:29]`, W `[30:31]` | divide by 1023, 1023, 1023, 3 |
| `Rgba64` | four UNorm16 channels | X, Y, Z, W in successive 16-bit lanes, X low | each lane `/65535` |
| `Short2` | raw signed Int16 `[-32768,32767]` | X low 16, Y high 16 | sign-extended raw integers |
| `Short4` | raw signed Int16 `[-32768,32767]` | X, Y, Z, W in successive 16-bit lanes | sign-extended raw integers |

The Bgra5551 W midpoint is exactly the one-bit UNorm rule: the binary32 value
immediately below `0.5` and exact `0.5` pack zero; the immediately greater
binary32 value packs one. The Rgba1010102 two-bit W lane follows the same
binary32-scale and midpoint-to-even sequence with maximum three.

SNorm endpoints are deliberately XNA-specific: input `-1` packs `-127` for
NormalizedByte and `-32767` for NormalizedShort, not the signed storage
minimum. A direct packed minimum (`0x80` or `0x8000`) still decodes to `-1`.
Raw `Byte4`, `Short2`, and `Short4` do not normalize their inputs or outputs.

## XNA half conversion

The implementation starts from `math.Float32bits` and reproduces XNA 4.0's
private `HalfUtils` integer algorithm. It preserves the sign of zero, handles
normal/subnormal shifts explicitly, and adds the retained low-bit term before
the 13-bit reduction to implement ties-to-even. No float64 arithmetic or
third-party float16 dependency participates in the conversion.

XNA 4.0's packed half is not IEEE 754 binary16 at exponent 31. All 65,536 bit
patterns decode to finite binary32 values:

| Case | Packed bits | Decoded binary32 bits/value |
|---|---:|---|
| positive zero | `0000` | `00000000`, `0` |
| negative zero | `8000` | `80000000`, `-0` |
| smallest subnormal | `0001` | `33800000`, `5.960464E-08` |
| largest subnormal | `03FF` | `387FC000`, `6.097555E-05` |
| smallest normal | `0400` | `38800000`, `6.103516E-05` |
| conventional maximum | `7BFF` | `477FE000`, `65504` |
| exponent-31 boundary | `7C00` | `47800000`, `65536` |
| maximum XNA half | `7FFF` | `47FFE000`, `131008` |

Positive/negative binary32 infinity and positive/negative NaN saturate to
`7FFF`/`FFFF`; sign is retained, while NaN payload and signaling state are not.
They decode to `+131008`/`-131008`, not infinity or NaN. Exact tie fixtures at
binary32 bits `3F801000` and `3F803000`, plus their adjacent bit patterns,
prove ties-to-even. The exhaustive result is:

```text
HalfSingle iterations=65536
positive zero=1
negative zero=1
subnormal=2046
normal=61440
exponent-31 zero-fraction=2
exponent-31 nonzero-fraction=2046
non-finite decodes=0
encode(decode(bits)) failures=0
```

## Equality, hash, and string behavior

Equality, `Equals(Self)`, `Equals(any)`, and both mapped operators compare the
canonical packed integer exactly. Decoded floating values are never compared;
distinct half bit patterns therefore remain distinct.

`GetHashCode` returns the widened packed integer for `uint8` and `uint16`, the
signed 32-bit reinterpretation for `uint32`, and
`int32(low32) XOR int32(high32)` for `uint64`. Asymmetric high/low fixtures
qualify the 64-bit fold.

All non-half types format the packed integer as uppercase, zero-padded
hexadecimal with exactly 2, 4, 8, or 16 digits. `HalfSingle` delegates to the
decoded `Single` invariant general format; `HalfVector2` and `HalfVector4`
delegate to the corresponding XNA vector representation. Signed zero formats
as `0`, and the seven-significant-digit/uppercase-exponent Single behavior is
shared with the already-qualified managed vector family.

## Independent behavioral and exhaustive evidence

Expected packed integers and binary32 bit strings are retained outside the
production helper in `tools/behavior/main.go` and focused package tests. They
were transcribed from public observations of the pinned original XNA runtime
and checked against its member/helper IL. Production code never generates its
own expected values. The corpus grew from 142 to 201 observations/assertions
with zero failures:

```text
PACKED_INTERFACE=8
PACKED_ALPHA=7
PACKED_16BIT_COLOR=6
PACKED_BYTE4=4
PACKED_HALF=16
PACKED_NORMALIZED_BYTE=4
PACKED_NORMALIZED_SHORT=4
PACKED_RG_RGBA=6
PACKED_SHORT=4
```

`tools/packed_vector_qualify` performs deterministic public-API exhaustive
round trips. Its generated report records 262,400 iterations and zero failures:

```text
Alpha8=256
Bgr565=65536
Bgra4444=65536
Bgra5551=65536
HalfSingle=65536
```

## Local strict-zero matrix

The final compiler-extracted matrix is:

| Type | Source | Expected Go | Target Go | Local diagnostics | Kind | TPacked/direct status |
|---|---:|---:|---:|---:|---|---|
| `IPackedVector` | 2 | 2 | 2 | 0 | interface | — |
| `IPackedVectorOfTPacked` | 1 | 2 | 2 | 0 | interface | `TPacked`; embeds base |
| `Alpha8` | 9 | 10 | 10 | 0 | struct | `uint8` / PASS |
| `Bgr565` | 10 | 11 | 11 | 0 | struct | `uint16` / PASS |
| `Bgra4444` | 10 | 11 | 11 | 0 | struct | `uint16` / PASS |
| `Bgra5551` | 10 | 11 | 11 | 0 | struct | `uint16` / PASS |
| `Byte4` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `HalfSingle` | 9 | 10 | 10 | 0 | struct | `uint16` / PASS |
| `HalfVector2` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `HalfVector4` | 10 | 11 | 11 | 0 | struct | `uint64` / PASS |
| `NormalizedByte2` | 10 | 11 | 11 | 0 | struct | `uint16` / PASS |
| `NormalizedByte4` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `NormalizedShort2` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `NormalizedShort4` | 10 | 11 | 11 | 0 | struct | `uint64` / PASS |
| `Rg32` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `Rgba1010102` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `Rgba64` | 10 | 11 | 11 | 0 | struct | `uint64` / PASS |
| `Short2` | 10 | 11 | 11 | 0 | struct | `uint32` / PASS |
| `Short4` | 10 | 11 | 11 | 0 | struct | `uint64` / PASS |

All PackedVector local mismatch categories are zero. Globally, every existing
mismatch, leak, allowlist, and unmeasured category remains zero. The normal
strict verifier remains red only for genuine missing surface elsewhere in the
profile. PackedVector code is pure managed Go and contains no cgo, unsafe,
native handle, or `internal/interop` dependency.
