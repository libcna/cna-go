# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 25 are complete. Milestones 22 through 25 were
produced in one session as **four local commits that have not been pushed**;
`develop` is 4 ahead of `origin/develop`.

```text
START   HEAD = origin/develop = 81d8fe7cd08f4303ff49ebfca98441ceaa59b9bc
FINAL   HEAD = 6259bfa
        origin/develop unchanged at 81d8fe7
        worktree clean, git diff --check clean
```

| #  | commit    | milestone                                        | types | Go identities |
| -- | --------- | ------------------------------------------------ | ----- | ------------- |
| 22 | `b8698d0` | the general CLR event architecture               | 0     | 0             |
| 23 | `f169408` | IUpdateable, IDrawable, the EventArgs carriers   | 5     | 19            |
| 24 | `d2068e0` | `System.IDisposable` as a zero-surface relation  | 0     | 0             |
| 25 | `6259bfa` | internal XNA interfaces, frontier closure        | 0     | 0             |

Five types, 19 mapped Go identities. Three of the four milestones deliberately
complete no type: they are mapping decisions and measurement.

## Scoreboard

```text                        start    final
TARGET_TYPES                   113      118
TARGET_MEMBERS                1703     1722
TOTAL_DIAGNOSTICS              321      316
MISSING_TYPE                   144      139
MISSING_MEMBER                 177      177
COMPLETE_TYPES                 108      113
PARTIAL_TYPES                    5        5
MISSING_TYPES                  144      139

INTERFACE_WITNESS_PROJECTIONS   25       25
PACKFROMVECTOR4_WITNESSES       17       17
TOVECTOR4_WITNESSES              8        8

mutation inventory             347      447
behavior corpus                564      575
```

Every mismatch, leak, allowlist, and unmeasured counter is zero throughout,
including the now load-bearing `EVENT_MAPPING_MISMATCH` and
`BASE_MAPPING_MISMATCH`. The five protected partial runtime types are untouched:
Game 39, GraphicsDeviceManager 40, GraphicsDevice 70, SpriteBatch 16,
Texture2D 12, combined 177. `MISSING_MEMBER` stayed at 177 on purpose: none of
them gained an event, because an event that never fires is not implemented.

## The event architecture

