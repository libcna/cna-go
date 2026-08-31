// SPDX-License-Identifier: MS-PL

#include <stddef.h>
#include <stdio.h>

#include <CNA/C/cna.h>
#include "../../../internal/interop/abi_manifest.h"

#ifndef CNA_GO_LAYOUT_ONLY
#define CHECK_PROTOTYPE(name) static name##_fn checked_##name = &name;
CNA_GO_REQUIRED_SYMBOLS(CHECK_PROTOTYPE)
#undef CHECK_PROTOTYPE
#endif

_Static_assert(CNA_ABI_VERSION == UINT32_C(0x00000700), "CNA C ABI must be exactly 0.7.0");
_Static_assert(CNA_RESULT_SUCCESS == 0, "CNA_RESULT_SUCCESS drift");
_Static_assert(CNA_RESULT_CALLBACK == 9, "CNA_RESULT_CALLBACK drift");
_Static_assert(CNA_FALSE == 0 && CNA_TRUE == 1, "CNA_Bool constants drift");
_Static_assert(CNA_INVALID_HANDLE == 0, "invalid handle drift");

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
    printf("sizeof_CNA_Bool=%zu\n", sizeof(CNA_Bool));
    printf("sizeof_CNA_Result=%zu\n", sizeof(CNA_Result));
    printf("sizeof_CNA_Handle=%zu\n", sizeof(CNA_Handle));
    printf("sizeof_CNA_GameTime=%zu\n", sizeof(CNA_GameTime));
    printf("alignof_CNA_GameTime=%zu\n", _Alignof(CNA_GameTime));
    printf("offsetof_CNA_GameTime_is_running_slowly=%zu\n", offsetof(CNA_GameTime, is_running_slowly));
    printf("sizeof_CNA_GameCallbacks=%zu\n", sizeof(CNA_GameCallbacks));
    printf("alignof_CNA_GameCallbacks=%zu\n", _Alignof(CNA_GameCallbacks));
    printf("offsetof_CNA_GameCallbacks_context=%zu\n", offsetof(CNA_GameCallbacks, context));
    printf("sizeof_CNA_GameFrameHooks=%zu\n", sizeof(CNA_GameFrameHooks));
    printf("alignof_CNA_GameFrameHooks=%zu\n", _Alignof(CNA_GameFrameHooks));
    printf("offsetof_CNA_GameFrameHooks_context=%zu\n", offsetof(CNA_GameFrameHooks, context));
    // CNA-Go assigns four of the five hook members conditionally, so their
    // POSITIONS are load-bearing: a member order that drifted between the
    // canonical header and CNA-Go's private manifest would install begin_draw
    // where end_run belongs and the mistake would be invisible at run time.
    printf("offsetof_CNA_GameFrameHooks_initialize=%zu\n", offsetof(CNA_GameFrameHooks, initialize));
    printf("offsetof_CNA_GameFrameHooks_begin_run=%zu\n", offsetof(CNA_GameFrameHooks, begin_run));
    printf("offsetof_CNA_GameFrameHooks_end_run=%zu\n", offsetof(CNA_GameFrameHooks, end_run));
    printf("offsetof_CNA_GameFrameHooks_begin_draw=%zu\n", offsetof(CNA_GameFrameHooks, begin_draw));
    printf("offsetof_CNA_GameFrameHooks_end_draw=%zu\n", offsetof(CNA_GameFrameHooks, end_draw));
    printf("sizeof_CNA_GameCreateInfo=%zu\n", sizeof(CNA_GameCreateInfo));
    printf("alignof_CNA_GameCreateInfo=%zu\n", _Alignof(CNA_GameCreateInfo));
    printf("offsetof_CNA_GameCreateInfo_callbacks=%zu\n", offsetof(CNA_GameCreateInfo, callbacks));
    printf("sizeof_CNA_Viewport=%zu\n", sizeof(CNA_Viewport));
    printf("alignof_CNA_Viewport=%zu\n", _Alignof(CNA_Viewport));
    printf("offsetof_CNA_Viewport_min_depth=%zu\n", offsetof(CNA_Viewport, min_depth));
    printf("sizeof_CNA_Texture2DInfo=%zu\n", sizeof(CNA_Texture2DInfo));
    printf("alignof_CNA_Texture2DInfo=%zu\n", _Alignof(CNA_Texture2DInfo));
    printf("offsetof_CNA_Texture2DInfo_width=%zu\n", offsetof(CNA_Texture2DInfo, width));
    printf("sizeof_CNA_SpriteBatchBeginInfo=%zu\n", sizeof(CNA_SpriteBatchBeginInfo));
    printf("sizeof_CNA_SpriteScaledCommand=%zu\n", sizeof(CNA_SpriteScaledCommand));
    printf("alignof_CNA_SpriteScaledCommand=%zu\n", _Alignof(CNA_SpriteScaledCommand));
    printf("offsetof_CNA_SpriteScaledCommand_scale=%zu\n", offsetof(CNA_SpriteScaledCommand, scale));
    printf("sizeof_CNA_GameEvent=%zu\n", sizeof(CNA_GameEvent));
    printf("sizeof_CNA_GameEventRegistrationHandle=%zu\n", sizeof(CNA_GameEventRegistrationHandle));
    printf("sizeof_CNA_GameEventCallback=%zu\n", sizeof(CNA_GameEventCallback));
    printf("sizeof_CNA_KeyboardState=%zu\n", sizeof(CNA_KeyboardState));
    printf("alignof_CNA_KeyboardState=%zu\n", _Alignof(CNA_KeyboardState));
    printf("offsetof_CNA_KeyboardState_pressed_key_words=%zu\n", offsetof(CNA_KeyboardState, pressed_key_words));
    return 0;
}
