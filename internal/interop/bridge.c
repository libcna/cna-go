// SPDX-License-Identifier: MS-PL

#include "bridge.h"
#include "abi_manifest.h"

#include <dlfcn.h>
#include <pthread.h>
#include <stdio.h>
#include <string.h>

typedef struct CnaGoApi {
#define CNA_GO_DECLARE(name) name##_fn name;
    CNA_GO_REQUIRED_SYMBOLS(CNA_GO_DECLARE)
#undef CNA_GO_DECLARE
} CnaGoApi;

static CnaGoApi api;
static void* library_handle;

static const char* const required_symbol_names[] = {
#define CNA_GO_NAME(name) #name,
    CNA_GO_REQUIRED_SYMBOLS(CNA_GO_NAME)
#undef CNA_GO_NAME
};

extern uint32_t cnaGoLifecycle(
    uint32_t kind,
    uint64_t game,
    int64_t total_ticks,
    int64_t elapsed_ticks,
    uint8_t running_slowly,
    uintptr_t context);

static CNA_Result forward_lifecycle(
    uint32_t kind,
    CNA_Handle game,
    const CNA_GameTime* game_time,
    void* context) {
    const int64_t total = game_time == NULL ? 0 : game_time->total_game_time_ticks;
    const int64_t elapsed = game_time == NULL ? 0 : game_time->elapsed_game_time_ticks;
    const uint8_t slow = game_time == NULL ? 0 : game_time->is_running_slowly;
    return cnaGoLifecycle(kind, game, total, elapsed, slow, (uintptr_t)context);
}

#define CNA_GO_CALLBACK(name, kind) \
    static CNA_Result name( \
        CNA_Handle game, \
        const CNA_GameTime* game_time, \
        void* context, \
        CNA_CallbackError* out_error) { \
        (void)out_error; \
        return forward_lifecycle(kind, game, game_time, context); \
    }

CNA_GO_CALLBACK(callback_initialize, CNA_GO_CALLBACK_INITIALIZE)
CNA_GO_CALLBACK(callback_load_content, CNA_GO_CALLBACK_LOAD_CONTENT)
CNA_GO_CALLBACK(callback_update, CNA_GO_CALLBACK_UPDATE)
CNA_GO_CALLBACK(callback_draw, CNA_GO_CALLBACK_DRAW)
CNA_GO_CALLBACK(callback_unload_content, CNA_GO_CALLBACK_UNLOAD_CONTENT)
CNA_GO_CALLBACK(callback_exiting, CNA_GO_CALLBACK_EXITING)

static void copy_error(char* destination, size_t capacity, const char* message) {
    if (destination == NULL || capacity == 0) {
        return;
    }
    snprintf(destination, capacity, "%s", message == NULL ? "unknown dynamic-loader error" : message);
}

