# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 33 are complete. Milestones 26 through 33 were
produced across two sessions as **local commits that have not been pushed**;
`develop` is ahead of `origin/develop` by those milestone commits plus the docs
commits that record them.

```text
SESSION START   HEAD = origin/develop = cc8d18001935aacf7179341d14c034bc18c2b0a4
SESSION END     HEAD = 0cfe535
LAST SOURCE-BEARING COMMIT = 5653899
                origin/develop unchanged at cc8d180
                worktree clean, git diff --check clean
                PUSHED = false
```

| #  | commit    | milestone                                             | types | Go identities |
| -- | --------- | ----------------------------------------------------- | ----- | ------------- |
| 30 | `b71d3c1` | Game's managed Components and Services; the engine     | 0     | 2             |
| 31 | `d0b4913` | the explicit base-call architecture                    | 0     | 0             |
| 32 | `e9eb653` | GameComponent; compiler-checked interface conformance  | 1     | 19            |
| 33 | `5653899` | the XNA-to-XNA base frontier, measured                 | 0     | 0             |

One type completed. Two of the four commits complete nothing on purpose: one is
language support that makes an existing surface usable, one is measurement.

## Scoreboard

```text                                     start    final
TARGET_TYPES                              121      122
TARGET_MEMBERS                           1755     1776
TOTAL_DIAGNOSTICS                         313      310
MISSING_TYPE                              136      135
MISSING_MEMBER                            177      175
COMPLETE_TYPES                            116      117
PARTIAL_TYPES                               5        5

REFERENCE_MEMBERS / REFERENCE_XNA_MEMBERS 2964     2964
EXPECTED_GO_MEMBERS                      3255     3255
BCL_INHERITED_MEMBER_PROJECTIONS           12       12

GAME_BASE_CALL_ADAPTERS                     0        5
GAME_BASE_CALL_DEFERRED_STEPS               0        4
DECLARED_INTERFACE_CONFORMANCE              0        2
XNA_BASE_RELATIONSHIPS                      0       12
XNA_BASE_DERIVED_TYPES                      0       41
XNA_DEFERRED_BASE_BLOCKERS                  0       25
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED    0      245

mutation inventory                        478      500
behavior corpus                           598      620
external canary tests                      16       34
```

Every mismatch, leak, allowlist and unmeasured counter is zero throughout.
**`MISSING_MEMBER` moved by exactly two**, both in the coherent slice:
`Game::Components` and `Game::Services`. No other partial type was touched.

## Game is a hybrid host, and the split is per member

The native CNA runtime owns the host, the frame loop, the window, the device and
the platform — `GameHost`'s role in the reference. Go owns the managed CLR
state. The split is read from IL per member, never assumed per type:

```text
Game::get_Components   ldarg.0; ldfld gameComponents; ret     7 bytes
Game::get_Services     ldarg.0; ldfld gameServices;   ret     7 bytes
```

Both fields are assigned once in the constructor and never reassigned, so the
getters cannot fail, allocate nothing, and return one stable identity. Routing
either through the C ABI would invent a native owner and a native failure mode
the reference does not have. **No C ABI function was added this session and no
CNA C++ changed.**

`GameCallbacks` is unchanged. Its five members are exactly what they were.

## Base behavior is never automatic

The load-bearing decision of the session. In CLR the derived class decides
whether and when the base runs, and the call site *is* the semantics: omitting
`base.Update(t)` means the component loop does not run that frame; moving it
changes what runs before what.

Running the base body around each callback would make every override a mandatory
base call at a position CNA-Go picked — a different contract that resembles
XNA's. So base bodies are reached explicitly:

```go
GameBaseInitialize(game) error
GameBaseLoadContent(game) error
GameBaseUpdate(game, gameTime) error
GameBaseDraw(game, gameTime) error
GameBaseUnloadContent(game) error
```

Package-level functions, so `Game`'s projected member surface gains no name
Microsoft never declared. They are **measured language support**, not XNA
identity: the registry pins each one to a real protected virtual `GameCallbacks`
projects, forbids an extra `GameBase*` helper, requires one per callback member,
and admits the names into `adapterFunctions` **from the registry** — there is no
allowlist. 13 negative controls; `REFERENCE_MEMBERS` and `EXPECTED_GO_MEMBERS`
did not move.

