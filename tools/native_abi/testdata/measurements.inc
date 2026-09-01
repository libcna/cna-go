// SPDX-License-Identifier: MS-PL
//
// The one measurement list, printed by BOTH ABI probes: the canonical one,
// which sees CNA's real headers, and the manifest one, which sees only
// CNA-Go's private declarations. cgo compiles bridge.c against the manifest
// and never against a CNA header, so the manifest's own struct definitions are
// what the shipped binding actually uses -- and until both sides were measured
// with the same list, nothing compared them numerically.
//
// CNA_GO_MEASURE(key, expression) is defined by the including translation unit.

CNA_GO_MEASURE(sizeof_CNA_Bool, sizeof(CNA_Bool))
CNA_GO_MEASURE(sizeof_CNA_Result, sizeof(CNA_Result))
CNA_GO_MEASURE(sizeof_CNA_Handle, sizeof(CNA_Handle))

CNA_GO_MEASURE(sizeof_CNA_StringView, sizeof(CNA_StringView))
CNA_GO_MEASURE(alignof_CNA_StringView, _Alignof(CNA_StringView))
CNA_GO_MEASURE(offsetof_CNA_StringView_data, offsetof(CNA_StringView, data))
CNA_GO_MEASURE(offsetof_CNA_StringView_byte_length, offsetof(CNA_StringView, byte_length))

CNA_GO_MEASURE(sizeof_CNA_Vector2, sizeof(CNA_Vector2))
CNA_GO_MEASURE(offsetof_CNA_Vector2_y, offsetof(CNA_Vector2, y))
CNA_GO_MEASURE(sizeof_CNA_Rectangle, sizeof(CNA_Rectangle))
CNA_GO_MEASURE(offsetof_CNA_Rectangle_width, offsetof(CNA_Rectangle, width))
CNA_GO_MEASURE(offsetof_CNA_Rectangle_height, offsetof(CNA_Rectangle, height))
CNA_GO_MEASURE(sizeof_CNA_Color, sizeof(CNA_Color))
CNA_GO_MEASURE(offsetof_CNA_Color_a, offsetof(CNA_Color, a))

CNA_GO_MEASURE(sizeof_CNA_GameTime, sizeof(CNA_GameTime))
CNA_GO_MEASURE(alignof_CNA_GameTime, _Alignof(CNA_GameTime))
CNA_GO_MEASURE(offsetof_CNA_GameTime_total_game_time_ticks, offsetof(CNA_GameTime, total_game_time_ticks))
CNA_GO_MEASURE(offsetof_CNA_GameTime_elapsed_game_time_ticks, offsetof(CNA_GameTime, elapsed_game_time_ticks))
CNA_GO_MEASURE(offsetof_CNA_GameTime_is_running_slowly, offsetof(CNA_GameTime, is_running_slowly))

CNA_GO_MEASURE(sizeof_CNA_CallbackError, sizeof(CNA_CallbackError))
CNA_GO_MEASURE(alignof_CNA_CallbackError, _Alignof(CNA_CallbackError))
CNA_GO_MEASURE(offsetof_CNA_CallbackError_message, offsetof(CNA_CallbackError, message))

CNA_GO_MEASURE(sizeof_CNA_GameCallbacks, sizeof(CNA_GameCallbacks))
CNA_GO_MEASURE(alignof_CNA_GameCallbacks, _Alignof(CNA_GameCallbacks))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_load_content, offsetof(CNA_GameCallbacks, load_content))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_update, offsetof(CNA_GameCallbacks, update))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_draw, offsetof(CNA_GameCallbacks, draw))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_unload_content, offsetof(CNA_GameCallbacks, unload_content))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_exiting, offsetof(CNA_GameCallbacks, exiting))
CNA_GO_MEASURE(offsetof_CNA_GameCallbacks_context, offsetof(CNA_GameCallbacks, context))

CNA_GO_MEASURE(sizeof_CNA_GameFrameHooks, sizeof(CNA_GameFrameHooks))
CNA_GO_MEASURE(alignof_CNA_GameFrameHooks, _Alignof(CNA_GameFrameHooks))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_initialize, offsetof(CNA_GameFrameHooks, initialize))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_begin_run, offsetof(CNA_GameFrameHooks, begin_run))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_end_run, offsetof(CNA_GameFrameHooks, end_run))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_begin_draw, offsetof(CNA_GameFrameHooks, begin_draw))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_end_draw, offsetof(CNA_GameFrameHooks, end_draw))
CNA_GO_MEASURE(offsetof_CNA_GameFrameHooks_context, offsetof(CNA_GameFrameHooks, context))

