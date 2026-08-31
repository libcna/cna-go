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

import "sync"

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

var (
	mu       sync.RWMutex
	resolver DeviceServiceResolver
	reader   ComponentServiceReader
)

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
