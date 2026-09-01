# CNA-Go resumable handoff

> **This file records the session that produced Foundation 38 through 43 and is
> kept for its detail.** Foundation 44 through 61 followed, each with its own
> evidence document under `docs/`, and `plan.md` carries the index and the
> normative rules they settled. The scoreboard below is the one that session
> measured; the LIVE one is whatever
> `go run ./tools/api_compat --mode report` prints now.


## Current state

Foundation Milestones 1 through 43 are complete. Milestones 38 through 43 were
produced in one session, as the milestone commits listed below plus the docs
commit that records them. Whether they have reached `origin/develop` is a live
fact rather than a stored one — see the note under the block.

```text
SESSION START HEAD          = 355cc27be47e0edee29d7d77951eddbfe277cebc
                              (== origin/develop at session start)
LAST SOURCE-BEARING COMMIT  = 0372390
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

`PUSHED` follows the same rule and for the same reason. The deterministic
artifact hash further down is pinned to the last source-bearing commit for the
same reason.

| #  | commit    | milestone                                                | types | Go identities |
| -- | --------- | -------------------------------------------------------- | ----- | ------------- |
| 38 | `50c77a9` | optional per-hook Game frame overrides                    | 0     | 0             |
| 39 | `9d0c095` | Game's disposal surface, `Disposed` raise site corrected  | 0     | 3             |
| 40 | `41ef6d0` | the base-typed public signature inventory                 | 0     | 0             |
| 41 | `d284b6b` | the XNA-to-XNA inheritance projection                     | 0     | 0             |
| 42 | `d404a3f` | Game's timing and presentation state                      | 0     | 10            |
| 43 | `0372390` | `Game.GraphicsDevice`, corrected resource strings         | 0     | 1             |

Fourteen members completed, all on `Game`. No type became complete and none was
meant to. Three of the six milestones complete nothing on purpose: two are
measurement and architecture, one is a language mechanism.

## Scoreboard

```text                                     start    final
TARGET_TYPES                              122      122
TARGET_MEMBERS                           1791     1805
TOTAL_DIAGNOSTICS                         295      281
MISSING_TYPE                              135      135
MISSING_MEMBER                            160      146
COMPLETE_TYPES                            117      117
PARTIAL_TYPES                               5        5

REFERENCE_MEMBERS / REFERENCE_XNA_MEMBERS 2964     2964
EXPECTED_GO_MEMBERS                      3255     3279
XNA_DECLARED member projections           3243     3243
BCL_INHERITED_MEMBER_PROJECTIONS           12       12
XNA_INHERITED_MEMBER_PROJECTIONS            0       24

GAME_BASE_CALL_ADAPTERS                     5        5
GAME_NATIVE_SIGNALS                         4        4
GAME_NATIVE_SIGNAL_RAISE_SITES              3        3
GAME_NATIVE_SIGNALS_LIFECYCLE_ONLY          -        1
GAME_MANAGED_EVENT_RAISE_SITES              -        1
GAME_FRAME_HOOKS                            4        4
GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE      0        4
GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES       0        4
GAME_CALLBACKS_MEMBERS                      -        5
XNA_BASE_RELATIONSHIPS                     12       12
XNA_BASE_DERIVED_TYPES                     41       41
XNA_DEFERRED_BASE_BLOCKERS                 25       23
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED  245      227
XNA_BASE_TYPED_SIGNATURE_POSITIONS          -       51
XNA_BASE_SUBSTITUTABILITY_NONE              -        3
XNA_BASE_SUBSTITUTABILITY_LATENT            -        9
XNA_BASE_SUBSTITUTABILITY_LIVE              -        0
XNA_COMPOSED_BASE_RELATIONSHIPS             0        1

