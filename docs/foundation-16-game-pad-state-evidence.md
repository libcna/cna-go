# Foundation Milestone 16 GamePadState evidence

## Authority and exact one-type closure

Foundation Milestone 16 completes exactly one public XNA type:

```text
Microsoft.Xna.Framework.Input.GamePadState
```

It became dependency-complete the moment the Foundation-15 game pad value
structs landed, and it is the direct payoff of that cluster: both of its
constructors take exactly those values.

Public surface authority is the pinned XNA 4.0 Windows contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
Reference behavior was read as IL from the retained
`Microsoft.Xna.Framework.dll`, SHA-256
`38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130`.

The closure is 15 source identities and 15 mapped Go identities: two
constructors, `IsButtonDown`, `IsButtonUp`, `Equals`, `GetHashCode`,
`ToString`, both operators, and the `Buttons`, `DPad`, `IsConnected`,
`PacketNumber`, `ThumbSticks`, and `Triggers` properties.

## The private XInput snapshot is managed arithmetic, not a native route

`GamePadState` keeps a private `XINPUT_STATE` field and fills it in both
constructors through `FillInternalState`. The IL shows that method is **pure
managed bit packing** over the public values: it reads the `GamePadButtons`,
`GamePadDPad`, `GamePadThumbSticks`, and `GamePadTriggers` properties and ORs
XInput bits into a struct. There is no P/Invoke, no marshalling, and no device
access anywhere on this path.

CNA-Go reproduces it as an ordinary unexported Go struct, `xinputGamePad`. It
is never exposed, never marshalled, and never handed to a native library. It
exists only because `IsButtonDown` reads the packed form rather than the public
values, so an accurate projection has to reproduce it.

Every XInput bit the reference packs coincides with the pinned `Buttons`
literal of the same name. That coincidence is **measured**, not assumed, by
`TestGamePadStateXInputBitsMatchPinnedButtons`:

| Bit | Value | Pinned `Buttons` literal |
|---|---:|---:|
| DPadUp / DPadDown / DPadLeft / DPadRight | 0x0001 / 0x0002 / 0x0004 / 0x0008 | 1 / 2 / 4 / 8 |
| Start / Back | 0x0010 / 0x0020 | 16 / 32 |
| LeftStick / RightStick | 0x0040 / 0x0080 | 64 / 128 |
| LeftShoulder / RightShoulder | 0x0100 / 0x0200 | 256 / 512 |
| BigButton | 0x0800 | 2048 |
| A / B / X / Y | 0x1000 / 0x2000 / 0x4000 / 0x8000 | 4096 / 8192 / 16384 / 32768 |

This is the third independent cross-validation of the Foundation-14 `Buttons`
enum, after the `GamePadButtons` constructor masks and the `IsButtonDown`
virtual-bit masks.

Analogue values are quantized exactly as the reference does: triggers by
`× 255` into a `uint8`, thumbsticks by `× 32767` into an `int16`, both with CIL
`conv` truncation toward zero. Because `GamePadTriggers` and
`GamePadThumbSticks` already clamp their inputs, these products cannot
overflow. ECMA-335 leaves the conversion of NaN unspecified; on the x86 and x64
runtimes float-to-int32 yields the integer indefinite value `0x80000000`, whose
low 16 and low 8 bits are both zero, so NaN maps to 0. Go leaves the same
conversion undefined, so `clrConvertToInt16` and `clrConvertToUInt8` spell the
rule out rather than leaving it to the compiler.

## Dead zones

`IsButtonDown` hard-codes `GamePadDeadZone.IndependentAxes`, so only that mode
is reachable from the completed public surface. The Circular and None modes are
reachable only through `GamePad.GetState`, which is not implemented, and are
therefore **deliberately absent** rather than written as unreachable code.

From the internal `GamePadDeadZoneUtils`:

```text
LeftStickDeadZoneSize  = 0x1EA9 = 7849
RightStickDeadZoneSize = 0x21F1 = 8689
TriggerDeadZoneSize    = 0x1E   = 30

ApplyLinearDeadZone(value, maxValue, deadZoneSize):
    if value < -deadZoneSize: value += deadZoneSize
    elif value >  deadZoneSize: value -= deadZoneSize
    else: return 0
    return MathHelper.Clamp(value / (maxValue - deadZoneSize), -1, 1)
```

Both reference comparisons are unordered branches, so a NaN input takes neither
branch and the function returns zero. That is reproduced exactly and tested.

## IsButtonDown

```text
effective = packedButtons & 0xFBFF          // the reference _normalButtonMask
for each thumbstick direction and trigger bit the caller actually asked about:
    add the bit if its dead-zoned axis is on the correct side of zero
return (button & effective) == button
```

The ten virtual bits are computed lazily — only when the requested mask
contains them — and each mask matches the pinned `Buttons` literal:
`LeftThumbstickLeft` 0x200000, `LeftThumbstickRight` 0x40000000,
`LeftThumbstickDown` 0x20000000, `LeftThumbstickUp` 0x10000000,
`RightThumbstickLeft` 0x8000000, `RightThumbstickRight` 0x4000000,
`RightThumbstickDown` 0x2000000, `RightThumbstickUp` 0x1000000,
`LeftTrigger` 0x800000, `RightTrigger` 0x400000.

Two consequences are reference behavior and are asserted rather than smoothed
over:

- **`IsButtonDown(0)` reports true.** The empty mask is trivially contained.
- **Every requested bit must be present.** `IsButtonDown(A|B)` is false when
  only `A` is down.

`IsButtonUp` is the exact logical negation.

Worked dead-zone fixtures, all asserted:

| Input | Quantized | Dead zone | Result |
|---|---:|---|---|
| thumbstick 0.5 | 16383 | 7849 | outside → direction reported |
| thumbstick 0.1 | 3276 | 7849 | inside → nothing reported |
| trigger 0.5 | 127 | 30 | outside → trigger reported |
| trigger 0.1 | 25 | 30 | inside → nothing reported |
| trigger NaN | 0 | 30 | inside → nothing reported |

## Constructors, hashing, and formatting

Both public constructors set `PacketNumber` to 0 and `IsConnected` to **true**.
`IsConnected` reports what the constructor stored — it is not a device read.

The second constructor combines its `Buttons` slice with a bitwise OR (a nil
slice combines to zero), builds the thumbstick and trigger values through their
own clamping constructors, and derives the directional pad with a **non-zero**
bit test. That differs in form from the `GamePadButtons` whole-mask rule but
agrees for the single-bit directional literals; both forms are reproduced as
written.

`GetHashCode` XORs the six stored member hash codes — thumbsticks, triggers,
buttons, the connected flag, the directional pad, and the packet number. The
derived XInput snapshot does not participate. `Boolean.GetHashCode` returns 1
or 0 and `Int32.GetHashCode` returns the value itself. Because four of the six
components are `Int32.MaxValue` for a default value, the arithmetic gives:

| Value | Hash |
|---|---:|
| zero `GamePadState` (not connected) | 0 |
| default-constructed (connected) | 1 |

`ToString` reports only `{IsConnected:True}` or `{IsConnected:False}`, using
CLR Boolean formatting.

The equality operator compares the six stored members and ignores the derived
snapshot, because the snapshot is a pure function of them.

## Structural and behavior evidence

`foundation16ValueStructClosures` reuses the Foundation-15 value-struct closure
category. `GamePadState` reports `PASS` with 15 source identities, 15 target Go
identities, `errorResults: 0`, and zero local diagnostics.

The exhaustive value-struct defect test now spans both milestones — 8 value
structs and 91 identities, 80 negative cases across 10 defects, each with an
asserted clean baseline. The declared mutation inventory grows from 210 to 217.

The behavior corpus grows from 476 to **487** observations with zero failures.
`GAME_PAD_STATE` contributes ten `PURE_XNA_DERIVED` observations covering the
constructor defaults, the slice combination, normal and combined button
queries, both dead-zone boundaries, the empty-mask rule, and the hash/string
fixtures, plus one `GO_LANGUAGE_PROJECTION` observation for the Go zero value
being disconnected.

## Runtime boundary

Completing `GamePadState` claims managed value behavior only. **No game pad
capability is claimed.** CNA-Go still exposes no `GamePad` type; there is no
polling, device enumeration, connection detection, dead-zone configuration,
vibration, or SDL route. The dead-zone arithmetic reproduced here is applied to
values the caller supplied, never to a device reading.

`capabilities --check` still reports `RUNTIME_CAPABILITIES=38 STATUS=PASS` and
`docs/runtime-capabilities.json` is unmodified. No CNA source, C binding, cgo
route, native mirror, layout, or callback was added; the ABI remains
23 / 67 / 96 / 28 / 2 / 5 with zero missing symbols and zero mismatches, and
both native reports were reproduced against the exact Foundation-11 pinned
library.

## Scoreboard

| Counter | Foundation 15 | Foundation 16 | Delta |
|---|---:|---:|---:|
| `TARGET_TYPES` | 102 | 103 | +1 |
| `TARGET_MEMBERS` | 1604 | 1619 | +15 |
| `TOTAL_DIAGNOSTICS` | 332 | 331 | −1 |
| `MISSING_TYPE` | 155 | 154 | −1 |
| `MISSING_MEMBER` | 177 | 177 | 0 |
| `COMPLETE_TYPES` | 97 | 98 | +1 |
| `PARTIAL_TYPES` | 5 | 5 | 0 |

Every preservation, mismatch, leak, allowlist, and unmeasured counter remains
zero. The dependency-complete node count moved 47 → 46.

## Why the safe seam is now exhausted

After this milestone, **no dependency-complete missing type remains that can be
completed without either a public-API policy decision or fabricated device
capability.** The remaining fully-resolved candidates fall into exactly three
groups:

1. **Blocked on mapping policy** — `AudioListener` and `AudioEmitter` (pure
   managed classes projected as fallible; the latter also needs setter-only
   fallibility), `IEffectMatrices`, `IEffectFog`, `IGameComponent`,
   `IGraphicsDeviceManager` (managed-interface projection policy), and
   `PresentationParameters` (would expose `System.IntPtr` as a public raw
   handle).
2. **Would fabricate device capability** — `TouchPanelCapabilities`,
   `GamePadCapabilities`, `RendererDetail`, `DisplayMode`, `Mouse`: read-only
   surface with no public constructor, so every getter could only ever report
   an invented value.
3. **Deferred families with unmapped BCL** — `MathTypeConverter`,
   `ContentManager`, the Content serializer attributes, the exception types,
   `LaunchParameters`, `GameServiceContainer`, `SpriteFont`, `Video`.

Group 1 is the productive frontier and needs a decision, not more search.