## Reference quirks now preserved — do not "correct" them

Everything in the Foundation 17-29 list still holds, plus:

- **Game does NOT sort its components.** It keeps two derived lists ordered
  incrementally and copies them per frame. There is no sort anywhere.
- **`UpdateOrderComparer` and `DrawOrderComparer` return 0 only for reference
  identity or two nulls; EQUAL orders return 1.** So `BinarySearch` can only
  succeed on a component already in the list, and `if (index < 0)` is a **"not
  already present"** guard, not an "equal order found" guard.
- **Ties are stable because of an explicit forward walk** past the run of
  equal-order elements, not because of a stable sort. `Array.BinarySearch`
  guarantees no ordering among equals, so the walk is what makes it
  deterministic. An order change re-places the component at the **end** of its
  new tie run.
- **The order-changed handlers read the SENDER, not the event args**, and
  re-place without re-subscribing. That is why `GameComponent` raises with
  `this`.
- **`GameComponent.OnEnabledChanged`/`OnUpdateOrderChanged` accept a sender,
  ignore it, and raise with `this`.** The engine depends on it.
- **`inRun` is raised only after `Initialize` returns**, so a component added
  from inside the override is still queued — and the drain re-reads `Count`, so
  it is initialized by that same drain — while one added later initializes
  immediately.
- **Initialization order is ADD order, not `UpdateOrder`.** `notYetInitialized`
  is a plain list; only the update and draw lists are ordered.
- **base `Initialize` calls `Initialize()` before `RemoveAt(0)`**, so a failing
  component stays at the head of the queue and a retry resumes there.
- **base `Update`/`Draw` snapshot before iterating**, so a mid-frame mutation
  applies to the next frame — but `Enabled`/`Visible` are read at **iteration**
  time, so a component disabled earlier that frame is skipped.
- **There is no `try`/`finally` in base `Update`.** A component that panics
  leaves the snapshot list populated and the next frame appends to it. The
  straight-line `Clear` is reproduced as such.
- **base `Draw` touches no device at all** — that is `BeginDraw`/`EndDraw`.
- **`GameComponent.Dispose` removes from `Components` before it announces, and
  is not idempotent**: a second `Dispose` raises `Disposed` again.
- **`GameComponent`'s constructor does not null-check its Game.**
- **`GameComponent.Initialize` is fallible because of the contract it
  implements**, not its own body, which is a bare `ret`. First member in the
  profile with that provenance.

## Two new general verifier rules

### Compiler-checked declared interface conformance

> a **complete** projected class that CLR metadata says implements a projected
> XNA interface must satisfy that interface's Go projection, on the **pointer**
> method set, and `go/types` must say so

Previously only the PackedVector family was checked that way; everything else
was structural, which cannot catch a method with the right name and the wrong
signature. It runs only for complete types, because a partial type's gap is
already `MISSING_MEMBER`. Five negative controls built from synthetic
`go/types` evidence.

### The XNA-to-XNA base frontier

Foundation 29 made a deferred **BCL** base name its blockers. There is a second
frontier and it was silent: `Texture2D` inherits nine public members from
`Texture` and `GraphicsResource` that CNA-Go does not project, and nothing
recorded it. `SpriteBatch` had the same silence over seven.

Twelve relationships, 41 derived types, 25 blockers, 245 unprojected inherited
public members. The substantive rule:

> **no derived type of a DEFERRED XNA base may be reported COMPLETE**

`Texture2D` and `SpriteBatch` are legitimately partial today, and that is now a
checked fact rather than a coincidence.

## A mapper defect this session fixed

The property branch rebuilt each accessor's results from scratch but **inherited**
the whole-member fallibility flag, making the accessor-level decision
one-directional: it could raise fallibility on a pure-managed owner but never
lower it on a native-backed one. `Game::Components` is the first get-only stored
property on a fallible-by-default owner and exposed the skew. Guarded now by a
structural invariant over **all 3255** expected members: every member's
`ErrorAdded` equals whether its results end in `error`.

## The frontier — measured, not summarised

**The exact next architecture decision is XNA-to-XNA class inheritance**, and
every remaining component type needs it. Projecting a derived XNA class needs
three things CNA-Go does not have:

