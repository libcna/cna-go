package graphics

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// OcclusionQuery counts the pixels a draw actually wrote, between a Begin and
// an End.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
//
//	.class public auto ansi beforefieldinit OcclusionQuery
//	       extends GraphicsResource
//	       implements IGraphicsResource
//
// # The type is a FOUR-FLAG state machine, and the flags are the whole contract
//
//	.field private int32 _pixelCount
//	.field private bool  _isAvailable
//	.field private bool  _isInBeginEndPair
//	.field private bool  _hasCalledBegin
//	.field private bool  _hasIsCompleteBeenQueried
//
// Every guard the type has is a test of one of those, and none of them is
// CNA's: the projection keeps them managed-side and reaches CNA only for the
// four operations that touch the GPU.
//
//	Begin        if (_isInBeginEndPair)         throw EndMustBeCalledBeforeBegin
//	             if (!_hasIsCompleteBeenQueried) throw IsCompleteMustBeCalled
//	             ... issue ...
//	             _isAvailable = false; _isInBeginEndPair = true;
//	             _hasCalledBegin = true; _hasIsCompleteBeenQueried = false
//	End          if (!_isInBeginEndPair)        throw BeginMustBeCalledBeforeEnd
//	             ... issue ...   _isInBeginEndPair = false
//	IsComplete   _hasIsCompleteBeenQueried = true
//	             if (no native query) return false
//	             if (!_hasCalledBegin)  return false
//	             ... read ...  _isAvailable = ...; if available _pixelCount = ...
//	             return _isAvailable
//	PixelCount   if (!IsComplete) throw DataNotAvailable
//	             return _pixelCount
//
// Three details a reader would guess wrong, and all three are the reason the
// flags exist rather than a single "in a pair" bool:
//
//  1. **The constructor sets `_hasIsCompleteBeenQueried` to TRUE.** A freshly
//     built query may Begin immediately; the guard exists to stop a SECOND
//     Begin before the first result was looked at.
//  2. **`IsComplete` is not a pure read.** It sets `_hasIsCompleteBeenQueried`
//     as its first statement -- before any early return -- so merely asking
//     re-arms Begin even when the answer is false.
//  3. **`PixelCount` calls `IsComplete`**, so reading the count also re-arms
//     Begin, and its refusal is the one the property carries rather than a
//     stale-value read.
//
// # What CNA supplies, and the capability that gates it
//
// `cna_occlusion_query_create` answers `CNA_RESULT_NOT_SUPPORTED` where the
// backend has no occlusion query. The reference's constructor has the same
// shape one level up -- it throws NotSupportedException from
// ProfileCapabilities when the profile lacks the feature -- so a refusal here
// is the same kind of answer, reported through the error channel rather than
// through a profile table CNA-Go does not project.
type OcclusionQuery struct {
	resource *GraphicsResource

	pixelCount               int32
	isAvailable              bool
	isInBeginEndPair         bool
	hasCalledBegin           bool
	hasIsCompleteBeenQueried bool
}

// errOcclusionQueryNil is the Go-only guard for a zero value.
var errOcclusionQueryNil = errors.New("occlusion query is nil or uninitialized")

// The three Microsoft messages this type's guards carry, read from
// Microsoft.Xna.Framework.dll.
//
// endMustBeCalledBeforeBegin is the SAME key SpriteBatch's pair guard uses and
// a different sentence from the one a reader expects: it reads "Begin cannot be
// called again until End has been successfully called", not "End must be
// called". The key names the situation, not the text.
const (
	occlusionEndMustBeCalledBeforeBegin = "Begin cannot be called again until End has been successfully called."
	occlusionBeginMustBeCalledBeforeEnd = "Begin must be called successfully before End can be called."
	occlusionIsCompleteMustBeCalled     = "Begin may not be called on this query object again before IsComplete has been checked."
	occlusionDataNotAvailable           = "The query data is not yet available. Use the IsComplete property to determine if the data is available before attempting to retrieve it."
)

