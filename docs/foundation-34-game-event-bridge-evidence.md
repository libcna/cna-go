# Foundation 34 — the native Game event bridge, and Game's four events

This milestone binds a CNA C API surface that already existed and had never been
reached from Go, and uses it to close eleven of `Microsoft.Xna.Framework.Game`'s
missing members.

**No CNA C++ changed and CNA was not rebuilt.** The pinned native artifact is
byte-identical to the one every milestone since Foundation 11 has used:

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f
```

Every ABI counter that moved moved because CNA-Go now *binds* more of that
unchanged binary. That distinction is the whole point of the section on the ABI
delta below.

## What the reference declares

`Microsoft.Xna.Framework.Game.dll`, the pinned XNA 4.0 Windows profile:

```text
.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Activated
.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Deactivated
.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Exiting
.field private [mscorlib]System.EventHandler`1<[mscorlib]System.EventArgs> Disposed
```

All four are `System.EventHandler\`1<System.EventArgs>`, so all four take the
settled Foundation-22 projection unchanged: `EventHandler[*EventArgs]`,
`EventSubscription`, and exactly two Go accessors per CLR event. Each `add_`/
`remove_` pair is the ordinary compiler-generated `Delegate.Combine`/
`Delegate.Remove` with an `Interlocked.CompareExchange` retry loop — 41 bytes
each, identical across all four — so there is no per-event behavior in the
accessors themselves.

Three protected virtual raise sites go with them, and the fourth event has none.

### `OnActivated` and `OnDeactivated` drop their sender

```text
OnActivated(object sender, EventArgs args)     // code size 22
  ldarg.0; ldfld Activated; brfalse.s RET
  ldarg.0; ldfld Activated
  ldarg.0                  // <- `this`, NOT the sender parameter
  ldarg.2                  // the args, forwarded unchanged
  callvirt EventHandler`1::Invoke(object, EventArgs)
```

`OnDeactivated` is byte-for-byte the same over the other field. The `sender`
parameter is accepted, ignored and replaced by `this` — the same shape
`GameComponent::OnEnabledChanged` has, and for the same reason: the framework
raises with the instance so a handler can identify it.

### `OnExiting` raises with a NULL sender

```text
OnExiting(object sender, EventArgs args)       // code size 22
  ldarg.0; ldfld Exiting; brfalse.s RET
  ldarg.0; ldfld Exiting
  ldnull                   // <- NULL, not `this` and not the parameter
  ldarg.2
  callvirt EventHandler`1::Invoke(object, EventArgs)
```

One IL instruction is the entire difference between this method and its two
siblings, and it is observable from every handler. It is reproduced exactly: a
CNA-Go `Exiting` handler receives a nil sender. Nothing in the reference ever
gives this event a sender.

### `Disposed` has no `On...` method at all

The raise is inline at the tail of `Dispose(bool)`:

```text
Dispose(bool disposing)                        // code size 149
  if (!disposing) return;                      // NOT idempotent: no isDisposed guard
  lock (this) {
      IGameComponent[] copy = new IGameComponent[gameComponents.Count];
      gameComponents.CopyTo(copy, 0);
      foreach (c in copy) if (c is IDisposable d) d.Dispose();
      if (graphicsDeviceManager is IDisposable d2) d2.Dispose();
      UnhookDeviceEvents();
      if (Disposed != null) Disposed(this, EventArgs.Empty);   // <- `ldarg.0`
  }
```

So `Disposed` is raised with the Game, with `EventArgs.Empty`, **after** the
components and the device manager have been disposed, and there is no protected
method to route it through. That is preserved: CNA-Go raises the `Disposed`
EventSource directly and declares no `OnDisposed`, because Microsoft declared
none.

## Where the three host-sourced events actually come from

`Game`'s constructor calls `EnsureHost()` before it even allocates
`gameComponents`, and `EnsureHost` subscribes six of `Game`'s own private
methods to `GameHost`:

```text
host.Activated   += HostActivated
host.Deactivated += HostDeactivated
host.Suspend     += HostSuspend
host.Resume      += HostResume
host.Idle        += HostIdle
host.Exiting     += HostExiting
```

