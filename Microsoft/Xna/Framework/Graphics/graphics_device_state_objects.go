package graphics

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 60 — the device's three state-object properties.
// ---------------------------------------------------------------------------

// # The getter returns the OBJECT, not a description of it
//
//	get_BlendState  ldarg.0; ldfld cachedBlendState; ret
//
// It answers with the very object the setter was given, and
// `InitializeDeviceState` stores NULL into all three, so a device answers nil
// until something sets one. That identity is the whole reason these getters
// cannot read CNA back: CNA holds the VALUES -- CNA_BlendState is a POD -- and
// a fresh object built from them would fail the first `device.BlendState() ==
// myState` a consumer writes.
//
// The cache lives on the GraphicsDevice facade, which Foundation 49 already
// made ONE object per manager per native generation, so the identity is as
// stable as the reference's field.
//
// # The setter is where the freeze happens
//
//	set_BlendState(value)
//	  if (value == null)
//	      throw new ArgumentNullException("value", FrameworkResources.NullNotAllowed);
//	  if (value == cachedBlendState && !blendStateDirty) return;
//	  ... end an active effect pass that carries the blend flag ...
//	  value.Apply(this);            // <- sets _parent and isBound = true
//	  cachedBlendState = value;
//	  cachedBlendFactor = value.cachedBlendFactor;
//	  cachedMultiSampleMask = value.cachedMultiSampleMask;
//
// `Apply` throws ObjectDisposedException first if the state object is disposed,
// and its first act on a state bound to a different device is to store the new
// parent and raise `isBound`. So a state object that has been handed to any
// device is read-only from then on, which is exactly what the freeze rule says
// and is reproduced here rather than approximated.
//
// The effect-pass step has no counterpart: CNA-Go maps no effect subsystem, so
// there is no active pass to end. It is recorded rather than silently skipped.

// nullNotAllowedParameter builds the reference's ArgumentNullException for a
// named parameter.
func nullNotAllowedParameter(name string) error {
	return fmt.Errorf("%w: %s: %s", errGraphicsResourceArgumentNull, name, nullNotAllowed)
}

// objectDisposedState is the ObjectDisposedException Apply throws first, whose
// message is `typeof(T).Name` -- the CLR's own one-argument constructor.
func objectDisposedState(clrTypeName string) error {
	short := clrTypeName[len("Microsoft.Xna.Framework.Graphics."):]
	return fmt.Errorf("%w: %s", interop.ErrDisposed, short)
}

// BlendState is GraphicsDevice::get_BlendState.
func (d *GraphicsDevice) BlendState() (*BlendState, error) {
	if d == nil || d.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if err := d.device.Live(); err != nil {
		return nil, err
	}
	return d.blendState, nil
}

// SetBlendState is GraphicsDevice::set_BlendState.
func (d *GraphicsDevice) SetBlendState(value *BlendState) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if value == nil {
		return nullNotAllowedParameter("value")
	}
	if value.IsDisposed() {
		return objectDisposedState(value.clrTypeName())
	}
	if err := d.device.SetBlendState(value.interopValue()); err != nil {
		return err
	}
	// Apply's own two stores, in its order: the parent first, then the freeze.
	value.resource.device = d
	value.isBound = true
	d.blendState = value
	return nil
}

// DepthStencilState is GraphicsDevice::get_DepthStencilState.
func (d *GraphicsDevice) DepthStencilState() (*DepthStencilState, error) {
	if d == nil || d.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if err := d.device.Live(); err != nil {
		return nil, err
	}
	return d.depthStencilState, nil
}

// SetDepthStencilState is GraphicsDevice::set_DepthStencilState, the same shape
// as set_BlendState over the depth and stencil values.
func (d *GraphicsDevice) SetDepthStencilState(value *DepthStencilState) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if value == nil {
		return nullNotAllowedParameter("value")
	}
	if value.IsDisposed() {
		return objectDisposedState(value.clrTypeName())
	}
	if err := d.device.SetDepthStencilState(value.interopValue()); err != nil {
		return err
	}
	value.resource.device = d
	value.isBound = true
	d.depthStencilState = value
	return nil
}