// NewOcclusionQuery is OcclusionQuery::.ctor(GraphicsDevice), 174 bytes:
//
//	if (graphicsDevice == null)
//	    throw new ArgumentNullException("graphicsDevice",
//	        FrameworkResources.DeviceCannotBeNullOnResourceCreate);
//	if (!graphicsDevice.profileCapabilities.OcclusionQuery)
//	    ThrowNotSupportedException(ProfileFeatureNotSupported, typeof(OcclusionQuery));
//	_hasIsCompleteBeenQueried = true;
//	... create the D3D query ...
//
// The profile check is NOT reproduced as a managed test, for the reason
// DrawPrimitives' profile cap is not: ProfileCapabilities is not a public XNA
// type, CNA-Go projects no part of it, and there is no measured table to test
// against. CNA answers the same question at the same moment --
// CNA_RESULT_NOT_SUPPORTED from the creation route -- so the refusal a consumer
// meets is in the same place, with the renderer's reason rather than an
// invented profile one.
func NewOcclusionQuery(graphicsDevice *GraphicsDevice) (*OcclusionQuery, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	resource, err := graphicsDevice.device.CreateOcclusionQuery()
	if err != nil {
		return nil, err
	}
	query := &OcclusionQuery{
		resource: newGraphicsResource(graphicsDevice, resource),
		// The constructor's one store, and the reason a fresh query may Begin.
		hasIsCompleteBeenQueried: true,
	}
	query.resource.bindDerived(query)
	return query, nil
}

// clrTypeName is System.Object::ToString's answer for an OcclusionQuery.
func (q *OcclusionQuery) clrTypeName() string {
	return "Microsoft.Xna.Framework.Graphics.OcclusionQuery"
}

func (q *OcclusionQuery) nativeResource() *interop.Resource {
	if q == nil || q.resource == nil {
		return nil
	}
	return q.resource.nativeResource()
}

// Begin is OcclusionQuery::Begin(). Its two guards run in the reference's
// order, and the four stores after the issue are all of them -- including the
// one that DISARMS itself, `_hasIsCompleteBeenQueried = false`.
func (q *OcclusionQuery) Begin() error {
	if q == nil {
		return errOcclusionQueryNil
	}
	if q.isInBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, occlusionEndMustBeCalledBeforeBegin)
	}
	if !q.hasIsCompleteBeenQueried {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, occlusionIsCompleteMustBeCalled)
	}
	resource := q.nativeResource()
	if resource == nil {
		return errOcclusionQueryNil
	}
	if err := resource.OcclusionQueryBegin(); err != nil {
		return err
	}
	q.isAvailable = false
	q.isInBeginEndPair = true
	q.hasCalledBegin = true
	q.hasIsCompleteBeenQueried = false
	return nil
}

// End is OcclusionQuery::End(), one guard and one store.
func (q *OcclusionQuery) End() error {
	if q == nil {
		return errOcclusionQueryNil
	}
	if !q.isInBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, occlusionBeginMustBeCalledBeforeEnd)
	}
	resource := q.nativeResource()
	if resource == nil {
		return errOcclusionQueryNil
	}
	if err := resource.OcclusionQueryEnd(); err != nil {
		return err
	}
	q.isInBeginEndPair = false
	return nil
}

// IsComplete is OcclusionQuery::get_IsComplete(), and it MUTATES.
//
// Its first statement is `_hasIsCompleteBeenQueried = true`, before either
// early return, so asking re-arms Begin whatever the answer is. The two early
// returns are a missing native query and a query that has never begun, and both
// answer false without reading anything.
func (q *OcclusionQuery) IsComplete() (bool, error) {
	if q == nil {
		return false, errOcclusionQueryNil
	}
	// First, and before every branch below.
	q.hasIsCompleteBeenQueried = true
	resource := q.nativeResource()
	if resource == nil {
		return false, nil
	}
	if !q.hasCalledBegin {
		return false, nil
	}
	available, err := resource.OcclusionQueryIsComplete()
	if err != nil {
		return false, err
	}
	q.isAvailable = available
	if !available {
		return false, nil
	}
	count, err := resource.OcclusionQueryPixelCount()
	if err != nil {
		return false, err
	}
	q.pixelCount = count
	return true, nil
}