1. a composition and forwarding rule for a base that is itself an XNA identity
   (the BCL rule holds a private *generic adapter*; an XNA base is a projected
   type with its own identity, constructor and scoreboard entry);
2. a **third provenance class** beside XNA-declared and BCL-inherited, so no
   member is counted twice — this moves `EXPECTED_GO_MEMBERS` for 41 types;
3. an override adapter for the base's protected virtuals, which is Foundation
   31's architecture generalised from one class to a family.

Exported Go embedding of an XNA base stays refused by `BASE_MAPPING_MISMATCH`.

### `DrawableGameComponent` — blocked five ways

XNA base composition; a derived-class override adapter for its protected
`LoadContent`/`UnloadContent`; `get_GraphicsDevice` returning a partial
`Graphics.GraphicsDevice` across the package boundary; an `Initialize` that
throws `InvalidOperationException(MissingGraphicsDeviceService)` when the
service is absent — which it always is; and a `Dispose(bool)` that base-calls
`GameComponent.Dispose(bool)` non-virtually.

### `GamerServicesComponent` — blocked three ways

XNA base composition; `Game.Window.Handle`, where `GameWindow` is a missing
type and `Game::Window` a missing member; and `GamerServicesDispatcher`, which
lives in `Microsoft.Xna.Framework.GamerServices.dll` — **not one of the seven
pinned assemblies**. Admitting it would not help: CNA has no GamerServices
runtime, so every member would be inert.

### `GraphicsDeviceManager` service publication — audited, blocked

The constructor registers itself under **both** `IGraphicsDeviceManager` and
`IGraphicsDeviceService`, with the duplicate check **before** the registration
and on the manager key only. CNA-Go can perform neither: `AddService` reproduces
the reference's assignability check, and the partial manager satisfies neither
contract — `CreateDevice`/`BeginDraw`/`EndDraw` are missing, `GraphicsDevice()`
is missing, and **the four device events have no raise path in CNA at all**.
Faking either would put an event on a contract that never fires.

Worth recording for whoever resolves it: the reference registers the services
**before** it touches `game.Window`, so the managed half precedes every blocked
step and nothing observable would be reordered by supplying it later.

### `Game`'s four events — the raise path EXISTS and is unbound

This is the most actionable unstarted work in the repository. `Game::Activated`,
`Deactivated`, `Exiting` and `Disposed` are eight missing accessors, and CNA
already publishes all four signals as canonical, **unbound** surface:

```c
CNA_GAME_EVENT_ACTIVATED    0
CNA_GAME_EVENT_DEACTIVATED  1
CNA_GAME_EVENT_DISPOSED     2
CNA_GAME_EVENT_EXITING      3

CNA_C_API CNA_Result cna_game_subscribe(CNA_Handle, CNA_GameEvent,
                                        CNA_GameEventCallback, void*,
                                        CNA_GameEventRegistrationHandle*);
CNA_C_API CNA_Result cna_game_unsubscribe(...);
```

They were **deliberately not implemented this session**: activation and disposal
are outside the components-and-services slice. Binding them would be additive
canonical surface with no CNA C++ change, and would move the ABI counters from
23 bound functions — a legitimate additive delta. Everything needed is audited.

`begin_run`, `end_run`, `begin_draw` and `end_draw` are likewise unbound in
`CNA_GameFrameHooks`; they correspond to four more missing `Game` members and
are not `GameCallbacks` members.

## Native callback order — audited

```text
CNA (documented contract)  initialize -> [components + device] -> load_content
                           -> begin_run -> update -> draw
XNA (Game::RunGame)        CreateDevice -> Initialize -> inRun = true
                           -> BeginRun -> Update -> loop{Update, Draw}
```

**The relative order of everything CNA-Go owns is preserved.** Managed component
initialization happens in the Initialize step and content loading after it, in
both; the device is created before content loads, in both. Where CNA-Go
substitutes the native host for `GameHost` it is recorded rather than hidden:
`inRun` is raised on the native `initialize` hook boundary and lowered when the
blocking `Run` returns, which are the two points `RunGame` assigns it.

## Deliberate concurrency projection