`GameHost` is precisely the role the native CNA runtime plays in CNA-Go. Three
of the four events are therefore **host-sourced**, and CNA already publishes all
three signals — plus a disposal signal — as canonical, shipped C API surface
that CNA-Go had simply never bound.

The two activation handlers are edge-triggered and record state before they
announce:

```text
HostActivated(sender, e)                       // code size 29
  if (isActive) return;                        // already active -> nothing at all
  isActive = true;                             // state FIRST
  OnActivated(this, EventArgs.Empty);          // then announce

HostDeactivated(sender, e)                     // code size 29, exact mirror
  if (!isActive) return;
  isActive = false;
  OnDeactivated(this, EventArgs.Empty);

HostExiting(sender, e)                         // code size 13, NO guard
  OnExiting(this, EventArgs.Empty);
```

`Game::isActive` is a private bool that the constructor never assigns, so a
Game starts inactive and the first activation is a real transition.

**`Game::get_IsActive` is a different thing and remains a missing member.** Its
body reads `GamerServicesDispatcher.get_IsInitialized()` and `Guide.get_IsVisible()`
from `Microsoft.Xna.Framework.GamerServices.dll`, which is not one of the seven
pinned assemblies. The private field is reproduced because it is what makes the
two events edge-triggered; the public getter is not, because it cannot be.

## What CNA already ships, verified against the exact pinned binary

`CNA/C/runtime.h` in the canonical 0.7.0 include tree:

```c
typedef CNA_Handle CNA_GameEventRegistrationHandle;
typedef uint32_t CNA_GameEvent;

#define CNA_GAME_EVENT_ACTIVATED    UINT32_C(0)
#define CNA_GAME_EVENT_DEACTIVATED  UINT32_C(1)
#define CNA_GAME_EVENT_DISPOSED     UINT32_C(2)
#define CNA_GAME_EVENT_EXITING      UINT32_C(3)
#define CNA_GAME_EVENT_MAXIMUM      CNA_GAME_EVENT_EXITING

typedef void (*CNA_GameEventCallback)(void* context);

CNA_C_API CNA_Result cna_game_subscribe(CNA_Handle game, CNA_GameEvent event,
                                        CNA_GameEventCallback callback, void* context,
                                        CNA_GameEventRegistrationHandle* out_registration);
CNA_C_API CNA_Result cna_game_unsubscribe(CNA_GameEventRegistrationHandle registration);
```

Both symbols are exported by the exact pinned artifact:

```text
cna_game_subscribe     00000000006f0ab0 T cna_game_subscribe@@CNA_C_API_0.1
cna_game_unsubscribe   00000000006f1020 T cna_game_unsubscribe@@CNA_C_API_0.1
```

The header is also explicit about what these handlers are, and the projection
depends on it:

> Every canonical game event carries nothing but its sender, so the handler
> receives only its context. […] The exiting **callback** in `CNA_GameCallbacks`
> is a different thing: it can stop the game by failing, while these handlers
> only observe.

## Measured native behavior

Three throwaway C probes were compiled in the shared `build-probe/` directory
and run against the pinned artifact. Everything below is measurement, not
inference.

### Delivery order and thread

```text
cna_game_run
  initialize hook              MAIN
  load_content                 MAIN
  begin_run hook               MAIN
  ACTIVATED                    MAIN   <- once, after begin_run, before update 1
  update / begin_draw / draw / end_draw ...
  cna_game_request_exit
  exiting callback             MAIN   <- CNA_GameCallbacks::exiting
  EXITING                      MAIN   <- after the exiting callback
  end_run hook                 MAIN
cna_game_run returns
cna_game_destroy
  unload_content               MAIN
  DISPOSED                     MAIN
cna_game_destroy returns
```

Every signal arrives on the game-creation thread. `EXITING` precedes the
`end_run` hook, which is exactly where CLR puts it: `HostExiting` fires from
inside `host.Run()`, and `RunGame` calls `EndRun()` only after `host.Run()`
returns.

### Subscription rules

