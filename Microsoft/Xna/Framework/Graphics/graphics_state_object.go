package graphics

import (
	"errors"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Foundation 59 — the graphics state objects, and the rule they share.
// ---------------------------------------------------------------------------

// The four state objects -- BlendState, DepthStencilState, RasterizerState and
// SamplerState -- are one family with one architecture, and the parts a reader
// should not have to reconstruct from four near-identical files are here.
//
// # They are GraphicsResources with NO native handle
//
// Every one of them is
//
//	.class public StateObject extends Microsoft.Xna.Framework.Graphics.GraphicsResource
//
// so each composes a GraphicsResource and inherits its nine public members. But
// none of them holds a native object: the reference keeps its values in
// `cached*` managed fields and pushes them to D3D only when the device applies
// the state, and CNA models them the same way -- CNA_BlendState and its three
// siblings are versioned C PODs passed BY VALUE, not handles.
//
//	ownership: MANAGED_VALUE
//
// The composed GraphicsResource therefore carries a nil resource, which is not
// a degenerate case: it is exactly the reference's own `_internalHandle == 0`
// branch, where Name and Tag answer from the local fields rather than from the
// device's cache. The same code path serves both, for the same reason.
//
// # They FREEZE, and the reference's word for it is `isBound`
//
//	set_ColorSourceBlend(Blend value)
//	  ThrowIfBound();
//	  cachedColorSourceBlend = value;
//
//	ThrowIfBound()
//	  if (isBound)
//	      throw new InvalidOperationException(String.Format(CurrentCulture,
//	          FrameworkResources.BoundStateObject, typeof(BlendState).Name));
//
// EVERY setter calls it, including the ones inside SetDefaults and inside the
// private constructors the static instances use. A state object becomes
// read-only the first time a GraphicsDevice binds it, and stays read-only.
//
// # The static instances are frozen FROM CONSTRUCTION
//
// This is the part the documentation sentence does not say. Each static field is
// built by a PRIVATE constructor whose last two statements are
//
//	set_Name(name);
//	isBound = true;
//
// so `BlendState.AlphaBlend.ColorSourceBlend = ...` throws immediately, on an
// object no device has ever seen. The public parameterless constructor ends with
// `isBound = false` instead, so a consumer's own instance is mutable until it is
// bound.
//
// # Fallibility
//
// Every SETTER is fallible, for a reason the reference states rather than a
// native one: the freeze is an InvalidOperationException the reference really
// throws, so a projection without an error channel could not report it. Every
// GETTER is one `ldfld` and carries none.

// errStateObjectInvalidOperation projects the InvalidOperationException
// ThrowIfBound raises.
var errStateObjectInvalidOperation = errors.New("graphics state object is read-only")

// boundStateObject is FrameworkResources.BoundStateObject, read by
// tools/resource_strings out of the retained Microsoft.Xna.Framework.dll under
// the key ThrowIfBound's IL names. It is a FORMAT string, and the reference
// substitutes the state object's own type name at BOTH placeholders.
const boundStateObject = "Cannot change read-only {0}. State objects become read-only the first time they are bound to a GraphicsDevice. To change property values, create a new {0} instance."

// throwIfBound is ThrowIfBound, carrying the reference's own formatted sentence.
// The substituted value is `typeof(T).Name`, which is the SHORT name.
//
// It takes the flag and the type identity as arguments rather than living on a
// shared struct, because the composition rule requires every state object to
// hold its `*GraphicsResource` as a field OF ITS OWN. A shared holder would put
// the base one level further down, where the verifier -- correctly -- cannot see
// it, and the rule that keeps the base private is worth more than the four
// duplicated fields it costs.
func throwIfBound(isBound bool, clrTypeName string) error {
	if !isBound {
		return nil
	}
	short := clrTypeName[strings.LastIndexByte(clrTypeName, '.')+1:]
	return fmt.Errorf("%w: %s", errStateObjectInvalidOperation,
		strings.ReplaceAll(boundStateObject, "{0}", short))
}

// newStateResource is the GraphicsResource half every state object composes: no
// device and NO NATIVE HANDLE, which is the reference's `_internalHandle == 0`
// branch and is why Name and Tag answer from managed storage here.
func newStateResource() *GraphicsResource { return newGraphicsResource(nil, nil) }

// disposeStateObject is every state object's Dispose(bool). All four are the
// same shape and none of them releases anything:
//
//	if (disposing) { try { ~StateObject(); } finally { base.Dispose(true); } }
//	else                                              base.Dispose(false);
//
// with `~StateObject()` a single `ret`. There is no native object to release --
// the values are managed and CNA takes them by value -- so the whole override is
// the base call, which sets the flag and raises Disposing once.
func disposeStateObject(resource *GraphicsResource, disposing bool) error {
	return resource.DisposeByBoolean(disposing)
}
