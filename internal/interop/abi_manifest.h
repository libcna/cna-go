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
typedef struct CNA_Vector3 { float x; float y; float z; } CNA_Vector3;
typedef struct CNA_Vector4 { float x; float y; float z; float w; } CNA_Vector4;
typedef struct CNA_Quaternion { float x; float y; float z; float w; } CNA_Quaternion;
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
#define CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DISPOSING UINT32_C(0)
#define CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_LOST UINT32_C(1)
#define CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_RESET UINT32_C(2)
#define CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING UINT32_C(3)
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
typedef struct CNA_SpriteTextCommand {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Handle sprite_font;
    CNA_StringView text;
    CNA_Vector2 position;
    CNA_Color color;
    float rotation;
    CNA_Vector2 origin;
    CNA_Vector2 scale;
    uint32_t effects;
    float layer_depth;
} CNA_SpriteTextCommand;
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

#ifndef CNA_C_GRAPHICS_STATE_H
typedef uint32_t CNA_SpriteSortMode;
typedef uint32_t CNA_ShaderStage;
typedef CNA_Handle CNA_GraphicsDeviceEventRegistrationHandle;
typedef uint32_t CNA_GraphicsDeviceEvent;
#define CNA_GRAPHICS_DEVICE_EVENT_DISPOSING CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DISPOSING
#define CNA_GRAPHICS_DEVICE_EVENT_DEVICE_LOST CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_LOST
#define CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESET CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_RESET
#define CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING CNA_GO_MANIFEST_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING
typedef struct CNA_ResourceCreatedEventInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool has_resource;
    uint8_t reserved[7];
} CNA_ResourceCreatedEventInfo;
typedef struct CNA_ResourceDestroyedEventInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool has_tag;
    uint8_t reserved[7];
    CNA_StringView name;
} CNA_ResourceDestroyedEventInfo;
typedef void (*CNA_GraphicsDeviceEventCallback)(CNA_Handle, void*);
typedef void (*CNA_GraphicsDeviceResourceCreatedCallback)(CNA_Handle, const CNA_ResourceCreatedEventInfo*, void*);
typedef void (*CNA_GraphicsDeviceResourceDestroyedCallback)(CNA_Handle, const CNA_ResourceDestroyedEventInfo*, void*);
typedef struct CNA_TextureSlotInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool bound;
    uint8_t reserved[7];
    CNA_Handle texture;
} CNA_TextureSlotInfo;
typedef uint32_t CNA_Blend;
typedef uint32_t CNA_BlendFunction;
typedef uint32_t CNA_ColorWriteChannels;
typedef uint32_t CNA_CompareFunction;
typedef uint32_t CNA_StencilOperation;
typedef uint32_t CNA_CullMode;
typedef uint32_t CNA_FillMode;
typedef uint32_t CNA_TextureAddressMode;
typedef uint32_t CNA_TextureFilter;
typedef struct CNA_BlendState {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_BlendFunction alpha_blend_function;
    CNA_Blend alpha_destination_blend;
    CNA_Blend alpha_source_blend;
    CNA_BlendFunction color_blend_function;
    CNA_Blend color_destination_blend;
    CNA_Blend color_source_blend;
    CNA_ColorWriteChannels color_write_channels;
    CNA_ColorWriteChannels color_write_channels1;
    CNA_ColorWriteChannels color_write_channels2;
    CNA_ColorWriteChannels color_write_channels3;
    CNA_Color blend_factor;
    int32_t multi_sample_mask;
} CNA_BlendState;
typedef struct CNA_DepthStencilState {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool depth_buffer_enable;
    CNA_Bool depth_buffer_write_enable;
    CNA_Bool stencil_enable;
    CNA_Bool two_sided_stencil_mode;
    CNA_CompareFunction depth_buffer_function;
    CNA_CompareFunction stencil_function;
    int32_t stencil_mask;
    int32_t stencil_write_mask;
    int32_t reference_stencil;
    CNA_StencilOperation stencil_fail;
    CNA_StencilOperation stencil_depth_buffer_fail;
    CNA_StencilOperation stencil_pass;
    CNA_CompareFunction counter_clockwise_stencil_function;
    CNA_StencilOperation counter_clockwise_stencil_fail;
    CNA_StencilOperation counter_clockwise_stencil_depth_buffer_fail;
    CNA_StencilOperation counter_clockwise_stencil_pass;
    uint32_t reserved;
} CNA_DepthStencilState;
typedef struct CNA_RasterizerState {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_CullMode cull_mode;
    CNA_FillMode fill_mode;
    float depth_bias;
    float slope_scale_depth_bias;
    CNA_Bool multi_sample_anti_alias;
    CNA_Bool scissor_test_enable;
    uint8_t reserved[2];
} CNA_RasterizerState;
typedef struct CNA_SamplerState {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_TextureAddressMode address_u;
    CNA_TextureAddressMode address_v;
    CNA_TextureAddressMode address_w;
    CNA_TextureFilter filter;
    int32_t max_anisotropy;
    int32_t max_mip_level;
    float mip_map_level_of_detail_bias;
    uint32_t reserved;
} CNA_SamplerState;
#endif

#ifndef CNA_C_CONTENT_H
typedef struct CNA_ContentManagerCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_StringView root_directory;
    uint64_t reserved;
} CNA_ContentManagerCreateInfo;
#endif

#ifndef CNA_C_RENDER_TARGET_H
typedef uint32_t CNA_DepthFormat;
typedef uint32_t CNA_RenderTargetUsage;
typedef uint32_t CNA_CubeMapFace;
typedef uint32_t CNA_PresentInterval;
typedef uint32_t CNA_DisplayOrientation;
typedef uint32_t CNA_RenderTargetKind;
typedef struct CNA_RenderTarget2DCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    CNA_Bool mip_map;
    uint8_t reserved0[3];
    CNA_SurfaceFormat format;
    CNA_DepthFormat depth_format;
    int32_t multi_sample_count;
    CNA_RenderTargetUsage usage;
    uint32_t reserved1;
} CNA_RenderTarget2DCreateInfo;
typedef struct CNA_RenderTargetInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_RenderTargetKind kind;
    uint32_t width;
    uint32_t height;
    uint32_t level_count;
    CNA_SurfaceFormat format;
    CNA_DepthFormat depth_format;
    int32_t multi_sample_count;
    CNA_RenderTargetUsage usage;
    CNA_Bool is_content_lost;
    CNA_Bool renderer_available;
    uint8_t reserved[2];
} CNA_RenderTargetInfo;
typedef struct CNA_RenderTargetCubeCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t size;
    CNA_Bool mip_map;
    uint8_t reserved[3];
    CNA_SurfaceFormat format;
    CNA_DepthFormat depth_format;
    int32_t multi_sample_count;
    CNA_RenderTargetUsage usage;
} CNA_RenderTargetCubeCreateInfo;
typedef struct CNA_RenderTargetBinding {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Handle render_target;
    int32_t array_slice;
    CNA_CubeMapFace cube_map_face;
} CNA_RenderTargetBinding;
#endif

/* The display block's second half. It sits AFTER the render-target block
   because CNA_GraphicsFormatSelection names CNA_DepthFormat, which that block
   is where this manifest declares. The canonical header has the same dependency
   and satisfies it by including CNA/C/graphics.h. */
