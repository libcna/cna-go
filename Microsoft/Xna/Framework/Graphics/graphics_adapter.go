package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 68 — GraphicsAdapter.
// ---------------------------------------------------------------------------

// GraphicsAdapter is Microsoft.Xna.Framework.Graphics.GraphicsAdapter, the type
// that describes one display adapter and negotiates formats with it.
//
// # Every value it reports comes from CNA, and CNA needs a DEVICE
//
// That is the shape of this milestone and the reason for its one recorded
// narrowing. All twelve `cna_graphics_adapter_*` routes take a
// **callback-scoped graphics-device handle**, which CNA requires as proof of an
// active runtime and the right thread. XNA's `Adapters` and `DefaultAdapter`
// are STATIC and answer before any device exists -- they are how a consumer
// picks one.
//
// So the two static members are projected, are FALLIBLE, and refuse outside a
// lifecycle callback. That is not a projection gap being hidden: it is CNA's
// documented requirement, stated where a consumer meets it, and the alternative
// -- inventing an adapter list from nothing -- would be reporting hardware that
// was never enumerated.
//
// # Its instances are snapshots, and are not owned
//
//	BORROWED. No CNA handle, nothing to destroy.
//
// CNA identifies an adapter by an INDEX rather than by a handle, and every
// query takes that index plus a live device. So a GraphicsAdapter here is the
// index and the values CNA reported when it was read, which is the same thing
// the reference's is: a managed object over data the driver was asked for once.
type GraphicsAdapter struct {
	index uint32
	// The values CNA reported, read once. The reference reads its own fields
	// the same way -- D3D9's adapter identifier is queried at enumeration --
	// so a getter here is a field read and cannot fail.
	isDefaultAdapter bool
	isWideScreen     bool
	vendorID         int32
	deviceID         int32
	revision         int32
	subSystemID      int32
	description      string
	deviceName       string
	currentMode      *DisplayMode
	supportedModes   *DisplayModeCollection
	monitorHandle    uintptr
	// device is the live device the adapter was read through. It is kept
	// because the three QUERY members ask CNA again, and it is BORROWED: an
	// adapter does not own the device it was enumerated from.
	device *GraphicsDevice
}

// errGraphicsAdapterNil is the Go-only guard for a zero GraphicsAdapter.
var errGraphicsAdapterNil = errors.New("GraphicsAdapter is nil or uninitialized")

// errNoLiveDeviceForAdapters is the refusal the two static members give outside
// a callback. It names CNA's requirement rather than XNA's behaviour, because
// the requirement is CNA's.
var errNoLiveDeviceForAdapters = errors.New(
	"CNA enumerates adapters through a callback-scoped graphics device, so GraphicsAdapter.Adapters and DefaultAdapter are reachable only from inside a lifecycle callback")

// adapterDevice is the live device the static members enumerate through. It is
// installed by GraphicsDevice when one becomes live and cleared when it stops
// being, so the static members answer exactly while a callback is running.
var adapterDevice *GraphicsDevice

// setAdapterEnumerationDevice is called from the device facade. It is
// unexported: a consumer never chooses which device adapters are read through,
// any more than they do in XNA.
func setAdapterEnumerationDevice(device *GraphicsDevice) { adapterDevice = device }

// GraphicsAdapterAdapters is GraphicsAdapter::get_Adapters:
//
//	ldsfld pAdapterList; ret
//
// one static field read over a list D3D9 enumeration filled. Here it is CNA's
// enumeration, performed on demand, so the member is fallible where the
// reference's is not -- and the difference is exactly the callback requirement
// above.
//
// It is a package-level function rather than a method because the CLR member is
// STATIC, which is the settled static-member rule.
func GraphicsAdapterAdapters() (*framework.ReadOnlyCollection[*GraphicsAdapter], error) {
	adapters, err := enumerateAdapters()
	if err != nil {
		return nil, err
	}
	return framework.NewReadOnlyCollectionOverReferences(adapters), nil
}

