# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 21 are complete. Milestones 17 through 21 were
produced in one session as **five local commits that have not been pushed**;
`develop` is 5 ahead of `origin/develop`.

```text
START   HEAD = origin/develop = 766e2a0b1c7ed06a2eb67abf00e2ab108afc7551
FINAL   HEAD = fc18d7b1bf9f41bb3a90d47d27a72222fe4bb4ef
        origin/develop unchanged at 766e2a0
        worktree clean, git diff --check clean
```

| # | commit    | milestone                                     | types | Go identities |
| - | --------- | ---------------------------------------------- | ----- | ------------- |
| 17 | `54b9989` | pure managed CLR class + per-operation fallibility | 2 | 20 |
| 18 | `680b52c` | projected CLR interface contracts              | 4     | 18            |
| 19 | `6dd2fb0` | `System.IntPtr` + `PresentationParameters`     | 1     | 23            |
| 20 | `2418e9e` | read-only `TouchCollection`                    | 2     | 19            |
| 21 | `fc18d7b` | `GameServiceContainer`                         | 1     | 4             |

Ten types, 84 mapped Go identities.

## Scoreboard

```text                        start    final
TARGET_TYPES                   103      113
TARGET_MEMBERS                1619     1703
TOTAL_DIAGNOSTICS              331      321
MISSING_TYPE                   154      144
MISSING_MEMBER                 177      177
COMPLETE_TYPES                  98      108
PARTIAL_TYPES                    5        5
MISSING_TYPES                  154      144

INTERFACE_WITNESS_PROJECTIONS   25       25
PACKFROMVECTOR4_WITNESSES       17       17
TOVECTOR4_WITNESSES              8        8

mutation inventory             217      347
behavior corpus                487      564
```

Every mismatch, leak, allowlist, and unmeasured counter is zero throughout. The
five protected partial runtime types are untouched: Game 39,
GraphicsDeviceManager 40, GraphicsDevice 70, SpriteBatch 16, Texture2D 12,
combined 177.

## Mapping rules this session settled

These are general rules, not per-type exceptions. All are declared in
`tools/api_compat/mapping-rules.json` and documented in
`docs/xna-go-mapping.md`.

1. **CLR `class` is not evidence of native backing.** A type is admitted to
   `pureManagedTypes` only when authoritative XNA IL proves its selected public
   behavior is entirely managed. Admission changes fallibility and nothing else:
   a class still projects as `*T`, a struct still projects as `T`. The five
   native-backed runtime types are deliberately excluded.

2. **Fallibility belongs to one projected operation.** Keys are
   `constructor|Name`, `method|Name`, `field|Name`, `property-get|Name`,
   `property-set|Name`, and `property|Name`. The whole-property key still marks
   both accessors and is correct only where the reference validates on read
   *and* on write. `ERROR_MAPPING_MISMATCH` names the accessor and the direction
   of every disagreement.

3. **Interface fallibility comes from reference implementor IL**, in the
   assembly that declares the interface — never from speculation about an
   implementor that does not exist. One contract may mix boundaries;
   `IEffectFog` is the first that does.

4. **`System.IntPtr` projects to `uintptr`**, meaning only the opaque
   pointer-sized bit value the contract carries at that position.
   `RAW_HANDLE_LEAK` admits a public `uintptr` **only** where XNA metadata
   declares `System.IntPtr` at that exact position; everything else still leaks.

5. **A BCL interface whose members the XNA type already declares publicly** adds
   no projected surface and needs no separate Go interface. Covers
   `IList<T>` and `System.IServiceProvider`.

6. **A collection that declares its own public enumerator type** projects that
   type from `GetEnumerator`; the `Iterator<T>` adapter is for collections that
   declare none.

## Reference quirks now preserved — do not "correct" them

- `AudioListener`/`AudioEmitter` constructors store `Vector3.Zero` **unflipped**
  while the getters flip, so `Position` and `Velocity` read back with a
  **negative-zero Z** (`0x80000000`). `Forward` and `Up` do not, because their
  stored values were flipped once already.
- `AudioEmitter.set_DopplerScale` guards with `bge.un.s`, so `+0`, `-0`,
  `+Infinity`, **and every NaN** are accepted; only negative-ordered values
  throw.
