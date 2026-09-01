# Foundation 55 — the inherited slot, and the CLR `this` a composed base lost

Two defects, one cause. Both were in the ONE composed XNA base relationship the
project already had, and both are in the shape every remaining graphics base
takes, so the graphics chain could not be composed over them.

`TOTAL_DIAGNOSTICS` stays at 208 and `MISSING_MEMBER` at 76. Nothing was
counted: one public member was added to `DrawableGameComponent` and one was
renamed, and the same two moved into the expected surface at the same time.
Milestones are not measured by the scoreboard moving.

## The exclusion rule was keyed on a name

`mapInheritedXNABaseMembers` subtracted the inherited members a derived class
"declares itself", keyed on `kind|name`, on this recorded claim:

> The exclusion is by CLR member name and kind, which is exactly what the CLR
> slot rules use for the members in this profile: no derived type in the
> contract overloads an inherited name.

The claim is false, and the counter-example is the composed family itself:

```text
GameComponent          public    Dispose()          newslot virtual FINAL
GameComponent          family    Dispose(bool)      newslot virtual
DrawableGameComponent  family    Dispose(bool)      virtual            -- override
```

`DrawableGameComponent` occupies the **protected** slot. The public `Dispose()`
is a different slot it never touches, so CLR inherits it — and a name-keyed
exclusion deleted it from the projected surface, silently, with no diagnostic.

The key is now the CLR slot identity: kind, name, generic arity and, for
methods, the parameter type list with `ref`/`out` marked. That is what the
runtime uses.

`XNA_INHERITED_PUBLIC_MEMBERS` moves 14 -> 15 and
`XNA_INHERITED_MEMBER_PROJECTIONS` 24 -> 25. The subtraction itself is now a
reported number, `XNA_INHERITED_PUBLIC_MEMBERS_OVERRIDDEN` = 3, so a rule that
drops a member it should have projected shows up as a count that moved.

### The overload namespace is shared

Once an inherited public member can share a name with a declared one, the two
are one overload group and the settled rule applies to the group:

```text
DrawableGameComponent.DisposeByBoolean(bool) error   -- declared
DrawableGameComponent.DisposeByNone()        error   -- inherited
```

which is exactly the pair `GameComponent` itself takes. An inherited member must
not be renamed by being inherited, so the caller resolves the inherited set
first and passes ONE overload-group map through both mappings.

### What the missing member cost

`Game::Dispose(true)` walks a snapshot of `Components` and calls
`IDisposable::Dispose()` on every element that has one. CNA-Go's loop looks for
the projected spelling of that member:

```go
type gameComponentDisposable interface{ DisposeByNone() error }
type gameComponentSimpleDisposable interface{ Dispose() error }
```

A `DrawableGameComponent` matched **neither**. It has `Dispose(bool)`, which is
not `IDisposable::Dispose`. So a drawable component added to `Game.Components`
was skipped in silence: `UnloadContent` never ran, the four device-service
handlers were never removed, it was never removed from the collection, and its
`Disposed` never fired.

Proved before the fix, on the unmodified tree:

```text
components before=1 after=1
Game.Dispose left 1 components
```

## `ldarg.0` is the whole object, and composition splits it

Fixing the first defect exposed the second. With `DisposeByNone` present, the
component was reached — and stayed in the collection anyway.

In CLR, `ldarg.0` inside a base body is the **whole** object. Private named
composition splits that object in two, and the base half has no way back:

```text
GameComponent::Dispose(bool)
  ldarg.0; callvirt Collection`1::Remove(!0)   -- removes the BASE half, and the
                                                 collection holds the DERIVED one
  ldfld Disposed; ldarg.0; ldsfld EventArgs::Empty; callvirt Invoke(object, !0)
                                              -- announces the BASE half as the
                                                 sender of the derived object's event
```

`Remove` returned false, and the reference `pop`s that result, so nothing
anywhere reported it. All three inherited events announced a sender no consumer
could match against the component it registered on.

A site that uses `ldarg.0` merely to REACH A FIELD is unaffected: the state it
reads is the base's own. Only object uses are affected, and there are six of
them in `GameComponent`:

| Go member              | uses | reference |
| ---------------------- | ---- | --------- |
| `SetEnabled`           | 1 | `set_Enabled`: the second `ldarg.0` is the sender argument |
| `SetUpdateOrder`       | 1 | `set_UpdateOrder`, same shape |
| `OnEnabledChanged`     | 1 | raises with `ldarg.0`, ignoring its own `sender` parameter |
| `OnUpdateOrderChanged` | 1 | same |
| `DisposeByBoolean`     | 2 | `Components.Remove(this)` **and** `Disposed(this, Empty)` |

### The projection

The composed base holds an unexported reference to the outermost derived
object, installed by the derived constructor and read only through an
unexported accessor:

```go
type GameComponent struct {
    ...
    derived IGameComponent   // the CLR `this`; nil when this IS the whole object
}

func (c *GameComponent) bindDerived(derived IGameComponent) { c.derived = derived }

