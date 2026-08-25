// SPDX-License-Identifier: MS-PL

#ifndef CNA_GO_ABI_MANIFEST_H
#define CNA_GO_ABI_MANIFEST_H

#include <stdint.h>

#ifndef CNA_C_ABI_H
typedef uint8_t CNA_Bool;
typedef uint32_t CNA_Result;
typedef uint64_t CNA_Handle;
#endif

#ifndef CNA_C_CORE_H
typedef struct CNA_StringView { const char* data; uint64_t byte_length; } CNA_StringView;
typedef struct CNA_Vector2 { float x; float y; } CNA_Vector2;
typedef struct CNA_Rectangle { int32_t x; int32_t y; int32_t width; int32_t height; } CNA_Rectangle;
typedef struct CNA_Color { uint8_t r; uint8_t g; uint8_t b; uint8_t a; } CNA_Color;
#endif

/* CNA-Go's own copy of the four canonical game-event identities. These are
   deliberately declared under private names and OUTSIDE the guard below, so a
   translation unit that also has the canonical CNA header can compare the two
   sets rather than silently preferring one. tools/native_abi does exactly that.
   The guarded block further down defines the canonical spellings from these
   when, and only when, the canonical header is absent. */
#define CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED UINT32_C(0)
#define CNA_GO_MANIFEST_GAME_EVENT_DEACTIVATED UINT32_C(1)
#define CNA_GO_MANIFEST_GAME_EVENT_DISPOSED UINT32_C(2)
#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(3)

#ifndef CNA_C_RUNTIME_H
typedef struct CNA_GameTime {
    int64_t total_game_time_ticks;
    int64_t elapsed_game_time_ticks;
    CNA_Bool is_running_slowly;
    uint8_t reserved[7];
} CNA_GameTime;
typedef struct CNA_CallbackError {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_StringView message;
} CNA_CallbackError;
typedef CNA_Result (*CNA_GameLifecycleCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_CallbackError*);
typedef CNA_Result (*CNA_GameBeginDrawCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_Bool*, CNA_CallbackError*);
typedef struct CNA_GameCallbacks {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_GameLifecycleCallback load_content;
    CNA_GameLifecycleCallback update;
    CNA_GameLifecycleCallback draw;
    CNA_GameLifecycleCallback unload_content;
    CNA_GameLifecycleCallback exiting;
    void* context;
} CNA_GameCallbacks;
typedef struct CNA_GameFrameHooks {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_GameLifecycleCallback initialize;
    CNA_GameLifecycleCallback begin_run;
    CNA_GameLifecycleCallback end_run;
    CNA_GameBeginDrawCallback begin_draw;
    CNA_GameLifecycleCallback end_draw;
    void* context;
} CNA_GameFrameHooks;
typedef struct CNA_GameCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool is_fixed_time_step;
    uint8_t reserved[7];
    int64_t target_elapsed_time_ticks;
    CNA_StringView window_title;
    const CNA_GameCallbacks* callbacks;
} CNA_GameCreateInfo;
typedef CNA_Handle CNA_GameEventRegistrationHandle;
typedef uint32_t CNA_GameEvent;
#define CNA_GAME_EVENT_ACTIVATED CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED
#define CNA_GAME_EVENT_DEACTIVATED CNA_GO_MANIFEST_GAME_EVENT_DEACTIVATED
#define CNA_GAME_EVENT_DISPOSED CNA_GO_MANIFEST_GAME_EVENT_DISPOSED
#define CNA_GAME_EVENT_EXITING CNA_GO_MANIFEST_GAME_EVENT_EXITING
#define CNA_GAME_EVENT_MAXIMUM CNA_GAME_EVENT_EXITING
typedef void (*CNA_GameEventCallback)(void*);
#endif

#ifndef CNA_C_GRAPHICS_DEVICE_H
typedef struct CNA_Viewport {
    int32_t x;
    int32_t y;
    int32_t width;
    int32_t height;
    float min_depth;
    float max_depth;
} CNA_Viewport;
#endif

