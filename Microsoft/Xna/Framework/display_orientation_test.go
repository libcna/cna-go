package framework

import "testing"

func TestDisplayOrientationRawValuesAndCombinations(t *testing.T) {
	if DisplayOrientationDefault != 0 || DisplayOrientationLandscapeLeft != 1 ||
		DisplayOrientationLandscapeRight != 2 || DisplayOrientationPortrait != 4 {
		t.Fatal("DisplayOrientation constants do not match XNA")
	}

	landscape := DisplayOrientationLandscapeLeft | DisplayOrientationLandscapeRight
	all := landscape | DisplayOrientationPortrait
	if int32(landscape) != 3 || int32(all) != 7 {
		t.Fatalf("flags combinations = %d and %d, want 3 and 7", landscape, all)
	}

	const unknownRaw int32 = 1 << 20
	if got := int32(DisplayOrientation(unknownRaw)); got != unknownRaw {
		t.Fatalf("unknown raw value = %d, want %d", got, unknownRaw)
	}
}

func TestGraphicsDeviceManagerSupportedOrientationsManagedState(t *testing.T) {
	manager := &GraphicsDeviceManager{}
	if got := manager.SupportedOrientations(); got != DisplayOrientationDefault {
		t.Fatalf("initial SupportedOrientations = %d, want Default", got)
	}
	if manager.isDeviceDirty {
		t.Fatal("initial manager dirty state is true")
	}

	manager.SetSupportedOrientations(DisplayOrientationDefault)
	if !manager.isDeviceDirty {
		t.Fatal("same-value setter did not mark the manager dirty")
	}

	manager.isDeviceDirty = false
	combined := DisplayOrientationLandscapeLeft | DisplayOrientationPortrait
	manager.SetSupportedOrientations(combined)
	if got := manager.SupportedOrientations(); got != combined {
		t.Fatalf("combined SupportedOrientations = %d, want %d", got, combined)
	}
	if !manager.isDeviceDirty {
		t.Fatal("changed-value setter did not mark the manager dirty")
	}

	manager.isDeviceDirty = false
	unknown := DisplayOrientation(1 << 20)
	manager.SetSupportedOrientations(unknown)
	manager.SetSupportedOrientations(DisplayOrientationLandscapeRight)
	manager.SetSupportedOrientations(unknown)
	if got := manager.SupportedOrientations(); got != unknown {
		t.Fatalf("multiple assignment result = %d, want %d", got, unknown)
	}
	if !manager.isDeviceDirty {
		t.Fatal("multiple setters did not retain dirty state")
	}
}

func TestGraphicsDeviceManagerSupportedOrientationsAfterDispose(t *testing.T) {
	manager := &GraphicsDeviceManager{}
	if err := manager.Dispose(true); err != nil {
		t.Fatalf("Dispose managed-only manager: %v", err)
	}
	value := DisplayOrientationLandscapeLeft | DisplayOrientationLandscapeRight
	manager.SetSupportedOrientations(value)
	if got := manager.SupportedOrientations(); got != value {
		t.Fatalf("post-disposal SupportedOrientations = %d, want %d", got, value)
	}
}
