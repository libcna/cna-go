package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The native game-event bridge is the first CNA-Go surface whose correctness
// depends on values the compiler cannot infer from a Go signature: an event
// identity, a registration-handle width, a callback shape and the position of
// the caller context in a five-parameter prototype. Each of those is pinned by
// a _Static_assert or an incompatible-pointer-types assignment, and each of the
// mutations below removes exactly one pin and requires the compile to fail.
//
// The mutations act on real source. Two translation units carry the pins:
//
//   - internal/interop/bridge.c compiled with CNA-Go's private manifest and no
//     canonical CNA header, which is exactly how the cgo build sees it;
//   - tools/native_abi/testdata/probe.c compiled WITH the canonical header,
//     which is the only place the private manifest and the canonical
//     declarations are compiled together.
//
// A mutation that still compiles is a hole in the ABI evidence, not a passing
// test.

type sourceMutation struct {
	name string
	file string
	old  string
	new  string
}

var bridgeMutations = []sourceMutation{
	// Foundation 50. Renaming CNA_SpriteCommand's destination member to the
	// position-and-scale pair the OTHER command carries is the same 16 bytes at
	// the same offset with the same alignment, so no measurement moves. What
	// breaks is the trampoline that writes it: bridge.c assigns
	// command.destination.x, and a member that is not there is not a silent
	// conversion. This is the class of defect only the bridge TU can catch.
	{
		name: "sprite-destination-typed-as-a-position-and-scale",
		file: "abi_manifest.h",
		old:  "    CNA_Rectangle destination;\n    CNA_Rectangle source;\n    CNA_Color color;\n    float rotation;\n    CNA_Vector2 origin;\n    uint32_t effects;\n    float layer_depth;\n} CNA_SpriteCommand;",
		new:  "    CNA_Vector2 destination_position;\n    CNA_Vector2 destination_scale;\n    CNA_Rectangle source;\n    CNA_Color color;\n    float rotation;\n    CNA_Vector2 origin;\n    uint32_t effects;\n    float layer_depth;\n} CNA_SpriteCommand;",
	},
	{
		name: "wrong-event-constant",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED UINT32_C(0)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_ACTIVATED UINT32_C(7)",
	},
	{
		name: "swapped-event-constants",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_DISPOSED UINT32_C(2)\n#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(3)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_DISPOSED UINT32_C(3)\n#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(2)",
	},
	{
		name: "registration-handle-narrowed",
		file: "abi_manifest.h",
		old:  "typedef CNA_Handle CNA_GameEventRegistrationHandle;",
		new:  "typedef uint32_t CNA_GameEventRegistrationHandle;",
	},
	{
		name: "event-identity-count-drift",
		file: "abi_manifest.h",
		old:  "#define CNA_GAME_EVENT_MAXIMUM CNA_GAME_EVENT_EXITING",
		new:  "#define CNA_GAME_EVENT_MAXIMUM UINT32_C(9)",
	},
	{
		name: "missing-unsubscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_unsubscribe) \\\n",
		new:  "",
	},
	{
		name: "missing-subscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_subscribe) \\\n",
		new:  "",
	},
	{
		name: "callback-returns-a-result",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef CNA_Result (*CNA_GameEventCallback)(void*);",
	},
	{
		name: "callback-drops-user-data",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef void (*CNA_GameEventCallback)(void);",
	},
	{
		name: "callback-takes-a-game-handle",
		file: "abi_manifest.h",
		old:  "typedef void (*CNA_GameEventCallback)(void*);",
		new:  "typedef void (*CNA_GameEventCallback)(CNA_Handle, void*);",
	},
	{
		name: "unsubscribe-takes-the-game",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);",
		new:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_Handle, CNA_GameEventRegistrationHandle);",
	},
	{
		name: "bridge-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_EVENT_DISPOSED = 2,",
		new:  "    CNA_GO_GAME_EVENT_DISPOSED = 3,",
	},
	{
		name: "bridge-mirror-count-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_EVENT_COUNT = 4",
		new:  "    CNA_GO_GAME_EVENT_COUNT = 5",
	},
	// The four optional frame-hook members. CNA-Go assigns each one behind a
	// declared capability, so their order and their types are both load-bearing
	// in a way they were not while only `initialize` was installed.
	{
		name: "frame-hook-member-order-drift",
		file: "abi_manifest.h",
		old:  "    CNA_GameLifecycleCallback begin_run;\n    CNA_GameLifecycleCallback end_run;\n    CNA_GameBeginDrawCallback begin_draw;",
		new:  "    CNA_GameBeginDrawCallback begin_draw;\n    CNA_GameLifecycleCallback begin_run;\n    CNA_GameLifecycleCallback end_run;",
	},
	{
		name: "begin-draw-member-narrowed-to-the-lifecycle-shape",
		file: "abi_manifest.h",
		old:  "    CNA_GameBeginDrawCallback begin_draw;",
		new:  "    CNA_GameLifecycleCallback begin_draw;",
	},
	{
		name: "begin-draw-callback-drops-the-out-parameter",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*CNA_GameBeginDrawCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_Bool*, CNA_CallbackError*);",
		new:  "typedef CNA_Result (*CNA_GameBeginDrawCallback)(CNA_Handle, const CNA_GameTime*, void*, CNA_CallbackError*);",
	},
	// The Foundation 42 timing setters, from the side the bridge translation
	// unit can see: a prototype that loses the game handle, gains a parameter,
	// or disappears from the required set.
	{
		name: "inactive-sleep-time-takes-no-game",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_inactive_sleep_time_ticks_fn)(CNA_Handle, int64_t);",
		new:  "typedef CNA_Result (*cna_game_set_inactive_sleep_time_ticks_fn)(int64_t);",
	},
	{
		name: "suppress-draw-takes-a-frame-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_suppress_draw_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_game_suppress_draw_fn)(CNA_Handle, uint32_t);",
	},
	{
		name: "missing-suppress-draw-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_suppress_draw) \\\n",
		new:  "",
	},
	{
		name: "missing-reset-elapsed-time-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_reset_elapsed_time) \\\n",
		new:  "",
	},
	{
		name: "frame-hook-mask-bit-collision",
		file: "bridge.h",
		old:  "    CNA_GO_FRAME_HOOK_BEGIN_DRAW = 1u << 2,",
		new:  "    CNA_GO_FRAME_HOOK_BEGIN_DRAW = 1u << 1,",
	},
	// Foundation 45. The window family is a SECOND identity space that also
	// starts at zero, so every one of these mutations produces a value that
	// looks entirely legitimate in the other family's table.
	{
		name: "window-event-identity-drift",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED UINT32_C(1)",
		new:  "#define CNA_GO_MANIFEST_GAME_WINDOW_EVENT_ORIENTATION_CHANGED UINT32_C(2)",
	},
	{
		name: "bridge-window-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED = 2,",
		new:  "    CNA_GO_GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_CHANGED = 1,",
	},
	{
		name: "window-event-count-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GAME_WINDOW_EVENT_COUNT = 3",
		new:  "    CNA_GO_GAME_WINDOW_EVENT_COUNT = 4",
	},
	{
		name: "missing-window-subscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_window_subscribe)",
		new:  "",
	},
	// Foundation 47's two frame steps. Both are CNA_Result(CNA_Handle), which
	// is the shape cna_game_run, cna_game_request_exit and cna_game_destroy
	// also have -- so a manifest that bound one where another belongs compiles
	// cleanly, and only the loader's dladdr identity check separates them.
	{
		name: "missing-tick-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_tick) \\\n",
		new:  "",
	},
	{
		name: "missing-run-one-frame-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_run_one_frame) \\\n",
		new:  "",
	},
	{
		name: "tick-takes-a-frame-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_tick_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_game_tick_fn)(CNA_Handle, uint32_t);",
	},
	// Foundation 48's GraphicsDeviceManager setters. Nine of the ten share the
	// shape CNA_Result(CNA_Handle, <one value>), so a manifest that bound one
	// where another belongs compiles whenever the value types match -- which
	// is why the width and height routes, both CNA_Result(CNA_Handle, int32_t),
	// are separated by the loader's dladdr identity check rather than by the
	// compiler.
	// Foundation 49's manager signal family. It is the profile's THIRD
	// numbering and the only one whose device events do not start at zero, so
	// a table indexed as if the three agreed is off by one -- and every value
	// involved is a plausible member of the other two families.
	{
		name: "manager-event-identity-drift",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESET UINT32_C(3)",
		new:  "#define CNA_GO_MANIFEST_GDM_EVENT_DEVICE_RESET UINT32_C(4)",
	},
	{
		name: "bridge-manager-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GDM_EVENT_DEVICE_CREATED = 1,",
		new:  "    CNA_GO_GDM_EVENT_DEVICE_CREATED = 0,",
	},
	{
		name: "manager-event-count-drift",
		file: "bridge.h",
		old:  "    CNA_GO_GDM_EVENT_COUNT = 5",
		new:  "    CNA_GO_GDM_EVENT_COUNT = 4",
	},
	{
		name: "missing-manager-subscribe-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_subscribe) \\\n",
		new:  "",
	},
	{
		name: "missing-manager-begin-draw-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_begin_draw) \\\n",
		new:  "",
	},
	{
		name: "missing-manager-apply-changes-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_apply_changes) \\\n",
		new:  "",
	},
	{
		name: "missing-manager-full-screen-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_graphics_device_manager_set_is_full_screen) \\\n",
		new:  "",
	},
	{
		name: "missing-window-title-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_game_set_window_title) \\\n",
		new:  "",
	},
	// The admission policy itself. The qualified encoded constant is what the
	// loader's rejection message reports, and the floor is what it enforces; a
	// policy that reported one range and enforced another would admit or refuse
	// libraries for reasons no diagnostic could explain.
	{
		name: "qualified-abi-constant-disagrees-with-the-floor",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_QUALIFIED_VERSION = 0x00001500u,",
		new:  "    CNA_GO_ABI_QUALIFIED_VERSION = 0x00001400u,",
	},
	{
		name: "abi-floor-raised-without-re-encoding",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MINIMUM_MINOR = 21,",
		new:  "    CNA_GO_ABI_MINIMUM_MINOR = 22,",
	},
	{
		name: "abi-decoding-mirror-drift",
		file: "bridge.h",
		old:  "#define CNA_GO_ABI_MINOR_OF(version) (((uint32_t)(version) >> 8) & UINT32_C(0xFF))",
		new:  "#define CNA_GO_ABI_MINOR_OF(version) (((uint32_t)(version) >> 4) & UINT32_C(0xFF))",
	},
}

