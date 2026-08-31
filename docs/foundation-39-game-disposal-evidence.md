# Foundation 39 — Game's disposal surface, and the corrected `Disposed` raise site

Foundation 34 bound `CNA_GAME_EVENT_DISPOSED` to the public `Game.Disposed`
event and recorded the divergence rather than hiding it: `Game::Dispose` was not
projected, and the native signal was the only disposal fact CNA-Go had.

This milestone projects `Dispose`, and the divergence is **corrected** instead of
preserved.

## Why the old binding was wrong

The pinned IL has exactly one raise site for `Game::Disposed`, at the tail of
`Dispose(bool)`:

```
IL_006d: ldarg.0
IL_006e: ldfld      EventHandler`1<EventArgs> Game::Disposed
IL_0073: brfalse.s  IL_0086
IL_0075: ldarg.0
IL_0076: ldfld      EventHandler`1<EventArgs> Game::Disposed
IL_007b: ldarg.0                                   // `this` is the sender
IL_007c: ldsfld     EventArgs EventArgs::Empty
IL_0081: callvirt   EventHandler`1::Invoke(object, !0)
```

There is no `OnDisposed`, and nothing else in the class touches the field. The
event is raised by managed disposal and by nothing else.

CNA raises its own signal from native game destruction, inside
`cna_game_destroy`. Those are different observable moments, and the difference is
not academic:

| | old binding | reference |
| -- | -- | -- |
| consumer runs, never disposes | **fires** | does not fire |
| consumer disposes, never runs | **does not fire** | fires |
| consumer disposes twice | n/a | fires **twice** |

XNA raise-site fidelity wins. The native signal stays bound, delivered and
counted — it is what proves a registration outlives `cna_game_destroy` — and it
raises nothing public.

## The rule that changed

The Foundation 36 registry rule was:

> every CLR event `Game` declares is bound to exactly one canonical CNA signal,
> through exactly the raise path the reference uses

That is satisfied by binding a signal to an event whose reference raise site is
not the host's at all, which is exactly what shipped. The stronger rule, and the
one measured now:

> every projected XNA event must have its **authoritative** XNA raise path, and
> a native signal may **implement** that path only when the semantics align

It is measured as two closed vocabularies that are one decision seen from two
ends, and they must agree:

| `RaisePath` | pairs with `NativeSignalRole` | events |
| -- | -- | -- |
| `NATIVE_HOST_SIGNAL` | `PUBLIC_EVENT_RAISE` | `Activated`, `Deactivated`, `Exiting` |
| `MANAGED` | `LIFECYCLE_ONLY` | `Disposed` |

A `MANAGED` raise path must name the projected **protected** member that raises
it, and that member must exist in the framework package; a `NATIVE_HOST_SIGNAL`
one must not name one at all. Every signal also records its `NativeSignalMoment`
— what the CNA signal actually means — which is how a `LIFECYCLE_ONLY` role
states *why* the semantics do not align rather than merely asserting it.

```text
GAME_NATIVE_SIGNALS                    4
GAME_NATIVE_SIGNAL_RAISE_SITES         3
GAME_NATIVE_SIGNALS_LIFECYCLE_ONLY     1
GAME_MANAGED_EVENT_RAISE_SITES         1
```

Ten new negative controls drive it, including the two that matter most: a native
signal driving a managed raise path (the divergence itself), and a host-raised
event demoted to `LIFECYCLE_ONLY`, which would leave its accessors unfireable.

## The projected members

```go
func (g *Game) DisposeByNone() error
func (g *Game) DisposeByBoolean(disposing bool) error
func (g *Game) Finalize() error
```

`Dispose()` is `newslot virtual final` — declared on the interface slot and
sealed — so it projects as a plain method. `GC.SuppressFinalize` has nothing to
suppress: CNA-Go registers no Go finalizer on a `Game`.

`Finalize` reproduces `try { Dispose(false); } finally { base.Finalize(); }` in
one line. It carries an `error` because `Game` is a native-backed facade and
every member of one does; `GameComponent::Finalize` carries none because
`GameComponent` is classified pure managed. The classification is per type and
is read from the reference, not chosen per member.

### `Dispose(bool)`, exactly

```csharp
if (!disposing) return;
lock (this)
{
    IGameComponent[] array = new IGameComponent[this.gameComponents.Count];
    this.gameComponents.CopyTo(array, 0);
    for (int i = 0; i < array.Length; i++)
        if (array[i] is IDisposable d) d.Dispose();
    if (this.graphicsDeviceManager is IDisposable m) m.Dispose();
    this.UnhookDeviceEvents();
    if (this.Disposed != null) this.Disposed(this, EventArgs.Empty);
}
```

**The snapshot is required, not defensive.** `GameComponent::Dispose(bool)` runs
`Game.Components.Remove(this)`, so disposing a component *mutates* the collection
being walked. The reference copies first and walks the copy; a component that
removes a different component from inside its own disposal is therefore still
disposed.

**There is no try/catch.** A component whose disposal fails propagates straight
out, leaving every later component undisposed and `Disposed` unraised.

**It is not idempotent.** There is no disposed flag anywhere in `Game`. Every
call re-runs the whole body and raises `Disposed` again. `GameComponent` already
carries exactly this behaviour for exactly this reason.

**The lock** is the settled `TryLock` projection `GameComponent::Dispose(bool)`
already uses, with the same recorded divergence: CLR monitors are reentrant and
reentry is reachable here (a `Disposed` handler may dispose again), Go has no
reentrant mutex, so a genuinely concurrent second disposer proceeds instead of
blocking rather than deadlocking on a path the reference merely recurses on.

### `component as IDisposable`, projected

`System.IDisposable` contributes **no** projected Go interface, so there is no
`framework.IDisposable` to assert against. What the `isinst` tests is whether the
component declares the interface's single member, and the settled overload rule
decides what that member is called in Go:

| the type declares | the Go member is |
| -- | -- |
| only `Dispose()` | `Dispose() error` |
| `Dispose()` and `Dispose(bool)` | `DisposeByNone() error` |

Microsoft's own two components take the second spelling. A consumer's own
component may legitimately take either, so both are accepted and nothing else
is — a method of another name or another shape is not the `IDisposable` member,
and the reference's `isinst` would not have matched it either. The two-overload
spelling is tried first because it is the one the reference's own components
have.

## The two steps that are not reproduced

Both are transitive on deferrals that already exist, and neither is faked.

`graphicsDeviceManager as IDisposable` reads a field with exactly one assignment
in the whole class, at the head of `RunGame`, which CNA-Go does not perform — the
architecture deferral both draw hooks already carry. The field is permanently
null, which is the state the reference itself has whenever no manager is
registered, so the guarded branch is simply not taken.

`UnhookDeviceEvents` **is** reproduced as written, and its guard is likewise
always false. Its whole body is `if (graphicsDeviceService != null) { remove four
device handlers }`, and `graphicsDeviceService` has exactly one assignment in the
class, inside `HookDeviceEvents`, which base `Initialize` calls and which CNA-Go
records as deferred. Nothing here assumes the absence: the unhook step removes
handlers a hook step added, so with the hook step unreached there is provably
nothing to remove, and the absence is unobservable **at this member**. What a
consumer could observe — their own registered `IGraphicsDeviceService` never
being subscribed to — belongs to `Initialize` and is recorded there.

## Native host destruction is not managed disposal

`Runtime.Run` destroys the native host when a run ends, because the native
generation cannot outlive it. That is an implementation detail of *running*, not
a statement that the XNA object was disposed — the reference's own `RunGame` does
not dispose the `Game` either.

So a consumer may call `Dispose` after `Run` returns and get the full managed
semantics on a `Game` with no native handle left. Nothing in the disposal path
reaches native code, acquires a handle, or pretends one is alive.

## Measured against the pinned artifact

```text                                          before    after
GAME_EVENT_DISPOSED_DELIVERIES                  100        —      (renamed)
GAME_NATIVE_DISPOSAL_SIGNALS                     —        100
GAME_DISPOSED_RAISED_DURING_RUN                  —          0
GAME_DISPOSED_RAISED_BY_MANAGED_DISPOSE          —         80
GAME_DISPOSED_REPEAT_CHECKS                      —         40
GAME_DISPOSE_AFTER_RUN_CYCLES                    —         40
```

The native signal still arrives exactly once per run, 100 times across 80 runs
including 20 second-run cycles — unchanged, which is the point: the bridge
lifetime evidence Foundation 37 established is not weakened by the correction.

`GAME_DISPOSED_RAISED_DURING_RUN = 0` is the correction itself, measured.

`80 = 2 × 40` is the non-idempotency proof. Each of the 40 dispose-after-run
cycles disposes twice and observes two raises; a projection that had invented a
disposed flag would report exactly half. Each cycle also checks that
`Dispose(false)` and `Finalize` raise nothing, and that managed disposal does not
move the native signal count — the two concepts stay separate on the wire, not
just in prose.

## Scoreboard

```text                      before   after
TOTAL_DIAGNOSTICS            295     292
MISSING_MEMBER               160     157
TARGET_MEMBERS              1791    1794
Game missing members          22      19

behavior corpus              633     639
external canary tests         53      59
mutation inventory           555     565
runtime capability rows       41      42
```

Three `Game` members completed: `Dispose()`, `Dispose(bool)` and `Finalize()`.
No type became complete — `Game` still has 19 missing members — and the ABI did
not move, because nothing here reaches native code.