// GraphicsAdapterDefaultAdapter is GraphicsAdapter::get_DefaultAdapter:
//
//	ldsfld pAdapterList; ldc.i4.0; get_Item(0); ret
//
// the FIRST adapter, by position, not the one whose IsDefaultAdapter is set.
// The reference enumerates with the default first, and CNA reports the same
// ordering; this returns element zero either way, because that is what the
// reference returns.
func GraphicsAdapterDefaultAdapter() (*GraphicsAdapter, error) {
	adapters, err := enumerateAdapters()
	if err != nil {
		return nil, err
	}
	if len(adapters) == 0 {
		return nil, fmt.Errorf("%w: CNA enumerated no adapters", errNoLiveDeviceForAdapters)
	}
	return adapters[0], nil
}

// enumerateAdapters reads every adapter CNA reports, once.
func enumerateAdapters() ([]*GraphicsAdapter, error) {
	facade := adapterDevice
	if facade == nil {
		return nil, errNoLiveDeviceForAdapters
	}
	device, err := facade.live()
	if err != nil {
		return nil, errNoLiveDeviceForAdapters
	}
	count, err := device.AdapterCount()
	if err != nil {
		return nil, err
	}
	adapters := make([]*GraphicsAdapter, 0, int(count))
	for index := uint32(0); uint64(index) < count; index++ {
		adapter, err := readAdapter(facade, device, index)
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, adapter)
	}
	return adapters, nil
}

// readAdapter takes the whole snapshot in one place, so every getter is a field
// read and none of them can fail.
func readAdapter(facade *GraphicsDevice, device *interop.Device, index uint32) (*GraphicsAdapter, error) {
	info, err := device.AdapterInfo(index)
	if err != nil {
		return nil, err
	}
	description, err := device.AdapterDescription(index, info.DescriptionBytes)
	if err != nil {
		return nil, err
	}
	deviceName, err := device.AdapterDeviceName(index, info.DeviceNameBytes)
	if err != nil {
		return nil, err
	}
	current, err := device.AdapterCurrentDisplayMode(index)
	if err != nil {
		return nil, err
	}
	reported, err := device.AdapterDisplayModes(index)
	if err != nil {
		return nil, err
	}
	modes := make([]*DisplayMode, 0, len(reported))
	for _, mode := range reported {
		modes = append(modes, &DisplayMode{width: mode.Width, height: mode.Height, format: SurfaceFormat(mode.Format)})
	}
	// The monitor handle is the one member whose failure is NOT fatal to the
	// snapshot: CNA answers CNA_RESULT_NOT_SUPPORTED on a headless renderer,
	// and XNA's own MonitorHandle is IntPtr.Zero when there is no monitor.
	monitor, monitorErr := device.AdapterMonitorHandle(index)
	if monitorErr != nil {
		monitor = 0
	}
	return &GraphicsAdapter{
		index:            info.Index,
		isDefaultAdapter: info.IsDefaultAdapter,
		isWideScreen:     info.IsWideScreen,
		vendorID:         info.VendorID,
		deviceID:         info.DeviceID,
		revision:         info.Revision,
		subSystemID:      info.SubSystemID,
		description:      description,
		deviceName:       deviceName,
		currentMode:      &DisplayMode{width: current.Width, height: current.Height, format: SurfaceFormat(current.Format)},
		supportedModes:   newDisplayModeCollection(modes),
		monitorHandle:    uintptr(monitor),
		device:           facade,
	}, nil
}

// Description is GraphicsAdapter::get_Description, one field read.
func (a *GraphicsAdapter) Description() string {
	if a == nil {
		return ""
	}
	return a.description
}

// DeviceName is GraphicsAdapter::get_DeviceName.
func (a *GraphicsAdapter) DeviceName() string {
	if a == nil {
		return ""
	}
	return a.deviceName
}

// VendorId is GraphicsAdapter::get_VendorId.
func (a *GraphicsAdapter) VendorId() int32 {
	if a == nil {
		return 0
	}
	return a.vendorID
}

// DeviceId is GraphicsAdapter::get_DeviceId.
func (a *GraphicsAdapter) DeviceId() int32 {
	if a == nil {
		return 0
	}
	return a.deviceID
}

// SubSystemId is GraphicsAdapter::get_SubSystemId. CNA reports ZERO here, and
// says so: "Adapter subsystem identifier; current CNA returns zero." The value
// is passed through rather than invented.
func (a *GraphicsAdapter) SubSystemId() int32 {
	if a == nil {
		return 0
	}
	return a.subSystemID
}

