package framework

import (
	"errors"
	"fmt"

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
// two dimension setters throw.
var errBackBufferDimension = errors.New("argument is out of range")

// backBufferDimMustBePositive is the exact Resources string those two throw
// sites load, read from the Microsoft.Xna.Framework.Resources.resources stream
// of the retained Microsoft.Xna.Framework.Game.dll.
const backBufferDimMustBePositive = "The back buffer dimension must be positive."

func backBufferDimensionError() error {
	return fmt.Errorf("%w: value: %s", errBackBufferDimension, backBufferDimMustBePositive)
}

// init installs the accessors the Graphics package uses to reach the three
// configuration slots whose Go enum types this package cannot name.
func init() {
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
	resource, err := game.runtime.CreateGraphicsDeviceManager()
	if err != nil {
		return nil, err
	}
	manager := newGraphicsDeviceManagerState()
	manager.runtime = game.runtime
	manager.resource = resource
	interop.RegisterOwner(manager, manager.runtime, manager.resource)
	return manager, nil
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

// Dispose releases the owned manager. The native handle is retained if CNA
// refuses destruction, allowing an owner-thread retry.
func (m *GraphicsDeviceManager) Dispose(disposing bool) error {
	_ = disposing
	if m == nil || m.resource == nil {
		return nil
	}
	if err := m.resource.Dispose(); err != nil {
		return err
	}
	interop.UnregisterOwner(m)
	return nil
}
