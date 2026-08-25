# Foundation 36 — the native game-signal bridge and the frame-hook frontier, measured

Foundations 34 and 35 made two decisions that lived only in prose and in the
code that implemented them:

- each of `Game`'s four CLR events is bound to one canonical CNA signal, through
  the reference's own raise path, with the sender the IL pushes;
- `Game`'s four frame-boundary protected virtuals are methods on `Game`, and the
  four canonical CNA hooks at those positions are deliberately not installed.

Prose is not a verifier. This milestone turns both into closed registries the
strict run enforces, with 31 negative controls that each remove one rule and
require the diagnostic.

No projected member changed, no ABI counter moved, and the pinned artifact is
untouched. `TARGET_MEMBERS`, `MISSING_MEMBER`, `COMPLETE_TYPES`,
`REFERENCE_MEMBERS` and `EXPECTED_GO_MEMBERS` are all exactly where Foundation
35 left them: this is measurement, not surface.

## The native game-signal registry

`gameNativeSignals` is keyed by CLR event name and declares, per event: the
canonical `CNA_GAME_EVENT_*` constant and its identity, the CLR event, the
projected raise site or the explicit absence of one, the sender that raise
pushes, whether the reference's host handler is edge-triggered, the reference
path from host signal to handler, and the runtime evidence class.

```text
GAME_NATIVE_SIGNALS                  = 4
GAME_NATIVE_SIGNAL_RAISE_SITES       = 3
GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED = 1
```

Three is not two short of four. `Disposed` genuinely has no `On...` method: the
reference invokes the delegate field directly from the tail of `Dispose(bool)`.
The registry records that absence, and the verifier checks it *as an absence* —
declaring `OnDisposed` fails, because the pinned contract declares no such
member.

One is not a gap either. `Deactivated` is `NOT_RUN_ENVIRONMENT`, because the
HEADLESS qualification artifact has no window manager and can never lose focus.

### The five claims the rule defends

1. **Every CLR event Game declares has a signal, and every signal names an event
   Game declares.** Both directions are checked against the pinned contract, so
   a projected accessor pair with no raise path and a signal for an invented
   event both fail.
2. **Each event's two projected accessors are the ones the registry names**, and
   both are present in the framework package. A renamed accessor is caught here
   rather than shipping as a silently dead binding.
3. **A declared raise site is a real protected virtual of Game with the exact
   `(any, *EventArgs) error` shape**, and an absent one means the reference
   really declares none.
4. **Every `On...` member Game projects is a declared raise site**, so a raise
   path cannot be added without being measured.
5. **The runtime evidence is honest.** `NOT_RUN_ENVIRONMENT` requires a reason;
   `VERIFIED_NATIVE` forbids one. An excuse standing next to a verification means
   the label and the evidence disagree, and that is now a failure rather than a
   reading comprehension exercise.

The sender vocabulary is closed to two values, because the IL only ever pushes
two things:

```text
GAME   ldarg.0   -- the Game itself, NOT the sender parameter the On... method was handed
NULL   ldnull    -- Game::OnExiting, and nothing else in this family
```

### What the registry deliberately does not check

The C-side chain — canonical header, CNA-Go's private manifest, `bridge.h`'s
mirror, the Go constants — is not checked here, because it is already checked
somewhere better. `bridge.c` carries `_Static_assert`s tying the mirror to the
manifest, `native_linux.go` carries zero-sized array assertions tying the Go
constants to the mirror, `probe.c` compares the manifest with the canonical
header, and `tools/native_abi` runs 19 mutations over all of it. Repeating that
in Go here would be a second, weaker copy.

## The frame-hook registry

`gameFrameHooks` is keyed by Go method name and declares the CLR member, the
exact projected signature, the canonical CNA hook at the same frame position,
whether it is installed, why it is not, the measured native ordering, the
reference base body, and every deferred step.

```text
GAME_FRAME_HOOKS               = 4
GAME_FRAME_HOOKS_INSTALLED     = 0
GAME_FRAME_HOOK_DEFERRED_STEPS = 4
```

