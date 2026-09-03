# Foundation 79 — BasicEffect, DirectionalLight and IEffectLights

```text
COMPLETE_TYPES   171 -> 174        MISSING_TYPE       86 -> 83
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  230 -> 258        DELIBERATELY_UNBOUND_ROUTES  +21
XNA_COMPOSED_BASE_RELATIONSHIPS  5 -> 6
```

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
```

## The measurement that decided the whole design

CNA's stock BasicEffect publishes **no EffectParameters**. The probe loaded
`{"cnjVersion":1,"type":"BasicEffect"}` through `ContentManager.Load<Effect>` on
both qualified artifacts:

```text
HEADLESS   EFFECT_LOADED   PARAMETER_COUNT 0   TECHNIQUE_COUNT 1  ("Default", 1 pass)
SOFTWARE   EFFECT_LOADED   PARAMETER_COUNT 0   TECHNIQUE_COUNT 1  ("Default", 1 pass)
```

That matters because the reference's BasicEffect is **managed state plus a push
in `OnApply`**, and the push writes fifteen parameters it looked up BY NAME —
`"Texture"`, `"DiffuseColor"`, `"WorldViewProj"`, `"DirLight0Direction"` and the
rest, all read off `CacheEffectParameters`' 478 bytes of `ldstr`. With no
parameters in existence there is nothing to write to, so the projection pushes
into CNA's own stock-effect state instead: `cna_basic_effect_*` plus the three
shared interface families `cna_effect_matrices_*`, `cna_effect_fog_*` and
`cna_effect_lights_*`.

**The managed half is kept.** It would have been simpler to forward every
property straight to CNA, and it would have been wrong twice over: it would have
made fourteen infallible members fallible, contradicting the `IEffectMatrices`
and `IEffectFog` signatures Foundation 18 measured from the same assembly, and
it would have let CNA's clamping answer getters whose reference bodies are one
`ldfld`.

## The fallibility split is the contract's, member by member

Fourteen properties are `ldarg.0; ldfld <field>; ret` to read and a store plus a
dirty-flag `or` to write. **Four** are not, and each of those four really does
cross:

```text
SpecularColor   specularColorParam.GetValueVector3()  / SetValue
SpecularPower   specularPowerParam.GetValueSingle()   / SetValue
FogColor        fogColorParam.GetValueVector3()       / SetValue
Texture         textureParam.GetValueTexture2D()      / SetValue
```

Each ends in `calli unmanaged stdcall` into `ID3DXBaseEffect`. So those four
carry an error result and their siblings do not — which is exactly what
`IEffectFog` already said about `FogColor` alone among its four properties.

`EnableDefaultLighting` is fallible for the same reason one level out: it is two
statements, and the second reaches twelve `DirectionalLight` setters.

`DirectionalLight` is the mirror image. All four getters are seven bytes over a
cached field; all four setters write through an `EffectParameter`. It reads
infallibly and writes fallibly, and `set_Enabled` writes **two** parameters —
disabling a light is expressed to the shader as zero colours while the cache
keeps the colour the property reports.

## Texture answers a managed field, and why that is not the parameter case

Foundation 72 recorded that CNA reports an effect texture as a HANDLE and not as
an object, so `EffectParameter::GetValueTexture2D` refuses rather than handing
back a fresh facade that would make `p.GetValueTexture2D() == myTexture`
silently false.

`BasicEffect::Texture` meets the same obstacle and answers it differently,
because the position is different. An `EffectParameter` is a **view** anything
holding another view can write, so a cache in the view could go stale;
`BasicEffect::Texture` is a property of the effect, and CNA publishes no
parameters at all, so the object the setter was given IS the current value.
Holding it reproduces the reference's observable exactly where refusing would
not. The native stress run checks the identity with a real texture, not just the
null case.

## Effect is now COMPOSED, and it widens at RETURNS

Composing `Effect` was a precondition for any stock effect, and every blocker
the relationship carried was false rather than merely deferred: the TRANSITIVE
one died when Foundation 72 projected `Effect`, and the SUBSYSTEM one said the
derived effects "reach EffectParameter, which calls unmanaged D3DX" — which is
what the REFERENCE does, not what this projection does.

It is also the **second** base whose returns widen, after `System.Exception` in
Foundation 76, and the measurement is stronger than the exception hierarchy's:

```text
.method public hidebysig virtual instance class Effect Clone()
  newobj instance void BasicEffect::.ctor(class BasicEffect)
