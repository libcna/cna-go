# Foundation 75 — `GraphicsDeviceInformation`, and the rest of `GraphicsDeviceManager`

This milestone projects the two root types the manager needed and closes the
manager itself. `Game` is now the profile's only partial type, and its one
missing member is `ShowMissingRequirementMessage(System.Exception)`.

```text
COMPLETE_TYPES   155 -> 158        MISSING_TYPE      100 -> 98
PARTIAL_TYPES      2 ->   1        MISSING_MEMBER      7 ->  1
graphicsManagerRemainingMissing    6 -> 0
```

## Reference authority

```text
Microsoft.Xna.Framework.Game.dll
  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
Microsoft.Xna.Framework.Graphics.dll
  560080fc39021c61...   (GraphicsAdapter, for the equality question below)
```

Three Microsoft resource strings were read out of the Game assembly's
`.resources` stream and are now registered:
`NoNullUseDefaultAdapter`, `NoCompatibleDevices` and
`NoCompatibleDevicesAfterRanking`. The first of those exposed a gap in the
resource-string gate: it scanned only single string literals, so a message
spelled across several source lines — which `NoCompatibleDevices` is, at five
CRLF-separated paragraphs — was invisible to it. The scanner now folds `+`
chains, and the message is checked byte for byte like every other.

## Where the three types live, and why

`GraphicsDeviceInformation` is declared in `Microsoft.Xna.Framework`, and all
three of its fields are `Microsoft.Xna.Framework.Graphics` types. The Graphics
package imports the framework package, so the dependency cannot be inverted, and
the settled cross-package cycle rule already answers the property half: the
three properties project as `graphics.GraphicsDeviceInformationAdapter` and
friends, while the state stays on the framework type as `any` and a raw `int32`.

What that rule did not answer is the **bodies**. `Equals`, `GetHashCode`,
`Clone` and the constructor are declared on the type, and each reaches something
only the Graphics package can spell. The split this milestone takes is:

- the **logic** stays where the reference declares it — which fields, in which
  order, with which short-circuit;
- each **operation it cannot spell** crosses through `internal/servicebridge`:
  allocate a `PresentationParameters`, clone one, read its ten values, ask for
  the default adapter, read a device's profile, collect candidates, rank them.

`Equals` and `GetHashCode` read a `PresentationSnapshot` — the ten values in the
reference's own order — rather than asking the Graphics package "are these
equal". Handing across the values instead of the answer is what keeps the
comparison chain in the type that declares it.

The two genuinely **private** helpers, `AddDevices` and
`GraphicsDeviceInformationComparer`, are projected in the Graphics package
instead, because every operand they touch is a Graphics type and neither is
public surface.

## A CLR interface Go cannot put on the class

Closing `GraphicsDeviceManager` made it a COMPLETE type, and the declared
interface conformance check runs only on complete types — so it fired for the
first time and failed:

```text
CLR metadata declares Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService
on this class, but *GraphicsDeviceManager does not satisfy the Go projection
graphics.IGraphicsDeviceService: AddDeviceCreatedHandler has the wrong signature
```

The real blocker is the interface's one non-event member,
`GraphicsDevice() *graphics.GraphicsDevice`. The cross-package rule relocates a
CLASS member whose type belongs to a descendant package; an INTERFACE member has
nowhere to go, because its signature is part of the interface, and the ancestor
package cannot name the descendant type at all.

The conformance is carried by `managerDeviceService`, an adapter in the Graphics
package — and this is not a workaround invented for the verifier. It is what the
projection already did in Foundation 49, and it is the object a consumer
resolving `IGraphicsDeviceService` out of the game's service container actually
receives.

`crossPackageInterfaceCarriers` records the pair with the exact blocking member,
and the verifier **compiler-checks the carrier** exactly as it checked the
class. A class that fails its declared interface with no registry entry is still
`INTERFACE_MAPPING_MISMATCH`, so the escape hatch cannot be reached by accident.
The verdict is `PASS_VIA_CARRIER` and it is counted separately.

## A reference defect, reproduced on purpose

`GraphicsDeviceInformation::set_Adapter` is

```text
IL_0000: ldarg.0
IL_0001: ldfld  adapter          // THIS.adapter, not `value`
IL_0006: brtrue.s IL_0018
IL_0008: throw ArgumentNullException("value", NoNullUseDefaultAdapter)
IL_0018: this.adapter = value
```

The guard tests the field the setter is about to **overwrite**, not the argument
it was given. So:

- assigning **null succeeds** whenever the current adapter is non-null, which it
  always is after the constructor;
- the "Adapter cannot be null. Try using GraphicsAdapter.DefaultAdapter
  instead." message can only ever be raised on an information whose adapter is
  **already** null.

Both halves are pinned by
`TestGraphicsDeviceInformationAdapterSetterReproducesTheReferenceBug`. Correcting
the guard would make CNA-Go refuse an assignment the reference accepts, which is
a different API and not the one this binding projects.

## `Clone`, and why it is fallible

```text
GraphicsDeviceInformation copy = new GraphicsDeviceInformation();
copy.presentationParameters = this.presentationParameters.Clone();
copy.adapter                = this.adapter;      // ALIASED
copy.graphicsProfile        = this.graphicsProfile;
```

The asymmetry is load-bearing: the parameters are **deep-copied**, so a clone
can be reconfigured without disturbing the source, while the adapter is
**aliased**, so both informations name the same device. `AddDevices` depends on
exactly that — it builds one base information per adapter and clones it per
display mode.

`Clone` is fallible because its first instruction is `newobj .ctor()`, and the
constructor's second statement is `adapter = GraphicsAdapter.DefaultAdapter`,
whose CNA-Go projection enumerates CNA's adapters. The clone throws that adapter
away immediately; a `Clone` that skipped the call would be a different method
that happens to produce the same fields.

