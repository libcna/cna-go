# Foundation 44 — migrating the native boundary from CNA ABI 0.7 to 0.21

Foundations 1 through 43 were qualified against CNA C ABI **0.7.0**, admitted by
exact encoded equality with `0x00000700`. The live CNA checkout has since moved
fourteen minor versions to **0.21.0**. This milestone migrates the boundary,
replaces exact-version admission with CNA's own stated compatibility contract,
and closes a measurement blind spot the old verifier could not see.

Nothing here adds a bound route. The point is the opposite: to establish, with
compiler and loader evidence rather than by name matching, that every route
CNA-Go already binds means the same thing at 0.21.0 that it meant at 0.7.0.

## Live dependency identity

```text
cnanext            next   0a6158e4ff764907065cd7259e3d29e331a52088   (1 behind origin)
sharp-runtimenext  next   4a49afb0cfe6a41e6e0af0bb62dc5175976731bb   (2 behind origin)
cna-go             develop e86f7c1ff8d6ddfbe82191e23c823bf8c3a5d889  (== origin, clean)
cna-go-template    develop 65254848d9fac02ace934db3879106834bafca97  (clean, no upstream)
```

Neither dependency checkout was modified. Nothing was rebuilt for CNA-Go: the
qualification artifact is an existing CNA build directory's output, retained at
`~/deps/cna-c-abi-0.21.0/` and proven byte-identical to
`cnanext/cmake-build-headless/modules/c-api/libcna_c_api.so`, with a header tree
byte-identical to `cnanext/modules/c-api/include`.

```text
CNA_ABI_VERSION            0.21.0   encoded 0x00001500
artifact                   libcna_c_api.so, 166,420,656 bytes
sha256                     c32bfbd307d695664f906ccf2834ec3f9ebc240fa388d544ac21ee3ebaeb731b
configuration              HEADLESS renderer, SDL3 platform, SDL3 audio,
                           net ON, video AUTO, Debug
canonical declarations     4054
exported cna_* routes      4054
declared-not-exported      0
exported-not-declared      0
ELF symbol-version node    CNA_C_API_0.1
```

The two-way count is the correspondence proof. A library built from different
headers than the ones the verifier compiles against would show routes declared
and unexported, or the reverse; here neither set has a single member the other
lacks.

## Phase 0B — every existing import, re-derived

All 31 bound routes still exist. That is where a name-based audit would have
stopped, and it is not enough: a route can keep its name while a parameter
widens or an aggregate grows. Two independent comparisons were run against the
retained 0.7 headers in `~/deps/cna-c-abi-0.7.0/include`.

**Declaration text.** Each of the 31 `CNA_C_API` declarations was extracted from
both header trees and compared. All 31 are character-for-character identical
after whitespace normalisation — same return type, same parameter count, order,
widths, signedness, pointer depth and constness.

**Compiled layout.** A probe printing 81 aggregate measurements — sizes,
alignments, every member offset and every scalar field width of every struct,
callback and enum the 31 routes touch — was compiled against both header trees
and run. The two outputs differ in exactly one line:

```diff
-abi_version=1792     (0x00000700)
+abi_version=5376     (0x00001500)
```

Every other measurement is identical. So the fourteen minor bumps were additive
with respect to CNA-Go's entire imported surface, which is what CNA's own
version script says minor bumps are. No bound route is grandfathered: each was
re-measured, not assumed.

Two canonical spellings did change, and neither is an ABI change.
`CNA_Texture2DInfo::format` is now typed `CNA_SurfaceFormat`,
`CNA_SpriteBatchBeginInfo::sort_mode` is now `CNA_SpriteSortMode` and
`CNA_SpriteScaledCommand::effects` is now `CNA_SpriteEffects`. All three are
`uint32_t` typedefs; the measured widths and offsets are unchanged, which is how
the probe reports them rather than how the header spells them.

## Phase 0D — the admission policy

The old policy was exact equality with one encoded number, and the old
documentation said so in as many words: "There is no same-major, newer-minor, or
generic 0.x compatibility policy." Deriving the current policy meant asking what
CNA itself promises, not what is convenient.

`modules/c-api/cmake/CnaCApiExports.map` is the live binary contract:

> CBIND-063: this name is NOT the ABI version and must not be bumped with it. It
> read `CNA_C_API_0.1` when the ABI was 0.1.0 and still reads it at 0.2.0, which
> looks like drift and is not: a version node is the name every already-linked
> consumer records in its own DT_NEEDED versioning, so renaming it turns every
> additive release into a hard break — exactly the failure this file exists to
> prevent. It changes only for a *major* ABI break, alongside a new node that
> keeps the old one for compatibility.

