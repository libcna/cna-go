package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestStateObjectDefaultsAreTheILsOwnConstants pins every value SetDefaults
// stores, for all four state objects. A default that drifted would change what
// a consumer's `new BlendState()` means without any test noticing.
func TestStateObjectDefaultsAreTheILsOwnConstants(t *testing.T) {
	blend := NewBlendState()
	if blend.ColorSourceBlend() != BlendOne || blend.ColorDestinationBlend() != BlendZero ||
		blend.AlphaSourceBlend() != BlendOne || blend.AlphaDestinationBlend() != BlendZero {
		t.Fatalf("blend factors = %v/%v %v/%v, want One/Zero on both channels",
			blend.ColorSourceBlend(), blend.ColorDestinationBlend(),
			blend.AlphaSourceBlend(), blend.AlphaDestinationBlend())
	}
	if blend.ColorBlendFunction() != BlendFunctionAdd || blend.AlphaBlendFunction() != BlendFunctionAdd {
		t.Fatal("blend functions default to Add on both channels")
	}
	for index, channels := range []ColorWriteChannels{
		blend.ColorWriteChannels(), blend.ColorWriteChannels1(),
		blend.ColorWriteChannels2(), blend.ColorWriteChannels3(),
	} {
		if channels != ColorWriteChannelsAll {
			t.Fatalf("ColorWriteChannels%d = %v, want All (ldc.i4.s 15)", index, channels)
		}
	}
	if blend.BlendFactor() != framework.ColorWhite() {
		t.Fatalf("BlendFactor = %+v, want Color.White", blend.BlendFactor())
	}
	if blend.MultiSampleMask() != -1 {
		t.Fatalf("MultiSampleMask = %d, want -1 (ldc.i4.m1)", blend.MultiSampleMask())
	}

	depth := NewDepthStencilState()
	if !depth.DepthBufferEnable() || !depth.DepthBufferWriteEnable() {
		t.Fatal("depth testing and depth writes both default to TRUE")
	}
	if depth.DepthBufferFunction() != CompareFunctionLessEqual {
		t.Fatalf("DepthBufferFunction = %v, want LessEqual (ldc.i4.3)", depth.DepthBufferFunction())
	}
	if depth.StencilEnable() || depth.TwoSidedStencilMode() {
		t.Fatal("stencil testing and two-sided mode both default to false")
	}
	if depth.StencilFunction() != CompareFunctionAlways ||
		depth.CounterClockwiseStencilFunction() != CompareFunctionAlways {
		t.Fatal("both stencil comparison functions default to Always")
	}
	for name, operation := range map[string]StencilOperation{
		"StencilPass": depth.StencilPass(), "StencilFail": depth.StencilFail(),
		"StencilDepthBufferFail":                 depth.StencilDepthBufferFail(),
		"CounterClockwiseStencilPass":            depth.CounterClockwiseStencilPass(),
		"CounterClockwiseStencilFail":            depth.CounterClockwiseStencilFail(),
		"CounterClockwiseStencilDepthBufferFail": depth.CounterClockwiseStencilDepthBufferFail(),
	} {
		if operation != StencilOperationKeep {
			t.Fatalf("%s = %v, want Keep", name, operation)
		}
	}
	if depth.StencilMask() != -1 || depth.StencilWriteMask() != -1 || depth.ReferenceStencil() != 0 {
		t.Fatalf("stencil masks = %d/%d reference %d, want -1/-1 and 0",
			depth.StencilMask(), depth.StencilWriteMask(), depth.ReferenceStencil())
	}

	raster := NewRasterizerState()
	// The two that surprise a reader: culling is counter-clockwise, not none,
	// and multisample antialiasing is ON.
	if raster.CullMode() != CullModeCullCounterClockwiseFace {
		t.Fatalf("CullMode = %v, want CullCounterClockwiseFace (ldc.i4.2)", raster.CullMode())
	}
	if !raster.MultiSampleAntiAlias() {
		t.Fatal("MultiSampleAntiAlias defaults to TRUE (ldc.i4.1)")
	}
	if raster.FillMode() != FillModeSolid || raster.ScissorTestEnable() ||
		raster.DepthBias() != 0 || raster.SlopeScaleDepthBias() != 0 {
		t.Fatal("the other four rasterizer defaults moved")
	}

	sampler := NewSamplerState()
	if sampler.Filter() != TextureFilterLinear {
		t.Fatalf("Filter = %v, want Linear", sampler.Filter())
	}
	if sampler.AddressU() != TextureAddressModeWrap || sampler.AddressV() != TextureAddressModeWrap ||
		sampler.AddressW() != TextureAddressModeWrap {
		t.Fatal("all three address modes default to Wrap")
	}
	if sampler.MaxAnisotropy() != 4 {
		t.Fatalf("MaxAnisotropy = %d, want 4 (ldc.i4.4)", sampler.MaxAnisotropy())
	}
	if sampler.MaxMipLevel() != 0 || sampler.MipMapLevelOfDetailBias() != 0 {
		t.Fatal("MaxMipLevel and MipMapLevelOfDetailBias default to zero")
	}
}