CNA_GO_MEASURE(sizeof_CNA_GameCreateInfo, sizeof(CNA_GameCreateInfo))
CNA_GO_MEASURE(alignof_CNA_GameCreateInfo, _Alignof(CNA_GameCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_GameCreateInfo_is_fixed_time_step, offsetof(CNA_GameCreateInfo, is_fixed_time_step))
CNA_GO_MEASURE(offsetof_CNA_GameCreateInfo_target_elapsed_time_ticks, offsetof(CNA_GameCreateInfo, target_elapsed_time_ticks))
CNA_GO_MEASURE(offsetof_CNA_GameCreateInfo_window_title, offsetof(CNA_GameCreateInfo, window_title))
CNA_GO_MEASURE(offsetof_CNA_GameCreateInfo_callbacks, offsetof(CNA_GameCreateInfo, callbacks))

CNA_GO_MEASURE(sizeof_CNA_Viewport, sizeof(CNA_Viewport))
CNA_GO_MEASURE(alignof_CNA_Viewport, _Alignof(CNA_Viewport))
CNA_GO_MEASURE(offsetof_CNA_Viewport_width, offsetof(CNA_Viewport, width))
CNA_GO_MEASURE(offsetof_CNA_Viewport_min_depth, offsetof(CNA_Viewport, min_depth))
CNA_GO_MEASURE(offsetof_CNA_Viewport_max_depth, offsetof(CNA_Viewport, max_depth))

CNA_GO_MEASURE(sizeof_CNA_Texture2DInfo, sizeof(CNA_Texture2DInfo))
CNA_GO_MEASURE(alignof_CNA_Texture2DInfo, _Alignof(CNA_Texture2DInfo))
CNA_GO_MEASURE(offsetof_CNA_Texture2DInfo_width, offsetof(CNA_Texture2DInfo, width))
CNA_GO_MEASURE(offsetof_CNA_Texture2DInfo_height, offsetof(CNA_Texture2DInfo, height))
CNA_GO_MEASURE(offsetof_CNA_Texture2DInfo_level_count, offsetof(CNA_Texture2DInfo, level_count))
CNA_GO_MEASURE(offsetof_CNA_Texture2DInfo_format, offsetof(CNA_Texture2DInfo, format))

CNA_GO_MEASURE(sizeof_CNA_SpriteBatchBeginInfo, sizeof(CNA_SpriteBatchBeginInfo))
CNA_GO_MEASURE(offsetof_CNA_SpriteBatchBeginInfo_sort_mode, offsetof(CNA_SpriteBatchBeginInfo, sort_mode))

CNA_GO_MEASURE(sizeof_CNA_Texture2DTransfer, sizeof(CNA_Texture2DTransfer))
CNA_GO_MEASURE(alignof_CNA_Texture2DTransfer, _Alignof(CNA_Texture2DTransfer))
CNA_GO_MEASURE(offsetof_CNA_Texture2DTransfer_level, offsetof(CNA_Texture2DTransfer, level))
CNA_GO_MEASURE(offsetof_CNA_Texture2DTransfer_has_rectangle, offsetof(CNA_Texture2DTransfer, has_rectangle))
CNA_GO_MEASURE(offsetof_CNA_Texture2DTransfer_rectangle, offsetof(CNA_Texture2DTransfer, rectangle))
CNA_GO_MEASURE(offsetof_CNA_Texture2DTransfer_start_index, offsetof(CNA_Texture2DTransfer, start_index))
CNA_GO_MEASURE(offsetof_CNA_Texture2DTransfer_element_count, offsetof(CNA_Texture2DTransfer, element_count))
CNA_GO_MEASURE(sizeof_CNA_TextureDataType, sizeof(CNA_TextureDataType))
CNA_GO_MEASURE(sizeof_CNA_PackedBgr565, sizeof(CNA_PackedBgr565))
CNA_GO_MEASURE(sizeof_CNA_PackedRgba64, sizeof(CNA_PackedRgba64))
CNA_GO_MEASURE(sizeof_CNA_PackedHalfVector4, sizeof(CNA_PackedHalfVector4))
CNA_GO_MEASURE(sizeof_CNA_PackedAlpha8, sizeof(CNA_PackedAlpha8))
CNA_GO_MEASURE(sizeof_CNA_Texture2DDecodeInfo, sizeof(CNA_Texture2DDecodeInfo))
CNA_GO_MEASURE(alignof_CNA_Texture2DDecodeInfo, _Alignof(CNA_Texture2DDecodeInfo))
CNA_GO_MEASURE(offsetof_CNA_Texture2DDecodeInfo_width, offsetof(CNA_Texture2DDecodeInfo, width))
CNA_GO_MEASURE(offsetof_CNA_Texture2DDecodeInfo_height, offsetof(CNA_Texture2DDecodeInfo, height))
CNA_GO_MEASURE(offsetof_CNA_Texture2DDecodeInfo_zoom, offsetof(CNA_Texture2DDecodeInfo, zoom))
CNA_GO_MEASURE(sizeof_CNA_TextureImageFormat, sizeof(CNA_TextureImageFormat))
CNA_GO_MEASURE(sizeof_CNA_DisplayMode, sizeof(CNA_DisplayMode))
CNA_GO_MEASURE(alignof_CNA_DisplayMode, _Alignof(CNA_DisplayMode))
CNA_GO_MEASURE(offsetof_CNA_DisplayMode_width, offsetof(CNA_DisplayMode, width))
CNA_GO_MEASURE(offsetof_CNA_DisplayMode_height, offsetof(CNA_DisplayMode, height))
CNA_GO_MEASURE(offsetof_CNA_DisplayMode_aspect_ratio, offsetof(CNA_DisplayMode, aspect_ratio))
CNA_GO_MEASURE(offsetof_CNA_DisplayMode_format, offsetof(CNA_DisplayMode, format))
CNA_GO_MEASURE(sizeof_CNA_Texture2DCreateInfo, sizeof(CNA_Texture2DCreateInfo))
CNA_GO_MEASURE(alignof_CNA_Texture2DCreateInfo, _Alignof(CNA_Texture2DCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_Texture2DCreateInfo_width, offsetof(CNA_Texture2DCreateInfo, width))
CNA_GO_MEASURE(offsetof_CNA_Texture2DCreateInfo_height, offsetof(CNA_Texture2DCreateInfo, height))
CNA_GO_MEASURE(offsetof_CNA_Texture2DCreateInfo_mip_map, offsetof(CNA_Texture2DCreateInfo, mip_map))
CNA_GO_MEASURE(offsetof_CNA_Texture2DCreateInfo_format, offsetof(CNA_Texture2DCreateInfo, format))
CNA_GO_MEASURE(sizeof_CNA_ClearOptions, sizeof(CNA_ClearOptions))
CNA_GO_MEASURE(sizeof_CNA_GraphicsDeviceStatus, sizeof(CNA_GraphicsDeviceStatus))
CNA_GO_MEASURE(sizeof_CNA_GraphicsProfile, sizeof(CNA_GraphicsProfile))
CNA_GO_MEASURE(sizeof_CNA_SpriteCommand, sizeof(CNA_SpriteCommand))
CNA_GO_MEASURE(alignof_CNA_SpriteCommand, _Alignof(CNA_SpriteCommand))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_texture, offsetof(CNA_SpriteCommand, texture))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_destination, offsetof(CNA_SpriteCommand, destination))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_source, offsetof(CNA_SpriteCommand, source))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_color, offsetof(CNA_SpriteCommand, color))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_rotation, offsetof(CNA_SpriteCommand, rotation))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_origin, offsetof(CNA_SpriteCommand, origin))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_effects, offsetof(CNA_SpriteCommand, effects))
CNA_GO_MEASURE(offsetof_CNA_SpriteCommand_layer_depth, offsetof(CNA_SpriteCommand, layer_depth))
CNA_GO_MEASURE(sizeof_CNA_SpriteScaledCommand, sizeof(CNA_SpriteScaledCommand))
CNA_GO_MEASURE(alignof_CNA_SpriteScaledCommand, _Alignof(CNA_SpriteScaledCommand))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_texture, offsetof(CNA_SpriteScaledCommand, texture))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_position, offsetof(CNA_SpriteScaledCommand, position))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_source, offsetof(CNA_SpriteScaledCommand, source))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_color, offsetof(CNA_SpriteScaledCommand, color))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_rotation, offsetof(CNA_SpriteScaledCommand, rotation))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_origin, offsetof(CNA_SpriteScaledCommand, origin))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_scale, offsetof(CNA_SpriteScaledCommand, scale))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_effects, offsetof(CNA_SpriteScaledCommand, effects))
CNA_GO_MEASURE(offsetof_CNA_SpriteScaledCommand_layer_depth, offsetof(CNA_SpriteScaledCommand, layer_depth))

