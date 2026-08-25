# Foundation 31 — the explicit Game base-call architecture

Foundation 30 gave `Game` its managed component engine. Nothing could reach it:
`GameCallbacks` projects CLR protected virtual **overrides**, and an override
that cannot call its base is not an override.

This milestone adds the five package-level functions that run the base bodies,
and — more importantly — establishes that **base behavior is never automatic**.

## The decision, and why the obvious alternative is wrong

In CLR the derived class decides:

```csharp
protected override void Update(GameTime t)
{
    this.spawner.Tick(t);
    base.Update(t);          // the derived class chose this, and chose HERE
}
```

Three facts follow, and all three are contract:

- Omitting `base.Update(t)` is legal, and means the component loop does not run
  that frame.
- Putting it first instead of last changes what runs before what.
- Calling it twice runs the loop twice.

The call site **is** the semantics.

If CNA-Go ran the base body automatically around each `GameCallbacks` call, every
override would become a mandatory base call at a position CNA-Go picked. That is
not XNA's contract; it is a different contract that resembles it. So:

```go
func GameBaseInitialize(game *Game) error
func GameBaseLoadContent(game *Game) error
func GameBaseUpdate(game *Game, gameTime GameTime) error
func GameBaseDraw(game *Game, gameTime GameTime) error
func GameBaseUnloadContent(game *Game) error
```

They are **package-level functions**, not methods on `Game`, so `Game`'s
projected XNA member surface gains no name Microsoft never declared. The first
parameter is the `this` a CLR base call passes implicitly.

`GameCallbacks` is unchanged. Its five members are exactly what they were.

## They are language support, and that is measured

The family is declared in `tools/api_compat/mapping-rules.json` under
`gameBaseCallAdapters` and in the executable registry `gameBaseCallAdapters`,
and `measureGameBaseCallAdapters` enforces four claims:

1. **Every adapter corresponds to a real protected XNA virtual.** The CLR member
   is looked up in the expected surface built from the pinned contract, and its
   declared accessibility must be `protected`. `SourceAccess` was added to the
   expected surface for exactly this, so the claim is measured rather than
   asserted in prose.
2. **No arbitrary extra base helper exists.** Every exported package-level
   framework function named `GameBase*` must be a declared adapter.
3. **No supported virtual is missing its helper.** Every `GameCallbacks` member
   must have exactly one adapter.
4. **Nothing here is an XNA identity.** The five names appear nowhere in the
   expected surface, and `REFERENCE_MEMBERS`, `EXPECTED_GO_MEMBERS` and
   `TARGET_MEMBERS` are unmoved.

There is **no allowlist**. The names reach `adapterFunctions` from the registry
in `init`, so a helper can only be admitted by being declared — and being
declared is what subjects it to measurement.

```text
GAME_BASE_CALL_ADAPTERS=5
GAME_BASE_CALL_DEFERRED_STEPS=4
```

### Negative controls

Thirteen defects, in one shared table behind both the named test and the
mutation inventory:

```text
adapter_absent_from_the_package
adapter_projected_as_a_method_on_game
adapter_signature_drops_the_game_parameter
adapter_signature_gains_a_result
arbitrary_extra_base_helper
adapter_missing_for_a_supported_virtual
adapter_declared_for_a_member_that_is_not_a_supported_virtual
error_result_with_no_recorded_reason
recorded_reason_with_no_error_result
deferred_step_records_no_reason
deferred_step_unclassified
deferred_step_marked_observable
adapter_records_no_reference_body
```

Two drift guards keep the three declarations aligned:
`mapping-rules.json` ↔ registry, and defect table ↔ mutation inventory.

The behavioral controls — automatic invocation, duplicated invocation, recursion
into the callbacks, and bypassing component semantics — are proved by running
code rather than by structure, in-repo and again from outside the module.

## The base bodies, from IL

### `Update` — 141 bytes

```csharp
Logger.BeginLogEvent(LoggingEvent.Update, "");
for (int i = 0; i < updateableComponents.Count; i++)
    currentlyUpdatingComponents.Add(updateableComponents[i]);
for (int j = 0; j < currentlyUpdatingComponents.Count; j++) {
    IUpdateable u = currentlyUpdatingComponents[j];
    if (u.Enabled) u.Update(gameTime);
}
currentlyUpdatingComponents.Clear();
FrameworkDispatcher.Update();
doneFirstUpdate = true;
Logger.EndLogEvent(LoggingEvent.Update, "");
```

**The snapshot does not freeze everything.** The first loop copies the ordered
list and the second iterates the copy, so adding or removing components during
the frame does not change which components that frame updates. But `Enabled` is
read at **iteration** time, not at snapshot time, so a component disabled
earlier in the same frame is skipped when its turn comes. Both halves are
qualified separately.

**There is no `try`/`finally`.** A component that throws leaves
`currentlyUpdatingComponents` populated, and the next frame's first loop
*appends* to it. The Go projection reproduces the structure exactly — a
straight-line `Clear`, never a deferred one — so a component that panics leaves
exactly the same debris. It cannot be reached through the error channel:
`IUpdateable.Update` is infallible, because the one shipped implementor's
`Update` is a bare `ret` of code size 1.

