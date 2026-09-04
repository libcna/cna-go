# Foundation 83 — OcclusionQuery, and the dynamic-buffer probe

```text
COMPLETE_TYPES   181 -> 182        MISSING_TYPE       76 -> 75
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  298 -> 304        DELIBERATELY_UNBOUND_ROUTES  +1
```

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.dll            (FrameworkResources, four messages)
```

## The type is a four-flag state machine

```text
_pixelCount  _isAvailable  _isInBeginEndPair  _hasCalledBegin  _hasIsCompleteBeenQueried
```

Every guard is a test of one of those and none is CNA's, so the whole machine is
projected managed-side and CNA is reached only for the four operations that
touch the GPU. Three details decide the behaviour and all three are pinned:

1. **The constructor's ONE store is `_hasIsCompleteBeenQueried = true`.** A
   fresh query may Begin immediately; the guard exists to stop a SECOND Begin
   before the first result was looked at.
2. **`IsComplete` is not a pure read.** The arming store is its FIRST statement,
   ahead of both early returns, so merely asking re-arms Begin even when the
   answer is false.
3. **`PixelCount` calls `IsComplete`**, so reading the count re-arms Begin too,
   and its refusal is the property's own rather than a stale-value read.

Detail 2 is not a curiosity. The first version of the native scenario asserted
"a query inside its own pair is not complete" during the first pair and then
asserted "a second Begin is refused" after it — and the first assertion **armed
the query**, so the second fired on correct code. The two claims cannot share a
pair, and the scenario now uses two.

## The profile check is CNA's refusal, in the same place

The reference's constructor throws `NotSupportedException` from
`ProfileCapabilities` when the profile lacks the feature. `ProfileCapabilities`
is not a public XNA type, CNA-Go projects no part of it, and there is no
measured table to test against — the same position `DrawPrimitives`' profile cap
is already in. `cna_occlusion_query_create` answers `CNA_RESULT_NOT_SUPPORTED`
where the backend has no query object, so a consumer meets a refusal at the same
moment with the renderer's reason instead of an invented profile one.

## A behaviour that contradicted the assumption

`IsComplete` **inside** a Begin/End pair answers differently on the two
artifacts, and neither answer is XNA's:

```text
                                   HEADLESS   SOFTWARE
OCCLUSION_QUERY_COMPLETIONS              20          0
OCCLUSION_QUERY_PENDING_CHECKS            0         20
OCCLUSION_QUERY_STALE_RESULT_CHECKS      20          0
OCCLUSION_QUERY_FRESH_RESULT_CHECKS       0         20
```

XNA's `Begin` sets `_isAvailable = false` and its `GetData` returns `S_FALSE`
until the pair is submitted, so a query inside its own pair reports incomplete.
CNA's route answers a different question — it is documented as "whether the
query result can be read without stalling the CPU" — and on HEADLESS, where the
first pair completes immediately, the second pair reports TRUE because the FIRST
pair's result is still readable. On SOFTWARE nothing ever completes within the
frame, so nothing is readable and the same check reports false.

The two columns are internally consistent: a stale result can only be reported
where a previous one exists. **The scenario records both outcomes rather than
asserting either**, because the assertion it started with was a physical
assumption that the measurement refuted.

The projection does not mask it. The reference's `IsComplete` re-reads the
runtime and overwrites its own flag as well, so the structure is faithful and
the answer is the runtime's — which is the settled position for every divergence
of this kind.

## The dynamic-buffer probe, which is the other half of this family

The ROADMAP's standing note said the two dynamic buffers have no dedicated CNA
routes and that "the hypothesis that the ordinary buffer routes plus a
BufferUsage implement them must be probed before anything is bound". The probe
was run against the canonical headers and the hypothesis is **confirmed, and
more precisely than it was stated**:

```text
CNA_VertexBufferCreateInfo.dynamic   "True to construct DynamicVertexBuffer;
                                      false to construct VertexBuffer."
