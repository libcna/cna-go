# Foundation 93 — the content serializer attributes

Five types: `ContentSerializerAttribute`,
`ContentSerializerCollectionItemNameAttribute`,
`ContentSerializerIgnoreAttribute`,
`ContentSerializerRuntimeTypeAttribute`,
`ContentSerializerTypeVersionAttribute`.

The family reaches no runtime at all. Every declared member is a field access, a
field copy, or a `String.IsNullOrEmpty` check, and the five inherited members
answer from the derived object's own fields.

## The base carried three blockers and none survived

`System.Attribute` has been a deferred BCL base since Foundation 29. The rule
Foundation 92 settled — a recorded blocker is a claim, and claims get
re-measured — is what this milestone applied, and all three came apart
differently.

### 1. "Go has no attribute metadata" — true, and not a reason to withhold

It is true that Go cannot ATTACH an attribute to a declaration. It does not
follow that the types cannot exist.

The pinned contract declares five classes with constructors and properties. It
declares no attaching operation, because attaching is a C# language act rather
than a member of the type. Measured further: the only places in the runtime
assembly that READ these attributes are `ReflectiveReader<T>` and
`ReflectiveReaderMemberHelper`, and both are `.class private` — neither is
public surface a consumer can reach. Foundation 92 had already measured that CNA
performs the type-reader dispatch itself.

So nothing a consumer reaches through this projection would have read an applied
attribute even if Go could apply one. The limitation is real, it is narrower
than the blocker made it sound, and it is now recorded on the adapter and on
each type instead of being used to withhold all five.

### 2. `GetCustomAttribute` and the inherited statics — answers itself

The blocker worried that the inherited static lookups take `Assembly`,
`MemberInfo`, `Module` and `ParameterInfo`, none of which the profile maps, and
that "whether inherited STATICS are part of a derived type's projected surface
is itself undecided".

Measured from the pinned mscorlib: `GetCustomAttribute`, `GetCustomAttributes`
and `IsDefined` are the only three, and all three are `static`. A static is not
surface a consumer reaches through a derived type in this model, so they are
excluded by name with their measured parameter shapes — the same kind of honest
exclusion `BinaryReader::ReadDecimal` got in Foundation 92.

### 3. `TypeId` returning `System.Object` — already settled

An object-typed member projects to `any`; `Model.Tag` does. And `TypeId`'s
default really does return `GetType()`, so `any` holding a `reflect.Type` is
what the CLR itself hands back, not a leak.

## What `System.Attribute`'s adapter holds: nothing

Measured from the pinned mscorlib, `System.Attribute` declares **no instance
field**. So `attributeBase` is an empty struct and each of the five inherited
members takes the OWNER, answering from the derived object's own data.

Its public instance surface is exactly five members — `Equals`, `GetHashCode`,
`Match`, `IsDefaultAttribute` and `TypeId`. The four `private` `_Attribute`
interface implementations and the `family` constructor are excluded with their
measured accessibility.

## The details a plausible implementation gets wrong

- **`AllowNull` defaults to TRUE.** The constructor's body is two instructions
  before the base call: `ldarg.0; ldc.i4.1; stfld allowNull`. Every other field
  takes its zero value.
- **`CollectionItemName` is not a field read.** An empty field answers the
  literal `"Item"`.
- **`HasCollectionItemName` reads the FIELD**, not the property. Reading the
  property would answer true always.
- **The setter refuses an empty value and names `"value"`** — the compiler's
  name for a setter's argument — while the two string CONSTRUCTORS on other
  types name their own parameters.
- **`set_ElementName` has no guard at all**, which is what makes the previous
  point a measured guard rather than a habit about strings.
- **`Clone` copies the FIELD**, so a clone of an attribute with no name set
  keeps the empty field and `HasCollectionItemName` stays false on both.
- **`IsDefaultAttribute` is always false**, including on a default-constructed
  `ContentSerializerAttribute`.
- **`Equals` compares fields, not references**, so two separately constructed
  attributes with equal fields are equal — the opposite of Go's `==` on two
  pointers.
- **`ContentSerializerTypeVersionAttribute`'s Int32 constructor has no guard**
  and stores a negative version, unlike the two string constructors.

## Falsifiability

`mutate93.py`, 36 planted defects, **36 killed, 0 survived**.

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 36 | 36 | 0 |
| equivalent | 1 | 0 | 1 |
| native | 0 | — | — |

There is no native table and there cannot be one: the family reaches no runtime.
That is not the Foundation 92 situation, where a native table was missing
because an asset was — here there is nothing native to reach.

### The mutation run corrected the code, again

Three survivors in the first pass all pointed at the same thing: the projected
`Equals` carried guards **no input could reach**.

`reflect.DeepEqual` answers `false` for a nil against a non-nil, `false` for two
different concrete types, and otherwise walks fields. The reference performs
those as three separate steps and the first attempt here reproduced all three —
so neutering the null check or the type check changed no answer, because
DeepEqual had already decided.

They were removed rather than kept. A guard no input can reach is not
documentation; it is a line that looks tested and is not. The same judgement
`Collection<T>`'s statically dead `items.IsReadOnly` guard got.

Three other survivors were real test gaps and were closed:

- the refused setter was only checked on an attribute whose field was ALREADY
  empty, so a store-then-refuse implementation looked identical;
- `set_ElementName` was only checked with the empty string on a fresh attribute,
  so a guard that refused clearing was invisible;
- `Clone`'s bools were all set to the same value, so a copy reading the wrong
  source field agreed by accident. The test now covers both arrangements.

### One equivalent mutant, asserted to survive

`standalone_collection_item_name_grows_a_fallback` adds an `"Item"` fallback to
`ContentSerializerCollectionItemNameAttribute`'s getter. It survives because
that type's CONSTRUCTOR refuses an empty name, so the field can never be empty
and the fallback is unreachable. The difference from
`ContentSerializerAttribute`'s getter is real in the IL and unobservable through
the projection's surface, because no instance with an empty field can be built.

The harness runs it in a separate table and asserts it SURVIVES.

## Native qualification, and a flake named rather than retried away

The family reaches no runtime, so nothing here has a native slice. Both
artifacts were re-qualified anyway, because the projection package changed:

| artifact | exit | CONTENT_CYCLES | STORAGE_ROOT_CHECKS | GAME_FRAME_STEP_TICKS |
| --- | ---: | ---: | ---: | ---: |
| HEADLESS | 0 | 20 | 20 | 80 |
| SOFTWARE | 0 | 20 | 20 | 80 |

The first HEADLESS run **failed**, at
`a first Tick delivered 2 updates, want one`. That assertion is in the
frame-step slice and this milestone touches nothing native, so the failure was
investigated rather than retried away: the machine was at **load average 55**
from unrelated builds, and CNA's fixed timestep catches up an extra frame when
the first Tick straddles two 16.7 ms intervals. The re-run at load 13 passed.

Recorded because it is a real property of the assertion: `GAME_FRAME_STEP_TICKS`
counts a timing outcome, and a loaded machine can change it without anything in
the binding changing. A future run that fails here should check the load before
looking for a regression.

## What this milestone does not claim

A consumer can construct any of these five, set them, read them back, compare
them and clone them. A consumer **cannot** annotate a declaration with one,
because Go has no attribute metadata and no syntax for it. That is stated on
`attributeBase` and on each type, and it is the honest shape of the family:
the types are complete and the language operation they exist to support is not
available.
