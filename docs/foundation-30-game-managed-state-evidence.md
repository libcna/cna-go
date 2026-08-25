# Foundation 30 — Game's managed Components and Services, and the component engine

This milestone answers one question with IL rather than with architecture
preference: **which parts of `Microsoft.Xna.Framework.Game` are managed CLR
state, and therefore belong in Go rather than behind the C ABI?**

The answer is that `Components`, `Services`, and the whole component-tracking
engine that feeds them are ordinary managed list work. None of it reaches a
host, a window, a device, or native code. Routing any of it through
`Go -> C ABI -> C++` would invent a native owner and a native failure mode the
reference does not have.

`Game` remains **partial** and that is deliberate. This milestone completes the
coherent components-and-services slice and nothing else.

## Reference authority

All seven pinned XNA assemblies were re-verified by hash this session and read
with `ikdasm`. The behavior of the inherited BCL collections is read from the
admitted implementation assembly, also re-verified by hash:

```text
Microsoft.Xna.Framework.Game.dll
  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0

mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
  5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
  ~/.wine-cna-xna40/drive_c/windows/Microsoft.NET/Framework/v4.0.30319/
```

The public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
`REFERENCE_MEMBERS` did not move.

## The two members, and why they are not fallible

```text
Game::get_Components
  IL_0000: ldarg.0
  IL_0001: ldfld  class GameComponentCollection Game::gameComponents
  IL_0006: ret                                        // code size 7

Game::get_Services
  IL_0000: ldarg.0
  IL_0001: ldfld  class GameServiceContainer Game::gameServices
  IL_0006: ret                                        // code size 7
```

Seven bytes each. No allocation, no validation, no throw site, no `callvirt`.
Both fields are assigned exactly once during construction and are never
reassigned anywhere in the class, so **the identity a caller observes never
changes** and a getter cannot fail.

`Game` is otherwise a native-backed facade in CNA-Go — the CNA runtime owns the
host and the frame loop — so its members are fallible by default. These two are
the exception, recorded in `managedStoredMembers` with accessor-level keys:

```go
"Microsoft.Xna.Framework.Game": {
    "property-get|Components": true,
    "property-get|Services":   true,
},
```

Both properties are get-only in the reference, so there is no setter to
classify.

### A mapper defect this exposed

`managedStoredMembers` is the mirror image of `managedFallibleMembers`: one
*lowers* fallibility on a native-backed owner, the other *raises* it on a
pure-managed one. The property branch of the mapper rebuilt each accessor's
result list from scratch but **inherited** the whole-member fallibility flag,
which made the accessor-level decision one-directional — it could raise
fallibility but never lower it. `Game::Components` is the first get-only stored
property on a fallible-by-default owner, and it is what surfaced the skew: the
projected results correctly lost their `error` while the flag still claimed one.

The fix recomputes the flag per accessor, exactly as the results are recomputed.
It is guarded by a structural invariant over the **whole** expected surface
rather than a targeted mutation:

> every expected member's `ErrorAdded` flag equals whether its projected results
> end in `error`

`TestProjectedFallibilityFlagAlwaysMatchesTheProjectedResults` checks all 3255.

## The construction point

The reference constructor is 338 bytes. The part CNA-Go owns is:

```csharp
gameServices   = new GameServiceContainer();      // field initializer
// ... base ctor, FrameworkDispatcher.Update(), EnsureHost(), launchParameters
gameComponents = new GameComponentCollection();
gameComponents.ComponentAdded   += GameComponentAdded;
gameComponents.ComponentRemoved += GameComponentRemoved;
content = new ContentManager(gameServices);
```

`gameServices` comes from a field initializer, so it exists before anything else
runs — which is why `ContentManager` can be handed it later in the same
constructor. Both objects are allocated in `NewGame` in that order, and Game's
own two handlers are subscribed immediately after the collection exists.

**The subscription order is observable and is preserved.** Game's handlers are
registered first, so a consumer's later handler always runs after the engine has
finished tracking or untracking the component, exactly as a CLR multicast
delegate runs its invocation list in subscription order. A consumer therefore
observes a consistent `Game` from inside their own `ComponentAdded`.