// A few pins cannot live in the bridge translation unit at all, and the reason
// is worth stating rather than working around. GNU C converts silently between
// void* and a function pointer, so swapping the callback and the caller context
// in cna_game_subscribe's prototype still compiles against a manifest that
// declares the same swap -- the manifest would simply be self-consistently
// wrong. The pin has to compare the manifest with the canonical declaration,
// which only the probe translation unit does.
var probeMutations = []sourceMutation{
	// Foundation 50 binds the SECOND sprite family, and the two commands are
	// the reason a mutation here matters more than usual: CNA_SpriteCommand and
	// CNA_SpriteScaledCommand carry the same handle, the same source rectangle,
	// the same colour, the same rotation, the same origin, the same effects and
	// the same depth, and differ only in ONE member -- a destination
	// CNA_Rectangle of four int32 against a position CNA_Vector2 plus a scale
	// CNA_Vector2 of four float32. Both are 16 bytes. A manifest that gave one
	// the other's member would be the SAME SIZE, would compile through every
	// static check, and would submit int32 pixels as float32 coordinates.
	//
	// The route pairing is the other half: the two prototypes differ only in
	// which command pointer they take, so binding submit_many where
	// submit_scaled_many belongs is a two-token edit that also compiles.
	// Foundation 54. The typed transfer routes take the transfer BY POINTER and
	// the payload as an untyped one, so C checks almost nothing about them.
	{
		name: "texture-transfer-passed-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_texture2d_set_data_fn)(CNA_Handle, CNA_TextureDataType, const CNA_Texture2DTransfer*, const void*, uint64_t);",
		new:  "typedef CNA_Result (*cna_texture2d_set_data_fn)(CNA_Handle, CNA_TextureDataType, CNA_Texture2DTransfer, const void*, uint64_t);",
	},
	{
		// get_data's destination is written and its out-count is a pointer.
		// Dropping the out-count is a shorter prototype the bridge would still
		// compile if it did not pass one -- and it does, so this is caught on
		// both sides. It is listed because the DEFECT it plants is real: a
		// transfer that cannot report how many elements it needed makes an
		// undersized destination unreportable.
		name: "get-data-drops-the-required-element-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_texture2d_get_data_fn)(CNA_Handle, CNA_TextureDataType, const CNA_Texture2DTransfer*, void*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_texture2d_get_data_fn)(CNA_Handle, CNA_TextureDataType, const CNA_Texture2DTransfer*, void*, uint64_t);",
	},
	// Foundation 53. The encode route's last two parameters are a capacity and
	// an out count -- a uint64 and a pointer to one -- and the bridge passes
	// both through, so nothing in C objects to their order until a prototype is
	// compared with the canonical one.
	{
		name: "copy-encoded-capacity-and-out-count-swapped",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_texture2d_copy_encoded_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, uint8_t*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_texture2d_copy_encoded_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, uint8_t*, uint64_t*, uint64_t);",
	},
	{
		// The destination is WRITTEN by the callee. Declaring it const compiles
		// on the bridge side, which only passes the pointer along.
		name: "copy-encoded-destination-declared-const",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_texture2d_copy_encoded_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, uint8_t*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_texture2d_copy_encoded_fn)(CNA_Handle, CNA_TextureImageFormat, uint32_t, uint32_t, const uint8_t*, uint64_t, uint64_t*);",
	},
	// Foundation 52 binds two routes whose structures are versioned and whose
	// members are same-width neighbours, which is the shape that hides a defect
	// best.
	{
		// The create info is passed BY POINTER. Taking it by value compiles in
		// the bridge -- the bridge builds the struct itself -- and passes the
		// struct where the callee expects an address.
		name: "texture-create-info-passed-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_texture2d_create_fn)(CNA_Handle, const CNA_Texture2DCreateInfo*, CNA_Handle*);",
		new:  "typedef CNA_Result (*cna_texture2d_create_fn)(CNA_Handle, CNA_Texture2DCreateInfo, CNA_Handle*);",
	},
	{
		// The display mode is filled BY the callee, so a by-value out parameter
		// would silently discard every field it wrote.
		name: "display-mode-out-parameter-passed-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_get_display_mode_fn)(CNA_Handle, CNA_DisplayMode*);",
		new:  "typedef CNA_Result (*cna_graphics_device_get_display_mode_fn)(CNA_Handle, CNA_DisplayMode);",
	},
	// Foundation 51 binds GraphicsDevice's render state, and three of its four
	// controls are about a C conversion that says nothing.
	{
		// clear_options is (handle, options, color, float depth, int32 stencil),
		// and the last two are a float and an int of the same width. Swapping
		// them in the manifest lets the bridge pass a depth where a stencil
		// belongs, and C converts BOTH ways without a word: a depth of 1.0f
		// arrives as the integer 1, and a stencil of 0 arrives as 0.0f, so a
		// clear still happens and clears the wrong thing.
		name: "clear-options-depth-and-stencil-swapped",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_clear_options_fn)(CNA_Handle, CNA_ClearOptions, CNA_Color, float, int32_t);",
		new:  "typedef CNA_Result (*cna_graphics_device_clear_options_fn)(CNA_Handle, CNA_ClearOptions, CNA_Color, int32_t, float);",
	},
	{
		// A colour passed by pointer where CNA takes it by value. Four bytes
		// against eight, and the callee would read the pointer's own bits as a
		// colour.
		name: "blend-factor-passed-by-pointer",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_set_blend_factor_fn)(CNA_Handle, CNA_Color);",
		new:  "typedef CNA_Result (*cna_graphics_device_set_blend_factor_fn)(CNA_Handle, const CNA_Color*);",
	},
	{
		// The viewport is the largest by-value struct CNA-Go passes: 24 bytes,
		// six members, and the calling convention splits it across registers.
		// Passing it by pointer compiles in the bridge only because the bridge
		// builds the struct itself.
		name: "viewport-passed-by-pointer",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_set_viewport_fn)(CNA_Handle, CNA_Viewport);",
		new:  "typedef CNA_Result (*cna_graphics_device_set_viewport_fn)(CNA_Handle, const CNA_Viewport*);",
	},
	{
		name: "sprite-submit-many-takes-the-scaled-command",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_sprite_batch_submit_many_fn)(CNA_Handle, const CNA_SpriteCommand*, uint64_t);",
		new:  "typedef CNA_Result (*cna_sprite_batch_submit_many_fn)(CNA_Handle, const CNA_SpriteScaledCommand*, uint64_t);",
	},
	{
		name: "sprite-scaled-submit-takes-the-destination-command",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_sprite_batch_submit_scaled_many_fn)(CNA_Handle, const CNA_SpriteScaledCommand*, uint64_t);",
		new:  "typedef CNA_Result (*cna_sprite_batch_submit_scaled_many_fn)(CNA_Handle, const CNA_SpriteCommand*, uint64_t);",
	},
	{
		name: "subscribe-user-data-before-callback",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, void*, CNA_GameEventCallback, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "subscribe-drops-the-out-registration",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*);",
	},
	{
		name: "subscribe-returns-the-registration",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_GameEventRegistrationHandle (*cna_game_subscribe_fn)(CNA_Handle, CNA_GameEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "unsubscribe-takes-a-narrowed-handle",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(CNA_GameEventRegistrationHandle);",
		new:  "typedef CNA_Result (*cna_game_unsubscribe_fn)(uint32_t);",
	},
	{
		name: "manifest-disagrees-with-canonical-header",
		file: "abi_manifest.h",
		old:  "#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(3)",
		new:  "#define CNA_GO_MANIFEST_GAME_EVENT_EXITING UINT32_C(2)",
	},
	// Foundation 63. The content routes. Creation takes the create-info BY
	// POINTER, and a by-value prototype is the defect the bridge catches:
	// bridge.c passes `&info`, and a struct parameter does not accept an
	// address.
	{
		name: "content-create-takes-the-info-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_content_manager_create_fn)(CNA_Handle, const CNA_ContentManagerCreateInfo*, CNA_Handle*);",
		new:  "typedef CNA_Result (*cna_content_manager_create_fn)(CNA_Handle, CNA_ContentManagerCreateInfo, CNA_Handle*);",
	},
	// The load's asset name is a CNA_StringView, not a bare pointer. A
	// prototype that took `const char*` would compile at the CALL only if the
	// bridge stopped building a view -- so this catches the shape that would
	// hand CNA a name with no length and let it read to the first NUL in
	// whatever follows.
	{
		name: "content-load-takes-a-bare-asset-pointer",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_content_manager_load_texture2d_fn)(CNA_Handle, CNA_StringView, CNA_Handle*);",
		new:  "typedef CNA_Result (*cna_content_manager_load_texture2d_fn)(CNA_Handle, const char*, CNA_Handle*);",
	},
	// The asset-path copy drops its capacity. The remaining arguments still
	// line up by type -- handle, view, pointer, pointer -- so a caller that
	// passed capacity where the count belongs would be writing the resolved
	// path into the caller's byte count.
	// Foundation 65. The index-buffer routes. Creation takes the create-info BY
	// POINTER; a by-value prototype is refused at the call, which passes `&info`.
	// Foundation 66. The declaration route takes an ELEMENT ARRAY and a count.
	// Dropping the count leaves a prototype whose remaining arguments still
	// line up by type, so only the bridge's own three-argument call catches it.
	{
		name: "vertex-declaration-create-drops-its-element-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_vertex_declaration_create_fn)(const CNA_VertexElement*, uint64_t, CNA_VertexDeclarationHandle*);",
		new:  "typedef CNA_Result (*cna_vertex_declaration_create_fn)(const CNA_VertexElement*, CNA_VertexDeclarationHandle*);",
	},
	// The raw transfers carry a BUFFER offset, a byte count, a vertex count and
	// a stride. Dropping the stride is the one that matters: without it CNA
	// could not know how far apart two vertices are, and the remaining
	// arguments still line up.
	{
		name: "vertex-buffer-set-data-raw-drops-its-stride",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_vertex_buffer_set_data_raw_at_fn)(CNA_VertexBufferHandle, uint64_t, const void*, uint64_t, uint64_t, uint32_t);",
		new:  "typedef CNA_Result (*cna_vertex_buffer_set_data_raw_at_fn)(CNA_VertexBufferHandle, uint64_t, const void*, uint64_t, uint64_t);",
	},
	// The indexed draw's six int32 arguments are all the same type, so dropping
	// one is caught only by the bridge's own call, which passes seven.
	{
		name: "device-draw-indexed-drops-its-start-index",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_draw_indexed_primitives_fn)(CNA_Handle, CNA_PrimitiveType, int32_t, int32_t, int32_t, int32_t, int32_t);",
		new:  "typedef CNA_Result (*cna_graphics_device_draw_indexed_primitives_fn)(CNA_Handle, CNA_PrimitiveType, int32_t, int32_t, int32_t, int32_t);",
	},
	// The binding route takes the array BY POINTER and a count; a by-value
	// struct parameter does not accept the address the bridge passes.
	{
		name: "device-set-vertex-buffers-takes-one-binding-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_set_vertex_buffers_fn)(CNA_Handle, const CNA_VertexBufferBinding*, uint64_t);",
		new:  "typedef CNA_Result (*cna_graphics_device_set_vertex_buffers_fn)(CNA_Handle, CNA_VertexBufferBinding, uint64_t);",
	},
	{
		name: "vertex-buffer-create-takes-the-info-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_vertex_buffer_create_fn)(CNA_Handle, const CNA_VertexBufferCreateInfo*, CNA_VertexBufferHandle*);",
		new:  "typedef CNA_Result (*cna_vertex_buffer_create_fn)(CNA_Handle, CNA_VertexBufferCreateInfo, CNA_VertexBufferHandle*);",
	},
	{
		name: "index-buffer-create-takes-the-info-by-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_index_buffer_create_fn)(CNA_Handle, const CNA_IndexBufferCreateInfo*, CNA_IndexBufferHandle*);",
		new:  "typedef CNA_Result (*cna_index_buffer_create_fn)(CNA_Handle, CNA_IndexBufferCreateInfo, CNA_IndexBufferHandle*);",
	},
	// The windowed upload's BUFFER offset comes before the transfer descriptor.
	// Dropping it leaves a prototype whose remaining arguments still line up by
	// type, so only the bridge's own call -- which passes six arguments -- can
	// catch it.
	{
		name: "index-buffer-set-data-at-drops-its-buffer-offset",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_index_buffer_set_data_at_fn)(CNA_IndexBufferHandle, uint64_t, const CNA_IndexBufferTransfer*, const void*, uint64_t);",
		new:  "typedef CNA_Result (*cna_index_buffer_set_data_at_fn)(CNA_IndexBufferHandle, const CNA_IndexBufferTransfer*, const void*, uint64_t);",
	},
	// The read route reports how many elements it needs. Dropping that output
	// makes an undersized destination unreportable.
	{
		name: "index-buffer-get-data-drops-its-required-element-count",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_index_buffer_get_data_fn)(CNA_IndexBufferHandle, const CNA_IndexBufferTransfer*, void*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_index_buffer_get_data_fn)(CNA_IndexBufferHandle, const CNA_IndexBufferTransfer*, void*, uint64_t);",
	},
	{
		name: "content-asset-path-copy-drops-its-capacity",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_content_manager_copy_asset_path_fn)(CNA_Handle, CNA_StringView, char*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_content_manager_copy_asset_path_fn)(CNA_Handle, CNA_StringView, char*, uint64_t*);",
	},
	// The two Foundation 42 mutations the BRIDGE translation unit cannot catch,
	// and the reason is worth stating. C converts silently between integer
	// widths and between a narrow unsigned return and a wider one, so a
	// manifest that narrowed a tick count to 32 bits -- capping the target step
	// at about 3.6 minutes -- or that turned a result code into a CNA_Bool
	// would still compile against itself. Only comparing the manifest with the
	// CANONICAL declaration catches them, which is what the probe's assignment
	// pins do.
	{
		name: "target-elapsed-time-narrowed-to-32-bits",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_target_elapsed_time_ticks_fn)(CNA_Handle, int64_t);",
		new:  "typedef CNA_Result (*cna_game_set_target_elapsed_time_ticks_fn)(CNA_Handle, int32_t);",
	},
	// Foundation 47. C narrows a CNA_Result to a CNA_Bool silently, so a
	// manifest that turned a frame step's result code into a Boolean would
	// compile against itself and report success for every failing frame.
	{
		name: "manager-back-buffer-width-narrowed",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_width_fn)(CNA_Handle, int32_t);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_set_preferred_back_buffer_width_fn)(CNA_Handle, int16_t);",
	},
	{
		name: "manager-subscribe-swaps-the-callback-and-the-context",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_subscribe_fn)(CNA_Handle, CNA_GraphicsDeviceManagerEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_subscribe_fn)(CNA_Handle, CNA_GraphicsDeviceManagerEvent, void*, CNA_GameEventCallback, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "manager-begin-draw-drops-the-out-parameter",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_begin_draw_fn)(CNA_Handle, CNA_Bool*);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_begin_draw_fn)(CNA_Handle);",
	},
	{
		name: "manager-full-screen-takes-an-int",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_set_is_full_screen_fn)(CNA_Handle, CNA_Bool);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_set_is_full_screen_fn)(CNA_Handle, int32_t);",
	},
	{
		name: "manager-apply-changes-takes-a-flag",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_graphics_device_manager_apply_changes_fn)(CNA_Handle);",
		new:  "typedef CNA_Result (*cna_graphics_device_manager_apply_changes_fn)(CNA_Handle, CNA_Bool);",
	},
	{
		name: "run-one-frame-returns-a-bool",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_run_one_frame_fn)(CNA_Handle);",
		new:  "typedef CNA_Bool (*cna_game_run_one_frame_fn)(CNA_Handle);",
	},
	{
		name: "mouse-visibility-returns-the-previous-value",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_is_mouse_visible_fn)(CNA_Handle, CNA_Bool);",
		new:  "typedef CNA_Bool (*cna_game_set_is_mouse_visible_fn)(CNA_Handle, CNA_Bool);",
	},
	{
		name: "probe-callback-returns-a-result",
		file: "probe.c",
		old:  "static void cna_go_probe_game_event(void* context) { (void)context; }",
		new:  "static CNA_Result cna_go_probe_game_event(void* context) { (void)context; return 0; }",
	},
	{
		name: "probe-callback-takes-a-game-handle",
		file: "probe.c",
		old:  "static void cna_go_probe_game_event(void* context) { (void)context; }",
		new:  "static void cna_go_probe_game_event(CNA_Handle game, void* context) { (void)game; (void)context; }",
	},
	// The begin_draw hook is the one CNA-Go installs that carries a value
	// channel, so its out-parameter is pinned from three directions: dropping
	// it, moving it past the error, and narrowing it to the result channel.
	{
		name: "probe-begin-draw-drops-the-out-parameter",
		file: "probe.c",
		old:  "    CNA_Bool* out_should_draw,\n    CNA_CallbackError* out_error) {\n    (void)game; (void)game_time; (void)context; (void)out_error;",
		new:  "    CNA_CallbackError* out_error) {\n    (void)game; (void)game_time; (void)context; (void)out_error;\n    CNA_Bool* out_should_draw = NULL;",
	},
	{
		name: "probe-begin-draw-swaps-the-out-parameter-and-the-error",
		file: "probe.c",
		old:  "    CNA_Bool* out_should_draw,\n    CNA_CallbackError* out_error) {",
		new:  "    CNA_CallbackError* out_error,\n    CNA_Bool* out_should_draw) {",
	},
	{
		name: "probe-begin-draw-returns-the-drawing-decision",
		file: "probe.c",
		old:  "static CNA_Result cna_go_probe_begin_draw(",
		new:  "static CNA_Bool cna_go_probe_begin_draw(",
	},
	// And the lifecycle shape the other three hooks install.
	{
		name: "probe-lifecycle-drops-the-game-time",
		file: "probe.c",
		old:  "static CNA_Result cna_go_probe_lifecycle(\n    CNA_Handle game,\n    const CNA_GameTime* game_time,\n    void* context,",
		new:  "static CNA_Result cna_go_probe_lifecycle(\n    CNA_Handle game,\n    void* context,",
	},
	// The ABI ADMISSION POLICY, from the side only this translation unit can
	// check: the policy's own numbers against the canonical header's.
	{
		name: "admission-floor-above-the-canonical-minor",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MINIMUM_MINOR = 21,",
		new:  "    CNA_GO_ABI_MINIMUM_MINOR = 99,",
	},
	{
		name: "admission-major-outside-the-canonical-major",
		file: "bridge.h",
		old:  "    CNA_GO_ABI_MAJOR = 0,",
		new:  "    CNA_GO_ABI_MAJOR = 1,",
	},
	{
		name: "encoded-version-mirror-shifts-the-minor-wrongly",
		file: "bridge.h",
		old:  "     (((uint32_t)(minor) & UINT32_C(0xFF)) << 8) | \\",
		new:  "     (((uint32_t)(minor) & UINT32_C(0xFF)) << 4) | \\",
	},
	{
		name: "bridge-callback-result-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_RESULT_CALLBACK = 9,",
		new:  "    CNA_GO_RESULT_CALLBACK = 10,",
	},
	{
		name: "bridge-success-result-mirror-drift",
		file: "bridge.h",
		old:  "    CNA_GO_RESULT_SUCCESS = 0,",
		new:  "    CNA_GO_RESULT_SUCCESS = 1,",
	},
	// A required symbol that CNA no longer declares. The bridge translation
	// unit cannot see this: it resolves symbols by string at run time, so a
	// stale entry compiles and fails only when a consumer loads a library.
	// Foundation 45's window prototypes, from the side only the probe can
	// check. C converts silently between integer widths and between a struct
	// pointer and another struct pointer of the same size, so a manifest that
	// narrowed a client dimension or wrote a Viewport where a Rectangle
	// belongs would compile happily against itself.
	{
		name: "window-subscribe-swaps-the-callback-and-the-context",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_subscribe_fn)(CNA_Handle, CNA_GameWindowEvent, CNA_GameEventCallback, void*, CNA_GameEventRegistrationHandle*);",
		new:  "typedef CNA_Result (*cna_game_window_subscribe_fn)(CNA_Handle, CNA_GameWindowEvent, void*, CNA_GameEventCallback, CNA_GameEventRegistrationHandle*);",
	},
	{
		name: "window-title-route-takes-a-bare-pointer",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_set_window_title_fn)(CNA_Handle, CNA_StringView);",
		new:  "typedef CNA_Result (*cna_game_set_window_title_fn)(CNA_Handle, const char*);",
	},
	{
		name: "end-screen-device-change-narrows-the-client-size",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_end_screen_device_change_fn)(CNA_Handle, CNA_StringView, int32_t, int32_t);",
		new:  "typedef CNA_Result (*cna_game_window_end_screen_device_change_fn)(CNA_Handle, CNA_StringView, int16_t, int16_t);",
	},
	{
		name: "client-bounds-writes-a-viewport",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_get_client_bounds_fn)(CNA_Handle, CNA_Rectangle*);",
		new:  "typedef CNA_Result (*cna_game_window_get_client_bounds_fn)(CNA_Handle, CNA_Viewport*);",
	},
	{
		name: "native-window-handle-narrowed-to-32-bits",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_get_native_handle_ext_fn)(CNA_Handle, uint64_t*);",
		new:  "typedef CNA_Result (*cna_game_window_get_native_handle_ext_fn)(CNA_Handle, uint32_t*);",
	},
	{
		name: "screen-device-name-copy-drops-its-capacity",
		file: "abi_manifest.h",
		old:  "typedef CNA_Result (*cna_game_window_copy_screen_device_name_fn)(CNA_Handle, char*, uint64_t, uint64_t*);",
		new:  "typedef CNA_Result (*cna_game_window_copy_screen_device_name_fn)(CNA_Handle, char*, uint64_t*);",
	},
	{
		name: "stale-required-symbol",
		file: "abi_manifest.h",
		old:  "    X(cna_keyboard_get_state)",
		new:  "    X(cna_game_removed_route_ext) \\\n    X(cna_keyboard_get_state)",
	},
}

