# Foundation 82 — FrameworkDispatcher and TitleContainer

```text
COMPLETE_TYPES   179 -> 181        MISSING_TYPE       78 -> 76
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  296 -> 298        FRONTIER_FAMILIES  12 -> 11
```

## Reference authority

```text
Microsoft.Xna.Framework.dll   38e7093f52d7474b...
```

Both types are `public abstract sealed` — C# for a static class — and each
declares exactly one member. They project as a type identity plus one
type-prefixed package function, which is the shape `MathHelper` already has.

## Two static members that need a running game

Neither reference member needs anything: `FrameworkDispatcher.Update()` is
static and `TitleContainer.OpenStream(name)` resolves its own path. Both CNA
routes take a game handle, and the canonical header says why:

```text
The canonical dispatcher is static and exists for applications that do not run
the game loop; a game handle is taken here only for thread affinity.
```

So both projections refuse outside a game where the reference would have
worked. That refusal is the projection's own — the reference has no such
failure, so there is no message to reproduce — and it is preferred to the
alternative, which is doing nothing and reporting success. A consumer who called
`FrameworkDispatcherUpdate` and got `nil` would believe media and audio had been
pumped when they had not.

## TitleContainer's guards are 383 bytes of managed string work

`OpenStream` is 213 bytes and three of its four guards run before anything
touches a file:

```text
if (string.IsNullOrEmpty(name)) throw new ArgumentNullException("name");
name = GetCleanPath(name);
if (IsCleanPathAbsolute(name))
    throw new ArgumentException(FrameworkResources.InvalidTitleContainerName);
try { new Uri(name.Replace('\\','/'), UriKind.Relative); }
catch (Exception inner) { throw new ArgumentException(..., inner); }
```

The first guard fires for the EMPTY string as well as null, and its
ArgumentNullException carries **no message** — so an empty name is an
argument-null failure rather than a not-found one. That distinction is pinned.

`GetCleanPath` is 256 bytes and `IsCleanPathAbsolute` is 87, and both are
transcribed operation by operation because they decide which of three refusals a
caller sees. Three details a reader would guess wrong, and all three are tested:

1. **The forward-slash replacement comes first**, so a caller may write either
   separator and every later test sees backslashes.
2. **`\.` collapses a path that is nothing but `\.` to `\`**, not to the empty
   string — which the absoluteness check then rejects.
3. **`a\b\..` becomes `a\`, not `a`.** `CollapseParentDirectory` removes from
   the separator AFTER the collapsed segment: `start` is
   `LastIndexOf('\\', position-1) + 1 = 2`, and `Remove(2, 3-2+3)` takes indices
   2 through 5 out of `a\b\..`. The first version of the test expected `a`;
   tracing the IL is what settled it, and the trace is in the test.

`IsCleanPathAbsolute`'s name is misleading and its body is what matters: six
tests, of which the first is a seven-character table read from the assembly's
static blob — `3A 00 2A 00 3F 00 22 00 3C 00 3E 00 7C 00`, i.e. `: * ? " < > |`.
The remaining five are what stops a caller escaping the title directory, and
they run AFTER `GetCleanPath` has collapsed every `..` it can, so what reaches
them is a `..` that would have escaped.

## The narrowing, and the message that cannot be reached

`cna_title_container_read_ext` hands back the WHOLE FILE. The header states the
trade: "This ABI has no stream handle for title content, and a title asset is
read to use it, so the count/copy pair delivers the whole file instead. That is
a deliberate narrowing: incremental reads over a title stream are not
available." So the returned `io.Reader` is over bytes already in memory.

The alternative — resolving the title path in Go and calling `os.Open`, which is
what `ContentManager.OpenStream` does — is not available: CNA exposes the title
path only to itself here, and guessing a directory would be inventing the one
thing this member exists to resolve.

The reference sorts the open's failure into two, `OpenStreamNotFound` and
`OpenStreamError`, and CNA reports both as `CNA_RESULT_IO`. **The second message
is registered and unreachable**, and retaining it is what keeps that a recorded
limitation rather than a forgotten branch.

## A bug only the device-backed run could find

The first native run reported a file that had **just been written** as not
found:

