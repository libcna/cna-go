# Foundation 33 — the XNA base frontier, measured

This milestone completes no type. It closes a hole.

Foundation 32 found that both types `GameComponent` was supposed to unblock are
blocked by the same thing — CNA-Go has no architecture for **XNA-to-XNA class
inheritance** — and then found that this blocker was not recorded anywhere at
all. Foundation 29 established that a deferred base must name what blocks it and
that the verifier fails one that records nothing. That discipline covered BCL
bases only. There is a second base frontier, and it was silent.

## The hole

`Microsoft.Xna.Framework.Graphics.Texture2D` declares sixteen members in the
pinned contract. It inherits nine more:

```text
from Texture           LevelCount, Format
from GraphicsResource  Name, Tag, GraphicsDevice, IsDisposed,
                       Dispose(), ToString(), Disposing
```

The contract does **not** redeclare them, because CLR metadata does not.
CNA-Go projects `Texture2D` with none of them, and until this milestone nothing
in the binding recorded that. The same silence covered `SpriteBatch`, which
inherits seven from `GraphicsResource`.

This is the Foundation-26 finding again — *"projecting only the seven members
`GameComponentCollection` declares would have produced a collection nothing
could be added to"* — but for a base **inside** the contract rather than outside
it, and with no registry watching.

## What is measured now

Every CLR class in the pinned profile that another class in the same profile
inherits from is recorded with a status, and a `DEFERRED` one must name what
blocks it:

```text
XNA_BASE_RELATIONSHIPS                    12
XNA_BASE_DERIVED_TYPES                    41
XNA_DEFERRED_BASE_BLOCKERS                25
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED 245
```

| base | derived types | public members it contributes | blockers |
|---|---:|---:|---|
| `GameComponent` | 2 | 9 | ARCHITECTURE, SUBSYSTEM |
| `Graphics.GraphicsResource` | 11 | 7 | ARCHITECTURE, NATIVE_OWNERSHIP |
| `Graphics.Texture` | 3 | 2 | ARCHITECTURE, TRANSITIVE |
| `Graphics.Texture2D` | 1 | 13 | ARCHITECTURE, TRANSITIVE |
| `Graphics.TextureCube` | 1 | 7 | ARCHITECTURE, TRANSITIVE |
| `Graphics.Effect` | 6 | 4 | ARCHITECTURE, TRANSITIVE, SUBSYSTEM |
| `Graphics.IndexBuffer` | 1 | 9 | ARCHITECTURE, TRANSITIVE |
| `Graphics.VertexBuffer` | 1 | 9 | ARCHITECTURE, TRANSITIVE |
| `Audio.SoundEffectInstance` | 1 | 14 | ARCHITECTURE, SUBSYSTEM |
| `Content.ContentManager` | 1 | 5 | ARCHITECTURE, SUBSYSTEM |
| `Content.ContentTypeReader` | 1 | 3 | ARCHITECTURE, SUBSYSTEM |
| `Design.MathTypeConverter` | 12 | 5 | ARCHITECTURE, SUBSYSTEM |

Counts are computed from the pinned contract, not hand-written, so they cannot
drift.

## The substantive rule

> **No derived type of a `DEFERRED` XNA base may be reported `COMPLETE`.**

Completeness asserts that a type's whole public surface is present. A derived
type of a deferred base is missing its inherited half, so the claim would be
false. `Texture2D` and `SpriteBatch` are legitimately `PARTIAL` today — their gap
is already `MISSING_MEMBER` — and this makes that a **checked fact** rather than
a coincidence that a future milestone could quietly break by "completing" one of
them without its inherited surface.

The other four rules are the Foundation-29 discipline transposed: every link the
contract declares must be recorded; a deferred base must name at least one
blocker; every blocker carries one of four recorded classes and a non-empty
detail; and a registry entry for a base nothing derives from is a stale claim.

Seven negative controls, in one shared table behind both the named test and the
mutation inventory, plus two closure tests that assert the registry covers
exactly the twelve links the contract declares over exactly forty-one derived
types, and a live test that no complete type inherits from a deferred base.

## The open architecture decision, stated exactly

Every one of the twelve carries the same `ARCHITECTURE` blocker, because it is a
necessary condition for projecting any of them. Projecting a derived XNA class
needs three things CNA-Go does not have:

1. **A composition and forwarding rule for a base that is itself an XNA
   identity.** The settled BCL base composition holds the base as a private
   generic adapter and re-exposes its public surface through measured forwarding
   members. An XNA base is not a generic adapter — it is a projected type with
   its own identity, its own constructor, and its own place in the scoreboard.
2. **A third provenance class.** Today every expected Go member is exactly one of
   XNA-declared or BCL-inherited, and a test partitions the whole surface to
   prove the two halves are disjoint and exhaustive. An XNA-inherited member is
   neither, and adding it moves `EXPECTED_GO_MEMBERS` for 41 types.
3. **An override adapter for the base's protected virtuals.** `GameCallbacks`
   solves this for `Game` alone. `DrawableGameComponent` overrides
   `GameComponent`'s `Initialize`, `Dispose(bool)`, and declares its own
   protected `LoadContent`/`UnloadContent`; a derived class needs both a way to
   supply overrides and a way to call its base — which is Foundation 31's
   architecture, generalized from one class to a family.

