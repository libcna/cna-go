package graphics

import (
	"errors"
	"sort"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file is the Graphics-package half of GraphicsDeviceManager's device
// selection: the private AddDevices enumeration that builds the candidate list,
// and the private GraphicsDeviceInformationComparer that ranks it.
//
// Neither is projected surface. Both are `private` in the reference, and both
// are here rather than in the framework package for the same reason the three
// GraphicsDeviceInformation properties are: every value they read is a
// Graphics type, and the framework package cannot name one. What IS projected
// -- FindBestDevice, RankDevices, CanResetDevice -- stays on
// GraphicsDeviceManager, where the reference declares it.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Game.dll
//	  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
//
// # The one platform substitution, measured
//
// `AddDevices(bool anySuitableDevice, List<GDI>)` filters adapters with the
// private `IsWindowOnAdapter(windowHandle, adapter)` when anySuitableDevice is
// false. That method is
//
//	WindowsGameWindow.ScreenFromAdapter(adapter) == WindowsGameWindow.ScreenFromHandle(handle)
//
// and ScreenFromAdapter, read from the same assembly, is a linear scan of
// `Screen.AllScreens` for the screen whose device name equals
// `adapter.DeviceName`. Both sides of the comparison therefore reduce to a
// DISPLAY DEVICE NAME, and both names are members the pinned contract already
// declares: GraphicsAdapter::get_DeviceName and GameWindow::get_ScreenDeviceName.
// The projection compares those two strings.
//
// This is not System.Windows.Forms reproduced -- CNA-Go has no Screen, and
// inventing one would be fabrication. It is the reference's own PREDICATE
// expressed through the two reference members that carry its operands.

// errNoManagerWindow is the failure AddDevices reports when the manager has no
// game, which is the reference's NullReferenceException on `this.game.Window`.
var errNoManagerWindow = errors.New("the graphics device manager has no game window")

// deviceCandidateContext is the manager state AddDevices and the comparer read.
// Collecting it once keeps every read of the manager in one place and makes the
// two bodies below arithmetic over values rather than a chain of bridge calls.
type deviceCandidateContext struct {
	manager                 *framework.GraphicsDeviceManager
	graphicsProfile         GraphicsProfile
	preferredBackBufferSize [2]int32
	preferredFormat         SurfaceFormat
	preferredDepthFormat    DepthFormat
	isFullScreen            bool
	preferMultiSampling     bool
	synchronizeWithVSync    bool
}

func newDeviceCandidateContext(manager *framework.GraphicsDeviceManager) deviceCandidateContext {
	return deviceCandidateContext{
		manager:         manager,
		graphicsProfile: GraphicsDeviceManagerGraphicsProfile(manager),
		preferredBackBufferSize: [2]int32{
			manager.PreferredBackBufferWidth(), manager.PreferredBackBufferHeight(),
		},
		preferredFormat:      GraphicsDeviceManagerPreferredBackBufferFormat(manager),
		preferredDepthFormat: GraphicsDeviceManagerPreferredDepthStencilFormat(manager),
		isFullScreen:         manager.IsFullScreen(),
		preferMultiSampling:  manager.PreferMultiSampling(),
		synchronizeWithVSync: manager.SynchronizeWithVerticalRetrace(),
	}
}

// collectDeviceCandidates is GraphicsDeviceManager::AddDevices(bool, List<GDI>).
//
//	handle = game.Window.Handle;
//	foreach (GraphicsAdapter adapter in GraphicsAdapter.Adapters) {
//	    if (!anySuitableDevice && !IsWindowOnAdapter(handle, adapter)) continue;
//	    try {
//	        if (!adapter.IsProfileSupported(graphicsProfile)) continue;
//	        GraphicsDeviceInformation info = new GraphicsDeviceInformation();
//	        info.Adapter                                    = adapter;
//	        info.GraphicsProfile                            = graphicsProfile;
//	        info.PresentationParameters.DeviceWindowHandle  = handle;
//	        info.PresentationParameters.MultiSampleCount    = 0;
//	        info.PresentationParameters.IsFullScreen        = IsFullScreen;
//	        info.PresentationParameters.PresentationInterval =
//	            SynchronizeWithVerticalRetrace ? PresentInterval.One : PresentInterval.Immediate;
//	        AddDevices(adapter, adapter.CurrentDisplayMode, info, foundDevices);
//	        if (isFullScreen)
//	            foreach (DisplayMode mode in adapter.SupportedDisplayModes)
//	                if (mode.Width >= 640 && mode.Height >= 480)
//	                    AddDevices(adapter, mode, info, foundDevices);
//	    } catch (NotSupportedException) { }
//	}
//
// Four details are the reference's and are reproduced:
//
//   - the base information is built ONCE per adapter and CLONED per mode, so
//     every candidate for one adapter shares its window handle and interval;
//   - `MultiSampleCount = 0` is set on the base and then overwritten by the
//     per-mode query, so the zero is not the value that survives;
//   - the full-screen mode sweep admits only modes at least 640x480, from the
//     literals 0x280 and 0x1e0;
//   - a `NotSupportedException` from one adapter skips that ADAPTER and
//     continues with the next, rather than failing the whole enumeration. In
//     CNA-Go the equivalent is a failure reported by the adapter's own members,
//     and it is swallowed at the same boundary.
func collectDeviceCandidates(manager *framework.GraphicsDeviceManager, anySuitableDevice bool) ([]any, error) {
	if manager == nil {
		return nil, nil
	}
	context := newDeviceCandidateContext(manager)
	window, _ := servicebridge.ReadManagerWindow(manager).(*framework.GameWindow)
	if window == nil {
		// The reference reads `this.game.Window` unguarded, so a manager with
		// no game is its NullReferenceException.
		return nil, errNoManagerWindow
	}
	handle, err := window.Handle()
	if err != nil {
		// GameWindow::get_Handle reaches the platform window; without one there
		// is no window to place on an adapter and no handle to record.
		return nil, err
	}
	windowScreen := ""
	if !anySuitableDevice {
		if windowScreen, err = window.ScreenDeviceName(); err != nil {
			return nil, err
		}
	}

	adapters, err := GraphicsAdapterAdapters()
	if err != nil {
		return nil, err
	}
	var found []any
	iterator := adapters.GetEnumerator()
	for {
		adapter, ok, iterateErr := iterator.Next()
		if iterateErr != nil {
			return nil, iterateErr
		}
		if !ok {
			break
		}
		if !anySuitableDevice && adapter.DeviceName() != windowScreen {
			continue
		}
		supported, err := adapter.IsProfileSupported(context.graphicsProfile)
		if err != nil {
			// The reference's `catch (NotSupportedException)`: this adapter is
			// skipped and the enumeration continues.
			continue
		}
		if !supported {
			continue
		}
		base, err := framework.NewGraphicsDeviceInformation()
		if err != nil {
			return nil, err
		}
		assignCandidateAdapter(base, adapter)
		SetGraphicsDeviceInformationGraphicsProfile(base, context.graphicsProfile)
		parameters := GraphicsDeviceInformationPresentationParameters(base)
		if parameters == nil {
			continue
		}
		parameters.SetDeviceWindowHandle(handle)
		parameters.SetMultiSampleCount(0)
		parameters.SetIsFullScreen(context.isFullScreen)
		interval := PresentIntervalImmediate
		if context.synchronizeWithVSync {
			interval = PresentIntervalOne
		}
		parameters.SetPresentationInterval(interval)

		found = addModeCandidate(context, adapter, adapter.CurrentDisplayMode(), base, found)
		if !context.isFullScreen {
			continue
		}
		modes := adapter.SupportedDisplayModes()
		if modes == nil {
			continue
		}
		modeIterator := modes.GetEnumerator()
		for {
			mode, ok, iterateErr := modeIterator.Next()
			if iterateErr != nil {
				return nil, iterateErr
			}
			if !ok {
				break
			}
			if mode == nil || mode.Width() < 640 || mode.Height() < 480 {
				continue
			}
			found = addModeCandidate(context, adapter, mode, base, found)
		}
	}
	return found, nil
}

// addModeCandidate is the second, private
// AddDevices(GraphicsAdapter, DisplayMode, GraphicsDeviceInformation, List<GDI>).
//
//	GraphicsDeviceInformation info = baseDeviceInfo.Clone();
//	if (IsFullScreen) {
//	    info.PresentationParameters.BackBufferWidth  = mode.Width;
//	    info.PresentationParameters.BackBufferHeight = mode.Height;
//	} else if (useResizedBackBuffer) {
//	    ... resizedBackBuffer* ...
//	} else {
//	    info.PresentationParameters.BackBufferWidth  = PreferredBackBufferWidth;
//	    info.PresentationParameters.BackBufferHeight = PreferredBackBufferHeight;
//	}
//	adapter.QueryBackBufferFormat(info.GraphicsProfile, mode.Format,
//	    PreferredDepthStencilFormat, PreferMultiSampling ? 16 : 0,
//	    out format, out depth, out samples);        // the RESULT is popped
//	info.PresentationParameters.BackBufferFormat   = format;
//	info.PresentationParameters.DepthStencilFormat = depth;
//	info.PresentationParameters.MultiSampleCount   = samples;
//	if (!foundDevices.Contains(info)) foundDevices.Add(info);
//
// Two details matter. The query's BOOL is discarded -- `IL_00ac: pop` -- so an
// inexact match is accepted and the substituted format is what the candidate
// carries. And the duplicate test is `List<T>.Contains`, which uses
// EqualityComparer<GraphicsDeviceInformation>.Default and therefore
// GraphicsDeviceInformation::Equals: two candidates that differ only in a
// PresentationParameters value nobody compares are the SAME candidate.
//
// The `useResizedBackBuffer` branch is unreachable in CNA-Go today. The
// reference raises that flag from GameWindow's ClientSizeChanged handler, which
// CNA-Go does not project, so the flag is never true; the branch is kept in the
// projection order above rather than deleted, because the flag is real.
func addModeCandidate(
	context deviceCandidateContext, adapter *GraphicsAdapter, mode *DisplayMode,
	base *framework.GraphicsDeviceInformation, found []any,
) []any {
	if mode == nil {
		// The reference dereferences the mode; a null one is its
		// NullReferenceException. An adapter with no current display mode
		// contributes no candidate rather than crashing the enumeration.
		return found
	}
	candidate, err := base.Clone()
	if err != nil {
		return found
	}
	parameters := GraphicsDeviceInformationPresentationParameters(candidate)
	if parameters == nil {
		return found
	}
	if context.isFullScreen {
		parameters.SetBackBufferWidth(mode.Width())
		parameters.SetBackBufferHeight(mode.Height())
	} else {
		parameters.SetBackBufferWidth(context.preferredBackBufferSize[0])
		parameters.SetBackBufferHeight(context.preferredBackBufferSize[1])
	}
	requestedSamples := int32(0)
	if context.preferMultiSampling {
		requestedSamples = 16
	}
	_, format, depthFormat, samples, err := adapter.QueryBackBufferFormat(
		GraphicsDeviceInformationGraphicsProfile(candidate), mode.Format(),
		context.preferredDepthFormat, requestedSamples)
	if err != nil {
		return found
	}
	parameters.SetBackBufferFormat(format)
	parameters.SetDepthStencilFormat(depthFormat)
	parameters.SetMultiSampleCount(samples)
	for _, existing := range found {
		if candidate.Equals(existing) {
			return found
		}
	}
	return append(found, candidate)
}

// assignCandidateAdapter assigns the adapter and discards the reference's
// ArgumentNullException branch, which the setter can only raise on an
// information whose adapter is ALREADY null -- a state AddDevices' freshly
// constructed information is never in. It is unexported because it is
// machinery, not the projected setter.
func assignCandidateAdapter(
	information *framework.GraphicsDeviceInformation, adapter *GraphicsAdapter,
) {
	_ = SetGraphicsDeviceInformationAdapter(information, adapter)
}

// ---------------------------------------------------------------------------
// GraphicsDeviceInformationComparer.
// ---------------------------------------------------------------------------

// rankDeviceCandidates is GraphicsDeviceManager::RankDevicesPlatform, which is
// one `foundDevices.Sort(new GraphicsDeviceInformationComparer(this))`.
//
// List<T>.Sort is Array.Sort, which is introsort and is NOT stable. Go's
// sort.Slice is also unstable, so the projection uses it rather than
// sort.SliceStable: imposing a stability the reference does not have would make
// CNA-Go's order deterministic where the reference's is not, and a consumer who
// relied on that would be relying on CNA-Go rather than on XNA.
func rankDeviceCandidates(manager *framework.GraphicsDeviceManager, candidates []any) {
	if manager == nil {
		return
	}
	context := newDeviceCandidateContext(manager)
	sort.Slice(candidates, func(i, j int) bool {
		left, _ := candidates[i].(*framework.GraphicsDeviceInformation)
		right, _ := candidates[j].(*framework.GraphicsDeviceInformation)
		return compareDeviceCandidates(context, left, right) < 0
	})
}

// compareDeviceCandidates is GraphicsDeviceInformationComparer::Compare, in the
// reference's exact order. Every step returns as soon as it can decide, and
// -1 means d1 sorts FIRST -- that is, d1 is the better device.
//
//  1. GraphicsProfile: HIGHER wins. `d1 > d2 ? -1 : 1`.
//  2. IsFullScreen: the one matching the MANAGER's IsFullScreen wins.
//  3. RankFormat of the back-buffer format: LOWER rank wins. The rank is 0
//     for an exact match with PreferredBackBufferFormat, 1 for a different
//     format of the same bit depth, and int.MaxValue otherwise.
//  4. MultiSampleCount: HIGHER wins.
//  5. Aspect ratio: the one closer to the preferred aspect wins, but ONLY if
//     the two distances differ by more than 0.2. The preferred aspect is
//     PreferredBackBufferWidth/Height, or 800/480 when either is zero.
//  6. Pixel count: the one closer to the target area wins. The target is the
//     adapter's current mode area when full screen with no preferred size,
//     and the preferred (or default) area otherwise.
//  7. Adapter: the DEFAULT adapter wins.
//  8. Otherwise 0.
//
// Step 5's 0.2 window is what makes the ordering non-total: two candidates can
// tie on aspect and be separated only by area, which is the reference's
// deliberate tolerance rather than a rounding artefact.
func compareDeviceCandidates(
	context deviceCandidateContext, d1, d2 *framework.GraphicsDeviceInformation,
) int32 {
	if d1 == nil || d2 == nil {
		return 0
	}
	profile1 := GraphicsDeviceInformationGraphicsProfile(d1)
	profile2 := GraphicsDeviceInformationGraphicsProfile(d2)
	if profile1 != profile2 {
		if profile1 > profile2 {
			return -1
		}
		return 1
	}
	parameters1 := GraphicsDeviceInformationPresentationParameters(d1)
	parameters2 := GraphicsDeviceInformationPresentationParameters(d2)
	if parameters1 == nil || parameters2 == nil {
		return 0
	}
	if parameters1.IsFullScreen() != parameters2.IsFullScreen() {
		if context.isFullScreen == parameters1.IsFullScreen() {
			return -1
		}
		return 1
	}
	rank1 := rankBackBufferFormat(context, parameters1.BackBufferFormat())
	rank2 := rankBackBufferFormat(context, parameters2.BackBufferFormat())
	if rank1 != rank2 {
		if rank1 < rank2 {
			return -1
		}
		return 1
	}
	if parameters1.MultiSampleCount() != parameters2.MultiSampleCount() {
		if parameters1.MultiSampleCount() > parameters2.MultiSampleCount() {
			return -1
		}
		return 1
	}

	preferredAspect := float32(graphicsDeviceManagerDefaultAspect())
	if context.preferredBackBufferSize[0] != 0 && context.preferredBackBufferSize[1] != 0 {
		preferredAspect = float32(context.preferredBackBufferSize[0]) / float32(context.preferredBackBufferSize[1])
	}
	aspect1 := float32(parameters1.BackBufferWidth()) / float32(parameters1.BackBufferHeight())
	aspect2 := float32(parameters2.BackBufferWidth()) / float32(parameters2.BackBufferHeight())
	distance1 := absFloat32(aspect1 - preferredAspect)
	distance2 := absFloat32(aspect2 - preferredAspect)
	if absFloat32(distance1-distance2) > 0.2 {
		if distance1 < distance2 {
			return -1
		}
		return 1
	}

	target1, target2 := preferredPixelTargets(context, d1, d2)
	area1 := absInt32(parameters1.BackBufferWidth()*parameters1.BackBufferHeight() - target1)
	area2 := absInt32(parameters2.BackBufferWidth()*parameters2.BackBufferHeight() - target2)
	if area1 != area2 {
		if area1 < area2 {
			return -1
		}
		return 1
	}

	adapter1 := GraphicsDeviceInformationAdapter(d1)
	adapter2 := GraphicsDeviceInformationAdapter(d2)
	if adapter1 == adapter2 {
		return 0
	}
	if adapter1 != nil && adapter1.IsDefaultAdapter() {
		return -1
	}
	if adapter2 != nil && adapter2.IsDefaultAdapter() {
		return 1
	}
	return 0
}

// preferredPixelTargets is the comparer's step-6 target area, whose THREE
// branches disagree about whether the two candidates share one target:
//
//   - full screen with no preferred size: each candidate's OWN adapter's
//     current display mode area, so the two targets differ;
//   - full screen with a preferred size, or windowed with one: the preferred
//     area, shared;
//   - windowed with no preferred size: the default 800x480 area, shared.
func preferredPixelTargets(
	context deviceCandidateContext, d1, d2 *framework.GraphicsDeviceInformation,
) (int32, int32) {
	preferredWidth, preferredHeight := context.preferredBackBufferSize[0], context.preferredBackBufferSize[1]
	if context.isFullScreen && (preferredWidth == 0 || preferredHeight == 0) {
		return currentModeArea(GraphicsDeviceInformationAdapter(d1)), currentModeArea(GraphicsDeviceInformationAdapter(d2))
	}
	if preferredWidth == 0 || preferredHeight == 0 {
		area := framework.GraphicsDeviceManagerDefaultBackBufferWidth() *
			framework.GraphicsDeviceManagerDefaultBackBufferHeight()
		return area, area
	}
	area := preferredWidth * preferredHeight
	return area, area
}

func currentModeArea(adapter *GraphicsAdapter) int32 {
	if adapter == nil {
		return 0
	}
	mode := adapter.CurrentDisplayMode()
	if mode == nil {
		return 0
	}
	return mode.Width() * mode.Height()
}

// rankBackBufferFormat is GraphicsDeviceInformationComparer::RankFormat:
//
//	if (format == PreferredBackBufferFormat) return 0;
//	if (SurfaceFormatBitDepth(format) == SurfaceFormatBitDepth(Preferred)) return 1;
//	return int.MaxValue;
func rankBackBufferFormat(context deviceCandidateContext, format SurfaceFormat) int32 {
	if format == context.preferredFormat {
		return 0
	}
	if surfaceFormatBitDepth(format) == surfaceFormatBitDepth(context.preferredFormat) {
		return 1
	}
	return 0x7FFFFFFF
}

// surfaceFormatBitDepth is
// GraphicsDeviceInformationComparer::SurfaceFormatBitDepth, whose `switch` maps
// exactly five of the twenty SurfaceFormat values and answers zero for the rest:
//
//	Color (0)        -> 32
//	Bgr565 (1)       -> 16
//	Bgra5551 (2)     -> 16
//	Bgra4444 (3)     -> 16
//	Rgba1010102 (9)  -> 32
//
// Every other format, including the four floating-point and the three
// compressed ones, is zero -- so they all share a bit depth and rank 1 against
// each other. That is the reference's table, not an omission.
func surfaceFormatBitDepth(format SurfaceFormat) int32 {
	switch format {
	case SurfaceFormatColor, SurfaceFormatRgba1010102:
		return 32
	case SurfaceFormatBgr565, SurfaceFormatBgra5551, SurfaceFormatBgra4444:
		return 16
	default:
		return 0
	}
}

// graphicsDeviceManagerDefaultAspect is DefaultBackBufferWidth /
// DefaultBackBufferHeight, computed as the reference computes it: two int32
// statics converted to float32 and divided.
func graphicsDeviceManagerDefaultAspect() float32 {
	return float32(framework.GraphicsDeviceManagerDefaultBackBufferWidth()) /
		float32(framework.GraphicsDeviceManagerDefaultBackBufferHeight())
}

func absFloat32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func absInt32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

// The seven bridge halves, named rather than inlined into init so each one is
// readable on its own and so a test can install the same set with one
// substituted -- which is how the ranking policy is exercised without a live
// native adapter.

func newPresentationParametersBridge() any { return NewPresentationParameters() }

func clonePresentationParametersBridge(parameters any) (any, bool) {
	typed, ok := parameters.(*PresentationParameters)
	if !ok || typed == nil {
		return nil, false
	}
	return typed.Clone(), true
}

func presentationSnapshotBridge(parameters any) (servicebridge.PresentationSnapshot, bool) {
	typed, ok := parameters.(*PresentationParameters)
	if !ok || typed == nil {
		return servicebridge.PresentationSnapshot{}, false
	}
	return servicebridge.PresentationSnapshot{
		BackBufferWidth:      typed.BackBufferWidth(),
		BackBufferHeight:     typed.BackBufferHeight(),
		BackBufferFormat:     int32(typed.BackBufferFormat()),
		DepthStencilFormat:   int32(typed.DepthStencilFormat()),
		MultiSampleCount:     typed.MultiSampleCount(),
		DisplayOrientation:   int32(typed.DisplayOrientation()),
		PresentationInterval: int32(typed.PresentationInterval()),
		RenderTargetUsage:    int32(typed.RenderTargetUsage()),
		DeviceWindowHandle:   typed.DeviceWindowHandle(),
		IsFullScreen:         typed.IsFullScreen(),
	}, true
}

func defaultAdapterBridge() (any, error) {
	adapter, err := GraphicsAdapterDefaultAdapter()
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

func deviceGraphicsProfileBridge(device any) (int32, bool) {
	typed, ok := device.(*GraphicsDevice)
	if !ok || typed == nil {
		return 0, false
	}
	profile, err := typed.GraphicsProfile()
	if err != nil {
		return 0, false
	}
	return int32(profile), true
}

func collectDeviceCandidatesBridge(manager any, anySuitableDevice bool) ([]any, error) {
	typed, ok := manager.(*framework.GraphicsDeviceManager)
	if !ok || typed == nil {
		return nil, nil
	}
	return collectDeviceCandidates(typed, anySuitableDevice)
}

func rankDeviceCandidatesBridge(manager any, candidates []any) {
	typed, ok := manager.(*framework.GraphicsDeviceManager)
	if !ok || typed == nil {
		return
	}
	rankDeviceCandidates(typed, candidates)
}

// installDeviceSelectionBridge registers this package's half, with the default
// adapter reader supplied so the set can be installed once for production and
// once, with a stub, for the ranking tests.
func installDeviceSelectionBridge(defaultAdapter servicebridge.DefaultAdapterReader) {
	servicebridge.SetDeviceSelectionBridge(
		newPresentationParametersBridge,
		clonePresentationParametersBridge,
		presentationSnapshotBridge,
		defaultAdapter,
		deviceGraphicsProfileBridge,
		collectDeviceCandidatesBridge,
		rankDeviceCandidatesBridge,
	)
}

func init() { installDeviceSelectionBridge(defaultAdapterBridge) }
