# Foundation Milestone 14 pure-managed batch A evidence

## Authority and batch scope

Foundation Milestone 14 is a multi-type autonomous batch. It completes
exactly **25 public XNA types** carrying **121 mapped Go identities**, all of
them ordinary or flags enums whose only public-signature dependency is
`System.Int32`.

The public authority is the pinned XNA 4.0 Windows runtime contract at
`tools/api_compat/reference/xna40-windows-runtime-contract.json`, SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`. Every
entry below was inspected independently from that contract rather than taken
from the milestone brief. Each has CLR kind `enum`, base `System.Enum`,
`System.Int32` underlying storage, no direct interfaces, no generic
parameters, and no declared constructor, method, property, event, or
operator. The inherited `System.Enum` interface list (`IComparable`,
`IConvertible`, `IFormattable`) is CLR base surface and contributes nothing
to the direct Go public projection.

The batch stopped at stop condition **A — 25 newly complete types** — before
the 150-identity ceiling was reached. Safe dependency-complete candidates
still remain; they are listed under *Remaining safe candidates*.

## Ordered completed types

Types were consumed in the established ranking order: highest reverse
fan-out first, ties broken by the smallest source closure, then by identity.
The dependency graph was regenerated and reranked after every completed type.

| # | XNA type | Kind | Flags | Source ids | Expected Go ids | Target Go ids | Fan-out | Local diagnostics |
|---:|---|---|---|---:|---:|---:|---:|---:|
| 1 | `Microsoft.Xna.Framework.Graphics.RenderTargetUsage` | enum | false | 4 | 3 | 3 | 3 | 0 |
| 2 | `Microsoft.Xna.Framework.Graphics.CubeMapFace` | enum | false | 7 | 6 | 6 | 3 | 0 |
| 3 | `Microsoft.Xna.Framework.Audio.AudioChannels` | enum | false | 3 | 2 | 2 | 2 | 0 |
| 4 | `Microsoft.Xna.Framework.Audio.AudioStopOptions` | enum | false | 3 | 2 | 2 | 2 | 0 |
| 5 | `Microsoft.Xna.Framework.Graphics.IndexElementSize` | enum | false | 3 | 2 | 2 | 2 | 0 |
| 6 | `Microsoft.Xna.Framework.Graphics.SetDataOptions` | enum | true | 4 | 3 | 3 | 2 | 0 |
| 7 | `Microsoft.Xna.Framework.Media.MediaState` | enum | false | 4 | 3 | 3 | 2 | 0 |
| 8 | `Microsoft.Xna.Framework.Graphics.EffectParameterClass` | enum | false | 6 | 5 | 5 | 2 | 0 |
| 9 | `Microsoft.Xna.Framework.Graphics.CompareFunction` | enum | false | 9 | 8 | 8 | 2 | 0 |
| 10 | `Microsoft.Xna.Framework.Graphics.EffectParameterType` | enum | false | 11 | 10 | 10 | 2 | 0 |
| 11 | `Microsoft.Xna.Framework.Input.Touch.GestureType` | enum | true | 12 | 11 | 11 | 2 | 0 |
| 12 | `Microsoft.Xna.Framework.Input.Buttons` | enum | true | 26 | 25 | 25 | 2 | 0 |
| 13 | `Microsoft.Xna.Framework.Audio.MicrophoneState` | enum | false | 3 | 2 | 2 | 1 | 0 |
| 14 | `Microsoft.Xna.Framework.Graphics.FillMode` | enum | false | 3 | 2 | 2 | 1 | 0 |
| 15 | `Microsoft.Xna.Framework.Media.MediaSourceType` | enum | false | 3 | 2 | 2 | 1 | 0 |
| 16 | `Microsoft.Xna.Framework.Audio.SoundState` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 17 | `Microsoft.Xna.Framework.Graphics.CullMode` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 18 | `Microsoft.Xna.Framework.Graphics.GraphicsDeviceStatus` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 19 | `Microsoft.Xna.Framework.Graphics.TextureAddressMode` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 20 | `Microsoft.Xna.Framework.Input.GamePadDeadZone` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 21 | `Microsoft.Xna.Framework.Media.VideoSoundtrackType` | enum | false | 4 | 3 | 3 | 1 | 0 |
| 22 | `Microsoft.Xna.Framework.Graphics.PresentInterval` | enum | false | 5 | 4 | 4 | 1 | 0 |
| 23 | `Microsoft.Xna.Framework.Graphics.PrimitiveType` | enum | false | 5 | 4 | 4 | 1 | 0 |
| 24 | `Microsoft.Xna.Framework.Input.Touch.TouchLocationState` | enum | false | 5 | 4 | 4 | 1 | 0 |
| 25 | `Microsoft.Xna.Framework.Graphics.BlendFunction` | enum | false | 6 | 5 | 5 | 1 | 0 |

Totals: 25 types, 150 source identities (121 named literals plus 25
synthetic `value__` storage fields), 121 expected Go identities, 121 target
Go identities, and zero local diagnostics for every type.

## Per-type contracts, dependencies, and reverse fan-out

Every type below has an **empty public-signature dependency set**: its only
signature type is `System.Int32`, which the BCL mapping table already covers.
No entry required a new mapping rule, so `docs/xna-go-mapping.md` is
unchanged.

### 1. `Microsoft.Xna.Framework.Graphics.RenderTargetUsage`

`RenderTargetUsage` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/render_target_usage.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `DiscardContents` | `RenderTargetUsageDiscardContents` | 0 |
| `PreserveContents` | `RenderTargetUsagePreserveContents` | 1 |
| `PlatformContents` | `RenderTargetUsagePlatformContents` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `RenderTargetUsageDiscardContents`.

Reverse fan-out 3 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.PresentationParameters`
- `Microsoft.Xna.Framework.Graphics.RenderTarget2D`
- `Microsoft.Xna.Framework.Graphics.RenderTargetCube`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 2. `Microsoft.Xna.Framework.Graphics.CubeMapFace`

`CubeMapFace` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/cube_map_face.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `PositiveX` | `CubeMapFacePositiveX` | 0 |
| `NegativeX` | `CubeMapFaceNegativeX` | 1 |
| `PositiveY` | `CubeMapFacePositiveY` | 2 |
| `NegativeY` | `CubeMapFaceNegativeY` | 3 |
| `PositiveZ` | `CubeMapFacePositiveZ` | 4 |
| `NegativeZ` | `CubeMapFaceNegativeZ` | 5 |

