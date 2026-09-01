package framework

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// GraphicsDeviceManager is the Go projection of
// Microsoft.Xna.Framework.GraphicsDeviceManager: the object a consumer
// configures a back buffer with, and the one that owns the native device.
//
// # Getters read managed state, setters reach CNA
//
// Every one of the reference's nine configuration getters is a single `ldfld`,
// and every setter is a store plus `isDeviceDirty = true` -- the value reaches
// the device later, when ChangeDevice runs. So the split is the settled one the
// Game timing members already use, and it falls out of the difference rather
// than being chosen:
//
//   - the getter is a field read of this object's own managed state. It
//     allocates nothing, validates nothing, reaches nothing and cannot fail;
//   - the setter validates as the reference does, stores as the reference does,
//     and then pushes to CNA's manager -- because CNA's ApplyChanges reads
//     CNA's copy, and a value that never reached it would be a setting that
//     appears to work and does not.
//
// The corresponding cna_graphics_device_manager_get_* routes are deliberately
// NOT bound, for the reason they are never bound in this binding: a native
// getter would be a second source of truth that could disagree with the field
// the setter wrote.
//
// # Three properties live in the Graphics package
//
// GraphicsProfile, PreferredBackBufferFormat and PreferredDepthStencilFormat
// are typed by Graphics-package enums, so the settled cross-package cycle rule
// projects them there as GraphicsDeviceManagerMember functions. Their VALUES
// are still this object's managed state; it holds them as the raw int32 the
// CLR enums are, and internal/servicebridge carries them across.
type GraphicsDeviceManager struct {
	runtime  *interop.Runtime
	resource *interop.Resource

	// The nine configuration fields, with the constructor's own defaults.
	// Three are held as raw int32 because their Go enum types live in the
	// Graphics package and this package cannot name them.
	graphicsProfile                int32
	backBufferFormat               int32
	depthStencilFormat             int32
	backBufferWidth                int32
	backBufferHeight               int32
	isFullScreen                   bool
	synchronizeWithVerticalRetrace bool
	allowMultiSampling             bool
	supportedOrientations          DisplayOrientation

	// isDeviceDirty is GraphicsDeviceManager::isDeviceDirty, which every
	// setter raises and which ApplyChanges consults. useResizedBackBuffer is
	// the flag the two dimension setters clear; nothing projected reads it,
	// and it is kept because both setters genuinely assign it.
	isDeviceDirty        bool
	useResizedBackBuffer bool

	// game is GraphicsDeviceManager::game, which the constructor stores and
	// Dispose reads to unregister the two services again.
	game *Game

	// The five events the type declares, each a private multicast delegate
	// field in the reference. Four of them are also the canonical
	// IGraphicsDeviceService events, which is why a consumer of that service
	// observes exactly these.
	deviceCreated   EventSource[*EventArgs]
	deviceResetting EventSource[*EventArgs]
	deviceReset     EventSource[*EventArgs]
	deviceDisposing EventSource[*EventArgs]
	disposed        EventSource[*EventArgs]

	// signals owns the five native subscriptions and the per-manager cgo
	// handle they carry.
	signals *interop.ManagerSignals

	// deviceFacade is GraphicsDeviceManager::device, the ONE GraphicsDevice
	// object the reference's field holds and its getter returns unchanged. It
	// is held as any because its Go type lives in the Graphics package;
	// internal/servicebridge is what reads and writes it from there.
	//
	// deviceFacadeGeneration is the native generation it was built for. The
	// reference replaces its field when ChangeDevice makes a new device, and
	// this is the equivalent boundary: a facade from an earlier Run belongs to
	// a dead generation and is replaced rather than handed out again.
	deviceFacade           any
	deviceFacadeGeneration uint64
}

// The two public static read-only fields, read from the type's own .cctor:
//
//	DefaultBackBufferWidth  = 0x320 = 800
//	DefaultBackBufferHeight = 0x1e0 = 480
//
// They are NOT GameWindow's defaults, which are 800x600 -- two different pairs
// of constants in one assembly, and the manager's is the one a game's back
// buffer starts at. CNA declares the same two values in
// runtime_graphics_manager.h.
const (
	graphicsDeviceManagerDefaultBackBufferWidth  int32 = 800
	graphicsDeviceManagerDefaultBackBufferHeight int32 = 480
)

