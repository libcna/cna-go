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

	// GameEvent delivers one canonical CNA game signal. It is deliberately
	// separate from the five lifecycle members above: those project XNA's
	// protected virtual overrides and are declared by the public
	// GameCallbacks contract, while this one is the private bridge for the
	// four CLR events Game declares. Adding it here leaves GameCallbacks
	// untouched, which is what keeps every existing external implementation
	// of that interface compiling.
	GameEvent(event uint32) error

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
		return createErr
	}
	tracef("Run: created Game handle %d", game)
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
		tracef("Run: game-event subscription failed: %v", subscribeErr)
		destroyErr := nativeGameDestroy(game)
		r.deactivate()
		callbackHandle.Delete()
		return errors.Join(subscribeErr, destroyErr)
	}
	r.mu.Lock()
	r.eventRegistrations = registrations
	r.mu.Unlock()
	tracef("Run: installed %d native game-event registrations", gameEventCount)

	tracef("Run: entering cna_game_run")
	runErr := nativeGameRun(game)
	tracef("Run: cna_game_run returned: %v", runErr)
	cleanupErr := r.disposeAllResources()
	tracef("Run: resource cleanup returned: %v", cleanupErr)
	destroyErr := nativeGameDestroy(game)
	tracef("Run: Game destroy returned: %v", destroyErr)
	// The registrations are released only now. CNA raises the disposal signal
	// from inside cna_game_destroy, and a registration handle stays valid
	// across that call, so releasing first would silently drop the event.
	unsubscribeErr := r.releaseGameEvents()
	tracef("Run: released game-event registrations: %v", unsubscribeErr)
	r.deactivate()
	callbackHandle.Delete()

	r.mu.Lock()
	callbackErr := r.callbackFailure
	r.mu.Unlock()
	if callbackErr != nil {
		return callbackErr
	}
	return errors.Join(runErr, cleanupErr, destroyErr, unsubscribeErr)
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

func (r *Runtime) Exit() error {
	game, err := r.activeGame(true)
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
