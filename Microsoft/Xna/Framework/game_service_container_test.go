package framework

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type serviceProbe interface{ Ping() int32 }

type serviceProbeImpl struct{ value int32 }

func (p *serviceProbeImpl) Ping() int32 { return p.value }

type unrelatedProbe struct{}

var serviceProbeType = reflect.TypeOf((*serviceProbe)(nil)).Elem()

// TestGameServiceContainerConstructorIsInfallible pins that construction only
// allocates the backing dictionary and starts empty.
func TestGameServiceContainerConstructorIsInfallible(t *testing.T) {
	container := NewGameServiceContainer()
	if container == nil {
		t.Fatal("NewGameServiceContainer returned nil")
	}
	provider, err := container.GetService(serviceProbeType)
	if err != nil || provider != nil {
		t.Fatalf("fresh container returned %v, %v", provider, err)
	}
}

// TestGameServiceContainerAddServiceValidatesInReferenceOrder pins all four
// checks and, critically, that the duplicate check runs **before** the
// assignability check.
func TestGameServiceContainerAddServiceValidatesInReferenceOrder(t *testing.T) {
	container := NewGameServiceContainer()

	if err := container.AddService(nil, &serviceProbeImpl{}); !errors.Is(err, errServiceArgumentNull) {
		t.Fatalf("nil service type = %v", err)
	}
	if err := container.AddService(serviceProbeType, nil); !errors.Is(err, errServiceArgumentNull) {
		t.Fatalf("nil provider = %v", err)
	}
	// Not assignable: an unrelated type cannot satisfy the interface.
	if err := container.AddService(serviceProbeType, &unrelatedProbe{}); !errors.Is(err, errServiceArgument) {
		t.Fatalf("unassignable provider = %v", err)
	}
	if err := container.AddService(serviceProbeType, &serviceProbeImpl{value: 7}); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	// A second registration of the same type is a duplicate.
	if err := container.AddService(serviceProbeType, &serviceProbeImpl{value: 8}); !errors.Is(err, errServiceArgument) {
		t.Fatalf("duplicate = %v", err)
	}
	// The reference checks the duplicate before assignability, so registering
	// an unassignable provider for an already-registered type reports the
	// duplicate message rather than the mismatch.
	err := container.AddService(serviceProbeType, &unrelatedProbe{})
	if !errors.Is(err, errServiceArgument) {
		t.Fatalf("duplicate-and-unassignable = %v", err)
	}
	if got := err.Error(); !strings.Contains(got, serviceAlreadyPresent) {
		t.Fatalf("duplicate-and-unassignable reported the wrong check: %q", got)
	}
	// The first registration survived every rejection.
	provider, getErr := container.GetService(serviceProbeType)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if provider.(serviceProbe).Ping() != 7 {
		t.Fatal("a rejected registration replaced the stored provider")
	}
}

// TestGameServiceContainerAcceptsConcreteAndInterfaceKeys pins that the
// assignability rule is the CLR's `type.IsAssignableFrom(provider.GetType())`,
// so both an interface key and the provider's own concrete type work.
func TestGameServiceContainerAcceptsConcreteAndInterfaceKeys(t *testing.T) {
	container := NewGameServiceContainer()
	provider := &serviceProbeImpl{value: 3}
	concreteType := reflect.TypeOf(provider)

	if err := container.AddService(concreteType, provider); err != nil {
		t.Fatalf("concrete key = %v", err)
	}
	if err := container.AddService(serviceProbeType, provider); err != nil {
		t.Fatalf("interface key = %v", err)
	}
	// One provider under two distinct keys is two registrations, not a
	// duplicate: the dictionary is keyed by service type.
	byConcrete, _ := container.GetService(concreteType)
	byInterface, _ := container.GetService(serviceProbeType)
	if byConcrete != any(provider) || byInterface != any(provider) {
		t.Fatal("the two keys did not both resolve to the provider")
	}
}

// TestGameServiceContainerGetServiceMissIsNotAFailure pins the reference's
// `ldnull; ret`: an unregistered type yields nil with no error.
func TestGameServiceContainerGetServiceMissIsNotAFailure(t *testing.T) {
	container := NewGameServiceContainer()
	if _, err := container.GetService(nil); !errors.Is(err, errServiceArgumentNull) {
		t.Fatalf("nil lookup = %v", err)
	}
	provider, err := container.GetService(reflect.TypeOf(&unrelatedProbe{}))
	if err != nil {
		t.Fatalf("missing service reported an error: %v", err)
	}
	if provider != nil {
		t.Fatalf("missing service returned %v", provider)
	}
}

// TestGameServiceContainerRemoveServiceIsForgiving pins that removing a
// service that was never registered is not an error: the reference discards
// the dictionary's Boolean result.
func TestGameServiceContainerRemoveServiceIsForgiving(t *testing.T) {
	container := NewGameServiceContainer()
	if err := container.RemoveService(nil); !errors.Is(err, errServiceArgumentNull) {
		t.Fatalf("nil removal = %v", err)
	}
	if err := container.RemoveService(serviceProbeType); err != nil {
		t.Fatalf("removing an absent service = %v", err)
	}

	provider := &serviceProbeImpl{value: 5}
	if err := container.AddService(serviceProbeType, provider); err != nil {
		t.Fatal(err)
	}
	if err := container.RemoveService(serviceProbeType); err != nil {
		t.Fatalf("RemoveService = %v", err)
	}
	got, err := container.GetService(serviceProbeType)
	if err != nil || got != nil {
		t.Fatalf("after removal = %v, %v", got, err)
	}
	// Removing twice is still not an error, and the type can be re-registered.
	if err := container.RemoveService(serviceProbeType); err != nil {
		t.Fatalf("second removal = %v", err)
	}
	if err := container.AddService(serviceProbeType, provider); err != nil {
		t.Fatalf("re-registration after removal = %v", err)
	}
}

// TestGameServiceContainerKeepsReferenceSemantics pins the CLR reference
// behavior: two variables naming one container share registrations.
func TestGameServiceContainerKeepsReferenceSemantics(t *testing.T) {
	container := NewGameServiceContainer()
	alias := container
	if err := alias.AddService(serviceProbeType, &serviceProbeImpl{value: 11}); err != nil {
		t.Fatal(err)
	}
	provider, err := container.GetService(serviceProbeType)
	if err != nil || provider == nil {
		t.Fatalf("aliased registration was not observed: %v, %v", provider, err)
	}
	if provider.(serviceProbe).Ping() != 11 {
		t.Fatal("the aliased registration carried the wrong provider")
	}
	other := NewGameServiceContainer()
	if got, _ := other.GetService(serviceProbeType); got != nil {
		t.Fatal("a separately constructed container shares registrations")
	}
}
