# Architecture

```text
Microsoft/Xna/Framework compatibility packages
                       ↓
internal/interop (private cgo boundary)
                       ↓
CNA stable C ABI
                       ↓
CNA C++: Microsoft::Xna::Framework
```

The public package tree mirrors XNA 4.0. `internal/interop` is the only package
allowed to map C symbols, result codes, handles, callbacks, UTF-8, ownership,
threading, and shutdown. Go's `internal` rule prevents applications from using
that boundary directly.

Pure values remain in Go. Native-backed objects eventually own opaque handles.
No `CNA/Framework` layer exists because CNA C++ has no `CNA::Framework`
namespace. A future public CNA package is valid only for concrete native
extensions that actually exist under `CNA::...`.
