# Foundation 18 — projected CLR interface contracts

Foundation 18 closes four dependency-complete CLR interfaces and settles the
managed-interface projection policy that had blocked them. It declares four
contracts and implements none of them: CNA-Go still has no effect runtime, no
game component, and no device manager bound to the contract.

## Reference authority

| assembly                                | sha256                                                             | declares                                  |
| --------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------- |
| `Microsoft.Xna.Framework.Graphics.dll`  | `560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55` | `IEffectMatrices`, `IEffectFog`           |
| `Microsoft.Xna.Framework.Game.dll`      | `b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0` | `IGameComponent`, `IGraphicsDeviceManager` |

Both assemblies are retained originals, read with `ikdasm`. The public surface
remains the pinned contract at SHA-256
`7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`.

## The rule this milestone settles

An interface contributes no fallibility of its own. Each projected operation is
classified by the execution boundary it crosses, and **the boundary is read
from the reference implementor IL in the assembly that declares the
interface** — never from a guess about an implementor that does not exist.

Where every shipped implementor agrees, that agreement is the contract's
measured behavior, not speculation. Where a contract has no implementor whose
behavior can be read, it stays deferred rather than guessed at.

Because Foundation 17 made fallibility per operation, one contract can now mix
the two boundaries honestly. `IEffectFog` is the first that does.

## `IEffectMatrices` — uniformly managed

3 source members, 6 projected identities, **no error result**.

```go
type IEffectMatrices interface {
    World() framework.Matrix
    SetWorld(value framework.Matrix)
    View() framework.Matrix
    SetView(value framework.Matrix)
    Projection() framework.Matrix
    SetProjection(value framework.Matrix)
}
```

All five shipped implementors — `AlphaTestEffect`, `BasicEffect`,
`DualTextureEffect`, `EnvironmentMapEffect`, `SkinnedEffect` — back every
accessor with a managed field read or write plus a managed dirty-flag OR:

```text
BasicEffect::set_World
  IL_0002: stfld  Matrix BasicEffect::world
  IL_0009: ldfld  EffectDirtyFlags BasicEffect::dirtyFlags
  IL_000e: ldc.i4.s 19
  IL_0010: or
  IL_0011: stfld  EffectDirtyFlags BasicEffect::dirtyFlags

BasicEffect::get_World
  IL_0001: ldfld  Matrix BasicEffect::world
```

No implementor touches a device on any of the six operations, so none of them
can fail.

## `IEffectFog` — measured mixed boundary

4 source members, 8 projected identities, **exactly two error results**.

```go
type IEffectFog interface {
    FogEnabled() bool
    SetFogEnabled(value bool)
    FogStart() float32
    SetFogStart(value float32)
    FogEnd() float32
    SetFogEnd(value float32)
    FogColor() (framework.Vector3, error)
    SetFogColor(value framework.Vector3) error
}
```

The same five implementors split, unanimously and in both directions:

| operation                          | all five implementors                              | fallible |
| ---------------------------------- | -------------------------------------------------- | -------- |
| `FogEnabled` get/set               | managed field + dirty flag                          | no       |
| `FogStart` get/set                 | managed field + dirty flag                          | no       |
| `FogEnd` get/set                   | managed field + dirty flag                          | no       |
| `FogColor` get/set                 | `EffectParameter` → unmanaged D3DX                  | **yes**  |

`BasicEffect::set_FogColor` is not a field store at all:

```text
IL_0001: ldfld    EffectParameter BasicEffect::fogColorParam
IL_0007: callvirt EffectParameter::SetValue(Vector3)
```

and `EffectParameter::GetValueVector3` ends in an unmanaged call whose HRESULT
is checked and converted into a throw:

```text
IL_0038: calli    unmanaged stdcall int32(native int, uint8*, float32*)
IL_003d: stloc.3
IL_003f: ldc.i4.0
IL_0040: bge.s    IL_0049
IL_0043: call     GraphicsHelpers::GetExceptionFromResult(uint32)
IL_0048: throw
...
IL_007c: newobj   System.InvalidCastException::.ctor()
IL_0081: throw
```

So both `FogColor` accessors genuinely cross a qualified runtime boundary and
both take an error result, while their six siblings do not. This is exactly
what the per-operation rule exists for, and it differs from
`AudioEmitter.DopplerScale`, where only *one* accessor of a property is
fallible.

## `IGameComponent` — runtime boundary

1 source member, 1 projected identity, fallible.

```go
type IGameComponent interface {
    Initialize() error
}
```

The two shipped implementors disagree about how much they do, and that is the
point. `GameComponent.Initialize` is a bare `ret`. `DrawableGameComponent.Initialize`
resolves `IGraphicsDeviceService` out of `Game.Services`, **throws
`System.InvalidOperationException` when the service is absent**, subscribes to
four device events, reads `GraphicsDevice`, and calls `LoadContent`:

```text
IL_0027: callvirt GameServiceContainer::GetService(Type)
IL_002c: isinst   IGraphicsDeviceService
IL_003c: brtrue.s IL_0049
IL_003e: call     Resources::get_MissingGraphicsDeviceService()
IL_0043: newobj   System.InvalidOperationException::.ctor(string)
IL_0048: throw
```

The contract therefore reaches the game and graphics runtime, and an
implementation that cannot initialize needs somewhere to say so. The projection
matches the established `GameCallbacks.Initialize(*Game) error`.

