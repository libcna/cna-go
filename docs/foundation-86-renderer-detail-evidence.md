# Foundation 86 — RendererDetail, and the audio blocker that was stale

```text
COMPLETE_TYPES   184 -> 185        MISSING_TYPE       73 -> 72
BOUND_FUNCTIONS  305 -> 305        DELIBERATELY_UNBOUND_ROUTES  +0
```

## Reference authority

```text
Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
```

This is the **first type in the project whose authority is the Xact assembly**,
and it is worth saying why that does not make it XACT work. `RendererDetail` is
produced by `AudioEngine.RendererDetails`, which this project does not project;
but the TYPE is two strings and five members with no native dependency at all.
Projecting it needs no audio engine, no bank and no device.

## Two strings, five members, and a detail in every one

| member | what a plausible projection gets wrong |
| --- | --- |
| `FriendlyName` / `RendererId` | crossing them — both are one `ldfld` over adjacent string fields |
| `op_Equality` | comparing a joined string. The reference tests the two fields SEPARATELY and short-circuits, so `("ab","c")` and `("a","bc")` are unequal |
| `op_Inequality` | writing an independent body; the reference is 11 bytes of `!op_Equality` |
| `Equals(object)` | dropping the exact TYPE test. The reference compares two runtime `Type`s with `op_Inequality`, not `isinst`, so a boxed value of any other type answers false rather than throwing at the unbox |
| `GetHashCode` | using Go's hash, or dropping the `IsNullOrEmpty` guard that makes an EMPTY string contribute zero |
| `ToString` | reporting the fields. It boxes and calls `ValueType::ToString`, which answers `GetType().ToString()` and nothing else |

## The constructor is `assembly`, and the projection says so

```text
.ctor(string name, string id)   15 bytes, two stores
```

There is no public constructor, so a consumer never builds one — they receive it
from a collection this project does not yet project. The projection exposes the
zero value and the five public members, which is exactly what the pinned
contract declares. `default(RendererDetail)` is a legal, reachable CLR value, so
that is not a degenerate case invented here.

## The string hash moved, and why

`GetHashCode` XORs `String.GetHashCode()` of the two fields, and the pinned
mscorlib algorithm already lived in the framework package as an unexported
`stringHashCode`. Go has no way to share an unexported function across packages,
and exporting it would have added public surface the contract does not declare.

So it moved to **`internal/bclhash`**, which is `internal` — a consumer of the
module cannot reach it — and the framework package's `stringHashCode` is now a
one-line delegation. `TestStringHashCodeMatchesThePinnedAlgorithm` still passes
against the same body, which is what makes the move a move rather than a
rewrite.

## A blocker that was stale, measured rather than restated

`xnaBaseRelationships` recorded this against `SoundEffectInstance`:

> CNA-Go has no audio backend: the qualification artifact pins a NULL audio
> renderer, so nothing behind it would play

That has not been true for some time. `docs/generated/runtime-capabilities.md`
already carried the correction from a direct C probe, and this milestone
measured it again through CNA-Go's own loader:

```text
cna_audio_get_capabilities -> is_playback_available = TRUE
    on ~/deps/cna-c-abi-0.21.0            (HEADLESS renderer)
    on ~/deps/cna-c-abi-0.21.0-software   (SOFTWARE renderer)
```

Both the base relationship and the frontier family now say what was measured.
The audio family's remaining blocker is CNA-Go's own surface and nothing else.

### The capability route is NOT bound, and that is a decision

`cna_audio_get_capabilities` was bound end to end during this milestone and then
**reverted**, because reading the reference settled the question against it:

```text
Helpers::GetExceptionFromErrorCode(int32)
    ...
    IL_0202:  newobj instance void NoAudioHardwareException::.ctor()
```

XNA does not probe for audio hardware. It makes the native call and MAPS the
returned error code, and `NoAudioHardwareException` comes out of that switch. So
a projection that probed up front would perform a call the reference never
makes, and the route would have been bound with no faithful call site — exactly
what the standing rule forbids. The measurement is kept; the binding is not.

`cna_sound_effect_create_pcm16` answers `CNA_RESULT_NOT_SUPPORTED` "when no
audio device is available", which is the same question asked at the moment the
reference asks it. That is where the milestone projecting `SoundEffect` should
put it.

## Planted defects

13 distinct defects, each a real way to get this wrong and each compiling.

```text
PLANTED   13      KILLED   13      SKIPPED   0
```

Two needed a test the first pass did not have, and both are the same lesson —
**a row that agrees with the mutant proves nothing**:

| defect | why the first assertion missed it |
| --- | --- |
| `equality_compares_the_concatenation` | the "fields swapped" row compares `("a","b")` with `("b","a")`, whose concatenations are `"ab"` and `"ba"` — the mutant answers false there too. A row whose concatenations MATCH while the fields differ, `("ab","c")` against `("a","bc")`, is what distinguishes them |
| `equals_accepts_any_type` | dropping the type assertion's `ok` compares the receiver against the ZERO value a failed assertion yields, which is false for every non-zero receiver. The row that catches it is a ZERO receiver asked about a non-RendererDetail |

A third was a harness bug rather than a test gap, and it is worth recording
because it would have scored a defect as unkillable: the anchor
`return nameHash ^ idHash` appears TWICE in the file — once in the doc comment
that quotes the reference's body, and once in the code. Replacing the first
occurrence mutated the COMMENT and left the projection intact. A mutation whose
anchor can match prose is not a mutation.

## What is proved, and where

```text
renderer_detail_test.go   5 tests: the two accessors and the zero value, the
                          two-field equality including a same-concatenation
                          pair, op_Inequality as the negation, Equals's null and
                          exact-type guards from a zero receiver, GetHashCode's
                          IsNullOrEmpty guard and its XOR including x^x = 0, and
                          ToString answering the type name for every value
external canary           1 test compiling the type from outside, where the only
                          value a consumer can build is the zero one
```

```text
FOUNDATION_MILESTONE_86_COMPLETE=true
```
