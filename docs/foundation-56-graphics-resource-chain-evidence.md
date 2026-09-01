# Foundation 56 — the graphics base chain, composed

`GraphicsResource -> Texture -> Texture2D` is projected, and `SpriteBatch`
arrives with it because it derives from `GraphicsResource` too. Three types
become complete, twenty-two inherited members appear on two derived types, and
the base that Foundation 25 measured as blocking seven missing types stops
blocking any of them for inheritance reasons.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 208     204
MISSING_TYPE                                      132     130
MISSING_MEMBER                                     76      74
COMPLETE_TYPES                                    120     123
PARTIAL_TYPES                                       5       4
UNEXPECTED_MEMBER                                   0       0
ABI_MISMATCHES                                      0       0
```

The scoreboard understates it, and that is worth stating plainly:
`MISSING_MEMBER` moved by two while **twenty-two** members were projected,
because composing a base ADDS to the expected surface at the same moment it
closes it. `SpriteBatch` went from ten missing members to ten by gaining eleven
and closing eleven.

## The chain, as the IL declares it

```text
.class public abstract GraphicsResource extends System.Object implements IDisposable
  .field private   string         _localName
  .field private   object         _localTag
  .field           GraphicsDevice _parent
  .field assembly  uint64         _internalHandle
  .field assembly  bool           isDisposed
  .field private   EventHandler`1<EventArgs> <backing_store>Disposing
  .method assembly .ctor()                    -- INTERNAL: no public constructor

.class public abstract Texture extends GraphicsResource
  .field SurfaceFormat _format
  .field int32         _levelCount
  -- public surface: get_Format and get_LevelCount, both `ldfld`. Nothing else.

.class public Texture2D extends Texture implements IGraphicsResource
  .field int32 _width
  .field int32 _height
```

Seven public members on `GraphicsResource`, two on `Texture`. Eleven classes
derive from `GraphicsResource` and three from `Texture`.

## Ownership: one native owner, at the base

The decision Foundation 25 recorded as `NATIVE_OWNERSHIP` and deferred:

> it is `public abstract` with an `assembly` constructor, carries
> `.field assembly uint64 _internalHandle`, and its Dispose(bool) dispatches to
> the C++/CLI ~GraphicsResource/!GraphicsResource pair. Who owns a graphics
> resource's lifetime across the C ABI is an open decision.

Answered as **OWNED, at the base**, for the reason the reference puts the handle
there: there is one native object per logical object, and a derived Go wrapper
holding its own `*interop.Resource` would be a second owner of the same thing.

```text
GraphicsResource.resource  *interop.Resource   OWNED, generation-checked
GraphicsResource.device    *GraphicsDevice     BORROWED facade, stored, never released
Texture.format/levelCount                      managed, from the created texture
Texture2D.width/height                         managed, from the created surface
```

`interop.Resource` carries its own kind tag, so the type-specific destruction
the reference's `ReleaseNativeObject` overrides perform is already inside
`Resource.Dispose`. That is why the family needs no per-derived release hook and
why one unexported `releaseNativeObject` covers `Texture2D` and `SpriteBatch`
both.

## Disposal, reproduced per override

```text
GraphicsResource::Dispose(bool)
  if (disposing) ~GraphicsResource();
  else           try { !GraphicsResource(); } finally { Object.Finalize(); }

  !GraphicsResource()  isDisposed = true;
  ~GraphicsResource()  if (!isDisposed) { !GraphicsResource();
                                          Disposing(this, EventArgs.Empty); }
```

Three facts a reader should not have to reconstruct.

**It is idempotent.** `GameComponent`'s disposal has no flag and re-runs on
every call; this one has `isDisposed` and raises `Disposing` exactly once, ever.

**The flag is set before the event.** A `Disposing` handler observes
`IsDisposed == true` — and, because the derived override releases first, a
resource whose native half is already gone.

**`Dispose(false)` announces nothing.** It sets the flag and returns. That is
the finalizer path, which Go never takes on its own, and it is projected because
the contract declares the member.

### The two overrides disagree, and both are reproduced as written

