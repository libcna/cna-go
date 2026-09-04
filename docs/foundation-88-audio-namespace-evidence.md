# Foundation 88 — DynamicSoundEffectInstance, Microphone, and the end of the Audio namespace

```text
COMPLETE_TYPES   187 -> 189        MISSING_TYPE       70 -> 68
BOUND_FUNCTIONS  327 -> 342        DELIBERATELY_UNBOUND_ROUTES  +6
COMPOSED_XNA_BASES 8 ->   9        IDENTITY_SITES     15 -> 16
```

**`Microsoft.Xna.Framework.Audio` is complete.** Nineteen types, and the last two
are the ones that needed the other seventeen first.

## Reference authority

```text
Microsoft.Xna.Framework.dll   38e7093f52d7474b...
```

## A streaming instance can never loop, and the reference says so twice

`DynamicSoundEffectInstance` OVERRIDES four members of its base, and two of them
are pure refusal:

```csharp
get_IsLooped   if (IsDisposed) throw new ObjectDisposedException(GetType().Name, ...);
               return false;                     // ALWAYS, `ldc.i4.0`

set_IsLooped   if (IsDisposed) throw new ObjectDisposedException(...);
               if (value) throw new InvalidOperationException(
                              FrameworkResources.InvalidDynamicIsLoopedCall);
               // setting FALSE stores nothing and returns
```

The getter answers a CONSTANT and the setter accepts only the value the getter
already reports. A consumer who sets false and reads it back sees no change
because there was none to make — and the base's own `looped` field is never
touched, which is what "stores nothing" means.

That is the composed-base virtual-dispatch problem in its plainest form, and it
is what made `SoundEffectInstance` the **first composed base outside the
graphics namespace**. Its identity site is `objectDisposed`, which every member
in the family reaches: the reference pushes `GetType().Name`, so a disposed
`DynamicSoundEffectInstance` must name itself. The literal that stood there
carried a note saying it would become an identity site the moment the derived
type arrived — this milestone is that moment.

## A defect the native assertion found, and the fix

The stress slice asserted the number the REFERENCE produces and CNA answered a
different one:

```text
one second at 22050Hz mono
    the reference  44098 bytes   float32 scale factor -> 22049 samples
    CNA            44100 bytes   the exact arithmetic
```

Both are the same question and they differ by one frame, because XNA computes
its scale factor in float32 and CNA does not.

So all **four** sample-conversion routes were bound end to end and then
**reverted**, and both types now compute the conversion with the managed
`AudioFormat` arithmetic — the format the reference builds in each case:

```text
DynamicSoundEffectInstance   AudioFormat.Create(sampleRate, channels, 16)
Microphone                   AudioFormat.Create(GetSampleRate(), 1, 16)
```

Capture is MONO, read off the constructor's IL rather than assumed.

**The fixture's sample rate is load-bearing and the scenario says so.** 22050
was chosen over 8000 *because* its answer is distinctive: `8000/1000` is exact
in binary32, so the reference and CNA both answer 16000 there and the divergence
is invisible. A fixture at 8000 would have agreed with the wrong projection.

This is the third time this session a native assertion caught a projection
defect no managed test could have. Foundation 87 found `set_IsLooped`'s missing
guard and its never-set packet flag; this one found a conversion answering the
renderer's number instead of the reference's.

## Microphone: projected in full, capture never performed

```text
MICROPHONE_CAPTURE_CALLS                   0          0
```

`Start` and `GetData` are projected because the pinned contract declares them.
The suite calls **neither**, and the counter above exists to be zero and to say
so out loud — the parent accounting FAILS the run on a non-zero value, because a
non-zero value is a run that began recording on someone's machine.

What is exercised is everything else: the count, the default, each device's
name, buffer duration, headset flag, sample rate and state, both conversions,
and every managed guard. `Stop` is called because stopping a device that is not
capturing is the reference's own no-op.

This machine reports three microphones and index 0 is named "Default Device".
The standing constraint against opening it is satisfied by never starting
capture at all rather than by skipping an index.

### Two shapes the verifier forced, and one it could not

| member | shape | why |
| --- | --- | --- |
| `Name` | an exported Go FIELD | the contract declares it `kind: field`, not a property |
| `BufferDuration` | infallible | the reference's getter is one `ldfld`; it needed a `managedStoredMembers` entry to say so |

`Name` being a field has two consequences Go cannot avoid. The `initonly` half
has no counterpart, so a consumer can assign to it where the reference would not
compile; and a field cannot carry a nil guard, so reading it through a nil
pointer panics where every method on the type answers. Both are recorded in the
source and pinned by a test rather than worked around — hiding the field behind
a getter would have projected a property the contract does not declare, which is
the larger divergence.

### `set_BufferDuration` has one `if` with three parts

```csharp
if (value.TotalMilliseconds < 100 || value.TotalMilliseconds > 1000 ||
    value.TotalMilliseconds % 10 != 0)
    throw new ArgumentOutOfRangeException("value", InvalidMicrophoneBufferDuration);
```

100ms to 1s INCLUSIVE and aligned to TEN milliseconds — so 110 and 250 are legal
and 105 is not. The message carries a double space before "10ms", which is in
the resource and is reproduced byte for byte.

### One member, two exception KINDS

