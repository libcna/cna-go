package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// The ranking policy is pure managed arithmetic over PresentationParameters and
// GraphicsAdapter, so it can be measured with no native device at all -- but
// GraphicsDeviceInformation's CONSTRUCTOR reads GraphicsAdapter.DefaultAdapter,
// which cannot be answered without one. These tests therefore install the same
// bridge production installs with the default-adapter reader replaced by a stub
// that reports no adapter. Nothing else is substituted: the parameter factory,
// the cloner, the snapshot reader and the ranker are the production ones.

// stubDefaultAdapter stands in for GraphicsAdapter.DefaultAdapter. It is ONE
// object, as the reference's cached static is, and it is not the default
// adapter -- so the comparer's last step cannot decide by accident.
var stubDefaultAdapter = &GraphicsAdapter{}

func withStubDefaultAdapter(t *testing.T) {
	t.Helper()
	installDeviceSelectionBridge(func() (any, error) { return stubDefaultAdapter, nil })
	t.Cleanup(func() { installDeviceSelectionBridge(defaultAdapterBridge) })
}

// newCandidate builds one GraphicsDeviceInformation through the projected
// surface only.
func newCandidate(t *testing.T, configure func(*PresentationParameters)) *framework.GraphicsDeviceInformation {
	t.Helper()
	information, err := framework.NewGraphicsDeviceInformation()
	if err != nil {
		t.Fatalf("NewGraphicsDeviceInformation: %v", err)
	}
	parameters := GraphicsDeviceInformationPresentationParameters(information)
	if parameters == nil {
		t.Fatal("the information has no PresentationParameters")
	}
	if configure != nil {
		configure(parameters)
	}
	return information
}

func TestSurfaceFormatBitDepthIsTheReferencesFiveEntryTable(t *testing.T) {
	// The reference's switch maps exactly five of the twenty formats. Every
	// other one is zero, including the floating-point and compressed families,
	// so they all share a bit depth and rank 1 against each other.
	cases := map[SurfaceFormat]int32{
		SurfaceFormatColor:           32,
		SurfaceFormatRgba1010102:     32,
		SurfaceFormatBgr565:          16,
		SurfaceFormatBgra5551:        16,
		SurfaceFormatBgra4444:        16,
		SurfaceFormatDxt1:            0,
		SurfaceFormatDxt5:            0,
		SurfaceFormatHalfVector4:     0,
		SurfaceFormatVector4:         0,
		SurfaceFormatHdrBlendable:    0,
		SurfaceFormatAlpha8:          0,
		SurfaceFormatNormalizedByte4: 0,
	}
	for format, want := range cases {
		if got := surfaceFormatBitDepth(format); got != want {
			t.Fatalf("surfaceFormatBitDepth(%v) = %d, want %d", format, got, want)
		}
	}
}

func TestRankBackBufferFormatIsExactThenBitDepthThenMaxInt(t *testing.T) {
	context := deviceCandidateContext{preferredFormat: SurfaceFormatColor}
	if got := rankBackBufferFormat(context, SurfaceFormatColor); got != 0 {
		t.Fatalf("exact match ranked %d, want 0", got)
	}
	// Rgba1010102 is a different format of the same 32-bit depth.
	if got := rankBackBufferFormat(context, SurfaceFormatRgba1010102); got != 1 {
		t.Fatalf("same bit depth ranked %d, want 1", got)
	}
	if got := rankBackBufferFormat(context, SurfaceFormatBgr565); got != 0x7FFFFFFF {
		t.Fatalf("different bit depth ranked %d, want int.MaxValue", got)
	}
	// The zero-depth families all agree with each other, which is the
	// reference's table rather than an accident.
	sixteenBit := deviceCandidateContext{preferredFormat: SurfaceFormatDxt1}
	if got := rankBackBufferFormat(sixteenBit, SurfaceFormatVector4); got != 1 {
		t.Fatalf("two unmapped formats ranked %d, want 1", got)
	}
}