Every one of the 49 public CLR events in the profile is
`System.EventHandler`1<T>` over one of five generic arguments, so two BCL shapes
close the whole surface:

```text
System.EventArgs        -> *framework.EventArgs
System.EventHandler<T>  -> framework.EventHandler[T]   func(sender any, args T) error
```

The generic argument is carried exactly; the old undeclared `mapType`
fall-through to `any` is gone, and degrading a handler is now a named event
defect. The handler's `error` is a Go projection of the CLR exception channel,
not an XNA return identity.

The settled accessor projection is unchanged — one CLR event still becomes
exactly two Go accessors. Four language adapters live in the framework package:
`EventArgs`, `EventHandler[T]`, `EventSubscription`, `EventSource[T]`.

Semantics read from the reference accessors rather than invented:

| property | behavior | source |
| -------- | -------- | ------ |
| `Add(nil)` | registers nothing, returns the zero token | `Delegate.Combine(x, null) -> x` |
| absent-token removal | harmless: zero, already-removed, foreign | `Delegate.Remove` of an absent delegate |
| ordering | registration order | |
| dispatch | over a snapshot taken under the lock | |
| mutation during raise | affects later raises only | |
| lock scope | no internal lock held while a handler runs | |
| failure | first error propagates, no later handler runs, list intact | |
| token lifetime | explicit removal only; no finalizer | |

**The one deliberate divergence**, recorded as a Go language projection: Go func
values are not comparable, so a token names the **registration**, not the
handler. Adding one handler twice yields two registrations and two tokens, where
CLR `Delegate.Remove` matches by delegate identity.

`EventArgs` carries one unexported byte. Measured on go1.24.4 linux/amd64, four
separate `new(struct{})` heap allocations all returned `runtime.zerobase` — 6
collisions out of 6 pairs — so a zero-size `EventArgs` would make every instance
pointer-equal and destroy the identity `Empty` depends on.

## Relationship rules this session settled

These are general rules, declared in `tools/api_compat/mapping-rules.json` and
documented in `docs/xna-go-mapping.md`.

1. **A non-XNA CLR base is a measured relationship, never Go embedding.** Go has
   no CLR inheritance and embedding would promote members the contract never
   declared. The table is exhaustive over the profile: 3 implied roots,
   `System.EventArgs` mapped, 8 deferred. A **deferred** base means no derived
   type may be projected, and projecting one is a diagnostic.

2. **A non-XNA CLR interface contributes no projected Go surface.** Either the
   XNA type already declares the members publicly, or it implements the
   interface explicitly so the member is not public surface. All 8 are declared
   and measured with `projectedMembers == 0`.

3. **`System.IDisposable` in particular creates nothing**: no `Disposable` type,
   no `Close` alias, no `io.Closer`, no finalizer, no ownership wrapper, and no
   `Dispose` synthesized from ancestry. Mapping it makes a dependency
   syntactically complete without deciding native ownership for anything.

4. **An internal XNA interface is declared, not skipped.** `IGraphicsResource`
   and `IDynamicGraphicsResource` are `.class interface private`, so they have
   no public member to project at all.

5. **A CLR class may derive from any MAPPED BCL base.** The class closure used
   to hardcode `System.Object`; it now records which relationship carried the
   base (`DIRECT` or the adapter name), and `UNDECIDED` fails.

6. **An interface's boundary is read per contract, not per class.**
   `IUpdateable` and `IDrawable` are infallible because their implementors'
   getters are one `ldfld` and their `Update`/`Draw` are a bare `ret`, while
   `IGameComponent` on the same class stays fallible because its `Initialize`
   throws. Event accessors carry an error from the accessor projection, counted
   separately from boundary operations.

## Reference quirks now preserved — do not "correct" them

Everything in the Foundation 17-21 list still holds, plus:

- `GameComponent::set_Enabled` and `set_UpdateOrder` compare first and raise
  **only when the value actually changes**.
- Every XNA raise site pushes the one shared `System.EventArgs::Empty` static
  field, so shared object identity is the faithful projection.
- `GraphicsDeviceManager` implements `System.IDisposable` **explicitly**
  (`private ... .override`), so its `Dispose()` is not public surface; its only
  projected `Dispose` is the `Dispose(bool)` the contract declares.
- `ResourceCreatedEventArgs` and `ResourceDestroyedEventArgs` declare
  `assembly` constructors, so they get **no** Go constructor.
  `ResourceDestroyedEventArgs`'s constructor stores its tag before its name.

## Reference authority

Unchanged, and all six re-verified by hash this session.

```text
Microsoft.Xna.Framework.dll             38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
Microsoft.Xna.Framework.Graphics.dll    560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.Game.dll        b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
Microsoft.Xna.Framework.Input.Touch.dll b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25
Microsoft.Xna.Framework.Xact.dll        a14d5364dca7cf49fb90639e87ba04d52b59a700dc9198efa5707ce8eae28f0a
Microsoft.Xna.Framework.Video.dll       17538b1ca9d48a993e2cd88c96b436df08e7abb4aec5d4758eb21feb580d6e06
```

Public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## Native provenance — unchanged

CNA was **not rebuilt**. The exact Foundation-11 pinned binary reproduces both
committed native reports.

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED          23/67/96/28/2/5, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED     native_stress reproduces every counter byte-identically
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED
```

`native_abi` reproduces the committed report key for key, differing only in
`header_root`, which the committed evidence stores normalized.

## The frontier — exhausted again, and the next step is a decision

Fourteen missing types are dependency-complete. **All fourteen are blocked on
behavior, not on a mapping**, and each was re-derived rather than assumed:

| type | why |
| ---- | --- |
| `Graphics.DisplayMode` | `assembly` ctor; display enumeration |
| `Input.GamePadCapabilities` | `assembly`/`private` ctor; device capability |
| `Input.Touch.TouchPanelCapabilities` | no constructor at all; device capability |
| `Input.Mouse` | device state plus `MouseMessageHooker` |
| `Audio.RendererDetail` | `assembly` ctor; XACT renderer enumeration |
| `Media.MediaSource` | `assembly` ctor; media backend |
| `GameWindow` | **new**: `public abstract`, `assembly` ctor, 8 bodyless public abstract members, real platform `Handle` |
| `Audio.AudioCategory` | `assembly` ctor needing `AudioEngine`; every method P/Invokes |
| `Audio.Cue` | XACT native |
| `Audio.SoundEffectInstance` | `assembly` ctors; 18 native throw sites |
| `Graphics.EffectAnnotation` | `assembly` ctor; unmanaged `calli` |
| `Media.Video` | `assembly` ctor needing `GraphicsDevice` and a content file |
| `FrameworkDispatcher` | drives the microphone, dynamic-audio, media and storage pumps |
| `TitleContainer` | needs `TitleLocation.Path` and the file system |

Six BCL shapes remain. For each, the types whose **only** remaining blocker it
is:

```text
5  System.Attribute            the five ContentSerializer* attributes
4  System.Exception            NoMicrophoneConnectedException, DeviceLostException,
                               DeviceNotResetException, NoSuitableGraphicsDeviceException
2  ReadOnlyCollection`1        Audio.Microphone, Media.VisualizationData
1  Collection`1                GameComponentCollection
1  Dictionary`2                LaunchParameters
1  System.Action`1             Content.ContentManager
```

`Microphone` and `ContentManager` stay device- and filesystem-blocked even with
their shape mapped.

### The next decision: how a BCL collection base projects

This is the highest-value one, and it is a **material public-API choice that
existing policy does not resolve**.

`GameComponentCollection` was reconsidered as directed — its prerequisites
(`IGameComponent`, the event projection, `GameComponentCollectionEventArgs`) are
all now complete — and re-derived in full. Its behavior **is** entirely managed
and IL-provable:

```text
InsertItem   IndexOf(item) != -1 -> ArgumentException; base.InsertItem;
             then raise ComponentAdded only if item is non-null, with a FRESH
             GameComponentCollectionEventArgs
RemoveItem   read base[index]; base.RemoveItem; then raise ComponentRemoved
             only if the removed item was non-null
SetItem      unconditionally throws NotSupportedException
ClearItems   raises ComponentRemoved for EVERY item, index 0 upward, and only
             THEN calls base.ClearItems
```

Note the asymmetry: `Insert`/`Remove` mutate **before** they raise; `Clear`
raises for the whole collection **before** it mutates. Do not guess this.

It is blocked anyway, for a projection reason. Its declared surface is seven
members — a constructor, four **protected** overrides, and two events. `Add`,
`Remove`, `Clear`, `Count`, the indexer, `IndexOf`, `Insert`, `RemoveAt`, and
`GetEnumerator` are all inherited from `Collection<IGameComponent>` and are not
declared members. Projecting only the seven would turn the strict gate green
while producing a collection **nothing can be added to**.

`LaunchParameters` is the same decision in starker form: it derives from
`Dictionary<string,string>` and its only declared member is a constructor.

Alternatives, recorded rather than chosen:

1. **Re-declare the inherited surface on the derived Go type.** Faithful for
   callers; needs a new expected-member category for inherited BCL surface, a
   scope rule for which inherited members count, and new `expectedCountFormula`
   arithmetic.
2. **Project only the declared members.** Faithful to the letter, useless, and
   leaves four protected overrides overriding nothing.
