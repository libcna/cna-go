# Foundation 63 — the Content subsystem, and Game::Content

A whole namespace arrives. `ContentManager` is the first type in
`Microsoft.Xna.Framework.Content` CNA-Go projects, it is COMPLETE on arrival,
and the two `Game` members it unblocks close with it.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 169     166
MISSING_TYPE                                      123     122
MISSING_MEMBER                                     46      44
COMPLETE_TYPES                                    130     131
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      134     135
TARGET_MEMBERS                                   2189    2202
BOUND_FUNCTIONS                                    95     104
PROTOTYPE_TYPE_POSITIONS                          324     357
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              245     251
behavior corpus                                   712     720
external canary tests                              81      83
native stress scenarios                            14      15
capability rows                                    62      63
```

`Game` goes from four missing members to two.

## Creation is deferred, and CNA is the reason

```il
ContentManager::.ctor(IServiceProvider)
  if (serviceProvider == null) throw new ArgumentNullException("serviceProvider");
  this.serviceProvider = serviceProvider;
```

That is the whole constructor. It reaches no device, no file and no pipeline —
the device is resolved lazily, at load time, out of the service provider it
stored.

CNA's is not:

```c
CNA_Result cna_content_manager_create(
    CNA_Handle graphics_device,          // callback-scoped, borrowed
    const CNA_ContentManagerCreateInfo*,
    CNA_Handle* out_content_manager);
```

A **callback-scoped** device handle exists only inside a lifecycle callback. A
projection that created eagerly would therefore refuse a constructor the
reference accepts anywhere — including at field-initializer time, which is where
`Game`'s own manager is built.

So the native manager is created at the **first operation that needs a device**,
and the constructors do what the reference's do: store and return. Every such
operation reaches the device by construction, so it is inside a callback when it
runs, and nothing had to be relaxed to make that true.

The observable consequence is small and is tested: `Unload` on a manager that
never loaded is a successful no-op, because there is no cache — which is also
what the reference's `Unload` over an empty cache does.

## Ownership

```text
ContentManager       OWNED, generation-checked, cna_content_manager_destroy
a loaded Texture2D   OWNED, INDEPENDENTLY -- not PARENT_OWNED
```

The second is the one worth stating. CNA says a loaded texture "remains valid
across content-manager unload or destruction and must be destroyed before the
parent game", which is the reference's rule too: `Unload` disposes what the
manager still holds, and a caller who kept a reference keeps a live object. So
the projection destroys a loaded texture itself, through the ordinary
`cna_texture2d_destroy` route, and never through the manager. The native
scenario destroys both handles it obtains and the manager separately, in that
order.

## The load set is CLOSED, and the refusal comes first

`Load<T>` in the reference is open: it reads an `.xnb` and hands back whatever
the content reader produced. CNA exposes one route per asset kind, and CNA-Go
binds the one whose asset type it projects:

```text
*graphics.Texture2D    cna_content_manager_load_texture2d
```

`SpriteFont`, `SoundEffect`, `TextureCube` and `Effect` have CNA routes too and
are absent for one reason each: **CNA-Go projects no Go type for them yet**. Each
is a missing TYPE rather than a missing loader, and binding a loader for a type
with no Go identity would be a route with nothing to return.

A `T` outside the set is refused **by name**, and the refusal happens **before**
the device is resolved. That ordering was a real defect, caught by its own test:
the first version resolved the native manager first, so a consumer asking for a
`SpriteFont` learned that no graphics device service was registered — true,
unhelpful, and a statement that would stop being true the moment they registered
one. The planted mutation that restores the old order fails the test.

## Two recorded divergences

Neither is papered over.

**`OpenStream` appends `.xnb`; CNA's loader does not.** The reference's
`OpenStream` is `TitleContainer.OpenStream(Path.Combine(RootDirectory, assetName) + ".xnb")`,
and the projection does exactly that. CNA resolves `<root>/<name>` and decodes
what it finds — the native scenario loads a loose PNG through it. So the two
members resolve to different files for the same asset name, and each is faithful
to the thing it projects: `OpenStream` to XNA's, `Load` to CNA's.

**`ReadAsset` does not bypass the cache.** The reference's reads without
consulting it. CNA offers one load route and it is cached, so a second
`ReadAsset` of one name answers from CNA's cache where the reference would read
again. Its `Action<IDisposable>` parameter is accepted and unused for a stated
reason: it exists so a disposable the reader created is registered with the
manager and released by `Unload`, and CNA's manager owns that relationship
itself.

## Game::Content, and where a member lives when two packages need it

```il
get_Content:  ldarg.0; ldfld ContentManager Game::content; ret
set_Content:  if (value == null) throw new ArgumentNullException();
              this.content = value;