// Revision is GraphicsAdapter::get_Revision, which CNA also reports as zero.
func (a *GraphicsAdapter) Revision() int32 {
	if a == nil {
		return 0
	}
	return a.revision
}

// IsDefaultAdapter is GraphicsAdapter::get_IsDefaultAdapter.
func (a *GraphicsAdapter) IsDefaultAdapter() bool {
	if a == nil {
		return false
	}
	return a.isDefaultAdapter
}

// IsWideScreen is GraphicsAdapter::get_IsWideScreen, which the reference
// computes as `CurrentDisplayMode.AspectRatio > 1.6`. CNA computes its own and
// reports it, and this passes CNA's through -- the two agree on every mode
// either can report, and a second computation would be a second answer.
func (a *GraphicsAdapter) IsWideScreen() bool {
	if a == nil {
		return false
	}
	return a.isWideScreen
}

// CurrentDisplayMode is GraphicsAdapter::get_CurrentDisplayMode.
func (a *GraphicsAdapter) CurrentDisplayMode() *DisplayMode {
	if a == nil {
		return nil
	}
	return a.currentMode
}

// SupportedDisplayModes is GraphicsAdapter::get_SupportedDisplayModes, whose
// collection is built once at enumeration.
func (a *GraphicsAdapter) SupportedDisplayModes() *DisplayModeCollection {
	if a == nil {
		return nil
	}
	return a.supportedModes
}

// MonitorHandle is GraphicsAdapter::get_MonitorHandle, one of the SIX members
// the raw-handle rule admits a public `uintptr` on, because the authoritative
// XNA metadata declares System.IntPtr at exactly this position.
//
// It is an opaque pointer-sized bit value and nothing more: it may not be
// dereferenced, and zero means the same thing IntPtr.Zero does -- on a headless
// renderer CNA reports no monitor and this answers zero.
func (a *GraphicsAdapter) MonitorHandle() uintptr {
	if a == nil {
		return 0
	}
	return a.monitorHandle
}

// GraphicsAdapterUseReferenceDevice is
// GraphicsAdapter::get_UseReferenceDevice, a STATIC property over a D3D9
// creation preference. CNA keeps the pair per adapter, so this reports the
// default adapter's -- which is what a static reader can mean when the value
// is per adapter underneath.
func GraphicsAdapterUseReferenceDevice() (bool, error) {
	adapter, err := GraphicsAdapterDefaultAdapter()
	if err != nil {
		return false, err
	}
	return adapter.preferences(false)
}

// SetGraphicsAdapterUseReferenceDevice is set_UseReferenceDevice.
func SetGraphicsAdapterUseReferenceDevice(value bool) error {
	return applyAdapterPreference(false, value)
}

// GraphicsAdapterUseNullDevice is get_UseNullDevice.
func GraphicsAdapterUseNullDevice() (bool, error) {
	adapter, err := GraphicsAdapterDefaultAdapter()
	if err != nil {
		return false, err
	}
	return adapter.preferences(true)
}

// SetGraphicsAdapterUseNullDevice is set_UseNullDevice.
func SetGraphicsAdapterUseNullDevice(value bool) error {
	return applyAdapterPreference(true, value)
}

// preferences re-reads the pair from CNA rather than answering from the
// snapshot, because a SETTER may have moved it since.
func (a *GraphicsAdapter) preferences(nullDevice bool) (bool, error) {
	if a == nil || a.device == nil {
		return false, errGraphicsAdapterNil
	}
	device, err := a.device.live()
	if err != nil {
		return false, errNoLiveDeviceForAdapters
	}
	info, err := device.AdapterInfo(a.index)
	if err != nil {
		return false, err
	}
	if nullDevice {
		return info.UseNullDevice, nil
	}
	return info.UseReferenceDevice, nil
}

// applyAdapterPreference writes ONE of the two flags and preserves the other,
// because CNA's route takes both at once and a setter must not silently clear
// its neighbour.
func applyAdapterPreference(nullDevice, value bool) error {
	adapter, err := GraphicsAdapterDefaultAdapter()
	if err != nil {
		return err
	}
	device, err := adapter.device.live()
	if err != nil {
		return errNoLiveDeviceForAdapters
	}
	info, err := device.AdapterInfo(adapter.index)
	if err != nil {
		return err
	}
	useNull, useReference := info.UseNullDevice, info.UseReferenceDevice
	if nullDevice {
		useNull = value
	} else {
		useReference = value
	}
	return device.SetAdapterDevicePreferences(adapter.index, useNull, useReference)
}

