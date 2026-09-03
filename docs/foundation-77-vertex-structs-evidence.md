# Foundation 77 — the four stock vertex structs

```text
COMPLETE_TYPES   159 -> 163        MISSING_TYPE       98 -> 94
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
INTERFACE_WITNESS_PROJECTIONS  28 -> 32
USER_PRIMITIVE_DRAWS  120 -> 220 per 20-cycle run, 0 refusals
```

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
```

## The element tables are read, not computed

Each of the four types carries a `public static initonly VertexDeclaration`
assigned once by its `.cctor`. Those tables are transcribed offset by offset
from the IL:

| type | elements | stride |
| --- | --- | ---: |
| `VertexPositionColor` | `(0, Vector3, Position, 0)`, `(12, Color, Color, 0)` | 16 |
| `VertexPositionTexture` | `(0, Vector3, Position, 0)`, `(12, Vector2, TextureCoordinate, 0)` | 20 |
| `VertexPositionColorTexture` | `+ (16, Vector2, TextureCoordinate, 0)` | 24 |
| `VertexPositionNormalTexture` | `(0, Vector3, Position, 0)`, `(12, Vector3, Normal, 0)`, `(24, Vector2, TextureCoordinate, 0)` | 32 |

They are **not** derived from the Go struct layout. Go's layout is not the CLR's
marshalled one, and a projection that computed offsets would be asserting a
coincidence rather than reproducing a table. The tests assert the transcribed
values, and the native stress run submits them to CNA.

The `.cctor` runs once and every read of the static field answers the same
object, so the projection caches: the static-field reader and the `IVertexType`
witness hand back the same `*VertexDeclaration`, and the four types have four
distinct ones.

## Three details that decide behaviour

1. **`op_Equality` compares the last field first.** `VertexPositionColor` tests
   `Color` before `Position`. The result is unaffected — equality is a
   conjunction — and the order is preserved because it is what the reference
   does.
2. **`Equals(object)` is a TYPE test before it is a field test.**
   `obj.GetType() != this.GetType()` returns false before any field is read, so
   a `VertexPositionColor` never equals a `VertexPositionColorTexture` whatever
   they hold.
3. **`GetHashCode` is `Helpers.SmartGetHashCode`** — the settled rule the GamePad
   value structs already use: XOR every complete 32-bit word of the marshalled
   struct, and substitute `Int32.MaxValue` for a zero result. A `Vector3` is
   three words, a `Vector2` two, a `Color` one. The zero substitution is
   deliberate and creates compatible collisions; the all-zero
   `VertexPositionColor` hashes to `2147483647`, and that is pinned.

The four hash values in the tests were computed from the rule rather than from
this implementation, the way Foundation 74's string hashes were.

## The IVertexType witnesses

All four implement `IVertexType::get_VertexDeclaration` as a `private hidebysig
newslot virtual final` with an `.override`, so the contract's public member set
has none of them — and Go needs the method exported for the type to satisfy the
interface at all. That is what a witness is.

Both halves of the settled witness rule hold here, which is why the registry
gains four entries: the member is absent from the public set, **and** the Go
type can declare it, because `VertexDeclaration` is a type in its own package.
`GraphicsDeviceManager`'s other interface fails the second half and is
deliberately absent for exactly that reason.

## Native qualification

`tools/native_stress` now makes **eleven** user-primitive submissions per cycle
rather than six: the original six overloads over the harness's own vertex type,
plus one per stock type, plus one with a **non-zero `vertexOffset`** — four
vertices, one triangle, starting at index one, which none of the six exercised.

```text
HEADLESS   USER_PRIMITIVE_DRAWS 220   REFUSALS 0   GUARD_CHECKS 20
SOFTWARE   USER_PRIMITIVE_DRAWS 220   REFUSALS 0   GUARD_CHECKS 20
```

CNA accepts every one, on both artifacts, which is what proves `FromType`
resolves each type's static declaration through its witness and that CNA accepts
the reference's own element tables.

**This is `VERIFIED_NATIVE_DRAW`, not `VERIFIED_PIXEL`.** The back buffer can be
read back on the SOFTWARE artifact — Foundation 73 proved that — but the colour a
drawn triangle produces is not independently predictable without an effect whose
output is known, and the stock effects are not projected yet. Upgrading these
draws to pixel evidence is the first thing the stock-effect milestone unblocks,
and saying so is the difference between recording a limitation and claiming a
result.

## Qualification

```text
go test ./...                        PASS
go run ./tools/behavior              OBSERVATIONS=737 ASSERTIONS=737 FAILURES=0
go run ./tools/api_compat            TOTAL_DIAGNOSTICS=94, all MISSING_TYPE;
                                     MISSING_MEMBER=0, PARTIAL_TYPES=0
go run ./tools/native_abi ...        BOUND_FUNCTIONS=230 ABI_MISMATCHES=0
go run ./tools/external_consumer     TESTS=102 FAILURES=0 STATUS=PASS
native_stress HEADLESS + SOFTWARE    20 cycles each, PASS
```

The ABI is untouched: the four types reach CNA through routes that were already
bound.
