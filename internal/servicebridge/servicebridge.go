// Package servicebridge is the one place the framework package and the
// Graphics package agree on something neither can name in the other.
//
// # Why it exists
//
// DrawableGameComponent::Initialize is declared on a type in
// Microsoft.Xna.Framework, and its body resolves
// Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService out of Game.Services:
//
//	this.deviceService = this.Game.Services.GetService(typeof(IGraphicsDeviceService))
//	                     as IGraphicsDeviceService;
//
// In the CLR both namespaces live in assemblies that reference each other's
// direction freely. In Go the Graphics package imports the framework package,
// so the framework package cannot name IGraphicsDeviceService, cannot build the
// reflect.Type token GetService is keyed by, and cannot spell the return type
// of the service's GraphicsDevice accessor.
//
// The settled cross-package cycle rule handles the MEMBER side of that -- a
// device-typed member projects into the descendant package -- but a private
// field resolution inside a framework-declared method body is not a member and
// the rule does not reach it.
//
// # What it is, and what it deliberately is not
//
// It is two function values, each installed once from a package init, and read
// by the other side. It is NOT a service locator, a registry of live objects,
// or a cache: nothing here holds a Game, a component, a device or a service
// beyond the duration of one call, so nothing here can keep an object alive.
//
// It adds no public API. Both halves are internal, so no XNA identity, no
// exported type and no exported function is created by this file, and
// tools/api_compat's unexpected-member scan sees nothing new.
//
// # Why "no resolver installed" is the correct answer rather than a gap
//
// The framework package does not import the Graphics package, so a consumer who
// never imports Graphics never runs its init and no resolver is installed. That
// is exactly right: to register an IGraphicsDeviceService a consumer must be
// able to NAME IGraphicsDeviceService, which requires importing the Graphics
// package. A program that cannot have registered the service resolves nothing,
// and DrawableGameComponent::Initialize reports the reference's own
// InvalidOperationException. The absence is correct by construction rather than
// by a runtime check.
package servicebridge

import (
	"errors"
	"sync"
)

// DeviceServiceResolver answers what the framework package cannot ask.
//
// services is a *framework.GameServiceContainer, passed as any because this
// package cannot import the framework package either -- the framework package
// imports THIS one, and the cycle would be immediate.
//
// The result is the registered service as any, plus a closure that reports
// whether it currently publishes a device. The closure exists because
// Initialize's last branch is
//
//	if (this.deviceService.GraphicsDevice != null) this.LoadContent();
//
// and the framework package can neither name GraphicsDevice nor compare one
// with nil. It is a closure rather than a captured bool because the device is
// live state that a service may publish later.
type DeviceServiceResolver func(services any) (service any, hasDevice func() bool, ok bool)

// ComponentServiceReader answers the mirror-image question for the Graphics
// package: which service did this DrawableGameComponent resolve?
//
// component is a *framework.DrawableGameComponent. The Graphics package needs
// it because DrawableGameComponent::get_GraphicsDevice is a device-typed member
// and therefore projects into the Graphics package, while the field it reads is
// private state of a framework type.
type ComponentServiceReader func(component any) (service any, ok bool)

// ManagerConfigurationSlot names one GraphicsDeviceManager configuration value
// whose Go enum type lives in the Graphics package.
//
// Three of the nine configuration properties are typed by Graphics-package
// enums -- GraphicsProfile, SurfaceFormat and DepthFormat -- so the settled
// cross-package cycle rule projects those MEMBERS into the Graphics package
// while their VALUES stay managed state on the framework-package object. The
// framework package holds them as the raw int32 the CLR enums are; these slots
// are how the Graphics package reads and writes them.
//
// The slot is an identity rather than a field name because a name would be a
// string the compiler cannot check.
type ManagerConfigurationSlot int

const (
	ManagerGraphicsProfile ManagerConfigurationSlot = iota
	ManagerPreferredBackBufferFormat
	ManagerPreferredDepthStencilFormat
)

