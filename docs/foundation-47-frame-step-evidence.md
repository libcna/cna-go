# Foundation 47 — Game.Tick, Game.RunOneFrame, and the standalone native session

`Game.Tick` and `Game.RunOneFrame` are complete. `Game` drops from six missing
members to four, and CNA-Go gains a runtime capability it did not have: a native
game a consumer drives one frame at a time, with no loop anywhere.

## The blocker was CNA-Go's, not CNA's

The retained analysis said a native game existed only inside the blocking `Run`
and that `cna_game_tick` is refused from inside a lifecycle callback — so neither
member had a reachable moment, and projecting one that could only ever fail would
be worse than leaving it missing.

Both halves were true. The conclusion was not, because the constraint was
CNA-Go's own architecture: `Run` created the native game and destroyed it. CNA
brackets a game with `cna_game_create` and `cna_game_destroy`, and
`cna_game_run` is a **separate call** that neither creates nor destroys
anything. Nothing upstream required the two to be one operation.

So this milestone split the **session** out of `Run`. The session is the live
`cna_game_create`/`cna_game_destroy` pair, its `cgo.Handle`, its process lock and
its locked OS thread — everything the first half of `Run` used to do inline.
`Run` now starts a session, runs the loop and ends it; a frame step starts one
and leaves it running.

## What CNA actually does, measured

`build-probe/f47-lifecycle.c` links directly against the qualified artifact —
no CNA-Go anywhere — and drives one game through every state. The first run of
that probe read its counters inside the same `printf` as the call it was
measuring, which C leaves free to evaluate in either order and this compiler
evaluates right to left; the sequenced rewrite is what the numbers below come
from.

```text
create                      = SUCCESS
tick on a never-run game    = SUCCESS   Update 1, Draw 1, Initialize 0, LoadContent 0
run_one_frame (first)       = SUCCESS   Initialize 1, LoadContent 1, Update 2, Draw 2
run_one_frame (second)      = SUCCESS   Initialize 1, LoadContent 1, Update 3, Draw 3
tick after initialization   = SUCCESS   Update 4, Draw 4
tick from inside a callback = INVALID_STATE
tick after run returned     = SUCCESS
destroy                     = SUCCESS   UnloadContent 1, Exiting 1
second create while alive   = INVALID_STATE
```

Five facts came out of that and every one of them shaped the design.

**A tick does not initialize.** `Update` and `Draw` are delivered and
`Initialize`/`LoadContent` are not — which is the reference's behaviour, because
`Game::Tick` is 682 bytes of clock arithmetic with no initialization step
anywhere in it.

**`run_one_frame` initializes exactly once.** CNA's `Game::RunOneFrame` is
`if (!hasInitialized_) { DoInitialize(); hasInitialized_ = true; } Tick();`, and
the second call adds nothing.

**A frame step from inside a lifecycle callback is refused**, with
`CNA_RESULT_INVALID_STATE`. The documented reason is exact: a frame step called
from within a frame would re-enter the loop it is part of.

**Destroying a game delivers `UnloadContent` and the exiting signal**, so ending
a standalone session runs the same shutdown a run does.

**Exactly one C-owned game may be active per process.** That single fact is why
a standalone session must have an ending, and it is what makes `Dispose` load-
bearing below.

## One upstream difference, recorded rather than hidden

XNA's `RunOneFrame` is

```csharp
if (this.host != null) this.host.RunOneFrame();
```

and `WindowsGameHost::RunOneFrame` is `this.gameWindow.Tick()` — pump the
platform's messages, then one `Game.Tick`. **It does not initialize.** CNA's
does.

So a first `RunOneFrame` on a CNA-Go Game delivers `Initialize` and
`LoadContent` where the reference's delivers neither. This is a difference in
CNA, not in the projection, and it is not worked around: CNA offers no route
that runs a host frame without initializing, and `cna_game_tick` — which does
not initialize — polls events too, so it is not that route either.

A second, smaller difference in the same family: CNA's `Tick` calls `PollEvents`
and XNA's does not, because in XNA the pump belongs to the window and
`RunOneFrame` is the member that reaches it. CNA-Go reports what CNA does.