`docs/releasing.md` adds that `CNA_ABI_VERSION` "moves when the ABI changes,
independently of a product release", and that a pre-1.0 minor bump may change
the public API.

Read together: **a major bump is the break; a minor bump is additive.** The
admitted range is therefore major 0, minor ≥ 21, patch unconstrained — the
qualified major with a floor at the qualified minor. The floor is not
decoration: a lower minor may simply not declare a route CNA-Go binds. The upper
end is open because CNA says an additive release keeps the contract.

Nothing rests on that promise alone. After the version check the loader still
resolves all 31 symbols by name, and now also confirms with `dladdr` that each
resolved address belongs to the symbol the manifest names. A rejection reports
the library path, the version it reported, and the admitted range:

```text
CNA native C ABI is unavailable: /path/to/libcna_c_api.so reports CNA C ABI
0.7.0 (0x00000700); CNA-Go admits major 0 with minor 21 or newer (qualified at
0.21.0)
```

The policy is evaluated in exactly one place — `cna_go_abi_admits` in
`bridge.c` — and Go asks that function rather than re-deriving the comparison.

## Phase 0C — what the verifier could not see

The old verifier was already compiler-backed, and its architecture is preserved.
It had one structural blind spot, and it was not small.

`abi_manifest.h` declares CNA-Go's own copy of every struct, callback and
constant, each guarded by `#ifndef CNA_C_<HEADER>_H`. cgo compiles `bridge.c`
with **no CNA header at all**, so those private definitions are what the shipped
binding actually passes to CNA. But `probe.c` includes the canonical header
first, which defines those guards — so the manifest's private definitions are
suppressed there, and every size, alignment and offset the old probe printed was
a *canonical* measurement. The manifest's own layouts were never measured
against anything. A manifest struct with a wrong field width or a missing
reserved byte would have passed every check.

The fix is a second translation unit, `manifest_probe.c`, which includes only
`abi_manifest.h` and `bridge.h` and prints the same 112-entry measurement list
from a shared `measurements.inc`. It carries an `#error` guard that fires if a
canonical header ever reaches it, because a leak would silently turn it into a
second canonical probe reporting perfect agreement while proving nothing. The
tool compares the two sets key by key.

```text
MANIFEST_LAYOUT_AGREEMENTS = 112     divergences = 0
```

Four further strengthenings:

- **Field widths.** Offsets and total sizes are not sufficient. Narrowing
  `CNA_StringView::byte_length` from 64 to 32 bits leaves `sizeof`, `_Alignof`
  and both member offsets unchanged, because the struct's padding absorbs it —
  while the callee reads four bytes CNA-Go never wrote. 31 of the 112
  measurements are now `sizeof` on a member expression.
- **`bridge.h`'s own mirrors.** `CNA_GO_RESULT_SUCCESS`, `CNA_GO_RESULT_CALLBACK`
  and the four `CNA_GO_GAME_EVENT_*` values were previously asserted only
  against literals that happened to match. `probe.c` now includes `bridge.h` and
  compares them with the canonical constants, closing the third side of the
  triangle: canonical ↔ manifest, manifest ↔ bridge, and now bridge ↔ canonical.
- **The encoded-version mirror.** CNA-Go re-implements
  `CNA_ABI_VERSION_ENCODE` so the loader can decode a version without a CNA
  header. `probe.c` is the only place both spellings exist, and it asserts they
  agree at three sample points including the field maxima.
- **Symbol identity.** Several bound routes share a prototype —
  `cna_game_run`, `cna_game_request_exit` and `cna_game_destroy` are all
  `CNA_Result(CNA_Handle)` — so a mis-pairing among them compiles cleanly and no
  static check can catch it. `cna_go_verify_symbol_identity` walks the resolved
  table with `dladdr` and requires each address to report the name the manifest
  lists.

The route table itself is no longer maintained beside the manifest. The tool
parses `CNA_GO_REQUIRED_SYMBOLS` and each route's `_fn` typedef out of
`abi_manifest.h` — the file the cgo build compiles — so 31 hand-written
parameter counts stopped existing. The derived total matches the retired table
exactly: `PROTOTYPE_TYPE_POSITIONS = 91`.

## Mutation controls

48 controls, up from 30. Every one plants a realistic defect in real source,
requires the gate to fail, and is restored.

