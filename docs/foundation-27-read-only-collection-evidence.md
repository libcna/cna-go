# Foundation 27 — `ReadOnlyCollection<T>` as a measured signature adapter

Foundation 26 answered how a BCL collection base projects. Foundation 27
answers the other half of the same family: what happens when the pinned
contract carries a BCL collection **in a signature** rather than as a base.

> A BCL type the pinned XNA contract carries at a **public signature position**
> needs a public Go spelling, or the member that returns it cannot be projected
> at all.

That is not a new footing. `System.TimeSpan` maps to `framework.TimeSpan` and
`System.EventHandler<T>` to `framework.EventHandler[T]` for exactly this
reason. `System.Collections.ObjectModel.ReadOnlyCollection<T>` joins them as
`*framework.ReadOnlyCollection[T]`.

**The two adapter roles are distinct and neither implies the other.** A *base*
adapter is private machinery a derived type composes and forwards. A
*signature* adapter is a public type because a projected member returns one.
`ReadOnlyCollection<T>` is now `SUPPORTED` as a signature adapter and remains
`DEFERRED` as a base, and the verifier asserts both facts.

## Reference authority

```text
Microsoft.Xna.Framework.dll  38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
mscorlib.dll                 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
```

The mscorlib is the same pinned .NET Framework 4.0 RTM binary Foundation 26
established, version 4.0.30319.1, the identity every XNA `.assembly extern
mscorlib 4.0.0.0` names.

## What the type is

```text
.class public auto ansi serializable beforefieldinit
       System.Collections.ObjectModel.ReadOnlyCollection`1<T>
{
  .field private class System.Collections.Generic.IList`1<!T> list
```

One field, and every public member forwards to it:

```text
get_Count       ldfld list; callvirt ICollection`1::get_Count
get_Item        ldfld list; ldarg.1; callvirt IList`1::get_Item
Contains        ldfld list; ldarg.1; callvirt ICollection`1::Contains
CopyTo          ldfld list; ldarg.1; ldarg.2; callvirt ICollection`1::CopyTo
IndexOf         ldfld list; ldarg.1; callvirt IList`1::IndexOf
GetEnumerator   ldfld list; callvirt IEnumerable`1::GetEnumerator
```

The constructor **stores** the list; it does not copy it:

```text
.ctor(IList`1<!T> list)
  ldarg.1; brtrue.s L            // null -> ArgumentNullException
  ldarg.0; ldarg.1; stfld list
```

### Read-only means no public mutation, not frozen data

This is the semantic the previous handoff flagged, and the evidence settles it
in the direction the flag warned about. Because the view stores the list, the
owner keeps writing and **every change is visible through the view**. Freezing
a copy would be a different type, and CNA-Go does not freeze one.

Read-only also needed **no new decision**. Every mutator is a private explicit
implementation:

```text
.method private ... 'System.Collections.Generic.ICollection<T>.Add'
.method private ... 'System.Collections.Generic.ICollection<T>.Clear'
.method private ... 'System.Collections.Generic.ICollection<T>.Remove'
.method private ... 'System.Collections.Generic.IList<T>.Insert'
.method private ... 'System.Collections.Generic.IList<T>.RemoveAt'
.method private ... 'System.Collections.Generic.IList<T>.set_Item'
```

so the settled BCL-interface rule already excludes them, and the public
indexer has a getter and no setter. The projected surface is exactly six
members and every type in it was already decided — `int32`, `T`, `[]T`, and the
settled `Iterator[T]`.

### The view is bound to the list instance

In the CLR the view holds the array **reference** it was handed. An owner that
later points its own field at a different array does not change what an
existing view shows, while writes into the captured array are visible
immediately.

A captured Go **slice header** reproduces both halves exactly. A `*[]T` would
be *more* live than the reference, and a copy would be less. Both directions
are tested.

### Enumeration semantics belong to the list, not the view

`GetEnumerator` forwards, so the enumerator a caller observes is the underlying
list's. That makes the mutation policy source-dependent, and the difference is
real:

| source | enumerator | version check |
| --- | --- | --- |
| `List<T>` | `List<T>.Enumerator` | yes, `_version`, checked before bounds |
| `T[]` | `SZArrayHelper.SZGenericArrayEnumerator<T>` | **no** |

The array enumerator holds only `_array`, `_index` and `_endIndex`:

```text
MoveNext   ldfld _index; ldfld _endIndex; bge.s <false>
           _index++; ldfld _index; ldfld _endIndex; clt; ret
