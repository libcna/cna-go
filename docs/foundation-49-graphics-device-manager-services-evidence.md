# Foundation 49 — the device service GraphicsDeviceManager publishes

`GraphicsDeviceManager` gains its five events, its four protected raisers, the
two service registrations its constructor makes, and the three
`IGraphicsDeviceManager` operations the reference implements privately. Fourteen
missing members close, `TOTAL_DIAGNOSTICS` drops from 256 to 242, and the
sentence that has ended the last three milestones' "what this does not claim"
sections is retired:

> **`GraphicsDeviceManager` still publishes no `IGraphicsDeviceService`.**

It publishes one now. `Game.GraphicsDevice` and `DrawableGameComponent` resolve
a device from a Game that registered nothing of its own, which is what a real
XNA program expects and what CNA-Go could not do since Foundation 43.

## The constructor makes two registrations, not one

```csharp
if (game.Services.GetService(typeof(IGraphicsDeviceManager)) != null)
    throw new ArgumentException(Resources.GraphicsDeviceManagerAlreadyPresent);
game.Services.AddService(typeof(IGraphicsDeviceManager), this);
game.Services.AddService(typeof(IGraphicsDeviceService), this);
```

One object, two contracts, two service tokens. `IGraphicsDeviceManager` lives in
`Microsoft.Xna.Framework` and the manager satisfies it directly.
`IGraphicsDeviceService` lives in `Microsoft.Xna.Framework.Graphics`, and its
ninth member is `GraphicsDevice GraphicsDevice { get; }` — a **Graphics-package
type returned from a framework-package object**, which is the cross-package
cycle in its most direct form.

So the second registration is an adapter: `managerDeviceService`, declared in the
Graphics package, holding the manager and forwarding all four event pairs to it,
with `GraphicsDevice()` calling the same `GraphicsDeviceManagerGraphicsDevice`
projection `Game.GraphicsDevice` uses. It is published and unpublished through
`internal/servicebridge`, the mechanism Foundation 46 built, so the framework
package never names the type it is registering.

**One difference from the reference is recorded rather than papered over.** XNA
registers `this` under both tokens, so `GetService(typeof(IGraphicsDeviceService))
== GetService(typeof(IGraphicsDeviceManager))` is true. In CNA-Go the second is
an adapter over the first, so that reference equality is false. Everything
observable *through* the interface is identical, the adapter is created **once
per manager** so its own identity is stable, and the device both tokens lead to
is the same object. A consumer comparing the two service registrations for
reference equality is the one program that can tell.

## The duplicate check has to run first

The check was written after native creation, and that was wrong in a way only a
live run shows: CNA's own manager refuses a second registration with **its own**
sentence,

```text
A GraphicsDeviceManager is already registered with this Game.
```

where the reference throws Microsoft's, which is a different sentence about a
different thing — and carries a double space after the first full stop, which
`tools/resource_strings` reads out of the assembly rather than trusting anyone to
retype:

```text
A graphics device manager is already registered.  The graphics device manager cannot be changed once it is set.
```

Creating first therefore reported CNA's message where the reference reports
Microsoft's, and left an orphaned native manager to unwind. The reference's IL
checks the service container as its **first** statement after the null check, and
the projection now does the same. Twenty stress cycles assert the exact
Microsoft string.

Every later failure in the constructor unwinds what the earlier steps did, in
reverse: a refused `IGraphicsDeviceManager` registration releases the native
manager; a refused device-service publication removes the first registration too;
a refused native subscription undoes both.

## Dispose removes only its own registrations

```csharp
if (this.game.Services.GetService(typeof(IGraphicsDeviceManager)) == this)
    this.game.Services.RemoveService(typeof(IGraphicsDeviceManager));
if (this.game.Services.GetService(typeof(IGraphicsDeviceService)) == this)
    this.game.Services.RemoveService(typeof(IGraphicsDeviceService));
```