## `IGraphicsDeviceManager` — runtime boundary

3 source members, 3 projected identities, all fallible.

```go
type IGraphicsDeviceManager interface {
    CreateDevice() error
    BeginDraw() (bool, error)
    EndDraw() error
}
```

In the reference, `CreateDevice` delegates to
`GraphicsDeviceManager.ChangeDevice(true)`, `BeginDraw` to `EnsureDevice`, and
`EndDraw` calls `GraphicsDevice.Present()` inside a `catch` for
`Graphics.DeviceLostException`. Every operation is device lifecycle.

`BeginDraw` shows the channel rule at work. Its Boolean is a **source** result —
whether drawing may proceed — and stays distinct from the `error`, which
reports whether the call itself failed. Declining to draw is `(false, nil)`;
failing is `(false, err)`. The two are asserted separately.

Declaring the contract binds no implementor. CNA-Go's `GraphicsDeviceManager`
remains a partial native-backed facade and is deliberately *not* projected as
implementing this interface; a test asserts that it does not.

## Deferred, with exact reasons

| interface                | fanout | why it is still deferred                                                                                                                                            |
| ------------------------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `IUpdateable`            | 1      | Declares two events typed `System.EventHandler<System.EventArgs>`. That BCL generic delegate has no declared mapping and currently degrades to `any`, which would lock a lossy public signature. |
| `IDrawable`              | 1      | Same two-event problem.                                                                                                                                              |
| `IGraphicsDeviceService` | 1      | Same event problem on four events, and its one property returns the protected partial `GraphicsDevice`.                                                              |
| `IEffectLights`          | —      | Not dependency-complete: three properties return the unimplemented `DirectionalLight`.                                                                              |

The event deferral is a real, specific mapping gap — a named Go handler type
for `System.EventHandler<T>` — not a semantics gap. It is recorded rather than
papered over with `any`.

## Verifier coverage

`managedInterfaceClosure` measures each contract: Go kind, classification flag,
boundary label, source identities against projected identities, accessor pairs,
and a per-operation fallibility verdict checked against a **pinned table of
which operations may be fallible**. That table, not the mapping tables, is the
authority, so editing a classification alone cannot silently move a boundary.

```text
IEffectMatrices         PURE_MANAGED               classified   3 -> 6   3 pairs  0 errors  PASS
IEffectFog              MIXED_MANAGED_AND_RUNTIME  classified   4 -> 8   4 pairs  2 errors  PASS
IGameComponent          RUNTIME                    unclassified 1 -> 1   0 pairs  1 error   PASS
IGraphicsDeviceManager  RUNTIME                    unclassified 3 -> 3   0 pairs  3 errors  PASS
```

### Negative fixtures

Target-side: 11 defects across 4 contracts, 37 applied and 7 skipped as
inexpressible on a contract of that shape. Skips are counted and asserted, not
silently dropped:

```text
IEffectMatrices         2 skipped  no fallible operation exists
IEffectFog              0 skipped  the mixed contract expresses every defect
IGameComponent          3 skipped  no parameters, no infallible operation, no value result
IGraphicsDeviceManager  2 skipped  no parameters, no infallible operation
```

The defects are `missing_type`, `wrong_package`, `projected_as_struct`,
`missing_first_member`, `missing_last_member`, `renamed_last_member`,
`unexpected_member`, `wrong_parameter`, `artificial_error`, `dropped_error`,
and `error_replaces_source_result`.

Classification-side: 4 defects attacking the interface rule in both directions
by mutating the real classification tables, each asserting the exact
accessor-and-direction wording:

```text
pure_managed_interface_demoted_to_runtime    IEffectMatrices loses six managed accessors
runtime_interface_admitted_as_pure_managed   IGraphicsDeviceManager loses its device errors
interface_runtime_operation_dropped          IEffectFog loses the one D3DX boundary
interface_runtime_operation_widened          IEffectFog invents six errors
```

The mutation inventory grows from 249 to 290.

## Conformance and behavior

Compile-time conformance doubles in the `graphics` and `framework` package
tests prove each method set is satisfiable exactly as projected; if a signature
drifts, the build fails rather than a test. The doubles are test-only and
reproduce no XNA behavior.

The behavior corpus grows from 535 to 541 with zero failures. All six new
observations are labeled `GO_LANGUAGE_PROJECTION`, because a contract with no
implementation has no XNA runtime behavior to observe — what is observed is
that the projection is satisfiable, that the fallibility split survives into
the method set, and that `BeginDraw`'s two channels stay separate.

## Structural effect

```text                       before   after
TARGET_TYPES                    105     109
TARGET_MEMBERS                 1639    1657
TOTAL_DIAGNOSTICS               329     325
MISSING_TYPE                    152     148
MISSING_MEMBER                  177     177
COMPLETE_TYPES                  100     104
PARTIAL_TYPES                     5       5
```

The five protected partial runtime types are untouched, and every mismatch,
leak, allowlist, and unmeasured counter stays at zero.

## No capability claim and no ABI change

Declaring these contracts adds no effect runtime, no shader, no component
lifecycle, no device creation, and no presentation. Nothing here reaches native
code, so the CNA-Go ABI is unchanged at `23 / 67 / 96 / 28 / 2 / 5` with no
missing symbols and no mismatches. CNA was not rebuilt.