#ifndef CNA_C_DISPLAY_H
typedef uint64_t CNA_NativeHandleValue;
typedef struct CNA_GraphicsAdapterInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t adapter_index;
    CNA_Bool is_default_adapter;
    CNA_Bool is_wide_screen;
    CNA_Bool use_null_device;
    CNA_Bool use_reference_device;
    int32_t vendor_id;
    int32_t device_id;
    int32_t revision;
    int32_t subsystem_id;
    uint64_t description_byte_length;
    uint64_t device_name_byte_length;
} CNA_GraphicsAdapterInfo;
typedef struct CNA_GraphicsFormatSelection {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool exact_match;
    uint8_t reserved[3];
    CNA_SurfaceFormat format;
    CNA_DepthFormat depth_format;
    int32_t multi_sample_count;
} CNA_GraphicsFormatSelection;
typedef struct CNA_PresentationParameters {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_SurfaceFormat back_buffer_format;
    int32_t back_buffer_width;
    int32_t back_buffer_height;
    CNA_DepthFormat depth_stencil_format;
    int32_t multi_sample_count;
    CNA_PresentInterval presentation_interval;
    CNA_DisplayOrientation display_orientation;
    CNA_RenderTargetUsage render_target_usage;
    CNA_Bool is_full_screen;
    CNA_Bool headless_ext;
    uint8_t reserved[2];
} CNA_PresentationParameters;
#endif

#ifndef CNA_C_GRAPHICS3D_H
typedef uint32_t CNA_BufferUsage;
typedef uint32_t CNA_IndexElementSize;
typedef uint32_t CNA_SetDataOptions;
typedef uint32_t CNA_VertexElementFormat;
typedef uint32_t CNA_VertexElementUsage;
typedef uint32_t CNA_PrimitiveType;
typedef struct CNA_VertexElement {
    int32_t offset;
    CNA_VertexElementFormat format;
    CNA_VertexElementUsage usage;
    int32_t usage_index;
} CNA_VertexElement;
#endif

#ifndef CNA_C_GRAPHICS_DEVICE_H
typedef struct CNA_BackBufferReadback {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Bool has_source_rectangle;
    uint8_t reserved[3];
    CNA_Rectangle source_rectangle;
    uint64_t start_index;
    uint64_t element_count;
} CNA_BackBufferReadback;
#endif

#ifndef CNA_C_GRAPHICS_DEVICE_H
typedef uint32_t CNA_UserVertexSource;
typedef struct CNA_UserPrimitives {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_PrimitiveType primitive_type;
    CNA_UserVertexSource vertex_source;
    const void* vertex_data;
    CNA_Handle vertex_declaration;
    int32_t vertex_offset;
    int32_t num_vertices;
    int32_t primitive_count;
    uint32_t reserved;
} CNA_UserPrimitives;
typedef struct CNA_UserIndices {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_IndexElementSize index_element_size;
    int32_t index_offset;
    const void* index_data;
} CNA_UserIndices;
#endif

#ifndef CNA_C_VERTEX_RESOURCES_H
typedef CNA_Handle CNA_VertexDeclarationHandle;
typedef CNA_Handle CNA_VertexBufferHandle;
typedef struct CNA_VertexBufferCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_VertexDeclarationHandle vertex_declaration;
    int32_t vertex_count;
    CNA_BufferUsage buffer_usage;
    CNA_Bool dynamic;
    uint8_t reserved[7];
} CNA_VertexBufferCreateInfo;
typedef struct CNA_VertexBufferInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t vertex_count;
    CNA_BufferUsage buffer_usage;
    CNA_Bool dynamic;
    CNA_Bool is_content_lost;
    CNA_Bool has_renderer;
    uint8_t reserved0;
    int32_t vertex_stride;
    uint64_t vertex_element_count;
} CNA_VertexBufferInfo;
typedef struct CNA_VertexBufferBinding {
    CNA_VertexBufferHandle vertex_buffer;
    int32_t vertex_offset;
    int32_t instance_frequency;
} CNA_VertexBufferBinding;
#endif

#ifndef CNA_C_INDEX_RESOURCES_H
typedef CNA_Handle CNA_IndexBufferHandle;
typedef struct CNA_IndexBufferCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t index_count;
    CNA_IndexElementSize index_element_size;
    CNA_BufferUsage buffer_usage;
    CNA_Bool dynamic;
    uint8_t reserved[3];
} CNA_IndexBufferCreateInfo;
typedef struct CNA_IndexBufferInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t index_count;
    CNA_IndexElementSize index_element_size;
    CNA_BufferUsage buffer_usage;
    CNA_Bool dynamic;
    CNA_Bool is_content_lost;
    CNA_Bool has_renderer;
    uint8_t reserved;
} CNA_IndexBufferInfo;
typedef struct CNA_IndexBufferTransfer {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_IndexElementSize index_element_size;
    CNA_SetDataOptions options;
    uint64_t start_index;
    uint64_t element_count;
} CNA_IndexBufferTransfer;
#endif

/* CNA_Matrix lives in CNA/C/math_values.h and is the widest value this manifest
   carries: sixteen floats in ROW-MAJOR order, which is the order an effect
   parameter's matrix value crosses in. */
#ifndef CNA_C_MATH_VALUES_H_MATRIX
#ifndef CNA_C_MATH_VALUES_H
typedef struct CNA_Matrix {
    float m11; float m12; float m13; float m14;
    float m21; float m22; float m23; float m24;
    float m31; float m32; float m33; float m34;
    float m41; float m42; float m43; float m44;
} CNA_Matrix;
#endif
#endif

#ifndef CNA_C_EFFECTS_H
typedef uint32_t CNA_EffectParameterClass;
typedef uint32_t CNA_EffectParameterType;
typedef uint32_t CNA_EffectValueType;
typedef uint32_t CNA_EffectTextureType;
typedef CNA_Handle CNA_EffectHandle;
typedef CNA_Handle CNA_EffectParameterHandle;
/* Foundation 79. The stock-effect family's own handle alias. */
typedef CNA_Handle CNA_DirectionalLightHandle;
typedef CNA_Handle CNA_EffectParameterCollectionHandle;
typedef CNA_Handle CNA_EffectAnnotationHandle;
typedef CNA_Handle CNA_EffectAnnotationCollectionHandle;
typedef CNA_Handle CNA_EffectPassHandle;
typedef CNA_Handle CNA_EffectPassCollectionHandle;
typedef CNA_Handle CNA_EffectTechniqueHandle;
typedef CNA_Handle CNA_EffectTechniqueCollectionHandle;
typedef struct CNA_EffectParameterInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t row_count;
    int32_t column_count;
    CNA_EffectParameterClass parameter_class;
    CNA_EffectParameterType parameter_type;
} CNA_EffectParameterInfo;
typedef struct CNA_EffectAnnotationInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t row_count;
    int32_t column_count;
    CNA_EffectParameterClass parameter_class;
    CNA_EffectParameterType parameter_type;
} CNA_EffectAnnotationInfo;
#endif

#ifndef CNA_C_TEXTURE_VOLUME_H
typedef struct CNA_Texture3DCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    uint32_t depth;
    CNA_Bool mip_map;
    uint8_t reserved0[3];
    CNA_SurfaceFormat format;
    uint32_t reserved1;
} CNA_Texture3DCreateInfo;
typedef struct CNA_Texture3DInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t width;
    uint32_t height;
    uint32_t depth;
    uint32_t level_count;
    CNA_SurfaceFormat format;
    uint32_t reserved;
} CNA_Texture3DInfo;
typedef struct CNA_Texture3DTransfer {
    uint32_t struct_size;
    uint32_t struct_version;
    int32_t level;
    int32_t left;
    int32_t top;
    int32_t right;
    int32_t bottom;
    int32_t front;
    int32_t back;
    uint32_t reserved;
    uint64_t start_index;
    uint64_t element_count;
} CNA_Texture3DTransfer;
typedef struct CNA_TextureCubeCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t size;
    CNA_Bool mip_map;
    uint8_t reserved0[3];
    CNA_SurfaceFormat format;
    uint32_t reserved1;
} CNA_TextureCubeCreateInfo;
typedef struct CNA_TextureCubeInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t size;
    uint32_t level_count;
    CNA_SurfaceFormat format;
    uint32_t reserved;
} CNA_TextureCubeInfo;
typedef struct CNA_TextureCubeTransfer {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t face;
    int32_t level;
    CNA_Bool has_rectangle;
    uint8_t reserved0[3];
    CNA_Rectangle rectangle;
    uint32_t reserved1;
    uint64_t start_index;
    uint64_t element_count;
} CNA_TextureCubeTransfer;
#endif

