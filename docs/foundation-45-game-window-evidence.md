# Foundation 45 — GameWindow, and the orientation question answered

`GameWindow` was the highest-value remaining frontier and carried exactly one
open sub-question. This milestone completes the type — 20 XNA members, 25 Go
identities, no partial — binds ten CNA window routes, and answers the open
question from the reference rather than from CNA.

`Game.Window` becomes the fifteenth `Game` member to be projected, and `Game`
drops from 8 missing members to 7.

## The open question: `SetSupportedOrientations`

The retained analysis said this member stood between a partial and a complete
`GameWindow`, because CNA exposes orientation configuration only on
`cna_graphics_device_manager_set_supported_orientations`, which takes a
**manager** handle and belongs to a different object. Current CNA 0.21.0 was
re-audited: there is still no `cna_game_window_set_supported_orientations`, and
`cna_touch_panel_set_display_orientation` belongs to the touch panel.

That framing was the wrong way round. The question is not what CNA offers; it
is what the reference does. `Microsoft.Xna.Framework.GameWindow::SetSupportedOrientations`
is `famorassem` abstract, and the selected Windows runtime profile ships exactly
one implementor:

```
.method famorassem hidebysig virtual instance void
        SetSupportedOrientations(DisplayOrientation orientations) cil managed
{
  // Code size       1 (0x1)
  IL_0000:  ret
}
```

**The body is a single `ret`.** In XNA 4.0 on Windows the member does nothing at
all. It is not a CNA gap, not a deferral and not a platform limitation: a
projection that forwarded it to the manager's native route would be *inventing*
behaviour the reference does not have, on an object that does not own the state.

The same audit settled a second member the same way.
`WindowsGameWindow::get_CurrentOrientation` is

```
IL_0000:  ldc.i4.0
IL_0001:  ret
```

a **constant**. The reference never asks the platform for an orientation in this
profile; it answers `DisplayOrientation.Default` on every machine, in every
window state. So `cna_game_window_get_current_orientation` is deliberately left
unbound: binding it would create a second source of truth that could report a
rotated orientation where XNA reports `Default` — and be wrong precisely by
being right.

Both members are therefore complete and infallible. Neither reaches CNA.

## The measured null-guard split

`WindowsGameWindow`'s members divide cleanly, and CNA-Go's projection uses that
division rather than a policy of its own. Every row was read from the IL:

| member | with no platform window | CNA-Go with no live native game |
| -- | -- | -- |
| `get_Handle` | `IntPtr.Zero` | `0`, no failure |
| `get_AllowUserResizing` | `false` | `false`, no failure |
| `set_AllowUserResizing` | nothing happens | nothing happens, no failure |
| `SetTitle` | nothing happens | nothing happens, no failure |
| `get_ScreenDeviceName` | `String.Empty` | `""`, no failure |
| `get_ClientBounds` | **NullReferenceException** | reports a failure |
| `BeginScreenDeviceChange` | **NullReferenceException** | reports a failure |
| `EndScreenDeviceChange(3)` | **NullReferenceException** | reports a failure |

The five guarded members test `if (this.mainForm == null)` and return a
documented fallback. The three unguarded ones dereference `mainForm` directly.
CNA-Go answers the fallback for the first group and reports a failure for the
second, and `native_stress` proves both halves against the same object in one
process: 120 guarded-fallback checks and 80 unguarded-failure checks across 20
cycles, before and after a run, with 120 live reads in between.

Returning a zero `Rectangle` from `ClientBounds` instead would have been the
tempting shortcut and is the reason the split matters: a zero-sized rectangle is
a legitimate value a real window can report — the HEADLESS artifact reports
exactly that — so it cannot also mean "there is no window".

## `Game.Window` identity

`Game::.ctor` calls `EnsureHost()` before its fifth statement and then reads
`host.Window` to subscribe its own `Paint` handler, so the host and its window
exist from construction. `Game::get_Window` is `host == null ? null :
host.Window`, and `WindowsGameHost::get_Window` is one `ldfld` over a field the
host constructor assigns. For a constructed `Game` the null branch is
unreachable and every call returns the **same** object.