### `Draw` — 107 bytes

The same shape over `IDrawable` and `Visible`. Three things it does **not**
contain, and all three matter:

- **No device access at all.** Base `Draw` does not clear, present, or reach
  `GraphicsDevice`. That work lives in `Game::BeginDraw` and `Game::EndDraw`,
  which delegate to `IGraphicsDeviceManager` and are separate protected virtuals
  CNA-Go does not project. Each `IDrawable` draws itself.
- No logging — that belongs to `BeginDraw`/`EndDraw`.
- No field assignment, which is why base `Draw` is the one member of the family
  with **no deferred step whatsoever**.

### `Initialize` — 78 bytes

```csharp
HookDeviceEvents();
while (notYetInitialized.Count > 0) {
    notYetInitialized[0].Initialize();
    notYetInitialized.RemoveAt(0);
}
if (graphicsDeviceService != null && graphicsDeviceService.GraphicsDevice != null)
    LoadContent();
```

`Initialize()` runs **before** `RemoveAt(0)`, so a component whose `Initialize`
fails stays at the head of the queue and the drain stops where it failed; a
retry resumes exactly there. `Count` is re-read every iteration, so a component
that adds another component from inside its own `Initialize` extends the same
drain — consistent, because `inRun` is still false and the add handler queues.

**Initialization order is ADD order, not `UpdateOrder`.** `notYetInitialized` is
a plain list; only the update and draw lists are ordered. This is asserted
in-repo, in the corpus, and from outside.

### `LoadContent` and `UnloadContent` — 1 byte each

```text
IL_0000:  ret
```

True authoritative no-ops. The projection faithfully does nothing at all — not
even to the pending queue, which is asserted.

## The four deferred reference steps

Every step the projection does not reproduce is recorded with a class and a
reason, and a deferral marked **observable** is rejected by the verifier as a
stop condition rather than accepted as a deferral.

| member | step | class | why |
|---|---|---|---|
| `Initialize` | `HookDeviceEvents()` | ARCHITECTURE | see below |
| `Initialize` | the conditional `LoadContent()` | ARCHITECTURE | its condition is the field `HookDeviceEvents` assigns |
| `Update` | `FrameworkDispatcher.Update()` | SUBSYSTEM | pumps media and audio; CNA-Go has neither backend |
| `Update` | `Logger.Begin/EndLogEvent` | UNOBSERVABLE | XNA's private profiling channel |

### `HookDeviceEvents` — two independent blocks

```csharp
graphicsDeviceService = Services.GetService(typeof(IGraphicsDeviceService))
                        as IGraphicsDeviceService;
if (graphicsDeviceService != null) {
    graphicsDeviceService.DeviceCreated   += DeviceCreated;
    graphicsDeviceService.DeviceResetting += DeviceResetting;
    graphicsDeviceService.DeviceReset     += DeviceReset;
    graphicsDeviceService.DeviceDisposing += DeviceDisposing;
}
```

1. **Package layering.** `IGraphicsDeviceService` lives in the **Graphics**
   package, which imports the framework package. The settled cross-package cycle
   rule already projects `Game`'s device-typed members into the descendant
   package for exactly this reason — `Game::GraphicsDevice` maps to
   `graphics.GameGraphicsDevice`, not to a framework method. The framework
   package cannot name the type.
2. **Nothing can publish the service.** The reference's own registrar is
   `GraphicsDeviceManager`, and CNA-Go's `GraphicsDeviceManager` is a partial
   native-backed facade satisfying **neither** `IGraphicsDeviceService` nor
   `IGraphicsDeviceManager`, so `GetService` would find nothing to store even if
   the type were nameable. See the audit below.

**Neither deferral is observable from the managed component part.** Both steps
sit outside the drain and touch neither the queue, the collection, nor either
derived list, and CNA-Go exposes no way to count an event's subscribers — so no
component's `Initialize` can tell whether they ran. That is why they are
deferrals rather than a stop condition.

The conditional `LoadContent()` is deferred with `HookDeviceEvents` because its
condition is exactly the field that step assigns: **with no device service, the
reference does not call `LoadContent` from `Initialize` either.** CNA-Go's
`LoadContent` arrives from the native `load_content` callback instead.

## The native callback-order audit

The CNA ABI documents its order as a contract, in `CNA/C/runtime.h`:

> this runs, then the runtime initializes its components and creates the device,
> and only then does `CNA_GameCallbacks::load_content` run. A first frame
> therefore delivers `initialize`, `load_content`, `begin_run`, `update`,
> `draw`.

XNA's, from `Game::RunGame`:

```csharp
graphicsDeviceManager = Services.GetService(typeof(IGraphicsDeviceManager)) as ...;
graphicsDeviceManager?.CreateDevice();
this.Initialize();          // drains components; calls LoadContent if a device exists
this.inRun = true;
this.BeginRun();
this.Update(this.gameTime); // the first Update is called directly
host.Run();                 // the loop: Update / Draw
this.EndRun();
finally { if (!endRunRequired) inRun = false; }
```

