# Foundation 48 — GraphicsDeviceManager's configuration surface

`GraphicsDeviceManager` goes from 40 missing members to 20. Twenty members
arrive: nine configuration properties (18 accessors), two static read-only
fields, `ApplyChanges` and `ToggleFullScreen`. `TOTAL_DIAGNOSTICS` drops from
276 to 256 — the largest single-milestone drop in the project so far.

## Getters read managed state, setters reach CNA

Every one of the reference's nine getters is a single `ldfld`. Every setter is a
store plus `isDeviceDirty = true`, and the value reaches the device **later**,
when `ChangeDevice` runs. So the split is the one the `Game` timing members
already settled, and it falls out of the difference rather than being chosen:

- the getter is a field read of this object's own managed state — no
  validation, no device, no throw site, and therefore no error result;
- the setter validates as the reference does, stores as the reference does, and
  then **pushes to CNA's manager** — because CNA's `ApplyChanges` reads CNA's
  copy, and a value that never reached it would be a setting that appears to
  work and does not.

The ten `cna_graphics_device_manager_get_*` routes are deliberately **not**
bound, for the reason they are never bound in this binding: a native getter
would be a second source of truth that could disagree with the field the setter
wrote.

### One member changed classification

`SupportedOrientations` has been projected since Foundation 13 with **both**
accessors infallible, and that was correct: the stored value reached nothing, so
there was nothing to refuse. It now pushes like its eight neighbours, so the
setter gained an error result and the getter did not.

Three pieces of evidence moved with it rather than being left stale:
`managedStoredMembers` went from `property|SupportedOrientations` to
`property-get|SupportedOrientations`; the Foundation 13 contract test now
requires the setter to be fallible; and the mutation control
`graphics_manager_orientation_setter_returns_error` was **inverted** — until
this milestone an error result was the defect, and now a setter that cannot
report a refused push is.

## Two default sizes, and they are different

```text
GameWindow::DefaultClientWidth   = 0x320 = 800
GameWindow::DefaultClientHeight  = 0x258 = 600
GraphicsDeviceManager::DefaultBackBufferWidth  = 0x320 = 800
GraphicsDeviceManager::DefaultBackBufferHeight = 0x1e0 = 480
```

Two pairs of constants in one assembly, agreeing on width and disagreeing on
height. The manager's pair is what a game's back buffer starts at, and it is
the pair CNA also declares, in `runtime_graphics_manager.h`. A behavior-corpus
row and a canary test both assert 800x480 by name so the two cannot be confused
later.

## The constructor's field initializers

```csharp
synchronizeWithVerticalRetrace = true;
depthStencilFormat             = DepthFormat.Depth24;      // 2, not the enum's zero
backBufferWidth                = DefaultBackBufferWidth;   // 800
backBufferHeight               = DefaultBackBufferHeight;  // 480
base..ctor();
if (game == null) throw new ArgumentNullException("game", Resources.GameCannotBeNull);
```

The initializers run **before** the base constructor and therefore before the
null check, so those four values are stored even on the path that throws. They
are separated into `newGraphicsDeviceManagerState` for exactly that reason — it
is the one part of the constructor that reaches nothing, and it is what the
managed tests measure without needing a native game.

`IsFullScreen` and `PreferMultiSampling` are **not** assigned, so they start
false; `PreferredBackBufferFormat` is not assigned either, so it starts at
`SurfaceFormat.Color`, which is zero.

## Three properties live in the Graphics package

`GraphicsProfile`, `PreferredBackBufferFormat` and
`PreferredDepthStencilFormat` are typed by Graphics-package enums, so the
settled cross-package cycle rule projects the **members** there as
`GraphicsDeviceManagerMember` functions.

Their **values** stay where the reference keeps them: on the manager, which is a
framework-package object that cannot name those enums. It holds them as the raw
`int32` the CLR enums are, and `internal/servicebridge` — the same bridge
Foundation 46 built for `DrawableGameComponent` — carries them across through
three named slots. The conversion happens in the Graphics package, where both
sides are nameable.

The canary asserts the other half of that rule from outside: those three are
**not** methods on the manager, and a consumer reaches them as package functions
taking one.

## Two members that are not stores

**`ApplyChanges`** is

```csharp
if (this.device != null && !this.isDeviceDirty) return;
this.ChangeDevice(false);
```

The guard is deliberately **not** re-implemented in Go over state this object
does not hold. CNA's manager is an XNA reimplementation carrying the same guard
over the same two facts, and every setter above pushes to it — so its
`isDeviceDirty` is raised exactly when this one is, and its device field is the
real one. Re-implementing the guard would mean maintaining a second copy of a
decision that has to agree.

**`ToggleFullScreen`** is

```csharp
this.IsFullScreen = !this.IsFullScreen;
this.ChangeDevice(false);
```

and it goes through the projected **setter**, exactly as the reference's
`call set_IsFullScreen` does, so the store, the dirty flag and the push all
happen. `cna_graphics_device_manager_toggle_full_screen` exists and is
deliberately **not** bound: it would flip CNA's own flag a second time, after
the setter had already pushed the flipped value. Twenty stress cycles assert
that two toggles return to the starting state.

## What the setters actually do, member by member