```

Seven bytes and seventeen. Both are managed in full, and `get_Content` joins
`managedStoredMembers` as the plainest entry in that table — one field read, no
validation, no throw site, and nothing about reading it reaches the content
pipeline.

The framework package cannot name `ContentManager`, because the Content package
imports it. The settled cross-package cycle rule therefore puts **both accessors
in the Content package**, as package functions taking the receiver first:

```text
Game::get_Content  ->  content.GameContent(*framework.Game) *ContentManager
Game::set_Content  ->  content.SetGameContent(*framework.Game, *ContentManager) error
```

The **field stays on Game**, where the reference keeps it. That is not a detail:
the alternative — a registry in the Content package keyed by `*Game` — would keep
every Game that ever had content alive for the life of the process, and a test
pins that two Games never share a manager.

Three closures in `internal/servicebridge` connect the halves, all typed `any` on
both sides, all installed from package inits, none exported anywhere public:

```text
SetGameContentAccessors   framework installs the field read and the field write
SetGameContentCreator     content   installs `new ContentManager(this.Services)`
```

The creator runs **inside `NewGame`**, at the reference's own statement position
— between the component collection's handlers and the window's `Paint`
subscription — so the manager exists from construction and its identity is fixed.
A program that never linked the Content package constructs a Game with an empty
slot rather than being refused, which is safe because the only two members that
can observe the field live in that package.

## Evidence

A fifteenth native scenario, twenty isolated cycles, against **both** qualified
artifacts, with identical results:

```text
CONTENT_CYCLES                20     CONTENT_LOADS                    20
CONTENT_MANAGER_CREATIONS     20     CONTENT_LOAD_PIXEL_CHECKS        20
CONTENT_IDENTITY_CHECKS       20     CONTENT_LOAD_READBACK_REFUSALS    0
CONTENT_ROOT_ROUND_TRIPS      20     CONTENT_CACHE_CHECKS             20
CONTENT_ASSET_PATH_CHECKS     20     CONTENT_TYPE_REFUSALS            20
CONTENT_UNLOAD_CALLS          20     CONTENT_LOAD_REFUSALS             0
CONTENT_DISPOSAL_CHECKS       20
```

Three of these carry the weight.

**The root directory round trip** is the proof no managed test can make. Once the
native manager exists, `RootDirectory` answers **CNA's** copy, because CNA is
what resolves an asset path and a value that disagreed with it would be a lie
about where a load looks. With no native half there is nothing to disagree with,
so the mutation that makes the getter answer the managed field survives every
in-package test and is caught only here.

**The pixel check** is what separates "CNA returned a texture" from "CNA decoded
this asset". The asset is the 2×2 PNG this tool already encodes, its four texels
are known, and all four are read back exactly. A pipeline handing back an empty
surface of the right size would pass a dimension check.

**The cache check** deletes the file and loads the same name again. It succeeds,
which is CNA's cache answering — the behaviour XNA guarantees for a reference
type, and the reason the projection keeps no cache of its own: two caches would
be two answers.

Nine mutations were planted; eight fail:

```text
CAUGHT   the getter builds a fresh manager per call
CAUGHT   the setter drops the argument-null guard
CAUGHT   the constructor never creates the manager
CAUGHT   the load reaches the device before deciding the type
CAUGHT   Dispose(false) takes the disposing branch
CAUGHT   the constructor accepts a nil service provider
CAUGHT   one content manager is shared by every Game
CAUGHT   the refusal does not name the unsupported type
SURVIVED RootDirectory answers the managed copy, not CNA
```

The survivor is recorded rather than counted: it is unreachable from the
in-package tests for the structural reason above, and it is the mutation the
native scenario's round trip exists to catch. Run against the artifacts, the
scenario fails it.

Three ABI prototype controls and two layout controls ship with the routes. The
prototype ones are the create-info passed **by value** where CNA takes a pointer,
a load taking a bare `const char*` instead of a `CNA_StringView` — which would
hand CNA a name with no length — and an asset-path copy that drops its capacity.
The layout ones are invisible to C: the root directory moved behind `reserved`,
which changes only where CNA reads eight bytes of pointer from, and the two
`uint32` header words swapped, which tells a **versioned** create-info that it was
handed a struct of size 1 and version 24.

## A verifier that had been failing since Foundation 58

`tools/external_consumer` did not run this milestone's tests: it did not compile,
and had not since the substitutability rule landed. The canary fixture still
spelled `SpriteBatch.Draw`'s texture position as `*graphics.Texture2D` and
`Texture2D.Bounds` as fallible, and both had changed underneath it.

That is exactly the failure mode the external canary exists to catch, arriving
from the other direction: a consumer-visible signature changed and the only
consumer in the repository was not rebuilt. It is repaired here rather than
noted — eleven signatures updated, plus an assignment proving a `RenderTarget2D`
satisfies the `Texture2DReference` position from outside the module, which is the
substitution rule's whole point for a downstream user.

```text
EXTERNAL_CONSUMER_TESTS=83  EXTERNAL_CONSUMER_FAILURES=0  STATUS=PASS
```

## What this milestone does not claim

- **One asset kind loads.** `Texture2D` and nothing else. The other four CNA
  loaders wait on Go types, not on this projection.
- **No `.xnb` was read.** CNA's pipeline decoded a loose PNG in both qualified
  environments. Whether it reads a compiled XNB container is untested here,
  because this repository has no XNB corpus and inventing one would be inventing
  the evidence with it.
- **`OpenStream` is not proved against a real asset.** It resolves CNA's path and
  appends the reference's `.xnb`; the scenario proves it fails with the resolved
  path in the message, which is what it can prove without such a corpus.
- **The service provider is `any`.** `System.IServiceProvider` is a BCL interface,
  which the settled rule projects to no Go interface, so a manager whose provider
  is not a `GameServiceContainer` cannot resolve a device and says so.
- **Nothing here is a content pipeline.** CNA-Go compiles no asset; it asks CNA to
  read one.
