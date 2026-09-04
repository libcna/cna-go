// SPDX-License-Identifier: MS-PL

#ifndef CNA_GO_BRIDGE_H
#define CNA_GO_BRIDGE_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef uint32_t CnaGoResult;
typedef uint64_t CnaGoHandle;

enum {
    CNA_GO_RESULT_SUCCESS = 0,
    /* The two codes the bridge itself can produce, from CNA's own numbering:
       an element array the bridge cannot convert, and a conversion buffer it
       cannot allocate. Both are stated here rather than invented, so a caller
       sees the same code CNA would have used. */
    CNA_GO_RESULT_INVALID_ARGUMENT = 1,
    CNA_GO_RESULT_OUT_OF_MEMORY = 4,
    CNA_GO_RESULT_CALLBACK = 9,

    /* The CNA C ABI admission policy, stated as a RANGE rather than one exact
       encoded number.

       CNA's own live binary contract is the ELF symbol-version node in
       modules/c-api/cmake/CnaCApiExports.map. That file says, in its own
       words, that the node name "is NOT the ABI version and must not be bumped
       with it", that it "changes only for a *major* ABI break", and that
       renaming it "turns every additive release into a hard break". So CNA
       states that a MAJOR bump is the break and a minor bump is additive, and
       docs/releasing.md separately states that CNA_ABI_VERSION "moves when the
       ABI changes, independently of a product release".

       CNA-Go therefore admits: the qualified major, and any minor at or above
       the qualified floor. Below the floor a route CNA-Go binds may simply not
       exist yet; a different major is the break CNA itself names. Every
       required symbol is still resolved by name after the version check, so a
       version inside the range that nevertheless lacks a route is rejected on
       the symbol rather than admitted. */
    CNA_GO_ABI_MAJOR = 0,
    CNA_GO_ABI_MINIMUM_MINOR = 21,
    CNA_GO_ABI_QUALIFIED_PATCH = 0,
    CNA_GO_ABI_QUALIFIED_VERSION = 0x00001500u,
    CNA_GO_CALLBACK_INITIALIZE = 1,
    CNA_GO_CALLBACK_LOAD_CONTENT = 2,
    CNA_GO_CALLBACK_UPDATE = 3,
    CNA_GO_CALLBACK_DRAW = 4,
    CNA_GO_CALLBACK_UNLOAD_CONTENT = 5,
    CNA_GO_CALLBACK_EXITING = 6,

    /* The four optional frame-boundary hooks. They are separate callback
       kinds rather than lifecycle members because CNA_GameCallbacks does not
       carry them: they live in CNA_GameFrameHooks, and CNA-Go installs each
       one only when the callback object supplies the matching override. */
    CNA_GO_CALLBACK_BEGIN_RUN = 7,
    CNA_GO_CALLBACK_END_RUN = 8,
    CNA_GO_CALLBACK_BEGIN_DRAW = 9,
    CNA_GO_CALLBACK_END_DRAW = 10,

    /* Which optional frame hooks cna_go_game_create must install. A bit that
       is clear leaves the corresponding CNA_GameFrameHooks member NULL, and a
       null member is simply not called -- the canonical header says so. */
    CNA_GO_FRAME_HOOK_BEGIN_RUN = 1u << 0,
    CNA_GO_FRAME_HOOK_END_RUN = 1u << 1,
    CNA_GO_FRAME_HOOK_BEGIN_DRAW = 1u << 2,
    CNA_GO_FRAME_HOOK_END_DRAW = 1u << 3,
    CNA_GO_FRAME_HOOK_ALL = 0xFu,

    /* The four canonical CNA game-event identities, mirrored so the Go side
       never spells a CNA_GAME_EVENT_* constant itself. abi_manifest.h holds
       the authoritative values and tools/native_abi static-asserts them
       against the canonical header. */
    CNA_GO_GAME_EVENT_ACTIVATED = 0,
    CNA_GO_GAME_EVENT_DEACTIVATED = 1,
    CNA_GO_GAME_EVENT_DISPOSED = 2,
    CNA_GO_GAME_EVENT_EXITING = 3,
    CNA_GO_GAME_EVENT_COUNT = 4,

    /* The three canonical GameWindow event identities, mirrored the same way.
       They are a SECOND numbering that starts again at zero, so a value from
       one set is a valid-looking value in the other: the two are kept in
       separate enumerations, indexed by separate tables, and asserted against
       both the manifest and the canonical header. */
    CNA_GO_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED = 0,
    CNA_GO_GAME_WINDOW_EVENT_ORIENTATION_CHANGED = 1,
    CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED = 2,
    CNA_GO_GAME_WINDOW_EVENT_COUNT = 3,

    /* The five canonical GraphicsDeviceManager event identities. This family
       does NOT start its device events at zero: DISPOSED is 0 and
       DEVICE_CREATED is 1, unlike the game and window families whose first
       identity is a device-independent one. A table indexed as if the three
       agreed would be off by one. */
    CNA_GO_GDM_EVENT_DISPOSED = 0,
    CNA_GO_GDM_EVENT_DEVICE_CREATED = 1,
    CNA_GO_GDM_EVENT_DEVICE_DISPOSING = 2,
    CNA_GO_GDM_EVENT_DEVICE_RESET = 3,
    CNA_GO_GDM_EVENT_DEVICE_RESETTING = 4,
    CNA_GO_GDM_EVENT_COUNT = 5
};

/* The device family's own identities, mirroring CNA_GRAPHICS_DEVICE_EVENT_* for
   the four payload-free events and adding the two that carry one. The order is
   CNA-Go's, and the four that mirror CNA are static-asserted against it. */
enum {
    CNA_GO_DEVICE_EVENT_DISPOSING = 0,
    CNA_GO_DEVICE_EVENT_DEVICE_LOST = 1,
    CNA_GO_DEVICE_EVENT_DEVICE_RESET = 2,
    CNA_GO_DEVICE_EVENT_DEVICE_RESETTING = 3,
    CNA_GO_DEVICE_EVENT_RESOURCE_CREATED = 4,
    CNA_GO_DEVICE_EVENT_RESOURCE_DESTROYED = 5,
    CNA_GO_DEVICE_EVENT_COUNT = 6
};

/* The encoded-version arithmetic, mirrored from CNA_ABI_VERSION_ENCODE in the
   canonical CNA header. tools/native_abi compiles both spellings in one
   translation unit and asserts they agree on sample values, so a mirror that
   drifted is a compile error rather than a version silently decoded wrong. */
#define CNA_GO_ABI_ENCODE(major, minor, patch) \
    ((((uint32_t)(major) & UINT32_C(0xFFFF)) << 16) | \
     (((uint32_t)(minor) & UINT32_C(0xFF)) << 8) | \
     ((uint32_t)(patch) & UINT32_C(0xFF)))
#define CNA_GO_ABI_MAJOR_OF(version) ((uint32_t)(version) >> 16)
#define CNA_GO_ABI_MINOR_OF(version) (((uint32_t)(version) >> 8) & UINT32_C(0xFF))
#define CNA_GO_ABI_PATCH_OF(version) ((uint32_t)(version) & UINT32_C(0xFF))

int cna_go_open(const char* path, char* error_buffer, size_t error_capacity);
void cna_go_close(void);
uint32_t cna_go_abi_version(void);

/* Non-zero when an encoded ABI version satisfies the admission policy above.
   This is the ONE place the policy is evaluated; Go asks this function rather
   than re-deriving the comparison, so the two cannot disagree. */
int cna_go_abi_admits(uint32_t version);

/* Function forms of the decoding macros. cgo cannot evaluate a function-like
   macro from Go, so Go reads the parts through these. */
uint32_t cna_go_abi_encode(uint32_t major, uint32_t minor, uint32_t patch);
uint32_t cna_go_abi_major_of(uint32_t version);
uint32_t cna_go_abi_minor_of(uint32_t version);
uint32_t cna_go_abi_patch_of(uint32_t version);

/* Non-zero when every resolved function pointer belongs to the symbol whose
   name the manifest lists, proven with dladdr rather than assumed from the
   resolution macro. Fills out_detail with the first disagreement. */
