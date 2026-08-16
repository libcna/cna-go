# Architecture

```text
Microsoft/Xna/Framework[/Graphics|Input|Content]
                         ↓
CNA/Framework[/Graphics|Input|Content]
                         ↓
CNA/Interop
                         ↓
CNA stable C ABI
                         ↓
CNA C++ core
```

The `Microsoft/Xna/Framework` import tree is the XNA 4.0 compatibility surface.
`CNA/Framework` is the CNA-native public surface. The compatibility layer may
alias types only where their contracts are genuinely identical; otherwise it
owns facade types and explicit conversions.

`CNA/Interop` is the only package allowed to map native symbols. Raw pointers,
handles, result codes, C++ exceptions, and Sharp Runtime types must not leak to
either public tree. Pure values remain in Go, native resources own opaque
handles, input crosses as snapshots, and repetitive work crosses in batches.
