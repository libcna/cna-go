package interop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"runtime/cgo"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	callbackInitialize    = 1
	callbackLoadContent   = 2
	callbackUpdate        = 3
	callbackDraw          = 4
	callbackUnloadContent = 5
	callbackExiting       = 6

	// The four optional frame-boundary hooks. They are separate kinds rather
	// than lifecycle members because CNA_GameCallbacks does not carry them:
	// they belong to CNA_GameFrameHooks, and each is installed only when the
	// framework reports a matching override.
	callbackBeginRun  = 7
	callbackEndRun    = 8
	callbackBeginDraw = 9
	callbackEndDraw   = 10
)

// FrameHookMask selects which optional CNA_GameFrameHooks members
// cna_go_game_create installs. native_linux.go carries compile-time assertions
// that these equal the C mirror in bridge.h.
//
// The `initialize` hook is not in the mask: it is the position
// Game::Initialize occupies, it is always installed, and it is not an optional
// override. The other four are installed if and only if their bit is set, and
// a member left NULL is one CNA simply does not call -- so an absent override
// leaves the native frame position behaving exactly as it did before this
// mechanism existed.
type FrameHookMask uint32

const (
	FrameHookBeginRun FrameHookMask = 1 << iota
	FrameHookEndRun
	FrameHookBeginDraw
	FrameHookEndDraw
)

// The four canonical CNA game-event identities. They mirror CNA_GAME_EVENT_*
// exactly; native_linux.go carries compile-time assertions that they equal the
// C values, so no Go file outside this package ever spells a CNA constant.
const (
	GameEventActivated   uint32 = 0
	GameEventDeactivated uint32 = 1
	GameEventDisposed    uint32 = 2
	GameEventExiting     uint32 = 3

	gameEventCount = 4
)

// The three canonical GameWindow signal identities. They are a SECOND
// numbering that also starts at zero, so a value from one family is a
// valid-looking value in the other; they are kept in separate constant sets,
// delivered through separate trampolines, and counted separately.
const (
	GameWindowEventClientSizeChanged       uint32 = 0
	GameWindowEventOrientationChanged      uint32 = 1
	GameWindowEventScreenDeviceNameChanged uint32 = 2

	gameWindowEventCount = 3
)

// GameWindowEventCount is the canonical window-signal identity count.
const GameWindowEventCount = gameWindowEventCount

type ownership uint8

const (
	managedValue ownership = iota
	owned
	borrowed
	parentOwned
	processGlobal
)

type resourceKind uint8

const (
	resourceGraphicsDeviceManager resourceKind = iota + 1
	resourceTexture2D
	resourceSpriteBatch
	resourceRenderTarget2D
	resourceContentManager
	resourceIndexBuffer
	resourceVertexDeclaration
	resourceVertexBuffer
	resourceSpriteFont
	resourceTexture3D
	resourceTextureCube
	resourceEffect
	// The eight effect VIEW kinds. Each is its own resource kind for the reason
	// every other kind is: destruction is per-kind, and a shared kind would
	// need a second field on Resource to say which of the eight destroy routes
	// to call -- which is a second source of truth about the same handle.
	resourceEffectTechniqueCollection
	resourceEffectTechnique
	resourceEffectPassCollection
	resourceEffectPass
	resourceEffectParameterCollection
	resourceEffectParameter
	resourceEffectAnnotationCollection
	resourceEffectAnnotation
)

// effectViewResourceKinds maps a view kind onto its resource kind, in the order
// the view constants declare them.
var effectViewResourceKinds = [8]resourceKind{
	resourceEffectTechniqueCollection,
	resourceEffectTechnique,
	resourceEffectPassCollection,
	resourceEffectPass,
	resourceEffectParameterCollection,
	resourceEffectParameter,
	resourceEffectAnnotationCollection,
	resourceEffectAnnotation,
}

// FrameTime is the private tick-exact lifecycle value passed into the public
// GameTime adapter.
type FrameTime struct {
	TotalTicks      int64
	ElapsedTicks    int64
	IsRunningSlowly bool
}

// Callbacks is implemented by the framework package. It contains no C/native
// types and is not visible to consumers because this package is internal.
type Callbacks interface {
	Initialize() error
	LoadContent() error
	Update(FrameTime) error
	Draw(FrameTime) error
	UnloadContent() error

	// GameEvent delivers one canonical CNA game signal. It is deliberately
	// separate from the five lifecycle members above: those project XNA's
	// protected virtual overrides and are declared by the public
	// GameCallbacks contract, while this one is the private bridge for the
	// four CLR events Game declares. Adding it here leaves GameCallbacks
	// untouched, which is what keeps every existing external implementation
	// of that interface compiling.
	GameEvent(event uint32) error

	// GameWindowEvent delivers one canonical CNA window signal. It is a
	// separate member from GameEvent rather than a wider identity space on
	// one, because the two families both number from zero: sharing an entry
	// point would make a mis-routed signal look like a valid one.
	GameWindowEvent(event uint32) error

	// TimingConfiguration reports the Game's configured timing and
	// presentation state. It is read once, on the owner thread, immediately
	// before cna_game_create, because the native loop has to START with what
	// the managed state says rather than with a literal -- XNA's own loop
	// reads those fields every frame, and a consumer may set them before Run.
	TimingConfiguration() TimingConfiguration

	// FrameHookOverrides reports which optional frame-boundary hooks this
	// caller wants installed. It is read exactly once, on the owner thread,
	// immediately before cna_game_create, because a Go callback object's
	// method set is fixed for the object's whole lifetime and the answer
	// therefore cannot change afterwards.
	FrameHookOverrides() FrameHookMask

	// The four optional frame-boundary overrides. Each is invoked only from
	// the native hook its mask bit installed, so a caller that reported no
	// bit for one of them is never asked for it.
	BeginRun() error
	EndRun() error
	BeginDraw() (bool, error)
	EndDraw() error
}

// TimingConfiguration is the Game's configured timing and presentation state as
// the native loop needs it: ticks rather than TimeSpan, because the C ABI
// counts in 100-nanosecond ticks and the conversion belongs on the public side.
type TimingConfiguration struct {
	TargetElapsedTicks int64
	InactiveSleepTicks int64
	IsFixedTimeStep    bool
	IsMouseVisible     bool
}

// Runtime owns one admitted native Game generation.
type Runtime struct {
	mu              sync.Mutex
	callbacks       Callbacks
	game            uint64
	generation      uint64
	ownerThread     uint64
	alive           bool
	inCallback      bool
	callbackFailure error
	resources       []*Resource
	title           string

	// gameEventDeliveries counts every canonical signal actually delivered,
	// per identity, for the life of the Runtime. It exists because the
	// disposal signal no longer raises a public event: Game::Disposed is
	// raised from managed Dispose(bool), so the native signal's only remaining
	// job is native lifetime qualification, and something has to be able to
	// see it. Nothing outside this module can: the framework package never
	// reads it, and only tools that already import this internal package do.
	gameEventDeliveries [gameEventCount]int

	// eventRegistrations holds the four owned CNA registration handles, one
	// per canonical game event. They are installed once, right after the
	// native game is created, and released after it is destroyed -- never in
	// the other order, because CNA raises the disposal signal from inside
	// cna_game_destroy and a registration released first would miss it.
	eventRegistrations [gameEventCount]uint64

	// The window signals, kept in their own two slots for the same reasons
	// and with the same lifetime: installed once the native game exists,
	// released after it is destroyed. Three, not four -- the window family is
	// its own numbering.
	windowEventDeliveries    [gameWindowEventCount]int
	windowEventRegistrations [gameWindowEventCount]uint64

	// The native SESSION: one live cna_game_create/cna_game_destroy pair,
	// its cgo.Handle, its process lock and its locked OS thread.
	//
	// Foundation 47 split the session out of Run because CNA supports one
	// without a loop. cna_game_tick and cna_game_run_one_frame drive a created
	// game directly, and the measured probe confirms it: a tick on a
	// never-initialized game runs Update and Draw with no Initialize, and
	// run_one_frame initializes first and then does the same.
	//
	// standalone records who started the session, because that decides who
	// ends it: a session Run created is destroyed when Run returns, and one a
	// frame step created outlives every call and is destroyed by Dispose.
	callbackHandle cgo.Handle
	sessionLive    bool
	standalone     bool
}

// Resource is an internal generation-checked owned-handle control block.
type Resource struct {
	mu         sync.Mutex
	runtime    *Runtime
	parent     *Resource
	generation uint64
	handle     uint64
	disposing  bool
	kind       resourceKind
	ownership  ownership
}

// Device represents a callback-borrowed graphics device. It retains no native
// handle across calls; every operation reacquires the callback-scoped handle.
type Device struct {
	runtime    *Runtime
	manager    *Resource
	generation uint64
	ownership  ownership
}

// DisplayMode is CNA_DisplayMode.
//
// CNA reports an aspect ratio of its own alongside the two dimensions. The
// projection does NOT use it: XNA's DisplayMode::get_AspectRatio is 38 bytes of
// managed arithmetic over the two fields, and reproducing that is exact where
// trusting a second computation would be a value that could disagree. The field
// is carried here anyway, because a route's output is measured as it is
// declared and dropping a member of the struct would leave the layout
// unchecked.
type DisplayMode struct {
	Width, Height int32
	AspectRatio   float32
	Format        uint32
}

// ScissorRectangle is CNA_Rectangle as the graphics device's clip rectangle.
//
// It is a distinct interop type rather than a reuse of the sprite command's
// four fields, for the reason every other interop struct here is its own: it
// crosses the boundary on its own routes and nothing about it is tied to a
// sprite.
type ScissorRectangle struct {
	X, Y          int32
	Width, Height int32
}

type Viewport struct {
	X, Y          int32
	Width, Height int32
	MinDepth      float32
	MaxDepth      float32
}

type TextureInfo struct {
	Width, Height uint32
	Levels        uint32
	Format        uint32
}

// The four state descriptors, flattened. Every field is the value CNA's POD
// carries, in CNA's own order, and the bridge builds the versioned structure on
// the C side so no CNA structure crosses cgo.
type BlendStateValue struct {
	AlphaBlendFunction, AlphaDestinationBlend, AlphaSourceBlend uint32
	ColorBlendFunction, ColorDestinationBlend, ColorSourceBlend uint32
	ColorWriteChannels, ColorWriteChannels1                     uint32
	ColorWriteChannels2, ColorWriteChannels3                    uint32
	BlendFactorR, BlendFactorG, BlendFactorB, BlendFactorA      uint8
	MultiSampleMask                                             int32
}

type DepthStencilStateValue struct {
	DepthBufferEnable, DepthBufferWriteEnable        bool
	StencilEnable, TwoSidedStencilMode               bool
	DepthBufferFunction, StencilFunction             uint32
	StencilFail, StencilDepthBufferFail, StencilPass uint32
	CounterClockwiseStencilFunction                  uint32
	CounterClockwiseStencilFail                      uint32
	CounterClockwiseStencilDepthBufferFail           uint32
	CounterClockwiseStencilPass                      uint32
	StencilMask, StencilWriteMask, ReferenceStencil  int32
}

type RasterizerStateValue struct {
	CullMode, FillMode                      uint32
	DepthBias, SlopeScaleDepthBias          float32
	MultiSampleAntiAlias, ScissorTestEnable bool
}

type SamplerStateValue struct {
	AddressU, AddressV, AddressW, Filter uint32
	MaxAnisotropy, MaxMipLevel           int32
	MipMapLevelOfDetailBias              float32
}

// SetBlendState is cna_graphics_device_set_blend_state.
func (d *Device) SetBlendState(value BlendStateValue) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetBlendState(handle, value)
}

// SetDepthStencilState is cna_graphics_device_set_depth_stencil_state.
func (d *Device) SetDepthStencilState(value DepthStencilStateValue) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetDepthStencilState(handle, value)
}

// SetRasterizerState is cna_graphics_device_set_rasterizer_state.
func (d *Device) SetRasterizerState(value RasterizerStateValue) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetRasterizerState(handle, value)
}

// CreateContentManager creates an owned game-child content manager.
//
// CNA requires a callback-scoped device handle, which the reference's
// constructor does not: it takes an IServiceProvider and resolves the device
// lazily. The projection therefore creates LAZILY too, at the first operation
// that needs the native manager -- which is inside a callback by construction.
func (d *Device) CreateContentManager(rootDirectory string) (*Resource, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	manager, err := nativeContentManagerCreate(handle, rootDirectory)
	if err != nil {
		return nil, err
	}
	return d.runtime.registerResource(manager, resourceContentManager, d.manager), nil
}

// Runtime reports the device's runtime, so a caller holding a live device can
// create the standalone objects a device-owned one needs -- a vertex
// declaration is the first. It is a plain accessor and cannot fail.
func (d *Device) Runtime() *Runtime {
	if d == nil {
		return nil
	}
	return d.runtime
}

// HandleOf reports the live CNA handle a resource owns, for the ONE thing that
// needs it: binding a buffer to this device. It is on Device rather than on
// Resource so that the generation and kind are checked against a device that is
// itself live, and it stays unexported outside this package by the raw-handle
// rule -- the Graphics package passes what it gets straight back in.
func (d *Device) HandleOf(resource *Resource) (uint64, error) {
	if d == nil || resource == nil {
		return 0, ErrDisposed
	}
	if _, err := d.nativeHandle(); err != nil {
		return 0, err
	}
	return resource.anyLiveHandle()
}

// SetVertexBuffers is cna_graphics_device_set_vertex_buffers. The bindings
// arrive as a flat int64 triple each -- handle, offset, frequency -- because no
// CNA struct crosses cgo.
func (d *Device) SetVertexBuffers(bindings []int64) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeDeviceSetVertexBuffers(handle, bindings)
}

// SetIndexBuffer is cna_graphics_device_set_index_buffer. A zero handle is
// CNA_INVALID_HANDLE and unbinds.
func (d *Device) SetIndexBuffer(buffer uint64) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeDeviceSetIndexBuffer(handle, buffer)
}

// DrawPrimitives is cna_graphics_device_draw_primitives.
func (d *Device) DrawPrimitives(primitiveType uint32, vertexStart, primitiveCount int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeDeviceDrawPrimitives(handle, primitiveType, vertexStart, primitiveCount)
}

// DrawIndexedPrimitives is cna_graphics_device_draw_indexed_primitives.
func (d *Device) DrawIndexedPrimitives(primitiveType uint32, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeDeviceDrawIndexedPrimitives(handle, primitiveType, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount)
}

// DrawInstancedPrimitives is cna_graphics_device_draw_instanced_primitives.
func (d *Device) DrawInstancedPrimitives(primitiveType uint32, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount, instanceCount int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeDeviceDrawInstancedPrimitives(handle, primitiveType, baseVertex, minVertexIndex, numVertices, startIndex, primitiveCount, instanceCount)
}

