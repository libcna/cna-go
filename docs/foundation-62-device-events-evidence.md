# Foundation 62 — GraphicsDevice's six events, and its disposal

Fifteen members close on one partial type — the largest single move of the
session — and the device event bridge joins the game and manager ones.

```text                                          before   after
TOTAL_DIAGNOSTICS                                 184     169
MISSING_MEMBER                                     61      46
COMPLETE_TYPES                                    130     130
UNEXPECTED_MEMBER                                   0       0

BOUND_FUNCTIONS                                    90      95
LAYOUTS                                           238     245
behavior corpus                                   709     712
capability rows                                    61      62
```

`GraphicsDevice` goes from 43 missing members to 28.

## The raise path is native, and that is the settled rule

All six events have a `raise_*` protected virtual in the reference and every
raise site is inside the device's own runtime code: the lost/reset detection,
the graphics-resource base constructor, and `Dispose`. XNA's device plays the
part CNA's device plays, so the canonical CNA signal **is** the reference's raise
path — the same shape `Game`'s three host-driven events have, and the opposite of
`Game::Disposed`, whose raise site is managed and whose native signal is bound
`LIFECYCLE_ONLY`.

```text
Disposing         CNA_GRAPHICS_DEVICE_EVENT_DISPOSING
DeviceLost        CNA_GRAPHICS_DEVICE_EVENT_DEVICE_LOST
DeviceReset       CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESET
DeviceResetting   CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING
ResourceCreated   cna_graphics_device_subscribe_resource_created
ResourceDestroyed cna_graphics_device_subscribe_resource_destroyed
```

Four are CNA identities and two are separate routes carrying a payload, so
CNA-Go indexes all six in one array and `bridge.c` static-asserts that the four
mirror CNA's numbering and that the two cannot alias it.

Subscription is **lazy**: a device nobody listens to installs nothing, and the
first `Add*` installs all six at once, so a partial failure cannot leave some
behind.

## What the two payload events can and cannot report

Neither carries the object, and CNA says why for each.

> The canonical event is raised from the graphics-resource base constructor, so
> the reported object is still under construction: its concrete type does not
> exist yet and no member of it can be queried.

> `name` borrows bytes that are valid only for the duration of the callback. The
> canonical `System::Object*` tag is caller-owned native state, so it is
> reported as presence only.

So `ResourceCreatedEventArgs::Resource` carries nil and
`ResourceDestroyedEventArgs::Tag` carries nil, while `Name` is real — copied out
of the callback-scoped bytes by the trampoline before they expire. Both are
divergences from the reference and both are recorded rather than filled with an
invention.

## Disposal really disposes the device the Game owns

That is what the reference does and it is reproduced. The facade's ownership is
still **BORROWED**: CNA-Go never disposes a device on its own, and the thing the
borrowed-device rule forbids is retaining or releasing the callback-scoped
handle without being asked. A consumer who calls `Dispose` has asked, exactly as
a consumer of the reference has.

`Dispose(false)` is the finalizer's branch. The reference's releases the
unmanaged half without raising; CNA offers one disposal route and no
unmanaged-only variant, so the false branch is a no-op here and says so rather
than calling the same route twice.

## The registration lifetime, and what CNA did not enforce

CNA's contract is explicit:

> A registration is a C-owned resource of the active game. It must be released
> with `cna_graphics_device_unsubscribe` before `cna_game_destroy` succeeds.

The facade lives in the Graphics package and the object whose disposal ends its
life — the `GraphicsDeviceManager` — lives in the framework one, so the release
crosses `internal/servicebridge`, the same seam the facade itself does, with no
public API on either side.

**Measured, and recorded as a fact about CNA rather than a claim about CNA-Go:**
removing that release and running the full isolated cycle produced **no
failure**. The pinned artifacts did not refuse a leaked registration. CNA-Go
releases anyway — the contract requires it, and leaking a C-owned resource is
wrong whether or not something notices — but the release is not proved by a
native refusal, and this milestone does not pretend it is.

The control that does bite is managed: a facade holding a signals value is
released **through the bridge**, and the field is cleared. A releaser installed
for the wrong type, or a release that does not clear the field, both fail it.

## Why the capability row is BACKEND_BLOCKED

REGISTRATION is proved on both artifacts: twenty isolated cycles each installing
and removing all six subscriptions. DELIVERY is not, and the reason is exact:

- `DeviceLost`, `DeviceReset` and `DeviceResetting` are raised only when a
  renderer really loses or resets a device, which neither the HEADLESS nor the
  SOFTWARE artifact can be made to do;
- `Disposing` is raised from a disposal a scenario must not perform on a device
  the Game goes on using.

That is the same classification `visible-rendering` carries, for the same kind
of reason.

## Falsification

| mutation | caught by |
| -------- | --------- |
| the bridge releaser is installed for the wrong type | `the bridge did not reach the Graphics package's releaser` |
| the facade never clears its signals | the same control |
| the registrations are never released | **nothing** — recorded above |

## One robustness fix the controls needed

`DeviceSignals.Release` deleted its `cgo.Handle` unconditionally, and a zero
handle — the state of a signals value that was built but never subscribed —
panics with "misuse of an invalid Handle". It is guarded.

## Qualification

```text
gofmt / go vet / go test ./... / -race    clean
api_compat report                         169 diagnostics, 0 mismatches, 0 leaks
behavior corpus                           712 observations, 0 failures
runtime capabilities                      62 rows, PASS
native ABI                                95 bound, 245 layouts, 0 mismatches
native stress, both artifacts             6 subscriptions per cycle, 0 crashes
```
