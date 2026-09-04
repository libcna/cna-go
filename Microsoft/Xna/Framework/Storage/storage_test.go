package storage

import (
	"errors"
	"io"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// Every test here runs WITHOUT a loaded runtime, which is deliberate rather
// than a limitation.
//
// The managed half of this family -- the argument guards, the End-twice latch,
// the fake-async completion, the disposal latch -- is the half a projection can
// get wrong, and none of it needs a filesystem. The half that does need one is
// exercised by the stress slice, which sets its own application name first so
// CNA builds a root under the project rather than under the user's documents.
//
// A test that touched real storage from here would be a test that wrote to
// whatever directory the host happened to give it.

// TestBeginShowSelectorGuardsAreMeasured walks the two the reference has, and
// the one it does NOT have.
func TestBeginShowSelectorGuardsAreMeasured(t *testing.T) {
	// player is checked against 0..3, with 0xff exempted.
	for _, player := range []framework.PlayerIndex{-1, 4, 100} {
		_, err := StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
			player, 0, 1, nil, nil)
		if !errors.Is(err, errStorageArgumentOutOfRange) {
			t.Fatalf("player %d was accepted: %v", player, err)
		}
	}
	// A negative size is refused.
	_, err := StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndexOne, -1, 1, nil, nil)
	if !errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatalf("a negative sizeInBytes was accepted: %v", err)
	}
	// When BOTH would trip, the reference checks the PLAYER first -- its guard
	// comes before the size guard in the IL -- so the refusal must name player.
	_, err = StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndex(9), -1, 1, nil, nil)
	if !errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatalf("both-bad = %v, want an out-of-range refusal", err)
	}
	if !contains(err.Error(), "player") {
		t.Fatalf("both-bad named %q; the player guard runs first", err.Error())
	}
	if contains(err.Error(), "sizeInBytes") {
		t.Fatalf("both-bad named sizeInBytes: %q", err.Error())
	}

	// A negative DIRECTORY COUNT is NOT refused. Only sizeInBytes carries a
	// `bge.s` guard in the reference, so this must reach the runtime refusal
	// rather than an argument one.
	_, err = StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndexOne, 0, -5, nil, nil)
	if errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatal("a negative directoryCount was refused; the reference validates only sizeInBytes")
	}
	if !errors.Is(err, errStorageNoRuntime) {
		t.Fatalf("expected the no-runtime refusal past the guards, got %v", err)
	}
}

// TestShowSelectorPlayerSentinelIsExempt pins the `ldc.i4 0xff` branch: the
// no-player sentinel skips the range check that would otherwise reject 255.
func TestShowSelectorPlayerSentinelIsExempt(t *testing.T) {
	// The overload that names NO player must not trip the range guard.
	_, err := StorageDeviceBeginShowSelectorByAsyncCallbackAndObject(nil, nil)
	if errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatal("the no-player overload tripped the player range guard")
	}
	if !errors.Is(err, errStorageNoRuntime) {
		t.Fatalf("expected the no-runtime refusal, got %v", err)
	}
	// And 255 passed explicitly is the same sentinel. The literal is written
	// out rather than taken from storageNoPlayer: a test that reads the same
	// constant the code does would follow it to a wrong value.
	_, err = StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndex(255), 0, 1, nil, nil)
	if errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatal("the 0xff sentinel was rejected by the range guard")
	}
	if storageNoPlayer != 255 {
		t.Fatalf("the sentinel is %d, want 255: the reference emits `ldc.i4 0xff`", storageNoPlayer)
	}
	// The range guard's own boundary: 3 is the last valid player and 4 is not.
	if _, err := StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndex(3), 0, 1, nil, nil); errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatal("player 3 was rejected; the reference accepts 0..3")
	}
	if _, err := StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
		framework.PlayerIndex(4), 0, 1, nil, nil); !errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatal("player 4 was accepted; the reference's range is 0..3")
	}
}

// TestEndOpenContainerRefusesNilAndSecondCall is EndShowSelector's counterpart:
// the container pairing carries the same endHasBeenCalled latch.
func TestEndOpenContainerRefusesNilAndSecondCall(t *testing.T) {
	device := &StorageDevice{handle: 4, player: storageNoPlayer}
	if _, err := device.EndOpenContainer(nil); !errors.Is(err, errStorageArgumentNull) {
		t.Fatalf("EndOpenContainer(nil) = %v, want the argument-null refusal", err)
	}
	stray := framework.NewCompletedAsyncResult(nil)
	if _, err := device.EndOpenContainer(stray); err == nil {
		t.Fatal("EndOpenContainer accepted a result no Begin produced")
	}
	result := framework.NewCompletedAsyncResult(nil)
	container := &StorageContainer{handle: 5, device: device}
	storeContainerResult(result, container)
	got, err := device.EndOpenContainer(result)
	if err != nil || got != container {
		t.Fatalf("first EndOpenContainer = %v, %v", got, err)
	}
	if _, err := device.EndOpenContainer(result); !errors.Is(err, errStorageInvalidOperation) {
		t.Fatalf("second EndOpenContainer = %v, want the invalid-operation refusal", err)
	}
}

