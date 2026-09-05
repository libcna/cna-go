# Foundation 99 — the GamerServices classification, measured

No type was projected in this milestone. What changed is a **classification**,
from `ACTIONABLE_LOCAL` to `BLOCKED_PLATFORM`, and the change is the result of
reading two IL bodies end to end rather than re-reading a note.

`GLOBAL_ACTIONABLE_LOCAL` 1 → **0**. `GLOBAL_UNREVIEWED` stays **0**.
`MISSING_TYPE` stays 1 and `COMPLETE_TYPES` stays 256: nothing was projected and
nothing was hidden.

## Why the entry was inconsistent with itself

`frontierActionableLocal` is documented in the registry as meaning "nothing
external blocks the family: the reference is readable, the dependencies are
projected, and the work is implementation." The `Blocker` field is required for
every classification except that one, "where 'nothing external' is the whole
claim."

Foundation 97 re-measured this family, corrected its NOTE — the note went on to
describe a blocker in detail — and left the CLASSIFICATION at
`ACTIONABLE_LOCAL`. So the entry simultaneously claimed nothing external blocked
it and described what blocked it. Only one of those can be true.

## What the component actually is

`GamerServicesComponent` lives in `Microsoft.Xna.Framework.Game.dll`, not in the
GamerServices assembly, and it is 94 bytes of IL over four members.

```
.ctor(Game game)                       8 bytes   base..ctor(game)
Initialize()                          61 bytes
Update(GameTime gameTime)             13 bytes
GamerServicesDispatcher_InstallingTitleUpdate  12 bytes   Game.Exit()
```

`Initialize` has four steps and **three are the dispatcher**:

1. `GamerServicesDispatcher.WindowHandle = Game.Window.Handle`
2. `GamerServicesDispatcher.InstallingTitleUpdate += <the private handler>`
3. `GamerServicesDispatcher.Initialize(Game.Services)`
4. `base.Initialize()`

`Update` has two and the **first** is `GamerServicesDispatcher.Update()`.

So pumping that dispatcher is the component's whole job. Everything else it does
is forward to `GameComponent`.

## What the dispatcher is

`GamerServicesDispatcher.Initialize` reaches
`KernelMethods.Initialize(UserPacketBuffer)`. `KernelMethods` contains a nested
`ProxyProcess` holding a `System.Diagnostics.Process serverProcess` and eight
`native int` handles named `triggerCallEvent`, `callDoneEvent`,
`proxyProcessWantsToTalk`, `sharedAsyncDataSafeToWrite`, `aSyncHResultPtr`,
`asyncManagedCallArgument`, `aSyncManagedCallTypePtr` and `parentExitEvent`.

The whole assembly contains exactly **seven** P/Invokes, and every one of them
is `pinvokeimpl("Kernel32.dll" winapi)`:

| P/Invoke | what it is for |
| --- | --- |
| `CreateEvent`, `SetEvent`, `WaitForMultipleObjects`, `CloseHandle` | the Win32 named events the two processes signal each other with |
| `CreateFileMapping`, `MapViewOfFile`, `UnmapViewOfFile` | the shared-memory region the packet buffer lives in |

`set_WindowHandle` installs a `Microsoft.Xna.Framework.Input.WindowMessageHooker`
on the game's HWND so the LIVE guide can intercept window messages.

That is Games for Windows LIVE: a separate proxy process, Win32 IPC and a
window-message hook. It is a **platform subsystem this profile's host does not
have**, which is the registry's own definition of `BLOCKED_PLATFORM`.

## Why this is not `BLOCKED_UPSTREAM_CNA`

The previous note reached for CNA: "CNA has 107 gamer/guide routes and NONE of
them is `GamerServicesDispatcher.Initialize`, `.Update` or `.WindowHandle`."
That is true and it is not the root. CNA's gamer and guide routes are its own
facility with its own semantics; a CNA route named `cna_gamer_services_update`
would not make the projection do what the reference does, because what the
reference does is drive a Windows LIVE proxy. Classifying this as a missing
native route would imply that adding one would unblock it, and it would not.

## Why the type is not projected anyway

Its `.ctor`, `base.Initialize()`, `base.Update(gameTime)` and the private
handler's `Game.Exit()` are all reproducible. A projection built from only those
would compile, satisfy `IGameComponent` and `IUpdateable`, and be addable to
`Game.Components`.

It would also do nothing. No window handle would be set, no title-update
subscription made, no dispatcher initialised and nothing pumped each frame. A
consumer who added it would get a component that silently fails at the one thing
it exists for, and would have no way to tell from its behaviour. That is worse
than its absence, and the absence is now classified rather than pending.

## What would lift it

A Games for Windows LIVE client on this host, or a CNA facility that IS the
dispatcher rather than a parallel one — meaning it drives the same LIVE session
the reference's proxy drives, so that a projected `Initialize` and `Update`
would have the reference's observable effect. Neither exists.

## What is NOT claimed

That the component could never be projected. That the 107 CNA gamer and guide
routes are useless — they are not; they are simply a different facility, and
nothing in this milestone measured them. And no claim is made about the Xbox 360
assembly, which was not read.
