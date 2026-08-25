package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// deviceService is a test-only conformer, exactly the shape an external
// package must be able to write: a stored device reference behind the one
// projected accessor, and private EventSource fields behind the four event
// pairs. Only the declaring type can raise; a consumer holding the contract
// can only subscribe.
type deviceService struct {
	device *GraphicsDevice

	deviceCreated   framework.EventSource[*framework.EventArgs]
	deviceDisposing framework.EventSource[*framework.EventArgs]
	deviceReset     framework.EventSource[*framework.EventArgs]
	deviceResetting framework.EventSource[*framework.EventArgs]
}

func (s *deviceService) GraphicsDevice() *GraphicsDevice { return s.device }

func (s *deviceService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.deviceCreated.Add(h)
}
func (s *deviceService) RemoveDeviceCreatedHandler(sub framework.EventSubscription) error {
	return s.deviceCreated.Remove(sub)
}
func (s *deviceService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.deviceDisposing.Add(h)
}
func (s *deviceService) RemoveDeviceDisposingHandler(sub framework.EventSubscription) error {
	return s.deviceDisposing.Remove(sub)
}
func (s *deviceService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.deviceReset.Add(h)
}
func (s *deviceService) RemoveDeviceResetHandler(sub framework.EventSubscription) error {
	return s.deviceReset.Remove(sub)
}
func (s *deviceService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.deviceResetting.Add(h)
}
func (s *deviceService) RemoveDeviceResettingHandler(sub framework.EventSubscription) error {
	return s.deviceResetting.Remove(sub)
}

var _ IGraphicsDeviceService = (*deviceService)(nil)

func TestGraphicsDeviceServiceAccessorIsInfallible(t *testing.T) {
	// The one non-event operation returns a value and no error, because the
	// reference implementor's get_GraphicsDevice is `ldarg.0; ldfld device;
	// ret`. A synthetic error here would be a fallibility defect, not caution.
	var service IGraphicsDeviceService = &deviceService{}
	if service.GraphicsDevice() != nil {
		t.Fatal("a service with no device must report nil, as the reference returns null")
	}
	device := &GraphicsDevice{}
	service = &deviceService{device: device}
	if service.GraphicsDevice() != device {
		t.Fatal("the accessor must hand over the stored reference")
	}
}

func TestGraphicsDeviceServiceRaisesItsOwnEvents(t *testing.T) {
	service := &deviceService{}
	var contract IGraphicsDeviceService = service

	var order []string
	record := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, args *framework.EventArgs) error {
			if sender != any(service) || args != framework.EventArgsEmpty() {
				order = append(order, name+":identity-lost")
				return nil
			}
			order = append(order, name)
			return nil
		}
	}
	created, err := contract.AddDeviceCreatedHandler(record("created"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contract.AddDeviceResettingHandler(record("resetting")); err != nil {
		t.Fatal(err)
	}
	if _, err := contract.AddDeviceResetHandler(record("reset")); err != nil {
		t.Fatal(err)
	}
	if _, err := contract.AddDeviceDisposingHandler(record("disposing")); err != nil {
		t.Fatal(err)
	}

	// Only the declaring type can raise. A consumer holding the contract has
	// no way to reach these.
	for _, raise := range []func() error{
		func() error { return service.deviceCreated.Raise(service, framework.EventArgsEmpty()) },
		func() error { return service.deviceResetting.Raise(service, framework.EventArgsEmpty()) },
		func() error { return service.deviceReset.Raise(service, framework.EventArgsEmpty()) },
		func() error { return service.deviceDisposing.Raise(service, framework.EventArgsEmpty()) },
	} {
		if err := raise(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"created", "resetting", "reset", "disposing"}
	if len(order) != len(want) {
		t.Fatalf("events = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("events = %v, want %v", order, want)
		}
	}

	// The four events are independent registration lists.
	if err := contract.RemoveDeviceCreatedHandler(created); err != nil {
		t.Fatal(err)
	}
	order = nil
	if err := service.deviceCreated.Raise(service, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if err := service.deviceReset.Raise(service, framework.EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "reset" {
		t.Fatalf("after removing the created handler, events = %v, want [reset]", order)
	}
}

func TestGraphicsDeviceServiceEventFailurePropagates(t *testing.T) {
	service := &deviceService{}
	boom := errors.New("handler failed")
	if _, err := service.AddDeviceResetHandler(
		func(any, *framework.EventArgs) error { return boom }); err != nil {
		t.Fatal(err)
	}
	if err := service.deviceReset.Raise(service, framework.EventArgsEmpty()); !errors.Is(err, boom) {
		t.Fatalf("raise = %v, want the handler failure", err)
	}
}

func TestGraphicsDeviceServiceAccessorsFollowTheSettledEventProjection(t *testing.T) {
	service := &deviceService{}
	// A nil handler registers nothing and returns the zero token; an absent
	// token is harmless.
	token, err := service.AddDeviceCreatedHandler(nil)
	if err != nil {
		t.Fatal(err)
	}
	if (token != framework.EventSubscription{}) {
		t.Fatal("a nil handler must return the zero token")
	}
	if err := service.RemoveDeviceCreatedHandler(token); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveDeviceCreatedHandler(framework.EventSubscription{}); err != nil {
		t.Fatal(err)
	}
}