// GraphicsDeviceManagerDefaultBackBufferWidth is the static read-only field
// GraphicsDeviceManager::DefaultBackBufferWidth, projected on the settled rule
// for a static read-only field: a zero-argument package function prefixed by
// the declaring type.
func GraphicsDeviceManagerDefaultBackBufferWidth() int32 {
	return graphicsDeviceManagerDefaultBackBufferWidth
}

// GraphicsDeviceManagerDefaultBackBufferHeight is
// GraphicsDeviceManager::DefaultBackBufferHeight.
func GraphicsDeviceManagerDefaultBackBufferHeight() int32 {
	return graphicsDeviceManagerDefaultBackBufferHeight
}

// The reference's DepthFormat.Depth24 is 2, which the constructor stores into
// depthStencilFormat. It is spelled as the raw CLR value here because the enum
// itself lives in the Graphics package.
const graphicsDeviceManagerDefaultDepthFormat int32 = 2

// errBackBufferDimension projects System.ArgumentOutOfRangeException, which the
// two dimension setters throw. errManagerArgument projects
// System.ArgumentNullException and System.ArgumentException, which the
// constructor throws.
var (
	errBackBufferDimension = errors.New("argument is out of range")
	errManagerArgument     = errors.New("argument is not valid")
)

// The two exact Resources strings the constructor's throw sites load, read from
// the retained Microsoft.Xna.Framework.Game.dll and checked by
// tools/resource_strings. The second contains a DOUBLE space before its second
// sentence, which is the reference's own.
const (
	gameCannotBeNull                    = "Game cannot be null."
	graphicsDeviceManagerAlreadyPresent = "A graphics device manager is already registered.  The graphics device manager cannot be changed once it is set."
)

// graphicsDeviceManagerServiceType is the CLR type token the reference's
// `ldtoken IGraphicsDeviceManager` pair resolves with. Unlike
// IGraphicsDeviceService it is a FRAMEWORK-package contract, so this package
// can build it and register the manager itself under it.
var graphicsDeviceManagerServiceType = reflect.TypeOf((*IGraphicsDeviceManager)(nil)).Elem()

// backBufferDimMustBePositive is the exact Resources string those two throw
// sites load, read from the Microsoft.Xna.Framework.Resources.resources stream
// of the retained Microsoft.Xna.Framework.Game.dll.
//
// The resource KEY is BackBufferDimMustBePositive and it describes the value
// badly, which is the third measured case of that in this profile: the string
// names the two PROPERTIES rather than "the dimension", and says "greater than
// zero" rather than "positive". Foundation 48 inferred it from the key and got
// it wrong; Foundation 49 read it and added tools/resource_strings so a
// claimed message that is not in a retained assembly is a test failure rather
// than a plausible-looking sentence.
const backBufferDimMustBePositive = "BackBufferWidth and BackBufferHeight must be greater than zero."

func backBufferDimensionError() error {
	return fmt.Errorf("%w: value: %s", errBackBufferDimension, backBufferDimMustBePositive)
}

