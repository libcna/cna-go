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

int main(void) {
#ifndef CNA_GO_LAYOUT_ONLY
#define USE_PROTOTYPE(name) (void)checked_##name;
    CNA_GO_REQUIRED_SYMBOLS(USE_PROTOTYPE)
#undef USE_PROTOTYPE
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
    printf("sizeof_CNA_KeyboardState=%zu\n", sizeof(CNA_KeyboardState));
    printf("alignof_CNA_KeyboardState=%zu\n", _Alignof(CNA_KeyboardState));
    printf("offsetof_CNA_KeyboardState_pressed_key_words=%zu\n", offsetof(CNA_KeyboardState, pressed_key_words));
    return 0;
}