// TestStaticStateObjectsAreFrozenFromConstruction is the fact the documentation
// sentence does not say.
//
// The message reads "State objects become read-only the first time they are
// bound to a GraphicsDevice", and the static instances are built by a private
// constructor whose LAST statement is `isBound = true`. So a preset is
// read-only on an object no device has ever seen.
func TestStaticStateObjectsAreFrozenFromConstruction(t *testing.T) {
	if err := BlendStateAlphaBlend().SetColorSourceBlend(BlendZero); !errors.Is(err, errStateObjectInvalidOperation) {
		t.Fatalf("BlendState.AlphaBlend accepted a write: %v", err)
	}
	if err := DepthStencilStateDefault().SetDepthBufferEnable(false); !errors.Is(err, errStateObjectInvalidOperation) {
		t.Fatalf("DepthStencilState.Default accepted a write: %v", err)
	}
	if err := RasterizerStateCullNone().SetFillMode(FillModeWireFrame); !errors.Is(err, errStateObjectInvalidOperation) {
		t.Fatalf("RasterizerState.CullNone accepted a write: %v", err)
	}
	if err := SamplerStateLinearClamp().SetMaxAnisotropy(16); !errors.Is(err, errStateObjectInvalidOperation) {
		t.Fatalf("SamplerState.LinearClamp accepted a write: %v", err)
	}
	// A consumer's own instance is mutable, which is the other half of the
	// same rule: the public constructor ends with `isBound = false`.
	own := NewBlendState()
	if err := own.SetColorSourceBlend(BlendZero); err != nil {
		t.Fatalf("a fresh BlendState refused a write: %v", err)
	}
	if own.ColorSourceBlend() != BlendZero {
		t.Fatal("the write did not take")
	}
}

// TestTheFrozenRefusalCarriesMicrosoftsFormattedSentence pins the exact
// message, including the substitution String.Format performs at BOTH `{0}`
// placeholders with `typeof(T).Name`.
func TestTheFrozenRefusalCarriesMicrosoftsFormattedSentence(t *testing.T) {
	err := BlendStateOpaque().SetMultiSampleMask(0)
	if err == nil {
		t.Fatal("a frozen state accepted a write")
	}
	want := "Cannot change read-only BlendState. State objects become read-only the first time they " +
		"are bound to a GraphicsDevice. To change property values, create a new BlendState instance."
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal = %q, want FrameworkResources.BoundStateObject formatted with the type name", err)
	}
	// And the substitution really is per type, not a constant.
	sampler := SamplerStatePointWrap().SetFilter(TextureFilterLinear)
	if sampler == nil || !strings.Contains(sampler.Error(), "read-only SamplerState.") {
		t.Fatalf("SamplerState refusal = %v, want its own type name substituted", sampler)
	}
}

