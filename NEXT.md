# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 36 are complete. Milestones 34 through 36 were
produced in one session, as the three milestone commits listed below plus the
docs commit that records them. Whether they have reached `origin/develop` is a
live fact rather than a stored one — see the note under the block.

```text
SESSION START HEAD          = 605b350a509af8399f742be989a580fb83db22cf
                              (== origin/develop at session start)
LAST SOURCE-BEARING COMMIT  = 16181b0
SESSION END                 = the develop HEAD this file is committed in;
                              resolve with `git rev-parse HEAD`
PUSHED AT COMMIT TIME       = false; resolve the live relationship with
                              `git status --short --branch`
                              worktree clean, git diff --check clean
```

**This file deliberately does not record its own commit's object id.** A commit
cannot contain the hash it will produce: writing the id in changes the content,
which changes the id. Every hash below therefore names a commit that is
**already an ancestor** when it is written; the session's own end state is
resolved from the repository, not asserted in it.

`PUSHED` follows the same rule and for the same reason. A commit cannot record
whether it was pushed, because the push happens after it exists. What is
recorded is the state at commit time, which stays true, plus the command that
resolves the live one. The deterministic artifact hash further down is pinned to
the last source-bearing commit for the same reason.

| #  | commit    | milestone                                              | types | Go identities |
| -- | --------- | ------------------------------------------------------ | ----- | ------------- |
| 34 | `9f661db` | the native Game event bridge, and Game's four events    | 0     | 11            |
| 35 | `9872de1` | Game's four frame-boundary protected virtuals           | 0     | 4             |
| 36 | `16181b0` | the signal and frame-hook registries, measured          | 0     | 0             |

Fifteen members completed, all in `Game`, all inside the authorized native
lifecycle slice. No type became complete and none was meant to.

## Scoreboard

```text                                     start    final
TARGET_TYPES                              122      122
TARGET_MEMBERS                           1776     1791
TOTAL_DIAGNOSTICS                         310      295
MISSING_TYPE                              135      135
MISSING_MEMBER                            175      160
COMPLETE_TYPES                            117      117
PARTIAL_TYPES                               5        5

REFERENCE_MEMBERS / REFERENCE_XNA_MEMBERS 2964     2964
EXPECTED_GO_MEMBERS                      3255     3255
BCL_INHERITED_MEMBER_PROJECTIONS           12       12

GAME_BASE_CALL_ADAPTERS                     5        5
GAME_BASE_CALL_DEFERRED_STEPS               4        4
DECLARED_INTERFACE_CONFORMANCE              2        2
XNA_BASE_RELATIONSHIPS                     12       12
XNA_BASE_DERIVED_TYPES                     41       41
XNA_DEFERRED_BASE_BLOCKERS                 25       25
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED  245      245

GAME_NATIVE_SIGNALS                         0        4
GAME_NATIVE_SIGNAL_RAISE_SITES              0        3
GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED        0        1
GAME_FRAME_HOOKS                            0        4
GAME_FRAME_HOOKS_INSTALLED                  0        0
GAME_FRAME_HOOK_DEFERRED_STEPS              0        4

mutation inventory                        500      531
behavior corpus                           620      628
external canary tests                      34       42
native ABI mutation controls                0       19
```

Every mismatch, leak, allowlist and unmeasured counter is zero throughout.
**`MISSING_MEMBER` moved by exactly fifteen, every one of them on `Game`**:
`Game` went from 37 missing members to 22. No other partial type was touched.

Per-type missing members now: `GraphicsDevice` 70, `GraphicsDeviceManager` 40,
`Game` 22, `SpriteBatch` 16, `Texture2D` 12.

## The ABI moved because CNA-Go binds more, not because CNA changed

```text                       before   after
BOUND_FUNCTIONS                  23      25
PROTOTYPE_TYPE_POSITIONS         67      75
C_GO_MEASUREMENTS                96     107
LAYOUTS                          28      31
CALLBACKS                         2       3
CONSTANTS                         5      10

MISSING_HEADER_SYMBOLS            0       0
MISSING_LIBRARY_SYMBOLS           0       0
ABI_MISMATCHES                    0       0
```

