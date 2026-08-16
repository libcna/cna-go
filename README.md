# CNA-Go

CNA-Go exposes [CNA](https://github.com/openeggbert/cna) to Go while preserving
the CNA and XNA 4.0 namespace hierarchy in import paths.

```text
Go game
   ↓
Microsoft/Xna/Framework compatibility packages
   ↓
CNA/Framework packages
   ↓
CNA/Interop → stable CNA C ABI → CNA C++
```

## Status

**Early scaffold.** The namespace/package layout and first local values exist.
The native C ABI is not implemented upstream yet, so the binding does not run a
game today.

## Package roots

- `github.com/openeggbert/cna-go/CNA/Framework`
- `github.com/openeggbert/cna-go/CNA/Framework/Graphics`
- `github.com/openeggbert/cna-go/CNA/Framework/Input`
- `github.com/openeggbert/cna-go/CNA/Framework/Content`
- `github.com/openeggbert/cna-go/Microsoft/Xna/Framework`
- matching `Graphics`, `Input`, and `Content` compatibility packages
- `CNA/Interop`, reserved for the low-level C ABI mapping

Go does not have C# namespaces. The import path preserves the hierarchy, while
the local package identifier remains the legal Go identifier `framework`.

```go
import xna "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

position := xna.Vector2{X: 100, Y: 100}
color := xna.CornflowerBlue
```

See [architecture](docs/architecture.md) and [plan](plan.md).

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