// TestStaticStateObjectsHaveStableIdentityAndTheReferencesNames pins what a
// `public static initonly` field promises: one object, for the life of the
// process, carrying the name the class initializer gave it.
func TestStaticStateObjectsHaveStableIdentityAndTheReferencesNames(t *testing.T) {
	if BlendStateAdditive() != BlendStateAdditive() {
		t.Fatal("two reads of BlendState.Additive returned different objects")
	}
	for name, got := range map[string]string{
		"BlendState.Opaque":                    BlendStateOpaque().Name(),
		"BlendState.AlphaBlend":                BlendStateAlphaBlend().Name(),
		"BlendState.Additive":                  BlendStateAdditive().Name(),
		"BlendState.NonPremultiplied":          BlendStateNonPremultiplied().Name(),
		"DepthStencilState.None":               DepthStencilStateNone().Name(),
		"DepthStencilState.Default":            DepthStencilStateDefault().Name(),
		"DepthStencilState.DepthRead":          DepthStencilStateDepthRead().Name(),
		"RasterizerState.CullNone":             RasterizerStateCullNone().Name(),
		"RasterizerState.CullClockwise":        RasterizerStateCullClockwise().Name(),
		"RasterizerState.CullCounterClockwise": RasterizerStateCullCounterClockwise().Name(),
		"SamplerState.PointWrap":               SamplerStatePointWrap().Name(),
		"SamplerState.PointClamp":              SamplerStatePointClamp().Name(),
		"SamplerState.LinearWrap":              SamplerStateLinearWrap().Name(),
		"SamplerState.LinearClamp":             SamplerStateLinearClamp().Name(),
		"SamplerState.AnisotropicWrap":         SamplerStateAnisotropicWrap().Name(),
		"SamplerState.AnisotropicClamp":        SamplerStateAnisotropicClamp().Name(),
	} {
		if got != name {
			t.Errorf("a preset is named %q, want %q", got, name)
		}
	}
	// A named GraphicsResource answers ToString with its Name, so every preset
	// answers with the reference's own string rather than with its type.
	if got := RasterizerStateCullClockwise().ToString(); got != "RasterizerState.CullClockwise" {
		t.Fatalf("preset ToString = %q", got)
	}
	// An UNNAMED state object falls back to the runtime type, through the
	// composed GraphicsResource and its CLR `this`.
	if got := NewSamplerState().ToString(); got != "Microsoft.Xna.Framework.Graphics.SamplerState" {
		t.Fatalf("fresh SamplerState ToString = %q", got)
	}
}