3. **A framework-package `Collection[T]` adapter** the derived type holds and
   re-exports. Still needs answer 1, and adds a public non-XNA support type.
4. **Defer** — what Foundation 25 does.

The same answer governs `ReadOnlyCollection<T>` (4 `Model*` types plus 2) and
`Dictionary<K,V>`.

### The other decision: CLR exception types

Untouched on purpose. `System.Exception` is the sole remaining blocker for four
XNA exception types, and `ExternalException` for three more. CNA-Go reports
failure through `error` results and has no exception hierarchy; both bases are
declared **DEFERRED** in the base relationship table, so no derived type may be
projected until the decision is taken.

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~250s under -race
go build ./... && go build -trimpath ./...
go run ./tools/api_compat --mode strict -report "" -missing ""   # expected red: 316 deferred
go run ./tools/api_compat --mode leak-only -report "" -missing ""
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -headers ~/deps/cna-c-abi-0.7.0/include \
    -library "$CNA_NATIVE_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
go run ./tools/api_compat --mode report     # run LAST so committed evidence keeps report mode
git diff --check
```

Pass `-report "" -missing ""` on any non-report run so committed evidence keeps
report mode, and send native reports to an explicit scratch `-output`.

### Deterministic artifact, isolated consumer, external canary

```sh
git ls-files -z | sort -z | tar --null --files-from=- \
  --transform 's,^,cna-go/,' --owner=0 --group=0 --numeric-owner \
  --mtime='@0' --mode='u+rw,go+r,go-w' --sort=name -cf - | gzip -n > OUT.tar.gz
```

At `6259bfa`, the last source-bearing commit, that yields sha256
`c64a2ab9e06800209e2f32a1e460fb8a56f529b377164f67e8032162c4447011` over 250
entries; at the final docs-only `685ec54` it yields
`c21280787fd0d2e33c4783a5e729cc11756e03233e08660676ca0e0d29f1ce2a` over the
same 250 entries. Both were reproduced independently in the same run, and the
delta between the two commits touches only `NEXT.md` and `plan.md`. Extracted into
`build-consumer/isolated`, it passes every gate with **no development-checkout
dependency** and regenerates `api-compat-report.json`,
`behavior-corpus-report.json`, and `missing-type-inventory.md` byte-identically
to the committed evidence. The Foundation-1 consumer fixture in
`build-consumer/consumer` builds against it and runs at exactly 60 and 600
native Draw callbacks.

The **external event conformance canary** is new and is mandatory for the event
architecture:

```sh
go run ./tools/external_consumer -source build-consumer/isolated/cna-go
```

It materialises `tools/external_consumer/testdata/eventcanary` as its own Go
module whose only requirement is the extracted artifact, with `GOWORK=off` and
no sibling checkout, and runs 7 tests proving an external type can satisfy
`IUpdateable` and `IDrawable`, own private `EventSource` fields, raise its own
events, and that a consumer holding only the contract has no way to raise.

## Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean, and
runs at exactly 60 and 600 native Draw callbacks against the pinned binary.
`SOURCE_CHANGED=NO`.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS, and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage, and most of XNA
remain unimplemented.

Nothing completed in Foundations 22-25 claims a runtime capability. The event
architecture raises no XNA event: CNA-Go has no `GameComponent`, no component
collection, and no component loop, so nothing calls `Update` or `Draw`.
`GraphicsDevice` remains a protected partial that raises nothing, so nothing
constructs either resource carrier. The three protected partials gained no event
member, deliberately.

```text
FOUNDATION_MILESTONE_22_COMPLETE=true
FOUNDATION_MILESTONE_23_COMPLETE=true
FOUNDATION_MILESTONE_24_COMPLETE=true
FOUNDATION_MILESTONE_25_COMPLETE=true
PUSHED=false
SAFE_MANAGED_FRONTIER=EXHAUSTED
NEXT_STEP=PUBLIC_API_DECISION_BCL_COLLECTION_BASE
```
