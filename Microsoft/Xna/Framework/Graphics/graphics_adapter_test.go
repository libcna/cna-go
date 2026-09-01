package graphics

import (
	"errors"
	"strings"
	"testing"
)

func modes(specs ...[3]int32) []*DisplayMode {
	built := make([]*DisplayMode, 0, len(specs))
	for _, spec := range specs {
		built = append(built, &DisplayMode{width: spec[0], height: spec[1], format: SurfaceFormat(spec[2])})
	}
	return built
}

// TestTheIndexerFiltersByFormatRatherThanIndexing pins DisplayModeCollection's
// most misleading member: its argument is a FORMAT, its result is a SEQUENCE,
// and a reader expecting `modes[3]` would be wrong about both halves.
func TestTheIndexerFiltersByFormatRatherThanIndexing(t *testing.T) {
	collection := newDisplayModeCollection(modes(
		[3]int32{800, 600, int32(SurfaceFormatColor)},
		[3]int32{1024, 768, int32(SurfaceFormatBgr565)},
		[3]int32{1920, 1080, int32(SurfaceFormatColor)},
	))
	matches := drain(t, collection.Item(SurfaceFormatColor))
	if len(matches) != 2 {
		t.Fatalf("%d modes matched Color, want 2", len(matches))
	}
	if matches[0].Width() != 800 || matches[1].Width() != 1920 {
		t.Fatalf("the filter did not preserve order: %d, %d", matches[0].Width(), matches[1].Width())
	}
	// A format nothing matches answers an EMPTY sequence rather than nil or an
	// error: the reference returns the fresh empty list it built.
	none := drain(t, collection.Item(SurfaceFormat(999)))
	if none == nil || len(none) != 0 {
		t.Fatalf("an unmatched format answered %v, want an empty sequence", none)
	}
	// And the enumerator is UNFILTERED, walking every mode in order.
	all := drain(t, collection.GetEnumerator())
	if len(all) != 3 {
		t.Fatalf("GetEnumerator walked %d modes, want 3", len(all))
	}
}

func drain(t *testing.T, sequence any) []*DisplayMode {
	t.Helper()
	iterator, ok := sequence.(interface {
		Next() (*DisplayMode, bool, error)
	})
	if !ok {
		t.Fatalf("the sequence is %T, not an iterator over display modes", sequence)
	}
	collected := make([]*DisplayMode, 0)
	for {
		value, more, err := iterator.Next()
		if err != nil {
			t.Fatalf("iterating: %v", err)
		}
		if !more {
			return collected
		}
		collected = append(collected, value)
	}
}

// TestTheStaticMembersReportCNAsRequirementRatherThanInventingAList is the
// milestone's honesty test. Every CNA adapter route takes a callback-scoped
// device; XNA's Adapters and DefaultAdapter are static and answer before one
// exists. The refusal must name that, and must not be a generic nil.
func TestTheStaticMembersReportCNAsRequirementRatherThanInventingAList(t *testing.T) {
	previous := adapterDevice
	adapterDevice = nil
	t.Cleanup(func() { adapterDevice = previous })

	adapters, err := GraphicsAdapterAdapters()
	if err == nil {
		t.Fatal("Adapters answered with no live device")
	}
	if adapters != nil {
		t.Fatal("Adapters returned a collection alongside its error")
	}
	if !errors.Is(err, errNoLiveDeviceForAdapters) {
		t.Fatalf("%v, want the callback-scope refusal", err)
	}
	if !strings.Contains(err.Error(), "callback") {
		t.Fatalf("%v, want the refusal to name CNA's requirement", err)
	}
	if _, err := GraphicsAdapterDefaultAdapter(); !errors.Is(err, errNoLiveDeviceForAdapters) {
		t.Fatalf("DefaultAdapter = %v", err)
	}
	// The two static preference properties reach the default adapter, so they
	// carry the same refusal rather than a different one.
	if _, err := GraphicsAdapterUseNullDevice(); !errors.Is(err, errNoLiveDeviceForAdapters) {
		t.Fatalf("UseNullDevice = %v", err)
	}
	if err := SetGraphicsAdapterUseReferenceDevice(true); !errors.Is(err, errNoLiveDeviceForAdapters) {
		t.Fatalf("SetUseReferenceDevice = %v", err)
	}
	// And so does the device's own Adapter, through a different path.
	if _, err := (&GraphicsDevice{}).Adapter(); err == nil {
		t.Fatal("an unconstructed device reported an adapter")
	}
}

