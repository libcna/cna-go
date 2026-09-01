package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// BlendState is Microsoft.Xna.Framework.Graphics.BlendState: twelve blend
// values, four static presets, and the freeze rule the whole state family
// shares. See graphics_state_object.go for the rule and the ownership.
//
// # Every default is read off SetDefaults
//
//	cachedColorSourceBlend       = Blend.One                 (ldc.i4.0)
//	cachedColorDestinationBlend  = Blend.Zero                (ldc.i4.1)
//	cachedColorBlendFunction     = BlendFunction.Add         (ldc.i4.0)
//	cachedAlphaSourceBlend       = Blend.One                 (ldc.i4.0)
//	cachedAlphaDestinationBlend  = Blend.Zero                (ldc.i4.1)
//	cachedAlphaBlendFunction     = BlendFunction.Add         (ldc.i4.0)
//	cachedColorWriteChannels{,1,2,3} = ColorWriteChannels.All (ldc.i4.s 15)
//	cachedBlendFactor            = Color.White
//	cachedMultiSampleMask        = -1                        (ldc.i4.m1)
type BlendState struct {
	// resource is the composed GraphicsResource, held as this type's OWN field
	// because that is what the composition rule requires. It carries no native
	// handle: a state object is MANAGED_VALUE.
	resource *GraphicsResource
	// isBound is the reference's own `assembly bool isBound`, the freeze flag
	// every setter consults.
	isBound bool

	colorSourceBlend      Blend
	colorDestinationBlend Blend
	colorBlendFunction    BlendFunction
	alphaSourceBlend      Blend
	alphaDestinationBlend Blend
	alphaBlendFunction    BlendFunction
	colorWriteChannels    ColorWriteChannels
	colorWriteChannels1   ColorWriteChannels
	colorWriteChannels2   ColorWriteChannels
	colorWriteChannels3   ColorWriteChannels
	blendFactor           framework.Color
	multiSampleMask       int32
}

// NewBlendState is BlendState::.ctor(), whose whole body is
//
//	SetDefaults();
//	isBound = false;
//
// under a `try/fault` that disposes a half-built object. Go has no half-built
// object to dispose: nothing here can fail.
func NewBlendState() *BlendState {
	state := &BlendState{resource: newStateResource()}
	state.setDefaults()
	state.resource.bindDerived(state)
	return state
}

func (b *BlendState) setDefaults() {
	b.colorSourceBlend = BlendOne
	b.colorDestinationBlend = BlendZero
	b.colorBlendFunction = BlendFunctionAdd
	b.alphaSourceBlend = BlendOne
	b.alphaDestinationBlend = BlendZero
	b.alphaBlendFunction = BlendFunctionAdd
	b.colorWriteChannels = ColorWriteChannelsAll
	b.colorWriteChannels1 = ColorWriteChannelsAll
	b.colorWriteChannels2 = ColorWriteChannelsAll
	b.colorWriteChannels3 = ColorWriteChannelsAll
	b.blendFactor = framework.ColorWhite()
	b.multiSampleMask = -1
}

// newPresetBlendState is BlendState::.ctor(Blend, Blend, String), the PRIVATE
// constructor the four static fields use:
//
//	SetDefaults();
//	ColorSourceBlend = sourceBlend;   AlphaSourceBlend = sourceBlend;
//	ColorDestinationBlend = destinationBlend;
//	AlphaDestinationBlend = destinationBlend;
//	Name = name;
//	isBound = TRUE;
//
// The last line is why a static preset is read-only before any device has seen
// it, and it is the reason this constructor is not projected: the reference
// declares it `private`.
func newPresetBlendState(sourceBlend, destinationBlend Blend, name string) *BlendState {
	state := NewBlendState()
	state.colorSourceBlend = sourceBlend
	state.alphaSourceBlend = sourceBlend
	state.colorDestinationBlend = destinationBlend
	state.alphaDestinationBlend = destinationBlend
	state.resource.SetName(name)
	state.isBound = true
	return state
}

// The four static instances, with the exact arguments BlendState::.cctor
// passes. They are package-level variables rather than functions because the
// reference declares them `public static initonly` FIELDS: one object each, for
// the life of the process, whose identity a consumer may compare.
//
//	Opaque           (One,        Zero)                 "BlendState.Opaque"
//	AlphaBlend       (One,        InverseSourceAlpha)   "BlendState.AlphaBlend"
//	Additive         (SourceAlpha, One)                 "BlendState.Additive"
//	NonPremultiplied (SourceAlpha, InverseSourceAlpha)  "BlendState.NonPremultiplied"
//
// The settled static-member rule spells a static as a package FUNCTION prefixed
// by its declaring type, so each is `BlendStateOpaque()` rather than a variable.
// The identity the reference's `initonly` field promises is kept by answering
// with the same object every time: a consumer comparing two reads gets equal
// pointers, as two reads of a static field do.
var (
	blendStateOpaque           = newPresetBlendState(BlendOne, BlendZero, "BlendState.Opaque")
	blendStateAlphaBlend       = newPresetBlendState(BlendOne, BlendInverseSourceAlpha, "BlendState.AlphaBlend")
	blendStateAdditive         = newPresetBlendState(BlendSourceAlpha, BlendOne, "BlendState.Additive")
	blendStateNonPremultiplied = newPresetBlendState(BlendSourceAlpha, BlendInverseSourceAlpha, "BlendState.NonPremultiplied")
)

