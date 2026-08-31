# Foundation 53 — Texture2D's stream surface, and a BCL mapping that had one direction

`Texture2D` gains the sized `FromStream` and both `SaveAs` overloads. Three
missing members close, `TOTAL_DIAGNOSTICS` drops from 215 to 212, and the
verifier's `System.IO.Stream` mapping stops being one-directional.

## One private method behind three public doors, again

`SaveAsPng` and `SaveAsJpeg` are eleven bytes of IL each over one private
`SaveAsImage(Stream, XnaImageFormat, int, int)`:

```text
SaveAsPng    ldc.i4.2      // XnaImageFormat.Png
SaveAsJpeg   ldc.i4.0      // XnaImageFormat.Jpeg
```

and the sized `FromStream` is twenty-three:

```csharp
XnaImageOperation op = (zoom == false) ? 1 : 3;
return new Texture2D(graphicsDevice, stream, width, height, op);
```

CNA carries the same distinction as a `zoom` bool with the same meaning —
cover-and-crop when true, fit while preserving the aspect ratio when false — in
`CNA_Texture2DDecodeInfo`, which the existing decode route already accepted and
CNA-Go had been passing as null.

### The one identity that does not cross unchanged

```text
XNA SharedConstants.XnaImageFormat    Jpeg 0    Png 2
CNA_TEXTURE_IMAGE_FORMAT_*            PNG  0    JPEG 1
```

Every other enum this project has crossed agreed on both sides. These two do
**not**, and the failure they invite is silent: forwarding XNA's own constant
would encode a **JPEG under `SaveAsPng`**, and nothing about the call would
report it — the bytes are valid, the write succeeds, the member returns nil.

The mapping is therefore made once, in the Graphics package where both names are
visible, and it is asserted three ways: an in-package test that requires the two
CNA identities to be 0 and 1 **and** to differ from XNA's, a behavior-corpus
row, and — the one that actually catches a regression — the stress scenario,
which checks the encoded bytes for the **PNG signature** and the **JPEG SOI
marker** rather than for a length. A length proves only that something was
written.

## A BCL mapping that had one direction

`bclTypes` maps `System.IO.Stream` to `io.Reader`. That is right for most
positions and it was wrong for exactly these two: `SaveAsPng` and `SaveAsJpeg`
hand the stream their encoded bytes, and a consumer given an `io.Reader` could
not pass the destination the member exists to fill.

The direction cannot be derived from the signature — the CLR spells both
`Stream` — so `tools/api_compat` now carries `writtenStreamParameters`, a
registry keyed by the full CLR member identity and naming the positions that are
**written**, measured from the member's body. The key is the full identity so an
overload that reads and one that writes stay distinguishable.

It fails loudly rather than silently. `applyStreamDirection` panics on an entry
whose position is out of range or is not an `io.Reader`, because a registry that
described the wrong member would otherwise become an expected signature nobody
wrote. Three controls ship with it: the rewrite itself, an entry pointing at an
`int32` (which must panic), and a check that every entry names a member the
pinned contract actually declares.

## The guards, and the one Go cannot have

`SaveAsImage` opens with three throws:

```csharp
if (stream == null)   throw new ArgumentNullException("stream", FrameworkResources.NullNotAllowed);
if (!stream.CanWrite) throw new ArgumentException("stream");
if (format != 0 && format != 2) throw new ArgumentException("format");
```

The **first** is reproduced, with Microsoft's own sentence, and it runs before
anything about the texture is read — so a nil texture written to a nil
destination reports the argument, which a test asserts.

The **second has no Go counterpart** and is recorded rather than invented: an
`io.Writer` has no `CanWrite`, and every one of them either writes or reports an
error when asked, so there is no state to test beforehand. Its shape is worth
recording while it is in view: `new ArgumentException(string)` takes a
**message**, not a parameter name, so the reference's exception reads literally
`"stream"`. A projection that turned that into a parameter name would be
reporting something the reference does not say.

