# Foundation 28 — the device-publication contract, and the frontier re-derived

Foundations 26 and 27 settled the BCL collection families. This milestone
consumes what they made reachable, and then re-derives the whole frontier so
what remains is characterised by evidence rather than by the previous
handoff's summary.

One type is completed: `Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService`.
Three types that the dependency graph reports as reachable are shown, from IL,
**not** to be — and each blocker is named down to the exact member.

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll  560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.Game.dll      b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
```

Both are the pinned Foundation-1 references, re-verified by hash and read with
`ikdasm`.

## `IGraphicsDeviceService`

The contract a component looks up in `Game.Services` to find the current
device and to learn when it is created, reset, or going away. Five CLR members
— one property and four events — project to nine Go identities.

### Every operation is infallible, and that is the interesting part

The declaration itself proves nothing: an abstract interface has no bodies. The
settled `evidenceRule` says the boundary is read from the reference
implementor's IL, and `Microsoft.Xna.Framework.Game.dll` ships exactly one:

```text
GraphicsDeviceManager::get_GraphicsDevice
  ldarg.0
  ldfld GraphicsDevice GraphicsDeviceManager::device
  ret
```

Seven bytes. It hands over a stored reference. It does not create a device,
reset one, query one, or reach native code, and it returns null before a device
exists. **Nothing in the contract can fail.**

This is the sharpest per-contract split in the profile, and it lands on one
class. `GraphicsDeviceManager` also implements `IGraphicsDeviceManager`, which
stays unclassified and fallible because its `CreateDevice`, `BeginDraw` and
`EndDraw` genuinely cross into the runtime. Two contracts on the same type
disagree about fallibility because the boundary is read **per contract, never
per class** — the same rule that made `IUpdateable` infallible while
`IGameComponent` stayed fallible on `GameComponent`.

The four event accessors carry an error from the settled event accessor
projection rather than from this contract's boundary. In the reference they are
the ordinary compiler-generated `Delegate.Combine`/`Delegate.Remove` pair over
`System.EventHandler`1<System.EventArgs>`.

### Declaring the contract publishes no device

CNA-Go has no implementor. `GraphicsDeviceManager` remains a protected partial
that raises none of these events, `Game` exposes no `Services` container, and
nothing in the binding resolves or publishes the contract.

The external canary proves a cross-package claim the in-repo tests cannot: a
type declared **outside** CNA-Go, in its own module, can satisfy a contract
declared in the **graphics** package whose accessors are spelled in
**framework**-package types. A conformer that declares the nine members and
nothing else compiles, which is the exact shape claim; and a consumer holding
the contract can subscribe and unsubscribe and do nothing else, because there
is no raise operation on the contract at all.

## Three types the graph says are reachable and the IL says are not

A type-level dependency graph cannot see a member-level blocker. All three were
re-derived rather than inherited from the previous handoff.

### `GameComponent` — blocked on `Game::Components`

Its accessors are managed field work and `Initialize` and `Update` are each a
bare `ret` of code size 1, exactly as Foundation 23 recorded. But `Dispose` is
not:

```text
GameComponent::Dispose(bool disposing)
  if (!disposing) return
  Monitor.Enter(this)
    if (get_Game() != null)
        get_Game().get_Components().Remove(this)     <-- Game::Components
    if (Disposed != null)
        Disposed(this, EventArgs.Empty)
  Monitor.Exit(this)
```

A disposed component **removes itself from its game's component collection**.
`Microsoft.Xna.Framework.Game::Components` is one of the 39 missing members of
the protected partial `Game`, so projecting `GameComponent` faithfully would
require implementing a member of a protected runtime partial. That is a
separate runtime milestone and is deliberately not entered here.

The finding is worth recording precisely, because the gap is now very small:
`Game.Components` returns a `GameComponentCollection`, and that type is
**complete** as of Foundation 26. What stands between CNA-Go and a real
component loop is no longer a mapping question at all — it is the runtime
decision about `Game`.

`GameComponent` blocks `DrawableGameComponent` and `GamerServicesComponent` in
turn.

### `GraphicsResource` — blocked on native ownership

It is the base of ten types and the sole remaining blocker of seven, so it is
the highest-value name on the list. It is also unambiguously native:

```text
.class public abstract auto ansi beforefieldinit GraphicsResource
       extends [mscorlib]System.Object
       implements [mscorlib]System.IDisposable
{
  .field private  string _localName
  .field private  object _localTag
  .field          GraphicsDevice _parent
  .field assembly uint64 _internalHandle          <-- a native object handle
  .field assembly bool   isDisposed
```

Three separate blockers, any one of which is sufficient:

1. **`public abstract`** with an **`assembly`** constructor, so the pinned
   contract declares no public constructor and nothing could create one.
2. **`assembly uint64 _internalHandle`** — it owns a native object.
3. `Dispose(bool)` dispatches to a C++/CLI destructor and finalizer pair,
   `~GraphicsResource()` and `!GraphicsResource()`, which release that handle.
   The assembly references `Microsoft.VisualC`, which is where that shape comes
   from.

This is exactly the decision Foundation 24 declared open: *mapping
`System.IDisposable` makes a dependency syntactically complete; it does not
decide native ownership or lifetime for any concrete type.* Projecting
`GraphicsResource` is that decision, not a BCL one.

### `Microphone` — reachable only because of Foundation 27

`Microphone::All` returns `ReadOnlyCollection<Microphone>`, so Foundation 27's
signature adapter moved it from blocked to dependency-complete. It stays
device-blocked for the reason Foundation 25 recorded: it is a capture device
with a `Finalize`, and `GetData` fills a caller's buffer from hardware.

The adapter unblocked its **signature**, not its **behavior**, and the
distinction is the point.

## The re-derived frontier

Eighteen missing types are dependency-complete. Every one is blocked on
behavior, and the blockers group into four kinds:

| kind | types |
| --- | --- |
| device or hardware | `Mouse`, `GamePadCapabilities`, `TouchPanelCapabilities`, `Microphone`, `DisplayMode`, `RendererDetail` |
| XACT or audio native | `AudioCategory`, `Cue`, `SoundEffectInstance` |
| media, content, filesystem | `MediaSource`, `Video`, `TitleContainer`, `FrameworkDispatcher` |
| runtime or ownership decision | `GraphicsResource`, `GameComponent`, `GameWindow`, `EffectAnnotation` |

The remaining BCL and CLR shapes, with the types each alone blocks:

```text
5  System.Attribute                  the five ContentSerializer* attributes
5  System.Exception                  four exception types plus ContentLoadException
3  ExternalException                 three more exception types
1  Dictionary`2                      LaunchParameters
4  ReadOnlyCollection`1 AS A BASE    the four Model*Collection types
```

`ReadOnlyCollection<T>` is `SUPPORTED` as a signature adapter and `DEFERRED`
only as a base; its four base consumers are blocked on their element types
regardless.

## Scoreboard

```text                          before   after
TARGET_TYPES                       120     121
TARGET_MEMBERS                    1746    1755    1 property + 8 event accessors
COMPLETE_TYPES                     115     116
MISSING_TYPE                       137     136
TOTAL_DIAGNOSTICS                  314     313
MISSING_MEMBER                     177     177    protected partials untouched
EXPECTED_GO_MEMBERS               3255    3255
behavior corpus                    595     598
external canary tests               14      16
```

Every mismatch, leak, allowlist and unmeasured counter is zero.