```

All five stock effects override `Clone` to return their own class. Returning the
composed `*Effect` would hand back the BASE HALF of a BasicEffect with **no path
to the object that owns it** — `effectBase` is unexported — so the downcast
would not merely be lost, it would be impossible. `EffectReference` carries
Effect's whole public surface plus that unexported method, so a consumer can use
the value without asserting and can still assert back to `*BasicEffect`.

Making Effect substitutable also widened its three PARAMETER positions, which
the verifier required rather than suggested: `SpriteBatch::Begin` ×2 and
`Effect::.ctor(Effect cloneSource)`.

**Disposal is deliberately not on the interface.** Effect DECLARES the protected
`Dispose(Boolean)` override and projects both `DisposeByNone` and
`DisposeByBoolean`; BasicEffect declares no Dispose at all, so its inherited
PUBLIC surface carries one `Dispose()` that takes no argument. The two types
spell disposal differently because the pinned metadata does, and an interface
member would have to impose one spelling on the other type.

## One virtual is dispatched and one is not

`Effect` declares both `Clone` and `OnApply` virtual. Only `OnApply` needs a
composition hook:

- **`OnApply`** is reached through the base object. `EffectPass` holds an
  `*Effect` in a field and calls `OnApply` on it; Go dispatches to the base's
  method and nothing in the language recovers the derived body. `effectVirtuals`
  and `bindDerivedEffect` exist for that one member.
- **`Clone`** needs nothing. It is a member of `EffectReference`, so a consumer
  holding a BasicEffect calls `BasicEffect::Clone` through Go's own method set,
  and the composed base is unreachable from outside the package.

The first draft dispatched both. The planted-defect run is what found it: the
mutation that removed `Clone`'s dispatch could not be killed, because no
production path executed it. A branch nothing executes is the shape this project
refuses everywhere else, so it was removed rather than covered.

## Effect is the first middle link with identity sites

`Effect` forwards its binding to `GraphicsResource` and holds no copy of the CLR
`this` — one chain, one holder. But two of its members report a CLR **type**,
and on a BasicEffect that type is BasicEffect:

```text
set_CurrentTechnique   ldarg.0 ... call Helpers::CheckDisposed(object, native int)
.ctor(Effect source)   ldarg.1 ... call Helpers::CheckDisposed(object, native int)
```

The second is `ldarg.1`, the clone SOURCE, and the projection's `cloneBase`
takes that source as its receiver — so the same resolution applies through it.
`xnaCompositionIdentities` previously required a middle link to have no sites at
all; that rule was two claims wearing one word, and it is now two: a middle link
holds no derived FIELD and declares no self accessor, and it may record sites
that reach the chain's accessor through the link it forwards to.

## 28 routes bound, 21 recorded as deliberately unbound

The cluster declares 49. The projection reaches 28, and the other 21 are in the
native-ABI registry with their measured reasons. They fall into one pattern with
two exceptions:

- **Thirteen GETTERS, class `MANAGED_REFERENCE`.** The reference reads its own
  field. Reading CNA back would let its clamping — or its derived quantity —
  answer an XNA getter. `get_DiffuseColor` is the sharpest case: what OnApply
  writes is the colour multiplied by alpha, so the reference's own getter and
  its own push already disagree.
- **Four DirectionalLight getters plus `cna_directional_light_create`**, same
  class. A free-standing native light is exactly the case in which the
  reference's constructor reaches nothing, because every setter it calls is
  guarded by `brfalse` on a parameter that a caller cannot obtain.
- **`cna_basic_effect_get_texture`, class `REPRESENTATION`** — the handle/object
  obstacle above.
- **`cna_effect_lights_enable_default`, class `CONTRACT_DIVERGENCE`.** CNA's
  preset is CNA's; `EffectHelpers::EnableDefaultLighting` is a measured
  constant, thirteen calls with `ldc.r4` operands. Those twelve vectors are the
  contract, and calling CNA's preset would make a native default answer for an
  XNA behaviour.

## Planted defects

34 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   34      KILLED   30      SKIPPED   0
```

22 are killed by the in-package tests, which need no device:

```text
world_setter_drops_the_worldviewproj_and_fog_flags   projection_setter_raises_the_world_setters_flags
fog_enabled_setter_loses_its_early_return            diffuse_color_setter_gains_an_early_return
fog_end_initialiser_is_zero                          diffuse_color_initialiser_is_zero
dirty_flags_start_clear                              one_light_is_recomputed_while_lighting_is_off
one_light_raises_the_flag_without_a_transition       one_light_uses_or_instead_of_and
default_lighting_ambient_constant_is_wrong           default_lighting_light0_diffuse_is_wrong
enable_default_lighting_forgets_lighting_enabled     texture_getter_refuses_instead_of_answering_the_field
light_direction_default_is_up                        light_clone_arm_drops_the_specular_colour
light_getter_reports_zero_while_disabled             light_diffuse_setter_skips_the_cache_while_disabled
light_constructor_enables_the_light                  effect_set_current_technique_names_the_base_half
effect_clone_base_names_the_base_half                effect_on_apply_ignores_the_derived_override
```

