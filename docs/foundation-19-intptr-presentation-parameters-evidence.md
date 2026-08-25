# Foundation 19 — System.IntPtr projection and PresentationParameters

Foundation 19 formalizes the `System.IntPtr` language projection, narrows the
`RAW_HANDLE_LEAK` gate by exactly that much and no further, and uses both to
close `Microsoft.Xna.Framework.Graphics.PresentationParameters`.

## Reference authority

```text
Microsoft.Xna.Framework.Graphics.dll
sha256 560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55
```

Read with `ikdasm`. Public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## `System.IntPtr` → `uintptr`

The projection is general and stated once. `uintptr` is the **opaque
pointer-sized bit value** the public XNA contract carries at that position, and
nothing more. It does **not** mean:

- that the value may be dereferenced,
- that it is a CNA handle,
- that it is an SDL pointer,
- that a window exists,
- that `unsafe.Pointer` is admissible anywhere,
- that native device creation is authorized.

`IntPtr.Zero` is `uintptr(0)`. CLR `IntPtr` is signed, but for this binding the
public values are opaque handles and bit patterns, so no signed numerical
ordering is claimed for them.

Six CLR members in the pinned profile declare `System.IntPtr`, and the test
enumerates all six rather than only the one this milestone implements:

| CLR member                                          | shape                | projected positions |
| --------------------------------------------------- | -------------------- | ------------------- |
| `GameWindow.Handle`                                  | read-only property   | 1 result            |
| `GraphicsAdapter.MonitorHandle`                      | read-only property   | 1 result            |
| `GraphicsDevice.Present(…, …, overrideWindowHandle)` | method parameter     | 1 parameter         |
| `PresentationParameters.DeviceWindowHandle`          | read/write property  | 1 result + 1 param  |
| `Mouse.WindowHandle`                                 | static read/write    | 1 result + 1 param  |
| `TouchPanel.WindowHandle`                            | static read/write    | 1 result + 1 param  |

## The narrowed `RAW_HANDLE_LEAK` rule

A public Go `uintptr` is an allowed language projection **only** at a signature
position where the authoritative XNA metadata declares `System.IntPtr`. Because
`System.IntPtr` is the only source type that maps to `uintptr`, the expected
surface already carries `uintptr` at precisely the admitted positions, so
positional agreement with the expected surface *is* the whole test — there is no
separate allowlist to drift.

The gate was narrowed by that much and no further. It still catches, and each
of these is a fixture:

| route                                              | still a leak |
| --------------------------------------------------- | ------------ |
| `uintptr` result where the source declares `int32`  | yes          |
| `uintptr` parameter where the source declares `int32` | yes        |
| `uintptr` drifted from parameter to result          | yes          |
| `uintptr` drifted to an unadmitted parameter index  | yes          |
| `[]uintptr` at an unadmitted position               | yes          |
| `*uintptr` at an unadmitted position                | yes          |
| `uintptr` on a member the profile never declares    | yes          |
| exported named type defined over `uintptr`          | yes          |
| member name containing `NativeHandle` / `RawHandle` | yes          |
| member or type name prefixed `Cna`                  | yes          |
| public `unsafe.Pointer`                             | **no** — it is a `PUBLIC_NATIVE_FFI_LEAK`, and the test asserts the two categories stay distinct |

Matching uses a whole-word `\buintptr\b`, so composite spellings are caught
while an identifier such as `uintptrish` is not.

Eleven fixtures run: two that must stay clean and ten that must leak, each
asserting the exact verdict so neither direction can regress silently.

## `PresentationParameters` — exact contract

CLR: `public class PresentationParameters : System.Object`, no interfaces,
13 source members, 23 projected Go identities, **no error result anywhere**.

Storage is one assembly-visible nested `Settings` value struct of ten public
fields. Every public accessor is one `ldflda` into it plus one `ldfld` or
`stfld`. **A descriptor is not a device**: nothing here creates, resets, or
presents a `GraphicsDevice`, enumerates a display, looks up a window, or
reaches SDL or CNA.

```go
func NewPresentationParameters() *PresentationParameters
func (p *PresentationParameters) Clone() *PresentationParameters

func (p *PresentationParameters) BackBufferWidth() int32
func (p *PresentationParameters) SetBackBufferWidth(value int32)
func (p *PresentationParameters) BackBufferHeight() int32
func (p *PresentationParameters) SetBackBufferHeight(value int32)
func (p *PresentationParameters) BackBufferFormat() SurfaceFormat
func (p *PresentationParameters) SetBackBufferFormat(value SurfaceFormat)
func (p *PresentationParameters) DepthStencilFormat() DepthFormat
func (p *PresentationParameters) SetDepthStencilFormat(value DepthFormat)
func (p *PresentationParameters) MultiSampleCount() int32
func (p *PresentationParameters) SetMultiSampleCount(value int32)
func (p *PresentationParameters) DisplayOrientation() framework.DisplayOrientation
func (p *PresentationParameters) SetDisplayOrientation(value framework.DisplayOrientation)
func (p *PresentationParameters) PresentationInterval() PresentInterval
func (p *PresentationParameters) SetPresentationInterval(value PresentInterval)
func (p *PresentationParameters) RenderTargetUsage() RenderTargetUsage
func (p *PresentationParameters) SetRenderTargetUsage(value RenderTargetUsage)
func (p *PresentationParameters) DeviceWindowHandle() uintptr
func (p *PresentationParameters) SetDeviceWindowHandle(value uintptr)
func (p *PresentationParameters) IsFullScreen() bool
func (p *PresentationParameters) SetIsFullScreen(value bool)
func (p *PresentationParameters) Bounds() framework.Rectangle
```