#ifndef CNA_C_SPRITE_FONT_H
typedef uint16_t CNA_Char16;
typedef struct CNA_SpriteFontGlyph {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Rectangle glyph_bounds;
    CNA_Rectangle cropping;
    CNA_Char16 character;
    uint16_t reserved;
    CNA_Vector3 kerning;
} CNA_SpriteFontGlyph;
typedef struct CNA_SpriteFontInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint64_t character_count;
    int32_t line_spacing;
    float spacing;
    CNA_Char16 default_character;
    CNA_Bool has_default_character;
    uint8_t reserved[5];
} CNA_SpriteFontInfo;
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
typedef CNA_Result (*cna_vertex_declaration_create_fn)(const CNA_VertexElement*, uint64_t, CNA_VertexDeclarationHandle*);
typedef CNA_Result (*cna_vertex_declaration_create_with_stride_fn)(int32_t, const CNA_VertexElement*, uint64_t, CNA_VertexDeclarationHandle*);
typedef CNA_Result (*cna_vertex_declaration_destroy_fn)(CNA_VertexDeclarationHandle);
typedef CNA_Result (*cna_vertex_declaration_get_stride_fn)(CNA_VertexDeclarationHandle, int32_t*);
typedef CNA_Result (*cna_vertex_buffer_create_fn)(CNA_Handle, const CNA_VertexBufferCreateInfo*, CNA_VertexBufferHandle*);
typedef CNA_Result (*cna_vertex_buffer_destroy_fn)(CNA_VertexBufferHandle);
typedef CNA_Result (*cna_vertex_buffer_get_info_fn)(CNA_VertexBufferHandle, CNA_VertexBufferInfo*);
typedef CNA_Result (*cna_vertex_buffer_set_data_raw_at_fn)(CNA_VertexBufferHandle, uint64_t, const void*, uint64_t, uint64_t, uint32_t);
typedef CNA_Result (*cna_vertex_buffer_get_data_raw_fn)(CNA_VertexBufferHandle, uint64_t, void*, uint64_t, uint64_t, uint32_t);
typedef CNA_Result (*cna_graphics_device_get_adapter_index_fn)(CNA_Handle, uint32_t*);
typedef CNA_Result (*cna_graphics_adapter_get_count_fn)(CNA_Handle, uint64_t*);
typedef CNA_Result (*cna_graphics_adapter_get_info_fn)(CNA_Handle, uint32_t, CNA_GraphicsAdapterInfo*);
typedef CNA_Result (*cna_graphics_adapter_copy_description_fn)(CNA_Handle, uint32_t, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_graphics_adapter_copy_device_name_fn)(CNA_Handle, uint32_t, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_graphics_adapter_get_current_display_mode_fn)(CNA_Handle, uint32_t, CNA_DisplayMode*);
typedef CNA_Result (*cna_graphics_adapter_get_display_mode_count_fn)(CNA_Handle, uint32_t, CNA_Bool, CNA_SurfaceFormat, uint64_t*);
typedef CNA_Result (*cna_graphics_adapter_copy_display_modes_fn)(CNA_Handle, uint32_t, CNA_Bool, CNA_SurfaceFormat, CNA_DisplayMode*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_graphics_adapter_set_device_preferences_fn)(CNA_Handle, uint32_t, CNA_Bool, CNA_Bool);
typedef CNA_Result (*cna_graphics_adapter_is_profile_supported_fn)(CNA_Handle, uint32_t, CNA_GraphicsProfile, CNA_Bool*);
typedef CNA_Result (*cna_graphics_adapter_query_render_target_format_fn)(CNA_Handle, uint32_t, CNA_GraphicsProfile, CNA_SurfaceFormat, CNA_DepthFormat, int32_t, CNA_GraphicsFormatSelection*);
typedef CNA_Result (*cna_graphics_adapter_query_backbuffer_format_fn)(CNA_Handle, uint32_t, CNA_GraphicsProfile, CNA_SurfaceFormat, CNA_DepthFormat, int32_t, CNA_GraphicsFormatSelection*);
typedef CNA_Result (*cna_graphics_adapter_get_native_monitor_handle_fn)(CNA_Handle, uint32_t, CNA_NativeHandleValue*);
typedef CNA_Result (*cna_graphics_device_set_vertex_buffers_fn)(CNA_Handle, const CNA_VertexBufferBinding*, uint64_t);
typedef CNA_Result (*cna_graphics_device_set_index_buffer_fn)(CNA_Handle, CNA_IndexBufferHandle);
typedef CNA_Result (*cna_graphics_device_draw_primitives_fn)(CNA_Handle, CNA_PrimitiveType, int32_t, int32_t);
typedef CNA_Result (*cna_graphics_device_draw_indexed_primitives_fn)(CNA_Handle, CNA_PrimitiveType, int32_t, int32_t, int32_t, int32_t, int32_t);
typedef CNA_Result (*cna_graphics_device_draw_instanced_primitives_fn)(CNA_Handle, CNA_PrimitiveType, int32_t, int32_t, int32_t, int32_t, int32_t, int32_t);
typedef CNA_Result (*cna_render_target_cube_create_fn)(CNA_Handle, const CNA_RenderTargetCubeCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_set_render_target_cube_fn)(CNA_Handle, CNA_Handle, CNA_CubeMapFace);
typedef CNA_Result (*cna_graphics_device_set_render_targets_fn)(CNA_Handle, const CNA_RenderTargetBinding*, uint64_t);
typedef CNA_Result (*cna_graphics_device_get_render_target_count_fn)(CNA_Handle, uint64_t*);
typedef CNA_Result (*cna_graphics_device_create_fn)(uint32_t, uint32_t, const CNA_PresentationParameters*, CNA_Handle*);
typedef CNA_Result (*cna_graphics_device_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_reset_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_reset_with_parameters_fn)(CNA_Handle, const CNA_PresentationParameters*, const uint32_t*);
typedef CNA_Result (*cna_graphics_device_get_presentation_parameters_fn)(CNA_Handle, CNA_PresentationParameters*);
typedef CNA_Result (*cna_graphics_device_get_backbuffer_data_window_fn)(CNA_Handle, const CNA_BackBufferReadback*, CNA_Color*, uint64_t);
typedef CNA_Result (*cna_graphics_device_draw_user_primitives_fn)(CNA_Handle, const CNA_UserPrimitives*);
typedef CNA_Result (*cna_graphics_device_draw_user_indexed_primitives_fn)(CNA_Handle, const CNA_UserPrimitives*, const CNA_UserIndices*);
typedef CNA_Result (*cna_index_buffer_create_fn)(CNA_Handle, const CNA_IndexBufferCreateInfo*, CNA_IndexBufferHandle*);
typedef CNA_Result (*cna_index_buffer_destroy_fn)(CNA_IndexBufferHandle);
typedef CNA_Result (*cna_index_buffer_get_info_fn)(CNA_IndexBufferHandle, CNA_IndexBufferInfo*);
typedef CNA_Result (*cna_index_buffer_set_data_fn)(CNA_IndexBufferHandle, const CNA_IndexBufferTransfer*, const void*, uint64_t);
typedef CNA_Result (*cna_index_buffer_set_data_at_fn)(CNA_IndexBufferHandle, uint64_t, const CNA_IndexBufferTransfer*, const void*, uint64_t);
typedef CNA_Result (*cna_index_buffer_get_data_fn)(CNA_IndexBufferHandle, const CNA_IndexBufferTransfer*, void*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_content_manager_create_fn)(CNA_Handle, const CNA_ContentManagerCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_content_manager_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_content_manager_get_root_directory_size_fn)(CNA_Handle, uint64_t*);
typedef CNA_Result (*cna_content_manager_copy_root_directory_fn)(CNA_Handle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_content_manager_set_root_directory_fn)(CNA_Handle, CNA_StringView);
typedef CNA_Result (*cna_content_manager_unload_fn)(CNA_Handle);
typedef CNA_Result (*cna_content_manager_load_texture2d_fn)(CNA_Handle, CNA_StringView, CNA_Handle*);
typedef CNA_Result (*cna_content_manager_get_asset_path_size_fn)(CNA_Handle, CNA_StringView, uint64_t*);
typedef CNA_Result (*cna_content_manager_copy_asset_path_fn)(CNA_Handle, CNA_StringView, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_content_manager_load_sprite_font_fn)(CNA_Handle, CNA_StringView, CNA_Handle*, CNA_Handle*);
typedef CNA_Result (*cna_texture3d_create_fn)(CNA_Handle, const CNA_Texture3DCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_texture3d_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_texture3d_get_info_fn)(CNA_Handle, CNA_Texture3DInfo*);
typedef CNA_Result (*cna_texture3d_set_data_fn)(CNA_Handle, const CNA_Texture3DTransfer*, const CNA_Color*, uint64_t);
typedef CNA_Result (*cna_texture3d_get_data_fn)(CNA_Handle, const CNA_Texture3DTransfer*, CNA_Color*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_texturecube_create_fn)(CNA_Handle, const CNA_TextureCubeCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_texturecube_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_texturecube_get_info_fn)(CNA_Handle, CNA_TextureCubeInfo*);
typedef CNA_Result (*cna_texturecube_set_data_fn)(CNA_Handle, const CNA_TextureCubeTransfer*, const CNA_Color*, uint64_t);
typedef CNA_Result (*cna_texturecube_get_data_fn)(CNA_Handle, const CNA_TextureCubeTransfer*, CNA_Color*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_sprite_font_get_info_fn)(CNA_Handle, CNA_SpriteFontInfo*);
typedef CNA_Result (*cna_sprite_batch_draw_string_fn)(CNA_Handle, const CNA_SpriteTextCommand*);
typedef CNA_Result (*cna_sprite_batch_begin_with_effect_fn)(CNA_Handle, CNA_SpriteSortMode, const CNA_BlendState*, const CNA_SamplerState*, const CNA_DepthStencilState*, const CNA_RasterizerState*, CNA_Handle, const CNA_Matrix*);
/* Foundation 79 -- the stock-effect routes. */
typedef CNA_Result (*cna_basic_effect_create_fn)(CNA_Handle, CNA_EffectHandle*);
typedef CNA_Result (*cna_basic_effect_set_vertex_color_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_basic_effect_set_prefer_per_pixel_lighting_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_basic_effect_set_diffuse_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_basic_effect_set_emissive_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_basic_effect_get_specular_color_fn)(CNA_EffectHandle, CNA_Vector3*);
typedef CNA_Result (*cna_basic_effect_set_specular_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_basic_effect_get_specular_power_fn)(CNA_EffectHandle, float*);
typedef CNA_Result (*cna_basic_effect_set_specular_power_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_basic_effect_set_alpha_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_basic_effect_set_texture_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_basic_effect_set_texture_fn)(CNA_EffectHandle, CNA_Handle);
typedef CNA_Result (*cna_effect_matrices_set_world_fn)(CNA_EffectHandle, CNA_Matrix);
typedef CNA_Result (*cna_effect_matrices_set_view_fn)(CNA_EffectHandle, CNA_Matrix);
typedef CNA_Result (*cna_effect_matrices_set_projection_fn)(CNA_EffectHandle, CNA_Matrix);
typedef CNA_Result (*cna_effect_fog_get_color_fn)(CNA_EffectHandle, CNA_Vector3*);
typedef CNA_Result (*cna_effect_fog_set_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_effect_fog_set_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_effect_fog_set_start_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_effect_fog_set_end_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_effect_lights_set_ambient_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_effect_lights_get_directional_light_fn)(CNA_EffectHandle, uint32_t, CNA_DirectionalLightHandle*);
typedef CNA_Result (*cna_effect_lights_set_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_directional_light_destroy_fn)(CNA_DirectionalLightHandle);
typedef CNA_Result (*cna_directional_light_set_diffuse_color_fn)(CNA_DirectionalLightHandle, CNA_Vector3);
typedef CNA_Result (*cna_directional_light_set_direction_fn)(CNA_DirectionalLightHandle, CNA_Vector3);
typedef CNA_Result (*cna_directional_light_set_specular_color_fn)(CNA_DirectionalLightHandle, CNA_Vector3);
typedef CNA_Result (*cna_directional_light_set_enabled_fn)(CNA_DirectionalLightHandle, CNA_Bool);
/* Foundation 80 -- AlphaTestEffect, DualTextureEffect and EffectMaterial. */
typedef CNA_Result (*cna_alpha_test_effect_create_fn)(CNA_Handle, CNA_EffectHandle*);
typedef CNA_Result (*cna_alpha_test_effect_set_diffuse_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_alpha_test_effect_set_alpha_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_alpha_test_effect_set_texture_fn)(CNA_EffectHandle, CNA_Handle);
typedef CNA_Result (*cna_alpha_test_effect_set_vertex_color_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_alpha_test_effect_set_alpha_function_fn)(CNA_EffectHandle, CNA_CompareFunction);
typedef CNA_Result (*cna_alpha_test_effect_set_reference_alpha_fn)(CNA_EffectHandle, int32_t);
typedef CNA_Result (*cna_dual_texture_effect_create_fn)(CNA_Handle, CNA_EffectHandle*);
typedef CNA_Result (*cna_dual_texture_effect_set_diffuse_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_dual_texture_effect_set_alpha_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_dual_texture_effect_set_texture_fn)(CNA_EffectHandle, uint32_t, CNA_Handle);
typedef CNA_Result (*cna_dual_texture_effect_set_vertex_color_enabled_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_effect_material_create_fn)(CNA_EffectHandle, CNA_EffectHandle*);
/* Foundation 82 -- the two root types. */
typedef CNA_Result (*cna_framework_dispatcher_update_fn)(CNA_Handle);
typedef CNA_Result (*cna_title_container_read_ext_fn)(CNA_Handle, CNA_StringView, uint8_t*, uint64_t, uint64_t*);
/* Foundation 83 -- OcclusionQuery. */
typedef CNA_Handle CNA_OcclusionQueryHandle;
typedef CNA_Result (*cna_occlusion_query_create_fn)(CNA_Handle, CNA_OcclusionQueryHandle*);
typedef CNA_Result (*cna_occlusion_query_destroy_fn)(CNA_OcclusionQueryHandle);
typedef CNA_Result (*cna_occlusion_query_begin_fn)(CNA_OcclusionQueryHandle);
typedef CNA_Result (*cna_occlusion_query_end_fn)(CNA_OcclusionQueryHandle);
typedef CNA_Result (*cna_occlusion_query_get_is_complete_fn)(CNA_OcclusionQueryHandle, CNA_Bool*);
typedef CNA_Result (*cna_occlusion_query_get_pixel_count_fn)(CNA_OcclusionQueryHandle, int32_t*);
/* Foundation 87 -- SoundEffect and SoundEffectInstance.
   CNA_AUDIO_CHANNELS_MONO is 1 and _STEREO is 2, which happen to match XNA's
   AudioChannels literals; the Go side maps them explicitly anyway. */
