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

/* The three canonical GAME WINDOW event identities, kept the same way and for
   the same reason: they are a second, independent numbering that indexes a
   second trampoline table, and a signal routed to the wrong projected event
   would be invisible. */
#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED UINT32_C(0)
#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED UINT32_C(1)
#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED UINT32_C(2)

/* The five canonical GRAPHICS DEVICE MANAGER event identities. A third
   numbering, and this one does NOT start its device events at zero: DISPOSED
   is 0 and DEVICE_CREATED is 1, so a table indexed as if it matched either of
   the other two families would be off by one. */
#define CNA_GO_MANIFEST_GDM_EVENT_DISPOSED UINT32_C(0)
#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_CREATED UINT32_C(1)
#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_DISPOSING UINT32_C(2)
#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESET UINT32_C(3)
#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESETTING UINT32_C(4)

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

#ifndef CNA_C_RUNTIME_WINDOW_H
typedef uint32_t CNA_GameWindowEvent;
#define CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED CNA_GO_MANIFEST_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED
#define CNA_GAME_WINDOW_EVENT_ORIENTATION_CHANGED CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED
#define CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED CNA_GO_MANIFEST_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED
#define CNA_GAME_WINDOW_EVENT_MAXIMUM CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED
#endif

#ifndef CNA_C_RUNTIME_GRAPHICS_MANAGER_H
typedef CNA_Handle CNA_GraphicsDeviceManagerHandle;
typedef uint32_t CNA_GraphicsDeviceManagerEvent;
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DISPOSED CNA_GO_MANIFEST_GDM_EVENT_DISPOSED
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_CREATED CNA_GO_MANIFEST_GDM_EVENT_DEVICE_CREATED
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_DISPOSING CNA_GO_MANIFEST_GDM_EVENT_DEVICE_DISPOSING
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESET CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESET
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESETTING CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESETTING
#define CNA_GRAPHICS_DEVICE_MANAGER_EVENT_MAXIMUM CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESETTING
#endif

#ifndef CNA_C_GRAPHICS_DEVICE_H
/* Three fixed-width identity aliases, each a uint32_t in the canonical header.
   They are declared here rather than spelled uint32_t at every use so a
   prototype names the same identity CNA names, and so the probe compares an
   alias with an alias. */
typedef uint32_t CNA_ClearOptions;
typedef uint32_t CNA_GraphicsDeviceStatus;
#endif

#ifndef CNA_C_GRAPHICS_H
typedef uint32_t CNA_SurfaceFormat;
#endif

#ifndef CNA_C_DISPLAY_H
typedef uint32_t CNA_GraphicsProfile;
typedef struct CNA_DisplayMode {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t width;
    int32_t height;
    float aspect_ratio;
    CNA_SurfaceFormat format;
} CNA_DisplayMode;
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
typedef struct CNA_Texture2DCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    CNA_Bool mip_map;
    uint8_t reserved[3];
    CNA_SurfaceFormat format;
} CNA_Texture2DCreateInfo;
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
typedef struct CNA_SpriteCommand {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Handle texture;
    CNA_Rectangle destination;
    CNA_Rectangle source;
    CNA_Color color;
    float rotation;
    CNA_Vector2 origin;
    uint32_t effects;
    float layer_depth;
} CNA_SpriteCommand;
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
typedef uint32_t CNA_TextureImageFormat;
typedef uint32_t CNA_TextureDataType;
#endif

/* The packed-storage aliases the Graphics package's element mapping depends on.
   Each is a plain unsigned integer in the canonical header, and CNA-Go passes
   Go structs of the same width straight through, so their sizes are measured
   here rather than assumed. They live under the MATH guard because that is the
   header the canonical ones are declared in. */
#ifndef CNA_C_MATH_VALUES_H
typedef uint8_t CNA_PackedAlpha8;
typedef uint16_t CNA_PackedBgr565;
typedef uint64_t CNA_PackedRgba64;
typedef uint64_t CNA_PackedHalfVector4;
#endif

