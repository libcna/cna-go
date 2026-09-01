package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// DepthStencilState is Microsoft.Xna.Framework.Graphics.DepthStencilState:
// sixteen depth and stencil values, three static presets, and the freeze rule
// the whole state family shares. See graphics_state_object.go.
//
// # Every default is read off SetDefaults
//
//	cachedDepthBufferEnable      = true            (ldc.i4.1)
//	cachedDepthBufferWriteEnable = true            (ldc.i4.1)
//	cachedDepthBufferFunction    = CompareFunction.LessEqual   (ldc.i4.3)
//	cachedStencilEnable          = false           (ldc.i4.0)
//	cachedStencilFunction        = CompareFunction.Always      (ldc.i4.0)
//	cachedStencilPass            = StencilOperation.Keep       (ldc.i4.0)
//	cachedStencilFail            = StencilOperation.Keep       (ldc.i4.0)
//	cachedStencilDepthBufferFail = StencilOperation.Keep       (ldc.i4.0)
//	cachedTwoSidedStencilMode    = false           (ldc.i4.0)
//	the four CounterClockwise* fields take the same Always/Keep values
//	cachedStencilMask            = -1              (ldc.i4.m1)
//	cachedStencilWriteMask       = -1              (ldc.i4.m1)
//	cachedReferenceStencil       = 0               (ldc.i4.0)
type DepthStencilState struct {
	// resource is the composed GraphicsResource, carrying no native handle.
	resource *GraphicsResource
	// isBound is the reference's own freeze flag.
	isBound bool

	depthBufferEnable                      bool
	depthBufferWriteEnable                 bool
	depthBufferFunction                    CompareFunction
	stencilEnable                          bool
	stencilFunction                        CompareFunction
	stencilPass                            StencilOperation
	stencilFail                            StencilOperation
	stencilDepthBufferFail                 StencilOperation
	twoSidedStencilMode                    bool
	counterClockwiseStencilFunction        CompareFunction
	counterClockwiseStencilPass            StencilOperation
	counterClockwiseStencilFail            StencilOperation
	counterClockwiseStencilDepthBufferFail StencilOperation
	stencilMask                            int32
	stencilWriteMask                       int32
	referenceStencil                       int32
}

// NewDepthStencilState is DepthStencilState::.ctor(): SetDefaults, then
// isBound = false.
func NewDepthStencilState() *DepthStencilState {
	state := &DepthStencilState{resource: newStateResource()}
	state.setDefaults()
	state.resource.bindDerived(state)
	return state
}

func (d *DepthStencilState) setDefaults() {
	d.depthBufferEnable = true
	d.depthBufferWriteEnable = true
	d.depthBufferFunction = CompareFunctionLessEqual
	d.stencilEnable = false
	d.stencilFunction = CompareFunctionAlways
	d.stencilPass = StencilOperationKeep
	d.stencilFail = StencilOperationKeep
	d.stencilDepthBufferFail = StencilOperationKeep
	d.twoSidedStencilMode = false
	d.counterClockwiseStencilFunction = CompareFunctionAlways
	d.counterClockwiseStencilPass = StencilOperationKeep
	d.counterClockwiseStencilFail = StencilOperationKeep
	d.counterClockwiseStencilDepthBufferFail = StencilOperationKeep
	d.stencilMask = -1
	d.stencilWriteMask = -1
	d.referenceStencil = 0
}

// newPresetDepthStencilState is DepthStencilState::.ctor(Boolean, Boolean,
// String), the PRIVATE constructor the three static fields use. Its last two
// statements are `Name = name; isBound = true`, which is why a preset is
// read-only before any device has seen it.
func newPresetDepthStencilState(depthBufferEnable, depthBufferWriteEnable bool, name string) *DepthStencilState {
	state := NewDepthStencilState()
	state.depthBufferEnable = depthBufferEnable
	state.depthBufferWriteEnable = depthBufferWriteEnable
	state.resource.SetName(name)
	state.isBound = true
	return state
}

// The three static instances, with the exact arguments the class initializer
// passes.
//
//	None      (false, false) "DepthStencilState.None"
//	Default   (true,  true)  "DepthStencilState.Default"
//	DepthRead (true,  false) "DepthStencilState.DepthRead"
var (
	depthStencilStateNone      = newPresetDepthStencilState(false, false, "DepthStencilState.None")
	depthStencilStateDefault   = newPresetDepthStencilState(true, true, "DepthStencilState.Default")
	depthStencilStateDepthRead = newPresetDepthStencilState(true, false, "DepthStencilState.DepthRead")
)

// DepthStencilStateNone is DepthStencilState::None.
func DepthStencilStateNone() *DepthStencilState { return depthStencilStateNone }

// DepthStencilStateDefault is DepthStencilState::Default.
func DepthStencilStateDefault() *DepthStencilState { return depthStencilStateDefault }

// DepthStencilStateDepthRead is DepthStencilState::DepthRead.
func DepthStencilStateDepthRead() *DepthStencilState { return depthStencilStateDepthRead }

// clrTypeName is System.Object::ToString's answer for a DepthStencilState.
func (d *DepthStencilState) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.DepthStencilState"
}

