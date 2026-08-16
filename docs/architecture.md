# Architecture

```text
Go game
   ↓
github.com/openeggbert/cna-go/cna (idiomatic public API)
   ↓
internal/interop (private cgo layer)
   ↓
CNA stable C ABI
   ↓
CNA C++ core → Sharp Runtime, subsystems, renderers
```

The public API preserves CNA/XNA concepts while following Go conventions:
the game lifecycle is an interface, failures are `error` values, and native
resources will implement explicit `Close` methods. Finalizers may eventually
act as a diagnostic safety net, never as the primary GPU-resource lifecycle.

Pure values such as `Vector2`, `Color`, and future matrix types live entirely
in Go. Native objects such as textures and sprite batches will keep opaque CNA
handles private. Strings cross the ABI as UTF-8; callbacks use C function
pointers plus rooted caller context; input crosses as snapshots; bulk drawing
and uploads are batched.

Sharp Runtime is below the C ABI. Go code never links to it or relies on its
types, ownership model, exceptions, or binary layout.

The native layer is deliberately empty until `openeggbert/cna` publishes the
canonical ABI headers. This avoids freezing guessed declarations into a public
Go module.