CNA-Go allocates one `GameWindow` in `NewGame` and returns it forever.
`Game.Window` is therefore classified as a managed stored member — two field
reads, reaching no window, no device and no platform — and is infallible.

The identity is not a tidiness point. A consumer subscribes through
`game.Window().AddClientSizeChangedHandler(...)` and raises through a second
`game.Window()` call; a projection that allocated a wrapper per call would pass
every test written with one local variable and silently orphan every real
subscription. Three of the 60 identity checks per stress run are before, during
and after the run, and the external canary re-derives the window from `Game` at
every step.

What CNA-Go deliberately does **not** reproduce is a native window at
construction time, because CNA-Go still creates the native game inside `Run`.
That difference is not hidden: it is exactly the state the reference itself is
in before its form exists, and the guard split above is the reference's own
answer to it.

## `assembly` maps to package scope

`GameWindow` declares six events. Three — `ScreenDeviceNameChanged`,
`ClientSizeChanged`, `OrientationChanged` — are `public`. Three —
`Activated`, `Deactivated`, `Paint` — are `assembly`: only
`Microsoft.Xna.Framework.Game.dll` can subscribe, and `Game::.ctor` is the one
place that does.

CLR `assembly` visibility is Go's unexported package scope, so those three
project as unexported registration lists with no accessor pair. Their protected
`On…` raisers *are* public contract members and are projected; with nothing
subscribed they raise nothing and report no failure, which is what the
reference's null check does. That is why the public contract lists three events
and the Go type carries six lists.

## The routes, and the three left unbound

Ten CNA routes are bound. Every one takes the **game** handle: CNA models the
window as a property of the game, so nothing here owns a native lifetime and
there is nothing to dispose.

```text
cna_game_window_get_allow_user_resizing        AllowUserResizing
cna_game_window_set_allow_user_resizing        SetAllowUserResizing
cna_game_window_get_client_bounds              ClientBounds
cna_game_window_get_native_handle_ext          Handle
cna_game_window_get_screen_device_name_size    ScreenDeviceName  (two-call read)
cna_game_window_copy_screen_device_name        ScreenDeviceName
cna_game_window_begin_screen_device_change     BeginScreenDeviceChange
cna_game_window_end_screen_device_change       EndScreenDeviceChange(3)
cna_game_set_window_title                      SetTitle
cna_game_window_subscribe                      the three public events
```

Three canonical routes are deliberately **not** bound, and each omission is a
measurement:

- `cna_game_window_get_title_size` / `cna_game_window_copy_title` —
  `GameWindow::get_Title` is `ldarg.0; ldfld string GameWindow::title; ret` on
  the abstract base. The field is managed state the setter wrote; a native
  getter would be a second source of truth. This is the same rule that keeps the
  `cna_game_get_*` timing getters unbound.
- `cna_game_window_get_current_orientation` — the reference is a constant, as
  above.

Six CNA-only window routes (`..._get_is_borderless_ext`,
`..._set_is_borderless_ext`, `..._minimize_ext`, `..._restore_ext`,
`..._get_native_window_ext`, `..._copy_type_name`) have no XNA member and are
outside the strict projection.

## Two identity spaces that both start at zero

`CNA_GameWindowEvent` numbers `CLIENT_SIZE_CHANGED = 0`,
`ORIENTATION_CHANGED = 1`, `SCREEN_DEVICE_NAME_CHANGED = 2`. `CNA_GameEvent`
numbers `ACTIVATED = 0` … `EXITING = 3`. **A value from either family is a
valid-looking value in the other.**

So the two are kept apart everywhere they could be confused: separate trampoline
tables in `bridge.c`, separate exported Go entry points (`cnaGoGameEvent` and
`cnaGoGameWindowEvent`), separate `Callbacks` members, separate delivery
counters, separate registration slots, and separate release loops — the last
because the tables have different **lengths**, three against four, and releasing
a three-slot array with the four-slot loop would read past it. `probe.c`
asserts that the two counts differ, so a future edit that made them equal is a
compile error rather than a silent overrun.

