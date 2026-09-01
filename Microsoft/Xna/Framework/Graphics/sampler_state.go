package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// SamplerState is Microsoft.Xna.Framework.Graphics.SamplerState: seven sampler
// values, six static presets, and the freeze rule the whole state family
// shares. See graphics_state_object.go.
//
// # Every default is read off SetDefaults
//
//	cachedFilter                  = TextureFilter.Linear      (ldc.i4.0)
//	cachedAddressU                = TextureAddressMode.Wrap   (ldc.i4.0)
//	cachedAddressV                = TextureAddressMode.Wrap   (ldc.i4.0)
//	cachedAddressW                = TextureAddressMode.Wrap   (ldc.i4.0)
//	cachedMaxAnisotropy           = 4                         (ldc.i4.4)
//	cachedMaxMipLevel             = 0                         (ldc.i4.0)
//	cachedMipMapLevelOfDetailBias = 0.0                       (ldc.r4)
type SamplerState struct {
	// resource is the composed GraphicsResource, carrying no native handle.
	resource *GraphicsResource
	// isBound is the reference's own freeze flag.
	isBound bool

	filter                  TextureFilter
	addressU                TextureAddressMode
	addressV                TextureAddressMode
	addressW                TextureAddressMode
	maxAnisotropy           int32
	maxMipLevel             int32
	mipMapLevelOfDetailBias float32
}

// NewSamplerState is SamplerState::.ctor(): SetDefaults, then isBound = false.
func NewSamplerState() *SamplerState {
	state := &SamplerState{resource: newStateResource()}
	state.setDefaults()
	state.resource.bindDerived(state)
	return state
}

func (s *SamplerState) setDefaults() {
	s.filter = TextureFilterLinear
	s.addressU = TextureAddressModeWrap
	s.addressV = TextureAddressModeWrap
	s.addressW = TextureAddressModeWrap
	s.maxAnisotropy = 4
	s.maxMipLevel = 0
	s.mipMapLevelOfDetailBias = 0
}

// newPresetSamplerState is SamplerState::.ctor(TextureFilter,
// TextureAddressMode, String), the PRIVATE constructor the six static fields
// use. The one address mode is applied to ALL THREE coordinates.
func newPresetSamplerState(filter TextureFilter, address TextureAddressMode, name string) *SamplerState {
	state := NewSamplerState()
	state.filter = filter
	state.addressU = address
	state.addressV = address
	state.addressW = address
	state.resource.SetName(name)
	state.isBound = true
	return state
}

// The six static instances, with the exact arguments the class initializer
// passes.
//
//	PointWrap        (Point,       Wrap)  "SamplerState.PointWrap"
//	PointClamp       (Point,       Clamp) "SamplerState.PointClamp"
//	LinearWrap       (Linear,      Wrap)  "SamplerState.LinearWrap"
//	LinearClamp      (Linear,      Clamp) "SamplerState.LinearClamp"
//	AnisotropicWrap  (Anisotropic, Wrap)  "SamplerState.AnisotropicWrap"
//	AnisotropicClamp (Anisotropic, Clamp) "SamplerState.AnisotropicClamp"
var (
	samplerStatePointWrap        = newPresetSamplerState(TextureFilterPoint, TextureAddressModeWrap, "SamplerState.PointWrap")
	samplerStatePointClamp       = newPresetSamplerState(TextureFilterPoint, TextureAddressModeClamp, "SamplerState.PointClamp")
	samplerStateLinearWrap       = newPresetSamplerState(TextureFilterLinear, TextureAddressModeWrap, "SamplerState.LinearWrap")
	samplerStateLinearClamp      = newPresetSamplerState(TextureFilterLinear, TextureAddressModeClamp, "SamplerState.LinearClamp")
	samplerStateAnisotropicWrap  = newPresetSamplerState(TextureFilterAnisotropic, TextureAddressModeWrap, "SamplerState.AnisotropicWrap")
	samplerStateAnisotropicClamp = newPresetSamplerState(TextureFilterAnisotropic, TextureAddressModeClamp, "SamplerState.AnisotropicClamp")
)