#ifndef CNA_C_GRAPHICS_H
typedef struct CNA_Texture2DInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    uint32_t level_count;
    uint32_t format;
} CNA_Texture2DInfo;
typedef struct CNA_SpriteBatchBeginInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t sort_mode;
    uint32_t reserved;
} CNA_SpriteBatchBeginInfo;
typedef struct CNA_SpriteScaledCommand {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Handle texture;
    CNA_Vector2 position;
    CNA_Rectangle source;
    CNA_Color color;
    float rotation;
    CNA_Vector2 origin;
    CNA_Vector2 scale;
    uint32_t effects;
    float layer_depth;
} CNA_SpriteScaledCommand;
#endif

#ifndef CNA_C_TEXTURE_H
typedef struct CNA_Texture2DDecodeInfo CNA_Texture2DDecodeInfo;
#endif

#ifndef CNA_C_INPUT_H
typedef struct CNA_KeyboardState {
    uint32_t struct_size;
    uint32_t struct_version;
    uint64_t pressed_key_words[4];
} CNA_KeyboardState;
#endif

typedef uint32_t (*cna_get_abi_version_fn)(void);
typedef CNA_Result (*cna_error_get_last_message_size_fn)(uint64_t*);
typedef CNA_Result (*cna_error_copy_last_message_fn)(char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_game_create_fn)(const CNA_GameCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_game_set_frame_hooks_ext_fn)(CNA_Handle, const CNA_GameFrameHooks*);
typedef CNA_Result (*cna_game_run_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_request_exit_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);
typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);
typedef CNA_Result (*cna_graphics_device_manager_create_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_manager_get_graphics_device_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_manager_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_get_graphics_device_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_get_viewport_fn)(CNA_Handle, CNA_Viewport*);
typedef CNA_Result (*cna_graphics_device_clear_rgba_fn)(CNA_Handle, float, float, float, float);
typedef CNA_Result (*cna_texture2d_create_from_encoded_memory_fn)(CNA_Handle, const uint8_t*, uint64_t, const CNA_Texture2DDecodeInfo*, CNA_Handle*);
typedef CNA_Result (*cna_texture2d_get_info_fn)(CNA_Handle, CNA_Texture2DInfo*);
typedef CNA_Result (*cna_texture2d_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_sprite_batch_create_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_sprite_batch_begin_fn)(CNA_Handle, const CNA_SpriteBatchBeginInfo*);
typedef CNA_Result (*cna_sprite_batch_submit_scaled_many_fn)(CNA_Handle, const CNA_SpriteScaledCommand*, uint64_t);
typedef CNA_Result (*cna_sprite_batch_end_fn)(CNA_Handle);
typedef CNA_Result (*cna_sprite_batch_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_keyboard_get_state_fn)(CNA_Handle, CNA_KeyboardState*);

#define CNA_GO_REQUIRED_SYMBOLS(X) \
    X(cna_get_abi_version) \
    X(cna_error_get_last_message_size) \
    X(cna_error_copy_last_message) \
    X(cna_game_create) \
    X(cna_game_set_frame_hooks_ext) \
    X(cna_game_run) \
    X(cna_game_request_exit) \
    X(cna_game_destroy) \
    X(cna_game_subscribe) \
    X(cna_game_unsubscribe) \
    X(cna_graphics_device_manager_create) \
    X(cna_graphics_device_manager_get_graphics_device) \
    X(cna_graphics_device_manager_destroy) \
    X(cna_game_get_graphics_device) \
    X(cna_graphics_device_get_viewport) \
    X(cna_graphics_device_clear_rgba) \
    X(cna_texture2d_create_from_encoded_memory) \
    X(cna_texture2d_get_info) \
    X(cna_texture2d_destroy) \
    X(cna_sprite_batch_create) \
    X(cna_sprite_batch_begin) \
    X(cna_sprite_batch_submit_scaled_many) \
    X(cna_sprite_batch_end) \
    X(cna_sprite_batch_destroy) \
    X(cna_keyboard_get_state)

#endif
