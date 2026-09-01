package graphics

import (
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 73 — GraphicsDevice's remaining render-target surface.
// ---------------------------------------------------------------------------
//
// # Binding does NOT transfer ownership
//
// The device holds a REFERENCE to what is bound, never the native object. Every
// binding here leaves the target's own CNA handle owned by the target, and
// unbinding releases nothing: `SetRenderTarget(nil)` restores the back buffer
// and the target is still alive, still disposable, and still the same Go
// object. That is the reference's model and CNA's alike -- CNA refuses to
// destroy a render target while it is bound, with CNA_RESULT_INVALID_STATE,
// which is the ownership statement from the other side.

// SetRenderTargetByRenderTargetCubeAndCubeMapFace is
// GraphicsDevice::SetRenderTarget(RenderTargetCube, CubeMapFace):
//
//	if (renderTarget != null) {
//	    RenderTargetBinding binding = new RenderTargetBinding(renderTarget, cubeMapFace);
//	    SetRenderTargets(&binding, 1);
//	} else {
//	    SetRenderTargets(null, 0);
//	}
//
// A null target restores the back buffer, exactly as the 2D overload's does,
// and the face is then not read at all -- which is why this overload does NOT
// range-check the face when the target is null and the binding constructor
// does.
func (d *GraphicsDevice) SetRenderTargetByRenderTargetCubeAndCubeMapFace(
	renderTarget *RenderTargetCube, cubeMapFace CubeMapFace,
) error {
	if d == nil || d.device == nil {
		return errGraphicsDeviceNil
	}
	if renderTarget == nil {
		if err := d.device.SetRenderTargetCube(nil, uint32(cubeMapFace)); err != nil {
			return err
		}
		d.renderTargets = nil
		return nil
	}
	binding, err := NewRenderTargetBindingByRenderTargetCubeAndCubeMapFace(renderTarget, cubeMapFace)
	if err != nil {
		return err
	}
	resource := renderTarget.nativeResource()
	if resource == nil {
		return interop.ErrDisposed
	}
	if err := d.device.SetRenderTargetCube(resource, uint32(cubeMapFace)); err != nil {
		return err
	}
	d.renderTargets = []RenderTargetBinding{binding}
	return nil
}

// SetRenderTargets is
// GraphicsDevice::SetRenderTargets(params RenderTargetBinding[]):
//
//	if (renderTargets == null || renderTargets.Length == 0)
//	    SetRenderTargets(null, 0);
//	else
//	    SetRenderTargets(pinned, renderTargets.Length);
//
// A null or empty array is the back buffer, not a refusal -- which is the
// clearest statement in the type that "no render target" is a STATE rather than
// an error. Everything after it is validated by the device: the reference
// checks that every target belongs to this device, that their dimensions and
// sample counts agree, and that none is bound twice; CNA validates the same
// list, in its own words, before it binds anything.
func (d *GraphicsDevice) SetRenderTargets(renderTargets []RenderTargetBinding) error {
	if d == nil || d.device == nil {
		return errGraphicsDeviceNil
	}
	if len(renderTargets) == 0 {
		if err := d.device.SetRenderTargets(nil, nil); err != nil {
			return err
		}
		d.renderTargets = nil
		return nil
	}
	resources := make([]*interop.Resource, len(renderTargets))
	faces := make([]uint32, len(renderTargets))
	for index := range renderTargets {
		if renderTargets[index].native == nil {
			// A zero-valued binding carries no target. The reference cannot
			// have one -- its constructors refuse null -- so this is a Go-only
			// refusal for a Go-only value, named as such.
			return fmt.Errorf("%w: renderTargets[%d] is a zero RenderTargetBinding and carries no target",
				errGraphicsResourceArgument, index)
		}
		resources[index] = renderTargets[index].native
		faces[index] = uint32(renderTargets[index].cubeMapFace)
	}
	if err := d.device.SetRenderTargets(resources, faces); err != nil {
		return err
	}
	d.renderTargets = append([]RenderTargetBinding(nil), renderTargets...)
	return nil
}

// GetRenderTargets is GraphicsDevice::GetRenderTargets():
//
//	if (currentRenderTargetCount == 0) return emptyRenderTargetBindings;
//	RenderTargetBinding[] copy = new RenderTargetBinding[currentRenderTargetCount];
//	Array.Copy(currentRenderTargets, copy, currentRenderTargetCount);
//	return copy;
//
// A FRESH array every call, over the bindings the device holds -- so a caller
// who mutates the result changes nothing, and two calls answer two arrays whose
// elements are equal. The bindings themselves are values, and the target inside
// each is the same object the setter was given.
//
// It is infallible, because the reference's body reaches nothing: it reads a
// managed count and copies a managed array. CNA-Go's is the same read over the
// same managed field, which is why the device keeps one.
func (d *GraphicsDevice) GetRenderTargets() []RenderTargetBinding {
	if d == nil || len(d.renderTargets) == 0 {
		return nil
	}
	return append([]RenderTargetBinding(nil), d.renderTargets...)
}
