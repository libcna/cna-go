# Foundation 98 — XACT

Five types: `AudioEngine`, `AudioCategory`, `SoundBank`, `WaveBank`, `Cue`. Over
58 bound CNA routes plus five subscription routes, and four deliberately left
unbound.

`BOUND_FUNCTIONS` 633 → 696, `COMPLETE_TYPES` 251 → 256, `MISSING_TYPE` 6 → 1.
**`Microsoft.Xna.Framework.Audio` is now complete**, and the only type left in
the whole 257-type profile is `GamerServicesComponent`.

## The recorded blocker was wrong, and measuring it twice is what found out

Foundation 97's frontier note said the XACT fixtures were reachable but EMPTY:

> an 80-byte file whose first four bytes are XGSF opens an engine, a 16-byte
> SDBK opens a sound bank and a 16-byte WBND opens a wave bank. Those are EMPTY
> banks — no cues, no categories, no waves — so construction, disposal, state
> and every refusal path are reachable and cue playback is not.

That was an honest measurement of the FLOOR — the smallest file each parser
will look past — and it was the wrong question. The right one is what the parser
accepts when the file is complete, and CNA answers it directly: its own XACT
demo ships a generator that writes all three formats, and reading it is
capability evidence of exactly the kind this project admits from CNA. It says
what CNA's parser accepts. It says nothing about what XNA means, and nothing in
this milestone's contract or behaviour came from it.

What the second measurement bought is the difference between a projection whose
`Cue` could never be played and one whose every native path executes. The
qualification log is the proof:

```
[AudioEngine] Loaded XGS: .../settings.xgs (2 categories, 1 variables)
[WaveBank]    Loaded XWB: .../waves.xwb bank="Waves" entries=2
[SoundBank]   Loaded XSB: .../sounds.xsb cues=2 sounds=2
```

Two authored categories, one settable global variable, two real PCM waves and
two named cues that play. `XACT_CUE_TRANSPORTS` is 5 and `XACT_APPLY_3D_CALLS`
is 2 where a floor fixture would have left both at zero.

**This is the seventh recorded blocker in Foundations 92–98 that re-measurement
overturned.** The pattern is now unmistakable: a blocker written from the
smallest thing that works is a note about the floor, not about the ceiling.

## The slice mutes before it plays, and proves it

`Cue.Play` reaches the machine's real audio output. Every authored category is
set to zero volume and the setting is counted — `XACT_MUTE_CHECKS` — before any
cue is played. A run whose mute count is zero has not earned its transports.

This is the third milestone to owe the host machine this courtesy and the third
to make it a measurement rather than a comment, after Foundation 96's
`MEDIA_HOME_CHECKS` and Foundation 97's `MEDIA_PLAYBACK_MUTE_CHECKS`.

## A value type that holds a handle

`AudioCategory` is `.class public sequential ansi sealed` — a CLR struct
implementing `IEquatable<AudioCategory>` — and the reference's copy of it holds
an engine reference and an INDEX. CNA has no index. Every `GetCategory` returns
a fresh OWNED handle, so the projection's value holds a handle, and three
consequences follow.

**Two lookups of one name are two handles that must compare EQUAL.** That is
what `cna_audio_category_equals` answers, and it is why this type's equality
reaches a runtime where the reference's compares two fields. The qualification
proves it: `GetCategory("Music")` twice, then `op_Equality`, and the run fails
if they differ.

**Nothing here destroys the handle.** A struct has no `Dispose` and the contract
declares none, so inventing one would add a member the contract does not have.
Each handle is registered as a CHILD of the engine and the engine's disposal
releases it. A game that looks a category up every frame accumulates handles
until then; that is a real cost and it is stated in the type's own
documentation rather than hidden.

**The ZERO value is legal and every member survives it.**
`default(AudioCategory)` is reachable in the reference, where it reaches a null
engine and throws `NullReferenceException`. Go cannot project that, so each
member answers a nil refusal — and the refusal is `errXactNil` and NOT a
disposal one, because a value that was never constructed was never disposed.

## The first struct whose every member is fallible

`isFallible`'s last line is `return t.Kind == "class" || t.Kind == "interface"`:
a CLR struct's members are infallible by default. That default is right for
every other struct in the profile, because every other struct holds its state in
its own fields.

`AudioCategory` holds a handle, so all eleven of its members reach XACT —
including the four a struct would normally answer from stored state,
`ToString`, `GetHashCode` and the two operators. The whole type is registered in
`runtimeReadMembers`, which is the first WHOLE-TYPE entry that registry has
carried.

## Four renderer routes deliberately unbound

CNA exports `cna_audio_engine_get_renderer_text_size`,
`cna_audio_engine_copy_renderer_text`, `cna_audio_engine_get_renderer_hash_code`
and `cna_audio_engine_renderers_equal`. All four are recorded as
`MANAGED_REFERENCE`, and the reason is the type they would answer for.

`RendererDetail` was projected in an earlier milestone as pure managed — two
string fields and five members that read them — and each of those three CNA
routes answers a different question:

- `ToString` is 17 bytes: box, then `ValueType::ToString`. It answers the CLR
  TYPE NAME and nothing about the renderer. A renderer text from CNA would be a
  second, different answer.
- `GetHashCode` is `(IsNullOrEmpty(_name) ? 0 : _name.GetHashCode()) ^
  (IsNullOrEmpty(_id) ? 0 : _id.GetHashCode())`, over the pinned mscorlib string
  hash. CNA cannot reproduce that, so binding it would replace a hash the
  projection computes CORRECTLY with one that is measurably wrong.
- `op_Equality` compares the two STRINGS and short-circuits on the name, so two
  renderers reporting the same name and id ARE equal however they are indexed.
  CNA's route compares two INDICES, which can only answer true for `i == i`.

So `AudioEngine.RendererDetails` binds exactly two routes — the friendly name
and the id — and computes the rest. Binding the other three would have made a
value type's identity depend on its position in a collection.

## The Disposing event is bound rather than raised locally

All four disposable XACT types declare `event EventHandler<EventArgs>
Disposing`, and the projection could have raised it from its own `Dispose`. It
does not, because destroying an `AudioEngine` disposes the banks and cues under
it: a projection that raised only its own `Dispose` would MISS every one of
those, and CNA is the only thing that knows they happened.

One trampoline serves all four types — `CNA_AudioEventCallback` carries only a
context — and the identity lives on the Go side in a `cgo.Handle` to the
subscription. The registration is released AFTER the native destroy, because the
destroy is what raises the event; releasing first would drop the notification
the caller subscribed for.

`XACT_DISPOSING_EVENTS` proves delivery rather than registration, and the slice
also removes one of two handlers and fails if the removed one runs.

## ContentVersion is 39, and the file header says 46

`AudioEngine.ContentVersion` is `public const int ContentVersion = 39` in the
pinned contract. CNA's parser accepts `46` in an `.xgs` header, and this
project's fixture writer writes 46.

Both are true of different things: 39 is what this XNA build's constant says,
and 46 is what the file format wants. The projected member reports the
CONTRACT's value, and a planted defect that made it report 46 is killed.

## Falsifiability

Thirteen focused defects were planted one at a time and each was killed by a
named test; the harness refuses to report unless every mutated file is
byte-identical afterwards, and it reported `RESTORED_CLEAN`.

| defect | killed by |
| --- | --- |
| the flattened emitter drops its leading DopplerScale | `TestFlatteningMatchesTheOrderTheBridgeFills` |
| the flattened listener swaps Up and Forward | `TestFlatteningMatchesTheOrderTheBridgeFills` |
| the disposal refusal drops the type name | `TestADisposedXactObjectRefusesAndNamesItsOwnType` |
| ContentVersion reports the file-format number | `TestAudioEngineContentVersionIsTheContractsAndNotTheFileFormats` |
| the finalizer path walks the managed children | `TestDisposalIsIdempotentAndTheFinalizerPathSkipsChildren` |
| a zero AudioCategory reaches a handle | `TestEveryZeroAudioCategoryMemberThatNeedsAHandleRefuses` |
| Equals(object) matches a foreign value | `TestTheZeroAudioCategoryIsLegalAndAnswersWithoutAHandle` |
| Apply3D accepts a nil listener | `TestApply3DRefusesANilArgumentBeforeReachingTheRuntime` |
| the bank constructor skips the engine check | `TestABankConstructorRefusesADisposedEngineByName` |
| the settings variable is authored read-only | `TestTheSettingsFixtureHasTheHeaderTheParserSeeksPast` |
| a cue's sound code is relative, not absolute | `TestTheSoundBankFixturePointsEachCueAtTheSoundItNames` |
| the wave data offset loses its alignment | `TestTheWaveBankFixtureAddressesItsWaveDataWhereItsHeaderSaysItIs` |
| the sound bank forgets the wave bank it names | `TestTheSoundBankFixturePointsEachCueAtTheSoundItNames` |

The fixture writers are tested at all because a wrong header does not fail
loudly: XACT's parsers report through CNA's LOG, never through the result code,
so a malformed file becomes a refusal the slice records and moves past. Without
those four tests a broken layout would show up as a silently skipped slice with
every counter at zero.

Two of the thirteen are worth naming for what they nearly hid. A cue's sound
code is an ABSOLUTE file offset; a relative index is a small number that still
parses. And the sound record's wave index sits at byte 9, so reading it a byte
early answers 256 for a wave index of 1 — which is why the test asserts the
SECOND cue too, since the first reads correctly at either offset because both
its bytes are zero.

## What is NOT proved

Silence is not audibility. Every category is muted before a cue is played, so
what the qualification shows is that transport, positioning and state reach XACT
and come back — not that anything sounded right.

The fixtures are this project's own, written to the layout CNA's parser accepts.
They are not compiled by Microsoft's XACT tool, and no claim is made that a
real authored bank behaves identically.