8 need a live device and are killed by the vertex-buffer stress child:

```text
clone_loses_the_derived_body                clone_shares_its_source_lights
published_lights_are_rebuilt_per_read       constructor_leaves_light0_disabled
specular_color_is_cached_managed_side       set_texture_forgets_the_managed_field
constructor_writes_the_wrong_specular_power constructor_writes_the_wrong_specular_colour
```

**Four are not killed, and each is named rather than counted as a pass:**

| defect | why |
| --- | --- |
| `default_lighting_enables_before_it_writes` | EQUIVALENT. Enabling first makes the colour setters write CNA directly; enabling last makes `set_Enabled` write both caches. Both orders converge on the same native state, in either runtime. |
| `constructor_skips_the_specular_tail` | EQUIVALENT **given a measured coincidence**: CNA's fresh basic effect already reports `SpecularColor = Vector3.One` and `SpecularPower = 16`. The read is live, which the two value mutations above prove — writing 17 or `Vector3.Zero` is killed. |
| `disposed_effect_still_answers` | EQUIVALENT. The mutation reaches the same zeroed handle, so the generation guard still refuses. That guard is `interop.Resource`'s and is covered where it lives. |
| `light_view_is_never_destroyed` | **NOT FALSIFIABLE with the available tooling.** CNA exposes no leak detector, and a view handle that is never destroyed changes nothing observable. Recorded as a claim the suite does not hold. |

## Native qualification

The BasicEffect slice runs at the END of the vertex-buffer scenario. That
placement was forced by a measurement: disposing the applied effect un-applies
it in CNA, and the first arrangement put the slice ahead of the draws and turned
every one of them back into "no effect has been applied".

```text
                                        HEADLESS   SOFTWARE
BASIC_EFFECT_CREATIONS                        20         20
BASIC_EFFECT_CREATION_REFUSALS                 0          0
BASIC_EFFECT_ROUND_TRIPS                      20         20
BASIC_EFFECT_LIGHT_CHECKS                     60         60
BASIC_EFFECT_APPLIES                          20         20
BASIC_EFFECT_CONTROL_DRAWS                    20         20
BASIC_EFFECT_DRAWS                            20         20
BASIC_EFFECT_CLONE_CHECKS                     20         20
BASIC_EFFECT_DISPOSAL_CHECKS                  20         20
```

**`BASIC_EFFECT_DRAWS` is not evidence that applying a BasicEffect unblocked the
draw.** The control immediately before it succeeds on both artifacts, so the
draw already worked at that point in the scenario. The control was added for
exactly this reason: without it the 20 successes would have read as a result
they are not. What the pair does say is that a draw still succeeds with a
BasicEffect applied, and that CNA accepted the pass.

The DrawUser evidence stays **`VERIFIED_NATIVE_DRAW`**. Upgrading it to
`VERIFIED_PIXEL` needs a predictable output colour, and that needs the back
buffer read back on SOFTWARE with a known material — which this milestone makes
possible and does not yet do.

## What is proved, and where

```text
basic_effect_test.go        11 tests: the nine field initialisers and the five
                            defaults that are a zero value, every setter's
                            measured dirty-flag word, the five boolean setters'
                            early return and the ten setters that have none, the
                            stored-value getters, EnableDefaultLighting's two
                            statements, oneLight's bracket and its transition,
                            the four device-backed refusals, the three declared
                            interfaces, EffectReference's downcast, and the
                            identity sites' runtime type name
directional_light_test.go    6 tests: the constructor's two arms, the cache the
                            getters answer, the disabled-light divergence,
                            set_Enabled's early return, the nil receiver, and
                            every constant of the default-lighting rig
native_stress vertex-buffer 20 cycles on two artifacts: creation, the
                            constructor's specular tail read back out of CNA,
                            the three lights' identity and write-through,
                            EnableDefaultLighting over lights that exist, the
                            four crossing properties round-tripped, texture
                            object identity, OnApply through the effect's own
                            pass, a control draw, Clone with its downcast and
                            its own lights, and disposal
external canary              1 test compiling the family from outside, including
                            both constructors, the fallibility split at exact
                            shapes, the three interfaces, and the widened
                            positions
build-probe/f79-basic-effect the parameter-count measurement on two artifacts
```

```text
FOUNDATION_MILESTONE_79_COMPLETE=true
```
