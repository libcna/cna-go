# Foundation 35 — Game's four frame-boundary virtuals, and the hooks that stay uninstalled

Four more `Microsoft.Xna.Framework.Game` members are complete:

```text
.method family hidebysig newslot virtual instance void BeginRun()      // code size 1
.method family hidebysig newslot virtual instance void EndRun()        // code size 1
.method family hidebysig newslot virtual instance bool BeginDraw()     // code size 36
.method family hidebysig newslot virtual instance void EndDraw()       // code size 31
```

CNA has canonical native hooks at exactly those four positions. This milestone
audits them, measures them against the pinned binary, and **deliberately leaves
them uninstalled**. The reasoning is the substance of this document.

No CNA C++ changed, CNA was not rebuilt, and no ABI counter moved: the pinned
artifact is still `e912cd1d…b116f` and this milestone binds nothing new.

## Why these four are methods on `Game`

The mapper redirects to `GameCallbacks` exactly the five protected virtuals that
contract declares — `Initialize`, `LoadContent`, `Update`, `Draw`,
`UnloadContent`. Every other protected virtual projects as a method on the
declaring type whose body is the reference's base body. That is not a new rule:
`GameComponent::OnEnabledChanged`, `GameComponent::Initialize` and
`GameComponent::Update` are all `family`, all projected as methods, and
`GameComponent` is a complete type on exactly that basis.

So the expected signatures follow from the existing mapping with no new
decision:

```go
func (g *Game) BeginRun() error
func (g *Game) EndRun() error
func (g *Game) BeginDraw() (bool, error)
func (g *Game) EndDraw() error
```

`Game` is a native-backed facade, so every member that is not one of the two
stored-property getters carries an error result. Neither `graphicsDeviceManager`
branch nor either `ret` has a throw site, so the only failure any of the four
actually has is the Go-only guard `Game.Run`, `Game.Exit` and the whole
`GameBase...` family already report: Go, unlike CLR, lets a consumer produce a
`&framework.Game{}` whose managed state was never allocated.

## The two run hooks

```text
BeginRun()   IL_0000: ret
EndRun()     IL_0000: ret
```

Authoritative no-ops. The reference declares each so a derived class has
something to override; neither does anything. This is the same provenance
`GameBaseLoadContent` and `GameBaseUnloadContent` already carry, and the
projection does nothing at all — not to the component engine, not to the pending
queue, not to `Components`, not to the event lists, and not to `inRun`.

`inRun` matters: `RunGame` raises it **before** it calls `BeginRun()`, so the
flag is the caller's business and not the virtual's. A `BeginRun` that raised it
would be reproducing the wrong statement.

## The two draw hooks read one private field

```text
BeginDraw()
  if (graphicsDeviceManager != null && !graphicsDeviceManager.BeginDraw())
      return false;
  Logger.BeginLogEvent((LoggingEvent)4, "");
  return true;

EndDraw()
  if (graphicsDeviceManager != null) graphicsDeviceManager.EndDraw();
  Logger.EndLogEvent((LoggingEvent)4, "");
```

`Game::graphicsDeviceManager` has exactly one assignment in the whole class, at
the head of `RunGame`:

```text
graphicsDeviceManager = Services.GetService(typeof(IGraphicsDeviceManager))
                        as IGraphicsDeviceManager;
if (graphicsDeviceManager != null) graphicsDeviceManager.CreateDevice();
```

CNA-Go does not perform that resolution, and the reason is architectural rather
than an oversight:

- the statement immediately after it calls `CreateDevice()` on whatever it
  found, and in CNA-Go the **native runtime owns the device and creates it
  itself**. Resolving without the `CreateDevice` the reference pairs it with
  would produce a Game that had found a manager it never asked to create
  anything;
- Foundation 30 separately audited and recorded that nothing in CNA-Go can
  register into `IGraphicsDeviceManager` in the first place. The reference's only
  registrar is `GraphicsDeviceManager`'s constructor, and the projected
  `GraphicsDeviceManager` is partial: `CreateDevice`, `BeginDraw`, `EndDraw` and
  `GraphicsDevice()` are all missing, so it satisfies neither service contract
  and `AddService`'s assignability check refuses it.