```text
[TitleContainer] Resolved path: .../cna-go-title-probe-4115885.bin
cna_title_container_read_ext failed with CNA result 14:
The destination capacity is smaller than the title file.
```

Every other size query in this binding is a separate `_size` route that
succeeds. This route sizes and copies through one entry point, and a zero
capacity answers `CNA_RESULT_BUFFER_TOO_SMALL` **with `out_bytes` filled in** —
which the header says plainly and the wrapper had treated as a failure. Nothing
managed could have caught it; the guards all pass and the read is the only part
that crosses.

The fixture moved twice for a second measured reason. The game's own executable
looked like the one file guaranteed to be under the title path, and CNA answers
`CNA result 5: Failed to open file` for a binary that is currently executing —
so a run over it would have measured that refusal rather than a read. The
scenario writes its own asset next to the executable instead, and asserts the
bytes come back.

## Planted defects

24 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   24      KILLED   23      SKIPPED   0
```

19 are killed by the in-package tests and 4 by the vertex-buffer stress child.

**The first pass killed 19, and the four extra kills needed inputs that were not
guessable.** Three of them are the path algorithm's own edges, and one is the
brute-force result:

| defect | the input that kills it |
| --- | --- |
| the parent loop starts at index zero | `\..\a` -- a `\..\` at index zero is never found by a loop that starts at 1, so the path survives to be refused; collapsing it yields `a`, which is INSIDE the title directory |
| a bare trailing parent is collapsed | `\..` -- the collapse is guarded by `at > 0`, and collapsing it gives the empty string, which the absoluteness check accepts |
| the resume value is zero rather than `Max(start-1, 1)` | `\\..\..\` -- a DOUBLE separator, found by brute-forcing paths over `{a, \, ., ..}` to depth seven after failing to construct one by hand |
| a failed read hands back a reader over nothing | reachable only with a game: every managed guard refuses first, so the typed-nil hazard lives in the stress and not in a unit test |

**One is not killed, and the attempt to kill it measured something.**

`dispatcher_requires_a_callback` tightens the dispatcher's lookup from
`activeGame(false)` to `activeGame(true)`. The case that would tell them apart
is the one CNA's header names -- "applications that do not run the game loop" --
so the probe pumped the dispatcher between `NewGame` and `Run`.

It refused. **`framework.NewGame` constructs no native game**: CNA-Go creates it
inside `Run`, so there is no constructed-but-not-running state a consumer can
reach and every call site the projection has is already inside a callback. The
two guards are indistinguishable through the public surface.

`false` is kept because it matches CNA's documented contract -- "Active owned or
callback-borrowed game handle" -- rather than being stricter than it, and the
probe is recorded in the scenario's source rather than deleted, because a later
milestone that gives Game a constructed-but-not-running state would make it
live.

## Native qualification

```text
                                HEADLESS   SOFTWARE
FRAMEWORK_DISPATCHER_UPDATES          60         60
TITLE_CONTAINER_READS                 20         20
TITLE_CONTAINER_READ_REFUSALS          0          0
TITLE_CONTAINER_GUARD_CHECKS          40         40
```

Three dispatcher pumps per cycle, because CNA documents calling it while the
loop runs as "harmless and does the work twice" — so a second and third call
must succeed as well as the first.

## What is proved, and where

```text
title_container_test.go      7 tests: every operation of GetCleanPath in the
                             order it performs them, the seven-character table
                             and the five relative tests of IsCleanPathAbsolute,
                             seven escape attempts refused and three
                             escape-looking names accepted, the empty name as an
                             argument-null failure distinct from the invalid-name
                             one, the control-character guard, both statics
                             refusing outside a game, and the nil reader a
                             refusal must hand back
native_stress vertex-buffer  20 cycles on two artifacts: three dispatcher pumps,
                             a written asset read back byte for byte through
                             CNA's own path resolution, a missing asset carrying
                             the reference's message, and the guards holding
                             from inside a game
external canary              1 test compiling both from outside, including the
                             two type identities, both shapes, and the three
                             distinct refusals
tools/resource_strings       three more Microsoft messages verified, one of them
                             registered as deliberately unreachable
```

```text
FOUNDATION_MILESTONE_82_COMPLETE=true
```
