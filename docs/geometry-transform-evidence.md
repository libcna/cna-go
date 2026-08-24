# Foundation Milestone 2 geometry and transform evidence

## Authority and computed closure

The dependency graph below was regenerated from the pinned Microsoft XNA
Framework 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json` (SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`).
Dependencies were computed independently from public base/interface, field,
property, parameter, and return type identities. BCL identities and self
references were excluded.

Starting with `Vector2`, `Vector3`, `Vector4`, `Quaternion`, and `Matrix` does
not produce a closed public surface. `Matrix.CreateReflection` and
`Matrix.CreateShadow` directly require `Plane`. `Plane` directly requires
`BoundingBox`, `BoundingSphere`, `BoundingFrustum`, and
`PlaneIntersectionType`. The bounding contracts then introduce `Ray` and
`ContainmentType`, and mutually reference the other bounding types. The exact
selected closure is therefore:

```text
Vector2
Vector3
Vector4
Quaternion
Matrix
Plane
Ray
BoundingBox
BoundingSphere
BoundingFrustum
ContainmentType
PlaneIntersectionType
```

The reason each type beyond the five original linear-algebra types entered is:

| Added type | First direct public-signature reason |
|---|---|
| `Plane` | `Matrix.CreateReflection` and `Matrix.CreateShadow` consume it. |
| `BoundingBox` | `Plane.Intersects(BoundingBox)` consumes it. |
| `BoundingSphere` | `Plane.Intersects(BoundingSphere)` consumes it. |
| `BoundingFrustum` | `Plane.Intersects(BoundingFrustum)` consumes it. |
| `PlaneIntersectionType` | The `Plane.Intersects` classification overloads return it. |
| `Ray` | Bounding-box/sphere/frustum intersection contracts consume or return nullable distances for it. |
| `ContainmentType` | Bounding-box/sphere/frustum `Contains` contracts return it. |

No type outside that recursive closure was pulled into the primary milestone.
`Color` and `Graphics.Viewport` were completed only after the closure because
their already-partial contracts were directly unblocked by the new vector and
matrix values.

## Regenerated source and mapped inventory

Counts below are source-member identities and verifier-derived Go-member
identities, not hand-forced targets. `REF_PARAMETERS` and `OUT_PARAMETERS`
count source parameter positions. `ARRAY_OVERLOADS` counts source members with
an array parameter. Operators are source `op_*` methods; static-method counts
exclude operators. Enum source counts include the CLR `value__` backing field,
which is intentionally absent from the Go count.

| Type | Source members | Expected Go members | Status before -> after | Direct XNA dependencies | Ref parameters | Out parameters | Nullable returns | Array overloads | Operators | Static properties | Static methods |
|---|---:|---:|---|---|---:|---:|---:|---:|---:|---:|---:|
| `Vector2` | 77 | 77 | partial -> complete | `Matrix`, `Quaternion` | 54 | 23 | 0 | 6 | 10 | 4 | 52 |
| `Vector3` | 88 | 88 | missing -> complete | `Matrix`, `Quaternion`, `Vector2` | 56 | 24 | 0 | 6 | 10 | 11 | 54 |
| `Vector4` | 85 | 85 | missing -> complete | `Matrix`, `Quaternion`, `Vector2`, `Vector3` | 56 | 25 | 0 | 4 | 10 | 6 | 54 |
| `Quaternion` | 55 | 55 | missing -> complete | `Matrix`, `Vector3` | 23 | 16 | 0 | 0 | 8 | 1 | 32 |
| `Matrix` | 107 | 114 | missing -> complete | `Plane`, `Quaternion`, `Vector3` | 36 | 36 | 0 | 0 | 10 | 1 | 66 |
| `Plane` | 30 | 30 | missing -> complete | `BoundingBox`, `BoundingFrustum`, `BoundingSphere`, `Matrix`, `PlaneIntersectionType`, `Quaternion`, `Vector3`, `Vector4` | 10 | 8 | 0 | 0 | 2 | 0 | 6 |
| `Ray` | 16 | 16 | missing -> complete | `BoundingBox`, `BoundingFrustum`, `BoundingSphere`, `Plane`, `Vector3` | 3 | 3 | 4 | 0 | 2 | 0 | 0 |
| `BoundingBox` | 33 | 33 | missing -> complete | `BoundingFrustum`, `BoundingSphere`, `ContainmentType`, `Plane`, `PlaneIntersectionType`, `Ray`, `Vector3` | 10 | 9 | 1 | 1 | 2 | 0 | 5 |
| `BoundingSphere` | 33 | 33 | missing -> complete | `BoundingBox`, `BoundingFrustum`, `ContainmentType`, `Matrix`, `Plane`, `PlaneIntersectionType`, `Ray`, `Vector3` | 11 | 10 | 1 | 0 | 2 | 0 | 6 |
| `BoundingFrustum` | 33 | 34 | missing -> complete | `BoundingBox`, `BoundingSphere`, `ContainmentType`, `Matrix`, `Plane`, `PlaneIntersectionType`, `Ray`, `Vector3` | 7 | 7 | 1 | 1 | 2 | 0 | 0 |
| `ContainmentType` | 4 | 3 | missing -> complete | none | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `PlaneIntersectionType` | 4 | 3 | missing -> complete | none | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