int cna_go_verify_symbol_identity(char* out_detail, size_t detail_capacity);
uint64_t cna_go_owner_thread_id(void);
uint32_t cna_go_bound_function_count(void);
const char* cna_go_bound_function_name(uint32_t index);
int cna_go_has_loaded_symbol(const char* name);
size_t cna_go_last_error_message(char* destination, size_t capacity);

/* frame_hook_overrides is a mask of CNA_GO_FRAME_HOOK_*. The `initialize`
   hook is always installed -- it is the position Game::Initialize occupies and
   is not an optional override -- and each of the other four members is
   assigned if and only if its bit is set. */
/* The Game's configured timing and presentation state, applied at creation.
   CNA_GameCreateInfo carries the first two; the other two have no creation
   field, so cna_go_game_create pushes them immediately after the game exists
   -- on the same owner thread, before anything can observe a frame. */
typedef struct CnaGoGameTiming {
    int64_t target_elapsed_time_ticks;
    int64_t inactive_sleep_time_ticks;
    uint8_t is_fixed_time_step;
    uint8_t is_mouse_visible;
} CnaGoGameTiming;

CnaGoResult cna_go_game_create(
    uintptr_t context,
    const char* title,
    uint64_t title_length,
    uint32_t frame_hook_overrides,
    const CnaGoGameTiming* timing,
    CnaGoHandle* out_game);

CnaGoResult cna_go_game_set_is_mouse_visible(CnaGoHandle game, uint8_t visible);
CnaGoResult cna_go_game_set_is_fixed_time_step(CnaGoHandle game, uint8_t fixed);
CnaGoResult cna_go_game_set_target_elapsed_time_ticks(CnaGoHandle game, int64_t ticks);
CnaGoResult cna_go_game_set_inactive_sleep_time_ticks(CnaGoHandle game, int64_t ticks);
CnaGoResult cna_go_game_reset_elapsed_time(CnaGoHandle game);
CnaGoResult cna_go_game_suppress_draw(CnaGoHandle game);
CnaGoResult cna_go_game_run(CnaGoHandle game);

/* The two frame steps. cna_game_tick is the canonical step and does not
   initialize; cna_game_run_one_frame initializes on first use and then wraps
   it. Both are refused by CNA from inside a lifecycle callback. */
CnaGoResult cna_go_game_tick(CnaGoHandle game);
CnaGoResult cna_go_game_run_one_frame(CnaGoHandle game);
CnaGoResult cna_go_game_request_exit(CnaGoHandle game);
CnaGoResult cna_go_game_destroy(CnaGoHandle game);

/* Installs exactly one native subscription per canonical game event.
   out_registrations must point at CNA_GO_GAME_EVENT_COUNT handles; every
   handle it fills is owned by the caller and must be released with
   cna_go_game_unsubscribe_events. A partial failure releases what it already
   installed and reports the first failing result. */
CnaGoResult cna_go_game_subscribe_events(
    CnaGoHandle game,
    uintptr_t context,
    CnaGoHandle* out_registrations);

/* Releases every non-zero registration handle and zeroes the slot. Returns the
   first failing result and still releases the remaining handles. */
CnaGoResult cna_go_game_unsubscribe_events(CnaGoHandle* registrations);

CnaGoResult cna_go_graphics_device_manager_create(CnaGoHandle game, CnaGoHandle* out_manager);
CnaGoResult cna_go_graphics_device_manager_get_device(CnaGoHandle manager, CnaGoHandle* out_device);
CnaGoResult cna_go_graphics_device_manager_destroy(CnaGoHandle manager);

/* GraphicsDeviceManager's configuration setters and the one command that
   applies them. Only setters: every reference getter is a managed field read. */
CnaGoResult cna_go_graphics_device_manager_set_graphics_profile(CnaGoHandle manager, uint32_t profile);
CnaGoResult cna_go_graphics_device_manager_set_is_full_screen(CnaGoHandle manager, uint8_t full_screen);
CnaGoResult cna_go_graphics_device_manager_set_prefer_multi_sampling(CnaGoHandle manager, uint8_t prefer);
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_format(CnaGoHandle manager, uint32_t format);
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_width(CnaGoHandle manager, int32_t width);
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_height(CnaGoHandle manager, int32_t height);
CnaGoResult cna_go_graphics_device_manager_set_preferred_depth_stencil_format(CnaGoHandle manager, uint32_t format);
CnaGoResult cna_go_graphics_device_manager_set_synchronize_with_vertical_retrace(CnaGoHandle manager, uint8_t synchronize);
CnaGoResult cna_go_graphics_device_manager_set_supported_orientations(CnaGoHandle manager, uint32_t orientations);
CnaGoResult cna_go_graphics_device_manager_apply_changes(CnaGoHandle manager);
CnaGoResult cna_go_graphics_device_manager_create_device(CnaGoHandle manager);
CnaGoResult cna_go_graphics_device_manager_begin_draw(CnaGoHandle manager, uint8_t* out_should_draw);
CnaGoResult cna_go_graphics_device_manager_end_draw(CnaGoHandle manager);

/* Installs exactly one native subscription per canonical manager event. The
   context is a per-MANAGER handle rather than the runtime's, because a game may
   in principle hold more than one manager object and a signal has to reach the
   one that was subscribed. */
CnaGoResult cna_go_graphics_device_manager_subscribe_events(
    CnaGoHandle manager,
    uintptr_t context,
    CnaGoHandle* out_registrations);
CnaGoResult cna_go_graphics_device_manager_unsubscribe_events(CnaGoHandle* registrations);

/* Installs exactly one native subscription per canonical DEVICE event. Four
   carry no payload and two do, so the six go through three CNA routes and one
   registration array. */
CnaGoResult cna_go_graphics_device_subscribe_events(
    CnaGoHandle device,
    uintptr_t context,
    CnaGoHandle* out_registrations);
CnaGoResult cna_go_graphics_device_unsubscribe_events(CnaGoHandle* registrations);
CnaGoResult cna_go_graphics_device_dispose(CnaGoHandle device);

CnaGoResult cna_go_index_buffer_create(
    CnaGoHandle device,
    int32_t index_count,
    uint32_t index_element_size,
    uint32_t buffer_usage,
    uint8_t dynamic,
    CnaGoHandle* out_index_buffer);
CnaGoResult cna_go_index_buffer_destroy(CnaGoHandle index_buffer);
CnaGoResult cna_go_index_buffer_get_info(
    CnaGoHandle index_buffer,
    int32_t* out_index_count,
    uint32_t* out_index_element_size,
    uint32_t* out_buffer_usage,
    uint8_t* out_dynamic,
    uint8_t* out_is_content_lost,
    uint8_t* out_has_renderer);
CnaGoResult cna_go_index_buffer_set_data(
    CnaGoHandle index_buffer,
    uint32_t index_element_size,
    uint32_t options,
    uint64_t start_index,
    uint64_t element_count,
    const void* data,
    uint64_t capacity);
CnaGoResult cna_go_index_buffer_set_data_at(
    CnaGoHandle index_buffer,
    uint64_t buffer_offset_in_bytes,
    uint32_t index_element_size,
    uint32_t options,
    uint64_t start_index,
    uint64_t element_count,
    const void* data,
    uint64_t capacity);
CnaGoResult cna_go_index_buffer_get_data(
    CnaGoHandle index_buffer,
    uint32_t index_element_size,
    uint64_t start_index,
    uint64_t element_count,
    void* destination,
    uint64_t capacity,
    uint64_t* out_element_count);

CnaGoResult cna_go_content_manager_create(
    CnaGoHandle device,
    const char* root_directory,
    uint64_t root_directory_length,
    CnaGoHandle* out_content_manager);