## Game does not sort its components

This is the finding that mattered most, because the obvious implementation is
wrong. `Game` declares **five** private lists beside the public collection:

```text
List<IUpdateable>    updateableComponents
List<IUpdateable>    currentlyUpdatingComponents
List<IDrawable>      drawableComponents
List<IDrawable>      currentlyDrawingComponents
List<IGameComponent> notYetInitialized
```

`updateableComponents` and `drawableComponents` are kept ordered
**incrementally**: a component is placed at its ordered position when it is
added, and re-placed when its order changes. There is no per-frame sort, and no
sort at all. The per-frame cost is a copy.

The relationship to `Components` is explicit and one-directional: `Components`
is the only public collection, and these lists are maintained purely by the two
handlers the constructor subscribes.

### The comparers are deliberately not total orders

`UpdateOrderComparer` and `DrawOrderComparer` are private sealed
`IComparer<T>` singletons whose IL is the same 47-byte shape:

```csharp
Compare(x, y):
    x == null && y == null -> 0
    x == null              -> 1      // a null sorts AFTER a non-null
    y == null              -> -1
    x.Equals(y)            -> 0      // Object.Equals: reference identity
    x.Order < y.Order      -> -1
    otherwise              -> 1      // EQUAL orders report 1, never 0
```

Two **distinct** components with the same order never compare equal. The only
way `BinarySearch` can return a non-negative index is that the searched
component is already in the list.

That reframes the guard in the add handler entirely:

```csharp
int i = updateableComponents.BinarySearch(u, UpdateOrderComparer.Default);
if (i < 0) { ... place and subscribe ... }
```

`if (i < 0)` is a **"not already present"** guard, not an "equal order found"
guard. A component that is somehow already tracked is placed nowhere and
subscribed to nothing.

### Ties are stable, and not by accident

`List<T>.BinarySearch` reaches
`ArraySortHelper<T>::InternalBinarySearch`, whose exact body is:

```csharp
int lo = index, hi = index + length - 1;
while (lo <= hi) {
    int i = lo + ((hi - lo) >> 1);
    int order = comparer.Compare(array[i], value);
    if (order == 0) return i;
    if (order < 0) lo = i + 1; else hi = i - 1;
}
return ~lo;
```

Because an equal-order list element reports `1` — "this element sorts after the
value being placed" — the search always converges to the left of a run of
equal-order elements. The reference then walks forward explicitly:

```csharp
i = ~i;
while (i < list.Count && list[i].UpdateOrder == u.UpdateOrder) i++;
list.Insert(i, u);
```

So a new component with an existing order lands **after** every component that
already has that order: ties keep insertion order. `Array.BinarySearch` gives no
ordering guarantee among equal elements, so **the forward walk is what makes the
result deterministic**, and it is reproduced literally rather than replaced by a
stable sort.

The same four-step placement appears in four sites — the two add paths and the
two order-changed paths — and is projected once as `orderedInsertionIndex`.

### The order-changed handlers read the sender

```csharp
private void UpdateableUpdateOrderChanged(object sender, EventArgs e)
{
    IUpdateable u = sender as IUpdateable;      // ldarg.1 — the SENDER
    updateableComponents.Remove(u);
    int i = updateableComponents.BinarySearch(u, UpdateOrderComparer.Default);
    if (i < 0) { i = ~i; while (...) i++; updateableComponents.Insert(i, u); }
}
```

It reads `sender`, never `e`. That is why `GameComponent` raises its
order-changed events with **itself** as sender rather than forwarding the sender
it was handed.

Removal comes first, so the search can never find the component and the
re-insertion always happens. And because the re-insertion uses the same forward
walk as a fresh add, a component whose order changes **to an order that already
exists** moves to the end of that order's run rather than back where it was. The
subscription is **not** renewed — the reference does not re-attach the handler.

### The `inRun` guard, in both directions