// RasterizerState is GraphicsDevice::get_RasterizerState.
func (d *GraphicsDevice) RasterizerState() (*RasterizerState, error) {
	if d == nil || d.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if err := d.device.Live(); err != nil {
		return nil, err
	}
	return d.rasterizerState, nil
}

// SetRasterizerState is GraphicsDevice::set_RasterizerState.
func (d *GraphicsDevice) SetRasterizerState(value *RasterizerState) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	if value == nil {
		return nullNotAllowedParameter("value")
	}
	if value.IsDisposed() {
		return objectDisposedState(value.clrTypeName())
	}
	if err := d.device.SetRasterizerState(value.interopValue()); err != nil {
		return err
	}
	value.resource.device = d
	value.isBound = true
	d.rasterizerState = value
	return nil
}

// The four descriptor conversions. Each is a field-for-field copy into the
// flattened interop value, and each lives beside the projection rather than in
// interop so the enum identities stay in the package that declares them.

func (b *BlendState) interopValue() interop.BlendStateValue {
	return interop.BlendStateValue{
		AlphaBlendFunction:    uint32(b.alphaBlendFunction),
		AlphaDestinationBlend: uint32(b.alphaDestinationBlend),
		AlphaSourceBlend:      uint32(b.alphaSourceBlend),
		ColorBlendFunction:    uint32(b.colorBlendFunction),
		ColorDestinationBlend: uint32(b.colorDestinationBlend),
		ColorSourceBlend:      uint32(b.colorSourceBlend),
		ColorWriteChannels:    uint32(b.colorWriteChannels),
		ColorWriteChannels1:   uint32(b.colorWriteChannels1),
		ColorWriteChannels2:   uint32(b.colorWriteChannels2),
		ColorWriteChannels3:   uint32(b.colorWriteChannels3),
		BlendFactorR:          b.blendFactor.R(),
		BlendFactorG:          b.blendFactor.G(),
		BlendFactorB:          b.blendFactor.B(),
		BlendFactorA:          b.blendFactor.A(),
		MultiSampleMask:       b.multiSampleMask,
	}
}

func (d *DepthStencilState) interopValue() interop.DepthStencilStateValue {
	return interop.DepthStencilStateValue{
		DepthBufferEnable:                      d.depthBufferEnable,
		DepthBufferWriteEnable:                 d.depthBufferWriteEnable,
		StencilEnable:                          d.stencilEnable,
		TwoSidedStencilMode:                    d.twoSidedStencilMode,
		DepthBufferFunction:                    uint32(d.depthBufferFunction),
		StencilFunction:                        uint32(d.stencilFunction),
		StencilFail:                            uint32(d.stencilFail),
		StencilDepthBufferFail:                 uint32(d.stencilDepthBufferFail),
		StencilPass:                            uint32(d.stencilPass),
		CounterClockwiseStencilFunction:        uint32(d.counterClockwiseStencilFunction),
		CounterClockwiseStencilFail:            uint32(d.counterClockwiseStencilFail),
		CounterClockwiseStencilDepthBufferFail: uint32(d.counterClockwiseStencilDepthBufferFail),
		CounterClockwiseStencilPass:            uint32(d.counterClockwiseStencilPass),
		StencilMask:                            d.stencilMask,
		StencilWriteMask:                       d.stencilWriteMask,
		ReferenceStencil:                       d.referenceStencil,
	}
}

func (r *RasterizerState) interopValue() interop.RasterizerStateValue {
	return interop.RasterizerStateValue{
		CullMode:             uint32(r.cullMode),
		FillMode:             uint32(r.fillMode),
		DepthBias:            r.depthBias,
		SlopeScaleDepthBias:  r.slopeScaleDepthBias,
		MultiSampleAntiAlias: r.multiSampleAntiAlias,
		ScissorTestEnable:    r.scissorTestEnable,
	}
}

func (s *SamplerState) interopValue() interop.SamplerStateValue {
	return interop.SamplerStateValue{
		AddressU:                uint32(s.addressU),
		AddressV:                uint32(s.addressV),
		AddressW:                uint32(s.addressW),
		Filter:                  uint32(s.filter),
		MaxAnisotropy:           s.maxAnisotropy,
		MaxMipLevel:             s.maxMipLevel,
		MipMapLevelOfDetailBias: s.mipMapLevelOfDetailBias,
	}
}
