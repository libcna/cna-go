# Foundation 68 — GraphicsAdapter, and a narrowing that belongs to CNA

Two types close and `GraphicsDevice::Adapter` with them. The milestone's shape
is one measured constraint: **CNA enumerates adapters through a device, and
XNA's adapter statics answer before one exists.**

```text                                          before   after
TOTAL_DIAGNOSTICS                                 152     149
MISSING_TYPE                                      117     115
MISSING_MEMBER                                     35      34
COMPLETE_TYPES                                    136     138
UNEXPECTED_MEMBER                                   0       0

TARGET_TYPES                                      140     142
TARGET_MEMBERS                                   2278    2301
BOUND_FUNCTIONS                                   124     137
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS              297     316
behavior corpus                                   735     737
external canary tests                              87      88
native stress scenarios                            17      18
capability rows                                    67      68
```

## The narrowing, stated where a consumer meets it

Every one of CNA's twelve `cna_graphics_adapter_*` routes takes a
**callback-scoped graphics-device handle**, which CNA requires as proof of an
active runtime and the right thread. `GraphicsAdapter.Adapters` and
`DefaultAdapter` are STATIC in XNA and answer before any device exists — they
are how a consumer picks one.

So both are projected, both are **fallible**, and both refuse outside a
lifecycle callback with a message that names CNA's requirement. That is not a
gap being hidden. The alternative — inventing an adapter list from nothing —
would be reporting hardware that was never enumerated, which is the one thing
this binding must not do.

```text
GraphicsAdapter   BORROWED. No CNA handle, nothing to destroy.
```

CNA identifies an adapter by an **index**, not a handle, and every query takes
that index plus a live device. So a `GraphicsAdapter` here is the index plus the
values CNA reported when it was read — which is what the reference's is too: a
managed object over data the driver was asked for once.

## Eleven readers and three queries, and the split is the reference's

```il
get_Description   ldarg.0; ldfld ...; ret     (and ten more like it)
```

The eleven readers answer from a snapshot taken once, in `readAdapter`, so all
eleven are `managedStoredMembers` and none can fail — exactly as the
reference's cannot. The three that ASK the adapter again —
`IsProfileSupported` and the two `Query…Format` members — reach CNA per call and
carry its error, where the reference answers from capability bits it cached at
enumeration. That is the one place this projection genuinely differs, and it is
where the difference is recorded.

Three details are passed through rather than invented:

- **`Revision` and `SubSystemId` are zero**, and CNA says so in its own header:
  "current CNA returns zero". Computing something plausible would be inventing a
  hardware fact.
- **`IsWideScreen` is CNA's**, not a second `AspectRatio > 1.6` computed here.
  Two computations of one predicate can disagree; one cannot.
- **`MonitorHandle` answers zero on a headless renderer.** CNA refuses the route
  with `CNA_RESULT_NOT_SUPPORTED` — "native monitor handles do not cross the
  stable CNA C ABI" — and `IntPtr.Zero` is what XNA reports with no monitor, so
  the refusal is absorbed rather than failing the whole snapshot. The mutation
  that propagates it fails the scenario.

`MonitorHandle` is one of the **six** signature positions the raw-handle rule
admits a public `uintptr` on, because the authoritative XNA metadata declares
`System.IntPtr` there.

## DisplayModeCollection's indexer is a filter

```il
get_Item(SurfaceFormat format)
  List<DisplayMode> matches = new List<DisplayMode>();
  foreach (DisplayMode mode in _displayModes)
      if (mode.Format == format) matches.Add(mode);
  return matches;
```

Its argument is a **format**, its result is a **sequence**, and a reader who saw
"indexer" and expected `modes[3]` would be wrong about both halves. The result's
CLR type is `IEnumerable<DisplayMode>`, which the settled BCL-interface rule
projects to `any` — holding this profile's sequence adapter,
`framework.Iterator[*DisplayMode]`.

## Two verifier catches worth naming

**The display-mode prototypes were wrong, and the ABI probe said so.** Both
routes carry a FILTER pair — a `CNA_Bool` and a `CNA_SurfaceFormat` — between
the adapter index and the output, and the first draft omitted them. The
canonical-header probe refused to compile with the exact mismatch. The filter is
now declared and deliberately **not used**: `DisplayModeCollection`'s indexer
filters in managed code exactly as the reference's does, and asking CNA to
filter as well would be a second answer that could differ.

**The adapter string reads do not use the length-then-copy shape**, and that was
also measured rather than assumed. `cna_graphics_adapter_copy_description`
answers a zero capacity with `CNA_RESULT 14` — "the graphics-adapter string
output buffer is too small" — rather than with the required count. The two
`byte_length` fields in `CNA_GraphicsAdapterInfo` **are** the length call for
this family, which is why that structure carries them, and the projection uses
them.

## Evidence

An eighteenth native scenario, twenty isolated cycles, against **both** qualified
artifacts with identical results:

```text
ADAPTER_CYCLES                20   ADAPTER_PROFILE_CHECKS           20
ADAPTER_ENUMERATIONS          20   ADAPTER_FORMAT_QUERIES           40
ADAPTER_SNAPSHOT_CHECKS       20   ADAPTER_PREFERENCE_CHECKS        20
ADAPTER_DEVICE_ADAPTER_CHECKS 20   ADAPTER_OUTSIDE_CALLBACK_CHECKS  20
```

The preference round trip is the sharpest of these. CNA's route takes **both**
flags at once, so a setter that passed a default for its neighbour would
silently reset it; the scenario sets each, reads both, clears one and requires
the other to survive. The mutation that clears the neighbour fails.

Eight mutations were planted. Five fail — three in-package and two in the native
scenario — and **three are recorded as unfalsifiable on these artifacts rather
than counted**:

```text
CAUGHT           the indexer stops filtering by format
CAUGHT           the static members answer an empty list instead of refusing
CAUGHT           the exact-match/values agreement is broken
CAUGHT (native)  setting one preference clears the other
CAUGHT (native)  a headless monitor refusal fails the whole snapshot
RECORDED         DefaultAdapter picks the flagged adapter          BLOCKED_HARDWARE
RECORDED         the device reports adapter zero instead of its own BLOCKED_HARDWARE
RECORDED         the exact-match flag is invented instead of reported BLOCKED_RENDERER
```

The first two recorded ones need a **multi-adapter machine**: with one adapter,
element zero, the flagged adapter and the device's own adapter are the same
object, so no test on this hardware can tell them apart. The third needs a
renderer that **substitutes** a format: CNA reports `exact=true` with every
requested value unchanged on both artifacts, so a hardcoded `true` is
indistinguishable from the reported one. A fourth candidate — answering nil
rather than an empty slice from the filter — was withdrawn as non-evidence: the
difference is not observable through the iterator the member returns.

Two ABI prototype controls and three layout controls ship with the thirteen
routes. `CNA_GraphicsAdapterInfo` carries **four adjacent `CNA_Bool`s**, so
swapping the two a consumer actually reads is invisible to C and makes every
adapter report the other's answer.

## What this milestone does not claim

- **No adapter was enumerated outside a callback**, because CNA cannot.
- **Only one adapter was ever seen.** Both qualified artifacts report a single
  adapter, so nothing here distinguishes "the first" from "the default" from
  "the device's own".
- **No format substitution was observed.** Every query on both artifacts
  returned exactly what it was asked for.
- **`Revision` and `SubSystemId` are zero because CNA returns zero**, not
  because the hardware has none.
- **`GraphicsDevice` is still PARTIAL at 18.** `Reset`, `Present`,
  `PresentationParameters`, the render-target members and the six `DrawUser*`
  overloads remain.
