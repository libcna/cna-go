package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// RasterizerState is Microsoft.Xna.Framework.Graphics.RasterizerState: six
// rasterizer values, three static presets, and the freeze rule the whole state
// family shares. See graphics_state_object.go.
//
// # Every default is read off SetDefaults
//
//	cachedCullMode             = CullMode.CullCounterClockwiseFace (ldc.i4.2)
//	cachedFillMode             = FillMode.Solid                    (ldc.i4.0)
//	cachedScissorTestEnable    = false                             (ldc.i4.0)
//	cachedMultiSampleAntiAlias = TRUE                              (ldc.i4.1)
//	cachedDepthBias            = 0.0                               (ldc.r4)
//	cachedSlopeScaleDepthBias  = 0.0                               (ldc.r4)
//
// The two that surprise a reader are worth naming: culling defaults to
// counter-clockwise rather than none, and multisample antialiasing defaults to
// ON.
type RasterizerState struct {
	// resource is the composed GraphicsResource, carrying no native handle.
	resource *GraphicsResource
	// isBound is the reference's own freeze flag.
	isBound bool

	cullMode             CullMode
	fillMode             FillMode
	scissorTestEnable    bool
	multiSampleAntiAlias bool
	depthBias            float32
	slopeScaleDepthBias  float32
}

// NewRasterizerState is RasterizerState::.ctor(): SetDefaults, then
// isBound = false.
func NewRasterizerState() *RasterizerState {
	state := &RasterizerState{resource: newStateResource()}
	state.setDefaults()
	state.resource.bindDerived(state)
	return state
}

func (r *RasterizerState) setDefaults() {
	r.cullMode = CullModeCullCounterClockwiseFace
	r.fillMode = FillModeSolid
	r.scissorTestEnable = false
	r.multiSampleAntiAlias = true
	r.depthBias = 0
	r.slopeScaleDepthBias = 0
}

// newPresetRasterizerState is RasterizerState::.ctor(CullMode, String), the
// PRIVATE constructor the three static fields use.
func newPresetRasterizerState(cullMode CullMode, name string) *RasterizerState {
	state := NewRasterizerState()
	state.cullMode = cullMode
	state.resource.SetName(name)
	state.isBound = true
	return state
}

// The three static instances, with the exact arguments the class initializer
// passes.
//
//	CullNone             (CullMode.None)                     "RasterizerState.CullNone"
//	CullClockwise        (CullMode.CullClockwiseFace)        "RasterizerState.CullClockwise"
//	CullCounterClockwise (CullMode.CullCounterClockwiseFace) "RasterizerState.CullCounterClockwise"
var (
	rasterizerStateCullNone             = newPresetRasterizerState(CullModeNone, "RasterizerState.CullNone")
	rasterizerStateCullClockwise        = newPresetRasterizerState(CullModeCullClockwiseFace, "RasterizerState.CullClockwise")
	rasterizerStateCullCounterClockwise = newPresetRasterizerState(CullModeCullCounterClockwiseFace, "RasterizerState.CullCounterClockwise")
)

// RasterizerStateCullNone is RasterizerState::CullNone.
func RasterizerStateCullNone() *RasterizerState { return rasterizerStateCullNone }

// RasterizerStateCullClockwise is RasterizerState::CullClockwise.
func RasterizerStateCullClockwise() *RasterizerState { return rasterizerStateCullClockwise }

// RasterizerStateCullCounterClockwise is RasterizerState::CullCounterClockwise.
func RasterizerStateCullCounterClockwise() *RasterizerState {
	return rasterizerStateCullCounterClockwise
}

// clrTypeName is System.Object::ToString's answer for a RasterizerState.
func (r *RasterizerState) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.RasterizerState"
}

// The six value properties. Every getter is one `ldfld`; every setter is
// `ThrowIfBound(); stfld`.

func (r *RasterizerState) CullMode() CullMode { return r.cullMode }

func (r *RasterizerState) SetCullMode(value CullMode) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.cullMode = value
	return nil
}

func (r *RasterizerState) FillMode() FillMode { return r.fillMode }

func (r *RasterizerState) SetFillMode(value FillMode) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.fillMode = value
	return nil
}

func (r *RasterizerState) ScissorTestEnable() bool { return r.scissorTestEnable }

func (r *RasterizerState) SetScissorTestEnable(value bool) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.scissorTestEnable = value
	return nil
}

func (r *RasterizerState) MultiSampleAntiAlias() bool { return r.multiSampleAntiAlias }

func (r *RasterizerState) SetMultiSampleAntiAlias(value bool) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.multiSampleAntiAlias = value
	return nil
}

func (r *RasterizerState) DepthBias() float32 { return r.depthBias }

func (r *RasterizerState) SetDepthBias(value float32) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.depthBias = value
	return nil
}

func (r *RasterizerState) SlopeScaleDepthBias() float32 { return r.slopeScaleDepthBias }

func (r *RasterizerState) SetSlopeScaleDepthBias(value float32) error {
	if err := throwIfBound(r.isBound, r.clrTypeName()); err != nil {
		return err
	}
	r.slopeScaleDepthBias = value
	return nil
}

// The nine members inherited from GraphicsResource, forwarded.

func (r *RasterizerState) GraphicsDevice() *GraphicsDevice { return r.resource.GraphicsDevice() }
func (r *RasterizerState) Name() string                    { return r.resource.Name() }
func (r *RasterizerState) SetName(value string)            { r.resource.SetName(value) }
func (r *RasterizerState) Tag() any                        { return r.resource.Tag() }
func (r *RasterizerState) SetTag(value any)                { r.resource.SetTag(value) }
func (r *RasterizerState) IsDisposed() bool                { return r.resource.IsDisposed() }
func (r *RasterizerState) ToString() string                { return r.resource.ToString() }

func (r *RasterizerState) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return r.resource.AddDisposingHandler(handler)
}

func (r *RasterizerState) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	return r.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (r *RasterizerState) DisposeByNone() error { return r.DisposeByBoolean(true) }

// DisposeByBoolean is RasterizerState::Dispose(bool), which releases nothing.
func (r *RasterizerState) DisposeByBoolean(disposing bool) error {
	return disposeStateObject(r.resource, disposing)
}
