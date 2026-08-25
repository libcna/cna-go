# Foundation 17 — pure managed CLR classes and per-operation fallibility

Foundation 17 replaces two mapping rules that were too coarse, and closes the
first two types that could not be projected honestly without them:
`Microsoft.Xna.Framework.Audio.AudioListener` and
`Microsoft.Xna.Framework.Audio.AudioEmitter`.

## Reference authority

Everything below is read from the retained original Microsoft assembly, not
from a reimplementation and not from FNA or MonoGame.

```text
Microsoft.Xna.Framework.dll
sha256 38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
```

Both classes and the helper they depend on are declared in that one assembly.
Its IL was read with `ikdasm`; the exception message was read from the
`Microsoft.Xna.Framework.FrameworkResources.resources` stream of the same
assembly. The public *surface* remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`; the
assembly is authority for *behavior* only.

## Rule 1 — CLR `class` is not evidence of native backing

The mapping engine used to treat every CLR `class` as a fallible native-backed
facade, so every member of every class gained a Go `error`. That is right for
`Game`, `GraphicsDeviceManager`, `GraphicsDevice`, `SpriteBatch`, and
`Texture2D`, and wrong for a class that never leaves managed memory.

A class is now classified as **pure managed** only when authoritative XNA IL
proves that its selected public behavior is backed entirely by managed fields
and deterministic managed code, and therefore

- owns no CNA native object,
- requires no FFI, no native allocation, and no native destruction,
- requires no renderer or device query,
- requires no callback registration and no thread-affinity lifecycle,
- depends on no external hardware state.

The classification lives in `pureManagedTypes`. The five native-backed runtime
types above are deliberately absent and keep their fallible facades.

**Classification does not change semantics.** An admitted class is still a CLR
reference type. Its constructor projects as `*T`, and two Go variables holding
one instance observe the same mutations. The closure measurement records the
constructor's result type for exactly this reason, and a defect fixture
projects the constructor as a value to prove the check bites.

## Rule 2 — fallibility belongs to one operation

`managedFallibleMembers` used to key on `property|Name`, which marks *both*
accessors. `AudioEmitter.DopplerScale` cannot be expressed that way: its getter
is one `ldfld` and its setter throws.

Keys are now produced by `fallibilityKeys` and are, most specific first:

```text
constructor|.ctor      one constructor
method|<Name>          one ordinary or static method
field|<Name>           one field projection
property-get|<Name>    that property's getter only
property-set|<Name>    that property's setter only
property|<Name>        both accessors of that property
```

`property|<Name>` remains supported because some XNA properties genuinely throw
from both accessors — `CurveKeyCollection`'s indexer validates its index on read
and on write. Spelling an assignment-only validation that way is a defect, not
a shorthand, and is fixtured as one.

Nothing about this is specific to `AudioEmitter`. The scheme is general, and
`TestFallibilityKeysAreAccessorSpecific` pins it against a synthetic owner with
no relation to the audio types.

## `UnsafeNativeStructures::FlipHandedness`

Both classes store their vectors in XACT's left-handed convention and convert
on every public read and every public write. The private helper is:

```text
IL_0004: ldfld float32 Vector3::X ; stfld X
IL_0012: ldfld float32 Vector3::Y ; stfld Y
IL_0020: ldfld float32 Vector3::Z ; neg ; stfld Z
```

so `FlipHandedness(v) = (v.X, v.Y, -v.Z)`. CIL `neg` and Go's unary minus are
both an IEEE-754 sign-bit flip, which makes the helper a **bit-exact
involution** for every binary32 value — both zeros, both infinities, and every
NaN payload. Applying it on write and again on read is therefore an exact
identity for callers, which is why no public accessor pair needs to be
described as lossy.

`flipHandedness` is unexported. It converts between a public convention and a
private storage convention and has no meaning to a caller.

## The negative-zero default

The constructors do **not** flip the zero vectors, but the getters do:

```text
.ctor: _Position = Vector3.Zero                  (no flip)
       _Velocity = Vector3.Zero                  (no flip)
       _Forward  = FlipHandedness(Vector3.Forward)
       _Up       = FlipHandedness(Vector3.Up)
