# Native ABI boundary

CNA-Go binds only CNA's canonical C ABI. It never links or reflects CNA's C++
ABI and does not route through another language binding.

## Admission and loading

CNA-Go admits **CNA C ABI major 0 with minor 21 or newer**, qualified at
0.21.0 (encoded `0x00001500`). A different major, a lower minor, a missing
required symbol, a resolved pointer that belongs to a different symbol, or a
loader failure rejects the library before Game creation. A rejection names the
library path, the version it reported, and the admitted range.

The range is CNA's own, not CNA-Go's preference.
`modules/c-api/cmake/CnaCApiExports.map` states that the ELF symbol-version
node `CNA_C_API_0.1` "is NOT the ABI version and must not be bumped with it",
that it "changes only for a *major* ABI break", and that renaming it "turns
every additive release into a hard break". So CNA declares a major bump to be
the break and minor bumps to be additive, and `docs/releasing.md` separately
states that `CNA_ABI_VERSION` "moves when the ABI changes, independently of a
product release". The floor is the qualified minor because a lower one may
simply not declare a route CNA-Go binds; the upper end is open because CNA
says an additive release keeps the contract. Nothing is taken on trust: after
the version check every required symbol is still resolved by name, and every
resolved address is still confirmed with `dladdr` to belong to the symbol the
manifest lists.

Foundation 1 originally admitted exactly `0x00000700` (0.7.0). That is history,
not the current contract; the migration that replaced it is recorded in
[Foundation 44](foundation-44-abi-migration-evidence.md), including the
compiler-backed proof that every route CNA-Go binds is byte-identical across
the fourteen minor bumps in between.

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

## Qualified artifact

The qualification library is the CNA C API built from the live `cnanext`
checkout, retained at `~/deps/cna-c-abi-0.21.0/libcna_c_api.so`. CNA's checkout
was not modified and nothing was rebuilt for CNA-Go: the artifact is
byte-identical to `cnanext/cmake-build-headless/modules/c-api/libcna_c_api.so`,
and the header tree beside it is byte-identical to
`cnanext/modules/c-api/include`.

```text
cnanext HEAD              0a6158e4ff764907065cd7259e3d29e331a52088 (next)
sharp-runtimenext HEAD    4a49afb0cfe6a41e6e0af0bb62dc5175976731bb (next)
configuration             CNA_GRAPHICS_RENDERER=HEADLESS, CNA_PLATFORM=SDL3,
                          CNA_AUDIO_PLATFORM=SDL3, CNA_ENABLE_NET=ON,
                          CNA_ENABLE_VIDEO=AUTO, CMAKE_BUILD_TYPE=Debug
artifact                  libcna_c_api.so, 166,420,656 bytes
sha256                    c32bfbd307d695664f906ccf2834ec3f9ebc240fa388d544ac21ee3ebaeb731b
canonical headers         ~/deps/cna-c-abi-0.21.0/include, 61 .h files
header tree sha256        62c3f6e4bec6d8396ec576986b4ec5b158af28cfe52aa7e2ede1e4c26c90bff6
reported ABI              0.21.0
canonical declarations    4054
exported cna_* routes     4054 (exact correspondence, both directions)
symbol-version node       CNA_C_API_0.1
```

### The header tree is pinned by content, and the live checkout has moved

The header root used to be recorded as a path. Milestone 55 measured why that is
not enough: `tools/native_abi`'s default `-headers ../../cnanext/modules/c-api/include`
reads the LIVE `cnanext` checkout, and that checkout has advanced past the
revision the pinned library was built from. The two trees differ:

```text
pinned  ~/deps/cna-c-abi-0.21.0/include   62c3f6e4bec6d8396ec576986b4ec5b158af28cfe52aa7e2ede1e4c26c90bff6
live    ../../cnanext/modules/c-api/include  2d7445e7b2c0c74d3b32fab6067ef701662076b9445a73243cdf9639f36698ed
```

