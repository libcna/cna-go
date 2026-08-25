# Foundation 37 — the bridge lifetime, proved rather than argued

Foundation 34 made three lifetime claims about the native game-event bridge and
supported them with a C probe and prose:

- the four registrations installed for one run are **released** when that run
  ends, and a second run installs four fresh ones;
- `Add` and `Remove` are pure managed list work, so they work with **no native
  game alive at all**;
- a signal arriving at a runtime that is no longer live is **dropped and
  recorded**, never delivered into a torn-down facade.

A probe proves what the C API does. It does not prove what the binding does with
it. This milestone closes that gap: one new crash-isolated stress scenario for
the parts that reach C, and eight unit tests for the parts that have a decision
in them but no native call.

No projected member changed, no ABI counter moved, and the pinned artifact is
untouched.

## The second run

`tools/native_stress` gained an `event-rerun` scenario. Each of its 20 isolated
subprocesses constructs **one** Go `Game`, subscribes to all four events once,
runs it to completion, subscribes and unsubscribes again with the native game
gone, and then runs the same `Game` a second time.

```text
GAME_EVENT_RERUN_CYCLES     20
GAME_EVENT_POST_RUN_CHECKS  20

GAME_EVENT_ACTIVATED_DELIVERIES     80     <- 60 + 20, NOT 60 + 40
GAME_EVENT_DEACTIVATED_DELIVERIES    0
GAME_EVENT_EXITING_DELIVERIES      100     <- 60 + 40, two per rerun cycle
GAME_EVENT_DISPOSED_DELIVERIES     100     <- 60 + 40, two per rerun cycle
GAME_EVENT_REMOVAL_CHECKS           80
GAME_EVENT_OWNER_THREAD_CHECKS      80

NATIVE_CRASHES 0   OBSERVED_UAF 0   OBSERVED_DOUBLE_FREE 0
```

`Exiting` and `Disposed` each arrive exactly once **per run**, on the same owner
goroutine, with no consumer resubscribing between them. That is the release-then-
reinstall claim, measured 20 times: if the first run's registrations had leaked,
the second run would have delivered each signal twice; if they had been released
too early, the first run's `Disposed` would never have arrived at all.

## Activated is delivered twice and raised once, and that is correct

The interesting number is 80 rather than 100.

`Game::isActive` is a private field the reference never resets, and the two
activation events are edge-triggered on it:

```text
HostActivated(sender, e)
  if (isActive) return;        // already active -> nothing at all
  isActive = true;
  OnActivated(this, EventArgs.Empty);
```

The first run leaves `isActive` true — nothing ever lowers it, because HEADLESS
can produce no deactivation. The second run's native activation signal therefore
arrives and is **suppressed by the guard**, exactly as `HostActivated` suppresses
a repeated host activation in CLR. The scenario asserts that suppression rather
than tolerating it: a second-run `Activated` raise is a test failure.

This is worth stating plainly because it is the first place the edge-trigger
guard changes an observable outcome, and it changes it in the direction the
reference does.

## What the unit tests cover

`internal/interop` had no test file before this. It has eight now, over the
decisions that do not reach C:

| test | claim |
| -- | -- |
| routes every identity | the four identities reach the framework unchanged, in arrival order |
| dead runtime delivers nothing | a signal at a non-live runtime is dropped and recorded as `ErrStaleGeneration` |
| records a handler failure | the failure is recorded, not discarded, and the **first** one wins |
| contains a panic | a panicking handler is recovered and recorded; nothing crosses the C frame |
| release is idempotent | the slots are zeroed, so a second release passes nothing rather than a stale handle CNA would refuse |
| names cover every identity | trace names and identities cannot drift, including the unknown case |
| identities are contiguous from zero | the Go end of the constant chain; the C end is compiler-checked and measured by `native_abi` |
| `Callbacks` gained exactly one member | five lifecycle names unchanged, `GameEvent(uint32) error` added, and it is internal |

The last one is the compatibility claim from the inside: the public
`GameCallbacks` contract was never touched, which is why no consumer had to
implement anything to receive these signals.

## Scoreboard

```text                                     before   after
GAME_EVENT_RERUN_CYCLES                     —       20
GAME_EVENT_POST_RUN_CHECKS                  —       20
native stress scenarios                      3        4
internal/interop tests                       0        8

TARGET_MEMBERS                            1791     1791
TOTAL_DIAGNOSTICS                          295      295
MISSING_MEMBER                             160      160
ABI (25/75/107/31/3/10)                      unchanged
```