Source identities 7 (synthetic `value__` plus 6 named literals);
expected Go identities 6; target Go identities 6; local diagnostics 0.
The Go zero value equals `CubeMapFacePositiveX`.

Reverse fan-out 3 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` *(protected partial runtime type; not modified)*
- `Microsoft.Xna.Framework.Graphics.RenderTargetBinding`
- `Microsoft.Xna.Framework.Graphics.TextureCube`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 3. `Microsoft.Xna.Framework.Audio.AudioChannels`

`AudioChannels` maps to package `audio` at `Microsoft/Xna/Framework/Audio/audio_channels.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Mono` | `AudioChannelsMono` | 1 |
| `Stereo` | `AudioChannelsStereo` | 2 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The pinned contract declares **no zero literal**, so the Go zero value is
an ordinary undefined raw value and equals no named constant.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Audio.DynamicSoundEffectInstance`
- `Microsoft.Xna.Framework.Audio.SoundEffect`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 4. `Microsoft.Xna.Framework.Audio.AudioStopOptions`

`AudioStopOptions` maps to package `audio` at `Microsoft/Xna/Framework/Audio/audio_stop_options.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `AsAuthored` | `AudioStopOptionsAsAuthored` | 0 |
| `Immediate` | `AudioStopOptionsImmediate` | 1 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The Go zero value equals `AudioStopOptionsAsAuthored`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Audio.AudioCategory`
- `Microsoft.Xna.Framework.Audio.Cue`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 5. `Microsoft.Xna.Framework.Graphics.IndexElementSize`

`IndexElementSize` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/index_element_size.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `SixteenBits` | `IndexElementSizeSixteenBits` | 0 |
| `ThirtyTwoBits` | `IndexElementSizeThirtyTwoBits` | 1 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The Go zero value equals `IndexElementSizeSixteenBits`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer`
- `Microsoft.Xna.Framework.Graphics.IndexBuffer`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 6. `Microsoft.Xna.Framework.Graphics.SetDataOptions`

`SetDataOptions` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/set_data_options.go` as a named `int32` type carrying the `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `None` | `SetDataOptionsNone` | 0 |
| `Discard` | `SetDataOptionsDiscard` | 1 |
| `NoOverwrite` | `SetDataOptionsNoOverwrite` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `SetDataOptionsNone`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer`
- `Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 7. `Microsoft.Xna.Framework.Media.MediaState`

`MediaState` maps to package `media` at `Microsoft/Xna/Framework/Media/media_state.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Stopped` | `MediaStateStopped` | 0 |
| `Playing` | `MediaStatePlaying` | 1 |
| `Paused` | `MediaStatePaused` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `MediaStateStopped`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Media.MediaPlayer`
- `Microsoft.Xna.Framework.Media.VideoPlayer`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 8. `Microsoft.Xna.Framework.Graphics.EffectParameterClass`

`EffectParameterClass` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/effect_parameter_class.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Scalar` | `EffectParameterClassScalar` | 0 |
| `Vector` | `EffectParameterClassVector` | 1 |
| `Matrix` | `EffectParameterClassMatrix` | 2 |
| `Object` | `EffectParameterClassObject` | 3 |
| `Struct` | `EffectParameterClassStruct` | 4 |

Source identities 6 (synthetic `value__` plus 5 named literals);
expected Go identities 5; target Go identities 5; local diagnostics 0.
The Go zero value equals `EffectParameterClassScalar`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.EffectAnnotation`
- `Microsoft.Xna.Framework.Graphics.EffectParameter`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 9. `Microsoft.Xna.Framework.Graphics.CompareFunction`

`CompareFunction` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/compare_function.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Always` | `CompareFunctionAlways` | 0 |
| `Never` | `CompareFunctionNever` | 1 |
| `Less` | `CompareFunctionLess` | 2 |
| `LessEqual` | `CompareFunctionLessEqual` | 3 |
| `Equal` | `CompareFunctionEqual` | 4 |
| `GreaterEqual` | `CompareFunctionGreaterEqual` | 5 |
| `Greater` | `CompareFunctionGreater` | 6 |
| `NotEqual` | `CompareFunctionNotEqual` | 7 |

Source identities 9 (synthetic `value__` plus 8 named literals);
expected Go identities 8; target Go identities 8; local diagnostics 0.
The Go zero value equals `CompareFunctionAlways`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.AlphaTestEffect`
- `Microsoft.Xna.Framework.Graphics.DepthStencilState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 10. `Microsoft.Xna.Framework.Graphics.EffectParameterType`

`EffectParameterType` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/effect_parameter_type.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Void` | `EffectParameterTypeVoid` | 0 |
| `Bool` | `EffectParameterTypeBool` | 1 |
| `Int32` | `EffectParameterTypeInt32` | 2 |
| `Single` | `EffectParameterTypeSingle` | 3 |
| `String` | `EffectParameterTypeString` | 4 |
| `Texture` | `EffectParameterTypeTexture` | 5 |
| `Texture1D` | `EffectParameterTypeTexture1D` | 6 |
| `Texture2D` | `EffectParameterTypeTexture2D` | 7 |
| `Texture3D` | `EffectParameterTypeTexture3D` | 8 |
| `TextureCube` | `EffectParameterTypeTextureCube` | 9 |

Source identities 11 (synthetic `value__` plus 10 named literals);
expected Go identities 10; target Go identities 10; local diagnostics 0.
The Go zero value equals `EffectParameterTypeVoid`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.EffectAnnotation`
- `Microsoft.Xna.Framework.Graphics.EffectParameter`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 11. `Microsoft.Xna.Framework.Input.Touch.GestureType`

`GestureType` maps to package `touch` at `Microsoft/Xna/Framework/Input/Touch/gesture_type.go` as a named `int32` type carrying the `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `None` | `GestureTypeNone` | 0 |
| `Tap` | `GestureTypeTap` | 1 |
| `DoubleTap` | `GestureTypeDoubleTap` | 2 |
| `Hold` | `GestureTypeHold` | 4 |
| `HorizontalDrag` | `GestureTypeHorizontalDrag` | 8 |
| `VerticalDrag` | `GestureTypeVerticalDrag` | 16 |
| `FreeDrag` | `GestureTypeFreeDrag` | 32 |
| `Pinch` | `GestureTypePinch` | 64 |
| `Flick` | `GestureTypeFlick` | 128 |
| `DragComplete` | `GestureTypeDragComplete` | 256 |
| `PinchComplete` | `GestureTypePinchComplete` | 512 |

