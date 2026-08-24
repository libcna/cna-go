# Foundation Milestone 6 PlayerIndex and Keyboard evidence

## Authority and exact closure

Foundation Milestone 6 completes exactly:

```text
Microsoft.Xna.Framework.PlayerIndex
Microsoft.Xna.Framework.Input.Keyboard.GetState(
    Microsoft.Xna.Framework.PlayerIndex
)
```

The pinned public contract remains
`tools/api_compat/reference/xna40-windows-runtime-contract.json` at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The independently inspected original `Microsoft.Xna.Framework.dll` has
SHA-256
`38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130`,
matching the retained assembly identity.

The closure contains seven source identities: five on `PlayerIndex`, including
the synthetic enum storage field, and two Keyboard methods. The established
enum projection excludes `value__`, producing six mapped Go identities: four
enum constants and two Keyboard functions.

The root `Microsoft/Xna/Framework` package does not import its `Input` child.
The child can therefore import the root package for the exact `PlayerIndex`
identity without a cycle, alias, duplicate enum, or separate fake type.

## PlayerIndex enum contract

`PlayerIndex` is an ordinary non-flags CLR enum with `System.Int32` underlying
storage. It maps to a named Go `int32` with explicit constants and no `iota`:

| XNA literal | Go identity | Raw value |
|---|---|---:|
| `One` | `PlayerIndexOne` | 0 |
| `Two` | `PlayerIndexTwo` | 1 |
| `Three` | `PlayerIndexThree` | 2 |
| `Four` | `PlayerIndexFour` | 3 |

The CLR enum domain is the complete underlying `int32` domain, not only the
four declared literals. `PlayerIndex(12345)` is representable and preserved.
No membership validator, count constant, flags marker, string helper, or
native constant was added.

## Direct Keyboard IL evidence

The Windows XNA 4.0 reference IL contains two static overloads. The
no-argument method loads raw value `255` and calls
`GetState(PlayerIndex)`. The player-index overload has code size 111 bytes and
begins by taking the address of a local 256-byte keyboard buffer and calling
the process-wide Win32 `GetKeyboardState` function:

```text
IL_0000: ldloca.s 3
IL_0002: call ... GetKeyboardState(unsigned int8*)
...
IL_006d: ldloc.2
IL_006e: ret
```

There is no `ldarg.0`, switch, comparison, conversion, array index, validation,
or other instruction that reads `playerIndex`. The argument is unused for the
entire method. The remaining IL handles the native failure, applies XNA's
internal Home-key policy, scans all 256 bytes, and builds the returned
`KeyboardState` value. This proves that defined and undefined `PlayerIndex`
values select the same process keyboard-state path.

## Go projection and runtime behavior

The exact mapped public functions are:

```go
func KeyboardGetStateByNone() (KeyboardState, error)

func KeyboardGetStateByPlayerIndex(
    playerIndex framework.PlayerIndex,
) (KeyboardState, error)
```

Both functions call one private `keyboardGetState` helper. The player overload
does not inspect or validate its argument. The helper obtains the current
runtime and invokes the already-qualified `Runtime.KeyboardState` route, which
ends at the existing canonical `cna_keyboard_get_state` C ABI function. No C,
cgo, native manifest, symbol, prototype, layout, callback, or constant changed.

The existing lifecycle requirement is unchanged: a current Game runtime must
be active and the call must occur in a lifecycle callback on its owner thread.
Pure tests prove that both overloads return the same zero value and error path
when no Game is active. Crash-isolated native stress proves the same failure
path before `Game.Run` and after shutdown, and proves successful value-equal
snapshots during an active callback for:

```text
One, Two, Three, Four, PlayerIndex(12345)
```

The retained HEADLESS fixture supplies deterministic process keyboard state.
`KeyboardState` is a value snapshot; object identity is neither exposed nor
relevant.

## Behavior and regression evidence

The `PURE_XNA_DERIVED` corpus grows from 227 to 234 observations/assertions
with zero failures:

| Group | Observations | Evidence |
|---|---:|---|
| `PLAYER_INDEX` | 2 | exact four-literal table and undefined raw value |
| `KEYBOARD_PLAYER_INDEX` | 5 | every defined value and raw `12345` use the same public no-current-runtime semantics |

Active-runtime state equivalence remains separate CNA/runtime evidence in the
native stress gate. Package tests also regress `Keys`, `KeyState`, construction,
pressed-key ordering, key queries/indexer, equality, hash, operators, both
overloads, and undefined player values without requiring a native library.

The verifier mutation inventory grows from 47 to 58 cases. The eleven focused
additions reject wrong PlayerIndex kind or underlying type, accidental flags,
wrong `One` or `Four`, missing `Four`, a missing Keyboard overload, an `int32`
parameter substitution, a wrong state result, a missing error result, and a
wrong overload name.

## Compiler-measured local strict-zero matrix

The generated API report retains this dedicated closure measurement:

| Type | Source members | Expected Go | Target Go | Local diagnostics | Kind |
|---|---:|---:|---:|---:|---|
| `PlayerIndex` | 5 | 4 | 4 | 0 | named `int32` enum, non-flags |
| `Input.Keyboard` | 2 | 2 | 2 | 0 | static-class identity |

Every local and global kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out,
language, unexpected-symbol, native-leak, allowlist, and unmeasured counter is
zero. Keyboard moves from partial with one missing member to complete. The
other five partial types and their combined 179 missing members are unchanged.

## ABI and scope boundary

Native ABI measurements remain exactly 23 bound functions, 67 prototype type
positions, 96 aggregate C/Go measurements, 28 layouts, two callbacks, and five
constants, with zero missing header/library symbols and zero mismatches.
`PlayerIndex` remains pure managed metadata and never crosses the ABI.

This milestone adds no GamePad, Mouse, Touch, vertex declaration/type/buffer,
graphics expansion, Design, Content/XNB, Effects, Audio, Media, Storage, or
GamerServices surface and makes no new platform or hardware capability claim.
