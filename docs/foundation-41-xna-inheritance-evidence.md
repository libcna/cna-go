# Foundation 41 — the XNA-to-XNA inheritance projection

Foundation 33 named this the largest open architecture decision and left it
untouched through eight milestones. Foundation 40 measured the one fact it
depended on. This milestone makes the decision and projects it.

## The rule

An XNA class that inherits another XNA class projects the base as **private
named composition plus explicit measured forwarding**:

```go
type DrawableGameComponent struct {
    base *GameComponent          // private, named, not embedded
    visible   bool               // the derived type's own state
    drawOrder int32
}
```

and never as Go embedding:

```go
type DrawableGameComponent struct {
    *GameComponent               // REFUSED: BASE_MAPPING_MISMATCH
}
```

### Why not embedding

**Embedding is not inheritance.** It promotes the base's whole method set,
including members the derived class overrides or hides, and the promoted method
would silently win wherever the derived one was not redeclared with exactly the
right shape. `DrawableGameComponent` overrides `Initialize`; an embedded base
would keep `GameComponent.Initialize` reachable and would make the override a
property of Go's promotion rules rather than a measured fact.

Explicit forwarding makes every inherited member a declared member of the derived
type. The verifier then checks the override set instead of trusting it.

**Embedding also publishes the base.** An exported embedded field is public
surface Microsoft never declared and hands a consumer the base object to mutate
behind the derived type's back. There is no public `Base`, `Parent` or
`AsGameComponent` member either — the contract declares none, and a
signature-projection requirement is the only thing that could justify one.

### Why it is sufficient

Foundation 40 measured the actual CLR substitutability requirement of the whole
profile. `GameComponent`, `GraphicsResource` and `MathTypeConverter` — **25 of
the 41 derived types between them** — are named in **zero** public signature
positions, and no family is live.

For such a family private composition is not a compromise. There is no position
in the contract for a derived value to flow through, so no public reference
abstraction can be justified by anything Microsoft declared.

That measurement is what makes `GameComponent` the first `COMPOSED` relationship
rather than a guess, and a test asserts the justification and the measurement
agree: if `GameComponent`'s requirement ever stops being `NONE`, the suite says
so.

## The third provenance class

Every projected public member now belongs to exactly one of three classes, and
the partition is asserted to be disjoint and exhaustive:

| class | meaning | projections |
| -- | -- | --: |
| `XNA_DECLARED` | the CLR type declares the member itself | **3243** |
| `BCL_INHERITED` | inherited from a supported BCL base outside the contract | 12 |
| `XNA_INHERITED` | inherited from another XNA contract class, `COMPOSED` | **24** |

`REFERENCE_MEMBERS` stays 2964 — the third class touches nothing Microsoft
declares — and the XNA-declared projection count stays 3243, which is the number
that must never move. `EXPECTED_GO_MEMBERS` grows from 3255 to **3279**, which is
the surface the projection makes representable.

Three rules keep the classes from overlapping:

- **An overridden member is not also inherited.** A derived class that
  redeclares an inherited member is overriding it, so the projected member is the
  derived one. `DrawableGameComponent::Initialize` is `XNA_DECLARED`;
  `Update`, which it does not redeclare, is `XNA_INHERITED`.
- **Constructors are not inherited.** CLR requires a derived class to declare its
  own, and every derived class in the profile does.
- **Protected is not public surface.** A protected member is inherited in CLR but
  is not projected onto the derived type.

The walk is transitive up the `COMPOSED` chain and stops at the first base that
is not an XNA class in the profile, which is where `BCL_INHERITED` takes over.

## What is measured

```text
XNA_COMPOSED_BASE_RELATIONSHIPS          1
XNA_COMPOSED_DERIVED_TYPES               2
XNA_COMPOSED_DERIVED_TYPES_PROJECTED     0
XNA_INHERITED_PUBLIC_MEMBERS            14
XNA_INHERITED_MEMBER_PROJECTIONS        24
XNA_INHERITED_ATTRIBUTED_MEMBERS        24
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED   245 -> 227
```

The last line is the point. 245 was a lump sum: eleven deferred families'
inherited surfaces added together with nothing attributed. Eighteen of those
members are now *individually* attributed to an exact base and an exact CLR
member, with an exact projected Go identity each — which is what turns a
recorded gap into a projection.

Six negative controls attack the composition rule against a derived type that
otherwise satisfies it: embedding the base exported, embedding it unexported,
holding it in an exported field, holding no base at all, exposing a `Base`
accessor, and exposing an `AsGameComponent` upcast. One more attacks the registry
side.

## `COMPOSED` is about inheritance, not completeness

`COMPOSED` states that the **inheritance** is projected: the inherited public
surface is enumerated, attributed and required on the derived type. It does not
state that any derived type is complete, and neither of the two is.

`XNA_COMPOSED_DERIVED_TYPES_PROJECTED = 0` records that honestly.

### `DrawableGameComponent`'s remaining blocker is not inheritance

The pinned IL is exact:

```csharp
public override void Initialize()
{
    base.Initialize();
    if (this.initialized) return;
    this.deviceService = this.Game.Services.GetService(typeof(IGraphicsDeviceService))
                         as IGraphicsDeviceService;
    if (this.deviceService == null)
        throw new InvalidOperationException(Resources.MissingGraphicsDeviceService);
    this.deviceService.DeviceCreated   += this.DeviceCreated;
    this.deviceService.DeviceResetting += this.DeviceResetting;
    this.deviceService.DeviceReset     += this.DeviceReset;
    this.deviceService.DeviceDisposing += this.DeviceDisposing;
    if (this.deviceService.GraphicsDevice != null) this.LoadContent();
    this.initialized = true;
}
```

`Initialize` is declared on `DrawableGameComponent`, which lives in the framework
package, and its body must resolve
`Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService` — a contract in the
**Graphics** package, which imports the framework package. The settled
cross-package cycle rule projects device-typed *members* into the descendant
package (`graphics.OwnerTypeMember`), but a private field resolution inside a
framework-declared method body cannot use that.

This matters more than it used to, because a consumer genuinely can register such
a service now: `IGraphicsDeviceService` is projected, and `Services().AddService`
takes any `reflect.Type`. So the branch is **observable**, and the absence must
not be assumed away — which is exactly why `Initialize` is not projected rather
than projected as an unconditional failure.

Two materially different designs would resolve it, and neither is selected by
existing repository precedent:

1. a cross-package resolver the Graphics package installs into the framework
   package at `init` time, or
2. a private scan of the service container's registration list plus a structural
   interface for the four device-event accessors, whose signatures are all
   framework-typed.

That is a genuine fork and it is recorded rather than guessed. It blocks
`DrawableGameComponent`'s completeness; it does not block the inheritance
architecture, which is what this milestone delivers.

`GamerServicesComponent` is blocked for an unrelated and harder reason:
`Game.Window.Handle` is a missing member of a missing type, and
`GamerServicesDispatcher` lives in `Microsoft.Xna.Framework.GamerServices.dll`,
which is not one of the seven pinned assemblies and has no CNA runtime behind it.

## Scoreboard

```text                                    before   after
EXPECTED_GO_MEMBERS                       3255    3279
XNA_INHERITED_MEMBER_PROJECTIONS             0      24
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED   245     227
XNA_DEFERRED_BASE_BLOCKERS                  25      23
TOTAL_DIAGNOSTICS                          292     292
REFERENCE_MEMBERS                         2964    2964
```

No type became complete, no diagnostic moved, and nothing Microsoft declares
changed. What changed is that an inherited XNA surface is now a projection with a
provenance rather than a number in a deferral.