// TestEndShowSelectorRefusesNilAndSecondCall pins the two failures the
// reference's End has.
func TestEndShowSelectorRefusesNilAndSecondCall(t *testing.T) {
	if _, err := StorageDeviceEndShowSelector(nil); !errors.Is(err, errStorageArgumentNull) {
		t.Fatalf("End(nil) = %v, want the argument-null refusal", err)
	}
	// A result that never came from a Begin is the reference's `isinst` failure.
	stray := framework.NewCompletedAsyncResult(nil)
	if _, err := StorageDeviceEndShowSelector(stray); err == nil {
		t.Fatal("End accepted a result no Begin produced")
	}

	// A paired result works once and is then refused, which is what
	// endHasBeenCalled is.
	result := framework.NewCompletedAsyncResult("state")
	device := &StorageDevice{handle: 7, player: storageNoPlayer}
	storeSelectorResult(result, device)
	got, err := StorageDeviceEndShowSelector(result)
	if err != nil || got != device {
		t.Fatalf("first End = %v, %v", got, err)
	}
	_, err = StorageDeviceEndShowSelector(result)
	if !errors.Is(err, errStorageInvalidOperation) {
		t.Fatalf("second End = %v, want the invalid-operation refusal", err)
	}
	if err == nil || !contains(err.Error(), cannotEndTwice) {
		t.Fatalf("the refusal carried %v, want the CannotEndTwice message", err)
	}
}

// TestAsyncResultIsAlreadyComplete pins the fake-async shape: the result a
// Begin hands back reports completion, and its wait handle does not block.
func TestAsyncResultIsAlreadyComplete(t *testing.T) {
	result := framework.NewCompletedAsyncResult("carried")
	if !result.IsCompleted() {
		t.Fatal("the result reported incomplete; XNA's Begin completes before it returns")
	}
	if !result.CompletedSynchronously() {
		t.Fatal("the result reported asynchronous completion")
	}
	if result.AsyncState() != "carried" {
		t.Fatalf("AsyncState = %v, want the state Begin was given", result.AsyncState())
	}
	// The wait handle must already be signalled. A receive that blocked here
	// would hang the test rather than fail it, so the select proves it cannot.
	select {
	case <-result.AsyncWaitHandle():
	default:
		t.Fatal("the wait handle was not signalled; the operation is already finished")
	}
}

// TestContainerGuardsCheckDisposalFirst pins ValidateArguments' ORDER: the
// disposal check runs before the path check, so a disposed container refuses an
// empty path with the disposal error rather than the argument one.
func TestContainerGuardsCheckDisposalFirst(t *testing.T) {
	container := &StorageContainer{handle: 1}
	// Live container, empty path -> the ARGUMENT refusal, and it is
	// ArgumentNull rather than ArgumentException.
	if err := container.CreateDirectory(""); !errors.Is(err, errStorageArgumentNull) {
		t.Fatalf("empty directory = %v, want the argument-null refusal", err)
	}
	if err := container.DeleteFile(""); !errors.Is(err, errStorageArgumentNull) {
		t.Fatalf("empty file = %v, want the argument-null refusal", err)
	}
	// Disposed container, empty path -> the DISPOSAL refusal, because
	// VerifyNotDisposed runs first.
	container.disposed = true
	if err := container.CreateDirectory(""); !errors.Is(err, errStorageDisposed) {
		t.Fatalf("disposed container = %v, want the disposal refusal: VerifyNotDisposed runs first", err)
	}
	if _, err := container.FileExists("save.dat"); !errors.Is(err, errStorageDisposed) {
		t.Fatalf("disposed FileExists = %v, want the disposal refusal", err)
	}
}

// TestGetNamesAcceptsAnEmptyPatternButPathsDoNot pins the asymmetry: the two
// enumerating members call VerifyNotDisposed DIRECTLY rather than through
// ValidateArguments, so an empty pattern means "every name" while an empty path
// is a refusal.
func TestGetNamesAcceptsAnEmptyPatternButPathsDoNot(t *testing.T) {
	container := &StorageContainer{handle: 1}
	if _, err := container.GetFileNamesByString(""); errors.Is(err, errStorageArgumentNull) {
		t.Fatal("an empty search pattern was refused; only PATHS go through ValidateArguments")
	}
	if _, err := container.GetDirectoryNamesByString(""); errors.Is(err, errStorageArgumentNull) {
		t.Fatal("an empty directory pattern was refused")
	}
	// But an empty PATH still is.
	if err := container.CreateDirectory(""); !errors.Is(err, errStorageArgumentNull) {
		t.Fatal("an empty path was accepted")
	}
	// The enumerating members skip ValidateArguments but NOT VerifyNotDisposed:
	// the reference calls it directly, which is the whole reason they are
	// written separately.
	container.disposed = true
	if _, err := container.GetFileNamesByString(""); !errors.Is(err, errStorageDisposed) {
		t.Fatalf("GetFileNames on a disposed container = %v, want the disposal refusal", err)
	}
	if _, err := container.GetDirectoryNamesByNone(); !errors.Is(err, errStorageDisposed) {
		t.Fatalf("GetDirectoryNames on a disposed container = %v, want the disposal refusal", err)
	}
}