// TestASnapshotAdapterAnswersEveryReaderWithoutFailing pins that the eleven
// readers are field reads: the snapshot is taken once, at enumeration, so none
// of them can fail and all of them answer after the device is gone.
func TestASnapshotAdapterAnswersEveryReaderWithoutFailing(t *testing.T) {
	adapter := &GraphicsAdapter{
		index:            2,
		isDefaultAdapter: true,
		isWideScreen:     true,
		vendorID:         0x10DE,
		deviceID:         0x1234,
		revision:         0,
		subSystemID:      0,
		description:      "a description",
		deviceName:       "a device name",
		currentMode:      &DisplayMode{width: 1920, height: 1080, format: SurfaceFormatColor},
		supportedModes:   newDisplayModeCollection(modes([3]int32{800, 600, int32(SurfaceFormatColor)})),
		monitorHandle:    0,
	}
	if adapter.Description() != "a description" || adapter.DeviceName() != "a device name" {
		t.Fatal("the two string readers do not answer the snapshot")
	}
	if adapter.VendorId() != 0x10DE || adapter.DeviceId() != 0x1234 {
		t.Fatal("the two identifier readers do not answer the snapshot")
	}
	// CNA reports ZERO for both of these and says so; the values are passed
	// through rather than invented.
	if adapter.Revision() != 0 || adapter.SubSystemId() != 0 {
		t.Fatal("Revision and SubSystemId are not CNA's zeros")
	}
	if !adapter.IsDefaultAdapter() || !adapter.IsWideScreen() {
		t.Fatal("the two flag readers do not answer the snapshot")
	}
	if adapter.CurrentDisplayMode().Width() != 1920 {
		t.Fatal("CurrentDisplayMode does not answer the snapshot")
	}
	if adapter.SupportedDisplayModes() == nil {
		t.Fatal("SupportedDisplayModes does not answer the snapshot")
	}
	// MonitorHandle is one of the six positions the raw-handle rule admits a
	// public uintptr on, and zero is IntPtr.Zero: on a headless renderer CNA
	// reports no monitor and this answers zero rather than failing.
	if adapter.MonitorHandle() != 0 {
		t.Fatal("MonitorHandle is not zero for an adapter with no monitor")
	}
}

// TestTheThreeQueryMembersNeedALiveDeviceAndSayWhy pins the split: the eleven
// readers are snapshot field reads, and these three ask CNA again.
func TestTheThreeQueryMembersNeedALiveDeviceAndSayWhy(t *testing.T) {
	adapter := &GraphicsAdapter{index: 0}
	if _, err := adapter.IsProfileSupported(GraphicsProfileReach); !errors.Is(err, errGraphicsAdapterNil) {
		t.Fatalf("IsProfileSupported with no device = %v", err)
	}
	if _, _, _, _, err := adapter.QueryBackBufferFormat(
		GraphicsProfileReach, SurfaceFormatColor, DepthFormatDepth24, 0); !errors.Is(err, errGraphicsAdapterNil) {
		t.Fatalf("QueryBackBufferFormat with no device = %v", err)
	}
	if _, _, _, _, err := adapter.QueryRenderTargetFormat(
		GraphicsProfileReach, SurfaceFormatColor, DepthFormatDepth24, 0); !errors.Is(err, errGraphicsAdapterNil) {
		t.Fatalf("QueryRenderTargetFormat with no device = %v", err)
	}
	// And an adapter with a facade whose device is gone reports the
	// callback-scope refusal rather than the uninitialized one.
	withFacade := &GraphicsAdapter{index: 0, device: &GraphicsDevice{}}
	if _, err := withFacade.IsProfileSupported(GraphicsProfileReach); !errors.Is(err, errNoLiveDeviceForAdapters) {
		t.Fatalf("IsProfileSupported on a dead device = %v", err)
	}
}

// TestAZeroAdapterIsRefusedRatherThanPanicking covers the Go-only guard.
func TestAZeroAdapterIsRefusedRatherThanPanicking(t *testing.T) {
	var absent *GraphicsAdapter
	if absent.Description() != "" || absent.VendorId() != 0 || absent.MonitorHandle() != 0 {
		t.Fatal("a nil adapter answered with values")
	}
	if absent.CurrentDisplayMode() != nil || absent.SupportedDisplayModes() != nil {
		t.Fatal("a nil adapter answered with objects")
	}
	var absentCollection *DisplayModeCollection
	if len(drain(t, absentCollection.GetEnumerator())) != 0 {
		t.Fatal("a nil collection enumerated something")
	}
	if len(drain(t, absentCollection.Item(SurfaceFormatColor))) != 0 {
		t.Fatal("a nil collection matched something")
	}
}