// init installs the accessors the Graphics package uses to reach the three
// configuration slots whose Go enum types this package cannot name.
func init() {
	// The signal-delivery counter goes through the bridge rather than onto the
	// type. An exported accessor for it would be public API the XNA contract
	// does not declare, and the verifier says so: it was tried and reported as
	// an UNEXPECTED_MEMBER.
	servicebridge.SetManagerSignalReader(func(manager any) ([]int, bool) {
		typed, ok := manager.(*GraphicsDeviceManager)
		if !ok || typed == nil || typed.signals == nil {
			return nil, false
		}
		deliveries := typed.signals.Deliveries()
		return deliveries[:], true
	})
	servicebridge.SetManagerDeviceFacadeAccessors(
		func(manager any) (any, uint64, bool) {
			typed, ok := manager.(*GraphicsDeviceManager)
			if !ok || typed == nil || typed.deviceFacade == nil {
				return nil, 0, false
			}
			return typed.deviceFacade, typed.deviceFacadeGeneration, true
		},
		func(manager any, facade any, generation uint64) {
			if typed, ok := manager.(*GraphicsDeviceManager); ok && typed != nil {
				typed.deviceFacade, typed.deviceFacadeGeneration = facade, generation
			}
		})
	servicebridge.SetManagerConfigurationAccessors(
		func(manager any, slot servicebridge.ManagerConfigurationSlot) (int32, bool) {
			typed, ok := manager.(*GraphicsDeviceManager)
			if !ok || typed == nil {
				return 0, false
			}
			switch slot {
			case servicebridge.ManagerGraphicsProfile:
				return typed.graphicsProfile, true
			case servicebridge.ManagerPreferredBackBufferFormat:
				return typed.backBufferFormat, true
			case servicebridge.ManagerPreferredDepthStencilFormat:
				return typed.depthStencilFormat, true
			default:
				return 0, false
			}
		},
		func(manager any, slot servicebridge.ManagerConfigurationSlot, value int32) error {
			typed, ok := manager.(*GraphicsDeviceManager)
			if !ok || typed == nil {
				return errors.New("GraphicsDeviceManager is nil")
			}
			switch slot {
			case servicebridge.ManagerGraphicsProfile:
				typed.graphicsProfile = value
				typed.isDeviceDirty = true
				return typed.push(func(resource *interop.Resource) error {
					return interop.ManagerSetGraphicsProfile(resource, uint32(value))
				})
			case servicebridge.ManagerPreferredBackBufferFormat:
				typed.backBufferFormat = value
				typed.isDeviceDirty = true
				return typed.push(func(resource *interop.Resource) error {
					return interop.ManagerSetPreferredBackBufferFormat(resource, uint32(value))
				})
			case servicebridge.ManagerPreferredDepthStencilFormat:
				typed.depthStencilFormat = value
				typed.isDeviceDirty = true
				return typed.push(func(resource *interop.Resource) error {
					return interop.ManagerSetPreferredDepthStencilFormat(resource, uint32(value))
				})
			default:
				return errors.New("unknown GraphicsDeviceManager configuration slot")
			}
		})
}

