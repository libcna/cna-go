# Foundation 46 — DrawableGameComponent, and the cross-package resolution it needed

`DrawableGameComponent` is complete: 18 declared members plus 12 inherited from
`GameComponent`, 30 Go identities, no partial. It is the profile's **first
projected XNA-to-XNA derived type** — `XNA_COMPOSED_DERIVED_TYPES_PROJECTED`
moves from 0 to 1 — and its one shipped `IDrawable`.

## The blocker was never the inheritance

Foundation 41 settled XNA-to-XNA inheritance: private named composition, explicit
measured forwarding, never Go embedding, no `Base`/`Parent`/`As…` accessor. That
machinery applied here unchanged and is not what stood in the way.

What stood in the way was one statement:

```csharp
this.deviceService = this.Game.Services.GetService(typeof(IGraphicsDeviceService))
                     as IGraphicsDeviceService;
```

`Initialize` is declared on a type in `Microsoft.Xna.Framework`. The Graphics
package imports the framework package, so the framework package cannot name
`IGraphicsDeviceService`, cannot build the `reflect.Type` token `GetService` is
keyed by, and cannot spell the return type of the service's device accessor.

The settled cross-package cycle rule covers the **member** side of this — a
device-typed member projects into the descendant package, which is where
`Game.GraphicsDevice` already lives. A private field resolution inside a
framework-declared method body is not a member, and the rule does not reach it.

## The design, and why it is the narrowest one

`internal/servicebridge` holds **two function values**, each installed once from a
package `init`:

```text
Graphics  init  -> SetDeviceServiceResolver(...)   framework asks, Graphics answers
framework init  -> SetComponentServiceReader(...)  Graphics asks, framework answers
```

The resolver is the only place that can build the token and compare a device
with nil; the reader is the only place that can read the component's private
service field, which the Graphics package needs to project
`get_GraphicsDevice`.

It adds **no public API** — both halves are internal, so `UNEXPECTED_TYPE` and
`UNEXPECTED_MEMBER` stay at zero. It retains **no object**: each closure receives
what it is asked about and returns immediately, so nothing here can keep a Game,
a component, a device or a service alive. And it needs **no reflection at the
call site** and no string matching on type names.

The framework side holds the service through an unexported structural interface
with eight exported method names — exactly `IGraphicsDeviceService`'s event
accessors, whose signatures are framework-typed on both sides. So a consumer's
own service satisfies it **with no adapter at all**. The ninth member,
`GraphicsDevice()`, is deliberately absent from that interface: the one place
`Initialize` needs it is a nil comparison, and the resolver supplies that as a
closure.

### "No resolver installed" is correct rather than a gap

The framework package does not import the Graphics package, so a consumer who
never imports Graphics never runs its `init` and no resolver exists. That is
exactly the right answer, not a degraded one: **to register an
`IGraphicsDeviceService` a consumer must be able to name it, which requires
importing the Graphics package.** A program that cannot have registered the
service resolves nothing and gets the reference's own
`MissingGraphicsDeviceService`. The two cases are indistinguishable because they
should be, and a test asserts it.

The rejected alternative was scanning the service container for the first
registration that structurally matches. It resolves the same case and is wrong
for a different one: XNA looks up by an exact type token, and a container holding
two structurally-matching registrations, or one unrelated object with those
method names, would resolve differently from the reference.

## The exception message the resource key does not describe

`get_GraphicsDevice` throws under the resource key
`PropertyCannotBeCalledBeforeInitialize`. The obvious reading of that key —
"This property cannot be called before Initialize()" — is **wrong**. Read out of
the `Microsoft.Xna.Framework.Resources.resources` stream of the retained
`Microsoft.Xna.Framework.Game.dll`, the string is:

```text
The GraphicsDevice property cannot be used before Initialize has been called.
```

It names the property and says "used", not "called". This is the second measured
case in the profile where the resource **value** is authority and the **key**
describes it badly, after `Resources::InactiveSleepTimeCannotBeZero`, whose
string admits zero.

The other string this type throws was already measured and is confirmed
unchanged: `MissingGraphicsDeviceService` is `"Drawable components require a
graphics device service in the game service container."` — and it is a different
string from `NoGraphicsDeviceService`, which `Game.GraphicsDevice` throws:
`"This property requires a graphics device service in the game service
container."` Three device-service messages, three throw sites, none
interchangeable.

## Four details the IL makes and a natural rewrite loses

**`initialized = true` is outside the guard.** The `brtrue` at the top of
`Initialize` jumps **to** the store, not past it, so a second call re-assigns the
flag and skips only the resolution body. Writing `if (initialized) return;` is
one statement shorter and changes the throw path.

**A failing `Initialize` leaves `initialized` false.** The store is after the
throw site, so a component whose service is missing retries the resolution on a
later `Initialize` — which is exactly what a consumer who registers the service
late needs.

**The four subscriptions happen before the device check.** A component
initialized while the service publishes no device is still wired for
`DeviceCreated` and will be told when one appears. Reordering them would silently
break late device creation.

**Two of the four handlers are empty and are subscribed anyway.**
`DeviceCreated` calls `LoadContent`, `DeviceDisposing` calls `UnloadContent`, and
`DeviceResetting` and `DeviceReset` are each a bare `ret`. The reference
subscribes all four; a projection that skipped the empty ones would install two
registrations where the reference installs four, and a service that counts its
subscribers would see the difference.