CNA_GO_MEASURE(sizeof_CNA_KeyboardState, sizeof(CNA_KeyboardState))
CNA_GO_MEASURE(alignof_CNA_KeyboardState, _Alignof(CNA_KeyboardState))
CNA_GO_MEASURE(offsetof_CNA_KeyboardState_pressed_key_words, offsetof(CNA_KeyboardState, pressed_key_words))

CNA_GO_MEASURE(sizeof_CNA_GameEvent, sizeof(CNA_GameEvent))
CNA_GO_MEASURE(sizeof_CNA_GameEventRegistrationHandle, sizeof(CNA_GameEventRegistrationHandle))
CNA_GO_MEASURE(sizeof_CNA_GameEventCallback, sizeof(CNA_GameEventCallback))
CNA_GO_MEASURE(sizeof_CNA_GameLifecycleCallback, sizeof(CNA_GameLifecycleCallback))
CNA_GO_MEASURE(sizeof_CNA_GameBeginDrawCallback, sizeof(CNA_GameBeginDrawCallback))

CNA_GO_MEASURE(constant_CNA_GAME_EVENT_ACTIVATED, CNA_GAME_EVENT_ACTIVATED)
CNA_GO_MEASURE(constant_CNA_GAME_EVENT_DEACTIVATED, CNA_GAME_EVENT_DEACTIVATED)
CNA_GO_MEASURE(constant_CNA_GAME_EVENT_DISPOSED, CNA_GAME_EVENT_DISPOSED)
CNA_GO_MEASURE(constant_CNA_GAME_EVENT_EXITING, CNA_GAME_EVENT_EXITING)
CNA_GO_MEASURE(constant_CNA_GAME_EVENT_MAXIMUM, CNA_GAME_EVENT_MAXIMUM)

CNA_GO_MEASURE(sizeof_CNA_GameWindowEvent, sizeof(CNA_GameWindowEvent))
CNA_GO_MEASURE(constant_CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED, CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED)
CNA_GO_MEASURE(constant_CNA_GAME_WINDOW_EVENT_ORIENTATION_CHANGED, CNA_GAME_WINDOW_EVENT_ORIENTATION_CHANGED)
CNA_GO_MEASURE(constant_CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED, CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED)
CNA_GO_MEASURE(constant_CNA_GAME_WINDOW_EVENT_MAXIMUM, CNA_GAME_WINDOW_EVENT_MAXIMUM)

// FIELD WIDTHS. Offsets and total sizes are not enough on their own: a length
// narrowed from 64 to 32 bits inside a struct whose padding absorbs the change
// leaves every size and every offset identical while the callee reads four
// bytes CNA-Go never wrote. sizeof on a null-pointer member expression is a
// constant expression and dereferences nothing.
#define CNA_GO_FIELD_WIDTH(type, field) sizeof(((type*)0)->field)

