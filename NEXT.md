# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 29 are complete. Milestones 26 through 29 were
produced in one session as **local commits that have not been pushed**;
`develop` is ahead of `origin/develop` by the five milestone commits below plus
the docs commits that record them.

```text
START   HEAD = origin/develop = ec7d14cdcb92be21b2ffd7da16b2bc2d23474c7b
LAST SOURCE-BEARING COMMIT  = 8e87678
        origin/develop unchanged at ec7d14c
        worktree clean, git diff --check clean
        PUSHED = false
```

Every artifact hash below is taken at `8e87678`, the last source-bearing
commit; the docs commits that follow it change the artifact hash and nothing
else, so re-run the command in this file to get the current one.

| #  | commit    | milestone                                            | types | Go identities |
| -- | --------- | ---------------------------------------------------- | ----- | ------------- |
| 26 | `8624662` | BCL base-class composition; GameComponentCollection  | 1     | 21            |
| 26 | `bdb747a` | the collection proved from outside the repository    | 0     | 0             |
| 27 | `612aebc` | ReadOnlyCollection<T> signature adapter; VisualizationData | 1 | 3         |
| 28 | `62be078` | IGraphicsDeviceService; frontier re-derived          | 1     | 9             |
| 29 | `8e87678` | the System.Exception audit; every deferral measured  | 0     | 0             |

Three types, 33 mapped Go identities. Two of the five commits deliberately
complete no type: they are proof and measurement.

## Scoreboard

```text                        start    final
TARGET_TYPES                   118      121
TARGET_MEMBERS                1722     1755
TOTAL_DIAGNOSTICS              316      313
MISSING_TYPE                   139      136
MISSING_MEMBER                 177      177
COMPLETE_TYPES                 113      116
PARTIAL_TYPES                    5        5

REFERENCE_XNA_MEMBERS         2964     2964
BCL_INHERITED_PUBLIC_MEMBERS     0       11
BCL_INHERITED_MEMBER_PROJECTIONS 0       12
EXPECTED_GO_MEMBERS           3243     3255

BCL_BASE_ADAPTERS                0        1
BCL_BASE_ADAPTER_CONSUMERS       0        1
BCL_SIGNATURE_ADAPTERS           0        1
BCL_SIGNATURE_ADAPTER_CARRIERS   0        6
BCL_DEFERRED_BASE_BLOCKERS       0       21

INTERFACE_WITNESS_PROJECTIONS   25       25
mutation inventory             447      478
behavior corpus                575      598
external canary tests            7       16
```

Every mismatch, leak, allowlist and unmeasured counter is zero throughout. The
five protected partial runtime types are untouched: `MISSING_MEMBER` stayed at
177 on purpose.

## The BCL base-class composition architecture

> A CLR class that inherits a supported BCL collection base projects as a
> concrete Go reference type that **contains a private generic adapter** for
> that base and re-exposes the base's **public** surface through measured
> forwarding members.

Two rules follow:

- **A public member inherited from a supported BCL base is still public CLR
  surface**, so it must not disappear because the XNA metadata does not declare
  it. Projecting only the seven members `GameComponentCollection` declares would
  have produced a collection nothing can be added to.
- **The adapter is implementation machinery.** Not an XNA type, not an exported
  field, not a public base-class object, not an embedded public API, not a
  handle.

```go
type GameComponentCollection struct {
    base collectionBase[IGameComponent]   // unexported, not embedded
    ...
}
```

Exported embedding is rejected by the verifier, and so is embedding the
*unexported* adapter: promotion would publish forwarding nobody measured. The
base's four protected virtuals are an **unexported** Go interface, so only a
type declared in this module can supply or reach a hook, and every mutating
public operation routes through it.

**A second, independent role** exists for the same family. A BCL type the
contract carries at a **public signature position** needs a public Go spelling —
the `System.TimeSpan` and `System.EventHandler<T>` footing —
so `ReadOnlyCollection<T>` is `*framework.ReadOnlyCollection[T]`. It is
`SUPPORTED` as a signature adapter and `DEFERRED` as a base, and neither implies
the other.

### Identity accounting

Three provenance classes, kept distinct. **`REFERENCE_MEMBERS` is not
falsified**: it still names exactly what Microsoft declares, and the pinned 3243
XNA-declared projection count did not move. Every expected Go member has exactly
one provenance class, and a test partitions the whole surface to prove the two
halves are disjoint and exhaustive.

