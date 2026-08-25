# Foundation 21 — the pure managed service registry

Foundation 21 closes `Microsoft.Xna.Framework.GameServiceContainer`, the last
dependency-complete type on the frontier that neither needs fabricated hardware
nor a runtime CNA-Go does not have.

## Reference authority

```text
Microsoft.Xna.Framework.Game.dll
sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
```

IL read with `ikdasm`; the four exception message strings read from the
`Microsoft.Xna.Framework.Resources.resources` stream of the same assembly.
Public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## Why it was reachable

Its only unsatisfied dependency was the declared direct interface
`System.IServiceProvider`. That interface's single member, `GetService(Type)`,
is **already an ordinary public member of the class**, so the interface adds no
projected surface.

This is the same shape as `TouchCollection`'s `IList<TouchLocation>` in
Foundation 20, and it is now stated as a general rule: a BCL interface whose
members the XNA type already declares publicly needs no separate Go interface,
because the concrete method set is the whole projection.

## Exact contract

CLR: `public class GameServiceContainer : System.Object, System.IServiceProvider`,
4 source members, 4 projected Go identities, 3 with an error result.

```go
func NewGameServiceContainer() *GameServiceContainer
func (c *GameServiceContainer) AddService(serviceType reflect.Type, provider any) error
func (c *GameServiceContainer) RemoveService(serviceType reflect.Type) error
func (c *GameServiceContainer) GetService(serviceType reflect.Type) (any, error)
```

`System.Type` projects to `reflect.Type` and `System.Object` to `any`, both
from the existing declared BCL table.

Storage is one private `Dictionary<Type, object>`, created by the constructor.
There is no ordering, no lifetime management, no disposal, and no runtime of its
own: nothing here creates a device, starts a game, or reaches native code.

### Constructor

Its whole body allocates the backing dictionary. It validates nothing and is
infallible — the only member of the type that is.

### `AddService` — four checks, and the order matters

```text
IL_0002: Type::op_Equality(type, null)  -> ArgumentNullException("type", ServiceTypeCannotBeNull)
IL_001a: brtrue.s provider              -> ArgumentNullException("provider", ServiceProviderCannotBeNull)
IL_0033: services.ContainsKey(type)     -> ArgumentException(ServiceAlreadyPresent, "type")
IL_0051: type.IsAssignableFrom(provider.GetType()) is false
                                        -> ArgumentException(formatted ServiceMustBeAssignable)
IL_0099: services.Add(type, provider)
```

The **duplicate check runs before the assignability check**. Registering an
unassignable provider for an already-registered type therefore reports the
duplicate, not the mismatch. That ordering is asserted rather than assumed.

The exact `Resources` strings the throw sites load are:

| resource                      | text                                                                        |
| ----------------------------- | ---------------------------------------------------------------------------- |
| `ServiceTypeCannotBeNull`     | `The service type cannot be null.`                                           |
| `ServiceProviderCannotBeNull` | `The service provider instance cannot be null.`                              |
| `ServiceAlreadyPresent`       | `Container already contains a service of this type.`                         |
| `ServiceMustBeAssignable`     | `Service provider object of type {0} must be assignable to service type {1}.` |

A reference quirk worth recording, though the Go projection does not depend on
it: at `IL_007a` the second format argument is `type.GetType().FullName`, not
`type.FullName`, so XNA's own message names the runtime type of the `Type`
object — `System.RuntimeType` — rather than the requested service type. The Go
error text is a language projection, so it names the service type, which is what
the message was clearly meant to say.

Assignability is the CLR's `type.IsAssignableFrom(provider.GetType())`, which
`reflect.Type.AssignableTo` reproduces for both a concrete key and an interface
key. Registering one provider under both its concrete type and an interface it
implements is two registrations, not a duplicate, because the dictionary is
keyed by service type.

The nil-provider check is CLR's `brtrue` on an object reference. Go's interface
nil is the faithful analogue for the values Go can express; a non-nil interface
holding a typed nil pointer has no CLR counterpart and is registered rather than
rejected. That boundary is documented rather than papered over.

### `RemoveService` is forgiving

```text
IL_0002: null check -> ArgumentNullException("type", ServiceTypeCannotBeNull)
IL_0020: services.Remove(type)
IL_0025: pop                       <- the Boolean result is discarded
```

**Removing a service that was never registered is not an error.** Only the nil
service type is rejected.

### `GetService` — a miss is an absence

```text
IL_0002: null check  -> ArgumentNullException("type", ServiceTypeCannotBeNull)
IL_0025: ContainsKey -> return services[type]
IL_0034: ldnull; ret <- otherwise null
```

An unregistered type yields `nil` **with no error**. A missing service is an
absence, not a failure, and the projection keeps that distinction: only a nil
service type produces an error.

## What completing it does not do

It wires nothing up. CNA-Go's `Game` remains a partial native-backed facade and
does not expose a `Services` property, so nothing in the binding populates or
consults a container. Completing this type adds no game component system, no
service discovery, and no lifecycle.

## Verifier coverage

`GameServiceContainer` joins the shared pure-managed type closure:

```text
GameServiceContainer  class  value=false  4 -> 4 identities, 3 errors,
                             reference projection *GameServiceContainer  PASS
```

The 14-defect matrix applies with three shape skips — the type has no property
accessors at all, so `wrong_setter_parameter`, `artificial_getter_error`, and
`artificial_setter_error` have nothing to attack. Across the six-type cluster:
**72 applied, 12 skipped**, accounting for all 84.

The mutation inventory grows from 336 to 347.

## Behavior corpus

Six new observations in group `GAME_SERVICE_CONTAINER`, taking the corpus from
558 to 564 with zero failures: a miss that is not a failure, an absent removal
that is not a failure, all four nil-argument rejections, add-and-resolve, the
three rejection paths including the duplicate-before-assignability ordering, and
reference-semantics aliasing.

## Structural effect

```text                       before   after
TARGET_TYPES                    112     113
TARGET_MEMBERS                 1699    1703
TOTAL_DIAGNOSTICS               322     321
MISSING_TYPE                    145     144
MISSING_MEMBER                  177     177
COMPLETE_TYPES                  107     108
PARTIAL_TYPES                     5       5
```

Every mismatch, leak, allowlist, and unmeasured counter stays at zero.

## No ABI change

Nothing here reaches native code. The CNA-Go ABI is unchanged at
`23 / 67 / 96 / 28 / 2 / 5` with no missing symbols and no mismatches. CNA was
not rebuilt.
