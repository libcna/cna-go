package storage

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// StorageDevice is Microsoft.Xna.Framework.Storage.StorageDevice:
//
//	.class public auto ansi sealed StorageDevice extends System.Object
//
// A place a game may save to, obtained through a selector and then opened into
// one or more named containers.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Storage.dll   798f678e9ae3d9af...
//
// # XNA's storage APM is FAKE async, measured from XNA
//
// BeginShowSelector's whole body after its two guards is
//
//	var result = new StorageDeviceAsyncResult(state, player);
//	if (callback != null) callback.Invoke(result);
//	return result;
//
// The callback fires BEFORE Begin returns, from managed code, and the result is
// already complete when the caller gets it. There is no thread and nothing to
// wait for. CNA's header says the same of its side -- "XNA's fake-async
// BeginXxx/EndXxx pair, which CNA completes synchronously" -- but the claim
// rests on the reference.
//
// That is why the projection hands the native call to BEGIN rather than to End:
// Begin is where the reference does its work too.
type StorageDevice struct {
	handle uint64
	// player is the `playerIndex` field, carried so BeginOpenContainer can pass
	// it on. 0xff is the reference's sentinel for "no player was named".
	player int32
}

// storageNoPlayer is the reference's `ldc.i4 0xff` sentinel: a selector that
// named no player stores 255 rather than a valid PlayerIndex, and the range
// check skips it explicitly.
const storageNoPlayer int32 = 0xff

// The sentinel errors these types answer, unexported because the XNA contract
// declares no error type here.
var (
	// errStorageArgumentNull projects System.ArgumentNullException.
	errStorageArgumentNull = errors.New("storage argument is nil")
	// errStorageArgumentOutOfRange projects System.ArgumentOutOfRangeException.
	errStorageArgumentOutOfRange = errors.New("storage argument is out of range")
	// errStorageArgument projects System.ArgumentException.
	errStorageArgument = errors.New("storage argument is invalid")
	// errStorageInvalidOperation projects System.InvalidOperationException.
	errStorageInvalidOperation = errors.New("storage operation is invalid")
	// errStorageDisposed projects System.ObjectDisposedException.
	errStorageDisposed = errors.New("the storage container has been disposed")
	// errStorageNoRuntime is the Go-only refusal a member answers when no
	// runtime is loaded. The reference reaches the filesystem directly.
	errStorageNoRuntime = errors.New("this member needs a loaded runtime")
)

// The FrameworkResources messages this family throws, verified byte for byte.
const (
	// cannotEndTwice is what a second End on one Begin raises.
	cannotEndTwice = "An \"End\" function can only be called once for each call to \"Begin.\""
	// invalidStoragePath is what a path that escapes the container's root
	// raises. It is the reference's OWN containment guard.
	invalidStoragePath = "The specified storage path is invalid."
)

func storageArgumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errStorageArgumentNull, parameter)
}

func storageOutOfRangeError(parameter string) error {
	return fmt.Errorf("%w: %s", errStorageArgumentOutOfRange, parameter)
}

// StorageDeviceBeginShowSelectorByAsyncCallbackAndObject is
// StorageDevice::BeginShowSelector(AsyncCallback, Object): no player, no space
// requirement.
func StorageDeviceBeginShowSelectorByAsyncCallbackAndObject(
	callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	return storageBeginShowSelector(0, false, 0, 0, false, callback, state)
}

// StorageDeviceBeginShowSelectorByPlayerIndexAndAsyncCallbackAndObject is
// StorageDevice::BeginShowSelector(PlayerIndex, AsyncCallback, Object), which
// the reference implements as the four-argument form with `ldc.i4.0` for the
// size and `ldc.i4.1` for the directory count.
func StorageDeviceBeginShowSelectorByPlayerIndexAndAsyncCallbackAndObject(
	player framework.PlayerIndex, callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	return storageBeginShowSelector(int32(player), true, 0, 1, false, callback, state)
}

// StorageDeviceBeginShowSelectorByInt32AndInt32AndAsyncCallbackAndObject is
// StorageDevice::BeginShowSelector(Int32, Int32, AsyncCallback, Object).
func StorageDeviceBeginShowSelectorByInt32AndInt32AndAsyncCallbackAndObject(
	sizeInBytes, directoryCount int32, callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	return storageBeginShowSelector(storageNoPlayer, false, sizeInBytes, directoryCount, true, callback, state)
}

// StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject
// is the four-argument overload the other three forward to, and the only one
// that validates.
//
// Measured guards, in the reference's order:
//
//	if (player != 0xff && (player < 0 || player > 3))
//	    throw new ArgumentOutOfRangeException("player");
//	if (sizeInBytes < 0)
//	    throw new ArgumentOutOfRangeException("sizeInBytes");
//
// `directoryCount` is NOT validated. A negative one is accepted, which is
// measured rather than assumed: only `sizeInBytes` has a `bge.s` guard.
func StorageDeviceBeginShowSelectorByPlayerIndexAndInt32AndInt32AndAsyncCallbackAndObject(
	player framework.PlayerIndex, sizeInBytes, directoryCount int32,
	callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	return storageBeginShowSelector(int32(player), true, sizeInBytes, directoryCount, true, callback, state)
}