CNA_GO_MEASURE(width_CNA_StringView_data, CNA_GO_FIELD_WIDTH(CNA_StringView, data))
CNA_GO_MEASURE(width_CNA_StringView_byte_length, CNA_GO_FIELD_WIDTH(CNA_StringView, byte_length))
CNA_GO_MEASURE(width_CNA_Rectangle_x, CNA_GO_FIELD_WIDTH(CNA_Rectangle, x))
CNA_GO_MEASURE(width_CNA_Color_r, CNA_GO_FIELD_WIDTH(CNA_Color, r))
CNA_GO_MEASURE(width_CNA_Vector2_x, CNA_GO_FIELD_WIDTH(CNA_Vector2, x))
CNA_GO_MEASURE(width_CNA_GameTime_total_game_time_ticks, CNA_GO_FIELD_WIDTH(CNA_GameTime, total_game_time_ticks))
CNA_GO_MEASURE(width_CNA_GameTime_elapsed_game_time_ticks, CNA_GO_FIELD_WIDTH(CNA_GameTime, elapsed_game_time_ticks))
CNA_GO_MEASURE(width_CNA_GameTime_is_running_slowly, CNA_GO_FIELD_WIDTH(CNA_GameTime, is_running_slowly))
CNA_GO_MEASURE(width_CNA_CallbackError_struct_size, CNA_GO_FIELD_WIDTH(CNA_CallbackError, struct_size))
CNA_GO_MEASURE(width_CNA_CallbackError_message, CNA_GO_FIELD_WIDTH(CNA_CallbackError, message))
CNA_GO_MEASURE(width_CNA_GameCallbacks_struct_size, CNA_GO_FIELD_WIDTH(CNA_GameCallbacks, struct_size))
CNA_GO_MEASURE(width_CNA_GameCallbacks_context, CNA_GO_FIELD_WIDTH(CNA_GameCallbacks, context))
CNA_GO_MEASURE(width_CNA_GameFrameHooks_begin_draw, CNA_GO_FIELD_WIDTH(CNA_GameFrameHooks, begin_draw))
CNA_GO_MEASURE(width_CNA_GameCreateInfo_struct_size, CNA_GO_FIELD_WIDTH(CNA_GameCreateInfo, struct_size))
CNA_GO_MEASURE(width_CNA_GameCreateInfo_is_fixed_time_step, CNA_GO_FIELD_WIDTH(CNA_GameCreateInfo, is_fixed_time_step))
CNA_GO_MEASURE(width_CNA_GameCreateInfo_target_elapsed_time_ticks, CNA_GO_FIELD_WIDTH(CNA_GameCreateInfo, target_elapsed_time_ticks))
CNA_GO_MEASURE(width_CNA_GameCreateInfo_window_title, CNA_GO_FIELD_WIDTH(CNA_GameCreateInfo, window_title))
CNA_GO_MEASURE(width_CNA_GameCreateInfo_callbacks, CNA_GO_FIELD_WIDTH(CNA_GameCreateInfo, callbacks))
CNA_GO_MEASURE(width_CNA_Viewport_x, CNA_GO_FIELD_WIDTH(CNA_Viewport, x))
CNA_GO_MEASURE(width_CNA_Viewport_min_depth, CNA_GO_FIELD_WIDTH(CNA_Viewport, min_depth))
CNA_GO_MEASURE(width_CNA_Texture2DInfo_struct_size, CNA_GO_FIELD_WIDTH(CNA_Texture2DInfo, struct_size))
CNA_GO_MEASURE(width_CNA_Texture2DInfo_width, CNA_GO_FIELD_WIDTH(CNA_Texture2DInfo, width))
CNA_GO_MEASURE(width_CNA_Texture2DInfo_format, CNA_GO_FIELD_WIDTH(CNA_Texture2DInfo, format))
CNA_GO_MEASURE(width_CNA_SpriteBatchBeginInfo_sort_mode, CNA_GO_FIELD_WIDTH(CNA_SpriteBatchBeginInfo, sort_mode))
CNA_GO_MEASURE(width_CNA_DisplayMode_aspect_ratio, CNA_GO_FIELD_WIDTH(CNA_DisplayMode, aspect_ratio))
CNA_GO_MEASURE(width_CNA_Texture2DCreateInfo_mip_map, CNA_GO_FIELD_WIDTH(CNA_Texture2DCreateInfo, mip_map))
CNA_GO_MEASURE(width_CNA_SpriteCommand_destination, CNA_GO_FIELD_WIDTH(CNA_SpriteCommand, destination))
CNA_GO_MEASURE(width_CNA_SpriteCommand_layer_depth, CNA_GO_FIELD_WIDTH(CNA_SpriteCommand, layer_depth))
CNA_GO_MEASURE(width_CNA_SpriteScaledCommand_texture, CNA_GO_FIELD_WIDTH(CNA_SpriteScaledCommand, texture))
CNA_GO_MEASURE(width_CNA_SpriteScaledCommand_rotation, CNA_GO_FIELD_WIDTH(CNA_SpriteScaledCommand, rotation))
CNA_GO_MEASURE(width_CNA_SpriteScaledCommand_effects, CNA_GO_FIELD_WIDTH(CNA_SpriteScaledCommand, effects))
CNA_GO_MEASURE(width_CNA_SpriteScaledCommand_layer_depth, CNA_GO_FIELD_WIDTH(CNA_SpriteScaledCommand, layer_depth))
CNA_GO_MEASURE(width_CNA_KeyboardState_struct_version, CNA_GO_FIELD_WIDTH(CNA_KeyboardState, struct_version))
CNA_GO_MEASURE(width_CNA_KeyboardState_pressed_key_words, CNA_GO_FIELD_WIDTH(CNA_KeyboardState, pressed_key_words))
CNA_GO_MEASURE(width_CNA_KeyboardState_pressed_key_word, CNA_GO_FIELD_WIDTH(CNA_KeyboardState, pressed_key_words[0]))

#undef CNA_GO_FIELD_WIDTH

// Foundation 58. The two render-target structures. bridge.c FILLS the create
// info field by field and READS every field of the info structure out, so every
// offset below is load-bearing: a field that moved would be written to or read
// from the wrong bytes with nothing on either side reporting it.
//
// The reserved arrays are measured through their neighbours' offsets rather
// than directly, which is what makes them evidence: `reserved0[3]` sits between
// a CNA_Bool and a CNA_SurfaceFormat, so dropping it would move `format` and
// dropping `reserved1` would change the size.
CNA_GO_MEASURE(sizeof_CNA_RenderTarget2DCreateInfo, sizeof(CNA_RenderTarget2DCreateInfo))
CNA_GO_MEASURE(alignof_CNA_RenderTarget2DCreateInfo, _Alignof(CNA_RenderTarget2DCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_width, offsetof(CNA_RenderTarget2DCreateInfo, width))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_height, offsetof(CNA_RenderTarget2DCreateInfo, height))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_mip_map, offsetof(CNA_RenderTarget2DCreateInfo, mip_map))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_format, offsetof(CNA_RenderTarget2DCreateInfo, format))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_depth_format, offsetof(CNA_RenderTarget2DCreateInfo, depth_format))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_multi_sample_count, offsetof(CNA_RenderTarget2DCreateInfo, multi_sample_count))
CNA_GO_MEASURE(offsetof_CNA_RenderTarget2DCreateInfo_usage, offsetof(CNA_RenderTarget2DCreateInfo, usage))