| probe | result |
| -- | -- |
| subscribe on an owned handle before `cna_game_run` | `CNA_RESULT_SUCCESS` |
| subscribe on the callback-borrowed handle inside the `initialize` hook | `CNA_RESULT_SUCCESS` |
| subscribe from a non-owner thread | `CNA_RESULT_THREAD` (8), nothing installed |
| subscribe with identity 4, one past `CNA_GAME_EVENT_MAXIMUM` | `CNA_RESULT_INVALID_ARGUMENT` (1) |
| subscribe with a null handler | `CNA_RESULT_INVALID_ARGUMENT` (1) |
| unsubscribe a live registration | `CNA_RESULT_SUCCESS` |
| unsubscribe the same handle again | `CNA_RESULT_INVALID_HANDLE` (2) |
| unsubscribe handle 0 | `CNA_RESULT_INVALID_HANDLE` (2) |
| unsubscribe from inside that registration's own callback | `CNA_RESULT_SUCCESS` |
| unsubscribe a live registration **after** `cna_game_destroy` | `CNA_RESULT_SUCCESS` |
| `sizeof(CNA_GameEventRegistrationHandle)` / `sizeof(CNA_GameEvent)` | 8 / 4 |

Two of those decide the design.

**Registration handles outlive the game.** They stay releasable after
`cna_game_destroy`, and the disposal signal is raised *from inside* that call.
Releasing before destroy would silently drop `Disposed`. CNA-Go therefore
releases strictly after.

**Owner-thread affinity is enforced by CNA itself.** `cna_game_subscribe` from
any other thread reports `CNA_RESULT_THREAD` and installs nothing.

### CNA invokes multiple registrations in REVERSE order

Two handlers were registered on `CNA_GAME_EVENT_ACTIVATED`, first `A1` then
`A2`. The measured delivery was `A2`, then `A1`.

This is the reason for the single-subscription design. Registering one native
callback per Go handler would have inverted the dispatch order the event
projection promises — CLR runs a multicast invocation list in **registration**
order. One native subscription per event per Game makes the question moot:
ordering is decided entirely by `EventSource`, which dispatches in registration
order over a snapshot taken under its lock.

### Exactly-once

`cna_game_request_exit` called twice in one frame delivers `EXITING` exactly
once and the `exiting` callback exactly once. `DISPOSED` is delivered exactly
once per `cna_game_destroy`.

## The bridge

```text
CNA signal
  -> one of four static C trampolines (the identity comes from which function
     was registered, because CNA_GameEventCallback carries only a context)
  -> cnaGoGameEvent(event, context)      // recovers panics; nothing crosses C
  -> Runtime.invokeGameEvent(event)
  -> interop.Callbacks.GameEvent(event)  // internal interface
  -> Game.raiseNativeGameEvent(event)
  -> the reference's own raise path for that event
  -> EventSource[*EventArgs].Raise
  -> the consumer's handlers, in registration order
```

### `GameCallbacks` is untouched

The bridge method was added to the **internal** `interop.Callbacks` interface,
which only the framework package implements. The public `GameCallbacks` contract
still has exactly its five members, and a consumer implements nothing new to
receive these signals. `TestGameCallbacksContractIsStillFiveMembers` and the
external canary's five-method conformer are the compile-time proof.

### Installed eagerly, at native game creation

The reference subscribes at construction because that is where the host is
created. CNA-Go subscribes where the *native* host is created: inside `Run`,
immediately after `cna_game_create` returns.

It is deliberately not lazy. CNA rejects `cna_game_subscribe` from any thread
but the owner, while a Go consumer may add an event handler from any goroutine
at any moment, so "install on first handler" would be a call CNA can refuse.
Eager installation on the owner thread is the only point at which the native
call is guaranteed legal — and it makes `Add`/`Remove` ordinary mutex-guarded Go
calls with no thread affinity at all, which is strictly better than the native
API underneath.

### Lifetime

```text
cna_game_create            -> subscribe 4          (owner thread, before the loop)
cna_game_run                                        ACTIVATED / DEACTIVATED / EXITING
dispose owned resources
cna_game_destroy                                    DISPOSED
unsubscribe 4                                       (registrations survive destroy)
Runtime.deactivate()
cgo.Handle.Delete()                                 (strictly last: no callback can follow)
```

