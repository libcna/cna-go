# Foundation 81 — EnvironmentMapEffect, SkinnedEffect, and the end of the stock effects

```text
COMPLETE_TYPES   177 -> 179        MISSING_TYPE       80 -> 78
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  271 -> 296        DELIBERATELY_UNBOUND_ROUTES  +11
INTERFACE_WITNESS_PROJECTIONS  32 -> 36
XNA_COMPOSED_DERIVED_TYPES_PROJECTED  20 -> 22   (all six of Effect's)
XNA_BASE_SUBSTITUTABILITY_LIVE  3 -> 4
```

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
Microsoft.Xna.Framework.dll            (FrameworkResources, three messages)
```

## LightingEnabled is EXPLICIT, and one refusal has nowhere to go

Both types implement exactly one IEffectLights member privately:

```text
.method private ... IEffectLights.get_LightingEnabled()
  IL_0000: ldc.i4.1
  IL_0001: ret                        -- two bytes: lighting is ALWAYS on

.method private ... IEffectLights.set_LightingEnabled(bool)
  if (!value) throw new NotSupportedException(string.Format(
      CultureInfo.CurrentCulture, FrameworkResources.CantDisableLighting,
      typeof(T).Name));                -- 51 bytes
```

Because both are explicit, the pinned contract lists **no LightingEnabled
property** on either type: in C# a consumer reaches it only through the
interface. Go has no explicit implementation, so both accessors exist on the
type and are registered as interface witnesses — the same machinery
GraphicsDeviceManager's IGraphicsDeviceManager and the four vertex structs'
IVertexType already use, and both halves of the witness rule hold (the member is
absent from the public set, and both types can declare it).

**The setter's refusal is dropped, and that is stated rather than hidden.**
`IEffectLights::set_LightingEnabled` is INFALLIBLE in its measured contract —
every other implementor's is one `stfld` plus a dirty-flag `or`, which is what
Foundation 18 measured and what the interface's signature says. Widening it for
these two would change a contract five implementors share to accommodate two. So
the projection answers the way the reference does for the legal value, `true`,
and drops the exception for `false`. The message is retained in the source as
`cantDisableLighting` and verified byte-for-byte by `tools/resource_strings`, so
the refusal is recorded as measured rather than forgotten.

This is the ONE place in the stock effects where a projected member cannot carry
a refusal the reference makes.

## Two setters store a flag as well as writing a parameter

EnvironmentMapEffect's `set_EnvironmentMapSpecular` and `set_FresnelFactor` are
59 bytes rather than 13, and the extra bytes are a shader permutation:

```text
set_EnvironmentMapSpecular:  param.SetValue(value);
                             specularEnabled = value != Vector3.Zero;
                             dirtyFlags |= ShaderIndex
set_FresnelFactor:           param.SetValue(value);
                             fresnelEnabled  = value != 0;
                             dirtyFlags |= ShaderIndex
```

The parameter write comes FIRST in both, so a failed write leaves the flag
alone — which the projection preserves and a planted defect that reorders them
is killed for.

## SkinnedEffect's three members that no other stock effect has

1. **`MaxBones`** is `public static literal int32 = int32(0x00000048)` — 72, the
   only public field in the family. It projects as a Go constant, which is the
   same thing: no storage, and usable as an array length.
2. **`set_WeightsPerVertex`** is the only managed setter in the family that
   VALIDATES. Anything but 1, 2 or 4 is an ArgumentOutOfRangeException carrying
   `FrameworkResources.SkinnedEffectWeightsPerVertex`, so it is fallible and its
   error channel carries the reference's own refusal. It also has **no early
   return**: assigning the value it already holds stores it again and raises
   ShaderIndex again, unlike every boolean permutation setter in the family.
3. **The bone-transform pair**, whose four guards use three exception types and
   are deliberately not symmetric:

```text
SetBoneTransforms   null OR EMPTY -> ArgumentNullException, NullNotAllowed
                    length > 72   -> ArgumentException, SkinnedEffectMaxBones
GetBoneTransforms   count <= 0    -> ArgumentOutOfRangeException, NO message
                    count > 72    -> ArgumentOutOfRangeException, WITH message
```

An empty array raising ArgumentNullException is the reference's choice and is
reproduced rather than tidied. And `GetBoneTransforms` ends with a loop the
projection would be wrong to skip:

```text
for (int i = 0; i < count; i++) result[i].M44 = 1;
```

The shader stores a bone as three rows and the fourth comes back undefined, so
the reference overwrites M44 on every returned matrix. The native run asserts
both halves: that a value written into M41 survives the round trip, and that
M44 comes back as exactly 1.

## TextureCube became the fourth substitutable base, ten milestones late

`EnvironmentMapEffect::EnvironmentMap`'s setter is the **only** TextureCube
PARAMETER position in the whole profile. TextureCube has been composed since
Foundation 71 and has had a projected derived type since Foundation 73, and its
substitutability requirement stayed LATENT the whole time because no position
named it. Projecting this effect made it LIVE, and the verifier reported that
rather than the author noticing:

```text
substitutability is LIVE and the base is not in substitutableBases, so its
parameter positions still take the concrete pointer and no derived value can
reach them
```

The widening also reached the six `TextureCube.SetData`/`GetData` package
functions, whose receiver-first parameter is a parameter position under the
settled generic-method rule.

`EffectParameter::GetValueTextureCube` is a RETURN and keeps the concrete
pointer: a caller uses every TextureCube member on what it hands back, and no
derived identity is at stake.

## A retention CNA takes, and XNA does not

Assigning a TextureCube to an EnvironmentMapEffect makes CNA refuse to destroy
it:

```text
cna_texturecube_destroy failed with CNA result 3:
The TextureCube is retained by an EffectParameter.
```

XNA's `Dispose` on the same texture is legal. The stress holds the divergence
from BOTH sides — it asserts the refusal happens, and asserts that
`SetEnvironmentMap(nil)` releases the cube so the same Dispose then succeeds.
The first arrangement disposed the cube in a convenient order and never met it;
the assertion exists because the run failed and the reason was worth keeping.

## 25 routes bound, 11 recorded as deliberately unbound

```text
bound      environment_map_effect_create + 5 setters + 3 get/set pairs
           skinned_effect_create + 6 setters + 2 get/set pairs
                                 + set_bone_transforms + copy_bone_transforms