// PixelCount is OcclusionQuery::get_PixelCount(), 26 bytes:
//
//	if (!IsComplete) throw new InvalidOperationException(DataNotAvailable);
//	return _pixelCount;
//
// It calls IsComplete rather than reading `_isAvailable`, so reading the count
// re-arms Begin as a side effect -- which is the reference's behaviour and is
// why the two members cannot be reordered.
func (q *OcclusionQuery) PixelCount() (int32, error) {
	if q == nil {
		return 0, errOcclusionQueryNil
	}
	complete, err := q.IsComplete()
	if err != nil {
		return 0, err
	}
	if !complete {
		return 0, fmt.Errorf("%w: %s", errSpriteInvalidOperation, occlusionDataNotAvailable)
	}
	return q.pixelCount, nil
}

// ---------------------------------------------------------------------------
// The inherited public surface of GraphicsResource, forwarded.
// ---------------------------------------------------------------------------

// GraphicsDevice is GraphicsResource::get_GraphicsDevice.
func (q *OcclusionQuery) GraphicsDevice() *GraphicsDevice {
	if q == nil {
		return nil
	}
	return q.resource.GraphicsDevice()
}

// Name is GraphicsResource::get_Name.
func (q *OcclusionQuery) Name() string {
	if q == nil {
		return ""
	}
	return q.resource.Name()
}

// SetName is GraphicsResource::set_Name.
func (q *OcclusionQuery) SetName(value string) {
	if q == nil {
		return
	}
	q.resource.SetName(value)
}

// Tag is GraphicsResource::get_Tag.
func (q *OcclusionQuery) Tag() any {
	if q == nil {
		return nil
	}
	return q.resource.Tag()
}

// SetTag is GraphicsResource::set_Tag.
func (q *OcclusionQuery) SetTag(value any) {
	if q == nil {
		return
	}
	q.resource.SetTag(value)
}

// IsDisposed is GraphicsResource::get_IsDisposed.
func (q *OcclusionQuery) IsDisposed() bool {
	if q == nil {
		return true
	}
	return q.resource.IsDisposed()
}

// ToString is GraphicsResource::ToString.
func (q *OcclusionQuery) ToString() string {
	if q == nil {
		return ""
	}
	return q.resource.ToString()
}

// AddDisposingHandler is add_Disposing.
func (q *OcclusionQuery) AddDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if q == nil {
		return framework.EventSubscription{}, errOcclusionQueryNil
	}
	return q.resource.AddDisposingHandler(handler)
}

// RemoveDisposingHandler is remove_Disposing.
func (q *OcclusionQuery) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if q == nil {
		return errOcclusionQueryNil
	}
	return q.resource.RemoveDisposingHandler(subscription)
}

// DisposeByNone is GraphicsResource::Dispose().
func (q *OcclusionQuery) DisposeByNone() error {
	return q.DisposeByBoolean(true)
}

// DisposeByBoolean is OcclusionQuery::Dispose(bool), which the type DECLARES --
// so unlike the stock effects this one projects both dispose members.
//
// The reference's body clears `_isAvailable` and releases the query; the
// projection releases the one CNA handle and lets the composed base do the
// rest, which is the shape every other graphics resource has.
func (q *OcclusionQuery) DisposeByBoolean(disposing bool) error {
	if q == nil {
		return errOcclusionQueryNil
	}
	var released error
	if !q.resource.IsDisposed() {
		q.isAvailable = false
		released = q.resource.releaseNativeObject()
	}
	baseErr := q.resource.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}