#ifndef CNA_C_TEXTURE_H
typedef struct CNA_Texture2DTransfer {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t level;
    CNA_Bool has_rectangle;
    uint8_t reserved[3];
    CNA_Rectangle rectangle;
    uint64_t start_index;
    uint64_t element_count;
} CNA_Texture2DTransfer;
typedef struct CNA_Texture2DDecodeInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    CNA_Bool zoom;
    uint8_t reserved[7];
} CNA_Texture2DDecodeInfo;
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
typedef CNA_Result (*cna_game_set_is_mouse_visible_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_game_set_is_fixed_time_step_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_game_set_target_elapsed_time_ticks_fn)(CNA_Handle, int64_t);
typedef CNA_Result (*cna_game_set_inactive_sleep_time_ticks_fn)(CNA_Handle, int64_t);
typedef CNA_Result (*cna_game_reset_elapsed_time_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_suppress_draw_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_tick_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_run_one_frame_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);
typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);
typedef CNA_Result (*cna_graphics_device_manager_create_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_manager_get_graphics_device_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_manager_destroy_fn)(CNA_Handle);

/* GraphicsDeviceManager's configuration surface.
   Only the SETTERS are bound. Every one of the reference's getters is a single
   `ldfld` over a managed field, so CNA-Go reads its own copy for the same
   reason the Game timing getters do: a native getter would be a second source
   of truth that could disagree with the field the setter wrote. The setters
   ARE bound, because the value has to reach the loop that applies it --
   cna_graphics_device_manager_apply_changes reads CNA's copy, not Go's. */
typedef CNA_Result (*cna_graphics_device_manager_set_graphics_profile_fn)(CNA_Handle, uint32_t);
typedef CNA_Result (*cna_graphics_device_manager_set_is_full_screen_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_graphics_device_manager_set_prefer_multi_sampling_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_format_fn)(CNA_Handle, uint32_t);
typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_width_fn)(CNA_Handle, int32_t);
typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_height_fn)(CNA_Handle, int32_t);
typedef CNA_Result (*cna_graphics_device_manager_set_preferred_depth_stencil_format_fn)(CNA_Handle, uint32_t);
typedef CNA_Result (*cna_graphics_device_manager_set_synchronize_with_vertical_retrace_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_graphics_device_manager_set_supported_orientations_fn)(CNA_Handle, uint32_t);
typedef CNA_Result (*cna_graphics_device_manager_apply_changes_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_manager_subscribe_fn)(CNA_Handle, CNA_GraphicsDeviceManagerEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);
typedef CNA_Result (*cna_graphics_device_manager_create_device_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_manager_begin_draw_fn)(CNA_Handle, CNA_Bool*);
typedef CNA_Result (*cna_graphics_device_manager_end_draw_fn)(CNA_Handle);
typedef CNA_Result (*cna_game_get_graphics_device_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_get_viewport_fn)(CNA_Handle, CNA_Viewport*);
typedef CNA_Result (*cna_graphics_device_clear_rgba_fn)(CNA_Handle, float, float, float, float);
typedef CNA_Result (*cna_texture2d_create_from_encoded_memory_fn)(CNA_Handle, const uint8_t*, uint64_t, const CNA_Texture2DDecodeInfo*, CNA_Handle*);
typedef CNA_Result (*cna_texture2d_get_info_fn)(CNA_Handle, CNA_Texture2DInfo*);
typedef CNA_Result (*cna_texture2d_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_sprite_batch_create_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_sprite_batch_begin_fn)(CNA_Handle, const CNA_SpriteBatchBeginInfo*);
typedef CNA_Result (*cna_sprite_batch_submit_scaled_many_fn)(CNA_Handle, const CNA_SpriteScaledCommand*, uint64_t);
typedef CNA_Result (*cna_sprite_batch_submit_many_fn)(CNA_Handle, const CNA_SpriteCommand*, uint64_t);
typedef CNA_Result (*cna_graphics_device_get_blend_factor_fn)(CNA_Handle, CNA_Color*);
typedef CNA_Result (*cna_graphics_device_set_blend_factor_fn)(CNA_Handle, CNA_Color);
typedef CNA_Result (*cna_graphics_device_get_multi_sample_mask_fn)(CNA_Handle, int32_t*);
typedef CNA_Result (*cna_graphics_device_set_multi_sample_mask_fn)(CNA_Handle, int32_t);
typedef CNA_Result (*cna_graphics_device_get_reference_stencil_fn)(CNA_Handle, int32_t*);
typedef CNA_Result (*cna_graphics_device_set_reference_stencil_fn)(CNA_Handle, int32_t);
typedef CNA_Result (*cna_graphics_device_get_scissor_rectangle_fn)(CNA_Handle, CNA_Rectangle*);
typedef CNA_Result (*cna_graphics_device_set_scissor_rectangle_fn)(CNA_Handle, CNA_Rectangle);
typedef CNA_Result (*cna_graphics_device_set_viewport_fn)(CNA_Handle, CNA_Viewport);
typedef CNA_Result (*cna_graphics_device_get_graphics_profile_fn)(CNA_Handle, CNA_GraphicsProfile*);
typedef CNA_Result (*cna_graphics_device_get_status_fn)(CNA_Handle, CNA_GraphicsDeviceStatus*);
typedef CNA_Result (*cna_graphics_device_get_is_disposed_fn)(CNA_Handle, CNA_Bool*);
typedef CNA_Result (*cna_graphics_device_clear_options_fn)(CNA_Handle, CNA_ClearOptions, CNA_Color, float, int32_t);
typedef CNA_Result (*cna_graphics_device_present_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_get_display_mode_fn)(CNA_Handle, CNA_DisplayMode*);
typedef CNA_Result (*cna_texture2d_create_fn)(CNA_Handle, const CNA_Texture2DCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_texture2d_get_encoded_byte_count_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, uint64_t*);
typedef CNA_Result (*cna_texture2d_copy_encoded_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, uint8_t*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_texture2d_set_data_fn)(CNA_Handle, CNA_TextureDataType, const CNA_Texture2DTransfer*, const void*, uint64_t);
typedef CNA_Result (*cna_texture2d_get_data_fn)(CNA_Handle, CNA_TextureDataType, const CNA_Texture2DTransfer*, void*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_sprite_batch_end_fn)(CNA_Handle);
typedef CNA_Result (*cna_sprite_batch_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_keyboard_get_state_fn)(CNA_Handle, CNA_KeyboardState*);

