# CNA-Go

CNA-Go exposes [CNA](https://github.com/openeggbert/cna) to Go through packages
matching the XNA 4.0 namespace hierarchy.

```text
Go game
   ↓
Microsoft/Xna/Framework[/Graphics|Input|Content]
   ↓
internal/interop
   ↓
CNA stable C ABI
   ↓
CNA C++ Microsoft::Xna::Framework implementation
```

## Status

**Early scaffold.** The compatibility package layout and first local values
exist. Native execution waits for the canonical C ABI in `openeggbert/cna`.

```go
import xna "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

position := xna.Vector2{X: 100, Y: 100}
color := xna.CornflowerBlue
```

Go has no C# namespaces, so the import path preserves the hierarchy and the
local package identifier is `framework`. There is deliberately no invented
`CNA/Framework` public package. CNA-specific extensions will be added only when
they mirror real native `CNA::...` APIs.

See [architecture](docs/architecture.md) and [plan](plan.md).

## License

CNA-Go is licensed under the [Microsoft Public License](LICENSE), matching CNA.