## Reference authority

The six XNA assemblies are unchanged and re-verified by hash. **New this
session**, and the authority for every inherited behavior:

```text
mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
              5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
              ~/.wine-cna-xna40/drive_c/windows/Microsoft.NET/Framework/v4.0.30319/
```

It is the right binary and not merely a plausible one: every pinned XNA assembly
declares `.assembly extern mscorlib 4.0.0.0` with public key token
`b77a5c561934e089`, which is that identity, and it is an implementation assembly
so `Collection<T>`, `List<T>`, `Dictionary<K,V>` and `ReadOnlyCollection<T>`
carry real IL. Modern .NET was not consulted; Mono's mscorlib was not consulted.

## Reference quirks now preserved — do not "correct" them

Everything in the Foundation 17-25 list still holds, plus:

- **`GameComponentCollection` mutates before it announces on Insert and Remove,
  and announces the whole collection before it mutates on Clear.** A failing
  handler therefore leaves an `Add` applied and a `Clear` unapplied.
- **`ClearItems` has no null check**, unlike `RemoveItem`, so it announces a nil
  element with a nil `GameComponent`. It also re-reads `base.Count` every
  iteration, so a handler that adds a component extends the loop.
- `Collection<T>.Insert` guards `index > Count` where `set_Item` and `RemoveAt`
  guard `index >= Count`; `set_Item` validates its index *before* reaching a
  `SetItem` that never succeeds.
- **`List<T>.Clear` increments `_version` unconditionally**, so clearing an
  already empty collection still invalidates live enumerators.
- `List<T>.Enumerator.MoveNext` checks the version **before** the bounds, so a
  mutation is reported even past the end.
- **An array-backed `ReadOnlyCollection<T>` is NOT version-checked.**
  `SZGenericArrayEnumerator<T>` holds only `_array`, `_index`, `_endIndex`.
- **`System.Single::Equals` returns true for two NaNs**, where Go's `==` is
  false, so a NaN search finds a NaN element.
- `ReadOnlyCollection<T>` stores the list rather than copying it, and is bound
  to the list *instance*: a captured Go slice header is exact where a `*[]T`
  would be too live and a copy too dead.
- `GraphicsDeviceManager::get_GraphicsDevice` is one `ldfld`, which is why
  `IGraphicsDeviceService` is infallible while `IGraphicsDeviceManager`, on the
  same class, is not.

## The frontier — measured, not summarised

Eighteen missing types are dependency-complete and every one is behavior
blocked. Three that the type-level graph calls reachable were shown from IL not
to be, each named to the exact member:

- **`GameComponent`** → blocked on `Game::Components`. Its `Dispose(bool)` runs
  `get_Game().get_Components().Remove(this)`, and `Components` is one of the 39
  missing members of the protected partial `Game`. The gap is now very small:
  `Components` returns a `GameComponentCollection`, which is **complete**. What
  stands between CNA-Go and a component loop is no longer a mapping question but
  the runtime decision about `Game`. Blocks `DrawableGameComponent` and
  `GamerServicesComponent`.
- **`GraphicsResource`** → blocked on native ownership three ways: `public
  abstract` with an `assembly` constructor, `.field assembly uint64
  _internalHandle`, and a `Dispose(bool)` that dispatches to the C++/CLI
  `~GraphicsResource`/`!GraphicsResource` pair. It alone blocks seven types.
- **`Microphone`** → became dependency-complete only because milestone 27
  unblocked `ReadOnlyCollection<Microphone>`. The adapter unblocked its
  signature, not its behavior; it stays device-blocked.

