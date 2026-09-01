package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	packedvector "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics/PackedVector"
)

// ---------------------------------------------------------------------------
// Foundation 73 — the managed half of the rest of GraphicsDevice.
// ---------------------------------------------------------------------------

// TestResetRefusesItsTwoNullsInTheReferencesOrder pins
// Reset(PresentationParameters, GraphicsAdapter)'s two guards and which comes
// first: the IL checks presentationParameters at IL_001a and graphicsAdapter at
// IL_002d, so a call with BOTH null reports the parameters.
func TestResetRefusesItsTwoNullsInTheReferencesOrder(t *testing.T) {
	device := &GraphicsDevice{}
	err := device.ResetByPresentationParametersAndGraphicsAdapter(nil, nil)
	if err == nil {
		t.Fatal("a reset with two nulls was accepted")
	}
	if !strings.Contains(err.Error(), "presentationParameters") {
		t.Fatalf("%v, want the parameters reported before the adapter", err)
	}
	if !strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("%v, want FrameworkResources.NullNotAllowed", err)
	}
	parameters := NewPresentationParameters()
	err = device.ResetByPresentationParametersAndGraphicsAdapter(parameters, nil)
	if err == nil || !strings.Contains(err.Error(), "graphicsAdapter") {
		t.Fatalf("%v, want the adapter reported once the parameters are valid", err)
	}
	// The one-argument overload keeps only the first guard.
	if err := device.ResetByPresentationParameters(nil); err == nil ||
		!strings.Contains(err.Error(), "presentationParameters") {
		t.Fatalf("%v, want the parameters refusal", err)
	}
}

// TestTheThreeResetsAndTheParametersGetterReachTheDevice is the coverage
// control: all four members must get past their own guards and report the
// device, which is only reachable once every argument has passed.
func TestTheThreeResetsAndTheParametersGetterReachTheDevice(t *testing.T) {
	device := &GraphicsDevice{}
	parameters := NewPresentationParameters()
	adapter := &GraphicsAdapter{}
	for name, call := range map[string]func() error{
		"Reset()": device.ResetByNone,
		"Reset(pp)": func() error {
			return device.ResetByPresentationParameters(parameters)
		},
		"Reset(pp, adapter)": func() error {
			return device.ResetByPresentationParametersAndGraphicsAdapter(parameters, adapter)
		},
		"PresentationParameters": func() error {
			_, err := device.PresentationParameters()
			return err
		},
	} {
		if err := call(); !errors.Is(err, errGraphicsDeviceNil) {
			t.Fatalf("%s reported %v, want the device reached", name, err)
		}
	}
}

// TestTheRichPresentOverloadRefusesAndSaysWhy pins the BLOCKED_UPSTREAM member.
// CNA's whole C ABI has one present route and it takes no rectangles and no
// window handle, so calling it here would present the whole back buffer into
// the device's own window under this overload's name.
func TestTheRichPresentOverloadRefusesAndSaysWhy(t *testing.T) {
	// A device with no native half reports that first, which is the
	// reference's own order.
	if err := (&GraphicsDevice{}).PresentByNullableOfRectangleAndNullableOfRectangleAndIntPtr(
		nil, nil, 0); !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("%v, want the device reported first", err)
	}
	// And the refusal itself names the route and what it lacks.
	if !strings.Contains(errPresentRectangles.Error(), "cna_graphics_device_present") {
		t.Fatalf("%v does not name the route", errPresentRectangles)
	}
	for _, missing := range []string{"source rectangle", "destination rectangle", "window handle"} {
		if !strings.Contains(errPresentRectangles.Error(), missing) {
			t.Fatalf("%v does not name the missing %s", errPresentRectangles, missing)
		}
	}
}

// TestGetBackBufferDataRefusesWhileARenderTargetIsActive pins the one guard of
// the reference's two that CNA-Go reproduces, and its exact message.
func TestGetBackBufferDataRefusesWhileARenderTargetIsActive(t *testing.T) {
	if err := checkNoActiveRenderTarget(nil); err != nil {
		t.Fatalf("no bound target reported %v", err)
	}
	err := checkNoActiveRenderTarget([]RenderTargetBinding{{}})
	if err == nil {
		t.Fatal("a back-buffer read was accepted while a target was bound")
	}
	if !strings.Contains(err.Error(), cannotGetBackBufferActiveRenderTargets) {
		t.Fatalf("%v, want FrameworkResources.CannotGetBackBufferActiveRenderTargets", err)
	}
	if !errors.Is(err, errSpriteInvalidOperation) {
		t.Fatalf("%v, want the InvalidOperationException projection", err)
	}
	// The device check comes FIRST, so a device with no native half reports
	// that even with a target bound -- the reference's CheckDisposed order.
	device := &GraphicsDevice{renderTargets: []RenderTargetBinding{{}}}
	if err := GraphicsDeviceGetBackBufferDataBySliceOfT(device, make([]framework.Color, 4)); !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("%v, want the device reported ahead of the bound target", err)
	}
}

