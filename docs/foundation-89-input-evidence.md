# Foundation 89 — the Input family, and what its touch half turned out to be

Five types close the Input namespace: `GamePad`, `GamePadCapabilities`,
`Mouse`, `TouchPanel` and `TouchPanelCapabilities`. The state structs
(`GamePadState`, `MouseState`, `TouchCollection`, `TouchLocation`,
`GestureSample`) were already projected, so what was missing was the static
readers and the two capability structs.

The milestone was scoped as "bind the twenty-two CNA input routes and project
five types". It ended with **eight routes bound and fourteen reverted**, because
of what the pinned IL says about touch.

## Authority

| what | file | sha256 |
| --- | --- | --- |
| GamePad, Mouse, `Helpers::ValidateOrientation`, FrameworkResources | `Microsoft.Xna.Framework.dll` | `38e7093f52d7474b…` |
| TouchPanel, TouchPanelCapabilities, TouchCollection | `Microsoft.Xna.Framework.Input.Touch.dll` | `b0585224c18022c3…` |
| native evidence | `~/deps/cna-c-abi-0.21.0/libcna_c_api.so` | `c32bfbd307d69566…` |

## The finding: XNA 4.0's Windows touch stack is a stub

`Microsoft.Xna.Framework.Input.Touch.dll` **declares no p/invoke anywhere**.
Not one. Every member is managed code over managed static fields, and the four
that would need a digitizer are hard-coded:

```
TouchPanelCapabilities::GetCaps()   ldloca.s V_0; initobj TouchPanelCapabilities;
                                    ldloc.0; ret        // ten bytes
TouchPanel::GetState()              zeroes a local XNAINPUT_TOUCH_LOCATION_STATE,
                                    never writes it, and updates the static
                                    collection from it  // always EMPTY
TouchPanel::ReadGesture()           29 bytes, two branches, BOTH `throw`.
                                    No `ret` instruction exists in the body.
TouchPanel::get_IsGestureAvailable  throws when gestures were never enabled,
                                    otherwise `ldc.i4.0`  // constant false
```

XNA 4.0's touch surface shipped for Windows Phone. The Windows build kept the
public API and removed the device half. So on the runtime this binding targets:

- `GetCapabilities()` reports `IsConnected == false` and
  `MaximumTouchCount == 0` on every machine, digitizer attached or not.
- `GetState()` answers an empty collection — whose `IsConnected` is **true**,
  because `Update` is called with the literal `true`. The type contradicts its
  own capabilities call, and the projection reproduces the contradiction rather
  than smoothing it over; `TestTouchPanelStateIsConnectedDisagreesWithCapabilities`
  pins it.
- `ReadGesture()` cannot return. Which of the two messages a consumer gets
  depends only on whether `EnabledGestures` was ever assigned.
- The documented idiom `while (IsGestureAvailable) ReadGesture();` never enters
  its body, which is what keeps ReadGesture's unconditional throw from breaking
  a correctly written consumer.

### Fourteen routes bound, measured, and reverted

CNA implements this family for real — `cna_touch_get_capabilities`,
`cna_touch_get_state`, `cna_touch_panel_read_gesture` and get/set routes for
all five properties — and they answer from an actual digitizer. All fourteen
were bound first: the manifest carried `CNA_TouchState`, `CNA_TouchLocation`,
`CNA_TouchCapabilities` and `CNA_GestureSample` mirrors, the bridge flattened
them, and `BOUND_FUNCTIONS` reached 364 with `ABI_MISMATCHES=0`.

Then the IL was read, and every one of them was removed.

The reason is the standing authority rule: the pinned XNA IL is the contract
**and behaviour** authority, and CNA is evidence of what the native layer can
do, never evidence of what XNA does. A consumer polling `TouchPanel.GetState`
in its update loop gets an empty collection from the reference on every
machine. A projection that returned real touches would answer a question the
reference answers differently, and a game built against it would behave one way
here and another way on the runtime this binding exists to match.

All fourteen are recorded in `deliberatelyUnboundRoutes` under
`CONTRACT_DIVERGENCE`, which is the class the audio sample-conversion routes
were given in Foundation 87 for the same reason.

`BOUND_FUNCTIONS` 342 → **350**. `DELIBERATELY_UNBOUND_ROUTES` 83 → **97**.

### The counter that exists to be zero

`TOUCH_PANEL_NATIVE_CALLS` is asserted to be zero by the parent accounting, and
every TouchPanel member is exercised **inside a live game with a working native
runtime available** so that the assertion means something. It is the same
device as `MICROPHONE_CAPTURE_CALLS` in Foundation 88: a counter whose value
being zero is the finding.