CNA_GO_MEASURE(sizeof_CNA_RenderTargetInfo, sizeof(CNA_RenderTargetInfo))
CNA_GO_MEASURE(alignof_CNA_RenderTargetInfo, _Alignof(CNA_RenderTargetInfo))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_kind, offsetof(CNA_RenderTargetInfo, kind))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_width, offsetof(CNA_RenderTargetInfo, width))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_height, offsetof(CNA_RenderTargetInfo, height))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_level_count, offsetof(CNA_RenderTargetInfo, level_count))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_format, offsetof(CNA_RenderTargetInfo, format))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_depth_format, offsetof(CNA_RenderTargetInfo, depth_format))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_multi_sample_count, offsetof(CNA_RenderTargetInfo, multi_sample_count))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_usage, offsetof(CNA_RenderTargetInfo, usage))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_is_content_lost, offsetof(CNA_RenderTargetInfo, is_content_lost))
CNA_GO_MEASURE(offsetof_CNA_RenderTargetInfo_renderer_available, offsetof(CNA_RenderTargetInfo, renderer_available))

// Foundation 60. The four state descriptors. bridge.c FILLS every field of each
// one, so every offset is load-bearing in the same way the render target's are.
//
// CNA_BlendState's blend_factor is a CNA_Color -- four bytes inside a run of
// uint32s -- and its offset is what decides whether the constant blend colour
// lands on the colour or on the multisample mask.
//
// The two reserved fields are measured through the structure SIZE rather than
// directly: CNA_DepthStencilState's trailing `reserved` is a full uint32 and
// CNA_RasterizerState's `reserved[2]` sits after two one-byte booleans, where
// the compiler would pad anyway. Only the first is a control; that distinction
// is the settled padding rule and is applied rather than re-argued.
CNA_GO_MEASURE(sizeof_CNA_BlendState, sizeof(CNA_BlendState))
CNA_GO_MEASURE(alignof_CNA_BlendState, _Alignof(CNA_BlendState))
CNA_GO_MEASURE(offsetof_CNA_BlendState_alpha_blend_function, offsetof(CNA_BlendState, alpha_blend_function))
CNA_GO_MEASURE(offsetof_CNA_BlendState_alpha_destination_blend, offsetof(CNA_BlendState, alpha_destination_blend))
CNA_GO_MEASURE(offsetof_CNA_BlendState_alpha_source_blend, offsetof(CNA_BlendState, alpha_source_blend))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_blend_function, offsetof(CNA_BlendState, color_blend_function))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_destination_blend, offsetof(CNA_BlendState, color_destination_blend))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_source_blend, offsetof(CNA_BlendState, color_source_blend))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_write_channels, offsetof(CNA_BlendState, color_write_channels))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_write_channels1, offsetof(CNA_BlendState, color_write_channels1))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_write_channels2, offsetof(CNA_BlendState, color_write_channels2))
CNA_GO_MEASURE(offsetof_CNA_BlendState_color_write_channels3, offsetof(CNA_BlendState, color_write_channels3))
CNA_GO_MEASURE(offsetof_CNA_BlendState_blend_factor, offsetof(CNA_BlendState, blend_factor))
CNA_GO_MEASURE(offsetof_CNA_BlendState_multi_sample_mask, offsetof(CNA_BlendState, multi_sample_mask))

CNA_GO_MEASURE(sizeof_CNA_DepthStencilState, sizeof(CNA_DepthStencilState))
CNA_GO_MEASURE(alignof_CNA_DepthStencilState, _Alignof(CNA_DepthStencilState))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_depth_buffer_enable, offsetof(CNA_DepthStencilState, depth_buffer_enable))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_depth_buffer_write_enable, offsetof(CNA_DepthStencilState, depth_buffer_write_enable))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_enable, offsetof(CNA_DepthStencilState, stencil_enable))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_two_sided_stencil_mode, offsetof(CNA_DepthStencilState, two_sided_stencil_mode))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_depth_buffer_function, offsetof(CNA_DepthStencilState, depth_buffer_function))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_function, offsetof(CNA_DepthStencilState, stencil_function))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_mask, offsetof(CNA_DepthStencilState, stencil_mask))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_write_mask, offsetof(CNA_DepthStencilState, stencil_write_mask))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_reference_stencil, offsetof(CNA_DepthStencilState, reference_stencil))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_fail, offsetof(CNA_DepthStencilState, stencil_fail))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_depth_buffer_fail, offsetof(CNA_DepthStencilState, stencil_depth_buffer_fail))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_stencil_pass, offsetof(CNA_DepthStencilState, stencil_pass))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_ccw_stencil_function, offsetof(CNA_DepthStencilState, counter_clockwise_stencil_function))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_ccw_stencil_fail, offsetof(CNA_DepthStencilState, counter_clockwise_stencil_fail))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_ccw_stencil_depth_buffer_fail, offsetof(CNA_DepthStencilState, counter_clockwise_stencil_depth_buffer_fail))
CNA_GO_MEASURE(offsetof_CNA_DepthStencilState_ccw_stencil_pass, offsetof(CNA_DepthStencilState, counter_clockwise_stencil_pass))

CNA_GO_MEASURE(sizeof_CNA_RasterizerState, sizeof(CNA_RasterizerState))
CNA_GO_MEASURE(alignof_CNA_RasterizerState, _Alignof(CNA_RasterizerState))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_cull_mode, offsetof(CNA_RasterizerState, cull_mode))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_fill_mode, offsetof(CNA_RasterizerState, fill_mode))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_depth_bias, offsetof(CNA_RasterizerState, depth_bias))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_slope_scale_depth_bias, offsetof(CNA_RasterizerState, slope_scale_depth_bias))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_multi_sample_anti_alias, offsetof(CNA_RasterizerState, multi_sample_anti_alias))
CNA_GO_MEASURE(offsetof_CNA_RasterizerState_scissor_test_enable, offsetof(CNA_RasterizerState, scissor_test_enable))

