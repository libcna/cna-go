package framework

import (
	"errors"
	"strings"
	"testing"

	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// The framework package cannot name PresentationParameters or GraphicsAdapter,
// so these tests install their own half of the device-selection bridge over
// stand-ins. That is not a weaker test: everything under test here is
// GraphicsDeviceInformation's OWN logic -- which fields, in which order, with
// which short-circuit -- and the bridge is exactly the boundary that logic
// reaches the Graphics package through. The real Graphics half is exercised in
// that package.

// fakeParameters stands in for a PresentationParameters.
type fakeParameters struct {
	snapshot servicebridge.PresentationSnapshot
}

// fakeAdapter stands in for a GraphicsAdapter, whose only role in these
// members is object identity.
type fakeAdapter struct{ name string }

// installFakeDeviceSelectionBridge wires the six hooks these members use and
// restores nothing: every test in this package that touches the bridge installs
// it, and the framework test binary links no Graphics package to conflict with.
func installFakeDeviceSelectionBridge(t *testing.T, adapter any, adapterErr error) {
	t.Helper()
	servicebridge.SetDeviceSelectionBridge(
		func() any { return &fakeParameters{} },
		func(parameters any) (any, bool) {
			typed, ok := parameters.(*fakeParameters)
			if !ok || typed == nil {
				return nil, false
			}
			// PresentationParameters::Clone is a member-wise copy, so the clone
			// is a DIFFERENT object with the same values.
			clone := *typed
			return &clone, true
		},
		func(parameters any) (servicebridge.PresentationSnapshot, bool) {
			typed, ok := parameters.(*fakeParameters)
			if !ok || typed == nil {
				return servicebridge.PresentationSnapshot{}, false
			}
			return typed.snapshot, true
		},
		func() (any, error) { return adapter, adapterErr },
		func(any) (int32, bool) { return 0, false },
		func(any, bool) ([]any, error) { return nil, nil },
		func(any, []any) {},
	)
}

func TestGraphicsDeviceInformationConstructorAllocatesAndTakesTheDefaultAdapter(t *testing.T) {
	adapter := &fakeAdapter{name: "default"}
	installFakeDeviceSelectionBridge(t, adapter, nil)

	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatalf("NewGraphicsDeviceInformation: %v", err)
	}
	if information.adapter != any(adapter) {
		t.Fatal("the constructor did not store GraphicsAdapter.DefaultAdapter")
	}
	if information.presentationParameters == nil {
		t.Fatal("the constructor did not allocate a PresentationParameters")
	}
	// The constructor does not assign graphicsProfile, so it starts at Reach,
	// which is zero.
	if information.graphicsProfile != 0 {
		t.Fatalf("graphicsProfile = %d, want the enum's zero", information.graphicsProfile)
	}
	// Two informations get two DIFFERENT parameter objects and the SAME
	// adapter.
	second, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	if second.presentationParameters == information.presentationParameters {
		t.Fatal("two informations share one PresentationParameters")
	}
	if second.adapter != information.adapter {
		t.Fatal("two informations report different default adapters")
	}
}

// TestGraphicsDeviceInformationConstructorReportsTheAdapterFailure is the
// fallibility claim: the constructor's second statement crosses into CNA, and
// a program with no live device cannot answer it.
func TestGraphicsDeviceInformationConstructorReportsTheAdapterFailure(t *testing.T) {
	planted := errors.New("no adapters")
	installFakeDeviceSelectionBridge(t, nil, planted)
	if _, err := NewGraphicsDeviceInformation(); !errors.Is(err, planted) {
		t.Fatalf("NewGraphicsDeviceInformation = %v, want the adapter failure", err)
	}
}