// BlendStateOpaque is BlendState::Opaque.
func BlendStateOpaque() *BlendState { return blendStateOpaque }

// BlendStateAlphaBlend is BlendState::AlphaBlend.
func BlendStateAlphaBlend() *BlendState { return blendStateAlphaBlend }

// BlendStateAdditive is BlendState::Additive.
func BlendStateAdditive() *BlendState { return blendStateAdditive }

// BlendStateNonPremultiplied is BlendState::NonPremultiplied.
func BlendStateNonPremultiplied() *BlendState { return blendStateNonPremultiplied }

// clrTypeName is System.Object::ToString's answer for a BlendState. A preset
// answers with its Name instead, because the private constructor set one.
func (b *BlendState) clrTypeName() string { return "Microsoft.Xna.Framework.Graphics.BlendState" }

// The twelve value properties. Every getter is one `ldfld`; every setter is
// `ThrowIfBound(); stfld`.

func (b *BlendState) ColorSourceBlend() Blend { return b.colorSourceBlend }

func (b *BlendState) SetColorSourceBlend(value Blend) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorSourceBlend = value
	return nil
}

func (b *BlendState) ColorDestinationBlend() Blend { return b.colorDestinationBlend }

func (b *BlendState) SetColorDestinationBlend(value Blend) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorDestinationBlend = value
	return nil
}

func (b *BlendState) ColorBlendFunction() BlendFunction { return b.colorBlendFunction }

func (b *BlendState) SetColorBlendFunction(value BlendFunction) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorBlendFunction = value
	return nil
}

func (b *BlendState) AlphaSourceBlend() Blend { return b.alphaSourceBlend }

func (b *BlendState) SetAlphaSourceBlend(value Blend) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.alphaSourceBlend = value
	return nil
}

func (b *BlendState) AlphaDestinationBlend() Blend { return b.alphaDestinationBlend }

func (b *BlendState) SetAlphaDestinationBlend(value Blend) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.alphaDestinationBlend = value
	return nil
}

func (b *BlendState) AlphaBlendFunction() BlendFunction { return b.alphaBlendFunction }

func (b *BlendState) SetAlphaBlendFunction(value BlendFunction) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.alphaBlendFunction = value
	return nil
}

func (b *BlendState) ColorWriteChannels() ColorWriteChannels { return b.colorWriteChannels }

func (b *BlendState) SetColorWriteChannels(value ColorWriteChannels) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorWriteChannels = value
	return nil
}

func (b *BlendState) ColorWriteChannels1() ColorWriteChannels { return b.colorWriteChannels1 }

func (b *BlendState) SetColorWriteChannels1(value ColorWriteChannels) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorWriteChannels1 = value
	return nil
}

func (b *BlendState) ColorWriteChannels2() ColorWriteChannels { return b.colorWriteChannels2 }

func (b *BlendState) SetColorWriteChannels2(value ColorWriteChannels) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorWriteChannels2 = value
	return nil
}

func (b *BlendState) ColorWriteChannels3() ColorWriteChannels { return b.colorWriteChannels3 }

func (b *BlendState) SetColorWriteChannels3(value ColorWriteChannels) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.colorWriteChannels3 = value
	return nil
}

func (b *BlendState) BlendFactor() framework.Color { return b.blendFactor }

func (b *BlendState) SetBlendFactor(value framework.Color) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.blendFactor = value
	return nil
}

func (b *BlendState) MultiSampleMask() int32 { return b.multiSampleMask }

func (b *BlendState) SetMultiSampleMask(value int32) error {
	if err := throwIfBound(b.isBound, b.clrTypeName()); err != nil {
		return err
	}
	b.multiSampleMask = value
	return nil
}

// The nine members inherited from GraphicsResource, forwarded.

func (b *BlendState) GraphicsDevice() *GraphicsDevice { return b.resource.GraphicsDevice() }
func (b *BlendState) Name() string                    { return b.resource.Name() }
func (b *BlendState) SetName(value string)            { b.resource.SetName(value) }
func (b *BlendState) Tag() any                        { return b.resource.Tag() }
func (b *BlendState) SetTag(value any)                { b.resource.SetTag(value) }
func (b *BlendState) IsDisposed() bool                { return b.resource.IsDisposed() }
func (b *BlendState) ToString() string                { return b.resource.ToString() }

func (b *BlendState) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return b.resource.AddDisposingHandler(handler)
}

func (b *BlendState) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	return b.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (b *BlendState) DisposeByNone() error { return b.DisposeByBoolean(true) }

// DisposeByBoolean is BlendState::Dispose(bool), which releases nothing.
func (b *BlendState) DisposeByBoolean(disposing bool) error {
	return disposeStateObject(b.resource, disposing)
}