The `cgo.Handle` the trampolines resolve is deleted only after every
registration is released, so a callback-after-free is not reachable. The
registration slots are zeroed under the same lock that publishes them, so a
second release passes nothing to CNA rather than handing it a stale handle it
would answer with `CNA_RESULT_INVALID_HANDLE`. A subscription failure destroys
the game it could not instrument and reports the join of both results.

### The callback error boundary

`CNA_GameEventCallback` returns `void`. This boundary has no result channel, so
unlike a lifecycle callback it **cannot stop the game** — which is what the
canonical header says it is for.

CNA-Go does not invent one and does not discard anything either. A handler
failure is recorded as the run's callback failure through the same
`recordCallbackFailure` path every lifecycle failure uses, so it surfaces from
`Game.Run()`; the frame loop continues to its normal exit. A panic is recovered
in the trampoline and recorded the same way. Nothing crosses the C frame.

This is a recorded native-boundary projection, not an equivalence claim: a CLR
exception from a `Game.Activated` handler propagates into the host's message
pump, and Go cannot reproduce that through a void C callback.

### `inCallback` is deliberately not raised

An event delivery does not mark the runtime as inside a callback, so a handler
cannot operate the native game through the borrowed-handle path. That follows
the header — "these handlers only observe" — and it matters most for `Disposed`,
which is delivered from inside `cna_game_destroy` while the native game is
already being torn down.

## Runtime qualification, per event

| event | structural bridge | native delivery | status |
| -- | -- | -- | -- |
| `Activated` | complete | 60/60 isolated cycles, once each, first in order | **VERIFIED_NATIVE** |
| `Exiting` | complete | 60/60 isolated cycles, once each, before `Disposed` | **VERIFIED_NATIVE** |
| `Disposed` | complete | 60/60 isolated cycles, once each, during native destroy | **VERIFIED_NATIVE** |
| `Deactivated` | complete | **0** — HEADLESS cannot lose focus | **NOT_RUN_ENVIRONMENT** |

The `Deactivated` row is the honest one. The qualification artifact runs a
HEADLESS renderer with no window manager, so there is no focus transition away
from the game to produce and `CNA_GAME_EVENT_DEACTIVATED` is never delivered.
The Go accessors, the edge-trigger guard and the raise path are all proved
without it — by `raiseNativeGameEvent` under test and by the corpus — and the
delivery counter is left at zero rather than being made to move. The capability
inventory records it as `game-activation-transitions` / `BACKEND_BLOCKED`.

Nothing was fabricated to make a counter advance.

## Where `Disposed` diverges, stated plainly

In CLR, `Game.Disposed` fires when the consumer calls `Game.Dispose()`. A
consumer who never disposes never sees it, and the timing is theirs.

In CNA-Go it fires when the **native** game is disposed, which happens inside
`Run` after the frame loop ends. `Game::Dispose()` and `Game::Dispose(Boolean)`
are still missing members, so a consumer cannot trigger it; and because CNA
already raises the canonical signal, CNA-Go does not add a second synthetic
raise in Go.

Two consequences are recorded rather than smoothed over:

- the **timing** is fixed by `Run` rather than chosen by the consumer;
- the reference's disposal **body** — disposing each `IDisposable` component,
  disposing the graphics device manager, `UnhookDeviceEvents` — is not run by
  CNA-Go, because the method that runs it is not projected yet.

Projecting `Game.Dispose` is a separate decision and belongs with the rest of
`Game`'s disposal surface, not with the event slice. It is recorded in the
handoff as such. What is *not* deferred is the event itself: a handler is
called, exactly once, with the Game as sender and the shared `EventArgs.Empty`,
which is what the reference's raise site does.