```

An array cannot change length, so there is no version to check, and an element
written during enumeration is simply observed. CNA-Go reproduces that rather
than adding an invalidation the reference does not have — which is why the
declared `mutationPolicy` now says the policy belongs to the list.

The reference array enumerator's other difference — its `get_Current` throws
`InvalidOperationException` before the first step and after the last, where
`List<T>.Enumerator.get_Current` is one unvalidated `ldfld` — is
unrepresentable through `Iterator[T]`, which fuses `MoveNext` and `Current`.
Neither state can be reached and neither is invented.

### Element equality is the BCL's, not Go's

`Contains` and `IndexOf` reach `EqualityComparer<T>.Default`. For
`System.Single` the BCL selects `GenericEqualityComparer<Single>`, which calls
the strongly typed `Equals`:

```text
System.Single::Equals(float32 obj)
  ldarg.1; ldarg.0; ldind.r4; bne.un.s L    // equal -> true
  ldarg.1; call IsNaN; brfalse.s L2
  ldarg.0; ldind.r4; call IsNaN; ret        // both NaN -> true
```

**NaN equals NaN**, where Go's `==` reports false. A search over a
`ReadOnlyCollection<float>` therefore finds a NaN element, and `singleEquals`
reproduces that. It is the equality counterpart of the existing
`compareSingle`, which already records the same NaN treatment for
`Single::CompareTo`. Signed zeros stay equal in both languages and need no
special case.

## `Media.VisualizationData`

The first consumer, and it is completed by this milestone. The whole type is
four fields and two one-`ldfld` getters:

```text
.ctor()
  ldc.i4 0x100; newarr System.Single -> frequencies
  ldc.i4 0x100; newarr System.Single -> samples
  newobj ReadOnlyCollection`1<float32>(IList`1<!0>)  over each

