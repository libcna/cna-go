# Foundation 96 — the media library and the picture graph

Six types: `MediaLibrary`, `MediaSource`, `Picture`, `PictureAlbum`,
`PictureCollection`, `PictureAlbumCollection`. Over **64 CNA routes**, one of
which is deliberately unbound.

`BOUND_FUNCTIONS` 512 → 581, `COMPLETE_TYPES` 241 → 247.

## What it unlocks

Foundation 95 projected ten metadata types with exactly ONE public entry point
between them — `Song.FromUri` — because none of them declares a constructor. A
song built from a URI carries no tags, so it has no artist, no album, and
therefore **no collection to walk**. Three of that milestone's mutants went
unscored for exactly that reason.

`MediaLibrary` declares two constructors and hands out eight collections
directly. **Two of those three are now scored**, and the third turned out to be
equivalent rather than unqualified.

## The slice had to be contained before it could run

The first run of this milestone's stress slice found **forty-one of the user's
photographs** and read their dimensions and dates.

That is precisely the scanning of user media directories the standing
constraint forbids, and it is what a `MediaLibrary` does by design: it reads the
host's music and picture directories.

Measured, in this order:

- `XDG_PICTURES_DIR` does **not** move them — the count stayed at 41.
- `HOME` does — the count went to 0.
- CNA resolves the directories through `~/.config/user-dirs.dirs`, not through
  the environment variable, which is why the first attempt at an isolated home
  found nothing even after seeding it.

So the slice now enumerates NOTHING until `HOME` is a directory the run was
explicitly given in `CNA_GO_MEDIA_HOME`, writes its own `user-dirs.dirs` inside
that home, and seeds it with three songs, three pictures and two sub-albums. A
run without one skips loudly and records `MEDIA_HOME_SKIPS`.

`MEDIA_HOME_CHECKS` is the containment proof, the same shape
`STORAGE_ROOT_CHECKS` has. The negative control is a run whose
`CNA_GO_MEDIA_HOME` names somewhere `HOME` is not: it enumerates nothing and
counts a skip.

## Why the fixture is three of each and not one

A walk over ONE element cannot tell an enumerator that starts at the first from
one that skips it — both visit one item. Three separates them, and it is what
kills `picture_collection_walk_starts_at_zero` and its album counterpart.

Two sub-albums for the same reason: the root album's children are the
subdirectories, and one child would not distinguish the two walks.

## MediaSource is a VALUE, and that is measured

Every other type in this namespace owns a native object and implements
IDisposable. `MediaSource` declares neither: its whole contract is a name, a
kind, a ToString and one static.

CNA agrees. There is **no `CNA_MediaSourceHandle`** — the available sources are
addressed by INDEX, and an open library answers its own source's type and name
off the LIBRARY handle. So the projection holds two values.

This forces one recorded divergence. `MediaLibrary(MediaSource)` opens by the
source's index, so a source from `GetAvailableMediaSources` works and one read
back from an open library does not — it carries no index. That is **refused with
a message naming which kind is which**, rather than silently opening the default
library. The stress slice proves both halves: it opens a library from an
enumerated source, and checks that the library's own source is refused.

## One route deliberately unbound

`cna_media_library_save_picture_from_stream` takes a native stream handle, and a
Go `io.Reader` has none — the same wall Foundation 92 met with `ContentReader`.
`SavePicture(String, Stream)` is projected over the BYTE route instead: the
stream is read into memory and the bytes handed on, which is what the
reference's two overloads differ by anyway.

The divergence is in the BUFFERING, not the outcome, and it is recorded on the
member and in `deliberatelyUnboundRoutes` as `SUBSUMED`. The reachability test
is what caught it: a bound route with no call site fails
`UNJUSTIFIED_BOUND_WITHOUT_CALL_SITE`.

## Falsifiability

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 19 | 19 | 0 |
| native | 6 | 6 | 0 |
| unqualified | 1 | 0 | 1 |

The first managed pass killed 17 of 26. Two survivors were real gaps:
`MediaSourceType`'s getter was only ever checked against the constant the test
itself used, so a constant implementation agreed; and `SavePicture`'s
three-way guard order was never checked with a DISPOSED library and a nil
stream, which is the only input that separates disposal-first from
argument-first.

The other seven needed a runtime and moved to the native table, where four more
slice gaps showed up: the source enumeration was never asserted non-empty, no
library was ever opened FROM an enumerated source, the root album's absent
parent was read and discarded, and the album walk had nothing to walk.

### The one defect this configuration cannot score

`root_album_reports_an_error_when_absent` needs a library with NO root picture
album, and the slice's isolated library always has one: it seeds pictures before
it opens the library, and a directory with pictures in it IS the root album.

The availability SHAPE is still covered —
`picture_from_token_ignores_availability` is killed by an invented token, which
really does name nothing — so what is unscored is this member's arm of the rule,
not the rule.

## A defect the tests found, again in the guard order

`SavePicture(String, Stream)` checked the runtime before the argument, so a nil
stream on a machine with no game answered "no runtime". The reference checks
disposal first and the argument next, and a nil stream is a programming error
that is true either way.

The member now runs three checks in a measured order — disposal, argument,
runtime — with the reason written at the call site. This is the second time in
two milestones that a Go-only runtime check was found sitting in front of an
argument check; `Song.FromUri` had the same defect in Foundation 95.
