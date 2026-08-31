# Foundation 38 — the optional per-hook frame-boundary override mechanism

Foundation 35 projected `Game::BeginRun`, `EndRun`, `BeginDraw` and `EndDraw` as
base bodies on `Game`. Foundation 36 measured the four canonical CNA hooks that
sit at the same frame positions and recorded that CNA-Go installed none of them,
because there was no way for a consumer to override the four virtuals and
installing a hook would have run a base body at a position CNA-Go picked.

This milestone supplies the missing half. A consumer may now override **any
subset** of the four, and each canonical hook is installed **if and only if**
that override exists.

## The decision, and what it is not

In CLR a derived `Game` may override any subset of these four virtuals. That is
the requirement, and it rules out three shapes that all look reasonable:

| rejected shape | why |
| -- | -- |
| four more members on `GameCallbacks` | breaks every external implementation of the five-member contract |
| one optional interface carrying all four | too coarse: a consumer who overrode one virtual would have to supply three no-ops, and a no-op override is **not** the same as no override — it installs a hook and takes the base's place |
| explicit registration functions | mutable per-`Game` callback state nothing else in the binding has, and an override set that can change under a running frame loop |

What was chosen instead: **four optional structural capabilities on the same
object already passed to `NewGame`**.

```go
type gameBeginRunOverride   interface{ BeginRun(*Game) error }
type gameEndRunOverride     interface{ EndRun(*Game) error }
type gameBeginDrawOverride  interface{ BeginDraw(*Game) (bool, error) }
type gameEndDrawOverride    interface{ EndDraw(*Game) error }
```

All four are **unexported**. Go interfaces are structural, so a consumer
satisfies one by declaring the exported method and never names the type:

```go
func (c *Callbacks) BeginDraw(game *framework.Game) (bool, error) {
    // derived work before base
    return game.BeginDraw()
}
```

`GameCallbacks` is untouched and still has exactly five members. No new exported
framework identity is published. There is no registration, replacement or
removal operation, because a Go object's method set is fixed for its lifetime
and there is nothing to mutate.

## Discovery

Once, in `NewGame`, at the same boundary where `GameCallbacks` becomes
associated with the `Game`. The results are private fields; nothing type-asserts
per frame, and the override set a `Game` runs with is decided by the object it
was constructed from and can never change afterwards.

## Installation

`cna_go_game_create` takes a mask of `CNA_GO_FRAME_HOOK_*` derived from those
same fields, so the mask and the dispatch cannot disagree — the same nil-ness
decides both. A bit that is clear leaves the `CNA_GameFrameHooks` member NULL,
which the canonical header defines as *simply not called*. A consumer who opts
into nothing therefore observes exactly the native behaviour they observed
before this mechanism existed.

`initialize` is not in the mask. It is the position `Game::Initialize` occupies,
it is always installed, and it is not an optional override.

## Base calls, and why there is still no `GameBase...` helper

The projected methods on `Game` **are** the base bodies, so calling
`game.BeginDraw()` inside an override is the Go projection of
`base.BeginDraw()`. That is the whole projection:

* zero calls — the base does not run;
* one call — it runs once, at that source position;
* two calls — it runs twice, and nothing deduplicates or suppresses the repeat.

The call site is the semantics, exactly as it is in CLR. `GameBaseBeginRun` and
friends are still refused: the base-call registry is keyed by the `GameCallbacks`
members, and a helper for something reachable as a method on `Game` is the
invented helper that registry's closure rule exists to stop.

### Recursion is impossible, and it is proved rather than argued

`Game.BeginDraw` reads only `Game`'s own state and never consults the callback
object, so an explicit base call cannot re-enter the override. Three separate
places prove it by counting override invocations against hook deliveries: the
in-package test, the behavior corpus, and the external canary running from a
module that cannot see anything unexported.

## `BeginDraw`'s three outcomes stay three outcomes

| outcome | meaning |
| -- | -- |
| hook absent | the native/default CNA behaviour remains in control |
| `(false, nil)` | skip this frame's `Draw` **and** `EndDraw` |
| `(true, nil)` | proceed |
| `(_, err)` | the established callback-failure path; the runtime's own drawing decision is left untouched |

A Boolean false is never turned into an error, and a failure never decides the
frame. Skipping `EndDraw` on a refused frame is the native runtime's own
behaviour and is measured, not assumed.

## Measured against the pinned artifact

Two new `native_stress` scenarios, 20 crash-isolated subprocesses each:

```text
FRAME_HOOK_OVERRIDE_CYCLES                20
FRAME_HOOK_BEGIN_RUN_DELIVERIES           20     exactly one per run
FRAME_HOOK_END_RUN_DELIVERIES             20     exactly one per run
FRAME_HOOK_BEGIN_DRAW_DELIVERIES         280
FRAME_HOOK_REFUSED_FRAMES                120
FRAME_HOOK_ADMITTED_FRAMES               160
FRAME_HOOK_END_DRAW_EXPECTED              80
FRAME_HOOK_END_DRAW_DELIVERIES            80     == expected
FRAME_HOOK_REFUSED_FRAME_SKIP_CHECKS      40
FRAME_HOOK_EXPLICIT_BASE_CALL_CHECKS      40
FRAME_HOOK_ORDER_CHECKS                   40
FRAME_HOOK_SUBSET_CYCLES                  20
FRAME_HOOK_UNINSTALLED_DELIVERIES          0
```

Two of those numbers carry the whole argument.

**`FRAME_HOOK_UNINSTALLED_DELIVERIES = 0`** is the subset scenario. Its callback
object declares only `BeginDraw`; `begin_run`, `end_run` and `end_draw` were
never delivered to it, because their members were left NULL.

**The 120 refused frames** are the proof that the *override's* answer and not the
base's reaches CNA. The base `BeginDraw` admits every frame — no
`IGraphicsDeviceManager` is registered, so it falls through to its
`ldc.i4.1; ret` — and every override call was verified to have received `true`
from it. A skipped `draw` is therefore something only the override could have
caused, and each of the 120 was checked to have skipped `draw` **and**
`end_draw`.

`begin_run` is first in the delivery order and `end_run` is last, in all 20
override cycles, on the owner goroutine only.

## Verifier

The frame-hook registry gained the installation class, the base-invocation
class, and the capability identity:

```text
GAME_FRAME_HOOKS                          4
GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE    4
GAME_FRAME_HOOKS_NEVER_INSTALLED          0
GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES     4
GAME_CALLBACKS_MEMBERS                    5
```

There are exactly **two** installation classes, `NEVER` and `ON_OVERRIDE`. There
is deliberately no `ALWAYS`: an unconditionally installed hook is the automatic
base behaviour Foundation 31 refused, so it is not a state this registry can
express and declaring one is an unclassified installation.

Each `ON_OVERRIDE` hook's capability is measured against **compiler evidence**,
not against the registry's own word for it: the named type must exist in the
framework package, be unexported, be a Go interface, declare exactly one method,
and that method must be exported and have the measured signature. No two hooks
may share a capability, and the exported spelling of one must not exist beside
it. Separately, no exported member or type in the framework package may be named
`...Override`, which refuses every spelling of a registration API by shape
rather than by a list.

24 new negative controls (`f38hook_*`) drive all of it, taking the mutation
inventory from 531 to 555. They include the four the design had to be able to
reject by name: a mandatory sixth `GameCallbacks` member, a bundled multi-hook
capability, an exported capability interface, an always-installed hook, and an
anonymous callback field on `Game`.

## ABI

No CNA function was added and CNA was **not** rebuilt. `cna_game_set_frame_hooks_ext`
was already bound and already called; this milestone fills in four members of a
table CNA-Go was already passing.

```text                       before   after
BOUND_FUNCTIONS                  25      25
PROTOTYPE_TYPE_POSITIONS         75      75
C_GO_MEASUREMENTS               107     112
LAYOUTS                          31      36
CALLBACKS                         3       3
CONSTANTS                        10      15
```

What moved and why: the five `CNA_GameFrameHooks` member offsets are now
measured, because CNA-Go assigns four of them conditionally and their positions
became load-bearing. The five new `_Static_assert`s pin the table's member
**order** portably — `begin_run` follows `initialize`, `end_run` follows
`begin_run`, and so on — and the same five appear in `bridge.c`, which cgo
compiles against CNA-Go's private manifest rather than the canonical header. The
pair pins the layout from both sides: the probe fails if the canonical table
changes, the bridge fails if the manifest does.

`CALLBACKS` stays at 3 and the claim behind it became true. `CNA_GameEventCallback`
was the only one actually pinned by an assignment probe; `CNA_GameLifecycleCallback`
and `CNA_GameBeginDrawCallback` now are too. The begin-draw shape matters most:
its `CNA_Bool*` out-parameter, and its position before the error, are what decide
which frames draw.

`native_abi`'s mutation controls went from 19 to 24, including three that attack
the begin-draw out-parameter directly (dropping it, moving it past the error,
returning the decision instead) and one that collides two frame-hook mask bits.

## Evidence counters

```text                    before   after
behavior corpus              628     633
external canary tests         42      53
mutation inventory           531     555
native ABI mutations          19      24
native stress scenarios        4       6
internal/interop tests         8      12
TOTAL_DIAGNOSTICS            295     295
```

`TOTAL_DIAGNOSTICS` does not move, and no type became complete. This milestone
adds no XNA identity: the four capabilities are Go language support, the four
hooks were already projected members of `Game`, and a separate accounting
control proves neither the capabilities nor the registry inflate any identity
counter.