The field is therefore permanently null. **That is a state the reference itself
has**: an XNA game with no `GraphicsDeviceManager` reaches exactly these
branches, `BeginDraw` falls through to its unconditional `ldc.i4.1; ret`, and
`EndDraw` is observably empty. Both bodies are reproduced as written; the null
branch is simply the one always taken. No value channel was invented and no
behavior was faked.

`Logger.BeginLogEvent`/`Logger.EndLogEvent` are the same `UNOBSERVABLE` deferral
`GameBaseUpdate` already records: the sink is not reachable from any projected
member, so reproducing or omitting the call is indistinguishable.

### The Boolean is a value channel, not a success flag

`DrawFrame` is where it matters:

```text
DrawFrame()                                    // code size 134
try {
  if (ShouldExit) return;
  if (!doneFirstUpdate) return;
  if (Window.IsMinimized) return;
  if (BeginDraw()) {                           // <- false skips Draw AND EndDraw
      gameTime.TotalGameTime   = totalGameTime;
      gameTime.ElapsedGameTime = lastFrameElapsedGameTime;
      gameTime.IsRunningSlowly = drawRunningSlowly;
      Draw(gameTime);
      EndDraw();
      doneFirstDraw = true;
  }
} finally { lastFrameElapsedGameTime = TimeSpan.Zero; }
```

So `false` skips **both** `Draw` and `EndDraw`. The project's settled rule that
a source result and an error are separate channels applies unchanged — it is
already how `IGraphicsDeviceManager.BeginDraw` is projected — and the Boolean is
never collapsed into the error. A refused call answers `(false, err)`: it does
not admit the frame *and* reports the refusal, which is the only combination
that is not a lie in either channel.

## The native hooks, measured

`CNA_GameFrameHooks` (canonical `CNA/C/runtime.h`) carries five members;
CNA-Go's bridge installs exactly one of them, `initialize`, and has since
Foundation 11:

```c
typedef struct CNA_GameFrameHooks {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_GameLifecycleCallback initialize;   // installed
    CNA_GameLifecycleCallback begin_run;    // audited, NOT installed
    CNA_GameLifecycleCallback end_run;      // audited, NOT installed
    CNA_GameBeginDrawCallback begin_draw;   // audited, NOT installed
    CNA_GameLifecycleCallback end_draw;     // audited, NOT installed
    void* context;
} CNA_GameFrameHooks;
```

A throwaway C probe in the shared `build-probe/` directory installed all five
against the pinned artifact and measured this ordering:

```text
cna_game_run
  initialize      -> load_content -> begin_run -> ACTIVATED
  update -> begin_draw -> draw -> end_draw
  update -> begin_draw -> draw -> end_draw
  ...
  cna_game_request_exit
  exiting callback -> EXITING -> end_run
cna_game_run returns
```

against the reference:

```text
RunGame(useBlockingRun: true)
  Services.GetService(IGraphicsDeviceManager) -> CreateDevice()
  Initialize()                        // base body loads content when a device exists
  inRun = true
  BeginRun()
  Update(gameTime)                    // the priming update
  host.Run() { loop: Tick -> Update, DrawFrame -> BeginDraw/Draw/EndDraw }
  EndRun()
```

**Every position corresponds.** `begin_run` fires after initialization and
content loading and before the first update, exactly where `BeginRun()` sits;
`end_run` fires after the loop ends, exactly where `EndRun()` sits; `begin_draw`
and `end_draw` bracket each `draw`, exactly as `DrawFrame` brackets `Draw`.

The Boolean channel corresponds too, and was measured rather than assumed. With
`out_should_draw` pre-set to `CNA_TRUE` by the ABI and set to `CNA_FALSE` on two
frames:

```text
updates=6  begin_draws=5  draws=3  end_draws=3
```

A refused frame delivered **neither `draw` nor `end_draw`** — the same shape as
`if (BeginDraw()) { Draw(); EndDraw(); }`. The sixth update requested exit and
produced no `begin_draw` at all, which is `DrawFrame`'s `if (ShouldExit) return`.