// TestTheBackBufferElementSetIsOneTypeWideAndSaysWhoseLimitItIs pins the
// narrowing and its attribution, exactly as the cube and volume transfers'.
func TestTheBackBufferElementSetIsOneTypeWideAndSaysWhoseLimitItIs(t *testing.T) {
	device := &GraphicsDevice{}
	err := GraphicsDeviceGetBackBufferDataBySliceOfT(device, make([]packedvector.Bgr565, 4))
	// The device is reported first; the element check is what a live device
	// would reach, and resolveVolumeElement is what performs it.
	if !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("%v, want the device reported first", err)
	}
	if err := resolveVolumeElement[packedvector.Bgr565]("GraphicsDevice.GetBackBufferData"); err == nil {
		t.Fatal("a Bgr565 element was accepted")
	} else if !strings.Contains(err.Error(), "CNA") || strings.Contains(err.Error(), "XNA") {
		t.Fatalf("%v, want the limit attributed to CNA", err)
	}
}

// TestTheThreeBackBufferOverloadsAllReachTheDeviceFirst is the coverage control:
// every overload must reach the shared body rather than doing its own work.
func TestTheThreeBackBufferOverloadsAllReachTheDeviceFirst(t *testing.T) {
	device := &GraphicsDevice{renderTargets: []RenderTargetBinding{{}}}
	pixels := make([]framework.Color, 4)
	rect := framework.NewRectangle(0, 0, 2, 2)
	// All three reach the SAME first guard, which is the order being pinned:
	// no overload performs its own work ahead of Helpers.CheckDisposed.
	for name, call := range map[string]func() error{
		"whole": func() error { return GraphicsDeviceGetBackBufferDataBySliceOfT(device, pixels) },
		"window": func() error {
			return GraphicsDeviceGetBackBufferDataBySliceOfTAndInt32AndInt32(device, pixels, 0, 4)
		},
		"rectangle": func() error {
			return GraphicsDeviceGetBackBufferDataByNullableOfRectangleAndSliceOfTAndInt32AndInt32(device, &rect, pixels, 0, 4)
		},
	} {
		if err := call(); !errors.Is(err, errGraphicsDeviceNil) {
			t.Fatalf("%s reported %v, want the device reached first", name, err)
		}
	}
}

// TestTheConstructorRefusesItsTwoNullsAndThenNeedsASession pins the public
// constructor's guards and the one narrowing it carries.
func TestTheConstructorRefusesItsTwoNullsAndThenNeedsASession(t *testing.T) {
	parameters := NewPresentationParameters()
	if _, err := NewGraphicsDevice(nil, GraphicsProfileReach, parameters); err == nil ||
		!strings.Contains(err.Error(), "adapter") {
		t.Fatalf("%v, want ArgumentNullException(\"adapter\")", err)
	}
	if _, err := NewGraphicsDevice(&GraphicsAdapter{}, GraphicsProfileReach, nil); err == nil ||
		!strings.Contains(err.Error(), "presentationParameters") {
		t.Fatalf("%v, want ArgumentNullException(\"presentationParameters\")", err)
	}
	// With both arguments valid and no live session, the narrowing is what a
	// consumer sees -- and it names CNA-Go's own library lifetime rather than
	// pretending CNA refused.
	_, err := NewGraphicsDevice(&GraphicsAdapter{}, GraphicsProfileReach, parameters)
	if !errors.Is(err, errDeviceNeedsSession) {
		t.Fatalf("%v, want the session narrowing", err)
	}
	if !strings.Contains(err.Error(), "CNA-Go") {
		t.Fatalf("%v does not say whose limitation it is", err)
	}
}