// TestManifestProbeRefusesACanonicalHeader proves the guard that makes the
// manifest probe worth running. abi_manifest.h suppresses its own definitions
// whenever a CNA header is present, so a canonical header leaking into this
// translation unit would turn it into a second canonical probe: it would report
// perfect agreement while measuring the same types twice.
func TestManifestProbeRefusesACanonicalHeader(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror",
		"-I"+headers, "-I"+directory, "-include", "CNA/C/cna.h",
		"-c", filepath.Join(directory, "manifest_probe.c"),
		"-o", filepath.Join(directory, "leaked.o"))
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("the manifest probe compiled with a canonical CNA header in scope")
	}
	if !strings.Contains(string(output), "must not see a canonical CNA header") {
		t.Fatalf("the manifest probe failed for the wrong reason:\n%s", output)
	}
}

// TestBridgeTranslationUnitCompilesUnmutated is the control. Every mutation
// below is only evidence if the unmutated source compiles cleanly under the
// same flags.
func TestBridgeTranslationUnitCompilesUnmutated(t *testing.T) {
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageInteropSources(t, root, directory, nil)
	if output, err := compileBridge(directory); err != nil {
		t.Fatalf("unmutated bridge translation unit did not compile: %v\n%s", err, output)
	}
}

// TestProbeTranslationUnitCompilesUnmutated is the same control for the probe,
// which is the only translation unit that sees the canonical CNA header and
// CNA-Go's private manifest at once.
func TestProbeTranslationUnitCompilesUnmutated(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	if output, err := compileProbe(directory, headers); err != nil {
		t.Fatalf("unmutated probe translation unit did not compile: %v\n%s", err, output)
	}
}

