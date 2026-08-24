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
    CNA_GO_CALLBACK_EXITING = 6
};

int cna_go_open(const char* path, char* error_buffer, size_t error_capacity);
void cna_go_close(void);
uint32_t cna_go_abi_version(void);
uint64_t cna_go_owner_thread_id(void);
uint32_t cna_go_bound_function_count(void);
const char* cna_go_bound_function_name(uint32_t index);
int cna_go_has_loaded_symbol(const char* name);
size_t cna_go_last_error_message(char* destination, size_t capacity);

CnaGoResult cna_go_game_create(
    uintptr_t context,
    const char* title,
    uint64_t title_length,
    CnaGoHandle* out_game);
CnaGoResult cna_go_game_run(CnaGoHandle game);
CnaGoResult cna_go_game_request_exit(CnaGoHandle game);
CnaGoResult cna_go_game_destroy(CnaGoHandle game);

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
