package graphics

import (
	"errors"
	"fmt"
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// This file projects Microsoft.Xna.Framework.Game::get_GraphicsDevice.
//
// It lives in the GRAPHICS package rather than on Game because of the settled
// cross-package cycle rule: an ancestor-namespace member that returns a
// descendant-namespace type projects as a descendant-package function named
// OwnerTypeMember. Graphics imports the framework package, so the framework
// package cannot name GraphicsDevice; here, both are nameable.
//
// # The reference body, exactly
//
//	IGraphicsDeviceService service = this.graphicsDeviceService;
//	if (service == null)
//	{
//	    service = this.Services.GetService(typeof(IGraphicsDeviceService))
//	              as IGraphicsDeviceService;
//	    if (service == null)
//	        throw new InvalidOperationException(Resources.NoGraphicsDeviceService);
//	}
//	return service.GraphicsDevice;
//
// The fallback is what makes this member reachable at all. Game::graphicsDeviceService
// is a private field with exactly one assignment in the whole class, inside
// HookDeviceEvents, which base Initialize calls and which CNA-Go records as a
// deferred step -- so the cached branch is never taken here.
//
// That is not a gap being worked around. It is the state the reference itself is
// in before Initialize runs, and the reference's own answer to it is the
// resolution below. A consumer who registers an IGraphicsDeviceService into
// Game.Services gets a device from this member exactly as they would in XNA;
// one who registers nothing gets the reference's own InvalidOperationException.
//
// # Nothing is faked
//
// CNA-Go publishes no IGraphicsDeviceService of its own -- GraphicsDeviceManager
// is a partial native-backed facade that satisfies neither service contract --
// so with no consumer-supplied service this member reports the reference's
// failure rather than inventing a device. The projected device it returns when a
// service IS registered is the consumer's own, unchanged: this member resolves
// and forwards, and creates nothing.

// errNoGraphicsDeviceService projects System.InvalidOperationException, which is
// the only failure the reference body has. It is unexported because the XNA
// public contract declares no error type here.
var errNoGraphicsDeviceService = errors.New("operation is not valid")

// noGraphicsDeviceService is the exact Resources string the throw site loads,
// read from the Microsoft.Xna.Framework.Resources.resources stream of the
// retained Microsoft.Xna.Framework.Game.dll.
const noGraphicsDeviceService = "This property requires a graphics device service in the game service container."

// graphicsDeviceServiceType is the CLR type token the reference's
// `ldtoken IGraphicsDeviceService; GetService` pair resolves with. It is the Go
// projection of that contract, which is what a consumer registers under.
var graphicsDeviceServiceType = reflect.TypeOf((*IGraphicsDeviceService)(nil)).Elem()

// GameGraphicsDevice is Game::get_GraphicsDevice.
//
// It is fallible for exactly one reference reason -- no registered
// IGraphicsDeviceService -- plus the Go-only guard every projected Game member
// carries, because Go can produce a Game whose constructor never ran and which
// therefore has no Services container to resolve out of.
func GameGraphicsDevice(game *framework.Game) (*GraphicsDevice, error) {
	if game == nil {
		return nil, errors.New("Game is nil or uninitialized")
	}
	services := game.Services()
	if services == nil {
		return nil, errors.New("Game is nil or uninitialized")
	}
	// The cached graphicsDeviceService field is never assigned in CNA-Go, so
	// the reference's first branch is not taken and the resolution always runs.
	// GetService reports no failure for an absent registration: it answers a
	// nil provider, exactly as the reference's `ldnull; ret` does.
	provider, err := services.GetService(graphicsDeviceServiceType)
	if err != nil {
		return nil, err
	}
	service, ok := provider.(IGraphicsDeviceService)
	if !ok {
		// Both the absent registration and a provider that is not the service
		// arrive here, which is what `isinst` followed by the null check does.
		return nil, fmt.Errorf("%w: %s", errNoGraphicsDeviceService, noGraphicsDeviceService)
	}
	return service.GraphicsDevice(), nil
}
