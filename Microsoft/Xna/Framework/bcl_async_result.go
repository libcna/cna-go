package framework

// This file is CNA-Go language support, not XNA surface.
//
// System.IAsyncResult, which StorageDevice's four BeginShowSelector overloads
// and BeginOpenContainer return and which their End counterparts consume. It
// needs a public Go spelling for the reason System.TimeSpan does: the pinned
// contract names it at a public signature position.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # XNA's storage APM is FAKE async, and that is measured from XNA
//
// StorageDevice::BeginShowSelector's whole body, after its two guards, is
//
//	var result = new StorageDeviceAsyncResult(state, player);
//	if (callback != null) callback.Invoke(result);
//	return result;
//
// The callback is invoked BEFORE Begin returns, from XNA's own IL. There is no
// thread, no queue and no work to wait for: the result is already complete when
// the caller receives it.
//
// CNA's header says the same thing about its side -- "the canonical API uses
// XNA's fake-async BeginXxx/EndXxx pair, which CNA completes synchronously" --
// but the claim above rests on the reference, not on CNA. CNA agreeing is a
// convenience, not the evidence.
//
// # What that makes each member
//
//	IsCompleted             always true
//	CompletedSynchronously  always true
//	AsyncWaitHandle         a handle that is already signalled
//	AsyncState              the state object the caller handed to Begin
//
// The three constants are not a simplification of a general case; they are what
// this async result is. A projection that returned false from IsCompleted would
// be describing an implementation the reference does not have.
type AsyncResult struct {
	state any
	// signalled is the projection of System.Threading.WaitHandle. A CLR wait
	// handle is a thing a caller BLOCKS on until it is set; a receive on a
	// closed Go channel is the same thing and returns immediately, which is the
	// only state this handle is ever in.
	signalled chan struct{}
}

// NewCompletedAsyncResult builds the already-complete result a Begin member
// returns.
//
// It is EXPORTED for the reason the ReadOnlyCollection constructors are: the
// XNA types that produce one -- StorageDevice's five Begin members -- live in a
// different Go package and cannot reach an unexported constructor. It is the
// language adapter's constructor, not an XNA identity, and it matches the
// reference's own `private` StorageDeviceAsyncResult..ctor rather than
// inventing a capability: a consumer receives one from Begin and has no reason
// to build one, but the projection of Begin must be able to.
func NewCompletedAsyncResult(state any) *AsyncResult {
	return newAsyncResult(state)
}

// newAsyncResult builds the already-complete result Begin returns.
func newAsyncResult(state any) *AsyncResult {
	signalled := make(chan struct{})
	close(signalled)
	return &AsyncResult{state: state, signalled: signalled}
}

// AsyncCallback is System.AsyncCallback, the delegate every Begin member takes
// and invokes -- from managed code, before it returns.
//
// It is a Go func for the reason EventHandler[T] is: a CLR delegate is a
// callable, and a projection that degraded it to `any` would erase the argument
// the callback receives, which for an APM callback is the whole point.
type AsyncCallback func(result *AsyncResult)

// AsyncState is IAsyncResult::get_AsyncState, the object the caller passed to
// Begin. It is `System.Object` and therefore `any`, and it is passed back
// untouched.
func (r *AsyncResult) AsyncState() any {
	if r == nil {
		return nil
	}
	return r.state
}

// AsyncWaitHandle is IAsyncResult::get_AsyncWaitHandle.
//
// System.Threading.WaitHandle projects to a receive-only channel, which is the
// Go type whose ROLE it is -- the settled rule that made a read-only Stream
// position an io.Reader. The channel this returns is always already closed, so
// a receive on it never blocks, because the operation it stands for is always
// already finished.
func (r *AsyncResult) AsyncWaitHandle() <-chan struct{} {
	if r == nil {
		return nil
	}
	return r.signalled
}

// CompletedSynchronously is IAsyncResult::get_CompletedSynchronously. It is
// TRUE for every result this projection produces, because Begin does all of its
// work before returning.
func (r *AsyncResult) CompletedSynchronously() bool { return r != nil }

// IsCompleted is IAsyncResult::get_IsCompleted, true for the same reason.
func (r *AsyncResult) IsCompleted() bool { return r != nil }