The secondary completion inventory is `Color` 165 source / 170 expected Go
members and `Graphics.Viewport` 14 source / 21 expected Go members. Both moved
from partial to complete without expanding `GraphicsDevice` or another native
runtime type.

## Nullable, ref/out, and arrays

`System.Nullable<T>` remains a measured source identity. A nullable input maps
to `*T` (`nil` is null); a nullable return maps to `(T, bool)` and an
`out Nullable<T>` appends the same `(T, bool)` pair. The Boolean means
`hasValue`, not failure. A genuine Go failure, where the formal runtime mapping
adds one, remains a separate final `error`. No sentinel float or NaN-as-null is
used. Verifier self-tests independently cover input, return, out, and final
error result ordering.

Every source `ref T` remains a `*T` input, every `out T` is an additional Go
result, and ref/out identities keep their generated overload shapes. Pointer
inputs are read or mutated only during the call and are not retained.

Vector array transforms write into caller-owned destination slices in source
loop order. They do not allocate replacement destinations. This preserves XNA
observable behavior for identical or overlapping source/destination storage,
including forward-overlap mutation. Nil slices, invalid positive ranges, and
invalid indices panic as the managed argument-failure projection; a negative
length performs no iterations, matching retained XNA observations.

## Binary32 policy

Public `System.Single` values and all stored/intermediate scalar values are
`float32`. Arithmetic is written as binary32 operations so rounding occurs at
the same expression boundaries as XNA; algorithms are not accumulated in a
float64 pipeline and cast only at the end. Transcendental operations use narrow
helpers that widen one `float32` argument only for the Go standard-library
call, then immediately round its result to `float32`.

Behavior fixtures compare `math.Float32bits` for signed zero, canonical NaN
results, infinity, large angles, reciprocal-first division, singular inversion,
and representative quaternion/matrix branches. XNA-era CLR `Single.GetHashCode`
is reproduced directly: both zeros hash as zero and other bit patterns retain
their signed 32-bit representation.

## Vector and quaternion conventions

`Vector3.Forward` is `(0,0,-1)` and `Backward` is `(0,0,1)`. The asymmetric
cross-product fixture proves `Right x Up == Backward`. Vector transforms use
XNA's row-vector matrix equations and the selected quaternion expansion.

Quaternion multiplication, `Concatenate`, axis-angle, yaw/pitch/roll, matrix
conversion, inverse, normalize, lerp, and slerp are independent mapped members.
`Concatenate(a,b)` applies `a` then `b`, corresponding to the retained XNA
result `Multiply(b,a)`. Golden branches cover multiply order, shortest-path
slerp, a nontrivial interpolation fraction, large-angle trigonometry, zero
inverse, and matrix conversion. Quaternion sign is never normalized merely
because `q` and `-q` represent the same rotation.

## Matrix conventions

`Matrix` is a Go value struct with exactly the public fields `M11` through
`M44`. It uses XNA row vectors, translation in `M41`-`M43`, `Forward == -Z`,
and XNA multiplication order. View and projection builders use the same
right-handed view and depth conventions. Asymmetric golden fixtures cover
translation placement, multiplication/inversion residual bits, rotations,
perspective infinity behavior, look-at/projection use through the frustum and
viewport, and exact string field order.