// AdapterInfo is CNA_GraphicsAdapterInfo, flattened.
//
// The two BYTE-LENGTH fields are carried, and the reason is measured rather
// than assumed. CNA-Go's usual string shape is a two-call length-then-copy, but
// `cna_graphics_adapter_copy_description` answers a zero capacity with
// CNA_RESULT 14 -- "the graphics-adapter string output buffer is too small" --
// rather than with SUCCESS and a required count. These fields ARE the length
// call for this family, which is why the info structure carries them.
type AdapterInfo struct {
	Index              uint32
	IsDefaultAdapter   bool
	IsWideScreen       bool
	UseNullDevice      bool
	UseReferenceDevice bool
	VendorID           int32
	DeviceID           int32
	Revision           int32
	SubSystemID        int32
	DescriptionBytes   uint64
	DeviceNameBytes    uint64
}

// DisplayModeValue is CNA_DisplayMode without its aspect ratio. XNA computes
// that from the two dimensions in 38 bytes of managed arithmetic, and trusting
// CNA's would be a second computation that could disagree.
type DisplayModeValue struct {
	Width  int32
	Height int32
	Format uint32
}

// FormatSelection is CNA_GraphicsFormatSelection: what the adapter chose when
// asked for a format, and whether it had to substitute anything.
type FormatSelection struct {
	ExactMatch       bool
	Format           uint32
	DepthFormat      uint32
	MultiSampleCount int32
}

// The twelve adapter queries. Every one takes this device's callback-scoped
// handle, which is CNA's own requirement.
// AdapterIndex is cna_graphics_device_get_adapter_index: which adapter this
// device was created with, as an index into the same enumeration the
// cna_graphics_adapter_* routes take. CNA warns that indices are point-in-time
// values a later adapter change may renumber, which is why the projection reads
// the index and the adapter TOGETHER rather than caching one.
func (d *Device) AdapterIndex() (uint32, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeDeviceAdapterIndex(handle)
}

func (d *Device) AdapterCount() (uint64, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeAdapterCount(handle)
}

func (d *Device) AdapterInfo(index uint32) (AdapterInfo, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return AdapterInfo{}, err
	}
	return nativeAdapterInfo(handle, index)
}

func (d *Device) AdapterDescription(index uint32, byteCount uint64) (string, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return "", err
	}
	return nativeAdapterDescription(handle, index, byteCount)
}

func (d *Device) AdapterDeviceName(index uint32, byteCount uint64) (string, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return "", err
	}
	return nativeAdapterDeviceName(handle, index, byteCount)
}

func (d *Device) AdapterCurrentDisplayMode(index uint32) (DisplayModeValue, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return DisplayModeValue{}, err
	}
	return nativeAdapterCurrentDisplayMode(handle, index)
}

func (d *Device) AdapterDisplayModes(index uint32) ([]DisplayModeValue, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	return nativeAdapterDisplayModes(handle, index)
}

func (d *Device) SetAdapterDevicePreferences(index uint32, nullDevice, referenceDevice bool) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeAdapterSetDevicePreferences(handle, index, nullDevice, referenceDevice)
}

func (d *Device) AdapterIsProfileSupported(index, profile uint32) (bool, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return false, err
	}
	return nativeAdapterIsProfileSupported(handle, index, profile)
}

func (d *Device) AdapterQueryFormat(index uint32, renderTarget bool, profile, format, depthFormat uint32, multiSampleCount int32) (FormatSelection, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return FormatSelection{}, err
	}
	return nativeAdapterQueryFormat(handle, index, renderTarget, profile, format, depthFormat, multiSampleCount)
}

func (d *Device) AdapterMonitorHandle(index uint32) (uint64, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeAdapterMonitorHandle(handle, index)
}

// VertexBufferInfo is CNA_VertexBufferInfo, flattened.
type VertexBufferInfo struct {
	VertexCount        int32
	BufferUsage        uint32
	Dynamic            bool
	IsContentLost      bool
	HasRenderer        bool
	VertexStride       int32
	VertexElementCount uint64
}

// CreateVertexDeclaration is cna_vertex_declaration_create or, when the caller
// supplies one, cna_vertex_declaration_create_with_stride.
//
// It takes NO device: a declaration is a standalone CNA object, so this is
// reachable outside a lifecycle callback, exactly as the reference's
// constructor is. The elements arrive as a flat int32 array of four fields
// each -- offset, format, usage, usage index -- because no CNA struct crosses
// cgo.
func (r *Runtime) CreateVertexDeclaration(stride int32, hasStride bool, elements []int32) (*Resource, error) {
	handle, err := nativeVertexDeclarationCreate(stride, hasStride, elements)
	if err != nil {
		return nil, err
	}
	return r.registerResource(handle, resourceVertexDeclaration, nil), nil
}

// VertexDeclarationStride is cna_vertex_declaration_get_stride.
func (resource *Resource) VertexDeclarationStride() (int32, error) {
	handle, err := resource.liveHandle(resourceVertexDeclaration)
	if err != nil {
		return 0, err
	}
	return nativeVertexDeclarationStride(handle)
}

// CreateVertexBuffer is cna_vertex_buffer_create. The device handle is
// callback-scoped, so this is reachable only from inside a lifecycle callback.
func (d *Device) CreateVertexBuffer(declaration *Resource, vertexCount int32, usage uint32, dynamic bool) (*Resource, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	declarationHandle, err := declaration.liveHandle(resourceVertexDeclaration)
	if err != nil {
		return nil, err
	}
	buffer, err := nativeVertexBufferCreate(handle, declarationHandle, vertexCount, usage, dynamic)
	if err != nil {
		return nil, err
	}
	return d.runtime.registerResource(buffer, resourceVertexBuffer, d.manager), nil
}

// VertexBufferInfo is cna_vertex_buffer_get_info.
func (resource *Resource) VertexBufferInfo() (VertexBufferInfo, error) {
	handle, err := resource.liveHandle(resourceVertexBuffer)
	if err != nil {
		return VertexBufferInfo{}, err
	}
	return nativeVertexBufferInfo(handle)
}

// SetVertexDataRaw and GetVertexDataRaw are the two RAW transfers, which are
// the ones XNA's generic SetData<T>/GetData<T> correspond to: both sides
// describe a vertex by an explicit byte stride rather than by a type identity,
// which is exactly what `sizeof(T)` is in the reference.
//
// Both offsets index THE BUFFER, not the caller's array -- which is what XNA's
// `offsetInBytes` means and the one place in this ABI where an offset does.
func (resource *Resource) SetVertexDataRaw(bufferOffsetInBytes uint64, data unsafe.Pointer, byteCount, vertexCount uint64, stride uint32) error {
	handle, err := resource.liveHandle(resourceVertexBuffer)
	if err != nil {
		return err
	}
	return nativeVertexBufferSetDataRawAt(handle, bufferOffsetInBytes, data, byteCount, vertexCount, stride)
}

func (resource *Resource) GetVertexDataRaw(bufferOffsetInBytes uint64, destination unsafe.Pointer, byteCount, vertexCount uint64, stride uint32) error {
	handle, err := resource.liveHandle(resourceVertexBuffer)
	if err != nil {
		return err
	}
	return nativeVertexBufferGetDataRaw(handle, bufferOffsetInBytes, destination, byteCount, vertexCount, stride)
}

// IndexBufferInfo is CNA_IndexBufferInfo, flattened. Everything CNA reports
// about a created index buffer, including the two renderer-state flags the
// projection does not publish and the dynamic flag that decides whether a
// streaming option is legal.
type IndexBufferInfo struct {
	IndexCount       int32
	IndexElementSize uint32
	BufferUsage      uint32
	Dynamic          bool
	IsContentLost    bool
	HasRenderer      bool
}

// The two CNA_INDEX_ELEMENT_SIZE_* identities, in CNA's own order. They happen
// to match XNA's IndexElementSize literals, and the Graphics package maps them
// explicitly anyway rather than casting: a shared numbering is a coincidence to
// be checked, not a rule to rely on.
const (
	IndexElementSizeSixteenBits   uint32 = 0
	IndexElementSizeThirtyTwoBits uint32 = 1
)

// The three CNA_SET_DATA_* identities.
const (
	SetDataNone        uint32 = 0
	SetDataDiscard     uint32 = 1
	SetDataNoOverwrite uint32 = 2
)

// CreateIndexBuffer is cna_index_buffer_create. The device handle is
// callback-scoped, so this is reachable only from inside a lifecycle callback.
func (d *Device) CreateIndexBuffer(indexCount int32, elementSize, usage uint32, dynamic bool) (*Resource, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	buffer, err := nativeIndexBufferCreate(handle, indexCount, elementSize, usage, dynamic)
	if err != nil {
		return nil, err
	}
	return d.runtime.registerResource(buffer, resourceIndexBuffer, d.manager), nil
}

// IndexBufferInfo is cna_index_buffer_get_info.
func (resource *Resource) IndexBufferInfo() (IndexBufferInfo, error) {
	handle, err := resource.liveHandle(resourceIndexBuffer)
	if err != nil {
		return IndexBufferInfo{}, err
	}
	return nativeIndexBufferInfo(handle)
}

// SetIndexData, SetIndexDataAt and GetIndexData are the three typed transfers.
//
// Each takes an unsafe.Pointer to the caller's array and its element count,
// because the element TYPE is decided by the caller and CNA identifies it by an
// index-element-size identity rather than by a Go type. The Graphics package is
// where a Go type is turned into that identity, and where the element width is
// checked against what the identity means -- interop copies bytes and validates
// nothing about their shape.
func (resource *Resource) SetIndexData(elementSize, options uint32, startIndex, elementCount uint64, data unsafe.Pointer, capacity uint64) error {
	handle, err := resource.liveHandle(resourceIndexBuffer)
	if err != nil {
		return err
	}
	return nativeIndexBufferSetData(handle, elementSize, options, startIndex, elementCount, data, capacity)
}

func (resource *Resource) SetIndexDataAt(bufferOffsetInBytes uint64, elementSize, options uint32, startIndex, elementCount uint64, data unsafe.Pointer, capacity uint64) error {
	handle, err := resource.liveHandle(resourceIndexBuffer)
	if err != nil {
		return err
	}
	return nativeIndexBufferSetDataAt(handle, bufferOffsetInBytes, elementSize, options, startIndex, elementCount, data, capacity)
}

func (resource *Resource) GetIndexData(elementSize uint32, startIndex, elementCount uint64, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	handle, err := resource.liveHandle(resourceIndexBuffer)
	if err != nil {
		return 0, err
	}
	return nativeIndexBufferGetData(handle, elementSize, startIndex, elementCount, destination, capacity)
}

// ContentRootDirectory is cna_content_manager_copy_root_directory.
func (resource *Resource) ContentRootDirectory() (string, error) {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return "", err
	}
	return nativeContentManagerRootDirectory(handle)
}

// SetContentRootDirectory is cna_content_manager_set_root_directory. CNA does
// NOT unload the existing cache, and neither does the reference's setter.
func (resource *Resource) SetContentRootDirectory(value string) error {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return err
	}
	return nativeContentManagerSetRootDirectory(handle, value)
}

// UnloadContent is cna_content_manager_unload. Handles already handed out stay
// valid and must still be destroyed, which is CNA's rule and the reference's.
func (resource *Resource) UnloadContent() error {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return err
	}
	return nativeContentManagerUnload(handle)
}

// LoadContentTexture2D is cna_content_manager_load_texture2d. The texture it
// returns is INDEPENDENTLY owned: it survives the manager's unload and its
// destruction, and must be destroyed before the parent game.
func (resource *Resource) LoadContentTexture2D(assetName string) (*Resource, TextureInfo, error) {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return nil, TextureInfo{}, err
	}
	texture, err := nativeContentManagerLoadTexture2D(handle, assetName)
	if err != nil {
		return nil, TextureInfo{}, err
	}
	owned := resource.runtime.registerResource(texture, resourceTexture2D, resource.parent)
	info, infoErr := nativeTextureInfo(texture)
	if infoErr != nil {
		_ = owned.Dispose()
		return nil, TextureInfo{}, infoErr
	}
	return owned, info, nil
}

// ContentAssetPath is the root directory joined with the asset name. CNA
// reports it whether or not a file exists there, which is what makes it usable
// for the reference's OpenStream.
func (resource *Resource) ContentAssetPath(assetName string) (string, error) {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return "", err
	}
	return nativeContentManagerAssetPath(handle, assetName)
}

// UserVertexSourceRawStream is CNA_USER_VERTEX_SOURCE_RAW_STREAM, the one
// identity CNA-Go uses: the other four name CNA vertex value types the Graphics
// package has no Go counterpart for, and a raw stream with an explicit
// declaration expresses every layout they do.
const UserVertexSourceRawStream uint32 = 0

// DrawUserPrimitives is cna_graphics_device_draw_user_primitives.
func (d *Device) DrawUserPrimitives(
	primitiveType uint32, vertexData unsafe.Pointer, declaration *Resource,
	vertexOffset, primitiveCount int32,
) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	declarationHandle, err := declaration.liveHandle(resourceVertexDeclaration)
	if err != nil {
		return err
	}
	return nativeDrawUserPrimitives(handle, primitiveType, UserVertexSourceRawStream,
		vertexData, declarationHandle, vertexOffset, 0, primitiveCount)
}

// DrawUserIndexedPrimitives is
// cna_graphics_device_draw_user_indexed_primitives.
func (d *Device) DrawUserIndexedPrimitives(
	primitiveType uint32, vertexData unsafe.Pointer, declaration *Resource,
	vertexOffset, numVertices, primitiveCount int32,
	indexElementSize uint32, indexOffset int32, indexData unsafe.Pointer,
) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	declarationHandle, err := declaration.liveHandle(resourceVertexDeclaration)
	if err != nil {
		return err
	}
	return nativeDrawUserIndexedPrimitives(handle, primitiveType, UserVertexSourceRawStream,
		vertexData, declarationHandle, vertexOffset, numVertices, primitiveCount,
		indexElementSize, indexOffset, indexData)
}

// ---------------------------------------------------------------------------
// Foundation 72 — the Effect cluster.
// ---------------------------------------------------------------------------

// The four collection kinds, and the eight view kinds behind them. They are
// separate numberings because a collection's COUNT and its element ACCESS are
// four routes each while a destroy is eight, and one shared numbering would
// make an off-by-one route a compile-time success.
const (
	effectCollectionTechnique uint32 = iota
	effectCollectionPass
	effectCollectionParameter
	effectCollectionAnnotation
)

const (
	effectViewTechniqueCollection uint32 = iota
	effectViewTechnique
	effectViewPassCollection
	effectViewPass
	effectViewParameterCollection
	effectViewParameter
	effectViewAnnotationCollection
	effectViewAnnotation
)

