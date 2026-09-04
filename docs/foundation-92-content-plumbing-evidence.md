# Foundation 92 — the content plumbing

Five types: `ContentReader`, `ContentTypeReader`, `ContentTypeReader<T>`,
`ContentTypeReaderManager`, `ResourceContentManager`. The layer between a
`ContentManager` and a compiled asset, which is what the Model family's native
draw slice has been waiting for.

## What this milestone settles

Three recorded blockers turned out to be claims rather than measurements. Each
was re-measured and each was wrong, in a different way.

### 1. `System.IO.BinaryReader` "depends on seeking" — HALF wrong

`ContentReader`'s base has been a deferred BCL base since Foundation 29, partly
on the ground that its inherited surface needs `Stream::Seek`.

Measured against the pinned IL: `ContentReader`'s entire body contains **zero**
`Stream::Seek` call sites. The one apparent hit is the substring of
`get_CanSeek`, inside a `private static` helper whose truncation check is
*skipped* when the stream cannot seek.

`Read7BitEncodedInt` is the half that is real — it is `protected`, so it is not
public surface either way, and `ReadString` is what needs it. It is projected
unexported for exactly that reason.

### 2. `System.Resources.ResourceManager` "has no CNA counterpart" — wrong

The record treated this as a whole BCL type to be projected. Measured: it
occupies **one** contract position and **one** member of it is used —
`GetObject(String)`. That is the settled role rule's case, so it maps to
`func(string) any`, the lookup a consumer supplies.

### 3. `ReadOnlyCollection` as a base — overturned in Foundation 90

Recorded here because it is the third of the same shape.

The rule this produced, now in `plan.md`:

> **A recorded blocker is a claim, and claims get re-measured.** A blocker
> written down in one milestone is evidence about what was known then, not
> about what is true now.

## The design decisions and what was rejected

### `ContentTypeReader` is substitutable, and the verifier was right

I first projected `ContentTypeReader` on the Game/GameCallbacks shape — a host
object plus a consumer callback struct. The verifier refused it: substitutability
on this type is **live**. `ContentReader::ReadObject` declares two
`ContentTypeReader typeReader` parameters, and `ContentTypeReader<T>` is a
projected type deriving from it -- the positions were already there, and the
requirement went from LATENT to LIVE the moment something could be handed to
one. `ContentManager` went live in the same milestone and the same way, when
`ResourceContentManager` became its first projected derived type.

It was right about the consequence, too. CNA performs the type-reader dispatch
itself, so a consumer **cannot** add a reader to a running load. The
substitutable base is the honest projection of a type a consumer can name and
receive but not usefully extend, and that is recorded rather than papered over.

### The byte source is narrowed to one method

`binaryReaderBase` held `owner *ContentReader` and used it for exactly one
thing. It now holds a `contentByteSource` — a single `readExact`. This is not a
test seam bolted on: it is the field's actual role, and narrowing it is what
makes the *decoding* reachable without a compiled asset while the handle, the
disposal latch and the short-read refusal stay behind it.

`ReadSingle` and `ReadDouble` keep their public members on `ContentReader`,
because the pinned contract **declares** both there -- XNA re-declares them
rather than inheriting them, which a comment in the projection had gotten
backwards. Only their shared decode moved onto the adapter, unexported, so this
family holds one answer about byte order rather than two.

### `nativestream` keys by the stream object

CNA's `cna_content_reader_create` needs a `CNA_StorageStreamHandle`, which only
`storage.h` produces. The registry in `internal/nativestream` maps a stream
object to its handle. Two alternatives were rejected and are recorded there: an
exported accessor would add public surface XNA does not have, and duplicating
the handle onto the reader would give one native object two owners.

### Three inherited `Read` overloads are NOT projected

`Read()`, `Read(Byte[],Int32,Int32)` and `Read(Char[],Int32,Int32)`. The reason
is a **mechanism** limitation, not a measured one: the inherited-projection model
cannot express an overloaded inherited member, so the registry cannot name them
and the verifier rejects a type that declares them anyway. Recorded rather than
worked around.

## Falsifiability

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 48 | 48 | 0 |
| equivalent | 2 | 0 | 2 |
| native | 0 | — | — |

The equivalent table's expected result IS survival; see below.

They cover: the constructor's guard and its totality; `get_IsValueType`'s
projection onto Go kinds; both virtual defaults; the abstract `Read`'s refusal
and the type it names; the generic reader's `typeof(T)`; both arms of the
measured cast; the manager's registry and its keying; `ResourceContentManager`'s
two guards, their **order**, its two distinct messages and its CLR `this`; and
the whole inherited decode — byte order at five widths, IEEE-754 at both,
`ReadBoolean`'s non-zero comparison, the four details of `Read7BitEncodedInt`,
`ReadString`'s byte-not-character length and its empty case, `ReadChar`'s
multi-byte decode, and the negative-count guards. Three more cover the identity sites the widened
receiver introduced, and the narrowing helper's null.

