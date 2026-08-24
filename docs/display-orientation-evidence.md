# Foundation Milestone 7 DisplayOrientation evidence

## Authority and exact closure

Foundation Milestone 7 completes exactly:

```text
Microsoft.Xna.Framework.DisplayOrientation
Microsoft.Xna.Framework.GraphicsDeviceManager.SupportedOrientations
```

The pinned public contract remains
`tools/api_compat/reference/xna40-windows-runtime-contract.json` at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.
The directly inspected Windows XNA assemblies match the retained hashes:

```text
Microsoft.Xna.Framework.dll
38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130

Microsoft.Xna.Framework.Game.dll
b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
```

The closure contains six CLR identities. `DisplayOrientation` contributes five
source fields including synthetic `value__`; the property contributes one CLR
identity. The enum projection excludes `value__`, while the writable property
expands into a getter and setter, producing six mapped Go identities.

## DisplayOrientation contract

`DisplayOrientation` is a `[Flags]` enum with `System.Int32` underlying storage.
It maps to a named Go `int32` marked by the established `// xna:flags`
directive. All constants are explicit; no `iota` or helper API is present.

| XNA literal | Go identity | Raw value |
|---|---|---:|
| `Default` | `DisplayOrientationDefault` | 0 |
| `LandscapeLeft` | `DisplayOrientationLandscapeLeft` | 1 |
| `LandscapeRight` | `DisplayOrientationLandscapeRight` | 2 |
| `Portrait` | `DisplayOrientationPortrait` | 4 |

The complete `int32` domain remains representable. Named flags can be combined
without normalization: left plus right is raw `3`, left plus portrait is `5`,
and all three are `7`. An unknown bit such as `1<<20` is likewise preserved.
No ordinary value boundary validates membership or requires one declared name.

## Direct GraphicsDeviceManager IL

The reference class declares private fields named `supportedOrientations` and
`isDeviceDirty`. Its 228-byte constructor assigns several unrelated defaults,
registers services and window events, and reads the default graphics profile.
It contains no store to either selected field. CLR zero initialization therefore
establishes the exact initial state:

```text
SupportedOrientations = DisplayOrientation.Default (0)
isDeviceDirty = false
```

The seven-byte getter is only a field load:

```text
IL_0000: ldarg.0
IL_0001: ldfld DisplayOrientation ...::supportedOrientations
IL_0006: ret
```

The 15-byte setter always performs both stores:

```text
IL_0000: ldarg.0
IL_0001: ldarg.1
IL_0002: stfld DisplayOrientation ...::supportedOrientations
IL_0007: ldarg.0
IL_0008: ldc.i4.1
IL_0009: stfld bool ...::isDeviceDirty
IL_000e: ret
```

There is no comparison, branch, validation, call, window/device access,
platform operation, or `ApplyChanges` invocation. Consequently the same-value
setter marks the manager dirty just like a changed-value setter. Every setter
stores the exact raw bit pattern and leaves dirty true. The setter has no
value-dependent exception path.

## Go state and lifecycle behavior

The exact public projection is:

```go
func (m *GraphicsDeviceManager) SupportedOrientations() DisplayOrientation
func (m *GraphicsDeviceManager) SetSupportedOrientations(value DisplayOrientation)
```

Neither accessor returns `error`. A native-backed owner does not make a
managed field accessor fallible. `GraphicsDeviceManager` now retains private
`supportedOrientations` and `isDeviceDirty` fields initialized to the XNA
constructor defaults. The setter always stores and marks dirty; no public
dirty accessor or test-only XNA member was added.

Package-internal tests cover initial dirty state, same-value assignment,
changed assignment, multiple assignments, combinations, and unknown bits.
Native lifecycle stress proves the public default and storage behavior during
`Initialize`, in a later callback, and after real manager disposal. Post-disposal
access remains managed and succeeds because neither accessor checks or enters
the disposed native resource. Constructor failure still returns before a
manager is created or registered.

The accessors contain zero native calls. No CNA, C ABI, cgo, symbol manifest,
layout, callback, constant, or `internal/interop` source changed. Actual window
orientation application is deliberately deferred to the future XNA operations
that consume the dirty state; no platform rotation capability is claimed.

## Structural, behavior, and negative evidence

The dedicated generated closure measurement reports:

| Slice | Source identities | Expected Go | Target Go | Local diagnostics |
|---|---:|---:|---:|---:|
| complete `DisplayOrientation` type | 5 | 4 | 4 | 0 |
| selected `SupportedOrientations` property | 1 | 2 | 2 | 0 |
| total | 6 | 6 | 6 | 0 |

`GraphicsDeviceManager` remains partial: its missing member count moves only
from 42 to 40. The other four partial types are unchanged. Every global
unexpected, kind, base/interface, field/property, signature, parameter,
return/error, overload/generic, enum/flags, event/operator/ref-out/language,
native-leak, allowlist, and unmeasured counter remains zero.

The `PURE_XNA_DERIVED` corpus grows from 234 to 242 observations/assertions with
zero failures. `DISPLAY_ORIENTATION` contributes three observations for exact
values, combinations, and an unknown bit. `GRAPHICS_MANAGER_ORIENTATION`
contributes five observations for initial state, same-value dirty behavior,
changed-value dirty behavior, multiple assignment, and managed post-disposal
state.

The verifier mutation inventory grows from 58 to 73 cases. The 15 focused
additions reject a wrong enum kind or underlying type, a missing flags marker,
wrong endpoint values, a missing literal, missing or read-only accessors, wrong
getter/setter types, a synthetic setter error, static or wrong-package
projection, and a public dirty-helper leak. No allowlist is used.

## ABI and scope boundary

Native ABI evidence remains exactly 23 bound functions, 67 prototype type
positions, 96 aggregate C/Go measurements, 28 layouts, two callbacks, and five
constants, with zero missing header/library symbols and zero mismatches.

This milestone adds no `ApplyChanges`, other GraphicsDeviceManager property or
event, GameWindow/current-orientation behavior, GraphicsDeviceInformation,
PresentationParameters, input family, vertex declaration/buffer, Content,
Effects, or platform-orientation claim.