Exactly what was added: `cna_game_subscribe` (5 parameters) and
`cna_game_unsubscribe` (1), so `PROTOTYPE_TYPE_POSITIONS` grows by
`(1+5)+(1+1)=8`; `CNA_GameEventCallback`; three layout measurements; and five
`_Static_assert`s over the four `CNA_GAME_EVENT_*` identities and
`CNA_GAME_EVENT_MAXIMUM`. Every pre-existing entry is byte-identical.

**CNA was not rebuilt and no CNA C++ changed.** Both symbols were already
exported by the pinned artifact and had simply never been reached from Go:

```text
cna_game_subscribe     00000000006f0ab0 T cna_game_subscribe@@CNA_C_API_0.1
cna_game_unsubscribe   00000000006f1020 T cna_game_unsubscribe@@CNA_C_API_0.1
```

## What the four events are, exactly

All four are `System.EventHandler\`1<System.EventArgs>`, so all four take the
settled two-accessor projection unchanged. Three are raised by `GameHost`,
whose part the native CNA runtime plays; the fourth is raised by managed
disposal.

| CLR event | CNA signal | raise site | sender | edge-triggered | runtime |
| -- | -- | -- | -- | -- | -- |
| `Activated` | `CNA_GAME_EVENT_ACTIVATED` (0) | `OnActivated` | the Game | yes | VERIFIED_NATIVE |
| `Deactivated` | `CNA_GAME_EVENT_DEACTIVATED` (1) | `OnDeactivated` | the Game | yes | **NOT_RUN_ENVIRONMENT** |
| `Disposed` | `CNA_GAME_EVENT_DISPOSED` (2) | *(none)* | the Game | no | VERIFIED_NATIVE |
| `Exiting` | `CNA_GAME_EVENT_EXITING` (3) | `OnExiting` | **null** | no | VERIFIED_NATIVE |

## Reference quirks now preserved — do not "correct" them

Everything in the Foundation 17-33 list still holds, plus:

- **`Game::OnExiting` raises with a NULL sender.** `ldnull`, not `ldarg.0`, and
  not the `sender` parameter it was handed. Its two siblings raise with `this`.
  One IL instruction is the whole difference and every handler can see it.
- **`Game::Disposed` has no `On...` method.** `Dispose(bool)` invokes the
  delegate field directly. Inventing an `OnDisposed` is now a verifier failure.
- **`HostActivated`/`HostDeactivated` are edge-triggered on the private
  `Game::isActive` field, and set it BEFORE they announce.** A handler always
  observes a Game that has already recorded the transition.
- **`Game::get_IsActive` is a different thing and stays missing.** It reads
  `GamerServicesDispatcher.IsInitialized` and `Guide.IsVisible` from
  `Microsoft.Xna.Framework.GamerServices.dll`, which is not one of the seven
  pinned assemblies. The private field is reproduced; the public getter cannot be.
- **`HostExiting` has no guard**, so every exit signal raises.
- **`Game::BeginRun` and `EndRun` are `IL_0000: ret`.** `RunGame` raises `inRun`
  *before* it calls `BeginRun`, so the flag is the caller's business, not the
  virtual's.
- **`DrawFrame` runs `if (BeginDraw()) { Draw(); EndDraw(); }`** — a false answer
  skips **both**. The measured CNA `begin_draw` hook does the same.
- **`Game::graphicsDeviceManager` has exactly one assignment in the whole
  class**, at the head of `RunGame`, and the statement after it calls
  `CreateDevice`. It is permanently null in CNA-Go, which is a state the
  reference itself has whenever no manager is registered.

## One native subscription per event, and why

CNA invokes multiple native registrations on one event in **reverse
registration order** — measured against the pinned artifact, two handlers on
`ACTIVATED` delivered second-then-first. Registering one native callback per Go
handler would therefore have silently inverted the dispatch order the event
projection promises, since CLR runs a multicast invocation list in registration
order.

One native subscription per event per Game makes the question moot and buys
something else: `Add` and `Remove` touch a mutex-guarded Go list and never reach
C, so a consumer may subscribe from any goroutine even though
`cna_game_subscribe` itself answers `CNA_RESULT_THREAD` (8) from any thread but
the owner.

### Lifetime

```text
cna_game_create            -> subscribe 4          (owner thread, before the loop)
cna_game_run                                        ACTIVATED / DEACTIVATED / EXITING
dispose owned resources
cna_game_destroy                                    DISPOSED
unsubscribe 4                                       (registrations survive destroy)
Runtime.deactivate()
cgo.Handle.Delete()                                 (strictly last)
```