#ifndef CNA_C_AUDIO_H
typedef uint32_t CNA_AudioChannels;
typedef uint32_t CNA_SoundState;
typedef struct CNA_SoundEffectCreateInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    uint32_t sample_rate;
    CNA_AudioChannels channels;
    uint64_t reserved;
} CNA_SoundEffectCreateInfo;
typedef struct CNA_SoundEffectInstanceInfo {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_SoundState state;
    CNA_Bool is_looped;
    uint8_t reserved0[3];
    float volume;
    float pitch;
    float pan;
    uint32_t reserved1;
} CNA_SoundEffectInstanceInfo;
#endif
typedef CNA_Result (*cna_sound_effect_create_pcm16_range_ext_fn)(CNA_Handle, const CNA_SoundEffectCreateInfo*, const uint8_t*, uint64_t, int32_t, int32_t, int32_t, int32_t, CNA_Handle*);
typedef CNA_Result (*cna_sound_effect_create_from_encoded_ext_fn)(CNA_Handle, const uint8_t*, uint64_t, CNA_Handle*);
typedef CNA_Result (*cna_sound_effect_get_duration_ticks_fn)(CNA_Handle, int64_t*);
typedef CNA_Result (*cna_sound_effect_create_instance_fn)(CNA_Handle, CNA_Handle*);
typedef CNA_Result (*cna_sound_effect_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_sound_effect_play_fn)(CNA_Handle, CNA_Bool*);
typedef CNA_Result (*cna_sound_effect_play_with_settings_fn)(CNA_Handle, float, float, float, CNA_Bool*);
typedef CNA_Result (*cna_sound_effect_set_master_volume_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_set_distance_scale_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_set_doppler_scale_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_set_speed_of_sound_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_instance_play_fn)(CNA_Handle);
typedef CNA_Result (*cna_sound_effect_instance_pause_fn)(CNA_Handle);
typedef CNA_Result (*cna_sound_effect_instance_resume_fn)(CNA_Handle);
typedef CNA_Result (*cna_sound_effect_instance_stop_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_sound_effect_instance_get_info_fn)(CNA_Handle, CNA_SoundEffectInstanceInfo*);
typedef CNA_Result (*cna_sound_effect_instance_set_volume_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_instance_set_pitch_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_instance_set_pan_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sound_effect_instance_set_is_looped_fn)(CNA_Handle, CNA_Bool);
typedef CNA_Result (*cna_sound_effect_instance_destroy_fn)(CNA_Handle);
#ifndef CNA_C_AUDIO_H
typedef struct CNA_AudioEmitter {
    uint32_t struct_size;
    uint32_t struct_version;
    float doppler_scale;
    CNA_Vector3 forward;
    CNA_Vector3 position;
    CNA_Vector3 up;
    CNA_Vector3 velocity;
} CNA_AudioEmitter;
typedef struct CNA_AudioListener {
    uint32_t struct_size;
    uint32_t struct_version;
    CNA_Vector3 forward;
    CNA_Vector3 position;
    CNA_Vector3 up;
    CNA_Vector3 velocity;
} CNA_AudioListener;
#endif
typedef CNA_Result (*cna_sound_effect_instance_apply_3d_multi_ext_fn)(CNA_Handle, const CNA_AudioListener*, uint64_t, const CNA_AudioEmitter*);
/* Foundation 84 -- the dynamic buffers' options-carrying upload. */
typedef CNA_Result (*cna_vertex_buffer_set_data_raw_at_with_options_fn)(CNA_VertexBufferHandle, uint64_t, const void*, uint64_t, uint64_t, uint32_t, CNA_SetDataOptions);
/* Foundation 81 -- EnvironmentMapEffect and SkinnedEffect. */
typedef CNA_Result (*cna_environment_map_effect_create_fn)(CNA_Handle, CNA_EffectHandle*);
typedef CNA_Result (*cna_environment_map_effect_set_diffuse_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_environment_map_effect_set_emissive_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_environment_map_effect_set_alpha_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_environment_map_effect_set_texture_fn)(CNA_EffectHandle, CNA_Handle);
typedef CNA_Result (*cna_environment_map_effect_set_environment_map_fn)(CNA_EffectHandle, CNA_Handle);
typedef CNA_Result (*cna_environment_map_effect_get_amount_fn)(CNA_EffectHandle, float*);
typedef CNA_Result (*cna_environment_map_effect_set_amount_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_environment_map_effect_get_specular_fn)(CNA_EffectHandle, CNA_Vector3*);
typedef CNA_Result (*cna_environment_map_effect_set_specular_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_environment_map_effect_get_fresnel_factor_fn)(CNA_EffectHandle, float*);
typedef CNA_Result (*cna_environment_map_effect_set_fresnel_factor_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_skinned_effect_create_fn)(CNA_Handle, CNA_EffectHandle*);
typedef CNA_Result (*cna_skinned_effect_set_diffuse_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_skinned_effect_set_emissive_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_skinned_effect_get_specular_color_fn)(CNA_EffectHandle, CNA_Vector3*);
typedef CNA_Result (*cna_skinned_effect_set_specular_color_fn)(CNA_EffectHandle, CNA_Vector3);
typedef CNA_Result (*cna_skinned_effect_get_specular_power_fn)(CNA_EffectHandle, float*);
typedef CNA_Result (*cna_skinned_effect_set_specular_power_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_skinned_effect_set_alpha_fn)(CNA_EffectHandle, float);
typedef CNA_Result (*cna_skinned_effect_set_prefer_per_pixel_lighting_fn)(CNA_EffectHandle, CNA_Bool);
typedef CNA_Result (*cna_skinned_effect_set_texture_fn)(CNA_EffectHandle, CNA_Handle);
typedef CNA_Result (*cna_skinned_effect_set_weights_per_vertex_fn)(CNA_EffectHandle, int32_t);
typedef CNA_Result (*cna_skinned_effect_set_bone_transforms_fn)(CNA_EffectHandle, const CNA_Matrix*, uint64_t);
typedef CNA_Result (*cna_skinned_effect_copy_bone_transforms_fn)(CNA_EffectHandle, uint64_t, CNA_Matrix*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_create_compiled_fn)(CNA_Handle, const uint8_t*, uint64_t, CNA_EffectHandle*);
typedef CNA_Result (*cna_content_manager_load_effect_fn)(CNA_Handle, CNA_StringView, CNA_EffectHandle*);
typedef CNA_Result (*cna_effect_clone_fn)(CNA_EffectHandle, CNA_EffectHandle*);
typedef CNA_Result (*cna_effect_destroy_fn)(CNA_EffectHandle);
typedef CNA_Result (*cna_effect_apply_fn)(CNA_EffectHandle);
typedef CNA_Result (*cna_effect_get_parameters_fn)(CNA_EffectHandle, CNA_EffectParameterCollectionHandle*);
typedef CNA_Result (*cna_effect_get_techniques_fn)(CNA_EffectHandle, CNA_EffectTechniqueCollectionHandle*);
typedef CNA_Result (*cna_effect_get_current_technique_fn)(CNA_EffectHandle, CNA_EffectTechniqueHandle*);
typedef CNA_Result (*cna_effect_set_current_technique_fn)(CNA_EffectHandle, CNA_EffectTechniqueHandle);
typedef CNA_Result (*cna_effect_technique_collection_get_count_fn)(CNA_EffectTechniqueCollectionHandle, uint64_t*);
typedef CNA_Result (*cna_effect_technique_collection_get_at_fn)(CNA_EffectTechniqueCollectionHandle, uint64_t, CNA_EffectTechniqueHandle*);
typedef CNA_Result (*cna_effect_technique_destroy_fn)(CNA_EffectTechniqueHandle);
typedef CNA_Result (*cna_effect_technique_get_name_byte_count_fn)(CNA_EffectTechniqueHandle, uint64_t*);
typedef CNA_Result (*cna_effect_technique_copy_name_fn)(CNA_EffectTechniqueHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_technique_get_passes_fn)(CNA_EffectTechniqueHandle, CNA_EffectPassCollectionHandle*);
typedef CNA_Result (*cna_effect_technique_get_annotations_fn)(CNA_EffectTechniqueHandle, CNA_EffectAnnotationCollectionHandle*);
typedef CNA_Result (*cna_effect_technique_collection_destroy_fn)(CNA_EffectTechniqueCollectionHandle);
typedef CNA_Result (*cna_effect_pass_collection_get_count_fn)(CNA_EffectPassCollectionHandle, uint64_t*);
typedef CNA_Result (*cna_effect_pass_collection_get_at_fn)(CNA_EffectPassCollectionHandle, uint64_t, CNA_EffectPassHandle*);
typedef CNA_Result (*cna_effect_pass_collection_destroy_fn)(CNA_EffectPassCollectionHandle);
typedef CNA_Result (*cna_effect_pass_destroy_fn)(CNA_EffectPassHandle);
typedef CNA_Result (*cna_effect_pass_get_name_byte_count_fn)(CNA_EffectPassHandle, uint64_t*);
typedef CNA_Result (*cna_effect_pass_copy_name_fn)(CNA_EffectPassHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_pass_get_annotations_fn)(CNA_EffectPassHandle, CNA_EffectAnnotationCollectionHandle*);
typedef CNA_Result (*cna_effect_pass_apply_fn)(CNA_EffectPassHandle);
typedef CNA_Result (*cna_effect_parameter_collection_get_count_fn)(CNA_EffectParameterCollectionHandle, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_collection_get_at_fn)(CNA_EffectParameterCollectionHandle, uint64_t, CNA_EffectParameterHandle*);
typedef CNA_Result (*cna_effect_parameter_collection_destroy_fn)(CNA_EffectParameterCollectionHandle);
typedef CNA_Result (*cna_effect_parameter_destroy_fn)(CNA_EffectParameterHandle);
typedef CNA_Result (*cna_effect_parameter_get_info_fn)(CNA_EffectParameterHandle, CNA_EffectParameterInfo*);
typedef CNA_Result (*cna_effect_parameter_get_name_byte_count_fn)(CNA_EffectParameterHandle, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_copy_name_fn)(CNA_EffectParameterHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_get_semantic_byte_count_fn)(CNA_EffectParameterHandle, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_copy_semantic_fn)(CNA_EffectParameterHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_get_elements_fn)(CNA_EffectParameterHandle, CNA_EffectParameterCollectionHandle*);
typedef CNA_Result (*cna_effect_parameter_get_structure_members_fn)(CNA_EffectParameterHandle, CNA_EffectParameterCollectionHandle*);
typedef CNA_Result (*cna_effect_parameter_get_annotations_fn)(CNA_EffectParameterHandle, CNA_EffectAnnotationCollectionHandle*);
typedef CNA_Result (*cna_effect_parameter_get_value_fn)(CNA_EffectParameterHandle, CNA_EffectValueType, void*);
typedef CNA_Result (*cna_effect_parameter_get_values_fn)(CNA_EffectParameterHandle, CNA_EffectValueType, uint64_t, void*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_set_value_fn)(CNA_EffectParameterHandle, CNA_EffectValueType, const void*);
typedef CNA_Result (*cna_effect_parameter_set_values_fn)(CNA_EffectParameterHandle, CNA_EffectValueType, const void*, uint64_t);
typedef CNA_Result (*cna_effect_parameter_get_value_string_byte_count_fn)(CNA_EffectParameterHandle, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_copy_value_string_fn)(CNA_EffectParameterHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_parameter_set_value_string_fn)(CNA_EffectParameterHandle, CNA_StringView);
typedef CNA_Result (*cna_effect_parameter_set_value_texture_fn)(CNA_EffectParameterHandle, CNA_EffectTextureType, CNA_Handle);
typedef CNA_Result (*cna_effect_annotation_collection_get_count_fn)(CNA_EffectAnnotationCollectionHandle, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_collection_get_at_fn)(CNA_EffectAnnotationCollectionHandle, uint64_t, CNA_EffectAnnotationHandle*);
typedef CNA_Result (*cna_effect_annotation_collection_destroy_fn)(CNA_EffectAnnotationCollectionHandle);
typedef CNA_Result (*cna_effect_annotation_destroy_fn)(CNA_EffectAnnotationHandle);
typedef CNA_Result (*cna_effect_annotation_get_info_fn)(CNA_EffectAnnotationHandle, CNA_EffectAnnotationInfo*);
typedef CNA_Result (*cna_effect_annotation_get_name_byte_count_fn)(CNA_EffectAnnotationHandle, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_copy_name_fn)(CNA_EffectAnnotationHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_get_semantic_byte_count_fn)(CNA_EffectAnnotationHandle, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_copy_semantic_fn)(CNA_EffectAnnotationHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_get_value_boolean_fn)(CNA_EffectAnnotationHandle, CNA_Bool*);
typedef CNA_Result (*cna_effect_annotation_get_value_int32_fn)(CNA_EffectAnnotationHandle, int32_t*);
typedef CNA_Result (*cna_effect_annotation_get_value_single_fn)(CNA_EffectAnnotationHandle, float*);
typedef CNA_Result (*cna_effect_annotation_get_value_string_byte_count_fn)(CNA_EffectAnnotationHandle, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_copy_value_string_fn)(CNA_EffectAnnotationHandle, char*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_effect_annotation_get_value_vector2_fn)(CNA_EffectAnnotationHandle, CNA_Vector2*);
typedef CNA_Result (*cna_effect_annotation_get_value_vector3_fn)(CNA_EffectAnnotationHandle, CNA_Vector3*);
typedef CNA_Result (*cna_effect_annotation_get_value_vector4_fn)(CNA_EffectAnnotationHandle, CNA_Vector4*);
typedef CNA_Result (*cna_effect_annotation_get_value_matrix_fn)(CNA_EffectAnnotationHandle, CNA_Matrix*);
typedef CNA_Result (*cna_sprite_font_copy_glyphs_fn)(CNA_Handle, CNA_SpriteFontGlyph*, uint64_t, uint64_t*);
typedef CNA_Result (*cna_sprite_font_set_default_character_fn)(CNA_Handle, CNA_Bool, CNA_Char16);
typedef CNA_Result (*cna_sprite_font_set_line_spacing_fn)(CNA_Handle, int32_t);
typedef CNA_Result (*cna_sprite_font_set_spacing_fn)(CNA_Handle, float);
typedef CNA_Result (*cna_sprite_font_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_subscribe_event_fn)(CNA_Handle, CNA_GraphicsDeviceEvent, CNA_GraphicsDeviceEventCallback, void*, CNA_GraphicsDeviceEventRegistrationHandle*);
typedef CNA_Result (*cna_graphics_device_subscribe_resource_created_fn)(CNA_Handle, CNA_GraphicsDeviceResourceCreatedCallback, void*, CNA_GraphicsDeviceEventRegistrationHandle*);
typedef CNA_Result (*cna_graphics_device_subscribe_resource_destroyed_fn)(CNA_Handle, CNA_GraphicsDeviceResourceDestroyedCallback, void*, CNA_GraphicsDeviceEventRegistrationHandle*);
typedef CNA_Result (*cna_graphics_device_unsubscribe_fn)(CNA_GraphicsDeviceEventRegistrationHandle);
typedef CNA_Result (*cna_graphics_device_dispose_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_get_texture_fn)(CNA_Handle, CNA_ShaderStage, uint32_t, CNA_TextureSlotInfo*);
typedef CNA_Result (*cna_graphics_device_set_texture_fn)(CNA_Handle, CNA_ShaderStage, uint32_t, CNA_Handle);
typedef CNA_Result (*cna_graphics_device_get_sampler_state_fn)(CNA_Handle, CNA_ShaderStage, uint32_t, CNA_SamplerState*);
typedef CNA_Result (*cna_graphics_device_set_sampler_state_fn)(CNA_Handle, CNA_ShaderStage, uint32_t, const CNA_SamplerState*);
typedef CNA_Result (*cna_graphics_device_set_blend_state_fn)(CNA_Handle, const CNA_BlendState*);
typedef CNA_Result (*cna_graphics_device_set_depth_stencil_state_fn)(CNA_Handle, const CNA_DepthStencilState*);
typedef CNA_Result (*cna_graphics_device_set_rasterizer_state_fn)(CNA_Handle, const CNA_RasterizerState*);
typedef CNA_Result (*cna_render_target2d_create_fn)(CNA_Handle, const CNA_RenderTarget2DCreateInfo*, CNA_Handle*);
typedef CNA_Result (*cna_render_target_get_info_fn)(CNA_Handle, CNA_RenderTargetInfo*);
typedef CNA_Result (*cna_render_target_destroy_fn)(CNA_Handle);
typedef CNA_Result (*cna_graphics_device_set_render_target2d_fn)(CNA_Handle, CNA_Handle);
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
    X(cna_graphics_device_get_adapter_index) \
    X(cna_graphics_adapter_get_count) \
    X(cna_graphics_adapter_get_info) \
    X(cna_graphics_adapter_copy_description) \
    X(cna_graphics_adapter_copy_device_name) \
    X(cna_graphics_adapter_get_current_display_mode) \
    X(cna_graphics_adapter_get_display_mode_count) \
    X(cna_graphics_adapter_copy_display_modes) \
    X(cna_graphics_adapter_set_device_preferences) \
    X(cna_graphics_adapter_is_profile_supported) \
    X(cna_graphics_adapter_query_render_target_format) \
    X(cna_graphics_adapter_query_backbuffer_format) \
    X(cna_graphics_adapter_get_native_monitor_handle) \
    X(cna_graphics_device_set_vertex_buffers) \
    X(cna_graphics_device_set_index_buffer) \
    X(cna_graphics_device_draw_primitives) \
    X(cna_graphics_device_draw_indexed_primitives) \
    X(cna_graphics_device_draw_instanced_primitives) \
    X(cna_render_target_cube_create) \
    X(cna_graphics_device_set_render_target_cube) \
    X(cna_graphics_device_set_render_targets) \
    X(cna_graphics_device_get_render_target_count) \
    X(cna_graphics_device_create) \
    X(cna_graphics_device_destroy) \
    X(cna_graphics_device_reset) \
    X(cna_graphics_device_reset_with_parameters) \
    X(cna_graphics_device_get_presentation_parameters) \
    X(cna_graphics_device_get_backbuffer_data_window) \
    X(cna_graphics_device_draw_user_primitives) \
    X(cna_graphics_device_draw_user_indexed_primitives) \
    X(cna_vertex_declaration_create) \
    X(cna_vertex_declaration_create_with_stride) \
    X(cna_vertex_declaration_destroy) \
    X(cna_vertex_declaration_get_stride) \
    X(cna_vertex_buffer_create) \
    X(cna_vertex_buffer_destroy) \
    X(cna_vertex_buffer_get_info) \
    X(cna_vertex_buffer_set_data_raw_at) \
    X(cna_vertex_buffer_get_data_raw) \
    X(cna_index_buffer_create) \
    X(cna_index_buffer_destroy) \
    X(cna_index_buffer_get_info) \
    X(cna_index_buffer_set_data) \
    X(cna_index_buffer_set_data_at) \
    X(cna_index_buffer_get_data) \
    X(cna_content_manager_create) \
    X(cna_content_manager_destroy) \
    X(cna_content_manager_get_root_directory_size) \
    X(cna_content_manager_copy_root_directory) \
    X(cna_content_manager_set_root_directory) \
    X(cna_content_manager_unload) \
    X(cna_content_manager_load_texture2d) \
    X(cna_content_manager_get_asset_path_size) \
    X(cna_content_manager_copy_asset_path) \
    X(cna_content_manager_load_sprite_font) \
    X(cna_texture3d_create) \
    X(cna_texture3d_destroy) \
    X(cna_texture3d_get_info) \
    X(cna_texture3d_set_data) \
    X(cna_texture3d_get_data) \
    X(cna_texturecube_create) \
    X(cna_texturecube_destroy) \
    X(cna_texturecube_get_info) \
    X(cna_texturecube_set_data) \
    X(cna_texturecube_get_data) \
    X(cna_sprite_font_get_info) \
    X(cna_sprite_batch_draw_string) \
    X(cna_sprite_batch_begin_with_effect) \
    X(cna_basic_effect_create) \
    X(cna_basic_effect_set_vertex_color_enabled) \
    X(cna_basic_effect_set_prefer_per_pixel_lighting) \
    X(cna_basic_effect_set_diffuse_color) \
    X(cna_basic_effect_set_emissive_color) \
    X(cna_basic_effect_get_specular_color) \
    X(cna_basic_effect_set_specular_color) \
    X(cna_basic_effect_get_specular_power) \
    X(cna_basic_effect_set_specular_power) \
    X(cna_basic_effect_set_alpha) \
    X(cna_basic_effect_set_texture_enabled) \
    X(cna_basic_effect_set_texture) \
    X(cna_effect_matrices_set_world) \
    X(cna_effect_matrices_set_view) \
    X(cna_effect_matrices_set_projection) \
    X(cna_effect_fog_get_color) \
    X(cna_effect_fog_set_color) \
    X(cna_effect_fog_set_enabled) \
    X(cna_effect_fog_set_start) \
    X(cna_effect_fog_set_end) \
    X(cna_effect_lights_set_ambient_color) \
    X(cna_effect_lights_get_directional_light) \
    X(cna_effect_lights_set_enabled) \
    X(cna_directional_light_destroy) \
    X(cna_directional_light_set_diffuse_color) \
    X(cna_directional_light_set_direction) \
    X(cna_directional_light_set_specular_color) \
    X(cna_directional_light_set_enabled) \
    X(cna_alpha_test_effect_create) \
    X(cna_alpha_test_effect_set_diffuse_color) \
    X(cna_alpha_test_effect_set_alpha) \
    X(cna_alpha_test_effect_set_texture) \
    X(cna_alpha_test_effect_set_vertex_color_enabled) \
    X(cna_alpha_test_effect_set_alpha_function) \
    X(cna_alpha_test_effect_set_reference_alpha) \
    X(cna_dual_texture_effect_create) \
    X(cna_dual_texture_effect_set_diffuse_color) \
    X(cna_dual_texture_effect_set_alpha) \
    X(cna_dual_texture_effect_set_texture) \
    X(cna_dual_texture_effect_set_vertex_color_enabled) \
    X(cna_effect_material_create) \
    X(cna_framework_dispatcher_update) \
    X(cna_title_container_read_ext) \
    X(cna_occlusion_query_create) \
    X(cna_occlusion_query_destroy) \
    X(cna_occlusion_query_begin) \
    X(cna_occlusion_query_end) \
    X(cna_occlusion_query_get_is_complete) \
    X(cna_occlusion_query_get_pixel_count) \
    X(cna_vertex_buffer_set_data_raw_at_with_options) \
    X(cna_sound_effect_create_pcm16_range_ext) \
    X(cna_sound_effect_create_from_encoded_ext) \
    X(cna_sound_effect_get_duration_ticks) \
    X(cna_sound_effect_create_instance) \
    X(cna_sound_effect_destroy) \
    X(cna_sound_effect_play) \
    X(cna_sound_effect_play_with_settings) \
    X(cna_sound_effect_set_master_volume) \
    X(cna_sound_effect_set_distance_scale) \
    X(cna_sound_effect_set_doppler_scale) \
    X(cna_sound_effect_set_speed_of_sound) \
    X(cna_sound_effect_instance_play) \
    X(cna_sound_effect_instance_pause) \
    X(cna_sound_effect_instance_resume) \
    X(cna_sound_effect_instance_stop) \
    X(cna_sound_effect_instance_get_info) \
    X(cna_sound_effect_instance_set_volume) \
    X(cna_sound_effect_instance_set_pitch) \
    X(cna_sound_effect_instance_set_pan) \
    X(cna_sound_effect_instance_set_is_looped) \
    X(cna_sound_effect_instance_destroy) \
    X(cna_sound_effect_instance_apply_3d_multi_ext) \
    X(cna_environment_map_effect_create) \
    X(cna_environment_map_effect_set_diffuse_color) \
    X(cna_environment_map_effect_set_emissive_color) \
    X(cna_environment_map_effect_set_alpha) \
    X(cna_environment_map_effect_set_texture) \
    X(cna_environment_map_effect_set_environment_map) \
    X(cna_environment_map_effect_get_amount) \
    X(cna_environment_map_effect_set_amount) \
    X(cna_environment_map_effect_get_specular) \
    X(cna_environment_map_effect_set_specular) \
    X(cna_environment_map_effect_get_fresnel_factor) \
    X(cna_environment_map_effect_set_fresnel_factor) \
    X(cna_skinned_effect_create) \
    X(cna_skinned_effect_set_diffuse_color) \
    X(cna_skinned_effect_set_emissive_color) \
    X(cna_skinned_effect_get_specular_color) \
    X(cna_skinned_effect_set_specular_color) \
    X(cna_skinned_effect_get_specular_power) \
    X(cna_skinned_effect_set_specular_power) \
    X(cna_skinned_effect_set_alpha) \
    X(cna_skinned_effect_set_prefer_per_pixel_lighting) \
    X(cna_skinned_effect_set_texture) \
    X(cna_skinned_effect_set_weights_per_vertex) \
    X(cna_skinned_effect_set_bone_transforms) \
    X(cna_skinned_effect_copy_bone_transforms) \
    X(cna_effect_create_compiled) \
    X(cna_content_manager_load_effect) \
    X(cna_effect_clone) \
    X(cna_effect_destroy) \
    X(cna_effect_apply) \
    X(cna_effect_get_parameters) \
    X(cna_effect_get_techniques) \
    X(cna_effect_get_current_technique) \
    X(cna_effect_set_current_technique) \
    X(cna_effect_technique_collection_get_count) \
    X(cna_effect_technique_collection_get_at) \
    X(cna_effect_technique_destroy) \
    X(cna_effect_technique_get_name_byte_count) \
    X(cna_effect_technique_copy_name) \
    X(cna_effect_technique_get_passes) \
    X(cna_effect_technique_get_annotations) \
    X(cna_effect_technique_collection_destroy) \
    X(cna_effect_pass_collection_get_count) \
    X(cna_effect_pass_collection_get_at) \
    X(cna_effect_pass_collection_destroy) \
    X(cna_effect_pass_destroy) \
    X(cna_effect_pass_get_name_byte_count) \
    X(cna_effect_pass_copy_name) \
    X(cna_effect_pass_get_annotations) \
    X(cna_effect_pass_apply) \
    X(cna_effect_parameter_collection_get_count) \
    X(cna_effect_parameter_collection_get_at) \
    X(cna_effect_parameter_collection_destroy) \
    X(cna_effect_parameter_destroy) \
    X(cna_effect_parameter_get_info) \
    X(cna_effect_parameter_get_name_byte_count) \
    X(cna_effect_parameter_copy_name) \
    X(cna_effect_parameter_get_semantic_byte_count) \
    X(cna_effect_parameter_copy_semantic) \
    X(cna_effect_parameter_get_elements) \
    X(cna_effect_parameter_get_structure_members) \
    X(cna_effect_parameter_get_annotations) \
    X(cna_effect_parameter_get_value) \
    X(cna_effect_parameter_get_values) \
    X(cna_effect_parameter_set_value) \
    X(cna_effect_parameter_set_values) \
    X(cna_effect_parameter_get_value_string_byte_count) \
    X(cna_effect_parameter_copy_value_string) \
    X(cna_effect_parameter_set_value_string) \
    X(cna_effect_parameter_set_value_texture) \
    X(cna_effect_annotation_collection_get_count) \
    X(cna_effect_annotation_collection_get_at) \
    X(cna_effect_annotation_collection_destroy) \
    X(cna_effect_annotation_destroy) \
    X(cna_effect_annotation_get_info) \
    X(cna_effect_annotation_get_name_byte_count) \
    X(cna_effect_annotation_copy_name) \
    X(cna_effect_annotation_get_semantic_byte_count) \
    X(cna_effect_annotation_copy_semantic) \
    X(cna_effect_annotation_get_value_boolean) \
    X(cna_effect_annotation_get_value_int32) \
    X(cna_effect_annotation_get_value_single) \
    X(cna_effect_annotation_get_value_string_byte_count) \
    X(cna_effect_annotation_copy_value_string) \
    X(cna_effect_annotation_get_value_vector2) \
    X(cna_effect_annotation_get_value_vector3) \
    X(cna_effect_annotation_get_value_vector4) \
    X(cna_effect_annotation_get_value_matrix) \
    X(cna_sprite_font_copy_glyphs) \
    X(cna_sprite_font_set_default_character) \
    X(cna_sprite_font_set_line_spacing) \
    X(cna_sprite_font_set_spacing) \
    X(cna_sprite_font_destroy) \
    X(cna_graphics_device_subscribe_event) \
    X(cna_graphics_device_subscribe_resource_created) \
    X(cna_graphics_device_subscribe_resource_destroyed) \
    X(cna_graphics_device_unsubscribe) \
    X(cna_graphics_device_dispose) \
    X(cna_graphics_device_get_texture) \
    X(cna_graphics_device_set_texture) \
    X(cna_graphics_device_get_sampler_state) \
    X(cna_graphics_device_set_sampler_state) \
    X(cna_graphics_device_set_blend_state) \
    X(cna_graphics_device_set_depth_stencil_state) \
    X(cna_graphics_device_set_rasterizer_state) \
    X(cna_render_target2d_create) \
    X(cna_render_target_get_info) \
    X(cna_render_target_destroy) \
    X(cna_graphics_device_set_render_target2d) \
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
