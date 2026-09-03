# Foundation 74 — the Dictionary base, `LaunchParameters`, and a roadmap that cannot go stale

This milestone composes the profile's second BCL base class, projects the one
XNA type that derives from it, closes one of `Game`'s two missing members, and
replaces ROADMAP.md's hand-typed counters with a generated, guarded scoreboard.

```text
COMPLETE_TYPES   154 -> 155        MISSING_TYPE     101 -> 100
PARTIAL_TYPES      2 ->   2        MISSING_MEMBER     8 ->   7
EXPECTED_GO_MEMBERS  3456 -> 3470  BCL_BASE_ADAPTERS  1 ->   2
GLOBAL_ACTIONABLE_LOCAL  100       GLOBAL_UNREVIEWED    0
```

## Reference authority

```text
Microsoft.Xna.Framework.Game.dll
  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
  5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
```

Both read with `ikdasm`. Nothing below is derived from CNA, from MonoGame, or
from how a modern .NET `Dictionary<K,V>` behaves.

## What Foundation 29 blocked, and what actually blocked it

Foundation 29 recorded six blockers on `System.Collections.Generic.Dictionary`2`.
Re-measuring them one at a time is what this milestone did first, and five of
the six turned out to be **missing Go spellings** rather than missing
subsystems — and a spelling is something the settled signature-adapter rule
already knows how to supply.

| blocker | what it actually needed | outcome |
| --- | --- | --- |
| `Keys` | a Go spelling of `Dictionary`2/KeyCollection` | signature adapter, 3 public members |
| `Values` | `Dictionary`2/ValueCollection` | signature adapter, 3 public members |
| `Comparer` | `IEqualityComparer<T>` | signature adapter, 2 abstract members |
| `GetEnumerator` | the nested `Enumerator` and `KeyValuePair<K,V>` | the settled `List<T>.Enumerator` rule plus one 3-member adapter |
| `OnDeserialization` | *nothing* — see below | projected |
| `GetObjectData` | `System.Runtime.Serialization` | **still blocked, and now measured** |

### `OnDeserialization` needed no subsystem at all

Its body opens

```
IL_0000: ldarg.0
IL_0001: ldfld  SerializationInfo Dictionary`2::m_siInfo
IL_0006: brtrue.s IL_0009
IL_0008: ret
```

`m_siInfo` has exactly **one** non-null writer in the whole pinned type: the
`family` constructor `.ctor(SerializationInfo, StreamingContext)` at IL offset
`IL_0008`. The CLR does not inherit constructors, `LaunchParameters` declares no
serialization constructor of its own, and CNA-Go deserialises nothing — so no
state any consumer can reach has `m_siInfo` set, and the empty branch is the
only reachable body. The member is projected as the empty body it *provably*
is. That is not a simplification; the alternative would have been to leave a
genuinely public member missing on the strength of a path nothing can enter.

### `GetObjectData` is blocked, and the blocker is now a number

The settled signature-adapter rule pins an adapter to the **exact** public
member inventory of the CLR type it spells. So projecting the
`SerializationInfo` parameter means projecting all of `SerializationInfo`, and
that reaches:

```text
System.Runtime.Serialization.SerializationInfo          43 public members
System.Decimal                                          89
System.DateTime                                         91
System.Runtime.Serialization.StreamingContext            6
System.Runtime.Serialization.SerializationInfoEnumerator 6
System.Runtime.Serialization.SerializationEntry          3
plus System.Runtime.Serialization.IFormatterConverter and System.TypeCode
```

`System.Decimal` and `System.DateTime` are named **nowhere else** in the XNA 4.0
Windows runtime profile. Projecting 238 public BCL members, two of them
128-bit-decimal and calendar arithmetic with their own conformance risk, to
satisfy one inherited method no XNA consumer calls, *is* the mscorlib port the
profile is defined to exclude.

So the member is recorded as the adapter's one
`BCL_PROJECTION_BLOCKED_EXTERNAL` exclusion. That is a new **kind** of
exclusion, and the distinction is the point:

- `NOT_PUBLIC_SURFACE` — every exclusion up to Foundation 73 — says the member
  is not part of what a derived CLR caller can reach at all. **Nothing is
  missing.**
- `BCL_PROJECTION_BLOCKED_EXTERNAL` says the member **is** public CLR surface,
  **is** absent from the Go projection, and names the exact external closure
  that would have to be projected first.

The verifier counts them separately (`BCL_ADAPTER_EXCLUSIONS_BLOCKED`) and
fails any blocked exclusion that names no closure, which is the same rule
Foundation 29 imposed on deferred BCL bases. An admission of a hole is not a
permission to have one.

