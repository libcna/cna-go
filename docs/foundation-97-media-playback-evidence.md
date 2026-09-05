# Foundation 97 — media playback

Four types: `MediaPlayer`, `MediaQueue`, `Video`, `VideoPlayer`. Over 55 CNA
routes.

`BOUND_FUNCTIONS` 581 → 637, `COMPLETE_TYPES` 247 → 251. **The whole
`Microsoft.Xna.Framework.Media` namespace is now projected.**

## The slice mutes before it plays, and proves it

`MediaPlayer` plays through the machine's real audio output. A test is not
entitled to make noise on someone's desktop, so the slice's first act is to set
`IsMuted`, read it back, set the volume to zero and read THAT back —
`MEDIA_PLAYBACK_MUTE_CHECKS` counts it — and nothing is played until both have
passed. The volume is set as well as the mute so that a host ignoring one flag
still stays silent.

This is the same shape as Foundation 96's `MEDIA_HOME_CHECKS`: a courtesy the
machine is owed, made into a measurement rather than a comment.

## A static class projects to a marker type plus package functions

All twenty `MediaPlayer` members are static. The contract declares no
constructor and no instance member, so there is no value to project — what a
consumer reaches is `MediaPlayer struct{}` carrying the CLR identity plus
type-prefixed package declarations. That is the settled shape `TitleContainer`
and `FrameworkDispatcher` already have.

The setters take a `Set` PREFIX rather than a suffix — `SetMediaPlayerVolume`,
not `MediaPlayerSetVolume` — which the verifier dictated and which reads the way
the CLR's `set_Volume` does.

## The first STATIC events in the profile

`ActiveSongChanged` and `MediaStateChanged` are `public static event
EventHandler<EventArgs>`. There is no instance to hang a registration list on,
so the lists are process-wide, exactly as the reference's own static delegate
fields are.

The native half forces one shape the reference does not have. CNA's two
subscribe routes each answer a registration handle and raise through a
trampoline carrying only a context, so the projection subscribes the FIRST time
a handler is added to an event and releases that registration when the LAST one
goes. Adding a second handler does not subscribe twice. The observable
behaviour is the reference's — a handler added is called, a handler removed is
not — and the difference is recorded where it lives.

## Video is complete and unreachable

The contract declares no constructor and no static factory. In XNA a `Video`
arrives from `ContentManager.Load<Video>`, and that cannot be bound: CNA has
load routes for textures, effects, sprite fonts, sound effects, texture cubes
and models, and **none for video**.

CNA *can* build one — `cna_video_create(graphics_device, file_name, …)` — and
exposing that would be inventing a constructor the contract does not declare,
which is the line the Model family holds. So the type is complete and
unreachable, and the reason is on the type.

`VideoPlayer` does declare a constructor, so a consumer can build one and then
has nothing to give it. Every other member works on a player with no video, and
the slice exercises all of them.

## Two types with no disposal, and their contracts agree

`MediaQueue` and `Video` declare no `Dispose`, no `Finalize` and no
`IsDisposed`. CNA agrees for the queue: `cna_media_player_get_queue` answers a
BORROWED handle, a view of the process-wide queue rather than an object the
caller owns. So neither projection carries a latch, and the shape says what the
contract says.

## The guard order, applied consistently at last

Foundation 95 found a Go-only runtime check sitting in front of an argument
check in `Song.FromUri`. Foundation 96 found the same in
`MediaLibrary.SavePicture`. This milestone's first mutation run found it in
**five more members** — every `MediaPlayer.Play` overload,
`GetVisualizationData` and the event add accessors — because the pattern had
been copied rather than the rule applied.

All of them now check the argument first, and `VideoPlayer.Play` checks
disposal, then the argument, then the runtime. The test that asserted the old
order was rewritten: it had been rationalising the code rather than measuring
the reference.

## Falsifiability

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 20 | 20 | 0 |
| native | 8 | 8 | 0 |
| unqualified | 2 | 0 | 2 |

The first managed pass killed 11 of 22 with 5 that would not compile. Reordering
the guards made eleven of those reachable; the rest moved to the native table.

### The native table needed the slice to ASSERT, not to call

Its first run killed 3 of 10. The seven survivors were all the same defect in
the slice: it invoked the transport and never checked where it landed.

`Pause`/`Resume` and `MoveNext`/`MovePrevious` are each other's inverse, so a
projection wiring one to the other returns success and lands in the wrong
place. The slice now checks the player's STATE after each transition and the
queue's ACTIVE INDEX around the two moves. The queue's bound and the disposed-
song refusal are matched on the projection's own message, because CNA refuses
those too and a slice relying on that would not notice the projection's guard
disappearing.

### The message match, and why it is not a sentinel

Exporting `ErrMediaArgumentOutOfRange` and `ErrMediaDisposed` made the checks
cleaner and put two members in the media package's public surface that the XNA
contract does not declare. `TestPackedVectorCurrentSurfaceAndConformance`
refused it, and it was right: a test's convenience is not a reason to widen an
API. The slice matches the message instead.

### Two defects this host cannot score

`visualization_writes_only_frequencies` and
`visualization_swaps_the_two_buffers` both need the visualization backend to
REPORT something. This host's reports silence — `MEDIA_VISUALIZATION_SILENT`
counts exactly that — and with all 512 values at zero, writing one buffer and
swapping the two produce the same answer.

The slice already reads the buffers WHILE something is playing, which is the
only time there could be data, so a host whose backend fills them would score
both.