CnaGoResult cna_go_content_manager_destroy(CnaGoHandle content_manager);
CnaGoResult cna_go_content_manager_get_root_directory_size(CnaGoHandle content_manager, uint64_t* out_bytes);
CnaGoResult cna_go_content_manager_copy_root_directory(
    CnaGoHandle content_manager, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_content_manager_set_root_directory(
    CnaGoHandle content_manager, const char* root_directory, uint64_t root_directory_length);
CnaGoResult cna_go_content_manager_unload(CnaGoHandle content_manager);
CnaGoResult cna_go_content_manager_load_texture2d(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, CnaGoHandle* out_texture);
CnaGoResult cna_go_content_manager_get_asset_path_size(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, uint64_t* out_bytes);
CnaGoResult cna_go_content_manager_copy_asset_path(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length,
    char* destination, uint64_t capacity, uint64_t* out_bytes);

/* Foundation 69. The SpriteFont family. The loader reports TWO owned handles
   for one asset -- the font and its glyph atlas -- because CNA retains the
   atlas for as long as the font lives and destroying the font first is the
   documented order. */
/* Foundation 71. The volume and cube texture families. Both transfer routes
   take CNA_Color elements ONLY -- unlike cna_texture2d_set_data, which takes a
   CNA_TextureDataType identity -- so the element set these can express is one
   type wide, and the Graphics package refuses any other by name. */
CnaGoResult cna_go_texture3d_create(
    CnaGoHandle device, uint32_t width, uint32_t height, uint32_t depth,
    uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture);
CnaGoResult cna_go_texture3d_destroy(CnaGoHandle texture);
CnaGoResult cna_go_texture3d_get_info(
    CnaGoHandle texture, uint32_t* out_width, uint32_t* out_height, uint32_t* out_depth,
    uint32_t* out_level_count, uint32_t* out_format);
CnaGoResult cna_go_texture3d_set_data(
    CnaGoHandle texture, int32_t level,
    int32_t left, int32_t top, int32_t right, int32_t bottom, int32_t front, int32_t back,
    uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity);
CnaGoResult cna_go_texture3d_get_data(
    CnaGoHandle texture, int32_t level,
    int32_t left, int32_t top, int32_t right, int32_t bottom, int32_t front, int32_t back,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity,
    uint64_t* out_required);
CnaGoResult cna_go_texturecube_create(
    CnaGoHandle device, uint32_t size, uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture);
CnaGoResult cna_go_texturecube_destroy(CnaGoHandle texture);
CnaGoResult cna_go_texturecube_get_info(
    CnaGoHandle texture, uint32_t* out_size, uint32_t* out_level_count, uint32_t* out_format);
CnaGoResult cna_go_texturecube_set_data(
    CnaGoHandle texture, uint32_t face, int32_t level, uint8_t has_rectangle,
    int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity);
CnaGoResult cna_go_texturecube_get_data(
    CnaGoHandle texture, uint32_t face, int32_t level, uint8_t has_rectangle,
    int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity,
    uint64_t* out_required);

CnaGoResult cna_go_content_manager_load_sprite_font(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length,
    CnaGoHandle* out_sprite_font, CnaGoHandle* out_texture);
CnaGoResult cna_go_sprite_font_get_info(
    CnaGoHandle sprite_font,
    uint64_t* out_character_count,
    int32_t* out_line_spacing,
    float* out_spacing,
    uint16_t* out_default_character,
    uint8_t* out_has_default_character);
CnaGoResult cna_go_sprite_font_copy_glyphs(
    CnaGoHandle sprite_font,
    uint64_t capacity,
    uint16_t* out_characters,
    int32_t* out_rectangles,
    float* out_kerning,
    uint64_t* out_count);
CnaGoResult cna_go_sprite_font_set_default_character(
    CnaGoHandle sprite_font, uint8_t has_value, uint16_t value);
CnaGoResult cna_go_sprite_font_set_line_spacing(CnaGoHandle sprite_font, int32_t line_spacing);
CnaGoResult cna_go_sprite_font_set_spacing(CnaGoHandle sprite_font, float spacing);
CnaGoResult cna_go_sprite_font_destroy(CnaGoHandle sprite_font);

/* Foundation 73 -- device reset, presentation parameters and back-buffer
   readback. The eleven presentation fields cross as scalars, which is the rule
   every other family here follows. */
CnaGoResult cna_go_render_target_cube_create(
    CnaGoHandle device, uint32_t size, uint8_t mip_map, uint32_t format, uint32_t depth_format,
    int32_t multi_sample_count, uint32_t usage, CnaGoHandle* out_render_target);
CnaGoResult cna_go_graphics_device_set_render_target_cube(
    CnaGoHandle device, CnaGoHandle render_target, uint32_t cube_map_face);
CnaGoResult cna_go_graphics_device_set_render_targets(
    CnaGoHandle device, const CnaGoHandle* handles, const uint32_t* faces, uint64_t count);
CnaGoResult cna_go_graphics_device_get_render_target_count(CnaGoHandle device, uint64_t* out_count);
CnaGoResult cna_go_graphics_device_create(
    uint32_t adapter_index, uint32_t graphics_profile, const int32_t* ints,
    uint8_t is_full_screen, uint8_t headless, CnaGoHandle* out_device);
CnaGoResult cna_go_graphics_device_destroy(CnaGoHandle device);
CnaGoResult cna_go_graphics_device_reset(CnaGoHandle device);
CnaGoResult cna_go_graphics_device_reset_with_parameters(
    CnaGoHandle device, const int32_t* ints, uint8_t is_full_screen, uint8_t headless,
    uint8_t has_adapter, uint32_t adapter_index);
CnaGoResult cna_go_graphics_device_get_presentation_parameters(
    CnaGoHandle device, int32_t* out_ints, uint8_t* out_is_full_screen, uint8_t* out_headless);
CnaGoResult cna_go_graphics_device_get_backbuffer_data_window(
    CnaGoHandle device, uint8_t has_rectangle, int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity);

/* Foundation 73 -- the six user-primitive draws. The two CNA structures are
   filled on the C side and everything crosses cgo as scalars plus two caller
   pointers, which is the rule every other family here follows. */
CnaGoResult cna_go_graphics_device_draw_user_primitives(
    CnaGoHandle device, uint32_t primitive_type, uint32_t vertex_source,
    const void* vertex_data, CnaGoHandle vertex_declaration,
    int32_t vertex_offset, int32_t num_vertices, int32_t primitive_count);
CnaGoResult cna_go_graphics_device_draw_user_indexed_primitives(
    CnaGoHandle device, uint32_t primitive_type, uint32_t vertex_source,
    const void* vertex_data, CnaGoHandle vertex_declaration,
    int32_t vertex_offset, int32_t num_vertices, int32_t primitive_count,
    uint32_t index_element_size, int32_t index_offset, const void* index_data);

/* Foundation 72 -- the Effect cluster. */
CnaGoResult cna_go_sprite_batch_begin_with_effect(
    CnaGoHandle batch,
    uint32_t sort_mode,
    const uint32_t* blend,
    const int32_t* blend_mask,
    const uint8_t* blend_factor,
    const uint32_t* sampler,
    const int32_t* sampler_ints,
    float sampler_bias,
    const uint8_t* depth_flags,
    const uint32_t* depth_words,
    const int32_t* depth_ints,
    uint32_t cull_mode,
    uint32_t fill_mode,
    float depth_bias,
    float slope_scale_depth_bias,
    uint8_t multi_sample_anti_alias,
    uint8_t scissor_test_enable,
    CnaGoHandle effect,
    uint8_t has_transform,
    const float* transform);
CnaGoResult cna_go_effect_create_compiled(
    CnaGoHandle device, const uint8_t* effect_code, uint64_t effect_code_count, CnaGoHandle* out_effect);
CnaGoResult cna_go_content_manager_load_effect(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, CnaGoHandle* out_effect);
CnaGoResult cna_go_effect_string(
    uint32_t kind, CnaGoHandle handle, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_effect_parameter_get_info(
    CnaGoHandle parameter, int32_t* out_rows, int32_t* out_columns,
    uint32_t* out_class, uint32_t* out_type);
CnaGoResult cna_go_effect_annotation_get_info(
    CnaGoHandle annotation, int32_t* out_rows, int32_t* out_columns,
    uint32_t* out_class, uint32_t* out_type);
CnaGoResult cna_go_effect_parameter_set_value_string(
    CnaGoHandle parameter, const char* value, uint64_t value_length);
CnaGoResult cna_go_effect_annotation_get_value_vector(
    CnaGoHandle annotation, uint32_t width, float* out_values);
CnaGoResult cna_go_effect_annotation_get_value_matrix(CnaGoHandle annotation, float* out_values);
/* Foundation 79 -- the stock-effect routes. */
CnaGoResult cna_go_basic_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect);
CnaGoResult cna_go_basic_effect_set_vertex_color_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_basic_effect_set_prefer_per_pixel_lighting(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_basic_effect_set_diffuse_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_basic_effect_set_emissive_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_basic_effect_get_specular_color(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_basic_effect_set_specular_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_basic_effect_get_specular_power(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_basic_effect_set_specular_power(CnaGoHandle effect, float value);
CnaGoResult cna_go_basic_effect_set_alpha(CnaGoHandle effect, float value);
CnaGoResult cna_go_basic_effect_set_texture_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_basic_effect_set_texture(CnaGoHandle effect, CnaGoHandle texture);
CnaGoResult cna_go_effect_matrices_set_world(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_effect_matrices_set_view(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_effect_matrices_set_projection(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_effect_fog_get_color(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_effect_fog_set_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_effect_fog_set_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_effect_fog_set_start(CnaGoHandle effect, float value);
CnaGoResult cna_go_effect_fog_set_end(CnaGoHandle effect, float value);
CnaGoResult cna_go_effect_lights_set_ambient_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_effect_lights_get_directional_light(CnaGoHandle effect, uint32_t index, CnaGoHandle* out_light);
CnaGoResult cna_go_effect_lights_set_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_directional_light_destroy(CnaGoHandle light);
CnaGoResult cna_go_directional_light_set_diffuse_color(CnaGoHandle light, const float* value);
CnaGoResult cna_go_directional_light_set_direction(CnaGoHandle light, const float* value);
CnaGoResult cna_go_directional_light_set_specular_color(CnaGoHandle light, const float* value);
CnaGoResult cna_go_directional_light_set_enabled(CnaGoHandle light, uint8_t value);

/* Foundation 80. */
CnaGoResult cna_go_alpha_test_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect);
CnaGoResult cna_go_alpha_test_effect_set_diffuse_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_alpha_test_effect_set_alpha(CnaGoHandle effect, float value);
CnaGoResult cna_go_alpha_test_effect_set_texture(CnaGoHandle effect, CnaGoHandle texture);
CnaGoResult cna_go_alpha_test_effect_set_vertex_color_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_alpha_test_effect_set_alpha_function(CnaGoHandle effect, uint32_t value);
CnaGoResult cna_go_alpha_test_effect_set_reference_alpha(CnaGoHandle effect, int32_t value);
CnaGoResult cna_go_dual_texture_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect);
CnaGoResult cna_go_dual_texture_effect_set_diffuse_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_dual_texture_effect_set_alpha(CnaGoHandle effect, float value);
CnaGoResult cna_go_dual_texture_effect_set_texture(CnaGoHandle effect, uint32_t texture_index, CnaGoHandle texture);
CnaGoResult cna_go_dual_texture_effect_set_vertex_color_enabled(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_effect_material_create(CnaGoHandle clone_source, CnaGoHandle* out_effect);

/* Foundation 82. */
CnaGoResult cna_go_framework_dispatcher_update(CnaGoHandle game);
CnaGoResult cna_go_title_container_read(CnaGoHandle game, const char* name, uint64_t name_length, uint8_t* destination, uint64_t capacity, uint64_t* out_bytes);

/* Foundation 83. */
CnaGoResult cna_go_occlusion_query_create(CnaGoHandle graphics_device, CnaGoHandle* out_occlusion_query);
CnaGoResult cna_go_occlusion_query_destroy(CnaGoHandle occlusion_query);
CnaGoResult cna_go_occlusion_query_begin(CnaGoHandle occlusion_query);
CnaGoResult cna_go_occlusion_query_end(CnaGoHandle occlusion_query);
CnaGoResult cna_go_occlusion_query_get_is_complete(CnaGoHandle occlusion_query, uint8_t* out_is_complete);
CnaGoResult cna_go_occlusion_query_get_pixel_count(CnaGoHandle occlusion_query, int32_t* out_pixel_count);

/* Foundation 84. */
CnaGoResult cna_go_vertex_buffer_set_data_raw_at_with_options(CnaGoHandle vertex_buffer, uint64_t buffer_offset_in_bytes, const void* data, uint64_t data_byte_count, uint64_t vertex_count, uint32_t vertex_stride, uint32_t options);

/* Foundation 81. */
CnaGoResult cna_go_environment_map_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect);
CnaGoResult cna_go_environment_map_effect_set_diffuse_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_environment_map_effect_set_emissive_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_environment_map_effect_set_alpha(CnaGoHandle effect, float value);
CnaGoResult cna_go_environment_map_effect_set_texture(CnaGoHandle effect, CnaGoHandle texture);
CnaGoResult cna_go_environment_map_effect_set_environment_map(CnaGoHandle effect, CnaGoHandle environment_map);
CnaGoResult cna_go_environment_map_effect_get_amount(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_environment_map_effect_set_amount(CnaGoHandle effect, float value);
CnaGoResult cna_go_environment_map_effect_get_specular(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_environment_map_effect_set_specular(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_environment_map_effect_get_fresnel_factor(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_environment_map_effect_set_fresnel_factor(CnaGoHandle effect, float value);
CnaGoResult cna_go_skinned_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect);
CnaGoResult cna_go_skinned_effect_set_diffuse_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_skinned_effect_set_emissive_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_skinned_effect_get_specular_color(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_skinned_effect_set_specular_color(CnaGoHandle effect, const float* value);
CnaGoResult cna_go_skinned_effect_get_specular_power(CnaGoHandle effect, float* out_value);
CnaGoResult cna_go_skinned_effect_set_specular_power(CnaGoHandle effect, float value);
CnaGoResult cna_go_skinned_effect_set_alpha(CnaGoHandle effect, float value);
CnaGoResult cna_go_skinned_effect_set_prefer_per_pixel_lighting(CnaGoHandle effect, uint8_t value);
CnaGoResult cna_go_skinned_effect_set_texture(CnaGoHandle effect, CnaGoHandle texture);
CnaGoResult cna_go_skinned_effect_set_weights_per_vertex(CnaGoHandle effect, int32_t value);
CnaGoResult cna_go_skinned_effect_set_bone_transforms(CnaGoHandle effect, const float* transforms, uint64_t transform_count);
CnaGoResult cna_go_skinned_effect_copy_bone_transforms(CnaGoHandle effect, uint64_t requested_count, float* destination, uint64_t capacity, uint64_t* out_count);

CnaGoResult cna_go_effect_apply(CnaGoHandle effect);
CnaGoResult cna_go_effect_destroy(CnaGoHandle effect);
CnaGoResult cna_go_effect_clone(CnaGoHandle effect, CnaGoHandle* out_clone);
CnaGoResult cna_go_effect_get_parameters(CnaGoHandle effect, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_get_techniques(CnaGoHandle effect, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_get_current_technique(CnaGoHandle effect, CnaGoHandle* out_technique);
CnaGoResult cna_go_effect_set_current_technique(CnaGoHandle effect, CnaGoHandle technique);
CnaGoResult cna_go_effect_technique_collection_get_count(CnaGoHandle collection, uint64_t* out_count);
CnaGoResult cna_go_effect_technique_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_technique);
CnaGoResult cna_go_effect_technique_collection_destroy(CnaGoHandle collection);
CnaGoResult cna_go_effect_technique_destroy(CnaGoHandle technique);
CnaGoResult cna_go_effect_technique_get_passes(CnaGoHandle technique, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_technique_get_annotations(CnaGoHandle technique, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_pass_collection_get_count(CnaGoHandle collection, uint64_t* out_count);
CnaGoResult cna_go_effect_pass_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_pass);
CnaGoResult cna_go_effect_pass_collection_destroy(CnaGoHandle collection);
CnaGoResult cna_go_effect_pass_destroy(CnaGoHandle pass);
CnaGoResult cna_go_effect_pass_get_annotations(CnaGoHandle pass, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_pass_apply(CnaGoHandle pass);
CnaGoResult cna_go_effect_parameter_collection_get_count(CnaGoHandle collection, uint64_t* out_count);
CnaGoResult cna_go_effect_parameter_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_parameter);
CnaGoResult cna_go_effect_parameter_collection_destroy(CnaGoHandle collection);
CnaGoResult cna_go_effect_parameter_destroy(CnaGoHandle parameter);
CnaGoResult cna_go_effect_parameter_get_elements(CnaGoHandle parameter, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_parameter_get_structure_members(CnaGoHandle parameter, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_parameter_get_annotations(CnaGoHandle parameter, CnaGoHandle* out_collection);
CnaGoResult cna_go_effect_parameter_get_value(CnaGoHandle parameter, uint32_t value_type, void* out_value);
CnaGoResult cna_go_effect_parameter_get_values(CnaGoHandle parameter, uint32_t value_type, uint64_t requested, void* destination, uint64_t capacity, uint64_t* out_count);
CnaGoResult cna_go_effect_parameter_set_value(CnaGoHandle parameter, uint32_t value_type, const void* value);
CnaGoResult cna_go_effect_parameter_set_values(CnaGoHandle parameter, uint32_t value_type, const void* values, uint64_t count);
CnaGoResult cna_go_effect_parameter_set_value_texture(CnaGoHandle parameter, uint32_t texture_type, CnaGoHandle texture);
CnaGoResult cna_go_effect_annotation_collection_get_count(CnaGoHandle collection, uint64_t* out_count);
CnaGoResult cna_go_effect_annotation_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_annotation);
CnaGoResult cna_go_effect_annotation_collection_destroy(CnaGoHandle collection);
CnaGoResult cna_go_effect_annotation_destroy(CnaGoHandle annotation);
CnaGoResult cna_go_effect_annotation_get_value_boolean(CnaGoHandle annotation, uint8_t* out_value);
CnaGoResult cna_go_effect_annotation_get_value_int32(CnaGoHandle annotation, int32_t* out_value);
CnaGoResult cna_go_effect_annotation_get_value_single(CnaGoHandle annotation, float* out_value);

CnaGoResult cna_go_sprite_batch_draw_string(
    CnaGoHandle sprite_batch,
    CnaGoHandle sprite_font,
    const char* text,
    uint64_t text_length,
    float position_x,
    float position_y,
    uint8_t red,
    uint8_t green,
    uint8_t blue,
    uint8_t alpha,
    float rotation,
    float origin_x,
    float origin_y,
    float scale_x,
    float scale_y,
    uint32_t effects,
    float layer_depth);
CnaGoResult cna_go_game_get_graphics_device(CnaGoHandle game, CnaGoHandle* out_device);
CnaGoResult cna_go_graphics_device_get_viewport(
    CnaGoHandle device,
    int32_t* x,
    int32_t* y,
    int32_t* width,
    int32_t* height,
    float* min_depth,
    float* max_depth);
CnaGoResult cna_go_graphics_device_get_blend_factor(CnaGoHandle device, uint8_t* out_r, uint8_t* out_g, uint8_t* out_b, uint8_t* out_a);
CnaGoResult cna_go_graphics_device_set_blend_factor(CnaGoHandle device, uint8_t r, uint8_t g, uint8_t b, uint8_t a);
CnaGoResult cna_go_graphics_device_get_multi_sample_mask(CnaGoHandle device, int32_t* out_mask);
CnaGoResult cna_go_graphics_device_set_multi_sample_mask(CnaGoHandle device, int32_t mask);
CnaGoResult cna_go_graphics_device_get_reference_stencil(CnaGoHandle device, int32_t* out_stencil);
CnaGoResult cna_go_graphics_device_set_reference_stencil(CnaGoHandle device, int32_t stencil);
CnaGoResult cna_go_graphics_device_get_scissor_rectangle(CnaGoHandle device, int32_t* out_x, int32_t* out_y, int32_t* out_width, int32_t* out_height);
CnaGoResult cna_go_graphics_device_set_scissor_rectangle(CnaGoHandle device, int32_t x, int32_t y, int32_t width, int32_t height);
CnaGoResult cna_go_graphics_device_set_viewport(CnaGoHandle device, int32_t x, int32_t y, int32_t width, int32_t height, float min_depth, float max_depth);
CnaGoResult cna_go_graphics_device_get_graphics_profile(CnaGoHandle device, uint32_t* out_profile);
CnaGoResult cna_go_graphics_device_get_status(CnaGoHandle device, uint32_t* out_status);
CnaGoResult cna_go_graphics_device_get_is_disposed(CnaGoHandle device, uint8_t* out_is_disposed);

// The four state descriptors cross the boundary as FLAT SCALARS, like every
// other CNA structure here: the bridge builds the versioned C POD on its own
// side and cgo never sees one.
CnaGoResult cna_go_graphics_device_get_texture(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    uint8_t* out_bound,
    CnaGoHandle* out_texture);

CnaGoResult cna_go_graphics_device_set_texture(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    CnaGoHandle texture);

CnaGoResult cna_go_graphics_device_get_sampler_state(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    uint32_t* out_words,
    int32_t* out_ints,
    float* out_bias);

CnaGoResult cna_go_graphics_device_set_sampler_state(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    const uint32_t* words,
    const int32_t* ints,
    float bias);

CnaGoResult cna_go_graphics_device_set_blend_state(
    CnaGoHandle device,
    uint32_t alpha_blend_function,
    uint32_t alpha_destination_blend,
    uint32_t alpha_source_blend,
    uint32_t color_blend_function,
    uint32_t color_destination_blend,
    uint32_t color_source_blend,
    uint32_t color_write_channels,
    uint32_t color_write_channels1,
    uint32_t color_write_channels2,
    uint32_t color_write_channels3,
    uint8_t blend_factor_r,
    uint8_t blend_factor_g,
    uint8_t blend_factor_b,
    uint8_t blend_factor_a,
    int32_t multi_sample_mask);

CnaGoResult cna_go_graphics_device_set_depth_stencil_state(
    CnaGoHandle device,
    uint8_t depth_buffer_enable,
    uint8_t depth_buffer_write_enable,
    uint8_t stencil_enable,
    uint8_t two_sided_stencil_mode,
    uint32_t depth_buffer_function,
    uint32_t stencil_function,
    int32_t stencil_mask,
    int32_t stencil_write_mask,
    int32_t reference_stencil,
    uint32_t stencil_fail,
    uint32_t stencil_depth_buffer_fail,
    uint32_t stencil_pass,
    uint32_t counter_clockwise_stencil_function,
    uint32_t counter_clockwise_stencil_fail,
    uint32_t counter_clockwise_stencil_depth_buffer_fail,
    uint32_t counter_clockwise_stencil_pass);

CnaGoResult cna_go_graphics_device_set_rasterizer_state(
    CnaGoHandle device,
    uint32_t cull_mode,
    uint32_t fill_mode,
    float depth_bias,
    float slope_scale_depth_bias,
    uint8_t multi_sample_anti_alias,
    uint8_t scissor_test_enable);


CnaGoResult cna_go_render_target2d_create(
    CnaGoHandle device,
    uint32_t width,
    uint32_t height,
    uint8_t mip_map,
    uint32_t format,
    uint32_t depth_format,
    int32_t multi_sample_count,
    uint32_t usage,
    CnaGoHandle* out_render_target);

CnaGoResult cna_go_render_target_get_info(
    CnaGoHandle render_target,
    uint32_t* out_kind,
    uint32_t* out_width,
    uint32_t* out_height,
    uint32_t* out_level_count,
    uint32_t* out_format,
    uint32_t* out_depth_format,
    int32_t* out_multi_sample_count,
    uint32_t* out_usage,
    uint8_t* out_is_content_lost,
    uint8_t* out_renderer_available);

CnaGoResult cna_go_render_target_destroy(CnaGoHandle render_target);

CnaGoResult cna_go_graphics_device_set_render_target2d(CnaGoHandle device, CnaGoHandle render_target);
CnaGoResult cna_go_graphics_device_clear_options(CnaGoHandle device, uint32_t options, uint8_t r, uint8_t g, uint8_t b, uint8_t a, float depth, int32_t stencil);
CnaGoResult cna_go_graphics_device_present(CnaGoHandle device);
CnaGoResult cna_go_graphics_device_get_display_mode(CnaGoHandle device, int32_t* out_width, int32_t* out_height, float* out_aspect_ratio, uint32_t* out_format);
CnaGoResult cna_go_texture2d_create(CnaGoHandle device, uint32_t width, uint32_t height, uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture);
CnaGoResult cna_go_texture2d_create_from_encoded_memory_sized(CnaGoHandle device, const uint8_t* data, uint64_t byte_count, uint32_t width, uint32_t height, uint8_t zoom, CnaGoHandle* out_texture);
CnaGoResult cna_go_texture2d_get_encoded_byte_count(CnaGoHandle texture, uint32_t image_format, uint32_t width, uint32_t height, uint64_t* out_byte_count);
CnaGoResult cna_go_texture2d_copy_encoded(CnaGoHandle texture, uint32_t image_format, uint32_t width, uint32_t height, uint8_t* destination, uint64_t capacity, uint64_t* out_byte_count);
CnaGoResult cna_go_texture2d_set_data(CnaGoHandle texture, uint32_t data_type, int32_t level, uint8_t has_rectangle, int32_t rect_x, int32_t rect_y, int32_t rect_width, int32_t rect_height, uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity);
CnaGoResult cna_go_texture2d_get_data(CnaGoHandle texture, uint32_t data_type, int32_t level, uint8_t has_rectangle, int32_t rect_x, int32_t rect_y, int32_t rect_width, int32_t rect_height, uint64_t start_index, uint64_t element_count, void* destination, uint64_t destination_capacity, uint64_t* out_required_elements);
CnaGoResult cna_go_graphics_device_clear_rgba(
    CnaGoHandle device,
    float r,
    float g,
    float b,
    float a);

CnaGoResult cna_go_texture2d_create_from_encoded_memory(
    CnaGoHandle device,
    const uint8_t* data,
    uint64_t byte_count,
    CnaGoHandle* out_texture);
CnaGoResult cna_go_texture2d_get_info(
    CnaGoHandle texture,
    uint32_t* width,
    uint32_t* height,
    uint32_t* levels,
    uint32_t* format);
CnaGoResult cna_go_texture2d_destroy(CnaGoHandle texture);

CnaGoResult cna_go_sprite_batch_create(CnaGoHandle device, CnaGoHandle* out_batch);
CnaGoResult cna_go_sprite_batch_begin(CnaGoHandle batch);
CnaGoResult cna_go_sprite_batch_draw_scaled(
    CnaGoHandle batch,
    CnaGoHandle texture,
    float position_x,
    float position_y,
    int32_t source_x,
    int32_t source_y,
    int32_t source_width,
    int32_t source_height,
    uint8_t color_r,
    uint8_t color_g,
    uint8_t color_b,
    uint8_t color_a,
    float rotation,
    float origin_x,
    float origin_y,
    float scale_x,
    float scale_y,
    uint32_t effects,
    float layer_depth);
CnaGoResult cna_go_sprite_batch_draw_destination(CnaGoHandle batch, CnaGoHandle texture, int32_t destination_x, int32_t destination_y, int32_t destination_width, int32_t destination_height, int32_t source_x, int32_t source_y, int32_t source_width, int32_t source_height, uint8_t color_r, uint8_t color_g, uint8_t color_b, uint8_t color_a, float rotation, float origin_x, float origin_y, uint32_t effects, float layer_depth);
CnaGoResult cna_go_sprite_batch_end(CnaGoHandle batch);
CnaGoResult cna_go_sprite_batch_destroy(CnaGoHandle batch);

/* GameWindow. Every route takes the GAME handle because CNA models the window
   as a property of the game; nothing here owns a new native lifetime. */
CnaGoResult cna_go_game_window_get_allow_user_resizing(CnaGoHandle game, uint8_t* out_allowed);
CnaGoResult cna_go_game_window_set_allow_user_resizing(CnaGoHandle game, uint8_t allowed);
CnaGoResult cna_go_game_window_get_client_bounds(
    CnaGoHandle game,
    int32_t* x,
    int32_t* y,
    int32_t* width,
    int32_t* height);
CnaGoResult cna_go_game_window_get_native_handle(CnaGoHandle game, uint64_t* out_handle);

/* Two-call string reads. The size route reports the byte count CNA would write
   and the copy route fills caller-owned storage; Go owns the buffer in both
   cases and no native string pointer is ever retained. */
CnaGoResult cna_go_game_window_get_screen_device_name_size(CnaGoHandle game, uint64_t* out_bytes);
CnaGoResult cna_go_game_window_copy_screen_device_name(
    CnaGoHandle game,
    char* destination,
    uint64_t capacity,
    uint64_t* out_bytes);

CnaGoResult cna_go_game_window_begin_screen_device_change(CnaGoHandle game, uint8_t will_be_full_screen);
CnaGoResult cna_go_game_window_end_screen_device_change(
    CnaGoHandle game,
    const char* screen_device_name,
    uint64_t screen_device_name_length,
    int32_t client_width,
    int32_t client_height);
CnaGoResult cna_go_game_set_window_title(
    CnaGoHandle game,
    const char* title,
    uint64_t title_length);

/* Installs exactly one native subscription per canonical window event, exactly
   as cna_go_game_subscribe_events does for the game's own four. A window
   registration IS a CNA_GameEventRegistrationHandle and is released through
   cna_game_unsubscribe like any other, but the release helper is separate
   because the two tables have different LENGTHS -- three against four -- and
   releasing a three-slot array with the four-slot loop would read past it. */
CnaGoResult cna_go_game_window_subscribe_events(
    CnaGoHandle game,
    uintptr_t context,
    CnaGoHandle* out_registrations);
CnaGoResult cna_go_game_window_unsubscribe_events(CnaGoHandle* registrations);

CnaGoResult cna_go_keyboard_get_state(
    CnaGoHandle game,
    uint64_t* word0,
    uint64_t* word1,
    uint64_t* word2,
    uint64_t* word3);

#ifdef __cplusplus
}
#endif


/* Foundation 87 -- SoundEffect and SoundEffectInstance. The two CNA structures
   are filled on the C side; everything else crosses cgo as scalars. */
CnaGoResult cna_go_sound_effect_create_pcm16_range(
    CnaGoHandle game, uint32_t sample_rate, uint32_t channels, const uint8_t* pcm_bytes,
    uint64_t byte_count, int32_t offset, int32_t count, int32_t loop_start, int32_t loop_length,
    CnaGoHandle* out_sound_effect);
CnaGoResult cna_go_sound_effect_create_from_encoded(
    CnaGoHandle game, const uint8_t* bytes, uint64_t byte_count, CnaGoHandle* out_sound_effect);
CnaGoResult cna_go_sound_effect_get_duration_ticks(CnaGoHandle sound_effect, int64_t* out_ticks);
CnaGoResult cna_go_sound_effect_create_instance(CnaGoHandle sound_effect, CnaGoHandle* out_instance);
CnaGoResult cna_go_sound_effect_destroy(CnaGoHandle sound_effect);
CnaGoResult cna_go_sound_effect_play(CnaGoHandle sound_effect, uint8_t* out_played);
CnaGoResult cna_go_sound_effect_play_with_settings(
    CnaGoHandle sound_effect, float volume, float pitch, float pan, uint8_t* out_played);
CnaGoResult cna_go_sound_effect_set_master_volume(CnaGoHandle game, float value);
CnaGoResult cna_go_sound_effect_set_distance_scale(CnaGoHandle game, float value);
CnaGoResult cna_go_sound_effect_set_doppler_scale(CnaGoHandle game, float value);
CnaGoResult cna_go_sound_effect_set_speed_of_sound(CnaGoHandle game, float value);
CnaGoResult cna_go_sound_effect_instance_play(CnaGoHandle instance);
CnaGoResult cna_go_sound_effect_instance_pause(CnaGoHandle instance);
CnaGoResult cna_go_sound_effect_instance_resume(CnaGoHandle instance);
CnaGoResult cna_go_sound_effect_instance_stop(CnaGoHandle instance, uint8_t immediate);
CnaGoResult cna_go_sound_effect_instance_get_info(
    CnaGoHandle instance, uint32_t* out_state, uint8_t* out_is_looped, float* out_scalars);
CnaGoResult cna_go_sound_effect_instance_set_volume(CnaGoHandle instance, float value);
CnaGoResult cna_go_sound_effect_instance_set_pitch(CnaGoHandle instance, float value);
CnaGoResult cna_go_sound_effect_instance_set_pan(CnaGoHandle instance, float value);
CnaGoResult cna_go_sound_effect_instance_set_is_looped(CnaGoHandle instance, uint8_t is_looped);
CnaGoResult cna_go_sound_effect_instance_destroy(CnaGoHandle instance);
/* Apply3D. The listeners cross as a flat array of 12 floats each -- forward,
   position, up, velocity -- and the emitter as 13, its Doppler scale first. */
/* Foundation 88 -- Microphone. Index-addressed, so every wrapper takes the
   game handle and the position rather than an owned resource. */
CnaGoResult cna_go_microphone_get_count(CnaGoHandle game, uint64_t* out_count);
CnaGoResult cna_go_microphone_get_default_index(CnaGoHandle game, uint64_t* out_index, uint8_t* out_available);
CnaGoResult cna_go_microphone_get_name_size_at(CnaGoHandle game, uint64_t index, uint64_t* out_bytes);
CnaGoResult cna_go_microphone_copy_name_at(
    CnaGoHandle game, uint64_t index, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_microphone_get_buffer_duration_ticks_at(
    CnaGoHandle game, uint64_t index, int64_t* out_ticks);
CnaGoResult cna_go_microphone_set_buffer_duration_ticks_at(
    CnaGoHandle game, uint64_t index, int64_t ticks);
CnaGoResult cna_go_microphone_get_is_headset_at(CnaGoHandle game, uint64_t index, uint8_t* out_value);
CnaGoResult cna_go_microphone_get_sample_rate_at(CnaGoHandle game, uint64_t index, int32_t* out_rate);
CnaGoResult cna_go_microphone_get_state_at(CnaGoHandle game, uint64_t index, uint32_t* out_state);
CnaGoResult cna_go_microphone_start_at(CnaGoHandle game, uint64_t index);
CnaGoResult cna_go_microphone_stop_at(CnaGoHandle game, uint64_t index);
CnaGoResult cna_go_microphone_get_data_at(
    CnaGoHandle game, uint64_t index, uint8_t* destination, uint64_t capacity, uint64_t* out_bytes);

/* Foundation 88 -- DynamicSoundEffectInstance. */
CnaGoResult cna_go_dynamic_sound_effect_instance_create(
    CnaGoHandle game, int32_t sample_rate, uint32_t channels, CnaGoHandle* out_instance);
CnaGoResult cna_go_dynamic_sound_effect_instance_get_pending_buffer_count(
    CnaGoHandle instance, int32_t* out_count);
CnaGoResult cna_go_dynamic_sound_effect_instance_submit_buffer(
    CnaGoHandle instance, const uint8_t* bytes, uint64_t byte_count, int32_t offset, int32_t count);

CnaGoResult cna_go_sound_effect_instance_apply_3d(
    CnaGoHandle instance, const float* listeners, uint64_t listener_count, const float* emitter);

/* The three floats cna_go_sound_effect_instance_get_info reports, by index. */
#define CNA_GO_SOUND_INSTANCE_VOLUME 0
#define CNA_GO_SOUND_INSTANCE_PITCH 1
#define CNA_GO_SOUND_INSTANCE_PAN 2


/* Foundation 89 -- the Input family, GamePad and Mouse only. The capability
   structure is filled on the C side and crosses as a flat byte array plus its
   one non-boolean field, so a thirty-eight-field cgo signature never exists,
   and the state reader flattens its CNA structure the same way -- no CNA type
   crosses cgo.

   The TOUCH routes CNA also offers are deliberately absent: the pinned
   Microsoft.Xna.Framework.Input.Touch.dll contains no p/invoke at all, so the
   XNA Windows runtime never reaches a digitizer and binding them would make
   this projection answer what the reference does not. They are recorded under
   CONTRACT_DIVERGENCE in the native census. */
CnaGoResult cna_go_gamepad_get_state(
    CnaGoHandle game, uint32_t player_index, uint8_t has_dead_zone, uint32_t dead_zone,
    uint8_t* out_connected, int32_t* out_packet, uint32_t* out_buttons, float* out_analog);

CnaGoResult cna_go_gamepad_get_capabilities(
    CnaGoHandle game, uint32_t player_index, uint32_t* out_type, uint8_t* out_flags);
CnaGoResult cna_go_gamepad_set_vibration(
    CnaGoHandle game, uint32_t player_index, float left, float right, uint8_t* out_applied);
CnaGoResult cna_go_mouse_get_state(CnaGoHandle game, int32_t* out_ints, uint32_t* out_buttons);
CnaGoResult cna_go_mouse_set_position(CnaGoHandle game, int32_t x, int32_t y);
CnaGoResult cna_go_mouse_get_window_handle(CnaGoHandle game, uint64_t* out_window);
CnaGoResult cna_go_mouse_set_window_handle(CnaGoHandle game, uint64_t window);

/* The four ints cna_go_mouse_get_state reports, by index. */
#define CNA_GO_MOUSE_X 0
#define CNA_GO_MOUSE_Y 1
#define CNA_GO_MOUSE_SCROLL 2
#define CNA_GO_MOUSE_HORIZONTAL_SCROLL 3

/* The twenty-four capability flags XNA declares, in the order
   GamePadCapabilities' properties are read. */
#define CNA_GO_GAMEPAD_FLAG_COUNT 25


/* Foundation 91 -- the Storage family, forty-three routes.

   Two flattenings the wrappers perform. A CNA_StringView becomes a pointer and
   a length, which is what every string route in this bridge already does. And
   the CNA_StorageCompletionCallback every selector and open route accepts is
   passed as NULL: XNA's own AsyncCallback is invoked from MANAGED code, by
   Begin itself, before it returns -- so a native callback would fire a second
   time for the same completion.

   The three `_ext` routes are not XNA surface. They exist here so the test
   harness can isolate itself in a project-controlled root and then PROVE it,
   which is the same judgement that made MICROPHONE_CAPTURE_CALLS a counter. */
CnaGoResult cna_go_storage_device_show_selector(CnaGoHandle* out_device);
CnaGoResult cna_go_storage_device_show_selector_for_player(uint32_t player, CnaGoHandle* out_device);
CnaGoResult cna_go_storage_device_show_selector_with_space(int32_t size_in_bytes, int32_t directory_count, CnaGoHandle* out_device);
CnaGoResult cna_go_storage_device_show_selector_for_player_with_space(uint32_t player, int32_t size_in_bytes, int32_t directory_count, CnaGoHandle* out_device);
CnaGoResult cna_go_storage_device_get_free_space(CnaGoHandle device, int64_t* out_free_space);
CnaGoResult cna_go_storage_device_get_is_connected(CnaGoHandle device, uint8_t* out_is_connected);
CnaGoResult cna_go_storage_device_get_total_space(CnaGoHandle device, int64_t* out_total_space);
CnaGoResult cna_go_storage_device_delete_container(CnaGoHandle device, const char* title_name, uint64_t title_name_length);
CnaGoResult cna_go_storage_device_destroy(CnaGoHandle device);
CnaGoResult cna_go_storage_container_open(CnaGoHandle device, const char* display_name, uint64_t display_name_length, CnaGoHandle* out_container);
CnaGoResult cna_go_storage_container_get_display_name_size(CnaGoHandle container, uint64_t* out_bytes);
CnaGoResult cna_go_storage_container_copy_display_name(CnaGoHandle container, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_storage_container_get_is_disposed(CnaGoHandle container, uint8_t* out_is_disposed);
CnaGoResult cna_go_storage_container_get_storage_device(CnaGoHandle container, CnaGoHandle* out_device);
CnaGoResult cna_go_storage_container_dispose(CnaGoHandle container);
CnaGoResult cna_go_storage_container_create_directory(CnaGoHandle container, const char* directory, uint64_t directory_length);
CnaGoResult cna_go_storage_container_directory_exists(CnaGoHandle container, const char* directory, uint64_t directory_length, uint8_t* out_exists);
CnaGoResult cna_go_storage_container_delete_directory(CnaGoHandle container, const char* directory, uint64_t directory_length);
CnaGoResult cna_go_storage_container_file_exists(CnaGoHandle container, const char* file, uint64_t file_length, uint8_t* out_exists);
CnaGoResult cna_go_storage_container_delete_file(CnaGoHandle container, const char* file, uint64_t file_length);
CnaGoResult cna_go_storage_container_get_directory_name_count(CnaGoHandle container, const char* search_pattern, uint64_t search_pattern_length, uint64_t* out_count);
CnaGoResult cna_go_storage_container_copy_directory_name(CnaGoHandle container, const char* search_pattern, uint64_t search_pattern_length, uint64_t index, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_storage_container_get_file_name_count(CnaGoHandle container, const char* search_pattern, uint64_t search_pattern_length, uint64_t* out_count);
CnaGoResult cna_go_storage_container_copy_file_name(CnaGoHandle container, const char* search_pattern, uint64_t search_pattern_length, uint64_t index, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_storage_container_create_file(CnaGoHandle container, const char* file, uint64_t file_length, CnaGoHandle* out_stream);
CnaGoResult cna_go_storage_container_open_file(CnaGoHandle container, const char* file, uint64_t file_length, uint32_t file_mode, CnaGoHandle* out_stream);
CnaGoResult cna_go_storage_container_open_file_access(CnaGoHandle container, const char* file, uint64_t file_length, uint32_t file_mode, uint32_t file_access, CnaGoHandle* out_stream);
CnaGoResult cna_go_storage_container_open_file_share(CnaGoHandle container, const char* file, uint64_t file_length, uint32_t file_mode, uint32_t file_access, uint32_t file_share, CnaGoHandle* out_stream);
CnaGoResult cna_go_storage_container_destroy(CnaGoHandle container);
CnaGoResult cna_go_storage_stream_read(CnaGoHandle stream, uint8_t* destination, uint64_t capacity, uint64_t* out_read);
CnaGoResult cna_go_storage_stream_write(CnaGoHandle stream, const uint8_t* data, uint64_t count);
CnaGoResult cna_go_storage_stream_seek(CnaGoHandle stream, int64_t offset, uint32_t origin, int64_t* out_position);
CnaGoResult cna_go_storage_stream_get_position(CnaGoHandle stream, int64_t* out_position);
CnaGoResult cna_go_storage_stream_get_length(CnaGoHandle stream, int64_t* out_length);
CnaGoResult cna_go_storage_stream_set_length(CnaGoHandle stream, int64_t length);
CnaGoResult cna_go_storage_stream_get_can_read(CnaGoHandle stream, uint8_t* out_can_read);
CnaGoResult cna_go_storage_stream_get_can_write(CnaGoHandle stream, uint8_t* out_can_write);
CnaGoResult cna_go_storage_stream_get_can_seek(CnaGoHandle stream, uint8_t* out_can_seek);
CnaGoResult cna_go_storage_stream_flush(CnaGoHandle stream);
CnaGoResult cna_go_storage_stream_close(CnaGoHandle stream);
CnaGoResult cna_go_storage_set_app_name_ext(const char* app_name, uint64_t app_name_length);
CnaGoResult cna_go_storage_get_root_size_ext(uint64_t* out_bytes);
CnaGoResult cna_go_storage_copy_root_ext(char* destination, uint64_t capacity, uint64_t* out_bytes);
#endif

CnaGoResult cna_go_vertex_declaration_create(
    int32_t vertex_stride,
    uint8_t has_stride,
    const int32_t* elements,
    uint64_t element_count,
    CnaGoHandle* out_declaration);
CnaGoResult cna_go_vertex_declaration_destroy(CnaGoHandle declaration);
CnaGoResult cna_go_vertex_declaration_get_stride(CnaGoHandle declaration, int32_t* out_stride);
CnaGoResult cna_go_vertex_buffer_create(
    CnaGoHandle device,
    CnaGoHandle declaration,
    int32_t vertex_count,
    uint32_t buffer_usage,
    uint8_t dynamic,
    CnaGoHandle* out_vertex_buffer);
CnaGoResult cna_go_vertex_buffer_destroy(CnaGoHandle vertex_buffer);
CnaGoResult cna_go_vertex_buffer_get_info(
    CnaGoHandle vertex_buffer,
    int32_t* out_vertex_count,
    uint32_t* out_buffer_usage,
    uint8_t* out_dynamic,
    uint8_t* out_is_content_lost,
    uint8_t* out_has_renderer,
    int32_t* out_vertex_stride,
    uint64_t* out_vertex_element_count);
CnaGoResult cna_go_vertex_buffer_set_data_raw_at(
    CnaGoHandle vertex_buffer,
    uint64_t buffer_offset_in_bytes,
    const void* data,
    uint64_t data_byte_count,
    uint64_t vertex_count,
    uint32_t vertex_stride);
CnaGoResult cna_go_vertex_buffer_get_data_raw(
    CnaGoHandle vertex_buffer,
    uint64_t buffer_offset_in_bytes,
    void* destination,
    uint64_t destination_byte_count,
    uint64_t vertex_count,
    uint32_t vertex_stride);

CnaGoResult cna_go_graphics_device_set_vertex_buffers(
    CnaGoHandle device,
    const int64_t* bindings,
    uint64_t binding_count);
CnaGoResult cna_go_graphics_device_set_index_buffer(CnaGoHandle device, CnaGoHandle index_buffer);
CnaGoResult cna_go_graphics_device_draw_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t vertex_start, int32_t primitive_count);
CnaGoResult cna_go_graphics_device_draw_indexed_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t base_vertex, int32_t min_vertex_index,
    int32_t num_vertices, int32_t start_index, int32_t primitive_count);
CnaGoResult cna_go_graphics_device_draw_instanced_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t base_vertex, int32_t min_vertex_index,
    int32_t num_vertices, int32_t start_index, int32_t primitive_count, int32_t instance_count);

CnaGoResult cna_go_graphics_adapter_get_count(CnaGoHandle device, uint64_t* out_count);
CnaGoResult cna_go_graphics_adapter_get_info(
    CnaGoHandle device,
    uint32_t adapter_index,
    uint32_t* out_index,
    uint8_t* out_is_default,
    uint8_t* out_is_wide_screen,
    uint8_t* out_use_null_device,
    uint8_t* out_use_reference_device,
    int32_t* out_vendor_id,
    int32_t* out_device_id,
    int32_t* out_revision,
    int32_t* out_subsystem_id,
    uint64_t* out_description_bytes,
    uint64_t* out_device_name_bytes);
CnaGoResult cna_go_graphics_adapter_copy_description(
    CnaGoHandle device, uint32_t adapter_index, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_graphics_adapter_copy_device_name(
    CnaGoHandle device, uint32_t adapter_index, char* destination, uint64_t capacity, uint64_t* out_bytes);
CnaGoResult cna_go_graphics_adapter_get_current_display_mode(
    CnaGoHandle device, uint32_t adapter_index, int32_t* out_width, int32_t* out_height, uint32_t* out_format);
CnaGoResult cna_go_graphics_adapter_get_display_mode_count(
    CnaGoHandle device, uint32_t adapter_index, uint64_t* out_count);
CnaGoResult cna_go_graphics_adapter_copy_display_modes(
    CnaGoHandle device, uint32_t adapter_index, int32_t* out_modes, uint64_t capacity, uint64_t* out_count);
CnaGoResult cna_go_graphics_adapter_set_device_preferences(
    CnaGoHandle device, uint32_t adapter_index, uint8_t use_null_device, uint8_t use_reference_device);
CnaGoResult cna_go_graphics_adapter_is_profile_supported(
    CnaGoHandle device, uint32_t adapter_index, uint32_t profile, uint8_t* out_supported);
CnaGoResult cna_go_graphics_adapter_query_format(
    CnaGoHandle device,
    uint32_t adapter_index,
    uint8_t render_target,
    uint32_t profile,
    uint32_t format,
    uint32_t depth_format,
    int32_t multi_sample_count,
    uint8_t* out_exact_match,
    uint32_t* out_format,
    uint32_t* out_depth_format,
    int32_t* out_multi_sample_count);
CnaGoResult cna_go_graphics_adapter_get_native_monitor_handle(
    CnaGoHandle device, uint32_t adapter_index, uint64_t* out_value);

CnaGoResult cna_go_graphics_device_get_adapter_index(CnaGoHandle device, uint32_t* out_index);