Source identities 12 (synthetic `value__` plus 11 named literals);
expected Go identities 11; target Go identities 11; local diagnostics 0.
The Go zero value equals `GestureTypeNone`.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Input.Touch.GestureSample`
- `Microsoft.Xna.Framework.Input.Touch.TouchPanel`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 12. `Microsoft.Xna.Framework.Input.Buttons`

`Buttons` maps to package `input` at `Microsoft/Xna/Framework/Input/buttons.go` as a named `int32` type carrying the `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `DPadUp` | `ButtonsDPadUp` | 1 |
| `DPadDown` | `ButtonsDPadDown` | 2 |
| `DPadLeft` | `ButtonsDPadLeft` | 4 |
| `DPadRight` | `ButtonsDPadRight` | 8 |
| `Start` | `ButtonsStart` | 16 |
| `Back` | `ButtonsBack` | 32 |
| `LeftStick` | `ButtonsLeftStick` | 64 |
| `RightStick` | `ButtonsRightStick` | 128 |
| `LeftShoulder` | `ButtonsLeftShoulder` | 256 |
| `RightShoulder` | `ButtonsRightShoulder` | 512 |
| `BigButton` | `ButtonsBigButton` | 2048 |
| `A` | `ButtonsA` | 4096 |
| `B` | `ButtonsB` | 8192 |
| `X` | `ButtonsX` | 16384 |
| `Y` | `ButtonsY` | 32768 |
| `RightThumbstickUp` | `ButtonsRightThumbstickUp` | 16777216 |
| `RightThumbstickDown` | `ButtonsRightThumbstickDown` | 33554432 |
| `RightThumbstickRight` | `ButtonsRightThumbstickRight` | 67108864 |
| `RightThumbstickLeft` | `ButtonsRightThumbstickLeft` | 134217728 |
| `LeftThumbstickUp` | `ButtonsLeftThumbstickUp` | 268435456 |
| `LeftThumbstickDown` | `ButtonsLeftThumbstickDown` | 536870912 |
| `LeftThumbstickRight` | `ButtonsLeftThumbstickRight` | 1073741824 |
| `LeftThumbstickLeft` | `ButtonsLeftThumbstickLeft` | 2097152 |
| `LeftTrigger` | `ButtonsLeftTrigger` | 8388608 |
| `RightTrigger` | `ButtonsRightTrigger` | 4194304 |

Source identities 26 (synthetic `value__` plus 25 named literals);
expected Go identities 25; target Go identities 25; local diagnostics 0.
The pinned contract declares **no zero literal**, so the Go zero value is
an ordinary undefined raw value and equals no named constant.

Reverse fan-out 2 — deferred direct consumers:

- `Microsoft.Xna.Framework.Input.GamePadButtons`
- `Microsoft.Xna.Framework.Input.GamePadState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 13. `Microsoft.Xna.Framework.Audio.MicrophoneState`

`MicrophoneState` maps to package `audio` at `Microsoft/Xna/Framework/Audio/microphone_state.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Started` | `MicrophoneStateStarted` | 0 |
| `Stopped` | `MicrophoneStateStopped` | 1 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The Go zero value equals `MicrophoneStateStarted`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Audio.Microphone`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 14. `Microsoft.Xna.Framework.Graphics.FillMode`

`FillMode` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/fill_mode.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Solid` | `FillModeSolid` | 0 |
| `WireFrame` | `FillModeWireFrame` | 1 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The Go zero value equals `FillModeSolid`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.RasterizerState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 15. `Microsoft.Xna.Framework.Media.MediaSourceType`

`MediaSourceType` maps to package `media` at `Microsoft/Xna/Framework/Media/media_source_type.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `LocalDevice` | `MediaSourceTypeLocalDevice` | 0 |
| `WindowsMediaConnect` | `MediaSourceTypeWindowsMediaConnect` | 4 |

Source identities 3 (synthetic `value__` plus 2 named literals);
expected Go identities 2; target Go identities 2; local diagnostics 0.
The Go zero value equals `MediaSourceTypeLocalDevice`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Media.MediaSource`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 16. `Microsoft.Xna.Framework.Audio.SoundState`

`SoundState` maps to package `audio` at `Microsoft/Xna/Framework/Audio/sound_state.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Playing` | `SoundStatePlaying` | 0 |
| `Paused` | `SoundStatePaused` | 1 |
| `Stopped` | `SoundStateStopped` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `SoundStatePlaying`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Audio.SoundEffectInstance`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 17. `Microsoft.Xna.Framework.Graphics.CullMode`

`CullMode` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/cull_mode.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `None` | `CullModeNone` | 0 |
| `CullClockwiseFace` | `CullModeCullClockwiseFace` | 1 |
| `CullCounterClockwiseFace` | `CullModeCullCounterClockwiseFace` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `CullModeNone`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.RasterizerState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 18. `Microsoft.Xna.Framework.Graphics.GraphicsDeviceStatus`

`GraphicsDeviceStatus` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/graphics_device_status.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Normal` | `GraphicsDeviceStatusNormal` | 0 |
| `Lost` | `GraphicsDeviceStatusLost` | 1 |
| `NotReset` | `GraphicsDeviceStatusNotReset` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `GraphicsDeviceStatusNormal`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` *(protected partial runtime type; not modified)*

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 19. `Microsoft.Xna.Framework.Graphics.TextureAddressMode`

`TextureAddressMode` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/texture_address_mode.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Wrap` | `TextureAddressModeWrap` | 0 |
| `Clamp` | `TextureAddressModeClamp` | 1 |
| `Mirror` | `TextureAddressModeMirror` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `TextureAddressModeWrap`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.SamplerState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 20. `Microsoft.Xna.Framework.Input.GamePadDeadZone`

`GamePadDeadZone` maps to package `input` at `Microsoft/Xna/Framework/Input/game_pad_dead_zone.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `None` | `GamePadDeadZoneNone` | 0 |
| `IndependentAxes` | `GamePadDeadZoneIndependentAxes` | 1 |
| `Circular` | `GamePadDeadZoneCircular` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `GamePadDeadZoneNone`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Input.GamePad`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 21. `Microsoft.Xna.Framework.Media.VideoSoundtrackType`

`VideoSoundtrackType` maps to package `media` at `Microsoft/Xna/Framework/Media/video_soundtrack_type.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Music` | `VideoSoundtrackTypeMusic` | 0 |
| `Dialog` | `VideoSoundtrackTypeDialog` | 1 |
| `MusicAndDialog` | `VideoSoundtrackTypeMusicAndDialog` | 2 |

Source identities 4 (synthetic `value__` plus 3 named literals);
expected Go identities 3; target Go identities 3; local diagnostics 0.
The Go zero value equals `VideoSoundtrackTypeMusic`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Media.Video`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 22. `Microsoft.Xna.Framework.Graphics.PresentInterval`