CNA_GO_MEASURE(sizeof_CNA_SamplerState, sizeof(CNA_SamplerState))
CNA_GO_MEASURE(alignof_CNA_SamplerState, _Alignof(CNA_SamplerState))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_address_u, offsetof(CNA_SamplerState, address_u))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_address_v, offsetof(CNA_SamplerState, address_v))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_address_w, offsetof(CNA_SamplerState, address_w))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_filter, offsetof(CNA_SamplerState, filter))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_max_anisotropy, offsetof(CNA_SamplerState, max_anisotropy))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_max_mip_level, offsetof(CNA_SamplerState, max_mip_level))
CNA_GO_MEASURE(offsetof_CNA_SamplerState_mip_map_level_of_detail_bias, offsetof(CNA_SamplerState, mip_map_level_of_detail_bias))

// Foundation 61. CNA_TextureSlotInfo. The `reserved[7]` between a one-byte
// CNA_Bool and a uint64 handle is the one that is NOT a control: the compiler
// pads to eight either way. The handle's offset is, and it decides whether a
// slot's occupant is read from the flag's bytes.
CNA_GO_MEASURE(sizeof_CNA_TextureSlotInfo, sizeof(CNA_TextureSlotInfo))
CNA_GO_MEASURE(alignof_CNA_TextureSlotInfo, _Alignof(CNA_TextureSlotInfo))
CNA_GO_MEASURE(offsetof_CNA_TextureSlotInfo_bound, offsetof(CNA_TextureSlotInfo, bound))
CNA_GO_MEASURE(offsetof_CNA_TextureSlotInfo_texture, offsetof(CNA_TextureSlotInfo, texture))

// Foundation 62. The two device event payload structures. Only the destroyed
// one carries data past its flag, and the StringView's offset is what decides
// whether a resource name is read from the reserved bytes.
CNA_GO_MEASURE(sizeof_CNA_ResourceCreatedEventInfo, sizeof(CNA_ResourceCreatedEventInfo))
CNA_GO_MEASURE(alignof_CNA_ResourceCreatedEventInfo, _Alignof(CNA_ResourceCreatedEventInfo))
CNA_GO_MEASURE(offsetof_CNA_ResourceCreatedEventInfo_has_resource, offsetof(CNA_ResourceCreatedEventInfo, has_resource))
CNA_GO_MEASURE(sizeof_CNA_ResourceDestroyedEventInfo, sizeof(CNA_ResourceDestroyedEventInfo))
CNA_GO_MEASURE(alignof_CNA_ResourceDestroyedEventInfo, _Alignof(CNA_ResourceDestroyedEventInfo))
CNA_GO_MEASURE(offsetof_CNA_ResourceDestroyedEventInfo_has_tag, offsetof(CNA_ResourceDestroyedEventInfo, has_tag))
CNA_GO_MEASURE(offsetof_CNA_ResourceDestroyedEventInfo_name, offsetof(CNA_ResourceDestroyedEventInfo, name))

// Foundation 63. CNA_ContentManagerCreateInfo. The two uint32 header fields sit
// before a CNA_StringView, so the root directory's offset is what decides
// whether CNA reads the root from the struct_size and struct_version bytes --
// a manager that "created successfully" with a garbage root, resolving every
// asset path under it. The trailing `reserved` is measured for its offset
// rather than its existence: a root that moved into it would be read as zero.
CNA_GO_MEASURE(sizeof_CNA_ContentManagerCreateInfo, sizeof(CNA_ContentManagerCreateInfo))
CNA_GO_MEASURE(alignof_CNA_ContentManagerCreateInfo, _Alignof(CNA_ContentManagerCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_ContentManagerCreateInfo_struct_size, offsetof(CNA_ContentManagerCreateInfo, struct_size))
CNA_GO_MEASURE(offsetof_CNA_ContentManagerCreateInfo_struct_version, offsetof(CNA_ContentManagerCreateInfo, struct_version))
CNA_GO_MEASURE(offsetof_CNA_ContentManagerCreateInfo_root_directory, offsetof(CNA_ContentManagerCreateInfo, root_directory))
CNA_GO_MEASURE(offsetof_CNA_ContentManagerCreateInfo_reserved, offsetof(CNA_ContentManagerCreateInfo, reserved))

// Foundation 65. The three index-buffer structures.
//
// The create-info's `index_element_size` and `buffer_usage` are ADJACENT
// uint32s, so swapping them is byte-identical to C and turns a 16-bit read-write
// buffer into a 32-bit write-only one. The info struct repeats the pair and adds
// three CNA_Bools whose offsets decide which flag a caller reads. The transfer's
// `start_index` and `element_count` are adjacent uint64s, the same shape the
// texture transfer's are, and swapping them turns "six indices from zero" into
// "zero indices from six" -- a transfer that succeeds and moves nothing.
CNA_GO_MEASURE(sizeof_CNA_IndexBufferCreateInfo, sizeof(CNA_IndexBufferCreateInfo))
CNA_GO_MEASURE(alignof_CNA_IndexBufferCreateInfo, _Alignof(CNA_IndexBufferCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferCreateInfo_index_count, offsetof(CNA_IndexBufferCreateInfo, index_count))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferCreateInfo_index_element_size, offsetof(CNA_IndexBufferCreateInfo, index_element_size))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferCreateInfo_buffer_usage, offsetof(CNA_IndexBufferCreateInfo, buffer_usage))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferCreateInfo_dynamic, offsetof(CNA_IndexBufferCreateInfo, dynamic))
CNA_GO_MEASURE(sizeof_CNA_IndexBufferInfo, sizeof(CNA_IndexBufferInfo))
CNA_GO_MEASURE(alignof_CNA_IndexBufferInfo, _Alignof(CNA_IndexBufferInfo))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_index_count, offsetof(CNA_IndexBufferInfo, index_count))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_index_element_size, offsetof(CNA_IndexBufferInfo, index_element_size))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_buffer_usage, offsetof(CNA_IndexBufferInfo, buffer_usage))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_dynamic, offsetof(CNA_IndexBufferInfo, dynamic))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_is_content_lost, offsetof(CNA_IndexBufferInfo, is_content_lost))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferInfo_has_renderer, offsetof(CNA_IndexBufferInfo, has_renderer))
CNA_GO_MEASURE(sizeof_CNA_IndexBufferTransfer, sizeof(CNA_IndexBufferTransfer))
CNA_GO_MEASURE(alignof_CNA_IndexBufferTransfer, _Alignof(CNA_IndexBufferTransfer))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferTransfer_index_element_size, offsetof(CNA_IndexBufferTransfer, index_element_size))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferTransfer_options, offsetof(CNA_IndexBufferTransfer, options))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferTransfer_start_index, offsetof(CNA_IndexBufferTransfer, start_index))
CNA_GO_MEASURE(offsetof_CNA_IndexBufferTransfer_element_count, offsetof(CNA_IndexBufferTransfer, element_count))