`GAME_FRAME_HOOKS_INSTALLED = 0` is the point. It is a measured zero with four
recorded reasons behind it, not an absence of information.

### The four claims

1. **Each declared hook names a real protected virtual of `Game`**, read from
   the pinned contract, projected onto receiver `Game` with the registry's exact
   parameters and results.
2. **None of them is a `GameCallbacks` member.** The mandatory override contract
   keeps exactly its five members; a frame hook that joined it would break every
   existing external implementation.
3. **None of them has a `GameBase...` helper.** The base-call registry is keyed
   by the `GameCallbacks` members, so `GameBaseBeginRun` and friends are exactly
   the invented helpers its closure rule exists to stop. This rule states it from
   the other side and checks the framework package for one.
4. **An uninstalled hook records why**, and every unreproduced reference step is
   classified `SUBSYSTEM`, `ARCHITECTURE` or `UNOBSERVABLE` with a reason — the
   same rule the base-call adapters follow, including the one that matters most:
   **a deferral marked observable is rejected**, because that is a stop condition
   rather than a deferral.

The signature check is where `BeginDraw`'s Boolean is pinned. The documented
signature must spell the registry's exact results, so
`func (g *Game) BeginDraw() (bool, error)` cannot quietly become
`func (g *Game) BeginDraw() error` — collapsing the frame's drawing decision
into the error channel — without both the registry and `mapping-rules.json`
saying so, and a drift guard compares those two.

## 31 negative controls

Each mutation removes exactly one rule and must raise
`LANGUAGE_MAPPING_MISMATCH`. They are driven from one shared table, so the named
test and the mutation inventory cannot drift; a guard test fails if either side
gains an entry the other lacks.

**Native signal bridge (16).** An event left with no signal; a signal for an
event Game does not declare; two events sharing one CNA identity; a signal that
names no CNA constant; a raise site that is not a projected method; an invented
`OnDisposed`; a raise site dropped for an event that has one; a raise site whose
projected shape is not `(any, *EventArgs) error`; a raise site absent from the
package; an accessor absent from the package; a sender that is neither `GAME`
nor `NULL`; a signal with no reference path; an unrecognised evidence class; an
unverified signal with no reason; a verified signal carrying an excuse; and an
`On...` member on Game that no signal declares.

**Frame-hook frontier (15).** A hook absent from the package; a hook for a
member Game does not project; a signature mismatch; a `GameBaseBeginRun` helper
appearing; a hook declared for one of the five `GameCallbacks` members; an
uninstalled hook with no reason; an installed hook still carrying the excuse for
not installing it; a hook naming no native hook; a hook with no measured native
ordering; a hook with no reference body; the three deferral rules
(unclassified, no reason, marked observable); a hook projected onto another
receiver; and a hook naming a public rather than protected member.

Two accounting controls sit alongside them. One proves the unmutated fixture is
clean and measures exactly what the registries declare — 4 signals, 4 hooks, 0
installed. The other proves neither registry is an XNA identity:
`REFERENCE_MEMBERS`, `EXPECTED_GO_MEMBERS` and `Game`'s projected member set are
all unchanged by measuring them.

```text
mutation inventory   500 -> 531
```

## Scoreboard

```text                                          before   after
GAME_NATIVE_SIGNALS                            —        4
GAME_NATIVE_SIGNAL_RAISE_SITES                 —        3
GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED           —        1
GAME_FRAME_HOOKS                               —        4
GAME_FRAME_HOOKS_INSTALLED                     —        0
GAME_FRAME_HOOK_DEFERRED_STEPS                 —        4

mutation inventory                            500      531

TARGET_MEMBERS                               1791     1791
TOTAL_DIAGNOSTICS                             295      295
MISSING_MEMBER                                160      160
COMPLETE_TYPES / PARTIAL_TYPES             117 / 5  117 / 5
REFERENCE_MEMBERS / EXPECTED_GO_MEMBERS  2964 / 3255  unchanged
ABI (25/75/107/31/3/10)                          unchanged
```

Every mismatch, leak, allowlist and unmeasured counter is zero, and the pinned
native artifact is byte-identical.