// TestCompareDeviceCandidatesFollowsTheReferenceOrder walks the comparer's
// eight steps, each with the earlier ones held equal so exactly one decides.
func TestCompareDeviceCandidatesFollowsTheReferenceOrder(t *testing.T) {
	withStubDefaultAdapter(t)

	t.Run("higher GraphicsProfile wins", func(t *testing.T) {
		low := newCandidate(t, nil)
		high := newCandidate(t, nil)
		SetGraphicsDeviceInformationGraphicsProfile(high, GraphicsProfileHiDef)
		context := deviceCandidateContext{}
		if compareDeviceCandidates(context, high, low) != -1 {
			t.Fatal("HiDef did not sort before Reach")
		}
		if compareDeviceCandidates(context, low, high) != 1 {
			t.Fatal("Reach did not sort after HiDef")
		}
	})

	t.Run("the manager's IsFullScreen decides", func(t *testing.T) {
		windowed := newCandidate(t, func(p *PresentationParameters) { p.SetIsFullScreen(false) })
		fullScreen := newCandidate(t, func(p *PresentationParameters) { p.SetIsFullScreen(true) })
		wantsFullScreen := deviceCandidateContext{isFullScreen: true}
		if compareDeviceCandidates(wantsFullScreen, fullScreen, windowed) != -1 {
			t.Fatal("a full-screen manager did not prefer the full-screen candidate")
		}
		wantsWindowed := deviceCandidateContext{isFullScreen: false}
		if compareDeviceCandidates(wantsWindowed, fullScreen, windowed) != 1 {
			t.Fatal("a windowed manager did not prefer the windowed candidate")
		}
	})

	t.Run("lower format rank wins", func(t *testing.T) {
		exact := newCandidate(t, func(p *PresentationParameters) { p.SetBackBufferFormat(SurfaceFormatColor) })
		sameDepth := newCandidate(t, func(p *PresentationParameters) { p.SetBackBufferFormat(SurfaceFormatRgba1010102) })
		context := deviceCandidateContext{preferredFormat: SurfaceFormatColor}
		if compareDeviceCandidates(context, exact, sameDepth) != -1 {
			t.Fatal("the exact format did not win")
		}
	})

	t.Run("higher MultiSampleCount wins", func(t *testing.T) {
		none := newCandidate(t, func(p *PresentationParameters) { p.SetMultiSampleCount(0) })
		many := newCandidate(t, func(p *PresentationParameters) { p.SetMultiSampleCount(8) })
		context := deviceCandidateContext{}
		if compareDeviceCandidates(context, many, none) != -1 {
			t.Fatal("more samples did not win")
		}
	})

	t.Run("aspect ratio decides only outside the 0.2 window", func(t *testing.T) {
		// The manager prefers 800x480, an aspect of 1.666...
		context := deviceCandidateContext{preferredBackBufferSize: [2]int32{800, 480}}
		// 1600x960 is the same aspect; 640x480 is 1.333, which is 0.333 away --
		// outside the window, so it loses.
		matching := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(1600)
			p.SetBackBufferHeight(960)
		})
		square := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(640)
			p.SetBackBufferHeight(480)
		})
		if compareDeviceCandidates(context, matching, square) != -1 {
			t.Fatal("the closer aspect did not win outside the tolerance")
		}
		// Two aspects whose distances differ by less than 0.2 tie here and are
		// decided by area instead. 800x480 and 848x480 differ by 0.1 in aspect.
		near := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(848)
			p.SetBackBufferHeight(480)
		})
		exact := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(800)
			p.SetBackBufferHeight(480)
		})
		// The exact size has zero area error and wins on step 6, not step 5.
		if compareDeviceCandidates(context, exact, near) != -1 {
			t.Fatal("inside the aspect tolerance, the closer AREA did not win")
		}
	})

	t.Run("closer pixel count wins", func(t *testing.T) {
		context := deviceCandidateContext{preferredBackBufferSize: [2]int32{800, 480}}
		exact := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(800)
			p.SetBackBufferHeight(480)
		})
		bigger := newCandidate(t, func(p *PresentationParameters) {
			p.SetBackBufferWidth(1600)
			p.SetBackBufferHeight(960)
		})
		if compareDeviceCandidates(context, exact, bigger) != -1 {
			t.Fatal("the exact pixel count did not win")
		}
	})

	t.Run("the default adapter wins the tie", func(t *testing.T) {
		context := deviceCandidateContext{}
		defaultOne := newCandidate(t, nil)
		otherOne := newCandidate(t, nil)
		if err := SetGraphicsDeviceInformationAdapter(defaultOne, &GraphicsAdapter{isDefaultAdapter: true}); err != nil {
			t.Fatal(err)
		}
		if err := SetGraphicsDeviceInformationAdapter(otherOne, &GraphicsAdapter{}); err != nil {
			t.Fatal(err)
		}
		if compareDeviceCandidates(context, defaultOne, otherOne) != -1 {
			t.Fatal("the default adapter did not win")
		}
		if compareDeviceCandidates(context, otherOne, defaultOne) != 1 {
			t.Fatal("the non-default adapter did not lose")
		}
	})

	t.Run("identical candidates tie", func(t *testing.T) {
		context := deviceCandidateContext{}
		left := newCandidate(t, nil)
		right := newCandidate(t, nil)
		if compareDeviceCandidates(context, left, right) != 0 {
			t.Fatal("two identical candidates did not tie")
		}
	})
}