The **third is unreachable** from either public entry point — both pass a
constant — and it stays unreachable here for the same reason.

## Two calls to encode, and why the second one's count wins

CNA reports the encoded size of an encode it has not performed yet
(`cna_texture2d_get_encoded_byte_count`) and then performs it
(`cna_texture2d_copy_encoded`). The returned slice is bounded by the **copy's**
own reported count, not by the measurement: a second encode could in principle
produce fewer bytes than the first measured, and trusting the first count would
return trailing zeros as image data.

## ABI

```text                                        before      after
BOUND_FUNCTIONS                               74      76
PROTOTYPE_TYPE_POSITIONS                     229     243
LAYOUTS / MANIFEST_LAYOUT_AGREEMENTS         146     152
native ABI mutation controls                  87      91
ABI_MISMATCHES / FINDINGS                      0       0
```

Four new controls, two of each kind:

- **`copy-encoded-capacity-and-out-count-swapped`** — the last two parameters are
  a `uint64_t` and a `uint64_t*`, and the bridge passes both straight through, so
  nothing in C objects until a prototype is compared with the canonical one.
- **`copy-encoded-destination-declared-const`** — the destination is **written**
  by the callee, and declaring it const compiles on the bridge side, which only
  forwards the pointer.
- **`decode-info-reserved-bytes-widened`** — an eighth reserved byte grows the
  structure from 24 to 32.
- **`texture-image-format-narrowed`** — a `uint16_t` identity. PNG is 0 and JPEG
  is 1, so every declared value survives sixteen bits; like the clear mask, the
  probe cannot see it, because the manifest declares the alias inside a guard the
  canonical header satisfies, and only the manifest-only probe's `sizeof`
  reports it.

## Evidence

The device-state scenario grows again, still 20 isolated cycles:

```text
DEVICE_STATE_TEXTURE_ENCODE_CHECKS       40   PNG and JPEG per cycle
DEVICE_STATE_TEXTURE_DECODE_SIZE_CHECKS  40   both zoom modes per cycle
DEVICE_STATE_TEXTURE_ENCODE_REFUSALS     20
```

A texture is decoded from the fixture PNG, encoded to **both** formats, each
encode checked for its own format signature, and then decoded back at a
requested 24×24 through both zoom modes, with the resulting dimensions read from
CNA. The refusal is a nil destination carrying Microsoft's sentence.

**Every counter from the other twelve scenarios is byte-identical.**

## Scoreboard

```text                                      before    after
TARGET_MEMBERS                              1928     1931
MISSING_MEMBER                                83       80
TOTAL_DIAGNOSTICS                            215      212
COMPLETE_TYPES                               120      120
UNEXPECTED_MEMBER                              0        0

behavior corpus                              681      684
external canary tests                         79       80
native stress scenarios                       13       13
native ABI mutation controls                  87       91
runtime capability rows                       54       54
```

`Texture2D` has six members left, and they are one blocker: the generic
`SetData`/`GetData` overloads, whose `!!0[]` parameters wait on the
generic-method projection rule.

## What this milestone does not claim

- **Nothing here proves the encoded image is CORRECT**, only that it is a PNG
  and a JPEG respectively, at the size requested, and that decoding one back
  produces a texture of the requested dimensions. Comparing pixels would need
  `GetData`, which is one of the six members still missing.
- **The `zoom` distinction is CNA's implementation of it.** XNA's operation
  identities 1 and 3 map onto CNA's `zoom` false and true by their documented
  meanings, which agree; what a HEADLESS renderer does with a non-square crop is
  CNA's decision and is not asserted here.
- **`!stream.CanWrite` has no projection.** The reference's second guard is
  recorded as absent rather than approximated.
- **`Texture2D` is still partial**, and the `System.IO.Stream` direction registry
  currently names exactly two positions. Every other `Stream` in the profile is
  still an `io.Reader`, which is correct for every one measured so far.