// Foundation 66. CNA_VertexElement and the two vertex-buffer structures.
//
// CNA_VertexElement is the one structure in this family that crosses as an
// ARRAY, and its four fields are all four bytes wide -- so any permutation of
// them is byte-identical to C and produces a declaration describing a different
// layout. The offset and usage_index are both int32 and adjacent to their enum
// neighbours, which is the swap a C compiler cannot see.
//
// The buffer create-info's `vertex_declaration` is a HANDLE before two int32s;
// the info struct's `vertex_stride` sits after three CNA_Bools, so its offset
// is what decides whether a stride is read out of the flag bytes.
CNA_GO_MEASURE(sizeof_CNA_VertexElement, sizeof(CNA_VertexElement))
CNA_GO_MEASURE(alignof_CNA_VertexElement, _Alignof(CNA_VertexElement))
CNA_GO_MEASURE(offsetof_CNA_VertexElement_offset, offsetof(CNA_VertexElement, offset))
CNA_GO_MEASURE(offsetof_CNA_VertexElement_format, offsetof(CNA_VertexElement, format))
CNA_GO_MEASURE(offsetof_CNA_VertexElement_usage, offsetof(CNA_VertexElement, usage))
CNA_GO_MEASURE(offsetof_CNA_VertexElement_usage_index, offsetof(CNA_VertexElement, usage_index))
CNA_GO_MEASURE(sizeof_CNA_VertexBufferCreateInfo, sizeof(CNA_VertexBufferCreateInfo))
CNA_GO_MEASURE(alignof_CNA_VertexBufferCreateInfo, _Alignof(CNA_VertexBufferCreateInfo))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferCreateInfo_vertex_declaration, offsetof(CNA_VertexBufferCreateInfo, vertex_declaration))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferCreateInfo_vertex_count, offsetof(CNA_VertexBufferCreateInfo, vertex_count))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferCreateInfo_buffer_usage, offsetof(CNA_VertexBufferCreateInfo, buffer_usage))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferCreateInfo_dynamic, offsetof(CNA_VertexBufferCreateInfo, dynamic))
CNA_GO_MEASURE(sizeof_CNA_VertexBufferInfo, sizeof(CNA_VertexBufferInfo))
CNA_GO_MEASURE(alignof_CNA_VertexBufferInfo, _Alignof(CNA_VertexBufferInfo))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_vertex_count, offsetof(CNA_VertexBufferInfo, vertex_count))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_buffer_usage, offsetof(CNA_VertexBufferInfo, buffer_usage))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_dynamic, offsetof(CNA_VertexBufferInfo, dynamic))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_is_content_lost, offsetof(CNA_VertexBufferInfo, is_content_lost))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_has_renderer, offsetof(CNA_VertexBufferInfo, has_renderer))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_vertex_stride, offsetof(CNA_VertexBufferInfo, vertex_stride))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferInfo_vertex_element_count, offsetof(CNA_VertexBufferInfo, vertex_element_count))

// Foundation 67. CNA_VertexBufferBinding, the one CNA struct ARRAY that crosses
// on the device's own surface. A 64-bit handle followed by two int32s: moving
// the handle behind them is byte-identical to C and makes CNA read a buffer
// token out of the offset and frequency words.
CNA_GO_MEASURE(sizeof_CNA_VertexBufferBinding, sizeof(CNA_VertexBufferBinding))
CNA_GO_MEASURE(alignof_CNA_VertexBufferBinding, _Alignof(CNA_VertexBufferBinding))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferBinding_vertex_buffer, offsetof(CNA_VertexBufferBinding, vertex_buffer))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferBinding_vertex_offset, offsetof(CNA_VertexBufferBinding, vertex_offset))
CNA_GO_MEASURE(offsetof_CNA_VertexBufferBinding_instance_frequency, offsetof(CNA_VertexBufferBinding, instance_frequency))

