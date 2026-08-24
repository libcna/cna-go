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
)

const (
	callbackInitialize    = 1
	callbackLoadContent   = 2
	callbackUpdate        = 3
	callbackDraw          = 4
	callbackUnloadContent = 5
	callbackExiting       = 6
)

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
)

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
	processRunMu      sync.Mutex
	nextGeneration    atomic.Uint64
	currentRuntime    atomic.Pointer[Runtime]
	ownerAssociations sync.Map
)

type ownerBinding struct {
	runtime  *Runtime
	resource *Resource
}

func NewRuntime(callbacks Callbacks) *Runtime {
	return &Runtime{callbacks: callbacks, title: "CNA-Go"}
}

func (r *Runtime) Run() error {
	if r == nil || r.callbacks == nil {
		return errors.New("native Game callbacks must not be nil")
	}
	processRunMu.Lock()
	defer processRunMu.Unlock()
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()
	tracef("Run: owner OS thread locked")

	libraryPath, err := nativeLibraryPath()
	if err != nil {
		return err
	}
	if err := nativeOpen(libraryPath); err != nil {
		return err
	}
	defer nativeClose()
	tracef("Run: admitted native library %q", libraryPath)

	generation := nextGeneration.Add(1)
	r.mu.Lock()
	if r.alive {
		r.mu.Unlock()
		return errors.New("Game is already running")
	}
	r.generation = generation
	r.ownerThread = nativeOwnerThreadID()
	r.alive = true
	r.callbackFailure = nil
	r.resources = nil
	r.mu.Unlock()
	currentRuntime.Store(r)

	callbackHandle := cgo.NewHandle(r)
	game, createErr := nativeGameCreate(uintptr(callbackHandle), r.title)
	if createErr != nil {
		callbackHandle.Delete()
		r.deactivate()
		return createErr
	}
	tracef("Run: created Game handle %d", game)
	r.mu.Lock()
	r.game = game
	r.mu.Unlock()

	tracef("Run: entering cna_game_run")
	runErr := nativeGameRun(game)
	tracef("Run: cna_game_run returned: %v", runErr)
	cleanupErr := r.disposeAllResources()
	tracef("Run: resource cleanup returned: %v", cleanupErr)
	destroyErr := nativeGameDestroy(game)
	tracef("Run: Game destroy returned: %v", destroyErr)
	r.deactivate()
	callbackHandle.Delete()

	r.mu.Lock()
	callbackErr := r.callbackFailure
	r.mu.Unlock()
	if callbackErr != nil {
		return callbackErr
	}
	return errors.Join(runErr, cleanupErr, destroyErr)
}

func (r *Runtime) Exit() error {
	game, err := r.activeGame(true)
	if err != nil {
		return err
	}
	return nativeGameRequestExit(game)
}

func (r *Runtime) Generation() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.generation
}

func (r *Runtime) invokeCallback(kind uint32, game uint64, frame FrameTime) (err error) {
	tracef("callback %s: enter (Game %d)", callbackName(kind), game)
	r.mu.Lock()
	if !r.alive || r.game != 0 && r.game != game {
		r.mu.Unlock()
		err = ErrStaleGeneration
		r.recordCallbackFailure(err)
		return err
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
		return r.callbacks.Initialize()
	case callbackLoadContent:
		return r.callbacks.LoadContent()
	case callbackUpdate:
		return r.callbacks.Update(frame)
	case callbackDraw:
		return r.callbacks.Draw(frame)
	case callbackUnloadContent:
		return r.callbacks.UnloadContent()
	case callbackExiting:
		return nil
	default:
		return fmt.Errorf("unknown CNA Game callback kind %d", kind)
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
	ABIVersion     uint32
	BoundSymbols   []string
	MissingSymbols []string
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
	textureHandle, err := texture.liveHandle(resourceTexture2D)
	if err != nil {
		return err
	}
	if resource.runtime != texture.runtime || resource.generation != texture.generation {
		return ErrStaleGeneration
	}
	return nativeSpriteBatchDrawScaled(batch, textureHandle, command)
}

func (resource *Resource) EndSpriteBatch() error {
	handle, err := resource.liveHandle(resourceSpriteBatch)
	if err != nil {
		return err
	}
	return nativeSpriteBatchEnd(handle)
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
