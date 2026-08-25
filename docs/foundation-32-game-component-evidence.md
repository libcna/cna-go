# Foundation 32 — GameComponent

`GameComponent` was blocked for six milestones by exactly one statement.
`Dispose(bool)` runs

```csharp
get_Game().get_Components().Remove(this)
```

and `Game` had no `Components`. Foundation 30 gave it one; nothing else about
the class changed, and the class is now complete.

It is the first type in the profile that CNA-Go both **declares** and **drives**:
the component engine of Foundation 30 and the base calls of Foundation 31 were
derived from the same IL, separately, and this milestone joins them with no
adapter in between.

## Reference authority

`Microsoft.Xna.Framework.Game.dll`, sha256
`b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0`, read with
`ikdasm`. Public surface remains the pinned contract at
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## The class is pure managed

Every member is one field access, one comparison, or one delegate invoke:

```text
.ctor(Game)              21 bytes    two stfld and the base call
get_Enabled              7 bytes     one ldfld
get_UpdateOrder          7 bytes     one ldfld
get_Game                 7 bytes     one ldfld
set_Enabled              29 bytes    compare, store, announce
set_UpdateOrder          29 bytes    compare, store, announce
Initialize               1 byte      ret
Update(GameTime)         1 byte      ret
Finalize                 17 bytes    Dispose(false), which returns immediately
OnEnabledChanged         22 bytes    null test and Invoke
OnUpdateOrderChanged     22 bytes    null test and Invoke
Dispose()                14 bytes    Dispose(true); GC.SuppressFinalize
Dispose(bool)            79 bytes    lock, remove, announce
```

Nothing reaches a device, a host, a window, or a CNA handle, so it joins
`pureManagedTypes` and starts from "no member is fallible".

## The four quirks that are load-bearing

### 1. The constructor validates nothing

`new GameComponent(null)` is legal, and the projection accepts nil. The Game
reference is read back in exactly one place — `Dispose(bool)`'s
`if (Game != null)` — so a component with no Game simply skips its removal and
still announces `Disposed`.

### 2. `OnEnabledChanged` and `OnUpdateOrderChanged` ignore their sender

```text
OnEnabledChanged(object sender, EventArgs args)
  ldarg.0; ldfld EnabledChanged; brfalse.s RET
  ldarg.0; ldfld EnabledChanged
  ldarg.0                       // <- `this`, NOT the sender parameter
  ldarg.2                       // the args, forwarded
  callvirt EventHandler`1::Invoke
```

The parameter is accepted, ignored, and replaced by `this`. **This is what makes
Game's engine work**: `UpdateableUpdateOrderChanged` reads the *sender* to decide
which component to re-place, so forwarding the argument through would break it.
Foundation 30 derived that dependency from the other side; this milestone
supplies the half that satisfies it.

### 3. The setters suppress an unchanged value

Compare, store, then announce — and nothing at all when the value is unchanged.
That suppression is why an assignment that changes nothing does not re-place the
component in Game's ordered list.

Store precedes announce, so a handler observes the new value.

### 4. `Dispose` removes before it announces, and is not idempotent

```csharp
if (!disposing) return;
lock (this) {
    if (Game != null) Game.Components.Remove(this);
    if (Disposed != null) Disposed(this, EventArgs.Empty);
}
```

A `Disposed` handler therefore observes a Game whose `Components` no longer
contains the component **and** whose update list has already untracked it,
because `Remove` reaches `RemoveItem`, which announces `ComponentRemoved`, which
Game's own engine handler consumes first.

The reference `pop`s `Remove`'s boolean and raises `Disposed` unconditionally, so
a second `Dispose` raises again. The projection does the same. The **error** is a
different channel and is not discarded.

## Fallibility, member by member

`GameComponent` is pure managed, so each of the six fallible members names its
own evidence and the other six are infallible:

| member | fallible | why |
|---|---|---|
| `Initialize` | yes | **the contract it implements**, not its own body |
| `Dispose()` / `Dispose(bool)` | yes | `Components.Remove`, and the `Disposed` raise |
| `OnEnabledChanged` / `OnUpdateOrderChanged` | yes | they invoke consumer handlers |
| `SetEnabled` / `SetUpdateOrder` | yes | they reach those raise sites |
| `.ctor` | no | validates nothing |
| `Update`, `Finalize` | no | observably no-ops |
| `Enabled`, `UpdateOrder`, `Game` getters | no | one `ldfld` each |

`Initialize` is the first member in the profile whose fallibility comes from a
**contract it implements** rather than from its own body. `IGameComponent`
declares an error result on the evidence of its *other* implementor —
`DrawableGameComponent.Initialize` throws `InvalidOperationException` when
`IGraphicsDeviceService` is absent — and Go requires an exact signature to
satisfy an interface. The member carries the channel and never uses it. That
reason is recorded in the classification table rather than left implicit.

## Compiler-checked interface conformance — a new general rule

Interface conformance must be witnessed by the compiler, with no hand-written
verifier exception. Until now the only conformance the verifier checked that way
was the PackedVector family's. Every other declaration was checked
structurally, member for member — necessary but not sufficient, since a
receiver-kind or signature mistake can pass a member comparison and still leave
the class unusable where the contract says it belongs.

`measureDeclaredInterfaceConformance` adds the general rule:

> a projected class that CLR metadata says implements a projected XNA interface
> must satisfy that interface's Go projection, on the **pointer** method set, and
> `go/types` must say so

It runs only for **complete** types, because a partial type is missing members by
definition and would report the same gap twice. `GraphicsDeviceManager` is the
live example: it declares `IGraphicsDeviceService` and `IGraphicsDeviceManager`
and satisfies neither, because 20 of its members are still missing — already
fully reported as `MISSING_MEMBER`.

```text
DECLARED_INTERFACE_CONFORMANCE=2