`GameComponent.Dispose(bool)`'s `lock (this)` is projected with `TryLock`. CLR's
`Monitor` is reentrant per thread and reentry is reachable here — a `Disposed` or
`ComponentRemoved` handler may dispose again — so a plain `Lock` would
**deadlock** where the reference merely recurses. The divergence is that a
genuinely concurrent second disposer proceeds instead of blocking; component
state is owner-thread state, the binding promises no cross-goroutine safety for
it, and a deadlock is the worse of the two errors. The whole suite is race-clean.

Nothing else gained a lock. Game's component state is single-threaded by the
same contract.

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~320s under -race
go build ./... && go build -trimpath ./...
go run ./tools/api_compat --mode strict -report "" -missing ""   # expected red: 310 deferred
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
report mode, and send native reports to an explicit scratch `-output`.

The mscorlib IL cache added this session lives at
`~/deps/xna-il-cache/mscorlib.il` (42 MB, from the admitted 4.0.30319.1 binary)
alongside the seven XNA disassemblies. It is reusable; do not regenerate it into
a scratch directory.

### Deterministic artifact, isolated consumer, external canary

```sh
git ls-files -z | sort -z | tar --null --files-from=- \
  --transform 's,^,cna-go/,' --owner=0 --group=0 --numeric-owner \
  --mtime='@0' --mode='u+rw,go+r,go-w' --sort=name -cf - | gzip -n > OUT.tar.gz
```

At `5653899`, the last source-bearing commit, that yields sha256
`45c09221be12d7578c5433b82bf39e3e89e27d05ed83c80a1e19355c9f293d71` over 274
entries, reproduced twice in the same run. The docs commit that follows changes
the artifact hash and nothing else — it edits three tracked files and adds none
— so re-run the command above to get the current one. Extracted into
`build-consumer/isolated`, it passes every gate with no development-checkout
dependency and regenerates `api-compat-report.json`,
`behavior-corpus-report.json`, `packed-vector-exhaustive-report.json` and
`missing-type-inventory.md` **byte-identically**. The Foundation-1 consumer
fixture in `build-consumer/consumer` builds against it and runs at exactly 60
and 600 native Draw callbacks.

The external canary now runs **34** tests:

```sh
go run ./tools/external_consumer -source build-consumer/isolated/cna-go
```

All twelve required component-loop claims are proved from outside: construct a
Game, obtain stable Components and Services identities, add components,
subscribe to collection and component events, call each base explicitly, observe
update and draw ordering (deliberately the reverse of each other), prove that
**omitting** the base call prevents base iteration, prove that **moving** the
base call changes ordering relative to user code, prove that disposing a
component removes it from `Game.Components` and stops its iteration, and prove
no native handle appears anywhere in the family's signatures.

## Native provenance — unchanged

CNA was **not rebuilt**, and no C ABI function was added.

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED          23/67/96/28/2/5, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED     native_stress reproduces every counter byte-identically
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED
```

`native_abi` reproduces the committed report key for key, differing only in
`header_root`, which the committed evidence stores normalized.

## Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean.
`SOURCE_CHANGED=NO`. Ruby and Swift sibling worktrees were not touched.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage and most of XNA remain
unimplemented.

What is new and what it does **not** claim: `Game` now owns a real component
engine, and a consumer can build one from outside the repository — but nothing
in it renders. `GameComponent.Update` and `IDrawable.Draw` are called on the
owner thread with a tick-exact `GameTime`; what a component does there is its
own. `Game.Services` is a working registry that **nothing in the binding
registers into**, because the reference's only registrar is
`GraphicsDeviceManager` and it cannot satisfy either service contract.

```text
FOUNDATION_MILESTONE_30_COMPLETE=true
FOUNDATION_MILESTONE_31_COMPLETE=true
FOUNDATION_MILESTONE_32_COMPLETE=true
FOUNDATION_MILESTONE_33_COMPLETE=true
PUSHED=false
SAFE_MANAGED_COMPONENT_FRONTIER=EXHAUSTED
NEXT_STEP=ARCHITECTURE_DECISION_XNA_TO_XNA_INHERITANCE
NEXT_STEP_ALTERNATIVE=BIND_CNA_GAME_EVENT_SIGNALS_FOR_GAME_ACTIVATED_DEACTIVATED_EXITING_DISPOSED
```
