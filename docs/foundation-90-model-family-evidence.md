# Foundation 90 — the Model family, and the base that had been deferred since 29

Twelve types close `Microsoft.Xna.Framework.Graphics`: `Model`, `ModelBone`,
`ModelMesh`, `ModelMeshPart`, four collections and their four nested
enumerators. It is the largest single family in the profile and the last one in
the Graphics namespace.

## Authority

| what | file | sha256 |
| --- | --- | --- |
| the whole family | `Microsoft.Xna.Framework.Graphics.dll` | `560080fc39021c61…` |
| `ReadOnlyCollection<T>` | `mscorlib.dll` 4.0.30319.1 | `5634668d4775b011…` |
| `ModelHasNoEffect`, `ModelHasNoIEffectMatrices` | `Microsoft.Xna.Framework.dll` | `38e7093f52d7474b…` |

## The blocker this milestone had to settle

`System.Collections.ObjectModel.ReadOnlyCollection<T>` had been `DEFERRED` **as
a base** since Foundation 29, with two recorded blockers. Both are now gone, and
one of them was settled rather than merely unblocked.

The `SUBSYSTEM` blocker dissolved with the element types — `ModelBone`,
`ModelMesh` and `ModelMeshPart` are projected by this milestone, and `Effect`
has been projected since Foundation 72.

The `ARCHITECTURE` blocker asked for **a rule for a derived member that HIDES an
inherited one**. All four collections declare

```
.method public hidebysig instance ModelBoneCollection/Enumerator GetEnumerator()
```

with neither `virtual` nor `newslot` — C# `new`, which hides
`ReadOnlyCollection<T>.GetEnumerator()` rather than overriding it. The settled
collision rule would have produced two hashed names, neither of them
`GetEnumerator`.

The answer Foundation 29 called *available but untested* is the one adopted
here: **reaching a hidden base member in C# requires a cast to the base, CNA-Go
projects no base type to cast to, so the inherited member is unreachable and is
not projected at all.** Each collection has exactly one Go method named
`GetEnumerator` — the derived one — and the collision rule never fires because
there is only one member to name. The inherited one is recorded in the adapter's
`Excluded` list with that reason.

Five of the six inherited members are forwarded; the sixth is excluded. The base
is now `COMPOSED`, and `BCL_DEFERRED_BASE_BLOCKERS` drops accordingly.

## Three wrap an array, one wraps a list, and it is observable twice

```
ModelBoneCollection      ModelBone[]
ModelMeshCollection      ModelMesh[]
ModelMeshPartCollection  ModelMeshPart[]
ModelEffectCollection    List<Effect>        <- the odd one
```

`ModelEffectCollection` is the only collection in the family that **mutates**:
it declares `assembly` `Add` and `Remove`, which is what
`ModelMeshPart.set_Effect` calls. Two consequences follow from the CLR storing
the `IList` *reference*, and both are reproduced:

1. **Its view is LIVE.** A consumer holding `mesh.Effects` sees every addition
   immediately; an array-backed view cannot change length at all. This is what
   `NewReadOnlyCollectionOverLiveReferences` exists for — a fourth constructor
   over the same adapter, taking a closure over the owner rather than a captured
   slice header, because a captured header freezes the length and a `*[]T` would
   hand out the storage.
2. **Its enumerator is VERSION-CHECKED.** It wraps `List<Effect>.Enumerator`,
   which fails fast once the list is mutated. The three array-backed enumerators
   have no version field because the reference's array enumerator has none.

So enumerating a mesh's `Effects` while assigning a part's `Effect` fails in the
reference, and doing the same to `Bones`, `Meshes` or `MeshParts` cannot. The
signatures carry the difference: `ModelEffectCollectionEnumerator.MoveNext`
returns `(bool, error)` and its three siblings return `bool`.

## ModelMeshPart.set_Effect is the family's only real behaviour

175 bytes, and it is a **reference count** over the parent mesh's `Effects`:

```
if (value == this.effect) return;                    // identity, and an early out
bool otherStillUsesOld = false, otherAlreadyUsesNew = false;
foreach part in parent.MeshParts, skipping this:
    Effect other = part.Effect;
    if      (ReferenceEquals(other, this.effect)) otherStillUsesOld  = true;
    else if (ReferenceEquals(other, value))       otherAlreadyUsesNew = true;
if (!otherStillUsesOld  && this.effect != null) parent.Effects.Remove(this.effect);
if (!otherAlreadyUsesNew && value != null)      parent.Effects.Add(value);
this.effect = value;
```

The old effect leaves only when no sibling still uses it; the new one joins only
when no sibling already has. That is what keeps `mesh.Effects` the set of
*distinct* effects its parts use — which is in turn what lets `Model.Draw` set
the transforms once per effect instead of once per part.

Three things a projection must not simplify, and each has a planted defect:
every comparison is `System.Object::ReferenceEquals` and never effect equality;
the early return tests the **field**, so assigning the same effect twice does
nothing at all; and the `else if` is **exclusive**, so a sibling on the old
effect suppresses only the `Remove`.

## Model.Draw draws through a process-wide static array

```
Matrix[] shared = Model.sharedDrawBoneMatrices;      // private STATIC
if (shared == null || shared.Length < bones.Count)
    shared = Model.sharedDrawBoneMatrices = new Matrix[bones.Count];
CopyAbsoluteBoneTransformsTo(shared);
```

`sharedDrawBoneMatrices` really is what its name says: **every Model in the
process draws through the same array.** It grows to fit and is never shrunk, so
its length is the high-water mark of every model drawn so far.

It is reproduced rather than replaced by a per-model buffer because the sharing
is observable: two goroutines drawing two models at once corrupt each other's
transforms exactly as two CLR threads do. `Model.Draw` is not thread-safe in the
reference, and a projection that quietly made it safe would describe a different
runtime.

## CopyAbsoluteBoneTransformsTo reads its own output

```
for (int i = 0; i < bones.Count; i++) {
    ModelBone bone = bones[i];
    if (bone.Parent == null) destination[i] = bone.transform;
    else destination[i] = bone.transform * destination[bone.Parent.Index];
}
```

A single forward pass that reads the slot it wrote earlier. It is correct only
because the content pipeline orders `Bones` so a parent always precedes its
children, and it is the whole reason the member exists — a `ModelBone` cannot
compute this alone, because it does not know the array the walk accumulates
into.

The multiplication order is `local * parent`, not the reverse. Two translations
commute, so the chain test cannot see the order; a separate test uses a
**rotation** and a translation, asserts first that they do *not* commute, and
only then checks the result.

## Two verifier gaps this family exposed, both fixed generally

Neither is a special case for Model. Both existed because every composed BCL
base until now lived in the *same package* as its only consumer and took an
*interface* type argument.

- **`adapterFieldType` rendered neither the package qualifier nor the pointer.**
  `ReadOnlyCollection[T]` is declared in `Microsoft/Xna/Framework` and its four
  consumers are in `Graphics`, so the field reads
  `framework.ReadOnlyCollection[*ModelBone]`. The fix qualifies an *unqualified*
  adapter name and renders a CLR class as a Go pointer — and it deliberately
  leaves `bclexception.State` alone, since that adapter already names its own
  package. Getting that wrong broke all eight exception types, which is how the
  condition was found.
- **The `ICollection<T>` conformance check matched on the GO name.** The
  required set names CLR members, and the two collections with a by-name lookup
  have *two* `Item` members whose Go names are both hashed. Matching on the CLR
  member name is the general rule; matching on the Go name reported the indexer
  missing when it was present twice.

## What is NOT claimed

`ModelMeshPart.draw` reaches the device — `SetVertexBuffer`, `Indices`,
`DrawIndexedPrimitives` — and **no native stress slice exercises it**, because
none can yet. The pinned contract declares **zero public constructors across all
twelve types**; a `Model` reaches a consumer only through
`ContentManager.Load<Model>`, and the content-reader family is not projected.
Nothing outside package `graphics` can build one to draw.