// TestBridgeABIMutationsFailToCompile removes one ABI pin at a time from the
// translation unit the cgo build actually compiles.
func TestBridgeABIMutationsFailToCompile(t *testing.T) {
	root := repositoryRoot(t)
	for _, mutation := range bridgeMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageInteropSources(t, root, directory, &mutation)
			output, err := compileBridge(directory)
			if err == nil {
				t.Fatalf("mutation %q compiled cleanly; the ABI pin it removes is not enforced", mutation.name)
			}
			_ = output
		})
	}
}

// TestProbeABIMutationsFailToCompile does the same for the canonical-header
// probe, including the one mutation that makes CNA-Go's private manifest
// disagree with the shipped CNA declarations.
func TestProbeABIMutationsFailToCompile(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	for _, mutation := range probeMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageProbeSources(t, root, directory, &mutation)
			if _, err := compileProbe(directory, headers); err == nil {
				t.Fatalf("mutation %q compiled cleanly; the ABI pin it removes is not enforced", mutation.name)
			}
		})
	}
}

// TestManifestRoutesAreSelfDescribing keeps the measured prototype table and
// the bridge's required-symbol list from drifting apart. They cannot drift any
// more: the table IS the manifest now, parsed from the same X-macro list the
// cgo build resolves, so this proves the parser reads it rather than that two
// hand-maintained copies still agree.
func TestManifestRoutesAreSelfDescribing(t *testing.T) {
	manifest := readSource(t, filepath.Join(repositoryRoot(t), "internal", "interop", "abi_manifest.h"))
	routes, err := parseManifestRoutes(manifest)
	if err != nil {
		t.Fatalf("parse the manifest: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("the manifest declares no bound routes")
	}
	seen := map[string]bool{}
	for _, entry := range routes {
		if seen[entry.name] {
			t.Fatalf("%s appears twice in the required-symbol list", entry.name)
		}
		seen[entry.name] = true
		if !strings.Contains(manifest, "X("+entry.name+")") {
			t.Fatalf("%s is measured but the bridge never resolves it", entry.name)
		}
	}
	for _, required := range []string{"cna_game_subscribe", "cna_game_unsubscribe", "cna_get_abi_version"} {
		if !seen[required] {
			t.Fatalf("%s is bound by the bridge but has no measured prototype", required)
		}
	}
}

// TestManifestRoutesRejectAStaleEntry proves the parser refuses a
// required-symbol entry with no prototype of its own rather than counting it as
// a route with zero parameters.
func TestManifestRoutesRejectAStaleEntry(t *testing.T) {
	manifest := readSource(t, filepath.Join(repositoryRoot(t), "internal", "interop", "abi_manifest.h"))
	mutated := strings.Replace(manifest,
		"    X(cna_keyboard_get_state)",
		"    X(cna_game_removed_route_ext) \\\n    X(cna_keyboard_get_state)", 1)
	if mutated == manifest {
		t.Fatal("the required-symbol list no longer ends with cna_keyboard_get_state")
	}
	if _, err := parseManifestRoutes(mutated); err == nil {
		t.Fatal("a required symbol with no typedef was accepted")
	}
}

// TestManifestProbeCompilesUnmutated is the control for the manifest-only
// translation unit: the one that measures CNA-Go's private declarations in the
// same environment cgo gives bridge.c.
func TestManifestProbeCompilesUnmutated(t *testing.T) {
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	if output, err := compileManifestProbe(directory); err != nil {
		t.Fatalf("unmutated manifest probe did not compile: %v\n%s", err, output)
	}
}

// TestUnmutatedProbesAgreeOnEveryMeasurement is the control for the comparison
// the layout mutations below falsify.
func TestUnmutatedProbesAgreeOnEveryMeasurement(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	directory := t.TempDir()
	stageProbeSources(t, root, directory, nil)
	canonical, manifest := measureStagedProbes(t, directory, headers)
	shared := 0
	for key, value := range canonical {
		other, ok := manifest[key]
		if !ok {
			continue
		}
		shared++
		if other != value {
			t.Fatalf("%s: canonical %d, manifest %d", key, value, other)
		}
	}
	if shared == 0 {
		t.Fatal("the two probes share no measurement, so comparing them proves nothing")
	}
}

// layoutMutations change CNA-Go's private manifest in ways that COMPILE
// cleanly everywhere -- in the bridge, in the canonical probe, and in the
// manifest probe -- because C aggregates are written by field name. Each one
// still changes the bytes the shipped binding would pass to CNA. They are the
// class the ABI evidence could not see until the manifest's own declarations
// were measured, and each must produce a divergence.
var layoutMutations = []sourceMutation{
	{
		// CNA_DisplayMode's aspect_ratio and format are neighbours of the same
		// width and a different KIND: a float and a uint32. Swapping them moves
		// no size and no later offset, and the bridge's assignments convert
		// silently in both directions -- an aspect ratio of 1.666 would arrive
		// as the format identity 1, which is Bgr565, and a format of 0 would
		// arrive as an aspect ratio of 0.0. Only the two offsets move.
		name: "display-mode-aspect-ratio-and-format-swapped",
		file: "abi_manifest.h",
		old:  "    float aspect_ratio;\n    CNA_SurfaceFormat format;\n} CNA_DisplayMode;",
		new:  "    CNA_SurfaceFormat format;\n    float aspect_ratio;\n} CNA_DisplayMode;",
	},
	{
		// The create info's reserved bytes are what CNA uses to state, rather
		// than imply, where `format` sits after the one-byte mip flag. This
		// mutation gives them a fourth byte, which pushes the format from 20 to
		// 24 and grows the structure.
		//
		// The obvious mutation -- DROPPING reserved[3] entirely -- was written
		// first and removed, because it is not observable: the compiler inserts
		// exactly those three bytes of padding to align a uint32 after a
		// uint8, so the declared and the implied layouts are identical. A
		// control that cannot fail is not evidence, and that is recorded here
		// rather than left as a passing test nobody re-derives.
		name: "texture-create-info-reserved-bytes-widened",
		file: "abi_manifest.h",
		old:  "    CNA_Bool mip_map;\n    uint8_t reserved[3];\n    CNA_SurfaceFormat format;",
		new:  "    CNA_Bool mip_map;\n    uint8_t reserved[4];\n    CNA_SurfaceFormat format;",
	},
	{
		// Foundation 58. CNA_RenderTarget2DCreateInfo's trailing `reserved1` is
		// a full uint32 and holds the structure at 40 bytes; dropping it makes
		// it 36, and struct_size is a value CNA VALIDATES, so this is the one
		// reserved field in the family whose removal is observable.
		//
		// The other two reserved fields in these structures -- `reserved0[3]`
		// after the mip flag and `reserved[2]` after the two booleans -- are
		// deliberately NOT mutated. The compiler inserts exactly those bytes as
		// padding, so declaring them and omitting them produce identical
		// layouts, and a control that cannot fail is not evidence. That is the
		// rule the texture create info's own comment settled and it is applied
		// here rather than re-argued.
		name: "render-target-create-info-trailing-reserved-dropped",
		file: "abi_manifest.h",
		old:  "    CNA_RenderTargetUsage usage;\n    uint32_t reserved1;\n} CNA_RenderTarget2DCreateInfo;",
		new:  "    CNA_RenderTargetUsage usage;\n} CNA_RenderTarget2DCreateInfo;",
	},
	{
		// The create info's format and depth format are adjacent uint32s of
		// different MEANING. Swapping them moves no size and no later offset,
		// and the bridge's assignments convert silently in both directions: a
		// requested SurfaceFormat.Color would arrive as DepthFormat.None and a
		// requested Depth24Stencil8 would arrive as SurfaceFormat.Bgra4444.
		// Only the two offsets move, and only the comparison catches it.
		name: "render-target-create-info-format-and-depth-format-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_SurfaceFormat format;\n    CNA_DepthFormat depth_format;\n    int32_t multi_sample_count;\n    CNA_RenderTargetUsage usage;\n    uint32_t reserved1;",
		new:  "    CNA_DepthFormat depth_format;\n    CNA_SurfaceFormat format;\n    int32_t multi_sample_count;\n    CNA_RenderTargetUsage usage;\n    uint32_t reserved1;",
	},
	{
		// CNA_RenderTargetInfo's two trailing booleans are one byte each and
		// adjacent. Swapping them is the most dangerous mutation in this file
		// and the least visible: the bridge would report a render target as
		// having no renderer storage exactly when its contents were lost, and
		// as having lost contents exactly when the renderer had none. Both
		// values are valid booleans either way.
		name: "render-target-info-content-lost-and-renderer-available-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_Bool is_content_lost;\n    CNA_Bool renderer_available;",
		new:  "    CNA_Bool renderer_available;\n    CNA_Bool is_content_lost;",
	},
	{
		// The decode info's seven reserved bytes hold the structure at 24
		// bytes after a one-byte zoom flag. An eighth grows it to 32, which is
		// what CNA would read past.
		name: "decode-info-reserved-bytes-widened",
		file: "abi_manifest.h",
		old:  "    CNA_Bool zoom;\n    uint8_t reserved[7];",
		new:  "    CNA_Bool zoom;\n    uint8_t reserved[8];",
	},
	{
		// A narrowed image-format identity. PNG is 0 and JPEG is 1, so every
		// declared value survives sixteen bits and nothing changes until CNA
		// adds a format above 65535 -- and, like the clear mask, the probe
		// cannot see it, because the manifest declares the alias inside a guard
		// the canonical header satisfies.
		name: "texture-image-format-narrowed",
		file: "abi_manifest.h",
		old:  "typedef uint32_t CNA_TextureImageFormat;",
		new:  "typedef uint16_t CNA_TextureImageFormat;",
	},
	{
		// The transfer's start_index and element_count are both uint64 and
		// adjacent. Swapping them is invisible to every compiler check and
		// turns "sixteen elements from zero" into "zero elements from
		// sixteen" -- a transfer that succeeds and copies nothing.
		name: "texture-transfer-start-and-count-swapped",
		file: "abi_manifest.h",
		old:  "    uint64_t start_index;\n    uint64_t element_count;",
		new:  "    uint64_t element_count;\n    uint64_t start_index;",
	},
	{
		// The packed-storage widths are what let CNA-Go pass a Go struct
		// straight through. A Bgr565 read as four bytes would stride twice as
		// far through the caller's array.
		name: "packed-bgr565-widened",
		file: "abi_manifest.h",
		old:  "typedef uint16_t CNA_PackedBgr565;",
		new:  "typedef uint32_t CNA_PackedBgr565;",
	},
	{
		// A narrowed clear mask. Target, DepthBuffer and Stencil are 1, 2 and 4,
		// so every declared bit survives 16 bits and nothing observable changes
		// until a caller passes a bit CNA adds later -- which is exactly the
		// kind of defect that ships.
		//
		// It is a LAYOUT control rather than a probe control, and the reason is
		// structural: the manifest declares CNA_ClearOptions inside
		// `#ifndef CNA_C_GRAPHICS_DEVICE_H`, so in the probe -- which includes
		// the canonical header -- the narrowed typedef is never compiled at
		// all. Only the manifest-only probe sees it, and only sizeof reports it.
		name: "clear-options-mask-narrowed",
		file: "abi_manifest.h",
		old:  "typedef uint32_t CNA_ClearOptions;",
		new:  "typedef uint16_t CNA_ClearOptions;",
	},
	{
		// A destination rectangle of int16 is the same four fields at the same
		// offset, and every screen coordinate above 32767 -- or below -32768 --
		// wraps. C narrows an int32 assignment into it without a word, so the
		// bridge compiles; only sizeof and the offsets after it move.
		name: "sprite-destination-rectangle-narrowed",
		file: "abi_manifest.h",
		old:  "    CNA_Handle texture;\n    CNA_Rectangle destination;",
		new:  "    CNA_Handle texture;\n    struct { int16_t x, y, width, height; } destination;",
	},
	{
		name: "sprite-command-source-and-colour-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_Rectangle source;\n    CNA_Color color;",
		new:  "    CNA_Color color;\n    CNA_Rectangle source;",
	},
	{
		name: "keyboard-state-word-count-doubled",
		file: "abi_manifest.h",
		old:  "    uint64_t pressed_key_words[4];",
		new:  "    uint64_t pressed_key_words[8];",
	},
	{
		name: "viewport-depth-widened-to-double",
		file: "abi_manifest.h",
		old:  "    float min_depth;\n    float max_depth;",
		new:  "    double min_depth;\n    double max_depth;",
	},
	{
		name: "texture-info-struct-size-widened",
		file: "abi_manifest.h",
		old:  "typedef struct CNA_Texture2DInfo {\n    uint32_t struct_size;",
		new:  "typedef struct CNA_Texture2DInfo {\n    uint64_t struct_size;",
	},
	{
		name: "game-time-tick-count-narrowed",
		file: "abi_manifest.h",
		old:  "    int64_t total_game_time_ticks;\n    int64_t elapsed_game_time_ticks;",
		new:  "    int32_t total_game_time_ticks;\n    int64_t elapsed_game_time_ticks;",
	},
	{
		name: "string-view-length-narrowed",
		file: "abi_manifest.h",
		old:  "typedef struct CNA_StringView { const char* data; uint64_t byte_length; } CNA_StringView;",
		new:  "typedef struct CNA_StringView { const char* data; uint32_t byte_length; } CNA_StringView;",
	},
	// Foundation 63. The content create-info's root directory moved behind the
	// reserved field. Every field still exists, every name still resolves and
	// bridge.c still compiles: `info.root_directory = ...` is valid either way.
	// What changes is WHERE CNA reads eight bytes of pointer from, which is the
	// difference between a content manager rooted at the caller's directory and
	// one rooted at whatever the zeroed reserved word points at.
	{
		name: "content-create-info-root-moved-behind-reserved",
		file: "abi_manifest.h",
		old:  "    uint32_t struct_size;\n    uint32_t struct_version;\n    CNA_StringView root_directory;\n    uint64_t reserved;\n} CNA_ContentManagerCreateInfo;",
		new:  "    uint32_t struct_size;\n    uint32_t struct_version;\n    uint64_t reserved;\n    CNA_StringView root_directory;\n} CNA_ContentManagerCreateInfo;",
	},
	// The two header words swapped. They are both uint32 and adjacent, so the
	// struct is byte-identical -- but bridge.c writes sizeof(info) into the
	// first and the version 1 into the second, so a swap tells CNA it was
	// handed a struct of size 1 and version 24. A versioned create-info exists
	// precisely to be checked, and this is the mutation that proves the check
	// is being fed the right two words.
	// Foundation 65. The index-buffer create-info's two identity words are
	// adjacent uint32s, so this swap is byte-identical to C and turns a 16-bit
	// read-write buffer into a 32-bit write-only one -- a buffer that creates
	// successfully, strides twice as far, and refuses every GetData.
	// Foundation 66. CNA_VertexElement's four fields are all four bytes wide,
	// so this permutation is byte-identical to C -- and it describes a
	// completely different layout: every element's OFFSET becomes its FORMAT.
	{
		name: "vertex-element-offset-and-format-swapped",
		file: "abi_manifest.h",
		old:  "    int32_t offset;\n    CNA_VertexElementFormat format;\n    CNA_VertexElementUsage usage;\n    int32_t usage_index;",
		new:  "    CNA_VertexElementFormat format;\n    int32_t offset;\n    CNA_VertexElementUsage usage;\n    int32_t usage_index;",
	},
	// The buffer create-info's handle moved behind the two int32s. Every field
	// still exists and bridge.c still compiles; what changes is where CNA reads
	// eight bytes of declaration handle from.
	{
		name: "vertex-create-info-declaration-moved-behind-the-counts",
		file: "abi_manifest.h",
		old:  "    CNA_VertexDeclarationHandle vertex_declaration;\n    int32_t vertex_count;\n    CNA_BufferUsage buffer_usage;",
		new:  "    int32_t vertex_count;\n    CNA_BufferUsage buffer_usage;\n    CNA_VertexDeclarationHandle vertex_declaration;",
	},
	// The info struct's stride moved ahead of the three flags, which changes
	// which bytes a stride is read from -- and the stride is what every
	// transfer's fit check is measured in.
	// Foundation 67. The binding array is the one place a CNA STRUCT ARRAY
	// crosses on the device's own surface, and its three fields are a 64-bit
	// handle and two int32s -- so moving the handle behind them is
	// byte-identical to C and makes CNA read a buffer token out of the offset
	// and frequency words.
	{
		name: "vertex-binding-handle-moved-behind-the-offsets",
		file: "abi_manifest.h",
		old:  "    CNA_VertexBufferHandle vertex_buffer;\n    int32_t vertex_offset;\n    int32_t instance_frequency;\n} CNA_VertexBufferBinding;",
		new:  "    int32_t vertex_offset;\n    int32_t instance_frequency;\n    CNA_VertexBufferHandle vertex_buffer;\n} CNA_VertexBufferBinding;",
	},
	{
		name: "vertex-info-stride-moved-ahead-of-the-flags",
		file: "abi_manifest.h",
		old:  "    CNA_Bool dynamic;\n    CNA_Bool is_content_lost;\n    CNA_Bool has_renderer;\n    uint8_t reserved0;\n    int32_t vertex_stride;",
		new:  "    int32_t vertex_stride;\n    CNA_Bool dynamic;\n    CNA_Bool is_content_lost;\n    CNA_Bool has_renderer;\n    uint8_t reserved0;",
	},
	{
		name: "index-create-info-element-size-and-usage-swapped",
		file: "abi_manifest.h",
		old:  "    int32_t index_count;\n    CNA_IndexElementSize index_element_size;\n    CNA_BufferUsage buffer_usage;\n    CNA_Bool dynamic;\n    uint8_t reserved[3];",
		new:  "    int32_t index_count;\n    CNA_BufferUsage buffer_usage;\n    CNA_IndexElementSize index_element_size;\n    CNA_Bool dynamic;\n    uint8_t reserved[3];",
	},
	// The info struct's three CNA_Bools are one byte each and adjacent, so
	// swapping two of them is invisible to C and makes a live buffer report
	// itself as content-lost.
	{
		name: "index-info-content-lost-and-has-renderer-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_Bool dynamic;\n    CNA_Bool is_content_lost;\n    CNA_Bool has_renderer;\n    uint8_t reserved;",
		new:  "    CNA_Bool dynamic;\n    CNA_Bool has_renderer;\n    CNA_Bool is_content_lost;\n    uint8_t reserved;",
	},
	// The transfer's window pair, the same shape the texture transfer's has:
	// two adjacent uint64s whose swap turns "six indices from zero" into "zero
	// indices from six" -- a transfer that succeeds and moves nothing.
	{
		name: "index-transfer-start-and-count-swapped",
		file: "abi_manifest.h",
		old:  "    CNA_SetDataOptions options;\n    uint64_t start_index;\n    uint64_t element_count;\n} CNA_IndexBufferTransfer;",
		new:  "    CNA_SetDataOptions options;\n    uint64_t element_count;\n    uint64_t start_index;\n} CNA_IndexBufferTransfer;",
	},
	{
		name: "content-create-info-size-and-version-swapped",
		file: "abi_manifest.h",
		old:  "    uint32_t struct_size;\n    uint32_t struct_version;\n    CNA_StringView root_directory;",
		new:  "    uint32_t struct_version;\n    uint32_t struct_size;\n    CNA_StringView root_directory;",
	},
}

// TestLayoutMutationsDiverge is the falsifiability proof for the manifest-side
// measurement. Every mutation compiles; the comparison is what catches it.
func TestLayoutMutationsDiverge(t *testing.T) {
	headers := canonicalHeaderRoot(t)
	root := repositoryRoot(t)
	for _, mutation := range layoutMutations {
		t.Run(mutation.name, func(t *testing.T) {
			directory := t.TempDir()
			stageProbeSources(t, root, directory, &mutation)
			canonical, manifest := measureStagedProbes(t, directory, headers)
			for key, value := range canonical {
				if other, ok := manifest[key]; ok && other != value {
					return
				}
			}
			t.Fatalf("mutation %q changed no measured value; the manifest layout is not pinned", mutation.name)
		})
	}
}

// measureStagedProbes compiles and runs both probes over one staged source
// tree and returns their measurements.
func measureStagedProbes(t *testing.T, directory, headers string) (map[string]int, map[string]int) {
	t.Helper()
	canonicalBinary := filepath.Join(directory, "canonical-probe")
	arguments := []string{
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I" + headers, "-I" + directory, "-DCNA_GO_LAYOUT_ONLY",
		filepath.Join(directory, "probe.c"), "-o", canonicalBinary,
	}
	if output, err := exec.Command("gcc", arguments...).CombinedOutput(); err != nil {
		t.Fatalf("canonical probe did not compile: %v\n%s", err, output)
	}
	manifestBinary := filepath.Join(directory, "manifest-probe")
	arguments = []string{
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I" + directory,
		filepath.Join(directory, "manifest_probe.c"), "-o", manifestBinary,
	}
	if output, err := exec.Command("gcc", arguments...).CombinedOutput(); err != nil {
		t.Fatalf("manifest probe did not compile: %v\n%s", err, output)
	}
	return runStagedProbe(t, canonicalBinary), runStagedProbe(t, manifestBinary)
}

func runStagedProbe(t *testing.T, binary string) map[string]int {
	t.Helper()
	output, err := exec.Command(binary).Output()
	if err != nil {
		t.Fatalf("run %s: %v", binary, err)
	}
	measurements := map[string]int{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		if value, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
			measurements[parts[0]] = value
		}
	}
	return measurements
}