`PresentInterval` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/present_interval.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Default` | `PresentIntervalDefault` | 0 |
| `One` | `PresentIntervalOne` | 1 |
| `Two` | `PresentIntervalTwo` | 2 |
| `Immediate` | `PresentIntervalImmediate` | 3 |

Source identities 5 (synthetic `value__` plus 4 named literals);
expected Go identities 4; target Go identities 4; local diagnostics 0.
The Go zero value equals `PresentIntervalDefault`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.PresentationParameters`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 23. `Microsoft.Xna.Framework.Graphics.PrimitiveType`

`PrimitiveType` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/primitive_type.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `TriangleList` | `PrimitiveTypeTriangleList` | 0 |
| `TriangleStrip` | `PrimitiveTypeTriangleStrip` | 1 |
| `LineList` | `PrimitiveTypeLineList` | 2 |
| `LineStrip` | `PrimitiveTypeLineStrip` | 3 |

Source identities 5 (synthetic `value__` plus 4 named literals);
expected Go identities 4; target Go identities 4; local diagnostics 0.
The Go zero value equals `PrimitiveTypeTriangleList`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.GraphicsDevice` *(protected partial runtime type; not modified)*

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 24. `Microsoft.Xna.Framework.Input.Touch.TouchLocationState`

`TouchLocationState` maps to package `touch` at `Microsoft/Xna/Framework/Input/Touch/touch_location_state.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Invalid` | `TouchLocationStateInvalid` | 0 |
| `Released` | `TouchLocationStateReleased` | 1 |
| `Pressed` | `TouchLocationStatePressed` | 2 |
| `Moved` | `TouchLocationStateMoved` | 3 |

Source identities 5 (synthetic `value__` plus 4 named literals);
expected Go identities 4; target Go identities 4; local diagnostics 0.
The Go zero value equals `TouchLocationStateInvalid`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Input.Touch.TouchLocation`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

### 25. `Microsoft.Xna.Framework.Graphics.BlendFunction`

`BlendFunction` maps to package `graphics` at `Microsoft/Xna/Framework/Graphics/blend_function.go` as a named `int32` type with no `// xna:flags` directive.

| CLR literal | Go constant | Raw `Int32` |
|---|---|---:|
| `Add` | `BlendFunctionAdd` | 0 |
| `Subtract` | `BlendFunctionSubtract` | 1 |
| `ReverseSubtract` | `BlendFunctionReverseSubtract` | 2 |
| `Min` | `BlendFunctionMin` | 3 |
| `Max` | `BlendFunctionMax` | 4 |

Source identities 6 (synthetic `value__` plus 5 named literals);
expected Go identities 5; target Go identities 5; local diagnostics 0.
The Go zero value equals `BlendFunctionAdd`.

Reverse fan-out 1 — deferred direct consumers:

- `Microsoft.Xna.Framework.Graphics.BlendState`

```text
DEPENDENT_IMPLEMENTATION_STARTED=false
```

## Incremental scoreboard ledger

The whole-profile scoreboard was regenerated with
`go run ./tools/api_compat --mode report` after **every** completed type, and
the public-signature dependency graph was regenerated and reranked immediately
afterwards. Every step moved the scoreboard by exactly the predicted amount.

| # | Completed type | Ids | TARGET_TYPES | TARGET_MEMBERS | TOTAL_DIAGNOSTICS | MISSING_TYPE | MISSING_MEMBER | COMPLETE_TYPES | PARTIAL_TYPES | Dependency-complete nodes |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 0 | *(Foundation 13 baseline)* | — | 65 | 1361 | 369 | 192 | 177 | 60 | 5 | 71 |
| 1 | `Microsoft.Xna.Framework.Graphics.RenderTargetUsage` | 3 | 66 | 1364 | 368 | 191 | 177 | 61 | 5 | 70 |
| 2 | `Microsoft.Xna.Framework.Graphics.CubeMapFace` | 6 | 67 | 1370 | 367 | 190 | 177 | 62 | 5 | 69 |
| 3 | `Microsoft.Xna.Framework.Audio.AudioChannels` | 2 | 68 | 1372 | 366 | 189 | 177 | 63 | 5 | 68 |
| 4 | `Microsoft.Xna.Framework.Audio.AudioStopOptions` | 2 | 69 | 1374 | 365 | 188 | 177 | 64 | 5 | 68 |
| 5 | `Microsoft.Xna.Framework.Graphics.IndexElementSize` | 2 | 70 | 1376 | 364 | 187 | 177 | 65 | 5 | 67 |
| 6 | `Microsoft.Xna.Framework.Graphics.SetDataOptions` | 3 | 71 | 1379 | 363 | 186 | 177 | 66 | 5 | 66 |
| 7 | `Microsoft.Xna.Framework.Media.MediaState` | 3 | 72 | 1382 | 362 | 185 | 177 | 67 | 5 | 65 |
| 8 | `Microsoft.Xna.Framework.Graphics.EffectParameterClass` | 5 | 73 | 1387 | 361 | 184 | 177 | 68 | 5 | 64 |
| 9 | `Microsoft.Xna.Framework.Graphics.CompareFunction` | 8 | 74 | 1395 | 360 | 183 | 177 | 69 | 5 | 63 |
| 10 | `Microsoft.Xna.Framework.Graphics.EffectParameterType` | 10 | 75 | 1405 | 359 | 182 | 177 | 70 | 5 | 63 |
| 11 | `Microsoft.Xna.Framework.Input.Touch.GestureType` | 11 | 76 | 1416 | 358 | 181 | 177 | 71 | 5 | 63 |
| 12 | `Microsoft.Xna.Framework.Input.Buttons` | 25 | 77 | 1441 | 357 | 180 | 177 | 72 | 5 | 63 |
| 13 | `Microsoft.Xna.Framework.Audio.MicrophoneState` | 2 | 78 | 1443 | 356 | 179 | 177 | 73 | 5 | 63 |
| 14 | `Microsoft.Xna.Framework.Graphics.FillMode` | 2 | 79 | 1445 | 355 | 178 | 177 | 74 | 5 | 62 |
| 15 | `Microsoft.Xna.Framework.Media.MediaSourceType` | 2 | 80 | 1447 | 354 | 177 | 177 | 75 | 5 | 62 |
| 16 | `Microsoft.Xna.Framework.Audio.SoundState` | 3 | 81 | 1450 | 353 | 176 | 177 | 76 | 5 | 61 |
| 17 | `Microsoft.Xna.Framework.Graphics.CullMode` | 3 | 82 | 1453 | 352 | 175 | 177 | 77 | 5 | 60 |
| 18 | `Microsoft.Xna.Framework.Graphics.GraphicsDeviceStatus` | 3 | 83 | 1456 | 351 | 174 | 177 | 78 | 5 | 59 |
| 19 | `Microsoft.Xna.Framework.Graphics.TextureAddressMode` | 3 | 84 | 1459 | 350 | 173 | 177 | 79 | 5 | 58 |
| 20 | `Microsoft.Xna.Framework.Input.GamePadDeadZone` | 3 | 85 | 1462 | 349 | 172 | 177 | 80 | 5 | 57 |
| 21 | `Microsoft.Xna.Framework.Media.VideoSoundtrackType` | 3 | 86 | 1465 | 348 | 171 | 177 | 81 | 5 | 57 |
| 22 | `Microsoft.Xna.Framework.Graphics.PresentInterval` | 4 | 87 | 1469 | 347 | 170 | 177 | 82 | 5 | 57 |
| 23 | `Microsoft.Xna.Framework.Graphics.PrimitiveType` | 4 | 88 | 1473 | 346 | 169 | 177 | 83 | 5 | 56 |
| 24 | `Microsoft.Xna.Framework.Input.Touch.TouchLocationState` | 4 | 89 | 1477 | 345 | 168 | 177 | 84 | 5 | 56 |
| 25 | `Microsoft.Xna.Framework.Graphics.BlendFunction` | 5 | 90 | 1482 | 344 | 167 | 177 | 85 | 5 | 55 |

