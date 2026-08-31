package graphics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// deviceServiceStub is a consumer's own IGraphicsDeviceService. Only a consumer
// can supply one: CNA-Go publishes none, because GraphicsDeviceManager is a
// partial native-backed facade satisfying neither service contract.
type deviceServiceStub struct {
	device    *GraphicsDevice
	created   framework.EventSource[*framework.EventArgs]
	disposing framework.EventSource[*framework.EventArgs]
	reset     framework.EventSource[*framework.EventArgs]
	resetting framework.EventSource[*framework.EventArgs]
}

func (s *deviceServiceStub) GraphicsDevice() *GraphicsDevice { return s.device }

func (s *deviceServiceStub) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.created.Add(h)
}
func (s *deviceServiceStub) RemoveDeviceCreatedHandler(t framework.EventSubscription) error {
	return s.created.Remove(t)
}
func (s *deviceServiceStub) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.disposing.Add(h)
}
func (s *deviceServiceStub) RemoveDeviceDisposingHandler(t framework.EventSubscription) error {
	return s.disposing.Remove(t)
}
func (s *deviceServiceStub) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.reset.Add(h)
}
func (s *deviceServiceStub) RemoveDeviceResetHandler(t framework.EventSubscription) error {
	return s.reset.Remove(t)
}
func (s *deviceServiceStub) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resetting.Add(h)
}
func (s *deviceServiceStub) RemoveDeviceResettingHandler(t framework.EventSubscription) error {
	return s.resetting.Remove(t)
}

var _ IGraphicsDeviceService = (*deviceServiceStub)(nil)

type deviceCallbacks struct{}

func (deviceCallbacks) Initialize(*framework.Game) error                 { return nil }
func (deviceCallbacks) LoadContent(*framework.Game) error                { return nil }
func (deviceCallbacks) Update(*framework.Game, framework.GameTime) error { return nil }
func (deviceCallbacks) Draw(*framework.Game, framework.GameTime) error   { return nil }
func (deviceCallbacks) UnloadContent(*framework.Game) error              { return nil }

func newDeviceGame(t *testing.T) *framework.Game {
	t.Helper()
	game, err := framework.NewGame(deviceCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return game
}

// TestGameGraphicsDeviceReportsTheReferenceFailureWithNoService pins the branch
// CNA-Go is always in until a consumer registers a service: the cached
// graphicsDeviceService field is never assigned, the resolution finds nothing,
// and the reference throws InvalidOperationException with a specific message.
func TestGameGraphicsDeviceReportsTheReferenceFailureWithNoService(t *testing.T) {
	game := newDeviceGame(t)
	device, err := GameGraphicsDevice(game)
	if err == nil {
		t.Fatal("GameGraphicsDevice reported no error with no registered service")
	}
	if device != nil {
		t.Fatal("GameGraphicsDevice returned a device with no registered service")
	}
	if !strings.Contains(err.Error(), "This property requires a graphics device service in the game service container.") {
		t.Fatalf("GameGraphicsDevice = %v, want the reference's NoGraphicsDeviceService message", err)
	}
}

// TestGameGraphicsDeviceResolvesAConsumersService is the fallback the reference
// body has and the reason this member is reachable at all: with a service in
// Game.Services, the member answers that service's device, unchanged.
func TestGameGraphicsDeviceResolvesAConsumersService(t *testing.T) {
	game := newDeviceGame(t)
	published := &GraphicsDevice{}
	service := &deviceServiceStub{device: published}
	if err := game.Services().AddService(graphicsDeviceServiceType, service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	device, err := GameGraphicsDevice(game)
	if err != nil {
		t.Fatalf("GameGraphicsDevice: %v", err)
	}
	if device != published {
		t.Fatal("GameGraphicsDevice returned a device other than the service's own")
	}

	// A service publishing no device answers nil, with no error. The reference
	// forwards get_GraphicsDevice unchanged and does not second-guess it: the
	// null check it performs is on the SERVICE, not on the device.
	service.device = nil
	device, err = GameGraphicsDevice(game)
	if err != nil {
		t.Fatalf("GameGraphicsDevice with a device-less service: %v", err)
	}
	if device != nil {
		t.Fatal("GameGraphicsDevice invented a device for a service that publishes none")
	}
}

// TestGameGraphicsDeviceRefusesAnUnconstructedGame covers the Go-only guard.
// A Game whose constructor never ran has no Services container to resolve out
// of, which is a state CLR cannot produce.
func TestGameGraphicsDeviceRefusesAnUnconstructedGame(t *testing.T) {
	for name, game := range map[string]*framework.Game{
		"nil":           nil,
		"unconstructed": {},
	} {
		device, err := GameGraphicsDevice(game)
		if err == nil {
			t.Fatalf("GameGraphicsDevice on a %s Game reported no error", name)
		}
		if device != nil {
			t.Fatalf("GameGraphicsDevice on a %s Game returned a device", name)
		}
	}
}

// TestGameGraphicsDeviceIsTheCrossPackageProjection records why this member is
// a package function rather than a method on Game: the framework package cannot
// name GraphicsDevice, because the Graphics package imports it.
func TestGameGraphicsDeviceIsTheCrossPackageProjection(t *testing.T) {
	var _ func(*framework.Game) (*GraphicsDevice, error) = GameGraphicsDevice
	if _, ok := reflect.TypeOf((*framework.Game)(nil)).MethodByName("GraphicsDevice"); ok {
		t.Fatal("Game declares a GraphicsDevice method; the cross-package rule projects it into the Graphics package")
	}
	// The resolution uses the projected contract's own type token, which is
	// what a consumer registers under.
	if graphicsDeviceServiceType != reflect.TypeOf((*IGraphicsDeviceService)(nil)).Elem() {
		t.Fatal("the service token is not the projected IGraphicsDeviceService contract")
	}
	if !errors.Is(errNoGraphicsDeviceService, errNoGraphicsDeviceService) {
		t.Fatal("the sentinel is not comparable")
	}
}