// The nine CNA_EFFECT_VALUE_* identities, and the four CNA_EFFECT_TEXTURE_*
// ones. Both are CNA's own numbering and the Graphics package maps XNA onto
// them explicitly rather than casting.
const (
	EffectValueBoolean         uint32 = 0
	EffectValueInt32           uint32 = 1
	EffectValueSingle          uint32 = 2
	EffectValueMatrix          uint32 = 3
	EffectValueMatrixTranspose uint32 = 4
	EffectValueQuaternion      uint32 = 5
	EffectValueVector2         uint32 = 6
	EffectValueVector3         uint32 = 7
	EffectValueVector4         uint32 = 8
)

const (
	EffectTextureBase uint32 = 0
	EffectTexture2D   uint32 = 1
	EffectTexture3D   uint32 = 2
	EffectTextureCube uint32 = 3
)

// The four collection kinds and the eight view kinds, exported for the Graphics
// package, which is the only caller: an unexported identity would make every
// call there pass a bare literal.
const (
	EffectCollectionTechnique  = effectCollectionTechnique
	EffectCollectionPass       = effectCollectionPass
	EffectCollectionParameter  = effectCollectionParameter
	EffectCollectionAnnotation = effectCollectionAnnotation
)

const (
	EffectViewTechniqueCollection  = effectViewTechniqueCollection
	EffectViewTechnique            = effectViewTechnique
	EffectViewPassCollection       = effectViewPassCollection
	EffectViewPass                 = effectViewPass
	EffectViewParameterCollection  = effectViewParameterCollection
	EffectViewParameter            = effectViewParameter
	EffectViewAnnotationCollection = effectViewAnnotationCollection
	EffectViewAnnotation           = effectViewAnnotation
)

// The eight effect string reads, by kind.
const (
	EffectStringTechniqueName      = effectStringTechniqueName
	EffectStringPassName           = effectStringPassName
	EffectStringParameterName      = effectStringParameterName
	EffectStringParameterSemantic  = effectStringParameterSemantic
	EffectStringParameterValue     = effectStringParameterValue
	EffectStringAnnotationName     = effectStringAnnotationName
	EffectStringAnnotationSemantic = effectStringAnnotationSemantic
	EffectStringAnnotationValue    = effectStringAnnotationValue
)

// EffectMetadata is the four values CNA_EffectParameterInfo and
// CNA_EffectAnnotationInfo both carry, which are the same four fields in the
// same order -- so one Go type serves both, and the two ABI structures are
// still measured separately.
type EffectMetadata struct {
	RowCount       int32
	ColumnCount    int32
	ParameterClass uint32
	ParameterType  uint32
}

// EffectView is an owned CNA view handle inside one effect's object graph: a
// collection, a technique, a pass, a parameter or an annotation.
//
// # Why views are a distinct control block
//
// CNA hands out a FRESH owned handle from every accessor -- the Foundation 72
// probe measured two cna_effect_get_parameters calls answering two different
// handles -- and the reference answers the same managed object forever. So the
// Graphics package asks ONCE per logical object and holds the answer, and this
// type is what it holds.
//
// A view OUTLIVES its effect: the probe destroyed an effect with views alive
// and the views still answered. So there is no destruction-order rule between
// an effect and its graph, which is why views register under the effect's own
// parent rather than under the effect.
type EffectView struct {
	resource *Resource
	kind     uint32
}

// CreateCompiledEffect is cna_effect_create_compiled. The device handle is
// callback-scoped, so this is reachable only from inside a lifecycle callback.
func (d *Device) CreateCompiledEffect(effectCode []byte) (*Resource, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	effect, err := nativeEffectCreateCompiled(handle, effectCode)
	if err != nil {
		return nil, err
	}
	return d.runtime.registerResource(effect, resourceEffect, d.manager), nil
}

// LoadContentEffect is cna_content_manager_load_effect.
func (resource *Resource) LoadContentEffect(assetName string) (*Resource, error) {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return nil, err
	}
	effect, err := nativeContentManagerLoadEffect(handle, assetName)
	if err != nil {
		return nil, err
	}
	return resource.runtime.registerResource(effect, resourceEffect, resource.parent), nil
}

// ApplyEffect is cna_effect_apply.
func (resource *Resource) ApplyEffect() error {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return err
	}
	return nativeEffectApply(handle)
}

// CloneEffect is cna_effect_clone.
func (resource *Resource) CloneEffect() (*Resource, error) {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return nil, err
	}
	clone, err := nativeEffectClone(handle)
	if err != nil {
		return nil, err
	}
	return resource.runtime.registerResource(clone, resourceEffect, resource.parent), nil
}

// registerEffectView wraps one owned view handle. Views register under the
// EFFECT's parent, not under the effect, because a view outlives its effect and
// a parent-child relationship would claim an ordering CNA does not have.
func (resource *Resource) registerEffectView(handle uint64, kind uint32) *EffectView {
	return &EffectView{
		resource: resource.runtime.registerResource(handle, effectViewResourceKinds[kind], resource.parent),
		kind:     kind,
	}
}

// EffectParameters is cna_effect_get_parameters. Call it ONCE per effect.
func (resource *Resource) EffectParameters() (*EffectView, error) {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return nil, err
	}
	view, err := nativeEffectParameters(handle)
	if err != nil {
		return nil, err
	}
	return resource.registerEffectView(view, effectViewParameterCollection), nil
}

// EffectTechniques is cna_effect_get_techniques. Call it ONCE per effect.
func (resource *Resource) EffectTechniques() (*EffectView, error) {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return nil, err
	}
	view, err := nativeEffectTechniques(handle)
	if err != nil {
		return nil, err
	}
	return resource.registerEffectView(view, effectViewTechniqueCollection), nil
}

// EffectCurrentTechnique is cna_effect_get_current_technique. The handle it
// reports is a FRESH OWNED view of whichever technique is selected -- the
// Foundation 72 probe measured two calls answering two different handles -- so
// it is registered like every other view and the caller disposes it after
// matching it against the techniques it already holds. Reading it without
// registering it leaked one handle per effect, which CNA reported at teardown
// as "All owned C child resources must be destroyed before the game".
func (resource *Resource) EffectCurrentTechnique() (*EffectView, error) {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return nil, err
	}
	technique, err := nativeEffectCurrentTechnique(handle)
	if err != nil {
		return nil, err
	}
	if technique == 0 {
		return nil, nil
	}
	return resource.registerEffectView(technique, effectViewTechnique), nil
}

// SetEffectCurrentTechnique is cna_effect_set_current_technique. A zero handle
// is CNA_INVALID_HANDLE, which CNA documents as clearing the selection.
func (resource *Resource) SetEffectCurrentTechnique(technique uint64) error {
	handle, err := resource.liveHandle(resourceEffect)
	if err != nil {
		return err
	}
	return nativeEffectSetCurrentTechnique(handle, technique)
}

// BeginSpriteBatchWithEffect is cna_sprite_batch_begin_with_effect. A nil
// effect is CNA_INVALID_HANDLE, which CNA documents as selecting the default
// sprite effect -- what a null Effect means to the canonical call. A nil
// transform is the identity the effect-only overload uses.
func (resource *Resource) BeginSpriteBatchWithEffect(
	sortMode uint32, blend BlendStateValue, sampler SamplerStateValue,
	depth DepthStencilStateValue, rasterizer RasterizerStateValue,
	effect *Resource, transform *[16]float32,
) error {
	batch, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	var effectHandle uint64
	if effect != nil {
		if effectHandle, err = effect.liveHandle(resourceEffect); err != nil {
			return err
		}
		if resource.runtime != effect.runtime || resource.generation != effect.generation {
			return ErrStaleGeneration
		}
	}
	return nativeSpriteBatchBeginWithEffect(batch, sortMode, blend, sampler, depth, rasterizer, effectHandle, transform)
}

// Handle is the raw view handle, for the one comparison CurrentTechnique needs.
func (v *EffectView) Handle() (uint64, error) { return v.liveHandle() }

// Dispose destroys the view through the destroy route its kind names.
func (v *EffectView) Dispose() error {
	if v == nil {
		return nil
	}
	return v.resource.Dispose()
}

// Count is the collection count for a collection view.
func (v *EffectView) Count(collectionKind uint32) (uint64, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return 0, err
	}
	return nativeEffectCollectionCount(effectCollectionRoutes[collectionKind][0], collectionKind, handle)
}

// At is the element access for a collection view. It reports a FRESH owned
// element handle, which the caller wraps once and keeps.
func (v *EffectView) At(collectionKind uint32, index uint64, elementKind uint32) (*EffectView, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return nil, err
	}
	element, err := nativeEffectCollectionAt(effectCollectionRoutes[collectionKind][1], collectionKind, handle, index)
	if err != nil {
		return nil, err
	}
	return v.resource.registerEffectView(element, elementKind), nil
}

// effectCollectionRoutes names the canonical route behind each collection
// operation, so a refusal reports CNA's route rather than the multiplexer's.
var effectCollectionRoutes = [4][2]string{
	{"cna_effect_technique_collection_get_count", "cna_effect_technique_collection_get_at"},
	{"cna_effect_pass_collection_get_count", "cna_effect_pass_collection_get_at"},
	{"cna_effect_parameter_collection_get_count", "cna_effect_parameter_collection_get_at"},
	{"cna_effect_annotation_collection_get_count", "cna_effect_annotation_collection_get_at"},
}

func (v *EffectView) liveHandle() (uint64, error) {
	if v == nil {
		return 0, ErrDisposed
	}
	return v.resource.liveHandle(effectViewResourceKinds[v.kind])
}

// String reads one of the eight effect strings.
func (v *EffectView) String(kind uint32) (string, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return "", err
	}
	return nativeEffectString(kind, handle)
}

// ParameterMetadata is cna_effect_parameter_get_info.
func (v *EffectView) ParameterMetadata() (EffectMetadata, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return EffectMetadata{}, err
	}
	return nativeEffectParameterInfo(handle)
}

// AnnotationMetadata is cna_effect_annotation_get_info.
func (v *EffectView) AnnotationMetadata() (EffectMetadata, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return EffectMetadata{}, err
	}
	return nativeEffectAnnotationInfo(handle)
}

// Elements, StructureMembers and Annotations are the three nested collections a
// parameter reports; Passes and Annotations are a technique's; Annotations is a
// pass's. Each returns a fresh owned view the caller keeps.
func (v *EffectView) Elements() (*EffectView, error) {
	return v.nested(nativeEffectParameterElements, effectViewParameterCollection)
}

func (v *EffectView) StructureMembers() (*EffectView, error) {
	return v.nested(nativeEffectParameterStructureMembers, effectViewParameterCollection)
}

func (v *EffectView) ParameterAnnotations() (*EffectView, error) {
	return v.nested(nativeEffectParameterAnnotations, effectViewAnnotationCollection)
}

func (v *EffectView) Passes() (*EffectView, error) {
	return v.nested(nativeEffectTechniquePasses, effectViewPassCollection)
}

func (v *EffectView) TechniqueAnnotations() (*EffectView, error) {
	return v.nested(nativeEffectTechniqueAnnotations, effectViewAnnotationCollection)
}

func (v *EffectView) PassAnnotations() (*EffectView, error) {
	return v.nested(nativeEffectPassAnnotations, effectViewAnnotationCollection)
}

func (v *EffectView) nested(read func(uint64) (uint64, error), kind uint32) (*EffectView, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return nil, err
	}
	nested, err := read(handle)
	if err != nil {
		return nil, err
	}
	return v.resource.registerEffectView(nested, kind), nil
}

// ApplyPass is cna_effect_pass_apply.
func (v *EffectView) ApplyPass() error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	return nativeEffectPassApply(handle)
}

// The parameter value transfers. Every one takes CNA's own tagged identity,
// which the Graphics package chooses from the XNA overload it is projecting.

func (v *EffectView) ParameterValue(valueType uint32, out unsafe.Pointer) error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	return nativeEffectParameterGetValue(handle, valueType, out)
}

func (v *EffectView) SetParameterValue(valueType uint32, value unsafe.Pointer) error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	return nativeEffectParameterSetValue(handle, valueType, value)
}

func (v *EffectView) ParameterValues(valueType uint32, requested uint64, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return 0, err
	}
	return nativeEffectParameterGetValues(handle, valueType, requested, destination, capacity)
}

func (v *EffectView) SetParameterValues(valueType uint32, values unsafe.Pointer, count uint64) error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	return nativeEffectParameterSetValues(handle, valueType, values, count)
}

func (v *EffectView) SetParameterValueString(value string) error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	return nativeEffectParameterSetValueString(handle, value)
}

func (v *EffectView) SetParameterValueTexture(textureType uint32, texture *Resource) error {
	handle, err := v.liveHandle()
	if err != nil {
		return err
	}
	var textureHandle uint64
	if texture != nil {
		textureHandle, err = texture.anyLiveHandle()
		if err != nil {
			return err
		}
	}
	return nativeEffectParameterSetValueTexture(handle, textureType, textureHandle)
}

// The four annotation value reads that are not strings.

func (v *EffectView) AnnotationBoolean() (bool, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return false, err
	}
	return nativeEffectAnnotationBoolean(handle)
}

func (v *EffectView) AnnotationInt32() (int32, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return 0, err
	}
	return nativeEffectAnnotationInt32(handle)
}

func (v *EffectView) AnnotationSingle() (float32, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return 0, err
	}
	return nativeEffectAnnotationSingle(handle)
}

func (v *EffectView) AnnotationVector(width uint32) ([4]float32, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return [4]float32{}, err
	}
	return nativeEffectAnnotationVector(handle, width)
}

func (v *EffectView) AnnotationMatrix() ([16]float32, error) {
	handle, err := v.liveHandle()
	if err != nil {
		return [16]float32{}, err
	}
	return nativeEffectAnnotationMatrix(handle)
}

// ---------------------------------------------------------------------------
// Foundation 71 — the volume and cube texture families.
// ---------------------------------------------------------------------------

// Texture3DInfo is CNA_Texture3DInfo, flattened.
type Texture3DInfo struct {
	Width, Height, Depth uint32
	Levels               uint32
	Format               uint32
}

// TextureCubeInfo is CNA_TextureCubeInfo, flattened. A cube's faces are square,
// so ONE dimension describes the whole texture.
type TextureCubeInfo struct {
	Size   uint32
	Levels uint32
	Format uint32
}

// Texture3DTransfer is CNA_Texture3DTransfer, flattened: a mip BOX rather than
// a rectangle, which is what makes it a different structure from the 2D one
// rather than a wider version of it.
type Texture3DTransfer struct {
	Level                                 int32
	Left, Top, Right, Bottom, Front, Back int32
	StartIndex, ElementCount              uint64
}

// TextureCubeTransfer is CNA_TextureCubeTransfer, flattened. The face is part
// of the transfer rather than of the handle, which is why every cube member
// takes one.
type TextureCubeTransfer struct {
	Face                     uint32
	Level                    int32
	HasRectangle             bool
	X, Y, Width, Height      int32
	StartIndex, ElementCount uint64
}