## GamePad: a missing controller is not an error

The three readers share one body:

```
if (ThrottleDisconnectedRetries(index)) return <empty>;
ErrorCodes result = UnsafeNativeMethods.GetState/GetCaps(index, ...);
ResetThrottleState(index, result);
if (result == 0x48f) return <empty>;          // ERROR_DEVICE_NOT_CONNECTED
if (result != success)
    throw new InvalidOperationException(FrameworkResources.InvalidController);
return new GamePadState(...);
```

`0x48f` is 1167, `ERROR_DEVICE_NOT_CONNECTED`. A disconnected controller
answers an **empty value with `IsConnected` false and no exception**; only some
other failure throws. A projection that turned a missing controller into an
error would break the four-index polling loop every XNA game writes.

This build machine has no controller attached, so `GAMEPADS_CONNECTED` is 0 and
that branch is the one the stress run actually takes. A zero there is a
measured result, not a skipped test.

`GetState(PlayerIndex)` is two instructions: it forwards with `ldc.i4.1`, which
is `GamePadDeadZone.IndependentAxes`. One Go wrapper serves both native routes
and a flag chooses, because both overloads are reachable.

The private retry throttle is not projected: nothing public reports it, CNA
does its own polling, and reproducing it would be reproducing a workaround for
XInput's latency rather than a behaviour.

`SetVibration` is the one member here that **drives** hardware rather than
sampling it. The stress slice calls it with two zeros on every call — the value
the reference itself uses to stop a motor — and records
`GAMEPAD_VIBRATION_CALLS` and `GAMEPAD_VIBRATIONS_APPLIED` separately so a run
cannot quietly skip it. The parent asserts applied ≤ connected.

## GamePadCapabilities: twenty-five flags in one array

The struct is 26 properties, 25 of them Boolean. They cross cgo as a **flat
byte array** in the order the contract declares them, filled on the C side, so
a thirty-eight-argument signature never exists. Named index constants in the
same order make a disagreement between the two sides a compile error rather
than a wrong answer, and `TestGamePadCapabilitiesReadTheirOwnSlot` walks all
twenty-five with exactly one flag raised.

CNA's `CNA_GamePadCapabilities` carries **eleven `_ext` flags XNA does not
declare** — light bar, trigger vibration motors, misc1, four paddles, touchpad,
gyro, accelerometer. They are not copied.

## Mouse: the reference body is Win32

`GetCursorPos`, `ScreenToClient`, `GetAsyncKeyState` ×5. There is no managed
contract to reproduce beyond the shape — positions are in the hooked window's
**client** coordinates and the scroll wheel is a running total in 120-unit
notches — so CNA's counterparts answer and their results are projected
unchanged.

`CNA_MouseState.horizontal_scroll_wheel` has no XNA counterpart. It crosses
with the rest of the structure and is dropped.

The stress slice writes the cursor position back to where it already is, so a
run does not move a pointer someone may be holding.

## What the verifier dictated

As in the audio family, `api_compat` named the members: `GamePadGetCapabilities`
(not `…ByPlayerIndex`), `GamePadSetVibration`, `MouseGetState`,
`MouseWindowHandle`/`SetMouseWindowHandle`, and the `Set…` prefix on every
TouchPanel setter.

`TouchPanel` was admitted to `pureManagedTypes` — on the strongest evidence in
that registry, since the whole assembly has no FFI — with four entries in
`managedFallibleMembers`: `method|ReadGesture`,
`property-get|IsGestureAvailable`, `property-set|EnabledGestures` and
`property-set|DisplayOrientation`. The two setters are marked `property-set`
rather than `property` because both getters are one `ldsfld` and cannot fail,
and `WindowHandle`, `DisplayWidth` and `DisplayHeight` are absent entirely:
their setters store and raise a flag with **no validation at all**, so a
negative width is accepted and read back unchanged.

`set_EnabledGestures` rejects any bit outside `0x3FF` and reports the literal
string `"EnabledGestures"` as the exception **message** — the reference calls
`ArgumentException(string)`, whose single parameter is the message, so what
reads like a parameter name is what a consumer sees. Reproduced, not corrected.

`ValidateOrientation` compares for **equality** against 0, 1, 2 and 4, so
`LandscapeLeft | LandscapeRight` (3) is refused even though both bits are
declared.