// NewGraphicsDeviceManager is GraphicsDeviceManager::.ctor(Game).
//
// The reference's field initializers run first, before the base constructor and
// therefore before the null check, so the defaults are stored even on the path
// that throws:
//
//	synchronizeWithVerticalRetrace = true;
//	depthStencilFormat             = DepthFormat.Depth24;   // 2
//	backBufferWidth                = DefaultBackBufferWidth;  // 800
//	backBufferHeight               = DefaultBackBufferHeight; // 480
//
// CNA-Go still requires a LIVE native game here, which the reference does not:
// its constructor is pure managed and a consumer calls it from their own Game
// subclass's constructor. That is a pre-existing CNA-Go constraint rather than
// something this milestone introduced -- the native manager is created here --
// and it is recorded rather than hidden.
func NewGraphicsDeviceManager(game *Game) (*GraphicsDeviceManager, error) {
	if game == nil || game.runtime == nil {
		return nil, errors.New("Game is nil or uninitialized")
	}
	// The reference's duplicate check comes BEFORE anything else is created:
	//
	//	if (game.Services.GetService(typeof(IGraphicsDeviceManager)) != null)
	//	    throw new ArgumentException(Resources.GraphicsDeviceManagerAlreadyPresent);
	//
	// It is an ArgumentException with no parameter name, unlike the null check
	// above it, and that difference is the reference's own.
	//
	// The order is load-bearing rather than tidy. CNA refuses a second native
	// manager itself, with its own message -- "A GraphicsDeviceManager is
	// already registered with this Game." -- so creating first and checking
	// afterwards reports CNA's sentence where the reference reports
	// Microsoft's, and leaves an orphaned native manager to unwind.
	if existing, _ := game.Services().GetService(graphicsDeviceManagerServiceType); existing != nil {
		return nil, fmt.Errorf("%w: %s", errManagerArgument, graphicsDeviceManagerAlreadyPresent)
	}
	resource, err := game.runtime.CreateGraphicsDeviceManager()
	if err != nil {
		return nil, err
	}
	manager := newGraphicsDeviceManagerState()
	manager.runtime = game.runtime
	manager.resource = resource
	manager.game = game
	interop.RegisterOwner(manager, manager.runtime, manager.resource)

	if err := game.Services().AddService(graphicsDeviceManagerServiceType, manager); err != nil {
		_ = manager.releaseNative()
		return nil, err
	}
	// The SECOND registration, which is what makes Game.GraphicsDevice and
	// DrawableGameComponent.Initialize work without a consumer supplying a
	// service of their own. It goes through internal/servicebridge because
	// this package can neither name IGraphicsDeviceService nor implement it:
	// the contract's device accessor returns a Graphics-package type.
	if err := servicebridge.PublishDeviceService(game.Services(), manager); err != nil {
		_ = game.Services().RemoveService(graphicsDeviceManagerServiceType)
		_ = manager.releaseNative()
		return nil, err
	}

	// One native subscription per canonical manager event, installed on the
	// owner thread the moment the native manager exists -- the same rule the
	// game and window families follow, and for the same reason.
	signals, subscribeErr := interop.SubscribeManagerEvents(resource, manager.raiseNativeManagerEvent)
	if subscribeErr != nil {
		_ = servicebridge.UnpublishDeviceService(game.Services(), manager)
		_ = game.Services().RemoveService(graphicsDeviceManagerServiceType)
		_ = manager.releaseNative()
		return nil, subscribeErr
	}
	manager.signals = signals

	// The reference also subscribes three GameWindow handlers here --
	// ClientSizeChanged, ScreenDeviceNameChanged and OrientationChanged -- and
	// CNA-Go deliberately does not. All three run PRIVATE handlers whose work
	// is resizing and re-orienting the back buffer, which is exactly what
	// CNA's own manager already does from its own subscriptions. Adding Go
	// subscriptions would run that work twice.
	return manager, nil
}

// releaseNative undoes the native half of construction when a later step of the
// constructor fails, so a refused registration leaves no orphaned manager.
func (m *GraphicsDeviceManager) releaseNative() error {
	if m.resource == nil {
		return nil
	}
	err := m.resource.Dispose()
	interop.UnregisterOwner(m)
	return err
}

// raiseNativeManagerEvent is the private end of the native manager bridge. It
// is not a projected member and is not reachable from outside this package.
//
// Each identity goes to the reference's own raise path. The four device events
// reach the protected On... methods, which is where the reference's private
// HandleDeviceLost, HandleDeviceReset, HandleDeviceResetting and
// HandleDisposing send them; the disposal signal raises the Disposed event
// directly, because the reference has no protected raiser for it.
func (m *GraphicsDeviceManager) raiseNativeManagerEvent(event uint32) error {
	if m == nil {
		return nil
	}
	switch event {
	case interop.ManagerEventDeviceCreated:
		return m.OnDeviceCreated(m, EventArgsEmpty())
	case interop.ManagerEventDeviceDisposing:
		return m.OnDeviceDisposing(m, EventArgsEmpty())
	case interop.ManagerEventDeviceReset:
		return m.OnDeviceReset(m, EventArgsEmpty())
	case interop.ManagerEventDeviceResetting:
		return m.OnDeviceResetting(m, EventArgsEmpty())
	case interop.ManagerEventDisposed:
		return m.disposed.Raise(m, EventArgsEmpty())
	default:
		return nil
	}
}

// The four protected raisers. Every one is the same 22-byte body:
//
//	if (this.<event> != null) this.<event>(sender, args);
//
// They pass the caller's sender and args through, and the native bridge above
// calls them with the manager and EventArgs.Empty -- which is what the
// reference's own private handlers pass.