| class | controls | caught by |
| -- | -- | -- |
| event identity, callback shape, handle width | 9 | bridge and probe compiles |
| prototype shape, parameter order, return type | 11 | probe compile against canonical |
| frame-hook member order and mask | 5 | both compiles |
| missing or stale required symbol | 5 | bridge compile / probe compile |
| ABI admission policy and encoded mirror | 8 | bridge and probe compiles |
| **aggregate layout** | **6** | **manifest ↔ canonical measurement** |
| canonical header leaking into the manifest probe | 1 | `#error` guard |

The six layout controls are the ones worth naming, because every one of them
**compiles cleanly everywhere** — in `bridge.c`, in `probe.c` and in
`manifest_probe.c` — since C aggregates are written by field name. They are
caught only by comparing the two measurement sets:

```text
sprite-command-source-and-colour-swapped
keyboard-state-word-count-doubled
viewport-depth-widened-to-double
texture-info-struct-size-widened
game-time-tick-count-narrowed
string-view-length-narrowed
```

Two candidate mutations were **discarded rather than kept**, because they turned
out not to be defects: removing `CNA_GameTime`'s seven reserved bytes and
narrowing `CNA_StringView::byte_length` both leave every size and offset
identical. The first is genuinely unobservable at this ABI; the second was
promoted into a real control only after field widths were added to the
measurement list. A control that cannot fail is not evidence.

## Before → after

```text                                        before      after
admitted ABI                       exactly 0.7.0   major 0, minor >= 21
                                                   (qualified 0.21.0)
qualified artifact ABI                     0.7.0   0.21.0
artifact audio backend                      NULL   SDL3
BOUND_FUNCTIONS                               31   31
PROTOTYPE_TYPE_POSITIONS                      91   91   (now derived)
ROUTE_TYPE_PAIRINGS                            -   31
LAYOUTS                                       36   112
MANIFEST_LAYOUT_AGREEMENTS                     -   112
C_GO_MEASUREMENTS                            128   207
CALLBACKS                                      3   3
CONSTANTS (canonical static asserts)          15   27
MANIFEST_SIDE_ASSERTIONS (bridge.c)            -   16
CANONICAL_DECLARATIONS                         -   4054
LIBRARY_EXPORTS                                -   4054
SYMBOL_IDENTITY_VERIFIED                       -   true
MISSING_HEADER_SYMBOLS                         0   0
MISSING_LIBRARY_SYMBOLS                        0   0
ABI_MISMATCHES / FINDINGS                      0   0
native ABI mutation controls                  30   48
```

**No bound route's contract changed.** That is the finding, and it is measured
rather than assumed.

## Behavioural equivalence

`tools/native_stress` was re-run against the 0.21.0 artifact and reproduces
**every counter identically** — all 60 counters, byte-for-byte, with the single
exception of `GO_RACE_STATUS`, which the run itself sets. `cna-go-template`
builds and runs at exactly 60 and exactly 600 native draw callbacks.

## A stale blocker, retired

Foundation 1 recorded `audio` as `BACKEND_BLOCKED` because its artifact used the
NULL audio backend. The current artifact uses SDL3, so that classification was
re-measured rather than restated. A direct C probe against the artifact —
`build-probe/f44-audio.c`, linked to CNA and not through CNA-Go — created a real
Game and reported:

```text
audio_playback_available = 1
sound_effect_create      = CNA_RESULT_SUCCESS
duration_ticks           = 5000000       (22,050 mono frames at 44.1 kHz = 0.5 s)
instance_play            = CNA_RESULT_SUCCESS
instance_state           = CNA_SOUND_STATE_PLAYING
instance_volume          = 1.0
```

A real SDL3 playback device opens in this environment and a PCM16 effect plays.
The capability row moved from `BACKEND_BLOCKED` to `UNIMPLEMENTED_CNA_GO`: the
remaining blocker is CNA-Go's own missing `Microsoft.Xna.Framework.Audio`
surface, not CNA and not the artifact.

`visible-rendering` and `hardware-renderer` are unchanged and still HEADLESS —
but the reason is now recorded as artifact selection rather than an upstream
gap, because CNA publishes SDL_RENDERER, SOFTWARE, OPENGL33, OPENGLES3 and
VULKAN builds of this same ABI.

## What this milestone deliberately does not claim

- No new route is bound and no XNA member is completed. `BOUND_FUNCTIONS` is
  unchanged at 31 on purpose.
- The artifact is not sanitizer-instrumented, so `NATIVE_SANITIZER_STATUS`
  remains `NOT_RUN`.
- HEADLESS still proves native draw execution and not visible output.
- The 0.7.0 artifact is retained for history. No gate loads it, and the
  Foundation 1 record of it is preserved rather than rewritten.