Four resource strings were verified byte for byte and registered:
`InvalidController`, `GesturesNotEnabled`, `GesturesNotAvailable` and
`InvalidDisplayOrientation`. `GesturesNotAvailable` really does carry two
spaces after its first sentence. `RESOURCE_STRINGS_VERIFIED` 71 → **75**.

## Falsifiability

Forty-four defects were planted across the family's five types, and
forty-one were killed. The managed table runs against both Input packages; the native table runs against the
stress slice, which is the only place the GamePad and Mouse routes execute and
the only place the TouchPanel-reaches-nothing assertion is meaningful.

Two rounds were needed, and the first round's survivors were all real gaps in
the tests rather than unkillable mutations:

- `display_height_reads_the_width` survived because the test assigned the
  **same** value to both fields. It now assigns complements, in both orders.
- `mouse_right_button_reads_the_middle_bit` survived because the test exercised
  the bit helper directly while the wiring lived inline inside `MouseGetState`,
  where no managed test could reach it. The expansion is now a named
  `mouseButtonStates`, which is a change made **for** falsifiability and is
  documented as such at the definition.
- `buttons_from_mask_drops_the_big_button` survived a round-trip check over a
  full mask -- the survivors still OR back together. Each literal is now
  expanded alone and must come back alone.

Planting also found a real projection defect that no test had caught:
`GamePadState.PacketNumber` was being left at zero. The reference's public
constructors do set it to zero, but the one `GetState` actually calls is
`GamePadState::.ctor(XINPUT_STATE&, ...)`, which reads it from the XInput
snapshot -- and PacketNumber participates in `Equals` and `GetHashCode`, so a
constant zero would collapse "the controller has not moved" into "the
controller reports the same values again". CNA reports the same number and it
is now stored.

### Totals

| table | planted | killed | survived | skipped |
| --- | ---: | ---: | ---: | ---: |
| managed | 33 | 33 | 0 | 0 |
| native | 11 | 8 | 3 | 0 |
| **total** | **44** | **41** | **3** | **0** |

Two of the mutations began in the managed table and were **moved** to the
native one when they proved to need a live runtime; they are counted once, in
the table they now run in.

The native round also drove a change to the slice itself. Both mouse
window-handle mutations survived the first two attempts, and the reason was
measured rather than guessed: the HEADLESS artifact reports `0x0` for both
`Mouse.WindowHandle` and `GameWindow.Handle`, so writing the handle back over
itself could not tell a dropped write from an honest one. The slice now writes
a **sentinel** and unbinds it -- CNA's own header calls the parameter an
"opaque native window value; zero unbinds", so nothing dereferences it -- and
both mutations die.

### The three unkilled survivors, each named

All three need an **attached game controller**, which this build machine does
not have. They are not scored as killed and they are not hidden:

1. `one_argument_get_state_forwards_the_wrong_dead_zone` — swapping
   `IndependentAxes` for `None` in the two-instruction forwarding body. A dead
   zone only changes an answer for a connected controller with a stick
   off-centre; with none attached both overloads answer the same empty state.
   The *constant* is pinned separately:
   `TestGamePadOneArgumentGetStateForwardsIndependentAxes` asserts
   `IndependentAxes == 1`, which is the `ldc.i4.1` the reference emits.
2. `gamepad_state_drops_the_packet_number` and
3. `gamepad_state_reads_the_button_mask_as_the_packet` — both sit on the
   CONNECTED branch, which no cycle on this machine enters.

A machine with a controller attached would kill all three, and the slice is
written so that it would: the capabilities/state presence cross-check, the
packet-number assertion on the disconnected branch and the applied-vibration
bound are already in place and would start exercising their other halves.

`capabilities_flag_conversion_inverts` is **not** on this list -- it looked
controller-dependent and turned out not to be. Inverting the flag conversion
raises `IsConnected` for a controller that is not there, and the
capabilities/state cross-check catches it on the very first index.

## Scoreboard

| counter | before | after |
| --- | ---: | ---: |
| COMPLETE_TYPES | 189 | 194 |
| MISSING_TYPE | 68 | 63 |
| PARTIAL_TYPES | 0 | 0 |
| MISSING_MEMBER | 0 | 0 |
| GLOBAL_UNREVIEWED | 0 | 0 |
| BOUND_FUNCTIONS | 342 | 350 |
| DELIBERATELY_UNBOUND_ROUTES | 83 | 97 |
| RESOURCE_STRINGS_VERIFIED | 71 | 75 |
| FRONTIER_FAMILIES | 9 | 8 |
| ABI_MISMATCHES | 0 | 0 |

The Input frontier family was deleted rather than emptied, which the verifier
requires.