// TestTheStaticPresetValuesAreTheClassInitializersArguments pins what each
// preset actually holds. A preset built from the wrong pair would still be a
// valid state object and would silently blend, cull or filter differently.
func TestTheStaticPresetValuesAreTheClassInitializersArguments(t *testing.T) {
	for name, preset := range map[string]struct {
		state        *BlendState
		source, dest Blend
	}{
		"Opaque":           {BlendStateOpaque(), BlendOne, BlendZero},
		"AlphaBlend":       {BlendStateAlphaBlend(), BlendOne, BlendInverseSourceAlpha},
		"Additive":         {BlendStateAdditive(), BlendSourceAlpha, BlendOne},
		"NonPremultiplied": {BlendStateNonPremultiplied(), BlendSourceAlpha, BlendInverseSourceAlpha},
	} {
		// The private constructor applies the pair to BOTH channels.
		if preset.state.ColorSourceBlend() != preset.source || preset.state.AlphaSourceBlend() != preset.source {
			t.Errorf("%s source = %v/%v, want %v on both channels", name,
				preset.state.ColorSourceBlend(), preset.state.AlphaSourceBlend(), preset.source)
		}
		if preset.state.ColorDestinationBlend() != preset.dest || preset.state.AlphaDestinationBlend() != preset.dest {
			t.Errorf("%s destination = %v/%v, want %v on both channels", name,
				preset.state.ColorDestinationBlend(), preset.state.AlphaDestinationBlend(), preset.dest)
		}
	}
	for name, preset := range map[string]struct {
		state               *DepthStencilState
		enable, writeEnable bool
	}{
		"None":      {DepthStencilStateNone(), false, false},
		"Default":   {DepthStencilStateDefault(), true, true},
		"DepthRead": {DepthStencilStateDepthRead(), true, false},
	} {
		if preset.state.DepthBufferEnable() != preset.enable ||
			preset.state.DepthBufferWriteEnable() != preset.writeEnable {
			t.Errorf("%s = %t/%t, want %t/%t", name, preset.state.DepthBufferEnable(),
				preset.state.DepthBufferWriteEnable(), preset.enable, preset.writeEnable)
		}
	}
	for name, preset := range map[string]struct {
		state *RasterizerState
		cull  CullMode
	}{
		"CullNone":             {RasterizerStateCullNone(), CullModeNone},
		"CullClockwise":        {RasterizerStateCullClockwise(), CullModeCullClockwiseFace},
		"CullCounterClockwise": {RasterizerStateCullCounterClockwise(), CullModeCullCounterClockwiseFace},
	} {
		if preset.state.CullMode() != preset.cull {
			t.Errorf("%s = %v, want %v", name, preset.state.CullMode(), preset.cull)
		}
	}
	for name, preset := range map[string]struct {
		state   *SamplerState
		filter  TextureFilter
		address TextureAddressMode
	}{
		"PointWrap":        {SamplerStatePointWrap(), TextureFilterPoint, TextureAddressModeWrap},
		"PointClamp":       {SamplerStatePointClamp(), TextureFilterPoint, TextureAddressModeClamp},
		"LinearWrap":       {SamplerStateLinearWrap(), TextureFilterLinear, TextureAddressModeWrap},
		"LinearClamp":      {SamplerStateLinearClamp(), TextureFilterLinear, TextureAddressModeClamp},
		"AnisotropicWrap":  {SamplerStateAnisotropicWrap(), TextureFilterAnisotropic, TextureAddressModeWrap},
		"AnisotropicClamp": {SamplerStateAnisotropicClamp(), TextureFilterAnisotropic, TextureAddressModeClamp},
	} {
		if preset.state.Filter() != preset.filter {
			t.Errorf("%s filter = %v, want %v", name, preset.state.Filter(), preset.filter)
		}
		// The one address mode goes to ALL THREE coordinates.
		if preset.state.AddressU() != preset.address || preset.state.AddressV() != preset.address ||
			preset.state.AddressW() != preset.address {
			t.Errorf("%s addresses = %v/%v/%v, want %v on all three", name,
				preset.state.AddressU(), preset.state.AddressV(), preset.state.AddressW(), preset.address)
		}
	}
}

// TestAStateObjectIsAGraphicsResourceWithNoNativeHandle pins the ownership.
func TestAStateObjectIsAGraphicsResourceWithNoNativeHandle(t *testing.T) {
	state := NewRasterizerState()
	if state.GraphicsDevice() != nil {
		t.Fatal("a state object has no device until one binds it")
	}
	if state.IsDisposed() {
		t.Fatal("a fresh state object reports disposed")
	}
	state.SetName("mine")
	state.SetTag(7)
	if state.Name() != "mine" || state.Tag() != 7 {
		t.Fatal("Name and Tag are managed storage on a state object")
	}
	raises := 0
	if _, err := state.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		raises++
		if sender != any(state) {
			return errors.New("Disposing announced something else")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if err := state.DisposeByNone(); err != nil {
		t.Fatalf("second Dispose: %v", err)
	}
	if raises != 1 || !state.IsDisposed() {
		t.Fatalf("Disposing raised %d times, IsDisposed=%t", raises, state.IsDisposed())
	}
}