`Decompose` maps its Boolean source return separately from its three out values:
`(bool, Vector3, Quaternion, Vector3)`. A mirrored non-uniform transform is
covered at exact bits. Singular inversion returns the same non-finite matrix
behavior rather than a Go error. `CreateReflection` and `CreateShadow` are
present and consume the completed `Plane` type; the ref reflection overload's
observable plane normalization is covered.

## Plane, ray, and bounds

Plane construction, normalization, dot families, matrix/quaternion transforms,
and classification overloads are complete. Degenerate three-point creation,
near-unit normalization, a coplanar box, and reflection normalization have
XNA-derived fixtures.

Ray fields are values. Its four intersection families return nullable distances
as `(float32, bool)`. Fixtures distinguish hit, null/miss, near-parallel box and
plane cases, and the XNA just-behind tolerance that clamps a valid intersection
to positive zero.

Bounding box/sphere creation, merge, contains, intersects, corners, hash,
equality, and string identities are complete. Fixtures include edge and tangent
classification, NaN behavior, exact box corner order, and the XNA point-cloud
sphere result. Sphere transform uses XNA's maximum transformed-axis scale for a
non-uniform or reflected matrix.

`BoundingFrustum` is a managed class projection and therefore a pointer facade,
not a value struct. It has no CNA handle. Matrix, six planes, and eight corners
are private managed state; the public `Matrix` setter rebuilds them. Plane
extraction and all eight corner positions follow XNA matrix conventions. The
intersection implementation includes a private GJK simplex and exposes no
helper type. Asymmetric fixtures cover plane bits, ordered corners, box/sphere/
frustum GJK results, ray distance, nil comparison, and matrix-based equality.

## Color and Viewport

Color float/vector constructors preserve XNA clamp-and-truncate packing,
including NaN and infinities. `ToVector3`, `ToVector4`, both
`FromNonPremultiplied` overloads, lerp, multiply, equality, operators, hash, and
string identities are complete. All 141 static predefined properties are
tested against an independently transcribed XNA-derived packed-value table;
`Transparent` is `0x00FFFFFF`, not zero.

`Graphics.Viewport.Project` and `Unproject` are managed matrix operations. The
golden viewport has nonzero X/Y, nondefault depth range, nonidentity world/view/
projection, exact projected and unprojected bit patterns, and singular inverse
NaN behavior. No graphics-device member or CNA ABI route was added.

`ToString` implementations use XNA field labels/order and invariant-style
single formatting. This milestone intentionally does not introduce a general
`CultureInfo` subsystem; culture-specific overloads do not exist in the
selected contracts.

## Strict local completion matrix

The final verifier-derived global report has 29 target types, 1,033 target
members, 23 complete types, six partial types, and 228 missing types. Each row
below has its full expected target count and zero local diagnostics in every
measured category.

| Type | Reference members | Expected Go members | Target Go members | Local diagnostics | Behavior |
|---|---:|---:|---:|---:|---|
| `Vector2` | 77 | 77 | 77 | 0 | PASS |
| `Vector3` | 88 | 88 | 88 | 0 | PASS |
| `Vector4` | 85 | 85 | 85 | 0 | PASS |
| `Quaternion` | 55 | 55 | 55 | 0 | PASS |
| `Matrix` | 107 | 114 | 114 | 0 | PASS |
| `Plane` | 30 | 30 | 30 | 0 | PASS |
| `Ray` | 16 | 16 | 16 | 0 | PASS |
| `BoundingBox` | 33 | 33 | 33 | 0 | PASS |
| `BoundingSphere` | 33 | 33 | 33 | 0 | PASS |
| `BoundingFrustum` | 33 | 34 | 34 | 0 | PASS |
| `ContainmentType` | 4 | 3 | 3 | 0 | PASS |
| `PlaneIntersectionType` | 4 | 3 | 3 | 0 | PASS |
| `Color` | 165 | 170 | 170 | 0 | PASS |
| `Graphics.Viewport` | 14 | 21 | 21 | 0 | PASS |

Globally, all unexpected-symbol, kind, base/interface, field/property, method,
parameter/return/error, overload/generic, enum/flags, event/operator/ref-out/
language, native-leak, allowlist, and unmeasured counters remain zero. The 408
remaining diagnostics are exactly 228 missing types plus 180 members on the six
explicitly deferred native/runtime partial types.
