package graphics

import (
	"errors"
	"testing"

	"github.com/openeggbert/cna-go/internal/interop"
)

// OcclusionQuery's four flags decide every guard the type has, and none of the
// guards is CNA's -- so the whole state machine is measured here without a
// device. What needs one is the four operations that touch the GPU, which are
// in the native-stress scenario.

// newManagedQuery is the object the constructor builds, minus the native half:
// the one store the constructor makes and nothing else.
func newManagedQuery() *OcclusionQuery {
	return &OcclusionQuery{hasIsCompleteBeenQueried: true}
}

// TestOcclusionQueryConstructorArmsBegin pins the constructor's ONE store, and
// pins that it is what lets a fresh query begin.
func TestOcclusionQueryConstructorArmsBegin(t *testing.T) {
	query := newManagedQuery()
	if !query.hasIsCompleteBeenQueried {
		t.Fatal("a fresh query is not armed; the constructor's only store is _hasIsCompleteBeenQueried = true")
	}
	if query.isInBeginEndPair || query.hasCalledBegin || query.isAvailable || query.pixelCount != 0 {
		t.Fatal("a fresh query has state the constructor does not set")
	}
	// A query with a nil device is refused before anything native.
	if _, err := NewOcclusionQuery(nil); !errors.Is(err, errGraphicsResourceArgumentNull) {
		t.Fatalf("NewOcclusionQuery(nil) = %v", err)
	}
}

// TestOcclusionQueryBeginGuardsRunInTheReferenceOrder pins both guards and the
// order they are tested in, which matters: a query that is BOTH in a pair and
// unarmed reports the pair.
func TestOcclusionQueryBeginGuardsRunInTheReferenceOrder(t *testing.T) {
	query := newManagedQuery()
	query.isInBeginEndPair = true
	query.hasIsCompleteBeenQueried = false
	err := query.Begin()
	if err == nil || !errors.Is(err, errSpriteInvalidOperation) {
		t.Fatalf("Begin inside a pair = %v", err)
	}
	if !containsSubstring(err.Error(), occlusionEndMustBeCalledBeforeBegin) {
		t.Fatalf("Begin inside a pair reported %q; the pair guard is tested FIRST", err)
	}
	// Out of a pair but unarmed, the second guard fires with its own message.
	query.isInBeginEndPair = false
	err = query.Begin()
	if err == nil || !containsSubstring(err.Error(), occlusionIsCompleteMustBeCalled) {
		t.Fatalf("Begin on an unarmed query = %v", err)
	}
	// Neither guard touched the state.
	if query.hasCalledBegin || query.isInBeginEndPair {
		t.Fatal("a refused Begin changed the query's state")
	}
}

// TestOcclusionQueryBeginStoresNothingWhenTheNativeCallFails pins the ORDER of
// Begin's body: the issue comes first and the four stores after it, so a failed
// issue leaves the query exactly as it was.
//
// A query with a dead-but-present native half is what reaches the call at all:
// one with no resource refuses before it, and the reordering would then be
// invisible.
func TestOcclusionQueryBeginStoresNothingWhenTheNativeCallFails(t *testing.T) {
	query := newManagedQuery()
	query.resource = newGraphicsResource(nil, &interop.Resource{})
	if err := query.Begin(); err == nil {
		t.Fatal("Begin succeeded over a dead native query")
	}
	if query.isInBeginEndPair || query.hasCalledBegin {
		t.Fatal("a failed Begin entered the pair")
	}
	if !query.hasIsCompleteBeenQueried {
		t.Fatal("a failed Begin disarmed the query; the stores come AFTER the issue")
	}
	// isAvailable is the first of the four stores and the easiest to move
	// above the call by accident.
	query.isAvailable = true
	if err := query.Begin(); err == nil {
		t.Fatal("Begin succeeded over a dead native query")
	}
	if !query.isAvailable {
		t.Fatal("a failed Begin cleared _isAvailable; every store follows the issue")
	}
}

