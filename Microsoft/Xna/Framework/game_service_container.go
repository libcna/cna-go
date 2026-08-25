package framework

import (
	"errors"
	"fmt"
	"reflect"
)

// Sentinel errors projecting the exact CLR exceptions GameServiceContainer
// throws. They are unexported because the XNA public contract declares no
// error type here.
var (
	// errServiceArgumentNull projects System.ArgumentNullException.
	errServiceArgumentNull = errors.New("service argument is nil")
	// errServiceArgument projects System.ArgumentException.
	errServiceArgument = errors.New("service argument is invalid")
)

// The exact Resources strings the reference throw sites load, read from the
// Microsoft.Xna.Framework.Resources.resources stream of the retained
// Microsoft.Xna.Framework.Game.dll.
const (
	serviceTypeCannotBeNull     = "The service type cannot be null."
	serviceProviderCannotBeNull = "The service provider instance cannot be null."
	serviceAlreadyPresent       = "Container already contains a service of this type."
	serviceMustBeAssignable     = "Service provider object of type %s must be assignable to service type %s."
)

func serviceArgumentNullError(parameter, message string) error {
	return fmt.Errorf("%w: %s: %s", errServiceArgumentNull, parameter, message)
}

func serviceArgumentError(message string) error {
	return fmt.Errorf("%w: %s", errServiceArgument, message)
}

// GameServiceContainer is XNA's pure managed service registry: a dictionary
// keyed by service type, with no ordering, no lifetime management, and no
// runtime of its own. Nothing here creates a device, starts a game, or reaches
// native code.
//
// It is a CLR class and keeps reference semantics, so two variables naming one
// container see the same registrations.
//
// XNA declares System.IServiceProvider on it, but that interface's only member
// is GetService, which the class already declares publicly, so the contract
// adds no projected surface.
//
// Completing this type wires nothing up. CNA-Go's Game remains a partial
// native-backed facade and does not expose a Services property, so nothing in
// the binding populates or consults a container.
type GameServiceContainer struct {
	services map[reflect.Type]any
	// order preserves registration order so the container's behavior does not
	// depend on Go's randomized map iteration. The reference exposes no
	// enumeration at all, so this is invisible; it exists only to keep the
	// implementation deterministic.
	order []reflect.Type
}

// NewGameServiceContainer reproduces the constructor, whose entire body
// allocates the backing dictionary. It validates nothing and cannot fail.
func NewGameServiceContainer() *GameServiceContainer {
	return &GameServiceContainer{services: make(map[reflect.Type]any)}
}

// AddService registers one service, running the reference's four checks in
// exactly its order:
//
//  1. a nil service type reports ArgumentNullException("type")
//  2. a nil provider reports ArgumentNullException("provider")
//  3. an already-registered type reports ArgumentException — note this comes
//     **before** the assignability check, so a duplicate registration with an
//     unassignable provider reports the duplicate, not the mismatch
//  4. a provider whose runtime type is not assignable to the service type
//     reports ArgumentException
//
// The nil-provider check is CLR's `brtrue` on an object reference. Go's
// interface nil is the faithful analogue for the values Go can express; a
// non-nil interface holding a typed nil pointer has no CLR counterpart and is
// registered rather than rejected.
func (c *GameServiceContainer) AddService(serviceType reflect.Type, provider any) error {
	if serviceType == nil {
		return serviceArgumentNullError("type", serviceTypeCannotBeNull)
	}
	if provider == nil {
		return serviceArgumentNullError("provider", serviceProviderCannotBeNull)
	}
	if c.services == nil {
		c.services = make(map[reflect.Type]any)
	}
	if _, present := c.services[serviceType]; present {
		return serviceArgumentError(serviceAlreadyPresent)
	}
	providerType := reflect.TypeOf(provider)
	if !providerType.AssignableTo(serviceType) {
		return serviceArgumentError(fmt.Sprintf(serviceMustBeAssignable, providerType, serviceType))
	}
	c.services[serviceType] = provider
	c.order = append(c.order, serviceType)
	return nil
}

// RemoveService unregisters one service. It rejects a nil service type and
// otherwise discards the dictionary's result, so **removing a service that was
// never registered is not an error**.
func (c *GameServiceContainer) RemoveService(serviceType reflect.Type) error {
	if serviceType == nil {
		return serviceArgumentNullError("type", serviceTypeCannotBeNull)
	}
	if _, present := c.services[serviceType]; !present {
		return nil
	}
	delete(c.services, serviceType)
	for i, registered := range c.order {
		if registered == serviceType {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	return nil
}

// GetService looks one service up. It rejects a nil service type and otherwise
// returns the registered provider, or **nil with no error** when the type was
// never registered. A missing service is an absence, not a failure, exactly as
// the reference's `ldnull; ret` says.
func (c *GameServiceContainer) GetService(serviceType reflect.Type) (any, error) {
	if serviceType == nil {
		return nil, serviceArgumentNullError("type", serviceTypeCannotBeNull)
	}
	provider, present := c.services[serviceType]
	if !present {
		return nil, nil
	}
	return provider, nil
}