// SamplerStatePointWrap is SamplerState::PointWrap.
func SamplerStatePointWrap() *SamplerState { return samplerStatePointWrap }

// SamplerStatePointClamp is SamplerState::PointClamp.
func SamplerStatePointClamp() *SamplerState { return samplerStatePointClamp }

// SamplerStateLinearWrap is SamplerState::LinearWrap.
func SamplerStateLinearWrap() *SamplerState { return samplerStateLinearWrap }

// SamplerStateLinearClamp is SamplerState::LinearClamp.
func SamplerStateLinearClamp() *SamplerState { return samplerStateLinearClamp }

// SamplerStateAnisotropicWrap is SamplerState::AnisotropicWrap.
func SamplerStateAnisotropicWrap() *SamplerState { return samplerStateAnisotropicWrap }

// SamplerStateAnisotropicClamp is SamplerState::AnisotropicClamp.
func SamplerStateAnisotropicClamp() *SamplerState { return samplerStateAnisotropicClamp }

// clrTypeName is System.Object::ToString's answer for a SamplerState.
func (s *SamplerState) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.SamplerState"
}

// The seven value properties. Every getter is one `ldfld`; every setter is
// `ThrowIfBound(); stfld`.

func (s *SamplerState) Filter() TextureFilter { return s.filter }

func (s *SamplerState) SetFilter(value TextureFilter) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.filter = value
	return nil
}

func (s *SamplerState) AddressU() TextureAddressMode { return s.addressU }

func (s *SamplerState) SetAddressU(value TextureAddressMode) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.addressU = value
	return nil
}

func (s *SamplerState) AddressV() TextureAddressMode { return s.addressV }

func (s *SamplerState) SetAddressV(value TextureAddressMode) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.addressV = value
	return nil
}

func (s *SamplerState) AddressW() TextureAddressMode { return s.addressW }

func (s *SamplerState) SetAddressW(value TextureAddressMode) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.addressW = value
	return nil
}

func (s *SamplerState) MaxAnisotropy() int32 { return s.maxAnisotropy }

func (s *SamplerState) SetMaxAnisotropy(value int32) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.maxAnisotropy = value
	return nil
}

func (s *SamplerState) MaxMipLevel() int32 { return s.maxMipLevel }

func (s *SamplerState) SetMaxMipLevel(value int32) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.maxMipLevel = value
	return nil
}

func (s *SamplerState) MipMapLevelOfDetailBias() float32 { return s.mipMapLevelOfDetailBias }

func (s *SamplerState) SetMipMapLevelOfDetailBias(value float32) error {
	if err := throwIfBound(s.isBound, s.clrTypeName()); err != nil {
		return err
	}
	s.mipMapLevelOfDetailBias = value
	return nil
}

// The nine members inherited from GraphicsResource, forwarded.

func (s *SamplerState) GraphicsDevice() *GraphicsDevice { return s.resource.GraphicsDevice() }
func (s *SamplerState) Name() string                    { return s.resource.Name() }
func (s *SamplerState) SetName(value string)            { s.resource.SetName(value) }
func (s *SamplerState) Tag() any                        { return s.resource.Tag() }
func (s *SamplerState) SetTag(value any)                { s.resource.SetTag(value) }
func (s *SamplerState) IsDisposed() bool                { return s.resource.IsDisposed() }
func (s *SamplerState) ToString() string                { return s.resource.ToString() }

func (s *SamplerState) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resource.AddDisposingHandler(handler)
}

func (s *SamplerState) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	return s.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (s *SamplerState) DisposeByNone() error { return s.DisposeByBoolean(true) }

// DisposeByBoolean is SamplerState::Dispose(bool), which releases nothing.
func (s *SamplerState) DisposeByBoolean(disposing bool) error {
	return disposeStateObject(s.resource, disposing)
}