## ABI delta — additive binding over an unchanged binary

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
native_library_sha256   e912cd1d...b116f  UNCHANGED
```

Exactly what was added, and nothing else:

- **2 newly bound functions**: `cna_game_subscribe` (5 parameters) and
  `cna_game_unsubscribe` (1). `PROTOTYPE_TYPE_POSITIONS` grows by
  `(1+5) + (1+1) = 8`, which is the return plus each parameter of each.
- **1 newly measured callback**: `CNA_GameEventCallback`, pinned by assigning a
  function of the exact shape to the canonical typedef under
  `-Werror=incompatible-pointer-types`.
- **3 newly measured layouts**: `sizeof(CNA_GameEvent)`,
  `sizeof(CNA_GameEventRegistrationHandle)` and `sizeof(CNA_GameEventCallback)`.
- **5 newly measured constants**: the four `CNA_GAME_EVENT_*` identities and
  `CNA_GAME_EVENT_MAXIMUM`, each `_Static_assert`ed against CNA-Go's own private
  copy in `abi_manifest.h`.

`C_GO_MEASUREMENTS` is `len(measurements) + PROTOTYPE_TYPE_POSITIONS`, so
`96 + 8 + 3 = 107` follows from the two lines above it.

**Every pre-existing entry is byte-identical.** The 23 previously bound
functions keep their names and parameter counts, the 28 previously measured
layouts keep their values, and the five original `_Static_assert`s are
unchanged. No claim is made that any new CNA ABI exists, that CNA was rebuilt,
or that any CNA build output was reproduced: `REPRODUCED_BUILD_OUTPUT` stays
`NOT_ESTABLISHED`.

## The 19 ABI mutation controls

`tools/native_abi/main_test.go` removes one ABI pin at a time from real source
and requires the compile to fail. Two translation units carry the pins:
`internal/interop/bridge.c` compiled with CNA-Go's private manifest and no CNA
header — exactly how the cgo build sees it — and
`tools/native_abi/testdata/probe.c` compiled *with* the canonical header, which
is the only place the private manifest and the shipped declarations meet.

Bridge translation unit (12): wrong event constant; swapped event constants;
narrowed registration handle; event-identity count drift; missing
`cna_game_unsubscribe`; missing `cna_game_subscribe`; callback returns a result;
callback drops its user data; callback takes a game handle; unsubscribe takes
the game; `bridge.h` mirror drift; mirror count drift.

Probe translation unit (7): user data before the callback; subscribe drops its
out-registration; subscribe returns the registration; unsubscribe takes a
narrowed handle; the private manifest disagreeing with the canonical header;
probe callback returns a result; probe callback takes a game handle.

Two unmutated controls compile cleanly under the same flags, and one test keeps
the measured prototype table and the bridge's required-symbol list from drifting
apart in either direction.

One of those placements is worth stating rather than working around: **the
user-data position cannot be pinned in the bridge translation unit at all.** GNU
C converts silently between `void*` and a function pointer, so swapping the
callback and the context in the prototype still compiles against a manifest that
declares the same swap — the manifest would simply be self-consistently wrong.
That pin has to compare the manifest with the canonical declaration, which only
the probe does.

## Scoreboard

```text                                     before   after
TARGET_MEMBERS                             1776    1787
TOTAL_DIAGNOSTICS                           310     299
MISSING_MEMBER                              175     164
Game missing members                         37      26

COMPLETE_TYPES                              117     117
PARTIAL_TYPES                                 5       5
MISSING_TYPE                                135     135
REFERENCE_MEMBERS / EXPECTED_GO_MEMBERS    2964 / 3255   unchanged

behavior corpus                             620     625
external canary tests                        34      39
native ABI mutation controls                  0      19
```

Eleven members, all in the coherent lifecycle slice: eight event accessors and
the three protected raise sites. Every mismatch, leak, allowlist and unmeasured
counter is zero throughout. `XNA_BASE_RELATIONSHIPS`, `XNA_BASE_DERIVED_TYPES`,
`XNA_DEFERRED_BASE_BLOCKERS` and `XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED` are
unchanged at 12 / 41 / 25 / 245: no derived type became complete and the
XNA-to-XNA inheritance frontier was not touched.

`GAME_BASE_CALL_ADAPTERS` stays at 5. No `GameBase...` helper was added, because
none of these members is a `GameCallbacks` override.