// CreateTexture3D is cna_texture3d_create. The device handle is
// callback-scoped, so this is reachable only from inside a lifecycle callback.
func (d *Device) CreateTexture3D(width, height, depth uint32, mipMap bool, format uint32) (*Resource, Texture3DInfo, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, Texture3DInfo{}, err
	}
	texture, err := nativeTexture3DCreate(handle, width, height, depth, mipMap, format)
	if err != nil {
		return nil, Texture3DInfo{}, err
	}
	resource := d.runtime.registerResource(texture, resourceTexture3D, d.manager)
	info, infoErr := nativeTexture3DInfo(texture)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, Texture3DInfo{}, infoErr
	}
	return resource, info, nil
}

// CreateTextureCube is cna_texturecube_create.
func (d *Device) CreateTextureCube(size uint32, mipMap bool, format uint32) (*Resource, TextureCubeInfo, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, TextureCubeInfo{}, err
	}
	texture, err := nativeTextureCubeCreate(handle, size, mipMap, format)
	if err != nil {
		return nil, TextureCubeInfo{}, err
	}
	resource := d.runtime.registerResource(texture, resourceTextureCube, d.manager)
	info, infoErr := nativeTextureCubeInfo(texture)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, TextureCubeInfo{}, infoErr
	}
	return resource, info, nil
}

// SetTexture3DData is cna_texture3d_set_data.
func (resource *Resource) SetTexture3DData(transfer Texture3DTransfer, data unsafe.Pointer, capacity uint64) error {
	handle, err := resource.liveHandle(resourceTexture3D)
	if err != nil {
		return err
	}
	return nativeTexture3DSetData(handle, transfer, data, capacity)
}

// GetTexture3DData is cna_texture3d_get_data.
func (resource *Resource) GetTexture3DData(transfer Texture3DTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	handle, err := resource.liveHandle(resourceTexture3D)
	if err != nil {
		return 0, err
	}
	return nativeTexture3DGetData(handle, transfer, destination, capacity)
}

// SetTextureCubeData is cna_texturecube_set_data.
func (resource *Resource) SetTextureCubeData(transfer TextureCubeTransfer, data unsafe.Pointer, capacity uint64) error {
	handle, err := resource.liveHandle(resourceTextureCube)
	if err != nil {
		return err
	}
	return nativeTextureCubeSetData(handle, transfer, data, capacity)
}

// GetTextureCubeData is cna_texturecube_get_data.
func (resource *Resource) GetTextureCubeData(transfer TextureCubeTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	handle, err := resource.liveHandle(resourceTextureCube)
	if err != nil {
		return 0, err
	}
	return nativeTextureCubeGetData(handle, transfer, destination, capacity)
}

// ---------------------------------------------------------------------------
// Foundation 69 — the SpriteFont family.
// ---------------------------------------------------------------------------

// SpriteFontInfo is CNA_SpriteFontInfo, flattened. It is the point-in-time
// snapshot CNA reports for a font: the three MUTABLE layout values plus the
// character count the glyph read is sized from.
type SpriteFontInfo struct {
	CharacterCount      uint64
	LineSpacing         int32
	Spacing             float32
	DefaultCharacter    uint16
	HasDefaultCharacter bool
}

// SpriteFontRectangle is CNA_Rectangle in the two positions a glyph carries it.
// It is its own type rather than a reuse of ScissorRectangle for the reason
// every other interop struct here is its own: a shared name would make a change
// to one route's meaning silently change another's.
type SpriteFontRectangle struct {
	X, Y, Width, Height int32
}

// SpriteFontGlyph is CNA_SpriteFontGlyph, flattened. The three kerning values
// cross as separate fields rather than as a vector because interop declares no
// vector type: the Graphics package is where they become a Vector3.
type SpriteFontGlyph struct {
	Character   uint16
	GlyphBounds SpriteFontRectangle
	Cropping    SpriteFontRectangle
	KerningX    float32
	KerningY    float32
	KerningZ    float32
}

// LoadContentSpriteFont is cna_content_manager_load_sprite_font, the one CNA
// content route that reports TWO owned handles for one asset.
//
// Both are registered under the content manager's own parent, which is the
// game: CNA documents a loaded font and its atlas as independently owned and
// destroyed before the parent game, exactly as a loaded texture is.
//
// # The registration ORDER is load-bearing
//
// CNA retains the atlas while the font lives, so `cna_texture2d_destroy`
// refuses with INVALID_STATE until the font is destroyed. disposeAllResources
// releases in REVERSE registration order, so the ATLAS is registered FIRST and
// the font second -- which makes the runtime's own teardown release the font
// before the texture it holds, whatever else is registered afterwards. A
// registration in the natural reading order would make every game teardown
// that loaded a font report CNA's refusal.
func (resource *Resource) LoadContentSpriteFont(assetName string) (*Resource, *Resource, SpriteFontInfo, error) {
	handle, err := resource.liveHandle(resourceContentManager)
	if err != nil {
		return nil, nil, SpriteFontInfo{}, err
	}
	font, texture, err := nativeContentManagerLoadSpriteFont(handle, assetName)
	if err != nil {
		return nil, nil, SpriteFontInfo{}, err
	}
	ownedTexture := resource.runtime.registerResource(texture, resourceTexture2D, resource.parent)
	ownedFont := resource.runtime.registerResource(font, resourceSpriteFont, resource.parent)
	info, infoErr := nativeSpriteFontInfo(font)
	if infoErr != nil {
		// The documented order, even on the failure path: the font first, so
		// the atlas becomes releasable.
		_ = ownedFont.Dispose()
		_ = ownedTexture.Dispose()
		return nil, nil, SpriteFontInfo{}, infoErr
	}
	return ownedFont, ownedTexture, info, nil
}

// SpriteFontInfo is cna_sprite_font_get_info.
func (resource *Resource) SpriteFontInfo() (SpriteFontInfo, error) {
	handle, err := resource.liveHandle(resourceSpriteFont)
	if err != nil {
		return SpriteFontInfo{}, err
	}
	return nativeSpriteFontInfo(handle)
}

// SpriteFontGlyphs is cna_sprite_font_copy_glyphs, the whole table in one call.
func (resource *Resource) SpriteFontGlyphs(capacity uint64) ([]SpriteFontGlyph, error) {
	handle, err := resource.liveHandle(resourceSpriteFont)
	if err != nil {
		return nil, err
	}
	return nativeSpriteFontGlyphs(handle, capacity)
}

// SetSpriteFontDefaultCharacter is cna_sprite_font_set_default_character.
func (resource *Resource) SetSpriteFontDefaultCharacter(hasValue bool, value uint16) error {
	handle, err := resource.liveHandle(resourceSpriteFont)
	if err != nil {
		return err
	}
	return nativeSpriteFontSetDefaultCharacter(handle, hasValue, value)
}

// SetSpriteFontLineSpacing is cna_sprite_font_set_line_spacing.
func (resource *Resource) SetSpriteFontLineSpacing(lineSpacing int32) error {
	handle, err := resource.liveHandle(resourceSpriteFont)
	if err != nil {
		return err
	}
	return nativeSpriteFontSetLineSpacing(handle, lineSpacing)
}

// SpriteTextCommand is CNA_SpriteTextCommand without its text, which crosses
// as a Go string beside it. Every field is one of the nine arguments
// SpriteFont::InternalDraw takes, in the same meaning.
type SpriteTextCommand struct {
	PositionX, PositionY float32
	Red, Green, Blue     uint8
	Alpha                uint8
	Rotation             float32
	OriginX, OriginY     float32
	ScaleX, ScaleY       float32
	Effects              uint32
	LayerDepth           float32
}

// DrawString is cna_sprite_batch_draw_string. The receiver is the SpriteBatch;
// the font must belong to the same game and is retained by CNA until a
// successful End, exactly as a drawn texture is.
func (resource *Resource) DrawString(font *Resource, text string, command SpriteTextCommand) error {
	batch, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	fontHandle, err := font.liveHandle(resourceSpriteFont)
	if err != nil {
		return err
	}
	if resource.runtime != font.runtime || resource.generation != font.generation {
		return ErrStaleGeneration
	}
	return nativeSpriteBatchDrawString(batch, fontHandle, text, command)
}

// SetSpriteFontSpacing is cna_sprite_font_set_spacing.
func (resource *Resource) SetSpriteFontSpacing(spacing float32) error {
	handle, err := resource.liveHandle(resourceSpriteFont)
	if err != nil {
		return err
	}
	return nativeSpriteFontSetSpacing(handle, spacing)
}

// MaxTextureSlots and MaxSamplerSlots are CNA_TEXTURE_COLLECTION_MAX_TEXTURES
// and CNA_MAX_SAMPLERS. Both are sixteen, and both are what the projected
// collections report as their length.
const (
	MaxTextureSlots = 16
	MaxSamplerSlots = 16
)

// TextureSlot reads one slot of one of the device's two texture collections.
func (d *Device) TextureSlot(stage, slot uint32) (TextureSlot, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return TextureSlot{}, err
	}
	return nativeGraphicsDeviceGetTexture(handle, stage, slot)
}

// SetTextureSlot binds a texture to one slot, or empties it when texture is nil.
func (d *Device) SetTextureSlot(stage, slot uint32, texture *Resource) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	var textureHandle uint64
	if texture != nil {
		textureHandle, err = texture.liveTextureHandle()
		if err != nil {
			return err
		}
	}
	return nativeGraphicsDeviceSetTexture(handle, stage, slot, textureHandle)
}

// SamplerSlot reads one entry of one of the device's two sampler collections.
func (d *Device) SamplerSlot(stage, slot uint32) (SamplerStateValue, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return SamplerStateValue{}, err
	}
	return nativeGraphicsDeviceGetSamplerState(handle, stage, slot)
}

// SetSamplerSlot replaces one entry of one of the device's two sampler
// collections.
func (d *Device) SetSamplerSlot(stage, slot uint32, value SamplerStateValue) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetSamplerState(handle, stage, slot, value)
}

// RenderTargetInfo is CNA_RenderTargetInfo, flattened.
//
// Every field is what CNA APPLIED, not what was asked for. That is the same
// split the reference has: RenderTarget2D::CreateRenderTarget passes its
// arguments to GraphicsAdapter::QueryFormat, which SELECTS a format, a depth
// format and a sample count, and the RenderTargetHelper stores the selected
// ones -- so DepthStencilFormat and MultiSampleCount report the selection, not
// the preference. `preferredFormat` and `preferredDepthFormat` are the
// reference's own parameter names.
//
// RendererAvailable is CNA's and has no XNA counterpart. CNA permits
// construction on a backend with no real off-screen storage: creation succeeds,
// this reports false, and binding reports NOT_SUPPORTED. It is carried so the
// projection can say which of those it is looking at rather than reporting a
// bind failure with no explanation.
type RenderTargetInfo struct {
	Kind              uint32
	Width, Height     uint32
	LevelCount        uint32
	Format            uint32
	DepthFormat       uint32
	MultiSampleCount  int32
	Usage             uint32
	IsContentLost     bool
	RendererAvailable bool
}

// SpriteDestinationCommand is CNA_SpriteCommand: a sprite placed by a
// DESTINATION RECTANGLE rather than by a position and a scale.
//
// CNA declares the two as separate structures and says why: with a position,
// the origin is measured in source-texture pixels and the scale applies after
// that offset, which a caller cannot reproduce by computing a rectangle without
// repeating the canonical arithmetic. XNA agrees from the other side -- its
// seven Draw overloads all funnel into one InternalDraw whose Vector4
// destination means (x, y, scaleX, scaleY) or (x, y, width, height) depending
// on a `scaleDestination` bool -- so the two families are two routes here too.
type SpriteDestinationCommand struct {
	DestinationX, DestinationY          int32
	DestinationWidth, DestinationHeight int32
	SourceX, SourceY                    int32
	SourceWidth, SourceHeight           int32
	Red, Green, Blue, Alpha             uint8
	Rotation                            float32
	OriginX, OriginY                    float32
	Effects                             uint32
	LayerDepth                          float32
}

type SpriteCommand struct {
	PositionX, PositionY      float32
	SourceX, SourceY          int32
	SourceWidth, SourceHeight int32
	Red, Green, Blue, Alpha   uint8
	Rotation                  float32
	OriginX, OriginY          float32
	ScaleX, ScaleY            float32
	Effects                   uint32
	LayerDepth                float32
}

var (
	processRunMu   sync.Mutex
	nextGeneration atomic.Uint64
	currentRuntime atomic.Pointer[Runtime]

	// standaloneHolder is the Runtime whose STANDALONE session currently holds
	// processRunMu, or nil. It exists so a second Runtime fails fast instead of
	// blocking forever: a standalone session has no bounded duration -- it
	// lives until Dispose -- so waiting on the mutex would be a hang rather
	// than a queue. A session Run owns is bounded by Run and is not recorded
	// here, because waiting for it is exactly the right behaviour.
	standaloneHolder  atomic.Pointer[Runtime]
	ownerAssociations sync.Map
)

type ownerBinding struct {
	runtime  *Runtime
	resource *Resource
}

func NewRuntime(callbacks Callbacks) *Runtime {
	return &Runtime{callbacks: callbacks, title: "CNA-Go"}
}

