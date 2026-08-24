# CNA-Go

> **Status:** early, measured binding foundation — functional for the qualified
> Foundation 1 closure, far from full XNA compatibility.

CNA-Go maps Microsoft XNA Framework 4.0 namespaces to Go import paths and
executes native-backed APIs only through CNA's canonical C ABI:

```text
Go game
   ↓
Microsoft/Xna/Framework[/Graphics|Input|Content]
   ↓
internal/interop (the only cgo/native boundary)
   ↓
CNA C ABI 0.7.0
```

There is deliberately no invented public `CNA/Framework` layer. Pure XNA
values are implemented as Go values; public packages expose neither C types
nor native handles.

## What is qualified

The current structural scoreboard maps the authoritative XNA 4.0 Windows
runtime profile (257 types and 2,964 members) to 257 expected Go types and
3,243 expected Go members. The strict verifier remains red because most XNA
surface is intentionally absent. Its leak/mapping gates are green.

Foundation 1 qualifies on **Linux amd64 desktop with cgo**:

- real CNA-driven `Game` lifecycle and tick-exact `GameTime`;
- locked owner OS thread, generation checks, `runtime/cgo.Handle` callbacks,
  and contained callback errors/panics;
- real GraphicsDeviceManager/device, native viewport and clear;
- PNG `Texture2D` creation from `io.Reader`;
- one exact real scaled SpriteBatch draw route;
- native keyboard snapshots with exact XNA `Keys` values;
- a complete first managed closure measured by the verifier and behavior
  corpus.

The admitted qualification artifact uses CNA ABI 0.7.0, the HEADLESS renderer,
and NULL audio. Native draw execution is proven, but visible rendering is not.
Windows, macOS, Android, iOS, and Web/Wasm are not qualified. Content/XNB,
Effects/3D, Audio, Media, Storage, Touch, and most of XNA remain unimplemented.

See the generated [runtime capability inventory](docs/generated/runtime-capabilities.md)
for evidence and limitations by capability.

## Native runtime

The Go build uses cgo but does not link a developer CNA build at compile time.
Supply an exact CNA C ABI 0.7 shared library at runtime:

```sh
export CNA_NATIVE_LIBRARY=/absolute/path/to/libcna_c_api.so
```

The override must be an absolute regular-file path. Without it, the Linux
loader searches for `libcna_c_api.so`. Wrong ABI versions and missing required
symbols fail before Game creation. CNA-Go contains no checkout-relative native
library fallback and does not distribute CNA binaries.

## Development and verification

The maintained sibling `cna-go-template` uses a `go.work` file for local
development. A published module version is not claimed. Final consumer
qualification instead extracts the audited CNA-Go source archive and uses a
temporary `replace` to that exact source tree.

Useful gates are:

```sh
go test ./...
go vet ./...
go test -race ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/capabilities --check
go run ./tools/native_abi -library /absolute/path/to/libcna_c_api.so
go run ./tools/native_stress
```

Normal structural strict mode is expected to exit nonzero until all mapped XNA
surface exists; its diagnostics are the work queue, not a compatibility claim.
The native ABI and stress commands require the qualified native environment.

The normative rules are in [plan.md](plan.md), the language projection in
[docs/xna-go-mapping.md](docs/xna-go-mapping.md), the native boundary in
[docs/native-abi.md](docs/native-abi.md), and the resumable handoff in
[NEXT.md](NEXT.md).

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
