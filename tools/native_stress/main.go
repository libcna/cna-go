// Command native_stress runs each native lifetime scenario in a crash-isolated
// subprocess. It does not claim sanitizer or leak-detector coverage.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	audio "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Audio"
	content "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Content"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	input "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input"
	touch "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Input/Touch"
	storage "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Storage"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

type counters struct {
	GameCycles           int `json:"GAME_CYCLES"`
	GameRecreationCycles int `json:"GAME_RECREATION_CYCLES"`
	TextureCycles        int `json:"TEXTURE_CYCLES"`
	// The render-target semantic slice. Binds and BindRefusals are mutually
	// exclusive per cycle and are counted separately on purpose: a refusal is
	// CNA's documented answer on a backend with no off-screen storage, and a
	// run that only ever refused must not read as a run that only ever bound.
	RenderTargetCycles             int `json:"RENDER_TARGET_CYCLES"`
	RenderTargetCreations          int `json:"RENDER_TARGET_CREATIONS"`
	RenderTargetDescriptionChecks  int `json:"RENDER_TARGET_DESCRIPTION_CHECKS"`
	RenderTargetSubstitutionChecks int `json:"RENDER_TARGET_SUBSTITUTION_CHECKS"`
	RenderTargetBinds              int `json:"RENDER_TARGET_BINDS"`
	RenderTargetBindRefusals       int `json:"RENDER_TARGET_BIND_REFUSALS"`
	RenderTargetUnbinds            int `json:"RENDER_TARGET_UNBINDS"`
	RenderTargetPixelChecks        int `json:"RENDER_TARGET_PIXEL_CHECKS"`
	RenderTargetReadbackRefusals   int `json:"RENDER_TARGET_READBACK_REFUSALS"`
	RenderTargetSpriteDraws        int `json:"RENDER_TARGET_SPRITE_DRAWS"`
	RenderTargetDisposalChecks     int `json:"RENDER_TARGET_DISPOSAL_CHECKS"`
	// InheritedDisposeVirtualChecks counts the runs of the one control that can
	// tell the inherited Dispose() reaching the DERIVED override from it
	// reaching the composed base's slot. Every managed observable agrees for
	// both; only the native handle disagrees.
	InheritedDisposeVirtualChecks int `json:"INHERITED_DISPOSE_VIRTUAL_CHECKS"`
	SpriteBatchCycles             int `json:"SPRITEBATCH_CYCLES"`
	// The content slice. Loads and LoadRefusals are counted separately for the
	// same reason the render-target bind is: a refusal is CNA answering "this
	// asset is not there", and a run that only ever refused must not read as a
	// run that loaded.
	ContentCycles               int `json:"CONTENT_CYCLES"`
	ContentManagerCreations     int `json:"CONTENT_MANAGER_CREATIONS"`
	ContentIdentityChecks       int `json:"CONTENT_IDENTITY_CHECKS"`
	ContentRootRoundTrips       int `json:"CONTENT_ROOT_ROUND_TRIPS"`
	ContentAssetPathChecks      int `json:"CONTENT_ASSET_PATH_CHECKS"`
	ContentLoads                int `json:"CONTENT_LOADS"`
	ContentLoadPixelChecks      int `json:"CONTENT_LOAD_PIXEL_CHECKS"`
	ContentLoadReadbackRefusals int `json:"CONTENT_LOAD_READBACK_REFUSALS"`
	ContentCacheChecks          int `json:"CONTENT_CACHE_CHECKS"`
	ContentLoadRefusals         int `json:"CONTENT_LOAD_REFUSALS"`
	ContentTypeRefusals         int `json:"CONTENT_TYPE_REFUSALS"`
	ContentUnloadCalls          int `json:"CONTENT_UNLOAD_CALLS"`
	ContentDisposalChecks       int `json:"CONTENT_DISPOSAL_CHECKS"`
	// The sprite-font slice. Foundation 69. Loads and LoadRefusals are counted
	// apart for the reason the texture load's are: a refusal is CNA answering
	// "this asset is not there", and a run that only ever refused must not read
	// as a run that loaded a font.
	SpriteFontCycles             int `json:"SPRITE_FONT_CYCLES"`
	SpriteFontLoads              int `json:"SPRITE_FONT_LOADS"`
	SpriteFontLoadRefusals       int `json:"SPRITE_FONT_LOAD_REFUSALS"`
	SpriteFontGlyphChecks        int `json:"SPRITE_FONT_GLYPH_CHECKS"`
	SpriteFontMeasureChecks      int `json:"SPRITE_FONT_MEASURE_CHECKS"`
	SpriteFontDivergenceChecks   int `json:"SPRITE_FONT_DIVERGENCE_CHECKS"`
	SpriteFontSetterRoundTrips   int `json:"SPRITE_FONT_SETTER_ROUND_TRIPS"`
	SpriteFontRefusals           int `json:"SPRITE_FONT_REFUSALS"`
	SpriteFontCacheChecks        int `json:"SPRITE_FONT_CACHE_CHECKS"`
	SpriteFontDrawStringSubmits  int `json:"SPRITE_FONT_DRAW_STRING_SUBMITS"`
	SpriteFontDrawStringGuards   int `json:"SPRITE_FONT_DRAW_STRING_GUARDS"`
	SpriteFontDrawStringRefusals int `json:"SPRITE_FONT_DRAW_STRING_REFUSALS"`
	// The volume/cube slice. Foundation 71. CNA documents a volume texture as a
	// RENDERER capability -- cna_texture3d_create returns NOT_SUPPORTED where
	// the renderer has no volume storage -- so a creation refusal is recorded
	// rather than failing the run.
	TextureVolumeCycles          int `json:"TEXTURE_VOLUME_CYCLES"`
	TextureCubeCreations         int `json:"TEXTURE_CUBE_CREATIONS"`
	TextureCubeCreationRefusals  int `json:"TEXTURE_CUBE_CREATION_REFUSALS"`
	TextureCubeRoundTrips        int `json:"TEXTURE_CUBE_ROUND_TRIPS"`
	TextureCubeTransferRefusals  int `json:"TEXTURE_CUBE_TRANSFER_REFUSALS"`
	Texture3DCreations           int `json:"TEXTURE_3D_CREATIONS"`
	Texture3DCreationRefusals    int `json:"TEXTURE_3D_CREATION_REFUSALS"`
	Texture3DRoundTrips          int `json:"TEXTURE_3D_ROUND_TRIPS"`
	Texture3DTransferRefusals    int `json:"TEXTURE_3D_TRANSFER_REFUSALS"`
	TextureVolumeElementRefusals int `json:"TEXTURE_VOLUME_ELEMENT_REFUSALS"`
	TextureVolumeDisposalChecks  int `json:"TEXTURE_VOLUME_DISPOSAL_CHECKS"`
	// The presentation slice. Foundation 73. Reset, Present, the back-buffer
	// readback, the cube render target and the owned device constructor.
	//
	// CNA documents the back-buffer readback as a RENDERER capability and
	// refuses it where the backend has no readback path, so a refusal is
	// recorded rather than failing the run -- the same shape the render-target
	// pixel check already has. Reset is NOT in that class: CNA's reset route is
	// declared unconditionally, so a refusal there is a defect.
	PresentationCycles            int `json:"PRESENTATION_CYCLES"`
	PresentationParameterReads    int `json:"PRESENTATION_PARAMETER_READS"`
	PresentationResetCalls        int `json:"PRESENTATION_RESET_CALLS"`
	PresentationResetRefusals     int `json:"PRESENTATION_RESET_REFUSALS"`
	PresentationRectangleRefusals int `json:"PRESENTATION_RECTANGLE_REFUSALS"`
	BackBufferReads               int `json:"BACK_BUFFER_READS"`
	BackBufferReadRefusals        int `json:"BACK_BUFFER_READ_REFUSALS"`
	BackBufferGuardChecks         int `json:"BACK_BUFFER_GUARD_CHECKS"`
	BackBufferPixelChecks         int `json:"BACK_BUFFER_PIXEL_CHECKS"`
	RenderTargetCubeCreations     int `json:"RENDER_TARGET_CUBE_CREATIONS"`
	RenderTargetCubeRefusals      int `json:"RENDER_TARGET_CUBE_REFUSALS"`
	RenderTargetCubeBinds         int `json:"RENDER_TARGET_CUBE_BINDS"`
	RenderTargetCubeBindRefusals  int `json:"RENDER_TARGET_CUBE_BIND_REFUSALS"`
	RenderTargetBindingChecks     int `json:"RENDER_TARGET_BINDING_CHECKS"`
	OwnedDeviceCreations          int `json:"OWNED_DEVICE_CREATIONS"`
	OwnedDeviceRefusals           int `json:"OWNED_DEVICE_REFUSALS"`
	OwnedDeviceDisposalChecks     int `json:"OWNED_DEVICE_DISPOSAL_CHECKS"`
	// The adapter slice. CNA enumerates adapters through a callback-scoped
	// device, so every one of these is reachable only from inside LoadContent.
	AdapterCycles                int `json:"ADAPTER_CYCLES"`
	AdapterEnumerations          int `json:"ADAPTER_ENUMERATIONS"`
	AdapterSnapshotChecks        int `json:"ADAPTER_SNAPSHOT_CHECKS"`
	AdapterDeviceAdapterChecks   int `json:"ADAPTER_DEVICE_ADAPTER_CHECKS"`
	AdapterProfileChecks         int `json:"ADAPTER_PROFILE_CHECKS"`
	AdapterFormatQueries         int `json:"ADAPTER_FORMAT_QUERIES"`
	AdapterPreferenceChecks      int `json:"ADAPTER_PREFERENCE_CHECKS"`
	AdapterOutsideCallbackChecks int `json:"ADAPTER_OUTSIDE_CALLBACK_CHECKS"`
	// The index-buffer slice. RoundTrips is the load-bearing one: it writes
	// indices to the live buffer and reads them back FROM IT, so a projection
	// that kept a managed copy would compare its own input.
	IndexBufferCycles            int `json:"INDEX_BUFFER_CYCLES"`
	IndexBufferCreations         int `json:"INDEX_BUFFER_CREATIONS"`
	IndexBufferDescriptionChecks int `json:"INDEX_BUFFER_DESCRIPTION_CHECKS"`
	IndexBufferRoundTrips        int `json:"INDEX_BUFFER_ROUND_TRIPS"`
	IndexBufferReadbackRefusals  int `json:"INDEX_BUFFER_READBACK_REFUSALS"`
	IndexBufferWindowRoundTrips  int `json:"INDEX_BUFFER_WINDOW_ROUND_TRIPS"`
	IndexBufferRefusals          int `json:"INDEX_BUFFER_REFUSALS"`
	IndexBufferWriteOnlyChecks   int `json:"INDEX_BUFFER_WRITE_ONLY_CHECKS"`
	IndexBufferDisposalChecks    int `json:"INDEX_BUFFER_DISPOSAL_CHECKS"`
	// The vertex-buffer slice. The declaration's CNA handle is created on the
	// first buffer that needs one, so DeclarationHandles proves the lazy
	// creation actually happened rather than being skipped.
	VertexBufferCycles            int `json:"VERTEX_BUFFER_CYCLES"`
	VertexBufferCreations         int `json:"VERTEX_BUFFER_CREATIONS"`
	VertexBufferDeclarationShares int `json:"VERTEX_BUFFER_DECLARATION_SHARES"`
	VertexBufferDescriptionChecks int `json:"VERTEX_BUFFER_DESCRIPTION_CHECKS"`
	VertexBufferRoundTrips        int `json:"VERTEX_BUFFER_ROUND_TRIPS"`
	VertexBufferReadbackRefusals  int `json:"VERTEX_BUFFER_READBACK_REFUSALS"`
	VertexBufferOffsetRoundTrips  int `json:"VERTEX_BUFFER_OFFSET_ROUND_TRIPS"`
	VertexBufferFromTypeChecks    int `json:"VERTEX_BUFFER_FROM_TYPE_CHECKS"`
	VertexBufferRefusals          int `json:"VERTEX_BUFFER_REFUSALS"`
	VertexBufferStrideChecks      int `json:"VERTEX_BUFFER_STRIDE_CHECKS"`
	VertexBufferBindChecks        int `json:"VERTEX_BUFFER_BIND_CHECKS"`
	VertexBufferIndexBindChecks   int `json:"VERTEX_BUFFER_INDEX_BIND_CHECKS"`
	VertexBufferDraws             int `json:"VERTEX_BUFFER_DRAWS"`
	VertexBufferDrawRefusals      int `json:"VERTEX_BUFFER_DRAW_REFUSALS"`
	// Foundation 72. The draw revalidation: the control refusal BEFORE any
	// effect is applied, the effect itself, and what the draw does after.
	VertexBufferDrawRefusalsBeforeApply int `json:"VERTEX_BUFFER_DRAW_REFUSALS_BEFORE_APPLY"`
	VertexBufferEffectLoads             int `json:"VERTEX_BUFFER_EFFECT_LOADS"`
	VertexBufferEffectRefusals          int `json:"VERTEX_BUFFER_EFFECT_REFUSALS"`
	VertexBufferEffectApplies           int `json:"VERTEX_BUFFER_EFFECT_APPLIES"`
	VertexBufferEffectApplyRefusals     int `json:"VERTEX_BUFFER_EFFECT_APPLY_REFUSALS"`
	VertexBufferEffectDisposalChecks    int `json:"VERTEX_BUFFER_EFFECT_DISPOSAL_CHECKS"`
	// Foundation 79. The BasicEffect slice. Creations and creation refusals are
	// counted apart for the reason every other native creation's are: a run
	// that only ever refused must not read as a run that made an effect.
	//
	// The four ROUND TRIPS are the four properties the reference backs with an
	// EffectParameter and this projection backs with CNA -- SpecularColor,
	// SpecularPower, FogColor and Texture -- and they are the only members whose
	// value crosses in both directions. LIGHT_CHECKS covers the three published
	// lights: their object identity, their write-through, and the disabled-light
	// divergence. APPLIES counts a pass applied through the effect's OWN
	// technique, which is the only path that reaches BasicEffect's OnApply.
	BasicEffectCreations         int `json:"BASIC_EFFECT_CREATIONS"`
	BasicEffectCreationRefusals  int `json:"BASIC_EFFECT_CREATION_REFUSALS"`
	BasicEffectRoundTrips        int `json:"BASIC_EFFECT_ROUND_TRIPS"`
	BasicEffectRoundTripRefusals int `json:"BASIC_EFFECT_ROUND_TRIP_REFUSALS"`
	BasicEffectLightChecks       int `json:"BASIC_EFFECT_LIGHT_CHECKS"`
	BasicEffectApplies           int `json:"BASIC_EFFECT_APPLIES"`
	BasicEffectApplyRefusals     int `json:"BASIC_EFFECT_APPLY_REFUSALS"`
	BasicEffectDraws             int `json:"BASIC_EFFECT_DRAWS"`
	BasicEffectDrawRefusals      int `json:"BASIC_EFFECT_DRAW_REFUSALS"`
	// The control for the draw above, taken immediately before the BasicEffect
	// exists and after everything else in the scenario has run. Without it, a
	// draw that succeeded after the apply would not be evidence that the apply
	// is what made it succeed -- the six user-primitive draws sit between the
	// scenario's own control and this point.
	BasicEffectControlDraws        int `json:"BASIC_EFFECT_CONTROL_DRAWS"`
	BasicEffectControlDrawRefusals int `json:"BASIC_EFFECT_CONTROL_DRAW_REFUSALS"`
	BasicEffectCloneChecks         int `json:"BASIC_EFFECT_CLONE_CHECKS"`
	BasicEffectDisposalChecks      int `json:"BASIC_EFFECT_DISPOSAL_CHECKS"`
	// Foundation 80. AlphaTestEffect, DualTextureEffect and EffectMaterial.
	// Creations and refusals are counted apart for the reason every other
	// native creation's are, and the round trips cover the properties the
	// reference backs with an EffectParameter -- two for AlphaTestEffect and
	// three for DualTextureEffect, whose second texture layer reaches the same
	// route at index one.
	UnlitEffectCreations        int `json:"UNLIT_EFFECT_CREATIONS"`
	UnlitEffectCreationRefusals int `json:"UNLIT_EFFECT_CREATION_REFUSALS"`
	UnlitEffectRoundTrips       int `json:"UNLIT_EFFECT_ROUND_TRIPS"`
	UnlitEffectApplies          int `json:"UNLIT_EFFECT_APPLIES"`
	UnlitEffectApplyRefusals    int `json:"UNLIT_EFFECT_APPLY_REFUSALS"`
	UnlitEffectCloneChecks      int `json:"UNLIT_EFFECT_CLONE_CHECKS"`
	UnlitEffectDisposalChecks   int `json:"UNLIT_EFFECT_DISPOSAL_CHECKS"`
	EffectMaterialCreations     int `json:"EFFECT_MATERIAL_CREATIONS"`
	EffectMaterialRefusals      int `json:"EFFECT_MATERIAL_REFUSALS"`
	EffectMaterialIdentityCheck int `json:"EFFECT_MATERIAL_IDENTITY_CHECKS"`
	// Foundation 82. The two root-type statics, which are the only projected
	// members that take a game handle for THREAD AFFINITY alone. Reads and
	// refusals are counted apart because a title asset that is not there is
	// CNA answering CNA_RESULT_IO, not a defect.
	FrameworkDispatcherUpdates  int `json:"FRAMEWORK_DISPATCHER_UPDATES"`
	TitleContainerReads         int `json:"TITLE_CONTAINER_READS"`
	TitleContainerReadRefusals  int `json:"TITLE_CONTAINER_READ_REFUSALS"`
	TitleContainerGuardChecks   int `json:"TITLE_CONTAINER_GUARD_CHECKS"`
	RootStaticOutsideGameChecks int `json:"ROOT_STATIC_OUTSIDE_GAME_CHECKS"`
	// Foundation 83. OcclusionQuery. Creations and refusals are counted apart
	// because CNA answers CNA_RESULT_NOT_SUPPORTED where the backend has no
	// query object, which is a renderer capability and not a defect -- the same
	// shape the volume-texture creation already has.
	OcclusionQueryCreations        int `json:"OCCLUSION_QUERY_CREATIONS"`
	OcclusionQueryCreationRefusals int `json:"OCCLUSION_QUERY_CREATION_REFUSALS"`
	OcclusionQueryPairs            int `json:"OCCLUSION_QUERY_PAIRS"`
	OcclusionQueryPairRefusals     int `json:"OCCLUSION_QUERY_PAIR_REFUSALS"`
	OcclusionQueryGuardChecks      int `json:"OCCLUSION_QUERY_GUARD_CHECKS"`
	OcclusionQueryCompletions      int `json:"OCCLUSION_QUERY_COMPLETIONS"`
	OcclusionQueryPendingChecks    int `json:"OCCLUSION_QUERY_PENDING_CHECKS"`
	OcclusionQueryDisposalChecks   int `json:"OCCLUSION_QUERY_DISPOSAL_CHECKS"`
	// What IsComplete answers INSIDE a pair, which is CNA's answer and not
	// XNA's. Counted in two buckets so the behaviour is visible either way.
	OcclusionQueryStaleResultChecks int `json:"OCCLUSION_QUERY_STALE_RESULT_CHECKS"`
	OcclusionQueryFreshResultChecks int `json:"OCCLUSION_QUERY_FRESH_RESULT_CHECKS"`
	// Foundation 87. SoundEffect and SoundEffectInstance. Playback IS available
	// on both qualified artifacts -- cna_audio_get_capabilities reports
	// is_playback_available=true -- so the creations are counted apart from the
	// refusals only because CNA documents NOT_SUPPORTED for a machine without
	// audio hardware and a run on such a machine must record rather than fail.
	SoundEffectCreations          int `json:"SOUND_EFFECT_CREATIONS"`
	SoundEffectCreationRefusals   int `json:"SOUND_EFFECT_CREATION_REFUSALS"`
	SoundEffectDescriptionChecks  int `json:"SOUND_EFFECT_DESCRIPTION_CHECKS"`
	SoundEffectGuardChecks        int `json:"SOUND_EFFECT_GUARD_CHECKS"`
	SoundEffectScalarChecks       int `json:"SOUND_EFFECT_SCALAR_CHECKS"`
	SoundEffectPlays              int `json:"SOUND_EFFECT_PLAYS"`
	SoundEffectPlayLimitChecks    int `json:"SOUND_EFFECT_PLAY_LIMIT_CHECKS"`
	SoundInstanceCreations        int `json:"SOUND_INSTANCE_CREATIONS"`
	SoundInstanceTransitions      int `json:"SOUND_INSTANCE_TRANSITIONS"`
	SoundInstanceScalarRoundTrips int `json:"SOUND_INSTANCE_SCALAR_ROUND_TRIPS"`
	SoundInstanceModeLatchChecks  int `json:"SOUND_INSTANCE_MODE_LATCH_CHECKS"`
	SoundInstanceApply3DChecks    int `json:"SOUND_INSTANCE_APPLY_3D_CHECKS"`
	SoundEffectDisposalChecks     int `json:"SOUND_EFFECT_DISPOSAL_CHECKS"`
	// Foundation 88. DynamicSoundEffectInstance and Microphone, which finish the
	// Audio namespace. The microphone counters cover ENUMERATION and
	// DESCRIPTION only: Start and GetData are projected and never called, so
	// there is deliberately no capture counter to report.
	DynamicInstanceCreations        int `json:"DYNAMIC_INSTANCE_CREATIONS"`
	DynamicInstanceRefusals         int `json:"DYNAMIC_INSTANCE_REFUSALS"`
	DynamicInstanceLoopChecks       int `json:"DYNAMIC_INSTANCE_LOOP_CHECKS"`
	DynamicInstanceSubmissions      int `json:"DYNAMIC_INSTANCE_SUBMISSIONS"`
	DynamicInstancePendingChecks    int `json:"DYNAMIC_INSTANCE_PENDING_CHECKS"`
	DynamicInstanceConversionChecks int `json:"DYNAMIC_INSTANCE_CONVERSION_CHECKS"`
	DynamicInstanceDisposalChecks   int `json:"DYNAMIC_INSTANCE_DISPOSAL_CHECKS"`
	MicrophoneEnumerations          int `json:"MICROPHONE_ENUMERATIONS"`
	MicrophonesFound                int `json:"MICROPHONES_FOUND"`
	MicrophoneDescriptionChecks     int `json:"MICROPHONE_DESCRIPTION_CHECKS"`
	MicrophoneGuardChecks           int `json:"MICROPHONE_GUARD_CHECKS"`
	MicrophoneCaptureCalls          int `json:"MICROPHONE_CAPTURE_CALLS"`
	// Foundation 89. The Input family. GAMEPAD_VIBRATION_CALLS is deliberately
	// separate from the reads: it is the only member here that DRIVES hardware
	// rather than sampling it, and it is always called with two zeros.
	GamePadCapabilityReads   int `json:"GAMEPAD_CAPABILITY_READS"`
	GamePadStateReads        int `json:"GAMEPAD_STATE_READS"`
	GamePadsConnected        int `json:"GAMEPADS_CONNECTED"`
	GamePadVibrationCalls    int `json:"GAMEPAD_VIBRATION_CALLS"`
	GamePadVibrationsApplied int `json:"GAMEPAD_VIBRATIONS_APPLIED"`
	MouseStateReads          int `json:"MOUSE_STATE_READS"`
	MouseHandleChecks        int `json:"MOUSE_HANDLE_CHECKS"`
	MousePositionWrites      int `json:"MOUSE_POSITION_WRITES"`
	TouchPanelManagedChecks  int `json:"TOUCH_PANEL_MANAGED_CHECKS"`
	TouchPanelNativeCalls    int `json:"TOUCH_PANEL_NATIVE_CALLS"`
	// Foundation 91. Storage. STORAGE_ROOT_CHECKS is the containment proof:
	// the slice does nothing until it has read back a root it recognises.
	StorageRootChecks      int `json:"STORAGE_ROOT_CHECKS"`
	StorageSelectorCycles  int `json:"STORAGE_SELECTOR_CYCLES"`
	StorageDeviceReads     int `json:"STORAGE_DEVICE_READS"`
	StorageContainerCycles int `json:"STORAGE_CONTAINER_CYCLES"`
	StorageFileWrites      int `json:"STORAGE_FILE_WRITES"`
	StorageFileReads       int `json:"STORAGE_FILE_READS"`
	StorageEnumerations    int `json:"STORAGE_ENUMERATIONS"`
	StorageDirectoryCycles int `json:"STORAGE_DIRECTORY_CYCLES"`
	StorageDisposalChecks  int `json:"STORAGE_DISPOSAL_CHECKS"`
	// Foundation 85. The first VERIFIED_PIXEL draw. Every counter here is over
	// the SOFTWARE artifact only, because HEADLESS has no back-buffer readback
	// and records a refusal instead -- which is why the refusal column exists.
	PixelDrawRefusals            int `json:"PIXEL_DRAW_REFUSALS"`
	PixelDrawWindingChecks       int `json:"PIXEL_DRAW_WINDING_CHECKS"`
	PixelDrawGeometryChecks      int `json:"PIXEL_DRAW_GEOMETRY_CHECKS"`
	PixelDrawMaterialChecks      int `json:"PIXEL_DRAW_MATERIAL_CHECKS"`
	PixelDrawAlphaChecks         int `json:"PIXEL_DRAW_ALPHA_CHECKS"`
	PixelDrawVertexColorHonoured int `json:"PIXEL_DRAW_VERTEX_COLOR_HONOURED"`
	PixelDrawVertexColorIgnored  int `json:"PIXEL_DRAW_VERTEX_COLOR_IGNORED"`
	PixelDrawLightingChecks      int `json:"PIXEL_DRAW_LIGHTING_CHECKS"`
	PixelDrawLightingIgnored     int `json:"PIXEL_DRAW_LIGHTING_IGNORED"`
	// Foundation 84. The two dynamic buffers. Creations and refusals are
	// counted apart because CNA_VertexBufferCreateInfo.dynamic is a renderer
	// capability like the query object is; the option counters are per
	// SetDataOptions value, because CNA documents Discard and NoOverwrite as
	// hints a family may implement differently and a refusal of one is not a
	// refusal of the upload.
	DynamicBufferCreations         int `json:"DYNAMIC_BUFFER_CREATIONS"`
	DynamicBufferCreationRefusals  int `json:"DYNAMIC_BUFFER_CREATION_REFUSALS"`
	DynamicBufferDescriptionChecks int `json:"DYNAMIC_BUFFER_DESCRIPTION_CHECKS"`
	DynamicBufferOptionUploads     int `json:"DYNAMIC_BUFFER_OPTION_UPLOADS"`
	DynamicBufferOptionRefusals    int `json:"DYNAMIC_BUFFER_OPTION_REFUSALS"`
	DynamicBufferRoundTrips        int `json:"DYNAMIC_BUFFER_ROUND_TRIPS"`
	DynamicBufferContentLostReads  int `json:"DYNAMIC_BUFFER_CONTENT_LOST_READS"`
	DynamicBufferLatchClears       int `json:"DYNAMIC_BUFFER_LATCH_CLEARS"`
	DynamicBufferGuardChecks       int `json:"DYNAMIC_BUFFER_GUARD_CHECKS"`
	DynamicBufferBindChecks        int `json:"DYNAMIC_BUFFER_BIND_CHECKS"`
	DynamicBufferDisposalChecks    int `json:"DYNAMIC_BUFFER_DISPOSAL_CHECKS"`
	// Foundation 81. EnvironmentMapEffect and SkinnedEffect, which close the
	// stock-effect family. The bone counters are separate because the copy is
	// the one ARRAY crossing in the family and CNA may refuse it where it
	// accepts the scalars.
	LitEffectCreations        int `json:"LIT_EFFECT_CREATIONS"`
	LitEffectCreationRefusals int `json:"LIT_EFFECT_CREATION_REFUSALS"`
	LitEffectLightChecks      int `json:"LIT_EFFECT_LIGHT_CHECKS"`
	LitEffectRoundTrips       int `json:"LIT_EFFECT_ROUND_TRIPS"`
	LitEffectBoneRoundTrips   int `json:"LIT_EFFECT_BONE_ROUND_TRIPS"`
	LitEffectBoneRefusals     int `json:"LIT_EFFECT_BONE_REFUSALS"`
	LitEffectApplies          int `json:"LIT_EFFECT_APPLIES"`
	LitEffectApplyRefusals    int `json:"LIT_EFFECT_APPLY_REFUSALS"`
	LitEffectCloneChecks      int `json:"LIT_EFFECT_CLONE_CHECKS"`
	LitEffectDisposalChecks   int `json:"LIT_EFFECT_DISPOSAL_CHECKS"`
	LitEffectRetentionChecks  int `json:"LIT_EFFECT_RETENTION_CHECKS"`
	LitEffectReleaseChecks    int `json:"LIT_EFFECT_RELEASE_CHECKS"`
	// The six user-primitive draws, and the four guards the projection makes
	// before CNA is reached.
	UserPrimitiveDraws          int `json:"USER_PRIMITIVE_DRAWS"`
	UserPrimitiveDrawRefusals   int `json:"USER_PRIMITIVE_DRAW_REFUSALS"`
	UserPrimitiveGuardChecks    int `json:"USER_PRIMITIVE_GUARD_CHECKS"`
	VertexBufferDrawGuardChecks int `json:"VERTEX_BUFFER_DRAW_GUARD_CHECKS"`
	VertexBufferUnbindChecks    int `json:"VERTEX_BUFFER_UNBIND_CHECKS"`
	VertexBufferDisposalChecks  int `json:"VERTEX_BUFFER_DISPOSAL_CHECKS"`
	CallbackErrorCycles         int `json:"CALLBACK_ERROR_CYCLES"`
	CallbackPanicCycles         int `json:"CALLBACK_PANIC_CYCLES"`
	WrongThreadChecks           int `json:"WRONG_THREAD_CHECKS"`
	OwnerThreadRetries          int `json:"OWNER_THREAD_RETRIES"`
	GCStressPoints              int `json:"GC_STRESS_POINTS"`
	NativeCrashes               int `json:"NATIVE_CRASHES"`
	ObservedUAF                 int `json:"OBSERVED_UAF"`
	ObservedDoubleFree          int `json:"OBSERVED_DOUBLE_FREE"`

	GameEventActivated   int `json:"GAME_EVENT_ACTIVATED_DELIVERIES"`
	GameEventDeactivated int `json:"GAME_EVENT_DEACTIVATED_DELIVERIES"`
	GameEventExiting     int `json:"GAME_EVENT_EXITING_DELIVERIES"`

	// The disposal counters are two different facts and are deliberately two
	// different counters. The native signal is CNA reporting native game
	// destruction from inside cna_game_destroy; the managed raise is
	// Game::Disposed, which the reference raises from Dispose(bool) and from
	// nowhere else. Foundation 39 stopped the first from driving the second.
	GameNativeDisposalSignals int `json:"GAME_NATIVE_DISPOSAL_SIGNALS"`
	GameDisposedDuringRun     int `json:"GAME_DISPOSED_RAISED_DURING_RUN"`
	GameDisposedByManagedCall int `json:"GAME_DISPOSED_RAISED_BY_MANAGED_DISPOSE"`
	GameDisposedRepeatChecks  int `json:"GAME_DISPOSED_REPEAT_CHECKS"`
	GameDisposeAfterRunCycles int `json:"GAME_DISPOSE_AFTER_RUN_CYCLES"`
	GameEventOrderChecks      int `json:"GAME_EVENT_ORDER_CHECKS"`
	GameEventRemovalChecks    int `json:"GAME_EVENT_REMOVAL_CHECKS"`
	GameEventOwnerThreadHits  int `json:"GAME_EVENT_OWNER_THREAD_CHECKS"`
	GameEventRerunCycles      int `json:"GAME_EVENT_RERUN_CYCLES"`
	GameEventPostRunChecks    int `json:"GAME_EVENT_POST_RUN_CHECKS"`

	// Foundation 50. Every Draw overload the profile declares, submitted to a
	// live native SpriteBatch inside a real draw callback, plus the two guards
	// InternalDraw applies before it queues anything.
	SpriteDrawCycles             int `json:"SPRITE_DRAW_CYCLES"`
	SpriteDrawScaledSubmits      int `json:"SPRITE_DRAW_SCALED_SUBMISSIONS"`
	SpriteDrawDestinationSubmits int `json:"SPRITE_DRAW_DESTINATION_SUBMISSIONS"`
	SpriteDrawNullTextureChecks  int `json:"SPRITE_DRAW_NULL_TEXTURE_CHECKS"`
	SpriteDrawOutsidePairChecks  int `json:"SPRITE_DRAW_OUTSIDE_PAIR_CHECKS"`
	SpriteDrawPairGuardChecks    int `json:"SPRITE_DRAW_PAIR_GUARD_CHECKS"`
	SpriteDrawBoundsChecks       int `json:"SPRITE_DRAW_TEXTURE_BOUNDS_CHECKS"`

	// Foundation 51. GraphicsDevice's render state, round-tripped through the
	// live device, plus the two masked Clear overloads and Present.
	DeviceStateCycles                 int `json:"DEVICE_STATE_CYCLES"`
	DeviceStateRoundTrips             int `json:"DEVICE_STATE_ROUND_TRIPS"`
	DeviceStateObjectRefusals         int `json:"DEVICE_STATE_OBJECT_REFUSALS"`
	DeviceStateObjectBinds            int `json:"DEVICE_STATE_OBJECT_BINDS"`
	SpriteBatchStateBegins            int `json:"SPRITE_BATCH_STATE_BEGINS"`
	DeviceCollectionIdentityChecks    int `json:"DEVICE_COLLECTION_IDENTITY_CHECKS"`
	DeviceCollectionRangeChecks       int `json:"DEVICE_COLLECTION_RANGE_CHECKS"`
	DeviceCollectionTextureRoundTrips int `json:"DEVICE_COLLECTION_TEXTURE_ROUND_TRIPS"`
	DeviceCollectionSamplerRoundTrips int `json:"DEVICE_COLLECTION_SAMPLER_ROUND_TRIPS"`
	DeviceEventSubscriptions          int `json:"DEVICE_EVENT_SUBSCRIPTIONS"`
	DeviceEventRegistrationChecks     int `json:"DEVICE_EVENT_REGISTRATION_CHECKS"`
	DeviceStateReadOnlyChecks         int `json:"DEVICE_STATE_READ_ONLY_CHECKS"`
	DeviceStateClearCalls             int `json:"DEVICE_STATE_CLEAR_CALLS"`
	DeviceStateClearRefusals          int `json:"DEVICE_STATE_CLEAR_REFUSALS"`
	DeviceStatePresentCalls           int `json:"DEVICE_STATE_PRESENT_CALLS"`
	DeviceStateStaleChecks            int `json:"DEVICE_STATE_STALE_CHECKS"`
	DeviceStateWrongThreadHits        int `json:"DEVICE_STATE_WRONG_THREAD_CHECKS"`

	// Foundation 52. The device's display mode, and an EMPTY texture created
	// from its dimensions and format rather than decoded from bytes.
	DeviceStateDisplayModeChecks int `json:"DEVICE_STATE_DISPLAY_MODE_CHECKS"`
	DeviceStateTextureCreations  int `json:"DEVICE_STATE_TEXTURE_CREATIONS"`
	DeviceStateTextureRefusals   int `json:"DEVICE_STATE_TEXTURE_REFUSALS"`

	// Foundation 53. A texture encoded to PNG and to JPEG, and decoded back at
	// a requested size through both zoom modes.
	DeviceStateEncodeChecks     int `json:"DEVICE_STATE_TEXTURE_ENCODE_CHECKS"`
	DeviceStateDecodeSizeChecks int `json:"DEVICE_STATE_TEXTURE_DECODE_SIZE_CHECKS"`
	DeviceStateEncodeRefusals   int `json:"DEVICE_STATE_TEXTURE_ENCODE_REFUSALS"`

	// Foundation 54. Typed transfers through the generic-method projection.
	DeviceStateTransferRoundTrips int `json:"DEVICE_STATE_TEXTURE_TRANSFER_ROUND_TRIPS"`
	DeviceStateTransferRefusals   int `json:"DEVICE_STATE_TEXTURE_TRANSFER_REFUSALS"`

	FrameHookOverrideCycles  int `json:"FRAME_HOOK_OVERRIDE_CYCLES"`
	FrameHookBeginRunHits    int `json:"FRAME_HOOK_BEGIN_RUN_DELIVERIES"`
	FrameHookEndRunHits      int `json:"FRAME_HOOK_END_RUN_DELIVERIES"`
	FrameHookBeginDrawHits   int `json:"FRAME_HOOK_BEGIN_DRAW_DELIVERIES"`
	FrameHookEndDrawHits     int `json:"FRAME_HOOK_END_DRAW_DELIVERIES"`
	FrameHookRefusedFrames   int `json:"FRAME_HOOK_REFUSED_FRAMES"`
	FrameHookAdmittedFrames  int `json:"FRAME_HOOK_ADMITTED_FRAMES"`
	FrameHookEndDrawExpected int `json:"FRAME_HOOK_END_DRAW_EXPECTED"`
	FrameHookSkipChecks      int `json:"FRAME_HOOK_REFUSED_FRAME_SKIP_CHECKS"`
	FrameHookBaseCallChecks  int `json:"FRAME_HOOK_EXPLICIT_BASE_CALL_CHECKS"`
	FrameHookOrderChecks     int `json:"FRAME_HOOK_ORDER_CHECKS"`
	FrameHookSubsetCycles    int `json:"FRAME_HOOK_SUBSET_CYCLES"`
	FrameHookUninstalledHits int `json:"FRAME_HOOK_UNINSTALLED_DELIVERIES"`

	TimingCycles            int `json:"GAME_TIMING_CYCLES"`
	TimingSettersApplied    int `json:"GAME_TIMING_SETTERS_APPLIED"`
	TimingWrongThreadChecks int `json:"GAME_TIMING_WRONG_THREAD_CHECKS"`
	TimingRangeChecks       int `json:"GAME_TIMING_RANGE_CHECKS"`
	TimingCreatedWithConfig int `json:"GAME_TIMING_CREATED_WITH_CONFIGURED_STEP"`

	WindowCycles              int `json:"GAME_WINDOW_CYCLES"`
	WindowIdentityChecks      int `json:"GAME_WINDOW_IDENTITY_CHECKS"`
	WindowGuardedFallbacks    int `json:"GAME_WINDOW_GUARDED_FALLBACK_CHECKS"`
	WindowUnguardedFailures   int `json:"GAME_WINDOW_UNGUARDED_FAILURE_CHECKS"`
	WindowLiveReads           int `json:"GAME_WINDOW_LIVE_READ_CHECKS"`
	WindowTitleSuppressions   int `json:"GAME_WINDOW_TITLE_SUPPRESSION_CHECKS"`
	WindowWrongThreadChecks   int `json:"GAME_WINDOW_WRONG_THREAD_CHECKS"`
	WindowScreenDeviceChanges int `json:"GAME_WINDOW_SCREEN_DEVICE_CHANGE_CYCLES"`
	WindowResizeRoundTrips    int `json:"GAME_WINDOW_RESIZE_ROUND_TRIPS"`
	// Whether the live window reported a positive client size. HEADLESS does
	// not, and that is a renderer fact rather than a binding one.
	WindowPositiveClientBounds int `json:"GAME_WINDOW_POSITIVE_CLIENT_BOUNDS"`
	// The three canonical window signals. HEADLESS never resizes, rotates or
	// changes screen, so these are expected to stay zero in this environment
	// and are recorded rather than asserted -- exactly as
	// GAME_EVENT_DEACTIVATED_DELIVERIES is.
	WindowEventClientSize   int `json:"GAME_WINDOW_EVENT_CLIENT_SIZE_DELIVERIES"`
	WindowEventOrientation  int `json:"GAME_WINDOW_EVENT_ORIENTATION_DELIVERIES"`
	WindowEventScreenDevice int `json:"GAME_WINDOW_EVENT_SCREEN_DEVICE_NAME_DELIVERIES"`

	FrameStepCycles            int `json:"GAME_FRAME_STEP_CYCLES"`
	FrameStepTicks             int `json:"GAME_FRAME_STEP_TICKS"`
	FrameStepRunOneFrames      int `json:"GAME_FRAME_STEP_RUN_ONE_FRAMES"`
	FrameStepInitializations   int `json:"GAME_FRAME_STEP_INITIALIZATIONS"`
	FrameStepTickInitChecks    int `json:"GAME_FRAME_STEP_TICK_DOES_NOT_INITIALIZE_CHECKS"`
	FrameStepUpdates           int `json:"GAME_FRAME_STEP_UPDATE_DELIVERIES"`
	FrameStepDraws             int `json:"GAME_FRAME_STEP_DRAW_DELIVERIES"`
	FrameStepSuppressChecks    int `json:"GAME_FRAME_STEP_SUPPRESS_DRAW_CHECKS"`
	FrameStepWrongThreadChecks int `json:"GAME_FRAME_STEP_WRONG_THREAD_CHECKS"`
	FrameStepCallbackRefusals  int `json:"GAME_FRAME_STEP_CALLBACK_REFUSAL_CHECKS"`
	FrameStepExitChecks        int `json:"GAME_FRAME_STEP_EXIT_CHECKS"`
	FrameStepSessionChecks     int `json:"GAME_FRAME_STEP_SESSION_LIFETIME_CHECKS"`
	FrameStepDisposeChecks     int `json:"GAME_FRAME_STEP_DISPOSE_CHECKS"`
	FrameStepRecreationChecks  int `json:"GAME_FRAME_STEP_RECREATION_CHECKS"`
	FrameStepRunAfterStepCycle int `json:"GAME_FRAME_STEP_RUN_ADOPTS_SESSION_CYCLES"`

	ManagerCycles           int `json:"GRAPHICS_MANAGER_CYCLES"`
	ManagerDefaultChecks    int `json:"GRAPHICS_MANAGER_DEFAULT_CHECKS"`
	ManagerSettersApplied   int `json:"GRAPHICS_MANAGER_SETTERS_APPLIED"`
	ManagerCrossPackageSets int `json:"GRAPHICS_MANAGER_CROSS_PACKAGE_SETTERS_APPLIED"`
	ManagerRangeChecks      int `json:"GRAPHICS_MANAGER_RANGE_CHECKS"`
	ManagerApplyChanges     int `json:"GRAPHICS_MANAGER_APPLY_CHANGES"`
	ManagerToggleChecks     int `json:"GRAPHICS_MANAGER_TOGGLE_FULL_SCREEN_CHECKS"`
	ManagerWrongThreadCheck int `json:"GRAPHICS_MANAGER_WRONG_THREAD_CHECKS"`

	ManagerServiceChecks       int `json:"GRAPHICS_MANAGER_SERVICE_REGISTRATION_CHECKS"`
	ManagerDuplicateChecks     int `json:"GRAPHICS_MANAGER_DUPLICATE_REGISTRATION_CHECKS"`
	ManagerGameDeviceChecks    int `json:"GRAPHICS_MANAGER_GAME_GRAPHICS_DEVICE_CHECKS"`
	ManagerDrawableChecks      int `json:"GRAPHICS_MANAGER_DRAWABLE_COMPONENT_CHECKS"`
	ManagerEventRaiseChecks    int `json:"GRAPHICS_MANAGER_EVENT_RAISE_CHECKS"`
	ManagerServiceRemovalCheck int `json:"GRAPHICS_MANAGER_SERVICE_REMOVAL_CHECKS"`
	// The five canonical manager signals. HEADLESS creates its device once and
	// never loses or resets it, so the reset and disposing counters are
	// expected to stay at zero and are recorded rather than asserted.
	ManagerSignalDeviceCreated   int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_CREATED_DELIVERIES"`
	ManagerSignalDeviceReset     int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_RESET_DELIVERIES"`
	ManagerSignalDeviceResetting int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_RESETTING_DELIVERIES"`
	ManagerSignalDeviceDisposing int `json:"GRAPHICS_MANAGER_SIGNAL_DEVICE_DISPOSING_DELIVERIES"`
	ManagerSignalDisposed        int `json:"GRAPHICS_MANAGER_SIGNAL_DISPOSED_DELIVERIES"`
}

type stressReport struct {
	SchemaVersion         int      `json:"schema_version"`
	Isolation             string   `json:"isolation"`
	GoRaceStatus          string   `json:"GO_RACE_STATUS"`
	NativeSanitizerStatus string   `json:"NATIVE_SANITIZER_STATUS"`
	NativeLibrarySHA256   string   `json:"native_library_sha256,omitempty"`
	Counters              counters `json:"counters"`
}

type stressGame struct {
	scenario string
	index    int
	// effectRoots are the temporary content roots the stock-effect loads used.
	// They outlive the load because CNA caches the asset by normalized key and
	// may consult the file again; they are removed when the scenario ends.
	effectRoots []string
	manager     *framework.GraphicsDeviceManager
	device      *graphics.GraphicsDevice
	data        []byte
	result      counters

	// eventOrder records every native game signal in delivery order, and
	// removedRan records whether a handler removed before Run ever fired.
	eventOrder      []string
	removedRan      bool
	ownerGoroutine  string
	eventGoroutines map[string]bool

	// The sprite-draw scenario's two live objects, created in LoadContent and
	// used from inside Draw, which is the only moment CNA has a render pass.
	spriteTexture *graphics.Texture2D
	spriteBatch   *graphics.SpriteBatch

	// runtime is captured from inside the first callback, which is the only
	// place it is reachable, and it survives the run. It is how the native
	// disposal signal is observed at all now that it raises no public event.
	runtime *interop.Runtime
	// disposedRaises counts public Game.Disposed raises. It must stay zero for
	// the whole run: the native signal no longer drives the event.
	disposedRaises int
}

var callbackSentinel = errors.New("native stress callback sentinel")

func main() {
	child := flag.String("child", "", "internal isolated scenario")
	index := flag.Int("index", 0, "internal scenario index")
	output := flag.String("output", "docs/generated/native-stress-report.json", "parent JSON report; empty disables writing")
	raceStatus := flag.String("race-status", "NOT_RUN", "PASS only when this invocation was built with -race")
	flag.Parse()
	if *child != "" {
		if err := runChild(*child, *index); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	result, err := runParent()
	if writeErr := writeStressReport(*output, *raceStatus, result); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(2)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeStressReport(path, raceStatus string, result counters) error {
	if path == "" {
		return nil
	}
	if raceStatus != "PASS" && raceStatus != "NOT_RUN" {
		return fmt.Errorf("invalid race status %q", raceStatus)
	}
	// The artifact identity, by CONTENT. Foundation 58 runs this against TWO
	// qualified artifacts -- a HEADLESS one and a SOFTWARE one -- and their
	// counters legitimately differ: only the software renderer can read a
	// render target's colour attachment back to the CPU. A report that did not
	// say which artifact produced it could not be read at all.
	report := stressReport{
		SchemaVersion:         1,
		Isolation:             "one native Game generation per subprocess",
		GoRaceStatus:          raceStatus,
		NativeSanitizerStatus: "NOT_RUN",
		NativeLibrarySHA256:   nativeLibraryDigest(),
		Counters:              result,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// nativeLibraryDigest is the SHA-256 of the artifact CNA_NATIVE_LIBRARY names.
// It is empty when the variable is unset, which is the case where the platform
// loader chose the library and CNA-Go cannot say which one it found.
func nativeLibraryDigest() string {
	path := os.Getenv("CNA_NATIVE_LIBRARY")
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func runParent() (counters, error) {
	executable, err := os.Executable()
	if err != nil {
		return counters{}, err
	}
	var total counters
	for _, scenario := range []string{"success", "callback-error", "callback-panic", "event-rerun", "frame-hook-override", "frame-hook-subset", "timing", "window", "frame-step", "frame-step-run", "graphics-manager", "sprite-draw", "device-state", "render-target", "content", "index-buffer", "vertex-buffer", "adapter", "sprite-font", "texture-volume", "presentation"} {
		for index := 0; index < 20; index++ {
			command := exec.Command(executable, "--child", scenario, "--index", fmt.Sprint(index))
			command.Env = os.Environ()
			output, runErr := command.CombinedOutput()
			if runErr != nil {
				total.NativeCrashes++
				return total, fmt.Errorf("isolated %s cycle %d failed: %w\n%s", scenario, index, runErr, output)
			}
			var one counters
			if err := decodeLastJSONLine(output, &one); err != nil {
				return total, fmt.Errorf("decode isolated %s cycle %d: %w: %q", scenario, index, err, output)
			}
			addCounters(&total, one)
		}
	}
	if total.GameCycles < 20 || total.GameRecreationCycles < 20 || total.TextureCycles < 20 || total.SpriteBatchCycles < 20 || total.CallbackErrorCycles < 20 || total.CallbackPanicCycles < 20 {
		return total, errors.New("native stress minimum was not met")
	}
	// One clean run per success cycle delivers exactly one Activated, one
	// Exiting and one Disposed, each proved in order and on the owner
	// goroutine. Deactivated has no minimum: HEADLESS cannot produce a focus
	// transition away from the game, and the counter records that honestly by
	// staying at zero.
	if total.GameEventActivated < 20 || total.GameEventExiting < 20 || total.GameNativeDisposalSignals < 20 {
		return total, errors.New("native game-event delivery minimum was not met")
	}
	// Twenty sprite-draw cycles, each submitting five position commands and
	// four destination commands through a live native SpriteBatch. The two
	// families are required SEPARATELY: a projection that sent every overload
	// down one route would still submit, and would place three of the seven
	// wrong.
	if total.SpriteDrawCycles < 20 || total.SpriteDrawScaledSubmits < 100 || total.SpriteDrawDestinationSubmits < 80 {
		return total, errors.New("native sprite-draw submission minimum was not met")
	}
	if total.SpriteDrawNullTextureChecks < 20 || total.SpriteDrawOutsidePairChecks < 40 ||
		total.SpriteDrawPairGuardChecks < 40 || total.SpriteDrawBoundsChecks < 20 {
		return total, errors.New("a sprite-draw guard or bounds proof did not run in every cycle")
	}
	// Five round trips per cycle -- blend factor, multisample mask, reference
	// stencil, scissor rectangle and viewport -- each written to the live
	// device and read back from it. A getter that answered from a managed cache
	// would pass these and would be a second source of truth, so the read is
	// required to come back from the device that was written.
	if total.DeviceStateCycles < 20 || total.DeviceStateRoundTrips < 100 {
		return total, errors.New("native device-state round-trip minimum was not met")
	}
	// Two display-mode checks and three texture creations per cycle, plus the
	// two refusals the projection makes before it reaches CNA.
	if total.DeviceStateDisplayModeChecks < 40 || total.DeviceStateTextureCreations < 60 ||
		total.DeviceStateTextureRefusals < 40 {
		return total, errors.New("a device-state display-mode or texture proof did not run in every cycle")
	}
	// Two encodes and two sized decodes per cycle, plus the one refusal the
	// projection makes before it reaches CNA.
	if total.DeviceStateEncodeChecks < 40 || total.DeviceStateDecodeSizeChecks < 40 ||
		total.DeviceStateEncodeRefusals < 20 {
		return total, errors.New("a texture encode or sized-decode proof did not run in every cycle")
	}
	// Three typed round trips per cycle -- a full-surface Color transfer, a
	// windowed one, and a rectangle one -- each written to the live texture and
	// read back from it, plus the two refusals the projection makes itself.
	if total.DeviceStateTransferRoundTrips < 60 || total.DeviceStateTransferRefusals < 40 {
		return total, errors.New("a texture transfer proof did not run in every cycle")
	}
	if total.DeviceStateReadOnlyChecks < 60 || total.DeviceStateClearCalls < 40 ||
		total.DeviceStateClearRefusals < 20 || total.DeviceStatePresentCalls < 20 ||
		total.DeviceStateStaleChecks < 20 || total.DeviceStateWrongThreadHits < 20 {
		return total, errors.New("a device-state proof did not run in every cycle")
	}
	// The adapter slice, per cycle: one enumeration, one snapshot checked, the
	// device's own adapter, one profile query, two format queries, the
	// preference pair round-tripped and the snapshot re-read.
	if total.AdapterCycles < 20 || total.AdapterEnumerations < 20 || total.AdapterSnapshotChecks < 20 ||
		total.AdapterDeviceAdapterChecks < 20 || total.AdapterProfileChecks < 20 ||
		total.AdapterFormatQueries < 40 || total.AdapterPreferenceChecks < 20 ||
		total.AdapterOutsideCallbackChecks < 20 {
		return total, errors.New("an adapter proof did not run in every cycle")
	}

	// The vertex-buffer slice, per cycle: one buffer created from a
	// declaration and one from a consumer's own IVertexType, the shared
	// declaration handle proved, and the two guards the projection makes
	// itself exercised.
	if total.VertexBufferCycles < 20 || total.VertexBufferCreations < 20 ||
		total.VertexBufferDeclarationShares < 20 || total.VertexBufferDescriptionChecks < 20 ||
		total.VertexBufferFromTypeChecks < 20 || total.VertexBufferRefusals < 20 ||
		total.VertexBufferStrideChecks < 20 || total.VertexBufferDisposalChecks < 20 ||
		total.VertexBufferBindChecks < 20 || total.VertexBufferIndexBindChecks < 20 ||
		total.VertexBufferDrawGuardChecks < 20 || total.VertexBufferUnbindChecks < 20 {
		return total, errors.New("a vertex-buffer proof did not run in every cycle")
	}
	// Exactly one of the two draw outcomes per cycle.
	if total.VertexBufferDraws+total.VertexBufferDrawRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("vertex-buffer draws %d and refusals %d do not account for %d cycles",
			total.VertexBufferDraws, total.VertexBufferDrawRefusals, total.VertexBufferCycles)
	}
	if total.VertexBufferOffsetRoundTrips != total.VertexBufferRoundTrips {
		return total, fmt.Errorf("%d vertex round trips produced %d offset ones",
			total.VertexBufferRoundTrips, total.VertexBufferOffsetRoundTrips)
	}

	// The index-buffer slice, per cycle: two buffers created, the description
	// CNA applied checked on one, the projection's own refusals exercised, the
	// WriteOnly read refused and disposal proved.
	if total.IndexBufferCycles < 20 || total.IndexBufferCreations < 20 ||
		total.IndexBufferDescriptionChecks < 20 || total.IndexBufferRefusals < 20 ||
		total.IndexBufferWriteOnlyChecks < 20 || total.IndexBufferDisposalChecks < 20 {
		return total, errors.New("an index-buffer proof did not run in every cycle")
	}
	// Foundation 72. The draw revalidation, per cycle: the CONTROL refusal
	// before anything is applied, one effect outcome, one apply outcome when
	// the effect loaded, and one draw outcome.
	//
	// The control is the load-bearing one. A cycle whose pre-apply draw
	// SUCCEEDED would prove nothing about the effect, because a draw that works
	// either way is not evidence that applying the effect made it work -- so
	// this requires the refusal rather than merely accounting for it.
	if total.VertexBufferDrawRefusalsBeforeApply != total.VertexBufferCycles {
		return total, fmt.Errorf("%d of %d cycles refused the draw before any effect was applied; the control must refuse in every one",
			total.VertexBufferDrawRefusalsBeforeApply, total.VertexBufferCycles)
	}
	if total.VertexBufferEffectLoads+total.VertexBufferEffectRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("effect loads %d and refusals %d do not account for %d cycles",
			total.VertexBufferEffectLoads, total.VertexBufferEffectRefusals, total.VertexBufferCycles)
	}
	if total.VertexBufferEffectApplies+total.VertexBufferEffectApplyRefusals != total.VertexBufferEffectLoads ||
		total.VertexBufferEffectDisposalChecks != total.VertexBufferEffectLoads {
		return total, fmt.Errorf("%d effect loads produced %d applies, %d apply refusals and %d disposal checks",
			total.VertexBufferEffectLoads, total.VertexBufferEffectApplies,
			total.VertexBufferEffectApplyRefusals, total.VertexBufferEffectDisposalChecks)
	}
	// Foundation 79. The BasicEffect slice is accounted for the same way: one
	// outcome per cycle for the creation, and every check downstream of a
	// creation counted against the number of creations rather than reported as
	// a bare total. A run in which the constructor refused every time is a run
	// with no BasicEffect evidence, and it must read as one.
	if total.BasicEffectCreations+total.BasicEffectCreationRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("BasicEffect creations %d and refusals %d do not account for %d cycles",
			total.BasicEffectCreations, total.BasicEffectCreationRefusals, total.VertexBufferCycles)
	}
	if total.BasicEffectLightChecks != 3*total.BasicEffectCreations {
		return total, fmt.Errorf("%d light checks over %d BasicEffect creations, want three each",
			total.BasicEffectLightChecks, total.BasicEffectCreations)
	}
	if total.BasicEffectControlDraws+total.BasicEffectControlDrawRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("BasicEffect control draws %d and refusals %d do not account for %d cycles",
			total.BasicEffectControlDraws, total.BasicEffectControlDrawRefusals, total.VertexBufferCycles)
	}
	if total.BasicEffectDraws+total.BasicEffectDrawRefusals != total.BasicEffectApplies {
		return total, fmt.Errorf("%d BasicEffect applies produced %d draws and %d draw refusals",
			total.BasicEffectApplies, total.BasicEffectDraws, total.BasicEffectDrawRefusals)
	}
	// Foundation 80. Two creations per cycle -- AlphaTestEffect and
	// DualTextureEffect -- and every downstream check counted against them.
	if total.UnlitEffectCreations+total.UnlitEffectCreationRefusals != 2*total.VertexBufferCycles {
		return total, fmt.Errorf("unlit-effect creations %d and refusals %d do not account for two per %d cycles",
			total.UnlitEffectCreations, total.UnlitEffectCreationRefusals, total.VertexBufferCycles)
	}
	if total.UnlitEffectApplies+total.UnlitEffectApplyRefusals != total.UnlitEffectCreations ||
		2*total.UnlitEffectRoundTrips != total.UnlitEffectCreations ||
		2*total.UnlitEffectCloneChecks != total.UnlitEffectCreations ||
		2*total.UnlitEffectDisposalChecks != total.UnlitEffectCreations {
		return total, fmt.Errorf("%d unlit-effect creations produced %d applies, %d apply refusals, %d round trips, %d clone checks and %d disposal checks",
			total.UnlitEffectCreations, total.UnlitEffectApplies, total.UnlitEffectApplyRefusals,
			total.UnlitEffectRoundTrips, total.UnlitEffectCloneChecks, total.UnlitEffectDisposalChecks)
	}
	// Foundation 81. Two creations per cycle again, and the bone round trip is
	// accounted for once per SkinnedEffect.
	if total.LitEffectCreations+total.LitEffectCreationRefusals != 2*total.VertexBufferCycles {
		return total, fmt.Errorf("lit-effect creations %d and refusals %d do not account for two per %d cycles",
			total.LitEffectCreations, total.LitEffectCreationRefusals, total.VertexBufferCycles)
	}
	if total.LitEffectLightChecks != total.LitEffectCreations ||
		total.LitEffectApplies+total.LitEffectApplyRefusals != total.LitEffectCreations ||
		2*total.LitEffectRoundTrips != total.LitEffectCreations ||
		2*total.LitEffectCloneChecks != total.LitEffectCreations ||
		2*total.LitEffectDisposalChecks != total.LitEffectCreations ||
		2*total.LitEffectRetentionChecks != total.LitEffectCreations ||
		2*total.LitEffectReleaseChecks != total.LitEffectCreations ||
		2*(total.LitEffectBoneRoundTrips+total.LitEffectBoneRefusals) != total.LitEffectCreations {
		return total, fmt.Errorf("%d lit-effect creations produced %d light checks, %d applies, %d apply refusals, %d round trips, %d bone round trips, %d bone refusals, %d clone checks and %d disposal checks",
			total.LitEffectCreations, total.LitEffectLightChecks, total.LitEffectApplies,
			total.LitEffectApplyRefusals, total.LitEffectRoundTrips, total.LitEffectBoneRoundTrips,
			total.LitEffectBoneRefusals, total.LitEffectCloneChecks, total.LitEffectDisposalChecks)
	}
	// Foundation 83. One creation outcome per cycle, and every check downstream
	// counted against the creations that succeeded.
	if total.OcclusionQueryCreations+total.OcclusionQueryCreationRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("occlusion-query creations %d and refusals %d do not account for %d cycles",
			total.OcclusionQueryCreations, total.OcclusionQueryCreationRefusals, total.VertexBufferCycles)
	}
	if total.OcclusionQueryDisposalChecks != total.OcclusionQueryCreations {
		return total, fmt.Errorf("%d occlusion-query creations produced %d disposal checks",
			total.OcclusionQueryCreations, total.OcclusionQueryDisposalChecks)
	}
	if total.OcclusionQueryStaleResultChecks+total.OcclusionQueryFreshResultChecks != total.OcclusionQueryPairs/2 {
		return total, fmt.Errorf("%d stale and %d fresh inside-pair checks over %d pairs",
			total.OcclusionQueryStaleResultChecks, total.OcclusionQueryFreshResultChecks, total.OcclusionQueryPairs)
	}
	if total.OcclusionQueryPairs+2*total.OcclusionQueryPairRefusals != 2*total.OcclusionQueryCreations ||
		total.OcclusionQueryGuardChecks+total.OcclusionQueryPairRefusals != total.OcclusionQueryCreations ||
		total.OcclusionQueryCompletions+total.OcclusionQueryPendingChecks+total.OcclusionQueryPairRefusals != total.OcclusionQueryCreations {
		return total, fmt.Errorf("%d occlusion-query creations produced %d pairs, %d pair refusals, %d guard checks, %d completions and %d pending checks",
			total.OcclusionQueryCreations, total.OcclusionQueryPairs, total.OcclusionQueryPairRefusals,
			total.OcclusionQueryGuardChecks, total.OcclusionQueryCompletions, total.OcclusionQueryPendingChecks)
	}
	// Foundation 84. One creation outcome per cycle, and every check downstream
	// counted against the creations that succeeded. The option counters are NOT
	// pinned to a multiple, because CNA may refuse an individual SetDataOptions
	// value where it accepts the upload -- what is pinned is that every upload
	// had exactly one outcome.
	if total.DynamicBufferCreations+total.DynamicBufferCreationRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("dynamic-buffer creations %d and refusals %d do not account for %d cycles",
			total.DynamicBufferCreations, total.DynamicBufferCreationRefusals, total.VertexBufferCycles)
	}
	if total.DynamicBufferDescriptionChecks != total.DynamicBufferCreations ||
		total.DynamicBufferGuardChecks != total.DynamicBufferCreations ||
		total.DynamicBufferBindChecks != total.DynamicBufferCreations ||
		total.DynamicBufferDisposalChecks != total.DynamicBufferCreations {
		return total, fmt.Errorf("%d dynamic-buffer creations produced %d description, %d guard, %d bind and %d disposal checks",
			total.DynamicBufferCreations, total.DynamicBufferDescriptionChecks,
			total.DynamicBufferGuardChecks, total.DynamicBufferBindChecks, total.DynamicBufferDisposalChecks)
	}
	// Eight uploads are attempted per creation -- three named vertex options,
	// two undefined vertex ones, two index ones and one offset -- and each has
	// exactly one outcome. A refusal of any of them fails the run rather than
	// being counted, so the refusal column is here to be zero and say so.
	if total.DynamicBufferOptionUploads+total.DynamicBufferOptionRefusals != 8*total.DynamicBufferCreations {
		return total, fmt.Errorf("%d dynamic uploads and %d refusals do not account for the eight per %d creations",
			total.DynamicBufferOptionUploads, total.DynamicBufferOptionRefusals, total.DynamicBufferCreations)
	}
	if total.DynamicBufferRoundTrips > total.DynamicBufferOptionUploads {
		return total, fmt.Errorf("%d dynamic round trips over %d uploads",
			total.DynamicBufferRoundTrips, total.DynamicBufferOptionUploads)
	}
	if total.DynamicBufferContentLostReads > 2*total.DynamicBufferCreations ||
		total.DynamicBufferLatchClears > total.DynamicBufferCreations {
		return total, fmt.Errorf("%d content-lost reads and %d latch clears over %d creations",
			total.DynamicBufferContentLostReads, total.DynamicBufferLatchClears, total.DynamicBufferCreations)
	}
	// Foundation 87. One creation outcome per cycle, and every check downstream
	// counted against the creations that succeeded. SOUND_EFFECT_PLAYS and
	// SOUND_EFFECT_PLAY_LIMIT_CHECKS are the two halves of Play's bool, which
	// is false ONLY when the voice limit was hit.
	if total.SoundEffectCreations+total.SoundEffectCreationRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("sound-effect creations %d and refusals %d do not account for %d cycles",
			total.SoundEffectCreations, total.SoundEffectCreationRefusals, total.VertexBufferCycles)
	}
	if total.SoundEffectDescriptionChecks != total.SoundEffectCreations ||
		total.SoundEffectGuardChecks != total.SoundEffectCreations ||
		total.SoundEffectScalarChecks != total.SoundEffectCreations ||
		total.SoundEffectDisposalChecks != total.SoundEffectCreations ||
		total.SoundInstanceCreations != total.SoundEffectCreations ||
		total.SoundInstanceModeLatchChecks != total.SoundEffectCreations {
		return total, fmt.Errorf("%d sound-effect creations produced %d description, %d guard, %d scalar, %d disposal, %d instance and %d latch checks",
			total.SoundEffectCreations, total.SoundEffectDescriptionChecks, total.SoundEffectGuardChecks,
			total.SoundEffectScalarChecks, total.SoundEffectDisposalChecks,
			total.SoundInstanceCreations, total.SoundInstanceModeLatchChecks)
	}
	if total.SoundEffectPlays+total.SoundEffectPlayLimitChecks != total.SoundEffectCreations {
		return total, fmt.Errorf("sound-effect plays %d and voice-limit answers %d do not account for %d creations",
			total.SoundEffectPlays, total.SoundEffectPlayLimitChecks, total.SoundEffectCreations)
	}
	// Five transport transitions -- Play, Pause, Resume, Stop and the
	// resume-after-stop that tells Resume apart from Play -- and three scalar
	// round trips per instance.
	if total.SoundInstanceTransitions != 5*total.SoundInstanceCreations ||
		total.SoundInstanceScalarRoundTrips != 3*total.SoundInstanceCreations {
		return total, fmt.Errorf("%d instances produced %d transitions and %d scalar round trips",
			total.SoundInstanceCreations, total.SoundInstanceTransitions, total.SoundInstanceScalarRoundTrips)
	}
	if total.SoundInstanceApply3DChecks > total.SoundInstanceCreations {
		return total, fmt.Errorf("%d Apply3D checks over %d instances",
			total.SoundInstanceApply3DChecks, total.SoundInstanceCreations)
	}
	// Foundation 88. One streaming-instance outcome per cycle, and every check
	// downstream counted against the creations that succeeded.
	if total.DynamicInstanceCreations+total.DynamicInstanceRefusals != total.VertexBufferCycles {
		return total, fmt.Errorf("dynamic-instance creations %d and refusals %d do not account for %d cycles",
			total.DynamicInstanceCreations, total.DynamicInstanceRefusals, total.VertexBufferCycles)
	}
	if total.DynamicInstanceLoopChecks != total.DynamicInstanceCreations ||
		total.DynamicInstanceSubmissions != total.DynamicInstanceCreations ||
		total.DynamicInstancePendingChecks != total.DynamicInstanceCreations ||
		total.DynamicInstanceConversionChecks != total.DynamicInstanceCreations ||
		total.DynamicInstanceDisposalChecks != total.DynamicInstanceCreations {
		return total, fmt.Errorf("%d streaming instances produced %d loop, %d submission, %d pending, %d conversion and %d disposal checks",
			total.DynamicInstanceCreations, total.DynamicInstanceLoopChecks,
			total.DynamicInstanceSubmissions, total.DynamicInstancePendingChecks,
			total.DynamicInstanceConversionChecks, total.DynamicInstanceDisposalChecks)
	}
	// The microphone enumeration runs once per cycle and the guard checks with
	// it. MICROPHONES_FOUND is whatever the machine has, so it is NOT pinned to
	// a number -- what IS pinned is that every microphone found was described.
	if total.MicrophoneEnumerations != total.VertexBufferCycles ||
		total.MicrophoneGuardChecks != total.VertexBufferCycles {
		return total, fmt.Errorf("%d microphone enumerations and %d guard checks over %d cycles",
			total.MicrophoneEnumerations, total.MicrophoneGuardChecks, total.VertexBufferCycles)
	}
	if total.MicrophoneDescriptionChecks != total.MicrophonesFound {
		return total, fmt.Errorf("%d microphones found produced %d description checks",
			total.MicrophonesFound, total.MicrophoneDescriptionChecks)
	}
	// THE CAPTURE COUNTER EXISTS TO BE ZERO. Microphone.Start and GetData are
	// projected and this suite calls neither: starting capture opens a real
	// recording device on whatever machine the suite runs on. A non-zero value
	// here is a run that began recording, and it fails rather than being
	// reported.
	if total.MicrophoneCaptureCalls != 0 {
		return total, fmt.Errorf("the suite made %d microphone capture calls; it must make none",
			total.MicrophoneCaptureCalls)
	}
	// Foundation 89. Four player indices read per cycle for both capabilities
	// and state, and four vibration calls. GAMEPADS_CONNECTED is whatever the
	// machine has and is NOT pinned -- this suite runs on a build machine with
	// no controller, and a zero there is the honest answer rather than a skip.
	if total.GamePadCapabilityReads != 4*total.VertexBufferCycles ||
		total.GamePadStateReads != 4*total.VertexBufferCycles ||
		total.GamePadVibrationCalls != 4*total.VertexBufferCycles {
		return total, fmt.Errorf("%d capability reads, %d state reads and %d vibration calls over %d cycles, want four each",
			total.GamePadCapabilityReads, total.GamePadStateReads,
			total.GamePadVibrationCalls, total.VertexBufferCycles)
	}
	// A vibration can only be APPLIED to a controller that is there, so the
	// applied count can never exceed the connected count.
	if total.GamePadVibrationsApplied > total.GamePadsConnected {
		return total, fmt.Errorf("%d vibrations were applied to %d connected controllers",
			total.GamePadVibrationsApplied, total.GamePadsConnected)
	}
	if total.MouseStateReads != total.VertexBufferCycles ||
		total.MouseHandleChecks != total.VertexBufferCycles ||
		total.MousePositionWrites != total.VertexBufferCycles {
		return total, fmt.Errorf("%d mouse state reads, %d handle checks and %d position writes over %d cycles",
			total.MouseStateReads, total.MouseHandleChecks,
			total.MousePositionWrites, total.VertexBufferCycles)
	}
	// TOUCH_PANEL_NATIVE_CALLS EXISTS TO BE ZERO, and for the same reason the
	// capture counter does. The whole pinned Input.Touch assembly declares no
	// p/invoke, so a TouchPanel member that reached the native layer would be
	// answering something the reference never asks. The projection binds no
	// touch route at all; this counter is what would notice if one came back.
	if total.TouchPanelNativeCalls != 0 {
		return total, fmt.Errorf("TouchPanel made %d native calls; the reference makes none",
			total.TouchPanelNativeCalls)
	}
	if total.TouchPanelManagedChecks != total.VertexBufferCycles {
		return total, fmt.Errorf("%d touch-panel managed checks over %d cycles",
			total.TouchPanelManagedChecks, total.VertexBufferCycles)
	}
	// Foundation 82. Three dispatcher pumps and two guard checks per cycle, and
	// exactly one outcome for the read.
	if total.FrameworkDispatcherUpdates != 3*total.VertexBufferCycles {
		return total, fmt.Errorf("%d dispatcher updates over %d cycles, want three each",
			total.FrameworkDispatcherUpdates, total.VertexBufferCycles)
	}
	if total.TitleContainerReads+total.TitleContainerReadRefusals != total.VertexBufferCycles ||
		total.TitleContainerGuardChecks != 2*total.VertexBufferCycles {
		return total, fmt.Errorf("title-container reads %d, refusals %d and guard checks %d do not account for %d cycles",
			total.TitleContainerReads, total.TitleContainerReadRefusals,
			total.TitleContainerGuardChecks, total.VertexBufferCycles)
	}
	if total.EffectMaterialCreations+total.EffectMaterialRefusals != total.VertexBufferCycles ||
		total.EffectMaterialIdentityCheck != total.EffectMaterialCreations {
		return total, fmt.Errorf("EffectMaterial creations %d, refusals %d and identity checks %d do not account for %d cycles",
			total.EffectMaterialCreations, total.EffectMaterialRefusals,
			total.EffectMaterialIdentityCheck, total.VertexBufferCycles)
	}
	if total.BasicEffectRoundTrips+total.BasicEffectRoundTripRefusals != total.BasicEffectCreations ||
		total.BasicEffectApplies+total.BasicEffectApplyRefusals != total.BasicEffectCreations ||
		total.BasicEffectDisposalChecks != total.BasicEffectCreations {
		return total, fmt.Errorf("%d BasicEffect creations produced %d round trips, %d round-trip refusals, %d applies, %d apply refusals and %d disposal checks",
			total.BasicEffectCreations, total.BasicEffectRoundTrips, total.BasicEffectRoundTripRefusals,
			total.BasicEffectApplies, total.BasicEffectApplyRefusals, total.BasicEffectDisposalChecks)
	}

	// A cycle that READ BACK must also have proved the windowed overload.
	if total.IndexBufferWindowRoundTrips != total.IndexBufferRoundTrips {
		return total, fmt.Errorf("%d index round trips produced %d windowed ones",
			total.IndexBufferRoundTrips, total.IndexBufferWindowRoundTrips)
	}

	// The content slice, per cycle: one manager created, its identity proved,
	// one root round trip through CNA, one resolved path, one type refusal the
	// projection makes itself, one unload and one disposal.
	if total.ContentCycles < 20 || total.ContentManagerCreations < 20 || total.ContentIdentityChecks < 20 ||
		total.ContentRootRoundTrips < 20 || total.ContentAssetPathChecks < 20 || total.ContentTypeRefusals < 20 ||
		total.ContentUnloadCalls < 20 || total.ContentDisposalChecks < 20 {
		return total, errors.New("a content proof did not run in every cycle")
	}
	// A cycle that LOADED must also have proved the cache. The pixel check is
	// separate: a renderer with no readback path refuses it, and that refusal
	// is recorded rather than counted as a pass.
	if total.ContentCacheChecks != total.ContentLoads {
		return total, fmt.Errorf("%d content loads produced %d cache proofs", total.ContentLoads, total.ContentCacheChecks)
	}
	if total.ContentLoadPixelChecks+total.ContentLoadReadbackRefusals != total.ContentLoads {
		return total, fmt.Errorf("content pixel checks %d and readback refusals %d do not account for %d loads",
			total.ContentLoadPixelChecks, total.ContentLoadReadbackRefusals, total.ContentLoads)
	}

	// Foundation 69. The sprite-font slice. A cycle either loaded a font or was
	// refused, and a cycle that LOADED must have proved all five slices --
	// including the divergence one, which is what separates "the projection
	// measures" from "the projection measures the way the reference does".
	if total.SpriteFontCycles < 20 {
		return total, errors.New("the sprite-font scenario did not run in every cycle")
	}
	if total.SpriteFontLoads+total.SpriteFontLoadRefusals != total.SpriteFontCycles {
		return total, fmt.Errorf("sprite-font loads %d and refusals %d do not account for %d cycles",
			total.SpriteFontLoads, total.SpriteFontLoadRefusals, total.SpriteFontCycles)
	}
	// Foundation 71. The volume/cube slice, per cycle: one outcome per family,
	// and the Go-only element narrowing exercised in every one.
	if total.TextureVolumeCycles < 20 {
		return total, errors.New("the texture-volume scenario did not run in every cycle")
	}
	if total.TextureCubeCreations+total.TextureCubeCreationRefusals != total.TextureVolumeCycles ||
		total.Texture3DCreations+total.Texture3DCreationRefusals != total.TextureVolumeCycles {
		return total, fmt.Errorf("cube %d/%d and volume %d/%d creations and refusals do not account for %d cycles",
			total.TextureCubeCreations, total.TextureCubeCreationRefusals,
			total.Texture3DCreations, total.Texture3DCreationRefusals, total.TextureVolumeCycles)
	}
	if total.TextureCubeRoundTrips+total.TextureCubeTransferRefusals != total.TextureCubeCreations ||
		total.Texture3DRoundTrips+total.Texture3DTransferRefusals != total.Texture3DCreations {
		return total, fmt.Errorf("cube %d/%d and volume %d/%d round trips and refusals do not account for their creations",
			total.TextureCubeRoundTrips, total.TextureCubeTransferRefusals,
			total.Texture3DRoundTrips, total.Texture3DTransferRefusals)
	}
	if total.TextureVolumeElementRefusals != total.TextureVolumeCycles*2 ||
		total.TextureVolumeDisposalChecks != total.TextureCubeCreations+total.Texture3DCreations {
		return total, fmt.Errorf("%d element refusals and %d disposal checks across %d cycles",
			total.TextureVolumeElementRefusals, total.TextureVolumeDisposalChecks, total.TextureVolumeCycles)
	}

	// Foundation 73. The presentation slice, per cycle: the parameters are read
	// and Reset is called every time, the rectangle Present refuses every time,
	// and the back-buffer read reports exactly one outcome every time.
	if total.PresentationCycles < 20 {
		return total, errors.New("the presentation scenario did not run in every cycle")
	}
	if total.PresentationParameterReads != total.PresentationCycles ||
		total.PresentationRectangleRefusals != total.PresentationCycles ||
		total.BackBufferGuardChecks != total.PresentationCycles {
		return total, fmt.Errorf("%d parameter reads, %d rectangle refusals and %d guard checks across %d cycles",
			total.PresentationParameterReads, total.PresentationRectangleRefusals,
			total.BackBufferGuardChecks, total.PresentationCycles)
	}
	// Three resets per cycle -- Reset(), Reset(pp) and Reset(pp, adapter) --
	// and CNA's reset route is unconditional, so none of them may refuse.
	if total.PresentationResetCalls+total.PresentationResetRefusals != total.PresentationCycles*3 {
		return total, fmt.Errorf("reset calls %d and refusals %d do not account for three per %d cycles",
			total.PresentationResetCalls, total.PresentationResetRefusals, total.PresentationCycles)
	}
	if total.BackBufferPixelChecks != total.BackBufferReads {
		return total, fmt.Errorf("%d back-buffer reads produced %d pixel checks",
			total.BackBufferReads, total.BackBufferPixelChecks)
	}
	// Foundation 85. One pixel-draw outcome per cycle, and on an artifact that
	// CAN read back, every downstream check runs. The vertex-colour buckets are
	// the exception the slice records rather than asserts, so they are pinned
	// to sum to the runs instead of to a value.
	if total.PixelDrawWindingChecks+total.PixelDrawRefusals != total.PresentationCycles {
		return total, fmt.Errorf("pixel-draw winding checks %d and refusals %d do not account for %d cycles",
			total.PixelDrawWindingChecks, total.PixelDrawRefusals, total.PresentationCycles)
	}
	if total.PixelDrawGeometryChecks != total.PixelDrawWindingChecks ||
		total.PixelDrawMaterialChecks != total.PixelDrawWindingChecks ||
		total.PixelDrawAlphaChecks != total.PixelDrawWindingChecks {
		return total, fmt.Errorf("%d pixel-draw winding checks produced %d geometry, %d material and %d alpha checks",
			total.PixelDrawWindingChecks, total.PixelDrawGeometryChecks,
			total.PixelDrawMaterialChecks, total.PixelDrawAlphaChecks)
	}
	if total.PixelDrawVertexColorHonoured+total.PixelDrawVertexColorIgnored != total.PixelDrawWindingChecks {
		return total, fmt.Errorf("vertex-colour outcomes %d honoured and %d ignored do not account for %d pixel draws",
			total.PixelDrawVertexColorHonoured, total.PixelDrawVertexColorIgnored, total.PixelDrawWindingChecks)
	}
	if total.PixelDrawLightingChecks+total.PixelDrawLightingIgnored != total.PixelDrawWindingChecks {
		return total, fmt.Errorf("lighting outcomes %d changed and %d ignored do not account for %d pixel draws",
			total.PixelDrawLightingChecks, total.PixelDrawLightingIgnored, total.PixelDrawWindingChecks)
	}
	// The pixel slice runs exactly where a readback exists, so its refusals
	// must track the back buffer's rather than drift from them.
	if total.PixelDrawRefusals != total.BackBufferReadRefusals {
		return total, fmt.Errorf("%d pixel-draw refusals against %d back-buffer read refusals; the two must agree",
			total.PixelDrawRefusals, total.BackBufferReadRefusals)
	}
	if total.BackBufferReads+total.BackBufferReadRefusals != total.PresentationCycles {
		return total, fmt.Errorf("back-buffer reads %d and refusals %d do not account for %d cycles",
			total.BackBufferReads, total.BackBufferReadRefusals, total.PresentationCycles)
	}
	if total.RenderTargetCubeCreations+total.RenderTargetCubeRefusals != total.PresentationCycles ||
		total.OwnedDeviceCreations+total.OwnedDeviceRefusals != total.PresentationCycles {
		return total, fmt.Errorf("cube targets %d/%d and owned devices %d/%d do not account for %d cycles",
			total.RenderTargetCubeCreations, total.RenderTargetCubeRefusals,
			total.OwnedDeviceCreations, total.OwnedDeviceRefusals, total.PresentationCycles)
	}
	if total.OwnedDeviceDisposalChecks != total.OwnedDeviceCreations {
		return total, fmt.Errorf("%d owned devices produced %d disposal checks",
			total.OwnedDeviceCreations, total.OwnedDeviceDisposalChecks)
	}

	if total.UserPrimitiveDraws+total.UserPrimitiveDrawRefusals != total.VertexBufferCycles*userPrimitiveSubmissions ||
		total.UserPrimitiveGuardChecks != total.VertexBufferCycles {
		return total, fmt.Errorf("user-primitive draws %d, refusals %d and guard proofs %d do not account for %d cycles",
			total.UserPrimitiveDraws, total.UserPrimitiveDrawRefusals,
			total.UserPrimitiveGuardChecks, total.VertexBufferCycles)
	}

	if total.SpriteFontDrawStringGuards != total.SpriteFontLoads {
		return total, fmt.Errorf("%d font loads produced %d DrawString guard proofs",
			total.SpriteFontLoads, total.SpriteFontDrawStringGuards)
	}
	if total.SpriteFontDrawStringSubmits+total.SpriteFontDrawStringRefusals != total.SpriteFontLoads*6 {
		return total, fmt.Errorf("DrawString submits %d and refusals %d do not account for six overloads across %d loads",
			total.SpriteFontDrawStringSubmits, total.SpriteFontDrawStringRefusals, total.SpriteFontLoads)
	}
	if total.SpriteFontGlyphChecks != total.SpriteFontLoads ||
		total.SpriteFontMeasureChecks != total.SpriteFontLoads ||
		total.SpriteFontDivergenceChecks != total.SpriteFontLoads ||
		total.SpriteFontSetterRoundTrips != total.SpriteFontLoads ||
		total.SpriteFontRefusals != total.SpriteFontLoads ||
		total.SpriteFontCacheChecks != total.SpriteFontLoads {
		return total, fmt.Errorf("%d font loads produced %d glyph, %d measure, %d divergence, %d setter, %d refusal and %d cache proofs",
			total.SpriteFontLoads, total.SpriteFontGlyphChecks, total.SpriteFontMeasureChecks,
			total.SpriteFontDivergenceChecks, total.SpriteFontSetterRoundTrips,
			total.SpriteFontRefusals, total.SpriteFontCacheChecks)
	}

	// Foundation 39. The native disposal signal never raises the public event,
	// and the public event is raised only by a managed Dispose call.
	if total.GameDisposedDuringRun != 0 {
		return total, fmt.Errorf("Game.Disposed was raised %d times from a run; its only reference raise site is managed Dispose(bool)", total.GameDisposedDuringRun)
	}
	if total.GameDisposeAfterRunCycles < 20 || total.GameDisposedRepeatChecks < 20 {
		return total, errors.New("managed disposal after the run was not proved in every cycle")
	}
	// Two raises per cycle: Dispose is not idempotent, so the second call
	// raises again. A projection that had invented a disposed flag would
	// report exactly half this.
	if total.GameDisposedByManagedCall != 2*total.GameDisposeAfterRunCycles {
		return total, fmt.Errorf("%d managed Dispose calls raised Disposed %d times, want two per cycle",
			2*total.GameDisposeAfterRunCycles, total.GameDisposedByManagedCall)
	}
	if total.GameEventOrderChecks < 20 || total.GameEventOwnerThreadHits < 20 {
		return total, errors.New("native game-event ordering or owner-goroutine minimum was not met")
	}
	if total.GameEventRemovalChecks < 80 {
		return total, errors.New("native game-event removal was not proved in every isolated cycle")
	}
	// The rerun scenario is the lifetime proof: a second Run on the same Go
	// Game installs four fresh registrations after the first four were
	// released, and Add/Remove work with no native game alive at all.
	if total.GameEventRerunCycles < 20 || total.GameEventPostRunChecks < 20 {
		return total, errors.New("native game-event rerun minimum was not met")
	}
	// The optional frame hooks. Each override cycle delivers begin_run and
	// end_run exactly once, so those two counters pin the cycle count; the
	// draw hooks fire per frame and have no fixed total, but every refused
	// frame must be proved to have skipped BOTH draw and end_draw, and every
	// override call must be proved to have reached the base exactly once.
	if total.FrameHookOverrideCycles < 20 || total.FrameHookSubsetCycles < 20 {
		return total, errors.New("native frame-hook override minimum was not met")
	}
	if total.FrameHookBeginRunHits != 20 || total.FrameHookEndRunHits != 20 {
		return total, fmt.Errorf("begin_run delivered %d times and end_run %d, want exactly one per override cycle",
			total.FrameHookBeginRunHits, total.FrameHookEndRunHits)
	}
	if total.FrameHookRefusedFrames < 20 || total.FrameHookAdmittedFrames < 20 {
		return total, errors.New("the frame-hook scenario produced no refused or no admitted frames, so neither branch was exercised")
	}
	// end_draw is compared against the admitted frames of the runs that
	// actually installed it. The subset scenario declares no EndDraw at all,
	// so its admitted frames deliver draw and no end_draw hook -- which is the
	// uninstalled-member behaviour, not a skipped frame.
	if total.FrameHookEndDrawHits != total.FrameHookEndDrawExpected {
		return total, fmt.Errorf("end_draw arrived %d times for %d admitted frames on a Game that installed it; a refused frame must skip it",
			total.FrameHookEndDrawHits, total.FrameHookEndDrawExpected)
	}
	if total.FrameHookEndDrawExpected >= total.FrameHookAdmittedFrames {
		return total, errors.New("every admitted frame installed end_draw, so the uninstalled branch was never exercised")
	}
	if total.FrameHookBeginDrawHits != total.FrameHookAdmittedFrames+total.FrameHookRefusedFrames {
		return total, fmt.Errorf("begin_draw arrived %d times for %d admitted and %d refused frames",
			total.FrameHookBeginDrawHits, total.FrameHookAdmittedFrames, total.FrameHookRefusedFrames)
	}
	if total.FrameHookSkipChecks < 20 || total.FrameHookOrderChecks < 20 || total.FrameHookBaseCallChecks < 20 {
		return total, errors.New("the refused-frame, ordering or explicit-base-call proof did not run in every cycle")
	}
	// The whole point of the capability being per hook: a Game that declares
	// only BeginDraw must never receive the other three.
	if total.FrameHookUninstalledHits != 0 {
		return total, fmt.Errorf("%d hooks were delivered for capabilities the callback object never declared", total.FrameHookUninstalledHits)
	}
	// Foundation 42. Six timing and presentation settings reach the live native
	// loop, one is refused from a non-owner goroutine, and a Game configured
	// before Run is created with what it was configured with.
	if total.TimingCycles < 20 || total.TimingCreatedWithConfig < 20 {
		return total, errors.New("native timing minimum was not met")
	}
	if total.TimingSettersApplied != 6*total.TimingCycles {
		return total, fmt.Errorf("%d timing settings reached a live native game across %d cycles, want six per cycle",
			total.TimingSettersApplied, total.TimingCycles)
	}
	if total.TimingWrongThreadChecks < 20 || total.TimingRangeChecks < 20 {
		return total, errors.New("the timing thread or range proof did not run in every cycle")
	}
	// Foundation 45. Each window cycle measures the same members three times:
	// before Run with no native window, during Run with one, and after Run
	// with none again. The guarded and unguarded families must BOTH have been
	// exercised in every cycle, because the split between them is the whole
	// claim.
	if total.WindowCycles < 20 {
		return total, errors.New("native window minimum was not met")
	}
	if total.WindowIdentityChecks != 3*total.WindowCycles {
		return total, fmt.Errorf("Game.Window identity was checked %d times across %d cycles, want three per cycle",
			total.WindowIdentityChecks, total.WindowCycles)
	}
	if total.WindowGuardedFallbacks != 6*total.WindowCycles {
		return total, fmt.Errorf("%d guarded-fallback checks across %d cycles, want six per cycle",
			total.WindowGuardedFallbacks, total.WindowCycles)
	}
	if total.WindowUnguardedFailures != 4*total.WindowCycles {
		return total, fmt.Errorf("%d unguarded-failure checks across %d cycles, want four per cycle",
			total.WindowUnguardedFailures, total.WindowCycles)
	}
	if total.WindowLiveReads != 6*total.WindowCycles {
		return total, fmt.Errorf("%d live window reads across %d cycles, want six per cycle",
			total.WindowLiveReads, total.WindowCycles)
	}
	if total.WindowTitleSuppressions < 20 || total.WindowWrongThreadChecks < 20 {
		return total, errors.New("the window title suppression or thread proof did not run in every cycle")
	}
	if total.WindowScreenDeviceChanges < 20 {
		return total, errors.New("the screen-device-change pair did not run in every cycle")
	}
	// Foundation 47. The frame-step lifecycle: a native game that exists
	// without a loop, driven a frame at a time, and destroyed by Dispose.
	if total.FrameStepCycles < 20 {
		return total, errors.New("native frame-step minimum was not met")
	}
	// Six steps per cycle: two Ticks, two RunOneFrames, one suppressed Tick
	// and one post-Exit Tick.
	if total.FrameStepTicks != 4*total.FrameStepCycles || total.FrameStepRunOneFrames != 2*total.FrameStepCycles {
		return total, fmt.Errorf("%d ticks and %d run-one-frames across %d cycles, want four and two per cycle",
			total.FrameStepTicks, total.FrameStepRunOneFrames, total.FrameStepCycles)
	}
	// Initialization happens exactly ONCE per session however many frames are
	// stepped, and the two Ticks that precede the first RunOneFrame deliver
	// none of it.
	if total.FrameStepInitializations != total.FrameStepCycles {
		return total, fmt.Errorf("%d initializations across %d frame-step cycles, want exactly one per session",
			total.FrameStepInitializations, total.FrameStepCycles)
	}
	if total.FrameStepTickInitChecks < 20 {
		return total, errors.New("the proof that a tick does not initialize did not run in every cycle")
	}
	// Every step delivers exactly one Update; only the suppressed one and the
	// post-Exit one skip Draw.
	if total.FrameStepUpdates != 6*total.FrameStepCycles {
		return total, fmt.Errorf("%d updates across %d cycles, want six per cycle", total.FrameStepUpdates, total.FrameStepCycles)
	}
	if total.FrameStepDraws >= total.FrameStepUpdates {
		return total, fmt.Errorf("%d draws for %d updates; SuppressDraw skipped none", total.FrameStepDraws, total.FrameStepUpdates)
	}
	if total.FrameStepSuppressChecks < 20 || total.FrameStepExitChecks < 20 {
		return total, errors.New("the suppress-draw or exit proof did not run in every cycle")
	}
	if total.FrameStepWrongThreadChecks < 20 || total.FrameStepCallbackRefusals < 20 {
		return total, errors.New("the owner-thread or in-callback refusal proof did not run in every cycle")
	}
	if total.FrameStepSessionChecks < 60 || total.FrameStepDisposeChecks < 20 || total.FrameStepRecreationChecks < 20 {
		return total, errors.New("the session lifetime, disposal or recreation proof did not run in every cycle")
	}
	if total.FrameStepRunAfterStepCycle < 20 {
		return total, errors.New("Run did not adopt a standalone session in every cycle")
	}
	// Foundation 48. GraphicsDeviceManager's nine configuration setters push to
	// CNA's own manager, which is the object ApplyChanges reads.
	if total.ManagerCycles < 20 || total.ManagerDefaultChecks < 20 {
		return total, errors.New("native graphics-manager minimum was not met")
	}
	if total.ManagerSettersApplied != 6*total.ManagerCycles {
		return total, fmt.Errorf("%d framework-typed settings across %d cycles, want six per cycle",
			total.ManagerSettersApplied, total.ManagerCycles)
	}
	if total.ManagerCrossPackageSets != 3*total.ManagerCycles {
		return total, fmt.Errorf("%d cross-package settings across %d cycles, want three per cycle",
			total.ManagerCrossPackageSets, total.ManagerCycles)
	}
	if total.ManagerRangeChecks < 20 || total.ManagerApplyChanges < 20 ||
		total.ManagerToggleChecks < 20 || total.ManagerWrongThreadCheck < 20 {
		return total, errors.New("a graphics-manager range, apply, toggle or thread proof did not run in every cycle")
	}
	// Foundation 49. The manager registers itself under both service contracts,
	// which is what finally makes Game.GraphicsDevice and
	// DrawableGameComponent.Initialize work with no consumer-supplied service.
	if total.ManagerServiceChecks != 2*total.ManagerCycles {
		return total, fmt.Errorf("%d service-registration checks across %d cycles, want two per cycle",
			total.ManagerServiceChecks, total.ManagerCycles)
	}
	if total.ManagerDuplicateChecks < 20 || total.ManagerServiceRemovalCheck < 20 {
		return total, errors.New("the duplicate-registration or service-removal proof did not run in every cycle")
	}
	if total.ManagerGameDeviceChecks < 20 || total.ManagerDrawableChecks < 20 {
		return total, errors.New("Game.GraphicsDevice or DrawableGameComponent did not resolve the published service in every cycle")
	}
	if total.ManagerEventRaiseChecks != 4*total.ManagerCycles {
		return total, fmt.Errorf("%d manager raiser checks across %d cycles, want four per cycle",
			total.ManagerEventRaiseChecks, total.ManagerCycles)
	}
	// The device-created signal is the one HEADLESS actually produces, and it
	// must arrive at least once per cycle: it is what proves the native
	// subscription is installed and routed rather than merely accepted.
	if total.ManagerSignalDeviceCreated < total.ManagerCycles {
		return total, fmt.Errorf("%d device-created signals across %d cycles, want at least one per cycle",
			total.ManagerSignalDeviceCreated, total.ManagerCycles)
	}
	return total, nil
}

func decodeLastJSONLine(output []byte, value any) error {
	lines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
	for index := len(lines) - 1; index >= 0; index-- {
		line := bytes.TrimSpace(lines[index])
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		if err := json.Unmarshal(line, value); err == nil {
			return nil
		}
	}
	return errors.New("no JSON result line found")
}

func runChild(scenario string, index int) error {
	if scenario == "frame-hook-override" || scenario == "frame-hook-subset" {
		return runFrameHookChild(scenario)
	}
	if scenario == "timing" {
		return runTimingChild()
	}
	if scenario == "window" {
		return runWindowChild()
	}
	if scenario == "frame-step" {
		return runFrameStepChild()
	}
	if scenario == "frame-step-run" {
		return runFrameStepRunChild()
	}
	if scenario == "graphics-manager" {
		return runGraphicsManagerChild()
	}
	game := &stressGame{scenario: scenario, index: index, data: encodedPNG()}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	// Foundation 82 tried to pump the dispatcher HERE, before Run, because
	// CNA's header names that case: "The canonical dispatcher is static and
	// exists for applications that do not run the game loop." It refuses.
	// framework.NewGame constructs no native game -- CNA-Go creates it inside
	// Run -- so there is no constructed-but-not-running state a consumer can
	// reach, and every call site the projection has is already inside a
	// callback. The probe is recorded rather than kept: it measured a state
	// this binding does not have.
	if err := game.subscribeGameEvents(host); err != nil {
		return err
	}
	if err := verifyKeyboardUnavailable("before Game.Run"); err != nil {
		return err
	}
	err = host.Run()
	game.recordGameEventDeliveries()
	if scenario == "event-rerun" {
		return runEventRerunChild(game, host, err)
	}
	if unavailableErr := verifyKeyboardUnavailable("after Game shutdown"); unavailableErr != nil {
		return unavailableErr
	}
	switch scenario {
	case "success":
		game.result.GameCycles = 1
		game.result.GameRecreationCycles = 1
		if err != nil {
			return err
		}
		if verifyErr := game.verifyGameEventDelivery(); verifyErr != nil {
			return verifyErr
		}
		if disposeErr := game.verifyManagedDisposalAfterRun(host); disposeErr != nil {
			return disposeErr
		}
		if _, staleErr := game.device.Viewport(); !errors.Is(staleErr, interop.ErrStaleGeneration) {
			game.result.ObservedUAF++
			return fmt.Errorf("stale graphics device was not rejected by generation: %w", staleErr)
		}
	case "device-state":
		game.result.DeviceStateCycles = 1
		if err != nil {
			return err
		}
		// Every facade from a finished run is stale, and the state members must
		// report that rather than reaching a device that is gone.
		if _, staleErr := game.device.BlendFactor(); !errors.Is(staleErr, interop.ErrStaleGeneration) {
			game.result.ObservedUAF++
			return fmt.Errorf("stale device BlendFactor was not rejected by generation: %w", staleErr)
		}
		game.result.DeviceStateStaleChecks++
	case "sprite-draw":
		game.result.SpriteDrawCycles = 1
		if err != nil {
			return err
		}
		if game.result.SpriteDrawScaledSubmits == 0 || game.result.SpriteDrawDestinationSubmits == 0 {
			return errors.New("the sprite-draw scenario submitted nothing")
		}
	case "adapter":
		game.result.AdapterCycles = 1
		if err != nil {
			return err
		}
		if game.result.AdapterEnumerations == 0 {
			return errors.New("the adapter scenario enumerated nothing")
		}
	case "vertex-buffer":
		game.result.VertexBufferCycles = 1
		if err != nil {
			return err
		}
		if game.result.VertexBufferCreations == 0 {
			return errors.New("the vertex-buffer scenario created nothing")
		}
		if game.result.VertexBufferRoundTrips+game.result.VertexBufferReadbackRefusals != game.result.VertexBufferCreations {
			return fmt.Errorf("vertex-buffer round trips %d and refusals %d do not account for %d creations",
				game.result.VertexBufferRoundTrips, game.result.VertexBufferReadbackRefusals, game.result.VertexBufferCreations)
		}
		// Foundation 72. Exactly one outcome for the effect, and the draw
		// accounted for once.
		if game.result.VertexBufferEffectLoads+game.result.VertexBufferEffectRefusals != 1 {
			return errors.New("the stock effect reported neither a load nor a refusal")
		}
		if game.result.VertexBufferDraws+game.result.VertexBufferDrawRefusals != 1 {
			return errors.New("the post-apply draw reported neither a success nor a refusal")
		}
		// Eleven submissions, each with exactly one outcome: the six overloads
		// over this file's stressVertex, plus Foundation 77's five over the
		// profile's own stock vertex types -- one per type, and one with a
		// non-zero vertexOffset.
		if game.result.UserPrimitiveDraws+game.result.UserPrimitiveDrawRefusals != userPrimitiveSubmissions {
			return fmt.Errorf("user-primitive draws %d and refusals %d do not account for the %d submissions",
				game.result.UserPrimitiveDraws, game.result.UserPrimitiveDrawRefusals, userPrimitiveSubmissions)
		}
		if game.result.UserPrimitiveGuardChecks == 0 {
			return errors.New("the user-primitive guards did not run")
		}
	case "index-buffer":
		game.result.IndexBufferCycles = 1
		if err != nil {
			return err
		}
		if game.result.IndexBufferCreations == 0 {
			return errors.New("the index-buffer scenario created nothing")
		}
		// Exactly one of the two outcomes per creation, for the same reason
		// the render-target readback has one.
		if game.result.IndexBufferRoundTrips+game.result.IndexBufferReadbackRefusals != game.result.IndexBufferCreations {
			return fmt.Errorf("index-buffer round trips %d and refusals %d do not account for %d creations",
				game.result.IndexBufferRoundTrips, game.result.IndexBufferReadbackRefusals, game.result.IndexBufferCreations)
		}
	case "content":
		game.result.ContentCycles = 1
		if err != nil {
			return err
		}
		if game.result.ContentManagerCreations == 0 {
			return errors.New("the content scenario created no native content manager")
		}
		// Exactly one of the two outcomes per cycle, for the same reason the
		// render-target bind has: a run reporting neither never reached the
		// load, and one reporting both ran twice.
		if game.result.ContentLoads+game.result.ContentLoadRefusals != game.result.ContentManagerCreations {
			return fmt.Errorf("content loads %d and refusals %d do not account for %d managers",
				game.result.ContentLoads, game.result.ContentLoadRefusals, game.result.ContentManagerCreations)
		}
	case "render-target":
		game.result.RenderTargetCycles = 1
		if err != nil {
			return err
		}
		if game.result.RenderTargetCreations == 0 {
			return errors.New("the render-target scenario created nothing")
		}
		// Exactly one of the two outcomes, every cycle. A run reporting neither
		// never reached the bind, and one reporting both would mean the
		// scenario ran twice in a process that should host one cycle.
		if game.result.RenderTargetBinds+game.result.RenderTargetBindRefusals != game.result.RenderTargetCreations {
			return fmt.Errorf("render-target binds %d and refusals %d do not account for %d creations",
				game.result.RenderTargetBinds, game.result.RenderTargetBindRefusals, game.result.RenderTargetCreations)
		}
		// A renderer that BOUND must have produced the pixels, drawn them
		// through the Texture2D position and disposed cleanly. A renderer that
		// refused proves the description and the substitution and nothing more,
		// which is what BLOCKED_RENDERER means here.
		if game.result.RenderTargetBinds > 0 {
			if game.result.RenderTargetSpriteDraws == 0 || game.result.RenderTargetDisposalChecks == 0 {
				return errors.New("the render target bound and the semantic slice did not complete")
			}
			// The pixel check is the one step a renderer may be unable to
			// perform, so it is required to be accounted for rather than
			// required to have happened.
			if game.result.RenderTargetPixelChecks+game.result.RenderTargetReadbackRefusals != game.result.RenderTargetBinds {
				return fmt.Errorf("render-target pixel checks %d and readback refusals %d do not account for %d binds",
					game.result.RenderTargetPixelChecks, game.result.RenderTargetReadbackRefusals, game.result.RenderTargetBinds)
			}
		}
	case "sprite-font":
		game.result.SpriteFontCycles = 1
		if err != nil {
			return err
		}
		// Exactly one of the two outcomes per cycle, for the reason the content
		// scenario has one: a run reporting neither never reached the load.
		if game.result.SpriteFontLoads+game.result.SpriteFontLoadRefusals != 1 {
			return fmt.Errorf("sprite-font loads %d and refusals %d do not account for one cycle",
				game.result.SpriteFontLoads, game.result.SpriteFontLoadRefusals)
		}
		if game.result.SpriteFontLoads > 0 &&
			(game.result.SpriteFontGlyphChecks == 0 || game.result.SpriteFontMeasureChecks == 0 ||
				game.result.SpriteFontDivergenceChecks == 0 || game.result.SpriteFontSetterRoundTrips == 0 ||
				game.result.SpriteFontCacheChecks == 0 || game.result.SpriteFontDrawStringGuards == 0) {
			return errors.New("a font loaded and the semantic slice did not complete")
		}
		// Six overloads, and exactly one outcome each: CNA submitted it, or
		// CNA refused it because this renderer cannot draw text. A run
		// reporting neither never reached the begin/end pair.
		if game.result.SpriteFontLoads > 0 &&
			game.result.SpriteFontDrawStringSubmits+game.result.SpriteFontDrawStringRefusals != 6 {
			return fmt.Errorf("DrawString submits %d and refusals %d do not account for six overloads",
				game.result.SpriteFontDrawStringSubmits, game.result.SpriteFontDrawStringRefusals)
		}
	case "texture-volume":
		game.result.TextureVolumeCycles = 1
		if err != nil {
			return err
		}
		// Exactly one outcome per family per cycle: CNA created it, or CNA
		// refused because this renderer has no cube or volume storage.
		if game.result.TextureCubeCreations+game.result.TextureCubeCreationRefusals != 1 {
			return errors.New("the cube family reported neither a creation nor a refusal")
		}
		if game.result.Texture3DCreations+game.result.Texture3DCreationRefusals != 1 {
			return errors.New("the volume family reported neither a creation nor a refusal")
		}
		if game.result.TextureVolumeElementRefusals == 0 {
			return errors.New("the one-element-wide narrowing was not exercised")
		}
	case "presentation":
		game.result.PresentationCycles = 1
		if err != nil {
			return err
		}
		// Reset is unconditional in CNA, so the run must have called it -- and
		// the parameters read that precedes it must have happened too.
		if game.result.PresentationParameterReads == 0 || game.result.PresentationResetCalls == 0 {
			return errors.New("the presentation scenario neither read the parameters nor reset")
		}
		// The rich Present overload's refusal is a CONTRACT refusal rather than
		// a renderer one, so it must happen in every cycle on every artifact.
		if game.result.PresentationRectangleRefusals == 0 {
			return errors.New("the rectangle Present overload did not refuse")
		}
		// Exactly one back-buffer outcome, and the Go-side guard proved either
		// way -- it needs no renderer at all.
		if game.result.BackBufferReads+game.result.BackBufferReadRefusals == 0 {
			return errors.New("the back-buffer read reported neither an outcome nor a refusal")
		}
		if game.result.BackBufferGuardChecks == 0 {
			return errors.New("the active-render-target guard was not exercised")
		}
		// A renderer that read the buffer back MUST have had its pixels checked.
		if game.result.BackBufferPixelChecks != game.result.BackBufferReads {
			return fmt.Errorf("%d back-buffer reads produced %d pixel checks",
				game.result.BackBufferReads, game.result.BackBufferPixelChecks)
		}
		// One cube-target outcome per cycle; a creation that succeeded must
		// have reached the bind and the binding round trip.
		if game.result.RenderTargetCubeCreations+game.result.RenderTargetCubeRefusals != 1 {
			return fmt.Errorf("cube target creations %d and refusals %d do not account for one cycle",
				game.result.RenderTargetCubeCreations, game.result.RenderTargetCubeRefusals)
		}
		if game.result.RenderTargetCubeCreations > 0 {
			if game.result.RenderTargetCubeBinds+game.result.RenderTargetCubeBindRefusals !=
				game.result.RenderTargetCubeCreations {
				return fmt.Errorf("cube binds %d and refusals %d do not account for %d creations",
					game.result.RenderTargetCubeBinds, game.result.RenderTargetCubeBindRefusals,
					game.result.RenderTargetCubeCreations)
			}
			if game.result.RenderTargetBindingChecks == 0 {
				return errors.New("a cube target was created and no binding was checked")
			}
		}
		// One owned-device outcome, and every owned device destroyed.
		if game.result.OwnedDeviceCreations+game.result.OwnedDeviceRefusals != 1 {
			return fmt.Errorf("owned device creations %d and refusals %d do not account for one cycle",
				game.result.OwnedDeviceCreations, game.result.OwnedDeviceRefusals)
		}
		if game.result.OwnedDeviceDisposalChecks != game.result.OwnedDeviceCreations {
			return fmt.Errorf("%d owned devices produced %d disposal checks",
				game.result.OwnedDeviceCreations, game.result.OwnedDeviceDisposalChecks)
		}
	case "callback-error":
		game.result.CallbackErrorCycles = 1
		if !errors.Is(err, callbackSentinel) {
			return fmt.Errorf("callback error was not returned from Run: %w", err)
		}
	case "callback-panic":
		game.result.CallbackPanicCycles = 1
		if err == nil {
			return errors.New("callback panic was not contained and returned")
		}
	default:
		return fmt.Errorf("unknown child scenario %q", scenario)
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// subscribeGameEvents registers on all four projected Game events BEFORE the
// native game exists, which is the point of the exercise: a Go consumer never
// waits for a native subscription, because the bridge installs exactly one per
// event when the native host is created and the Go registration list is
// entirely managed state.
//
// It also registers and immediately removes a fifth handler. That handler must
// never run, which is what proves removal reaches the delivery path rather than
// only the registration list.
func (g *stressGame) subscribeGameEvents(host *framework.Game) error {
	g.eventGoroutines = map[string]bool{}
	record := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, args *framework.EventArgs) error {
			g.eventOrder = append(g.eventOrder, name)
			if args != framework.EventArgsEmpty() {
				return fmt.Errorf("%s carried args that are not EventArgs.Empty", name)
			}
			// Exiting is the one event the reference raises with a null
			// sender; the other three raise with the Game.
			if name == "Exiting" {
				if sender != nil {
					return fmt.Errorf("Exiting sender = %v, want nil", sender)
				}
			} else if sender != any(host) {
				return fmt.Errorf("%s sender = %v, want the Game", name, sender)
			}
			g.eventGoroutines[currentGoroutineLabel()] = true
			return nil
		}
	}
	if _, err := host.AddActivatedHandler(record("Activated")); err != nil {
		return err
	}
	if _, err := host.AddDeactivatedHandler(record("Deactivated")); err != nil {
		return err
	}
	if _, err := host.AddExitingHandler(record("Exiting")); err != nil {
		return err
	}
	// Disposed does NOT join the ordered log. Its reference raise site is
	// managed Dispose(bool), so during a run it must never fire at all; this
	// handler exists to prove exactly that, and to fire when the run is over
	// and the consumer disposes on purpose.
	if _, err := host.AddDisposedHandler(func(sender any, args *framework.EventArgs) error {
		g.disposedRaises++
		if args != framework.EventArgsEmpty() {
			return fmt.Errorf("Disposed carried args that are not EventArgs.Empty")
		}
		if sender != any(host) {
			return fmt.Errorf("Disposed sender = %v, want the Game", sender)
		}
		g.eventGoroutines[currentGoroutineLabel()] = true
		return nil
	}); err != nil {
		return err
	}
	removed, err := host.AddExitingHandler(func(any, *framework.EventArgs) error {
		g.removedRan = true
		return nil
	})
	if err != nil {
		return err
	}
	if err := host.RemoveExitingHandler(removed); err != nil {
		return err
	}
	// Removing the same token twice, and removing it from a different event,
	// must both be inert.
	if err := host.RemoveExitingHandler(removed); err != nil {
		return err
	}
	return host.RemoveDisposedHandler(removed)
}

func (g *stressGame) recordGameEventDeliveries() {
	for _, name := range g.eventOrder {
		switch name {
		case "Activated":
			g.result.GameEventActivated++
		case "Deactivated":
			g.result.GameEventDeactivated++
		case "Exiting":
			g.result.GameEventExiting++
		}
	}
	// The native disposal signal is read from the internal runtime, which is
	// the only place it is observable now that it raises nothing public.
	if g.runtime != nil {
		g.result.GameNativeDisposalSignals = g.runtime.GameEventDeliveries()[interop.GameEventDisposed]
	}
	g.result.GameDisposedDuringRun = g.disposedRaises
	if !g.removedRan {
		g.result.GameEventRemovalChecks++
	}
	if len(g.eventGoroutines) == 1 && g.eventGoroutines[g.ownerGoroutine] {
		g.result.GameEventOwnerThreadHits++
	}
}

// verifyGameEventDelivery holds the exact ordering the pinned runtime produces
// for a clean run, measured against libcna_c_api.so:
//
//	initialize -> load_content -> begin_run -> ACTIVATED -> update/draw...
//	-> exiting callback -> EXITING -> end_run -> [cna_game_run returns]
//	-> unload_content -> DISPOSED -> [cna_game_destroy returns]
//
// The two facts that matter to the projection are that Exiting precedes
// Disposed and that each is delivered exactly once. Deactivated is NOT asserted:
// the qualification artifact runs a HEADLESS renderer with no window manager,
// so no focus transition away from the game can be produced, and inventing one
// to make a counter move would be fabricating evidence.
func (g *stressGame) verifyGameEventDelivery() error {
	if g.result.GameEventActivated != 1 {
		return fmt.Errorf("Activated delivered %d times, want exactly 1", g.result.GameEventActivated)
	}
	if g.result.GameEventExiting != 1 {
		return fmt.Errorf("Exiting delivered %d times, want exactly 1", g.result.GameEventExiting)
	}
	// The native disposal signal still arrives exactly once per run, from
	// inside cna_game_destroy, and that is what proves the four registrations
	// outlive native destruction. It raises nothing public.
	if g.result.GameNativeDisposalSignals != 1 {
		return fmt.Errorf("the native disposal signal arrived %d times, want exactly 1", g.result.GameNativeDisposalSignals)
	}
	if g.result.GameDisposedDuringRun != 0 {
		return fmt.Errorf("Game.Disposed was raised %d times during a run; its only reference raise site is managed Dispose(bool)", g.result.GameDisposedDuringRun)
	}
	exiting := -1
	for i, name := range g.eventOrder {
		if name == "Exiting" {
			exiting = i
		}
	}
	if exiting < 0 {
		return fmt.Errorf("delivery order %v contains no Exiting", g.eventOrder)
	}
	if g.eventOrder[0] != "Activated" {
		return fmt.Errorf("delivery order %v does not start with Activated", g.eventOrder)
	}
	g.result.GameEventOrderChecks++
	if g.removedRan {
		return errors.New("a handler removed before Run was still delivered")
	}
	if g.result.GameEventOwnerThreadHits != 1 {
		return fmt.Errorf("game events were delivered on %d distinct goroutines, want the owner goroutine only", len(g.eventGoroutines))
	}
	return nil
}

// verifyManagedDisposalAfterRun is the other half of the Foundation 39
// correction, proved against a Game whose native generation is already gone.
//
// Three facts, none of which the old native-signal binding could have produced:
//
//  1. Disposing AFTER the run raises Game.Disposed. The reference's raise site
//     is managed Dispose(bool), and managed state outlives the native host, so
//     a consumer who disposes when the run is over still gets the event.
//  2. It raises with the Game as sender and EventArgs.Empty as args, checked
//     by the handler itself.
//  3. Dispose is NOT idempotent. A second call raises again, because Game
//     carries no disposed flag anywhere. A projection that invented one would
//     report one raise here instead of two.
//
// Nothing here reaches native code. There is no live handle left and none is
// fabricated: the whole body is managed component and event work.
func (g *stressGame) verifyManagedDisposalAfterRun(host *framework.Game) error {
	before := g.disposedRaises
	if before != 0 {
		return fmt.Errorf("Game.Disposed had already been raised %d times before any Dispose call", before)
	}
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose after the run: %w", err)
	}
	if g.disposedRaises != 1 {
		return fmt.Errorf("one managed Dispose raised Game.Disposed %d times, want 1", g.disposedRaises)
	}
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("second Dispose after the run: %w", err)
	}
	if g.disposedRaises != 2 {
		return fmt.Errorf("a second Dispose raised Game.Disposed %d times in total, want 2; Game has no disposed flag", g.disposedRaises)
	}
	// Dispose(false) is the finalizer path and does nothing at all.
	if err := host.DisposeByBoolean(false); err != nil {
		return fmt.Errorf("Dispose(false) after the run: %w", err)
	}
	if err := host.Finalize(); err != nil {
		return fmt.Errorf("Finalize after the run: %w", err)
	}
	if g.disposedRaises != 2 {
		return fmt.Errorf("Dispose(false) or Finalize raised Game.Disposed; total is %d, want 2", g.disposedRaises)
	}
	// The native disposal signal count did not move: managed disposal reaches
	// no native code at all.
	if g.runtime != nil {
		if signals := g.runtime.GameEventDeliveries()[interop.GameEventDisposed]; signals != g.result.GameNativeDisposalSignals {
			return fmt.Errorf("managed disposal changed the native disposal signal count from %d to %d",
				g.result.GameNativeDisposalSignals, signals)
		}
	}
	g.result.GameDisposedByManagedCall = g.disposedRaises
	g.result.GameDisposedRepeatChecks++
	g.result.GameDisposeAfterRunCycles++
	return nil
}

// currentGoroutineLabel identifies the delivering goroutine without exposing
// any runtime identity beyond this file. Every native game event must arrive on
// the same locked owner goroutine that entered cna_game_run.
func currentGoroutineLabel() string {
	buffer := make([]byte, 64)
	buffer = buffer[:runtime.Stack(buffer, false)]
	line := string(buffer)
	if index := bytes.IndexByte([]byte(line), '['); index > 0 {
		return line[:index]
	}
	return line
}

// runEventRerunChild proves the native subscription lifetime claim end to end:
// the four registrations installed for one run are released when that run ends,
// a SECOND Run on the same Go Game installs four fresh ones, and the projected
// accessors keep working across both without a consumer resubscribing.
//
// It also records the one behavior a second run makes visible. Game::isActive
// is a private field the reference never resets, and the two activation events
// are edge-triggered on it, so a second run's activation signal is SUPPRESSED --
// the game was already active as far as the managed half is concerned. That is
// what HostActivated does in CLR too, and it is recorded here rather than
// smoothed over.
func runEventRerunChild(game *stressGame, host *framework.Game, firstErr error) error {
	if firstErr != nil {
		return fmt.Errorf("first run: %w", firstErr)
	}
	first := game.result
	if first.GameEventActivated != 1 || first.GameEventExiting != 1 || first.GameNativeDisposalSignals != 1 {
		return fmt.Errorf("first run delivered %v and %d native disposal signals", game.eventOrder, first.GameNativeDisposalSignals)
	}
	if first.GameDisposedDuringRun != 0 {
		return fmt.Errorf("the first run raised Game.Disposed %d times", first.GameDisposedDuringRun)
	}

	// Add and remove handlers with no native game alive at all. Both are pure
	// managed list work, so both must be ordinary successes.
	postRun, err := host.AddExitingHandler(func(any, *framework.EventArgs) error { return nil })
	if err != nil {
		return fmt.Errorf("subscribe after the run ended: %w", err)
	}
	if err := host.RemoveExitingHandler(postRun); err != nil {
		return fmt.Errorf("unsubscribe after the run ended: %w", err)
	}
	game.result.GameEventPostRunChecks++

	game.eventOrder = nil
	game.eventGoroutines = map[string]bool{}
	game.manager = nil
	game.device = nil
	if err := host.Run(); err != nil {
		return fmt.Errorf("second run: %w", err)
	}
	second := game.eventOrder
	activated, exiting := 0, 0
	for _, name := range second {
		switch name {
		case "Activated":
			activated++
		case "Exiting":
			exiting++
		}
	}
	if exiting != 1 {
		return fmt.Errorf("second run delivered %v; Exiting must arrive exactly once", second)
	}
	// Two runs, two native destructions, two disposal signals -- and still no
	// public Disposed raise, because nobody disposed anything.
	signals := 0
	if game.runtime != nil {
		signals = game.runtime.GameEventDeliveries()[interop.GameEventDisposed]
	}
	if signals != 2 {
		return fmt.Errorf("two runs produced %d native disposal signals, want 2", signals)
	}
	if game.disposedRaises != 0 {
		return fmt.Errorf("two runs raised Game.Disposed %d times with no Dispose call", game.disposedRaises)
	}
	// The edge-trigger guard is the whole point: the managed half never saw a
	// deactivation, so the second run's activation signal raises nothing.
	if activated != 0 {
		return fmt.Errorf("second run raised Activated %d times; isActive was already true, so the guard must suppress it", activated)
	}
	if len(game.eventGoroutines) != 1 || !game.eventGoroutines[game.ownerGoroutine] {
		return fmt.Errorf("second run delivered on %d goroutines", len(game.eventGoroutines))
	}
	// The counter is refreshed to the two-run total BEFORE managed disposal is
	// proved, because that proof asserts managed disposal does not move it.
	game.result.GameNativeDisposalSignals = signals
	if disposeErr := game.verifyManagedDisposalAfterRun(host); disposeErr != nil {
		return disposeErr
	}
	game.result.GameEventRerunCycles++
	game.result.GameEventActivated = first.GameEventActivated + activated
	game.result.GameEventExiting = first.GameEventExiting + exiting
	game.result.GameDisposedDuringRun = 0
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

func (g *stressGame) Initialize(host *framework.Game) error {
	g.ownerGoroutine = currentGoroutineLabel()
	if current, ok := interop.CurrentRuntime(); ok {
		g.runtime = current
	}
	manager, err := framework.NewGraphicsDeviceManager(host)
	if err != nil {
		return err
	}
	g.manager = manager
	if got := manager.SupportedOrientations(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("initial SupportedOrientations = %d, want Default", got)
	}
	configured := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	manager.SetSupportedOrientations(configured)
	if got := manager.SupportedOrientations(); got != configured {
		return fmt.Errorf("Initialize SupportedOrientations = %d, want %d", got, configured)
	}
	return nil
}

func (g *stressGame) LoadContent(_ *framework.Game) error {
	configured := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationPortrait
	if got := g.manager.SupportedOrientations(); got != configured {
		return fmt.Errorf("LoadContent SupportedOrientations = %d, want %d", got, configured)
	}
	g.manager.SetSupportedOrientations(configured)
	unknown := framework.DisplayOrientation(1 << 20)
	g.manager.SetSupportedOrientations(unknown)
	if got := g.manager.SupportedOrientations(); got != unknown {
		return fmt.Errorf("raw SupportedOrientations = %d, want %d", got, unknown)
	}
	if err := verifyKeyboardPlayerIndexSnapshots(); err != nil {
		return err
	}
	device, err := graphics.GraphicsDeviceManagerGraphicsDevice(g.manager)
	if err != nil {
		return err
	}
	g.device = device
	if g.scenario == "sprite-draw" {
		texture, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
		if err != nil {
			return err
		}
		batch, err := graphics.NewSpriteBatch(device)
		if err != nil {
			return err
		}
		g.spriteTexture = texture
		g.spriteBatch = batch
		// The two guards are checked HERE, outside any begin/end pair, which is
		// the state InternalDraw's second throw is about. A nil texture is
		// checked first because the reference checks it first: its
		// ArgumentNullException is thrown before the pair is even read, so this
		// call is outside a pair AND has a nil texture, and must report the
		// argument.
		nullErr := batch.DrawByTexture2DAndVector2AndColor(nil, framework.Vector2{}, framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255))
		if nullErr == nil || !strings.Contains(nullErr.Error(), "This method does not accept null for this parameter.") {
			return fmt.Errorf("nil-texture Draw outside a pair = %v, want the ArgumentNullException message", nullErr)
		}
		g.result.SpriteDrawNullTextureChecks++
		outsideErr := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255))
		if outsideErr == nil || !strings.Contains(outsideErr.Error(), "Begin must be called successfully before a Draw can be called.") {
			return fmt.Errorf("Draw outside a pair = %v, want the InvalidOperationException message", outsideErr)
		}
		g.result.SpriteDrawOutsidePairChecks++
		endErr := batch.End()
		if endErr == nil || !strings.Contains(endErr.Error(), "Begin must be called successfully before End can be called.") {
			return fmt.Errorf("End outside a pair = %v, want the InvalidOperationException message", endErr)
		}
		g.result.SpriteDrawPairGuardChecks++
		bounds := texture.Bounds()
		width := texture.Width()
		height := texture.Height()
		if bounds.X != 0 || bounds.Y != 0 || bounds.Width != width || bounds.Height != height {
			return fmt.Errorf("Bounds = %+v, want (0,0,%d,%d)", bounds, width, height)
		}
		g.result.SpriteDrawBoundsChecks++
		return nil
	}
	if g.scenario != "success" || g.index != 0 {
		return nil
	}
	for cycle := 0; cycle < 20; cycle++ {
		texture, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
		if err != nil {
			return err
		}
		batch, err := graphics.NewSpriteBatch(device)
		if err != nil {
			return err
		}
		if cycle == 0 {
			if err := g.manager.Dispose(true); !errors.Is(err, interop.ErrChildrenAlive) {
				return fmt.Errorf("parent-before-child result: %w", err)
			}
			wrongThread := make(chan error, 1)
			go func() { wrongThread <- texture.DisposeByBoolean(true) }()
			if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
				return fmt.Errorf("wrong-thread texture disposal result: %w", err)
			}
			g.result.WrongThreadChecks++
			if err := texture.DisposeByBoolean(true); err != nil {
				return fmt.Errorf("owner-thread retry: %w", err)
			}
			g.result.OwnerThreadRetries++
		} else if cycle == 1 {
			// # The inherited Dispose() must reach the DERIVED body
			//
			// GraphicsResource::Dispose() is `callvirt Dispose(bool)`, which
			// dispatches to Texture2D's override and releases the native
			// texture. A Go forwarding that called the COMPOSED BASE's
			// DisposeByNone instead would set the flag, raise Disposing and
			// leak the CNA texture -- and every managed observable would agree
			// with the correct version, because the flag and the event are the
			// base's either way.
			//
			// The one observable that disagrees is the native handle. After the
			// derived body runs, the resource is destroyed and a later
			// operation reports ErrDisposed; after the base body runs, the
			// handle is still live and the same operation SUCCEEDS. That is why
			// this control lives here and not in a managed unit test.
			if err := texture.DisposeByNone(); err != nil {
				return fmt.Errorf("inherited Dispose(): %w", err)
			}
			if !texture.IsDisposed() {
				return errors.New("inherited Dispose() left IsDisposed false; the base half runs in a finally")
			}
			var sink bytes.Buffer
			if err := texture.SaveAsPng(&sink, 4, 4); !errors.Is(err, interop.ErrDisposed) {
				return fmt.Errorf("SaveAsPng after the inherited Dispose() = %v, want ErrDisposed: "+
					"the native texture is still alive, so Dispose() reached the composed base's slot instead of Texture2D's override", err)
			}
			g.result.InheritedDisposeVirtualChecks++
		} else if err := texture.DisposeByBoolean(true); err != nil {
			return err
		}
		if err := texture.DisposeByBoolean(true); err != nil {
			g.result.ObservedDoubleFree++
			return fmt.Errorf("double texture Dispose was not idempotent: %w", err)
		}
		if err := batch.DisposeByBoolean(true); err != nil {
			return err
		}
		if err := batch.DisposeByBoolean(true); err != nil {
			g.result.ObservedDoubleFree++
			return fmt.Errorf("double SpriteBatch Dispose was not idempotent: %w", err)
		}
		g.result.TextureCycles++
		g.result.SpriteBatchCycles++
		runtime.GC()
		g.result.GCStressPoints++
	}
	return nil
}

// removeDeviceHandler is the removal half of the event loop above, split out
// because Go has no way to name a method value's receiver twice in a map.
func removeDeviceHandler(device *graphics.GraphicsDevice, name string, subscription framework.EventSubscription) error {
	switch name {
	case "Disposing":
		return device.RemoveDisposingHandler(subscription)
	case "DeviceLost":
		return device.RemoveDeviceLostHandler(subscription)
	case "DeviceReset":
		return device.RemoveDeviceResetHandler(subscription)
	case "DeviceResetting":
		return device.RemoveDeviceResettingHandler(subscription)
	}
	return fmt.Errorf("unknown device event %q", name)
}

func verifyKeyboardPlayerIndexSnapshots() error {
	baseline, err := input.KeyboardGetStateByNone()
	if err != nil {
		return fmt.Errorf("read process keyboard baseline: %w", err)
	}
	for _, playerIndex := range playerIndexFixtures() {
		state, err := input.KeyboardGetStateByPlayerIndex(playerIndex)
		if err != nil {
			return fmt.Errorf("read process keyboard state for PlayerIndex(%d): %w", playerIndex, err)
		}
		if !input.KeyboardStateOperatorEqualityByKeyboardStateAndKeyboardState(baseline, state) {
			return fmt.Errorf("PlayerIndex(%d) selected a different keyboard snapshot", playerIndex)
		}
	}
	return nil
}

func verifyKeyboardUnavailable(stage string) error {
	baseline, baselineError := input.KeyboardGetStateByNone()
	if baselineError == nil {
		return fmt.Errorf("KeyboardGetStateByNone unexpectedly succeeded %s", stage)
	}
	for _, playerIndex := range playerIndexFixtures() {
		state, err := input.KeyboardGetStateByPlayerIndex(playerIndex)
		if err == nil {
			return fmt.Errorf("KeyboardGetStateByPlayerIndex(%d) unexpectedly succeeded %s", playerIndex, stage)
		}
		if state != baseline || err.Error() != baselineError.Error() {
			return fmt.Errorf("KeyboardGetStateByPlayerIndex(%d) used a different unavailable-runtime path %s", playerIndex, stage)
		}
	}
	return nil
}

func playerIndexFixtures() []framework.PlayerIndex {
	return []framework.PlayerIndex{
		framework.PlayerIndexOne,
		framework.PlayerIndexTwo,
		framework.PlayerIndexThree,
		framework.PlayerIndexFour,
		framework.PlayerIndex(12345),
	}
}

func (g *stressGame) Update(_ *framework.Game, _ framework.GameTime) error {
	switch g.scenario {
	case "callback-error":
		return callbackSentinel
	case "callback-panic":
		panic("native stress callback panic")
	default:
		return nil
	}
}

func (g *stressGame) Draw(host *framework.Game, _ framework.GameTime) error {
	if g.scenario == "sprite-draw" {
		if err := g.drawEverySpriteOverload(); err != nil {
			return err
		}
	}
	if g.scenario == "device-state" {
		if err := g.exerciseDeviceState(); err != nil {
			return err
		}
	}
	if g.scenario == "render-target" {
		if err := g.exerciseRenderTarget(); err != nil {
			return err
		}
	}
	if g.scenario == "content" {
		if err := g.exerciseContent(host); err != nil {
			return err
		}
	}
	if g.scenario == "index-buffer" {
		if err := g.exerciseIndexBuffer(); err != nil {
			return err
		}
	}
	if g.scenario == "vertex-buffer" {
		if err := g.exerciseVertexBuffer(host); err != nil {
			return err
		}
	}
	if g.scenario == "adapter" {
		if err := g.exerciseAdapter(); err != nil {
			return err
		}
	}
	if g.scenario == "sprite-font" {
		if err := g.exerciseSpriteFont(host); err != nil {
			return err
		}
	}
	if g.scenario == "texture-volume" {
		if err := g.exerciseTextureVolume(); err != nil {
			return err
		}
	}
	if g.scenario == "presentation" {
		if err := g.exercisePresentation(); err != nil {
			return err
		}
	}
	return host.Exit()
}

// isNativeRefusal reports whether an error is CNA answering "this renderer
// cannot do that" rather than the binding getting something wrong.
//
// It matches on the CNA result code, not on the message: the message is
// documentation and may be reworded, while CNA_RESULT_NOT_SUPPORTED is part of
// the ABI. A refusal is recorded as a renderer limitation; anything else is a
// defect and fails the scenario.
func isNativeRefusal(err error) bool {
	var native *interop.NativeError
	return errors.As(err, &native) && native.Code == 6
}

// renderTargetClearColor is the deterministic content the render-target
// semantic test writes and reads back. It is a colour no other value in the
// scenario produces, so a readback that matched by accident would have to
// produce these exact four bytes.
var renderTargetClearColor = framework.NewColorByInt32AndInt32AndInt32AndInt32(203, 67, 21, 255)

// exerciseRenderTarget is the render-target semantic slice, end to end, inside
// a draw callback -- the only moment CNA lends a device handle out.
//
//	create -> bind -> clear -> unbind -> read back THROUGH THE Texture2D SURFACE
//
// The last step is the point. `Texture2DGetDataBySliceOfT` takes a
// Texture2DReference, and passing a *RenderTarget2D to it is the Go spelling of
// C#'s `renderTarget` flowing into a Texture2D position. It proves the
// substitution and the render-target contents with one call, because the pixels
// come back through the base's member.
//
// # Two renderers, two outcomes, both recorded
//
// CNA permits a render target to be CREATED on a backend with no real
// off-screen storage: creation succeeds, RendererAvailable is false, and
// binding reports NOT_SUPPORTED. The HEADLESS artifact is such a backend and the
// SOFTWARE one is not, so this scenario asserts what each can actually do and
// counts them separately. A binding that failed on a renderer that HAS storage
// is a defect; one that failed on a renderer that does not is the documented
// contract.
func (g *stressGame) exerciseRenderTarget() error {
	device := g.device
	const size = 8

	target, err := graphics.NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32(device, size, size)
	if err != nil {
		return fmt.Errorf("NewRenderTarget2D: %w", err)
	}
	g.result.RenderTargetCreations++

	// The Texture2D half of its surface answers before anything is bound, from
	// the description CNA applied.
	if target.Width() != size || target.Height() != size {
		return fmt.Errorf("render target is %dx%d, want %dx%d", target.Width(), target.Height(), size, size)
	}
	if target.Bounds() != framework.NewRectangle(0, 0, size, size) {
		return fmt.Errorf("render target Bounds = %+v", target.Bounds())
	}
	if target.LevelCount() < 1 {
		return fmt.Errorf("render target LevelCount = %d", target.LevelCount())
	}
	if got := target.ToString(); got != "Microsoft.Xna.Framework.Graphics.RenderTarget2D" {
		return fmt.Errorf("render target ToString = %q; the CLR `this` must reach the outermost object across three composition links", got)
	}
	if target.RenderTargetUsage() != graphics.RenderTargetUsageDiscardContents {
		return fmt.Errorf("render target usage = %v, want the constructor's DiscardContents default", target.RenderTargetUsage())
	}
	g.result.RenderTargetDescriptionChecks++

	// It satisfies the Texture2D parameter position, which is the substitution
	// under test. The assignment is the proof; it does not compile otherwise.
	var asTexture graphics.Texture2DReference = target
	if asTexture == nil {
		return errors.New("a RenderTarget2D does not satisfy Texture2DReference")
	}
	g.result.RenderTargetSubstitutionChecks++

	bindErr := device.SetRenderTargetByRenderTarget2D(target)
	if bindErr != nil {
		// The documented refusal on a backend with no off-screen storage. It is
		// recorded rather than treated as a pass or a failure.
		g.result.RenderTargetBindRefusals++
		fmt.Fprintf(os.Stderr, "render-target bind refused: %v\n", bindErr)
		return target.DisposeByNone()
	}
	g.result.RenderTargetBinds++

	if err := device.ClearByColor(renderTargetClearColor); err != nil {
		return fmt.Errorf("Clear into the render target: %w", err)
	}
	if err := device.SetRenderTargetByRenderTarget2D(nil); err != nil {
		return fmt.Errorf("restore the back buffer: %w", err)
	}
	g.result.RenderTargetUnbinds++

	// The readback, through the BASE's member. It is the only step that needs
	// the renderer to be able to copy a colour attachment back to the CPU, and
	// that is a per-renderer capability rather than a per-binding one: the
	// HEADLESS artifact binds and clears and then refuses this with
	//
	//	Texture2D::GetData: this graphics renderer cannot read a render
	//	target's colour attachment back to the CPU
	//
	// so the refusal is counted and the slice continues. A pixel check is
	// evidence only where the renderer can produce one.
	pixels := make([]framework.Color, size*size)
	readErr := graphics.Texture2DGetDataBySliceOfT(target, pixels)
	switch {
	case readErr == nil:
		for index, pixel := range pixels {
			if pixel != renderTargetClearColor {
				return fmt.Errorf("render target pixel %d = %+v, want the cleared %+v", index, pixel, renderTargetClearColor)
			}
		}
		g.result.RenderTargetPixelChecks++
	case isNativeRefusal(readErr):
		g.result.RenderTargetReadbackRefusals++
		fmt.Fprintf(os.Stderr, "render-target readback refused: %v\n", readErr)
	default:
		return fmt.Errorf("GetData through the Texture2D surface: %w", readErr)
	}

	// And a SpriteBatch draws it, which is the seven live substitutability
	// positions exercised rather than only asserted.
	batch, batchErr := graphics.NewSpriteBatch(device)
	if batchErr != nil {
		return fmt.Errorf("NewSpriteBatch: %w", batchErr)
	}
	if err := batch.BeginByNone(); err != nil {
		return fmt.Errorf("Begin: %w", err)
	}
	if err := batch.DrawByTexture2DAndVector2AndColor(target, framework.Vector2{X: 1, Y: 1},
		framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)); err != nil {
		return fmt.Errorf("Draw a render target as a texture: %w", err)
	}
	if err := batch.End(); err != nil {
		return fmt.Errorf("End: %w", err)
	}
	g.result.RenderTargetSpriteDraws++
	if err := batch.DisposeByNone(); err != nil {
		return fmt.Errorf("SpriteBatch disposal: %w", err)
	}

	// Disposal through the INHERITED member, and the idempotence the
	// GraphicsResource flag gives it.
	disposals := 0
	if _, err := target.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		disposals++
		if sender != any(target) {
			return errors.New("Disposing announced something other than the render target")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("AddDisposingHandler: %w", err)
	}
	if err := target.DisposeByNone(); err != nil {
		return fmt.Errorf("render target disposal: %w", err)
	}
	if err := target.DisposeByNone(); err != nil {
		return fmt.Errorf("second render target disposal: %w", err)
	}
	if disposals != 1 || !target.IsDisposed() {
		return fmt.Errorf("Disposing raised %d times, IsDisposed=%t", disposals, target.IsDisposed())
	}
	g.result.RenderTargetDisposalChecks++
	return nil
}

// exerciseDeviceState round-trips GraphicsDevice's render state through the
// LIVE device, inside a draw callback, which is the only moment CNA lends a
// device handle out.
//
// Every write is read back from the device rather than from anything CNA-Go
// holds, which is the whole point: the reference answers these getters from a
// managed cache its own constructor fills, CNA-Go cannot fill one because it
// does not create the device, and a projection that cached anyway would pass a
// test that only compared what it had just been given.
func (g *stressGame) exerciseDeviceState() error {
	device := g.device

	factor := framework.NewColorByInt32AndInt32AndInt32AndInt32(12, 34, 56, 78)
	if err := device.SetBlendFactor(factor); err != nil {
		return fmt.Errorf("SetBlendFactor: %w", err)
	}
	readFactor, err := device.BlendFactor()
	if err != nil {
		return fmt.Errorf("BlendFactor: %w", err)
	}
	if readFactor != factor {
		return fmt.Errorf("BlendFactor round trip = %+v, want %+v", readFactor, factor)
	}
	g.result.DeviceStateRoundTrips++

	// Foundation 60. The three state-object properties, through the LIVE
	// device: a null is the reference's ArgumentNullException, a set pushes the
	// descriptor to CNA and FREEZES the object, and the getter answers with the
	// very object that was set rather than with a fresh one built from values.
	if _, initial := device.BlendState(); initial != nil {
		return fmt.Errorf("BlendState on a live device: %w", initial)
	}
	if refusal := device.SetBlendState(nil); refusal == nil ||
		!strings.Contains(refusal.Error(), "This method does not accept null for this parameter.") {
		return fmt.Errorf("SetBlendState(nil) = %v, want the ArgumentNullException message", refusal)
	}
	g.result.DeviceStateObjectRefusals++

	ownBlend := graphics.NewBlendState()
	if err := ownBlend.SetColorSourceBlend(graphics.BlendSourceAlpha); err != nil {
		return fmt.Errorf("a fresh BlendState refused a write: %w", err)
	}
	if err := device.SetBlendState(ownBlend); err != nil {
		return fmt.Errorf("SetBlendState: %w", err)
	}
	readBlend, readErr := device.BlendState()
	if readErr != nil {
		return fmt.Errorf("BlendState: %w", readErr)
	}
	if readBlend != ownBlend {
		return errors.New("BlendState returned a different object; the getter answers with the one the setter was given")
	}
	if err := ownBlend.SetColorSourceBlend(graphics.BlendOne); err == nil {
		return errors.New("a bound BlendState accepted a write; Apply raises isBound")
	}
	if ownBlend.GraphicsDevice() != device {
		return errors.New("Apply did not store the device as the state's parent")
	}
	g.result.DeviceStateObjectBinds++

	if err := device.SetDepthStencilState(graphics.DepthStencilStateDepthRead()); err != nil {
		return fmt.Errorf("SetDepthStencilState: %w", err)
	}
	if err := device.SetRasterizerState(graphics.RasterizerStateCullNone()); err != nil {
		return fmt.Errorf("SetRasterizerState: %w", err)
	}
	readDepth, _ := device.DepthStencilState()
	readRaster, _ := device.RasterizerState()
	if readDepth != graphics.DepthStencilStateDepthRead() || readRaster != graphics.RasterizerStateCullNone() {
		return errors.New("the device did not cache the preset objects it was given")
	}
	g.result.DeviceStateObjectBinds += 2

	// And the two state-carrying Begin overloads, which reach CNA through one
	// route carrying all four descriptors and then perform SetRenderState's
	// managed half.
	stateBatch, stateBatchErr := graphics.NewSpriteBatch(device)
	if stateBatchErr != nil {
		return fmt.Errorf("NewSpriteBatch: %w", stateBatchErr)
	}
	if err := stateBatch.BeginBySpriteSortModeAndBlendState(
		graphics.SpriteSortModeDeferred, graphics.BlendStateAdditive()); err != nil {
		return fmt.Errorf("Begin(sortMode, blendState): %w", err)
	}
	if bound, _ := device.BlendState(); bound != graphics.BlendStateAdditive() {
		return errors.New("Begin did not apply its blend state to the device")
	}
	// SetRenderState substitutes the reference's defaults for the three nulls.
	if depth, _ := device.DepthStencilState(); depth != graphics.DepthStencilStateNone() {
		return errors.New("Begin did not substitute DepthStencilState.None for a null")
	}
	if raster, _ := device.RasterizerState(); raster != graphics.RasterizerStateCullCounterClockwise() {
		return errors.New("Begin did not substitute RasterizerState.CullCounterClockwise for a null")
	}
	if err := stateBatch.End(); err != nil {
		return fmt.Errorf("End after a state Begin: %w", err)
	}
	g.result.SpriteBatchStateBegins++

	if err := stateBatch.BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerState(
		graphics.SpriteSortModeDeferred, graphics.BlendStateOpaque(), graphics.SamplerStatePointClamp(),
		graphics.DepthStencilStateDefault(), graphics.RasterizerStateCullClockwise()); err != nil {
		return fmt.Errorf("Begin with four states: %w", err)
	}
	if err := stateBatch.End(); err != nil {
		return fmt.Errorf("End after the four-state Begin: %w", err)
	}
	g.result.SpriteBatchStateBegins++
	if err := stateBatch.DisposeByNone(); err != nil {
		return fmt.Errorf("state SpriteBatch disposal: %w", err)
	}

	// Foundation 61. The four collections, through the LIVE device.
	textures, texturesErr := device.Textures()
	if texturesErr != nil {
		return fmt.Errorf("Textures: %w", texturesErr)
	}
	again, _ := device.Textures()
	if textures != again {
		return errors.New("Textures returned two objects; the reference holds one per device")
	}
	vertexTextures, _ := device.VertexTextures()
	if vertexTextures == textures {
		return errors.New("Textures and VertexTextures are the same collection")
	}
	g.result.DeviceCollectionIdentityChecks++

	// The refusal must be the PROJECTION's ArgumentOutOfRangeException, not
	// CNA's own out-of-range answer: an off-by-one bound would still refuse,
	// through the wrong guard, with the wrong identity.
	for _, index := range []int32{-1, 16} {
		_, readErr := textures.Item(index)
		writeErr := textures.SetItem(index, nil)
		for name, err := range map[string]error{"get": readErr, "set": writeErr} {
			if err == nil {
				return fmt.Errorf("Textures[%d] %s was accepted; the indexer refuses out of range", index, name)
			}
			if !strings.Contains(err.Error(), "index is out of range") {
				return fmt.Errorf("Textures[%d] %s = %v, want the projection's ArgumentOutOfRangeException", index, name, err)
			}
		}
	}
	g.result.DeviceCollectionRangeChecks++

	// An empty slot answers nil, a bound one answers the texture that was
	// bound, and unbinding empties it again.
	slotTexture, slotErr := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
	if slotErr != nil {
		return fmt.Errorf("slot texture: %w", slotErr)
	}
	if err := textures.SetItem(0, slotTexture); err != nil {
		return fmt.Errorf("Textures[0] = texture: %w", err)
	}
	readSlot, readSlotErr := textures.Item(0)
	if readSlotErr != nil {
		return fmt.Errorf("Textures[0]: %w", readSlotErr)
	}
	if readSlot == nil {
		return errors.New("Textures[0] answered nil for a slot it had just bound")
	}
	if readSlot.Format() != slotTexture.Format() || readSlot.LevelCount() != slotTexture.LevelCount() {
		return errors.New("Textures[0] answered with a different texture")
	}
	if err := textures.SetItem(0, nil); err != nil {
		return fmt.Errorf("Textures[0] = nil: %w", err)
	}
	if empty, err := textures.Item(0); err != nil || empty != nil {
		return fmt.Errorf("Textures[0] after unbinding = %v, %v; want nil and no error", empty, err)
	}
	g.result.DeviceCollectionTextureRoundTrips++
	if err := slotTexture.DisposeByNone(); err != nil {
		return fmt.Errorf("slot texture disposal: %w", err)
	}

	samplers, samplersErr := device.SamplerStates()
	if samplersErr != nil {
		return fmt.Errorf("SamplerStates: %w", samplersErr)
	}
	// A slot nothing has written answers with what CNA reports, materialised.
	reported, reportedErr := samplers.Item(0)
	if reportedErr != nil || reported == nil {
		return fmt.Errorf("SamplerStates[0] = %v, %v", reported, reportedErr)
	}
	if err := samplers.SetItem(0, graphics.SamplerStatePointClamp()); err != nil {
		return fmt.Errorf("SamplerStates[0] = PointClamp: %w", err)
	}
	if bound, err := samplers.Item(0); err != nil || bound != graphics.SamplerStatePointClamp() {
		return fmt.Errorf("SamplerStates[0] after setting = %v, %v; the getter answers with the object that was set", bound, err)
	}
	if refusal := samplers.SetItem(0, nil); refusal == nil {
		return errors.New("SamplerStates[0] = nil was accepted")
	}
	g.result.DeviceCollectionSamplerRoundTrips++

	// Foundation 62. The six device events, subscribed against the LIVE device.
	// Registration is what is proved here: CNA raises DeviceLost, DeviceReset
	// and DeviceResetting only when a renderer really loses or resets a device,
	// which neither qualified artifact can be made to do, and it raises
	// Disposing from a disposal this scenario must not perform on a device the
	// Game owns and goes on using.
	deviceRaises := 0
	subscriptions := 0
	for name, add := range map[string]func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error){
		"Disposing": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDisposingHandler(h)
		},
		"DeviceLost": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceLostHandler(h)
		},
		"DeviceReset": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceResetHandler(h)
		},
		"DeviceResetting": func(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
			return device.AddDeviceResettingHandler(h)
		},
	} {
		subscription, err := add(func(sender any, args *framework.EventArgs) error {
			deviceRaises++
			return nil
		})
		if err != nil {
			return fmt.Errorf("Add%sHandler: %w", name, err)
		}
		subscriptions++
		if err := removeDeviceHandler(device, name, subscription); err != nil {
			return fmt.Errorf("Remove%sHandler: %w", name, err)
		}
	}
	createdSubscription, createdErr := device.AddResourceCreatedHandler(
		func(sender any, args *graphics.ResourceCreatedEventArgs) error { return nil })
	if createdErr != nil {
		return fmt.Errorf("AddResourceCreatedHandler: %w", createdErr)
	}
	destroyedSubscription, destroyedErr := device.AddResourceDestroyedHandler(
		func(sender any, args *graphics.ResourceDestroyedEventArgs) error { return nil })
	if destroyedErr != nil {
		return fmt.Errorf("AddResourceDestroyedHandler: %w", destroyedErr)
	}
	subscriptions += 2
	if err := device.RemoveResourceCreatedHandler(createdSubscription); err != nil {
		return fmt.Errorf("RemoveResourceCreatedHandler: %w", err)
	}
	if err := device.RemoveResourceDestroyedHandler(destroyedSubscription); err != nil {
		return fmt.Errorf("RemoveResourceDestroyedHandler: %w", err)
	}
	if subscriptions != 6 {
		return fmt.Errorf("%d device event subscriptions, want six", subscriptions)
	}
	g.result.DeviceEventSubscriptions += subscriptions
	// Every registration must be released before cna_game_destroy succeeds, and
	// the manager's disposal is what does it. A leaked registration would make
	// the whole isolated cycle fail at teardown, which is the strongest form
	// this control can take.
	g.result.DeviceEventRegistrationChecks++

	if err := device.SetMultiSampleMask(0x0f0f0f0f); err != nil {
		return fmt.Errorf("SetMultiSampleMask: %w", err)
	}
	mask, err := device.MultiSampleMask()
	if err != nil {
		return fmt.Errorf("MultiSampleMask: %w", err)
	}
	if mask != 0x0f0f0f0f {
		return fmt.Errorf("MultiSampleMask round trip = %#x, want 0x0f0f0f0f", mask)
	}
	g.result.DeviceStateRoundTrips++

	if err := device.SetReferenceStencil(37); err != nil {
		return fmt.Errorf("SetReferenceStencil: %w", err)
	}
	stencil, err := device.ReferenceStencil()
	if err != nil {
		return fmt.Errorf("ReferenceStencil: %w", err)
	}
	if stencil != 37 {
		return fmt.Errorf("ReferenceStencil round trip = %d, want 37", stencil)
	}
	g.result.DeviceStateRoundTrips++

	scissor := framework.NewRectangle(3, 5, 64, 48)
	if err := device.SetScissorRectangle(scissor); err != nil {
		return fmt.Errorf("SetScissorRectangle: %w", err)
	}
	readScissor, err := device.ScissorRectangle()
	if err != nil {
		return fmt.Errorf("ScissorRectangle: %w", err)
	}
	if readScissor != scissor {
		return fmt.Errorf("ScissorRectangle round trip = %+v, want %+v", readScissor, scissor)
	}
	g.result.DeviceStateRoundTrips++

	// The viewport's getter has been projected since Foundation 1 and its
	// setter arrives now, so this round trip crosses two milestones' work.
	viewport := graphics.NewViewportByInt32AndInt32AndInt32AndInt32(2, 4, 320, 200)
	viewport.SetMinDepth(0.25)
	viewport.SetMaxDepth(0.75)
	if err := device.SetViewport(viewport); err != nil {
		return fmt.Errorf("SetViewport: %w", err)
	}
	readViewport, err := device.Viewport()
	if err != nil {
		return fmt.Errorf("Viewport: %w", err)
	}
	if readViewport.X() != 2 || readViewport.Y() != 4 || readViewport.Width() != 320 || readViewport.Height() != 200 ||
		readViewport.MinDepth() != 0.25 || readViewport.MaxDepth() != 0.75 {
		return fmt.Errorf("Viewport round trip = %s", readViewport.ToString())
	}
	g.result.DeviceStateRoundTrips++

	// Three read-only members. Their values come from the device rather than
	// from a constant here, so what is asserted is that each is a value the
	// enum actually declares -- not a number CNA-Go chose.
	profile, err := device.GraphicsProfile()
	if err != nil {
		return fmt.Errorf("GraphicsProfile: %w", err)
	}
	if profile != graphics.GraphicsProfileReach && profile != graphics.GraphicsProfileHiDef {
		return fmt.Errorf("GraphicsProfile = %d, which is not a declared value", profile)
	}
	g.result.DeviceStateReadOnlyChecks++

	status, err := device.GraphicsDeviceStatus()
	if err != nil {
		return fmt.Errorf("GraphicsDeviceStatus: %w", err)
	}
	if status != graphics.GraphicsDeviceStatusNormal && status != graphics.GraphicsDeviceStatusLost &&
		status != graphics.GraphicsDeviceStatusNotReset {
		return fmt.Errorf("GraphicsDeviceStatus = %d, which is not a declared value", status)
	}
	g.result.DeviceStateReadOnlyChecks++

	disposed, err := device.IsDisposed()
	if err != nil {
		return fmt.Errorf("IsDisposed: %w", err)
	}
	if disposed {
		return errors.New("the live device reports itself disposed from inside a draw callback")
	}
	g.result.DeviceStateReadOnlyChecks++

	// Both masked Clear overloads, through the same route, with the same mask.
	options := graphics.ClearOptionsTarget | graphics.ClearOptionsDepthBuffer
	if err := device.ClearByClearOptionsAndColorAndSingleAndInt32(options, factor, 1, 0); err != nil {
		return fmt.Errorf("Clear(ClearOptions, Color, ...): %w", err)
	}
	g.result.DeviceStateClearCalls++
	if err := device.ClearByClearOptionsAndVector4AndSingleAndInt32(
		options, framework.NewVector4BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1), 1, 0); err != nil {
		return fmt.Errorf("Clear(ClearOptions, Vector4, ...): %w", err)
	}
	g.result.DeviceStateClearCalls++

	// CNA refuses a non-finite depth with CNA_RESULT_INVALID_ARGUMENT, and the
	// refusal must surface rather than be swallowed. A projection that ignored
	// the result would look identical from Go.
	nonFinite := float32(math.Inf(1))
	if refusal := device.ClearByClearOptionsAndColorAndSingleAndInt32(options, factor, nonFinite, 0); refusal == nil {
		return errors.New("a non-finite clear depth was accepted")
	}
	g.result.DeviceStateClearRefusals++

	if err := device.PresentByNone(); err != nil {
		return fmt.Errorf("Present: %w", err)
	}
	g.result.DeviceStatePresentCalls++

	// The device's display mode. Its two computed members are reproduced from
	// the dimensions rather than taken from CNA's own aspect ratio, so what is
	// checked here is that they agree with the dimensions CNA reported.
	mode, err := device.DisplayMode()
	if err != nil {
		return fmt.Errorf("DisplayMode: %w", err)
	}
	if mode.Width() <= 0 || mode.Height() <= 0 {
		return fmt.Errorf("DisplayMode = %s, which has a non-positive dimension", mode.ToString())
	}
	g.result.DeviceStateDisplayModeChecks++
	safe := mode.TitleSafeArea()
	wantAspect := float32(mode.Width()) / float32(mode.Height())
	if safe.X != 0 || safe.Y != 0 || safe.Width != mode.Width() || safe.Height != mode.Height() ||
		mode.AspectRatio() != wantAspect {
		return fmt.Errorf("DisplayMode computed members disagree with its dimensions: %s", mode.ToString())
	}
	g.result.DeviceStateDisplayModeChecks++

	// An EMPTY texture, created from a stated size and format rather than
	// decoded. Three per cycle: the three-argument constructor, the
	// five-argument one with the same defaults, and one with a mip chain.
	for index, create := range []func() (*graphics.Texture2D, error){
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 32, 16)
		},
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
				device, 32, 16, false, graphics.SurfaceFormatColor)
		},
		func() (*graphics.Texture2D, error) {
			return graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
				device, 64, 64, true, graphics.SurfaceFormatColor)
		},
	} {
		created, createErr := create()
		if createErr != nil {
			return fmt.Errorf("empty texture %d: %w", index, createErr)
		}
		width, height := created.Width(), created.Height()
		wantWidth, wantHeight := int32(32), int32(16)
		if index == 2 {
			wantWidth, wantHeight = 64, 64
		}
		if width != wantWidth || height != wantHeight {
			return fmt.Errorf("empty texture %d = %dx%d, want %dx%d", index, width, height, wantWidth, wantHeight)
		}
		if err := created.DisposeByBoolean(true); err != nil {
			return fmt.Errorf("empty texture %d disposal: %w", index, err)
		}
		g.result.DeviceStateTextureCreations++
	}

	// A texture encoded to PNG and to JPEG, then decoded back at a REQUESTED
	// size through both of the reference's zoom modes.
	//
	// The encoded bytes are checked for their format signature rather than for
	// a length, because a length proves only that something was written: PNG
	// begins with the eight-byte magic and JPEG with SOI. A projection that sent
	// XNA's own format identity through -- SaveAsPng passes 2, and CNA's PNG is
	// 0 while its JPEG is 1 -- would encode a JPEG under the PNG member and this
	// is what catches it.
	source, err := graphics.Texture2DFromStreamByGraphicsDeviceAndStream(device, bytes.NewReader(g.data))
	if err != nil {
		return fmt.Errorf("source texture: %w", err)
	}
	var png, jpeg bytes.Buffer
	if err := source.SaveAsPng(&png, 32, 32); err != nil {
		return fmt.Errorf("SaveAsPng: %w", err)
	}
	if got := png.Bytes(); len(got) < 8 || string(got[1:4]) != "PNG" {
		return fmt.Errorf("SaveAsPng wrote %d bytes that do not begin with the PNG signature", len(got))
	}
	g.result.DeviceStateEncodeChecks++
	if err := source.SaveAsJpeg(&jpeg, 32, 32); err != nil {
		return fmt.Errorf("SaveAsJpeg: %w", err)
	}
	if got := jpeg.Bytes(); len(got) < 3 || got[0] != 0xff || got[1] != 0xd8 {
		return fmt.Errorf("SaveAsJpeg wrote %d bytes that do not begin with the JPEG SOI marker", len(got))
	}
	g.result.DeviceStateEncodeChecks++

	for _, zoom := range []bool{false, true} {
		decoded, decodeErr := graphics.Texture2DFromStreamByGraphicsDeviceAndStreamAndInt32AndInt32AndBoolean(
			device, bytes.NewReader(png.Bytes()), 24, 24, zoom)
		if decodeErr != nil {
			return fmt.Errorf("sized decode (zoom=%t): %w", zoom, decodeErr)
		}
		width, height := decoded.Width(), decoded.Height()
		if width != 24 || height != 24 {
			return fmt.Errorf("sized decode (zoom=%t) = %dx%d, want 24x24", zoom, width, height)
		}
		if err := decoded.DisposeByBoolean(true); err != nil {
			return fmt.Errorf("sized decode disposal: %w", err)
		}
		g.result.DeviceStateDecodeSizeChecks++
	}

	// The one guard SaveAsImage's prologue has that Go can express: a nil
	// destination carries Microsoft's own sentence.
	if refusal := source.SaveAsPng(nil, 8, 8); refusal == nil ||
		!strings.Contains(refusal.Error(), "This method does not accept null for this parameter.") {
		return fmt.Errorf("SaveAsPng to a nil writer = %v, want the reference's message", refusal)
	}
	g.result.DeviceStateEncodeRefusals++
	if err := source.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("source texture disposal: %w", err)
	}

	// Typed transfers, through the generic-method projection. Each writes a
	// pattern to the live texture and reads it back FROM the texture, so a
	// projection that kept a managed copy would pass a test that compared its
	// own input.
	transferTexture, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 4, 4)
	if err != nil {
		return fmt.Errorf("transfer texture: %w", err)
	}
	written := make([]framework.Color, 16)
	for index := range written {
		written[index] = framework.NewColorByInt32AndInt32AndInt32AndInt32(
			int32(index*16%256), int32(index*8%256), int32(index*4%256), 255)
	}
	if err := graphics.Texture2DSetDataBySliceOfT(transferTexture, written); err != nil {
		return fmt.Errorf("SetData: %w", err)
	}
	readBack := make([]framework.Color, 16)
	if err := graphics.Texture2DGetDataBySliceOfT(transferTexture, readBack); err != nil {
		return fmt.Errorf("GetData: %w", err)
	}
	for index := range written {
		if readBack[index] != written[index] {
			return fmt.Errorf("texel %d round-tripped as %v, want %v", index, readBack[index], written[index])
		}
	}
	g.result.DeviceStateTransferRoundTrips++

	// A WINDOW of the same array, through the three-argument overload.
	windowed := make([]framework.Color, 16)
	if err := graphics.Texture2DGetDataBySliceOfTAndInt32AndInt32(transferTexture, windowed, 0, 16); err != nil {
		return fmt.Errorf("windowed GetData: %w", err)
	}
	if windowed[0] != written[0] || windowed[15] != written[15] {
		return errors.New("a windowed transfer did not reproduce the full surface")
	}
	g.result.DeviceStateTransferRoundTrips++

	// A RECTANGLE, through the five-argument overload the other two funnel
	// into. Two by two at the origin is four texels of the sixteen.
	region := framework.NewRectangle(0, 0, 2, 2)
	corner := make([]framework.Color, 4)
	if err := graphics.Texture2DGetDataByInt32AndNullableOfRectangleAndSliceOfTAndInt32AndInt32(
		transferTexture, 0, &region, corner, 0, 4); err != nil {
		return fmt.Errorf("rectangle GetData: %w", err)
	}
	if corner[0] != written[0] {
		return fmt.Errorf("the rectangle transfer's first texel is %v, want %v", corner[0], written[0])
	}
	g.result.DeviceStateTransferRoundTrips++

	// The two refusals the projection makes before CNA is reached: an element
	// type outside the eighteen CNA declares, and a transfer window that leaves
	// the array.
	if refusal := graphics.Texture2DSetDataBySliceOfT(transferTexture, []int64{1, 2, 3, 4}); refusal == nil ||
		!strings.Contains(refusal.Error(), "is not one of the eighteen element types") {
		return fmt.Errorf("an unsupported element type produced %v", refusal)
	}
	g.result.DeviceStateTransferRefusals++
	if refusal := graphics.Texture2DSetDataBySliceOfTAndInt32AndInt32(transferTexture, written, 8, 16); refusal == nil {
		return errors.New("a transfer window past the end of the array was accepted")
	}
	g.result.DeviceStateTransferRefusals++
	if err := transferTexture.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("transfer texture disposal: %w", err)
	}

	// The two refusals the projection makes itself, before CNA is reached: a
	// nil device carries Microsoft's own sentence, and a negative dimension is
	// refused rather than converted into an enormous uint32.
	if _, refusal := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(nil, 4, 4); refusal == nil ||
		!strings.Contains(refusal.Error(), "The GraphicsDevice must not be null when creating new resources.") {
		return fmt.Errorf("nil-device texture creation = %v, want the reference's message", refusal)
	}
	g.result.DeviceStateTextureRefusals++
	if _, refusal := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, -1, 4); refusal == nil {
		return errors.New("a negative texture width was accepted")
	}
	g.result.DeviceStateTextureRefusals++

	// The owner-thread policy reaches these members too. A second goroutine
	// must be refused before it can touch the device.
	wrongThread := make(chan error, 1)
	go func() { _, callErr := wrongThread2Read(device); wrongThread <- callErr }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("device state from a second goroutine = %v, want ErrWrongThread", err)
	}
	g.result.DeviceStateWrongThreadHits++
	return nil
}

// wrongThread2Read is the one device read the wrong-thread proof makes, kept
// separate so the goroutine body stays a single call.
func wrongThread2Read(device *graphics.GraphicsDevice) (framework.Color, error) {
	return device.BlendFactor()
}

// drawEverySpriteOverload submits one command through each of the profile's
// seven Draw overloads, inside one begin/end pair, in a live draw callback.
//
// Four of the seven reach cna_sprite_batch_submit_scaled_many and three reach
// cna_sprite_batch_submit_many, and the two counters below are what separate
// them: a projection that routed a rectangle overload through the scaled family
// would draw something, and would draw it in the wrong place, which no return
// value reports.
func (g *stressGame) drawEverySpriteOverload() error {
	batch := g.spriteBatch
	texture := g.spriteTexture
	white := framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)
	source := framework.NewRectangle(0, 0, 16, 16)
	destination := framework.NewRectangle(4, 8, 32, 24)
	origin := framework.NewVector2BySingleAndSingle(2, 3)

	// A second Begin inside a pair is the reference's EndMustBeCalledBeforeBegin
	// throw, and it is checked while a pair is genuinely open rather than
	// simulated.
	if err := batch.BeginByNone(); err != nil {
		return err
	}
	if again := batch.BeginByNone(); again == nil ||
		!strings.Contains(again.Error(), "Begin cannot be called again until End has been successfully called.") {
		return fmt.Errorf("second Begin = %v, want the InvalidOperationException message", again)
	}
	g.result.SpriteDrawPairGuardChecks++

	scaled := []func() error{
		func() error {
			return batch.DrawByTexture2DAndVector2AndColor(texture, framework.NewVector2BySingleAndSingle(1, 2), white)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(texture, framework.NewVector2BySingleAndSingle(3, 4), &source, white)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				texture, framework.NewVector2BySingleAndSingle(5, 6), &source, white, 0.5, origin, 2, graphics.SpriteEffectsFlipHorizontally, 0.25)
		},
		func() error {
			return batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				texture, framework.NewVector2BySingleAndSingle(7, 8), &source, white, 0.5, origin, framework.NewVector2BySingleAndSingle(2, 3), graphics.SpriteEffectsFlipVertically, 0.75)
		},
	}
	for index, draw := range scaled {
		if err := draw(); err != nil {
			return fmt.Errorf("scaled Draw overload %d: %w", index, err)
		}
		g.result.SpriteDrawScaledSubmits++
	}

	destinations := []func() error{
		func() error { return batch.DrawByTexture2DAndRectangleAndColor(texture, destination, white) },
		func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(texture, destination, &source, white)
		},
		func() error {
			return batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColorAndSingleAndVector2AndSpriteEffectsAndSingle(
				texture, destination, &source, white, 0.5, origin, graphics.SpriteEffectsFlipHorizontally, 0.5)
		},
	}
	for index, draw := range destinations {
		if err := draw(); err != nil {
			return fmt.Errorf("destination Draw overload %d: %w", index, err)
		}
		g.result.SpriteDrawDestinationSubmits++
	}

	// A nil source rectangle is the static nullRectangle the three-argument
	// overloads pass, and it must reach the same route rather than being
	// refused. Both families are exercised with one.
	if err := batch.DrawByTexture2DAndVector2AndNullableOfRectangleAndColor(texture, framework.NewVector2BySingleAndSingle(9, 10), nil, white); err != nil {
		return fmt.Errorf("scaled Draw with a nil source: %w", err)
	}
	g.result.SpriteDrawScaledSubmits++
	if err := batch.DrawByTexture2DAndRectangleAndNullableOfRectangleAndColor(texture, destination, nil, white); err != nil {
		return fmt.Errorf("destination Draw with a nil source: %w", err)
	}
	g.result.SpriteDrawDestinationSubmits++

	if err := batch.End(); err != nil {
		return err
	}
	// The pair is closed, so the guard is armed again -- which is what proves
	// End cleared the flag rather than only flushing.
	if after := batch.DrawByTexture2DAndVector2AndColor(texture, framework.Vector2{}, white); after == nil ||
		!strings.Contains(after.Error(), "Begin must be called successfully before a Draw can be called.") {
		return fmt.Errorf("Draw after End = %v, want the InvalidOperationException message", after)
	}
	g.result.SpriteDrawOutsidePairChecks++
	return nil
}

func (g *stressGame) UnloadContent(_ *framework.Game) error {
	if g.manager != nil {
		if err := g.manager.Dispose(true); err != nil {
			return err
		}
		postDispose := framework.DisplayOrientationLandscapeLeft | framework.DisplayOrientationLandscapeRight
		g.manager.SetSupportedOrientations(postDispose)
		if got := g.manager.SupportedOrientations(); got != postDispose {
			return fmt.Errorf("post-disposal SupportedOrientations = %d, want %d", got, postDispose)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// The optional per-hook frame-boundary overrides, against the pinned runtime.
// ---------------------------------------------------------------------------

// frameHookGame is a consumer whose callback object declares optional frame
// overrides. Nothing here names an unexported binding type: the opt-in is the
// exported method, and which methods this type declares is decided by the
// `subset` flag at construction, exactly as a consumer decides it by writing
// them or not.
//
// The claim it proves is the one no in-process test can: that the CNA runtime
// really delivers the installed hooks at the measured frame positions, really
// skips draw AND end_draw on a frame the override refused, and really never
// delivers a hook whose capability the object does not declare.
type frameHookGame struct {
	// declaresEndDraw records whether THIS object declares the EndDraw
	// override, which is what decides whether an admitted frame should deliver
	// the end_draw hook at all.
	declaresEndDraw bool
	order           []string
	updates         int
	baseAnswers     []bool
	baseCalls       int
	result          counters
	ownerGoroutine  string
	goroutines      map[string]bool
	uninstalled     []string
}

func (g *frameHookGame) record(entry string) {
	g.order = append(g.order, entry)
	g.goroutines[currentGoroutineLabel()] = true
}

func (g *frameHookGame) Initialize(*framework.Game) error {
	g.ownerGoroutine = currentGoroutineLabel()
	g.goroutines = map[string]bool{}
	return nil
}

func (g *frameHookGame) LoadContent(*framework.Game) error { return nil }

// Update ends the run after enough frames that both the refused and the
// admitted branch are exercised several times.
func (g *frameHookGame) Update(host *framework.Game, _ framework.GameTime) error {
	g.updates++
	g.record("update")
	if g.updates >= 8 {
		return host.Exit()
	}
	return nil
}

func (g *frameHookGame) Draw(*framework.Game, framework.GameTime) error {
	g.record("draw")
	g.result.FrameHookAdmittedFrames++
	if g.declaresEndDraw {
		g.result.FrameHookEndDrawExpected++
	}
	return nil
}

func (g *frameHookGame) UnloadContent(*framework.Game) error { return nil }

// BeginDraw is the one optional override both scenarios declare. It calls the
// base explicitly -- the Go projection of base.BeginDraw() -- records what the
// base answered, and then refuses every other frame with its OWN answer.
//
// The base always admits the frame here, because no IGraphicsDeviceManager is
// registered. So a skipped draw is positive proof that the override's answer
// and not the base's is what reached CNA.
func (g *frameHookGame) BeginDraw(host *framework.Game) (bool, error) {
	g.record("begin_draw")
	g.result.FrameHookBeginDrawHits++
	answer, err := host.BeginDraw()
	if err != nil {
		return false, err
	}
	g.baseCalls++
	g.baseAnswers = append(g.baseAnswers, answer)
	if g.result.FrameHookBeginDrawHits%2 == 0 {
		g.result.FrameHookRefusedFrames++
		g.record("refused")
		return false, nil
	}
	return true, nil
}

// frameHookAllGame adds the other three overrides. The subset scenario uses
// frameHookGame directly and therefore declares only BeginDraw, which is what
// makes "an omitted capability never receives its hook" measurable.
type frameHookAllGame struct{ frameHookGame }

func (g *frameHookAllGame) BeginRun(host *framework.Game) error {
	g.record("begin_run")
	g.result.FrameHookBeginRunHits++
	return host.BeginRun()
}

func (g *frameHookAllGame) EndRun(host *framework.Game) error {
	g.record("end_run")
	g.result.FrameHookEndRunHits++
	return host.EndRun()
}

func (g *frameHookAllGame) EndDraw(host *framework.Game) error {
	g.record("end_draw")
	g.result.FrameHookEndDrawHits++
	return host.EndDraw()
}

// frameHookSubsetGame declares only BeginDraw, and additionally records any of
// the other three arriving. They cannot: their CNA_GameFrameHooks members are
// left NULL because no capability was declared for them, and a null member is
// one the canonical header says is simply not called. The recording exists so
// that if they ever did arrive, the run would fail loudly rather than quietly
// gaining behaviour.
type frameHookSubsetGame struct{ frameHookGame }

func runFrameHookChild(scenario string) error {
	var callbacks framework.GameCallbacks
	var state *frameHookGame
	switch scenario {
	case "frame-hook-override":
		every := &frameHookAllGame{frameHookGame{declaresEndDraw: true}}
		callbacks, state = every, &every.frameHookGame
	case "frame-hook-subset":
		subset := &frameHookSubsetGame{}
		callbacks, state = subset, &subset.frameHookGame
	default:
		return fmt.Errorf("unknown frame-hook scenario %q", scenario)
	}
	host, err := framework.NewGame(callbacks)
	if err != nil {
		return err
	}
	if err := host.Run(); err != nil {
		return err
	}
	if err := state.verify(scenario); err != nil {
		return err
	}
	runtime.GC()
	state.result.GCStressPoints++
	data, _ := json.Marshal(state.result)
	fmt.Println(string(data))
	return nil
}

// verify holds every claim the scenario exists to prove.
func (g *frameHookGame) verify(scenario string) error {
	if len(g.uninstalled) != 0 {
		g.result.FrameHookUninstalledHits += len(g.uninstalled)
		return fmt.Errorf("hooks %v arrived for capabilities that were never declared", g.uninstalled)
	}
	// Every override call reached the base exactly once, and the base admitted
	// every frame -- which is what makes a skipped draw the override's doing.
	if g.baseCalls != g.result.FrameHookBeginDrawHits {
		return fmt.Errorf("the override ran %d times and called the base %d times", g.result.FrameHookBeginDrawHits, g.baseCalls)
	}
	for i, answer := range g.baseAnswers {
		if !answer {
			return fmt.Errorf("base BeginDraw refused frame %d with no manager registered", i)
		}
	}
	if g.result.FrameHookBeginDrawHits == 0 {
		return errors.New("begin_draw was never delivered, so the hook was not installed")
	}
	g.result.FrameHookBaseCallChecks++

	// A refused frame delivers neither draw nor end_draw; an admitted one
	// delivers draw and then end_draw, in that order.
	refused, admitted := 0, 0
	for i := 0; i < len(g.order); i++ {
		if g.order[i] != "begin_draw" {
			continue
		}
		next := g.order[i+1:]
		if len(next) > 0 && next[0] == "refused" {
			refused++
			for _, entry := range next[1:] {
				if entry == "begin_draw" {
					break
				}
				if entry == "draw" || entry == "end_draw" {
					return fmt.Errorf("a refused frame still delivered %q; order=%v", entry, g.order)
				}
			}
			continue
		}
		admitted++
		if len(next) == 0 || next[0] != "draw" {
			return fmt.Errorf("an admitted frame did not deliver draw next; order=%v", g.order)
		}
		if scenario == "frame-hook-override" && (len(next) < 2 || next[1] != "end_draw") {
			return fmt.Errorf("an admitted frame did not deliver end_draw after draw; order=%v", g.order)
		}
	}
	if refused == 0 || admitted == 0 {
		return fmt.Errorf("scenario exercised %d refused and %d admitted frames; both branches are required", refused, admitted)
	}
	g.result.FrameHookSkipChecks++

	if len(g.goroutines) != 1 || !g.goroutines[g.ownerGoroutine] {
		return fmt.Errorf("frame hooks were delivered on %d goroutines, want the owner goroutine only", len(g.goroutines))
	}

	switch scenario {
	case "frame-hook-override":
		// begin_run before every frame, end_run after every frame, once each.
		if g.result.FrameHookBeginRunHits != 1 || g.result.FrameHookEndRunHits != 1 {
			return fmt.Errorf("begin_run fired %d times and end_run %d, want exactly once each",
				g.result.FrameHookBeginRunHits, g.result.FrameHookEndRunHits)
		}
		if g.order[0] != "begin_run" {
			return fmt.Errorf("delivery order %v does not start with begin_run", g.order)
		}
		if g.order[len(g.order)-1] != "end_run" {
			return fmt.Errorf("delivery order %v does not end with end_run", g.order)
		}
		g.result.FrameHookOrderChecks++
		g.result.FrameHookOverrideCycles++
	case "frame-hook-subset":
		// The three undeclared capabilities installed nothing, so none of
		// their hooks can appear anywhere in the order.
		for _, entry := range g.order {
			switch entry {
			case "begin_run", "end_run", "end_draw":
				g.result.FrameHookUninstalledHits++
				return fmt.Errorf("hook %q was delivered to a callback object that declares only BeginDraw", entry)
			}
		}
		g.result.FrameHookOrderChecks++
		g.result.FrameHookSubsetCycles++
	}
	return nil
}

func encodedPNG() []byte {
	value := image.NewRGBA(image.Rect(0, 0, 2, 2))
	value.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	value.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	value.SetRGBA(0, 1, color.RGBA{B: 255, A: 255})
	value.SetRGBA(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var output bytes.Buffer
	if err := png.Encode(&output, value); err != nil {
		panic(err)
	}
	return output.Bytes()
}

// addCounters sums every counter mechanically.
//
// It used to be a hand-written list of one line per field, and that list was a
// silent single point of failure: a counter added to the struct but not to the
// list stayed at zero in the aggregate report while its scenario ran perfectly,
// so the evidence would read "this was never measured" for something that was.
// Foundation 45 hit exactly that. Reflection cannot drift, and the panic below
// is a programmer invariant -- every counter is an int by construction.
func addCounters(target *counters, value counters) {
	destination := reflect.ValueOf(target).Elem()
	source := reflect.ValueOf(value)
	for index := 0; index < destination.NumField(); index++ {
		field := destination.Field(index)
		if field.Kind() != reflect.Int {
			panic(fmt.Sprintf("counters.%s is %s; every counter must be an int", destination.Type().Field(index).Name, field.Kind()))
		}
		field.SetInt(field.Int() + source.Field(index).Int())
	}
}

// ---------------------------------------------------------------------------
// Game's timing and presentation state, against the pinned runtime.
// ---------------------------------------------------------------------------

// timingGame configures a Game before Run and then, from inside a lifecycle
// callback on the owner thread, drives every timing setter against the live
// native loop.
//
// The claim it proves is the one no in-process test can: these are not stored
// values that look like they work. They reach CNA, CNA accepts them, and the
// values a Game was configured with BEFORE Run are what cna_game_create was
// handed -- a rejected target step would have failed creation outright.
type timingGame struct {
	result counters
}

func (g *timingGame) Initialize(host *framework.Game) error {
	// Inside a lifecycle callback on the owner thread: every setter must reach
	// the live native game.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(100000)); err != nil {
		return fmt.Errorf("SetTargetElapsedTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(0)); err != nil {
		return fmt.Errorf("SetInactiveSleepTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetIsFixedTimeStep(false); err != nil {
		return fmt.Errorf("SetIsFixedTimeStep during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SetIsMouseVisible(true); err != nil {
		return fmt.Errorf("SetIsMouseVisible during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.SuppressDraw(); err != nil {
		return fmt.Errorf("SuppressDraw during a run: %w", err)
	}
	g.result.TimingSettersApplied++
	if err := host.ResetElapsedTime(); err != nil {
		return fmt.Errorf("ResetElapsedTime during a run: %w", err)
	}
	g.result.TimingSettersApplied++

	// The managed field is what the getter reads, and it holds what was stored
	// regardless of where the value went afterwards.
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("TargetElapsedTime = %d after setting it to 100000", got)
	}
	if host.IsFixedTimeStep() || !host.IsMouseVisible() {
		return fmt.Errorf("flags = %t/%t after setting them false/true", host.IsFixedTimeStep(), host.IsMouseVisible())
	}

	// From another goroutine CNA answers CNA_RESULT_THREAD, and the projection
	// reports it rather than pretending the loop was told.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- host.SetIsFixedTimeStep(true) }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a timing setter from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.TimingWrongThreadChecks++

	// The argument checks are the reference's own and come before anything
	// native: TargetElapsedTime rejects zero, InactiveSleepTime accepts it.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(0)); err == nil {
		return errors.New("TargetElapsedTime accepted zero; the reference compares with op_LessThanOrEqual")
	}
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(0)); err != nil {
		return fmt.Errorf("InactiveSleepTime rejected zero: %w", err)
	}
	g.result.TimingSettersApplied-- // the second accepted set is not a seventh setting
	g.result.TimingRangeChecks++
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("a rejected TargetElapsedTime still stored: %d", got)
	}
	// Restore the accounting: exactly six settings reached the loop.
	g.result.TimingSettersApplied++
	return nil
}

func (g *timingGame) LoadContent(*framework.Game) error                       { return nil }
func (g *timingGame) Update(host *framework.Game, _ framework.GameTime) error { return host.Exit() }
func (g *timingGame) Draw(*framework.Game, framework.GameTime) error          { return nil }
func (g *timingGame) UnloadContent(*framework.Game) error                     { return nil }

// runTimingChild configures the Game BEFORE Run, which is the state a real
// consumer sets a frame rate in, and proves the configured values are what the
// native game is created with.
func runTimingChild() error {
	game := &timingGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	// A non-default, valid step. If cna_game_create did not accept it -- or if
	// the create path still passed a literal -- this run would not start.
	if err := host.SetTargetElapsedTime(framework.TimeSpanFromTicks(83333)); err != nil {
		return fmt.Errorf("configure before Run: %w", err)
	}
	if err := host.SetIsMouseVisible(true); err != nil {
		return fmt.Errorf("configure mouse visibility before Run: %w", err)
	}
	if err := host.SetInactiveSleepTime(framework.TimeSpanFromTicks(1000)); err != nil {
		return fmt.Errorf("configure inactive sleep before Run: %w", err)
	}
	if err := host.Run(); err != nil {
		return err
	}
	game.result.TimingCreatedWithConfig++
	game.result.TimingCycles++
	// After the run the managed state is still readable, and still says what
	// the last successful set stored.
	if got := host.TargetElapsedTime().Ticks(); got != 100000 {
		return fmt.Errorf("TargetElapsedTime after the run = %d, want the value Initialize stored", got)
	}
	// With no live native game the setters store and report success again.
	if err := host.SetIsFixedTimeStep(true); err != nil {
		return fmt.Errorf("SetIsFixedTimeStep after the run: %w", err)
	}
	if !host.IsFixedTimeStep() {
		return errors.New("a post-run set did not reach the managed field")
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// windowGame proves GameWindow against a LIVE native game, which is the half
// the pure-Go tests structurally cannot reach: with no native game every
// guarded member answers its fallback, so only a run can show that the same
// member reads a real window when there is one.
type windowGame struct {
	result   counters
	window   *framework.GameWindow
	deviceOK bool
}

func (g *windowGame) Initialize(host *framework.Game) error {
	window := host.Window()
	// The identity a consumer captured BEFORE Run is the identity the callback
	// sees. A projection that allocated per call would hand out a second
	// object here and silently orphan every subscription made before Run.
	if window != g.window {
		return errors.New("Game.Window returned a different object inside a callback than before Run")
	}
	g.result.WindowIdentityChecks++

	// The UNGUARDED member now succeeds. Before Run it reported the
	// reference's NullReferenceException; the difference between the two is
	// the whole point of the measured guard split.
	bounds, err := window.ClientBounds()
	if err != nil {
		return fmt.Errorf("ClientBounds with a live window: %w", err)
	}
	g.result.WindowLiveReads++
	// The measured SIZE is the renderer's, not the binding's. The HEADLESS
	// artifact reports a 0x0 client rectangle while its graphics device
	// reports an 800x480 viewport, so a positive size is COUNTED rather than
	// required: requiring it would make this scenario a renderer test that
	// fails for a reason CNA-Go cannot fix, and asserting 0x0 would bake a
	// headless artifact's answer into the contract.
	if bounds.Width > 0 && bounds.Height > 0 {
		g.result.WindowPositiveClientBounds++
	}

	// The guarded members answer without failing. Their VALUES are the
	// platform's and are not asserted -- a headless window legitimately has no
	// screen device name and a zero handle -- but a failure would be a real
	// defect and is.
	if _, err := window.Handle(); err != nil {
		return fmt.Errorf("Handle with a live window: %w", err)
	}
	if _, err := window.ScreenDeviceName(); err != nil {
		return fmt.Errorf("ScreenDeviceName with a live window: %w", err)
	}
	g.result.WindowLiveReads += 2

	// AllowUserResizing round-trips only if the platform honours it. Both
	// calls must report success; whether the value came back is COUNTED rather
	// than required, because a headless window has no resize grip to grant.
	before, err := window.AllowUserResizing()
	if err != nil {
		return fmt.Errorf("AllowUserResizing with a live window: %w", err)
	}
	if err := window.SetAllowUserResizing(!before); err != nil {
		return fmt.Errorf("SetAllowUserResizing with a live window: %w", err)
	}
	after, err := window.AllowUserResizing()
	if err != nil {
		return fmt.Errorf("AllowUserResizing after assignment: %w", err)
	}
	if after == !before {
		g.result.WindowResizeRoundTrips++
	}
	g.result.WindowLiveReads++

	// Title: the managed field is authoritative, and the assignment reaches
	// the live loop.
	if got := window.Title(); got != "" {
		return fmt.Errorf("Title = %q at the start of a run, want the constructor's String.Empty", got)
	}
	if err := window.SetTitleProperty("cna-go window stress"); err != nil {
		return fmt.Errorf("SetTitleProperty with a live window: %w", err)
	}
	if got := window.Title(); got != "cna-go window stress" {
		return fmt.Errorf("Title = %q after assignment", got)
	}
	g.result.WindowLiveReads++

	// The suppression proof, and it needs a live run to exist at all.
	//
	// set_Title's guard is `if (this.title != value)`, so an UNCHANGED
	// assignment performs no platform call. From a non-owner goroutine that is
	// observable and nothing else is: an unchanged assignment succeeds because
	// it never reaches the boundary, while a changed one is refused for being
	// off the owner thread. A projection that dropped the guard would fail the
	// first of the two.
	unchanged := make(chan error, 1)
	go func() { unchanged <- window.SetTitleProperty("cna-go window stress") }()
	if err := <-unchanged; err != nil {
		return fmt.Errorf("an unchanged title from a non-owner goroutine reported %v; the guard should have stopped it before the boundary", err)
	}
	g.result.WindowTitleSuppressions++
	changed := make(chan error, 1)
	go func() { changed <- window.SetTitleProperty("a different title") }()
	if err := <-changed; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a changed title from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.WindowWrongThreadChecks++
	// The refused assignment still stored the managed field before it reached
	// the boundary, exactly as the reference's own order does: the store
	// precedes the SetTitle call.
	if got := window.Title(); got != "a different title" {
		return fmt.Errorf("Title = %q after a refused native call; the reference stores before it calls", got)
	}

	// The screen-device-change pair, which is the other unguarded family.
	name, err := window.ScreenDeviceName()
	if err != nil {
		return fmt.Errorf("ScreenDeviceName before a device change: %w", err)
	}
	if err := window.BeginScreenDeviceChange(false); err != nil {
		return fmt.Errorf("BeginScreenDeviceChange with a live window: %w", err)
	}
	if err := window.EndScreenDeviceChangeByString(name); err != nil {
		return fmt.Errorf("EndScreenDeviceChange with a live window: %w", err)
	}
	g.result.WindowScreenDeviceChanges++

	// CurrentOrientation is the reference's constant and stays it even with a
	// live window: the reference never asks the platform in this profile.
	if got := window.CurrentOrientation(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("CurrentOrientation = %v with a live window, want the reference's constant Default", got)
	}
	window.SetSupportedOrientations(framework.DisplayOrientationPortrait)
	if got := window.CurrentOrientation(); got != framework.DisplayOrientationDefault {
		return fmt.Errorf("CurrentOrientation = %v after SetSupportedOrientations; the reference's body is one `ret`", got)
	}
	g.result.WindowLiveReads++

	if runtime, ok := interop.CurrentRuntime(); ok {
		deliveries := runtime.GameWindowEventDeliveries()
		g.result.WindowEventClientSize += deliveries[interop.GameWindowEventClientSizeChanged]
		g.result.WindowEventOrientation += deliveries[interop.GameWindowEventOrientationChanged]
		g.result.WindowEventScreenDevice += deliveries[interop.GameWindowEventScreenDeviceNameChanged]
		g.deviceOK = true
	}
	return nil
}

func (g *windowGame) LoadContent(*framework.Game) error                       { return nil }
func (g *windowGame) Update(host *framework.Game, _ framework.GameTime) error { return host.Exit() }
func (g *windowGame) Draw(*framework.Game, framework.GameTime) error          { return nil }
func (g *windowGame) UnloadContent(*framework.Game) error                     { return nil }

// runWindowChild measures the window before, during and after one run. The
// before and after halves are what prove the guard split is a split: the same
// member answers a fallback with no native window and a real value with one.
func runWindowChild() error {
	game := &windowGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	window := host.Window()
	if window == nil {
		return errors.New("Game.Window is nil after construction; the reference's EnsureHost runs in the constructor")
	}
	if host.Window() != window {
		return errors.New("Game.Window is not a stable identity before Run")
	}
	game.window = window
	game.result.WindowIdentityChecks++

	// Before Run: the five guarded members answer the reference's own
	// fallbacks and report nothing.
	handle, err := window.Handle()
	if err != nil || handle != 0 {
		return fmt.Errorf("Handle before Run = %#x, %v; want IntPtr.Zero and no failure", handle, err)
	}
	allow, err := window.AllowUserResizing()
	if err != nil || allow {
		return fmt.Errorf("AllowUserResizing before Run = %t, %v; want false and no failure", allow, err)
	}
	if err := window.SetAllowUserResizing(true); err != nil {
		return fmt.Errorf("SetAllowUserResizing before Run: %w", err)
	}
	name, err := window.ScreenDeviceName()
	if err != nil || name != "" {
		return fmt.Errorf("ScreenDeviceName before Run = %q, %v; want String.Empty and no failure", name, err)
	}
	if err := window.SetTitleMethod("before"); err != nil {
		return fmt.Errorf("SetTitle before Run: %w", err)
	}
	game.result.WindowGuardedFallbacks += 5

	// And the three unguarded ones report the reference's failure.
	if _, err := window.ClientBounds(); err == nil {
		return errors.New("ClientBounds succeeded before Run; the reference dereferences a null form")
	}
	if err := window.BeginScreenDeviceChange(true); err == nil {
		return errors.New("BeginScreenDeviceChange succeeded before Run")
	}
	if err := window.EndScreenDeviceChangeByStringAndInt32AndInt32("screen", 320, 240); err == nil {
		return errors.New("EndScreenDeviceChange succeeded before Run")
	}
	game.result.WindowUnguardedFailures += 3

	if err := host.Run(); err != nil {
		return err
	}
	if !game.deviceOK {
		return errors.New("the window scenario never reached a live runtime")
	}
	game.result.WindowCycles++

	// After the run the window is the same object, the managed title survives,
	// and every member is back to its no-native-window behaviour.
	if host.Window() != window {
		return errors.New("Game.Window changed identity after the run")
	}
	game.result.WindowIdentityChecks++
	if got := window.Title(); got != "a different title" {
		return fmt.Errorf("Title after the run = %q; the managed field outlives the native window", got)
	}
	if _, err := window.ClientBounds(); err == nil {
		return errors.New("ClientBounds succeeded after the run; the native window is gone")
	}
	game.result.WindowUnguardedFailures++
	handle, err = window.Handle()
	if err != nil || handle != 0 {
		return fmt.Errorf("Handle after the run = %#x, %v; want IntPtr.Zero and no failure", handle, err)
	}
	game.result.WindowGuardedFallbacks++

	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// frameStepGame drives a native game a frame at a time, with no loop anywhere.
//
// It declares BeginDraw and EndDraw so the optional frame hooks are installed
// too: a frame step has to reach the same hook positions a looped frame does,
// and a projection that only worked inside cna_game_run would not.
type frameStepGame struct {
	result counters

	initializes, loads, updates, draws, unloads int
	beginDraws, endDraws                        int

	// suppressNextDraw asks BeginDraw to refuse exactly one frame, which is
	// how SuppressDraw's effect is observed without a loop.
	host             *framework.Game
	tickFromCallback error
	callbackAsked    bool
}

func (g *frameStepGame) Initialize(host *framework.Game) error {
	g.initializes++
	// A frame step from INSIDE a lifecycle callback must be refused: CNA
	// answers CNA_RESULT_INVALID_STATE because a frame step called from within
	// a frame would re-enter the loop it is part of.
	if !g.callbackAsked {
		g.callbackAsked = true
		g.tickFromCallback = host.Tick()
	}
	return nil
}

func (g *frameStepGame) LoadContent(*framework.Game) error { g.loads++; return nil }

func (g *frameStepGame) Update(*framework.Game, framework.GameTime) error {
	g.updates++
	return nil
}

func (g *frameStepGame) Draw(*framework.Game, framework.GameTime) error {
	g.draws++
	return nil
}

func (g *frameStepGame) UnloadContent(*framework.Game) error {
	g.unloads++
	return nil
}

func (g *frameStepGame) BeginDraw(host *framework.Game) (bool, error) {
	g.beginDraws++
	return host.BeginDraw()
}

func (g *frameStepGame) EndDraw(host *framework.Game) error {
	g.endDraws++
	return host.EndDraw()
}

// runFrameStepChild is the whole lifecycle in one isolated process: create by
// stepping, step, dispose, and step again.
func runFrameStepChild() error {
	game := &frameStepGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	game.host = host

	runtimeOf := func() (*interop.Runtime, bool) { return interop.CurrentRuntime() }

	// Before the first step there is no native game at all.
	if _, live := runtimeOf(); live {
		return errors.New("a native runtime was current before the first frame step")
	}
	game.result.FrameStepSessionChecks++

	// TICK ONE. It creates the session and delivers exactly one Update and one
	// Draw -- and NO Initialize and no LoadContent, because Game::Tick has no
	// initialization step and CNA's does not either.
	if err := host.Tick(); err != nil {
		return fmt.Errorf("first Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.initializes != 0 || game.loads != 0 {
		return fmt.Errorf("a first Tick delivered %d initializes and %d loads; Tick does not initialize",
			game.initializes, game.loads)
	}
	if game.updates != 1 {
		return fmt.Errorf("a first Tick delivered %d updates, want one", game.updates)
	}
	game.result.FrameStepTickInitChecks++
	current, live := runtimeOf()
	if !live || !current.HasStandaloneSession() {
		return errors.New("the first Tick did not create a standalone session")
	}
	game.result.FrameStepSessionChecks++

	if err := host.Tick(); err != nil {
		return fmt.Errorf("second Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.initializes != 0 {
		return fmt.Errorf("a second Tick initialized %d times", game.initializes)
	}

	// RUN ONE FRAME. CNA initializes on first use, which the reference does
	// NOT do -- recorded as a measured upstream difference rather than hidden.
	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("first RunOneFrame: %w", err)
	}
	game.result.FrameStepRunOneFrames++
	if game.initializes != 1 || game.loads != 1 {
		return fmt.Errorf("a first RunOneFrame delivered %d initializes and %d loads, want one each",
			game.initializes, game.loads)
	}
	// The in-callback refusal was taken during that Initialize.
	if game.tickFromCallback == nil {
		return errors.New("a Tick from inside a lifecycle callback succeeded; CNA refuses a re-entrant frame step")
	}
	game.result.FrameStepCallbackRefusals++

	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("second RunOneFrame: %w", err)
	}
	game.result.FrameStepRunOneFrames++
	if game.initializes != 1 {
		return fmt.Errorf("initialization ran %d times across a session, want exactly one", game.initializes)
	}
	game.result.FrameStepInitializations = game.initializes

	// A step from another goroutine is refused: the goroutine that took the
	// first step owns the session's OS thread for its whole life.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- host.Tick() }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a Tick from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	game.result.FrameStepWrongThreadChecks++

	// SUPPRESS DRAW. The next step updates and does not draw.
	drawsBefore := game.draws
	if err := host.SuppressDraw(); err != nil {
		return fmt.Errorf("SuppressDraw: %w", err)
	}
	if err := host.Tick(); err != nil {
		return fmt.Errorf("suppressed Tick: %w", err)
	}
	game.result.FrameStepTicks++
	if game.draws != drawsBefore {
		return fmt.Errorf("a suppressed Tick drew %d times", game.draws-drawsBefore)
	}
	game.result.FrameStepSuppressChecks++

	// EXIT, from outside a lifecycle callback -- a state that could not exist
	// before a session could outlive a call. CNA's request-exit also suppresses
	// the next draw, so the step after it updates and does not draw.
	if err := host.Exit(); err != nil {
		return fmt.Errorf("Exit outside a callback: %w", err)
	}
	drawsBefore = game.draws
	if err := host.Tick(); err != nil {
		return fmt.Errorf("Tick after Exit: %w", err)
	}
	game.result.FrameStepTicks++
	if game.draws != drawsBefore {
		return fmt.Errorf("the step after Exit drew %d times; CNA's request-exit suppresses the next draw", game.draws-drawsBefore)
	}
	game.result.FrameStepExitChecks++

	game.result.FrameStepUpdates = game.updates
	game.result.FrameStepDraws = game.draws

	// DISPOSE ends the session, and CNA delivers UnloadContent and the exiting
	// signal from inside cna_game_destroy.
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	if game.unloads != 1 {
		return fmt.Errorf("Dispose delivered %d UnloadContent callbacks, want one", game.unloads)
	}
	if _, stillLive := runtimeOf(); stillLive {
		return errors.New("a native runtime was still current after Dispose")
	}
	game.result.FrameStepDisposeChecks++
	game.result.FrameStepSessionChecks++

	// And a step AFTER Dispose starts a fresh session, because Game keeps no
	// disposed flag -- the reference does not either, which is why Dispose is
	// not idempotent anywhere in this profile.
	if err := host.Tick(); err != nil {
		return fmt.Errorf("Tick after Dispose: %w", err)
	}
	if _, revived := runtimeOf(); !revived {
		return errors.New("a Tick after Dispose created no session")
	}
	game.result.FrameStepRecreationChecks++
	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("second Dispose: %w", err)
	}

	game.result.FrameStepCycles++
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// runFrameStepRunChild proves the ownership rule: Run ADOPTS a session a frame
// step created and does not destroy it, because whoever created the native game
// destroys it.
//
// This is the reference's own shape. XNA's Run calls host.Run() on a host the
// constructor already made, and CNA's Game::Run skips DoInitialize when
// hasInitialized_ is set -- so a stepped-then-run Game keeps one native game
// and one initialization.
type frameStepRunGame struct {
	initializes, updates int
	exitAfter            int
}

func (g *frameStepRunGame) Initialize(*framework.Game) error { g.initializes++; return nil }
func (g *frameStepRunGame) LoadContent(*framework.Game) error {
	return nil
}
func (g *frameStepRunGame) Update(host *framework.Game, _ framework.GameTime) error {
	g.updates++
	if g.updates >= g.exitAfter {
		return host.Exit()
	}
	return nil
}
func (g *frameStepRunGame) Draw(*framework.Game, framework.GameTime) error { return nil }
func (g *frameStepRunGame) UnloadContent(*framework.Game) error            { return nil }

func runFrameStepRunChild() error {
	game := &frameStepRunGame{exitAfter: 3}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	var result counters

	if err := host.RunOneFrame(); err != nil {
		return fmt.Errorf("RunOneFrame: %w", err)
	}
	if game.initializes != 1 {
		return fmt.Errorf("RunOneFrame initialized %d times, want one", game.initializes)
	}
	current, live := interop.CurrentRuntime()
	if !live || !current.HasStandaloneSession() {
		return errors.New("RunOneFrame created no standalone session")
	}

	// Run adopts it. The session is NOT re-created, so initialization does not
	// happen again, and Run does not destroy what it did not create.
	if err := host.Run(); err != nil {
		return fmt.Errorf("Run after a frame step: %w", err)
	}
	if game.initializes != 1 {
		return fmt.Errorf("Run re-initialized the adopted session: %d initializations", game.initializes)
	}
	current, stillLive := interop.CurrentRuntime()
	if !stillLive || !current.HasStandaloneSession() {
		return errors.New("Run destroyed a session it did not create")
	}
	result.FrameStepRunAfterStepCycle++

	if err := host.DisposeByNone(); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	if _, after := interop.CurrentRuntime(); after {
		return errors.New("Dispose left the adopted session alive")
	}
	runtime.GC()
	result.GCStressPoints++
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
	return nil
}

// graphicsManagerGame proves GraphicsDeviceManager's configuration surface
// against a LIVE native manager, which is the half the managed tests
// structurally cannot reach: with no native manager every setter stores and
// pushes nothing, so only a run can show that the value arrives.
type graphicsManagerGame struct {
	result counters
}

func (g *graphicsManagerGame) Initialize(host *framework.Game) error {
	manager, err := framework.NewGraphicsDeviceManager(host)
	if err != nil {
		return fmt.Errorf("NewGraphicsDeviceManager: %w", err)
	}

	// The constructor's own field initializers, on a manager that now has a
	// native one behind it.
	if manager.PreferredBackBufferWidth() != framework.GraphicsDeviceManagerDefaultBackBufferWidth() ||
		manager.PreferredBackBufferHeight() != framework.GraphicsDeviceManagerDefaultBackBufferHeight() ||
		!manager.SynchronizeWithVerticalRetrace() || manager.IsFullScreen() || manager.PreferMultiSampling() {
		return fmt.Errorf("constructor defaults: %dx%d vsync=%t fullscreen=%t multisample=%t",
			manager.PreferredBackBufferWidth(), manager.PreferredBackBufferHeight(),
			manager.SynchronizeWithVerticalRetrace(), manager.IsFullScreen(), manager.PreferMultiSampling())
	}
	if graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager) != graphics.DepthFormatDepth24 {
		return fmt.Errorf("PreferredDepthStencilFormat = %v, want Depth24",
			graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager))
	}
	g.result.ManagerDefaultChecks++

	// The six framework-typed setters. Each stores AND reaches CNA's manager;
	// a refused push would surface here rather than being swallowed.
	type setting struct {
		name  string
		apply func() error
		check func() bool
	}
	for _, s := range []setting{
		{"PreferredBackBufferWidth", func() error { return manager.SetPreferredBackBufferWidth(1024) },
			func() bool { return manager.PreferredBackBufferWidth() == 1024 }},
		{"PreferredBackBufferHeight", func() error { return manager.SetPreferredBackBufferHeight(576) },
			func() bool { return manager.PreferredBackBufferHeight() == 576 }},
		{"IsFullScreen", func() error { return manager.SetIsFullScreen(false) },
			func() bool { return !manager.IsFullScreen() }},
		{"SynchronizeWithVerticalRetrace", func() error { return manager.SetSynchronizeWithVerticalRetrace(false) },
			func() bool { return !manager.SynchronizeWithVerticalRetrace() }},
		{"PreferMultiSampling", func() error { return manager.SetPreferMultiSampling(true) },
			func() bool { return manager.PreferMultiSampling() }},
		{"SupportedOrientations", func() error {
			return manager.SetSupportedOrientations(framework.DisplayOrientationLandscapeLeft)
		}, func() bool { return manager.SupportedOrientations() == framework.DisplayOrientationLandscapeLeft }},
	} {
		if err := s.apply(); err != nil {
			return fmt.Errorf("Set%s against a live manager: %w", s.name, err)
		}
		if !s.check() {
			return fmt.Errorf("Set%s did not reach the managed field", s.name)
		}
		g.result.ManagerSettersApplied++
	}

	// The three Graphics-typed ones, which travel through internal/servicebridge
	// because the framework package cannot name their enums.
	if err := graphics.SetGraphicsDeviceManagerGraphicsProfile(manager, graphics.GraphicsProfileReach); err != nil {
		return fmt.Errorf("SetGraphicsProfile: %w", err)
	}
	if graphics.GraphicsDeviceManagerGraphicsProfile(manager) != graphics.GraphicsProfileReach {
		return errors.New("GraphicsProfile did not round-trip across the package boundary")
	}
	g.result.ManagerCrossPackageSets++
	if err := graphics.SetGraphicsDeviceManagerPreferredBackBufferFormat(manager, graphics.SurfaceFormatColor); err != nil {
		return fmt.Errorf("SetPreferredBackBufferFormat: %w", err)
	}
	if graphics.GraphicsDeviceManagerPreferredBackBufferFormat(manager) != graphics.SurfaceFormatColor {
		return errors.New("PreferredBackBufferFormat did not round-trip")
	}
	g.result.ManagerCrossPackageSets++
	if err := graphics.SetGraphicsDeviceManagerPreferredDepthStencilFormat(manager, graphics.DepthFormatDepth16); err != nil {
		return fmt.Errorf("SetPreferredDepthStencilFormat: %w", err)
	}
	if graphics.GraphicsDeviceManagerPreferredDepthStencilFormat(manager) != graphics.DepthFormatDepth16 {
		return errors.New("PreferredDepthStencilFormat did not round-trip")
	}
	g.result.ManagerCrossPackageSets++

	// The one validation, at its exact boundary, against a live manager: a
	// rejected value never reaches CNA either.
	if err := manager.SetPreferredBackBufferWidth(0); err == nil {
		return errors.New("SetPreferredBackBufferWidth(0) was accepted; the IL compares with bgt on zero")
	}
	if manager.PreferredBackBufferWidth() != 1024 {
		return fmt.Errorf("a rejected width stored: %d", manager.PreferredBackBufferWidth())
	}
	g.result.ManagerRangeChecks++

	// A setter from another goroutine is refused, exactly as the timing
	// setters and the window members are.
	wrongThread := make(chan error, 1)
	go func() { wrongThread <- manager.SetPreferMultiSampling(false) }()
	if err := <-wrongThread; !errors.Is(err, interop.ErrWrongThread) {
		return fmt.Errorf("a manager setter from a non-owner goroutine reported %v, want ErrWrongThread", err)
	}
	g.result.ManagerWrongThreadCheck++

	if err := manager.ApplyChanges(); err != nil {
		return fmt.Errorf("ApplyChanges: %w", err)
	}
	g.result.ManagerApplyChanges++

	// ToggleFullScreen flips through the projected setter, so the managed flag
	// follows the device.
	before := manager.IsFullScreen()
	if err := manager.ToggleFullScreen(); err != nil {
		return fmt.Errorf("ToggleFullScreen: %w", err)
	}
	if manager.IsFullScreen() == before {
		return errors.New("ToggleFullScreen did not flip the managed flag")
	}
	if err := manager.ToggleFullScreen(); err != nil {
		return fmt.Errorf("second ToggleFullScreen: %w", err)
	}
	if manager.IsFullScreen() != before {
		return errors.New("two ToggleFullScreen calls did not return to the starting state")
	}
	g.result.ManagerToggleChecks++

	// ------------------------------------------------------------------
	// Foundation 49. The two service registrations the constructor makes, and
	// what they unlock.
	// ------------------------------------------------------------------

	// Registration one: the manager itself, under the framework-package
	// IGraphicsDeviceManager contract. It is the manager object, not an
	// adapter, because that contract is nameable from the framework package.
	registeredManager, err := host.Services().GetService(
		reflect.TypeOf((*framework.IGraphicsDeviceManager)(nil)).Elem())
	if err != nil || registeredManager == nil {
		return fmt.Errorf("IGraphicsDeviceManager is not registered: %v %v", registeredManager, err)
	}
	if registeredManager != any(manager) {
		return errors.New("IGraphicsDeviceManager resolves to something other than the manager itself")
	}
	g.result.ManagerServiceChecks++

	// Registration two: an adapter over the manager, under the
	// Graphics-package IGraphicsDeviceService contract. It is an adapter
	// because no framework-package type can declare that contract's device
	// accessor, and its identity is stable across resolutions.
	registeredService, err := host.Services().GetService(
		reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if err != nil || registeredService == nil {
		return fmt.Errorf("IGraphicsDeviceService is not registered: %v %v", registeredService, err)
	}
	service, ok := registeredService.(graphics.IGraphicsDeviceService)
	if !ok {
		return errors.New("the registered device service does not satisfy the contract")
	}
	again, _ := host.Services().GetService(reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if again != registeredService {
		return errors.New("the device service adapter is not a stable identity")
	}
	g.result.ManagerServiceChecks++

	// A second manager is refused with the reference's own ArgumentException.
	if _, duplicateErr := framework.NewGraphicsDeviceManager(host); duplicateErr == nil {
		return errors.New("a second GraphicsDeviceManager was accepted")
	} else if !strings.Contains(duplicateErr.Error(), "A graphics device manager is already registered.") {
		return fmt.Errorf("a second manager reported %v, want the reference's message", duplicateErr)
	}
	g.result.ManagerDuplicateChecks++

	// THE PAYOFF. Game.GraphicsDevice has reported the reference's
	// InvalidOperationException since Foundation 43, because CNA-Go published
	// no service of its own and only a consumer could register one. It now
	// resolves the manager's.
	gameDevice, gameDeviceErr := graphics.GameGraphicsDevice(host)
	if gameDeviceErr != nil {
		return fmt.Errorf("Game.GraphicsDevice with a published service: %w", gameDeviceErr)
	}
	if gameDevice != service.GraphicsDevice() {
		return errors.New("Game.GraphicsDevice answered a device other than the published service's")
	}
	g.result.ManagerGameDeviceChecks++

	// And DrawableGameComponent.Initialize, which threw
	// MissingGraphicsDeviceService for the same reason, now resolves and
	// subscribes.
	component := framework.NewDrawableGameComponent(host)
	if err := component.Initialize(); err != nil {
		return fmt.Errorf("DrawableGameComponent.Initialize with a published service: %w", err)
	}
	componentDevice, componentErr := graphics.DrawableGameComponentGraphicsDevice(component)
	if componentErr != nil {
		return fmt.Errorf("DrawableGameComponent.GraphicsDevice: %w", componentErr)
	}
	if componentDevice != service.GraphicsDevice() {
		return errors.New("the component resolved a different device from the published service's")
	}
	if err := component.DisposeByBoolean(true); err != nil {
		return fmt.Errorf("component Dispose: %w", err)
	}
	g.result.ManagerDrawableChecks++

	// The four protected raisers reach a consumer's handlers on the live
	// manager, which is the surface the device service publishes.
	raises := map[string]int{}
	for name, add := range map[string]func(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error){
		"created":   manager.AddDeviceCreatedHandler,
		"resetting": manager.AddDeviceResettingHandler,
		"reset":     manager.AddDeviceResetHandler,
		"disposing": manager.AddDeviceDisposingHandler,
	} {
		key := name
		if _, err := add(func(sender any, args *framework.EventArgs) error {
			raises[key]++
			return nil
		}); err != nil {
			return fmt.Errorf("subscribe %s: %w", name, err)
		}
	}
	for name, raise := range map[string]func(any, *framework.EventArgs) error{
		"created":   manager.OnDeviceCreated,
		"resetting": manager.OnDeviceResetting,
		"reset":     manager.OnDeviceReset,
		"disposing": manager.OnDeviceDisposing,
	} {
		if err := raise(manager, framework.EventArgsEmpty()); err != nil {
			return fmt.Errorf("On%s: %w", name, err)
		}
		if raises[name] != 1 {
			return fmt.Errorf("On%s reached its handler %d times", name, raises[name])
		}
		g.result.ManagerEventRaiseChecks++
	}

	// CreateDevice is the operation that would raise DeviceCreated. It is one
	// of the three IGraphicsDeviceManager witnesses and is called here for two
	// reasons: it is the only way to exercise that witness against a live
	// manager, and it is the raise site the signal counters below measure.
	if err := manager.CreateDevice(); err != nil {
		return fmt.Errorf("CreateDevice: %w", err)
	}
	shouldDraw, beginErr := manager.BeginDraw()
	if beginErr != nil {
		return fmt.Errorf("BeginDraw: %w", beginErr)
	}
	if shouldDraw {
		if err := manager.EndDraw(); err != nil {
			return fmt.Errorf("EndDraw: %w", err)
		}
	}

	// The native signals CNA actually delivered. The counter is read through
	// internal/servicebridge rather than from the manager, because an exported
	// accessor for it would be public API the XNA contract does not declare.
	deliveries, haveDeliveries := servicebridge.ReadManagerSignalDeliveries(manager)
	if !haveDeliveries || len(deliveries) != interop.ManagerEventCount {
		return fmt.Errorf("manager signal deliveries unavailable: %v %d", haveDeliveries, len(deliveries))
	}
	g.result.ManagerSignalDisposed += deliveries[interop.ManagerEventDisposed]
	g.result.ManagerSignalDeviceCreated += deliveries[interop.ManagerEventDeviceCreated]
	g.result.ManagerSignalDeviceDisposing += deliveries[interop.ManagerEventDeviceDisposing]
	g.result.ManagerSignalDeviceReset += deliveries[interop.ManagerEventDeviceReset]
	g.result.ManagerSignalDeviceResetting += deliveries[interop.ManagerEventDeviceResetting]

	if err := manager.Dispose(true); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	// Dispose unregisters both services, and only its own: the reference's own
	// `if (GetService(...) == this)` guard.
	afterManager, _ := host.Services().GetService(reflect.TypeOf((*framework.IGraphicsDeviceManager)(nil)).Elem())
	afterService, _ := host.Services().GetService(reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem())
	if afterManager != nil || afterService != nil {
		return fmt.Errorf("Dispose left registrations behind: manager=%v service=%v", afterManager != nil, afterService != nil)
	}
	g.result.ManagerServiceRemovalCheck++

	g.result.ManagerCycles++
	return nil
}

func (g *graphicsManagerGame) LoadContent(*framework.Game) error { return nil }
func (g *graphicsManagerGame) Update(host *framework.Game, _ framework.GameTime) error {
	return host.Exit()
}
func (g *graphicsManagerGame) Draw(*framework.Game, framework.GameTime) error { return nil }
func (g *graphicsManagerGame) UnloadContent(*framework.Game) error            { return nil }

func runGraphicsManagerChild() error {
	game := &graphicsManagerGame{}
	host, err := framework.NewGame(game)
	if err != nil {
		return err
	}
	if err := host.Run(); err != nil {
		return err
	}
	runtime.GC()
	game.result.GCStressPoints++
	data, _ := json.Marshal(game.result)
	fmt.Println(string(data))
	return nil
}

// exerciseContent is the content scenario: one ContentManager per cycle, over
// the live device the manager's own service resolves, inside the callback CNA
// requires a device handle to be live in.
//
// What it proves, and what it cannot:
//
//   - The manager Game's constructor created is the one this reaches, and it is
//     the same object on every call. Identity is managed and always provable.
//   - The root directory ROUND TRIPS THROUGH CNA once the native manager
//     exists. This is the proof that the getter is not answering a managed copy,
//     which no managed test can make: with no native half there is nothing to
//     disagree with.
//   - The resolved asset path is CNA's, and contains the root CNA was given.
//   - A load either succeeds or is refused, and the two are counted apart.
//     There is no compiled `.xnb` corpus in this repository, so a refusal is the
//     expected outcome on the pinned artifacts and is recorded as BLOCKED_ASSET
//     rather than reported as a pass.
func (g *stressGame) exerciseContent(host *framework.Game) error {
	manager := content.GameContent(host)
	if manager == nil {
		return errors.New("the Game's ContentManager is nil inside LoadContent; the constructor creates it")
	}
	if content.GameContent(host) != manager {
		return errors.New("GameContent answered two different managers; the reference reads one field")
	}
	g.result.ContentIdentityChecks++

	// A directory that exists and holds one file with a name no asset uses.
	// The point is a root CNA can normalise and echo back, not a corpus.
	root, err := os.MkdirTemp("", "cna-go-content")
	if err != nil {
		return fmt.Errorf("content root: %w", err)
	}
	defer os.RemoveAll(root)

	if err := manager.SetRootDirectory(root); err != nil {
		return fmt.Errorf("SetRootDirectory before the native manager exists: %w", err)
	}

	// OpenStream is the first member that needs the native manager, so this is
	// where it is created. The stream itself is expected to fail: nothing has
	// written the `.xnb` the reference's OpenStream opens.
	if _, err := manager.OpenStream("nothing-here"); err == nil {
		return errors.New("OpenStream opened a stream for an asset that does not exist")
	}
	g.result.ContentManagerCreations++

	// The round trip that only a native manager can prove. The value is written
	// through the projection and read back from CNA; a getter answering a
	// managed field would pass with the write removed.
	second := filepath.Join(root, "second")
	if err := os.Mkdir(second, 0o755); err != nil {
		return fmt.Errorf("second content root: %w", err)
	}
	if err := manager.SetRootDirectory(second); err != nil {
		return fmt.Errorf("SetRootDirectory: %w", err)
	}
	readBack, err := manager.RootDirectory()
	if err != nil {
		return fmt.Errorf("RootDirectory: %w", err)
	}
	if readBack != second {
		return fmt.Errorf("RootDirectory = %q after setting %q; the value must come back from CNA", readBack, second)
	}
	g.result.ContentRootRoundTrips++

	// CNA resolves the path, and it must be under the root CNA holds. A
	// projection that joined the path itself would agree here by accident and
	// disagree the moment CNA normalised anything.
	resolved, err := manager.OpenStream("asset")
	if err == nil {
		return errors.New("OpenStream opened a stream for an asset with no file")
	}
	if resolved != nil {
		return errors.New("OpenStream returned a reader alongside its error")
	}
	if !strings.Contains(err.Error(), second) {
		return fmt.Errorf("the OpenStream failure %v does not name the resolved path under %q", err, second)
	}
	g.result.ContentAssetPathChecks++

	// The asset itself. CNA's content pipeline resolves `<root>/<name>` and
	// decodes what it finds there, so the PNG this tool already encodes for the
	// sprite scenario IS a loadable asset -- the load below is a real one, not a
	// probe of the failure path.
	if err := os.WriteFile(filepath.Join(second, "asset"), g.data, 0o600); err != nil {
		return fmt.Errorf("writing the content asset: %w", err)
	}

	// The typed load. A T outside the closed set is refused by the projection
	// before CNA is reached; the projected one reaches CNA and is answered.
	type unprojectedAsset struct{}
	if _, err := content.ContentManagerLoad[*unprojectedAsset](manager, "asset"); err == nil {
		return errors.New("Load of an unprojected asset type reported no error")
	}
	g.result.ContentTypeRefusals++

	texture, loadErr := content.ContentManagerLoad[*graphics.Texture2D](manager, "asset")
	switch {
	case loadErr == nil:
		if texture == nil {
			return errors.New("Load reported success and produced no texture")
		}
		// The asset is the 2x2 PNG this tool encodes, and its four texels are
		// known. Checking them is what separates "CNA returned a texture" from
		// "CNA decoded THIS asset": a pipeline that handed back an empty
		// surface of the right size would pass a dimension check.
		if texture.Width() != 2 || texture.Height() != 2 {
			return fmt.Errorf("loaded texture is %dx%d, want the asset's 2x2", texture.Width(), texture.Height())
		}
		pixels := make([]framework.Color, 4)
		if err := graphics.Texture2DGetDataBySliceOfT(texture, pixels); err != nil {
			// A renderer with no readback path. Recorded, not passed.
			g.result.ContentLoadReadbackRefusals++
			fmt.Fprintf(os.Stderr, "content readback refused: %v\n", err)
		} else {
			want := []framework.Color{
				framework.NewColorByInt32AndInt32AndInt32(255, 0, 0),
				framework.NewColorByInt32AndInt32AndInt32(0, 255, 0),
				framework.NewColorByInt32AndInt32AndInt32(0, 0, 255),
				framework.NewColorByInt32AndInt32AndInt32(255, 255, 255),
			}
			for index := range want {
				if pixels[index] != want[index] {
					return fmt.Errorf("loaded texel %d = %+v, want %+v: the content pipeline did not decode the asset",
						index, pixels[index], want[index])
				}
			}
			g.result.ContentLoadPixelChecks++
		}
		// A second load of the same name. CNA caches by normalized key, so this
		// must succeed with the file already deleted -- which is the observable
		// difference between CNA's cache and a projection that re-read.
		if err := os.Remove(filepath.Join(second, "asset")); err != nil {
			return fmt.Errorf("removing the content asset: %w", err)
		}
		cached, cacheErr := content.ContentManagerLoad[*graphics.Texture2D](manager, "asset")
		if cacheErr != nil {
			return fmt.Errorf("a second Load of a cached asset: %w", cacheErr)
		}
		if cached == nil || cached.Width() != 2 {
			return errors.New("the cached Load produced no usable texture")
		}
		g.result.ContentCacheChecks++
		g.result.ContentLoads++
		// Both handles are independently owned: CNA says a loaded texture
		// "remains valid across content-manager unload or destruction and must
		// be destroyed before the parent game". So both are destroyed here, by
		// this scenario, and neither is left to the manager.
		if err := cached.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the cached texture: %w", err)
		}
		if err := texture.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the loaded texture: %w", err)
		}
	default:
		// CNA_RESULT_IO for an asset that is not there. Recorded, not passed.
		g.result.ContentLoadRefusals++
		fmt.Fprintf(os.Stderr, "content load refused: %v\n", loadErr)
	}

	if err := manager.Unload(); err != nil {
		return fmt.Errorf("Unload: %w", err)
	}
	g.result.ContentUnloadCalls++

	// Disposal is the manager's own, and it must be idempotent and must close
	// every member -- the same shape every other native-backed type here has.
	if err := manager.DisposeByNone(); err != nil {
		return fmt.Errorf("DisposeByNone: %w", err)
	}
	if err := manager.DisposeByNone(); err != nil {
		return fmt.Errorf("a second DisposeByNone: %w", err)
	}
	if _, err := manager.RootDirectory(); err == nil {
		return errors.New("RootDirectory answered after Dispose")
	}
	if err := manager.Unload(); err == nil {
		return errors.New("Unload succeeded after Dispose")
	}
	g.result.ContentDisposalChecks++
	return nil
}

// exerciseIndexBuffer is the index-buffer scenario. One 16-bit and one 32-bit
// buffer per cycle, created on the live device inside LoadContent -- which CNA
// requires, because cna_index_buffer_create takes a callback-scoped device
// handle.
//
// What it proves, and what it cannot:
//
//   - CNA reports the count, width and usage it APPLIED, and the projection
//     records those rather than the request. A renderer that widened an index
//     would be visible instead of hidden.
//   - Indices written through the projection come back FROM CNA'S BUFFER
//     unchanged, whole-array and through a window. A projection keeping a
//     managed copy would pass a test that compared its own input.
//   - The three refusals the projection makes ITSELF, before CNA is reached:
//     an oversized transfer, a 32-bit transfer into a 16-bit buffer, and a
//     GetData on a WriteOnly buffer.
//   - Disposal destroys the CNA buffer and every later transfer is refused.
func (g *stressGame) exerciseIndexBuffer() error {
	device := g.device

	sixteen, err := graphics.NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		device, graphics.IndexElementSizeSixteenBits, 6, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("NewIndexBuffer(SixteenBits): %w", err)
	}
	g.result.IndexBufferCreations++

	if sixteen.IndexCount() != 6 {
		return fmt.Errorf("IndexCount = %d, want 6", sixteen.IndexCount())
	}
	if sixteen.IndexElementSize() != graphics.IndexElementSizeSixteenBits {
		return fmt.Errorf("IndexElementSize = %d, want SixteenBits", sixteen.IndexElementSize())
	}
	if sixteen.BufferUsage() != graphics.BufferUsageNone {
		return fmt.Errorf("BufferUsage = %d, want None", sixteen.BufferUsage())
	}
	if sixteen.GraphicsDevice() != device {
		return errors.New("the buffer does not report the device it was created on")
	}
	if got := sixteen.ToString(); got != "Microsoft.Xna.Framework.Graphics.IndexBuffer" {
		return fmt.Errorf("ToString = %q; the CLR `this` must reach the outermost object", got)
	}
	g.result.IndexBufferDescriptionChecks++

	written := []uint16{0, 1, 2, 2, 3, 0}
	if err := graphics.IndexBufferSetDataBySliceOfT(sixteen, written); err != nil {
		return fmt.Errorf("SetData: %w", err)
	}
	readBack := make([]uint16, len(written))
	if err := graphics.IndexBufferGetDataBySliceOfT(sixteen, readBack); err != nil {
		// A renderer with no index readback path. Recorded, not passed.
		g.result.IndexBufferReadbackRefusals++
		fmt.Fprintf(os.Stderr, "index-buffer readback refused: %v\n", err)
	} else {
		for at := range written {
			if readBack[at] != written[at] {
				return fmt.Errorf("index %d read back as %d, want %d", at, readBack[at], written[at])
			}
		}
		g.result.IndexBufferRoundTrips++

		// The windowed overload, which indexes the CALLER'S array. CNA reads
		// back from buffer index zero and writes into the window, so the first
		// two indices land at positions 2 and 3 of the destination.
		windowed := make([]uint16, 6)
		if err := graphics.IndexBufferGetDataBySliceOfTAndInt32AndInt32(sixteen, windowed, 2, 2); err != nil {
			return fmt.Errorf("windowed GetData: %w", err)
		}
		if windowed[2] != written[0] || windowed[3] != written[1] {
			return fmt.Errorf("windowed read back %v, want %d and %d at positions 2 and 3",
				windowed, written[0], written[1])
		}
		if windowed[0] != 0 || windowed[1] != 0 || windowed[4] != 0 || windowed[5] != 0 {
			return fmt.Errorf("windowed GetData wrote outside its window: %v", windowed)
		}
		g.result.IndexBufferWindowRoundTrips++
	}

	// The three refusals the projection makes before CNA is reached.
	if err := graphics.IndexBufferSetDataBySliceOfT(sixteen, make([]uint16, 7)); err == nil {
		return errors.New("a transfer larger than the buffer was accepted")
	}
	if err := graphics.IndexBufferSetDataBySliceOfT(sixteen, make([]uint32, 6)); err == nil {
		return errors.New("a 32-bit transfer into a 16-bit buffer was accepted")
	}
	g.result.IndexBufferRefusals++

	// A 32-bit buffer, created from a Go type rather than the enum, and a
	// WriteOnly one whose GetData must be refused by the projection.
	thirtyTwo, err := graphics.NewIndexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		device, reflect.TypeOf(uint32(0)), 4, graphics.BufferUsageWriteOnly)
	if err != nil {
		return fmt.Errorf("NewIndexBuffer(typeof(uint32), WriteOnly): %w", err)
	}
	if thirtyTwo.IndexElementSize() != graphics.IndexElementSizeThirtyTwoBits {
		return fmt.Errorf("the Type constructor produced %d, want ThirtyTwoBits", thirtyTwo.IndexElementSize())
	}
	if thirtyTwo.BufferUsage() != graphics.BufferUsageWriteOnly {
		return fmt.Errorf("BufferUsage = %d, want WriteOnly as CNA applied it", thirtyTwo.BufferUsage())
	}
	if err := graphics.IndexBufferSetDataBySliceOfT(thirtyTwo, []uint32{9, 8, 7, 6}); err != nil {
		return fmt.Errorf("SetData on a WriteOnly buffer: %w", err)
	}
	if err := graphics.IndexBufferGetDataBySliceOfT(thirtyTwo, make([]uint32, 4)); err == nil {
		return errors.New("GetData on a WriteOnly buffer was accepted")
	} else if !strings.Contains(err.Error(),
		"Calling GetData on a resource that was created with BufferUsage.WriteOnly is not supported.") {
		return fmt.Errorf("WriteOnly GetData = %v, want the reference's message", err)
	}
	g.result.IndexBufferWriteOnlyChecks++

	// Disposal destroys the CNA buffer, is idempotent, and closes every
	// transfer while leaving the three properties answering.
	if err := thirtyTwo.DisposeByNone(); err != nil {
		return fmt.Errorf("DisposeByNone: %w", err)
	}
	if err := thirtyTwo.DisposeByNone(); err != nil {
		return fmt.Errorf("a second DisposeByNone: %w", err)
	}
	if !thirtyTwo.IsDisposed() {
		return errors.New("a disposed buffer reports not disposed")
	}
	if err := graphics.IndexBufferSetDataBySliceOfT(thirtyTwo, []uint32{1}); err == nil {
		return errors.New("a disposed buffer accepted a transfer")
	}
	if thirtyTwo.IndexCount() != 4 {
		return errors.New("a disposed buffer stopped answering IndexCount")
	}
	if err := sixteen.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the 16-bit buffer: %w", err)
	}
	g.result.IndexBufferDisposalChecks++
	return nil
}

// stressVertex is a consumer's own vertex type: a Go struct implementing
// IVertexType, which is what the Type-keyed VertexBuffer constructor resolves.
// Its layout is deliberately the one its declaration describes -- a Vector3 at
// 0 and a Color at 12, sixteen bytes -- because FromType's last check compares
// exactly those two numbers.
type stressVertex struct {
	Position framework.Vector3
	Colour   framework.Color
}

var stressVertexDeclaration = func() *graphics.VertexDeclaration {
	declaration, err := graphics.NewVertexDeclarationByInt32AndSliceOfVertexElement(16, []graphics.VertexElement{
		graphics.NewVertexElement(0, graphics.VertexElementFormatVector3, graphics.VertexElementUsagePosition, 0),
		graphics.NewVertexElement(12, graphics.VertexElementFormatColor, graphics.VertexElementUsageColor, 0),
	})
	if err != nil {
		panic(err)
	}
	return declaration
}()

func (stressVertex) VertexDeclaration() *graphics.VertexDeclaration { return stressVertexDeclaration }

// exerciseVertexBuffer is the vertex-buffer scenario. What it proves:
//
//   - A declaration's CNA handle is created LAZILY, on the first buffer that
//     needs one, and the SAME handle serves a second buffer -- which is what
//     makes the deferral safe rather than merely cheap.
//   - Vertices written through the projection come back FROM CNA'S BUFFER
//     unchanged, whole-buffer and from a byte offset into it.
//   - The Type-keyed constructor resolves a consumer's own IVertexType and
//     produces a buffer with that type's declaration.
//   - The two refusals the projection makes ITSELF: an oversized transfer and
//     a stride below sizeof(T).
func (g *stressGame) exerciseVertexBuffer(host *framework.Game) error {
	device := g.device

	buffer, err := graphics.NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		device, stressVertexDeclaration, 4, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("NewVertexBuffer: %w", err)
	}
	g.result.VertexBufferCreations++

	if buffer.VertexCount() != 4 {
		return fmt.Errorf("VertexCount = %d, want 4", buffer.VertexCount())
	}
	if buffer.VertexDeclaration() != stressVertexDeclaration {
		return errors.New("VertexDeclaration did not answer the caller's own object")
	}
	if buffer.BufferUsage() != graphics.BufferUsageNone {
		return fmt.Errorf("BufferUsage = %d, want None", buffer.BufferUsage())
	}
	if buffer.GraphicsDevice() != device {
		return errors.New("the buffer does not report the device it was created on")
	}
	if got := buffer.ToString(); got != "Microsoft.Xna.Framework.Graphics.VertexBuffer" {
		return fmt.Errorf("ToString = %q; the CLR `this` must reach the outermost object", got)
	}
	g.result.VertexBufferDescriptionChecks++

	// A SECOND buffer over the same declaration. The declaration's CNA handle
	// was created by the first and must be reused, not rebuilt: a second handle
	// would be a second native owner for one managed object.
	second, err := graphics.NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		device, stressVertexDeclaration, 2, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("a second buffer over the same declaration: %w", err)
	}
	if second.VertexDeclaration() != stressVertexDeclaration {
		return errors.New("the second buffer answered a different declaration")
	}
	g.result.VertexBufferDeclarationShares++

	written := []stressVertex{
		{Position: framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3), Colour: framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(4, 5, 6), Colour: framework.NewColorByInt32AndInt32AndInt32(0, 255, 0)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(7, 8, 9), Colour: framework.NewColorByInt32AndInt32AndInt32(0, 0, 255)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(10, 11, 12), Colour: framework.NewColorByInt32AndInt32AndInt32(255, 255, 255)},
	}
	if err := graphics.VertexBufferSetDataBySliceOfT(buffer, written); err != nil {
		return fmt.Errorf("SetData: %w", err)
	}
	readBack := make([]stressVertex, len(written))
	if err := graphics.VertexBufferGetDataBySliceOfT(buffer, readBack); err != nil {
		g.result.VertexBufferReadbackRefusals++
		fmt.Fprintf(os.Stderr, "vertex-buffer readback refused: %v\n", err)
	} else {
		for at := range written {
			if readBack[at] != written[at] {
				return fmt.Errorf("vertex %d read back as %+v, want %+v", at, readBack[at], written[at])
			}
		}
		g.result.VertexBufferRoundTrips++

		// The BUFFER offset, which is what XNA's offsetInBytes means and the
		// one offset in CNA's transfer family that indexes the buffer. Reading
		// two vertices from byte 32 must produce the third and fourth.
		fromOffset := make([]stressVertex, 2)
		if err := graphics.VertexBufferGetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
			buffer, 32, fromOffset, 0, 2, 0); err != nil {
			return fmt.Errorf("GetData from a byte offset: %w", err)
		}
		if fromOffset[0] != written[2] || fromOffset[1] != written[3] {
			return fmt.Errorf("reading from byte 32 produced %+v, want the third and fourth vertices", fromOffset)
		}
		g.result.VertexBufferOffsetRoundTrips++
	}

	// A declaration with an EXPLICIT stride WIDER than its elements need. The
	// projection passes the stride CNA's stride-less route would have
	// recomputed as 16, so this is the control that proves it does not: CNA
	// must report a 32-byte vertex, and every fit check is then measured in
	// that.
	padded, err := graphics.NewVertexDeclarationByInt32AndSliceOfVertexElement(32, []graphics.VertexElement{
		graphics.NewVertexElement(0, graphics.VertexElementFormatVector3, graphics.VertexElementUsagePosition, 0),
		graphics.NewVertexElement(12, graphics.VertexElementFormatColor, graphics.VertexElementUsageColor, 0),
	})
	if err != nil {
		return fmt.Errorf("a padded declaration: %w", err)
	}
	paddedBuffer, err := graphics.NewVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		device, padded, 2, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("a buffer over a padded declaration: %w", err)
	}
	// Four 16-byte vertices fit two 32-byte ones exactly, and five do not.
	// Both answers come from the stride CNA applied, not from the elements.
	if err := graphics.VertexBufferSetDataBySliceOfT(paddedBuffer, make([]stressVertex, 4)); err != nil {
		return fmt.Errorf("filling a padded buffer: %w", err)
	}
	if err := graphics.VertexBufferSetDataBySliceOfT(paddedBuffer, make([]stressVertex, 5)); err == nil {
		return errors.New("a padded buffer accepted more bytes than its stride allows")
	}
	if err := paddedBuffer.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the padded buffer: %w", err)
	}
	if err := padded.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the padded declaration: %w", err)
	}
	g.result.VertexBufferStrideChecks++

	// The Type-keyed constructor, over a consumer's own IVertexType.
	fromType, err := graphics.NewVertexBufferByGraphicsDeviceAndTypeAndInt32AndBufferUsage(
		device, reflect.TypeOf(stressVertex{}), 3, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("NewVertexBuffer(typeof(stressVertex)): %w", err)
	}
	if fromType.VertexDeclaration() != stressVertexDeclaration {
		return errors.New("the Type-keyed constructor did not use the type's own declaration")
	}
	if fromType.VertexCount() != 3 {
		return fmt.Errorf("the Type-keyed buffer has %d vertices, want 3", fromType.VertexCount())
	}
	g.result.VertexBufferFromTypeChecks++

	// The two refusals the projection makes before CNA is reached.
	if err := graphics.VertexBufferSetDataBySliceOfT(buffer, make([]stressVertex, 5)); err == nil {
		return errors.New("a transfer larger than the buffer was accepted")
	}
	if err := graphics.VertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32(
		buffer, 0, written, 0, 1, 8); err == nil {
		return errors.New("a vertex stride below sizeof(T) was accepted")
	}
	g.result.VertexBufferRefusals++

	// Foundation 67: BINDING and DRAWING. The buffers are bound to the live
	// device, the bound objects are read back and must be THE SAME Go objects,
	// and three draw calls are submitted.
	if err := device.SetVertexBufferByVertexBuffer(buffer); err != nil {
		return fmt.Errorf("SetVertexBuffer: %w", err)
	}
	bound := device.GetVertexBuffers()
	if len(bound) != 1 || bound[0].VertexBuffer() != buffer {
		return errors.New("GetVertexBuffers did not answer the object that was bound")
	}
	if bound[0].VertexOffset() != 0 || bound[0].InstanceFrequency() != 0 {
		return fmt.Errorf("the bound binding is %d/%d, want zeros",
			bound[0].VertexOffset(), bound[0].InstanceFrequency())
	}
	g.result.VertexBufferBindChecks++

	indices, err := graphics.NewIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		device, graphics.IndexElementSizeSixteenBits, 6, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("an index buffer to bind: %w", err)
	}
	if err := graphics.IndexBufferSetDataBySliceOfT(indices, []uint16{0, 1, 2, 2, 3, 0}); err != nil {
		return fmt.Errorf("filling the index buffer: %w", err)
	}
	if err := device.SetIndices(indices); err != nil {
		return fmt.Errorf("SetIndices: %w", err)
	}
	if device.Indices() != indices {
		return errors.New("Indices did not answer the object that was bound")
	}
	g.result.VertexBufferIndexBindChecks++

	// Foundation 67 measured three draws refusing with "no effect has been
	// applied", and classified that BACKEND_BLOCKED / EFFECT_DEPENDENCY.
	// Foundation 72 removes the dependency, so the same three draws run TWICE:
	// once with nothing applied, which must still refuse, and once after a real
	// Effect's pass has been applied.
	//
	// The first half is the control. If it ever stops refusing, the second
	// half proves nothing -- a pass that succeeds either way is not evidence
	// that applying the effect is what made it succeed.
	beforeDraw := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1)
	if beforeDraw != nil {
		g.result.VertexBufferDrawRefusalsBeforeApply++
	}

	// A real Effect, through the type's own public surface. The empty-effect
	// route CNA offers has NO XNA counterpart and is deliberately unbound, so
	// this is ContentManager.Load<Effect> over CNA's own `.cnj` stock-effect
	// descriptor -- the one shape that does not need
	// CNA_GRAPHICS_CAPABILITY_COMPILED_EFFECTS, which the Foundation 72 probe
	// measured FALSE on all three published artifacts.
	effect, effectErr := g.loadStockEffect(host)
	switch {
	case effectErr != nil:
		g.result.VertexBufferEffectRefusals++
		fmt.Fprintf(os.Stderr, "stock effect refused: %v\n", effectErr)
	default:
		g.result.VertexBufferEffectLoads++
		technique := effect.CurrentTechnique()
		if technique == nil {
			return errors.New("a loaded effect has no current technique")
		}
		pass := technique.Passes().ItemPropertySignatureCA1DC5FC(0)
		if pass == nil {
			return errors.New("a loaded effect's current technique has no first pass")
		}
		if err := pass.Apply(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("EffectPass.Apply: %w", err)
			}
			g.result.VertexBufferEffectApplyRefusals++
			fmt.Fprintf(os.Stderr, "EffectPass.Apply refused: %v\n", err)
		} else {
			g.result.VertexBufferEffectApplies++
		}
	}

	// Three draws, now with whatever the effect left applied. CNA answers for
	// whether the backend can execute them; what the projection owns is that
	// they are submitted with the right arguments and that its own guards ran
	// first.
	drawErr := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1)
	indexedErr := device.DrawIndexedPrimitives(graphics.PrimitiveTypeTriangleList, 0, 0, 4, 0, 2)
	instancedErr := device.DrawInstancedPrimitives(graphics.PrimitiveTypeTriangleList, 0, 0, 4, 0, 2, 2)
	switch {
	case drawErr == nil && indexedErr == nil && instancedErr == nil:
		g.result.VertexBufferDraws++
	default:
		// A backend that still cannot draw. Recorded, not passed, and the
		// reason CNA gives is printed so a reader can see whether it is still
		// the effect dependency or something else.
		g.result.VertexBufferDrawRefusals++
		fmt.Fprintf(os.Stderr, "draw refused after apply: %v / %v / %v\n", drawErr, indexedErr, instancedErr)
	}
	// The six user-primitive draws, with the effect still applied. Their vertex
	// data is the SAME stressVertex array the buffer round trip used, so the
	// declaration CNA is given is the one FromType resolved from that type.
	userVertices := []stressVertex{
		{Position: framework.NewVector3BySingleAndSingleAndSingle(0, 0, 0), Colour: framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0), Colour: framework.NewColorByInt32AndInt32AndInt32(0, 255, 0)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0), Colour: framework.NewColorByInt32AndInt32AndInt32(0, 0, 255)},
	}
	userIndices16 := []int16{0, 1, 2}
	userIndices32 := []int32{0, 1, 2}
	userDeclaration := stressVertexDeclaration
	for name, submit := range map[string]func() error{
		"primitives": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 1)
		},
		"primitives+decl": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 1, userDeclaration)
		},
		"indexed16": func() error {
			return graphics.GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 3, userIndices16, 0, 1)
		},
		"indexed32": func() error {
			return graphics.GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 3, userIndices32, 0, 1)
		},
		"indexed16+decl": func() error {
			return graphics.GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 3, userIndices16, 0, 1, userDeclaration)
		},
		"indexed32+decl": func() error {
			return graphics.GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 3, userIndices32, 0, 1, userDeclaration)
		},
		// Foundation 77. The same six draws again, but over the profile's own
		// four stock vertex types rather than this file's stressVertex.
		//
		// The point is not a second draw. It is that FromType resolves each
		// type's STATIC VertexDeclaration through its IVertexType witness, and
		// that CNA accepts the element table the reference's own `.cctor`
		// builds -- offsets 0/12, 0/12/16 and 0/12/24 -- rather than one this
		// repository computed from a Go struct layout.
		"stock VertexPositionColor": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, stockPositionColorTriangle(), 0, 1)
		},
		"stock VertexPositionTexture": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, stockPositionTextureTriangle(), 0, 1)
		},
		"stock VertexPositionColorTexture": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, stockPositionColorTextureTriangle(), 0, 1)
		},
		"stock VertexPositionNormalTexture": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, stockPositionNormalTextureTriangle(), 0, 1)
		},
		// A NON-ZERO vertexOffset, which the six above never exercise: four
		// vertices, one triangle, starting at index one. A projection that
		// ignored the offset would submit the wrong three.
		"stock offset triangle": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, graphics.PrimitiveTypeTriangleList, stockPositionColorQuad(), 1, 1)
		},
	} {
		if err := submit(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("DrawUser %s: %w", name, err)
			}
			g.result.UserPrimitiveDrawRefusals++
			fmt.Fprintf(os.Stderr, "DrawUser %s refused: %v\n", name, err)
			continue
		}
		g.result.UserPrimitiveDraws++
	}
	// The four guards the projection makes ITSELF, before CNA is reached.
	for name, refuse := range map[string]func() error{
		"nil vertex data": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, []stressVertex(nil), 0, 1, userDeclaration)
		},
		"nil declaration": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 1, nil)
		},
		"zero primitives": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 0, userDeclaration)
		},
		"window past the array": func() error {
			return graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, graphics.PrimitiveTypeTriangleList, userVertices, 0, 2, userDeclaration)
		},
	} {
		if err := refuse(); err == nil {
			return fmt.Errorf("DrawUser accepted %s", name)
		}
	}
	g.result.UserPrimitiveGuardChecks++

	// Foundation 79. A BasicEffect through its own public constructor, which is
	// a different door from the `.cnj` load above: cna_basic_effect_create
	// rather than cna_content_manager_load_effect.
	//
	// It runs LAST in this scenario, and that placement is deliberate. The
	// slice disposes the effects it makes, and CNA treats disposing the applied
	// effect as un-applying it -- the first arrangement put this ahead of the
	// draws and turned every one of them back into "no effect has been
	// applied". Running it after the draws leaves the evidence above intact and
	// costs nothing, because the slice applies its own effect before the one
	// draw it makes.
	//
	// Everything in it is the half the managed tests cannot reach. The managed
	// state, the dirty flags and the default-lighting rig are measured without
	// a device in basic_effect_test.go; what needs one is the four properties
	// that cross, the three lights CNA publishes, OnApply's push and the
	// disposal of the light views.
	if err := g.exerciseBasicEffect(device); err != nil {
		return err
	}

	// Foundation 80. The two unlit stock effects and EffectMaterial, on the
	// same live device and after the same draws.
	if err := g.exerciseUnlitEffects(device); err != nil {
		return err
	}

	// Foundation 81. The last two stock effects, on the same live device.
	if err := g.exerciseLitEffects(device); err != nil {
		return err
	}

	// Foundation 82. The two root-type statics, which need only a game.
	if err := g.exerciseRootStatics(); err != nil {
		return err
	}

	// Foundation 87. The audio family, which needs a game and no device.
	if err := g.exerciseSoundEffect(); err != nil {
		return err
	}

	// Foundation 89. The Input statics, which need a game and no device.
	if err := g.exerciseInputStatics(); err != nil {
		return err
	}

	// Foundation 91. Storage, which needs neither a game nor a device -- but
	// which MUST prove where it is writing before it writes anything.
	if err := g.exerciseStorage(); err != nil {
		return err
	}

	// Foundation 88. The two types that finish the Audio namespace.
	if err := g.exerciseDynamicSoundEffectInstance(); err != nil {
		return err
	}
	if err := g.exerciseMicrophone(); err != nil {
		return err
	}

	// Foundation 83. OcclusionQuery, on the same live device.
	if err := g.exerciseOcclusionQuery(device); err != nil {
		return err
	}

	// Foundation 84. The two dynamic buffers, on the same live device and
	// after the same draws.
	if err := g.exerciseDynamicBuffers(device); err != nil {
		return err
	}

	if effect != nil {
		if err := effect.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the stock effect: %w", err)
		}
		if !effect.IsDisposed() {
			return errors.New("the effect is not disposed after Dispose")
		}
		g.result.VertexBufferEffectDisposalChecks++
	}

	// The one draw guard the projection makes ITSELF, and the member that must
	// NOT make it.
	instanced, err := graphics.NewVertexBufferBindingByVertexBufferAndInt32AndInt32(buffer, 0, 1)
	if err != nil {
		return fmt.Errorf("an instanced binding: %w", err)
	}
	if err := device.SetVertexBuffers([]graphics.VertexBufferBinding{instanced}); err != nil {
		return fmt.Errorf("SetVertexBuffers with a frequency: %w", err)
	}
	if err := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1); err == nil {
		return errors.New("a non-instanced draw was accepted with a non-zero instance frequency")
	} else if !strings.Contains(err.Error(),
		"Non-instanced draw calls are not valid when a vertex buffer is bound with a non-zero instance frequency.") {
		return fmt.Errorf("the non-instanced refusal = %v, want the reference's message", err)
	}
	if err := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 0); err == nil {
		return errors.New("a zero primitive count was accepted")
	}
	// The INSTANCED member must NOT carry that refusal -- an instanced draw is
	// exactly the call a non-zero frequency is for. With the same instanced
	// binding still applied it must reach CNA, whose own answer (no effect
	// applied, on these artifacts) is a different failure entirely.
	if err := device.DrawInstancedPrimitives(graphics.PrimitiveTypeTriangleList, 0, 0, 4, 0, 2, 2); err != nil &&
		strings.Contains(err.Error(),
			"Non-instanced draw calls are not valid when a vertex buffer is bound with a non-zero instance frequency.") {
		return errors.New("DrawInstancedPrimitives carried the non-instanced refusal")
	}
	g.result.VertexBufferDrawGuardChecks++

	// Unbinding is a nil buffer, not an error.
	if err := device.SetVertexBufferByVertexBuffer(nil); err != nil {
		return fmt.Errorf("unbinding: %w", err)
	}
	if len(device.GetVertexBuffers()) != 0 {
		return errors.New("unbinding left a binding in the table")
	}
	if err := device.SetIndices(nil); err != nil {
		return fmt.Errorf("unbinding the index buffer: %w", err)
	}
	if device.Indices() != nil {
		return errors.New("unbinding left an index buffer bound")
	}
	if err := indices.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the bound index buffer: %w", err)
	}
	g.result.VertexBufferUnbindChecks++

	// Disposal destroys the CNA buffer, is idempotent, and leaves the shared
	// declaration alive -- the buffer does not own it.
	for _, disposable := range []*graphics.VertexBuffer{second, fromType, buffer} {
		if err := disposable.DisposeByNone(); err != nil {
			return fmt.Errorf("DisposeByNone: %w", err)
		}
		if err := disposable.DisposeByNone(); err != nil {
			return fmt.Errorf("a second DisposeByNone: %w", err)
		}
	}
	if stressVertexDeclaration.IsDisposed() {
		return errors.New("disposing the buffers disposed the shared declaration")
	}
	if err := graphics.VertexBufferSetDataBySliceOfT(buffer, written); err == nil {
		return errors.New("a disposed buffer accepted a transfer")
	}
	g.result.VertexBufferDisposalChecks++
	return nil
}

// exerciseAdapter is the adapter scenario. Everything here runs INSIDE
// LoadContent, because CNA enumerates adapters through a callback-scoped device
// handle -- which is the milestone's whole shape.
func (g *stressGame) exerciseAdapter() error {
	adapters, err := graphics.GraphicsAdapterAdapters()
	if err != nil {
		return fmt.Errorf("GraphicsAdapter.Adapters inside a callback: %w", err)
	}
	if adapters.Count() < 1 {
		return errors.New("CNA enumerated no adapters")
	}
	g.result.AdapterEnumerations++

	first, err := adapters.Item(0)
	if err != nil {
		return fmt.Errorf("the first adapter: %w", err)
	}
	byDefault, err := graphics.GraphicsAdapterDefaultAdapter()
	if err != nil {
		return fmt.Errorf("DefaultAdapter: %w", err)
	}
	// DefaultAdapter is element ZERO by position, which is what the reference
	// returns -- not the one whose IsDefaultAdapter flag is set. On a
	// single-adapter machine the two agree, so this checks the property that
	// IS decidable here: it is the first enumerated adapter.
	if byDefault.Description() != first.Description() || byDefault.DeviceName() != first.DeviceName() {
		return errors.New("DefaultAdapter is not the first enumerated adapter")
	}
	if byDefault.CurrentDisplayMode() == nil {
		return errors.New("the adapter reports no current display mode")
	}
	if byDefault.CurrentDisplayMode().Width() <= 0 || byDefault.CurrentDisplayMode().Height() <= 0 {
		return fmt.Errorf("the current display mode is %dx%d",
			byDefault.CurrentDisplayMode().Width(), byDefault.CurrentDisplayMode().Height())
	}
	supported := byDefault.SupportedDisplayModes()
	if supported == nil {
		return errors.New("the adapter reports no supported display modes")
	}
	walked := 0
	iterator := supported.GetEnumerator()
	for {
		mode, more, iterErr := iterator.Next()
		if iterErr != nil {
			return fmt.Errorf("walking the supported modes: %w", iterErr)
		}
		if !more {
			break
		}
		if mode.Width() <= 0 || mode.Height() <= 0 {
			return fmt.Errorf("a supported mode is %dx%d", mode.Width(), mode.Height())
		}
		walked++
	}
	// The indexer FILTERS: every mode it answers must carry the format asked
	// for, and there can be no more of them than the whole list.
	filtered, filteredOK := supported.Item(byDefault.CurrentDisplayMode().Format()).(interface {
		Next() (*graphics.DisplayMode, bool, error)
	})
	if !filteredOK {
		return errors.New("the indexer did not answer a display-mode sequence")
	}
	matched := 0
	for {
		mode, more, iterErr := filtered.Next()
		if iterErr != nil {
			return fmt.Errorf("walking the filtered modes: %w", iterErr)
		}
		if !more {
			break
		}
		if mode.Format() != byDefault.CurrentDisplayMode().Format() {
			return errors.New("the indexer answered a mode of another format")
		}
		matched++
	}
	if matched > walked {
		return fmt.Errorf("the filter answered %d of %d modes", matched, walked)
	}
	g.result.AdapterSnapshotChecks++

	// The device's OWN adapter, by the index CNA reports for it.
	deviceAdapter, err := g.device.Adapter()
	if err != nil {
		return fmt.Errorf("GraphicsDevice.Adapter: %w", err)
	}
	if deviceAdapter.CurrentDisplayMode() == nil {
		return errors.New("the device's adapter reports no current display mode")
	}
	g.result.AdapterDeviceAdapterChecks++

	for _, profile := range []graphics.GraphicsProfile{graphics.GraphicsProfileReach, graphics.GraphicsProfileHiDef} {
		if _, err := byDefault.IsProfileSupported(profile); err != nil {
			return fmt.Errorf("IsProfileSupported(%d): %w", profile, err)
		}
	}
	g.result.AdapterProfileChecks++

	// The two format queries. CNA reports what it SELECTED and whether it had
	// to substitute; both cross, and the projection reports CNA's flag rather
	// than a comparison of its own.
	for _, renderTarget := range []bool{false, true} {
		query := byDefault.QueryBackBufferFormat
		if renderTarget {
			query = byDefault.QueryRenderTargetFormat
		}
		exact, format, depth, samples, queryErr := query(
			graphics.GraphicsProfileReach, graphics.SurfaceFormatColor, graphics.DepthFormatDepth24, 0)
		if queryErr != nil {
			return fmt.Errorf("format query (render target %t): %w", renderTarget, queryErr)
		}
		if samples < 0 {
			return fmt.Errorf("the query selected %d samples", samples)
		}
		// The flag and the values must AGREE: an exact match means every
		// requested value came back unchanged. A projection reporting a flag it
		// invented would break this the moment the two diverged, and a
		// projection reporting CNA's cannot.
		unchanged := format == graphics.SurfaceFormatColor && depth == graphics.DepthFormatDepth24 && samples == 0
		if exact && !unchanged {
			return fmt.Errorf("the query reported an exact match but selected %d/%d/%d", format, depth, samples)
		}
		g.result.AdapterFormatQueries++
	}

	// The preference pair round-trips, and setting ONE must not clear the
	// other -- CNA's route takes both at once, so a setter that passed a
	// default for its neighbour would silently reset it.
	if err := graphics.SetGraphicsAdapterUseNullDevice(true); err != nil {
		return fmt.Errorf("SetUseNullDevice: %w", err)
	}
	if err := graphics.SetGraphicsAdapterUseReferenceDevice(true); err != nil {
		return fmt.Errorf("SetUseReferenceDevice: %w", err)
	}
	nullDevice, err := graphics.GraphicsAdapterUseNullDevice()
	if err != nil {
		return fmt.Errorf("UseNullDevice: %w", err)
	}
	referenceDevice, err := graphics.GraphicsAdapterUseReferenceDevice()
	if err != nil {
		return fmt.Errorf("UseReferenceDevice: %w", err)
	}
	if !nullDevice || !referenceDevice {
		return fmt.Errorf("the preference pair round-tripped as %t/%t; setting one cleared the other",
			nullDevice, referenceDevice)
	}
	if err := graphics.SetGraphicsAdapterUseNullDevice(false); err != nil {
		return fmt.Errorf("clearing UseNullDevice: %w", err)
	}
	nullDevice, _ = graphics.GraphicsAdapterUseNullDevice()
	referenceDevice, _ = graphics.GraphicsAdapterUseReferenceDevice()
	if nullDevice || !referenceDevice {
		return fmt.Errorf("clearing one preference left %t/%t; the other must be preserved",
			nullDevice, referenceDevice)
	}
	if err := graphics.SetGraphicsAdapterUseReferenceDevice(false); err != nil {
		return fmt.Errorf("clearing UseReferenceDevice: %w", err)
	}
	g.result.AdapterPreferenceChecks++

	// The snapshot is VALUES, so it keeps answering: reading it again after
	// every query above must give the same answers.
	if byDefault.Description() != first.Description() ||
		byDefault.CurrentDisplayMode().Width() != deviceAdapter.CurrentDisplayMode().Width() {
		return errors.New("an adapter snapshot changed under the queries")
	}
	if byDefault.Description() == "" && byDefault.DeviceName() == "" {
		fmt.Fprintln(os.Stderr, "the adapter reports neither a description nor a device name")
	}
	g.result.AdapterOutsideCallbackChecks++
	return nil
}

// ---------------------------------------------------------------------------
// Foundation 69 — the sprite-font scenario.
// ---------------------------------------------------------------------------

// spriteFontAtlas is the same one-pixel 24-bit BMP CNA's own content tests use
// as a font atlas. What a font needs from its atlas here is that it decodes,
// not what it contains.
var spriteFontAtlas = []byte{
	0x42, 0x4D, 0x3A, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x13, 0x0B,
	0x00, 0x00, 0x13, 0x0B, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1E, 0x14,
	0x0A, 0x00,
}

// spriteFontDescriptor is CNA's own `.cnj` SpriteFont format, authored here
// rather than compiled: there is no `.xnb` font corpus in this repository and
// CNA reads both containers through the same loader.
//
// The three glyphs are chosen so the measurement is not symmetric. 'B' carries
// a NEGATIVE bearing on each side, which is what makes InternalMeasure's two
// clamps observable and what the Foundation 69 probe measured CNA's own
// measurement disagreeing about. Characters ascend, which the reference's
// binary search requires and CNA's reader enforces.
const spriteFontDescriptor = `{"cnjVersion":1,"type":"SpriteFont",` +
	`"texture":"cna-go-stress-font-atlas.bmp",` +
	`"lineSpacing":10,"spacing":1.0,` +
	`"glyphs":[` +
	`{"char":63,"source":[0,0,4,8],"crop":[0,0,4,8],"kerning":[1.0,4.0,2.0]},` +
	`{"char":65,"source":[4,0,5,8],"crop":[0,0,5,8],"kerning":[0.0,5.0,0.0]},` +
	`{"char":66,"source":[9,0,6,12],"crop":[0,0,6,12],"kerning":[-3.0,6.0,-2.0]}` +
	`]}`

// exerciseSpriteFont is the sprite-font scenario. Everything here runs inside a
// lifecycle callback, because CNA's content manager is created from a
// callback-scoped device.
//
// What it proves that a unit test cannot:
//
//   - a REAL CNA font is loaded through ContentManager.Load<SpriteFont>, and
//     the glyph table the projection measures with is the one CNA reported;
//   - the measurement is the REFERENCE's, and it is proved so by reproducing
//     the exact case the Foundation 69 probe measured CNA's own measurement
//     disagreeing about;
//   - the three setters really reach CNA and really come back;
//   - a font is CNA-cached by asset name, exactly as a texture is;
//   - the two owned handles are released, in CNA's order, by the game's own
//     teardown -- which is what the reverse registration order exists for.
func (g *stressGame) exerciseSpriteFont(host *framework.Game) error {
	manager := content.GameContent(host)
	if manager == nil {
		return errors.New("the Game's ContentManager is nil inside the sprite-font scenario")
	}
	root, err := os.MkdirTemp("", "cna-go-sprite-font")
	if err != nil {
		return fmt.Errorf("sprite-font root: %w", err)
	}
	defer os.RemoveAll(root)
	if err := os.WriteFile(filepath.Join(root, "cna-go-stress-font-atlas.bmp"), spriteFontAtlas, 0o600); err != nil {
		return fmt.Errorf("writing the font atlas: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cna-go-stress-font.cnj"), []byte(spriteFontDescriptor), 0o600); err != nil {
		return fmt.Errorf("writing the font descriptor: %w", err)
	}
	if err := manager.SetRootDirectory(root); err != nil {
		return fmt.Errorf("SetRootDirectory: %w", err)
	}
	// OpenStream is what creates the native manager, and it is expected to fail
	// for a name with no `.xnb`.
	if _, err := manager.OpenStream("cna-go-stress-font"); err == nil {
		return errors.New("OpenStream opened a stream for an asset with no .xnb")
	}

	font, loadErr := content.ContentManagerLoad[*graphics.SpriteFont](manager, "cna-go-stress-font")
	if loadErr != nil {
		// CNA could not read the fixture. Recorded, not passed.
		g.result.SpriteFontLoadRefusals++
		fmt.Fprintf(os.Stderr, "sprite-font load refused: %v\n", loadErr)
		return nil
	}
	if font == nil {
		return errors.New("Load<SpriteFont> reported success and produced no font")
	}
	g.result.SpriteFontLoads++

	// The descriptor's own values, read back through the projection from the
	// snapshot CNA reported.
	if font.LineSpacing() != 10 {
		return fmt.Errorf("LineSpacing = %d, want the descriptor's 10", font.LineSpacing())
	}
	if font.Spacing() != 1 {
		return fmt.Errorf("Spacing = %v, want the descriptor's 1", font.Spacing())
	}
	if _, ok := font.DefaultCharacter(); ok {
		return errors.New("DefaultCharacter has a value; the descriptor names none")
	}
	characters := font.Characters()
	if characters == nil || characters.Count() != 3 {
		return fmt.Errorf("Characters() reported %v entries, want the descriptor's three", characters)
	}
	for index, want := range []uint16{'?', 'A', 'B'} {
		got, err := characters.Item(int32(index))
		if err != nil || got != want {
			return fmt.Errorf("Characters()[%d] = (%v, %v), want %v", index, got, err, want)
		}
	}
	if font.Characters() != characters {
		return errors.New("Characters() built a second view; the reference caches the first")
	}
	g.result.SpriteFontGlyphChecks++

	// The measurement, over the table CNA reported. These are the descriptor's
	// own numbers run through the reference's arithmetic, so a projection that
	// measured a DIFFERENT font -- or the same font with different glyph
	// metrics -- fails here rather than passing on a plausible size.
	for _, probe := range []struct {
		text string
		x, y float32
	}{
		{"", 0, 0},
		{"A", 5, 10},
		{"?", 7, 10},
		{"AA", 11, 10},
		{"BA", 10, 12},
		{"A\nA", 5, 20},
		{"A\r\nA", 5, 20},
		{"\r", 0, 10},
	} {
		size, err := font.MeasureStringByString(probe.text)
		if err != nil {
			return fmt.Errorf("MeasureString(%q): %w", probe.text, err)
		}
		if size.X != probe.x || size.Y != probe.y {
			return fmt.Errorf("MeasureString(%q) = (%v, %v), want (%v, %v)",
				probe.text, size.X, size.Y, probe.x, probe.y)
		}
	}
	g.result.SpriteFontMeasureChecks++

	// The measured divergence, reproduced against a live font. CNA's own
	// cna_sprite_font_measure_utf8 answers 4 for "B" and 7 for "AB" because it
	// adds the last glyph's negative right bearing unclamped; the reference's
	// final statement is `result.X += Math.Max(rightBearing, 0f)` and answers 6
	// and 9. The projection must answer the REFERENCE's numbers, over CNA's
	// data. This is the one assertion that separates "the algorithm is the
	// reference's" from "the algorithm is whatever the runtime does".
	for _, probe := range []struct {
		text string
		x    float32
	}{{"B", 6}, {"AB", 9}, {"AB\nA", 9}} {
		size, err := font.MeasureStringByString(probe.text)
		if err != nil {
			return fmt.Errorf("MeasureString(%q): %w", probe.text, err)
		}
		if size.X != probe.x {
			return fmt.Errorf("MeasureString(%q).X = %v, want the reference's %v (CNA's own measure answers two less)",
				probe.text, size.X, probe.x)
		}
	}
	g.result.SpriteFontDivergenceChecks++

	// A character with no glyph and no default character is the reference's
	// ArgumentException, with the exact FrameworkResources sentence CNA cannot
	// produce.
	if _, err := font.MeasureStringByString("Z"); err == nil {
		return errors.New("an unknown character with no default character measured successfully")
	} else if !strings.Contains(err.Error(), "is not available in this SpriteFont") {
		return fmt.Errorf("the unknown-character refusal is not the reference's: %v", err)
	}
	g.result.SpriteFontRefusals++

	// The three setters, each written through the projection and each proved to
	// have reached CNA by a value only CNA could refuse.
	missing := uint16('#')
	if err := font.SetDefaultCharacter(&missing); err == nil {
		return errors.New("SetDefaultCharacter accepted a character the font does not have")
	}
	fallback := uint16('?')
	if err := font.SetDefaultCharacter(&fallback); err != nil {
		return fmt.Errorf("SetDefaultCharacter('?'): %w", err)
	}
	if value, ok := font.DefaultCharacter(); !ok || value != '?' {
		return fmt.Errorf("DefaultCharacter() = (%v, %v) after setting '?'", value, ok)
	}
	// With a default character, the unknown character now measures as it.
	unknown, err := font.MeasureStringByString("Z")
	if err != nil {
		return fmt.Errorf("MeasureString(\"Z\") with a default character: %w", err)
	}
	if unknown.X != 7 {
		return fmt.Errorf("the fallback glyph measured %v, want '?' at 1+4+2", unknown.X)
	}
	if err := font.SetLineSpacing(20); err != nil {
		return fmt.Errorf("SetLineSpacing(20): %w", err)
	}
	if err := font.SetSpacing(2.5); err != nil {
		return fmt.Errorf("SetSpacing(2.5): %w", err)
	}
	// CNA narrows what XNA stores: a non-finite spacing is refused by
	// cna_sprite_font_set_spacing and the reference stores it. The refusal is
	// CNA's own, recorded rather than reworded.
	if err := font.SetSpacing(float32(math.NaN())); err == nil {
		return errors.New("CNA accepted a non-finite spacing; the pinned artifact refuses one")
	}
	if !math.IsNaN(float64(font.Spacing())) {
		return errors.New("the managed store did not precede the refused native push")
	}
	if err := font.SetSpacing(2.5); err != nil {
		return fmt.Errorf("restoring the spacing: %w", err)
	}
	measured, err := font.MeasureStringByString("AB")
	if err != nil {
		return fmt.Errorf("MeasureString after the setters: %w", err)
	}
	if measured.X != 5+2.5+0+(-3)+6 || measured.Y != 20+0 {
		return fmt.Errorf("MeasureString(\"AB\") = %v after setting spacing 2.5 and line spacing 20", measured)
	}
	g.result.SpriteFontSetterRoundTrips++

	// The cache is CNA's, exactly as it is for a texture: a second Load of the
	// same name answers with the files already deleted.
	if err := os.Remove(filepath.Join(root, "cna-go-stress-font.cnj")); err != nil {
		return fmt.Errorf("removing the font descriptor: %w", err)
	}
	cached, cacheErr := content.ContentManagerLoad[*graphics.SpriteFont](manager, "cna-go-stress-font")
	if cacheErr != nil {
		return fmt.Errorf("a second Load of a cached font: %w", cacheErr)
	}
	if cached == nil {
		return errors.New("the cached Load produced no font")
	}
	// The projection is a NEW Go object over CNA's cached handles, which is a
	// measured difference from the reference: XNA's ContentManager caches the
	// managed object and returns the same reference. CNA caches natively and
	// CNA-Go builds a facade per call, so the two fonts are equal in every
	// observable and are not the same object.
	if cached == font {
		return errors.New("the second Load returned the same Go object; CNA-Go builds a facade per call")
	}
	if cached.LineSpacing() != 10 {
		return fmt.Errorf("the cached font's LineSpacing = %d, want the descriptor's 10 on a fresh facade", cached.LineSpacing())
	}
	g.result.SpriteFontCacheChecks++

	// All six DrawString overloads, through a live SpriteBatch. The two
	// argument guards come BEFORE the begin/end check, exactly as Draw's null
	// texture does, so they are exercised outside a pair.
	batch, batchErr := graphics.NewSpriteBatch(g.device)
	if batchErr != nil {
		return fmt.Errorf("NewSpriteBatch for DrawString: %w", batchErr)
	}
	white := framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)
	nilFontErr := batch.DrawStringBySpriteFontAndStringAndVector2AndColor(
		nil, "A", framework.Vector2{}, white)
	if nilFontErr == nil || !strings.Contains(nilFontErr.Error(), "spriteFont") {
		return fmt.Errorf("a nil SpriteFont outside a pair = %v, want ArgumentNullException(\"spriteFont\")", nilFontErr)
	}
	nilTextErr := batch.DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(
		font, nil, framework.Vector2{}, white)
	if nilTextErr == nil || !strings.Contains(nilTextErr.Error(), "text") {
		return fmt.Errorf("a nil StringBuilder outside a pair = %v, want ArgumentNullException(\"text\")", nilTextErr)
	}
	outsideErr := batch.DrawStringBySpriteFontAndStringAndVector2AndColor(
		font, "A", framework.Vector2{}, white)
	if outsideErr == nil || !strings.Contains(outsideErr.Error(), "Begin must be called successfully before a Draw can be called.") {
		return fmt.Errorf("DrawString outside a pair = %v, want the InvalidOperationException message", outsideErr)
	}
	g.result.SpriteFontDrawStringGuards++

	if err := batch.BeginByNone(); err != nil {
		return fmt.Errorf("Begin before DrawString: %w", err)
	}
	var builder strings.Builder
	builder.WriteString("AB")
	origin := framework.NewVector2BySingleAndSingle(1, 2)
	scale := framework.NewVector2BySingleAndSingle(2, 3)
	for _, submit := range []func() error{
		func() error {
			return batch.DrawStringBySpriteFontAndStringAndVector2AndColor(
				font, "AB", framework.NewVector2BySingleAndSingle(4, 5), white)
		},
		func() error {
			return batch.DrawStringBySpriteFontAndStringBuilderAndVector2AndColor(
				font, &builder, framework.NewVector2BySingleAndSingle(4, 5), white)
		},
		func() error {
			return batch.DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				font, "AB", framework.NewVector2BySingleAndSingle(4, 5), white,
				0.25, origin, 2, graphics.SpriteEffectsFlipHorizontally, 0.5)
		},
		func() error {
			return batch.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
				font, &builder, framework.NewVector2BySingleAndSingle(4, 5), white,
				0.25, origin, 2, graphics.SpriteEffectsFlipVertically, 0.5)
		},
		func() error {
			return batch.DrawStringBySpriteFontAndStringAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				font, "AB", framework.NewVector2BySingleAndSingle(4, 5), white,
				0.25, origin, scale, graphics.SpriteEffectsNone, 0.5)
		},
		func() error {
			return batch.DrawStringBySpriteFontAndStringBuilderAndVector2AndColorAndSingleAndVector2AndVector2AndSpriteEffectsAndSingle(
				font, &builder, framework.NewVector2BySingleAndSingle(4, 5), white,
				0.25, origin, scale, graphics.SpriteEffectsNone, 0.5)
		},
	} {
		if err := submit(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("DrawString: %w", err)
			}
			// A renderer that cannot draw text. Recorded, not passed.
			g.result.SpriteFontDrawStringRefusals++
			fmt.Fprintf(os.Stderr, "DrawString refused: %v\n", err)
			continue
		}
		g.result.SpriteFontDrawStringSubmits++
	}
	if err := batch.End(); err != nil {
		return fmt.Errorf("End after DrawString: %w", err)
	}
	if err := batch.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the DrawString batch: %w", err)
	}

	// Nothing disposes either font. XNA's SpriteFont is not IDisposable, so the
	// two owned CNA handles per load are released by the game's own teardown --
	// in CNA's order, because the atlas is registered before the font and the
	// runtime releases in reverse. A wrong order would surface as CNA's
	// INVALID_STATE from the whole run.
	return nil
}

// ---------------------------------------------------------------------------
// Foundation 71 — the cube and volume texture scenario.
// ---------------------------------------------------------------------------

// exerciseTextureVolume creates a real TextureCube and a real Texture3D on the
// live device, round-trips Color texels through each, proves the one-element
// narrowing is a refusal rather than a silent reinterpretation, and disposes
// both.
//
// CNA documents both creations as RENDERER capabilities -- cna_texture3d_create
// returns NOT_SUPPORTED where the renderer has no volume storage, and a cube
// "may succeed even when face storage is unavailable" -- so a refusal from
// either is recorded as a renderer limitation rather than failing the run. That
// is the same shape the render-target bind already has.
func (g *stressGame) exerciseTextureVolume() error {
	// The Go-only element narrowing, proved on objects that never reach CNA:
	// the accepted set is one type wide because CNA's cube and volume transfer
	// routes take CNA_Color and carry no data-type identity.
	//
	// It is exercised HERE, before either creation, so a renderer that refuses
	// both still proves it.
	for _, refuse := range []func() error{
		func() error {
			return graphics.TextureCubeSetDataByCubeMapFaceAndSliceOfT[float32](nil, graphics.CubeMapFacePositiveX, nil)
		},
		func() error { return graphics.Texture3DSetDataBySliceOfT[float32](nil, nil) },
	} {
		if err := refuse(); err == nil {
			return errors.New("a float32 element was accepted by a cube or volume transfer")
		}
	}
	g.result.TextureVolumeElementRefusals += 2

	white := framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255)
	red := framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)

	// The cube. Four texels per face at size two.
	cube, cubeErr := graphics.NewTextureCube(g.device, 2, false, graphics.SurfaceFormatColor)
	switch {
	case cubeErr != nil:
		g.result.TextureCubeCreationRefusals++
		fmt.Fprintf(os.Stderr, "TextureCube creation refused: %v\n", cubeErr)
	default:
		g.result.TextureCubeCreations++
		if cube.Size() != 2 {
			return fmt.Errorf("TextureCube.Size = %d, want the created cube's 2", cube.Size())
		}
		if cube.Format() != graphics.SurfaceFormatColor || cube.LevelCount() < 1 {
			return fmt.Errorf("the cube's inherited description is %v / %d", cube.Format(), cube.LevelCount())
		}
		// A write to ONE face and a read back FROM THAT FACE. A projection that
		// ignored the face would pass a write-then-read of the same face and
		// fail the second face's check below.
		written := []framework.Color{white, red, red, white}
		if err := graphics.TextureCubeSetDataByCubeMapFaceAndSliceOfT(cube, graphics.CubeMapFacePositiveX, written); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("TextureCube.SetData: %w", err)
			}
			g.result.TextureCubeTransferRefusals++
			fmt.Fprintf(os.Stderr, "TextureCube transfer refused: %v\n", err)
		} else {
			readBack := make([]framework.Color, 4)
			if err := graphics.TextureCubeGetDataByCubeMapFaceAndSliceOfT(cube, graphics.CubeMapFacePositiveX, readBack); err != nil {
				if !isNativeRefusal(err) {
					return fmt.Errorf("TextureCube.GetData: %w", err)
				}
				g.result.TextureCubeTransferRefusals++
				fmt.Fprintf(os.Stderr, "TextureCube readback refused: %v\n", err)
			} else {
				for index := range written {
					if readBack[index] != written[index] {
						return fmt.Errorf("cube texel %d read back %+v, want %+v", index, readBack[index], written[index])
					}
				}
				g.result.TextureCubeRoundTrips++
			}
		}
		if err := cube.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the cube: %w", err)
		}
		if !cube.IsDisposed() {
			return errors.New("the cube is not disposed after Dispose")
		}
		if err := cube.DisposeByNone(); err != nil {
			return fmt.Errorf("a second cube Dispose: %w", err)
		}
		g.result.TextureVolumeDisposalChecks++
	}

	// The volume. Eight voxels at 2x2x2.
	volume, volumeErr := graphics.NewTexture3D(g.device, 2, 2, 2, false, graphics.SurfaceFormatColor)
	switch {
	case volumeErr != nil:
		g.result.Texture3DCreationRefusals++
		fmt.Fprintf(os.Stderr, "Texture3D creation refused: %v\n", volumeErr)
	default:
		g.result.Texture3DCreations++
		if volume.Width() != 2 || volume.Height() != 2 || volume.Depth() != 2 {
			return fmt.Errorf("Texture3D is %dx%dx%d, want the created volume's 2x2x2",
				volume.Width(), volume.Height(), volume.Depth())
		}
		written := []framework.Color{white, red, red, white, red, white, white, red}
		if err := graphics.Texture3DSetDataBySliceOfT(volume, written); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("Texture3D.SetData: %w", err)
			}
			g.result.Texture3DTransferRefusals++
			fmt.Fprintf(os.Stderr, "Texture3D transfer refused: %v\n", err)
		} else {
			readBack := make([]framework.Color, 8)
			if err := graphics.Texture3DGetDataBySliceOfT(volume, readBack); err != nil {
				if !isNativeRefusal(err) {
					return fmt.Errorf("Texture3D.GetData: %w", err)
				}
				g.result.Texture3DTransferRefusals++
				fmt.Fprintf(os.Stderr, "Texture3D readback refused: %v\n", err)
			} else {
				for index := range written {
					if readBack[index] != written[index] {
						return fmt.Errorf("volume voxel %d read back %+v, want %+v", index, readBack[index], written[index])
					}
				}
				g.result.Texture3DRoundTrips++
			}
		}
		if err := volume.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the volume: %w", err)
		}
		if !volume.IsDisposed() {
			return errors.New("the volume is not disposed after Dispose")
		}
		g.result.TextureVolumeDisposalChecks++
	}
	return nil
}

// stockEffectDescriptor is CNA's own `.cnj` envelope for a stock effect: the
// envelope's `type` names it and there is no separate field.
//
// This is the ONE shape of Effect a qualified artifact can produce. The
// Foundation 72 probe measured CNA_GRAPHICS_CAPABILITY_COMPILED_EFFECTS FALSE
// on HEADLESS, SOFTWARE and OPENGL33 alike, so Effect's compiled-bytecode
// constructor is refused everywhere CNA-Go can be qualified -- and
// cna_content_manager_load_effect's stock-descriptor path is not gated by it.
const stockEffectDescriptor = `{"cnjVersion":1,"type":"BasicEffect"}`

// loadStockEffect writes a `.cnj` stock-effect descriptor and loads it through
// ContentManager.Load<Effect>, which is the public surface a consumer has.
func (g *stressGame) loadStockEffect(host *framework.Game) (*graphics.Effect, error) {
	manager := content.GameContent(host)
	if manager == nil {
		return nil, errors.New("the Game's ContentManager is nil")
	}
	root, err := os.MkdirTemp("", "cna-go-effect")
	if err != nil {
		return nil, err
	}
	// The directory outlives this call: CNA caches the loaded asset and the
	// scenario keeps using the effect, so removing the root here would be
	// removing a file the cache may still consult.
	g.effectRoots = append(g.effectRoots, root)
	if err := os.WriteFile(filepath.Join(root, "cna-go-stock-effect.cnj"), []byte(stockEffectDescriptor), 0o600); err != nil {
		return nil, err
	}
	if err := manager.SetRootDirectory(root); err != nil {
		return nil, err
	}
	// OpenStream is what creates the native manager; it is expected to fail for
	// a name with no `.xnb`.
	if _, err := manager.OpenStream("cna-go-stock-effect"); err == nil {
		return nil, errors.New("OpenStream opened a stream for an asset with no .xnb")
	}
	return content.ContentManagerLoad[*graphics.Effect](manager, "cna-go-stock-effect")
}

// exerciseBasicEffect is Foundation 79's slice. It runs inside the vertex-buffer
// scenario because that is where a live device and a bound buffer already are,
// and because the draw it ends with is the same draw the effect above feeds.
func (g *stressGame) exerciseBasicEffect(device *graphics.GraphicsDevice) error {
	// The control, taken before anything in this slice exists. On an artifact
	// where the draw already works it succeeds and says so; on one where it
	// does not, a success after the apply below is attributable to the apply.
	if err := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("the BasicEffect control draw: %w", err)
		}
		g.result.BasicEffectControlDrawRefusals++
	} else {
		g.result.BasicEffectControlDraws++
	}

	effect, err := graphics.NewBasicEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewBasicEffectByGraphicsDevice: %w", err)
		}
		g.result.BasicEffectCreationRefusals++
		fmt.Fprintf(os.Stderr, "BasicEffect creation refused: %v\n", err)
		return nil
	}
	g.result.BasicEffectCreations++

	// The constructor's tail: DirectionalLight0.Enabled = true, SpecularColor =
	// Vector3.One, SpecularPower = 16. The first is managed and is asserted
	// here because it is the ONE default the field initialisers do not set;
	// the other two crossed into CNA and are read back below.
	if light := effect.DirectionalLight0(); light == nil || !light.Enabled() {
		return errors.New("the constructor did not enable DirectionalLight0")
	}

	// The constructor's other two tail statements crossed into CNA, so they are
	// read BACK out of it -- which is the only way the two writes are evidence
	// of anything. SpecularColor is Vector3.One and SpecularPower is 16, and
	// both are read before this slice writes anything of its own.
	if specular, err := effect.SpecularColor(); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("the constructed SpecularColor: %w", err)
		}
	} else if specular != framework.Vector3One() {
		return fmt.Errorf("a constructed BasicEffect's SpecularColor = %v, want Vector3.One", specular)
	}
	if power, err := effect.SpecularPower(); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("the constructed SpecularPower: %w", err)
		}
	} else if power != 16 {
		return fmt.Errorf("a constructed BasicEffect's SpecularPower = %v, want 16", power)
	}

	// The three lights are FIELDS in the reference, so every read answers the
	// same object -- an identity a projection that rebuilt a wrapper per call
	// would break.
	for index, pair := range [][2]*graphics.DirectionalLight{
		{effect.DirectionalLight0(), effect.DirectionalLight0()},
		{effect.DirectionalLight1(), effect.DirectionalLight1()},
		{effect.DirectionalLight2(), effect.DirectionalLight2()},
	} {
		if pair[0] == nil || pair[0] != pair[1] {
			return fmt.Errorf("DirectionalLight%d is not the same object on two reads", index)
		}
	}
	if effect.DirectionalLight0() == effect.DirectionalLight1() {
		return errors.New("two of the three published lights are the same object")
	}
	g.result.BasicEffectLightChecks++

	// The write-through, and the divergence disabling creates. Light 1 starts
	// DISABLED, so the colour write reaches the cache and not CNA; enabling it
	// afterwards is what publishes it, and the getter reports the cache
	// throughout.
	light1 := effect.DirectionalLight1()
	colour := framework.NewVector3BySingleAndSingleAndSingle(0.25, 0.5, 0.75)
	if err := light1.SetDiffuseColor(colour); err != nil {
		return fmt.Errorf("DirectionalLight1.SetDiffuseColor: %w", err)
	}
	if got := light1.DiffuseColor(); got != colour {
		return fmt.Errorf("a disabled light's DiffuseColor = %v, want the cached %v", got, colour)
	}
	if err := light1.SetEnabled(true); err != nil {
		return fmt.Errorf("DirectionalLight1.SetEnabled: %w", err)
	}
	direction := framework.NewVector3BySingleAndSingleAndSingle(0, -1, 0)
	if err := light1.SetDirection(direction); err != nil {
		return fmt.Errorf("DirectionalLight1.SetDirection: %w", err)
	}
	if got := light1.Direction(); got != direction {
		return fmt.Errorf("Direction = %v after a native write", got)
	}
	if err := light1.SetEnabled(false); err != nil {
		return fmt.Errorf("DirectionalLight1.SetEnabled(false): %w", err)
	}
	if got := light1.DiffuseColor(); got != colour {
		return fmt.Errorf("disabling a light changed its reported DiffuseColor to %v", got)
	}
	g.result.BasicEffectLightChecks++

	// The default-lighting rig, over lights that DO have a native half, which
	// is the only place its twelve writes actually cross.
	if err := effect.EnableDefaultLighting(); err != nil {
		return fmt.Errorf("EnableDefaultLighting: %w", err)
	}
	if !effect.LightingEnabled() {
		return errors.New("EnableDefaultLighting left LightingEnabled false")
	}
	g.result.BasicEffectLightChecks++

	// The four properties that cross. Each is written and read back through
	// CNA, so a value that did not survive the round trip is a real
	// disagreement rather than a managed field answering itself.
	specular := framework.NewVector3BySingleAndSingleAndSingle(0.5, 0.25, 0.125)
	fogColour := framework.NewVector3BySingleAndSingleAndSingle(0.75, 0.5, 0.25)
	roundTripped := true
	for _, step := range []struct {
		name  string
		write func() error
		check func() (bool, string, error)
	}{
		{"SpecularColor",
			func() error { return effect.SetSpecularColor(specular) },
			func() (bool, string, error) {
				got, err := effect.SpecularColor()
				return got == specular, fmt.Sprintf("%v", got), err
			}},
		{"SpecularPower",
			func() error { return effect.SetSpecularPower(24) },
			func() (bool, string, error) {
				got, err := effect.SpecularPower()
				return got == 24, fmt.Sprintf("%v", got), err
			}},
		{"FogColor",
			func() error { return effect.SetFogColor(fogColour) },
			func() (bool, string, error) {
				got, err := effect.FogColor()
				return got == fogColour, fmt.Sprintf("%v", got), err
			}},
	} {
		if err := step.write(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("Set%s: %w", step.name, err)
			}
			roundTripped = false
			fmt.Fprintf(os.Stderr, "BasicEffect Set%s refused: %v\n", step.name, err)
			continue
		}
		agreed, got, err := step.check()
		if err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("%s: %w", step.name, err)
			}
			roundTripped = false
			fmt.Fprintf(os.Stderr, "BasicEffect %s refused: %v\n", step.name, err)
			continue
		}
		if !agreed {
			return fmt.Errorf("%s round-tripped as %s", step.name, got)
		}
	}
	// Texture is the fourth, and it is the one whose getter answers a managed
	// field: CNA reports a handle and the property's value is an object, so the
	// setter crosses and the getter does not.
	//
	// The claim that makes that projection right rather than merely convenient
	// is OBJECT IDENTITY -- the getter answers the same Texture2D the setter
	// was given, which is what the reference answers and what a handle cannot
	// carry. It is checked with a real texture, and then a null assignment is
	// checked too, because that is the reference's own null.
	surface, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 4, 4)
	if err != nil {
		return fmt.Errorf("a texture for the effect: %w", err)
	}
	if err := effect.SetTexture(surface); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("SetTexture: %w", err)
		}
		roundTripped = false
		fmt.Fprintf(os.Stderr, "BasicEffect SetTexture refused: %v\n", err)
	} else {
		texture, err := effect.Texture()
		if err != nil {
			return fmt.Errorf("Texture: %w", err)
		}
		if texture != surface {
			return errors.New("Texture did not answer the SAME object the setter was given")
		}
		if err := effect.SetTexture(nil); err != nil {
			return fmt.Errorf("SetTexture(nil): %w", err)
		}
		if texture, err := effect.Texture(); err != nil || texture != nil {
			return fmt.Errorf("Texture after a null assignment = %v, %v", texture, err)
		}
	}
	if err := surface.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the effect's texture: %w", err)
	}
	if roundTripped {
		g.result.BasicEffectRoundTrips++
	} else {
		g.result.BasicEffectRoundTripRefusals++
	}

	// The managed state, pushed. Applying the effect's own pass is what calls
	// OnApply, and OnApply is the only place the fourteen managed properties
	// reach CNA.
	effect.SetWorld(framework.MatrixIdentity())
	effect.SetView(framework.MatrixIdentity())
	effect.SetProjection(framework.MatrixIdentity())
	effect.SetDiffuseColor(framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0))
	effect.SetAlpha(1)
	effect.SetVertexColorEnabled(true)
	effect.SetFogEnabled(false)
	technique := effect.CurrentTechnique()
	if technique == nil {
		return errors.New("a constructed BasicEffect has no current technique")
	}
	pass := technique.Passes().ItemPropertySignatureCA1DC5FC(0)
	if pass == nil {
		return errors.New("a constructed BasicEffect's technique has no first pass")
	}
	if err := pass.Apply(); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("BasicEffect pass Apply: %w", err)
		}
		g.result.BasicEffectApplyRefusals++
		fmt.Fprintf(os.Stderr, "BasicEffect Apply refused: %v\n", err)
	} else {
		g.result.BasicEffectApplies++
		// The claim that applying a BasicEffect satisfies CNA's
		// "no effect has been applied" requirement, made falsifiable: the
		// control at the top of this scenario measured the same draw refusing
		// with exactly that message before anything was applied.
		if err := device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("a draw after a BasicEffect pass: %w", err)
			}
			g.result.BasicEffectDrawRefusals++
			fmt.Fprintf(os.Stderr, "draw after a BasicEffect apply refused: %v\n", err)
		} else {
			g.result.BasicEffectDraws++
		}
	}

	// Clone, and the downcast that is the whole reason Effect widens at
	// returns. The clone must be a BasicEffect, must be a DIFFERENT object, and
	// must carry the thirteen values the clone constructor copies.
	cloned, err := effect.Clone()
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("BasicEffect.Clone: %w", err)
		}
		fmt.Fprintf(os.Stderr, "BasicEffect Clone refused: %v\n", err)
	} else {
		clone, ok := cloned.(*graphics.BasicEffect)
		if !ok {
			return errors.New("Clone did not hand back a BasicEffect")
		}
		if clone == effect {
			return errors.New("Clone answered the same object")
		}
		if clone.DiffuseColor() != effect.DiffuseColor() ||
			clone.Alpha() != effect.Alpha() ||
			clone.VertexColorEnabled() != effect.VertexColorEnabled() ||
			clone.LightingEnabled() != effect.LightingEnabled() {
			return errors.New("the clone constructor did not copy the thirteen managed values")
		}
		// The clone has its OWN lights, which is what
		// CacheEffectParameters(cloneSource) builds.
		if clone.DirectionalLight0() == effect.DirectionalLight0() {
			return errors.New("the clone shares a light object with its source")
		}
		g.result.BasicEffectCloneChecks++
		if err := clone.Dispose(); err != nil {
			return fmt.Errorf("disposing the cloned BasicEffect: %w", err)
		}
	}

	// Disposal releases the three light views before the effect behind them.
	if err := effect.Dispose(); err != nil {
		return fmt.Errorf("disposing the BasicEffect: %w", err)
	}
	if !effect.IsDisposed() {
		return errors.New("the BasicEffect is not disposed after Dispose")
	}
	// A disposed effect's members refuse rather than reaching a released
	// handle, which is what the generation check is for.
	if _, err := effect.SpecularColor(); err == nil {
		return errors.New("SpecularColor answered on a disposed BasicEffect")
	}
	g.result.BasicEffectDisposalChecks++
	return nil
}

// exerciseUnlitEffects is Foundation 80's slice: AlphaTestEffect,
// DualTextureEffect and EffectMaterial against a live device.
//
// It covers exactly what the managed tests cannot: the properties whose
// reference bodies reach an EffectParameter and which therefore reach CNA here,
// OnApply's push through each effect's own pass, EffectMaterial's construction
// from a source effect, and disposal.
func (g *stressGame) exerciseUnlitEffects(device *graphics.GraphicsDevice) error {
	surface, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 4, 4)
	if err != nil {
		return fmt.Errorf("a texture for the unlit effects: %w", err)
	}
	defer func() { _ = surface.DisposeByNone() }()

	alphaTest, err := graphics.NewAlphaTestEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewAlphaTestEffectByGraphicsDevice: %w", err)
		}
		g.result.UnlitEffectCreationRefusals += 2
		fmt.Fprintf(os.Stderr, "AlphaTestEffect creation refused: %v\n", err)
		return nil
	}
	g.result.UnlitEffectCreations++

	dual, err := graphics.NewDualTextureEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewDualTextureEffectByGraphicsDevice: %w", err)
		}
		g.result.UnlitEffectCreationRefusals++
		fmt.Fprintf(os.Stderr, "DualTextureEffect creation refused: %v\n", err)
		return nil
	}
	g.result.UnlitEffectCreations++

	// The crossing properties, written and read back through CNA. FogColor is
	// the shared one; the textures are the per-type ones, and the texture claim
	// is OBJECT IDENTITY, which a handle cannot carry.
	fogColour := framework.NewVector3BySingleAndSingleAndSingle(0.25, 0.5, 0.75)
	for name, effect := range map[string]interface {
		SetFogColor(framework.Vector3) error
		FogColor() (framework.Vector3, error)
	}{"AlphaTestEffect": alphaTest, "DualTextureEffect": dual} {
		if err := effect.SetFogColor(fogColour); err != nil {
			return fmt.Errorf("%s.SetFogColor: %w", name, err)
		}
		got, err := effect.FogColor()
		if err != nil {
			return fmt.Errorf("%s.FogColor: %w", name, err)
		}
		if got != fogColour {
			return fmt.Errorf("%s.FogColor round-tripped as %v", name, got)
		}
	}
	if err := alphaTest.SetTexture(surface); err != nil {
		return fmt.Errorf("AlphaTestEffect.SetTexture: %w", err)
	}
	if texture, err := alphaTest.Texture(); err != nil || texture != surface {
		return fmt.Errorf("AlphaTestEffect.Texture = %v, %v; the getter answers the object the setter was given", texture, err)
	}
	// Both DualTextureEffect layers reach ONE route at two indices, so setting
	// each and reading both back is what proves the index is honoured.
	if err := dual.SetTexture(surface); err != nil {
		return fmt.Errorf("DualTextureEffect.SetTexture: %w", err)
	}
	if err := dual.SetTexture2(nil); err != nil {
		return fmt.Errorf("DualTextureEffect.SetTexture2(nil): %w", err)
	}
	first, err := dual.Texture()
	if err != nil || first != surface {
		return fmt.Errorf("DualTextureEffect.Texture = %v, %v", first, err)
	}
	second, err := dual.Texture2()
	if err != nil || second != nil {
		return fmt.Errorf("DualTextureEffect.Texture2 = %v, %v", second, err)
	}
	g.result.UnlitEffectRoundTrips++

	// OnApply, through each effect's own pass. AlphaTestEffect's push includes
	// the pair no other stock effect has.
	alphaTest.SetWorld(framework.MatrixIdentity())
	alphaTest.SetAlphaFunction(graphics.CompareFunctionGreaterEqual)
	alphaTest.SetReferenceAlpha(128)
	alphaTest.SetVertexColorEnabled(true)
	dual.SetWorld(framework.MatrixIdentity())
	dual.SetVertexColorEnabled(true)
	for name, effect := range map[string]interface {
		CurrentTechnique() *graphics.EffectTechnique
	}{"AlphaTestEffect": alphaTest, "DualTextureEffect": dual} {
		technique := effect.CurrentTechnique()
		if technique == nil {
			return fmt.Errorf("a constructed %s has no current technique", name)
		}
		pass := technique.Passes().ItemPropertySignatureCA1DC5FC(0)
		if pass == nil {
			return fmt.Errorf("a constructed %s's technique has no first pass", name)
		}
		if err := pass.Apply(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("%s pass Apply: %w", name, err)
			}
			g.result.UnlitEffectApplyRefusals++
			fmt.Fprintf(os.Stderr, "%s Apply refused: %v\n", name, err)
		} else {
			g.result.UnlitEffectApplies++
		}
	}

	// Clone, and the downcast. Each must answer its OWN class.
	clonedAlphaTest, err := alphaTest.Clone()
	if err != nil {
		return fmt.Errorf("AlphaTestEffect.Clone: %w", err)
	}
	typedAlphaTest, ok := clonedAlphaTest.(*graphics.AlphaTestEffect)
	if !ok {
		return errors.New("AlphaTestEffect.Clone did not hand back an AlphaTestEffect")
	}
	if typedAlphaTest.AlphaFunction() != graphics.CompareFunctionGreaterEqual ||
		typedAlphaTest.ReferenceAlpha() != 128 ||
		!typedAlphaTest.VertexColorEnabled() {
		return errors.New("the AlphaTestEffect clone constructor did not copy its eleven values")
	}
	clonedDual, err := dual.Clone()
	if err != nil {
		return fmt.Errorf("DualTextureEffect.Clone: %w", err)
	}
	typedDual, ok := clonedDual.(*graphics.DualTextureEffect)
	if !ok {
		return errors.New("DualTextureEffect.Clone did not hand back a DualTextureEffect")
	}
	if !typedDual.VertexColorEnabled() {
		return errors.New("the DualTextureEffect clone constructor did not copy its nine values")
	}
	g.result.UnlitEffectCloneChecks++

	// EffectMaterial, from a source effect rather than a device -- which is
	// what its one constructor takes and what CNA's route takes.
	material, err := graphics.NewEffectMaterial(alphaTest)
	switch {
	case err != nil:
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewEffectMaterial: %w", err)
		}
		g.result.EffectMaterialRefusals++
		fmt.Fprintf(os.Stderr, "EffectMaterial creation refused: %v\n", err)
	default:
		g.result.EffectMaterialCreations++
		// The one thing the type adds is its class name, and ToString is where
		// a consumer sees it.
		if got := material.ToString(); got != "Microsoft.Xna.Framework.Graphics.EffectMaterial" {
			return fmt.Errorf("EffectMaterial.ToString = %q", got)
		}
		// Clone is INHERITED and answers an Effect, because the reference
		// declares no override.
		clone, cloneErr := material.Clone()
		if cloneErr != nil {
			return fmt.Errorf("EffectMaterial.Clone: %w", cloneErr)
		}
		// The assertion is POSITIVE as well as negative: the clone must be a
		// plain Effect, which is also the only way to reach a dispose member --
		// EffectReference carries none, because Effect and its derived types
		// spell disposal differently.
		plain, isPlain := clone.(*graphics.Effect)
		if !isPlain {
			return errors.New("EffectMaterial.Clone did not answer a plain Effect; the reference overrides nothing and Effect::Clone builds one")
		}
		g.result.EffectMaterialIdentityCheck++
		if err := plain.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing the material's clone: %w", err)
		}
		if err := material.Dispose(); err != nil {
			return fmt.Errorf("disposing the EffectMaterial: %w", err)
		}
	}

	for name, effect := range map[string]interface {
		Dispose() error
		IsDisposed() bool
	}{
		"AlphaTestEffect": alphaTest, "DualTextureEffect": dual,
		"AlphaTestEffect clone": typedAlphaTest, "DualTextureEffect clone": typedDual,
	} {
		if err := effect.Dispose(); err != nil {
			return fmt.Errorf("disposing the %s: %w", name, err)
		}
		if !effect.IsDisposed() {
			return fmt.Errorf("the %s is not disposed after Dispose", name)
		}
	}
	if _, err := alphaTest.FogColor(); err == nil {
		return errors.New("FogColor answered on a disposed AlphaTestEffect")
	}
	g.result.UnlitEffectDisposalChecks++
	return nil
}

// exerciseLitEffects is Foundation 81's slice: EnvironmentMapEffect and
// SkinnedEffect, the last two stock effects, against a live device.
//
// It covers what the managed tests cannot: the six and four properties that
// cross, the three lights each publishes, the bone-transform round trip through
// CNA, OnApply through each effect's own pass, Clone with its downcast, and
// disposal.
func (g *stressGame) exerciseLitEffects(device *graphics.GraphicsDevice) error {
	surface, err := graphics.NewTexture2DByGraphicsDeviceAndInt32AndInt32(device, 4, 4)
	if err != nil {
		return fmt.Errorf("a texture for the lit effects: %w", err)
	}
	defer func() { _ = surface.DisposeByNone() }()

	environment, err := graphics.NewEnvironmentMapEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewEnvironmentMapEffectByGraphicsDevice: %w", err)
		}
		g.result.LitEffectCreationRefusals += 2
		fmt.Fprintf(os.Stderr, "EnvironmentMapEffect creation refused: %v\n", err)
		return nil
	}
	g.result.LitEffectCreations++

	skinned, err := graphics.NewSkinnedEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewSkinnedEffectByGraphicsDevice: %w", err)
		}
		g.result.LitEffectCreationRefusals++
		fmt.Fprintf(os.Stderr, "SkinnedEffect creation refused: %v\n", err)
		return nil
	}
	g.result.LitEffectCreations++

	// The three published lights, on both types: stable identity and a
	// write-through that reaches CNA.
	for name, effect := range map[string]graphics.IEffectLights{
		"EnvironmentMapEffect": environment, "SkinnedEffect": skinned,
	} {
		if effect.DirectionalLight0() == nil || effect.DirectionalLight0() != effect.DirectionalLight0() {
			return fmt.Errorf("%s's DirectionalLight0 is not a stable object", name)
		}
		if effect.DirectionalLight0() == effect.DirectionalLight1() {
			return fmt.Errorf("%s published one object for two lights", name)
		}
		// Lighting is ALWAYS on for these two, whatever the setter is told.
		effect.SetLightingEnabled(false)
		if !effect.LightingEnabled() {
			return fmt.Errorf("%s reported LightingEnabled false", name)
		}
		if err := effect.EnableDefaultLighting(); err != nil {
			return fmt.Errorf("%s.EnableDefaultLighting: %w", name, err)
		}
		g.result.LitEffectLightChecks++
	}

	// EnvironmentMapEffect's four scalar/vector crossings, round-tripped.
	specular := framework.NewVector3BySingleAndSingleAndSingle(0.5, 0.25, 0.125)
	if err := environment.SetEnvironmentMapAmount(0.75); err != nil {
		return fmt.Errorf("SetEnvironmentMapAmount: %w", err)
	}
	if amount, err := environment.EnvironmentMapAmount(); err != nil || amount != 0.75 {
		return fmt.Errorf("EnvironmentMapAmount = %v, %v", amount, err)
	}
	if err := environment.SetEnvironmentMapSpecular(specular); err != nil {
		return fmt.Errorf("SetEnvironmentMapSpecular: %w", err)
	}
	if got, err := environment.EnvironmentMapSpecular(); err != nil || got != specular {
		return fmt.Errorf("EnvironmentMapSpecular = %v, %v", got, err)
	}
	if err := environment.SetFresnelFactor(0.5); err != nil {
		return fmt.Errorf("SetFresnelFactor: %w", err)
	}
	if factor, err := environment.FresnelFactor(); err != nil || factor != 0.5 {
		return fmt.Errorf("FresnelFactor = %v, %v", factor, err)
	}
	// The two texture positions, including the profile's only TextureCube one.
	if err := environment.SetTexture(surface); err != nil {
		return fmt.Errorf("EnvironmentMapEffect.SetTexture: %w", err)
	}
	if texture, err := environment.Texture(); err != nil || texture != surface {
		return fmt.Errorf("EnvironmentMapEffect.Texture = %v, %v", texture, err)
	}
	cube, cubeErr := graphics.NewTextureCube(device, 4, false, graphics.SurfaceFormatColor)
	switch {
	case cubeErr != nil:
		if !isNativeRefusal(cubeErr) {
			return fmt.Errorf("a cube for the environment map: %w", cubeErr)
		}
		fmt.Fprintf(os.Stderr, "TextureCube creation refused: %v\n", cubeErr)
	default:
		if err := environment.SetEnvironmentMap(cube); err != nil {
			return fmt.Errorf("SetEnvironmentMap: %w", err)
		}
		if got, err := environment.EnvironmentMap(); err != nil || got != cube {
			return fmt.Errorf("EnvironmentMap = %v, %v; the getter answers the object the setter was given", got, err)
		}
		// A measured divergence, recorded rather than avoided. CNA RETAINS the
		// cube an EnvironmentMapEffect points at -- cna_texturecube_destroy
		// answers CNA result 3, "The TextureCube is retained by an
		// EffectParameter" -- where XNA's Dispose on the same texture is legal.
		if err := cube.DisposeByNone(); err == nil {
			return errors.New("disposing a TextureCube an effect retains was accepted; CNA documents a refusal")
		}
		g.result.LitEffectRetentionChecks++
		// The RELEASE is proved on a SECOND cube, and that is not fussiness.
		// CNA documents a repeated dispose as success, so disposing the same
		// cube again after the release would succeed whether the release
		// happened or not -- a planted defect that made SetEnvironmentMap(nil)
		// a no-op survived exactly that assertion.
		released, releasedErr := graphics.NewTextureCube(device, 4, false, graphics.SurfaceFormatColor)
		if releasedErr != nil {
			return fmt.Errorf("a second cube for the release check: %w", releasedErr)
		}
		if err := environment.SetEnvironmentMap(released); err != nil {
			return fmt.Errorf("SetEnvironmentMap(second): %w", err)
		}
		if err := environment.SetEnvironmentMap(nil); err != nil {
			return fmt.Errorf("SetEnvironmentMap(nil): %w", err)
		}
		// First dispose of THIS cube, so a success is the release and nothing
		// else.
		if err := released.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing a released environment map: %w", err)
		}
		g.result.LitEffectReleaseChecks++
		// The first cube is still retained; disposing it again is the
		// documented repeat and is cleanup rather than evidence.
		_ = cube.DisposeByNone()
	}

	// SkinnedEffect's two scalar crossings and its texture.
	if err := skinned.SetSpecularColor(specular); err != nil {
		return fmt.Errorf("SkinnedEffect.SetSpecularColor: %w", err)
	}
	if got, err := skinned.SpecularColor(); err != nil || got != specular {
		return fmt.Errorf("SkinnedEffect.SpecularColor = %v, %v", got, err)
	}
	if err := skinned.SetSpecularPower(24); err != nil {
		return fmt.Errorf("SkinnedEffect.SetSpecularPower: %w", err)
	}
	if power, err := skinned.SpecularPower(); err != nil || power != 24 {
		return fmt.Errorf("SkinnedEffect.SpecularPower = %v, %v", power, err)
	}
	if err := skinned.SetTexture(surface); err != nil {
		return fmt.Errorf("SkinnedEffect.SetTexture: %w", err)
	}
	g.result.LitEffectRoundTrips++

	// The bone transforms, which are the one array crossing in the family. The
	// constructor already pushed 72 identities; this pushes a distinguishable
	// set and reads it back.
	bones := make([]framework.Matrix, 8)
	for index := range bones {
		bones[index] = framework.MatrixIdentity()
		bones[index].M41 = float32(index)
		// M44 is deliberately NOT 1. The reference overwrites it on every
		// matrix GetBoneTransforms returns, and sending 1 would make that
		// correction indistinguishable from the value coming back unchanged.
		bones[index].M44 = 7
	}
	if err := skinned.SetBoneTransforms(bones); err != nil {
		return fmt.Errorf("SetBoneTransforms: %w", err)
	}
	read, err := skinned.GetBoneTransforms(int32(len(bones)))
	switch {
	case err != nil:
		if !isNativeRefusal(err) {
			return fmt.Errorf("GetBoneTransforms: %w", err)
		}
		g.result.LitEffectBoneRefusals++
		fmt.Fprintf(os.Stderr, "GetBoneTransforms refused: %v\n", err)
	default:
		if len(read) != len(bones) {
			return fmt.Errorf("GetBoneTransforms(%d) answered %d", len(bones), len(read))
		}
		for index := range read {
			if read[index].M41 != float32(index) {
				return fmt.Errorf("bone %d round-tripped M41 as %v", index, read[index].M41)
			}
			// The M44 = 1 the reference writes over every returned matrix. The
			// input carried 7, so a 1 here is the correction and not an echo.
			if read[index].M44 != 1 {
				return fmt.Errorf("bone %d came back with M44 = %v; the reference forces 1 over whatever was stored", index, read[index].M44)
			}
		}
		g.result.LitEffectBoneRoundTrips++
	}

	// OnApply through each effect's own pass.
	for name, effect := range map[string]interface {
		CurrentTechnique() *graphics.EffectTechnique
	}{"EnvironmentMapEffect": environment, "SkinnedEffect": skinned} {
		technique := effect.CurrentTechnique()
		if technique == nil {
			return fmt.Errorf("a constructed %s has no current technique", name)
		}
		pass := technique.Passes().ItemPropertySignatureCA1DC5FC(0)
		if pass == nil {
			return fmt.Errorf("a constructed %s's technique has no first pass", name)
		}
		if err := pass.Apply(); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("%s pass Apply: %w", name, err)
			}
			g.result.LitEffectApplyRefusals++
			fmt.Fprintf(os.Stderr, "%s Apply refused: %v\n", name, err)
		} else {
			g.result.LitEffectApplies++
		}
	}

	// Clone, and the downcast each must answer.
	clonedEnvironment, err := environment.Clone()
	if err != nil {
		return fmt.Errorf("EnvironmentMapEffect.Clone: %w", err)
	}
	typedEnvironment, ok := clonedEnvironment.(*graphics.EnvironmentMapEffect)
	if !ok {
		return errors.New("EnvironmentMapEffect.Clone did not hand back an EnvironmentMapEffect")
	}
	clonedSkinned, err := skinned.Clone()
	if err != nil {
		return fmt.Errorf("SkinnedEffect.Clone: %w", err)
	}
	typedSkinned, ok := clonedSkinned.(*graphics.SkinnedEffect)
	if !ok {
		return errors.New("SkinnedEffect.Clone did not hand back a SkinnedEffect")
	}
	// WeightsPerVertex is one of the twelve values the clone constructor copies.
	if err := skinned.SetWeightsPerVertex(2); err != nil {
		return fmt.Errorf("SetWeightsPerVertex(2): %w", err)
	}
	secondClone, err := skinned.Clone()
	if err != nil {
		return fmt.Errorf("SkinnedEffect.Clone after SetWeightsPerVertex: %w", err)
	}
	if secondClone.(*graphics.SkinnedEffect).WeightsPerVertex() != 2 {
		return errors.New("the SkinnedEffect clone constructor did not copy WeightsPerVertex")
	}
	// The clones have their OWN lights.
	if typedEnvironment.DirectionalLight0() == environment.DirectionalLight0() ||
		typedSkinned.DirectionalLight0() == skinned.DirectionalLight0() {
		return errors.New("a clone shares a light object with its source")
	}
	g.result.LitEffectCloneChecks++

	for name, effect := range map[string]interface {
		Dispose() error
		IsDisposed() bool
	}{
		"EnvironmentMapEffect": environment, "SkinnedEffect": skinned,
		"EnvironmentMapEffect clone": typedEnvironment, "SkinnedEffect clone": typedSkinned,
		"SkinnedEffect second clone": secondClone.(*graphics.SkinnedEffect),
	} {
		if err := effect.Dispose(); err != nil {
			return fmt.Errorf("disposing the %s: %w", name, err)
		}
		if !effect.IsDisposed() {
			return fmt.Errorf("the %s is not disposed after Dispose", name)
		}
	}
	if _, err := skinned.SpecularColor(); err == nil {
		return errors.New("SpecularColor answered on a disposed SkinnedEffect")
	}
	g.result.LitEffectDisposalChecks++
	return nil
}

// exerciseRootStatics is Foundation 82's slice: FrameworkDispatcher.Update and
// TitleContainer.OpenStream.
//
// Neither takes a device. Both take a game handle for thread affinity, which is
// the one thing about them that cannot be measured without a running game --
// the guards are all managed and are measured in title_container_test.go.
func (g *stressGame) exerciseRootStatics() error {
	// The dispatcher, pumped repeatedly. CNA documents calling it while the
	// loop runs as "harmless and does the work twice", so a second call must
	// succeed as well as the first.
	for range 3 {
		if err := framework.FrameworkDispatcherUpdate(); err != nil {
			return fmt.Errorf("FrameworkDispatcherUpdate: %w", err)
		}
		g.result.FrameworkDispatcherUpdates++
	}

	// A title asset that is really there.
	//
	// CNA resolves the title path itself -- it logs "[TitleContainer] Resolved
	// path" as the executable's own -- and nothing in the projection can ask it
	// where that is. So the asset is WRITTEN next to the executable and read
	// back through the member, which is the only way to prove the resolution
	// and the copy together.
	//
	// The executable itself was the first fixture and is not usable: CNA
	// answers CNA result 5, "Failed to open file", for a binary that is
	// currently executing, so a run over it would have measured that refusal
	// rather than a read.
	name := fmt.Sprintf("cna-go-title-probe-%d.bin", os.Getpid())
	assetPath := filepath.Join(filepath.Dir(os.Args[0]), name)
	want := []byte("cna-go title container probe")
	if err := os.WriteFile(assetPath, want, 0o600); err != nil {
		return fmt.Errorf("writing a title asset: %w", err)
	}
	defer func() { _ = os.Remove(assetPath) }()
	reader, err := framework.TitleContainerOpenStream(name)
	switch {
	case err != nil:
		if !isNativeRefusal(err) {
			return fmt.Errorf("TitleContainerOpenStream(%q): %w", name, err)
		}
		g.result.TitleContainerReadRefusals++
		fmt.Fprintf(os.Stderr, "TitleContainerOpenStream refused: %v\n", err)
	default:
		content, readErr := io.ReadAll(reader)
		if readErr != nil {
			return fmt.Errorf("reading the title stream: %w", readErr)
		}
		if !bytes.Equal(content, want) {
			return fmt.Errorf("the title stream carried %q, want %q", content, want)
		}
		g.result.TitleContainerReads++
	}

	// An asset that is NOT there is CNA's CNA_RESULT_IO, carrying the
	// reference's own OpenStreamNotFound text.
	missing, missingErr := framework.TitleContainerOpenStream("cna-go-no-such-title-asset.bin")
	if missingErr == nil {
		return errors.New("TitleContainerOpenStream opened an asset that does not exist")
	}
	if !strings.Contains(missingErr.Error(), "File not found.") {
		return fmt.Errorf("a missing title asset reported %v, without the reference's message", missingErr)
	}
	// The reader must be NIL and not a reader over nothing. This is the only
	// place the failing READ is reachable -- every managed guard refuses before
	// it -- so it is the only place the typed-nil hazard can be held.
	if missing != nil {
		return errors.New("a failed title read handed back a non-nil reader")
	}
	g.result.TitleContainerGuardChecks++

	// The guards run BEFORE the game handle is looked up, which is what makes
	// them measurable without one -- and is asserted here from inside a game so
	// the ordering is held from both sides.
	for _, refused := range []string{"", "..\\secret", "a:b"} {
		if _, err := framework.TitleContainerOpenStream(refused); err == nil {
			return fmt.Errorf("TitleContainerOpenStream(%q) was accepted inside a game", refused)
		}
	}
	g.result.TitleContainerGuardChecks++
	return nil
}

// exerciseOcclusionQuery is Foundation 83's slice.
//
// The type's four flags and every guard they gate are measured without a device
// in occlusion_query_test.go. What needs one is the Begin/End pair reaching the
// GPU, the completion read, and the ONE path the managed tests cannot produce:
// a query that really does complete and answers a pixel count.
func (g *stressGame) exerciseOcclusionQuery(device *graphics.GraphicsDevice) error {
	query, err := graphics.NewOcclusionQuery(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewOcclusionQuery: %w", err)
		}
		g.result.OcclusionQueryCreationRefusals++
		fmt.Fprintf(os.Stderr, "OcclusionQuery creation refused: %v\n", err)
		return nil
	}
	g.result.OcclusionQueryCreations++

	// A fresh query is ARMED, so Begin is legal immediately -- which is the
	// constructor's one store seen from outside.
	if err := query.Begin(); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("OcclusionQuery.Begin: %w", err)
		}
		g.result.OcclusionQueryPairRefusals++
		fmt.Fprintf(os.Stderr, "OcclusionQuery.Begin refused: %v\n", err)
		if err := query.DisposeByNone(); err != nil {
			return fmt.Errorf("disposing a query that could not begin: %w", err)
		}
		g.result.OcclusionQueryDisposalChecks++
		return nil
	}

	// Inside the pair: a second Begin is refused by the pair guard, and it is
	// refused MANAGED-side, before CNA sees anything.
	if err := query.Begin(); err == nil {
		return errors.New("a second Begin inside a pair was accepted")
	}
	// A draw, so the query has something to count. It is BEST EFFORT and its
	// outcome is deliberately not asserted: this slice runs after the effect
	// slices, which dispose the effects they made, and CNA treats disposing the
	// applied effect as un-applying it -- so the draw answers "no effect has
	// been applied" here. What the query measures is not what this scenario is
	// about; that the pair reaches the GPU and the guards hold is.
	_ = device.DrawPrimitives(graphics.PrimitiveTypeTriangleList, 0, 1)
	if err := query.End(); err != nil {
		return fmt.Errorf("OcclusionQuery.End: %w", err)
	}
	g.result.OcclusionQueryPairs++

	// A second End outside the pair is refused, again managed-side.
	if err := query.End(); err == nil {
		return errors.New("a second End outside a pair was accepted")
	}
	// Begin DISARMED the query, so a second pair is refused until IsComplete is
	// asked. This is the store that is easiest to omit and the only place it
	// shows: the managed tests never run a Begin that succeeds.
	//
	// It has to be asserted HERE, before anything asks IsComplete. The first
	// arrangement checked "a query inside its own pair is not complete" during
	// the pair, and that call ARMED the query -- IsComplete's first statement
	// is the arming store, before every early return -- so this assertion then
	// fired on correct code. The two claims cannot share a pair, and the
	// inside-the-pair one is made in the second pair below.
	if err := query.Begin(); err == nil {
		return errors.New("a second Begin was accepted without IsComplete being checked")
	}
	g.result.OcclusionQueryGuardChecks++

	// The completion read. A query that has not finished answers false and
	// PixelCount refuses; one that has finished answers a count. BOTH are
	// legitimate outcomes of a single frame, so both are counted rather than
	// one of them being waited for.
	complete, err := query.IsComplete()
	if err != nil {
		return fmt.Errorf("OcclusionQuery.IsComplete: %w", err)
	}
	if complete {
		count, countErr := query.PixelCount()
		if countErr != nil {
			return fmt.Errorf("OcclusionQuery.PixelCount on a complete query: %w", countErr)
		}
		if count < 0 {
			return fmt.Errorf("OcclusionQuery.PixelCount answered %d", count)
		}
		g.result.OcclusionQueryCompletions++
	} else {
		if _, countErr := query.PixelCount(); countErr == nil {
			return errors.New("PixelCount answered on a query that is not complete")
		}
		g.result.OcclusionQueryPendingChecks++
	}

	// IsComplete re-armed Begin, whatever it answered, so a second pair is
	// legal -- which is the side effect the managed tests pin and this proves
	// end to end.
	if err := query.Begin(); err != nil {
		return fmt.Errorf("Begin after IsComplete re-armed it: %w", err)
	}
	// What IsComplete answers INSIDE a pair is recorded rather than asserted,
	// because the measurement contradicted the assumption.
	//
	// XNA's Begin sets `_isAvailable = false` and its GetData returns S_FALSE
	// until the pair is submitted, so a query inside its own pair reports
	// incomplete. CNA answers a different question -- its route is documented
	// as "whether the query result can be read without stalling the CPU" -- and
	// on both qualified artifacts it reports TRUE inside the second pair,
	// because the FIRST pair's result is still readable. Inside the first pair,
	// where nothing has ever completed, it reports false.
	//
	// The projection does not mask that: the reference's IsComplete re-reads
	// the runtime and overwrites its own flag too, so the structure is
	// faithful and the answer is the runtime's. Both outcomes are counted, so a
	// run in which the behaviour changed would show as a moved count rather
	// than as silence.
	inside, insideErr := query.IsComplete()
	if insideErr != nil {
		return fmt.Errorf("IsComplete inside a pair: %w", insideErr)
	}
	if inside {
		g.result.OcclusionQueryStaleResultChecks++
	} else {
		g.result.OcclusionQueryFreshResultChecks++
	}
	if err := query.End(); err != nil {
		return fmt.Errorf("the second End: %w", err)
	}
	g.result.OcclusionQueryPairs++

	if err := query.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the OcclusionQuery: %w", err)
	}
	if !query.IsDisposed() {
		return errors.New("the OcclusionQuery is not disposed after Dispose")
	}
	// A disposed query refuses rather than reaching a released handle.
	//
	// IsComplete is the member that proves it, and Begin is NOT: Begin's arming
	// guard refuses first whatever the handle is doing, so an assertion on
	// Begin would pass over a query whose CNA handle had been leaked.
	// IsComplete reaches CNA with no managed guard in front of it.
	if _, err := query.IsComplete(); err == nil {
		return errors.New("IsComplete answered on a disposed OcclusionQuery; its CNA handle was not released")
	}
	if err := query.Begin(); err == nil {
		return errors.New("Begin answered on a disposed OcclusionQuery")
	}
	g.result.OcclusionQueryDisposalChecks++
	return nil
}

// exerciseDynamicBuffers is Foundation 84's slice.
//
// The managed half -- the guard orders, the latch, SetContentLost's raise rule
// and the IDynamicGraphicsResource dispatch -- is measured without a device in
// dynamic_buffer_test.go. What needs one is everything CNA decides:
//
//   - that a buffer created with the `dynamic` flag reports itself dynamic, and
//     that a plain one does not, which is the only place the flag is visible;
//   - that every SetDataOptions value crosses and the data comes back;
//   - that the content-lost read reaches CNA and answers, and that a successful
//     upload clears a latch that was set by hand -- the ONE path that proves
//     the clear runs after a real native upload rather than only in a unit
//     test;
//   - that a dynamic buffer BINDS where its base does, which is the
//     substitutability claim end to end.
func (g *stressGame) exerciseDynamicBuffers(device *graphics.GraphicsDevice) error {
	vertex, err := graphics.NewDynamicVertexBufferByGraphicsDeviceAndVertexDeclarationAndInt32AndBufferUsage(
		device, stressVertexDeclaration, 4, graphics.BufferUsageNone)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewDynamicVertexBuffer: %w", err)
		}
		g.result.DynamicBufferCreationRefusals++
		fmt.Fprintf(os.Stderr, "dynamic vertex buffer creation refused: %v\n", err)
		return nil
	}
	g.result.DynamicBufferCreations++

	index, err := graphics.NewDynamicIndexBufferByGraphicsDeviceAndIndexElementSizeAndInt32AndBufferUsage(
		device, graphics.IndexElementSizeSixteenBits, 6, graphics.BufferUsageNone)
	if err != nil {
		return fmt.Errorf("NewDynamicIndexBuffer: %w", err)
	}

	// The inherited description, answered through the composed base, and the
	// CLR `this` reaching the outermost object.
	if vertex.VertexCount() != 4 || vertex.VertexDeclaration() != stressVertexDeclaration {
		return fmt.Errorf("the dynamic buffer reported %d vertices and %v",
			vertex.VertexCount(), vertex.VertexDeclaration())
	}
	if got := vertex.ToString(); got != "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer" {
		return fmt.Errorf("ToString = %q; the CLR `this` must reach the outermost object", got)
	}
	if got := index.ToString(); got != "Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer" {
		return fmt.Errorf("the index buffer's ToString = %q", got)
	}
	if index.IndexCount() != 6 || index.IndexElementSize() != graphics.IndexElementSizeSixteenBits {
		return fmt.Errorf("the dynamic index buffer reported %d indices at %v",
			index.IndexCount(), index.IndexElementSize())
	}
	g.result.DynamicBufferDescriptionChecks++

	written := []stressVertex{
		{Position: framework.NewVector3BySingleAndSingleAndSingle(21, 22, 23), Colour: framework.NewColorByInt32AndInt32AndInt32(10, 20, 30)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(24, 25, 26), Colour: framework.NewColorByInt32AndInt32AndInt32(40, 50, 60)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(27, 28, 29), Colour: framework.NewColorByInt32AndInt32AndInt32(70, 80, 90)},
		{Position: framework.NewVector3BySingleAndSingleAndSingle(30, 31, 32), Colour: framework.NewColorByInt32AndInt32AndInt32(100, 110, 120)},
	}
	// Every SetDataOptions value the enum has, one upload each. None is a
	// hint at all; Discard and NoOverwrite are, and CNA documents that a
	// windowed upload cannot keep NoOverwrite's promise and gives the whole
	// buffer instead -- a cost difference, not a result difference, so the
	// readback below must be identical whichever option wrote it.
	for _, options := range []graphics.SetDataOptions{
		graphics.SetDataOptionsNone,
		graphics.SetDataOptionsDiscard,
		graphics.SetDataOptionsNoOverwrite,
	} {
		if err := graphics.DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
			vertex, written, 0, int32(len(written)), options); err != nil {
			// A refusal here is a DEFECT, not a capability, and the measurement
			// that makes it so is this: the SAME upload on a buffer created
			// WITHOUT the dynamic flag is refused by CNA for every non-None
			// option, and accepted for None. The header says as much --
			// "non-None values require a supported dynamic-buffer overload" --
			// and a probe run confirmed it, three refusals and one acceptance.
			//
			// So this assertion is the only thing that can see the `dynamic`
			// flag at all: nothing in the public contract reports it, and
			// IsContentLost answers false whichever way it was created. A
			// buffer that CNA will take a Discard for is a buffer CNA built
			// dynamic.
			g.result.DynamicBufferOptionRefusals++
			return fmt.Errorf("CNA refused options %d on a DYNAMIC vertex buffer: %w -- it accepts every option on a dynamic buffer and refuses non-None on a static one, so the dynamic flag did not reach the creation", options, err)
		}
		g.result.DynamicBufferOptionUploads++

		readBack := make([]stressVertex, len(written))
		if err := graphics.VertexBufferGetDataBySliceOfT[stressVertex](vertex, readBack); err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("reading back an options-%d upload: %w", options, err)
			}
			fmt.Fprintf(os.Stderr, "dynamic vertex readback refused: %v\n", err)
			continue
		}
		for i := range written {
			if readBack[i] != written[i] {
				return fmt.Errorf("options %d changed vertex %d: wrote %+v, read %+v",
					options, i, written[i], readBack[i])
			}
		}
		g.result.DynamicBufferRoundTrips++
	}

	// The index side, through its own options-carrying overload and through the
	// OFFSET one, which reaches CNA's second route.
	indices := []uint16{0, 1, 2, 2, 3, 0}
	// The two values the ENUM does not name, which the reference's converter
	// accepts by a bit test and CNA refuses by name. They are the only native
	// witness the conversion has: a projection that handed CNA the caller's raw
	// value would be refused here and is accepted by the reference.
	//
	//	3  = Discard|NoOverwrite -> bit 0 wins  -> CNA_SET_DATA_DISCARD
	//	99 = 0b1100011           -> bit 0 set   -> CNA_SET_DATA_DISCARD
	for _, undefined := range []graphics.SetDataOptions{
		graphics.SetDataOptionsDiscard | graphics.SetDataOptionsNoOverwrite,
		99,
	} {
		if err := graphics.DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
			vertex, written, 0, int32(len(written)), undefined); err != nil {
			return fmt.Errorf("CNA refused SetDataOptions(%d): %w -- ConvertXnaSetDataOptionsToDx is a BIT TEST and maps it to Discard, so the caller's raw value must not reach CNA", undefined, err)
		}
		g.result.DynamicBufferOptionUploads++
	}

	// Discard, so the same reasoning applies: CNA refuses it on a static index
	// buffer, which makes this the index side's only view of the flag.
	if err := graphics.DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		index, indices, 0, int32(len(indices)), graphics.SetDataOptionsDiscard); err != nil {
		g.result.DynamicBufferOptionRefusals++
		return fmt.Errorf("CNA refused Discard on a DYNAMIC index buffer: %w -- the dynamic flag did not reach the creation", err)
	}
	g.result.DynamicBufferOptionUploads++
	// And the index side's own witness for the converter: 99 is undefined and
	// the reference maps it to Discard by its bit test, so it must be accepted
	// here too. The two sides need the assertion separately because they reach
	// CNA through different routes.
	if err := graphics.DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		index, indices, 0, int32(len(indices)), 99); err != nil {
		return fmt.Errorf("CNA refused SetDataOptions(99) on a dynamic index buffer: %w -- the raw value must not reach CNA", err)
	}
	g.result.DynamicBufferOptionUploads++

	// The content-lost read, which really does reach CNA. It answers FALSE on
	// both qualified artifacts -- CNA documents the field as "currently always
	// false" -- so what is asserted is that it answers at all and that the
	// answer is the documented one. A true here would be new information and is
	// reported rather than swallowed.
	for name, read := range map[string]func() (bool, error){
		"vertex": vertex.IsContentLost,
		"index":  index.IsContentLost,
	} {
		lost, err := read()
		if err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("%s IsContentLost: %w", name, err)
			}
			fmt.Fprintf(os.Stderr, "%s IsContentLost refused: %v\n", name, err)
			continue
		}
		if lost {
			return fmt.Errorf("%s IsContentLost answered true on a qualified artifact; CNA documents the field as always false, so this is new information", name)
		}
		g.result.DynamicBufferContentLostReads++
	}

	// The clear across a REAL upload, as far as the public surface reaches.
	//
	// What this scenario CANNOT do is set the latch first, and the reason is
	// the contract's rather than a gap here: SetContentLost is `assembly` in
	// the reference and unexported in the projection, so no consumer -- and no
	// consumer-shaped binary like this one -- can arm it. Exporting a hook for
	// the scenario's benefit would add public surface the pinned contract does
	// not declare, which is the one thing this project does not do for
	// convenience. The armed-then-cleared path is measured on the objects
	// themselves in dynamic_buffer_test.go, where the member is reachable.
	//
	// What IS measurable here is the whole path either side of it: a real
	// native upload runs, the latch is down afterwards, and no ContentLost is
	// delivered -- which is what a consumer streaming geometry every frame
	// actually observes.
	raised := 0
	subscription, err := vertex.AddContentLostHandler(func(any, *framework.EventArgs) error {
		raised++
		return nil
	})
	if err != nil {
		return fmt.Errorf("subscribing to ContentLost: %w", err)
	}
	if err := graphics.DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		vertex, written, 0, int32(len(written)), graphics.SetDataOptionsDiscard); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("the upload that runs CopyData's tail: %w", err)
		}
		fmt.Fprintf(os.Stderr, "the latch-clearing upload was refused: %v\n", err)
	} else {
		lost, err := vertex.IsContentLost()
		if err != nil {
			return fmt.Errorf("IsContentLost after a successful upload: %w", err)
		}
		if lost {
			return errors.New("the buffer reports content lost after a successful upload; CopyData's tail did not run")
		}
		if raised != 0 {
			return fmt.Errorf("an upload raised ContentLost %d times; SetContentLost raises only on true", raised)
		}
		g.result.DynamicBufferLatchClears++
	}
	if err := vertex.RemoveContentLostHandler(subscription); err != nil {
		return fmt.Errorf("unsubscribing from ContentLost: %w", err)
	}

	// The projection's own refusal, which must happen before CNA sees anything:
	// five vertices into a four-vertex buffer.
	if err := graphics.DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		vertex, make([]stressVertex, 5), 0, 5, graphics.SetDataOptionsNone); err == nil {
		return errors.New("a five-vertex upload into a four-vertex dynamic buffer was accepted")
	}
	// And the offset overload, whose offset indexes THIS BUFFER: one vertex
	// written a stride in must land on the second vertex and leave the first
	// alone, which is the one thing the offsetless route cannot express.
	//
	// The readback is what makes the offset assertable at all: an upload that
	// ignored it would write vertex 0 and leave vertex 1 as it was, which is
	// the same slice of bytes the caller handed in and would look correct
	// anywhere else. Writing a DISTINCT vertex at the offset and checking both
	// slots is the only shape that can tell the two apart.
	marker := []stressVertex{
		{Position: framework.NewVector3BySingleAndSingleAndSingle(-1, -2, -3), Colour: framework.NewColorByInt32AndInt32AndInt32(1, 2, 3)},
	}
	if err := graphics.DynamicVertexBufferSetDataByInt32AndSliceOfTAndInt32AndInt32AndInt32AndSetDataOptions(
		vertex, 16, marker, 0, 1, 0, graphics.SetDataOptionsNoOverwrite); err != nil {
		g.result.DynamicBufferOptionRefusals++
		return fmt.Errorf("CNA refused an offset NoOverwrite upload on a DYNAMIC vertex buffer: %w", err)
	}
	g.result.DynamicBufferOptionUploads++
	offsetRead := make([]stressVertex, len(written))
	if err := graphics.VertexBufferGetDataBySliceOfT[stressVertex](vertex, offsetRead); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("reading back the offset upload: %w", err)
		}
		fmt.Fprintf(os.Stderr, "the offset readback was refused: %v\n", err)
	} else {
		if offsetRead[0] != written[0] {
			return fmt.Errorf("an upload at byte 16 overwrote vertex 0: wrote %+v, read %+v", written[0], offsetRead[0])
		}
		if offsetRead[1] != marker[0] {
			return fmt.Errorf("an upload at byte 16 did not land on vertex 1: wrote %+v, read %+v", marker[0], offsetRead[1])
		}
		g.result.DynamicBufferRoundTrips++
	}
	g.result.DynamicBufferGuardChecks++

	// The substitutability claim end to end: a dynamic buffer binds where its
	// base does, on the live device, through both positions.
	if err := device.SetVertexBufferByVertexBuffer(vertex); err != nil {
		return fmt.Errorf("binding a DynamicVertexBuffer: %w", err)
	}
	if err := device.SetIndices(index); err != nil {
		return fmt.Errorf("binding a DynamicIndexBuffer: %w", err)
	}
	// The getter answers the BASE, which is the return the settled rule leaves
	// concrete: a consumer who bound a dynamic buffer holds the dynamic object
	// they made and gets its VertexBuffer half back from the device. That the
	// device holds SOMETHING is what is assertable through the public surface.
	if device.Indices() == nil {
		return errors.New("the device holds no index buffer after a DynamicIndexBuffer was bound")
	}
	if device.Indices().IndexCount() != index.IndexCount() {
		return fmt.Errorf("the device holds a %d-index buffer and %d were bound",
			device.Indices().IndexCount(), index.IndexCount())
	}
	// And unbinding through the same positions, which is a NULL to the
	// reference whether the null was typed or not.
	var noVertex *graphics.DynamicVertexBuffer
	if err := device.SetVertexBufferByVertexBuffer(noVertex); err != nil {
		return fmt.Errorf("unbinding with a typed-nil DynamicVertexBuffer: %w", err)
	}
	var noIndex *graphics.DynamicIndexBuffer
	if err := device.SetIndices(noIndex); err != nil {
		return fmt.Errorf("unbinding with a typed-nil DynamicIndexBuffer: %w", err)
	}
	if device.Indices() != nil {
		return errors.New("a typed-nil DynamicIndexBuffer did not unbind")
	}
	g.result.DynamicBufferBindChecks++

	if err := vertex.Dispose(); err != nil {
		return fmt.Errorf("disposing the DynamicVertexBuffer: %w", err)
	}
	if err := index.Dispose(); err != nil {
		return fmt.Errorf("disposing the DynamicIndexBuffer: %w", err)
	}
	if !vertex.IsDisposed() || !index.IsDisposed() {
		return errors.New("a dynamic buffer is not disposed after Dispose")
	}
	// A disposed buffer refuses rather than reaching a released handle, and it
	// names ITSELF when it does -- the identity site, proved on a real object.
	err = graphics.DynamicVertexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		vertex, written, 0, int32(len(written)), graphics.SetDataOptionsNone)
	if err == nil {
		return errors.New("SetData answered on a disposed DynamicVertexBuffer; its CNA handle was not released")
	}
	if !strings.Contains(err.Error(), "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer") {
		return fmt.Errorf("the disposal refusal said %q; the reference names the object's own type", err)
	}
	err = graphics.DynamicIndexBufferSetDataBySliceOfTAndInt32AndInt32AndSetDataOptions(
		index, indices, 0, int32(len(indices)), graphics.SetDataOptionsNone)
	if err == nil {
		return errors.New("SetData answered on a disposed DynamicIndexBuffer")
	}
	if !strings.Contains(err.Error(), "Microsoft.Xna.Framework.Graphics.DynamicIndexBuffer") {
		return fmt.Errorf("the index disposal refusal said %q", err)
	}
	g.result.DynamicBufferDisposalChecks++
	return nil
}

// exerciseSoundEffect is Foundation 87's slice.
//
// # It plays SILENCE, deliberately
//
// Every fixture below is an all-zero PCM16 buffer. That is not a shortcut: the
// qualified artifacts open a REAL playback device -- cna_audio_get_capabilities
// reports is_playback_available=true on both -- so a fixture with signal in it
// would make audible noise on the machine running the suite, twenty times per
// cycle, for no evidence gained.
//
// Silence exercises creation, instance lifetime, transport, state and the
// scalar round trips identically: every one of them is about STRUCTURE, and
// none of them reads a sample back. What silence cannot prove is audibility,
// and this scenario does not claim it -- the same position the ROADMAP already
// records for XACT, where "structural state is not audibility".
//
// # What needs a device and what does not
//
// The managed half -- the four static setters' four different validation
// shapes, the seven constructor guards, the pan/Apply3D mode latch, every
// disposal refusal -- is measured without a runtime in the package tests. What
// needs one is CNA accepting the buffer, the instance existing, the transport
// reaching the mixer, and the state coming back.
func (g *stressGame) exerciseSoundEffect() error {
	// One second of 8kHz mono silence. 8000 is chosen over 44100 because its
	// thousandth IS representable in binary32, so the byte count is the exact
	// 16000 rather than the truncated number 44100 produces -- a fixture whose
	// size is arithmetic rather than a measurement.
	const sampleRate = 8000
	pcm := make([]byte, 16000)

	effect, err := audio.NewSoundEffectBySliceOfByteAndInt32AndAudioChannels(
		pcm, sampleRate, audio.AudioChannelsMono)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewSoundEffect: %w", err)
		}
		g.result.SoundEffectCreationRefusals++
		fmt.Fprintf(os.Stderr, "sound effect creation refused: %v\n", err)
		return nil
	}
	g.result.SoundEffectCreations++

	// The duration CNA computed, which is what the projection stores rather
	// than recomputing it managed-side. One second of 8kHz mono is one second.
	if ticks := effect.Duration().Ticks(); ticks != 10_000_000 {
		return fmt.Errorf("Duration = %d ticks for one second of 8kHz mono silence, want 10000000", ticks)
	}
	if effect.IsDisposed() {
		return errors.New("a fresh sound effect reports itself disposed")
	}
	if effect.Name() != "" {
		return fmt.Errorf("a fresh sound effect is named %q", effect.Name())
	}
	if err := effect.SetName("stress-silence"); err != nil {
		return fmt.Errorf("SetName: %w", err)
	}
	if effect.Name() != "stress-silence" {
		return errors.New("SetName did not store")
	}
	g.result.SoundEffectDescriptionChecks++

	// The seven-argument constructor over the same buffer, with a loop region
	// that covers exactly the frames. A zero loopLength is REWRITTEN by the
	// reference to the whole range and CNA documents the same rule, so both
	// forms must be accepted.
	frames := int32(len(pcm) / 2)
	for name, build := range map[string]func() (*audio.SoundEffect, error){
		"explicit loop": func() (*audio.SoundEffect, error) {
			return audio.NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
				pcm, 0, int32(len(pcm)), sampleRate, audio.AudioChannelsMono, 0, frames)
		},
		"zero loop length": func() (*audio.SoundEffect, error) {
			return audio.NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
				pcm, 0, int32(len(pcm)), sampleRate, audio.AudioChannelsMono, 0, 0)
		},
		"windowed": func() (*audio.SoundEffect, error) {
			return audio.NewSoundEffectBySliceOfByteAndInt32AndInt32AndInt32AndAudioChannelsAndInt32AndInt32(
				pcm, 2, int32(len(pcm))-2, sampleRate, audio.AudioChannelsMono, 0, 0)
		},
	} {
		built, buildErr := build()
		if buildErr != nil {
			return fmt.Errorf("the %s constructor: %w", name, buildErr)
		}
		// The WINDOWED form takes two bytes fewer, which is one frame at mono
		// PCM16 -- and CNA's duration must reflect the COUNT it was given. A
		// creation that ignored the count would report the whole buffer.
		if name == "windowed" {
			if ticks := built.Duration().Ticks(); ticks >= 10_000_000 {
				return fmt.Errorf("a windowed effect reports %d ticks, want less than the whole buffer's 10000000", ticks)
			}
		} else if ticks := built.Duration().Ticks(); ticks != 10_000_000 {
			return fmt.Errorf("the %s effect reports %d ticks, want the whole buffer's 10000000", name, ticks)
		}
		if disposeErr := built.Dispose(); disposeErr != nil {
			return fmt.Errorf("disposing the %s effect: %w", name, disposeErr)
		}
		// Disposal is IDEMPOTENT, which the reference's flag makes it: a second
		// call must succeed rather than reaching a released handle.
		if disposeErr := built.Dispose(); disposeErr != nil {
			return fmt.Errorf("a second Dispose of the %s effect: %w", name, disposeErr)
		}
	}
	g.result.SoundEffectGuardChecks++

	// The four process-wide scalars, each set to a value the class initializer
	// does not use and read back. The getters are managed fields, so what this
	// proves is that the native write was ACCEPTED -- a refused one would
	// leave MasterVolume's field alone and update SpeedOfSound's, which is the
	// asymmetry the package tests pin from the other side.
	if err := audio.SetSoundEffectMasterVolume(0.5); err != nil {
		return fmt.Errorf("SetMasterVolume: %w", err)
	}
	if got := audio.SoundEffectMasterVolume(); got != 0.5 {
		return fmt.Errorf("MasterVolume = %v after a successful set", got)
	}
	if err := audio.SetSoundEffectSpeedOfSound(400); err != nil {
		return fmt.Errorf("SetSpeedOfSound: %w", err)
	}
	if err := audio.SetSoundEffectDopplerScale(0); err != nil {
		return fmt.Errorf("SetDopplerScale(0): %w -- zero is legal there", err)
	}
	if err := audio.SetSoundEffectDistanceScale(0); err != nil {
		return fmt.Errorf("SetDistanceScale(0): %w", err)
	}
	// Zero is CLAMPED to Single.Epsilon, silently, and the getter shows it.
	if got := audio.SoundEffectDistanceScale(); got == 0 {
		return errors.New("SetDistanceScale(0) stored a zero; the reference clamps to Single.Epsilon")
	}
	g.result.SoundEffectScalarChecks++
	// Put them back so the rest of the run sees the defaults.
	_ = audio.SetSoundEffectMasterVolume(1)
	_ = audio.SetSoundEffectSpeedOfSound(343.5)
	_ = audio.SetSoundEffectDopplerScale(1)
	_ = audio.SetSoundEffectDistanceScale(1)

	// Play, which needs the dispatcher to have run. exerciseRootStatics pumps
	// it three times immediately before this slice, so the precondition is met
	// -- and a run that reordered them would fail here rather than silently.
	played, err := effect.PlayByNone()
	if err != nil {
		return fmt.Errorf("SoundEffect.Play: %w", err)
	}
	if played {
		g.result.SoundEffectPlays++
	} else {
		// FALSE means the voice limit, and nothing else.
		g.result.SoundEffectPlayLimitChecks++
	}

	// An instance, and its whole transport.
	instance, err := effect.CreateInstance()
	if err != nil {
		return fmt.Errorf("CreateInstance: %w", err)
	}
	g.result.SoundInstanceCreations++
	if state, stateErr := instance.State(); stateErr != nil {
		return fmt.Errorf("a fresh instance's State: %w", stateErr)
	} else if state != audio.SoundStateStopped {
		return fmt.Errorf("a fresh instance is %v, want Stopped", state)
	}
	for _, step := range []struct {
		name string
		call func() error
		want audio.SoundState
	}{
		{"Play", instance.Play, audio.SoundStatePlaying},
		{"Pause", instance.Pause, audio.SoundStatePaused},
		{"Resume", instance.Resume, audio.SoundStatePlaying},
		{"Stop", instance.StopByNone, audio.SoundStateStopped},
	} {
		if err := step.call(); err != nil {
			return fmt.Errorf("SoundEffectInstance.%s: %w", step.name, err)
		}
		state, stateErr := instance.State()
		if stateErr != nil {
			return fmt.Errorf("State after %s: %w", step.name, stateErr)
		}
		if state != step.want {
			return fmt.Errorf("after %s the instance is %v, want %v", step.name, state, step.want)
		}
		g.result.SoundInstanceTransitions++
	}

	// IsLooped must be set BEFORE the first Play, on both sides: the reference
	// refuses once its packet is submitted and CNA refuses "after playback has
	// begun". The instance above has already played, so a fresh one is what
	// this claim needs -- and the refusal on the PLAYED one is asserted too,
	// because a projection that dropped the guard would pass the first half.
	if err := instance.SetIsLooped(true); err == nil {
		return errors.New("SetIsLooped was accepted after Play; the packet is submitted by then")
	}
	loopable, err := effect.CreateInstance()
	if err != nil {
		return fmt.Errorf("an instance for the loop flag: %w", err)
	}
	if err := loopable.SetIsLooped(true); err != nil {
		return fmt.Errorf("SetIsLooped before any Play: %w", err)
	}
	if !loopable.IsLooped() {
		return errors.New("SetIsLooped did not store")
	}
	if err := loopable.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the loopable instance: %w", err)
	}
	g.result.SoundInstanceScalarRoundTrips++

	// Resume on a STOPPED instance, which is what tells Resume apart from Play.
	//
	// MEASURED: a stopped instance that is resumed stays STOPPED. Resume lifts
	// a pause and does not start playback, so a projection that routed it to
	// cna_sound_effect_instance_play would leave the instance Playing here --
	// and every other transport assertion in this slice would still pass,
	// because the two agree everywhere except on a stopped instance.
	if err := instance.StopByNone(); err != nil {
		return fmt.Errorf("stopping before the resume check: %w", err)
	}
	if err := instance.Resume(); err != nil {
		return fmt.Errorf("Resume on a stopped instance: %w", err)
	}
	state, stateErr := instance.State()
	if stateErr != nil {
		return fmt.Errorf("State after resuming a stopped instance: %w", stateErr)
	}
	if state != audio.SoundStateStopped {
		return fmt.Errorf("a stopped instance that was resumed is %v, want Stopped -- Resume lifts a pause and does not start playback", state)
	}
	g.result.SoundInstanceTransitions++

	// The two scalars that have no such precondition, each set and read back.
	// The getter is a managed field the setter stores AFTER its native write,
	// so a value that comes back is one CNA accepted.
	for name, check := range map[string]func() error{
		"Volume": func() error {
			if err := instance.SetVolume(0.25); err != nil {
				return err
			}
			if got := instance.Volume(); got != 0.25 {
				return fmt.Errorf("Volume = %v", got)
			}
			return nil
		},
		"Pitch": func() error {
			if err := instance.SetPitch(-0.5); err != nil {
				return err
			}
			if got := instance.Pitch(); got != -0.5 {
				return fmt.Errorf("Pitch = %v", got)
			}
			return nil
		},
	} {
		if err := check(); err != nil {
			return fmt.Errorf("the %s round trip: %w", name, err)
		}
		g.result.SoundInstanceScalarRoundTrips++
	}

	// The MODE LATCH, end to end. A fresh instance is neither 2D nor 3D; the
	// first of the two members to be called decides, and after a Play the other
	// one refuses.
	panned, err := effect.CreateInstance()
	if err != nil {
		return fmt.Errorf("an instance for the pan half: %w", err)
	}
	if err := panned.SetPan(0.5); err != nil {
		return fmt.Errorf("SetPan on a fresh instance: %w", err)
	}
	if got := panned.Pan(); got != 0.5 {
		return fmt.Errorf("Pan = %v after a successful set", got)
	}
	// Apply3D on the SAME instance is still legal, because no packet has been
	// submitted -- the guard clears the flag each time.
	if err := panned.Apply3DByAudioListenerAndAudioEmitter(
		audio.NewAudioListener(), audio.NewAudioEmitter()); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("Apply3D before any Play: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Apply3D refused before Play: %v\n", err)
	} else {
		g.result.SoundInstanceApply3DChecks++
	}
	if err := panned.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the panned instance: %w", err)
	}
	g.result.SoundInstanceModeLatchChecks++

	// Disposal, and the ordering CNA requires: the effect releases its
	// instances before releasing itself, so disposing the EFFECT while an
	// instance is still live must succeed.
	if err := effect.Dispose(); err != nil {
		return fmt.Errorf("disposing the effect with a live instance: %w", err)
	}
	if !effect.IsDisposed() || !instance.IsDisposed() {
		return errors.New("disposing the effect did not dispose its instance")
	}
	// A disposed effect refuses rather than reaching a released handle, and
	// names ITSELF when it does.
	if _, err := effect.CreateInstance(); err == nil {
		return errors.New("CreateInstance answered on a disposed SoundEffect")
	} else if !strings.Contains(err.Error(), "SoundEffect") {
		return fmt.Errorf("the disposal refusal said %q", err)
	}
	if err := instance.Play(); err == nil {
		return errors.New("Play answered on a disposed SoundEffectInstance")
	}
	// The instance's own disposal is idempotent too, and it was already
	// disposed by its effect -- so this is a SECOND release of a handle CNA has
	// already destroyed, which the managed flag must stop before it happens.
	if err := instance.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing an already-released instance: %w", err)
	}
	// get_Name has NO disposal check, so it still answers.
	if effect.Name() != "stress-silence" {
		return errors.New("a disposed effect stopped answering its name")
	}
	g.result.SoundEffectDisposalChecks++
	return nil
}

// exerciseDynamicSoundEffectInstance is Foundation 88's streaming slice.
//
// It plays SILENCE, for the reason the SoundEffect slice does: the qualified
// artifacts open a real playback device.
func (g *stressGame) exerciseDynamicSoundEffectInstance() error {
	// 22050 rather than 8000, and the choice is load-bearing: its conversion
	// answer is DISTINCTIVE. One second of 22050Hz mono is 44098 bytes -- the
	// truncated number the float32 scale factor produces -- where 8000Hz gives
	// the round 16000. An instance built with the wrong rate answers the wrong
	// number, and a fixture at 8000 would agree with a projection that
	// hardcoded 8000.
	const sampleRate = 22050
	instance, err := audio.NewDynamicSoundEffectInstance(sampleRate, audio.AudioChannelsMono)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("NewDynamicSoundEffectInstance: %w", err)
		}
		g.result.DynamicInstanceRefusals++
		fmt.Fprintf(os.Stderr, "dynamic instance creation refused: %v\n", err)
		return nil
	}
	g.result.DynamicInstanceCreations++

	// It can NEVER loop, and both halves are proved on a live object: the
	// getter always answers false and the setter refuses true.
	looped, err := instance.IsLooped()
	if err != nil {
		return fmt.Errorf("IsLooped: %w", err)
	}
	if looped {
		return errors.New("a fresh DynamicSoundEffectInstance reports itself looped")
	}
	if err := instance.SetIsLooped(true); err == nil {
		return errors.New("SetIsLooped(true) was accepted; a streaming instance has no loop")
	}
	if err := instance.SetIsLooped(false); err != nil {
		return fmt.Errorf("SetIsLooped(false): %w", err)
	}
	g.result.DynamicInstanceLoopChecks++

	// The two conversions, which unlike SoundEffect's STATIC pair really do
	// reach CNA. One second of 8kHz mono is 16000 bytes, and CNA must agree.
	oneSecond := framework.TimeSpanFromTicks(10_000_000)
	size, err := instance.GetSampleSizeInBytes(oneSecond)
	if err != nil {
		return fmt.Errorf("GetSampleSizeInBytes: %w", err)
	}
	// The exact number, because it is the one the instance's OWN sample rate
	// produces. 22050 mono PCM16 truncates to 22049 samples and 44098 bytes.
	if size != 44098 {
		return fmt.Errorf("one second at %dHz mono = %d bytes, want the measured 44098", sampleRate, size)
	}
	back, err := instance.GetSampleDuration(size)
	if err != nil {
		return fmt.Errorf("GetSampleDuration: %w", err)
	}
	if back.Ticks() <= 0 {
		return fmt.Errorf("GetSampleDuration answered %d ticks for %d bytes", back.Ticks(), size)
	}
	g.result.DynamicInstanceConversionChecks++

	// A submission of silence, and the pending count that reports it. The count
	// is a LIVE read, unlike every cached scalar on the base.
	pcm := make([]byte, 1600)
	before, err := instance.PendingBufferCount()
	if err != nil {
		return fmt.Errorf("PendingBufferCount: %w", err)
	}
	if before != 0 {
		return fmt.Errorf("a fresh streaming instance has %d pending buffers", before)
	}
	if err := instance.SubmitBufferBySliceOfByte(pcm); err != nil {
		return fmt.Errorf("SubmitBuffer: %w", err)
	}
	g.result.DynamicInstanceSubmissions++
	after, err := instance.PendingBufferCount()
	if err != nil {
		return fmt.Errorf("PendingBufferCount after a submission: %w", err)
	}
	if after <= before {
		return fmt.Errorf("a submitted buffer did not raise the pending count: %d then %d", before, after)
	}
	g.result.DynamicInstancePendingChecks++

	// The projection's own refusals, before CNA sees anything.
	if err := instance.SubmitBufferBySliceOfByteAndInt32AndInt32(pcm, 1, 4); err == nil {
		return errors.New("a misaligned offset was accepted")
	}
	if err := instance.SubmitBufferBySliceOfByteAndInt32AndInt32(pcm, 0, 5); err == nil {
		return errors.New("a misaligned count was accepted")
	}

	if err := instance.Play(); err != nil {
		return fmt.Errorf("DynamicSoundEffectInstance.Play: %w", err)
	}
	if err := instance.StopByNone(); err != nil {
		return fmt.Errorf("stopping the streaming instance: %w", err)
	}
	if err := instance.DisposeByNone(); err != nil {
		return fmt.Errorf("disposing the streaming instance: %w", err)
	}
	if !instance.IsDisposed() {
		return errors.New("the streaming instance is not disposed after Dispose")
	}
	// A disposed instance names ITSELF, which is the identity site made live.
	if _, err := instance.IsLooped(); err == nil {
		return errors.New("IsLooped answered on a disposed streaming instance")
	} else if !strings.Contains(err.Error(), "DynamicSoundEffectInstance") {
		return fmt.Errorf("the disposal refusal said %q; the reference names the object's own type", err)
	}
	g.result.DynamicInstanceDisposalChecks++
	return nil
}

// exerciseMicrophone is Foundation 88's enumeration slice.
//
// # It NEVER starts capture
//
// Microphone.Start and Microphone.GetData are projected because the pinned
// contract declares them, and this scenario calls NEITHER. Starting capture
// opens a real recording device on whatever machine the suite runs on, and that
// is not something a test suite does. The counter MICROPHONE_CAPTURE_CALLS
// exists to be ZERO and to say so out loud: a run that started reporting a
// non-zero value would be a run that had begun recording.
//
// What IS exercised is everything else: the count, the default, each device's
// name, buffer duration, headset flag, sample rate and state, the two sample
// conversions, and every managed guard. Stop is called because stopping a
// device that is not capturing is safe and is the reference's own no-op.
func (g *stressGame) exerciseMicrophone() error {
	all, err := audio.MicrophoneAll()
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("Microphone.All: %w", err)
		}
		fmt.Fprintf(os.Stderr, "microphone enumeration refused: %v\n", err)
		return nil
	}
	g.result.MicrophoneEnumerations++
	count := all.Count()
	g.result.MicrophonesFound += int(count)

	// The default, which may legitimately be absent: CNA reports availability
	// separately from the index, and the reference answers null.
	defaultMicrophone, err := audio.MicrophoneDefault()
	if err != nil {
		return fmt.Errorf("Microphone.Default: %w", err)
	}
	if count == 0 && defaultMicrophone != nil {
		return errors.New("a machine with no microphones reported a default one")
	}

	for index := int32(0); index < count; index++ {
		microphone, itemErr := all.Item(index)
		if itemErr != nil {
			return fmt.Errorf("Microphone.All[%d]: %w", index, itemErr)
		}
		if microphone == nil {
			return fmt.Errorf("Microphone.All[%d] is nil", index)
		}
		// Every description read, none of which touches capture.
		if _, err := microphone.SampleRate(); err != nil {
			return fmt.Errorf("microphone %d SampleRate: %w", index, err)
		}
		if _, err := microphone.IsHeadset(); err != nil {
			return fmt.Errorf("microphone %d IsHeadset: %w", index, err)
		}
		state, stateErr := microphone.State()
		if stateErr != nil {
			return fmt.Errorf("microphone %d State: %w", index, stateErr)
		}
		// Nothing has started it, so it must be stopped.
		if state != audio.MicrophoneStateStopped {
			return fmt.Errorf("microphone %d is %v before anything started it", index, state)
		}
		// The two conversions, which are instance members here.
		size, sizeErr := microphone.GetSampleSizeInBytes(framework.TimeSpanFromTicks(10_000_000))
		if sizeErr != nil {
			return fmt.Errorf("microphone %d GetSampleSizeInBytes: %w", index, sizeErr)
		}
		if _, durationErr := microphone.GetSampleDuration(size); durationErr != nil {
			return fmt.Errorf("microphone %d GetSampleDuration: %w", index, durationErr)
		}
		// Stop on a device that is not capturing is the reference's no-op.
		if stopErr := microphone.Stop(); stopErr != nil && !isNativeRefusal(stopErr) {
			return fmt.Errorf("microphone %d Stop: %w", index, stopErr)
		}
		g.result.MicrophoneDescriptionChecks++
	}

	// The managed guards, which need no device at all -- so they run whether or
	// not this machine has a microphone.
	probe := &audio.Microphone{}
	if err := probe.SetBufferDuration(framework.TimeSpanFromTicks(90 * 10000)); err == nil {
		return errors.New("a 90ms buffer duration was accepted; the floor is 100ms")
	}
	if err := probe.SetBufferDuration(framework.TimeSpanFromTicks(105 * 10000)); err == nil {
		return errors.New("a 105ms buffer duration was accepted; it must be 10ms aligned")
	}
	if _, err := probe.GetSampleDuration(-1); err == nil {
		return errors.New("a negative size was accepted")
	}
	if _, err := probe.GetDataBySliceOfByte(nil); err == nil {
		return errors.New("GetData accepted an empty buffer")
	}
	g.result.MicrophoneGuardChecks++
	return nil
}

// exerciseInputStatics is Foundation 89's slice: GamePad and Mouse against the
// live event state, and TouchPanel against nothing at all.
//
// # It calls SetVibration only with two zeros
//
// SetVibration is the one member in this family that DRIVES hardware rather
// than sampling it, and a test suite that spun a stranger's controller motors
// would be doing something a test suite does not do. Both magnitudes are 0.0
// on every call, which is the value the reference itself uses to stop a motor.
// The counters record the calls and how many were applied, so a run cannot
// quietly skip them.
//
// # A machine with no controller is a RESULT, not a skip
//
// The reference answers an empty GamePadState with IsConnected false when
// XInput reports ERROR_DEVICE_NOT_CONNECTED, and does NOT throw. This build
// machine has no controller attached, so that branch is the one the run
// actually takes -- and it is the branch worth proving, because a projection
// that turned a missing controller into an error would break the polling loop
// every game writes. GAMEPADS_CONNECTED reports whatever is there.
//
// # TouchPanel is here to prove it reaches nothing
//
// Every TouchPanel member is exercised inside a live game with a real native
// runtime available, and TOUCH_PANEL_NATIVE_CALLS must still be zero. That is
// the assertion the whole touch finding rests on.
func (g *stressGame) exerciseInputStatics() error {
	indices := []framework.PlayerIndex{
		framework.PlayerIndexOne, framework.PlayerIndexTwo,
		framework.PlayerIndexThree, framework.PlayerIndexFour,
	}
	for _, index := range indices {
		capabilities, err := input.GamePadGetCapabilities(index)
		if err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("GamePad.GetCapabilities(%v): %w", index, err)
			}
			fmt.Fprintf(os.Stderr, "gamepad capabilities refused: %v\n", err)
		}
		g.result.GamePadCapabilityReads++

		state, err := input.GamePadGetStateByPlayerIndex(index)
		if err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("GamePad.GetState(%v): %w", index, err)
			}
			fmt.Fprintf(os.Stderr, "gamepad state refused: %v\n", err)
		}
		g.result.GamePadStateReads++

		// The two must AGREE about presence. A projection that read the
		// connected flag from the wrong place would disagree here, and this is
		// the one cross-check no managed test can make.
		if capabilities.IsConnected() != state.IsConnected() {
			return fmt.Errorf("GamePad %v: capabilities say connected=%v and state says connected=%v",
				index, capabilities.IsConnected(), state.IsConnected())
		}
		if state.IsConnected() {
			g.result.GamePadsConnected++
		} else {
			// The empty state the ERROR_DEVICE_NOT_CONNECTED branch returns.
			// Every value in it must be zero, which is what makes a
			// disconnected controller safe to poll without checking first.
			if state.PacketNumber() != 0 {
				return fmt.Errorf("GamePad %v is disconnected but reported packet %d",
					index, state.PacketNumber())
			}
			if capabilities.GamePadType() != input.GamePadTypeUnknown {
				return fmt.Errorf("GamePad %v is disconnected but reported type %v",
					index, capabilities.GamePadType())
			}
		}

		// Two zeros: a stop, never a start. The Boolean says whether it was
		// applied, and a controller that is not there cannot apply it.
		applied, err := input.GamePadSetVibration(index, 0, 0)
		if err != nil {
			if !isNativeRefusal(err) {
				return fmt.Errorf("GamePad.SetVibration(%v): %w", index, err)
			}
			fmt.Fprintf(os.Stderr, "gamepad vibration refused: %v\n", err)
		}
		g.result.GamePadVibrationCalls++
		if applied {
			g.result.GamePadVibrationsApplied++
			if !state.IsConnected() {
				return fmt.Errorf("GamePad %v applied a vibration while disconnected", index)
			}
		}
	}

	// Mouse. The position write goes to where the cursor already is, so the
	// run does not move a pointer the user may be holding.
	mouse, err := input.MouseGetState()
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("Mouse.GetState: %w", err)
		}
		fmt.Fprintf(os.Stderr, "mouse state refused: %v\n", err)
	}
	g.result.MouseStateReads++
	if err := input.MouseSetPosition(mouse.X(), mouse.Y()); err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.SetPosition: %w", err)
	}
	g.result.MousePositionWrites++

	// The window handle round-trips through the native side, which is the one
	// thing about it a managed test cannot check.
	//
	// A DIFFERENT value has to go across for the round trip to prove anything.
	// Writing the handle back over itself proves nothing here: the HEADLESS
	// artifact starts with no hooked window, so the value is 0 and a dropped
	// write and an honest one both read back the same 0. Measured, not assumed
	// -- both Mouse.WindowHandle and GameWindow.Handle report 0x0 on this
	// artifact.
	//
	// So a SENTINEL is written and then unbound. CNA's own header says the
	// parameter is an "opaque native window value; zero unbinds", so nothing
	// dereferences it, and the write is the last thing this slice does with the
	// mouse -- the state read and the position write are already behind it.
	handle, err := input.MouseWindowHandle()
	if err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.WindowHandle: %w", err)
	}
	const mouseHandleSentinel = uintptr(0x5CA1AB1E)
	if err := input.SetMouseWindowHandle(mouseHandleSentinel); err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.set_WindowHandle(sentinel): %w", err)
	}
	written, err := input.MouseWindowHandle()
	if err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.WindowHandle after the write: %w", err)
	}
	if written != mouseHandleSentinel {
		return fmt.Errorf("mouse window handle was set to %#x and read back %#x",
			mouseHandleSentinel, written)
	}
	// Restore what was there, which on this artifact unbinds again.
	if err := input.SetMouseWindowHandle(handle); err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.set_WindowHandle: %w", err)
	}
	readBack, err := input.MouseWindowHandle()
	if err != nil && !isNativeRefusal(err) {
		return fmt.Errorf("Mouse.WindowHandle after restoring: %w", err)
	}
	if readBack != handle {
		return fmt.Errorf("mouse window handle did not round-trip: %#x became %#x", handle, readBack)
	}
	g.result.MouseHandleChecks++

	// TouchPanel, inside a live game with a native runtime available. Nothing
	// below may reach it.
	if capabilities := touch.TouchPanelGetCapabilities(); capabilities.IsConnected() {
		return errors.New("TouchPanel reported a connected panel inside a live game; GetCaps returns a zeroed struct")
	}
	state := touch.TouchPanelGetState()
	if state.Count() != 0 {
		return fmt.Errorf("TouchPanel reported %d touches inside a live game; GetState updates from a zeroed state",
			state.Count())
	}
	if !state.IsConnected() {
		return errors.New("TouchPanel state reported disconnected; Update is called with the literal true")
	}
	if _, err := touch.TouchPanelReadGesture(); err == nil {
		return errors.New("ReadGesture returned a sample; its body has no ret instruction")
	}
	if err := touch.SetTouchPanelEnabledGestures(touch.GestureTypeTap); err != nil {
		return fmt.Errorf("TouchPanel.set_EnabledGestures(Tap): %w", err)
	}
	available, err := touch.TouchPanelIsGestureAvailable()
	if err != nil {
		return fmt.Errorf("TouchPanel.IsGestureAvailable after assignment: %w", err)
	}
	if available {
		return errors.New("IsGestureAvailable was true inside a live game; the reference returns ldc.i4.0")
	}
	if _, err := touch.TouchPanelReadGesture(); err == nil {
		return errors.New("ReadGesture returned a sample after gestures were enabled")
	}
	g.result.TouchPanelManagedChecks++
	return nil
}

// exerciseStorage is Foundation 91's slice, and it opens by proving where it is
// allowed to write.
//
// # Containment is enforced, not assumed
//
// The standing constraint is that a test uses only a project-controlled root
// and never the host's own directories. On this platform CNA builds its root
// from XDG_DATA_HOME, so an unconfigured run would write to
// ~/.local/share/<app> -- outside the project, in the user's home. That was
// measured rather than guessed: the slice printed the root before it was
// allowed to do anything else.
//
// So the harness sets XDG_DATA_HOME to a directory inside the repository, and
// this slice REFUSES TO CONTINUE unless the root it reads back actually lives
// there. A run that cannot prove containment does nothing and says so, which is
// the same shape as MICROPHONE_CAPTURE_CALLS: a safety claim is worth more when
// something measures it.
func (g *stressGame) exerciseStorage() error {
	const appName = "cna-go-stress-storage"
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	// The root this run is permitted to touch. Without it, nothing below runs.
	permitted := os.Getenv("CNA_GO_STORAGE_ROOT")
	if permitted == "" {
		fmt.Fprintln(os.Stderr, "storage slice skipped: CNA_GO_STORAGE_ROOT names no project-controlled root")
		return nil
	}
	if err := runtime.StorageSetApplicationName(appName); err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("storage set app name: %w", err)
		}
		fmt.Fprintf(os.Stderr, "storage app name refused: %v\n", err)
		return nil
	}
	root, err := runtime.StorageRoot()
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("storage root: %w", err)
		}
		fmt.Fprintf(os.Stderr, "storage root refused: %v\n", err)
		return nil
	}
	// THE CONTAINMENT PROOF. A root outside the permitted directory fails the
	// run rather than being worked around.
	if !strings.HasPrefix(root, permitted) {
		return fmt.Errorf("storage root %q is outside the permitted root %q; refusing to touch it",
			root, permitted)
	}
	g.result.StorageRootChecks++

	// Only now may anything be written.
	device, err := storage.StorageDeviceBeginShowSelectorByAsyncCallbackAndObject(nil, nil)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("BeginShowSelector: %w", err)
		}
		fmt.Fprintf(os.Stderr, "storage selector refused: %v\n", err)
		return nil
	}
	selected, err := storage.StorageDeviceEndShowSelector(device)
	if err != nil {
		return fmt.Errorf("EndShowSelector: %w", err)
	}
	g.result.StorageSelectorCycles++

	// A second End on the same result is the reference's CannotEndTwice, and it
	// is the one refusal in this family a managed test cannot reach through a
	// real device.
	if _, err := storage.StorageDeviceEndShowSelector(device); err == nil {
		return errors.New("a second EndShowSelector succeeded; the reference refuses it")
	}

	if _, err := selected.IsConnected(); err != nil {
		return fmt.Errorf("IsConnected: %w", err)
	}
	if _, err := selected.FreeSpace(); err != nil {
		return fmt.Errorf("FreeSpace: %w", err)
	}
	if _, err := selected.TotalSpace(); err != nil {
		return fmt.Errorf("TotalSpace: %w", err)
	}
	g.result.StorageDeviceReads++

	opened, err := selected.BeginOpenContainer("stress", nil, nil)
	if err != nil {
		return fmt.Errorf("BeginOpenContainer: %w", err)
	}
	container, err := selected.EndOpenContainer(opened)
	if err != nil {
		return fmt.Errorf("EndOpenContainer: %w", err)
	}
	g.result.StorageContainerCycles++

	if name, err := container.DisplayName(); err != nil {
		return fmt.Errorf("DisplayName: %w", err)
	} else if name == "" {
		return errors.New("the container reported an empty display name")
	}

	// A file, written and read back. This is the only place the projection's
	// io.ReadWriteSeeker claim is actually exercised.
	payload := []byte("cna-go stress payload")
	// Start from a known state. A run that failed part-way through leaves the
	// file behind, and the next run could then not tell CreateFile from an
	// OpenFile of something that already existed.
	if present, err := container.FileExists("stress.dat"); err != nil {
		return fmt.Errorf("FileExists before create: %w", err)
	} else if present {
		if err := container.DeleteFile("stress.dat"); err != nil {
			return fmt.Errorf("DeleteFile before create: %w", err)
		}
	}
	stream, err := container.CreateFile("stress.dat")
	if err != nil {
		return fmt.Errorf("CreateFile: %w", err)
	}
	written, err := stream.Write(payload)
	if err != nil || written != len(payload) {
		return fmt.Errorf("Write = %d, %v", written, err)
	}
	if closer, ok := stream.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("Close after write: %w", err)
		}
	}
	g.result.StorageFileWrites++

	exists, err := container.FileExists("stress.dat")
	if err != nil {
		return fmt.Errorf("FileExists: %w", err)
	}
	if !exists {
		return errors.New("the file just written does not exist")
	}

	reopened, err := container.OpenFileByStringAndFileMode("stress.dat", framework.FileModeOpen)
	if err != nil {
		return fmt.Errorf("OpenFile: %w", err)
	}
	buffer := make([]byte, len(payload))
	read, err := io.ReadFull(reopened, buffer)
	if err != nil {
		return fmt.Errorf("ReadFull = %d, %w", read, err)
	}
	if string(buffer) != string(payload) {
		return fmt.Errorf("read back %q, want %q", buffer, payload)
	}
	// Seek back and read the first byte again, which is what makes this a
	// ReadWriteSeeker rather than a ReadWriter.
	if _, err := reopened.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("Seek: %w", err)
	}
	single := make([]byte, 1)
	if _, err := reopened.Read(single); err != nil {
		return fmt.Errorf("Read after Seek: %w", err)
	}
	if single[0] != payload[0] {
		return fmt.Errorf("after seeking to the start the first byte was %q", single)
	}
	// Read PAST the end. CNA reports the end as a zero-length read with no
	// error; io.Reader requires io.EOF, and this is the only place that
	// translation is observable.
	if _, err := reopened.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("Seek to end: %w", err)
	}
	if _, err := reopened.Read(single); !errors.Is(err, io.EOF) {
		return fmt.Errorf("reading past the end = %v, want io.EOF", err)
	}
	if closer, ok := reopened.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("Close after read: %w", err)
		}
	}
	g.result.StorageFileReads++

	// The enumeration, which is the two-step count-then-copy path.
	names, err := container.GetFileNamesByNone()
	if err != nil {
		return fmt.Errorf("GetFileNames: %w", err)
	}
	found := false
	for _, name := range names {
		if strings.Contains(name, "stress.dat") {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("GetFileNames answered %v, which does not include the file just written", names)
	}
	g.result.StorageEnumerations++

	// A directory, created and enumerated and removed.
	if err := container.CreateDirectory("sub"); err != nil {
		return fmt.Errorf("CreateDirectory: %w", err)
	}
	present, err := container.DirectoryExists("sub")
	if err != nil || !present {
		return fmt.Errorf("DirectoryExists = %v, %v", present, err)
	}
	if err := container.DeleteDirectory("sub"); err != nil {
		return fmt.Errorf("DeleteDirectory: %w", err)
	}
	g.result.StorageDirectoryCycles++

	// Clean up the file, then dispose. A run must not leave the next one a
	// pre-existing file to trip over.
	if err := container.DeleteFile("stress.dat"); err != nil {
		return fmt.Errorf("DeleteFile: %w", err)
	}
	if present, err := container.FileExists("stress.dat"); err != nil {
		return fmt.Errorf("FileExists after delete: %w", err)
	} else if present {
		return errors.New("the file still exists after DeleteFile")
	}
	if err := container.Dispose(); err != nil {
		return fmt.Errorf("Dispose: %w", err)
	}
	disposed, err := container.IsDisposed()
	if err != nil || !disposed {
		return fmt.Errorf("IsDisposed after Dispose = %v, %v", disposed, err)
	}
	// A disposed container refuses, which is VerifyNotDisposed.
	if _, err := container.FileExists("stress.dat"); err == nil {
		return errors.New("a disposed container answered FileExists")
	}
	g.result.StorageDisposalChecks++
	return nil
}

// exercisePresentation is Foundation 73's scenario: the rest of GraphicsDevice
// against a live one.
//
// It reads the presentation parameters back out of CNA, resets the device three
// ways, proves the rectangle Present overload refuses for a CONTRACT reason
// rather than a renderer one, reads the back buffer, creates and binds a cube
// render target, and creates a device the CONSUMER owns and destroys it.
//
// # What may refuse, and what may not
//
// CNA declares `cna_graphics_device_reset` unconditionally, so a refusal there
// is a defect and fails the run. It documents the back-buffer readback and cube
// render targets as renderer capabilities, so a refusal from either is recorded
// as a limitation -- the same shape the render-target and volume scenarios have.
//
// The rectangle Present overload's refusal is neither: it is CNA-Go declining to
// present the WHOLE back buffer under a name that promises a sub-rectangle, so
// it must happen on every artifact in every cycle.
func (g *stressGame) exercisePresentation() error {
	device := g.device
	if device == nil {
		return errors.New("the presentation scenario ran with no device")
	}

	// The parameters CNA is actually running with. This is a real read: the
	// getter calls cna_graphics_device_get_presentation_parameters and builds a
	// fresh managed object from the reported struct.
	parameters, parametersErr := device.PresentationParameters()
	if parametersErr != nil {
		return fmt.Errorf("PresentationParameters: %w", parametersErr)
	}
	if parameters == nil {
		return errors.New("PresentationParameters answered nil with no error")
	}
	// A back buffer of no size would mean the struct never crossed the boundary,
	// which is the failure this read exists to catch.
	if parameters.BackBufferWidth() <= 0 || parameters.BackBufferHeight() <= 0 {
		return fmt.Errorf("CNA reported a %dx%d back buffer",
			parameters.BackBufferWidth(), parameters.BackBufferHeight())
	}
	// Clone is a MANAGED copy, so mutating it must not touch the original --
	// which is what proves the getter hands back an object rather than a view.
	clone := parameters.Clone()
	clone.SetBackBufferWidth(parameters.BackBufferWidth() + 32)
	if parameters.BackBufferWidth() == clone.BackBufferWidth() {
		return errors.New("PresentationParameters.Clone answered the same object")
	}
	g.result.PresentationParameterReads++

	// The three Resets, in the order the overload group is declared. Each one
	// reaches cna_graphics_device_reset; the two argument-carrying ones send the
	// parameters CNA just reported, so the device ends where it started.
	adapter, adapterErr := device.Adapter()
	if adapterErr != nil {
		return fmt.Errorf("GraphicsDevice.Adapter: %w", adapterErr)
	}
	for _, reset := range []struct {
		name string
		call func() error
	}{
		{"Reset()", device.ResetByNone},
		{"Reset(pp)", func() error { return device.ResetByPresentationParameters(parameters) }},
		{"Reset(pp, adapter)", func() error {
			return device.ResetByPresentationParametersAndGraphicsAdapter(parameters, adapter)
		}},
	} {
		switch err := reset.call(); {
		case err == nil:
			g.result.PresentationResetCalls++
		case isNativeRefusal(err):
			// Recorded rather than ignored: CNA declares this route
			// unconditionally, so a refusal is a divergence worth seeing in the
			// report even though the aggregate check below tolerates it.
			g.result.PresentationResetRefusals++
			fmt.Fprintf(os.Stderr, "%s refused by CNA: %v\n", reset.name, err)
		default:
			return fmt.Errorf("%s: %w", reset.name, err)
		}
	}

	// The rectangle Present overload. Its refusal names CNA's one present route
	// and the three things that route cannot carry, and it must NOT be a native
	// refusal -- CNA never sees this call.
	source := framework.NewRectangle(0, 0, 4, 4)
	presentErr := device.PresentByNullableOfRectangleAndNullableOfRectangleAndIntPtr(&source, &source, 0)
	if presentErr == nil {
		return errors.New("the rectangle Present overload presented something")
	}
	if isNativeRefusal(presentErr) {
		return fmt.Errorf("the rectangle Present overload reached CNA: %w", presentErr)
	}
	g.result.PresentationRectangleRefusals++

	// The active-render-target guard, which needs no renderer: it is the Go
	// side of GetBackBufferData and is proved on its own.
	if guardErr := checkBackBufferGuardIsReachable(device); guardErr != nil {
		return guardErr
	}
	g.result.BackBufferGuardChecks++

	// The back-buffer read itself, over the whole buffer at the size CNA just
	// reported. A renderer with no readback path refuses, which is recorded.
	//
	// # This is the profile's first real PIXEL readback
	//
	// Every draw proof before Foundation 73 was VERIFIED_NATIVE_DRAW: CNA
	// accepted the submission and no artifact could read the result back. So the
	// buffer is CLEARED to a colour nothing else in this process uses, and the
	// pixels that come back are checked against it. A renderer that can read
	// back and answers the wrong colour is a defect and fails the run; one that
	// refuses is recorded as the limitation it is.
	marker := framework.NewColorByInt32AndInt32AndInt32AndInt32(17, 34, 51, 255)
	if clearErr := device.ClearByColor(marker); clearErr != nil && !isNativeRefusal(clearErr) {
		return fmt.Errorf("Clear before the back-buffer read: %w", clearErr)
	}
	pixels := make([]framework.Color, parameters.BackBufferWidth()*parameters.BackBufferHeight())
	switch readErr := graphics.GraphicsDeviceGetBackBufferDataBySliceOfT(device, pixels); {
	case readErr == nil:
		g.result.BackBufferReads++
		if err := checkClearedPixels(pixels, marker); err != nil {
			return err
		}
		g.result.BackBufferPixelChecks++
	case isNativeRefusal(readErr):
		g.result.BackBufferReadRefusals++
		fmt.Fprintf(os.Stderr, "back-buffer readback refused: %v\n", readErr)
	default:
		return fmt.Errorf("GetBackBufferData: %w", readErr)
	}

	// Foundation 85. The first draw in the project whose evidence is the
	// TEXELS rather than CNA's acceptance.
	if err := g.exercisePixelDraw(device, parameters); err != nil {
		return err
	}

	if err := g.exerciseCubeRenderTarget(device); err != nil {
		return err
	}
	return g.exerciseOwnedDevice(device)
}

// exercisePixelDraw is Foundation 85's slice, and it is the first in the
// project to make a DRAW's evidence VERIFIED_PIXEL rather than
// VERIFIED_NATIVE_DRAW.
//
// # What changed, and why it could not be done before
//
// Foundation 73 read the back buffer for the first time and checked it against
// a CLEAR colour. Every draw proof since has stopped at "CNA accepted the
// submission": the qualified HEADLESS artifact has no readback path, and until
// the stock effects landed there was no way to give a draw a PREDICTABLE output
// colour. Both halves now exist -- the SOFTWARE artifact reads the buffer back,
// and a BasicEffect with lighting, texturing, fog and vertex colour all off is
// a known solid material.
//
// So this slice draws with a known material and checks the texels that come
// back. Everything it asserts is a colour a consumer would see.
//
// # HEADLESS cannot run it, and says so
//
// The readback is a renderer capability CNA documents as such, and HEADLESS
// refuses it -- BACK_BUFFER_READ_REFUSALS is 20 there and BACK_BUFFER_READS is
// 20 on SOFTWARE. The slice therefore records a refusal and returns rather than
// failing, exactly as the existing back-buffer check does. The counters make
// the difference visible instead of letting a skipped claim look like a passing
// one.
func (g *stressGame) exercisePixelDraw(device *graphics.GraphicsDevice, parameters *graphics.PresentationParameters) error {
	// A marker nothing else in this process uses, so a texel that still holds
	// it was NOT drawn to.
	marker := framework.NewColorByInt32AndInt32AndInt32AndInt32(17, 34, 51, 255)
	width, height := int(parameters.BackBufferWidth()), int(parameters.BackBufferHeight())
	if width <= 0 || height <= 0 {
		return fmt.Errorf("the back buffer is %dx%d", width, height)
	}

	effect, err := graphics.NewBasicEffectByGraphicsDevice(device)
	if err != nil {
		if !isNativeRefusal(err) {
			return fmt.Errorf("the pixel slice's BasicEffect: %w", err)
		}
		g.result.PixelDrawRefusals++
		fmt.Fprintf(os.Stderr, "pixel draw skipped, no BasicEffect: %v\n", err)
		return nil
	}
	// BasicEffect declares no Dispose of its own, so its inherited PUBLIC
	// surface carries one that takes no argument.
	defer func() { _ = effect.Dispose() }()

	// A solid material: no lighting, no texture, no fog, no vertex colour, full
	// alpha. Every one of those is already the constructor's default, and every
	// one is set anyway -- a fixture that relies on a default it does not state
	// stops being evidence the moment the default is what breaks.
	//
	// DiffuseColor is deliberately NOT set here. The first two draws use the
	// constructor's own Vector3.One, which is what lets the winding check pin
	// the default material as a pixel fact.
	effect.SetLightingEnabled(false)
	effect.SetTextureEnabled(false)
	effect.SetFogEnabled(false)
	effect.SetVertexColorEnabled(false)
	effect.SetAlpha(1)

	// The vertex colour is YELLOW on every fixture below, and no assertion ever
	// expects yellow. That is deliberate: with VertexColorEnabled false the
	// material must win, so a projection that leaked the vertex colour through
	// would be caught by the colour assertions rather than by a separate one.
	yellow := framework.NewColorByInt32AndInt32AndInt32(255, 255, 0)
	corner := func(x, y float32) graphics.VertexPositionColor {
		return graphics.NewVertexPositionColor(
			framework.NewVector3BySingleAndSingleAndSingle(x, y, 0), yellow)
	}
	// A fresh BasicEffect carries identity World, View and Projection, so these
	// positions ARE clip space. The two triangles are the same three corners in
	// opposite winding order.
	counterClockwise := []graphics.VertexPositionColor{corner(-1, -1), corner(3, -1), corner(-1, 3)}
	clockwise := []graphics.VertexPositionColor{counterClockwise[0], counterClockwise[2], counterClockwise[1]}
	// A HALF-screen triangle. A full-screen one cannot be told apart from a
	// clear, so the geometry claim needs one that leaves texels alone.
	half := []graphics.VertexPositionColor{corner(-1, -1), corner(-1, 1), corner(1, -1)}

	// draw clears to the marker, applies the effect and submits one triangle,
	// then reports every distinct colour in the buffer. It returns nil when the
	// renderer refuses at any step, which is how HEADLESS leaves the slice.
	draw := func(vertices []graphics.VertexPositionColor) (map[[4]byte]int, []framework.Color, error) {
		if err := device.ClearByColor(marker); err != nil {
			if !isNativeRefusal(err) {
				return nil, nil, fmt.Errorf("the pixel slice's clear: %w", err)
			}
			return nil, nil, nil
		}
		technique := effect.CurrentTechnique()
		if technique == nil {
			return nil, nil, errors.New("a constructed BasicEffect has no current technique")
		}
		pass := technique.Passes().ItemPropertySignatureCA1DC5FC(0)
		if pass == nil {
			return nil, nil, errors.New("a constructed BasicEffect's technique has no first pass")
		}
		// Apply runs OnApply, which pushes the dirty subset -- so the material
		// each fixture sets below reaches CNA here and nowhere else.
		if err := pass.Apply(); err != nil {
			if !isNativeRefusal(err) {
				return nil, nil, fmt.Errorf("the pixel slice's Apply: %w", err)
			}
			return nil, nil, nil
		}
		if err := graphics.GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
			device, graphics.PrimitiveTypeTriangleList, vertices, 0, 1); err != nil {
			if !isNativeRefusal(err) {
				return nil, nil, fmt.Errorf("the pixel slice's draw: %w", err)
			}
			return nil, nil, nil
		}
		pixels := make([]framework.Color, width*height)
		if err := graphics.GraphicsDeviceGetBackBufferDataBySliceOfT(device, pixels); err != nil {
			if !isNativeRefusal(err) {
				return nil, nil, fmt.Errorf("the pixel slice's readback: %w", err)
			}
			return nil, nil, nil
		}
		histogram := map[[4]byte]int{}
		for _, pixel := range pixels {
			histogram[[4]byte{pixel.R(), pixel.G(), pixel.B(), pixel.A()}]++
		}
		return histogram, pixels, nil
	}

	markerKey := [4]byte{marker.R(), marker.G(), marker.B(), marker.A()}
	// The material the first two draws use is the constructor's OWN default,
	// which the reference sets to Vector3.One:
	//
	//	ldc.r4 1; ldc.r4 1; ldc.r4 1; newobj Vector3::.ctor -> diffuseColor
	//
	// so the texels must come back white. That makes the winding check pin the
	// default material as a PIXEL fact at no extra cost -- a constructor that
	// left the colour at zero would draw black and fail here.
	white := [4]byte{255, 255, 255, 255}
	solid := func(histogram map[[4]byte]int, want [4]byte) error {
		if count := histogram[want]; count != width*height {
			return fmt.Errorf("%d of %d texels are (%d,%d,%d,%d); the buffer holds %v",
				count, width*height, want[0], want[1], want[2], want[3], histogram)
		}
		return nil
	}

	// (1) The DEFAULT rasterizer state CULLS a counter-clockwise triangle.
	//
	// This is the one claim that needs both halves to mean anything: an empty
	// buffer proves nothing on its own, because a renderer that draws nothing
	// at all would give the same answer. The SAME three corners in the opposite
	// winding order, through the same effect and the same state, must fill the
	// buffer.
	if err := device.SetRasterizerState(graphics.RasterizerStateCullCounterClockwise()); err != nil {
		return fmt.Errorf("restoring the default rasterizer state: %w", err)
	}
	culled, _, err := draw(counterClockwise)
	if err != nil {
		return err
	}
	if culled == nil {
		g.result.PixelDrawRefusals++
		fmt.Fprintf(os.Stderr, "pixel draw skipped: this renderer has no back-buffer readback\n")
		return nil
	}
	if err := solid(culled, markerKey); err != nil {
		return fmt.Errorf("a counter-clockwise triangle was NOT culled by the default state: %w", err)
	}
	drawn, _, err := draw(clockwise)
	if err != nil {
		return err
	}
	if err := solid(drawn, white); err != nil {
		return fmt.Errorf("the clockwise triangle did not fill the buffer with the default material: %w", err)
	}
	g.result.PixelDrawWindingChecks++

	// (2) The GEOMETRY reaches the rasteriser. With culling off, a half-screen
	// triangle must leave about half the buffer untouched -- and the corners it
	// covers are measured, not assumed, because a renderer's clip-space Y
	// direction is its own business.
	if err := device.SetRasterizerState(graphics.RasterizerStateCullNone()); err != nil {
		return fmt.Errorf("CullNone: %w", err)
	}
	histogram, pixels, err := draw(half)
	if err != nil {
		return err
	}
	if histogram == nil {
		return errors.New("the readback stopped answering part-way through the pixel slice")
	}
	if len(histogram) != 2 {
		return fmt.Errorf("a half-screen triangle produced %d distinct colours, want the material and the marker: %v",
			len(histogram), histogram)
	}
	total := width * height
	// A tenth either way, which the measured split (192080 of 384000) sits well
	// inside. The tolerance exists because the diagonal's rasterisation is the
	// renderer's, not because the fraction is unknown.
	if histogram[white] < total*2/5 || histogram[white] > total*3/5 {
		return fmt.Errorf("a half-screen triangle covered %d of %d texels", histogram[white], total)
	}
	at := func(x, y int) [4]byte {
		pixel := pixels[y*width+x]
		return [4]byte{pixel.R(), pixel.G(), pixel.B(), pixel.A()}
	}
	// Measured on the SOFTWARE artifact: the two LEFT corners are inside the
	// triangle and the two RIGHT corners are outside it. A draw that ignored
	// its vertices and filled everything, or one that drew the complementary
	// half, fails here and passes the count check above.
	for _, row := range []struct {
		x, y int
		want [4]byte
		name string
	}{
		{2, 2, white, "top-left"},
		{2, height - 3, white, "bottom-left"},
		{width - 3, 2, markerKey, "top-right"},
		{width - 3, height - 3, markerKey, "bottom-right"},
	} {
		if got := at(row.x, row.y); got != row.want {
			return fmt.Errorf("the %s corner is (%d,%d,%d,%d), want (%d,%d,%d,%d)",
				row.name, got[0], got[1], got[2], got[3], row.want[0], row.want[1], row.want[2], row.want[3])
		}
	}
	g.result.PixelDrawGeometryChecks++

	// (3) The MATERIAL decides the texel. The same geometry with a different
	// DiffuseColor must come back a different colour, and the colour must be
	// the one that was set -- which is what makes every diffuse push in the
	// stock-effect family observable for the first time.
	effect.SetDiffuseColor(framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0))
	histogram, _, err = draw(half)
	if err != nil {
		return err
	}
	green := [4]byte{0, 255, 0, 255}
	// Both halves: the new colour is there AND the old one is gone. Only the
	// second catches a renderer that ignored the push and kept drawing white.
	if histogram[green] < total*2/5 || histogram[white] != 0 {
		return fmt.Errorf("DiffuseColor (0,1,0) produced %v; the material must be green and the default white gone", histogram)
	}
	g.result.PixelDrawMaterialChecks++

	// (4) ALPHA premultiplies the colour AND lands in the alpha channel.
	// Measured exactly: (0,1,0) at Alpha 0.5 comes back (0,127,0,127), which is
	// 255*0.5 truncated. This is the strongest single assertion in the slice --
	// nothing but a real shader evaluation produces that pair of numbers.
	effect.SetAlpha(0.5)
	histogram, _, err = draw(half)
	if err != nil {
		return err
	}
	halfGreen := [4]byte{0, 127, 0, 127}
	if histogram[halfGreen] < total*2/5 {
		return fmt.Errorf("Alpha 0.5 over DiffuseColor (0,1,0) produced %v, want (0,127,0,127)", histogram)
	}
	g.result.PixelDrawAlphaChecks++
	effect.SetAlpha(1)

	// (5) VertexColorEnabled, which this renderer does NOT honour.
	//
	// The fixture's vertices are yellow and the material is green. XNA's
	// BasicEffect selects a shader that reads the vertex colour when the flag is
	// on, so a faithful renderer would come back yellow. CNA's software
	// renderer comes back GREEN, and the flag reaches it: OnApply pushes
	// cna_basic_effect_set_vertex_color_enabled on the shader-index dirty bit,
	// and the push succeeds.
	//
	// So this is RECORDED in two counters rather than asserted. The projection
	// is not masking anything -- the flag crosses and the renderer ignores it --
	// and a run in which the behaviour changed would move a count rather than
	// pass in silence.
	effect.SetVertexColorEnabled(true)
	histogram, _, err = draw(half)
	if err != nil {
		return err
	}
	switch {
	case histogram[[4]byte{255, 255, 0, 255}] >= total*2/5:
		g.result.PixelDrawVertexColorHonoured++
	case histogram[green] >= total*2/5:
		g.result.PixelDrawVertexColorIgnored++
	default:
		return fmt.Errorf("VertexColorEnabled produced neither the vertex colour nor the material: %v", histogram)
	}
	effect.SetVertexColorEnabled(false)

	// (6) LIGHTING, which this renderer does NOT honour either.
	//
	// EnableDefaultLighting installs the reference's three measured rigs and
	// turns lighting on, so a renderer with a lighting model would come back a
	// different colour. CNA's software renderer comes back the SAME
	// (0,255,0,255): together with the vertex-colour result above, what it
	// evaluates is a flat material -- DiffuseColor and Alpha -- and nothing
	// per-vertex or per-light.
	//
	// Recorded in two counters for the reason the vertex-colour outcome is. The
	// asserted half is that the draw still covers its half of the buffer, so a
	// lighting rig that made the geometry vanish would still be caught.
	effect.SetLightingEnabled(true)
	effect.EnableDefaultLighting()
	histogram, _, err = draw(half)
	if err != nil {
		return err
	}
	lit := [4]byte{}
	litCount := 0
	for colour, count := range histogram {
		if colour != markerKey && count > litCount {
			lit, litCount = colour, count
		}
	}
	switch {
	case litCount < total*2/5:
		return fmt.Errorf("a lit draw covered %d texels, want about half: %v", litCount, histogram)
	case lit != green:
		g.result.PixelDrawLightingChecks++
	default:
		g.result.PixelDrawLightingIgnored++
	}
	fmt.Fprintf(os.Stderr, "PIXEL: lit material is (%d,%d,%d,%d) against the unlit (0,255,0,255)\n",
		lit[0], lit[1], lit[2], lit[3])

	// The device states this slice changed, put back the way it found them, so
	// the scenarios after it see the defaults they were written against.
	if err := device.SetRasterizerState(graphics.RasterizerStateCullCounterClockwise()); err != nil {
		return fmt.Errorf("restoring the rasterizer state: %w", err)
	}
	return nil
}

// The four stock-vertex fixtures, and the one four-vertex array the non-zero
// vertexOffset draw windows into. Each is built through the type's projected
// constructor, so the vertex data CNA receives is what a consumer would send.

func stockPositionColorTriangle() []graphics.VertexPositionColor {
	return []graphics.VertexPositionColor{
		graphics.NewVertexPositionColor(framework.NewVector3BySingleAndSingleAndSingle(0, 0, 0),
			framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)),
		graphics.NewVertexPositionColor(framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0),
			framework.NewColorByInt32AndInt32AndInt32(0, 255, 0)),
		graphics.NewVertexPositionColor(framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0),
			framework.NewColorByInt32AndInt32AndInt32(0, 0, 255)),
	}
}

func stockPositionColorQuad() []graphics.VertexPositionColor {
	triangle := stockPositionColorTriangle()
	return append(triangle, graphics.NewVertexPositionColor(
		framework.NewVector3BySingleAndSingleAndSingle(1, 1, 0),
		framework.NewColorByInt32AndInt32AndInt32(255, 255, 0)))
}

func stockPositionTextureTriangle() []graphics.VertexPositionTexture {
	return []graphics.VertexPositionTexture{
		graphics.NewVertexPositionTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 0, 0),
			framework.NewVector2BySingleAndSingle(0, 0)),
		graphics.NewVertexPositionTexture(framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0),
			framework.NewVector2BySingleAndSingle(1, 0)),
		graphics.NewVertexPositionTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0),
			framework.NewVector2BySingleAndSingle(0, 1)),
	}
}

func stockPositionColorTextureTriangle() []graphics.VertexPositionColorTexture {
	return []graphics.VertexPositionColorTexture{
		graphics.NewVertexPositionColorTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 0, 0),
			framework.NewColorByInt32AndInt32AndInt32(255, 0, 0), framework.NewVector2BySingleAndSingle(0, 0)),
		graphics.NewVertexPositionColorTexture(framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0),
			framework.NewColorByInt32AndInt32AndInt32(0, 255, 0), framework.NewVector2BySingleAndSingle(1, 0)),
		graphics.NewVertexPositionColorTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0),
			framework.NewColorByInt32AndInt32AndInt32(0, 0, 255), framework.NewVector2BySingleAndSingle(0, 1)),
	}
}

func stockPositionNormalTextureTriangle() []graphics.VertexPositionNormalTexture {
	up := framework.NewVector3BySingleAndSingleAndSingle(0, 0, 1)
	return []graphics.VertexPositionNormalTexture{
		graphics.NewVertexPositionNormalTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 0, 0),
			up, framework.NewVector2BySingleAndSingle(0, 0)),
		graphics.NewVertexPositionNormalTexture(framework.NewVector3BySingleAndSingleAndSingle(1, 0, 0),
			up, framework.NewVector2BySingleAndSingle(1, 0)),
		graphics.NewVertexPositionNormalTexture(framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0),
			up, framework.NewVector2BySingleAndSingle(0, 1)),
	}
}

// userPrimitiveSubmissions is how many user-primitive draws one vertex-buffer
// cycle makes: the six overloads over this file's own vertex type, plus one per
// stock vertex type and one with a non-zero vertexOffset. It is asserted rather
// than counted so an accidentally deleted submission is a failure.
const userPrimitiveSubmissions = 11

// checkBackBufferGuardIsReachable proves the one guard GetBackBufferData carries
// that the reference carries too, without needing a renderer that can read back.
//
// It binds a cube face, calls the read, and requires the refusal to be the
// managed one rather than anything CNA said -- then unbinds. The bind itself may
// be refused by the renderer, in which case there is nothing to prove and the
// check reports success; the aggregate accounting counts the CHECK, not the
// refusal, so a renderer with no cube storage still passes.
func checkBackBufferGuardIsReachable(device *graphics.GraphicsDevice) error {
	bound := device.GetRenderTargets()
	if len(bound) == 0 {
		return nil
	}
	err := graphics.GraphicsDeviceGetBackBufferDataBySliceOfT(device, make([]framework.Color, 4))
	if err == nil {
		return errors.New("a back-buffer read succeeded while a render target was bound")
	}
	if isNativeRefusal(err) {
		return fmt.Errorf("the active-target guard let the call reach CNA: %w", err)
	}
	if !strings.Contains(err.Error(), "Cannot use GetBackBufferData when a render target is active") {
		return fmt.Errorf("%v, want FrameworkResources.CannotGetBackBufferActiveRenderTargets", err)
	}
	return nil
}

// checkClearedPixels requires every texel that came back to be the colour the
// buffer was just cleared to.
//
// The comparison is EXACT rather than approximate: a clear writes one value to
// every texel with no filtering or blending, so a renderer that reports a
// readback at all must report that value. An approximate check here would hide
// exactly the failure this exists to catch -- a readback that returns a
// plausible-looking buffer that is not the one that was drawn.
func checkClearedPixels(pixels []framework.Color, want framework.Color) error {
	if len(pixels) == 0 {
		return errors.New("the back-buffer read reported success over no pixels")
	}
	for index, pixel := range pixels {
		if pixel.R() != want.R() || pixel.G() != want.G() ||
			pixel.B() != want.B() || pixel.A() != want.A() {
			return fmt.Errorf("back-buffer texel %d is (%d,%d,%d,%d), want the cleared (%d,%d,%d,%d)",
				index, pixel.R(), pixel.G(), pixel.B(), pixel.A(),
				want.R(), want.G(), want.B(), want.A())
		}
	}
	return nil
}

// exerciseCubeRenderTarget creates a real RenderTargetCube, binds one face,
// reads the binding back through GetRenderTargets, unbinds and disposes.
//
// The whole slice is skipped when CNA refuses the creation, which it documents
// as a renderer capability. The BINDING checks are managed and run whenever a
// target exists, because they are what the type exists to carry.
func (g *stressGame) exerciseCubeRenderTarget(device *graphics.GraphicsDevice) error {
	target, createErr := graphics.NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormat(
		device, 64, false, graphics.SurfaceFormatColor, graphics.DepthFormatNone)
	switch {
	case createErr != nil && isNativeRefusal(createErr):
		g.result.RenderTargetCubeRefusals++
		fmt.Fprintf(os.Stderr, "RenderTargetCube creation refused: %v\n", createErr)
		return nil
	case createErr != nil:
		return fmt.Errorf("NewRenderTargetCube: %w", createErr)
	}
	g.result.RenderTargetCubeCreations++
	if target.Size() != 64 {
		return fmt.Errorf("RenderTargetCube.Size = %d, want the created 64", target.Size())
	}
	// The fourth link of the composition chain, on a REAL object: the cube
	// forwards Texture's members through TextureCube, and TextureCube through
	// Texture, so a wrong forward is a wrong answer here rather than a compile
	// error.
	if target.LevelCount() < 1 || target.Format() != graphics.SurfaceFormatColor {
		return fmt.Errorf("RenderTargetCube reported %d levels at format %v",
			target.LevelCount(), target.Format())
	}
	if target.GraphicsDevice() != device {
		return errors.New("RenderTargetCube.GraphicsDevice did not answer the device that made it")
	}

	// A binding over it, and the two accessors.
	binding, bindingErr := graphics.NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(
		target, graphics.CubeMapFaceNegativeZ)
	if bindingErr != nil {
		return fmt.Errorf("NewRenderTargetBinding: %w", bindingErr)
	}
	if binding.CubeMapFace() != graphics.CubeMapFaceNegativeZ || binding.RenderTarget() == nil {
		return errors.New("the binding did not carry its face and target")
	}
	g.result.RenderTargetBindingChecks++

	// The bind. A renderer with no cube attachment refuses, which is recorded.
	switch bindErr := device.SetRenderTargetByRenderTargetCubeAndCubeMapFace(
		target, graphics.CubeMapFaceNegativeZ); {
	case bindErr == nil:
		g.result.RenderTargetCubeBinds++
		// Bound, so GetRenderTargets must answer exactly this binding -- and
		// the array must be a COPY, which is what the reference hands back.
		reported := device.GetRenderTargets()
		if len(reported) != 1 {
			return fmt.Errorf("GetRenderTargets answered %d bindings after one bind", len(reported))
		}
		if reported[0].CubeMapFace() != graphics.CubeMapFaceNegativeZ {
			return fmt.Errorf("the bound face is %v, want NegativeZ", reported[0].CubeMapFace())
		}
		again := device.GetRenderTargets()
		if &reported[0] == &again[0] {
			return errors.New("GetRenderTargets answered the same array twice")
		}
		// The guard is reachable only while something is bound, so it is
		// proved HERE rather than before the bind.
		if guardErr := checkBackBufferGuardIsReachable(device); guardErr != nil {
			return guardErr
		}
		// Back to the back buffer, which is what an empty array means.
		if unbindErr := device.SetRenderTargets(nil); unbindErr != nil && !isNativeRefusal(unbindErr) {
			return fmt.Errorf("SetRenderTargets(nil): %w", unbindErr)
		}
	case isNativeRefusal(bindErr):
		g.result.RenderTargetCubeBindRefusals++
		fmt.Fprintf(os.Stderr, "cube render-target bind refused: %v\n", bindErr)
	default:
		return fmt.Errorf("SetRenderTarget(cube, face): %w", bindErr)
	}

	if disposeErr := target.DisposeByNone(); disposeErr != nil {
		return fmt.Errorf("RenderTargetCube.Dispose: %w", disposeErr)
	}
	if !target.IsDisposed() {
		return errors.New("a disposed RenderTargetCube does not report itself disposed")
	}
	// A second Dispose is a no-op, which is the reference's contract and the
	// one thing a double-free would break.
	if disposeErr := target.DisposeByNone(); disposeErr != nil {
		return fmt.Errorf("RenderTargetCube.Dispose twice: %w", disposeErr)
	}
	return nil
}

// exerciseOwnedDevice creates a GraphicsDevice the CONSUMER owns through the
// type's one public constructor, and destroys it.
//
// This is the only place in the profile where a GraphicsDevice is OWNED rather
// than borrowed from the Game, and it is the ownership CNA tells apart itself:
// cna_graphics_device_destroy accepts a caller-created handle and refuses a
// Game's. Creating one INSIDE a running game and destroying it before the game
// ends is exactly the interleaving that would surface a confusion between the
// two, which is why it runs here rather than in a unit test.
func (g *stressGame) exerciseOwnedDevice(device *graphics.GraphicsDevice) error {
	adapter, adapterErr := device.Adapter()
	if adapterErr != nil {
		return fmt.Errorf("GraphicsDevice.Adapter: %w", adapterErr)
	}
	parameters, parametersErr := device.PresentationParameters()
	if parametersErr != nil {
		return fmt.Errorf("PresentationParameters: %w", parametersErr)
	}
	profile, profileErr := device.GraphicsProfile()
	if profileErr != nil {
		return fmt.Errorf("GraphicsDevice.GraphicsProfile: %w", profileErr)
	}
	owned, createErr := graphics.NewGraphicsDevice(adapter, profile, parameters)
	switch {
	case createErr != nil && isNativeRefusal(createErr):
		g.result.OwnedDeviceRefusals++
		fmt.Fprintf(os.Stderr, "owned GraphicsDevice creation refused: %v\n", createErr)
		return nil
	case createErr != nil:
		return fmt.Errorf("NewGraphicsDevice: %w", createErr)
	}
	g.result.OwnedDeviceCreations++
	// It must be a DIFFERENT device from the game's, or the constructor handed
	// back a facade over the borrowed one and the ownership is a fiction.
	if owned == device {
		return errors.New("the constructor answered the game's own device")
	}
	// IsDisposed is FALLIBLE on this type -- it asks CNA -- so the answer and
	// the error are both meaningful.
	if disposed, err := owned.IsDisposed(); err != nil {
		return fmt.Errorf("the owned device's IsDisposed: %w", err)
	} else if disposed {
		return errors.New("a freshly constructed device reports itself disposed")
	}
	// It answers its own parameters, which proves the handle is live rather
	// than merely non-nil.
	ownedParameters, ownedErr := owned.PresentationParameters()
	if ownedErr != nil {
		return fmt.Errorf("the owned device's PresentationParameters: %w", ownedErr)
	}
	if ownedParameters.BackBufferWidth() <= 0 {
		return errors.New("the owned device reported a zero-width back buffer")
	}
	if disposeErr := owned.DisposeByNone(); disposeErr != nil {
		return fmt.Errorf("the owned device's Dispose: %w", disposeErr)
	}
	// A destroyed owned device is unusable, and it must report THAT rather
	// than answering from the running game's device -- which is what it did
	// before the Foundation 73 interop fix, because a zeroed owned handle took
	// the borrowed path. Either answer proves it: an error, or a live device
	// saying it is disposed.
	if disposed, err := owned.IsDisposed(); err == nil && !disposed {
		return errors.New("a destroyed owned device answered a live, undisposed device")
	}
	if _, err := owned.PresentationParameters(); err == nil {
		return errors.New("a destroyed owned device still answers its presentation parameters")
	}
	// And the GAME's device is untouched, which is the confusion this whole
	// slice exists to rule out.
	if disposed, err := device.IsDisposed(); err != nil {
		return fmt.Errorf("the game's device stopped answering IsDisposed: %w", err)
	} else if disposed {
		return errors.New("destroying an owned device disposed the game's device")
	}
	if _, err := device.PresentationParameters(); err != nil {
		return fmt.Errorf("the game's device stopped answering after an owned device was destroyed: %w", err)
	}
	g.result.OwnedDeviceDisposalChecks++
	return nil
}