// TestContainerDisposalLatch pins that Dispose raises Disposing once, that a
// second Dispose is a no-op, and that Finalize does NOT raise.
func TestContainerDisposalLatch(t *testing.T) {
	container := &StorageContainer{handle: 1}
	raised := 0
	if _, err := container.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		raised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposingHandler: %v", err)
	}
	if disposed, err := container.IsDisposed(); err == nil && disposed {
		t.Fatal("a fresh container reported disposed")
	}
	if err := container.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if raised != 1 {
		t.Fatalf("Disposing raised %d times, want 1", raised)
	}
	disposed, err := container.IsDisposed()
	if err != nil || !disposed {
		t.Fatalf("IsDisposed after Dispose = %v, %v", disposed, err)
	}
	// A second Dispose raises nothing and fails nothing.
	if err := container.Dispose(); err != nil {
		t.Fatalf("second Dispose: %v", err)
	}
	if raised != 1 {
		t.Fatalf("Disposing raised %d times after a second Dispose, want 1", raised)
	}

	// Finalize on a FRESH container releases without raising, because the
	// reference's finalizer runs Dispose(false) and skips the managed half.
	other := &StorageContainer{handle: 2}
	otherRaised := 0
	if _, err := other.AddDisposingHandler(func(sender any, args *framework.EventArgs) error {
		otherRaised++
		return nil
	}); err != nil {
		t.Fatalf("AddDisposingHandler: %v", err)
	}
	if err := other.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if otherRaised != 0 {
		t.Fatal("Finalize raised Disposing; the reference's Dispose(false) skips the managed half")
	}
	if disposed, _ := other.IsDisposed(); !disposed {
		t.Fatal("Finalize did not mark the container disposed")
	}
}

// TestStorageDeviceIsFallibleAfterDisposal pins that get_StorageDevice carries
// the disposal refusal -- it LOOKS like a field read and is not, because the
// reference reads it after VerifyNotDisposed.
func TestStorageDeviceIsFallibleAfterDisposal(t *testing.T) {
	device := &StorageDevice{handle: 3, player: 0}
	container := &StorageContainer{handle: 1, device: device}
	got, err := container.StorageDevice()
	if err != nil || got != device {
		t.Fatalf("StorageDevice = %v, %v", got, err)
	}
	container.disposed = true
	if _, err := container.StorageDevice(); !errors.Is(err, errStorageDisposed) {
		t.Fatalf("StorageDevice on a disposed container = %v, want the disposal refusal", err)
	}
	// IsDisposed is the ONE property that still answers, because asking a
	// disposed object whether it is disposed has to work.
	disposed, err := container.IsDisposed()
	if err != nil || !disposed {
		t.Fatalf("IsDisposed = %v, %v: it must answer after disposal", disposed, err)
	}
}

// TestStorageStreamRefusesAfterClose pins the stream's own latch and its
// io.EOF translation contract.
func TestStorageStreamRefusesAfterClose(t *testing.T) {
	stream := newStorageStream(9)
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// A second Close is a no-op, which is what CNA documents and what
	// io.Closer callers rely on.
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := stream.Read(make([]byte, 4)); !errors.Is(err, errStorageStreamClosed) {
		t.Fatalf("Read after Close = %v", err)
	}
	if _, err := stream.Write([]byte{1}); !errors.Is(err, errStorageStreamClosed) {
		t.Fatalf("Write after Close = %v", err)
	}
	if _, err := stream.Seek(0, io.SeekStart); !errors.Is(err, errStorageStreamClosed) {
		t.Fatalf("Seek after Close = %v", err)
	}
	// An out-of-range whence is refused before the handle is touched.
	live := newStorageStream(10)
	if _, err := live.Seek(0, 99); !errors.Is(err, errStorageArgumentOutOfRange) {
		t.Fatalf("Seek with a bad whence = %v", err)
	}
	// A zero-length read and write are no-ops rather than refusals, which is
	// what io.Reader and io.Writer require.
	if n, err := live.Read(nil); n != 0 || err != nil {
		t.Fatalf("empty Read = %d, %v", n, err)
	}
	if n, err := live.Write(nil); n != 0 || err != nil {
		t.Fatalf("empty Write = %d, %v", n, err)
	}
}

// TestStorageStreamSatisfiesReadWriteSeeker is the compile-time claim the
// return positions make.
func TestStorageStreamSatisfiesReadWriteSeeker(t *testing.T) {
	var _ io.ReadWriteSeeker = newStorageStream(1)
	var _ io.Closer = newStorageStream(1)
}

func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
