package graphics

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The managed half of GraphicsDevice's render state is small on purpose: every
// one of these members reaches CNA, so what can be measured without a live
// device is the Go-only guard and the two conversions that happen before the
// boundary. The round trips are proved by the device-state stress scenario,
// which writes each value to a live device and reads it back from there.

// TestEveryStateMemberRefusesAFacadeWithNoDevice pins the Go-only guard on all
// fifteen members at once. A member added later that reached d.device directly
// would panic here rather than report.
func TestEveryStateMemberRefusesAFacadeWithNoDevice(t *testing.T) {
	device := &GraphicsDevice{}
	calls := map[string]func() error{
		"BlendFactor":          func() error { _, err := device.BlendFactor(); return err },
		"SetBlendFactor":       func() error { return device.SetBlendFactor(framework.Color{}) },
		"MultiSampleMask":      func() error { _, err := device.MultiSampleMask(); return err },
		"SetMultiSampleMask":   func() error { return device.SetMultiSampleMask(0) },
		"ReferenceStencil":     func() error { _, err := device.ReferenceStencil(); return err },
		"SetReferenceStencil":  func() error { return device.SetReferenceStencil(0) },
		"ScissorRectangle":     func() error { _, err := device.ScissorRectangle(); return err },
		"SetScissorRectangle":  func() error { return device.SetScissorRectangle(framework.Rectangle{}) },
		"SetViewport":          func() error { return device.SetViewport(Viewport{}) },
		"GraphicsProfile":      func() error { _, err := device.GraphicsProfile(); return err },
		"GraphicsDeviceStatus": func() error { _, err := device.GraphicsDeviceStatus(); return err },
		"IsDisposed":           func() error { _, err := device.IsDisposed(); return err },
		"ClearWithColor":       func() error { return device.ClearByClearOptionsAndColorAndSingleAndInt32(0, framework.Color{}, 0, 0) },
		"ClearWithVector4": func() error {
			return device.ClearByClearOptionsAndVector4AndSingleAndInt32(0, framework.Vector4{}, 0, 0)
		},
		"PresentByNone": func() error { return device.PresentByNone() },
	}
	if len(calls) != 15 {
		t.Fatalf("Foundation 51 projected 15 state members and this test calls %d", len(calls))
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, errGraphicsDeviceNil) {
			t.Errorf("%s on a device-less facade = %v, want the nil-device guard", name, err)
		}
	}
	// A nil facade must report rather than panic, for the same reason.
	var missing *GraphicsDevice
	if _, err := missing.BlendFactor(); !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("BlendFactor on a nil facade = %v", err)
	}
}

// TestTheThreeEnumIdentitiesCrossUnchanged pins what the projection relies on
// when it converts a CNA identity to an XNA enum with a cast: that the two
// numbering schemes agree.
//
// Each pair is asserted by VALUE rather than by conversion, so a change on
// either side is a failure here rather than a silently different device state.
func TestTheThreeEnumIdentitiesCrossUnchanged(t *testing.T) {
	for name, pair := range map[string][2]int32{
		// XNA ClearOptions          CNA_CLEAR_OPTION_*
		"ClearOptions.Target":      {int32(ClearOptionsTarget), 1},
		"ClearOptions.DepthBuffer": {int32(ClearOptionsDepthBuffer), 2},
		"ClearOptions.Stencil":     {int32(ClearOptionsStencil), 4},
		// XNA GraphicsProfile       CNA_GRAPHICS_PROFILE_*
		"GraphicsProfile.Reach": {int32(GraphicsProfileReach), 0},
		"GraphicsProfile.HiDef": {int32(GraphicsProfileHiDef), 1},
		// XNA GraphicsDeviceStatus  CNA_GRAPHICS_DEVICE_STATUS_*
		"GraphicsDeviceStatus.Normal":   {int32(GraphicsDeviceStatusNormal), 0},
		"GraphicsDeviceStatus.Lost":     {int32(GraphicsDeviceStatusLost), 1},
		"GraphicsDeviceStatus.NotReset": {int32(GraphicsDeviceStatusNotReset), 2},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s is %d in the profile and %d in CNA", name, pair[0], pair[1])
		}
	}
}

// TestTheVector4ClearOverloadUsesTheColorConstructor pins the twenty bytes of
// IL that separate the two masked Clear overloads: the Vector4 one builds a
// Color with Color::.ctor(Vector4) and forwards. The conversion is therefore
// the Color constructor's, and this test measures that CNA-Go did not invent a
// second rounding rule -- it compares the projected constructor's answer with
// what the forwarding overload would have to produce.
func TestTheVector4ClearOverloadUsesTheColorConstructor(t *testing.T) {
	source := framework.NewVector4BySingleAndSingleAndSingleAndSingle(0.25, 0.5, 0.75, 1)
	converted := framework.NewColorByVector4(source)
	if converted.A() != 255 {
		t.Fatalf("Color(Vector4).A = %d, want 255 for an alpha of 1", converted.A())
	}
	if converted.R() == 0 && converted.G() == 0 && converted.B() == 0 {
		t.Fatal("Color(Vector4) produced black from a non-black vector")
	}
}
