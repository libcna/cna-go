package graphics

import (
	"errors"
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file registers a GraphicsDeviceManager into a Game's service container
// under Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService, which is what
// the reference's constructor does and what makes Game.GraphicsDevice and
// DrawableGameComponent.Initialize work without a consumer supplying a service
// of their own.
//
// # Why the registration is an adapter rather than the manager
//
// The manager is a framework-package type and the contract's GraphicsDevice
// accessor returns a Graphics-package type, so the manager cannot IMPLEMENT the
// interface -- naming the return type would be an import cycle. The reference
// registers `this`; CNA-Go registers a small adapter over the manager.
//
// That is the one observable difference, and it is bounded: a consumer that
// resolves the service gets an object whose GraphicsDevice, DeviceCreated,
// DeviceResetting, DeviceReset and DeviceDisposing are the manager's own, and
// only reference EQUALITY with the manager differs. The adapter is created once
// per manager and stored in the container, so the identity a consumer observes
// is stable across every GetService.

// managerDeviceService is the adapter. Every member forwards; it holds no state
// of its own beyond the manager it was built for.
type managerDeviceService struct {
	manager *framework.GraphicsDeviceManager
}

// GraphicsDevice is IGraphicsDeviceService::GraphicsDevice, forwarded to the
// manager's own projected accessor.
//
// The contract's accessor is infallible and the manager's is not -- it reaches
// a live native manager and can be refused -- so a refusal answers nil here,
// which is exactly what the reference's own property does before a device
// exists. A consumer that wants the failure calls the manager's accessor.
func (s *managerDeviceService) GraphicsDevice() *GraphicsDevice {
	device, err := GraphicsDeviceManagerGraphicsDevice(s.manager)
	if err != nil {
		return nil
	}
	return device
}

func (s *managerDeviceService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.manager.AddDeviceCreatedHandler(h)
}

func (s *managerDeviceService) RemoveDeviceCreatedHandler(t framework.EventSubscription) error {
	return s.manager.RemoveDeviceCreatedHandler(t)
}

func (s *managerDeviceService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.manager.AddDeviceDisposingHandler(h)
}

func (s *managerDeviceService) RemoveDeviceDisposingHandler(t framework.EventSubscription) error {
	return s.manager.RemoveDeviceDisposingHandler(t)
}

func (s *managerDeviceService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.manager.AddDeviceResetHandler(h)
}

func (s *managerDeviceService) RemoveDeviceResetHandler(t framework.EventSubscription) error {
	return s.manager.RemoveDeviceResetHandler(t)
}

func (s *managerDeviceService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.manager.AddDeviceResettingHandler(h)
}

func (s *managerDeviceService) RemoveDeviceResettingHandler(t framework.EventSubscription) error {
	return s.manager.RemoveDeviceResettingHandler(t)
}

// The adapter is an IGraphicsDeviceService by construction, and this line is
// what makes that a compile error rather than a runtime surprise.
var _ IGraphicsDeviceService = (*managerDeviceService)(nil)

// init installs the publisher pair the framework package's constructor and
// Dispose call.
func init() {
	servicebridge.SetDeviceServicePublisher(
		func(services any, manager any) error {
			container, ok := services.(*framework.GameServiceContainer)
			if !ok || container == nil {
				return errors.New("service container is not a GameServiceContainer")
			}
			typed, ok := manager.(*framework.GraphicsDeviceManager)
			if !ok || typed == nil {
				return errors.New("manager is not a GraphicsDeviceManager")
			}
			return container.AddService(graphicsDeviceServiceType, &managerDeviceService{manager: typed})
		},
		func(services any, manager any) error {
			container, ok := services.(*framework.GameServiceContainer)
			if !ok || container == nil {
				return nil
			}
			typed, ok := manager.(*framework.GraphicsDeviceManager)
			if !ok || typed == nil {
				return nil
			}
			// The reference removes the registration only when it is still its
			// own: `if (GetService(...) == this) RemoveService(...)`. The
			// equivalent question here is whether the registered adapter was
			// built for THIS manager.
			registered, err := container.GetService(graphicsDeviceServiceType)
			if err != nil || registered == nil {
				return nil
			}
			adapter, ok := registered.(*managerDeviceService)
			if !ok || adapter.manager != typed {
				return nil
			}
			return container.RemoveService(graphicsDeviceServiceType)
		})
}
