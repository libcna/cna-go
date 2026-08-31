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
    CNA_GO_RESULT_CALLBACK = 9,
    CNA_GO_ABI_VERSION = 0x00000700u,
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
    CNA_GO_GAME_EVENT_COUNT = 4
};

int cna_go_open(const char* path, char* error_buffer, size_t error_capacity);
void cna_go_close(void);
uint32_t cna_go_abi_version(void);
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
CnaGoResult cna_go_game_get_graphics_device(CnaGoHandle game, CnaGoHandle* out_device);
CnaGoResult cna_go_graphics_device_get_viewport(
    CnaGoHandle device,
    int32_t* x,
    int32_t* y,
    int32_t* width,
    int32_t* height,
    float* min_depth,
    float* max_depth);
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
CnaGoResult cna_go_sprite_batch_end(CnaGoHandle batch);
CnaGoResult cna_go_sprite_batch_destroy(CnaGoHandle batch);

CnaGoResult cna_go_keyboard_get_state(
    CnaGoHandle game,
    uint64_t* word0,
    uint64_t* word1,
    uint64_t* word2,
    uint64_t* word3);

#ifdef __cplusplus
}
#endif

#endif