// startSession opens the library, creates the native game, installs both
// signal families, and marks the Runtime live. It is the whole of what used to
// be the first half of Run.
//
// standalone records who is starting it. The session's SHAPE is identical
// either way -- same generation, same owner thread, same subscriptions, same
// timing and frame-hook read -- and only its ENDING differs.
func (r *Runtime) startSession(standalone bool) error {
	if held := standaloneHolder.Load(); held != nil && held != r {
		return errors.New("another Game holds the process's native session; dispose it first")
	}
	processRunMu.Lock()
	goruntime.LockOSThread()
	tracef("session: owner OS thread locked (standalone=%t)", standalone)

	unwind := func(err error) error {
		goruntime.UnlockOSThread()
		processRunMu.Unlock()
		return err
	}

	libraryPath, err := nativeLibraryPath()
	if err != nil {
		return unwind(err)
	}
	if err := nativeOpen(libraryPath); err != nil {
		return unwind(err)
	}
	tracef("session: admitted native library %q", libraryPath)
	unwindOpen := func(err error) error {
		nativeClose()
		return unwind(err)
	}

	generation := nextGeneration.Add(1)
	r.mu.Lock()
	if r.alive {
		r.mu.Unlock()
		return unwindOpen(errors.New("Game is already running"))
	}
	r.generation = generation
	r.ownerThread = nativeOwnerThreadID()
	r.alive = true
	r.callbackFailure = nil
	r.resources = nil
	r.mu.Unlock()
	currentRuntime.Store(r)

	// The optional frame-hook mask is read exactly once, here, on the owner
	// thread and before the native game exists. A Go callback object's method
	// set is fixed for the object's whole lifetime, so there is nothing to
	// re-read later and no mutable per-Game registration state anywhere.
	frameHooks := r.callbacks.FrameHookOverrides()
	timing := r.callbacks.TimingConfiguration()
	callbackHandle := cgo.NewHandle(r)
	game, createErr := nativeGameCreate(uintptr(callbackHandle), r.title, frameHooks, timing)
	if createErr != nil {
		callbackHandle.Delete()
		r.deactivate()
		return unwindOpen(createErr)
	}
	tracef("session: created Game handle %d", game)
	r.mu.Lock()
	r.game = game
	r.mu.Unlock()

	// One native subscription per canonical event, installed eagerly on the
	// owner thread the moment the native game exists. It is not installed
	// lazily on the first Go handler: CNA rejects cna_game_subscribe from any
	// other thread with CNA_RESULT_THREAD, and a Go consumer is free to add an
	// event handler from any goroutine at any time, so the only point at which
	// the call is guaranteed legal is right here.
	registrations, subscribeErr := nativeGameSubscribeEvents(game, uintptr(callbackHandle))
	if subscribeErr != nil {
		tracef("session: game-event subscription failed: %v", subscribeErr)
		destroyErr := nativeGameDestroy(game)
		r.deactivate()
		callbackHandle.Delete()
		return unwindOpen(errors.Join(subscribeErr, destroyErr))
	}
	r.mu.Lock()
	r.eventRegistrations = registrations
	r.mu.Unlock()
	tracef("session: installed %d native game-event registrations", gameEventCount)

	// The window signals, on the same rule and at the same moment. XNA's
	// GameWindow exists from the host's construction and raises its three
	// events for the whole life of the host, so the subscription window is
	// the native game's lifetime and not some later point.
	windowRegistrations, windowSubscribeErr := nativeGameWindowSubscribeEvents(game, uintptr(callbackHandle))
	if windowSubscribeErr != nil {
		tracef("session: window-event subscription failed: %v", windowSubscribeErr)
		releaseErr := r.releaseGameEvents()
		destroyErr := nativeGameDestroy(game)
		r.deactivate()
		callbackHandle.Delete()
		return unwindOpen(errors.Join(windowSubscribeErr, releaseErr, destroyErr))
	}
	r.mu.Lock()
	r.windowEventRegistrations = windowRegistrations
	r.callbackHandle = callbackHandle
	r.sessionLive = true
	r.standalone = standalone
	r.mu.Unlock()
	if standalone {
		standaloneHolder.Store(r)
	}
	tracef("session: installed %d native window-event registrations", gameWindowEventCount)
	return nil
}

// endSession is the second half of the old Run, unchanged in order.
//
// The registrations are released only AFTER the destroy. CNA raises the
// disposal signal from inside cna_game_destroy, and a registration handle stays
// valid across that call, so releasing first would silently drop the event.
func (r *Runtime) endSession() error {
	r.mu.Lock()
	if !r.sessionLive {
		r.mu.Unlock()
		return nil
	}
	game := r.game
	handle := r.callbackHandle
	standalone := r.standalone
	r.sessionLive = false
	r.standalone = false
	r.callbackHandle = 0
	r.mu.Unlock()

	cleanupErr := r.disposeAllResources()
	tracef("session: resource cleanup returned: %v", cleanupErr)
	destroyErr := nativeGameDestroy(game)
	tracef("session: Game destroy returned: %v", destroyErr)
	unsubscribeErr := r.releaseGameEvents()
	windowUnsubscribeErr := r.releaseGameWindowEvents()
	r.deactivate()
	handle.Delete()
	nativeClose()
	if standalone {
		standaloneHolder.CompareAndSwap(r, nil)
	}
	goruntime.UnlockOSThread()
	processRunMu.Unlock()
	tracef("session: ended (standalone=%t)", standalone)
	return errors.Join(cleanupErr, destroyErr, unsubscribeErr, windowUnsubscribeErr)
}

// Run projects Game::Run. It starts a session unless one is already live for
// this Runtime, runs the native loop, and ends the session only if it started
// it -- because whoever created the native game destroys it.
//
// Adopting an existing standalone session is the reference's own behaviour:
// XNA's Run calls host.Run() on a host the constructor already created, and
// CNA's Game::Run skips its own initialization when hasInitialized_ is already
// set. A frame-stepped Game that is then Run therefore keeps one native game
// and one initialization, exactly as the reference does.
func (r *Runtime) Run() error {
	if r == nil || r.callbacks == nil {
		return errors.New("native Game callbacks must not be nil")
	}
	r.mu.Lock()
	adopted := r.sessionLive
	r.mu.Unlock()
	if !adopted {
		if err := r.startSession(false); err != nil {
			return err
		}
	} else if err := r.requireOwnerThread(); err != nil {
		return err
	}

	r.mu.Lock()
	game := r.game
	r.mu.Unlock()

	tracef("Run: entering cna_game_run")
	runErr := nativeGameRun(game)
	tracef("Run: cna_game_run returned: %v", runErr)

	var endErr error
	if !adopted {
		endErr = r.endSession()
	}

	r.mu.Lock()
	callbackErr := r.callbackFailure
	r.mu.Unlock()
	if callbackErr != nil {
		return callbackErr
	}
	return errors.Join(runErr, endErr)
}

// Tick projects Game::Tick and RunOneFrame projects Game::RunOneFrame. Both
// start a standalone session on first use, because in the reference the host
// exists from the Game's construction and a frame step has something to drive.
//
// The two are NOT the same call and the difference was measured rather than
// assumed. Against the qualified artifact, on a game that has never run:
//
//	cna_game_tick            Update and Draw, and NO Initialize or LoadContent
//	cna_game_run_one_frame   Initialize and LoadContent first, then the same
//
// and a second run_one_frame initializes nothing further. That mirrors the
// reference's own split -- Tick is the clock step and does not initialize --
// with one measured CNA difference recorded in the milestone evidence: XNA's
// RunOneFrame does not initialize either, and CNA's does.
func (r *Runtime) Tick() error {
	return r.frameStep("cna_game_tick", nativeGameTick)
}

func (r *Runtime) RunOneFrame() error {
	return r.frameStep("cna_game_run_one_frame", nativeGameRunOneFrame)
}

// frameStep is the shared body. It starts a standalone session when none is
// live, and otherwise drives the live one -- which may be a standalone session
// this Runtime already owns.
//
// A frame step from inside a lifecycle callback is refused by CNA itself with
// CNA_RESULT_INVALID_STATE, because a frame step called from within a frame
// would re-enter the loop it is part of. That refusal is measured, not
// documented-and-trusted, and CNA-Go reports it rather than reproducing it.
func (r *Runtime) frameStep(operation string, step func(uint64) error) error {
	if r == nil || r.callbacks == nil {
		return errors.New("native Game callbacks must not be nil")
	}
	r.mu.Lock()
	live := r.sessionLive
	r.mu.Unlock()
	if !live {
		if err := r.startSession(true); err != nil {
			return err
		}
	} else if err := r.requireOwnerThread(); err != nil {
		return err
	}
	r.mu.Lock()
	game := r.game
	r.mu.Unlock()
	tracef("%s: entering", operation)
	err := step(game)
	tracef("%s: returned %v", operation, err)
	if err != nil {
		return err
	}
	r.mu.Lock()
	callbackErr := r.callbackFailure
	r.callbackFailure = nil
	r.mu.Unlock()
	return callbackErr
}

// EndStandaloneSession destroys a session a frame step started, and does
// nothing at all when there is none.
//
// It is deliberately NOT called by Run: a session Run started is Run's to end,
// and one it adopted belongs to whoever created it. Game::Dispose is the only
// caller, because Dispose is the member a consumer already uses to release a
// Game and because CNA admits exactly one C-owned game per process -- a
// standalone session that nothing ended would make the next one impossible.
func (r *Runtime) EndStandaloneSession() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	standalone := r.sessionLive && r.standalone
	r.mu.Unlock()
	if !standalone {
		return nil
	}
	if err := r.requireOwnerThread(); err != nil {
		return err
	}
	return r.endSession()
}

// HasStandaloneSession reports whether a frame step created a native game that
// is still alive. It exists so the framework package can describe the state in
// a diagnostic without reaching for the handle.
func (r *Runtime) HasStandaloneSession() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessionLive && r.standalone
}

// requireOwnerThread refuses a call from any goroutine other than the one whose
// OS thread the session locked. CNA does not thread-check cna_game_run or the
// window routes, so this is CNA-Go's own rule rather than a reported one -- and
// it is the same rule every other owner-thread operation already applies.
func (r *Runtime) requireOwnerThread() error {
	r.mu.Lock()
	owner := r.ownerThread
	r.mu.Unlock()
	if owner != nativeOwnerThreadID() {
		return ErrWrongThread
	}
	return nil
}

// releaseGameEvents releases every installed registration exactly once. The
// slots are zeroed under the same lock that publishes them, so a second call
// releases nothing rather than handing CNA a stale handle -- which it answers
// with CNA_RESULT_INVALID_HANDLE, not success.
func (r *Runtime) releaseGameEvents() error {
	r.mu.Lock()
	registrations := r.eventRegistrations
	r.eventRegistrations = [gameEventCount]uint64{}
	r.mu.Unlock()
	installed := false
	for _, handle := range registrations {
		if handle != 0 {
			installed = true
			break
		}
	}
	if !installed {
		return nil
	}
	return nativeGameUnsubscribeEvents(&registrations)
}

// invokeGameEvent delivers one canonical CNA game signal to the framework.
//
// It is not invokeCallback. A CNA_GameEventCallback returns void, so this
// boundary has no result channel at all: it cannot stop the game, and the
// canonical header says so directly -- "The exiting callback in
// CNA_GameCallbacks is a different thing: it can stop the game by failing,
// while these handlers only observe."
//
// Two consequences are load-bearing and are reproduced rather than papered
// over. A handler failure is recorded as the run's callback failure and
// surfaces from Run, so nothing is discarded, but it does not end the frame
// loop. And inCallback is deliberately NOT raised: an observation point is not
// an operation point, and the disposal signal in particular is delivered from
// inside cna_game_destroy, where the native game is already being torn down.
func (r *Runtime) invokeGameEvent(event uint32) {
	tracef("game event %s: enter", gameEventName(event))
	r.mu.Lock()
	alive := r.alive
	callbacks := r.callbacks
	r.mu.Unlock()
	if !alive || callbacks == nil {
		r.recordCallbackFailure(ErrStaleGeneration)
		tracef("game event %s: dropped, runtime is not live", gameEventName(event))
		return
	}
	if int(event) < gameEventCount {
		r.mu.Lock()
		r.gameEventDeliveries[event]++
		r.mu.Unlock()
	}
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic in Game event handler: %v\n%s", recovered, debug.Stack())
			}
		}()
		err = callbacks.GameEvent(event)
	}()
	if err != nil {
		r.recordCallbackFailure(err)
	}
	tracef("game event %s: return %v", gameEventName(event), err)
}

// GameEventDeliveries reports how many times each canonical signal was
// delivered to this Runtime, indexed by the GameEvent* identities. It survives
// deactivate() and a second Run adds to it, so a caller can compare two runs.
//
// It is the only way the disposal signal is observable at all now that it
// raises no public event, and it is deliberately confined to this internal
// package: a projected XNA member that exposed a native delivery count would be
// surface Microsoft never declared.
// releaseGameWindowEvents is releaseGameEvents for the window family, kept
// separate because the two tables have different lengths and because a window
// registration must never be released with a game slot's handle.
func (r *Runtime) releaseGameWindowEvents() error {
	r.mu.Lock()
	registrations := r.windowEventRegistrations
	r.windowEventRegistrations = [gameWindowEventCount]uint64{}
	r.mu.Unlock()
	installed := false
	for _, handle := range registrations {
		if handle != 0 {
			installed = true
			break
		}
	}
	if !installed {
		return nil
	}
	return nativeGameWindowUnsubscribeEvents(&registrations)
}

// invokeGameWindowEvent is invokeGameEvent for the window family. It is a
// separate entry point on purpose: both numberings start at zero, so one
// shared trampoline could route a window signal into a game event without any
// value looking wrong.
func (r *Runtime) invokeGameWindowEvent(event uint32) {
	tracef("window event %s: enter", gameWindowEventName(event))
	r.mu.Lock()
	alive := r.alive
	callbacks := r.callbacks
	r.mu.Unlock()
	if !alive || callbacks == nil {
		r.recordCallbackFailure(ErrStaleGeneration)
		tracef("window event %s: dropped, runtime is not live", gameWindowEventName(event))
		return
	}
	if int(event) < gameWindowEventCount {
		r.mu.Lock()
		r.windowEventDeliveries[event]++
		r.mu.Unlock()
	}
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic in GameWindow event handler: %v\n%s", recovered, debug.Stack())
			}
		}()
		err = callbacks.GameWindowEvent(event)
	}()
	if err != nil {
		r.recordCallbackFailure(err)
	}
	tracef("window event %s: return %v", gameWindowEventName(event), err)
}

// GameWindowEventDeliveries reports how many times each canonical window
// signal was delivered, for native qualification only.
func (r *Runtime) GameWindowEventDeliveries() [gameWindowEventCount]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.windowEventDeliveries
}

func gameWindowEventName(event uint32) string {
	switch event {
	case GameWindowEventClientSizeChanged:
		return "ClientSizeChanged"
	case GameWindowEventOrientationChanged:
		return "OrientationChanged"
	case GameWindowEventScreenDeviceNameChanged:
		return "ScreenDeviceNameChanged"
	default:
		return fmt.Sprintf("window-event-%d", event)
	}
}

func (r *Runtime) GameEventDeliveries() [gameEventCount]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gameEventDeliveries
}

// GameEventCount is the number of canonical signal identities, so a caller can
// size its own tables without spelling a CNA constant.
const GameEventCount = gameEventCount

func gameEventName(event uint32) string {
	switch event {
	case GameEventActivated:
		return "Activated"
	case GameEventDeactivated:
		return "Deactivated"
	case GameEventDisposed:
		return "Disposed"
	case GameEventExiting:
		return "Exiting"
	default:
		return fmt.Sprintf("unknown(%d)", event)
	}
}

// Exit projects Game::Exit. It no longer requires an active lifecycle callback.
//
// The callback requirement was correct while the only way to reach a live
// native game was from inside one: Run blocks the owner thread in
// cna_game_run, so outside a callback there was nothing live to ask. Foundation
// 47's standalone session makes "live, on the owner thread, outside a callback"
// a reachable state, and it is exactly the state a frame-stepped consumer calls
// Exit from. CNA agrees: cna_game_request_exit resolves the game with GetGame
// rather than GetCallableGame, so it carries no callback restriction of its own.
func (r *Runtime) Exit() error {
	game, err := r.activeGame(false)
	if err != nil {
		return err
	}
	return nativeGameRequestExit(game)
}

// The four timing and presentation settings, and the two frame commands.
//
// Each reports whether a live native game received it. XNA keeps these as
// managed fields its own loop reads, so the projected getter is a field read
// and the SETTER is what has to reach the loop; with no native game there is
// nothing to reach, and the value is carried in at creation instead. That is
// not a swallowed failure: it is the difference between "the runtime refused"
// and "there is no runtime yet", and the caller is told which.
func (r *Runtime) SetIsMouseVisible(value bool) (bool, error) {
	return r.applyToLiveGame(func(game uint64) error { return nativeGameSetIsMouseVisible(game, value) })
}