### Constructor defaults — one quirk

The constructor's entire body after the base call is:

```text
IL_0006: ldarg.0
IL_0007: ldc.i4.1
IL_0008: call PresentationParameters::set_IsFullScreen(bool)
```

So **`IsFullScreen` defaults to `true`** and every other member takes its CLR
zero:

| member                 | default                              |
| ---------------------- | ------------------------------------ |
| `BackBufferWidth`      | `0`                                  |
| `BackBufferHeight`     | `0`                                  |
| `BackBufferFormat`     | `SurfaceFormat.Color` (0)            |
| `DepthStencilFormat`   | `DepthFormat.None` (0)               |
| `MultiSampleCount`     | `0`                                  |
| `DisplayOrientation`   | `DisplayOrientation.Default` (0)     |
| `PresentationInterval` | `PresentInterval.Default` (0)        |
| `RenderTargetUsage`    | `RenderTargetUsage.DiscardContents` (0) |
| `DeviceWindowHandle`   | `IntPtr.Zero` → `uintptr(0)`         |
| `IsFullScreen`         | **`true`**                           |
| `Bounds`               | `Rectangle(0, 0, 0, 0)` (computed)   |

A full-screen default on a zero-sized back buffer reads oddly, and in practice
`GraphicsDeviceManager` overwrites it before any device is created. It is
nonetheless what the reference constructor does, and it is preserved.

### No validation anywhere

Every setter is a bare `stfld`. Negative extents, a negative
`MultiSampleCount`, and undefined raw enum values all round-trip unchanged, and
no member checks the value against any device capability. That is the whole
point of a descriptor.

### `Bounds` is computed, not stored

```text
get_Bounds: ldc.i4.0; ldc.i4.0; <BackBufferWidth>; <BackBufferHeight>;
            newobj Rectangle::.ctor(int32, int32, int32, int32)
```

It always originates at `(0, 0)`, tracks the two extents live, and propagates a
degenerate extent rather than clamping it.

### `Clone`

```text
Clone: newobj PresentationParameters::.ctor()
       <clone>.settings = <this>.settings      // one stfld of a value struct
       return clone
```

The single value-struct assignment copies every member, so the clone is fully
independent — including `IsFullScreen`, whose constructor-assigned `true` is
overwritten by the source's value. `Clone` is pure managed and infallible.

The descriptor itself keeps CLR reference semantics: two variables naming one
instance observe the same mutations. `Clone` is the only thing that copies.

### `IsFullScreen` storage

The reference stores an `int32` whose setter normalizes any true value to 1 and
whose getter reports `field != 0`. A Go `bool` is exactly that two-valued
normalization, so no `int32` mirror is needed to reproduce the observable
behavior; the round trip is asserted in both directions.

### `DeviceWindowHandle` and the runtime boundary

Both accessors are a bare field read and write with no validation, so any bit
pattern round-trips exactly — `0`, `1`, `^uintptr(0)`, and arbitrary values are
all asserted.

Implementing this property **does not authorize** `GraphicsDevice` creation,
`GraphicsDevice.Reset`, a `GraphicsAdapter` implementation, display
enumeration, native window lookup, SDL calls, renderer presentation, or any
actual swap-chain behavior. A descriptor may store a platform handle without the
binding knowing what to do with it.

## What XNA 4.0 does not have

`PresentationParameters` has **no `Clear` method** in the XNA 4.0 Windows
profile. The IL declares exactly one constructor, `Clone`, ten read/write
properties, and the read-only `Bounds`. No `Clear`, no `Equals` override, no
`ToString` override, and no operators were invented to fill it out, and none of
MonoGame's or FNA's additional members or different defaults were copied.

## Verifier coverage

`PresentationParameters` joins the shared `managedClassClosure` category as the
Foundation-19 closure, so the whole 13-defect matrix from Foundation 17 applies
to it unchanged:

```text
PresentationParameters  PASS  13 source -> 23 Go identities, 10 accessor pairs,
                              0 errors, reference projection *PresentationParameters
```

Two shared defects were made owner-relative while adding it, because the
type lives in the package the fixtures previously relocated *to*:
`wrong_package` now picks a destination that is never the owner's own package,
and `wrong_setter_parameter` uses `complex128`, which no XNA member maps to, so
it can never coincide with a correct parameter type.

The mutation inventory grows from 290 to 314: 13 managed-class defects on
`PresentationParameters` plus the 11 raw-handle fixtures.

## Behavior corpus

Seven new observations in group `PRESENTATION_PARAMETERS`, taking the corpus
from 541 to 548 with zero failures: the ten constructed defaults including the
`true` full-screen quirk, the degenerate default `Bounds`, `Bounds` tracking a
live extent change, the no-validation round trip, the four-value
`DeviceWindowHandle` bit round trip, `Clone` independence with the overwritten
`IsFullScreen`, and reference-semantics aliasing.

## Structural effect

```text                       before   after
TARGET_TYPES                    109     110
TARGET_MEMBERS                 1657    1680
TOTAL_DIAGNOSTICS               325     324
MISSING_TYPE                    148     147
MISSING_MEMBER                  177     177
COMPLETE_TYPES                  104     105
PARTIAL_TYPES                     5       5
INTERNAL_TYPE_LEAK                0       0
RAW_HANDLE_LEAK                   0       0
PUBLIC_NATIVE_FFI_LEAK            0       0
```

## No capability claim and no ABI change

Nothing in this milestone reaches native code. The CNA-Go ABI is unchanged at
`23 / 67 / 96 / 28 / 2 / 5` with no missing symbols and no mismatches. CNA was
not rebuilt.
