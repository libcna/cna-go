// SPDX-License-Identifier: MS-PL

#include <stddef.h>
#include <stdio.h>

#include <CNA/C/cna.h>
#include "../../../internal/interop/abi_manifest.h"
#include "../../../internal/interop/bridge.h"

#ifndef CNA_GO_LAYOUT_ONLY
#define CHECK_PROTOTYPE(name) static name##_fn checked_##name = &name;
CNA_GO_REQUIRED_SYMBOLS(CHECK_PROTOTYPE)
#undef CHECK_PROTOTYPE
#endif

// The ADMISSION POLICY, checked against the canonical headers the probe was
// pointed at rather than against one frozen encoded number. CNA-Go admits the
// qualified major and any minor at or above the qualified floor, which is what
// CNA's own version-script comment describes: the symbol-version node changes
// only for a major break, and minor releases are additive.
_Static_assert(CNA_ABI_VERSION_MAJOR == CNA_GO_ABI_MAJOR,
               "canonical CNA ABI major is outside CNA-Go's admission policy");
_Static_assert(CNA_ABI_VERSION_MINOR >= CNA_GO_ABI_MINIMUM_MINOR,
               "canonical CNA ABI minor is below CNA-Go's qualified floor");

// CNA-Go mirrors CNA_ABI_VERSION_ENCODE so the loader can decode a version
// without a CNA header. This is the only translation unit that can see both
// spellings, so it is where the mirror is proven rather than trusted.
_Static_assert(CNA_GO_ABI_ENCODE(0, 21, 0) == CNA_ABI_VERSION_ENCODE(0, 21, 0),
               "encoded-version mirror drift at the qualified version");
_Static_assert(CNA_GO_ABI_ENCODE(1, 2, 3) == CNA_ABI_VERSION_ENCODE(1, 2, 3),
               "encoded-version mirror drift at a mixed sample");
_Static_assert(CNA_GO_ABI_ENCODE(0, 255, 255) == CNA_ABI_VERSION_ENCODE(0, 255, 255),
               "encoded-version mirror drift at the field maxima");
_Static_assert(CNA_GO_ABI_QUALIFIED_VERSION == CNA_ABI_VERSION_ENCODE(0, 21, 0),
               "the qualified encoded constant must be CNA's own encoding of 0.21.0");

_Static_assert(CNA_RESULT_SUCCESS == 0, "CNA_RESULT_SUCCESS drift");
_Static_assert(CNA_RESULT_CALLBACK == 9, "CNA_RESULT_CALLBACK drift");
_Static_assert(CNA_FALSE == 0 && CNA_TRUE == 1, "CNA_Bool constants drift");
_Static_assert(CNA_INVALID_HANDLE == 0, "invalid handle drift");

// bridge.h's own private result mirrors, compared with the canonical values in
// the one translation unit that sees both. bridge.c uses CNA_GO_RESULT_* and is
// compiled by cgo without any CNA header, so before this the two were only ever
// asserted against literals that happened to match.
_Static_assert(CNA_GO_RESULT_SUCCESS == CNA_RESULT_SUCCESS, "bridge result-success mirror drift");
_Static_assert(CNA_GO_RESULT_CALLBACK == CNA_RESULT_CALLBACK, "bridge result-callback mirror drift");

// bridge.h's game-event mirror against the canonical identities. bridge.c
// already compares it with the manifest; this closes the third side, so the
// canonical header, the manifest and the bridge cannot pairwise agree while all
// three drift together.
_Static_assert(CNA_GO_GAME_EVENT_ACTIVATED == CNA_GAME_EVENT_ACTIVATED, "bridge activation mirror drift");
_Static_assert(CNA_GO_GAME_EVENT_DEACTIVATED == CNA_GAME_EVENT_DEACTIVATED, "bridge deactivation mirror drift");
_Static_assert(CNA_GO_GAME_EVENT_DISPOSED == CNA_GAME_EVENT_DISPOSED, "bridge disposal mirror drift");
_Static_assert(CNA_GO_GAME_EVENT_EXITING == CNA_GAME_EVENT_EXITING, "bridge exit mirror drift");
_Static_assert(CNA_GO_GAME_EVENT_COUNT == CNA_GAME_EVENT_MAXIMUM + 1, "bridge game-event count drift");

// The four canonical game-event identities, compared against CNA-Go's own
// private copy in abi_manifest.h. The manifest's copy is what the cgo build
// uses, because that build never sees a CNA header; these five assertions are
// the only place the two are compiled together, so they are what stops a
// signal from being routed to the wrong projected event.
_Static_assert(CNA_GAME_EVENT_ACTIVATED == CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED, "activation identity drift");
_Static_assert(CNA_GAME_EVENT_DEACTIVATED == CNA_GO_MANIFEST_GAME_EVENT_DEACTIVATED, "deactivation identity drift");
_Static_assert(CNA_GAME_EVENT_DISPOSED == CNA_GO_MANIFEST_GAME_EVENT_DISPOSED, "disposal identity drift");
_Static_assert(CNA_GAME_EVENT_EXITING == CNA_GO_MANIFEST_GAME_EVENT_EXITING, "exit identity drift");
_Static_assert(CNA_GAME_EVENT_MAXIMUM == CNA_GAME_EVENT_EXITING, "highest game-event identity drift");