## Two guards, two owners

```csharp
// DrawableGameComponent::Dispose(bool)
if (disposing) { this.UnloadContent(); if (deviceService != null) { remove ×4 } }
base.Dispose(disposing);        // OUTSIDE the guard

// GameComponent::Dispose(bool)
if (disposing) { Game.Components.Remove(this); Disposed(this, EventArgs.Empty); }
```

The derived class guards **only its own work** and hands the flag on
unconditionally; the base then guards its **entire** body on the same flag. So
`Dispose(false)` does nothing observable — and that is the base's decision rather
than the derived class's. A `if (!disposing) return;` at the top of the derived
method would produce the same observable result today and would be wrong the
moment the base body changed.

Neither is idempotent. There is no disposed flag in either class, so a second
`Dispose(true)` removes and raises again, and a test measures exactly two raises
for two calls.

## Virtual calls Go cannot make

`Initialize` ends with `if (deviceService.GraphicsDevice != null)
this.LoadContent();`, and `DeviceCreated`/`DeviceDisposing` call
`LoadContent`/`UnloadContent`. All three are `callvirt`, so in the CLR a
subclass override runs.

Go has no virtual dispatch, and the settled rule since Foundation 31 is that base
behaviour is explicit and never automatic. Those calls therefore reach the bodies
projected here, which are the reference's own: `LoadContent`, `UnloadContent` and
`Draw` are each a bare `ret` of code size 1. A consumer's own `LoadContent` is
not reached, exactly as a consumer's own `Update` is not reached by base
`Update`.

This is recorded as a `GO_LANGUAGE_LIMITATION`, not worked around. The
Foundation 38 frame-hook mechanism does not apply: it discovers overrides on the
object a consumer hands to `NewGame`, and `DrawableGameComponent`'s constructor
takes a `Game` and nothing else — adding a parameter would change a signature the
contract fixes, and adding an opt-in method would be public API the contract does
not declare.

## Fallibility, member by member

`DrawableGameComponent` is classified **pure managed**, which is worth stating
because the type is the profile's bridge to the graphics runtime and looks
native-backed from outside. It is not: three scalar fields, two delegate fields,
a lookup in a managed dictionary, four `Delegate.Combine` subscriptions, and four
bodies that are a single `ret`. The one member that reaches a device does not
touch one — it null-checks a field and forwards to the **service's** property.

Nine members are fallible and each names its own evidence: `Initialize` and
`Dispose` (both reach consumer code and can throw), `OnVisibleChanged` and
`OnDrawOrderChanged` (invoke consumer handlers), `set_Visible` and
`set_DrawOrder` (announce through those raisers), `get_GraphicsDevice` (throws),
and the two **inherited** setters `set_Enabled` and `set_UpdateOrder`.

The last two are the subtle ones. Fallibility is classified against the owner a
member is **projected on**, so a derived type has to name the inherited members
that fail as well as its own; leaving them out would make
`GameComponent::set_Enabled` infallible on `DrawableGameComponent` and fallible
on `GameComponent`, for the same body.

Six members are deliberately absent from that list: the constructor stores two
fields and calls the base, `Draw`/`LoadContent`/`UnloadContent` are bare `ret`s,
and `get_Visible`/`get_DrawOrder` are one `ldfld` each. `IDrawable` already
required `Draw` to be infallible, on this exact evidence, before the type
existed.

## Scoreboard

```text                                      before    after
TARGET_TYPES                                 123      124
TARGET_MEMBERS                              1831     1860
COMPLETE_TYPES                               118      119
PARTIAL_TYPES                                  5        5
MISSING_TYPE                                 134      133
MISSING_MEMBER                               145      145
TOTAL_DIAGNOSTICS                            279      278

XNA_COMPOSED_DERIVED_TYPES                     2        2
XNA_COMPOSED_DERIVED_TYPES_PROJECTED           0        1
XNA_INHERITED_MEMBER_PROJECTIONS              24       24

behavior corpus                              655      661
external canary tests                         69       72
runtime capability rows                       45       46
```

Two counters deliberately do not move, and both are correct.

`MISSING_MEMBER` stays at 145 because `DrawableGameComponent` was a missing
TYPE: none of its members was previously counted as a missing member, and the
whole type arrives complete.

`XNA_INHERITED_MEMBER_PROJECTIONS` stays at 24 because it counts what the
inheritance projection EXPECTS -- twelve members across both derived types --
not what is implemented. The counter that moves for implementation is
`XNA_COMPOSED_DERIVED_TYPES_PROJECTED`, and it is the one that was zero.

## What this milestone does not claim

- **No native code is reached.** Nothing here binds a CNA route; `BOUND_FUNCTIONS`
  is unchanged at 41.
- **A consumer's `LoadContent` override is not called by `Initialize`.** See the
  virtual-call section; this is a Go language limitation and is recorded as one.
- **`GamerServicesComponent`, the other derived type, is still missing.** Its
  blocker is no longer `Game.Window.Handle` — Foundation 45 removed that one — but
  `GamerServicesDispatcher` lives in `Microsoft.Xna.Framework.GamerServices.dll`,
  which is not one of the seven pinned assemblies and has no CNA runtime behind
  it.
- **`GraphicsDeviceManager` still publishes no `IGraphicsDeviceService`.** A
  device therefore comes from a consumer's own registration or not at all, which
  is the same state `Game.GraphicsDevice` has been in since Foundation 43.