`MISSING_MEMBER` stayed at exactly 177 and `PARTIAL_TYPES` at exactly 5 through
every one of the 25 steps. This batch removed `MISSING_TYPE` diagnostics only.

## Protected partial types

The five partial runtime types were off limits for this batch and are
unchanged:

| Partial type | Missing members |
|---|---:|
| `Microsoft.Xna.Framework.Game` | 39 |
| `Microsoft.Xna.Framework.GraphicsDeviceManager` | 40 |
| `Microsoft.Xna.Framework.Graphics.GraphicsDevice` | 70 |
| `Microsoft.Xna.Framework.Graphics.SpriteBatch` | 16 |
| `Microsoft.Xna.Framework.Graphics.Texture2D` | 12 |

`COMBINED_MISSING_MEMBER=177`, unchanged.

Three completed enums — `CubeMapFace`, `GraphicsDeviceStatus`, and
`PrimitiveType` — have `Microsoft.Xna.Framework.Graphics.GraphicsDevice` among
their deferred reverse consumers. No `GraphicsDevice` member was added. A
completed parameter or property type never authorizes its consumer.

## Structural and mutation evidence

The compiler-extracted local closure for all 25 types is measured by one new
table-driven report category, `foundation14EnumClosures`. Each entry records
the XNA identity, Go name, package path, source/expected/target identity
counts, local diagnostics, CLR kind, Go kind, underlying storage, expected and
actual flags markers, `value__` exclusion, and a per-literal expected/actual
raw-value row. All 25 report `"status": "PASS"` with 121 target Go identities
and zero local diagnostics.

One table-driven category replaces 25 near-identical bespoke closures. The
table `foundation14Enums` in `tools/api_compat/verify.go` is authoritative and
is itself admitted against the pinned contract by
`TestFoundation14EnumMappedContracts`, which checks kind, flags, underlying
storage, base type, absence of direct interfaces, exactly one `value__` storage
field, the exact literal name set in both directions, the exact raw values, the
derived package path, the derived Go type name, the const projection of every
literal, the absence of any projected `value__` identity, and the total of 121
mapped identities.

The local strict-zero matrix holds for every completed type:

```text
MISSING_MEMBER=0
TYPE_KIND_MISMATCH=0
BASE_MAPPING_MISMATCH=0
INTERFACE_MAPPING_MISMATCH=0
FIELD_MAPPING_MISMATCH=0
PROPERTY_MAPPING_MISMATCH=0
METHOD_SIGNATURE_MAPPING_MISMATCH=0
PARAMETER_MAPPING_MISMATCH=0
RETURN_MAPPING_MISMATCH=0
ERROR_MAPPING_MISMATCH=0
OVERLOAD_MAPPING_MISMATCH=0
GENERIC_MAPPING_MISMATCH=0
ENUM_VALUE_MISMATCH=0
FLAGS_MAPPING_MISMATCH=0
EVENT_MAPPING_MISMATCH=0
OPERATOR_MAPPING_MISMATCH=0
REF_OUT_MAPPING_MISMATCH=0
LANGUAGE_MAPPING_MISMATCH=0
```

### Negative verifier fixtures

Fourteen structural defects are defined in `foundation14EnumDefects`:

| Defect | Raised category | Applies to |
|---|---|---|
| `missing_type` | `MISSING_TYPE` | every type |
| `wrong_package` | `MISSING_TYPE` | every type |
| `wrong_kind` | `TYPE_KIND_MISMATCH` | every type |
| `wrong_underlying_type` | `TYPE_KIND_MISMATCH` | every type |
| `accidentally_flags` | `FLAGS_MAPPING_MISMATCH` | ordinary enums |
| `flags_directive_dropped` | `FLAGS_MAPPING_MISMATCH` | flags enums |
| `wrong_first_value` | `ENUM_VALUE_MISMATCH` | every type |
| `wrong_last_value` | `ENUM_VALUE_MISMATCH` | every type |
| `iota_renumbering` | `ENUM_VALUE_MISMATCH` | every type whose pinned values are not already `0,1,2,…` |
| `missing_last_literal` | `MISSING_MEMBER` | every type |
| `renamed_literal` | `MISSING_MEMBER` | every type |
| `value_storage_projected` | `UNEXPECTED_MEMBER` | every type |
| `invented_constant` | `UNEXPECTED_MEMBER` | every type |
| `exported_helper` | `UNEXPECTED_MEMBER` | every type |