// OnDeviceCreated is GraphicsDeviceManager::OnDeviceCreated.
func (m *GraphicsDeviceManager) OnDeviceCreated(sender any, args *EventArgs) error {
	return m.deviceCreated.Raise(sender, args)
}

// OnDeviceDisposing is GraphicsDeviceManager::OnDeviceDisposing.
func (m *GraphicsDeviceManager) OnDeviceDisposing(sender any, args *EventArgs) error {
	return m.deviceDisposing.Raise(sender, args)
}

// OnDeviceReset is GraphicsDeviceManager::OnDeviceReset.
func (m *GraphicsDeviceManager) OnDeviceReset(sender any, args *EventArgs) error {
	return m.deviceReset.Raise(sender, args)
}

// OnDeviceResetting is GraphicsDeviceManager::OnDeviceResetting.
func (m *GraphicsDeviceManager) OnDeviceResetting(sender any, args *EventArgs) error {
	return m.deviceResetting.Raise(sender, args)
}

// The five event accessor pairs. The first four are also the canonical
// IGraphicsDeviceService events, which is why the adapter the Graphics package
// registers forwards straight to them.

// AddDeviceCreatedHandler registers a handler for
// GraphicsDeviceManager::DeviceCreated.
func (m *GraphicsDeviceManager) AddDeviceCreatedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return m.deviceCreated.Add(handler)
}

// RemoveDeviceCreatedHandler removes the registration the token names.
func (m *GraphicsDeviceManager) RemoveDeviceCreatedHandler(subscription EventSubscription) error {
	return m.deviceCreated.Remove(subscription)
}

// AddDeviceResettingHandler registers a handler for DeviceResetting.
func (m *GraphicsDeviceManager) AddDeviceResettingHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return m.deviceResetting.Add(handler)
}

// RemoveDeviceResettingHandler removes the registration the token names.
func (m *GraphicsDeviceManager) RemoveDeviceResettingHandler(subscription EventSubscription) error {
	return m.deviceResetting.Remove(subscription)
}

// AddDeviceResetHandler registers a handler for DeviceReset.
func (m *GraphicsDeviceManager) AddDeviceResetHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return m.deviceReset.Add(handler)
}

// RemoveDeviceResetHandler removes the registration the token names.
func (m *GraphicsDeviceManager) RemoveDeviceResetHandler(subscription EventSubscription) error {
	return m.deviceReset.Remove(subscription)
}

// AddDeviceDisposingHandler registers a handler for DeviceDisposing.
func (m *GraphicsDeviceManager) AddDeviceDisposingHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return m.deviceDisposing.Add(handler)
}

// RemoveDeviceDisposingHandler removes the registration the token names.
func (m *GraphicsDeviceManager) RemoveDeviceDisposingHandler(subscription EventSubscription) error {
	return m.deviceDisposing.Remove(subscription)
}

// AddDisposedHandler registers a handler for GraphicsDeviceManager::Disposed,
// which has no protected raiser: the reference invokes the delegate field
// directly from the end of Dispose(bool).
func (m *GraphicsDeviceManager) AddDisposedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return m.disposed.Add(handler)
}

// RemoveDisposedHandler removes the registration the token names.
func (m *GraphicsDeviceManager) RemoveDisposedHandler(subscription EventSubscription) error {
	return m.disposed.Remove(subscription)
}

// The three IGraphicsDeviceManager operations. In the reference all three are
// PRIVATE explicit interface implementations, so they are interface WITNESSES
// rather than declared public members of this type -- exported here only
// because Go has no explicit implementation and an interface a type satisfies
// needs exported method names.

// CreateDevice is IGraphicsDeviceManager::CreateDevice, which the reference
// implements as `ChangeDevice(true)`. Game::Initialize calls it, so a consumer
// normally never does.
func (m *GraphicsDeviceManager) CreateDevice() error {
	if m == nil || m.resource == nil {
		return errors.New("GraphicsDeviceManager is nil or uninitialized")
	}
	return interop.ManagerCreateDevice(m.resource)
}