func compileBridge(directory string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-c", filepath.Join(directory, "bridge.c"),
		"-o", filepath.Join(directory, "bridge.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

func compileProbe(directory, headers string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-I"+headers,
		"-c", filepath.Join(directory, "probe.c"),
		"-o", filepath.Join(directory, "probe.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

// compileManifestProbe deliberately passes NO canonical include root. That is
// the whole point of the translation unit: it must measure CNA-Go's private
// declarations, and it refuses to compile if a CNA header reaches it.
func compileManifestProbe(directory string) (string, error) {
	command := exec.Command("gcc",
		"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types",
		"-c", filepath.Join(directory, "manifest_probe.c"),
		"-o", filepath.Join(directory, "manifest_probe.o"),
		"-I"+directory)
	output, err := command.CombinedOutput()
	return string(output), err
}

func stageInteropSources(t *testing.T, root, directory string, mutation *sourceMutation) {
	t.Helper()
	for _, name := range []string{"bridge.c", "bridge.h", "abi_manifest.h"} {
		content := readSource(t, filepath.Join(root, "internal", "interop", name))
		if mutation != nil && mutation.file == name {
			content = applyMutation(t, content, *mutation)
		}
		writeSource(t, filepath.Join(directory, name), content)
	}
}

func stageProbeSources(t *testing.T, root, directory string, mutation *sourceMutation) {
	t.Helper()
	// The staged copy is flat, so both probes' relative includes of the private
	// headers become sibling includes. Nothing else about them changes.
	flatten := func(source string) string {
		source = strings.Replace(source, `#include "../../../internal/interop/abi_manifest.h"`, `#include "abi_manifest.h"`, 1)
		return strings.Replace(source, `#include "../../../internal/interop/bridge.h"`, `#include "bridge.h"`, 1)
	}
	staged := map[string]string{
		"probe.c":          flatten(readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "probe.c"))),
		"manifest_probe.c": flatten(readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "manifest_probe.c"))),
		"measurements.inc": readSource(t, filepath.Join(root, "tools", "native_abi", "testdata", "measurements.inc")),
		"abi_manifest.h":   readSource(t, filepath.Join(root, "internal", "interop", "abi_manifest.h")),
		"bridge.h":         readSource(t, filepath.Join(root, "internal", "interop", "bridge.h")),
	}
	if mutation != nil {
		content, ok := staged[mutation.file]
		if !ok {
			t.Fatalf("probe mutation %q names an unstaged file %q", mutation.name, mutation.file)
		}
		staged[mutation.file] = applyMutation(t, content, *mutation)
	}
	for name, content := range staged {
		writeSource(t, filepath.Join(directory, name), content)
	}
}

