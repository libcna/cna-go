# CNA-Go resumable handoff

## Current state

Foundation Milestones 1 through 16 are complete. Milestone 15,
`PURE_MANAGED_BATCH_B`, completed 12 public XNA types carrying 122 mapped Go
identities in three clusters; Milestone 16 then completed
`Microsoft.Xna.Framework.Input.GamePadState`, the type that cluster B unlocked.

Foundation 14 was committed and pushed at
`186f6b163e85587cd949f3c284ac58cde730711c` before this milestone began, so
Foundation 15 started from a clean, synchronized `develop`. Preserve the
established namespace, enum, value-struct, interop, and measured-absence rules.

## Foundation 15 batch

```text
Cluster A — last safe leaf enums (46 identities)
  Graphics.ColorWriteChannels   6   flags
  Graphics.StencilOperation     8
  Graphics.TextureFilter        9
  Input.GamePadType            10
  Graphics.Blend               13

Cluster B — GamePad/Mouse value structs (57 identities)
  Input.GamePadThumbSticks      8
  Input.GamePadTriggers         8
  Input.GamePadDPad            10
  Input.GamePadButtons         17
  Input.MouseState             14

Cluster C — Touch value structs (19 identities)
  Input.Touch.GestureSample     7
  Input.Touch.TouchLocation    12
```

Cluster A closes the safe pure-managed leaf-enum category: no
dependency-complete missing enum remains anywhere in the graph.

## Two authorities, kept distinct

Public surface remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

Reference *behavior* for the value structs was read as IL from the retained
original assemblies, re-verified by hash:

```text
Microsoft.Xna.Framework.dll             38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130
Microsoft.Xna.Framework.Input.Touch.dll b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25
```

They live in `~/Downloads/win/`. `ikdasm` and `monodis` are installed and were
used to read the IL. Every clamping rule, hash algorithm, ToString layout, and
equality rule comes from that IL — nothing from a reimplementation.

## Semantics that must not be "corrected"

- `GamePadThumbSticks` clamps NaN to 1 (XNA `Vector2.Min` keeps its second
  operand); `GamePadTriggers` propagates NaN (`System.Math.Min`/`Max` do), and
  `Math.Max(-0, 0)` turns negative zero into positive zero.
- `GamePadDPad`'s parameter order is `(up, down, left, right)` while its
  fields are declared `up, right, down, left`.
- `GamePadButtons` masks match the pinned `Buttons` literals exactly;
  thumbstick-direction and trigger literals have no field.
- Game pad values hash via `Helpers.SmartGetHashCode` (XOR of 32-bit words,
  zero substituted with `Int32.MaxValue`); `MouseState` uses its own XOR with
  **no** substitution, so a zero snapshot hashes to 0.
- `TouchLocation.op_Equality` compares all seven fields including both state
  fields; `Equals(TouchLocation)` ignores both. They disagree by design.
- Pressed-name accumulators compare against `Pressed` exactly, so an arbitrary
  raw `ButtonState` contributes no name.

## Structural scoreboard

```text
TARGET_TYPES=103
TARGET_MEMBERS=1619
TOTAL_DIAGNOSTICS=331
MISSING_TYPE=154
MISSING_MEMBER=177
COMPLETE_TYPES=98
PARTIAL_TYPES=5
MISSING_TYPES=154

INTERFACE_WITNESS_PROJECTIONS=25
PACKFROMVECTOR4_WITNESS_PROJECTIONS=17
TOVECTOR4_WITNESS_PROJECTIONS=8
```

Every unexpected-symbol, mapping-mismatch, native-leak, allowlist, and
unmeasured counter is zero. All 25 Foundation-14 enum closures, all 5
Foundation-15 enum closures, and all 7 value-struct closures report `PASS`,
the last with `errorResults: 0`.

The five partial types are unchanged: Game 39, GraphicsDeviceManager 40,
GraphicsDevice 70, SpriteBatch 16, Texture2D 12; combined 177.

## Verifier and behavior evidence

Behavior corpus: 487 observations, 487 assertions, zero failures.