mutation inventory                        531      565
behavior corpus                           628      647
external canary tests                      42       64
native ABI mutation controls               19       30
native stress scenarios                     4        7
internal/interop tests                      8       15
runtime capability rows                    41       43
```

Every mismatch, leak, allowlist and unmeasured counter is zero throughout.
**`MISSING_MEMBER` moved by exactly fourteen, every one of them on `Game`**:
`Game` went from 22 missing members to 8. No other type was touched.

Per-type missing members now: `GraphicsDevice` 70, `GraphicsDeviceManager` 40,
`SpriteBatch` 16, `Texture2D` 12, `Game` 8.

## ABI — CNA was not rebuilt

```text                       before   after
BOUND_FUNCTIONS                  25      31
PROTOTYPE_TYPE_POSITIONS         75      91
C_GO_MEASUREMENTS               107     128
LAYOUTS                          31      36
CALLBACKS                         3       3
CONSTANTS                        10      15

MISSING_HEADER_SYMBOLS            0       0
MISSING_LIBRARY_SYMBOLS           0       0
ABI_MISMATCHES                    0       0
```

Six symbols already exported by the pinned artifact are now reached:
`cna_game_set_is_mouse_visible`, `cna_game_set_is_fixed_time_step`,
`cna_game_set_target_elapsed_time_ticks`,
`cna_game_set_inactive_sleep_time_ticks`, `cna_game_reset_elapsed_time` and
`cna_game_suppress_draw`. `LAYOUTS` and `CONSTANTS` moved because CNA-Go now
assigns four `CNA_GameFrameHooks` members conditionally, so their offsets and
their ORDER became load-bearing and are pinned in **both** translation units.

`CALLBACKS` stays at 3 and its claim became true: `CNA_GameEventCallback` was
the only one actually pinned by an assignment probe, and
`CNA_GameLifecycleCallback` and `CNA_GameBeginDrawCallback` now are too.

## The three decisions this session was given, and what was done

### 1. Optional Game frame-hook overrides — implemented as specified

`GameCallbacks` keeps **exactly five members**. Four optional, **unexported**,
single-method structural capabilities on the same object passed to `NewGame`:

```go
type gameBeginRunOverride   interface{ BeginRun(*Game) error }
type gameEndRunOverride     interface{ EndRun(*Game) error }
type gameBeginDrawOverride  interface{ BeginDraw(*Game) (bool, error) }
type gameEndDrawOverride    interface{ EndDraw(*Game) error }
```

A consumer opts in by declaring the exported method and never names the type.
Discovery happens once, in `NewGame`. There is no registration API, no exported
capability contract, and no `GameBase*` helper — the projected methods on `Game`
ARE the base bodies, so `game.BeginDraw()` inside an override is the Go
projection of `base.BeginDraw()`.

Each native hook is installed **iff** its capability exists, through a mask
`cna_go_game_create` derives from the same captured fields the dispatch reads.

Measured against the pinned artifact:

```text
FRAME_HOOK_BEGIN_RUN_DELIVERIES        20     exactly one per run
FRAME_HOOK_REFUSED_FRAMES             120     each skipped draw AND end_draw
FRAME_HOOK_END_DRAW_DELIVERIES         80     == admitted frames that installed it
FRAME_HOOK_UNINSTALLED_DELIVERIES       0
```

The 120 refusals are the proof that the OVERRIDE's answer reaches CNA: the base
admits every frame (verified per call), so a skipped draw is something only the
override could have caused. The zero is the subset scenario — a Game declaring
only `BeginDraw` never received the other three hooks.

### 2. `Game.Disposed` raise-site fidelity — corrected, not preserved

`CNA_GAME_EVENT_DISPOSED` no longer raises the public event. `Game::Disposed`
has exactly one raise site in the whole class, the tail of `Dispose(bool)`, and
that is where CNA-Go raises it.

The registry rule changed with it. It was "every event is bound to one canonical
signal through the raise path the reference uses", which is satisfied by a
binding whose semantics do not align. It is now:

> every projected XNA event must have its AUTHORITATIVE XNA raise path, and a
> native signal may IMPLEMENT that path only when the semantics align

measured as two vocabularies that are one decision seen from two ends:
`NATIVE_HOST_SIGNAL` ↔ `PUBLIC_EVENT_RAISE` for the three the host raises, and
`MANAGED` ↔ `LIFECYCLE_ONLY` for `Disposed`.

The native signal stays bound, delivered and counted — it is what proves a
registration outlives `cna_game_destroy` — and raises nothing public.

```text
GAME_NATIVE_DISPOSAL_SIGNALS               100    unchanged
GAME_DISPOSED_RAISED_DURING_RUN              0    the correction, measured
GAME_DISPOSED_RAISED_BY_MANAGED_DISPOSE     80    == 2 x 40 cycles
```

`80 = 2 × 40` is the non-idempotency proof. `Game` carries no disposed flag, so
a second `Dispose` disposes again and raises again; a projection that had
invented a flag would report half.

### 3. XNA-to-XNA inheritance — measured first, then decided

**Foundation 40 measured the requirement before any design was chosen.** Fifty-one
public signature positions in the profile name a class another class derives
from. Three families — `GameComponent`, `GraphicsResource`, `MathTypeConverter`,
carrying **25 of the 41 derived types** — are named in **zero** of them. None is
live.

For a family named in zero positions, private composition is not a compromise:
there is no position in the contract for a derived value to flow through, so no
public reference abstraction can be justified by anything Microsoft declared.

**Foundation 41 then projected the rule**: private named composition plus
explicit measured forwarding, never Go embedding, no public `Base`/`Parent`/`As…`
accessor. `XNA_INHERITED` joined `XNA_DECLARED` and `BCL_INHERITED` as the third
provenance class, asserted disjoint and exhaustive.

`GameComponent` is the first `COMPOSED` relationship. `COMPOSED` is about
inheritance, not completeness: `XNA_COMPOSED_DERIVED_TYPES_PROJECTED = 0`.

## Reference quirks now preserved — do not "correct" them

Everything in the Foundation 17-37 list still holds, plus:

- **`Game::Dispose(bool)` is NOT idempotent.** There is no disposed flag
  anywhere in the class. Every call re-copies the components, disposes each
  again, and raises `Disposed` AGAIN.
- **The component snapshot in `Dispose(bool)` is required, not defensive.**
  `GameComponent::Dispose(bool)` runs `Game.Components.Remove(this)`, so
  disposing a component MUTATES the collection being walked.
- **`Dispose(bool)` has no try/catch.** A failing component leaves every later
  one undisposed and `Disposed` unraised.
- **`set_InactiveSleepTime` accepts ZERO and `set_TargetElapsedTime` does not.**
  `op_LessThan` versus `op_LessThanOrEqual`; one IL instruction. Only the
  resource KEY is called `InactiveSleepTimeCannotBeZero` — the string it names
  says "greater than or equal to zero", which is what the comparison admits.
- **`TargetElasped` is Microsoft's own typo** in the resource key
  `Resources::get_TargetElaspedCannotBeZero`, and is left as it is.
- **`Game::get_GraphicsDevice` FALLS BACK** to resolving `IGraphicsDeviceService`
  out of `Services` when its cached field is null. That fallback is the only
  reason the member is reachable, because the cached field is never assigned.
- **`Game::.ctor` does not assign `isMouseVisible`**, so it starts false, while
  `isFixedTimeStep` starts true and `targetElapsedTime` starts at 166,667 ticks.
- **`GameWindow::set_Title` suppresses an unchanged value** and calls the
  protected `SetTitle` only when it actually changed.

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race -p 1 ./...   # api_compat takes ~475s under -race; -p 1 avoids an OOM kill
go build ./... && go build -trimpath ./...
go run ./tools/api_compat --mode strict -report "" -missing ""   # expected red: 281 deferred
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

`go test -race ./...` was OOM-killed once in this session when it ran every
package in parallel. `-p 1` runs them serially and completes.

The IL caches live at `~/deps/xna-il-cache/` — the seven XNA disassemblies plus
`mscorlib.il`. **The retained assemblies themselves are NOT in `~/deps`.** The
`.resources` streams that carry the exact exception messages were read from a
copy of `Microsoft.Xna.Framework.Game.dll` found in another session's scratch
tree, which is disposable. Retaining the seven assemblies somewhere durable
under `~/deps` is worth doing before the next milestone that needs a message
string.

### Deterministic artifact, isolated consumer, external canary

```sh
git ls-files -z | sort -z | tar --null --files-from=- \
  --transform 's,^,cna-go/,' --owner=0 --group=0 --numeric-owner \
  --mtime='@0' --mode='u+rw,go+r,go-w' --sort=name -cf - | gzip -n > OUT.tar.gz