Side by side:

| step | XNA | CNA | CNA-Go |
|---|---|---|---|
| device creation | `CreateDevice()` before `Initialize` | between `initialize` and `load_content` | native |
| managed component init | inside `Game.Initialize` | not CNA's concern — Go owns it | inside the `initialize` hook, when the override calls `GameBaseInitialize` |
| content load | inside `Initialize`, gated on a device | `load_content`, after the device exists | `load_content` callback |
| `inRun` raised | after `Initialize()` returns | — | after the Go `Initialize` callback returns |
| first update | called directly, then the loop | `update` | `update` callback |
| `inRun` lowered | `RunGame`'s `finally` | — | after the blocking `Run` returns |

**The relative order of everything CNA-Go owns is preserved.** Managed component
initialization happens in the Initialize step and content loading after it, in
both. The device is created before content loads, in both.

Where CNA-Go substitutes the native host for `GameHost`, it is recorded rather
than hidden: `inRun` is raised on the native `initialize` hook boundary and
lowered when the blocking `Run` returns, which are the two points `RunGame`
assigns it.

`begin_run`, `end_run`, `begin_draw` and `end_draw` exist in
`CNA_GameFrameHooks` and are **not bound** by CNA-Go's bridge. That is not a
gap this milestone creates: `Game::BeginRun`, `EndRun`, `BeginDraw` and `EndDraw`
are four of `Game`'s 37 remaining missing members, they are not `GameCallbacks`
members, and they are outside this slice. **No CNA change was needed and none
was made.**

## Fallibility

All five return `error`, and every reason is recorded — an error result with no
recorded reason, and a recorded reason with no error result, are both rejected.

- **`GUARD`**, on all five: Go — unlike CLR, where a constructor always ran —
  lets a consumer write `&framework.Game{}`. CNA-Go's settled answer at a public
  entry point taking such a `Game` is an error result; `Game.Run` and `Game.Exit`
  already report exactly this condition. It has no CLR counterpart and is
  labelled as the Go-only guard it is.
- **`REFERENCE`**, on `GameBaseInitialize` only: the drain calls
  `IGameComponent.Initialize`, which the settled contract projects as fallible.

A test asserts that for a *constructed* Game the other four never fail, even with
a component whose `Initialize` would fail.

## External consumer canary

The canary grew from **16** tests to **29**, and now includes a realistic
downstream `UserGame` that satisfies `GameCallbacks` with its own state and
decides per member whether and where to call its base. Nothing in it embeds
`Game`, subclasses anything, or names an unexported identifier.

All twelve required claims are proved from a module whose only dependency is an
extracted CNA-Go source artifact, with `GOWORK=off` and no sibling checkout:

```text
 1 construct Game                          TestConsumerConstructsAGameAndReachesItsManagedState
 2 stable Components / Services identity   same
 3 add GameComponents                      TestConsumerAddsComponentsAndSubscribesToTheCollection
 4 subscribe to component events           same, and TestConsumerObservesEnabledVisibleAndOrderChanges
 5 callback calls GameBaseInitialize       TestConsumerCallbackCallsTheBaseAndSeesTheComponentLoop
 6 callback calls GameBaseUpdate           same
 7 callback calls GameBaseDraw             same
 8 component Update ordering observable    same  (update order is the REVERSE of draw order)
 9 omitting the base call prevents it      TestOmittingTheBaseCallPreventsBaseComponentIteration
10 base-call position changes ordering     TestTheBaseCallPositionChangesOrderingRelativeToUserCode
11 removal stops iteration                 TestRemovingAComponentStopsItsBaseIteration
12 no public native handle                 TestBaseCallsExposeNoNativeHandle
```

Claim 11's stronger form — disposing a component removes it from
`Game.Components` — needs `GameComponent`, which is the next milestone.

`TestBaseCallsNeverReEnterTheConsumerCallbacks` is the recursion control, and its
proof is that it terminates: every override in it calls its base, so a base that
called back would not return.

## Scoreboard

```text                        before   after
TARGET_TYPES                    121      121
TARGET_MEMBERS                 1757     1757
TOTAL_DIAGNOSTICS               311      311
MISSING_MEMBER                  175      175
REFERENCE_MEMBERS              2964     2964
EXPECTED_GO_MEMBERS            3255     3255

GAME_BASE_CALL_ADAPTERS           0        5
GAME_BASE_CALL_DEFERRED_STEPS     0        4

behavior corpus                 603      612
mutation inventory              480      493
external canary tests            16       29
```

**No counter of XNA identity moved, and that is the point.** The family is
language support: it makes an existing projected surface usable without adding
one member to it. Every mismatch, leak, allowlist and unmeasured counter is zero.

## What this milestone did NOT do

- No C ABI function was added and no CNA C++ changed.
- `GameCallbacks` is unchanged.
- No `Game` member was completed; `Game` still has 37 missing members.
- `System.Exception` was not reopened.
- `GraphicsResource` ownership was not reopened.