// The sixteen value properties. Every getter is one `ldfld`; every setter is
// `ThrowIfBound(); stfld`.

func (d *DepthStencilState) DepthBufferEnable() bool { return d.depthBufferEnable }

func (d *DepthStencilState) SetDepthBufferEnable(value bool) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.depthBufferEnable = value
	return nil
}

func (d *DepthStencilState) DepthBufferWriteEnable() bool { return d.depthBufferWriteEnable }

func (d *DepthStencilState) SetDepthBufferWriteEnable(value bool) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.depthBufferWriteEnable = value
	return nil
}

func (d *DepthStencilState) DepthBufferFunction() CompareFunction { return d.depthBufferFunction }

func (d *DepthStencilState) SetDepthBufferFunction(value CompareFunction) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.depthBufferFunction = value
	return nil
}

func (d *DepthStencilState) StencilEnable() bool { return d.stencilEnable }

func (d *DepthStencilState) SetStencilEnable(value bool) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilEnable = value
	return nil
}

func (d *DepthStencilState) StencilFunction() CompareFunction { return d.stencilFunction }

func (d *DepthStencilState) SetStencilFunction(value CompareFunction) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilFunction = value
	return nil
}

func (d *DepthStencilState) StencilPass() StencilOperation { return d.stencilPass }

func (d *DepthStencilState) SetStencilPass(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilPass = value
	return nil
}

func (d *DepthStencilState) StencilFail() StencilOperation { return d.stencilFail }

func (d *DepthStencilState) SetStencilFail(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilFail = value
	return nil
}

func (d *DepthStencilState) StencilDepthBufferFail() StencilOperation {
	return d.stencilDepthBufferFail
}

func (d *DepthStencilState) SetStencilDepthBufferFail(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilDepthBufferFail = value
	return nil
}

func (d *DepthStencilState) TwoSidedStencilMode() bool { return d.twoSidedStencilMode }

func (d *DepthStencilState) SetTwoSidedStencilMode(value bool) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.twoSidedStencilMode = value
	return nil
}

func (d *DepthStencilState) CounterClockwiseStencilFunction() CompareFunction {
	return d.counterClockwiseStencilFunction
}

func (d *DepthStencilState) SetCounterClockwiseStencilFunction(value CompareFunction) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.counterClockwiseStencilFunction = value
	return nil
}

func (d *DepthStencilState) CounterClockwiseStencilPass() StencilOperation {
	return d.counterClockwiseStencilPass
}

func (d *DepthStencilState) SetCounterClockwiseStencilPass(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.counterClockwiseStencilPass = value
	return nil
}

func (d *DepthStencilState) CounterClockwiseStencilFail() StencilOperation {
	return d.counterClockwiseStencilFail
}

func (d *DepthStencilState) SetCounterClockwiseStencilFail(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.counterClockwiseStencilFail = value
	return nil
}

func (d *DepthStencilState) CounterClockwiseStencilDepthBufferFail() StencilOperation {
	return d.counterClockwiseStencilDepthBufferFail
}

func (d *DepthStencilState) SetCounterClockwiseStencilDepthBufferFail(value StencilOperation) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.counterClockwiseStencilDepthBufferFail = value
	return nil
}

func (d *DepthStencilState) StencilMask() int32 { return d.stencilMask }

func (d *DepthStencilState) SetStencilMask(value int32) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilMask = value
	return nil
}

func (d *DepthStencilState) StencilWriteMask() int32 { return d.stencilWriteMask }

func (d *DepthStencilState) SetStencilWriteMask(value int32) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.stencilWriteMask = value
	return nil
}

func (d *DepthStencilState) ReferenceStencil() int32 { return d.referenceStencil }

func (d *DepthStencilState) SetReferenceStencil(value int32) error {
	if err := throwIfBound(d.isBound, d.clrTypeName()); err != nil {
		return err
	}
	d.referenceStencil = value
	return nil
}

// The nine members inherited from GraphicsResource, forwarded.

func (d *DepthStencilState) GraphicsDevice() *GraphicsDevice { return d.resource.GraphicsDevice() }
func (d *DepthStencilState) Name() string                    { return d.resource.Name() }
func (d *DepthStencilState) SetName(value string)            { d.resource.SetName(value) }
func (d *DepthStencilState) Tag() any                        { return d.resource.Tag() }
func (d *DepthStencilState) SetTag(value any)                { d.resource.SetTag(value) }
func (d *DepthStencilState) IsDisposed() bool                { return d.resource.IsDisposed() }
func (d *DepthStencilState) ToString() string                { return d.resource.ToString() }

func (d *DepthStencilState) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return d.resource.AddDisposingHandler(handler)
}

func (d *DepthStencilState) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	return d.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose(), inherited.
func (d *DepthStencilState) DisposeByNone() error { return d.DisposeByBoolean(true) }

// DisposeByBoolean is DepthStencilState::Dispose(bool), which releases nothing.
func (d *DepthStencilState) DisposeByBoolean(disposing bool) error {
	return disposeStateObject(d.resource, disposing)
}