// BeginDraw is IGraphicsDeviceManager::BeginDraw:
//
//	if (!EnsureDevice()) return false;
//	beginDrawOk = true;
//	return true;
//
// The Boolean is the source result -- whether drawing may proceed -- and stays
// a channel separate from the error, exactly as the contract declares it.
func (m *GraphicsDeviceManager) BeginDraw() (bool, error) {
	if m == nil || m.resource == nil {
		return false, errors.New("GraphicsDeviceManager is nil or uninitialized")
	}
	return interop.ManagerBeginDraw(m.resource)
}

// EndDraw is IGraphicsDeviceManager::EndDraw, which presents the frame and
// swallows DeviceLostException and DeviceNotResetException.
func (m *GraphicsDeviceManager) EndDraw() error {
	if m == nil || m.resource == nil {
		return errors.New("GraphicsDeviceManager is nil or uninitialized")
	}
	return interop.ManagerEndDraw(m.resource)
}

// newGraphicsDeviceManagerState is the reference constructor's FIELD
// INITIALIZER block, which runs before the base constructor and therefore
// before the null check -- so these values are stored even on the path that
// throws. It is separated because it is the one part of the constructor that
// reaches nothing, and it is what the managed tests measure.
//
// graphicsProfile starts at zero rather than at ReadDefaultGraphicsProfile's
// answer: that method probes the platform for HiDef support, and CNA-Go asks
// CNA rather than guessing. GraphicsProfile.Reach is zero, which is what a
// machine with no HiDef adapter would get anyway.
func newGraphicsDeviceManagerState() *GraphicsDeviceManager {
	return &GraphicsDeviceManager{
		graphicsProfile:                0,
		backBufferFormat:               0,
		depthStencilFormat:             graphicsDeviceManagerDefaultDepthFormat,
		backBufferWidth:                graphicsDeviceManagerDefaultBackBufferWidth,
		backBufferHeight:               graphicsDeviceManagerDefaultBackBufferHeight,
		synchronizeWithVerticalRetrace: true,
		supportedOrientations:          DisplayOrientationDefault,
		isDeviceDirty:                  false,
	}
}

// push carries one configuration value to CNA's manager, or reports that there
// is no live one to carry it to.
//
// With no live native manager the value is still STORED, exactly as the
// reference stores it: the reference's setter never reaches a device either,
// and the difference between "the runtime refused" and "there is no runtime
// yet" is what the caller is told.
func (m *GraphicsDeviceManager) push(operation func(*interop.Resource) error) error {
	if m == nil || m.resource == nil {
		return nil
	}
	err := operation(m.resource)
	if errors.Is(err, interop.ErrStaleGeneration) || errors.Is(err, interop.ErrDisposed) {
		return nil
	}
	return err
}

// PreferredBackBufferWidth is GraphicsDeviceManager::get_PreferredBackBufferWidth,
// one `ldfld` over the field the constructor set to DefaultBackBufferWidth.
func (m *GraphicsDeviceManager) PreferredBackBufferWidth() int32 {
	return m.backBufferWidth
}

// SetPreferredBackBufferWidth is set_PreferredBackBufferWidth:
//
//	if (value <= 0)
//	    throw new ArgumentOutOfRangeException("value", Resources.BackBufferDimMustBePositive);
//	backBufferWidth = value; useResizedBackBuffer = false; isDeviceDirty = true;
//
// The comparison is `bgt` on zero, so ZERO IS REJECTED and one is accepted.
// Both other stores are part of the setter and are reproduced: a resized back
// buffer is forgotten the moment a consumer states a preference.
func (m *GraphicsDeviceManager) SetPreferredBackBufferWidth(value int32) error {
	if value <= 0 {
		return backBufferDimensionError()
	}
	m.backBufferWidth = value
	m.useResizedBackBuffer = false
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetPreferredBackBufferWidth(resource, value)
	})
}

// PreferredBackBufferHeight is get_PreferredBackBufferHeight.
func (m *GraphicsDeviceManager) PreferredBackBufferHeight() int32 {
	return m.backBufferHeight
}