// ManagerConfigurationReader reads one slot from a
// *framework.GraphicsDeviceManager, passed as any for the usual reason.
type ManagerConfigurationReader func(manager any, slot ManagerConfigurationSlot) (int32, bool)

// ManagerConfigurationWriter writes one slot, performing the same store,
// dirty-flag raise and native push the framework-package setters do.
type ManagerConfigurationWriter func(manager any, slot ManagerConfigurationSlot, value int32) error

// DeviceServicePublisher registers a GraphicsDeviceManager into a Game's
// service container under Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService.
//
// The framework package cannot do it itself for two reasons, and only the
// second is obvious: it cannot build the reflect.Type token, and the manager
// cannot IMPLEMENT the contract at all, because the contract's GraphicsDevice
// accessor returns a Graphics-package type. So the Graphics package registers
// a small adapter over the manager rather than the manager itself, which is the
// one observable difference from the reference and is recorded as such.
//
// unregister removes it again, and only when the registration is still the one
// this manager published -- the reference's Dispose checks the same thing with
// `GetService(...) == this`.
type DeviceServicePublisher func(services any, manager any) error
type DeviceServiceUnpublisher func(services any, manager any) error

var (
	mu            sync.RWMutex
	resolver      DeviceServiceResolver
	reader        ComponentServiceReader
	managerReader ManagerConfigurationReader
	managerWriter ManagerConfigurationWriter
	publisher     DeviceServicePublisher
	unpublisher   DeviceServiceUnpublisher
	facadeReader  ManagerDeviceFacadeReader
	facadeWriter  ManagerDeviceFacadeWriter
	signalReader  ManagerSignalReader
)

// ManagerSignalReader reports how many times each canonical GraphicsDeviceManager
// signal has been delivered to one manager.
//
// It exists because a signal that raises a consumer event leaves no other
// trace to count, and it lives HERE rather than on the manager because a
// counter is not part of the XNA contract: an exported accessor for it would
// be an UNEXPECTED_MEMBER, which is exactly what the verifier said when it was
// tried. Only tools inside the module can reach this package.
type ManagerSignalReader func(manager any) ([]int, bool)

// ManagerDeviceFacadeReader and ManagerDeviceFacadeWriter carry the ONE
// GraphicsDevice facade a manager hands out.
//
// GraphicsDeviceManager::device is a field in the reference, so repeated reads
// of GraphicsDevice return the same object -- the same identity property
// Game.Window has, and for the same reason: a consumer compares it, stores it
// and passes it around. The facade type lives in the Graphics package and the
// field on a framework-package object, so neither side can hold it alone.
//
// The generation travels with it because a facade outlives nothing: each Run
// gets a new one, and a facade cached across that boundary would answer a
// stale generation forever instead of being replaced as the reference replaces
// its field.
type ManagerDeviceFacadeReader func(manager any) (facade any, generation uint64, ok bool)
type ManagerDeviceFacadeWriter func(manager any, facade any, generation uint64)

// SetDeviceServiceResolver installs the Graphics package's resolver. It is
// called once, from that package's init, and installing a second one is a
// programmer error rather than a runtime condition: there is exactly one
// Graphics package.
func SetDeviceServiceResolver(value DeviceServiceResolver) {
	mu.Lock()
	defer mu.Unlock()
	resolver = value
}

// ResolveDeviceService runs the installed resolver, or reports that there is
// none. A program that never imported the Graphics package takes the second
// path, which is indistinguishable -- correctly -- from one whose service
// container holds no IGraphicsDeviceService.
func ResolveDeviceService(services any) (any, func() bool, bool) {
	mu.RLock()
	current := resolver
	mu.RUnlock()
	if current == nil {
		return nil, nil, false
	}
	return current(services)
}

// SetComponentServiceReader installs the framework package's reader, from that
// package's init.
func SetComponentServiceReader(value ComponentServiceReader) {
	mu.Lock()
	defer mu.Unlock()
	reader = value
}