// storageBeginShowSelector is the shared body. It performs the native selection
// EAGERLY, because that is where the reference does its work too: Begin builds
// the completed result and invokes the callback before returning.
func storageBeginShowSelector(
	player int32, hasPlayer bool, sizeInBytes, directoryCount int32, hasSpace bool,
	callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	if player != storageNoPlayer && (player < 0 || player > 3) {
		return nil, storageOutOfRangeError("player")
	}
	if hasSpace && sizeInBytes < 0 {
		return nil, storageOutOfRangeError("sizeInBytes")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	handle, err := runtime.StorageShowSelector(uint32(player), hasPlayer, sizeInBytes, directoryCount, hasSpace)
	if err != nil {
		return nil, err
	}
	result := framework.NewCompletedAsyncResult(state)
	storeSelectorResult(result, &StorageDevice{handle: handle, player: selectorPlayer(player, hasPlayer)})
	// The callback is invoked before Begin returns, exactly as the reference
	// invokes AsyncCallback::Invoke before its own `ret`.
	if callback != nil {
		callback(result)
	}
	return result, nil
}

// selectorPlayer is the value the reference stores in playerIndex: the named
// player, or its 0xff sentinel when none was named.
func selectorPlayer(player int32, hasPlayer bool) int32 {
	if hasPlayer {
		return player
	}
	return storageNoPlayer
}

// StorageDeviceEndShowSelector is StorageDevice::EndShowSelector(IAsyncResult),
// measured in full:
//
//	var typed = result as StorageDeviceAsyncResult;
//	if (typed == null) throw new ArgumentNullException("result");
//	if (typed.endHasBeenCalled)
//	    throw new InvalidOperationException(FrameworkResources.CannotEndTwice);
//	typed.endHasBeenCalled = true;
//	...
//
// Two measured quirks. A result of the WRONG TYPE raises
// ArgumentNullException rather than ArgumentException -- the `isinst` and the
// null test collapse into one branch. And the second End on one Begin is
// refused, which is what the endHasBeenCalled latch exists for.
func StorageDeviceEndShowSelector(result *framework.AsyncResult) (*StorageDevice, error) {
	if result == nil {
		return nil, storageArgumentNullError("result")
	}
	device, ok := takeSelectorResult(result)
	if !ok {
		// Either the result came from a different Begin, or End has already
		// been called on this one.
		return nil, fmt.Errorf("%w: %s", errStorageInvalidOperation, cannotEndTwice)
	}
	return device, nil
}

// deviceChanged is the process-wide StorageDevice::DeviceChanged event. It is
// STATIC in the reference -- the event says a device appeared or vanished, not
// that this one did -- so the subscription lives beside the type rather than on
// an instance.
var deviceChanged framework.EventSource[*framework.EventArgs]

// StorageDeviceAddDeviceChangedHandler subscribes to
// StorageDevice::DeviceChanged.
func StorageDeviceAddDeviceChangedHandler(
	handler framework.EventHandler[*framework.EventArgs],
) (framework.EventSubscription, error) {
	return deviceChanged.Add(handler)
}

// StorageDeviceRemoveDeviceChangedHandler unsubscribes from it.
func StorageDeviceRemoveDeviceChangedHandler(subscription framework.EventSubscription) error {
	return deviceChanged.Remove(subscription)
}

// FreeSpace is StorageDevice::get_FreeSpace.
func (d *StorageDevice) FreeSpace() (int64, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok || d == nil {
		return 0, errStorageNoRuntime
	}
	return runtime.StorageDeviceFreeSpace(d.handle)
}

// TotalSpace is StorageDevice::get_TotalSpace.
func (d *StorageDevice) TotalSpace() (int64, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok || d == nil {
		return 0, errStorageNoRuntime
	}
	return runtime.StorageDeviceTotalSpace(d.handle)
}

// IsConnected is StorageDevice::get_IsConnected.
func (d *StorageDevice) IsConnected() (bool, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok || d == nil {
		return false, errStorageNoRuntime
	}
	return runtime.StorageDeviceIsConnected(d.handle)
}

// DeleteContainer is StorageDevice::DeleteContainer(String).
func (d *StorageDevice) DeleteContainer(titleName string) error {
	if d == nil {
		return errStorageNoRuntime
	}
	if titleName == "" {
		return storageArgumentNullError("titleName")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return runtime.StorageDeviceDeleteContainer(d.handle, titleName)
}

// BeginOpenContainer is StorageDevice::BeginOpenContainer(String,
// AsyncCallback, Object), whose body is the same fake-async shape:
//
//	var result = new StorageContainerOpenAsyncResult(this, playerIndex,
//	                                                 displayName, state);
//	if (callback != null) callback.Invoke(result);
//	return result;
//
// It validates NOTHING -- not even a null displayName. The name is carried to
// End and the failure, if any, happens there.
func (d *StorageDevice) BeginOpenContainer(
	displayName string, callback framework.AsyncCallback, state any,
) (*framework.AsyncResult, error) {
	if d == nil {
		return nil, errStorageNoRuntime
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	handle, err := runtime.StorageContainerOpen(d.handle, displayName)
	if err != nil {
		return nil, err
	}
	result := framework.NewCompletedAsyncResult(state)
	storeContainerResult(result, &StorageContainer{handle: handle, device: d})
	if callback != nil {
		callback(result)
	}
	return result, nil
}

// EndOpenContainer is StorageDevice::EndOpenContainer(IAsyncResult), which
// carries the same two refusals EndShowSelector does.
func (d *StorageDevice) EndOpenContainer(result *framework.AsyncResult) (*StorageContainer, error) {
	if result == nil {
		return nil, storageArgumentNullError("result")
	}
	container, ok := takeContainerResult(result)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errStorageInvalidOperation, cannotEndTwice)
	}
	return container, nil
}