The enum machinery is now milestone-agnostic: `TestBatchEnumMappedContracts`
and `TestBatchEnumDefectsRejectedForEveryType` cover all 30 pinned enums and
167 enum identities across 366 negative cases.
`TestFoundation15ValueStructDefectsRejectedForEveryType` adds 70 value-struct
negative cases across 10 defects, including `projected_as_class` (value
semantics), `synthetic_error_result` (infallibility), and `unexpected_mutator`
(immutability). The declared mutation inventory is 217 cases. The value-struct defect matrix
spans 8 types, 91 identities, and 80 negative cases across both milestones.

A verifier defect was fixed: `System.TimeSpan` is now package-qualified as
`framework.TimeSpan` outside the framework package, matching what
`mapping-rules.json` always declared. Expected counts are unchanged at
257 / 3,243.

## Native ABI provenance — Foundation-14 caveat corrected

The Foundation-14 handoff claimed the exact ABI-0.7 library admitted in
Foundation 11 was no longer on this machine. **That was wrong.** A complete
search of all 47 `libcna_c_api.so*` files on the machine found it, still
hash-matching:

```text
sha256 e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f
```

It had survived only in a Foundation-11 era working directory under the system
temporary directory. It has since been **preserved durably**, byte-identical,
at:

```text
~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
```

That directory carries a `PROVENANCE.md` recording the three distinctions
below. Verify the copy by hash rather than by trusting any path:

```sh
sha256sum ~/deps/cna-c-abi-0.7.0-pinned-foundation11/libcna_c_api.so
# e912cd1d239d2c76d67677af4df643703e4348f6a7d6b8983904d95c937b116f

# Or rediscover it anywhere on the machine by hash alone:
find / -name 'libcna_c_api.so*' -type f 2>/dev/null \
  -exec sha256sum {} + | grep ^e912cd1d
```

The header root `~/deps/cna-c-abi-0.7.0/include` reproduces the committed
report against it, so no part of the recipe depends on the temporary
directory any more.

Re-running `native_abi` against that exact binary reproduces the committed
report key for key, differing only in `header_root`, which the committed
evidence stores normalized. The native stress suite reproduces every counter
identically with `GO_RACE_STATUS=PASS`.

```text
EXACT_BINARY_PROVENANCE=VERIFIED
ABI_COMPATIBILITY=VERIFIED           23/67/96/28/2/5, 0 missing, 0 mismatches
BEHAVIORAL_EQUIVALENCE=VERIFIED      on the exercised 20-cycle stress surface
REPRODUCED_BUILD_OUTPUT=NOT_ESTABLISHED   CNA was not rebuilt from source
```

`~/deps/cna-c-abi-0.7.0/libcna_c_api.so`
(`c62949d23d3745964f5e557a06665875621ed4cb6e2930e3f282afd5911f2dcb`) is a
**different, more recent build** of the same ABI. It passes every gate
identically but is not the admitted binary and must never be described as
byte-identical to it. The preservation step did not replace, modify, or rebuild
it, and CNA was not rebuilt from source at any point.