Two further exclusions are `LANGUAGE_MAPPING_LIMITATION`: `KeyCollection` and
`ValueCollection` each declare a **public** `.ctor(Dictionary<K,V>)` whose only
parameter type is the base the composition rule deliberately keeps unexported.
No exported Go function can take one. The profile's only producer of either view
is `get_Keys`/`get_Values`, which hand back the cached instance.

## Why the adapter is not a Go map

`Dictionary<K,V>` enumerates its **entries array** from index 0, skipping slots
whose `hashCode` is negative. `Remove` pushes the freed slot onto the **head**
of the free list and `Insert` takes it back, so:

```text
add alpha, beta, gamma, delta   ->  alpha beta gamma delta
remove beta                     ->  alpha gamma delta
add epsilon                     ->  alpha EPSILON gamma delta
```

Epsilon lands in beta's old slot and enumerates **second**. A Go map randomises
iteration order; a slice appends. Either would be wrong at the first `foreach`,
not merely unidiomatic. That is why `dictionaryBase[TKey, TValue]` reproduces
`buckets`, `entries`, `count`, `freeList`, `freeCount` and `version`, and why
`type LaunchParameters map[string]string` is refused by
`verifyBCLBaseComposition` rather than merely discouraged.

Three more behaviours are reproduced rather than harmonised:

- **`Clear` on an empty dictionary does not bump `version`.** Its whole body,
  including `version++`, is guarded by `count > 0`. `List<T>.Clear` falls
  through to `_version++` unconditionally. The two BCL collections genuinely
  disagree, and a live enumerator survives one and not the other.
- **`MoveNext` compares the version BEFORE the bounds test**, so a dictionary
  mutated after enumeration ran off the end still fails a further step.
- **`Keys` and `Values` are cached and live.** Each is allocated once, stored in
  a private field and returned forever after, and it reads *through* to the
  dictionary rather than copying.

## The null-key guard is statically dead

`Insert`, `FindEntry` and `Remove` each open with
`box !TKey; brfalse; ThrowHelper.ThrowArgumentNullException`. The profile's only
consumer is `LaunchParameters`, whose base is `Dictionary<string,string>`, and
CNA-Go maps `System.String` to Go `string`, which has no null. The branch is
unreachable for every key any consumer can supply.

This is the same shape as `Collection<T>`'s dead `items.IsReadOnly` guard, and
it gets the same treatment: **not projected as a failure mode**, and written
down so a future consumer over a nullable key type reopens it rather than
inherits the omission silently. The consequence is measurable —
`ContainsKey`, `Remove`, `TryGetValue` and the indexer *setter* are infallible,
while `get_Item` (KeyNotFound) and `Add` (duplicate) are the profile's only two
fallible inherited operations.

## `String::GetHashCode`, reproduced

`Comparer` returns `EqualityComparer<string>.Default`, which is
`GenericEqualityComparer<string>` because `System.String` implements
`IEquatable<string>`. Its `GetHashCode` forwards to `String::GetHashCode`, whose
pinned body is

```text
hash1 = hash2 = 0x15051505
pint  = (int*)chars                 // two UTF-16 units per int32
len   = this.Length                 // in CHARS
while (len > 0) {
    hash1 = ((hash1 << 5) + hash1 + (hash1 >> 27)) ^ pint[0];
    if (len <= 2) break;
    hash2 = ((hash2 << 5) + hash2 + (hash2 >> 27)) ^ pint[1];
    pint += 2; len -= 4;
}
return hash1 + (hash2 * 0x5d588b65);
```

Three details are load-bearing. The unit is a **UTF-16 code unit**, so a Go
string is converted rather than hashed as bytes; the `int32` load is
**little-endian**, which is what the pinned Windows x86/x64 profile is; and a
CLR string is **null-terminated in memory**, so the loop legitimately reads one
zero unit past the last character.

This mscorlib has **no randomised-hashing branch at all** — reading the binary
is what settles that. Ten values are pinned in
`TestStringHashCodeMatchesThePinnedAlgorithm`, computed from the IL listing
rather than from this implementation, and the lengths exercise every exit of the
loop. Two of them are non-ASCII, so a byte-wise implementation fails.

## `LaunchParameters`

The class declares exactly one public member. Its constructor is