Both removals are guarded by an identity comparison, so a manager that was
disposed **after** something else took over its token removes nothing. That
guard is preserved on both halves — the adapter's, through the bridge, compares
the adapter's manager with this one rather than the adapter with itself, because
the adapter is the object in the container and the manager is the object being
disposed.

## Three members that are witnesses, not public API

`IGraphicsDeviceManager` declares `CreateDevice`, `BeginDraw` and `EndDraw`, and
`GraphicsDeviceManager` implements all three as **private explicit interface
implementations** — three `.override` directives, `CompilerControlledAccess`,
absent from the type's public member set. In Go there is no explicit
implementation: a method is exported or it is not, and an unexported one cannot
satisfy an interface the Graphics package declares.

They are therefore projected as exported methods and registered as **interface
witnesses**, the category Foundation 40 created for exactly this. The registry
that admits them is explicit rather than general:

```go
var explicitInterfaceWitnessOwners = map[string]bool{
    "Microsoft.Xna.Framework.GraphicsDeviceManager|Microsoft.Xna.Framework.IGraphicsDeviceManager": true,
}
```

Widening the gate to "any exported method required by any mapped interface" was
tried first and produced **13 false admissions** — `Color.PackFromVector4` and the
manager's own event accessors among them — so the narrow registry is what
shipped, with a `deliberatelyUnregistered` entry recording why
`IGraphicsDeviceService` is not in it: the adapter, not the manager, is that
interface's implementation. `INTERFACE_WITNESS_PROJECTIONS` moves from 25 to 28.

## The third native signal family

The game family has four identities and the window family three. The manager
family has five, and its numbering is the reason each family gets its own
trampoline table rather than sharing one:

```text
CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DISPOSED        = 0
CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_CREATED  = 1
CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_DISPOSING= 2
CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESET    = 3
CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESETTING= 4
```

`DISPOSED` is zero here and the game family's zero is `ACTIVATED`. A single
shared table indexed by a raw identity would deliver a manager's disposal to a
game's activation handler and compile perfectly, so `bridge.c` carries a
`_Static_assert` that the two families' zeroes are not the same constant.

Each of the four device signals is routed to the **reference's own raise path** —
the protected `On…` method, which is where XNA's private `HandleDeviceLost`,
`HandleDeviceReset`, `HandleDeviceResetting` and `HandleDisposing` send them. The
disposal signal raises `Disposed` directly, because the reference declares no
protected raiser for it; that is the same LIFECYCLE_ONLY shape Foundation 39
settled for `Game.Disposed`.

Measured, 20 cycles: `DeviceCreated` is delivered and routed **20 times out of
20**. The reset, resetting and disposing counters are **zero**, and that is
reported rather than hidden — a HEADLESS device is created once and never lost,
so there is nothing to reset and nothing to lose.

## One device facade per manager, per generation

`GraphicsDeviceManager::get_GraphicsDevice` is a single `ldfld`. CNA-Go's
projection allocated a **fresh** `*GraphicsDevice` on every call, which passes
every test written with one local variable and fails the moment two callers
compare what they got — which is precisely what happens when `Game.GraphicsDevice`
and a `DrawableGameComponent`'s device are supposed to be the same device.

The facade is now cached on the manager, through `internal/servicebridge`,
together with the **native generation** it was built for. A new `Run` gets a new
generation and therefore a new facade, which is where the reference replaces its
own field rather than handing out an object bound to a dead device.

## A message that was inferred, and the tool that now forbids that

Foundation 48 wrote the back-buffer range message as

```text
The back buffer dimension must be positive.
```

That sentence was **inferred from the resource key** `BackBufferDimMustBePositive`
and it is not the string. Read out of the retained
`Microsoft.Xna.Framework.Game.dll`, it is

```text
BackBufferWidth and BackBufferHeight must be greater than zero.
```