- `PresentationParameters`'s constructor's only statement is
  `IsFullScreen = true`, so a fresh descriptor is full-screen on a zero-sized
  back buffer. XNA 4.0 has **no `Clear`** on this type.
- `TouchCollection`'s `IList<T>` mutators throw `NotSupportedException`
  unconditionally, validating nothing first, so `Insert(-99, x)` reports
  not-supported rather than out-of-range. `CopyTo` checks capacity in **64-bit**
  arithmetic. `IndexOf` uses the equality operator, so a location `Equals`
  accepts is missed. `FindById`'s miss yields a **zero** location, not the
  `Id` -1 sentinel `TryGetPreviousLocation` uses.
- `GameServiceContainer.AddService` checks for a **duplicate before checking
  assignability**. `RemoveService` on an unregistered type succeeds and
  `GetService` on one returns nil with no error.

## Reference authority

Behavior comes from retained original Microsoft assemblies in `~/Downloads/win`,
verified by hash and read with `ikdasm`. Exception message strings were read
from each assembly's embedded `.resources` stream with `monodis --mresources`.

```text
Microsoft.Xna.Framework.dll             38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
Microsoft.Xna.Framework.Graphics.dll    560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.Game.dll        b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
Microsoft.Xna.Framework.Input.Touch.dll b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25
Microsoft.Xna.Framework.Xact.dll        a14d5364dca7cf49fb90639e87ba04d52b59a700dc9198efa5707ce8eae28f0a
Microsoft.Xna.Framework.Video.dll       17538b1ca9d48a993e2cd88c96b436df08e7abb4aec5d4758eb21feb580d6e06
```

Public *surface* remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`. FNA and
MonoGame remain comparators only and were not consulted.

## Native provenance — unchanged

CNA was **not rebuilt**. The exact Foundation-11 pinned binary reproduces both
committed native reports.

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED          23/67/96/28/2/5, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED     on the exercised 20-cycle stress surface
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED
```

Rediscover it by hash rather than by path:

```sh
find / -name 'libcna_c_api.so*' -type f 2>/dev/null -exec sha256sum {} + | grep ^e912cd1d
```

`native_abi` reproduces the committed report key for key, differing only in
`header_root`, which the committed evidence stores normalized. `native_stress`
reproduces every counter identically with `GO_RACE_STATUS=PASS`.

## Toolchain

The Go toolchain lived only under the system temporary directory. It is now
preserved durably, byte-for-byte, at `~/deps/go1.24.4` (go1.24.4 linux/amd64),
so a future session does not have to re-download it. `GOCACHE` was already
durable at `~/.cache/go-build`.

## The frontier is exhausted — the next milestone is a decision

**Every dependency-complete missing type has been individually re-derived from
IL this session.** All eleven that remain are deferred with evidence, not with
an assumption:

| type | why |
| ---- | --- |
| `Graphics.DisplayMode` | `assembly` constructor only; values come from display enumeration |
| `Input.GamePadCapabilities` | `assembly`/`private` constructors only; device capability |
| `Input.Touch.TouchPanelCapabilities` | **no constructor at all**; device capability |
| `Audio.RendererDetail` | `assembly` constructor only; values come from XACT renderer enumeration |
| `Audio.AudioCategory` | `assembly` constructor needing an `AudioEngine`; every method P/Invokes `UnsafeNativeMethods.Engine` |
| `Audio.SoundEffectInstance` | `assembly` constructors; 18 throw sites through `Helpers.ThrowExceptionFromErrorCode` |
| `Graphics.EffectAnnotation` | `assembly` constructor; unmanaged `calli` through `GraphicsHelpers.GetExceptionFromResult` |
| `Media.Video` | `assembly` constructor needing a `GraphicsDevice` and a content file |
| `Media.MediaSource` | `assembly` constructor; values come from the media backend |
| `FrameworkDispatcher` | `Update()` drives the microphone, dynamic-audio, media, and storage pumps |
| `Input.Mouse` | device state plus `MouseMessageHooker` |
| `TitleContainer` | needs `TitleLocation.Path` and the file system |