`TestFoundation14EnumDefectsRejectedForEveryType` applies every applicable
defect to every one of the 25 enums — **304 negative cases**. Each case first
asserts a clean unmutated baseline, so no defect can pass by accident, then
asserts both that the defect raises its category and that the type's closure
measurement drops to `FAIL`.

The `iota_renumbering` defect is the direct negative fixture for the enum
policy's `iota` prohibition: an `iota` block renumbers literals `0,1,2,…` in
source order, which is exactly the drift it produces. It is skipped only for
enums whose pinned values already are `0,1,2,…`, where the renumbering is
provably invisible.

The declared fixture inventory `tools/api_compat/testdata/mutations.json` grows
from 157 to **177 cases**; the 20 new entries bind each defect to a
representative batch enum, including a flags enum (`Buttons`, `SetDataOptions`),
an enum with no zero literal (`AudioChannels`), and a sparse-valued enum
(`MediaSourceType`).

Each owning package additionally carries a source-level self-test. `flagsDirectiveAt`
parses the real declaration site with `go/parser` and asserts the `xna:flags`
directive is present exactly for the flags enums and absent for the ordinary
ones — measuring the actual source, not an extractor model field.
`TestFlagsDirectiveDetector` is the negative fixture for the detector itself,
rejecting `// xna:flags=false`, `// not-xna:flags`, `//xna:flags`, and a comment
that merely mentions the directive.

Manual allowlists and unmeasured structural categories remain zero.

## Behavior provenance

The behavior corpus grows from 280 to **411 observations / 411 assertions with
zero failures**. Each completed type contributes a dedicated group named after
the type (for example `RENDER_TARGET_USAGE`, `BUTTONS`, `MEDIA_STATE`).

`PURE_XNA_DERIVED` observations, per type:

- `<type>.complete-raw-table` — the complete pinned literal-to-raw-value table;
- `<type>.underlying-system-int32` — `System.Int32` storage measured through
  `reflect.Kind`.

For the three flags enums (`SetDataOptions`, `GestureType`, `Buttons`) two more
`PURE_XNA_DERIVED` observations record the pinned bit structure:

- `<type>.flags-union` — the exact union of every literal
  (`SetDataOptions`=3, `GestureType`=1023, `Buttons`=2145451007);
- `<type>.flags-disjoint-single-bits` — every non-zero literal is a distinct
  single bit.

`GO_LANGUAGE_PROJECTION` observations, per type:

- `<type>.zero-value-<literal>` — the Go zero value equals the named zero
  literal; or `<type>.zero-value-unnamed` for `AudioChannels` and `Buttons`,
  whose pinned contracts declare no zero literal, recording that the Go zero
  equals no named constant;
- `<type>.arbitrary-positive-raw` — an undefined positive raw value and 12345
  remain representable;
- `<type>.negative-raw` — `-1` remains representable.

The two provenance categories are never conflated. Go's permissive named-`int32`
raw domain, its zero-value rule, and its copy semantics are labeled
`GO_LANGUAGE_PROJECTION` and are never presented as XNA runtime behavior. Bit
unions for the flags enums are pinned metadata arithmetic, not a claim that any
gesture, buffer write, or game pad button is observable.

The focused package tests and the behavior corpus pass with
`CNA_NATIVE_LIBRARY` unset and instantiate no Game, GraphicsDevice, audio
engine, media player, touch panel, game pad, native loader, or cgo route. This
is pure managed metadata.

## Runtime and ABI boundary — no capability inflation

Completing these enums proves managed metadata only. Explicitly:

- `RenderTargetUsage` does **not** imply render targets, discard-contents, or
  preserve-contents support.
- `CubeMapFace` does **not** imply cube maps or cube render targets.
- `PrimitiveType` does **not** imply `DrawPrimitives`.
- `Blend*`, `CompareFunction`, `CullMode`, `FillMode`, `StencilOperation`,
  `TextureFilter`, and `TextureAddressMode` peers do **not** imply render
  state, rasterization, or sampling.
- `AudioChannels`, `AudioStopOptions`, `MicrophoneState`, and `SoundState` do
  **not** imply any audio engine, microphone, or sound playback.
- `MediaState`, `MediaSourceType`, and `VideoSoundtrackType` do **not** imply
  any media or video playback.
- `GestureType` and `TouchLocationState` do **not** imply a touch panel.
- `Buttons` and `GamePadDeadZone` do **not** imply game pad polling, connection
  state, dead-zone handling, or vibration.
- `EffectParameterClass` and `EffectParameterType` do **not** imply effects,
  shaders, or 3D.
- `GraphicsDeviceStatus` does **not** imply device loss or reset handling.
- `IndexElementSize` and `SetDataOptions` do **not** imply index or vertex
  buffers.
- `PresentInterval` does **not** imply presentation or vsync control.

`go run ./tools/capabilities --check` reports `RUNTIME_CAPABILITIES=38
STATUS=PASS`, unchanged, and `docs/runtime-capabilities.json` is unmodified. No
capability claim was added.

No native mirror constant, C function, SDL mapping, native enum conversion,
manifest entry, layout, or callback was added, and no Foundation-14 type
crosses the C ABI. No CNA source was changed and no cgo route was added.

The canonical ABI remains exactly **23** bound functions, **67** prototype type
positions, **96** C/Go measurements, **28** layouts, **2** callbacks, and **5**
constants, with zero missing header symbols, zero missing library symbols, and
zero mismatches.

## New namespaces

Three XNA namespaces gained their first Go package in this batch:

| Namespace | Package path | Go package | Types |
|---|---|---|---:|
| `Microsoft.Xna.Framework.Audio` | `Microsoft/Xna/Framework/Audio` | `audio` | 4 |
| `Microsoft.Xna.Framework.Media` | `Microsoft/Xna/Framework/Media` | `media` | 3 |
| `Microsoft.Xna.Framework.Input.Touch` | `Microsoft/Xna/Framework/Input/Touch` | `touch` | 2 |

Each package path is the deterministic result of the established
`packagePathForNamespace` rule (namespace dots become path separators); no new
namespace policy was created. The nested `Input/Touch` package follows the
precedent already set by `Graphics/PackedVector`, and each new package carries
only a `doc.go` and its enum files. Creating these packages is a namespace fact
and carries no audio, media, or touch capability claim.