// ComponentService runs the installed reader, or reports that there is none.
func ComponentService(component any) (any, bool) {
	mu.RLock()
	current := reader
	mu.RUnlock()
	if current == nil {
		return nil, false
	}
	return current(component)
}

// SetManagerConfigurationAccessors installs the framework package's reader and
// writer, from that package's init.
func SetManagerConfigurationAccessors(read ManagerConfigurationReader, write ManagerConfigurationWriter) {
	mu.Lock()
	defer mu.Unlock()
	managerReader, managerWriter = read, write
}

// ReadManagerConfiguration reads one slot, or reports that no reader is
// installed. The framework package always installs one, so a false result here
// means the value was not a GraphicsDeviceManager.
func ReadManagerConfiguration(manager any, slot ManagerConfigurationSlot) (int32, bool) {
	mu.RLock()
	current := managerReader
	mu.RUnlock()
	if current == nil {
		return 0, false
	}
	return current(manager, slot)
}

// WriteManagerConfiguration writes one slot.
func WriteManagerConfiguration(manager any, slot ManagerConfigurationSlot, value int32) error {
	mu.RLock()
	current := managerWriter
	mu.RUnlock()
	if current == nil {
		return errors.New("no GraphicsDeviceManager configuration writer is installed")
	}
	return current(manager, slot, value)
}

// SetDeviceServicePublisher installs the Graphics package's publisher pair,
// from that package's init.
func SetDeviceServicePublisher(publish DeviceServicePublisher, unpublish DeviceServiceUnpublisher) {
	mu.Lock()
	defer mu.Unlock()
	publisher, unpublisher = publish, unpublish
}

// PublishDeviceService registers the manager's adapter, or does nothing when no
// publisher is installed.
//
// Doing nothing is the correct answer rather than a failure, and it is the same
// argument the resolver makes: a program that never imported the Graphics
// package can neither name IGraphicsDeviceService nor resolve one, so a
// registration nobody could look up would be invisible either way.
func PublishDeviceService(services any, manager any) error {
	mu.RLock()
	current := publisher
	mu.RUnlock()
	if current == nil {
		return nil
	}
	return current(services, manager)
}

// UnpublishDeviceService removes the registration this manager published.
func UnpublishDeviceService(services any, manager any) error {
	mu.RLock()
	current := unpublisher
	mu.RUnlock()
	if current == nil {
		return nil
	}
	return current(services, manager)
}

// SetManagerDeviceFacadeAccessors installs the framework package's pair, from
// that package's init.
func SetManagerDeviceFacadeAccessors(read ManagerDeviceFacadeReader, write ManagerDeviceFacadeWriter) {
	mu.Lock()
	defer mu.Unlock()
	facadeReader, facadeWriter = read, write
}

// ReadManagerDeviceFacade returns the cached facade and the generation it was
// built for.
func ReadManagerDeviceFacade(manager any) (any, uint64, bool) {
	mu.RLock()
	current := facadeReader
	mu.RUnlock()
	if current == nil {
		return nil, 0, false
	}
	return current(manager)
}

// WriteManagerDeviceFacade stores the facade for a generation.
func WriteManagerDeviceFacade(manager any, facade any, generation uint64) {
	mu.RLock()
	current := facadeWriter
	mu.RUnlock()
	if current == nil {
		return
	}
	current(manager, facade, generation)
}

// SetManagerSignalReader installs the framework package's reader, from that
// package's init.
func SetManagerSignalReader(read ManagerSignalReader) {
	mu.Lock()
	defer mu.Unlock()
	signalReader = read
}

// ReadManagerSignalDeliveries reports the per-identity delivery counts, or that
// there is nothing to report.
func ReadManagerSignalDeliveries(manager any) ([]int, bool) {
	mu.RLock()
	current := signalReader
	mu.RUnlock()
	if current == nil {
		return nil, false
	}
	return current(manager)
}