| member | validates | stores | also |
| -- | -- | -- | -- |
| `PreferredBackBufferWidth` | `> 0`, `bgt` on zero | width | clears `useResizedBackBuffer` |
| `PreferredBackBufferHeight` | `> 0` | height | clears `useResizedBackBuffer` |
| `IsFullScreen` | nothing | flag | — |
| `SynchronizeWithVerticalRetrace` | nothing | flag | — |
| `PreferMultiSampling` | nothing | `allowMultiSampling` | — |
| `SupportedOrientations` | nothing | flags | — |
| `GraphicsProfile` | nothing | profile | — |
| `PreferredBackBufferFormat` | nothing | format | — |
| `PreferredDepthStencilFormat` | nothing | format | — |

All nine raise `isDeviceDirty` unconditionally: **none of them has an
inequality guard**, so unlike `GameWindow.Title` or `GameComponent.Enabled`,
assigning the same value again still dirties the device. A test asserts that
for all six framework-typed setters.

The property whose getter and field disagree is preserved as it is:
`PreferMultiSampling` reads `allowMultiSampling`.

`useResizedBackBuffer` is cleared by the two dimension setters and by no other
— a resized back buffer is forgotten the moment a consumer states a preference —
and a test asserts that `SetIsFullScreen` leaves it alone.

The comparison is `bgt` on zero, so **zero is rejected and one is accepted**,
and both boundaries are asserted.

The exact `Resources` string both setters throw was **inferred from its key in
this milestone and corrected in the next one**. The key is
`BackBufferDimMustBePositive`; the string is `"BackBufferWidth and
BackBufferHeight must be greater than zero."` — it names the two properties
rather than "the dimension", and says "greater than zero" rather than
"positive". Foundation 49 read it out of the retained assembly and added
`tools/resource_strings` so a claimed message that is not in one is a test
failure rather than a plausible-looking sentence.

## Evidence

Eleven native stress scenarios now, 20 isolated cycles each:

```text
GRAPHICS_MANAGER_CYCLES                            20
GRAPHICS_MANAGER_DEFAULT_CHECKS                    20
GRAPHICS_MANAGER_SETTERS_APPLIED                  120   six per cycle
GRAPHICS_MANAGER_CROSS_PACKAGE_SETTERS_APPLIED     60   three per cycle
GRAPHICS_MANAGER_RANGE_CHECKS                      20
GRAPHICS_MANAGER_APPLY_CHANGES                     20
GRAPHICS_MANAGER_TOGGLE_FULL_SCREEN_CHECKS         20
GRAPHICS_MANAGER_WRONG_THREAD_CHECKS               20
```

Every one of the nine settings is asserted to have reached the **managed field**
and to have been accepted by **CNA's manager**; a refused push surfaces as a
failure rather than being swallowed. The three cross-package settings
round-trip through `internal/servicebridge` and back, which is what makes the
two halves one object rather than two.

The eight in-package tests measure the managed half on a manager with no native
one behind it — which is not a shortcut, because that is exactly the state the
reference is in between construction and the first `ApplyChanges`.

## ABI

```text                                        before      after
BOUND_FUNCTIONS                               43      53
PROTOTYPE_TYPE_POSITIONS                     132     161
native ABI mutation controls                  63      68
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         117     117
ABI_MISMATCHES / FINDINGS                      0       0
```

Nine of the ten new routes share the shape
`CNA_Result(CNA_Handle, <one value>)`, and two of them —
`set_preferred_back_buffer_width` and `set_preferred_back_buffer_height` — are
byte-for-byte the same prototype. A manifest that bound one where the other
belongs compiles through every static check, and only the loader's `dladdr`
identity check separates them.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1862     1882
MISSING_MEMBER                               143      123
TOTAL_DIAGNOSTICS                            276      256
COMPLETE_TYPES                               119      119
PARTIAL_TYPES                                  5        5

behavior corpus                              663      665
external canary tests                         73       74
native stress scenarios                       10       11
native ABI mutation controls                  63       68
runtime capability rows                       47       48
```

`GraphicsDeviceManager` stays partial, and the twenty members that remain are
one blocker with three faces: `FindBestDevice`, `CanResetDevice` and
`RankDevices` take or return `GraphicsDeviceInformation`, whose `Adapter`
property needs `GraphicsAdapter`; `PreparingDeviceSettings` and
`OnPreparingDeviceSettings` need `PreparingDeviceSettingsEventArgs`, which wraps
a `GraphicsDeviceInformation`; and `RankDevices` additionally needs
`List<GraphicsDeviceInformation>`.

## What this milestone does not claim

- **The constructor still needs a live native game.** The reference's is pure
  managed and a consumer calls it from their own `Game` subclass's constructor;
  CNA-Go creates the native manager there, so it needs one. That is a
  pre-existing constraint rather than something this milestone introduced, and
  it is recorded rather than hidden.
- **`GraphicsDeviceManager` still publishes no `IGraphicsDeviceService`.** The
  reference's constructor registers itself under both that contract and
  `IGraphicsDeviceManager`, which is what makes `Game.GraphicsDevice` and
  `DrawableGameComponent.Initialize` work without a consumer-supplied service.
  Doing that from Go needs an adapter, because the framework package cannot
  implement an interface whose method returns a Graphics-package type, and that
  is the next milestone rather than this one.
- **Nothing here proves a device was reconfigured.** `ApplyChanges` and
  `ToggleFullScreen` are proved to reach CNA and to be accepted; the HEADLESS
  artifact has no window to resize and no screen to go full on.
- **`GraphicsProfile` starts at zero rather than at
  `ReadDefaultGraphicsProfile`'s answer.** That method probes the platform for
  HiDef support; CNA-Go asks CNA rather than guessing, and zero is
  `GraphicsProfile.Reach`, which is what a machine with no HiDef adapter would
  get anyway.