## Why the hooks stay uninstalled

The correspondence is exact, and CNA-Go still does not install them. Three
reasons, in order of weight.

**1. Base behavior is never automatic — Foundation 31's rule, applied.** In CLR
the derived class decides whether and when the base runs, and the call site *is*
the semantics. Installing `begin_draw` and forwarding it into `Game.BeginDraw()`
would run the base body at a position CNA-Go picked and would make that base
call mandatory. There is no override mechanism for these four members today, so
the forwarding would be provably inert — but it would prejudge the exact
decision an override mechanism has to make.

**2. It would be inert.** With no `IGraphicsDeviceManager` reachable, all four
base bodies are observably empty: two are `ret`, `BeginDraw` always answers
true, `EndDraw` always does nothing. Two of the hooks fire **per frame**, so
installing them buys a native callback on every frame that provably cannot
change any observable.

**3. Nothing is hidden by not installing them.** The measured ordering above is
recorded here, the hook table is measured by `native_abi` (`sizeof` and the
`context` offset of `CNA_GameFrameHooks` have been in the report since
Foundation 11), and the four unbound members are named in the handoff. The
frontier is documented, not silent.

## The public design decision that remains open

If a consumer is ever to *override* `BeginRun`, `EndRun`, `BeginDraw` or
`EndDraw`, CNA-Go needs a mechanism, and there is more than one materially
different public shape for it. Per the session's own stop rule, that branch is
stopped and reported rather than guessed at.

What is already settled and must not be reopened:

- **`GameCallbacks` keeps exactly five members.** Adding four more would break
  every existing external implementation. Two tests hold this — one in the
  framework package and one in the external canary, whose conformer predates
  this work.
- **No `GameBaseBeginRun`/`GameBaseEndRun`/`GameBaseBeginDraw`/`GameBaseEndDraw`.**
  The `GameBase...` registry is keyed by the `GameCallbacks` members and its
  closure rule rejects a helper for anything else. Adding one because the names
  look symmetrical is precisely the mistake that rule exists to stop; the base
  bodies are reachable as `Game` methods, which is where Microsoft declared
  them.

The alternatives that remain plausible, none of which is uniquely determined by
existing CNA-Go style:

1. **An optional secondary interface** — for example a `GameFrameCallbacks` that
   a `GameCallbacks` implementation may also satisfy, discovered by a runtime
   type assertion at `NewGame`. Additive and non-breaking, but it introduces a
   second callback contract and has to answer what a partial implementation
   means.
2. **Per-hook optional capability interfaces** — one single-method interface per
   hook, each independently assertable. Finer-grained and avoids the partial-
   implementation question, but multiplies exported contracts by four.
3. **Explicit registration** — package-level functions that install one override
   at a time on a constructed `Game`. Keeps every contract single-purpose and
   makes installation an observable act, but introduces mutable per-Game
   callback state that nothing else in the binding has.

Each of the three also has to decide, separately, whether the native hooks are
installed eagerly or only once an override is present, and what `BeginDraw`'s
Boolean means when an override is absent.

**None of this blocks the four members**, which is why they are complete now: a
projected protected virtual is satisfied by a method that runs the reference's
base body, exactly as `GameComponent`'s are.

## Scoreboard

```text                                     before   after
TARGET_MEMBERS                             1787    1791
TOTAL_DIAGNOSTICS                           299     295
MISSING_MEMBER                              164     160
Game missing members                         26      22

behavior corpus                             625     628
external canary tests                        39      42

ABI (25/75/107/31/3/10)                  unchanged
GAME_BASE_CALL_ADAPTERS                       5       5
GameCallbacks members                         5       5
```

`COMPLETE_TYPES`, `PARTIAL_TYPES`, `MISSING_TYPE`, `REFERENCE_MEMBERS`,
`EXPECTED_GO_MEMBERS` and the four XNA-base-frontier counters are all unchanged.
Every mismatch, leak, allowlist and unmeasured counter is zero.