Everything else in the profile is blocked on an **unmapped BCL shape**, and each
of those is a new public-API decision that this session deliberately did not
take. Ranked by how many missing types each would unblock:

```text
  26  System.IDisposable                             disposal / ownership design
  24  System.EventArgs        \ 
  21  System.EventHandler`1   / the event handler type
  13  System.ComponentModel.ITypeDescriptorContext   the Design namespace
  12  System.Globalization.CultureInfo
  12  System.Collections.IDictionary
   9  System.Collections.ObjectModel.ReadOnlyCollection`1
   8  System.Exception (+3 ExternalException)        CLR exception types as Go types
   6  System.Attribute                               content serializer attributes
   4  System.Collections.Generic.List`1+Enumerator
```

### The two highest-value decisions

**1. The event handler type** (`System.EventArgs` + `System.EventHandler<T>`).
Nine missing types are blocked on this *alone*, including
`Graphics.GraphicsResource` (fan-out 10), `IDrawable`, `IUpdateable`, and
`IGraphicsDeviceService`. The event *accessor* mapping is already settled
(`AddXHandler` returning `EventSubscription`, `RemoveXHandler` taking one); what
is unsettled is the handler parameter type, which currently degrades to `any`
through an undeclared `mapType` fallthrough. Foundation 18 deferred three
interfaces on exactly this rather than lock in `any`. Plausible options:

- a named `EventHandler` func type plus an `EventArgs` adapter, added to
  `languageAdapters` alongside `EventSubscription`;
- a generic `EventHandlerOf[T]` func type mirroring the CLR generic;
- keep `any` but *declare* it, accepting a lossy signature.

**2. Disposal and ownership** (`System.IDisposable`). Twenty-six missing types
declare it. It is also the second half of what `GraphicsResource` needs, and it
touches the protected partial `GraphicsDevice`. The prompt-level guidance has
consistently deferred this as a separate architecture milestone.

Resolving both would unblock `GraphicsResource` and, through it, the whole
graphics resource family.

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~200s under -race
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode strict     # expected red: 321 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -headers ~/deps/cna-c-abi-0.7.0/include \
    -library "$CNA_NATIVE_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
go run ./tools/api_compat --mode report     # run LAST so committed evidence keeps report mode
git diff --check
```

Always send native reports to an explicit `-output` under a scratch directory so
a locally rebuilt library or an absolute header path never rewrites the
committed evidence.

### Deterministic artifact and isolated consumer

```sh
git ls-files -z | sort -z | tar --null --files-from=- \
  --transform 's,^,cna-go/,' --owner=0 --group=0 --numeric-owner \
  --mtime='@0' --mode='u+rw,go+r,go-w' --sort=name -cf - | gzip -n > OUT.tar.gz
```

At `fc18d7b` that yields sha256
`0244074eb42a0b163aa40b38230558aa9c07368270bc28b33cd08e4444dce8a4` over 237
entries, reproduced independently in the same run. Extracted into
`build-consumer/isolated`, it passes every gate with **no development-checkout
dependency** and regenerates `api-compat-report.json` and
`behavior-corpus-report.json` byte-identically to the committed evidence. The
external consumer fixture in `build-consumer/consumer` builds against it and
runs at exactly 60 and 600 native Draw callbacks.

### Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean, and
runs at exactly 60 and 600 native Draw callbacks against the pinned binary.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS, and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage, and most of XNA
remain unimplemented.

Nothing completed in Foundations 17–21 claims a runtime capability. The audio
descriptors open no device and create no XACT state; the effect and device
interfaces are declarations with no implementation; `PresentationParameters` is
a descriptor that creates, resets, enumerates, and presents nothing;
`TouchCollection` polls no panel; and `GameServiceContainer` is not reachable
from `Game`, which still exposes no `Services` property.

```text
FOUNDATION_MILESTONE_17_COMPLETE=true
FOUNDATION_MILESTONE_18_COMPLETE=true
FOUNDATION_MILESTONE_19_COMPLETE=true
FOUNDATION_MILESTONE_20_COMPLETE=true
FOUNDATION_MILESTONE_21_COMPLETE=true
PUSHED=false
SAFE_MANAGED_FRONTIER=EXHAUSTED
NEXT_STEP=PUBLIC_API_DECISION
```