It names both properties and says "greater than zero". This is the third measured
case in the profile where the key describes its value badly, after
`InactiveSleepTimeCannotBeZero` (whose string admits zero) and
`PropertyCannotBeCalledBeforeInitialize` (whose string says "used", not
"called") — and the first where CNA-Go got it wrong in shipped code.

Fixing the string is the small half. The large half is `tools/resource_strings`,
which makes this class of defect a test failure:

```text
RESOURCE_STRINGS_CLAIMED=15
RESOURCE_STRINGS_VERIFIED=15
RESOURCE_STRINGS_SOURCE_CONSTANTS=14
RESOURCE_STRINGS_ASSEMBLIES=10
RESOURCE_STRINGS_FINDINGS=0
RESOURCE_STRINGS_STATUS=PASS
```

It holds a registry of every Microsoft message CNA-Go claims, each with the
assembly it came from and its resource key, and requires the exact bytes to
appear in that retained assembly. It then walks the Go source with `go/ast`,
collects every message-shaped string constant under `Microsoft/`, and requires
each one to be in the registry — so a **new** invented message fails as loudly as
a wrong one. One registry entry restores the `{0}`/`{1}` placeholders a Go format
string spells differently; it is the only spelling transform, and it is named.

## A bound route nothing called

`cna_graphics_device_manager_dispose` was bound in Foundation 48. Its single call
site was removed during this milestone, and the typedef, the X-macro entry, the
trampoline, the `_Static_assert` and `interop.ManagerDispose` all stayed. Nothing
failed. `BOUND_FUNCTIONS` reported a boundary one route wider than the one CNA-Go
exercises.

`tools/native_abi` now measures the whole chain, following the source rather than
a naming convention:

```text
route         cna_foo                the abi_manifest.h X-macro entry
trampoline    the C function whose body calls api.cna_foo(...)
cgo call      the Go function whose body calls C.<that trampoline>(...)
reachability  that Go function must be reachable, inside package interop,
              from something the rest of the module actually names
```

The roots of that reachability are the three ways the module can reach into the
package: an `interop.X` selector written outside it, an exported method, and
`init`. The check is proved falsifiable by the defect that motivated it — with
the route still bound it named `cna_graphics_device_manager_dispose` and nothing
else, and it passes with the route removed.

Its first version was a regexp over raw source, and its **own doc comment** —
which names `interop.ManagerDispose` as the defect being described — was enough
to make the dead route look reached. It reads the parsed syntax tree now. A
mention in a comment is not a call, and a check that its own prose can satisfy is
not a check.

### Why the route is not called

Two reasons, and the first is sufficient. The event that call exists to raise is
raised from `Dispose(bool)`, at the reference's own raise site, so calling it
would produce a second `Disposed` nobody asked for.

The second is an observation, not an accusation: it was seen to **SIGSEGV** once,
when a stress game disposed its manager from `UnloadContent` during run teardown.
A standalone C reproduction was attempted in four shapes — from `unload_content`,
from `exiting`, after `run` returned, and with the manager still alive across
`cna_game_destroy` — and **none of them crashed**. The trigger is not understood,
so CNA is not accused of a defect.

## A counter that tried to become public API

`tools/native_stress` needs the per-identity signal delivery counts, and the first
attempt exposed them as `GraphicsDeviceManager.NativeSignalDeliveriesForTests`.
The verifier reported it immediately:

```text
UNEXPECTED_MEMBER  GraphicsDeviceManager.NativeSignalDeliveriesForTests
                   exported member does not map to the selected XNA profile
                   or a declared language adapter
```

It was right. A test-only counter is not in the XNA contract, and the invariant
that no such member exists is worth more than the convenience. The counter moved
to `internal/servicebridge`, which the stress tool can import and a consumer
cannot, and `UNEXPECTED_MEMBER` is back to **0**.

## Evidence

The graphics-manager stress scenario, still 20 isolated cycles:

```text
GRAPHICS_MANAGER_SERVICE_REGISTRATION_CHECKS       40   both tokens, every cycle
GRAPHICS_MANAGER_DUPLICATE_REGISTRATION_CHECKS     20   Microsoft's exact message
GRAPHICS_MANAGER_GAME_GRAPHICS_DEVICE_CHECKS       20
GRAPHICS_MANAGER_DRAWABLE_COMPONENT_CHECKS         20
GRAPHICS_MANAGER_EVENT_RAISE_CHECKS                80   four raisers per cycle
GRAPHICS_MANAGER_SERVICE_REMOVAL_CHECKS            20
GRAPHICS_MANAGER_SIGNAL_DEVICE_CREATED_DELIVERIES  20
GRAPHICS_MANAGER_SIGNAL_DEVICE_RESET_DELIVERIES     0
GRAPHICS_MANAGER_SIGNAL_DEVICE_RESETTING_DELIVERIES 0
GRAPHICS_MANAGER_SIGNAL_DEVICE_DISPOSING_DELIVERIES 0
GRAPHICS_MANAGER_SIGNAL_DISPOSED_DELIVERIES         0
```

Every counter from the other ten scenarios reproduces unchanged.

`GRAPHICS_MANAGER_DRAWABLE_COMPONENT_CHECKS` is the milestone in one number: a
`DrawableGameComponent` whose `Initialize` resolves a device from a Game the test
registered **nothing** on. Before this milestone that call reported
`MissingGraphicsDeviceService` and was correct to.

The stress tool's counter accumulation was also rewritten. It was a hand-written
field list, and the window scenario's counters had been silently reporting zeros
because the list was never extended; it is a reflection loop now that panics if
any counter is not an `int`, so a new counter cannot be silently dropped again.

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1882     1896
MISSING_MEMBER                               123      109
TOTAL_DIAGNOSTICS                            256      242
COMPLETE_TYPES                               119      119
PARTIAL_TYPES                                  5        5
UNEXPECTED_MEMBER                              0        0
INTERFACE_WITNESS_PROJECTIONS                 25       28

behavior corpus                              665      667
external canary tests                         74       75
native stress scenarios                       11       11
native ABI mutation controls                  68       75
runtime capability rows                       48       49

BOUND_FUNCTIONS                               53       57
PROTOTYPE_TYPE_POSITIONS                     161      174
MANIFEST_SIDE_ASSERTIONS                      21       28
MANIFEST_LAYOUT_AGREEMENTS                   117      117
ABI_MISMATCHES / FINDINGS                      0        0
```

`GraphicsDeviceManager` remains the profile's one partial type of interest, and
its six remaining members are still the single blocker Foundation 48 named:
`FindBestDevice`, `CanResetDevice` and `RankDevices` need
`GraphicsDeviceInformation`, which needs `GraphicsAdapter`;
`PreparingDeviceSettings` and `OnPreparingDeviceSettings` need
`PreparingDeviceSettingsEventArgs`, which wraps one.

## What this milestone does not claim

- **The two service tokens do not lead to the same object.** They lead to the
  same *device* and to the same observable behaviour; the `IGraphicsDeviceService`
  registration is a stable per-manager adapter rather than the manager itself,
  because no framework-package type can declare that interface's device accessor.
- **No device reset, resetting or disposing signal was observed.** Three of the
  five identities are bound, delivered-capable and counted at zero. A HEADLESS
  device is created once and never lost, so the evidence for those three is that
  the wiring exists, not that it fired.
- **The manager's constructor still needs a live native game**, which the
  reference's does not. Unchanged from Foundation 48 and still recorded.
- **The three `GameWindow` handlers the reference's constructor installs are
  still deliberately absent.** All three run private back-buffer resize work that
  CNA's own manager already does from its own subscriptions; adding them would run
  it twice.
- **`cna_graphics_device_manager_dispose` is not bound.** The single SIGSEGV that
  motivated removing its call site is recorded as an unreproduced observation, not
  as an upstream defect.
