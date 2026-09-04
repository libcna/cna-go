# Foundation 87 — SoundEffect and SoundEffectInstance

```text
COMPLETE_TYPES   185 -> 187        MISSING_TYPE       72 -> 70
BOUND_FUNCTIONS  305 -> 327        DELIBERATELY_UNBOUND_ROUTES  +17
RESOURCE_STRINGS  59 ->  68
```

## Reference authority

```text
Microsoft.Xna.Framework.dll   38e7093f52d7474b...
mscorlib (pinned .NET Framework 4.0)   for TimeSpan::Interval and FromMilliseconds
```

The two types had to land together: the pinned contract declares **no public
constructor** for `SoundEffectInstance`. One comes from `CreateInstance` or from
`Play`'s pool, and nothing else.

## The measurement that decided the milestone

**`GetSampleSizeInBytes` does not return round numbers**, because the scale
factor is computed in float32:

```text
samples = (int)(duration.TotalMilliseconds * (double)((float)SampleRate / 1000f))
```

`(float)44100 / 1000f` is `44.099998474121094`, not `44.1`, because 44.1 has no
binary32 representation. One second therefore scales to `44099.998...` and
**truncates to 44099 samples**.

| one second, mono | bytes | why |
| --- | ---: | --- |
| 8000 Hz | 16000 | `8000/1000` is `8.0`, exact in binary32 |
| 48000 Hz | 96000 | `48.0`, exact |
| 44100 Hz | **88198** | truncates to 44099 samples — two bytes short of 88200 |
| 22050 Hz | **44098** | truncates to 22049 samples |
| 44100 Hz **stereo** | 176400 | the round number, and not because the truncation went away |

That last row is the clearest thing in the milestone. 44099 is ODD, so
`samples % Channels` is 1 and the alignment step **adds one sample back** — which
is what makes the step an addition rather than a round-up. At mono the remainder
is always zero and the truncation stands.

The float32 arithmetic decides the DURATION too, and there by a whole
millisecond rather than a rounding bit:

```text
269462 bytes at 11025Hz mono
    float32:  (float)134731 * 1000f / (float)11025  ->  12221 ms  ->  122210000 ticks
    float64:  134731.0 * 1000 / 11025.0             ->  12220 ms  ->  122200000 ticks
```

Every one of the five byte counts round-trips to exactly one second, because
`TimeSpan.FromMilliseconds` rounds to a whole millisecond — so every duration
this family produces has a tick count that is a multiple of 10000.

## No two static setters validate the same way

Four scalars, four different shapes, none guessable from the signature:

| setter | guard | zero | NaN | order |
| --- | --- | --- | --- | --- |
| `MasterVolume` | `blt.un 0` / `bgt.un 1` | legal | **throws** | native, then store |
| `SpeedOfSound` | `ble.un 0` | **throws** | throws | store, then native |
| `DopplerScale` | `blt.un 0` | **legal** | throws | store, then native |
| `DistanceScale` | `bge.un 0` | clamped | **stored** | store, then native |

`DistanceScale` is the odd one twice over. Its guard branches with `bge.un`,
which jumps PAST the throw when the comparison is unordered, so a NaN is
accepted there and refused by the other three; and the clamp that follows is an
ORDERED `ble`, which a NaN fails, so the NaN survives the clamp and is stored.
Zero and everything below `Single.Epsilon` becomes `Single.Epsilon` **silently** —
a consumer who sets zero and reads the getter back does not get zero.

The store order matters too, and the two ends disagree: `MasterVolume` writes
native first, so a failed call leaves the getter reporting what is in effect;
`SpeedOfSound` stores first, so a failed call leaves it already updated.

## Three "before the first Play" preconditions, and one flag behind all of them

```text
set_Pan     if (!isPacketSubmitted) is3d = false;  if (is3d)  throw InvalidPanCall
Apply3D     if (!isPacketSubmitted) is3d = true;   if (!is3d) throw InvalidApply3DCall
set_IsLooped                                       if (isPacketSubmitted) throw InvalidIsLoopedCall
```

`set_Pan` and `Apply3D` are exact mirrors: before the packet is submitted,
whichever is called decides the instance's mode; afterwards the member for the
other mode refuses. CNA describes the same rule from its own side, under the
heading *"Aim before you play"*.

**The native run found two defects here that no managed test could have.**
`set_IsLooped` had no guard at all, and `isPacketSubmitted` was never set — which
also meant the mode latch could never latch. CNA surfaced it by refusing
`set_is_looped` with `INVALID_STATE` "after playback has begun", exactly as the
reference does. `Play` now sets the flag on success, which is the moment CNA
names: *"the reference implementation submits its audio packet on the first
`..._play`"*.

## Twenty-two routes bound, seventeen recorded unbound

Two CNA routes serve four XNA members, because the reference makes the same
reduction:

- `cna_sound_effect_create_pcm16_range_ext` — "the canonical seven-argument
  constructor". XNA's three-argument constructor forwards to `FromBuffer` with a
  zero offset, the whole length and a zero loop region.
- `cna_sound_effect_instance_apply_3d_multi_ext` — XNA's single-listener
  `Apply3D` is 20 bytes that build a one-element array and forward.

Of the seventeen recorded unbound, the class that matters is **`REDUNDANT_READ`**:
`IsDisposed`, `Name`, `Duration` and the four scalar getters are all `ldfld` or
`ldsfld` in the reference, and `get_Name` has **no disposal check** — it answers
after `Dispose`, which a native read of a released handle cannot do.

`Duration` is the one that goes the other way: the reference stores it at
construction, but what it stores is derived from the buffer the RENDERER
received, so the projection reads CNA's ticks once and stores those. Same choice
`VertexBuffer` made for its stride.