func applyMutation(t *testing.T, content string, mutation sourceMutation) string {
	t.Helper()
	if !strings.Contains(content, mutation.old) {
		t.Fatalf("mutation %q no longer matches %s; the evidence it carries is stale", mutation.name, mutation.file)
	}
	return strings.Replace(content, mutation.old, mutation.new, 1)
}

func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeSource(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	t.Fatal("could not locate the module root")
	return ""
}

// canonicalHeaderRoot resolves the canonical CNA include tree the way every
// other native gate does: from CNA_C_API_INCLUDE when it is set, and otherwise
// from the path the qualification artifact documents. A checkout without the
// headers skips rather than fails, because the probe mutations measure the
// canonical declarations and cannot be evaluated without them.
func canonicalHeaderRoot(t *testing.T) string {
	t.Helper()
	candidates := []string{os.Getenv("CNA_C_API_INCLUDE")}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "deps", "cna-c-abi-0.21.0", "include"))
	}
	candidates = append(candidates, filepath.Join(repositoryRoot(t), "..", "..", "cnanext", "modules", "c-api", "include"))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(candidate, "CNA", "C", "cna.h")); err == nil {
			return candidate
		}
	}
	t.Skip("canonical CNA C headers are not available; set CNA_C_API_INCLUDE")
	return ""
}