```text
call Dictionary`2<string,string>::.ctor()
ParseCommandLineArguments(Environment.GetCommandLineArgs())
```

and `ParseCommandLineArguments` is `assembly`, so it is not projected surface —
but everything it does is observable through the dictionary it fills:

```text
separators = { '/', '-' }
if (args.Length <= 1) return;
for (i = 1; i < args.Length; i++) {
    argument = args[i].TrimStart(separators);
    ParseKeyValuePair(argument, out key, out value);   // splits at the FIRST ':'
    if (!ContainsKey(key) && key != string.Empty) Add(key, value);
}
```

Four consequences are pinned by table-driven tests: `args[0]` is skipped; a
*run* of leading separators is trimmed, so `--x` and `/-/x` both key `x`; the
first colon splits, so `/path:C:/games/x` keys `path` with value `C:/games/x`;
and a duplicate keeps the **first** occurrence because `ContainsKey` is tested
before `Add`, while an empty key is dropped entirely.

`Game::get_LaunchParameters` is one `ldfld` of an object the constructor
allocates immediately after `EnsureHost()`. The identity is therefore fixed for
the Game's life, and XNA hands out the **instance**, not a copy: a consumer's
`Add` is visible through `game.LaunchParameters()` afterwards. Both halves are
tested, inside and from the external consumer.

## The `TryGetValue` result order

`TryGetValue` is the profile's **first** member with both a non-void return and
an `out` parameter. Every earlier one returns `void`, where "append the out
parameter" and "prepend it" are indistinguishable. The settled direction rule is
`remove-input-and-append-return`, so the projection is

```go
func (l *LaunchParameters) TryGetValue(key string) (bool, string)
```

— the declared `bool` first, the out parameter after it. It is deliberately not
reshuffled into Go's value-then-ok idiom, which would make one member disagree
with the rule every other member follows.

## The frontier, and a ROADMAP that cannot go stale

ROADMAP.md used to carry its scoreboard and its per-family table by hand, and a
hand-typed count is wrong the moment the next milestone lands.

`tools/api_compat/frontier.go` now holds a registry that **partitions the live
missing-type set**. Every missing type belongs to exactly one family; every
family carries a classification and, unless it is `ACTIONABLE_LOCAL`, a named
blocker; a family that claims a type the verifier no longer reports missing, or
that has emptied out, is a verifier failure. `docs/generated/remaining-work.md`
is generated from it, and two new counters fall out:

```text
GLOBAL_ACTIONABLE_LOCAL   100
GLOBAL_UNREVIEWED           0
```

`GLOBAL_UNREVIEWED` is never written by hand — the exhaustiveness check assigns
it — so it can only ever be nonzero because something was forgotten.

ROADMAP.md's scoreboard is now one marked block of `KEY VALUE` lines checked
against `api-compat-report.json` and `native-abi-report.json` by
`TestRoadmapScoreboardMatchesTheGeneratedReports`. The guard is proved by
`TestRoadmapStalenessGuardRejectsAStaleNumber`, which plants three stale values
and one invented counter and requires each to be caught.

The guard is deliberately narrow: it looks at ROADMAP.md and nothing else.
`docs/foundation-NN-*.md` quote the scoreboard of the milestone they record, and
those quotes are correct *for that milestone*. Treating historical evidence as
stale would be a category error.

## Falsifiability

`bcl_dictionary_mutation_test.go` builds four deliberately wrong versions and
requires the assertion the real code passes to reject each:

| mutant | killed by |
| --- | --- |
| append-only backing store (what a Go map or a slice gives) | the pinned `alpha, epsilon, gamma, delta` enumeration order |
| last-wins duplicate handling in the parser | `/k:first /k:second` keeps `first` |
| an enumerator that does not capture the version | a step after a mutation must report the enumeration failure |
| a `LaunchParameters` getter that copies | a consumer's `Add` must reach the Game's own field |

## Qualification

```text
go test ./...                        PASS
go run ./tools/behavior              OBSERVATIONS=737 ASSERTIONS=737 FAILURES=0
go run ./tools/api_compat            TOTAL_DIAGNOSTICS=107, all of them
                                     MISSING_TYPE/MISSING_MEMBER; every mismatch,
                                     leak, unexpected and allowlist category 0
go run ./tools/native_abi ...        BOUND_FUNCTIONS=230 ABI_MISMATCHES=0
                                     MISSING_LIBRARY_SYMBOLS=0 LOADED_ABI=0.21.0
go run ./tools/external_consumer     TESTS=99 FAILURES=0 STATUS=PASS
```

The ABI is untouched: this milestone binds no route, because nothing in it
reaches CNA. `LaunchParameters` is pure managed — its constructor reads
`os.Args` and parses strings — and that is why it is admitted to
`pureManagedTypes` rather than given a native half it does not have.

Twenty-eight new tests in the framework package and two in the external
consumer canary, none skipped.