## Two internal packages, and why neither is public surface

`FrameworkDispatcher.UpdateCalledAtLeastOnce` is `assembly` and not in the
contract, and `SoundEffect::Play`'s FIRST statement reads it — from another
namespace. Go cannot share an unexported symbol across packages, so it lives in
**`internal/dispatcher`**. Exporting an accessor would have added a member the
contract does not declare; duplicating the flag would have given one process two
answers.

That is the second time this session: `internal/bclhash` arrived in Foundation
86 for the same reason.

## Playing SILENCE, deliberately

Every native fixture is an all-zero PCM16 buffer. The qualified artifacts open a
REAL playback device — `cna_audio_get_capabilities` reports
`is_playback_available = true` on both — so a fixture with signal in it would
make audible noise on the machine running the suite, twenty times per cycle, for
no evidence gained.

Silence exercises creation, instance lifetime, transport, state and the scalar
round trips identically: every one of them is about STRUCTURE and none reads a
sample back. What silence cannot prove is audibility, and this scenario does not
claim it.

## Native qualification

```text
                                    HEADLESS   SOFTWARE
SOUND_EFFECT_CREATIONS                    20         20
SOUND_EFFECT_CREATION_REFUSALS             0          0
SOUND_EFFECT_DESCRIPTION_CHECKS           20         20
SOUND_EFFECT_GUARD_CHECKS                 20         20
SOUND_EFFECT_SCALAR_CHECKS                20         20
SOUND_EFFECT_PLAYS                        20         20
SOUND_EFFECT_PLAY_LIMIT_CHECKS             0          0
SOUND_INSTANCE_CREATIONS                  20         20
SOUND_INSTANCE_TRANSITIONS               100        100
SOUND_INSTANCE_SCALAR_ROUND_TRIPS         60         60
SOUND_INSTANCE_MODE_LATCH_CHECKS          20         20
SOUND_INSTANCE_APPLY_3D_CHECKS            20         20
SOUND_EFFECT_DISPOSAL_CHECKS              20         20
```

Both artifacts create the effect, read the duration CNA computed, run all five
transport transitions, round-trip the scalars, exercise the mode latch, position
the instance in 3D, and release the effect with a live instance still attached.

## Planted defects

49 distinct defects, each a real way to get this wrong and each compiling.

```text
PLANTED   49      KILLED   43      SKIPPED   0
```

Six needed an assertion the first pass did not have, and three are worth naming
because each is a different kind of gap:

| defect | why the first assertion missed it |
| --- | --- |
| `resume_calls_play` | Resume and Play agree everywhere EXCEPT on a stopped instance. The probe measured that a stopped instance which is resumed stays Stopped, so the assertion is a Stop-then-Resume pair — and every other transport check passes under the mutant |
| `duration_from_size_scales_in_float64` | every size in the first test agreed between float32 and float64 after the millisecond rounding. A case where they differ by a whole millisecond had to be searched for: 269462 bytes at 11025Hz |
| `state_falls_back_to_playing` | CNA reports only its three defined identities, so the fallback branch NEVER EXECUTED. The mapping moved into its own function so a test can reach the branch — a mutation of a path that never runs is not a killed mutation |

### Five are not killed, and each is named

| defect | why |
| --- | --- |
| `master_volume_stores_before_the_native_call` | the order is observable only when the native call FAILS, and nothing in the qualified environment makes the mixer refuse a legal volume |
| `dispose_is_not_idempotent` | `interop.Resource.Dispose` carries its own guard, so the managed flag is redundant against this backend. EQUIVALENT here and load-bearing against one that is not |
| `creation_ignores_the_offset` | the offset changes which bytes are read and nothing in the contract reads them back. `Duration` follows the COUNT, which is asserted, and the count is a different argument |
| `creation_ignores_the_loop_region` | the loop region has no public reader at all. The reference stores it privately and only the mixer acts on it |
| `instance_volume_is_never_pushed` | XNA's getter is a managed field, so it reports the cached value whether the push happened or not. CNA's info structure carries the volume, and no contract member reads it |
| `apply_3d_sends_the_listener_fields_in_the_wrong_order` | four vectors that CNA consumes and nothing reads back. The boundary a spatial-audio readback would move |

The last four are the audio family's version of the boundary Foundation 85
moved for graphics: a push whose only observable is what the mixer does with it.

## What is proved, and where

```text
sound_effect_test.go          8 tests: the class initializer's five values, all
                              four static setters' distinct shapes including the
                              NaN each accepts or refuses, the Epsilon clamp, the
                              maxVelocity recomputation, both conversions' guard
                              orders, and the MEASURED byte counts with the
                              float32 divergence that produces them
sound_effect_instance_test.go 10 tests: the constructor's three stores, all three
                              scalar ranges, the pan/Apply3D mode latch from both
                              ends, the IsLooped packet guard, the packet that a
                              FAILED Play must not submit, the state mapping's
                              unreachable fallback, and every disposal refusal
framework_dispatcher_flag_test.go  the flag Update sets BEFORE it can refuse
native_stress vertex-buffer   20 cycles on two artifacts: creation from PCM,
                              CNA's own duration, three constructor forms, the
                              four scalars pushed and read back, a fire-and-forget
                              Play, five transport transitions including the
                              Stop-then-Resume that tells Resume from Play, the
                              scalar round trips, the mode latch, Apply3D, and an
                              effect released with a live instance attached
external canary               1 test compiling both types from outside, including
                              the measured 88198
```

```text
FOUNDATION_MILESTONE_87_COMPLETE=true
```