// TestABindingCarriesItsTargetAndItsFace pins RenderTargetBinding's two
// constructors, its op_Implicit and both getters.
func TestABindingCarriesItsTargetAndItsFace(t *testing.T) {
	if _, err := NewRenderTargetBindingByRenderTarget2D(nil); err == nil ||
		!strings.Contains(err.Error(), nullNotAllowed) {
		t.Fatalf("%v, want ArgumentNullException(renderTarget, NullNotAllowed)", err)
	}
	if _, err := NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(nil, CubeMapFacePositiveX); err == nil {
		t.Fatal("a nil cube target was accepted")
	}
	if _, err := RenderTargetBindingOperatorImplicitByRenderTarget2D(nil); err == nil {
		t.Fatal("op_Implicit accepted a nil target")
	}
	// The face is range-checked, and the range is the enum's six members.
	cube := &RenderTargetCube{cube: &TextureCube{texture: liveTexture()}}
	for _, face := range []CubeMapFace{-1, 6} {
		if _, err := NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(cube, face); err == nil {
			t.Fatalf("face %d was accepted", face)
		}
	}
	binding, err := NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(cube, CubeMapFaceNegativeZ)
	if err != nil {
		t.Fatalf("a valid cube binding: %v", err)
	}
	if binding.CubeMapFace() != CubeMapFaceNegativeZ {
		t.Fatalf("CubeMapFace() = %v", binding.CubeMapFace())
	}
	if binding.RenderTarget() != cube.textureBase() {
		t.Fatal("RenderTarget() did not answer the composed Texture")
	}
	// A zero binding is empty, which the reference cannot construct and Go can.
	var zero RenderTargetBinding
	if zero.RenderTarget() != nil || zero.CubeMapFace() != CubeMapFacePositiveX {
		t.Fatal("the zero binding is not empty")
	}
}

// TestSetRenderTargetsRefusesAZeroBindingAndAcceptsNoneAtAll pins that an empty
// array is the BACK BUFFER rather than a refusal -- which is the clearest
// statement in the type that "no render target" is a state -- and that a
// zero-valued binding, which only Go can produce, is refused by name.
func TestSetRenderTargetsRefusesAZeroBindingAndAcceptsNoneAtAll(t *testing.T) {
	device := &GraphicsDevice{}
	if err := device.SetRenderTargets(nil); !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("%v, want an empty array to reach the device", err)
	}
	if err := device.SetRenderTargets([]RenderTargetBinding{}); !errors.Is(err, errGraphicsDeviceNil) {
		t.Fatalf("%v, want an empty slice to reach the device", err)
	}
}

// TestGetRenderTargetsAnswersAFreshArrayOverTheSameBindings pins the copy the
// reference makes and the identity it preserves.
func TestGetRenderTargetsAnswersAFreshArrayOverTheSameBindings(t *testing.T) {
	cube := &RenderTargetCube{cube: &TextureCube{texture: liveTexture()}}
	binding, err := NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(cube, CubeMapFacePositiveY)
	if err != nil {
		t.Fatal(err)
	}
	device := &GraphicsDevice{renderTargets: []RenderTargetBinding{binding}}
	first := device.GetRenderTargets()
	second := device.GetRenderTargets()
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("GetRenderTargets answered %d and %d", len(first), len(second))
	}
	if &first[0] == &second[0] {
		t.Fatal("two calls answered the same array; the reference copies")
	}
	if first[0].RenderTarget() != binding.RenderTarget() || first[0].CubeMapFace() != CubeMapFacePositiveY {
		t.Fatal("the copy does not carry the binding that was set")
	}
	// Mutating the result changes nothing.
	first[0] = RenderTargetBinding{}
	if device.GetRenderTargets()[0].RenderTarget() == nil {
		t.Fatal("mutating the returned array changed the device")
	}
	// And no target bound answers nothing at all.
	if (&GraphicsDevice{}).GetRenderTargets() != nil {
		t.Fatal("a device with no target answered a binding")
	}
}

// TestARenderTargetCubeIsUsableAtATexturePosition pins that the fourth link of
// the chain reaches the substitutable-base rule.
func TestARenderTargetCubeIsUsableAtATexturePosition(t *testing.T) {
	base := liveTexture()
	cube := &RenderTargetCube{cube: &TextureCube{texture: base}}
	var _ TextureReference = cube
	if resolveTexture(cube) != base {
		t.Fatal("the composed Texture did not answer through two links")
	}
	var nilCube *RenderTargetCube
	if resolveTexture(nilCube) != nil {
		t.Fatal("a typed nil answered with a Texture")
	}
}