unbound    every getter beside a bound setter, MANAGED_REFERENCE (8)
           the three texture getters, REPRESENTATION (3)
```

`cna_skinned_effect_get_vertex_color_enabled` and its setter add no registry
row: the pinned contract declares **no** VertexColorEnabled on SkinnedEffect, so
they back no projected member and are not routes a projected member could have
used.

## Planted defects

25 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   25      KILLED   24      SKIPPED   0
```

15 are killed by the in-package tests and 9 by the vertex-buffer stress child.

**The first pass killed only 20, and all five survivors were the tests' fault
rather than the code's.** They are recorded because each is a different way an
assertion can look strong and hold nothing:

| survivor | why it survived | what the assertion is now |
| --- | --- | --- |
| the two flag-storing setters, reordered | the path never executed: both refuse at the top when there is no native effect, so the reorder was never reached | an effect whose native half is RELEASED, where the parameter write actually fails and the ordering becomes observable |
| `LightingEnabled` setter stores its argument | the assertion read a CONSTANT getter, which answers true however much state the setter corrupts | the setter must store nothing at all -- no flag, no dirty bit -- and the `true` call comes LAST so a storing mutation ends holding it |
| `SetEnvironmentMap(nil)` releases nothing | the release was checked by disposing the SAME cube again, and CNA documents a repeated dispose as success | a SECOND cube nothing has tried to dispose, where a success can only be the release |

**One is not killed, and it turned out to be a measurement rather than a gap.**

`bone_transforms_skip_the_m44_correction` removes the loop that forces
`M44 = 1` on every matrix `GetBoneTransforms` returns. It survived the first
pass over identity input, which proved nothing -- identity's M44 is already 1.
The input now carries `M44 = 7`, and it survives ANYWAY: with the correction
removed, `M44` still comes back as exactly 1.

That is CNA normalising the fourth row itself, and it is worth stating plainly.
The correction is kept because the CONTRACT has it -- the reference writes it
unconditionally and a backend that stored all sixteen floats would need it --
and it is redundant on this artifact rather than load-bearing. The projection
does not depend on a measurement of CNA to be right, which is the whole point of
keeping the reference's body; what the measurement adds is that nothing on this
artifact can tell the two apart.

## Native qualification

The slice runs in the vertex-buffer scenario after Foundation 79's and 80's, for
the same reason those run late: disposing the applied effect un-applies it.

```text
                                HEADLESS   SOFTWARE
LIT_EFFECT_CREATIONS                  40         40
LIT_EFFECT_CREATION_REFUSALS           0          0
LIT_EFFECT_LIGHT_CHECKS               40         40
LIT_EFFECT_ROUND_TRIPS                20         20
LIT_EFFECT_BONE_ROUND_TRIPS           20         20
LIT_EFFECT_BONE_REFUSALS               0          0
LIT_EFFECT_APPLIES                    40         40
LIT_EFFECT_APPLY_REFUSALS              0          0
LIT_EFFECT_RETENTION_CHECKS           20         20
LIT_EFFECT_RELEASE_CHECKS             20         20
LIT_EFFECT_CLONE_CHECKS               20         20
LIT_EFFECT_DISPOSAL_CHECKS            20         20
```

Both artifacts report identical counters. `LIT_EFFECT_BONE_ROUND_TRIPS` is the
one worth reading twice: it means CNA accepted eight 4x4 matrices, handed them
back in the order they were sent -- a value written into M41 survives per index
-- and that the projection's M44 correction ran, because the assertion checks
for exactly 1 and CNA does not send one.

## What is proved, and where

```text
skinned_effect_test.go       7 tests: the constant LightingEnabled getter and
                             the setter that cannot change it, the two
                             flag-storing setters refusing before they store,
                             EnvironmentMapEffect's LIT matrix words, the
                             validating setter with its message and its absent
                             early return, all four bone guards and their three
                             exception types, MaxBones as a usable constant, the
                             three-interface split across all six effects, and
                             TextureCubeReference's two halves
native_stress vertex-buffer  20 cycles on two artifacts: creation, the three
                             published lights on both types, EnableDefaultLighting
                             over lights that exist, the four and two crossing
                             properties round-tripped, the TextureCube position
                             with its retention and its release, the bone array
                             round-tripped with M44 forced, OnApply through each
                             effect's own pass, Clone with its downcast and its
                             copied WeightsPerVertex, and disposal
external canary              1 test compiling both from outside, including the
                             witness pair, the widened TextureCube position, the
                             constant, all four guards, and all six of Effect's
                             derived types at a widened Effect position
tools/resource_strings       three more Microsoft messages verified byte-for-byte
```

```text
FOUNDATION_MILESTONE_81_COMPLETE=true
```