func (r *Runtime) SetIsFixedTimeStep(value bool) (bool, error) {
	return r.applyToLiveGame(func(game uint64) error { return nativeGameSetIsFixedTimeStep(game, value) })
}

func (r *Runtime) SetTargetElapsedTimeTicks(ticks int64) (bool, error) {
	return r.applyToLiveGame(func(game uint64) error { return nativeGameSetTargetElapsedTimeTicks(game, ticks) })
}

func (r *Runtime) SetInactiveSleepTimeTicks(ticks int64) (bool, error) {
	return r.applyToLiveGame(func(game uint64) error { return nativeGameSetInactiveSleepTimeTicks(game, ticks) })
}

func (r *Runtime) ResetElapsedTime() (bool, error) {
	return r.applyToLiveGame(nativeGameResetElapsedTime)
}

func (r *Runtime) SuppressDraw() (bool, error) {
	return r.applyToLiveGame(nativeGameSuppressDraw)
}

// applyToLiveGame runs one owner-thread operation against the live native game,
// or reports that there is none.
//
// requireCallback is false: every one of these is documented for an "active
// owned or callback-borrowed game handle", so a consumer may set a timing value
// from a lifecycle callback and from the owner thread between frames alike. The
// thread check is kept, because CNA answers CNA_RESULT_THREAD from any other
// thread and reporting that is more useful than reproducing it.
func (r *Runtime) applyToLiveGame(operation func(game uint64) error) (bool, error) {
	game, err := r.activeGame(false)
	if err != nil {
		if errors.Is(err, ErrStaleGeneration) {
			return false, nil
		}
		return false, err
	}
	return true, operation(game)
}

func (r *Runtime) Generation() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.generation
}

// invokeCallback runs one native callback that has no value channel of its
// own. Every lifecycle member and three of the four optional frame hooks are
// this shape.
func (r *Runtime) invokeCallback(kind uint32, game uint64, frame FrameTime) error {
	_, err := r.dispatchCallback(kind, game, frame)
	return err
}

// invokeBeginDrawCallback runs the begin_draw frame hook, whose Boolean is a
// value channel and stays strictly separate from its error. A refusal is
// (false, nil) and is not an error; a failure answers through the established
// callback-failure path and leaves the runtime's own drawing decision alone.
func (r *Runtime) invokeBeginDrawCallback(game uint64, frame FrameTime) (bool, error) {
	return r.dispatchCallback(callbackBeginDraw, game, frame)
}

func (r *Runtime) dispatchCallback(kind uint32, game uint64, frame FrameTime) (shouldDraw bool, err error) {
	// The default mirrors the canonical header: out_should_draw arrives as
	// CNA_TRUE and a null handler draws, so every path that never reaches a
	// begin_draw override leaves the frame drawing.
	shouldDraw = true
	tracef("callback %s: enter (Game %d)", callbackName(kind), game)
	r.mu.Lock()
	if !r.alive || r.game != 0 && r.game != game {
		r.mu.Unlock()
		err = ErrStaleGeneration
		r.recordCallbackFailure(err)
		return shouldDraw, err
	}
	if r.game == 0 {
		r.game = game
	}
	r.inCallback = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.inCallback = false
		r.mu.Unlock()
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic in Game callback: %v\n%s", recovered, debug.Stack())
		}
		if err != nil {
			r.recordCallbackFailure(err)
		}
		tracef("callback %s: return %v", callbackName(kind), err)
	}()

	switch kind {
	case callbackInitialize:
		return shouldDraw, r.callbacks.Initialize()
	case callbackLoadContent:
		return shouldDraw, r.callbacks.LoadContent()
	case callbackUpdate:
		return shouldDraw, r.callbacks.Update(frame)
	case callbackDraw:
		return shouldDraw, r.callbacks.Draw(frame)
	case callbackUnloadContent:
		return shouldDraw, r.callbacks.UnloadContent()
	case callbackExiting:
		return shouldDraw, nil
	case callbackBeginRun:
		return shouldDraw, r.callbacks.BeginRun()
	case callbackEndRun:
		return shouldDraw, r.callbacks.EndRun()
	case callbackEndDraw:
		return shouldDraw, r.callbacks.EndDraw()
	case callbackBeginDraw:
		return r.callbacks.BeginDraw()
	default:
		return shouldDraw, fmt.Errorf("unknown CNA Game callback kind %d", kind)
	}
}

func callbackName(kind uint32) string {
	switch kind {
	case callbackInitialize:
		return "Initialize"
	case callbackLoadContent:
		return "LoadContent"
	case callbackUpdate:
		return "Update"
	case callbackDraw:
		return "Draw"
	case callbackUnloadContent:
		return "UnloadContent"
	case callbackExiting:
		return "Exiting"
	case callbackBeginRun:
		return "BeginRun"
	case callbackEndRun:
		return "EndRun"
	case callbackBeginDraw:
		return "BeginDraw"
	case callbackEndDraw:
		return "EndDraw"
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func tracef(format string, values ...any) {
	if os.Getenv("CNA_GO_TRACE") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "[CNA-Go trace] "+format+"\n", values...)
}

func (r *Runtime) recordCallbackFailure(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	if r.callbackFailure == nil {
		r.callbackFailure = err
	}
	r.mu.Unlock()
}

func (r *Runtime) activeGame(requireCallback bool) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.alive || r.game == 0 {
		return 0, ErrStaleGeneration
	}
	if r.ownerThread != nativeOwnerThreadID() {
		return 0, ErrWrongThread
	}
	if requireCallback && !r.inCallback {
		return 0, ErrOutsideCallback
	}
	return r.game, nil
}

func (r *Runtime) deactivate() {
	currentRuntime.CompareAndSwap(r, nil)
	r.mu.Lock()
	r.alive = false
	r.inCallback = false
	r.game = 0
	r.ownerThread = 0
	r.mu.Unlock()
	ownerAssociations.Range(func(key, value any) bool {
		binding, ok := value.(ownerBinding)
		if ok && binding.runtime == r {
			ownerAssociations.Delete(key)
		}
		return true
	})
}

func CurrentRuntime() (*Runtime, bool) {
	current := currentRuntime.Load()
	return current, current != nil
}

// NativeVerification is compiler/tooling evidence about an explicitly named
// CNA library. It is internal so public XNA packages cannot become ABI probes.
type NativeVerification struct {
	ABIVersion             uint32
	BoundSymbols           []string
	MissingSymbols         []string
	SymbolIdentityVerified bool
	SymbolIdentityDetail   string
}

// VerifyNativeLibrary loads one explicit artifact using the same admission
// path as Runtime, measures its exports, and unloads it without creating a Game.
func VerifyNativeLibrary(path string) (NativeVerification, error) {
	if !filepath.IsAbs(path) {
		return NativeVerification{}, errors.New("native verification path must be absolute")
	}
	processRunMu.Lock()
	defer processRunMu.Unlock()
	if err := nativeOpen(path); err != nil {
		return NativeVerification{}, err
	}
	defer nativeClose()
	result := NativeVerification{ABIVersion: nativeABIVersion(), BoundSymbols: nativeBoundSymbols()}
	for _, symbol := range result.BoundSymbols {
		if !nativeHasLoadedSymbol(symbol) {
			result.MissingSymbols = append(result.MissingSymbols, symbol)
		}
	}
	result.SymbolIdentityVerified, result.SymbolIdentityDetail = nativeSymbolIdentity()
	return result, nil
}

func nativeLibraryPath() (string, error) {
	if explicit := os.Getenv("CNA_NATIVE_LIBRARY"); explicit != "" {
		if !filepath.IsAbs(explicit) {
			return "", fmt.Errorf("%w: CNA_NATIVE_LIBRARY must name an absolute file", ErrNativeUnavailable)
		}
		info, err := os.Stat(explicit)
		if err != nil || !info.Mode().IsRegular() {
			return "", fmt.Errorf("%w: CNA_NATIVE_LIBRARY does not name a regular file: %s", ErrNativeUnavailable, explicit)
		}
		return explicit, nil
	}
	return "libcna_c_api.so", nil
}

func (r *Runtime) CreateGraphicsDeviceManager() (*Resource, error) {
	game, err := r.activeGame(true)
	if err != nil {
		return nil, err
	}
	handle, err := nativeGraphicsDeviceManagerCreate(game)
	if err != nil {
		return nil, err
	}
	return r.registerResource(handle, resourceGraphicsDeviceManager, nil), nil
}

func (r *Runtime) GameDevice() (*Device, error) {
	if _, err := r.activeGame(true); err != nil {
		return nil, err
	}
	return &Device{runtime: r, generation: r.Generation(), ownership: borrowed}, nil
}

// The five canonical GraphicsDeviceManager signal identities. This family is
// the profile's THIRD independent numbering, and unlike the other two its
// device events do not start at zero: Disposed is 0.
const (
	ManagerEventDisposed        uint32 = 0
	ManagerEventDeviceCreated   uint32 = 1
	ManagerEventDeviceDisposing uint32 = 2
	ManagerEventDeviceReset     uint32 = 3
	ManagerEventDeviceResetting uint32 = 4

	managerEventCount = 5
)

// ManagerEventCount is the canonical manager-signal identity count.
const ManagerEventCount = managerEventCount

// ManagerSignals owns one manager's five native subscriptions and the
// cgo.Handle they carry as context.
//
// The context is per MANAGER rather than per Runtime, unlike the game and
// window families: those two belong to the game itself and there is one, while
// a manager is an object a consumer creates and a signal has to reach the one
// that was subscribed.
// The six canonical graphics-device signals. Four are CNA identities and two are
// separate CNA routes that carry a payload, so CNA-Go indexes all six in one
// array and bridge.c static-asserts that the four mirror CNA's numbering and
// that the two do not alias it.
const (
	DeviceEventDisposing uint32 = iota
	DeviceEventDeviceLost
	DeviceEventDeviceReset
	DeviceEventDeviceResetting
	DeviceEventResourceCreated
	DeviceEventResourceDestroyed
	deviceEventCount = 6
)

// DeviceEventCount is the exported count, for qualification tooling.
const DeviceEventCount = deviceEventCount

// DeviceSignalPayload is what the two payload-carrying device events report.
//
// Neither carries the OBJECT. CNA states why for each: ResourceCreated fires
// from the graphics-resource base constructor, where the concrete type does not
// exist yet, and the destroyed resource's tag is caller-owned native state. So
// the C event reports PRESENCE, and the name -- which is the one value that
// survives -- is copied out of callback-scoped bytes before they expire.
type DeviceSignalPayload struct {
	HasResource bool
	HasTag      bool
	Name        string
}

// DeviceSignals is the manager family's shape over the device's six events.
type DeviceSignals struct {
	mu            sync.Mutex
	runtime       *Runtime
	handle        cgo.Handle
	registrations [deviceEventCount]uint64
	sink          func(event uint32, payload DeviceSignalPayload) error
	deliveries    [deviceEventCount]int
	released      bool
}

// SubscribeDeviceEvents installs one native subscription per canonical device
// event, on the owner thread, against the callback-scoped device handle.
func SubscribeDeviceEvents(d *Device, sink func(event uint32, payload DeviceSignalPayload) error) (*DeviceSignals, error) {
	if sink == nil {
		return nil, errors.New("device signal sink must not be nil")
	}
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	signals := &DeviceSignals{runtime: d.runtime, sink: sink}
	signals.handle = cgo.NewHandle(signals)
	registrations, subscribeErr := nativeDeviceSubscribeEvents(handle, uintptr(signals.handle))
	if subscribeErr != nil {
		signals.handle.Delete()
		return nil, subscribeErr
	}
	signals.registrations = registrations
	return signals, nil
}

// Release releases every installed registration exactly once, for the reason
// ManagerSignals.Release is idempotent: a second release would hand CNA a stale
// registration and CNA answers that with CNA_RESULT_INVALID_HANDLE.
func (s *DeviceSignals) Release() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.released {
		s.mu.Unlock()
		return nil
	}
	s.released = true
	registrations := s.registrations
	s.registrations = [deviceEventCount]uint64{}
	handle := s.handle
	s.handle = 0
	s.mu.Unlock()
	err := nativeDeviceUnsubscribeEvents(&registrations)
	// A zero handle is one SubscribeDeviceEvents never created, which is the
	// state a signals value that was built but never subscribed is in. Deleting
	// it would panic with "misuse of an invalid Handle", so the release is
	// guarded rather than assumed.
	if handle != 0 {
		handle.Delete()
	}
	return err
}

// Deliveries reports how many times each canonical device signal arrived, for
// native qualification only.
func (s *DeviceSignals) Deliveries() [deviceEventCount]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deliveries
}

func (s *DeviceSignals) deliver(event uint32, payload DeviceSignalPayload) {
	s.mu.Lock()
	sink := s.sink
	released := s.released
	if int(event) < deviceEventCount {
		s.deliveries[event]++
	}
	s.mu.Unlock()
	if released || sink == nil {
		return
	}
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic in GraphicsDevice event handler: %v\n%s", recovered, debug.Stack())
			}
		}()
		err = sink(event, payload)
	}()
	if err != nil && s.runtime != nil {
		s.runtime.recordCallbackFailure(err)
	}
}

// DisposeDevice is cna_graphics_device_dispose: the reference's own
// GraphicsDevice::Dispose, which really disposes the device the Game owns. It is
// reached only when a consumer asks, which is what the reference does too; the
// facade's ownership stays BORROWED and CNA-Go never calls this on its own.
func (d *Device) DisposeDevice() error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceDispose(handle)
}

type ManagerSignals struct {
	mu            sync.Mutex
	runtime       *Runtime
	handle        cgo.Handle
	registrations [managerEventCount]uint64
	sink          func(event uint32) error
	deliveries    [managerEventCount]int
	released      bool
}

// SubscribeManagerEvents installs one native subscription per canonical manager
// event, on the owner thread, the moment the native manager exists.
func SubscribeManagerEvents(manager *Resource, sink func(event uint32) error) (*ManagerSignals, error) {
	if sink == nil {
		return nil, errors.New("manager signal sink must not be nil")
	}
	handle, err := managerHandle(manager)
	if err != nil {
		return nil, err
	}
	signals := &ManagerSignals{runtime: manager.runtime, sink: sink}
	signals.handle = cgo.NewHandle(signals)
	registrations, subscribeErr := nativeManagerSubscribeEvents(handle, uintptr(signals.handle))
	if subscribeErr != nil {
		signals.handle.Delete()
		return nil, subscribeErr
	}
	signals.registrations = registrations
	return signals, nil
}