/* GameWindow. Every route takes the GAME handle: CNA models the window as a
   property of the game rather than as a separate object, so there is no window
   handle to own and nothing here is a new lifetime.

   Three canonical window routes are deliberately NOT bound, and each omission
   is a measurement rather than an oversight:

     cna_game_window_get_title_size / cna_game_window_copy_title
         GameWindow::get_Title is one ldfld over the abstract base's own
         managed field. Binding the native getter would create a second source
         of truth that could disagree with the field the setter wrote.
     cna_game_window_get_current_orientation
         WindowsGameWindow::get_CurrentOrientation is `ldc.i4.0; ret`. The
         reference never asks the platform in this profile, so neither does
         CNA-Go. */
typedef CNA_Result (*cna_game_window_get_allow_user_resizing_fn)(CNA_Handle, CNA_Bool*);
typedef CNA_Result (*cna_game_window_set_allow_user_resizing_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_game_window_get_client_bounds_fn)(CNA_Handle, CNA_Rectangle*);
typedef CNA_Result (*cna_game_window_get_native_handle_ext_fn)(CNA_Handle, uint64_t*);
typedef CNA_Result (*cna_game_window_get_screen_device_name_size_fn)(CNA_Handle, uint64_t*);
typedef CNA_Result (*cna_game_window_copy_screen_device_name_fn)(CNA_Handle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_game_window_begin_screen_device_change_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_game_window_end_screen_device_change_fn)(CNA_Handle, CNA_StringView, int32_t, int32_t);
typedef CNA_Result (*cna_game_set_window_title_fn)(CNA_Handle, CNA_StringView);
typedef CNA_Result (*cna_game_window_subscribe_fn)(CNA_Handle, CNA_GameWindowEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);