There is **no native table**, and that is the honest state: nothing in this
family can be reached against a real CNA reader yet, because
`cna_content_reader_create` needs a storage stream carrying a compiled asset and
the project authors none. Nine of these defects survived the first run for
exactly that reason, and the fix was to make the pure decoding reachable — not
to score them as killed.

### Two equivalent mutants, asserted to survive

`seven_bit_int_forgets_the_shift_mask` replaces `<< (shift & 31)` with
`<< shift`. The refusal at `shift == 35` fires at the **top** of the loop, before
the byte that would use the shift is read, so `shift` takes only 0, 7, 14, 21
and 28 at the shift site. Every one is below 31 and the mask cannot change the
result.

The mask is still written, and not as decoration: C# masks a shift count by 31
implicitly, while Go answers 0 for a shift at or past the width. The `& 31` is
what makes the Go spelling match the reference **without depending on the
refusal** to keep the count small.

`read_asset_drops_its_own_disposal_guard` neuters `ContentManagerReadAsset`'s
disposal check. It survives because this projection has `ReadAsset` **delegate
to** `Load` -- the inverse of the reference, where `Load` calls `ReadAsset`, and
a divergence already recorded on the member -- so `Load`'s own guard fires and
produces the same message from the same identity.

The guard is kept because the reference gives `ReadAsset` its own site, and
because the delegation is precisely what makes it redundant: reverse the
delegation and the redundancy is gone. Deleting a guard the reference has, on
the strength of a divergence this projection introduced, is the wrong direction.

The harness runs both in a separate table and asserts they SURVIVE. A kill would
mean the argument is wrong, or that the tests pin something the reference does
not promise.

### One thing this milestone deliberately does NOT claim

`ContentManagerReadAsset` forwards the substitutable **reference** to
`ContentManagerLoad`, not the narrowed `*ContentManager`, because the CLR `this`
does not change across a call. That is **not observable today**: `ReadAsset`'s
own guard fires before the delegation and `Load`'s only identity site is the
same disposal check.

A test asserting the forwarding was written, and it passed with the narrowing
planted -- it was measuring `ReadAsset`'s guard while appearing to measure the
call. It was rewritten to claim only what it shows. The forwarding stays as the
faithful spelling, with its unobservability recorded at the call site.

## Native qualification

Both artifacts, 20 cycles each, exit 0:

| artifact | exit | CONTENT_CYCLES | CONTENT_LOADS | CONTENT_TYPE_REFUSALS | STORAGE_ROOT_CHECKS |
| --- | ---: | ---: | ---: | ---: | ---: |
| HEADLESS | 0 | 20 | 20 | 20 | 20 |
| SOFTWARE | 0 | 20 | 20 | 20 | 20 |

Storage is re-qualified here rather than taken from Foundation 91, because this
milestone changes `storageStream`: it registers and forgets a native handle so
`ContentReader` can be built over one.

The storage slice needs `CNA_GO_STORAGE_ROOT` and `XDG_DATA_HOME` pointed at a
project-controlled directory, and it SKIPS loudly without them rather than
writing to the host's `~/.local/share`. **A first pass here ran without them and
every `STORAGE_*` counter came back zero.** The slice announced the skip on
stderr, which is what stopped it being read as a pass; the counters are what
proved it. The qualification was re-run with both set to a directory under
`build-probe/`, inside the repository and inside the shared probe directory the
build rules already name, and every storage counter came back at 20.

Foundation 91's own root was `build-probe/storage-root`, and that is the one to
reuse; this run used a second directory beside it before that was noticed, and
the duplicate was removed afterwards.

## Resource strings

Six messages registered and verified byte-for-byte against the retained
`Microsoft.Xna.Framework.dll` resource set, under the keys the reference's own
throw sites call: `BadXnbWrongType`, `BadXnbSize`, `BadXnbMagic`,
`BadXnbVersion`, `OpenResourceNotFound`, `OpenResourceNotBinary`.

`clrSpelling` gained `%v` alongside `%s`, because `BadXnbWrongType`'s second and
third positions hold types and Go has no verb that prints a `reflect.Type` the
way `String.Format` prints a `System.Type`. Both restore to the same `{n}`, and
they are now replaced in the order they appear in a single scan.

## The external consumer canary caught the widening

Adding `ResourceContentManager` made `ContentManager` a substitutable base, so
`ContentManagerLoad` and `ContentManagerReadAsset` widened their receiver from
`*ContentManager` to `ContentManagerReference`. The canary **failed to compile**,
which is what it is for: a public signature change is not allowed to be silent.
It was updated with the reason recorded at the pinned line.