Exported Go embedding of an XNA base is already refused by
`BASE_MAPPING_MISMATCH` and stays refused: it would publish a Go type the
contract never declares, promote a method set the derived type never declared,
and imply a Go subtype relationship CLR inheritance does not create.

## The two component types, member by member

### `DrawableGameComponent`

Beyond the architecture blocker:

- `get_GraphicsDevice` returns `Graphics.GraphicsDevice` — a partial type,
  reached across the package boundary the cross-package cycle rule governs — and
  throws `InvalidOperationException(PropertyCannotBeCalledBeforeInitialize)`
  when `deviceService` is null.
- `Initialize` calls `base.Initialize()`, guards on a private `initialized`
  flag, resolves `IGraphicsDeviceService` out of `Services`, **throws
  `InvalidOperationException(MissingGraphicsDeviceService)` when it is absent**,
  subscribes four device handlers, calls `LoadContent` when a device exists, and
  sets `initialized`. Nothing in CNA-Go can publish that service, so a faithful
  `Initialize` would always throw.
- `Dispose(bool)` calls the virtual `UnloadContent`, unsubscribes the same four
  handlers, and then calls `GameComponent.Dispose(bool)` with a non-virtual
  `call` — which is a base call, and needs Foundation 31's architecture
  generalized.
- `LoadContent` and `UnloadContent` are `family newslot virtual` no-ops whose
  whole purpose is to be overridden.

### `GamerServicesComponent`

Beyond the architecture blocker:

- `Initialize` reads `Game.Window.Handle`. `GameWindow` is a missing type,
  `Game::Window` is a missing member, and the value is a native window handle.
- It then calls `GamerServicesDispatcher.set_WindowHandle`, `add_InstallingTitleUpdate`
  and `Initialize(IServiceProvider)`, and `Update` calls
  `GamerServicesDispatcher.Update()`. That class lives in
  `Microsoft.Xna.Framework.GamerServices.dll`, which is **not one of the seven
  pinned assemblies**, so there is no reference authority for it. Admitting the
  assembly would not help: CNA has no GamerServices runtime at all, so every
  member would be inert.

## `GraphicsDeviceManager` service publication — audited, and blocked

`GraphicsDeviceManager(Game)` does register itself into `Game.Services`, and
the IL is unambiguous:

```csharp
if (game == null) throw new ArgumentNullException("game", Resources.GameCannotBeNull);
this.game = game;
if (game.Services.GetService(typeof(IGraphicsDeviceManager)) != null)
    throw new ArgumentException(Resources.GraphicsDeviceManagerAlreadyPresent);
game.Services.AddService(typeof(IGraphicsDeviceManager), this);
game.Services.AddService(typeof(IGraphicsDeviceService),  this);
game.Window.ClientSizeChanged      += GameWindowClientSizeChanged;
game.Window.ScreenDeviceNameChanged += GameWindowScreenDeviceNameChanged;
game.Window.OrientationChanged      += GameWindowOrientationChanged;
this.graphicsProfile = ReadDefaultGraphicsProfile();
```

Two service keys, the duplicate check **before** the registration, and the
duplicate check on `IGraphicsDeviceManager` only.

**CNA-Go cannot perform either registration**, and the reason is structural
rather than a matter of effort. `GameServiceContainer.AddService` reproduces the
reference's assignability check, and CNA-Go's `GraphicsDeviceManager` is
assignable to neither key:

- `IGraphicsDeviceManager` needs `CreateDevice`, `BeginDraw`, `EndDraw`. All
  three are missing members; `CreateDevice` delegates to `ChangeDevice`, which is
  device configuration.
- `IGraphicsDeviceService` needs `GraphicsDevice()` and four two-accessor device
  events. `GraphicsDevice()` is a missing member, and the four events have **no
  raise path**: CNA publishes no device-created, device-reset, device-resetting
  or device-disposing signal, so accessors that exist but never fire would not be
  the contract.

Registering it anyway would make `AddService` fail, and faking either interface
would put an event on the contract that never fires. Both are refused. The new
`DECLARED_INTERFACE_CONFORMANCE` measurement makes this precise from the other
side: `GraphicsDeviceManager` is excluded from the conformance claim **because it
is partial**, and its 20 missing members are the gap.

The ordering is worth recording for whoever resolves it: the reference registers
the services **before** it touches `game.Window`, so the managed half precedes
every blocked step and nothing observable would be reordered by supplying it
later.

## Scoreboard

```text                                     before   after
XNA_BASE_RELATIONSHIPS                        0       12
XNA_BASE_DERIVED_TYPES                        0       41
XNA_DEFERRED_BASE_BLOCKERS                    0       25
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED      0      245

mutation inventory                          493      500

TARGET_TYPES / TARGET_MEMBERS              122/1776  unchanged
TOTAL_DIAGNOSTICS                           310      310
COMPLETE_TYPES / PARTIAL_TYPES / MISSING    117/5/135 unchanged
```

**No type was completed and no counter of coverage moved.** That is what this
milestone is: measurement. Every mismatch, leak, allowlist and unmeasured counter
remains zero — the existing binding was already consistent with the new rule,
which is itself the finding: `Texture2D` and `SpriteBatch` were partial for the
right reason, and now that is checked.