get_Frequencies   ldfld frequenciesCollection
get_Samples       ldfld samplesCollection
```

It validates nothing, reaches no device and touches no native code, so it is
admitted as pure managed and every member is infallible except the inherited
indexer's bounds failure.

The two arrays are `assembly` fields — not public surface. They are what the
media backend writes into, and the public views are live over them, so a caller
holding `Frequencies` sees each refresh without asking again.

Completing it starts no playback. CNA-Go has no `MediaPlayer`, no `Song`, and
no media backend, so nothing ever writes into the buffers and both views stay
256 zeros for their whole lifetime.

## The adapter unblocks six signature positions

```text
Microsoft.Xna.Framework.Audio.AudioEngine::RendererDetails
Microsoft.Xna.Framework.Audio.Microphone::All
Microsoft.Xna.Framework.Graphics.GraphicsAdapter::Adapters
Microsoft.Xna.Framework.Graphics.SpriteFont::Characters
Microsoft.Xna.Framework.Media.VisualizationData::Frequencies
Microsoft.Xna.Framework.Media.VisualizationData::Samples
```

Only `VisualizationData` is completed by it. The other four owners stay blocked
on device enumeration, XACT, or content, which the regenerated frontier
confirms rather than assumes.

## Why `ReadOnlyCollection<T>` is still deferred **as a base**

Its four base consumers are `ModelBoneCollection`, `ModelEffectCollection`,
`ModelMeshCollection` and `ModelMeshPartCollection`. Each is blocked twice
over, and the second blocker is a projection question rather than a dependency:

1. **Element types.** Each needs `ModelBone`, `Effect`, `ModelMesh` or
   `ModelMeshPart`, all of which are missing and content-pipeline blocked.
2. **Member hiding.** Each *declares* a `GetEnumerator` returning its own
   nested `Enumerator`, which **hides** the inherited `GetEnumerator`. The
   settled collision rule would resolve that mechanically into two hashed
   names, neither of which is `GetEnumerator` — a poor answer for the primary
   enumeration entry point.

   The principled answer is available and is recorded rather than taken: a
   hidden inherited member is unreachable through any projected surface,
   because CNA-Go projects no base type to cast to. Nothing needs it decided
   yet, and deciding it with no consumer to test against would be guessing.

`GameComponentCollection` did not raise this: its four declared members
**override** rather than hide, and the one name collision was across kinds and
resolved cleanly.

## Why `Dictionary<K,V>` is still deferred

This is the one family the milestone did **not** advance, and the reason is
precise. The behavior is establishable — the IL is right there. The **public
surface mapping** is not.

Six public members of the pinned `Dictionary<K,V>` cannot be projected in
already-decided terms:

| member | what it needs |
| --- | --- |
| `GetObjectData(SerializationInfo, StreamingContext)` | the `System.Runtime.Serialization` subsystem |
| `OnDeserialization(object)` | the deserialization callback contract |
| `Keys` | the public nested `KeyCollection`, with its own surface and nested enumerator |
| `Values` | the public nested `ValueCollection`, likewise |
| `Comparer` | `IEqualityComparer<TKey>` as a public interface |
| `GetEnumerator` | the public nested `Dictionary<K,V>.Enumerator` over `KeyValuePair<K,V>` |

The first two are decisive on their own. Both are

```text
.method public hidebysig newslot virtual instance void GetObjectData(...)
.method public hidebysig newslot virtual instance void OnDeserialization(object)
```

— **public**, not explicit implementations — so `LaunchParameters`, a public
non-sealed class, genuinely exposes them and C# code can call them. There is no
faithful complete projection without a serialization architecture.

Projecting a subset instead would make `LaunchParameters` a **partial type**
with new `MISSING_MEMBER` diagnostics, which is not a BCL-mapping outcome and
is not this milestone's to take. `LaunchParameters` is `Dictionary<K,V>`'s only
consumer in the profile, so nothing else waits on it.

Recording this is the point: the blocker is now four named sub-decisions
attributable to exact CLR members, not "a dictionary is hard".

## Verifier

New measurement: a **BCL signature-adapter registry**, pinning each adapter's
exported Go surface to the exact public CLR member inventory, with a rationale
per member and a reason per exclusion. Without it an adapter type would be a
hole in the unexpected-member scan, because every exported member on an adapter
receiver is admitted.

The measurement runs only when the framework package was actually extracted, so
an isolated per-type fixture — which models one XNA type and no package — is
not asked for the whole adapter surface.

Nine new negative controls, `LANGUAGE_MAPPING_MISMATCH` throughout:

```text
adapter_type_absent            public_member_missing
enumerator_missing             extra_public_member
read_only_made_mutable_add     read_only_made_mutable_setter
read_only_made_mutable_clear   excluded_member_promoted
sync_root_promoted
```

Three of them are the read-only claim attacked directly: promoting `Add`,
`Clear`, or a `SetItem` would make a read-only view writable, which is the
whole point of the type.

## Scoreboard

```text                          before   after
TARGET_TYPES                       119     120
TARGET_MEMBERS                    1743    1746
COMPLETE_TYPES                     114     115
MISSING_TYPE                       138     137
TOTAL_DIAGNOSTICS                  315     314
MISSING_MEMBER                     177     177     protected partials untouched
EXPECTED_GO_MEMBERS               3255    3255     no inherited surface added
mutation inventory                 469     478
behavior corpus                    588     595
```

`EXPECTED_GO_MEMBERS` did not move, which is the right outcome: a signature
adapter changes how an existing member's type is **spelled**, it does not add
inherited surface the way a composed base does. Every mismatch, leak, allowlist
and unmeasured counter is zero.