// TestCanonicalHeaderTreeDigestIsDeterministicAndOrderIndependent holds the
// content identity the header root is now recorded by.
//
// The digest exists because the default header root is a LIVE checkout that has
// already moved past the pinned library, and a path cannot say which headers a
// report was produced against. Two properties make it evidence: it is stable
// across runs, and it changes when any header changes -- including a change
// that only moves bytes between two files, which the per-file path and length
// prefixes are there to catch.
func TestCanonicalHeaderTreeDigestIsDeterministicAndOrderIndependent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "CNA", "C"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("CNA/C/a.h", "void cna_one(void);\n")
	write("CNA/C/b.h", "void cna_two(void);\n")
	write("CNA/C/notes.txt", "ignored: not a header\n")

	first, files, err := hashHeaderTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 {
		t.Fatalf("digest covered %d files, want the two .h files", files)
	}
	second, _, err := hashHeaderTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("the header digest is not deterministic")
	}
	// A byte moved from one header to another. Without the per-file path and
	// length prefixes the concatenation would be identical.
	write("CNA/C/a.h", "void cna_one(void);")
	write("CNA/C/b.h", "\nvoid cna_two(void);\n")
	moved, _, err := hashHeaderTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if moved == first {
		t.Fatal("moving a byte between two headers left the digest unchanged")
	}
	// A new header changes it too, and so does the file count.
	write("CNA/C/c.h", "void cna_three(void);\n")
	added, addedFiles, err := hashHeaderTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if added == moved || addedFiles != 3 {
		t.Fatalf("adding a header produced digest-equal=%t files=%d", added == moved, addedFiles)
	}
}

