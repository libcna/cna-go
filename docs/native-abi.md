# Native ABI boundary

CNA-Go binds only CNA's canonical C ABI. It never links or reflects CNA's C++
ABI and does not route through another language binding.

## Admission and loading

Foundation 1 admits exactly encoded ABI version `0x00000700` (0.7.0). A
different version, a missing required symbol, or a loader failure rejects the
library before Game creation. There is no same-major, newer-minor, or generic
0.x compatibility policy.

On qualified Linux builds, `internal/interop/bridge.c` uses `dlopen` with
`RTLD_NOW|RTLD_LOCAL` and resolves one reviewed manifest. An explicit runtime
artifact is selected with:

```text
CNA_NATIVE_LIBRARY=/absolute/path/to/libcna_c_api.so
```

The value must be an absolute regular-file path. With no override the platform
loader searches for `libcna_c_api.so`; CNA-Go has no repository-relative or
developer-machine fallback. A native library is not embedded in the Go module.

Foundation 1 requires Linux, cgo, and a C compiler. Pure-Go/no-cgo use and cgo
cross-compilation are not claimed. Apple `dlopen` and Windows
`LoadLibrary/GetProcAddress` implementations remain platform work.

## Qualified Foundation 1 artifact

No retained sibling artifact had the required 0.7 ABI: inspected candidates
reported either 0.1 or 0.8. The qualification library was therefore built in
an isolated temporary tree from CNA revision
`a09196a6477f69a7a57c8364f990658d31531a5b`, the independently selected source
revision whose canonical header declares ABI 0.7. CNA's checkout was not
modified. The build used the exact submodule revisions recorded by that tree,
sharp-runtime revision `54578590b328aa9612fe38bfddca9fd8ca795144`, GCC
14.2.0, Release mode, HEADLESS rendering, NULL audio, and networking enabled as
required by that C API source. A warning suppression for a later
sharp-runtime/CNA-0.7 overloaded-virtual incompatibility was compiler-only; no
CNA source was patched.

The admitted `libcna_c_api.so` is 16,799,760 bytes with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`.
The loader reports ABI 0.7.0 and the compiler/loader verifier reports 23 bound
functions, 67 prototype type positions, 96 aggregate C/Go measurements, 28
layout measurements, two callback shapes, five constants, zero missing header
symbols, zero missing library symbols, and zero ABI mismatches.

This artifact is qualification input, not a distributed CNA-Go file. It is not
sanitizer-instrumented, so `NATIVE_SANITIZER_STATUS=NOT_RUN`; stress results do
not constitute native leak-freedom evidence. Its HEADLESS renderer proves
native graphics execution but not visible output, and NULL audio cannot qualify
audio behavior.

## Typed manifest

`internal/interop/abi_manifest.h` records each bound symbol as a typed C
function pointer. `bridge.c` resolves those entries once and exposes narrow,
typed bridge functions to cgo. Go never casts or invokes a raw function pointer.
The current closure is:

- ABI/error: version and last-error copy routes;
- Game: create, frame hooks, run, request-exit, destroy;
- device manager/device: create, borrow device, viewport, clear;
- texture: decode from encoded memory, info, destroy;
- SpriteBatch: create, begin, scaled submission, end, destroy;
- keyboard: state snapshot.

For every entry the manifest fixes return type, parameter order, pointer depth,
fixed-width representation, and callback signature. Image bytes and command
arrays are caller-owned for the duration of the call and are synchronously
copied/consumed. Native strings are length-delimited `CNA_StringView` values;
last-error text is copied into Go-owned memory.

## Independent verification

`go run ./tools/native_abi -library /absolute/path/to/libcna_c_api.so` performs
three independent checks:

1. GCC compiles pointer assignments from every private manifest typedef to the
   declaration in canonical CNA headers with incompatible-pointer warnings as
   errors.
2. A canonical-header probe measures sizes, alignments, offsets, callback
   types, ABI/result constants, and fixed-width representations.
3. The production loader admits the selected library and proves every manifest
   export is present.

The generated result is `docs/generated/native-abi-report.json`. The canonical
headers verify the bridge; CNA-Go's private declarations do not verify
themselves.

## Go/cgo safety and callbacks

The goroutine entering `Game.Run` calls `runtime.LockOSThread` before loading,
creation, callbacks, resource cleanup, and destruction. It unlocks only after
native callback registration is dead and the Game is destroyed. Native
thread-affine operations compare the current native thread ID with this owner.
On a wrong-thread destroy, the resource handle is preserved so the owner thread
can retry.

Persistent callback state uses `runtime/cgo.Handle` converted to an integer
`user_data` value. C never retains or dereferences a pointer into Go-managed
memory. The handle remains live from callback-table installation through Game
destruction and is deleted afterward. Transient Go byte slices passed to C are
valid only for a synchronous call; no Go pointer is retained by CNA.

Every exported callback trampoline recovers panics. Callback errors and recovered
panics are stored in generation-owned Go state, reported to C as
`CNA_RESULT_CALLBACK`, and returned from the outer `Game.Run`; neither a panic
nor a Go error crosses C.

## Ownership and generation

The private ownership categories are `MANAGED_VALUE`, `OWNED`, `BORROWED`,
`PARENT_OWNED`, and `PROCESS_GLOBAL`.

| Facade/state | Category | Authority |
|---|---|---|
| Game | OWNED | created and destroyed by one `Run` generation |
| GraphicsDeviceManager | OWNED Game child | explicit `Dispose`; children first |
| GraphicsDevice | BORROWED | reacquired during each lifecycle callback |
| Texture2D | OWNED Game child | explicit idempotent `Dispose` |
| SpriteBatch | OWNED Game child | explicit idempotent `Dispose` |
| callback context | Go-owned registration state | `cgo.Handle`, deleted after destroy |

Each run receives a monotonically increasing generation. Operations reject a
stale generation deterministically. Native children are destroyed in reverse
registration order before the Game. Failed destruction does not clear a
handle. Go finalizers are not used; explicit disposal and Game teardown are the
only correctness mechanisms.