// TestPreferredPixelTargetsHasThreeBranches pins the one step whose two
// candidates can have DIFFERENT targets.
func TestPreferredPixelTargetsHasThreeBranches(t *testing.T) {
	withStubDefaultAdapter(t)
	left := newCandidate(t, nil)
	right := newCandidate(t, nil)

	// Windowed with no preferred size: the shared 800x480 default area.
	windowed := deviceCandidateContext{}
	target1, target2 := preferredPixelTargets(windowed, left, right)
	if target1 != 800*480 || target2 != 800*480 {
		t.Fatalf("windowed default targets = %d/%d", target1, target2)
	}
	// Any preferred size: the shared preferred area.
	preferred := deviceCandidateContext{preferredBackBufferSize: [2]int32{1024, 768}}
	target1, target2 = preferredPixelTargets(preferred, left, right)
	if target1 != 1024*768 || target2 != 1024*768 {
		t.Fatalf("preferred targets = %d/%d", target1, target2)
	}
	// Full screen with no preferred size: EACH candidate's own adapter's
	// current mode area, so the two differ.
	if err := SetGraphicsDeviceInformationAdapter(left, &GraphicsAdapter{
		currentMode: &DisplayMode{width: 1920, height: 1080},
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetGraphicsDeviceInformationAdapter(right, &GraphicsAdapter{
		currentMode: &DisplayMode{width: 1280, height: 720},
	}); err != nil {
		t.Fatal(err)
	}
	fullScreen := deviceCandidateContext{isFullScreen: true}
	target1, target2 = preferredPixelTargets(fullScreen, left, right)
	if target1 != 1920*1080 || target2 != 1280*720 {
		t.Fatalf("full-screen targets = %d/%d, want each candidate's own adapter", target1, target2)
	}
}

// TestRankDeviceCandidatesSortsInPlace is the RankDevices claim seen through
// the bridge the projected member uses.
func TestRankDeviceCandidatesSortsInPlace(t *testing.T) {
	withStubDefaultAdapter(t)
	reach := newCandidate(t, nil)
	hiDef := newCandidate(t, nil)
	SetGraphicsDeviceInformationGraphicsProfile(hiDef, GraphicsProfileHiDef)

	candidates := []any{reach, hiDef}
	manager := &framework.GraphicsDeviceManager{}
	rankDeviceCandidates(manager, candidates)
	if candidates[0] != any(hiDef) || candidates[1] != any(reach) {
		t.Fatal("ranking did not put the higher profile first, in place")
	}
}

// TestDeviceSelectionBridgeIsTheProductionOne proves the substitution above is
// narrow: the parameter factory, cloner and snapshot reader under test are the
// ones production installs.
func TestDeviceSelectionBridgeIsTheProductionOne(t *testing.T) {
	parameters, ok := newPresentationParametersBridge().(*PresentationParameters)
	if !ok || parameters == nil {
		t.Fatal("the bridge factory did not build a PresentationParameters")
	}
	parameters.SetBackBufferWidth(1234)
	clone, ok := clonePresentationParametersBridge(parameters)
	if !ok {
		t.Fatal("the bridge cloner refused a PresentationParameters")
	}
	typed, _ := clone.(*PresentationParameters)
	if typed == parameters || typed.BackBufferWidth() != 1234 {
		t.Fatal("the bridge cloner did not member-wise copy")
	}
	snapshot, ok := presentationSnapshotBridge(parameters)
	if !ok || snapshot.BackBufferWidth != 1234 {
		t.Fatalf("the bridge snapshot = %+v", snapshot)
	}
	if _, ok := presentationSnapshotBridge("not parameters"); ok {
		t.Fatal("the bridge snapshot accepted something else")
	}
	var _ servicebridge.PresentationSnapshot = snapshot
}