// IsProfileSupported is GraphicsAdapter::IsProfileSupported(GraphicsProfile),
// which asks the adapter and is therefore fallible here where the reference's
// is not: the reference answers from capability bits it cached at enumeration,
// and CNA answers per call.
func (a *GraphicsAdapter) IsProfileSupported(graphicsProfile GraphicsProfile) (bool, error) {
	if a == nil || a.device == nil {
		return false, errGraphicsAdapterNil
	}
	device, err := a.device.live()
	if err != nil {
		return false, errNoLiveDeviceForAdapters
	}
	return device.AdapterIsProfileSupported(a.index, uint32(graphicsProfile))
}

// QueryBackBufferFormat is
// GraphicsAdapter::QueryBackBufferFormat(GraphicsProfile, SurfaceFormat,
// DepthFormat, int32, out SurfaceFormat, out DepthFormat, out int32).
//
// The member RETURNS a bool -- whether every requested value was accepted
// without substitution -- and the three `out` parameters project as extra
// results after it, on the settled out-parameter rule, in declaration order and
// before the language-added error.
//
// CNA reports that same flag as `exact_match`, so the bool is CNA's answer
// rather than a comparison this projection performs: a renderer that accepted a
// format while reporting a substitution would otherwise be invisible.
func (a *GraphicsAdapter) QueryBackBufferFormat(
	graphicsProfile GraphicsProfile, format SurfaceFormat, depthFormat DepthFormat, multiSampleCount int32,
) (bool, SurfaceFormat, DepthFormat, int32, error) {
	return a.queryFormat(false, graphicsProfile, format, depthFormat, multiSampleCount)
}

// QueryRenderTargetFormat is the same shape against the render-target route.
func (a *GraphicsAdapter) QueryRenderTargetFormat(
	graphicsProfile GraphicsProfile, format SurfaceFormat, depthFormat DepthFormat, multiSampleCount int32,
) (bool, SurfaceFormat, DepthFormat, int32, error) {
	return a.queryFormat(true, graphicsProfile, format, depthFormat, multiSampleCount)
}

// queryFormat is the shared body of both query members.
func (a *GraphicsAdapter) queryFormat(
	renderTarget bool, graphicsProfile GraphicsProfile, format SurfaceFormat, depthFormat DepthFormat, multiSampleCount int32,
) (bool, SurfaceFormat, DepthFormat, int32, error) {
	if a == nil || a.device == nil {
		return false, 0, 0, 0, errGraphicsAdapterNil
	}
	device, err := a.device.live()
	if err != nil {
		return false, 0, 0, 0, errNoLiveDeviceForAdapters
	}
	selection, err := device.AdapterQueryFormat(a.index, renderTarget,
		uint32(graphicsProfile), uint32(format), uint32(depthFormat), multiSampleCount)
	if err != nil {
		return false, 0, 0, 0, err
	}
	return selection.ExactMatch, SurfaceFormat(selection.Format),
		DepthFormat(selection.DepthFormat), selection.MultiSampleCount, nil
}

// Adapter is GraphicsDevice::get_Adapter:
//
//	ldarg.0; ldfld pCurrentAdapter; ret
//
// one field read of the adapter the device was created with. It is FALLIBLE
// here where the reference's is not, for the reason every adapter member is:
// CNA identifies an adapter by an index, and the index is read from the live
// device.
//
// The index and the snapshot are read TOGETHER, never cached apart. CNA says
// adapter indices are point-in-time values a later adapter change may renumber,
// so an adapter object built from a stale index would describe the wrong
// hardware while looking perfectly valid.
func (d *GraphicsDevice) Adapter() (*GraphicsAdapter, error) {
	device, err := d.live()
	if err != nil {
		return nil, err
	}
	index, err := device.AdapterIndex()
	if err != nil {
		return nil, err
	}
	return readAdapter(d, device, index)
}
