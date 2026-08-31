package graphics

import (
	"errors"
	"fmt"
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file projects Microsoft.Xna.Framework.DrawableGameComponent::get_GraphicsDevice,
// and installs the resolver DrawableGameComponent::Initialize needs.
//
// Both halves live here for the same reason Game.GraphicsDevice does: the
// cross-package cycle rule puts an ancestor-namespace member that returns a
// descendant-namespace type in the descendant package, named
// OwnerTypeMember. This package imports the framework one, so both
// IGraphicsDeviceService and DrawableGameComponent are nameable here and
// neither is nameable there.
//
// # The reference body, exactly
//
//	if (this.deviceService == null)
//	    throw new InvalidOperationException(Resources.PropertyCannotBeCalledBeforeInitialize);
//	return this.deviceService.GraphicsDevice;
//
// The guard is on the SERVICE, not on the device, so a component whose service
// publishes no device answers nil with no failure -- the same shape
// Game.GraphicsDevice has, and for the same reason.

// errComponentInvalidOperation projects System.InvalidOperationException, the
// only failure the reference body has.
var errComponentInvalidOperation = errors.New("operation is not valid")

// propertyCannotBeCalledBeforeInitialize is the exact Resources string the
// throw site loads, read from the Microsoft.Xna.Framework.Resources.resources
// stream of the retained Microsoft.Xna.Framework.Game.dll.
//
// The resource KEY is PropertyCannotBeCalledBeforeInitialize and the string it
// names says something narrower and more specific -- it names the property and
// says "used", not "called". The value is authority, not the key, exactly as
// it is for Resources::InactiveSleepTimeCannotBeZero.
const propertyCannotBeCalledBeforeInitialize = "The GraphicsDevice property cannot be used before Initialize has been called."

// init installs the device-service resolver the framework package cannot write.
//
// It is the whole of what internal/servicebridge exists for: the token below is
// a reflect.Type over a contract declared in THIS package, and the closure's
// `service.GraphicsDevice() != nil` is a comparison the framework package
// cannot spell. Nothing is retained -- the closure receives the container it is
// asked about and returns immediately.
//
// A program that never imports this package never runs this init, so no
// resolver exists and DrawableGameComponent::Initialize reports the reference's
// MissingGraphicsDeviceService. That is correct rather than a gap: registering
// an IGraphicsDeviceService requires naming it, which requires importing this
// package.
func init() {
	servicebridge.SetDeviceServiceResolver(func(services any) (any, func() bool, bool) {
		container, ok := services.(*framework.GameServiceContainer)
		if !ok || container == nil {
			return nil, nil, false
		}
		// GetService's own failures are argument failures on a nil type,
		// which cannot happen here; the reference's `as` cast turns a
		// registration of the wrong type into null rather than a throw, and
		// the type assertion below is that cast.
		value, err := container.GetService(graphicsDeviceServiceType)
		if err != nil || value == nil {
			return nil, nil, false
		}
		service, matched := value.(IGraphicsDeviceService)
		if !matched {
			return nil, nil, false
		}
		return service, func() bool { return service.GraphicsDevice() != nil }, true
	})
}

// DrawableGameComponentGraphicsDevice is
// DrawableGameComponent::get_GraphicsDevice.
//
// It is fallible for exactly one reference reason -- the component's service
// field is still null, which is what "before Initialize()" means -- plus the
// Go-only nil-receiver guard every projected member of a framework type
// carries.
func DrawableGameComponentGraphicsDevice(component *framework.DrawableGameComponent) (*GraphicsDevice, error) {
	if component == nil {
		return nil, errors.New("DrawableGameComponent is nil")
	}
	value, resolved := servicebridge.ComponentService(component)
	if !resolved || value == nil {
		return nil, fmt.Errorf("%w: %s", errComponentInvalidOperation, propertyCannotBeCalledBeforeInitialize)
	}
	service, matched := value.(IGraphicsDeviceService)
	if !matched {
		return nil, fmt.Errorf("%w: %s", errComponentInvalidOperation, propertyCannotBeCalledBeforeInitialize)
	}
	return service.GraphicsDevice(), nil
}