```

`Vector3.Zero`, `Vector3.Up`, and `Vector3.Forward` are `(0,0,0)`, `(0,1,0)`,
and `(0,0,-1)` in the `Vector3::.cctor` of the same assembly. So on a freshly
constructed listener or emitter:

| property   | getter result bits                    | equals the source constant |
| ---------- | ------------------------------------- | -------------------------- |
| `Position` | `0x00000000,0x00000000,0x80000000`    | `Vector3.Zero` (Z is `-0`) |
| `Velocity` | `0x00000000,0x00000000,0x80000000`    | `Vector3.Zero` (Z is `-0`) |
| `Forward`  | identical to `Vector3.Forward`        | bit-exact                  |
| `Up`       | identical to `Vector3.Up`             | bit-exact                  |

`Position` and `Velocity` default to a **negative zero** on Z. `Forward` and
`Up` do not, because their stored values were already flipped once. The sign is
invisible to `Vector3` equality — `-0 == 0` — but it is a real, observable
difference in the bits, and it is reproduced rather than normalized away. A
previous handoff described the round-trip as identity, which is true, but did
not note that the constructed defaults are not a round-trip.

## `AudioListener` — exact contract

CLR: `public class AudioListener : System.Object`, no interfaces, 5 source
members, 9 projected Go identities, no error result anywhere.

```go
func NewAudioListener() *AudioListener

func (l *AudioListener) Position() framework.Vector3
func (l *AudioListener) SetPosition(value framework.Vector3)
func (l *AudioListener) Velocity() framework.Vector3
func (l *AudioListener) SetVelocity(value framework.Vector3)
func (l *AudioListener) Forward() framework.Vector3
func (l *AudioListener) SetForward(value framework.Vector3)
func (l *AudioListener) Up() framework.Vector3
func (l *AudioListener) SetUp(value framework.Vector3)
```

Every getter is `ldflda listenerData; ldfld _X; call FlipHandedness`. Every
setter is `ldflda listenerData; ldarg.1; call FlipHandedness; stfld _X`. **No
accessor validates anything**: a zero, denormal, infinite, or NaN orientation is
stored unchanged, and `Forward` and `Up` are never normalized or
orthogonalized.

`XACT_LISTENER_DATA` has a fifth field, `private void* pCone`. It is private,
never written by any public member, and has no public projection. No native
XACT storage is created here just because a native representation exists
elsewhere.

## `AudioEmitter` — exact contract

CLR: `public class AudioEmitter : System.Object`, no interfaces, 6 source
members, 11 projected Go identities, exactly one error result.

```go
func NewAudioEmitter() *AudioEmitter

func (e *AudioEmitter) Position() framework.Vector3
func (e *AudioEmitter) SetPosition(value framework.Vector3)
func (e *AudioEmitter) Velocity() framework.Vector3
func (e *AudioEmitter) SetVelocity(value framework.Vector3)
func (e *AudioEmitter) Forward() framework.Vector3
func (e *AudioEmitter) SetForward(value framework.Vector3)
func (e *AudioEmitter) Up() framework.Vector3
func (e *AudioEmitter) SetUp(value framework.Vector3)
func (e *AudioEmitter) DopplerScale() float32
func (e *AudioEmitter) SetDopplerScale(value float32) error
```

The four positional properties behave exactly as the listener's. The
constructor additionally sets `_DopplerScale`, `ChannelCount`,
`ChannelRadius`, and `CurveDistanceScaler` to 1; only the first is public in
the XNA 4.0 Windows profile, and the other three are kept as unexported
managed state so this type's fields match the reference field for field.
`XACT_EMITTER_DATA`'s cone, curve, and channel-azimuth pointers have no managed
meaning to reproduce and are not modelled.

### `set_DopplerScale` — exact validation

```text
IL_0000: ldarg.1
IL_0001: ldc.r4     0.0
IL_0006: bge.un.s   IL_0018
IL_0008: ldstr      "value"
IL_000d: call       FrameworkResources::get_InvalidEmitterDopplerScale()
IL_0012: newobj     System.ArgumentOutOfRangeException::.ctor(string, string)
IL_0017: throw
IL_0018: ... stfld  _DopplerScale
```

`bge.un` branches to the store when the ordered comparison `value >= 0`
succeeds **or** when the comparison is unordered. The throw is therefore reached
on exactly the negative-ordered values, and `value < 0` in Go is its exact
complement:

| input                       | `bge.un` branch taken | result                                |
| --------------------------- | --------------------- | ------------------------------------- |
| ordinary positive           | yes                   | stored                                |
| `+0`                        | yes                   | stored                                |
| `-0`                        | yes                   | stored, keeping the `0x80000000` bits |
| `+Infinity`                 | yes                   | stored                                |
| `NaN`                       | yes (unordered)       | **stored** — NaN does not throw       |
| ordinary negative           | no                    | `ArgumentOutOfRangeException`         |
| smallest negative denormal  | no                    | `ArgumentOutOfRangeException`         |
| `-Infinity`                 | no                    | `ArgumentOutOfRangeException`         |

Accepting NaN reads like an oversight in the reference. It is preserved.

A rejected assignment throws before the `stfld`, so the previously stored value
is untouched.

The getter is unaffected. `get_DopplerScale` is one `ldflda` plus one `ldfld`
with no validation at all, so it returns the stored bits verbatim — including a
`-0` or a NaN that a previous accepted assignment stored.

### Exception versus error, kept distinct

| aspect                     | value                                                                       |
| -------------------------- | --------------------------------------------------------------------------- |
| CLR exception type         | `System.ArgumentOutOfRangeException`                                        |
| CLR parameter name         | `"value"`                                                                    |
| CLR message resource key   | `FrameworkResources.InvalidEmitterDopplerScale`                             |
| exact CLR message          | `The doppler scale of an audio emitter must be greater than or equal to zero.` |
| trigger condition          | `bge.un(value, 0.0f)` not taken, i.e. `value < 0` in Go terms               |
| Go projection              | non-nil `error` from `SetDopplerScale` only                                 |
| Go behavior on rejection   | stored value unchanged, no panic                                            |

The Go error is a **language projection**, not XNA behavior. The behavior corpus
labels the acceptance/rejection column `PURE_XNA_DERIVED` and the error channel
itself `GO_LANGUAGE_PROJECTION`, so the two authorities stay separate. The
established `error` mapping is used; no new error architecture was invented, and
nothing panics.

## Verifier coverage

`managedClassClosure` measures each pure-managed class: CLR kind, base type,
classification flag, package and name, source identities against projected
identities, the constructor's reference projection, accessor-pair count, and a
per-accessor fallibility verdict.

```text
AudioListener  PASS  5 source -> 9 Go identities, 4 accessor pairs, 0 errors, ref *AudioListener
AudioEmitter   PASS  6 source -> 11 Go identities, 5 accessor pairs, 1 error,  ref *AudioEmitter
               fallible getters 0, fallible setters 1, fallible other 0