int cna_go_open(const char* path, char* error_buffer, size_t error_capacity) {
    if (library_handle != NULL) {
        copy_error(error_buffer, error_capacity, "CNA library is already open");
        return 0;
    }
    dlerror();
    library_handle = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (library_handle == NULL) {
        copy_error(error_buffer, error_capacity, dlerror());
        return 0;
    }
#define CNA_GO_RESOLVE(name) do { \
        void* symbol = dlsym(library_handle, #name); \
        const char* resolution_error = dlerror(); \
        if (resolution_error != NULL || symbol == NULL) { \
            char message[512]; \
            snprintf(message, sizeof(message), "missing required symbol %s: %s", #name, resolution_error == NULL ? "not exported" : resolution_error); \
            copy_error(error_buffer, error_capacity, message); \
            dlclose(library_handle); \
            library_handle = NULL; \
            memset(&api, 0, sizeof(api)); \
            return 0; \
        } \
        memcpy(&api.name, &symbol, sizeof(symbol)); \
    } while (0);
    CNA_GO_REQUIRED_SYMBOLS(CNA_GO_RESOLVE)
#undef CNA_GO_RESOLVE
    return 1;
}

void cna_go_close(void) {
    if (library_handle != NULL) {
        dlclose(library_handle);
    }
    library_handle = NULL;
    memset(&api, 0, sizeof(api));
}

uint32_t cna_go_abi_version(void) {
    return api.cna_get_abi_version == NULL ? 0 : api.cna_get_abi_version();
}

uint64_t cna_go_owner_thread_id(void) {
    pthread_t thread = pthread_self();
    uint64_t result = 0;
    const size_t copy = sizeof(thread) < sizeof(result) ? sizeof(thread) : sizeof(result);
    memcpy(&result, &thread, copy);
    return result;
}

uint32_t cna_go_bound_function_count(void) {
    return (uint32_t)(sizeof(required_symbol_names) / sizeof(required_symbol_names[0]));
}

const char* cna_go_bound_function_name(uint32_t index) {
    if (index >= cna_go_bound_function_count()) {
        return NULL;
    }
    return required_symbol_names[index];
}

int cna_go_has_loaded_symbol(const char* name) {
    return library_handle != NULL && name != NULL && dlsym(library_handle, name) != NULL;
}

size_t cna_go_last_error_message(char* destination, size_t capacity) {
    if (api.cna_error_get_last_message_size == NULL || api.cna_error_copy_last_message == NULL) {
        return 0;
    }
    uint64_t required = 0;
    if (api.cna_error_get_last_message_size(&required) != 0) {
        return 0;
    }
    if (destination != NULL && capacity > 0) {
        uint64_t written = 0;
        (void)api.cna_error_copy_last_message(destination, (uint64_t)capacity, &written);
    }
    return (size_t)required;
}

CnaGoResult cna_go_game_create(uintptr_t context, const char* title, uint64_t title_length, CnaGoHandle* out_game) {
    CNA_GameCallbacks callbacks;
    memset(&callbacks, 0, sizeof(callbacks));
    callbacks.struct_size = (uint32_t)sizeof(callbacks);
    callbacks.struct_version = 1;
    callbacks.load_content = callback_load_content;
    callbacks.update = callback_update;
    callbacks.draw = callback_draw;
    callbacks.unload_content = callback_unload_content;
    callbacks.exiting = callback_exiting;
    callbacks.context = (void*)context;

    CNA_GameCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.is_fixed_time_step = 1;
    info.target_elapsed_time_ticks = 166667;
    info.window_title.data = title;
    info.window_title.byte_length = title_length;
    info.callbacks = &callbacks;
    CNA_Result result = api.cna_game_create(&info, out_game);
    if (result != 0) {
        return result;
    }

    CNA_GameFrameHooks hooks;
    memset(&hooks, 0, sizeof(hooks));
    hooks.struct_size = (uint32_t)sizeof(hooks);
    hooks.struct_version = 1;
    hooks.initialize = callback_initialize;
    hooks.context = (void*)context;
    result = api.cna_game_set_frame_hooks_ext(*out_game, &hooks);
    if (result != 0) {
        (void)api.cna_game_destroy(*out_game);
        *out_game = 0;
    }
    return result;
}

CnaGoResult cna_go_game_run(CnaGoHandle game) { return api.cna_game_run(game); }
CnaGoResult cna_go_game_request_exit(CnaGoHandle game) { return api.cna_game_request_exit(game); }
CnaGoResult cna_go_game_destroy(CnaGoHandle game) { return api.cna_game_destroy(game); }

CnaGoResult cna_go_graphics_device_manager_create(CnaGoHandle game, CnaGoHandle* out_manager) {
    return api.cna_graphics_device_manager_create(game, out_manager);
}
CnaGoResult cna_go_graphics_device_manager_get_device(CnaGoHandle manager, CnaGoHandle* out_device) {
    return api.cna_graphics_device_manager_get_graphics_device(manager, out_device);
}
CnaGoResult cna_go_graphics_device_manager_destroy(CnaGoHandle manager) {
    return api.cna_graphics_device_manager_destroy(manager);
}
CnaGoResult cna_go_game_get_graphics_device(CnaGoHandle game, CnaGoHandle* out_device) {
    return api.cna_game_get_graphics_device(game, out_device);
}
CnaGoResult cna_go_graphics_device_get_viewport(CnaGoHandle device, int32_t* x, int32_t* y, int32_t* width, int32_t* height, float* min_depth, float* max_depth) {
    CNA_Viewport viewport;
    const CNA_Result result = api.cna_graphics_device_get_viewport(device, &viewport);
    if (result == 0) {
        *x = viewport.x;
        *y = viewport.y;
        *width = viewport.width;
        *height = viewport.height;
        *min_depth = viewport.min_depth;
        *max_depth = viewport.max_depth;
    }
    return result;
}
CnaGoResult cna_go_graphics_device_clear_rgba(CnaGoHandle device, float r, float g, float b, float a) {
    return api.cna_graphics_device_clear_rgba(device, r, g, b, a);
}

CnaGoResult cna_go_texture2d_create_from_encoded_memory(CnaGoHandle device, const uint8_t* data, uint64_t byte_count, CnaGoHandle* out_texture) {
    return api.cna_texture2d_create_from_encoded_memory(device, data, byte_count, NULL, out_texture);
}
CnaGoResult cna_go_texture2d_get_info(CnaGoHandle texture, uint32_t* width, uint32_t* height, uint32_t* levels, uint32_t* format) {
    CNA_Texture2DInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    const CNA_Result result = api.cna_texture2d_get_info(texture, &info);
    if (result == 0) {
        *width = info.width;
        *height = info.height;
        *levels = info.level_count;
        *format = info.format;
    }
    return result;
}
CnaGoResult cna_go_texture2d_destroy(CnaGoHandle texture) { return api.cna_texture2d_destroy(texture); }

CnaGoResult cna_go_sprite_batch_create(CnaGoHandle device, CnaGoHandle* out_batch) {
    return api.cna_sprite_batch_create(device, out_batch);
}
CnaGoResult cna_go_sprite_batch_begin(CnaGoHandle batch) {
    CNA_SpriteBatchBeginInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.sort_mode = 0;
    return api.cna_sprite_batch_begin(batch, &info);
}
CnaGoResult cna_go_sprite_batch_draw_scaled(CnaGoHandle batch, CnaGoHandle texture, float position_x, float position_y, int32_t source_x, int32_t source_y, int32_t source_width, int32_t source_height, uint8_t color_r, uint8_t color_g, uint8_t color_b, uint8_t color_a, float rotation, float origin_x, float origin_y, float scale_x, float scale_y, uint32_t effects, float layer_depth) {
    CNA_SpriteScaledCommand command;
    memset(&command, 0, sizeof(command));
    command.struct_size = (uint32_t)sizeof(command);
    command.struct_version = 1;
    command.texture = texture;
    command.position.x = position_x;
    command.position.y = position_y;
    command.source.x = source_x;
    command.source.y = source_y;
    command.source.width = source_width;
    command.source.height = source_height;
    command.color.r = color_r;
    command.color.g = color_g;
    command.color.b = color_b;
    command.color.a = color_a;
    command.rotation = rotation;
    command.origin.x = origin_x;
    command.origin.y = origin_y;
    command.scale.x = scale_x;
    command.scale.y = scale_y;
    command.effects = effects;
    command.layer_depth = layer_depth;
    return api.cna_sprite_batch_submit_scaled_many(batch, &command, 1);
}
CnaGoResult cna_go_sprite_batch_end(CnaGoHandle batch) { return api.cna_sprite_batch_end(batch); }
CnaGoResult cna_go_sprite_batch_destroy(CnaGoHandle batch) { return api.cna_sprite_batch_destroy(batch); }

CnaGoResult cna_go_keyboard_get_state(CnaGoHandle game, uint64_t* word0, uint64_t* word1, uint64_t* word2, uint64_t* word3) {
    CNA_KeyboardState state;
    memset(&state, 0, sizeof(state));
    state.struct_size = (uint32_t)sizeof(state);
    state.struct_version = 1;
    const CNA_Result result = api.cna_keyboard_get_state(game, &state);
    if (result == 0) {
        *word0 = state.pressed_key_words[0];
        *word1 = state.pressed_key_words[1];
        *word2 = state.pressed_key_words[2];
        *word3 = state.pressed_key_words[3];
    }
    return result;
}