```text
Texture2D::Dispose(bool)
  if (disposing) { try { ~Texture2D(); } finally { base.Dispose(true); } }
  else           { try { !Texture2D(); } finally { base.Dispose(false); } }

  !Texture2D()  if (!isDisposed) { ReleaseNativeObject(TRUE); CleanupSavedData(); }
```

`!Texture2D()` is the same body on both branches and passes a hardcoded
`ldc.i4.1` regardless of `disposing`, so **`Texture2D.Dispose(false)` destroys
the native texture**.

```text
SpriteBatch::Dispose(bool)
  try {
      if (disposing && !IsDisposed) { spriteEffect?.Dispose(); DisposePlatformData(); }
  } finally { base.Dispose(disposing); }
```

`SpriteBatch`'s release is guarded on `disposing` and **does not** run on the
finalizer path. The two ILs genuinely disagree; smoothing them to match would
have been a choice neither Microsoft made.

Both base calls are in a `finally`, so the flag and the event are not skipped by
a failing release, and the release's error is the one returned.

## The fallibility correction

`isFallible` now reads the **declaring** type, not the type a member is
projected onto, because whether a member reaches a runtime boundary is a
property of its own body. Otherwise a base member would need registering once
per derived type, and the eleven `GraphicsResource` descendants would have
needed eleven identical entries.

Six members entered `managedStoredMembers` on their own IL, and three left
fallibility they should never have had:

| member | body | was | is |
| ------ | ---- | --- | -- |
| `GraphicsResource::get_GraphicsDevice` | `ldfld _parent` | — | infallible |
| `GraphicsResource::get_IsDisposed` | `ldfld isDisposed` | — | infallible |
| `GraphicsResource::Name` get+set | `Dictionary` under a `Monitor` | — | infallible |
| `GraphicsResource::Tag` get+set | the same | — | infallible |
| `Texture::get_Format` | `ldfld _format` | — | infallible |
| `Texture::get_LevelCount` | `ldfld _levelCount` | — | infallible |
| `Texture2D::get_Width` | `ldfld _width` | `(int32, error)` | `int32` |
| `Texture2D::get_Height` | `ldfld _height` | `(int32, error)` | `int32` |
| `Texture2D::get_Bounds` | two of the above | `(Rectangle, error)` | `Rectangle` |

`Name` and `Tag` are the interesting pair: their bodies LOOK like they reach a
device.

```text
get_Name  if (_internalHandle != 0)
              return _parent.Resources.GetCachedName(_internalHandle);
          return _localName;
```

`DeviceResourceManager::GetCachedName` is 79 bytes of
`Dictionary<ulong, ResourceData>` under a `Monitor`, answering `String.Empty`
for an absent key. No D3D call, no throw site. It is per-resource storage
reached indirectly — which is exactly why both accessors are listed here while
the `GraphicsDeviceManager` setters, which really do push to a device, are not.

**This paragraph originally said CNA has no counterpart and no route to one.
That was wrong, and Foundation 57 measured it.** CNA has twelve
`cna_graphics_resource_*` routes, including `set_name`, `copy_name`, `get_tag`,
`set_tag` and `get_is_disposed`. The classification above is unchanged, because
it is read off the REFERENCE's body — but the reason it stands is now measured
rather than assumed, and every unused route carries a recorded reason in
`tools/native_abi`'s `deliberatelyUnboundRoutes`. See
[foundation-57](foundation-57-unbound-routes-evidence.md).

The three `Texture2D` corrections removed an error channel nothing could ever
put a value in. The comment that justified it claimed the getters "read a
disposed-checked native texture"; they do not, in either runtime. CNA-Go caches
CNA's reported description at construction exactly as the reference caches
D3D's, and both then answer from a managed field — including after disposal,
which the tests now pin.

## Where the description comes from

```text
Texture2D::InitializeDescription(Nullable<SurfaceFormat> format)
  GetLevelDesc(0, &desc);
  if (!format.HasValue) format = ConvertWindowsFormatToXna(desc.Format);
  _width = desc.Width; _height = desc.Height;
  Texture::InitializeDescription(format.Value);   // _format, then _levelCount
```

The nullable is load-bearing. A **constructor** passes the format it was asked
for, so `_format` is the REQUESTED one; **FromStream** passes no value, so
`_format` is whatever the decoder produced. Width, height and level count come
from the created surface on both paths. CNA-Go reproduces exactly that split.