```

At `0372390`, the last source-bearing commit, that yields sha256
`723ccf6617bdcca6207799698e00dc559c1b758e38e48b59790efd73c049744f` over 298
entries, reproduced twice in the same run. The docs commit that follows it edits
tracked files and adds none, so the entry count stays 298 while the hash does
not; re-run the command above for the current value.

Extracted into `build-consumer/isolated`, it builds and tests with `GOWORK=off`,
and regenerates `api-compat-report.json`, `behavior-corpus-report.json`,
`packed-vector-exhaustive-report.json`, `missing-type-inventory.md` and
`runtime-capabilities.md` **byte-identically**.

The external canary now runs **64** tests:

```sh
go run ./tools/external_consumer -source build-consumer/isolated/cna-go
```

The twenty-two new ones prove, from a module that cannot see
`internal/interop` at all: any SUBSET of the four frame-hook overrides is
accepted and an omitted one is never satisfied by accident; an explicit base
call invokes the base only and ordering around it follows source order;
`Game.Disposed` fires only from `Dispose` and fires again on a second call; both
projected spellings of `IDisposable::Dispose` are found on a consumer's own
component and one declaring neither is skipped; the timing getters are
infallible and the two `TimeSpan` setters have different boundaries; and
`graphics.GameGraphicsDevice` reports the reference's `InvalidOperationException`
with no registered service and forwards a registered one's device unchanged.

## Native provenance — unchanged

CNA was **not rebuilt**, and no C ABI function was added to CNA.

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED          31/91/128/36/3/15, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED     native_stress reproduces every counter
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED
```