// TestOcclusionQueryEndRefusesOutsideAPair pins the one guard End has, and its
// message -- which is a DIFFERENT sentence from Begin's.
func TestOcclusionQueryEndRefusesOutsideAPair(t *testing.T) {
	query := newManagedQuery()
	err := query.End()
	if err == nil || !containsSubstring(err.Error(), occlusionBeginMustBeCalledBeforeEnd) {
		t.Fatalf("End outside a pair = %v", err)
	}
	if containsSubstring(err.Error(), occlusionEndMustBeCalledBeforeBegin) {
		t.Fatal("End's refusal carried Begin's message; the two keys read differently")
	}
}

// TestOcclusionQueryIsCompleteArmsBeginEvenWhenItAnswersFalse is the detail the
// whole type turns on: `_hasIsCompleteBeenQueried = true` is IsComplete's first
// statement, before either early return.
func TestOcclusionQueryIsCompleteArmsBeginEvenWhenItAnswersFalse(t *testing.T) {
	// A query with no native half takes the FIRST early return.
	query := newManagedQuery()
	query.hasIsCompleteBeenQueried = false
	complete, err := query.IsComplete()
	if err != nil || complete {
		t.Fatalf("IsComplete on a query with no native half = %v, %v", complete, err)
	}
	if !query.hasIsCompleteBeenQueried {
		t.Fatal("IsComplete answered false without arming Begin; the store is its FIRST statement")
	}

	// A query that has a native half but has never begun takes the SECOND
	// early return, and arms just the same.
	begun := newManagedQuery()
	begun.resource = newGraphicsResource(nil, &interop.Resource{})
	begun.hasIsCompleteBeenQueried = false
	begun.hasCalledBegin = false
	complete, err = begun.IsComplete()
	if err != nil || complete {
		t.Fatalf("IsComplete before any Begin = %v, %v", complete, err)
	}
	if !begun.hasIsCompleteBeenQueried {
		t.Fatal("the never-begun early return did not arm Begin")
	}
}

// TestOcclusionQueryPixelCountRefusesUntilComplete pins the refusal and the
// SIDE EFFECT: PixelCount calls IsComplete, so reading the count arms Begin too.
func TestOcclusionQueryPixelCountRefusesUntilComplete(t *testing.T) {
	query := newManagedQuery()
	query.hasIsCompleteBeenQueried = false
	count, err := query.PixelCount()
	if err == nil || !containsSubstring(err.Error(), occlusionDataNotAvailable) {
		t.Fatalf("PixelCount before completion = %v, %v", count, err)
	}
	if count != 0 {
		t.Fatalf("a refused PixelCount answered %d", count)
	}
	if !query.hasIsCompleteBeenQueried {
		t.Fatal("PixelCount did not arm Begin; it calls IsComplete rather than reading the flag")
	}
	// A query the read has already completed answers the stored count without
	// reaching a device, because IsComplete's first early return is taken.
	//
	// This is the one shape the managed half can produce: no native resource,
	// so IsComplete answers false. The complete path needs a device and is in
	// the stress scenario.
}

// TestOcclusionQueryRefusesANilReceiver covers the Go-only guard on all five
// members.
func TestOcclusionQueryRefusesANilReceiver(t *testing.T) {
	var query *OcclusionQuery
	if err := query.Begin(); !errors.Is(err, errOcclusionQueryNil) {
		t.Fatalf("Begin on nil = %v", err)
	}
	if err := query.End(); !errors.Is(err, errOcclusionQueryNil) {
		t.Fatalf("End on nil = %v", err)
	}
	if _, err := query.IsComplete(); !errors.Is(err, errOcclusionQueryNil) {
		t.Fatalf("IsComplete on nil = %v", err)
	}
	if _, err := query.PixelCount(); !errors.Is(err, errOcclusionQueryNil) {
		t.Fatalf("PixelCount on nil = %v", err)
	}
	if !query.IsDisposed() {
		t.Fatal("a nil query does not report itself disposed")
	}
}