func TestGraphicsDeviceInformationEqualsComparesValuesNotObjects(t *testing.T) {
	adapter := &fakeAdapter{name: "a"}
	installFakeDeviceSelectionBridge(t, adapter, nil)
	left, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	// Distinct parameter objects with identical values are EQUAL: the reference
	// compares nine property values, not the objects.
	if left.presentationParameters == right.presentationParameters {
		t.Fatal("the fixture handed out one parameter object twice")
	}
	if !left.Equals(right) || !right.Equals(left) {
		t.Fatal("two informations with identical values are not equal")
	}
	if left.Equals(nil) || left.Equals("not an information") {
		t.Fatal("Equals matched something that is not a GraphicsDeviceInformation")
	}

	// Each of the eleven contributors, changed one at a time, must break it.
	mutations := map[string]func(*GraphicsDeviceInformation){
		"adapter":         func(g *GraphicsDeviceInformation) { g.adapter = &fakeAdapter{name: "b"} },
		"graphicsProfile": func(g *GraphicsDeviceInformation) { g.graphicsProfile = 1 },
		"backBufferWidth": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.BackBufferWidth = 1 })
		},
		"backBufferHeight": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.BackBufferHeight = 1 })
		},
		"backBufferFormat": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.BackBufferFormat = 1 })
		},
		"depthStencilFormat": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.DepthStencilFormat = 1 })
		},
		"multiSampleCount": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.MultiSampleCount = 1 })
		},
		"displayOrientation": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.DisplayOrientation = 1 })
		},
		"presentationInterval": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.PresentationInterval = 1 })
		},
		"renderTargetUsage": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.RenderTargetUsage = 1 })
		},
		"deviceWindowHandle": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.DeviceWindowHandle = 1 })
		},
		"isFullScreen": func(g *GraphicsDeviceInformation) {
			mutateSnapshot(g, func(s *servicebridge.PresentationSnapshot) { s.IsFullScreen = true })
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed, err := NewGraphicsDeviceInformation()
			if err != nil {
				t.Fatal(err)
			}
			mutate(changed)
			if left.Equals(changed) {
				t.Fatalf("a change to %s did not break equality", name)
			}
			// The hash must move with it, or the hash is not reading that value.
			if left.GetHashCode() == changed.GetHashCode() {
				t.Fatalf("a change to %s did not move the hash", name)
			}
		})
	}
}

func mutateSnapshot(g *GraphicsDeviceInformation, mutate func(*servicebridge.PresentationSnapshot)) {
	parameters, _ := g.presentationParameters.(*fakeParameters)
	mutate(&parameters.snapshot)
}

func TestGraphicsDeviceInformationCloneDeepCopiesParametersAndAliasesTheAdapter(t *testing.T) {
	adapter := &fakeAdapter{name: "a"}
	installFakeDeviceSelectionBridge(t, adapter, nil)
	source, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	source.graphicsProfile = 1
	mutateSnapshot(source, func(s *servicebridge.PresentationSnapshot) { s.BackBufferWidth = 1280 })

	clone, err := source.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if clone == source {
		t.Fatal("Clone returned the source")
	}
	if clone.graphicsProfile != 1 {
		t.Fatalf("clone profile = %d", clone.graphicsProfile)
	}
	// The adapter is ALIASED.
	if clone.adapter != source.adapter {
		t.Fatal("Clone copied the adapter instead of aliasing it")
	}
	// The parameters are DEEP-copied: same values, different object, and a
	// later change to one is invisible to the other.
	if clone.presentationParameters == source.presentationParameters {
		t.Fatal("Clone aliased the PresentationParameters instead of cloning it")
	}
	if !clone.Equals(source) {
		t.Fatal("a fresh clone is not equal to its source")
	}
	mutateSnapshot(clone, func(s *servicebridge.PresentationSnapshot) { s.BackBufferWidth = 640 })
	if clone.Equals(source) {
		t.Fatal("a change to the clone reached the source")
	}
}

// TestGraphicsDeviceInformationCloneCarriesTheConstructorsFailure pins the
// consequence of Clone's first instruction being `newobj .ctor()`.
func TestGraphicsDeviceInformationCloneCarriesTheConstructorsFailure(t *testing.T) {
	installFakeDeviceSelectionBridge(t, &fakeAdapter{}, nil)
	source, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	planted := errors.New("no adapters")
	installFakeDeviceSelectionBridge(t, nil, planted)
	if _, err := source.Clone(); !errors.Is(err, planted) {
		t.Fatalf("Clone = %v, want the constructor's adapter failure", err)
	}
}

// TestGraphicsDeviceInformationAdapterSetterReproducesTheReferenceBug is the
// one place a defect is copied on purpose.
//
// set_Adapter's guard tests THIS.adapter, not `value`, so a null assignment
// succeeds whenever the current adapter is non-null, and the refusal can only
// happen on an information that is already in that state.
func TestGraphicsDeviceInformationAdapterSetterReproducesTheReferenceBug(t *testing.T) {
	installFakeDeviceSelectionBridge(t, &fakeAdapter{name: "a"}, nil)
	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	// Assigning null succeeds, because the CURRENT adapter is not null.
	if err := information.writeAdapter(nil); err != nil {
		t.Fatalf("assigning nil was refused: %v", err)
	}
	if information.readAdapter() != nil {
		t.Fatal("the nil assignment did not take effect")
	}
	// And now the guard fires, on a NON-null assignment.
	err = information.writeAdapter(&fakeAdapter{name: "b"})
	if err == nil || !strings.Contains(err.Error(), "Try using GraphicsAdapter.DefaultAdapter instead.") {
		t.Fatalf("assigning a real adapter over a nil one = %v, want the reference's refusal", err)
	}
	if information.readAdapter() != nil {
		t.Fatal("the refused assignment took effect anyway")
	}
}