```

`ERROR_MAPPING_MISMATCH` diagnostics now name the accessor and the direction of
the disagreement, so all four accessor cases are distinguishable in the report:

```text
property getter expected fallible, projected infallible
property getter expected infallible, projected fallible
property setter expected fallible, projected infallible
property setter expected infallible, projected fallible
```

with `constructor`, `method`, `field projection`, and `event accessor` used for
the non-accessor kinds.

### Negative fixtures

Target-side, 13 defects × 2 classes = 26 cases, each asserted to raise its
category *and* to drop the closure to `FAIL`, after asserting a clean baseline:

```text
missing_type  wrong_package  projected_as_named_type  value_semantics_constructor
missing_first_member  missing_last_member  renamed_last_member  unexpected_member
wrong_setter_parameter  artificial_getter_error  artificial_setter_error
artificial_constructor_error  native_facade_projection
```

Classification-side, 6 defects that attack the two general rules themselves by
mutating the real classification tables and re-deriving the expected surface.
Each asserts both the category and the exact accessor-and-direction wording:

```text
managed_class_demoted_to_native_facade            CLR `class` alone makes it fallible
managed_class_with_validating_setter_demoted      the same, with a real throw present
native_backed_class_admitted_as_pure_managed      Texture2D loses its native errors
accessor_fallibility_widened_to_whole_property    property| replaces property-set|
accessor_fallibility_moved_to_the_getter          the wrong accessor is marked
accessor_fallibility_dropped                      the one genuine throw is lost
```

All 32 are also registered in `testdata/mutations.json`, taking the declared
mutation inventory from 217 to 249.

## Behavior corpus

48 new observations in group `AUDIO_DESCRIPTOR`, taking the corpus from 487 to
535 observations with zero failures: constructed defaults as raw bits for both
types, the negative-zero equality note, six bit-exact round-trip families across
both types, the twelve-row `DopplerScale` table with its stored bits, the
twelve matching error-channel projections, and reference-semantics aliasing for
both types.

## Structural effect

```text                       before   after
TARGET_TYPES                    103     105
TARGET_MEMBERS                 1619    1639
TOTAL_DIAGNOSTICS               331     329
MISSING_TYPE                    154     152
MISSING_MEMBER                  177     177
COMPLETE_TYPES                   98     100
PARTIAL_TYPES                     5       5
```

The five protected partial runtime types are untouched, and every mismatch,
leak, allowlist, and unmeasured counter stays at zero. `MISSING_MEMBER` is
unchanged because no member of a protected type was implemented.

## No ABI change

Neither type reaches native code, so the CNA-Go ABI is unchanged at
`23 / 67 / 96 / 28 / 2 / 5` with no missing header symbols, no missing library
symbols, and no ABI mismatches. CNA was not rebuilt.
