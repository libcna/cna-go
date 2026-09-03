# Foundation 80 — AlphaTestEffect, DualTextureEffect and EffectMaterial

```text
COMPLETE_TYPES   174 -> 177        MISSING_TYPE       83 -> 80
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BOUND_FUNCTIONS  258 -> 271        DELIBERATELY_UNBOUND_ROUTES  +10
XNA_COMPOSED_DERIVED_TYPES_PROJECTED  17 -> 20
```

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll   560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
```

## The shape was settled; what this milestone measured is where it does NOT hold

Foundation 79 established the stock-effect shape against BasicEffect: managed
state with the reference's dirty flags, a push in `OnApply`, and only the
properties the reference backs with an `EffectParameter` reaching CNA directly.
All three types here follow it.

The interesting result is the one that stopped a refactor. Sharing an accessor
body across the five stock effects looked obvious — `set_World` is 23 bytes in
every one of them — and it is **wrong**, measured from the `or` operand of each
setter's own IL:

| setter | BasicEffect | AlphaTestEffect | DualTextureEffect |
| --- | --- | --- | --- |
| `set_World` | `19` = WorldViewProj\|World\|Fog | `17` = WorldViewProj\|Fog | `17` |
| `set_View` | `21` = WorldViewProj\|EyePosition\|Fog | `17` = WorldViewProj\|Fog | `17` |
| `set_Projection` | `1` | `1` | `1` |
| `set_DiffuseColor` | `8` | `8` | `8` |
| `set_FogEnabled` | `0xa0` | `0xa0` | `0xa0` |
| `set_VertexColorEnabled` | `0x80` | `0x80` | `0x80` |
| `set_AlphaFunction` | — | `64` | — |

The two unlit effects raise TWO matrix flags where BasicEffect raises three,
because they have no world parameter and no eye position. Every other word
agrees. So each type declares its own accessors with its own measured word, and
only the WRITE that has no flags in it — the three-matrix push — is shared, as
`pushEffectMatrices`.

`TestUnlitEffectsRaiseTwoMatrixFlagsWhereBasicEffectRaisesThree` holds both
halves of that table, including the assertion that BasicEffect's two setters do
**not** raise the unlit pair, so a future collapse into one body fails there.

## What crosses, per type

```text
AlphaTestEffect     FogColor, Texture                      2 of 16 members
DualTextureEffect   FogColor, Texture, Texture2            3 of 13
EffectMaterial      nothing -- it declares one constructor  0 of 1
```

Each crossing accessor is 12 bytes to read and 13 to write in the reference,
against 7 and 22-or-23 for every managed one.

## Two details that decide behaviour

1. **Neither `AlphaFunction` nor `ReferenceAlpha` validates.** Both setters are
   a store and an `or`, 23 bytes with no guard, so an undeclared
   `CompareFunction` and a reference alpha outside 0..255 are stored and
   reported back unchanged. That is also why their CNA getters stay unbound: a
   route that clamped would answer something the reference never answers.
2. **One CNA route backs two properties.**
   `cna_dual_texture_effect_set_texture` takes a layer index, "zero or one",
   where the contract declares `Texture` and `Texture2`. The projection sends
   index 0 and index 1, and the native run sets one layer and clears the other
   to prove the index is honoured rather than ignored.

## EffectMaterial is eight bytes, and that is the whole type

```text
.ctor(Effect cloneSource)
  ldarg.0; ldarg.1; call instance void Effect::.ctor(class Effect); ret
```

It declares one member and overrides nothing — not `Clone`, not `OnApply`. So
the projection is Effect's inherited public surface plus a class name, and three
consequences follow that are easy to get wrong by not looking:

- **`Clone` answers an `Effect`, not an `EffectMaterial`**, because the
  reference declares no override and `Effect::Clone` builds an Effect. The
  native run asserts that positively.
- **There is no `OnApply`.** Effect declares it `protected internal`, and the
  settled rule projects a protected member on the type that DECLARES it; an
  inherited surface is the PUBLIC surface. `EffectReference` therefore dropped
  `OnApply` this milestone — an interface member would have required one that
  the contract does not give this type.
- **It installs `bindDerived` and NOT `bindDerivedEffect`.** The first carries
  the CLR `this` for `ToString`; the second installs an override, and this type
  has none.

### A hazard CNA documents and this type does not answer

`cna_effect_material_retain_parameter_texture_ext` exists because "a parameter
holds a raw pointer nothing owns". It backs no member of the pinned contract, so
nothing calls it, and a consumer who writes a texture into one of this
material's parameters and then disposes that texture leaves CNA holding a
dangling pointer. That is inherited from Foundation 72's
`cna_effect_parameter_set_value_texture` binding rather than introduced here,
and it is recorded rather than papered over.

## 13 routes bound, 10 recorded as deliberately unbound

```text
bound      alpha_test_effect_create + 6 setters
           dual_texture_effect_create + 4 setters (one indexed, two properties)
           effect_material_create