## Skipped candidates

Every candidate below outranked or tied part of the consumed batch and was
materially considered, inspected against the pinned contract, and skipped. In
each case the loop recorded the reason and continued scanning lower-ranked safe
candidates rather than stopping.

| Candidate | Fan-out | Category | Exact reason |
|---|---:|---|---|
| `Microsoft.Xna.Framework.Design.MathTypeConverter` | 12 | `SKIPPED_FOR_BATCH` — unmapped BCL dependency | Derives from `System.ComponentModel.ExpandableObjectConverter` and returns `PropertyDescriptorCollection`; `ITypeDescriptorContext` is also unmapped. Rule 13: would require a new BCL projection subsystem. |
| `Microsoft.Xna.Framework.Graphics.GraphicsResource` | 11 | `SKIPPED_FOR_BATCH` — ownership/lifetime architecture | `IDisposable` runtime base with `Dispose`, `Dispose(bool)`, `Finalize`, an `IsDisposed` property, a `Disposing` event, and a `GraphicsDevice` property. Rules 6, 8, and 9, and hard-skip `GraphicsResource lifetime/disposal architecture`. Also requires the protected partial `GraphicsDevice` (stop condition F). |
| `Microsoft.Xna.Framework.Graphics.IEffectMatrices` | 5 | `SKIPPED_FOR_BATCH` — new general mapping policy | Pure managed `Matrix` properties and dependency-complete, but projecting a settable-property interface would require a new general managed-interface mapping decision (error results, witness policy) that the enum-only batch must not invent. Belongs to the deferred Effects/3D family and has no implementor in scope. |
| `Microsoft.Xna.Framework.Graphics.IEffectFog` | 5 | `SKIPPED_FOR_BATCH` — new general mapping policy | Same as `IEffectMatrices`; `Boolean`, `Single`, and `Vector3` settable properties, deferred Effects/3D family, no implementor in scope. |
| `Microsoft.Xna.Framework.IGameComponent` | 3 | `SKIPPED_FOR_BATCH` — Game runtime lifecycle | A single `Initialize()` lifecycle method on the Game runtime, not a pure managed public-data leaf. Rules 5 and 8. |
| `Microsoft.Xna.Framework.Audio.AudioListener` | 3 | `SKIPPED_FOR_BATCH` — insufficient authoritative semantics | Pure managed `Vector3` property bag with a default constructor, but the pinned contract carries no constructor default state. `Forward`, `Up`, `Position`, and `Velocity` defaults are not established by retained authoritative XNA evidence. Rule 15, and rule 16 — guessing them would be fake behavior. |
| `Microsoft.Xna.Framework.Audio.AudioEmitter` | 3 | `SKIPPED_FOR_BATCH` — insufficient authoritative semantics | Same as `AudioListener`, plus a `DopplerScale` property whose default value and argument-validation semantics are not established by the pinned contract. Rules 15 and 16. |
| `Microsoft.Xna.Framework.Graphics.DisplayMode` | 3 | `SKIPPED_FOR_BATCH` — fake adapter capability | Read-only `Format`, `Width`, `Height`, `AspectRatio`, and `TitleSafeArea` with no public constructor; it can only be produced by a real adapter. Hard-skip `fake display/adapter capability`. |
| `Microsoft.Xna.Framework.Content.ContentManager` | 3 | `SKIPPED_FOR_BATCH` — deferred family and unmapped BCL | Needs `System.IServiceProvider`, `System.Action\`1`, and `IDisposable`, and belongs to the deferred Content/XNB family. Rules 12 and 13. |
| `Microsoft.Xna.Framework.Graphics.ColorWriteChannels` | 1 | `NOT_REACHED` — batch limit | A safe pure-managed flags enum that remains dependency-complete. Not skipped for cause; the 25-type ceiling (stop condition A) was reached first. |
| `Microsoft.Xna.Framework.Graphics.StencilOperation` | 1 | `NOT_REACHED` — batch limit | Safe pure-managed ordinary enum; ceiling reached. |
| `Microsoft.Xna.Framework.Graphics.TextureFilter` | 1 | `NOT_REACHED` — batch limit | Safe pure-managed ordinary enum; ceiling reached. |
| `Microsoft.Xna.Framework.Input.GamePadType` | 1 | `NOT_REACHED` — batch limit | Safe pure-managed ordinary enum; ceiling reached. |
| `Microsoft.Xna.Framework.Graphics.Blend` | 1 | `NOT_REACHED` — batch limit | Safe pure-managed ordinary enum; ceiling reached. |

No candidate was skipped because it was merely inconvenient, and no mapping
policy was invented to make a candidate fit.

## Checkpoints

| Point | Gates run | Result |
|---|---|---|
| After each of the 25 completed types | `gofmt`, the type's focused package tests, the package's shared flags-directive detector test, full `api_compat --mode report`, dependency-graph regeneration and rerank | 25/25 `PASS`, zero gofmt findings, scoreboard delta exactly as predicted every time |
| After the verifier measurement landed | `go build ./tools/...`, `TestFoundation14EnumMappedContracts` | `PASS`; 25/25 closures `PASS`, 121 target identities |
| After the negative fixtures landed | `TestFoundation14EnumDefectsRejectedForEveryType` (304 cases), `TestMutationFixtures` (177 cases) | `PASS` |
| After the behavior corpus landed | `go run ./tools/behavior` | 411 observations / 411 assertions / 0 failures |
| Batch-complete full checkpoint | `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test ./...`, `go test -race ./...`, `api_compat --mode leak-only`, `packed_vector_qualify`, `capabilities --check` | all `PASS`; leak, allowlist, and unmeasured counters zero; 262,400 PackedVector iterations with zero failures; 38 runtime capabilities unchanged |
| Native regression (final only) | `native_abi`, `go run -race ./tools/native_stress --race-status PASS` | ABI 23/67/96/28/2/5 with zero missing symbols and zero mismatches; stress reproduces every counter exactly with zero crashes, UAF, or double-free |

No checkpoint failed, so the loop never had to stop to diagnose a regression.

Because every type in this batch is a leaf enum with an empty public-signature
dependency set, the per-type focused gate was made stronger than the milestone
minimum: the **full** structural verifier was regenerated after every single
type rather than only every fifth. The expensive suites — `go test -race ./...`,
the exhaustive PackedVector sweep, the native ABI check, and the native stress
suite — were correctly deferred to the batch boundary.