GameComponent -> IGameComponent   pointerSatisfies=true  PASS
GameComponent -> IUpdateable      pointerSatisfies=true  PASS
```

Four negative controls, built from synthetic `go/types` evidence so they attack
the rule rather than the binding: a conforming class, a value-receiver-only
class (which still conforms through the pointer method set), an absent method,
and — the one a structural comparison cannot catch — **a method with the right
name and the wrong signature**. A fifth control pins the completeness gate.

## The concurrency projection, and its one divergence

`Dispose(bool)`'s body is inside `lock (this)`. CLR's `Monitor` is **reentrant
per thread**, and reentry here is reachable: a `Disposed` handler, or a
`ComponentRemoved` handler, may call `Dispose` again.

Go has no reentrant mutex and no supported thread identity, so a plain `Lock`
would **deadlock** on a path the reference merely recurses on — strictly worse
than the reference. The projection takes the lock with `TryLock`: an uncontended
call holds it for the whole critical section exactly as `Monitor` does, and a
reentrant call proceeds without re-acquiring, which is what `Monitor`'s recursion
does.

The divergence is that a genuinely **concurrent** second disposer would proceed
instead of blocking. That is deliberate, and recorded rather than hidden:
CNA-Go's Game and component state is owner-thread state, the binding promises no
cross-goroutine safety for it, and a guaranteed deadlock on a reachable
single-threaded path is the worse of the two errors. A test proves reentry
terminates and raises twice, and the whole suite is race-clean.

## The frontier this unblocks — and does not

Both types `GameComponent` was said to block were re-derived from IL. **Neither
is now implementable**, and the reason is the same for both: they are the
profile's only two XNA classes that inherit from another XNA class CNA-Go
projects, and CNA-Go has no architecture for XNA-to-XNA inheritance.

### `DrawableGameComponent` — five distinct blockers

1. **XNA base composition.** It `extends GameComponent`, and the contract does
   **not** redeclare `GameComponent`'s nine public members on it. A faithful
   projection would have to re-expose them, exactly as the BCL base composition
   does — but that registry is for BCL bases, and Go embedding of an XNA base is
   already refused by `BASE_MAPPING_MISMATCH`.
2. **A derived-class override adapter.** `LoadContent` and `UnloadContent` are
   `family newslot virtual`, the same problem `GameCallbacks` solves for `Game` —
   and it would need its own solution, since `GameCallbacks` is Game's.
3. **`get_GraphicsDevice`** returns `Graphics.GraphicsDevice`: a partial type,
   reached across the package boundary the cross-package cycle rule governs, and
   it throws `InvalidOperationException` before `Initialize` has run.
4. **`Initialize`** resolves `IGraphicsDeviceService` out of `Services` and
   throws when it is absent. Nothing in CNA-Go can publish that service, so a
   faithful `Initialize` would always fail.
5. **`Dispose(bool)`** calls the virtual `UnloadContent` and unsubscribes four
   device events.

### `GamerServicesComponent` — three distinct blockers

1. The same XNA base composition.
2. `Game.Window.Handle` — `GameWindow` is a missing type, `Game::Window` is a
   missing member, and the handle is a native window handle.
3. `GamerServicesDispatcher` lives in `Microsoft.Xna.Framework.GamerServices.dll`,
   which is **not one of the seven pinned assemblies**. Even admitting it would
   not help: there is no GamerServices runtime in CNA at all.

## Scoreboard

```text                        before   after
TARGET_TYPES                    121      122
TARGET_MEMBERS                 1757     1776
TOTAL_DIAGNOSTICS               311      310
MISSING_TYPE                    136      135
COMPLETE_TYPES                  116      117
PARTIAL_TYPES                     5        5
MISSING_MEMBER                  175      175
REFERENCE_MEMBERS              2964     2964
EXPECTED_GO_MEMBERS            3255     3255

DECLARED_INTERFACE_CONFORMANCE    0        2

behavior corpus                 612      620
external canary tests            29       34
```

19 mapped Go identities for one type. `MISSING_MEMBER` did not move: no partial
type was touched. Every mismatch, leak, allowlist and unmeasured counter is zero.

## What this milestone did NOT do

- No C ABI function was added and no CNA C++ changed.
- `GameCallbacks` is unchanged.
- No `Game` member was completed; `Game` still has 37 missing members, and
  `Window`, `Content`, the timing controls, activation and disposal are all
  untouched.
- `System.Exception` was not reopened: component failure still travels on the
  opaque Go error channel.
- `GraphicsResource` ownership was not reopened.
