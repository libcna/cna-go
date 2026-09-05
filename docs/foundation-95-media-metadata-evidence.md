# Foundation 95 — the media metadata graph

Ten types: `Song`, `Album`, `Artist`, `Genre`, `Playlist` and their five
collections. The music metadata half of
`Microsoft.Xna.Framework.Media`, over **105 CNA routes** — the largest native
binding in one milestone so far.

`BOUND_FUNCTIONS` 407 → 512, `COMPLETE_TYPES` 231 → 241.

## Why these ten and not fewer

The graph is connected and this is its minimum closed slice. `Album.Songs`
returns a `SongCollection`, `Artist.Albums` an `AlbumCollection`, `Song.Artist`
an `Artist`. There is no smaller subset whose public surface closes.

What is deliberately NOT here: `MediaLibrary` (148 routes), `MediaPlayer` (41),
`MediaQueue`, `MediaSource`, and `Video`/`VideoPlayer` (42). Those are
enumeration and playback, and each is its own question.

## The BCL closure was small, and one mapping is measured

`System.Uri` is named at exactly ONE signature position, `Song.FromUri(String,
Uri)`, and the reference does exactly two things with it: compares it against
null, and hands it to a P/Invoke that marshals it as a string. Nothing reads a
scheme, a host or a path. So it maps to `string`, and the null comparison
becomes an empty-string check — the same refusal for the only value a Go caller
can pass in a null's place.

## Two fallibility arms the rules were missing

**`ToString`, `GetHashCode` and the operators are blanket-exempted from
fallibility**, because in this profile they have always read stored state:
`GraphicsResource::ToString` answers a managed name and reaches nothing.

The media entities are the first where that is false. `Song::ToString` is
`ThrowIfDisposed(); return get_Name()` and the name is NATIVE; `GetHashCode`
caches that same name's hash; the two operators forward to an `Equals` that asks
CNA whether two handles denote the same object. A member that reaches a runtime
can be refused by it, so `runtimeReadMembers` answers the exemption — from a
registry, because the default is right for every other type.

**`get_IsDisposed` is one `ldfld`** over a private bool, measured the same way
SoundEffect's was, so all ten go in `managedStoredMembers`. CNA has a
`_get_is_disposed` route for every one and none is bound to this member: asking
CNA whether an object is disposed means touching a handle the consumer has
already disposed, which is what the latch exists to avoid.

## A defect the tests found in the projection

`mediaObject.usable()` checks `o == nil`, and that check **cannot fire through a
nil outer pointer**: reaching the embedded field is itself the dereference, so
the panic happens before the guard runs. The first run of
`TestAMemberOnANilReceiverRefusesRatherThanPanicking` panicked.

Each of the ten types now shadows it with a guard that tests its own pointer
first. Ten near-identical methods are the price of embedding a shared guard
behind a typed receiver, and it is paid there rather than by asking every member
to remember the check.

## The index bound was narrowed to make it reachable

The five collection indexers sit behind a runtime guard, so a test with no
runtime cannot reach the bound comparison through them — and the bound needs no
runtime, only a count. It is now `checkCollectionIndex`, a named function, for
the same reason `binaryReaderBase`'s byte source was narrowed in Foundation 92:
the mutation run showed four bound defects surviving because nothing could
reach them.

## Falsifiability

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 29 | 29 | 0 |
| native | 5 | 5 | 0 |
| equivalent | 1 | 0 | 1 |
| unqualified | 1 | 0 | 1 |

**Updated by Foundation 96.** Two of the three defects this milestone could not
score are now scored, because the media library gives the collections an entry
point and its stress slice seeds an isolated one with three songs and three
pictures. `enumerator_starts_at_zero` is killed.
`enumerator_rereads_the_count_each_step` turned out to be EQUIVALENT rather
than unqualified: no collection in this family changes size during a walk, so
re-reading the count cannot differ. Only the leak remains unscoreable.

The first managed pass killed 16 of 33. Seventeen survivors: eleven were real
test gaps — nil receivers tested on one type instead of ten, `Finalize`'s latch
checked on `Album` while the mutant was on `Song`, an `Equals` ordering test
that accepted "false and no error" as a pass — and the rest needed a runtime.

### The native slice, and the fixture it needed

The family has ONE public entry point: `Song.FromUri`, a public static. The
contract declares no constructor on any of the ten, and the `MediaLibrary` that
would enumerate them is not projected. So the slice builds a song from a URI and
walks what it can reach.

The first run measured that **CNA validates the file**: a URI naming nothing
answers `CNA_RESULT_IO` with "Could not find file". So the slice authors one —
44 bytes of RIFF plus a tenth of a second of silence, the smallest thing that is
unambiguously a WAV — the way the content slice authors its PNG. With it,
`MEDIA_SONG_CYCLES` is 20 and `MEDIA_SONG_READS` 120.

### Three defects this host cannot score, named rather than omitted

- **`dispose_skips_the_destroy`** leaks the native object. Nothing in the
  projection's surface observes a leak: every later member refuses on the
  managed latch before it would reach the handle. A functional slice cannot see
  this.
- **`enumerator_starts_at_zero`** and **`enumerator_rereads_the_count_each_step`**
  needed a COLLECTION, and a song built from a URI has no artist until something
  reads its tags. Foundation 96 supplied one: `MediaLibrary` hands out
  collections directly and its slice seeds an isolated library with three songs.
  The first is now killed; the second is equivalent.

They are in their own table whose expected result is survival, so the totals
stay honest rather than quietly omitting them.

### The harness now refuses to start over a dirty tree

A native run was killed by a timeout mid-mutation, and the restore is in a
`finally` that a SIGKILL does not run. The planted defect stayed in the working
tree, and the next run reported its anchor "not found" — which looks like a
harness bug and is actually a corrupted source file.

`assert_clean()` now checks every mutant's replacement text before the run
starts and refuses with the file name if one is still applied. That is a
failure mode worth catching loudly: a mutation left in place would otherwise be
committed.