This is recorded as `VERIFIED_MANAGED` rather than `VERIFIED_NATIVE`, and the
native slice is named as work the content-plumbing milestone unblocks. No new
CNA route was bound, so nothing speculative was added: the device calls the
draw path makes are already-projected members with existing call sites.

## Falsifiability

Forty-two defects were planted across the family and the live-list source it
needed. There is no NATIVE table: the family is pure managed, and the one member
that reaches a device has no reachable call site (see above).

Three rounds were needed, and each round's survivors were a different kind of
problem — which is the point of running it more than once.

**Round one — 28 killed, 8 survived, 6 did not compile.** Five survivors were
blind tests and one was a blind *fixture*:

- `absolute_reads_the_parents_LOCAL_transform` survived because the chain
  fixture's root was the IDENTITY matrix. With an identity root every bone's
  parent has the same local and absolute transform, so reading the wrong one is
  invisible. The root now carries a translation, and the test asserts outright
  that the child's local and absolute transforms DIFFER before checking
  anything else.
- `draw_skips_filling_the_bone_array` survived because the test checked the
  array's LENGTH and not its contents. It now compares every element against
  `CopyAbsoluteBoneTransformsTo`'s own output.
- Three `live_*` survivors exposed something structural rather than a weak
  test: `liveListSource`'s bounds checks and its version-checked iterator are
  **unreachable through the Model family**, because `ModelEffectCollection`
  hides `GetEnumerator` and forwards only what it forwards. They are reached
  from the adapter's own package instead, which is where the five new
  `TestLiveList*` tests live.

**Round two — 39 killed, 3 survived.** Two of the three turned out to be
**equivalent mutants**, and saying so is more useful than chasing them:

- `set_effect_flags_are_not_exclusive` widened the second `case` of a Go
  `switch`. Go takes the first matching case, so a part the first case already
  matched never reaches the second — the mutation could not change any
  behaviour. The switch's own semantics enforce the exclusivity the reference's
  `else if` does. The mutation was replaced with the defect that IS
  distinguishable: two independent `if`s.
- `set_effect_adds_a_nil_effect` survived for a reason that is faithful rather
  than accidental. Assigning nil where another part also holds nil finds
  `ReferenceEquals(other, value)` true and suppresses the Add anyway — which is
  what the reference does. Seeing the guard needs every OTHER part to hold a
  non-nil effect, and the test now builds exactly that.

**Round three — 41 killed, 1 survived, 0 skipped.** Every planted defect either
dies or is named below.

| round | planted | killed | survived | skipped |
| --- | ---: | ---: | ---: | ---: |
| one | 42 | 28 | 8 | 6 |
| two | 42 | 39 | 3 | 0 |
| **three** | **42** | **41** | **1** | **0** |

### The one unkilled survivor, named

`mesh_draw_hoists_the_technique` — hoisting `CurrentTechnique.Passes` out of
the inner loop instead of re-fetching it every iteration. Distinguishing the two
requires an effect whose `CurrentTechnique` CHANGES while its passes are being
applied, which needs a live device and a real shader. It is not scored as
killed. The projection re-fetches because the reference re-fetches — IL_0049
repeats the whole `get_CurrentTechnique; get_Passes; get_Item` chain — and the
comment at the call site says so, but no test in this milestone can prove it.
The content-plumbing milestone, which makes a drawable Model reachable, is what
would.

## Scoreboard

| counter | before | after |
| --- | ---: | ---: |
| COMPLETE_TYPES | 194 | 206 |
| MISSING_TYPE | 63 | 51 |
| PARTIAL_TYPES | 0 | 0 |
| MISSING_MEMBER | 0 | 0 |
| GLOBAL_UNREVIEWED | 0 | 0 |
| RESOURCE_STRINGS_VERIFIED | 75 | 77 |
| FRONTIER_FAMILIES | 8 | 7 |
| BOUND_FUNCTIONS | 350 | 350 |
| ABI_MISMATCHES | 0 | 0 |

`BOUND_FUNCTIONS` is unchanged on purpose: the family is pure managed, and the
134 `cna_model*` routes CNA offers — 43 of them `_ext` — describe a model
representation this projection does not reach, because the reference reaches it
through content deserialization rather than through a native model object.
