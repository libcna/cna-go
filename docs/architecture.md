# Architecture

```text
consumer callbacks and XNA-shaped Go values/facades
                         |
 Microsoft/Xna/Framework[/Graphics|Input|Content]
                         |
      internal/interop runtime + ownership registry
                         |
          one cgo bridge and typed dlsym manifest
                         |
                 canonical CNA C ABI 0.7
                         |
             CNA C++ XNA implementation
```

## Public boundary

Go import paths represent XNA namespaces. Public packages expose only mapped
XNA types and the documented Go language adapters needed where C# features do
not exist. There is no invented `CNA/Framework` package. Native handles, result
codes, callback contexts, ownership records, and C declarations cannot leave
`internal/interop`.

Pure XNA values are Go structs with copy semantics. Native reference objects are
pointer facades containing unexported control state. The structural extractor
measures the actual source surface, and the formal mapper compares it with the
pinned XNA 4.0 Windows runtime contract. A red strict report means missing XNA
work, while leak-only must remain green.

The formal language projection—including properties, overloads, statics,
operators, generics, nested types, events, ref/out, BCL types, and Game's
inheritance adaptation—is normative in `xna-go-mapping.md` and its machine
rules.

## Game host and callback flow

Go cannot subclass XNA `Game`, so `Game` is a concrete native host and
`GameCallbacks` is the explicit protected-virtual adapter:

```text
NewGame(callbacks)
  -> Game.Run locks the current OS thread
  -> dlopen + exact ABI/symbol admission
  -> cgo.NewHandle(runtime state)
  -> CNA Game create + callback-table copy
  -> CNA owns and drives Initialize/Load/Update/Draw/Unload
  -> callback trampoline recovers panic and stores error
  -> native run returns
  -> children destroyed, Game destroyed, generation invalidated
  -> cgo.Handle deleted, library closed, OS thread unlocked
  -> stored callback error/panic returned to caller
```

There is no Go-side frame loop. `GameTime` is built only from native CNA tick
snapshots. `Game.Exit` calls CNA's request-exit route.

## Thread and generation state

A goroutine is not a native-thread identity. All Game creation, callbacks,
owner resources, shutdown, and destruction occur while the Run goroutine is
locked with `runtime.LockOSThread`. Interop also records a native `pthread_t`
identity. A wrong-thread operation returns an error; a failed destroy retains
its handle for a later owner-thread retry.

Every successful Run obtains a monotonic generation token. Owned resources and
borrowed facades carry that token. Shutdown invalidates it and unregisters
owner associations, so a facade retained from Game 1 is rejected during or
after Game 2.

## Ownership

- `MANAGED_VALUE`: no native lifetime (`Vector2`, `GameTime`, `Viewport`).
- `OWNED`: exactly one Go control block must destroy the CNA handle.
- `BORROWED`: never destroyed or retained beyond its CNA scope; GraphicsDevice
  reacquires its callback-scoped native handle on each operation.
- `PARENT_OWNED`: child facade invalidated with its native parent.
- `PROCESS_GLOBAL`: process-wide native state with no object handle.

Game owns its native Game. GraphicsDeviceManager, Texture2D, and SpriteBatch are
owned Game children. GraphicsDevice is borrowed. Explicit `Dispose` is
authoritative and idempotent after success; finalizers are not used. Parent
disposal is rejected while children remain. Game teardown makes a reverse-order
best effort and reports failures.

## Memory and callback safety

Persistent user data is a `runtime/cgo.Handle` encoded as `uintptr`; C never
retains a Go pointer. Byte buffers for synchronous texture decoding remain live
only for the call. CNA copies callback tables and sprite commands according to
the canonical headers. No C function pointer reaches Go or a public package.

Panic recovery exists both around user callback invocation and at the exported
cgo trampoline. The first error/panic wins and is returned by `Game.Run` after
native shutdown. See `native-abi.md` for the typed manifest and compiler-backed
proof.

## Deliberate deferrals

Foundation 1 has no fake ContentManager/XNB, BasicEffect, 3D capability layer,
audio, media, storage, or mobile/Web facade. Native runtime qualification is a
separate claim from whether `go build` can target an operating system.