// TestDeliberatelyUnboundRoutesAreChecked holds the four claims the registry
// makes about itself. A registry that could name a route CNA never declared, or
// one the manifest already resolves, would be prose with a struct around it.
func TestDeliberatelyUnboundRoutesAreChecked(t *testing.T) {
	declared := map[string]struct{}{}
	for _, entry := range deliberatelyUnboundRoutes {
		declared[entry.Route] = struct{}{}
	}
	// The control: the real registry against a header set that declares exactly
	// it, and a manifest that binds none of it.
	control := report{}
	if got := verifyUnboundRoutes(&control, declared, map[string]struct{}{}); got != len(deliberatelyUnboundRoutes) {
		t.Fatalf("verifyUnboundRoutes counted %d of %d", got, len(deliberatelyUnboundRoutes))
	}
	if len(control.Findings) != 0 {
		t.Fatalf("the unmutated registry produced findings: %v", control.Findings)
	}

	// A route the canonical headers do not declare. This is the stale-entry
	// case: CNA renames or removes a route and the registry keeps explaining
	// why CNA-Go does not bind something that no longer exists.
	missing := report{}
	shrunk := map[string]struct{}{}
	for name := range declared {
		shrunk[name] = struct{}{}
	}
	delete(shrunk, deliberatelyUnboundRoutes[0].Route)
	verifyUnboundRoutes(&missing, shrunk, map[string]struct{}{})
	if len(missing.Findings) == 0 {
		t.Fatal("a route the canonical headers do not declare produced no finding")
	}

	// A route that is BOTH recorded as unbound and bound by the manifest. That
	// is the contradiction the registry exists to make impossible: a route
	// cannot be deliberately unbound and resolved at the same time.
	both := report{}
	verifyUnboundRoutes(&both, declared, map[string]struct{}{deliberatelyUnboundRoutes[0].Route: {}})
	if len(both.Findings) == 0 {
		t.Fatal("a route recorded as unbound AND bound produced no finding")
	}

	// Every real entry carries a recorded class, a member and a detail, and no
	// route is recorded twice.
	seen := map[string]bool{}
	for _, entry := range deliberatelyUnboundRoutes {
		if seen[entry.Route] {
			t.Fatalf("%s is recorded twice", entry.Route)
		}
		seen[entry.Route] = true
		if _, known := unboundRouteClasses[entry.Class]; !known {
			t.Fatalf("%s has unrecorded class %q", entry.Route, entry.Class)
		}
		if strings.TrimSpace(entry.Member) == "" || strings.TrimSpace(entry.Detail) == "" {
			t.Fatalf("%s records no member or no detail", entry.Route)
		}
		if !strings.HasPrefix(entry.Route, "cna_") {
			t.Fatalf("%s is not a canonical CNA route name", entry.Route)
		}
	}
}

// TestEveryUnboundRouteIsDeclaredByThePinnedHeaders is the same closure against
// the REAL canonical headers, so a registry entry cannot be a name nobody
// checked. It skips rather than fails when the pinned header tree is absent,
// which is the same rule the probe mutations already follow.
func TestEveryUnboundRouteIsDeclaredByThePinnedHeaders(t *testing.T) {
	root := os.Getenv("CNA_GO_CANONICAL_HEADERS")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "deps", "cna-c-abi-0.21.0", "include")
	}
	if _, err := os.Stat(filepath.Join(root, "CNA", "C")); err != nil {
		t.Skipf("pinned canonical headers are not present at %s", root)
	}
	declared, err := canonicalDeclarations(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range deliberatelyUnboundRoutes {
		if _, exists := declared[entry.Route]; !exists {
			t.Errorf("%s is recorded as deliberately unbound and the pinned headers do not declare it", entry.Route)
		}
	}
}
