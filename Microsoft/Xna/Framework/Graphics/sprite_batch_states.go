package graphics

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
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

// BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerStateAndEffect
// is SpriteBatch::Begin(SpriteSortMode, BlendState, SamplerState,
// DepthStencilState, RasterizerState, Effect) -- Foundation 72, and the last
// Begin overload the type was missing.
//
// It forwards to the seven-argument Begin with Matrix.Identity, which is what
// CNA's own route expresses as a NULL transform.
//
// A null Effect is not a refusal: SpriteBatch::SetRenderState resolves it to
// the stock sprite effect, and CNA documents CNA_INVALID_HANDLE as selecting
// exactly that. So the two runtimes agree, and this overload is the difference
// between "the batch draws with the default shader" and "the batch draws with
// mine" rather than between drawing and refusing.
func (b *SpriteBatch) BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerStateAndEffect(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState, effect *Effect,
) error {
	return b.beginWithEffect(sortMode, blendState, samplerState, depthStencilState, rasterizerState, effect, nil)
}

// BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerStateAndEffectAndMatrix
// is the seven-argument Begin every other overload funnels into.
//
// The transform reaches CNA as sixteen floats in row-major order, and a caller
// that passes Matrix.Identity gets the same result as the six-argument
// overload -- which is the identity CNA substitutes for a null transform, so
// the two paths agree by construction rather than by coincidence.
func (b *SpriteBatch) BeginBySpriteSortModeAndBlendStateAndSamplerStateAndDepthStencilStateAndRasterizerStateAndEffectAndMatrix(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState, effect *Effect,
	transformMatrix framework.Matrix,
) error {
	transform := matrixToRowMajor(transformMatrix)
	return b.beginWithEffect(sortMode, blendState, samplerState, depthStencilState, rasterizerState, effect, &transform)
}

// beginWithStates is the two effect-free overloads' forwarder. It passes a nil
// effect and a nil transform, which is EXACTLY what those two overloads pass in
// the reference: every Begin funnels into the seven-argument one, and the two
// short ones supply `null` and `Matrix.Identity`.
func (b *SpriteBatch) beginWithStates(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState,
) error {
	return b.beginWithEffect(sortMode, blendState, samplerState, depthStencilState, rasterizerState, nil, nil)
}

// beginWithEffect is the whole shared body: the pair guard, the null
// substitutions, the native call, and only then the managed effects.
//
// # All FOUR overloads reach ONE native route
//
// Foundation 60 sent the two effect-free overloads to
// `cna_sprite_batch_begin_with_states`, which cannot express an effect. Now that
// all four exist, they all reach `cna_sprite_batch_begin_with_effect`, because
// the REFERENCE has one Begin body and this is the route whose prototype
// matches it: CNA documents CNA_INVALID_HANDLE as selecting the stock sprite
// effect and a null transform as the identity, which is precisely what the two
// short overloads supply.
//
// Keeping the narrower route bound as well would give one reference path two
// native paths that could drift, which is the shape Foundation 50 accepted for
// Draw only because CNA declares those two commands for a measured reason. Here
// there is none: the wider route's contract contains the narrower one's.
func (b *SpriteBatch) beginWithEffect(
	sortMode SpriteSortMode, blendState *BlendState, samplerState *SamplerState,
	depthStencilState *DepthStencilState, rasterizerState *RasterizerState,
	effect *Effect, transform *[16]float32,
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
	// A disposed EFFECT is refused for the reason a disposed state object is:
	// the reference's SetRenderState dereferences it, and CNA-Go's own object
	// still holds a handle CNA has released.
	if effect != nil && effect.IsDisposed() {
		return objectDisposedState(effect.clrTypeName())
	}
	if err := b.resource().BeginSpriteBatchWithEffect(
		uint32(sortMode), blendState.interopValue(), samplerState.interopValue(),
		depthStencilState.interopValue(), rasterizerState.interopValue(),
		effect.nativeResource(), transform); err != nil {
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
