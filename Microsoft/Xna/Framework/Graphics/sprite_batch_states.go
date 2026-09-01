package graphics

import (
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 60 — the two state-carrying SpriteBatch.Begin overloads.
// ---------------------------------------------------------------------------

// # A null state argument is a DEFAULT, not a refusal
//
// Every Begin overload funnels into the seven-argument one, which stores its
// arguments and then calls SetRenderState. That method is where the nulls are
// resolved, and its IL names each substitute:
//
//	blendState        ?? BlendState.AlphaBlend
//	depthStencilState ?? DepthStencilState.None
//	rasterizerState   ?? RasterizerState.CullCounterClockwise
//	samplerState      ?? SamplerState.LinearClamp   (applied as SamplerStates[0])
//
// CNA documents exactly the same four substitutes for its own null arguments,
// which is a useful agreement but not the authority: these are read off
// SetRenderState.
//
// # SetRenderState applies them TO THE DEVICE
//
// It calls `GraphicsDevice::set_BlendState` and its two siblings, so after
// `spriteBatch.Begin(mode, myBlend)` the device answers `myBlend` from
// `BlendState` and `myBlend` is frozen. That is observable and is reproduced.
//
// CNA takes all four descriptors on one route,
// `cna_sprite_batch_begin_with_states`, so the values reach the renderer ONCE
// rather than through four separate device calls followed by a fifth. The
// managed half of set_BlendState -- the device's cached object and the state's
// freeze -- is performed here so the observable result is the reference's,
// and the native half is performed by the one route that needs it.

// BeginBySpriteSortModeAndBlendState is
// SpriteBatch::Begin(SpriteSortMode, BlendState).
//
// It forwards to the seven-argument Begin with four nulls and Matrix.Identity,
// so the sampler, depth-stencil and rasterizer states are the defaults
// SetRenderState substitutes.
func (b *SpriteBatch) BeginBySpriteSortModeAndBlendState(sortMode SpriteSortMode, blendState *BlendState) error {
	return b.beginWithStates(sortMode, blendState, nil, nil, nil)
}

// BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerState
// is SpriteBatch::Begin(SpriteSortMode, BlendState, SamplerState,
// DepthStencilState, RasterizerState), which forwards to the seven-argument
// Begin with a null Effect and Matrix.Identity.
func (b *SpriteBatch) BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerState(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState,
) error {
	return b.beginWithStates(sortMode, blendState, samplerState, depthStencilState, rasterizerState)
}

// beginWithStates is the shared body: the pair guard, the null substitutions,
// the native call, and only then the managed effects.
func (b *SpriteBatch) beginWithStates(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState,
) error {
	if b == nil {
		return interop.ErrDisposed
	}
	// The reference's own first statement, ahead of any state the object holds.
	if b.inBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, endMustBeCalledBeforeBegin)
	}
	if b.resource() == nil {
		return interop.ErrDisposed
	}
	if blendState == nil {
		blendState = BlendStateAlphaBlend()
	}
	if samplerState == nil {
		samplerState = SamplerStateLinearClamp()
	}
	if depthStencilState == nil {
		depthStencilState = DepthStencilStateNone()
	}
	if rasterizerState == nil {
		rasterizerState = RasterizerStateCullCounterClockwise()
	}
	for _, disposed := range []struct {
		is   bool
		name string
	}{
		{blendState.IsDisposed(), blendState.clrTypeName()},
		{samplerState.IsDisposed(), samplerState.clrTypeName()},
		{depthStencilState.IsDisposed(), depthStencilState.clrTypeName()},
		{rasterizerState.IsDisposed(), rasterizerState.clrTypeName()},
	} {
		if disposed.is {
			return objectDisposedState(disposed.name)
		}
	}
	if err := b.resource().BeginSpriteBatchWithStates(
		uint32(sortMode), blendState.interopValue(), samplerState.interopValue(),
		depthStencilState.interopValue(), rasterizerState.interopValue()); err != nil {
		return err
	}
	// SetRenderState's managed half, in its order: the device caches the object
	// and the object freezes. It runs only after CNA accepts, for the reason the
	// pair flag does -- a refused Begin must leave everything as it was.
	device := b.graphicsResource.GraphicsDevice()
	for _, state := range []*stateBinding{
		{resource: blendState.resource, bound: &blendState.isBound},
		{resource: samplerState.resource, bound: &samplerState.isBound},
		{resource: depthStencilState.resource, bound: &depthStencilState.isBound},
		{resource: rasterizerState.resource, bound: &rasterizerState.isBound},
	} {
		state.resource.device = device
		*state.bound = true
	}
	if device != nil {
		device.blendState = blendState
		device.depthStencilState = depthStencilState
		device.rasterizerState = rasterizerState
	}
	b.inBeginEndPair = true
	return nil
}

// stateBinding is Apply's two stores, named so the loop above reads as the two
// things it does rather than as an index.
type stateBinding struct {
	resource *GraphicsResource
	bound    *bool
}