## Ownership: whoever created the native game destroys it

Three moments create or end a session, and the rule between them is one
sentence.

| caller | if no session | if this Runtime already has one |
| -- | -- | -- |
| `Run` | starts one, runs, **ends it** | adopts it, runs, **leaves it** |
| `Tick` / `RunOneFrame` | starts a **standalone** one | steps it |
| `Dispose` | nothing | ends it, **only if standalone** |

Adopting is the reference's own shape rather than a convenience. XNA's `Run`
calls `host.Run()` on a host the constructor already created, and CNA's
`Game::Run` skips `DoInitialize` when `hasInitialized_` is already set — so a
stepped-then-run Game keeps one native game and one initialization. Twenty
isolated cycles prove exactly that: `RunOneFrame`, then `Run`, one
initialization total, and the session still alive when `Run` returns.

### Why `Dispose` ends it

`Game::Dispose(bool)` in the reference disposes components and raises
`Disposed`; it does not touch the host, which the process owns until it exits.
CNA-Go cannot copy that, because CNA admits one C-owned game per process: a
standalone session nothing ended would make the next `Game` impossible to
create.

The step is therefore added, and added narrowly:

- it runs **after** the reference's managed body, so every component's own
  `Dispose` and every `Disposed` handler observes exactly what it observed
  before, and the managed order the corpus and the canary measure is unchanged;
- it runs **only** when a standalone session is live, a state the reference
  cannot have, so a Game that only ever used `Run` sees a byte-for-byte
  unchanged `Dispose`;
- it is not a divergence from the reference's `Dispose` at all. The reference's
  host is created by the **constructor**; CNA-Go's is created by a frame step.
  Ending it is the disposal of a resource whose creation moved, not the disposal
  of one the reference keeps.

A frame step **after** `Dispose` starts a fresh session. That follows from a
measured reference fact rather than from convenience: `Game` carries no disposed
flag anywhere, which is the same reason `Dispose` is not idempotent in this
profile. The consequence is recorded: the new session is a new native game, so
its clock and its initialization state start again.

## `Exit` outside a callback

`Runtime.Exit` required an active lifecycle callback. That was correct while the
only way to reach a live native game was from inside one — `Run` blocks the owner
thread in `cna_game_run`, so outside a callback there was nothing live to ask.

A standalone session makes "live, on the owner thread, outside a callback" a
reachable state, and it is exactly the state a frame-stepped consumer calls
`Exit` from. CNA agrees: `cna_game_request_exit` resolves the game with
`GetGame` rather than `GetCallableGame`, so it carries no callback restriction
of its own. The requirement was relaxed to match, and nothing about the
`Run`-driven path changes, because outside a callback during a run there is no
live game on the owner thread to reach anyway.

Measured consequence, recorded because it surprises: CNA's request-exit sets
`suppressDraw_` as well as clearing `RunApplication`, so the frame **after**
`Exit` updates and does not draw. Twenty cycles assert it.

## Threading

The goroutine that takes the first frame step owns the session's OS thread for
its whole life. Every later step, `Run`, and `Dispose` must come from that same
goroutine; anything else reports `ErrWrongThread`, proved twenty times from a
second goroutine.

CNA does **not** thread-check `cna_game_run`, `cna_game_tick` or the window
routes — that was read from `GetGame` and `GetCallableGame` in
`CnaCApiRuntime.cpp`, neither of which compares a thread. So the owner-thread
rule here is CNA-Go's own, applied for the same reason it is applied everywhere
else, rather than a refusal being reported.

A second `Runtime` that tries to start a session while a **standalone** one is
live fails fast with a diagnostic instead of blocking on the process mutex. A
standalone session has no bounded duration — it lives until `Dispose` — so
waiting on it would be a hang rather than a queue. A session `Run` owns is
bounded by `Run`, so waiting for that one is exactly right and is unchanged.

## The Run refactor is behaviour-preserving

`Run`'s body moved into `startSession`/`endSession` without changing one step's
order, including the one that is easy to get wrong: the event registrations are
released only **after** `cna_game_destroy`, because CNA raises the disposal
signal from inside destroy and a registration released first would silently drop
it.