## Re-running gates

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...          # api_compat takes ~135s under -race
go build ./...
go build -trimpath ./...
go run ./tools/api_compat --mode report
go run ./tools/api_compat --mode strict   # expected 331 deferred diagnostics
go run ./tools/api_compat --mode leak-only
go run ./tools/behavior
go run ./tools/packed_vector_qualify
go run ./tools/capabilities --check
go run ./tools/native_abi -headers "$PINNED_HEADERS" -library "$PINNED_LIBRARY" -output "$SCRATCH/native-abi-verify.json"
go run -race ./tools/native_stress --race-status PASS --output "$SCRATCH/native-stress-verify.json"
git diff --check
```

Rerun `--mode report` last so the committed evidence keeps its report mode.
Always send native reports to an explicit `-output` under the scratchpad so a
locally rebuilt library or an absolute header path never rewrites the
committed evidence.

## Dependency graph

The counting rule is unchanged: base type, declared direct interfaces, and
every member signature type are public-signature dependencies; mapped BCL
types are satisfied; the five partial runtime types count as present. The
count moved 55 → 46 across Foundations 15 and 16.

Cluster B unlocked `Microsoft.Xna.Framework.Input.GamePadState`, which
Foundation 16 completed. That unlocked nothing further that is safe.

## No safe candidate remains

**After Foundation 16 the safe pure-managed seam is exhausted.** Every
dependency-complete missing type that is still fully resolved is blocked on
either a public-API policy decision or a fabricated device capability. Search
is no longer the bottleneck — a decision is. The next session should not spend
effort re-ranking; it should start from the list below.

## Blocked on a policy decision, not on semantics

Everything else that is dependency-complete and fully resolved is blocked on a
decision that shapes public API policy. These are recorded so the next session
does not re-derive them:

1. **Pure managed CLR classes project as fallible.** `mapType` treats every
   CLR `class` as a native-backed facade, so every member gains a Go `error`.
   That is right for `Game`, `GraphicsDevice`, and `Texture2D` and wrong for
   pure managed classes. The IL proves `Audio.AudioListener` is nine pure
   field accesses: `Position`/`Velocity` default to `Vector3.Zero`,
   `Forward`/`Up` to `Vector3.Forward`/`Vector3.Up`, and the internal
   `FlipHandedness` (X, Y, −Z) is an involution applied on both read and
   write, so public round-trips are identity. Projecting it honestly means
   classifying it as a managed type. `managedTypes` already exists for exactly
   this, but extending it changes projected public signatures.

2. **Setter-only fallibility cannot be expressed.** `Audio.AudioEmitter` adds
   `DopplerScale`, which defaults to 1 and whose **setter** throws
   `ArgumentOutOfRangeException` when the value is `< 0` — the IL uses
   `bge.un`, so **NaN does not throw**. The getter is pure field access. The
   existing `managedFallibleMembers` table marks a property fallible on *both*
   accessors, so an honest projection needs a new setter-only key.

3. **Managed-interface projection policy.** `IEffectMatrices`, `IEffectFog`,
   `IGameComponent`, and `IGraphicsDeviceManager` are dependency-complete but
   projecting settable-property interfaces needs a general decision about
   error results and witness policy, with no implementor in scope.

4. **`PresentationParameters`** is dependency-complete but exposes
   `DeviceWindowHandle` as `System.IntPtr`, which would surface a raw handle
   in public Go API and interacts with the `RAW_HANDLE_LEAK` gate.

Types whose only surface is read-only device capability with no public
constructor — `TouchPanelCapabilities`, `GamePadCapabilities`,
`RendererDetail`, `DisplayMode` — stay deferred permanently under the
no-fabricated-capability rule, not pending a decision.

## The next milestone is a decision, not a type

Foundation 16 completed `GamePadState`: 15 source identities, 15 mapped Go
identities, `errorResults: 0`. Its private XInput snapshot is reproduced as
unexported managed bit packing, only the `IndependentAxes` dead-zone mode is
reachable and therefore implemented, and every packed bit is measured to equal
its pinned `Buttons` literal. See
`docs/foundation-16-game-pad-state-evidence.md`.

The productive frontier is now group 1 above. Recommended order once the policy
is settled:

1. Classify pure managed CLR classes so they stop projecting fallible members,
   then complete `Audio.AudioListener` (9 identities, fan-out 3).
2. Add setter-only fallibility, then complete `Audio.AudioEmitter`
   (11 identities, fan-out 3).
3. Decide managed-interface projection, then take `IEffectMatrices` and
   `IEffectFog` (fan-out 5 each).
4. Decide whether `System.IntPtr` may appear in public Go surface at all; that
   gates `PresentationParameters` (13 identities, fan-out 2) and interacts with
   the `RAW_HANDLE_LEAK` gate.

`Graphics.GraphicsResource` (fan-out 11) remains the largest single unlock and
the real architectural milestone, but it needs the disposal/ownership design
plus the partial `GraphicsDevice`, so it is not a next step until the questions
above are settled.

```text
SELECTED_ONLY=true
STARTED=false
```

## Worktree provenance

Foundation 15 started on clean `develop` with `HEAD` and `origin/develop` both
at `186f6b163e85587cd949f3c284ac58cde730711c` and landed as exactly one commit.
Foundation 16 started from that commit and landed as one more. Neither used
per-type commits and history was not rewritten.

```text
FOUNDATION_MILESTONE_15_COMPLETE=true
BATCH_NAME=PURE_MANAGED_BATCH_B
FOUNDATION_MILESTONE_16_COMPLETE=true
```