// The frame-hook table's MEMBER ORDER, pinned portably rather than by byte
// offsets. CNA-Go assigns four of the five members conditionally, so a table
// whose members drifted apart between the canonical header and CNA-Go's
// private manifest would install begin_draw where end_run belongs -- and a
// function pointer written to the wrong slot is invisible until a frame runs.
//
// The same five assertions appear in bridge.c, which is compiled against the
// manifest instead of the canonical header. Together they pin both sides: this
// translation unit fails if the canonical table changes, and that one fails if
// the manifest does.
_Static_assert(offsetof(CNA_GameFrameHooks, begin_run) ==
                   offsetof(CNA_GameFrameHooks, initialize) + sizeof(CNA_GameLifecycleCallback),
               "CNA_GameFrameHooks::begin_run must follow initialize");
_Static_assert(offsetof(CNA_GameFrameHooks, end_run) ==
                   offsetof(CNA_GameFrameHooks, begin_run) + sizeof(CNA_GameLifecycleCallback),
               "CNA_GameFrameHooks::end_run must follow begin_run");
_Static_assert(offsetof(CNA_GameFrameHooks, begin_draw) ==
                   offsetof(CNA_GameFrameHooks, end_run) + sizeof(CNA_GameLifecycleCallback),
               "CNA_GameFrameHooks::begin_draw must follow end_run");
_Static_assert(offsetof(CNA_GameFrameHooks, end_draw) ==
                   offsetof(CNA_GameFrameHooks, begin_draw) + sizeof(CNA_GameBeginDrawCallback),
               "CNA_GameFrameHooks::end_draw must follow begin_draw");
_Static_assert(offsetof(CNA_GameFrameHooks, context) ==
                   offsetof(CNA_GameFrameHooks, end_draw) + sizeof(CNA_GameLifecycleCallback),
               "CNA_GameFrameHooks::context must follow end_draw");

#ifndef CNA_GO_LAYOUT_ONLY
// The event callback ABI, pinned the same way every bound function prototype
// is: by assigning a function of the exact shape to the canonical typedef
// under -Werror=incompatible-pointer-types. A handler that returned a result,
// took a game handle, or dropped the context would not compile.
static void cna_go_probe_game_event(void* context) { (void)context; }
static CNA_GameEventCallback checked_CNA_GameEventCallback = &cna_go_probe_game_event;

// The two frame-hook callback ABIs, pinned the same way. CNA-Go installs both:
// the lifecycle shape backs initialize, begin_run, end_run and end_draw, and
// the begin_draw shape is the only one carrying a value channel of its own.
// The CNA_Bool out-parameter and its POSITION -- before the error, after the
// context -- are what decides which frames draw, so a probe that wrote to the
// wrong slot would silently make every refusal ineffective.
static CNA_Result cna_go_probe_lifecycle(
    CNA_Handle game,
    const CNA_GameTime* game_time,
    void* context,
    CNA_CallbackError* out_error) {
    (void)game; (void)game_time; (void)context; (void)out_error;
    return CNA_RESULT_SUCCESS;
}
static CNA_GameLifecycleCallback checked_CNA_GameLifecycleCallback = &cna_go_probe_lifecycle;

static CNA_Result cna_go_probe_begin_draw(
    CNA_Handle game,
    const CNA_GameTime* game_time,
    void* context,
    CNA_Bool* out_should_draw,
    CNA_CallbackError* out_error) {
    (void)game; (void)game_time; (void)context; (void)out_error;
    if (out_should_draw != NULL) {
        *out_should_draw = CNA_TRUE;
    }
    return CNA_RESULT_SUCCESS;
}
static CNA_GameBeginDrawCallback checked_CNA_GameBeginDrawCallback = &cna_go_probe_begin_draw;
#endif

int main(void) {
#ifndef CNA_GO_LAYOUT_ONLY
#define USE_PROTOTYPE(name) (void)checked_##name;
    CNA_GO_REQUIRED_SYMBOLS(USE_PROTOTYPE)
#undef USE_PROTOTYPE
    (void)checked_CNA_GameEventCallback;
    (void)checked_CNA_GameLifecycleCallback;
    (void)checked_CNA_GameBeginDrawCallback;
#endif
    printf("abi_version=%u\n", (unsigned)CNA_ABI_VERSION);
    printf("abi_major=%u\n", (unsigned)CNA_ABI_VERSION_MAJOR);
    printf("abi_minor=%u\n", (unsigned)CNA_ABI_VERSION_MINOR);
    printf("abi_patch=%u\n", (unsigned)CNA_ABI_VERSION_PATCH);
#define CNA_GO_MEASURE(key, expression) printf(#key "=%zu\n", (size_t)(expression));
#include "measurements.inc"
#undef CNA_GO_MEASURE
    return 0;
}