unbound    every getter beside a bound setter, MANAGED_REFERENCE (8)
           the two texture getters, REPRESENTATION (2)
```

`EffectMaterial` adds no registry row: its two `_ext` routes back no member of
the pinned contract, and this registry is about routes a projected member could
have used.

## Planted defects

22 distinct defects, each a real way to get this wrong and each compiling.
**Every one executed its path; none was skipped.**

```text
PLANTED   22      KILLED   20      SKIPPED   0
```

13 are killed by the in-package tests, which need no device: both types' field
initialisers, the two matrix flag words, the AlphaTest flag, both
non-validating setters, the early returns, and EffectMaterial's class name and
null guard. 7 need a live device and are killed by the vertex-buffer stress
child: the two Clone bodies, the AlphaTestEffect clone's copied pair, the
managed texture field on both types, the second layer's own field, and a
managed-side cache standing in for a CNA read.

**Two are not killed, and they are the same defect twice:**

| defect | why |
| --- | --- |
| `dual_texture_ignores_the_layer_index` — send index 0 for both layers | NOT FALSIFIABLE with the bound surface |
| `alpha_test_on_apply_skips_the_alpha_test_push` — elide the AlphaTest branch | NOT FALSIFIABLE with the bound surface |

Both mutate a PUSH whose only observable is what the shader draws. Their
read-back routes -- `cna_dual_texture_effect_get_texture` and
`cna_alpha_test_effect_get_alpha_function` -- are deliberately unbound, for the
object-identity and non-validating-store reasons recorded above, so nothing in
the projection can read the pushed value back.

This is the boundary of what the current evidence can hold, and it is the same
boundary that makes `VERIFIED_PIXEL` the next thing worth building: a
back-buffer readback on SOFTWARE with a known material would turn both of these
into killable defects, and neither is killable without one.

The one earlier survivor is fixed rather than excused.
`effect_material_accepts_a_null_source` survived a first run because the test
asserted only that the constructor errored; it now asserts the base
constructor's own `ArgumentNullException("cloneSource", NullNotAllowed)`, and
the mutation is killed.

## Native qualification

The slice runs in the vertex-buffer scenario, after Foundation 79's, for the
same reason that one runs last: disposing the applied effect un-applies it.

```text
                                HEADLESS   SOFTWARE
UNLIT_EFFECT_CREATIONS                40         40
UNLIT_EFFECT_CREATION_REFUSALS         0          0
UNLIT_EFFECT_ROUND_TRIPS              20         20
UNLIT_EFFECT_APPLIES                  40         40
UNLIT_EFFECT_APPLY_REFUSALS            0          0
UNLIT_EFFECT_CLONE_CHECKS             20         20
UNLIT_EFFECT_DISPOSAL_CHECKS          20         20
EFFECT_MATERIAL_CREATIONS             20         20
EFFECT_MATERIAL_IDENTITY_CHECKS       20         20
```

**One flake, named rather than hidden.** The first SOFTWARE run failed in the
`frame-step` scenario with "a first Tick delivered 2 updates, want one" — a
timing-sensitive fixed-step catch-up assertion this milestone does not touch.
The scenario passed twice in isolation immediately afterwards and the full run
passed on the retry. It is recorded here because a green report that took two
attempts is not the same result as one that took one.

## What is proved, and where

```text
alpha_test_effect_test.go    7 tests: both types' field initialisers, the two
                             matrix flag words that differ from BasicEffect's
                             AND the assertion that BasicEffect's differ from
                             theirs, the AlphaTest flag no other effect raises,
                             both non-validating setters, the boolean setters'
                             early return, the two-layer split, the
                             native-backed refusals, the two interfaces each
                             satisfies and the third neither does, and
                             EffectMaterial's name, identity site and null guard
native_stress vertex-buffer  20 cycles on two artifacts: creation, FogColor
                             round-tripped through CNA on both types, texture
                             object identity, the layer index honoured, OnApply
                             through each effect's own pass, Clone with its
                             downcast and its eleven or nine copied values,
                             EffectMaterial built from a source effect with its
                             ToString and its inherited Clone, and disposal
external canary              1 test compiling all three from outside at their
                             exact shapes, including the fallibility split, the
                             interfaces each does and does not implement, and
                             EffectMaterial's absent OnApply
```

```text
FOUNDATION_MILESTONE_80_COMPLETE=true
```