// Release releases every installed registration exactly once and deletes the
// context handle. It is idempotent, because a second release would hand CNA a
// stale registration and CNA answers that with CNA_RESULT_INVALID_HANDLE.
func (s *ManagerSignals) Release() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.released {
		s.mu.Unlock()
		return nil
	}
	s.released = true
	registrations := s.registrations
	s.registrations = [managerEventCount]uint64{}
	handle := s.handle
	s.handle = 0
	s.mu.Unlock()
	err := nativeManagerUnsubscribeEvents(&registrations)
	handle.Delete()
	return err
}

// Deliveries reports how many times each canonical manager signal arrived, for
// native qualification only.
func (s *ManagerSignals) Deliveries() [managerEventCount]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deliveries
}

// deliver routes one signal to the sink with the containment every trampoline
// applies: a Go panic never crosses the C frame, and a handler failure is
// recorded on the runtime rather than dropped.
func (s *ManagerSignals) deliver(event uint32) {
	s.mu.Lock()
	sink := s.sink
	released := s.released
	if int(event) < managerEventCount {
		s.deliveries[event]++
	}
	s.mu.Unlock()
	if released || sink == nil {
		return
	}
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic in GraphicsDeviceManager event handler: %v\n%s", recovered, debug.Stack())
			}
		}()
		err = sink(event)
	}()
	if err != nil && s.runtime != nil {
		s.runtime.recordCallbackFailure(err)
	}
}

// ManagerCreateDevice, ManagerBeginDraw and ManagerEndDraw are the three
// IGraphicsDeviceManager operations. In the reference all three are PRIVATE
// explicit interface implementations, so they are not part of the type's
// declared public member set -- they are interface witnesses.
func ManagerCreateDevice(manager *Resource) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerCreateDevice(handle)
}

func ManagerBeginDraw(manager *Resource) (bool, error) {
	handle, err := managerHandle(manager)
	if err != nil {
		return false, err
	}
	return nativeManagerBeginDraw(handle)
}

func ManagerEndDraw(manager *Resource) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerEndDraw(handle)
}

// managerHandle resolves a live GraphicsDeviceManager handle on the owner
// thread. Unlike DeviceForManager it does NOT require an active lifecycle
// callback: the reference's setters are managed field stores a consumer makes
// from its own constructor, so requiring a callback would refuse the position
// every real XNA program sets them from.
func managerHandle(manager *Resource) (uint64, error) {
	if manager == nil {
		return 0, ErrDisposed
	}
	manager.mu.Lock()
	handle := manager.handle
	runtime := manager.runtime
	generation := manager.generation
	manager.mu.Unlock()
	if handle == 0 || runtime == nil {
		return 0, ErrDisposed
	}
	if _, err := runtime.activeGame(false); err != nil {
		return 0, err
	}
	if err := runtime.validateGeneration(generation, false); err != nil {
		return 0, err
	}
	return handle, nil
}

// The GraphicsDeviceManager configuration setters, one per projected property.
// Each is the push half of the settled managed-store-plus-native-push split:
// the framework package keeps the value the reference's own field keeps, and
// these carry it to the manager CNA applies at ChangeDevice time.
func ManagerSetGraphicsProfile(manager *Resource, profile uint32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetGraphicsProfile(handle, profile)
}

func ManagerSetIsFullScreen(manager *Resource, value bool) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetIsFullScreen(handle, value)
}

func ManagerSetPreferMultiSampling(manager *Resource, value bool) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetPreferMultiSampling(handle, value)
}

func ManagerSetPreferredBackBufferFormat(manager *Resource, format uint32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetPreferredBackBufferFormat(handle, format)
}

func ManagerSetPreferredBackBufferWidth(manager *Resource, width int32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetPreferredBackBufferWidth(handle, width)
}

func ManagerSetPreferredBackBufferHeight(manager *Resource, height int32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetPreferredBackBufferHeight(handle, height)
}

func ManagerSetPreferredDepthStencilFormat(manager *Resource, format uint32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetPreferredDepthStencilFormat(handle, format)
}

func ManagerSetSynchronizeWithVerticalRetrace(manager *Resource, value bool) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetSynchronizeWithVerticalRetrace(handle, value)
}

func ManagerSetSupportedOrientations(manager *Resource, orientations uint32) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerSetSupportedOrientations(handle, orientations)
}

// ManagerApplyChanges is GraphicsDeviceManager::ApplyChanges. CNA implements
// the reference's own guard -- a device that exists and is not dirty is left
// alone -- so CNA-Go does not re-implement it over state it does not hold.
func ManagerApplyChanges(manager *Resource) error {
	handle, err := managerHandle(manager)
	if err != nil {
		return err
	}
	return nativeManagerApplyChanges(handle)
}

func DeviceForManager(manager *Resource) (*Device, error) {
	if manager == nil {
		return nil, ErrDisposed
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.handle == 0 {
		return nil, ErrDisposed
	}
	if err := manager.runtime.validateGeneration(manager.generation, true); err != nil {
		return nil, err
	}
	return &Device{runtime: manager.runtime, manager: manager, generation: manager.generation, ownership: borrowed}, nil
}

// Live reports whether the facade still names the device generation it was made
// for. It is the check a member that answers from MANAGED state still needs: a
// CNA-Go GraphicsDevice facade can outlive the native device, which the
// reference's device object cannot, so a cached state object could otherwise be
// handed back for a device that is gone.
func (d *Device) Live() error {
	if d == nil || d.runtime == nil {
		return ErrDisposed
	}
	if _, err := d.runtime.activeGame(true); err != nil {
		return err
	}
	return d.runtime.validateGeneration(d.generation, true)
}

func (d *Device) nativeHandle() (uint64, error) {
	if d == nil || d.runtime == nil {
		return 0, ErrDisposed
	}
	game, err := d.runtime.activeGame(true)
	if err != nil {
		return 0, err
	}
	if err := d.runtime.validateGeneration(d.generation, true); err != nil {
		return 0, err
	}
	if d.manager == nil {
		return nativeGameGetGraphicsDevice(game)
	}
	d.manager.mu.Lock()
	handle := d.manager.handle
	d.manager.mu.Unlock()
	if handle == 0 {
		return 0, ErrDisposed
	}
	return nativeGraphicsDeviceManagerGetDevice(handle)
}

func (d *Device) Viewport() (Viewport, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return Viewport{}, err
	}
	return nativeGraphicsDeviceViewport(handle)
}

// The graphics device's render-state accessors.
//
// Every one of them asks CNA rather than reading a managed cache, and that is a
// measured decision rather than a shortcut. The reference caches these values
// in fields its own constructor initialises when it creates the D3D device;
// CNA-Go does not create the device, so a managed cache here would start at
// Go's zero values and disagree with the live device until something wrote to
// it. Asking CNA is one source of truth, and it is the same source the setters
// push to.
//
// The consequence is recorded rather than hidden: five of the reference's
// getters are single `ldfld`s and carry no error, and these carry one.

func (d *Device) BlendFactor() (uint8, uint8, uint8, uint8, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return nativeGraphicsDeviceBlendFactor(handle)
}

func (d *Device) SetBlendFactor(r, g, b, a uint8) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetBlendFactor(handle, r, g, b, a)
}

func (d *Device) MultiSampleMask() (int32, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeGraphicsDeviceMultiSampleMask(handle)
}

func (d *Device) SetMultiSampleMask(mask int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetMultiSampleMask(handle, mask)
}

func (d *Device) ReferenceStencil() (int32, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeGraphicsDeviceReferenceStencil(handle)
}

func (d *Device) SetReferenceStencil(stencil int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetReferenceStencil(handle, stencil)
}

func (d *Device) ScissorRectangle() (ScissorRectangle, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return ScissorRectangle{}, err
	}
	return nativeGraphicsDeviceScissorRectangle(handle)
}

func (d *Device) SetScissorRectangle(rectangle ScissorRectangle) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetScissorRectangle(handle, rectangle)
}

func (d *Device) SetViewport(viewport Viewport) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceSetViewport(handle, viewport)
}

func (d *Device) GraphicsProfile() (uint32, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeGraphicsDeviceGraphicsProfile(handle)
}

func (d *Device) Status() (uint32, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return 0, err
	}
	return nativeGraphicsDeviceStatus(handle)
}

func (d *Device) IsDisposed() (bool, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return false, err
	}
	return nativeGraphicsDeviceIsDisposed(handle)
}

// ClearWithOptions is cna_graphics_device_clear_options, which is a different
// route from Clear: it selects buffers with a mask and carries a depth and a
// stencil, where Clear takes four floats and clears what CNA decides.
func (d *Device) ClearWithOptions(options uint32, r, g, b, a uint8, depth float32, stencil int32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceClearOptions(handle, options, r, g, b, a, depth, stencil)
}

func (d *Device) DisplayMode() (DisplayMode, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return DisplayMode{}, err
	}
	return nativeGraphicsDeviceDisplayMode(handle)
}

func (d *Device) Present() error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDevicePresent(handle)
}

func (d *Device) Clear(red, green, blue, alpha float32) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	return nativeGraphicsDeviceClear(handle, red, green, blue, alpha)
}

func (d *Device) CreateTextureFromEncoded(data []byte) (*Resource, TextureInfo, error) {
	if len(data) == 0 {
		return nil, TextureInfo{}, errors.New("encoded texture data is empty")
	}
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, TextureInfo{}, err
	}
	texture, err := nativeTextureCreateEncoded(handle, data)
	if err != nil {
		return nil, TextureInfo{}, err
	}
	resource := d.runtime.registerResource(texture, resourceTexture2D, d.manager)
	info, infoErr := nativeTextureInfo(texture)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, TextureInfo{}, infoErr
	}
	return resource, info, nil
}

// CreateTexture is cna_texture2d_create: an EMPTY texture of a stated size,
// mip configuration and surface format, as opposed to CreateTextureFromEncoded,
// which decodes bytes. It registers and reads back exactly as that one does,
// including disposing the native texture if the read-back fails.
func (d *Device) CreateTexture(width, height uint32, mipMap bool, format uint32) (*Resource, TextureInfo, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, TextureInfo{}, err
	}
	texture, err := nativeTextureCreate(handle, width, height, mipMap, format)
	if err != nil {
		return nil, TextureInfo{}, err
	}
	resource := d.runtime.registerResource(texture, resourceTexture2D, d.manager)
	info, infoErr := nativeTextureInfo(texture)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, TextureInfo{}, infoErr
	}
	return resource, info, nil
}

// CreateRenderTarget2D creates an owned game-child render target.
//
// CNA permits creation on a backend with no real off-screen storage, so the
// returned info's RendererAvailable is part of the answer rather than an error:
// creation succeeded and binding will not.
func (d *Device) CreateRenderTarget2D(width, height uint32, mipMap bool, format, depthFormat uint32, multiSampleCount int32, usage uint32) (*Resource, RenderTargetInfo, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, RenderTargetInfo{}, err
	}
	target, err := nativeRenderTarget2DCreate(handle, width, height, mipMap, format, depthFormat, multiSampleCount, usage)
	if err != nil {
		return nil, RenderTargetInfo{}, err
	}
	resource := d.runtime.registerResource(target, resourceRenderTarget2D, d.manager)
	info, infoErr := nativeRenderTargetInfo(target)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, RenderTargetInfo{}, infoErr
	}
	return resource, info, nil
}

// RenderTargetInfo re-reads the target's applied description. IsContentLost is
// the one field that changes over a target's life, which is why the projection
// re-reads rather than caching the whole structure.
func (resource *Resource) RenderTargetInfo() (RenderTargetInfo, error) {
	handle, err := resource.liveHandle(resourceRenderTarget2D)
	if err != nil {
		return RenderTargetInfo{}, err
	}
	return nativeRenderTargetInfo(handle)
}

// SetRenderTarget2D binds one target, or restores the back buffer when the
// target is nil. It is the device's operation, not the target's, exactly as
// GraphicsDevice::SetRenderTarget is.
func (d *Device) SetRenderTarget2D(target *Resource) error {
	handle, err := d.nativeHandle()
	if err != nil {
		return err
	}
	var targetHandle uint64
	if target != nil {
		targetHandle, err = target.liveHandle(resourceRenderTarget2D)
		if err != nil {
			return err
		}
	}
	return nativeGraphicsDeviceSetRenderTarget2D(handle, targetHandle)
}

// TextureImageFormat is CNA_TextureImageFormat: PNG is 0 and JPEG is 1.
//
// XNA numbers its own SharedConstants.XnaImageFormat differently -- SaveAsJpeg
// passes 0 and SaveAsPng passes 2 -- so this is one of the few identities that
// does NOT cross unchanged, and the mapping is made once, in the Graphics
// package, where both names are visible.
const (
	TextureImageFormatPNG  uint32 = 0
	TextureImageFormatJPEG uint32 = 1
)

// TextureTransfer is CNA_Texture2DTransfer: which mip level, which optional
// rectangle, and which window of the caller's array one transfer covers.
type TextureTransfer struct {
	Level               int32
	HasRectangle        bool
	X, Y, Width, Height int32
	StartIndex          uint64
	ElementCount        uint64
}

// The eighteen CNA_TEXTURE_DATA_* element representations, in CNA's own order.
// They are the closed set a texture transfer's element type may be, and the
// Graphics package maps one Go type onto each.
const (
	TextureDataColor           uint32 = 0
	TextureDataBgr565          uint32 = 1
	TextureDataBgra5551        uint32 = 2
	TextureDataBgra4444        uint32 = 3
	TextureDataByte            uint32 = 4
	TextureDataNormalizedByte2 uint32 = 5
	TextureDataNormalizedByte4 uint32 = 6
	TextureDataRgba1010102     uint32 = 7
	TextureDataRg32            uint32 = 8
	TextureDataRgba64          uint32 = 9
	TextureDataAlpha8          uint32 = 10
	TextureDataSingle          uint32 = 11
	TextureDataVector2         uint32 = 12
	TextureDataVector4         uint32 = 13
	TextureDataHalfSingle      uint32 = 14
	TextureDataHalfVector2     uint32 = 15
	TextureDataHalfVector4     uint32 = 16
	TextureDataUShort          uint32 = 17
)

// SetTextureData and GetTextureData are the two typed transfers.
//
// Both take an unsafe.Pointer to the caller's array and its element count,
// because the element TYPE is decided by the caller and CNA identifies it by a
// CNA_TEXTURE_DATA_* identity rather than by a size. The Graphics package is
// where a Go type is turned into that identity, and it is also where the
// element size is checked against what the identity means -- interop copies
// bytes and validates nothing about their shape.
func (resource *Resource) SetTextureData(dataType uint32, transfer TextureTransfer, data unsafe.Pointer, capacity uint64) error {
	handle, err := resource.liveTextureHandle()
	if err != nil {
		return err
	}
	return nativeTextureSetData(handle, dataType, transfer, data, capacity)
}

func (resource *Resource) GetTextureData(dataType uint32, transfer TextureTransfer, destination unsafe.Pointer, capacity uint64) (uint64, error) {
	handle, err := resource.liveTextureHandle()
	if err != nil {
		return 0, err
	}
	return nativeTextureGetData(handle, dataType, transfer, destination, capacity)
}

