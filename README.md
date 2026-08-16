# CNA-Go

CNA-Go is the Go language binding for [CNA](https://github.com/openeggbert/cna),
the native C++ XNA-inspired game framework. It is intended to provide familiar
`Game`, `GraphicsDevice`, `Texture2D`, `SpriteBatch`, content, and input concepts
through an idiomatic Go API while all engine and renderer work remains in CNA.

```text
Go game → CNA-Go → CNA stable C ABI → CNA C++ → native renderer
```

## Status

**Early scaffold.** This first commit establishes project documentation, local
value types, the Go lifecycle shape, tests, and the private interop boundary.
It does not yet run a game: the stable CNA C ABI it must bind to has not been
implemented in `openeggbert/cna` yet. `cna.Run` therefore returns
`cna.ErrNativeUnavailable` explicitly.

## Design direction

- Preserve CNA/XNA concepts, but use Go interfaces and `error` values.
- Keep math and other small value operations in Go.
- Hide opaque native handles in `internal/interop`.
- Give every owned native resource an explicit, idempotent `Close` method.
- Use UTF-8, fixed-width ABI types, input snapshots, and batched boundary calls.
- Keep Sharp Runtime completely private to the native CNA implementation.

See [the architecture](docs/architecture.md) and [implementation plan](plan.md)
for the boundaries and phased roadmap.

## Development

The scaffold has no third-party Go dependencies:

```bash
go test ./...
```

Go 1.22 or newer is required. A C/C++ toolchain and a CNA native library will
become requirements only when the cgo layer is implemented.

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
See [NOTICE.md](NOTICE.md) for compatibility and attribution notices.