```text
Microphone::GetSampleDuration(int)         ArgumentException(InvalidBufferSize)
SoundEffect::GetSampleDuration(int,int,ch) ArgumentOutOfRangeException("sizeInBytes", InvalidBufferSize)
```

Two members that read the same and carry the SAME message throw different kinds.
A projection that shared one helper between them would be wrong for one of the
two, and nothing in the signatures says which.

## A signature the compile-time check caught

`cna_microphone_get_default_index_ext` takes a THIRD parameter,
`CNA_Bool* out_available`, because a machine with microphones need not have a
default: the index is "left unchanged when there is no default". The manifest's
prototype check refused to compile the wrong declaration, which is exactly what
it is for. `MicrophoneDefault` answers a nil `Microphone` and no error in that
case, which is what the reference's `null` means.

## Native qualification

```text
                                    HEADLESS   SOFTWARE
DYNAMIC_INSTANCE_CREATIONS                20         20
DYNAMIC_INSTANCE_REFUSALS                  0          0
DYNAMIC_INSTANCE_LOOP_CHECKS              20         20
DYNAMIC_INSTANCE_SUBMISSIONS              20         20
DYNAMIC_INSTANCE_PENDING_CHECKS           20         20
DYNAMIC_INSTANCE_CONVERSION_CHECKS        20         20
DYNAMIC_INSTANCE_DISPOSAL_CHECKS          20         20
MICROPHONE_ENUMERATIONS                   20         20
MICROPHONES_FOUND                         60         60
MICROPHONE_DESCRIPTION_CHECKS             60         60
MICROPHONE_GUARD_CHECKS                   20         20
MICROPHONE_CAPTURE_CALLS                   0          0
```

## Planted defects

29 distinct defects, each a real way to get this wrong and each compiling.

```text
PLANTED   29      KILLED   21      SKIPPED   0
```

Four needed an assertion the first pass did not have, and two of the four are
the same lesson in different clothes:

| defect | why the first assertion missed it |
| --- | --- |
| `dynamic_is_looped_reports_the_base_field` | the fixture never set the base's `looped`, so forwarding to it and returning a constant gave the same answer. Setting it and asserting false anyway is what separates them |
| `submit_buffer_uses_the_wrong_channel_count` | the fixture was MONO, whose BlockAlign is 2 — the same value the mutant hardcodes. A STEREO instance aligns to four, and a two-byte buffer it must refuse |
| `buffer_duration_aligns_to_100ms` | every accepted value in the first test was a multiple of 100. 110 and 250 are legal and distinguish the two alignments |
| `dynamic_ctor_checks_channels_first` | a call with both arguments wrong reports whichever guard runs first |

### Eight are not killed, and they have ONE root cause between them

| defect | why |
| --- | --- |
| `buffer_duration_stores_before_the_native_call` | the order is observable only when the native call FAILS, and nothing in this environment makes the mixer refuse a legal duration |
| `math_mod_truncates_the_wrong_way` | EQUIVALENT for the only test this helper does. The mutant's remainder differs in SIGN and magnitude but is zero exactly when the correct one is |
| `dynamic_ctor_sends_the_wrong_sample_rate` | a consequence of the correct fix above: the conversions are now MANAGED, so what CNA built is no longer read back through any contract member |
| `submit_buffer_ignores_the_offset` | the submitted bytes have no reader. CNA copies them into its mixer and nothing in the contract asks what arrived |
| `microphone_all_stops_at_the_first_device` | the collection's own `Count` is the only count the public surface has, so a projection that built one entry reports one entry consistently |
| `microphone_default_ignores_availability` | this machine HAS a default, so the branch that matters is not taken |
| `microphone_name_is_never_read` | the name has no independent source of truth through the contract. An empty name is also a legal answer for a real device |
| `microphone_sample_rate_uses_the_wrong_index` | **this machine's three microphones all report 44100Hz**, so an index mix-up answers the same number |

The last four share one cause worth stating plainly: **the microphone family's
reads have no independent source of truth in the contract, and this machine's
devices are homogeneous.** A machine with mixed-rate or differently-named
devices would move three of them, and that is a property of the environment
rather than of the suite.

## What is proved, and where

```text
dynamic_sound_effect_instance_test.go  5 tests: both constructor guards and
                                       their order, the two OVERRIDDEN accessors
                                       from both ends including the base field
                                       they do not read, the identity that must
                                       name the derived type, the four shared
                                       window checks including a STEREO
                                       alignment, and the conversion guards
microphone_test.go                     6 tests: the three-part buffer-duration
                                       guard with both inclusive bounds and the
                                       10ms alignment, the exception KIND that
                                       differs from SoundEffect's sibling, the
                                       state fallback, GetData's window checks,
                                       the Name field's two language
                                       limitations, and every nil receiver
native_stress vertex-buffer            20 cycles on two artifacts: a streaming
                                       instance created, its refusal to loop, a
                                       submission raising a LIVE pending count,
                                       the conversion asserted at the
                                       reference's own 44098, disposal naming
                                       the derived type; and every microphone
                                       enumerated and described with ZERO
                                       capture calls
external canary                        1 test compiling both types from outside,
                                       including the Name field a consumer can
                                       write
```

```text
FOUNDATION_MILESTONE_88_COMPLETE=true
```