func (c *GameComponent) self() IGameComponent {
    if c.derived != nil {
        return c.derived
    }
    return c
}
```

Nothing about the settled composition rule changes. There is still no embedding,
no exported base field, and no public `Base`, `Parent` or `AsGameComponent`
accessor — the verifier still refuses each. This is what makes private
composition **correct** rather than merely private.

The inherited `Dispose()` forwards to the derived `DisposeByBoolean`, not to the
composed base's, because `GameComponent::Dispose()` is `callvirt` and dispatches
to the override. Forwarding to the base would reproduce the base's slot and skip
`UnloadContent` and the four removals — the exact bug Go's lack of virtual
dispatch invites.

## Verifier support

`measureXNACompositionIdentity` parses the package's Go **syntax tree**, because
the claim is about method bodies, which the declaration-level surface
deliberately does not carry. Five claims:

1. every COMPOSED XNA base has an identity entry, so a future composed base
   cannot skip the decision by saying nothing;
2. the Go base declares the unexported derived-reference field;
3. the base declares the unexported `self` and `bind` members, and each actually
   touches that field;
4. every recorded identity site reaches `self` **exactly as many times** as the
   reference pushes `ldarg.0` as an object there;
5. every projected derived type's constructor installs itself.

The count in (4) is load-bearing. `DisposeByBoolean` has two identity uses, and a
"does the member reach `self` at all" test passes with one of them still spelled
as the bare receiver — which is precisely the first mutation below.

```text
XNA_COMPOSED_IDENTITY_SITES    5
XNA_COMPOSED_IDENTITY_USES     6
XNA_COMPOSED_IDENTITY_BINDINGS 1
```

## Falsification

Eight mutations, each staged over a temporary copy of the real package sources,
each compiling — the bare receiver is a valid `GameComponent` everywhere it
appears.

| mutation | caught by |
| -------- | --------- |
| `Disposed` announces the base half | verifier + behaviour |
| `Components.Remove` names the base half | verifier + behaviour |
| `EnabledChanged` announces the base half | verifier + behaviour |
| `UpdateOrderChanged` announces the base half | verifier |
| `set_Enabled` passes the base half as sender | verifier |
| `set_UpdateOrder` passes the base half as sender | verifier |
| derived constructor does not install the CLR `this` | verifier + behaviour |
| `self` returns the receiver and never reads `derived` | verifier |
| `self` reads `derived` and discards it (`&& false`) | **behaviour only** |

The last row is recorded rather than hidden. It compiles, it is called the
recorded number of times at every site, and it comes from a constructor that
binds — nothing structural distinguishes it. That is the exact boundary between
the two kinds of evidence, and it is why the behaviour tests exist beside the
gate rather than instead of it.

## Scoreboard

```text                                          before   after
TOTAL_DIAGNOSTICS                                 208     208
MISSING_MEMBER                                     76      76
COMPLETE_TYPES                                    120     120
PARTIAL_TYPES                                       5       5
UNEXPECTED_MEMBER                                   0       0

TARGET_MEMBERS                                   1937    1938
EXPECTED_GO_MEMBERS                              3279    3280
XNA_INHERITED_PUBLIC_MEMBERS                       14      15
XNA_INHERITED_MEMBER_PROJECTIONS                   24      25
XNA_INHERITED_PUBLIC_MEMBERS_OVERRIDDEN             -       3
XNA_INHERITED_ATTRIBUTED_MEMBERS                   24      25
XNA_COMPOSED_IDENTITY_SITES                         -       5
XNA_COMPOSED_IDENTITY_USES                          -       6
XNA_COMPOSED_IDENTITY_BINDINGS                      -       1

behavior corpus                                   687     690
composition-identity mutation controls              0       9
native ABI header-tree pinning                   path  content
```

## One more thing the milestone measured

The native ABI verifier recorded its header root as a PATH, and the default
`-headers ../../cnanext/modules/c-api/include` reads a live checkout that has
advanced past the artifact the pinned library was built from. The two trees
differ:

```text
pinned  ~/deps/cna-c-abi-0.21.0/include      62c3f6e4...c90bff6
live    ../../cnanext/modules/c-api/include  2d7445e7...698ed
```

Both produce identical measurements at this revision -- the divergence is
documentation comments in `CNA/C/devices.h` -- so nothing was wrong, and
nothing would have said so either. The report now records
`canonical_header_sha256` and `canonical_header_files` beside the library's own
digest, on the principle the library already followed: reports retain content
identity, not an ephemeral qualification path. `schema_version` moves 2 -> 3.

`TARGET_MEMBERS` moves 1937 -> 1938 and `EXPECTED_GO_MEMBERS` 3279 -> 3280:
`DrawableGameComponent.Dispose` became `DisposeByBoolean`, and the inherited
`DisposeByNone` is a projection the surface did not have.

## Why this had to come first

Every remaining graphics base takes the same shape:

```text
GraphicsResource   public    Dispose()          virtual FINAL
GraphicsResource   family    Dispose(bool)      newslot virtual
Texture2D          family    Dispose(bool)      override
RenderTarget2D     family    Dispose(bool)      override
```

and `GraphicsResource::~GraphicsResource` raises `Disposing(this, Empty)` while
`ToString()` falls back to `Object::ToString()` on `this`. Composing that chain
over a name-keyed exclusion would have dropped `Dispose()` from every texture,
and composing it without an object identity would have announced the wrong
sender from every resource. Both rules are now settled and measured.