// Foundation 68. The two adapter structures.
//
// CNA_GraphicsAdapterInfo carries FOUR adjacent CNA_Bools between two uint32s
// and four int32s, so any permutation among them is byte-identical to C and
// changes which flag a caller reads -- is_default_adapter and is_wide_screen
// are the pair that decides what DefaultAdapter and IsWideScreen answer. Its
// two trailing uint64 byte lengths are the ones the string reads depend on:
// swapping them makes a description be read at the device name's length.
//
// CNA_GraphicsFormatSelection puts a CNA_Bool and three reserved bytes before
// two format words, so the reserved run's WIDTH decides where the selected
// format is read from.
CNA_GO_MEASURE(sizeof_CNA_GraphicsAdapterInfo, sizeof(CNA_GraphicsAdapterInfo))
CNA_GO_MEASURE(alignof_CNA_GraphicsAdapterInfo, _Alignof(CNA_GraphicsAdapterInfo))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_adapter_index, offsetof(CNA_GraphicsAdapterInfo, adapter_index))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_is_default_adapter, offsetof(CNA_GraphicsAdapterInfo, is_default_adapter))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_is_wide_screen, offsetof(CNA_GraphicsAdapterInfo, is_wide_screen))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_use_null_device, offsetof(CNA_GraphicsAdapterInfo, use_null_device))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_use_reference_device, offsetof(CNA_GraphicsAdapterInfo, use_reference_device))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_vendor_id, offsetof(CNA_GraphicsAdapterInfo, vendor_id))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_device_id, offsetof(CNA_GraphicsAdapterInfo, device_id))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_revision, offsetof(CNA_GraphicsAdapterInfo, revision))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_subsystem_id, offsetof(CNA_GraphicsAdapterInfo, subsystem_id))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_description_byte_length, offsetof(CNA_GraphicsAdapterInfo, description_byte_length))
CNA_GO_MEASURE(offsetof_CNA_GraphicsAdapterInfo_device_name_byte_length, offsetof(CNA_GraphicsAdapterInfo, device_name_byte_length))
CNA_GO_MEASURE(sizeof_CNA_GraphicsFormatSelection, sizeof(CNA_GraphicsFormatSelection))
CNA_GO_MEASURE(alignof_CNA_GraphicsFormatSelection, _Alignof(CNA_GraphicsFormatSelection))
CNA_GO_MEASURE(offsetof_CNA_GraphicsFormatSelection_exact_match, offsetof(CNA_GraphicsFormatSelection, exact_match))
CNA_GO_MEASURE(offsetof_CNA_GraphicsFormatSelection_format, offsetof(CNA_GraphicsFormatSelection, format))
CNA_GO_MEASURE(offsetof_CNA_GraphicsFormatSelection_depth_format, offsetof(CNA_GraphicsFormatSelection, depth_format))
CNA_GO_MEASURE(offsetof_CNA_GraphicsFormatSelection_multi_sample_count, offsetof(CNA_GraphicsFormatSelection, multi_sample_count))

// Foundation 69 — the SpriteFont family.
//
// CNA_SpriteFontGlyph is the one array element in this manifest whose fields
// have THREE different widths in one struct: two 16-byte rectangles, a uint16
// character, a uint16 reserved and three floats. The reserved half-word is what
// makes the kerning vector land on a four-byte boundary, so a manifest that
// dropped it would read the kerning from the character's address and every
// glyph after the first from the wrong offset.
//
// CNA_SpriteFontInfo puts a uint16 and a CNA_Bool before a five-byte reserved
// run, so the run's WIDTH is what decides the structure's size; a manifest one
// byte short would make CNA reject struct_size on every call.
CNA_GO_MEASURE(sizeof_CNA_Vector3, sizeof(CNA_Vector3))
CNA_GO_MEASURE(alignof_CNA_Vector3, _Alignof(CNA_Vector3))
CNA_GO_MEASURE(offsetof_CNA_Vector3_y, offsetof(CNA_Vector3, y))
CNA_GO_MEASURE(offsetof_CNA_Vector3_z, offsetof(CNA_Vector3, z))
CNA_GO_MEASURE(sizeof_CNA_SpriteFontGlyph, sizeof(CNA_SpriteFontGlyph))
CNA_GO_MEASURE(alignof_CNA_SpriteFontGlyph, _Alignof(CNA_SpriteFontGlyph))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontGlyph_glyph_bounds, offsetof(CNA_SpriteFontGlyph, glyph_bounds))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontGlyph_cropping, offsetof(CNA_SpriteFontGlyph, cropping))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontGlyph_character, offsetof(CNA_SpriteFontGlyph, character))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontGlyph_kerning, offsetof(CNA_SpriteFontGlyph, kerning))
CNA_GO_MEASURE(width_CNA_SpriteFontGlyph_character, sizeof(((CNA_SpriteFontGlyph*)0)->character))
CNA_GO_MEASURE(sizeof_CNA_SpriteFontInfo, sizeof(CNA_SpriteFontInfo))
CNA_GO_MEASURE(alignof_CNA_SpriteFontInfo, _Alignof(CNA_SpriteFontInfo))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontInfo_character_count, offsetof(CNA_SpriteFontInfo, character_count))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontInfo_line_spacing, offsetof(CNA_SpriteFontInfo, line_spacing))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontInfo_spacing, offsetof(CNA_SpriteFontInfo, spacing))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontInfo_default_character, offsetof(CNA_SpriteFontInfo, default_character))
CNA_GO_MEASURE(offsetof_CNA_SpriteFontInfo_has_default_character, offsetof(CNA_SpriteFontInfo, has_default_character))
CNA_GO_MEASURE(width_CNA_SpriteFontInfo_default_character, sizeof(((CNA_SpriteFontInfo*)0)->default_character))

// CNA_SpriteTextCommand. A handle and a two-word CNA_StringView sit between the
// two version words and the first vector, so the text view's WIDTH decides
// where every geometry field after it is read from; a manifest that made the
// view one word would put the position where the byte length is.
CNA_GO_MEASURE(sizeof_CNA_SpriteTextCommand, sizeof(CNA_SpriteTextCommand))
CNA_GO_MEASURE(alignof_CNA_SpriteTextCommand, _Alignof(CNA_SpriteTextCommand))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_sprite_font, offsetof(CNA_SpriteTextCommand, sprite_font))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_text, offsetof(CNA_SpriteTextCommand, text))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_position, offsetof(CNA_SpriteTextCommand, position))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_color, offsetof(CNA_SpriteTextCommand, color))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_rotation, offsetof(CNA_SpriteTextCommand, rotation))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_origin, offsetof(CNA_SpriteTextCommand, origin))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_scale, offsetof(CNA_SpriteTextCommand, scale))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_effects, offsetof(CNA_SpriteTextCommand, effects))
CNA_GO_MEASURE(offsetof_CNA_SpriteTextCommand_layer_depth, offsetof(CNA_SpriteTextCommand, layer_depth))