// SetPreferredBackBufferHeight is set_PreferredBackBufferHeight, the same
// shape as the width setter down to the same resource string.
func (m *GraphicsDeviceManager) SetPreferredBackBufferHeight(value int32) error {
	if value <= 0 {
		return backBufferDimensionError()
	}
	m.backBufferHeight = value
	m.useResizedBackBuffer = false
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetPreferredBackBufferHeight(resource, value)
	})
}

// IsFullScreen is get_IsFullScreen, one `ldfld`. The constructor does not
// assign it, so it starts false.
func (m *GraphicsDeviceManager) IsFullScreen() bool { return m.isFullScreen }

// SetIsFullScreen is set_IsFullScreen: store, then isDeviceDirty = true. It
// validates nothing.
func (m *GraphicsDeviceManager) SetIsFullScreen(value bool) error {
	m.isFullScreen = value
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetIsFullScreen(resource, value)
	})
}

// SynchronizeWithVerticalRetrace is get_SynchronizeWithVerticalRetrace. The
// constructor stores TRUE, so a game vsyncs unless it says otherwise.
func (m *GraphicsDeviceManager) SynchronizeWithVerticalRetrace() bool {
	return m.synchronizeWithVerticalRetrace
}

// SetSynchronizeWithVerticalRetrace is set_SynchronizeWithVerticalRetrace.
func (m *GraphicsDeviceManager) SetSynchronizeWithVerticalRetrace(value bool) error {
	m.synchronizeWithVerticalRetrace = value
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetSynchronizeWithVerticalRetrace(resource, value)
	})
}

// PreferMultiSampling is get_PreferMultiSampling, whose backing field is named
// allowMultiSampling -- the property and the field disagree in the reference
// and the field name is kept.
func (m *GraphicsDeviceManager) PreferMultiSampling() bool { return m.allowMultiSampling }

// SetPreferMultiSampling is set_PreferMultiSampling.
func (m *GraphicsDeviceManager) SetPreferMultiSampling(value bool) error {
	m.allowMultiSampling = value
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetPreferMultiSampling(resource, value)
	})
}

// SupportedOrientations is get_SupportedOrientations, one `ldfld`.
func (m *GraphicsDeviceManager) SupportedOrientations() DisplayOrientation {
	return m.supportedOrientations
}

// SetSupportedOrientations is set_SupportedOrientations.
//
// It gained an error result in Foundation 48 and the reason is worth stating:
// until then the stored value reached nothing, so the member was correctly
// classified as pure managed state on both accessors. It now pushes to CNA's
// manager like its eight neighbours, so the setter can be refused and says so
// while the getter still cannot fail.
func (m *GraphicsDeviceManager) SetSupportedOrientations(value DisplayOrientation) error {
	m.supportedOrientations = value
	m.isDeviceDirty = true
	return m.push(func(resource *interop.Resource) error {
		return interop.ManagerSetSupportedOrientations(resource, uint32(value))
	})
}

// ApplyChanges is GraphicsDeviceManager::ApplyChanges:
//
//	if (this.device != null && !this.isDeviceDirty) return;
//	this.ChangeDevice(false);
//
// The guard is not re-implemented here over state this object does not hold.
// CNA's manager is an XNA reimplementation and carries the same guard over the
// same two facts, and every setter above pushes to it -- so its `isDeviceDirty`
// is raised exactly when this one is, and its device field is the real one.
func (m *GraphicsDeviceManager) ApplyChanges() error {
	if m == nil || m.resource == nil {
		return errors.New("GraphicsDeviceManager is nil or uninitialized")
	}
	return interop.ManagerApplyChanges(m.resource)
}