// CreateTextureFromEncodedSized decodes bytes into a texture of a REQUESTED
// size, which is cna_texture2d_create_from_encoded_memory with a decode info
// where CreateTextureFromEncoded passes null.
//
// `zoom` is CNA's own flag and means what XNA's means: cover-and-crop when
// true, fit while preserving the aspect ratio when false.
func (d *Device) CreateTextureFromEncodedSized(data []byte, width, height uint32, zoom bool) (*Resource, TextureInfo, error) {
	if len(data) == 0 {
		return nil, TextureInfo{}, errors.New("encoded texture data is empty")
	}
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, TextureInfo{}, err
	}
	texture, err := nativeTextureCreateEncodedSized(handle, data, width, height, zoom)
	if err != nil {
		return nil, TextureInfo{}, err
	}
	resource := d.runtime.registerResource(texture, resourceTexture2D, d.manager)
	info, infoErr := nativeTextureInfo(texture)
	if infoErr != nil {
		_ = resource.Dispose()
		return nil, TextureInfo{}, infoErr
	}
	return resource, info, nil
}

// EncodeTexture asks CNA for the encoded byte count and then for the bytes, in
// that order, because CNA reports the size of an encode it has not performed
// yet and a caller cannot size the buffer any other way.
//
// The two calls are a measurement and then a copy, and the copy's own reported
// count is what bounds the returned slice: a second encode could in principle
// produce fewer bytes than the first measured, and trusting the first count
// would return trailing zeros as image data.
func (resource *Resource) EncodeTexture(imageFormat, width, height uint32) ([]byte, error) {
	handle, err := resource.liveTextureHandle()
	if err != nil {
		return nil, err
	}
	count, err := nativeTextureEncodedByteCount(handle, imageFormat, width, height)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	buffer := make([]byte, count)
	written, err := nativeTextureCopyEncoded(handle, imageFormat, width, height, buffer)
	if err != nil {
		return nil, err
	}
	if written > count {
		return nil, fmt.Errorf("cna_texture2d_copy_encoded wrote %d bytes into a %d byte buffer", written, count)
	}
	return buffer[:written], nil
}

func (d *Device) CreateSpriteBatch() (*Resource, error) {
	handle, err := d.nativeHandle()
	if err != nil {
		return nil, err
	}
	batch, err := nativeSpriteBatchCreate(handle)
	if err != nil {
		return nil, err
	}
	return d.runtime.registerResource(batch, resourceSpriteBatch, d.manager), nil
}

// The GameWindow routes. Two shapes, and which one a member gets is read from
// the reference implementor rather than chosen:
//
//   - windowValue reports (value, live, error). WindowsGameWindow guards
//     get_Handle, get_AllowUserResizing, set_AllowUserResizing, SetTitle and
//     get_ScreenDeviceName with `if (mainForm == null)`, so with no window the
//     reference returns a documented fallback instead of failing. `live` is
//     false there and the caller supplies the reference's own fallback.
//   - windowRequired returns an error when there is no live native game.
//     get_ClientBounds, BeginScreenDeviceChange and EndScreenDeviceChange have
//     NO null guard in the reference: they dereference mainForm directly and
//     throw NullReferenceException. Reporting a failure is that behaviour.
func (r *Runtime) windowGame(required bool) (uint64, bool, error) {
	game, err := r.activeGame(false)
	if err != nil {
		if errors.Is(err, ErrStaleGeneration) && !required {
			return 0, false, nil
		}
		return 0, false, err
	}
	return game, true, nil
}

// WindowHandle is GameWindow::get_Handle. With no window the reference answers
// IntPtr.Zero, so `live` false means exactly that.
func (r *Runtime) WindowHandle() (uintptr, bool, error) {
	game, live, err := r.windowGame(false)
	if err != nil || !live {
		return 0, live, err
	}
	value, callErr := nativeGameWindowNativeHandle(game)
	return uintptr(value), true, callErr
}

func (r *Runtime) WindowAllowUserResizing() (bool, bool, error) {
	game, live, err := r.windowGame(false)
	if err != nil || !live {
		return false, live, err
	}
	value, callErr := nativeGameWindowAllowUserResizing(game)
	return value, true, callErr
}

func (r *Runtime) SetWindowAllowUserResizing(value bool) (bool, error) {
	game, live, err := r.windowGame(false)
	if err != nil || !live {
		return live, err
	}
	return true, nativeGameWindowSetAllowUserResizing(game, value)
}

func (r *Runtime) WindowScreenDeviceName() (string, bool, error) {
	game, live, err := r.windowGame(false)
	if err != nil || !live {
		return "", live, err
	}
	value, callErr := nativeGameWindowScreenDeviceName(game)
	return value, true, callErr
}

// SetWindowTitle is GameWindow::SetTitle, whose Windows implementor guards on
// the form and otherwise does nothing.
func (r *Runtime) SetWindowTitle(title string) (bool, error) {
	game, live, err := r.windowGame(false)
	if err != nil || !live {
		return live, err
	}
	return true, nativeGameSetWindowTitle(game, title)
}

// WindowClientBounds is GameWindow::get_ClientBounds, which the reference
// implements WITHOUT a null guard: with no window it throws
// NullReferenceException, and here it reports a failure.
func (r *Runtime) WindowClientBounds() (int32, int32, int32, int32, error) {
	game, _, err := r.windowGame(true)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return nativeGameWindowClientBounds(game)
}

func (r *Runtime) BeginScreenDeviceChange(willBeFullScreen bool) error {
	game, _, err := r.windowGame(true)
	if err != nil {
		return err
	}
	return nativeGameWindowBeginScreenDeviceChange(game, willBeFullScreen)
}

func (r *Runtime) EndScreenDeviceChange(screenDeviceName string, clientWidth, clientHeight int32) error {
	game, _, err := r.windowGame(true)
	if err != nil {
		return err
	}
	return nativeGameWindowEndScreenDeviceChange(game, screenDeviceName, clientWidth, clientHeight)
}

func (r *Runtime) KeyboardState() ([4]uint64, error) {
	game, err := r.activeGame(true)
	if err != nil {
		return [4]uint64{}, err
	}
	return nativeKeyboardState(game)
}

func (r *Runtime) validateGeneration(generation uint64, requireCallback bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.alive || r.generation != generation {
		return ErrStaleGeneration
	}
	if r.ownerThread != nativeOwnerThreadID() {
		return ErrWrongThread
	}
	if requireCallback && !r.inCallback {
		return ErrOutsideCallback
	}
	return nil
}

func (r *Runtime) registerResource(handle uint64, kind resourceKind, parent *Resource) *Resource {
	resource := &Resource{runtime: r, parent: parent, generation: r.Generation(), handle: handle, kind: kind, ownership: owned}
	r.mu.Lock()
	r.resources = append(r.resources, resource)
	r.mu.Unlock()
	return resource
}

func (resource *Resource) Dispose() error {
	if resource == nil {
		return nil
	}
	resource.mu.Lock()
	if resource.handle == 0 {
		resource.mu.Unlock()
		return nil
	}
	// A native destroy may synchronously drive UnloadContent. Treat disposal
	// re-entered from that callback as the same in-progress operation instead
	// of waiting on ourselves.
	if resource.disposing {
		resource.mu.Unlock()
		return nil
	}
	generation := resource.generation
	kind := resource.kind
	handle := resource.handle
	resource.mu.Unlock()
	if err := resource.runtime.validateGeneration(generation, false); err != nil {
		return err
	}
	if kind == resourceGraphicsDeviceManager && resource.runtime.hasLiveChildren(resource) {
		return ErrChildrenAlive
	}
	resource.mu.Lock()
	if resource.handle == 0 || resource.disposing {
		resource.mu.Unlock()
		return nil
	}
	if resource.handle != handle || resource.generation != generation {
		resource.mu.Unlock()
		return ErrStaleGeneration
	}
	resource.disposing = true
	resource.mu.Unlock()

	err := destroyResource(kind, handle)
	resource.mu.Lock()
	resource.disposing = false
	if err == nil && resource.handle == handle {
		resource.handle = 0
	}
	resource.mu.Unlock()
	if err != nil {
		return err
	}
	return nil
}

func destroyResource(kind resourceKind, handle uint64) error {
	switch kind {
	case resourceGraphicsDeviceManager:
		return nativeGraphicsDeviceManagerDestroy(handle)
	case resourceTexture2D:
		return nativeTextureDestroy(handle)
	case resourceSpriteBatch:
		return nativeSpriteBatchDestroy(handle)
	case resourceContentManager:
		return nativeContentManagerDestroy(handle)
	case resourceIndexBuffer:
		return nativeIndexBufferDestroy(handle)
	case resourceVertexDeclaration:
		return nativeVertexDeclarationDestroy(handle)
	case resourceVertexBuffer:
		return nativeVertexBufferDestroy(handle)
	case resourceSpriteFont:
		return nativeSpriteFontDestroy(handle)
	case resourceTexture3D:
		return nativeTexture3DDestroy(handle)
	case resourceEffect:
		return nativeEffectDestroy(handle)
	case resourceEffectTechniqueCollection:
		return nativeEffectViewDestroy(effectViewTechniqueCollection, handle)
	case resourceEffectTechnique:
		return nativeEffectViewDestroy(effectViewTechnique, handle)
	case resourceEffectPassCollection:
		return nativeEffectViewDestroy(effectViewPassCollection, handle)
	case resourceEffectPass:
		return nativeEffectViewDestroy(effectViewPass, handle)
	case resourceEffectParameterCollection:
		return nativeEffectViewDestroy(effectViewParameterCollection, handle)
	case resourceEffectParameter:
		return nativeEffectViewDestroy(effectViewParameter, handle)
	case resourceEffectAnnotationCollection:
		return nativeEffectViewDestroy(effectViewAnnotationCollection, handle)
	case resourceEffectAnnotation:
		return nativeEffectViewDestroy(effectViewAnnotation, handle)
	case resourceTextureCube:
		// cna_texturecube_destroy is documented as destroying a TextureCube but
		// NOT a RenderTargetCube, which is the same per-kind split the render
		// target already has.
		return nativeTextureCubeDestroy(handle)
	case resourceRenderTarget2D:
		// A render target is a distinct CNA kind with its own destroy, and
		// cna_texture2d_destroy is documented as destroying a Texture2D but NOT
		// a render target. Routing it through the texture destroy would leak.
		return nativeRenderTargetDestroy(handle)
	default:
		return errors.New("unknown owned CNA resource kind")
	}
}

func (r *Runtime) hasLiveChildren(parent *Resource) bool {
	r.mu.Lock()
	resources := append([]*Resource(nil), r.resources...)
	r.mu.Unlock()
	for _, child := range resources {
		if child.parent != parent {
			continue
		}
		child.mu.Lock()
		alive := child.handle != 0
		child.mu.Unlock()
		if alive {
			return true
		}
	}
	return false
}

func (r *Runtime) disposeAllResources() error {
	r.mu.Lock()
	resources := append([]*Resource(nil), r.resources...)
	r.mu.Unlock()
	var failures []error
	for i := len(resources) - 1; i >= 0; i-- {
		if err := resources[i].Dispose(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (resource *Resource) BeginSpriteBatch() error {
	handle, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	return nativeSpriteBatchBegin(handle)
}

func (resource *Resource) DrawSprite(texture *Resource, command SpriteCommand) error {
	batch, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	textureHandle, err := texture.liveTextureHandle()
	if err != nil {
		return err
	}
	if resource.runtime != texture.runtime || resource.generation != texture.generation {
		return ErrStaleGeneration
	}
	return nativeSpriteBatchDrawScaled(batch, textureHandle, command)
}

// DrawSpriteToDestination is the destination-rectangle half of the same
// submission. It applies exactly the checks DrawSprite applies, in the same
// order, because a stale texture is stale for either family.
func (resource *Resource) DrawSpriteToDestination(texture *Resource, command SpriteDestinationCommand) error {
	batch, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	textureHandle, err := texture.liveTextureHandle()
	if err != nil {
		return err
	}
	if resource.runtime != texture.runtime || resource.generation != texture.generation {
		return ErrStaleGeneration
	}
	return nativeSpriteBatchDrawDestination(batch, textureHandle, command)
}

func (resource *Resource) EndSpriteBatch() error {
	handle, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	return nativeSpriteBatchEnd(handle)
}

// liveTextureHandle is liveHandle over the kinds CNA accepts where a TEXTURE
// handle is required.
//
// CNA's texture routes are documented as taking a "Texture2D or matching
// render-target handle", and that is the native fact the whole Go
// substitutability question rests on: to CNA a render target IS a texture. A
// kind check that admitted only resourceTexture2D would refuse at the binding
// what CNA accepts at the ABI.
func (resource *Resource) liveTextureHandle() (uint64, error) {
	if resource == nil {
		return 0, ErrDisposed
	}
	resource.mu.Lock()
	kind := resource.kind
	resource.mu.Unlock()
	if kind != resourceTexture2D && kind != resourceRenderTarget2D {
		return 0, ErrDisposed
	}
	return resource.liveHandle(kind)
}

func (resource *Resource) liveHandle(kind resourceKind) (uint64, error) {
	if resource == nil {
		return 0, ErrDisposed
	}
	resource.mu.Lock()
	defer resource.mu.Unlock()
	if resource.kind != kind || resource.handle == 0 {
		return 0, ErrDisposed
	}
	if err := resource.runtime.validateGeneration(resource.generation, true); err != nil {
		return 0, err
	}
	return resource.handle, nil
}

// anyLiveHandle is liveHandle without the kind test, for the one caller that
// legitimately does not care which buffer kind it is binding: Device.HandleOf,
// whose caller already holds a typed Go object. The GENERATION check is not
// relaxed, so a stale handle is still refused.
func (resource *Resource) anyLiveHandle() (uint64, error) {
	if resource == nil {
		return 0, ErrDisposed
	}
	resource.mu.Lock()
	defer resource.mu.Unlock()
	if resource.handle == 0 {
		return 0, ErrDisposed
	}
	if err := resource.runtime.validateGeneration(resource.generation, true); err != nil {
		return 0, err
	}
	return resource.handle, nil
}

func RegisterOwner(owner any, runtime *Runtime, resource *Resource) {
	ownerAssociations.Store(ownerPointer(owner), ownerBinding{runtime: runtime, resource: resource})
}

func UnregisterOwner(owner any) {
	ownerAssociations.Delete(ownerPointer(owner))
}

func BindingForOwner(owner any) (*Runtime, *Resource, bool) {
	value, ok := ownerAssociations.Load(ownerPointer(owner))
	if !ok {
		return nil, nil, false
	}
	binding := value.(ownerBinding)
	if binding.runtime == nil {
		return nil, nil, false
	}
	return binding.runtime, binding.resource, true
}

func ownerPointer(owner any) uintptr {
	return reflectPointer(owner)
}