```csharp
// GameComponentAdded
if (inRun) e.GameComponent.Initialize();
else       notYetInitialized.Add(e.GameComponent);

// GameComponentRemoved
if (!inRun) notYetInitialized.Remove(e.GameComponent);
```

`inRun` is set in `RunGame`, and its position is exact:

```csharp
graphicsDeviceManager = Services.GetService(typeof(IGraphicsDeviceManager)) as ...;
graphicsDeviceManager?.CreateDevice();
this.Initialize();          // inRun is still FALSE here
this.inRun = true;          // only now
this.BeginRun();
...
finally { if (!endRunRequired) inRun = false; }
```

So a component added from **inside** the `Initialize` override is still queued —
and because the drain loop re-reads `Count`, it is initialized by that same
drain — while one added afterwards is initialized on the spot. CNA-Go raises the
flag at the same point of the sequence, on the native `initialize` frame hook,
and lowers it when the blocking `Run` returns. The native CNA host plays
`GameHost`'s part; that substitution is recorded rather than hidden.

### Initialization comes first, and a failure stops everything after it

In `GameComponentAdded` the initialization step precedes both placements. A
component whose `Initialize` fails is therefore never placed in either derived
list and never subscribed to, because the reference's exception leaves the rest
of the handler unreached. In Go the same is true for a second reason:
`EventSource.Raise` stops at the first failing handler, so a consumer's own
`ComponentAdded` handler does not run either.

The collection has nevertheless **already mutated**, because `InsertItem` mutates
before it announces. That asymmetry was pinned in Foundation 26 and this
milestone depends on it rather than changing it.

### Nil components

`ClearItems` announces every element with no null check, unlike `RemoveItem`,
so `Clear` is the one path that hands the engine a nil component. The reference
survives it because `List.Remove(null)` finds nothing and both `isinst` tests
fail. The projection does the same. `InsertItem` announces nothing for a nil
component, so a nil never reaches the add handler at all.

### The two type tests are independent

A component that is `IUpdateable` but not `IDrawable` is tracked in exactly one
list, and a component that is neither is still initialized or queued. Both `isinst`
tests are separate `if`s over the same component and neither gates the
initialization step.

## Event subscription tokens

CLR unsubscribes by delegate equality:

```csharp
u.UpdateOrderChanged -= this.UpdateableUpdateOrderChanged;
```

Go function values are not comparable, so the settled projection names the
*registration* instead of the handler. The registration therefore has to travel
with the component, which is why the two derived lists hold
`{component, subscription}` entries rather than bare components. That is an
implementation consequence of an already-settled decision, not a new one, and it
falls out naturally: the order-changed handler re-inserts the same entry, which
is exactly the reference not re-attaching its handler.

## What this milestone did NOT do

- **No C ABI was added.** Not one function. The component engine is managed CLR
  behavior and stays in Go.
- **No CNA C++ change.** The pinned Foundation-11 binary is untouched.
- **`GameCallbacks` is unchanged.** Its five members are exactly as before.
- **No unrelated `Game` member was completed.** `Window`, `Content`,
  `LaunchParameters`, `IsMouseVisible`, the timing controls, `RunOneFrame`,
  `Tick`, `SuppressDraw`, `GraphicsDevice`, activation and disposal all remain
  missing, and `Game` remains partial.
- **`System.Exception` was not reopened.** Component initialization failure
  travels on the existing opaque Go error channel.

## Scoreboard

```text                        before   after
TARGET_TYPES                    121      121
TARGET_MEMBERS                 1755     1757
TOTAL_DIAGNOSTICS               313      311
MISSING_MEMBER                  177      175
MISSING_TYPE                    136      136
COMPLETE_TYPES                  116      116
PARTIAL_TYPES                     5        5
REFERENCE_MEMBERS              2964     2964
EXPECTED_GO_MEMBERS            3255     3255

behavior corpus                 598      603
mutation inventory              478      480
```

Every mismatch, leak, allowlist and unmeasured counter is zero.

### Game's missing members

```text
before   39
completed 2   Components, Services
after    37
```

Both belong to the coherent slice. Nothing else was touched.
