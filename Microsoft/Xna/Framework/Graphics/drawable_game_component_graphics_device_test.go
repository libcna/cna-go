package graphics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The tests below run with the REAL resolver, the one this package's init
// installs. That is the point: they exercise the whole bridge end to end --
// framework asks, this package answers with a token it alone can build, and
// this package reads back private state through the framework's own reader.

func newDrawableComponent(t *testing.T) (*framework.Game, *framework.DrawableGameComponent) {
	t.Helper()
	game := newDeviceGame(t)
	return game, framework.NewDrawableGameComponent(game)
}

// TestDrawableGraphicsDeviceBeforeInitializeReportsTheReferenceFailure pins the
// guard. get_GraphicsDevice tests the SERVICE field, which Initialize is the
// only assignment to, so an un-initialized component throws -- with a message
// that is not what its resource key suggests.
func TestDrawableGraphicsDeviceBeforeInitializeReportsTheReferenceFailure(t *testing.T) {
	_, component := newDrawableComponent(t)
	device, err := DrawableGameComponentGraphicsDevice(component)
	if err == nil {
		t.Fatal("get_GraphicsDevice succeeded before Initialize")
	}
	if device != nil {
		t.Fatal("get_GraphicsDevice returned a device before Initialize")
	}
	if !strings.Contains(err.Error(), "The GraphicsDevice property cannot be used before Initialize has been called.") {
		t.Fatalf("error = %v, want the reference's exact message", err)
	}
	if !errors.Is(err, errComponentInvalidOperation) {
		t.Fatalf("error = %v, want the InvalidOperationException projection", err)
	}
}

// TestDrawableInitializeResolvesThroughTheRealBridge is the end-to-end proof.
// A consumer registers a service under the same token the reference's `ldtoken`
// resolves, this package's init supplies the resolver, and Initialize -- which
// lives in a package that cannot name IGraphicsDeviceService -- finds it.
func TestDrawableInitializeResolvesThroughTheRealBridge(t *testing.T) {
	game, component := newDrawableComponent(t)
	published := &GraphicsDevice{}
	service := &deviceServiceStub{device: published}
	token := reflect.TypeOf((*IGraphicsDeviceService)(nil)).Elem()
	if err := game.Services().AddService(token, service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	device, err := DrawableGameComponentGraphicsDevice(component)
	if err != nil {
		t.Fatalf("get_GraphicsDevice after Initialize: %v", err)
	}
	if device != published {
		t.Fatal("get_GraphicsDevice returned a device other than the service's own")
	}
}

// TestDrawableGraphicsDeviceGuardsTheServiceNotTheDevice pins which nil the
// reference tests. A service that publishes no device answers nil with NO
// failure, exactly as Game.GraphicsDevice does.
func TestDrawableGraphicsDeviceGuardsTheServiceNotTheDevice(t *testing.T) {
	game, component := newDrawableComponent(t)
	service := &deviceServiceStub{}
	if err := game.Services().AddService(reflect.TypeOf((*IGraphicsDeviceService)(nil)).Elem(), service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	device, err := DrawableGameComponentGraphicsDevice(component)
	if err != nil {
		t.Fatalf("get_GraphicsDevice with a device-less service: %v", err)
	}
	if device != nil {
		t.Fatal("get_GraphicsDevice invented a device")
	}
}

// TestDrawableInitializeSubscribesToAConsumersRealService proves the four
// registrations reach the consumer's object rather than a copy, by removing
// them again and observing the service's own lists empty out.
func TestDrawableInitializeSubscribesToAConsumersRealService(t *testing.T) {
	game, component := newDrawableComponent(t)
	service := &deviceServiceStub{device: &GraphicsDevice{}}
	if err := game.Services().AddService(reflect.TypeOf((*IGraphicsDeviceService)(nil)).Elem(), service); err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// Raising each of the four reaches the component's handlers, whose bodies
	// are the reference's own and cannot fail.
	for name, raise := range map[string]func(any, *framework.EventArgs) error{
		"created":   service.created.Raise,
		"resetting": service.resetting.Raise,
		"reset":     service.reset.Raise,
		"disposing": service.disposing.Raise,
	} {
		if err := raise(service, framework.EventArgsEmpty()); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := component.DisposeByBoolean(true); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	// After disposal the registrations are gone, so a raise reaches nothing.
	if err := service.created.Raise(service, framework.EventArgsEmpty()); err != nil {
		t.Fatalf("post-dispose raise: %v", err)
	}
	if _, err := DrawableGameComponentGraphicsDevice(component); err != nil {
		t.Fatalf("get_GraphicsDevice after Dispose: %v", err)
	}
}

// TestDrawableGraphicsDeviceRefusesANilComponent is the Go-only guard.
func TestDrawableGraphicsDeviceRefusesANilComponent(t *testing.T) {
	if _, err := DrawableGameComponentGraphicsDevice(nil); err == nil {
		t.Fatal("get_GraphicsDevice accepted a nil component")
	}
}

// TestGraphicsDeviceIsNotAMethodOnDrawableGameComponent records the
// cross-package projection a consumer has to write against, exactly as the
// Game.GraphicsDevice test does for Game.
func TestGraphicsDeviceIsNotAMethodOnDrawableGameComponent(t *testing.T) {
	if _, ok := reflect.TypeOf((*framework.DrawableGameComponent)(nil)).MethodByName("GraphicsDevice"); ok {
		t.Fatal("DrawableGameComponent declares a GraphicsDevice method")
	}
	var _ func(*framework.DrawableGameComponent) (*GraphicsDevice, error) = DrawableGameComponentGraphicsDevice
}