Releasing before destroy would silently drop `Disposed`: the signal is raised
from *inside* `cna_game_destroy`, and a registration handle stays valid across
it (measured). The `cgo.Handle` is deleted after every release, so
callback-after-free is unreachable. A second release passes nothing to CNA
rather than a stale handle, which CNA answers with `CNA_RESULT_INVALID_HANDLE`.

### The error boundary

`CNA_GameEventCallback` returns `void`, so this boundary **cannot stop the
game**, and the canonical header says so: *"The exiting callback in
CNA_GameCallbacks is a different thing: it can stop the game by failing, while
these handlers only observe."*

A handler failure is recorded through the same callback-failure path every
lifecycle failure uses and surfaces from `Game.Run`; a panic is recovered in the
trampoline and recorded the same way. Nothing is discarded and nothing crosses
the C frame. `inCallback` is deliberately **not** raised: an observation point is
not an operation point, and `Disposed` in particular arrives while the native
game is being torn down.

## Measured native ordering

```text
cna_game_run
  initialize hook -> load_content -> begin_run hook -> ACTIVATED
  update -> begin_draw -> draw -> end_draw   (per frame)
  cna_game_request_exit
  exiting callback -> EXITING -> end_run hook
cna_game_run returns
cna_game_destroy
  unload_content -> DISPOSED
cna_game_destroy returns
```

Against `RunGame`: `CreateDevice -> Initialize -> inRun = true -> BeginRun ->
Update -> host.Run(){loop} -> EndRun`. Every position corresponds, and `EXITING`
precedes `end_run` exactly as `HostExiting` fires from inside `host.Run()`
before `EndRun()` runs.

Every signal is delivered on the owner thread. A double
`cna_game_request_exit` delivers `EXITING` exactly once. Setting
`out_should_draw` to `CNA_FALSE` delivered neither `draw` nor `end_draw`.

## The frame hooks are audited and deliberately uninstalled

`CNA_GameFrameHooks` carries five members; CNA-Go installs exactly one,
`initialize`, and has since Foundation 11. `begin_run`, `end_run`, `begin_draw`
and `end_draw` correspond position for position to the four projected virtuals
and are **not** installed:

1. **Base behavior is never automatic** (Foundation 31). Forwarding a hook into
   a base body would run the base where CNA-Go picked, make that base call
   mandatory, and prejudge the override design.
2. **It would be inert.** With no reachable `IGraphicsDeviceManager` all four
   base bodies are observably empty, and two of the hooks fire per frame.
3. **Nothing is hidden.** `GAME_FRAME_HOOKS_INSTALLED = 0` is a measured zero
   with four recorded reasons behind it.

## What is genuinely blocked, and the exact next decisions

### 1. The frame-hook override mechanism — STOPPED and reported

If a consumer is ever to *override* `BeginRun`, `EndRun`, `BeginDraw` or
`EndDraw`, CNA-Go needs a mechanism, and more than one materially different
public shape remains plausible. Per the stop rule that branch was stopped rather
than guessed.

Settled and not to be reopened: **`GameCallbacks` keeps exactly five members**
(adding four would break every existing external implementation, and two tests
hold it), and **no `GameBase*` helper** for any of the four (the registry's
closure rule rejects a helper for anything that is not a `GameCallbacks`
member).

The three candidates, none uniquely determined by existing style:

1. an **optional secondary interface** a `GameCallbacks` implementation may also
   satisfy, discovered by a type assertion at `NewGame` — additive, but it adds
   a second callback contract and must answer what a partial implementation means;
2. **per-hook capability interfaces**, one single-method interface each — finer
   grained, but four more exported contracts;
3. **explicit registration** functions that install one override at a time —
   single-purpose contracts and an observable installation act, but it introduces
   mutable per-Game callback state nothing else in the binding has.

Each must separately decide whether the native hooks are installed eagerly or
only once an override exists, and what `BeginDraw`'s Boolean means with no
override present.

### 2. `Game`'s disposal surface — a real fork, deliberately not taken

`Game::Dispose()`, `Dispose(Boolean)` and `Finalize` are still missing, and
completing them is **not** a mechanical follow-up. In CLR, `Game.Disposed` fires
only from `Dispose(bool)`, so a consumer who never disposes never sees it. In
CNA-Go it fires from the native disposal signal at the end of `Run`.

