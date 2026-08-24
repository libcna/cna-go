# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 4 are complete. Milestone 1 established the
real CNA C-ABI runtime architecture. Milestone 2 completed managed geometry,
Color, and Viewport. Milestone 3 completed Curve and the collection/iterator
projection. Milestone 4 adds no C function and completes exactly the managed
PackedVector family:

```text
IPackedVector                 IPackedVectorOfTPacked[TPacked]
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

Preserve these invariants:

- namespace paths remain `Microsoft/Xna/Framework[/...]`;
- `internal/interop` is the only C/cgo/symbol/handle/callback/ownership/thread boundary;
- public XNA packages contain no C types or raw/native handles;
- structs remain Go values and XNA classes remain pointer facades;
- native resources route only through canonical CNA C ABI 0.7.0;
- absent API remains structurally measured, never hidden by a fake or no-op.

## Foundation 4 language projection

The declared closure has 171 XNA source members and 189 mapped Go members.
The generic collision rule preserves `IPackedVector` and maps
`IPackedVector<TPacked>` to `IPackedVectorOfTPacked[TPacked]`. General owner
generic substitution maps CLR `!n` to the nth declared parameter, so the
generic property is exactly:

```go
PackedValue() TPacked
SetPackedValue(TPacked)
```

Both interfaces are pure managed value interfaces; none of their methods gain
synthetic `error` results. The generic interface embeds `IPackedVector`.

All seventeen concrete XNA structs remain Go value structs with one private
fixed-width packed integer. Getters and converters use value receivers;
`SetPackedValue` and `PackFromVector4` use pointer receivers. Compiler
`go/types` evidence proves that every `*T` satisfies the exact constructed
`IPackedVectorOfTPacked[uint8|uint16|uint32|uint64]` and transitively
`IPackedVector`, while `T` does not satisfy the mutable interface.

Explicit CLR implementations require 25 measured Go witnesses: seventeen
`PackFromVector4` methods and eight reduced-format `ToVector4` methods. They are
reported separately and do not inflate target member counts. Missing/wrong
witnesses fail interface/signature checks; bogus public methods remain
`UNEXPECTED_MEMBER`. `ALLOWLIST_ENTRIES` remains zero.

## Packed behavior

UNorm, SNorm, raw unsigned, and raw signed paths preserve XNA binary32
scale/clamp order followed by `System.Math.Round` midpoint-to-even behavior.
NaN maps to zero; infinities map to the appropriate clamp endpoint before
integer conversion. Direct `PackedValue` assignment is bit-transparent.

SNorm `-1` packs to `-127` or `-32767`; a directly assigned signed minimum
still decodes to `-1`. Byte4 is raw `[0,255]` numeric data. Short2/Short4 are
raw signed Int16 data with complete `[-32768,32767]` clamping and exact
two's-complement lanes.

XNA half conversion is intentionally non-IEEE at exponent 31. All 65,536 bit
patterns decode to finite binary32 values and round-trip. `0x7C00` decodes to
65536 and `0x7FFF` to 131008. Float infinity and NaN saturate to signed
`0x7FFF`, discarding payload/signaling state while retaining sign. Signed zero,
subnormals, threshold-adjacent values, and ties are exact.

Equality and both operators compare packed bits. Hashes are the packed value
for widths through 32 bits and low32 XOR high32 for 64 bits. Non-half strings
are fixed-width uppercase packed hex; half values delegate to XNA Single/Vector
string formatting. Full layouts and fixtures are in
`docs/packed-vector-evidence.md`.

## Structural scoreboard

The retained XNA contract SHA-256 remains
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
and contains 257 types / 2,964 members. The formal Go projection remains 257
types / 3,243 members.

```text
TARGET_TYPES=54
TARGET_MEMBERS=1278
TOTAL_DIAGNOSTICS=383
MISSING_TYPE=203
MISSING_MEMBER=180
COMPLETE_TYPES=48
PARTIAL_TYPES=6
MISSING_TYPES=203

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, internal-type/raw-handle/public-FFI leak, allowlist, and unmeasured
counter is zero. All nineteen PackedVector types have zero local diagnostics.

The exact remaining partial types are unchanged:

- `Microsoft.Xna.Framework.Game`
- `Microsoft.Xna.Framework.GraphicsDeviceManager`
- `Microsoft.Xna.Framework.Graphics.GraphicsDevice`
- `Microsoft.Xna.Framework.Graphics.SpriteBatch`
- `Microsoft.Xna.Framework.Graphics.Texture2D`
- `Microsoft.Xna.Framework.Input.Keyboard`

Their 180 missing members are unchanged native/runtime work.
`Keyboard.GetState(PlayerIndex)` remains absent.

## Behavioral evidence

The managed `PURE_XNA_DERIVED` corpus now has 201 observations, 201 assertions,
and zero failures. Foundation 4 contributes 59 observations:

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

The generated exhaustive report has 262,400 iterations and zero failures:
Alpha8 256; Bgr565, Bgra4444, Bgra5551, and HalfSingle 65,536 each.

## ABI and native regression

The admitted native library remains the exact HEADLESS/NULL-audio ABI 0.7.0
artifact with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`.
Milestone 4 leaves every ABI count unchanged:

```text
BOUND_FUNCTIONS=23
PROTOTYPE_TYPE_POSITIONS=67
C_GO_MEASUREMENTS=96
LAYOUTS=28
CALLBACKS=2
CONSTANTS=5
MISSING_HEADER_SYMBOLS=0
MISSING_LIBRARY_SYMBOLS=0
ABI_MISMATCHES=0
```

Ordinary and Go-race native stress retain 20 Game, recreation, texture,
SpriteBatch, callback-error, and callback-panic cycles, 80 forced-GC points,
and zero crash/UAF/double-free observations. `GO_RACE_STATUS=PASS`; native
sanitizers remain `NOT_RUN`.

The maintained `cna-go-template` source is unchanged at commit
`65254848d9fac02ace934db3879106834bafca97`. Its test, vet, race, build,
trimpath, exact 60-Draw, and exact 600-Draw gates pass against the same admitted
library. Visible rendering remains backend-blocked; no native runtime or
hardware capability claim changed.

## Re-running gates

Use Go 1.24.4 with Linux amd64 cgo and GCC. Set `CNA_NATIVE_LIBRARY` to an
absolute exact ABI-0.7 library path for native gates.

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict       # expected 383 missing diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY"
go run -race ./tools/native_stress --race-status PASS
git diff --check
```

Source-artifact and isolated-consumer qualification use a deterministic archive
of the exact worktree, not the development checkout. The external milestone
handoff records the archive filename, SHA-256, entry count, two-pass
determinism, audit counters, and isolated 60/600 results so the hash is not
recursively embedded in the archive it describes.

## One next dependency-complete milestone

The freshly regenerated missing inventory selects exactly the managed
vertex-element descriptor closure:

```text
Microsoft.Xna.Framework.Graphics.VertexElement
Microsoft.Xna.Framework.Graphics.VertexElementFormat
Microsoft.Xna.Framework.Graphics.VertexElementUsage
```

The pinned contract currently gives three types, 37 source identities, and 39
mapped Go identities. Recompute those counts before implementation. Do not mix
that milestone with `IVertexType`, native `VertexDeclaration`/buffers, Design,
Content/XNB, Effects/3D, native partial cleanup, or another family.

```text
FOUNDATION_MILESTONE_4_COMPLETE=true
```
