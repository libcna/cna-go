# Foundation 94 — the Design converters

Thirteen types: `MathTypeConverter` and the twelve leaf converters over XNA's
math types. The whole namespace is string parsing, string formatting and field
description; no member touches CNA, a graphics device or a game.

**With this milestone, NOTHING in the project is deferred.** Every base
relationship — twelve XNA, twelve BCL — is COMPOSED, MAPPED or IMPLIED.

## The blockers, measured

`System.ComponentModel.ExpandableObjectConverter` carried three, and each named
a SUBSYSTEM. Measured, each subsystem was a handful of members.

| blocker | recorded as | measured |
| --- | --- | --- |
| `ITypeDescriptorContext` | "the single largest blocker in the profile at thirteen types" | 38 parameter positions, **zero members called** — every one is a pass-through |
| `CultureInfo` | "culture-aware conversion, which twelve types depend on" | **two** members: `get_CurrentCulture`, and via `get_TextInfo`, `get_ListSeparator` |
| `IDictionary` | "shared with System.Exception::Data" | **one** member, `get_Item`, and every key is a property name |

So the mappings follow the measurement:

- `ITypeDescriptorContext` → `any`. An opaque token, which is what
  `System.Object` already maps to.
- `IDictionary` → `map[string]any`. A name-to-value lookup a consumer writes
  literally, which matters because `CreateInstance`'s whole job is to take one.
- `CultureInfo` → a projected type sized to those two members. In this profile a
  culture IS the thing that separates list items.

## Two corrections the measurement forced

### `InstanceDescriptor` is required, and reading signatures alone said it was not

No member takes one or returns one. It is still needed twice over:

    MathTypeConverter::CanConvertTo(context, destinationType)
        destinationType == typeof(InstanceDescriptor)  ->  true
    TypeConverter::CanConvertFrom(context, sourceType)
        sourceType == typeof(InstanceDescriptor)       ->  true

Both answer `true` for it, so a consumer must be able to NAME it to ask. And
every leaf's `ConvertTo` **constructs** one. The projection carries the type
being built plus the arguments; the reference carries a `ConstructorInfo`, which
a Go struct has no counterpart for, and that difference in SHAPE is recorded on
the type.

### `PropertyDescriptor` is not a `reflect.StructField`

Reading only the vector converters says it could be — `Vector2Converter` wraps
`Type::GetField("X")`. But `ColorConverter` wraps `Type::GetProperty("R")`,
because XNA's `Color` exposes its channels as properties over a packed value and
this projection follows it. A `StructField` covers the first kind and cannot
cover the second, so what is projected is what both kinds share: a name, and a
way to read that component off a value.

## The details a plausible implementation gets wrong

- **`supportStringConvert` defaults to TRUE**, set before the base call, and
  exactly six of the twelve leaves clear it.
- **`CanConvertFrom` consults the flag and `CanConvertTo` does not.** Clearing
  it stops string PARSING and leaves string FORMATTING alone.
- **The split takes the BARE list separator; the join takes the separator plus
  ONE space.** The two directions genuinely disagree.
- **The refusal names the EXPECTED PARAMETERS**, joined by the bare separator —
  not the string that failed.
- **The element type is per converter**: `float32` for the four vectors,
  `int32` for `Point`, `uint8` for `Color`. Measured from which
  `ConvertToValues<T>` each leaf calls, not inferred from the value type.
- **Six leaves call the helpers not at all.** Three declare no `ConvertFrom`
  (`Rectangle`, `Plane`, `Matrix`) and three declare one whose entire body
  forwards to the base (`Ray`, `BoundingBox`, `BoundingSphere`). The projection
  keeps the distinction because the CONTRACT does.
- **`MatrixConverter` describes SEVENTEEN**: `Translation` — a computed property
  — FIRST, then `M11` through `M44` as fields. Its `CreateInstance` reads the
  sixteen cells and not `Translation`.
- **`ConvertTo` refuses a nil `destinationType` FIRST**, before the value is
  looked at.

## Two protected fields, a shape the project had not projected

`MathTypeConverter` declares `propertyDescriptions` and `supportStringConvert`
as `family` fields the contract carries. Every earlier projected field in the
profile has been public.

The verifier gained the arm it was missing: a field whose projected Go name is
UNEXPORTED cannot be in the member scan, which collects exported surface, so it
is looked for in the type's field list instead. The condition is the Go name and
not the contract's recorded accessibility, because the contract carries no
`access` for fields at all.

## Falsifiability

`mutate94.py`, 49 planted defects, **49 killed, 0 survived**.

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 49 | 49 | 0 |
| equivalent | 2 | 0 | 2 |
| native | 0 | — | — |

There is no native table and none is possible: the namespace reaches no runtime.

### The mutation run found fourteen real test gaps

The first pass killed 34 of 48 and the fourteen survivors were all gaps rather
than equivalents. Among them: the count check was never given a string with too
MANY components, so `!=` and `<` agreed; the failure message was checked for the
parameter NAMES but not for how they join; `Color`'s channels were never
formatted, so an out-of-order read agreed; `Matrix`'s sixteen cell descriptors
were never read, so a cell mapped to the wrong field agreed with the one
`Translation` assertion that did run; and `InstanceDescriptor`'s copy on the way
IN was never exercised, only the copy on the way out.

Every one is now covered, and the second pass killed all 49.

### Two equivalent mutants, asserted to survive

`convert_to_values_does_not_trim` removes the outer `Trim`. Element trimming
subsumes it: every part of a split string keeps its own spaces and the element
parse removes them. It is belt-and-braces in the reference too, and reproduced
because the reference has it.

`list_separator_ignores_the_receiver` reads the package-level current culture
instead of the receiver. `CultureInfo` has no exported constructor and the
contract declares none, so the only instance a consumer can reach IS the current
culture and the two can never differ. The field is still read from the receiver,
because a projection that read a package variable would be right by accident and
wrong the moment a second culture existed.

## The negative fixtures had to stop borrowing

Four verifier fixtures needed a DEFERRED base to mutate, and took whichever
entry happened to be deferred: `GraphicsResource` until Foundation 56, `Effect`
until 79, `MathTypeConverter` until this milestone. There is now none left.

They STAGE one instead, over a real base with a real derived type so that only
the STATUS is synthetic. That is not a weakening — it is what the rule always
deserved. A negative fixture that depends on the project having unfinished work
stops testing the moment the work finishes, which is exactly what happened three
times running.