Projecting managed `Dispose` would give `Disposed` **two** raise paths. Whoever
takes this on must choose one:

- keep the native signal and give managed `Dispose` no raise (diverges from the
  reference's raise site), or
- drop the native binding and raise only from managed `Dispose` (matches the
  reference exactly, but `Disposed` then never fires unless a consumer disposes,
  and the native runtime's own disposal becomes unobservable).

This session took neither: the instruction was not to add a second synthetic
raise where CNA already raises the canonical one, so the native signal is bound
and the divergence is recorded rather than smoothed. The reference body also
needs `IDisposable` component disposal, the device-manager disposal step and
`UnhookDeviceEvents`, the last two of which are already-known deferrals.

### 3. XNA-to-XNA class inheritance — unchanged since Foundation 33

Still the largest architecture decision, and still untouched: 12 relationships,
41 derived types, 25 blockers, 245 unprojected inherited public members. It
needs a composition and forwarding rule for a base that is itself an XNA
identity, a **third provenance class** so no member is counted twice, and an
override adapter for the base's protected virtuals. `DrawableGameComponent` is
blocked five ways and `GamerServicesComponent` three; neither was made complete
to move the scoreboard.

### 4. `Game`'s remaining 22 members

`Window`, `Content` ×2, `LaunchParameters`, `GraphicsDevice`, `IsActive`,
`IsMouseVisible` ×2, `InactiveSleepTime` ×2, `TargetElapsedTime` ×2,
`IsFixedTimeStep` ×2, `Tick`, `RunOneFrame`, `SuppressDraw`, `ResetElapsedTime`,
`ShowMissingRequirementMessage`, `Dispose` ×2, `Finalize`. None was in this
session's authorized slice. `IsActive` and `Window` carry known blockers
(GamerServices and the missing `GameWindow` type respectively).

## Two new verifier rules

### The native game-signal registry

> every CLR event `Game` declares is bound to exactly one canonical CNA signal,
> through exactly the raise path the reference uses

Both directions are checked; a declared raise site must be a real protected
virtual with the exact `(any, *EventArgs) error` shape; an *absent* raise site
must genuinely be absent from the pinned contract; every `On...` member `Game`
projects must be a declared raise site; and the runtime evidence must be honest —
`NOT_RUN_ENVIRONMENT` requires a reason and `VERIFIED_NATIVE` forbids one.

### The frame-hook registry

> each of `Game`'s four frame-boundary virtuals is a method on `Game`, records
> the canonical CNA hook at the same position, and records whether it is
> installed

It also states the base-call closure from the other side — no `GameBaseBeginRun`
may exist — and pins `BeginDraw`'s Boolean as a channel separate from its error,
in the registry and in `mapping-rules.json`, which a drift guard compares.

31 negative controls drive both rules from one shared table, plus two accounting
controls proving neither registry inflates any XNA identity counter.

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~290s under -race
go build ./... && go build -trimpath ./...
go run ./tools/api_compat --mode strict -report "" -missing ""   # expected red: 295 deferred
go run ./tools/api_compat --mode leak-only -report "" -missing ""
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -headers ~/deps/cna-c-abi-0.7.0/include \
    -library "$CNA_NATIVE_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
go run ./tools/api_compat --mode report     # run LAST so committed evidence keeps report mode
git diff --check
```

Pass `-report "" -missing ""` on any non-report run so committed evidence keeps
report mode, and send native reports to an explicit scratch `-output`. The
committed `native-abi-report.json` stores `header_root` **normalized** as
`canonical-cna/modules/c-api/include`; a fresh run writes the absolute path, and
that one field is the only difference.

`tools/native_abi` now also carries a Go test: 19 mutations that each remove one
ABI pin from real source and require the compile to fail, plus two unmutated
controls. It needs `gcc` and the canonical headers, and skips rather than fails
when `CNA_C_API_INCLUDE` is unset and `~/deps/cna-c-abi-0.7.0/include` is absent.

The IL caches live at `~/deps/xna-il-cache/` — the seven XNA disassemblies plus
`mscorlib.il` (42 MB, from the admitted 4.0.30319.1 binary). They are reusable;
do not regenerate them into a scratch directory.

### Deterministic artifact, isolated consumer, external canary

```sh
git ls-files -z | sort -z | tar --null --files-from=- \
  --transform 's,^,cna-go/,' --owner=0 --group=0 --numeric-owner \
  --mtime='@0' --mode='u+rw,go+r,go-w' --sort=name -cf - | gzip -n > OUT.tar.gz
```

At `16181b0`, the last source-bearing commit, that yields sha256
`d9361e2772b6b1b64659d800ed5a51b58f1ed494326b9cfbf468a31f75ffcb43` over 282
entries, reproduced twice in the same run. The docs commit that follows it edits
tracked files and adds none, so the entry count is unchanged and the artifact
hash is not; re-run the command above for the current value rather than trusting
a literal measured over a file that is still being written.

Extracted into `build-consumer/isolated`, it passes every gate with no
development-checkout dependency and regenerates `api-compat-report.json`,
`behavior-corpus-report.json`, `packed-vector-exhaustive-report.json`,
`missing-type-inventory.md` and `runtime-capabilities.md` **byte-identically**.

The external canary now runs **42** tests:

```sh
go run ./tools/external_consumer -source build-consumer/isolated/cna-go
```

The seven new ones prove, from a module that cannot see `internal/interop` at
all: all eight event accessors reachable with tokens that round-trip; the
`Exiting` null-sender quirk; duplicate registrations removed separately and
every absent token inert; no `uintptr`, `unsafe.Pointer`, `cgo.Handle` or
`interop.` anywhere in the family's signatures; a Game still fully usable after
every handler is removed; all four frame hooks with `BeginDraw`'s Boolean
separate from its error; and `GameCallbacks` still exactly five members, checked
against a conformer written before any of this existed.

## Native provenance — unchanged

CNA was **not rebuilt**, and no C ABI function was added to CNA.

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED          25/75/107/31/3/10, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED     native_stress reproduces every counter byte-identically
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED
```

`native_stress` gained six counters and the minimums that go with them:

```text
GAME_EVENT_ACTIVATED_DELIVERIES     60
GAME_EVENT_DEACTIVATED_DELIVERIES    0     <- HEADLESS cannot produce one
GAME_EVENT_EXITING_DELIVERIES       60
GAME_EVENT_DISPOSED_DELIVERIES      60
GAME_EVENT_ORDER_CHECKS             20
GAME_EVENT_REMOVAL_CHECKS           60
GAME_EVENT_OWNER_THREAD_CHECKS      60
```

The deactivation zero has **no minimum** and is deliberately left at zero.
Inventing a way to move it would be fabricating evidence.

## Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean.
`SOURCE_CHANGED=NO`. It builds against the live checkout and runs at exactly 60
and 600 native Draw callbacks with the event bridge installed, as does the
Foundation-1 consumer fixture in `build-consumer/consumer` against the extracted
artifact. Ruby and Swift sibling worktrees were not touched.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage and most of XNA remain
unimplemented.

What is new and what it does **not** claim: a `Game` now announces activation,
exit and disposal to Go handlers, driven by the real native runtime — but
`Deactivated` has never been delivered in this environment and is not claimed to
work; the four frame-boundary virtuals exist as base bodies that nothing calls
automatically, and **no override mechanism for them exists**; and `Game.Disposed`
fires where the native game is disposed rather than where a consumer calls
`Dispose`, because `Dispose` is not projected.

```text
FOUNDATION_MILESTONE_34_COMPLETE=true
FOUNDATION_MILESTONE_35_COMPLETE=true
FOUNDATION_MILESTONE_36_COMPLETE=true
PUSHED_AT_COMMIT_TIME=false
PUSH_STATE=RESOLVE_WITH_GIT_STATUS_SHORT_BRANCH
GAME_NATIVE_LIFECYCLE_SLICE=EXHAUSTED
NEXT_STEP=ARCHITECTURE_DECISION_XNA_TO_XNA_INHERITANCE
NEXT_STEP_ALTERNATIVE_1=FRAME_HOOK_OVERRIDE_MECHANISM_THREE_CANDIDATES_REPORTED
NEXT_STEP_ALTERNATIVE_2=GAME_DISPOSAL_SURFACE_TWO_RAISE_PATHS_MUST_BE_RECONCILED
```