CNA_VertexBufferInfo.dynamic          "True when this handle owns a DynamicVertexBuffer."
CNA_VertexBufferInfo.is_content_lost  "Dynamic content-loss state; currently always false."
CNA_VertexBufferTransfer.options      "non-None values require a supported
                                       dynamic-buffer overload"
cna_vertex_buffer_subscribe_content_lost / _unsubscribe_content_lost
cna_index_buffer_subscribe_content_lost  / _unsubscribe_content_lost
```

Three consequences for the milestone that projects them:

- The creation flag and the info fields are **already bound** — Foundation 65
  and 66 plumbed `dynamic` and `is_content_lost` through when they projected the
  static buffers.
- `IsContentLost` will always answer **false** and the `ContentLost` event will
  never fire on either qualified artifact, because CNA documents the state as
  "currently always false". That is a limitation to record, not a defect.
- What actually remains is composing **VertexBuffer** and **IndexBuffer**, which
  are still `DEFERRED` in `xnaBaseRelationships`, plus the options-carrying
  upload routes and the two subscription pairs.

Nothing is bound on that hypothesis in this milestone.

## Planted defects

16 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   16      KILLED   15      SKIPPED   0
```

10 are killed by the in-package tests and 5 by the vertex-buffer stress child.
Three of the five needed assertions that the first pass did not have, and each
is a distinct lesson:

| defect | why the first assertion missed it |
| --- | --- |
| the constructor leaves Begin unarmed | the managed tests build the object by hand, so the CONSTRUCTOR's store is never exercised — it belongs to the device-backed suite |
| Begin stores before the native call | a query with no resource refuses before the call, so the reordering was invisible; a query with a DEAD resource reaches the call and fails there |
| Dispose leaves the CNA handle live | the assertion used `Begin`, whose arming guard refuses first whatever the handle is doing. `IsComplete` reaches CNA with no managed guard in front of it, and is the member that proves the release |

**One is not killed, on either artifact.** `is_complete_ignores_the_native_answer`
inverts the availability test. On HEADLESS it turns twenty completions into
twenty pending checks and on SOFTWARE the reverse, and the accounting balances
either way because both outcomes are legitimate. The SOFTWARE attempt was run
specifically to see whether reading a pixel count from a query CNA calls
incomplete would fail; it does not. Nothing in the bound surface can see CNA's
own answer independently of the member under test — the same class as Foundation
80's two push defects, and the same thing a pixel-level check would move.

## Native qualification

```text
                                HEADLESS   SOFTWARE
OCCLUSION_QUERY_CREATIONS             20         20
OCCLUSION_QUERY_CREATION_REFUSALS      0          0
OCCLUSION_QUERY_PAIRS                 40         40
OCCLUSION_QUERY_GUARD_CHECKS          20         20
OCCLUSION_QUERY_DISPOSAL_CHECKS       20         20
```

Both artifacts create the query and run both pairs; they differ only in the
completion columns above, which is the renderer's timing rather than the
projection's.

## What is proved, and where

```text
occlusion_query_test.go      6 tests: the constructor's one store, both Begin
                             guards and the order they are tested in, End's own
                             message, the arming store running ahead of both
                             early returns, PixelCount's refusal and its arming
                             side effect, the stores that must follow a failed
                             issue, and the nil receiver on all five members
native_stress vertex-buffer  20 cycles on two artifacts: a fresh query beginning
                             immediately, the pair guard, the disarm that makes a
                             second pair illegal, the completion and pending
                             paths both counted, the inside-pair answer recorded
                             in two buckets, and a disposal proved through
                             IsComplete rather than through a guarded member
external canary              1 test compiling the type from outside, including
                             both dispose members -- which it projects and the
                             stock effects do not, because it DECLARES
                             Dispose(bool)
tools/resource_strings       four more Microsoft messages verified, one of them
                             a key whose sentence is not what the key suggests
```

```text
FOUNDATION_MILESTONE_83_COMPLETE=true
```