`native_stress` gained three scenarios and eighteen counters, and one counter was
renamed. `GAME_EVENT_DISPOSED_DELIVERIES` became
`GAME_NATIVE_DISPOSAL_SIGNALS` because it now counts the native signal rather
than a public event raise — the value is unchanged at 100, which is the point:
the bridge-lifetime evidence Foundation 37 established is not weakened by the
raise-site correction.

`GAME_EVENT_DEACTIVATED_DELIVERIES` is still deliberately zero. Inventing a way
to move it would be fabricating evidence.

## Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean.
`SOURCE_CHANGED=NO`. It builds against the live checkout and runs at exactly 60
and 600 native Draw callbacks, as does the Foundation-1 consumer fixture in
`build-consumer/consumer` against the extracted artifact. Ruby and Swift sibling
worktrees were not touched.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage and most of XNA remain
unimplemented.

What is new and what it does **not** claim: a consumer may now override any
subset of `Game`'s four frame-boundary virtuals and configure its timing, and
both reach the real native loop — but `Deactivated` has still never been
delivered in this environment; `Game.Window`, `Game.Content` and
`Game.LaunchParameters` do not exist; and `Game.GraphicsDevice` answers a device
only when a CONSUMER registers an `IGraphicsDeviceService`, because CNA-Go
publishes none.

## What is genuinely blocked, and the exact next work

### 1. `GameWindow` — fully researched, one open sub-question

This is the highest-value next milestone and it is de-risked. `GameWindow` is
`public abstract` with an `assembly` constructor, so it comes from `Game.Window`
and a consumer never constructs one. Twenty public/protected members → about 25
Go identities, all in the framework package: no Graphics dependency.

CNA covers nearly all of it, and every needed function is in the pinned binary:

| member | CNA function |
| -- | -- |
| `AllowUserResizing` get/set | `cna_game_window_get/set_allow_user_resizing` |
| `ClientBounds` | `cna_game_window_get_client_bounds` |
| `CurrentOrientation` | `cna_game_window_get_current_orientation` |
| `Handle` | `cna_game_window_get_native_handle_ext` |
| `ScreenDeviceName` | `..._get_screen_device_name_size` + `..._copy_screen_device_name` |
| `BeginScreenDeviceChange` | `cna_game_window_begin_screen_device_change` |
| `EndScreenDeviceChange(string,int,int)` | `cna_game_window_end_screen_device_change` |
| `SetTitle` | `cna_game_set_window_title` |
| the three events | `cna_game_window_subscribe`, released by `cna_game_unsubscribe` |

Three things are already settled and need no new decision:

- `Handle` → `uintptr`. `GameWindow::Handle` is one of the six members the
  `pointerSizedHandles` rule names explicitly.
- `Title` is a **managed field** in the abstract base — `get_Title` is one
  `ldfld` — and `set_Title` null-checks, suppresses an unchanged value, stores,
  and only then calls the protected `SetTitle`. So the getter does not bind a
  native call at all.
- `EndScreenDeviceChange(string)` is concrete and forwards to the three-argument
  overload with `ClientBounds.Width`/`Height`.
- `cna_game_window_subscribe` mirrors `cna_game_subscribe` exactly — same shape,
  same release route — so the settled one-subscription-per-event bridge applies
  unchanged. Identities are `CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED` (0),
  `ORIENTATION_CHANGED` (1), `SCREEN_DEVICE_NAME_CHANGED` (2).

**The open sub-question, and the only one:** `GameWindow::SetSupportedOrientations`
is `famorassem` (protected-internal) and abstract, and **CNA has no window
counterpart**. The only CNA orientation setter is
`cna_graphics_device_manager_set_supported_orientations`, which takes a MANAGER
handle rather than a game handle, and is a different object. In XNA the caller is
`GraphicsDeviceManager`, whose CNA-Go projection stores the value as managed
state. Deciding what this member does is what stands between a PARTIAL and a
COMPLETE `GameWindow`, and creating a sixth partial type would cut against the
repository's own selection rule.

### 2. `Game.Tick` and `Game.RunOneFrame` — blocked on a lifecycle decision

CNA exports `cna_game_tick` and `cna_game_run_one_frame`, so the binding is
trivial. What is not settled is WHEN a native game exists.

Today it exists only inside `Run`, which blocks; and `cna_game_tick` is
documented as **refused from inside a lifecycle callback**. So there is no
reachable moment at which either member could succeed, and projecting one that
can only ever report failure would be worse than leaving it missing.

Making them work needs a non-blocking native game lifecycle, and at least three
materially different shapes exist: eager creation in `NewGame`, lazy creation on
the first `Tick`, or an explicit non-blocking start projecting XNA's
assembly-visible `StartGameLoop`. That is a genuine architecture fork.

### 3. `DrawableGameComponent` — the inheritance is done, the device is not

`GameComponent` is `COMPOSED` and the inherited surface is enumerated and
attributed. What blocks the derived type is not inheritance:

```csharp
public override void Initialize()
{
    base.Initialize();
    if (this.initialized) return;
    this.deviceService = this.Game.Services.GetService(typeof(IGraphicsDeviceService))
                         as IGraphicsDeviceService;
    if (this.deviceService == null)
        throw new InvalidOperationException(Resources.MissingGraphicsDeviceService);
    ... subscribe four device handlers ...
    if (this.deviceService.GraphicsDevice != null) this.LoadContent();
    this.initialized = true;
}
```

`Initialize` is declared on `DrawableGameComponent`, which lives in the framework
package, and its body must name `Graphics.IGraphicsDeviceService` — a contract in
a package that imports the framework one. The cross-package cycle rule projects
device-typed MEMBERS into the descendant package, which a private field
resolution inside a framework-declared method body cannot use.

A consumer genuinely can register that service now, so the branch is
**observable** and must not be assumed absent. Two designs would resolve it and
neither is selected by precedent:

1. a resolver the Graphics package installs into the framework package at `init`
   time (the framework declares the four device-event accessors as a private
   structural interface — all their signatures are framework-typed, so only
   `GraphicsDevice()` is unnameable), or
2. a private scan of the service container's own registration list for a
   `reflect.Type` identifying the contract, plus that same structural interface.

