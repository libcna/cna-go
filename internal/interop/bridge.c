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
CnaGoResult cna_go_sprite_batch_end(CnaGoHandle batch) { return api.cna_sprite_batch_end(batch); }
CnaGoResult cna_go_sprite_batch_destroy(CnaGoHandle batch) { return api.cna_sprite_batch_destroy(batch); }

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
