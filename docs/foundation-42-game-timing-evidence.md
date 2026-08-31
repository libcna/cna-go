# Foundation 42 — Game's timing and presentation state

Ten of `Game`'s nineteen remaining members are complete. All six canonical CNA
operations behind them were already exported by the pinned artifact and had
simply never been reached from Go.

```text
InactiveSleepTime   TimeSpan  get/set
TargetElapsedTime   TimeSpan  get/set
IsFixedTimeStep     bool      get/set
IsMouseVisible      bool      get/set
SuppressDraw()
ResetElapsedTime()
```

## Getters are field reads; setters have to reach the loop

In the reference every one of these getters is a single `ldfld`, and the
**managed loop reads the same fields every frame**. CNA-Go's loop is the native
one, so the split falls out of that difference rather than from a choice:

- the **getter** stays a field read of `Game`'s own managed state. Seven bytes
  of IL, no validation, no host, no window, no device, no throw site — so it is
  classified `managedStoredMembers` and is **infallible**, like `Components` and
  `Services` before it. It works before `Run`, during it, and after it.
- the **setter** validates as the reference does, stores as the reference does,
  and then pushes the value to the native loop — because that loop is what the
  reference's own loop would have read. A stored value the loop never sees would
  be a setter that appears to work and does not.

The setters are therefore fallible where the reference's are not, and that is
recorded rather than smoothed: the native call can genuinely be refused, from
the wrong thread or with an argument CNA rejects, and swallowing that would be
the one thing the projection may not do.

## The `cna_game_get_*` functions are deliberately not bound

CNA exports getters for all four values. Binding them would introduce a **second
source of truth** that could disagree with the first. The reference reads its own
field; so does this.

## Two setters that look symmetrical and are not

```
set_InactiveSleepTime   op_LessThan          ->  ZERO IS ACCEPTED
set_TargetElapsedTime   op_LessThanOrEqual   ->  zero is rejected
```

One IL instruction is the whole difference, and it is observable. The message
`set_InactiveSleepTime` loads even says the value "cannot be zero" — and the
comparison it sits behind admits zero anyway. The message is reproduced verbatim
because it is the reference's; the boundary is the IL's.

Both messages are the exact `Resources` strings the throw sites load.

## The constructor's defaults are now load-bearing

Read from `Game::.ctor`:

```text
maximumElapsedTime  = TimeSpan.FromMilliseconds(500)
isFixedTimeStep     = true
targetElapsedTime   = TimeSpan.FromTicks(0x28b0b)     // 166667, one sixtieth of a second
inactiveSleepTime   = TimeSpan.FromMilliseconds(20)   // 200000 ticks
isMouseVisible      = (not assigned)                  // false
updatesSinceRunningSlowly1/2 = int.MaxValue
```

These stopped being documentation and became behaviour. `cna_go_game_create`
used to pass the literals `is_fixed_time_step = 1` and
`target_elapsed_time_ticks = 166667`; it now passes **what the managed state
actually says**, because a consumer may configure a `Game` before `Run` and the
reference's loop would honour that. `CNA_GameCreateInfo` has no field for the
other two, so they are pushed immediately after creation, on the owner thread,
before the loop can run a frame — and a failure there destroys the game rather
than leaving one whose configured state was silently not applied.

## `IsMouseVisible` and the window that is not there

```csharp
this.isMouseVisible = value;
if (this.Window != null) this.Window.IsMouseVisible = value;
```

Store first, then propagate — guarded, because `get_Window` returns null when the
`Game` has no host. CNA-Go has no `GameWindow` and no `GameHost`: the native
runtime owns the window, and `cna_game_set_is_mouse_visible` publishes the same
effect. So the guarded branch projects exactly: with a live native game the value
reaches the window, and with none there is no window to reach — which is the state
the reference itself is in before `Run`.

## Measured against the pinned artifact

A new crash-isolated scenario, 20 subprocesses:

```text
GAME_TIMING_CYCLES                          20
GAME_TIMING_SETTERS_APPLIED                120     == 6 per cycle
GAME_TIMING_WRONG_THREAD_CHECKS             20
GAME_TIMING_RANGE_CHECKS                    20
GAME_TIMING_CREATED_WITH_CONFIGURED_STEP    20
```

`120 = 6 × 20` is the claim that matters: every one of the six settings reached a
**live** native loop and CNA accepted it, from inside a lifecycle callback on the
owner thread.

`GAME_TIMING_CREATED_WITH_CONFIGURED_STEP = 20` is the other one. Each cycle
configures a non-default target step *before* `Run`. If the create path still
passed a literal — or if CNA had rejected the value — the run would not have
started at all.

The wrong-thread check runs the same setter from another goroutine and requires
`ErrWrongThread`: a value the loop will not honour must not look applied.

## ABI

CNA was **not** rebuilt and no CNA C++ changed. Six symbols already in the pinned
binary are now reached:

```text                       before   after
BOUND_FUNCTIONS                  25      31
PROTOTYPE_TYPE_POSITIONS         91      91     (75 -> 91)
C_GO_MEASUREMENTS               112     128
LAYOUTS                          36      36
CALLBACKS                         3       3
CONSTANTS                        15      15
```

Six new mutation controls, and two of them are worth naming. A tick count
narrowed from `int64_t` to `int32_t` would cap the target step at about 3.6
minutes rather than failing, and a result code turned into a `CNA_Bool` would
lose every failure — and **neither is caught by the bridge translation unit**,
because C converts silently between integer widths and between a narrow unsigned
return and a wider one. They are caught by the probe, which compares CNA-Go's
private manifest against the canonical declaration. The four that the bridge
*can* catch — a lost game handle, an extra parameter, a missing symbol — stay
there.

Native ABI mutation controls: 24 → 30.

## Scoreboard

```text                      before   after
TOTAL_DIAGNOSTICS            292     282
MISSING_MEMBER               157     147
TARGET_MEMBERS              1794    1804
Game missing members          19       9

behavior corpus              639     644
external canary tests         59      62
native stress scenarios        6       7
native ABI mutations          24      30
runtime capability rows       42      43
```

`Game`'s nine remaining members are all blocked on something outside itself:
`Content` ×2 and `LaunchParameters` on missing types, `GraphicsDevice` on the
partial `Graphics.GraphicsDevice`, `Window` on the missing `GameWindow`,
`IsActive` on `Microsoft.Xna.Framework.GamerServices.dll` — not one of the seven
pinned assemblies — `ShowMissingRequirementMessage` on the `System.Exception`
frontier, and `Tick`/`RunOneFrame` on the frame-step re-entrancy rules, which CNA
exports and which are the natural next slice.