The exact `Resources` string the throw site loads is
`"Drawable components require a graphics device service in the game service container."`

`GamerServicesComponent` is blocked harder: `Game.Window.Handle` is a missing
member of a missing type, and `GamerServicesDispatcher` lives in
`Microsoft.Xna.Framework.GamerServices.dll`, which is not one of the seven pinned
assemblies.

### 4. `Game`'s last eight members

`Content` ×2 and `LaunchParameters` need missing types — and `LaunchParameters`
extends `Dictionary<string,string>`, so it is really a BCL decision, with CNA's
whole `cna_game_launch_parameters_*` family waiting behind it. `Window` needs
`GameWindow`. `IsActive` reads `GamerServicesDispatcher.IsInitialized` and
`Guide.IsVisible` from an assembly outside the pinned seven and cannot be
projected. `ShowMissingRequirementMessage` takes a `System.Exception` and waits
on that frontier. `Tick` and `RunOneFrame` are item 2.

### 5. The base-typed substitutability frontier

No family is LIVE, and the day one becomes live is named precisely:
`Texture2D` has 17 positions, nine on `SpriteBatch`, which CNA-Go projects.
**Projecting `RenderTarget2D` while `SpriteBatch.Draw` takes a `Texture2D` makes
it live**, and `TestNoBaseFamilyHasALiveSubstitutabilityRequirementYet` says so
in those words. At that point the narrowest reusable public reference
abstraction has to be chosen, and until then it must not be.

## Verifier rules added this session

### The frame-hook override registry (Foundation 38)

Two installation classes, `NEVER` and `ON_OVERRIDE`, and deliberately no
`ALWAYS`: an unconditional hook is the automatic base behaviour Foundation 31
refused. Each `ON_OVERRIDE` hook's capability is measured against **compiler
evidence** — exists, unexported, an interface, exactly one method, exported
method, exact signature, not shared, no exported twin — and no exported member or
type in the framework package may be named `…Override`, which refuses every
registration API by shape rather than by a list.

### The authoritative raise path (Foundation 39)

Replaces the Foundation 36 rule. Two paired vocabularies, `RaisePath` and
`NativeSignalRole`, plus a required `NativeSignalMoment` so a `LIFECYCLE_ONLY`
role states WHY the semantics do not align.

### The substitutability inventory (Foundation 40)

Mechanical, from the pinned contract, over every public position including behind
arrays, by-reference markers and generic arguments. The relationships come from
the contract's own `baseType` fields and are cross-checked against the registry,
so neither can go stale alone.

### The composition rule (Foundation 41)

Private named composition, never embedding, no base accessor, and every inherited
member attributed to exactly one provenance class.

34 new negative controls across the four, taking the mutation inventory from 531
to 565, plus six new native ABI mutation controls.

```text
FOUNDATION_MILESTONE_38_COMPLETE=true
FOUNDATION_MILESTONE_39_COMPLETE=true
FOUNDATION_MILESTONE_40_COMPLETE=true
FOUNDATION_MILESTONE_41_COMPLETE=true
FOUNDATION_MILESTONE_42_COMPLETE=true
FOUNDATION_MILESTONE_43_COMPLETE=true
PUSHED_AT_COMMIT_TIME=false
PUSH_STATE=RESOLVE_WITH_GIT_STATUS_SHORT_BRANCH
GAME_MANAGED_SLICE=EXHAUSTED
NEXT_STEP=GAMEWINDOW_ONE_OPEN_SUBQUESTION_SETSUPPORTEDORIENTATIONS
NEXT_STEP_ALTERNATIVE_1=DRAWABLEGAMECOMPONENT_DEVICE_SERVICE_RESOLUTION_TWO_DESIGNS
NEXT_STEP_ALTERNATIVE_2=GAME_TICK_RUNONEFRAME_NON_BLOCKING_LIFECYCLE_THREE_SHAPES
NEXT_STEP_ALTERNATIVE_3=BCL_DICTIONARY_FRONTIER_UNBLOCKS_LAUNCHPARAMETERS
