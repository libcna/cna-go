// SPDX-License-Identifier: MS-PL

/* dladdr is a GNU extension of <dlfcn.h>. cna_go_verify_symbol_identity uses it
   to prove that every resolved pointer belongs to the symbol the manifest
   names, so the declaration has to be visible. */
#ifndef _GNU_SOURCE
#define _GNU_SOURCE 1
#endif

#include "bridge.h"
#include "abi_manifest.h"

#include <dlfcn.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
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

extern void cnaGoGameEvent(uint32_t event, uintptr_t context);

/* One trampoline per canonical event identity. CNA_GameEventCallback carries
   nothing but the caller context, so the identity has to come from the
   function that was registered rather than from a parameter. */
#define CNA_GO_EVENT_CALLBACK(name, event) \
    static void name(void* context) { \
        cnaGoGameEvent((uint32_t)(event), (uintptr_t)context); \
    }

CNA_GO_EVENT_CALLBACK(event_activated, CNA_GO_GAME_EVENT_ACTIVATED)
CNA_GO_EVENT_CALLBACK(event_deactivated, CNA_GO_GAME_EVENT_DEACTIVATED)
CNA_GO_EVENT_CALLBACK(event_disposed, CNA_GO_GAME_EVENT_DISPOSED)
CNA_GO_EVENT_CALLBACK(event_exiting, CNA_GO_GAME_EVENT_EXITING)

static const CNA_GameEventCallback event_callbacks[CNA_GO_GAME_EVENT_COUNT] = {
    event_activated,
    event_deactivated,
    event_disposed,
    event_exiting
};

static const CNA_GameEvent event_identities[CNA_GO_GAME_EVENT_COUNT] = {
    CNA_GAME_EVENT_ACTIVATED,
    CNA_GAME_EVENT_DEACTIVATED,
    CNA_GAME_EVENT_DISPOSED,
    CNA_GAME_EVENT_EXITING
};

extern void cnaGoGameWindowEvent(uint32_t event, uintptr_t context);

/* One trampoline per canonical WINDOW event identity, for the same reason the
   game's four have one each: CNA_GameEventCallback carries only the caller
   context, so the identity has to come from the function that was registered.
   The two families deliberately use different Go entry points -- a window
   signal must never be able to arrive as a game signal, and both numberings
   start at zero. */
#define CNA_GO_WINDOW_EVENT_CALLBACK(name, event) \
    static void name(void* context) { \
        cnaGoGameWindowEvent((uint32_t)(event), (uintptr_t)context); \
    }

CNA_GO_WINDOW_EVENT_CALLBACK(window_event_client_size_changed, CNA_GO_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED)
CNA_GO_WINDOW_EVENT_CALLBACK(window_event_orientation_changed, CNA_GO_GAME_WINDOW_EVENT_ORIENTATION_CHANGED)
CNA_GO_WINDOW_EVENT_CALLBACK(window_event_screen_device_name_changed, CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED)

static const CNA_GameEventCallback window_event_callbacks[CNA_GO_GAME_WINDOW_EVENT_COUNT] = {
    window_event_client_size_changed,
    window_event_orientation_changed,
    window_event_screen_device_name_changed
};

static const CNA_GameWindowEvent window_event_identities[CNA_GO_GAME_WINDOW_EVENT_COUNT] = {
    CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED,
    CNA_GAME_WINDOW_EVENT_ORIENTATION_CHANGED,
    CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED
};

/* bridge.h's window mirror against the manifest's, exactly as the game events
   are compared. tools/native_abi separately compares the manifest's copy with
   the canonical header's. */
_Static_assert(CNA_GO_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED == CNA_GAME_WINDOW_EVENT_CLIENT_SIZE_CHANGED, "client-size identity drift");
_Static_assert(CNA_GO_GAME_WINDOW_EVENT_ORIENTATION_CHANGED == CNA_GAME_WINDOW_EVENT_ORIENTATION_CHANGED, "orientation identity drift");
_Static_assert(CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED == CNA_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED, "screen-device-name identity drift");
_Static_assert(CNA_GO_GAME_WINDOW_EVENT_COUNT == CNA_GAME_WINDOW_EVENT_MAXIMUM + 1, "window-event identity count drift");
_Static_assert(sizeof(CNA_GameWindowEvent) == sizeof(CNA_GameEvent), "window-event identity width drift");

/* bridge.h's mirror is what the Go side switches on, and the two arrays above
   are indexed by it. If the mirror and the manifest ever disagreed, a signal
   would be routed to the wrong projected event with nothing to catch it. */
_Static_assert(CNA_GO_GAME_EVENT_ACTIVATED == CNA_GAME_EVENT_ACTIVATED, "activation identity drift");
_Static_assert(CNA_GO_GAME_EVENT_DEACTIVATED == CNA_GAME_EVENT_DEACTIVATED, "deactivation identity drift");
_Static_assert(CNA_GO_GAME_EVENT_DISPOSED == CNA_GAME_EVENT_DISPOSED, "disposal identity drift");
_Static_assert(CNA_GO_GAME_EVENT_EXITING == CNA_GAME_EVENT_EXITING, "exit identity drift");
_Static_assert(CNA_GO_GAME_EVENT_COUNT == CNA_GAME_EVENT_MAXIMUM + 1, "game-event identity count drift");
_Static_assert(sizeof(CNA_GameEventRegistrationHandle) == sizeof(CNA_Handle), "registration handle width drift");

/* The optional frame-hook mask. Each bit selects exactly one CNA_GameFrameHooks
   member, so two bits that collided would install one hook and silently drop
   the other -- a consumer's declared override would simply never be called.
   native_linux.go separately compares these against the Go constants. */
_Static_assert(CNA_GO_FRAME_HOOK_BEGIN_RUN != CNA_GO_FRAME_HOOK_END_RUN &&
                   CNA_GO_FRAME_HOOK_BEGIN_RUN != CNA_GO_FRAME_HOOK_BEGIN_DRAW &&
                   CNA_GO_FRAME_HOOK_BEGIN_RUN != CNA_GO_FRAME_HOOK_END_DRAW &&
                   CNA_GO_FRAME_HOOK_END_RUN != CNA_GO_FRAME_HOOK_BEGIN_DRAW &&
                   CNA_GO_FRAME_HOOK_END_RUN != CNA_GO_FRAME_HOOK_END_DRAW &&
                   CNA_GO_FRAME_HOOK_BEGIN_DRAW != CNA_GO_FRAME_HOOK_END_DRAW,
               "frame-hook mask bits must select distinct hooks");
_Static_assert((CNA_GO_FRAME_HOOK_BEGIN_RUN | CNA_GO_FRAME_HOOK_END_RUN |
                CNA_GO_FRAME_HOOK_BEGIN_DRAW | CNA_GO_FRAME_HOOK_END_DRAW) == CNA_GO_FRAME_HOOK_ALL,
               "frame-hook mask must cover exactly the four optional hooks");

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

CNA_GO_CALLBACK(callback_initialize, CNA_GO_CALLBACK_INITIALIZE)
CNA_GO_CALLBACK(callback_load_content, CNA_GO_CALLBACK_LOAD_CONTENT)
CNA_GO_CALLBACK(callback_update, CNA_GO_CALLBACK_UPDATE)
CNA_GO_CALLBACK(callback_draw, CNA_GO_CALLBACK_DRAW)
CNA_GO_CALLBACK(callback_unload_content, CNA_GO_CALLBACK_UNLOAD_CONTENT)
CNA_GO_CALLBACK(callback_exiting, CNA_GO_CALLBACK_EXITING)
CNA_GO_CALLBACK(callback_begin_run, CNA_GO_CALLBACK_BEGIN_RUN)
CNA_GO_CALLBACK(callback_end_run, CNA_GO_CALLBACK_END_RUN)
CNA_GO_CALLBACK(callback_end_draw, CNA_GO_CALLBACK_END_DRAW)

extern uint32_t cnaGoBeginDraw(
    uint64_t game,
    int64_t total_ticks,
    int64_t elapsed_ticks,
    uint8_t running_slowly,
    uintptr_t context,
    uint8_t* out_should_draw);

/* begin_draw is the one frame hook with a value channel of its own, so it
   cannot share the lifecycle trampoline. The canonical header initializes
   out_should_draw to CNA_TRUE and documents that a null handler draws, so this
   mirrors both: the local default is 1, and the caller's slot is written only
   when the Go side answered successfully. A failing override therefore leaves
   the runtime's own decision exactly as it found it and reports the failure
   through the established callback-result channel instead. */
static CNA_Result callback_begin_draw(
    CNA_Handle game,
    const CNA_GameTime* game_time,
    void* context,
    CNA_Bool* out_should_draw,
    CNA_CallbackError* out_error) {
    (void)out_error;
    const int64_t total = game_time == NULL ? 0 : game_time->total_game_time_ticks;
    const int64_t elapsed = game_time == NULL ? 0 : game_time->elapsed_game_time_ticks;
    const uint8_t slow = game_time == NULL ? 0 : game_time->is_running_slowly;
    uint8_t should_draw = 1;
    const uint32_t result = cnaGoBeginDraw(game, total, elapsed, slow, (uintptr_t)context, &should_draw);
    if (result == CNA_GO_RESULT_SUCCESS && out_should_draw != NULL) {
        *out_should_draw = (CNA_Bool)(should_draw != 0);
    }
    return result;
}

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

uint32_t cna_go_abi_encode(uint32_t major, uint32_t minor, uint32_t patch) {
    return CNA_GO_ABI_ENCODE(major, minor, patch);
}

uint32_t cna_go_abi_major_of(uint32_t version) { return CNA_GO_ABI_MAJOR_OF(version); }
uint32_t cna_go_abi_minor_of(uint32_t version) { return CNA_GO_ABI_MINOR_OF(version); }
uint32_t cna_go_abi_patch_of(uint32_t version) { return CNA_GO_ABI_PATCH_OF(version); }

/* The qualified encoded constant must be exactly what the policy's own parts
   encode. A floor raised without re-encoding the qualified constant, or the
   reverse, would make the loader report one range and enforce another. */
_Static_assert(CNA_GO_ABI_QUALIFIED_VERSION ==
                   CNA_GO_ABI_ENCODE(CNA_GO_ABI_MAJOR, CNA_GO_ABI_MINIMUM_MINOR, CNA_GO_ABI_QUALIFIED_PATCH),
               "the qualified ABI constant must encode the admission policy's own parts");
_Static_assert(CNA_GO_ABI_MAJOR_OF(CNA_GO_ABI_QUALIFIED_VERSION) == CNA_GO_ABI_MAJOR,
               "encoded major must decode back to the policy major");
_Static_assert(CNA_GO_ABI_MINOR_OF(CNA_GO_ABI_QUALIFIED_VERSION) == CNA_GO_ABI_MINIMUM_MINOR,
               "encoded minor must decode back to the policy floor");

int cna_go_abi_admits(uint32_t version) {
    if (CNA_GO_ABI_MAJOR_OF(version) != (uint32_t)CNA_GO_ABI_MAJOR) {
        return 0;
    }
    return CNA_GO_ABI_MINOR_OF(version) >= (uint32_t)CNA_GO_ABI_MINIMUM_MINOR;
}

/* The resolution macro pairs each api field with the string form of the SAME
   macro argument, so a field cannot be filled from a differently named symbol
   by editing one side alone. That is a textual argument, not a measurement, and
   several bound routes share a prototype -- cna_game_run, cna_game_request_exit
   and cna_game_destroy are all CNA_Result(CNA_Handle) -- so a mis-pairing among
   them would compile cleanly. dladdr turns the argument into evidence: it
   reports the symbol that actually owns each resolved address. */