func TestPreparingDeviceSettingsEventArgsCarriesTheInstance(t *testing.T) {
	installFakeDeviceSelectionBridge(t, &fakeAdapter{}, nil)
	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	args := NewPreparingDeviceSettingsEventArgs(information)
	if args.GraphicsDeviceInformation() != information {
		t.Fatal("the args did not hand back the information they were given")
	}
	// The constructor validates nothing.
	if NewPreparingDeviceSettingsEventArgs(nil).GraphicsDeviceInformation() != nil {
		t.Fatal("a nil information was not stored as null")
	}
}

func TestGraphicsDeviceManagerPreparingDeviceSettingsEventRaisesInOrder(t *testing.T) {
	installFakeDeviceSelectionBridge(t, &fakeAdapter{}, nil)
	manager := &GraphicsDeviceManager{}
	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	args := NewPreparingDeviceSettingsEventArgs(information)

	var order []string
	var senders []any
	var seen []*PreparingDeviceSettingsEventArgs
	record := func(name string) EventHandler[*PreparingDeviceSettingsEventArgs] {
		return func(sender any, received *PreparingDeviceSettingsEventArgs) error {
			order = append(order, name)
			senders = append(senders, sender)
			seen = append(seen, received)
			return nil
		}
	}
	first, err := manager.AddPreparingDeviceSettingsHandler(record("first"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddPreparingDeviceSettingsHandler(record("second")); err != nil {
		t.Fatal(err)
	}
	if err := manager.OnPreparingDeviceSettings(manager, args); err != nil {
		t.Fatalf("OnPreparingDeviceSettings: %v", err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("invocation order = %v", order)
	}
	for i := range senders {
		if senders[i] != any(manager) {
			t.Fatalf("handler %d saw sender %v", i, senders[i])
		}
		if seen[i] != args {
			t.Fatalf("handler %d saw a different args object", i)
		}
	}
	// A raise with no handlers is the reference's null guard, and answers
	// nothing.
	if err := manager.RemovePreparingDeviceSettingsHandler(first); err != nil {
		t.Fatal(err)
	}
	order = nil
	if err := manager.OnPreparingDeviceSettings(nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "second" {
		t.Fatalf("after removal, order = %v", order)
	}
}

func TestGraphicsDeviceManagerCanResetDeviceNeedsBothOperands(t *testing.T) {
	installFakeDeviceSelectionBridge(t, &fakeAdapter{}, nil)
	manager := &GraphicsDeviceManager{}
	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CanResetDevice(nil); !errors.Is(err, errGraphicsDeviceInformationNil) {
		t.Fatalf("CanResetDevice(nil) = %v", err)
	}
	// With no device the reference dereferences null; CNA-Go reports it.
	if _, err := manager.CanResetDevice(information); !errors.Is(err, errNoDeviceToReset) {
		t.Fatalf("CanResetDevice with no device = %v", err)
	}
}

// TestGraphicsDeviceManagerCanResetDeviceComparesProfiles installs a bridge
// whose device reports a profile, so the comparison itself is measured.
func TestGraphicsDeviceManagerCanResetDeviceComparesProfiles(t *testing.T) {
	devicePlaceholder := &fakeAdapter{name: "device"}
	servicebridge.SetDeviceSelectionBridge(
		func() any { return &fakeParameters{} },
		func(parameters any) (any, bool) {
			typed, _ := parameters.(*fakeParameters)
			clone := *typed
			return &clone, true
		},
		func(parameters any) (servicebridge.PresentationSnapshot, bool) {
			typed, _ := parameters.(*fakeParameters)
			return typed.snapshot, true
		},
		func() (any, error) { return &fakeAdapter{}, nil },
		func(device any) (int32, bool) {
			if device != any(devicePlaceholder) {
				return 0, false
			}
			return 1, true
		},
		func(any, bool) ([]any, error) { return nil, nil },
		func(any, []any) {},
	)
	manager := &GraphicsDeviceManager{deviceFacade: devicePlaceholder}
	information, err := NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatal(err)
	}
	// The information starts at profile zero and the device reports one.
	can, err := manager.CanResetDevice(information)
	if err != nil || can {
		t.Fatalf("CanResetDevice across profiles = %v, %v", can, err)
	}
	information.graphicsProfile = 1
	can, err = manager.CanResetDevice(information)
	if err != nil || !can {
		t.Fatalf("CanResetDevice with matching profiles = %v, %v", can, err)
	}
}
