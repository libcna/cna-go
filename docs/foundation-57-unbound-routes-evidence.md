# Foundation 57 — the routes CNA-Go deliberately does not bind

Foundation 56 justified `GraphicsResource.Name` with a sentence about CNA:

> CNA has no such cache and no route that could reach one.

CNA has twelve. The classification survives measurement, but the reasoning that
produced it did not, and nothing in the project could have caught that — there
was no place to record "CNA offers this and CNA-Go answers it managed-side
anyway", so the claim was prose in a comment.

This milestone closes **zero** members and adds **zero** types. It converts one
wrong sentence into twelve measured, checked decisions.

## What CNA actually offers

```text
cna_graphics_resource_get_graphics_device
cna_graphics_resource_get_is_disposed
cna_graphics_resource_get_name_byte_count
cna_graphics_resource_copy_name
cna_graphics_resource_set_name
cna_graphics_resource_get_string_byte_count
cna_graphics_resource_copy_string
cna_graphics_resource_get_tag
cna_graphics_resource_set_tag
cna_graphics_resource_dispose
cna_graphics_resource_subscribe_disposing
cna_graphics_resource_unsubscribe_disposing
```

All twelve are declared by the pinned 0.21.0 headers and exported by the pinned
library under `CNA_C_API_0.1`. Between them they could back `Name`, `Tag`,
`IsDisposed`, `GraphicsDevice`, `ToString`, `Dispose` and `Disposing` — which is
`GraphicsResource`'s **entire** public surface.

## The measurement

One probe run against `~/deps/cna-c-abi-0.21.0/libcna_c_api.so`, on a live
device inside a lifecycle callback, over a real `Texture2D` handle and a real
`SpriteBatch` handle.

```text
fresh texture   get_is_disposed          -> false, no error
                copy_name                -> ""
                set_name("probe-name")   -> success
                copy_name                -> "probe-name"
                set_name("a\0b")         -> CNA result 11,
                                            "The graphics-resource name is not valid UTF-8."

SpriteBatch     get_is_disposed          -> CNA result 2,
                                            "The handle does not refer to a supported
                                             graphics resource for this call."
                set_name                 -> the same refusal
                get_name_byte_count      -> the same refusal

after cna_graphics_resource_dispose(texture)
                result                   -> success
                get_is_disposed          -> true
                copy_name                -> "probe-name"     (survives)
                cna_texture2d_copy_encoded -> 70 BYTES        (still works)
                a repeated dispose       -> success, as documented

after CNA-Go's own per-kind destroy
                the handle is gone, so CNA-Go answers without a native call
```

Two of those lines decide the whole family.

**The routes refuse a SpriteBatch.** CNA's graphics-resource identity does not
cover its sprite batch; XNA's does — `SpriteBatch extends GraphicsResource` in
the pinned contract, and Foundation 56 composed exactly that. Binding the routes
would make one XNA member answer for textures and fail for sprite batches, which
is a per-kind behaviour the contract does not have.

**`cna_graphics_resource_dispose` is a flag, not a release.** After it, the same
texture still encoded 70 bytes of PNG. The reference's `Dispose(true)` releases
the native object and then sets the flag; CNA's per-kind destroy is what does
the first half, and this route would only set a CNA flag on a resource CNA-Go is
about to destroy.

And one more, from the header rather than the run: `CNA_GraphicsResourceTag` is
a `uint64_t` opaque token. `GraphicsResource::Tag` is `System.Object`, which
projects to Go `any`. A `uint64` cannot carry an arbitrary Go value without a
side registry keyed by a token CNA owns, which would be a second lifetime to get
wrong for a member whose reference implementation is a dictionary entry.

## The registry

`tools/native_abi` now carries `deliberatelyUnboundRoutes`: for each route, the
projected XNA member it could back, a class, and the measured reason.

| class | meaning |
| ----- | ------- |
| `CONTRACT_DIVERGENCE` | binding it would change the member's observable behaviour away from the reference's |
| `KIND_PARTIAL` | the route does not accept every resource kind the member covers |
| `REPRESENTATION` | the C type cannot represent the CLR type |
| `MANAGED_REFERENCE` | the reference's own implementation reaches no runtime at all |

The pinned artifact declares 4,054 routes and CNA-Go binds 78. Almost all of the
difference is uninteresting — the XNA surface those routes would back is not
projected, which the missing-type inventory already says. A route is interesting
only when the member **is** projected and CNA-Go answers it managed-side anyway,
and those are exactly the twelve recorded here.

### It is checked, not merely written down

Four claims, each with a control:

1. every recorded route is declared by the canonical headers — a route CNA
   renames or removes becomes a finding rather than a stale explanation;
2. no recorded route is also bound by the manifest — a route cannot be
   deliberately unbound and resolved at the same time;
3. every entry carries a recorded class, a member and a non-empty detail;
4. no route is recorded twice.

`TestDeliberatelyUnboundRoutesAreChecked` plants (1) and (2) and requires each to
produce a finding. `TestEveryUnboundRouteIsDeclaredByThePinnedHeaders` runs (1)
against the real pinned header tree.

```text
DELIBERATELY_UNBOUND_ROUTES   12
FINDINGS                       0
```

## Corrections

The wrong sentence is corrected in all four places it reached: the
`GraphicsResource` type comment, the `managedStoredMembers` entry in
`tools/api_compat`, the `graphics-resource-chain` capability row, and the
Foundation 56 evidence — which now says what was wrong rather than quietly
reading correctly.

## Qualification

```text
go test ./... / -race                     clean
api_compat report                         204 diagnostics, unchanged
behavior corpus                           694 observations, unchanged
runtime capabilities                      56 rows, PASS
native ABI                                78 bound, 12 deliberately unbound, 0 findings
native stress                             0 crashes
```

Nothing in the projected surface moved, and nothing should have: this milestone
is about whether the reasons behind it are true.