Both produce identical measurements at this revision -- the divergence is
documentation comments in `CNA/C/devices.h`, from cnanext's browser
device-type change -- so nothing was wrong. But nothing would have SAID so
either, and a declaration change would have been read against a library that
does not have it.

The report therefore records `canonical_header_sha256` and
`canonical_header_files` beside the existing `native_library_sha256`, on the
principle the library already followed: reports retain content identity, not an
ephemeral qualification path. The qualification invocation names the pinned
tree explicitly:

```sh
go run ./tools/native_abi \
  -headers ~/deps/cna-c-abi-0.21.0/include \
  -library ~/deps/cna-c-abi-0.21.0/libcna_c_api.so
```

Header/library correspondence is measured rather than assumed: the canonical
headers declare 4,054 `cna_*` routes and the library exports exactly those
4,054 names, with no route declared and unexported and none exported and
undeclared. The verifier compares both counts and reports a mismatch.

This artifact is qualification input, not a distributed CNA-Go file. It is not
sanitizer-instrumented, so `NATIVE_SANITIZER_STATUS=NOT_RUN`; stress results do
not constitute native leak-freedom evidence. Its HEADLESS renderer proves
native graphics execution but not visible output. Its audio backend is SDL3
rather than the NULL backend Foundation 1 used, so audio is no longer blocked
by the artifact — only by CNA-Go's own missing audio surface.

### The retired Foundation 1 artifact

Foundation 1 through 43 were qualified against a separate 16,799,760-byte
`libcna_c_api.so` with SHA-256
`e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f`, built from
CNA revision `a09196a6477f69a7a57c8364f990658d31531a5b` with sharp-runtime
`54578590b328aa9612fe38bfddca9fd8ca795144`, GCC 14.2.0, Release, HEADLESS
rendering and **NULL** audio. That artifact declared ABI 0.7.0 and is retained
for history only. No current gate loads it.

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

`go run ./tools/native_abi -headers /absolute/path/to/cna/modules/c-api/include -library /absolute/path/to/libcna_c_api.so`
performs five independent checks:

1. GCC compiles pointer assignments from every private manifest typedef to the
   declaration of the **same name** in canonical CNA headers, with
   incompatible-pointer warnings as errors. Each route is paired with its own
   canonical declaration, not with a compatible neighbour.
2. A canonical-header probe measures sizes, alignments, offsets, field widths,
   callback types, ABI/result constants and the admission policy itself. It is
   the only translation unit that sees the canonical header, CNA-Go's private
   manifest and `bridge.h` at once, so it is where all three mirrors are
   compared rather than trusted.
3. A **manifest-only** probe measures the same list with no canonical header in
   scope at all — the exact environment cgo gives `bridge.c`. The two
   measurement sets are compared key by key. Without this, CNA-Go's own struct
   declarations were never measured against anything: the canonical probe
   measures canonical types, because `abi_manifest.h` suppresses its private
   definitions whenever a CNA header is present. The manifest probe refuses to
   compile if a canonical header reaches it.
4. The production loader admits the selected library under the range policy,
   proves every manifest export is present, and confirms with `dladdr` that
   every resolved pointer belongs to the symbol the manifest names. That last
   check is what separates routes which share a prototype: `cna_game_run`,
   `cna_game_request_exit` and `cna_game_destroy` are all
   `CNA_Result(CNA_Handle)`, so a mis-pairing among them would compile cleanly.
5. The canonical declaration count and the library's exported `cna_*` count are
   compared, and the header-declared ABI is compared with the ABI the loaded
   library reports.

The route table is no longer maintained beside the manifest: `tools/native_abi`
parses `CNA_GO_REQUIRED_SYMBOLS` and each route's own `_fn` typedef out of
`abi_manifest.h`, which is the file the cgo build compiles. A required symbol
with no prototype of its own is an error rather than a route counted as taking
zero arguments.

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