Every deferred base now names its blockers — 21 across seven bases, each
`SUBSYSTEM` (an inherited member's type belongs to an unmapped .NET subsystem)
or `ARCHITECTURE` (a cross-cutting decision no single member carries). The
verifier **fails a deferred base that records nothing**.

```text
IMPLIED   System.Object / ValueType / Enum
MAPPED    System.EventArgs
COMPOSED  Collection`1                        1 consumer, adds 12 identities
DEFERRED  Dictionary`2                        1 derived,  6 blockers
DEFERRED  ReadOnlyCollection`1 (base role)    4 derived,  2 blockers
DEFERRED  System.Exception                    5 derived,  5 blockers
DEFERRED  ExternalException                   3 derived,  1 blocker
DEFERRED  System.Attribute                    5 derived,  3 blockers
DEFERRED  ExpandableObjectConverter           1 derived,  3 blockers
DEFERRED  System.IO.BinaryReader              1 derived,  1 blocker
```

### `Dictionary<K,V>` — blocked on surface, not behavior

The IL is readable. Six public members cannot be projected in already-decided
terms: `GetObjectData` and `OnDeserialization` are declared **public, not
explicit**, so `LaunchParameters` genuinely exposes them and they need
`System.Runtime.Serialization`; `Keys`/`Values` return public nested collections
with their own surfaces; `Comparer` needs `IEqualityComparer<TKey>`;
`GetEnumerator` needs the nested `Enumerator` over `KeyValuePair<K,V>`.
Projecting a subset would make `LaunchParameters` a partial type, which is not a
BCL-mapping outcome. It is the only consumer.

### `System.Exception` — the audited decision

All eight derived types declare **only constructors**, so this is the
`GameComponentCollection` shape again and a partial projection is not an escape.
Three inherited members are blocked on three distinct subsystems
(`IDictionary`, `MethodBase`, `Serialization`), and two architecture obstacles
are the material ones:

1. **Is an XNA exception type a Go `error`?** If yes, every fallible operation's
   error contract changes from an opaque error to a possibly typed CLR
   exception, reopening ~30 individually evidenced fallibility decisions
   together and making four unexported sentinel error families public API. If
   no, the eight types are inert objects nothing constructs, returns, or
   catches — worse than the collection nothing could be added to.
2. **`StackTrace`** is captured at throw time; a constructed Go value has no
   throw site.

`System.Attribute` was audited too and is not easier: Go has no attribute
metadata, so the five types would be inert; and the inherited statics raise a
question that has never arisen — are inherited **statics** part of a derived
type's projected surface at all?

## Re-running gates

```sh
export PATH=~/deps/go1.24.4/bin:$PATH
export CNA_NATIVE_LIBRARY=~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so

gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~260s under -race
go build ./... && go build -trimpath ./...
go run ./tools/api_compat --mode strict -report "" -missing ""   # expected red: 313 deferred
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

At `8e87678` that yields sha256
`546ef5098bebd38d98e502f562b5087c13d8a55c1eaaa5f0d9f730278b04ddc1` over 264
entries, reproduced twice in the same run. Extracted into
`build-consumer/isolated`, it passes every gate with no development-checkout
dependency and regenerates `api-compat-report.json`,
`behavior-corpus-report.json` and `missing-type-inventory.md` byte-identically.
The Foundation-1 consumer fixture in `build-consumer/consumer` builds against it
and runs at exactly 60 and 600 native Draw callbacks.

The external canary now runs **16** tests and is mandatory for both the event
architecture and the collection projection:

```sh
go run ./tools/external_consumer -source build-consumer/isolated/cna-go
```

Its `CollectionProbe` records what each handler observed `Count` to be at the
moment it ran, so an outside caller can tell "mutate then announce" from
"announce then mutate" **without seeing any implementation** — which is the
sharpest available proof that the projection is faithful.

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

## Maintained template

`cna-go-template` is unchanged, on `develop` at `6525484`, worktree clean.
`SOURCE_CHANGED=NO`.

## Qualification artifact caveats — unchanged

CNA ABI 0.7.0, HEADLESS renderer, NULL audio. Native draw execution is proven;
visible rendering is not. Windows, macOS, Android, iOS and Web/Wasm are not
qualified. Content/XNB, Effects/3D, Audio, Media, Storage and most of XNA remain
unimplemented.

Nothing completed in Foundations 26-29 claims a runtime capability.
`GameComponentCollection` is usable by an external consumer and is used by
nothing: CNA-Go has no `GameComponent` and `Game` exposes no `Components`.
`VisualizationData` starts no playback — there is no media backend, so both its
buffers stay 256 zeros. `IGraphicsDeviceService` publishes no device.

```text
FOUNDATION_MILESTONE_26_COMPLETE=true
FOUNDATION_MILESTONE_27_COMPLETE=true
FOUNDATION_MILESTONE_28_COMPLETE=true
FOUNDATION_MILESTONE_29_COMPLETE=true
PUSHED=false
SAFE_MANAGED_FRONTIER=EXHAUSTED
NEXT_STEP=RUNTIME_DECISION_GAME_COMPONENTS_OR_PUBLIC_API_DECISION_SYSTEM_EXCEPTION
```