The proof is that the eight pre-existing native stress scenarios reproduce
**every counter identically** — all sixty of them, byte for byte — and
`cna-go-template` still runs at exactly 60 and 600 native draw callbacks.

## Evidence

Two new stress scenarios, twenty isolated cycles each:

```text
GAME_FRAME_STEP_CYCLES                            20
GAME_FRAME_STEP_TICKS                             80
GAME_FRAME_STEP_RUN_ONE_FRAMES                    40
GAME_FRAME_STEP_INITIALIZATIONS                   20   exactly one per session
GAME_FRAME_STEP_TICK_DOES_NOT_INITIALIZE_CHECKS   20
GAME_FRAME_STEP_UPDATE_DELIVERIES                120   six steps per cycle
GAME_FRAME_STEP_DRAW_DELIVERIES                   80   two of six suppressed
GAME_FRAME_STEP_SUPPRESS_DRAW_CHECKS              20
GAME_FRAME_STEP_EXIT_CHECKS                       20
GAME_FRAME_STEP_WRONG_THREAD_CHECKS               20
GAME_FRAME_STEP_CALLBACK_REFUSAL_CHECKS           20
GAME_FRAME_STEP_SESSION_LIFETIME_CHECKS           60
GAME_FRAME_STEP_DISPOSE_CHECKS                    20
GAME_FRAME_STEP_RECREATION_CHECKS                 20
GAME_FRAME_STEP_RUN_ADOPTS_SESSION_CYCLES         20
```

`120` updates for `80` draws is the SuppressDraw and post-Exit evidence in one
number, and the verifier requires draws to be strictly fewer than updates rather
than equal to a constant — a scenario in which nothing was ever suppressed would
fail.

Neither the behavior corpus nor the external canary takes a frame, and the
reason is recorded in both: a frame step creates the process's one C-owned
native game and holds it until `Dispose`, so a corpus that stepped one would
decide the outcome of every later row in the same process. They pin the
signatures and the Go-only unconstructed-Game guard; the live evidence is the
stress tool, which isolates every cycle in its own subprocess for exactly that
reason.

## ABI

```text                                        before      after
BOUND_FUNCTIONS                               41      43
PROTOTYPE_TYPE_POSITIONS                     128     132
native ABI mutation controls                  59      63
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         117     117
ABI_MISMATCHES / FINDINGS                      0       0
```

`cna_game_tick` and `cna_game_run_one_frame` are both
`CNA_Result(CNA_Handle)` — the same shape as `cna_game_run`,
`cna_game_request_exit` and `cna_game_destroy`. A manifest that bound one where
another belongs would compile cleanly through every static check, and only the
loader's `dladdr` identity check separates them. That check was added in
Foundation 44 for precisely this class of route and this is the first family to
depend on it.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1860     1862
MISSING_MEMBER                               145      143
TOTAL_DIAGNOSTICS                            278      276
COMPLETE_TYPES                               119      119
PARTIAL_TYPES                                  5        5

behavior corpus                              661      663
external canary tests                         72       73
native stress scenarios                        8       10
native ABI mutation controls                  59       63
runtime capability rows                       46       47
```

`Game`'s four remaining missing members are `Content`, `LaunchParameters`,
`IsActive` and `ShowMissingRequirementMessage`, and none of them is a lifecycle
question any more.

## What this milestone does not claim

- **`Tick` is not `RunOneFrame` and neither is `Run`.** They are three distinct
  members with three distinct behaviours, and the corpus and the canary both
  assert that all three exist separately.
- **A frame-stepped Game and a Run Game are one lifecycle, not two.** `Run`
  adopts a standalone session rather than making a second one, so there is no
  second creation path, no second initialization and no second destruction
  owner.
- **Nothing claims a visible frame.** The artifact is still HEADLESS; what is
  proved is that a frame step reaches the same callbacks, the same frame hooks
  and the same suppression the loop does.
- **A frame step after `Dispose` starts a NEW native game.** The reference would
  have kept its host and its clock. That consequence is measured, asserted and
  recorded rather than papered over with a disposed flag the reference does not
  have.