int cna_go_verify_symbol_identity(char* out_detail, size_t detail_capacity) {
    if (library_handle == NULL) {
        copy_error(out_detail, detail_capacity, "no CNA library is open");
        return 0;
    }
    void* const resolved[] = {
#define CNA_GO_ADDRESS(name) (void*)(uintptr_t)api.name,
        CNA_GO_REQUIRED_SYMBOLS(CNA_GO_ADDRESS)
#undef CNA_GO_ADDRESS
    };
    const uint32_t count = cna_go_bound_function_count();
    for (uint32_t index = 0; index < count; ++index) {
        Dl_info info;
        memset(&info, 0, sizeof(info));
        if (resolved[index] == NULL) {
            char message[512];
            snprintf(message, sizeof(message), "%s resolved to a null address", required_symbol_names[index]);
            copy_error(out_detail, detail_capacity, message);
            return 0;
        }
        if (dladdr(resolved[index], &info) == 0 || info.dli_sname == NULL) {
            char message[512];
            snprintf(message, sizeof(message), "%s has no dynamic-symbol identity", required_symbol_names[index]);
            copy_error(out_detail, detail_capacity, message);
            return 0;
        }
        if (strcmp(info.dli_sname, required_symbol_names[index]) != 0) {
            char message[512];
            snprintf(message, sizeof(message), "%s is bound to %s", required_symbol_names[index], info.dli_sname);
            copy_error(out_detail, detail_capacity, message);
            return 0;
        }
    }
    copy_error(out_detail, detail_capacity, "");
    return 1;
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

CnaGoResult cna_go_game_set_is_mouse_visible(CnaGoHandle game, uint8_t visible) {
    return api.cna_game_set_is_mouse_visible(game, (CNA_Bool)(visible != 0));
}
CnaGoResult cna_go_game_set_is_fixed_time_step(CnaGoHandle game, uint8_t fixed) {
    return api.cna_game_set_is_fixed_time_step(game, (CNA_Bool)(fixed != 0));
}
CnaGoResult cna_go_game_set_target_elapsed_time_ticks(CnaGoHandle game, int64_t ticks) {
    return api.cna_game_set_target_elapsed_time_ticks(game, ticks);
}
CnaGoResult cna_go_game_set_inactive_sleep_time_ticks(CnaGoHandle game, int64_t ticks) {
    return api.cna_game_set_inactive_sleep_time_ticks(game, ticks);
}
CnaGoResult cna_go_game_reset_elapsed_time(CnaGoHandle game) {
    return api.cna_game_reset_elapsed_time(game);
}
CnaGoResult cna_go_game_suppress_draw(CnaGoHandle game) {
    return api.cna_game_suppress_draw(game);
}

CnaGoResult cna_go_game_create(uintptr_t context, const char* title, uint64_t title_length, uint32_t frame_hook_overrides, const CnaGoGameTiming* timing, CnaGoHandle* out_game) {
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
    /* The Game's own configured values, not literals. XNA's Game constructor
       sets isFixedTimeStep = true and targetElapsedTime = 166667 ticks, and a
       consumer may change either before Run; the reference's loop would honour
       that, so the native loop is handed what the managed state actually says. */
    info.is_fixed_time_step = (CNA_Bool)(timing != NULL && timing->is_fixed_time_step != 0);
    info.target_elapsed_time_ticks = timing == NULL ? 166667 : timing->target_elapsed_time_ticks;
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
    /* Each optional member is assigned if and only if the Go side reported a
       matching override. An unset bit leaves the member NULL, which the
       canonical header defines as "simply not called", so the native default
       behaviour at that frame position is untouched. */
    if ((frame_hook_overrides & CNA_GO_FRAME_HOOK_BEGIN_RUN) != 0) {
        hooks.begin_run = callback_begin_run;
    }
    if ((frame_hook_overrides & CNA_GO_FRAME_HOOK_END_RUN) != 0) {
        hooks.end_run = callback_end_run;
    }
    if ((frame_hook_overrides & CNA_GO_FRAME_HOOK_BEGIN_DRAW) != 0) {
        hooks.begin_draw = callback_begin_draw;
    }
    if ((frame_hook_overrides & CNA_GO_FRAME_HOOK_END_DRAW) != 0) {
        hooks.end_draw = callback_end_draw;
    }
    hooks.context = (void*)context;
    result = api.cna_game_set_frame_hooks_ext(*out_game, &hooks);
    if (result != 0) {
        (void)api.cna_game_destroy(*out_game);
        *out_game = 0;
        return result;
    }

    /* The two settings CNA_GameCreateInfo has no field for, pushed on the
       owner thread before the loop can run a frame. A failure here destroys
       the game rather than leaving one whose configured state was silently
       not applied. */
    if (timing != NULL) {
        result = api.cna_game_set_inactive_sleep_time_ticks(*out_game, timing->inactive_sleep_time_ticks);
        if (result == 0) {
            result = api.cna_game_set_is_mouse_visible(*out_game, (CNA_Bool)(timing->is_mouse_visible != 0));
        }
        if (result != 0) {
            (void)api.cna_game_destroy(*out_game);
            *out_game = 0;
        }
    }
    return result;
}

CnaGoResult cna_go_game_unsubscribe_events(CnaGoHandle* registrations) {
    if (registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    CNA_Result first = 0;
    for (int i = 0; i < CNA_GO_GAME_EVENT_COUNT; i++) {
        if (registrations[i] == 0) {
            continue;
        }
        const CNA_Result result = api.cna_game_unsubscribe(registrations[i]);
        registrations[i] = 0;
        if (result != 0 && first == 0) {
            first = result;
        }
    }
    return first;
}

CnaGoResult cna_go_game_subscribe_events(CnaGoHandle game, uintptr_t context, CnaGoHandle* out_registrations) {
    if (out_registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    for (int i = 0; i < CNA_GO_GAME_EVENT_COUNT; i++) {
        out_registrations[i] = 0;
    }
    for (int i = 0; i < CNA_GO_GAME_EVENT_COUNT; i++) {
        CNA_GameEventRegistrationHandle registration = 0;
        const CNA_Result result = api.cna_game_subscribe(
            game,
            event_identities[i],
            event_callbacks[i],
            (void*)context,
            &registration);
        if (result != 0) {
            (void)cna_go_game_unsubscribe_events(out_registrations);
            return result;
        }
        out_registrations[i] = registration;
    }
    return 0;
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
CnaGoResult cna_go_graphics_device_get_blend_factor(CnaGoHandle device, uint8_t* out_r, uint8_t* out_g, uint8_t* out_b, uint8_t* out_a) {
    CNA_Color color;
    memset(&color, 0, sizeof(color));
    CNA_Result result = api.cna_graphics_device_get_blend_factor(device, &color);
    *out_r = color.r;
    *out_g = color.g;
    *out_b = color.b;
    *out_a = color.a;
    return result;
}

CnaGoResult cna_go_graphics_device_set_blend_factor(CnaGoHandle device, uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
    CNA_Color color;
    memset(&color, 0, sizeof(color));
    color.r = r;
    color.g = g;
    color.b = b;
    color.a = a;
    return api.cna_graphics_device_set_blend_factor(device, color);
}

CnaGoResult cna_go_graphics_device_get_multi_sample_mask(CnaGoHandle device, int32_t* out_mask) {
    return api.cna_graphics_device_get_multi_sample_mask(device, out_mask);
}

CnaGoResult cna_go_graphics_device_set_multi_sample_mask(CnaGoHandle device, int32_t mask) {
    return api.cna_graphics_device_set_multi_sample_mask(device, mask);
}

CnaGoResult cna_go_graphics_device_get_reference_stencil(CnaGoHandle device, int32_t* out_stencil) {
    return api.cna_graphics_device_get_reference_stencil(device, out_stencil);
}

CnaGoResult cna_go_graphics_device_set_reference_stencil(CnaGoHandle device, int32_t stencil) {
    return api.cna_graphics_device_set_reference_stencil(device, stencil);
}

CnaGoResult cna_go_graphics_device_get_scissor_rectangle(CnaGoHandle device, int32_t* out_x, int32_t* out_y, int32_t* out_width, int32_t* out_height) {
    CNA_Rectangle rectangle;
    memset(&rectangle, 0, sizeof(rectangle));
    CNA_Result result = api.cna_graphics_device_get_scissor_rectangle(device, &rectangle);
    *out_x = rectangle.x;
    *out_y = rectangle.y;
    *out_width = rectangle.width;
    *out_height = rectangle.height;
    return result;
}

CnaGoResult cna_go_graphics_device_set_scissor_rectangle(CnaGoHandle device, int32_t x, int32_t y, int32_t width, int32_t height) {
    CNA_Rectangle rectangle;
    memset(&rectangle, 0, sizeof(rectangle));
    rectangle.x = x;
    rectangle.y = y;
    rectangle.width = width;
    rectangle.height = height;
    return api.cna_graphics_device_set_scissor_rectangle(device, rectangle);
}

CnaGoResult cna_go_graphics_device_set_viewport(CnaGoHandle device, int32_t x, int32_t y, int32_t width, int32_t height, float min_depth, float max_depth) {
    CNA_Viewport viewport;
    memset(&viewport, 0, sizeof(viewport));
    viewport.x = x;
    viewport.y = y;
    viewport.width = width;
    viewport.height = height;
    viewport.min_depth = min_depth;
    viewport.max_depth = max_depth;
    return api.cna_graphics_device_set_viewport(device, viewport);
}

CnaGoResult cna_go_graphics_device_get_graphics_profile(CnaGoHandle device, uint32_t* out_profile) {
    return api.cna_graphics_device_get_graphics_profile(device, out_profile);
}

CnaGoResult cna_go_graphics_device_get_status(CnaGoHandle device, uint32_t* out_status) {
    return api.cna_graphics_device_get_status(device, out_status);
}

// fillBlendState and its three siblings build a versioned CNA state descriptor
// from flat scalars. They are static because the descriptor never leaves this
// translation unit: the boundary carries numbers, and the STRUCTURE -- its
// size, its version and every offset -- is CNA-Go's private manifest's, pinned
// against the canonical headers by tools/native_abi.
static void cna_go_fill_blend_state(
    CNA_BlendState* state,
    const uint32_t* blend,
    int32_t multi_sample_mask,
    const uint8_t* blend_factor) {
    memset(state, 0, sizeof(*state));
    state->struct_size = (uint32_t)sizeof(*state);
    state->struct_version = 1;
    state->alpha_blend_function = blend[0];
    state->alpha_destination_blend = blend[1];
    state->alpha_source_blend = blend[2];
    state->color_blend_function = blend[3];
    state->color_destination_blend = blend[4];
    state->color_source_blend = blend[5];
    state->color_write_channels = blend[6];
    state->color_write_channels1 = blend[7];
    state->color_write_channels2 = blend[8];
    state->color_write_channels3 = blend[9];
    state->blend_factor.r = blend_factor[0];
    state->blend_factor.g = blend_factor[1];
    state->blend_factor.b = blend_factor[2];
    state->blend_factor.a = blend_factor[3];
    state->multi_sample_mask = multi_sample_mask;
}

static void cna_go_fill_depth_stencil_state(
    CNA_DepthStencilState* state,
    const uint8_t* flags,
    const uint32_t* words,
    const int32_t* ints) {
    memset(state, 0, sizeof(*state));
    state->struct_size = (uint32_t)sizeof(*state);
    state->struct_version = 1;
    state->depth_buffer_enable = (CNA_Bool)(flags[0] != 0);
    state->depth_buffer_write_enable = (CNA_Bool)(flags[1] != 0);
    state->stencil_enable = (CNA_Bool)(flags[2] != 0);
    state->two_sided_stencil_mode = (CNA_Bool)(flags[3] != 0);
    state->depth_buffer_function = words[0];
    state->stencil_function = words[1];
    state->stencil_fail = words[2];
    state->stencil_depth_buffer_fail = words[3];
    state->stencil_pass = words[4];
    state->counter_clockwise_stencil_function = words[5];
    state->counter_clockwise_stencil_fail = words[6];
    state->counter_clockwise_stencil_depth_buffer_fail = words[7];
    state->counter_clockwise_stencil_pass = words[8];
    state->stencil_mask = ints[0];
    state->stencil_write_mask = ints[1];
    state->reference_stencil = ints[2];
}

static void cna_go_fill_rasterizer_state(
    CNA_RasterizerState* state,
    uint32_t cull_mode,
    uint32_t fill_mode,
    float depth_bias,
    float slope_scale_depth_bias,
    uint8_t multi_sample_anti_alias,
    uint8_t scissor_test_enable) {
    memset(state, 0, sizeof(*state));
    state->struct_size = (uint32_t)sizeof(*state);
    state->struct_version = 1;
    state->cull_mode = cull_mode;
    state->fill_mode = fill_mode;
    state->depth_bias = depth_bias;
    state->slope_scale_depth_bias = slope_scale_depth_bias;
    state->multi_sample_anti_alias = (CNA_Bool)(multi_sample_anti_alias != 0);
    state->scissor_test_enable = (CNA_Bool)(scissor_test_enable != 0);
}

static void cna_go_fill_sampler_state(
    CNA_SamplerState* state,
    const uint32_t* words,
    const int32_t* ints,
    float bias) {
    memset(state, 0, sizeof(*state));
    state->struct_size = (uint32_t)sizeof(*state);
    state->struct_version = 1;
    state->address_u = words[0];
    state->address_v = words[1];
    state->address_w = words[2];
    state->filter = words[3];
    state->max_anisotropy = ints[0];
    state->max_mip_level = ints[1];
    state->mip_map_level_of_detail_bias = bias;
}

CnaGoResult cna_go_graphics_device_get_texture(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    uint8_t* out_bound,
    CnaGoHandle* out_texture) {
    CNA_TextureSlotInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    const CNA_Result result = api.cna_graphics_device_get_texture(device, stage, slot, &info);
    if (result != 0) {
        return result;
    }
    *out_bound = (uint8_t)(info.bound != 0);
    *out_texture = info.texture;
    return result;
}

CnaGoResult cna_go_graphics_device_set_texture(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    CnaGoHandle texture) {
    return api.cna_graphics_device_set_texture(device, stage, slot, texture);
}

CnaGoResult cna_go_graphics_device_get_sampler_state(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    uint32_t* out_words,
    int32_t* out_ints,
    float* out_bias) {
    CNA_SamplerState state;
    memset(&state, 0, sizeof(state));
    state.struct_size = (uint32_t)sizeof(state);
    state.struct_version = 1;
    const CNA_Result result = api.cna_graphics_device_get_sampler_state(device, stage, slot, &state);
    if (result != 0) {
        return result;
    }
    out_words[0] = state.address_u;
    out_words[1] = state.address_v;
    out_words[2] = state.address_w;
    out_words[3] = state.filter;
    out_ints[0] = state.max_anisotropy;
    out_ints[1] = state.max_mip_level;
    *out_bias = state.mip_map_level_of_detail_bias;
    return result;
}

CnaGoResult cna_go_graphics_device_set_sampler_state(
    CnaGoHandle device,
    uint32_t stage,
    uint32_t slot,
    const uint32_t* words,
    const int32_t* ints,
    float bias) {
    CNA_SamplerState state;
    cna_go_fill_sampler_state(&state, words, ints, bias);
    return api.cna_graphics_device_set_sampler_state(device, stage, slot, &state);
}

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
    int32_t multi_sample_mask) {
    const uint32_t blend[10] = {
        alpha_blend_function, alpha_destination_blend, alpha_source_blend,
        color_blend_function, color_destination_blend, color_source_blend,
        color_write_channels, color_write_channels1, color_write_channels2, color_write_channels3};
    const uint8_t factor[4] = {blend_factor_r, blend_factor_g, blend_factor_b, blend_factor_a};
    CNA_BlendState state;
    cna_go_fill_blend_state(&state, blend, multi_sample_mask, factor);
    return api.cna_graphics_device_set_blend_state(device, &state);
}

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
    uint32_t counter_clockwise_stencil_pass) {
    const uint8_t flags[4] = {
        depth_buffer_enable, depth_buffer_write_enable, stencil_enable, two_sided_stencil_mode};
    const uint32_t words[9] = {
        depth_buffer_function, stencil_function, stencil_fail, stencil_depth_buffer_fail,
        stencil_pass, counter_clockwise_stencil_function, counter_clockwise_stencil_fail,
        counter_clockwise_stencil_depth_buffer_fail, counter_clockwise_stencil_pass};
    const int32_t ints[3] = {stencil_mask, stencil_write_mask, reference_stencil};
    CNA_DepthStencilState state;
    cna_go_fill_depth_stencil_state(&state, flags, words, ints);
    return api.cna_graphics_device_set_depth_stencil_state(device, &state);
}

CnaGoResult cna_go_graphics_device_set_rasterizer_state(
    CnaGoHandle device,
    uint32_t cull_mode,
    uint32_t fill_mode,
    float depth_bias,
    float slope_scale_depth_bias,
    uint8_t multi_sample_anti_alias,
    uint8_t scissor_test_enable) {
    CNA_RasterizerState state;
    cna_go_fill_rasterizer_state(&state, cull_mode, fill_mode, depth_bias,
                                 slope_scale_depth_bias, multi_sample_anti_alias, scissor_test_enable);
    return api.cna_graphics_device_set_rasterizer_state(device, &state);
}


CnaGoResult cna_go_render_target2d_create(
    CnaGoHandle device,
    uint32_t width,
    uint32_t height,
    uint8_t mip_map,
    uint32_t format,
    uint32_t depth_format,
    int32_t multi_sample_count,
    uint32_t usage,
    CnaGoHandle* out_render_target) {
    CNA_RenderTarget2DCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.width = width;
    info.height = height;
    info.mip_map = (CNA_Bool)(mip_map != 0);
    info.format = format;
    info.depth_format = depth_format;
    info.multi_sample_count = multi_sample_count;
    info.usage = usage;
    return api.cna_render_target2d_create(device, &info, out_render_target);
}

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
    uint8_t* out_renderer_available) {
    CNA_RenderTargetInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    const CNA_Result result = api.cna_render_target_get_info(render_target, &info);
    if (result != 0) {
        return result;
    }
    *out_kind = info.kind;
    *out_width = info.width;
    *out_height = info.height;
    *out_level_count = info.level_count;
    *out_format = info.format;
    *out_depth_format = info.depth_format;
    *out_multi_sample_count = info.multi_sample_count;
    *out_usage = info.usage;
    *out_is_content_lost = (uint8_t)(info.is_content_lost != 0);
    *out_renderer_available = (uint8_t)(info.renderer_available != 0);
    return result;
}

CnaGoResult cna_go_render_target_destroy(CnaGoHandle render_target) {
    return api.cna_render_target_destroy(render_target);
}

CnaGoResult cna_go_graphics_device_set_render_target2d(CnaGoHandle device, CnaGoHandle render_target) {
    return api.cna_graphics_device_set_render_target2d(device, render_target);
}

CnaGoResult cna_go_graphics_device_get_is_disposed(CnaGoHandle device, uint8_t* out_is_disposed) {
    return api.cna_graphics_device_get_is_disposed(device, out_is_disposed);
}

CnaGoResult cna_go_graphics_device_clear_options(CnaGoHandle device, uint32_t options, uint8_t r, uint8_t g, uint8_t b, uint8_t a, float depth, int32_t stencil) {
    CNA_Color color;
    memset(&color, 0, sizeof(color));
    color.r = r;
    color.g = g;
    color.b = b;
    color.a = a;
    return api.cna_graphics_device_clear_options(device, options, color, depth, stencil);
}

CnaGoResult cna_go_graphics_device_present(CnaGoHandle device) {
    return api.cna_graphics_device_present(device);
}

CnaGoResult cna_go_graphics_device_get_display_mode(CnaGoHandle device, int32_t* out_width, int32_t* out_height, float* out_aspect_ratio, uint32_t* out_format) {
    CNA_DisplayMode mode;
    memset(&mode, 0, sizeof(mode));
    mode.struct_size = (uint32_t)sizeof(mode);
    mode.struct_version = 1;
    CNA_Result result = api.cna_graphics_device_get_display_mode(device, &mode);
    *out_width = mode.width;
    *out_height = mode.height;
    *out_aspect_ratio = mode.aspect_ratio;
    *out_format = mode.format;
    return result;
}

CnaGoResult cna_go_texture2d_create(CnaGoHandle device, uint32_t width, uint32_t height, uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture) {
    CNA_Texture2DCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.width = width;
    info.height = height;
    info.mip_map = mip_map;
    info.format = format;
    return api.cna_texture2d_create(device, &info, out_texture);
}

CnaGoResult cna_go_texture2d_create_from_encoded_memory_sized(CnaGoHandle device, const uint8_t* data, uint64_t byte_count, uint32_t width, uint32_t height, uint8_t zoom, CnaGoHandle* out_texture) {
    CNA_Texture2DDecodeInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.width = width;
    info.height = height;
    info.zoom = zoom;
    return api.cna_texture2d_create_from_encoded_memory(device, data, byte_count, &info, out_texture);
}

CnaGoResult cna_go_texture2d_get_encoded_byte_count(CnaGoHandle texture, uint32_t image_format, uint32_t width, uint32_t height, uint64_t* out_byte_count) {
    return api.cna_texture2d_get_encoded_byte_count(texture, image_format, width, height, out_byte_count);
}

CnaGoResult cna_go_texture2d_copy_encoded(CnaGoHandle texture, uint32_t image_format, uint32_t width, uint32_t height, uint8_t* destination, uint64_t capacity, uint64_t* out_byte_count) {
    return api.cna_texture2d_copy_encoded(texture, image_format, width, height, destination, capacity, out_byte_count);
}

static void cna_go_fill_texture_transfer(CNA_Texture2DTransfer* transfer, int32_t level, uint8_t has_rectangle, int32_t rect_x, int32_t rect_y, int32_t rect_width, int32_t rect_height, uint64_t start_index, uint64_t element_count) {
    memset(transfer, 0, sizeof(*transfer));
    transfer->struct_size = (uint32_t)sizeof(*transfer);
    transfer->struct_version = 1;
    transfer->level = level;
    transfer->has_rectangle = has_rectangle;
    transfer->rectangle.x = rect_x;
    transfer->rectangle.y = rect_y;
    transfer->rectangle.width = rect_width;
    transfer->rectangle.height = rect_height;
    transfer->start_index = start_index;
    transfer->element_count = element_count;
}

CnaGoResult cna_go_texture2d_set_data(CnaGoHandle texture, uint32_t data_type, int32_t level, uint8_t has_rectangle, int32_t rect_x, int32_t rect_y, int32_t rect_width, int32_t rect_height, uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity) {
    CNA_Texture2DTransfer transfer;
    cna_go_fill_texture_transfer(&transfer, level, has_rectangle, rect_x, rect_y, rect_width, rect_height, start_index, element_count);
    return api.cna_texture2d_set_data(texture, data_type, &transfer, data, data_capacity);
}

CnaGoResult cna_go_texture2d_get_data(CnaGoHandle texture, uint32_t data_type, int32_t level, uint8_t has_rectangle, int32_t rect_x, int32_t rect_y, int32_t rect_width, int32_t rect_height, uint64_t start_index, uint64_t element_count, void* destination, uint64_t destination_capacity, uint64_t* out_required_elements) {
    CNA_Texture2DTransfer transfer;
    cna_go_fill_texture_transfer(&transfer, level, has_rectangle, rect_x, rect_y, rect_width, rect_height, start_index, element_count);
    return api.cna_texture2d_get_data(texture, data_type, &transfer, destination, destination_capacity, out_required_elements);
}

CnaGoResult cna_go_sprite_batch_draw_destination(CnaGoHandle batch, CnaGoHandle texture, int32_t destination_x, int32_t destination_y, int32_t destination_width, int32_t destination_height, int32_t source_x, int32_t source_y, int32_t source_width, int32_t source_height, uint8_t color_r, uint8_t color_g, uint8_t color_b, uint8_t color_a, float rotation, float origin_x, float origin_y, uint32_t effects, float layer_depth) {
    CNA_SpriteCommand command;
    memset(&command, 0, sizeof(command));
    command.struct_size = (uint32_t)sizeof(command);
    command.struct_version = 1;
    command.texture = texture;
    command.destination.x = destination_x;
    command.destination.y = destination_y;
    command.destination.width = destination_width;
    command.destination.height = destination_height;
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
    command.effects = effects;
    command.layer_depth = layer_depth;
    return api.cna_sprite_batch_submit_many(batch, &command, 1);
}
CnaGoResult cna_go_sprite_batch_end(CnaGoHandle batch) { return api.cna_sprite_batch_end(batch); }
CnaGoResult cna_go_sprite_batch_destroy(CnaGoHandle batch) { return api.cna_sprite_batch_destroy(batch); }

extern void cnaGoGraphicsDeviceManagerEvent(uint32_t event, uintptr_t context);

/* One trampoline per canonical manager event identity, for the third time and
   the same reason: CNA_GameEventCallback carries only the caller context, so
   the identity has to come from the function that was registered. */
#define CNA_GO_GDM_EVENT_CALLBACK(name, event) \
    static void name(void* context) { \
        cnaGoGraphicsDeviceManagerEvent((uint32_t)(event), (uintptr_t)context); \
    }

CNA_GO_GDM_EVENT_CALLBACK(gdm_event_disposed, CNA_GO_GDM_EVENT_DISPOSED)
CNA_GO_GDM_EVENT_CALLBACK(gdm_event_device_created, CNA_GO_GDM_EVENT_DEVICE_CREATED)
CNA_GO_GDM_EVENT_CALLBACK(gdm_event_device_disposing, CNA_GO_GDM_EVENT_DEVICE_DISPOSING)
CNA_GO_GDM_EVENT_CALLBACK(gdm_event_device_reset, CNA_GO_GDM_EVENT_DEVICE_RESET)
CNA_GO_GDM_EVENT_CALLBACK(gdm_event_device_resetting, CNA_GO_GDM_EVENT_DEVICE_RESETTING)

static const CNA_GameEventCallback gdm_event_callbacks[CNA_GO_GDM_EVENT_COUNT] = {
    gdm_event_disposed,
    gdm_event_device_created,
    gdm_event_device_disposing,
    gdm_event_device_reset,
    gdm_event_device_resetting
};

static const CNA_GraphicsDeviceManagerEvent gdm_event_identities[CNA_GO_GDM_EVENT_COUNT] = {
    CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DISPOSED,
    CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_CREATED,
    CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_DISPOSING,
    CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESET,
    CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESETTING
};

_Static_assert(CNA_GO_GDM_EVENT_DISPOSED == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DISPOSED, "manager disposal identity drift");
_Static_assert(CNA_GO_GDM_EVENT_DEVICE_CREATED == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_CREATED, "manager device-created identity drift");
_Static_assert(CNA_GO_GDM_EVENT_DEVICE_DISPOSING == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_DISPOSING, "manager device-disposing identity drift");
_Static_assert(CNA_GO_GDM_EVENT_DEVICE_RESET == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESET, "manager device-reset identity drift");
_Static_assert(CNA_GO_GDM_EVENT_DEVICE_RESETTING == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_DEVICE_RESETTING, "manager device-resetting identity drift");
_Static_assert(CNA_GO_GDM_EVENT_COUNT == CNA_GRAPHICS_DEVICE_MANAGER_EVENT_MAXIMUM + 1, "manager event count drift");

/* The three families' first device event sits at a DIFFERENT index in each, so
   a table shared between them would be silently wrong. */
_Static_assert(CNA_GO_GDM_EVENT_DEVICE_CREATED != CNA_GO_GAME_EVENT_ACTIVATED,
               "the manager and game families must not be indexed as one");

CnaGoResult cna_go_graphics_device_manager_create_device(CnaGoHandle manager) {
    return api.cna_graphics_device_manager_create_device(manager);
}

CnaGoResult cna_go_graphics_device_manager_begin_draw(CnaGoHandle manager, uint8_t* out_should_draw) {
    CNA_Bool should = 0;
    const CNA_Result result = api.cna_graphics_device_manager_begin_draw(manager, &should);
    if (result == 0) {
        *out_should_draw = (uint8_t)(should != 0);
    }
    return result;
}

CnaGoResult cna_go_graphics_device_manager_end_draw(CnaGoHandle manager) {
    return api.cna_graphics_device_manager_end_draw(manager);
}

CnaGoResult cna_go_graphics_device_manager_unsubscribe_events(CnaGoHandle* registrations) {
    if (registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    CNA_Result first = 0;
    for (int i = 0; i < CNA_GO_GDM_EVENT_COUNT; i++) {
        if (registrations[i] == 0) {
            continue;
        }
        const CNA_Result result = api.cna_game_unsubscribe(registrations[i]);
        registrations[i] = 0;
        if (result != 0 && first == 0) {
            first = result;
        }
    }
    return first;
}

CnaGoResult cna_go_graphics_device_manager_subscribe_events(CnaGoHandle manager, uintptr_t context, CnaGoHandle* out_registrations) {
    if (out_registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    for (int i = 0; i < CNA_GO_GDM_EVENT_COUNT; i++) {
        out_registrations[i] = 0;
    }
    for (int i = 0; i < CNA_GO_GDM_EVENT_COUNT; i++) {
        CNA_GameEventRegistrationHandle registration = 0;
        const CNA_Result result = api.cna_graphics_device_manager_subscribe(
            manager,
            gdm_event_identities[i],
            gdm_event_callbacks[i],
            (void*)context,
            &registration);
        if (result != 0) {
            (void)cna_go_graphics_device_manager_unsubscribe_events(out_registrations);
            return result;
        }
        out_registrations[i] = registration;
    }
    return 0;
}

extern void cnaGoGraphicsDeviceEvent(uint32_t event, uintptr_t context);
extern void cnaGoGraphicsDeviceResourceCreated(uint8_t has_resource, uintptr_t context);
extern void cnaGoGraphicsDeviceResourceDestroyed(
    uint8_t has_tag, const char* name, uint64_t name_length, uintptr_t context);

/* One trampoline per canonical DEVICE event identity, for the fourth time and
   the same reason: CNA_GraphicsDeviceEventCallback carries the device handle and
   the caller context, not the identity, so the identity has to come from the
   function that was registered.

   The device handle the callback receives is deliberately DISCARDED. It is the
   callback-scoped handle CNA already lends the Go side through the facade, and
   retaining it would be the one thing the borrowed-device rule forbids. */
#define CNA_GO_DEVICE_EVENT_CALLBACK(name, event) \
    static void name(CNA_Handle device, void* context) { \
        (void)device; \
        cnaGoGraphicsDeviceEvent((uint32_t)(event), (uintptr_t)context); \
    }

CNA_GO_DEVICE_EVENT_CALLBACK(device_event_disposing, CNA_GO_DEVICE_EVENT_DISPOSING)
CNA_GO_DEVICE_EVENT_CALLBACK(device_event_device_lost, CNA_GO_DEVICE_EVENT_DEVICE_LOST)
CNA_GO_DEVICE_EVENT_CALLBACK(device_event_device_reset, CNA_GO_DEVICE_EVENT_DEVICE_RESET)
CNA_GO_DEVICE_EVENT_CALLBACK(device_event_device_resetting, CNA_GO_DEVICE_EVENT_DEVICE_RESETTING)

static void device_event_resource_created(
    CNA_Handle device, const CNA_ResourceCreatedEventInfo* info, void* context) {
    (void)device;
    cnaGoGraphicsDeviceResourceCreated(
        (uint8_t)(info != NULL && info->has_resource != 0), (uintptr_t)context);
}

static void device_event_resource_destroyed(
    CNA_Handle device, const CNA_ResourceDestroyedEventInfo* info, void* context) {
    (void)device;
    if (info == NULL) {
        cnaGoGraphicsDeviceResourceDestroyed(0, NULL, 0, (uintptr_t)context);
        return;
    }
    cnaGoGraphicsDeviceResourceDestroyed(
        (uint8_t)(info->has_tag != 0), info->name.data, info->name.byte_length, (uintptr_t)context);
}

static const CNA_GraphicsDeviceEventCallback device_event_callbacks[4] = {
    device_event_disposing,
    device_event_device_lost,
    device_event_device_reset,
    device_event_device_resetting
};

static const CNA_GraphicsDeviceEvent device_event_identities[4] = {
    CNA_GRAPHICS_DEVICE_EVENT_DISPOSING,
    CNA_GRAPHICS_DEVICE_EVENT_DEVICE_LOST,
    CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESET,
    CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING
};

_Static_assert(CNA_GO_DEVICE_EVENT_DISPOSING == CNA_GRAPHICS_DEVICE_EVENT_DISPOSING, "device disposing identity drift");
_Static_assert(CNA_GO_DEVICE_EVENT_DEVICE_LOST == CNA_GRAPHICS_DEVICE_EVENT_DEVICE_LOST, "device lost identity drift");
_Static_assert(CNA_GO_DEVICE_EVENT_DEVICE_RESET == CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESET, "device reset identity drift");
_Static_assert(CNA_GO_DEVICE_EVENT_DEVICE_RESETTING == CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING, "device resetting identity drift");

/* The two payload events sit ABOVE CNA's own identity space, which has no name
   for them: they are separate routes rather than identities. Asserting that
   they do not collide with a canonical one is what stops a future CNA identity
   from silently aliasing them. */
_Static_assert(CNA_GO_DEVICE_EVENT_RESOURCE_CREATED > CNA_GRAPHICS_DEVICE_EVENT_DEVICE_RESETTING,
               "the payload-carrying device events must not alias a canonical identity");

CnaGoResult cna_go_graphics_device_unsubscribe_events(CnaGoHandle* registrations) {
    if (registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    CNA_Result first = 0;
    for (int i = 0; i < CNA_GO_DEVICE_EVENT_COUNT; i++) {
        if (registrations[i] == 0) {
            continue;
        }
        const CNA_Result result = api.cna_graphics_device_unsubscribe(registrations[i]);
        registrations[i] = 0;
        if (result != 0 && first == 0) {
            first = result;
        }
    }
    return first;
}

CnaGoResult cna_go_graphics_device_subscribe_events(
    CnaGoHandle device, uintptr_t context, CnaGoHandle* out_registrations) {
    if (out_registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    for (int i = 0; i < CNA_GO_DEVICE_EVENT_COUNT; i++) {
        out_registrations[i] = 0;
    }
    for (int i = 0; i < 4; i++) {
        CNA_GraphicsDeviceEventRegistrationHandle registration = 0;
        const CNA_Result result = api.cna_graphics_device_subscribe_event(
            device, device_event_identities[i], device_event_callbacks[i], (void*)context, &registration);
        if (result != 0) {
            (void)cna_go_graphics_device_unsubscribe_events(out_registrations);
            return result;
        }
        out_registrations[i] = registration;
    }
    CNA_GraphicsDeviceEventRegistrationHandle created = 0;
    CNA_Result result = api.cna_graphics_device_subscribe_resource_created(
        device, device_event_resource_created, (void*)context, &created);
    if (result != 0) {
        (void)cna_go_graphics_device_unsubscribe_events(out_registrations);
        return result;
    }
    out_registrations[CNA_GO_DEVICE_EVENT_RESOURCE_CREATED] = created;
    CNA_GraphicsDeviceEventRegistrationHandle destroyed = 0;
    result = api.cna_graphics_device_subscribe_resource_destroyed(
        device, device_event_resource_destroyed, (void*)context, &destroyed);
    if (result != 0) {
        (void)cna_go_graphics_device_unsubscribe_events(out_registrations);
        return result;
    }
    out_registrations[CNA_GO_DEVICE_EVENT_RESOURCE_DESTROYED] = destroyed;
    return 0;
}

CnaGoResult cna_go_graphics_device_dispose(CnaGoHandle device) {
    return api.cna_graphics_device_dispose(device);
}

static CNA_StringView cna_go_view(const char* data, uint64_t length) {
    CNA_StringView view;
    view.data = data;
    view.byte_length = length;
    return view;
}

static void cna_go_fill_index_transfer(
    CNA_IndexBufferTransfer* transfer,
    uint32_t index_element_size,
    uint32_t options,
    uint64_t start_index,
    uint64_t element_count) {
    memset(transfer, 0, sizeof(*transfer));
    transfer->struct_size = (uint32_t)sizeof(*transfer);
    transfer->struct_version = 1;
    transfer->index_element_size = index_element_size;
    transfer->options = options;
    transfer->start_index = start_index;
    transfer->element_count = element_count;
}

CnaGoResult cna_go_index_buffer_create(
    CnaGoHandle device,
    int32_t index_count,
    uint32_t index_element_size,
    uint32_t buffer_usage,
    uint8_t dynamic,
    CnaGoHandle* out_index_buffer) {
    CNA_IndexBufferCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.index_count = index_count;
    info.index_element_size = index_element_size;
    info.buffer_usage = buffer_usage;
    info.dynamic = (CNA_Bool)(dynamic != 0);
    return api.cna_index_buffer_create(device, &info, out_index_buffer);
}

CnaGoResult cna_go_index_buffer_destroy(CnaGoHandle index_buffer) {
    return api.cna_index_buffer_destroy(index_buffer);
}

CnaGoResult cna_go_index_buffer_get_info(
    CnaGoHandle index_buffer,
    int32_t* out_index_count,
    uint32_t* out_index_element_size,
    uint32_t* out_buffer_usage,
    uint8_t* out_dynamic,
    uint8_t* out_is_content_lost,
    uint8_t* out_has_renderer) {
    CNA_IndexBufferInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_index_buffer_get_info(index_buffer, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_index_count = info.index_count;
    *out_index_element_size = info.index_element_size;
    *out_buffer_usage = info.buffer_usage;
    *out_dynamic = info.dynamic ? 1u : 0u;
    *out_is_content_lost = info.is_content_lost ? 1u : 0u;
    *out_has_renderer = info.has_renderer ? 1u : 0u;
    return result;
}

CnaGoResult cna_go_index_buffer_set_data(
    CnaGoHandle index_buffer,
    uint32_t index_element_size,
    uint32_t options,
    uint64_t start_index,
    uint64_t element_count,
    const void* data,
    uint64_t capacity) {
    CNA_IndexBufferTransfer transfer;
    cna_go_fill_index_transfer(&transfer, index_element_size, options, start_index, element_count);
    return api.cna_index_buffer_set_data(index_buffer, &transfer, data, capacity);
}

CnaGoResult cna_go_index_buffer_set_data_at(
    CnaGoHandle index_buffer,
    uint64_t buffer_offset_in_bytes,
    uint32_t index_element_size,
    uint32_t options,
    uint64_t start_index,
    uint64_t element_count,
    const void* data,
    uint64_t capacity) {
    CNA_IndexBufferTransfer transfer;
    cna_go_fill_index_transfer(&transfer, index_element_size, options, start_index, element_count);
    return api.cna_index_buffer_set_data_at(index_buffer, buffer_offset_in_bytes, &transfer, data, capacity);
}

CnaGoResult cna_go_index_buffer_get_data(
    CnaGoHandle index_buffer,
    uint32_t index_element_size,
    uint64_t start_index,
    uint64_t element_count,
    void* destination,
    uint64_t capacity,
    uint64_t* out_element_count) {
    CNA_IndexBufferTransfer transfer;
    // GetData's options must be None; CNA rejects any other value here.
    cna_go_fill_index_transfer(&transfer, index_element_size, 0u, start_index, element_count);
    return api.cna_index_buffer_get_data(index_buffer, &transfer, destination, capacity, out_element_count);
}

CnaGoResult cna_go_content_manager_create(
    CnaGoHandle device,
    const char* root_directory,
    uint64_t root_directory_length,
    CnaGoHandle* out_content_manager) {
    CNA_ContentManagerCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.root_directory = cna_go_view(root_directory, root_directory_length);
    return api.cna_content_manager_create(device, &info, out_content_manager);
}

CnaGoResult cna_go_content_manager_destroy(CnaGoHandle content_manager) {
    return api.cna_content_manager_destroy(content_manager);
}

CnaGoResult cna_go_content_manager_get_root_directory_size(CnaGoHandle content_manager, uint64_t* out_bytes) {
    return api.cna_content_manager_get_root_directory_size(content_manager, out_bytes);
}

CnaGoResult cna_go_content_manager_copy_root_directory(
    CnaGoHandle content_manager, char* destination, uint64_t capacity, uint64_t* out_bytes) {
    return api.cna_content_manager_copy_root_directory(content_manager, destination, capacity, out_bytes);
}

CnaGoResult cna_go_content_manager_set_root_directory(
    CnaGoHandle content_manager, const char* root_directory, uint64_t root_directory_length) {
    return api.cna_content_manager_set_root_directory(
        content_manager, cna_go_view(root_directory, root_directory_length));
}

CnaGoResult cna_go_content_manager_unload(CnaGoHandle content_manager) {
    return api.cna_content_manager_unload(content_manager);
}

CnaGoResult cna_go_content_manager_load_texture2d(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, CnaGoHandle* out_texture) {
    return api.cna_content_manager_load_texture2d(
        content_manager, cna_go_view(asset_name, asset_name_length), out_texture);
}

CnaGoResult cna_go_content_manager_get_asset_path_size(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, uint64_t* out_bytes) {
    return api.cna_content_manager_get_asset_path_size(
        content_manager, cna_go_view(asset_name, asset_name_length), out_bytes);
}

CnaGoResult cna_go_content_manager_copy_asset_path(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length,
    char* destination, uint64_t capacity, uint64_t* out_bytes) {
    return api.cna_content_manager_copy_asset_path(
        content_manager, cna_go_view(asset_name, asset_name_length), destination, capacity, out_bytes);
}

/* Foundation 73 -- device reset, presentation parameters and back-buffer
   readback.
   The eight int32-shaped presentation fields cross as ONE array in a fixed
   order rather than as eight arguments, so a field added on either side is a
   length change the layout probe sees rather than an argument a compiler
   silently accepts in the wrong position. */
#define CNA_GO_PRESENTATION_FORMAT 0
#define CNA_GO_PRESENTATION_WIDTH 1
#define CNA_GO_PRESENTATION_HEIGHT 2
#define CNA_GO_PRESENTATION_DEPTH_FORMAT 3
#define CNA_GO_PRESENTATION_MULTI_SAMPLE 4
#define CNA_GO_PRESENTATION_INTERVAL 5
#define CNA_GO_PRESENTATION_ORIENTATION 6
#define CNA_GO_PRESENTATION_USAGE 7
#define CNA_GO_PRESENTATION_COUNT 8

static void cna_go_fill_presentation(
    CNA_PresentationParameters* parameters, const int32_t* ints,
    uint8_t is_full_screen, uint8_t headless) {
    memset(parameters, 0, sizeof(*parameters));
    parameters->struct_size = (uint32_t)sizeof(*parameters);
    parameters->struct_version = 1;
    parameters->back_buffer_format = (CNA_SurfaceFormat)ints[CNA_GO_PRESENTATION_FORMAT];
    parameters->back_buffer_width = ints[CNA_GO_PRESENTATION_WIDTH];
    parameters->back_buffer_height = ints[CNA_GO_PRESENTATION_HEIGHT];
    parameters->depth_stencil_format = (CNA_DepthFormat)ints[CNA_GO_PRESENTATION_DEPTH_FORMAT];
    parameters->multi_sample_count = ints[CNA_GO_PRESENTATION_MULTI_SAMPLE];
    parameters->presentation_interval = (CNA_PresentInterval)ints[CNA_GO_PRESENTATION_INTERVAL];
    parameters->display_orientation = (CNA_DisplayOrientation)ints[CNA_GO_PRESENTATION_ORIENTATION];
    parameters->render_target_usage = (CNA_RenderTargetUsage)ints[CNA_GO_PRESENTATION_USAGE];
    parameters->is_full_screen = (CNA_Bool)(is_full_screen != 0);
    parameters->headless_ext = (CNA_Bool)(headless != 0);
}

CnaGoResult cna_go_render_target_cube_create(
    CnaGoHandle device, uint32_t size, uint8_t mip_map, uint32_t format, uint32_t depth_format,
    int32_t multi_sample_count, uint32_t usage, CnaGoHandle* out_render_target) {
    CNA_RenderTargetCubeCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.size = size;
    info.mip_map = (CNA_Bool)(mip_map != 0);
    info.format = format;
    info.depth_format = depth_format;
    info.multi_sample_count = multi_sample_count;
    info.usage = usage;
    return api.cna_render_target_cube_create(device, &info, out_render_target);
}

CnaGoResult cna_go_graphics_device_set_render_target_cube(
    CnaGoHandle device, CnaGoHandle render_target, uint32_t cube_map_face) {
    return api.cna_graphics_device_set_render_target_cube(device, render_target, cube_map_face);
}

/* The binding array crosses as TWO parallel arrays, handles and faces, because
   no CNA struct crosses cgo. array_slice is left at zero for every element:
   CNA requires it to be zero for a cube target, and refuses a nonzero one for a
   2D target outright, so the only value the canonical contract admits is the
   one this fills in. */
CnaGoResult cna_go_graphics_device_set_render_targets(
    CnaGoHandle device, const CnaGoHandle* handles, const uint32_t* faces, uint64_t count) {
    CNA_RenderTargetBinding* bindings;
    CnaGoResult result;
    uint64_t at;
    if (count == 0) {
        return api.cna_graphics_device_set_render_targets(device, NULL, 0);
    }
    bindings = (CNA_RenderTargetBinding*)calloc((size_t)count, sizeof(CNA_RenderTargetBinding));
    if (bindings == NULL) {
        return CNA_GO_RESULT_OUT_OF_MEMORY;
    }
    for (at = 0; at < count; at++) {
        bindings[at].struct_size = (uint32_t)sizeof(CNA_RenderTargetBinding);
        bindings[at].struct_version = 1;
        bindings[at].render_target = handles[at];
        bindings[at].array_slice = 0;
        bindings[at].cube_map_face = faces[at];
    }
    result = api.cna_graphics_device_set_render_targets(device, bindings, count);
    free(bindings);
    return result;
}

CnaGoResult cna_go_graphics_device_get_render_target_count(CnaGoHandle device, uint64_t* out_count) {
    return api.cna_graphics_device_get_render_target_count(device, out_count);
}

CnaGoResult cna_go_graphics_device_create(
    uint32_t adapter_index, uint32_t graphics_profile, const int32_t* ints,
    uint8_t is_full_screen, uint8_t headless, CnaGoHandle* out_device) {
    CNA_PresentationParameters parameters;
    cna_go_fill_presentation(&parameters, ints, is_full_screen, headless);
    return api.cna_graphics_device_create(adapter_index, graphics_profile, &parameters, out_device);
}

CnaGoResult cna_go_graphics_device_destroy(CnaGoHandle device) {
    return api.cna_graphics_device_destroy(device);
}

CnaGoResult cna_go_graphics_device_reset(CnaGoHandle device) {
    return api.cna_graphics_device_reset(device);
}

CnaGoResult cna_go_graphics_device_reset_with_parameters(
    CnaGoHandle device, const int32_t* ints, uint8_t is_full_screen, uint8_t headless,
    uint8_t has_adapter, uint32_t adapter_index) {
    CNA_PresentationParameters parameters;
    cna_go_fill_presentation(&parameters, ints, is_full_screen, headless);
    return api.cna_graphics_device_reset_with_parameters(
        device, &parameters, has_adapter != 0 ? &adapter_index : NULL);
}

CnaGoResult cna_go_graphics_device_get_presentation_parameters(
    CnaGoHandle device, int32_t* out_ints, uint8_t* out_is_full_screen, uint8_t* out_headless) {
    CNA_PresentationParameters parameters;
    CnaGoResult result;
    memset(&parameters, 0, sizeof(parameters));
    parameters.struct_size = (uint32_t)sizeof(parameters);
    parameters.struct_version = 1;
    result = api.cna_graphics_device_get_presentation_parameters(device, &parameters);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    out_ints[CNA_GO_PRESENTATION_FORMAT] = (int32_t)parameters.back_buffer_format;
    out_ints[CNA_GO_PRESENTATION_WIDTH] = parameters.back_buffer_width;
    out_ints[CNA_GO_PRESENTATION_HEIGHT] = parameters.back_buffer_height;
    out_ints[CNA_GO_PRESENTATION_DEPTH_FORMAT] = (int32_t)parameters.depth_stencil_format;
    out_ints[CNA_GO_PRESENTATION_MULTI_SAMPLE] = parameters.multi_sample_count;
    out_ints[CNA_GO_PRESENTATION_INTERVAL] = (int32_t)parameters.presentation_interval;
    out_ints[CNA_GO_PRESENTATION_ORIENTATION] = (int32_t)parameters.display_orientation;
    out_ints[CNA_GO_PRESENTATION_USAGE] = (int32_t)parameters.render_target_usage;
    *out_is_full_screen = parameters.is_full_screen ? 1u : 0u;
    *out_headless = parameters.headless_ext ? 1u : 0u;
    return result;
}

CnaGoResult cna_go_graphics_device_get_backbuffer_data_window(
    CnaGoHandle device, uint8_t has_rectangle, int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity) {
    CNA_BackBufferReadback readback;
    memset(&readback, 0, sizeof(readback));
    readback.struct_size = (uint32_t)sizeof(readback);
    readback.struct_version = 1;
    readback.has_source_rectangle = (CNA_Bool)(has_rectangle != 0);
    readback.source_rectangle.x = x;
    readback.source_rectangle.y = y;
    readback.source_rectangle.width = width;
    readback.source_rectangle.height = height;
    readback.start_index = start_index;
    readback.element_count = element_count;
    return api.cna_graphics_device_get_backbuffer_data_window(
        device, &readback, (CNA_Color*)destination, capacity);
}

/* Foundation 73 -- the six user-primitive draws. */
static void cna_go_fill_user_primitives(
    CNA_UserPrimitives* primitives, uint32_t primitive_type, uint32_t vertex_source,
    const void* vertex_data, CnaGoHandle vertex_declaration,
    int32_t vertex_offset, int32_t num_vertices, int32_t primitive_count) {
    memset(primitives, 0, sizeof(*primitives));
    primitives->struct_size = (uint32_t)sizeof(*primitives);
    primitives->struct_version = 1;
    primitives->primitive_type = primitive_type;
    primitives->vertex_source = vertex_source;
    primitives->vertex_data = vertex_data;
    primitives->vertex_declaration = vertex_declaration;
    primitives->vertex_offset = vertex_offset;
    primitives->num_vertices = num_vertices;
    primitives->primitive_count = primitive_count;
}

CnaGoResult cna_go_graphics_device_draw_user_primitives(
    CnaGoHandle device, uint32_t primitive_type, uint32_t vertex_source,
    const void* vertex_data, CnaGoHandle vertex_declaration,
    int32_t vertex_offset, int32_t num_vertices, int32_t primitive_count) {
    CNA_UserPrimitives primitives;
    cna_go_fill_user_primitives(&primitives, primitive_type, vertex_source, vertex_data,
        vertex_declaration, vertex_offset, num_vertices, primitive_count);
    return api.cna_graphics_device_draw_user_primitives(device, &primitives);
}

CnaGoResult cna_go_graphics_device_draw_user_indexed_primitives(
    CnaGoHandle device, uint32_t primitive_type, uint32_t vertex_source,
    const void* vertex_data, CnaGoHandle vertex_declaration,
    int32_t vertex_offset, int32_t num_vertices, int32_t primitive_count,
    uint32_t index_element_size, int32_t index_offset, const void* index_data) {
    CNA_UserPrimitives primitives;
    CNA_UserIndices indices;
    cna_go_fill_user_primitives(&primitives, primitive_type, vertex_source, vertex_data,
        vertex_declaration, vertex_offset, num_vertices, primitive_count);
    memset(&indices, 0, sizeof(indices));
    indices.struct_size = (uint32_t)sizeof(indices);
    indices.struct_version = 1;
    indices.index_element_size = index_element_size;
    indices.index_offset = index_offset;
    indices.index_data = index_data;
    return api.cna_graphics_device_draw_user_indexed_primitives(device, &primitives, &indices);
}

/* Foundation 72 -- the Effect cluster. */
/* The effect-bearing Begin. CNA expresses BOTH canonical overloads with one
   route: a null transform is the identity the effect-only overload uses, and
   CNA_INVALID_HANDLE selects the default sprite effect, which is what a null
   Effect means to the canonical call. */
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
    const float* transform) {
    CNA_BlendState blendState;
    CNA_SamplerState samplerState;
    CNA_DepthStencilState depthState;
    CNA_RasterizerState rasterizerState;
    CNA_Matrix matrix;
    cna_go_fill_blend_state(&blendState, blend, blend_mask[0], blend_factor);
    cna_go_fill_sampler_state(&samplerState, sampler, sampler_ints, sampler_bias);
    cna_go_fill_depth_stencil_state(&depthState, depth_flags, depth_words, depth_ints);
    cna_go_fill_rasterizer_state(&rasterizerState, cull_mode, fill_mode, depth_bias,
                                 slope_scale_depth_bias, multi_sample_anti_alias, scissor_test_enable);
    if (has_transform != 0) {
        memcpy(&matrix, transform, sizeof(matrix));
    }
    return api.cna_sprite_batch_begin_with_effect(
        batch, sort_mode, &blendState, &samplerState, &depthState, &rasterizerState,
        effect, has_transform != 0 ? &matrix : NULL);
}

CnaGoResult cna_go_effect_create_compiled(
    CnaGoHandle device, const uint8_t* effect_code, uint64_t effect_code_count, CnaGoHandle* out_effect) {
    return api.cna_effect_create_compiled(device, effect_code, effect_code_count, out_effect);
}

CnaGoResult cna_go_content_manager_load_effect(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length, CnaGoHandle* out_effect) {
    return api.cna_content_manager_load_effect(
        content_manager, cna_go_view(asset_name, asset_name_length), out_effect);
}

/* The six string reads in the effect cluster share ONE trampoline, selected by a
   kind, because their size-then-copy pair is identical in every one and six
   copies of it would be six chances to pair the wrong two routes.
   A zero capacity asks only for the size. */
#define CNA_GO_EFFECT_STRING_TECHNIQUE_NAME 0u
#define CNA_GO_EFFECT_STRING_PASS_NAME 1u
#define CNA_GO_EFFECT_STRING_PARAMETER_NAME 2u
#define CNA_GO_EFFECT_STRING_PARAMETER_SEMANTIC 3u
#define CNA_GO_EFFECT_STRING_PARAMETER_VALUE 4u
#define CNA_GO_EFFECT_STRING_ANNOTATION_NAME 5u
#define CNA_GO_EFFECT_STRING_ANNOTATION_SEMANTIC 6u
#define CNA_GO_EFFECT_STRING_ANNOTATION_VALUE 7u

CnaGoResult cna_go_effect_string(
    uint32_t kind, CnaGoHandle handle, char* destination, uint64_t capacity, uint64_t* out_bytes) {
    if (capacity == 0) {
        switch (kind) {
        case CNA_GO_EFFECT_STRING_TECHNIQUE_NAME:
            return api.cna_effect_technique_get_name_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_PASS_NAME:
            return api.cna_effect_pass_get_name_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_PARAMETER_NAME:
            return api.cna_effect_parameter_get_name_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_PARAMETER_SEMANTIC:
            return api.cna_effect_parameter_get_semantic_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_PARAMETER_VALUE:
            return api.cna_effect_parameter_get_value_string_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_ANNOTATION_NAME:
            return api.cna_effect_annotation_get_name_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_ANNOTATION_SEMANTIC:
            return api.cna_effect_annotation_get_semantic_byte_count(handle, out_bytes);
        case CNA_GO_EFFECT_STRING_ANNOTATION_VALUE:
            return api.cna_effect_annotation_get_value_string_byte_count(handle, out_bytes);
        default:
            return CNA_GO_RESULT_INVALID_ARGUMENT;
        }
    }
    switch (kind) {
    case CNA_GO_EFFECT_STRING_TECHNIQUE_NAME:
        return api.cna_effect_technique_copy_name(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_PASS_NAME:
        return api.cna_effect_pass_copy_name(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_PARAMETER_NAME:
        return api.cna_effect_parameter_copy_name(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_PARAMETER_SEMANTIC:
        return api.cna_effect_parameter_copy_semantic(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_PARAMETER_VALUE:
        return api.cna_effect_parameter_copy_value_string(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_ANNOTATION_NAME:
        return api.cna_effect_annotation_copy_name(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_ANNOTATION_SEMANTIC:
        return api.cna_effect_annotation_copy_semantic(handle, destination, capacity, out_bytes);
    case CNA_GO_EFFECT_STRING_ANNOTATION_VALUE:
        return api.cna_effect_annotation_copy_value_string(handle, destination, capacity, out_bytes);
    default:
        return CNA_GO_RESULT_INVALID_ARGUMENT;
    }
}

CnaGoResult cna_go_effect_parameter_get_info(
    CnaGoHandle parameter, int32_t* out_rows, int32_t* out_columns,
    uint32_t* out_class, uint32_t* out_type) {
    CNA_EffectParameterInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_effect_parameter_get_info(parameter, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_rows = info.row_count;
    *out_columns = info.column_count;
    *out_class = info.parameter_class;
    *out_type = info.parameter_type;
    return result;
}

CnaGoResult cna_go_effect_annotation_get_info(
    CnaGoHandle annotation, int32_t* out_rows, int32_t* out_columns,
    uint32_t* out_class, uint32_t* out_type) {
    CNA_EffectAnnotationInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_effect_annotation_get_info(annotation, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_rows = info.row_count;
    *out_columns = info.column_count;
    *out_class = info.parameter_class;
    *out_type = info.parameter_type;
    return result;
}

CnaGoResult cna_go_effect_parameter_set_value_string(
    CnaGoHandle parameter, const char* value, uint64_t value_length) {
    return api.cna_effect_parameter_set_value_string(parameter, cna_go_view(value, value_length));
}

/* The three annotation vector reads share one trampoline selected by WIDTH, and
   the destination is always four floats: a Vector2 read writes two of them and
   a Vector3 three, so one Go buffer serves all three and no CNA struct crosses
   cgo. */
CnaGoResult cna_go_effect_annotation_get_value_vector(
    CnaGoHandle annotation, uint32_t width, float* out_values) {
    CNA_Vector2 v2;
    CNA_Vector3 v3;
    CNA_Vector4 v4;
    CnaGoResult result;
    switch (width) {
    case 2u:
        result = api.cna_effect_annotation_get_value_vector2(annotation, &v2);
        if (result == CNA_GO_RESULT_SUCCESS) { out_values[0] = v2.x; out_values[1] = v2.y; }
        return result;
    case 3u:
        result = api.cna_effect_annotation_get_value_vector3(annotation, &v3);
        if (result == CNA_GO_RESULT_SUCCESS) { out_values[0] = v3.x; out_values[1] = v3.y; out_values[2] = v3.z; }
        return result;
    case 4u:
        result = api.cna_effect_annotation_get_value_vector4(annotation, &v4);
        if (result == CNA_GO_RESULT_SUCCESS) {
            out_values[0] = v4.x; out_values[1] = v4.y; out_values[2] = v4.z; out_values[3] = v4.w;
        }
        return result;
    default:
        return CNA_GO_RESULT_INVALID_ARGUMENT;
    }
}

CnaGoResult cna_go_effect_annotation_get_value_matrix(CnaGoHandle annotation, float* out_values) {
    CNA_Matrix matrix;
    CnaGoResult result;
    memset(&matrix, 0, sizeof(matrix));
    result = api.cna_effect_annotation_get_value_matrix(annotation, &matrix);
    if (result == CNA_GO_RESULT_SUCCESS) {
        memcpy(out_values, &matrix, sizeof(matrix));
    }
    return result;
}


/* ---------------------------------------------------------------------------
 * Foundation 79 -- the stock-effect routes.
 *
 * BasicEffect's own nineteen, plus the three interface families every stock
 * effect shares: six matrices, eight fog and six lights, and DirectionalLight's
 * ten. Each is generated from the canonical header's own prototype, and a
 * CNA_Vector3 or CNA_Matrix crosses as a flat float array the way the effect
 * annotation readers already do, so the Go side never depends on a C struct
 * layout.
 * ------------------------------------------------------------------------- */

CnaGoResult cna_go_basic_effect_create(CnaGoHandle graphics_device, CnaGoHandle* out_effect) {
    return api.cna_basic_effect_create(graphics_device, out_effect);
}

CnaGoResult cna_go_basic_effect_set_vertex_color_enabled(CnaGoHandle effect, uint8_t value) {
    return api.cna_basic_effect_set_vertex_color_enabled(effect, value);
}

CnaGoResult cna_go_basic_effect_set_prefer_per_pixel_lighting(CnaGoHandle effect, uint8_t value) {
    return api.cna_basic_effect_set_prefer_per_pixel_lighting(effect, value);
}

CnaGoResult cna_go_basic_effect_set_diffuse_color(CnaGoHandle effect, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_basic_effect_set_diffuse_color(effect, value_value);
}

CnaGoResult cna_go_basic_effect_set_emissive_color(CnaGoHandle effect, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_basic_effect_set_emissive_color(effect, value_value);
}

CnaGoResult cna_go_basic_effect_get_specular_color(CnaGoHandle effect, float* out_value) {
    CnaGoResult result;
    CNA_Vector3 value_out_value;
    memset(&value_out_value, 0, sizeof(value_out_value));
    result = api.cna_basic_effect_get_specular_color(effect, &value_out_value);
    if (result == CNA_GO_RESULT_SUCCESS) {
        memcpy(out_value, &value_out_value, sizeof(value_out_value));
    }
    return result;
}

CnaGoResult cna_go_basic_effect_set_specular_color(CnaGoHandle effect, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_basic_effect_set_specular_color(effect, value_value);
}

CnaGoResult cna_go_basic_effect_get_specular_power(CnaGoHandle effect, float* out_value) {
    return api.cna_basic_effect_get_specular_power(effect, out_value);
}

CnaGoResult cna_go_basic_effect_set_specular_power(CnaGoHandle effect, float value) {
    return api.cna_basic_effect_set_specular_power(effect, value);
}

CnaGoResult cna_go_basic_effect_set_alpha(CnaGoHandle effect, float value) {
    return api.cna_basic_effect_set_alpha(effect, value);
}

CnaGoResult cna_go_basic_effect_set_texture_enabled(CnaGoHandle effect, uint8_t value) {
    return api.cna_basic_effect_set_texture_enabled(effect, value);
}

CnaGoResult cna_go_basic_effect_set_texture(CnaGoHandle effect, CnaGoHandle texture) {
    return api.cna_basic_effect_set_texture(effect, texture);
}

CnaGoResult cna_go_effect_matrices_set_world(CnaGoHandle effect, const float* value) {
    CNA_Matrix value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_effect_matrices_set_world(effect, value_value);
}

CnaGoResult cna_go_effect_matrices_set_view(CnaGoHandle effect, const float* value) {
    CNA_Matrix value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_effect_matrices_set_view(effect, value_value);
}

CnaGoResult cna_go_effect_matrices_set_projection(CnaGoHandle effect, const float* value) {
    CNA_Matrix value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_effect_matrices_set_projection(effect, value_value);
}

CnaGoResult cna_go_effect_fog_get_color(CnaGoHandle effect, float* out_value) {
    CnaGoResult result;
    CNA_Vector3 value_out_value;
    memset(&value_out_value, 0, sizeof(value_out_value));
    result = api.cna_effect_fog_get_color(effect, &value_out_value);
    if (result == CNA_GO_RESULT_SUCCESS) {
        memcpy(out_value, &value_out_value, sizeof(value_out_value));
    }
    return result;
}

CnaGoResult cna_go_effect_fog_set_color(CnaGoHandle effect, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_effect_fog_set_color(effect, value_value);
}

CnaGoResult cna_go_effect_fog_set_enabled(CnaGoHandle effect, uint8_t value) {
    return api.cna_effect_fog_set_enabled(effect, value);
}

CnaGoResult cna_go_effect_fog_set_start(CnaGoHandle effect, float value) {
    return api.cna_effect_fog_set_start(effect, value);
}

CnaGoResult cna_go_effect_fog_set_end(CnaGoHandle effect, float value) {
    return api.cna_effect_fog_set_end(effect, value);
}

CnaGoResult cna_go_effect_lights_set_ambient_color(CnaGoHandle effect, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_effect_lights_set_ambient_color(effect, value_value);
}

CnaGoResult cna_go_effect_lights_get_directional_light(CnaGoHandle effect, uint32_t index, CnaGoHandle* out_light) {
    return api.cna_effect_lights_get_directional_light(effect, index, out_light);
}

CnaGoResult cna_go_effect_lights_set_enabled(CnaGoHandle effect, uint8_t value) {
    return api.cna_effect_lights_set_enabled(effect, value);
}

CnaGoResult cna_go_directional_light_destroy(CnaGoHandle light) {
    return api.cna_directional_light_destroy(light);
}

CnaGoResult cna_go_directional_light_set_diffuse_color(CnaGoHandle light, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_directional_light_set_diffuse_color(light, value_value);
}

CnaGoResult cna_go_directional_light_set_direction(CnaGoHandle light, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_directional_light_set_direction(light, value_value);
}

CnaGoResult cna_go_directional_light_set_specular_color(CnaGoHandle light, const float* value) {
    CNA_Vector3 value_value;
    memcpy(&value_value, value, sizeof(value_value));
    return api.cna_directional_light_set_specular_color(light, value_value);
}

CnaGoResult cna_go_directional_light_set_enabled(CnaGoHandle light, uint8_t value) {
    return api.cna_directional_light_set_enabled(light, value);
}

CnaGoResult cna_go_effect_apply(CnaGoHandle effect) {
    return api.cna_effect_apply(effect);
}

CnaGoResult cna_go_effect_destroy(CnaGoHandle effect) {
    return api.cna_effect_destroy(effect);
}

CnaGoResult cna_go_effect_clone(CnaGoHandle effect, CnaGoHandle* out_clone) {
    return api.cna_effect_clone(effect, out_clone);
}

CnaGoResult cna_go_effect_get_parameters(CnaGoHandle effect, CnaGoHandle* out_collection) {
    return api.cna_effect_get_parameters(effect, out_collection);
}

CnaGoResult cna_go_effect_get_techniques(CnaGoHandle effect, CnaGoHandle* out_collection) {
    return api.cna_effect_get_techniques(effect, out_collection);
}

CnaGoResult cna_go_effect_get_current_technique(CnaGoHandle effect, CnaGoHandle* out_technique) {
    return api.cna_effect_get_current_technique(effect, out_technique);
}

CnaGoResult cna_go_effect_set_current_technique(CnaGoHandle effect, CnaGoHandle technique) {
    return api.cna_effect_set_current_technique(effect, technique);
}

CnaGoResult cna_go_effect_technique_collection_get_count(CnaGoHandle collection, uint64_t* out_count) {
    return api.cna_effect_technique_collection_get_count(collection, out_count);
}

CnaGoResult cna_go_effect_technique_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_technique) {
    return api.cna_effect_technique_collection_get_at(collection, index, out_technique);
}

CnaGoResult cna_go_effect_technique_collection_destroy(CnaGoHandle collection) {
    return api.cna_effect_technique_collection_destroy(collection);
}

CnaGoResult cna_go_effect_technique_destroy(CnaGoHandle technique) {
    return api.cna_effect_technique_destroy(technique);
}

CnaGoResult cna_go_effect_technique_get_passes(CnaGoHandle technique, CnaGoHandle* out_collection) {
    return api.cna_effect_technique_get_passes(technique, out_collection);
}

CnaGoResult cna_go_effect_technique_get_annotations(CnaGoHandle technique, CnaGoHandle* out_collection) {
    return api.cna_effect_technique_get_annotations(technique, out_collection);
}

CnaGoResult cna_go_effect_pass_collection_get_count(CnaGoHandle collection, uint64_t* out_count) {
    return api.cna_effect_pass_collection_get_count(collection, out_count);
}

CnaGoResult cna_go_effect_pass_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_pass) {
    return api.cna_effect_pass_collection_get_at(collection, index, out_pass);
}

CnaGoResult cna_go_effect_pass_collection_destroy(CnaGoHandle collection) {
    return api.cna_effect_pass_collection_destroy(collection);
}

CnaGoResult cna_go_effect_pass_destroy(CnaGoHandle pass) {
    return api.cna_effect_pass_destroy(pass);
}

CnaGoResult cna_go_effect_pass_get_annotations(CnaGoHandle pass, CnaGoHandle* out_collection) {
    return api.cna_effect_pass_get_annotations(pass, out_collection);
}

CnaGoResult cna_go_effect_pass_apply(CnaGoHandle pass) {
    return api.cna_effect_pass_apply(pass);
}

CnaGoResult cna_go_effect_parameter_collection_get_count(CnaGoHandle collection, uint64_t* out_count) {
    return api.cna_effect_parameter_collection_get_count(collection, out_count);
}

CnaGoResult cna_go_effect_parameter_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_parameter) {
    return api.cna_effect_parameter_collection_get_at(collection, index, out_parameter);
}

CnaGoResult cna_go_effect_parameter_collection_destroy(CnaGoHandle collection) {
    return api.cna_effect_parameter_collection_destroy(collection);
}

CnaGoResult cna_go_effect_parameter_destroy(CnaGoHandle parameter) {
    return api.cna_effect_parameter_destroy(parameter);
}

CnaGoResult cna_go_effect_parameter_get_elements(CnaGoHandle parameter, CnaGoHandle* out_collection) {
    return api.cna_effect_parameter_get_elements(parameter, out_collection);
}

CnaGoResult cna_go_effect_parameter_get_structure_members(CnaGoHandle parameter, CnaGoHandle* out_collection) {
    return api.cna_effect_parameter_get_structure_members(parameter, out_collection);
}

CnaGoResult cna_go_effect_parameter_get_annotations(CnaGoHandle parameter, CnaGoHandle* out_collection) {
    return api.cna_effect_parameter_get_annotations(parameter, out_collection);
}

CnaGoResult cna_go_effect_parameter_get_value(CnaGoHandle parameter, uint32_t value_type, void* out_value) {
    return api.cna_effect_parameter_get_value(parameter, value_type, out_value);
}

CnaGoResult cna_go_effect_parameter_get_values(CnaGoHandle parameter, uint32_t value_type, uint64_t requested, void* destination, uint64_t capacity, uint64_t* out_count) {
    return api.cna_effect_parameter_get_values(parameter, value_type, requested, destination, capacity, out_count);
}

CnaGoResult cna_go_effect_parameter_set_value(CnaGoHandle parameter, uint32_t value_type, const void* value) {
    return api.cna_effect_parameter_set_value(parameter, value_type, value);
}

CnaGoResult cna_go_effect_parameter_set_values(CnaGoHandle parameter, uint32_t value_type, const void* values, uint64_t count) {
    return api.cna_effect_parameter_set_values(parameter, value_type, values, count);
}


CnaGoResult cna_go_effect_parameter_set_value_texture(CnaGoHandle parameter, uint32_t texture_type, CnaGoHandle texture) {
    return api.cna_effect_parameter_set_value_texture(parameter, texture_type, texture);
}

CnaGoResult cna_go_effect_annotation_collection_get_count(CnaGoHandle collection, uint64_t* out_count) {
    return api.cna_effect_annotation_collection_get_count(collection, out_count);
}

CnaGoResult cna_go_effect_annotation_collection_get_at(CnaGoHandle collection, uint64_t index, CnaGoHandle* out_annotation) {
    return api.cna_effect_annotation_collection_get_at(collection, index, out_annotation);
}

CnaGoResult cna_go_effect_annotation_collection_destroy(CnaGoHandle collection) {
    return api.cna_effect_annotation_collection_destroy(collection);
}

CnaGoResult cna_go_effect_annotation_destroy(CnaGoHandle annotation) {
    return api.cna_effect_annotation_destroy(annotation);
}

CnaGoResult cna_go_effect_annotation_get_value_boolean(CnaGoHandle annotation, uint8_t* out_value) {
    return api.cna_effect_annotation_get_value_boolean(annotation, out_value);
}

CnaGoResult cna_go_effect_annotation_get_value_int32(CnaGoHandle annotation, int32_t* out_value) {
    return api.cna_effect_annotation_get_value_int32(annotation, out_value);
}

CnaGoResult cna_go_effect_annotation_get_value_single(CnaGoHandle annotation, float* out_value) {
    return api.cna_effect_annotation_get_value_single(annotation, out_value);
}

/* Foundation 71 -- the volume and cube texture families. */
CnaGoResult cna_go_texture3d_create(
    CnaGoHandle device, uint32_t width, uint32_t height, uint32_t depth,
    uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture) {
    CNA_Texture3DCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.width = width;
    info.height = height;
    info.depth = depth;
    info.mip_map = (CNA_Bool)(mip_map != 0);
    info.format = format;
    return api.cna_texture3d_create(device, &info, out_texture);
}

CnaGoResult cna_go_texture3d_destroy(CnaGoHandle texture) {
    return api.cna_texture3d_destroy(texture);
}

CnaGoResult cna_go_texture3d_get_info(
    CnaGoHandle texture, uint32_t* out_width, uint32_t* out_height, uint32_t* out_depth,
    uint32_t* out_level_count, uint32_t* out_format) {
    CNA_Texture3DInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_texture3d_get_info(texture, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_width = info.width;
    *out_height = info.height;
    *out_depth = info.depth;
    *out_level_count = info.level_count;
    *out_format = info.format;
    return result;
}

static void cna_go_fill_texture3d_transfer(
    CNA_Texture3DTransfer* transfer, int32_t level,
    int32_t left, int32_t top, int32_t right, int32_t bottom, int32_t front, int32_t back,
    uint64_t start_index, uint64_t element_count) {
    memset(transfer, 0, sizeof(*transfer));
    transfer->struct_size = (uint32_t)sizeof(*transfer);
    transfer->struct_version = 1;
    transfer->level = level;
    transfer->left = left;
    transfer->top = top;
    transfer->right = right;
    transfer->bottom = bottom;
    transfer->front = front;
    transfer->back = back;
    transfer->start_index = start_index;
    transfer->element_count = element_count;
}

CnaGoResult cna_go_texture3d_set_data(
    CnaGoHandle texture, int32_t level,
    int32_t left, int32_t top, int32_t right, int32_t bottom, int32_t front, int32_t back,
    uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity) {
    CNA_Texture3DTransfer transfer;
    cna_go_fill_texture3d_transfer(&transfer, level, left, top, right, bottom, front, back,
        start_index, element_count);
    return api.cna_texture3d_set_data(texture, &transfer, (const CNA_Color*)data, data_capacity);
}

CnaGoResult cna_go_texture3d_get_data(
    CnaGoHandle texture, int32_t level,
    int32_t left, int32_t top, int32_t right, int32_t bottom, int32_t front, int32_t back,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity,
    uint64_t* out_required) {
    CNA_Texture3DTransfer transfer;
    cna_go_fill_texture3d_transfer(&transfer, level, left, top, right, bottom, front, back,
        start_index, element_count);
    return api.cna_texture3d_get_data(texture, &transfer, (CNA_Color*)destination, capacity, out_required);
}

CnaGoResult cna_go_texturecube_create(
    CnaGoHandle device, uint32_t size, uint8_t mip_map, uint32_t format, CnaGoHandle* out_texture) {
    CNA_TextureCubeCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.size = size;
    info.mip_map = (CNA_Bool)(mip_map != 0);
    info.format = format;
    return api.cna_texturecube_create(device, &info, out_texture);
}

CnaGoResult cna_go_texturecube_destroy(CnaGoHandle texture) {
    return api.cna_texturecube_destroy(texture);
}

CnaGoResult cna_go_texturecube_get_info(
    CnaGoHandle texture, uint32_t* out_size, uint32_t* out_level_count, uint32_t* out_format) {
    CNA_TextureCubeInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_texturecube_get_info(texture, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_size = info.size;
    *out_level_count = info.level_count;
    *out_format = info.format;
    return result;
}

static void cna_go_fill_texturecube_transfer(
    CNA_TextureCubeTransfer* transfer, uint32_t face, int32_t level, uint8_t has_rectangle,
    int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count) {
    memset(transfer, 0, sizeof(*transfer));
    transfer->struct_size = (uint32_t)sizeof(*transfer);
    transfer->struct_version = 1;
    transfer->face = face;
    transfer->level = level;
    transfer->has_rectangle = (CNA_Bool)(has_rectangle != 0);
    transfer->rectangle.x = x;
    transfer->rectangle.y = y;
    transfer->rectangle.width = width;
    transfer->rectangle.height = height;
    transfer->start_index = start_index;
    transfer->element_count = element_count;
}

CnaGoResult cna_go_texturecube_set_data(
    CnaGoHandle texture, uint32_t face, int32_t level, uint8_t has_rectangle,
    int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, const void* data, uint64_t data_capacity) {
    CNA_TextureCubeTransfer transfer;
    cna_go_fill_texturecube_transfer(&transfer, face, level, has_rectangle, x, y, width, height,
        start_index, element_count);
    return api.cna_texturecube_set_data(texture, &transfer, (const CNA_Color*)data, data_capacity);
}

CnaGoResult cna_go_texturecube_get_data(
    CnaGoHandle texture, uint32_t face, int32_t level, uint8_t has_rectangle,
    int32_t x, int32_t y, int32_t width, int32_t height,
    uint64_t start_index, uint64_t element_count, void* destination, uint64_t capacity,
    uint64_t* out_required) {
    CNA_TextureCubeTransfer transfer;
    cna_go_fill_texturecube_transfer(&transfer, face, level, has_rectangle, x, y, width, height,
        start_index, element_count);
    return api.cna_texturecube_get_data(texture, &transfer, (CNA_Color*)destination, capacity, out_required);
}

/* Foundation 69 -- the SpriteFont family.
   The loader is the one CNA route that produces two owned handles from one
   asset. Both cross as handles; the ORDER their destruction must take is
   CNA's rule and is enforced by the Go side, which destroys the font first. */
CnaGoResult cna_go_content_manager_load_sprite_font(
    CnaGoHandle content_manager, const char* asset_name, uint64_t asset_name_length,
    CnaGoHandle* out_sprite_font, CnaGoHandle* out_texture) {
    return api.cna_content_manager_load_sprite_font(
        content_manager, cna_go_view(asset_name, asset_name_length), out_sprite_font, out_texture);
}

CnaGoResult cna_go_sprite_font_get_info(
    CnaGoHandle sprite_font,
    uint64_t* out_character_count,
    int32_t* out_line_spacing,
    float* out_spacing,
    uint16_t* out_default_character,
    uint8_t* out_has_default_character) {
    CNA_SpriteFontInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_sprite_font_get_info(sprite_font, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_character_count = info.character_count;
    *out_line_spacing = info.line_spacing;
    *out_spacing = info.spacing;
    *out_default_character = (uint16_t)info.default_character;
    *out_has_default_character = info.has_default_character ? 1u : 0u;
    return result;
}

/* The glyph table crosses as THREE flat arrays rather than one struct array,
   for the reason every other array does: no CNA struct crosses cgo. Element i
   of each describes the same glyph, which is the correspondence the reference's
   four parallel Lists have. */
CnaGoResult cna_go_sprite_font_copy_glyphs(
    CnaGoHandle sprite_font,
    uint64_t capacity,
    uint16_t* out_characters,
    int32_t* out_rectangles,
    float* out_kerning,
    uint64_t* out_count) {
    CNA_SpriteFontGlyph* glyphs;
    CnaGoResult result;
    uint64_t at;
    if (capacity == 0) {
        return api.cna_sprite_font_copy_glyphs(sprite_font, NULL, 0, out_count);
    }
    glyphs = (CNA_SpriteFontGlyph*)calloc((size_t)capacity, sizeof(CNA_SpriteFontGlyph));
    if (glyphs == NULL) {
        return CNA_GO_RESULT_OUT_OF_MEMORY;
    }
    result = api.cna_sprite_font_copy_glyphs(sprite_font, glyphs, capacity, out_count);
    if (result == CNA_GO_RESULT_SUCCESS) {
        for (at = 0; at < *out_count && at < capacity; at++) {
            out_characters[at] = (uint16_t)glyphs[at].character;
            out_rectangles[at * 8 + 0] = glyphs[at].glyph_bounds.x;
            out_rectangles[at * 8 + 1] = glyphs[at].glyph_bounds.y;
            out_rectangles[at * 8 + 2] = glyphs[at].glyph_bounds.width;
            out_rectangles[at * 8 + 3] = glyphs[at].glyph_bounds.height;
            out_rectangles[at * 8 + 4] = glyphs[at].cropping.x;
            out_rectangles[at * 8 + 5] = glyphs[at].cropping.y;
            out_rectangles[at * 8 + 6] = glyphs[at].cropping.width;
            out_rectangles[at * 8 + 7] = glyphs[at].cropping.height;
            out_kerning[at * 3 + 0] = glyphs[at].kerning.x;
            out_kerning[at * 3 + 1] = glyphs[at].kerning.y;
            out_kerning[at * 3 + 2] = glyphs[at].kerning.z;
        }
    }
    free(glyphs);
    return result;
}

CnaGoResult cna_go_sprite_font_set_default_character(
    CnaGoHandle sprite_font, uint8_t has_value, uint16_t value) {
    return api.cna_sprite_font_set_default_character(
        sprite_font, (CNA_Bool)(has_value != 0), (CNA_Char16)value);
}

CnaGoResult cna_go_sprite_font_set_line_spacing(CnaGoHandle sprite_font, int32_t line_spacing) {
    return api.cna_sprite_font_set_line_spacing(sprite_font, line_spacing);
}

CnaGoResult cna_go_sprite_font_set_spacing(CnaGoHandle sprite_font, float spacing) {
    return api.cna_sprite_font_set_spacing(sprite_font, spacing);
}

CnaGoResult cna_go_sprite_font_destroy(CnaGoHandle sprite_font) {
    return api.cna_sprite_font_destroy(sprite_font);
}

/* One string per call rather than an array, because the canonical
   DrawString has no batched form: cna_sprite_batch_draw_string takes one
   command, exactly as the reference's six overloads each produce one
   InternalDraw. */
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
    float layer_depth) {
    CNA_SpriteTextCommand command;
    memset(&command, 0, sizeof(command));
    command.struct_size = (uint32_t)sizeof(command);
    command.struct_version = 1;
    command.sprite_font = sprite_font;
    command.text = cna_go_view(text, text_length);
    command.position.x = position_x;
    command.position.y = position_y;
    command.color.r = red;
    command.color.g = green;
    command.color.b = blue;
    command.color.a = alpha;
    command.rotation = rotation;
    command.origin.x = origin_x;
    command.origin.y = origin_y;
    command.scale.x = scale_x;
    command.scale.y = scale_y;
    command.effects = effects;
    command.layer_depth = layer_depth;
    return api.cna_sprite_batch_draw_string(sprite_batch, &command);
}

CnaGoResult cna_go_graphics_device_manager_set_graphics_profile(CnaGoHandle manager, uint32_t profile) {
    return api.cna_graphics_device_manager_set_graphics_profile(manager, profile);
}
CnaGoResult cna_go_graphics_device_manager_set_is_full_screen(CnaGoHandle manager, uint8_t full_screen) {
    return api.cna_graphics_device_manager_set_is_full_screen(manager, (CNA_Bool)(full_screen != 0));
}
CnaGoResult cna_go_graphics_device_manager_set_prefer_multi_sampling(CnaGoHandle manager, uint8_t prefer) {
    return api.cna_graphics_device_manager_set_prefer_multi_sampling(manager, (CNA_Bool)(prefer != 0));
}
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_format(CnaGoHandle manager, uint32_t format) {
    return api.cna_graphics_device_manager_set_preferred_back_buffer_format(manager, format);
}
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_width(CnaGoHandle manager, int32_t width) {
    return api.cna_graphics_device_manager_set_preferred_back_buffer_width(manager, width);
}
CnaGoResult cna_go_graphics_device_manager_set_preferred_back_buffer_height(CnaGoHandle manager, int32_t height) {
    return api.cna_graphics_device_manager_set_preferred_back_buffer_height(manager, height);
}
CnaGoResult cna_go_graphics_device_manager_set_preferred_depth_stencil_format(CnaGoHandle manager, uint32_t format) {
    return api.cna_graphics_device_manager_set_preferred_depth_stencil_format(manager, format);
}
CnaGoResult cna_go_graphics_device_manager_set_synchronize_with_vertical_retrace(CnaGoHandle manager, uint8_t synchronize) {
    return api.cna_graphics_device_manager_set_synchronize_with_vertical_retrace(manager, (CNA_Bool)(synchronize != 0));
}
CnaGoResult cna_go_graphics_device_manager_set_supported_orientations(CnaGoHandle manager, uint32_t orientations) {
    return api.cna_graphics_device_manager_set_supported_orientations(manager, orientations);
}
CnaGoResult cna_go_graphics_device_manager_apply_changes(CnaGoHandle manager) {
    return api.cna_graphics_device_manager_apply_changes(manager);
}

CnaGoResult cna_go_game_tick(CnaGoHandle game) {
    return api.cna_game_tick(game);
}

CnaGoResult cna_go_game_run_one_frame(CnaGoHandle game) {
    return api.cna_game_run_one_frame(game);
}

CnaGoResult cna_go_game_window_get_allow_user_resizing(CnaGoHandle game, uint8_t* out_allowed) {
    CNA_Bool allowed = 0;
    const CNA_Result result = api.cna_game_window_get_allow_user_resizing(game, &allowed);
    if (result == 0) {
        *out_allowed = (uint8_t)(allowed != 0);
    }
    return result;
}

CnaGoResult cna_go_game_window_set_allow_user_resizing(CnaGoHandle game, uint8_t allowed) {
    return api.cna_game_window_set_allow_user_resizing(game, (CNA_Bool)(allowed != 0));
}

CnaGoResult cna_go_game_window_get_client_bounds(
    CnaGoHandle game,
    int32_t* x,
    int32_t* y,
    int32_t* width,
    int32_t* height) {
    CNA_Rectangle bounds;
    memset(&bounds, 0, sizeof(bounds));
    const CNA_Result result = api.cna_game_window_get_client_bounds(game, &bounds);
    if (result == 0) {
        *x = bounds.x;
        *y = bounds.y;
        *width = bounds.width;
        *height = bounds.height;
    }
    return result;
}

CnaGoResult cna_go_game_window_get_native_handle(CnaGoHandle game, uint64_t* out_handle) {
    return api.cna_game_window_get_native_handle_ext(game, out_handle);
}

CnaGoResult cna_go_game_window_get_screen_device_name_size(CnaGoHandle game, uint64_t* out_bytes) {
    return api.cna_game_window_get_screen_device_name_size(game, out_bytes);
}

CnaGoResult cna_go_game_window_copy_screen_device_name(
    CnaGoHandle game,
    char* destination,
    uint64_t capacity,
    uint64_t* out_bytes) {
    return api.cna_game_window_copy_screen_device_name(game, destination, capacity, out_bytes);
}

CnaGoResult cna_go_game_window_begin_screen_device_change(CnaGoHandle game, uint8_t will_be_full_screen) {
    return api.cna_game_window_begin_screen_device_change(game, (CNA_Bool)(will_be_full_screen != 0));
}

CnaGoResult cna_go_game_window_end_screen_device_change(
    CnaGoHandle game,
    const char* screen_device_name,
    uint64_t screen_device_name_length,
    int32_t client_width,
    int32_t client_height) {
    CNA_StringView name;
    name.data = screen_device_name;
    name.byte_length = screen_device_name_length;
    return api.cna_game_window_end_screen_device_change(game, name, client_width, client_height);
}

CnaGoResult cna_go_game_set_window_title(CnaGoHandle game, const char* title, uint64_t title_length) {
    CNA_StringView view;
    view.data = title;
    view.byte_length = title_length;
    return api.cna_game_set_window_title(game, view);
}

CnaGoResult cna_go_game_window_unsubscribe_events(CnaGoHandle* registrations) {
    if (registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    CNA_Result first = 0;
    for (int i = 0; i < CNA_GO_GAME_WINDOW_EVENT_COUNT; i++) {
        if (registrations[i] == 0) {
            continue;
        }
        const CNA_Result result = api.cna_game_unsubscribe(registrations[i]);
        registrations[i] = 0;
        if (result != 0 && first == 0) {
            first = result;
        }
    }
    return first;
}

CnaGoResult cna_go_game_window_subscribe_events(CnaGoHandle game, uintptr_t context, CnaGoHandle* out_registrations) {
    if (out_registrations == NULL) {
        return 1; /* CNA_RESULT_INVALID_ARGUMENT */
    }
    for (int i = 0; i < CNA_GO_GAME_WINDOW_EVENT_COUNT; i++) {
        out_registrations[i] = 0;
    }
    for (int i = 0; i < CNA_GO_GAME_WINDOW_EVENT_COUNT; i++) {
        CNA_GameEventRegistrationHandle registration = 0;
        const CNA_Result result = api.cna_game_window_subscribe(
            game,
            window_event_identities[i],
            window_event_callbacks[i],
            (void*)context,
            &registration);
        if (result != 0) {
            (void)cna_go_game_window_unsubscribe_events(out_registrations);
            return result;
        }
        out_registrations[i] = registration;
    }
    return 0;
}

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

// The vertex-declaration and vertex-buffer routes.
//
// Elements cross as a FLAT int32 array of four fields each, not as a
// CNA_VertexElement array: the settled boundary rule is that no CNA struct
// crosses cgo, so the Go side writes four int32s per element and this
// translation unit is where they become the C structure.
CnaGoResult cna_go_vertex_declaration_create(
    int32_t vertex_stride,
    uint8_t has_stride,
    const int32_t* elements,
    uint64_t element_count,
    CnaGoHandle* out_declaration) {
    CNA_VertexElement* converted;
    CnaGoResult result;
    uint64_t at;
    if (element_count == 0) {
        return CNA_GO_RESULT_INVALID_ARGUMENT;
    }
    converted = (CNA_VertexElement*)calloc((size_t)element_count, sizeof(CNA_VertexElement));
    if (converted == NULL) {
        return CNA_GO_RESULT_OUT_OF_MEMORY;
    }
    for (at = 0; at < element_count; at++) {
        converted[at].offset = elements[at * 4 + 0];
        converted[at].format = (CNA_VertexElementFormat)elements[at * 4 + 1];
        converted[at].usage = (CNA_VertexElementUsage)elements[at * 4 + 2];
        converted[at].usage_index = elements[at * 4 + 3];
    }
    if (has_stride) {
        result = api.cna_vertex_declaration_create_with_stride(
            vertex_stride, converted, element_count, out_declaration);
    } else {
        result = api.cna_vertex_declaration_create(converted, element_count, out_declaration);
    }
    free(converted);
    return result;
}

CnaGoResult cna_go_vertex_declaration_destroy(CnaGoHandle declaration) {
    return api.cna_vertex_declaration_destroy(declaration);
}

CnaGoResult cna_go_vertex_declaration_get_stride(CnaGoHandle declaration, int32_t* out_stride) {
    return api.cna_vertex_declaration_get_stride(declaration, out_stride);
}

CnaGoResult cna_go_vertex_buffer_create(
    CnaGoHandle device,
    CnaGoHandle declaration,
    int32_t vertex_count,
    uint32_t buffer_usage,
    uint8_t dynamic,
    CnaGoHandle* out_vertex_buffer) {
    CNA_VertexBufferCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    info.vertex_declaration = declaration;
    info.vertex_count = vertex_count;
    info.buffer_usage = buffer_usage;
    info.dynamic = (CNA_Bool)(dynamic != 0);
    return api.cna_vertex_buffer_create(device, &info, out_vertex_buffer);
}

CnaGoResult cna_go_vertex_buffer_destroy(CnaGoHandle vertex_buffer) {
    return api.cna_vertex_buffer_destroy(vertex_buffer);
}

CnaGoResult cna_go_vertex_buffer_get_info(
    CnaGoHandle vertex_buffer,
    int32_t* out_vertex_count,
    uint32_t* out_buffer_usage,
    uint8_t* out_dynamic,
    uint8_t* out_is_content_lost,
    uint8_t* out_has_renderer,
    int32_t* out_vertex_stride,
    uint64_t* out_vertex_element_count) {
    CNA_VertexBufferInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_vertex_buffer_get_info(vertex_buffer, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_vertex_count = info.vertex_count;
    *out_buffer_usage = info.buffer_usage;
    *out_dynamic = info.dynamic ? 1u : 0u;
    *out_is_content_lost = info.is_content_lost ? 1u : 0u;
    *out_has_renderer = info.has_renderer ? 1u : 0u;
    *out_vertex_stride = info.vertex_stride;
    *out_vertex_element_count = info.vertex_element_count;
    return result;
}

CnaGoResult cna_go_vertex_buffer_set_data_raw_at(
    CnaGoHandle vertex_buffer,
    uint64_t buffer_offset_in_bytes,
    const void* data,
    uint64_t data_byte_count,
    uint64_t vertex_count,
    uint32_t vertex_stride) {
    return api.cna_vertex_buffer_set_data_raw_at(
        vertex_buffer, buffer_offset_in_bytes, data, data_byte_count, vertex_count, vertex_stride);
}

CnaGoResult cna_go_vertex_buffer_get_data_raw(
    CnaGoHandle vertex_buffer,
    uint64_t buffer_offset_in_bytes,
    void* destination,
    uint64_t destination_byte_count,
    uint64_t vertex_count,
    uint32_t vertex_stride) {
    return api.cna_vertex_buffer_get_data_raw(
        vertex_buffer, buffer_offset_in_bytes, destination, destination_byte_count, vertex_count, vertex_stride);
}

// Bindings cross as a FLAT int64 triple per binding -- handle, offset,
// frequency -- for the same reason vertex elements cross flat: no CNA struct
// crosses cgo, and this translation unit is where the triples become
// CNA_VertexBufferBinding values.
CnaGoResult cna_go_graphics_device_set_vertex_buffers(
    CnaGoHandle device,
    const int64_t* bindings,
    uint64_t binding_count) {
    CNA_VertexBufferBinding* converted;
    CnaGoResult result;
    uint64_t at;
    if (binding_count == 0) {
        return api.cna_graphics_device_set_vertex_buffers(device, NULL, 0);
    }
    converted = (CNA_VertexBufferBinding*)calloc((size_t)binding_count, sizeof(CNA_VertexBufferBinding));
    if (converted == NULL) {
        return CNA_GO_RESULT_OUT_OF_MEMORY;
    }
    for (at = 0; at < binding_count; at++) {
        converted[at].vertex_buffer = (CNA_VertexBufferHandle)bindings[at * 3 + 0];
        converted[at].vertex_offset = (int32_t)bindings[at * 3 + 1];
        converted[at].instance_frequency = (int32_t)bindings[at * 3 + 2];
    }
    result = api.cna_graphics_device_set_vertex_buffers(device, converted, binding_count);
    free(converted);
    return result;
}

CnaGoResult cna_go_graphics_device_set_index_buffer(CnaGoHandle device, CnaGoHandle index_buffer) {
    return api.cna_graphics_device_set_index_buffer(device, index_buffer);
}

CnaGoResult cna_go_graphics_device_draw_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t vertex_start, int32_t primitive_count) {
    return api.cna_graphics_device_draw_primitives(device, primitive_type, vertex_start, primitive_count);
}

CnaGoResult cna_go_graphics_device_draw_indexed_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t base_vertex, int32_t min_vertex_index,
    int32_t num_vertices, int32_t start_index, int32_t primitive_count) {
    return api.cna_graphics_device_draw_indexed_primitives(
        device, primitive_type, base_vertex, min_vertex_index, num_vertices, start_index, primitive_count);
}

CnaGoResult cna_go_graphics_device_draw_instanced_primitives(
    CnaGoHandle device, uint32_t primitive_type, int32_t base_vertex, int32_t min_vertex_index,
    int32_t num_vertices, int32_t start_index, int32_t primitive_count, int32_t instance_count) {
    return api.cna_graphics_device_draw_instanced_primitives(
        device, primitive_type, base_vertex, min_vertex_index, num_vertices, start_index,
        primitive_count, instance_count);
}

// The twelve adapter routes, flattened. Every one takes a callback-scoped
// device handle, which is CNA's own requirement and is why GraphicsAdapter's
// two STATIC members are reachable only inside a lifecycle callback.
CnaGoResult cna_go_graphics_adapter_get_count(CnaGoHandle device, uint64_t* out_count) {
    return api.cna_graphics_adapter_get_count(device, out_count);
}

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
    uint64_t* out_device_name_bytes) {
    CNA_GraphicsAdapterInfo info;
    CnaGoResult result;
    memset(&info, 0, sizeof(info));
    info.struct_size = (uint32_t)sizeof(info);
    info.struct_version = 1;
    result = api.cna_graphics_adapter_get_info(device, adapter_index, &info);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_index = info.adapter_index;
    *out_is_default = info.is_default_adapter ? 1u : 0u;
    *out_is_wide_screen = info.is_wide_screen ? 1u : 0u;
    *out_use_null_device = info.use_null_device ? 1u : 0u;
    *out_use_reference_device = info.use_reference_device ? 1u : 0u;
    *out_vendor_id = info.vendor_id;
    *out_device_id = info.device_id;
    *out_revision = info.revision;
    *out_subsystem_id = info.subsystem_id;
    *out_description_bytes = info.description_byte_length;
    *out_device_name_bytes = info.device_name_byte_length;
    return result;
}

CnaGoResult cna_go_graphics_adapter_copy_description(
    CnaGoHandle device, uint32_t adapter_index, char* destination, uint64_t capacity, uint64_t* out_bytes) {
    return api.cna_graphics_adapter_copy_description(device, adapter_index, destination, capacity, out_bytes);
}

CnaGoResult cna_go_graphics_adapter_copy_device_name(
    CnaGoHandle device, uint32_t adapter_index, char* destination, uint64_t capacity, uint64_t* out_bytes) {
    return api.cna_graphics_adapter_copy_device_name(device, adapter_index, destination, capacity, out_bytes);
}

CnaGoResult cna_go_graphics_adapter_get_current_display_mode(
    CnaGoHandle device, uint32_t adapter_index, int32_t* out_width, int32_t* out_height, uint32_t* out_format) {
    CNA_DisplayMode mode;
    CnaGoResult result;
    memset(&mode, 0, sizeof(mode));
    mode.struct_size = (uint32_t)sizeof(mode);
    mode.struct_version = 1;
    result = api.cna_graphics_adapter_get_current_display_mode(device, adapter_index, &mode);
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_width = mode.width;
    *out_height = mode.height;
    *out_format = mode.format;
    return result;
}

CnaGoResult cna_go_graphics_adapter_get_display_mode_count(
    CnaGoHandle device, uint32_t adapter_index, uint64_t* out_count) {
    // The filter is never applied here: DisplayModeCollection's indexer does
    // the filtering in managed code, exactly as the reference's does, so
    // asking CNA to filter as well would be a second answer that could differ.
    return api.cna_graphics_adapter_get_display_mode_count(device, adapter_index, 0, 0, out_count);
}

// Display modes cross as a FLAT int32 triple each -- width, height, format --
// for the same reason every other array does: no CNA struct crosses cgo.
CnaGoResult cna_go_graphics_adapter_copy_display_modes(
    CnaGoHandle device, uint32_t adapter_index, int32_t* out_modes, uint64_t capacity, uint64_t* out_count) {
    CNA_DisplayMode* modes;
    CnaGoResult result;
    uint64_t at;
    if (capacity == 0) {
        return api.cna_graphics_adapter_copy_display_modes(device, adapter_index, 0, 0, NULL, 0, out_count);
    }
    modes = (CNA_DisplayMode*)calloc((size_t)capacity, sizeof(CNA_DisplayMode));
    if (modes == NULL) {
        return CNA_GO_RESULT_OUT_OF_MEMORY;
    }
    for (at = 0; at < capacity; at++) {
        modes[at].struct_size = (uint32_t)sizeof(CNA_DisplayMode);
        modes[at].struct_version = 1;
    }
    result = api.cna_graphics_adapter_copy_display_modes(device, adapter_index, 0, 0, modes, capacity, out_count);
    if (result == CNA_GO_RESULT_SUCCESS) {
        for (at = 0; at < *out_count && at < capacity; at++) {
            out_modes[at * 3 + 0] = modes[at].width;
            out_modes[at * 3 + 1] = modes[at].height;
            out_modes[at * 3 + 2] = (int32_t)modes[at].format;
        }
    }
    free(modes);
    return result;
}

CnaGoResult cna_go_graphics_adapter_set_device_preferences(
    CnaGoHandle device, uint32_t adapter_index, uint8_t use_null_device, uint8_t use_reference_device) {
    return api.cna_graphics_adapter_set_device_preferences(
        device, adapter_index, (CNA_Bool)(use_null_device != 0), (CNA_Bool)(use_reference_device != 0));
}

CnaGoResult cna_go_graphics_adapter_is_profile_supported(
    CnaGoHandle device, uint32_t adapter_index, uint32_t profile, uint8_t* out_supported) {
    CNA_Bool supported = 0;
    CnaGoResult result = api.cna_graphics_adapter_is_profile_supported(
        device, adapter_index, profile, &supported);
    *out_supported = supported ? 1u : 0u;
    return result;
}

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
    int32_t* out_multi_sample_count) {
    CNA_GraphicsFormatSelection selection;
    CnaGoResult result;
    memset(&selection, 0, sizeof(selection));
    selection.struct_size = (uint32_t)sizeof(selection);
    selection.struct_version = 1;
    if (render_target) {
        result = api.cna_graphics_adapter_query_render_target_format(
            device, adapter_index, profile, format, depth_format, multi_sample_count, &selection);
    } else {
        result = api.cna_graphics_adapter_query_backbuffer_format(
            device, adapter_index, profile, format, depth_format, multi_sample_count, &selection);
    }
    if (result != CNA_GO_RESULT_SUCCESS) {
        return result;
    }
    *out_exact_match = selection.exact_match ? 1u : 0u;
    *out_format = selection.format;
    *out_depth_format = selection.depth_format;
    *out_multi_sample_count = selection.multi_sample_count;
    return result;
}

CnaGoResult cna_go_graphics_adapter_get_native_monitor_handle(
    CnaGoHandle device, uint32_t adapter_index, uint64_t* out_value) {
    return api.cna_graphics_adapter_get_native_monitor_handle(device, adapter_index, out_value);
}

CnaGoResult cna_go_graphics_device_get_adapter_index(CnaGoHandle device, uint32_t* out_index) {
    return api.cna_graphics_device_get_adapter_index(device, out_index);
}