## `GetHashCode`, and the one contributor that cannot be reproduced

The reference XORs eleven values. Ten of them are exactly reproducible from the
pinned mscorlib: `Int32.GetHashCode` returns the value, `Boolean.GetHashCode`
returns 1 or 0, `IntPtr.GetHashCode` is `(int)(ulong)m_value`, and
`Enum.GetHashCode` is the underlying `Int32`'s.

The eleventh is `adapter.GetHashCode()`. `GraphicsAdapter` declares neither
`Equals` nor `GetHashCode` — checked in the Graphics assembly — so the
contributor is `System.Object`'s identity hash, which the CLR derives from a
sync-block index that is unspecified and **differs between runs of the reference
itself**. CNA-Go contributes a per-object identity hash with the same two
properties: stable for the object's lifetime, distinct between objects. A caller
comparing a hash against a number the reference produced is comparing two
unspecified values, which is equally true of the reference against itself.

The same fact settles `Equals`: `other.adapter.Equals(this.adapter)` is
reference identity, which Go's `==` on the pointer facade spells exactly.

## The ranking policy

`RankDevices` forwards to `foundDevices.Sort(new GraphicsDeviceInformationComparer(this))`,
and the comparer's eight steps are, in order:

| # | step | winner |
| --- | --- | --- |
| 1 | `GraphicsProfile` | higher |
| 2 | `IsFullScreen` | the one matching the MANAGER's `IsFullScreen` |
| 3 | back-buffer format rank | lower — 0 exact, 1 same bit depth, `int.MaxValue` otherwise |
| 4 | `MultiSampleCount` | higher |
| 5 | aspect ratio | closer to the preferred aspect, **only** if the two distances differ by more than 0.2 |
| 6 | pixel count | closer to the target area |
| 7 | adapter | the DEFAULT adapter |
| 8 | — | tie |

Two details are easy to lose. Step 5's **0.2 window** makes the ordering
deliberately non-total: two candidates can tie on aspect and be separated only
by area. And step 6's target has three branches, one of which gives the two
candidates **different** targets — full screen with no preferred size compares
each candidate against its own adapter's current display mode.

`SurfaceFormatBitDepth` maps exactly **five** of the twenty `SurfaceFormat`
values (`Color` and `Rgba1010102` to 32, `Bgr565`, `Bgra5551` and `Bgra4444` to
16) and answers zero for every other one — so the floating-point and compressed
families all share a bit depth and rank 1 against each other. That is the
reference's table, not an omission, and it is pinned.

`List<T>.Sort` is `Array.Sort`, which is introsort and is **not stable**. The
projection uses `sort.Slice` rather than `sort.SliceStable`, because imposing a
stability the reference does not have would make CNA-Go deterministic where XNA
is not.

## The one platform substitution, measured

`AddDevices` filters adapters with the private
`IsWindowOnAdapter(windowHandle, adapter)` when `anySuitableDevice` is false:

```text
WindowsGameWindow.ScreenFromAdapter(adapter) == WindowsGameWindow.ScreenFromHandle(handle)
```

`ScreenFromAdapter`, read from the same assembly, is a linear scan of
`Screen.AllScreens` for the screen whose device name equals
`adapter.DeviceName`. So **both sides of the comparison reduce to a display
device name**, and both names are members the pinned contract already declares:
`GraphicsAdapter::get_DeviceName` and `GameWindow::get_ScreenDeviceName`. The
projection compares those two strings.

This is not `System.Windows.Forms.Screen` reproduced — CNA-Go has none, and
inventing one would be fabrication. It is the reference's own **predicate**
expressed through the two reference members that carry its operands.

## `List<T>` at a signature position

`RankDevices(List<GraphicsDeviceInformation>)` is the **only** position in the
whole XNA 4.0 Windows profile that carries `System.Collections.Generic.List<T>`
itself — the other four `List`1` hits are the nested `Enumerator`, which the
settled rule already maps.

What the reference does with it here is sort it in place and read it by index.
The settled signature rule — a BCL type at a public signature position takes the
standard-library Go type whose ROLE it is, chosen from what the profile's
positions measurably do with it — gives a Go slice. It is the same reasoning
that made `System.IO.Stream` an `io.Reader` because every stream position in the
profile is read.

A slice cannot grow or shrink through the callee and a `List<T>` can. No
projected member in the profile does: the private `AddDevices` that appends is
not public surface.

## Falsifiability

Twenty-one new tests, none skipped.

`GraphicsDeviceInformation`'s equality and hash are measured by mutating each of
the **twelve** contributors one at a time and requiring both to move — a hash
that ignored a value would survive the equality assertion and is killed by the
hash one. The clone's deep/alias asymmetry is measured in both directions: a
change to the clone must not reach the source, and the adapter must be the same
object. The comparer's eight steps are each measured with the earlier ones held
equal, so exactly one decides.

## Qualification

```text
go test ./...                        PASS
go run ./tools/behavior              OBSERVATIONS=737 ASSERTIONS=737 FAILURES=0
go run ./tools/api_compat            TOTAL_DIAGNOSTICS=99, all of them
                                     MISSING_TYPE/MISSING_MEMBER; every mismatch,
                                     leak, unexpected and allowlist category 0
go run ./tools/native_abi ...        BOUND_FUNCTIONS=230 ABI_MISMATCHES=0
go run ./tools/external_consumer     TESTS=100 FAILURES=0 STATUS=PASS
go run ./tools/resource_strings      CLAIMED=44 VERIFIED=44 FINDINGS=0
```

The ABI is untouched: this milestone binds no route. Everything it added is
managed policy over members that were already bound.