#define CNA_GO_REQUIRED_SYMBOLS(X) \
    X(cna_get_abi_version) \
    X(cna_error_get_last_message_size) \
    X(cna_error_copy_last_message) \
    X(cna_game_create) \
    X(cna_game_set_frame_hooks_ext) \
    X(cna_game_run) \
    X(cna_game_request_exit) \
    X(cna_game_destroy) \
    X(cna_game_set_is_mouse_visible) \
    X(cna_game_set_is_fixed_time_step) \
    X(cna_game_set_target_elapsed_time_ticks) \
    X(cna_game_set_inactive_sleep_time_ticks) \
    X(cna_game_reset_elapsed_time) \
    X(cna_game_suppress_draw) \
    X(cna_game_tick) \
    X(cna_game_run_one_frame) \
    X(cna_game_subscribe) \
    X(cna_game_unsubscribe) \
    X(cna_graphics_device_manager_create) \
    X(cna_graphics_device_manager_get_graphics_device) \
    X(cna_graphics_device_manager_destroy) \
    X(cna_graphics_device_manager_set_graphics_profile) \
    X(cna_graphics_device_manager_set_is_full_screen) \
    X(cna_graphics_device_manager_set_prefer_multi_sampling) \
    X(cna_graphics_device_manager_set_preferred_back_buffer_format) \
    X(cna_graphics_device_manager_set_preferred_back_buffer_width) \
    X(cna_graphics_device_manager_set_preferred_back_buffer_height) \
    X(cna_graphics_device_manager_set_preferred_depth_stencil_format) \
    X(cna_graphics_device_manager_set_synchronize_with_vertical_retrace) \
    X(cna_graphics_device_manager_set_supported_orientations) \
    X(cna_graphics_device_manager_apply_changes) \
    X(cna_graphics_device_manager_subscribe) \
    X(cna_graphics_device_manager_create_device) \
    X(cna_graphics_device_manager_begin_draw) \
    X(cna_graphics_device_manager_end_draw) \
    X(cna_game_get_graphics_device) \
    X(cna_graphics_device_get_viewport) \
    X(cna_graphics_device_clear_rgba) \
    X(cna_texture2d_create_from_encoded_memory) \
    X(cna_texture2d_get_info) \
    X(cna_texture2d_destroy) \
    X(cna_sprite_batch_create) \
    X(cna_sprite_batch_begin) \
    X(cna_sprite_batch_submit_scaled_many) \
    X(cna_sprite_batch_submit_many) \
    X(cna_graphics_device_get_blend_factor) \
    X(cna_graphics_device_set_blend_factor) \
    X(cna_graphics_device_get_multi_sample_mask) \
    X(cna_graphics_device_set_multi_sample_mask) \
    X(cna_graphics_device_get_reference_stencil) \
    X(cna_graphics_device_set_reference_stencil) \
    X(cna_graphics_device_get_scissor_rectangle) \
    X(cna_graphics_device_set_scissor_rectangle) \
    X(cna_graphics_device_set_viewport) \
    X(cna_graphics_device_get_graphics_profile) \
    X(cna_graphics_device_get_status) \
    X(cna_graphics_device_get_is_disposed) \
    X(cna_graphics_device_clear_options) \
    X(cna_graphics_device_present) \
    X(cna_graphics_device_get_display_mode) \
    X(cna_texture2d_create) \
    X(cna_texture2d_get_encoded_byte_count) \
    X(cna_texture2d_copy_encoded) \
    X(cna_texture2d_set_data) \
    X(cna_texture2d_get_data) \
    X(cna_sprite_batch_end) \
    X(cna_sprite_batch_destroy) \
    X(cna_keyboard_get_state) \
    X(cna_game_window_get_allow_user_resizing) \
    X(cna_game_window_set_allow_user_resizing) \
    X(cna_game_window_get_client_bounds) \
    X(cna_game_window_get_native_handle_ext) \
    X(cna_game_window_get_screen_device_name_size) \
    X(cna_game_window_copy_screen_device_name) \
    X(cna_game_window_begin_screen_device_change) \
    X(cna_game_window_end_screen_device_change) \
    X(cna_game_set_window_title) \
    X(cna_game_window_subscribe)

#endif