## The verifier work this needed

**The identity registry gained a forwarding link.** `Texture` has no identity
site of its own and holds no copy of the CLR `this`; it takes a binding and
passes it to `GraphicsResource`. A middle link that kept its own copy would be a
second answer that could disagree, so the registry records `ForwardsTo` and the
gate refuses a middle link that also declares state or sites.

**`GraphicsResource` has two identity sites, and they are different kinds of
use.** `Dispose(bool)` needs the OBJECT, as the `Disposing` sender.
`ToString` needs its TYPE:

```text
ToString()
  local = Name; if (!IsNullOrEmpty(local)) return local;
  return base.ToString();      // System.Object -> the RUNTIME type's full name
```

An unnamed `Texture2D` must answer
`"Microsoft.Xna.Framework.Graphics.Texture2D"`, resolved through the CLR `this`
across two composition links.

```text
XNA_COMPOSED_BASE_RELATIONSHIPS      1 ->  3
XNA_COMPOSED_DERIVED_TYPES           2 -> 16
XNA_COMPOSED_DERIVED_TYPES_PROJECTED 1 ->  4
XNA_COMPOSED_IDENTITY_SITES          5 ->  7
XNA_COMPOSED_IDENTITY_USES           6 ->  8
XNA_COMPOSED_IDENTITY_FORWARDS       -  ->  1
XNA_COMPOSED_IDENTITY_BINDINGS       1 ->  4
XNA_INHERITED_PUBLIC_MEMBERS        15 -> 119
XNA_INHERITED_MEMBER_PROJECTIONS    25 -> 171
XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED 227 -> 144
```

## Two fixtures moved rather than being weakened

`deferredBaseFixtureName` was `GraphicsResource`, and
`TestATypeWithUnprojectedInheritedMembersIsNotComplete` measured `Texture2D`
over the deferred `Texture`. Both now measure a family that is still deferred —
`BasicEffect` over `Effect`, which reaches a shader subsystem CNA-Go maps no
part of — because a fixture that is relaxed to keep passing stops being
evidence. `Effect` is also the entry least likely to be composed next.

## Falsification

Seven mutations on the projection, each compiling:

| mutation | caught by |
| -------- | --------- |
| the constructor's format is ignored and the surface's used | `TestTextureDescriptionIsCachedAtConstruction` |
| width read from the height field | the same |
| level count wired to the format | the same |
| the `isDisposed` guard removed | `TestGraphicsResourceDisposalIsIdempotentAndRaisesOnce` |
| the flag set AFTER the event | the same |
| `Dispose(false)` announces | `TestGraphicsResourceDisposeFalseSetsTheFlagAndRaisesNothing` |
| the inherited `Dispose()` forwards to the composed base's slot | **`tools/native_stress` only** |

The last row is the one worth reading. Every managed observable agrees for both
versions — the flag and the event are the base's either way — so the control has
to be the native handle:

```text
SaveAsPng after the inherited Dispose() = <nil>, want ErrDisposed:
the native texture is still alive, so Dispose() reached the composed base's
slot instead of Texture2D's override
```

Seven more mutations attack the identity mechanism through the verifier: the
`ToString` fallback answering with the base's name, the `Disposing` sender being
the base half, `self` never reading the installed object, and each of the three
constructors failing to install it — plus the middle link swallowing the
forwarded binding instead of passing it on.

## Qualification

```text
gofmt / go vet / go test ./...            clean
go test -race ./...                       clean
api_compat report                         204 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           694 observations, 0 failures
runtime capabilities                      56 rows, PASS
native ABI (pinned 0.21.0 headers)        78 routes, 164 layouts, 0 mismatches
native stress                             0 crashes, 0 UAF, 0 double frees
cna-go-template 60 frames                 PASS
```

## What this unblocks

`RenderTarget2D` derives from `Texture2D`, whose base chain is now composed and
whose surface is complete, so inheritance is no longer its blocker. What remains
is CNA — render-target creation, binding and unbinding routes — and the
substitutability question a derived texture raises wherever the profile names
`Texture2D` in a public signature. Those are the next two things to measure, in
that order.