// ToggleFullScreen is GraphicsDeviceManager::ToggleFullScreen:
//
//	this.IsFullScreen = !this.IsFullScreen;
//	this.ChangeDevice(false);
//
// It goes through the projected SETTER rather than the field, exactly as the
// reference's `call set_IsFullScreen` does, so the store, the dirty flag and
// the push to CNA all happen. cna_graphics_device_manager_toggle_full_screen
// is deliberately NOT bound: it would flip CNA's own flag a second time, after
// the setter had already pushed the flipped value.
func (m *GraphicsDeviceManager) ToggleFullScreen() error {
	if m == nil || m.resource == nil {
		return errors.New("GraphicsDeviceManager is nil or uninitialized")
	}
	if err := m.SetIsFullScreen(!m.IsFullScreen()); err != nil {
		return err
	}
	return interop.ManagerApplyChanges(m.resource)
}

// Dispose is GraphicsDeviceManager::Dispose(bool), whose whole body is behind
// `if (disposing)` and whose first act is to unregister the two services it
// registered -- and only when the registration is still its own:
//
//	if (game != null) {
//	    if (game.Services.GetService(typeof(IGraphicsDeviceService)) == this)
//	        game.Services.RemoveService(typeof(IGraphicsDeviceService));
//	    ... the same for IGraphicsDeviceManager ...
//	}
//	... dispose the device, raise Disposed ...
//
// The native handle is retained if CNA refuses destruction, allowing an
// owner-thread retry.
func (m *GraphicsDeviceManager) Dispose(disposing bool) error {
	if m == nil || m.resource == nil {
		return nil
	}
	if !disposing {
		return nil
	}
	if m.game != nil {
		_ = servicebridge.UnpublishDeviceService(m.game.Services(), m)
		if registered, _ := m.game.Services().GetService(graphicsDeviceManagerServiceType); registered == any(m) {
			_ = m.game.Services().RemoveService(graphicsDeviceManagerServiceType)
		}
	}
	// The registrations are released BEFORE the handle is destroyed, and the
	// Disposed event is raised from HERE rather than from a native signal.
	// Both halves of that are the reference's own shape and one of them is
	// also an upstream crash avoided.
	//
	// The reference raises Disposed from the end of Dispose(bool) and from
	// nowhere else -- the field is private and the type declares no protected
	// raiser for it -- which is exactly the situation Foundation 39 settled for
	// Game.Disposed. So the native disposal signal is bound, delivered and
	// counted, and raises nothing public: it is a LIFECYCLE_ONLY signal.
	//
	// Releasing first is what makes that unambiguous. CNA may raise its own
	// disposal signal from destruction, and a registration still installed
	// would turn one managed raise into two.
	//
	// cna_graphics_device_manager_dispose is deliberately NOT called, and the
	// first reason is sufficient on its own: the event that call exists to
	// raise is raised HERE, from the reference's own raise site, so calling it
	// would produce a second Disposed nobody asked for.
	//
	// It was also observed to SEGFAULT once, when a stress game disposed its
	// manager from UnloadContent during the run teardown. That is recorded as
	// an observation rather than as an upstream defect: a standalone C
	// reproduction was attempted in four shapes -- from unload_content, from
	// exiting, after run returned, and with the manager still alive across
	// cna_game_destroy -- and none of them crashed, so the trigger is not
	// understood and CNA is not accused of it.
	//
	// The route is therefore not bound either. It was, briefly, and the
	// binding outlived its only call site; the reachability check in
	// tools/native_abi named it, because a bound route nothing calls reports a
	// boundary wider than the one that is measured.
	releaseErr := m.signals.Release()
	m.signals = nil
	// The device facade's own registrations, if a consumer ever listened to
	// one. CNA requires every graphics-device registration released before
	// cna_game_destroy succeeds, and the facade is a Graphics-package object a
	// framework-package type cannot name -- so the release crosses the same
	// bridge the facade itself does.
	if facade, _, cached := servicebridge.ReadManagerDeviceFacade(m); cached {
		if err := servicebridge.ReleaseDeviceFacadeSignals(facade); err != nil && releaseErr == nil {
			releaseErr = err
		}
	}
	if err := m.resource.Dispose(); err != nil {
		return err
	}
	interop.UnregisterOwner(m)
	raiseErr := m.disposed.Raise(m, EventArgsEmpty())
	if releaseErr != nil {
		return releaseErr
	}
	return raiseErr
}