## ABI

```text                                        before      after
BOUND_FUNCTIONS                               31      41
PROTOTYPE_TYPE_POSITIONS                      91     128
LAYOUTS                                      112     117
MANIFEST_LAYOUT_AGREEMENTS                   112     117
CONSTANTS (canonical static asserts)          27      37
MANIFEST_SIDE_ASSERTIONS (bridge.c)           16      21
CALLBACKS                                      3       3
SYMBOL_IDENTITY_VERIFIED                    true    true
MISSING_HEADER_SYMBOLS                         0       0
MISSING_LIBRARY_SYMBOLS                        0       0
ABI_MISMATCHES / FINDINGS                      0       0
native ABI mutation controls                  48      59
```

The eleven new mutation controls are the window family's own: a drifted window
event identity, a drifted bridge mirror, a window-event count that silently
matches the game count, two missing required symbols, a subscribe prototype with
its callback and context swapped, a title route taking a bare pointer instead of
a `CNA_StringView`, a client size narrowed to 16 bits, `ClientBounds` writing a
`CNA_Viewport` where a `CNA_Rectangle` belongs, a native handle narrowed to 32
bits, and a copy route that dropped its capacity argument. Every one compiles
against the manifest alone and fails only against the canonical header, which is
what the probe translation unit exists for.

## Scoreboard

```text                                      before    after
TARGET_TYPES                                 122      123
TARGET_MEMBERS                              1805     1831
COMPLETE_TYPES                               117      118
PARTIAL_TYPES                                  5        5
MISSING_TYPE                                 135      134
MISSING_MEMBER                               146      145
TOTAL_DIAGNOSTICS                            281      279

behavior corpus                              647      655
external canary tests                         64       69
native stress scenarios                        7        8
native ABI mutation controls                  48       59
runtime capability rows                       43       45
```

`GameWindow` is **complete**, not partial. Every mismatch, leak, allowlist and
unmeasured counter stays zero.

## Two verifier defects found and fixed

**`addCounters` was a hand-written list.** `native_stress` summed each child's
counters field by field, one line per counter, so a counter added to the struct
but not to the list stayed at zero in the aggregate report while its scenario ran
perfectly. Foundation 45 hit exactly that: the whole window scenario reported
zeros on its first run. It is now a reflection loop over every field, with a
panic if any counter is ever not an `int`. This class of defect cannot recur.

**`managedStoredMemberIdentities` had already drifted.** `mapping-rules.json` is
hashed into every report and is the record of what the binding claims its rules
are; it documented **one** managed-stored member while the executable table held
**seven**. Since that table *lowers* fallibility — every entry removes an error
result a type's classification would otherwise add — a reader of the report could
not have discovered the other six. The list is now complete in
`owner|key` form and a new test compares it with the table in both directions;
removing one entry from either side fails it.

## What this milestone does not claim

- **No window signal has ever been delivered.** The HEADLESS artifact has no
  window manager, so it never resizes, rotates or changes screen, and all three
  delivery counters are zero. The subscription lifetime, the routing, the raise
  paths and the handler-failure contract are proved without one; a real
  delivery is `NOT_RUN_ENVIRONMENT` and is recorded as such, exactly as
  `GAME_EVENT_DEACTIVATED_DELIVERIES` is.
- **`ClientBounds` measures 0x0 under HEADLESS** while the same run's graphics
  device reports an 800x480 viewport. A headless window has no client area, so
  the positive-size observation is counted rather than required. Asserting 0x0
  would bake a headless artifact's answer into the contract; asserting a
  positive size would make the scenario fail for a reason the binding cannot fix.
- **The `ArgumentNullException` branch of `set_Title` is unreachable from Go.**
  `string` is not nullable, so no caller can supply the value the reference
  rejects. CNA-Go does not invent a `*string` parameter for a position the XNA
  contract declares as `System.String`.
- **`Game.Window` still has no native window before `Run`.** Making one exist at
  construction time is the `Tick`/`RunOneFrame` lifecycle question and is not
  answered here.