## Dependency-graph methodology

Dependency completeness counts a node's base type, its declared direct
interfaces, and every member signature type as public-signature dependencies.
BCL types covered by the mapping table are satisfied; the five partial runtime
types count as present.

Regenerating the Foundation-13 baseline under this methodology reproduces
exactly **71** dependency-complete missing nodes, matching the published
Foundation-13 handoff. The historical 71/72 nuance was not reopened:
`Microsoft.Xna.Framework.GameComponent` remains dependency-*incomplete* because
its declared direct interfaces `IGameComponent` and `IUpdateable` are still
missing, and the interface-inclusive rule stays authoritative.

The count moved 71 → 55 across the batch. It falls rather than rises because
each completed type leaves the missing set while unlocking comparatively few
new nodes. The batch did unlock genuinely new dependency-complete nodes,
including `Microsoft.Xna.Framework.Input.GamePadButtons` (unlocked by
`Buttons`), `Microsoft.Xna.Framework.Input.Touch.TouchLocation` and
`GestureSample` (unlocked by `TouchLocationState` and `GestureType`),
`Microsoft.Xna.Framework.Graphics.PresentationParameters` (unlocked by
`RenderTargetUsage` and `PresentInterval`), `Microsoft.Xna.Framework.Media.MediaSource`
and `Video` (unlocked by `MediaSourceType` and `VideoSoundtrackType`), and
`Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService`.

## Remaining safe candidates

Five pure-managed enums remain dependency-complete and safe under the same
rules, all at reverse fan-out 1: `ColorWriteChannels` (flags, 6 literals),
`StencilOperation` (8), `TextureFilter` (9), `GamePadType` (10), and `Blend`
(13) — 46 further identities. Eight value structs are also dependency-complete
with no unmapped BCL dependency: `TouchPanelCapabilities`, `RendererDetail`,
`GestureSample`, `GamePadThumbSticks`, `GamePadTriggers`, `GamePadDPad`,
`MouseState`, and `GamePadButtons`; each would need its own constructor,
equality, hash, and operator projection and was not assessed for this
enum-only batch.

## Foundation-13 regression checks

| Type | Required state | Measured |
|---|---|---|
| `ButtonState` | `Released=0`, `Pressed=1`, `flags=false`, named `int32`, arbitrary raws representable, no `iota`, no `xna:flags`, no helper API | unchanged, `PASS` |
| `PrimitiveType` | `TriangleList=0`, `TriangleStrip=1`, `LineList=2`, `LineStrip=3` | newly completed in this batch with exactly these values, `PASS` |
| `GraphicsProfile`, `DepthFormat`, `SurfaceFormat`, `ClearOptions`, `BufferUsage`, `DisplayOrientation`, `PlayerIndex`, `Keyboard`, `VertexElement` | complete and unchanged | `PASS` |
| PackedVector exhaustive sweep | 262,400 iterations, 0 failures | `PASS` |
| Interface witnesses | 25 total, 17 `PackFromVector4`, 8 `ToVector4` | unchanged, `PASS` |

`PrimitiveType` was listed in the Foundation-14 brief among the regression
types, but it was in fact a *missing* type at the batch baseline. It was
selected on its own merits by the ranking rule and completed as batch entry 23;
its raw table matches the brief's stated values exactly.

## Final structural scoreboard

| Counter | Foundation 13 | Foundation 14 | Delta |
|---|---:|---:|---:|
| `REFERENCE_TYPES` | 257 | 257 | 0 |
| `REFERENCE_MEMBERS` | 2964 | 2964 | 0 |
| `EXPECTED_GO_TYPES` | 257 | 257 | 0 |
| `EXPECTED_GO_MEMBERS` | 3243 | 3243 | 0 |
| `TARGET_TYPES` | 65 | 90 | +25 |
| `TARGET_MEMBERS` | 1361 | 1482 | +121 |
| `TOTAL_DIAGNOSTICS` | 369 | 344 | −25 |
| `MISSING_TYPE` | 192 | 167 | −25 |
| `MISSING_MEMBER` | 177 | 177 | 0 |
| `COMPLETE_TYPES` | 60 | 85 | +25 |
| `PARTIAL_TYPES` | 5 | 5 | 0 |
| `MISSING_TYPES` | 192 | 167 | −25 |

Preservation, mismatch, and safety counters, all zero before and after:

```text
UNEXPECTED_TYPE=0
UNEXPECTED_MEMBER=0
TYPE_KIND_MISMATCH=0
BASE_MAPPING_MISMATCH=0
INTERFACE_MAPPING_MISMATCH=0
FIELD_MAPPING_MISMATCH=0
PROPERTY_MAPPING_MISMATCH=0
METHOD_SIGNATURE_MAPPING_MISMATCH=0
PARAMETER_MAPPING_MISMATCH=0
RETURN_MAPPING_MISMATCH=0
ERROR_MAPPING_MISMATCH=0
OVERLOAD_MAPPING_MISMATCH=0
GENERIC_MAPPING_MISMATCH=0
ENUM_VALUE_MISMATCH=0
FLAGS_MAPPING_MISMATCH=0
EVENT_MAPPING_MISMATCH=0
OPERATOR_MAPPING_MISMATCH=0
REF_OUT_MAPPING_MISMATCH=0
LANGUAGE_MAPPING_MISMATCH=0
INTERNAL_TYPE_LEAK=0
RAW_HANDLE_LEAK=0
PUBLIC_NATIVE_FFI_LEAK=0
ALLOWLIST_ENTRIES=0
UNMEASURED_STRUCTURAL_CATEGORY=0
```

Witness counters, unchanged:

```text
INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Normal strict mode remains expected-red at 344 diagnostics, which is the
deferred XNA work queue and not a compatibility claim. Leak-only mode passes.

## Re-running the Foundation-14 gates

Go 1.24.4, Linux amd64, cgo, GCC, and an exact ABI-0.7 CNA library:

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict   # expected 344 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -library "$CNA_NATIVE_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
git diff --check
```

Strict mode rewrites `docs/generated/api-compat-report.json` with
`"mode": "strict"`; rerun `--mode report` last so the committed evidence keeps
its report mode. Verify the native ABI and stress counters out of tree with an
explicit `-output` under the scratchpad so a locally rebuilt ABI-0.7 library or
an absolute developer header path never rewrites the committed evidence.
